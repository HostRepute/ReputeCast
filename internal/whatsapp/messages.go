package whatsapp

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// Messages in the message log, and the rules they follow: quiet hours
// (the SMS window, when the workspace keeps WhatsApp to it), the keywords
// and the 24-hour customer service window.

// outgoing is a message to write to the message log.
type outgoing struct {
	id           uuid.UUID
	kind         string // campaign, automation, test, reply
	campaignID   *uuid.UUID
	automationID *uuid.UUID
	step         string
	runID        *uuid.UUID
	contactID    *uuid.UUID
	listID       *uuid.UUID
	recipient    string
	country      string
	timeZone     string
	zoneSource   string
	body         string
	status       string
	reason       string
	after        *time.Time
	// template, language and category for a template; text otherwise.
	template string
	language string
	category string
	params   []string
	price    *float64
}

// insertMessage writes one message: the log's row, then WhatsApp's own.
func insertMessage(ctx context.Context, q *store.Queries, ws *store.Workspace, accountID uuid.UUID, o outgoing, now time.Time) error {
	body := o.body
	if err := q.ChanMessageInsert(ctx, store.ChanMessageInsertParams{ID: o.id, WorkspaceID: ws.ID, Channel: "whatsapp", Kind: o.kind, CampaignID: o.campaignID,
		AutomationID: o.automationID, AutomationStep: o.step, RunID: o.runID, ContactID: o.contactID, ListID: o.listID, Recipient: o.recipient, Country: o.country,
		TimeZone: o.timeZone, ZoneSource: o.zoneSource, Body: &body, Units: 1, Price: o.price, Status: o.status, Reason: o.reason, SendAfter: o.after,
		CreatedAt: now}); err != nil {
		return err
	}
	kind := "template"
	if o.template == "" {
		kind = "text"
	}
	params := o.params
	if params == nil {
		params = []string{}
	}
	return q.WhatsAppMessageInsert(ctx, store.WhatsAppMessageInsertParams{MessageID: o.id, WorkspaceID: ws.ID, AccountID: uuidPtr(accountID), Kind: kind,
		Template: o.template, Language: o.language, Category: o.category, Params: jsonOf(params)})
}

// rules are what WhatsApp sends keep to.
type rules struct {
	// Quiet is whether WhatsApp keeps to the window.
	Quiet    bool
	Window   channels.Window
	Keywords sms.Keywords
}

// rules reads the window and keywords: the SMS rules' once the workspace
// has them, else the defaults with WhatsApp's own quiet-hours choice.
func (s *Service) rules(ctx context.Context, q *store.Queries, ws *store.Workspace) rules {
	r := rules{Quiet: true, Window: channels.DefaultWindow, Keywords: sms.DefaultKeywords}
	if set, err := q.WhatsAppSettings(ctx, ws.ID); err == nil {
		r.Quiet = set.QuietHours
	}
	if row, err := q.PushChannelWindow(ctx, ws.ID); err == nil {
		r.Window = channels.Window{From: int(row.QuietFrom), To: int(row.QuietTo)}.Clamp()
		r.Quiet = row.WhatsappSameWindow
		r.Keywords = sms.Keywords{OptOut: row.OptOutKeywords, OptIn: row.OptInKeywords, Help: row.HelpKeywords}
	}
	return r
}

// WhatsAppSettingsInput is whether WhatsApp keeps to the quiet hours.
type WhatsAppSettingsInput struct {
	QuietHours bool `json:"quietHours"`
}

// SaveSettings changes whether WhatsApp keeps to the quiet hours (logged).
// With SMS rules it's their "WhatsApp keeps to the same window".
func (s *Service) SaveSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WhatsAppSettingsInput) error {
	before := s.rules(ctx, s.Q, ws)
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := s.ensureSettings(ctx, q, ws, now); err != nil {
			return err
		}
		if err := q.WhatsAppSettingsSetQuiet(ctx, store.WhatsAppSettingsSetQuietParams{QuietHours: in.QuietHours, At: now, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		if _, err := q.WhatsAppSetSameWindow(ctx, store.WhatsAppSetSameWindowParams{SameWindow: in.QuietHours, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp_settings.updated", TargetType: "whatsapp_settings", TargetName: "Quiet hours",
			Before: map[string]any{"quiet_hours": before.Quiet}, After: map[string]any{"quiet_hours": in.QuietHours, "window": before.Window.String()}})
	})
}

// windowOpen is the customer service window of a number: open until 24
// hours after its last message.
func (s *Service) windowOpen(ctx context.Context, q *store.Queries, ws *store.Workspace, e164 string, now time.Time) (bool, *time.Time, *store.WhatsappInbound) {
	in, err := q.WhatsAppLastInbound(ctx, store.WhatsAppLastInboundParams{WorkspaceID: ws.ID, Phone: e164})
	if err != nil {
		return false, nil, nil
	}
	until := in.ReceivedAt.Add(24 * time.Hour)
	return until.After(now), &until, &in
}

// requireSend is the send permission check for people: campaigns:send and
// whatsapp:send.
func (s *Service) requireSend(ctx context.Context, ws *store.Workspace, p *authz.Principal) error {
	if canSend(p) {
		return nil
	}
	perm := authz.WhatsAppSend
	if !p.Can(authz.CampaignsSend) {
		perm = authz.CampaignsSend
	}
	return api.Forbidden(perm, s.helpers(ctx, ws, perm))
}

// canSend reports whether p may send WhatsApp campaigns.
func canSend(p *authz.Principal) bool { return p.Can(authz.CampaignsSend) && p.Can(authz.WhatsAppSend) }
