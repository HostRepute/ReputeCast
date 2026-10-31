package audience_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

type sentMail struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (f *sentMail) SendSystem(_ context.Context, m mailer.Message) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return "Postmark", nil
}

func (f *sentMail) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

type jobView struct {
	ID          string
	Kind        string
	Status      string
	Total, Done int64
	Result      struct{ Rows, Sent, Confirmed, Suppressed, Cooldown, Failed int64 }
	DownloadURL string
	Error       string
}

// workspaceID reads the paging workspace's id.
func workspaceID(t *testing.T, env *apptest.Env) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT workspace_id FROM contacts WHERE email = 'reader00@paging.example'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// Review 2026-11 AU-4, AU-5: an export of everything matching runs in the
// background, in batches, and only its starter downloads it.
func TestExportRunsInTheBackground(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	ws := workspaceID(t, env)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name)
		SELECT gen_random_uuid(), $1, 'bulk' || g || '@paging.example', 'Bulk' FROM generate_series(1, 4500) g`, ws); err != nil {
		t.Fatal(err)
	}
	var job jobView
	c.Do(http.MethodPost, base+"/contacts/jobs/export", map[string]any{"allMatching": true, "filter": map[string]any{"q": "bulk"}}).MustStatus(t, 202).JSON(t, &job)
	if job.Kind != "export" || job.Status != "queued" || job.Total != 4500 {
		t.Fatalf("started: %+v", job)
	}
	if err := env.App.Audience().RunJob(ctx, uuid.MustParse(job.ID)); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodGet, base+"/contacts/jobs/"+job.ID, nil).MustStatus(t, 200).JSON(t, &job)
	if job.Status != "done" || job.Done != 4500 || job.Result.Rows != 4500 || job.DownloadURL == "" {
		t.Fatalf("finished: %+v", job)
	}
	r := c.Do(http.MethodGet, job.DownloadURL, nil).MustStatus(t, 200)
	recs, err := csv.NewReader(bytes.NewReader(r.Body)).ReadAll()
	if err != nil || len(recs) != 4501 || recs[0][0] != "email" {
		t.Fatalf("file: %d rows (%v)", len(recs), err)
	}
	// Another member who can export can't take someone else's file.
	other := env.CreateUser(apptest.UserOpts{Email: "admin@paging.example", Name: "Ada Admin"})
	env.AddMember(mustWorkspace(t, env, ws), other.ID, "admin")
	if _, err := env.App.Pool.Exec(ctx, `UPDATE users SET twofa_grace_until = now() + interval '1 day' WHERE id = $1`, other.ID); err != nil {
		t.Fatal(err)
	}
	oc := env.Client()
	oc.SignIn("admin@paging.example", "")
	oc.Do(http.MethodGet, job.DownloadURL, nil).MustStatus(t, 404)
	// The tray offers it to its starter only.
	var tray struct {
		Items []struct{ Kind, Status, DownloadURL, Title string }
	}
	c.Do(http.MethodGet, base+"/jobs", nil).MustStatus(t, 200).JSON(t, &tray)
	found := false
	for _, it := range tray.Items {
		if it.Kind == "export" && it.Status == "done" && it.DownloadURL == job.DownloadURL {
			found = true
		}
	}
	if !found {
		t.Fatalf("tray: %+v", tray.Items)
	}
	var started, finished, downloaded int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE action = 'contacts.export_started'), count(*) FILTER (WHERE action = 'contacts.export'),
		count(*) FILTER (WHERE action = 'contacts.export_downloaded') FROM audit_events WHERE workspace_id = $1`, ws).Scan(&started, &finished, &downloaded)
	if started != 1 || finished != 1 || downloaded != 1 {
		t.Fatalf("audit: %d %d %d", started, finished, downloaded)
	}
	// Seven days later the file is gone.
	audience := env.App.Audience()
	audience.Now = func() time.Time { return time.Now().AddDate(0, 0, 8) }
	if err := audience.SweepJobFiles(ctx); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodGet, job.DownloadURL, nil).MustStatus(t, 404)
}

