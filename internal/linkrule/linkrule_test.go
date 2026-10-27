package linkrule_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/hostrepute/reputecast/internal/linkrule"
	"github.com/hostrepute/reputecast/internal/liquid"
)

func TestTrackable(t *testing.T) {
	for href, want := range map[string]bool{
		"https://acme.io/changelog":             true,
		"http://acme.io":                        true,
		"{{ contact.website }}":                 true,
		"https://acme.io/?ref={{ contact.id }}": true,
		"":                                      false,
		"#top":                                  false,
		"mailto:hi@acme.io":                     false,
		"TEL:+1555":                             false,
		"javascript:void(0)":                    false,
		"{{ unsubscribe_url }}":                 false,
		"{{ preferences_url }}":                 false,
		"{{ view_in_browser_url }}":             false,
		"https://acme.io/p?a={{ rc_poll_1_a }}": false,
		"{{ upgrade_url }}":                     false,
		"https://acme.io/r/{{ rc_rec_3 }}":      false,
		"ftp://files.acme.io":                   false,
		"/relative/path":                        false,
	} {
		if got := linkrule.Trackable(href); got != want {
			t.Errorf("Trackable(%q) = %v, want %v", href, got, want)
		}
	}
}

func TestRewrite(t *testing.T) {
	src := `<p><a class="b" href="https://acme.io/a">Read <b>the</b> docs</a> <a href='mailto:x@y'>mail</a> <a href=https://acme.io/b>B</a></p>`
	out, links := linkrule.Rewrite(src, func(l linkrule.Link) string { return "{{ rc_link_" + strconv.Itoa(l.Position) + " }}" })
	want := `<p><a class="b" href="{{ rc_link_0 }}">Read <b>the</b> docs</a> <a href='mailto:x@y'>mail</a> <a href="{{ rc_link_1 }}">B</a></p>`
	if out != want {
		t.Fatalf("rewritten:\n%s\nwant\n%s", out, want)
	}
	if len(links) != 2 || links[0].Label != "Read the docs" || links[1].URL != "https://acme.io/b" {
		t.Fatalf("links: %+v", links)
	}
}

// TestMarkNumbersLikeTheEngine: the click map's marks are the positions
// the engine gives the same email's links, blocks that render per reader
// included.
func TestMarkNumbersLikeTheEngine(t *testing.T) {
	liquid.RegisterBlock("linkrule-test", func([]byte) string {
		return `<a href="https://acme.io/r/{{ referral.code }}">Share</a><a href="{{ unsubscribe_url }}">x</a><a href="https://acme.io/rewards">Rewards</a>`
	})
	src := `<a href="https://acme.io/first">First</a><!-- rc:linkrule-test {"layout":"bar"} --><a href="#x">anchor</a>` +
		`<!--rc:if seg1--><a href="https://acme.io/members">Members</a><!--rc:fi seg1--><a href="{{ preferences_url }}">prefs</a><a href="https://acme.io/last">Last</a>`
	_, links := linkrule.Rewrite(liquid.ExpandBlocks(src), func(l linkrule.Link) string { return l.URL })
	byURL := map[string]int{}
	for _, l := range links {
		byURL[l.URL] = l.Position
	}
	marked := linkrule.Mark(src)
	re := regexp.MustCompile(`<a data-rc-link="(\d+)" href="([^"]+)"`)
	got := re.FindAllStringSubmatch(marked, -1)
	if len(got) != 3 {
		t.Fatalf("%d marked links in %s", len(got), marked)
	}
	for _, m := range got {
		pos, _ := strconv.Atoi(m[1])
		if want, ok := byURL[m[2]]; !ok || want != pos {
			t.Errorf("%s marked %d, the engine numbers it %d", m[2], pos, want)
		}
	}
	if byURL["https://acme.io/last"] != 4 {
		t.Fatalf("the engine's numbering: %v", byURL)
	}
}

// TestImageLinkLabels: a link without text is named by the anchor's
// aria-label or title, else its image's alt text.
func TestImageLinkLabels(t *testing.T) {
	src := `<a href="https://acme.io"><img src="logo.png" alt="Acme" title="ignored"></a>` +
		`<a href="https://acme.io/hero" aria-label="Hero image"><img src="h.png" alt="Hero"></a>` +
		`<a href="https://acme.io/x"><img src="x.png"></a>`
	_, links := linkrule.Rewrite(src, func(l linkrule.Link) string { return l.URL })
	if len(links) != 3 || links[0].Label != "Acme" || links[1].Label != "Hero image" || links[2].Label != "" {
		t.Fatalf("labels: %+v", links)
	}
}

// TestRows: links with nothing but separators between them are one row
// (a navigation bar); a link elsewhere, or behind other markup, isn't.
func TestRows(t *testing.T) {
	src := `<a href="https://acme.io"><img alt="Acme"></a><div><a href="https://acme.io/blog">Blog</a> &middot; <a href="https://docs.acme.io">Docs</a> | ` +
		`<a href="https://acme.io/changelog">Changelog</a></div><p>Read <a href="https://docs.acme.io/flags">the docs</a> and ` +
		`<a href="https://acme.io/x">more</a>.</p><p><a href="{{ unsubscribe_url }}">Unsubscribe</a> · <a href="{{ preferences_url }}">Preferences</a></p>` +
		`<a href="https://acme.io/a">A</a><br><a href="https://acme.io/b">B</a>`
	rows := linkrule.Rows(src)
	if len(rows) != 1 || len(rows[0]) != 3 || rows[0][0] != 1 || rows[0][2] != 3 {
		t.Fatalf("rows: %v", rows)
	}
}
