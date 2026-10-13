package mailgun_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/mailgun"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

const (
	apiKey     = "key-test-private"
	signingKey = "key-55c5c5c5c55f55ca5cd5f55d5c555c55"
	domain     = "mg.acme.io"
)

// fake is a fake Mailgun API for one domain.
type fake struct {
	mu          sync.Mutex
	customLimit bool
	domainState string
	hooks       map[string][]string
	unsubAdds   int
	tracking    map[string]string
	n           atomic.Int64
}

func (f *fake) with(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

func newFake() *fake {
	return &fake{domainState: "active", hooks: map[string][]string{"delivered": {"https://other.example/hook"}},
		tracking: map[string]string{"open": "true", "click": `"htmlonly"`, "unsubscribe": "true"}}
}

func message(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

func formValues(t *testing.T, r *http.Request) map[string][]string {
	t.Helper()
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	form, err := mr.ReadForm(10 << 20)
	if err != nil {
		return nil
	}
	return form.Value
}

func (f *fake) handler(t *testing.T) func(env *contracttest.Env) http.Handler {
	return func(env *contracttest.Env) http.Handler {
		mux := http.NewServeMux()
		guard := func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				switch {
				case env.Mode() == contracttest.ModeOutage:
					message(w, http.StatusInternalServerError, "Internal Server Error")
					return
				case env.Mode() == contracttest.ModeAuth || !ok || user != "api" || pass != apiKey:
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = io.WriteString(w, "Forbidden")
					return
				}
				next(w, r)
			}
		}
		mux.HandleFunc("POST /v3/"+domain+"/messages", guard(func(w http.ResponseWriter, r *http.Request) {
			v := formValues(t, r)
			if v == nil || len(v["to"]) != 1 || v["subject"] == nil {
				message(w, http.StatusBadRequest, "from parameter is missing")
				return
			}
			if strings.Contains(v["to"][0], contracttest.RejectAddress) {
				message(w, http.StatusBadRequest, "'to' parameter is not a valid address. please check documentation")
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "<20261012134102." + strconv.FormatInt(f.n.Add(1), 10) + "@" + domain + ">", "message": "Queued. Thank you."})
		}))
		mux.HandleFunc("POST /v3/unknown.acme.io/messages", guard(func(w http.ResponseWriter, _ *http.Request) {
			message(w, http.StatusNotFound, "Domain not found: unknown.acme.io")
		}))
		domainJSON := func(name, state string) map[string]any {
			return map[string]any{
				"domain": map[string]any{"name": name, "state": state, "type": map[bool]string{true: "sandbox", false: "custom"}[strings.HasPrefix(name, "sandbox")]},
				"sending_dns_records": []map[string]any{
					{"is_active": true, "name": name, "record_type": "TXT", "valid": "valid", "value": "v=spf1 include:mailgun.org ~all"},
					{"is_active": true, "name": "pdk1._domainkey." + name, "record_type": "CNAME", "valid": map[bool]string{true: "valid", false: "unknown"}[state == "active"], "value": "pdk1._domainkey.0a1b2c.dkim1.mgsend.net"},
					{"is_active": true, "name": "email." + name, "record_type": "CNAME", "valid": "valid", "value": "mailgun.org"},
				},
				"receiving_dns_records": []map[string]any{
					{"is_active": true, "priority": "10", "record_type": "MX", "valid": "invalid", "value": "mxa.mailgun.org"},
					{"is_active": true, "priority": "10", "record_type": "MX", "valid": "valid", "value": "mxb.mailgun.org"},
				},
			}
		}
		mux.HandleFunc("GET /v4/domains/{name}", guard(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.PathValue("name"), "unknown.") {
				message(w, http.StatusNotFound, "Domain not found")
				return
			}
			f.mu.Lock()
			state := f.domainState
			f.mu.Unlock()
			_ = json.NewEncoder(w).Encode(domainJSON(r.PathValue("name"), state))
		}))
		mux.HandleFunc("PUT /v4/domains/{name}/verify", guard(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(domainJSON(r.PathValue("name"), "active"))
		}))
		mux.HandleFunc("POST /v4/domains", guard(func(w http.ResponseWriter, r *http.Request) {
			v := formValues(t, r)
			if v["use_automatic_sender_security"][0] != "true" {
				message(w, http.StatusBadRequest, "expected automatic sender security")
				return
			}
			if v["name"][0] == domain {
				message(w, http.StatusBadRequest, "This domain name is already taken")
				return
			}
			_ = json.NewEncoder(w).Encode(domainJSON(v["name"][0], "unverified"))
		}))
		mux.HandleFunc("GET /v5/accounts/limit/custom/monthly", guard(func(w http.ResponseWriter, _ *http.Request) {
			f.mu.Lock()
			custom := f.customLimit
			f.mu.Unlock()
			if !custom {
				message(w, http.StatusNotFound, "No threshold for account")
				return
			}
			_, _ = io.WriteString(w, `{"limit": 100000, "current": 4210, "period": "1m"}`)
		}))
		mux.HandleFunc("POST /v1/analytics/metrics", guard(func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Resolution string   `json:"resolution"`
				Metrics    []string `json:"metrics"`
				Start      string   `json:"start"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			if _, err := time.Parse(time.RFC1123Z, in.Start); err != nil {
				message(w, http.StatusBadRequest, "start must be RFC 2822")
				return
			}
			if in.Resolution == "month" {
				_, _ = io.WriteString(w, `{"items":[],"aggregates":{"metrics":{"processed_count":8765}}}`)
				return
			}
			_, _ = io.WriteString(w, `{"items":[],"aggregates":{"metrics":{"processed_count":4000,"delivered_count":3900,"bounced_count":80,"complained_count":2}}}`)
		}))
		mux.HandleFunc("GET /v5/accounts/http_signing_key", guard(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"http_signing_key": signingKey})
		}))
		mux.HandleFunc("GET /v3/domains/"+domain+"/webhooks", guard(func(w http.ResponseWriter, _ *http.Request) {
			f.mu.Lock()
			defer f.mu.Unlock()
			out := map[string]any{}
			for id, urls := range f.hooks {
				out[id] = map[string]any{"urls": urls}
			}
			out["opened"] = nil
			_ = json.NewEncoder(w).Encode(map[string]any{"webhooks": out})
		}))
		mux.HandleFunc("POST /v3/domains/"+domain+"/webhooks", guard(func(w http.ResponseWriter, r *http.Request) {
			v := formValues(t, r)
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.hooks[v["id"][0]]) > 0 {
				message(w, http.StatusBadRequest, "Webhook already exists")
				return
			}
			f.hooks[v["id"][0]] = v["url"]
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "Webhook has been created", "webhook": map[string]any{"urls": v["url"]}})
		}))
		mux.HandleFunc("PUT /v3/domains/"+domain+"/webhooks/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
			v := formValues(t, r)
			f.mu.Lock()
			defer f.mu.Unlock()
			f.hooks[r.PathValue("id")] = v["url"]
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "Webhook has been updated", "webhook": map[string]any{"urls": v["url"]}})
		}))
		mux.HandleFunc("GET /v3/"+domain+"/{table}", guard(func(w http.ResponseWriter, r *http.Request) {
			base := "http://" + r.Host + "/v3/" + domain + "/" + r.PathValue("table")
			switch r.PathValue("table") {
			case "bounces":
				if r.URL.Query().Get("page") == "" {
					_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{
						{"address": "Hard@Example.com", "code": "550", "error": "No such mailbox", "created_at": "Sun, 11 Oct 2026 01:49:40 UTC"},
						{"address": "ancient@example.com", "code": "550", "error": "No such mailbox", "created_at": "Thu, 11 Dec 2025 01:49:40 UTC"},
					}, "paging": map[string]string{"next": base + "?page=next&address=hard%40example.com&limit=1000"}})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "paging": map[string]string{"next": base + "?page=next&address=zzz"}})
			case "complaints":
				_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"address": "angry@example.com", "created_at": "Mon, 12 Oct 2026 10:00:00 UTC"}}, "paging": map[string]string{}})
			case "unsubscribes":
				_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{
					{"address": "gone@example.com", "tags": []string{"*"}, "created_at": "Mon, 12 Oct 2026 10:00:00 UTC"},
					{"address": "picky@example.com", "tags": []string{"promotions"}, "created_at": "Mon, 12 Oct 2026 10:00:00 UTC"},
				}, "paging": map[string]string{}})
			case "events":
				id := r.URL.Query().Get("message-id")
				events := map[string]string{
					"m-delivered": `{"items":[{"event":"opened","timestamp":1760276600.5,"recipient":"a@example.com","message":{"headers":{"message-id":"m-delivered"}},"user-variables":{}},{"event":"delivered","timestamp":1760276462.6,"recipient":"a@example.com","delivery-status":{"code":250,"message":"OK"},"message":{"headers":{"message-id":"m-delivered"}},"user-variables":{}}]}`,
					"m-bounced":   `{"items":[{"event":"failed","severity":"permanent","reason":"bounce","timestamp":1760276462.6,"recipient":"b@example.com","delivery-status":{"code":550,"message":"5.1.1 User unknown","enhanced-code":"5.1.1"},"message":{"headers":{"message-id":"m-bounced"}},"user-variables":{}}]}`,
					"m-queued":    `{"items":[{"event":"accepted","timestamp":1760276462.6,"recipient":"c@example.com","message":{"headers":{"message-id":"m-queued"}}}]}`,
				}[id]
				if events == "" {
					events = `{"items":[]}`
				}
				_, _ = io.WriteString(w, events)
			default:
				message(w, http.StatusNotFound, "not found")
			}
		}))
		mux.HandleFunc("POST /v3/"+domain+"/unsubscribes", guard(func(w http.ResponseWriter, r *http.Request) {
			var in []struct {
				Address string   `json:"address"`
				Tags    []string `json:"tags"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in) > 1000 || in[0].Tags[0] != "*" {
				message(w, http.StatusBadRequest, "Batch size should be less than 1000")
				return
			}
			f.mu.Lock()
			f.unsubAdds++
			f.mu.Unlock()
			message(w, http.StatusOK, strconv.Itoa(len(in))+" addresses have been added to the unsubscribes table")
		}))
		mux.HandleFunc("GET /v3/domains/"+domain+"/tracking", guard(func(w http.ResponseWriter, _ *http.Request) {
			f.mu.Lock()
			defer f.mu.Unlock()
			_, _ = io.WriteString(w, `{"tracking":{"open":{"active":`+f.tracking["open"]+`},"click":{"active":`+f.tracking["click"]+`},"unsubscribe":{"active":`+f.tracking["unsubscribe"]+`,"html_footer":"on","text_footer":"on"},"web_scheme":"https"}}`)
		}))
		mux.HandleFunc("PUT /v3/domains/"+domain+"/tracking/{kind}", guard(func(w http.ResponseWriter, r *http.Request) {
			v := formValues(t, r)
			f.mu.Lock()
			f.tracking[r.PathValue("kind")] = v["active"][0]
			f.mu.Unlock()
			message(w, http.StatusOK, "Domain tracking settings have been updated")
		}))
		return mux
	}
}

