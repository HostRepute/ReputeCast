package earn

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// The payments event log's groups of event types.
var eventGroups = map[string][]string{
	"payments": {"invoice.paid", "checkout.session.completed", "invoice.upcoming"},
	"failures": {"invoice.payment_failed"},
	"refunds":  {"charge.refunded", "charge.dispute.created"},
	"changes":  {"customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted", "customer.updated"},
	"trials":   {"customer.subscription.trial_will_end"},
}

var eventRanges = map[string]time.Duration{"24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour, "30d": 30 * 24 * time.Hour, "90d": RetentionDays * 24 * time.Hour}

// EarnEventFilter narrows the payments event log.
type EarnEventFilter struct {
	Group    string `query:"group" enum:"all,payments,failures,refunds,changes,trials" default:"all" doc:"A group of event types"`
	Range    string `query:"range" enum:"24h,7d,30d,90d" default:"7d" doc:"How far back (raw events are kept 90 days)"`
	Q        string `query:"q" maxLength:"200" doc:"Search the event type, the event id, the paid member or the raw event"`
	Test     bool   `query:"test" doc:"Include test-mode events"`
	Page     int    `query:"page" minimum:"1" default:"1"`
	PageSize int    `query:"pageSize" minimum:"1" maximum:"100" default:"8"`
}

// EarnEvent is one line of the payments event log: the provider's event
// next to what ReputeCast did with it.
type EarnEvent struct {
	ID         string    `json:"id"`
	Mode       string    `json:"mode" enum:"live,test"`
	EventID    string    `json:"eventId"`
	Type       string    `json:"type"`
	Source     string    `json:"source" enum:"webhook,gap_check,history,manual"`
	OccurredAt time.Time `json:"occurredAt"`
	ReceivedAt time.Time `json:"receivedAt"`
	// Status: applied, ignored (nothing to do), failed (retried), rejected
	// (a bad signature, never applied), pending.
	Status string `json:"status" enum:"pending,applied,ignored,failed,rejected"`
	// Outcome names what was done ("renewed", "past_due"); Detail has its
	// facts (tier, dates, the member email sent).
	Outcome     string         `json:"outcome"`
	Detail      map[string]any `json:"detail"`
	AmountCents *int64         `json:"amountCents,omitempty"`
	Currency    string         `json:"currency,omitempty"`
	MemberID    string         `json:"memberId,omitempty"`
	ContactID   string         `json:"contactId,omitempty"`
	MemberName  string         `json:"memberName,omitempty"`
	MemberEmail string         `json:"memberEmail,omitempty"`
	// EventURL opens the event in the provider's dashboard.
	EventURL string `json:"eventUrl,omitempty"`
}

// EarnEventLog is a page of the payments event log.
type EarnEventLog struct {
	Items    []EarnEvent `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Range    string      `json:"range"`
	// Types are the event types the log holds, for the search's hints.
	Types []string `json:"types"`
}

// EventLog is a page of the payments event log, newest first. Readers'
// own changes in checkout or the portal show here, never in the audit log.
func (s *Service) EventLog(ctx context.Context, ws *store.Workspace, p *authz.Principal, f EarnEventFilter) (EarnEventLog, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 8
	}
	window, ok := eventRanges[f.Range]
	if !ok {
		f.Range, window = "7d", eventRanges["7d"]
	}
	types := eventGroups[f.Group]
	if types == nil {
		types = []string{}
	}
	since := s.now().Add(-window)
	params := store.PayEventLogParams{WorkspaceID: ws.ID, Since: since, IncludeTest: f.Test, Types: types, Q: f.Q,
		Off: int32((f.Page - 1) * f.PageSize), Lim: int32(f.PageSize)}
	rows, err := s.Q.PayEventLog(ctx, params)
	if err != nil {
		return EarnEventLog{}, err
	}
	total, err := s.Q.PayEventLogCount(ctx, store.PayEventLogCountParams{WorkspaceID: ws.ID, Since: since, IncludeTest: f.Test, Types: types, Q: f.Q})
	if err != nil {
		return EarnEventLog{}, err
	}
	seen, err := s.Q.PayEventTypesSeen(ctx, ws.ID)
	if err != nil {
		return EarnEventLog{}, err
	}
	if seen == nil {
		seen = []string{}
	}
	hide := !p.Can(authz.SubscribersView)
	short := map[string]string{}
	if tiers, err := s.Q.EarnTiers(ctx, ws.ID); err == nil {
		names := s.listNamer(ctx, ws.ID)
		for _, t := range tiers {
			short[t.Name] = shortTier(t.Name, names(&t.ID))
		}
	}
	out := EarnEventLog{Items: []EarnEvent{}, Total: total, Page: f.Page, PageSize: f.PageSize, Range: f.Range, Types: seen}
	for _, r := range rows {
		e := EarnEvent{ID: r.ID.String(), Mode: r.Mode, EventID: r.ProviderEventID, Type: r.Type, Source: r.Source, OccurredAt: r.OccurredAt,
			ReceivedAt: r.ReceivedAt, Status: r.Status, Outcome: r.Outcome, Detail: map[string]any{}, AmountCents: r.AmountCents, Currency: r.Currency,
			MemberName: r.MemberName, MemberEmail: r.MemberEmail}
		if len(r.OutcomeDetail) > 0 {
			_ = json.Unmarshal(r.OutcomeDetail, &e.Detail)
		}
		if t, ok := e.Detail["tier"].(string); ok && short[t] != "" {
			e.Detail["tier"] = short[t]
		}
		if r.SubscriptionID != nil {
			e.MemberID = r.SubscriptionID.String()
		}
		if r.ContactID != nil {
			e.ContactID = r.ContactID.String()
		}
		if hide {
			e.MemberEmail = ""
		}
		if r.ProviderEventID != "" && r.SignatureOk {
			e.EventURL = dashboard(DefaultProvider, payments.Mode(r.Mode), "events", r.ProviderEventID)
		}
		out.Items = append(out.Items, e)
	}
	return out, nil
}
