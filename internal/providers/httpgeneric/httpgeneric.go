// Package httpgeneric is the generic HTTP adapter (spec 5.2). It sends
// through any HTTP API described by a mapping instead of Go code: a request
// template (method, URL, headers and a JSON body with Liquid-style fields),
// where the response carries the provider's message id, and how the
// provider's webhooks are authenticated and read.
//
// A mapping is the "http" object of an HTTP adapter's manifest — the format
// the Adapters tab saves into installed_adapters and that community
// manifests ship:
//
//	{"method": "POST", "url": "https://api.example-esp.com/v1/messages",
//	 "headers": {"Authorization": "Bearer {{ secrets.api_key }}"},
//	 "body": "{\"from\": \"{{ sender }}\", \"to\": \"{{ recipient }}\", \"subject\": \"{{ subject }}\", \"html\": \"{{ html }}\", \"headers\": {{ headers_json }}}",
//	 "message_id_path": "$.data.id",
//	 "webhook_mapping": "status",
//	 "webhook": {"auth": "hmac", "signature_header": "X-Signature"}}
//
// A provider connected through the adapter keeps its mapping in its
// settings ({"http": {…}}), or is opened from an installed manifest with
// FromManifest. docs/adapters.md documents every field.
package httpgeneric

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "httpgeneric"

// Mapping describes one provider's HTTP API. JSON names follow the
// manifest's "http" object.
type Mapping struct {
	// Method is POST (default), PUT or PATCH.
	Method string `json:"method,omitempty"`
	// URL is https:// (http:// only for a loopback address). Placeholders may
	// appear in the path and query, never in the host.
	URL string `json:"url"`
	// Headers are header templates ("Bearer {{ secrets.api_key }}").
	// Content-Type defaults to application/json.
	Headers map[string]string `json:"headers,omitempty"`
	// Body is the JSON request body template.
	Body string `json:"body"`
	// MessageIDPath is the JSONPath of the provider's message id in a 2xx
	// response body ("$.data.id"); MessageIDHeader names a response header
	// that carries it instead ("X-Message-Id"). Without either, the
	// ReputeCast message id stands in for it.
	MessageIDPath   string `json:"message_id_path,omitempty"`
	MessageIDHeader string `json:"message_id_header,omitempty"`
	// ErrorCodePath and ErrorMessagePath read the provider's error code and
	// message from a failed response ("$.error.code", "$.error.message").
	ErrorCodePath    string `json:"error_code_path,omitempty"`
	ErrorMessagePath string `json:"error_message_path,omitempty"`
	// WebhookMapping is "status" (the payload's status field says
	// delivered, bounced, complained…; the default) or "jsonpath" (the
	// paths and type map in Webhook).
	WebhookMapping string `json:"webhook_mapping,omitempty"`
	// Webhook says how webhook requests are authenticated and read.
	Webhook WebhookSpec `json:"webhook,omitzero"`
	// Check is an optional read-only request ReputeCast makes to check the
	// credentials at connect time and the API's health every minute.
	// Without it, health comes from send results alone.
	Check *CheckSpec `json:"check,omitempty"`
	// Fields are the connection values the templates read beyond the
	// adapter's own (secrets.api_key, secrets.webhook_signing_key): a field
	// marked secret is read as secrets.<key>, others as settings.<key>.
	Fields []providers.Field `json:"fields,omitempty"`
}

// CheckSpec is a GET request that succeeds (2xx) while the credentials and
// the API are fine ("https://api.example-esp.com/v1/account").
type CheckSpec struct {
	URL string `json:"url"`
	// Headers default to the send request's headers (its authentication).
	Headers map[string]string `json:"headers,omitempty"`
}

// Terms is what an HTTP adapter's manifest says about the provider's terms.
type Terms struct {
	Bulk providers.BulkPolicy `json:"bulk"`
	Note string               `json:"note,omitempty"`
	// URL is the provider's acceptable-use policy.
	URL string `json:"url,omitempty"`
}