func newProvider(t *testing.T, env *contracttest.Env, withKey bool, dom string) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(mailgun.Settings{Domain: dom, Region: "eu", BaseURL: env.URL, MonthlyLimit: 50000})
	creds := map[string]string{"api_key": apiKey}
	if withKey {
		creds["webhook_signing_key"] = signingKey
	}
	c, _ := json.Marshal(creds)
	p, err := mailgun.New(providers.Config{Settings: settings, Credentials: c})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var sigRe = regexp.MustCompile(`"token": "([0-9a-f]+)", "timestamp": "([0-9]+)", "signature": "__SIGNATURE__"`)

func signed(t *testing.T, file string) (*http.Request, []byte) {
	raw, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	m := sigRe.FindSubmatch(raw)
	if m == nil {
		t.Fatalf("%s has no signature placeholder", file)
	}
	// Signed now, as Mailgun signs a delivery when it sends it.
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	block := fmt.Sprintf(`"token": "%s", "timestamp": "%s", "signature": "%s"`, m[1], ts, mailgun.Sign(signingKey, ts, string(m[1])))
	body := bytes.Replace(raw, m[0], []byte(block), 1)
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/mailgun/token", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r, body
}

func webhook(file string) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) { return signed(t, file) }
}

func TestContract(t *testing.T) {
	f := newFake()
	contracttest.Run(t, contracttest.Suite{
		Adapter: mailgun.Adapter,
		API:     f.handler(t),
		New:     func(t *testing.T, env *contracttest.Env) providers.Provider { return newProvider(t, env, true, domain) },
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || reqs[0].Path != "/v3/"+domain+"/messages" {
				t.Fatalf("want one POST /v3/{domain}/messages, got %+v", reqs)
			}
			r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", bytes.NewReader(reqs[0].Body))
			r.Header = reqs[0].Header
			v := formValues(t, r)
			m := msgs[0]
			if v["from"][0] != `"Acme Newsletter" <news@news.acme.io>` || v["v:rc_message"][0] != m.ID.String() || v["v:campaign"][0] != "contract" {
				t.Fatalf("form: %v", v)
			}
			if v["h:List-Unsubscribe-Post"][0] != "List-Unsubscribe=One-Click" || v["h:X-ReputeCast-Message"][0] != m.ID.String() {
				t.Fatalf("headers: %v", v)
			}
			if v["o:tracking"][0] != "no" || v["o:tracking-clicks"][0] != "no" || v["o:tracking-opens"][0] != "no" {
				t.Fatalf("Mailgun tracking must be off on every message: %v", v)
			}
			if !strings.HasPrefix(res[0].ProviderMessageID, "20261012134102.") || strings.ContainsAny(res[0].ProviderMessageID, "<>") {
				t.Fatalf("provider id without brackets: %q", res[0].ProviderMessageID)
			}
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			// Mailgun signs timestamp and token, not the body.
			bad := regexp.MustCompile(`"token": "[0-9a-f]`).ReplaceAll(body, []byte(`"token": "x`))
			r.Body = io.NopCloser(bytes.NewReader(bad))
			return r, bad
		},
		Webhooks: []contracttest.Webhook{
			{Name: "Delivered", Request: webhook("delivered.json"), Want: []providers.Event{{
				Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "20261012134102.58bbb7ce85a423e5@mg.acme.io",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "delivered", OriginalCode: "250 2.0.0",
			}}},
			{Name: "FailedPermanent", Request: webhook("failed-permanent.json"), Want: []providers.Event{{
				Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "20261012134102.6a1b2c3d4e5f6a7b@mg.acme.io",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "failed · permanent · bounce", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown",
			}}},
			{Name: "FailedTemporary", Request: webhook("failed-temporary.json"), Want: []providers.Event{{
				Type: providers.EventDeferred, Recipient: "p.s@yahoo.com", ProviderMessageID: "20261012134102.7b2c3d4e5f6a7b8c@mg.acme.io",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e43"), OriginalType: "failed · temporary · generic", OriginalCode: "421 4.7.0", BounceClass: "reputation",
			}}},
			{Name: "FailedBlock", Request: webhook("failed-block.json"), Want: []providers.Event{{
				Type: providers.EventSoftBounce, Recipient: "k.n@hotmail.com", ProviderMessageID: "20261012134102.8c3d4e5f6a7b8c9d@mg.acme.io",
				OriginalType: "failed · permanent · bounce", OriginalCode: "550 5.7.1", BounceClass: "spam_block",
			}}},
			{Name: "Suppressed", Request: webhook("suppressed.json"), Want: []providers.Event{{
				Type: providers.EventRejected, Recipient: "old@example.org", ProviderMessageID: "20261012134102.9d4e5f6a7b8c9d0e@mg.acme.io",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e44"), OriginalType: "failed · permanent · suppress-bounce", OriginalCode: "605", BounceClass: "mailbox_unknown",
			}}},
			{Name: "Complained", Request: webhook("complained.json"), Want: []providers.Event{{
				Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "20261012134102.ae5f6a7b8c9d0e1f@mg.acme.io",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "complained",
			}}},
		},
	})
}

