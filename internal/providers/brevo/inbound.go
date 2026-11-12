package brevo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/received"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// Receiving mail (M58). Brevo's inbound parsing receives mail for a
// domain whose MX points at inbound1/inbound2.sendinblue.com and posts
// each message, parsed into JSON, to the inbound webhook for the domain
// (https://developers.brevo.com/docs/inbound-parse-webhooks). ReputeCast
// creates the webhook with the route's secret as its bearer token: Brevo
// doesn't sign webhooks.

// inboundMX are Brevo's inbound mail servers and their priorities.
var inboundMX = []string{"10 inbound1.sendinblue.com", "20 inbound2.sendinblue.com"}

// spamScore is the score at which rspamd, which Brevo's spam score comes
// from, adds its spam header by default.
const spamScore = 6.0

// maxReportAttachment caps the attachment data read for a report.
const maxReportAttachment = 8 << 20

var (
	_ providers.InboundReceiver  = (*Provider)(nil)
	_ providers.InboundRegistrar = (*Provider)(nil)
)

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	setup := providers.InboundSetup{Automatic: true, Steps: []i18n.Text{mInboundDomain.Keep(i18n.Args{"domain": route.Domain})}}
	for _, v := range inboundMX {
		setup.MX = append(setup.MX, providers.DNSRecord{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: v})
	}
	return setup
}

func (p *Provider) inboundWebhooks(ctx context.Context) ([]webhook, error) {
	var out struct {
		Webhooks []webhook `json:"webhooks"`
	}
	err := p.api.Do(ctx, "GET /v3/webhooks", http.MethodGet, "/v3/webhooks?type=inbound", nil, &out)
	return out.Webhooks, classify(err)
}

// RegisterInbound implements providers.InboundRegistrar: an inbound webhook
// (type inbound, event inboundEmailProcessed) for the domain, with the
// route's secret as its bearer token. One already posting to the hook is
// updated.
func (p *Provider) RegisterInbound(ctx context.Context, route providers.InboundRoute, hookURL string) error {
	if route.Secret == "" {
		return errors.New("the inbound route has no secret")
	}
	hooks, err := p.inboundWebhooks(ctx)
	if err != nil {
		return err
	}
	body := map[string]any{"url": hookURL, "description": "ReputeCast", "events": []string{"inboundEmailProcessed"}, "domain": strings.ToLower(route.Domain),
		"auth": map[string]string{"type": "bearer", "token": route.Secret}}
	for _, h := range hooks {
		if sameURL(h.URL, hookURL) {
			return classify(p.api.Do(ctx, "PUT /v3/webhooks/{id}", http.MethodPut, "/v3/webhooks/"+strconv.FormatInt(h.ID, 10), body, nil))
		}
	}
	body["type"] = "inbound"
	return classify(p.api.Do(ctx, "POST /v3/webhooks", http.MethodPost, "/v3/webhooks", body, nil))
}

// UnregisterInbound implements providers.InboundRegistrar: every inbound
// webhook posting to the hook goes; one already gone counts as removed.
func (p *Provider) UnregisterInbound(ctx context.Context, _ providers.InboundRoute, hookURL string) error {
	hooks, err := p.inboundWebhooks(ctx)
	if err != nil {
		return err
	}
	for _, h := range hooks {
		if !sameURL(h.URL, hookURL) {
			continue
		}
		err := p.api.Do(ctx, "DELETE /v3/webhooks/{id}", http.MethodDelete, "/v3/webhooks/"+strconv.FormatInt(h.ID, 10), nil, nil)
		if err != nil && !apiclient.NotFound(err) {
			return classify(err)
		}
	}
	return nil
}

// VerifyInbound implements providers.InboundReceiver: the bearer token is
// the route's secret.
func (p *Provider) VerifyInbound(_ context.Context, route providers.InboundRoute, r *http.Request, _ []byte) error {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || !webhookauth.Equal(strings.TrimSpace(token), route.Secret) {
		return fmt.Errorf("%w: the request doesn't carry the route's token", providers.ErrBadSignature)
	}
	return nil
}

type mailbox struct {
	Name    string `json:"Name"`
	Address string `json:"Address"`
}

