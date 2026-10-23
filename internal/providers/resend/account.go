package resend

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
)

var (
	_ providers.Verifier           = (*Provider)(nil)
	_ providers.QuotaReporter      = (*Provider)(nil)
	_ providers.DomainManager      = (*Provider)(nil)
	_ providers.WebhookRegistrar   = (*Provider)(nil)
	_ providers.WebhookUnregistrar = (*Provider)(nil)
	_ providers.WebhookInspector   = (*Provider)(nil)
	_ providers.WebhookVerifier    = (*Provider)(nil)
	_ providers.SuppressionSyncer  = (*Provider)(nil)
	_ providers.TrackingManager    = (*Provider)(nil)
	_ providers.EventBackfiller    = (*Provider)(nil)
	_ providers.ReputationReporter = (*Provider)(nil)
	_ providers.StatusPage         = (*Provider)(nil)
)

// sendingOnly reports the error a Sending access key gets on any endpoint
// but sending: the key is valid, it just can't read the account.
func sendingOnly(err error) bool {
	var ce *providers.CallError
	return errors.As(err, &ce) && ce.Status == http.StatusUnauthorized && ce.Code == "restricted_api_key"
}

type domainSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Region string `json:"region"`
}

func (p *Provider) listDomains(ctx context.Context) ([]domainSummary, http.Header, error) {
	res, err := p.api.Exec(ctx, apiclient.Request{Op: "GET /domains", Method: http.MethodGet, Path: "/domains"})
	if err != nil {
		return nil, nil, err
	}
	var list struct {
		Data []domainSummary `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &list); err != nil {
		return nil, nil, &providers.CallError{Op: "GET /domains", Status: res.Status, Temporary: true, Message: "unexpected response: " + err.Error()}
	}
	return list.Data, res.Header, nil
}

// Verify implements providers.Verifier.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) { return p.Quota(ctx) }

// Quota implements providers.QuotaReporter: the emails sent this month
// (Resend's x-resend-monthly-quota header) against the plan allowance
// entered at connect time. Until a domain is verified Resend only sends
// test mail to the account owner, which shows as sandbox.
func (p *Provider) Quota(ctx context.Context) (providers.Quota, error) {
	domains, header, err := p.listDomains(ctx)
	if sendingOnly(err) {
		var q providers.Quota
		q.Account, q.AccountText = say(mAccountSending, nil)
		return q, nil
	}
	if err != nil {
		return providers.Quota{}, err
	}
	verified := 0
	for _, d := range domains {
		if d.Status == "verified" {
			verified++
		}
	}
	q := providers.Quota{Sandbox: verified == 0}
	// "2 verified domains · 50,000 per month plan"
	account := i18n.Args{"verified": verified, "plan": "no"}
	if used, err := strconv.ParseInt(strings.TrimSpace(header.Get("X-Resend-Monthly-Quota")), 10, 64); err == nil {
		q.Kind, q.Used, q.Max = "monthly_plan", used, int64(p.s.MonthlyLimit)
		if p.s.MonthlyLimit > 0 {
			account["plan"], account["limit"] = "yes", int64(p.s.MonthlyLimit)
		}
	}
	q.Account, q.AccountText = say(mAccount, account)
	return q, nil
}

type domainRecord struct {
	Record   string `json:"record"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	Status   string `json:"status"`
	Priority *int   `json:"priority"`
}

type domainInfo struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Status            string         `json:"status"`
	OpenTracking      bool           `json:"open_tracking"`
	ClickTracking     bool           `json:"click_tracking"`
	TrackingSubdomain string         `json:"tracking_subdomain"`
	Records           []domainRecord `json:"records"`
}

// AddDomain implements providers.DomainManager: POST /domains in the
// configured region. A domain already in the account is read instead.
func (p *Provider) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	body := map[string]any{"name": domain}
	if p.s.Region != "" {
		body["region"] = p.s.Region
	}
	var d domainInfo
	err := p.api.Do(ctx, "POST /domains", http.MethodPost, "/domains", body, &d)
	if err == nil {
		return records(d, domain), nil
	}
	err = classify(err)
	var ce *providers.CallError
	if errors.As(err, &ce) && !ce.Auth && !ce.Temporary && strings.Contains(strings.ToLower(ce.Message), "already") {
		return p.DomainStatus(ctx, domain)
	}
	return nil, err
}