func TestSigningKeyFromTheAPI(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, false, domain).(providers.WebhookVerifier)
	r, body := signed(t, "delivered.json")
	if err := p.VerifyWebhook(context.Background(), r, body); err != nil {
		t.Fatalf("the key is read from GET /v5/accounts/http_signing_key: %v", err)
	}
	if !strings.Contains(env.Requests()[0].Path, "/v5/accounts/http_signing_key") {
		t.Fatalf("requests: %+v", env.Requests())
	}
	if err := p.VerifyWebhook(context.Background(), r, []byte(`{"event-data":{}}`)); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("an unsigned request is refused: %v", err)
	}
}

func TestWebhookReplays(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, true, domain).(providers.WebhookVerifier)
	r, body := signed(t, "failed-permanent.json")
	if err := p.VerifyWebhook(context.Background(), r, body); err != nil {
		t.Fatal(err)
	}
	// A redelivery, serialized differently, is the same event.
	var v map[string]any
	_ = json.Unmarshal(body, &v)
	again, _ := json.MarshalIndent(v, "", "\t")
	if err := p.VerifyWebhook(context.Background(), r, again); err != nil {
		t.Fatalf("a redelivery of the same event passes: %v", err)
	}
	// The signature covers the token, not the event: a captured one can't
	// carry another event.
	forged := bytes.Replace(body, []byte("john@example.com"), []byte("ceo@acme.io"), 1)
	if bytes.Equal(forged, body) {
		t.Fatal("the fixture changed")
	}
	if err := p.VerifyWebhook(context.Background(), r, forged); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a reused token with another event is a forgery: %v", err)
	}
	// Retries come for eight hours; much older signatures are replays.
	ts := strconv.FormatInt(time.Now().Add(-mailgun.Window-time.Hour).Unix(), 10)
	const token = "f0e1d2c3b4a5968778695a4b3c2d1e0f0e1d2c3b4a59687786"
	old := fmt.Sprintf(`{"signature":{"token":%q,"timestamp":%q,"signature":%q},"event-data":{"event":"delivered"}}`, token, ts, mailgun.Sign(signingKey, ts, token))
	if err := p.VerifyWebhook(context.Background(), r, []byte(old)); !errors.Is(err, webhookauth.ErrStale) {
		t.Fatalf("a request signed long ago is a replay: %v", err)
	}
}

