package texting

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// SmsSuppression is a number on the SMS suppression list, partly masked.
type SmsSuppression struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Reason string `json:"reason" enum:"opted_out,landline,not_in_service,invalid,filtered,provider_list,manual"`
	// Code is the provider's own code ("Twilio 30005") or the word ("STOP").
	Code   string    `json:"code"`
	Source string    `json:"source"`
	At     time.Time `json:"at"`
	// Campaign or Automation is what the number was sent; Context what
	// else it was ("Opt-in confirmation").
	Campaign   string `json:"campaign,omitempty"`
	Automation string `json:"automation,omitempty"`
	Context    string `json:"context,omitempty"`
	// Removable is false for numbers that opted out by text: only the
	// person can come back, by texting START.
	Removable bool `json:"removable"`
}

// SmsSuppressionProvider is a provider to filter by.
type SmsSuppressionProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SmsSuppressionList is Audience › Suppressions filtered to SMS.
type SmsSuppressionList struct {
	Items     []SmsSuppression         `json:"items"`
	Total     int64                    `json:"total"`
	Matching  int64                    `json:"matching"`
	Counts    map[string]int64         `json:"counts"`
	Providers []SmsSuppressionProvider `json:"providers"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"pageSize"`
	CanEdit   bool                     `json:"canEdit"`
	Reason    string                   `json:"reason,omitempty"`
}

// SmsSuppressionFilter filters the list.
type SmsSuppressionFilter struct {
	Reason   string
	Provider string
	Query    string
	Page     int
}

const suppressionPage = 25

// Suppressions lists the SMS suppression list.
func (s *Service) Suppressions(ctx context.Context, ws *store.Workspace, p *authz.Principal, f SmsSuppressionFilter) (SmsSuppressionList, error) {
	out := SmsSuppressionList{Items: []SmsSuppression{}, Counts: map[string]int64{}, Providers: []SmsSuppressionProvider{}, Page: max(f.Page, 1),
		PageSize: suppressionPage, CanEdit: p.Can(authz.SubscribersEdit)}
	var prov *uuid.UUID
	if f.Provider != "" {
		id, err := uuid.Parse(f.Provider)
		if err != nil {
			return out, api.Invalid("provider", "Pick an SMS provider of this workspace.")
		}
		prov = &id
	}
	q := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, f.Query)
	rows, err := s.Q.ChanSuppressions(ctx, store.ChanSuppressionsParams{WorkspaceID: ws.ID, Channel: "sms", Reason: f.Reason, ProviderID: prov, Q: q,
		Lim: suppressionPage, Off: int32((out.Page - 1) * suppressionPage)})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		it := SmsSuppression{ID: r.ID.String(), Number: maskPhone(r.Phone), Reason: r.Reason, Code: r.OriginalCode, Source: r.Source, At: r.CreatedAt,
			Campaign: r.CampaignName, Automation: r.AutomationName, Context: r.Context, Removable: r.Reason != "opted_out"}
		if r.ProviderName != "" && it.Source == "" {
			it.Source = r.ProviderName
		}
		out.Items = append(out.Items, it)
	}
	if out.Matching, err = s.Q.ChanSuppressionCount(ctx, store.ChanSuppressionCountParams{WorkspaceID: ws.ID, Channel: "sms", Reason: f.Reason, ProviderID: prov, Q: q}); err != nil {
		return out, err
	}
	counts, err := s.Q.ChanSuppressionCounts(ctx, store.ChanSuppressionCountsParams{WorkspaceID: ws.ID, Channel: "sms"})
	if err != nil {
		return out, err
	}
	for _, c := range counts {
		out.Counts[c.Reason] = c.N
		out.Total += c.N
	}
	provs, err := s.Q.SmsProviders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, pr := range provs {
		out.Providers = append(out.Providers, SmsSuppressionProvider{ID: pr.ID.String(), Name: pr.Name})
	}
	if !out.CanEdit {
		out.Reason = authz.DeniedMessage(authz.SubscribersEdit, nil)
	}
	return out, nil
}

// SmsSuppressionRemove takes a number off the list.
type SmsSuppressionRemove struct {
	Reason string `json:"reason" minLength:"3" maxLength:"500"`
}

