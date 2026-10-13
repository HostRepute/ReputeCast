package connect_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/connect"
	"github.com/hostrepute/reputecast/internal/providers"
)

const ws = "/workspaces/acme-newsletter"

// fakeESP is an API provider for the tests: "good-key" is accepted, every
// other key is rejected the way Mailgun rejects it.
type fakeESP struct {
	cfg providers.Config
	key string
}

var (
	fakeMu       sync.Mutex
	fakeWebhooks []string
	fakeSent     []providers.Message
)

func init() {
	providers.Register(providers.Descriptor{
		Adapter: "fakeesp", Name: "Fake ESP", Kind: "api",
		Capabilities: providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "Registered", DomainAPI: "full", SuppressionSync: true, BatchSize: 100,
			DedicatedIPs: "paid", Sandbox: "mode", QuotaDiscovery: true, Pricing: "$0.50 per 1,000"},
		Fields: []providers.Field{{Key: "api_key", Label: "API key", Secret: true, Required: true}, {Key: "region", Label: "Region", Options: []string{"us", "eu"}},
			{Key: "monthly_limit", Label: "Plan · emails per month"}},
		New: func(cfg providers.Config) (providers.Provider, error) {
			var c struct {
				Key string `json:"api_key"`
			}
			_ = json.Unmarshal(cfg.Credentials, &c)
			return &fakeESP{cfg: cfg, key: c.Key}, nil
		},
	})
}

func init() {
	providers.Register(providers.Descriptor{
		Adapter: "fakemanual", Name: "Fake Grid", Kind: "api",
		Capabilities: providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "Event Grid", BatchSize: 1},
		Fields:       []providers.Field{{Key: "connection_string", Label: "Connection string", Secret: true, Required: true}, {Key: "per_minute", Label: "Sending limit · per minute"}},
		New:          func(cfg providers.Config) (providers.Provider, error) { return &fakeManual{}, nil },
	})
}

// fakeManual can't register its webhook: it is set up by hand and sends the
// generated secret in a header.
type fakeManual struct{}

func (f *fakeManual) Adapter() string { return "fakemanual" }
func (f *fakeManual) Capabilities() providers.Capabilities {
	return providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "Event Grid", BatchSize: 1}
}
func (f *fakeManual) Send(_ context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return make([]providers.SendResult, len(msgs)), nil
}
func (f *fakeManual) ParseWebhook(context.Context, *http.Request, []byte) ([]providers.Event, error) {
	return []providers.Event{}, nil
}
func (f *fakeManual) VerifyWebhook(context.Context, *http.Request, []byte) error { return nil }
func (f *fakeManual) WebhookSecretHeaders() []string                             { return []string{"X-Fake-Secret"} }
func (f *fakeManual) Verify(context.Context) (providers.Quota, error)            { return providers.Quota{}, nil }

func (f *fakeESP) Adapter() string { return "fakeesp" }
func (f *fakeESP) Capabilities() providers.Capabilities {
	d, _ := providers.Lookup("fakeesp")
	return d.Capabilities
}
func (f *fakeESP) Send(_ context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	fakeMu.Lock()
	defer fakeMu.Unlock()
	out := make([]providers.SendResult, len(msgs))
	for i, m := range msgs {
		fakeSent = append(fakeSent, m)
		out[i] = providers.SendResult{MessageID: m.ID, ProviderMessageID: "fake-" + m.ID.String()}
	}
	return out, nil
}
func (f *fakeESP) ParseWebhook(context.Context, *http.Request, []byte) ([]providers.Event, error) {
	return []providers.Event{}, nil
}
func (f *fakeESP) Verify(context.Context) (providers.Quota, error) {
	if f.key != "good-key" {
		return providers.Quota{}, &providers.CallError{Op: "GET /v3/domains", Status: 401, Code: "Forbidden", Message: "Invalid private key", Auth: true}
	}
	return providers.Quota{Kind: "monthly_plan", Max: 50000, Account: "Active · 50,000 per month plan"}, nil
}
func (f *fakeESP) RegisterWebhook(_ context.Context, url string) error {
	fakeMu.Lock()
	defer fakeMu.Unlock()
	fakeWebhooks = append(fakeWebhooks, url)
	return nil
}
func (f *fakeESP) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	return f.DomainStatus(ctx, domain)
}
func (f *fakeESP) DomainStatus(_ context.Context, domain string) ([]providers.DNSRecord, error) {
	return []providers.DNSRecord{
		{Purpose: "dkim", Type: "CNAME", Name: "fk1._domainkey." + domain, Value: "fk1.dkim.fake.example", Status: "pending"},
		{Purpose: "return_path", Type: "CNAME", Name: "fb." + domain, Value: "bounces.fake.example", Status: "pending"},
		{Purpose: "spf", Type: "TXT", Name: "fb." + domain, Value: "v=spf1 include:spf.fake.example ~all", Status: "pending"},
	}, nil
}
func (f *fakeESP) ListSuppressions(context.Context, time.Time) ([]providers.Suppression, error) {
	return []providers.Suppression{{Email: "a@x.example", Reason: "hard_bounce"}, {Email: "b@x.example", Reason: "hard_bounce"}, {Email: "c@x.example", Reason: "complaint"}}, nil
}
func (f *fakeESP) AddSuppressions(context.Context, []string) error { return nil }
func (f *fakeESP) Backfill(context.Context, []string) ([]providers.Event, error) {
	return []providers.Event{}, nil
}

