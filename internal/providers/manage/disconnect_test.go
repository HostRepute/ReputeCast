package manage_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// fakeRemove registers and removes its webhook, and can refuse to.
type fakeRemove struct{}

var removed struct {
	sync.Mutex
	urls []string
	err  error
}

func (fakeRemove) Adapter() string { return "fakeremove" }
func (fakeRemove) Capabilities() providers.Capabilities {
	return providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "Registered", BatchSize: 1, RemovesWebhook: true}
}
func (fakeRemove) Send(context.Context, []providers.Message) ([]providers.SendResult, error) {
	return nil, nil
}
func (fakeRemove) ParseWebhook(context.Context, *http.Request, []byte) ([]providers.Event, error) {
	return nil, nil
}
func (fakeRemove) RegisterWebhook(context.Context, string) error { return nil }
func (fakeRemove) UnregisterWebhook(_ context.Context, url string) error {
	removed.Lock()
	defer removed.Unlock()
	if removed.err != nil {
		return removed.err
	}
	removed.urls = append(removed.urls, url)
	return nil
}

func init() {
	providers.Register(providers.Descriptor{Adapter: "fakeremove", Name: "Fake Remove", Kind: "api", Capabilities: fakeRemove{}.Capabilities(),
		Fields: []providers.Field{{Key: "token", Label: "API token", Secret: true, Required: true}},
		New:    func(providers.Config) (providers.Provider, error) { return fakeRemove{}, nil }})
}

// insertProvider adds a provider no route uses.
func insertProvider(t *testing.T, env *apptest.Env, w store.Workspace, adapter, config string) store.Provider {
	t.Helper()
	ctx := context.Background()
	id := uuid.Must(uuid.NewV7())
	p := store.Provider{ID: id}
	creds, err := env.App.Secrets.Seal([]byte(`{"token":"t0ken"}`), manage.CredentialsAD(p))
	if err != nil {
		t.Fatal(err)
	}
	token, _ := authn.NewToken()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, connected_at)
		VALUES ($1,$2,$3,$3,'test',$4,$5,$6,'healthy','','shared','{}','{}','{}','{}','{}','{}', now())`, id, w.ID, adapter, config, creds, token); err != nil {
		t.Fatal(err)
	}
	if p, err = env.App.Q.ProvGetAny(ctx, id); err != nil {
		t.Fatal(err)
	}
	return p
}

// Removing a provider removes the webhook ReputeCast registered there, or
// says it has to be removed by hand.
func TestDisconnectRemovesTheWebhook(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	w := demoWorkspace(t, env)
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	type result struct{ Webhook, URL, Detail string }
	disconnect := func(p store.Provider) result {
		t.Helper()
		var r result
		owner.Do("DELETE", ws+"/providers/"+p.ID.String(), nil).MustStatus(t, 200).JSON(t, &r)
		if n := count(t, env, `SELECT count(*) FROM providers WHERE id = $1 AND disconnected_at IS NOT NULL`, p.ID); n != 1 {
			t.Fatalf("%s is still connected", p.Adapter)
		}
		return r
	}

	p := insertProvider(t, env, w, "fakeremove", "{}")
	if r := disconnect(p); r.Webhook != "removed" || r.URL != manage.WebhookURL(env.App.URLs, p) {
		t.Fatalf("removing its webhook: %+v", r)
	}
	removed.Lock()
	got := append([]string(nil), removed.urls...)
	removed.Unlock()
	if len(got) != 1 || got[0] != manage.WebhookURL(env.App.URLs, p) {
		t.Fatalf("the provider was asked to remove %v", got)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.disconnect' AND target_id = $1 AND metadata->>'webhook' = 'removed'`, p.ID.String()); n != 1 {
		t.Fatal("the disconnect isn't audited with its webhook")
	}

	// The provider refuses: the provider is removed anyway, and the answer says why.
	removed.Lock()
	removed.err = errors.New("401 unauthorized")
	removed.Unlock()
	if r := disconnect(insertProvider(t, env, w, "fakeremove", "{}")); r.Webhook != "failed" || r.Detail != "401 unauthorized" {
		t.Fatalf("a refused removal: %+v", r)
	}
	removed.Lock()
	removed.err = nil
	removed.Unlock()

	// An adapter that can't remove its webhook: by hand.
	if r := disconnect(insertProvider(t, env, w, "faketest", "{}")); r.Webhook != "manual" {
		t.Fatalf("an adapter without removal: %+v", r)
	}

	// Demo providers are never called.
	removed.Lock()
	before := len(removed.urls)
	removed.Unlock()
	if r := disconnect(insertProvider(t, env, w, "fakeremove", `{"demo": true}`)); r.Webhook != "removed" {
		t.Fatalf("a demo provider: %+v", r)
	}
	removed.Lock()
	after := len(removed.urls)
	removed.Unlock()
	if after != before {
		t.Fatal("a demo provider's webhook was removed at the provider")
	}
}
