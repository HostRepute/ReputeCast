package mailgun

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
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

type dnsRecord struct {
	RecordType string          `json:"record_type"`
	Name       string          `json:"name"`
	Value      string          `json:"value"`
	Valid      string          `json:"valid"`
	Priority   json.RawMessage `json:"priority"`
}

type domainResponse struct {
	Domain struct {
		Name  string `json:"name"`
		State string `json:"state"`
		Type  string `json:"type"`
	} `json:"domain"`
	Sending   []dnsRecord `json:"sending_dns_records"`
	Receiving []dnsRecord `json:"receiving_dns_records"`
}

func (p *Provider) domain(ctx context.Context, name string) (domainResponse, error) {
	var d domainResponse
	err := p.api.Do(ctx, "GET /v4/domains/{name}", http.MethodGet, apiclient.Path("/v4/domains/%s", name), nil, &d)
	if err != nil {
		return d, p.classify(err, name)
	}
	return d, nil
}

// Verify implements providers.Verifier.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) { return p.Quota(ctx) }

// Quota implements providers.QuotaReporter: the account's custom monthly
// sending limit when one is set (Mailgun stops sending at it), else the
// plan allowance entered at connect time against the messages processed
// this month (what Mailgun bills). A sandbox domain only sends to
// authorized recipients.
func (p *Provider) Quota(ctx context.Context) (providers.Quota, error) {
	d, err := p.domain(ctx, p.s.Domain)
	if err != nil {
		return providers.Quota{}, err
	}
	q := providers.Quota{Sandbox: d.Domain.Type == "sandbox"}
	// "Domain mg.acme.io · active", "Sandbox domain · authorized recipients
	// only", then " · custom limit 100,000 per month".
	account := i18n.Args{"sandbox": yesNo(q.Sandbox), "domain": p.s.Domain, "state": d.Domain.State, "limit": "no"}
	q.Account, q.AccountText = say(mAccount, account)
	var limit struct {
		Limit   int64 `json:"limit"`
		Current int64 `json:"current"`
	}
	err = p.api.Do(ctx, "GET /v5/accounts/limit/custom/monthly", http.MethodGet, "/v5/accounts/limit/custom/monthly", nil, &limit)
	if err == nil && limit.Limit > 0 {
		q.Kind, q.Max, q.Used = "monthly_plan", limit.Limit, limit.Current
		account["limit"], account["n"] = "yes", limit.Limit
		q.Account, q.AccountText = say(mAccount, account)
		return q, nil
	}
	now := time.Now().UTC()
	if m, err := p.metrics(ctx, time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), now, "month", "processed_count"); err == nil {
		q.Kind, q.Used, q.Max = "monthly_plan", int64(m["processed_count"]), int64(p.s.MonthlyLimit)
	}
	return q, nil
}

// metrics reads account-wide totals from the Metrics API.
func (p *Provider) metrics(ctx context.Context, start, end time.Time, resolution string, names ...string) (map[string]float64, error) {
	body := map[string]any{"start": start.Format(time.RFC1123Z), "end": end.Format(time.RFC1123Z), "resolution": resolution,
		"dimensions": []string{"time"}, "metrics": names, "include_aggregates": true}
	var out struct {
		Aggregates struct {
			Metrics map[string]json.RawMessage `json:"metrics"`
		} `json:"aggregates"`
	}
	if err := p.api.Do(ctx, "POST /v1/analytics/metrics", http.MethodPost, "/v1/analytics/metrics", body, &out); err != nil {
		return nil, err
	}
	m := map[string]float64{}
	for k, raw := range out.Aggregates.Metrics {
		if f, err := strconv.ParseFloat(strings.Trim(strings.TrimSpace(string(raw)), `"%`), 64); err == nil {
			m[k] = f
		}
	}
	return m, nil
}

// Reputation implements providers.ReputationReporter with Mailgun's own
// definitions over the last seven days: bounces per processed message and
// complaints per delivered message, account-wide (what its limits use).
func (p *Provider) Reputation(ctx context.Context) (providers.Reputation, error) {
	now := time.Now().UTC()
	m, err := p.metrics(ctx, now.AddDate(0, 0, -7), now, "day", "processed_count", "delivered_count", "bounced_count", "complained_count")
	if err != nil {
		return providers.Reputation{}, err
	}
	r := providers.Reputation{Window: "7 d"}
	if m["processed_count"] > 0 {
		r.Bounce = m["bounced_count"] / m["processed_count"] * 100
	}
	if m["delivered_count"] > 0 {
		r.Complaint = m["complained_count"] / m["delivered_count"] * 100
	}
	return r, nil
}

