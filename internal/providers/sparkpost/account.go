package sparkpost

import (
	"cmp"
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
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
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
	_ providers.EventBackfiller    = (*Provider)(nil)
	_ providers.ReputationReporter = (*Provider)(nil)
	_ providers.StatusPage         = (*Provider)(nil)
)

type account struct {
	CustomerID json.RawMessage `json:"customer_id"`
	Status     string          `json:"status"`
	Options    struct {
		RestTrackingDefault *bool `json:"rest_tracking_default"`
		ClickTracking       *bool `json:"click_tracking"`
	} `json:"options"`
	Subscription struct {
		Name string `json:"name"`
	} `json:"subscription"`
	Usage *struct {
		Day struct {
			Used  int64 `json:"used"`
			Limit int64 `json:"limit"`
		} `json:"day"`
		Month struct {
			Used  int64 `json:"used"`
			Limit int64 `json:"limit"`
		} `json:"month"`
	} `json:"usage"`
}

func (p *Provider) account(ctx context.Context, usage bool) (account, error) {
	path := "/api/v1/account"
	if usage {
		path += "?include=usage"
	}
	var out struct {
		Results account `json:"results"`
	}
	err := p.api.Do(ctx, "GET /api/v1/account", http.MethodGet, path, nil, &out)
	return out.Results, err
}

// Verify implements providers.Verifier.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) { return p.Quota(ctx) }

// Quota implements providers.QuotaReporter from the account's monthly (or,
// without one, daily) sending limit and usage. Enterprise accounts have no
// usage data.
func (p *Provider) Quota(ctx context.Context) (providers.Quota, error) {
	a, err := p.account(ctx, true)
	if err != nil {
		return providers.Quota{}, classify(err)
	}
	q := providers.Quota{Account: "Account " + strings.Trim(string(a.CustomerID), `"`) + " · " + cmp.Or(a.Status, "active")}
	if a.Subscription.Name != "" {
		q.Account += " · " + a.Subscription.Name
	}
	if u := a.Usage; u != nil {
		switch {
		case u.Month.Limit > 0:
			q.Kind, q.Max, q.Used = "monthly_plan", u.Month.Limit, u.Month.Used
		case u.Day.Limit > 0:
			q.Kind, q.Max, q.Used = "rolling_24h", u.Day.Limit, u.Day.Used
		}
	}
	return q, nil
}

// Reputation implements providers.ReputationReporter from the deliverability
// metrics of the last seven days: bounces per targeted message and spam
// complaints per accepted message.
func (p *Provider) Reputation(ctx context.Context) (providers.Reputation, error) {
	q := url.Values{"from": {time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02T15:04")}, "metrics": {"count_targeted,count_accepted,count_bounce,count_spam_complaint"}}
	var out struct {
		Results []map[string]float64 `json:"results"`
	}
	if err := p.api.Do(ctx, "GET /api/v1/metrics/deliverability", http.MethodGet, "/api/v1/metrics/deliverability?"+q.Encode(), nil, &out); err != nil {
		return providers.Reputation{}, classify(err)
	}
	r := providers.Reputation{Window: "7 d"}
	if len(out.Results) == 0 {
		return r, nil
	}
	m := out.Results[0]
	if m["count_targeted"] > 0 {
		r.Bounce = m["count_bounce"] / m["count_targeted"] * 100
	}
	if m["count_accepted"] > 0 {
		r.Complaint = m["count_spam_complaint"] / m["count_accepted"] * 100
	}
	return r, nil
}

type sendingDomain struct {
	DKIM struct {
		Public        string `json:"public"`
		Selector      string `json:"selector"`
		SigningDomain string `json:"signing_domain"`
	} `json:"dkim"`
	Status struct {
		DKIMStatus  string `json:"dkim_status"`
		CNAMEStatus string `json:"cname_status"`
	} `json:"status"`
}

// AddDomain implements providers.DomainManager: a sending domain with a
// 2048-bit DKIM key, which also serves as bounce domain (a CNAME to
// SparkPost), as SparkPost recommends. A domain already in the account is
// read instead.
func (p *Provider) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	body := map[string]any{"domain": domain, "generate_dkim": true, "dkim_key_length": 2048}
	var created struct {
		Results sendingDomain `json:"results"`
	}
	err := p.api.Do(ctx, "POST /api/v1/sending-domains", http.MethodPost, "/api/v1/sending-domains", body, &created)
	if err != nil {
		var ce *providers.CallError
		if errors.As(err, &ce) && ce.Status == http.StatusConflict {
			return p.DomainStatus(ctx, domain)
		}
		return nil, classify(err)
	}
	return p.records(created.Results, domain), nil
}

// DomainStatus implements providers.DomainManager. A domain whose DKIM or
// bounce CNAME isn't valid yet is verified again first.
func (p *Provider) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	var got struct {
		Results sendingDomain `json:"results"`
	}
	path := apiclient.Path("/api/v1/sending-domains/%s", domain)
	if err := p.api.Do(ctx, "GET /api/v1/sending-domains/{domain}", http.MethodGet, path, nil, &got); err != nil {
		return nil, classify(err)
	}
	d := got.Results
	if d.Status.DKIMStatus != "valid" || d.Status.CNAMEStatus != "valid" {
		var v struct {
			Results struct {
				DKIMStatus  string `json:"dkim_status"`
				CNAMEStatus string `json:"cname_status"`
			} `json:"results"`
		}
		if err := p.api.Do(ctx, "POST /api/v1/sending-domains/{domain}/verify", http.MethodPost, path+"/verify", map[string]bool{"dkim_verify": true, "cname_verify": true}, &v); err == nil {
			d.Status.DKIMStatus, d.Status.CNAMEStatus = cmp.Or(v.Results.DKIMStatus, d.Status.DKIMStatus), cmp.Or(v.Results.CNAMEStatus, d.Status.CNAMEStatus)
		}
	}
	return p.records(d, domain), nil
}

