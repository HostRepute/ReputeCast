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

// TestClassifierAlongsideTierRun: the mailbox classifier and a tier run
// update the same contacts at once without deadlocking, and between them
// reach every contact. The tier run goes in batches (more than two here,
// with many contacts joined at the same moment across a batch's edge), and
// a second run finds nothing left to change (review 2026-11 OP-13).
func TestClassifierAlongsideTierRun(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com"})
	ws := env.CreateWorkspace("Busy audience", owner.ID)
	const contacts = 12000
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, mailbox_provider, created_at)
		SELECT gen_random_uuid(), $1, 'reader' || g || '@busy-corp.example', 'other', now() - (g % 10) * interval '40 days'
		FROM generate_series(1, $2::int) g`, ws.ID, contacts); err != nil {
		t.Fatal(err)
	}
	dns := &fakeDNS{got: map[string]int{}, mx: map[string]string{"busy-corp.example": "aspmx.l.google.com"}}
	c := &mailbox.Classifier{Q: env.App.Q, MX: &mailbox.MXChecker{Q: env.App.Q, Resolver: dns, Timeout: time.Second}}

	var wg sync.WaitGroup
	var tierErr, classErr error
	var changed int64
	wg.Go(func() {
		run, err := env.App.Audience().RecomputeTiers(ctx, ws.ID)
		changed, tierErr = run.Changed, err
	})
	wg.Go(func() { _, classErr = c.ClassifyPending(ctx, ws.ID, 30*time.Second) })
	wg.Wait()
	if tierErr != nil || classErr != nil {
		t.Fatalf("tiers: %v; classifier: %v", tierErr, classErr)
	}
	if changed == 0 {
		t.Fatal("the tier run changed nothing")
	}
	// Contacts the tier run held were left for the classifier's next pass.
	if _, err := c.ClassifyPending(ctx, ws.ID, 30*time.Second); err != nil {
		t.Fatal(err)
	}
	var waiting int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND mailbox_checked_at IS NULL`, ws.ID).Scan(&waiting); err != nil {
		t.Fatal(err)
	}
	if waiting != 0 {
		t.Fatalf("%d contacts still wait for their mailbox provider", waiting)
	}
	again, err := env.App.Audience().RecomputeTiers(ctx, ws.ID)
	if err != nil || again.Changed != 0 {
		t.Fatalf("a second run changed %d (%v): the first missed contacts", again.Changed, err)
	}
	var tiers int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(DISTINCT tier) FROM contacts WHERE workspace_id = $1`, ws.ID).Scan(&tiers); err != nil {
		t.Fatal(err)
	}
	if tiers < 3 {
		t.Fatalf("%d tiers across ten join dates a season apart", tiers)
	}
}

// TestClassifierSkipsLockedContacts: a contact another transaction holds
// (a tier batch, an edit) doesn't keep the classifier waiting; it is
// classified on the next pass. A classifier that waited could deadlock with
// a tier run, which is what kept the first tier run of a million contacts
// starting over (review 2026-11 OP-13).
func TestClassifierSkipsLockedContacts(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com"})
	ws := env.CreateWorkspace("Locked contact", owner.ID)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, mailbox_provider)
		SELECT gen_random_uuid(), $1, 'reader' || g || '@held-corp.example', 'other' FROM generate_series(1, 20) g`, ws.ID); err != nil {
		t.Fatal(err)
	}
	tx, err := env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT id FROM contacts WHERE email = 'reader7@held-corp.example' FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	dns := &fakeDNS{got: map[string]int{}, mx: map[string]string{"held-corp.example": "aspmx.l.google.com"}}
	c := &mailbox.Classifier{Q: env.App.Q, MX: &mailbox.MXChecker{Q: env.App.Q, Resolver: dns, Timeout: time.Second}}
	wait, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	n, err := c.ClassifyPending(wait, ws.ID, 5*time.Second)
	if err != nil || wait.Err() != nil {
		t.Fatalf("the classifier waited for the held contact: %v %v", err, wait.Err())
	}
	if n != 19 {
		t.Fatalf("classified %d of the 19 contacts nobody holds", n)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var held string
	if err := env.App.Pool.QueryRow(ctx, `SELECT mailbox_provider FROM contacts WHERE email = 'reader7@held-corp.example'`).Scan(&held); err != nil {
		t.Fatal(err)
	}
	if held != mailbox.Other {
		t.Fatalf("the held contact was classified as %s while held", held)
	}
	if n, err := c.ClassifyPending(ctx, ws.ID, 5*time.Second); err != nil || n != 1 {
		t.Fatalf("the next pass classified %d (%v), want the one that was held", n, err)
	}
}
