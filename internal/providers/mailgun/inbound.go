package mailgun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/received"
)

// Receiving mail (M58). Mailgun receives mail for a domain of the account
// whose MX points at it and runs the account's routes: ReputeCast adds one
// route that forwards every message for the route's domain, as raw MIME,
// to the inbound hook and stops
// (https://documentation.mailgun.com/docs/mailgun/user-manual/receive-forward-store/receive-http).
// A forward is signed like a webhook: timestamp, token and signature
// fields, with the account's HTTP webhook signing key.

// MX hosts of Mailgun's receiving, per region.
var (
	usMX = []string{"mxa.mailgun.org", "mxb.mailgun.org"}
	euMX = []string{"mxa.eu.mailgun.org", "mxb.eu.mailgun.org"}
)

var (
	_ providers.InboundReceiver  = (*Provider)(nil)
	_ providers.InboundRegistrar = (*Provider)(nil)
)

func (p *Provider) eu() bool {
	return strings.EqualFold(p.s.Region, "eu") || strings.HasPrefix(p.api.BaseURL, EUBaseURL)
}

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	hosts := usMX
	if p.eu() {
		hosts = euMX
	}
	setup := providers.InboundSetup{Automatic: true}
	for _, h := range hosts {
		setup.MX = append(setup.MX, providers.DNSRecord{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: "10 " + h})
	}
	setup.Steps = append(setup.Steps, mInboundDomain.Keep(i18n.Args{"domain": route.Domain}))
	return setup
}

// routeDescription names the Mailgun route of an inbound route, which is how
// UnregisterInbound finds it.
func routeDescription(route providers.InboundRoute) string { return "ReputeCast " + route.ID.String() }

type mailgunRoute struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Expression  string   `json:"expression"`
	Actions     []string `json:"actions"`
}

// routes lists the account's routes (Mailgun lists at most 1000 a page).
func (p *Provider) routes(ctx context.Context) ([]mailgunRoute, error) {
	var out []mailgunRoute
	for skip := 0; ; skip += 1000 {
		var page struct {
			TotalCount int            `json:"total_count"`
			Items      []mailgunRoute `json:"items"`
		}
		path := fmt.Sprintf("/v3/routes?skip=%d&limit=1000", skip)
		if err := p.api.Do(ctx, "GET /v3/routes", http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Items...)
		if len(page.Items) < 1000 || len(out) >= page.TotalCount {
			return out, nil
		}
	}
}

// RegisterInbound implements providers.InboundRegistrar: one route,
// match_recipient(".*@<domain>") → forward("<url>/mime"), stop(), described
// "ReputeCast <route id>". A route already there for the inbound route is
// updated.
func (p *Provider) RegisterInbound(ctx context.Context, route providers.InboundRoute, hookURL string) error {
	fields := [][2]string{
		{"priority", "0"},
		{"description", routeDescription(route)},
		{"expression", fmt.Sprintf(`match_recipient(".*@%s")`, regexp.QuoteMeta(strings.ToLower(route.Domain)))},
		{"action", fmt.Sprintf(`forward("%s/mime")`, strings.TrimRight(hookURL, "/"))},
		{"action", "stop()"},
	}
	list, err := p.routes(ctx)
	if err != nil {
		return err
	}
	ct, body := form(fields)
	for _, r := range list {
		if r.Description == routeDescription(route) {
			_, err := p.api.Exec(ctx, apiclient.Request{Op: "PUT /v3/routes/{id}", Method: http.MethodPut, Path: apiclient.Path("/v3/routes/%s", r.ID), ContentType: ct, Body: body})
			return err
		}
	}
	_, err = p.api.Exec(ctx, apiclient.Request{Op: "POST /v3/routes", Method: http.MethodPost, Path: "/v3/routes", ContentType: ct, Body: body})
	return err
}