func (p *Provider) records(d sendingDomain, domain string) []providers.DNSRecord {
	out := []providers.DNSRecord{}
	if d.DKIM.Selector != "" && d.DKIM.Public != "" {
		signing := cmp.Or(d.DKIM.SigningDomain, domain)
		out = append(out, providers.DNSRecord{Purpose: "dkim", Type: "TXT", Name: d.DKIM.Selector + "._domainkey." + signing,
			Value: "v=DKIM1; k=rsa; h=sha256; p=" + d.DKIM.Public, Status: domainStatus(d.Status.DKIMStatus)})
	}
	target := "sparkpostmail.com"
	if strings.Contains(p.api.BaseURL, ".eu.") || strings.EqualFold(p.s.Region, "eu") {
		target = "eu.sparkpostmail.com"
	}
	return append(out, providers.DNSRecord{Purpose: "return_path", Type: "CNAME", Name: domain, Value: target, Status: domainStatus(d.Status.CNAMEStatus)})
}

func domainStatus(s string) string {
	switch s {
	case "valid":
		return "verified"
	case "invalid":
		return "wrong_value"
	}
	return "pending"
}

// webhookEvents are the events ReputeCast subscribes to; opens and clicks
// are tracked by ReputeCast itself.
var webhookEvents = []string{"delivery", "bounce", "out_of_band", "delay", "spam_complaint", "policy_rejection",
	"generation_failure", "generation_rejection", "list_unsubscribe", "link_unsubscribe"}

