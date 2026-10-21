package scoring_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/scoring"
)

type page struct {
	Configured     bool                  `json:"configured"`
	Version        int                   `json:"version"`
	AppliedVersion int                   `json:"appliedVersion"`
	Config         scoring.ScoringConfig `json:"config"`
	Contacts       int64                 `json:"contacts"`
	Bands          struct{ Cold, Warm, Hot int64 }
	UsedBy         []struct {
		Kind, Name          string
		Where, Status, Href string
		Entered             int64
		Contacts, Of        *int64
	} `json:"usedBy"`
	Recalculation *struct {
		ID, Status string
		Total      int64
		Done       int64
		Summary    *scoring.ScoringRecalcSummary
	} `json:"recalculation"`
}

func service(env *apptest.Env) *scoring.Service {
	return &scoring.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}
}

func countOf(t *testing.T, env *apptest.Env, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// The demo's set-up reads as ScoringRules shows it, for everyone with
// subscribers:view; only scoring:edit changes it.
func TestScoringPageAndPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	for _, email := range []string{"linh@acme.io", "daniel@acme.io", "aisha@acme.io"} {
		c := env.Client()
		c.SignInDemo(email)
		var p page
		c.Do("GET", "/workspaces/acme-newsletter/scoring", nil).MustStatus(t, 200).JSON(t, &p)
		// The boards' 48,213 and Cold 29,190, with the 338 registrants new to
		// Acme Summit 2026 (X21, brief §8.12), who start at 0.
		if !p.Configured || p.Contacts != 48213+338 || p.Bands.Hot != 4816 || p.Bands.Warm != 14207 || p.Bands.Cold != 29190+338 || len(p.Config.Rules) != 8 {
			t.Fatalf("%s: %+v", email, p)
		}
		// Used by · 3: the automation, the segment and #143's Book a demo
		// block (conditional content, Score band is Hot).
		if len(p.UsedBy) != 3 || p.UsedBy[0].Name != "Sales-ready handoff" || p.UsedBy[0].Entered != 37 || p.UsedBy[1].Contacts == nil || *p.UsedBy[1].Contacts != 1532 {
			t.Fatalf("used by: %+v", p.UsedBy)
		}
		if b := p.UsedBy[2]; b.Kind != "block" || b.Name != "Book a demo" || b.Where != "Weekly Digest #143" || b.Status != "draft" ||
			b.Contacts == nil || *b.Contacts != 3164 || b.Of == nil || *b.Of != 31604 || !strings.HasSuffix(b.Href, "/edit?select=10&panel=settings") {
			t.Fatalf("the conditional block: %+v", b)
		}
		if email == "linh@acme.io" {
			continue
		}
		// Daniel (Content Writer) and Aisha (Viewer) read; editing needs scoring:edit.
		body := map[string]any{"config": p.Config, "baseVersion": p.Version}
		c.Do("PUT", "/workspaces/acme-newsletter/scoring", body).MustStatus(t, 403)
		c.Do("POST", "/workspaces/acme-newsletter/scoring/preview", map[string]any{"config": p.Config}).MustStatus(t, 403)
		c.Do("POST", "/workspaces/acme-newsletter/scoring/recalculate", nil).MustStatus(t, 403)
	}
}

