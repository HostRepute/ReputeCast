package sendgrid_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/sendgrid"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

const apiKey = "SG.test-key"

var (
	keyOnce sync.Once
	key     *ecdsa.PrivateKey
)

func signingKey(t *testing.T) *ecdsa.PrivateKey {
	keyOnce.Do(func() {
		var err error
		if key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err != nil {
			t.Fatal(err)
		}
	})
	return key
}

func publicKey(t *testing.T) string {
	der, err := x509.MarshalPKIXPublicKey(&signingKey(t).PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(der)
}

func sign(t *testing.T, ts string, body []byte) string {
	h := sha256.Sum256(append([]byte(ts), body...))
	sig, err := ecdsa.SignASN1(rand.Reader, signingKey(t), h[:])
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{"message": msg, "field": nil}}})
}

// fake is a fake SendGrid API.
type fake struct {
	mu          sync.Mutex
	pubKey      string
	hooks       []map[string]any
	signedOn    []string
	restricted  bool
	sendStatus  int
	sendMessage string
	domainValid bool
	globalAdds  [][]string
	tracking    map[string]bool
	n           atomic.Int64
}

func (f *fake) handler(env *contracttest.Env) http.Handler {
	mux := http.NewServeMux()
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch {
			case env.Mode() == contracttest.ModeOutage:
				writeErr(w, http.StatusServiceUnavailable, "service unavailable")
				return
			case env.Mode() == contracttest.ModeAuth || r.Header.Get("Authorization") != "Bearer "+apiKey:
				writeErr(w, http.StatusUnauthorized, "The provided authorization grant is invalid, expired, or revoked")
				return
			}
			next(w, r)
		}
	}
	readOnly := func(next http.HandlerFunc) http.HandlerFunc {
		return guard(func(w http.ResponseWriter, r *http.Request) {
			f.mu.Lock()
			restricted := f.restricted
			f.mu.Unlock()
			if restricted {
				writeErr(w, http.StatusForbidden, "access forbidden")
				return
			}
			next(w, r)
		})
	}
	mux.HandleFunc("POST /v3/mail/send", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Personalizations []struct {
				To []struct{ Email string } `json:"to"`
			} `json:"personalizations"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Personalizations) != 1 {
			writeErr(w, http.StatusBadRequest, "invalid request")
			return
		}
		f.mu.Lock()
		status, msg := f.sendStatus, f.sendMessage
		f.mu.Unlock()
		if status != 0 {
			writeErr(w, status, msg)
			return
		}
		if in.Personalizations[0].To[0].Email == contracttest.RejectAddress {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"errors":[{"message":"Does not contain a valid address.","field":"personalizations.0.to.0.email","help":"http://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/errors.html#message.personalizations.to"}]}`)
			return
		}
		w.Header().Set("X-Message-Id", "XBg2anf2TqCy6WXK"+strconv.FormatInt(f.n.Add(1), 10))
		w.WriteHeader(http.StatusAccepted)
	}))
	mux.HandleFunc("GET /v3/user/credits", readOnly(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"remain":38212,"total":50000,"overage":0,"used":11788,"last_reset":"2026-10-01","next_reset":"2026-11-01","reset_frequency":"monthly"}`)
	}))
	mux.HandleFunc("GET /v3/user/account", readOnly(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"type":"paid","reputation":99.5}`)
	}))
	mux.HandleFunc("GET /v3/user/webhooks/event/settings/all", readOnly(func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"max_allowed": 5, "webhooks": f.hooks})
	}))
	save := func(w http.ResponseWriter, r *http.Request, id string) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		defer f.mu.Unlock()
		if id == "" {
			id = "wh-" + strconv.Itoa(len(f.hooks)+1)
			in["id"] = id
			f.hooks = append(f.hooks, in)
		} else {
			for _, h := range f.hooks {
				if h["id"] == id {
					for k, v := range in {
						h[k] = v
					}
				}
			}
			in["id"] = id
		}
		_ = json.NewEncoder(w).Encode(in)
	}
	mux.HandleFunc("POST /v3/user/webhooks/event/settings", readOnly(func(w http.ResponseWriter, r *http.Request) { save(w, r, "") }))
	mux.HandleFunc("PATCH /v3/user/webhooks/event/settings/{id}", readOnly(func(w http.ResponseWriter, r *http.Request) { save(w, r, r.PathValue("id")) }))
	mux.HandleFunc("PATCH /v3/user/webhooks/event/settings/signed/{id}", readOnly(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.signedOn = append(f.signedOn, r.PathValue("id"))
		for _, h := range f.hooks {
			if h["id"] == r.PathValue("id") {
				h["public_key"] = f.pubKey
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": r.PathValue("id"), "public_key": f.pubKey})
	}))
	mux.HandleFunc("POST /v3/user/webhooks/event/test", readOnly(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	domain := func() map[string]any {
		f.mu.Lock()
		defer f.mu.Unlock()
		return map[string]any{"id": 302183, "domain": "news.acme.io", "subdomain": "em1234", "default": false, "valid": f.domainValid, "dns": map[string]any{
			"mail_cname": map[string]any{"valid": f.domainValid, "type": "cname", "host": "em1234.news.acme.io", "data": "u1446226.wl.sendgrid.net"},
			"dkim1":      map[string]any{"valid": true, "type": "cname", "host": "s1._domainkey.news.acme.io", "data": "s1.domainkey.u1446226.wl.sendgrid.net"},
			"dkim2":      map[string]any{"valid": false, "type": "cname", "host": "s2._domainkey.news.acme.io", "data": "s2.domainkey.u1446226.wl.sendgrid.net"},
		}}
	}
	mux.HandleFunc("GET /v3/whitelabel/domains", readOnly(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("domain") != "news.acme.io" {
			_, _ = io.WriteString(w, `[]`)
			return
		}
		_ = json.NewEncoder(w).Encode([]any{domain()})
	}))
	mux.HandleFunc("POST /v3/whitelabel/domains", readOnly(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in["automatic_security"] != true {
			writeErr(w, http.StatusBadRequest, "automatic_security expected")
			return
		}
		d := domain()
		d["domain"] = in["domain"]
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(d)
	}))
	mux.HandleFunc("POST /v3/whitelabel/domains/302183/validate", readOnly(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":302183,"valid":false,"validation_results":{"mail_cname":{"valid":false,"reason":"Expected your CNAME record to be \"u1446226.wl.sendgrid.net\" but found \"example.com\"."},"dkim1":{"valid":true,"reason":null},"dkim2":{"valid":false,"reason":"No CNAME record found for s2._domainkey.news.acme.io"}}}`)
	}))
	mux.HandleFunc("GET /v3/suppression/{list}", readOnly(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		switch r.PathValue("list") {
		case "bounces":
			if offset == 0 {
				items := make([]map[string]any, 500)
				for i := range items {
					items[i] = map[string]any{"created": 1760000000 + i, "email": "B" + strconv.Itoa(i) + "@example.com", "reason": "550 5.1.1 unknown", "status": "5.1.1"}
				}
				_ = json.NewEncoder(w).Encode(items)
				return
			}
			_, _ = io.WriteString(w, `[{"created":1760276000,"email":"last@example.com","reason":"550 5.1.1 unknown","status":"5.1.1"}]`)
		case "spam_reports":
			_, _ = io.WriteString(w, `[{"created":1443651141,"email":"angry@example.com","ip":"10.63.202.100"}]`)
		case "unsubscribes":
			_, _ = io.WriteString(w, `[{"created":1443651141,"email":"gone@example.com"}]`)
		case "invalid_emails":
			_, _ = io.WriteString(w, `[{"created":1449953655,"email":"typo@gmial.com","reason":"Mail domain mentioned in email address is unknown"}]`)
		default:
			writeErr(w, http.StatusNotFound, "not found")
		}
	}))
	mux.HandleFunc("POST /v3/asm/suppressions/global", readOnly(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			RecipientEmails []string `json:"recipient_emails"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		f.globalAdds = append(f.globalAdds, in.RecipientEmails)
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(in)
	}))
	mux.HandleFunc("GET /v3/tracking_settings/{kind}", readOnly(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]bool{"enabled": f.tracking[r.PathValue("kind")], "enable_text": false})
	}))
	mux.HandleFunc("PATCH /v3/tracking_settings/{kind}", readOnly(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]bool
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		f.tracking[r.PathValue("kind")] = in["enabled"]
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(in)
	}))
	mux.HandleFunc("GET /v3/stats", readOnly(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"date":"2026-10-11","stats":[{"metrics":{"requests":1000,"delivered":960,"bounces":20,"spam_reports":1}}]},{"date":"2026-10-12","stats":[{"metrics":{"requests":1000,"delivered":980,"bounces":10,"spam_reports":0}}]}]`)
	}))
	return mux
}

// with runs fn while holding the fake's lock (the handler runs on other goroutines).
func (f *fake) with(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

func newFake(t *testing.T) *fake {
	return &fake{pubKey: publicKey(t), tracking: map[string]bool{"click": true, "open": true}}
}

func newProvider(t *testing.T, env *contracttest.Env, withKey bool) providers.Provider {
	t.Helper()
	s := sendgrid.Settings{BaseURL: env.URL, IPPool: "newsletters"}
	if withKey {
		s.WebhookVerificationKey = publicKey(t)
	}
	settings, _ := json.Marshal(s)
	p, err := sendgrid.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{"api_key":"` + apiKey + `"}`)})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func signedRequest(t *testing.T, path string, body []byte) *http.Request {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io"+path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(sendgrid.TimestampHeader, ts)
	r.Header.Set(sendgrid.SignatureHeader, sign(t, ts, body))
	return r
}