type setupT struct {
	ID, Adapter, Step, Status, ProviderID, Provider string
	Info                                            struct {
		Flow, Name, Title, Heading, KeyHint string
		CanRegister, CanPoll                bool
		Fields                              []struct {
			Key, Value, Type string
			Secret           bool
			Set              bool
			Options          []struct{ Value, Label string }
		}
	}
	Summaries  struct{ Provider, Connection, Domain, Test, Streams string }
	Connection struct {
		Status, Account, Summary string
		Error                    *struct{ Field, Code, Message, Detail string }
		Webhook                  *struct {
			Mode, URL string
			SelfTest  *struct {
				OK           bool
				Code, Reason string
			}
			Registration *struct{ Status, Detail string }
		}
		CanContinue bool
	}
	Domain  struct{ DomainID, Name, Suggested string }
	Streams struct {
		Streams []struct {
			Stream, Role, RoleLabel, Failover string
			Enabled, Allowed, HasRoute        bool
			Current                           []string
		}
		Defaults string
	}
	Done *struct {
		Title string
		Rows  []struct{ Status, Lead, Text string }
	}
}

func audits(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCatalog(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var cat struct {
		Builtin []struct {
			Adapter, Name, Short, Terms, TermsTone, Pricing string
			Recommended, BulkAllowed                        bool
			Chips                                           []struct{ Label, Kind string }
			Connected                                       int
		}
		Community []struct {
			Name, Version, Summary, Adapter, Reason string
			Configure                               bool
		}
		CanConnect bool
	}
	c.Do("GET", ws+"/connect/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	if !cat.CanConnect || len(cat.Builtin) < 5 {
		t.Fatalf("catalog: %+v", cat)
	}
	first := cat.Builtin[0]
	if first.Adapter != "ses" || first.Name != "Amazon SES" || !first.Recommended || first.Terms != "Recommended" || first.Connected != 1 {
		t.Fatalf("SES card: %+v", first)
	}
	labels := []string{}
	for _, ch := range first.Chips {
		labels = append(labels, ch.Kind+":"+ch.Label)
	}
	if got := strings.Join(labels, ","); got != "yes:Webhooks (SNS),yes:Domain via API,yes:Suppression sync,yes:Batch 50,partial:Dedicated IPs, paid,yes:Sandbox detection,yes:Quota discovery" {
		t.Fatalf("SES chips: %s", got)
	}
	names := map[string]string{}
	for _, b := range cat.Builtin {
		names[b.Adapter] = b.Name + "|" + b.Terms
	}
	if names["smtp"] != "Any SMTP server|Depends on the server" || names["postal"] != "Postal (self-hosted)|Your own policy" || names["fakeesp"] != "Fake ESP|Bulk marketing allowed" {
		t.Fatalf("names: %v", names)
	}
	// Installed community adapters show (the unmaintained one doesn't), then
	// the generic HTTP adapter, configured on the Adapters page.
	if len(cat.Community) != 3 || cat.Community[0].Name != "Scaleway Transactional Email" || cat.Community[0].Adapter != "" || cat.Community[0].Reason == "" ||
		!strings.Contains(cat.Community[0].Summary, "webhooks, suppression pull") || cat.Community[2].Name != "Generic HTTP adapter" || !cat.Community[2].Configure {
		t.Fatalf("community: %+v", cat.Community)
	}
	// Adapters added later appear by themselves, after the design's order.
	if n := len(cat.Builtin); n < 13 || cat.Builtin[n-2].Adapter != "fakeesp" || cat.Builtin[n-1].Adapter != "fakemanual" {
		t.Fatalf("builtin order: %d, last %s", len(cat.Builtin), cat.Builtin[len(cat.Builtin)-1].Adapter)
	}
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/connect/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	if cat.CanConnect {
		t.Fatal("viewers can't connect")
	}
	v.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "fakeesp"}).MustStatus(t, 403)
}

