package sendgrid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/received"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// Receiving mail (M58). SendGrid's Inbound Parse receives mail for a
// hostname of an authenticated domain whose MX points at mx.sendgrid.net
// and posts each message, as multipart form data with the raw message, to
// the hostname's URL
// (https://www.twilio.com/docs/sendgrid/for-developers/parsing-email/setting-up-the-inbound-parse-webhook).
// ReputeCast creates the hostname's parse setting with the hook's basic
// authentication in the URL, and attaches a security policy that signs
// every post like the Event Webhook (ECDSA over timestamp and body,
// https://www.twilio.com/docs/sendgrid/for-developers/parsing-email/securing-your-parse-webhooks)
// where the account can have one.

// InboundMX is SendGrid's inbound mail server.
const InboundMX = "mx.sendgrid.net"

// ParseWindow is how old a signed Inbound Parse timestamp may be: SendGrid
// retries a post for 72 hours and doesn't document whether a retry is
// signed again
// (https://support.sendgrid.com/hc/en-us/articles/46513815674395-Understanding-Inbound-Parse-Webhook-Retry-Logic).
const ParseWindow = 74 * time.Hour

// unsigned is the material cached for a parse setting without a signing
// policy: basic authentication is all there is to check.
const unsigned = "unsigned"

var (
	_ providers.InboundReceiver  = (*Provider)(nil)
	_ providers.InboundRegistrar = (*Provider)(nil)
)

// parseKeys caches each parse setting's public key (or unsigned), shared by
// every adapter instance.
var parseKeys webhookauth.Material

func (p *Provider) parseKeyID(hostname string) string {
	return webhookauth.Key(p.api.BaseURL, p.c.APIKey, "parse", strings.ToLower(hostname))
}

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	args := i18n.Args{"domain": route.Domain}
	return providers.InboundSetup{
		Automatic: true,
		MX:        []providers.DNSRecord{{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: "10 " + InboundMX}},
		Steps:     []i18n.Text{mInboundAuthenticated.Keep(args), mInboundAutomaticSecurity.Keep(args)},
	}
}

type parseSetting struct {
	URL            string `json:"url"`
	Hostname       string `json:"hostname"`
	SpamCheck      bool   `json:"spam_check"`
	SendRaw        bool   `json:"send_raw"`
	SecurityPolicy string `json:"security_policy,omitempty"`
}

type securityPolicy struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Signature *struct {
		PublicKey string `json:"public_key"`
	} `json:"signature,omitempty"`
}

func policyName(route providers.InboundRoute) string { return "ReputeCast " + route.ID.String() }

