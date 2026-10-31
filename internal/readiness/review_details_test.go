package readiness_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// SR-13: what Review & send shows beside a finding. A fix says what it
// changes under its button and lists its effects for the fix review; a
// listed IP shows its technical details and the list's DNS answers; and
// only an unusual send asks for the recipient count to be typed.

func fixOf(t *testing.T, c readiness.ReadinessCheck, action string) readiness.ReadinessFix {
	t.Helper()
	for _, f := range c.Fixes {
		if f.Action == action {
			return f
		}
	}
	t.Fatalf("%s has no %s fix: %+v", c.Key, action, c.Fixes)
	return readiness.ReadinessFix{}
}

// "Send via SES only · +3 min · +$0.31" and the rest of the board's notes.
func TestFixNotesAndEffects(t *testing.T) {
	f := digest()
	tag := "?utm_source=newsletter&utm_medium=email&utm_campaign=weekly-digest-142"
	f.Content.Report.LinkList = links("https://acme.io/changelog"+tag, "https://acme.io/roadmap?utm_source=newsletter&utm_medium=email")
	f.Rules = []readiness.RuleFacts{utmRule()}
	res := stored(t, readiness.Assemble(f))

	ip := check(t, res, "reputation.ip_blocklists.203.0.113.25")
	route := fixOf(t, ip, readiness.FixRouteOnly)
	if !strings.HasPrefix(route.Note, "+") || !strings.Contains(route.Note, " min · +$") {
		t.Errorf("route note: %q", route.Note)
	}
	labels := []string{}
	for _, e := range route.Effects {
		labels = append(labels, e.Label)
	}
	if strings.Join(labels, ", ") != "Route, Sending time, Cost, Warm-up, Applies to" || route.Effects[0].Value != "100% via SES" {
		t.Errorf("route effects: %+v", route.Effects)
	}
	for _, x := range ip.Fixes {
		if x.Kind == "link" && x.Note != "No change to this send" {
			t.Errorf("delisting note: %+v", x)
		}
	}

	rule := check(t, res, "rules.r-utm")
	add := fixOf(t, rule, readiness.FixRuleAddParam)
	if add.Note != "Changes 1 link · preview first" || len(add.Effects) != 2 || add.Effects[0].Before != "https://acme.io/roadmap?utm_source=newsletter&utm_medium=email" ||
		add.Effects[0].Value != "https://acme.io/roadmap?utm_source=newsletter&utm_medium=email&utm_campaign=weekly-digest-142" || add.Effects[1].Label != "Unchanged" {
		t.Errorf("parameter effects: %q %+v", add.Note, add.Effects)
	}

	// In Vietnamese, from the same stored run.
	vi := check(t, res.In(i18n.For(i18n.Vietnamese)), "reputation.ip_blocklists.203.0.113.25")
	if n := fixOf(t, vi, readiness.FixRouteOnly).Note; n == route.Note || !strings.Contains(n, "phút") {
		t.Errorf("in Vietnamese: %q", n)
	}
}

// The listed IP's technical details: the list and its zone, the scope, the
// other lists, the source, and the list's answers as dig prints them.
func TestListingTechnicalDetails(t *testing.T) {
	f := digest()
	since := now.AddDate(0, 0, -4)
	f.Reputation.Listings[0].Since = &since
	f.Reputation.Listings[0].Answer = "127.0.0.2"
	f.Reputation.Listings[0].TXT = "Your ISP AS64500 is UCEPROTECT-Level3 listed for hosting a total of 41 abusers."
	c := check(t, stored(t, readiness.Assemble(f)), "reputation.ip_blocklists.203.0.113.25")
	rows := map[string]string{}
	for _, r := range c.Technical {
		rows[r.Label] = r.Value
	}
	if rows["Blocklist"] != "UCEPROTECT Level 3 · dnsbl-3.uceprotect.net" || rows["Listing scope"] == "" ||
		rows["Other lists"] != "203.0.113.25 is clean on the other 83" || !strings.HasPrefix(rows["Source"], "HostRepute hourly check, ") ||
		!strings.Contains(rows["Source"], " · listed since ") {
		t.Fatalf("technical rows: %+v", c.Technical)
	}
	want := []string{"$ dig +short 25.113.0.203.dnsbl-3.uceprotect.net A", "127.0.0.2",
		"$ dig +short 25.113.0.203.dnsbl-3.uceprotect.net TXT", `"Your ISP AS64500 is UCEPROTECT-Level3 listed for hosting a total of 41 abusers."`}
	if strings.Join(c.Raw, "\n") != strings.Join(want, "\n") {
		t.Fatalf("raw DNS:\n%s", strings.Join(c.Raw, "\n"))
	}
	// Without the list's answers there is nothing raw to show.
	f.Reputation.Listings[0].Answer, f.Reputation.Listings[0].TXT = "", ""
	if c := check(t, stored(t, readiness.Assemble(f)), "reputation.ip_blocklists.203.0.113.25"); len(c.Raw) != 0 {
		t.Errorf("raw without answers: %v", c.Raw)
	}
}

