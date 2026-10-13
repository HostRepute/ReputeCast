package mailjet

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
	"github.com/hostrepute/reputecast/internal/providers/bounce"
)

var (
	_ providers.Verifier           = (*Provider)(nil)
	_ providers.QuotaReporter      = (*Provider)(nil)
	_ providers.DomainManager      = (*Provider)(nil)
	_ providers.WebhookRegistrar   = (*Provider)(nil)
	_ providers.WebhookInspector   = (*Provider)(nil)
	_ providers.WebhookVerifier    = (*Provider)(nil)
	_ providers.SuppressionSyncer  = (*Provider)(nil)
	_ providers.TrackingManager    = (*Provider)(nil)
	_ providers.EventBackfiller    = (*Provider)(nil)
	_ providers.ReputationReporter = (*Provider)(nil)
	_ providers.StatusPage         = (*Provider)(nil)
)

// list decodes a REST API envelope {"Count", "Data", "Total"}.
func (p *Provider) list(ctx context.Context, op, path string, out any) error {
	var env struct {
		Data json.RawMessage `json:"Data"`
	}
	if err := p.api.Do(ctx, op, http.MethodGet, path, nil, &env); err != nil {
		return err
	}
	if len(env.Data) == 0 {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}

type counters struct {
	MessageSentCount        float64
	MessageHardBouncedCount float64
	MessageSoftBouncedCount float64
	MessageSpamCount        float64
}

// stats sums the API key's message counters per day since from.
func (p *Provider) stats(ctx context.Context, from time.Time) (counters, error) {
	q := url.Values{"CounterSource": {"APIKey"}, "CounterTiming": {"Message"}, "CounterResolution": {"Day"},
		"FromTS": {strconv.FormatInt(from.Unix(), 10)}, "ToTS": {strconv.FormatInt(time.Now().Unix(), 10)}}
	var days []counters
	if err := p.list(ctx, "GET /v3/REST/statcounters", "/v3/REST/statcounters?"+q.Encode(), &days); err != nil {
		return counters{}, err
	}
	var sum counters
	for _, d := range days {
		sum.MessageSentCount += d.MessageSentCount
		sum.MessageHardBouncedCount += d.MessageHardBouncedCount
		sum.MessageSoftBouncedCount += d.MessageSoftBouncedCount
		sum.MessageSpamCount += d.MessageSpamCount
	}
	return sum, nil
}

// Verify implements providers.Verifier.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) { return p.Quota(ctx) }

// Quota implements providers.QuotaReporter: Mailjet has no plan API, so the
// allowance is the one entered at connect time, against the messages the
// key sent this month; the key's state (active, locked, quarantined) is the
// account status.
func (p *Provider) Quota(ctx context.Context) (providers.Quota, error) {
	var keys []struct {
		APIKey          string `json:"APIKey"`
		IsActive        bool   `json:"IsActive"`
		Runlevel        string `json:"Runlevel"`
		QuarantineValue int64  `json:"QuarantineValue"`
	}
	if err := p.list(ctx, "GET /v3/REST/apikey", "/v3/REST/apikey", &keys); err != nil {
		return providers.Quota{}, err
	}
	q := providers.Quota{Account: "API key active"}
	for _, k := range keys {
		if k.APIKey != strings.TrimSpace(p.c.APIKey) {
			continue
		}
		switch {
		case !k.IsActive:
			q.Account = "API key inactive"
		case k.QuarantineValue > 0:
			q.Account = "API key in quarantine · " + strconv.FormatInt(k.QuarantineValue, 10) + " per hour"
		case k.Runlevel != "" && k.Runlevel != "Normal":
			q.Account = "API key " + strings.ToLower(k.Runlevel)
		}
	}
	now := time.Now().UTC()
	if c, err := p.stats(ctx, time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)); err == nil {
		q.Kind, q.Used, q.Max = "monthly_plan", int64(c.MessageSentCount), int64(p.s.MonthlyLimit)
	}
	return q, nil
}

// Reputation implements providers.ReputationReporter from the key's
// counters over the last seven days: bounces and spam reports per sent
// message, as Mailjet's limits count them.
func (p *Provider) Reputation(ctx context.Context) (providers.Reputation, error) {
	c, err := p.stats(ctx, time.Now().UTC().AddDate(0, 0, -7))
	if err != nil {
		return providers.Reputation{}, err
	}
	r := providers.Reputation{Window: "7 d"}
	if c.MessageSentCount > 0 {
		r.Bounce = (c.MessageHardBouncedCount + c.MessageSoftBouncedCount) / c.MessageSentCount * 100
		r.Complaint = c.MessageSpamCount / c.MessageSentCount * 100
	}
	return r, nil
}

