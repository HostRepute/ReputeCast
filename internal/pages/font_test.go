package pages

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The theme note names fonts by their names, and the system font and a
// serif in the reader's words.
func TestFontNameInEachLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	for _, c := range []struct{ got, want string }{
		{fontName(i18n.Default, "plex", ""), "IBM Plex Sans"},
		{fontName(vi, "inter", ""), "Inter"},
		{fontName(i18n.Default, "system", ""), "the system font"},
		{fontName(vi, "system", ""), "phông chữ của hệ thống"},
		{fontName(i18n.Default, "serif", ""), "a serif"},
		{fontName(vi, "custom", "Literata"), "Literata"},
	} {
		if c.got != c.want {
			t.Errorf("%q, want %q", c.got, c.want)
		}
	}
}

// An address from a name in another language keeps its letters, without
// their marks; letters with no Latin form are dropped.
func TestSlugifyFoldsMarks(t *testing.T) {
	for in, want := range map[string]string{"Trang của tôi": "trang-cua-toi", "Đăng ký sự kiện": "dang-ky-su-kien", "Straße & Ærø": "strasse-aero",
		"Warm-up planner": "warm-up-planner", "中文 page": "page"} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