func TestSignMatchesMailgunsExample(t *testing.T) {
	// hex(HMAC-SHA256(key, timestamp+token)), the computation Mailgun documents.
	got := mailgun.Sign("key-secret", "1529006854", "a8ce0edb2dd8301dee6c2405235584e45aa91d1e9f979f3de0")
	if len(got) != 64 || strings.Trim(got, "0123456789abcdef") != "" {
		t.Fatalf("signature: %q", got)
	}
	if got == mailgun.Sign("key-secret", "1529006855", "a8ce0edb2dd8301dee6c2405235584e45aa91d1e9f979f3de0") {
		t.Fatal("the timestamp is signed")
	}
}

func TestSendErrors(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, true, "unknown.acme.io")
	_, err := p.Send(context.Background(), []providers.Message{contracttest.Message(0)})
	if !errors.Is(err, providers.ErrUnauthorized) || !strings.Contains(err.Error(), "doesn't know the domain unknown.acme.io") {
		t.Fatalf("an unknown domain is a settings problem: %v", err)
	}
}

func TestQuota(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, true, domain).(providers.QuotaReporter)
	q, err := p.Quota(context.Background())
	if err != nil || q.Kind != "monthly_plan" || q.Used != 8765 || q.Max != 50000 || q.Sandbox || q.Account != "Domain mg.acme.io · active" {
		t.Fatalf("plan quota: %+v, %v", q, err)
	}
	f.with(func() { f.customLimit = true })
	q, err = p.Quota(context.Background())
	if err != nil || q.Max != 100000 || q.Used != 4210 || !strings.Contains(q.Account, "custom limit 100,000 per month") {
		t.Fatalf("custom limit: %+v, %v", q, err)
	}
	sandbox := newProvider(t, env, true, "sandbox123.mailgun.org").(providers.QuotaReporter)
	if q, _ := sandbox.Quota(context.Background()); !q.Sandbox {
		t.Fatalf("a sandbox domain is detected: %+v", q)
	}
	env.SetMode(contracttest.ModeAuth)
	if _, err := p.Quota(context.Background()); !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("rejected key: %v", err)
	}
}

