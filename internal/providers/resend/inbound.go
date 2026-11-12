package resend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/received"
)

// Receiving mail (M58). Resend receives mail for a domain with receiving
// turned on (its MX record is shown on the domain's page) and sends an
// email.received event, metadata only, to the webhooks subscribed to it
// (https://resend.com/docs/webhooks/emails/received). ReputeCast creates a
// webhook for its inbound hook, signed like every Resend webhook (Standard
// Webhooks), and reads the message in the job with the received emails API
// (https://resend.com/docs/api-reference/emails/retrieve-received-email),
// downloading the raw message when Resend offers it.

// inboundEvents are the events the inbound hook's webhook gets.
var inboundEvents = []string{"email.received"}

// maxRaw caps the raw message ReputeCast downloads.
const maxRaw = 50 << 20

var (
	_ providers.InboundReceiver  = (*Provider)(nil)
	_ providers.InboundRegistrar = (*Provider)(nil)
)

// InboundSetup implements providers.InboundReceiver. Resend's MX record
// differs from domain to domain (receive.forge.rmta.net, a regional
// variant, or an Amazon SES endpoint), so people copy it from Resend.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	return providers.InboundSetup{Automatic: true, Steps: []i18n.Text{mInboundReceiving.Keep(i18n.Args{"domain": route.Domain})}}
}

// RegisterInbound implements providers.InboundRegistrar: a webhook for
// email.received at the hook.
func (p *Provider) RegisterInbound(ctx context.Context, _ providers.InboundRoute, hookURL string) error {
	return p.registerHook(ctx, hookURL, inboundEvents)
}

// UnregisterInbound implements providers.InboundRegistrar: the hook's
// webhooks go.
func (p *Provider) UnregisterInbound(ctx context.Context, _ providers.InboundRoute, hookURL string) error {
	return p.UnregisterWebhook(ctx, hookURL)
}

// VerifyInbound implements providers.InboundReceiver: the signature, with
// the signing secret of the webhook posting to the hook (never the secret
// of the events webhook entered at connect time).
func (p *Provider) VerifyInbound(ctx context.Context, route providers.InboundRoute, r *http.Request, body []byte) error {
	path := pathOf(route.URL)
	if path == "" {
		path = r.URL.Path
	}
	return p.verifyFromAPI(ctx, r, body, path)
}

type receivedEvent struct {
	Type string `json:"type"`
	Data struct {
		EmailID string   `json:"email_id"`
		To      []string `json:"to"`
	} `json:"data"`
}

// receivedEmail is GET /emails/receiving/{id}'s answer.
type receivedEmail struct {
	ID             string            `json:"id"`
	To             []string          `json:"to"`
	From           string            `json:"from"`
	Subject        string            `json:"subject"`
	MessageID      string            `json:"message_id"`
	HTML           *string           `json:"html"`
	Text           *string           `json:"text"`
	Headers        map[string]string `json:"headers"`
	CreatedAt      string            `json:"created_at"`
	Authentication *struct {
		SPF   string `json:"spf"`
		DKIM  string `json:"dkim"`
		DMARC string `json:"dmarc"`
	} `json:"authentication"`
	Raw *struct {
		DownloadURL string `json:"download_url"`
	} `json:"raw"`
	Attachments []struct {
		Filename    *string `json:"filename"`
		ContentType string  `json:"content_type"`
		Size        int64   `json:"size"`
	} `json:"attachments"`
}

// ParseInbound implements providers.InboundReceiver: the event names the
// received email, which the API returns with its raw message's download
// link, its parsed content and the receiving server's verdicts.
func (p *Provider) ParseInbound(ctx context.Context, _ providers.InboundRoute, _ *http.Request, body []byte) ([]providers.InboundMessage, error) {
	var ev receivedEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, fmt.Errorf("resend inbound: %w", err)
	}
	if ev.Type != "email.received" || ev.Data.EmailID == "" {
		return []providers.InboundMessage{}, nil
	}
	var e receivedEmail
	if err := p.api.Do(ctx, "GET /emails/receiving/{id}", http.MethodGet, apiclient.Path("/emails/receiving/%s", ev.Data.EmailID), nil, &e); err != nil {
		return nil, err
	}
	m := providers.InboundMessage{ProviderMessageID: ev.Data.EmailID, Recipients: []string{}, ReceivedAt: parseTime(e.CreatedAt)}
	to := e.To
	if len(to) == 0 {
		to = ev.Data.To
	}
	for _, a := range to {
		if a = received.Bare(a); a != "" {
			m.Recipients = append(m.Recipients, a)
		}
	}
	if a := e.Authentication; a != nil {
		m.SPF, m.DKIM, m.DMARC = received.Verdict(a.SPF), received.Verdict(a.DKIM), received.Verdict(a.DMARC)
	}
	if e.Raw != nil && e.Raw.DownloadURL != "" {
		raw, err := p.download(ctx, e.Raw.DownloadURL)
		if err != nil {
			return nil, err
		}
		m.Raw = raw
		return []providers.InboundMessage{m}, nil
	}
	headers := map[string][]string{}
	for k, v := range e.Headers {
		headers[k] = []string{v}
	}
	m.Headers = headers
	m.From = received.Header(headers, "From")
	if m.From == "" {
		m.From = e.From
	}
	m.To = append([]string{}, e.To...)
	m.Subject, m.MessageID = e.Subject, e.MessageID
	m.InReplyTo = received.Header(headers, "In-Reply-To")
	m.References = received.MessageIDs(received.Header(headers, "References"))
	if e.Text != nil {
		m.Text = *e.Text
	}
	if e.HTML != nil {
		m.HTML = *e.HTML
	}
	for _, a := range e.Attachments {
		name := ""
		if a.Filename != nil {
			name = *a.Filename
		}
		m.Attachments = append(m.Attachments, providers.InboundAttachment{Name: name, ContentType: a.ContentType, Size: a.Size})
	}
	return []providers.InboundMessage{m}, nil
}

// download reads the raw message from its signed link, without the API
// key: the link carries its own signature.
func (p *Provider) download(ctx context.Context, link string) ([]byte, error) {
	const op = "GET raw received email"
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, &providers.CallError{Op: op, Message: "the raw message's link is not an HTTPS URL"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.api.HTTP.Do(req)
	if err != nil {
		return nil, apiclient.NetworkError(op, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return nil, &providers.CallError{Op: op, Status: resp.StatusCode, Code: fmt.Sprintf("HTTP %d", resp.StatusCode),
			Temporary: resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxRaw+1))
	if err != nil {
		return nil, apiclient.NetworkError(op, err)
	}
	if len(raw) > maxRaw {
		return nil, &providers.CallError{Op: op, Message: fmt.Sprintf("the message is larger than %d MB", maxRaw>>20)}
	}
	return raw, nil
}
