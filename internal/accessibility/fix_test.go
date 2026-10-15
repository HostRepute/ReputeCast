package accessibility

import (
	"strings"
	"testing"
)

func mustFix(t *testing.T, src string, f FixRequest) string {
	t.Helper()
	out, err := ApplyFix(src, f)
	if err != nil {
		t.Fatalf("fix %s: %v", f.Kind, err)
	}
	return out
}

func issueOf(t *testing.T, src, kind string) *AccessibilityIssue {
	t.Helper()
	r := check(t, []Version{{Lang: "en", MJML: src}}, Options{})
	return find(r, kind)
}

// Fixing each issue of #143 makes it go away; the rest stays.
func TestFixesOnDigest(t *testing.T) {
	src := digest(t)
	r := check(t, []Version{{Lang: "en", MJML: src}}, Options{})
	alt, ct, h, size := find(r, KindAltText), find(r, KindContrast), find(r, KindHeading), find(r, KindTextSize)

	out := mustFix(t, src, FixRequest{Kind: FixAltText, Path: alt.Path, Value: `A timeline with a flag change "scheduled" for 2 a.m.`})
	if issueOf(t, out, KindAltText) != nil || !strings.Contains(out, `alt="A timeline with a flag change &#34;scheduled&#34; for 2 a.m."`) {
		t.Errorf("alt text: %s", out[strings.Index(out, "scheduled-rollouts.png")-20:strings.Index(out, "scheduled-rollouts.png")+120])
	}
	out = mustFix(t, src, FixRequest{Kind: FixDecorative, Path: alt.Path})
	if issueOf(t, out, KindAltText) != nil || !strings.Contains(out, `alt=""`) {
		t.Errorf("decorative")
	}

	out = mustFix(t, src, FixRequest{Kind: FixColor, Path: ct.Path, Target: ct.Fix.Target, From: ct.Fix.From, Value: "#0F766E"})
	if issueOf(t, out, KindContrast) != nil || !strings.Contains(out, `background-color="#0F766E" color="#ffffff" font-size="15px"`) {
		t.Errorf("contrast fix")
	}

	out = mustFix(t, src, FixRequest{Kind: FixHeading, Headings: h.Fix.Headings})
	if issueOf(t, out, KindHeading) != nil || strings.Count(out, "<h2 ") != 2 || strings.Contains(out, "<h3") || strings.Contains(out, "</h3>") {
		t.Errorf("headings: %d h2", strings.Count(out, "<h2 "))
	}
	// The style already set size and margin: nothing is added.
	if !strings.Contains(out, `<h2 style="margin:0;font-size:17px;line-height:1.3;font-weight:700">Scheduled rollouts</h2>`) {
		t.Errorf("heading look changed")
	}

	out = mustFix(t, src, FixRequest{Kind: FixTextSize, Path: size.Path, ToPx: 12})
	if issueOf(t, out, KindTextSize) != nil || !strings.Contains(out, `font-size="12px" line-height="1.6" color="#5B6472">Acme Inc`) {
		t.Errorf("text size")
	}

	click := r.Issues[2]
	out = mustFix(t, src, FixRequest{Kind: FixLinkText, Path: click.Path, Index: click.Element.Index, Value: "Scheduled rollouts · Acme Docs"})
	if !strings.Contains(out, `style="color:#0F766E">Scheduled rollouts · Acme Docs</a> to see how it works.`) {
		t.Errorf("link text")
	}
	r2 := check(t, []Version{{Lang: "en", MJML: out}}, Options{})
	if len(r2.Issues) != 5 {
		t.Errorf("one issue fixed, five left: %v", kinds(r2))
	}
}

// A heading that relied on the old level's default look keeps it.
func TestHeadingFixKeepsLook(t *testing.T) {
	src := mj(`<mj-section><mj-column><mj-text color="#111111"><h1>Top</h1></mj-text><mj-text color="#111111"><h3 style="color:#111111">Card</h3><h3>Other</h3></mj-text></mj-column></mj-section>`)
	is := issueOf(t, src, KindHeading)
	if is == nil || len(is.Fix.Headings) != 2 {
		t.Fatalf("issue: %+v", is)
	}
	out := mustFix(t, src, FixRequest{Kind: FixHeading, Headings: is.Fix.Headings})
	if !strings.Contains(out, `<h2 style="color:#111111;font-size:1.17em;margin:1em 0">Card</h2>`) || !strings.Contains(out, `<h2 style="font-size:1.17em;margin:1em 0">Other</h2>`) {
		t.Fatalf("look: %s", out)
	}
	if issueOf(t, out, KindHeading) != nil {
		t.Fatal("still skipped")
	}
}

