package engine

import (
	"testing"
)

func TestLanguageContent(t *testing.T) {
	c := sendContent{
		Subject: "Hello", HTML: `<p>Hi <a href="https://acme.io/en">docs</a></p>`, Links: map[string]int{}, Tracking: Tracking{Opens: true, Clicks: true},
		Variants: []abVariant{{Key: "A"}, {Key: "B", Subject: "Hello B"}},
		Source:   "en", DefaultLanguage: "en", Fallback: "en", Offered: []string{"en", "de", "es", "vi"},
		Languages: []langContent{
			{Language: "de", Status: "translated", Subject: "Hallo", HTML: `<p>Hallo <a href="https://acme.io/de">Doku</a> <a href="https://acme.io/de2">mehr</a></p>`,
				Variants: []abVariant{{Key: "B", Subject: "Hallo B"}}},
			{Language: "es", Status: "translated", Subject: "Hola", HTML: `<p>Hola <a href="https://acme.io/es">docs</a></p>`},
			{Language: "vi", Status: "draft"},
		},
	}
	if got := c.forVariant("de|B").Subject; got != "Hallo B" {
		t.Errorf("German B: %q", got)
	}
	// Spanish has no B subject of its own: it keeps its own, never English.
	if got := c.forVariant("es|B").Subject; got != "Hola" {
		t.Errorf("Spanish B: %q", got)
	}
	if got := c.forVariant("B").Subject; got != "Hello B" {
		t.Errorf("source B: %q", got)
	}
	if got := renderKey(c, "vi", ""); got != "" {
		t.Errorf("a draft renders the source: %q", got)
	}
	if got := renderKey(c, "de", "B"); got != "de|B" {
		t.Errorf("German B key: %q", got)
	}
	links, err := (&Engine{}).indexLinks(c)
	if err != nil {
		t.Fatal(err)
	}
	// Source 1 link, German 2, Spanish 1: positions follow each other.
	if len(links) != 4 || c.Links["de|"] != 1 || c.Links["es|"] != 3 {
		t.Fatalf("links %d, offsets %v", len(links), c.Links)
	}
	if got := languageContentKey(c, "de", "B"); got != "de|" {
		t.Errorf("German B uses German's links: %q", got)
	}
	ch, ok := c.chooser()
	if !ok {
		t.Fatal("chooser")
	}
	for contact, want := range map[string]messageLanguage{
		"de-AT": {Language: "de", Contact: "de", Reason: "match"},
		"vi":    {Language: "en", Contact: "vi", Reason: "draft"},
		"":      {Language: "en", Contact: "", Reason: "no_language"},
		"pt-BR": {Language: "en", Contact: "pt", Reason: "not_offered"},
	} {
		if got := c.pick(ch, ok, contact); got != want {
			t.Errorf("pick(%q) = %+v, want %+v", contact, got, want)
		}
	}
	// Without content languages nothing is recorded.
	if got := (sendContent{}).pick(ch, false, "de"); got != (messageLanguage{}) {
		t.Errorf("no languages: %+v", got)
	}
}
