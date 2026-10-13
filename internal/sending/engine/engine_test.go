package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
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

// contactSpec describes a batch of test contacts.
type contactSpec struct {
	n        int
	mailbox  string
	domain   string
	tier     string
	source   string
	engaged  bool
	status   string
	suppress bool
	paused   bool
	role     bool
}

type suite struct {
	env   *apptest.Env
	e     *engine.Engine
	ctx   context.Context
	ws    uuid.UUID
	sofia uuid.UUID
	start time.Time
	n     int
}

type fixture struct {
	*suite
	t        *testing.T
	list     uuid.UUID
	campaign uuid.UUID
}

func newSuite(t *testing.T) *suite {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	s := &suite{env: env, e: env.App.SendingEngine(), ctx: context.Background()}
	if err := env.App.Pool.QueryRow(s.ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&s.ws); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(s.ctx, `SELECT id FROM users WHERE email = 'sofia@acme.io'`).Scan(&s.sofia); err != nil {
		t.Fatal(err)
	}
	s.start = time.Now().Truncate(time.Second)
	start := s.start
	s.e.Now = func() time.Time { return start }
	return s
}

// fixture makes a copy of Weekly Digest #142 (content, sender, A/B test)
// that sends to a list of the test's own contacts.
func (s *suite) fixture(t *testing.T, name string) *fixture {
	t.Helper()
	s.n++
	f := &fixture{suite: s, t: t, list: uuid.Must(uuid.NewV7()), campaign: uuid.Must(uuid.NewV7())}
	f.exec(`DELETE FROM sending_rate_windows`)
	f.exec(`INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, $3)`, f.list, s.ws, "Send test "+name)
	f.exec(`INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, reply_to, audience, editor_mode, mjml, html, plain_text,
		ab_test, schedule, thresholds, tracking, created_by, updated_by, content_hash)
		SELECT $1, workspace_id, $2, 'draft', subject, preheader, sender_id, reply_to, $3, editor_mode, mjml, html, plain_text,
		ab_test, '{"mode":"now"}', thresholds, tracking, created_by, updated_by, content_hash
		FROM campaigns WHERE workspace_id = $4 AND name = 'Weekly Digest #142'`, f.campaign, name, fmt.Sprintf(`{"lists":[%q]}`, f.list), s.ws)
	return f
}

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.env.App.Pool.Exec(f.ctx, sql, args...); err != nil {
		f.t.Fatalf("%s: %v", sql, err)
	}
}