// A complaint rate without data reads n/a with the reason, never 0%
// (F-2, F-3, C13): Gmail and iCloud never report complaints.
func TestComplaintRateNotReported(t *testing.T) {
	f := digest()
	f.Route.Shares[1].Complaint = nil
	res := stored(t, readiness.Assemble(f))
	c := check(t, res, "reputation.provider.p-postal")
	if c.Status != readiness.Pass || !strings.Contains(c.Title, "Postal bounce 0.9% and complaint n/a, ") || !strings.Contains(c.Short, "complaints n/a") ||
		!strings.HasSuffix(c.Detail, "and Gmail and iCloud never do.") {
		t.Fatalf("not reported: %+v", c)
	}
	if vi := check(t, res.In(i18n.For(i18n.Vietnamese)), "reputation.provider.p-postal"); !strings.Contains(vi.Title, "khiếu nại không có") {
		t.Errorf("in Vietnamese: %q", vi.Title)
	}
	// Too few messages to judge: still n/a, not 0%.
	f.Route.Shares[1].Sent7d = 3
	if c := check(t, stored(t, readiness.Assemble(f)), "reputation.provider.p-postal"); !strings.Contains(c.Detail, "complaint n/a so far") {
		t.Errorf("too few: %q", c.Detail)
	}
}

// Only an unusual send asks for the count: much larger than usual, a first
// send to a list, from a domain or through a provider, a reputation warning
// above low impact. A routine send is confirmed with checkboxes.
func TestTypedConfirmationOnlyWhenUnusual(t *testing.T) {
	f := digest()
	f.Confirm = readiness.ConfirmFacts{Usual: 30000, Sends: 5, Multiple: 1.5}
	if c := readiness.Assemble(f).Estimates.Confirm; c == nil || c.Typed || len(c.Reasons) != 0 || c.Usual != 30000 {
		t.Fatalf("routine: %+v", c)
	}
	f.Confirm = readiness.ConfirmFacts{Usual: 15000, Sends: 5, Multiple: 1.5, FirstList: "Beta Program", FirstDomain: "news.acme.io", FirstProviders: []string{"Postal"}}
	res := stored(t, readiness.Assemble(f))
	c := res.Estimates.Confirm
	want := []string{"31,604 recipients is 2.1× the usual 15,000 for these lists", "the first send to Beta Program", "the first send from news.acme.io", "the first send through Postal"}
	if c == nil || !c.Typed || strings.Join(c.Reasons, " | ") != strings.Join(want, " | ") {
		t.Fatalf("unusual: %+v", c)
	}
	if vi := res.In(i18n.For(i18n.Vietnamese)).Estimates.Confirm; vi == nil || vi.Reasons[1] != "lần gửi đầu tiên tới Beta Program" {
		t.Errorf("in Vietnamese: %+v", vi)
	}
	// A listing above low impact asks too.
	f = digest()
	f.Reputation.Listings[0].Impact = "medium"
	if c := readiness.Assemble(f).Estimates.Confirm; c == nil || !c.Typed || c.Reasons[0] != "203.0.113.25 is on UCEPROTECT Level 3 (medium impact)" {
		t.Fatalf("reputation: %+v", c)
	}
	// HostRepute's hourly check time goes with the run.
	if at := readiness.Assemble(digest()).Estimates.HourlyAt; at == nil || !at.Equal(now.Add(-time.Minute)) {
		t.Errorf("hourly check: %v", at)
	}
}

