package mjml

import "strings"

// AttrSynced marks a synced block in an email (review 2026-10, CO-11): the
// element, a section or a wrapper, is the workspace's synced block of that
// kind ("header", "footer", or a named block's key such as
// "b-0123456789ab", R4.A3), and is replaced whenever the block changes
// (internal/brandkit). Like every rc-* attribute it is left out of the
// compiled HTML.
const AttrSynced = "rc-synced"

// SyncedKinds lists the kinds of synced blocks a document holds, in the
// order they first appear.
func SyncedKinds(src string) []string {
	if !strings.Contains(src, AttrSynced) {
		return nil
	}
	root, err := Parse(src)
	if err != nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	root.Walk(func(n *Node, _ []int) {
		if v, ok := n.Get(AttrSynced); ok && v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	})
	return out
}

// ReplaceSynced puts block in place of every element marked
// rc-synced="kind" (not inside another such element). block is the synced
// block's MJML, one element. It returns the document unchanged, and false,
// when nothing is marked, nothing changes or the document doesn't parse.
func ReplaceSynced(src, kind, block string) (string, bool) {
	if kind == "" || !strings.Contains(src, AttrSynced) {
		return src, false
	}
	root, err := Parse(src)
	if err != nil {
		return src, false
	}
	type span struct{ start, end int }
	var spans []span
	var walk func(n *Node)
	walk = func(n *Node) {
		if v, ok := n.Get(AttrSynced); ok && v == kind && n.End > n.Start {
			spans = append(spans, span{n.Start, n.End})
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	if len(spans) == 0 {
		return src, false
	}
	var b strings.Builder
	last := 0
	for _, sp := range spans {
		if sp.start < last {
			continue
		}
		b.WriteString(src[last:sp.start])
		b.WriteString(block)
		last = sp.end
	}
	b.WriteString(src[last:])
	out := b.String()
	return out, out != src
}

// UnmarkSynced unlinks the elements marked rc-synced="kind": they keep
// their content as ordinary sections and no longer follow the block (it
// was deleted, R4.A3). It returns the document unchanged, and false, when
// nothing is marked or the document doesn't parse.
func UnmarkSynced(src, kind string) (string, bool) {
	if kind == "" || !strings.Contains(src, AttrSynced) {
		return src, false
	}
	root, err := Parse(src)
	if err != nil {
		return src, false
	}
	var marked []*Node
	root.Walk(func(n *Node, _ []int) {
		if v, ok := n.Get(AttrSynced); ok && v == kind {
			marked = append(marked, n)
		}
	})
	// From the last: taking an attribute out moves only what follows it.
	changed := false
	for i := len(marked) - 1; i >= 0; i-- {
		if next, ok := RemoveAttr(src, marked[i], AttrSynced); ok {
			src, changed = next, true
		}
	}
	return src, changed
}

const fragPre, fragPost = "<mjml><mj-body>", "</mj-body></mjml>"

// WrapFragment puts body elements in a document so Parse reads them;
// UnwrapFragment takes them back out.
func WrapFragment(block string) string { return fragPre + block + fragPost }

// UnwrapFragment is the body elements of a document made by WrapFragment.
func UnwrapFragment(doc string) string {
	return strings.TrimSuffix(strings.TrimPrefix(doc, fragPre), fragPost)
}

// MarkSynced makes block, one MJML element, the synced block of a kind: it
// sets rc-synced on its outermost element. ok is false when block isn't a
// single section or wrapper.
func MarkSynced(block, kind string) (string, bool) {
	doc := WrapFragment(strings.TrimSpace(block))
	root, err := Parse(doc)
	if err != nil {
		return block, false
	}
	body := Body(root)
	if body == nil {
		return block, false
	}
	var top []*Node
	for _, c := range body.Children {
		if strings.HasPrefix(c.Tag, "mj-") {
			top = append(top, c)
		}
	}
	if len(top) != 1 || (top[0].Tag != "mj-section" && top[0].Tag != "mj-wrapper") {
		return block, false
	}
	out, ok := SetAttr(doc, top[0], AttrSynced, kind)
	if !ok {
		return block, false
	}
	return UnwrapFragment(out), true
}