// AddDomain implements providers.DomainManager: a domain with automatic
// sender security (DKIM through CNAMEs Mailgun rotates) and HTTPS
// tracking links. A domain already in the account is read instead.
func (p *Provider) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	ct, body := form([][2]string{{"name", domain}, {"use_automatic_sender_security", "true"}, {"dkim_key_size", "2048"}, {"web_scheme", "https"}})
	res, err := p.api.Exec(ctx, apiclient.Request{Op: "POST /v4/domains", Method: http.MethodPost, Path: "/v4/domains", ContentType: ct, Body: body})
	if err != nil {
		var ce *providers.CallError
		if errors.As(err, &ce) && ce.Status == http.StatusBadRequest && strings.Contains(strings.ToLower(ce.Message), "already") {
			return p.DomainStatus(ctx, domain)
		}
		return nil, p.classify(err, domain)
	}
	var d domainResponse
	if err := json.Unmarshal(res.Body, &d); err != nil {
		return nil, &providers.CallError{Op: "POST /v4/domains", Status: res.Status, Temporary: true, Message: "unexpected response: " + err.Error()}
	}
	return records(d, domain), nil
}

// DomainStatus implements providers.DomainManager. A domain that isn't
// active yet is verified again, so the records show Mailgun's latest check.
func (p *Provider) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	d, err := p.domain(ctx, domain)
	if err != nil {
		return nil, err
	}
	if d.Domain.State != "active" {
		var v domainResponse
		if err := p.api.Do(ctx, "PUT /v4/domains/{name}/verify", http.MethodPut, apiclient.Path("/v4/domains/%s/verify", domain), nil, &v); err == nil && len(v.Sending) > 0 {
			d = v
		}
	}
	return records(d, domain), nil
}

func records(d domainResponse, domain string) []providers.DNSRecord {
	out := []providers.DNSRecord{}
	for _, r := range d.Sending {
		rec := providers.DNSRecord{Type: strings.ToUpper(r.RecordType), Name: strings.TrimSuffix(r.Name, "."), Value: r.Value, Status: validity(r.Valid)}
		switch {
		case rec.Type == "TXT" && strings.HasPrefix(strings.ToLower(r.Value), "v=spf1"):
			rec.Purpose = "spf"
		case strings.Contains(strings.ToLower(r.Name), "._domainkey."):
			rec.Purpose = "dkim"
		default:
			continue // the tracking CNAME: not needed to send
		}
		out = append(out, rec)
	}
	// Bounces come back to Mailgun through the domain's MX records.
	for _, r := range d.Receiving {
		if !strings.EqualFold(r.RecordType, "MX") {
			continue
		}
		prio := strings.Trim(strings.TrimSpace(string(r.Priority)), `"`)
		if prio == "" || prio == "null" {
			prio = "10"
		}
		name := strings.TrimSuffix(r.Name, ".")
		if name == "" {
			name = domain
		}
		out = append(out, providers.DNSRecord{Purpose: "return_path", Type: "MX", Name: name, Value: prio + " " + r.Value, Status: validity(r.Valid)})
	}
	return out
}

func validity(v string) string {
	switch v {
	case "valid":
		return "verified"
	case "invalid":
		return "wrong_value"
	}
	return "pending"
}

// webhookEvents are the domain webhooks ReputeCast registers. Opens and
// clicks are tracked by ReputeCast itself.
var webhookEvents = []string{"delivered", "permanent_fail", "temporary_fail", "complained", "unsubscribed"}

func (p *Provider) webhooks(ctx context.Context) (map[string][]string, error) {
	var out struct {
		Webhooks map[string]*struct {
			URLs []string `json:"urls"`
			URL  string   `json:"url"`
		} `json:"webhooks"`
	}
	if err := p.api.Do(ctx, "GET /v3/domains/{domain}/webhooks", http.MethodGet, apiclient.Path("/v3/domains/%s/webhooks", p.s.Domain), nil, &out); err != nil {
		return nil, p.classify(err, p.s.Domain)
	}
	hooks := map[string][]string{}
	for id, h := range out.Webhooks {
		if h == nil {
			continue
		}
		urls := h.URLs
		if len(urls) == 0 && h.URL != "" {
			urls = []string{h.URL}
		}
		hooks[id] = urls
	}
	return hooks, nil
}

