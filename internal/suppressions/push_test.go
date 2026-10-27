package suppressions_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/ses"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/suppressions"
)

// fakeSES is the part of the SES v2 API the suppression sync and a send
// use: suppression lists per tenant ("" is the account-level list) and
// SendEmail, which drops mail to an address on the list it sends through,
// as SES does.
type fakeSES struct {
	mu        sync.Mutex
	lists     map[string]map[string]string // tenant → email → reason
	puts      []string                     // "tenant|email|reason"
	deletes   []string                     // "tenant|email"
	sent      []string
	dropped   []string
	failLists bool
}

func (f *fakeSES) handler() http.Handler {
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("PUT /v2/email/suppression/addresses", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ EmailAddress, Reason, TenantName string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.lists[in.TenantName] == nil {
			f.lists[in.TenantName] = map[string]string{}
		}
		f.lists[in.TenantName][in.EmailAddress] = in.Reason
		f.puts = append(f.puts, in.TenantName+"|"+in.EmailAddress+"|"+in.Reason)
		write(w, map[string]any{})
	})
	mux.HandleFunc("GET /v2/email/suppression/addresses", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.failLists {
			w.Header().Set("X-Amzn-ErrorType", "ServiceUnavailable")
			w.WriteHeader(http.StatusServiceUnavailable)
			write(w, map[string]string{"message": "Service Unavailable"})
			return
		}
		out := []map[string]any{}
		for email, reason := range f.lists[r.URL.Query().Get("TenantName")] {
			out = append(out, map[string]any{"EmailAddress": email, "Reason": reason, "LastUpdateTime": float64(time.Now().Unix())})
		}
		write(w, map[string]any{"SuppressedDestinationSummaries": out})
	})
	mux.HandleFunc("DELETE /v2/email/suppression/addresses/{email}", func(w http.ResponseWriter, r *http.Request) {
		email, _ := url.PathUnescape(r.PathValue("email"))
		tenant := r.URL.Query().Get("TenantName")
		f.mu.Lock()
		defer f.mu.Unlock()
		delete(f.lists[tenant], email)
		f.deletes = append(f.deletes, tenant+"|"+email)
		write(w, map[string]any{})
	})
	mux.HandleFunc("POST /v2/email/outbound-emails", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Destination struct{ ToAddresses []string }
			TenantName  string
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, to := range in.Destination.ToAddresses {
			if _, ok := f.lists[in.TenantName][to]; ok {
				f.dropped = append(f.dropped, to)
			} else {
				f.sent = append(f.sent, to)
			}
		}
		write(w, map[string]string{"MessageId": "0100018f2b4d5c6e-fake-000000"})
	})
	return mux
}

type pushEnv struct {
	env *apptest.Env
	ws  store.Workspace
	svc *suppressions.Service
	ses *fakeSES
	url string
}

func newPushEnv(t *testing.T) *pushEnv {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Owner"})
	f := &fakeSES{lists: map[string]map[string]string{}}
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return &pushEnv{env: env, ws: env.CreateWorkspace("Push test", owner.ID), ses: f, url: srv.URL,
		svc: &suppressions.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Factory: &manage.Factory{Box: env.App.Secrets},
			Log: slog.New(slog.NewTextHandler(io.Discard, nil))}}
}

