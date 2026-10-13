package mailjet_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/mailjet"
)

const (
	apiKey    = "mj-public-key"
	secretKey = "mj-secret-key"
	secret    = "whsec-test-mailjet"
)

type fake struct {
	mu        sync.Mutex
	callbacks []map[string]any
	excluded  []int
	dkimOK    bool
	n         atomic.Int64
}

func (f *fake) with(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

func envelope(w http.ResponseWriter, data any) {
	b, _ := json.Marshal(data)
	n := 1
	var arr []any
	if json.Unmarshal(b, &arr) == nil {
		n = len(arr)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"Count": n, "Data": data, "Total": n})
}

func (f *fake) handler(env *contracttest.Env) http.Handler {
	mux := http.NewServeMux()
	guard := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			switch {
			case env.Mode() == contracttest.ModeOutage:
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = io.WriteString(w, `{"ErrorIdentifier":"ab12","StatusCode":503,"ErrorMessage":"Service unavailable"}`)
				return
			case env.Mode() == contracttest.ModeAuth || !ok || user != apiKey || pass != secretKey:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"ErrorIdentifier":"8e579ed2-30ee-4ad9-9a1a-9c9ef3b8c613","StatusCode":401,"ErrorMessage":"API key authentication/authorization failure. You may be unauthorized to access the API or your API key may be expired. Visit API keys management section to check your keys."}`)
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("POST /v3.1/send", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Messages []struct {
				To          []struct{ Email string } `json:"To"`
				CustomID    string                   `json:"CustomID"`
				TrackOpens  string                   `json:"TrackOpens"`
				TrackClicks string                   `json:"TrackClicks"`
			} `json:"Messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Messages) == 0 || len(in.Messages) > 50 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"ErrorIdentifier":"06df1144","ErrorCode":"mj-0002","ErrorMessage":"Malformed JSON, please review the syntax and properties types.","StatusCode":400}`)
			return
		}
		out, failed := []any{}, false
		for _, m := range in.Messages {
			if m.To[0].Email == contracttest.RejectAddress {
				failed = true
				out = append(out, map[string]any{"Status": "error", "Errors": []map[string]any{{"ErrorIdentifier": "88b5ca9f", "ErrorCode": "mj-0013",
					"StatusCode": 400, "ErrorMessage": "\"" + m.To[0].Email + "\" is an invalid email address.", "ErrorRelatedTo": []string{"To[0].Email"}}}})
				continue
			}
			id := 288230379175149952 + f.n.Add(1)
			out = append(out, map[string]any{"Status": "success", "CustomID": m.CustomID, "To": []map[string]any{{"Email": m.To[0].Email,
				"MessageUUID": "cb927469-36fd-4c02-bce4-0d199929a207", "MessageID": id, "MessageHref": "https://api.mailjet.com/v3/REST/message/" + strconv.FormatInt(id, 10)}}})
		}
		if failed {
			w.WriteHeader(http.StatusBadRequest)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"Messages": out})
	}))
	mux.HandleFunc("GET /v3/REST/apikey", guard(func(w http.ResponseWriter, _ *http.Request) {
		envelope(w, []map[string]any{{"APIKey": "other-key", "IsActive": false}, {"APIKey": apiKey, "SecretKey": secretKey, "IsActive": true, "Runlevel": "Normal", "QuarantineValue": 0}})
	}))
	mux.HandleFunc("GET /v3/REST/statcounters", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("CounterSource") != "APIKey" || r.URL.Query().Get("FromTS") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		envelope(w, []map[string]any{
			{"MessageSentCount": 1000, "MessageHardBouncedCount": 10, "MessageSoftBouncedCount": 5, "MessageSpamCount": 1},
			{"MessageSentCount": 1000, "MessageHardBouncedCount": 10, "MessageSoftBouncedCount": 5, "MessageSpamCount": 0},
		})
	}))
	mux.HandleFunc("GET /v3/REST/sender", guard(func(w http.ResponseWriter, _ *http.Request) {
		envelope(w, []map[string]any{{"ID": 77, "Email": "*@news.acme.io", "Status": "Inactive"}, {"ID": 78, "Email": "hello@acme.io", "Status": "Active"}})
	}))
	mux.HandleFunc("POST /v3/REST/sender", guard(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		envelope(w, []map[string]any{{"ID": 79, "Email": "*@fresh.acme.io", "Status": "Inactive", "DNSID": -1}})
	}))
	mux.HandleFunc("POST /v3/REST/sender/{id}/validate", guard(func(w http.ResponseWriter, _ *http.Request) {
		envelope(w, []map[string]any{{"ValidationMethod": "ActivationEmail", "Errors": map[string]any{}, "GlobalError": ""}})
	}))
	mux.HandleFunc("GET /v3/REST/dns/{domain}", guard(func(w http.ResponseWriter, r *http.Request) {
		d := r.PathValue("domain")
		f.mu.Lock()
		dkim := map[bool]string{true: "OK", false: "Not checked"}[f.dkimOK]
		f.mu.Unlock()
		envelope(w, []map[string]any{{"ID": 1234, "Domain": d, "DKIMRecordName": "mailjet._domainkey." + d + ".", "DKIMRecordValue": "k=rsa; p=MIGf",
			"DKIMStatus": dkim, "SPFRecordValue": "v=spf1 include:spf.mailjet.com ?all", "SPFStatus": "Not found",
			"OwnerShipToken": "123a4bc56d7890efg123h4ijk56l78mn", "OwnerShipTokenRecordName": "mailjet._123a4bc56." + d + "."}})
	}))
	mux.HandleFunc("POST /v3/REST/dns/{id}/check", guard(func(w http.ResponseWriter, _ *http.Request) {
		envelope(w, []map[string]any{{"DKIMErrors": "", "DKIMStatus": "OK", "SPFErrors": "", "SPFStatus": "Error"}})
	}))
	mux.HandleFunc("GET /v3/REST/eventcallbackurl", guard(func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		envelope(w, f.callbacks)
	}))
	mux.HandleFunc("POST /v3/REST/eventcallbackurl", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		in["ID"] = len(f.callbacks) + 1
		f.callbacks = append(f.callbacks, in)
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		envelope(w, []any{in})
	}))
	mux.HandleFunc("PUT /v3/REST/eventcallbackurl/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		id, _ := strconv.Atoi(r.PathValue("id"))
		f.mu.Lock()
		for _, c := range f.callbacks {
			if n, ok := c["ID"].(int); ok && n == id {
				for k, v := range in {
					c[k] = v
				}
			}
		}
		f.mu.Unlock()
		envelope(w, []any{in})
	}))
	mux.HandleFunc("GET /v3/REST/contact", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("IsExcludedFromCampaigns") != "true" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		envelope(w, []map[string]any{
			{"Email": "Angry@Example.com", "IsExcludedFromCampaigns": true, "IsSpamComplaining": true, "ExclusionFromCampaignsUpdatedAt": "2026-10-12T10:00:00Z"},
			{"Email": "subscribed@example.com", "IsExcludedFromCampaigns": false, "ExclusionFromCampaignsUpdatedAt": "2026-10-12T11:00:00Z"},
			{"Email": "gone@example.com", "IsExcludedFromCampaigns": true, "UnsubscribedAt": "2026-10-11T09:00:00Z", "ExclusionFromCampaignsUpdatedAt": "2026-10-11T09:00:00Z"},
			{"Email": "old@example.com", "IsExcludedFromCampaigns": true, "ExclusionFromCampaignsUpdatedAt": "2025-01-02T09:00:00Z"},
		})
	}))
	mux.HandleFunc("POST /v3/REST/contact/managemanycontacts", guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Contacts []struct {
				Email                   string
				IsExcludedFromCampaigns bool
			}
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if !in.Contacts[0].IsExcludedFromCampaigns {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.with(func() { f.excluded = append(f.excluded, len(in.Contacts)) })
		envelope(w, []map[string]any{{"JobID": 123}})
	}))
	mux.HandleFunc("GET /v3/REST/message/{id}", guard(func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{"100": "sent", "101": "hardbounced", "102": "queued", "103": "spam"}[r.PathValue("id")]
		if status == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		envelope(w, []map[string]any{{"ID": json.Number(r.PathValue("id")), "Status": status, "ArrivedAt": "2026-10-12T13:41:02Z"}})
	}))
	return mux
}

