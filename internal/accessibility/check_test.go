package accessibility

import (
	"os"
	"strings"
	"testing"
)

func digest(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/digest143.mjml")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func check(t *testing.T, vs []Version, opts Options) AccessibilityReport {
	t.Helper()
	r, err := Check(vs, opts)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	return r
}

func kinds(r AccessibilityReport) []string {
	out := []string{}
	for _, is := range r.Issues {
		out = append(out, is.Kind)
	}
	return out
}

func find(r AccessibilityReport, kind string) *AccessibilityIssue {
	for i := range r.Issues {
		if r.Issues[i].Kind == kind {
			return &r.Issues[i]
		}
	}
	return nil
}

// Weekly Digest #143 has the brief's six issues, listed once for its three
// languages, serious first.
func TestDigestIssues(t *testing.T) {
	src := digest(t)
	es := strings.Replace(src, ">Click here<", ">Haz clic aquí<", 1)
	de := strings.Replace(src, ">Click here<", ">Hier klicken<", 1)
	r := check(t, []Version{{Lang: "es", MJML: es}, {Lang: "en", MJML: src, Source: true}, {Lang: "de", MJML: de}},
		Options{Titles: map[string]PageTitle{"https://docs.acme.io/scheduled-rollouts": {Title: "Scheduled rollouts · Acme Docs", Lang: "en"}}})
	want := []string{KindAltText, KindContrast, KindLinkText, KindLinkText, KindHeading, KindTextSize}
	if got := kinds(r); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("issues: %v, want %v", got, want)
	}
	if r.Serious != 2 || r.Moderate != 4 || r.Summary() != "6 issues: 2 serious, 4 moderate" {
		t.Fatalf("counts: %d serious, %d moderate (%s)", r.Serious, r.Moderate, r.Summary())
	}
	if r.Languages[0].Code != "en" || !r.Languages[0].Source {
		t.Fatalf("the source is listed first: %+v", r.Languages)
	}
	for _, is := range r.Issues {
		if strings.Join(is.Languages, ",") != "en,es,de" {
			t.Errorf("%s in %v, want en,es,de", is.ID, is.Languages)
		}
	}
	alt := r.Issues[0]
	if alt.Title != "The hero image has no alt text" || strings.Join(alt.Location, " › ") != "mjml › mj-body › mj-section · hero › mj-image" || !alt.Fix.Decorative {
		t.Errorf("alt: %+v", alt)
	}
	ct := r.Issues[1]
	if ct.Title != "“Plan your migration”: white text on #5EBFB0 is 2.1:1. It needs 4.5:1." || ct.Fix.Target != "background" {
		t.Errorf("contrast: %s %+v", ct.Title, ct.Fix)
	}
	var sugg []string
	for _, s := range ct.Fix.Suggestions {
		sugg = append(sugg, s.Color+" "+s.Reason+" "+FormatRatio(s.Ratio))
	}
	if strings.Join(sugg, " | ") != "#0F766E other_buttons 5.4:1 | #348377 nearest_aa 4.5:1 | #276259 aaa 7.0:1" {
		t.Errorf("suggestions: %v", sugg)
	}
	click := r.Issues[2]
	if click.Title != "“Click here” doesn't say where the link goes" || strings.Join(click.Location, " › ") != "mjml › mj-body › mj-section · intro › a" {
		t.Errorf("click here: %s %v", click.Title, click.Location)
	}
	if click.Versions[0].Suggestion != "Scheduled rollouts · Acme Docs" || click.Versions[1].Suggestion != "" || click.Versions[1].Text != "Haz clic aquí" {
		t.Errorf("link text per language: %+v", click.Versions)
	}
	if click.Detail != "Also “Haz clic aquí” and “Hier klicken”. The linked page is in English, so Spanish and German get a field to write the words." {
		t.Errorf("detail: %s", click.Detail)
	}
	if bare := r.Issues[3]; bare.Title != "A bare URL is the link text: https://docs.acme.io/migrate-v2" {
		t.Errorf("bare URL: %s", bare.Title)
	}
	h := r.Issues[4]
	if h.Title != "Both card titles are Heading 3, right after the Heading 1" || len(h.Fix.Headings) != 2 || h.Fix.Headings[1].Text != "Faster CSV exports" ||
		strings.Join(h.Location, " › ") != "mjml › mj-body › mj-section · cards › h3 × 2" {
		t.Errorf("headings: %s %+v %v", h.Title, h.Fix.Headings, h.Location)
	}
	if s := r.Issues[5]; s.Title != "Footer text is 11 px. The minimum is 12 px." || s.Fix.ToPx != 12 || s.Fix.FromPx != 11 {
		t.Errorf("size: %s %+v", s.Title, s.Fix)
	}
	c := r.Counts
	if c.Images != 7 || c.ImagesOK != 6 || c.Links != 15 || c.LinksOK != 13 || c.RoleAdded != 1 || strings.Join(c.Languages, ",") != "en,es,de" || c.Dir != "ltr" {
		t.Errorf("counts: %+v", c)
	}
}

