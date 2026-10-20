package mjml

import (
	"encoding/json"
	"html"
	"regexp"
	"sort"
	"strings"
)

// ReputeCast keeps a few settings of its own on MJML elements, as rc-*
// attributes the MJML compilers don't know (design expansion, M21):
//
//	rc-visible  the block's Visibility rule (internal/visibility)
//	rc-name     the block's name in the layer tree ("Beta corner")
//	rc-poll     a Poll block's settings (internal/polls)
//	rc-omit     a block the editor shows and the email leaves out (a Sponsor
//	            block without a booking, internal/sponsors)
//
// They are content: versioned, copied with a block, kept by language
// versions. Before an email is compiled they come out (both compilers would
// report them as unknown attributes) and every conditional block is wrapped
// in two markers that survive the compile:
//
//	<mj-raw><!--rc:if a1b2c3--></mj-raw> … the block … <mj-raw><!--rc:fi a1b2c3--></mj-raw>
//
// so the compiled HTML of the whole email can be cut down to one version
// per recipient without compiling it again (internal/visibility.Apply).

// AttrVisible, AttrName, AttrPoll and AttrOmit are ReputeCast's own attributes.
const (
	AttrVisible = "rc-visible"
	AttrName    = "rc-name"
	AttrPoll    = "rc-poll"
	AttrOmit    = "rc-omit"
)

// Marker comments around a conditional block in compiled HTML.
const (
	MarkerIf = "<!--rc:if "
	MarkerFi = "<!--rc:fi "
)

// rawParents are the elements that take <mj-raw> children: a conditional
// block anywhere else (a navbar link, a social element) can't be marked and
// is shown to everyone.
var rawParents = map[string]bool{
	"mj-body": true, "mj-wrapper": true, "mj-section": true, "mj-group": true, "mj-column": true, "mj-hero": true,
}

// visibleHeader is the part of a Visibility rule the compile step reads: the
// block's id, and whether it is shown to everyone (nothing to mark).
type visibleHeader struct {
	ID   string `json:"id"`
	Show string `json:"show"`
}

// ConditionalID returns the id of a conditional element: one with a
// Visibility rule that doesn't show it to everyone. "" otherwise.
func ConditionalID(n *Node) string {
	raw, ok := n.Get(AttrVisible)
	if !ok || strings.TrimSpace(raw) == "" {
		return ""
	}
	var h visibleHeader
	if json.Unmarshal([]byte(raw), &h) != nil || !validID(h.ID) || h.Show == "" || h.Show == "everyone" {
		return ""
	}
	return h.ID
}

var idRe = regexp.MustCompile(`^[a-z0-9]{1,24}$`)

func validID(id string) bool { return idRe.MatchString(id) }

// rcAttr matches one rc-* attribute in an opening tag, with its value.
var rcAttr = regexp.MustCompile(`\s+rc-[a-z-]+\s*=\s*("[^"]*"|'[^']*'|[^\s>"']+)`)

