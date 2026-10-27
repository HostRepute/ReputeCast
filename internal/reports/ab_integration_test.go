package reports_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// TestABEndToEnd sends A/B tests through the real engine and the demo
// transport (deliveries through webhook normalization, opens and clicks
// through the tracking recorder) and reads the result from the report: the
// decision the engine kept, the samples apart from the recipients who got
// the variant after the test (review AN-1).
func TestABEndToEnd(t *testing.T) {
	s := sendtest.NewSuite(t)
	t.Run("ClearWinner", func(t *testing.T) { testABWinner(t, s.Fixture(t, "AB clear winner")) })
	t.Run("NoClearWinner", func(t *testing.T) { testABNoWinner(t, s.Fixture(t, "AB no clear winner")) })
	t.Run("SendTimeInLocalTime", func(t *testing.T) { testABSendTime(t, s.Fixture(t, "AB send time")) })
}

type abCard struct {
	State, Outcome, Winner, Default, Metric string
	Remaining                               int64
	MinSample                               int64
	Threshold                               float64
	Confidence                              *float64
	Significant                             bool
	DecidedAfter                            *float64
	Variants                                []struct {
		Key                   string
		Default, Winner       bool
		Recipients, Remaining int64
		Eligible, Conversions int64
		DecisionRate          *float64
		MetricRate            *float64
	}
}

func abReport(t *testing.T, f *sendtest.Fixture) abCard {
	t.Helper()
	c := f.Env.Client()
	c.SignInDemo("linh@acme.io")
	var r struct {
		ABTest *abCard
	}
	c.Do("GET", fmt.Sprintf("/workspaces/acme-newsletter/campaigns/%s/report", f.Campaign), nil).MustStatus(t, 200).JSON(t, &r)
	if r.ABTest == nil {
		t.Fatal("the report has no A/B card")
	}
	return *r.ABTest
}

func setAB(f *sendtest.Fixture, minSample int, metric string) {
	f.Exec(`UPDATE campaigns SET ab_test = $2 WHERE id = $1`, f.Campaign, fmt.Sprintf(`{"kind":"subject","variants":[{"key":"A","subject":"Three launches this week"},
		{"key":"B","subject":"The Postgres deep dive, plus three launches"}],"test_percent":40,"metric":%q,"duration_hours":1,"min_sample":%d}`, metric, minSample))
}

