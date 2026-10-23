package polls

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// SetPollOpen closes a poll now, or reopens it: answers made after
// reopening count as usual. Both are audited.
func (s *Service) SetPollOpen(ctx context.Context, ws *store.Workspace, p *authz.Principal, pollID uuid.UUID, open bool) (PollResults, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.PollByID(ctx, store.PollByIDParams{WorkspaceID: ws.ID, ID: pollID})
		if err != nil {
			return api.MapError(err, "Poll")
		}
		now := s.now()
		closed := r.ClosedAt != nil || (r.ClosesAt != nil && !now.Before(*r.ClosesAt))
		target := "poll"
		name := r.Question
		if r.CampaignName != "" {
			name = r.CampaignName + " · " + r.Question
		}
		if !open {
			if closed {
				return api.Conflict(mPollClosed.Text(i18n.LangFrom(ctx), nil))
			}
			if err := q.PollSetClosed(ctx, store.PollSetClosedParams{ID: r.ID, ClosedAt: &now, ClosedBy: userOf(p), ReopenedAt: r.ReopenedAt, Now: now}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "poll.closed", TargetType: target, TargetID: r.ID.String(), TargetName: name,
				Before: map[string]any{"status": "open"}, After: map[string]any{"status": "closed"}, Metadata: campaignMeta(r.CampaignID)})
		}
		if !closed {
			return api.Conflict(mPollOpen.Text(i18n.LangFrom(ctx), nil))
		}
		// Reopening clears a scheduled close that has passed: the poll stays
		// open until a person closes it.
		var closes *time.Time
		if r.ClosesAt != nil && now.Before(*r.ClosesAt) {
			closes = r.ClosesAt
		}
		if err := q.PollSetClosesAt(ctx, store.PollSetClosesAtParams{ID: r.ID, ClosesAt: closes, Now: now}); err != nil {
			return err
		}
		if err := q.PollSetClosed(ctx, store.PollSetClosedParams{ID: r.ID, ReopenedAt: &now, Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "poll.reopened", TargetType: target, TargetID: r.ID.String(), TargetName: name,
			Before: map[string]any{"status": "closed", "closedAt": r.ClosedAt, "closesAt": r.ClosesAt}, After: map[string]any{"status": "open", "closesAt": closes},
			Metadata: campaignMeta(r.CampaignID)})
	})
	if err != nil {
		return PollResults{}, err
	}
	return s.PollResultsOf(ctx, ws, p, pollID)
}

func campaignMeta(id *uuid.UUID) map[string]any {
	if id == nil {
		return nil
	}
	return map[string]any{"campaign": id.String()}
}

// PollTestAnswer is one test address's answer to a campaign's poll.
type PollTestAnswer struct {
	PollID  string    `json:"pollId"`
	BlockID string    `json:"blockId"`
	Email   string    `json:"email"`
	Answer  int       `json:"answer"`
	At      time.Time `json:"at"`
}

// PollTestAnswers are the test answers of a campaign's polls, never counted.
type PollTestAnswers struct {
	Items []PollTestAnswer `json:"items"`
}

// TestAnswers lists a campaign's test answers.
func (s *Service) TestAnswers(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (PollTestAnswers, error) {
	rows, err := s.Q.PollTestAnswers(ctx, store.PollTestAnswersParams{WorkspaceID: ws.ID, CampaignID: &campaignID})
	out := PollTestAnswers{Items: []PollTestAnswer{}}
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, PollTestAnswer{PollID: r.PollID.String(), BlockID: r.BlockID, Email: r.TestEmail, Answer: int(r.Answer), At: r.UpdatedAt})
	}
	return out, nil
}

// ClearTestAnswers deletes a campaign's test answers (audited: answers
// deleted).
func (s *Service) ClearTestAnswers(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (int64, error) {
	var n int64
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.PollCampaignContent(ctx, store.PollCampaignContentParams{WorkspaceID: ws.ID, ID: campaignID})
		if err != nil {
			return api.MapError(err, "Campaign")
		}
		n, err = q.PollTestAnswersClear(ctx, store.PollTestAnswersClearParams{WorkspaceID: ws.ID, CampaignID: &campaignID})
		if err != nil || n == 0 {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "poll.answers_deleted", TargetType: "campaign", TargetID: campaignID.String(), TargetName: c.Name,
			Metadata: map[string]any{"answers": n, "kind": "test"}})
	})
	return n, err
}
