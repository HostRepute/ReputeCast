package digests

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

var (
	mIssueNotOpen    = msgs.M("issue.notOpen")
	mReleaseNotHeld  = msgs.M("issue.releaseNotHeld")
	mReleaseBlocked  = msgs.M("issue.releaseBlocked")
	mRebuildState    = msgs.M("issue.rebuildState")
	mIssueNoTemplate = msgs.M("issue.noTemplate")
	mIssueNoItems    = msgs.M("issue.noItems")
)

func isUnique(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

func pctOf(n, d int64) *float64 {
	if d <= 0 {
		return nil
	}
	v := math.Round(float64(n)/float64(d)*1000) / 10
	return &v
}

// displayStatus is an issue's status as the screens show it: a prepared
// issue with an open request is waiting for approval.
func displayStatus(status string, approval *DigestIssueApproval) string {
	if status == "prepared" && approval != nil && approval.Status == "pending" {
		return "waiting"
	}
	return status
}

// Issues lists a digest's issues, newest first, each compared with the
// sent issue before it.
func (s *Service) Issues(ctx context.Context, ws *store.Workspace, id uuid.UUID) (DigestIssues, error) {
	l := i18n.LangFrom(ctx)
	d, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestIssues{}, api.MapError(err, "Digest")
	}
	rows, err := s.Q.DigestIssues(ctx, store.DigestIssuesParams{DigestID: id, Lim: 52})
	if err != nil {
		return DigestIssues{}, err
	}
	campaignIDs, issueIDs := []uuid.UUID{}, []uuid.UUID{}
	for _, r := range rows {
		issueIDs = append(issueIDs, r.ID)
		if r.CampaignID != nil {
			campaignIDs = append(campaignIDs, *r.CampaignID)
		}
	}
	results := map[uuid.UUID]store.DigestIssueResultsRow{}
	if res, err := s.Q.DigestIssueResults(ctx, campaignIDs); err == nil {
		for _, r := range res {
			results[r.CampaignID] = r
		}
	} else {
		return DigestIssues{}, err
	}
	approvalsByCampaign := map[uuid.UUID]*DigestIssueApproval{}
	if aps, err := s.Q.DigestIssueApprovals(ctx, campaignIDs); err == nil {
		for _, a := range aps {
			if a.CampaignID == nil {
				continue
			}
			ap := &DigestIssueApproval{ID: a.ID.String(), Status: a.Status, RequestedAt: a.RequestedAt, DecidedAt: a.DecidedAt, Rules: nonNil(a.MatchedRules)}
			if a.DecidedByName != "" {
				ap.DecidedBy = &DigestPerson{Name: a.DecidedByName, Initials: initials(a.DecidedByName), Color: a.DecidedByColor}
			}
			approvalsByCampaign[*a.CampaignID] = ap
		}
	} else {
		return DigestIssues{}, err
	}
	counts := map[uuid.UUID]int{}
	if cs, err := s.Q.DigestIssueItemCounts(ctx, issueIDs); err == nil {
		for _, c := range cs {
			counts[c.IssueID] = int(c.N)
		}
	}
	out := DigestIssues{Items: make([]DigestIssue, 0, len(rows))}
	// The collecting issue counts its new posts as they stand.
	open := s.openPosts(ctx, d.Digest)
	// Who the next issue goes to, counted once at most: counting reads every
	// contact on its lists, and the estimates and the approval check both
	// need it.
	next := sync.OnceValues(func() (*store.DigestCampaignTemplateRow, DigestAudience) {
		tpl, err := s.template(ctx, d.Digest)
		if err != nil || tpl == nil {
			return nil, DigestAudience{Empty: true}
		}
		a, err := s.audienceOf(ctx, ws, tpl.Audience)
		if err != nil {
			return nil, DigestAudience{Empty: true}
		}
		return tpl, a
	})
	for _, r := range rows {
		it := DigestIssue{ID: r.ID.String(), SendAt: r.SendAt, Deadline: r.Deadline, SkipReason: r.SkipReason, SkipNote: skipNoteIn(r.SkipNote, r.SkippedBy, l),
			SkippedBy: r.SkippedByName, SkippedAt: r.SkippedAt, HoldReason: r.HoldReason, Posts: counts[r.ID], FeedTries: feedTriesIn(r.FeedTries, l),
			PreparedAt: r.PreparedAt, SentAt: r.SentAt}
		if r.Number != nil {
			n := int(*r.Number)
			it.Number = &n
		}
		if r.CampaignID != nil {
			it.CampaignID = r.CampaignID.String()
			it.Approval = approvalsByCampaign[*r.CampaignID]
			if res, ok := results[*r.CampaignID]; ok && res.Sent > 0 {
				sent := res.Sent
				it.Recipients = &sent
				it.Delivered = pctOf(res.Delivered, res.Sent)
				it.HumanOpens = pctOf(res.OpenedHuman, res.Delivered)
				it.Clicks = pctOf(res.Clicked, res.Delivered)
			}
		}
		it.Status = displayStatus(r.Status, it.Approval)
		if r.Status == "collecting" {
			it.Posts = open
		}
		if it.Recipients == nil && (r.Status == "collecting" || r.Status == "preparing" || r.Status == "prepared" || r.Status == "held" || r.Status == "draft") {
			if r.CampaignAudience != nil && *r.CampaignAudience > 0 {
				n := *r.CampaignAudience
				it.Recipients = &n
			} else if _, a := next(); !a.Empty && a.Final > 0 {
				est := a.Final
				it.Recipients = &est
			}
			it.Estimated = it.Recipients != nil
		}
		out.Items = append(out.Items, it)
	}
	// Compare each sent issue with the sent one before it.
	var prev *DigestIssue
	for i := len(out.Items) - 1; i >= 0; i-- {
		it := &out.Items[i]
		if it.Status != "sent" && it.Status != "sending" {
			continue
		}
		if prev == nil {
			it.FirstIssue = true
		} else {
			pd := it.Posts - prev.Posts
			it.PostsDelta = &pd
			if it.Recipients != nil && prev.Recipients != nil {
				rd := *it.Recipients - *prev.Recipients
				it.RecipientsDelta = &rd
			}
			it.HumanOpensDelta = delta(it.HumanOpens, prev.HumanOpens)
			it.ClicksDelta = delta(it.Clicks, prev.Clicks)
		}
		prev = it
	}
	needs := sync.OnceValue(func() bool {
		tpl, a := next()
		return tpl != nil && s.needsApproval(ctx, ws, d.Digest, tpl, a)
	})
	for i := range out.Items {
		switch out.Items[i].Status {
		case "collecting", "preparing":
			out.Items[i].NeedsApproval = needs()
		}
	}
	return out, nil
}

