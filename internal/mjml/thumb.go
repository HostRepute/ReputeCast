package mjml

import (
	"regexp"
	"strconv"
	"strings"
)

// MJMLThumbItem is one part of a template's thumbnail in the gallery, a
// miniature email (Composer and ComposerEmpty boards): a block of height H
// (px) and width W (% of the thumbnail; 0 = full), an accent pill for
// buttons, or a row of blocks for columns. Kind says what it stands for.
type MJMLThumbItem struct {
	Kind   string          `json:"kind,omitempty" enum:"header,image,heading,text,button,footer" doc:"What the part stands for; empty in thumbnails stored before kinds"`
	H      int             `json:"h,omitempty"`
	W      int             `json:"w,omitempty"`
	Accent bool            `json:"accent,omitempty"`
	Row    []MJMLThumbItem `json:"row,omitempty"`
}

// Thumbnail kinds.
const (
	thumbHeader  = "header"
	thumbImage   = "image"
	thumbHeading = "heading"
	thumbText    = "text"
	thumbButton  = "button"
	thumbFooter  = "footer"
)

// headingRe finds a heading in a text: an h1–h3, or a font size of 20 px
// or more on the mj-text.
var headingRe = regexp.MustCompile(`(?i)<h[1-3][\s>]`)

// isHeading tells whether a section's first text reads as a heading.
func isHeading(n *Node) bool {
	x := n.Find("mj-text")
	if x == nil {
		return false
	}
	if size, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(x.Attr("font-size")), "px")); err == nil && size >= 20 {
		return true
	}
	return headingRe.MatchString(x.Inner)
}

// The gallery thumbnail is 120 px tall with 10 px of padding and 6 px
// between bars, so the bars have 100 px.
const (
	thumbHeight = 100
	thumbGap    = 6
)

// height is the space a bar takes in the thumbnail.
func (it MJMLThumbItem) height() int {
	h := it.H
	for _, r := range it.Row {
		h = max(h, r.H)
	}
	return h
}

// Thumb derives the gallery's miniature email from a document: the header
// (logo and links), a block for a hero image, a row of blocks for columns,
// lines for headings and text, a pill for a button, and the footer. The
// header and the footer frame it; between them it keeps at most five parts
// and stops at the first one that doesn't fit, so no part is cut off.
func Thumb(src string) []MJMLThumbItem {
	items := thumbItems(src)
	out := make([]MJMLThumbItem, 0, len(items))
	used, body := 0, 0
	full := false
	for _, it := range items {
		if it.Kind == thumbHeader || it.Kind == thumbFooter {
			out = append(out, it)
			continue
		}
		if full || body == 5 {
			continue
		}
		h := it.height()
		if body > 0 {
			h += thumbGap
		}
		if used+h > thumbHeight {
			full = true
			continue
		}
		used += h
		body++
		out = append(out, it)
	}
	return out
}

func thumbItems(src string) []MJMLThumbItem {
	root, err := Parse(src)
	if err != nil {
		return []MJMLThumbItem{}
	}
	body := Body(root)
	if body == nil {
		return []MJMLThumbItem{}
	}
	out := []MJMLThumbItem{}
	total := len(body.Children)
	textLines := []int{70, 50, 85, 60}
	tl := 0
	for i, n := range body.Children {
		label := SectionLabel(n, i, total)
		cols := 0
		images := 0
		buttons := 0
		for _, c := range n.Children {
			if c.Tag == "mj-column" {
				cols++
			}
		}
		n.Walk(func(x *Node, _ []int) {
			switch x.Tag {
			case "mj-image":
				images++
			case "mj-button":
				buttons++
			}
		})
		switch {
		case label == "header":
			out = append(out, MJMLThumbItem{Kind: thumbHeader, H: 10, W: 40})
		case label == "divider":
			continue
		case label == "footer":
			out = append(out, MJMLThumbItem{Kind: thumbFooter, H: 8})
		case cols >= 2 && images > 0:
			row := make([]MJMLThumbItem, cols)
			for j := range row {
				row[j] = MJMLThumbItem{Kind: thumbImage, H: 28}
			}
			out = append(out, MJMLThumbItem{Row: row})
		case label == "hero" || label == "image" || images > 0:
			out = append(out, MJMLThumbItem{Kind: thumbImage, H: 34})
		case strings.HasSuffix(label, "columns"):
			row := make([]MJMLThumbItem, cols)
			for j := range row {
				row[j] = MJMLThumbItem{Kind: thumbText, H: 18}
			}
			out = append(out, MJMLThumbItem{Row: row})
		case isHeading(n):
			out = append(out, MJMLThumbItem{Kind: thumbHeading, H: 8, W: 80})
		default:
			out = append(out, MJMLThumbItem{Kind: thumbText, H: 8, W: textLines[tl%len(textLines)]})
			tl++
		}
		if buttons > 0 && cols < 2 {
			out = append(out, MJMLThumbItem{Kind: thumbButton, H: 14, W: 35, Accent: true})
		}
	}
	return out
}
