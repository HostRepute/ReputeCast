package ai

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/reports"
)

// A report's numbers as a summary or an answer writes them (review AN-6,
// AN-14). The model writes every number as its key in braces, {delivered},
// never as digits: the numbers are filled in afterwards, each in the
// reader's format (47,548 · 47.548; 41.2% · 41,2%), so each number written
// is the report's number for the metric it names. Digits the model wrote
// anyway, and a person's edits before a summary is kept, are checked
// against the report's numbers.

// numKind is how a number is written.
type numKind int

const (
	// kindCount: 47,548.
	kindCount numKind = iota
	// kindRate: a percentage with one decimal, as the report writes rates
	// of delivered or sent (41.2%, Outlook's 1.0% bounces).
	kindRate
	// kindSmallRate: a percentage with two decimals, as the report writes
	// complaint, unsubscribe and overall bounce rates (0.24%, 0.04%).
	kindSmallRate
	// kindShare: a whole percentage (38% of clicks, 98% confidence).
	kindShare
	// kindPoints: percentage points with one decimal (0.8).
	kindPoints
)

// reportNumber is one number a summary may write.
type reportNumber struct {
	Value float64
	Kind  numKind
}

// reportNumbers are a report's numbers by key.
type reportNumbers map[string]reportNumber

func (n reportNumbers) count(key string, v int64) {
	n[key] = reportNumber{Value: float64(v), Kind: kindCount}
}

func (n reportNumbers) rate(key string, p *float64) {
	if p != nil {
		n[key] = reportNumber{Value: *p, Kind: kindRate}
	}
}

// small is a rate the report writes with two decimals.
func (n reportNumbers) small(key string, p *float64) {
	if p != nil {
		n[key] = reportNumber{Value: *p, Kind: kindSmallRate}
	}
}

func (n reportNumbers) share(key string, v float64) { n[key] = reportNumber{Value: v, Kind: kindShare} }

// numbersFrom collects a report's numbers: the headline, the top links,
// the A/B test (its decision, confidence and lift), each mailbox
// provider's rates and the unsubscribe reasons.
func numbersFrom(r reports.CampaignReport) reportNumbers {
	n := reportNumbers{}
	h := r.Headline
	n.count("sent", h.Sent)
	n.count("delivered", h.Delivered)
	n.rate("deliveredRate", h.DeliveredRate)
	n.count("humanOpens", h.HumanOpens)
	n.rate("humanOpenRate", h.HumanOpenRate)
	n.count("measurable", h.Measurable)
	n.rate("humanOpenRateMeasurable", h.HumanOpenRateMeasurable)
	n.count("privacyOpens", h.PrivacyOpens)
	n.rate("privacyOpenRate", h.PrivacyOpenRate)
	n.count("scannerOpens", h.MachineOpens)
	n.count("clicks", h.Clicks)
	n.rate("clickRate", h.ClickRate)
	n.rate("clickToOpen", h.ClickToOpen)
	n.count("botClicks", h.BotClicks)
	n.count("unsubscribes", h.Unsubscribes)
	n.small("unsubscribeRate", h.UnsubscribeRate)
	n.count("complaints", h.Complaints)
	n.small("complaintRate", h.ComplaintRate)
	n.small("complaintRateReported", h.ComplaintRateReported)
	n.count("bounces", h.Bounces)
	n.small("bounceRate", h.BounceRate)
	n.count("undelivered", h.Undelivered)
	for i, l := range topLinks(r) {
		n.share(fmt.Sprintf("link%d.share", i+1), math.Round(l.Share))
		n.count(fmt.Sprintf("link%d.clicks", i+1), l.UniqueClicks)
	}
	if t := r.ABTest; t != nil {
		if t.Confidence != nil {
			n.share("ab.confidence", math.Round(*t.Confidence))
		}
		if t.Lift != nil {
			n["ab.lift"] = reportNumber{Value: *t.Lift, Kind: kindPoints}
		}
		n.count("ab.minSample", t.MinSample)
		n.share("ab.threshold", t.Threshold)
		n.count("ab.remaining", t.Remaining)
		for _, v := range t.Variants {
			k := "ab." + v.Key + "."
			n.count(k+"recipients", v.Recipients)
			n.rate(k+"clickRate", v.ClickRate)
			n.rate(k+"humanOpenRate", v.HumanOpenRate)
			n.rate(k+"metricRate", v.MetricRate)
			n.rate(k+"decisionRate", v.DecisionRate)
			n.small(k+"complaintRate", v.ComplaintRate)
		}
	}
	for _, m := range r.Mailboxes {
		k := "mailbox." + m.Key + "."
		n.count(k+"sent", m.Sent)
		n.rate(k+"deliveredRate", m.DeliveredRate)
		n.rate(k+"bounceRate", m.BounceRate)
		n.rate(k+"humanOpenRateMeasurable", m.HumanOpenRateMeasurable)
		n.rate(k+"privacyOpenRate", m.PrivacyOpenRate)
		n.rate(k+"clickRate", m.ClickRate)
		n.small(k+"complaintRate", m.ComplaintRate)
		n.small(k+"unsubscribeRate", m.UnsubscribeRate)
	}
	for _, u := range r.Unsubscribes.Reasons {
		if u.Count > 0 {
			n.count("unsubscribe."+u.Key+".count", u.Count)
			n.share("unsubscribe."+u.Key+".share", math.Round(u.Share))
		}
	}
	for _, b := range r.Bounces.Rows {
		n.count("bounce."+b.Type+".count", b.Count)
	}
	return n
}

