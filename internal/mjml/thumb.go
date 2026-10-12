package mjml

import "strings"

// ThumbItem is one bar of a template's wireframe thumbnail in the gallery:
// a grey block of height H (px) and width W (% of the thumbnail; 0 = full),
// an accent pill for buttons, or a row of blocks for columns.
type ThumbItem struct {
	H      int         `json:"h,omitempty"`
	W      int         `json:"w,omitempty"`
	Accent bool        `json:"accent,omitempty"`
	Row    []ThumbItem `json:"row,omitempty"`
}

// Thumb derives the gallery wireframe from a document: a short bar for the
// header, a block for a hero image, a row of blocks for columns, lines for
// text and a pill for a button. It keeps at most five items so it fits the
// 120 px thumbnail.
func Thumb(src string) []ThumbItem {
	root, err := Parse(src)
	if err != nil {
		return []ThumbItem{}
	}
	body := Body(root)
	if body == nil {
		return []ThumbItem{}
	}
	out := []ThumbItem{}
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
			out = append(out, ThumbItem{H: 10, W: 40})
		case label == "divider":
			continue
		case label == "footer":
			out = append(out, ThumbItem{H: 8})
		case cols >= 2 && images > 0:
			row := make([]ThumbItem, cols)
			for j := range row {
				row[j] = ThumbItem{H: 28}
			}
			out = append(out, ThumbItem{Row: row})
		case label == "hero" || label == "image" || images > 0:
			out = append(out, ThumbItem{H: 34})
		case strings.HasSuffix(label, "columns"):
			row := make([]ThumbItem, cols)
			for j := range row {
				row[j] = ThumbItem{H: 18}
			}
			out = append(out, ThumbItem{Row: row})
		default:
			out = append(out, ThumbItem{H: 8, W: textLines[tl%len(textLines)]})
			tl++
		}
		if buttons > 0 && len(out) < 5 && cols < 2 {
			out = append(out, ThumbItem{H: 14, W: 35, Accent: true})
		}
	}
	return out
}
