package mcpserver

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// The accessibility summary an agent's call keeps is the accessibility
// package's own: the same English, and each language's words.
func TestAccessibilitySummaryKept(t *testing.T) {
	for _, r := range []accessibility.AccessibilityReport{
		{},
		{Issues: []accessibility.AccessibilityIssue{{}, {}, {}}, Serious: 1, Moderate: 2},
	} {
		kept := accessibilitySummary(r)
		if got := kept.In(i18n.Default); got != r.Summary() {
			t.Errorf("English: %q, want %q", got, r.Summary())
		}
		vi := i18n.For(i18n.Vietnamese)
		if got := kept.In(vi); got == "" || got != r.SummaryIn(vi) {
			t.Errorf("Vietnamese: %q, want %q", got, r.SummaryIn(vi))
		}
	}
}

// A list argument reads as each language writes a list, its kept messages
// in that language too.
func TestKeptListsInEveryLanguage(t *testing.T) {
	healthy := keep("outcome.healthy", i18n.Args{"names": []string{"SES", "Postal"}})
	if got := audit.Render(healthy, i18n.Default); got != "SES and Postal healthy" {
		t.Errorf("English: %q", got)
	}
	if got := audit.Render(healthy, i18n.For(i18n.Vietnamese)); got != "SES và Postal ổn định" {
		t.Errorf("Vietnamese: %q", got)
	}
	fields := keep("detail.changed", i18n.Args{"fields": []any{keep("field.subject", nil), keep("field.content", nil)}})
	if got := audit.Render(fields, i18n.Default); got != "subject and content" {
		t.Errorf("English: %q", got)
	}
	if got := audit.Render(fields, i18n.For(i18n.Vietnamese)); got != "tiêu đề và nội dung" {
		t.Errorf("Vietnamese: %q", got)
	}
}

// A proposed referral decision keeps the agent's reason, as before.
func TestReferralDecisionDetail(t *testing.T) {
	d := keep("detail.referralDecision", i18n.Args{"decision": "dont_pause", "referrer": "Bob", "hasReason": "yes", "reason": "same IP"})
	if got := d.In(i18n.Default); got != "proposed: not count, and pause the link of the held referrals of Bob · same IP" {
		t.Errorf("English: %q", got)
	}
	d = keep("detail.referralDecision", i18n.Args{"decision": "count", "referrer": "Bob", "hasReason": "no", "reason": ""})
	if got := d.In(i18n.For(i18n.Vietnamese)); got != "đã đề xuất: tính các lượt giới thiệu bị giữ của Bob" {
		t.Errorf("Vietnamese: %q", got)
	}
}
