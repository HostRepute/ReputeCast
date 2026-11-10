package reports

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/counters"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// GeoIP says whether the server has a GeoIP database loaded (geoip.DB):
// without one, no message gets a country or a city.
type GeoIP interface{ Loaded() bool }

// TopPlaces is how many countries and cities the report lists before it
// folds the rest into "other".
const TopPlaces = 20

// MinPlace is the fewest human opens a city needs to be named (and a
// country, on a read-only link): fewer fold into "other", so a small
// place can't point at a few readers.
const MinPlace = 5

// deviceNames are the English words of devices (tracking.ClientParts
// keys), which the CSV uses and the app translates.
var deviceNames = map[string]string{"mobile": "Mobile", "tablet": "Tablet", "desktop": "Desktop"}

const (
	otherCountries = "Other countries"
	otherCities    = "Other cities"
)

// emptyBreakdown has every list, so the API never sends a null.
func emptyBreakdown() ReportBreakdown {
	return ReportBreakdown{Clients: []ReportDimRow{}, Systems: []ReportDimRow{}, Devices: []ReportDimRow{}, Countries: []ReportDimRow{},
		Cities: []ReportDimRow{}, Hours: timeBuckets(0, 24), Weekdays: timeBuckets(1, 7)}
}

func timeBuckets(from, n int) []ReportTimeBucket {
	out := make([]ReportTimeBucket, n)
	for i := range out {
		out[i].Key = from + i
	}
	return out
}

// breakdown reads the campaign's mail clients, devices, places and times.
// Only campaigns whose sends recorded them (campaigns.breakdown) are
// measured; imported history never is.
func (s *Service) breakdown(ctx context.Context, ws *store.Workspace, c store.ReportCampaignRow, privacy int64) (ReportBreakdown, error) {
	b := emptyBreakdown()
	b.Timezone = provstate.Location(ws.Timezone).String()
	b.GeoIP = s.Geo != nil && s.Geo.Loaded()
	b.PrivacyOpens = privacy
	b.Measured = c.Breakdown && c.Origin == ""
	if !b.Measured {
		return b, nil
	}
	rows, err := s.Q.ReportDims(ctx, c.ID)
	if err != nil {
		return b, err
	}
	fillBreakdown(&b, rows)
	return b, nil
}

// fillBreakdown sorts campaign_dims rows into the breakdown's lists.
func fillBreakdown(b *ReportBreakdown, rows []store.ReportDimsRow) {
	var clients, systems, devices, countries, cities []ReportDimRow
	for _, r := range rows {
		row := ReportDimRow{Key: r.Key, Label: dimLabel(r.Dim, r.Key), HumanOpens: r.OpenedHuman, Clicks: r.Clicked}
		switch r.Dim {
		case counters.DimClient:
			clients = append(clients, row)
		case counters.DimOS:
			systems = append(systems, row)
		case counters.DimDevice:
			devices = append(devices, row)
		case counters.DimCountry:
			b.Located += r.OpenedHuman
			countries = append(countries, row)
		case counters.DimCity:
			cc, city, ok := strings.Cut(r.Key, "|")
			if !ok || city == "" {
				continue
			}
			row.Country, row.Label = cc, city
			cities = append(cities, row)
		case counters.DimHour:
			if h, err := strconv.Atoi(r.Key); err == nil && h >= 0 && h < 24 {
				b.Hours[h].HumanOpens += r.OpenedHuman
				b.Hours[h].Clicks += r.Clicked
			}
		case counters.DimWeekday:
			if d, err := strconv.Atoi(r.Key); err == nil && d >= 1 && d <= 7 {
				b.Weekdays[d-1].HumanOpens += r.OpenedHuman
				b.Weekdays[d-1].Clicks += r.Clicked
			}
		}
	}
	b.Clients, b.Systems, b.Devices = ranked(clients), ranked(systems), ranked(devices)
	b.Countries = fold(ranked(countries), otherCountries, func(kept int, _ ReportDimRow) bool { return kept < TopPlaces })
	b.Cities = fold(ranked(cities), otherCities, func(kept int, r ReportDimRow) bool { return kept < TopPlaces && r.HumanOpens >= MinPlace })
}

// dimLabel names a key: tracking's product names for mail apps and
// systems (never translated), the English words of devices and of the
// rest, else the key itself.
func dimLabel(dim, key string) string {
	var name string
	switch dim {
	case counters.DimClient:
		name = tracking.AppName(key)
	case counters.DimOS:
		name = tracking.OSName(key)
	case counters.DimDevice:
		name = deviceNames[key]
	}
	switch {
	case name != "":
		return name
	case key == "other":
		return "Other"
	case key == "unknown":
		return "Unknown"
	}
	return key
}

// rest are the keys that sort last: what couldn't be told apart.
func rest(key string) bool { return key == "other" || key == "unknown" }

// ranked orders rows by human opens, then clicks, with the rest last, and
// gives each its share of the dimension's human opens. Never nil.
func ranked(rows []ReportDimRow) []ReportDimRow {
	if rows == nil {
		return []ReportDimRow{}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if rest(a.Key) != rest(b.Key) {
			return !rest(a.Key)
		}
		if a.HumanOpens != b.HumanOpens {
			return a.HumanOpens > b.HumanOpens
		}
		if a.Clicks != b.Clicks {
			return a.Clicks > b.Clicks
		}
		return a.Key < b.Key
	})
	shares(rows)
	return rows
}

// shares sets each row's part of the rows' human opens.
func shares(rows []ReportDimRow) {
	var total int64
	for _, r := range rows {
		total += r.HumanOpens
	}
	for i := range rows {
		rows[i].Share = 0
		if p := rate(rows[i].HumanOpens, total); p != nil {
			rows[i].Share = *p
		}
	}
}

// fold keeps the ranked rows keep accepts (told how many are kept so far)
// and folds the others, with an "other" row already there, into one
// "other" row at the end that counts the places it holds. Shares stay of
// every row's human opens. Never nil.
func fold(rows []ReportDimRow, label string, keep func(kept int, r ReportDimRow) bool) []ReportDimRow {
	out := make([]ReportDimRow, 0, len(rows)+1)
	other := ReportDimRow{Key: "other", Label: label}
	for _, r := range rows {
		switch {
		case r.Key == "other":
			other.HumanOpens += r.HumanOpens
			other.Clicks += r.Clicks
			other.Places += r.Places
		case keep(len(out), r):
			out = append(out, r)
		default:
			other.HumanOpens += r.HumanOpens
			other.Clicks += r.Clicks
			other.Places++
		}
	}
	if other.Places > 0 {
		out = append(out, other)
	}
	shares(out)
	return out
}