func TestContract(t *testing.T) {
	f := newFake(t)
	contracttest.Run(t, contracttest.Suite{
		Adapter: sendgrid.Adapter,
		API:     f.handler,
		New:     func(t *testing.T, env *contracttest.Env) providers.Provider { return newProvider(t, env, true) },
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || reqs[0].Path != "/v3/mail/send" {
				t.Fatalf("want one Mail Send call, got %+v", reqs)
			}
			var sent struct {
				Personalizations []struct {
					To         []struct{ Email, Name string }
					CustomArgs map[string]string `json:"custom_args"`
				}
				From             struct{ Email, Name string }
				Content          []struct{ Type, Value string }
				Headers          map[string]string
				IPPoolName       string                    `json:"ip_pool_name"`
				TrackingSettings map[string]map[string]any `json:"tracking_settings"`
			}
			if err := json.Unmarshal(reqs[0].Body, &sent); err != nil {
				t.Fatal(err)
			}
			m := msgs[0]
			p := sent.Personalizations[0]
			if p.To[0].Email != m.To.Address || p.CustomArgs[sendgrid.ArgMessage] != m.ID.String() || p.CustomArgs["campaign"] != "contract" {
				t.Fatalf("personalization: %+v", p)
			}
			if sent.From.Email != m.From.Address || sent.From.Name != m.From.Name || sent.IPPoolName != "newsletters" {
				t.Fatalf("from/pool: %+v %q", sent.From, sent.IPPoolName)
			}
			if len(sent.Content) != 2 || sent.Content[0].Type != "text/plain" || sent.Content[1].Type != "text/html" {
				t.Fatalf("content order: %+v", sent.Content)
			}
			if sent.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" || sent.Headers["X-ReputeCast-Message"] != m.ID.String() {
				t.Fatalf("headers: %v", sent.Headers)
			}
			for _, k := range []string{"click_tracking", "open_tracking", "subscription_tracking"} {
				if sent.TrackingSettings[k]["enable"] != false {
					t.Fatalf("%s must be off on every message: %v", k, sent.TrackingSettings)
				}
			}
			if !strings.HasPrefix(res[0].ProviderMessageID, "XBg2anf2TqCy6WXK") {
				t.Fatalf("provider id is the X-Message-Id: %q", res[0].ProviderMessageID)
			}
		},
		Webhooks: []contracttest.Webhook{{
			Name: "EventBatch",
			Request: func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
				body, err := os.ReadFile("testdata/events.json")
				if err != nil {
					t.Fatal(err)
				}
				return signedRequest(t, "/hooks/sendgrid/token", body), body
			},
			Want: []providers.Event{
				{Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "XBg2anf2TqCy6WXKQFhieQ",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "delivered", OriginalCode: "250 2.0.0"},
				{Type: providers.EventDeferred, Recipient: "p.s@yahoo.com", ProviderMessageID: "14c5d75ce93.dfd.64b469",
					OriginalType: "deferred", OriginalCode: "451 4.7.1", BounceClass: "temporary"},
				{Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "Ua9z9lSTSaqYBWJe_Xfc-Q",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "bounce · Invalid Address", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown"},
				{Type: providers.EventSoftBounce, Recipient: "k.n@hotmail.com", ProviderMessageID: "LRzXl_NHStOGhQ4kofSm_A",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "bounce · blocked · Reputation", OriginalCode: "550 5.7.1", BounceClass: "reputation"},
				{Type: providers.EventRejected, Recipient: "old@example.org", ProviderMessageID: "W0AYcfVMQqWIdpqhC3uCGQ",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e43"), OriginalType: "dropped · Bounced Address", OriginalCode: "5.0.0", BounceClass: "mailbox_unknown"},
				{Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "Pl9X5Z1aR2WZ2V4Z0q5Z7A",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e44"), OriginalType: "spamreport"},
			},
		}},
	})
}