func TestDomains(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, true, domain).(providers.DomainManager)
	recs, err := p.AddDomain(context.Background(), "news.acme.io")
	if err != nil {
		t.Fatal(err)
	}
	want := []providers.DNSRecord{
		{Purpose: "spf", Type: "TXT", Name: "news.acme.io", Value: "v=spf1 include:mailgun.org ~all", Status: "verified"},
		{Purpose: "dkim", Type: "CNAME", Name: "pdk1._domainkey.news.acme.io", Value: "pdk1._domainkey.0a1b2c.dkim1.mgsend.net", Status: "pending"},
		{Purpose: "return_path", Type: "MX", Name: "news.acme.io", Value: "10 mxa.mailgun.org", Status: "wrong_value"},
		{Purpose: "return_path", Type: "MX", Name: "news.acme.io", Value: "10 mxb.mailgun.org", Status: "verified"},
	}
	if len(recs) != len(want) {
		t.Fatalf("records: %+v", recs)
	}
	for i := range want {
		if recs[i] != want[i] {
			t.Fatalf("record %d: %+v, want %+v", i, recs[i], want[i])
		}
	}
	// An existing domain is read; one not active yet is verified again.
	f.with(func() { f.domainState = "unverified" })
	recs, err = p.AddDomain(context.Background(), domain)
	if err != nil || recs[1].Status != "verified" {
		t.Fatalf("existing domain: %+v, %v", recs, err)
	}
	verified := false
	for _, r := range env.Requests() {
		verified = verified || r.Path == "/v4/domains/"+domain+"/verify"
	}
	if !verified {
		t.Fatal("an unverified domain is verified again")
	}
	// Another domain Mailgun doesn't know is not a settings problem.
	_, err = p.DomainStatus(context.Background(), "unknown.acme.io")
	if err == nil || errors.Is(err, providers.ErrUnauthorized) || !strings.Contains(err.Error(), "doesn't know the domain unknown.acme.io") {
		t.Fatalf("an unknown domain other than the sending one: %v", err)
	}
}