func delta(a, b *float64) *float64 {
	if a == nil || b == nil {
		return nil
	}
	v := math.Round((*a-*b)*10) / 10
	return &v
}

func feedTries(raw json.RawMessage) []DigestFeedTry {
	out := []DigestFeedTry{}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []DigestFeedTry{}
	}
	return out
}

// storedFeedTry is a failed read at preparation as digest_issues.feed_tries
// keeps it: its English, and the same words as a message for a reader's
// language (tries from before have none).
type storedFeedTry struct {
	DigestFeedTry
	Text *i18n.Text `json:"text,omitempty"`
}

func storedFeedTries(raw json.RawMessage) []storedFeedTry {
	out := []storedFeedTry{}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []storedFeedTry{}
	}
	return out
}

// feedTriesIn is an issue's failed reads for a reader.
func feedTriesIn(raw json.RawMessage, l i18n.Lang) []DigestFeedTry {
	tries := storedFeedTries(raw)
	out := make([]DigestFeedTry, 0, len(tries))
	for _, t := range tries {
		try := t.DigestFeedTry
		try.Error = textIn(t.Text, l, try.Error)
		out = append(out, try)
	}
	return out
}

// skipNoteIn is why an issue was skipped, for a reader: what a person
// wrote stays as written; what a digest wrote when its owner lost a
// permission is written in the reader's language, as what the owner can no
// longer do, never the permission key (review X-1).
func skipNoteIn(note string, by *uuid.UUID, l i18n.Lang) string {
	if perm, ok := strings.CutPrefix(note, ownerLostPrefix); ok && by == nil {
		lost := ""
		switch authz.Permission(perm) {
		case authz.CampaignsSend:
			lost = "send"
		case authz.CampaignsEdit:
			lost = "edit"
		}
		return mIssueOwnerLost.Text(l, i18n.Args{"lost": lost})
	}
	return note
}