// R1.U3: a listed link is replaced in one click only with where it really
// leads, once that address was looked up on the URI blocklists and found on
// none; the fix review shows the link before and after. Otherwise the link
// is edited in the composer: no address is guessed.
func TestListedLinkReplacedOnlyWithKnownDestination(t *testing.T) {
	f := digest()
	listed := contentcheck.ContentCheckLink{URL: "https://go.acme-promo.link/sale", Host: "go.acme-promo.link", Text: "Claim your discount", Kind: "button", Path: []int{2, 0, 3},
		Probed: true, Status: 200, Final: "https://acme.io/pricing/annual", FinalClean: true}
	f.Content.Report.LinkList = append(f.Content.Report.LinkList, listed)
	// A guessed fix stored by an older content check is never offered.
	f.Content.Report.Issues = []contentcheck.ContentCheckIssue{{ID: "links.blocklist", Severity: contentcheck.Block, Group: "Links", URL: listed.URL, Path: listed.Path,
		Title: "Button links to go.acme-promo.link, listed on SURBL and Spamhaus DBL", Tag: "Link domain on SURBL and Spamhaus DBL",
		Fixes: []contentcheck.ContentCheckFix{{Kind: "replace_link", Label: "Replace with acme.io/sale", Value: "https://acme.io/sale"}, {Kind: "edit_link", Label: "Edit link"}}}}
	res := stored(t, readiness.Assemble(f))
	c := check(t, res, "links.blocklist.go.acme-promo.link")
	replace := fixOf(t, c, readiness.FixReplaceLink)
	if replace.Label != "Replace with destination URL" || replace.Params["to"] != "https://acme.io/pricing/annual" || replace.Params["from"] != listed.URL {
		t.Fatalf("replace: %+v", replace)
	}
	if replace.Note != "Changes the link" || len(replace.Effects) != 3 || replace.Effects[0].Label != "“Claim your discount” button" ||
		replace.Effects[0].Before != listed.URL || replace.Effects[0].Value != "https://acme.io/pricing/annual" ||
		replace.Effects[1].Value != "Where the link leads today, on none of Spamhaus DBL and SURBL" {
		t.Fatalf("effects: %q %+v", replace.Note, replace.Effects)
	}
	if vi := check(t, res.In(i18n.For(i18n.Vietnamese)), c.Key); fixOf(t, vi, readiness.FixReplaceLink).Label != "Thay bằng URL đích" {
		t.Errorf("in Vietnamese: %+v", vi.Fixes)
	}

	// The destination wasn't found clean (listed, or its lookup failed), or
	// the link didn't redirect: Edit link only.
	for _, final := range []string{listed.Final, ""} {
		g := digest()
		x := listed
		x.Final, x.FinalClean = final, false
		g.Content.Report.LinkList = append(g.Content.Report.LinkList, x)
		g.Content.Report.Issues = f.Content.Report.Issues
		c := check(t, stored(t, readiness.Assemble(g)), "links.blocklist.go.acme-promo.link")
		if len(c.Fixes) != 1 || c.Fixes[0].Kind != "link" || c.Fixes[0].Label != "Edit link" {
			t.Errorf("without a clean destination (final %q): %+v", final, c.Fixes)
		}
	}

	// A shortener's destination follows the same rule.
	g := digest()
	short := contentcheck.ContentCheckLink{URL: "https://bit.ly/3xK9tQ", Host: "bit.ly", Text: "Read the post", Kind: "link", Path: []int{1}, Probed: true, Status: 200,
		Final: "https://blog.acme.io/annual-plans"}
	g.Content.Report.LinkList = append(g.Content.Report.LinkList, short)
	g.Content.Report.Issues = []contentcheck.ContentCheckIssue{{ID: "links.shortener", Severity: contentcheck.Warn, Group: "Links", URL: short.URL, Path: short.Path}}
	if c := check(t, stored(t, readiness.Assemble(g)), "links.shortener"); hasFix(c, readiness.FixReplaceLink) {
		t.Errorf("a shortener's destination nobody checked: %+v", c.Fixes)
	}
	g.Content.Report.LinkList[len(g.Content.Report.LinkList)-1].FinalClean = true
	if c := check(t, stored(t, readiness.Assemble(g)), "links.shortener"); fixOf(t, c, readiness.FixReplaceLink).Params["to"] != short.Final {
		t.Errorf("a shortener's clean destination: %+v", c.Fixes)
	}
}