func TestConnectAPIProvider(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "fakeesp"}).MustStatus(t, 201).JSON(t, &st)
	if st.Step != "connection" || st.Info.Flow != "api" || st.Info.Heading != "Paste your Fake ESP API key" || !st.Info.CanRegister || !st.Info.CanPoll || len(st.Info.Fields) != 3 {
		t.Fatalf("new setup: %+v", st)
	}
	// Starting again resumes the open setup.
	var again setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "fakeesp"}).MustStatus(t, 201).JSON(t, &again)
	if again.ID != st.ID {
		t.Fatal("open setup not resumed")
	}
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "nope"}).MustStatus(t, 404)
	base := ws + "/connect/setups/" + st.ID

	// A rejected key is the error state; nothing is saved.
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"api_key": "bad-key", "region": "us"}}).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Status != "failed" || st.Connection.Error == nil || st.Connection.Error.Field != "api_key" || st.Connection.Error.Code != "invalid_credentials" ||
		!strings.Contains(st.Connection.Error.Detail, "401 Unauthorized") || st.ProviderID != "" {
		t.Fatalf("rejected key: %+v %+v", st.Connection, st.Connection.Error)
	}
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"region": "us"}}).MustStatus(t, 422)
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"api_key": "good-key", "monthly_limit": "lots"}}).MustStatus(t, 422)
	for _, f := range st.Info.Fields {
		if f.Key == "monthly_limit" && f.Type != "number" || f.Key == "region" && f.Type != "text" {
			t.Fatalf("field types: %+v", st.Info.Fields)
		}
	}

	// The right key connects the provider and registers the webhook once
	// the endpoint answered its self-test.
	r := c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"api_key": "good-key", "region": "eu", "monthly_limit": "50,000"}}).MustStatus(t, 200)
	if strings.Contains(string(r.Body), "good-key") {
		t.Fatal("the key came back")
	}
	r.JSON(t, &st)
	w := st.Connection.Webhook
	if st.Connection.Status != "connected" || st.ProviderID == "" || w == nil || w.SelfTest == nil || !w.SelfTest.OK || w.Registration == nil ||
		w.Registration.Status != "done" || !st.Connection.CanContinue || st.Connection.Summary != "eu · active" {
		t.Fatalf("connected: %+v / %+v", st.Connection, w)
	}
	fakeMu.Lock()
	registered := len(fakeWebhooks) > 0 && strings.HasSuffix(fakeWebhooks[len(fakeWebhooks)-1], w.URL[strings.Index(w.URL, "/hooks/"):])
	fakeMu.Unlock()
	if !registered {
		t.Fatalf("webhook not registered: %v", fakeWebhooks)
	}
	var enc []byte
	var cfg json.RawMessage
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT credentials_enc, config FROM providers WHERE id = $1`, st.ProviderID).Scan(&enc, &cfg); err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	_ = json.Unmarshal(cfg, &settings)
	if strings.Contains(string(enc), "good-key") || strings.Contains(string(cfg), "good-key") || settings["region"] != "eu" || settings["monthly_limit"] != float64(50000) {
		t.Fatalf("credentials stored in clear: %s / %s", enc, cfg)
	}
	for _, f := range st.Info.Fields {
		if f.Key == "api_key" && (!f.Set || f.Value != "") || f.Key == "region" && f.Value != "eu" {
			t.Fatalf("fields: %+v", st.Info.Fields)
		}
	}
	if audits(t, env, "provider.connect") != 1 || audits(t, env, "provider.connect_check") != 2 {
		t.Fatal("audit entries")
	}

	// Domain step: the provider's records for news.acme.io.
	c.Do("PATCH", base, map[string]any{"step": "domain"}).MustStatus(t, 200).JSON(t, &st)
	if st.Step != "domain" || st.Domain.Suggested != "news.acme.io" {
		t.Fatalf("domain step: %+v", st.Domain)
	}
	c.Do("POST", base+"/domain", map[string]any{"domain": "news.acme.io"}).MustStatus(t, 200).JSON(t, &st)
	if st.Domain.DomainID == "" || !strings.HasPrefix(st.Summaries.Domain, "news.acme.io · ") {
		t.Fatalf("domain chosen: %+v %+v", st.Domain, st.Summaries)
	}
	var dom struct {
		Records []struct{ Provider, Purpose, Label, Status, Name, Value string }
	}
	c.Do("GET", ws+"/domains/"+st.Domain.DomainID+"?provider="+st.ProviderID+"&refresh=true", nil).MustStatus(t, 200).JSON(t, &dom)
	labels := []string{}
	for _, r := range dom.Records {
		labels = append(labels, r.Label+":"+r.Status)
	}
	if got := strings.Join(labels, ","); got != "DKIM:missing,Return-path / MAIL FROM:missing,SPF for MAIL FROM:missing,SPF for the sending domain:wrong_value,DMARC:verified" {
		t.Fatalf("records: %s", got)
	}

	// Test send: through the provider, recorded, with a timeline.
	var test struct {
		MessageID, SenderID, To string
		Senders                 []struct{ ID, Label string }
		Timeline                []struct{ Key, Title, Status, Detail string }
		Listening               bool
	}
	c.Do("GET", base+"/test", nil).MustStatus(t, 200).JSON(t, &test)
	if test.To != "linh@acme.io" || len(test.Senders) != 6 || test.MessageID != "" {
		t.Fatalf("test form: %+v", test)
	}
	digest := ""
	for _, s := range test.Senders {
		if strings.Contains(s.Label, "digest@") {
			digest = s.ID
		}
	}
	c.Do("POST", base+"/test", map[string]any{"senderId": digest, "to": "linh@acme.io", "subject": "ReputeCast test from Fake ESP"}).MustStatus(t, 200).JSON(t, &test)
	if test.MessageID == "" || len(test.Timeline) != 5 || test.Timeline[0].Status != "done" || test.Timeline[1].Status != "done" || test.Timeline[3].Status != "waiting" || !test.Listening {
		t.Fatalf("timeline: %+v", test.Timeline)
	}
	fakeMu.Lock()
	last := fakeSent[len(fakeSent)-1]
	fakeMu.Unlock()
	if last.From.Address != "digest@news.acme.io" || last.To.Address != "linh@acme.io" || !strings.Contains(last.HTML, "This is a test from") || last.Stream != providers.StreamBroadcast {
		t.Fatalf("sent: %+v", last)
	}
	// The delivery event arrives (here written as the normalizer would).
	mid := uuid.MustParse(test.MessageID)
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO delivery_events (id, workspace_id, message_id, provider_id, type, occurred_at, original_type, original_code, diagnostic)
		SELECT $1, workspace_id, id, provider_id, 'delivered', now(), 'delivered', '250', '2.0.0 OK' FROM messages WHERE id = $2`, uuid.Must(uuid.NewV7()), mid); err != nil {
		t.Fatal(err)
	}
	var test2 struct {
		Timeline  []struct{ Key, Status, Title string }
		RoundTrip string
	}
	c.Do("GET", base+"/test", nil).MustStatus(t, 200).JSON(t, &test2)
	if test2.Timeline[3].Status != "done" || !strings.HasPrefix(test2.RoundTrip, "Round trip ") || !strings.HasSuffix(test2.RoundTrip, "Webhooks work.") {
		t.Fatalf("delivered: %+v / %s", test2.Timeline, test2.RoundTrip)
	}

	// Streams: a 20% split of Broadcasts and the Automations fallback.
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &st)
	if len(st.Streams.Streams) != 3 || !st.Streams.Streams[0].HasRoute || st.Streams.Streams[0].Role != "split" || st.Streams.Streams[2].Role != "fallback" {
		t.Fatalf("streams view: %+v", st.Streams)
	}
	c.Do("PUT", base+"/streams", map[string]any{"streams": []map[string]any{
		{"stream": "broadcast", "enabled": true, "role": "split", "weight": 20},
		{"stream": "automation", "enabled": true, "role": "fallback"},
		{"stream": "system", "enabled": false},
	}}).MustStatus(t, 200).JSON(t, &st)
	var routing struct {
		Streams []struct {
			Stream   string
			Fallback string
			Splits   []struct {
				ProviderID string
				Weight     int
			}
		}
	}
	c.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &routing)
	weights := map[string]int{}
	for _, sp := range routing.Streams[0].Splits {
		weights[sp.ProviderID] = sp.Weight
	}
	if weights[st.ProviderID] != 20 || len(weights) != 3 || routing.Streams[1].Fallback != st.ProviderID {
		t.Fatalf("routing: %+v", routing.Streams)
	}
	total := 0
	for _, w := range weights {
		total += w
	}
	if total != 100 || st.Summaries.Streams != "Broadcasts, Automations" {
		t.Fatalf("weights %v, summary %q", weights, st.Summaries.Streams)
	}

	// Contacts and finish.
	var contacts struct {
		Suppressions struct {
			Available                  bool
			Total, Bounces, Complaints int
		}
		Lists, Suggested []string
	}
	c.Do("GET", base+"/contacts", nil).MustStatus(t, 200).JSON(t, &contacts)
	if !contacts.Suppressions.Available || contacts.Suppressions.Total != 3 || contacts.Suppressions.Bounces != 2 || len(contacts.Lists) != 3 {
		t.Fatalf("contacts: %+v", contacts)
	}
	c.Do("POST", base+"/finish", map[string]any{"pullSuppressions": true, "startEmpty": true}).MustStatus(t, 200).JSON(t, &st)
	if st.Status != "finished" || st.Step != "done" || st.Done == nil || st.Done.Title != "Fake ESP is connected" {
		t.Fatalf("finished: %+v", st)
	}
	rows := []string{}
	for _, r := range st.Done.Rows {
		rows = append(rows, r.Status+" "+r.Lead+r.Text)
	}
	joined := strings.Join(rows, "\n")
	for _, want := range []string{"done Fake ESP access", "done Events: webhooks registered at", "pending Domain news.acme.io", "done Streams: Fake ESP is a weighted share of Broadcasts; the fallback for Automations",
		"done Suppressions: 3 addresses imported"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("summary is missing %q:\n%s", want, joined)
		}
	}
	var mode string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT mode FROM provider_suppression_sync WHERE provider_id = $1`, st.ProviderID).Scan(&mode); err != nil || mode != "two_way" {
		t.Fatalf("sync mode %q %v", mode, err)
	}
	for _, a := range []string{"provider.setup_start", "provider.setup_domain", "provider.test_send", "provider.setup_streams", "routing.update", "provider.setup_finish"} {
		if audits(t, env, a) == 0 {
			t.Fatalf("no audit %s", a)
		}
	}
}

func TestManualWebhookSecret(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st struct {
		ID         string
		Connection struct {
			Webhook *struct {
				URL, SecretHeader string
				NeedsSecret       bool
				Registration      *struct{ Status string }
			}
		}
	}
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "fakemanual"}).MustStatus(t, 201).JSON(t, &st)
	base := ws + "/connect/setups/" + st.ID
	// Before the connection there's no secret to show.
	c.Do("POST", base+"/webhook-secret", nil).MustStatus(t, 409)
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"connection_string": "endpoint=https://x;accesskey=y", "per_minute": "30"}}).MustStatus(t, 200).JSON(t, &st)
	w := st.Connection.Webhook
	if w == nil || w.Registration == nil || w.Registration.Status != "manual" || !w.NeedsSecret || w.SecretHeader != "X-Fake-Secret" {
		t.Fatalf("manual webhook: %+v", w)
	}
	var sec struct{ URL, Secret, Header string }
	r := c.Do("POST", base+"/webhook-secret", nil).MustStatus(t, 200)
	r.JSON(t, &sec)
	if len(sec.Secret) < 20 || sec.Header != "X-Fake-Secret" || sec.URL != w.URL {
		t.Fatalf("secret: %+v", sec)
	}
	// It isn't part of the setup itself.
	if strings.Contains(string(c.Do("GET", base, nil).MustStatus(t, 200).Body), sec.Secret) {
		t.Fatal("the secret is in the setup")
	}
	if audits(t, env, "provider.webhook_secret_show") != 1 {
		t.Fatal("showing the secret isn't audited")
	}
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("POST", base+"/webhook-secret", nil).MustStatus(t, 403)
}

// Provider URLs go through the SSRF guard: a metadata address is refused
// with a message that says why, whatever the instance allows.
func TestBlockedProviderAddress(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "postal"}).MustStatus(t, 201).JSON(t, &st)
	var res struct {
		ProviderID string
		Connection struct {
			Status string
			Error  *struct{ Field, Code, Message string }
		}
	}
	c.Do("POST", ws+"/connect/setups/"+st.ID+"/connection", map[string]any{"values": map[string]string{"server_url": "http://169.254.169.254", "api_key": "abc123"}}).
		MustStatus(t, 200).JSON(t, &res)
	e := res.Connection.Error
	if res.Connection.Status != "failed" || res.ProviderID != "" || e == nil || e.Code != "address_blocked" || e.Field != "server_url" ||
		!strings.Contains(e.Message, "providers.allow_private_networks") {
		t.Fatalf("blocked: %+v %+v", res.Connection, e)
	}
}

func TestConnectPollingAndCancel(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "fakeesp"}).MustStatus(t, 201).JSON(t, &st)
	base := ws + "/connect/setups/" + st.ID
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"api_key": "good-key"}}).MustStatus(t, 200).JSON(t, &st)
	c.Do("POST", base+"/polling", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Webhook == nil || st.Connection.Webhook.Mode != "polling" || !st.Connection.CanContinue {
		t.Fatalf("polling: %+v", st.Connection.Webhook)
	}
	// A tunnel URL must be https.
	c.Do("POST", base+"/webhook", map[string]any{"publicUrl": "http://insecure.example"}).MustStatus(t, 422)
	// Cancelling disconnects the provider no route uses.
	c.Do("DELETE", base, nil).MustStatus(t, 204)
	var disconnected bool
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT disconnected_at IS NOT NULL FROM providers WHERE id = $1`, st.ProviderID).Scan(&disconnected); err != nil || !disconnected {
		t.Fatalf("provider still connected: %v", err)
	}
	c.Do("DELETE", base, nil).MustStatus(t, 409)
}

