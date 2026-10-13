package httpgeneric_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/httpgeneric"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

const (
	apiKey     = "esp-test-key"
	signingKey = "esp-signing-secret"
)

// bodyTemplate is how an admin would map the example ESP's send call.
const bodyTemplate = `{
  "from": "{{ sender }}",
  "to": [{"email": "{{ recipient }}", "name": "{{ recipient_name }}"}],
  "reply_to": {{ reply_to | default: sender_email }},
  "subject": "{{ subject }}",
  "html": "{{ html }}",
  "text": "{{ text }}",
  "headers": {{ headers_json }},
  "metadata": {"reputecast_id": "{{ message_id }}", "stream": "{{ stream }}"},
  "tracking": {"opens": false, "clicks": false}
}`

func mapping(base string) httpgeneric.Mapping {
	return httpgeneric.Mapping{
		Method:           "POST",
		URL:              base + "/v1/{{ settings.account }}/messages?region={{ settings.region | default: 'eu' }}",
		Headers:          map[string]string{"Authorization": "Bearer {{ secrets.api_key }}", "X-Client": "ReputeCast"},
		Body:             bodyTemplate,
		MessageIDPath:    "$.data.id",
		ErrorCodePath:    "$.error.code",
		ErrorMessagePath: "$.error.message",
		WebhookMapping:   "jsonpath",
		Webhook: httpgeneric.WebhookSpec{
			Auth: "hmac", SignatureHeader: "X-ESP-Signature", SignaturePrefix: "sha256=", TimestampHeader: "X-ESP-Timestamp",
			Secret:     "{{ secrets.webhook_signing_key }}",
			EventsPath: "$.events[*]", TypePath: "$.event", RecipientPath: "$.recipient", MessageIDPath: "$.message_id",
			ReputeCastIDPath: "$.metadata.reputecast_id", CodePath: "$.smtp.code", DiagnosticPath: "$.smtp.message", TimePath: "$.ts",
			Types: map[string]string{"delivered": "delivered", "bounce_permanent": "hard_bounce", "bounce_temporary": "soft_bounce", "complaint": "complaint", "sent": "ignore"},
		},
		Check: &httpgeneric.CheckSpec{URL: base + "/v1/{{ settings.account }}/me"},
	}
}

func fakeAPI(env *contracttest.Env) http.Handler {
	var n atomic.Int64
	mux := http.NewServeMux()
	fail := func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case env.Mode() == contracttest.ModeOutage:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":{"code":"unavailable","message":"Try again later."}}`)
			return true
		case env.Mode() == contracttest.ModeAuth || r.Header.Get("Authorization") != "Bearer "+apiKey:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":{"code":"invalid_api_key","message":"The API key is not valid."}}`)
			return true
		}
		return false
	}
	mux.HandleFunc("POST /v1/acme/messages", func(w http.ResponseWriter, r *http.Request) {
		if fail(w, r) {
			return
		}
		var in struct {
			To []struct{ Email string } `json:"to"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.To) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"code":"bad_json","message":"The body is not JSON."}}`)
			return
		}
		if in.To[0].Email == contracttest.RejectAddress {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = io.WriteString(w, `{"error":{"code":"recipient_suppressed","message":"The recipient is on the suppression list."}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"id":"esp-`+strconv.FormatInt(n.Add(1), 10)+`"}}`)
	})
	mux.HandleFunc("GET /v1/acme/me", func(w http.ResponseWriter, r *http.Request) {
		if fail(w, r) {
			return
		}
		_, _ = io.WriteString(w, `{"account":"acme"}`)
	})
	return mux
}

