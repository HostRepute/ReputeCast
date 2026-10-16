package languages

import (
	"strings"
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Spanish": "es", "español": "es", "es_MX": "es-MX", "ES": "es", "es-mx": "es-MX", "de-AT": "de-AT",
		"zh-hant-tw": "zh-Hant-TW", "pt_br": "pt-BR", "Tiếng Việt": "vi", "eng": "en", "": "", "es-419": "es-419",
	}
	for in, want := range cases {
		got, ok := Normalize(in)
		if !ok || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"klingon", "x-y", "es-MXX1", "123"} {
		if got, ok := Normalize(bad); ok {
			t.Errorf("Normalize(%q) = %q, want refused", bad, got)
		}
	}
	if Suggest("Spanosh") != "es" || Suggest("Germna") != "de" {
		t.Errorf("suggestions: %q %q", Suggest("Spanosh"), Suggest("Germna"))
	}
	if Base("es-MX") != "es" || Base("") != "" {
		t.Error("base")
	}
	if l, ok := Lookup("ar"); !ok || !l.RTL {
		t.Error("Arabic is right to left")
	}
}

// doc wraps sections in an MJML document.
func doc(sections ...string) string {
	return "<mjml><mj-head><mj-title>x</mj-title></mj-head><mj-body width=\"600px\">\n" + strings.Join(sections, "\n") + "\n</mj-body></mjml>"
}

func section(inner string) string {
	return "<mj-section><mj-column>" + inner + "</mj-column></mj-section>"
}

func text(s string) string { return "<mj-text>" + s + "</mj-text>" }

var (
	header   = `<!-- rc:header-logo --><mj-section><mj-column width="40%"><mj-image src="https://x/acme-logo.png" alt="Acme" width="96px"></mj-image></mj-column><mj-column width="60%"><mj-text><a href="https://acme.io/blog">Blog</a> · <a href="https://docs.acme.io">Docs</a></mj-text></mj-column></mj-section>`
	hero     = section(`<mj-image src="https://x/scheduled-rollouts.png" width="552px"></mj-image>`)
	headline = section(`<mj-text font-size="26px"><h1 style="margin:0">Scheduled rollouts are here</h1></mj-text>`)
	intro    = section(text(`Hi {{ first_name | default: "there" }}, you can now schedule flag changes ahead of time. <a href="https://docs.acme.io/rollouts">Click here</a> to see how it works.`))
	intro9   = section(text(`Hi {{ first_name | default: "there" }}, you can now schedule flag changes ahead of time, in any timezone. <a href="https://docs.acme.io/rollouts">Click here</a> to see how it works.`))
	button   = section(`<mj-button href="https://docs.acme.io/migrate-v2" background-color="#5EBFB0">Plan your migration</mj-button>`)
	footer   = `<!-- rc:footer --><mj-section><mj-column><mj-text font-size="11px">Acme Inc · 500 Market St<br><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>`
	introDE  = section(text(`Hallo {{ first_name | default: "zusammen" }}, Sie können Flag-Änderungen jetzt im Voraus planen. <a href="https://docs.acme.io/rollouts">Hier klicken</a>.`))
	headDE   = section(`<mj-text font-size="26px"><h1 style="margin:0">Geplante Rollouts sind da</h1></mj-text>`)
	buttonDE = section(`<mj-button href="https://docs.acme.io/migrate-v2" background-color="#5EBFB0">Migration planen</mj-button>`)
	footerDE = `<!-- rc:footer --><mj-section><mj-column><mj-text font-size="11px">Acme Inc · 500 Market St<br><a href="{{ unsubscribe_url }}">Abmelden</a></mj-text></mj-column></mj-section>`
)

func TestBlocks(t *testing.T) {
	bs := Blocks(doc(header, hero, headline, intro, button, footer), "")
	if len(bs) != 6 {
		t.Fatalf("blocks: %d", len(bs))
	}
	names := []string{}
	for _, b := range bs {
		names = append(names, b.Name)
	}
	want := []string{"Header · logo and links", "Hero image", "Headline · Heading 1", "Intro", "Button", "Footer"}
	if strings.Join(names, "|") != strings.Join(want, "|") {
		t.Fatalf("names: %q", names)
	}
	if bs[0].Text != `Logo, alt "Acme" · Blog · Docs` {
		t.Errorf("header text: %q", bs[0].Text)
	}
	if bs[1].Text != "scheduled-rollouts.png · no alt text" {
		t.Errorf("hero text: %q", bs[1].Text)
	}
	if !strings.Contains(bs[3].Text, `{{ first_name | default: "there" }}`) {
		t.Errorf("tokens kept: %q", bs[3].Text)
	}
	// A style change keeps the translatable fingerprint.
	styled := Blocks(doc(strings.Replace(button, "#5EBFB0", "#0F766E", 1)), "")[0]
	if styled.Hash != bs[4].Hash || styled.Raw == bs[4].Raw {
		t.Error("style changes keep the text hash and change the raw hash")
	}
	// HTML-only content is one block.
	if h := Blocks("", "<html><body><p>Hello</p></body></html>"); len(h) != 1 || h[0].Text != "Hello" {
		t.Errorf("html block: %+v", h)
	}
}

