package accessibility

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The walk reads the compiled HTML the way a mail client paints it, from
// inline styles and presentational attributes (the way MJML writes every
// style that matters): the color, size and weight of each piece of text,
// the nearest background color behind it, links, images and headings.

// state is what an element passes to its children.
type state struct {
	color Color
	size  float64
	bold  bool
	// bg is the nearest opaque background; overImage is set when a
	// background image is closer to the text than any solid color.
	bg        Color
	overImage bool
	path      string
}

// textRun is one visible text node.
type textRun struct {
	path      string
	text      string
	fg, bg    Color
	overImage bool
	size      float64
	bold      bool
	link      *linkInfo
	button    bool
}

type linkInfo struct {
	path  string
	index int
	href  string
	text  strings.Builder
	alts  []string
	label string
	imgs  []*imgInfo
}

func (l *linkInfo) name() string {
	parts := []string{strings.TrimSpace(collapse(l.text.String()))}
	for _, a := range l.alts {
		if strings.TrimSpace(a) != "" {
			parts = append(parts, strings.TrimSpace(a))
		}
	}
	if s := strings.TrimSpace(strings.Join(parts, " ")); s != "" {
		return s
	}
	return strings.TrimSpace(l.label)
}

type imgInfo struct {
	path   string
	index  int
	src    string
	alt    string
	hasAlt bool
	link   *linkInfo
}

type headingInfo struct {
	path  string
	index int
	level int
	text  strings.Builder
	size  float64
}

type walker struct {
	runs     []*textRun
	links    []*linkInfo
	images   []*imgInfo
	headings []*headingInfo
	counts   map[string]map[string]int
	link     *linkInfo
	heading  *headingInfo
}

func (w *walker) next(path, kind string) int {
	if w.counts == nil {
		w.counts = map[string]map[string]int{}
	}
	m := w.counts[path]
	if m == nil {
		m = map[string]int{}
		w.counts[path] = m
	}
	i := m[kind]
	m[kind]++
	return i
}

// walkHTML parses compiled HTML and collects what the checks read.
func walkHTML(src string) *walker {
	w := &walker{}
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return w
	}
	w.visit(doc, state{color: black, size: 16, bg: white})
	return w
}

// declarations parses an inline style into lower-cased properties, in
// order (a later declaration wins).
func declarations(style string) [][2]string {
	var out [][2]string
	for _, part := range strings.Split(style, ";") {
		k, v, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		if k != "" && v != "" {
			out = append(out, [2]string{k, v})
		}
	}
	return out
}

func attrOf(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

var headingSize = map[atom.Atom]float64{atom.H1: 2, atom.H2: 1.5, atom.H3: 1.17, atom.H4: 1, atom.H5: 0.83, atom.H6: 0.67}

func headingLevel(a atom.Atom) int {
	switch a {
	case atom.H1:
		return 1
	case atom.H2:
		return 2
	case atom.H3:
		return 3
	case atom.H4:
		return 4
	case atom.H5:
		return 5
	case atom.H6:
		return 6
	}
	return 0
}

// cssLength reads a font size relative to the parent's ("13px", "10pt",
// "1.2em", "120%", "small").
func cssLength(v string, parent float64) (float64, bool) {
	s := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "!important")))
	s = strings.TrimSpace(s)
	switch s {
	case "xx-small":
		return 9, true
	case "x-small":
		return 10, true
	case "small":
		return 13, true
	case "medium":
		return 16, true
	case "large":
		return 18, true
	case "x-large":
		return 24, true
	case "xx-large":
		return 32, true
	case "smaller":
		return parent / 1.2, true
	case "larger":
		return parent * 1.2, true
	case "inherit", "initial", "unset", "":
		return parent, false
	}
	units := []struct {
		suffix string
		scale  float64
		rel    bool
	}{{"px", 1, false}, {"pt", 4.0 / 3, false}, {"rem", 16, false}, {"em", 1, true}, {"%", 0.01, true}}
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			f, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(s, u.suffix)), 64)
			if err != nil {
				return parent, false
			}
			if u.rel {
				return f * u.scale * parent, true
			}
			return f * u.scale, true
		}
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, true
	}
	return parent, false
}