func TestWebhookRegistration(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, true, domain)
	const hook = "https://reputecast.acme.io/hooks/mailgun/tok-1"
	if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), hook); err != nil {
		t.Fatal(err)
	}
	f.with(func() {
		if got := f.hooks["delivered"]; len(got) != 2 || got[0] != "https://other.example/hook" || got[1] != hook {
			t.Fatalf("other URLs are kept: %v", got)
		}
		for _, id := range []string{"permanent_fail", "temporary_fail", "complained", "unsubscribed"} {
			if got := f.hooks[id]; len(got) != 1 || got[0] != hook {
				t.Fatalf("%s: %v", id, got)
			}
		}
		if _, ok := f.hooks["opened"]; ok {
			t.Fatal("opens are ReputeCast's")
		}
	})
	calls := len(env.Requests())
	if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), hook+"/"); err != nil || len(env.Requests()) != calls+1 {
		t.Fatalf("registering again changes nothing: %v", err)
	}
	st, err := p.(providers.WebhookInspector).InspectWebhook(context.Background(), hook)
	if err != nil || !st.Registered || st.ID != "delivered,permanent_fail,temporary_fail,complained,unsubscribed" {
		t.Fatalf("inspect: %+v, %v", st, err)
	}
	f.with(func() {
		f.hooks["permanent_fail"] = []string{"https://a.example", "https://b.example", "https://c.example"}
	})
	if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), "https://reputecast.acme.io/hooks/mailgun/tok-2"); err == nil || !strings.Contains(err.Error(), "three URLs") {
		t.Fatalf("Mailgun's three URLs per event: %v", err)
	}
}

