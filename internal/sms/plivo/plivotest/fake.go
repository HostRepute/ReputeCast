// Package plivotest is an in-memory Plivo: the account, Message, Number,
// Application, 10DLC, toll-free verification and Pricing resources the
// adapter calls, answered from recorded payload shapes, with failure modes
// and helpers that sign callbacks and inbound texts with Plivo's V3
// signature.
package plivotest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"

	"github.com/hostrepute/reputecast/internal/sms/contracttest"
	"github.com/hostrepute/reputecast/internal/sms/plivo"
	"github.com/hostrepute/reputecast/internal/sms/smsfake"
)

// The sample credentials.
const (
	AuthID    = "MAODUZYTQ0Y2FMYJBLOW"
	AuthToken = "recorded-plivo-auth-token"
)

// Fake is the in-memory Plivo.
type Fake struct {
	mu       sync.Mutex
	rec      smsfake.Recorder
	Mode     smsfake.Mode
	Rejected bool
	AppURL   string
	seq      int
}

// New returns the sample account.
func New() *Fake { return &Fake{} }

// Handler serves the fake.
func (f *Fake) Handler() http.Handler { return http.HandlerFunc(f.serve) }

// Client serves the fake in process.
func (f *Fake) Client() *http.Client { return smsfake.Client(f.Handler()) }

// Server starts the fake on a loopback address.
func (f *Fake) Server() *httptest.Server { return httptest.NewServer(f.Handler()) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	body := f.rec.Record(r)
	switch f.Mode.Of() {
	case contracttest.ModeOutage:
		writeJSON(w, 503, map[string]any{"error": "service unavailable"})
		return
	case contracttest.ModeAuth:
		writeJSON(w, 401, map[string]any{"error": "Authentication failed"})
		return
	}
	user, pass, ok := r.BasicAuth()
	f.mu.Lock()
	rejected := f.Rejected
	f.mu.Unlock()
	if !ok || user != AuthID || pass != AuthToken || rejected {
		writeJSON(w, 401, map[string]any{"error": "Authentication failed"})
		return
	}
	base := "/v1/Account/" + AuthID
	p := strings.TrimPrefix(r.URL.Path, base)
	var in map[string]any
	_ = json.Unmarshal(body, &in)
	switch {
	case r.Method == http.MethodGet && p == "/":
		writeJSON(w, 200, map[string]any{"name": "Acme Inc", "auth_id": AuthID, "account_type": "standard", "cash_credits": "50.00000"})
	case r.Method == http.MethodPost && p == "/Message/":
		if "+"+fmt.Sprint(in["dst"]) == contracttest.InvalidNumber {
			writeJSON(w, 400, map[string]any{"api_id": "e9b2", "error": "dst parameter not valid"})
			return
		}
		f.mu.Lock()
		f.seq++
		id := fmt.Sprintf("db3ce55a-7f1d-11e1-8ea7-%012d", f.seq)
		f.mu.Unlock()
		writeJSON(w, 202, map[string]any{"api_id": "e9b3", "message": "message(s) queued", "message_uuid": []string{id}})
	case r.Method == http.MethodGet && p == "/Number/":
		writeJSON(w, 200, map[string]any{"objects": []any{
			map[string]any{"number": "18885550170", "sms_enabled": true, "type": "tollfree", "country": "United States"},
			map[string]any{"number": "13125550133", "sms_enabled": true, "type": "local", "country": "United States"},
		}})
	case r.Method == http.MethodPost && p == "/Application/":
		f.mu.Lock()
		f.AppURL = fmt.Sprint(in["message_url"])
		f.mu.Unlock()
		writeJSON(w, 201, map[string]any{"app_id": "20372631212782780", "message": "created"})
	case r.Method == http.MethodPost && strings.HasPrefix(p, "/Number/"):
		writeJSON(w, 202, map[string]any{"message": "changed"})
	case r.Method == http.MethodGet && p == "/TollfreeVerification/":
		writeJSON(w, 200, map[string]any{"objects": []any{map[string]any{"number": "18885550170", "status": "APPROVED", "usecase": "Event reminders",
			"volume": "10,000", "last_modified": "2026-10-06T14:02:00Z"}}})
	case r.Method == http.MethodGet && p == "/10dlc/Brand/":
		writeJSON(w, 200, map[string]any{"brands": []any{map[string]any{"brand_id": "BRPXS6E", "brand_name": "Acme Inc", "registration_status": "COMPLETED"}}})
	case r.Method == http.MethodGet && p == "/10dlc/Campaign/":
		writeJSON(w, 200, map[string]any{"campaigns": []any{map[string]any{"campaign_id": "CMPT4EP", "brand_id": "BRPXS6E", "registration_status": "PENDING",
			"description": "Acme Summit attendee updates", "usecase": "MIXED"}}})
	case r.Method == http.MethodGet && p == "/10dlc/Campaign/CMPT4EP/Number/":
		writeJSON(w, 200, map[string]any{"phone_numbers": []any{map[string]any{"number": "13125550133"}}})
	case r.Method == http.MethodGet && p == "/Pricing/":
		rates := map[string]string{"US": "0.00550", "GB": "0.04240"}
		writeJSON(w, 200, map[string]any{"country_iso": r.URL.Query().Get("country_iso"), "message": map[string]any{"outbound": map[string]any{"rate": rates[r.URL.Query().Get("country_iso")]}}})
	default:
		writeJSON(w, 404, map[string]any{"error": "not found"})
	}
}

func signed(token, hookURL string, form url.Values) (*http.Request, []byte) {
	body := []byte(form.Encode())
	r := httptest.NewRequest(http.MethodPost, hookURL, strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	nonce := "05429567804466091622"
	r.Header.Set(plivo.NonceHeader, nonce)
	r.Header.Set(plivo.SignatureHeader, plivo.Signature(token, http.MethodPost, hookURL, nonce, form))
	return r, body
}

// Receipt is a status callback.
func Receipt(token, hookURL, uuid, to, from, status, code string) (*http.Request, []byte) {
	form := url.Values{"MessageUUID": {uuid}, "Status": {status}, "To": {strings.TrimPrefix(to, "+")}, "From": {strings.TrimPrefix(from, "+")},
		"Units": {"1"}, "TotalRate": {"0.00550"}, "TotalAmount": {"0.00550"}, "ErrorCode": {code}, "MessageTime": {"2026-10-14 15:00:04+00:00"}}
	return signed(token, hookURL, form)
}

// Inbound is a text someone sent to a number in the account.
func Inbound(token, hookURL, uuid, from, to, text string) (*http.Request, []byte) {
	form := url.Values{"MessageUUID": {uuid}, "From": {strings.TrimPrefix(from, "+")}, "To": {strings.TrimPrefix(to, "+")}, "Text": {text}, "Type": {"sms"},
		"TotalAmount": {"0.00000"}, "Units": {"1"}}
	return signed(token, hookURL, form)
}