// ownerLostPrefix starts the note of an issue skipped because its owner
// lost a permission: "the owner lost campaigns:send".
const ownerLostPrefix = "the owner lost "

// openPosts counts the new posts of the next issue (as many as fit).
func (s *Service) openPosts(ctx context.Context, d store.Digest) int {
	all, err := s.items(ctx, d, 400, false)
	if err != nil {
		return 0
	}
	b := DefaultBlock()
	if tpl, err := s.template(ctx, d); err == nil && tpl != nil {
		b, _ = BlockOf(tpl.Mjml)
	}
	n := len(newItems(all))
	return min(n, b.Max)
}

// issueFor loads an issue of a digest of the workspace.
func (s *Service) issueFor(ctx context.Context, ws *store.Workspace, digestID, issueID uuid.UUID) (store.Digest, store.DigestIssue, error) {
	r, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: digestID})
	if err != nil {
		return store.Digest{}, store.DigestIssue{}, api.MapError(err, "Digest")
	}
	issue, err := s.Q.DigestIssueGet(ctx, store.DigestIssueGetParams{ID: issueID, DigestID: digestID})
	if err != nil {
		return r.Digest, issue, api.MapError(err, "Issue")
	}
	return r.Digest, issue, nil
}

// DigestSkipInput skips an issue, with an optional note.
type DigestSkipInput struct {
	Note string `json:"note,omitempty" maxLength:"500"`
}

// Skip skips an issue by hand: nothing goes out, its posts stay new for the
// next issue, it uses no number.
func (s *Service) Skip(ctx context.Context, ws *store.Workspace, p *authz.Principal, digestID, issueID uuid.UUID, in DigestSkipInput) (DigestIssues, error) {
	l := i18n.LangFrom(ctx)
	d, issue, err := s.issueFor(ctx, ws, digestID, issueID)
	if err != nil {
		return DigestIssues{}, err
	}
	switch issue.Status {
	case "collecting", "preparing", "prepared", "held", "draft":
	default:
		return DigestIssues{}, api.Conflict(mIssueNotOpen.Text(l, nil))
	}
	if err := s.skip(ctx, ws, d, issue, "person", strings.TrimSpace(in.Note), actorID(p)); err != nil {
		return DigestIssues{}, err
	}
	if err := s.ensureCollecting(ctx, ws, digestID); err != nil {
		return DigestIssues{}, err
	}
	return s.Issues(ctx, ws, digestID)
}

// skip settles an issue as skipped (or not sent, in draft mode): its
// campaign leaves the campaigns list, its approval request goes.
func (s *Service) skip(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, reason, note string, by *uuid.UUID) error {
	now := s.now()
	status := "skipped"
	if reason == "not_sent" {
		status, reason = "not_sent", ""
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		locked, err := q.DigestIssueLock(ctx, issue.ID)
		if err != nil {
			return err
		}
		switch locked.Status {
		case "collecting", "preparing", "prepared", "held", "draft":
		default:
			return nil // settled meanwhile
		}
		if locked.CampaignID != nil {
			if err := q.DigestCampaignCancelApproval(ctx, store.DigestCampaignCancelApprovalParams{CampaignID: locked.CampaignID, Now: now,
				Reason: firstNonEmpty(map[string]string{"not_approved": "not approved in time", "person": "issue skipped"}[reason], "issue skipped")}); err != nil {
				return err
			}
			if err := q.DigestResolveHolds(ctx, store.DigestResolveHoldsParams{CampaignID: *locked.CampaignID, Now: now}); err != nil {
				return err
			}
			if err := q.DigestCampaignRemove(ctx, store.DigestCampaignRemoveParams{ID: *locked.CampaignID, Now: now}); err != nil {
				return err
			}
		}
		if err := q.DigestIssueSkip(ctx, store.DigestIssueSkipParams{ID: locked.ID, Status: status, Now: now, SkipReason: reason, SkipNote: clip(note, 500), SkippedBy: by}); err != nil {
			return err
		}
		var actor *audit.Actor
		if by == nil {
			actor = s.systemActor(ctx, d)
		}
		action := "digest.issue_skipped"
		if status == "not_sent" {
			action = "digest.issue_not_sent"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "digest", TargetID: d.ID.String(), TargetName: d.Name,
			Actor: actor, Reason: note, Before: map[string]any{"status": locked.Status}, After: map[string]any{"status": status, "reason": reason},
			Metadata: map[string]any{"issue_id": locked.ID.String(), "send_at": locked.SendAt, "summary": skipSummary(reason, note, locked)}})
	})
}