func mustWorkspace(t *testing.T, env *apptest.Env, id uuid.UUID) store.Workspace {
	t.Helper()
	w, err := env.App.Q.GetWorkspace(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// Review 2026-11 AU-4: a bulk resend runs in the background, skips the
// confirmed and the suppressed, holds the cooldown, and says so.
func TestBulkResendReportsWhatItDid(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	mail := &sentMail{}
	env.App.Mail = mail
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET consent = CASE WHEN email < 'reader30' THEN 'pending' ELSE 'double' END WHERE email LIKE 'reader%'`); err != nil {
		t.Fatal(err)
	}
	ws := workspaceID(t, env)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO suppressions (id, workspace_id, email, reason, source) VALUES (gen_random_uuid(), $1, 'reader01@paging.example', 'complaint', 'webhook')`, ws); err != nil {
		t.Fatal(err)
	}
	// One got an email a minute ago.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO optin_sends (contact_id, sent_at) SELECT id, now() - interval '1 minute' FROM contacts WHERE email = 'reader02@paging.example'`); err != nil {
		t.Fatal(err)
	}
	// Without a System provider nothing starts.
	c.Do(http.MethodPost, base+"/contacts/bulk", map[string]any{"action": "resend_optin", "allMatching": true, "filter": map[string]any{}}).MustStatus(t, 409)
	// With one (a stand-in), the job covers everyone matching.
	svc := env.App.Audience()
	w := mustWorkspace(t, env, ws)
	started, err := svc.StartJob(ctx, &w, audience.JobResendOptIn, audience.Selection{AllMatching: true})
	if err != nil || started.Total != 61 {
		t.Fatalf("started: %+v %v", started, err)
	}
	if err := svc.RunJob(ctx, uuid.MustParse(started.ID)); err != nil {
		t.Fatal(err)
	}
	var job jobView
	c.Do(http.MethodGet, base+"/contacts/jobs/"+started.ID, nil).MustStatus(t, 200).JSON(t, &job)
	r := job.Result
	if job.Status != "done" || r.Sent != 28 || r.Suppressed != 1 || r.Cooldown != 1 || r.Confirmed != 31 || len(mail.sent) != 28 {
		t.Fatalf("resend: %+v, %d emails", job, len(mail.sent))
	}
}

// Review 2026-11 AU-6: the team takes contacts off a list or a tag, in
// bulk or one at a time; it is recorded as theirs, not an unsubscribe.
func TestRemoveFromListAndTag(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	var list struct{ ID string }
	c.Do(http.MethodPost, base+"/lists", map[string]any{"name": "Beta"}).MustStatus(t, 201).JSON(t, &list)
	c.Do(http.MethodPost, base+"/contacts/bulk", map[string]any{"action": "add_to_list", "listId": list.ID, "allMatching": true, "filter": map[string]any{"q": "reader1"}}).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/contacts/bulk", map[string]any{"action": "add_tag", "tag": "beta", "allMatching": true, "filter": map[string]any{"q": "reader1"}}).MustStatus(t, 200)
	var out struct{ Done int64 }
	c.Do(http.MethodPost, base+"/contacts/bulk", map[string]any{"action": "remove_tag", "tag": "BETA", "allMatching": true, "filter": map[string]any{"q": "reader1"}}).MustStatus(t, 200).JSON(t, &out)
	if out.Done != 10 {
		t.Fatalf("tags removed: %d", out.Done)
	}
	c.Do(http.MethodPost, base+"/contacts/bulk", map[string]any{"action": "remove_from_list", "listId": list.ID, "allMatching": true, "filter": map[string]any{"q": "reader1"}}).
		MustStatus(t, 200).JSON(t, &out)
	if out.Done != 10 {
		t.Fatalf("removed: %d", out.Done)
	}
	var memberships, removed, unsubscribed int
	_ = env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM list_memberships WHERE list_id = $1),
		(SELECT count(*) FROM consent_events WHERE list_id = $1 AND action = 'removed' AND source = 'team'),
		(SELECT count(*) FROM consent_events WHERE list_id = $1 AND action = 'unsubscribed')`, list.ID).Scan(&memberships, &removed, &unsubscribed)
	if memberships != 0 || removed != 10 || unsubscribed != 0 {
		t.Fatalf("memberships %d, removed %d, unsubscribed %d", memberships, removed, unsubscribed)
	}
	// One at a time, from the profile (and the API).
	c.Do(http.MethodPost, base+"/contacts/bulk", map[string]any{"action": "add_to_list", "listId": list.ID, "allMatching": true, "filter": map[string]any{"q": "reader20"}}).MustStatus(t, 200)
	var page pageOf
	c.Do(http.MethodGet, base+"/contacts?q=reader20", nil).MustStatus(t, 200).JSON(t, &page)
	c.Do(http.MethodDelete, base+"/contacts/"+page.Items[0].ID+"/lists/"+list.ID, nil).MustStatus(t, 204)
	var audited int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action IN ('contacts.remove_from_list', 'contacts.remove_tag', 'contact.remove_from_list')`).Scan(&audited)
	if audited != 3 {
		t.Fatalf("audited %d", audited)
	}
}

