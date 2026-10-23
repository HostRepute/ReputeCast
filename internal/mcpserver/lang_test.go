package mcpserver

import (
	"encoding/json"
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

// A language version's language is kept as its code and named in each
// reader's language, as stored and as read back.
func TestLanguageKeptAsItsCode(t *testing.T) {
	d := keep("detail.languageChanged", i18n.Args{"language": audit.LanguageArg("es"), "fields": []any{keep("field.subject", nil)}})
	raw, _ := json.Marshal(d)
	var back i18n.Text
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	for _, kept := range []i18n.Text{d, back} {
		if got := audit.Render(kept, i18n.Default); got != "Spanish: subject" {
			t.Errorf("English: %q", got)
		}
		if got := audit.Render(kept, i18n.For(i18n.Vietnamese)); got != "Tiếng Tây Ban Nha: tiêu đề" {
			t.Errorf("Vietnamese: %q", got)
		}
	}
}