func skipSummary(reason, note string, issue store.DigestIssue) string {
	when := ""
	if issue.SendAt != nil {
		when = issue.SendAt.UTC().Format("Mon 2 Jan 15:04 UTC") + " · "
	}
	switch reason {
	case "no_items":
		return when + "no new posts"
	case "feed":
		return when + "the feed couldn't be read"
	case "not_approved":
		return when + "not approved in time"
	case "held":
		return when + "held by Send Readiness past the late limit"
	case "person":
		if note != "" {
			return when + "skipped: “" + note + "”"
		}
		return when + "skipped by a person"
	}
	return when + "not sent within the late limit"
}

// systemActor is the actor of what a digest does on its own: "System ·
// Engineering blog roundup, turned on by Sofia Rossi".
func (s *Service) systemActor(ctx context.Context, d store.Digest) *audit.Actor {
	name := "System · " + d.Name
	if d.OwnerID != nil {
		if u, err := s.Q.GetUser(ctx, *d.OwnerID); err == nil {
			name += ", turned on by " + u.Name
		}
	}
	return &audit.Actor{Type: authz.ActorSystem, Name: name}
}

// IssueActor is the audit actor of an issue's automatic send (the sending
// engine asks): the digest on its owner's authority. nil for campaigns
// that aren't issues.
func (s *Service) IssueActor(ctx context.Context, campaignID uuid.UUID) *audit.Actor {
	issue, err := s.Q.DigestIssueByCampaign(ctx, &campaignID)
	if err != nil || issue.Mode != "auto" {
		return nil
	}
	d, err := s.Q.DigestByID(ctx, issue.DigestID)
	if err != nil {
		return nil
	}
	return s.systemActor(ctx, d)
}

// Release puts a held issue back on its way once the block is fixed: it
// waits for its approval, if the rules ask for one, and goes out at its
// send time (now, if that passed). Releasing needs campaigns:send.
func (s *Service) Release(ctx context.Context, ws *store.Workspace, p *authz.Principal, digestID, issueID uuid.UUID) (DigestIssues, error) {
	l := i18n.LangFrom(ctx)
	d, issue, err := s.issueFor(ctx, ws, digestID, issueID)
	if err != nil {
		return DigestIssues{}, err
	}
	if issue.Status != "held" || issue.CampaignID == nil {
		return DigestIssues{}, api.Conflict(mReleaseNotHeld.Text(l, nil))
	}
	ch := s.checks()
	if ch == nil {
		return DigestIssues{}, api.Err(http.StatusServiceUnavailable, "readiness_unavailable", mReadinessUnavailable.Text(l, nil))
	}
	run, err := ch.Run(ctx, ws, *issue.CampaignID, readiness.RunOptions{Trigger: readiness.TriggerManual, By: &p.UserID})
	if errors.Is(err, readiness.ErrRunning) {
		return DigestIssues{}, api.Err(http.StatusConflict, "readiness_running", mReadinessRunning.Text(l, nil))
	}
	if err != nil {
		return DigestIssues{}, err
	}
	if run.Verdict == readiness.VerdictBlocked {
		return DigestIssues{}, api.Err(http.StatusConflict, "readiness_blocked", mReleaseBlocked.Text(l, i18n.Args{"count": run.Counts.Blocks}))
	}
	// A request the approvers sent back is asked for again, in the owner's name.
	if ap := s.approvals(); ap != nil {
		if _, err := s.Q.DigestOpenApproval(ctx, issue.CampaignID); errors.Is(err, pgx.ErrNoRows) {
			owner := d.OwnerID
			if owner == nil {
				owner = &p.UserID
			}
			number, _ := s.Q.DigestNextNumber(ctx, d.ID)
			if _, err := ap.RequestForIssue(ctx, ws, *owner, *issue.CampaignID, issueRequest(ctx, s, d, int(number), false)); err != nil && !errors.Is(err, approvals.ErrNoApprovers) {
				s.log().WarnContext(ctx, "ask approval of a released issue", "digest", d.ID, "err", err)
			}
		} else if err != nil {
			return DigestIssues{}, err
		}
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.DigestIssueRelease(ctx, store.DigestIssueReleaseParams{ID: issue.ID, Now: now}); err != nil {
			return err
		}
		at := now.Add(time.Minute)
		if issue.SendAt != nil && issue.SendAt.After(at) {
			at = *issue.SendAt
		}
		sched, _ := json.Marshal(map[string]any{"mode": "at", "at": at.UTC().Format(time.RFC3339), "timezone": ParseSchedule(d.Schedule, ws.Timezone).Timezone})
		if err := q.DigestCampaignSchedule(ctx, store.DigestCampaignScheduleParams{ID: *issue.CampaignID, At: at, Schedule: sched, Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "digest.issue_released", TargetType: "digest", TargetID: d.ID.String(), TargetName: d.Name,
			Before: map[string]any{"status": "held", "hold": issue.HoldReason}, After: map[string]any{"status": "prepared", "send_at": at},
			Metadata: map[string]any{"issue_id": issue.ID.String(), "campaign_id": issue.CampaignID.String(), "run_id": run.ID, "permission": "campaigns:send"}})
	})
	if err != nil {
		return DigestIssues{}, err
	}
	return s.Issues(ctx, ws, digestID)
}

