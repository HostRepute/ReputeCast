// Package twiliotest is an in-memory Twilio: the parts of Programmable
// Messaging, Pricing, Messaging compliance and Webhooks the adapter calls,
// answered from recorded payload shapes, with failure modes and helpers that
// sign status callbacks and inbound texts as Twilio would. Tests serve it
// with httptest; the demo runs it in process, so nothing leaves the machine.
package twiliotest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/sms/smsfake"
	"github.com/hostrepute/reputecast/internal/sms/twilio"
)

// The sample account (brief §4: Acme Inc, live, Account SID AC…3f9).
const (
	AccountSID = "AC8e2c5d1b7f4a9e06c3d2b1a0f9e8d3f9"
	KeySID     = "SK41d0b9e7c6a5f4e3d2c1b0a9f8e7d6c2" //nolint:gosec // the recorded sample account, not a credential
	KeySecret  = "recorded-twilio-key-secret-7f3a"    //nolint:gosec // the recorded sample account, not a credential
)

// Number is a number in the account.
type Number struct {
	SID, Phone string
	SMS        bool
	SmsURL     string
}

// Fake is the in-memory Twilio.
type Fake struct {
	mu   sync.Mutex
	rec  smsfake.Recorder
	Mode smsfake.Mode
	// Rejected answers 401 to every call (the API key was revoked).
	Rejected bool
	Numbers  []Number
	// TollFree is the toll-free verification of +1 888 555 0142.
	TollFreeStatus, TollFreeReason string
	// Campaign is the A2P campaign of +1 312 555 0187.
	BrandStatus, CampaignStatus, CampaignReason string
	Prices                                      map[string]string
	SharedKeySecret                             string
	sent                                        []url.Values
	seq                                         int
}

// New returns the sample account.
func New() *Fake {
	return &Fake{
		Numbers: []Number{
			{SID: "PN0142a1b2c3d4e5f60718293a4b5c6d7e", Phone: "+18885550142", SMS: true},
			{SID: "PN0187f6e5d4c3b2a1908172635a4b3c2d", Phone: "+13125550187", SMS: true},
			{SID: "PN0109c4d3e2f1a0b9c8d7e6f5a4b3c2d1", Phone: "+14155550109", SMS: true},
		},
		TollFreeStatus: "TWILIO_APPROVED", BrandStatus: "APPROVED", CampaignStatus: "PENDING",
		Prices: map[string]string{"US": "0.0110", "CA": "0.0110"},
	}
}

// Handler serves the fake.
func (f *Fake) Handler() http.Handler { return http.HandlerFunc(f.serve) }

// Client serves the fake in process.
func (f *Fake) Client() *http.Client { return smsfake.Client(f.Handler()) }

// Server starts the fake on a loopback address.
func (f *Fake) Server() *httptest.Server { return httptest.NewServer(f.Handler()) }

// Requests are what the fake received.
func (f *Fake) Requests() []smsfake.Recorded { return f.rec.Requests() }