func sameURL(a, b string) bool { return strings.TrimRight(a, "/") == strings.TrimRight(b, "/") }

func hasURL(urls []string, u string) bool {
	return slices.ContainsFunc(urls, func(x string) bool { return sameURL(x, u) })
}

// RegisterWebhook implements providers.WebhookRegistrar: the URL is added to
// each of webhookEvents next to any URLs already there (Mailgun allows three
// per event).
func (p *Provider) RegisterWebhook(ctx context.Context, hookURL string) error {
	hooks, err := p.webhooks(ctx)
	if err != nil {
		return err
	}
	for _, id := range webhookEvents {
		urls := hooks[id]
		if hasURL(urls, hookURL) {
			continue
		}
		if len(urls) >= 3 {
			ce := &providers.CallError{Op: "PUT /v3/domains/{domain}/webhooks", Code: "too_many_urls"}
			ce.Message, ce.Text = say(mTooManyURLs, i18n.Args{"event": id})
			return ce
		}
		var fields [][2]string
		op, method, path := "POST /v3/domains/{domain}/webhooks", http.MethodPost, apiclient.Path("/v3/domains/%s/webhooks", p.s.Domain)
		if len(urls) > 0 {
			op, method, path = "PUT /v3/domains/{domain}/webhooks/{id}", http.MethodPut, apiclient.Path("/v3/domains/%s/webhooks/%s", p.s.Domain, id)
		} else {
			fields = append(fields, [2]string{"id", id})
		}
		for _, u := range append(slices.Clone(urls), hookURL) {
			fields = append(fields, [2]string{"url", u})
		}
		ct, body := form(fields)
		if _, err := p.api.Exec(ctx, apiclient.Request{Op: op, Method: method, Path: path, ContentType: ct, Body: body}); err != nil {
			return p.classify(err, p.s.Domain)
		}
	}
	return nil
}

// UnregisterWebhook implements providers.WebhookUnregistrar: the URL is
// taken off every event that lists it. The event keeps its other URLs
// (PUT /v3/domains/{domain}/webhooks/{id}), or its webhook is deleted when
// the URL was the only one (DELETE /v3/domains/{domain}/webhooks/{id},
// https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/domain-webhooks).
func (p *Provider) UnregisterWebhook(ctx context.Context, hookURL string) error {
	hooks, err := p.webhooks(ctx)
	if err != nil {
		return err
	}
	for _, id := range slices.Sorted(maps.Keys(hooks)) {
		if !hasURL(hooks[id], hookURL) {
			continue
		}
		keep := slices.DeleteFunc(slices.Clone(hooks[id]), func(u string) bool { return sameURL(u, hookURL) })
		path := apiclient.Path("/v3/domains/%s/webhooks/%s", p.s.Domain, id)
		if len(keep) == 0 {
			err = p.api.Do(ctx, "DELETE /v3/domains/{domain}/webhooks/{id}", http.MethodDelete, path, nil, nil)
		} else {
			fields := make([][2]string, len(keep))
			for i, u := range keep {
				fields[i] = [2]string{"url", u}
			}
			ct, body := form(fields)
			_, err = p.api.Exec(ctx, apiclient.Request{Op: "PUT /v3/domains/{domain}/webhooks/{id}", Method: http.MethodPut, Path: path, ContentType: ct, Body: body})
		}
		var ce *providers.CallError
		if errors.As(err, &ce) && ce.Status == http.StatusNotFound {
			continue // removed in Mailgun since the listing
		}
		if err != nil {
			return p.classify(err, p.s.Domain)
		}
	}
	return nil
}

