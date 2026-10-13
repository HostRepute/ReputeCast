// Package smtp is the generic SMTP adapter (wneessen/go-mail). Plain SMTP
// has no event webhooks: bounces and abuse reports arrive in a bounce
// mailbox that ReputeCast reads over IMAP or POP3 (package bouncebox), and
// ParseWebhook turns those reports (RFC 3464 DSNs, ARF) into events. MTAs
// that can pipe bounces to an HTTP endpoint may also POST them to the
// webhook URL, signed with the provider's webhook secret.
package smtp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	gomail "github.com/wneessen/go-mail"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/providers/smtp/bouncebox"
	"github.com/hostrepute/reputecast/internal/providers/smtp/dsn"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "smtp"

// SignatureHeader carries hex(HMAC-SHA256(webhook secret, body)) on reports
// posted to the webhook URL.
const SignatureHeader = "X-ReputeCast-Signature"

// BatchSize is how many messages one connection sends per Send call.
const BatchSize = 25

// Settings are the non-secret connection settings.
type Settings struct {
	Host string `json:"host"`
	Port int    `json:"port,omitempty"`
	// TLS is "starttls" (default), "tls" (implicit) or "none".
	TLS      string `json:"tls,omitempty"`
	Username string `json:"username,omitempty"`
	// Auth is "auto" (default), "plain", "login" or "none".
	Auth string `json:"auth,omitempty"`
	HELO string `json:"helo,omitempty"`
	// ReturnPath is the envelope sender, usually the bounce mailbox address,
	// so bounces come back to where ReputeCast reads them.
	ReturnPath    string            `json:"return_path,omitempty"`
	BounceMailbox *bouncebox.Config `json:"bounce_mailbox,omitempty"`
}

// Credentials are write-only.
type Credentials struct {
	Password       string `json:"password,omitempty"`
	BouncePassword string `json:"bounce_password,omitempty"`
}

func init() { providers.Register(Descriptor()) }

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Generic SMTP",
		Kind:    "smtp",
		Capabilities: providers.Capabilities{
			Bulk:      providers.BulkYourPolicy,
			Webhooks:  "Bounce mailbox",
			BatchSize: BatchSize,
			Pricing:   "your server or relay",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkYourPolicy, Note: "Your SMTP server or relay's acceptable-use policy applies.",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Bounce mailbox (IMAP/POP3)",
		DomainSetup:     "Manual DNS",
		SuppressionMode: "Local only",
		Fields: []providers.Field{
			{Key: "host", Label: "SMTP host", Required: true},
			{Key: "port", Label: "Port", Required: true, Options: []string{"587", "465", "25", "2525"}},
			{Key: "tls", Label: "Encryption", Required: true, Options: []string{"starttls", "tls", "none"}},
			{Key: "username", Label: "User name"},
			{Key: "password", Label: "Password", Secret: true},
			{Key: "return_path", Label: "Bounce address", Help: "The envelope sender; bounces go here, so point it at the bounce mailbox."},
			{Key: "bounce_mailbox", Label: "Bounce mailbox (IMAP or POP3)", Help: "SMTP has no webhooks. ReputeCast reads bounces and complaints from this mailbox every minute."},
			{Key: "bounce_password", Label: "Bounce mailbox password", Secret: true},
		},
		New: New,
	}
}

// Provider sends through one SMTP server.
type Provider struct {
	cfg  providers.Config
	s    Settings
	c    Credentials
	dial func(ctx context.Context, network, addr string) (net.Conn, error)
}

var _ providers.Provider = (*Provider)(nil)

// New builds the adapter.
func New(cfg providers.Config) (providers.Provider, error) {
	p := &Provider{cfg: cfg}
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &p.s); err != nil {
			return nil, fmt.Errorf("smtp settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("smtp credentials: %w", err)
		}
	}
	if p.s.TLS == "" {
		p.s.TLS = "starttls"
	}
	if p.s.Port == 0 {
		p.s.Port = map[string]int{"starttls": 587, "tls": 465, "none": 25}[p.s.TLS]
	}
	if p.s.BounceMailbox != nil {
		p.s.BounceMailbox.Password = p.c.BouncePassword
	}
	return p, nil
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities { return Descriptor().Capabilities }

// BounceMailbox returns the bounce mailbox configuration with its password,
// or nil when none is set up.
func (p *Provider) BounceMailbox() *bouncebox.Config { return p.s.BounceMailbox }

