package accessibility

import (
	"strings"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// FixTextRole makes a Text block a paragraph or a heading (the Settings
// tab's Text role).
const FixTextRole = "text_role"

// ErrMixedText means a Text block holds several paragraphs: a heading gets
// a Text block of its own.
var ErrMixedText = errText(mErrMixedText, nil)

// roleLook is what an element gets from the browser's defaults: its font
// size, vertical margin and weight.
var roleLook = map[string][3]string{
	"p":  {"1em", "1em 0", "inherit"},
	"h1": {"2em", "0.67em 0", "bold"},
	"h2": {"1.5em", "0.83em 0", "bold"},
	"h3": {"1.17em", "1em 0", "bold"},
	"h4": {"1em", "1.33em 0", "bold"},
	"h5": {"0.83em", "1.67em 0", "bold"},
	"h6": {"0.67em", "2.33em 0", "bold"},
}

func blockTag(tag string) bool {
	_, ok := roleLook[tag]
	return ok || tag == "div"
}

// fixTextRole changes the element a Text block's words sit in. The text
// keeps its look: what the old element got from its defaults and the new
// one would change is written into its style.
func fixTextRole(d *doc, n *mjml.Node, role string) (string, error) {
	if _, ok := roleLook[role]; !ok {
		return d.src, errText(mErrChooseRole, nil)
	}
	if n.Tag != "mj-text" {
		return d.src, ErrFixTarget
	}
	s, e, ok := innerRange(d.src, n)
	if !ok {
		return d.src, ErrFixTarget
	}
	content := d.src[s:e]
	var top []*htmlElement
	for _, el := range elementsOf(content) {
		if el.depth == 0 {
			top = append(top, el)
		}
	}
	// Words outside any element, or only inline elements: wrap them.
	blocks := 0
	for _, el := range top {
		if blockTag(el.tag) {
			blocks++
		}
	}
	if blocks == 0 {
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			return d.src, errText(mErrEmptyText, nil)
		}
		// Written out rather than inherited: Outlook gives headings its own
		// size and weight.
		style := "margin:0"
		if role != "p" {
			size := strings.TrimSpace(n.Attr("font-size"))
			if size == "" {
				size = trimFloat(defaultFontSize(d, "mj-text")) + "px"
			}
			weight := strings.TrimSpace(n.Attr("font-weight"))
			if weight == "" {
				weight = "normal"
			}
			style += ";font-size:" + size + ";font-weight:" + weight
		}
		lead := content[:strings.Index(content, trimmed)]
		wrapped := lead + "<" + role + ` style="` + style + `">` + trimmed + "</" + role + ">" + content[len(lead)+len(trimmed):]
		return d.src[:s] + wrapped + d.src[e:], nil
	}
	if blocks > 1 || len(top) > 1 || strings.TrimSpace(textOutside(content, top)) != "" {
		return d.src, ErrMixedText
	}
	el := top[0]
	old := el.tag
	if old == "div" {
		old = "p"
	}
	if old == role && el.tag == role {
		return d.src, nil
	}
	var all []edit
	all = append(all, edit{el.start + 1, el.nameEnd, role})
	if el.closeStart >= 0 {
		all = append(all, edit{el.closeStart + 2, el.closeStart + 2 + len(el.tag), role})
	}
	style := ""
	_, attrs, _, _ := openTag(content, el.start)
	for _, a := range attrs {
		if a.name == "style" {
			style = content[a.valStart:a.valEnd]
		}
	}
	decls := declarations(style)
	has := func(p string) bool {
		for _, dcl := range decls {
			if dcl[0] == p || (p == "margin" && strings.HasPrefix(dcl[0], "margin")) || ((p == "font-size" || p == "font-weight") && dcl[0] == "font") {
				return true
			}
		}
		return false
	}
	from, to := roleLook[old], roleLook[role]
	var extra []string
	if el.tag == "div" {
		// A div has no margin of its own.
		from[1] = "0"
	}
	for i, prop := range []string{"font-size", "margin", "font-weight"} {
		if from[i] != to[i] && !has(prop) {
			extra = append(extra, prop+":"+from[i])
		}
	}
	if len(extra) > 0 {
		next := strings.TrimSpace(style)
		if next != "" && !strings.HasSuffix(next, ";") {
			next += ";"
		}
		next += strings.Join(extra, ";")
		if ed, ok := setAttrEdit(content, el.start, "style", next); ok {
			all = append(all, ed)
		}
	}
	return d.src[:s] + applyEdits(content, all) + d.src[e:], nil
}

// textOutside is the content's text outside the given top-level elements.
func textOutside(content string, top []*htmlElement) string {
	var b strings.Builder
	last := 0
	for _, el := range top {
		end := el.closeEnd
		if end < 0 {
			end = el.tagEnd
		}
		if el.start > last {
			b.WriteString(content[last:el.start])
		}
		last = end
	}
	if last < len(content) {
		b.WriteString(content[last:])
	}
	return b.String()
}
