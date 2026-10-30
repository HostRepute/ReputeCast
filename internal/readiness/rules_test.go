package readiness_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// SR-10: the workspace's sending rules are a Send Readiness group of their
// own. A broken rule warns or blocks with its severity, says what broke it,
// and shows the rule and the links behind it in its technical details; a
// kept rule passes.

func utmRule() readiness.RuleFacts {
	return readiness.RuleFacts{ID: "r-utm", Name: "Links need UTM tags", Kind: readiness.RuleLinkParam, Operator: "on_every", Value: "utm_campaign",
		Severity: "warn", AppliesTo: "broadcast", Message: "ReputeCast counts clicks either way; the tag is for your web analytics.", AddedBy: "Marcus Chen"}
}

func withRules(f readiness.Facts, rules ...readiness.RuleFacts) readiness.Facts {
	f.Rules = rules
	return f
}

// The board's UTM rule: 2 of 14 links miss utm_campaign, a Warn with
// "Add utm_campaign=weekly-digest-142", confirmed before sending.
func TestRuleLinkParam(t *testing.T) {
	f := digest()
	tag := "?utm_source=newsletter&utm_medium=email&utm_campaign=weekly-digest-142"
	f.Content.Report.LinkList = links("https://acme.io/changelog"+tag, "https://docs.acme.io/flags"+tag, "https://acme.io/roadmap?utm_source=newsletter&utm_medium=email",
		"https://docs.acme.io/changelog?utm_source=newsletter")
	res := stored(t, readiness.Assemble(withRules(f, utmRule())))
	c := check(t, res, "rules.r-utm")
	if c.Status != readiness.Warn || c.Group != readiness.GroupRules || c.Title != "Workspace rule: links need UTM tags · 2 of 4 links have no utm_campaign" ||
		c.Ack != "I understand 2 of 4 links have no utm_campaign, which the workspace rule asks for." {
		t.Fatalf("broken: %+v", c)
	}
	if len(c.Fixes) != 1 || c.Fixes[0].Action != readiness.FixRuleAddParam || c.Fixes[0].Label != "Add utm_campaign=weekly-digest-142" ||
		c.Fixes[0].Params["param"] != "utm_campaign" || c.Fixes[0].Params["value"] != "weekly-digest-142" {
		t.Fatalf("fix: %+v", c.Fixes)
	}
	if len(c.Raw) != 2 || !strings.HasPrefix(c.Raw[0], "https://acme.io/roadmap") || len(c.Technical) < 3 || c.Technical[0].Label != "Rule" ||
		!strings.Contains(c.Technical[0].Value, "added by Marcus Chen") || c.Technical[2].Value != "2 of 4, both on acme.io domains" {
		t.Fatalf("technical details: %+v · %v", c.Technical, c.Raw)
	}
	var g *readiness.ReadinessGroup
	for i := range res.Groups {
		if res.Groups[i].Key == readiness.GroupRules {
			g = &res.Groups[i]
		}
	}
	if g == nil || g.Title != "Workspace rules" || g.Caption != "Settings › Sending rules" {
		t.Fatalf("group: %+v", g)
	}
	// In Vietnamese, from the same stored run.
	vi := check(t, res.In(i18n.For(i18n.Vietnamese)), "rules.r-utm")
	if vi.Title == c.Title || !strings.Contains(vi.Title, "utm_campaign") {
		t.Errorf("in Vietnamese: %q", vi.Title)
	}
	// Every link tagged: the rule passes.
	f.Content.Report.LinkList = links("https://acme.io/changelog"+tag, "https://docs.acme.io/flags"+tag)
	if c := check(t, stored(t, readiness.Assemble(withRules(f, utmRule()))), "rules.r-utm"); c.Status != readiness.Pass || len(c.Fixes) != 0 ||
		c.Title != "Links need UTM tags: all 2 links have utm_campaign" {
		t.Fatalf("kept: %+v", c)
	}
}