// Sofia changes rule 3 from +15 to +25 (ScoringStates): the preview says
// who moves, saving recalculates every contact in the background, keeps the
// previous scores until it finishes, and doesn't start "Sales-ready
// handoff" for the 1,108 who cross 50 only because of the change.
func TestScoringRuleChange(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("sofia@acme.io")
	var p page
	c.Do("GET", "/workspaces/acme-newsletter/scoring", nil).MustStatus(t, 200).JSON(t, &p)
	edited := p.Config.Clone()
	edited.Rules[2].Points = 25

	var imp scoring.ScoringImpact
	c.Do("POST", "/workspaces/acme-newsletter/scoring/preview", map[string]any{"config": edited}).MustStatus(t, 200).JSON(t, &imp)
	// Cold counts the Summit's 338 new registrants (X21): no events, still 0.
	if imp.Changed != 1312 || imp.BandsAfter != (scoring.ScoringBandCounts{Cold: 28986 + 338, Warm: 13303, Hot: 5924}) || len(imp.Flows) != 1 || imp.Flows[0].Crossed != 1108 {
		t.Fatalf("preview: %+v", imp)
	}
	// A cap that can't hold one event's points is refused.
	bad := p.Config.Clone()
	bad.Rules[1].Cap = scoring.ScoringRuleCap{Kind: scoring.CapPointsWeek, Points: 3}
	pr := c.Do("PUT", "/workspaces/acme-newsletter/scoring", map[string]any{"config": bad, "baseVersion": p.Version}).MustStatus(t, 422).Problem(t)
	if b, _ := json.Marshal(pr); !json.Valid(b) || len(b) == 0 {
		t.Fatalf("problem: %v", pr)
	}
	// Someone else saved in between: a conflict.
	c.Do("PUT", "/workspaces/acme-newsletter/scoring", map[string]any{"config": edited, "baseVersion": p.Version - 1}).MustStatus(t, 409)

	before := countOf(t, env, `SELECT count(*) FROM automation_runs r JOIN automations a ON a.id = r.automation_id WHERE a.name = 'Sales-ready handoff'`)
	var saved page
	c.Do("PUT", "/workspaces/acme-newsletter/scoring", map[string]any{"config": edited, "baseVersion": p.Version}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Version != 2 || saved.AppliedVersion != 1 || saved.Recalculation == nil || saved.Recalculation.Status != "running" || saved.Recalculation.Total != 48213+338 {
		t.Fatalf("saved: %+v %+v", saved, saved.Recalculation)
	}
	// Until it finishes, contacts keep their previous scores.
	if saved.Bands.Hot != 4816 {
		t.Fatalf("bands while recalculating: %+v", saved.Bands)
	}
	if err := service(env).Recalculate(ctx, uuid.MustParse(saved.Recalculation.ID)); err != nil {
		t.Fatal(err)
	}
	var done page
	c.Do("GET", "/workspaces/acme-newsletter/scoring", nil).MustStatus(t, 200).JSON(t, &done)
	// The Summit's 338 new registrants (X21) stay Cold: no events, still 0.
	if done.AppliedVersion != 2 || done.Bands.Hot != 5924 || done.Bands.Warm != 13303 || done.Bands.Cold != 28986+338 {
		t.Fatalf("after: %+v", done.Bands)
	}
	if rc := done.Recalculation; rc == nil || rc.Status != "finished" || rc.Summary == nil || rc.Summary.Changed != 1312 || len(rc.Summary.Flows) != 1 || rc.Summary.Flows[0].Started != 0 {
		t.Fatalf("recalculation: %+v", rc)
	}
	// No one entered the automation because of the change.
	if after := countOf(t, env, `SELECT count(*) FROM automation_runs r JOIN automations a ON a.id = r.automation_id WHERE a.name = 'Sales-ready handoff'`); after != before {
		t.Fatalf("Sales-ready handoff: %d → %d", before, after)
	}
	if n := countOf(t, env, `SELECT count(*) FROM automation_events WHERE kind = 'score_crosses'`); n != 0 {
		t.Fatalf("score_crosses events: %d", n)
	}
	// The audit log and Sofia's notification.
	for _, action := range []string{"scoring.rule_updated", "scoring.recalculation_started", "scoring.recalculation_finished"} {
		if n := countOf(t, env, `SELECT count(*) FROM audit_events WHERE action = $1 AND occurred_at > now() - interval '1 hour'`, action); n != 1 {
			t.Errorf("audit %s: %d", action, n)
		}
	}
	if n := countOf(t, env, `SELECT count(*) FROM notifications n JOIN users u ON u.id = n.user_id WHERE u.email = 'sofia@acme.io' AND n.kind = 'score_recalculated'`); n != 1 {
		t.Errorf("notification: %d", n)
	}
}

// "Start it for 1,108 contacts": the recalculation queues the trigger for
// the contacts who cross because of the change.
func TestScoringRuleChangeStartsAutomations(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	var p page
	c.Do("GET", "/workspaces/acme-newsletter/scoring", nil).MustStatus(t, 200).JSON(t, &p)
	edited := p.Config.Clone()
	edited.Rules[2].Points = 25
	var saved page
	c.Do("PUT", "/workspaces/acme-newsletter/scoring", map[string]any{"config": edited, "baseVersion": p.Version, "startAutomations": true}).MustStatus(t, 200).JSON(t, &saved)
	if err := service(env).Recalculate(ctx, uuid.MustParse(saved.Recalculation.ID)); err != nil {
		t.Fatal(err)
	}
	if n := countOf(t, env, `SELECT count(*) FROM automation_events WHERE kind = 'score_crosses' AND payload ->> 'cause' = 'rule_change'`); n != 1108 {
		t.Fatalf("trigger events: %d", n)
	}
}

// New activity is scored within a tick; the nightly run applies field
// conditions; a band-only change applies at once.
func TestScoringTickNightlyAndBands(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	svc := service(env)
	pool := env.App.Pool
	var ws, contact, campaign, provider uuid.UUID
	var score int
	if err := pool.QueryRow(ctx, `SELECT c.workspace_id, c.id, cs.score FROM contacts c JOIN contact_scores cs ON cs.contact_id = c.id
		WHERE c.email = 'hannah@bergconsulting.de'`).Scan(&ws, &contact, &score); err != nil || score != 47 {
		t.Fatalf("Hannah: %d %v", score, err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM campaigns WHERE name = 'October Product Update'`).Scan(&campaign); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM providers WHERE workspace_id = $1 ORDER BY id LIMIT 1`, ws).Scan(&provider); err != nil {
		t.Fatal(err)
	}
	// Hannah clicks a link in October Product Update: +5 for the click and
	// +2 for its open unless this week's opens are at their cap, 47 → 52
	// or 54: she crosses 50 and the live trigger fires.
	msg := uuid.Must(uuid.NewV7())
	now := time.Now()
	if _, err := pool.Exec(ctx, `INSERT INTO messages (id, workspace_id, campaign_id, contact_id, provider_id, stream, status, email, sent_at, delivered_at, queued_at)
		VALUES ($1,$2,$3,$4,$5,'broadcast','delivered','hannah@bergconsulting.de',$6,$6,$6)`, msg, ws, campaign, contact, provider, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, type, occurred_at, url, received_at)
		VALUES ($1,$2,$3,$4,$5,$6,'click',$7,'https://docs.acme.io/flags',$8)`, uuid.Must(uuid.NewV7()), ws, msg, campaign, contact, provider, now.Add(-time.Minute), now); err != nil {
		t.Fatal(err)
	}
	svc.Now = func() time.Time { return now.Add(5 * time.Second) }
	if err := svc.TickWorkspace(ctx, ws); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT score FROM contact_scores WHERE contact_id = $1`, contact).Scan(&score); err != nil || (score != 52 && score != 54) {
		t.Fatalf("after the click: %d %v", score, err)
	}
	clicked := score
	if n := countOf(t, env, `SELECT count(*) FROM automation_events WHERE kind = 'score_crosses' AND contact_id = $1 AND payload ->> 'cause' = 'activity'`, contact); n != 1 {
		t.Fatalf("trigger event: %d", n)
	}
	// The same events again change nothing (ticks overlap on purpose).
	if err := svc.TickWorkspace(ctx, ws); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT score FROM contact_scores WHERE contact_id = $1`, contact).Scan(&score); err != nil || score != clicked {
		t.Fatalf("second tick: %d %v", score, err)
	}
	// She moves to the business plan: the nightly run adds rule 6's +10.
	if _, err := pool.Exec(ctx, `UPDATE contacts SET attributes = attributes || '{"plan":"business"}' WHERE id = $1`, contact); err != nil {
		t.Fatal(err)
	}
	res, err := svc.NightlyWorkspace(ctx, ws, "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT score FROM contact_scores WHERE contact_id = $1`, contact).Scan(&score); err != nil || score != clicked+10 || res.FieldsChanged != 1 {
		t.Fatalf("nightly: %d, %+v %v", score, res, err)
	}
	// Moving a threshold applies at once, without a recalculation.
	c := env.Client()
	c.SignInDemo("sofia@acme.io")
	var p page
	c.Do("GET", "/workspaces/acme-newsletter/scoring", nil).MustStatus(t, 200).JSON(t, &p)
	cfg := p.Config.Clone()
	cfg.Bands.HotFrom = 60
	var saved page
	c.Do("PUT", "/workspaces/acme-newsletter/scoring", map[string]any{"config": cfg, "baseVersion": p.Version}).MustStatus(t, 200).JSON(t, &saved)
	if saved.AppliedVersion != saved.Version || saved.Bands.Hot >= 4816 {
		t.Fatalf("bands: %+v", saved)
	}
	if n := countOf(t, env, `SELECT count(*) FROM audit_events WHERE action = 'scoring.bands_updated'`); n != 1 {
		t.Fatalf("audit: %d", n)
	}
}

