package demo

import (
	"context"
	"io"
	"log/slog"
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/scoring"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// The scoring and growth sample (audience-insights brief §4): the numbers
// ScoringRules, ScoringStates, ScoringProfile, GrowthSources and
// GrowthStates show, read back through the services that serve them.
func TestScoringGrowthSample(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds the whole demo")
	}
	pool := testdb.New(t)
	cfg := config.Defaults()
	box, err := secrets.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := Seed(ctx, pool, &cfg, box, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("seed: %v", err)
	}
	q := store.New(pool)
	ws, err := q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}

	// Scoring: the spread, bands by tier, and rule 3 at +25.
	sc := &scoring.Service{Pool: pool, Q: q}
	page, err := sc.Page(ctx, &ws)
	if err != nil {
		t.Fatal(err)
	}
	spread := []int64{18406, 10784, 6912, 4401, 2894, 1966, 1207, 806, 497, 340}
	for i, b := range page.Spread {
		if b.Contacts != spread[i] {
			t.Errorf("bucket %d–%d: %d, want %d", b.From, b.To, b.Contacts, spread[i])
		}
	}
	tiers := map[string][3]int64{"active": {4702, 12960, 11750}, "cooling": {114, 1198, 8572}, "at_risk": {0, 49, 5253}, "sunset": {0, 0, 3615}}
	for _, r := range page.ByTier {
		if w := tiers[r.Tier]; r.Hot != w[0] || r.Warm != w[1] || r.Cold != w[2] {
			t.Errorf("tier %s: %d / %d / %d, want %v", r.Tier, r.Hot, r.Warm, r.Cold, w)
		}
	}
	edited := page.Config.Clone()
	edited.Rules[2].Points = 25
	imp, err := sc.Preview(ctx, &ws, scoring.ScoringPreviewInput{Config: edited})
	if err != nil {
		t.Fatal(err)
	}
	if imp.Changed != 1312 || imp.BandsAfter != (scoring.ScoringBandCounts{Cold: 28986, Warm: 13303, Hot: 5924}) {
		t.Errorf("rule 3 at +25: %d change band, %+v", imp.Changed, imp.BandsAfter)
	}
	if len(imp.Flows) != 1 || imp.Flows[0].Crossed != 1108 || imp.Flows[0].Name != "Sales-ready handoff" {
		t.Errorf("automations: %+v", imp.Flows)
	}
	scores := map[string]int{"sam.rivera@outlook.com": 52, "priya.n@gmail.com": 78, "d.alvarez@yahoo.com": 12, "hannah@bergconsulting.de": 47,
		"kenji.w@icloud.com": 18, "lena.fischer@hotmail.com": 0, "omar.haddad@gmail.com": 64, "g.okafor@outlook.com": 0, "t.nowak@gmail.com": 21, "aisha@rahmanlabs.io": 91}
	var sam uuid.UUID
	for email, want := range scores {
		var id uuid.UUID
		var got int
		if err := pool.QueryRow(ctx, `SELECT c.id, cs.score FROM contacts c JOIN contact_scores cs ON cs.contact_id = c.id WHERE c.email = $1`, email).Scan(&id, &got); err != nil || got != want {
			t.Errorf("%s: %d, want %d (%v)", email, got, want, err)
		}
		if email == "sam.rivera@outlook.com" {
			sam = id
		}
	}
	card, err := sc.ContactScore(ctx, &ws, sam, nil)
	if err != nil {
		t.Fatal(err)
	}
	lc := card.LastChange
	if card.Score != 52 || lc == nil || lc.Before != 47 || lc.After != 52 || lc.Cause.EmailName != "October Product Update" || lc.Cause.LinkLabel != "Read the docs" {
		t.Errorf("Sam: %d, last change %+v", card.Score, lc)
	}
	var hotPaid, entered int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM contacts c JOIN contact_scores cs ON cs.contact_id = c.id
		WHERE c.workspace_id = $1 AND cs.score >= 50 AND lower(c.attributes ->> 'plan') IN ('team', 'business')`, ws.ID).Scan(&hotPaid); err != nil || hotPaid != 1532 {
		t.Errorf("Hot on paid plans: %d (%v)", hotPaid, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM automation_runs r JOIN automations a ON a.id = r.automation_id WHERE a.name = 'Sales-ready handoff'`).Scan(&entered); err != nil || entered != 37 {
		t.Errorf("Sales-ready handoff entered: %d (%v)", entered, err)
	}

	// Growth: the last 30 days.
	gs := &growth.Service{Pool: pool, Q: q}
	g, err := gs.Page(ctx, &ws, growth.GrowthQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if g.Tiles.Joined != 1756 || g.Tiles.Left != 472 || g.Tiles.Net != 1284 || g.Tiles.Imports != 9509 || math.Round(g.Tiles.NetPct*1000) != 27 {
		t.Errorf("tiles: %+v", g.Tiles)
	}
	if g.Departures != (growth.GrowthDepartures{Unsubscribed: 333, HardBounced: 104, Complained: 27, Erased: 8}) {
		t.Errorf("departures: %+v", g.Departures)
	}
	joined := map[string]int64{}
	for _, r := range g.Sources {
		if r.Joined != nil {
			joined[r.Source.Name] = *r.Joined
		}
	}
	for name, want := range map[string]int64{"Newsletter signup": 731, "Footer mini form": 612, "Beta apply": 43, "Website signup form": 214, "Zapier · Beta application (Typeform)": 38} {
		if joined[name] != want {
			t.Errorf("%s joined %d, want %d", name, joined[name], want)
		}
	}
	if math.Round(g.Workspace.Bounce*1000) != 6 || math.Round(g.Workspace.Complaint*10000) != 4 {
		t.Errorf("workspace rates: %+v", g.Workspace)
	}
	if g.UTM.Joins != 1490 || g.UTM.None.Joins != 716 || len(g.UTM.Rows) != 5 || g.UTM.Rows[0].Joins != 214 || g.UTM.Other.Count != 14 || g.UTM.Other.Joins != 116 {
		t.Errorf("utm: %+v", g.UTM)
	}
	if g.Referrers.None != 493 || len(g.Referrers.Rows) != 5 || g.Referrers.Rows[0].Joins != 241 || g.Referrers.Other.Count != 52 {
		t.Errorf("referrers: %+v", g.Referrers)
	}
	cohorts := []struct {
		n    int64
		r30  float64
		r90  float64
		done bool
	}{{1402, 0.959, 0.912, true}, {1288, 0.963, 0, false}, {1351, 0.951, 0, false}, {1612, 0, 0, false}}
	if len(g.Cohorts) != 4 {
		t.Fatalf("cohorts: %+v", g.Cohorts)
	}
	for i, c := range g.Cohorts {
		w := cohorts[i]
		if c.Contacts != w.n || (w.r30 > 0 && (c.After30.Rate == nil || math.Round(*c.After30.Rate*1000)/1000 != w.r30)) ||
			(w.done && (c.After90.Rate == nil || math.Round(*c.After90.Rate*1000)/1000 != w.r90)) {
			t.Errorf("cohort %s: %+v", c.Month, c)
		}
	}
	var beta uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM lists WHERE workspace_id = $1 AND name = 'Beta Program'`, ws.ID).Scan(&beta); err != nil {
		t.Fatal(err)
	}
	one, err := gs.Page(ctx, &ws, growth.GrowthQuery{List: beta.String()})
	if err != nil {
		t.Fatal(err)
	}
	if one.Tiles.Joined != 108 || one.Tiles.Left != 27 || one.Tiles.Net != 81 {
		t.Errorf("Beta Program: %+v", one.Tiles)
	}
	eng, err := q.GetWorkspaceBySlug(ctx, "acme-engineering-blog")
	if err != nil {
		t.Fatal(err)
	}
	if p, err := gs.Page(ctx, &eng, growth.GrowthQuery{}); err != nil || !p.OnlyImports {
		t.Errorf("Acme Engineering Blog has only imports: %v %v", p.OnlyImports, err)
	}
}