// topLinks are the three most-clicked links.
func topLinks(r reports.CampaignReport) []reports.ReportLink {
	links := slices.Clone(r.ClickMap.Links)
	slices.SortStableFunc(links, func(a, b reports.ReportLink) int {
		switch {
		case a.Share > b.Share:
			return -1
		case a.Share < b.Share:
			return 1
		}
		return 0
	})
	if len(links) > 3 {
		links = links[:3]
	}
	return links
}

// format writes a number in a region's format, with the decimals the
// report shows it with.
func (n reportNumber) format(r i18n.Region) string {
	switch n.Kind {
	case kindRate:
		return r.Percent(n.Value, 1)
	case kindSmallRate:
		return r.Percent(n.Value, 2)
	case kindShare:
		return r.Percent(n.Value, 0)
	case kindPoints:
		return r.Decimal(n.Value, 1)
	}
	return r.Int(int64(math.Round(n.Value)))
}

// forms are the ways a number may be written in a region: as format writes
// it, and percentages and points with none to two decimals.
func (n reportNumber) forms(r i18n.Region) []string {
	out := []string{n.format(r)}
	switch n.Kind {
	case kindRate, kindSmallRate, kindShare:
		for d := 0; d <= 2; d++ {
			out = append(out, r.Percent(n.Value, d))
		}
	case kindPoints:
		for d := 0; d <= 2; d++ {
			out = append(out, r.Decimal(n.Value, d))
		}
	}
	return out
}