func newProvider(t *testing.T, env *contracttest.Env) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(map[string]any{"http": mapping(env.URL), "account": "acme", "adapter": "Example ESP"})
	creds, _ := json.Marshal(map[string]string{"api_key": apiKey, "webhook_signing_key": signingKey})
	p, err := httpgeneric.New(providers.Config{Settings: settings, Credentials: creds, WebhookSecret: "reputecast-generated"})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func signed(file string) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		mac := hmac.New(sha256.New, []byte(signingKey))
		mac.Write([]byte(ts + "."))
		mac.Write(body)
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/httpgeneric/token", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-ESP-Timestamp", ts)
		r.Header.Set("X-ESP-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		return r, body
	}
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Adapter: httpgeneric.Adapter,
		API:     fakeAPI,
		New:     newProvider,
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != len(msgs) {
				t.Fatalf("want one request per message, got %d", len(reqs))
			}
			r := reqs[0]
			if r.Path != "/v1/acme/messages?region=eu" || r.Header.Get("X-Client") != "ReputeCast" || r.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("request: %s %v", r.Path, r.Header)
			}
			var body struct {
				From     string
				To       []struct{ Email, Name string }
				ReplyTo  string `json:"reply_to"`
				Subject  string
				HTML     string
				Headers  map[string]string
				Metadata map[string]string
			}
			if err := json.Unmarshal(r.Body, &body); err != nil {
				t.Fatalf("body is not JSON: %v\n%s", err, r.Body)
			}
			m := msgs[0]
			if body.From != `"Acme Newsletter" <news@news.acme.io>` || body.To[0].Email != m.To.Address || body.ReplyTo != m.From.Address ||
				body.Subject != m.Subject || body.HTML != m.HTML || body.Metadata["reputecast_id"] != m.ID.String() || body.Metadata["stream"] != "broadcast" {
				t.Fatalf("body: %+v", body)
			}
			if body.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" || body.Headers["X-ReputeCast-Message"] != m.ID.String() {
				t.Fatalf("headers: %v", body.Headers)
			}
			if res[0].ProviderMessageID != "esp-1" {
				t.Fatalf("provider message id %q, want esp-1", res[0].ProviderMessageID)
			}
		},
		Webhooks: []contracttest.Webhook{{
			Name: "JSONPathMapping", Request: signed("events.json"),
			Want: []providers.Event{
				{Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "esp-7f3a9c",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "delivered", OriginalCode: "250 2.0.0"},
				{Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "esp-7f3a9d",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "bounce_permanent", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown"},
				{Type: providers.EventSoftBounce, Recipient: "b.h@yahoo.com", ProviderMessageID: "esp-7f3a9e",
					OriginalType: "bounce_temporary", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full"},
				{Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "esp-7f3a9f",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "complaint"},
			},
		}},
	})
}

func TestEventDetails(t *testing.T) {
	env := contracttest.NewEnv(t, nil)
	env.URL = "https://api.example-esp.com"
	p := newProvider(t, env)
	req, body := signed("events.json")(t, env)
	evs, err := p.ParseWebhook(context.Background(), req, body)
	if err != nil {
		t.Fatal(err)
	}
	if evs[0].Diagnostic != "OK id=1vA9xQ-0001aB-3k" || !evs[0].OccurredAt.Equal(time.Unix(1760276462, 0)) {
		t.Fatalf("delivered: %+v", evs[0])
	}
	if !evs[3].OccurredAt.Equal(time.Date(2026, 10, 12, 13, 41, 30, 0, time.UTC)) {
		t.Fatalf("RFC 3339 time: %v", evs[3].OccurredAt)
	}
	var raw map[string]any
	if err := json.Unmarshal(evs[1].Raw, &raw); err != nil || raw["event"] != "bounce_permanent" {
		t.Fatalf("raw keeps the one event: %s", evs[1].Raw)
	}
}

