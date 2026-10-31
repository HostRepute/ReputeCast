package sparkpost_test

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
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/sparkpost"
)

const (
	apiKey = "sp-test-key"
	secret = "whsec-test-sparkpost"
)

type fake struct {
	mu           sync.Mutex
	hooks        []map[string]any
	stale        []map[string]any // listed but already deleted: deleting one answers 404
	hookN        int
	accountPut   map[string]any
	suppressions [][]map[string]string
	limitHit     bool
	n            atomic.Int64
	// keys are the transmissions' Idempotency-Keys with the first request's
	// body and answer, as SparkPost keeps them for 24 hours; inProgress
	// answers as if the first request with a key were still running.
	keys       map[string]keyedRequest
	inProgress bool
	// sendingIPs is GET /api/v1/sending-ips's answer; "" lists the
	// account's IPs in two pools.
	sendingIPs string
}

// sendingIPs are the account's dedicated IPs: two in the "marketing" pool
// the provider sends through, one in the default pool.
const sendingIPs = `{"results":[` +
	`{"external_ip":"198.51.100.31","hostname":"mta31.sparkpostmail.com","ip_pool":"marketing","customer_provided":false,"auto_warmup_enabled":false},` +
	`{"external_ip":"198.51.100.4","hostname":"mta4.sparkpostmail.com","ip_pool":"default","customer_provided":false,"auto_warmup_enabled":false},` +
	`{"external_ip":"198.51.100.30","hostname":"mta30.sparkpostmail.com","ip_pool":"marketing","customer_provided":false,"auto_warmup_enabled":true,"auto_warmup_stage":5}]}`

type keyedRequest struct {
	body   []byte
	answer string
}

func (f *fake) with(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{"message": msg, "code": code}}})
}

