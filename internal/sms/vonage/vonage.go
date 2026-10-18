// Package vonage is the Vonage SMS adapter (the SMS API): texts with an API
// key and secret, delivery receipts and inbound texts on the account's
// callback addresses, set through the account API and signed with the
// account's signature secret (HMAC-SHA256), the account's numbers and
// outbound prices.
//
// Vonage reports no registration status through its API, so senders'
// registrations are set by hand. Its opt-outs reach ReputeCast as replies
// and as refused texts.
package vonage

import (
	"context"
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // Vonage's "MD5 hash" signature method
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
)

// Adapter is the registry key.
const Adapter = "vonage"

const apiHost = "https://rest.nexmo.com"

func init() { sms.Register(Descriptor()) }

// Descriptor is the adapter's manifest.
func Descriptor() sms.Descriptor {
	return sms.Descriptor{
		Manifest: sms.Manifest{
			Adapter: Adapter, Name: "Vonage",
			Summary: "Pay as you go · strong sender ID coverage in Europe",
			Badges: []sms.Badge{
				{Capability: sms.CapSenderIDs, Support: sms.Yes}, {Capability: sms.CapTollFree, Support: sms.Yes}, {Capability: sms.Cap10DLC, Support: sms.Yes},
				{Capability: sms.CapReceipts, Support: sms.Yes}, {Capability: sms.CapInbound, Support: sms.Yes},
				{Capability: sms.CapOptOutSync, Support: sms.Partial, Note: "from replies and refused texts"},
				{Capability: sms.CapRegistration, Support: sms.Partial, Note: "set by hand"},
				{Capability: sms.CapPricing, Support: sms.Yes},
			},
			Fields: []sms.Field{
				{Key: "api_key", Label: "API key", Required: true},
				{Key: "api_secret", Label: "API secret", Secret: true, Required: true},
				{Key: "signature_secret", Label: "Signature secret", Secret: true, Required: true,
					Help: "Settings › API settings › Signed webhooks, with HMAC SHA-256. ReputeCast checks every receipt and reply against it."},
			},
			Webhooks: "registered",
			Console: func(kind, _ string) string {
				switch kind {
				case "registration":
					return "https://dashboard.nexmo.com/sms/brands"
				case "numbers":
					return "https://dashboard.nexmo.com/your-numbers"
				}
				return "https://dashboard.nexmo.com/settings"
			},
			Version: "1.0.0", Maintainer: "ReputeCast",
		},
		New: New,
	}
}

// Provider is a Vonage account.
type Provider struct {
	cfg               sms.Config
	key, secret, sigS string
	http              *http.Client
}

// New opens a Vonage account.
func New(cfg sms.Config) (sms.Provider, error) {
	p := &Provider{cfg: cfg, key: strings.TrimSpace(cfg.Value("api_key")), secret: cfg.Value("api_secret"), sigS: cfg.Value("signature_secret"), http: cfg.Client()}
	if p.key == "" {
		return nil, errors.New("vonage: the API key is required")
	}
	return p, nil
}

var (
	_ sms.SenderLister     = (*Provider)(nil)
	_ sms.WebhookRegistrar = (*Provider)(nil)
	_ sms.PricingReader    = (*Provider)(nil)
)

// Adapter implements sms.Provider.
func (p *Provider) Adapter() string { return Adapter }

func (p *Provider) base() string {
	if p.cfg.BaseURL != "" {
		return strings.TrimRight(p.cfg.BaseURL, "/")
	}
	return apiHost
}

func (p *Provider) auth() url.Values { return url.Values{"api_key": {p.key}, "api_secret": {p.secret}} }

// Check implements sms.Provider: the balance call proves the key.
func (p *Provider) Check(ctx context.Context) (sms.Account, error) {
	var b struct {
		Value float64 `json:"value"`
	}
	if _, err := sms.Do(ctx, p.http, sms.Call{Op: "check", Method: http.MethodGet, URL: p.base() + "/account/get-balance?" + p.auth().Encode()}, &b); err != nil {
		return sms.Account{}, err
	}
	return sms.Account{ID: p.key, Name: "Vonage account " + sms.Hint(p.key, 0, 4), Hint: sms.Hint(p.key, 0, 4), Live: true}, nil
}

// statusReason maps the SMS API's submission status codes.
func statusReason(code string) (sms.Reason, bool) {
	switch code {
	case "1":
		return sms.ReasonOther, true // throttled
	case "3":
		return sms.ReasonInvalid, false
	case "5":
		return sms.ReasonOther, true
	case "7", "33":
		return sms.ReasonOptedOut, false // barred, de-activated
	case "15", "29", "9", "6", "12":
		return sms.ReasonRejected, false
	}
	return sms.ReasonOther, false
}

