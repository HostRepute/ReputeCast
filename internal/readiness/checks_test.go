package readiness_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// The SR-7 checks, one test each, on the Weekly Digest of the boards.

func mustCheck(t *testing.T, res readiness.ReadinessResult, key string) readiness.ReadinessCheck {
	t.Helper()
	c, ok := byKey(res)[key]
	if !ok {
		keys := []string{}
		for _, c := range res.Checks {
			keys = append(keys, c.Key)
		}
		t.Fatalf("no %s check in %v", key, keys)
	}
	return c
}

func noCheck(t *testing.T, res readiness.ReadinessResult, key string) {
	t.Helper()
	if c, ok := byKey(res)[key]; ok {
		t.Fatalf("unexpected %s: %+v", key, c)
	}
}

// A/B variants B and C are checked like the campaign: a spammy subject in
// B warns on its own row, a content variant without a footer blocks.
func TestVariantsAreChecked(t *testing.T) {
	f := digest()
	f.Campaign.ABKind = "subject"
	f.Content.Variants = []readiness.VariantReport{{Key: "B", Subject: "LAST CHANCE!!! 50% OFF", Report: contentcheck.ContentCheckReport{
		SubjectHeuristics: []string{"all caps", "repeated punctuation", "urgency"}}}}
	res := readiness.Assemble(f)
	c := mustCheck(t, res, "content.spam.variant.B")
	if c.Status != readiness.Warn || !strings.Contains(c.Title, "Variant B") || c.Ack == "" || !strings.Contains(c.Fixes[0].Href, "variant=B") {
		t.Fatalf("variant B: %+v", c)
	}
	if spam := mustCheck(t, res, "content.spam"); spam.Status != readiness.Pass || !strings.Contains(spam.Detail, "B: “LAST CHANCE") {
		t.Fatalf("A's row names both subjects: %+v", spam)
	}

	f = digest()
	f.Campaign.ABKind = "content"
	f.Content.Variants = []readiness.VariantReport{{Key: "C", Subject: f.Campaign.Subject, Content: true, Report: contentcheck.ContentCheckReport{
		Issues: []contentcheck.ContentCheckIssue{{ID: "compliance.footer", Severity: contentcheck.Block, Title: "No unsubscribe link or postal address"}}}}}
	res = readiness.Assemble(f)
	if c := mustCheck(t, res, "compliance.footer.variant.C"); c.Status != readiness.Block || res.Verdict != readiness.VerdictBlocked {
		t.Fatalf("variant C's footer: %+v (%s)", c, res.Verdict)
	}
	if c := mustCheck(t, res, "content.spam"); c.Status != readiness.Pass {
		t.Fatalf("A's subject is fine: %+v", c)
	}
}

// A broken personalization tag blocks (the send would stop at preparing);
// a field that doesn't exist warns even with a fallback.
func TestPersonalizationErrorsBlock(t *testing.T) {
	f := digest()
	f.Content.Report.Issues = []contentcheck.ContentCheckIssue{{ID: "personalization.syntax", Severity: contentcheck.Block, Title: "Body: Line 3: unterminated tag"}}
	res := readiness.Assemble(f)
	c := mustCheck(t, res, "content.personalization")
	if c.Status != readiness.Block || !strings.Contains(c.Detail, "Line 3") || res.Verdict != readiness.VerdictBlocked {
		t.Fatalf("syntax: %+v", c)
	}

	f = digest()
	f.Content.Report.Issues = []contentcheck.ContentCheckIssue{{ID: "personalization.unknown_field", Severity: contentcheck.Warn,
		Title: "{{ frist_name }} isn't a contact field, so everyone gets “there”"}}
	if c := mustCheck(t, readiness.Assemble(f), "content.personalization"); c.Status != readiness.Warn || !strings.Contains(c.Title, "frist_name") || c.Ack == "" {
		t.Fatalf("unknown field: %+v", c)
	}
}

