package reports

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/retryaudience"
	"github.com/hostrepute/reputecast/internal/store"
)

// "Retry after fixing" (review AN-7, R5.L5): the addresses blocked for the
// sender's reputation are valid, so they are never suppressed for it.
// Once the block is fixed, a retry sends them the issue again, once; the
// Investigate dialog says how many it would go to, or which retry exists.
// The campaigns domain makes the retry (campaigns.CreateRetry) and it goes
// out from Review & send.

// finishedSend are the statuses of a campaign whose send has ended.
var finishedSend = map[string]bool{"sent": true, "cancelled": true, "failed": true}

// blockedRetry is the retry state of a campaign's blocked addresses.
func (s *Service) blockedRetry(ctx context.Context, ws *store.Workspace, c store.ReportCampaignRow) (*ReportBlockedRetry, error) {
	out := &ReportBlockedRetry{}
	if issue, ok := retryaudience.Of(c.Audience); ok {
		out.Unavailable = "is_retry"
		if name, err := s.Q.RetryIssueName(ctx, store.RetryIssueNameParams{WorkspaceID: ws.ID, ID: issue}); err == nil {
			out.Of = &ReportCampaignRef{ID: issue.String(), Name: name}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return out, nil
	}
	if r, err := s.Q.RetryCampaignOf(ctx, store.RetryCampaignOfParams{WorkspaceID: ws.ID, CampaignID: c.ID.String()}); err == nil {
		out.Campaign = &ReportRetryCampaign{ID: r.ID.String(), Name: r.Name, Status: r.Status, StartedAt: r.StartedAt}
		return out, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if !finishedSend[c.Status] {
		out.Unavailable = "not_finished"
		return out, nil
	}
	keep, err := retryaudience.LoadIssue(ctx, s.Q, ws.ID, c.ID, uuid.Nil)
	if err != nil {
		return nil, err
	}
	out.Blocked = int64(keep.Len())
	if out.Blocked == 0 {
		out.Unavailable = "none_left"
		return out, nil
	}
	// Who it goes to now: the audience's own rules, as Review & send will
	// count them.
	if s.RetryAudience != nil {
		audience, err := retryaudience.With(c.Audience, c.ID)
		if err != nil {
			return nil, err
		}
		if out.Recipients, err = s.RetryAudience(ctx, ws, audience); err != nil {
			return nil, err
		}
	} else {
		out.Recipients = out.Blocked
	}
	return out, nil
}