// errReason maps delivery receipts' err-code.
func errReason(code string) (sms.Reason, bool) {
	switch code {
	case "1", "2", "7", "8", "12":
		return sms.ReasonUnreachable, true // unknown, absent subscriber, busy, network
	case "3":
		return sms.ReasonNotInService, false // absent subscriber, permanent
	case "4":
		return sms.ReasonOptedOut, false // barred by the user
	case "6", "14", "50":
		return sms.ReasonFiltered, false // anti-spam, blocked by carrier, entity filter
	case "9":
		return sms.ReasonInvalid, false
	case "11", "10":
		return sms.ReasonRejected, false
	}
	return sms.ReasonOther, false
}

// Send implements sms.Provider.
func (p *Provider) Send(ctx context.Context, m sms.Message) (sms.SendResult, error) {
	form := p.auth()
	form.Set("to", sms.E164Digits(m.To))
	form.Set("from", sms.E164Digits(m.From))
	form.Set("text", m.Body)
	if m.Encoding == sms.UCS2 {
		form.Set("type", "unicode")
	}
	form.Set("status-report-req", "1")
	if m.StatusURL != "" {
		form.Set("callback", m.StatusURL)
	}
	form.Set("client-ref", m.ID.String())
	var out struct {
		Count    string `json:"message-count"`
		Messages []struct {
			Status    string `json:"status"`
			MessageID string `json:"message-id"`
			Price     string `json:"message-price"`
			ErrorText string `json:"error-text"`
		} `json:"messages"`
	}
	if _, err := sms.Do(ctx, p.http, sms.Call{Op: "send", Method: http.MethodPost, URL: p.base() + "/sms/json", Form: form}, &out); err != nil {
		return sms.SendResult{}, err
	}
	if len(out.Messages) == 0 {
		return sms.SendResult{}, &sms.SendError{Message: "Vonage returned no message", Temporary: true}
	}
	res := sms.SendResult{Segments: len(out.Messages)}
	for _, msg := range out.Messages {
		if msg.Status != "0" {
			if msg.Status == "4" || msg.Status == "14" {
				return sms.SendResult{}, fmt.Errorf("%w: %s", sms.ErrCredentialsRejected, msg.ErrorText)
			}
			reason, temp := statusReason(msg.Status)
			return sms.SendResult{}, &sms.SendError{Code: msg.Status, Message: msg.ErrorText, Reason: reason, Temporary: temp, Throttled: msg.Status == "1"}
		}
		if res.ProviderMessageID == "" {
			res.ProviderMessageID = msg.MessageID
		}
		if v, err := strconv.ParseFloat(msg.Price, 64); err == nil {
			total := v
			if res.Price != nil {
				total += *res.Price
			}
			res.Price = &total
		}
	}
	return res, nil
}

// Signature computes the signed-webhook signature: every parameter except
// sig, sorted by name, written "&name=value" (with & and = in values
// replaced by _), signed with HMAC-SHA256 and hex-encoded. method "md5hash"
// is Vonage's older scheme: the string plus the secret, MD5-hashed.
func Signature(method, secret string, v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		if k != "sig" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	clean := strings.NewReplacer("&", "_", "=", "_")
	for _, k := range keys {
		b.WriteString("&" + k + "=" + clean.Replace(v.Get(k)))
	}
	if method == "md5hash" {
		sum := md5.Sum([]byte(b.String() + secret)) //nolint:gosec // the provider's scheme
		return hex.EncodeToString(sum[:])
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(b.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

// params reads a webhook: Vonage sends query parameters (GET), a form or
// JSON depending on the account's settings.
func params(r *http.Request, body []byte) url.Values {
	v := url.Values{}
	if r != nil {
		for k, vs := range r.URL.Query() {
			v[k] = vs
		}
	}
	ct := ""
	if r != nil {
		ct = r.Header.Get("Content-Type")
	}
	return merge(v, ct, body)
}

func merge(v url.Values, ct string, body []byte) url.Values {
	if len(body) == 0 {
		return v
	}
	if strings.HasPrefix(ct, "application/json") {
		var m map[string]any
		if json.Unmarshal(body, &m) == nil {
			for k, x := range m {
				v.Set(k, fmt.Sprint(x))
			}
		}
		return v
	}
	if f, err := url.ParseQuery(string(body)); err == nil {
		for k, vs := range f {
			v[k] = vs
		}
	}
	return v
}

// VerifyWebhook implements sms.Provider: the signature and the timestamp
// (within 5 minutes, so an old request can't be replayed).
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, body []byte) error {
	v := params(r, body)
	sig := v.Get("sig")
	if sig == "" || p.sigS == "" {
		return sms.ErrBadSignature
	}
	if ts, err := strconv.ParseInt(v.Get("timestamp"), 10, 64); err != nil || abs(p.cfg.Clock().Unix()-ts) > 300 {
		return sms.ErrBadSignature
	}
	want := Signature("hmac-sha256", p.sigS, v)
	if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(want)) {
		if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(Signature("md5hash", p.sigS, v))) {
			return sms.ErrBadSignature
		}
	}
	return nil
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func plus(digits string) string {
	if digits == "" || strings.HasPrefix(digits, "+") {
		return digits
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return digits // a sender ID
		}
	}
	return "+" + digits
}

