package campaigns_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

type listed struct {
	Items []struct {
		ID         string     `json:"id"`
		Name       string     `json:"name"`
		Status     string     `json:"status"`
		Tags       []string   `json:"tags"`
		ArchivedAt *time.Time `json:"archivedAt"`
		DeletedAt  *time.Time `json:"deletedAt"`
	} `json:"items"`
	Total  int64 `json:"total"`
	Counts struct{ All, Drafts, Sent, Archived, Deleted int64 }
}

func (l listed) find(name string) (string, bool) {
	for _, it := range l.Items {
		if it.Name == name {
			return it.ID, true
		}
	}
	return "", false
}

// TestDeleteArchiveRestore: drafts and campaigns that sent nothing are
// deleted (restorable for 30 days, their open approval closed), sent ones
// archived; each is audited and the list's tabs follow (review 2026-11 CA-1).
func TestDeleteArchiveRestore(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	flash := campaignID(t, env, "Flash sale: 48 hours only")
	c.Do(http.MethodDelete, ws+"/campaigns/"+flash, nil).MustStatus(t, http.StatusNoContent)
	if auditCount(t, env, "campaign.delete", flash) != 1 {
		t.Fatal("the delete isn't in the audit log")
	}
	var l listed
	c.Do(http.MethodGet, ws+"/campaigns?status=drafts&pageSize=50", nil).MustStatus(t, 200).JSON(t, &l)
	if _, ok := l.find("Flash sale: 48 hours only"); ok || l.Counts.Deleted != 1 || l.Counts.Drafts != 9 {
		t.Fatalf("a deleted draft leaves Drafts for Deleted: %+v", l.Counts)
	}
	c.Do(http.MethodGet, ws+"/campaigns?status=deleted", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) != 1 || l.Items[0].ID != flash || l.Items[0].DeletedAt == nil {
		t.Fatalf("deleted tab: %+v", l.Items)
	}
	// A deleted campaign is gone everywhere else, and can't be deleted twice.
	c.Do(http.MethodDelete, ws+"/campaigns/"+flash, nil).MustStatus(t, http.StatusNotFound)
	c.Do(http.MethodPost, ws+"/campaigns/"+flash+"/restore", nil).MustStatus(t, http.StatusNoContent)
	c.Do(http.MethodPost, ws+"/campaigns/"+flash+"/restore", nil).MustStatus(t, http.StatusConflict)
	c.Do(http.MethodGet, ws+"/campaigns?status=drafts&pageSize=50", nil).MustStatus(t, 200).JSON(t, &l)
	if _, ok := l.find("Flash sale: 48 hours only"); !ok || l.Counts.Deleted != 0 {
		t.Fatalf("restored draft: %+v", l.Counts)
	}

	// Waiting for approval: deleting closes the request.
	nov := campaignID(t, env, "November Product Update")
	c.Do(http.MethodDelete, ws+"/campaigns/"+nov, nil).MustStatus(t, http.StatusNoContent)
	var open int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM approval_requests WHERE campaign_id = $1 AND status IN ('pending', 'approved')`, nov).Scan(&open); err != nil || open != 0 {
		t.Fatalf("the approval request stays open: %d %v", open, err)
	}
	var status string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT status FROM campaigns WHERE id = $1`, nov).Scan(&status); err != nil || status != "draft" {
		t.Fatalf("a deleted campaign whose approval closed is a draft again: %q %v", status, err)
	}
	// Restorable for 30 days only.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaigns SET deleted_at = now() - interval '31 days' WHERE id = $1`, nov); err != nil {
		t.Fatal(err)
	}
	p := c.Do(http.MethodPost, ws+"/campaigns/"+nov+"/restore", nil).MustStatus(t, http.StatusGone).Problem(t)
	if p["code"] != "restore_expired" {
		t.Fatalf("expired restore: %+v", p)
	}
	// Maintenance removes it for good, with what hangs off it.
	before := time.Now().Add(-30 * 24 * time.Hour)
	if n, err := store.New(env.App.Pool).PurgeDeletedCampaigns(context.Background(), &before); err != nil || n != 1 {
		t.Fatalf("purge: %d %v", n, err)
	}
	c.Do(http.MethodPost, ws+"/campaigns/"+nov+"/restore", nil).MustStatus(t, http.StatusNotFound)

	// Sent: archived, not deleted; its report still opens.
	oct := campaignID(t, env, "October Product Update")
	p = c.Do(http.MethodDelete, ws+"/campaigns/"+oct, nil).MustStatus(t, http.StatusConflict).Problem(t)
	if p["code"] != "archive_instead" {
		t.Fatalf("deleting a sent campaign: %+v", p)
	}
	c.Do(http.MethodPost, ws+"/campaigns/"+oct+"/archive", nil).MustStatus(t, http.StatusNoContent)
	c.Do(http.MethodPost, ws+"/campaigns/"+oct+"/archive", nil).MustStatus(t, http.StatusNoContent) // already archived: nothing to do
	if auditCount(t, env, "campaign.archive", oct) != 1 {
		t.Fatal("archiving twice logs once")
	}
	c.Do(http.MethodGet, ws+"/campaigns?pageSize=100", nil).MustStatus(t, 200).JSON(t, &l)
	if _, ok := l.find("October Product Update"); ok || l.Counts.Archived != 1 {
		t.Fatalf("archived campaigns leave All: %+v", l.Counts)
	}
	c.Do(http.MethodGet, ws+"/campaigns?status=archived", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) != 1 || l.Items[0].ArchivedAt == nil {
		t.Fatalf("archived tab: %+v", l.Items)
	}
	c.Do(http.MethodGet, ws+"/campaigns/"+oct+"/report", nil).MustStatus(t, http.StatusOK)
	c.Do(http.MethodPost, ws+"/campaigns/"+oct+"/unarchive", nil).MustStatus(t, http.StatusNoContent)
	c.Do(http.MethodPost, ws+"/campaigns/"+oct+"/unarchive", nil).MustStatus(t, http.StatusConflict)
	// A draft can't be archived; a scheduled campaign can't be deleted.
	p = c.Do(http.MethodPost, ws+"/campaigns/"+flash+"/archive", nil).MustStatus(t, http.StatusConflict).Problem(t)
	if p["code"] != "delete_instead" {
		t.Fatalf("archiving a draft: %+v", p)
	}
	c.Do(http.MethodDelete, ws+"/campaigns/"+campaignID(t, env, "Weekly Digest #142"), nil).MustStatus(t, http.StatusConflict)

	// A viewer can't.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do(http.MethodDelete, ws+"/campaigns/"+flash, nil).MustStatus(t, http.StatusForbidden)
	v.Do(http.MethodPost, ws+"/campaigns/"+oct+"/archive", nil).MustStatus(t, http.StatusForbidden)
}