// The first-name tag's fallback, with the recipients who need it (board:
// "The first-name tag shows "there" for the 1,817 recipients without a
// first name"); without a fallback they'd get an empty greeting.
func TestPersonalizationFallbacks(t *testing.T) {
	f := digest()
	f.Audience.NoFirstName = 1817
	f.Content.Fields = []readiness.FieldUse{{Field: "first_name", HasFallback: true, Fallback: "there"}}
	c := mustCheck(t, readiness.Assemble(f), "content.personalization")
	if c.Status != readiness.Pass || c.Title != "Personalization has fallbacks" || c.Detail != `The first-name tag shows "there" for the 1,817 recipients without a first name` {
		t.Fatalf("fallback: %q / %q", c.Title, c.Detail)
	}
	f.Content.Fields[0].HasFallback = false
	if c := mustCheck(t, readiness.Assemble(f), "content.personalization"); c.Status != readiness.Warn || !strings.Contains(c.Title, "1,817") {
		t.Fatalf("no fallback: %+v", c)
	}
	f.Audience.NoFirstName = 0
	if c := mustCheck(t, readiness.Assemble(f), "content.personalization"); c.Status != readiness.Pass || c.Title != "Every recipient has a first name" {
		t.Fatalf("nobody needs it: %+v", c)
	}
	f.Content.Fields = nil
	if c := mustCheck(t, readiness.Assemble(f), "content.personalization"); c.Status != readiness.Pass || c.Title != "No personalization tags" {
		t.Fatalf("no tags: %+v", c)
	}
}

// Spam heuristics cover the preheader and the body, not only the subject.
func TestSpamInPreheaderAndBody(t *testing.T) {
	f := digest()
	f.Content.Report.PreheaderHeuristics = []string{"urgency"}
	c := mustCheck(t, readiness.Assemble(f), "content.spam")
	if c.Status != readiness.Warn || c.Title != "The preheader trips spam heuristics" || c.Detail != "Preheader: urgency phrasing" || c.Ack == "" {
		t.Fatalf("preheader: %+v", c)
	}
	f = digest()
	f.Content.Report.BodyHeuristics = []string{"spam phrases"}
	if c := mustCheck(t, readiness.Assemble(f), "content.spam"); c.Status != readiness.Warn || !strings.Contains(c.Title, "body text") {
		t.Fatalf("body: %+v", c)
	}
	// The board's passing row.
	c = mustCheck(t, readiness.Assemble(digest()), "content.spam")
	if c.Title != "No spam triggers in subject, preheader or body" || c.Detail != `Subject: "Weekly Digest #142: feature flags are GA" · no all-caps, no urgency phrases` {
		t.Fatalf("pass: %q / %q", c.Title, c.Detail)
	}
}

// One image-ratio threshold: 60%, the composer's.
func TestImageRatioThresholdIsTheComposers(t *testing.T) {
	f := digest()
	f.Content.Report.ImageShare = contentcheck.ImageShareLimit
	if c := mustCheck(t, readiness.Assemble(f), "content.image_ratio"); c.Status != readiness.Warn {
		t.Fatalf("at the limit: %+v", c)
	}
	f.Content.Report.ImageShare = contentcheck.ImageShareLimit - 0.01
	if c := mustCheck(t, readiness.Assemble(f), "content.image_ratio"); c.Status != readiness.Pass || c.Detail != "7 images, 1,140 words · recommended under 60%" {
		t.Fatalf("under: %+v", c)
	}
}

// The size Gmail clips is measured after the links are rewritten for
// tracking (board: "67 KB with tracking links, under Gmail's 102 KB clip").
func TestSizeMeasuredWithTracking(t *testing.T) {
	f := digest()
	f.Content.TrackedBytes = 67 * 1024
	c := mustCheck(t, readiness.Assemble(f), "content.size")
	if c.Status != readiness.Pass || c.Title != "67 KB with tracking links, under Gmail's 102 KB clip" ||
		c.Detail != "64 KB of HTML including 11 KB of inlined CSS, plus about 3 KB once links are rewritten for tracking" {
		t.Fatalf("board: %q / %q", c.Title, c.Detail)
	}
	// 82 KB of HTML is under the warning, 88 KB as sent isn't.
	f.Content.Report.SizeBytes = 82 * 1024
	f.Content.TrackedBytes = 88 * 1024
	if c := mustCheck(t, readiness.Assemble(f), "content.size"); c.Status != readiness.Warn || !strings.Contains(c.Title, "88 KB with tracking links") {
		t.Fatalf("near the clip once tracked: %+v", c)
	}
}

func manyLinks(n, probed int) []contentcheck.ContentCheckLink {
	out := []contentcheck.ContentCheckLink{}
	for i := range n {
		l := contentcheck.ContentCheckLink{URL: fmt.Sprintf("https://acme.io/p/%d", i), Host: "acme.io", Kind: "link", Path: []int{}}
		if i < probed {
			l.Probed, l.Status, l.Millis = true, 200, 100+i
		}
		out = append(out, l)
	}
	return out
}