// Sent are the texts it accepted, oldest first.
func (f *Fake) Sent() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.sent...)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func twErr(w http.ResponseWriter, status, code int, msg string) {
	writeJSON(w, status, map[string]any{"code": code, "message": msg, "more_info": fmt.Sprintf("https://www.twilio.com/docs/errors/%d", code), "status": status})
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	body := f.rec.Record(r)
	switch f.Mode.Of() {
	case smsfake.ModeOutage:
		twErr(w, http.StatusServiceUnavailable, 20500, "Service Unavailable")
		return
	case smsfake.ModeAuth:
		twErr(w, http.StatusUnauthorized, 20003, "Authenticate")
		return
	}
	user, pass, ok := r.BasicAuth()
	f.mu.Lock()
	rejected := f.Rejected
	f.mu.Unlock()
	if !ok || user != KeySID || pass != KeySecret || rejected {
		twErr(w, http.StatusUnauthorized, 20003, "Authenticate")
		return
	}
	p := r.URL.Path
	acct := "/2010-04-01/Accounts/" + AccountSID
	switch {
	case r.Method == http.MethodGet && p == acct+".json":
		writeJSON(w, 200, map[string]any{"sid": AccountSID, "friendly_name": "Acme Inc", "status": "active", "type": "Full"})
	case r.Method == http.MethodPost && p == acct+"/Messages.json":
		f.send(w, body)
	case r.Method == http.MethodGet && p == acct+"/IncomingPhoneNumbers.json":
		f.mu.Lock()
		out := []map[string]any{}
		for _, n := range f.Numbers {
			out = append(out, map[string]any{"sid": n.SID, "phone_number": n.Phone, "capabilities": map[string]any{"sms": n.SMS, "voice": true}, "sms_url": n.SmsURL})
		}
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"incoming_phone_numbers": out})
	case r.Method == http.MethodPost && strings.HasPrefix(p, acct+"/IncomingPhoneNumbers/"):
		sid := strings.TrimSuffix(strings.TrimPrefix(p, acct+"/IncomingPhoneNumbers/"), ".json")
		form, _ := url.ParseQuery(string(body))
		f.mu.Lock()
		found := false
		for i := range f.Numbers {
			if f.Numbers[i].SID == sid {
				f.Numbers[i].SmsURL, found = form.Get("SmsUrl"), true
			}
		}
		f.mu.Unlock()
		if !found {
			twErr(w, 404, 20404, "The requested resource was not found")
			return
		}
		writeJSON(w, 200, map[string]any{"sid": sid})
	case r.Method == http.MethodGet && p == "/messaging/v1/Tollfree/Verifications":
		f.mu.Lock()
		v := map[string]any{"sid": "HH0142", "tollfree_phone_number_sid": f.Numbers[0].SID, "status": f.TollFreeStatus, "rejection_reason": f.TollFreeReason,
			"use_case_summary": "Event reminders and updates for registrants", "message_volume": "10,000", "opt_in_type": "WEB_FORM",
			"date_created": "2026-10-01T15:12:00Z", "date_updated": "2026-10-06T14:02:00Z"}
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"verifications": []any{v}})
	case r.Method == http.MethodGet && p == "/messaging/v1/a2p/BrandRegistrations":
		f.mu.Lock()
		st := f.BrandStatus
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"data": []any{map[string]any{"sid": "BN0187", "status": st, "brand_name": "Acme Inc", "brand_type": "STANDARD", "date_updated": "2026-10-02T16:40:00Z"}}})
	case r.Method == http.MethodGet && p == "/messaging/v1/Services":
		writeJSON(w, 200, map[string]any{"services": []any{map[string]any{"sid": "MG0187", "friendly_name": "Acme Summit"}}})
	case r.Method == http.MethodGet && p == "/messaging/v1/Services/MG0187/PhoneNumbers":
		writeJSON(w, 200, map[string]any{"phone_numbers": []any{map[string]any{"sid": "PN0187f6e5d4c3b2a1908172635a4b3c2d", "phone_number": "+13125550187"}}})
	case r.Method == http.MethodGet && p == "/messaging/v1/Services/MG0187/Compliance/Usa2p":
		f.mu.Lock()
		st, reason := f.CampaignStatus, f.CampaignReason
		f.mu.Unlock()
		updated := "2026-10-09T13:20:00Z"
		if st == "FAILED" {
			updated = "2026-10-13T15:05:00Z"
		}
		writeJSON(w, 200, map[string]any{"compliance": []any{map[string]any{"sid": "QE0187", "campaign_status": st, "description": "Acme Summit attendee updates",
			"brand_registration_sid": "BN0187", "date_updated": updated, "failure_reason": reason, "us_app_to_person_usecase": "MIXED",
			"message_flow": "People register for Acme Summit 2026 at news.acme.io/summit and tick an unchecked SMS consent box."}}})
	case r.Method == http.MethodGet && strings.HasPrefix(p, "/pricing/v1/Messaging/Countries/"):
		iso := strings.TrimPrefix(p, "/pricing/v1/Messaging/Countries/")
		f.mu.Lock()
		price, ok := f.Prices[iso]
		f.mu.Unlock()
		if !ok {
			twErr(w, 404, 20404, "The requested resource was not found")
			return
		}
		writeJSON(w, 200, map[string]any{"iso_country": iso, "price_unit": "USD", "outbound_sms_prices": []any{map[string]any{"carrier": "Default",
			"prices": []any{map[string]any{"number_type": "local", "base_price": price, "current_price": price}}}}})
	case r.Method == http.MethodPost && p == "/webhooks/v1/Webhooks/SharedKeys":
		secret := randomHex(16)
		f.mu.Lock()
		f.SharedKeySecret = secret
		f.mu.Unlock()
		writeJSON(w, 201, map[string]any{"id": "webhooks_sharedkey_01jacmesummitreputecast00", "friendlyName": "ReputeCast", "secret": secret,
			"createdAt": time.Now().UTC().Format(time.RFC3339)})
	case r.Method == http.MethodPost && p == "/webhooks/v1/Webhooks/Settings":
		writeJSON(w, 201, map[string]any{"id": "webhooks_setting_01jacmesummitreputecast00"})
	case r.Method == http.MethodPost && p == "/webhooks/v1/Webhooks/Rules":
		writeJSON(w, 201, map[string]any{"id": "webhooks_rule_01jacmesummitreputecast000"})
	default:
		twErr(w, 404, 20404, "The requested resource "+p+" was not found")
	}
}

