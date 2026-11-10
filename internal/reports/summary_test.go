package reports_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/store"
)

// TestReportSummary: a member who can view reports turns the sending
// summary on (Notifications); the hourly tick claims each period once and
// queues its email, which goes only to members who may still read reports,
// with the sections their role shows.
func TestReportSummary(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pool := env.App.Pool
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	userID := func(email string) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	var wsID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	linh, daniel, sofia, aisha := userID("linh@acme.io"), userID("daniel@acme.io"), userID("sofia@acme.io"), userID("aisha@acme.io")

	// The Monday 09:30, in Linh's zone, that ends the week the October
	// Product Update started in: whatever day the suite runs, that week
	// holds a campaign.
	var started time.Time
	if err := pool.QueryRow(ctx, `SELECT started_at FROM campaigns WHERE workspace_id = $1 AND name = 'October Product Update'`, wsID).Scan(&started); err != nil {
		t.Fatal(err)
	}
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip(err)
	}
	local := started.In(ny)
	monday := local.Day() + 7 - (int(local.Weekday())+6)%7
	from := time.Date(local.Year(), local.Month(), monday-7, 0, 0, 0, 0, ny)
	to := time.Date(local.Year(), local.Month(), monday, 0, 0, 0, 0, ny)
	now := time.Date(local.Year(), local.Month(), monday, 9, 30, 0, 0, ny)

	mail := &fakeMail{}
	svc := &reports.Service{Pool: pool, Q: env.App.Q, Audit: env.App.Audit, URLs: env.App.URLs, Jobs: env.App.Jobs,
		Mail:   func() mailer.Sender { return mail },
		Health: func(context.Context, *store.Workspace) (int, string, bool, error) { return 92, "Good", true, nil },
		Now:    func() time.Time { return now }}
	worker := &reports.SummarySendWorker{S: svc}
	// tick runs the hourly job and returns the emails it queued; send runs
	// them as River would.
	tick := func() []reports.SummarySendArgs {
		t.Helper()
		n, err := svc.QueueSummaries(ctx)
		if err != nil {
			t.Fatal(err)
		}
		rows, err := pool.Query(ctx, `DELETE FROM river_job WHERE kind = 'reports.summary_send' RETURNING args`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []reports.SummarySendArgs
		for rows.Next() {
			var raw []byte
			var a reports.SummarySendArgs
			if err := rows.Scan(&raw); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				t.Fatal(err)
			}
			out = append(out, a)
		}
		if len(out) != n {
			t.Fatalf("the tick says it queued %d emails, River holds %d", n, len(out))
		}
		return out
	}
	send := func(jobs []reports.SummarySendArgs) {
		t.Helper()
		for _, a := range jobs {
			if err := worker.Work(ctx, &river.Job[reports.SummarySendArgs]{Args: a}); err != nil {
				t.Fatal(err)
			}
		}
	}
	to1 := func(id uuid.UUID) []mailer.Message {
		var email string
		_ = pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, id).Scan(&email)
		var out []mailer.Message
		for _, m := range mail.all() {
			if m.To == email {
				out = append(out, m)
			}
		}
		return out
	}
	hasRow := func(id uuid.UUID) bool {
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM report_summaries WHERE user_id = $1`, id).Scan(&n)
		return n == 1
	}

	// The demo turned Linh's weekly summary on. Daniel (Content Writer)
	// asks too, with a role that reads reports but not the audience or
	// deliverability.
	exec(`UPDATE roles SET permissions = array_remove(array_remove(permissions, 'subscribers:view'), 'deliverability:view') WHERE workspace_id = $1 AND key = 'content-writer'`, wsID)
	exec(`INSERT INTO report_summaries (user_id, workspace_id, frequency) VALUES ($1, $2, 'weekly')`, daniel, wsID)
	jobs := tick()
	if len(jobs) != 2 {
		t.Fatalf("queued %d emails, want Linh's and Daniel's", len(jobs))
	}
	if a := jobs[0]; !a.From.Equal(from) || !a.To.Equal(to) || a.Frequency != "weekly" {
		t.Fatalf("period %v → %v, want %v → %v", a.From, a.To, from, to)
	}
	// Ticks that overlap claim the week once.
	if again := tick(); len(again) != 0 {
		t.Fatalf("the week was claimed twice: %d", len(again))
	}
	send(jobs)
	got := to1(linh)
	if len(got) != 1 {
		t.Fatalf("Linh got %d summaries", len(got))
	}
	msg := got[0]
	if msg.Kind != "report_summary" || !strings.HasPrefix(msg.Subject, "Acme Newsletter") || msg.WorkspaceID == nil || *msg.WorkspaceID != wsID || msg.Text == "" {
		t.Fatalf("summary: %+v", msg)
	}
	// Every email campaign started in the week is in the totals, and the
	// most opened link to their reports.
	rows, err := pool.Query(ctx, `SELECT c.id, c.name, coalesce((SELECT sum(sent) FROM campaign_counters WHERE campaign_id = c.id), 0)::bigint
		FROM campaigns c WHERE c.workspace_id = $1 AND c.deleted_at IS NULL AND c.channel = 'email' AND c.origin = '' AND c.started_at >= $2 AND c.started_at < $3`, wsID, from, to)
	if err != nil {
		t.Fatal(err)
	}
	var names, ids []string
	var sent int64
	for rows.Next() {
		var id uuid.UUID
		var name string
		var n int64
		if err := rows.Scan(&id, &name, &n); err != nil {
			t.Fatal(err)
		}
		ids, names, sent = append(ids, id.String()), append(names, name), sent+n
	}
	rows.Close()
	if len(names) == 0 {
		t.Fatal("no campaign started in the week")
	}
	region := applang.ForUserID(ctx, env.App.Q, linh).Region
	if !strings.Contains(msg.Text, region.Int(sent)) || !strings.Contains(msg.HTML, region.Int(sent)) {
		t.Errorf("the total sent %s is missing:\n%s", region.Int(sent), msg.Text)
	}
	if len(names) <= 5 {
		for i, name := range names {
			if link := "/w/acme-newsletter/campaigns/" + ids[i] + "/report"; !strings.Contains(msg.Text, name) || !strings.Contains(msg.Text, link) || !strings.Contains(msg.HTML, link) {
				t.Errorf("%s or its report link is missing:\n%s", name, msg.Text)
			}
		}
	}
	// Sections as the role shows them: Linh sees the audience and sender
	// health, Daniel's role shows neither.
	dan := to1(daniel)
	if len(dan) != 1 {
		t.Fatalf("Daniel got %d summaries", len(dan))
	}
	if l, d := strings.Count(msg.HTML, "<h2"), strings.Count(dan[0].HTML, "<h2"); l != d+2 {
		t.Errorf("sections: Linh %d, Daniel %d", l, d)
	}

	// A role that stops reading reports gets nothing, and keeps the setting
	// for when it does again.
	exec(`UPDATE roles SET permissions = array_remove(permissions, 'reports:view') WHERE workspace_id = $1 AND key = 'content-writer'`, wsID)
	exec(`UPDATE report_summaries SET last_period_start = NULL WHERE user_id = $1`, daniel)
	send(tick())
	if len(to1(daniel)) != 1 || !hasRow(daniel) {
		t.Errorf("a member without reports:view: %d summaries, setting kept %v", len(to1(daniel)), hasRow(daniel))
	}

	// A member removed after the claim gets nothing, and the setting goes.
	exec(`INSERT INTO report_summaries (user_id, workspace_id, frequency) VALUES ($1, $2, 'weekly')`, sofia, wsID)
	jobs = tick()
	if len(jobs) != 1 || jobs[0].UserID != sofia {
		t.Fatalf("queued for Sofia: %+v", jobs)
	}
	exec(`DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, wsID, sofia)
	send(jobs)
	if len(to1(sofia)) != 0 || hasRow(sofia) {
		t.Errorf("a former member: %d summaries, setting kept %v", len(to1(sofia)), hasRow(sofia))
	}

	// A deleted workspace sends nothing.
	exec(`UPDATE report_summaries SET last_period_start = NULL WHERE user_id = $1`, linh)
	exec(`UPDATE workspaces SET deleted_at = now() WHERE id = $1`, wsID)
	if jobs := tick(); len(jobs) != 0 {
		t.Errorf("a deleted workspace queued %d summaries", len(jobs))
	}
	exec(`UPDATE workspaces SET deleted_at = NULL WHERE id = $1`, wsID)

	// Turned off: nothing.
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	path := "/workspaces/acme-newsletter/report-summary"
	var s reports.ReportSummarySetting
	c.Do("PUT", path, map[string]any{"frequency": "off"}).MustStatus(t, 200).JSON(t, &s)
	if s.Frequency != "off" || s.NextAt != nil || hasRow(linh) {
		t.Fatalf("off: %+v", s)
	}
	if jobs := tick(); len(jobs) != 0 || len(to1(linh)) != 1 {
		t.Errorf("summaries after turning it off: %d queued, %d sent", len(jobs), len(to1(linh)))
	}

	audits := func(action string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = $1 AND target_id = $2`, action, linh.String()).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if audits("report_summary.remove") != 1 {
		t.Errorf("turning it off is audited once: %d", audits("report_summary.remove"))
	}
	// From here the clock is the real one: turning the summary on counts
	// the period Linh is in as behind her, so nothing goes out at once.
	now = time.Now()
	c.Do("PUT", path, map[string]any{"frequency": "weekly"}).MustStatus(t, 200).JSON(t, &s)
	if s.Frequency != "weekly" || s.NextAt == nil || !s.NextAt.After(now) || s.NextAt.After(now.Add(7*24*time.Hour)) ||
		s.NextAt.In(ny).Weekday() != time.Monday || s.NextAt.In(ny).Hour() != 9 {
		t.Fatalf("weekly: %+v", s)
	}
	if jobs := tick(); len(jobs) != 0 {
		t.Errorf("turning it on queued %d summaries at once", len(jobs))
	}
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &s)
	if s.Frequency != "weekly" || s.NextAt == nil || audits("report_summary.set") != 1 {
		t.Fatalf("read back: %+v, %d audit entries", s, audits("report_summary.set"))
	}
	// Saving the same choice changes nothing; monthly comes on the 1st.
	c.Do("PUT", path, map[string]any{"frequency": "weekly"}).MustStatus(t, 200)
	c.Do("PUT", path, map[string]any{"frequency": "monthly"}).MustStatus(t, 200).JSON(t, &s)
	if s.Frequency != "monthly" || s.NextAt == nil || s.NextAt.In(ny).Day() != 1 || s.NextAt.In(ny).Hour() != 9 || audits("report_summary.set") != 2 {
		t.Fatalf("monthly: %+v, %d audit entries", s, audits("report_summary.set"))
	}
	c.Do("PUT", path, map[string]any{"frequency": "daily"}).MustStatus(t, 422)

	// Only people get it: an API key that reads reports can't read or change
	// the summary of the person who made it.
	var key struct{ Secret string }
	c.Do("POST", "/workspaces/acme-newsletter/api-keys", map[string]any{"name": "Reports bot", "permissions": []string{"reports:view"}}).MustStatus(t, 201).JSON(t, &key)
	bot := env.Client()
	bot.Bearer = key.Secret
	if p := bot.Do("PUT", path, map[string]any{"frequency": "off"}).MustStatus(t, 403).Problem(t); p["code"] != "humans_only" {
		t.Fatalf("an API key changed the summary: %v", p)
	}
	bot.Do("GET", path, nil).MustStatus(t, 403)
	// A Viewer reads reports, so she can have her own; Linh's stays.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("PUT", path, map[string]any{"frequency": "weekly"}).MustStatus(t, 200).JSON(t, &s)
	if s.Frequency != "weekly" || !hasRow(aisha) {
		t.Fatalf("Aisha's summary: %+v", s)
	}
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &s)
	if s.Frequency != "monthly" {
		t.Fatalf("Aisha's choice changed Linh's: %+v", s)
	}
}

type fakeMail struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (f *fakeMail) SendSystem(_ context.Context, m mailer.Message) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return "Postmark", nil
}

func (f *fakeMail) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

func (f *fakeMail) all() []mailer.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]mailer.Message(nil), f.sent...)
}