func TestPolicyAndMinimumSize(t *testing.T) {
	r := check(t, []Version{{Lang: "en", MJML: digest(t)}}, Options{Policy: PolicyBlock, MinTextPx: 14})
	if r.Policy != PolicyBlock || r.MinTextPx != 14 {
		t.Fatalf("policy: %s %d", r.Policy, r.MinTextPx)
	}
	// At 14 px the 12 px header links and the 13 px buttons are too small as well.
	n := 0
	for _, is := range r.Issues {
		if is.Kind == KindTextSize {
			n++
		}
	}
	if n < 3 {
		t.Errorf("text size issues at 14 px: %d", n)
	}
	if r := check(t, []Version{{Lang: "en", MJML: digest(t)}}, Options{MinTextPx: 40}); r.MinTextPx != DefaultMinTextPx {
		t.Errorf("out of range minimum falls back to 12 px: %d", r.MinTextPx)
	}
}

func mj(body string) string {
	return `<mjml><mj-head><mj-attributes><mj-all font-family="Arial"></mj-all></mj-attributes></mj-head><mj-body background-color="#FFFFFF">` + body + `</mj-body></mjml>`
}

// Contrast compares the text color with the nearest background: the
// element's, then the column's, the section's, the wrapper's, the body's.
func TestContrastNearestBackground(t *testing.T) {
	// #6B6B6B passes on white (5.3:1) and fails on the dark greys.
	text := `<mj-text color="#6B6B6B" font-size="15px">Grey text</mj-text>`
	cases := []struct {
		name string
		body string
		bad  bool
	}{
		{"body white", `<mj-section><mj-column>` + text + `</mj-column></mj-section>`, false},
		{"body dark", strings.Replace(`<mj-section><mj-column>`+text+`</mj-column></mj-section>`, "", "", 1), false},
		{"section dark", `<mj-section background-color="#555555"><mj-column>` + text + `</mj-column></mj-section>`, true},
		{"column light over dark section", `<mj-section background-color="#222222"><mj-column background-color="#FFFFFF">` + text + `</mj-column></mj-section>`, false},
		{"wrapper dark", `<mj-wrapper background-color="#555555"><mj-section><mj-column>` + text + `</mj-column></mj-section></mj-wrapper>`, true},
		{"element background", `<mj-section><mj-column><mj-text color="#6B6B6B" container-background-color="#666666">Grey on grey</mj-text></mj-column></mj-section>`, true},
		{"inline span", `<mj-section><mj-column><mj-text color="#111111">Fine <span style="color:#BBBBBB">pale</span></mj-text></mj-column></mj-section>`, true},
		{"inline background", `<mj-section><mj-column><mj-text color="#111111"><span style="background:#333333">dark mark</span></mj-text></mj-column></mj-section>`, true},
	}
	for _, c := range cases {
		src := mj(c.body)
		if c.name == "body dark" {
			src = strings.Replace(src, `<mj-body background-color="#FFFFFF">`, `<mj-body background-color="#333333">`, 1)
			c.bad = true
		}
		r := check(t, []Version{{Lang: "en", MJML: src}}, Options{})
		got := find(r, KindContrast) != nil
		if got != c.bad {
			t.Errorf("%s: contrast issue %v, want %v (%v)", c.name, got, c.bad, kinds(r))
		}
	}
	// Grey #777777 on white is 4.47:1: it fails AA and shows as 4.4:1.
	r := check(t, []Version{{Lang: "en", MJML: mj(`<mj-section><mj-column><mj-text color="#777777" font-size="15px">Almost</mj-text></mj-column></mj-section>`)}}, Options{})
	if is := find(r, KindContrast); is == nil || !strings.Contains(is.Title, "is 4.4:1") || is.Fix.Target != "text" {
		t.Fatalf("777 on white: %+v", is)
	}
}

