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
}

type fixture struct {
	t        *testing.T
	env      *apptest.Env
	e        *engine.Engine
	ctx      context.Context
	ws       uuid.UUID
	digest   uuid.UUID
	campaign uuid.UUID
	sofia    uuid.UUID
	start    time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	f := &fixture{t: t, env: env, e: env.App.SendingEngine(), ctx: context.Background()}
	q := func(sql string, args ...any) uuid.UUID {
		var id uuid.UUID
		if err := env.App.Pool.QueryRow(f.ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return id
	}
	f.ws = q(`SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`)
	f.digest = q(`SELECT id FROM lists WHERE workspace_id = $1 AND name = 'Weekly Digest'`, f.ws)
	f.campaign = q(`SELECT id FROM campaigns WHERE workspace_id = $1 AND name = 'Weekly Digest #142'`, f.ws)
	f.sofia = q(`SELECT id FROM users WHERE email = 'sofia@acme.io'`)
	f.start = time.Now().Truncate(time.Second)
	start := f.start
	f.e.Now = func() time.Time { return start }
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

// addContacts inserts contacts in the Weekly Digest list.
func (f *fixture) addContacts(specs ...contactSpec) {
	f.t.Helper()
	var contacts, members [][]any
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
			email := fmt.Sprintf("reader.%05d@%s", i, s.domain)
			contacts = append(contacts, []any{id, f.ws, email, "Reader", fmt.Sprint(i), status, tier, s.source, s.mailbox, lastOpen})
			members = append(members, []any{f.digest, id, "subscribed", f.start.Add(-30 * 24 * time.Hour)})
			if s.suppress {
				f.exec(`INSERT INTO suppressions (id, workspace_id, email, reason) VALUES ($1, $2, $3, 'hard_bounce')`, uuid.Must(uuid.NewV7()), f.ws, email)
			}
			if s.paused {
				defer f.exec(`INSERT INTO contact_preferences (contact_id, paused_until) VALUES ($1, $2)`, id, f.start.Add(24*time.Hour))
			}
		}
	}
	err := pgxCopy(f, "contacts", []string{"id", "workspace_id", "email", "first_name", "last_name", "status", "tier", "source", "mailbox_provider", "last_open_at"}, contacts)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := pgxCopy(f, "list_memberships", []string{"list_id", "contact_id", "status", "subscribed_at"}, members); err != nil {
		f.t.Fatal(err)
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

func (f *fixture) status() string {
	f.t.Helper()
	var s string
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT status FROM campaigns WHERE id = $1`, f.campaign).Scan(&s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func TestSendEndToEnd(t *testing.T) {
	f := newFixture(t)
	f.addContacts(
		contactSpec{n: 400, mailbox: "gmail", domain: "gmail.com", engaged: true},
		contactSpec{n: 150, mailbox: "outlook", domain: "outlook.com", engaged: true},
		contactSpec{n: 60, mailbox: "yahoo", domain: "yahoo.com", engaged: true},
		contactSpec{n: 50, mailbox: "icloud", domain: "icloud.com", engaged: true},
		contactSpec{n: 40, mailbox: "other", domain: "example.org", engaged: true},
		contactSpec{n: 5, mailbox: "gmail", domain: "gmail.com", suppress: true},
		contactSpec{n: 3, mailbox: "gmail", domain: "gmail.com", status: "unsubscribed"},
		contactSpec{n: 2, mailbox: "gmail", domain: "gmail.com", paused: true},
	)
	f.startSend(engine.StartOptions{ABWindow: 2 * time.Minute})

	if s := f.status(); s != "sending" {
		t.Fatalf("status %q after start, want sending", s)
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1`, f.campaign); n != 700 {
		t.Fatalf("%d messages queued, want 700 (suppressed, unsubscribed and paused contacts left out)", n)
	}
	var excluded engine.Excluded
	var raw []byte
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT excluded FROM campaign_sends WHERE campaign_id = $1`, f.campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(raw, &excluded)
	if excluded.Suppressed != 5 || excluded.Unsubscribed != 3 || excluded.Paused != 2 {
		t.Fatalf("excluded %+v, want 5 suppressed, 3 unsubscribed, 2 paused", excluded)
	}
	// Outlook and iCloud go to SES only (the demo's routing condition).
	if n := f.count(`SELECT count(*) FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.campaign_id = $1 AND m.mailbox_provider IN ('outlook','icloud') AND p.adapter <> 'ses'`, f.campaign); n != 0 {
		t.Fatalf("%d Outlook/iCloud recipients routed away from SES", n)
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
		t.Fatalf("send didn't finish: done=%v status=%s", done, f.status())
	}
	sent := f.count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1`, f.campaign)
	if sent != 700 {
		t.Fatalf("counters say %d sent, want 700", sent)
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
	if n := f.count(`SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = 'send_finished'`, f.sofia); n != 1 {
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
		if ev.Address == "" || len(ev.Address) > 30 {
			t.Fatalf("event address not masked: %q", ev.Address)
		}
	}
}

func TestAutoPauseOutlookComplaints(t *testing.T) {
	f := newFixture(t)
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(
		contactSpec{n: 1400, mailbox: "outlook", domain: "hotmail.com", tier: "at_risk", source: "import:mailchimp"},
		contactSpec{n: 900, mailbox: "gmail", domain: "gmail.com", engaged: true},
	)
	f.startSend(engine.StartOptions{Scenario: "outlook-complaints"})
	_, _, err := f.e.Drive(f.ctx, f.campaign, f.start, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var kind, scope, mailbox, metric string
	var value, limit float64
	var held int64
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT kind, scope, mailbox_provider, metric, value, threshold, held FROM campaign_incidents WHERE campaign_id = $1 ORDER BY tripped_at LIMIT 1`, f.campaign).
		Scan(&kind, &scope, &mailbox, &metric, &value, &limit, &held); err != nil {
		t.Fatalf("no incident: %v", err)
	}
	if kind != "threshold" || scope != "mailbox_provider" || mailbox != "outlook" || metric != "complaint" || value <= limit {
		t.Fatalf("incident %s %s %s %s %.2f/%.2f", kind, scope, mailbox, metric, value, limit)
	}
	if held == 0 {
		t.Fatal("nothing held")
	}
	if n := f.count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'gmail' AND status IN ('queued','held')`, f.campaign); n != 0 {
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
	_, done, err := f.e.Drive(f.ctx, f.campaign, f.start.Add(20*time.Minute), 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Fatalf("send didn't finish after resuming: %s", f.status())
	}
}

func TestPostalOutageHoldsShare(t *testing.T) {
	f := newFixture(t)
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(contactSpec{n: 3000, mailbox: "gmail", domain: "gmail.com", engaged: true})
	f.startSend(engine.StartOptions{Scenario: "postal-outage"})
	// The scenario takes Postal down 14 minutes into the send; start the
	// clock 13 minutes in.
	f.exec(`UPDATE campaigns SET started_at = $2 WHERE id = $1`, f.campaign, f.start.Add(-13*time.Minute))
	_, _, err := f.e.Drive(f.ctx, f.campaign, f.start, 8*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var routeStatus string
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT cr.status FROM campaign_routes cr JOIN providers p ON p.id = cr.provider_id WHERE cr.campaign_id = $1 AND p.adapter = 'postal'`, f.campaign).Scan(&routeStatus); err != nil {
		t.Fatal(err)
	}
	if routeStatus != "held" {
		rows, _ := f.env.App.Pool.Query(f.ctx, `SELECT p.adapter, m.status, count(*), max(m.attempts), max(m.error), min(m.sent_at), max(m.sent_at) FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.campaign_id = $1 GROUP BY 1, 2`, f.campaign)
		for rows.Next() {
			var a, st, e string
			var n, att int64
			var s1, s2 *time.Time
			_ = rows.Scan(&a, &st, &n, &att, &e, &s1, &s2)
			t.Logf("%s %s %d attempts<=%d err=%q sent %v..%v", a, st, n, att, e, s1, s2)
		}
		rows.Close()
		t.Logf("start %v", f.start)
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
		t.Fatalf("status %s; with only the held share left the campaign waits for a decision", f.status())
	}
	var ses, postal uuid.UUID
	_ = f.env.App.Pool.QueryRow(f.ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = 'ses'`, f.ws).Scan(&ses)
	_ = f.env.App.Pool.QueryRow(f.ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = 'postal'`, f.ws).Scan(&postal)
	moved, err := f.e.MoveShare(f.ctx, f.campaign, f.sofia, postal, ses)
	if err != nil {
		t.Fatal(err)
	}
	if moved != d.Remaining {
		t.Fatalf("moved %d, the incident held %d", moved, d.Remaining)
	}
	_, done, err := f.e.Drive(f.ctx, f.campaign, f.start.Add(8*time.Minute), 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.status() != "sent" {
		t.Fatalf("send didn't finish after the move: %s", f.status())
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('campaign.route_hold', 'campaign.share_move')`, f.campaign.String()); n != 2 {
		t.Fatalf("%d audit entries for hold and move, want 2", n)
	}
}