func boldWeight(v string) (bool, bool) {
	s := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "!important")))
	switch s {
	case "bold", "bolder":
		return true, true
	case "normal", "lighter":
		return false, true
	}
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n >= 700, true
	}
	return false, false
}

// fontSizes are the sizes of <font size="1"> to "7".
var fontSizes = []float64{10, 13, 16, 18, 24, 32, 48}

func hasURL(v string) bool { return strings.Contains(strings.ToLower(v), "url(") }

// firstColor finds a color among the tokens of a background shorthand.
func firstColor(v string) (Color, bool) {
	if c, ok := ParseColor(v); ok {
		return c, true
	}
	// Functions keep their spaces: split outside parentheses.
	var tokens []string
	depth, start := 0, 0
	for i, r := range v {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case unicode.IsSpace(r) && depth == 0:
			tokens = append(tokens, v[start:i])
			start = i + 1
		}
	}
	tokens = append(tokens, v[start:])
	for _, t := range tokens {
		if c, ok := ParseColor(t); ok {
			return c, true
		}
	}
	return Color{}, false
}

func (w *walker) visit(n *html.Node, st state) {
	switch n.Type {
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			w.visit(c, st)
		}
		return
	case html.TextNode:
		w.text(n, st)
		return
	case html.ElementNode:
	default:
		return
	}
	switch n.DataAtom {
	case atom.Head, atom.Style, atom.Script, atom.Title, atom.Template, atom.Noscript:
		return
	}
	if p, ok := pathOfElement(n); ok {
		st.path = p
	}
	hidden := false
	maxHeightZero, overflowHidden := false, false
	// Presentational attributes first; inline styles override them.
	if v, ok := attrOf(n, "bgcolor"); ok {
		if c, ok := ParseColor(v); ok && c.A > 0 {
			st.bg, st.overImage = c.Over(st.bg), false
		}
	}
	if v, ok := attrOf(n, "background"); ok && strings.TrimSpace(v) != "" {
		st.overImage = true
	}
	if n.DataAtom == atom.Font {
		if v, ok := attrOf(n, "color"); ok {
			if c, ok := ParseColor(v); ok {
				st.color = c
			}
		}
		if v, ok := attrOf(n, "size"); ok {
			if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && i >= 1 && i <= 7 {
				st.size = fontSizes[i-1]
			}
		}
	}
	if f, ok := headingSize[n.DataAtom]; ok {
		st.size *= f
		st.bold = true
	}
	switch n.DataAtom {
	case atom.B, atom.Strong, atom.Th:
		st.bold = true
	case atom.Small:
		st.size /= 1.2
	case atom.Big:
		st.size *= 1.2
	}
	isLink := false
	inherited := st.color
	if n.DataAtom == atom.A {
		if _, ok := attrOf(n, "href"); ok {
			isLink = true
			// Links set no color of their own get the client's link blue.
			st.color = defaultLink
		}
	}
	if style, ok := attrOf(n, "style"); ok {
		for _, d := range declarations(style) {
			k, v := d[0], d[1]
			switch k {
			case "color":
				if c, ok := ParseColor(v); ok {
					st.color = c
				} else if lv := strings.ToLower(strings.TrimSpace(v)); strings.HasPrefix(lv, "inherit") || strings.HasPrefix(lv, "currentcolor") {
					st.color = inherited
				}
			case "font-size":
				if s, ok := cssLength(v, st.size); ok {
					st.size = s
				}
			case "font-weight":
				if b, ok := boldWeight(v); ok {
					st.bold = b
				}
			case "font":
				for _, tok := range strings.Fields(v) {
					if b, ok := boldWeight(tok); ok && b {
						st.bold = true
					}
					if strings.ContainsAny(tok, "0123456789") && (strings.HasSuffix(tok, "px") || strings.HasSuffix(tok, "pt") || strings.HasSuffix(tok, "em")) {
						if s, ok := cssLength(strings.Split(tok, "/")[0], st.size); ok {
							st.size = s
						}
					}
				}
			case "background", "background-color", "background-image":
				if hasURL(v) {
					st.overImage = true
					continue
				}
				if c, ok := firstColor(v); ok && c.A > 0 {
					st.bg = c.Over(st.bg)
					if c.Opaque() {
						st.overImage = false
					}
				}
			case "display":
				if strings.HasPrefix(strings.ToLower(v), "none") {
					hidden = true
				}
			case "visibility":
				if strings.HasPrefix(strings.ToLower(v), "hidden") {
					hidden = true
				}
			case "opacity":
				if f, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(v, "!important")), 64); err == nil && f == 0 {
					hidden = true
				}
			case "max-height":
				if s, ok := cssLength(v, st.size); ok && s == 0 {
					maxHeightZero = true
				}
			case "overflow":
				if strings.HasPrefix(strings.ToLower(v), "hidden") {
					overflowHidden = true
				}
			}
		}
	}
	if maxHeightZero && overflowHidden {
		hidden = true
	}
	if _, ok := attrOf(n, "hidden"); ok {
		hidden = true
	}
	if hidden {
		return
	}
	switch {
	case n.DataAtom == atom.Img:
		w.image(n, st)
		return
	case isLink && w.link == nil:
		href, _ := attrOf(n, "href")
		l := &linkInfo{path: st.path, href: strings.TrimSpace(href), index: w.next(st.path, "a")}
		if v, ok := attrOf(n, "aria-label"); ok {
			l.label = v
		} else if v, ok := attrOf(n, "title"); ok {
			l.label = v
		}
		w.links = append(w.links, l)
		w.link = l
		defer func() { w.link = nil }()
	}
	if lvl := headingLevel(n.DataAtom); lvl > 0 && w.heading == nil {
		h := &headingInfo{path: st.path, level: lvl, index: w.next(st.path, "h"), size: st.size}
		w.headings = append(w.headings, h)
		w.heading = h
		defer func() { w.heading = nil }()
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		w.visit(c, st)
	}
}

