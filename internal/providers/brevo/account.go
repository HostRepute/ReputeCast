package brevo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

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
	_ providers.WebhookInspector   = (*Provider)(nil)
	_ providers.WebhookUnregistrar = (*Provider)(nil)
	_ providers.WebhookVerifier    = (*Provider)(nil)
	_ providers.SuppressionSyncer  = (*Provider)(nil)
	_ providers.EventBackfiller    = (*Provider)(nil)
	_ providers.ReputationReporter = (*Provider)(nil)
	_ providers.StatusPage         = (*Provider)(nil)
)

// Verify implements providers.Verifier.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) { return p.Quota(ctx) }

// Quota implements providers.QuotaReporter from GET /v3/account: the email
// credits left on the plan. A subscription plan with a monthly allowance
// entered at connect time shows as used against it; the free plan's daily
// allowance and pay-as-you-go credits show as what is left.
func (p *Provider) Quota(ctx context.Context) (providers.Quota, error) {
	var acct struct {
		CompanyName string `json:"companyName"`
		Plan        []struct {
			Type        string          `json:"type"`
			CreditsType string          `json:"creditsType"`
			Credits     json.RawMessage `json:"credits"`
		} `json:"plan"`
	}
	if err := p.api.Do(ctx, "GET /v3/account", http.MethodGet, "/v3/account", nil, &acct); err != nil {
		return providers.Quota{}, classify(err)
	}
	q := providers.Quota{Account: "Brevo account"}
	if acct.CompanyName != "" {
		q.Account = acct.CompanyName
	}
	// "Acme · free plan · 212 emails left today": the account's name and its
	// email plan, kept as a message when there is one plan to tell.
	account := i18n.Args{"company": "none", "name": acct.CompanyName, "plan": "none"}
	if acct.CompanyName != "" {
		account["company"] = "yes"
	}
	plans := 0
	for _, pl := range acct.Plan {
		if pl.Type == "sms" {
			continue
		}
		credits, err := strconv.ParseFloat(strings.Trim(strings.TrimSpace(string(pl.Credits)), `"`), 64)
		if err != nil {
			continue
		}
		left := int64(credits)
		plans++
		account["n"] = strconv.FormatInt(left, 10)
		switch pl.Type {
		case "free":
			// 300 a day on the free plan.
			q.Account += " · free plan · " + strconv.FormatInt(left, 10) + " emails left"
			account["plan"], account["today"] = "free", "no"
			if left <= 300 {
				q.Kind, q.Max, q.Used = "rolling_24h", 300, 300-left
				q.Account += " today"
				account["today"] = "yes"
			}
		case "subscription":
			if p.s.MonthlyLimit > 0 {
				q.Kind, q.Max, q.Used = "monthly_plan", int64(p.s.MonthlyLimit), max(0, int64(p.s.MonthlyLimit)-left)
			}
			q.Account += " · subscription · " + strconv.FormatInt(left, 10) + " emails left"
			account["plan"] = "subscription"
		default: // payAsYouGo
			q.Account += " · " + strconv.FormatInt(left, 10) + " prepaid credits left"
			account["plan"] = "credits"
		}
	}
	if plans <= 1 {
		q.Account, q.AccountText = say(mAccount, account)
	}
	return q, nil
}

// Reputation implements providers.ReputationReporter from the transactional
// statistics of the last seven days: bounces per request and spam reports
// per delivered message.
func (p *Provider) Reputation(ctx context.Context) (providers.Reputation, error) {
	var st struct {
		Requests    float64 `json:"requests"`
		Delivered   float64 `json:"delivered"`
		HardBounces float64 `json:"hardBounces"`
		SoftBounces float64 `json:"softBounces"`
		SpamReports float64 `json:"spamReports"`
	}
	if err := p.api.Do(ctx, "GET /v3/smtp/statistics/aggregatedReport", http.MethodGet, "/v3/smtp/statistics/aggregatedReport?days=7", nil, &st); err != nil {
		return providers.Reputation{}, classify(err)
	}
	r := providers.Reputation{Window: "7 d"}
	if st.Requests > 0 {
		r.Bounce = (st.HardBounces + st.SoftBounces) / st.Requests * 100
	}
	if st.Delivered > 0 {
		r.Complaint = st.SpamReports / st.Delivered * 100
	}
	return r, nil
}

