package digests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// FeedRetry is how often a feed that failed at preparation is read again,
// until the send time.
const FeedRetry = 15 * time.Minute

// Prepare builds an issue from the feed's new posts and checks it: the
// feed is read again; no new posts skips the issue; a feed that fails is
// retried every 15 minutes until the send time. The issue is a campaign
// that runs Send Readiness and the approval rules: Blocked holds it for a
// person, a matching rule asks the approvers in the owner's name, and in
// draft mode it becomes a draft someone sends through Review & send.
func (s *Service) Prepare(ctx context.Context, issueID uuid.UUID) error {
	issue, err := s.Q.DigestIssueByID(ctx, issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if issue.Status != "collecting" && issue.Status != "preparing" {
		return nil
	}
	d, err := s.Q.DigestByID(ctx, issue.DigestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if d.State != "on" {
		return nil
	}
	ws, err := s.workspace(ctx, d.WorkspaceID)
	if err != nil {
		return err
	}
	now := s.now()
	// The feed, read at preparation.
	res, err := s.read(ctx, ws, d, ViaPreparation)
	if err != nil {
		return err
	}
	if !res.OK {
		return s.feedFailed(ctx, ws, d, issue, res.Err, now)
	}
	if d, err = s.Q.DigestByID(ctx, d.ID); err != nil {
		return err
	}
	all, err := s.items(ctx, d, 400, false)
	if err != nil {
		return err
	}
	fresh := newItems(all)
	if len(fresh) == 0 {
		if err := s.skip(ctx, ws, d, issue, "no_items", "", nil); err != nil {
			return err
		}
		return s.ensureCollecting(ctx, ws, d.ID)
	}
	sched := ParseSchedule(d.Schedule, ws.Timezone)
	// An item-count issue gets its send time now: after the preparation
	// lead, inside the sending window, at most once a day.
	if sched.Frequency == FreqItems || issue.SendAt == nil {
		var lastSent *time.Time
		if last, err := s.Q.DigestLastSentIssue(ctx, d.ID); err == nil {
			lastSent = last.StartedAt
		}
		send := sched.ItemsSendAt(now, time.Duration(d.PrepareHours)*time.Hour, lastSent)
		deadline := send.Add(time.Duration(d.LateHours) * time.Hour)
		issue.SendAt, issue.Deadline = &send, &deadline
		if err := s.Q.DigestIssueSetTimes(ctx, store.DigestIssueSetTimesParams{ID: issue.ID, SendAt: &send, PrepareAt: &now, Deadline: &deadline, Mode: d.Mode, Now: now}); err != nil {
			return err
		}
	}
	posts := make([]BlockItem, 0, len(fresh))
	for _, it := range fresh {
		posts = append(posts, blockItem(it.DigestItemsRow))
	}
	number, err := s.Q.DigestNextNumber(ctx, d.ID)
	if err != nil {
		return err
	}
	label := mVersionPrepared.Text(versionLang(ctx, s.Q, d.OwnerID), i18n.Args{"number": fmt.Sprint(number), "count": len(posts)})
	campaignID, b, err := s.materialize(ctx, ws, d, issue, posts, int(number), "issue", d.OwnerID, label)
	if err != nil {
		// The template doesn't render: the issue is held for a person.
		s.log().WarnContext(ctx, "build a digest issue", "digest", d.ID, "err", err)
		return s.holdIssue(ctx, ws, d, issue, nil, "readiness", false, now)
	}
	issue.CampaignID = &campaignID
	// Send Readiness at preparation.
	var run *readiness.ReadinessRun
	if ch := s.checks(); ch != nil {
		r, err := ch.Run(ctx, ws, campaignID, readiness.RunOptions{Trigger: readiness.TriggerSchedule})
		if err != nil && !errors.Is(err, readiness.ErrRunning) {
			return err
		}
		if err == nil {
			run = &r
		}
	}
	blocked := run != nil && run.Verdict == readiness.VerdictBlocked
	if d.Mode == "draft" {
		return s.draftReady(ctx, ws, d, issue, int(number), len(b.Items), now)
	}
	// The approval rules, for the owner.
	requested := false
	if ap := s.approvals(); ap != nil && d.OwnerID != nil {
		rid, err := ap.RequestForIssue(ctx, ws, *d.OwnerID, campaignID, issueRequest(ctx, s, d, int(number), blocked))
		switch {
		case errors.Is(err, approvals.ErrNoApprovers):
			return s.holdIssue(ctx, ws, d, issue, run, "no_approvers", false, now)
		case err != nil:
			return err
		}
		requested = rid != uuid.Nil
	}
	if blocked {
		return s.holdIssue(ctx, ws, d, issue, run, "readiness", requested, now)
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		sched, _ := json.Marshal(map[string]any{"mode": "at", "at": issue.SendAt.UTC().Format(time.RFC3339), "timezone": location(ParseSchedule(d.Schedule, ws.Timezone).Timezone).String()})
		if err := q.DigestCampaignSchedule(ctx, store.DigestCampaignScheduleParams{ID: campaignID, At: *issue.SendAt, Schedule: sched, Now: now}); err != nil {
			return err
		}
		if err := q.DigestIssuePrepared(ctx, store.DigestIssuePreparedParams{ID: issue.ID, Status: "prepared", Now: now, ItemCount: int32(len(b.Items)),
			SendAt: issue.SendAt, Deadline: issue.Deadline, CampaignID: &campaignID}); err != nil {
			return err
		}
		return s.auditPrepared(ctx, q, ws, d, issue, int(number), len(b.Items), requested, "prepared")
	})
}

func (s *Service) auditPrepared(ctx context.Context, q *store.Queries, ws *store.Workspace, d store.Digest, issue store.DigestIssue, number, posts int, requested bool, status string) error {
	summary := fmt.Sprintf("issue #%d · %d %s", number, posts, map[bool]string{true: "post", false: "posts"}[posts == 1])
	if requested {
		summary += " · approval requested"
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "digest.issue_prepared", TargetType: "digest", TargetID: d.ID.String(), TargetName: d.Name,
		Actor: s.systemActor(ctx, d), After: map[string]any{"status": status, "posts": posts, "send_at": issue.SendAt},
		Metadata: map[string]any{"issue_id": issue.ID.String(), "number": number, "summary": summary}})
}

