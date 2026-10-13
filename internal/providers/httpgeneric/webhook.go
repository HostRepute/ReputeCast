package httpgeneric

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"hash"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// WebhookSpec says how a provider's webhook requests are authenticated and
// how their payload maps to ReputeCast events.
type WebhookSpec struct {
	// Auth is how a request proves it comes from the provider:
	//   - "hmac" (the default): a signature header, the HMAC of the body
	//     (or of SignedPayload) with the signing secret;
	//   - "standard": Standard Webhooks / Svix (webhook-id,
	//     webhook-timestamp, webhook-signature; a whsec_ secret);
	//   - "basic": HTTP basic auth, user Username and the secret as password;
	//   - "header": the secret itself in TokenHeader;
	//   - "none": nothing but the unguessable webhook URL.
	Auth string `json:"auth,omitempty"`
	// SignatureHeader carries the signature (default X-Signature).
	SignatureHeader string `json:"signature_header,omitempty"`
	// Algorithm is sha256 (default), sha1 or sha512.
	Algorithm string `json:"algorithm,omitempty"`
	// Encoding of the signature: hex (default) or base64.
	Encoding string `json:"encoding,omitempty"`
	// SignaturePrefix is stripped from the header value ("sha256=").
	SignaturePrefix string `json:"signature_prefix,omitempty"`
	// TimestampHeader names a signed timestamp header; requests more than
	// five minutes old are refused as replays.
	TimestampHeader string `json:"timestamp_header,omitempty"`
	// SignedPayload is what is signed, with {timestamp} and {body}
	// standing for the timestamp header and the raw body. Default
	// "{timestamp}.{body}" with a timestamp header, else "{body}".
	SignedPayload string `json:"signed_payload,omitempty"`
	// Secret is a template for the signing secret ("{{ secrets.signing_key }}").
	// Default: the webhook_signing_key credential, else the webhook secret
	// ReputeCast generated for the provider.
	Secret string `json:"secret,omitempty"`
	// TokenHeader carries the secret for auth "header".
	TokenHeader string `json:"token_header,omitempty"`
	// Username is the basic-auth user for auth "basic" (default reputecast).
	Username string `json:"username,omitempty"`

	// EventsPath selects the events in the payload (default "$": the
	// payload is one event, or an array of events).
	EventsPath string `json:"events_path,omitempty"`
	// Paths inside one event. With webhook_mapping "status" they default
	// to $.status, $.recipient, $.message_id, $.reputecast_id, $.code,
	// $.reason, $.timestamp, $.url and $.user_agent.
	TypePath         string `json:"type_path,omitempty"`
	RecipientPath    string `json:"recipient_path,omitempty"`
	MessageIDPath    string `json:"message_id_path,omitempty"`
	ReputeCastIDPath string `json:"reputecast_id_path,omitempty"`
	CodePath         string `json:"code_path,omitempty"`
	DiagnosticPath   string `json:"diagnostic_path,omitempty"`
	TimePath         string `json:"time_path,omitempty"`
	URLPath          string `json:"url_path,omitempty"`
	UserAgentPath    string `json:"user_agent_path,omitempty"`
	// Types maps the provider's event types to ReputeCast's (delivered,
	// deferred, hard_bounce, soft_bounce, bounce, complaint, rejected, open,
	// click, unsubscribe, ignore). Types not listed fall back to the status
	// vocabulary (delivered, bounced, complained…) and are otherwise ignored.
	Types map[string]string `json:"types,omitempty"`
}

