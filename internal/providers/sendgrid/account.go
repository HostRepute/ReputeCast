package sendgrid

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

var (
	_ providers.Verifier           = (*Provider)(nil)
	_ providers.QuotaReporter      = (*Provider)(nil)
	_ providers.DomainManager      = (*Provider)(nil)
	_ providers.WebhookRegistrar   = (*Provider)(nil)
	_ providers.WebhookInspector   = (*Provider)(nil)
	_ providers.WebhookTester      = (*Provider)(nil)
	_ providers.WebhookVerifier    = (*Provider)(nil)
	_ providers.SuppressionSyncer  = (*Provider)(nil)
	_ providers.TrackingManager    = (*Provider)(nil)
	_ providers.ReputationReporter = (*Provider)(nil)
	_ providers.StatusPage         = (*Provider)(nil)
)

// restricted reports the 403 a valid key gets for a scope it lacks (a
// Mail Send-only key reading the account), as opposed to a 401 bad key.
func restricted(err error) bool {
	var ce *providers.CallError
	return errors.As(err, &ce) && ce.Status == http.StatusForbidden && strings.Contains(strings.ToLower(ce.Message), "access forbidden")
}

// Verify implements providers.Verifier.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) { return p.Quota(ctx) }

// Quota implements providers.QuotaReporter from the plan's credits
// (GET /v3/user/credits) and the account type and reputation.
func (p *Provider) Quota(ctx context.Context) (providers.Quota, error) {
	var q providers.Quota
	var credits struct {
		Remain         int64  `json:"remain"`
		Total          int64  `json:"total"`
		Overage        int64  `json:"overage"`
		Used           int64  `json:"used"`
		ResetFrequency string `json:"reset_frequency"`
	}
	err := p.api.Do(ctx, "GET /v3/user/credits", http.MethodGet, "/v3/user/credits", nil, &credits)
	switch {
	case err == nil:
		q.Kind, q.Max, q.Used = "monthly_plan", credits.Total, credits.Used
		if credits.ResetFrequency == "daily" {
			q.Kind = "rolling_24h"
		}
	case restricted(err):
	default:
		return providers.Quota{}, classify(err)
	}
	var acct struct {
		Type       string  `json:"type"`
		Reputation float64 `json:"reputation"`
	}
	err = p.api.Do(ctx, "GET /v3/user/account", http.MethodGet, "/v3/user/account", nil, &acct)
	switch {
	case err == nil:
		kind := "Paid account"
		if acct.Type == "free" {
			kind = "Free account"
		}
		q.Account = kind + " · reputation " + strconv.FormatFloat(acct.Reputation, 'f', -1, 64) + "%"
	case restricted(err):
		q.Account = "Restricted API key · account details need more scopes"
	default:
		return providers.Quota{}, classify(err)
	}
	return q, nil
}

// webhookSettings is one Event Webhook.
type webhookSettings struct {
	ID               string `json:"id,omitempty"`
	Enabled          bool   `json:"enabled"`
	URL              string `json:"url"`
	FriendlyName     string `json:"friendly_name,omitempty"`
	PublicKey        string `json:"public_key,omitempty"`
	Delivered        bool   `json:"delivered"`
	Bounce           bool   `json:"bounce"`
	Dropped          bool   `json:"dropped"`
	Deferred         bool   `json:"deferred"`
	SpamReport       bool   `json:"spam_report"`
	Unsubscribe      bool   `json:"unsubscribe"`
	GroupUnsubscribe bool   `json:"group_unsubscribe"`
	GroupResubscribe bool   `json:"group_resubscribe"`
	Processed        bool   `json:"processed"`
	Open             bool   `json:"open"`
	Click            bool   `json:"click"`
}

func (p *Provider) listWebhooks(ctx context.Context) ([]webhookSettings, error) {
	var all struct {
		Webhooks []webhookSettings `json:"webhooks"`
	}
	err := p.api.Do(ctx, "GET /v3/user/webhooks/event/settings/all", http.MethodGet, "/v3/user/webhooks/event/settings/all", nil, &all)
	return all.Webhooks, err
}

func sameURL(a, b string) bool { return strings.TrimRight(a, "/") == strings.TrimRight(b, "/") }