// Link counts say how many were checked: 40 of 60 isn't "all respond".
func TestHonestLinkCounts(t *testing.T) {
	f := digest()
	f.Content.Report.LinkList = manyLinks(60, 40)
	res := readiness.Assemble(f)
	c := mustCheck(t, res, "links.respond")
	if c.Status != readiness.Unknown || !strings.HasPrefix(c.Title, "40 of 60 links respond; 20 couldn’t be checked") || !strings.Contains(c.Detail, "20 links weren’t requested") {
		t.Fatalf("40 of 60: %+v", c)
	}
	if !res.Incomplete {
		t.Error("links not checked make the run incomplete")
	}
	f.Content.Report.LinkList = manyLinks(14, 14)
	if c := mustCheck(t, readiness.Assemble(f), "links.respond"); c.Status != readiness.Pass || c.Title != "14 of 14 links respond" || !strings.HasPrefix(c.Detail, "HTTP 200 · slowest acme.io/p/13") {
		t.Fatalf("board: %q / %q", c.Title, c.Detail)
	}
}

// A site that refuses automated requests (LinkedIn's 999) is unknown, not
// broken and not fine.
func TestBotBlockedLinksAreUnknown(t *testing.T) {
	f := digest()
	f.Content.Report.LinkList = append(links("https://acme.io/blog"), contentcheck.ContentCheckLink{URL: "https://www.linkedin.com/company/acme", Host: "www.linkedin.com",
		Kind: "social", Path: []int{}, Probed: true, Status: 999, BotBlocked: true})
	c := mustCheck(t, readiness.Assemble(f), "links.respond")
	if c.Status != readiness.Unknown || !strings.Contains(c.Detail, "linkedin.com/company/acme refused the automated check (HTTP 999)") {
		t.Fatalf("999: %+v", c)
	}
}

// The links of every language version are requested and counted: a link
// broken only in the Vietnamese version warns.
func TestLinksOfEveryLanguageVersion(t *testing.T) {
	f := digest()
	vi := contentcheck.ContentCheckReport{LinkList: append(links("https://acme.io/blog"), contentcheck.ContentCheckLink{URL: "https://acme.io/vi/blog", Host: "acme.io",
		Kind: "link", Path: []int{1}, Probed: true, Status: 404, Broken: true})}
	f.Content.Versions = []readiness.VersionReport{{Language: "vi", Report: vi}}
	f.Languages.Source = "en"
	c := mustCheck(t, readiness.Assemble(f), "links.respond")
	if c.Status != readiness.Warn || !strings.Contains(c.Title, "1 of 4 links") || !strings.Contains(c.Detail, "acme.io/vi/blog returns 404") {
		t.Fatalf("Vietnamese link: %+v", c)
	}
}

// Gmail's spam rate comes from Postmaster Tools: over 0.3% blocks, over
// 0.1% warns, and it's named in the bulk-sender row.
func TestGmailPostmasterSpamRate(t *testing.T) {
	day := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	f := digest()
	f.Compliance.Gmail = &readiness.PostmasterFacts{Domain: "news.acme.io", Day: day, SpamRate: f64(0.04)}
	c := mustCheck(t, readiness.Assemble(f), "compliance.bulk_sender")
	if c.Status != readiness.Pass || !strings.Contains(c.Detail, "Gmail spam rate 0.04%") {
		t.Fatalf("under: %+v", c)
	}
	f.Compliance.Gmail.SpamRate = f64(0.18)
	if c := mustCheck(t, readiness.Assemble(f), "compliance.bulk_sender"); c.Status != readiness.Warn || !strings.Contains(c.Title, "Gmail spam rate 0.18%") || c.Ack == "" {
		t.Fatalf("over 0.1%%: %+v", c)
	}
	f.Compliance.Gmail.SpamRate = f64(0.34)
	if c := mustCheck(t, readiness.Assemble(f), "compliance.bulk_sender"); c.Status != readiness.Block || !strings.Contains(c.Title, "over Google’s 0.3% limit") {
		t.Fatalf("over 0.3%%: %+v", c)
	}
}