func (f *fake) handler(env *contracttest.Env) http.Handler {
	mux := http.NewServeMux()
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch {
			case env.Mode() == contracttest.ModeOutage:
				writeErr(w, http.StatusServiceUnavailable, "1900", "Message generation module not available")
				return
			case env.Mode() == contracttest.ModeAuth || r.Header.Get("Authorization") != apiKey:
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"errors": [ {"message": "Unauthorized."} ]}`)
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("POST /api/v1/transmissions", guard(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Options    map[string]any `json:"options"`
			Recipients []struct {
				Address struct{ Email string } `json:"address"`
			} `json:"recipients"`
		}
		key := r.Header.Get("Idempotency-Key")
		if err := json.Unmarshal(body, &in); err != nil || len(in.Recipients) != 1 || !strings.HasPrefix(key, "rc-") {
			writeErr(w, http.StatusUnprocessableEntity, "1300", "invalid data format/type")
			return
		}
		f.mu.Lock()
		limit, busy := f.limitHit, f.inProgress
		first, seen := f.keys[key]
		f.mu.Unlock()
		switch {
		case limit:
			writeErr(w, 420, "2102", "Exceed Sending Limit (daily)")
			return
		case busy:
			writeErr(w, http.StatusConflict, "1601", "A request with this idempotency key is in progress")
			return
		case seen && bytes.Equal(first.body, body):
			// The same request again: the first one's answer, replayed.
			w.Header().Set("X-Idempotent-Replayed", "true")
			_, _ = io.WriteString(w, first.answer)
			return
		case seen:
			writeErr(w, http.StatusConflict, "1600", "This idempotency key was already used with a different request")
			return
		}
		id := strconv.FormatInt(11668787484950529+f.n.Add(1), 10)
		answer := `{"results":{"total_rejected_recipients":0,"total_accepted_recipients":1,"id":"` + id + `"}}`
		if in.Recipients[0].Address.Email == contracttest.RejectAddress {
			answer = `{"errors":[{"message":"transmission created, but with validation errors","code":"2000"}],"results":{"rcpt_to_errors":[{"message":"invalid data format/type","description":"Invalid recipient address.email: ` +
				contracttest.RejectAddress + `","code":"1300"}],"total_rejected_recipients":1,"total_accepted_recipients":0,"id":"` + id + `"}}`
		}
		f.with(func() {
			if f.keys == nil {
				f.keys = map[string]keyedRequest{}
			}
			f.keys[key] = keyedRequest{body: body, answer: answer}
		})
		_, _ = io.WriteString(w, answer)
	}))
	mux.HandleFunc("GET /api/v1/sending-ips", guard(func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.limitHit {
			writeErr(w, 420, "2102", "Exceed Sending Limit (hourly)")
			return
		}
		if f.sendingIPs == "" {
			_, _ = io.WriteString(w, sendingIPs)
			return
		}
		_, _ = io.WriteString(w, f.sendingIPs)
	}))
	mux.HandleFunc("GET /api/v1/account", guard(func(w http.ResponseWriter, r *http.Request) {
		usage := ""
		if r.URL.Query().Get("include") == "usage" {
			usage = `,"usage":{"timestamp":"2026-10-13T05:19:00.932Z","day":{"used":2203,"limit":50000},"month":{"used":122596,"limit":1500000},"sandbox":{"used":3,"limit":5}}`
		}
		_, _ = io.WriteString(w, `{"results":{"customer_id":123456,"status":"active","subscription":{"name":"Premier 1.5M"},"options":{"rest_tracking_default":true,"click_tracking":true}`+usage+`}}`)
	}))
	mux.HandleFunc("PUT /api/v1/account", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.with(func() { f.accountPut = in })
		_, _ = io.WriteString(w, `{"results":{"message":"Successfully updated account."}}`)
	}))
	mux.HandleFunc("GET /api/v1/metrics/deliverability", guard(func(w http.ResponseWriter, r *http.Request) {
		if _, err := time.Parse("2006-01-02T15:04", r.URL.Query().Get("from")); err != nil {
			writeErr(w, http.StatusBadRequest, "1300", "from must be YYYY-MM-DDTHH:MM")
			return
		}
		_, _ = io.WriteString(w, `{"results":[{"count_targeted":10000,"count_accepted":9500,"count_bounce":150,"count_spam_complaint":5}]}`)
	}))
	domainJSON := func(dkim, cname string) string {
		return `{"results":{"tracking_domain":"click.acme.io","status":{"ownership_verified":true,"spf_status":"unverified","dkim_status":"` + dkim + `","cname_status":"` + cname +
			`","mx_status":"pending","compliance_status":"valid"},"dkim":{"headers":"from:to:subject:date","public":"MIGfMA0GCSq","selector":"scph1012"},"shared_with_subaccounts":false}}`
	}
	mux.HandleFunc("POST /api/v1/sending-domains", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in["domain"] == "mail.acme.io" {
			writeErr(w, http.StatusConflict, "1602", "resource conflict")
			return
		}
		_, _ = io.WriteString(w, `{"results":{"message":"Successfully Created domain.","domain":"`+in["domain"].(string)+`","dkim":{"public":"MIGfMA0GCSq","selector":"scph1012","signing_domain":"`+in["domain"].(string)+`","headers":"from:to:subject:date"}}}`)
	}))
	mux.HandleFunc("GET /api/v1/sending-domains/{domain}", guard(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, domainJSON("unverified", "invalid"))
	}))
	mux.HandleFunc("POST /api/v1/sending-domains/{domain}/verify", guard(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"results":{"ownership_verified":true,"dkim_status":"valid","cname_status":"invalid","dns":{"dkim_error":"","cname_error":"CNAME not found"}}}`)
	}))
	mux.HandleFunc("GET /api/v1/webhooks", guard(func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"results": slices.Concat(f.hooks, f.stale)})
	}))
	mux.HandleFunc("POST /api/v1/webhooks", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		// A counter, not len(f.hooks): ids must not repeat after a delete.
		f.hookN++
		in["id"] = "12affc24-" + strconv.Itoa(f.hookN)
		f.hooks = append(f.hooks, in)
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"results": map[string]any{"id": in["id"]}})
	}))
	mux.HandleFunc("PUT /api/v1/webhooks/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		for _, h := range f.hooks {
			if h["id"] == r.PathValue("id") {
				for k, v := range in {
					h[k] = v
				}
			}
		}
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"results": map[string]any{"id": r.PathValue("id")}})
	}))
	mux.HandleFunc("DELETE /api/v1/webhooks/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		n := len(f.hooks)
		f.hooks = slices.DeleteFunc(f.hooks, func(h map[string]any) bool { return h["id"] == r.PathValue("id") })
		if len(f.hooks) == n {
			writeErr(w, http.StatusNotFound, "1600", "resource not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.HandleFunc("GET /api/v1/webhooks/{id}/batch-status", guard(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"results":[{"batch_id":"032d330540298f54f0e8bcc1373f3cfd","ts":"2026-10-13T01:38:08.000Z","attempts":7,"response_code":"200","latency":160},{"batch_id":"13c6764994a8f6b4e29906d5712ca7d","ts":"2026-10-13T00:38:08.000Z","attempts":2,"failure_code":"400","response_code":"400","latency":210}]}`)
	}))
	mux.HandleFunc("POST /api/v1/webhooks/{id}/validate", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Message map[string]any `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Message == nil {
			writeErr(w, http.StatusBadRequest, "1400", "message is required")
			return
		}
		_, _ = io.WriteString(w, `{"results":{"msg":"Test POST to endpoint succeeded","response":{"status":200,"headers":{"Content-Type":"text/plain"},"body":"OK"}}}`)
	}))
	mux.HandleFunc("GET /api/v1/suppression-list", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "initial" {
			_, _ = io.WriteString(w, `{"results":[{"recipient":"Hard@Example.com","source":"Bounce Rule","type":"transactional","created":"2026-10-11T01:01:01+00:00"},{"recipient":"hard@example.com","source":"Bounce Rule","type":"non_transactional","created":"2026-10-11T01:01:01+00:00"},{"recipient":"angry@example.com","source":"Spam Complaint","type":"non_transactional","created":"2026-10-12T01:01:01+00:00"}],"links":[{"href":"/api/v1/suppression-list?cursor=WycyMDE4","rel":"next"}],"total_count":5}`)
			return
		}
		_, _ = io.WriteString(w, `{"results":[{"recipient":"gone@example.com","source":"List Unsubscribe","type":"non_transactional","created":"2026-10-12T02:01:01+00:00"},{"recipient":"no-tracking@example.com","source":"Tracking Consent Revoked","type":"open_tracking","created":"2026-10-12T02:30:01+00:00"},{"recipient":"admin@example.com","source":"Manually Added","type":"transactional","created":"2026-10-12T03:01:01+00:00"}],"links":[],"total_count":6}`)
	}))
	mux.HandleFunc("PUT /api/v1/suppression-list/", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Recipients []map[string]string `json:"recipients"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.with(func() { f.suppressions = append(f.suppressions, in.Recipients) })
		_, _ = io.WriteString(w, `{"results":{"message":"Suppression List successfully updated"}}`)
	}))
	mux.HandleFunc("GET /api/v1/events/message", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("transmissions") == "" {
			writeErr(w, http.StatusBadRequest, "1400", "transmissions required")
			return
		}
		_, _ = io.WriteString(w, `{"results":[
			{"type":"delay","bounce_class":"21","raw_reason":"421 4.4.0 DNS lookup failed","transmission_id":"t-1","rcpt_to":"a@example.com","timestamp":"2026-10-12T13:00:00.000+00:00"},
			{"type":"delivery","transmission_id":"t-1","rcpt_to":"a@example.com","timestamp":"2026-10-12T13:05:00.000+00:00","rcpt_meta":{"rc_message":"0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"}},
			{"type":"bounce","bounce_class":"10","raw_reason":"550 5.1.1 unknown user","transmission_id":"t-2","rcpt_to":"b@example.com","timestamp":"2026-10-12T13:01:00.000+00:00"}
		],"total_count":3,"links":{}}`)
	}))
	return mux
}

