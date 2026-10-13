// Package remote runs community adapters: sending adapters that live in
// their own process and speak the adapter protocol (docs/adapters.md,
// "Community adapters"), signed JSON over HTTP(S). A Provider proxies every
// providers.Provider call to the adapter process, and implements exactly the
// optional interfaces the installed manifest declares, so core code (health
// checks, suppression sync, the connect wizard) sees a community adapter
// like a built-in one. The Runtime registers installed adapters in the
// provider registry as "community:<id>" and keeps them in step with the
// database.
package remote

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
)

// conn is one connected provider reached through an installation.
type conn struct {
	in  *Installation
	cfg providers.Config
	wc  *Connection

	// The webhook endpoint asks for a handshake, then for verification, of
	// the same request: one webhooks/verify call answers both.
	mu   sync.Mutex
	memo *verifyMemo
}

type verifyMemo struct {
	req       *http.Request
	sum       [32]byte
	handshake []byte
	handled   bool
	err       error
}

// base implements Provider, Verifier, WebhookVerifier and WebhookHandshaker.
type base struct{ *conn }

type quotaPart struct{ *conn }
type suppressionPart struct{ *conn }
type domainPart struct{ *conn }
type registrarPart struct{ *conn }

var (
	_ providers.Provider          = base{}
	_ providers.Verifier          = base{}
	_ providers.WebhookVerifier   = base{}
	_ providers.WebhookHandshaker = base{}
	_ providers.QuotaReporter     = quotaPart{}
	_ providers.SuppressionSyncer = suppressionPart{}
	_ providers.DomainManager     = domainPart{}
	_ providers.WebhookRegistrar  = registrarPart{}
)

// Open returns the provider for a connection made through the installation.
// Its method set matches the manifest: QuotaReporter, SuppressionSyncer,
// DomainManager and WebhookRegistrar are only implemented when the
// corresponding capability is declared, because core code decides what to
// do from the interfaces a provider implements.
func (in *Installation) Open(cfg providers.Config) (providers.Provider, error) {
	wc, err := in.connection(cfg)
	if err != nil {
		return nil, err
	}
	c := &conn{in: in, cfg: cfg, wc: wc}
	caps := in.Manifest.Capabilities
	b, q, s, d, r := base{c}, quotaPart{c}, suppressionPart{c}, domainPart{c}, registrarPart{c}
	mask := 0
	if caps.QuotaDiscovery != "" {
		mask |= 1
	}
	if caps.SuppressionSync != "" {
		mask |= 2
	}
	if caps.DomainSetup != "" {
		mask |= 4
	}
	if caps.RegisterWebhooks {
		mask |= 8
	}
	switch mask {
	case 1:
		return &struct {
			base
			quotaPart
		}{b, q}, nil
	case 2:
		return &struct {
			base
			suppressionPart
		}{b, s}, nil
	case 3:
		return &struct {
			base
			quotaPart
			suppressionPart
		}{b, q, s}, nil
	case 4:
		return &struct {
			base
			domainPart
		}{b, d}, nil
	case 5:
		return &struct {
			base
			quotaPart
			domainPart
		}{b, q, d}, nil
	case 6:
		return &struct {
			base
			suppressionPart
			domainPart
		}{b, s, d}, nil
	case 7:
		return &struct {
			base
			quotaPart
			suppressionPart
			domainPart
		}{b, q, s, d}, nil
	case 8:
		return &struct {
			base
			registrarPart
		}{b, r}, nil
	case 9:
		return &struct {
			base
			quotaPart
			registrarPart
		}{b, q, r}, nil
	case 10:
		return &struct {
			base
			suppressionPart
			registrarPart
		}{b, s, r}, nil
	case 11:
		return &struct {
			base
			quotaPart
			suppressionPart
			registrarPart
		}{b, q, s, r}, nil
	case 12:
		return &struct {
			base
			domainPart
			registrarPart
		}{b, d, r}, nil
	case 13:
		return &struct {
			base
			quotaPart
			domainPart
			registrarPart
		}{b, q, d, r}, nil
	case 14:
		return &struct {
			base
			suppressionPart
			domainPart
			registrarPart
		}{b, s, d, r}, nil
	case 15:
		return &struct {
			base
			quotaPart
			suppressionPart
			domainPart
			registrarPart
		}{b, q, s, d, r}, nil
	}
	return &struct{ base }{b}, nil
}

