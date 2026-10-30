package engine_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers/simulator"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// SR-5 (C4): the estimate made before a send matches the send itself,
// played on a virtual clock with the demo transport: SES and Postal (at its
// warm-up ramp) in parallel, the mailbox throttles, and the sample, wait and
// rest of Weekly Digest #142's A/B test.
func TestEstimateMatchesTheSend(t *testing.T) {
	s := sendtest.NewSuite(t)
	t.Run("ABTestOnTwoRoutes", func(t *testing.T) {
		f := s.Fixture(t, "Estimate A/B")
		f.AddContacts(sendtest.ContactSpec{N: 900, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
			sendtest.ContactSpec{N: 450, Mailbox: "outlook", Domain: "outlook.com", Engaged: true},
			sendtest.ContactSpec{N: 150, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true})
		compareWithSend(t, f)
	})
	t.Run("ThrottledMailboxes", func(t *testing.T) {
		f := s.Fixture(t, "Estimate throttles")
		f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
		f.AddContacts(sendtest.ContactSpec{N: 1200, Mailbox: "outlook", Domain: "outlook.com", Engaged: true},
			sendtest.ContactSpec{N: 300, Mailbox: "icloud", Domain: "icloud.com", Engaged: true})
		compareWithSend(t, f)
	})
}

// compareWithSend starts the fixture's send, estimates it from what the
// audience build planned (recipients per provider and mailbox provider),
// drives it to the end and compares when its 99th percentile message went
// out: within 5% of the send's length or 20 seconds. It sends through the
// steady demo transport: a provider that throttles a call slows its route
// (to half its rate for a minute at least), which no estimate can know in
// advance; the engine's answer to throttling has tests of its own. For the
// same reason the send runs with limits it can't reach: a few chance block
// bounces at Outlook would pause it there and end the send early.
func compareWithSend(t *testing.T, f *sendtest.Fixture) {
	t.Helper()
	limitsOutOfReach(f)
	f.StartSend(engine.StartOptions{Scenario: simulator.ScenarioSteady})
	cells := map[uuid.UUID]map[string]int64{}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT provider_id, mailbox_provider, count(*) FROM messages WHERE campaign_id = $1 GROUP BY 1, 2`, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var p uuid.UUID
		var mbp string
		var n int64
		if err := rows.Scan(&p, &mbp, &n); err != nil {
			t.Fatal(err)
		}
		if cells[p] == nil {
			cells[p] = map[string]int64{}
		}
		cells[p][mbp] = n
	}
	rows.Close()
	est, err := f.E.Estimate(f.Ctx, f.Campaign, engine.EstimateOptions{Start: f.Start, Recipients: cells})
	if err != nil {
		t.Fatal(err)
	}
	if _, done := f.Drive(est.End.Sub(f.Start) + 15*time.Minute); !done {
		f.Dump()
		t.Fatalf("the send didn't end within its estimate (%s) and 15 minutes", est.End.Sub(f.Start))
	}
	var last time.Time
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT percentile_disc(0.99) WITHIN GROUP (ORDER BY sent_at) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`,
		f.Campaign).Scan(&last); err != nil {
		t.Fatal(err)
	}
	took, want := last.Sub(f.Start), est.End.Sub(f.Start)
	tolerance := max(want/20, 20*time.Second)
	if d := took - want; d < -tolerance || d > tolerance {
		for _, p := range est.Providers {
			t.Logf("route %s: %d at %.1f/s (%s), %d days, done %s", p.ProviderID, p.Recipients, p.Rate, p.RateFrom, p.Days, p.Done.Sub(f.Start))
		}
		if est.AB != nil {
			t.Logf("A/B: sample %d until %s, decided %v", est.AB.Sample, est.AB.SampleEnd.Sub(f.Start), est.AB.DecideAt)
		}
		t.Fatalf("the send took %s, the estimate said %s", took, want)
	}
}
