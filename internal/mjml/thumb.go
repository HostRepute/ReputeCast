package mjml

import "strings"

// MJMLThumbItem is one bar of a template's wireframe thumbnail in the gallery:
// a grey block of height H (px) and width W (% of the thumbnail; 0 = full),
// an accent pill for buttons, or a row of blocks for columns.
type MJMLThumbItem struct {
	H      int             `json:"h,omitempty"`
	W      int             `json:"w,omitempty"`
	Accent bool            `json:"accent,omitempty"`
	Row    []MJMLThumbItem `json:"row,omitempty"`
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

// Thumb derives the gallery wireframe from a document: a short bar for the
// header, a block for a hero image, a row of blocks for columns, lines for
// text and a pill for a button. It keeps at most five items and stops at the
// first one that doesn't fit the thumbnail, so no bar is cut off.
func Thumb(src string) []MJMLThumbItem {
	items := thumbItems(src)
	used := 0
	for i, it := range items {
		if i > 0 {
			used += thumbGap
		}
		used += it.height()
		if used > thumbHeight {
			return items[:i]
		}
	}
	return items
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
		if len(out) >= 5 {
			break
		}
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
			out = append(out, MJMLThumbItem{H: 10, W: 40})
		case label == "divider":
			continue
		case label == "footer":
			out = append(out, MJMLThumbItem{H: 8})
		case cols >= 2 && images > 0:
			row := make([]MJMLThumbItem, cols)
			for j := range row {
				row[j] = MJMLThumbItem{H: 28}
			}
			out = append(out, MJMLThumbItem{Row: row})
		case label == "hero" || label == "image" || images > 0:
			out = append(out, MJMLThumbItem{H: 34})
		case strings.HasSuffix(label, "columns"):
			row := make([]MJMLThumbItem, cols)
			for j := range row {
				row[j] = MJMLThumbItem{H: 18}
			}
			out = append(out, MJMLThumbItem{Row: row})
		default:
			out = append(out, MJMLThumbItem{H: 8, W: textLines[tl%len(textLines)]})
			tl++
		}
		if buttons > 0 && len(out) < 5 && cols < 2 {
			out = append(out, MJMLThumbItem{H: 14, W: 35, Accent: true})
		}
	}
	return out
}