// connection is what an operation sends about the connected provider: only
// the values the manifest's config declares, secrets from the sealed
// credentials and everything else from the settings.
func (in *Installation) connection(cfg providers.Config) (*Connection, error) {
	c := &Connection{Settings: map[string]any{}, Credentials: map[string]any{}}
	if cfg.ProviderID != uuid.Nil {
		c.ProviderID = cfg.ProviderID.String()
	}
	if cfg.WorkspaceID != uuid.Nil {
		c.WorkspaceID = cfg.WorkspaceID.String()
	}
	settings, creds := map[string]any{}, map[string]any{}
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
			return nil, fmt.Errorf("%s settings: %w", in.Manifest.Key(), err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &creds); err != nil {
			return nil, fmt.Errorf("%s credentials: %w", in.Manifest.Key(), err)
		}
	}
	for _, f := range in.Manifest.Config {
		src, dst := settings, c.Settings
		if f.Secret {
			src, dst = creds, c.Credentials
		}
		switch v := src[f.Key].(type) {
		case string, float64, bool:
			dst[f.Key] = v
		}
	}
	if in.Manifest.Capabilities.Webhooks {
		c.WebhookSecret = cfg.WebhookSecret
	}
	return c, nil
}

// Adapter implements providers.Provider.
func (b base) Adapter() string { return b.in.Manifest.Key() }

// Capabilities implements providers.Provider: what the manifest declares.
func (b base) Capabilities() providers.Capabilities { return b.in.Manifest.ProviderCapabilities() }

// Send implements providers.Provider. A batch larger than MaxSendRequest
// once encoded goes out in several calls; when a later call fails after an
// earlier one was accepted, its messages become temporary per-message errors
// rather than failing the whole batch, so a retry never sends twice.
func (b base) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	if len(msgs) == 0 {
		return []providers.SendResult{}, nil
	}
	if limit := max(b.in.Manifest.Capabilities.Batch, 1); len(msgs) > limit {
		return nil, fmt.Errorf("%s: batch of %d exceeds the declared %d", b.in.Manifest.Key(), len(msgs), limit)
	}
	wire := make([]WireMessage, len(msgs))
	sizes := make([]int, len(msgs))
	for i, m := range msgs {
		wire[i] = wireMessage(m)
		raw, err := json.Marshal(wire[i])
		if err != nil {
			return nil, err
		}
		sizes[i] = len(raw)
		if sizes[i] > MaxSendRequest-64<<10 {
			return nil, &providers.CallError{Op: OpSend, Message: fmt.Sprintf("message %s is %d bytes, more than the protocol carries", m.ID, sizes[i])}
		}
	}
	results := make([]providers.SendResult, 0, len(msgs))
	accepted := false
	for start := 0; start < len(msgs); {
		end, size := start, 0
		for end < len(msgs) && (end == start || size+sizes[end] <= MaxSendRequest-64<<10) {
			size += sizes[end]
			end++
		}
		res, err := b.sendChunk(ctx, msgs[start:end], wire[start:end])
		if err != nil {
			if !accepted {
				return nil, err
			}
			for _, m := range msgs[start:] {
				results = append(results, providers.SendResult{MessageID: m.ID, Err: tempError(err)})
			}
			return results, nil
		}
		for _, r := range res {
			if r.Err == nil {
				accepted = true
			}
		}
		results = append(results, res...)
		start = end
	}
	return results, nil
}

// sendKey derives the idempotency key of a send from its message ids, so a
// retry of the same batch is recognized by the adapter.
func sendKey(msgs []providers.Message) string {
	h := sha256.New()
	for _, m := range msgs {
		h.Write(m.ID[:])
	}
	return "send-" + hex.EncodeToString(h.Sum(nil))[:32]
}

func (b base) sendChunk(ctx context.Context, msgs []providers.Message, wire []WireMessage) ([]providers.SendResult, error) {
	var out SendOutput
	if err := b.in.call(ctx, OpSend, b.wc, sendKey(msgs), SendInput{Messages: wire}, &out); err != nil {
		return nil, err
	}
	res, err := sendResults(msgs, out)
	if err != nil {
		return nil, b.in.invalidAnswer(OpSend, err)
	}
	return res, nil
}

