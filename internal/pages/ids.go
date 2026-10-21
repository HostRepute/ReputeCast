package pages

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// UniqueIDs gives every block of a page an id of its own: a top-level
// section without one gets one from its kind, and a repeated id (a block
// duplicated in the builder) gets a number ("b-faq-2"). Countdowns name the
// blocks they reveal and hide by these ids. src is sanitized HTML.
func UniqueIDs(src string) string {
	nodes, err := html.ParseFragment(strings.NewReader(src), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return src
	}
	seen := map[string]bool{}
	changed := false
	setAttr := func(n *html.Node, key, val string) {
		for i := range n.Attr {
			if n.Attr[i].Key == key {
				n.Attr[i].Val = val
				return
			}
		}
		n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
	}
	free := func(base string) string {
		id := base
		for i := 2; seen[id]; i++ {
			id = fmt.Sprintf("%s-%d", base, i)
		}
		seen[id] = true
		return id
	}
	var visit func(n *html.Node, top bool)
	visit = func(n *html.Node, top bool) {
		if n.Type == html.ElementNode {
			id := attr(n, "data-pb-id")
			switch {
			case id != "" && seen[id]:
				setAttr(n, "data-pb-id", free(clip(id, 50)))
				changed = true
			case id != "":
				seen[id] = true
			case top && n.Data == "section":
				kind := attr(n, "data-pb")
				if kind == "" {
					kind = "section"
				}
				setAttr(n, "data-pb-id", free("b-"+clip(kind, 40)))
				changed = true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c, false)
		}
	}
	for _, n := range nodes {
		visit(n, true)
	}
	if !changed {
		return src
	}
	var buf bytes.Buffer
	for _, n := range nodes {
		if err := html.Render(&buf, n); err != nil {
			return src
		}
	}
	// html.Render writes custom elements and attributes as they were; a
	// second Sanitize pass keeps the stored form canonical.
	return Sanitize(buf.String())
}