func (f *fixture) count(sql string, args ...any) int64 {
	f.t.Helper()
	var n int64
	if err := f.env.App.Pool.QueryRow(f.ctx, sql, args...).Scan(&n); err != nil {
		f.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

// addContacts inserts contacts in the fixture's list.
func (f *fixture) addContacts(specs ...contactSpec) {
	f.t.Helper()
	var contacts, members [][]any
	var paused []uuid.UUID
	i := 0
	for _, s := range specs {
		for range s.n {
			i++
			id := uuid.Must(uuid.NewV7())
			status := s.status
			if status == "" {
				status = "active"
			}
			tier := s.tier
			if tier == "" {
				tier = "active"
			}
			var lastOpen *time.Time
			if s.engaged {
				t := f.start.Add(-48 * time.Hour)
				lastOpen = &t
			}
			email := fmt.Sprintf("w12.%d.%05d@%s", f.n, i, s.domain)
			contacts = append(contacts, []any{id, f.ws, email, "Reader", fmt.Sprint(i), status, tier, s.source, s.mailbox, lastOpen, lastOpen, s.role})
			members = append(members, []any{f.list, id, "subscribed", f.start.Add(-30 * 24 * time.Hour)})
			if s.suppress {
				f.exec(`INSERT INTO suppressions (id, workspace_id, email, reason) VALUES ($1, $2, $3, 'hard_bounce')`, uuid.Must(uuid.NewV7()), f.ws, email)
			}
			if s.paused {
				paused = append(paused, id)
			}
		}
	}
	if err := pgxCopy(f, "contacts", []string{"id", "workspace_id", "email", "first_name", "last_name", "status", "tier", "source", "mailbox_provider",
		"last_open_at", "last_human_open_at", "role_address"}, contacts); err != nil {
		f.t.Fatal(err)
	}
	if err := pgxCopy(f, "list_memberships", []string{"list_id", "contact_id", "status", "subscribed_at"}, members); err != nil {
		f.t.Fatal(err)
	}
	for _, id := range paused {
		f.exec(`INSERT INTO contact_preferences (contact_id, paused_until) VALUES ($1, $2)`, id, f.start.Add(24*time.Hour))
	}
}

func pgxCopy(f *fixture, table string, cols []string, rows [][]any) error {
	conn, err := f.env.App.Pool.Acquire(f.ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	_, err = conn.CopyFrom(f.ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows))
	return err
}

func (f *fixture) startSend(o engine.StartOptions) {
	f.t.Helper()
	if o.By == uuid.Nil {
		o.By = f.sofia
	}
	o.Now = true
	if err := f.e.StartWith(f.ctx, f.campaign, o); err != nil {
		f.t.Fatalf("start: %v", err)
	}
	if err := f.e.DrivePrepare(f.ctx, f.campaign, f.start); err != nil {
		f.t.Fatalf("prepare: %v", err)
	}
}

// dump logs the send's state when a test fails.
func (f *fixture) dump() {
	f.t.Helper()
	rows, _ := f.env.App.Pool.Query(f.ctx, `SELECT p.adapter, m.status, count(*), max(m.attempts), max(m.error), max(m.next_attempt_at)
		FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.campaign_id = $1 GROUP BY 1, 2`, f.campaign)
	for rows.Next() {
		var a, st, e string
		var n, att int64
		var next *time.Time
		_ = rows.Scan(&a, &st, &n, &att, &e, &next)
		f.t.Logf("messages %s %s %d attempts<=%d next=%v err=%q", a, st, n, att, next, e)
	}
	rows.Close()
	rows, _ = f.env.App.Pool.Query(f.ctx, `SELECT p.adapter, cr.status, cr.status_detail, cr.planned FROM campaign_routes cr JOIN providers p ON p.id = cr.provider_id WHERE cr.campaign_id = $1`, f.campaign)
	for rows.Next() {
		var a, st, d string
		var n int64
		_ = rows.Scan(&a, &st, &d, &n)
		f.t.Logf("route %s %s %s planned=%d", a, st, d, n)
	}
	rows.Close()
	rows, _ = f.env.App.Pool.Query(f.ctx, `SELECT kind, scope, mailbox_provider, metric, held, resolution FROM campaign_incidents WHERE campaign_id = $1`, f.campaign)
	for rows.Next() {
		var k, sc, mb, me, res string
		var held int64
		_ = rows.Scan(&k, &sc, &mb, &me, &held, &res)
		f.t.Logf("incident %s %s %s %s held=%d resolution=%q", k, sc, mb, me, held, res)
	}
	rows.Close()
	var holds, ab, phase, status string
	_ = f.env.App.Pool.QueryRow(f.ctx, `SELECT s.holds::text, s.ab::text, s.phase, c.status FROM campaign_sends s JOIN campaigns c ON c.id = s.campaign_id WHERE s.campaign_id = $1`, f.campaign).Scan(&holds, &ab, &phase, &status)
	f.t.Logf("campaign %s, send phase=%s holds=%s ab=%s", status, phase, holds, ab)
}

func (f *fixture) status() string {
	f.t.Helper()
	var s string
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT status FROM campaigns WHERE id = $1`, f.campaign).Scan(&s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func (f *fixture) providerID(adapter string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = $2 AND disconnected_at IS NULL ORDER BY connected_at LIMIT 1`, f.ws, adapter).Scan(&id); err != nil {
		f.t.Fatal(err)
	}
	return id
}

func TestSending(t *testing.T) {
	s := newSuite(t)
	t.Run("EndToEnd", func(t *testing.T) { testEndToEnd(t, s.fixture(t, "End to end")) })
	t.Run("AutoPauseOutlookComplaints", func(t *testing.T) { testAutoPause(t, s.fixture(t, "Auto-pause")) })
	t.Run("PostalOutageHoldsShare", func(t *testing.T) { testOutage(t, s.fixture(t, "Outage")) })
	t.Run("PauseResumeCancel", func(t *testing.T) { testPauseResumeCancel(t, s.fixture(t, "Pause")) })
	t.Run("ScheduledStart", func(t *testing.T) { testScheduled(t, s.fixture(t, "Scheduled")) })
	t.Run("API", func(t *testing.T) { testAPI(t, s.fixture(t, "API")) })
	t.Run("Tracking", func(t *testing.T) { testTracking(t, s.fixture(t, "Tracking")) })
	t.Run("DemoSendReset", func(t *testing.T) { testDemoReset(t, s.fixture(t, "Demo reset")) })
	t.Run("BestTime", func(t *testing.T) { testBestTime(t, s.fixture(t, "Best time")) })
}

// testBestTime sends at each recipient's best time: a contact who usually
// opens at a given hour gets the send then (within 24 hours), one without
// opens at the start; a send scheduled that way waits for its time.
func testBestTime(t *testing.T, f *fixture) {
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(contactSpec{n: 6, mailbox: "gmail", domain: "gmail.com", engaged: true})
	hour := (f.start.UTC().Hour() + 3) % 24
	// Three of them opened at that hour on several days (one machine fetch
	// at another hour doesn't count).
	f.exec(`INSERT INTO delivery_events (id, workspace_id, contact_id, type, occurred_at, machine)
		SELECT gen_random_uuid(), c.workspace_id, c.id, 'open', date_trunc('day', $2::timestamptz) - (d || ' days')::interval + make_interval(hours => $3) + interval '7 minutes', false
		FROM (SELECT c.* FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1 ORDER BY c.id LIMIT 3) c, generate_series(1, 4) d`,
		f.list, f.start, hour)
	f.exec(`INSERT INTO delivery_events (id, workspace_id, contact_id, type, occurred_at, machine)
		SELECT gen_random_uuid(), c.workspace_id, c.id, 'open', $2::timestamptz - interval '2 days', true
		FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1`, f.list, f.start)

	later := f.start.Add(36 * time.Hour)
	f.exec(`UPDATE campaigns SET schedule = $2 WHERE id = $1`, f.campaign, fmt.Sprintf(`{"mode":"best_time","at":%q,"timezone":"UTC"}`, later.Format(time.RFC3339)))
	if err := f.e.Start(f.ctx, f.campaign, f.sofia); err != nil {
		t.Fatal(err)
	}
	if f.status() != "scheduled" {
		t.Fatalf("a best-time send for later is %s, want scheduled", f.status())
	}
	f.startSend(engine.StartOptions{})
	want := f.start.UTC().Truncate(24 * time.Hour).Add(time.Duration(hour) * time.Hour)
	if want.Before(f.start) {
		want = want.Add(24 * time.Hour)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND next_attempt_at = $2`, f.campaign, want); n != 3 {
		f.dump()
		t.Fatalf("%d recipients due at their usual hour %v, want 3", n, want)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND next_attempt_at IS NULL`, f.campaign); n != 3 {
		t.Fatalf("%d recipients without opens go at the start, want 3", n)
	}
	snap, err := f.e.Snapshot(f.ctx, f.ws, f.campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.FinishesAt == nil || !snap.FinishesAt.Equal(want) {
		t.Fatalf("the monitor's finish time %v, want %v (the last recipient's hour)", snap.FinishesAt, want)
	}
	if _, done, err := f.e.Drive(f.ctx, f.campaign, f.start, want.Sub(f.start)+time.Minute); err != nil || !done {
		t.Fatalf("the send didn't finish at the last recipient's hour: done=%v err=%v", done, err)
	}
}

// testDemoReset runs the demo command's send, then starts it again with
// --reset: what the first send added outside the campaign (provider daily
// stats, suppressions from its bounces, the bounced contacts' status) is
// taken back, so each run starts from the same demo.
func testDemoReset(t *testing.T, f *fixture) {
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(contactSpec{n: 2000, mailbox: "gmail", domain: "gmail.com", engaged: true})
	stats := func() int64 {
		return f.count(`SELECT coalesce(sum(d.sent + d.delivered + d.bounced + d.complained), 0) FROM provider_daily_stats d
			JOIN providers p ON p.id = d.provider_id WHERE p.workspace_id = $1`, f.ws)
	}
	suppressed := func() int64 { return f.count(`SELECT count(*) FROM suppressions WHERE workspace_id = $1`, f.ws) }
	inactive := func() int64 {
		return f.count(`SELECT count(*) FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1 AND c.status <> 'active'`, f.list)
	}
	baseStats, baseSuppressed := stats(), suppressed()
	res, err := f.e.DemoSend(f.ctx, engine.DemoSendOptions{Campaign: "Demo reset", Elapsed: 3 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if res.Recipients != 2000 || res.Sent == 0 {
		t.Fatalf("demo send: %d recipients, %d sent", res.Recipients, res.Sent)
	}
	if stats() <= baseStats {
		t.Fatal("the demo send didn't count in its providers' daily stats")
	}
	bounced := inactive()
	if bounced == 0 || suppressed() != baseSuppressed+bounced {
		t.Fatalf("%d contacts bounced or complained, suppressions went from %d to %d", bounced, baseSuppressed, suppressed())
	}
	if _, err := f.e.DemoSend(f.ctx, engine.DemoSendOptions{Campaign: "Demo reset", Reset: true}); err != nil {
		t.Fatal(err)
	}
	if got := stats(); got != baseStats {
		t.Fatalf("provider daily stats %d after the reset, want %d", got, baseStats)
	}
	if got := suppressed(); got != baseSuppressed {
		t.Fatalf("%d suppressions after the reset, want %d", got, baseSuppressed)
	}
	if n := inactive(); n != 0 {
		t.Fatalf("%d contacts still bounced or complained after the reset", n)
	}
	if n := f.count(`SELECT recipients FROM campaign_sends WHERE campaign_id = $1`, f.campaign); n != 2000 {
		t.Fatalf("the send started again for %d recipients, want all 2,000", n)
	}
	if err := f.e.Cancel(f.ctx, f.campaign, f.sofia); err != nil {
		t.Fatal(err)
	}
}

func testEndToEnd(t *testing.T, f *fixture) {
	f.addContacts(
		contactSpec{n: 400, mailbox: "gmail", domain: "gmail.com", engaged: true},
		contactSpec{n: 150, mailbox: "outlook", domain: "outlook.com", engaged: true},
		contactSpec{n: 60, mailbox: "yahoo", domain: "yahoo.com", engaged: true},
		contactSpec{n: 50, mailbox: "icloud", domain: "icloud.com", engaged: true},
		contactSpec{n: 40, mailbox: "other", domain: "example.org", engaged: true},
		contactSpec{n: 5, mailbox: "gmail", domain: "gmail.com", suppress: true},
		contactSpec{n: 3, mailbox: "gmail", domain: "gmail.com", status: "unsubscribed"},
		contactSpec{n: 2, mailbox: "gmail", domain: "gmail.com", paused: true},
		contactSpec{n: 4, mailbox: "other", domain: "example.org", role: true},
	)
	f.startSend(engine.StartOptions{ABWindow: 2 * time.Minute})

	if s := f.status(); s != "sending" {
		t.Fatalf("status %q after start, want sending", s)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1`, f.campaign); n != 700 {
		t.Fatalf("%d messages queued, want 700 (suppressed, unsubscribed, paused and role addresses left out)", n)
	}
	var excluded engine.Excluded
	var raw []byte
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT excluded FROM campaign_sends WHERE campaign_id = $1`, f.campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(raw, &excluded)
	if excluded.Suppressed != 5 || excluded.Unsubscribed != 3 || excluded.Paused != 2 || excluded.Role != 4 {
		t.Fatalf("excluded %+v, want 5 suppressed, 3 unsubscribed, 2 paused, 4 role addresses", excluded)
	}
	// Outlook and iCloud go to SES only (the demo's routing condition).
	if n := f.count(`SELECT count(*) FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.campaign_id = $1 AND m.mailbox_provider IN ('outlook','icloud') AND p.adapter <> 'ses'`, f.campaign); n != 0 {
		t.Fatalf("%d Outlook/iCloud recipients routed away from SES", n)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND route_note LIKE 'Postal share of the 90/10 split, warm-up day %'`, f.campaign); n == 0 {
		t.Fatal("no route notes on the Postal share")
	}
	if n := f.count(`SELECT coalesce(sum(planned), 0) FROM campaign_routes WHERE campaign_id = $1`, f.campaign); n != 700 {
		t.Fatalf("routes plan %d, want 700", n)
	}
	if n := f.count(`SELECT count(*) FROM campaign_links WHERE campaign_id = $1`, f.campaign); n == 0 {
		t.Fatal("no tracked links indexed")
	}
	// The A/B sample goes first; the rest waits for the winner.
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held'`, f.campaign); n == 0 || n >= 700 {
		t.Fatalf("%d held for the A/B winner", n)
	}

	_, done, err := f.e.Drive(f.ctx, f.campaign, f.start, 25*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.status() != "sent" {
		f.dump()
		t.Fatalf("send didn't finish: done=%v status=%s", done, f.status())
	}
	if n := f.count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1`, f.campaign); n != 700 {
		t.Fatalf("counters say %d sent, want 700", n)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.campaign); n != 700 {
		t.Fatalf("%d messages have sent_at, want 700", n)
	}
	if n := f.count(`SELECT count(DISTINCT provider_message_id) FROM messages WHERE campaign_id = $1`, f.campaign); n != 700 {
		t.Fatalf("%d distinct provider message ids, want 700 (nothing sent twice)", n)
	}
	if n := f.count(`SELECT coalesce(sum(delivered), 0) FROM campaign_counters WHERE campaign_id = $1`, f.campaign); n < 650 {
		t.Fatalf("only %d delivered", n)
	}
	if n := f.count(`SELECT coalesce(sum(opened_human + opened_machine), 0) FROM campaign_counters WHERE campaign_id = $1`, f.campaign); n == 0 {
		t.Fatal("no opens recorded")
	}
	if n := f.count(`SELECT count(*) FROM delivery_events WHERE campaign_id = $1 AND type = 'open' AND machine`, f.campaign); n == 0 {
		t.Fatal("no machine opens classified (Apple Mail Privacy Protection)")
	}
	var winner string
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT ab->>'winner' FROM campaign_sends WHERE campaign_id = $1`, f.campaign).Scan(&winner); err != nil || winner == "" {
		t.Fatalf("A/B winner %q: %v", winner, err)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('campaign.send_start', 'campaign.send_finish', 'campaign.ab_winner')`, f.campaign.String()); n != 3 {
		t.Fatalf("%d audit entries for start, A/B winner and finish, want 3", n)
	}
	if n := f.count(`SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = 'send_finished' AND data->>'campaignId' = $2`, f.sofia, f.campaign.String()); n != 1 {
		t.Fatalf("%d send_finished notifications for the starter", n)
	}
	snap, err := f.e.Snapshot(f.ctx, f.ws, f.campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.State != "finished" || snap.Sent != 700 || snap.StartedBy != "Sofia Rossi" {
		t.Fatalf("snapshot %s sent=%d by %q", snap.State, snap.Sent, snap.StartedBy)
	}
	if len(snap.Mailboxes) != 5 || len(snap.Providers) != 2 {
		t.Fatalf("snapshot rows: %d mailbox, %d provider", len(snap.Mailboxes), len(snap.Providers))
	}
	for _, ev := range snap.Events {
		if ev.Address == "" || len(ev.Address) > 40 || !containsDots(ev.Address) {
			t.Fatalf("event address not masked: %q", ev.Address)
		}
	}
}

func containsDots(s string) bool {
	for _, r := range s {
		if r == '·' {
			return true
		}
	}
	return false
}

func testAutoPause(t *testing.T, f *fixture) {
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(
		// Outlook sends at 5 /s, so 2,500 take over eight minutes: complaints
		// arrive one to six minutes after delivery and trip the threshold
		// while most of the share is still queued.
		contactSpec{n: 2500, mailbox: "outlook", domain: "hotmail.com", tier: "at_risk", source: "import:mailchimp"},
		contactSpec{n: 600, mailbox: "gmail", domain: "gmail.com", engaged: true},
	)
	f.startSend(engine.StartOptions{Scenario: "outlook-complaints"})
	if _, _, err := f.e.Drive(f.ctx, f.campaign, f.start, 20*time.Minute); err != nil {
		t.Fatal(err)
	}
	var kind, scope, mailbox, metric string
	var value, limit float64
	var held int64
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT kind, scope, mailbox_provider, metric, value, threshold, held FROM campaign_incidents WHERE campaign_id = $1 ORDER BY tripped_at LIMIT 1`, f.campaign).
		Scan(&kind, &scope, &mailbox, &metric, &value, &limit, &held); err != nil {
		f.dump()
		t.Fatalf("no incident: %v", err)
	}
	if kind != "threshold" || scope != "mailbox_provider" || mailbox != "outlook" || metric != "complaint" || value <= limit {
		f.dump()
		t.Fatalf("incident %s %s %s %s %.2f/%.2f", kind, scope, mailbox, metric, value, limit)
	}
	if held == 0 {
		t.Fatal("nothing held")
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'gmail' AND status IN ('queued','held')`, f.campaign); n != 0 {
		f.dump()
		t.Fatalf("%d Gmail recipients left; the rest of the campaign should have finished", n)
	}
	snap, err := f.e.Snapshot(f.ctx, f.ws, f.campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incident == nil || snap.Incident.Detail.Changes == nil || len(snap.Incident.Detail.Options) != 3 {
		t.Fatalf("snapshot incident %+v", snap.Incident)
	}
	if len(snap.Incident.Detail.Causes) == 0 || snap.Incident.Detail.Note == "" {
		t.Fatal("incident has no likely causes or note")
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.auto_pause'`, f.campaign.String()); n != 1 {
		t.Fatalf("%d auto-pause audit entries", n)
	}
	if n := f.count(`SELECT count(*) FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1`, f.campaign.String()); n == 0 {
		t.Fatal("nobody was told about the auto-pause")
	}
	res, err := f.e.ResumeScope(f.ctx, f.campaign, f.sofia, engine.ResumeRequest{Mailbox: "outlook", WithChanges: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Excluded == 0 || res.Rate == 0 {
		t.Fatalf("resume with changes excluded %d at %.0f /s", res.Excluded, res.Rate)
	}
	if n := f.count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND resolution = 'resumed_with_changes'`, f.campaign); n != 1 {
		t.Fatal("incident not resolved as resumed with changes")
	}
	_, done, err := f.e.Drive(f.ctx, f.campaign, f.start.Add(20*time.Minute), 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		f.dump()
		t.Fatalf("send didn't finish after resuming: %s", f.status())
	}
}

func testOutage(t *testing.T, f *fixture) {
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(contactSpec{n: 3000, mailbox: "gmail", domain: "gmail.com", engaged: true})
	f.startSend(engine.StartOptions{Scenario: "postal-outage"})
	// The scenario takes Postal down 14 minutes into the send; start the
	// clock 13 minutes in.
	f.exec(`UPDATE campaigns SET started_at = $2 WHERE id = $1`, f.campaign, f.start.Add(-13*time.Minute))
	if _, _, err := f.e.Drive(f.ctx, f.campaign, f.start, 8*time.Minute); err != nil {
		t.Fatal(err)
	}
	var routeStatus string
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT cr.status FROM campaign_routes cr JOIN providers p ON p.id = cr.provider_id WHERE cr.campaign_id = $1 AND p.adapter = 'postal'`, f.campaign).Scan(&routeStatus); err != nil {
		t.Fatal(err)
	}
	if routeStatus != "held" {
		f.dump()
		t.Fatalf("Postal route is %s, want held", routeStatus)
	}
	snap, err := f.e.Snapshot(f.ctx, f.ws, f.campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incident == nil || snap.Incident.Kind != "provider_outage" || snap.Incident.Detail.ErrorKind != "connection errors" {
		t.Fatalf("incident %+v", snap.Incident)
	}
	d := snap.Incident.Detail
	if d.Move == nil || d.Move.TargetShort != "SES" || d.Policy != "pause_and_ask" || len(d.Log) == 0 || d.Remaining == 0 {
		t.Fatalf("outage detail %+v", d)
	}
	if f.status() != "paused" {
		f.dump()
		t.Fatalf("status %s; with only the held share left the campaign waits for a decision", f.status())
	}
	moved, err := f.e.MoveShare(f.ctx, f.campaign, f.sofia, f.providerID("postal"), f.providerID("ses"))
	if err != nil {
		t.Fatal(err)
	}
	if moved != d.Remaining {
		t.Fatalf("moved %d, the incident held %d", moved, d.Remaining)
	}
	_, done, err := f.e.Drive(f.ctx, f.campaign, f.start.Add(8*time.Minute), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.status() != "sent" {
		f.dump()
		t.Fatalf("send didn't finish after the move: done=%v %s", done, f.status())
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('campaign.route_hold', 'campaign.share_move')`, f.campaign.String()); n != 2 {
		t.Fatalf("%d audit entries for hold and move, want 2", n)
	}
}

func testPauseResumeCancel(t *testing.T, f *fixture) {
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(contactSpec{n: 600, mailbox: "gmail", domain: "gmail.com", engaged: true})
	f.startSend(engine.StartOptions{})
	if _, _, err := f.e.Drive(f.ctx, f.campaign, f.start, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := f.e.Pause(f.ctx, f.campaign, f.sofia, "Checking a typo"); err != nil {
		t.Fatal(err)
	}
	sentAtPause := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.campaign)
	if _, done, err := f.e.Drive(f.ctx, f.campaign, f.start.Add(10*time.Second), 30*time.Second); err != nil || !done {
		t.Fatalf("a paused send kept running: done=%v err=%v", done, err)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.campaign); n != sentAtPause {
		t.Fatalf("%d sent while paused", n-sentAtPause)
	}
	snap, err := f.e.Snapshot(f.ctx, f.ws, f.campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.State != "paused" || snap.PausedBy != "Sofia Rossi" || snap.PauseReason != "Checking a typo" {
		t.Fatalf("paused snapshot %s by %q (%q)", snap.State, snap.PausedBy, snap.PauseReason)
	}
	if err := f.e.Resume(f.ctx, f.campaign, f.sofia); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.e.Drive(f.ctx, f.campaign, f.start.Add(40*time.Second), 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := f.e.Cancel(f.ctx, f.campaign, f.sofia); err != nil {
		t.Fatal(err)
	}
	if f.status() != "cancelled" {
		t.Fatalf("status %s after cancel", f.status())
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status IN ('queued', 'held')`, f.campaign); n != 0 {
		t.Fatalf("%d messages still queued after cancel", n)
	}
	for _, action := range []string{"campaign.send_pause", "campaign.send_resume", "campaign.send_cancel"} {
		if n := f.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = $2`, f.campaign.String(), action); n != 1 {
			t.Fatalf("%d %s audit entries", n, action)
		}
	}
	if err := f.e.Pause(f.ctx, f.campaign, f.sofia, ""); err == nil {
		t.Fatal("pausing a cancelled send must fail")
	}
}

func testScheduled(t *testing.T, f *fixture) {
	f.addContacts(contactSpec{n: 20, mailbox: "gmail", domain: "gmail.com", engaged: true})
	at := f.start.Add(48 * time.Hour)
	f.exec(`UPDATE campaigns SET schedule = $2 WHERE id = $1`, f.campaign, fmt.Sprintf(`{"mode":"at","at":%q,"timezone":"America/New_York"}`, at.Format(time.RFC3339)))
	if err := f.e.Start(f.ctx, f.campaign, f.sofia); err != nil {
		t.Fatal(err)
	}
	if f.status() != "scheduled" {
		t.Fatalf("status %s, want scheduled", f.status())
	}
	if n := f.count(`SELECT count(*) FROM river_job WHERE kind = 'sending.start' AND args->>'campaign_id' = $1 AND scheduled_at = $2`, f.campaign.String(), at); n != 1 {
		t.Fatalf("%d start jobs at the scheduled time", n)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.send_schedule'`, f.campaign.String()); n != 1 {
		t.Fatal("scheduling wasn't audited")
	}
	if err := f.e.Cancel(f.ctx, f.campaign, f.sofia); err != nil {
		t.Fatal(err)
	}
	if f.status() != "draft" {
		t.Fatalf("unscheduling left %s, want draft", f.status())
	}
}