func (w *walker) text(n *html.Node, st state) {
	t := n.Data
	if w.link != nil {
		w.link.text.WriteString(t)
	}
	if w.heading != nil {
		w.heading.text.WriteString(t)
	}
	if !hasWords(t) || st.size < 1 {
		return
	}
	fg := st.color.Over(st.bg)
	w.runs = append(w.runs, &textRun{path: st.path, text: collapse(t), fg: fg, bg: st.bg, overImage: st.overImage, size: st.size, bold: st.bold, link: w.link})
}

func (w *walker) image(n *html.Node, st state) {
	wd, _ := attrOf(n, "width")
	ht, _ := attrOf(n, "height")
	if pxOf(wd) > 0 && pxOf(wd) <= 2 && pxOf(ht) > 0 && pxOf(ht) <= 2 {
		return // a tracking pixel
	}
	alt, has := attrOf(n, "alt")
	src, _ := attrOf(n, "src")
	im := &imgInfo{path: st.path, index: w.next(st.path, "img"), src: src, alt: alt, hasAlt: has, link: w.link}
	if w.link != nil {
		w.link.imgs = append(w.link.imgs, im)
		w.link.alts = append(w.link.alts, alt)
	}
	w.images = append(w.images, im)
}

func pxOf(v string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(v), "px"), 64)
	if err != nil {
		return 0
	}
	return f
}

// hasWords reports whether text has a letter or digit (separators such as
// "·" and spacing characters don't count as text to read).
func hasWords(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// roundPx is a font size as people write it: 11, 12.5.
func roundPx(v float64) float64 { return math.Round(v*10) / 10 }
