package auditlog_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/auditlog"
)

const ws = "/workspaces/acme-newsletter"

type eventList struct {
	Items []struct {
		ID, Action, Target, Summary, Result, Reason, RequestID string
		Actor                                                  struct{ Type, Name, Label, Role string }
		Source                                                 struct{ Channel, Detail string }
	}
	NextCursor   string
	Total        int64
	ActionGroups []string
	Stream       struct {
		Enabled bool
		Host    string
	}
}

func TestAuditLog(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l eventList
	c.Do("GET", ws+"/audit", nil).MustStatus(t, 200).JSON(t, &l)
	if l.Total < 41000 || !l.Stream.Enabled || l.Stream.Host != "warehouse.acme.io" || len(l.Items) < 10 {
		t.Fatalf("list: total=%d stream=%+v items=%d", l.Total, l.Stream, len(l.Items))
	}
	if first := l.Items[0]; first.Action != "approval.requested" || first.Actor.Name != "Daniel Park" || first.Actor.Role != "Content Writer" || first.Source.Channel != "web" {
		t.Fatalf("first: %+v", first)
	}
	var routing string
	for _, it := range l.Items {
		if it.Action == "routing.updated" && it.RequestID == "req_8c1f2a9d" {
			routing = it.ID
			if it.Target != "Routing rule · Broadcasts" || it.Summary != "SES 95% / Postal 5% → SES 90% / Postal 10%" {
				t.Fatalf("routing row: %+v", it)
			}
		}
		if it.Action == "report.read" && (it.Actor.Label != "MCP agent" || it.Source.Detail != "rc_agent_7a…c2") {
			t.Fatalf("agent row: %+v", it)
		}
	}
	if routing == "" {
		t.Fatal("routing change in the last 7 days")
	}

	c.Do("GET", ws+"/audit?result=denied", nil).MustStatus(t, 200).JSON(t, &l)
	for _, it := range l.Items {
		if it.Result != "denied" {
			t.Fatalf("denied filter: %+v", it)
		}
	}
	if len(l.Items) != 3 {
		t.Fatalf("denied events: %d", len(l.Items))
	}
	c.Do("GET", ws+"/audit?actorType=mcp_agent", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) != 5 {
		t.Fatalf("agent events: %d", len(l.Items))
	}
	c.Do("GET", ws+"/audit?q=req_8c1f2a9d", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) != 1 {
		t.Fatalf("request id search: %d", len(l.Items))
	}
	c.Do("GET", ws+"/audit?action=routing&range=all", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) != 2 {
		t.Fatalf("action group: %d", len(l.Items))
	}
	// A category of the action filter covers several groups (Routing & providers).
	c.Do("GET", ws+"/audit?category=routing&range=all&limit=200", nil).MustStatus(t, 200).JSON(t, &l)
	groups := map[string]bool{}
	for _, it := range l.Items {
		g, _, _ := strings.Cut(it.Action, ".")
		groups[g] = true
	}
	if !groups["routing"] || len(groups) < 2 {
		t.Fatalf("routing category: %v", groups)
	}
	for g := range groups {
		if !slices.Contains(auditlog.Categories[1].Groups, g) {
			t.Fatalf("routing category has %s", g)
		}
	}
	c.Do("GET", ws+"/audit?category=nope", nil).MustStatus(t, 422)
	// Paging with a cursor over the whole history.
	c.Do("GET", ws+"/audit?range=all&limit=20", nil).MustStatus(t, 200).JSON(t, &l)
	if l.NextCursor == "" || len(l.Items) != 20 {
		t.Fatal("first page has a cursor")
	}
	var next eventList
	c.Do("GET", ws+"/audit?range=all&limit=20&cursor="+l.NextCursor, nil).MustStatus(t, 200).JSON(t, &next)
	if len(next.Items) != 20 || next.Items[0].ID == l.Items[19].ID {
		t.Fatal("second page continues after the first")
	}

	var detail struct {
		What, Workspace, Client, Permission, Link, LinkLabel string
		Diff                                                 []struct{ Kind, Text string }
		Related                                              []string
		JSON                                                 json.RawMessage
	}
	c.Do("GET", ws+"/audit/events/"+routing, nil).MustStatus(t, 200).JSON(t, &detail)
	if detail.What != "Changed the Broadcasts routing rule" || detail.Client != "Chrome 141 on macOS 26" || detail.Permission != "providers:manage" ||
		detail.Workspace != "Acme Newsletter" || detail.LinkLabel != "Open routing" || len(detail.Related) != 1 {
		t.Fatalf("detail: %+v", detail)
	}
	var kinds []string
	for _, d := range detail.Diff {
		kinds = append(kinds, d.Kind)
	}
	if strings.Join(kinds, ",") != "same,del,add,same,same" || !strings.Contains(detail.Diff[1].Text, `"ses_us_east_1": 95`) {
		t.Fatalf("diff: %+v", detail.Diff)
	}

	// Exports stream CSV and JSON, and are themselves logged.
	r := c.Do("GET", ws+"/audit/export?format=csv&range=7d", nil).MustStatus(t, 200)
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/csv") || !strings.Contains(string(r.Body), "routing.updated") || !strings.HasPrefix(string(r.Body), "occurred_at,") {
		t.Fatalf("csv: %s", r.Body[:min(200, len(r.Body))])
	}
	r = c.Do("GET", ws+"/audit/export?format=json&result=denied", nil).MustStatus(t, 200)
	var rows []map[string]any
	if err := json.Unmarshal(r.Body, &rows); err != nil || len(rows) != 3 {
		t.Fatalf("json export: %d %v", len(rows), err)
	}
	var n int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'audit.exported'`).Scan(&n)
	if n != 2 {
		t.Fatalf("exports logged: %d", n)
	}

	// Settings: retention and the stream need workspace:manage.
	var res struct {
		Settings struct{ RetentionDays int32 }
		Secret   string
	}
	c.Do("PUT", ws+"/audit/settings", map[string]any{"retentionDays": 90}).MustStatus(t, 200).JSON(t, &res)
	if res.Settings.RetentionDays != 90 {
		t.Fatalf("retention: %+v", res)
	}
	c.Do("PUT", ws+"/audit/settings", map[string]any{"retentionDays": 10}).MustStatus(t, 422)
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/audit", nil).MustStatus(t, 200)
	v.Do("PUT", ws+"/audit/settings", map[string]any{"retentionDays": 120}).MustStatus(t, 403)

	// Stream to a receiver: one signed batch, the cursor moves.
	var got []byte
	var sig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		sig = r.Header.Get("X-ReputeCast-Signature")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c.Do("PUT", ws+"/audit/settings", map[string]any{"stream": map[string]any{"enabled": true, "url": srv.URL}}).MustStatus(t, 422) // private address
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	c.Do("PUT", ws+"/audit/settings", map[string]any{"stream": map[string]any{"enabled": true, "url": srv.URL, "rotateSecret": true}}).MustStatus(t, 200).JSON(t, &res)
	if res.Secret == "" {
		t.Fatal("new stream secret shown once")
	}
	svc := &auditlog.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true }}
	w := &auditlog.StreamWorker{S: svc, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := w.Work(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "audit.settings_updated") || sig == "" {
		t.Fatalf("stream batch: %s sig=%q", got, sig)
	}
}

func TestDiffAndFormat(t *testing.T) {
	before := json.RawMessage(`{"stream":"broadcasts","weights":{"ses":95,"postal":5},"conditions":[]}`)
	after := json.RawMessage(`{"stream":"broadcasts","weights":{"ses":90,"postal":10},"conditions":[]}`)
	d := auditlog.Diff(before, after)
	if len(d) != 4 || d[1].Kind != "del" || d[1].Text != `"weights": { "ses": 95, "postal": 5 },` || d[3].Text != `"conditions": []` {
		t.Fatalf("diff: %+v", d)
	}
	if got := auditlog.Client("Mozilla/5.0 (Macintosh; Intel Mac OS X 26_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"); got != "Chrome 141 on macOS 26" {
		t.Fatalf("client: %q", got)
	}
	if got := auditlog.Describe("member.role_changed", "member", "Aisha Bello"); got != "Changed the role of member “Aisha Bello”" {
		t.Fatalf("describe: %q", got)
	}
}