func TestStatusMappingAndAuthModes(t *testing.T) {
	body, err := os.ReadFile("testdata/status.json")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		spec httpgeneric.WebhookSpec
		sign func(r *http.Request)
	}{
		{"hmac-default", httpgeneric.WebhookSpec{}, func(r *http.Request) {
			mac := hmac.New(sha256.New, []byte("reputecast-generated"))
			mac.Write(body)
			r.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
		}},
		{"hmac-base64", httpgeneric.WebhookSpec{SignatureHeader: "X-Hub-Signature", Encoding: "base64"}, func(r *http.Request) {
			mac := hmac.New(sha256.New, []byte("reputecast-generated"))
			mac.Write(body)
			r.Header.Set("X-Hub-Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		}},
		{"basic", httpgeneric.WebhookSpec{Auth: "basic"}, func(r *http.Request) { r.SetBasicAuth("reputecast", "reputecast-generated") }},
		{"header", httpgeneric.WebhookSpec{Auth: "header", TokenHeader: "X-Webhook-Token"}, func(r *http.Request) { r.Header.Set("X-Webhook-Token", "reputecast-generated") }},
		{"standard", httpgeneric.WebhookSpec{Auth: "standard", Secret: "{{ secrets.signing }}"}, func(r *http.Request) {
			ts := strconv.FormatInt(time.Now().Unix(), 10)
			sig, err := webhookauth.SignStandard("whsec_"+base64.StdEncoding.EncodeToString([]byte("standard-secret")), "msg_1", ts, body)
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("webhook-id", "msg_1")
			r.Header.Set("webhook-timestamp", ts)
			r.Header.Set("webhook-signature", "v1,bm90LXRoaXMtb25l "+sig)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := httpgeneric.Mapping{URL: "https://api.example-esp.com/v1/send", Body: `{"to":"{{ recipient }}"}`, WebhookMapping: "status", Webhook: tc.spec}
			settings, _ := json.Marshal(map[string]any{"http": m})
			creds, _ := json.Marshal(map[string]string{"signing": "whsec_" + base64.StdEncoding.EncodeToString([]byte("standard-secret"))})
			p, err := httpgeneric.New(providers.Config{Settings: settings, Credentials: creds, WebhookSecret: "reputecast-generated"})
			if err != nil {
				t.Fatal(err)
			}
			// A secret sent as it is must not be stored with the raw event.
			if hs := p.(providers.WebhookSecrets).WebhookSecretHeaders(); (tc.name == "header") != (len(hs) == 1 && hs[0] == "X-Webhook-Token") {
				t.Fatalf("secret headers: %v", hs)
			}
			r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/httpgeneric/x", bytes.NewReader(body))
			if _, err := p.ParseWebhook(context.Background(), r, body); !errors.Is(err, providers.ErrBadSignature) {
				t.Fatalf("an unsigned request must be refused: %v", err)
			}
			tc.sign(r)
			evs, err := p.ParseWebhook(context.Background(), r, body)
			if err != nil {
				t.Fatal(err)
			}
			if len(evs) != 5 {
				t.Fatalf("got %d events", len(evs))
			}
			want := []struct {
				typ   providers.EventType
				rcpt  string
				id    string
				code  string
				class string
			}{
				{providers.EventDelivered, "a.k@gmail.com", "m-1@esp.example", "250 2.0.0", ""},
				// Refused because of the sender: not the address's fault.
				{providers.EventSoftBounce, "k.n@hotmail.com", "m-2", "550 5.7.1", "spam_block"},
				{providers.EventSoftBounce, "p.s@yahoo.com", "m-3", "421 4.7.0", "reputation"},
				{providers.EventComplaint, "m.g@aol.com", "m-4", "", ""},
				{providers.EventClick, "a.k@gmail.com", "m-1", "", ""},
			}
			for i, w := range want {
				e := evs[i]
				if e.Type != w.typ || e.Recipient != w.rcpt || e.ProviderMessageID != w.id || e.OriginalCode != w.code || e.BounceClass != w.class {
					t.Fatalf("event %d: %+v", i, e)
				}
			}
			if evs[0].MessageID == nil || evs[0].MessageID.String() != "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e50" {
				t.Fatalf("ReputeCast id: %v", evs[0].MessageID)
			}
			if !evs[2].OccurredAt.Equal(time.Unix(1760276464, 0)) {
				t.Fatalf("milliseconds: %v", evs[2].OccurredAt)
			}
			if evs[4].URL != "https://acme.io/launch" || evs[4].UserAgent != "Mozilla/5.0" {
				t.Fatalf("click: %+v", evs[4])
			}
		})
	}
}

func TestStaleTimestampIsRefused(t *testing.T) {
	env := contracttest.NewEnv(t, nil)
	env.URL = "https://api.example-esp.com"
	p := newProvider(t, env)
	body, _ := os.ReadFile("testdata/events.json")
	ts := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	mac := hmac.New(sha256.New, []byte(signingKey))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/httpgeneric/token", bytes.NewReader(body))
	r.Header.Set("X-ESP-Timestamp", ts)
	r.Header.Set("X-ESP-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	if _, err := p.ParseWebhook(context.Background(), r, body); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a replayed request must be refused: %v", err)
	}
}

func TestBounceCodes(t *testing.T) {
	m := httpgeneric.Mapping{URL: "https://api.example-esp.com/v1/send", Body: `{"to":"{{ recipient }}"}`, WebhookMapping: "status", Webhook: httpgeneric.WebhookSpec{Auth: "none"}}
	settings, _ := json.Marshal(map[string]any{"http": m})
	p, err := httpgeneric.New(providers.Config{Settings: settings})
	if err != nil {
		t.Fatal(err)
	}
	type want struct {
		typ   providers.EventType
		class string
	}
	for code, w := range map[string]want{
		"":                  {providers.EventHardBounce, "other"},
		"550 5.1.1":         {providers.EventHardBounce, "mailbox_unknown"},
		"452 4.2.2":         {providers.EventSoftBounce, "mailbox_full"},
		"invalid_recipient": {providers.EventHardBounce, "mailbox_unknown"},
		"user-unknown":      {providers.EventHardBounce, "mailbox_unknown"},
		"domain_not_found":  {providers.EventHardBounce, "domain_invalid"},
		"hard_bounce":       {providers.EventHardBounce, "other"},
		"mailbox_full":      {providers.EventSoftBounce, "mailbox_full"},
		"spam_block":        {providers.EventSoftBounce, "spam_block"},
		"rate_limit":        {providers.EventSoftBounce, "other"},
	} {
		body := []byte(fmt.Sprintf(`{"status":"bounced","recipient":"a@b.example","code":%q}`, code))
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/httpgeneric/x", bytes.NewReader(body))
		evs, err := p.ParseWebhook(context.Background(), r, body)
		if err != nil || len(evs) != 1 {
			t.Fatalf("%q: %v, %v", code, evs, err)
		}
		if evs[0].Type != w.typ || evs[0].BounceClass != w.class || evs[0].OriginalCode != code {
			t.Errorf("bounced with code %q: %s/%s, want %s/%s", code, evs[0].Type, evs[0].BounceClass, w.typ, w.class)
		}
	}
}

func TestErrorsAndCheck(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	p := newProvider(t, env)
	m := contracttest.Message(0)
	m.To.Address = contracttest.RejectAddress
	res, err := p.Send(context.Background(), []providers.Message{m})
	if err != nil {
		t.Fatal(err)
	}
	if e := res[0].Err; e == nil || e.Code != "recipient_suppressed" || e.Message != "The recipient is on the suppression list." || e.Temporary {
		t.Fatalf("the mapped error code and message must be kept: %+v", res[0].Err)
	}
	v, ok := p.(providers.Verifier)
	if !ok {
		t.Fatal("a mapping with a check request verifies credentials")
	}
	q, err := v.Verify(context.Background())
	if err != nil || q.Account != "Example ESP · reachable" {
		t.Fatalf("verify: %+v, %v", q, err)
	}
	env.SetMode(contracttest.ModeAuth)
	if _, err := v.Verify(context.Background()); !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("rejected credentials: %v", err)
	}
	// Without a check request there is no probe that could hide an outage.
	mm := mapping(env.URL)
	mm.Check = nil
	settings, _ := json.Marshal(map[string]any{"http": mm, "account": "acme"})
	plain, err := httpgeneric.New(providers.Config{Settings: settings})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := plain.(providers.Verifier); ok {
		t.Fatal("a mapping without a check request must not implement Verifier")
	}
}

