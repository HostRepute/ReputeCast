// Package sendtest drives campaign sends end to end in tests: the real
// API's data, the sending engine on a virtual clock and the demo transport
// whose delivery and engagement events go through the same normalization
// as provider webhooks. Reports, Send Readiness and the live monitor are
// tested on what production writes, not on demo rows (review F-5).
package sendtest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/sending/engine"
)

// The engine tests share one seeded demo (seeding the 48,213 contacts takes
// a while) and give every scenario its own campaign and list, driven on a
// virtual clock with the demo transport.

// ContactSpec describes a batch of test contacts.
type ContactSpec struct {
	N        int
	Mailbox  string
	Domain   string
	Tier     string
	Source   string
	Engaged  bool
	Status   string
	Suppress bool
	Paused   bool
	Role     bool
	// Alias makes the batch plus-tagged aliases of the fixture's first
	// contacts (same domain): the same mailboxes.
	Alias bool
	// RolePattern gives the batch role addresses (info+…@) without the flag.
	RolePattern bool
	// Stopped: the sunset policy stopped their Broadcasts.
	Stopped bool
	// StoppedPair adds, for each contact (Stopped), a newer active alias.
	StoppedPair bool
	// Unclassified leaves the mailbox provider to the classifier, which
	// looks the domain's MX record up when the send starts; otherwise
	// Mailbox is final.
	Unclassified bool
}

// Suite is one seeded demo shared by a test's scenarios, with the sending
// engine on a virtual clock that starts at Start.
type Suite struct {
	Env   *apptest.Env
	E     *engine.Engine
	Ctx   context.Context
	WS    uuid.UUID
	Sofia uuid.UUID
	Start time.Time
	n     int
}

// Fixture is one scenario: a copy of Weekly Digest #142 (content, sender,
// A/B test) sending to a list of the scenario's own contacts.
type Fixture struct {
	*Suite
	T        *testing.T
	List     uuid.UUID
	Campaign uuid.UUID
}

// NewSuite seeds the demo (once per test run) and builds the engine.
func NewSuite(t *testing.T) *Suite {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	s := &Suite{Env: env, E: env.App.SendingEngine(), Ctx: context.Background()}
	if err := env.App.Pool.QueryRow(s.Ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&s.WS); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(s.Ctx, `SELECT id FROM users WHERE email = 'sofia@acme.io'`).Scan(&s.Sofia); err != nil {
		t.Fatal(err)
	}
	s.Start = time.Now().Truncate(time.Second)
	start := s.Start
	s.E.Now = func() time.Time { return start }
	return s
}

// Fixture makes a copy of Weekly Digest #142 (content, sender, A/B test)
// that sends to a list of the test's own contacts.
func (s *Suite) Fixture(t *testing.T, name string) *Fixture {
	t.Helper()
	s.n++
	f := &Fixture{Suite: s, T: t, List: uuid.Must(uuid.NewV7()), Campaign: uuid.Must(uuid.NewV7())}
	f.Exec(`DELETE FROM sending_rate_windows`)
	f.Exec(`INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, $3)`, f.List, s.WS, "Send test "+name)
	f.Exec(`INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, reply_to, audience, editor_mode, mjml, html, plain_text,
		ab_test, schedule, thresholds, tracking, created_by, updated_by, content_hash)
		SELECT $1, workspace_id, $2, 'draft', subject, preheader, sender_id, reply_to, $3, editor_mode, mjml, html, plain_text,
		ab_test, '{"mode":"now"}', thresholds, tracking, created_by, updated_by, content_hash
		FROM campaigns WHERE workspace_id = $4 AND name = 'Weekly Digest #142'`, f.Campaign, name, fmt.Sprintf(`{"lists":[%q]}`, f.List), s.WS)
	return f
}

// Exec runs SQL or fails the test.
func (f *Fixture) Exec(sql string, args ...any) {
	f.T.Helper()
	if _, err := f.Env.App.Pool.Exec(f.Ctx, sql, args...); err != nil {
		f.T.Fatalf("%s: %v", sql, err)
	}
}

// Count runs a counting query or fails the test.
func (f *Fixture) Count(sql string, args ...any) int64 {
	f.T.Helper()
	var n int64
	if err := f.Env.App.Pool.QueryRow(f.Ctx, sql, args...).Scan(&n); err != nil {
		f.T.Fatalf("%s: %v", sql, err)
	}
	return n
}

