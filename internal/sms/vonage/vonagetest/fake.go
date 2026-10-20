// Package vonagetest is an in-memory Vonage SMS API: sending, the balance,
// numbers, account settings and pricing, answered from recorded payload
// shapes, with failure modes and helpers that sign delivery receipts and
// inbound texts as Vonage's signed webhooks do.
package vonagetest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/sms/smsfake"
	"github.com/hostrepute/reputecast/internal/sms/vonage"
)

// The sample account (brief §4: API key …7c21).
const (
	APIKey          = "a1b27c21"
	APISecret       = "recorded-vonage-secret"
	SignatureSecret = "recorded-vonage-signature-secret-9d2e"
)

// Fake is the in-memory Vonage.
type Fake struct {
	mu       sync.Mutex
	rec      smsfake.Recorder
	Mode     smsfake.Mode
	Rejected bool
	Prices   map[string]string
	DR, MO   string
	sent     []url.Values
	seq      int
}

// New returns the sample account.
func New() *Fake {
	return &Fake{Prices: map[string]string{"GB": "0.04500000", "DE": "0.08800000", "NL": "0.09600000", "IE": "0.07600000", "US": "0.00840000"}}
}

// Handler serves the fake.
func (f *Fake) Handler() http.Handler { return http.HandlerFunc(f.serve) }

// Client serves the fake in process.
func (f *Fake) Client() *http.Client { return smsfake.Client(f.Handler()) }

// Server starts the fake on a loopback address.
func (f *Fake) Server() *httptest.Server { return httptest.NewServer(f.Handler()) }

// Sent are the texts it accepted.
func (f *Fake) Sent() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.sent...)
}

// Callbacks are the account's receipt and inbound addresses.
func (f *Fake) Callbacks() (dr, mo string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.DR, f.MO
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	body := f.rec.Record(r)
	v := r.URL.Query()
	if form, err := url.ParseQuery(string(body)); err == nil {
		for k, vs := range form {
			v[k] = vs
		}
	}
	switch f.Mode.Of() {
	case smsfake.ModeOutage:
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	case smsfake.ModeAuth:
		if r.URL.Path == "/sms/json" {
			writeJSON(w, 200, map[string]any{"message-count": "1", "messages": []any{map[string]any{"status": "4", "error-text": "Bad Credentials"}}})
			return
		}
		writeJSON(w, 401, map[string]any{"error-code": "401", "error-code-label": "authentication failed"})
		return
	}
	f.mu.Lock()
	rejected := f.Rejected
	f.mu.Unlock()
	if v.Get("api_key") != APIKey || v.Get("api_secret") != APISecret || rejected {
		if r.URL.Path == "/sms/json" {
			writeJSON(w, 200, map[string]any{"message-count": "1", "messages": []any{map[string]any{"status": "4", "error-text": "Bad Credentials"}}})
			return
		}
		writeJSON(w, 401, map[string]any{"error-code": "401", "error-code-label": "authentication failed"})
		return
	}
	switch r.URL.Path {
	case "/account/get-balance":
		writeJSON(w, 200, map[string]any{"value": 75.20, "autoReload": false})
	case "/sms/json":
		to := "+" + v.Get("to")
		if to == smsfake.InvalidNumber {
			writeJSON(w, 200, map[string]any{"message-count": "1", "messages": []any{map[string]any{"status": "3", "error-text": "Invalid to address"}}})
			return
		}
		f.mu.Lock()
		f.seq++
		id := fmt.Sprintf("1800000%09d", f.seq)
		f.sent = append(f.sent, v)
		price := f.Prices["GB"]
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"message-count": "1", "messages": []any{map[string]any{"to": v.Get("to"), "message-id": id, "status": "0",
			"remaining-balance": "75.15", "message-price": price, "network": "23410"}}})
	case "/account/numbers":
		// A UK long number for replies; the sample sends from the sender ID ACME.
		writeJSON(w, 200, map[string]any{"count": 1, "numbers": []any{map[string]any{"country": "GB", "msisdn": "447700900000", "type": "mobile-lvn",
			"features": []string{"SMS", "VOICE"}}}})
	case "/account/settings":
		f.mu.Lock()
		f.DR, f.MO = v.Get("drCallBackUrl"), v.Get("moCallBackUrl")
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"mo-callback-url": v.Get("moCallBackUrl"), "dr-callback-url": v.Get("drCallBackUrl")})
	case "/account/get-pricing/outbound/sms":
		c := v.Get("country")
		f.mu.Lock()
		price, ok := f.Prices[c]
		f.mu.Unlock()
		if !ok {
			writeJSON(w, 200, map[string]any{"countryCode": c, "defaultPrice": "0"})
			return
		}
		writeJSON(w, 200, map[string]any{"countryCode": c, "defaultPrice": price, "currency": "USD"})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// signed builds a JSON webhook to hookURL with a timestamp and signature.
func signed(secret, hookURL string, v url.Values, at time.Time) (*http.Request, []byte) {
	v.Set("timestamp", strconv.FormatInt(at.Unix(), 10))
	v.Set("nonce", "4a1d6e2b-0c7f-4b9e-8f3a-2d5c6b7a8e9f")
	v.Set("sig", vonage.Signature("hmac-sha256", secret, v))
	m := map[string]string{}
	for k := range v {
		m[k] = v.Get(k)
	}
	body, _ := json.Marshal(m)
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, hookURL, strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	return r, body
}

// Receipt is a delivery receipt for a text (status delivered, failed,
// expired…; errCode "0" when none).
func Receipt(secret, hookURL, messageID, to, from, status, errCode string, at time.Time) (*http.Request, []byte) {
	v := url.Values{"msisdn": {strings.TrimPrefix(to, "+")}, "to": {strings.TrimPrefix(from, "+")}, "network-code": {"23410"}, "messageId": {messageID},
		"price": {"0.04500000"}, "status": {status}, "scts": {at.UTC().Format("0601021504")}, "err-code": {errCode}, "api-key": {APIKey},
		"message-timestamp": {at.UTC().Format("2006-01-02 15:04:05")}}
	return signed(secret, hookURL, v, at)
}

// Inbound is a text someone sent to a Vonage number.
func Inbound(secret, hookURL, messageID, from, to, text string, at time.Time) (*http.Request, []byte) {
	v := url.Values{"msisdn": {strings.TrimPrefix(from, "+")}, "to": {strings.TrimPrefix(to, "+")}, "messageId": {messageID}, "text": {text}, "type": {"text"},
		"keyword": {strings.ToUpper(strings.Fields(text + " x")[0])}, "api-key": {APIKey}, "message-timestamp": {at.UTC().Format("2006-01-02 15:04:05")}}
	return signed(secret, hookURL, v, at)
}
