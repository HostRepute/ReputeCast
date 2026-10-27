// Package linkrule is the one rule for which of an email's links are
// tracked (review AN-5). The sending engine numbers and redirects the
// links it picks; the report's click map draws its bubbles on the links
// Mark picks with the same rule and numbers the same way, so a bubble is
// always on the link its clicks came from.
package linkrule

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/liquid"
)

// Attr is the attribute Mark adds to each tracked link: its position.
const Attr = "data-rc-link"

var (
	anchorRe   = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a\s*>`)
	hrefRe     = regexp.MustCompile(`(?is)(\shref\s*=\s*)("([^"]*)"|'([^']*)'|([^\s>"']+))`)
	tagRe      = regexp.MustCompile(`(?s)<[^>]*>`)
	spaceRe    = regexp.MustCompile(`\s+`)
	systemLink = regexp.MustCompile(`^\{\{-?\s*(unsubscribe_url|preferences_url|view_in_browser_url)\b`)
	// markOrAnchor finds, in document order, the markers of blocks that
	// render per reader (liquid.ExpandBlocks) and the links.
	markOrAnchor = regexp.MustCompile(`(?is)<!--\s*rc:[a-z][a-z-]*\s*(?:\{[\s\S]*?\})?\s*-->|<a\b[^>]*>.*?</a\s*>`)
)

// Link is a tracked link: its position among the email's tracked links,
// its address as written (it may be personalized with Liquid) and its
// text.
type Link struct {
	Position int
	URL      string
	Label    string
}

// Trackable says whether a link goes through the click redirect. Anchors,
// mailto:, tel:, sms:, javascript: and data: links, the unsubscribe,
// preference and view-online links, poll and survey answers, Earn's
// personal links and the Recommendations block's buttons aren't tracked;
// absolute web links (possibly personalized) are.
func Trackable(href string) bool {
	if href == "" || strings.HasPrefix(href, "#") {
		return false
	}
	lower := strings.ToLower(href)
	for _, p := range []string{"mailto:", "tel:", "sms:", "javascript:", "data:"} {
		if strings.HasPrefix(lower, p) {
			return false
		}
	}
	if systemLink.MatchString(href) || strings.Contains(href, "unsubscribe_url") || strings.Contains(href, "preferences_url") || strings.Contains(href, "view_in_browser_url") {
		return false
	}
	// Poll answers and survey links aren't clicks: the answer page records
	// the answer, and the click map shows a poll as one region.
	if strings.Contains(href, "rc_poll_") || strings.Contains(href, "rc_survey_") {
		return false
	}
	// Earn's personal links: the upgrade page knows the reader and credits
	// the email itself, and the billing link is the reader's own.
	if strings.Contains(href, "upgrade_url") || strings.Contains(href, "manage_subscription_url") {
		return false
	}
	// The Recommendations block's buttons open the hosted recommendations
	// page, which counts them (internal/recommendations); they sit in a
	// loop, so one tracked link couldn't tell the newsletters apart.
	if strings.Contains(href, "rc_rec_") {
		return false
	}
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(href, "{{")
}

// Rewrite replaces the address of each tracked link of src with what to
// returns for it (the engine's per-recipient variable), numbering the links
// from 0 in document order, and returns the links. src is the email as the
// engine numbers it: blocks already expanded (liquid.ExpandBlocks).
func Rewrite(src string, to func(Link) string) (string, []Link) {
	links := []Link{}
	out := anchorRe.ReplaceAllStringFunc(src, func(a string) string {
		attrs, inner, start, end, href, ok := parse(a)
		if !ok || !Trackable(href) {
			return a
		}
		l := Link{Position: len(links), URL: href, Label: label(attrs, inner)}
		links = append(links, l)
		return "<a" + attrs[:start] + `"` + to(l) + `"` + attrs[end:] + ">" + inner + "</a>"
	})
	return out, links
}