// Rebuild builds a prepared issue again from the current template and
// checks it again. Its approval request is checked against the change.
func (s *Service) Rebuild(ctx context.Context, ws *store.Workspace, p *authz.Principal, digestID, issueID uuid.UUID) (DigestIssues, error) {
	l := i18n.LangFrom(ctx)
	d, issue, err := s.issueFor(ctx, ws, digestID, issueID)
	if err != nil {
		return DigestIssues{}, err
	}
	switch issue.Status {
	case "prepared", "held", "draft":
	default:
		return DigestIssues{}, api.Conflict(mRebuildState.Text(l, nil))
	}
	if issue.CampaignID == nil {
		return DigestIssues{}, api.Conflict(mRebuildState.Text(l, nil))
	}
	st, err := s.Q.DigestCampaignState(ctx, *issue.CampaignID)
	if err != nil {
		return DigestIssues{}, err
	}
	switch st.Status {
	case "draft", "approval", "approved", "scheduled":
	default:
		return DigestIssues{}, api.Conflict(mRebuildState.Text(l, nil))
	}
	posts, err := s.Q.DigestIssueItems(ctx, issue.ID)
	if err != nil {
		return DigestIssues{}, err
	}
	items := make([]BlockItem, 0, len(posts))
	for _, it := range posts {
		items = append(items, blockItemOf(it))
	}
	number, err := s.Q.DigestNextNumber(ctx, d.ID)
	if err != nil {
		return DigestIssues{}, err
	}
	if _, _, err := s.materialize(ctx, ws, d, issue, items, int(number), "issue", &p.UserID, mVersionRebuilt.Text(l, nil)); err != nil {
		if errors.Is(err, errNoTemplate) {
			return DigestIssues{}, api.Conflict(mIssueNoTemplate.Text(l, nil))
		}
		return DigestIssues{}, err
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "digest.issue_rebuilt", TargetType: "digest", TargetID: d.ID.String(), TargetName: d.Name,
		Metadata: map[string]any{"issue_id": issue.ID.String(), "campaign_id": issue.CampaignID.String(), "posts": len(items)}}); err != nil {
		return DigestIssues{}, err
	}
	if ap := s.approvals(); ap != nil {
		ap.CampaignChanged(ctx, ws, *issue.CampaignID)
	}
	if ch := s.checks(); ch != nil {
		if _, err := ch.Run(ctx, ws, *issue.CampaignID, readiness.RunOptions{Trigger: readiness.TriggerComposer, By: &p.UserID}); err != nil && !errors.Is(err, readiness.ErrRunning) {
			return DigestIssues{}, err
		}
	}
	return s.Issues(ctx, ws, digestID)
}