// RegisterWebhook implements providers.WebhookRegistrar: an Event Webhook
// without opens and clicks (ReputeCast's own), with signature verification
// turned on. An existing webhook for the URL is updated.
func (p *Provider) RegisterWebhook(ctx context.Context, hookURL string) error {
	hooks, err := p.listWebhooks(ctx)
	if err != nil {
		return err
	}
	want := webhookSettings{Enabled: true, URL: hookURL, FriendlyName: "ReputeCast", Delivered: true, Bounce: true, Dropped: true, Deferred: true,
		SpamReport: true, Unsubscribe: true, GroupUnsubscribe: true}
	id := ""
	for _, h := range hooks {
		if sameURL(h.URL, hookURL) {
			id = h.ID
		}
	}
	var saved webhookSettings
	if id != "" {
		err = p.api.Do(ctx, "PATCH /v3/user/webhooks/event/settings/{id}", http.MethodPatch, "/v3/user/webhooks/event/settings/"+url.PathEscape(id), want, &saved)
	} else {
		err = p.api.Do(ctx, "POST /v3/user/webhooks/event/settings", http.MethodPost, "/v3/user/webhooks/event/settings", want, &saved)
	}
	if err != nil {
		return err
	}
	if saved.ID != "" {
		id = saved.ID
	}
	if id == "" {
		return &providers.CallError{Op: "POST /v3/user/webhooks/event/settings", Message: "SendGrid didn't return the webhook's id"}
	}
	var signed struct {
		PublicKey string `json:"public_key"`
	}
	if err := p.api.Do(ctx, "PATCH /v3/user/webhooks/event/settings/signed/{id}", http.MethodPatch, "/v3/user/webhooks/event/settings/signed/"+url.PathEscape(id),
		map[string]bool{"enabled": true}, &signed); err != nil {
		return err
	}
	if signed.PublicKey != "" {
		verificationKeys.Set(p.keyID(pathOf(hookURL)), signed.PublicKey)
	}
	return nil
}

// InspectWebhook implements providers.WebhookInspector.
func (p *Provider) InspectWebhook(ctx context.Context, hookURL string) (providers.WebhookStatus, error) {
	st := providers.WebhookStatus{Queued: -1, Sent: -1}
	hooks, err := p.listWebhooks(ctx)
	if err != nil {
		return st, err
	}
	for _, h := range hooks {
		if sameURL(h.URL, hookURL) {
			st.Registered, st.ID, st.URL = h.Enabled, h.ID, h.URL
		}
	}
	return st, nil
}

// SendTestEvent implements providers.WebhookTester.
func (p *Provider) SendTestEvent(ctx context.Context, hookURL string) error {
	body := map[string]string{"url": hookURL}
	if hooks, err := p.listWebhooks(ctx); err == nil {
		for _, h := range hooks {
			if sameURL(h.URL, hookURL) {
				body["id"] = h.ID
			}
		}
	}
	return p.api.Do(ctx, "POST /v3/user/webhooks/event/test", http.MethodPost, "/v3/user/webhooks/event/test", body, nil)
}

type dnsEntry struct {
	Valid bool   `json:"valid"`
	Type  string `json:"type"`
	Host  string `json:"host"`
	Data  string `json:"data"`
}

type authDomain struct {
	ID      int64               `json:"id"`
	Domain  string              `json:"domain"`
	Default bool                `json:"default"`
	Valid   bool                `json:"valid"`
	DNS     map[string]dnsEntry `json:"dns"`
}

// findDomain returns the authenticated domain for domain, or nil.
func (p *Provider) findDomain(ctx context.Context, domain string) (*authDomain, error) {
	var list []authDomain
	path := "/v3/whitelabel/domains?limit=50&domain=" + url.QueryEscape(domain)
	if err := p.api.Do(ctx, "GET /v3/whitelabel/domains", http.MethodGet, path, nil, &list); err != nil {
		return nil, err
	}
	var best *authDomain
	for i := range list {
		d := &list[i]
		if !strings.EqualFold(d.Domain, domain) {
			continue
		}
		if best == nil || d.Valid && !best.Valid || d.Default && !best.Default {
			best = d
		}
	}
	return best, nil
}

