package reports_test

import (
	"encoding/json"
	"fmt"
	"strings"
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
	t.Run("ContentClickMaps", func(t *testing.T) { testABContentMaps(t, s.Fixture(t, "AB content maps")) })
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

// clicks has a share of a variant's sample click its first link (people,
// ten minutes after their mail went out).
func clicks(t *testing.T, f *sendtest.Fixture, variant string, every int) int {
	t.Helper()
	return clicksOn(t, f, variant, 0, every)
}

// clicksOn has a share of a variant's sample click a link (people, ten
// minutes after their mail went out).
func clicksOn(t *testing.T, f *sendtest.Fixture, variant string, link, every int) int {
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
			evs = append(evs, sending.Engagement{MessageID: id, Click: true, Link: link, UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/129.0",
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
	// A fifth of each variant's sample clicks, and nobody else: with the
	// demo transport's clicks (about 4% of each, at random), one variant
	// would lead significantly about one run in twenty.
	f.DriveWithout(5*time.Minute, "click")
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND variant <> '' AND NOT ab_rollout AND sent_at IS NULL`, f.Campaign); n != 0 {
		f.Dump()
		t.Fatalf("%d of the sample not sent after 5 minutes", n)
	}
	clicks(t, f, "A", 5)
	clicks(t, f, "B", 5)
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

// testABContentMaps: a test on content has an email per variant, so there is
// no one map of everyone's clicks (review AN-5). The report marks its click
// map per variant; each variant's map is its own email with its own links,
// numbered from its first, and counts its own recipients' clicks; the
// campaign's map is variant A's and says nothing of B's links; the bot
// clicks the headline leaves out are every variant's.
func testABContentMaps(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = $2 WHERE id = $1`, f.Campaign, `{"kind":"content","variants":[{"key":"A"},
		{"key":"B","html":"<html><body><h1>Variant B</h1><p><a href=\"https://acme.io/b-guide\">Read the B guide</a></p><p><a href=\"https://acme.io/b-blog\">B blog</a></p><p>Acme Inc · 500 Market St, San Francisco, CA 94105 · <a href=\"{{ unsubscribe_url }}\">Unsubscribe</a></p></body></html>"}],
		"test_percent":40,"metric":"clicks","duration_hours":1,"min_sample":150}`)
	f.AddContacts(sendtest.ContactSpec{N: 1600, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{ABWindow: 40 * time.Minute})
	f.Drive(5 * time.Minute)
	var offB int
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT (content->'links'->>'B')::int FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&offB); err != nil || offB <= 0 {
		t.Fatalf("variant B's first link position: %d %v", offB, err)
	}
	onA := clicksOn(t, f, "A", 0, 5)
	onB := clicksOn(t, f, "B", offB, 3)
	// A scanner on one of B's links: a bot click on B's email only.
	var scanned uuid.UUID
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT id FROM messages WHERE campaign_id = $1 AND variant = 'B' AND NOT ab_rollout AND sent_at IS NOT NULL
		ORDER BY id DESC LIMIT 1`, f.Campaign).Scan(&scanned); err != nil {
		t.Fatal(err)
	}
	if err := f.E.Recorder.RecordEngagement(f.Ctx, []sending.Engagement{{MessageID: scanned, Click: true, Link: offB + 1, UserAgent: "Mozilla/5.0 Barracuda Sentinel",
		IP: "198.51.100.250", Method: "GET", At: f.Start.Add(20 * time.Minute)}}); err != nil {
		t.Fatal(err)
	}
	if _, done := f.Drive(3 * time.Hour); !done {
		f.Dump()
		t.Fatal("the send didn't finish")
	}
	type mapLink struct {
		URL          string
		Position     int
		UniqueClicks int64
		BotClicks    int64
	}
	var r struct {
		Headline struct{ BotClicks int64 }
		ClickMap struct {
			PerVariant bool
			Total      int64
			BotTotal   int64
			Links      []mapLink
			Variants   []struct {
				Key       string
				SameEmail bool
				HTML      string
				Total     int64
				BotTotal  int64
				Links     []mapLink
			}
		}
		Insights []struct{ Kind, Variant string }
	}
	c := f.Env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("GET", fmt.Sprintf("/workspaces/acme-newsletter/campaigns/%s/report", f.Campaign), nil).MustStatus(t, 200).JSON(t, &r)
	m := r.ClickMap
	if !m.PerVariant || len(m.Variants) != 2 || m.Variants[0].Key != "A" || m.Variants[1].Key != "B" {
		t.Fatalf("a test on content has a map per variant: per variant %v, %d maps", m.PerVariant, len(m.Variants))
	}
	a, b := m.Variants[0], m.Variants[1]
	if !a.SameEmail || b.SameEmail || !strings.Contains(b.HTML, "b-guide") || !strings.Contains(b.HTML, `data-rc-link="0"`) {
		t.Fatalf("B's map is drawn on B's own email: same email %v, html %q", b.SameEmail, b.HTML)
	}
	if len(b.Links) != 2 || b.Links[0].Position != 0 || b.Links[0].URL != "https://acme.io/b-guide" || b.Links[1].URL != "https://acme.io/b-blog" {
		t.Fatalf("B's links, numbered from its first: %+v", b.Links)
	}
	// What each variant's recipients did, from the clicks recorded one by
	// one (the demo transport's readers click too): the people who clicked
	// each link, and the bot clicks.
	people := func(variant string, from, to int) int64 {
		return f.Count(`SELECT count(*) FROM tracking_link_clicks k JOIN messages m ON m.id = k.message_id
			WHERE m.campaign_id = $1 AND m.variant = $2 AND k.link BETWEEN $3 AND $4`, f.Campaign, variant, from, to)
	}
	bots := func(variant string, from, to int) int64 {
		return f.Count(`SELECT count(*) FROM delivery_events e JOIN messages m ON m.id = e.message_id
			WHERE e.campaign_id = $1 AND e.type = 'click' AND e.machine AND m.variant = $2 AND e.link BETWEEN $3 AND $4`, f.Campaign, variant, from, to)
	}
	if want := people("B", offB, offB); b.Links[0].UniqueClicks != want || want < int64(onB) {
		t.Fatalf("B's guide: %d people on B's map, %d clicked it (%d of them here)", b.Links[0].UniqueClicks, want, onB)
	}
	if want := people("B", offB, offB+1); b.Total != want {
		t.Fatalf("B's map counts %d people's clicks, B's recipients made %d", b.Total, want)
	}
	if want := bots("B", offB, offB+1); b.BotTotal != want || b.Links[1].BotClicks < 1 {
		t.Fatalf("B's map: %d bot clicks (%d on the blog link), want %d with the scanner's", b.BotTotal, b.Links[1].BotClicks, want)
	}
	if want := people("A", 0, offB-1); a.Total != want || want < int64(onA) {
		t.Fatalf("A's map counts %d people's clicks, A's recipients made %d (%d of them here)", a.Total, want, onA)
	}
	if want := bots("A", 0, offB-1); a.BotTotal != want {
		t.Fatalf("A's map: %d bot clicks, want %d", a.BotTotal, want)
	}
	// The campaign's map is A's email, with A's links only. (Its counts are
	// every click on those links: the demo transport's readers click by
	// position, whichever email they got, so they can be more than A's.)
	if m.Total < a.Total {
		t.Fatalf("the campaign's map counts %d clicks, A's map %d", m.Total, a.Total)
	}
	for _, l := range m.Links {
		if strings.Contains(l.URL, "acme.io/b-") {
			t.Fatalf("the campaign's map has B's link %s", l.URL)
		}
	}
	// The tile's bot clicks are every variant's, not only the map's (A's):
	// the scanner's click on B's email is one of them.
	want := f.Count(`SELECT count(*) FROM delivery_events WHERE campaign_id = $1 AND type = 'click' AND machine AND link >= 0`, f.Campaign)
	if r.Headline.BotClicks != want || r.Headline.BotClicks < m.BotTotal+1 {
		t.Fatalf("headline bot clicks %d, want every link's %d (the campaign's map has %d)", r.Headline.BotClicks, want, m.BotTotal)
	}
	for _, in := range r.Insights {
		if in.Kind == "top_link" && in.Variant == "" {
			t.Fatalf("the top link of a test on content names its variant: %+v", in)
		}
	}
}
