package push

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// MoveScheduled moves a scheduled push campaign that hasn't started to
// another time: the calendar's reschedule (review 2026-11 CA-9). Push goes
// out at one time for everyone, so a time in each recipient's zone is
// refused.
func (s *Service) MoveScheduled(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, to approvals.ChannelMove) error {
	l := i18n.LangFrom(ctx)
	if !canSend(p) {
		perm := authz.PushSend
		if !p.Can(authz.CampaignsSend) {
			perm = authz.CampaignsSend
		}
		return api.Forbidden(perm, s.helpers(ctx, ws, perm))
	}
	if to.Local {
		return api.Invalid("body.mode", mProblemMoveLocal.Text(l, nil))
	}
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	// Moving a send schedules it again, on the mover's behalf.
	if err := audienceScope(l, p, c); err != nil {
		return err
	}
	now := s.now()
	if !to.At.After(now) {
		return api.Invalid("body.time", mProblemFutureTime.Text(l, nil))
	}
	at := to.At
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var status, phase string
		if err := tx.QueryRow(ctx, `SELECT c.status, coalesce(ps.phase, '') FROM campaigns c LEFT JOIN push_sends ps ON ps.campaign_id = c.id WHERE c.id = $1 FOR UPDATE OF c`, id).
			Scan(&status, &phase); err != nil {
			return err
		}
		if status != "scheduled" || phase != "scheduled" {
			return api.Conflict(mProblemMove.Text(l, nil))
		}
		if err := q.PushCampaignScheduleSave(ctx, store.PushCampaignScheduleSaveParams{CampaignID: id, ScheduleMode: "at", SendAt: &at, Outside: c.Outside}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE push_sends SET scheduled_at = $2 WHERE campaign_id = $1`, id, at); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET scheduled_at = $2, updated_at = $3 WHERE id = $1`, id, at, now); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.rescheduled", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"scheduled_at": c.ScheduledAt}, After: map[string]any{"scheduled_at": at},
			Metadata: map[string]any{"channel": "push", "permission": string(authz.CampaignsSend)}})
	})
}
