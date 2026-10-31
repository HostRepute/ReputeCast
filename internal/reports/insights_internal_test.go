package reports

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

func mailboxRow(key, name string, sent int64, bounceRate float64) ReportMailboxRow {
	r := ReportMailboxRow{Key: key, Name: name}
	r.Sent, r.BounceRate = sent, &bounceRate
	return r
}

// TestMailboxBouncesInsight: a mailbox provider bouncing twice the others
// (with enough bounces to mean something) is an insight whose action opens
// that provider's tool: Microsoft SNDS for Outlook, Postmaster Tools for
// Gmail (review AN-15).
func TestMailboxBouncesInsight(t *testing.T) {
	en := i18n.Default
	board := []ReportMailboxRow{
		mailboxRow("gmail", "Gmail", 22071, 0.35), mailboxRow("outlook", "Outlook", 10076, 1.0),
		mailboxRow("yahoo", "Yahoo", 4318, 0.7), mailboxRow("icloud", "iCloud", 3838, 0.3), mailboxRow("other", "Other", 7677, 0.8),
	}
	in := mailboxBouncesInsight(en, board)
	if in == nil {
		t.Fatal("Outlook at 1.0% against 0.5% elsewhere is an insight")
	}
	if in.Kind != "mailbox_bounces" || in.Tone != "warn" || in.Action != "open_snds" || in.Link != sndsURL || in.ActionLabel != "Open Microsoft SNDS" {
		t.Fatalf("Outlook's insight: %+v", in)
	}
	if in.Title != "Outlook bounced 1.0%, twice the rate of the other mailbox providers (0.5%)." {
		t.Fatalf("title %q", in.Title)
	}
	if !strings.HasPrefix(in.Detail, "101 of 10,076 messages.") || !strings.HasSuffix(in.Detail, "SNDS shows how Outlook sees your sending IPs.") {
		t.Fatalf("detail %q", in.Detail)
	}
	// Gmail bouncing most: Postmaster Tools.
	gmail := []ReportMailboxRow{mailboxRow("gmail", "Gmail", 20000, 1.2), mailboxRow("outlook", "Outlook", 10000, 0.3), mailboxRow("yahoo", "Yahoo", 5000, 0.4)}
	if in := mailboxBouncesInsight(en, gmail); in == nil || in.Action != "open_postmaster" || in.Link != postmasterURL || in.Position != nil {
		t.Fatalf("Gmail's insight: %+v", in)
	}
	// Yahoo has no tool to open: the insight stands without an action.
	yahoo := []ReportMailboxRow{mailboxRow("yahoo", "Yahoo", 20000, 1.2), mailboxRow("outlook", "Outlook", 10000, 0.3)}
	if in := mailboxBouncesInsight(en, yahoo); in == nil || in.Action != "" || in.Link != "" {
		t.Fatalf("Yahoo's insight: %+v", in)
	}
	// Under twice the others, or too few bounces to mean something: none.
	if in := mailboxBouncesInsight(en, []ReportMailboxRow{mailboxRow("outlook", "Outlook", 10000, 0.9), mailboxRow("gmail", "Gmail", 10000, 0.5)}); in != nil {
		t.Fatalf("1.8 times the others: %+v", in)
	}
	if in := mailboxBouncesInsight(en, []ReportMailboxRow{mailboxRow("outlook", "Outlook", 1000, 1.0), mailboxRow("gmail", "Gmail", 10000, 0.2)}); in != nil {
		t.Fatalf("10 bounces: %+v", in)
	}
	if in := mailboxBouncesInsight(en, board[:1]); in != nil {
		t.Fatalf("one mailbox provider: %+v", in)
	}
}

func insightLink(pos int, label string, unique int64, share float64) ReportLink {
	return ReportLink{Position: pos, Label: label, URL: "https://docs.acme.io/" + label, UniqueClicks: unique, Clicks: unique, Share: share}
}

