package texting

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

// MoveScheduled moves a scheduled SMS campaign that hasn't sent a text to
// another time: the calendar's reschedule (review 2026-11 CA-9). Each
// queued text gets its new moment in its recipient's zone, inside quiet
// hours, as a send would have planned it; held texts keep waiting for
// their sender.
func (s *Service) MoveScheduled(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, to approvals.ChannelMove) error {
	if !p.Can(authz.SMSSend) {
		return api.Forbidden(authz.SMSSend, s.helpers(ctx, ws, authz.SMSSend))
	}
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	if c.Status != "scheduled" {
		return api.Conflict(say(ctx, mUnscheduleOnly, nil))
	}
	// Moving a send schedules it again, on the mover's behalf.
	if err := audienceScope(ctx, p, c); err != nil {
		return err
	}
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return err
	}
	w := r.Window
	moved := c
	if to.Local {
		if to.LocalMinute < w.From || to.LocalMinute >= w.To {
			return api.Invalid("body.time", say(ctx, mInsideWindow, i18n.Args{"window": w.String()}))
		}
		d, m := to.LocalDate, int32(to.LocalMinute)
		moved.ScheduleMode, moved.SendAt, moved.LocalDate, moved.LocalMinute = "local", nil, &d, &m
	} else {
		at := to.At
		moved.ScheduleMode, moved.SendAt, moved.LocalDate, moved.LocalMinute = "at", &at, nil, nil
	}
	now := s.now()
	var first time.Time
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var cur string
		if err := tx.QueryRow(ctx, `SELECT status FROM campaigns WHERE id = $1 FOR UPDATE`, c.ID).Scan(&cur); err != nil {
			return err
		}
		if cur != "scheduled" {
			return api.Conflict(say(ctx, mUnscheduleOnly, nil))
		}
		var sent int64
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status NOT IN ('queued','held','skipped')`, c.ID).Scan(&sent); err != nil {
			return err
		}
		if sent > 0 {
			return api.Conflict(say(ctx, mTextsWentOut, nil))
		}
		// One new moment per zone: a campaign's texts share a handful of zones.
		rows, err := tx.Query(ctx, `SELECT DISTINCT time_zone FROM channel_messages WHERE campaign_id = $1 AND status = 'queued'`, c.ID)
		if err != nil {
			return err
		}
		zones, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		times, closes := make([]time.Time, len(zones)), make([]time.Time, len(zones))
		for i, z := range zones {
			t := s.sendTime(moved, w, Recipient{Zone: z}, now)
			times[i], closes[i] = t, w.Closes(t, location(z))
			if first.IsZero() || t.Before(first) {
				first = t
			}
		}
		if first.IsZero() {
			first = s.sendTime(moved, w, Recipient{Zone: ws.Timezone}, now)
		}
		if _, err := tx.Exec(ctx, `UPDATE channel_messages m SET send_after = u.t, expires_at = u.e, updated_at = $5
			FROM unnest($2::text[], $3::timestamptz[], $4::timestamptz[]) AS u(z, t, e)
			WHERE m.campaign_id = $1 AND m.status = 'queued' AND m.time_zone = u.z`, c.ID, zones, times, closes, now); err != nil {
			return err
		}
		if err := q.SmsCampaignScheduleSave(ctx, store.SmsCampaignScheduleSaveParams{ScheduleMode: moved.ScheduleMode, SendAt: moved.SendAt, LocalDate: moved.LocalDate,
			LocalMinute: moved.LocalMinute, PendingPolicy: c.PendingPolicy, Thresholds: c.Thresholds, CampaignID: c.ID}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE sms_sends SET scheduled_at = $2 WHERE campaign_id = $1`, c.ID, first); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET scheduled_at = $2, updated_at = $3 WHERE id = $1`, c.ID, first, now); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.rescheduled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before:   map[string]any{"scheduled_at": c.ScheduledAt, "schedule": scheduleLabel(c, location(ws.Timezone))},
			After:    map[string]any{"scheduled_at": first, "schedule": scheduleLabel(moved, location(ws.Timezone))},
			Metadata: map[string]any{"channel": "sms", "permission": string(authz.CampaignsSend)}})
	})
	if err != nil {
		return err
	}
	s.kick(ctx)
	return nil
}