// statusTypes is the vocabulary of webhook_mapping "status" and the
// fallback for a type map.
var statusTypes = map[string]string{
	"delivered": "delivered", "delivery": "delivered",
	"deferred": "deferred", "delayed": "deferred", "delivery_delayed": "deferred",
	"soft_bounce": "soft_bounce", "soft_bounced": "soft_bounce", "softbounce": "soft_bounce", "temporary_failure": "soft_bounce",
	"bounced": "bounce", "bounce": "bounce", "failed": "bounce",
	"hard_bounce": "hard_bounce", "hard_bounced": "hard_bounce", "hardbounce": "hard_bounce", "permanent_failure": "hard_bounce",
	"complained": "complaint", "complaint": "complaint", "spam": "complaint", "spam_report": "complaint", "spamreport": "complaint",
	"rejected": "rejected", "dropped": "rejected", "blocked": "rejected",
	"opened": "open", "open": "open", "clicked": "click", "click": "click",
	"unsubscribed": "unsubscribe", "unsubscribe": "unsubscribe",
}

var normalizedTypes = map[string]bool{"delivered": true, "deferred": true, "hard_bounce": true, "soft_bounce": true, "bounce": true,
	"complaint": true, "rejected": true, "open": true, "click": true, "unsubscribe": true, "ignore": true}

// hookSpec is a compiled WebhookSpec.
type hookSpec struct {
	auth      string
	header    string
	hash      func() hash.Hash
	encoding  string
	prefix    string
	tsHeader  string
	payload   string
	secret    *template
	token     string
	user      string
	events    *jsonPath
	typ       *jsonPath
	recipient []*jsonPath
	msgID     *jsonPath
	rcID      *jsonPath
	code      *jsonPath
	diag      *jsonPath
	at        *jsonPath
	url       *jsonPath
	ua        *jsonPath
	types     map[string]string
}