// parseSettingOf reads the hostname's parse setting; nil when there is none.
func (p *Provider) parseSettingOf(ctx context.Context, hostname string) (*parseSetting, error) {
	var s parseSetting
	err := p.api.Do(ctx, "GET /v3/user/webhooks/parse/settings/{hostname}", http.MethodGet, apiclient.Path("/v3/user/webhooks/parse/settings/%s", hostname), nil, &s)
	if apiclient.NotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (p *Provider) policies(ctx context.Context) ([]securityPolicy, error) {
	var out struct {
		Policies []securityPolicy `json:"policies"`
	}
	err := p.api.Do(ctx, "GET /v3/user/webhooks/security/policies", http.MethodGet, "/v3/user/webhooks/security/policies", nil, &out)
	return out.Policies, err
}

// unavailableFeature reports a refusal that means the account (or the key)
// can't use security policies, not that it can't receive.
func unavailableFeature(err error) bool {
	var ce *providers.CallError
	return errors.As(err, &ce) && (ce.Status == http.StatusUnauthorized || ce.Status == http.StatusForbidden || ce.Status == http.StatusNotFound)
}

// signingPolicy returns the id of the route's signing policy, created when
// missing, and its public key; "" when the account can't have one.
func (p *Provider) signingPolicy(ctx context.Context, route providers.InboundRoute) (string, string, error) {
	list, err := p.policies(ctx)
	if unavailableFeature(err) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	for _, sp := range list {
		if sp.Name == policyName(route) && sp.Signature != nil && sp.Signature.PublicKey != "" {
			return sp.ID, sp.Signature.PublicKey, nil
		}
	}
	var created struct {
		Policy securityPolicy `json:"policy"`
	}
	body := map[string]any{"name": policyName(route), "signature": map[string]bool{"enabled": true}}
	err = p.api.Do(ctx, "POST /v3/user/webhooks/security/policies", http.MethodPost, "/v3/user/webhooks/security/policies", body, &created)
	if unavailableFeature(err) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if created.Policy.Signature == nil {
		return created.Policy.ID, "", nil
	}
	return created.Policy.ID, created.Policy.Signature.PublicKey, nil
}

// RegisterInbound implements providers.InboundRegistrar: the hostname's
// parse setting (POST, or PATCH when it exists) with the raw message, spam
// checking and the hook's basic authentication, then the route's signing
// policy attached to it.
func (p *Provider) RegisterInbound(ctx context.Context, route providers.InboundRoute, hookURL string) error {
	target, err := received.HookURL(hookURL, route.Secret)
	if err != nil {
		return err
	}
	host := strings.ToLower(route.Domain)
	existing, err := p.parseSettingOf(ctx, host)
	if err != nil {
		return err
	}
	setting := parseSetting{URL: target, Hostname: host, SpamCheck: true, SendRaw: true}
	if existing == nil {
		if err := p.api.Do(ctx, "POST /v3/user/webhooks/parse/settings", http.MethodPost, "/v3/user/webhooks/parse/settings", setting, nil); err != nil {
			return err
		}
	}
	id, key, err := p.signingPolicy(ctx, route)
	if err != nil {
		return err
	}
	if existing == nil && id == "" {
		parseKeys.Set(p.parseKeyID(host), unsigned)
		return nil
	}
	patch := map[string]any{"url": setting.URL, "spam_check": true, "send_raw": true}
	if id != "" {
		patch["security_policy"] = id
	}
	if err := p.api.Do(ctx, "PATCH /v3/user/webhooks/parse/settings/{hostname}", http.MethodPatch, apiclient.Path("/v3/user/webhooks/parse/settings/%s", host), patch, nil); err != nil {
		return err
	}
	if key == "" {
		key = unsigned
	}
	parseKeys.Set(p.parseKeyID(host), key)
	return nil
}

// UnregisterInbound implements providers.InboundRegistrar: the hostname's
// parse setting goes when it still posts to ReputeCast, and so does the
// route's signing policy.
func (p *Provider) UnregisterInbound(ctx context.Context, route providers.InboundRoute, hookURL string) error {
	host := strings.ToLower(route.Domain)
	s, err := p.parseSettingOf(ctx, host)
	if err != nil {
		return err
	}
	if s != nil && received.SameURL(s.URL, hookURL) {
		err := p.api.Do(ctx, "DELETE /v3/user/webhooks/parse/settings/{hostname}", http.MethodDelete, apiclient.Path("/v3/user/webhooks/parse/settings/%s", host), nil, nil)
		if err != nil && !apiclient.NotFound(err) {
			return err
		}
	}
	list, err := p.policies(ctx)
	if unavailableFeature(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, sp := range list {
		if sp.Name != policyName(route) {
			continue
		}
		err := p.api.Do(ctx, "DELETE /v3/user/webhooks/security/policies/{id}", http.MethodDelete, apiclient.Path("/v3/user/webhooks/security/policies/%s", sp.ID), nil, nil)
		if err != nil && !apiclient.NotFound(err) {
			return err
		}
	}
	return nil
}

// fetchParseKey reads the public key of the policy attached to the
// hostname's parse setting, or unsigned when it has none.
func (p *Provider) fetchParseKey(ctx context.Context, hostname string) (string, error) {
	s, err := p.parseSettingOf(ctx, hostname)
	if err != nil {
		return "", err
	}
	if s == nil {
		return "", fmt.Errorf("SendGrid has no Inbound Parse setting for %s", hostname)
	}
	if s.SecurityPolicy == "" {
		return unsigned, nil
	}
	var out struct {
		Policy securityPolicy `json:"policy"`
	}
	if err := p.api.Do(ctx, "GET /v3/user/webhooks/security/policies/{id}", http.MethodGet, apiclient.Path("/v3/user/webhooks/security/policies/%s", s.SecurityPolicy), nil, &out); err != nil {
		return "", err
	}
	if out.Policy.Signature == nil || out.Policy.Signature.PublicKey == "" {
		return unsigned, nil
	}
	return out.Policy.Signature.PublicKey, nil
}

// VerifyInbound implements providers.InboundReceiver: the hook's basic
// authentication, and the post's signature when the parse setting has a
// signing policy.
func (p *Provider) VerifyInbound(ctx context.Context, route providers.InboundRoute, r *http.Request, body []byte) error {
	if err := received.CheckBasic(r, route); err != nil {
		return err
	}
	host := strings.ToLower(route.Domain)
	sig, ts := r.Header.Get(SignatureHeader), r.Header.Get(TimestampHeader)
	return parseKeys.Verify(ctx, p.parseKeyID(host), func(ctx context.Context) (string, error) { return p.fetchParseKey(ctx, host) }, func(key string) error {
		if key == unsigned {
			return nil
		}
		if sig == "" || ts == "" {
			return fmt.Errorf("%w: the parse setting signs its posts but this one isn't signed", providers.ErrBadSignature)
		}
		if err := Verify(key, sig, ts, body); err != nil {
			return err
		}
		return webhookauth.Within(ts, now(), ParseWindow)
	})
}

// ParseInbound implements providers.InboundReceiver: email is the raw
// message, envelope's to the recipients, spam_score SpamAssassin's score,
// SPF and dkim SendGrid's checks.
func (p *Provider) ParseInbound(_ context.Context, _ providers.InboundRoute, r *http.Request, body []byte) ([]providers.InboundMessage, error) {
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || params["boundary"] == "" {
		return nil, fmt.Errorf("sendgrid inbound parse: not multipart form data")
	}
	form, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(int64(len(body)) + 1)
	if err != nil {
		return nil, fmt.Errorf("sendgrid inbound parse: %w", err)
	}
	defer func() { _ = form.RemoveAll() }()
	field := func(k string) string {
		if v := form.Value[k]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	raw := field("email")
	if raw == "" {
		return nil, fmt.Errorf("sendgrid inbound parse: no email field; the parse setting must post the raw message")
	}
	m := providers.InboundMessage{Raw: []byte(raw), Recipients: []string{},
		Spam: received.SpamScore(field("spam_score"), received.SpamAssassin), SPF: received.Verdict(field("SPF")), DKIM: dkimResult(field("dkim"))}
	var env struct {
		To []string `json:"to"`
	}
	if json.Unmarshal([]byte(field("envelope")), &env) == nil {
		for _, a := range env.To {
			if a = strings.ToLower(strings.TrimSpace(a)); a != "" {
				m.Recipients = append(m.Recipients, a)
			}
		}
	}
	return []providers.InboundMessage{m}, nil
}

// dkimResult reads Inbound Parse's dkim field ("{@sendgrid.com : pass}",
// one result per signature): pass when one passes and none fails.
func dkimResult(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "fail"):
		return "fail"
	case strings.Contains(s, "pass"):
		return "pass"
	case strings.Contains(s, "none"):
		return "none"
	}
	return ""
}
