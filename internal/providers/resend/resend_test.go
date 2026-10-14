package resend_test

import (
	"bytes"
	"context"
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

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/resend"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

const apiKey = "re_test_123"

var (
	signingSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("resend-test-signing-secret"))
	rotatedSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("resend-rotated-secret"))
)

type fake struct {
	mu            sync.Mutex
	sendingOnly   bool
	hookSecret    string
	hooks         []map[string]any
	domainStatus  string
	patchedDomain map[string]any
	suppressed    [][]string
	n             atomic.Int64
	// unmatched makes the batch endpoint answer without ids.
	unmatched bool
}

func newFake() *fake {
	return &fake{hookSecret: signingSecret, domainStatus: "verified", hooks: []map[string]any{
		{"id": "wh_1", "endpoint": "https://reputecast.acme.io/hooks/resend/tok-fetch", "status": "enabled", "events": []string{"email.delivered"}},
	}}
}

func writeErr(w http.ResponseWriter, status int, name, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"statusCode": status, "name": name, "message": msg})
}

func (f *fake) handler(env *contracttest.Env) http.Handler {
	mux := http.NewServeMux()
	guard := func(next http.HandlerFunc, sendingAllowed bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch {
			case env.Mode() == contracttest.ModeOutage:
				writeErr(w, http.StatusServiceUnavailable, "service_unavailable", "API is temporarily unavailable")
				return
			case env.Mode() == contracttest.ModeAuth || r.Header.Get("Authorization") != "Bearer "+apiKey:
				writeErr(w, http.StatusUnauthorized, "missing_api_key", "Missing API key in the authorization header.")
				return
			case !strings.HasPrefix(r.Header.Get("User-Agent"), "ReputeCast/"):
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, "error code: 1010")
				return
			}
			f.mu.Lock()
			restricted := f.sendingOnly && !sendingAllowed
			f.mu.Unlock()
			if restricted {
				writeErr(w, http.StatusUnauthorized, "restricted_api_key", "This API key is restricted to only send emails.")
				return
			}
			next(w, r)
		}
	}
	type email struct {
		To   []string          `json:"to"`
		Tags []json.RawMessage `json:"tags"`
	}
	mux.HandleFunc("POST /emails", guard(func(w http.ResponseWriter, r *http.Request) {
		var in email
		_ = json.NewDecoder(r.Body).Decode(&in)
		if !strings.HasPrefix(r.Header.Get("Idempotency-Key"), "rc-") {
			writeErr(w, http.StatusBadRequest, "invalid_idempotency_key", "missing key")
			return
		}
		if len(in.To) == 1 && in.To[0] == contracttest.RejectAddress {
			writeErr(w, http.StatusUnprocessableEntity, "validation_error", "Invalid `to` field. The email address needs to follow the `email@example.com` format.")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "re-" + strconv.FormatInt(f.n.Add(1), 10)})
	}, true))
	mux.HandleFunc("POST /emails/batch", guard(func(w http.ResponseWriter, r *http.Request) {
		var in []email
		_ = json.NewDecoder(r.Body).Decode(&in)
		if !strings.HasPrefix(r.Header.Get("Idempotency-Key"), "rc-batch-") || r.Header.Get("X-Batch-Validation") != "strict" {
			writeErr(w, http.StatusBadRequest, "invalid_idempotency_key", "missing key")
			return
		}
		out := []map[string]string{}
		for _, e := range in {
			if e.To[0] == contracttest.RejectAddress {
				writeErr(w, http.StatusUnprocessableEntity, "validation_error", "Invalid `to` field.")
				return
			}
			out = append(out, map[string]string{"id": "re-" + strconv.FormatInt(f.n.Add(1), 10)})
		}
		f.mu.Lock()
		if f.unmatched {
			out = out[:0]
		}
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"data": out})
	}, true))
	mux.HandleFunc("GET /domains", guard(func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		st := f.domainStatus
		f.mu.Unlock()
		w.Header().Set("X-Resend-Monthly-Quota", "1234")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": []map[string]string{{"id": "dom_1", "name": "news.acme.io", "status": st, "region": "us-east-1"}}})
	}, false))
	domain := func() map[string]any {
		f.mu.Lock()
		defer f.mu.Unlock()
		return map[string]any{"object": "domain", "id": "dom_1", "name": "news.acme.io", "status": f.domainStatus, "open_tracking": true, "click_tracking": true, "tracking_subdomain": "links",
			"records": []map[string]any{
				{"record": "DKIM", "name": "resend._domainkey", "value": "p=MIGfMA0GCSq", "type": "TXT", "status": "verified", "ttl": "Auto"},
				{"record": "SPF", "name": "send", "type": "MX", "ttl": "Auto", "status": "pending", "value": "feedback-smtp.us-east-1.amazonses.com", "priority": 10},
				{"record": "SPF", "name": "send", "value": `"v=spf1 include:amazonses.com ~all"`, "type": "TXT", "ttl": "Auto", "status": "failed"},
				{"record": "Tracking", "type": "CNAME", "name": "links", "value": "links1.resend-dns.com", "ttl": "Auto", "status": "not_started"},
			}}
	}
	mux.HandleFunc("POST /domains", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Name, Region string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in.Name == "taken.acme.io" {
			writeErr(w, http.StatusForbidden, "validation_error", "The `taken.acme.io` domain has been registered already.")
			return
		}
		_ = json.NewEncoder(w).Encode(domain())
	}, false))
	mux.HandleFunc("GET /domains/dom_1", guard(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(domain()) }, false))
	mux.HandleFunc("PATCH /domains/dom_1", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		f.patchedDomain = in
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]string{"object": "domain", "id": "dom_1"})
	}, false))
	mux.HandleFunc("POST /domains/dom_1/verify", guard(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"object": "domain", "id": "dom_1"})
	}, false))
	mux.HandleFunc("GET /webhooks", guard(func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "has_more": false, "data": f.hooks})
	}, false))
	mux.HandleFunc("GET /webhooks/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, h := range f.hooks {
			if h["id"] == r.PathValue("id") {
				out := map[string]any{"signing_secret": f.hookSecret}
				for k, v := range h {
					out[k] = v
				}
				_ = json.NewEncoder(w).Encode(out)
				return
			}
		}
		writeErr(w, http.StatusNotFound, "not_found", "Webhook not found")
	}, false))
	mux.HandleFunc("POST /webhooks", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		id := "wh_" + strconv.Itoa(len(f.hooks)+1)
		f.hooks = append(f.hooks, map[string]any{"id": id, "endpoint": in["endpoint"], "status": "enabled", "events": in["events"]})
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]string{"object": "webhook", "id": id, "signing_secret": signingSecret})
	}, false))
	mux.HandleFunc("PATCH /webhooks/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		for _, h := range f.hooks {
			if h["id"] == r.PathValue("id") {
				h["events"], h["status"] = in["events"], in["status"]
			}
		}
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]string{"object": "webhook", "id": r.PathValue("id")})
	}, false))
	mux.HandleFunc("GET /webhooks/{id}/events", guard(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": []map[string]string{{"status": "success"}, {"status": "success"}, {"status": "attempting"}, {"status": "failed"}}})
	}, false))
	mux.HandleFunc("GET /suppressions", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after") == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "has_more": true, "data": []map[string]any{
				{"id": "s3", "email": "Hard@Example.com", "origin": "bounce", "created_at": "2026-10-12 10:00:00.000+00"},
				{"id": "s2", "email": "angry@example.com", "origin": "complaint", "created_at": "2026-10-11 09:00:00.000+00"},
			}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "has_more": false, "data": []map[string]any{
			{"id": "s1", "email": "old@example.com", "origin": "manual", "created_at": "2026-01-02 09:00:00.000+00"},
		}})
	}, false))
	mux.HandleFunc("POST /suppressions/batch/add", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Emails []string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		f.suppressed = append(f.suppressed, in.Emails)
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}, false))
	mux.HandleFunc("GET /emails/metrics", guard(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"object":"metrics","totals":{"sent":2000,"delivered":1900,"bounced":40,"complained":1}}`)
	}, false))
	mux.HandleFunc("GET /emails/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		last := map[string]string{"e-del": "delivered", "e-bounce": "bounced", "e-spam": "complained", "e-queued": "queued", "e-open": "opened"}[r.PathValue("id")]
		if last == "" {
			writeErr(w, http.StatusNotFound, "not_found", "Email not found")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "email", "id": r.PathValue("id"), "to": []string{"Reader <Reader@Example.com>"}, "created_at": "2026-10-12 13:41:02.674981+00",
			"last_event": last, "tags": []map[string]string{{"name": "rc_message", "value": "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"}}})
	}, false))
	return mux
}

func newProvider(t *testing.T, env *contracttest.Env, creds map[string]string) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(resend.Settings{BaseURL: env.URL, Region: "eu-west-1", MonthlyLimit: 50000})
	if creds == nil {
		creds = map[string]string{"api_key": apiKey, "webhook_signing_secret": signingSecret}
	}
	c, _ := json.Marshal(creds)
	p, err := resend.New(providers.Config{Settings: settings, Credentials: c})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func signedRequest(t *testing.T, path, secret string, body []byte) *http.Request {
	t.Helper()
	id := "msg_" + uuid.NewString()
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig, err := webhookauth.SignStandard(secret, id, ts, body)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io"+path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("svix-id", id)
	r.Header.Set("svix-timestamp", ts)
	r.Header.Set("svix-signature", sig)
	return r
}

func webhook(file string) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		return signedRequest(t, "/hooks/resend/token", signingSecret, body), body
	}
}

func TestContract(t *testing.T) {
	f := newFake()
	contracttest.Run(t, contracttest.Suite{
		Adapter: resend.Adapter,
		API:     f.handler,
		New:     func(t *testing.T, env *contracttest.Env) providers.Provider { return newProvider(t, env, nil) },
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, _ []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || reqs[0].Path != "/emails/batch" {
				t.Fatalf("want one POST /emails/batch, got %+v", reqs)
			}
			var sent []struct {
				From    string
				To      []string
				Headers map[string]string
				Tags    []struct{ Name, Value string }
			}
			if err := json.Unmarshal(reqs[0].Body, &sent); err != nil {
				t.Fatal(err)
			}
			for i, m := range sent {
				if m.From != `"Acme Newsletter" <news@news.acme.io>` || len(m.To) != 1 || m.To[0] != msgs[i].To.Address {
					t.Fatalf("message %d: %+v", i, m)
				}
				if m.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" || m.Headers["X-ReputeCast-Message"] != msgs[i].ID.String() {
					t.Fatalf("message %d headers: %v", i, m.Headers)
				}
				if len(m.Tags) < 2 || m.Tags[0].Name != resend.TagMessage || m.Tags[0].Value != msgs[i].ID.String() {
					t.Fatalf("message %d tags: %v", i, m.Tags)
				}
			}
		},
		Webhooks: []contracttest.Webhook{
			{Name: "Delivered", Request: webhook("delivered.json"), Want: []providers.Event{{
				Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "56761188-7520-42d8-8898-ff6fc54ce618",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "email.delivered",
			}}},
			{Name: "BouncedPermanent", Request: webhook("bounced-permanent.json"), Want: []providers.Event{{
				Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "9a3f1c00-7520-42d8-8898-ff6fc54ce618",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "email.bounced · Permanent · NoEmail", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown",
			}}},
			{Name: "BouncedTransient", Request: webhook("bounced-transient.json"), Want: []providers.Event{{
				Type: providers.EventSoftBounce, Recipient: "b.h@yahoo.com", ProviderMessageID: "9a3f1c01-7520-42d8-8898-ff6fc54ce618",
				OriginalType: "email.bounced · Transient · MailboxFull", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full",
			}}},
			{Name: "DeliveryDelayed", Request: webhook("delivery-delayed.json"), Want: []providers.Event{{
				Type: providers.EventDeferred, Recipient: "p.s@yahoo.com", ProviderMessageID: "9a3f1c02-7520-42d8-8898-ff6fc54ce618",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e43"), OriginalType: "email.delivery_delayed", BounceClass: "temporary",
			}}},
			{Name: "Complained", Request: webhook("complained.json"), Want: []providers.Event{{
				Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "9a3f1c03-7520-42d8-8898-ff6fc54ce618",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "email.complained",
			}}},
			{Name: "Suppressed", Request: webhook("suppressed.json"), Want: []providers.Event{{
				Type: providers.EventRejected, Recipient: "old@example.org", ProviderMessageID: "9a3f1c04-7520-42d8-8898-ff6fc54ce618",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e44"), OriginalType: "email.suppressed · OnAccountSuppressionList", OriginalCode: "previous_bounce", BounceClass: "mailbox_unknown",
			}}},
			{Name: "Failed", Request: webhook("failed.json"), Want: []providers.Event{{
				Type: providers.EventRejected, Recipient: "late@example.com", ProviderMessageID: "9a3f1c05-7520-42d8-8898-ff6fc54ce618",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e45"), OriginalType: "email.failed", OriginalCode: "reached_daily_quota", BounceClass: "other",
			}}},
		},
	})
}

func TestBatchWithAnInvalidMessageSendsTheRestOneByOne(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, nil)
	msgs := []providers.Message{contracttest.Message(0), contracttest.Message(1), contracttest.Message(2)}
	msgs[1].To.Address = contracttest.RejectAddress
	res, err := p.Send(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Err != nil || res[2].Err != nil || res[0].ProviderMessageID == "" || res[2].ProviderMessageID == "" {
		t.Fatalf("valid messages must still go out: %+v", res)
	}
	if e := res[1].Err; e == nil || e.Temporary || e.Code != "validation_error" {
		t.Fatalf("the invalid message fails alone, permanently: %+v", res[1].Err)
	}
	paths, keys := []string{}, map[string]bool{}
	for _, r := range env.Requests() {
		paths = append(paths, r.Path)
		if r.Path == "/emails" {
			keys[r.Header.Get("Idempotency-Key")] = true
		}
	}
	if strings.Join(paths, ",") != "/emails/batch,/emails,/emails,/emails" {
		t.Fatalf("requests: %v", paths)
	}
	for _, m := range msgs {
		if !keys["rc-"+m.ID.String()] {
			t.Fatalf("single sends are keyed by the message id: %v", keys)
		}
	}
}

func TestAcceptedBatchWithoutIDs(t *testing.T) {
	f := newFake()
	f.unmatched = true
	env := contracttest.NewEnv(t, f.handler)
	msgs := []providers.Message{contracttest.Message(0), contracttest.Message(1)}
	res, err := newProvider(t, env, nil).Send(context.Background(), msgs)
	if err != nil || len(res) != 2 {
		t.Fatalf("an accepted batch is not an error: %+v, %v", res, err)
	}
	for i, r := range res {
		// Sending again could deliver the batch twice.
		if r.MessageID != msgs[i].ID || r.Err != nil || r.ProviderMessageID != msgs[i].ID.String() {
			t.Fatalf("result %d: %+v", i, r)
		}
	}
}

func TestQuotaAndSendingOnlyKey(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, nil).(providers.QuotaReporter)
	q, err := p.Quota(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if q.Kind != "monthly_plan" || q.Used != 1234 || q.Max != 50000 || q.Sandbox || q.Account != "1 verified domain · 50,000 per month plan" {
		t.Fatalf("quota: %+v", q)
	}
	f.mu.Lock()
	f.domainStatus = "pending"
	f.mu.Unlock()
	if q, _ := p.Quota(context.Background()); !q.Sandbox {
		t.Fatalf("no verified domain means test mail only: %+v", q)
	}
	f.mu.Lock()
	f.sendingOnly = true
	f.mu.Unlock()
	q, err = p.Quota(context.Background())
	if err != nil || q.Kind != "" || !strings.HasPrefix(q.Account, "Sending access key") {
		t.Fatalf("a sending-only key is valid, not rejected: %+v, %v", q, err)
	}
	env.SetMode(contracttest.ModeAuth)
	if _, err := p.Quota(context.Background()); !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("a missing key is rejected: %v", err)
	}
}

func TestWebhookSecretFromTheAPI(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, map[string]string{"api_key": apiKey}).(providers.WebhookVerifier)
	body, _ := os.ReadFile("testdata/delivered.json")
	if err := p.VerifyWebhook(context.Background(), signedRequest(t, "/hooks/resend/tok-fetch", signingSecret, body), body); err != nil {
		t.Fatalf("the secret of the webhook pointing at this URL is read from Resend: %v", err)
	}
	if err := p.VerifyWebhook(context.Background(), signedRequest(t, "/hooks/resend/unknown", signingSecret, body), body); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("no webhook for the URL: %v", err)
	}
	// Rotation: the cached secret is refreshed once it fails (after a minute).
	f.mu.Lock()
	f.hookSecret = rotatedSecret
	f.mu.Unlock()
	if err := p.VerifyWebhook(context.Background(), signedRequest(t, "/hooks/resend/tok-fetch", rotatedSecret, body), body); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a secret fetched less than a minute ago is not fetched again: %v", err)
	}
	stale := signedRequest(t, "/hooks/resend/tok-fetch", signingSecret, body)
	stale.Header.Set("svix-timestamp", strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10))
	if err := p.VerifyWebhook(context.Background(), stale, body); !errors.Is(err, webhookauth.ErrStale) {
		t.Fatalf("an old request is a replay: %v", err)
	}
}

func TestDomains(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, nil).(providers.DomainManager)
	recs, err := p.AddDomain(context.Background(), "news.acme.io")
	if err != nil {
		t.Fatal(err)
	}
	want := []providers.DNSRecord{
		{Purpose: "dkim", Type: "TXT", Name: "resend._domainkey.news.acme.io", Value: "p=MIGfMA0GCSq", Status: "verified"},
		{Purpose: "return_path", Type: "MX", Name: "send.news.acme.io", Value: "10 feedback-smtp.us-east-1.amazonses.com", Status: "pending"},
		{Purpose: "spf", Type: "TXT", Name: "send.news.acme.io", Value: "v=spf1 include:amazonses.com ~all", Status: "wrong_value"},
	}
	if len(recs) != len(want) {
		t.Fatalf("records: %+v", recs)
	}
	for i := range want {
		if recs[i] != want[i] {
			t.Fatalf("record %d: %+v, want %+v", i, recs[i], want[i])
		}
	}
	var created map[string]any
	for _, r := range env.Requests() {
		if r.Method == http.MethodPost && r.Path == "/domains" {
			_ = json.Unmarshal(r.Body, &created)
		}
	}
	if created["region"] != "eu-west-1" {
		t.Fatalf("domains are added in the configured region: %v", created)
	}
	f.mu.Lock()
	f.domainStatus = "not_started"
	f.mu.Unlock()
	if _, err := p.DomainStatus(context.Background(), "news.acme.io"); err != nil {
		t.Fatal(err)
	}
	verified := false
	for _, r := range env.Requests() {
		verified = verified || r.Path == "/domains/dom_1/verify"
	}
	if !verified {
		t.Fatal("a domain whose check never ran is sent for verification")
	}
	if _, err := p.DomainStatus(context.Background(), "other.acme.io"); err == nil {
		t.Fatal("an unknown domain is an error")
	}
}

func TestWebhookRegistration(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, nil)
	reg := p.(providers.WebhookRegistrar)
	if err := reg.RegisterWebhook(context.Background(), "https://reputecast.acme.io/hooks/resend/tok-new"); err != nil {
		t.Fatal(err)
	}
	if err := reg.RegisterWebhook(context.Background(), "https://reputecast.acme.io/hooks/resend/tok-fetch"); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	if len(f.hooks) != 2 || f.hooks[0]["status"] != "enabled" || len(f.hooks[0]["events"].([]any)) != 6 {
		t.Fatalf("hooks: %+v", f.hooks)
	}
	f.mu.Unlock()
	st, err := p.(providers.WebhookInspector).InspectWebhook(context.Background(), "https://reputecast.acme.io/hooks/resend/tok-fetch/")
	if err != nil || !st.Registered || st.ID != "wh_1" || st.Sent != 2 || st.Queued != 1 {
		t.Fatalf("inspect: %+v, %v", st, err)
	}
}

func TestSuppressionSync(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, nil).(providers.SuppressionSyncer)
	all, err := p.ListSuppressions(context.Background(), time.Time{})
	if err != nil || len(all) != 3 || all[0].Email != "hard@example.com" || all[0].Reason != "hard_bounce" || all[1].Reason != "complaint" || all[2].Reason != "manual" {
		t.Fatalf("all: %+v, %v", all, err)
	}
	recent, err := p.ListSuppressions(context.Background(), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(recent) != 2 {
		t.Fatalf("since: %+v, %v", recent, err)
	}
	emails := make([]string, 230)
	for i := range emails {
		emails[i] = "x" + strconv.Itoa(i) + "@example.com"
	}
	if err := p.AddSuppressions(context.Background(), emails); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.suppressed) != 3 || len(f.suppressed[0]) != 100 || len(f.suppressed[2]) != 30 {
		t.Fatalf("pushed in chunks of 100: %d", len(f.suppressed))
	}
}

func TestTracking(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	tm := newProvider(t, env, nil).(providers.TrackingManager)
	tr, err := tm.Tracking(context.Background())
	if err != nil || !tr.OpenPixel || !tr.LinkRewrite || tr.Detail != "Open tracking is on for news.acme.io and click tracking for news.acme.io." {
		t.Fatalf("tracking: %+v, %v", tr, err)
	}
	if err := tm.DisableTracking(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.patchedDomain["open_tracking"] != false || f.patchedDomain["click_tracking"] != false {
		t.Fatalf("patch: %v", f.patchedDomain)
	}
}

func TestBackfillAndReputation(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env, nil)
	evs, err := p.(providers.EventBackfiller).Backfill(context.Background(), []string{"e-del", "e-bounce", "e-spam", "e-queued", "e-open", "e-gone"})
	if err != nil {
		t.Fatal(err)
	}
	types := []providers.EventType{}
	for _, e := range evs {
		types = append(types, e.Type)
		if e.Recipient != "reader@example.com" || e.MessageID == nil || len(e.Raw) == 0 {
			t.Fatalf("event: %+v", e)
		}
	}
	if len(types) != 4 || types[0] != providers.EventDelivered || types[1] != providers.EventSoftBounce || types[2] != providers.EventComplaint || types[3] != providers.EventDelivered {
		t.Fatalf("backfilled: %v", types)
	}
	rep, err := p.(providers.ReputationReporter).Reputation(context.Background())
	if err != nil || rep.Bounce != 2 || rep.Window != "7 d" || rep.Complaint < 0.0526 || rep.Complaint > 0.0527 {
		t.Fatalf("reputation: %+v, %v", rep, err)
	}
}

func TestSettingsFromAForm(t *testing.T) {
	// A connection form may store numbers as strings.
	p, err := resend.New(providers.Config{Settings: json.RawMessage(`{"region":"eu-west-1","monthly_limit":"50,000"}`), Credentials: json.RawMessage(`{"api_key":"re_x"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Capabilities().Pricing; got != "plan, 50,000 per month" {
		t.Fatalf("pricing: %q", got)
	}
}