// nextIssue is the issue being collected, materialized as a campaign so it
// can be checked, previewed or tested as it would be built now.
func (s *Service) nextCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, digestID uuid.UUID) (uuid.UUID, error) {
	l := i18n.LangFrom(ctx)
	row, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: digestID})
	if err != nil {
		return uuid.Nil, api.MapError(err, "Digest")
	}
	d := row.Digest
	issue, err := s.Q.DigestIssueOpen(ctx, digestID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Off or paused: a next issue all the same, for previews and checks.
		sendAt, prepareAt, deadline := s.nextTimes(ctx, d, s.now())
		iid := uuid.Must(uuid.NewV7())
		if err := s.Q.DigestIssueInsert(ctx, store.DigestIssueInsertParams{ID: iid, DigestID: digestID, WorkspaceID: ws.ID, Mode: d.Mode, SendAt: sendAt,
			PrepareAt: prepareAt, Deadline: deadline, Now: s.now()}); err != nil && !isUnique(err) {
			return uuid.Nil, err
		}
		issue, err = s.Q.DigestIssueOpen(ctx, digestID)
	}
	if err != nil {
		return uuid.Nil, err
	}
	if issue.Status != "collecting" && issue.CampaignID != nil {
		return *issue.CampaignID, nil
	}
	all, err := s.items(ctx, d, 400, false)
	if err != nil {
		return uuid.Nil, err
	}
	posts := []BlockItem{}
	for _, it := range newItems(all) {
		posts = append(posts, blockItem(it.DigestItemsRow))
	}
	if len(posts) == 0 {
		// Nothing new yet: the newest posts that pass the filters show what
		// an issue looks like.
		for _, it := range all {
			if it.Status == "sent" || it.Status == "new" {
				posts = append(posts, blockItem(it.DigestItemsRow))
			}
		}
	}
	if len(posts) == 0 {
		return uuid.Nil, api.Conflict(mIssueNoItems.Text(l, nil))
	}
	number, err := s.Q.DigestNextNumber(ctx, digestID)
	if err != nil {
		return uuid.Nil, err
	}
	id, _, err := s.materialize(ctx, ws, d, issue, posts, int(number), "next", actorID(p), mVersionNext.Text(l, nil))
	if errors.Is(err, errNoTemplate) {
		return uuid.Nil, api.Conflict(mIssueNoTemplate.Text(l, nil))
	}
	return id, err
}

// DigestCheck is the next issue checked by Send Readiness.
type DigestCheck struct {
	CampaignID string `json:"campaignId"`
	RunID      string `json:"runId"`
	Verdict    string `json:"verdict"`
}

// Check runs Send Readiness on the next issue as it would be built now.
func (s *Service) Check(ctx context.Context, ws *store.Workspace, p *authz.Principal, digestID uuid.UUID) (DigestCheck, error) {
	id, err := s.nextCampaign(ctx, ws, p, digestID)
	if err != nil {
		return DigestCheck{}, err
	}
	ch := s.checks()
	if ch == nil {
		return DigestCheck{CampaignID: id.String()}, nil
	}
	run, err := ch.Run(ctx, ws, id, readiness.RunOptions{Trigger: readiness.TriggerManual, By: actorID(p)})
	if errors.Is(err, readiness.ErrRunning) {
		return DigestCheck{CampaignID: id.String()}, nil
	}
	if err != nil {
		return DigestCheck{}, err
	}
	return DigestCheck{CampaignID: id.String(), RunID: run.ID, Verdict: run.Verdict}, nil
}

// DigestTestInput sends the next issue as a test.
type DigestTestInput struct {
	To []string `json:"to" minItems:"1" maxItems:"10"`
}

// SendTest sends the next issue, as it would be built now, as a test.
func (s *Service) SendTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, digestID uuid.UUID, in DigestTestInput) (campaigns.ComposerTestSendResult, error) {
	id, err := s.nextCampaign(ctx, ws, p, digestID)
	if err != nil {
		return campaigns.ComposerTestSendResult{}, err
	}
	if s.Tests == nil || s.Tests() == nil {
		return campaigns.ComposerTestSendResult{}, api.Err(http.StatusServiceUnavailable, "tests_unavailable", mTestsUnavailable.Text(i18n.LangFrom(ctx), nil))
	}
	return s.Tests().SendTest(ctx, ws, p, id, campaigns.ComposerTestSend{To: in.To})
}

// issueRequest describes an issue to the approvals domain.
func issueRequest(ctx context.Context, s *Service, d store.Digest, number int, held bool) approvals.IssueRequest {
	return approvals.IssueRequest{Digest: d.Name, Number: number, Held: held, Actor: s.systemActor(ctx, d)}
}