func TestPreviewMasksSecrets(t *testing.T) {
	env := contracttest.NewEnv(t, nil)
	env.URL = "https://api.example-esp.com"
	p := newProvider(t, env).(*httpgeneric.Checked)
	r, err := p.Preview(httpgeneric.SampleMessage())
	if err != nil {
		t.Fatal(err)
	}
	if r.Header.Get("Authorization") != "Bearer ••••" || strings.Contains(r.Body, apiKey) {
		t.Fatalf("secrets leak into the preview: %v", r.Header)
	}
	if r.URL != "https://api.example-esp.com/v1/acme/messages?region=eu" || !json.Valid([]byte(r.Body)) {
		t.Fatalf("preview: %s %s", r.URL, r.Body)
	}
	if !strings.Contains(r.Body, `"html": "<p>Hello \"Linh\" — this is a <b>sample</b>.</p>"`) {
		t.Fatalf("html is escaped for JSON, not for HTML: %s", r.Body)
	}
}

func TestValidate(t *testing.T) {
	ok := httpgeneric.Mapping{URL: "https://api.example-esp.com/v1/messages", Body: `{"to": "{{ recipient }}", "headers": {{ headers_json }}}`}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []struct {
		field string
		m     httpgeneric.Mapping
	}{
		{"method", httpgeneric.Mapping{Method: "GET", URL: ok.URL, Body: ok.Body}},
		{"url", httpgeneric.Mapping{URL: "http://api.example-esp.com/v1", Body: ok.Body}},
		{"url", httpgeneric.Mapping{URL: "https://{{ settings.host }}/v1", Body: ok.Body}},
		{"url", httpgeneric.Mapping{URL: "https://user:pw@api.example-esp.com/v1", Body: ok.Body}},
		{"body", httpgeneric.Mapping{URL: ok.URL, Body: `{"to": {{ recipient }`}},
		{"body", httpgeneric.Mapping{URL: ok.URL, Body: `{"to": "{{ recipient | shout }}"}`}},
		{"body", httpgeneric.Mapping{URL: ok.URL, Body: `{"to": {{ recipient }} "x"}`}},
		{"headers", httpgeneric.Mapping{URL: ok.URL, Body: ok.Body, Headers: map[string]string{"Host": "evil"}}},
		{"message_id_path", httpgeneric.Mapping{URL: ok.URL, Body: ok.Body, MessageIDPath: "data.id"}},
		{"webhook.types", httpgeneric.Mapping{URL: ok.URL, Body: ok.Body, WebhookMapping: "jsonpath", Webhook: httpgeneric.WebhookSpec{TypePath: "$.e", Types: map[string]string{"x": "exploded"}}}},
		{"webhook.auth", httpgeneric.Mapping{URL: ok.URL, Body: ok.Body, WebhookMapping: "status", Webhook: httpgeneric.WebhookSpec{Auth: "magic"}}},
	}
	for _, b := range bad {
		err := b.m.Validate()
		var fe *httpgeneric.FieldError
		if !errors.As(err, &fe) || fe.Field != b.field {
			t.Fatalf("%+v: want a %s error, got %v", b.m, b.field, err)
		}
	}
	local := httpgeneric.Mapping{URL: "http://127.0.0.1:8025/send", Body: ok.Body}
	if err := local.Validate(); err != nil {
		t.Fatalf("plain http is fine for a local test server: %v", err)
	}
}