func tempError(err error) *providers.SendError {
	se := &providers.SendError{Temporary: true, Message: err.Error()}
	var ce *providers.CallError
	if errors.As(err, &ce) {
		se.Throttled, se.Code, se.Message = ce.Throttled, ce.Code, ce.Message
	}
	return se
}

// hopHeaders are not forwarded with webhooks.
var hopHeaders = map[string]bool{"connection": true, "keep-alive": true, "proxy-connection": true, "proxy-authorization": true,
	"proxy-authenticate": true, "te": true, "trailer": true, "transfer-encoding": true, "upgrade": true, "cookie": true, "host": true}

// webhookRequest is the forwarded form of a provider's webhook request.
func (b base) webhookRequest(r *http.Request, body []byte) WebhookRequest {
	w := WebhookRequest{Method: r.Method, Headers: map[string][]string{}, Body: body}
	if w.Method == "" {
		w.Method = http.MethodPost
	}
	switch {
	case r.URL != nil && r.URL.IsAbs():
		w.URL = r.URL.String()
	case r.URL != nil && strings.HasPrefix(r.URL.Path, "/hooks/"):
		w.URL = b.in.PublicURL + r.URL.RequestURI()
	default:
		w.URL = b.in.PublicURL + "/hooks/" + b.in.Manifest.Key() + "/"
	}
	size := 0
	for k, vs := range r.Header {
		lk := strings.ToLower(k)
		if hopHeaders[lk] {
			continue
		}
		for _, v := range vs {
			if size += len(lk) + len(v); size > 64<<10 {
				return w
			}
			w.Headers[lk] = append(w.Headers[lk], v)
		}
	}
	return w
}

// VerifyWebhook implements providers.WebhookVerifier: the adapter checks the
// provider's signature. When the adapter can't answer, the error wraps
// providers.ErrUnavailable (the endpoint asks the provider to retry), never
// a verdict on the signature.
func (b base) VerifyWebhook(ctx context.Context, r *http.Request, body []byte) error {
	if !b.in.Manifest.Capabilities.Webhooks {
		return fmt.Errorf("%w: the %s adapter doesn't declare webhooks", providers.ErrBadSignature, b.in.Manifest.Name)
	}
	m := b.verify(ctx, r, body)
	return m.err
}

// Handshake implements providers.WebhookHandshaker: an adapter answers a
// subscription handshake from webhooks/verify.
func (b base) Handshake(ctx context.Context, r *http.Request, body []byte) (bool, []byte, error) {
	if !b.in.Manifest.Capabilities.Webhooks {
		return false, nil, nil
	}
	m := b.verify(ctx, r, body)
	return m.handled, m.handshake, nil
}

func (b base) verify(ctx context.Context, r *http.Request, body []byte) *verifyMemo {
	sum := sha256.Sum256(body)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.memo != nil && b.memo.req == r && b.memo.sum == sum {
		return b.memo
	}
	m := &verifyMemo{req: r, sum: sum}
	var out WebhookVerifyOutput
	err := b.in.call(ctx, OpWebhookVerify, b.wc, "", WebhookVerifyInput{Request: b.webhookRequest(r, body)}, &out)
	switch {
	case errors.Is(err, providers.ErrBadSignature):
		m.err = err
	case err != nil:
		m.err = fmt.Errorf("%w: %w", providers.ErrUnavailable, err)
	case out.Handshake != nil:
		if len(out.Handshake.Body) > maxHandshake {
			m.err = fmt.Errorf("%w: %w", providers.ErrUnavailable, b.in.invalidAnswer(OpWebhookVerify, invalid("the handshake answer is larger than 16 KiB")))
			break
		}
		m.handled, m.handshake = true, []byte(out.Handshake.Body)
		if len(bytes.TrimSpace(m.handshake)) == 0 {
			m.handshake = nil
		}
	}
	b.memo = m
	return m
}

// ParseWebhook implements providers.Provider: the adapter normalizes the
// payload. With a verified context (the normalization job), the adapter is
// told not to verify again. Failures that can pass (the adapter is down, the
// provider is) wrap providers.ErrUnavailable, so the job retries instead of
// dropping the events; a payload the adapter refuses for good does not.
func (b base) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !b.in.Manifest.Capabilities.Webhooks {
		return nil, fmt.Errorf("%w: the %s adapter doesn't declare webhooks", providers.ErrNotSupported, b.in.Manifest.Name)
	}
	var out WebhookParseOutput
	in := WebhookParseInput{Request: b.webhookRequest(r, body), Verified: providers.SignatureVerified(ctx)}
	if err := b.in.call(ctx, OpWebhookParse, b.wc, "", in, &out); err != nil {
		var ce *providers.CallError
		switch {
		case errors.Is(err, providers.ErrBadSignature) || errors.Is(err, providers.ErrNotSupported):
			return nil, err
		case errors.As(err, &ce) && !ce.Temporary && !ce.Auth:
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w", providers.ErrUnavailable, err)
	}
	evs, err := events(out)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", providers.ErrUnavailable, b.in.invalidAnswer(OpWebhookParse, err))
	}
	return evs, nil
}

