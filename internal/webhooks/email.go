package webhooks

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// The "Email" group: campaign and automation email (transactional mail has
// its own message.* group). Review 2026-11 EN-6.
const (
	EmailSent       = "email.sent"
	EmailDelivered  = "email.delivered"
	EmailDeferred   = "email.deferred"
	EmailBounced    = "email.bounced"
	EmailComplained = "email.complained"
	EmailOpened     = "email.opened"
	EmailClicked    = "email.clicked"
)

// EmailEvent is the data of an email.* event.
type EmailEvent struct {
	MessageID         string    `json:"message_id"`
	Stream            string    `json:"stream" enum:"broadcast,automation"`
	To                string    `json:"to"`
	ContactID         string    `json:"contact_id,omitempty"`
	CampaignID        string    `json:"campaign_id,omitempty"`
	AutomationID      string    `json:"automation_id,omitempty"`
	AutomationStep    string    `json:"automation_step,omitempty"`
	Provider          string    `json:"provider,omitempty"`
	ProviderMessageID string    `json:"provider_message_id,omitempty"`
	MailboxProvider   string    `json:"mailbox_provider,omitempty"`
	At                time.Time `json:"at"`
	// Reason is a bounce's or deferral's code and text.
	Reason string `json:"reason,omitempty"`
	// URL is the clicked link (email.clicked).
	URL string `json:"url,omitempty"`
	// Machine marks an open a privacy proxy or scanner made (email.opened).
	Machine bool `json:"machine,omitempty"`
}

// Listens reports whether any endpoint of the workspace takes event.
func Listens(ctx context.Context, q store.DBTX, workspaceID uuid.UUID, event string) (bool, error) {
	return store.New(q).HooksListen(ctx, store.HooksListenParams{WorkspaceID: workspaceID, Event: event})
}

// EmitEmail queues an email.* event about one campaign or automation
// message, in the transaction that records what it reports. Nothing is read
// when no endpoint of the workspace takes the event.
func EmitEmail(ctx context.Context, q store.DBTX, workspaceID, messageID uuid.UUID, event string, at time.Time, extra func(*EmailEvent)) error {
	ok, err := Listens(ctx, q, workspaceID, event)
	if err != nil || !ok {
		return err
	}
	m, err := store.New(q).HooksEmailMessage(ctx, messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if m.Stream != "broadcast" && m.Stream != "automation" {
		return nil
	}
	data := EmailEvent{MessageID: m.ID.String(), Stream: m.Stream, To: m.Email, AutomationStep: m.AutomationNode, Provider: m.Provider,
		ProviderMessageID: m.ProviderMessageID, MailboxProvider: m.MailboxProvider, At: at.UTC()}
	if m.ContactID != nil {
		data.ContactID = m.ContactID.String()
	}
	if m.CampaignID != nil {
		data.CampaignID = m.CampaignID.String()
	}
	if m.AutomationID != nil {
		data.AutomationID = m.AutomationID.String()
	}
	if extra != nil {
		extra(&data)
	}
	_, err = Emit(ctx, q, workspaceID, m.Slug, event, data)
	return err
}
