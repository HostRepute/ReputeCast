package postmark

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/received"
)

// Receiving mail (M58). Postmark receives mail for the server's inbound
// domain (its MX points at inbound.postmarkapp.com) and posts each message
// as JSON to the server's inbound webhook
// (https://postmarkapp.com/developer/user-guide/inbound/inbound-domain-forwarding,
// https://postmarkapp.com/developer/webhooks/inbound-webhook). ReputeCast
// sets both on the server, with the raw message included and the hook's
// basic authentication in the URL.

// InboundMX is Postmark's inbound mail server.
const InboundMX = "inbound.postmarkapp.com"

var (
	_ providers.InboundReceiver  = (*Provider)(nil)
	_ providers.InboundRegistrar = (*Provider)(nil)
)

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	return providers.InboundSetup{Automatic: true,
		MX: []providers.DNSRecord{{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: "10 " + InboundMX}}}
}

type inboundServer struct {
	InboundHookURL string `json:"InboundHookUrl"`
	InboundDomain  string
}

// RegisterInbound implements providers.InboundRegistrar: PUT /server with
// the inbound webhook, the inbound domain and RawEmailEnabled
// (https://postmarkapp.com/developer/api/server-api#edit-server).
func (p *Provider) RegisterInbound(ctx context.Context, route providers.InboundRoute, hookURL string) error {
	target, err := received.HookURL(hookURL, route.Secret)
	if err != nil {
		return err
	}
	body := map[string]any{"InboundHookUrl": target, "InboundDomain": route.Domain, "RawEmailEnabled": true}
	return p.api.Do(ctx, "PUT /server", http.MethodPut, "/server", body, nil)
}

// UnregisterInbound implements providers.InboundRegistrar: the server's
// inbound webhook and domain are cleared when they are still ReputeCast's.
func (p *Provider) UnregisterInbound(ctx context.Context, route providers.InboundRoute, hookURL string) error {
	var s inboundServer
	if err := p.api.Do(ctx, "GET /server", http.MethodGet, "/server", nil, &s); err != nil {
		return err
	}
	body := map[string]any{}
	if s.InboundHookURL != "" && received.SameURL(s.InboundHookURL, hookURL) {
		body["InboundHookUrl"] = ""
	}
	if s.InboundDomain != "" && strings.EqualFold(s.InboundDomain, route.Domain) {
		body["InboundDomain"] = ""
	}
	if len(body) == 0 {
		return nil
	}
	return p.api.Do(ctx, "PUT /server", http.MethodPut, "/server", body, nil)
}

// VerifyInbound implements providers.InboundReceiver: the hook's basic
// authentication.
func (p *Provider) VerifyInbound(_ context.Context, route providers.InboundRoute, r *http.Request, _ []byte) error {
	return received.CheckBasic(r, route)
}

type inboundAddress struct {
	Email string
	Name  string
}

// inboundMessage is Postmark's inbound webhook payload.
type inboundMessage struct {
	From              string
	FromFull          inboundAddress
	ToFull            []inboundAddress
	OriginalRecipient string
	Subject           string
	MessageID         string
	TextBody          string
	HTMLBody          string `json:"HtmlBody"`
	Headers           []header
	Attachments       []struct {
		Name          string
		Content       string
		ContentType   string
		ContentLength int64
	}
	RawEmail string
}

// ParseInbound implements providers.InboundReceiver: RawEmail when the
// server includes it, else the parsed fields.
func (p *Provider) ParseInbound(_ context.Context, _ providers.InboundRoute, _ *http.Request, body []byte) ([]providers.InboundMessage, error) {
	var in inboundMessage
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, fmt.Errorf("postmark inbound: %w", err)
	}
	headers := map[string][]string{}
	for _, h := range in.Headers {
		headers[h.Name] = append(headers[h.Name], h.Value)
	}
	m := providers.InboundMessage{ProviderMessageID: in.MessageID, Recipients: []string{}}
	if r := strings.ToLower(strings.TrimSpace(in.OriginalRecipient)); r != "" {
		m.Recipients = append(m.Recipients, r)
	}
	// SpamAssassin's verdict: "No, score=-0.1 …" or "Yes, score=…".
	switch status := strings.ToLower(received.Header(headers, "X-Spam-Status")); {
	case strings.HasPrefix(status, "yes"):
		m.Spam = "spam"
	case strings.HasPrefix(status, "no"):
		m.Spam = "pass"
	}
	if in.RawEmail != "" {
		m.Raw = []byte(in.RawEmail)
		return []providers.InboundMessage{m}, nil
	}
	m.From = received.Address(in.FromFull.Name, in.FromFull.Email)
	if m.From == "" {
		m.From = in.From
	}
	m.To = []string{}
	for _, a := range in.ToFull {
		m.To = append(m.To, received.Address(a.Name, a.Email))
	}
	m.Subject, m.Text, m.HTML, m.Headers = in.Subject, in.TextBody, in.HTMLBody, headers
	m.MessageID = received.Header(headers, "Message-ID")
	m.InReplyTo = received.Header(headers, "In-Reply-To")
	m.References = received.MessageIDs(received.Header(headers, "References"))
	for _, a := range in.Attachments {
		data, err := received.Base64(a.Content)
		if err != nil {
			return nil, fmt.Errorf("postmark inbound: attachment %q: %w", a.Name, err)
		}
		size := a.ContentLength
		if size == 0 {
			size = int64(len(data))
		}
		m.Attachments = append(m.Attachments, providers.InboundAttachment{Name: a.Name, ContentType: a.ContentType, Size: size, Data: data})
	}
	return []providers.InboundMessage{m}, nil
}