// UnregisterInbound implements providers.InboundRegistrar: DELETE
// /v3/routes/{id} for the route RegisterInbound described; one already
// gone counts as removed.
func (p *Provider) UnregisterInbound(ctx context.Context, route providers.InboundRoute, _ string) error {
	list, err := p.routes(ctx)
	if err != nil {
		return err
	}
	for _, r := range list {
		if r.Description != routeDescription(route) {
			continue
		}
		if err := p.api.Do(ctx, "DELETE /v3/routes/{id}", http.MethodDelete, apiclient.Path("/v3/routes/%s", r.ID), nil, nil); err != nil && !apiclient.NotFound(err) {
			return err
		}
	}
	return nil
}

// forwarded reads a route forward's form: url-encoded for a URL ending in
// mime, multipart for others (with attachments). It reads only the body it
// is given.
func forwarded(r *http.Request, body []byte) (url.Values, error) {
	mt, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mt == "multipart/form-data" {
		f, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(int64(len(body)) + 1)
		if err != nil {
			return nil, fmt.Errorf("mailgun forward: %w", err)
		}
		defer func() { _ = f.RemoveAll() }()
		return url.Values(f.Value), nil
	}
	v, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("mailgun forward: %w", err)
	}
	return v, nil
}

// VerifyInbound implements providers.InboundReceiver: the forward's
// signature, recent (Window), and its token tied to the message it first
// came with, as webhooks are checked (VerifyWebhook).
func (p *Provider) VerifyInbound(ctx context.Context, _ providers.InboundRoute, r *http.Request, body []byte) error {
	f, err := forwarded(r, body)
	if err != nil {
		return fmt.Errorf("%w: %w", providers.ErrBadSignature, err)
	}
	token := f.Get("token")
	if err := p.checkSignature(ctx, f.Get("timestamp"), token, f.Get("signature")); err != nil {
		return err
	}
	return replays.Check(ctx, p.cfg.Tokens, "mailgun:inbound", token, []byte(f.Get("recipient")+"\x00"+f.Get("body-mime")), now())
}

// ParseInbound implements providers.InboundReceiver: body-mime is the
// message as received, recipient the envelope's RCPT TO, and the spam
// filter's MIME headers (when the domain's spam filter marks spam) the
// verdicts.
func (p *Provider) ParseInbound(_ context.Context, _ providers.InboundRoute, r *http.Request, body []byte) ([]providers.InboundMessage, error) {
	f, err := forwarded(r, body)
	if err != nil {
		return nil, err
	}
	raw := f.Get("body-mime")
	if raw == "" {
		return nil, fmt.Errorf("mailgun forward: no body-mime field; the route must forward to a URL ending in mime")
	}
	m := providers.InboundMessage{Raw: []byte(raw), Recipients: []string{}}
	for _, a := range strings.Split(f.Get("recipient"), ",") {
		if a = strings.ToLower(strings.TrimSpace(a)); a != "" {
			m.Recipients = append(m.Recipients, a)
		}
	}
	h := messageHeaders(f.Get("message-headers"))
	if len(h) == 0 {
		h = received.RawHeaders(m.Raw)
	}
	switch strings.ToLower(received.Header(h, "X-Mailgun-Sflag")) {
	case "yes":
		m.Spam = "spam"
	case "no":
		m.Spam = "pass"
	}
	m.SPF = received.Verdict(received.Header(h, "X-Mailgun-Spf"))
	m.DKIM = received.Verdict(received.Header(h, "X-Mailgun-Dkim-Check-Result"))
	if ts := f.Get("timestamp"); ts != "" {
		var sec float64
		if _, err := fmt.Sscan(ts, &sec); err == nil {
			m.ReceivedAt = floatTime(sec)
		}
	}
	return []providers.InboundMessage{m}, nil
}

// messageHeaders reads message-headers, the message's headers as a JSON
// list of [name, value] pairs.
func messageHeaders(s string) map[string][]string {
	var pairs [][]string
	if json.Unmarshal([]byte(s), &pairs) != nil {
		return nil
	}
	h := map[string][]string{}
	for _, kv := range pairs {
		if len(kv) == 2 {
			h[kv[0]] = append(h[kv[0]], kv[1])
		}
	}
	return h
}