// Large text needs 3:1.
func TestContrastLargeText(t *testing.T) {
	body := `<mj-section><mj-column><mj-text color="#2B9A8C" font-size="26px" font-weight="700">Headline</mj-text></mj-column></mj-section>`
	if r := check(t, []Version{{Lang: "en", MJML: mj(body)}}, Options{}); find(r, KindContrast) != nil {
		t.Fatalf("26 px bold at 3.4:1 passes: %v", kinds(r))
	}
	small := strings.Replace(body, "26px", "15px", 1)
	if r := check(t, []Version{{Lang: "en", MJML: mj(small)}}, Options{}); find(r, KindContrast) == nil {
		t.Fatalf("15 px at 3.4:1 fails")
	}
}

// Over a background image contrast can't be checked: no issue, counted.
func TestContrastOverImage(t *testing.T) {
	hero := `<mj-hero mode="fixed-height" height="200px" background-url="https://x.test/b.png" background-color="#FFFFFF"><mj-text color="#EEEEEE">Over the image</mj-text></mj-hero>`
	section := `<mj-section background-url="https://x.test/b.png"><mj-column><mj-text color="#EEEEEE">Also over an image</mj-text></mj-column></mj-section>`
	r := check(t, []Version{{Lang: "en", MJML: mj(hero + section)}}, Options{})
	if find(r, KindContrast) != nil {
		t.Fatalf("text over images must not count: %v", kinds(r))
	}
	if r.Counts.OverImage != 2 {
		t.Fatalf("over image: %d", r.Counts.OverImage)
	}
	// A solid background between the text and the image is checked again.
	solid := `<mj-section background-url="https://x.test/b.png"><mj-column background-color="#FFFFFF"><mj-text color="#EEEEEE">On white</mj-text></mj-column></mj-section>`
	if r := check(t, []Version{{Lang: "en", MJML: mj(solid)}}, Options{}); find(r, KindContrast) == nil {
		t.Fatalf("pale text on a white column is checked")
	}
}

// A linked image without alt text is one issue (the link has no text), not two.
func TestLinkedImage(t *testing.T) {
	body := `<mj-section><mj-column><mj-image src="https://x.test/logo.png" href="https://acme.io" width="96px"></mj-image><mj-image src="https://x.test/hero.png"></mj-image><mj-image src="https://x.test/spacer.png" alt=""></mj-image></mj-column></mj-section>`
	r := check(t, []Version{{Lang: "en", MJML: mj(body)}}, Options{})
	if got := strings.Join(kinds(r), ","); got != "alt_text,link_no_text" {
		t.Fatalf("issues: %s", got)
	}
	linked := find(r, KindLinkNoText)
	if linked.Element.Role != "linked_image" || linked.Fix.Kind != KindAltText || linked.Fix.Decorative || !strings.HasPrefix(linked.Title, "A linked image") {
		t.Errorf("linked image: %+v", linked)
	}
	if alt := find(r, KindAltText); !alt.Fix.Decorative {
		t.Errorf("an image that isn't linked can be decorative")
	}
	// alt="" is a decorative image: it counts as handled.
	if r.Counts.Images != 3 || r.Counts.ImagesOK != 1 {
		t.Errorf("images: %+v", r.Counts)
	}
}

func TestLinksAndButtonsWithoutText(t *testing.T) {
	body := `<mj-section><mj-column><mj-button href="https://acme.io/go" background-color="#0F766E" color="#FFFFFF"></mj-button><mj-text color="#111111">See <a href="https://acme.io/x"></a> and <a href="https://acme.io/y" aria-label="Pricing"></a></mj-text></mj-column></mj-section>`
	r := check(t, []Version{{Lang: "en", MJML: mj(body)}}, Options{})
	n := 0
	for _, is := range r.Issues {
		if is.Kind == KindLinkNoText {
			n++
			if is.Severity != Serious || is.Fix.Kind != KindLinkText {
				t.Errorf("no text: %+v", is)
			}
		}
	}
	if n != 2 {
		t.Fatalf("a button and a link have no text (aria-label names the third): %v", kinds(r))
	}
}

