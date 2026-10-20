package agency

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

func f(v float64) *float64 { return &v }

// The Usage board's September, row by row.
func TestRebillMatchesTheBoard(t *testing.T) {
	cases := []struct {
		name             string
		rate             AgencyRate
		costMicros       int64
		emails, segments int64
		cost, toRebill   float64
	}{
		{"Contoso", AgencyRate{CostPlusPct: f(20), MonthlyFee: f(150)}, 84621000, 846210, 0, 84.62, 251.54},
		{"Fabrikam", AgencyRate{CostPlusPct: f(15), MonthlyFee: f(300)}, 221488000 + 412100000, 2214880, 38412, 633.59, 1028.63},
		{"Tailspin", AgencyRate{Per1000Emails: f(0.25), MonthlyFee: f(100)}, 40211800, 402118, 0, 40.21, 200.53},
		{"Alpine", AgencyRate{CostPlusPct: f(20), MonthlyFee: f(100)}, 9644000 + 46310000, 96440, 4210, 55.95, 167.14},
		{"Coho", AgencyRate{Flat: f(120)}, 15000000, 41320, 0, 15, 120},
		{"Litware", AgencyRate{CostPlusPct: f(25), MonthlyFee: f(80)}, 35000000, 33960, 0, 35, 123.75},
		// Fabrikam's August, on the rate before (the client details' history).
		{"Fabrikam Aug", AgencyRate{CostPlusPct: f(15), MonthlyFee: f(300)}, 557565100, 0, 0, 557.57, 941.20},
		{"Fabrikam May", AgencyRate{CostPlusPct: f(10), MonthlyFee: f(250)}, 411070000, 0, 0, 411.07, 702.18},
		{"per segment", AgencyRate{PerSegment: f(0.0125)}, 0, 0, 38412, 0, 480.15},
	}
	for _, c := range cases {
		r := rebill(c.rate, c.costMicros, c.emails, c.segments)
		if r.Cost != c.cost || r.ToRebill == nil || *r.ToRebill != c.toRebill {
			t.Errorf("%s: cost %v to rebill %v, want %v and %v", c.name, r.Cost, r.ToRebill, c.cost, c.toRebill)
		}
	}
	// No rate: not billed, the cost still shows.
	if r := rebill(AgencyRate{}, 682000, 6820, 0); r.ToRebill != nil || r.Cost != 0.68 {
		t.Fatalf("not billed: %+v", r)
	}
	// The parts always add up to the total.
	r := rebill(AgencyRate{CostPlusPct: f(15), MonthlyFee: f(300)}, 633588000, 0, 0)
	if r.Cost+r.CostPlus+r.Monthly != *r.ToRebill || r.CostPlus != 95.04 {
		t.Fatalf("parts: %+v", r)
	}
}

func TestCentsRoundsHalvesUp(t *testing.T) {
	for in, want := range map[float64]float64{18.805: 18.81, 0.339: 0.34, 2.5: 2.5, 1.005: 1.01, 95.0382: 95.04, 0.004999: 0} {
		if got := cents(in); got != want {
			t.Errorf("cents(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestRateForAMonth(t *testing.T) {
	ws := uuid.New()
	rates := []store.AgencyRatesRow{
		{WorkspaceID: ws, EffectiveFrom: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), Rate: json.RawMessage(`{"costPlusPct":10,"monthlyFee":250}`)},
		{WorkspaceID: ws, EffectiveFrom: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Rate: json.RawMessage(`{"costPlusPct":15,"monthlyFee":300}`)},
	}
	if r, _ := rateFor(rates, ws, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)); r == nil || *r.CostPlusPct != 10 {
		t.Fatalf("May: %+v", r)
	}
	if r, since := rateFor(rates, ws, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)); r == nil || *r.MonthlyFee != 300 || since == nil || since.Month() != time.June {
		t.Fatalf("September: %+v since %v", r, since)
	}
	if r, _ := rateFor(rates, ws, time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)); r != nil {
		t.Fatalf("before the first rate: %+v", r)
	}
}

func TestDiffPairsChangesAboutTheSameThing(t *testing.T) {
	from := []string{"Background: #1E3A8A", "Text: The holidays, wrapped", "Button: Shop now", "Snow overlay: 40%"}
	to := []string{"Background: #1E3A8A", "Text: The holidays, wrapped", "Button: Shop gifts", "Snow overlay: 25%", "Image: Snowflake"}
	got := diffLines(from, to)
	want := []DiffLine{{"del", "Button: Shop now"}, {"add", "Button: Shop gifts"}, {"del", "Snow overlay: 40%"}, {"add", "Snow overlay: 25%"}, {"add", "Image: Snowflake"}}
	if !slices.Equal(got, want) {
		t.Fatalf("diff: %+v", got)
	}
}

func TestSummaryLinesOfMJML(t *testing.T) {
	src := `<mj-section background-color="#1E3A8A"><mj-column><mj-text>The holidays, wrapped</mj-text><mj-image src="x.png" alt="Gift box"/>` +
		`<mj-button href="#">Shop gifts</mj-button></mj-column></mj-section>`
	got := summaryLines(KindSection, LibraryContent{MJML: src})
	want := []string{"Background: #1E3A8A", "Text: The holidays, wrapped", "Image: Gift box", "Button: Shop gifts"}
	if !slices.Equal(got, want) {
		t.Fatalf("summary: %v", got)
	}
	if got := summaryLines(KindSMS, LibraryContent{Body: "Flash sale today"}); !slices.Equal(got, []string{"Text: Flash sale today"}) {
		t.Fatalf("sms: %v", got)
	}
}

func TestAgencyTone(t *testing.T) {
	for _, c := range []struct {
		state string
		score int
		want  string
	}{{"ready", 99, "ok"}, {"ready", 80, "ok"}, {"ready", 71, "warn"}, {"ready", 65, "warn"}, {"ready", 62, "bad"}, {"pending", 0, "neutral"}} {
		if got := agencyTone(c.state, c.score); got != c.want {
			t.Errorf("%s %d: %s, want %s", c.state, c.score, got, c.want)
		}
	}
}

func TestRenamedName(t *testing.T) {
	taken := map[string]bool{"Welcome series (template)": true}
	got, err := renamedName("Welcome series", func(n string) (bool, error) { return taken[n], nil })
	if err != nil || got != "Welcome series (template 2)" {
		t.Fatalf("renamed: %q %v", got, err)
	}
	got, _ = renamedName("Win-back", func(n string) (bool, error) { return taken[n], nil })
	if got != "Win-back (template)" {
		t.Fatalf("renamed: %q", got)
	}
}
