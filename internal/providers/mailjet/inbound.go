package mailjet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/received"
)

// Receiving mail (M58). Mailjet's Parse API receives mail for a validated
// domain whose MX points at parse.mailjet.com and posts each message,
// parsed into JSON, to the parse route's URL
// (https://dev.mailjet.com/docs/email-api/parse-api/basic-setup,
// https://dev.mailjet.com/docs/email-api/parse-api/use-own-domain). Mailjet
// doesn't sign the posts: the route's URL carries the hook's basic
// authentication, as Mailjet recommends.

// InboundMX is Mailjet's parse server.
const InboundMX = "parse.mailjet.com"

var (
	_ providers.InboundReceiver  = (*Provider)(nil)
	_ providers.InboundRegistrar = (*Provider)(nil)
)

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	return providers.InboundSetup{
		Automatic: true,
		MX:        []providers.DNSRecord{{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: "10 " + InboundMX}},
		Steps:     []i18n.Text{mInboundDomain.Keep(i18n.Args{"domain": route.Domain})},
	}
}

type parseRoute struct {
	ID    int64  `json:"ID"`
	Email string `json:"Email"`
	URL   string `json:"Url"`
}

// catchAll is the parse route's address: every address of the domain.
func catchAll(route providers.InboundRoute) string { return "*@" + strings.ToLower(route.Domain) }

func (p *Provider) parseRoutes(ctx context.Context) ([]parseRoute, error) {
	var out []parseRoute
	err := p.list(ctx, "GET /v3/REST/parseroute", "/v3/REST/parseroute?Limit=1000", &out)
	return out, err
}

// RegisterInbound implements providers.InboundRegistrar: a parse route for
// every address of the domain posting to the hook, with its basic
// authentication in the URL. A route for the domain or the hook is
// updated.
func (p *Provider) RegisterInbound(ctx context.Context, route providers.InboundRoute, hookURL string) error {
	target, err := received.HookURL(hookURL, route.Secret)
	if err != nil {
		return err
	}
	routes, err := p.parseRoutes(ctx)
	if err != nil {
		return err
	}
	body := map[string]any{"Url": target, "Email": catchAll(route)}
	for _, r := range routes {
		if strings.EqualFold(r.Email, catchAll(route)) || received.SameURL(r.URL, hookURL) {
			err := p.api.Do(ctx, "PUT /v3/REST/parseroute/{id}", http.MethodPut, "/v3/REST/parseroute/"+strconv.FormatInt(r.ID, 10), body, nil)
			var ce *providers.CallError
			if errors.As(err, &ce) && ce.Status == http.StatusNotModified {
				return nil
			}
			return err
		}
	}
	return p.api.Do(ctx, "POST /v3/REST/parseroute", http.MethodPost, "/v3/REST/parseroute", body, nil)
}

// UnregisterInbound implements providers.InboundRegistrar: DELETE
// /v3/REST/parseroute/{id} for every parse route posting to the hook; one
// already gone counts as removed.
func (p *Provider) UnregisterInbound(ctx context.Context, _ providers.InboundRoute, hookURL string) error {
	routes, err := p.parseRoutes(ctx)
	if err != nil {
		return err
	}
	for _, r := range routes {
		if !received.SameURL(r.URL, hookURL) {
			continue
		}
		err := p.api.Do(ctx, "DELETE /v3/REST/parseroute/{id}", http.MethodDelete, "/v3/REST/parseroute/"+strconv.FormatInt(r.ID, 10), nil, nil)
		var ce *providers.CallError
		if err != nil && (!errors.As(err, &ce) || ce.Status != http.StatusNotFound) {
			return err
		}
	}
	return nil
}

// VerifyInbound implements providers.InboundReceiver: the hook's basic
// authentication.
func (p *Provider) VerifyInbound(_ context.Context, route providers.InboundRoute, r *http.Request, _ []byte) error {
	return received.CheckBasic(r, route)
}

// headerValues reads a Parse API header: a string, or a list of strings for
// headers that may repeat.
func headerValues(raw json.RawMessage) []string {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		return many
	}
	return nil
}

func headerMap(raw map[string]json.RawMessage) map[string][]string {
	h := make(map[string][]string, len(raw))
	for k, v := range raw {
		h[k] = headerValues(v)
	}
	return h
}

// ParseInbound implements providers.InboundReceiver: the Parse API's
// fields (Sender and Recipient are the envelope's), its parts and their
// base64 attachments, and SpamAssassin's score.
func (p *Provider) ParseInbound(_ context.Context, _ providers.InboundRoute, _ *http.Request, body []byte) ([]providers.InboundMessage, error) {
	var in map[string]json.RawMessage
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, fmt.Errorf("mailjet parse: %w", err)
	}
	str := func(k string) string {
		var s string
		_ = json.Unmarshal(in[k], &s)
		return s
	}
	var rawHeaders map[string]json.RawMessage
	_ = json.Unmarshal(in["Headers"], &rawHeaders)
	headers := headerMap(rawHeaders)
	m := providers.InboundMessage{
		From:       str("From"),
		Subject:    str("Subject"),
		To:         received.Addresses(received.Header(headers, "To")),
		MessageID:  received.Header(headers, "Message-ID"),
		InReplyTo:  received.Header(headers, "In-Reply-To"),
		References: received.MessageIDs(received.Header(headers, "References")),
		Headers:    headers,
		Text:       str("Text-part"),
		HTML:       str("Html-part"),
		Recipients: []string{},
		Spam:       received.SpamScore(str("SpamAssassinScore"), received.SpamAssassin),
	}
	if m.From == "" {
		m.From = received.Header(headers, "From")
	}
	for _, a := range strings.Split(str("Recipient"), ",") {
		if a = received.Bare(a); a != "" {
			m.Recipients = append(m.Recipients, a)
		}
	}
	var parts []struct {
		Headers    map[string]json.RawMessage `json:"Headers"`
		ContentRef string                     `json:"ContentRef"`
	}
	_ = json.Unmarshal(in["Parts"], &parts)
	for _, part := range parts {
		if !strings.HasPrefix(part.ContentRef, "Attachment") {
			continue
		}
		data, err := received.Base64(str(part.ContentRef))
		if err != nil {
			return nil, fmt.Errorf("mailjet parse: %s: %w", part.ContentRef, err)
		}
		ph := headerMap(part.Headers)
		ct, ctParams, _ := mime.ParseMediaType(received.Header(ph, "Content-Type"))
		_, cdParams, _ := mime.ParseMediaType(received.Header(ph, "Content-Disposition"))
		name := cdParams["filename"]
		if name == "" {
			name = ctParams["name"]
		}
		m.Attachments = append(m.Attachments, providers.InboundAttachment{Name: name, ContentType: ct, Size: int64(len(data)), Data: data})
	}
	return []providers.InboundMessage{m}, nil
}