// recipient is an entry of Recipients: a mailbox as documented, or an
// address as Brevo's sample sends it.
type recipient string

func (r *recipient) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*r = recipient(s)
		return nil
	}
	var m mailbox
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*r = recipient(m.Address)
	return nil
}

type inboundItem struct {
	MessageID   string                       `json:"MessageId"`
	InReplyTo   *string                      `json:"InReplyTo"`
	From        mailbox                      `json:"From"`
	To          []mailbox                    `json:"To"`
	Recipients  []recipient                  `json:"Recipients"`
	Subject     string                       `json:"Subject"`
	RawHTMLBody *string                      `json:"RawHtmlBody"`
	RawTextBody *string                      `json:"RawTextBody"`
	Markdown    string                       `json:"ExtractedMarkdownMessage"`
	SpamScore   json.Number                  `json:"SpamScore"`
	Spam        *struct{ Score json.Number } `json:"Spam"`
	Headers     map[string]json.RawMessage   `json:"Headers"`
	Attachments []struct {
		Name          string `json:"Name"`
		ContentType   string `json:"ContentType"`
		ContentLength int64  `json:"ContentLength"`
		DownloadToken string `json:"DownloadToken"`
	} `json:"Attachments"`
}

// report reports whether an attachment's type is one a report comes as
// (a DMARC aggregate report, a feedback report): its data is read.
func report(contentType string) bool {
	ct := strings.ToLower(contentType)
	for _, t := range []string{"zip", "gzip", "xml", "feedback-report", "message/rfc822"} {
		if strings.Contains(ct, t) {
			return true
		}
	}
	return false
}

// ParseInbound implements providers.InboundReceiver: every item of the
// payload, from Brevo's parsed fields; a report's attachments are read
// with their download token.
func (p *Provider) ParseInbound(ctx context.Context, _ providers.InboundRoute, _ *http.Request, body []byte) ([]providers.InboundMessage, error) {
	var in struct {
		Items []inboundItem `json:"items"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, fmt.Errorf("brevo inbound: %w", err)
	}
	out := make([]providers.InboundMessage, 0, len(in.Items))
	for _, it := range in.Items {
		headers := map[string][]string{}
		for k, raw := range it.Headers {
			var one string
			if json.Unmarshal(raw, &one) == nil {
				headers[k] = []string{one}
				continue
			}
			var many []string
			_ = json.Unmarshal(raw, &many)
			headers[k] = many
		}
		m := providers.InboundMessage{
			From: received.Address(it.From.Name, it.From.Address), Subject: it.Subject, MessageID: it.MessageID, To: []string{},
			References: received.MessageIDs(received.Header(headers, "References")), Headers: headers, Recipients: []string{},
		}
		if it.InReplyTo != nil {
			m.InReplyTo = *it.InReplyTo
		}
		for _, a := range it.To {
			m.To = append(m.To, received.Address(a.Name, a.Address))
		}
		for _, r := range it.Recipients {
			if a := received.Bare(string(r)); a != "" {
				m.Recipients = append(m.Recipients, a)
			}
		}
		switch {
		case it.RawTextBody != nil:
			m.Text = *it.RawTextBody
		default:
			m.Text = it.Markdown
		}
		if it.RawHTMLBody != nil {
			m.HTML = *it.RawHTMLBody
		}
		score := it.SpamScore.String()
		if score == "" && it.Spam != nil {
			score = it.Spam.Score.String()
		}
		m.Spam = received.SpamScore(score, spamScore)
		for _, a := range it.Attachments {
			att := providers.InboundAttachment{Name: a.Name, ContentType: a.ContentType, Size: a.ContentLength}
			if report(a.ContentType) && a.DownloadToken != "" && a.ContentLength <= maxReportAttachment {
				res, err := p.api.Exec(ctx, apiclient.Request{Op: "GET /v3/inbound/attachments/{token}", Method: http.MethodGet, Path: apiclient.Path("/v3/inbound/attachments/%s", a.DownloadToken)})
				if err != nil {
					return nil, classify(err)
				}
				att.Data = res.Body
				att.Size = int64(len(res.Body))
			}
			m.Attachments = append(m.Attachments, att)
		}
		out = append(out, m)
	}
	return out, nil
}