// A Block rule stops the send: a shortener when the rule keeps them out.
func TestRuleShortenersBlock(t *testing.T) {
	f := digest()
	rule := readiness.RuleFacts{ID: "r-short", Name: "No URL shorteners", Kind: readiness.RuleLinkDomain, Operator: "not_one_of", Value: readiness.Shorteners,
		Severity: "block", AppliesTo: "both"}
	res := readiness.Assemble(withRules(f, rule))
	if c := check(t, stored(t, res), "rules.r-short"); c.Status != readiness.Pass || c.Title != "No URL shorteners: none of the 3 links use one" {
		t.Fatalf("kept: %+v", c)
	}
	f.Content.Report.LinkList = append(f.Content.Report.LinkList, contentcheck.ContentCheckLink{URL: "https://bit.ly/3xK9tQ", Host: "bit.ly", Kind: "link", Probed: true, Status: 200})
	res = readiness.Assemble(withRules(f, rule))
	c := check(t, stored(t, res), "rules.r-short")
	if res.Verdict != readiness.VerdictBlocked || c.Status != readiness.Block || c.Title != "Workspace rule: no URL shorteners · 1 of 4 links point to bit.ly" ||
		c.Ack != "" || len(c.Raw) != 1 {
		t.Fatalf("broken: %s %+v", res.Verdict, c)
	}
}

// The subject's length and required text.
func TestRuleSubjectAndText(t *testing.T) {
	f := digest()
	long := readiness.RuleFacts{ID: "r-len", Name: "Subject under 30 characters", Kind: readiness.RuleSubjectLength, Operator: "at_most", Value: "30", Severity: "warn", AppliesTo: "broadcast"}
	c := check(t, stored(t, readiness.Assemble(withRules(f, long))), "rules.r-len")
	if c.Status != readiness.Warn || c.Title != "Workspace rule: subject under 30 characters · the subject has 40 characters, over 30" || len(c.Fixes) != 1 || c.Fixes[0].Kind != "link" {
		t.Fatalf("subject: %+v", c)
	}
	f.Content.FooterText = "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105 · Unsubscribe"
	address := readiness.RuleFacts{ID: "r-addr", Name: "Footer must include the postal address", Kind: readiness.RuleContainsText, Operator: "footer_contains",
		Value: "500 Market St", Severity: "block", AppliesTo: "both"}
	if c := check(t, stored(t, readiness.Assemble(withRules(f, address))), "rules.r-addr"); c.Status != readiness.Pass {
		t.Fatalf("address in the footer: %+v", c)
	}
	f.Content.FooterText = "Unsubscribe · Email preferences"
	if c := check(t, stored(t, readiness.Assemble(withRules(f, address))), "rules.r-addr"); c.Status != readiness.Block ||
		c.Title != "Workspace rule: footer must include the postal address · the footer doesn't show “500 Market St”" {
		t.Fatalf("address missing: %+v", c)
	}
}

// AddLinkParam tags the web links that miss the parameter, keeping
// fragments, entities, merge tags, mailto: and tagged links as they are.
func TestAddLinkParam(t *testing.T) {
	src := `<a href="https://acme.io/roadmap">a</a> <a href='https://acme.io/blog?ref=x&amp;y=1#top'>b</a> <a href="https://acme.io/?utm_campaign=x">c</a>` +
		`<a href="mailto:hi@acme.io">d</a> <a href="{{ unsubscribe_url }}">e</a> <mj-button href="https://docs.acme.io/flags">f</mj-button>`
	out, n := readiness.AddLinkParam(src, "utm_campaign", "weekly-digest-142")
	want := `<a href="https://acme.io/roadmap?utm_campaign=weekly-digest-142">a</a> <a href='https://acme.io/blog?ref=x&amp;y=1&amp;utm_campaign=weekly-digest-142#top'>b</a> ` +
		`<a href="https://acme.io/?utm_campaign=x">c</a><a href="mailto:hi@acme.io">d</a> <a href="{{ unsubscribe_url }}">e</a> ` +
		`<mj-button href="https://docs.acme.io/flags?utm_campaign=weekly-digest-142">f</mj-button>`
	if n != 3 || out != want {
		t.Fatalf("%d links:\n%s\nwant\n%s", n, out, want)
	}
}