func TestAlign(t *testing.T) {
	k := func(raw, hash string) AlignKey { return AlignKey{Raw: raw, Hash: hash} }
	before := []AlignKey{k("a", "A"), k("b", "B"), k("c", "C"), k("d", "D")}
	// b edited in place, a new block after c, d moved first.
	after := []AlignKey{k("d", "D"), k("a", "A"), k("b2", "B2"), k("c", "C"), k("e", "E")}
	got := Align(before, after)
	want := []int{3, 0, 1, 2, -1}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("align = %v, want %v", got, want)
		}
	}
	// A style-only change matches by text before positions do.
	got = Align([]AlignKey{k("x", "X"), k("y", "Y")}, []AlignKey{k("y2", "Y"), k("x", "X")})
	if got[0] != 1 || got[1] != 0 {
		t.Fatalf("text match: %v", got)
	}
}

// translation sets up a source and a German version translated from it.
func translation(t *testing.T, sourceV6, german, sourceV9 string) ([]Row, Progress) {
	t.Helper()
	src6 := Blocks(sourceV6, "")
	keys, next, _ := SyncSource(nil, src6, 1, BlockChange{Version: 6})
	de := Blocks(german, "")
	links := FreshLinks(src6, keys, de, "Subject", "Preheader", 6)
	src9 := Blocks(sourceV9, "")
	keys9, _, changed := SyncSource(keys, src9, next, BlockChange{Version: 9, By: "Sofia Rossi", At: time.Date(2026, 10, 15, 9, 12, 0, 0, time.UTC)})
	_ = changed
	return Compare(src9, keys9, de, links, false, HashText("Subject"), HashText("Preheader"))
}

func TestOutdatedPerBlock(t *testing.T) {
	v6 := doc(header, hero, headline, intro, button, footer)
	de := doc(header, hero, headDE, introDE, buttonDE, footerDE)
	v9 := doc(header, hero, headline, intro9, button, footer)
	rows, p := translation(t, v6, de, v9)
	if p.Status != StatusOutdated || p.Attention != 1 || p.Changed != 1 || p.Blocks != 6 || p.UpToDate != 5 {
		t.Fatalf("progress: %+v", p)
	}
	r := rows[3]
	if r.State != RowChanged || r.Number != 4 || r.Changed == nil || r.Changed.Version != 9 || r.Changed.By != "Sofia Rossi" || r.FromVersion != 6 {
		t.Fatalf("intro row: %+v", r)
	}
	if !strings.Contains(r.Before, "ahead of time.") || strings.Contains(r.Before, "timezone") {
		t.Errorf("diff's old side: %q", r.Before)
	}
	// Up to date again once marked.
	src6 := Blocks(v6, "")
	keys, next, _ := SyncSource(nil, src6, 1, BlockChange{Version: 6})
	deBlocks := Blocks(de, "")
	links := FreshLinks(src6, keys, deBlocks, "Subject", "Preheader", 6)
	src9 := Blocks(v9, "")
	keys9, _, _ := SyncSource(keys, src9, next, BlockChange{Version: 9})
	links = MarkUpToDate(links, src9, keys9, "Subject", "Preheader", 9, []string{keys9[3].Key})
	_, p = Compare(src9, keys9, deBlocks, links, false, HashText("Subject"), HashText("Preheader"))
	if p.Status != StatusUpToDate {
		t.Fatalf("after marking the block: %+v", p)
	}
	// A subject change outdates the version too.
	_, p = Compare(src9, keys9, deBlocks, links, false, HashText("New subject"), HashText("Preheader"))
	if p.Status != StatusOutdated || !p.SubjectChanged {
		t.Fatalf("subject change: %+v", p)
	}
}

func TestNewAndRemovedBlocks(t *testing.T) {
	office := section(text("Join our office hours on Thursday at 17:00 CET."))
	waitlist := section(text("Join the beta waitlist."))
	waitlistDE := section(text("Melden Sie sich für die Beta-Warteliste an."))
	v6 := doc(header, headline, intro, waitlist, footer)
	de := doc(header, headDE, introDE, waitlistDE, footerDE)
	v9 := doc(header, headline, intro, office, footer)
	rows, p := translation(t, v6, de, v9)
	states := []string{}
	for _, r := range rows {
		states = append(states, r.State)
	}
	// The waitlist block was edited into office hours in place? No: its text
	// changed completely at the same position, which reads as an edit.
	if p.Blocks != 5 {
		t.Fatalf("blocks: %+v %v", p, states)
	}
	// Removing a block and adding one elsewhere.
	v9b := doc(header, headline, intro, footer, office)
	rows, p = translation(t, v6, de, v9b)
	states = states[:0]
	for _, r := range rows {
		states = append(states, r.State)
	}
	want := "up_to_date,up_to_date,up_to_date,removed,up_to_date,new"
	if strings.Join(states, ",") != want {
		t.Fatalf("states: %v, want %s", states, want)
	}
	if p.Attention != 2 || p.Status != StatusOutdated {
		t.Fatalf("progress: %+v", p)
	}
}