// AddDomain implements providers.DomainManager with automated security
// (SendGrid hosts DKIM and SPF behind CNAMEs, so key rotation needs no DNS
// change). A domain already authenticated is read instead.
func (p *Provider) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	d, err := p.findDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	if d == nil {
		var created authDomain
		body := map[string]any{"domain": domain, "automatic_security": true, "default": false}
		if err := p.api.Do(ctx, "POST /v3/whitelabel/domains", http.MethodPost, "/v3/whitelabel/domains", body, &created); err != nil {
			return nil, err
		}
		d = &created
	}
	return dnsRecords(d, nil), nil
}

// DomainStatus implements providers.DomainManager: a domain that isn't
// valid yet is validated again, so the records show SendGrid's latest check.
func (p *Provider) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	d, err := p.findDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, &providers.CallError{Op: "GET /v3/whitelabel/domains", Status: http.StatusNotFound, Message: domain + " is not authenticated in SendGrid"}
	}
	var results map[string]struct {
		Valid  bool    `json:"valid"`
		Reason *string `json:"reason"`
	}
	if !d.Valid {
		var v struct {
			Valid   bool            `json:"valid"`
			Results json.RawMessage `json:"validation_results"`
		}
		if err := p.api.Do(ctx, "POST /v3/whitelabel/domains/{id}/validate", http.MethodPost, "/v3/whitelabel/domains/"+strconv.FormatInt(d.ID, 10)+"/validate", struct{}{}, &v); err == nil {
			_ = json.Unmarshal(v.Results, &results)
		}
	}
	reasons := map[string]string{}
	for k, r := range results {
		e, ok := d.DNS[k]
		switch {
		case !ok:
		case r.Valid:
			e.Valid = true
			d.DNS[k] = e
		case r.Reason != nil:
			reasons[k] = *r.Reason
		}
	}
	return dnsRecords(d, reasons), nil
}

// dnsRecords lists a domain's records in a stable order: return path,
// SPF, then DKIM.
func dnsRecords(d *authDomain, reasons map[string]string) []providers.DNSRecord {
	purposes := map[string]string{"mail_cname": "return_path", "mail_server": "return_path", "subdomain_spf": "spf", "dkim1": "dkim", "dkim2": "dkim", "dkim": "dkim"}
	order := []string{"mail_cname", "mail_server", "subdomain_spf", "dkim1", "dkim2", "dkim"}
	out := []providers.DNSRecord{}
	for _, k := range order {
		e, ok := d.DNS[k]
		if !ok || e.Host == "" {
			continue
		}
		rec := providers.DNSRecord{Purpose: purposes[k], Type: strings.ToUpper(e.Type), Name: strings.TrimSuffix(e.Host, "."), Value: e.Data, Status: "pending"}
		reason := strings.ToLower(reasons[k])
		switch {
		case e.Valid:
			rec.Status = "verified"
		case strings.Contains(reason, "but found"):
			rec.Status = "wrong_value"
		case strings.Contains(reason, "not found") || strings.HasPrefix(reason, "no "):
			rec.Status = "missing"
		}
		if rec.Type == "MX" && !strings.Contains(rec.Value, " ") {
			rec.Value = "10 " + rec.Value
		}
		out = append(out, rec)
	}
	return out
}

// suppressionLists are the SendGrid lists ReputeCast pulls, with the
// shared list's reason for each. Blocks are left out: they are about the
// sender (an IP or content block), not the address.
var suppressionLists = []struct{ path, reason, code string }{
	{"/v3/suppression/bounces", "hard_bounce", ""},
	{"/v3/suppression/invalid_emails", "hard_bounce", "invalid_email"},
	{"/v3/suppression/spam_reports", "complaint", "spam_report"},
	{"/v3/suppression/unsubscribes", "unsubscribe", "global_unsubscribe"},
}

// ListSuppressions implements providers.SuppressionSyncer.
func (p *Provider) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	out := []providers.Suppression{}
	for _, l := range suppressionLists {
		for offset := 0; offset < 1_000_000; offset += 500 {
			q := url.Values{"limit": {"500"}, "offset": {strconv.Itoa(offset)}}
			if !since.IsZero() {
				q.Set("start_time", strconv.FormatInt(since.Unix(), 10))
			}
			var page []struct {
				Created int64  `json:"created"`
				Email   string `json:"email"`
				Reason  string `json:"reason"`
				Status  string `json:"status"`
			}
			if err := p.api.Do(ctx, "GET "+l.path, http.MethodGet, l.path+"?"+q.Encode(), nil, &page); err != nil {
				return nil, err
			}
			for _, s := range page {
				code := l.code
				if code == "" {
					code = strings.TrimSpace(s.Status)
				}
				out = append(out, providers.Suppression{Email: strings.ToLower(strings.TrimSpace(s.Email)), Reason: l.reason, Code: code, CreatedAt: time.Unix(s.Created, 0).UTC()})
			}
			if len(page) < 500 {
				break
			}
		}
	}
	return out, nil
}

