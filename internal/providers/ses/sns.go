package ses

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // SNS SignatureVersion 1 is RSA-SHA1
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
)

// snsHost matches the hosts SNS signs from and sends subscription links on.
var snsHost = regexp.MustCompile(`^sns\.[a-z0-9-]+\.amazonaws\.com(\.cn)?$`)

// Envelope is an SNS HTTP(S) message.
type Envelope struct {
	Type             string `json:"Type"`
	MessageID        string `json:"MessageId"`
	Token            string `json:"Token,omitempty"`
	TopicArn         string `json:"TopicArn"`
	Subject          string `json:"Subject,omitempty"`
	Message          string `json:"Message"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
	SubscribeURL     string `json:"SubscribeURL,omitempty"`
	UnsubscribeURL   string `json:"UnsubscribeURL,omitempty"`
}

// StringToSign builds the canonical string SNS signs for this message type.
func (e Envelope) StringToSign() string {
	var b strings.Builder
	add := func(k, v string) {
		b.WriteString(k)
		b.WriteByte('\n')
		b.WriteString(v)
		b.WriteByte('\n')
	}
	switch e.Type {
	case "Notification":
		add("Message", e.Message)
		add("MessageId", e.MessageID)
		if e.Subject != "" {
			add("Subject", e.Subject)
		}
		add("Timestamp", e.Timestamp)
		add("TopicArn", e.TopicArn)
		add("Type", e.Type)
	default: // SubscriptionConfirmation, UnsubscribeConfirmation
		add("Message", e.Message)
		add("MessageId", e.MessageID)
		add("SubscribeURL", e.SubscribeURL)
		add("Timestamp", e.Timestamp)
		add("Token", e.Token)
		add("TopicArn", e.TopicArn)
		add("Type", e.Type)
	}
	return b.String()
}

// certCache keeps SNS signing certificates by URL until they expire.
var certCache sync.Map // url → *x509.Certificate

// ValidCertURL reports whether u may host an SNS signing certificate.
func ValidCertURL(u string) bool {
	pu, err := url.Parse(u)
	return err == nil && pu.Scheme == "https" && snsHost.MatchString(pu.Hostname()) && pu.Port() == "" && strings.HasSuffix(pu.Path, ".pem")
}

func (p *Provider) cert(ctx context.Context, certURL string) (*x509.Certificate, error) {
	if !ValidCertURL(certURL) {
		return nil, fmt.Errorf("signing certificate URL %q is not an SNS URL", certURL)
	}
	if v, ok := certCache.Load(certURL); ok {
		c := v.(*x509.Certificate)
		if time.Now().Before(c.NotAfter) {
			return c, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, certURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signing certificate: HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("signing certificate is not PEM")
	}
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	certCache.Store(certURL, c)
	return c, nil
}

// verify checks an envelope's signature and topic.
func (p *Provider) verify(ctx context.Context, e Envelope) error {
	if len(p.s.TopicARNs) > 0 && !slices.Contains(p.s.TopicARNs, e.TopicArn) {
		return providers.ErrBadSignature
	}
	sig, err := base64.StdEncoding.DecodeString(e.Signature)
	if err != nil || len(sig) == 0 {
		return providers.ErrBadSignature
	}
	c, err := p.cert(ctx, e.SigningCertURL)
	if err != nil {
		return fmt.Errorf("%w: %w", providers.ErrBadSignature, err)
	}
	pub, ok := c.PublicKey.(*rsa.PublicKey)
	if !ok {
		return providers.ErrBadSignature
	}
	msg := []byte(e.StringToSign())
	switch e.SignatureVersion {
	case "1":
		sum := sha1.Sum(msg) //nolint:gosec // SNS SignatureVersion 1
		err = rsa.VerifyPKCS1v15(pub, crypto.SHA1, sum[:], sig)
	case "2":
		sum := sha256.Sum256(msg)
		err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig)
	default:
		return providers.ErrBadSignature
	}
	if err != nil {
		return providers.ErrBadSignature
	}
	return nil
}

func parseEnvelope(body []byte) (Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(body, &e); err != nil {
		return e, fmt.Errorf("%w: not an SNS message", providers.ErrBadSignature)
	}
	return e, nil
}

// VerifyWebhook implements providers.WebhookVerifier.
func (p *Provider) VerifyWebhook(ctx context.Context, _ *http.Request, body []byte) error {
	e, err := parseEnvelope(body)
	if err != nil {
		return err
	}
	return p.verify(ctx, e)
}

// Handshake implements providers.WebhookHandshaker: it confirms SNS
// subscriptions to the provider's topic by visiting SubscribeURL.
func (p *Provider) Handshake(ctx context.Context, _ *http.Request, body []byte) (bool, []byte, error) {
	e, err := parseEnvelope(body)
	if err != nil || e.Type == "Notification" {
		return false, nil, nil
	}
	if err := p.verify(ctx, e); err != nil {
		return true, nil, err
	}
	if e.Type != "SubscriptionConfirmation" {
		return true, nil, nil // UnsubscribeConfirmation: nothing to do
	}
	u, err := url.Parse(e.SubscribeURL)
	if err != nil || u.Scheme != "https" || !snsHost.MatchString(u.Hostname()) {
		return true, nil, fmt.Errorf("%w: SubscribeURL is not an SNS URL", providers.ErrBadSignature)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.SubscribeURL, nil)
	if err != nil {
		return true, nil, err
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return true, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		return true, nil, fmt.Errorf("confirm SNS subscription: HTTP %d", resp.StatusCode)
	}
	return true, nil, nil
}

// sesEvent is an SES event (configuration-set event publishing) or a
// notification (identity feedback); both arrive as the SNS Message.
type sesEvent struct {
	EventType        string `json:"eventType"`
	NotificationType string `json:"notificationType"`
	Mail             struct {
		Timestamp   string              `json:"timestamp"`
		MessageID   string              `json:"messageId"`
		Destination []string            `json:"destination"`
		Tags        map[string][]string `json:"tags"`
	} `json:"mail"`
	Bounce *struct {
		BounceType        string `json:"bounceType"`
		BounceSubType     string `json:"bounceSubType"`
		BouncedRecipients []struct {
			EmailAddress   string `json:"emailAddress"`
			Action         string `json:"action"`
			Status         string `json:"status"`
			DiagnosticCode string `json:"diagnosticCode"`
		} `json:"bouncedRecipients"`
		Timestamp    string `json:"timestamp"`
		ReportingMTA string `json:"reportingMTA"`
	} `json:"bounce"`
	Complaint *struct {
		ComplainedRecipients []struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"complainedRecipients"`
		Timestamp             string `json:"timestamp"`
		ComplaintFeedbackType string `json:"complaintFeedbackType"`
		ComplaintSubType      string `json:"complaintSubType"`
		UserAgent             string `json:"userAgent"`
	} `json:"complaint"`
	Delivery *struct {
		Timestamp    string   `json:"timestamp"`
		Recipients   []string `json:"recipients"`
		SMTPResponse string   `json:"smtpResponse"`
		ReportingMTA string   `json:"reportingMTA"`
	} `json:"delivery"`
	Reject *struct {
		Reason string `json:"reason"`
	} `json:"reject"`
	DeliveryDelay *struct {
		Timestamp         string `json:"timestamp"`
		DelayType         string `json:"delayType"`
		ExpirationTime    string `json:"expirationTime"`
		ReportingMTA      string `json:"reportingMTA"`
		DelayedRecipients []struct {
			EmailAddress   string `json:"emailAddress"`
			Status         string `json:"status"`
			DiagnosticCode string `json:"diagnosticCode"`
		} `json:"delayedRecipients"`
	} `json:"deliveryDelay"`
	Open *struct {
		Timestamp string `json:"timestamp"`
		UserAgent string `json:"userAgent"`
	} `json:"open"`
	Click *struct {
		Timestamp string `json:"timestamp"`
		Link      string `json:"link"`
		UserAgent string `json:"userAgent"`
	} `json:"click"`
	Subscription *struct {
		Timestamp string `json:"timestamp"`
	} `json:"subscription"`
}