type dnsRecord struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	HostName string `json:"host_name"`
	Status   bool   `json:"status"`
}

type domainInfo struct {
	Verified      bool                 `json:"verified"`
	Authenticated bool                 `json:"authenticated"`
	DNSRecords    map[string]dnsRecord `json:"dns_records"`
}

// AddDomain implements providers.DomainManager. A domain already in the
// account is read instead.
func (p *Provider) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	var created domainInfo
	err := p.api.Do(ctx, "POST /v3/senders/domains", http.MethodPost, "/v3/senders/domains", map[string]string{"name": domain}, &created)
	if err != nil {
		var ce *providers.CallError
		if errors.As(err, &ce) && ce.Status == http.StatusBadRequest && strings.Contains(strings.ToLower(ce.Message), "already") {
			return p.DomainStatus(ctx, domain)
		}
		return nil, classify(err)
	}
	return records(created, domain), nil
}

// DomainStatus implements providers.DomainManager. A domain not
// authenticated yet is checked again first.
func (p *Provider) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	path := apiclient.Path("/v3/senders/domains/%s", domain)
	var d domainInfo
	if err := p.api.Do(ctx, "GET /v3/senders/domains/{domain}", http.MethodGet, path, nil, &d); err != nil {
		return nil, classify(err)
	}
	if !d.Authenticated {
		if err := p.api.Do(ctx, "PUT /v3/senders/domains/{domain}/authenticate", http.MethodPut, path+"/authenticate", nil, nil); err == nil {
			var again domainInfo
			if err := p.api.Do(ctx, "GET /v3/senders/domains/{domain}", http.MethodGet, path, nil, &again); err == nil {
				d = again
			}
		}
	}
	return records(d, domain), nil
}

// records lists the domain's records: the Brevo code (ownership), DKIM and
// DMARC.
func records(d domainInfo, domain string) []providers.DNSRecord {
	keys := make([]string, 0, len(d.DNSRecords))
	for k := range d.DNSRecords {
		keys = append(keys, k)
	}
	rank := func(k string) int {
		switch {
		case strings.Contains(k, "brevo"):
			return 0
		case strings.Contains(k, "dkim"):
			return 1
		case strings.Contains(k, "spf"):
			return 2
		case strings.Contains(k, "dmarc"):
			return 3
		}
		return 4
	}
	sort.Slice(keys, func(i, j int) bool {
		if rank(keys[i]) != rank(keys[j]) {
			return rank(keys[i]) < rank(keys[j])
		}
		return keys[i] < keys[j]
	})
	out := []providers.DNSRecord{}
	for _, k := range keys {
		r := d.DNSRecords[k]
		purpose := map[int]string{0: "verification", 1: "dkim", 2: "spf", 3: "dmarc"}[rank(k)]
		if purpose == "" || r.Value == "" {
			continue
		}
		status := "pending"
		if r.Status {
			status = "verified"
		}
		out = append(out, providers.DNSRecord{Purpose: purpose, Type: strings.ToUpper(r.Type), Name: absolute(r.HostName, domain), Value: r.Value, Status: status})
	}
	return out
}

func absolute(host, domain string) string {
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	switch {
	case host == "" || host == "@":
		return domain
	case strings.EqualFold(host, domain) || strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(domain)):
		return host
	}
	return host + "." + domain
}

// webhookEvents are the transactional events ReputeCast subscribes to;
// opens and clicks are tracked by ReputeCast itself.
var webhookEvents = []string{"delivered", "hardBounce", "softBounce", "deferred", "blocked", "invalid", "spam", "unsubscribed"}

