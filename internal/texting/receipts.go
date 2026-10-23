package texting

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// FilteredLimit: a number filtered by carriers this many times in
// FilteredWindow goes on the suppression list.
const (
	FilteredLimit  = 3
	FilteredWindow = 30 * 24 * time.Hour
)

// errUnmatched is a receipt for a text ReputeCast doesn't know yet: the
// provider can answer before the send call returns its id.
var errUnmatched = errors.New("texting: receipt for an unknown text")

// Normalize reads a stored webhook into receipts and inbound texts and
// applies them. A receipt that arrives before its text's id is known is
// retried (errUnmatched) for a minute, then recorded as unmatched.
func (s *Service) Normalize(ctx context.Context, rawID uuid.UUID) error {
	raw, err := s.Q.SmsRaw(ctx, rawID)
	if err != nil {
		if notFound(err) {
			return nil
		}
		return err
	}
	if raw.NormalizedAt != nil || !raw.SignatureOk || raw.ProviderID == nil {
		return nil
	}
	p, err := s.Q.SmsProviderAny(ctx, *raw.ProviderID)
	if err != nil {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	now := s.now()
	ad, err := s.open(p)
	if err != nil {
		return err
	}
	events, err := ad.ParseWebhook(ctx, raw.ContentType, raw.Body)
	if err != nil {
		return s.Q.SmsRawNormalized(ctx, store.SmsRawNormalizedParams{At: &now, NormalizeError: err.Error(), ID: raw.ID})
	}
	kind, summary := "other", ""
	applied := int32(0)
	for _, e := range events {
		if summary == "" {
			summary = e.Summary
		}
		if e.OccurredAt.IsZero() {
			e.OccurredAt = raw.ReceivedAt
		}
		switch e.Kind {
		case sms.EventInbound:
			kind = "inbound"
			if err := s.applyInbound(ctx, &ws, p, raw.ID, e); err != nil {
				return err
			}
		default:
			kind = "receipt"
			err := s.applyReceipt(ctx, &ws, p, raw.ID, e)
			if errors.Is(err, errUnmatched) {
				if now.Sub(raw.ReceivedAt) < time.Minute {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
		}
		applied++
	}
	return s.Q.SmsRawNormalized(ctx, store.SmsRawNormalizedParams{At: &now, Events: applied, Summary: summary, Kind: kind, ID: raw.ID})
}

// statusOf maps a receipt to a message status.
func statusOf(k sms.EventKind) string {
	switch k {
	case sms.EventQueued:
		return "sending"
	case sms.EventSent:
		return "sent"
	case sms.EventDelivered:
		return "delivered"
	case sms.EventUndelivered:
		return "undelivered"
	}
	return "failed"
}

// applyReceipt records a delivery receipt on its text: the status, the
// provider's code, what the failure means for the number (suppressed, its
// line type, one more carrier filter) and the outgoing webhook event.
func (s *Service) applyReceipt(ctx context.Context, ws *store.Workspace, p store.SmsProvider, rawID uuid.UUID, e sms.Event) error {
	var m store.ChannelMessage
	var err error
	switch {
	case e.MessageID != nil:
		m, err = s.Q.ChanMessage(ctx, *e.MessageID)
	case e.ProviderMessageID != "":
		m, err = s.Q.ChanMessageByProviderID(ctx, store.ChanMessageByProviderIDParams{ProviderID: &p.ID, ProviderMessageID: e.ProviderMessageID})
	default:
		return errUnmatched
	}
	if err != nil {
		if notFound(err) {
			return errUnmatched
		}
		return err
	}
	if m.WorkspaceID != p.WorkspaceID {
		return errUnmatched
	}
	now := s.now()
	_ = s.Q.SmsProviderTouchReceipt(ctx, store.SmsProviderTouchReceiptParams{At: now, ID: p.ID})
	status := statusOf(e.Kind)
	reason := ""
	if status == "undelivered" || status == "failed" {
		reason = string(e.Reason)
		if reason == "" {
			reason = string(sms.ReasonOther)
		}
	}
	detail := map[string]any{"summary": e.Summary}
	if reason != "" {
		detail["reason"] = reason
		detail["temporary"] = e.Reason == sms.ReasonUnreachable
	}
	suppress := false
	if e.Reason == sms.ReasonFiltered && m.Recipient != "" {
		var n int64
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE workspace_id = $1 AND recipient = $2 AND reason = 'filtered'
			AND coalesce(failed_at, updated_at) > $3 AND id <> $4`, m.WorkspaceID, m.Recipient, now.Add(-FilteredWindow), m.ID).Scan(&n)
		detail["filtered"] = n + 1
		detail["filteredLimit"] = FilteredLimit
		suppress = n+1 >= FilteredLimit
	}
	if e.Reason.Suppresses() {
		suppress = true
	}
	detail["suppressed"] = suppress
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if status != "sending" {
			if err := q.ChanMessageReceipt(ctx, store.ChanMessageReceiptParams{Status: status, Reason: reason, ProviderCode: e.Code, At: e.OccurredAt, ID: m.ID}); err != nil {
				return err
			}
		}
		if e.Price != nil && *e.Price > 0 && m.Price == nil {
			if _, err := tx.Exec(ctx, `UPDATE channel_messages SET price = $2 / greatest(units, 1), cost = $2 WHERE id = $1`, m.ID, round4(*e.Price)); err != nil {
				return err
			}
		}
		if err := q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: m.WorkspaceID, MessageID: m.ID,
			Kind: string(e.Kind), ProviderCode: e.Code, Detail: jsonOf(detail), RawEventID: &rawID, OccurredAt: e.OccurredAt}); err != nil {
			return err
		}
		switch status {
		case "delivered":
			_, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "sms.delivered", s.hookData(m, map[string]any{"delivered_at": e.OccurredAt}))
			return err
		case "undelivered", "failed":
			_, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "sms.failed", s.hookData(m, map[string]any{"reason": reason, "provider_code": e.Code,
				"temporary": e.Reason == sms.ReasonUnreachable}))
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if suppress && m.Recipient != "" {
		r := e.Reason
		if r == sms.ReasonFiltered {
			r = "filtered"
		}
		s.suppressFromFailure(ctx, ws, p, m.Recipient, r, codeLabel(p, e.Code), m.CampaignID, m.AutomationID, contextOf(m.Kind))
	}
	if (status == "undelivered" || status == "failed") && m.CampaignID != nil && m.Kind == "campaign" {
		s.checkAutoPause(ctx, ws, *m.CampaignID)
	}
	return nil
}

// codeLabel is the original code as the suppression list shows it:
// "Twilio 30005".
func codeLabel(p store.SmsProvider, code string) string {
	if code == "" {
		return p.Name
	}
	return p.Name + " " + code
}

// hookData is an SMS webhook event's data, the number in full: endpoints
// are the workspace's own systems.
func (s *Service) hookData(m store.ChannelMessage, extra map[string]any) map[string]any {
	d := map[string]any{"message_id": m.ID.String(), "phone": m.Recipient, "country": m.Country, "kind": m.Kind, "sender": m.Sender}
	if m.ContactID != nil {
		d["contact_id"] = m.ContactID.String()
	}
	if m.CampaignID != nil {
		d["campaign_id"] = m.CampaignID.String()
	}
	if m.AutomationID != nil {
		d["automation_id"] = m.AutomationID.String()
	}
	for k, v := range extra {
		d[k] = v
	}
	return d
}

func init() {
	webhooks.Events = append(webhooks.Events,
		webhooks.WebhookEventInfo{Name: "sms.delivered", Group: "SMS", Description: "A carrier delivered a text.", DescriptionText: msgs.M("webhook.sms.delivered").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "sms.failed", Group: "SMS", Description: "A text failed, with the reason and the provider's own code.", DescriptionText: msgs.M("webhook.sms.failed").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "sms.clicked", Group: "SMS", Description: "A person clicked a text's short link (link previews carry machine: true).", DescriptionText: msgs.M("webhook.sms.clicked").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "sms.opted_out", Group: "SMS", Description: "A number texted STOP: SMS consent withdrawn on every list.", DescriptionText: msgs.M("webhook.sms.opted_out").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "sms.opted_in", Group: "SMS", Description: "A number texted START: the consents STOP removed came back.", DescriptionText: msgs.M("webhook.sms.opted_in").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "sms.replied", Group: "SMS", Description: "A reply that isn't a keyword, with its text.", DescriptionText: msgs.M("webhook.sms.replied").Keep(nil)},
	)
}