func newProvider(t *testing.T, env *contracttest.Env) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(mailjet.Settings{BaseURL: env.URL, MonthlyLimit: 15000})
	creds, _ := json.Marshal(mailjet.Credentials{APIKey: apiKey, SecretKey: secretKey})
	p, err := mailjet.New(providers.Config{Settings: settings, Credentials: creds, WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func webhook(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
	body, err := os.ReadFile("testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/mailjet/token", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.SetBasicAuth(mailjet.WebhookUser, secret)
	return r, body
}

func TestContract(t *testing.T) {
	f := &fake{}
	contracttest.Run(t, contracttest.Suite{
		Adapter: mailjet.Adapter,
		API:     f.handler,
		New:     newProvider,
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || reqs[0].Path != "/v3.1/send" {
				t.Fatalf("want one batch call, got %+v", reqs)
			}
			var sent struct {
				Messages []struct {
					From        struct{ Email, Name string }
					CustomID    string
					TrackOpens  string
					TrackClicks string
					Priority    int
					Headers     map[string]string
				}
				AdvanceErrorHandling bool
			}
			if err := json.Unmarshal(reqs[0].Body, &sent); err != nil {
				t.Fatal(err)
			}
			if len(sent.Messages) != len(msgs) || !sent.AdvanceErrorHandling {
				t.Fatalf("batch: %+v", sent)
			}
			for i, m := range sent.Messages {
				if m.CustomID != msgs[i].ID.String() || m.TrackOpens != "disabled" || m.TrackClicks != "disabled" || m.Priority != 0 {
					t.Fatalf("message %d: %+v", i, m)
				}
				if m.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" || m.Headers["X-ReputeCast-Message"] != msgs[i].ID.String() {
					t.Fatalf("headers %d: %v", i, m.Headers)
				}
			}
			if res[0].ProviderMessageID != "288230379175149953" {
				t.Fatalf("message ids above 2^53 stay exact: %q", res[0].ProviderMessageID)
			}
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			r.SetBasicAuth(mailjet.WebhookUser, "wrong-secret")
			r.Body = io.NopCloser(bytes.NewReader(body))
			return r, body
		},
		Webhooks: []contracttest.Webhook{{
			Name: "GroupedEvents", Request: webhook,
			Want: []providers.Event{
				{Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "288230379175149952",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "sent", OriginalCode: "250 2.0.0"},
				{Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "288230379175149953",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "bounce · hard · recipient · user unknown", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown"},
				{Type: providers.EventSoftBounce, Recipient: "b.h@yahoo.com", ProviderMessageID: "288230379175149954",
					OriginalType: "bounce · soft · recipient · quota exceeded", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full"},
				{Type: providers.EventSoftBounce, Recipient: "k.n@hotmail.com", ProviderMessageID: "288230379175149955",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e43"), OriginalType: "bounce · soft · spam · sender blocked", OriginalCode: "554 5.7.1", BounceClass: "spam_block"},
				{Type: providers.EventRejected, Recipient: "old@example.org", ProviderMessageID: "288230379175149956",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e44"), OriginalType: "blocked · mailjet · preblocked", OriginalCode: "preblocked", BounceClass: "policy_block"},
				{Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "288230379175149957",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "spam · JMRPP"},
				{Type: providers.EventUnsubscribe, Recipient: "gone@example.com", ProviderMessageID: "288230379175149958",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e45"), OriginalType: "unsub"},
			},
		}},
	})
}

func TestSingleAndEmptyCallbacks(t *testing.T) {
	env := contracttest.NewEnv(t, nil)
	env.URL = "https://api.mailjet.test"
	p := newProvider(t, env)
	for body, n := range map[string]int{``: 0, `[]`: 0, `{"event":"sent","time":1760276462,"MessageID":1,"email":"a@example.com","smtp_reply":"sent (250 OK)"}`: 1} {
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/mailjet/token", strings.NewReader(body))
		r.SetBasicAuth(mailjet.WebhookUser, secret)
		evs, err := p.ParseWebhook(context.Background(), r, []byte(body))
		if err != nil || len(evs) != n {
			t.Fatalf("%q: %v, %v", body, evs, err)
		}
	}
}

func TestPartialBatch(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	msgs := []providers.Message{contracttest.Message(0), contracttest.Message(1), contracttest.Message(2)}
	msgs[1].To.Address = contracttest.RejectAddress
	msgs[0].Stream = providers.StreamSystem
	res, err := newProvider(t, env).Send(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Err != nil || res[2].Err != nil || res[1].Err == nil || res[1].Err.Code != "mj-0013" || res[1].Err.Temporary {
		t.Fatalf("a 400 with per-message results: %+v", res)
	}
	var sent struct{ Messages []struct{ Priority int } }
	_ = json.Unmarshal(env.Requests()[0].Body, &sent)
	if sent.Messages[0].Priority != 2 || sent.Messages[2].Priority != 0 {
		t.Fatalf("priority: transactional 2, marketing 0: %+v", sent.Messages)
	}
}

func TestQuotaAndReputation(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	q, err := p.(providers.QuotaReporter).Quota(context.Background())
	if err != nil || q.Kind != "monthly_plan" || q.Used != 2000 || q.Max != 15000 || q.Account != "API key active" {
		t.Fatalf("quota: %+v, %v", q, err)
	}
	rep, err := p.(providers.ReputationReporter).Reputation(context.Background())
	if err != nil || rep.Bounce != 1.5 || rep.Complaint != 0.05 {
		t.Fatalf("reputation: %+v, %v", rep, err)
	}
	env.SetMode(contracttest.ModeAuth)
	if _, err := p.(providers.QuotaReporter).Quota(context.Background()); !errors.Is(err, providers.ErrUnauthorized) {
		t.Fatalf("rejected keys: %v", err)
	}
}

func TestDomains(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env).(providers.DomainManager)
	recs, err := p.DomainStatus(context.Background(), "news.acme.io")
	if err != nil {
		t.Fatal(err)
	}
	want := []providers.DNSRecord{
		{Purpose: "verification", Type: "TXT", Name: "mailjet._123a4bc56.news.acme.io", Value: "123a4bc56d7890efg123h4ijk56l78mn", Status: "verified"},
		{Purpose: "dkim", Type: "TXT", Name: "mailjet._domainkey.news.acme.io", Value: "k=rsa; p=MIGf", Status: "verified"},
		{Purpose: "spf", Type: "TXT", Name: "news.acme.io", Value: "v=spf1 include:spf.mailjet.com ?all", Status: "wrong_value"},
	}
	if len(recs) != 3 || recs[0] != want[0] || recs[1] != want[1] || recs[2] != want[2] {
		t.Fatalf("records: %+v", recs)
	}
	if _, err := p.AddDomain(context.Background(), "fresh.acme.io"); err != nil {
		t.Fatal(err)
	}
	created := false
	for _, r := range env.Requests() {
		created = created || r.Method == http.MethodPost && r.Path == "/v3/REST/sender"
	}
	if !created {
		t.Fatal("a new domain gets a *@domain sender")
	}
}

func TestCallbacks(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	const hook = "https://reputecast.acme.io/hooks/mailjet/tok-1"
	for range 2 {
		if err := p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), hook); err != nil {
			t.Fatal(err)
		}
	}
	f.with(func() {
		if len(f.callbacks) != 5 {
			t.Fatalf("one callback per event type: %v", f.callbacks)
		}
		u, err := url.Parse(f.callbacks[0]["Url"].(string))
		if err != nil || u.User.Username() != mailjet.WebhookUser || u.Host != "reputecast.acme.io" || f.callbacks[0]["Version"] != float64(2) {
			t.Fatalf("callback URL carries basic auth: %v", f.callbacks[0])
		}
		if pw, _ := u.User.Password(); pw != secret {
			t.Fatal("the password is the webhook secret")
		}
	})
	st, err := p.(providers.WebhookInspector).InspectWebhook(context.Background(), hook)
	if err != nil || !st.Registered || st.ID != "1,2,3,4,5" {
		t.Fatalf("inspect: %+v, %v", st, err)
	}
	err = p.(providers.WebhookRegistrar).RegisterWebhook(context.Background(), "https://reputecast.acme.io/hooks/mailjet/tok-2")
	if err == nil || !strings.Contains(err.Error(), "one callback URL per event") {
		t.Fatalf("callbacks of another URL are left alone: %v", err)
	}
}

func TestSuppressionsAndBackfill(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	p := newProvider(t, env)
	list, err := p.(providers.SuppressionSyncer).ListSuppressions(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(list) != 2 || list[0].Email != "angry@example.com" || list[0].Reason != "complaint" || list[1].Reason != "unsubscribe" {
		t.Fatalf("suppressions: %+v, %v", list, err)
	}
	emails := make([]string, 1100)
	for i := range emails {
		emails[i] = "x" + strconv.Itoa(i) + "@example.com"
	}
	if err := p.(providers.SuppressionSyncer).AddSuppressions(context.Background(), emails); err != nil {
		t.Fatal(err)
	}
	f.with(func() {
		if len(f.excluded) != 3 || f.excluded[2] != 100 {
			t.Fatalf("chunks: %v", f.excluded)
		}
	})
	evs, err := p.(providers.EventBackfiller).Backfill(context.Background(), []string{"100", "101", "102", "103", "104", "not-a-number"})
	if err != nil || len(evs) != 3 || evs[0].Type != providers.EventDelivered || evs[1].Type != providers.EventHardBounce || evs[2].Type != providers.EventComplaint {
		t.Fatalf("backfill: %+v, %v", evs, err)
	}
	tr, err := p.(providers.TrackingManager).Tracking(context.Background())
	if err != nil || tr.OpenPixel || tr.LinkRewrite {
		t.Fatalf("tracking: %+v, %v", tr, err)
	}
}

func TestAuthHeader(t *testing.T) {
	f := &fake{}
	env := contracttest.NewEnv(t, f.handler)
	if _, err := newProvider(t, env).Send(context.Background(), []providers.Message{contracttest.Message(0)}); err != nil {
		t.Fatal(err)
	}
	if got := env.Requests()[0].Header.Get("Authorization"); got != "Basic "+base64.StdEncoding.EncodeToString([]byte(apiKey+":"+secretKey)) {
		t.Fatalf("auth: %q", got)
	}
}
