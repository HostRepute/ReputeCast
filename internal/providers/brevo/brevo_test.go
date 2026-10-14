package brevo_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/brevo"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
)

const (
	apiKey = "xkeysib-test"
	secret = "whsec-test-brevo"
)

type fake struct {
	mu        sync.Mutex
	keys      map[string]bool
	hooks     []map[string]any
	plan      string
	throttled bool
	n         atomic.Int64
	created   int
	// gone are webhook ids still listed but deleted already, as in a race
	// with another client.
	gone map[int]bool
}

func (f *fake) with(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
}

func (f *fake) handler(env *contracttest.Env) http.Handler {
	mux := http.NewServeMux()
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch {
			case env.Mode() == contracttest.ModeOutage:
				writeErr(w, http.StatusBadGateway, "", "Bad gateway")
				return
			case env.Mode() == contracttest.ModeAuth || r.Header.Get("api-key") != apiKey:
				writeErr(w, http.StatusUnauthorized, "unauthorized", "Key not found")
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("POST /v3/smtp/email", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			To      []struct{ Email string } `json:"to"`
			Headers map[string]string        `json:"headers"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.To) != 1 {
			writeErr(w, http.StatusBadRequest, "invalid_parameter", "bad body")
			return
		}
		key := in.Headers["IdempotencyKey"]
		f.mu.Lock()
		seen := f.keys[key]
		f.keys[key] = true
		f.mu.Unlock()
		switch {
		case key == "":
			writeErr(w, http.StatusBadRequest, "invalid_parameter", "idempotency key expected")
		case seen:
			writeErr(w, http.StatusBadRequest, "duplicate_parameter", "Request already processed")
		case in.To[0].Email == "twice@example.com":
			writeErr(w, http.StatusBadRequest, "duplicate_parameter", "duplicate email address in to and bcc")
		case in.To[0].Email == contracttest.RejectAddress:
			writeErr(w, http.StatusBadRequest, "invalid_parameter", "email is not valid in to")
		default:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"messageId": "<202610121341." + strconv.FormatInt(f.n.Add(1), 10) + "@smtp-relay.mailin.fr>"})
		}
	}))
	mux.HandleFunc("GET /v3/account", guard(func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		plan := f.plan
		f.mu.Unlock()
		_, _ = io.WriteString(w, `{"email":"linh@acme.io","companyName":"Acme Newsletter","plan":[`+plan+`,{"type":"sms","creditsType":"sendLimit","credits":15}],"relay":{"enabled":true}}`)
	}))
	mux.HandleFunc("GET /v3/smtp/statistics/aggregatedReport", guard(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"range":"2026-10-06|2026-10-12","requests":4000,"delivered":3800,"hardBounces":60,"softBounces":20,"spamReports":2,"blocked":5,"invalid":1}`)
	}))
	domain := func(authenticated bool) string {
		return `{"domain":"news.acme.io","verified":true,"authenticated":` + strconv.FormatBool(authenticated) + `,"dns_records":{` +
			`"dkim_record":{"type":"TXT","value":"v=DKIM1; k=rsa; p=MIGfMA0GCSq","host_name":"mail._domainkey.","status":` + strconv.FormatBool(authenticated) + `},` +
			`"brevo_code":{"type":"TXT","value":"brevo-code:abc123def456","host_name":"@","status":true},` +
			`"dmarc_record":{"type":"TXT","value":"v=DMARC1; p=none; rua=mailto:rua@dmarc.brevo.com","host_name":"_dmarc","status":false}}}`
	}
	var authenticated atomic.Bool
	mux.HandleFunc("POST /v3/senders/domains", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Name string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in.Name == "news.acme.io" {
			writeErr(w, http.StatusBadRequest, "duplicate_parameter", "Domain already exists")
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, strings.Replace(domain(false), `"domain":"news.acme.io"`, `"id":5,"domain_name":"`+in.Name+`"`, 1))
	}))
	mux.HandleFunc("GET /v3/senders/domains/{domain}", guard(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, domain(authenticated.Load()))
	}))
	mux.HandleFunc("PUT /v3/senders/domains/{domain}/authenticate", guard(func(w http.ResponseWriter, _ *http.Request) {
		authenticated.Store(true)
		_, _ = io.WriteString(w, `{"domain_name":"news.acme.io","message":"Domain has been authenticated successfully."}`)
	}))
	mux.HandleFunc("GET /v3/webhooks", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "transactional" {
			writeErr(w, http.StatusBadRequest, "invalid_parameter", "type")
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"webhooks": f.hooks})
	}))
	mux.HandleFunc("POST /v3/webhooks", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		in["id"] = 9863 + f.created
		f.created++
		f.hooks = append(f.hooks, in)
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": in["id"]})
	}))
	mux.HandleFunc("PUT /v3/webhooks/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		id, _ := strconv.Atoi(r.PathValue("id"))
		f.mu.Lock()
		for _, h := range f.hooks {
			if h["id"] == id {
				for k, v := range in {
					h[k] = v
				}
			}
		}
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.HandleFunc("DELETE /v3/webhooks/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		f.mu.Lock()
		defer f.mu.Unlock()
		for i, h := range f.hooks {
			if h["id"] == id && !f.gone[id] {
				f.hooks = slices.Delete(f.hooks, i, i+1)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		writeErr(w, http.StatusNotFound, "document_not_found", "Webhook ID not found")
	}))
	mux.HandleFunc("GET /v3/smtp/blockedContacts", guard(func(w http.ResponseWriter, r *http.Request) {
		if (r.URL.Query().Get("startDate") == "") != (r.URL.Query().Get("endDate") == "") {
			writeErr(w, http.StatusBadRequest, "invalid_parameter", "startDate and endDate go together")
			return
		}
		if r.URL.Query().Get("offset") != "0" {
			_, _ = io.WriteString(w, `{"contacts":[],"count":3}`)
			return
		}
		_, _ = io.WriteString(w, `{"contacts":[{"email":"Hard@Example.com","senderEmail":"news@news.acme.io","reason":{"code":"hardBounce","message":"Hard bounce"},"blockedAt":"2026-10-11T12:30:00Z"},{"email":"angry@example.com","senderEmail":null,"reason":{"code":"contactFlaggedAsSpam","message":"Spam"},"blockedAt":"2026-10-12T12:30:00Z"},{"email":"gone@example.com","reason":{"code":"unsubscribedViaEmail","message":"Unsubscribed"},"blockedAt":"2026-10-12T13:30:00Z"}],"count":3}`)
	}))
	mux.HandleFunc("GET /v3/smtp/statistics/events", guard(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		throttled := f.throttled
		f.mu.Unlock()
		if throttled {
			writeErr(w, http.StatusTooManyRequests, "too_many_requests", "Rate limit exceeded")
			return
		}
		id := r.URL.Query().Get("messageId")
		switch id {
		case "<m-1@smtp-relay.mailin.fr>":
			_, _ = io.WriteString(w, `{"events":[{"date":"2026-10-12T13:41:05Z","email":"a@example.com","event":"opened","messageId":"`+id+`"},{"date":"2026-10-12T13:41:02Z","email":"a@example.com","event":"delivered","messageId":"`+id+`"},{"date":"2026-10-12T13:41:00Z","email":"a@example.com","event":"requests","messageId":"`+id+`"}]}`)
		case "<m-2@smtp-relay.mailin.fr>":
			_, _ = io.WriteString(w, `{"events":[{"date":"2026-10-12T13:41:02Z","email":"b@example.com","event":"hardBounces","reason":"550 5.1.1 unknown user","messageId":"`+id+`"}]}`)
		default:
			_, _ = io.WriteString(w, `{"events":[]}`)
		}
	}))
	return mux
}