// DMARC alignment is judged per provider: Postal without a verified DKIM
// key passes through its return path under relaxed alignment, not under
// strict SPF alignment (aspf=s).
func TestDMARCAlignmentPerProvider(t *testing.T) {
	f := digest()
	f.Auth.DKIM[1] = readiness.ProviderAuthFacts{Short: "Postal", OK: false, Detail: "DKIM records not verified"}
	f.Auth.DMARCRecord = "v=DMARC1; p=quarantine"
	if c := mustCheck(t, readiness.Assemble(f), "auth.dmarc"); c.Status != readiness.Pass {
		t.Fatalf("aligned through SPF: %+v", c)
	}
	f.Auth.DMARCRecord = "v=DMARC1; p=quarantine; aspf=s"
	c := mustCheck(t, readiness.Assemble(f), "auth.dmarc")
	if c.Status != readiness.Block || c.Title != "Mail through Postal won’t pass DMARC" || !strings.Contains(c.Detail, "aspf=s") {
		t.Fatalf("strict: %+v", c)
	}
}

// One-click unsubscribe depends on each provider: one that doesn't sign
// the headers warns; one that sets its own is named.
func TestOneClickPerProvider(t *testing.T) {
	f := digest()
	f.Route.Shares[1].OneClick = "unsigned"
	c := mustCheck(t, readiness.Assemble(f), "compliance.list_unsubscribe")
	if c.Status != readiness.Warn || !strings.Contains(c.Title, "Postal") || c.Ack == "" || !hasFix(c, "route_only") {
		t.Fatalf("unsigned: %+v", c)
	}
	f.Route.Shares[1].OneClick = "replaced"
	if c := mustCheck(t, readiness.Assemble(f), "compliance.list_unsubscribe"); c.Status != readiness.Pass || !strings.Contains(c.Detail, "Postal sets its own one-click unsubscribe") {
		t.Fatalf("replaced: %+v", c)
	}
}

// Dedicated IPs need forward-confirmed reverse DNS: missing blocks, a
// failed lookup is unknown, a pass is named in the IP's row.
func TestReverseDNSOfDedicatedIPs(t *testing.T) {
	f := digest()
	f.Reputation.Listings = nil
	f.Reputation.Status = readiness.Pass
	f.Reputation.ReverseDNS = []readiness.ReverseDNSFacts{{IP: "203.0.113.25", Provider: "Postal", PTR: "mta.acme.io", Status: "pass"}}
	res := readiness.Assemble(f)
	noCheck(t, res, "reputation.rdns.203.0.113.25")
	if c := mustCheck(t, res, "reputation.ip_blocklists.203.0.113.25"); !strings.Contains(c.Detail, "reverse DNS mta.acme.io") {
		t.Fatalf("pass: %+v", c)
	}
	f.Reputation.ReverseDNS[0] = readiness.ReverseDNSFacts{IP: "203.0.113.25", Provider: "Postal", Status: "missing"}
	c := mustCheck(t, readiness.Assemble(f), "reputation.rdns.203.0.113.25")
	if c.Status != readiness.Block || c.Title != "Postal IP 203.0.113.25 has no reverse DNS" || !strings.Contains(c.Detail, "5.7.25") || !hasFix(c, "route_only") {
		t.Fatalf("missing: %+v", c)
	}
	f.Reputation.ReverseDNS[0] = readiness.ReverseDNSFacts{IP: "203.0.113.25", Provider: "Postal", Status: "error", Trouble: readiness.TroubleTimeout}
	if c := mustCheck(t, readiness.Assemble(f), "reputation.rdns.203.0.113.25"); c.Status != readiness.Unknown {
		t.Fatalf("lookup failed: %+v", c)
	}
}

// The tracking domain carries every click: named among the link domains
// when it resolves; blocks when it doesn't; unknown when DNS doesn't answer;
// warns over plain HTTP.
func TestTrackingDomain(t *testing.T) {
	f := digest()
	f.Content.TrackClicks, f.Content.TrackOpens = true, true
	f.Content.TrackingHost, f.Content.TrackingHTTPS, f.Content.TrackingDNS = "go.news.acme.io", true, "ok"
	res := readiness.Assemble(f)
	noCheck(t, res, "links.tracking_domain")
	c := mustCheck(t, res, "links.blocklists")
	if c.Title != "Link domains checked live on Spamhaus DBL and SURBL: none listed" ||
		c.Detail != "4 domains: acme.io, docs.acme.io, github.com and the tracking domain go.news.acme.io" {
		t.Fatalf("board: %q / %q", c.Title, c.Detail)
	}
	f.Content.TrackingDNS = "missing"
	if c := mustCheck(t, readiness.Assemble(f), "links.tracking_domain"); c.Status != readiness.Block {
		t.Fatalf("missing: %+v", c)
	}
	f.Content.TrackingDNS = readiness.TroubleServFail
	if c := mustCheck(t, readiness.Assemble(f), "links.tracking_domain"); c.Status != readiness.Unknown || !strings.Contains(c.Detail, "SERVFAIL") {
		t.Fatalf("servfail: %+v", c)
	}
	f.Content.TrackingDNS, f.Content.TrackingHTTPS = "ok", false
	if c := mustCheck(t, readiness.Assemble(f), "links.tracking_domain"); c.Status != readiness.Warn || c.Ack == "" {
		t.Fatalf("plain HTTP: %+v", c)
	}
}

