package auditlog_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/auditlog"
	"github.com/hostrepute/reputecast/internal/maintenance"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

type instanceList struct {
	Items []struct {
		ID, Action, Target, Summary, Result, Reason string
		Workspace, WorkspaceSlug                    string
		Actor                                       struct{ Type, Name, Label string }
		Source                                      struct{ Channel, Detail string }
	}
	NextCursor    string
	RetentionDays int
	Workspaces    []struct{ Slug, Name string }
}

func TestInstanceAuditLog(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	admin := env.Client()
	admin.SignInDemo("linh@acme.io")

	var l instanceList
	admin.Do("GET", "/admin/audit?workspace=instance&limit=20", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) < 7 || l.RetentionDays != 365 || len(l.Workspaces) != 2 {
		t.Fatalf("instance list: %d items, retention %d, workspaces %v", len(l.Items), l.RetentionDays, l.Workspaces)
	}
	byAction := map[string]int{}
	for i, it := range l.Items {
		if it.Workspace != "" {
			t.Fatalf("workspace entry in the instance-only view: %+v", it)
		}
		byAction[it.Action] = i
	}
	rej := l.Items[byAction["signin.rejected"]]
	if rej.Actor.Name != "System" || rej.Actor.Label != "automatic" || rej.Result != "denied" || rej.Target != "kim@gmail.com" ||
		rej.Summary != "Google Workspace SSO · email domain not allowed · reference auth_7f3k9dqe" || rej.Source.Channel != "auth" || rej.Source.Detail != "203.0.113.77" {
		t.Fatalf("rejection row: %+v", rej)
	}
	// The reference users see on the error page finds the entry.
	var found instanceList
	admin.Do("GET", "/admin/audit?q=auth_7f3k9dqe", nil).MustStatus(t, 200).JSON(t, &found)
	if len(found.Items) != 1 || found.Items[0].ID != rej.ID {
		t.Fatalf("search by reference: %+v", found.Items)
	}
	// All workspaces: workspace entries carry their workspace.
	var all instanceList
	admin.Do("GET", "/admin/audit?limit=50", nil).MustStatus(t, 200).JSON(t, &all)
	hasWorkspace := false
	for _, it := range all.Items {
		if it.WorkspaceSlug == "acme-newsletter" && it.Workspace == "Acme Newsletter" {
			hasWorkspace = true
		}
	}
	if !hasWorkspace || all.NextCursor == "" {
		t.Fatalf("all: workspace rows %v, next %q", hasWorkspace, all.NextCursor)
	}
	var one instanceList
	admin.Do("GET", "/admin/audit?workspace=acme-engineering-blog&range=all&limit=200", nil).MustStatus(t, 200).JSON(t, &one)
	for _, it := range one.Items {
		if it.WorkspaceSlug != "acme-engineering-blog" {
			t.Fatalf("workspace filter: %+v", it)
		}
	}
	var denied instanceList
	admin.Do("GET", "/admin/audit?workspace=instance&result=denied", nil).MustStatus(t, 200).JSON(t, &denied)
	for _, it := range denied.Items {
		if it.Result != "denied" {
			t.Fatalf("result filter: %+v", it)
		}
	}
	admin.Do("GET", "/admin/audit?workspace=nope", nil).MustStatus(t, 422)

	// Detail: what happened, the instance level and the admin screen.
	var d struct {
		What, Workspace, AdminPath, LinkLabel string
		Event                                 struct{ Action string }
		Diff                                  []struct{ Kind, Text string }
	}
	deact := l.Items[byAction["user.deactivated"]]
	admin.Do("GET", "/admin/audit/events/"+deact.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Workspace != "Instance level" || d.AdminPath != "users?q=tom%40acme.io" || d.What != "Deactivated user Tom Becker · tom@acme.io" || len(d.Diff) == 0 {
		t.Fatalf("detail: %+v", d)
	}
	sso := l.Items[byAction["sso.provider.updated"]]
	admin.Do("GET", "/admin/audit/events/"+sso.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.What != "Changed SSO provider “Microsoft Entra ID”" || !strings.HasPrefix(d.AdminPath, "sso/") {
		t.Fatalf("sso detail: %+v", d)
	}

	// Exports, with the workspace column, are audited at instance level.
	r := admin.Do("GET", "/admin/audit/export?workspace=instance&format=csv", nil).MustStatus(t, 200)
	if !strings.HasPrefix(string(r.Body), "occurred_at,workspace,actor_type") || !strings.Contains(string(r.Body), "signin.rejected") {
		t.Fatalf("csv: %.300s", r.Body)
	}
	r = admin.Do("GET", "/admin/audit/export?workspace=instance&format=json&range=all", nil).MustStatus(t, 200)
	var rows []map[string]any
	if err := json.Unmarshal(r.Body, &rows); err != nil || len(rows) < 7 {
		t.Fatalf("json: %v %d", err, len(rows))
	}
	var n int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'audit.exported' AND workspace_id IS NULL`).Scan(&n)
	if n != 2 {
		t.Fatalf("%d instance exports audited", n)
	}

	// Retention.
	var st struct {
		Settings struct {
			RetentionDays    int
			WorkspaceStreams []struct{ Slug, Host string }
		}
	}
	admin.Do("PUT", "/admin/audit/settings", map[string]any{"retentionDays": 180}).MustStatus(t, 200).JSON(t, &st)
	if st.Settings.RetentionDays != 180 || len(st.Settings.WorkspaceStreams) != 1 || st.Settings.WorkspaceStreams[0].Host != "warehouse.acme.io" {
		t.Fatalf("settings: %+v", st)
	}
	admin.Do("PUT", "/admin/audit/settings", map[string]any{"retentionDays": 7}).MustStatus(t, 422)
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'audit.settings_updated' AND workspace_id IS NULL AND metadata->>'summary' = 'retention 180 days (was 365)'`).Scan(&n)
	if n != 1 {
		t.Fatal("retention change not audited")
	}

	// Instance admins only.
	for _, email := range []string{"aisha@acme.io", "marcus@acme.io"} {
		c := env.Client()
		c.SignInDemo(email)
		c.Do("GET", "/admin/audit", nil).MustStatus(t, 403)
		c.Do("GET", "/admin/audit/events/"+rej.ID, nil).MustStatus(t, 403)
		c.Do("GET", "/admin/audit/export", nil).MustStatus(t, 403)
	}
}

func TestInstanceAuditStreamAndRetention(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var mu sync.Mutex
	var batches []map[string]any
	var sigOK bool
	secretCh := make(chan string, 1)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var b map[string]any
		_ = json.Unmarshal(body, &b)
		mu.Lock()
		batches = append(batches, b)
		select {
		case secret := <-secretCh:
			ts := r.Header.Get(webhooks.HeaderTimestamp)
			var unix int64
			_ = json.Unmarshal([]byte(ts), &unix)
			sigOK = r.Header.Get(webhooks.HeaderSignature) == webhooks.Sign(secret, unix, body)
		default:
		}
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer sink.Close()
	s := &auditlog.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true }}
	res, err := s.UpdateInstanceSettings(ctx, auditlog.AuditSettingsInput{Stream: &auditlog.AuditStreamInput{Enabled: true, URL: sink.URL + "/audit"}})
	if err != nil || res.Secret == "" || !res.Settings.Stream.Enabled {
		t.Fatalf("enable stream: %v %+v", err, res)
	}
	secretCh <- res.Secret
	// Only entries after the stream was turned on go out, in order.
	if err := env.App.Audit.Log(ctx, auditLogEvent("signin.succeeded")); err != nil {
		t.Fatal(err)
	}
	w := &auditlog.InstanceStreamWorker{S: s, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := w.Work(ctx, &river.Job[auditlog.InstanceStreamArgs]{}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	if len(batches) != 1 || batches[0]["scope"] != "instance" || !sigOK {
		t.Fatalf("batches %v, signature ok %v", batches, sigOK)
	}
	events := batches[0]["events"].([]any)
	mu.Unlock()
	for _, e := range events {
		if e.(map[string]any)["action"] == "approval.requested" {
			t.Fatal("a workspace entry was streamed as an instance entry")
		}
	}
	got, _ := s.InstanceSettings(ctx)
	if got.Stream.Delivered != int64(len(events)) || got.Stream.LastOKAt == nil {
		t.Fatalf("stream state: %+v", got.Stream)
	}
	// Nothing new: no request.
	_ = w.Work(ctx, &river.Job[auditlog.InstanceStreamArgs]{})
	mu.Lock()
	if len(batches) != 1 {
		t.Fatalf("%d batches", len(batches))
	}
	mu.Unlock()

	// The hourly cleanup applies the instance retention to instance entries.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_name, action, target_type, target_name)
		VALUES ($1, NULL, now() - interval '400 days', 'system', 'System', 'signin.failed', 'user', 'old@acme.io')`, uuid.Must(uuid.NewV7())); err != nil {
		t.Fatal(err)
	}
	c := &maintenance.CleanupWorker{Pool: env.App.Pool, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := c.Work(ctx, &river.Job[maintenance.CleanupArgs]{}); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE target_name = 'old@acme.io'`).Scan(&n)
	if n != 0 {
		t.Fatal("an instance entry past retention was kept")
	}
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id IS NULL AND occurred_at > now() - interval '30 days'`).Scan(&n)
	if n < 7 {
		t.Fatalf("recent instance entries were purged: %d left", n)
	}
}

func auditLogEvent(action string) audit.Event {
	return audit.Event{Action: action, TargetType: "user", TargetName: "stream@acme.io", Actor: &audit.Actor{Type: "system", Name: "System"}}
}