func TestColorFixInsideText(t *testing.T) {
	src := mj(`<mj-section><mj-column><mj-text color="#111111">Read the <a href="https://acme.io" style="color:#9CA3AF;text-decoration:underline">guide</a></mj-text></mj-column></mj-section>`)
	is := issueOf(t, src, KindContrast)
	if is == nil || is.Fix.Target != "text" || is.Fix.From != "#9CA3AF" {
		t.Fatalf("issue: %+v", is)
	}
	out := mustFix(t, src, FixRequest{Kind: FixColor, Path: is.Path, Target: "text", From: is.Fix.From, Value: is.Fix.Suggestions[0].Color})
	if issueOf(t, out, KindContrast) != nil || strings.Contains(out, "#9CA3AF") {
		t.Fatalf("inline color not changed: %s", out)
	}
}

func TestTextSizeFixDoesNotShrink(t *testing.T) {
	// The block's own size (16 px from mj-attributes) stays; only the 10 px
	// span grows.
	src := strings.Replace(mj(`<mj-section><mj-column><mj-text color="#111111">Body <span style="font-size:10px">fine print</span></mj-text></mj-column></mj-section>`),
		`<mj-all font-family="Arial"></mj-all>`, `<mj-all font-family="Arial"></mj-all><mj-text font-size="16px"></mj-text>`, 1)
	is := issueOf(t, src, KindTextSize)
	if is == nil {
		t.Fatal("no size issue")
	}
	out := mustFix(t, src, FixRequest{Kind: FixTextSize, Path: is.Path, ToPx: 12})
	if strings.Contains(out, `color="#111111" font-size=`) || !strings.Contains(out, `font-size:12px">fine print`) {
		t.Fatalf("size fix: %s", out)
	}
}

func TestFixInRawHTML(t *testing.T) {
	src := mj(`<mj-raw><table><tr><td style="background:#ffffff"><a href="https://x.com/acme"><img src="https://x.test/x.png" width="28" height="28"></a></td></tr></table></mj-raw>`)
	is := issueOf(t, src, KindLinkNoText)
	if is == nil || is.Element.Role != "linked_image" {
		t.Fatalf("raw linked image: %+v", is)
	}
	out := mustFix(t, src, FixRequest{Kind: FixAltText, Path: is.Path, Index: 0, Value: "Acme on X"})
	if issueOf(t, out, KindLinkNoText) != nil || !strings.Contains(out, `<img src="https://x.test/x.png" width="28" height="28" alt="Acme on X">`) {
		t.Fatalf("raw alt: %s", out)
	}
}

func TestFixErrors(t *testing.T) {
	src := digest(t)
	if _, err := ApplyFix(src, FixRequest{Kind: FixAltText, Path: []int{40, 0}, Value: "x"}); err != ErrFixTarget {
		t.Errorf("missing element: %v", err)
	}
	if _, err := ApplyFix(src, FixRequest{Kind: FixAltText, Path: []int{1, 0, 0}}); err == nil {
		t.Error("empty alt text")
	}
	if _, err := ApplyFix(src, FixRequest{Kind: FixColor, Path: []int{5, 0, 0}, Value: "teal-ish"}); err == nil {
		t.Error("bad color")
	}
	if _, err := ApplyFix(src, FixRequest{Kind: FixTextSize, Path: []int{7, 0, 0}, ToPx: 30}); err == nil {
		t.Error("size out of range")
	}
	if _, err := ApplyFix(src, FixRequest{Kind: FixHeading, Headings: []AccessibilityHeadingChange{{Path: []int{3, 0, 1}, Index: 0, From: 2, To: 1}}}); err != ErrFixTarget {
		t.Errorf("a heading that changed since: %v", err)
	}
}