// TestOfficialSignature checks Verify against the test vector of SendGrid's
// own Go library (helpers/eventwebhook): the trailing CRLF is signed too.
func TestOfficialSignature(t *testing.T) {
	const pub = "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE83T4O/n84iotIvIW4mdBgQ/7dAfSmpqIM8kF9mN1flpVKS3GRqe62gw+2fNNRaINXvVpiglSI8eNEc6wEA3F+g=="
	const sig = "MEUCIGHQVtGj+Y3LkG9fLcxf3qfI10QysgDWmMOVmxG0u6ZUAiEAyBiXDWzM+uOe5W0JuG+luQAbPIqHh89M15TluLtEZtM="
	body := []byte(`[{"email":"hello@world.com","event":"dropped","reason":"Bounced Address","sg_event_id":"ZHJvcC0xMDk5NDkxOS1MUnpYbF9OSFN0T0doUTRrb2ZTbV9BLTA","sg_message_id":"LRzXl_NHStOGhQ4kofSm_A.filterdrecv-p3mdw1-756b745b58-kmzbl-18-5F5FC76C-9.0","smtp-id":"<LRzXl_NHStOGhQ4kofSm_A@ismtpd0039p1iad1.sendgrid.net>","timestamp":1600112492}]` + "\r\n")
	if err := sendgrid.Verify(pub, sig, "1600112502", body); err != nil {
		t.Fatalf("official vector: %v", err)
	}
	if err := sendgrid.Verify(pub, sig, "1600112502", bytes.TrimSpace(body)); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatal("the raw body is signed, trailing CRLF included")
	}
	if err := sendgrid.Verify(pub, sig, "1600112503", body); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatal("the timestamp is signed")
	}
}