// RemoveSuppression takes a number off the SMS suppression list, with a
// reason. A number that opted out by text stays: only the person can come
// back, by texting START.
func (s *Service) RemoveSuppression(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsSuppressionRemove) error {
	sup, err := s.Q.ChanSuppression(ctx, store.ChanSuppressionParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Suppressed number")
	}
	if sup.Reason == "opted_out" {
		return api.Err(http.StatusConflict, "opted_out_by_text", "This number opted out by text. Only the person can come back, by texting START.")
	}
	reason := strings.TrimSpace(in.Reason)
	if len(reason) < 3 {
		return api.Invalid("body.reason", "Say why the number can get texts again.")
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.ChanSuppressionRemove(ctx, store.ChanSuppressionRemoveParams{At: &now, RemovedBy: uuidPtr(p.UserID), RemovedReason: reason, ID: sup.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_suppression.removed", TargetType: "phone", TargetID: sup.ID.String(),
			TargetName: maskPhone(sup.Phone), Reason: reason, Before: map[string]any{"reason": sup.Reason, "code": sup.OriginalCode}, After: map[string]any{"suppressed": false}})
	})
	if err != nil {
		return err
	}
	s.syncRemove(ctx, ws, sup.Phone)
	return nil
}

// SmsSuppressionAdd puts numbers on the list by hand.
type SmsSuppressionAdd struct {
	Numbers []string `json:"numbers" minItems:"1" maxItems:"100"`
	Note    string   `json:"note,omitempty" maxLength:"500"`
}

// AddSuppressions puts numbers on the list (reason Manual).
func (s *Service) AddSuppressions(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsSuppressionAdd) (int, error) {
	phones := []string{}
	for i, raw := range in.Numbers {
		n, err := phone.Parse(raw, "")
		if err != nil {
			return 0, api.Invalid("body.numbers["+itoa(i)+"]", "“"+raw+"” isn't a phone number in international format, like +1 312 555 0123.")
		}
		phones = append(phones, n.E164)
	}
	now := s.now()
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		for _, ph := range phones {
			if err := q.ChanSuppressionInsert(ctx, store.ChanSuppressionInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Channel: "sms", Phone: ph,
				Reason: "manual", OriginalCode: strings.TrimSpace(in.Note), Source: p.Name, CreatedBy: uuidPtr(p.UserID), CreatedAt: now}); err != nil {
				return err
			}
		}
		masked := make([]string, 0, len(phones))
		for _, ph := range phones {
			masked = append(masked, maskPhone(ph))
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_suppression.added", TargetType: "phone", TargetName: strings.Join(masked, ", "),
			After: map[string]any{"numbers": masked, "reason": "manual", "note": in.Note}})
	})
	if err != nil {
		return 0, err
	}
	for _, ph := range phones {
		s.syncAdd(ctx, ws, ph)
	}
	return len(phones), nil
}

func itoa(i int) string { return strconv.Itoa(i) }

// suppressFromFailure puts a number on the list after a provider said it
// can't get texts (not in service, landline, invalid, the provider's own
// opt-out list), and notes the line type.
func (s *Service) suppressFromFailure(ctx context.Context, ws *store.Workspace, prov store.SmsProvider, ph string, reason sms.Reason, code string,
	campaignID, automationID *uuid.UUID, contextText string) {
	r := string(reason)
	switch reason {
	case sms.ReasonOptedOut:
		r = "provider_list"
	case sms.ReasonLandline:
		_ = s.Q.ChanSetLineType(ctx, store.ChanSetLineTypeParams{PhoneLineType: "landline", PhoneLineSource: "provider", WorkspaceID: ws.ID, Phone: ph})
	}
	_ = s.Q.ChanSuppressionInsert(ctx, store.ChanSuppressionInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Channel: "sms", Phone: ph, Reason: r,
		OriginalCode: code, Source: prov.Name, SourceProviderID: &prov.ID, CampaignID: campaignID, AutomationID: automationID, Context: contextText,
		CreatedAt: s.now()})
}

// syncAdd and syncRemove keep providers' opt-out lists in line, where the
// adapter syncs both ways.
func (s *Service) syncAdd(ctx context.Context, ws *store.Workspace, ph string) {
	provs, err := s.Q.SmsProviders(ctx, ws.ID)
	if err != nil {
		return
	}
	for _, pr := range provs {
		p := providerOf(pr)
		if ad, err := s.open(p); err == nil {
			if sy, ok := ad.(sms.OptOutSyncer); ok {
				_ = sy.AddOptOuts(ctx, []string{ph})
			}
		}
	}
}

func (s *Service) syncRemove(ctx context.Context, ws *store.Workspace, ph string) {
	provs, err := s.Q.SmsProviders(ctx, ws.ID)
	if err != nil {
		return
	}
	for _, pr := range provs {
		p := providerOf(pr)
		if ad, err := s.open(p); err == nil {
			if sy, ok := ad.(sms.OptOutSyncer); ok {
				_ = sy.RemoveOptOut(ctx, ph)
			}
		}
	}
}
