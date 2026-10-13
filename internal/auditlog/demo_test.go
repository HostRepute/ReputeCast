package auditlog_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

// The demo's audit stream points at a fictional address: it is never sent
// to, until someone gives it an address of their own.
func TestDemoStreamIsNeverSent(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	streams := func() int {
		rows, err := env.App.Q.AuditStreamWorkspaces(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return len(rows)
	}
	if n := streams(); n != 0 {
		t.Fatalf("demo streams selected for sending: %d", n)
	}
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	same := "https://warehouse.acme.io/ingest/audit"
	if err := env.App.Q.AuditStreamConfigure(ctx, store.AuditStreamConfigureParams{ID: w.ID, Enabled: true, Url: &same}); err != nil {
		t.Fatal(err)
	}
	if n := streams(); n != 0 {
		t.Fatal("saving the same address made the demo stream real")
	}
	mine := "https://siem.example.net/ingest"
	if err := env.App.Q.AuditStreamConfigure(ctx, store.AuditStreamConfigureParams{ID: w.ID, Enabled: true, Url: &mine}); err != nil {
		t.Fatal(err)
	}
	if n := streams(); n != 1 {
		t.Fatalf("a stream with a new address isn't sent: %d", n)
	}
}
