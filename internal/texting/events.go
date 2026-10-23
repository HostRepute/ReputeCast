package texting

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// SmsEvent is one raw provider event beside its normalized one.
type SmsEvent struct {
	ID       string    `json:"id"`
	At       time.Time `json:"at"`
	Provider string    `json:"provider"`
	Adapter  string    `json:"adapter"`
	// Raw is the raw event as the provider sent it, summed up
	// ("MessageStatus=delivered").
	Raw  string `json:"raw"`
	Kind string `json:"kind" enum:"receipt,inbound,test,other"`
	// Normalized is what ReputeCast made of it: queued, sent, delivered,
	// undelivered, failed, opted_out, opted_in, help, yes, reply, or
	// unmatched; pending while it waits for its job.
	Normalized string `json:"normalized"`
	// Reason, Code and the facts the detail is worded from.
	Reason           string   `json:"reason,omitempty"`
	Code             string   `json:"code,omitempty"`
	Temporary        bool     `json:"temporary"`
	Suppressed       bool     `json:"suppressed"`
	Filtered         int64    `json:"filtered"`
	Keyword          string   `json:"keyword,omitempty"`
	Lists            []string `json:"lists"`
	Replied          bool     `json:"replied"`
	ProviderAnswered bool     `json:"providerAnswered"`
	SignatureOK      bool     `json:"signatureOk"`
	MessageRef       string   `json:"messageRef,omitempty"`
	Number           string   `json:"number"`
	Source           string   `json:"source"`
	Error            string   `json:"error,omitempty"`
}

// SmsOptOutSyncRow is a provider's opt-out sync.
type SmsOptOutSyncRow struct {
	Provider   string     `json:"provider"`
	Adapter    string     `json:"adapter"`
	Mode       string     `json:"mode" enum:"two_way,read_only,off"`
	Pulled     int64      `json:"pulled"`
	Pushed     int64      `json:"pushed"`
	LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
	Status     string     `json:"status" enum:"in_sync,failing"`
	Detail     string     `json:"detail,omitempty"`
}

// SmsEventList is the Events tab.
type SmsEventList struct {
	Items      []SmsEvent               `json:"items"`
	Total      int64                    `json:"total"`
	ThisMonth  int64                    `json:"thisMonth"`
	Month      time.Time                `json:"month"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"pageSize"`
	Providers  []SmsSuppressionProvider `json:"providers"`
	OptOutSync []SmsOptOutSyncRow       `json:"optOutSync"`
	Suppressed int64                    `json:"suppressed"`
}

// SmsEventFilter filters the Events tab: receipts, keywords, replies or
// failures, and a provider.
type SmsEventFilter struct {
	Kind     string
	Provider string
	Page     int
}

const eventPage = 25

