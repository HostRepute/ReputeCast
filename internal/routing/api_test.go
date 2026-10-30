package routing_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/routing"
)

const ws = "/workspaces/acme-newsletter"

type streamIn struct {
	Stream      string           `json:"stream"`
	Splits      []map[string]any `json:"splits"`
	Fallback    string           `json:"fallback,omitempty"`
	Failover    string           `json:"failover"`
	Conditions  []map[string]any `json:"conditions"`
	BaseVersion string           `json:"baseVersion,omitempty"`
}

type config struct {
	Streams []struct {
		Stream  string
		Version string
		Splits  []struct {
			ProviderID string
			Weight     int
		}
		Fallback   string
		Failover   string
		Conditions []struct {
			Field      string
			Values     []string
			ProviderID string
			Splits     []map[string]any
			Note       string
		}
		Volume *struct {
			Count    int64
			Per      string
			Smallest int64
		}
	}
	Providers []struct {
		ID, Name, Adapter string
		BulkAllowed       bool
		WarmingUp         bool
	}
	SavedBy string
}

// input turns the saved routing back into a save body.
func input(c config) []streamIn {
	out := []streamIn{}
	for _, s := range c.Streams {
		in := streamIn{Stream: s.Stream, Fallback: s.Fallback, Failover: s.Failover, Splits: []map[string]any{}, Conditions: []map[string]any{}}
		for _, sp := range s.Splits {
			in.Splits = append(in.Splits, map[string]any{"providerId": sp.ProviderID, "weight": sp.Weight})
		}
		for _, cd := range s.Conditions {
			c := map[string]any{"field": cd.Field, "values": cd.Values, "providerId": cd.ProviderID, "note": cd.Note}
			if len(cd.Splits) > 0 {
				c["splits"] = cd.Splits
			}
			in.Conditions = append(in.Conditions, c)
		}
		out = append(out, in)
	}
	return out
}

func TestRoutingSaveAndBulkTerms(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var c config
	owner.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &c)
	if len(c.Streams) != 4 || c.Streams[0].Stream != "broadcast" || len(c.Streams[0].Splits) != 2 || len(c.Streams[0].Conditions) != 2 {
		t.Fatalf("routing %+v", c.Streams)
	}
	if v := c.Streams[0].Volume; v == nil || v.Per != "week" || v.Count == 0 || v.Smallest == 0 {
		t.Fatalf("broadcast volume %+v", v)
	}
	ids := map[string]string{}
	for _, p := range c.Providers {
		ids[p.Adapter] = p.ID
	}

	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("PUT", ws+"/routing", map[string]any{"streams": input(c)}).MustStatus(t, 403)

	// Weights must add up to 100.
	bad := input(c)
	bad[0].Splits[1]["weight"] = 20
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": bad}).MustStatus(t, 422)

	// Broadcasts falling back to Postmark's transactional stream breaks its terms.
	bulk := input(c)
	bulk[0].Fallback = ids["postmark"]
	p := owner.Do("PUT", ws+"/routing", map[string]any{"streams": bulk}).MustStatus(t, 409).Problem(t)
	if p["code"] != "bulk_terms" {
		t.Fatalf("problem %+v", p)
	}
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": bulk, "acknowledgeBulkTerms": true}).MustStatus(t, 200)

	// A plain change: Postal's share to 15%.
	next := input(c)
	next[0].Splits[0]["weight"] = 85
	next[0].Splits[1]["weight"] = 15
	var saved config
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": next}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Streams[0].Splits[1].Weight != 15 || saved.SavedBy != "Linh Nguyen" {
		t.Fatalf("saved %+v by %q", saved.Streams[0].Splits, saved.SavedBy)
	}
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'routing.update'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("routing.update audit = %d %v", n, err)
	}
}