// Review 2026-11 AU-7: an address is corrected in place; another
// contact's or a suppressed address is refused.
func TestCorrectAnAddress(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	ws := workspaceID(t, env)
	var page pageOf
	c.Do(http.MethodGet, base+"/contacts?q=reader05@paging.example", nil).MustStatus(t, 200).JSON(t, &page)
	id := page.Items[0].ID
	c.Do(http.MethodPatch, base+"/contacts/"+id, map[string]any{"email": "reader06@paging.example"}).MustStatus(t, 409)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO suppressions (id, workspace_id, email, reason, source) VALUES (gen_random_uuid(), $1, 'gone@outlook.com', 'hard_bounce', 'webhook')`, ws); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodPatch, base+"/contacts/"+id, map[string]any{"email": "gone@outlook.com"}).MustStatus(t, 409)
	c.Do(http.MethodPatch, base+"/contacts/"+id, map[string]any{"email": "not an address"}).MustStatus(t, 422)
	var p struct{ Email, MailboxProvider string }
	c.Do(http.MethodPatch, base+"/contacts/"+id, map[string]any{"email": "  Reader05.New@Gmail.com "}).MustStatus(t, 200).JSON(t, &p)
	if p.Email != "reader05.new@gmail.com" || p.MailboxProvider != "gmail" {
		t.Fatalf("changed: %+v", p)
	}
	var events, audits int
	_ = env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM consent_events WHERE contact_id = $1 AND action = 'email_changed' AND note LIKE '%reader05@paging.example%'),
		(SELECT count(*) FROM audit_events WHERE action = 'contact.update' AND target_id = $1::text AND before->>'email' = 'reader05@paging.example')`, uuid.MustParse(id)).Scan(&events, &audits)
	if events != 1 || audits != 1 {
		t.Fatalf("history %d, audit %d", events, audits)
	}
}

// Review 2026-11 AU-10: joined between two days of the workspace's time
// zone; the contacts of one import; an import's tag on every row.
func TestJoinedBetweenAndImportFilters(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	ws := workspaceID(t, env)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE workspaces SET timezone = 'Pacific/Auckland' WHERE id = $1`, ws); err != nil {
		t.Fatal(err)
	}
	// 30 Sep 23:30 in UTC is 1 Oct 12:30 in Auckland.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, created_at) VALUES (gen_random_uuid(), $1, 'kiwi@paging.example', '2026-09-30T23:30:00Z')`, ws); err != nil {
		t.Fatal(err)
	}
	var page pageOf
	c.Do(http.MethodGet, base+"/contacts?q=kiwi&joinedAfter=2026-10-01&joinedBefore=2026-10-01", nil).MustStatus(t, 200).JSON(t, &page)
	if page.Total != 1 {
		t.Fatalf("1 Oct in Auckland: %d", page.Total)
	}
	c.Do(http.MethodGet, base+"/contacts?q=kiwi&joinedBefore=2026-09-30", nil).MustStatus(t, 200).JSON(t, &page)
	if page.Total != 0 {
		t.Fatalf("30 Sep in Auckland: %d", page.Total)
	}
	c.Do(http.MethodGet, base+"/contacts?joinedBefore=yesterday", nil).MustStatus(t, 422)
	// The contacts of one import.
	imp := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO imports (id, workspace_id, source, status) VALUES ($1, $2, 'csv', 'done')`, imp, ws); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET import_id = $1 WHERE email LIKE 'reader1%'`, imp); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodGet, base+"/contacts?import="+imp.String(), nil).MustStatus(t, 200).JSON(t, &page)
	if page.Total != 10 {
		t.Fatalf("from the import: %d", page.Total)
	}
	c.Do(http.MethodGet, fmt.Sprintf("%s/contacts?import=%s", base, "nope"), nil).MustStatus(t, 422)
}
