// Package telnyxtest is an in-memory Telnyx (API v2): balance, messages,
// phone numbers, messaging profiles, opt-outs, 10DLC and toll-free
// verification, answered from recorded payload shapes, with failure modes
// and helpers that sign webhooks with an Ed25519 key as Telnyx does.
package telnyxtest

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/sms/contracttest"
	"github.com/hostrepute/reputecast/internal/sms/smsfake"
	"github.com/hostrepute/reputecast/internal/sms/telnyx"
)

// APIKey is the sample key.
const APIKey = "KEY0187ACMEINC5A1B2C3D4E5F"

// The signing key is fixed so the public key is a constant tests and the
// demo can paste.
var signing = ed25519.NewKeyFromSeed([]byte("reputecast-telnyx-fake-seed-0001"))

// PublicKey is the base64 public key of the fake's signing key.
var PublicKey = base64.StdEncoding.EncodeToString(signing.Public().(ed25519.PublicKey))

// Fake is the in-memory Telnyx.
type Fake struct {
	mu       sync.Mutex
	rec      smsfake.Recorder
	Mode     smsfake.Mode
	Rejected bool
	OptOuts  []map[string]string
	Profile  string
	seq      int
}

// New returns the sample account.
func New() *Fake {
	return &Fake{OptOuts: []map[string]string{{"from": "+13125550144", "to": "+18885550190", "keyword": "stop", "created_at": "2026-10-12T15:00:00Z"}}}
}

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

func errs(w http.ResponseWriter, status int, code, title string) {
	writeJSON(w, status, map[string]any{"errors": []any{map[string]any{"code": code, "title": title, "detail": title}}})
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	body := f.rec.Record(r)
	switch f.Mode.Of() {
	case contracttest.ModeOutage:
		errs(w, 503, "10006", "Service unavailable")
		return
	case contracttest.ModeAuth:
		errs(w, 401, "10009", "Authentication failed")
		return
	}
	f.mu.Lock()
	rejected := f.Rejected
	f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer "+APIKey || rejected {
		errs(w, 401, "10009", "Authentication failed")
		return
	}
	var in map[string]any
	_ = json.Unmarshal(body, &in)
	p := r.URL.Path
	switch {
	case r.Method == http.MethodGet && p == "/v2/balance":
		writeJSON(w, 200, map[string]any{"data": map[string]any{"balance": "300.00", "currency": "USD", "record_type": "balance"}})
	case r.Method == http.MethodPost && p == "/v2/messages":
		if fmt.Sprint(in["to"]) == contracttest.InvalidNumber {
			errs(w, 400, "40310", "Invalid 'to' address")
			return
		}
		f.mu.Lock()
		f.seq++
		id := fmt.Sprintf("40385f64-5717-4562-b3fc-%012d", f.seq)
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"data": map[string]any{"id": id, "parts": 1, "to": []any{map[string]any{"phone_number": in["to"], "status": "queued"}},
			"cost": map[string]any{"amount": "0.0040", "currency": "USD"}}})
	case r.Method == http.MethodGet && p == "/v2/phone_numbers":
		writeJSON(w, 200, map[string]any{"data": []any{
			map[string]any{"id": "1293384261075731499", "phone_number": "+18885550190", "phone_number_type": "toll_free"},
			map[string]any{"id": "1293384261075731500", "phone_number": "+13125550191", "phone_number_type": "local"},
		}})
	case r.Method == http.MethodPost && p == "/v2/messaging_profiles":
		f.mu.Lock()
		f.Profile = "40017d80-1e1e-4d6b-8a58-acme00000001"
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"data": map[string]any{"id": "40017d80-1e1e-4d6b-8a58-acme00000001", "webhook_url": in["webhook_url"]}})
	case (r.Method == http.MethodPatch) && strings.HasPrefix(p, "/v2/"):
		writeJSON(w, 200, map[string]any{"data": map[string]any{"id": "ok"}})
	case r.Method == http.MethodGet && p == "/v2/messaging_optouts":
		f.mu.Lock()
		list := append([]map[string]string(nil), f.OptOuts...)
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"data": list, "meta": map[string]any{"total_results": len(list)}})
	case r.Method == http.MethodGet && p == "/v2/messaging_tollfree/verification/requests":
		writeJSON(w, 200, map[string]any{"records": []any{map[string]any{"verificationStatus": "Verified", "useCase": "Event reminders", "messageVolume": "10,000",
			"updatedAt": "2026-10-06T14:02:00Z", "phoneNumbers": []any{map[string]any{"phoneNumber": "+18885550190"}}}}})
	case r.Method == http.MethodGet && p == "/10dlc/phone_number_campaigns":
		writeJSON(w, 200, map[string]any{"records": []any{map[string]any{"phoneNumber": "+13125550191", "campaignId": "CKSV5RT"}}})
	case r.Method == http.MethodGet && p == "/10dlc/campaign/CKSV5RT":
		writeJSON(w, 200, map[string]any{"campaignId": "CKSV5RT", "brandId": "BXR0T2P", "description": "Acme Summit attendee updates", "campaignStatus": "TCR_PENDING", "usecase": "MIXED"})
	case r.Method == http.MethodGet && p == "/10dlc/brand/BXR0T2P":
		writeJSON(w, 200, map[string]any{"brandId": "BXR0T2P", "displayName": "Acme Inc", "identityStatus": "VERIFIED"})
	default:
		errs(w, 404, "10005", "Resource not found")
	}
}

// Signed builds a webhook to hookURL signed at at.
func Signed(hookURL string, payload map[string]any, at time.Time) (*http.Request, []byte) {
	body, _ := json.Marshal(payload)
	ts := strconv.FormatInt(at.Unix(), 10)
	r := httptest.NewRequest(http.MethodPost, hookURL, strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(telnyx.TimestampHeader, ts)
	r.Header.Set(telnyx.SignatureHeader, base64.StdEncoding.EncodeToString(ed25519.Sign(signing, []byte(ts+"|"+string(body)))))
	return r, body
}

// Receipt is a message.finalized webhook.
func Receipt(hookURL, id, to, from, status, code string, at time.Time) (*http.Request, []byte) {
	pl := map[string]any{"id": id, "from": map[string]any{"phone_number": from}, "to": []any{map[string]any{"phone_number": to, "status": status}}, "parts": 1,
		"cost": map[string]any{"amount": "0.0040", "currency": "USD"}, "errors": []any{}}
	if code != "" {
		pl["errors"] = []any{map[string]any{"code": code, "title": "Delivery failed"}}
	}
	return Signed(hookURL, map[string]any{"data": map[string]any{"event_type": "message.finalized", "id": "evt-" + id, "occurred_at": at.UTC().Format(time.RFC3339),
		"payload": pl}}, at)
}

// Inbound is a message.received webhook.
func Inbound(hookURL, id, from, to, text string, at time.Time) (*http.Request, []byte) {
	return Signed(hookURL, map[string]any{"data": map[string]any{"event_type": "message.received", "id": "evt-" + id, "occurred_at": at.UTC().Format(time.RFC3339),
		"payload": map[string]any{"id": id, "text": text, "from": map[string]any{"phone_number": from}, "to": []any{map[string]any{"phone_number": to, "status": "webhook_delivered"}}}}}, at)
}