// PrepareCompile turns a document into what the compilers get: rc-*
// attributes removed, blocks marked rc-omit left out and conditional blocks
// wrapped in their markers. A document that doesn't parse is returned
// unchanged (the compiler reports its error). A block whose id appears
// twice is marked once, the first time: the composer gives a copied block an
// id of its own.
func PrepareCompile(src string) string {
	if !strings.Contains(src, "rc-") {
		return src
	}
	root, err := Parse(src)
	if err != nil {
		return src
	}
	type edit struct {
		start, end int
		text       string
		order      int
	}
	var edits []edit
	seen := map[string]bool{}
	var walk func(n, parent *Node)
	walk = func(n, parent *Node) {
		if strings.HasPrefix(n.Tag, "mj-") && n.End > n.Start && parent != nil && omitted(n) {
			edits = append(edits, edit{start: n.Start, end: n.End, text: "", order: 1})
			return
		}
		if strings.HasPrefix(n.Tag, "mj-") && n.End > n.Start {
			open := strings.IndexByte(src[n.Start:n.End], '>')
			if open > 0 {
				tag := src[n.Start : n.Start+open+1]
				if strings.Contains(tag, "rc-") {
					cleaned := rcAttr.ReplaceAllString(tag, "")
					if cleaned != tag {
						edits = append(edits, edit{start: n.Start, end: n.Start + open + 1, text: cleaned, order: 1})
					}
				}
			}
			if id := ConditionalID(n); id != "" && parent != nil && rawParents[parent.Tag] && !seen[id] {
				seen[id] = true
				edits = append(edits,
					edit{start: n.Start, end: n.Start, text: "<mj-raw>" + MarkerIf + id + "--></mj-raw>", order: 0},
					edit{start: n.End, end: n.End, text: "<mj-raw>" + MarkerFi + id + "--></mj-raw>", order: 2})
			}
		}
		for _, c := range n.Children {
			walk(c, n)
		}
	}
	walk(root, nil)
	if len(edits) == 0 {
		return src
	}
	// Apply from the start; an insertion at an element's start comes before
	// the edit of its opening tag, an insertion at its end after its close.
	sort.SliceStable(edits, func(i, j int) bool {
		if edits[i].start != edits[j].start {
			return edits[i].start < edits[j].start
		}
		return edits[i].order < edits[j].order
	})
	var b strings.Builder
	last := 0
	for _, e := range edits {
		if e.start < last {
			continue // overlapping edit (a malformed document): skip it
		}
		b.WriteString(src[last:e.start])
		b.WriteString(e.text)
		last = e.end
	}
	b.WriteString(src[last:])
	return b.String()
}

// omitted reports whether an element is shown in the editor only.
func omitted(n *Node) bool {
	v, ok := n.Get(AttrOmit)
	return ok && v != "" && v != "0" && v != "false"
}

// StripMarkers removes the conditional markers from compiled HTML: the
// email every recipient would get if no block were conditional.
func StripMarkers(html string) string {
	if !strings.Contains(html, "<!--rc:") {
		return html
	}
	return markerRe.ReplaceAllString(html, "")
}

var markerRe = regexp.MustCompile(`<!--rc:(?:if|fi) [a-z0-9]{1,24}-->`)

// attrRe matches one attribute of an opening tag by name, with its value.
func attrRe(name string) *regexp.Regexp {
	return regexp.MustCompile(`\s+` + regexp.QuoteMeta(name) + `(?:\s*=\s*("[^"]*"|'[^']*'|[^\s>"']+))?`)
}

// openingTag returns the end of the element's opening tag (just past '>').
func openingTag(src string, n *Node) (int, bool) {
	if n == nil || n.Start < 0 || n.End > len(src) || n.End <= n.Start {
		return 0, false
	}
	gt := strings.IndexByte(src[n.Start:n.End], '>')
	if gt < 0 {
		return 0, false
	}
	return n.Start + gt + 1, true
}

// SetAttr sets an attribute of an element in the source: it replaces the
// value or adds the attribute after the tag's name. Values are written
// double-quoted, escaped.
func SetAttr(src string, n *Node, name, value string) (string, bool) {
	end, ok := openingTag(src, n)
	if !ok {
		return src, false
	}
	tag := src[n.Start:end]
	esc := html.EscapeString(value)
	var next string
	if loc := attrRe(name).FindStringIndex(tag); loc != nil {
		next = tag[:loc[0]] + " " + name + `="` + esc + `"` + tag[loc[1]:]
	} else {
		at := len("<" + n.Tag)
		if at > len(tag) || !strings.EqualFold(tag[:at], "<"+n.Tag) {
			return src, false
		}
		next = tag[:at] + " " + name + `="` + esc + `"` + tag[at:]
	}
	return src[:n.Start] + next + src[end:], true
}

// RemoveAttr removes an attribute of an element in the source.
func RemoveAttr(src string, n *Node, name string) (string, bool) {
	end, ok := openingTag(src, n)
	if !ok {
		return src, false
	}
	tag := src[n.Start:end]
	next := attrRe(name).ReplaceAllString(tag, "")
	if next == tag {
		return src, false
	}
	return src[:n.Start] + next + src[end:], true
}