func newFake() *fake {
	return &fake{keys: map[string]bool{}, plan: `{"type":"subscription","creditsType":"sendLimit","credits":15000}`}
}

func newProvider(t *testing.T, env *contracttest.Env) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(brevo.Settings{BaseURL: env.URL, MonthlyLimit: 20000})
	p, err := brevo.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{"api_key":"` + apiKey + `"}`), WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func webhook(file string) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/brevo/token", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+secret)
		return r, body
	}
}

func TestContract(t *testing.T) {
	f := newFake()
	contracttest.Run(t, contracttest.Suite{
		Adapter: brevo.Adapter,
		API:     f.handler,
		New:     newProvider,
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || reqs[0].Path != "/v3/smtp/email" {
				t.Fatalf("want one POST /v3/smtp/email, got %+v", reqs)
			}
			var sent struct {
				Sender  struct{ Email, Name string } `json:"sender"`
				Headers map[string]string            `json:"headers"`
				Tags    []string                     `json:"tags"`
			}
			if err := json.Unmarshal(reqs[0].Body, &sent); err != nil {
				t.Fatal(err)
			}
			m := msgs[0]
			if sent.Headers[brevo.CustomHeader] != "rc_message:"+m.ID.String()+"|rc_stream:broadcast" || sent.Headers["IdempotencyKey"] != m.ID.String() {
				t.Fatalf("headers: %v", sent.Headers)
			}
			if _, ok := sent.Headers["List-Unsubscribe"]; ok {
				t.Fatal("Brevo takes no standard headers; it sets its own List-Unsubscribe")
			}
			if sent.Headers["Feedback-ID"] != "contract:acme:reputecast" || sent.Sender.Name != "Acme Newsletter" {
				t.Fatalf("headers: %v, sender %+v", sent.Headers, sent.Sender)
			}
			if !strings.HasPrefix(res[0].ProviderMessageID, "202610121341.") || strings.ContainsAny(res[0].ProviderMessageID, "<>") {
				t.Fatalf("provider id: %q", res[0].ProviderMessageID)
			}
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			r.Header.Set("Authorization", "Bearer wrong-secret")
			r.Body = io.NopCloser(bytes.NewReader(body))
			return r, body
		},
		Webhooks: []contracttest.Webhook{
			{Name: "Delivered", Request: webhook("delivered.json"), Want: []providers.Event{{
				Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "202610121341.5787683@smtp-relay.mailin.fr",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "delivered",
			}}},
			{Name: "HardBounce", Request: webhook("hard-bounce.json"), Want: []providers.Event{{
				Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "202610121341.5787684@smtp-relay.mailin.fr",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "hard_bounce", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown",
			}}},
			{Name: "SoftBounce", Request: webhook("soft-bounce.json"), Want: []providers.Event{{
				Type: providers.EventSoftBounce, Recipient: "b.h@yahoo.com", ProviderMessageID: "202610121341.5787685@smtp-relay.mailin.fr",
				OriginalType: "soft_bounce", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full",
			}}},
			{Name: "Spam", Request: webhook("spam.json"), Want: []providers.Event{{
				Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "202610121341.5787686@smtp-relay.mailin.fr",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "spam",
			}}},
			{Name: "Blocked", Request: webhook("blocked.json"), Want: []providers.Event{{
				Type: providers.EventRejected, Recipient: "old@example.org", ProviderMessageID: "202610121341.5787687@smtp-relay.mailin.fr",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e44"), OriginalType: "blocked", BounceClass: "policy_block",
			}}},
			{Name: "Unsubscribed", Request: webhook("unsubscribed.json"), Want: []providers.Event{{
				Type: providers.EventUnsubscribe, Recipient: "gone@example.com", ProviderMessageID: "202610121341.5787688@smtp-relay.mailin.fr",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e45"), OriginalType: "unsubscribed",
			}}},
		},
	})
}

func TestEventTimes(t *testing.T) {
	env := contracttest.NewEnv(t, nil)
	env.URL = "https://api.brevo.test"
	p := newProvider(t, env)
	for file, want := range map[string]time.Time{
		"delivered.json":   time.UnixMilli(1760276462223).UTC(),
		"soft-bounce.json": time.Unix(1760276480, 0).UTC(),
	} {
		r, body := webhook(file)(t, env)
		evs, err := p.ParseWebhook(context.Background(), r, body)
		if err != nil || !evs[0].OccurredAt.Equal(want) {
			t.Fatalf("%s: %v, %v", file, evs, err)
		}
	}
}

func TestRetriedSendIsNotSentTwice(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	m := contracttest.Message(0)
	first, err := p.Send(context.Background(), []providers.Message{m})
	if err != nil || first[0].Err != nil {
		t.Fatal(first, err)
	}
	again, err := p.Send(context.Background(), []providers.Message{m})
	if err != nil || again[0].Err != nil || again[0].ProviderMessageID != m.ID.String() {
		t.Fatalf("a retry Brevo refuses as a duplicate was sent the first time: %+v, %v", again, err)
	}
	// The same code about another parameter is a refusal, not a success.
	other := contracttest.Message(1)
	other.To.Address = "twice@example.com"
	res, err := p.Send(context.Background(), []providers.Message{other})
	if err != nil || res[0].Err == nil || res[0].Err.Code != "duplicate_parameter" || res[0].Err.Temporary {
		t.Fatalf("a duplicated parameter fails the message: %+v, %v", res, err)
	}
}

func TestQuotaAndReputation(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	q, err := p.(providers.QuotaReporter).Quota(context.Background())
	if err != nil || q.Kind != "monthly_plan" || q.Max != 20000 || q.Used != 5000 || q.Account != "Acme Newsletter · subscription · 15000 emails left" {
		t.Fatalf("subscription: %+v, %v", q, err)
	}
	f.with(func() { f.plan = `{"type":"free","creditsType":"sendLimit","credits":250}` })
	q, err = p.(providers.QuotaReporter).Quota(context.Background())
	if err != nil || q.Kind != "rolling_24h" || q.Max != 300 || q.Used != 50 {
		t.Fatalf("free plan: %+v, %v", q, err)
	}
	rep, err := p.(providers.ReputationReporter).Reputation(context.Background())
	if err != nil || rep.Bounce != 2 || rep.Complaint < 0.0526 || rep.Complaint > 0.0527 {
		t.Fatalf("reputation: %+v, %v", rep, err)
	}
	env.SetMode(contracttest.ModeAuth)
	if _, err := p.(providers.QuotaReporter).Quota(context.Background()); !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("rejected key: %v", err)
	}
}

func TestDomains(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env).(providers.DomainManager)
	recs, err := p.AddDomain(context.Background(), "fresh.acme.io")
	if err != nil {
		t.Fatal(err)
	}
	want := []providers.DNSRecord{
		{Purpose: "verification", Type: "TXT", Name: "fresh.acme.io", Value: "brevo-code:abc123def456", Status: "verified"},
		{Purpose: "dkim", Type: "TXT", Name: "mail._domainkey.fresh.acme.io", Value: "v=DKIM1; k=rsa; p=MIGfMA0GCSq", Status: "pending"},
		{Purpose: "dmarc", Type: "TXT", Name: "_dmarc.fresh.acme.io", Value: "v=DMARC1; p=none; rua=mailto:rua@dmarc.brevo.com", Status: "pending"},
	}
	if len(recs) != 3 || recs[0] != want[0] || recs[1] != want[1] || recs[2] != want[2] {
		t.Fatalf("records: %+v", recs)
	}
	// An existing domain is read; one not authenticated is authenticated.
	recs, err = p.AddDomain(context.Background(), "news.acme.io")
	if err != nil || recs[1].Status != "verified" {
		t.Fatalf("existing: %+v, %v", recs, err)
	}
}

func TestWebhooksSuppressionsBackfill(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	const hook = "https://reputecast.acme.io/hooks/brevo/tok-1"
	for range 2 {
		if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), hook); err != nil {
			t.Fatal(err)
		}
	}
	f.with(func() {
		if len(f.hooks) != 1 {
			t.Fatalf("hooks: %v", f.hooks)
		}
		auth := f.hooks[0]["auth"].(map[string]any)
		if auth["type"] != "bearer" || auth["token"] != secret || f.hooks[0]["batched"] != false || len(f.hooks[0]["events"].([]any)) != 8 {
			t.Fatalf("webhook: %v", f.hooks[0])
		}
	})
	st, err := p.(providers.WebhookInspector).InspectWebhook(context.Background(), hook)
	if err != nil || !st.Registered || st.ID != "9863" {
		t.Fatalf("inspect: %+v, %v", st, err)
	}
	sync := p.(providers.SuppressionSyncer)
	list, err := sync.ListSuppressions(context.Background(), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(list) != 3 || list[0].Email != "hard@example.com" || list[0].Reason != "hard_bounce" || list[1].Reason != "complaint" || list[2].Reason != "unsubscribe" {
		t.Fatalf("suppressions: %+v, %v", list, err)
	}
	if err := sync.AddSuppressions(context.Background(), []string{"a@example.com"}); !errors.Is(err, providers.ErrNotSupported) {
		t.Fatalf("Brevo can't take suppressions over its API: %v", err)
	}
	evs, err := p.(providers.EventBackfiller).Backfill(context.Background(), []string{"m-1@smtp-relay.mailin.fr", "m-2@smtp-relay.mailin.fr", "m-3@smtp-relay.mailin.fr"})
	if err != nil || len(evs) != 2 || evs[0].Type != providers.EventDelivered || evs[1].Type != providers.EventHardBounce || evs[1].OriginalCode != "550 5.1.1" {
		t.Fatalf("backfill: %+v, %v", evs, err)
	}
	f.with(func() { f.throttled = true })
	evs, err = p.(providers.EventBackfiller).Backfill(context.Background(), []string{"m-1@smtp-relay.mailin.fr"})
	if err != nil || len(evs) != 0 {
		t.Fatalf("a throttled backfill stops quietly: %+v, %v", evs, err)
	}
}

func TestUnregisterWebhook(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	ctx := context.Background()
	const hook, other = "https://reputecast.acme.io/hooks/brevo/tok-1", "https://hooks.acme.io/brevo"
	for _, u := range []string{hook, other} {
		if err := p.(providers.WebhookRegistrar).RegisterWebhook(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	// One someone else deleted after the list: Brevo answers 404.
	f.with(func() {
		f.hooks = append(f.hooks, map[string]any{"id": 7, "url": hook + "/", "type": "transactional"})
		f.gone = map[int]bool{7: true}
	})
	if err := p.(providers.WebhookUnregistrar).UnregisterWebhook(ctx, hook); err != nil {
		t.Fatal(err)
	}
	var deleted []string
	for _, r := range env.Requests() {
		if r.Method == http.MethodDelete {
			deleted = append(deleted, r.Path)
		}
	}
	if !slices.Equal(deleted, []string{"/v3/webhooks/9863", "/v3/webhooks/7"}) {
		t.Fatalf("deleted %v", deleted)
	}
	f.with(func() {
		if len(f.hooks) != 2 || f.hooks[0]["url"] != other || f.hooks[1]["id"] != 7 {
			t.Fatalf("webhooks left: %v", f.hooks)
		}
	})
}