// holdIssue holds a prepared issue for a person: nothing is sent anyway.
// The owner is told, and so are the approvers when a request is open.
func (s *Service) holdIssue(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, run *readiness.ReadinessRun, reason string,
	requested bool, now time.Time) error {
	number, _ := s.Q.DigestNextNumber(ctx, d.ID)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		held := now
		if err := q.DigestIssuePrepared(ctx, store.DigestIssuePreparedParams{ID: issue.ID, Status: "held", Now: now, HeldAt: &held, HoldReason: reason,
			ItemCount: int32(s.countIssueItems(ctx, q, issue.ID)), SendAt: issue.SendAt, Deadline: issue.Deadline, CampaignID: issue.CampaignID}); err != nil {
			return err
		}
		if issue.CampaignID != nil {
			st, err := q.DigestCampaignState(ctx, *issue.CampaignID)
			if err != nil {
				return err
			}
			if st.Status == "scheduled" {
				back := "draft"
				if open, err := q.DigestOpenApproval(ctx, issue.CampaignID); err == nil {
					back = map[string]string{"pending": "approval", "approved": "approved"}[open.Status]
				}
				if err := q.DigestCampaignUnschedule(ctx, store.DigestCampaignUnscheduleParams{ID: *issue.CampaignID, Status: back, Now: now}); err != nil {
					return err
				}
			}
			if err := q.DigestResolveHolds(ctx, store.DigestResolveHoldsParams{CampaignID: *issue.CampaignID, Now: now}); err != nil {
				return err
			}
		}
		block := ""
		if run != nil {
			for _, ch := range run.Checks {
				if ch.Status == readiness.Block {
					block = firstNonEmpty(ch.FlatTitle, ch.Title)
					break
				}
			}
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "digest.issue_held", TargetType: "digest", TargetID: d.ID.String(), TargetName: d.Name,
			Actor: s.systemActor(ctx, d), Reason: block, After: map[string]any{"status": "held", "hold": reason},
			Metadata: map[string]any{"issue_id": issue.ID.String(), "number": number}}); err != nil {
			return err
		}
		return s.notifyHeld(ctx, tx, q, ws, d, issue, int(number), reason, block, requested)
	})
}