func (m Mapping) compileWebhook() (*hookSpec, error) {
	w := m.Webhook
	mode := strings.TrimSpace(m.WebhookMapping)
	if mode == "" && w.TypePath == "" && len(w.Types) == 0 && w.Auth == "" && w.SignatureHeader == "" {
		return nil, nil // no webhooks: events come from nowhere
	}
	if mode == "" {
		mode = "status"
		if w.TypePath != "" {
			mode = "jsonpath"
		}
	}
	if mode != "status" && mode != "jsonpath" {
		return nil, &FieldError{Field: "webhook_mapping", Message: "Use status or jsonpath."}
	}
	h := &hookSpec{auth: strings.ToLower(strings.TrimSpace(w.Auth)), header: w.SignatureHeader, encoding: w.Encoding, prefix: w.SignaturePrefix,
		tsHeader: w.TimestampHeader, payload: w.SignedPayload, token: w.TokenHeader, user: w.Username, types: map[string]string{}}
	if h.auth == "" {
		h.auth = "hmac"
	}
	switch h.auth {
	case "hmac":
		if h.header == "" {
			h.header = "X-Signature"
		}
		var err error
		if h.hash, err = webhookauth.Hash(w.Algorithm); err != nil {
			return nil, &FieldError{Field: "webhook.algorithm", Message: err.Error()}
		}
		switch strings.ToLower(h.encoding) {
		case "", "hex", "base64":
		default:
			return nil, &FieldError{Field: "webhook.encoding", Message: "Use hex or base64."}
		}
		if h.payload == "" {
			h.payload = "{body}"
			if h.tsHeader != "" {
				h.payload = "{timestamp}.{body}"
			}
		}
		if !strings.Contains(h.payload, "{body}") {
			return nil, &FieldError{Field: "webhook.signed_payload", Message: "The signed payload must include {body}."}
		}
		if strings.Contains(h.payload, "{timestamp}") && h.tsHeader == "" {
			return nil, &FieldError{Field: "webhook.signed_payload", Message: "Name the timestamp header the {timestamp} comes from."}
		}
	case "standard", "none":
	case "basic":
		if h.user == "" {
			h.user = "reputecast"
		}
	case "header":
		if !validHeaderName(h.token) {
			return nil, &FieldError{Field: "webhook.token_header", Message: "Name the header that carries the secret."}
		}
	default:
		return nil, &FieldError{Field: "webhook.auth", Message: "Use hmac, standard, basic, header or none."}
	}
	if strings.TrimSpace(w.Secret) != "" {
		t, err := parseTemplate(w.Secret, modeText)
		if err != nil {
			return nil, &FieldError{Field: "webhook.secret", Message: err.Error()}
		}
		h.secret = t
	}
	paths := []struct {
		dst      **jsonPath
		src, def string
		field    string
	}{
		{&h.events, w.EventsPath, "$", "webhook.events_path"},
		{&h.typ, w.TypePath, "$.status", "webhook.type_path"},
		{&h.msgID, w.MessageIDPath, "$.message_id", "webhook.message_id_path"},
		{&h.rcID, w.ReputeCastIDPath, "$.reputecast_id", "webhook.reputecast_id_path"},
		{&h.code, w.CodePath, "$.code", "webhook.code_path"},
		{&h.diag, w.DiagnosticPath, "$.reason", "webhook.diagnostic_path"},
		{&h.at, w.TimePath, "$.timestamp", "webhook.time_path"},
		{&h.url, w.URLPath, "$.url", "webhook.url_path"},
		{&h.ua, w.UserAgentPath, "$.user_agent", "webhook.user_agent_path"},
	}
	for _, p := range paths {
		src := p.src
		if src == "" && (mode == "status" || p.field == "webhook.events_path") {
			src = p.def
		}
		jp, err := parsePath(src)
		if err != nil {
			return nil, &FieldError{Field: p.field, Message: err.Error()}
		}
		*p.dst = jp
	}
	if h.typ == nil {
		return nil, &FieldError{Field: "webhook.type_path", Message: "Enter the JSONPath of the event type, for example $.event."}
	}
	recipients := []string{w.RecipientPath}
	if w.RecipientPath == "" && mode == "status" {
		recipients = []string{"$.recipient", "$.email"}
	}
	for _, src := range recipients {
		jp, err := parsePath(src)
		if err != nil {
			return nil, &FieldError{Field: "webhook.recipient_path", Message: err.Error()}
		}
		if jp != nil {
			h.recipient = append(h.recipient, jp)
		}
	}
	for k, v := range w.Types {
		v = strings.ToLower(strings.TrimSpace(v))
		if !normalizedTypes[v] {
			return nil, &FieldError{Field: "webhook.types", Message: fmt.Sprintf("%q maps to %q; use delivered, deferred, hard_bounce, soft_bounce, bounce, complaint, rejected, open, click, unsubscribe or ignore.", k, v)}
		}
		h.types[strings.ToLower(strings.TrimSpace(k))] = v
	}
	return h, nil
}

// signingSecret is the secret webhook requests are checked with.
func (p *Provider) signingSecret() string {
	h := p.c.hook
	if h.secret != nil {
		if s, err := h.secret.render(map[string]any{"secrets": p.secrets, "settings": p.settings}); err == nil && s != "" {
			return s
		}
		return ""
	}
	if s := p.secrets["webhook_signing_key"]; s != "" {
		return s
	}
	return p.cfg.WebhookSecret
}

// VerifyWebhook implements providers.WebhookVerifier.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, body []byte) error {
	h := p.c.hook
	if h == nil {
		return fmt.Errorf("%w: this mapping has no webhook", providers.ErrBadSignature)
	}
	secret := p.signingSecret()
	switch h.auth {
	case "none":
		return nil
	case "basic":
		return webhookauth.Basic(r, h.user, secret)
	case "header":
		if !webhookauth.Equal(strings.TrimSpace(r.Header.Get(h.token)), secret) {
			return providers.ErrBadSignature
		}
		return nil
	case "standard":
		return webhookauth.Standard(secret, r.Header, body, now())
	}
	if secret == "" {
		return fmt.Errorf("%w: no signing secret is configured", providers.ErrBadSignature)
	}
	ts := ""
	if h.tsHeader != "" {
		ts = strings.TrimSpace(r.Header.Get(h.tsHeader))
		if err := webhookauth.Fresh(ts, now()); err != nil {
			return err
		}
	}
	head, tail, _ := strings.Cut(strings.ReplaceAll(h.payload, "{timestamp}", ts), "{body}")
	want := webhookauth.HMAC(h.hash, []byte(secret), []byte(head), body, []byte(tail))
	for _, sig := range strings.FieldsFunc(r.Header.Get(h.header), func(r rune) bool { return r == ',' || r == ' ' }) {
		got, err := webhookauth.Decode(strings.TrimPrefix(sig, h.prefix), h.encoding)
		if err == nil && len(got) > 0 && hmac.Equal(got, want) {
			return nil
		}
	}
	return providers.ErrBadSignature
}