func TestMessageID(t *testing.T) {
	for in, want := range map[string]string{
		"XBg2anf2TqCy6WXKQFhieQ.filter0905p1mdw1-4434-59E0C6FF-3.0":               "XBg2anf2TqCy6WXKQFhieQ",
		" 14c5d75ce93.dfd.64b469.filter0001.16648.5515E0B88.0":                    "14c5d75ce93.dfd.64b469",
		"LRzXl_NHStOGhQ4kofSm_A.recvd-5c8b7d6dc9-mkhpx-1-5E1D1B8E-1.0":            "LRzXl_NHStOGhQ4kofSm_A",
		"14c5d75ce93.dfd.64b469.filter0001.16648.5515E0B88.000000000000000000000": "14c5d75ce93.dfd.64b469",
		"Ua9z9lSTSaqYBWJe_Xfc-Q": "Ua9z9lSTSaqYBWJe_Xfc-Q",
	} {
		if got := sendgrid.MessageID(in); got != want {
			t.Fatalf("MessageID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSendErrors(t *testing.T) {
	f := newFake(t)
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, true)
	f.with(func() {
		f.sendStatus, f.sendMessage = http.StatusForbidden, "The from address does not match a verified Sender Identity. Mail cannot be sent until this error is resolved."
	})
	res, err := p.Send(context.Background(), []providers.Message{contracttest.Message(0)})
	if err != nil || res[0].Err == nil || res[0].Err.Temporary || res[0].Err.Code != "sender_not_verified" {
		t.Fatalf("an unverified sender fails the message, not the credentials: %+v, %v", res, err)
	}
	f.with(func() { f.sendStatus, f.sendMessage = http.StatusUnauthorized, "Maximum credits exceeded" })
	_, err = p.Send(context.Background(), []providers.Message{contracttest.Message(0)})
	var ce *providers.CallError
	if !errors.As(err, &ce) || !ce.Throttled || errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("running out of credits is a quota, not a bad key: %v", err)
	}
}

func TestQuotaAndRestrictedKey(t *testing.T) {
	f := newFake(t)
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, true).(providers.QuotaReporter)
	q, err := p.Quota(context.Background())
	if err != nil || q.Kind != "monthly_plan" || q.Max != 50000 || q.Used != 11788 || q.Account != "Paid account · reputation 99.5%" {
		t.Fatalf("quota: %+v, %v", q, err)
	}
	f.with(func() { f.restricted = true })
	q, err = p.Quota(context.Background())
	if err != nil || q.Kind != "" || !strings.HasPrefix(q.Account, "Restricted API key") {
		t.Fatalf("a Mail Send-only key is valid: %+v, %v", q, err)
	}
	env.SetMode(contracttest.ModeAuth)
	if _, err := p.Quota(context.Background()); !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("a revoked key is rejected: %v", err)
	}
}

