package suppressions_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/suppressions"

	// The Brevo adapter can list its suppressions but take none.
	_ "github.com/hostrepute/reputecast/internal/providers/brevo"
)

// A provider whose API can't take suppressions (AddSuppressions returns
// providers.ErrNotSupported) still has its own list pulled, and its sync
// switches to pull instead of failing every run.
func TestPullOnlyProviderSyncsWithoutError(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Owner"})
	ws := env.CreateWorkspace("Sync test", owner.ID)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("api-key") != "xkeysib-test" || r.URL.Path != "/v3/smtp/blockedContacts" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, `{"contacts":[{"email":"blocked@example.com","reason":{"code":"hardBounce"},"blockedAt":"2026-10-12T10:00:00Z"}],"count":1}`)
	}))
	defer api.Close()

	id := uuid.Must(uuid.NewV7())
	creds, err := env.App.Secrets.Seal([]byte(`{"api_key":"xkeysib-test"}`), manage.CredentialsAD(store.Provider{ID: id}))
	if err != nil {
		t.Fatal(err)
	}
	token, _ := authn.NewToken()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, config, credentials_enc, webhook_token, capabilities)
		VALUES ($1, $2, 'brevo', 'Brevo', $3, $4, $5, '{}')`, id, ws.ID, `{"base_url":"`+api.URL+`"}`, creds, token); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Q.IngestAddSuppression(ctx, store.IngestAddSuppressionParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID,
		Email: "manual@example.com", Reason: "manual", Source: "manual", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	p, err := env.App.Q.ProvGetAny(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	svc := &suppressions.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Factory: &manage.Factory{Box: env.App.Secrets},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := svc.SyncProvider(ctx, p); err != nil {
		t.Fatalf("a pull-only provider must not fail the sync: %v", err)
	}
	row, err := env.App.Q.SuppSyncGet(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if row.Mode != "pull" || row.Status != "ok" || row.Pulled != 1 || row.Pushed != 0 {
		t.Fatalf("sync row: %+v", row)
	}
	var reason string
	if err := env.App.Pool.QueryRow(ctx, `SELECT reason FROM suppressions WHERE workspace_id = $1 AND email = 'blocked@example.com'`, ws.ID).Scan(&reason); err != nil || reason != "hard_bounce" {
		t.Fatalf("pulled address: %q, %v", reason, err)
	}
	// The next run only pulls.
	if err := svc.SyncProvider(ctx, p); err != nil {
		t.Fatal(err)
	}

	// Set to push only, the sync has nothing it can do: it turns off
	// rather than starting to pull.
	pushID := uuid.Must(uuid.NewV7())
	pushCreds, err := env.App.Secrets.Seal([]byte(`{"api_key":"xkeysib-test"}`), manage.CredentialsAD(store.Provider{ID: pushID}))
	if err != nil {
		t.Fatal(err)
	}
	pushToken, _ := authn.NewToken()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, config, credentials_enc, webhook_token, capabilities)
		VALUES ($1, $2, 'brevo', 'Brevo push', $3, $4, $5, '{}')`, pushID, ws.ID, `{"base_url":"`+api.URL+`"}`, pushCreds, pushToken); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO provider_suppression_sync (provider_id, mode) VALUES ($1, 'push')`, pushID); err != nil {
		t.Fatal(err)
	}
	pp, err := env.App.Q.ProvGetAny(ctx, pushID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SyncProvider(ctx, pp); err != nil {
		t.Fatalf("a push-only sync to a pull-only provider must not fail: %v", err)
	}
	if row, err := env.App.Q.SuppSyncGet(ctx, pushID); err != nil || row.Mode != "off" || row.Pulled != 0 {
		t.Fatalf("push-only sync row: %+v, %v", row, err)
	}
}