func TestSuppressionSync(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, true, domain).(providers.SuppressionSyncer)
	list, err := p.ListSuppressions(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, s := range list {
		got = append(got, s.Email+"/"+s.Reason)
	}
	if strings.Join(got, ",") != "hard@example.com/hard_bounce,angry@example.com/complaint,gone@example.com/unsubscribe" {
		t.Fatalf("suppressions: %v", got)
	}
	emails := make([]string, 2500)
	for i := range emails {
		emails[i] = "x" + strconv.Itoa(i) + "@example.com"
	}
	if err := p.AddSuppressions(context.Background(), emails); err != nil {
		t.Fatal(err)
	}
	f.with(func() {
		if f.unsubAdds != 3 {
			t.Fatalf("chunks of 1,000: %d", f.unsubAdds)
		}
	})
}

func TestTrackingBackfillReputation(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, true, domain)
	tm := p.(providers.TrackingManager)
	tr, err := tm.Tracking(context.Background())
	if err != nil || tr.OpenPixel || !tr.LinkRewrite || !strings.HasPrefix(tr.Detail, "Unsubscribe tracking is on") {
		t.Fatalf("tracking: %+v, %v", tr, err)
	}
	if err := tm.DisableTracking(context.Background()); err != nil {
		t.Fatal(err)
	}
	if tr, _ := tm.Tracking(context.Background()); tr.LinkRewrite || tr.Detail != "Open, click and unsubscribe tracking are off for mg.acme.io." {
		t.Fatalf("after disable: %+v", tr)
	}
	evs, err := p.(providers.EventBackfiller).Backfill(context.Background(), []string{"<m-delivered>", "m-bounced", "m-queued", "m-gone"})
	if err != nil || len(evs) != 2 || evs[0].Type != providers.EventDelivered || evs[1].Type != providers.EventHardBounce || evs[1].OriginalCode != "550 5.1.1" {
		t.Fatalf("backfill: %+v, %v", evs, err)
	}
	rep, err := p.(providers.ReputationReporter).Reputation(context.Background())
	if err != nil || rep.Bounce != 2 || rep.Complaint < 0.0512 || rep.Complaint > 0.0513 {
		t.Fatalf("reputation: %+v, %v", rep, err)
	}
}

func TestBasicAuthHeader(t *testing.T) {
	f := newFake()
	env := contracttest.NewEnv(t, f.handler(t))
	p := newProvider(t, env, true, domain)
	if _, err := p.Send(context.Background(), []providers.Message{contracttest.Message(0)}); err != nil {
		t.Fatal(err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("api:"+apiKey))
	if got := env.Requests()[0].Header.Get("Authorization"); got != want {
		t.Fatalf("auth header: %q", got)
	}
}
