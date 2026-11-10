package reports_test

import (
	"math"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type breakdownRow struct {
	Key, Label, Country string
	Places              int
	HumanOpens, Clicks  int64
	Share               float64
}

type breakdownReport struct {
	Headline  struct{ HumanOpens, Clicks, PrivacyOpens int64 }
	Campaign  struct{ Origin string }
	Breakdown struct {
		Measured, GeoIP                              bool
		Clients, Systems, Devices, Countries, Cities []breakdownRow
		Located, PrivacyOpens                        int64
		Hours, Weekdays                              []struct {
			Key                int
			HumanOpens, Clicks int64
		}
		Timezone string
	}
}

func sumShares(rows []breakdownRow) float64 {
	var s float64
	for _, r := range rows {
		s += r.Share
	}
	return s
}

// The demo's October Product Update recorded where and when people read it
// (Report board, M57 G-12): every dimension but the places adds up to the
// headline's human opens, read-only links show no cities and fold small
// countries, and the CSV carries the sections.
func TestReportBreakdown(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	oct := campaignID(t, c, "October Product Update")
	base := "/workspaces/acme-newsletter/campaigns/" + oct + "/report"

	var r breakdownReport
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &r)
	b := r.Breakdown
	if !b.Measured || b.GeoIP || b.Timezone == "" || b.PrivacyOpens != r.Headline.PrivacyOpens {
		t.Fatalf("breakdown: measured %v, geoip %v, timezone %q, privacy %d", b.Measured, b.GeoIP, b.Timezone, b.PrivacyOpens)
	}
	// The board's first mail client, and unknown systems last.
	if len(b.Clients) == 0 || b.Clients[0].Key != "gmail" || b.Clients[0].Label != "Gmail" {
		t.Fatalf("clients: %+v", b.Clients)
	}
	if last := b.Systems[len(b.Systems)-1]; last.Key != "unknown" {
		t.Fatalf("systems end with %+v", last)
	}
	for name, rows := range map[string][]breakdownRow{"clients": b.Clients, "systems": b.Systems, "devices": b.Devices, "countries": b.Countries} {
		var opens, clicks int64
		for _, row := range rows {
			opens += row.HumanOpens
			clicks += row.Clicks
		}
		if s := sumShares(rows); math.Abs(s-100) > 0.1 {
			t.Errorf("%s shares add up to %.2f", name, s)
		}
		if name != "countries" && (opens != r.Headline.HumanOpens || clicks != r.Headline.Clicks) {
			t.Errorf("%s: %d opens, %d clicks; headline %d, %d", name, opens, clicks, r.Headline.HumanOpens, r.Headline.Clicks)
		}
		if name == "countries" && (opens != b.Located || opens >= r.Headline.HumanOpens) {
			t.Errorf("countries: %d opens, located %d", opens, b.Located)
		}
	}
	var hours, days int64
	for _, h := range b.Hours {
		hours += h.HumanOpens
	}
	for _, d := range b.Weekdays {
		days += d.HumanOpens
	}
	if len(b.Hours) != 24 || len(b.Weekdays) != 7 || hours != r.Headline.HumanOpens || days != r.Headline.HumanOpens {
		t.Fatalf("times: %d hours (%d opens), %d weekdays (%d opens)", len(b.Hours), hours, len(b.Weekdays), days)
	}
	// Cities name their country; the small ones are one "other" row.
	if len(b.Cities) == 0 || b.Cities[0].Country == "" || b.Cities[len(b.Cities)-1].Key != "other" || b.Cities[len(b.Cities)-1].Places == 0 {
		t.Fatalf("cities: %+v", b.Cities)
	}
	for _, city := range b.Cities {
		if city.Key != "other" && city.HumanOpens < 5 {
			t.Errorf("city %q with %d opens is named", city.Label, city.HumanOpens)
		}
	}
	small := 0
	for _, country := range b.Countries {
		if country.HumanOpens < 5 {
			small++
		}
	}
	if small == 0 {
		t.Fatal("the demo has no small country to fold on a read-only link")
	}

	// A read-only link: countries only, small ones folded.
	var link struct{ URL string }
	c.Do("POST", base+"/shares", map[string]any{}).MustStatus(t, 201).JSON(t, &link)
	var shared struct{ Report breakdownReport }
	env.Client().Do("GET", "/shared-reports/"+link.URL[strings.LastIndex(link.URL, "/")+1:], nil).MustStatus(t, 200).JSON(t, &shared)
	sb := shared.Report.Breakdown
	if !sb.Measured || len(sb.Cities) != 0 || len(sb.Clients) != len(b.Clients) {
		t.Fatalf("shared breakdown: %d cities, %d clients", len(sb.Cities), len(sb.Clients))
	}
	other := sb.Countries[len(sb.Countries)-1]
	if other.Key != "other" || other.Places != small || len(sb.Countries) != len(b.Countries)-small+1 {
		t.Fatalf("shared countries: %+v", sb.Countries)
	}
	for _, country := range sb.Countries {
		if country.Key != "other" && country.HumanOpens < 5 {
			t.Errorf("shared link names %q with %d opens", country.Key, country.HumanOpens)
		}
	}
	if s := sumShares(sb.Countries); math.Abs(s-100) > 0.1 {
		t.Errorf("shared country shares add up to %.2f", s)
	}

	body := string(c.Do("GET", base+"/export.csv", nil).MustStatus(t, 200).Body)
	for _, want := range []string{"Mail clients,Human opens", "Systems,", "Devices,", "Countries,", "Cities,Country", "Hour of day (", "Day of week (", "Gmail,"} {
		if !strings.Contains(body, want) {
			t.Errorf("CSV lacks %q", want)
		}
	}

	// Campaigns sent before ReputeCast recorded breakdowns have none.
	var old breakdownReport
	c.Do("GET", "/workspaces/acme-newsletter/campaigns/"+campaignID(t, c, "Weekly Digest #141")+"/report", nil).MustStatus(t, 200).JSON(t, &old)
	if old.Breakdown.Measured || old.Breakdown.Clients == nil || len(old.Breakdown.Clients) != 0 || len(old.Breakdown.Hours) != 24 {
		t.Fatalf("an older campaign's breakdown: %+v", old.Breakdown)
	}
}