// AddSuppressions implements providers.SuppressionSyncer through the global
// unsubscribe group, which stops all mail to the address at SendGrid.
func (p *Provider) AddSuppressions(ctx context.Context, emails []string) error {
	for chunk := range slices.Chunk(emails, 500) {
		if err := p.api.Do(ctx, "POST /v3/asm/suppressions/global", http.MethodPost, "/v3/asm/suppressions/global", map[string]any{"recipient_emails": chunk}, nil); err != nil {
			return err
		}
	}
	return nil
}

// Tracking implements providers.TrackingManager. ReputeCast turns SendGrid's
// open, click and subscription tracking off on every message it sends, so
// account-level tracking never touches its mail.
func (p *Provider) Tracking(ctx context.Context) (providers.Tracking, error) {
	var click struct {
		Enabled bool `json:"enabled"`
	}
	var open struct {
		Enabled bool `json:"enabled"`
	}
	if err := p.api.Do(ctx, "GET /v3/tracking_settings/click", http.MethodGet, "/v3/tracking_settings/click", nil, &click); err != nil {
		return providers.Tracking{}, err
	}
	if err := p.api.Do(ctx, "GET /v3/tracking_settings/open", http.MethodGet, "/v3/tracking_settings/open", nil, &open); err != nil {
		return providers.Tracking{}, err
	}
	t := providers.Tracking{Scope: "Every message ReputeCast sends"}
	var on []string
	if open.Enabled {
		on = append(on, "open")
	}
	if click.Enabled {
		on = append(on, "click")
	}
	if len(on) == 0 {
		t.Detail = "Open and click tracking are off for the account and on every message ReputeCast sends."
	} else {
		t.Detail = "Account-level " + strings.Join(on, " and ") + " tracking is on, but ReputeCast turns SendGrid's tracking off on every message it sends."
	}
	return t, nil
}

// DisableTracking implements providers.TrackingManager: account-level open
// and click tracking off.
func (p *Provider) DisableTracking(ctx context.Context) error {
	if err := p.api.Do(ctx, "PATCH /v3/tracking_settings/click", http.MethodPatch, "/v3/tracking_settings/click", map[string]bool{"enabled": false}, nil); err != nil {
		return err
	}
	return p.api.Do(ctx, "PATCH /v3/tracking_settings/open", http.MethodPatch, "/v3/tracking_settings/open", map[string]bool{"enabled": false}, nil)
}

// Reputation implements providers.ReputationReporter from the account's
// global stats over the last seven days: bounces per request and spam
// reports per delivered message.
func (p *Provider) Reputation(ctx context.Context) (providers.Reputation, error) {
	start := time.Now().UTC().AddDate(0, 0, -6).Format("2006-01-02")
	var days []struct {
		Stats []struct {
			Metrics struct {
				Requests    float64 `json:"requests"`
				Delivered   float64 `json:"delivered"`
				Bounces     float64 `json:"bounces"`
				SpamReports float64 `json:"spam_reports"`
			} `json:"metrics"`
		} `json:"stats"`
	}
	if err := p.api.Do(ctx, "GET /v3/stats", http.MethodGet, "/v3/stats?start_date="+start, nil, &days); err != nil {
		return providers.Reputation{}, err
	}
	var req, del, bounces, spam float64
	for _, d := range days {
		for _, s := range d.Stats {
			req += s.Metrics.Requests
			del += s.Metrics.Delivered
			bounces += s.Metrics.Bounces
			spam += s.Metrics.SpamReports
		}
	}
	r := providers.Reputation{Window: "7 d"}
	if req > 0 {
		r.Bounce = bounces / req * 100
	}
	if del > 0 {
		r.Complaint = spam / del * 100
	}
	return r, nil
}

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "SendGrid status", "https://status.sendgrid.com"
}