// DomainStatus implements providers.DomainManager. Resend checks DNS when
// asked: a domain whose check never ran or failed is sent for verification
// again, so the next status shows the new result.
func (p *Provider) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	domains, _, err := p.listDomains(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range domains {
		if !strings.EqualFold(s.Name, domain) {
			continue
		}
		var d domainInfo
		if err := p.api.Do(ctx, "GET /domains/{id}", http.MethodGet, apiclient.Path("/domains/%s", s.ID), nil, &d); err != nil {
			return nil, err
		}
		switch d.Status {
		case "not_started", "failed", "temporary_failure", "partially_failed":
			_ = p.api.Do(ctx, "POST /domains/{id}/verify", http.MethodPost, apiclient.Path("/domains/%s/verify", s.ID), struct{}{}, nil)
		}
		return records(d, domain), nil
	}
	ce := &providers.CallError{Op: "GET /domains", Status: http.StatusNotFound}
	ce.Message, ce.Text = say(mDomainMissing, i18n.Args{"domain": domain})
	return nil, ce
}

func records(d domainInfo, domain string) []providers.DNSRecord {
	out := []providers.DNSRecord{}
	for _, r := range d.Records {
		rec := providers.DNSRecord{Type: strings.ToUpper(r.Type), Name: absolute(r.Name, domain), Value: strings.Trim(strings.TrimSpace(r.Value), `"`), Status: recordStatus(r.Status)}
		switch {
		case r.Record == "DKIM":
			rec.Purpose = "dkim"
		case r.Record == "SPF" && rec.Type == "MX":
			rec.Purpose = "return_path"
			prio := 10
			if r.Priority != nil {
				prio = *r.Priority
			}
			rec.Value = strconv.Itoa(prio) + " " + rec.Value
		case r.Record == "SPF":
			rec.Purpose = "spf"
		default:
			continue // tracking and receiving records: not needed to send
		}
		out = append(out, rec)
	}
	return out
}

// absolute turns Resend's relative record names ("resend._domainkey",
// "send") into full names.
func absolute(name, domain string) string {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".")
	switch {
	case name == "" || name == "@":
		return domain
	case strings.EqualFold(name, domain) || strings.HasSuffix(strings.ToLower(name), "."+strings.ToLower(domain)):
		return name
	}
	return name + "." + domain
}

func recordStatus(s string) string {
	switch s {
	case "verified":
		return "verified"
	case "failed", "temporary_failure":
		return "wrong_value"
	}
	return "pending"
}

type webhook struct {
	ID       string   `json:"id"`
	Endpoint string   `json:"endpoint"`
	Status   string   `json:"status"`
	Events   []string `json:"events"`
}

// eachWebhook calls fn with the team's webhooks, page by page, until fn
// returns false.
func (p *Provider) eachWebhook(ctx context.Context, fn func(webhook) bool) error {
	after := ""
	for range 20 {
		path := "/webhooks?limit=100"
		if after != "" {
			path += "&after=" + url.QueryEscape(after)
		}
		var list struct {
			HasMore bool      `json:"has_more"`
			Data    []webhook `json:"data"`
		}
		if err := p.api.Do(ctx, "GET /webhooks", http.MethodGet, path, nil, &list); err != nil {
			return err
		}
		for _, h := range list.Data {
			if !fn(h) {
				return nil
			}
		}
		if !list.HasMore || len(list.Data) == 0 {
			return nil
		}
		after = list.Data[len(list.Data)-1].ID
	}
	return nil
}

// findWebhook returns the first webhook whose endpoint matches, or nil.
func (p *Provider) findWebhook(ctx context.Context, match func(endpoint string) bool) (*webhook, error) {
	var found *webhook
	err := p.eachWebhook(ctx, func(h webhook) bool {
		if match(h.Endpoint) {
			found = &h
		}
		return found == nil
	})
	return found, err
}

func sameURL(a, b string) bool { return strings.TrimRight(a, "/") == strings.TrimRight(b, "/") }

// RegisterWebhook implements providers.WebhookRegistrar: one webhook for
// webhookEvents. An existing webhook for the URL is updated and enabled. Its
// signing secret arrives with the answer, or is read back from the API when
// events arrive.
func (p *Provider) RegisterWebhook(ctx context.Context, hookURL string) error {
	existing, err := p.findWebhook(ctx, func(e string) bool { return sameURL(e, hookURL) })
	if err != nil {
		return err
	}
	if existing != nil {
		body := map[string]any{"endpoint": hookURL, "events": webhookEvents, "status": "enabled"}
		return p.api.Do(ctx, "PATCH /webhooks/{id}", http.MethodPatch, apiclient.Path("/webhooks/%s", existing.ID), body, nil)
	}
	body := map[string]any{"endpoint": hookURL, "events": webhookEvents}
	var created struct {
		SigningSecret string `json:"signing_secret"`
	}
	if err := p.api.Do(ctx, "POST /webhooks", http.MethodPost, "/webhooks", body, &created); err != nil {
		return err
	}
	secrets.Set(p.secretKey(pathOf(hookURL)), created.SigningSecret)
	return nil
}

