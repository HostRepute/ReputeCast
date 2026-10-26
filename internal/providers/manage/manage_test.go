package manage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/postmark"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

const ws = "/workspaces/acme-newsletter"

// fake is a test adapter whose health probe fails on demand.
type fake struct{}

var fakeErr struct {
	sync.Mutex
	err error
}

func setFakeErr(err error) {
	fakeErr.Lock()
	fakeErr.err = err
	fakeErr.Unlock()
}

func (fake) Adapter() string { return "faketest" }
func (fake) Capabilities() providers.Capabilities {
	return providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "Registered", BatchSize: 1, QuotaDiscovery: true}
}
func (fake) Send(context.Context, []providers.Message) ([]providers.SendResult, error) {
	return nil, nil
}
func (fake) ParseWebhook(context.Context, *http.Request, []byte) ([]providers.Event, error) {
	return nil, nil
}
func (fake) Quota(context.Context) (providers.Quota, error) {
	fakeErr.Lock()
	defer fakeErr.Unlock()
	if fakeErr.err != nil {
		return providers.Quota{}, fakeErr.err
	}
	return providers.Quota{Kind: "rolling_24h", Max: 1000, Used: 10, Account: "Fake · fine"}, nil
}

func init() {
	providers.Register(providers.Descriptor{Adapter: "faketest", Name: "Fake ESP", Kind: "api", Capabilities: fake{}.Capabilities(),
		Fields: []providers.Field{{Key: "token", Label: "API token", Secret: true, Required: true}},
		New:    func(providers.Config) (providers.Provider, error) { return fake{}, nil }})
}

func service(env *apptest.Env, now func() time.Time) *manage.Service {
	return &manage.Service{
		Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, URLs: env.App.URLs,
		Factory:      &manage.Factory{Box: env.App.Secrets, HTTP: http.DefaultClient},
		Log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		Jobs:         func() *jobs.Client { return env.App.Jobs },
		Sending:      func() sending.Starter { return env.App.Sending },
		Now:          now,
		CheckAddress: func(context.Context, string) (manage.AddressProblem, bool) { return manage.AddressProblem{}, true },
	}
}