type webhook struct {
	ID     string   `json:"id,omitempty"`
	Name   string   `json:"name"`
	Target string   `json:"target"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

func (p *Provider) webhooks(ctx context.Context) ([]webhook, error) {
	var out struct {
		Results []webhook `json:"results"`
	}
	if err := p.api.Do(ctx, "GET /api/v1/webhooks", http.MethodGet, "/api/v1/webhooks", nil, &out); err != nil {
		return nil, classify(err)
	}
	return out.Results, nil
}

func sameURL(a, b string) bool { return strings.TrimRight(a, "/") == strings.TrimRight(b, "/") }

func (p *Provider) findWebhook(ctx context.Context, hookURL string) (*webhook, error) {
	hooks, err := p.webhooks(ctx)
	if err != nil {
		return nil, err
	}
	for i := range hooks {
		if sameURL(hooks[i].Target, hookURL) {
			return &hooks[i], nil
		}
	}
	return nil, nil
}

// RegisterWebhook implements providers.WebhookRegistrar: a webhook with
// basic auth (SparkPost posts a test batch to it before saving, so the
// endpoint must be reachable). An existing webhook for the URL is updated.
func (p *Provider) RegisterWebhook(ctx context.Context, hookURL string) error {
	existing, err := p.findWebhook(ctx, hookURL)
	if err != nil {
		return err
	}
	body := map[string]any{"name": "ReputeCast", "target": hookURL, "events": webhookEvents, "active": true,
		"auth_type": "basic", "auth_credentials": map[string]string{"username": WebhookUser, "password": p.cfg.WebhookSecret}}
	if existing != nil {
		return classify(p.api.Do(ctx, "PUT /api/v1/webhooks/{id}", http.MethodPut, apiclient.Path("/api/v1/webhooks/%s", existing.ID), body, nil))
	}
	return classify(p.api.Do(ctx, "POST /api/v1/webhooks", http.MethodPost, "/api/v1/webhooks", body, nil))
}

// InspectWebhook implements providers.WebhookInspector. SparkPost's batch
// status lists the batches that failed or were retried in the last day;
// those still failing count as queued.
func (p *Provider) InspectWebhook(ctx context.Context, hookURL string) (providers.WebhookStatus, error) {
	st := providers.WebhookStatus{Queued: -1, Sent: -1}
	hook, err := p.findWebhook(ctx, hookURL)
	if err != nil || hook == nil {
		return st, err
	}
	st.Registered, st.ID, st.URL = hook.Active, hook.ID, hook.Target
	var batches struct {
		Results []struct {
			ResponseCode string `json:"response_code"`
		} `json:"results"`
	}
	if err := p.api.Do(ctx, "GET /api/v1/webhooks/{id}/batch-status", http.MethodGet, apiclient.Path("/api/v1/webhooks/%s/batch-status", hook.ID)+"?limit=1000", nil, &batches); err == nil {
		st.Queued = 0
		for _, b := range batches.Results {
			if b.ResponseCode != "200" {
				st.Queued++
			}
		}
	}
	return st, nil
}

// SendTestEvent implements providers.WebhookTester with the webhook's
// validate call (an empty event batch).
func (p *Provider) SendTestEvent(ctx context.Context, hookURL string) error {
	hook, err := p.findWebhook(ctx, hookURL)
	if err != nil {
		return err
	}
	if hook == nil {
		return &providers.CallError{Op: "POST /api/v1/webhooks/{id}/validate", Code: "not_registered", Message: "No SparkPost webhook points at " + hookURL + ". Register it first."}
	}
	var out struct {
		Results struct {
			Msg      string `json:"msg"`
			Response struct {
				Status int `json:"status"`
			} `json:"response"`
		} `json:"results"`
	}
	if err := p.api.Do(ctx, "POST /api/v1/webhooks/{id}/validate", http.MethodPost, apiclient.Path("/api/v1/webhooks/%s/validate", hook.ID),
		map[string]any{"message": map[string]any{"msys": map[string]any{}}}, &out); err != nil {
		return classify(err)
	}
	if s := out.Results.Response.Status; s != 0 && s != http.StatusOK {
		return &providers.CallError{Op: "POST /api/v1/webhooks/{id}/validate", Status: s, Message: "the endpoint answered " + strconv.Itoa(s) + " to SparkPost's test batch"}
	}
	return nil
}

// ListSuppressions implements providers.SuppressionSyncer over both of
// SparkPost's lists (transactional and non-transactional), one entry per
// address.
func (p *Provider) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	q := url.Values{"types": {"transactional,non_transactional"}, "per_page": {"10000"}, "cursor": {"initial"}}
	if !since.IsZero() {
		q.Set("from", since.UTC().Format("2006-01-02T15:04:05Z"))
	}
	next := "/api/v1/suppression-list?" + q.Encode()
	seen := map[string]bool{}
	out := []providers.Suppression{}
	for range 1000 {
		var page struct {
			Results []struct {
				Recipient string `json:"recipient"`
				Type      string `json:"type"`
				Source    string `json:"source"`
				Created   string `json:"created"`
			} `json:"results"`
			Links []struct {
				Href string `json:"href"`
				Rel  string `json:"rel"`
			} `json:"links"`
		}
		if err := p.api.Do(ctx, "GET /api/v1/suppression-list", http.MethodGet, next, nil, &page); err != nil {
			return nil, classify(err)
		}
		for _, s := range page.Results {
			email := strings.ToLower(strings.TrimSpace(s.Recipient))
			// open_tracking entries only stop tracking, never sending.
			if email == "" || seen[email] || s.Type == "open_tracking" {
				continue
			}
			seen[email] = true
			created, _ := time.Parse(time.RFC3339, s.Created)
			out = append(out, providers.Suppression{Email: email, Reason: suppressionReason(s.Source), Code: s.Source, CreatedAt: created.UTC()})
		}
		next = ""
		for _, l := range page.Links {
			if l.Rel == "next" && strings.HasPrefix(l.Href, "/api/v1/") {
				next = l.Href
			}
		}
		if next == "" || len(page.Results) == 0 {
			break
		}
	}
	return out, nil
}

func suppressionReason(source string) string {
	switch source {
	case "Bounce Rule":
		return "hard_bounce"
	case "Spam Complaint":
		return "complaint"
	case "List Unsubscribe", "Unsubscribe Link":
		return "unsubscribe"
	}
	return "manual"
}

// AddSuppressions implements providers.SuppressionSyncer: each address is
// added to both lists, since ReputeCast marks its messages transactional
// (5,000 addresses, 10,000 entries per call).
func (p *Provider) AddSuppressions(ctx context.Context, emails []string) error {
	for chunk := range slices.Chunk(emails, 5000) {
		entries := make([]map[string]string, 0, 2*len(chunk))
		for _, e := range chunk {
			for _, typ := range []string{"transactional", "non_transactional"} {
				entries = append(entries, map[string]string{"recipient": e, "type": typ, "description": "Suppressed in ReputeCast"})
			}
		}
		if err := p.api.Do(ctx, "PUT /api/v1/suppression-list", http.MethodPut, "/api/v1/suppression-list/", map[string]any{"recipients": entries}, nil); err != nil {
			return classify(err)
		}
	}
	return nil
}

// Tracking implements providers.TrackingManager from the account's
// defaults. ReputeCast turns open and click tracking off on every
// transmission, so they never touch its mail.
func (p *Provider) Tracking(ctx context.Context) (providers.Tracking, error) {
	a, err := p.account(ctx, false)
	if err != nil {
		return providers.Tracking{}, classify(err)
	}
	t := providers.Tracking{Scope: "Every transmission ReputeCast sends"}
	opens := a.Options.RestTrackingDefault == nil || *a.Options.RestTrackingDefault
	clicks := opens && (a.Options.ClickTracking == nil || *a.Options.ClickTracking)
	switch {
	case opens || clicks:
		t.Detail = "The account tracks opens and clicks by default, but ReputeCast turns tracking off on every transmission it sends."
	default:
		t.Detail = "Open and click tracking are off by default and on every transmission ReputeCast sends."
	}
	return t, nil
}

// DisableTracking implements providers.TrackingManager: the account's
// default open and click tracking off.
func (p *Provider) DisableTracking(ctx context.Context) error {
	body := map[string]any{"options": map[string]bool{"rest_tracking_default": false, "click_tracking": false}}
	return classify(p.api.Do(ctx, "PUT /api/v1/account", http.MethodPut, "/api/v1/account", body, nil))
}

// Backfill implements providers.EventBackfiller from the events API (ten
// days of history): the latest outcome of each transmission.
func (p *Provider) Backfill(ctx context.Context, ids []string) ([]providers.Event, error) {
	out := []providers.Event{}
	from := time.Now().UTC().AddDate(0, 0, -10).Format("2006-01-02T15:04:05Z")
	for chunk := range slices.Chunk(ids, 100) {
		q := url.Values{"transmissions": {strings.Join(chunk, ",")}, "from": {from}, "per_page": {"10000"},
			"events": {"delivery,bounce,out_of_band,delay,spam_complaint,policy_rejection,generation_failure,generation_rejection"}}
		var page struct {
			Results []json.RawMessage `json:"results"`
		}
		if err := p.api.Do(ctx, "GET /api/v1/events/message", http.MethodGet, "/api/v1/events/message?"+q.Encode(), nil, &page); err != nil {
			return out, classify(err)
		}
		latest := map[string]providers.Event{}
		for _, raw := range page.Results {
			var e event
			if json.Unmarshal(raw, &e) != nil {
				continue
			}
			ev, ok := normalize(e)
			if !ok {
				continue
			}
			ev.Raw = append(json.RawMessage(nil), raw...)
			if cur, ok := latest[ev.ProviderMessageID]; !ok || ev.OccurredAt.After(cur.OccurredAt) {
				latest[ev.ProviderMessageID] = ev
			}
		}
		for _, id := range chunk {
			if ev, ok := latest[id]; ok {
				out = append(out, ev)
			}
		}
	}
	return out, nil
}

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "SparkPost status", "https://status.sparkpost.com"
}