type webhook struct {
	ID     int64    `json:"id"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Type   string   `json:"type"`
}

func (p *Provider) webhooks(ctx context.Context) ([]webhook, error) {
	var out struct {
		Webhooks []webhook `json:"webhooks"`
	}
	err := p.api.Do(ctx, "GET /v3/webhooks", http.MethodGet, "/v3/webhooks?type=transactional", nil, &out)
	return out.Webhooks, classify(err)
}

func sameURL(a, b string) bool { return strings.TrimRight(a, "/") == strings.TrimRight(b, "/") }

// RegisterWebhook implements providers.WebhookRegistrar: a transactional
// webhook, one event per call, with bearer authentication. An existing
// webhook for the URL is updated.
func (p *Provider) RegisterWebhook(ctx context.Context, hookURL string) error {
	hooks, err := p.webhooks(ctx)
	if err != nil {
		return err
	}
	auth := map[string]string{"type": "bearer", "token": p.cfg.WebhookSecret}
	for _, h := range hooks {
		if sameURL(h.URL, hookURL) {
			body := map[string]any{"url": hookURL, "description": "ReputeCast", "events": webhookEvents, "auth": auth, "batched": false}
			return classify(p.api.Do(ctx, "PUT /v3/webhooks/{id}", http.MethodPut, "/v3/webhooks/"+strconv.FormatInt(h.ID, 10), body, nil))
		}
	}
	body := map[string]any{"url": hookURL, "description": "ReputeCast", "events": webhookEvents, "type": "transactional", "channel": "email", "auth": auth, "batched": false}
	return classify(p.api.Do(ctx, "POST /v3/webhooks", http.MethodPost, "/v3/webhooks", body, nil))
}

// UnregisterWebhook implements providers.WebhookUnregistrar with
// DELETE /v3/webhooks/{webhookId} for each transactional webhook pointing at
// hookURL (https://developers.brevo.com/reference/delete-webhook).
func (p *Provider) UnregisterWebhook(ctx context.Context, hookURL string) error {
	hooks, err := p.webhooks(ctx)
	if err != nil {
		return err
	}
	for _, h := range hooks {
		if !sameURL(h.URL, hookURL) {
			continue
		}
		err = p.api.Do(ctx, "DELETE /v3/webhooks/{id}", http.MethodDelete, "/v3/webhooks/"+strconv.FormatInt(h.ID, 10), nil, nil)
		var ce *providers.CallError
		if err != nil && (!errors.As(err, &ce) || ce.Status != http.StatusNotFound) {
			return classify(err)
		}
	}
	return nil
}

// InspectWebhook implements providers.WebhookInspector.
func (p *Provider) InspectWebhook(ctx context.Context, hookURL string) (providers.WebhookStatus, error) {
	st := providers.WebhookStatus{Queued: -1, Sent: -1}
	hooks, err := p.webhooks(ctx)
	if err != nil {
		return st, err
	}
	for _, h := range hooks {
		if sameURL(h.URL, hookURL) {
			st.Registered, st.ID, st.URL = true, strconv.FormatInt(h.ID, 10), h.URL
		}
	}
	return st, nil
}

// ListSuppressions implements providers.SuppressionSyncer with Brevo's
// blocked transactional contacts.
func (p *Provider) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	out := []providers.Suppression{}
	seen := map[string]bool{}
	for offset := 0; offset < 1_000_000; offset += 100 {
		q := url.Values{"limit": {"100"}, "offset": {strconv.Itoa(offset)}, "sort": {"desc"}}
		if !since.IsZero() {
			q.Set("startDate", since.UTC().Format("2006-01-02"))
			q.Set("endDate", time.Now().UTC().Format("2006-01-02"))
		}
		var page struct {
			Contacts []struct {
				Email  string `json:"email"`
				Reason struct {
					Code string `json:"code"`
				} `json:"reason"`
				BlockedAt string `json:"blockedAt"`
			} `json:"contacts"`
		}
		if err := p.api.Do(ctx, "GET /v3/smtp/blockedContacts", http.MethodGet, "/v3/smtp/blockedContacts?"+q.Encode(), nil, &page); err != nil {
			return nil, classify(err)
		}
		for _, c := range page.Contacts {
			email := strings.ToLower(strings.TrimSpace(c.Email))
			if email == "" || seen[email] {
				continue
			}
			seen[email] = true
			reason := "manual"
			switch c.Reason.Code {
			case "hardBounce":
				reason = "hard_bounce"
			case "contactFlaggedAsSpam":
				reason = "complaint"
			case "unsubscribedViaEmail", "unsubscribedViaMA", "unsubscribedViaApi":
				reason = "unsubscribe"
			}
			at, _ := time.Parse(time.RFC3339, c.BlockedAt)
			out = append(out, providers.Suppression{Email: email, Reason: reason, Code: c.Reason.Code, CreatedAt: at.UTC()})
		}
		if len(page.Contacts) < 100 {
			break
		}
	}
	return out, nil
}

// AddSuppressions implements providers.SuppressionSyncer. Brevo's API can
// list blocked transactional contacts but only block one per contact and
// sender, so pushing isn't supported: ReputeCast enforces the shared list
// before handing mail to Brevo.
func (p *Provider) AddSuppressions(context.Context, []string) error { return providers.ErrNotSupported }

// statEvents maps Brevo's statistics events to the latest outcome of a
// message.
var statEvents = map[string]providers.EventType{
	"delivered": providers.EventDelivered, "opened": providers.EventDelivered, "clicks": providers.EventDelivered,
	"loadedByProxy": providers.EventDelivered, "unsubscribed": providers.EventDelivered,
	"hardBounces": providers.EventHardBounce, "invalid": providers.EventHardBounce,
	"softBounces": providers.EventSoftBounce, "bounces": providers.EventSoftBounce,
	"deferred": providers.EventDeferred, "blocked": providers.EventRejected, "error": providers.EventRejected,
	"spam": providers.EventComplaint,
}

// Backfill implements providers.EventBackfiller from the transactional
// event report (30 days): the latest event of each message. Brevo allows
// few report calls per hour, so a throttled backfill returns what it has
// and the next run continues.
func (p *Provider) Backfill(ctx context.Context, ids []string) ([]providers.Event, error) {
	out := []providers.Event{}
	for _, id := range ids {
		q := url.Values{"messageId": {"<" + strings.Trim(id, "<>") + ">"}, "days": {"30"}, "limit": {"100"}, "sort": {"desc"}}
		var rep struct {
			Events []struct {
				Date      string `json:"date"`
				Email     string `json:"email"`
				Event     string `json:"event"`
				MessageID string `json:"messageId"`
				Reason    string `json:"reason"`
			} `json:"events"`
		}
		raw, err := p.api.DoRaw(ctx, "GET /v3/smtp/statistics/events", http.MethodGet, "/v3/smtp/statistics/events?"+q.Encode(), nil, &rep)
		if err != nil {
			var ce *providers.CallError
			if errors.As(err, &ce) && ce.Throttled {
				return out, nil
			}
			return out, classify(err)
		}
		var best *providers.Event
		for _, e := range rep.Events {
			typ, ok := statEvents[e.Event]
			if !ok {
				continue
			}
			at, _ := time.Parse(time.RFC3339, e.Date)
			if best != nil && !at.After(best.OccurredAt) {
				continue
			}
			ev := providers.Event{Type: typ, ProviderMessageID: strings.Trim(id, "<>"), Recipient: strings.ToLower(e.Email), OccurredAt: at.UTC(),
				OriginalType: "report · " + e.Event, Raw: raw}
			ev.OriginalCode, ev.Diagnostic = bounce.Split(e.Reason)
			switch typ {
			case providers.EventHardBounce, providers.EventSoftBounce, providers.EventDeferred, providers.EventRejected:
				ev.BounceClass = bounce.Classify(ev.OriginalCode, e.Reason)
			}
			best = &ev
		}
		if best != nil {
			out = append(out, *best)
		}
	}
	return out, nil
}

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "Brevo status", "https://status.brevo.com"
}