func TestDraftStillInSource(t *testing.T) {
	v9 := doc(header, hero, headline, intro, button, footer)
	src := Blocks(v9, "")
	keys, _, _ := SyncSource(nil, src, 1, BlockChange{Version: 9})
	// A draft copied from the source with two blocks translated.
	vi := doc(header, hero, section(`<mj-text font-size="26px"><h1 style="margin:0">Triển khai theo lịch đã ra mắt</h1></mj-text>`), introDE, button, footer)
	vb := Blocks(vi, "")
	links := FreshLinks(src, keys, vb, "S", "P", 9)
	_, p := Compare(src, keys, vb, links, true, HashText("S"), HashText("P"))
	// Header, button and footer still say what English says; the hero has
	// nothing to translate.
	if p.Status != StatusDraft || p.Still != 3 || p.Translated != 3 {
		t.Fatalf("draft progress: %+v", p)
	}
}

func TestChooser(t *testing.T) {
	st := Settings{Default: "en", Fallback: "en", Languages: []string{"en", "es", "de", "vi"}, offered: map[string]bool{"en": true, "es": true, "de": true, "vi": true}}
	ch := NewChooser("en", st, []VersionState{{Language: "es", Status: "translated"}, {Language: "de", Status: "translated", Held: false}, {Language: "vi", Status: "draft"}})
	cases := []struct {
		contact, lang, reason string
	}{
		{"es-MX", "es", ReasonMatch},
		{"de-AT", "de", ReasonMatch},
		{"en", "en", ReasonMatch},
		{"", "en", ReasonNoLanguage},
		{"pt-BR", "en", ReasonNotOffered},
		{"vi", "en", ReasonDraft},
	}
	for _, c := range cases {
		got := ch.Pick(c.contact)
		if got.Language != c.lang || got.Reason != c.reason {
			t.Errorf("Pick(%q) = %+v, want %s %s", c.contact, got, c.lang, c.reason)
		}
	}
	// Missing: no version at all; held: readers get the fallback.
	ch = NewChooser("en", st, []VersionState{{Language: "de", Status: "translated", Held: true}})
	if got := ch.Pick("es"); got.Reason != ReasonMissing || got.Language != "en" {
		t.Errorf("missing: %+v", got)
	}
	if got := ch.Pick("de"); got.Reason != ReasonHeld || got.Language != "en" || !got.Fallback() {
		t.Errorf("held: %+v", got)
	}
	// A fallback that is ready is used before the source.
	st.Fallback = "es"
	ch = NewChooser("en", st, []VersionState{{Language: "es", Status: "translated"}})
	if got := ch.Pick(""); got.Language != "es" {
		t.Errorf("fallback first: %+v", got)
	}
	// A removed language is never sent.
	st2 := Settings{Default: "en", Fallback: "en", Languages: []string{"en"}, offered: map[string]bool{"en": true}}
	ch = NewChooser("en", st2, []VersionState{{Language: "de", Status: "translated"}})
	if got := ch.Pick("de"); got.Language != "en" || got.Reason != ReasonNotOffered {
		t.Errorf("removed language: %+v", got)
	}
	// An explicit request.
	st.Fallback = "en"
	ch = NewChooser("en", st, []VersionState{{Language: "es", Status: "translated"}})
	if got := ch.Request("pt-BR"); got.Language != "en" || got.Reason != ReasonRequested {
		t.Errorf("request: %+v", got)
	}
}

func TestReplaceInBlock(t *testing.T) {
	src := doc(intro, button)
	bs := Blocks(src, "")
	out, n := replaceInBlock(src, bs, []int{0}, "https://docs.acme.io/rollouts", "https://docs.acme.io/scheduled")
	if n != 1 || !strings.Contains(out, "https://docs.acme.io/scheduled") {
		t.Fatalf("replace: %d", n)
	}
	if _, n := replaceInBlock(src, bs, []int{1}, "https://docs.acme.io/rollouts", "x"); n != 0 {
		t.Fatal("only the given blocks change")
	}
}

func TestEmptyTexts(t *testing.T) {
	out := emptyTexts(doc(intro, button))
	if strings.Contains(out, "schedule flag changes") || !strings.Contains(out, "Plan your migration") {
		t.Fatalf("layout only: %s", out)
	}
}