// InspectWebhook implements providers.WebhookInspector: registered when
// the URL receives deliveries and permanent failures.
func (p *Provider) InspectWebhook(ctx context.Context, hookURL string) (providers.WebhookStatus, error) {
	st := providers.WebhookStatus{Queued: -1, Sent: -1, URL: hookURL}
	hooks, err := p.webhooks(ctx)
	if err != nil {
		return st, err
	}
	var on []string
	for _, id := range webhookEvents {
		if hasURL(hooks[id], hookURL) {
			on = append(on, id)
		}
	}
	st.ID = strings.Join(on, ",")
	st.Registered = hasURL(hooks["delivered"], hookURL) && hasURL(hooks["permanent_fail"], hookURL)
	return st, nil
}

// suppressionTables are the domain's suppression lists with the shared
// list's reason for each.
var suppressionTables = []struct{ table, reason string }{
	{"bounces", "hard_bounce"},
	{"complaints", "complaint"},
	{"unsubscribes", "unsubscribe"},
}

// ListSuppressions implements providers.SuppressionSyncer. Mailgun pages
// its tables by address, not by date, so every page is read and filtered.
// Unsubscribes from a single tag are left out: only "*" stops all mail.
func (p *Provider) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	out := []providers.Suppression{}
	for _, t := range suppressionTables {
		next := apiclient.Path("/v3/%s/", p.s.Domain) + t.table + "?limit=1000"
		for range 1000 {
			var page struct {
				Items []struct {
					Address   string   `json:"address"`
					Code      string   `json:"code"`
					Error     string   `json:"error"`
					Tags      []string `json:"tags"`
					CreatedAt string   `json:"created_at"`
				} `json:"items"`
				Paging struct {
					Next string `json:"next"`
				} `json:"paging"`
			}
			// Paging links are absolute URLs on the API host.
			res, err := p.api.Exec(ctx, apiclient.Request{Op: "GET /v3/{domain}/" + t.table, Method: http.MethodGet, Path: next})
			if err != nil {
				return nil, p.classify(err, p.s.Domain)
			}
			if err := json.Unmarshal(res.Body, &page); err != nil {
				return nil, &providers.CallError{Op: "GET /v3/{domain}/" + t.table, Status: res.Status, Temporary: true, Message: "unexpected response: " + err.Error()}
			}
			for _, s := range page.Items {
				if t.table == "unsubscribes" && len(s.Tags) > 0 && !slices.Contains(s.Tags, "*") {
					continue
				}
				created, _ := time.Parse(time.RFC1123, s.CreatedAt)
				if !since.IsZero() && !created.IsZero() && created.Before(since) {
					continue
				}
				out = append(out, providers.Suppression{Email: strings.ToLower(strings.TrimSpace(s.Address)), Reason: t.reason, Code: s.Code, CreatedAt: created.UTC()})
			}
			if len(page.Items) == 0 || page.Paging.Next == "" {
				break
			}
			next = page.Paging.Next
		}
	}
	return out, nil
}

// AddSuppressions implements providers.SuppressionSyncer: addresses join
// the domain's unsubscribe table for every tag ("*"), which stops all mail
// to them (1,000 per call).
func (p *Provider) AddSuppressions(ctx context.Context, emails []string) error {
	for chunk := range slices.Chunk(emails, 1000) {
		items := make([]map[string]any, len(chunk))
		for i, e := range chunk {
			items[i] = map[string]any{"address": e, "tags": []string{"*"}}
		}
		if err := p.api.Do(ctx, "POST /v3/{domain}/unsubscribes", http.MethodPost, apiclient.Path("/v3/%s/unsubscribes", p.s.Domain), items, nil); err != nil {
			return p.classify(err, p.s.Domain)
		}
	}
	return nil
}

// RemoveSuppressions implements providers.SuppressionRemover on the sending
// domain's unsubscribe list (where AddSuppressions puts them).
func (p *Provider) RemoveSuppressions(ctx context.Context, emails []string) error {
	for _, e := range emails {
		err := p.api.Do(ctx, "DELETE /v3/{domain}/unsubscribes", http.MethodDelete, apiclient.Path("/v3/%s/unsubscribes/%s", p.s.Domain, e), nil, nil)
		if err != nil && !apiclient.NotFound(err) {
			return p.classify(err, p.s.Domain)
		}
	}
	return nil
}

// SuppressionScope implements providers.SuppressionScoper: Mailgun keeps its
// lists per sending domain.
func (p *Provider) SuppressionScope() providers.SuppressionScope {
	return providers.SuppressionScope{Scope: providers.SuppressionScopeDomain, Writable: true}
}