// clicks has a share of a variant's sample click (people, an hour after
// their mail went out).
func clicks(t *testing.T, f *sendtest.Fixture, variant string, every int) int {
	t.Helper()
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT id, sent_at FROM messages WHERE campaign_id = $1 AND variant = $2 AND NOT ab_rollout AND sent_at IS NOT NULL ORDER BY id`,
		f.Campaign, variant)
	if err != nil {
		t.Fatal(err)
	}
	var evs []sending.Engagement
	i := 0
	for rows.Next() {
		var id uuid.UUID
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			t.Fatal(err)
		}
		if i%every == 0 {
			evs = append(evs, sending.Engagement{MessageID: id, Click: true, Link: 0, UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/129.0",
				IP: fmt.Sprintf("198.51.100.%d", i%200+1), Method: "GET", At: at.Add(10 * time.Minute)})
		}
		i++
	}
	rows.Close()
	if err := f.E.Recorder.RecordEngagement(f.Ctx, evs); err != nil {
		t.Fatal(err)
	}
	return len(evs)
}

func testABWinner(t *testing.T, f *sendtest.Fixture) {
	setAB(f, 150, "clicks")
	f.AddContacts(sendtest.ContactSpec{N: 1600, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{ABWindow: 40 * time.Minute})
	f.Drive(5 * time.Minute)
	held := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held'`, f.Campaign)
	sample := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND variant <> '' AND sent_at IS NOT NULL`, f.Campaign)
	if held == 0 || sample == 0 {
		f.Dump()
		t.Fatalf("%d held, %d sample sent", held, sample)
	}
	// While the test runs, the card says so.
	if card := abReport(t, f); card.State != "testing" || card.Winner != "" {
		t.Fatalf("during the test: %+v", card)
	}
	clicked := clicks(t, f, "B", 4) // a quarter of B's sample clicks
	if _, done := f.Drive(3 * time.Hour); !done {
		f.Dump()
		t.Fatal("the send didn't finish")
	}
	var raw []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT ab_test->'decision' FROM campaigns WHERE id = $1`, f.Campaign).Scan(&raw); err != nil || len(raw) == 0 {
		t.Fatalf("no decision kept: %v", err)
	}
	var d struct {
		Outcome, Winner string
		Confidence      float64
		MinSample       int64 `json:"min_sample"`
	}
	_ = json.Unmarshal(raw, &d)
	if d.Outcome != "winner" || d.Winner != "B" || d.Confidence < 95 || d.MinSample != 150 {
		t.Fatalf("decision %s", raw)
	}
	// The rest got B, counted apart from the sample.
	rollout := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND ab_rollout AND variant = 'B'`, f.Campaign)
	if rollout != held {
		t.Fatalf("%d of %d held recipients got B after the test", rollout, held)
	}
	if n := f.Count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1 AND variant = ''`, f.Campaign); n != rollout {
		t.Fatalf("the rollout's cells count %d sent, want %d", n, rollout)
	}
	card := abReport(t, f)
	if card.State != "decided" || card.Winner != "B" || card.Outcome != "winner" || !card.Significant || card.Remaining != rollout {
		t.Fatalf("card %+v, want B decided with %d remaining", card, rollout)
	}
	if card.MinSample != 150 || card.Threshold != 95 || card.Confidence == nil || *card.Confidence < 95 || card.DecidedAfter == nil {
		t.Fatalf("card's rule and statistics: %+v", card)
	}
	var sampled int64
	for _, v := range card.Variants {
		sampled += v.Recipients
		if v.Key == "B" && (!v.Winner || v.Remaining != rollout) {
			t.Fatalf("variant B: %+v", v)
		}
		if v.Key == "A" && (!v.Default || v.Winner) {
			t.Fatalf("variant A: %+v", v)
		}
		if v.DecisionRate == nil || v.Eligible < 150 {
			t.Fatalf("variant %s has no decision numbers: %+v", v.Key, v)
		}
	}
	if sampled != sample {
		t.Fatalf("the card's samples add up to %d recipients, want %d (the rollout apart)", sampled, sample)
	}
	// Each variant has its click map (review AN-5): the same email (a
	// subject test), each variant's recipients' clicks.
	var r struct {
		ClickMap struct {
			Variants []struct {
				Key       string
				SameEmail bool
				Links     []struct {
					Position     int
					Clicks       int64
					UniqueClicks int64
				}
			}
		}
	}
	c := f.Env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("GET", fmt.Sprintf("/workspaces/acme-newsletter/campaigns/%s/report", f.Campaign), nil).MustStatus(t, 200).JSON(t, &r)
	if len(r.ClickMap.Variants) != 2 || r.ClickMap.Variants[0].Key != "A" || r.ClickMap.Variants[1].Key != "B" || !r.ClickMap.Variants[1].SameEmail {
		t.Fatalf("variant click maps: %+v", r.ClickMap.Variants)
	}
	var onB int64
	for _, l := range r.ClickMap.Variants[1].Links {
		if l.Position == 0 {
			onB = l.UniqueClicks
		}
	}
	if onB < int64(clicked) {
		t.Fatalf("B's map has %d people on the first link, want at least the %d who clicked it", onB, clicked)
	}
}

