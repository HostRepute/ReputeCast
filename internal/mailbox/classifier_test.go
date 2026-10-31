package mailbox_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mailbox"
)

// fakeDNS answers MX queries from a table; a domain missing from it times
// out (no definite answer).
type fakeDNS struct {
	mu  sync.Mutex
	mx  map[string]string
	got map[string]int
}

func (f *fakeDNS) LookupMX(_ context.Context, name string) ([]*net.MX, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got[name]++
	host, ok := f.mx[name]
	if !ok {
		return nil, &net.DNSError{Err: "i/o timeout", Name: name, IsTimeout: true}
	}
	return []*net.MX{{Host: host, Pref: 1}}, nil
}

func (f *fakeDNS) LookupHost(_ context.Context, name string) ([]string, error) {
	return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
}

func TestClassifier(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com"})
	ws := env.CreateWorkspace("Mailbox test", owner.ID)
	other := env.CreateWorkspace("Mailbox other", owner.ID)
	add := func(ws uuid.UUID, email, provider string) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, mailbox_provider) VALUES ($1, $2, $3, $4)`,
			uuid.Must(uuid.NewV7()), ws, email, provider); err != nil {
			t.Fatal(err)
		}
	}
	add(ws.ID, "ana@acme-corp.example", mailbox.Other)
	add(ws.ID, "bo@acme-corp.example", mailbox.Other)
	add(ws.ID, "cy@northwind.example", mailbox.Other)
	add(ws.ID, "di@berg.example", mailbox.Other)
	add(ws.ID, "ed@flaky.example", mailbox.Other)
	add(ws.ID, "fi@gmail.com", mailbox.Gmail)
	add(other.ID, "gu@acme-corp.example", mailbox.Other)
	add(other.ID, "hq@globex.example", mailbox.Other)

	dns := &fakeDNS{got: map[string]int{}, mx: map[string]string{
		"acme-corp.example": "aspmx.l.google.com",
		"northwind.example": "northwind-example.mail.protection.outlook.com",
		"berg.example":      "mx.berg.example",
		"globex.example":    "globex-example.mail.protection.outlook.com",
	}}
	c := &mailbox.Classifier{Q: env.App.Q, MX: &mailbox.MXChecker{Q: env.App.Q, Resolver: dns, Timeout: time.Second}}

	n, err := c.ClassifyPending(ctx, ws.ID, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// A domain's answer applies to its contacts in every workspace: 3 on
	// Google Workspace (one in the other workspace), 1 Microsoft 365, 1 other.
	if n != 5 {
		t.Fatalf("classified %d contacts, want 5", n)
	}
	type row struct {
		provider, detail string
		checked          bool
	}
	get := func(email string) row {
		t.Helper()
		var r row
		if err := env.App.Pool.QueryRow(ctx, `SELECT mailbox_provider, mailbox_detail, mailbox_checked_at IS NOT NULL FROM contacts WHERE email = $1 ORDER BY workspace_id = $2 DESC LIMIT 1`,
			email, ws.ID).Scan(&r.provider, &r.detail, &r.checked); err != nil {
			t.Fatal(err)
		}
		return r
	}
	for email, want := range map[string]row{
		"ana@acme-corp.example": {mailbox.Gmail, "Google Workspace", true},
		"bo@acme-corp.example":  {mailbox.Gmail, "Google Workspace", true},
		"cy@northwind.example":  {mailbox.Outlook, "Microsoft 365", true},
		"di@berg.example":       {mailbox.Other, "", true},
		// No definite answer: still waiting, retried later.
		"ed@flaky.example": {mailbox.Other, "", false},
		"fi@gmail.com":     {mailbox.Gmail, "", false},
	} {
		if got := get(email); got != want {
			t.Errorf("%s: %+v, want %+v", email, got, want)
		}
	}
	// A domain only the other workspace has waits for its own send or the
	// sweep.
	var globex string
	if err := env.App.Pool.QueryRow(ctx, `SELECT mailbox_provider FROM contacts WHERE email = 'hq@globex.example'`).Scan(&globex); err != nil {
		t.Fatal(err)
	}
	if globex != mailbox.Other || dns.got["globex.example"] != 0 {
		t.Fatalf("a workspace-scoped run looked up another workspace's domain: %s", globex)
	}

	// A failed lookup isn't retried at once.
	before := dns.got["flaky.example"]
	if _, err := c.ClassifyPending(ctx, ws.ID, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if dns.got["flaky.example"] != before {
		t.Fatal("a domain whose lookup just failed was looked up again")
	}

	// The sweep covers every workspace.
	if n, err := c.Sweep(ctx, 10*time.Second, 100); err != nil || n != 1 {
		t.Fatalf("sweep: %d, %v", n, err)
	}
	if got := get("hq@globex.example"); got != (row{mailbox.Outlook, "Microsoft 365", true}) {
		t.Fatalf("sweep: %+v", got)
	}
	if dns.got["acme-corp.example"] != 1 {
		t.Fatalf("acme-corp.example looked up %d times, want once (then cached)", dns.got["acme-corp.example"])
	}

	// Addresses that aren't contacts come from the cache.
	if got := c.ForEmail(ctx, "zoe@acme-corp.example"); got.Provider != mailbox.Gmail || got.Detail != "Google Workspace" {
		t.Fatalf("ForEmail cached: %+v", got)
	}
	if got := c.ForEmail(ctx, "yo@hotmail.com"); got.Provider != mailbox.Outlook {
		t.Fatalf("ForEmail consumer: %+v", got)
	}
	// A domain never seen answers Other until it's looked up (no job client
	// here, so nothing is queued).
	if got := c.ForEmail(ctx, "x@unknown.example"); got.Provider != mailbox.Other {
		t.Fatalf("ForEmail unknown: %+v", got)
	}
}
