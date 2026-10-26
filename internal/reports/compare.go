package reports

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/store"
)

// Every tile compares the issue with the median of the workspace's last
// issues before it (review AN-11): "▲ 2.4% vs your last 6 issues" for the
// count sent, percentage points for the rates. Until the results are final
// nothing is compared: the numbers can still rise.

// CompareIssues is how many earlier issues the medians are of.
const CompareIssues = 6

// ReportComparison is one tile's change against the median of the last
// issues.
type ReportComparison struct {
	Metric string `json:"metric" enum:"sent,delivered,humanOpens,privacyOpens,clicks,unsubscribes,complaints,bounces"`
	// Change is the difference with the median: percent of the median for
	// the count sent, percentage points for the rates.
	Change float64 `json:"change"`
	Unit   string  `json:"unit" enum:"percent,points"`
	Median float64 `json:"median"`
	// Tone is how the change reads: good (more deliveries, opens, clicks;
	// fewer unsubscribes, complaints, bounces), bad, or neutral (the count
	// sent, privacy-protected opens, no change).
	Tone string `json:"tone" enum:"good,bad,neutral"`
}

// ReportComparisons are the tiles' comparisons.
type ReportComparisons struct {
	// Issues is how many earlier issues the medians are of (up to 6).
	Issues int                `json:"issues"`
	Tiles  []ReportComparison `json:"tiles"`
	// Unavailable says why nothing is compared: not_final (results still
	// coming in) or no_history (no earlier issue).
	Unavailable string `json:"unavailable,omitempty" enum:"not_final,no_history"`
}

// issueRates are one issue's rates, as the tiles show them.
type issueRates struct {
	sent                                                         float64
	delivered, humanMeasurable, privacy, clicks, unsubs, bounces *float64
	complaints                                                   *float64
}

func ratesOf(all, reported counts) issueRates {
	return issueRates{sent: float64(all.sent), delivered: rate(all.delivered, all.sent), humanMeasurable: rate(all.human, all.measurable()),
		privacy: rate(all.privacy, all.delivered), clicks: rate(all.clicked, all.delivered), unsubs: rate(all.unsub, all.delivered),
		bounces: rate(all.hard+all.soft, all.sent), complaints: rate(reported.complained, reported.delivered)}
}

// totalsOf sums counter cells: all of them, and those of mailbox providers
// that report complaints.
func totalsOf(cells []store.ReportCounterCellsRow) (all, reported counts) {
	for _, c := range cells {
		all.add(c)
		if mailbox.ReportsComplaints(c.MailboxProvider) {
			reported.add(c)
		}
	}
	return all, reported
}

// compare builds the tiles' comparisons for a campaign whose results are
// final.
func (s *Service) compare(ctx context.Context, ws *store.Workspace, c store.ReportCampaignRow, cells []store.ReportCounterCellsRow, final bool) (ReportComparisons, error) {
	out := ReportComparisons{Tiles: []ReportComparison{}}
	if !final {
		out.Unavailable = "not_final"
		return out, nil
	}
	before := time.Now()
	if c.StartedAt != nil {
		before = *c.StartedAt
	}
	rows, err := s.Q.ReportRecentIssues(ctx, store.ReportRecentIssuesParams{WorkspaceID: ws.ID, CampaignID: c.ID, Before: &before, Lim: CompareIssues})
	if err != nil {
		return out, err
	}
	type issue struct{ all, reported counts }
	byID := map[uuid.UUID]*issue{}
	order := []uuid.UUID{}
	for _, r := range rows {
		is := byID[r.CampaignID]
		if is == nil {
			is = &issue{}
			byID[r.CampaignID] = is
			order = append(order, r.CampaignID)
		}
		cell := store.ReportCounterCellsRow{Sent: r.Sent, Delivered: r.Delivered, HardBounced: r.HardBounced, SoftBounced: r.SoftBounced,
			Complained: r.Complained, OpenedHuman: r.OpenedHuman, OpenedPrivacy: r.OpenedPrivacy, OpenedMachine: r.OpenedMachine, Clicked: r.Clicked,
			Unsubscribed: r.Unsubscribed}
		is.all.add(cell)
		if mailbox.ReportsComplaints(r.MailboxProvider) {
			is.reported.add(cell)
		}
	}
	if len(order) == 0 {
		out.Unavailable = "no_history"
		return out, nil
	}
	out.Issues = len(order)
	past := make([]issueRates, 0, len(order))
	for _, id := range order {
		is := byID[id]
		r := ratesOf(is.all, is.reported)
		if legacyOpens(is.all) {
			r.humanMeasurable, r.privacy = nil, nil
		}
		past = append(past, r)
	}
	out.Tiles = compareTiles(ratesOf(totalsOf(cells)), past)
	return out, nil
}

// compareTiles compares an issue's numbers with the medians of the earlier
// issues': the count sent in percent of its median, the rates in
// percentage points. A rate an issue doesn't have (nothing delivered, no
// mailbox provider that reports complaints) is left out of its median; a
// tile none of them has isn't compared.
func compareTiles(now issueRates, past []issueRates) []ReportComparison {
	out := []ReportComparison{}
	sents := make([]float64, 0, len(past))
	for _, p := range past {
		sents = append(sents, p.sent)
	}
	if m, ok := median(sents); ok && m > 0 {
		out = append(out, ReportComparison{Metric: "sent", Change: round((now.sent-m)/m*100, 1), Unit: "percent", Median: m, Tone: "neutral"})
	}
	// Up is good or bad by metric.
	for _, k := range []struct {
		metric string
		pick   func(issueRates) *float64
		upGood *bool
	}{
		{"delivered", func(r issueRates) *float64 { return r.delivered }, ptrBool(true)},
		{"humanOpens", func(r issueRates) *float64 { return r.humanMeasurable }, ptrBool(true)},
		{"privacyOpens", func(r issueRates) *float64 { return r.privacy }, nil},
		{"clicks", func(r issueRates) *float64 { return r.clicks }, ptrBool(true)},
		{"unsubscribes", func(r issueRates) *float64 { return r.unsubs }, ptrBool(false)},
		{"complaints", func(r issueRates) *float64 { return r.complaints }, ptrBool(false)},
		{"bounces", func(r issueRates) *float64 { return r.bounces }, ptrBool(false)},
	} {
		cur := k.pick(now)
		if cur == nil {
			continue
		}
		vals := []float64{}
		for _, p := range past {
			if v := k.pick(p); v != nil {
				vals = append(vals, *v)
			}
		}
		m, ok := median(vals)
		if !ok {
			continue
		}
		change := round(*cur-m, 2)
		tone := "neutral"
		if k.upGood != nil && change != 0 {
			if (change > 0) == *k.upGood {
				tone = "good"
			} else {
				tone = "bad"
			}
		}
		out = append(out, ReportComparison{Metric: k.metric, Change: change, Unit: "points", Median: round(m, 2), Tone: tone})
	}
	return out
}

// legacyOpens says an issue was counted before privacy-protected opens
// were counted apart (imported history, sample data, counters not rebuilt
// since): Apple Mail's prefetches are in its machine opens, so its human
// opens of the measurable and its privacy-protected opens can't be
// compared.
func legacyOpens(c counts) bool { return c.privacy == 0 && c.machine > 0 }

func ptrBool(b bool) *bool { return &b }

// median of the values (the mean of the middle two for an even count).
func median(vs []float64) (float64, bool) {
	if len(vs) == 0 {
		return 0, false
	}
	s := slices.Clone(vs)
	slices.Sort(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2], true
	}
	return (s[n/2-1] + s[n/2]) / 2, true
}