// Manifest is an installed HTTP adapter's manifest.json (the
// installed_adapters.manifest column).
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Maintainer   string   `json:"maintainer"`
	Kind         string   `json:"kind"`
	Capabilities []string `json:"capabilities,omitempty"`
	SourceURL    string   `json:"source_url,omitempty"`
	Terms        *Terms   `json:"terms,omitempty"`
	// Pricing is the provider's pricing model ("$0.80 per 1,000").
	Pricing string  `json:"pricing,omitempty"`
	HTTP    Mapping `json:"http"`
}

// Settings are a connected provider's non-secret settings (providers.config).
// Every other top-level string or number is a connection value the
// templates read as settings.<key>.
type Settings struct {
	// Adapter names the installed HTTP adapter the mapping came from.
	Adapter string   `json:"adapter,omitempty"`
	HTTP    *Mapping `json:"http"`
	Terms   *Terms   `json:"terms,omitempty"`
	Pricing string   `json:"pricing,omitempty"`
}

// reserved settings keys are not template values.
var reserved = map[string]bool{"adapter": true, "http": true, "terms": true, "pricing": true}

func init() { providers.Register(Descriptor()) }

// defaultTerms applies when a mapping's manifest says nothing about terms.
var defaultTerms = Terms{Bulk: providers.BulkYourPolicy,
	Note: "ReputeCast can't read the terms of a provider connected through an HTTP mapping. Check its acceptable-use policy before routing Broadcasts to it."}

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Generic HTTP adapter",
		Kind:    "http",
		Capabilities: providers.Capabilities{
			Bulk:      providers.BulkYourPolicy,
			Webhooks:  "Webhooks",
			BatchSize: 1,
			Pricing:   "set by your provider",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: defaultTerms.Bulk, Note: defaultTerms.Note,
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Mapped webhooks",
		DomainSetup:     "Manual DNS",
		SuppressionMode: "Local only",
		Fields: []providers.Field{
			{Key: "api_key", Label: "API key", Secret: true, Help: "Read by the mapping's templates as {{ secrets.api_key }}."},
			{Key: "webhook_signing_key", Label: "Webhook signing secret", Secret: true,
				Help: "When the provider signs webhooks with a secret of its own. Without it, ReputeCast's webhook secret for this provider is used."},
		},
		New: New,
	}
}

// Provider sends through one mapped HTTP API.
type Provider struct {
	cfg      providers.Config
	name     string
	m        Mapping
	terms    Terms
	pricing  string
	settings map[string]string
	secrets  map[string]string
	c        compiled
	api      *apiclient.Client
}

var (
	_ providers.Provider       = (*Provider)(nil)
	_ providers.WebhookSecrets = (*Provider)(nil)
)

// compiled is a parsed mapping.
type compiled struct {
	method   string
	base     string // scheme://host[:port]
	path     *template
	headers  []headerTemplate
	body     *template
	msgID    *jsonPath
	errCode  *jsonPath
	errMsg   *jsonPath
	hook     *hookSpec
	hasHooks bool
	check    *checkRequest
}

type checkRequest struct {
	base    string
	path    *template
	headers []headerTemplate
}

type headerTemplate struct {
	name string
	t    *template
}

// New builds the adapter from a connected provider whose settings hold the
// mapping ({"http": {…}}).
func New(cfg providers.Config) (providers.Provider, error) {
	var s Settings
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &s); err != nil {
			return nil, fmt.Errorf("httpgeneric settings: %w", err)
		}
	}
	if s.HTTP == nil {
		return nil, errors.New("httpgeneric: the provider's settings have no HTTP mapping")
	}
	terms := defaultTerms
	if s.Terms != nil && s.Terms.Bulk != "" {
		terms = *s.Terms
	}
	return open(*s.HTTP, s.Adapter, terms, s.Pricing, cfg)
}

// FromManifest builds the adapter for a provider connected through an
// installed HTTP adapter: the mapping, terms and pricing come from the
// manifest, the connection values and secrets from cfg.
func FromManifest(raw []byte, cfg providers.Config) (providers.Provider, error) {
	m, err := ParseManifest(raw)
	if err != nil {
		return nil, err
	}
	terms := defaultTerms
	if m.Terms != nil && m.Terms.Bulk != "" {
		terms = *m.Terms
	}
	return open(m.HTTP, m.Name, terms, m.Pricing, cfg)
}