// ParseWebhook implements sms.Provider: a delivery receipt or an inbound
// text. The body holds the parameters (JSON or a form); for GET callbacks
// the ingest stores the query string as the body.
func (p *Provider) ParseWebhook(_ context.Context, contentType string, body []byte) ([]sms.Event, error) {
	v := merge(url.Values{}, contentType, body)
	at := p.cfg.Clock()
	if t, err := time.Parse("2006-01-02 15:04:05", v.Get("message-timestamp")); err == nil {
		at = t
	}
	if status := v.Get("status"); status != "" && v.Get("messageId") != "" && v.Get("text") == "" {
		e := sms.Event{ProviderMessageID: v.Get("messageId"), To: plus(v.Get("msisdn")), From: plus(v.Get("to")), OccurredAt: at,
			Summary: "status=" + status + func() string {
				if c := v.Get("err-code"); c != "" && c != "0" {
					return " · err-code=" + c
				}
				return ""
			}()}
		switch status {
		case "accepted", "buffered":
			e.Kind = sms.EventSent
		case "delivered":
			e.Kind = sms.EventDelivered
		case "expired", "failed", "rejected", "unknown":
			e.Kind = sms.EventUndelivered
			if status == "rejected" {
				e.Kind = sms.EventFailed
			}
		default:
			return nil, nil
		}
		if c := v.Get("err-code"); c != "" && c != "0" {
			e.Code = c
			e.Reason, _ = errReason(c)
		}
		if pr, err := strconv.ParseFloat(v.Get("price"), 64); err == nil {
			e.Price = &pr
		}
		return []sms.Event{e}, nil
	}
	if v.Get("msisdn") != "" && (v.Has("text") || v.Get("type") != "") {
		return []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: v.Get("messageId"), From: plus(v.Get("msisdn")), To: plus(v.Get("to")), Body: v.Get("text"),
			OccurredAt: at, Summary: "Inbound · text=" + sms.Quote(v.Get("text"))}}, nil
	}
	return nil, nil
}

// Senders implements sms.SenderLister: the account's numbers. Sender IDs
// aren't numbers in the account; the wizard adds them by name.
func (p *Provider) Senders(ctx context.Context) ([]sms.Sender, error) {
	var out struct {
		Numbers []struct {
			Country  string   `json:"country"`
			MSISDN   string   `json:"msisdn"`
			Type     string   `json:"type"`
			Features []string `json:"features"`
		} `json:"numbers"`
	}
	if _, err := sms.Do(ctx, p.http, sms.Call{Op: "numbers", Method: http.MethodGet, URL: p.base() + "/account/numbers?" + p.auth().Encode()}, &out); err != nil {
		return nil, err
	}
	ss := []sms.Sender{}
	for _, n := range out.Numbers {
		e := plus(n.MSISDN)
		pn, err := phone.Parse(e, "")
		if err != nil {
			continue
		}
		kind := "mobile"
		switch {
		case n.Type == "toll-free":
			kind = "toll_free"
		case pn.Country == "US":
			kind = "10dlc"
		}
		ss = append(ss, sms.Sender{Kind: kind, Value: e, Countries: []string{strings.ToUpper(n.Country)}, Throughput: 30})
	}
	return ss, nil
}

// RegisterWebhooks implements sms.WebhookRegistrar: the account's delivery
// receipt and inbound addresses. Vonage signs them with the signature
// secret the person entered, so nothing new comes back.
func (p *Provider) RegisterWebhooks(ctx context.Context, hooks sms.Webhooks, _ []sms.Sender) (sms.WebhookSetup, error) {
	form := p.auth()
	form.Set("drCallBackUrl", hooks.StatusURL)
	form.Set("moCallBackUrl", hooks.InboundURL)
	_, err := sms.Do(ctx, p.http, sms.Call{Op: "settings", Method: http.MethodPost, URL: p.base() + "/account/settings", Form: form}, nil)
	return sms.WebhookSetup{}, err
}

// Prices implements sms.PricingReader.
func (p *Provider) Prices(ctx context.Context, countries []string) (map[string]float64, error) {
	out := map[string]float64{}
	for _, c := range countries {
		q := p.auth()
		q.Set("country", strings.ToUpper(c))
		var pr struct {
			DefaultPrice string `json:"defaultPrice"`
		}
		if _, err := sms.Do(ctx, p.http, sms.Call{Op: "pricing", Method: http.MethodGet, URL: p.base() + "/account/get-pricing/outbound/sms?" + q.Encode()}, &pr); err != nil {
			return nil, err
		}
		if v, err := strconv.ParseFloat(pr.DefaultPrice, 64); err == nil && v > 0 {
			out[strings.ToUpper(c)] = v
		}
	}
	return out, nil
}