// Mark adds Attr with its position to each tracked link of src, numbered
// as the engine numbers the links of the same email: the links that
// blocks rendering per reader add (a referral block's) are counted where
// the block is, though the block stays as written.
func Mark(src string) string {
	pos := 0
	return markOrAnchor.ReplaceAllStringFunc(src, func(m string) string {
		if strings.HasPrefix(m, "<!--") {
			if expanded := liquid.ExpandBlocks(m); expanded != m {
				_, links := Rewrite(expanded, func(l Link) string { return l.URL })
				pos += len(links)
			}
			return m
		}
		attrs, inner, _, _, href, ok := parse(m)
		if !ok || !Trackable(href) {
			return m
		}
		out := "<a " + Attr + `="` + strconv.Itoa(pos) + `"` + attrs + ">" + inner + "</a>"
		pos++
		return out
	})
}

// parse splits an anchor into its attributes and inner HTML and finds its
// href: the attribute value's bounds within attrs (quotes included) and the
// address.
func parse(a string) (attrs, inner string, start, end int, href string, ok bool) {
	m := anchorRe.FindStringSubmatch(a)
	if m == nil {
		return "", "", 0, 0, "", false
	}
	attrs, inner = m[1], m[2]
	hm := hrefRe.FindStringSubmatchIndex(attrs)
	if hm == nil {
		return attrs, inner, 0, 0, "", false
	}
	raw := ""
	for _, g := range []int{3, 4, 5} {
		if hm[2*g] >= 0 {
			raw = attrs[hm[2*g]:hm[2*g+1]]
			break
		}
	}
	return attrs, inner, hm[4], hm[5], strings.TrimSpace(html.UnescapeString(raw)), true
}

// Label is a link's text: its inner HTML without tags, on one line, at most
// 120 characters.
func Label(inner string) string {
	s := html.UnescapeString(tagRe.ReplaceAllString(inner, " "))
	s = strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
	if r := []rune(s); len(r) > 120 {
		s = string(r[:119]) + "…"
	}
	return s
}

// nameRe finds what names a link without text: the anchor's aria-label or
// title, an image's alt text.
var nameRe = regexp.MustCompile(`(?is)\b(aria-label|title|alt)\s*=\s*(?:"([^"]*)"|'([^']*)')`)

// label is a link's text, or for a link without any (an image) the
// anchor's aria-label or title, else the image's alt text.
func label(attrs, inner string) string {
	if s := Label(inner); s != "" {
		return s
	}
	for i, src := range []string{attrs, inner} {
		for _, m := range nameRe.FindAllStringSubmatch(src, -1) {
			if i == 1 && !strings.EqualFold(m[1], "alt") {
				continue
			}
			if s := Label(m[2] + m[3]); s != "" {
				return s
			}
		}
	}
	return ""
}

// separatorRe is what may stand between the links of one row: spaces and
// separators such as "·" or "|" (entities count as separators).
var separatorRe = regexp.MustCompile(`^[\s·•|/,–—-]*$`)
var entityRe = regexp.MustCompile(`&[a-zA-Z0-9#]+;`)

// Rows finds the tracked links of src that sit on one line with nothing but
// separators between them (a navigation bar: "Blog · Docs · Changelog"),
// as runs of at least two positions, numbered as Rewrite and Mark number
// them. The click map shows such a row as one entry.
func Rows(src string) [][]int {
	var out [][]int
	var run []int
	pos, prevEnd := 0, -1
	flush := func() {
		if len(run) >= 2 {
			out = append(out, run)
		}
		run = nil
	}
	for _, ix := range markOrAnchor.FindAllStringIndex(src, -1) {
		m := src[ix[0]:ix[1]]
		if strings.HasPrefix(m, "<!--") {
			if expanded := liquid.ExpandBlocks(m); expanded != m {
				_, links := Rewrite(expanded, func(l Link) string { return l.URL })
				pos += len(links)
			}
			flush()
			prevEnd = -1
			continue
		}
		_, _, _, _, href, ok := parse(m)
		if !ok || !Trackable(href) {
			flush()
			prevEnd = -1
			continue
		}
		between := ""
		if prevEnd >= 0 {
			between = entityRe.ReplaceAllString(src[prevEnd:ix[0]], "·")
		}
		if prevEnd < 0 || !separatorRe.MatchString(between) {
			flush()
		}
		run = append(run, pos)
		pos++
		prevEnd = ix[1]
	}
	flush()
	return out
}