// ParseManifest reads and validates an HTTP adapter's manifest.
func ParseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, fmt.Errorf("the manifest is not valid JSON: %w", err)
	}
	if m.Kind != "" && m.Kind != "http" {
		return m, fmt.Errorf("the manifest describes a %s adapter, not an HTTP mapping", m.Kind)
	}
	if m.Terms != nil {
		if err := validBulk(m.Terms.Bulk); err != nil {
			return m, err
		}
	}
	if err := m.HTTP.Validate(); err != nil {
		return m, err
	}
	return m, nil
}

func validBulk(b providers.BulkPolicy) error {
	switch b {
	case "", providers.BulkAllowed, providers.BulkBroadcastStreams, providers.BulkForbidden, providers.BulkYourPolicy:
		return nil
	}
	return fmt.Errorf("terms.bulk %q is not one of allowed, broadcast_streams_only, forbidden or your_policy", b)
}

func open(m Mapping, name string, terms Terms, pricing string, cfg providers.Config) (providers.Provider, error) {
	c, err := m.compile()
	if err != nil {
		return nil, err
	}
	if err := validBulk(terms.Bulk); err != nil {
		return nil, err
	}
	p := &Provider{cfg: cfg, name: name, m: m, terms: terms, pricing: pricing, c: c, settings: map[string]string{}, secrets: map[string]string{}}
	if len(cfg.Settings) > 0 {
		var all map[string]any
		if err := json.Unmarshal(cfg.Settings, &all); err == nil {
			for k, v := range all {
				if reserved[k] {
					continue
				}
				switch v.(type) {
				case string, float64, bool:
					p.settings[k] = stringify(v)
				}
			}
		}
	}
	if len(cfg.Credentials) > 0 {
		var all map[string]any
		if err := json.Unmarshal(cfg.Credentials, &all); err != nil {
			return nil, fmt.Errorf("httpgeneric credentials: %w", err)
		}
		for k, v := range all {
			if s, ok := v.(string); ok {
				p.secrets[k] = s
			}
		}
	}
	p.api = apiclient.New(c.base, cfg.HTTPClient)
	p.api.Decode = p.decodeError
	if c.check != nil {
		return &Checked{Provider: p}, nil
	}
	return p, nil
}

// Validate checks a mapping: method, URL, header names, that the body
// template renders to JSON, and the JSONPaths. Errors are sentences that
// can be shown next to the field.
func (m Mapping) Validate() error {
	c, err := m.compile()
	if err != nil {
		return err
	}
	sample := templateData(sampleMessage(), map[string]string{}, map[string]string{})
	if c.body != nil {
		out, err := c.body.render(sample)
		if err != nil {
			return &FieldError{Field: "body", Message: err.Error()}
		}
		if !json.Valid([]byte(out)) {
			return &FieldError{Field: "body", Message: "The body template isn't valid JSON once its {{ }} fields are filled in."}
		}
	}
	return nil
}

// FieldError is a mapping problem with the field it concerns.
type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }

