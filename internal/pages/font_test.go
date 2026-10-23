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
