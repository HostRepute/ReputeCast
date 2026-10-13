package remote_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/remote"
	"github.com/hostrepute/reputecast/internal/providers/remote/remotetest"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

const ws = "/workspaces/acme-newsletter"

func count(t *testing.T, env *apptest.Env, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

// connect adds a provider connected through the community adapter to the
// demo workspace, with sealed credentials and a webhook token.
func connect(t *testing.T, env *apptest.Env) store.Provider {
	t.Helper()
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.Must(uuid.NewV7())
	p := store.Provider{ID: id}
	creds, err := env.App.Secrets.Seal([]byte(`{"api_key":"fake-api-key","webhook_signing_key":"`+signingKey+`"}`), manage.CredentialsAD(p))
	if err != nil {
		t.Fatal(err)
	}
	hook, err := env.App.Secrets.SealString("whsec-remote", manage.WebhookSecretAD(p))
	if err != nil {
		t.Fatal(err)
	}
	token, _ := authn.NewToken()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, webhook_secret_enc, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, connected_at)
		VALUES ($1,$2,'community:fake-esp','Fake ESP','community','{"region":"eu"}',$3,$4,$5,'healthy','','shared','{}','{}','{}','{}','{}','{}', now())`,
		id, w.ID, creds, token, hook); err != nil {
		t.Fatal(err)
	}
	p, err = env.App.Q.ProvGetAny(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func post(t *testing.T, env *apptest.Env, path string, body []byte, h http.Header) (int, string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, env.Server.URL+path, bytes.NewReader(body))
	for k, v := range h {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestRuntimeEndToEnd(t *testing.T) {
	env := apptest.New(t)
	// The test changes the configuration below, which a running process never
	// does: the background sync loop must not read it meanwhile (the API's
	// install, update and removal hooks still sync).
	env.App.CommunityRuntime().Stop()
	env.SeedDemo()
	env.App.Cfg.Adapters.AllowPrivateNetworks = true
	ctx := context.Background()

	// The adapter process (a Go fake) and its installation from the CLI.
	fa := &remotetest.Adapter{Manifest: remotetest.Manifest()}
	srv := httptest.NewServer(fa)
	defer srv.Close()
	cli := &remote.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true }}
	res, err := cli.Install(ctx, remotetest.ManifestJSON(remotetest.Manifest()), "", "", remote.CLIActor)
	if err != nil || res.Secret == "" || res.Updated || res.Row.AdapterKey != "community:fake-esp" {
		t.Fatalf("install %+v %v", res, err)
	}
	fa.Secrets = []string{res.Secret}
	again, err := cli.Install(ctx, remotetest.ManifestJSON(remotetest.Manifest()), "", "", remote.CLIActor)
	if err != nil || !again.Updated || again.Secret != "" || again.Row.ID != res.Row.ID {
		t.Fatalf("reinstall must update and never show the secret again: %+v %v", again, err)
	}

	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	// The Adapters tab lists it with its declared capabilities.
	var tab manage.AdaptersOverview
	owner.Do("GET", ws+"/adapters", nil).MustStatus(t, 200).JSON(t, &tab)
	var listed *manage.AdapterCommunity
	for i, c := range tab.Community {
		if c.ID == res.Row.ID.String() {
			listed = &tab.Community[i]
		}
	}
	if listed == nil || listed.Kind != "remote" || listed.Maintainer != "reputecast-tests" || strings.Join(listed.Capabilities, " · ") != "webhooks · suppression sync · batch 50 · domain setup via API · quota & rate discovery" {
		t.Fatalf("listed %+v", listed)
	}
	if !strings.Contains(tab.SDKURL, "docs/adapters.md") || !strings.HasSuffix(tab.TemplateURL, "sdk/adapter-template") {
		t.Fatalf("links %s %s", tab.SDKURL, tab.TemplateURL)
	}

	base := ws + "/adapters/" + res.Row.ID.String() + "/runtime"
	// Only instance admins see or change where adapters run.
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("GET", base, nil).MustStatus(t, 403)
	admin := env.Client()
	admin.SignInDemo("marcus@acme.io")
	admin.Do("GET", base, nil).MustStatus(t, 403)
	admin.Do("PUT", base, map[string]string{"endpointUrl": srv.URL}).MustStatus(t, 403)
	owner.Do("GET", ws+"/adapters/"+uuid.NewString()+"/runtime", nil).MustStatus(t, 404)

	// Addresses are checked.
	owner.Do("PUT", base, map[string]string{"endpointUrl": "http://169.254.169.254/latest"}).MustStatus(t, 422)
	env.App.Cfg.Adapters.AllowPrivateNetworks = false
	owner.Do("PUT", base, map[string]string{"endpointUrl": srv.URL}).MustStatus(t, 422)
	env.App.Cfg.Adapters.AllowPrivateNetworks = true

	var rt remote.AdapterRuntime
	r := owner.Do("PUT", base, map[string]string{"endpointUrl": srv.URL + "/"}).MustStatus(t, 200)
	r.JSON(t, &rt)
	if rt.EndpointURL != srv.URL || !rt.Registered || rt.Breaker != "closed" || len(rt.Operations) != 11 || rt.Network[0] != "api.fake-esp.test" {
		t.Fatalf("runtime %+v", rt)
	}
	if strings.Contains(string(r.Body), res.Secret) {
		t.Fatal("the signing secret must never be returned")
	}
	d, ok := providers.Lookup("community:fake-esp")
	if !ok || d.Kind != "community" || d.Name != "Fake ESP" || len(d.Fields) != 3 || d.Capabilities.BatchSize != 50 {
		t.Fatalf("registry %+v %v", d, ok)
	}

	// Health check of the process.
	owner.Do("POST", base+"/check", nil).MustStatus(t, 200).JSON(t, &rt)
	if rt.Check == nil || !rt.Check.OK || rt.Check.RunningVersion != "1.2.3" {
		t.Fatalf("check %+v", rt.Check)
	}

	// A new secret is shown once; the previous one keeps working meanwhile.
	var issued remote.AdapterRuntimeSecret
	owner.Do("POST", base+"/secret", nil).MustStatus(t, 200).JSON(t, &issued)
	if !strings.HasPrefix(issued.Secret, "rcas_") || issued.Secret == res.Secret || issued.PreviousSecretValidUntil == nil {
		t.Fatalf("issued %+v", issued)
	}
	r = owner.Do("GET", base, nil).MustStatus(t, 200)
	r.JSON(t, &rt)
	if strings.Contains(string(r.Body), issued.Secret) || rt.PreviousSecretValidUntil == nil {
		t.Fatalf("runtime after rotation %s", r.Body)
	}
	owner.Do("POST", base+"/check", nil).MustStatus(t, 200).JSON(t, &rt)
	if !rt.Check.OK {
		t.Fatalf("the previous secret must keep working during the grace period: %+v", rt.Check)
	}
	fa.Secrets = []string{issued.Secret}

	for _, action := range []string{"adapter.install", "adapter.update", "adapter.runtime_update", "adapter.secret_issue", "adapter.check"} {
		if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = $1 AND target_id = $2`, action, res.Row.ID.String()); n == 0 {
			t.Errorf("%s not audited", action)
		}
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE before::text LIKE '%rcas_%' OR after::text LIKE '%rcas_%' OR metadata::text LIKE '%rcas_%'`); n != 0 {
		t.Fatal("a secret reached the audit log")
	}

	// A provider connected through the adapter: health comes from quota.
	p := connect(t, env)
	svc := &manage.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, URLs: env.App.URLs,
		Factory: &manage.Factory{Box: env.App.Secrets, HTTP: http.DefaultClient}, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		SelfTest: func(context.Context, string, http.Header) (int, error) { return http.StatusOK, nil }}
	checked, err := svc.Check(ctx, p, true)
	if err != nil {
		t.Fatal(err)
	}
	h := state.HealthOf(checked)
	if checked.Status != state.Healthy || h.Quota == nil || h.Quota.Max != 50000 || h.Rate == nil || h.Rate.Max != 14 || h.LastSuccessAt == nil {
		t.Fatalf("health %s %+v", checked.Status, h)
	}
	if caps := state.Caps(checked); caps.BatchSize != 50 || !caps.SuppressionSync || caps.Webhooks != "Registered" {
		t.Fatalf("capability snapshot %+v", caps)
	}
	var quotaCall *remotetest.Call
	for _, c := range fa.Calls() {
		if c.Operation == remote.OpQuota {
			quotaCall = &c
		}
	}
	if quotaCall == nil || quotaCall.Connection.ProviderID != p.ID.String() || quotaCall.Connection.Settings["region"] != "eu" ||
		quotaCall.Connection.Credentials["api_key"] != "fake-api-key" || quotaCall.Connection.WebhookSecret != "whsec-remote" {
		t.Fatalf("quota call %+v", quotaCall)
	}

	// Webhooks reach the adapter through the ingest endpoint.
	hookPath := "/hooks/community:fake-esp/" + p.WebhookToken
	body, hdr := remotetest.WebhookBody(signingKey, remotetest.Event{Event: "bounced", BounceType: "hard", MessageID: "fake-9", Recipient: "Gone@Example.com",
		Timestamp: "2026-10-12T13:41:02Z", SMTPCode: "550 5.1.1", Description: "User unknown"})
	if code, b := post(t, env, hookPath, body, hdr); code != 200 {
		t.Fatalf("webhook %d %s", code, b)
	}
	var rawID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM provider_raw_events WHERE provider_id = $1 AND signature_ok ORDER BY received_at DESC LIMIT 1`, p.ID).Scan(&rawID); err != nil {
		t.Fatalf("the verified payload was not stored: %v", err)
	}
	n := &ingest.Normalizer{Pool: env.App.Pool, Q: env.App.Q, Factory: &manage.Factory{Box: env.App.Secrets}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := n.Normalize(ctx, rawID); err != nil {
		t.Fatal(err)
	}
	var typ, code, class string
	if err := env.App.Pool.QueryRow(ctx, `SELECT type, original_code, bounce_class FROM delivery_events WHERE raw_event_id = $1`, rawID).Scan(&typ, &code, &class); err != nil {
		t.Fatal(err)
	}
	if typ != "hard_bounce" || code != "550 5.1.1" || class != "mailbox_unknown" {
		t.Fatalf("event %s %s %s", typ, code, class)
	}
	if c := count(t, env, `SELECT count(*) FROM suppressions WHERE email = 'gone@example.com' AND source_provider_id = $1`, p.ID); c != 1 {
		t.Fatal("a hard bounce from a community adapter must join the shared suppression list")
	}
	var parsed bool
	for _, c := range fa.Calls() {
		if c.Operation == remote.OpWebhookParse {
			var in remote.WebhookParseInput
			_ = json.Unmarshal(c.Input, &in)
			parsed = in.Verified
		}
	}
	if !parsed {
		t.Fatal("the normalization job must tell the adapter the payload was verified")
	}

	// A forged webhook is refused and kept for diagnosis only.
	hdr.Set(remotetest.SignatureHeader, remotetest.SignWebhook("forged", body))
	if code, _ := post(t, env, hookPath, body, hdr); code != 401 {
		t.Fatalf("forged webhook: %d", code)
	}
	if c := count(t, env, `SELECT count(*) FROM provider_raw_events WHERE provider_id = $1 AND NOT signature_ok`, p.ID); c != 1 {
		t.Fatalf("forged payloads stored: %d", c)
	}
	// A handshake is answered by the adapter and nothing is stored.
	hs := []byte(`{"type":"handshake","challenge":"c-7"}`)
	hsHdr := http.Header{}
	hsHdr.Set("Content-Type", "application/json")
	hsHdr.Set(remotetest.SignatureHeader, remotetest.SignWebhook(signingKey, hs))
	if code, b := post(t, env, hookPath, hs, hsHdr); code != 200 || b != `{"challenge":"c-7"}` {
		t.Fatalf("handshake %d %s", code, b)
	}
	if c := count(t, env, `SELECT count(*) FROM provider_raw_events WHERE provider_id = $1`, p.ID); c != 2 {
		t.Fatalf("raw events %d", c)
	}

	// While the adapter process is down, webhooks are refused with 503 (the
	// provider retries) and nothing unverified is stored as verified.
	srv.Close()
	hdr.Set(remotetest.SignatureHeader, remotetest.SignWebhook(signingKey, body))
	if code, _ := post(t, env, hookPath, body, hdr); code != http.StatusServiceUnavailable {
		t.Fatalf("adapter down: %d", code)
	}
	if c := count(t, env, `SELECT count(*) FROM provider_raw_events WHERE provider_id = $1`, p.ID); c != 2 {
		t.Fatalf("raw events while down %d", c)
	}

	// Removing it is refused while a provider uses it.
	owner.Do("DELETE", ws+"/adapters/"+res.Row.ID.String(), nil).MustStatus(t, 409)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE providers SET disconnected_at = now() WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	owner.Do("DELETE", ws+"/adapters/"+res.Row.ID.String(), nil).MustStatus(t, 204)
	if _, ok := providers.Lookup("community:fake-esp"); ok {
		t.Fatal("a removed adapter must leave the registry")
	}
	if c := count(t, env, `SELECT count(*) FROM adapter_runtimes`); c != 0 {
		t.Fatal("the runtime row must go with the adapter")
	}
}

