package placement_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/placement"
)

// A test's stored name compares its providers in English; each reader
// reads the comparison in their words, and the content's own name stays
// as it is, even when it says "vs".
func TestNameInVietnamese(t *testing.T) {
	vi, en := i18n.For(i18n.Vietnamese), i18n.For(i18n.English)
	cases := []struct{ name, got, want string }{
		{"compared", placement.NameIn(vi, "Weekly Digest #142 · SES vs Postal", "Weekly Digest #142"), "Weekly Digest #142 · SES so với Postal"},
		{"in English", placement.NameIn(en, "Weekly Digest #142 · SES vs Postal", "Weekly Digest #142"), "Weekly Digest #142 · SES vs Postal"},
		{"one provider", placement.NameIn(vi, "Password reset · Postmark", "Password reset"), "Password reset · Postmark"},
		{"the content says vs", placement.NameIn(vi, "Cats vs dogs · SES vs Postal", "Cats vs dogs"), "Cats vs dogs · SES so với Postal"},
		{"a name of its own", placement.NameIn(vi, "Q4 check", "Weekly Digest #142"), "Q4 check"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %q, want %q", c.name, c.got, c.want)
		}
	}
}