func (m Mapping) compile() (compiled, error) {
	var c compiled
	c.method = strings.ToUpper(strings.TrimSpace(m.Method))
	if c.method == "" {
		c.method = http.MethodPost
	}
	if !slices.Contains([]string{http.MethodPost, http.MethodPut, http.MethodPatch}, c.method) {
		return c, &FieldError{Field: "method", Message: "Use POST, PUT or PATCH."}
	}
	base, rest, err := splitURL(m.URL)
	if err != nil {
		return c, &FieldError{Field: "url", Message: err.Error()}
	}
	c.base = base
	if c.path, err = parseTemplate(rest, modeURL); err != nil {
		return c, &FieldError{Field: "url", Message: err.Error()}
	}
	if c.headers, err = compileHeaders(m.Headers, "headers"); err != nil {
		return c, err
	}
	if strings.TrimSpace(m.Body) == "" {
		return c, &FieldError{Field: "body", Message: "Enter the request body template."}
	}
	if c.body, err = parseTemplate(m.Body, modeJSON); err != nil {
		return c, &FieldError{Field: "body", Message: err.Error()}
	}
	if m.MessageIDPath != "" && m.MessageIDHeader != "" {
		return c, &FieldError{Field: "message_id_path", Message: "Read the message id from the response body or from a header, not both."}
	}
	if c.msgID, err = parsePath(m.MessageIDPath); err != nil {
		return c, &FieldError{Field: "message_id_path", Message: err.Error()}
	}
	if c.errCode, err = parsePath(m.ErrorCodePath); err != nil {
		return c, &FieldError{Field: "error_code_path", Message: err.Error()}
	}
	if c.errMsg, err = parsePath(m.ErrorMessagePath); err != nil {
		return c, &FieldError{Field: "error_message_path", Message: err.Error()}
	}
	if c.hook, err = m.compileWebhook(); err != nil {
		return c, err
	}
	c.hasHooks = c.hook != nil
	if m.Check != nil && strings.TrimSpace(m.Check.URL) != "" {
		ck := &checkRequest{headers: c.headers}
		var rest string
		if ck.base, rest, err = splitURL(m.Check.URL); err != nil {
			return c, &FieldError{Field: "check.url", Message: err.Error()}
		}
		if ck.path, err = parseTemplate(rest, modeURL); err != nil {
			return c, &FieldError{Field: "check.url", Message: err.Error()}
		}
		if len(m.Check.Headers) > 0 {
			if ck.headers, err = compileHeaders(m.Check.Headers, "check.headers"); err != nil {
				return c, err
			}
		}
		c.check = ck
	}
	return c, nil
}

func compileHeaders(hs map[string]string, field string) ([]headerTemplate, error) {
	out := []headerTemplate{}
	for _, name := range slices.Sorted(maps.Keys(hs)) {
		if !validHeaderName(name) {
			return nil, &FieldError{Field: field, Message: fmt.Sprintf("%q is not a valid header name.", name)}
		}
		switch strings.ToLower(name) {
		case "host", "content-length", "transfer-encoding", "connection":
			return nil, &FieldError{Field: field, Message: name + " is set by ReputeCast and can't be mapped."}
		}
		t, err := parseTemplate(hs[name], modeText)
		if err != nil {
			return nil, &FieldError{Field: field, Message: name + ": " + err.Error()}
		}
		out = append(out, headerTemplate{name: name, t: t})
	}
	return out, nil
}

// sentence is an error whose text is shown to the person as it is.
type sentence string

func (s sentence) Error() string { return string(s) }

// splitURL separates the literal scheme and host from the path template.
func splitURL(raw string) (base, rest string, err error) {
	s := strings.TrimSpace(raw)
	scheme, after, ok := strings.Cut(s, "://")
	if !ok || (scheme != "https" && scheme != "http") {
		return "", "", sentence("Enter an https:// URL, for example https://api.example-esp.com/v1/messages.")
	}
	end := strings.IndexAny(after, "/?#")
	if end < 0 {
		end = len(after)
	}
	host := after[:end]
	if strings.Contains(host, "{{") || strings.Contains(host, "}}") {
		return "", "", sentence("The host must be written out; {{ }} fields may only appear in the path and query.")
	}
	u, perr := url.Parse(scheme + "://" + host)
	if perr != nil || u.Host == "" || u.User != nil {
		return "", "", sentence("Enter an https:// URL, for example https://api.example-esp.com/v1/messages.")
	}
	if scheme == "http" && !loopback(u.Hostname()) {
		return "", "", sentence("Use https://; plain http:// is only allowed for a local test server.")
	}
	rest = after[end:]
	if strings.Contains(rest, "#") {
		return "", "", sentence("Remove the #fragment from the URL.")
	}
	if rest == "" {
		rest = "/"
	}
	return scheme + "://" + u.Host, rest, nil
}

