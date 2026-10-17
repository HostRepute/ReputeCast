package i18n

import (
	"strings"
	"testing"
)

// Text a workspace writes in a language ReputeCast isn't translated into
// (the built-in labels of its public pages) is formatted with Format and
// FormatHTML, and checked with Arguments.

func TestFormatHTMLEscapesTextAndKeepsTags(t *testing.T) {
	got, err := FormatHTML(Lang{Locale: "es", Region: "es-ES"}, "Hola <b>{name}</b> & bienvenido. <link>Ver</link>",
		Args{"name": "Lucía <script>"}, Tags{"link": func(s string) string { return `<a href="/x">` + s + `</a>` }})
	if err != nil {
		t.Fatal(err)
	}
	if want := `Hola <b>Lucía &lt;script&gt;</b> &amp; bienvenido. <a href="/x">Ver</a>`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := FormatHTML(Default, "{broken", nil, nil); err == nil {
		t.Fatal("a broken message should fail")
	}
}

func TestArgumentsListsArgumentsAndTags(t *testing.T) {
	got, err := Arguments("{count, plural, one {# issue by {name}} other {# issues}} <link>more</link>")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "<link>,count,name" {
		t.Fatalf("got %v", got)
	}
}

func TestPluralRulesOfOtherLanguages(t *testing.T) {
	msg := "{n, plural, one {# день} few {# дня} many {# дней} other {# дня}}"
	for n, want := range map[int]string{1: "1 день", 3: "3 дня", 5: "5 дней", 21: "21 день"} {
		got, err := Format(Lang{Locale: "ru", Region: Standard}, msg, Args{"n": n})
		if err != nil || got != want {
			t.Fatalf("ru %d: got %q (%v), want %q", n, got, err, want)
		}
	}
	// The app languages keep their rules.
	if got, _ := Format(For(English), "{n, plural, one {# day} other {# days}}", Args{"n": 1}); got != "1 day" {
		t.Fatalf("en: %q", got)
	}
}