type sender struct {
	ID     int64  `json:"ID"`
	Email  string `json:"Email"`
	Status string `json:"Status"`
}

func (p *Provider) domainSender(ctx context.Context, domain string) (*sender, error) {
	var senders []sender
	if err := p.list(ctx, "GET /v3/REST/sender", "/v3/REST/sender?Limit=1000", &senders); err != nil {
		return nil, err
	}
	for i := range senders {
		if strings.EqualFold(senders[i].Email, "*@"+domain) {
			return &senders[i], nil
		}
	}
	return nil, nil
}

// AddDomain implements providers.DomainManager: a domain sender
// (*@domain), which makes Mailjet set up the domain's DNS checks.
func (p *Provider) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	existing, err := p.domainSender(ctx, domain)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		body := map[string]any{"Email": "*@" + domain, "EmailType": "bulk", "Name": domain}
		if err := p.api.Do(ctx, "POST /v3/REST/sender", http.MethodPost, "/v3/REST/sender", body, nil); err != nil {
			return nil, err
		}
	}
	return p.DomainStatus(ctx, domain)
}

type dnsInfo struct {
	ID                       int64  `json:"ID"`
	DKIMRecordName           string `json:"DKIMRecordName"`
	DKIMRecordValue          string `json:"DKIMRecordValue"`
	DKIMStatus               string `json:"DKIMStatus"`
	SPFRecordValue           string `json:"SPFRecordValue"`
	SPFStatus                string `json:"SPFStatus"`
	OwnerShipToken           string `json:"OwnerShipToken"`
	OwnerShipTokenRecordName string `json:"OwnerShipTokenRecordName"`
}

// DomainStatus implements providers.DomainManager. DNS checks that aren't
// OK run again, and an inactive domain sender is validated.
func (p *Provider) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	var infos []dnsInfo
	if err := p.list(ctx, "GET /v3/REST/dns/{domain}", apiclient.Path("/v3/REST/dns/%s", domain), &infos); err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, &providers.CallError{Op: "GET /v3/REST/dns/{domain}", Status: http.StatusNotFound, Message: domain + " is not set up in Mailjet"}
	}
	d := infos[0]
	if d.DKIMStatus != "OK" || d.SPFStatus != "OK" {
		var checks []struct {
			DKIMStatus string `json:"DKIMStatus"`
			SPFStatus  string `json:"SPFStatus"`
		}
		var env struct {
			Data json.RawMessage `json:"Data"`
		}
		if err := p.api.Do(ctx, "POST /v3/REST/dns/{id}/check", http.MethodPost, "/v3/REST/dns/"+strconv.FormatInt(d.ID, 10)+"/check", struct{}{}, &env); err == nil {
			if json.Unmarshal(env.Data, &checks) == nil && len(checks) > 0 {
				d.DKIMStatus, d.SPFStatus = cmp.Or(checks[0].DKIMStatus, d.DKIMStatus), cmp.Or(checks[0].SPFStatus, d.SPFStatus)
			}
		}
	}
	owner := "pending"
	if s, err := p.domainSender(ctx, domain); err == nil && s != nil {
		if s.Status != "Active" {
			var env struct {
				Data []struct {
					ValidationMethod string `json:"ValidationMethod"`
					GlobalError      string `json:"GlobalError"`
				} `json:"Data"`
			}
			if err := p.api.Do(ctx, "POST /v3/REST/sender/{id}/validate", http.MethodPost, "/v3/REST/sender/"+strconv.FormatInt(s.ID, 10)+"/validate", struct{}{}, &env); err == nil &&
				len(env.Data) > 0 && env.Data[0].GlobalError == "" && env.Data[0].ValidationMethod != "" {
				s.Status = "Active"
			}
		}
		if s.Status == "Active" {
			owner = "verified"
		}
	}
	out := []providers.DNSRecord{}
	if d.OwnerShipTokenRecordName != "" {
		out = append(out, providers.DNSRecord{Purpose: "verification", Type: "TXT", Name: strings.TrimSuffix(d.OwnerShipTokenRecordName, "."), Value: d.OwnerShipToken, Status: owner})
	}
	if d.DKIMRecordName != "" {
		out = append(out, providers.DNSRecord{Purpose: "dkim", Type: "TXT", Name: strings.TrimSuffix(d.DKIMRecordName, "."), Value: d.DKIMRecordValue, Status: checkStatus(d.DKIMStatus)})
	}
	if d.SPFRecordValue != "" {
		out = append(out, providers.DNSRecord{Purpose: "spf", Type: "TXT", Name: domain, Value: d.SPFRecordValue, Status: checkStatus(d.SPFStatus)})
	}
	return out, nil
}

