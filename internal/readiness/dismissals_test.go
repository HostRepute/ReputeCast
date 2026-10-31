package readiness_test

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// CO-14: a warning the composer also shows keeps the keys of the composer's
// warnings it is made of; when someone dismissed all of them there, it is
// marked with who and why, stays a warning, and counts as acknowledged at
// send time. One of two dismissed isn't enough.
func TestDismissedInTheComposer(t *testing.T) {
	f := digest()
	short := func(path int, url string) contentcheck.ContentCheckIssue {
		return contentcheck.ContentCheckIssue{ID: "links.shortener", Severity: contentcheck.Warn, Group: "Links", Title: "Link goes through a URL shortener",
			Path: []int{0, path}, Section: 0, URL: url}
	}
	a, b := short(1, "https://bit.ly/x"), short(2, "https://bit.ly/y")
	f.Content.Report.Issues = append(f.Content.Report.Issues, a, b)
	run := readiness.ReadinessRun{ID: "run-1", RanAt: now, ReadinessResult: stored(t, readiness.Assemble(f))}
	keyA, keyB := campaigns.IssueKey(a, f.Campaign.Subject), campaigns.IssueKey(b, f.Campaign.Subject)
	var c readiness.ReadinessCheck
	for _, x := range run.Checks {
		if x.Key == "links.shortener" {
			c = x
		}
	}
	if c.Status != readiness.Warn || len(c.IssueKeys) != 2 || c.IssueKeys[0] != keyA || c.IssueKeys[1] != keyB {
		t.Fatalf("issue keys: %+v", c)
	}
	// A warning made of nothing the composer shows has no keys.
	for _, x := range run.Checks {
		if x.Key == "audience.unengaged" && len(x.IssueKeys) > 0 {
			t.Errorf("unengaged: %v", x.IssueKeys)
		}
	}

	linh := campaigns.ComposerDismissal{ID: "d1", Key: keyA, CheckID: "links.shortener", Reason: "wrong", ReasonText: "the check is wrong here",
		Note: "Our own short domain", By: "Linh Nguyen", At: now.Add(-time.Hour)}
	readiness.MarkDismissed(&run, map[string]campaigns.ComposerDismissal{keyA: linh})
	if d := find(run, "links.shortener").Dismissal; d != nil {
		t.Fatalf("one of two dismissed: %+v", d)
	}
	marcus := linh
	marcus.ID, marcus.Key, marcus.By, marcus.At = "d2", keyB, "Marcus Chen", now
	readiness.MarkDismissed(&run, map[string]campaigns.ComposerDismissal{keyA: linh, keyB: marcus})
	d := find(run, "links.shortener").Dismissal
	if d == nil || d.ID != "d2" || d.By != "Marcus Chen" || d.ReasonText != "the check is wrong here" || d.Note != "Our own short domain" || d.Count != 2 {
		t.Fatalf("both dismissed: %+v", d)
	}
	if keys := readiness.DismissedKeys(run); len(keys) != 1 || keys[0] != "links.shortener" {
		t.Errorf("dismissed keys: %v", keys)
	}
	if find(run, "links.shortener").Status != readiness.Warn {
		t.Error("a dismissed warning is still a warning")
	}

	// At send time it is acknowledged, though nobody ticked it.
	_, conf, digests := confirmedRun(t)
	if ch := readiness.CheckSendTime(run, conf, digests); ch.Held() {
		t.Fatalf("held on a dismissed warning: %+v", ch.Holds)
	}
	readiness.MarkDismissed(&run, nil)
	if ch := readiness.CheckSendTime(run, conf, digests); !ch.Held() {
		t.Fatal("undone: the warning nobody confirmed holds the send")
	}
}

// When the Accessibility group runs, an image's missing alt text is its
// check: the composer's alt-text warning for the same element is the one
// a dismissal names.
func TestDismissedAltTextInAccessibility(t *testing.T) {
	f := digest()
	alt := contentcheck.ContentCheckIssue{ID: "content.alt_text", Severity: contentcheck.Warn, Group: "Content", Title: "Hero image has no alt text", Path: []int{1, 0, 0}}
	f.Content.Report.Issues = append(f.Content.Report.Issues, alt)
	f.Accessibility = accessibility.ReadinessFacts{Ran: true, Checked: true, Report: accessibility.AccessibilityReport{Policy: "warn", Serious: 1,
		Issues: []accessibility.AccessibilityIssue{{ID: "alt_text:1-0-0:0", Kind: accessibility.KindAltText, Severity: "serious", Title: "The hero image has no alt text",
			Short: "the hero image's alt text", Name: "The hero image", Languages: []string{"en"}, Path: []int{1, 0, 0}}}}}
	run := readiness.ReadinessRun{ID: "run-1", RanAt: now, ReadinessResult: stored(t, readiness.Assemble(f))}
	c := find(run, "accessibility.alt_text:1-0-0:0")
	if c.Status != readiness.Warn || len(c.IssueKeys) != 1 || c.IssueKeys[0] != campaigns.IssueKey(alt, f.Campaign.Subject) {
		t.Fatalf("alt text: %+v", c)
	}
	readiness.MarkDismissed(&run, map[string]campaigns.ComposerDismissal{c.IssueKeys[0]: {ID: "d1", Reason: "decorative", ReasonText: "the image is decorative", By: "Linh Nguyen", At: now}})
	if d := find(run, "accessibility.alt_text:1-0-0:0").Dismissal; d == nil || d.ReasonText != "the image is decorative" {
		t.Fatalf("dismissed: %+v", d)
	}
	// Under a Block policy nothing is dismissed: blocks never are.
	f.Accessibility.Report.Policy = "block"
	if c := find(readiness.ReadinessRun{ReadinessResult: readiness.Assemble(f)}, "accessibility.alt_text:1-0-0:0"); c.Status != readiness.Block || len(c.IssueKeys) != 0 {
		t.Fatalf("block: %+v", c)
	}
}

func find(run readiness.ReadinessRun, key string) readiness.ReadinessCheck {
	for _, c := range run.Checks {
		if c.Key == key {
			return c
		}
	}
	return readiness.ReadinessCheck{}
}