// UnregisterWebhook implements providers.WebhookUnregistrar: DELETE
// /webhooks/{id} for every webhook pointing at the URL
// (https://resend.com/docs/api-reference/webhooks/delete-webhook). A
// webhook already gone (404) counts as removed.
func (p *Provider) UnregisterWebhook(ctx context.Context, hookURL string) error {
	// Listed in full before any delete, which would move the list's cursor.
	var ids []string
	err := p.eachWebhook(ctx, func(h webhook) bool {
		if sameURL(h.Endpoint, hookURL) {
			ids = append(ids, h.ID)
		}
		return true
	})
	if err != nil {
		return err
	}
	for _, id := range ids {
		err := p.api.Do(ctx, "DELETE /webhooks/{id}", http.MethodDelete, apiclient.Path("/webhooks/%s", id), nil, nil)
		var ce *providers.CallError
		if errors.As(err, &ce) && ce.Status == http.StatusNotFound {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// InspectWebhook implements providers.WebhookInspector, with the delivery
// state of the webhook's latest 100 events from Resend's log.
func (p *Provider) InspectWebhook(ctx context.Context, hookURL string) (providers.WebhookStatus, error) {
	st := providers.WebhookStatus{Queued: -1, Sent: -1}
	hook, err := p.findWebhook(ctx, func(e string) bool { return sameURL(e, hookURL) })
	if err != nil || hook == nil {
		return st, err
	}
	st.Registered, st.ID, st.URL = hook.Status != "disabled", hook.ID, hook.Endpoint
	var log struct {
		Data []struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := p.api.Do(ctx, "GET /webhooks/{id}/events", http.MethodGet, apiclient.Path("/webhooks/%s/events", hook.ID)+"?limit=100", nil, &log); err == nil {
		st.Queued, st.Sent = 0, 0
		for _, e := range log.Data {
			switch e.Status {
			case "success":
				st.Sent++
			case "pending", "attempting":
				st.Queued++
			}
		}
	}
	return st, nil
}

// ListSuppressions implements providers.SuppressionSyncer with Resend's
// account suppression list (newest first).
func (p *Provider) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	out := []providers.Suppression{}
	after := ""
	for range 1000 {
		path := "/suppressions?limit=100"
		if after != "" {
			path += "&after=" + url.QueryEscape(after)
		}
		var page struct {
			HasMore bool `json:"has_more"`
			Data    []struct {
				ID        string `json:"id"`
				Email     string `json:"email"`
				Origin    string `json:"origin"`
				CreatedAt string `json:"created_at"`
			} `json:"data"`
		}
		if err := p.api.Do(ctx, "GET /suppressions", http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		older := 0
		for _, s := range page.Data {
			created := parseTime(s.CreatedAt)
			if !since.IsZero() && !created.IsZero() && created.Before(since) {
				older++
				continue
			}
			reason := "manual"
			switch s.Origin {
			case "bounce":
				reason = "hard_bounce"
			case "complaint":
				reason = "complaint"
			}
			out = append(out, providers.Suppression{Email: strings.ToLower(s.Email), Reason: reason, Code: s.Origin, CreatedAt: created})
		}
		if !page.HasMore || len(page.Data) == 0 || older == len(page.Data) {
			break
		}
		after = page.Data[len(page.Data)-1].ID
	}
	return out, nil
}

// AddSuppressions implements providers.SuppressionSyncer (100 per call).
func (p *Provider) AddSuppressions(ctx context.Context, emails []string) error {
	for i := 0; i < len(emails); i += 100 {
		chunk := emails[i:min(i+100, len(emails))]
		if err := p.api.Do(ctx, "POST /suppressions/batch/add", http.MethodPost, "/suppressions/batch/add", map[string]any{"emails": chunk}, nil); err != nil {
			return classify(err)
		}
	}
	return nil
}

// Tracking implements providers.TrackingManager: Resend tracks opens and
// clicks per domain, once the domain has a tracking subdomain.
func (p *Provider) Tracking(ctx context.Context) (providers.Tracking, error) {
	infos, err := p.domainInfos(ctx)
	if err != nil {
		return providers.Tracking{}, err
	}
	t := providers.Tracking{}
	t.Scope, t.ScopeText = say(mTrackingScope, nil)
	var opens, clicks []string
	for _, d := range infos {
		if d.TrackingSubdomain == "" {
			continue
		}
		if d.OpenTracking {
			t.OpenPixel = true
			opens = append(opens, d.Name)
		}
		if d.ClickTracking {
			t.LinkRewrite = true
			clicks = append(clicks, d.Name)
		}
	}
	on := i18n.Args{"opens": strings.Join(opens, ", "), "clicks": strings.Join(clicks, ", ")}
	switch {
	case len(opens) > 0 && len(clicks) > 0:
		t.Detail, t.DetailText = say(mTrackingBoth, on)
	case len(opens) > 0:
		t.Detail, t.DetailText = say(mTrackingOpen, on)
	case len(clicks) > 0:
		t.Detail, t.DetailText = say(mTrackingClick, on)
	default:
		t.Detail, t.DetailText = say(mTrackingOff, nil)
	}
	return t, nil
}

// DisableTracking implements providers.TrackingManager.
func (p *Provider) DisableTracking(ctx context.Context) error {
	infos, err := p.domainInfos(ctx)
	if err != nil {
		return err
	}
	for _, d := range infos {
		if !d.OpenTracking && !d.ClickTracking {
			continue
		}
		body := map[string]any{"open_tracking": false, "click_tracking": false}
		if err := p.api.Do(ctx, "PATCH /domains/{id}", http.MethodPatch, apiclient.Path("/domains/%s", d.ID), body, nil); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) domainInfos(ctx context.Context) ([]domainInfo, error) {
	domains, _, err := p.listDomains(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domainInfo, 0, len(domains))
	for _, s := range domains {
		var d domainInfo
		if err := p.api.Do(ctx, "GET /domains/{id}", http.MethodGet, apiclient.Path("/domains/%s", s.ID), nil, &d); err != nil {
			return nil, err
		}
		if d.Name == "" {
			d.Name = s.Name
		}
		if d.ID == "" {
			d.ID = s.ID
		}
		out = append(out, d)
	}
	return out, nil
}

// Backfill implements providers.EventBackfiller with GET /emails/{id}: the
// email's last event. Emails still in flight give nothing.
func (p *Provider) Backfill(ctx context.Context, ids []string) ([]providers.Event, error) {
	out := []providers.Event{}
	for _, id := range ids {
		var e struct {
			ID        string          `json:"id"`
			To        []string        `json:"to"`
			CreatedAt string          `json:"created_at"`
			LastEvent string          `json:"last_event"`
			Tags      json.RawMessage `json:"tags"`
		}
		raw, err := p.api.DoRaw(ctx, "GET /emails/{id}", http.MethodGet, apiclient.Path("/emails/%s", id), nil, &e)
		if err != nil {
			var ce *providers.CallError
			if errors.As(err, &ce) && ce.Status == http.StatusNotFound {
				continue
			}
			return out, err
		}
		ev := providers.Event{ProviderMessageID: e.ID, OriginalType: "last_event · " + e.LastEvent, OccurredAt: parseTime(e.CreatedAt)}
		switch e.LastEvent {
		case "delivered", "opened", "clicked":
			ev.Type = providers.EventDelivered
		case "delivery_delayed":
			ev.Type, ev.BounceClass = providers.EventDeferred, bounce.Temporary
		case "complained":
			ev.Type = providers.EventComplaint
		case "bounced":
			// The email's state doesn't say whether the bounce was
			// permanent: counted as soft, so nothing is suppressed on a
			// guess (Resend's suppression list, synced separately, has the
			// permanent ones).
			ev.Type, ev.BounceClass = providers.EventSoftBounce, bounce.Other
		case "failed", "suppressed", "canceled":
			ev.Type, ev.BounceClass = providers.EventRejected, bounce.Other
		default:
			continue // queued, scheduled, sent: still in flight
		}
		if mid, err := uuid.Parse(tagValue(e.Tags, TagMessage)); err == nil {
			ev.MessageID = &mid
		}
		if len(e.To) > 0 {
			ev.Recipient = strings.ToLower(addressOf(e.To[0]))
		}
		if ev.OccurredAt.IsZero() {
			ev.OccurredAt = now().UTC()
		}
		ev.Raw = raw
		out = append(out, ev)
	}
	return out, nil
}

// Reputation implements providers.ReputationReporter from Resend's email
// metrics over the last seven days, with Resend's own definitions (bounced
// per sent, complained per delivered) that its 4% and 0.08% limits use.
func (p *Provider) Reputation(ctx context.Context) (providers.Reputation, error) {
	var m struct {
		Totals struct {
			Sent       float64 `json:"sent"`
			Delivered  float64 `json:"delivered"`
			Bounced    float64 `json:"bounced"`
			Complained float64 `json:"complained"`
		} `json:"totals"`
	}
	if err := p.api.Do(ctx, "GET /emails/metrics", http.MethodGet, "/emails/metrics", nil, &m); err != nil {
		return providers.Reputation{}, err
	}
	r := providers.Reputation{Window: "7 d"}
	if t := m.Totals; t.Sent > 0 {
		r.Bounce = t.Bounced / t.Sent * 100
	}
	if t := m.Totals; t.Delivered > 0 {
		r.Complaint = t.Complained / t.Delivered * 100
	}
	return r, nil
}

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "Resend status", "https://resend-status.com"
}