func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validHeaderName(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		if r <= 32 || r >= 127 || strings.ContainsRune(`()<>@,;:\"/[]?={}`, r) {
			return false
		}
	}
	return true
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Name is the installed HTTP adapter the provider uses ("" when the mapping
// was entered for this provider only).
func (p *Provider) Name() string { return p.name }

// Capabilities implements providers.Provider: what the mapping can do.
func (p *Provider) Capabilities() providers.Capabilities {
	c := Descriptor().Capabilities
	c.Bulk = p.terms.Bulk
	if !p.c.hasHooks {
		c.Webhooks = ""
	}
	if p.pricing != "" {
		c.Pricing = p.pricing
	}
	return c
}

// Terms returns what the mapping's manifest says about the provider's terms.
func (p *Provider) Terms() Terms { return p.terms }

// templateData is what the templates can read for one message.
func templateData(m providers.Message, settings, secrets map[string]string) map[string]any {
	headers := map[string]any{mimemsg.HeaderMessage: m.ID.String()}
	for k, v := range m.Headers {
		headers[k] = v
	}
	list := make([]any, 0, len(headers))
	for _, k := range slices.Sorted(maps.Keys(headers)) {
		list = append(list, map[string]any{"name": k, "value": headers[k]})
	}
	tags := map[string]any{}
	for k, v := range m.Tags {
		tags[k] = v
	}
	replyTo, replyToName := "", ""
	if m.ReplyTo != nil {
		replyTo, replyToName = m.ReplyTo.Address, m.ReplyTo.Name
	}
	return map[string]any{
		"message_id":          m.ID.String(),
		"stream":              string(m.Stream),
		"sender":              mimemsg.Address(m.From),
		"sender_email":        m.From.Address,
		"sender_name":         m.From.Name,
		"recipient":           m.To.Address,
		"recipient_name":      m.To.Name,
		"recipient_formatted": mimemsg.Address(m.To),
		"reply_to":            replyTo,
		"reply_to_name":       replyToName,
		"subject":             m.Subject,
		"html":                m.HTML,
		"text":                m.Text,
		"headers":             headers,
		"headers_json":        headers,
		"headers_list":        list,
		"tags":                tags,
		"settings":            settings,
		"secrets":             secrets,
	}
}

// Rendered is a request built from the mapping, for the "Test with a sample
// message" preview. Secret values are replaced by "••••".
type Rendered struct {
	Method string
	URL    string
	Header http.Header
	Body   string
}

// Preview renders the request for msg without sending it, with secrets masked.
func (p *Provider) Preview(msg providers.Message) (Rendered, error) {
	masked := map[string]string{}
	for k := range p.secrets {
		masked[k] = "••••"
	}
	path, header, body, err := p.render(msg, masked)
	if err != nil {
		return Rendered{}, err
	}
	return Rendered{Method: p.c.method, URL: p.c.base + path, Header: header, Body: string(body)}, nil
}

func (p *Provider) render(msg providers.Message, secrets map[string]string) (string, http.Header, []byte, error) {
	data := templateData(msg, p.settings, secrets)
	path, err := p.c.path.render(data)
	if err != nil {
		return "", nil, nil, err
	}
	header := http.Header{}
	for _, h := range p.c.headers {
		v, err := h.t.render(data)
		if err != nil {
			return "", nil, nil, err
		}
		header.Set(h.name, v)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
	body, err := p.c.body.render(data)
	if err != nil {
		return "", nil, nil, err
	}
	if !json.Valid([]byte(body)) {
		return "", nil, nil, errors.New("the body template didn't produce valid JSON for this message")
	}
	return path, header, []byte(body), nil
}

// Send implements providers.Provider: one request per message.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	op := p.c.method + " " + p.c.base
	results := make([]providers.SendResult, 0, len(msgs))
	for i, m := range msgs {
		path, header, body, err := p.render(m, p.secrets)
		if err != nil {
			results = append(results, providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: "Template", Message: err.Error()}})
			continue
		}
		res, err := p.api.Exec(ctx, apiclient.Request{Op: op, Method: p.c.method, Path: path, ContentType: header.Get("Content-Type"), Body: body, Header: header})
		if err != nil {
			var ce *providers.CallError
			if !errors.As(err, &ce) {
				return nil, err
			}
			if ce.Auth || ce.Temporary && !ce.Throttled {
				// The credentials or the service failed: nothing of this
				// call went out. Fail it, or the rest of it when earlier
				// messages were accepted.
				if i == 0 {
					return nil, ce
				}
				for _, rest := range msgs[i:] {
					results = append(results, providers.SendResult{MessageID: rest.ID, Err: &providers.SendError{Temporary: true, Code: errCode(ce), Message: ce.Error()}})
				}
				return results, nil
			}
			results = append(results, providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Temporary: ce.Throttled, Throttled: ce.Throttled, Code: errCode(ce), Message: ce.Message}})
			continue
		}
		results = append(results, providers.SendResult{MessageID: m.ID, ProviderMessageID: p.messageID(m, res)})
	}
	return results, nil
}