// ParseWebhook implements providers.Provider for SNS notifications.
func (p *Provider) ParseWebhook(ctx context.Context, _ *http.Request, body []byte) ([]providers.Event, error) {
	e, err := parseEnvelope(body)
	if err != nil {
		return nil, err
	}
	if !providers.SignatureVerified(ctx) {
		if err := p.verify(ctx, e); err != nil {
			return nil, err
		}
	}
	if e.Type != "Notification" {
		return []providers.Event{}, nil
	}
	var se sesEvent
	if err := json.Unmarshal([]byte(e.Message), &se); err != nil {
		return nil, fmt.Errorf("ses event: %w", err)
	}
	evs := normalize(se)
	raw := json.RawMessage(e.Message)
	if !json.Valid(raw) {
		raw = append(json.RawMessage(nil), body...)
	}
	for i := range evs {
		evs[i].Raw = raw
	}
	return evs, nil
}

func normalize(se sesEvent) []providers.Event {
	kind := se.EventType
	if kind == "" {
		kind = se.NotificationType
	}
	base := providers.Event{ProviderMessageID: se.Mail.MessageID}
	if v := se.Mail.Tags[TagMessage]; len(v) > 0 {
		if id, err := uuid.Parse(v[0]); err == nil {
			base.MessageID = &id
		}
	}
	out := []providers.Event{}
	switch kind {
	case "Bounce":
		if se.Bounce == nil {
			return out
		}
		typ := providers.EventSoftBounce
		if se.Bounce.BounceType == "Permanent" {
			typ = providers.EventHardBounce
		}
		for _, r := range se.Bounce.BouncedRecipients {
			ev := base
			ev.Type = typ
			ev.Recipient = strings.ToLower(r.EmailAddress)
			ev.OccurredAt = parseTime(se.Bounce.Timestamp)
			ev.OriginalType = "Bounce · " + se.Bounce.BounceType + " · " + se.Bounce.BounceSubType
			ev.OriginalCode, ev.Diagnostic = bounce.Split(r.DiagnosticCode)
			if ev.OriginalCode == "" && r.Status != "" {
				ev.OriginalCode = r.Status
			}
			ev.BounceClass = sesClass(se.Bounce.BounceSubType, r.Status, r.DiagnosticCode)
			out = append(out, ev)
		}
	case "Complaint":
		if se.Complaint == nil {
			return out
		}
		for _, r := range se.Complaint.ComplainedRecipients {
			ev := base
			ev.Type = providers.EventComplaint
			ev.Recipient = strings.ToLower(r.EmailAddress)
			ev.OccurredAt = parseTime(se.Complaint.Timestamp)
			ev.OriginalType = "Complaint"
			if se.Complaint.ComplaintFeedbackType != "" {
				ev.OriginalType += " · " + se.Complaint.ComplaintFeedbackType
			}
			ev.Diagnostic = "ARF feedback via SES"
			if se.Complaint.UserAgent != "" {
				ev.Diagnostic += " · " + se.Complaint.UserAgent
			}
			ev.UserAgent = se.Complaint.UserAgent
			out = append(out, ev)
		}
	case "Delivery":
		if se.Delivery == nil {
			return out
		}
		for _, r := range se.Delivery.Recipients {
			ev := base
			ev.Type = providers.EventDelivered
			ev.Recipient = strings.ToLower(r)
			ev.OccurredAt = parseTime(se.Delivery.Timestamp)
			ev.OriginalType = "Delivery"
			code, rest := bounce.Split(se.Delivery.SMTPResponse)
			ev.OriginalCode = code
			ev.Diagnostic = rest
			if se.Delivery.ReportingMTA != "" {
				ev.Diagnostic = strings.TrimSpace(rest + " · smtpResponse from " + se.Delivery.ReportingMTA)
			}
			out = append(out, ev)
		}
	case "Reject":
		reason := ""
		if se.Reject != nil {
			reason = se.Reject.Reason
		}
		for _, r := range se.Mail.Destination {
			ev := base
			ev.Type = providers.EventRejected
			ev.Recipient = strings.ToLower(r)
			ev.OccurredAt = parseTime(se.Mail.Timestamp)
			ev.OriginalType = "Reject"
			if reason != "" {
				ev.OriginalType += " · " + reason
			}
			ev.Diagnostic = "SES refused the message before sending"
			ev.BounceClass = bounce.SpamBlock
			out = append(out, ev)
		}
	case "DeliveryDelay":
		if se.DeliveryDelay == nil {
			return out
		}
		for _, r := range se.DeliveryDelay.DelayedRecipients {
			ev := base
			ev.Type = providers.EventDeferred
			ev.Recipient = strings.ToLower(r.EmailAddress)
			ev.OccurredAt = parseTime(se.DeliveryDelay.Timestamp)
			ev.OriginalType = "DeliveryDelay · " + se.DeliveryDelay.DelayType
			ev.OriginalCode, ev.Diagnostic = bounce.Split(r.DiagnosticCode)
			if ev.OriginalCode == "" {
				ev.OriginalCode = r.Status
			}
			ev.BounceClass = bounce.Classify(r.Status, r.DiagnosticCode+" "+se.DeliveryDelay.DelayType)
			if se.DeliveryDelay.DelayType == "MailboxFull" {
				ev.BounceClass = bounce.MailboxFull
			}
			out = append(out, ev)
		}
	case "Open":
		for _, r := range se.Mail.Destination {
			ev := base
			ev.Type = providers.EventOpen
			ev.Recipient = strings.ToLower(r)
			ev.OriginalType = "Open"
			if se.Open != nil {
				ev.OccurredAt = parseTime(se.Open.Timestamp)
				ev.UserAgent = se.Open.UserAgent
			}
			out = append(out, ev)
		}
	case "Click":
		for _, r := range se.Mail.Destination {
			ev := base
			ev.Type = providers.EventClick
			ev.Recipient = strings.ToLower(r)
			ev.OriginalType = "Click"
			if se.Click != nil {
				ev.OccurredAt = parseTime(se.Click.Timestamp)
				ev.URL = se.Click.Link
				ev.UserAgent = se.Click.UserAgent
			}
			out = append(out, ev)
		}
	case "Subscription":
		for _, r := range se.Mail.Destination {
			ev := base
			ev.Type = providers.EventUnsubscribe
			ev.Recipient = strings.ToLower(r)
			ev.OriginalType = "Subscription"
			if se.Subscription != nil {
				ev.OccurredAt = parseTime(se.Subscription.Timestamp)
			}
			out = append(out, ev)
		}
	}
	for i := range out {
		if out[i].OccurredAt.IsZero() {
			out[i].OccurredAt = time.Now().UTC()
		}
	}
	return out
}

// sesClass maps SES bounce subtypes to the shared taxonomy.
func sesClass(subType, status, diagnostic string) string {
	switch subType {
	case "NoEmail":
		return bounce.MailboxUnknown
	case "MailboxFull":
		return bounce.MailboxFull
	case "Suppressed", "OnAccountSuppressionList":
		return bounce.PolicyBlock
	case "ContentRejected", "AttachmentRejected":
		return bounce.SpamBlock
	case "MessageTooLarge":
		return bounce.Other
	}
	return bounce.Classify(status, diagnostic)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}
