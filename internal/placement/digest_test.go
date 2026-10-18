package placement_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/digests"
	"github.com/hostrepute/reputecast/internal/placement"
)

// A digest's next issue (brief §3.3): the search offers the digest rather
// than its hidden template and next-issue campaigns, planning builds the
// issue once and reuses it while the dialog changes, and starting a test
// builds it again so the copies are the issue as it would go out then.
func TestDigestNextIssue(t *testing.T) {
	f := setup(t)
	f.svc.Digests = &digests.Service{Pool: f.env.App.Pool, Q: f.env.App.Q, Audit: f.env.App.Audit, URLs: f.env.App.URLs}
	seed := f.seed(t, "yahoo", "pt-digest@seeds.test", "Bulk", false)
	ctx := context.Background()
	var dg uuid.UUID
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT id FROM digests WHERE workspace_id = $1 AND name = 'Engineering blog roundup'`, f.ws.ID).Scan(&dg); err != nil {
		t.Fatalf("the demo's digest: %v", err)
	}

	found, err := f.svc.SearchContent(f.ctx, &f.ws, "Engineering blog")
	if err != nil {
		t.Fatal(err)
	}
	// Issues already sent (#1, #2) are campaigns like any other.
	var digest, hidden int
	for _, it := range found.Items {
		switch {
		case it.Kind == "digest" && it.ID == dg.String() && it.Note == "on":
			digest++
		case it.Kind == "campaign" && it.Note != "sent":
			hidden++
		}
	}
	if digest != 1 || hidden != 0 {
		t.Fatalf("search = %+v", found.Items)
	}

	in := placement.PlacementRunInput{Content: placement.PlacementContentRef{Kind: "digest", ID: dg.String()}, Through: "route", Seeds: []string{seed.Seed.ID}}
	p, err := f.svc.Plan(f.ctx, &f.ws, in)
	if err != nil {
		t.Fatal(err)
	}
	if p.Content.Kind != "digest" || p.Content.ID != dg.String() || p.Content.Name != "Engineering blog roundup #3" || p.Stream != "broadcast" || p.Copies != 1 {
		t.Fatalf("plan content = %+v, stream %s, copies %d", p.Content, p.Stream, p.Copies)
	}
	version := func() (id uuid.UUID, v int) {
		t.Helper()
		if err := f.env.App.Pool.QueryRow(ctx, `SELECT c.id, c.version FROM digest_issues i JOIN campaigns c ON c.id = i.campaign_id
			WHERE i.digest_id = $1 AND i.status = 'collecting'`, dg).Scan(&id, &v); err != nil {
			t.Fatalf("the next issue's campaign: %v", err)
		}
		return id, v
	}
	cid, v1 := version()
	if _, err := f.svc.Plan(f.ctx, &f.ws, in); err != nil {
		t.Fatal(err)
	}
	if _, v := version(); v != v1 {
		t.Fatalf("planning again rebuilt the issue: version %d, then %d", v1, v)
	}

	d, err := f.svc.Start(f.ctx, &f.ws, in)
	if err != nil {
		t.Fatal(err)
	}
	if d.ContentKind != "digest" || d.ContentID != dg.String() || d.ContentNote != "digest" || d.Counts.Copies != 1 {
		t.Fatalf("test = %s %s %s, %d copies", d.ContentKind, d.ContentID, d.ContentNote, d.Counts.Copies)
	}
	if id, v := version(); id != cid || v != v1+1 {
		t.Fatalf("starting didn't build the issue again: %s v%d, then %s v%d", cid, v1, id, v)
	}
}