func TestRouteTester(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("aisha@acme.io") // viewers may test routes
	type result struct {
		Provider, MailboxProvider string
		DKIMSelector              string `json:"dkimSelector"`
		Steps                     []struct {
			Kind, Field, Value string
			Bucket             int
		}
		Fallback struct{ Policy string }
		Cost     struct {
			Basis   string
			Per1000 *float64
		}
	}
	var r result
	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": "sam.rivera@outlook.com", "senderDomain": "news.acme.io"}).MustStatus(t, 200).JSON(t, &r)
	if r.MailboxProvider != "outlook" || r.Fallback.Policy != "pause_and_ask" || r.Steps[1].Kind != "split" {
		t.Fatalf("outlook route %+v", r)
	}
	// A condition that splits by weight (PR-12): product announcements from
	// updates.acme.io go SES 70 / Postal 30, sticky per address.
	var cond struct {
		Provider string
		Steps    []struct {
			Kind, Field, Value string
			Bucket             *int
			Splits             []struct {
				Short  string
				Weight int
			}
		}
	}
	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": "sam.rivera@outlook.com", "senderDomain": "updates.acme.io"}).MustStatus(t, 200).JSON(t, &cond)
	if s := cond.Steps[1]; s.Kind != "condition" || s.Field != "sender_domain" || s.Value != "updates.acme.io" || len(s.Splits) != 2 || s.Splits[0].Weight != 70 || s.Bucket == nil {
		t.Fatalf("condition split steps %+v", cond.Steps)
	}

	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": "pat@gmail.com"}).MustStatus(t, 200).JSON(t, &r)
	if r.Steps[1].Kind != "split" || r.Steps[1].Bucket < 0 || r.Steps[1].Bucket > 99 {
		t.Fatalf("gmail steps %+v", r.Steps)
	}
	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "system", "email": "pat@gmail.com"}).MustStatus(t, 200).JSON(t, &r)
	if r.Provider != "Postmark" || r.Fallback.Policy != "automatic" {
		t.Fatalf("system route %+v", r)
	}
	owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": "not-an-address"}).MustStatus(t, 422)
}

// PR-2: a save replaces only the streams it lists, each checked against the
// version it started from, and keeps a split in the order it was saved;
// reordering a split moves nobody.
func TestRoutingSavesChangedStreamsWithVersions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	ctx := context.Background()
	var c config
	owner.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &c)
	for _, s := range c.Streams {
		if s.Version == "" {
			t.Fatalf("stream %s has no version", s.Stream)
		}
	}
	rules := func(stream string) []string {
		rows, err := env.App.Pool.Query(ctx, `SELECT r.id::text || r.updated_at::text FROM routing_rules r JOIN workspaces w ON w.id = r.workspace_id
			WHERE w.slug = 'acme-newsletter' AND r.stream = $1 ORDER BY r.id`, stream)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatal(err)
			}
			out = append(out, s)
		}
		return out
	}
	automation := rules("automation")
	route := func(email string) string {
		var r struct{ ProviderID string }
		owner.Do("POST", ws+"/routing/test", map[string]any{"stream": "broadcast", "email": email}).MustStatus(t, 200).JSON(t, &r)
		return r.ProviderID
	}
	addresses := make([]string, 40)
	before := map[string]string{}
	for i := range addresses {
		addresses[i] = fmt.Sprintf("reader%d@gmail.com", i)
		before[addresses[i]] = route(addresses[i])
	}

	// Broadcasts only, with its split reordered (Postal first).
	b := input(c)[0]
	b.Splits[0], b.Splits[1] = b.Splits[1], b.Splits[0]
	b.BaseVersion = c.Streams[0].Version
	var saved config
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": []streamIn{b}}).MustStatus(t, 200).JSON(t, &saved)
	if got := saved.Streams[0].Splits; got[0].ProviderID != c.Streams[0].Splits[1].ProviderID || got[0].Weight != c.Streams[0].Splits[1].Weight {
		t.Fatalf("the split keeps its saved order: %+v", got)
	}
	if saved.Streams[0].Version == c.Streams[0].Version || saved.Streams[1].Version != c.Streams[1].Version {
		t.Fatalf("versions before %s/%s after %s/%s", c.Streams[0].Version, c.Streams[1].Version, saved.Streams[0].Version, saved.Streams[1].Version)
	}
	if got := rules("automation"); !slices.Equal(got, automation) {
		t.Fatalf("a Broadcasts save rewrote Automations: %v → %v", automation, got)
	}
	for _, a := range addresses {
		if got := route(a); got != before[a] {
			t.Fatalf("%s moved from %s to %s when the split was reordered", a, before[a], got)
		}
	}

	// A tab still on the old version is refused and saves nothing.
	stale := input(c)[0]
	stale.Splits[0]["weight"], stale.Splits[1]["weight"] = 80, 20
	stale.BaseVersion = c.Streams[0].Version
	p := owner.Do("PUT", ws+"/routing", map[string]any{"streams": []streamIn{stale}}).MustStatus(t, 409).Problem(t)
	meta, _ := p["meta"].(map[string]any)
	if p["code"] != "routing_changed" || fmt.Sprint(meta["streams"]) != "[broadcast]" {
		t.Fatalf("stale save: %+v", p)
	}
	var after config
	owner.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &after)
	if after.Streams[0].Version != saved.Streams[0].Version {
		t.Fatal("a refused save changed the routing")
	}

	// Another stream saves fine on its own version.
	a := input(c)[1]
	a.Failover = "pause_and_ask"
	a.BaseVersion = c.Streams[1].Version
	owner.Do("PUT", ws+"/routing", map[string]any{"streams": []streamIn{a}}).MustStatus(t, 200).JSON(t, &after)
	if after.Streams[1].Failover != "pause_and_ask" || after.Streams[0].Version != saved.Streams[0].Version {
		t.Fatalf("automation save: %+v", after.Streams[1])
	}
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'routing.update' AND metadata->'streams' = '["automation"]'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("audit for the automation save = %d %v", n, err)
	}
}