// The Adapters tab saves {"name","kind":"http","capabilities":[…],"http":{method,url,body,message_id_path,webhook_mapping}}.
func TestManifestFromAdaptersTab(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	manifest, _ := json.Marshal(map[string]any{
		"name": "Example ESP", "kind": "http", "capabilities": []string{"webhooks"},
		"terms": map[string]any{"bulk": "allowed", "note": "Marketing to opted-in lists is allowed.", "url": "https://example-esp.com/aup"},
		"http": map[string]any{"method": "POST", "url": env.URL + "/v1/acme/messages", "body": bodyTemplate,
			"headers": map[string]string{"Authorization": "Bearer {{ secrets.api_key }}"}, "message_id_path": "$.data.id", "webhook_mapping": "status"},
	})
	if _, err := httpgeneric.ParseManifest(manifest); err != nil {
		t.Fatal(err)
	}
	creds, _ := json.Marshal(map[string]string{"api_key": apiKey})
	p, err := httpgeneric.FromManifest(manifest, providers.Config{Credentials: creds, WebhookSecret: "reputecast-generated"})
	if err != nil {
		t.Fatal(err)
	}
	if c := p.Capabilities(); c.Bulk != providers.BulkAllowed || c.Webhooks == "" || c.BatchSize != 1 {
		t.Fatalf("capabilities: %+v", c)
	}
	res, err := p.Send(context.Background(), []providers.Message{contracttest.Message(0)})
	if err != nil || res[0].Err != nil || !strings.HasPrefix(res[0].ProviderMessageID, "esp-") {
		t.Fatalf("send: %+v, %v", res, err)
	}
	if _, err := httpgeneric.ParseManifest([]byte(`{"name":"x","kind":"wasm","http":{}}`)); err == nil {
		t.Fatal("a WASM manifest is not an HTTP mapping")
	}
}