// Social icons next to their text link to the same address aren't empty links.
func TestSocialIconsWithText(t *testing.T) {
	body := `<mj-section><mj-column><mj-social><mj-social-element name="github" href="https://github.com/acme" src="https://x.test/gh.png">GitHub</mj-social-element></mj-social></mj-column></mj-section>`
	r := check(t, []Version{{Lang: "en", MJML: mj(body)}}, Options{})
	if len(r.Issues) != 0 {
		t.Fatalf("social icon with a text label: %v", r.Issues)
	}
}

func TestPhrasesPerLanguage(t *testing.T) {
	cases := []struct {
		lang, text string
		vague      bool
	}{
		{"en", "Click here", true}, {"en", "read more →", true}, {"en", "Learn more.", true}, {"en", "Read the migration guide", false},
		{"es", "Haz clic aquí", true}, {"es-MX", "Leer más", true}, {"es", "Guía de migración", false},
		{"de", "Hier klicken", true}, {"de-AT", "Weiterlesen", true}, {"de", "Zur Migrationsanleitung", false},
		{"fr", "Cliquez ici", true}, {"fr", "En savoir plus", true},
		{"pt-BR", "Clique aqui", true}, {"pt", "Saiba mais", true},
		{"vi", "Bấm vào đây", true}, {"vi", "Xem thêm", true}, {"vi", "Hướng dẫn chuyển đổi", false},
		// Decomposed Vietnamese (as some editors paste it) is composed first.
		{"vi", "Bấm vào đây", true},
		// Languages without a list only get the bare-URL and no-text checks.
		{"it", "Clicca qui", false},
		// Another language's phrase isn't flagged.
		{"en", "Hier klicken", false},
	}
	for _, c := range cases {
		if got := SaysNothing(c.text, c.lang); got != c.vague {
			t.Errorf("SaysNothing(%q, %s) = %v", c.text, c.lang, got)
		}
	}
	if !HasPhrases("pt-BR") || HasPhrases("it") {
		t.Error("HasPhrases")
	}
}

func TestBareURL(t *testing.T) {
	cases := []struct {
		text, href string
		bare       bool
	}{
		{"https://docs.acme.io/migrate-v2", "https://docs.acme.io/migrate-v2", true},
		{"www.acme.io", "https://www.acme.io", true},
		{"docs.acme.io/flags", "https://docs.acme.io/flags", true},
		{"acme.io", "https://acme.io", true},
		{"acme.io", "https://acme.io/pricing", true},
		{"acme.io", "https://docs.example.com", false},
		{"Migration guide", "https://docs.acme.io/migrate-v2", false},
		{"v2.0 is out", "https://acme.io", false},
	}
	for _, c := range cases {
		if got := BareURL(c.text, c.href); got != c.bare {
			t.Errorf("BareURL(%q) = %v", c.text, got)
		}
	}
	// Every language gets the bare-URL check, even without a phrase list.
	body := `<mj-section><mj-column><mj-text color="#111111"><a href="https://acme.io/guide" style="color:#0F766E">https://acme.io/guide</a></mj-text></mj-column></mj-section>`
	if r := check(t, []Version{{Lang: "it", MJML: mj(body)}}, Options{}); find(r, KindLinkText) == nil {
		t.Error("bare URL in Italian")
	}
}

