package awseum

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
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/sms"
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

// StringToSign is the canonical string SNS signs for this message type.
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
	default:
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

// ParseEnvelope reads an SNS message.
func ParseEnvelope(body []byte) (Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(body, &e); err != nil || e.Type == "" {
		return e, fmt.Errorf("%w: not an SNS message", sms.ErrBadSignature)
	}
	return e, nil
}

// ValidCertURL reports whether u may host an SNS signing certificate.
func ValidCertURL(u string) bool {
	pu, err := url.Parse(u)
	return err == nil && pu.Scheme == "https" && snsHost.MatchString(pu.Hostname()) && pu.Port() == "" && strings.HasSuffix(pu.Path, ".pem")
}

var certCache sync.Map // url → *x509.Certificate

func (p *Provider) cert(ctx context.Context, certURL string) (*x509.Certificate, error) {
	if !ValidCertURL(certURL) {
		return nil, fmt.Errorf("signing certificate URL %q is not an SNS URL", certURL)
	}
	if v, ok := certCache.Load(certURL); ok {
		if c := v.(*x509.Certificate); time.Now().Before(c.NotAfter) {
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

// verify checks an envelope's signature and that it comes from the topic
// ReputeCast created (once it knows it).
func (p *Provider) verify(ctx context.Context, e Envelope) error {
	if topic := p.cfg.Value("topic_arn"); topic != "" && e.TopicArn != topic {
		return sms.ErrBadSignature
	}
	sig, err := base64.StdEncoding.DecodeString(e.Signature)
	if err != nil || len(sig) == 0 {
		return sms.ErrBadSignature
	}
	c, err := p.cert(ctx, e.SigningCertURL)
	if err != nil {
		return fmt.Errorf("%w: %w", sms.ErrBadSignature, err)
	}
	pub, ok := c.PublicKey.(*rsa.PublicKey)
	if !ok {
		return sms.ErrBadSignature
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
		return sms.ErrBadSignature
	}
	if err != nil {
		return sms.ErrBadSignature
	}
	return nil
}

// VerifyWebhook implements sms.Provider.
func (p *Provider) VerifyWebhook(ctx context.Context, _ *http.Request, body []byte) error {
	e, err := ParseEnvelope(body)
	if err != nil {
		return err
	}
	return p.verify(ctx, e)
}

// Handshake implements sms.WebhookHandshaker: it confirms the topic's
// subscription by visiting SubscribeURL.
func (p *Provider) Handshake(ctx context.Context, _ *http.Request, body []byte) (bool, error) {
	e, err := ParseEnvelope(body)
	if err != nil || e.Type == "Notification" {
		return false, nil
	}
	if err := p.verify(ctx, e); err != nil {
		return true, err
	}
	if e.Type != "SubscriptionConfirmation" {
		return true, nil
	}
	u, err := url.Parse(e.SubscribeURL)
	if err != nil || u.Scheme != "https" || !snsHost.MatchString(u.Hostname()) {
		return true, fmt.Errorf("%w: SubscribeURL is not an SNS URL", sms.ErrBadSignature)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.SubscribeURL, nil)
	if err != nil {
		return true, err
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return true, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		return true, fmt.Errorf("confirm SNS subscription: HTTP %d", resp.StatusCode)
	}
	return true, nil
}