// TestBulkTagsAndFilters: bulk actions go one campaign at a time, skipping
// with a reason; tags and the list's filters narrow it (review 2026-11 CA-2).
func TestBulkTagsAndFilters(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	flash, oct := campaignID(t, env, "Flash sale: 48 hours only"), campaignID(t, env, "October Product Update")

	var res struct {
		Done    int
		Skipped []struct{ ID, Name, Reason string }
	}
	c.Do(http.MethodPost, ws+"/campaigns/bulk", map[string]any{"ids": []string{flash, oct}, "action": "tag", "tag": "  Black  Friday "}).MustStatus(t, 200).JSON(t, &res)
	if res.Done != 2 || len(res.Skipped) != 0 {
		t.Fatalf("tag: %+v", res)
	}
	var tags struct{ Tags []string }
	c.Do(http.MethodGet, ws+"/campaigns/tags", nil).MustStatus(t, 200).JSON(t, &tags)
	if !slices.Equal(tags.Tags, []string{"black friday"}) {
		t.Fatalf("tags: %v", tags.Tags)
	}
	var l listed
	c.Do(http.MethodGet, ws+"/campaigns?tag=black+friday", nil).MustStatus(t, 200).JSON(t, &l)
	if l.Total != 2 || len(l.Items) != 2 || !slices.Equal(l.Items[0].Tags, []string{"black friday"}) {
		t.Fatalf("tag filter: %+v", l)
	}
	c.Do(http.MethodPost, ws+"/campaigns/bulk", map[string]any{"ids": []string{flash}, "action": "tag", "tag": "!!"}).MustStatus(t, http.StatusUnprocessableEntity)

	// Archive both: the draft is skipped with why, the sent one archived.
	c.Do(http.MethodPost, ws+"/campaigns/bulk", map[string]any{"ids": []string{flash, oct, oct}, "action": "archive"}).MustStatus(t, 200).JSON(t, &res)
	if res.Done != 1 || len(res.Skipped) != 1 || res.Skipped[0].ID != flash || res.Skipped[0].Name != "Flash sale: 48 hours only" || res.Skipped[0].Reason == "" {
		t.Fatalf("archive: %+v", res)
	}
	c.Do(http.MethodPost, ws+"/campaigns/bulk", map[string]any{"ids": []string{flash, oct}, "action": "untag", "tag": "black friday"}).MustStatus(t, 200).JSON(t, &res)
	if res.Done != 2 {
		t.Fatalf("untag: %+v", res)
	}
	c.Do(http.MethodGet, ws+"/campaigns/tags", nil).MustStatus(t, 200).JSON(t, &tags)
	if len(tags.Tags) != 0 {
		t.Fatalf("tags after untag: %v", tags.Tags)
	}

	// Search reaches subjects; LIKE's wildcards are literal.
	var subject string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT subject FROM campaigns WHERE id = $1`, flash).Scan(&subject); err != nil || len(subject) < 6 {
		t.Fatalf("subject: %q %v", subject, err)
	}
	c.Do(http.MethodGet, ws+"/campaigns?q="+urlQuery(subject[:6]), nil).MustStatus(t, 200).JSON(t, &l)
	if _, ok := l.find("Flash sale: 48 hours only"); !ok {
		t.Fatalf("subject search %q: %+v", subject[:6], l.Items)
	}
	// "%" is a character, not a wildcard: only the subject that holds one
	// (Black Friday Early Access, "15% off") matches.
	c.Do(http.MethodGet, ws+"/campaigns?q=%25", nil).MustStatus(t, 200).JSON(t, &l)
	if _, ok := l.find("Black Friday Early Access"); l.Total != 1 || !ok {
		t.Fatalf("%% matched %d campaigns", l.Total)
	}
	// Dates: the window holds what went or goes out in it.
	var all listed
	c.Do(http.MethodGet, ws+"/campaigns?pageSize=100", nil).MustStatus(t, 200).JSON(t, &all)
	c.Do(http.MethodGet, ws+"/campaigns?from=2000-01-01&to=2000-01-02", nil).MustStatus(t, 200).JSON(t, &l)
	if l.Total != 0 {
		t.Fatalf("a day in 2000: %d", l.Total)
	}
	c.Do(http.MethodGet, ws+"/campaigns?from=2000-01-01", nil).MustStatus(t, 200).JSON(t, &l)
	if l.Total != all.Total {
		t.Fatalf("since 2000: %d of %d", l.Total, all.Total)
	}
	// A list of the audience.
	var listID uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT (audience->'lists'->>0)::uuid FROM campaigns WHERE id = $1`, oct).Scan(&listID); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodGet, ws+"/campaigns?list="+listID.String()+"&status=archived", nil).MustStatus(t, 200).JSON(t, &l)
	if l.Total != 1 {
		t.Fatalf("archived on the list: %d", l.Total)
	}
	c.Do(http.MethodGet, ws+"/campaigns?sender="+uuid.NewString(), nil).MustStatus(t, 200).JSON(t, &l)
	if l.Total != 0 {
		t.Fatalf("an unknown sender: %d", l.Total)
	}
}