// Verify implements providers.Verifier: the adapter checks the credentials.
func (b base) Verify(ctx context.Context) (providers.Quota, error) {
	var out WireQuota
	if err := b.in.call(ctx, OpVerify, b.wc, "", struct{}{}, &out); err != nil {
		return providers.Quota{}, err
	}
	q, err := quota(out)
	if err != nil {
		return providers.Quota{}, b.in.invalidAnswer(OpVerify, err)
	}
	return q, nil
}

// Quota implements providers.QuotaReporter (quotaDiscovery).
func (q quotaPart) Quota(ctx context.Context) (providers.Quota, error) {
	var out WireQuota
	if err := q.in.call(ctx, OpQuota, q.wc, "", struct{}{}, &out); err != nil {
		return providers.Quota{}, err
	}
	res, err := quota(out)
	if err != nil {
		return providers.Quota{}, q.in.invalidAnswer(OpQuota, err)
	}
	return res, nil
}

// ListSuppressions implements providers.SuppressionSyncer: every page of the
// provider's list since a time (nothing for push-only adapters).
func (s suppressionPart) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	out := []providers.Suppression{}
	if !s.in.Manifest.Capabilities.Pull() {
		return out, nil
	}
	in := SuppressionsListInput{}
	if !since.IsZero() {
		v := since.UTC().Format(time.RFC3339)
		in.Since = &v
	}
	for page := 0; page < MaxSuppressionPages; page++ {
		var res SuppressionsListOutput
		if err := s.in.call(ctx, OpSuppressionsList, s.wc, "", in, &res); err != nil {
			return nil, err
		}
		list, err := suppressions(res)
		if err != nil {
			return nil, s.in.invalidAnswer(OpSuppressionsList, err)
		}
		out = append(out, list...)
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		if in.Cursor != nil && *in.Cursor == *res.NextCursor {
			return nil, s.in.invalidAnswer(OpSuppressionsList, invalid("nextCursor repeats the cursor it was given"))
		}
		in.Cursor = res.NextCursor
	}
	return out, nil
}

// AddSuppressions implements providers.SuppressionSyncer, 500 addresses per
// call (nothing for pull-only adapters: the shared list is enforced locally).
func (s suppressionPart) AddSuppressions(ctx context.Context, emails []string) error {
	if !s.in.Manifest.Capabilities.Push() {
		return nil
	}
	for i := 0; i < len(emails); i += SuppressionPushBatch {
		chunk := emails[i:min(i+SuppressionPushBatch, len(emails))]
		if err := s.in.call(ctx, OpSuppressionsAdd, s.wc, "", SuppressionsAddInput{Emails: chunk}, nil); err != nil {
			return err
		}
	}
	return nil
}

// AddDomain implements providers.DomainManager (domainSetup).
func (d domainPart) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	return d.domain(ctx, OpDomainAdd, domain)
}

// DomainStatus implements providers.DomainManager (domainSetup).
func (d domainPart) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	return d.domain(ctx, OpDomainStatus, domain)
}

func (d domainPart) domain(ctx context.Context, op, domain string) ([]providers.DNSRecord, error) {
	var out DomainOutput
	if err := d.in.call(ctx, op, d.wc, "", DomainInput{Domain: strings.ToLower(strings.TrimSpace(domain))}, &out); err != nil {
		return nil, err
	}
	recs, err := records(out)
	if err != nil {
		return nil, d.in.invalidAnswer(op, err)
	}
	return recs, nil
}

// RegisterWebhook implements providers.WebhookRegistrar (registerWebhooks).
func (r registrarPart) RegisterWebhook(ctx context.Context, url string) error {
	return r.in.call(ctx, OpWebhookRegister, r.wc, "", RegisterInput{URL: url}, nil)
}
