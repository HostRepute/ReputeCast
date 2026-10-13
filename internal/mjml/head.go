package mjml

import (
	"html"
	"sort"
	"strings"
)

// SetHeadMeta writes <mj-title> (the subject) and <mj-preview> (the
// preheader) into a document so the compiled HTML's title and hidden preview
// text always match the fields. Empty values remove the element. A document
// that doesn't parse is returned unchanged.
func SetHeadMeta(src, title, preview string) string {
	root, err := Parse(src)
	if err != nil {
		return src
	}
	head := Head(root)
	el := func(tag, v string) string {
		if v == "" {
			return ""
		}
		return "<" + tag + ">" + html.EscapeString(v) + "</" + tag + ">"
	}
	if head == nil {
		meta := strings.TrimSpace(el("mj-title", title) + "\n    " + el("mj-preview", preview))
		if meta == "" {
			return src
		}
		open := strings.IndexByte(src[root.Start:], '>')
		if open < 0 {
			return src
		}
		at := root.Start + open + 1
		return src[:at] + "\n  <mj-head>\n    " + meta + "\n  </mj-head>" + src[at:]
	}
	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	for _, tag := range []struct{ name, value string }{{"mj-title", title}, {"mj-preview", preview}} {
		var found *Node
		for _, c := range head.Children {
			if c.Tag == tag.name {
				found = c
				break
			}
		}
		if found != nil {
			edits = append(edits, edit{found.Start, found.End, el(tag.name, tag.value)})
			continue
		}
		if tag.value == "" {
			continue
		}
		// Insert right after <mj-head>.
		open := strings.IndexByte(src[head.Start:], '>')
		if open < 0 {
			continue
		}
		at := head.Start + open + 1
		edits = append(edits, edit{at, at, "\n    " + el(tag.name, tag.value)})
	}
	// Apply from the end so offsets stay valid.
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	out := src
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		out = out[:e.start] + e.text + out[e.end:]
	}
	return out
}

// HeadMeta reads the title and preview text of a document.
func HeadMeta(src string) (title, preview string) {
	root, err := Parse(src)
	if err != nil {
		return "", ""
	}
	head := Head(root)
	if head == nil {
		return "", ""
	}
	for _, c := range head.Children {
		switch c.Tag {
		case "mj-title":
			title = html.UnescapeString(strings.TrimSpace(c.Inner))
		case "mj-preview":
			preview = html.UnescapeString(strings.TrimSpace(c.Inner))
		}
	}
	return title, preview
}
