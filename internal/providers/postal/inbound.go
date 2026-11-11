package postal

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/received"
)

// Receiving mail (M58). A Postal mail server receives mail for its
// domains (their MX points at the Postal server) and hands each message to
// a route's endpoint: an HTTP endpoint posts it, signed like Postal's
// webhooks (https://docs.postalserver.io/features/routing-incoming-email,
// https://docs.postalserver.io/developer/http-payloads). Postal's API has
// no routes or endpoints, so people add both in Postal (InboundManual):
// a route for every address of the domain (*) to an HTTP endpoint at the
// hook that delivers the raw message.

var _ providers.InboundReceiver = (*Provider)(nil)

// mxHost is the Postal server's host, which its MX records name unless the
// installation lists others (dns.mx_records).
func (p *Provider) mxHost() string {
	u, err := url.Parse(p.s.ServerURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	host := p.mxHost()
	args := i18n.Args{"domain": route.Domain, "url": route.URL, "host": host}
	setup := providers.InboundSetup{Steps: []i18n.Text{mInboundEndpoint.Keep(args), mInboundRoute.Keep(args), mInboundMX.Keep(args)}}
	if host != "" {
		setup.MX = []providers.DNSRecord{{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: "10 " + host}}
	}
	return setup
}

// VerifyInbound implements providers.InboundReceiver: Postal signs an HTTP
// endpoint's posts as it signs webhooks, with the server's key.
func (p *Provider) VerifyInbound(ctx context.Context, _ providers.InboundRoute, r *http.Request, body []byte) error {
	return p.VerifyWebhook(ctx, r, body)
}

// rawPayload is what an HTTP endpoint with the format "Delivered as the
// raw message" posts.
type rawPayload struct {
	ID       json.Number `json:"id"`
	RcptTo   string      `json:"rcpt_to"`
	MailFrom string      `json:"mail_from"`
	Message  string      `json:"message"`
	Base64   bool        `json:"base64"`
}

// ParseInbound implements providers.InboundReceiver: the raw message
// (base64), the envelope recipient and Postal's message id, sent as JSON
// or as form data.
func (p *Provider) ParseInbound(_ context.Context, _ providers.InboundRoute, r *http.Request, body []byte) ([]providers.InboundMessage, error) {
	var in rawPayload
	mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mt == "application/x-www-form-urlencoded" {
		v, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("postal inbound: %w", err)
		}
		b, _ := strconv.ParseBool(v.Get("base64"))
		in = rawPayload{ID: json.Number(v.Get("id")), RcptTo: v.Get("rcpt_to"), MailFrom: v.Get("mail_from"), Message: v.Get("message"), Base64: b}
	} else if err := json.Unmarshal(body, &in); err != nil {
		return nil, fmt.Errorf("postal inbound: %w", err)
	}
	if in.Message == "" {
		return nil, fmt.Errorf("postal inbound: no message; the HTTP endpoint's format must be Delivered as the raw message")
	}
	raw := []byte(in.Message)
	if in.Base64 {
		var err error
		if raw, err = received.Base64(in.Message); err != nil {
			return nil, fmt.Errorf("postal inbound: the message isn't base64: %w", err)
		}
	}
	m := providers.InboundMessage{Raw: raw, Recipients: []string{}, ProviderMessageID: in.ID.String()}
	if rcpt := strings.ToLower(strings.TrimSpace(in.RcptTo)); rcpt != "" {
		m.Recipients = append(m.Recipients, rcpt)
	}
	return []providers.InboundMessage{m}, nil
}