// TestTopLinkInsight: a link with 30% or more of its email's human clicks is
// an insight whose action shows it on the click map, at its position. A
// test on content has an email per variant: the link is measured against
// its own email's clicks and the insight names the variant, so "Show on the
// click map" opens that variant's map (review AN-5).
func TestTopLinkInsight(t *testing.T) {
	en := i18n.Default
	m := ReportClickMap{Links: []ReportLink{insightLink(0, "Blog", 205, 6), insightLink(6, "Read the docs", 1297, 38), insightLink(8, "See changelog", 819, 24)}}
	in := topLinkInsight(en, m)
	if in == nil || in.Kind != "top_link" || in.Action != "show_link" || in.Position == nil || *in.Position != 6 || in.Variant != "" {
		t.Fatalf("top link: %+v", in)
	}
	if in.Title != "The “Read the docs” link drew 38% of human clicks." || in.Detail != "1,297 people clicked “Read the docs”, more than any other link." {
		t.Fatalf("words: %q / %q", in.Title, in.Detail)
	}
	if in.ActionLabel != "Show on the click map" {
		t.Fatalf("action %q", in.ActionLabel)
	}
	if in := topLinkInsight(en, ReportClickMap{Links: []ReportLink{insightLink(0, "a", 40, 29), insightLink(1, "b", 30, 22)}}); in != nil {
		t.Fatalf("no link at 30%%: %+v", in)
	}
	// A test on content: A's top link has 35% of A's clicks, B's 52% of B's.
	perVariant := ReportClickMap{PerVariant: true, Links: []ReportLink{insightLink(0, "A docs", 120, 35)}, Variants: []ReportVariantClickMap{
		{Key: "A", SameEmail: true, Links: []ReportLink{insightLink(0, "A docs", 120, 35), insightLink(1, "A blog", 80, 23)}},
		{Key: "B", Links: []ReportLink{insightLink(0, "B guide", 260, 52), insightLink(1, "B blog", 90, 18)}},
	}}
	in = topLinkInsight(en, perVariant)
	if in == nil || in.Variant != "B" || in.Position == nil || *in.Position != 0 {
		t.Fatalf("content test: %+v", in)
	}
	if in.Title != "In variant B's email, the “B guide” link drew 52% of human clicks." ||
		in.Detail != "260 people who got variant B clicked “B guide”, more than any other link of that email." {
		t.Fatalf("content test words: %q / %q", in.Title, in.Detail)
	}
	vi := topLinkInsight(i18n.For(i18n.Vietnamese), perVariant)
	if vi == nil || !strings.Contains(vi.Title, "phiên bản B") {
		t.Fatalf("Vietnamese: %+v", vi)
	}
}

// TestUnrecordedABDecision: a test decided before the engine kept its
// decisions has no snapshot of what decided it. Its card names the winner
// but claims no confidence or significance: the samples' results since
// aren't what it was decided on (review AN-1).
func TestUnrecordedABDecision(t *testing.T) {
	started := time.Date(2026, 9, 3, 14, 0, 0, 0, time.UTC)
	decided := started.Add(2 * time.Hour)
	cfg := parseAB([]byte(`{"kind":"subject","variants":[{"key":"A","subject":"Three launches"},{"key":"B","subject":"What shipped"}],
		"test_percent":20,"metric":"clicks","duration_hours":2,"winner":"B","decided_at":"` + decided.Format(time.RFC3339) + `"}`))
	cell := func(variant string, sent, delivered, clicked int64) store.ReportCounterCellsRow {
		return store.ReportCounterCellsRow{Variant: variant, Sent: sent, Delivered: delivered, Clicked: clicked, MailboxProvider: "gmail"}
	}
	// B's sample clicked a little more: 61% confidence on today's numbers.
	cells := []store.ReportCounterCellsRow{cell("A", 2400, 2390, 120), cell("B", 2400, 2390, 131), cell("", 19200, 19100, 1100)}
	c := store.ReportCampaignRow{StartedAt: &started, Status: "sent"}
	got := abResult(i18n.Default, cfg, sendAB{}, cells, c, decided.Add(96*time.Hour))
	if got.State != "decided" || got.Winner != "B" || got.Outcome != OutcomeUnrecorded {
		t.Fatalf("state %q winner %q outcome %q", got.State, got.Winner, got.Outcome)
	}
	if got.Confidence != nil || got.PValue != nil || got.Lift != nil || got.Significant || got.Leader != "" {
		t.Fatalf("a decision without its record claims statistics: confidence %v p %v lift %v significant %v leader %q",
			got.Confidence, got.PValue, got.Lift, got.Significant, got.Leader)
	}
	if got.Remaining != 19200 || got.RolloutAt == nil || !got.RolloutAt.Equal(decided) {
		t.Fatalf("the rest got B at its decision: %d at %v", got.Remaining, got.RolloutAt)
	}
	for _, v := range got.Variants {
		if v.DecisionRate != nil {
			t.Fatalf("variant %s has a rate as of a decision nobody recorded", v.Key)
		}
		if v.MetricRate == nil {
			t.Fatalf("variant %s has no results", v.Key)
		}
	}
	// The same test with its decision kept is a decision with statistics.
	kept := parseAB([]byte(`{"kind":"subject","variants":[{"key":"A"},{"key":"B"}],"metric":"clicks","duration_hours":2,"winner":"B",
		"decision":{"metric":"clicks","variants":[{"key":"A","eligible":2390,"conversions":40},{"key":"B","eligible":2390,"conversions":80}],
		"outcome":"winner","winner":"B","default":"A","hours":2,"decided_at":"` + decided.Format(time.RFC3339) + `"}}`))
	if got := abResult(i18n.Default, kept, sendAB{}, cells, c, decided.Add(96*time.Hour)); got.Outcome != "winner" || !got.Significant || got.Confidence == nil {
		t.Fatalf("a kept decision: %+v", got)
	}
}