func demoWorkspace(t *testing.T, env *apptest.Env) store.Workspace {
	t.Helper()
	w, err := env.App.Q.GetWorkspaceBySlug(context.Background(), "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func providerByAdapter(t *testing.T, env *apptest.Env, adapter string) store.Provider {
	t.Helper()
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM providers WHERE adapter = $1 AND disconnected_at IS NULL ORDER BY connected_at LIMIT 1`, adapter).Scan(&id); err != nil {
		t.Fatalf("provider %s: %v", adapter, err)
	}
	p, err := env.App.Q.ProvGetAny(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// addFake connects the fake adapter to the demo workspace.
func addFake(t *testing.T, env *apptest.Env, w store.Workspace, lastEvent *time.Time) store.Provider {
	t.Helper()
	ctx := context.Background()
	id := uuid.Must(uuid.NewV7())
	p := store.Provider{ID: id}
	creds, err := env.App.Secrets.Seal([]byte(`{"token":"t0ken"}`), manage.CredentialsAD(p))
	if err != nil {
		t.Fatal(err)
	}
	secret, err := env.App.Secrets.SealString("whsec", manage.WebhookSecretAD(p))
	if err != nil {
		t.Fatal(err)
	}
	token, _ := authn.NewToken()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, webhook_secret_enc, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, last_event_at, connected_at)
		VALUES ($1,$2,'faketest','Fake ESP','test','{}',$3,$4,$5,'healthy','','shared','{}','{}','{}','{}','{}','{}',$6, now())`,
		id, w.ID, creds, token, secret, lastEvent); err != nil {
		t.Fatal(err)
	}
	p, err = env.App.Q.ProvGetAny(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// sendFake records n messages handed to a provider ago ago, with no event yet.
func sendFake(t *testing.T, env *apptest.Env, w store.Workspace, p store.Provider, n int, ago time.Duration) {
	t.Helper()
	for i := 0; i < n; i++ {
		id := uuid.Must(uuid.NewV7())
		if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO messages (id, workspace_id, stream, email, provider_id, provider_message_id, status, queued_at, sent_at)
			VALUES ($1, $2, 'broadcast', $3, $4, $5, 'sent', now() - $6::interval, now() - $6::interval)`,
			id, w.ID, "reader."+id.String()[24:]+"@example.com", p.ID, "fake-"+id.String(), ago.String()); err != nil {
			t.Fatal(err)
		}
	}
}

func count(t *testing.T, env *apptest.Env, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func TestProvidersOverview(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	r := owner.Do("GET", ws+"/providers", nil).MustStatus(t, 200)
	var o struct {
		Summary struct {
			Connected     int
			SentThisMonth int64
			Suppressions  int64
			Pushed        int64
		}
		Providers []struct {
			Name, Adapter, Status, Account string
			Bounce, Complaint              struct{ Value *float64 }
			Quota                          struct {
				Kind      string
				Max, Used int64
			}
			Warmup      *struct{ Day, Days int }
			Credentials []struct {
				Key string
				Set bool
			}
			Capabilities []struct{ Label, Kind string }
		}
		Routing []struct {
			Stream   string
			Failover string
		}
	}
	r.JSON(t, &o)
	// With billing@northwind.example's role entry and the Sunset tier's 287
	// invalid addresses and spam traps from its verification (X14).
	if o.Summary.Connected != 3 || o.Summary.Suppressions != 2162 || o.Summary.SentThisMonth == 0 {
		t.Fatalf("summary %+v", o.Summary)
	}
	// Providers get only the hard bounces and complaints that apply to all
	// mail (PR-1): fewer than the shared list holds.
	if o.Summary.Pushed == 0 || o.Summary.Pushed >= o.Summary.Suppressions {
		t.Fatalf("pushed %d of %d", o.Summary.Pushed, o.Summary.Suppressions)
	}
	rates := map[string][2]float64{"ses": {0.6, 0.04}, "postmark": {0.2, 0.01}, "postal": {0.9, 0.05}}
	for _, p := range o.Providers {
		want := rates[p.Adapter]
		if p.Bounce.Value == nil || *p.Bounce.Value != want[0] || p.Complaint.Value == nil || *p.Complaint.Value != want[1] {
			t.Errorf("%s rates = %v / %v, want %v", p.Name, p.Bounce.Value, p.Complaint.Value, want)
		}
		for _, c := range p.Credentials {
			if !c.Set {
				t.Errorf("%s credential %s not set", p.Name, c.Key)
			}
		}
		switch p.Adapter {
		case "ses":
			if p.Quota.Kind != "rolling_24h" || p.Quota.Max != 50000 || p.Account != "Production · no enforcement" {
				t.Errorf("SES card %+v", p)
			}
		case "postal":
			if p.Warmup == nil || p.Warmup.Day != 9 || p.Warmup.Days != 21 || p.Status != state.WarmingUp {
				t.Errorf("Postal warm-up %+v", p.Warmup)
			}
		}
	}
	if len(o.Routing) != 4 || o.Routing[0].Failover != "pause_and_ask" || o.Routing[3].Stream != "transactional" {
		t.Fatalf("routing %+v", o.Routing)
	}
	// Credentials are write-only: no value ever comes back.
	if bytes.Contains(r.Body, []byte("demo-")) || bytes.Contains(r.Body, []byte("sample credentials")) {
		t.Fatal("credential values leaked into the overview")
	}

	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("GET", ws+"/providers", nil).MustStatus(t, 200)
	env.Client().Do("GET", ws+"/providers", nil).MustStatus(t, 401)
}

func TestProviderSettingsAndCredentials(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ses := providerByAdapter(t, env, "ses")
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	path := ws + "/providers/" + ses.ID.String()

	body := map[string]any{"label": "eu-west-1", "pricing": map[string]any{"model": "metered", "per1000": 0.12}, "limits": map[string]any{"bounce": 4, "complaint": 0.08}}
	viewer.Do("PATCH", path, body).MustStatus(t, 403)
	var card struct {
		Label  string
		Bounce struct{ Limit float64 }
		Cost   struct{ Per1000 float64 }
	}
	owner.Do("PATCH", path, body).MustStatus(t, 200).JSON(t, &card)
	if card.Label != "eu-west-1" || card.Bounce.Limit != 4 || card.Cost.Per1000 != 0.12 {
		t.Fatalf("settings not applied: %+v", card)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.update' AND target_id = $1`, ses.ID.String()); n != 1 {
		t.Fatalf("provider.update audit entries = %d", n)
	}

	// Credentials: only declared secret keys, never echoed back.
	owner.Do("PUT", path+"/credentials", map[string]any{"values": map[string]string{"region": "x"}}).MustStatus(t, 422)
	r := owner.Do("PUT", path+"/credentials", map[string]any{"values": map[string]string{"secret_access_key": "AKIA-new-secret-value"}}).MustStatus(t, 200)
	if bytes.Contains(r.Body, []byte("AKIA-new-secret-value")) {
		t.Fatal("new credential echoed back")
	}
	fresh, _ := env.App.Q.ProvGetAny(context.Background(), ses.ID)
	if bytes.Contains(fresh.CredentialsEnc, []byte("AKIA-new-secret-value")) {
		t.Fatal("credentials stored in clear text")
	}
	f := &manage.Factory{Box: env.App.Secrets}
	raw, err := f.Credentials(fresh)
	if err != nil || !bytes.Contains(raw, []byte("AKIA-new-secret-value")) {
		t.Fatalf("credentials not replaced: %s %v", raw, err)
	}
	var meta struct{ Keys []string }
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT metadata->'keys' FROM audit_events WHERE action = 'provider.credentials_replace' AND target_id = $1`, ses.ID.String()).Scan(&meta.Keys); err != nil || len(meta.Keys) != 1 {
		t.Fatalf("credential audit: %v %v", meta.Keys, err)
	}

	// A provider that routes mail can't be disconnected.
	owner.Do("DELETE", path, nil).MustStatus(t, 409)
	viewer.Do("DELETE", path, nil).MustStatus(t, 403)
}

func TestOutageFailsOverAndRecovers(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	fp := addFake(t, env, w, nil)
	pm := providerByAdapter(t, env, "postmark")
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")

	// Automations: the fake provider first, Postmark as automatic fallback.
	var cfg map[string]any
	owner.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &cfg)
	streams := []map[string]any{}
	for _, s := range cfg["streams"].([]any) {
		m := s.(map[string]any)
		conds := []map[string]any{}
		for _, c := range m["conditions"].([]any) {
			cm := c.(map[string]any)
			cond := map[string]any{"field": cm["field"], "values": cm["values"], "providerId": cm["providerId"], "note": cm["note"]}
			if sp, ok := cm["splits"]; ok && sp != nil {
				cond["splits"] = sp
			}
			conds = append(conds, cond)
		}
		in := map[string]any{"stream": m["stream"], "splits": m["splits"], "failover": m["failover"], "conditions": conds}
		if fb, _ := m["fallback"].(string); fb != "" {
			in["fallback"] = fb
		}
		if m["stream"] == "automation" {
			in["splits"] = []map[string]any{{"providerId": fp.ID.String(), "weight": 100}}
			in["fallback"] = pm.ID.String()
		}
		streams = append(streams, in)
	}
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": streams}).MustStatus(t, 200)

	t0 := time.Now().Add(-30 * time.Minute)
	at := t0
	svc := service(env, func() time.Time { return at })
	setFakeErr(&providers.CallError{Op: "GET /quota", Status: 503, Code: "ServiceUnavailable", Temporary: true, Cause: providers.ErrUnavailable})
	for i := 0; i < 2; i++ {
		var err error
		if fp, err = svc.Check(ctx, fp, true); err != nil {
			t.Fatal(err)
		}
	}
	if fp.Status != state.Down {
		t.Fatalf("status after two failed checks = %s", fp.Status)
	}
	var mode string
	var fallback *uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT mode, fallback_provider_id FROM stream_failovers WHERE provider_id = $1 AND stream = 'automation' AND ended_at IS NULL`, fp.ID).Scan(&mode, &fallback); err != nil {
		t.Fatalf("no automation failover: %v", err)
	}
	if mode != "automatic" || fallback == nil || *fallback != pm.ID {
		t.Fatalf("failover %s → %v", mode, fallback)
	}
	if n := count(t, env, `SELECT count(*) FROM incidents WHERE provider_id = $1 AND kind IN ('provider_down','failover')`, fp.ID); n != 2 {
		t.Fatalf("incidents = %d, want down + failover", n)
	}
	if n := count(t, env, `SELECT count(*) FROM notifications WHERE kind = 'provider_down' AND workspace_id = $1`, w.ID); n == 0 {
		t.Fatal("nobody was told the provider is down")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.down' AND target_id = $1 AND actor_type = 'system'`, fp.ID.String()); n != 1 {
		t.Fatalf("provider.down audit = %d", n)
	}
	// The overview reports the outage with the automation stream failed over.
	var o struct {
		Outage *struct {
			ProviderID string
			Streams    []struct{ Stream, State, FallbackShort string }
		}
	}
	owner.Do("GET", ws+"/providers", nil).MustStatus(t, 200).JSON(t, &o)
	if o.Outage == nil || o.Outage.ProviderID != fp.ID.String() {
		t.Fatalf("overview outage = %+v", o.Outage)
	}
	foundAuto := false
	for _, s := range o.Outage.Streams {
		if s.Stream == "automation" && s.State == "failover_active" && s.FallbackShort == "Postmark" {
			foundAuto = true
		}
	}
	if !foundAuto {
		t.Fatalf("automation stream not failed over: %+v", o.Outage.Streams)
	}

	// Healthy again: recovering, then back after ten healthy minutes.
	setFakeErr(nil)
	at = t0.Add(5 * time.Minute)
	var err error
	if fp, err = svc.Check(ctx, fp, true); err != nil {
		t.Fatal(err)
	}
	if fp.Status != state.Down {
		t.Fatalf("recovered too early: %s", fp.Status)
	}
	at = t0.Add(16 * time.Minute)
	if fp, err = svc.Check(ctx, fp, true); err != nil {
		t.Fatal(err)
	}
	if fp.Status != state.Healthy {
		t.Fatalf("status after ten healthy minutes = %s", fp.Status)
	}
	if n := count(t, env, `SELECT count(*) FROM stream_failovers WHERE provider_id = $1 AND ended_at IS NULL`, fp.ID); n != 0 {
		t.Fatalf("%d failovers still active", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.recovered' AND target_id = $1`, fp.ID.String()); n != 1 {
		t.Fatalf("provider.recovered audit = %d", n)
	}
}

func TestRejectedCredentialsMarkAuthFailure(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	fp := addFake(t, env, w, nil)
	svc := service(env, time.Now)
	setFakeErr(&providers.CallError{Op: "GET /quota", Status: 401, Code: "Unauthorized", Message: "Bad token", Auth: true, Cause: providers.ErrUnauthorized})
	defer setFakeErr(nil)
	fp, err := svc.Check(ctx, fp, true)
	if err != nil {
		t.Fatal(err)
	}
	h := state.HealthOf(fp)
	if fp.Status != state.Down || h.Down == nil || h.Down.Reason != "auth" {
		t.Fatalf("status %s, down %+v", fp.Status, h.Down)
	}
}

func TestWebhooksSilentDegrades(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	last := time.Now().Add(-7 * time.Hour)
	fp := addFake(t, env, w, &last)
	// Mail went out after the last event, an hour ago: overdue.
	sendFake(t, env, w, fp, manage.SilentMinOverdue, time.Hour)
	svc := service(env, time.Now)
	setFakeErr(nil)
	fp, err := svc.Check(ctx, fp, true)
	if err != nil {
		t.Fatal(err)
	}
	if fp.Status != state.Degraded {
		t.Fatalf("status = %s, want degraded", fp.Status)
	}
	if n := count(t, env, `SELECT count(*) FROM notifications WHERE kind = 'webhooks_silent' AND workspace_id = $1`, w.ID); n == 0 {
		t.Fatal("nobody was told the webhooks went quiet")
	}
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var o struct {
		Silent *struct {
			ProviderID string
			Accepted   int64
			Overdue    int64
			Checks     []struct{ Key string }
			Awaiting   []struct{ Status string }
		}
	}
	owner.Do("GET", ws+"/providers", nil).MustStatus(t, 200).JSON(t, &o)
	if o.Silent == nil || o.Silent.ProviderID != fp.ID.String() || o.Silent.Accepted != manage.SilentMinOverdue || o.Silent.Overdue != manage.SilentMinOverdue ||
		len(o.Silent.Checks) == 0 || len(o.Silent.Awaiting) == 0 {
		t.Fatalf("silent = %+v", o.Silent)
	}
}

// PR-6: quiet webhooks are "silent" only with enough mail past its 30-minute
// grace and no event for 6 hours. A quiet night followed by a welcome email
// is no alarm, nor is a burst still inside its grace.
func TestNoSilentAlarmAfterAQuietPeriod(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	last := time.Now().Add(-7 * time.Hour)
	fp := addFake(t, env, w, &last)
	setFakeErr(nil)
	check := func(at time.Time) store.Provider {
		t.Helper()
		p, err := service(env, func() time.Time { return at }).Check(ctx, fp, true)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	// One welcome email a minute ago, after seven quiet hours.
	sendFake(t, env, w, fp, 1, time.Minute)
	if p := check(time.Now()); p.Status != state.Healthy {
		t.Fatalf("one email after a quiet period: %s %s", p.Status, p.StatusDetail)
	}
	// A burst still inside its grace.
	sendFake(t, env, w, fp, 40, 5*time.Minute)
	if p := check(time.Now()); p.Status != state.Healthy {
		t.Fatalf("mail inside its 30-minute grace: %s", p.Status)
	}
	if n := count(t, env, `SELECT count(*) FROM incidents WHERE kind = 'webhooks_silent' AND provider_id = $1`, fp.ID); n != 0 {
		t.Fatalf("%d silent alarms", n)
	}
	// Forty minutes on, the burst is overdue: now it's silent.
	if p := check(time.Now().Add(40 * time.Minute)); p.Status != state.Degraded || p.StatusDetail != state.DetailWebhooksSilent {
		t.Fatalf("overdue mail: %s %s", p.Status, p.StatusDetail)
	}

	// Below the minimum, a few overdue messages are no alarm.
	few := addFake(t, env, w, &last)
	sendFake(t, env, w, few, manage.SilentMinOverdue-1, 2*time.Hour)
	p, err := service(env, time.Now).Check(ctx, few, true)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != state.Healthy {
		t.Fatalf("%d overdue messages: %s", manage.SilentMinOverdue-1, p.Status)
	}
}

// fakePoll is a provider whose events come by polling its events API.
type fakePoll struct{ fake }

func (fakePoll) Adapter() string { return "fakepoll" }
func (fakePoll) Backfill(_ context.Context, ids []string) ([]providers.Event, error) {
	out := make([]providers.Event, 0, len(ids))
	for _, id := range ids {
		out = append(out, providers.Event{Type: providers.EventDelivered, ProviderMessageID: id, OccurredAt: time.Now(), OriginalType: "Delivered", Raw: json.RawMessage(`{}`)})
	}
	return out, nil
}

func init() {
	providers.Register(providers.Descriptor{Adapter: "fakepoll", Name: "Fake polling ESP", Kind: "api", Capabilities: fake{}.Capabilities(),
		Fields: []providers.Field{{Key: "token", Label: "API token", Secret: true, Required: true}},
		New:    func(providers.Config) (providers.Provider, error) { return fakePoll{}, nil }})
}

// PR-6: a provider whose events come by polling gets its freshness from the
// poll, so it never reads as silent while polling works.
func TestPollingUpdatesEventFreshness(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	last := time.Now().Add(-7 * time.Hour)
	fp := addFake(t, env, w, &last)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE providers SET adapter = 'fakepoll', config = '{"events":"polling"}' WHERE id = $1`, fp.ID); err != nil {
		t.Fatal(err)
	}
	sendFake(t, env, w, fp, 20, time.Hour)
	n := &ingest.Normalizer{Pool: env.App.Pool, Q: env.App.Q, Factory: &manage.Factory{Box: env.App.Secrets}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	got, err := n.Backfill(ctx, fp.ID)
	if err != nil || got != 20 {
		t.Fatalf("backfill %d %v", got, err)
	}
	p, err := env.App.Q.ProvGetAny(ctx, fp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.LastEventAt == nil || time.Since(*p.LastEventAt) > time.Minute {
		t.Fatalf("polling didn't update freshness: %v", p.LastEventAt)
	}
	setFakeErr(nil)
	if p, err = service(env, time.Now).Check(ctx, p, true); err != nil || p.Status != state.Healthy {
		t.Fatalf("polled provider %s %v", p.Status, err)
	}
}

func TestPostmarkWebhookIngest(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	pm := providerByAdapter(t, env, "postmark")
	f := &manage.Factory{Box: env.App.Secrets}
	secret := f.WebhookSecret(pm)
	if secret == "" {
		t.Fatal("demo Postmark has no webhook secret")
	}
	// A message ReputeCast sent through Postmark.
	msgID := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, email, provider_id, provider_message_id, status, queued_at, sent_at)
		VALUES ($1, $2, 'system', 'john@example.com', $3, '9a3f1c00-6105-42a2-a16a-77a8eac79483', 'sent', now(), now())`, msgID, w.ID, pm.ID); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile("../postmark/testdata/bounce-hard.json")
	if err != nil {
		t.Fatal(err)
	}
	payload = bytes.Replace(payload, []byte("0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), []byte(msgID.String()), 1)
	url := env.Server.URL + "/hooks/postmark/" + pm.WebhookToken

	post := func(user, pass string) int {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		if user != "" {
			req.SetBasicAuth(user, pass)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if code := post(postmark.WebhookUser, "wrong"); code != http.StatusUnauthorized {
		t.Fatalf("bad secret answered %d", code)
	}
	if code := post(postmark.WebhookUser, secret); code != http.StatusOK {
		t.Fatalf("webhook answered %d", code)
	}
	// Unknown token: nothing stored.
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, env.Server.URL+"/hooks/postmark/nope", strings.NewReader("{}"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown token answered %d", resp.StatusCode)
	}

	var rawID uuid.UUID
	var verified bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT id, signature_ok FROM provider_raw_events WHERE provider_id = $1 AND signature_ok ORDER BY received_at DESC LIMIT 1`, pm.ID).Scan(&rawID, &verified); err != nil {
		t.Fatalf("raw event not stored: %v", err)
	}
	if n := count(t, env, `SELECT count(*) FROM provider_raw_events WHERE provider_id = $1 AND NOT signature_ok`, pm.ID); n != 1 {
		t.Fatalf("unverified payloads kept = %d", n)
	}
	// The River job normalizes it; run it here.
	n := &ingest.Normalizer{Pool: env.App.Pool, Q: env.App.Q, Factory: f, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := n.Normalize(ctx, rawID); err != nil {
		t.Fatal(err)
	}
	if err := n.Normalize(ctx, rawID); err != nil { // redelivery is a no-op
		t.Fatal(err)
	}
	if c := count(t, env, `SELECT count(*) FROM delivery_events WHERE provider_id = $1 AND type = 'hard_bounce' AND message_id = $2`, pm.ID, msgID); c != 1 {
		t.Fatalf("delivery events = %d", c)
	}
	var status string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status FROM messages WHERE id = $1`, msgID).Scan(&status); err != nil || status != "bounced" {
		t.Fatalf("message status %q %v", status, err)
	}
	if c := count(t, env, `SELECT count(*) FROM suppressions WHERE workspace_id = $1 AND email = 'john@example.com' AND reason = 'hard_bounce'`, w.ID); c != 1 {
		t.Fatal("hard bounce not suppressed")
	}
}

func TestEventLogAndExport(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var log struct {
		Items []struct{ Type, Recipient, Provider string }
		Total int
	}
	owner.Do("GET", ws+"/provider-events", nil).MustStatus(t, 200).JSON(t, &log)
	// The nine events of the board are the newest; the audience demo adds
	// older deliveries for the contact timelines, and the reports demo the
	// bounces of the sample campaigns.
	if log.Total < 9 || len(log.Items) < 9 {
		t.Fatalf("event log total %d", log.Total)
	}
	for _, e := range log.Items {
		if !strings.Contains(e.Recipient, "···") {
			t.Fatalf("recipient not masked: %s", e.Recipient)
		}
	}
	owner.Do("GET", ws+"/provider-events?type=hard_bounce", nil).MustStatus(t, 200).JSON(t, &log)
	if log.Total < 3 {
		t.Fatalf("hard bounces = %d", log.Total)
	}
	r := owner.Do("GET", ws+"/provider-events/export", nil).MustStatus(t, 200)
	if !strings.Contains(string(r.Body), "r.wong@outlook.com") {
		t.Fatalf("owner export should carry full addresses: %s", r.Body)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider_events.export'`); n != 1 {
		t.Fatalf("export audit = %d", n)
	}
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	vr := viewer.Do("GET", ws+"/provider-events/export", nil)
	if vr.Status == 200 && strings.Contains(string(vr.Body), "r.wong@outlook.com") {
		t.Fatal("a viewer without subscribers:export got full addresses")
	}
}

func TestSuppressionSyncAndTracking(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	var st struct {
		Total int64
		Rows  []struct {
			Provider, Direction, Status string
			Pulled                      int64
		}
	}
	owner.Do("GET", ws+"/suppression-sync", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Total != 2162 || len(st.Rows) != 4 { // with the Sunset tier's 287 (X14)
		t.Fatalf("sync status %+v", st)
	}
	for _, r := range st.Rows {
		if r.Provider == "Postal" && (r.Direction != "local" || r.Pulled != 41) {
			t.Fatalf("Postal row %+v", r)
		}
	}
	viewer.Do("POST", ws+"/suppression-sync/run", nil).MustStatus(t, 403)
	owner.Do("POST", ws+"/suppression-sync/run", nil).MustStatus(t, 202)

	var tc struct {
		Items []struct {
			ProviderID, Provider, Status, Mode string
		}
	}
	owner.Do("GET", ws+"/tracking-checks", nil).MustStatus(t, 200).JSON(t, &tc)
	var pmID string
	for _, i := range tc.Items {
		if i.Provider == "Postmark" {
			pmID = i.ProviderID
			if i.Status != "warn" {
				t.Fatalf("Postmark tracking %+v", i)
			}
		}
	}
	viewer.Do("POST", ws+"/providers/"+pmID+"/tracking", map[string]string{"action": "keep_provider"}).MustStatus(t, 403)
	var after struct{ Mode string }
	owner.Do("POST", ws+"/providers/"+pmID+"/tracking", map[string]string{"action": "keep_provider"}).MustStatus(t, 200).JSON(t, &after)
	if after.Mode != "provider" {
		t.Fatalf("mode = %s", after.Mode)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action LIKE 'provider.tracking%' AND target_id = $1`, pmID); n != 1 {
		t.Fatalf("tracking audit = %d", n)
	}
}

func TestAdapters(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var a struct {
		Builtin   []struct{ Adapter string }
		Community []struct {
			ID, Name, Version, Update, Latest string
		}
		CanInstall bool
	}
	owner.Do("GET", ws+"/adapters", nil).MustStatus(t, 200).JSON(t, &a)
	if !a.CanInstall || len(a.Community) != 3 {
		t.Fatalf("adapters %+v", a)
	}
	got := map[string]bool{}
	for _, b := range a.Builtin {
		got[b.Adapter] = true
	}
	for _, want := range []string{"ses", "postmark", "postal", "smtp"} {
		if !got[want] {
			t.Errorf("built-in %s missing", want)
		}
	}
	if got["faketest"] {
		// The test adapter registers itself; it is listed like any other.
		t.Log("test adapter listed")
	}
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	for _, c := range a.Community {
		switch c.Update {
		case "update_available":
			viewer.Do("POST", ws+"/adapters/"+c.ID+"/update", nil).MustStatus(t, 403)
			var up struct{ Version, Update string }
			owner.Do("POST", ws+"/adapters/"+c.ID+"/update", nil).MustStatus(t, 200).JSON(t, &up)
			if up.Version != c.Latest || up.Update != "up_to_date" {
				t.Fatalf("after update %+v", up)
			}
		case "unmaintained":
			viewer.Do("DELETE", ws+"/adapters/"+c.ID, nil).MustStatus(t, 403)
			owner.Do("DELETE", ws+"/adapters/"+c.ID, nil).MustStatus(t, 204)
		}
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action IN ('adapter.update','adapter.remove')`); n != 2 {
		t.Fatalf("adapter audit = %d", n)
	}
	// Manifests must be public https URLs.
	owner.Do("POST", ws+"/adapters", map[string]string{"manifestUrl": "https://127.0.0.1/manifest.json"}).MustStatus(t, 422)
	owner.Do("POST", ws+"/adapters/http", map[string]any{"name": "Example ESP", "endpoint": "GET http://x", "bodyTemplate": "{}", "messageIdPath": "$.id", "webhookMapping": "status"}).MustStatus(t, 422)
	var saved struct{ Name, Kind string }
	owner.Do("POST", ws+"/adapters/http", map[string]any{"name": "Example ESP", "endpoint": "POST https://api.example-esp.com/v1/messages",
		"bodyTemplate": `{"to": "{{ recipient }}", "headers": {{ headers_json }}}`, "messageIdPath": "$.data.id", "webhookMapping": "status"}).MustStatus(t, 201).JSON(t, &saved)
	if saved.Kind != "http" {
		t.Fatalf("saved %+v", saved)
	}
}

func TestHeldBroadcastDecision(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	ses := providerByAdapter(t, env, "ses")
	var camp uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM campaigns WHERE workspace_id = $1 AND name = 'Weekly Digest #142'`, w.ID).Scan(&camp); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET status = 'sending', started_at = now() WHERE id = $1`, camp); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO campaign_routes (campaign_id, provider_id, planned, status, status_detail) VALUES ($1, $2, 28444, 'held', '{}')
		ON CONFLICT (campaign_id, provider_id) DO UPDATE SET status = 'held', planned = 28444`, camp, ses.ID); err != nil {
		t.Fatal(err)
	}
	path := ws + "/providers/" + ses.ID.String() + "/held/" + camp.String()
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("POST", path, map[string]string{"action": "wait"}).MustStatus(t, 403)
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	owner.Do("POST", path, map[string]string{"action": "teleport"}).MustStatus(t, 422)
	owner.Do("POST", path, map[string]string{"action": "wait"}).MustStatus(t, 200)
	var detail string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status_detail FROM campaign_routes WHERE campaign_id = $1 AND provider_id = $2`, camp, ses.ID).Scan(&detail); err != nil {
		t.Fatal(err)
	}
	var d struct{ Decision string }
	_ = json.Unmarshal([]byte(detail), &d)
	if d.Decision != "wait" {
		t.Fatalf("decision = %q", detail)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action LIKE 'campaign.%held%' OR action LIKE 'provider.held%'`); n == 0 {
		t.Fatal("decision not audited")
	}
}

// The CloudEvents validation handshake (Azure Event Grid) is confirmed for a
// known webhook URL only.
func TestWebhookValidationHandshake(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pm := providerByAdapter(t, env, "postmark")
	// options answers the status and the allowed origin of a handshake.
	options := func(path, origin string) (int, string) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodOptions, env.Server.URL+path, nil)
		if origin != "" {
			req.Header.Set("WebHook-Request-Origin", origin)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode, resp.Header.Get("WebHook-Allowed-Origin")
	}
	if code, allowed := options("/hooks/postmark/"+pm.WebhookToken, "eventgrid.azure.net"); code != http.StatusOK || allowed != "eventgrid.azure.net" {
		t.Fatalf("handshake answered %d, origin %q", code, allowed)
	}
	if code, _ := options("/hooks/postmark/nope-nope-nope-nope", "eventgrid.azure.net"); code != http.StatusNotFound {
		t.Fatalf("unknown token answered %d", code)
	}
	if code, _ := options("/hooks/postmark/"+pm.WebhookToken, "bad origin/path"); code != http.StatusNotFound {
		t.Fatalf("invalid origin answered %d", code)
	}
}

// PR-7: the health job vets a fallback before any mail moves to it. A
// fallback in sandbox mode can't take Automations, so the stream waits on
// the down provider and the outage card says which check failed.
func TestFailoverVetsTheFallback(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	fp := addFake(t, env, w, nil)
	pm := providerByAdapter(t, env, "postmark")
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var cfg struct {
		Streams []struct {
			Stream, Version string
		}
	}
	owner.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &cfg)
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": []map[string]any{{"stream": "automation", "failover": "automatic", "conditions": []any{},
		"splits": []map[string]any{{"providerId": fp.ID.String(), "weight": 100}}, "fallback": pm.ID.String(), "baseVersion": cfg.Streams[1].Version}}}).MustStatus(t, 200)
	// Postmark's account can only send to verified addresses for now.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE providers SET health = jsonb_set(health, '{sandbox}', 'true') WHERE id = $1`, pm.ID); err != nil {
		t.Fatal(err)
	}
	svc := service(env, time.Now)
	setFakeErr(&providers.CallError{Op: "GET /quota", Status: 503, Code: "ServiceUnavailable", Temporary: true, Cause: providers.ErrUnavailable})
	defer setFakeErr(nil)
	for i := 0; i < 2; i++ {
		var err error
		if fp, err = svc.Check(ctx, fp, true); err != nil {
			t.Fatal(err)
		}
	}
	var mode string
	var fallback *uuid.UUID
	var detail []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT mode, fallback_provider_id, detail FROM stream_failovers WHERE provider_id = $1 AND stream = 'automation' AND ended_at IS NULL`,
		fp.ID).Scan(&mode, &fallback, &detail); err != nil {
		t.Fatal(err)
	}
	var d struct {
		Checks []struct{ Key, Result string }
	}
	_ = json.Unmarshal(detail, &d)
	sandboxFailed := false
	for _, c := range d.Checks {
		sandboxFailed = sandboxFailed || c.Key == "sandbox" && c.Result == "fail"
	}
	if mode != "held" || fallback != nil || !sandboxFailed {
		t.Fatalf("failover %s → %v, detail %s", mode, fallback, detail)
	}
	if n := count(t, env, `SELECT count(*) FROM incidents WHERE provider_id = $1 AND kind = 'fallback_failed'`, fp.ID); n != 1 {
		t.Fatalf("fallback_failed incidents = %d", n)
	}
	var o struct {
		Outage *struct {
			Streams []struct {
				Stream, State  string
				FallbackChecks []struct{ Key, Result, Text string }
			}
		}
	}
	owner.Do("GET", ws+"/providers", nil).MustStatus(t, 200).JSON(t, &o)
	if o.Outage == nil {
		t.Fatal("no outage on the overview")
	}
	for _, s := range o.Outage.Streams {
		if s.Stream != "automation" {
			continue
		}
		failed := false
		for _, c := range s.FallbackChecks {
			if c.Key == "sandbox" && c.Result == "fail" && strings.Contains(c.Text, "sandbox") {
				failed = true
			}
		}
		if s.State == "failover_active" || !failed {
			t.Fatalf("automation on the outage card: %+v", s)
		}
	}
}

// PR-7: messages a provider refuses for good are watched: a run of them
// marks it "Rejecting messages", opens an incident and tells the team;
// once it accepts mail again the incident closes.
func TestRejectionsAreWatched(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	fp := addFake(t, env, w, nil)
	setFakeErr(nil)
	now := time.Now()
	results := make([]providers.SendResult, 0, 20)
	for i := 0; i < 15; i++ {
		results = append(results, providers.SendResult{MessageID: uuid.Must(uuid.NewV7()), Err: &providers.SendError{Code: "MessageRejected", Message: "Email address is not verified."}})
	}
	for i := 0; i < 5; i++ {
		results = append(results, providers.SendResult{MessageID: uuid.Must(uuid.NewV7()), ProviderMessageID: "ok"})
	}
	if err := manage.RecordSend(ctx, env.App.Pool, fp, results, nil, now); err != nil {
		t.Fatal(err)
	}
	fp, err := service(env, time.Now).Check(ctx, fp, true)
	if err != nil {
		t.Fatal(err)
	}
	if fp.Status != state.Degraded || fp.StatusDetail != state.DetailRejecting {
		t.Fatalf("status %s %q", fp.Status, fp.StatusDetail)
	}
	if h := state.HealthOf(fp); h.Rejecting == nil || h.Rejecting.Rejected != 15 || h.Rejecting.Total != 20 || h.Rejecting.Code != "MessageRejected" {
		t.Fatalf("rejecting %+v", h.Rejecting)
	}
	if n := count(t, env, `SELECT count(*) FROM incidents WHERE provider_id = $1 AND kind = 'provider_rejecting' AND resolved_at IS NULL`, fp.ID); n != 1 {
		t.Fatalf("rejecting incidents = %d", n)
	}
	if n := count(t, env, `SELECT count(*) FROM notifications WHERE kind = 'provider_degraded' AND workspace_id = $1 AND data->>'reason' = 'rejecting'`, w.ID); n == 0 {
		t.Fatal("nobody was told")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.rejecting' AND target_id = $1`, fp.ID.String()); n != 1 {
		t.Fatalf("provider.rejecting audit = %d", n)
	}
	// A quarter of an hour later the window is clean.
	later := now.Add(15 * time.Minute)
	if fp, err = service(env, func() time.Time { return later }).Check(ctx, fp, true); err != nil {
		t.Fatal(err)
	}
	if fp.Status != state.Healthy || state.HealthOf(fp).Rejecting != nil {
		t.Fatalf("after: %s", fp.Status)
	}
	if n := count(t, env, `SELECT count(*) FROM incidents WHERE provider_id = $1 AND kind = 'provider_rejecting' AND resolved_at IS NULL`, fp.ID); n != 0 {
		t.Fatalf("open rejecting incidents = %d", n)
	}
}

// PR-10: a bounce mailbox that can't be read reaches the provider's health:
// after 15 minutes of failed reads the provider is Degraded, an incident
// opens and the team is told; the first good read closes it.
func TestBounceMailboxErrorsReachHealth(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	fp := addFake(t, env, w, nil)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE providers SET config = '{"bounce_mailbox":{"address":"bounces@news.acme.io","host":"imap.acme.io"}}' WHERE id = $1`, fp.ID); err != nil {
		t.Fatal(err)
	}
	t0 := time.Now()
	at := t0
	svc := service(env, func() time.Time { return at })
	authErr := errors.New("the bounce mailbox rejected the user name or password")
	svc.RecordMailboxPoll(ctx, fp, authErr)
	if n := count(t, env, `SELECT count(*) FROM incidents WHERE provider_id = $1 AND kind = 'bounce_mailbox_failing'`, fp.ID); n != 0 {
		t.Fatal("one failed read is no alarm")
	}
	at = t0.Add(16 * time.Minute)
	svc.RecordMailboxPoll(ctx, fp, authErr)
	p, err := env.App.Q.ProvGetAny(ctx, fp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != state.Degraded || p.StatusDetail != state.DetailMailboxFailing {
		t.Fatalf("status %s %q", p.Status, p.StatusDetail)
	}
	if h := state.HealthOf(p); h.BounceMailbox == nil || h.BounceMailbox.Error != authErr.Error() {
		t.Fatalf("mailbox state %+v", h.BounceMailbox)
	}
	setFakeErr(nil)
	if p, err = svc.Check(ctx, p, true); err != nil || p.Status != state.Degraded {
		t.Fatalf("the health check keeps it degraded: %s %v", p.Status, err)
	}
	if n := count(t, env, `SELECT count(*) FROM incidents WHERE provider_id = $1 AND kind = 'bounce_mailbox_failing' AND resolved_at IS NULL`, fp.ID); n != 1 {
		t.Fatalf("mailbox incidents = %d", n)
	}
	if n := count(t, env, `SELECT count(*) FROM notifications WHERE workspace_id = $1 AND data->>'reason' = 'bounce_mailbox'`, w.ID); n == 0 {
		t.Fatal("nobody was told")
	}
	at = t0.Add(20 * time.Minute)
	svc.RecordMailboxPoll(ctx, p, nil)
	if p, err = env.App.Q.ProvGetAny(ctx, fp.ID); err != nil || p.Status != state.Healthy {
		t.Fatalf("after a good read: %s %v", p.Status, err)
	}
	if n := count(t, env, `SELECT count(*) FROM incidents WHERE provider_id = $1 AND kind = 'bounce_mailbox_failing' AND resolved_at IS NULL`, fp.ID); n != 0 {
		t.Fatalf("open mailbox incidents = %d", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('provider.bounce_mailbox_failing', 'provider.bounce_mailbox_ok')`, fp.ID.String()); n != 2 {
		t.Fatalf("mailbox audit = %d", n)
	}
}
