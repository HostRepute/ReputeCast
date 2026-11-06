package whatsapp

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// MoveScheduled moves a scheduled WhatsApp campaign that hasn't started to
// another time: the calendar's reschedule (review 2026-11 CA-9). Its
// messages are planned when the send starts, each in its recipient's zone
// and quiet hours, so the move changes the schedule and when the send
// starts: for a local time, when that time first comes anywhere.
func (s *Service) MoveScheduled(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, to approvals.ChannelMove) error {
	if err := s.requireSend(ctx, ws, p); err != nil {
		return err
	}
	l := langOf(ctx)
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	// Moving a send schedules it again, on the mover's behalf.
	if err := audienceScope(l, p, c); err != nil {
		return err
	}
	if !to.At.After(s.now()) {
		return api.Invalid("body.time", mFutureTime.Text(l, nil))
	}
	mode, at := "at", to.At
	var sendAt, localDate *time.Time
	var localMinute *int32
	if to.Local {
		if rl := s.rules(ctx, s.Q, ws); rl.Quiet && (to.LocalMinute < rl.Window.From || to.LocalMinute >= rl.Window.To) {
			return api.Invalid("body.time", mInsideWindow.Text(l, i18n.Args{"window": rl.Window.String()}))
		}
		d, m := to.LocalDate, int32(to.LocalMinute)
		mode, localDate, localMinute = "local", &d, &m
	} else {
		sendAt = &at
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var status, phase string
		if err := tx.QueryRow(ctx, `SELECT c.status, coalesce(sn.phase, '') FROM campaigns c LEFT JOIN whatsapp_sends sn ON sn.campaign_id = c.id WHERE c.id = $1 FOR UPDATE OF c`, id).
			Scan(&status, &phase); err != nil {
			return err
		}
		if status != "scheduled" || phase != "scheduled" {
			return api.Conflict(mMoveOnly.Text(l, nil))
		}
		if _, err := tx.Exec(ctx, `UPDATE whatsapp_campaigns SET schedule_mode = $2, send_at = $3, local_date = $4, local_minute = $5 WHERE campaign_id = $1`,
			id, mode, sendAt, localDate, localMinute); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE whatsapp_sends SET scheduled_at = $2 WHERE campaign_id = $1`, id, at); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET scheduled_at = $2, updated_at = $3 WHERE id = $1`, id, at, now); err != nil {
			return err
		}
		after := map[string]any{"scheduled_at": at, "mode": mode}
		if to.Local {
			after["local"] = to.LocalDate.Format("2006-01-02") + " " + time.Date(0, 1, 1, 0, to.LocalMinute, 0, 0, time.UTC).Format("15:04")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.rescheduled", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"scheduled_at": c.ScheduledAt, "mode": c.ScheduleMode}, After: after,
			Metadata: map[string]any{"channel": "whatsapp", "permission": string(authz.CampaignsSend)}})
	})
}