func checkStatus(s string) string {
	switch s {
	case "OK":
		return "verified"
	case "Error":
		return "wrong_value"
	case "Not found":
		return "missing"
	}
	return "pending"
}

// webhookEvents are the event types ReputeCast subscribes to (one callback
// URL each); opens and clicks are tracked by ReputeCast itself.
var webhookEvents = []string{"sent", "bounce", "blocked", "spam", "unsub"}

type callback struct {
	ID        int64  `json:"ID"`
	EventType string `json:"EventType"`
	URL       string `json:"Url"`
	IsBackup  bool   `json:"IsBackup"`
	Status    string `json:"Status"`
}

func (p *Provider) callbacks(ctx context.Context) ([]callback, error) {
	var cbs []callback
	err := p.list(ctx, "GET /v3/REST/eventcallbackurl", "/v3/REST/eventcallbackurl?Limit=100", &cbs)
	return cbs, err
}

// withoutUser drops credentials from a URL so callback URLs compare by
// address.
func withoutUser(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.User = nil
	return strings.TrimRight(u.String(), "/")
}

// RegisterWebhook implements providers.WebhookRegistrar: one callback per
// event type, grouped (Version 2), on a URL that carries basic auth
// credentials. Mailjet allows one callback URL per event type and API key;
// one already pointing elsewhere is left alone and reported.
func (p *Provider) RegisterWebhook(ctx context.Context, hookURL string) error {
	u, err := url.Parse(hookURL)
	if err != nil {
		return err
	}
	u.User = url.UserPassword(WebhookUser, p.cfg.WebhookSecret)
	target := u.String()
	cbs, err := p.callbacks(ctx)
	if err != nil {
		return err
	}
	var taken []string
	for _, typ := range webhookEvents {
		var existing *callback
		for i := range cbs {
			if cbs[i].EventType == typ && !cbs[i].IsBackup {
				existing = &cbs[i]
			}
		}
		switch {
		case existing == nil:
			body := map[string]any{"EventType": typ, "Url": target, "Version": 2, "IsBackup": false, "Status": "alive"}
			err = p.api.Do(ctx, "POST /v3/REST/eventcallbackurl", http.MethodPost, "/v3/REST/eventcallbackurl", body, nil)
		case withoutUser(existing.URL) == withoutUser(hookURL):
			body := map[string]any{"Url": target, "Version": 2, "Status": "alive"}
			err = p.api.Do(ctx, "PUT /v3/REST/eventcallbackurl/{id}", http.MethodPut, "/v3/REST/eventcallbackurl/"+strconv.FormatInt(existing.ID, 10), body, nil)
		default:
			taken = append(taken, typ)
			continue
		}
		if err != nil {
			var ce *providers.CallError
			if errors.As(err, &ce) && ce.Status == http.StatusNotModified {
				continue
			}
			return err
		}
	}
	if len(taken) > 0 {
		return &providers.CallError{Op: "POST /v3/REST/eventcallbackurl", Code: "callback_taken",
			Message: "Mailjet allows one callback URL per event and API key, and " + strings.Join(taken, ", ") + " already post to another URL. Use an API key only ReputeCast uses, or remove those callbacks in Mailjet, then register again."}
	}
	return nil
}

// InspectWebhook implements providers.WebhookInspector: registered when the
// sent and bounce callbacks point at the URL and are alive.
func (p *Provider) InspectWebhook(ctx context.Context, hookURL string) (providers.WebhookStatus, error) {
	st := providers.WebhookStatus{Queued: -1, Sent: -1, URL: hookURL}
	cbs, err := p.callbacks(ctx)
	if err != nil {
		return st, err
	}
	var ids []string
	ok := map[string]bool{}
	for _, c := range cbs {
		if !c.IsBackup && withoutUser(c.URL) == withoutUser(hookURL) {
			ids = append(ids, strconv.FormatInt(c.ID, 10))
			ok[c.EventType] = c.Status == "alive"
		}
	}
	st.ID = strings.Join(ids, ",")
	st.Registered = ok["sent"] && ok["bounce"]
	return st, nil
}