// AddContacts inserts contacts in the fixture's list.
func (f *Fixture) AddContacts(specs ...ContactSpec) {
	f.T.Helper()
	var contacts, members [][]any
	var paused, stopped []uuid.UUID
	i, aliased := 0, 0
	for _, s := range specs {
		for range s.N {
			i++
			id := uuid.Must(uuid.NewV7())
			status := s.Status
			if status == "" {
				status = "active"
			}
			tier := s.Tier
			if tier == "" {
				tier = "active"
			}
			var lastOpen *time.Time
			if s.Engaged {
				t := f.Start.Add(-48 * time.Hour)
				lastOpen = &t
			}
			mbx, checked := s.Mailbox, &f.Start
			if s.Unclassified {
				mbx, checked = "other", nil
			}
			email := fmt.Sprintf("w12.%d.%05d@%s", f.n, i, s.Domain)
			switch {
			case s.Alias:
				aliased++
				email = fmt.Sprintf("w12.%d.%05d+alias@%s", f.n, aliased, s.Domain)
			case s.RolePattern:
				email = fmt.Sprintf("info+w12.%d.%05d@%s", f.n, i, s.Domain)
			}
			contacts = append(contacts, []any{id, f.WS, email, "Reader", fmt.Sprint(i), status, tier, s.Source, mbx, checked, lastOpen, lastOpen, s.Role})
			members = append(members, []any{f.List, id, "subscribed", f.Start.Add(-30 * 24 * time.Hour)})
			if s.Suppress {
				f.Exec(`INSERT INTO suppressions (id, workspace_id, email, reason) VALUES ($1, $2, $3, 'hard_bounce')`, uuid.Must(uuid.NewV7()), f.WS, email)
			}
			if s.Paused {
				paused = append(paused, id)
			}
			if s.Stopped || s.StoppedPair {
				stopped = append(stopped, id)
			}
			if s.StoppedPair {
				aid := uuid.Must(uuid.NewV7())
				contacts = append(contacts, []any{aid, f.WS, strings.Replace(email, "@", "+pair@", 1), "Reader", fmt.Sprint(i), status, tier, s.Source, mbx, checked, lastOpen, lastOpen, s.Role})
				members = append(members, []any{f.List, aid, "subscribed", f.Start.Add(-30 * 24 * time.Hour)})
			}
		}
	}
	if err := pgxCopy(f, "contacts", []string{"id", "workspace_id", "email", "first_name", "last_name", "status", "tier", "source", "mailbox_provider",
		"mailbox_checked_at", "last_open_at", "last_human_open_at", "role_address"}, contacts); err != nil {
		f.T.Fatal(err)
	}
	if err := pgxCopy(f, "list_memberships", []string{"list_id", "contact_id", "status", "subscribed_at"}, members); err != nil {
		f.T.Fatal(err)
	}
	for _, id := range paused {
		f.Exec(`INSERT INTO contact_preferences (contact_id, paused_until) VALUES ($1, $2)`, id, f.Start.Add(24*time.Hour))
	}
	if len(stopped) > 0 {
		f.Exec(`UPDATE contacts SET broadcasts_stopped_at = $2 WHERE id = ANY($1)`, stopped, f.Start.Add(-24*time.Hour))
	}
}

func pgxCopy(f *Fixture, table string, cols []string, rows [][]any) error {
	conn, err := f.Env.App.Pool.Acquire(f.Ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	_, err = conn.CopyFrom(f.Ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows))
	return err
}

// StartSend starts the send now and builds its audience.
func (f *Fixture) StartSend(o engine.StartOptions) {
	f.T.Helper()
	if o.By == uuid.Nil {
		o.By = f.Sofia
	}
	o.Now = true
	if err := f.E.StartWith(f.Ctx, f.Campaign, o); err != nil {
		f.T.Fatalf("start: %v", err)
	}
	if err := f.E.DrivePrepare(f.Ctx, f.Campaign, f.Start); err != nil {
		f.T.Fatalf("prepare: %v", err)
	}
}

// Dump logs the send's state when a test fails.
func (f *Fixture) Dump() {
	f.T.Helper()
	rows, _ := f.Env.App.Pool.Query(f.Ctx, `SELECT p.adapter, m.status, count(*), max(m.attempts), max(m.error), max(m.next_attempt_at)
		FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.campaign_id = $1 GROUP BY 1, 2`, f.Campaign)
	for rows.Next() {
		var a, st, e string
		var n, att int64
		var next *time.Time
		_ = rows.Scan(&a, &st, &n, &att, &e, &next)
		f.T.Logf("messages %s %s %d attempts<=%d next=%v err=%q", a, st, n, att, next, e)
	}
	rows.Close()
	rows, _ = f.Env.App.Pool.Query(f.Ctx, `SELECT p.adapter, cr.status, cr.status_detail, cr.planned FROM campaign_routes cr JOIN providers p ON p.id = cr.provider_id WHERE cr.campaign_id = $1`, f.Campaign)
	for rows.Next() {
		var a, st, d string
		var n int64
		_ = rows.Scan(&a, &st, &d, &n)
		f.T.Logf("route %s %s %s planned=%d", a, st, d, n)
	}
	rows.Close()
	rows, _ = f.Env.App.Pool.Query(f.Ctx, `SELECT kind, scope, mailbox_provider, metric, held, resolution FROM campaign_incidents WHERE campaign_id = $1`, f.Campaign)
	for rows.Next() {
		var k, sc, mb, me, res string
		var held int64
		_ = rows.Scan(&k, &sc, &mb, &me, &held, &res)
		f.T.Logf("incident %s %s %s %s held=%d resolution=%q", k, sc, mb, me, held, res)
	}
	rows.Close()
	var holds, ab, phase, status string
	_ = f.Env.App.Pool.QueryRow(f.Ctx, `SELECT s.holds::text, s.ab::text, s.phase, c.status FROM campaign_sends s JOIN campaigns c ON c.id = s.campaign_id WHERE s.campaign_id = $1`, f.Campaign).Scan(&holds, &ab, &phase, &status)
	f.T.Logf("campaign %s, send phase=%s holds=%s ab=%s", status, phase, holds, ab)
}

// Status is the campaign's status.
func (f *Fixture) Status() string {
	f.T.Helper()
	var s string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT status FROM campaigns WHERE id = $1`, f.Campaign).Scan(&s); err != nil {
		f.T.Fatal(err)
	}
	return s
}

// ProviderID is the workspace's connected provider of an adapter.
func (f *Fixture) ProviderID(adapter string) uuid.UUID {
	f.T.Helper()
	var id uuid.UUID
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = $2 AND disconnected_at IS NULL ORDER BY connected_at LIMIT 1`, f.WS, adapter).Scan(&id); err != nil {
		f.T.Fatal(err)
	}
	return id
}

// Drive runs the send for d of virtual time from the fixture's start and
// returns where it stopped and whether the send ended.
func (f *Fixture) Drive(d time.Duration) (time.Time, bool) {
	f.T.Helper()
	at, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, d)
	if err != nil {
		f.T.Fatalf("drive: %v", err)
	}
	return at, done
}