// placeholderRe finds the numbers a model wrote as their keys.
var placeholderRe = regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z0-9_]+)*)\}`)

// fill writes each {key} as the report's number in the region's format and
// returns the keys that aren't the report's.
func fill(text string, nums reportNumbers, r i18n.Region) (string, []string) {
	var unknown []string
	out := placeholderRe.ReplaceAllStringFunc(text, func(m string) string {
		key := m[1 : len(m)-1]
		n, ok := nums[key]
		if !ok {
			if !slices.Contains(unknown, m) {
				unknown = append(unknown, m)
			}
			return m
		}
		return n.format(r)
	})
	return out, unknown
}

// numberTokenRe finds numbers as any region writes them: 47,548 · 47.548 ·
// 47 548 · 41.2% · 41,2 %.
var numberTokenRe = regexp.MustCompile(`\d+(?:[.,\x{00A0}\x{202F} ]\d{3})*(?:[.,]\d+)?(?:[\x{00A0}\x{202F} ]?%)?`)

// context numbers a text may write that aren't results: the 72 hours
// after which results are final, the 7-day conversion window, the last 6
// issues compared, the 95% a winner needs.
var contextNumbers = []string{"72", "7", "6", "95%", "100%"}

// uncheckedNumbers returns the numbers in text that aren't the report's,
// in the reader's region or the standard format; extra are texts whose
// numbers it may quote as written (the providers' bounce codes).
func uncheckedNumbers(text string, nums reportNumbers, r i18n.Region, extra ...string) []string {
	allowed := map[string]bool{}
	for _, c := range contextNumbers {
		allowed[c] = true
	}
	for _, e := range extra {
		for _, tok := range numberTokenRe.FindAllString(e, -1) {
			allowed[normalizeSpace(tok)] = true
		}
	}
	for _, region := range []i18n.Region{r, i18n.Standard} {
		for _, n := range nums {
			for _, f := range n.forms(region) {
				allowed[normalizeSpace(f)] = true
			}
		}
	}
	var out []string
	for _, tok := range numberTokenRe.FindAllString(text, -1) {
		tok = normalizeSpace(strings.TrimRight(tok, ".,"))
		if tok == "" || allowed[tok] || smallOrYear(tok) {
			continue
		}
		if !slices.Contains(out, tok) {
			out = append(out, tok)
		}
	}
	return out
}

// normalizeSpace writes a number's spaces as plain ones.
func normalizeSpace(s string) string {
	return strings.NewReplacer(" ", " ", " ", " ").Replace(s)
}

// smallOrYear leaves out what isn't a result: a year, a variant's number,
// a one-digit count of things ("two links", "3 sentences").
func smallOrYear(v string) bool {
	if strings.HasSuffix(v, "%") {
		return false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return false
	}
	return (n >= 1990 && n <= 2100 && len(v) == 4) || (n >= 0 && n < 10 && len(v) == 1)
}

// numbersForModel lists the numbers by key for the prompt, with their
// values (so the model can compare them) in key order.
func numbersForModel(nums reportNumbers) map[string]float64 {
	out := make(map[string]float64, len(nums))
	keys := make([]string, 0, len(nums))
	for k := range nums {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = nums[k].Value
	}
	return out
}

// reportContext is what a summary or an answer gets of a report: its
// aggregate numbers by key and what they are about (link labels, the A/B
// test, mailbox providers, unsubscribe reasons, bounce types and the
// providers' codes). There is no address, name or other contact data in a
// report.
type reportContext struct {
	Campaign string             `json:"campaign"`
	Numbers  map[string]float64 `json:"numbers"`
	Links    []contextLink      `json:"topLinks"`
	ABTest   *contextAB         `json:"abTest,omitempty"`
	// Mailboxes are the mailbox providers; ReportsComplaints is false for
	// those that never tell senders about complaints (Gmail, iCloud).
	Mailboxes []contextMailbox `json:"mailboxProviders"`
	Reasons   []contextReason  `json:"unsubscribeReasons"`
	Bounces   []contextBounce  `json:"bounceTypes"`
	// Final is false while opens and clicks are still coming in.
	Final bool `json:"final"`
}

type contextLink struct {
	Label string `json:"label"`
	// URL is the link's host and path, without its query or fragment.
	URL string `json:"url"`
	Key string `json:"key" doc:"link1, link2, link3: link1.share is its share of clicks"`
}

type contextAB struct {
	Kind   string `json:"kind"`
	Metric string `json:"metric"`
	// Outcome: winner (significant), not_significant or too_small (the
	// default went to everyone else), manual (a person picked).
	Outcome  string           `json:"outcome"`
	Winner   string           `json:"winner,omitempty"`
	Default  string           `json:"default"`
	Variants []contextVariant `json:"variants"`
}

type contextVariant struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type contextMailbox struct {
	Key               string `json:"key"`
	Name              string `json:"name"`
	ReportsComplaints bool   `json:"reportsComplaints"`
}

type contextReason struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type contextBounce struct {
	Type  string   `json:"type"`
	Label string   `json:"label"`
	Codes []string `json:"codes"`
}

// contextOf builds what a model gets of a report.
func contextOf(r reports.CampaignReport, nums reportNumbers) reportContext {
	c := reportContext{Campaign: r.Campaign.Name, Numbers: numbersForModel(nums), Links: []contextLink{}, Mailboxes: []contextMailbox{},
		Reasons: []contextReason{}, Bounces: []contextBounce{}, Final: r.Campaign.Final}
	for i, l := range topLinks(r) {
		label := l.Label
		if label == "" {
			label = hostOf(l.URL)
		}
		c.Links = append(c.Links, contextLink{Label: label, URL: pageOf(l.URL), Key: fmt.Sprintf("link%d", i+1)})
	}
	if t := r.ABTest; t != nil {
		ab := &contextAB{Kind: t.Kind, Metric: t.Metric, Outcome: t.Outcome, Default: t.Default, Variants: []contextVariant{}}
		if t.State != "testing" {
			ab.Winner = t.Winner
		}
		for _, v := range t.Variants {
			ab.Variants = append(ab.Variants, contextVariant{Key: v.Key, Label: v.Label})
		}
		c.ABTest = ab
	}
	for _, m := range r.Mailboxes {
		c.Mailboxes = append(c.Mailboxes, contextMailbox{Key: m.Key, Name: m.Name, ReportsComplaints: mailbox.ReportsComplaints(m.Key)})
	}
	for _, u := range r.Unsubscribes.Reasons {
		if u.Count > 0 {
			c.Reasons = append(c.Reasons, contextReason{Key: u.Key, Label: u.Label})
		}
	}
	for _, b := range r.Bounces.Rows {
		cb := contextBounce{Type: b.Type, Label: b.Label, Codes: []string{}}
		for _, code := range b.Codes {
			cb.Codes = append(cb.Codes, code.CodeText())
		}
		c.Bounces = append(c.Bounces, cb)
	}
	return c
}

// numbersRule is how every report prompt asks for numbers.
const numbersRule = "Write every number as its key from <report> in braces, such as {delivered} or {mailbox.outlook.bounceRate}, never as digits: the numbers are filled in afterwards in the reader's format. Use only keys that are there. Describe the numbers: don't explain causes they don't show and don't give advice."

// checkedText fills in a model's numbers and returns what isn't the
// report's: keys it made up and digits that aren't the report's numbers.
func checkedText(text string, nums reportNumbers, region i18n.Region, extra ...string) (string, []string) {
	filled, unknown := fill(text, nums, region)
	unmatched := append([]string{}, unknown...)
	for _, v := range uncheckedNumbers(placeholderRe.ReplaceAllString(filled, ""), nums, region, extra...) {
		if !strings.Contains(strings.Join(unmatched, " "), v) {
			unmatched = append(unmatched, v)
		}
	}
	return filled, unmatched
}

// pageOf is a link's host and path, without the query or fragment (they
// can carry tracking parameters): what the link is about.
func pageOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return hostOf(raw)
	}
	return u.Hostname() + strings.TrimSuffix(u.EscapedPath(), "/")
}