func (s *Service) countIssueItems(ctx context.Context, q *store.Queries, issueID uuid.UUID) int {
	rows, err := q.DigestIssueItemCounts(ctx, []uuid.UUID{issueID})
	if err != nil || len(rows) == 0 {
		return 0
	}
	return int(rows[0].N)
}

// notifyHeld tells the owner, and the approvers when a request is open,
// that an issue is held ("Digest issue held"), each in their language.
func (s *Service) notifyHeld(ctx context.Context, tx pgx.Tx, q *store.Queries, ws *store.Workspace, d store.Digest, issue store.DigestIssue, number int,
	reason, block string, requested bool) error {
	loc := location(ParseSchedule(d.Schedule, ws.Timezone).Timezone)
	name := issueName(d.Name, number)
	// The check's own words follow a colon: first letter lowered, no full stop.
	what := trimDot(lowerFirst(block))
	link := "/w/" + ws.Slug + "/campaigns/digests/" + d.ID.String()
	if issue.CampaignID != nil {
		link = "/w/" + ws.Slug + "/campaigns/" + issue.CampaignID.String() + "/review"
	}
	notification := func(l i18n.Lang) notify.Notification {
		date, clock := whenIn(l, issue.Deadline, loc)
		args := i18n.Args{"issue": name, "date": date, "time": clock, "block": what}
		body := mHeldReadiness
		switch {
		case reason == "no_approvers":
			body = mHeldNoApprovers
		case reason == "changes":
			body = mHeldChanges
		case reason == "rejected":
			body = mHeldRejected
		case what != "":
			body = mHeldBlocked
		}
		return notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindDigestIssueHeld, Severity: "bad", Title: mHeldTitle.Text(l, nil), Body: body.Text(l, args),
			Summary: mHeldSummary.Text(l, args), Link: link, Data: map[string]any{"digest_id": d.ID.String(), "issue_id": issue.ID.String(), "reason": reason}}
	}
	targets := []uuid.UUID{}
	if d.OwnerID != nil {
		targets = append(targets, *d.OwnerID)
	}
	if requested {
		ids, err := notify.MembersWith(ctx, tx, ws.ID, authz.CampaignsApprove)
		if err != nil {
			return err
		}
		targets = append(targets, ids...)
	}
	told := map[uuid.UUID]bool{}
	for _, u := range targets {
		if told[u] {
			continue
		}
		told[u] = true
		if err := notify.ToUser(ctx, tx, u, notification(memberLang(ctx, q, u))); err != nil {
			return err
		}
	}
	return q.DigestIssueNotified(ctx, store.DigestIssueNotifiedParams{ID: issue.ID, Kind: "held"})
}

// lowerFirst lowers a sentence's first letter to follow a colon, leaving
// acronyms ("DMARC is…") alone.
func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) > 1 && unicode.IsUpper(r[0]) && unicode.IsLower(r[1]) {
		r[0] = unicode.ToLower(r[0])
	}
	return string(r)
}

