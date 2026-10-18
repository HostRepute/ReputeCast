package placement_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/placement"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/visibility"
)

// testVisibility is the app's adapter over internal/visibility (X11).
type testVisibility struct{ s *visibility.Service }

func (v testVisibility) Versions(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) ([]placement.ConditionalVersion, error) {
	r, err := v.s.Report(ctx, ws, campaignID, visibility.ReportInput{})
	if err != nil {
		return nil, err
	}
	out := []placement.ConditionalVersion{}
	for _, x := range r.Versions {
		h := visibility.Hidden{}
		for _, id := range x.Hidden {
			h[id] = true
		}
		out = append(out, placement.ConditionalVersion{Number: x.Number, Shows: x.Shows, Recipients: x.Recipients, Hidden: h.String()})
	}
	return out, nil
}

func (testVisibility) Cut(html, hidden string) string { return visibility.Cut(html, hidden) }

// Weekly Digest #143 has conditional content (X11's demo): the plan lists
// its versions and picks the one with the most recipients, a test can ask
// for another, and the copies carry the version's blocks only, without the
// markers.
func TestConditionalVersions(t *testing.T) {
	f := setup(t)
	f.svc.Visibility = testVisibility{s: &visibility.Service{Pool: f.env.App.Pool, Q: f.env.App.Q,
		Segments: &segments.Service{Pool: f.env.App.Pool, Q: f.env.App.Q, Audit: f.env.App.Audit, SQL: &segments.SQLRunner{Pool: f.env.App.Pool}}}}
	seed := f.seed(t, "yahoo", "pt-versions@seeds.test", "Bulk", false)
	in := placement.PlacementRunInput{Content: placement.PlacementContentRef{Kind: "campaign", ID: f.campaign(t, "Weekly Digest #143")}, Through: "route",
		Seeds: []string{seed.Seed.ID}}
	p, err := f.svc.Plan(f.ctx, &f.ws, in)
	if err != nil {
		t.Fatal(err)
	}
	vs := p.Content.Conditional
	if len(vs) < 2 {
		t.Fatalf("versions = %+v", vs)
	}
	most := vs[0]
	for _, v := range vs {
		if v.Recipients > most.Recipients {
			most = v
		}
	}
	if p.Content.ConditionalVersion != most.Number {
		t.Fatalf("picked version %d, the most recipients are version %d (%d)", p.Content.ConditionalVersion, most.Number, most.Recipients)
	}

	other := vs[0]
	if other.Number == most.Number {
		other = vs[1]
	}
	in.Version = other.Number
	d, err := f.svc.Start(f.ctx, &f.ws, in)
	if err != nil {
		t.Fatal(err)
	}
	if d.ConditionalVersion != other.Number || d.ConditionalShows != other.Shows {
		t.Fatalf("test sends version %d %q, want %d %q", d.ConditionalVersion, d.ConditionalShows, other.Number, other.Shows)
	}
	var html string
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT html FROM placement_tests WHERE id = $1`, d.ID).Scan(&html); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<!--rc:") {
		t.Fatal("the copies keep the conditional markers")
	}
}