// ParseWebhook implements providers.Provider: it verifies the request and
// maps every selected event.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if p.c.hook == nil {
		return nil, fmt.Errorf("%w: this mapping has no webhook", providers.ErrBadSignature)
	}
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	doc, err := decodeJSON(body)
	if err != nil {
		return nil, fmt.Errorf("httpgeneric webhook: %w", err)
	}
	h := p.c.hook
	var items []any
	for _, v := range h.events.all(doc) {
		if a, ok := v.([]any); ok {
			items = append(items, a...)
		} else {
			items = append(items, v)
		}
	}
	out := []providers.Event{}
	for _, it := range items {
		if ev, ok := h.normalize(it); ok {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (h *hookSpec) normalize(item any) (providers.Event, bool) {
	raw := strings.TrimSpace(h.typ.text(item))
	key := strings.ToLower(raw)
	kind, ok := h.types[key]
	if !ok {
		kind = statusTypes[key]
	}
	if kind == "" || kind == "ignore" {
		return providers.Event{}, false
	}
	ev := providers.Event{OriginalType: raw, ProviderMessageID: strings.Trim(h.msgID.text(item), "<>")}
	for _, rp := range h.recipient {
		if v := rp.text(item); v != "" {
			ev.Recipient = strings.ToLower(v)
			break
		}
	}
	if id, err := uuid.Parse(h.rcID.text(item)); err == nil {
		ev.MessageID = &id
	}
	codeText := h.code.text(item)
	code, rest := bounce.Split(codeText)
	if code == "" && codeText != "" && len(codeText) <= 40 {
		code, rest = codeText, ""
	}
	ev.OriginalCode = code
	ev.Diagnostic = strings.TrimSpace(firstNonEmpty(h.diag.text(item), rest))
	switch kind {
	case "bounce":
		// A plain "bounced" is permanent unless the code says otherwise.
		kind = "hard_bounce"
		if code != "" && !bounce.Permanent(code) {
			kind = "soft_bounce"
		}
	case "hard_bounce":
		if code != "" && !bounce.Permanent(code) && strings.HasPrefix(code, "4") {
			kind = "soft_bounce"
		}
	}
	ev.Type = providers.EventType(kind)
	switch ev.Type {
	case providers.EventHardBounce, providers.EventSoftBounce, providers.EventDeferred, providers.EventRejected:
		ev.BounceClass = bounce.Classify(code, codeText+" "+ev.Diagnostic)
	}
	ev.URL = h.url.text(item)
	ev.UserAgent = h.ua.text(item)
	ev.OccurredAt = parseTime(item, h.at)
	b, err := json.Marshal(item)
	if err != nil {
		b = []byte("{}")
	}
	ev.Raw = b
	return ev, true
}

// parseTime reads Unix seconds or milliseconds (number or string) or an
// RFC 3339 time; it falls back to now.
func parseTime(item any, p *jsonPath) time.Time {
	v, ok := p.first(item)
	if !ok {
		return now().UTC()
	}
	s := strings.TrimSpace(stringify(v))
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) && f > 0 {
		if f > 1e12 {
			f /= 1000
		}
		sec := math.Floor(f)
		return time.Unix(int64(sec), int64((f-sec)*1e9)).UTC()
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return now().UTC()
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