// Tracking implements providers.TrackingManager from the domain's tracking
// settings. ReputeCast turns open and click tracking off on every message
// (o:tracking=no), but unsubscribe tracking still inserts Mailgun's own
// unsubscribe link, so it shows as link rewriting.
func (p *Provider) Tracking(ctx context.Context) (providers.Tracking, error) {
	var out struct {
		Tracking struct {
			Open        struct{ Active json.RawMessage } `json:"open"`
			Click       struct{ Active json.RawMessage } `json:"click"`
			Unsubscribe struct{ Active json.RawMessage } `json:"unsubscribe"`
		} `json:"tracking"`
	}
	if err := p.api.Do(ctx, "GET /v3/domains/{domain}/tracking", http.MethodGet, apiclient.Path("/v3/domains/%s/tracking", p.s.Domain), nil, &out); err != nil {
		return providers.Tracking{}, p.classify(err, p.s.Domain)
	}
	t := providers.Tracking{}
	t.Scope, t.ScopeText = say(mTrackingScope, i18n.Args{"domain": p.s.Domain})
	opens, clicks, unsub := active(out.Tracking.Open.Active), active(out.Tracking.Click.Active), active(out.Tracking.Unsubscribe.Active)
	t.LinkRewrite = unsub
	// The domain's open and click tracking that is on: open, click or both.
	on := ""
	switch {
	case opens && clicks:
		on = "both"
	case opens:
		on = "open"
	case clicks:
		on = "click"
	}
	switch {
	case unsub:
		t.Detail, t.DetailText = say(mTrackingUnsub, i18n.Args{"on": on})
	case on != "":
		t.Detail, t.DetailText = say(mTrackingOn, i18n.Args{"on": on})
	default:
		t.Detail, t.DetailText = say(mTrackingOff, i18n.Args{"domain": p.s.Domain})
	}
	return t, nil
}

func active(raw json.RawMessage) bool {
	switch strings.Trim(strings.ToLower(strings.TrimSpace(string(raw))), `"`) {
	case "true", "yes", "htmlonly":
		return true
	}
	return false
}

// DisableTracking implements providers.TrackingManager: the domain's open,
// click and unsubscribe tracking off.
func (p *Provider) DisableTracking(ctx context.Context) error {
	for _, kind := range []string{"open", "click", "unsubscribe"} {
		ct, body := form([][2]string{{"active", "false"}})
		if _, err := p.api.Exec(ctx, apiclient.Request{Op: "PUT /v3/domains/{domain}/tracking/" + kind, Method: http.MethodPut,
			Path: apiclient.Path("/v3/domains/%s/tracking/", p.s.Domain) + kind, ContentType: ct, Body: body}); err != nil {
			return p.classify(err, p.s.Domain)
		}
	}
	return nil
}

// Backfill implements providers.EventBackfiller from the events API: the
// latest event of each message. Messages still in flight give nothing.
func (p *Provider) Backfill(ctx context.Context, ids []string) ([]providers.Event, error) {
	out := []providers.Event{}
	for _, id := range ids {
		q := url.Values{"message-id": {strings.Trim(id, "<>")}, "ascending": {"no"}, "limit": {"25"}}
		var page struct {
			Items []json.RawMessage `json:"items"`
		}
		if err := p.api.Do(ctx, "GET /v3/{domain}/events", http.MethodGet, apiclient.Path("/v3/%s/events", p.s.Domain)+"?"+q.Encode(), nil, &page); err != nil {
			return out, p.classify(err, p.s.Domain)
		}
		for _, raw := range page.Items {
			var ed eventData
			if json.Unmarshal(raw, &ed) != nil {
				continue
			}
			if ed.Event == "opened" || ed.Event == "clicked" {
				ed.Event = "delivered"
			}
			ev, ok := normalize(ed)
			if !ok || ev.Type == providers.EventUnsubscribe {
				continue
			}
			if ev.ProviderMessageID == "" {
				ev.ProviderMessageID = strings.Trim(id, "<>")
			}
			ev.Raw = append(json.RawMessage(nil), raw...)
			out = append(out, ev)
			break
		}
	}
	return out, nil
}

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "Mailgun status", "https://status.mailgun.com"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