// ListSuppressions implements providers.SuppressionSyncer with the
// contacts excluded from campaigns: spam complainers, unsubscribed
// contacts and manual exclusions.
func (p *Provider) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	out := []providers.Suppression{}
	for offset := 0; offset < 1_000_000; offset += 1000 {
		var contacts []struct {
			Email              string `json:"Email"`
			IsSpamComplaining  bool   `json:"IsSpamComplaining"`
			UnsubscribedAt     string `json:"UnsubscribedAt"`
			ExclusionUpdatedAt string `json:"ExclusionFromCampaignsUpdatedAt"`
		}
		path := "/v3/REST/contact?IsExcludedFromCampaigns=true&Limit=1000&Offset=" + strconv.Itoa(offset)
		if err := p.list(ctx, "GET /v3/REST/contact", path, &contacts); err != nil {
			return nil, err
		}
		for _, c := range contacts {
			at := parseTime(c.ExclusionUpdatedAt)
			if !since.IsZero() && !at.IsZero() && at.Before(since) {
				continue
			}
			reason := "manual"
			switch {
			case c.IsSpamComplaining:
				reason = "complaint"
			case c.UnsubscribedAt != "":
				reason = "unsubscribe"
			}
			out = append(out, providers.Suppression{Email: strings.ToLower(strings.TrimSpace(c.Email)), Reason: reason, Code: "IsExcludedFromCampaigns", CreatedAt: at})
		}
		if len(contacts) < 1000 {
			break
		}
	}
	return out, nil
}

// AddSuppressions implements providers.SuppressionSyncer: the addresses are
// excluded from Mailjet's campaigns (an asynchronous job, 500 per call).
func (p *Provider) AddSuppressions(ctx context.Context, emails []string) error {
	for chunk := range slices.Chunk(emails, 500) {
		contacts := make([]map[string]any, len(chunk))
		for i, e := range chunk {
			contacts[i] = map[string]any{"Email": e, "IsExcludedFromCampaigns": true}
		}
		if err := p.api.Do(ctx, "POST /v3/REST/contact/managemanycontacts", http.MethodPost, "/v3/REST/contact/managemanycontacts", map[string]any{"Contacts": contacts}, nil); err != nil {
			return err
		}
	}
	return nil
}

// Tracking implements providers.TrackingManager. Mailjet's account
// tracking setting has no API, but ReputeCast sends every message with
// TrackOpens and TrackClicks disabled, so Mailjet never tracks its mail.
func (p *Provider) Tracking(context.Context) (providers.Tracking, error) {
	return providers.Tracking{Scope: "Every message ReputeCast sends",
		Detail: "ReputeCast sends every message with Mailjet's TrackOpens and TrackClicks disabled, whatever the account setting."}, nil
}

// DisableTracking implements providers.TrackingManager: nothing to change,
// tracking is off on every message.
func (p *Provider) DisableTracking(context.Context) error { return nil }

// Backfill implements providers.EventBackfiller with GET /v3/REST/message:
// each message's current state.
func (p *Provider) Backfill(ctx context.Context, ids []string) ([]providers.Event, error) {
	out := []providers.Event{}
	for _, id := range ids {
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			continue
		}
		var msgs []struct {
			ID             json.Number `json:"ID"`
			Status         string      `json:"Status"`
			StatePermanent bool        `json:"StatePermanent"`
			ArrivedAt      string      `json:"ArrivedAt"`
		}
		var env struct {
			Data json.RawMessage `json:"Data"`
		}
		raw, err := p.api.DoRaw(ctx, "GET /v3/REST/message/{id}", http.MethodGet, "/v3/REST/message/"+id, nil, &env)
		if err != nil {
			var ce *providers.CallError
			if errors.As(err, &ce) && ce.Status == http.StatusNotFound {
				continue
			}
			return out, err
		}
		if json.Unmarshal(env.Data, &msgs) != nil || len(msgs) == 0 {
			continue
		}
		m := msgs[0]
		ev := providers.Event{ProviderMessageID: id, OriginalType: "message · " + m.Status, OccurredAt: parseTime(m.ArrivedAt), Raw: raw}
		switch m.Status {
		case "sent", "opened", "clicked", "unsub":
			ev.Type = providers.EventDelivered
		case "hardbounced":
			ev.Type, ev.BounceClass = providers.EventHardBounce, bounce.Other
		case "softbounced", "bounce":
			ev.Type, ev.BounceClass = providers.EventSoftBounce, bounce.Other
		case "spam":
			ev.Type = providers.EventComplaint
		case "blocked":
			ev.Type, ev.BounceClass = providers.EventRejected, bounce.Other
		case "deferred":
			ev.Type, ev.BounceClass = providers.EventDeferred, bounce.Temporary
		default:
			continue // queued, unknown: still in flight
		}
		if ev.OccurredAt.IsZero() {
			ev.OccurredAt = time.Now().UTC()
		}
		out = append(out, ev)
	}
	return out, nil
}

func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "Mailjet status", "https://status.mailjet.com"
}