// PR-14: the route tester's Campaign tab runs the next broadcast's audience
// through the saved routing and an unsaved draft. Moving the split from
// 90/10 to 85/15 moves only the difference, from SES to Postal, and the
// counts add up to the audience.
func TestCampaignSimulation(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("aisha@acme.io") // viewers may test routes
	var c config
	owner.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &c)

	// The shared card counts what providers get (PR-1), and a failover
	// target's quota says what it counts (Postmark's month, SES's day).
	var view struct {
		Shared  struct{ Suppressions, Pushed int64 }
		Streams []struct {
			Stream      string
			FailoverVet *struct {
				Checks []struct{ Key, Result, Period string }
			}
		}
	}
	owner.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &view)
	if view.Shared.Pushed == 0 || view.Shared.Pushed >= view.Shared.Suppressions {
		t.Fatalf("shared %+v", view.Shared)
	}
	periods := map[string]string{}
	for _, st := range view.Streams {
		if st.FailoverVet == nil {
			continue
		}
		for _, ch := range st.FailoverVet.Checks {
			if ch.Key == "quota" {
				periods[st.Stream] = ch.Period
			}
		}
	}
	if periods["automation"] != "month" || periods["system"] != "day" {
		t.Fatalf("quota periods %v", periods)
	}
	type share struct {
		ProviderID string
		Count      int64
	}
	type result struct {
		None          bool
		Name          string
		Total, Sunset int64
		Shares        []share
		Conditions    []struct {
			Field string
			Count int64
		}
		Draft *struct {
			Shares []share
			Moved  int64
		}
	}
	var sim result
	owner.Do("POST", ws+"/routing/test-campaign", map[string]any{}).MustStatus(t, 200).JSON(t, &sim)
	if sim.None || sim.Total == 0 || len(sim.Shares) < 2 || sim.Draft != nil {
		t.Fatalf("simulation %+v", sim)
	}
	var sum int64
	saved := map[string]int64{}
	for _, s := range sim.Shares {
		sum += s.Count
		saved[s.ProviderID] = s.Count
	}
	if sum != sim.Total {
		t.Fatalf("shares add up to %d of %d", sum, sim.Total)
	}

	in := input(c)
	var postal string
	for i := range in {
		if in[i].Stream == "broadcast" {
			in[i].Splits[0]["weight"], in[i].Splits[1]["weight"] = 85, 15
			postal = in[i].Splits[1]["providerId"].(string)
		}
	}
	var draft result
	owner.Do("POST", ws+"/routing/test-campaign", map[string]any{"draft": map[string]any{"streams": in}}).MustStatus(t, 200).JSON(t, &draft)
	if draft.Draft == nil || draft.Total != sim.Total {
		t.Fatalf("draft simulation %+v", draft)
	}
	after := map[string]int64{}
	for _, s := range draft.Draft.Shares {
		after[s.ProviderID] = s.Count
	}
	gained := after[postal] - saved[postal]
	if gained <= 0 || draft.Draft.Moved != gained {
		t.Fatalf("85/15 moved %d, Postal gained %d (saved %v, draft %v)", draft.Draft.Moved, gained, saved, after)
	}

	// A larger audience than one simulation reads: the first recipients
	// go through the routing and the counts are scaled to the whole
	// audience, whose size stays exact.
	wsRow, err := env.App.Q.GetWorkspaceBySlug(context.Background(), "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	small := &routing.Service{Pool: env.App.Pool, Q: env.App.Q, SimulateLimit: 1000}
	est, err := small.SimulateCampaign(context.Background(), &wsRow, routing.RouteCampaignInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !est.Estimated || est.Simulated == 0 || est.Simulated > 1000 || est.Total != sim.Total || est.Sunset != sim.Sunset || len(est.Shares) != len(sim.Shares) {
		t.Fatalf("estimate: %+v (exact %+v)", est, sim)
	}
	var estSum int64
	for _, s := range est.Shares {
		estSum += s.Count
	}
	if d := estSum - est.Total; d < -int64(len(est.Shares)) || d > int64(len(est.Shares)) {
		t.Fatalf("estimated shares add up to %d of %d", estSum, est.Total)
	}
}