// decodeError reads the provider's error code and message from a failed
// response where the mapping says they are; otherwise the body is the message.
func (p *Provider) decodeError(_ int, body []byte) (string, string) {
	raw := strings.TrimSpace(string(body))
	if p.c.errCode == nil && p.c.errMsg == nil {
		return "", raw
	}
	doc, err := decodeJSON(body)
	if err != nil {
		return "", raw
	}
	code, msg := p.c.errCode.text(doc), p.c.errMsg.text(doc)
	if msg == "" {
		msg = raw
	}
	return code, msg
}

func errCode(ce *providers.CallError) string {
	if ce.Code != "" {
		return ce.Code
	}
	if ce.Status != 0 {
		return fmt.Sprintf("HTTP %d", ce.Status)
	}
	return "Unavailable"
}

func (p *Provider) messageID(m providers.Message, res *apiclient.Response) string {
	if p.m.MessageIDHeader != "" {
		if v := strings.Trim(strings.TrimSpace(res.Header.Get(p.m.MessageIDHeader)), "<>"); v != "" {
			return v
		}
	}
	if p.c.msgID != nil {
		if doc, err := decodeJSON(res.Body); err == nil {
			if v := strings.Trim(p.c.msgID.text(doc), "<>"); v != "" {
				return v
			}
		}
	}
	// The provider names no id: events find the message through the
	// ReputeCast id the mapping passes along (the reputecast_id_path).
	return m.ID.String()
}

// sampleMessage is the message used to check templates and for previews.
func sampleMessage() providers.Message {
	id := uuid.MustParse("0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40")
	return providers.Message{
		ID:      id,
		Stream:  providers.StreamBroadcast,
		From:    mail.Address{Name: "Acme Newsletter", Address: "news@news.acme.io"},
		To:      mail.Address{Name: "Linh Nguyen", Address: "linh@example.com"},
		Subject: "Your October update",
		HTML:    `<p>Hello "Linh" — this is a <b>sample</b>.</p>`,
		Text:    "Hello \"Linh\" — this is a sample.",
		Headers: map[string]string{
			"List-Unsubscribe":      "<https://news.acme.io/p/acme/unsubscribe/abc>",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		},
		Tags: map[string]string{"campaign": "sample"},
	}
}

// SampleMessage is the message "Test with a sample message" renders.
func SampleMessage() providers.Message { return sampleMessage() }

// Checked is the adapter for a mapping with a check request: it also
// implements providers.Verifier, so credentials are checked at connect time
// and the health job probes the API. A mapping without one has no Verify:
// a probe that can't fail would hide an outage.
type Checked struct {
	*Provider
}

var _ providers.Verifier = (*Checked)(nil)

// Verify implements providers.Verifier with the mapping's check request.
func (c *Checked) Verify(ctx context.Context) (providers.Quota, error) {
	p := c.Provider
	ck := p.c.check
	data := templateData(sampleMessage(), p.settings, p.secrets)
	path, err := ck.path.render(data)
	if err != nil {
		return providers.Quota{}, err
	}
	header := http.Header{}
	for _, h := range ck.headers {
		v, err := h.t.render(data)
		if err != nil {
			return providers.Quota{}, err
		}
		header.Set(h.name, v)
	}
	api := apiclient.New(ck.base, p.cfg.HTTPClient)
	api.Decode = p.decodeError
	if _, err := api.Exec(ctx, apiclient.Request{Op: "GET " + ck.base, Method: http.MethodGet, Path: path, Header: header}); err != nil {
		return providers.Quota{}, err
	}
	acct := "HTTP mapping"
	if p.name != "" {
		acct = p.name
	}
	return providers.Quota{Account: acct + " · reachable"}, nil
}

// now is the clock, replaced in tests.
var now = time.Now
