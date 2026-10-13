package manage_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// hooking is a test adapter that registers its own webhook.
type hooking struct{ fake }

var registered struct {
	sync.Mutex
	urls []string
}

func (hooking) Adapter() string { return "fakehooks" }

func (hooking) RegisterWebhook(_ context.Context, url string) error {
	registered.Lock()
	defer registered.Unlock()
	registered.urls = append(registered.urls, url)
	return nil
}

func init() {
	providers.Register(providers.Descriptor{Adapter: "fakehooks", Name: "Hooking ESP", Kind: "api", Capabilities: fake{}.Capabilities(),
		Fields: []providers.Field{{Key: "token", Label: "API token", Secret: true, Required: true}},
		New:    func(providers.Config) (providers.Provider, error) { return hooking{}, nil }})
}

// addProvider connects a test adapter to the workspace.
func addProvider(t *testing.T, env *apptest.Env, w store.Workspace, adapter, name string) store.Provider {
	t.Helper()
	p := addFake(t, env, w, nil)
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE providers SET adapter = $2, name = $3 WHERE id = $1`, p.ID, adapter, name); err != nil {
		t.Fatal(err)
	}
	p, err := env.App.Q.ProvGetAny(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPublicURLMovesWebhooks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	manual := addProvider(t, env, w, "faketest", "Fake ESP")
	auto := addProvider(t, env, w, "fakehooks", "Hooking ESP")
	svc := service(env, time.Now)

	// The demo's own providers have no real webhook and are left out.
	mv, err := svc.PublicURLMoved(ctx, "https://mail.acme.io")
	if err != nil {
		t.Fatal(err)
	}
	if mv.Reregistering != 1 || len(mv.Manual) != 1 || mv.Manual[0].Provider != "Fake ESP" || mv.Manual[0].Workspace != w.Name ||
		mv.Manual[0].URL != "https://mail.acme.io/hooks/faketest/"+manual.WebhookToken {
		t.Fatalf("move: %+v", mv)
	}
	if n := count(t, env, `SELECT count(*) FROM river_job WHERE kind = 'providers.move_webhooks'`); n != 1 {
		t.Fatalf("jobs: %d", n)
	}

	// The job registers the address the instance has when it runs.
	worker := &manage.MoveWebhooksWorker{S: svc, PublicURL: func(context.Context) string { return "https://mail.acme.io" }}
	if err := worker.Work(ctx, &river.Job[manage.MoveWebhooksArgs]{Args: manage.MoveWebhooksArgs{ProviderIDs: []uuid.UUID{auto.ID}}}); err != nil {
		t.Fatal(err)
	}
	registered.Lock()
	got := strings.Join(registered.urls, " ")
	registered.Unlock()
	if got != "https://mail.acme.io/hooks/fakehooks/"+auto.WebhookToken {
		t.Fatalf("registered: %q", got)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.webhook_register' AND target_id = $1 AND metadata->>'reason' = 'public URL changed'`, auto.ID.String()); n != 1 {
		t.Fatalf("audit: %d", n)
	}
}

// When the re-registration can't be queued, nothing claims to move the
// webhooks: every provider is listed for a manual update.
func TestPublicURLMovedWithoutJobs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	w := demoWorkspace(t, env)
	auto := addProvider(t, env, w, "fakehooks", "Hooking ESP")
	svc := service(env, time.Now)
	svc.Jobs = nil
	mv, err := svc.PublicURLMoved(context.Background(), "https://mail.acme.io")
	if err == nil || mv.Reregistering != 0 || len(mv.Manual) != 1 || mv.Manual[0].URL != "https://mail.acme.io/hooks/fakehooks/"+auto.WebhookToken {
		t.Fatalf("move without jobs: %+v %v", mv, err)
	}
}
