package transactional

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// A scheduled message (sent with send_at) waits in the outbox until its
// time. Until then it can be cancelled or moved, by API and on the message
// page (review 2026-11 TX-1).

// TransactionalSchedule is a scheduled message's state after a cancel or a
// move.
type TransactionalSchedule struct {
	ID     string     `json:"id"`
	Status string     `json:"status" enum:"scheduled,cancelled"`
	SendAt *time.Time `json:"sendAt,omitempty"`
}

// TransactionalRescheduleInput moves a scheduled message.
type TransactionalRescheduleInput struct {
	SendAt time.Time `json:"sendAt" doc:"The new send time, up to 72 hours from now"`
}

// lockScheduled locks a message that still waits for its send time, or
// says why it can't change: it is gone, already on its way or done.
func (s *Service) lockScheduled(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (store.TxnScheduledForUpdateRow, error) {
	r, err := q.TxnScheduledForUpdate(ctx, store.TxnScheduledForUpdateParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return r, problem(http.StatusNotFound, "not_found", say(ctx, "message.notFound", i18n.Args{"id": id.String(), "days": MetadataRetentionDays}))
	}
	if err != nil {
		return r, err
	}
	switch {
	case r.CancelledAt != nil:
		return r, problem(http.StatusConflict, "not_scheduled", say(ctx, "schedule.cancelled", nil))
	case r.SendAt == nil || r.Status != "queued" || r.NextAttemptAt == nil || !r.NextAttemptAt.Equal(*r.SendAt) || r.ClaimedAt != nil ||
		!r.NextAttemptAt.After(s.now().Add(5*time.Second)):
		return r, problem(http.StatusConflict, "not_scheduled", say(ctx, "schedule.notScheduled", nil))
	}
	return r, nil
}

// actorLabel names who acts: the person, or the API key.
func actorLabel(ctx context.Context) string {
	if p := api.PrincipalFrom(ctx); p != nil {
		return p.Name
	}
	return ""
}

// CancelScheduled cancels a message before its send time: its sealed
// payload goes, the log keeps it as Cancelled and message.cancelled is
// queued for webhooks. Audited.
func (s *Service) CancelScheduled(ctx context.Context, ws *store.Workspace, id uuid.UUID) (TransactionalSchedule, error) {
	now := s.now()
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := s.lockScheduled(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if err := q.TxnCancelScheduled(ctx, id); err != nil {
			return err
		}
		if err := q.TxnMarkCancelled(ctx, store.TxnMarkCancelledParams{ID: id, At: &now, By: actorLabel(ctx)}); err != nil {
			return err
		}
		// The sealed payload and the rendered content go now, as the cancel
		// dialog says.
		if err := q.TxnOutboxDone(ctx, []uuid.UUID{id}); err != nil {
			return err
		}
		if err := q.TxnRetentionContentRemove(ctx, []uuid.UUID{id}); err != nil {
			return err
		}
		if err := q.TxnRetentionContentRemoved(ctx, store.TxnRetentionContentRemovedParams{Now: now, Ids: []uuid.UUID{id}}); err != nil {
			return err
		}
		if err := EmitMessage(ctx, tx, ws.ID, ws.Slug, id, EventCancelled, "", now); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "transactional.message_cancelled", TargetType: "transactional_message", TargetID: id.String(),
			TargetName: r.TemplateName + " to " + r.Email, Before: map[string]any{"send_at": r.SendAt},
			Metadata: map[string]any{"template": r.TemplateKey, "recipient": r.Email, "summary": "Cancelled " + r.TemplateName + " to " + r.Email + " before its send time"}})
	})
	if err != nil {
		return TransactionalSchedule{}, err
	}
	return TransactionalSchedule{ID: id.String(), Status: "cancelled"}, nil
}

// Reschedule moves a scheduled message to another time, within send_at's
// limit (72 hours from now). Audited.
func (s *Service) Reschedule(ctx context.Context, ws *store.Workspace, id uuid.UUID, in TransactionalRescheduleInput) (TransactionalSchedule, error) {
	now := s.now()
	at := in.SendAt.UTC().Truncate(time.Second)
	switch {
	case !at.After(now.Add(time.Minute)):
		return TransactionalSchedule{}, api.Invalid("body.sendAt", say(ctx, "schedule.future", nil))
	case at.After(now.Add(MaxScheduleAge)):
		return TransactionalSchedule{}, api.Invalid("body.sendAt", say(ctx, "schedule.tooFar", i18n.Args{"hours": int(MaxScheduleAge.Hours())}))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := s.lockScheduled(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if err := q.TxnRescheduleMessage(ctx, store.TxnRescheduleMessageParams{ID: id, SendAt: &at}); err != nil {
			return err
		}
		if err := q.TxnRescheduleSendAt(ctx, store.TxnRescheduleSendAtParams{ID: id, SendAt: &at}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "transactional.message_rescheduled", TargetType: "transactional_message", TargetID: id.String(),
			TargetName: r.TemplateName + " to " + r.Email, Before: map[string]any{"send_at": r.SendAt}, After: map[string]any{"send_at": at},
			Metadata: map[string]any{"template": r.TemplateKey, "recipient": r.Email}})
	})
	if err != nil {
		return TransactionalSchedule{}, err
	}
	return TransactionalSchedule{ID: id.String(), Status: "scheduled", SendAt: &at}, nil
}