// Sam Rivera's card (ScoringProfile).
func TestContactScore(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("daniel@acme.io")
	var sam uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM contacts WHERE email = 'sam.rivera@outlook.com'`).Scan(&sam); err != nil {
		t.Fatal(err)
	}
	var card scoring.ContactScore
	c.Do("GET", "/workspaces/acme-newsletter/contacts/"+sam.String()+"/score", nil).MustStatus(t, 200).JSON(t, &card)
	if card.Score != 52 || card.Band != "hot" || card.LastChange == nil || card.LastChange.Delta != 5 || card.LastChange.Cause.LinkLabel != "Read the docs" {
		t.Fatalf("card: %+v %+v", card, card.LastChange)
	}
	points := map[int]int{}
	for _, r := range card.Breakdown {
		points[r.Number] = r.Points
		if r.Number == 7 && (r.Points != 0 || r.ZeroEmail != "Weekly Digest #139") {
			t.Fatalf("soft bounce: %+v", r)
		}
	}
	if points[1] != 22 || points[2] != 20 || points[6] != 10 {
		t.Fatalf("breakdown: %v", points)
	}
	var in, entered bool
	for _, u := range card.Uses {
		switch u.Kind {
		case "segment":
			in = u.In
		case "automation":
			entered = u.In
			if u.CrossedAt == nil || u.LiveSince == nil || !u.CrossedAt.Before(*u.LiveSince) {
				t.Fatalf("crossed before it went live: %+v", u)
			}
		}
	}
	if !in || entered {
		t.Fatalf("uses: %+v", card.Uses)
	}
	if len(card.Line) != 91 || card.Line[90].Score == nil || *card.Line[90].Score != 52 {
		t.Fatalf("line: %d points", len(card.Line))
	}
}
