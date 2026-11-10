package reports

import (
	"fmt"
	"testing"

	"github.com/hostrepute/reputecast/internal/store"
)

func TestFillBreakdown(t *testing.T) {
	rows := []store.ReportDimsRow{
		{Dim: "client", Key: "other", OpenedHuman: 50, Clicked: 1},
		{Dim: "client", Key: "outlook", OpenedHuman: 20, Clicked: 4},
		{Dim: "client", Key: "apple_mail", OpenedHuman: 30, Clicked: 2},
		{Dim: "device", Key: "unknown", OpenedHuman: 60},
		{Dim: "device", Key: "mobile", OpenedHuman: 40},
		{Dim: "hour", Key: "9", OpenedHuman: 7, Clicked: 2},
		{Dim: "hour", Key: "24", OpenedHuman: 1},
		{Dim: "weekday", Key: "7", OpenedHuman: 3},
		{Dim: "city", Key: "VN|Hanoi", OpenedHuman: 12, Clicked: 3},
		{Dim: "city", Key: "VN|Hue", OpenedHuman: 4, Clicked: 1},
		{Dim: "city", Key: "broken", OpenedHuman: 9},
	}
	for i := range 25 {
		rows = append(rows, store.ReportDimsRow{Dim: "country", Key: fmt.Sprintf("C%02d", i), OpenedHuman: int64(100 - i)})
	}
	b := emptyBreakdown()
	fillBreakdown(&b, rows)

	// Ranked by human opens, what can't be told apart last, product names.
	if len(b.Clients) != 3 || b.Clients[0].Label != "Apple Mail" || b.Clients[1].Key != "outlook" || b.Clients[2].Key != "other" || b.Clients[0].Share != 30 {
		t.Fatalf("clients: %+v", b.Clients)
	}
	if b.Devices[0].Key != "mobile" || b.Devices[1].Key != "unknown" || b.Devices[0].Label != "Mobile" {
		t.Fatalf("devices: %+v", b.Devices)
	}
	// The top 20 countries, then the other 5 as one row.
	if len(b.Countries) != TopPlaces+1 || b.Countries[TopPlaces].Key != "other" || b.Countries[TopPlaces].Places != 5 || b.Countries[TopPlaces].HumanOpens != 80+79+78+77+76 {
		t.Fatalf("countries: %+v", b.Countries[len(b.Countries)-1])
	}
	if b.Located != 2200 {
		t.Fatalf("located %d", b.Located)
	}
	// A city under five opens folds; a key without a country is dropped.
	if len(b.Cities) != 2 || b.Cities[0].Label != "Hanoi" || b.Cities[0].Country != "VN" || b.Cities[1].Key != "other" || b.Cities[1].Places != 1 || b.Cities[1].HumanOpens != 4 {
		t.Fatalf("cities: %+v", b.Cities)
	}
	if len(b.Hours) != 24 || b.Hours[9].HumanOpens != 7 || b.Hours[9].Clicks != 2 || b.Weekdays[6].Key != 7 || b.Weekdays[6].HumanOpens != 3 {
		t.Fatalf("times: %+v %+v", b.Hours[9], b.Weekdays[6])
	}

	// A read-only link: no cities, countries under five folded into the
	// existing "other" row, shares of the same total.
	r := CampaignReport{Breakdown: ReportBreakdown{Cities: b.Cities, Countries: []ReportDimRow{
		{Key: "US", HumanOpens: 90}, {Key: "IS", HumanOpens: 4}, {Key: "LU", HumanOpens: 1}, {Key: "other", HumanOpens: 5, Places: 3},
	}}}
	redact(&r)
	c := r.Breakdown.Countries
	if len(r.Breakdown.Cities) != 0 || r.Breakdown.Cities == nil || len(c) != 2 || c[1].Key != "other" || c[1].Places != 5 || c[1].HumanOpens != 10 || c[0].Share != 90 || c[1].Share != 10 {
		t.Fatalf("redacted: %+v %+v", r.Breakdown.Cities, c)
	}
}