// Events lists raw provider events beside their normalized ones.
func (s *Service) Events(ctx context.Context, ws *store.Workspace, f SmsEventFilter) (SmsEventList, error) {
	l := i18n.LangFrom(ctx)
	month := monthStart(s.now(), location(ws.Timezone))
	out := SmsEventList{Items: []SmsEvent{}, Month: month, Page: max(f.Page, 1), PageSize: eventPage, Providers: []SmsSuppressionProvider{}, OptOutSync: []SmsOptOutSyncRow{}}
	var prov *uuid.UUID
	if f.Provider != "" {
		id, err := uuid.Parse(f.Provider)
		if err != nil {
			return out, api.Invalid("provider", say(ctx, mPickProvider, nil))
		}
		prov = &id
	}
	where := `r.workspace_id = $1 AND ($2::uuid IS NULL OR r.provider_id = $2)`
	switch f.Kind {
	case "receipts":
		where += ` AND r.kind = 'receipt'`
	case "keywords":
		where += ` AND EXISTS (SELECT 1 FROM sms_inbound i WHERE i.raw_event_id = r.id AND i.kind <> 'reply')`
	case "replies":
		where += ` AND EXISTS (SELECT 1 FROM sms_inbound i WHERE i.raw_event_id = r.id AND i.kind = 'reply')`
	case "failures":
		where += ` AND (NOT r.signature_ok OR r.normalize_error <> '' OR EXISTS (SELECT 1 FROM channel_message_events e WHERE e.raw_event_id = r.id AND e.kind IN ('undelivered','failed')))`
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE r.received_at >= $3) FROM sms_raw_events r WHERE `+where, ws.ID, prov, month).
		Scan(&out.Total, &out.ThisMonth); err != nil {
		return out, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT r.id, r.received_at, coalesce(p.name, ''), coalesce(p.adapter, ''), r.summary, r.kind, r.signature_ok, r.normalized_at,
			r.normalize_error, coalesce(r.events, 0)
		FROM sms_raw_events r LEFT JOIN sms_providers p ON p.id = r.provider_id
		WHERE `+where+` ORDER BY r.received_at DESC, r.id DESC LIMIT $3 OFFSET $4`, ws.ID, prov, eventPage, (out.Page-1)*eventPage)
	if err != nil {
		return out, err
	}
	type rawRow struct {
		e      SmsEvent
		id     uuid.UUID
		normAt *time.Time
	}
	raws := []rawRow{}
	for rows.Next() {
		var r rawRow
		var n int32
		if err := rows.Scan(&r.id, &r.e.At, &r.e.Provider, &r.e.Adapter, &r.e.Raw, &r.e.Kind, &r.e.SignatureOK, &r.normAt, &r.e.Error, &n); err != nil {
			rows.Close()
			return out, err
		}
		r.e.ID = r.id.String()
		r.e.Lists = []string{}
		raws = append(raws, r)
	}
	rows.Close()
	for _, r := range raws {
		e := r.e
		switch {
		case !e.SignatureOK:
			e.Normalized = "rejected"
		case r.normAt == nil:
			e.Normalized = "pending"
		default:
			e.Normalized = "unmatched"
		}
		// A receipt: its message event.
		var kind, code string
		var detail []byte
		var msgID uuid.UUID
		if err := s.Pool.QueryRow(ctx, `SELECT e.kind, e.provider_code, e.detail, e.message_id FROM channel_message_events e WHERE e.raw_event_id = $1
			AND e.kind IN ('queued','sent','delivered','undelivered','failed') ORDER BY e.occurred_at LIMIT 1`, r.id).Scan(&kind, &code, &detail, &msgID); err == nil {
			e.Normalized, e.Code = kind, code
			var d struct {
				Reason     string `json:"reason"`
				Temporary  bool   `json:"temporary"`
				Suppressed bool   `json:"suppressed"`
				Filtered   int64  `json:"filtered"`
			}
			_ = json.Unmarshal(detail, &d)
			e.Reason, e.Temporary, e.Suppressed, e.Filtered = d.Reason, d.Temporary, d.Suppressed, d.Filtered
			if m, err := s.Q.ChanMessage(ctx, msgID); err == nil {
				e.Number = maskPhone(m.Recipient)
				e.MessageRef = shortRef(m.ProviderMessageID)
				e.Source = s.sourceOf(ctx, l, m)
			}
		}
		// An inbound text: what it did.
		var ph, ikind, keyword string
		var effect []byte
		var campaign, automation *uuid.UUID
		if err := s.Pool.QueryRow(ctx, `SELECT phone, kind, keyword, effect, campaign_id, automation_id FROM sms_inbound WHERE raw_event_id = $1 LIMIT 1`, r.id).
			Scan(&ph, &ikind, &keyword, &effect, &campaign, &automation); err == nil {
			e.Normalized = map[string]string{sms.KeywordOptOut: "opted_out", sms.KeywordOptIn: "opted_in", sms.KeywordHelp: "help", sms.KeywordYes: "yes",
				sms.KeywordReply: "reply"}[ikind]
			e.Keyword, e.Number = keyword, maskPhone(ph)
			var eff inboundEffect
			_ = json.Unmarshal(effect, &eff)
			e.Lists, e.Replied, e.ProviderAnswered = eff.Lists, eff.Replied, eff.ProviderAnswered
			if e.Lists == nil {
				e.Lists = []string{}
			}
			e.Source = s.sourceName(ctx, campaign, automation)
		}
		out.Items = append(out.Items, e)
	}
	provs, err := s.Q.SmsProviders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	syncs, _ := s.Q.SmsOptOutSync(ctx, ws.ID)
	for _, p := range provs {
		out.Providers = append(out.Providers, SmsSuppressionProvider{ID: p.ID.String(), Name: p.Name})
		d, _ := sms.Lookup(p.Adapter)
		row := SmsOptOutSyncRow{Provider: p.Name, Adapter: p.Adapter, Status: "in_sync",
			Mode: map[sms.Support]string{sms.Yes: "two_way", sms.Partial: "read_only", sms.No: "off"}[d.Has(sms.CapOptOutSync)]}
		for _, sy := range syncs {
			if sy.ProviderID == p.ID {
				row.Mode, row.Pulled, row.Pushed, row.LastSyncAt, row.Status, row.Detail = sy.Mode, int64(sy.Pulled), int64(sy.Pushed), sy.LastSyncAt, sy.Status,
					answerIn(l, sy.Detail)
			}
		}
		out.OptOutSync = append(out.OptOutSync, row)
	}
	n, err := s.Q.ChanSuppressionCount(ctx, store.ChanSuppressionCountParams{WorkspaceID: ws.ID, Channel: "sms"})
	if err == nil {
		out.Suppressed = n
	}
	return out, nil
}

// shortRef shortens a provider's message id for the table: SM9a41…c207.
func shortRef(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:6] + "…" + id[len(id)-4:]
}

// sourceOf names what a text was: a campaign, a flow, a confirmation (in
// a reader's language).
func (s *Service) sourceOf(ctx context.Context, l i18n.Lang, m store.ChannelMessage) string {
	if n := s.sourceName(ctx, m.CampaignID, m.AutomationID); n != "" {
		return n
	}
	return contextIn(l, contextOf(m.Kind))
}

func (s *Service) sourceName(ctx context.Context, campaign, automation *uuid.UUID) string {
	var name string
	switch {
	case campaign != nil:
		_ = s.Pool.QueryRow(ctx, `SELECT name FROM campaigns WHERE id = $1`, *campaign).Scan(&name)
	case automation != nil:
		_ = s.Pool.QueryRow(ctx, `SELECT name FROM automations WHERE id = $1`, *automation).Scan(&name)
	}
	return name
}
