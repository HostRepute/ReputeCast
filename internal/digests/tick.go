package digests

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// RecheckLead is how long before the send time Send Readiness runs again
// on a prepared issue, as for any scheduled campaign.
const RecheckLead = 15 * time.Minute

// Tick moves every digest along once (the minute job): it checks owners,
// reads feeds every 30 minutes, starts preparations, retries failed feeds,
// rechecks and starts prepared issues at their time, follows issues that
// are sending and settles those past their late limit.
func (s *Service) Tick(ctx context.Context) error {
	rows, err := s.Q.DigestsActive(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, d := range rows {
		if err := s.tickDigest(ctx, d); err != nil {
			s.log().WarnContext(ctx, "digest tick", "digest", d.ID, "err", err)
			errs = append(errs, err)
		}
	}
	_ = s.Q.DigestFeedReadsPrune(ctx, s.now().Add(-90*24*time.Hour))
	return errors.Join(errs...)
}

func (s *Service) tickDigest(ctx context.Context, d store.Digest) error {
	ws, err := s.workspace(ctx, d.WorkspaceID)
	if err != nil {
		return err
	}
	now := s.now()
	if d.State == "on" {
		paused, err := s.checkOwner(ctx, ws, d)
		if err != nil || paused {
			return err
		}
		if err := s.ensureCollecting(ctx, ws, d.ID); err != nil {
			return err
		}
		if d.LastReadAt == nil || now.Sub(*d.LastReadAt) >= ReadEvery {
			s.enqueue(ctx, ReadArgs{DigestID: d.ID, Via: ViaSchedule})
		}
	}
	issues, err := s.Q.DigestIssuesInFlight(ctx, d.ID)
	if err != nil {
		return err
	}
	for _, issue := range issues {
		if err := s.tickIssue(ctx, ws, d, issue, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) tickIssue(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, now time.Time) error {
	switch issue.Status {
	case "collecting":
		if d.State != "on" {
			return nil
		}
		sched := ParseSchedule(d.Schedule, ws.Timezone)
		if sched.Frequency == FreqItems {
			all, err := s.items(ctx, d, 400, false)
			if err != nil {
				return err
			}
			if len(newItems(all)) >= max(sched.Count, 1) {
				s.enqueue(ctx, PrepareArgs{IssueID: issue.ID})
			}
			return nil
		}
		if issue.PrepareAt != nil && !now.Before(*issue.PrepareAt) {
			s.enqueue(ctx, PrepareArgs{IssueID: issue.ID})
		}
	case "preparing":
		// The feed failed at preparation: again every 15 minutes, until the send time.
		if issue.SendAt != nil && !now.Before(*issue.SendAt) {
			return s.skipForFeed(ctx, ws, d, issue)
		}
		tries := feedTries(issue.FeedTries)
		if len(tries) == 0 || now.Sub(tries[len(tries)-1].At) >= FeedRetry {
			s.enqueue(ctx, PrepareArgs{IssueID: issue.ID})
		}
	case "prepared":
		return s.tickPrepared(ctx, ws, d, issue, now)
	case "held":
		if issue.Deadline != nil && !now.Before(*issue.Deadline) {
			if err := s.skip(ctx, ws, d, issue, "held", "", nil); err != nil {
				return err
			}
			return s.ensureCollecting(ctx, ws, d.ID)
		}
	case "draft":
		return s.tickDraft(ctx, ws, d, issue, now)
	case "sending":
		return s.followSend(ctx, ws, d, issue, now)
	}
	return nil
}

// tickPrepared rechecks a prepared issue 15 minutes before its send time,
// and starts it at its time once its approval (when one is needed) is in.
func (s *Service) tickPrepared(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, now time.Time) error {
	if issue.CampaignID == nil || issue.SendAt == nil {
		return nil
	}
	st, err := s.Q.DigestCampaignState(ctx, *issue.CampaignID)
	if err != nil {
		return err
	}
	switch st.Status {
	case "sending", "paused", "sent", "cancelled", "failed":
		return s.markSending(ctx, ws, d, issue, now)
	}
	if st.DeletedAt != nil {
		return s.skip(ctx, ws, d, issue, "person", "", nil)
	}
	// The open request.
	var request *store.ApprovalRequest
	if r, err := s.Q.DigestOpenApproval(ctx, issue.CampaignID); err == nil {
		request = &r
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if issue.Deadline != nil && !now.Before(*issue.Deadline) {
		reason := "not_approved"
		if request == nil || request.Status == "approved" {
			reason = "held"
		}
		if err := s.skip(ctx, ws, d, issue, reason, "", nil); err != nil {
			return err
		}
		return s.ensureCollecting(ctx, ws, d.ID)
	}
	// An approver sent it back: it waits for a person.
	if request == nil {
		if last, err := s.Q.ApprovalLatestForCampaign(ctx, issue.CampaignID); err == nil {
			switch last.Status {
			case "changes_requested":
				return s.holdIssue(ctx, ws, d, issue, nil, "changes", false, now)
			case "rejected":
				return s.holdIssue(ctx, ws, d, issue, nil, "rejected", false, now)
			}
		}
	}
	ch := s.checks()
	// Send Readiness again 15 minutes before the send.
	if ch != nil && now.Before(*issue.SendAt) && !now.Before(issue.SendAt.Add(-RecheckLead)) {
		latest, err := ch.Latest(ctx, *issue.CampaignID)
		if err != nil {
			return err
		}
		if latest == nil || latest.RanAt.Before(issue.SendAt.Add(-RecheckLead)) {
			run, err := ch.Run(ctx, ws, *issue.CampaignID, readiness.RunOptions{Trigger: readiness.TriggerSchedule})
			if errors.Is(err, readiness.ErrRunning) {
				return nil
			}
			if err != nil {
				return err
			}
			if run.Verdict == readiness.VerdictBlocked {
				return s.holdIssue(ctx, ws, d, issue, &run, "readiness", request != nil && request.Status == "pending", now)
			}
		}
		return nil
	}
	if now.Before(*issue.SendAt) {
		return nil
	}
	// Its time: it waits for a pending approval.
	if request != nil && request.Status == "pending" {
		return nil
	}
	if d.OwnerID == nil || s.starter() == nil || s.approvals() == nil {
		return nil
	}
	// The send-time recheck: Send Readiness again, and the approval must
	// still cover the issue.
	rc, err := s.approvals().SendRecheck(ctx, *issue.CampaignID)
	if err != nil {
		return err
	}
	if rc.Verdict == readiness.VerdictBlocked {
		// An approval that stopped covering the issue (it changed) waits for
		// the approvers like a pending one; a block waits for a person.
		if r, err := s.Q.DigestOpenApproval(ctx, issue.CampaignID); err == nil && r.Status == "pending" {
			return nil
		}
		run, _ := ch.Latest(ctx, *issue.CampaignID)
		if run != nil && run.Verdict != readiness.VerdictBlocked {
			// Not a Send Readiness block: the approval gap (invalidated).
			return nil
		}
		return s.holdIssue(ctx, ws, d, issue, run, "readiness", false, now)
	}
	err = s.starter().StartIssue(ctx, *issue.CampaignID, *d.OwnerID, sending.Recheck{Verdict: rc.Verdict, Blocking: rc.Blocking, Throttles: rc.Throttles,
		Audience: rc.Audience, RunID: rc.RunID})
	if err != nil {
		if errors.Is(err, sending.ErrNotReady) {
			run, _ := ch.Latest(ctx, *issue.CampaignID)
			return s.holdIssue(ctx, ws, d, issue, run, "readiness", false, now)
		}
		if errors.Is(err, sending.ErrUnavailable) {
			return nil
		}
		return err
	}
	return s.markSending(ctx, ws, d, issue, now)
}

// tickDraft follows a draft-mode issue: someone sends it through Review &
// send; not sent within the late limit, it is marked Not sent.
func (s *Service) tickDraft(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, now time.Time) error {
	if issue.CampaignID == nil {
		return nil
	}
	st, err := s.Q.DigestCampaignState(ctx, *issue.CampaignID)
	if err != nil {
		return err
	}
	switch st.Status {
	case "sending", "paused", "sent", "cancelled", "failed":
		return s.markSending(ctx, ws, d, issue, now)
	case "scheduled":
		return nil // someone scheduled it
	}
	if st.DeletedAt != nil {
		return s.skip(ctx, ws, d, issue, "person", "", nil)
	}
	if issue.Deadline != nil && !now.Before(*issue.Deadline) {
		if err := s.skip(ctx, ws, d, issue, "not_sent", "", nil); err != nil {
			return err
		}
		return s.ensureCollecting(ctx, ws, d.ID)
	}
	return nil
}

// markSending records that an issue started: it gets its number, its posts
// are used up, and the next issue starts collecting.
func (s *Service) markSending(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, now time.Time) error {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		locked, err := q.DigestIssueLock(ctx, issue.ID)
		if err != nil {
			return err
		}
		switch locked.Status {
		case "prepared", "held", "draft":
		default:
			return nil
		}
		number, err := q.DigestNextNumber(ctx, d.ID)
		if err != nil {
			return err
		}
		if err := q.DigestIssueSending(ctx, store.DigestIssueSendingParams{ID: issue.ID, Number: &number, Now: now}); err != nil {
			return err
		}
		if locked.CampaignID != nil {
			if err := q.DigestCampaignSetRole(ctx, store.DigestCampaignSetRoleParams{ID: *locked.CampaignID, DigestRole: "issue", Name: issueName(d.Name, int(number))}); err != nil {
				return err
			}
		}
		// A send counts as a good read of the feed for the failure streak.
		return q.DigestSetFeedSkips(ctx, store.DigestSetFeedSkipsParams{ID: d.ID, FeedSkips: 0})
	})
	if err != nil {
		return err
	}
	return s.ensureCollecting(ctx, ws, d.ID)
}

// followSend settles a sending issue once its campaign finished.
func (s *Service) followSend(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, now time.Time) error {
	if issue.CampaignID == nil {
		return nil
	}
	st, err := s.Q.DigestCampaignState(ctx, *issue.CampaignID)
	if err != nil {
		return err
	}
	switch st.Status {
	case "sent", "cancelled", "failed":
		at := now
		if st.CompletedAt != nil {
			at = *st.CompletedAt
		}
		return s.Q.DigestIssueSent(ctx, store.DigestIssueSentParams{ID: issue.ID, Now: at})
	}
	return nil
}

// skipForFeed skips an issue whose feed couldn't be read by its send time;
// three such skips in a row pause the digest and tell its owner.
func (s *Service) skipForFeed(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue) error {
	if err := s.skip(ctx, ws, d, issue, "feed", "", nil); err != nil {
		return err
	}
	_ = s.notifyFeed(ctx, ws, d, issue, nil, true)
	streak := d.FeedSkips + 1
	if err := s.Q.DigestSetFeedSkips(ctx, store.DigestSetFeedSkipsParams{ID: d.ID, FeedSkips: streak}); err != nil {
		return err
	}
	if streak >= 3 {
		recent, err := s.Q.DigestIssues(ctx, store.DigestIssuesParams{DigestID: d.ID, Lim: 3})
		if err != nil {
			return err
		}
		skips := []map[string]any{}
		for _, r := range recent {
			if r.Status == "skipped" && r.SkipReason == "feed" && r.SendAt != nil {
				kind := "unreachable"
				tries := feedTries(r.FeedTries)
				if len(tries) > 0 {
					kind = tries[len(tries)-1].Kind
				}
				skips = append(skips, map[string]any{"send_at": r.SendAt, "kind": kind})
			}
		}
		fresh, err := s.Q.DigestByID(ctx, d.ID)
		if err != nil {
			return err
		}
		return s.pause(ctx, ws, fresh, "feed", "", nil, map[string]any{"skips": skips})
	}
	return s.ensureCollecting(ctx, ws, d.ID)
}

// checkOwner pauses a digest whose owner can't send its issues any more:
// they lost campaigns:send (automatic mode) or campaigns:edit (draft
// mode), were deactivated or left the workspace.
func (s *Service) checkOwner(ctx context.Context, ws *store.Workspace, d store.Digest) (bool, error) {
	if d.OwnerID == nil {
		return false, nil
	}
	m, err := s.member(ctx, s.Q, ws.ID, *d.OwnerID)
	if err != nil {
		return false, err
	}
	need := authz.CampaignsSend
	if d.Mode == "draft" {
		need = authz.CampaignsEdit
	}
	if m != nil && m.Active && m.Perms.Has(need) {
		return false, nil
	}
	detail := map[string]any{}
	if u, err := s.Q.GetUser(ctx, *d.OwnerID); err == nil {
		detail["owner"] = u.Name
	}
	if m == nil || !m.Active {
		detail["left"] = true
	} else {
		detail["to"] = m.RoleName
		// The role it had: the last audit entry of a role change for the member.
		if from := s.previousRole(ctx, ws.ID, *d.OwnerID); from != "" {
			detail["from"] = from
		}
	}
	detail["permission"] = string(need)
	if err := s.pause(ctx, ws, d, "owner", "", nil, detail); err != nil {
		return false, err
	}
	// The issue waiting to go out is skipped: it can't send on its owner's authority.
	if issue, err := s.Q.DigestIssueOpen(ctx, d.ID); err == nil {
		if issue.Status == "collecting" {
			_ = s.Q.DigestIssueDelete(ctx, issue.ID)
		} else if err := s.skip(ctx, ws, d, issue, "person", "the owner lost "+string(need), nil); err != nil {
			return true, err
		}
	}
	return true, nil
}

// previousRole is the role a member had before their last role change.
func (s *Service) previousRole(ctx context.Context, wsID, userID uuid.UUID) string {
	before, err := s.Q.DigestPreviousRole(ctx, store.DigestPreviousRoleParams{WorkspaceID: &wsID, UserID: userID.String()})
	if err != nil {
		return ""
	}
	return before
}