func testABNoWinner(t *testing.T, f *sendtest.Fixture) {
	setAB(f, 150, "clicks")
	f.AddContacts(sendtest.ContactSpec{N: 1600, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{ABWindow: 40 * time.Minute})
	held := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held'`, f.Campaign)
	if _, done := f.Drive(3 * time.Hour); !done {
		f.Dump()
		t.Fatal("the send didn't finish")
	}
	card := abReport(t, f)
	if card.State != "no_winner" || card.Winner != "A" || card.Default != "A" || card.Significant {
		t.Fatalf("without a clear winner the default goes out: %+v", card)
	}
	if card.Outcome != "not_significant" && card.Outcome != "too_small" {
		t.Fatalf("outcome %q", card.Outcome)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND ab_rollout AND variant = 'A'`, f.Campaign); n != held || card.Remaining != held {
		t.Fatalf("%d got A after the test, the card says %d; want %d", n, card.Remaining, held)
	}
	// The audit entry says what the rule decided.
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE action = 'campaign.ab_winner_picked' AND target_id = $1 AND after->>'winner' = 'A'
		AND after->>'outcome' IN ('not_significant', 'too_small')`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d audit entries for the decision", n)
	}
}

// testABSendTime: a send-time test's variants go out at their time of day
// in each recipient's own time zone, the test is decided only after the
// last of them plus its window, and the rest get the winner's time in
// their time zone.
func testABSendTime(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = '{"kind":"send_time","variants":[{"key":"A","send_time":"09:00"},{"key":"B","send_time":"15:00"}],
		"test_percent":50,"metric":"clicks","duration_hours":4}' WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 40, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.Exec(`UPDATE contacts SET timezone = CASE WHEN right(email, 1) IN ('0', '2', '4') OR email LIKE '%0@%' THEN 'Asia/Ho_Chi_Minh' ELSE 'America/Los_Angeles' END
		WHERE id IN (SELECT contact_id FROM list_memberships WHERE list_id = $1)`, f.List)
	f.StartSend(engine.StartOptions{})
	next := func(hhmm, tz string) time.Time {
		loc, _ := time.LoadLocation(tz)
		var h, m int
		_, _ = fmt.Sscanf(hhmm, "%d:%d", &h, &m)
		l := f.Start.In(loc)
		at := time.Date(l.Year(), l.Month(), l.Day(), h, m, 0, 0, loc)
		if !at.After(f.Start) {
			at = time.Date(l.Year(), l.Month(), l.Day()+1, h, m, 0, 0, loc)
		}
		return at
	}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT m.variant, c.timezone, m.next_attempt_at FROM messages m JOIN contacts c ON c.id = m.contact_id
		WHERE m.campaign_id = $1 AND m.variant <> '' AND NOT m.ab_rollout`, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	var last time.Time
	n := 0
	for rows.Next() {
		var variant, tz string
		var at *time.Time
		if err := rows.Scan(&variant, &tz, &at); err != nil {
			t.Fatal(err)
		}
		want := next(map[string]string{"A": "09:00", "B": "15:00"}[variant], tz)
		if at == nil || !at.Equal(want) {
			t.Fatalf("variant %s in %s goes out at %v, want %v", variant, tz, at, want)
		}
		if want.After(last) {
			last = want
		}
		n++
	}
	rows.Close()
	if n == 0 {
		t.Fatal("no sample")
	}
	var ab struct {
		DecideAt *time.Time `json:"decide_at"`
	}
	var raw []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT ab FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(raw, &ab)
	if ab.DecideAt == nil || !ab.DecideAt.Equal(last.Add(4*time.Hour)) {
		t.Fatalf("decided at %v, want 4 h after the last variant time %v", ab.DecideAt, last)
	}
	// A person picks B: the rest get B at 15:00 their time.
	if err := f.E.PickWinner(f.Ctx, f.Campaign, f.Sofia, "B"); err != nil {
		t.Fatal(err)
	}
	rows, err = f.Env.App.Pool.Query(f.Ctx, `SELECT c.timezone, m.next_attempt_at FROM messages m JOIN contacts c ON c.id = m.contact_id
		WHERE m.campaign_id = $1 AND m.ab_rollout`, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	n = 0
	for rows.Next() {
		var tz string
		var at *time.Time
		if err := rows.Scan(&tz, &at); err != nil {
			t.Fatal(err)
		}
		if want := next("15:00", tz); at == nil || !at.Equal(want) {
			t.Fatalf("the rest in %s get B at %v, want %v", tz, at, want)
		}
		n++
	}
	rows.Close()
	if n == 0 {
		t.Fatal("nobody waited for the test")
	}
	if card := abReport(t, f); card.State != "decided" || card.Winner != "B" || card.Outcome != "manual" {
		t.Fatalf("a person's pick: %+v", card)
	}
}