func TestWebhookRegistrationAndKeyLookup(t *testing.T) {
	f := newFake(t)
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, false)
	const hook = "https://reputecast.acme.io/hooks/sendgrid/tok-1"
	if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), hook); err != nil {
		t.Fatal(err)
	}
	if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), hook+"/"); err != nil {
		t.Fatal(err)
	}
	f.with(func() {
		if len(f.hooks) != 1 || f.hooks[0]["open"] != false || f.hooks[0]["bounce"] != true || len(f.signedOn) != 2 {
			t.Fatalf("one webhook, updated in place, signing on: %+v %v", f.hooks, f.signedOn)
		}
		// A webhook set up elsewhere, whose key no adapter has seen yet.
		f.hooks = append(f.hooks, map[string]any{"id": "wh-9", "url": "https://reputecast.acme.io/hooks/sendgrid/tok-2", "enabled": true, "public_key": f.pubKey})
	})
	st, err := p.(providers.WebhookInspector).InspectWebhook(context.Background(), hook)
	if err != nil || !st.Registered || st.ID != "wh-1" {
		t.Fatalf("inspect: %+v, %v", st, err)
	}
	if err := p.(providers.WebhookTester).SendTestEvent(context.Background(), hook); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile("testdata/events.json")
	// A fresh adapter (the factory builds one per request) verifies with the
	// key registration returned, and reads the key of another webhook from
	// SendGrid for the webhook pointing at the request's path.
	fresh := newProvider(t, env, false).(providers.WebhookVerifier)
	if err := fresh.VerifyWebhook(context.Background(), signedRequest(t, "/hooks/sendgrid/tok-1", body), body); err != nil {
		t.Fatalf("verify with the key registration returned: %v", err)
	}
	if err := fresh.VerifyWebhook(context.Background(), signedRequest(t, "/prefix/hooks/sendgrid/tok-2", body), body); err == nil {
		t.Fatal("the request path must end the webhook's path, not the other way round")
	}
	if err := fresh.VerifyWebhook(context.Background(), signedRequest(t, "/hooks/sendgrid/tok-2", body), body); err != nil {
		t.Fatalf("verify with the key read from SendGrid: %v", err)
	}
	if err := fresh.VerifyWebhook(context.Background(), signedRequest(t, "/hooks/sendgrid/other", body), body); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("no webhook for this URL: %v", err)
	}
	// Retries can carry the first attempt's signature for 24 hours; older
	// requests are replays.
	for age, ok := range map[time.Duration]bool{20 * time.Hour: true, sendgrid.Window + time.Hour: false} {
		old := signedRequest(t, "/hooks/sendgrid/tok-1", body)
		ts := strconv.FormatInt(time.Now().Add(-age).Unix(), 10)
		old.Header.Set(sendgrid.TimestampHeader, ts)
		old.Header.Set(sendgrid.SignatureHeader, sign(t, ts, body))
		if err := fresh.VerifyWebhook(context.Background(), old, body); (err == nil) != ok || (err != nil && !errors.Is(err, webhookauth.ErrStale)) {
			t.Fatalf("a request signed %s ago: %v", age, err)
		}
	}
	unsigned, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/sendgrid/tok-1", bytes.NewReader(body))
	if err := fresh.VerifyWebhook(context.Background(), unsigned, body); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("an unsigned request is refused: %v", err)
	}
}

