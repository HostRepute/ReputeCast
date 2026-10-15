package transactional

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// SoftBounceLimit soft bounces in SoftBounceWindow put an address on the
// suppression list ("Soft bounces don't suppress the address; three in 7
// days would.").
const (
	SoftBounceLimit  = 3
	SoftBounceWindow = 7 * 24 * time.Hour
)

// Outcome is told what the sending engine's transactional outbox did, in
// the transaction that recorded it: message.sent for messages a provider
// accepted, message.failed for those every provider failed.
func (s *Service) Outcome(ctx context.Context, tx pgx.Tx, wsID uuid.UUID, sent []uuid.UUID, failed map[uuid.UUID]string, at time.Time) error {
	ws, err := store.New(tx).GetWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	for _, id := range sent {
		if err := EmitMessage(ctx, tx, ws.ID, ws.Slug, id, EventSent, "", at); err != nil {
			return err
		}
	}
	for id, reason := range failed {
		if err := EmitMessage(ctx, tx, ws.ID, ws.Slug, id, EventFailed, reason, at); err != nil {
			return err
		}
	}
	return nil
}

// ApplyEvent is the webhook ingest's hook for provider events of
// transactional messages: it emits the message.* webhook event, records a
// soft bounce as the message's final Bounced status, and suppresses an
// address after SoftBounceLimit soft bounces in a week.
func (s *Service) ApplyEvent(ctx context.Context, tx pgx.Tx, p store.Provider, msgID uuid.UUID, stream string, ev providers.Event) error {
	if stream != string(routing.StreamTransactional) {
		return nil
	}
	q := store.New(tx)
	ws, err := q.GetWorkspace(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	at := ev.OccurredAt
	if at.IsZero() {
		at = s.now()
	}
	reason := strings.TrimSpace(ev.OriginalCode + " " + ev.Diagnostic)
	event := ""
	switch ev.Type {
	case providers.EventDelivered:
		event = EventDelivered
	case providers.EventDeferred:
		event = EventDeferred
	case providers.EventHardBounce:
		event = EventBounced
	case providers.EventSoftBounce:
		event = EventBounced
		if err := q.TxnSetBounced(ctx, store.TxnSetBouncedParams{ID: msgID, Error: clip(reason, 500)}); err != nil {
			return err
		}
		if err := s.softBounceRule(ctx, q, p, ev, at); err != nil {
			return err
		}
	case providers.EventComplaint:
		event = EventComplained
	case providers.EventRejected:
		event = EventRejected
	case providers.EventOpen:
		event = EventOpened
	case providers.EventClick:
		event = EventClicked
	}
	if event == "" {
		return nil
	}
	return EmitMessage(ctx, tx, ws.ID, ws.Slug, msgID, event, reason, at)
}

// softBounceRule suppresses an address that soft-bounced SoftBounceLimit
// times in SoftBounceWindow.
func (s *Service) softBounceRule(ctx context.Context, q *store.Queries, p store.Provider, ev providers.Event, at time.Time) error {
	rcpt := strings.ToLower(strings.TrimSpace(ev.Recipient))
	if rcpt == "" {
		return nil
	}
	n, err := q.TxnSoftBounces(ctx, store.TxnSoftBouncesParams{WorkspaceID: p.WorkspaceID, Recipient: rcpt, Since: at.Add(-SoftBounceWindow)})
	if err != nil || n < SoftBounceLimit {
		return err
	}
	pid := p.ID
	_, err = q.TxnSuppressSoftBounces(ctx, store.TxnSuppressSoftBouncesParams{ID: newID(), WorkspaceID: p.WorkspaceID, Email: rcpt, SourceProviderID: &pid,
		OriginalCode: clip(ev.OriginalCode, 100), Detail: "Soft-bounced 3 times in 7 days (last: " + clip(strings.TrimSpace(ev.OriginalCode+" "+ev.Diagnostic), 200) + ")", CreatedAt: at})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// Engagement is the tracking endpoints' hook: the first open or click of a
// tracked transactional message fires message.opened or message.clicked.
func (s *Service) Engagement(ctx context.Context, tx pgx.Tx, msgID uuid.UUID, click bool, at time.Time) error {
	var wsID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT workspace_id FROM messages WHERE id = $1`, msgID).Scan(&wsID); err != nil {
		return err
	}
	ws, err := store.New(tx).GetWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	event := EventOpened
	if click {
		event = EventClicked
	}
	return EmitMessage(ctx, tx, ws.ID, ws.Slug, msgID, event, "", at)
}

// TrackedLink returns a transactional message's tracked link at a position
// (1-based), for the click redirect.
func (s *Service) TrackedLink(ctx context.Context, q *store.Queries, msgID uuid.UUID, pos int) (string, bool) {
	links, err := q.TxnMessageLinks(ctx, msgID)
	if err != nil || pos < 1 || pos > len(links) {
		return "", false
	}
	return links[pos-1], true
}