func urlQuery(s string) string {
	out := ""
	for _, r := range s {
		if r == ' ' {
			out += "+"
			continue
		}
		out += string(r)
	}
	return out
}

// TestCalendar: scheduled and sent campaigns and digest issues between two
// days; a range over 62 days is refused (review 2026-11 CA-3).
func TestCalendar(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("aisha@acme.io") // viewers see it too
	var scheduledAt time.Time
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT scheduled_at FROM campaigns WHERE name = 'Weekly Digest #142'`).Scan(&scheduledAt); err != nil {
		t.Fatal(err)
	}
	from := scheduledAt.AddDate(0, 0, -20).Format(time.DateOnly)
	to := scheduledAt.AddDate(0, 0, 20).Format(time.DateOnly)
	var cal struct {
		Items []struct {
			Kind, ID, Name, Status string
			At                     time.Time
		}
	}
	c.Do(http.MethodGet, ws+"/campaigns/calendar?from="+from+"&to="+to, nil).MustStatus(t, 200).JSON(t, &cal)
	kinds := map[string]int{}
	found := false
	for i, it := range cal.Items {
		kinds[it.Kind]++
		if it.Name == "Weekly Digest #142" && it.Kind == "scheduled" && it.At.Equal(scheduledAt) {
			found = true
		}
		if i > 0 && it.At.Before(cal.Items[i-1].At) {
			t.Fatalf("not in time order: %+v", cal.Items)
		}
	}
	if !found || kinds["sent"] == 0 {
		t.Fatalf("calendar: %v %+v", kinds, cal.Items)
	}
	c.Do(http.MethodGet, ws+"/campaigns/calendar?from=2026-01-01&to=2026-04-01", nil).MustStatus(t, http.StatusUnprocessableEntity)
	c.Do(http.MethodGet, ws+"/campaigns/calendar?from=2026-01-02&to=2026-01-01", nil).MustStatus(t, http.StatusUnprocessableEntity)
}

// TestFollowUp: the follow-up's audience is the report's bucket: people
// who got it and never opened (opens by privacy proxies alone count as
// opens, so they aren't sent again), or who clicked it (review 2026-11 CA-4).
func TestFollowUp(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	oct := campaignID(t, env, "October Product Update")
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	type counts struct{ Received, NonOpeners, MachineOnly, Clickers int64 }
	var before, after counts
	c.Do(http.MethodGet, ws+"/campaigns/"+oct+"/follow-up", nil).MustStatus(t, 200).JSON(t, &before)

	// Five readers: a person opened, a privacy proxy opened, nobody
	// opened, a person clicked, a scanner clicked (and opened).
	events := map[string][][3]any{
		"human":   {{"open", false, "human"}},
		"machine": {{"open", true, "machine"}},
		"none":    {},
		"clicker": {{"open", false, "human"}, {"click", false, "human"}},
		"scanner": {{"open", true, "bot"}, {"click", true, "bot"}},
	}
	for name, evs := range events {
		id := uuid.Must(uuid.NewV7())
		email := "reader@" + name + ".followup.example" // a domain each: the segment preview counts them one by one
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name) VALUES ($1, $2, $3, 'F')`, id, wsID, email); err != nil {
			t.Fatal(err)
		}
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, campaign_id, contact_id, email, status, queued_at, sent_at)
			VALUES ($1, $2, 'broadcast', $3, $4, $5, 'delivered', now(), now())`, uuid.Must(uuid.NewV7()), wsID, oct, id, email); err != nil {
			t.Fatal(err)
		}
		for _, e := range evs {
			if _, err := env.App.Pool.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, contact_id, campaign_id, type, occurred_at, machine, class)
				VALUES ($1, $2, $3, $4, $5, now(), $6, $7)`, uuid.Must(uuid.NewV7()), wsID, id, oct, e[0], e[1], e[2]); err != nil {
				t.Fatal(err)
			}
		}
	}
	c.Do(http.MethodGet, ws+"/campaigns/"+oct+"/follow-up", nil).MustStatus(t, 200).JSON(t, &after)
	if d := (counts{after.Received - before.Received, after.NonOpeners - before.NonOpeners, after.MachineOnly - before.MachineOnly, after.Clickers - before.Clickers}); d != (counts{5, 1, 2, 1}) {
		t.Fatalf("buckets moved by %+v, want 5 received, 1 non-opener, 2 machine-only (proxy and scanner), 1 clicker", d)
	}

	members := func(segment string) []string {
		t.Helper()
		var seg struct{ Definition map[string]any }
		c.Do(http.MethodGet, ws+"/segments/"+segment, nil).MustStatus(t, 200).JSON(t, &seg)
		out := []string{}
		for name := range events {
			def := map[string]any{"groups": []any{map[string]any{"match": "all", "conditions": append(
				slices.Clone(seg.Definition["groups"].([]any)[0].(map[string]any)["conditions"].([]any)),
				map[string]any{"field": "email_domain", "op": "is", "value": name + ".followup.example"})}}}
			var p struct{ Count int64 }
			c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "builder", "definition": def}).MustStatus(t, 200).JSON(t, &p)
			if p.Count > 0 {
				out = append(out, name)
			}
		}
		slices.Sort(out)
		return out
	}
	follow := func(kind string) (string, []string) {
		t.Helper()
		var d struct {
			ID       string
			Name     string
			Subject  string
			Audience struct{ Segments []string }
		}
		c.Do(http.MethodPost, ws+"/campaigns/"+oct+"/follow-up", map[string]any{"kind": kind, "subject": "In case you missed it", "delayHours": 0}).
			MustStatus(t, http.StatusCreated).JSON(t, &d)
		if d.Subject != "In case you missed it" || len(d.Audience.Segments) != 1 {
			t.Fatalf("follow-up draft: %+v", d)
		}
		if auditCount(t, env, "campaign.follow_up", d.ID) != 1 {
			t.Fatal("the follow-up isn't in the audit log")
		}
		return d.ID, members(d.Audience.Segments[0])
	}
	if _, got := follow("non_openers"); !slices.Equal(got, []string{"none"}) {
		t.Fatalf("non-openers: %v", got)
	}
	if _, got := follow("clickers"); !slices.Equal(got, []string{"clicker"}) {
		t.Fatalf("clickers: %v", got)
	}
	// A draft has no report to follow up.
	c.Do(http.MethodPost, ws+"/campaigns/"+campaignID(t, env, "Flash sale: 48 hours only")+"/follow-up", map[string]any{"kind": "clickers", "subject": "x"}).
		MustStatus(t, http.StatusConflict)
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do(http.MethodPost, ws+"/campaigns/"+oct+"/follow-up", map[string]any{"kind": "clickers", "subject": "x"}).MustStatus(t, http.StatusForbidden)
}