func TestHeadingRuns(t *testing.T) {
	h := func(level int, text string) string {
		return `<mj-text color="#111111"><h` + string(rune('0'+level)) + ` style="margin:0">` + text + `</h` + string(rune('0'+level)) + `></mj-text>`
	}
	cases := []struct {
		name   string
		levels []int
		issues []string // "from→to×count"
	}{
		{"in order", []int{1, 2, 3, 2, 3}, nil},
		{"starts at 2", []int{2, 3}, nil},
		{"skip", []int{1, 3}, []string{"3→2×1"}},
		{"siblings", []int{1, 3, 3}, []string{"3→2×2"}},
		{"subtree moves up", []int{1, 3, 4}, []string{"3→2×1", "4→3×1"}},
		{"back to the top", []int{1, 3, 1, 2}, []string{"3→2×1"}},
		{"two skips", []int{1, 3, 2, 4}, []string{"3→2×1", "4→3×1"}},
	}
	for _, c := range cases {
		var body strings.Builder
		body.WriteString(`<mj-section><mj-column>`)
		for i, l := range c.levels {
			body.WriteString(h(l, "Title "+string(rune('A'+i))))
		}
		body.WriteString(`</mj-column></mj-section>`)
		r := check(t, []Version{{Lang: "en", MJML: mj(body.String())}}, Options{})
		var got []string
		for _, is := range r.Issues {
			if is.Kind != KindHeading {
				continue
			}
			ch := is.Fix.Headings
			got = append(got, string(rune('0'+ch[0].From))+"→"+string(rune('0'+ch[0].To))+"×"+string(rune('0'+len(ch))))
		}
		if strings.Join(got, ",") != strings.Join(c.issues, ",") {
			t.Errorf("%s: %v, want %v", c.name, got, c.issues)
		}
	}
}

func TestTextSize(t *testing.T) {
	body := `<mj-section><mj-column><mj-text color="#111111">Default size</mj-text><mj-text color="#111111" font-size="16px">Big with <span style="font-size:10px">tiny print</span></mj-text><mj-text color="#111111" font-size="9pt">Points</mj-text></mj-column></mj-section>`
	r := check(t, []Version{{Lang: "en", MJML: mj(body)}}, Options{})
	var sizes []string
	for _, is := range r.Issues {
		if is.Kind == KindTextSize {
			sizes = append(sizes, trimFloat(is.Fix.FromPx))
		}
	}
	// MJML's default is 13 px; 10 px inline; 9 pt is 12 px.
	if strings.Join(sizes, ",") != "10" {
		t.Errorf("small text: %v", sizes)
	}
}

// Hidden text (the preheader) is neither checked for contrast nor size.
func TestHiddenTextIgnored(t *testing.T) {
	src := `<mjml><mj-head><mj-preview>Hidden preview</mj-preview></mj-head><mj-body><mj-section><mj-column><mj-text color="#111111">Shown <span style="display:none;color:#FFFFFF;font-size:1px">gone</span></mj-text></mj-column></mj-section></mj-body></mjml>`
	if r := check(t, []Version{{Lang: "en", MJML: src}}, Options{}); len(r.Issues) != 0 {
		t.Fatalf("hidden text: %+v", r.Issues)
	}
}

// HTML-only emails are checked too; their issues have no element path.
func TestHTMLOnly(t *testing.T) {
	src := `<!doctype html><html><body style="background:#ffffff"><table width="600"><tr><td style="color:#999999;font-size:11px">Grey <a href="https://acme.io/more" style="color:#0F766E">Read more</a><img src="https://x.test/a.png"></td></tr></table></body></html>`
	r := check(t, []Version{{Lang: "en", HTML: src}}, Options{})
	if r.Mode != "html" {
		t.Fatalf("mode %s", r.Mode)
	}
	got := strings.Join(kinds(r), ",")
	if got != "alt_text,contrast,link_text,text_size" {
		t.Fatalf("issues: %s", got)
	}
	for _, is := range r.Issues {
		if len(is.Path) != 0 {
			t.Errorf("%s has a path in an HTML-only email", is.ID)
		}
	}
	if r.Counts.RoleAdded != 1 || r.Counts.LayoutTables != 1 {
		t.Errorf("the layout table is marked: %+v", r.Counts)
	}
}

func TestCompileError(t *testing.T) {
	if _, err := Check([]Version{{Lang: "en", MJML: "<mjml><mj-body><mj-section></mj-body></mjml>"}}, Options{}); err == nil {
		t.Fatal("broken MJML must fail the check")
	}
}

// An issue in only some versions lists only those languages.
func TestMergeLanguages(t *testing.T) {
	en := mj(`<mj-section><mj-column><mj-image src="https://x.test/a.png" alt="A chart"></mj-image></mj-column></mj-section>`)
	de := strings.Replace(en, ` alt="A chart"`, "", 1)
	r := check(t, []Version{{Lang: "en", MJML: en, Source: true}, {Lang: "de", MJML: de}}, Options{})
	if len(r.Issues) != 1 || strings.Join(r.Issues[0].Languages, ",") != "de" {
		t.Fatalf("issues: %+v", r.Issues)
	}
}
