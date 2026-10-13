package mjml

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Lint returns layout warnings the MJML compiler doesn't report: images
// wider than their column (Outlook draws them at full width and breaks the
// layout) and columns adding up to more than their section.
func Lint(src string) []MJMLProblem {
	out := []MJMLProblem{}
	root, err := Parse(src)
	if err != nil {
		return out
	}
	body := Body(root)
	if body == nil {
		return out
	}
	bodyWidth := px(body.Attr("width"))
	if bodyWidth == 0 {
		bodyWidth = 600
	}
	var section func(n *Node, width float64)
	section = func(n *Node, width float64) {
		l, r := hpadding(n, "padding", 0)
		inner := width - l - r
		var cols []*Node
		for _, c := range n.Children {
			switch c.Tag {
			case "mj-column":
				cols = append(cols, c)
			case "mj-group":
				gw := colWidth(c, inner, 1)
				section(c, gw)
			case "mj-section":
				section(c, inner)
			}
		}
		if len(cols) == 0 {
			return
		}
		sumPct := 0.0
		sized := 0
		for _, c := range cols {
			w := c.Attr("width")
			if strings.HasSuffix(strings.TrimSpace(w), "%") {
				v, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(w), "%"), 64)
				sumPct += v
				sized++
			} else if px(w) > 0 {
				sumPct += float64(px(w)) / inner * 100
				sized++
			}
		}
		if sumPct > 100.5 {
			out = append(out, MJMLProblem{Line: n.Line, Tag: n.Tag, Message: fmt.Sprintf("Line %d: the columns of this section add up to %.0f%%; MJML squeezes them and the layout shifts. Make them add up to 100%%.", n.Line, sumPct)})
		}
		free := len(cols) - sized
		for _, c := range cols {
			cw := colWidth(c, inner, 0)
			if cw == 0 && free > 0 {
				cw = inner * math.Max(0, 100-sumPct) / 100 / float64(free)
			}
			pl, pr := hpadding(c, "padding", 0)
			box := cw - pl - pr
			label := c.Attr("width")
			if label == "" {
				label = fmt.Sprintf("%.0f%%", cw/inner*100)
			}
			c.Walk(func(x *Node, _ []int) {
				if x.Tag != "mj-image" {
					return
				}
				iw := px(x.Attr("width"))
				il, ir := hpadding(x, "padding", 25)
				avail := box - il - ir
				if iw > 0 && float64(iw) > avail+0.5 {
					out = append(out, MJMLProblem{Line: x.Line, Tag: "mj-image", Message: fmt.Sprintf("Line %d: mj-image width %dpx exceeds the %s column (%.0fpx); it will be scaled down. Set width=\"%.0fpx\" or remove it.", x.Line, iw, label, cw, math.Floor(avail))})
				}
			})
		}
	}
	for _, n := range body.Children {
		switch n.Tag {
		case "mj-section", "mj-wrapper":
			section(n, float64(bodyWidth))
		}
	}
	return out
}

func colWidth(c *Node, inner float64, def float64) float64 {
	w := strings.TrimSpace(c.Attr("width"))
	switch {
	case strings.HasSuffix(w, "%"):
		v, _ := strconv.ParseFloat(strings.TrimSuffix(w, "%"), 64)
		return inner * v / 100
	case px(w) > 0:
		return float64(px(w))
	}
	return inner * def
}

// hpadding returns the left and right padding of an element from its
// padding shorthand and padding-left/right, with def for unset sides.
func hpadding(n *Node, attr string, def float64) (float64, float64) {
	l, r := def, def
	if v, ok := n.Get(attr); ok {
		parts := strings.Fields(v)
		vals := make([]float64, len(parts))
		for i, p := range parts {
			vals[i] = float64(px(p))
		}
		switch len(vals) {
		case 1:
			l, r = vals[0], vals[0]
		case 2, 3:
			l, r = vals[1], vals[1]
		case 4:
			r, l = vals[1], vals[3]
		}
	}
	if v, ok := n.Get(attr + "-left"); ok {
		l = float64(px(v))
	}
	if v, ok := n.Get(attr + "-right"); ok {
		r = float64(px(v))
	}
	return l, r
}