func (p *Provider) client() (*gomail.Client, error) {
	opts := []gomail.Option{gomail.WithPort(p.s.Port), gomail.WithTimeout(30 * time.Second)}
	if p.s.HELO != "" {
		opts = append(opts, gomail.WithHELO(p.s.HELO))
	}
	tlsCfg := &tls.Config{ServerName: p.s.Host, MinVersion: tls.VersionTLS12}
	switch p.s.TLS {
	case "tls":
		opts = append(opts, gomail.WithSSL(), gomail.WithTLSConfig(tlsCfg))
	case "none":
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	default:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory), gomail.WithTLSConfig(tlsCfg))
	}
	if p.s.Username != "" && p.s.Auth != "none" {
		auth := gomail.SMTPAuthAutoDiscover
		switch p.s.Auth {
		case "plain":
			auth = gomail.SMTPAuthPlain
		case "login":
			auth = gomail.SMTPAuthLogin
		}
		opts = append(opts, gomail.WithSMTPAuth(auth), gomail.WithUsername(p.s.Username), gomail.WithPassword(p.c.Password))
	}
	if p.dial != nil {
		opts = append(opts, gomail.WithDialContextFunc(p.dial))
	}
	return gomail.NewClient(p.s.Host, opts...)
}

// connect dials and authenticates, mapping failures to CallErrors.
func (p *Provider) connect(ctx context.Context) (*gomail.Client, error) {
	c, err := p.client()
	if err != nil {
		return nil, &providers.CallError{Op: "SMTP connect", Message: err.Error(), Cause: err}
	}
	if err := c.DialWithContext(ctx); err != nil {
		return nil, dialError(err)
	}
	return c, nil
}

func dialError(err error) error {
	msg := err.Error()
	low := strings.ToLower(msg)
	ce := &providers.CallError{Op: "SMTP connect", Message: msg, Cause: err}
	switch {
	case strings.Contains(low, "535") || strings.Contains(low, "authentication") || strings.Contains(low, "auth failed") || strings.Contains(low, "534"):
		ce.Op, ce.Status, ce.Auth = "SMTP AUTH", 535, true
	default:
		ce.Temporary = true
	}
	return ce
}

// Send implements providers.Provider over one SMTP connection.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	if len(msgs) == 0 {
		return []providers.SendResult{}, nil
	}
	if len(msgs) > BatchSize {
		return nil, fmt.Errorf("smtp: batch of %d exceeds %d", len(msgs), BatchSize)
	}
	c, err := p.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	now := time.Now()
	gms := make([]*gomail.Msg, len(msgs))
	ids := make([]string, len(msgs))
	for i, m := range msgs {
		gm, id := mimemsg.New(m, now, nil)
		if p.s.ReturnPath != "" {
			_ = gm.EnvelopeFrom(p.s.ReturnPath)
		}
		gms[i], ids[i] = gm, id
	}
	_ = c.Send(gms...) // per-message errors are read below
	results := make([]providers.SendResult, len(msgs))
	sent := 0
	for i, gm := range gms {
		results[i] = providers.SendResult{MessageID: msgs[i].ID}
		if !gm.HasSendError() {
			results[i].ProviderMessageID = ids[i]
			sent++
			continue
		}
		se := &gomail.SendError{}
		code, msg := "SMTP", gm.SendError().Error()
		temp := gm.SendErrorIsTemp()
		if errors.As(gm.SendError(), &se) {
			if se.ErrorCode() != 0 {
				code = fmt.Sprintf("%d", se.ErrorCode())
				if esc := se.EnhancedStatusCode(); esc != "" {
					code += " " + esc
				}
				temp = se.ErrorCode() < 500
			}
		}
		results[i].Err = &providers.SendError{Temporary: temp, Throttled: strings.HasPrefix(code, "421") || strings.HasPrefix(code, "451"), Code: code, Message: msg}
	}
	if sent == 0 {
		// The server refused everything with the same temporary error: treat
		// as unavailable so the route is retried, not each message failed.
		allTemp := true
		for _, r := range results {
			if r.Err == nil || !r.Err.Temporary {
				allTemp = false
			}
		}
		if allTemp && results[0].Err != nil {
			return nil, &providers.CallError{Op: "SMTP DATA", Code: results[0].Err.Code, Message: results[0].Err.Message, Temporary: true}
		}
	}
	return results, nil
}