func TestDomains(t *testing.T) {
	f := newFake(t)
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, true).(providers.DomainManager)
	recs, err := p.AddDomain(context.Background(), "fresh.acme.io")
	if err != nil || len(recs) != 3 || recs[0].Purpose != "return_path" || recs[0].Type != "CNAME" || recs[1].Purpose != "dkim" || recs[1].Status != "verified" {
		t.Fatalf("add: %+v, %v", recs, err)
	}
	recs, err = p.DomainStatus(context.Background(), "news.acme.io")
	if err != nil {
		t.Fatal(err)
	}
	if recs[0].Status != "wrong_value" || recs[1].Status != "verified" || recs[2].Status != "missing" {
		t.Fatalf("validation results: %+v", recs)
	}
	if _, err := p.DomainStatus(context.Background(), "other.acme.io"); err == nil {
		t.Fatal("an unknown domain is an error")
	}
}

func TestSuppressionSync(t *testing.T) {
	f := newFake(t)
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, true).(providers.SuppressionSyncer)
	list, err := p.ListSuppressions(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 504 || list[0].Email != "b0@example.com" || list[0].Reason != "hard_bounce" || list[0].Code != "5.1.1" {
		t.Fatalf("bounces: %d %+v", len(list), list[0])
	}
	reasons := map[string]string{}
	for _, s := range list[500:] {
		reasons[s.Email] = s.Reason + "/" + s.Code
	}
	if reasons["typo@gmial.com"] != "hard_bounce/invalid_email" || reasons["angry@example.com"] != "complaint/spam_report" || reasons["gone@example.com"] != "unsubscribe/global_unsubscribe" {
		t.Fatalf("reasons: %v", reasons)
	}
	since := false
	for _, r := range env.Requests() {
		since = since || strings.Contains(r.Path, "start_time=1767225600")
	}
	if !since {
		t.Fatal("only suppressions since the last sync are listed")
	}
	emails := make([]string, 1200)
	for i := range emails {
		emails[i] = "x" + strconv.Itoa(i) + "@example.com"
	}
	if err := p.AddSuppressions(context.Background(), emails); err != nil {
		t.Fatal(err)
	}
	f.with(func() {
		if len(f.globalAdds) != 3 || len(f.globalAdds[2]) != 200 {
			t.Fatalf("chunks: %d", len(f.globalAdds))
		}
	})
}

func TestTrackingAndReputation(t *testing.T) {
	f := newFake(t)
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, true)
	tm := p.(providers.TrackingManager)
	tr, err := tm.Tracking(context.Background())
	if err != nil || tr.OpenPixel || tr.LinkRewrite || !strings.Contains(tr.Detail, "Account-level open and click tracking is on") {
		t.Fatalf("tracking is off on ReputeCast's messages whatever the account says: %+v, %v", tr, err)
	}
	if err := tm.DisableTracking(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.with(func() {
		if f.tracking["click"] || f.tracking["open"] {
			t.Fatalf("disable: %v", f.tracking)
		}
	})
	rep, err := p.(providers.ReputationReporter).Reputation(context.Background())
	if err != nil || rep.Bounce != 1.5 || rep.Window != "7 d" || rep.Complaint < 0.0515 || rep.Complaint > 0.0516 {
		t.Fatalf("reputation: %+v, %v", rep, err)
	}
}