// The route's fallback must pass the vetting failover runs (C11).
func TestFallbackIsVetted(t *testing.T) {
	f := digest()
	f.Route.Fallback = &readiness.FallbackFacts{ProviderID: "p-pm", Name: "Postmark"}
	res := readiness.Assemble(f)
	noCheck(t, res, "routing.fallback")
	if c := mustCheck(t, res, "routing.health"); !strings.Contains(c.Detail, "failover: pause and ask, to Postmark") {
		t.Fatalf("fallback named: %+v", c)
	}
	f.Route.Fallback = &readiness.FallbackFacts{ProviderID: "p-pm", Name: "Postmark", Failed: "domain", Text: readiness.RawText("The sender isn't verified on Postmark.")}
	c := mustCheck(t, readiness.Assemble(f), "routing.fallback")
	if c.Status != readiness.Warn || !strings.Contains(c.Title, "Postmark") || !strings.Contains(c.Detail, "isn't verified") || c.Ack == "" {
		t.Fatalf("vetting failed: %+v", c)
	}
}

// Images load over HTTPS (board: "All 7 images load over HTTPS" · "Hosted
// on cdn.acme.io · largest 182 KB"); plain HTTP and broken images warn.
func TestImagesOverHTTPS(t *testing.T) {
	f := digest()
	f.Content.Report.ImageList = []contentcheck.ContentCheckImage{}
	for i := range 7 {
		f.Content.Report.ImageList = append(f.Content.Report.ImageList, contentcheck.ContentCheckImage{URL: fmt.Sprintf("https://cdn.acme.io/%d.png", i), Host: "cdn.acme.io",
			Path: []int{i}, Probed: true, Status: 200, Bytes: int64(20+i*27) * 1024})
	}
	c := mustCheck(t, readiness.Assemble(f), "content.images")
	if c.Status != readiness.Pass || c.Title != "All 7 images load over HTTPS" || c.Detail != "Hosted on cdn.acme.io · largest 182 KB" {
		t.Fatalf("board: %q / %q", c.Title, c.Detail)
	}
	f.Content.Report.ImageList[2] = contentcheck.ContentCheckImage{URL: "http://old.acme.io/2.png", Host: "old.acme.io", Insecure: true, Path: []int{2}}
	if c := mustCheck(t, readiness.Assemble(f), "content.images"); c.Status != readiness.Warn || c.Title != "1 of 7 images load over plain HTTP" {
		t.Fatalf("insecure: %q", c.Title)
	}
	f.Content.Report.ImageList[3].Broken, f.Content.Report.ImageList[3].Status = true, 404
	if c := mustCheck(t, readiness.Assemble(f), "content.images"); c.Status != readiness.Warn || !strings.Contains(c.Detail, "cdn.acme.io/3.png returns 404") {
		t.Fatalf("broken: %+v", c)
	}
}

// The new rows read in Vietnamese too.
func TestReviewChecksInVietnamese(t *testing.T) {
	f := digest()
	f.Audience.NoFirstName = 1817
	f.Content.Fields = []readiness.FieldUse{{Field: "first_name", HasFallback: true, Fallback: "bạn"}}
	f.Content.TrackedBytes = 67 * 1024
	vi := stored(t, readiness.Assemble(f)).In(i18n.For(i18n.Vietnamese))
	if c := check(t, vi, "content.personalization"); !strings.Contains(c.Detail, "1.817") || !strings.Contains(c.Detail, "bạn") {
		t.Errorf("personalization in Vietnamese: %q", c.Detail)
	}
	if c := check(t, vi, "content.size"); !strings.Contains(c.Title, "67 KB kể cả liên kết theo dõi") {
		t.Errorf("size in Vietnamese: %q", c.Title)
	}
	if c := check(t, vi, "links.respond"); c.Title != "3/3 liên kết phản hồi" {
		t.Errorf("links in Vietnamese: %q", c.Title)
	}
}
