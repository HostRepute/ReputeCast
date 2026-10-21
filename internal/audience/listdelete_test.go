package audience_test

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/maintenance"
	"github.com/hostrepute/reputecast/internal/store"
)

// TestDeleteAndRestoreList walks the Dialogs board's Delete list: the
// dialog's numbers, what pauses, sends skipping the list, and the restore
// that turns it all back on.
func TestDeleteAndRestoreList(t *testing.T) {
	env, c := demoEnv(t)
	ctx := context.Background()
	pool := env.App.Pool
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")

	var wsID, product, digest uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	listID := func(name string) uuid.UUID {
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT id FROM lists WHERE workspace_id = $1 AND name = $2 AND deleted_at IS NULL`, wsID, name).Scan(&id); err != nil {
			t.Fatalf("list %s: %v", name, err)
		}
		return id
	}
	product, digest = listID("Product Updates"), listID("Weekly Digest")

	// The Welcome series starts when someone joins Product Updates, and
	// Weekly Digest #142 also goes to Product Updates, two days from now.
	sendAt := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE automations SET trigger = jsonb_set(trigger, '{listId}', to_jsonb($1::text)) WHERE workspace_id = $2 AND name = 'Welcome series'`, product.String(), wsID)
	exec(`UPDATE campaigns SET audience = jsonb_build_object('lists', jsonb_build_array($1::text, $2::text)), status = 'scheduled', scheduled_at = $3
	      WHERE workspace_id = $4 AND name = 'Weekly Digest #142'`, digest.String(), product.String(), sendAt, wsID)

	status := func(table, name string) string {
		t.Helper()
		var s string
		if err := pool.QueryRow(ctx, `SELECT status FROM `+table+` WHERE workspace_id = $1 AND name = $2`, wsID, name).Scan(&s); err != nil {
			t.Fatalf("%s %s: %v", table, name, err)
		}
		return s
	}
	audience := func(lists, exclude []uuid.UUID) int64 {
		t.Helper()
		n, err := env.App.Q.ComposerCountListAudience(ctx, store.ComposerCountListAudienceParams{WorkspaceID: wsID, Lists: lists, ExcludeLists: exclude})
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	productOnly, digestWithout := audience([]uuid.UUID{product}, nil), audience([]uuid.UUID{digest}, []uuid.UUID{product})
	if productOnly == 0 || digestWithout == 0 {
		t.Fatalf("audience before: %d on Product Updates, %d on Weekly Digest without it", productOnly, digestWithout)
	}

	path := ws + "/lists/" + product.String()
	t.Run("the dialog states the blast radius", func(t *testing.T) {
		var im struct {
			Name                          string
			Members, Elsewhere            int64
			OtherLists                    []string
			OtherListCount                int
			Forms, Automations, Campaigns []struct{ Name string }
		}
		c.Do(http.MethodGet, path+"/impact", nil).MustStatus(t, 200).JSON(t, &im)
		if im.Name != "Product Updates" || im.Members != 14110 || im.Elsewhere <= 0 || im.Elsewhere >= im.Members {
			t.Fatalf("impact %+v", im)
		}
		// Acme Summit 2026, the SMS demo's private list, is one of them (X21, brief §8.12).
		if im.OtherListCount != 3 || !slices.Contains(im.OtherLists, "Weekly Digest") {
			t.Errorf("other lists %v (%d)", im.OtherLists, im.OtherListCount)
		}
		one := func(what string, items []struct{ Name string }, want string) {
			if len(items) != 1 || items[0].Name != want {
				t.Errorf("%s: %+v, want %q", what, items, want)
			}
		}
		one("forms", im.Forms, "Newsletter signup")
		one("automations", im.Automations, "Welcome series")
		one("campaigns", im.Campaigns, "Weekly Digest #142")
		viewer.Do(http.MethodGet, path+"/impact", nil).MustStatus(t, 403)
	})

	t.Run("a member limited to other lists can't touch this one", func(t *testing.T) {
		exec(`UPDATE workspace_members SET scope = jsonb_build_object('lists', jsonb_build_array($1::text))
		      WHERE workspace_id = $2 AND user_id = (SELECT id FROM users WHERE email = 'sofia@acme.io')`, listID("Beta Program").String(), wsID)
		sofia := env.Client()
		sofia.SignInDemo("sofia@acme.io")
		sofia.Do(http.MethodGet, path+"/impact", nil).MustStatus(t, 403)
		sofia.Do(http.MethodPatch, path, map[string]any{"name": "Renamed"}).MustStatus(t, 403)
		sofia.Do(http.MethodDelete, path, nil).MustStatus(t, 403)
		// Her own list's dialog counts the lists outside her scope without naming them.
		var im struct {
			OtherLists     []string
			OtherListCount int
		}
		sofia.Do(http.MethodGet, ws+"/lists/"+listID("Beta Program").String()+"/impact", nil).MustStatus(t, 200).JSON(t, &im)
		// With Acme Summit 2026 (X21, brief §8.12).
		if len(im.OtherLists) != 0 || im.OtherListCount != 3 {
			t.Errorf("other lists for a scoped member: %v (%d)", im.OtherLists, im.OtherListCount)
		}
	})

	t.Run("deleting pauses what uses the list and sends skip it", func(t *testing.T) {
		viewer.Do(http.MethodDelete, path, nil).MustStatus(t, 403)
		c.Do(http.MethodDelete, path, nil).MustStatus(t, 204)

		var lists struct{ Items []struct{ Name string } }
		c.Do(http.MethodGet, ws+"/lists", nil).MustStatus(t, 200).JSON(t, &lists)
		for _, l := range lists.Items {
			if l.Name == "Product Updates" {
				t.Fatal("the deleted list is still listed")
			}
		}
		if s := status("forms", "Newsletter signup"); s != "paused" {
			t.Errorf("form is %s", s)
		}
		if s := status("automations", "Welcome series"); s != "paused" {
			t.Errorf("automation is %s", s)
		}
		var scheduled *time.Time
		if err := pool.QueryRow(ctx, `SELECT scheduled_at FROM campaigns WHERE workspace_id = $1 AND name = 'Weekly Digest #142'`, wsID).Scan(&scheduled); err != nil {
			t.Fatal(err)
		}
		if s := status("campaigns", "Weekly Digest #142"); s == "scheduled" || scheduled != nil {
			t.Errorf("campaign is %s at %v", s, scheduled)
		}
		for _, a := range []string{"list.delete", "form.pause", "automation.pause", "campaign.unscheduled"} {
			if audits(t, env, a) == 0 {
				t.Errorf("no %s audit entry", a)
			}
		}
		// The list adds nobody; as an exclusion it still excludes.
		if n := audience([]uuid.UUID{product}, nil); n != 0 {
			t.Errorf("the deleted list still adds %d recipients", n)
		}
		if n := audience([]uuid.UUID{digest}, []uuid.UUID{product}); n != digestWithout {
			t.Errorf("excluding the deleted list: %d, want %d", n, digestWithout)
		}
		c.Do(http.MethodGet, path+"/impact", nil).MustStatus(t, 404)
		c.Do(http.MethodDelete, path, nil).MustStatus(t, 404)
	})

	t.Run("its audit entry offers the restore", func(t *testing.T) {
		var entry uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT id FROM audit_events WHERE action = 'list.delete' AND target_id = $1`, product.String()).Scan(&entry); err != nil {
			t.Fatal(err)
		}
		var d struct {
			Restore *struct{ Kind, ID, Name string }
		}
		c.Do(http.MethodGet, ws+"/audit/events/"+entry.String(), nil).MustStatus(t, 200).JSON(t, &d)
		if d.Restore == nil || d.Restore.Kind != "list" || d.Restore.ID != product.String() || d.Restore.Name != "Product Updates" {
			t.Fatalf("restore %+v", d.Restore)
		}
	})

	t.Run("a list that took the name blocks the restore", func(t *testing.T) {
		var other struct{ ID string }
		c.Do(http.MethodPost, ws+"/lists", map[string]any{"name": "Product Updates"}).MustStatus(t, 201).JSON(t, &other)
		c.Do(http.MethodPost, path+"/restore", nil).MustStatus(t, 409)
		c.Do(http.MethodDelete, ws+"/lists/"+other.ID, nil).MustStatus(t, 204)
	})

	t.Run("restoring turns back on what the delete paused", func(t *testing.T) {
		viewer.Do(http.MethodPost, path+"/restore", nil).MustStatus(t, 403)
		var r struct {
			Name                                string
			Forms, Automations, Campaigns, Left []string
		}
		c.Do(http.MethodPost, path+"/restore", nil).MustStatus(t, 200).JSON(t, &r)
		if r.Name != "Product Updates" || !slices.Equal(r.Forms, []string{"Newsletter signup"}) || !slices.Equal(r.Automations, []string{"Welcome series"}) ||
			!slices.Equal(r.Campaigns, []string{"Weekly Digest #142"}) || len(r.Left) != 0 {
			t.Fatalf("restore %+v", r)
		}
		if s := status("forms", "Newsletter signup"); s != "live" {
			t.Errorf("form is %s", s)
		}
		if s := status("automations", "Welcome series"); s != "live" {
			t.Errorf("automation is %s", s)
		}
		var scheduled *time.Time
		if err := pool.QueryRow(ctx, `SELECT scheduled_at FROM campaigns WHERE workspace_id = $1 AND name = 'Weekly Digest #142'`, wsID).Scan(&scheduled); err != nil {
			t.Fatal(err)
		}
		if s := status("campaigns", "Weekly Digest #142"); s != "scheduled" || scheduled == nil || !scheduled.Equal(sendAt) {
			t.Errorf("campaign is %s at %v, want scheduled at %v", s, scheduled, sendAt)
		}
		if n := audience([]uuid.UUID{product}, nil); n != productOnly {
			t.Errorf("restored list adds %d, want %d", n, productOnly)
		}
		if audits(t, env, "list.restore") != 1 {
			t.Error("no list.restore audit entry")
		}
		c.Do(http.MethodPost, path+"/restore", nil).MustStatus(t, 409)
	})

	t.Run("what changed since stays as it is", func(t *testing.T) {
		c.Do(http.MethodDelete, path, nil).MustStatus(t, 204)
		// Someone edits the paused form meanwhile.
		exec(`UPDATE forms SET updated_at = updated_at + interval '1 second' WHERE workspace_id = $1 AND name = 'Newsletter signup'`, wsID)
		var r struct{ Forms, Left []string }
		c.Do(http.MethodPost, path+"/restore", nil).MustStatus(t, 200).JSON(t, &r)
		if len(r.Forms) != 0 || !slices.Contains(r.Left, "Newsletter signup") {
			t.Fatalf("restore %+v", r)
		}
		if s := status("forms", "Newsletter signup"); s != "paused" {
			t.Errorf("form is %s", s)
		}
	})

	t.Run("after 30 days the list goes for good", func(t *testing.T) {
		c.Do(http.MethodDelete, path, nil).MustStatus(t, 204)
		exec(`UPDATE lists SET deleted_at = now() - interval '31 days' WHERE id = $1`, product)
		c.Do(http.MethodPost, path+"/restore", nil).MustStatus(t, 409)
		w := &maintenance.CleanupWorker{Pool: pool, Log: slog.New(slog.DiscardHandler)}
		if err := w.Work(ctx, nil); err != nil {
			t.Fatal(err)
		}
		var lists, members int
		if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM lists WHERE id = $1), (SELECT count(*) FROM list_memberships WHERE list_id = $1)`, product).Scan(&lists, &members); err != nil {
			t.Fatal(err)
		}
		if lists != 0 || members != 0 {
			t.Fatalf("purge left %d lists and %d memberships", lists, members)
		}
		c.Do(http.MethodPost, path+"/restore", nil).MustStatus(t, 404)
		var contacts int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1`, wsID).Scan(&contacts); err != nil || contacts < 48213 {
			t.Fatalf("contacts after the purge: %d (%v)", contacts, err)
		}
	})
}
