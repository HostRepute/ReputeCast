package audience_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The Audience header's counts and a segment preview's count are kept for
// a minute (review 2026-11 OP-12): a change made through the app shows at
// once, one made behind its back within the minute.
func TestCountsAreKeptUntilTheAudienceChanges(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	header := func() int64 {
		t.Helper()
		var a struct{ Contacts int64 }
		c.Do(http.MethodGet, ws+"/audience", nil).MustStatus(t, 200).JSON(t, &a)
		return a.Contacts
	}
	type preview struct {
		Count  int64
		Total  int64
		Cached bool
	}
	def := map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{{"field": "tier", "op": "is", "value": "active"}}}}}
	count := func() preview {
		t.Helper()
		var p preview
		c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "builder", "definition": def}).MustStatus(t, 200).JSON(t, &p)
		return p
	}

	before := header()
	first := count()
	if first.Cached {
		t.Fatal("the first count can't be kept")
	}
	if again := count(); !again.Cached || again.Count != first.Count {
		t.Fatalf("the same definition again: %+v, first %+v", again, first)
	}

	// Behind the app's back: kept until the minute is up.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, tier, attributes) VALUES ($1, $2, 'behind@example.com', 'active', '{}')`,
		uuid.Must(uuid.NewV7()), w.ID); err != nil {
		t.Fatal(err)
	}
	if n := header(); n != before {
		t.Fatalf("the header counted again: %d → %d", before, n)
	}

	// Through the app: the audit entry drops the workspace's counts.
	var list string
	if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM lists WHERE workspace_id = $1 ORDER BY created_at LIMIT 1`, w.ID).Scan(&list); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodPost, ws+"/contacts", map[string]any{"email": "through@example.com", "lists": []string{list}, "consent": "single"}).MustStatus(t, 201)
	if n := header(); n != before+2 {
		t.Fatalf("after adding a contact: %d, want %d", n, before+2)
	}
	after := count()
	// The active contact added behind the app's back is counted now.
	if after.Cached || after.Total != first.Total+2 || after.Count <= first.Count {
		t.Fatalf("the preview after the change: %+v, first %+v", after, first)
	}
}
