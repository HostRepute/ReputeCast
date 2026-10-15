package transactional

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Webhook events of the "Transactional messages" group. Each carries the
// message's tags and metadata.
const (
	EventSent       = "message.sent"
	EventDelivered  = "message.delivered"
	EventDeferred   = "message.deferred"
	EventBounced    = "message.bounced"
	EventComplained = "message.complained"
	EventRejected   = "message.rejected"
	EventFailed     = "message.failed"
	EventOpened     = "message.opened"
	EventClicked    = "message.clicked"
)

// TransactionalMessageEvent is the data of a message.* webhook event.
type TransactionalMessageEvent struct {
	MessageID         string            `json:"message_id"`
	Template          string            `json:"template"`
	TemplateVersion   *int32            `json:"template_version,omitempty"`
	Category          string            `json:"category,omitempty"`
	To                string            `json:"to"`
	Status            string            `json:"status"`
	Reason            string            `json:"reason,omitempty"`
	Provider          string            `json:"provider,omitempty"`
	ProviderMessageID string            `json:"provider_message_id,omitempty"`
	Tags              []string          `json:"tags"`
	Metadata          map[string]string `json:"metadata"`
	Test              bool              `json:"test,omitempty"`
	At                time.Time         `json:"at"`
}

// emitMessage queues a message.* webhook event in tx, the transaction that
// records what the event reports.
func (s *Service) emitMessage(ctx context.Context, tx pgx.Tx, ws *store.Workspace, id uuid.UUID, event string) error {
	return EmitMessage(ctx, tx, ws.ID, ws.Slug, id, event, "", s.now())
}

// EmitMessage queues a message.* webhook event for a transactional message
// (the sending engine and webhook ingest call it through the hooks).
// reason overrides the message's error ("452 4.2.2 mailbox full").
func EmitMessage(ctx context.Context, tx pgx.Tx, wsID uuid.UUID, slug string, id uuid.UUID, event, reason string, at time.Time) error {
	q := store.New(tx)
	m, err := q.TxnMessageEvent(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	// Opens and clicks fire once per message; the rest once per state.
	if (event == EventOpened || event == EventClicked || event == EventSent || event == EventFailed || event == EventRejected) && slices.Contains(m.EventsEmitted, event) {
		return nil
	}
	data := TransactionalMessageEvent{MessageID: id.String(), Template: m.TemplateKey, TemplateVersion: m.TemplateVersion, Category: m.CategoryName,
		To: m.Email, Status: displayStatus(m.Status, m.RejectReason, m.Test), Provider: m.ProviderName, ProviderMessageID: m.ProviderMessageID,
		Tags: nonNilStrings(m.Tags), Metadata: map[string]string{}, Test: m.Test, At: at.UTC()}
	if m.Source == "smtp" && data.Template == "" {
		data.Template = "raw"
	}
	_ = json.Unmarshal(m.Metadata, &data.Metadata)
	switch {
	case reason != "":
		data.Reason = reason
	case m.RejectReason != "":
		data.Reason = m.RejectReason
	case event == EventFailed || event == EventBounced || event == EventDeferred:
		data.Reason = m.Error
	}
	if _, err := webhooks.Emit(ctx, tx, wsID, slug, event, data); err != nil {
		return err
	}
	return q.TxnMarkEmitted(ctx, store.TxnMarkEmittedParams{MessageID: id, Event: event})
}

// displayStatus is the status the log shows: Test for test-mode messages,
// Rejected for those stopped before sending, else the delivery status.
func displayStatus(status, reject string, test bool) string {
	switch {
	case test:
		return "test"
	case reject != "":
		return "rejected"
	}
	switch status {
	case "held":
		return "queued"
	case "cancelled":
		return "rejected"
	}
	return status
}