func (f *Fake) send(w http.ResponseWriter, body []byte) {
	form, _ := url.ParseQuery(string(body))
	to := form.Get("To")
	switch to {
	case smsfake.InvalidNumber:
		twErr(w, 400, 21211, "Invalid 'To' Phone Number: "+to)
		return
	case smsfake.LandlineNumber:
		twErr(w, 400, 21614, "'To' number is not a valid mobile number")
		return
	}
	if form.Get("From") == "" || form.Get("Body") == "" {
		twErr(w, 400, 21602, "Message body is required")
		return
	}
	f.mu.Lock()
	f.seq++
	sid := fmt.Sprintf("SM%030x", f.seq)
	f.sent = append(f.sent, form)
	f.mu.Unlock()
	writeJSON(w, 201, map[string]any{"sid": sid, "status": "queued", "to": to, "from": form.Get("From"), "num_segments": "1", "price": nil, "error_code": nil})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// signed builds a webhook Twilio sends to hookURL, signed with the shared
// key (HMAC-SHA256).
func signed(secret, hookURL string, form url.Values) (*http.Request, []byte) {
	body := []byte(form.Encode())
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, hookURL, strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set(twilio.SettingsHeader, "key=webhooks_sharedkey_01jacmesummitreputecast00,algorithm=HMAC_SHA256")
	r.Header.Set(twilio.SignatureHeader, twilio.Sign(secret, hookURL, form))
	return r, body
}

// Receipt is a status callback for a text.
func Receipt(secret, hookURL, sid, to, from, status, code string) (*http.Request, []byte) {
	form := url.Values{"MessageSid": {sid}, "SmsSid": {sid}, "MessageStatus": {status}, "SmsStatus": {status}, "To": {to}, "From": {from},
		"AccountSid": {AccountSID}, "ApiVersion": {"2010-04-01"}}
	if code != "" {
		form.Set("ErrorCode", code)
	}
	return signed(secret, hookURL, form)
}

// Inbound is a text someone sent to a number in the account. optOut is
// what Advanced Opt-Out made of it ("STOP"), or "".
func Inbound(secret, hookURL, sid, from, to, body, optOut string) (*http.Request, []byte) {
	form := url.Values{"MessageSid": {sid}, "SmsMessageSid": {sid}, "From": {from}, "To": {to}, "Body": {body}, "NumMedia": {"0"},
		"AccountSid": {AccountSID}, "ApiVersion": {"2010-04-01"}}
	if optOut != "" {
		form.Set("OptOutType", optOut)
	}
	return signed(secret, hookURL, form)
}