func newProvider(t *testing.T, env *contracttest.Env) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(sparkpost.Settings{Region: "eu", IPPool: "marketing", BaseURL: env.URL})
	p, err := sparkpost.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{"api_key":"` + apiKey + `"}`), WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func batchRequest(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
	body, err := os.ReadFile("testdata/batch.json")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/sparkpost/token", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-MessageSystems-Batch-ID", "032d330540298f54f0e8bcc1373f3cfd")
	r.SetBasicAuth(sparkpost.WebhookUser, secret)
	return r, body
}

func TestContract(t *testing.T) {
	f := &fake{}
	contracttest.Run(t, contracttest.Suite{
		Adapter:    sparkpost.Adapter,
		API:        f.handler,
		New:        newProvider,
		SendingIPs: &providers.SendingIPs{Dedicated: true, IPs: []string{"198.51.100.30", "198.51.100.31"}},
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || reqs[0].Path != "/api/v1/transmissions" || reqs[0].Header.Get("Idempotency-Key") != "rc-"+msgs[0].ID.String() {
				t.Fatalf("want one idempotent transmission, got %+v", reqs)
			}
			var sent struct {
				Options    map[string]any `json:"options"`
				Recipients []struct {
					Address  struct{ Email, Name string } `json:"address"`
					Metadata map[string]string            `json:"metadata"`
				} `json:"recipients"`
				Content struct {
					From    struct{ Email, Name string } `json:"from"`
					Headers map[string]string            `json:"headers"`
				} `json:"content"`
			}
			if err := json.Unmarshal(reqs[0].Body, &sent); err != nil {
				t.Fatal(err)
			}
			if sent.Options["open_tracking"] != false || sent.Options["click_tracking"] != false || sent.Options["transactional"] != true || sent.Options["ip_pool"] != "marketing" {
				t.Fatalf("options: %v", sent.Options)
			}
			if sent.Recipients[0].Metadata[sparkpost.MetaMessage] != msgs[0].ID.String() || sent.Recipients[0].Metadata["campaign"] != "contract" {
				t.Fatalf("recipient metadata: %v", sent.Recipients[0].Metadata)
			}
			if sent.Content.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" || sent.Content.Headers["X-ReputeCast-Message"] != msgs[0].ID.String() {
				t.Fatalf("headers: %v", sent.Content.Headers)
			}
			if !strings.HasPrefix(res[0].ProviderMessageID, "1166878748495") {
				t.Fatalf("provider id is the transmission id: %q", res[0].ProviderMessageID)
			}
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			r.SetBasicAuth(sparkpost.WebhookUser, "wrong-secret")
			r.Body = io.NopCloser(bytes.NewReader(body))
			return r, body
		},
		IdempotencyKey: func(r contracttest.Recorded) string { return r.Header.Get("Idempotency-Key") },
		Webhooks: []contracttest.Webhook{{
			Name: "Batch", Request: batchRequest,
			Want: []providers.Event{
				{Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "65832150921904130",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "delivery"},
				{Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "65832150921904131",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "bounce · class 10 Invalid Recipient", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown"},
				{Type: providers.EventSoftBounce, Recipient: "k.n@hotmail.com", ProviderMessageID: "65832150921904132",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "bounce · class 51 Spam Block", OriginalCode: "554 5.7.1", BounceClass: "spam_block"},
				{Type: providers.EventDeferred, Recipient: "b.h@yahoo.com", ProviderMessageID: "65832150921904133",
					OriginalType: "delay · class 22 Mailbox Full", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full"},
				{Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "65832150921904135",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e43"), OriginalType: "spam_complaint · abuse"},
				{Type: providers.EventRejected, Recipient: "old@example.org", ProviderMessageID: "65832150921904136",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e44"), OriginalType: "policy_rejection", OriginalCode: "554 5.7.1", BounceClass: "policy_block"},
				{Type: providers.EventUnsubscribe, Recipient: "gone@example.com", ProviderMessageID: "65832150921904137",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e45"), OriginalType: "list_unsubscribe"},
			},
		}},
	})
}

func TestTestBatchHasNoEvents(t *testing.T) {
	env := contracttest.NewEnv(t, nil)
	env.URL = "https://api.sparkpost.test"
	p := newProvider(t, env)
	for _, body := range []string{`[]`, `[{"msys":{}}]`} {
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/sparkpost/token", strings.NewReader(body))
		r.SetBasicAuth(sparkpost.WebhookUser, secret)
		evs, err := p.ParseWebhook(context.Background(), r, []byte(body))
		if err != nil || len(evs) != 0 {
			t.Fatalf("%s: %v, %v", body, evs, err)
		}
	}
}

// SparkPost keeps a transmission's Idempotency-Key for 24 hours (review
// GS-1): sending a message again is answered with the first transmission,
// a message rendered again since is refused as a key already used (409,
// 1600), which means an earlier attempt handed it over, and a key whose
// first request still runs is answered later (409, 1601).
func TestRetriedSendIsNotSentTwice(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	if w := providers.IdempotencyWindow(p); w != 24*time.Hour {
		t.Fatalf("idempotency window %v, want SparkPost's 24 hours", w)
	}
	ctx := context.Background()
	m := contracttest.Message(0)
	first, err := p.Send(ctx, []providers.Message{m})
	if err != nil || first[0].Err != nil {
		t.Fatal(first, err)
	}
	again, err := p.Send(ctx, []providers.Message{m})
	if err != nil || again[0].Err != nil || again[0].ProviderMessageID != first[0].ProviderMessageID {
		t.Fatalf("the same message again: %+v, %v; want the first transmission replayed", again, err)
	}
	if n := f.n.Load(); n != 1 {
		t.Fatalf("%d transmissions for one message", n)
	}
	rendered := m
	rendered.HTML = "<p>Rendered again since the first try.</p>"
	res, err := p.Send(ctx, []providers.Message{rendered})
	if err != nil || res[0].Err != nil || res[0].ProviderMessageID != m.ID.String() {
		t.Fatalf("a key already used: %+v, %v; want it counted as handed over", res, err)
	}
	if n := f.n.Load(); n != 1 {
		t.Fatalf("%d transmissions for one message", n)
	}
	f.with(func() { f.inProgress = true })
	_, err = p.Send(ctx, []providers.Message{contracttest.Message(1)})
	var ce *providers.CallError
	if !errors.As(err, &ce) || !ce.Temporary || ce.Throttled || ce.Code != "1601" {
		t.Fatalf("a key whose first request still runs: %v; want a temporary error", err)
	}
}

func TestSendingLimitIsThrottling(t *testing.T) {
	f := &fake{limitHit: true}
	env := contracttest.NewEnv(t, f.handler)
	_, err := newProvider(t, env).Send(context.Background(), []providers.Message{contracttest.Message(0)})
	var ce *providers.CallError
	if !errors.As(err, &ce) || !ce.Throttled || ce.Code != "2102" {
		t.Fatalf("420 is SparkPost's sending limit: %v", err)
	}
}

func TestQuotaAndReputation(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	q, err := p.(providers.QuotaReporter).Quota(context.Background())
	if err != nil || q.Kind != "monthly_plan" || q.Max != 1500000 || q.Used != 122596 || q.Account != "Account 123456 · active · Premier 1.5M" {
		t.Fatalf("quota: %+v, %v", q, err)
	}
	rep, err := p.(providers.ReputationReporter).Reputation(context.Background())
	if err != nil || rep.Bounce != 1.5 || rep.Complaint < 0.0526 || rep.Complaint > 0.0527 {
		t.Fatalf("reputation: %+v, %v", rep, err)
	}
	env.SetMode(contracttest.ModeAuth)
	if _, err := p.(providers.QuotaReporter).Quota(context.Background()); !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("rejected key: %v", err)
	}
}

func TestDomains(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env).(providers.DomainManager)
	recs, err := p.AddDomain(context.Background(), "news.acme.io")
	if err != nil {
		t.Fatal(err)
	}
	want := []providers.DNSRecord{
		{Purpose: "dkim", Type: "TXT", Name: "scph1012._domainkey.news.acme.io", Value: "v=DKIM1; k=rsa; h=sha256; p=MIGfMA0GCSq", Status: "pending"},
		{Purpose: "return_path", Type: "CNAME", Name: "news.acme.io", Value: "eu.sparkpostmail.com", Status: "pending"},
	}
	if len(recs) != 2 || recs[0] != want[0] || recs[1] != want[1] {
		t.Fatalf("records: %+v", recs)
	}
	// An existing domain is read and verified again.
	recs, err = p.AddDomain(context.Background(), "mail.acme.io")
	if err != nil || recs[0].Status != "verified" || recs[1].Status != "wrong_value" {
		t.Fatalf("existing: %+v, %v", recs, err)
	}
	// A zone apex can't take the bounce CNAME: DKIM only.
	for _, apex := range []string{"acme.io", "acme.co.uk"} {
		recs, err = p.AddDomain(context.Background(), apex)
		if err != nil || len(recs) != 1 || recs[0].Purpose != "dkim" {
			t.Fatalf("apex %s: %+v, %v", apex, recs, err)
		}
	}
}

func TestWebhooks(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	const hook = "https://reputecast.acme.io/hooks/sparkpost/tok-1"
	for range 2 {
		if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), hook); err != nil {
			t.Fatal(err)
		}
	}
	f.with(func() {
		if len(f.hooks) != 1 {
			t.Fatalf("registered twice: %v", f.hooks)
		}
		h := f.hooks[0]
		creds := h["auth_credentials"].(map[string]any)
		if h["auth_type"] != "basic" || creds["username"] != sparkpost.WebhookUser || creds["password"] != secret || len(h["events"].([]any)) != 10 {
			t.Fatalf("webhook: %v", h)
		}
	})
	st, err := p.(providers.WebhookInspector).InspectWebhook(context.Background(), hook+"/")
	if err != nil || !st.Registered || st.Queued != 1 || st.ID != "12affc24-1" {
		t.Fatalf("inspect: %+v, %v", st, err)
	}
	if err := p.(providers.WebhookTester).SendTestEvent(context.Background(), hook); err != nil {
		t.Fatal(err)
	}
	if err := p.(providers.WebhookTester).SendTestEvent(context.Background(), "https://elsewhere.example/hook"); err == nil {
		t.Fatal("testing an unregistered webhook is an error")
	}
}

func TestUnregisterWebhook(t *testing.T) {
	const hook = "https://reputecast.acme.io/hooks/sparkpost/tok-1"
	f := &fake{
		hooks: []map[string]any{
			{"id": "mine", "target": hook},
			{"id": "theirs", "target": "https://reputecast.acme.io/hooks/sparkpost/tok-10"},
			{"id": "mine-again", "target": hook + "/"},
		},
		stale: []map[string]any{{"id": "gone", "target": hook}},
	}
	env := contracttest.NewEnv(t, f.handler)
	if err := newProvider(t, env).(providers.WebhookUnregistrar).UnregisterWebhook(context.Background(), hook); err != nil {
		t.Fatalf("a webhook already gone (404) is removed all the same: %v", err)
	}
	deleted := []string{}
	for _, r := range env.Requests() {
		if r.Method == http.MethodDelete {
			deleted = append(deleted, r.Path)
		}
	}
	if strings.Join(deleted, ",") != "/api/v1/webhooks/mine,/api/v1/webhooks/mine-again,/api/v1/webhooks/gone" {
		t.Fatalf("every webhook pointing at the URL is deleted: %v", deleted)
	}
	f.with(func() {
		if len(f.hooks) != 1 || f.hooks[0]["id"] != "theirs" {
			t.Fatalf("a webhook pointing elsewhere is kept: %v", f.hooks)
		}
	})
}

func TestSuppressionsTrackingBackfill(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	sync := p.(providers.SuppressionSyncer)
	list, err := sync.ListSuppressions(context.Background(), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, s := range list {
		got = append(got, s.Email+"/"+s.Reason)
	}
	if strings.Join(got, ",") != "hard@example.com/hard_bounce,angry@example.com/complaint,gone@example.com/unsubscribe,admin@example.com/manual" {
		t.Fatalf("suppressions: %v", got)
	}
	if !strings.Contains(env.Requests()[0].Path, "from=2026-10-01T00%3A00%3A00Z") {
		t.Fatalf("only changes since the last sync: %s", env.Requests()[0].Path)
	}
	if err := sync.AddSuppressions(context.Background(), []string{"a@example.com", "b@example.com"}); err != nil {
		t.Fatal(err)
	}
	f.with(func() {
		if len(f.suppressions) != 1 || len(f.suppressions[0]) != 4 || f.suppressions[0][1]["type"] != "non_transactional" {
			t.Fatalf("both lists: %v", f.suppressions)
		}
	})
	tm := p.(providers.TrackingManager)
	tr, err := tm.Tracking(context.Background())
	if err != nil || tr.OpenPixel || tr.LinkRewrite || !strings.HasPrefix(tr.Detail, "The account tracks opens and clicks by default") {
		t.Fatalf("tracking: %+v, %v", tr, err)
	}
	if err := tm.DisableTracking(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.with(func() {
		opts := f.accountPut["options"].(map[string]any)
		if opts["rest_tracking_default"] != false || opts["click_tracking"] != false {
			t.Fatalf("disable: %v", f.accountPut)
		}
	})
	evs, err := p.(providers.EventBackfiller).Backfill(context.Background(), []string{"t-1", "t-2", "t-3"})
	if err != nil || len(evs) != 2 || evs[0].Type != providers.EventDelivered || evs[0].MessageID == nil || evs[1].Type != providers.EventHardBounce {
		t.Fatalf("backfill: %+v, %v", evs, err)
	}
}

// Transmissions go out through the connection's IP pool, or the default
// pool without one: only that pool's IPs count. An account without
// dedicated IPs lists none; throttling (420) is an error, never an empty
// list.
func TestSendingIPs(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	read := func(p providers.Provider) (providers.SendingIPs, error) {
		return p.(providers.IPReporter).SendingIPs(context.Background())
	}
	if got, err := read(newProvider(t, env)); err != nil || !got.Dedicated || !slices.Equal(got.IPs, []string{"198.51.100.30", "198.51.100.31"}) {
		t.Fatalf("the marketing pool: %+v, %v", got, err)
	}
	settings, _ := json.Marshal(sparkpost.Settings{Region: "eu", BaseURL: env.URL})
	byDefault, err := sparkpost.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{"api_key":"` + apiKey + `"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := read(byDefault); err != nil || !slices.Equal(got.IPs, []string{"198.51.100.4"}) {
		t.Fatalf("the default pool: %+v, %v", got, err)
	}
	f.with(func() { f.sendingIPs = `{"results":[]}` })
	if got, err := read(byDefault); err != nil || got.Dedicated || got.IPs == nil || len(got.IPs) != 0 {
		t.Fatalf("shared IPs: %+v, %v", got, err)
	}
	f.with(func() { f.limitHit = true })
	if got, err := read(byDefault); err == nil || len(got.IPs) != 0 {
		t.Fatalf("throttled: %+v, %v", got, err)
	}
}