func trimDot(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '.' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

// draftReady makes a draft-mode issue a draft campaign someone sends
// through Review & send: the owner and the members with campaigns:send
// are told.
func (s *Service) draftReady(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, number, posts int, now time.Time) error {
	loc := location(ParseSchedule(d.Schedule, ws.Timezone).Timezone)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.DigestIssuePrepared(ctx, store.DigestIssuePreparedParams{ID: issue.ID, Status: "draft", Now: now, ItemCount: int32(posts), SendAt: issue.SendAt,
			Deadline: issue.Deadline, CampaignID: issue.CampaignID}); err != nil {
			return err
		}
		if err := s.auditPrepared(ctx, q, ws, d, issue, number, posts, false, "draft"); err != nil {
			return err
		}
		name := issueName(d.Name, number)
		notification := func(l i18n.Lang) notify.Notification {
			date, clock := whenIn(l, issue.Deadline, loc)
			args := i18n.Args{"issue": name, "count": posts, "date": date, "time": clock}
			return notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindDigestDraftReady, Severity: "info", Title: mDraftTitle.Text(l, nil),
				Body: mDraftBody.Text(l, args), Summary: mDraftSummary.Text(l, args), Link: "/w/" + ws.Slug + "/campaigns/" + issue.CampaignID.String() + "/review",
				Data: map[string]any{"digest_id": d.ID.String(), "issue_id": issue.ID.String()}}
		}
		ids, err := notify.MembersWith(ctx, tx, ws.ID, authz.CampaignsSend)
		if err != nil {
			return err
		}
		if d.OwnerID != nil {
			ids = append([]uuid.UUID{*d.OwnerID}, ids...)
		}
		told := map[uuid.UUID]bool{}
		for _, u := range ids {
			if told[u] {
				continue
			}
			told[u] = true
			if err := notify.ToUser(ctx, tx, u, notification(memberLang(ctx, q, u))); err != nil {
				return err
			}
		}
		return q.DigestIssueNotified(ctx, store.DigestIssueNotifiedParams{ID: issue.ID, Kind: "draft"})
	})
}

// feedFailed records a failed read at preparation: the issue is retried
// every 15 minutes until its send time, then skipped. The owner hears at
// the first failure and when the issue is skipped.
func (s *Service) feedFailed(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, readErr error, now time.Time) error {
	tries := storedFeedTries(issue.FeedTries)
	kind := "unreachable"
	detail, text := mTryFailed.Text(i18n.Default, nil), mTryFailed.Keep(nil)
	if readErr != nil {
		detail, text = readErr.Error(), feed.TextOf(readErr, s.feedAddress(ctx, d))
		if feed.IsNotFeed(readErr) {
			kind = "invalid"
		}
		if fe, ok := feed.AsFetchError(readErr); ok {
			detail = fe.Detail
			if fe.Kind == feed.ErrTimeout {
				kind, detail, text = "timeout", mTryTimedOut.Text(i18n.Default, nil), mTryTimedOut.Keep(nil)
			}
		}
	}
	try := storedFeedTry{DigestFeedTry: DigestFeedTry{At: now, Error: detail, Kind: kind}}
	if !text.IsZero() {
		try.Text = &text
	}
	tries = append(tries, try)
	tb, _ := json.Marshal(tries)
	if err := s.Q.DigestIssuePreparing(ctx, store.DigestIssuePreparingParams{ID: issue.ID, FeedTries: tb, Now: now}); err != nil {
		return err
	}
	if len(tries) == 1 {
		return s.notifyFeed(ctx, ws, d, issue, readErr, false)
	}
	return nil
}

// notifyFeed is "Digest feed failing": at the first failed preparation,
// then when the issue is skipped.
func (s *Service) notifyFeed(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, readErr error, skipped bool) error {
	if d.OwnerID == nil {
		return nil
	}
	loc := location(ParseSchedule(d.Schedule, ws.Timezone).Timezone)
	feedURL := s.feedAddress(ctx, d)
	l := memberLang(ctx, s.Q, *d.OwnerID)
	_, send := whenIn(l, issue.SendAt, loc)
	args := i18n.Args{"digest": d.Name, "time": send, "address": shortURL(feedURL)}
	body := mFeedFailingUnknown
	switch {
	case skipped:
		body = mFeedSkipped
	case readErr != nil:
		body, args["why"] = mFeedFailing, trimDot(feedText(l, readErr, feedURL))
	}
	n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindDigestFeedFailing, Severity: "warn", Title: mFeedTitle.Text(l, nil), Body: body.Text(l, args),
		Summary: mFeedFailingSummary.Text(l, args), Link: "/w/" + ws.Slug + "/campaigns/digests/" + d.ID.String(),
		Data: map[string]any{"digest_id": d.ID.String(), "issue_id": issue.ID.String(), "skipped": skipped}}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return notify.ToUser(ctx, tx, *d.OwnerID, n)
	})
}