func TestSelfTestPrivateURL(t *testing.T) {
	now := time.Now()
	for _, u := range []string{"http://localhost:5210/hooks/x/y", "https://10.0.4.18/hooks/x/y", "http://reputecast.local/hooks/x/y"} {
		r := connect.SelfTest(context.Background(), u, "token-0123456789abcdef", now, false)
		if r.OK || !r.Private || r.Code != "ECONNREFUSED" {
			t.Errorf("%s: %+v", u, r)
		}
		if p := connect.Preflight(context.Background(), u, now, false); p == nil || !p.Private {
			t.Errorf("preflight %s: %+v", u, p)
		}
	}
	if p := connect.Preflight(context.Background(), "http://localhost:5210/hooks/x/y", now, true); p != nil {
		t.Errorf("allowed private URL: %+v", p)
	}
	if p := connect.Preflight(context.Background(), "https://203.0.113.9/hooks/x/y", now, false); p != nil {
		t.Errorf("public address: %+v", p)
	}
}

func TestConnectSMTP(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	sink := apptest.NewSMTPSink(t)
	host, port, _ := net.SplitHostPort(sink.Addr)
	n, _ := strconv.Atoi(port)
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "smtp"}).MustStatus(t, 201).JSON(t, &st)
	if st.Info.Flow != "smtp" || st.Info.Title != "Connect an SMTP server" {
		t.Fatalf("smtp setup: %+v", st.Info)
	}
	base := ws + "/connect/setups/" + st.ID
	var res struct {
		ProviderID string
		Connection struct {
			Status string
			SMTP   *struct {
				OK                  bool
				Summary, Transcript string
			}
			SMTPForm    struct{ Host, MaxRate, DailyCap string }
			CanContinue bool
		}
	}
	c.Do("POST", base+"/connection", map[string]any{"smtp": map[string]any{"host": host, "port": n, "tls": "none", "username": "acme", "password": "secret",
		"maxRate": "5 per second", "dailyCap": "20,000"}}).MustStatus(t, 200).JSON(t, &res)
	if res.Connection.Status != "connected" || res.Connection.SMTP == nil || res.Connection.SMTP.Summary != "No TLS · AUTH PLAIN accepted" ||
		!strings.HasPrefix(res.Connection.SMTP.Transcript, "220 sink.test ESMTP Sink · EHLO ") || !strings.Contains(res.Connection.SMTP.Transcript, "250-AUTH PLAIN LOGIN") ||
		res.Connection.CanContinue || res.Connection.SMTPForm.MaxRate != "5 per second" {
		t.Fatalf("smtp: %+v %+v", res.Connection, res.Connection.SMTP)
	}
	var health struct {
		Quota struct{ Max int64 }
		Rate  struct {
			Max   float64
			Label string
		}
	}
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT health FROM providers WHERE id = $1`, res.ProviderID).Scan(&health); err != nil {
		t.Fatal(err)
	}
	if health.Quota.Max != 20000 || health.Rate.Max != 5 || health.Rate.Label != "set manually" {
		t.Fatalf("limits: %+v", health)
	}
	// Wrong credentials.
	sink.Reject = true
	var failed struct {
		Connection struct {
			Status string
			Error  *struct{ Field, Message string }
		}
	}
	c.Do("POST", base+"/connection", map[string]any{"smtp": map[string]any{"host": host, "port": n, "tls": "none", "username": "acme", "password": "wrong"}}).MustStatus(t, 200).JSON(t, &failed)
	if failed.Connection.Status != "failed" || failed.Connection.Error == nil || failed.Connection.Error.Field != "password" {
		t.Fatalf("rejected smtp: %+v", failed.Connection)
	}
	// The bounce mailbox can't be reached here: the test says so.
	var mb struct {
		Connection struct {
			Mailbox *struct {
				OK    bool
				Error string
			}
		}
	}
	c.Do("POST", base+"/mailbox", map[string]any{"address": "bounces@news.acme.io", "protocol": "imap", "host": "127.0.0.1", "port": 1, "tls": "none",
		"username": "bounces@news.acme.io", "password": "x", "folder": "INBOX", "pollEvery": "1m", "afterProcessing": "move"}).MustStatus(t, 200).JSON(t, &mb)
	if mb.Connection.Mailbox == nil || mb.Connection.Mailbox.OK || mb.Connection.Mailbox.Error == "" {
		t.Fatalf("mailbox: %+v", mb.Connection.Mailbox)
	}
}

func TestDemoSetupAndSettings(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var list struct {
		Items []struct{ ID, Name, Step, Status, Provider string }
	}
	c.Do("GET", ws+"/connect/setups?adapter=ses&status=finished", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Name != "Amazon SES" || list.Items[0].Provider != "Amazon SES" || list.Items[0].Step != "done" {
		t.Fatalf("setups: %+v", list.Items)
	}
	setupID := list.Items[0].ID
	// The only unfinished setup is the Mailgun attempt whose key was
	// rejected (SetupError); nothing was connected for it.
	c.Do("GET", ws+"/connect/setups?status=open", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Name != "Mailgun" || list.Items[0].Step != "connection" || list.Items[0].Provider != "" {
		t.Fatalf("open setups: %+v", list.Items)
	}
	var st struct {
		Status     string
		Summaries  struct{ Provider, Connection, Domain, Test, Streams string }
		Connection struct {
			Access, RoleARN, Region       string
			ProvisionDone, ProvisionTotal int
			Regions                       []struct{ Label string }
			Sandbox                       *struct{ Region, Website string }
			Template                      *struct{ LaunchURL, Policy string }
		}
		Done *struct {
			Title   string
			Minutes int
			Rows    []struct{ Status, Lead, Text string }
		}
	}
	c.Do("GET", ws+"/connect/setups/"+setupID, nil).MustStatus(t, 200).JSON(t, &st)
	if st.Status != "finished" || st.Summaries.Connection != "us-east-1 · production" || st.Summaries.Domain != "news.acme.io · 7 of 7" || st.Summaries.Test != "delivered in 2.4 s" ||
		st.Summaries.Streams != "Broadcasts, Automations, System" || st.Connection.ProvisionDone != 7 || st.Connection.ProvisionTotal != 7 {
		t.Fatalf("demo setup: %+v %+v", st.Summaries, st.Connection)
	}
	if len(st.Connection.Regions) != 3 || st.Connection.Regions[0].Label != "us-east-1 · N. Virginia · production access, quota 50,000 per day" ||
		st.Connection.Regions[1].Label != "eu-west-1 · Ireland · sandbox" || st.Connection.Sandbox == nil || st.Connection.Sandbox.Region != "eu-west-1" || st.Connection.Sandbox.Website != "https://acme.io" {
		t.Fatalf("regions: %+v %+v", st.Connection.Regions, st.Connection.Sandbox)
	}
	if st.Connection.Template == nil || !strings.Contains(st.Connection.Template.LaunchURL, "quickcreate") || !strings.Contains(st.Connection.Template.Policy, "ses:SendRawEmail") {
		t.Fatalf("template: %+v", st.Connection.Template)
	}
	if st.Done == nil || st.Done.Title != "Amazon SES is connected" || st.Done.Minutes != 9 || len(st.Done.Rows) != 7 {
		t.Fatalf("done: %+v", st.Done)
	}
	if st.Done.Rows[0].Text != " via CloudFormation role `ReputeCastSES`, account 123456789012, region us-east-1 (production, 50,000 per day, 14 per second)" ||
		st.Done.Rows[3].Text != " news.acme.io: DKIM (3 CNAMEs), MAIL FROM bounce.news.acme.io, SPF and DMARC p=quarantine all verified" {
		t.Fatalf("rows: %+v", st.Done.Rows)
	}
	var test struct {
		Timeline []struct{ Key, Title, Detail, Status string }
		Auth     *struct{ Receiver, Header, DKIM, SPF, DMARC string }
	}
	c.Do("GET", ws+"/connect/setups/"+setupID+"/test", nil).MustStatus(t, 200).JSON(t, &test)
	if len(test.Timeline) != 5 || test.Timeline[2].Title != "Sent to Gmail" || test.Timeline[2].Status != "done" || test.Timeline[3].Status != "done" || test.Timeline[4].Status != "waiting" {
		t.Fatalf("demo timeline: %+v", test.Timeline)
	}
	if test.Auth == nil || test.Auth.Receiver != "Gmail" || test.Auth.DKIM != "pass" || test.Auth.SPF != "pass" || test.Auth.DMARC != "pass" ||
		test.Auth.Header != "Authentication-Results: mx.google.com; dkim=pass header.d=news.acme.io; spf=pass smtp.mailfrom=bounce.news.acme.io; dmarc=pass (p=QUARANTINE) header.from=news.acme.io" {
		t.Fatalf("auth: %+v", test.Auth)
	}

	// Settings › Providers.
	var page struct {
		Providers []struct {
			Name, StatusLabel, Streams, IdentityLabel, Identity, ConnectedVia string
			Cost                                                              struct{ Basis string }
		}
		Defaults struct {
			Bounce, Complaint, MailboxComplaint, ProviderError float64
			PauseScope, SuppressionSync                        string
			ConfirmOver                                        int64
			Throttles                                          map[string]float64
		}
		CanManage bool
	}
	c.Do("GET", ws+"/settings/providers", nil).MustStatus(t, 200).JSON(t, &page)
	if len(page.Providers) != 3 || page.Providers[0].Identity != "us-east-1 · 123456789012" || page.Providers[0].ConnectedVia != "CloudFormation role" ||
		page.Providers[2].StatusLabel != "Warming up · day 9" || page.Providers[2].Identity != "mta.acme.io · 203.0.113.25" || !strings.HasPrefix(page.Providers[0].Streams, "Broadcasts 90% · Automations") {
		t.Fatalf("providers: %+v", page.Providers)
	}
	if page.Defaults.Bounce != 2 || page.Defaults.Complaint != 0.1 || page.Defaults.ProviderError != 5 || page.Defaults.ConfirmOver != 5000 ||
		page.Defaults.Throttles["outlook"] != 5 || page.Defaults.SuppressionSync != "15m" || !page.CanManage {
		t.Fatalf("defaults: %+v", page.Defaults)
	}
	c.Do("PUT", ws+"/settings/providers", map[string]any{"bounce": 1.5, "complaint": 0.08, "mailboxComplaint": 0.1, "providerError": 5, "pauseScope": "campaign",
		"confirmOver": 10000, "throttles": map[string]float64{"outlook": 6, "gmail": 0}, "suppressionSync": "1h"}).MustStatus(t, 200).JSON(t, &page)
	if page.Defaults.Bounce != 1.5 || page.Defaults.PauseScope != "campaign" || page.Defaults.Throttles["outlook"] != 6 || page.Defaults.Throttles["yahoo"] != 4 {
		t.Fatalf("saved: %+v", page.Defaults)
	}
	c.Do("PUT", ws+"/settings/providers", map[string]any{"bounce": 1.5, "complaint": 0.08, "mailboxComplaint": 0.1, "providerError": 5, "pauseScope": "route",
		"confirmOver": 10000, "throttles": map[string]float64{"hotmail": 1}, "suppressionSync": "1h"}).MustStatus(t, 422)
	var rot struct {
		Rotated int
		Page    struct {
			Defaults struct{ SecretsRotatedAt *time.Time }
		}
	}
	c.Do("POST", ws+"/settings/providers/rotate-secrets", nil).MustStatus(t, 200).JSON(t, &rot)
	if rot.Rotated != 3 || rot.Page.Defaults.SecretsRotatedAt == nil {
		t.Fatalf("rotation: %+v", rot)
	}
	if audits(t, env, "sending.defaults_update") != 1 || audits(t, env, "provider.webhook_secrets_rotate") != 1 {
		t.Fatal("audit")
	}
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/settings/providers", nil).MustStatus(t, 200)
	v.Do("PUT", ws+"/settings/providers", map[string]any{"bounce": 2}).MustStatus(t, 403)
	v.Do("POST", ws+"/settings/providers/rotate-secrets", nil).MustStatus(t, 403)
}