// Verify implements providers.Verifier: connect, TLS and authenticate.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) {
	c, err := p.connect(ctx)
	if err != nil {
		return providers.Quota{}, err
	}
	_ = c.Close()
	return providers.Quota{Account: p.s.Host + " · accepting mail"}, nil
}

// VerifyWebhook checks the HMAC on reports posted over HTTP.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, body []byte) error {
	sig, err := hex.DecodeString(strings.TrimSpace(r.Header.Get(SignatureHeader)))
	if err != nil || len(sig) == 0 || p.cfg.WebhookSecret == "" {
		return providers.ErrBadSignature
	}
	mac := hmac.New(sha256.New, []byte(p.cfg.WebhookSecret))
	mac.Write(body)
	if !hmac.Equal(mac.Sum(nil), sig) {
		return providers.ErrBadSignature
	}
	return nil
}

// Sign returns the signature header value for a report body.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// ParseWebhook turns a bounce or complaint email into events. Reports read
// from the bounce mailbox arrive with providers.WithSignatureVerified.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	rep, err := dsn.Parse(body)
	if errors.Is(err, dsn.ErrNotAReport) {
		return []providers.Event{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("bounce report: %w", err)
	}
	evs := Events(rep)
	raw := rawReport(rep)
	for i := range evs {
		evs[i].Raw = raw
	}
	return evs, nil
}

// rawReport is the JSON form of a report kept with each event (the email
// itself is stored by the ingest).
func rawReport(rep *dsn.Report) json.RawMessage {
	orig := map[string]string{}
	for _, k := range []string{"Message-Id", "X-Reputecast-Message", "Subject", "To", "From"} {
		if v := rep.Original.Get(k); v != "" {
			orig[k] = v
		}
	}
	b, _ := json.Marshal(map[string]any{
		"kind": rep.Kind, "reporting_mta": rep.ReportingMTA, "mailbox": rep.Mailbox, "recipients": rep.Recipients,
		"feedback_type": rep.FeedbackType, "user_agent": rep.UserAgent, "original": orig,
	})
	return b
}

// Events maps a parsed report to normalized events.
func Events(rep *dsn.Report) []providers.Event {
	var msgID *uuid.UUID
	if id, err := uuid.Parse(strings.TrimSpace(rep.Original.Get(mimemsg.HeaderMessage))); err == nil {
		msgID = &id
	}
	provMsg := strings.TrimSpace(rep.Original.Get("Message-Id"))
	at := rep.ArrivalDate.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	out := []providers.Event{}
	if rep.Kind == dsn.KindARF {
		rcpt := rep.ARFRecipient
		if rcpt == "" {
			rcpt = strings.ToLower(strings.Trim(rep.Original.Get("To"), "<> "))
		}
		ev := providers.Event{Type: providers.EventComplaint, MessageID: msgID, ProviderMessageID: provMsg, Recipient: rcpt, OccurredAt: at,
			OriginalType: "ARF · " + rep.FeedbackType, Diagnostic: "Abuse report"}
		if rep.UserAgent != "" {
			ev.Diagnostic += " from " + rep.UserAgent
		}
		ev.UserAgent = rep.UserAgent
		return append(out, ev)
	}
	for _, rc := range rep.Recipients {
		ev := providers.Event{MessageID: msgID, ProviderMessageID: provMsg, Recipient: rc.FinalRecipient, OccurredAt: at, OriginalType: "DSN · " + rc.Status}
		if rc.Status == "" {
			ev.OriginalType = "DSN · " + rc.Action
		}
		code, rest := bounce.Split(rc.DiagnosticCode)
		ev.OriginalCode = code
		if ev.OriginalCode == "" {
			ev.OriginalCode = rc.Status
		}
		ev.Diagnostic = rest
		switch rc.Action {
		case "delivered", "relayed", "expanded":
			ev.Type = providers.EventDelivered
		case "delayed":
			ev.Type = providers.EventDeferred
			ev.BounceClass = bounce.Classify(rc.Status, rc.DiagnosticCode)
		default: // failed
			if bounce.Permanent(rc.Status) || rc.Status == "" && bounce.Permanent(code) {
				ev.Type = providers.EventHardBounce
			} else {
				ev.Type = providers.EventSoftBounce
			}
			ev.BounceClass = bounce.Classify(rc.Status, rc.DiagnosticCode)
		}
		out = append(out, ev)
	}
	return out
}