// TestOverviewRecentIsOneLightPage: Overview's "Recent" lists what went
// out, latest start first, as the campaigns list has them (CA-7).
func TestOverviewRecentIsOneLightPage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var ov struct {
		Recent struct {
			Items []struct {
				Name      string
				Status    string
				StartedAt *time.Time
			}
		}
	}
	c.Do(http.MethodGet, ws+"/overview/recent-campaigns", nil).MustStatus(t, 200).JSON(t, &ov.Recent)
	items := ov.Recent.Items
	if len(items) == 0 || len(items) > 4 {
		t.Fatalf("recent: %+v", items)
	}
	for i, it := range items {
		if !slices.Contains([]string{"sending", "paused", "sent", "cancelled", "failed"}, it.Status) {
			t.Fatalf("recent holds %s: %+v", it.Status, it)
		}
		if i > 0 && it.StartedAt != nil && items[i-1].StartedAt != nil && it.StartedAt.After(*items[i-1].StartedAt) {
			t.Fatalf("not latest first: %+v", items)
		}
	}
}

// TestListTenThousandCampaigns: with 10,000 sent campaigns, a page of the
// list, a filtered page and Overview's Recent each answer quickly: the
// page's ids are chosen first and only they are joined (review 2026-11 CA-7).
func TestListTenThousandCampaigns(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, reply_to, audience, editor_mode, mjml, html, plain_text,
		ab_test, schedule, thresholds, tracking, created_by, updated_by, content_hash, started_at, completed_at, tags)
		SELECT gen_random_uuid(), c.workspace_id, 'Bulk ' || g, 'sent', 'Issue ' || g, c.preheader, c.sender_id, c.reply_to, c.audience, c.editor_mode, c.mjml, c.html, c.plain_text,
		       NULL, '{"mode":"now"}', c.thresholds, c.tracking, c.created_by, c.updated_by, c.content_hash, now() - (g || ' hours')::interval, now() - (g || ' hours')::interval,
		       CASE WHEN g % 100 = 0 THEN '{rare}'::text[] ELSE '{}'::text[] END
		FROM campaigns c, generate_series(1, 10000) g WHERE c.name = 'October Product Update'`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `ANALYZE campaigns`); err != nil {
		t.Fatal(err)
	}
	timed := func(path string, v any) time.Duration {
		t.Helper()
		start := time.Now()
		c.Do(http.MethodGet, ws+path, nil).MustStatus(t, 200).JSON(t, v)
		return time.Since(start)
	}
	var l listed
	if d := timed("/campaigns?status=sent&page=200", &l); d > 2*time.Second || l.Total < 10000 || len(l.Items) != 25 {
		t.Fatalf("page 200 of sent: %s, total %d, %d items", d, l.Total, len(l.Items))
	}
	if d := timed("/campaigns?tag=rare&q=bulk", &l); d > 2*time.Second || l.Total != 100 {
		t.Fatalf("tag and search: %s, total %d", d, l.Total)
	}
	var recent struct{ Items []struct{ Name string } }
	if d := timed("/overview/recent-campaigns", &recent); d > time.Second || len(recent.Items) != 4 || recent.Items[0].Name != "Bulk 1" {
		t.Fatalf("recent: %s %+v", d, recent.Items)
	}
}