// sesProvider connects an Amazon SES provider on the fake API, with the
// workspace's tenant or without one.
func (e *pushEnv) sesProvider(t *testing.T, tenant string) store.Provider {
	t.Helper()
	ctx := context.Background()
	id := uuid.Must(uuid.NewV7())
	creds, err := e.env.App.Secrets.Seal([]byte(`{"access_key_id":"AKIDEXAMPLE","secret_access_key":"secret"}`), manage.CredentialsAD(store.Provider{ID: id}))
	if err != nil {
		t.Fatal(err)
	}
	settings, _ := json.Marshal(ses.Settings{Region: "us-east-1", Access: "keys", Endpoint: e.url, Tenant: tenant})
	token, _ := authn.NewToken()
	if _, err := e.env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, config, credentials_enc, webhook_token, capabilities)
		VALUES ($1, $2, 'ses', 'Amazon SES', $3, $4, $5, '{}')`, id, e.ws.ID, settings, creds, token); err != nil {
		t.Fatal(err)
	}
	p, err := e.env.App.Q.ProvGetAny(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (e *pushEnv) suppress(t *testing.T, email, reason, scope string, created time.Time) {
	t.Helper()
	if _, err := e.env.App.Pool.Exec(context.Background(), `INSERT INTO suppressions (id, workspace_id, email, reason, scope, source, created_at)
		VALUES ($1, $2, $3, $4, $5, 'test', $6)`, uuid.Must(uuid.NewV7()), e.ws.ID, email, reason, scope, created); err != nil {
		t.Fatal(err)
	}
}

func (e *pushEnv) syncRow(t *testing.T, p store.Provider) store.ProviderSuppressionSync {
	t.Helper()
	row, err := e.env.App.Q.SuppSyncGet(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

// PR-1: only hard bounces and complaints that apply to all mail go to SES,
// on the workspace's tenant list; an unsubscribed contact still gets a
// transactional message through SES.
func TestSuppressionPushKeepsUnsubscribesOffSES(t *testing.T) {
	e := newPushEnv(t)
	ctx := context.Background()
	p := e.sesProvider(t, "acme-newsletter")
	now := time.Now()
	e.suppress(t, "bounced@example.com", "hard_bounce", "all", now.Add(-time.Hour))
	e.suppress(t, "complained@example.com", "complaint", "all", now.Add(-time.Hour))
	e.suppress(t, "unsubscribed@example.com", "unsubscribe", "marketing", now.Add(-time.Hour))
	e.suppress(t, "manual@example.com", "manual", "all", now.Add(-time.Hour))
	e.suppress(t, "info@example.com", "role", "marketing", now.Add(-time.Hour))

	if err := e.svc.SyncProvider(ctx, p); err != nil {
		t.Fatal(err)
	}
	slices.Sort(e.ses.puts)
	if want := []string{"acme-newsletter|bounced@example.com|BOUNCE", "acme-newsletter|complained@example.com|COMPLAINT"}; !slices.Equal(e.ses.puts, want) {
		t.Fatalf("pushed %v, want %v", e.ses.puts, want)
	}
	row := e.syncRow(t, p)
	if row.Pushed != 2 || row.Pending != 0 || row.Scope != "tenant" || row.Status != "ok" || row.PushSeq == 0 {
		t.Fatalf("sync row %+v", row)
	}

	// The unsubscribed contact is suppressed for marketing only: ReputeCast
	// lets a transactional message through, and SES delivers it, because it
	// isn't on the tenant's list. The hard-bounced address stays stopped.
	allMail, err := e.env.App.Q.TxnRouteSuppressedAllMail(ctx, store.TxnRouteSuppressedAllMailParams{WorkspaceID: e.ws.ID, Email: "unsubscribed@example.com"})
	if err != nil || allMail {
		t.Fatalf("an unsubscribe must not stop transactional mail: %v %v", allMail, err)
	}
	adapter, err := e.svc.Factory.Open(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	reset := providers.Message{ID: uuid.Must(uuid.NewV7()), Stream: providers.StreamSystem, From: mail.Address{Address: "no-reply@acme.io"},
		To: mail.Address{Address: "unsubscribed@example.com"}, Subject: "Reset your password", Text: "Reset it"}
	if _, err := adapter.Send(ctx, []providers.Message{reset}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.ses.sent, []string{"unsubscribed@example.com"}) || len(e.ses.dropped) != 0 {
		t.Fatalf("SES sent %v, dropped %v", e.ses.sent, e.ses.dropped)
	}
	if _, ok := e.ses.lists[""]["unsubscribed@example.com"]; ok {
		t.Fatal("nothing goes to the account-level list")
	}

	// A late bounce (an event dated before the last push) and a removal both
	// reach SES on the next sync; until then they are pending.
	e.suppress(t, "late@example.com", "hard_bounce", "all", now.Add(-48*time.Hour))
	if _, err := e.env.App.Pool.Exec(ctx, `DELETE FROM suppressions WHERE workspace_id = $1 AND email = 'bounced@example.com'`, e.ws.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := e.env.App.Q.SuppPendingCount(ctx, store.SuppPendingCountParams{WorkspaceID: e.ws.ID, ProviderID: &p.ID}); err != nil || n != 2 {
		t.Fatalf("pending %d %v, want 2", n, err)
	}
	if err := e.svc.SyncProvider(ctx, p); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(e.ses.puts, "acme-newsletter|late@example.com|BOUNCE") || !slices.Equal(e.ses.deletes, []string{"acme-newsletter|bounced@example.com"}) {
		t.Fatalf("puts %v deletes %v", e.ses.puts, e.ses.deletes)
	}
	if row := e.syncRow(t, p); row.Pending != 0 || row.Removed != 1 || row.Pushed != 3 {
		t.Fatalf("after the second sync %+v", row)
	}

	// A failed pull doesn't skip the push.
	e.ses.failLists = true
	e.suppress(t, "newer@example.com", "complaint", "all", now)
	if err := e.svc.SyncProvider(ctx, p); err == nil || !strings.Contains(err.Error(), "pull") {
		t.Fatalf("a failed pull is reported: %v", err)
	}
	if !slices.Contains(e.ses.puts, "acme-newsletter|newer@example.com|COMPLAINT") {
		t.Fatalf("the push still ran: %v", e.ses.puts)
	}
	if row := e.syncRow(t, p); row.Status != "error" || !strings.HasPrefix(row.StatusDetail, "pull:") {
		t.Fatalf("status %+v", row)
	}
}

// Without a tenant, SES's account-level list is shared by every app on the
// AWS account: ReputeCast reads it and never writes to it.
func TestSuppressionSyncOnlyReadsTheSESAccountList(t *testing.T) {
	e := newPushEnv(t)
	ctx := context.Background()
	p := e.sesProvider(t, "")
	e.ses.lists[""] = map[string]string{"bounced-elsewhere@example.com": "BOUNCE"}
	e.suppress(t, "bounced@example.com", "hard_bounce", "all", time.Now())
	if err := e.svc.SyncProvider(ctx, p); err != nil {
		t.Fatal(err)
	}
	if len(e.ses.puts) != 0 || len(e.ses.deletes) != 0 {
		t.Fatalf("wrote the account list: puts %v deletes %v", e.ses.puts, e.ses.deletes)
	}
	row := e.syncRow(t, p)
	if row.Pulled != 1 || row.Pushed != 0 || row.Pending != 0 || row.Scope != "account" {
		t.Fatalf("sync row %+v", row)
	}
	var reason string
	if err := e.env.App.Pool.QueryRow(ctx, `SELECT reason FROM suppressions WHERE workspace_id = $1 AND email = 'bounced-elsewhere@example.com'`, e.ws.ID).Scan(&reason); err != nil || reason != "hard_bounce" {
		t.Fatalf("pulled %q %v", reason, err)
	}
	// Removing it from the shared list leaves the account list alone.
	if _, err := e.env.App.Pool.Exec(ctx, `DELETE FROM suppressions WHERE workspace_id = $1`, e.ws.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.SyncProvider(ctx, p); err != nil {
		t.Fatal(err)
	}
	if len(e.ses.deletes) != 0 {
		t.Fatalf("deleted from the account list: %v", e.ses.deletes)
	}
}
