package mjml

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

// Section is one top-level block of <mj-body>.
type Section struct {
	// Index is the position among mj-body's elements (comments excluded).
	Index int    `json:"index"`
	Tag   string `json:"tag"`
	// Label names the section the way the layer tree does: "header", "hero",
	// "2 columns", "text", "footer".
	Label string `json:"label"`
	// Marker is the ReputeCast block it was inserted from ("footer"), if any.
	Marker string `json:"marker,omitempty"`
	Line   int    `json:"line"`
	// Hash fingerprints the section's source, to tell which sections changed
	// between two versions.
	Hash string `json:"-"`
}

var (
	unsubscribeRe = regexp.MustCompile(`(?i)unsubscribe|\{\{-?\s*unsubscribe_url`)
	addressRe     = regexp.MustCompile(`(?i)\b\d{1,6}\s+[A-Za-zÀ-ÿ.'-]+(\s+[A-Za-zÀ-ÿ.'-]+){0,4}\s+(st|street|ave|avenue|rd|road|blvd|boulevard|ln|lane|dr|drive|way|ct|court|pl|place|sq|square|pkwy|parkway|hwy|highway)\b`)
	tagRe         = regexp.MustCompile(`<[^>]*>`)
)

// Outline lists the sections of a document's body with readable labels.
func Outline(src string) ([]Section, error) {
	root, err := Parse(src)
	if err != nil {
		return nil, err
	}
	body := Body(root)
	if body == nil {
		return []Section{}, nil
	}
	out := make([]Section, 0, len(body.Children))
	for i, n := range body.Children {
		sum := sha256.Sum256([]byte(src[n.Start:n.End]))
		out = append(out, Section{Index: i, Tag: n.Tag, Label: SectionLabel(n, i, len(body.Children)), Marker: n.Marker, Line: n.Line, Hash: hex.EncodeToString(sum[:8])})
	}
	return out, nil
}

// SectionLabel names a top-level body element: the ReputeCast marker when
// there is one, otherwise what it looks like.
func SectionLabel(n *Node, index, total int) string {
	if n.Marker != "" {
		return markerLabel(n.Marker)
	}
	switch n.Tag {
	case "mj-hero":
		return "hero"
	case "mj-raw":
		return "raw HTML"
	case "mj-wrapper":
		return "wrapper"
	}
	cols := 0
	images, texts, buttons, dividers, navs := 0, 0, 0, 0, 0
	maxImage := 0
	var text strings.Builder
	n.Walk(func(x *Node, _ []int) {
		switch x.Tag {
		case "mj-column", "mj-group":
			if x.Tag == "mj-column" {
				cols++
			}
		case "mj-image":
			images++
			if w := px(x.Attr("width")); w > maxImage {
				maxImage = w
			}
		case "mj-text":
			texts++
			text.WriteString(x.Inner)
			text.WriteByte(' ')
		case "mj-button":
			buttons++
		case "mj-divider":
			dividers++
		case "mj-navbar", "mj-social":
			navs++
		}
	})
	plain := tagRe.ReplaceAllString(text.String(), " ")
	switch {
	case index == total-1 && (unsubscribeRe.MatchString(text.String()) || addressRe.MatchString(plain)):
		return "footer"
	case unsubscribeRe.MatchString(text.String()):
		return "footer"
	case dividers > 0 && images == 0 && texts == 0 && buttons == 0:
		return "divider"
	case index == 0 && images > 0 && maxImage > 0 && maxImage <= 200 && (navs > 0 || texts > 0):
		return "header"
	case cols >= 2:
		return strconv.Itoa(cols) + " columns"
	case images > 0 && (maxImage == 0 || maxImage >= 400) && texts > 0:
		return "hero"
	case images > 0 && texts == 0:
		return "image"
	case buttons > 0 && texts == 0:
		return "button"
	case texts > 0:
		return "text"
	}
	return strings.TrimPrefix(n.Tag, "mj-")
}

func markerLabel(m string) string {
	switch m {
	case "header-logo", "header":
		return "header"
	case "article-cards", "article-card":
		return "article cards"
	case "product-card", "product-cards":
		return "product cards"
	}
	return strings.ReplaceAll(m, "-", " ")
}

// px parses "600px" or "600" into 600; percentages and empty values give 0.
func px(v string) int {
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "px"))
	if v == "" || strings.HasSuffix(v, "%") {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return int(f)
}

// ChangedSections compares two documents and returns the labels of the
// sections of `next` that differ from `base` (added, edited or moved), plus
// the labels of sections removed from base.
func ChangedSections(base, next string) []string {
	a, err1 := Outline(base)
	b, err2 := Outline(next)
	if err1 != nil || err2 != nil {
		return []string{"the email body"}
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s.Hash]++
	}
	var out []string
	add := func(l string) {
		for _, x := range out {
			if x == l {
				return
			}
		}
		out = append(out, l)
	}
	for _, s := range b {
		if seen[s.Hash] > 0 {
			seen[s.Hash]--
			continue
		}
		add(s.Label)
	}
	kept := map[string]int{}
	for _, s := range b {
		kept[s.Hash]++
	}
	for _, s := range a {
		if kept[s.Hash] > 0 {
			kept[s.Hash]--
			continue
		}
		if len(b) < len(a) {
			add(s.Label)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}