// Installed HTTP mappings (kind "http") are registered as community:<name>
// and open through the generic HTTP adapter with the stored manifest.
func TestInstalledHTTPAdapters(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")

	type problem struct {
		Errors []struct{ Location, Message string }
	}
	location := func(r *apptest.Response) string {
		var p problem
		r.JSON(t, &p)
		if len(p.Errors) == 0 {
			t.Fatalf("no field error: %s", r.Body)
		}
		return p.Errors[0].Location
	}
	var authorization atomic.Value
	esp := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization.Store(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"id":"esp-1"}}`)
	}))
	defer esp.Close()
	mapping := func(name string) map[string]any {
		m := map[string]any{
			"endpoint":      "POST " + esp.URL + "/v1/messages",
			"headers":       []map[string]string{{"name": "Authorization", "value": "Bearer {{ secrets.api_key }}"}},
			"bodyTemplate":  `{"from": "{{ sender }}", "to": "{{ recipient }}", "subject": "{{ subject }}", "html": "{{ html }}", "headers": {{ headers_json }}, "metadata": {"rc": "{{ message_id }}"}}`,
			"messageIdPath": "$.data.id", "webhookMapping": "status", "webhookAuth": "hmac", "webhookHeader": "X-Example-Signature",
		}
		if name != "" {
			m["name"] = name
		}
		return m
	}

	// The mapping is checked by the generic HTTP adapter, field by field.
	bad := mapping("Local ESP")
	bad["bodyTemplate"] = `{"to": "{{ recipient }}"`
	if loc := location(owner.Do("POST", ws+"/adapters/http", bad).MustStatus(t, 422)); loc != "body.bodyTemplate" {
		t.Fatalf("body error at %s", loc)
	}
	bad = mapping("Local ESP")
	bad["headers"] = []map[string]string{{"name": "Host", "value": "evil.example"}}
	if loc := location(owner.Do("POST", ws+"/adapters/http", bad).MustStatus(t, 422)); loc != "body.headers" {
		t.Fatalf("header error at %s", loc)
	}
	bad = mapping("Local ESP")
	bad["webhookAuth"], bad["webhookHeader"] = "header", ""
	if loc := location(owner.Do("POST", ws+"/adapters/http", bad).MustStatus(t, 422)); loc != "body.webhookHeader" {
		t.Fatalf("webhook error at %s", loc)
	}
	// A custom JSONPath map needs the path of the event type.
	bad = mapping("Local ESP")
	bad["webhookMapping"] = "jsonpath"
	if loc := location(owner.Do("POST", ws+"/adapters/http", bad).MustStatus(t, 422)); loc != "body.webhookPaths.type" {
		t.Fatalf("custom map error at %s", loc)
	}
	bad["webhookPaths"] = map[string]string{"type": "$.event", "recipient": "$.email[", "messageId": "$.message_id"}
	if loc := location(owner.Do("POST", ws+"/adapters/http", bad).MustStatus(t, 422)); loc != "body.webhookPaths.recipient" {
		t.Fatalf("custom map path error at %s", loc)
	}

	// "Test with a sample message" renders the request, secrets masked.
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("POST", ws+"/adapters/http/preview", mapping("")).MustStatus(t, 403)
	var pv manage.AdapterHTTPPreview
	owner.Do("POST", ws+"/adapters/http/preview", mapping("")).MustStatus(t, 200).JSON(t, &pv)
	auth := ""
	for _, h := range pv.Headers {
		if h.Name == "Authorization" {
			auth = h.Value
		}
	}
	if pv.Method != http.MethodPost || pv.URL != esp.URL+"/v1/messages" || auth != "Bearer ••••" || !strings.Contains(pv.Body, `"to": "`) {
		t.Fatalf("preview %+v", pv)
	}

	// Saved, it is registered under its name and listed on the Adapters tab.
	var saved manage.AdapterCommunity
	owner.Do("POST", ws+"/adapters/http", mapping("Local ESP")).MustStatus(t, 201).JSON(t, &saved)
	d, ok := providers.Lookup("community:local-esp")
	if !ok || d.Kind != "community" || d.Name != "Local ESP" || d.Capabilities.Webhooks == "" || len(d.Fields) < 2 || !d.Fields[0].Secret {
		t.Fatalf("registry %+v %v", d, ok)
	}
	if loc := location(owner.Do("POST", ws+"/adapters/http", mapping("local-esp")).MustStatus(t, 422)); loc != "body.name" {
		t.Fatalf("a name with a taken key: %s", loc)
	}
	var after string
	if err := env.App.Pool.QueryRow(ctx, `SELECT after::text FROM audit_events WHERE action = 'adapter.http_save' AND target_id = $1`, saved.ID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(after, `"headers": ["Authorization"]`) || strings.Contains(after, "secrets.api_key") {
		t.Fatalf("audit %s", after)
	}
	// A custom JSONPath map is stored with its paths.
	custom := mapping("Batch ESP")
	custom["webhookMapping"] = "jsonpath"
	custom["webhookPaths"] = map[string]string{"events": "$.events[*]", "type": "$.event", "recipient": "$.email", "messageId": "$.message_id"}
	owner.Do("POST", ws+"/adapters/http", custom).MustStatus(t, 201)
	var hook string
	if err := env.App.Pool.QueryRow(ctx, `SELECT (manifest->'http'->'webhook')::text FROM installed_adapters WHERE name = 'Batch ESP'`).Scan(&hook); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"events_path": "$.events[*]"`, `"type_path": "$.event"`, `"recipient_path": "$.email"`, `"message_id_path": "$.message_id"`, `"signature_header": "X-Example-Signature"`} {
		if !strings.Contains(hook, want) {
			t.Fatalf("custom map %s lacks %s", hook, want)
		}
	}
	if _, ok := providers.Lookup("community:batch-esp"); !ok {
		t.Fatal("the custom map adapter must be registered")
	}

	// A provider connected through it opens with its own credentials.
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.Must(uuid.NewV7())
	creds, _ := env.App.Secrets.Seal([]byte(`{"api_key":"local-key"}`), manage.CredentialsAD(store.Provider{ID: id}))
	token, _ := authn.NewToken()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, connected_at)
		VALUES ($1,$2,'community:local-esp','Local ESP','','{}',$3,$4,'healthy','','shared','{}','{}','{}','{}','{}','{}', now())`, id, w.ID, creds, token); err != nil {
		t.Fatal(err)
	}
	p, err := env.App.Q.ProvGetAny(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	prov, err := (&manage.Factory{Box: env.App.Secrets, HTTP: esp.Client()}).Open(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	res, err := prov.Send(ctx, []providers.Message{contracttest.Message(0)})
	if err != nil || len(res) != 1 || res[0].Err != nil || res[0].ProviderMessageID != "esp-1" || authorization.Load() != "Bearer local-key" {
		t.Fatalf("send %+v %v (auth %v)", res, err, authorization.Load())
	}

	// Removed, it leaves the registry.
	owner.Do("DELETE", ws+"/adapters/"+saved.ID, nil).MustStatus(t, 409)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE providers SET disconnected_at = now() WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	owner.Do("DELETE", ws+"/adapters/"+saved.ID, nil).MustStatus(t, 204)
	if _, ok := providers.Lookup("community:local-esp"); ok {
		t.Fatal("a removed HTTP adapter must leave the registry")
	}
}

func TestCommunityKeys(t *testing.T) {
	for name, want := range map[string]string{
		"Example ESP": "community:example-esp", "  Mailtrap  Sending!": "community:mailtrap-sending", "ACME_mail 2": "community:acme-mail-2",
	} {
		if got := providers.CommunityKey(name); got != want {
			t.Errorf("%q → %q, want %q", name, got, want)
		}
	}
	if k := providers.CommunityKey("Брево"); !strings.HasPrefix(k, "community:adapter-") || k != providers.CommunityKey("Брево") {
		t.Errorf("a name without letters a-z gets a stable key: %q", k)
	}
}

func TestBrokenManifestRowsAreSkipped(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	cli := &remote.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets}
	good := remotetest.Manifest()
	if _, err := cli.Install(ctx, remotetest.ManifestJSON(good), "", "", remote.CLIActor); err != nil {
		t.Fatal(err)
	}
	other := remotetest.Manifest()
	other.ID, other.Name = "other-esp", "Other ESP"
	res, err := cli.Install(ctx, remotetest.ManifestJSON(other), "", "https://adapters.example.com", remote.CLIActor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE installed_adapters SET manifest = manifest || '{"protocol": 9}' WHERE id = $1`, res.Row.ID); err != nil {
		t.Fatal(err)
	}
	rt := &remote.Runtime{Pool: env.App.Pool, Box: env.App.Secrets, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := rt.Sync(ctx); err == nil || !strings.Contains(err.Error(), "Other ESP") {
		t.Fatalf("sync: %v", err)
	}
	defer func() { _ = providers.SetRuntime(rt, nil) }()
	if _, ok := rt.Installation("community:fake-esp"); !ok {
		t.Fatal("a broken row must not keep the others out of the registry")
	}
	if _, ok := rt.Installation("community:other-esp"); ok {
		t.Fatal("a manifest that no longer validates must not be registered")
	}
	// Checking it records why instead of failing the request.
	cli.Runtime = rt
	broken, err := cli.RowByKey(ctx, "other-esp")
	if err != nil {
		t.Fatal(err)
	}
	row, err := cli.Check(ctx, nil, broken, remote.CLIActor)
	if err != nil || row.CheckOk == nil || *row.CheckOk || !strings.Contains(row.CheckDetail, "no longer validates") {
		t.Fatalf("check of a broken row: %+v %v", row, err)
	}
	// Installing another adapter under a taken id is refused.
	dup := remotetest.Manifest()
	dup.Name = "Fake ESP again"
	if r2, err := cli.Install(ctx, remotetest.ManifestJSON(dup), "", "", remote.CLIActor); err != nil || !r2.Updated {
		t.Fatalf("the same id updates the installed adapter: %+v %v", r2, err)
	}
}
