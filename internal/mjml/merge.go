package mjml

import (
	"sort"
	"strings"
)

// MergedSection is a section a merge took from the other side.
type MergedSection struct {
	Index int    `json:"index"`
	Label string `json:"label"`
	MJML  string `json:"mjml"`
}

// Merge is a three-way merge of two edits of the same version, section by
// section (review 2026-10, CO-10): when two people changed different
// sections, each keeps theirs.
type Merge struct {
	// MJML is mine with their sections (and head) put in. Its head has no
	// <mj-title> or <mj-preview>: they follow the subject and preheader,
	// which are merged on their own (SetHeadMeta puts them back).
	MJML string
	// Theirs are the sections taken from theirs, in mine's positions.
	Theirs []MergedSection
	// Head is the inner MJML of their mj-head (styles, fonts, attributes)
	// when theirs changed it and mine didn't.
	Head string
}

type mergeDoc struct {
	src      string
	sections []*Node
	head     *Node
	// open is where a head goes when the document has none: just after <mjml>.
	open int
}

func parseDoc(src string) (mergeDoc, bool) {
	// The title and preview follow the subject and preheader: not part of
	// the comparison.
	src = SetHeadMeta(src, "", "")
	root, err := Parse(src)
	if err != nil {
		return mergeDoc{}, false
	}
	body := Body(root)
	if body == nil {
		return mergeDoc{}, false
	}
	d := mergeDoc{src: src, head: Head(root), open: -1}
	if i := strings.IndexByte(src[root.Start:], '>'); i >= 0 {
		d.open = root.Start + i + 1
	}
	for _, n := range body.Children {
		if strings.HasPrefix(n.Tag, "mj-") && n.End > n.Start {
			d.sections = append(d.sections, n)
		}
	}
	return d, true
}

func (d mergeDoc) text(n *Node) string {
	if n == nil || n.End <= n.Start {
		return ""
	}
	return d.src[n.Start:n.End]
}

// headKey is what the head holds besides the title and preview, with
// spacing ignored.
func (d mergeDoc) headKey() string {
	if d.head == nil {
		return ""
	}
	var parts []string
	for _, c := range d.head.Children {
		if !strings.HasPrefix(c.Tag, "mj-") || c.Tag == "mj-title" || c.Tag == "mj-preview" {
			continue
		}
		parts = append(parts, strings.Join(strings.Fields(d.text(c)), " "))
	}
	return strings.Join(parts, "\n")
}

// headInner is the head's content between <mj-head> and </mj-head>.
func (d mergeDoc) headInner() string {
	t := d.text(d.head)
	open := strings.IndexByte(t, '>')
	end := strings.LastIndex(t, "</")
	if open < 0 || end <= open {
		return ""
	}
	return strings.TrimSpace(t[open+1 : end])
}

// MergeSections merges theirs and mine, both edited from base, when they
// changed different sections. ok is false when they changed the same
// section (or the head) differently, or added, removed or moved sections:
// a person then chooses.
func MergeSections(base, theirs, mine string) (Merge, bool) {
	b, ok1 := parseDoc(base)
	t, ok2 := parseDoc(theirs)
	m, ok3 := parseDoc(mine)
	if !ok1 || !ok2 || !ok3 || len(b.sections) != len(t.sections) || len(b.sections) != len(m.sections) {
		return Merge{}, false
	}
	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	var out Merge
	bh, th, mh := b.headKey(), t.headKey(), m.headKey()
	switch {
	case th == bh || th == mh:
	case mh == bh:
		out.Head = t.headInner()
		head := "<mj-head>\n    " + out.Head + "\n  </mj-head>"
		switch {
		case m.head != nil:
			edits = append(edits, edit{m.head.Start, m.head.End, head})
		case m.open >= 0:
			edits = append(edits, edit{m.open, m.open, "\n  " + head})
		default:
			return Merge{}, false
		}
	default:
		return Merge{}, false
	}
	for i := range b.sections {
		bs, ts, ms := b.text(b.sections[i]), t.text(t.sections[i]), m.text(m.sections[i])
		switch {
		case ts == bs || ts == ms:
		case ms == bs:
			edits = append(edits, edit{m.sections[i].Start, m.sections[i].End, ts})
			out.Theirs = append(out.Theirs, MergedSection{Index: i, Label: SectionLabel(t.sections[i], i, len(t.sections)), MJML: ts})
		default:
			return Merge{}, false
		}
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	src := m.src
	for _, e := range edits {
		src = src[:e.start] + e.text + src[e.end:]
	}
	out.MJML = src
	return out, true
}
