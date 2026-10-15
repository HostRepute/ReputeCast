package languages

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// Block is one top-level block of an email: a section of <mj-body>, the
// unit translations are tracked by. A block of an HTML-only email is the
// whole email.
type Block struct {
	Index int `json:"index"`
	// Tag is the MJML element ("mj-section").
	Tag string `json:"tag"`
	// Label is how the layer tree names it ("header", "2 columns", "text").
	Label string `json:"label"`
	// Name says what it is in the compare view ("Header · logo and links",
	// "Hero image", "Two cards").
	Name string `json:"name"`
	// Heading is the highest heading level inside (1–3), 0 without one.
	Heading  int            `json:"heading"`
	Segments []BlockSegment `json:"segments"`
	// Hash fingerprints what is translated (texts, labels, alt texts):
	// style and link changes don't make a translation outdated.
	Hash string `json:"-"`
	// Raw fingerprints the block's whole source, to match blocks between
	// versions.
	Raw string `json:"-"`
	// Text is a readable summary of the block, columns on their own lines.
	Text string `json:"text"`
	// Start and End are the block's byte range in the source.
	Start int `json:"-"`
	End   int `json:"-"`
}

// BlockSegment is one piece of a block a translator reads or writes: a text, a
// button label, an image's alt text.
type BlockSegment struct {
	// Path is the element index path from the block element.
	Path []int `json:"path"`
	// Kind is text, button, image, social, nav, raw or table.
	Kind string `json:"kind"`
	// Text is the plain text, Liquid kept ("Hi {{ first_name | default: "there" }}").
	Text string `json:"text"`
	// HTML is the inner HTML of a text, to edit it with its links.
	HTML string `json:"html,omitempty"`
	// Strong marks a title (bold or a heading) shown in bold in the compare view.
	Strong bool `json:"strong"`
	// Column is the column the segment sits in (columns are shown on their own lines).
	Column  int    `json:"column"`
	Heading int    `json:"heading"`
	URL     string `json:"url,omitempty"`
	Alt     string `json:"alt,omitempty"`
}

var (
	headingRe = regexp.MustCompile(`(?i)<h([1-6])\b`)
	brRe      = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</h[1-6]>|</li>|</div>`)
	anyTagRe  = regexp.MustCompile(`<[^>]*>`)
	spaceRe   = regexp.MustCompile(`\s+`)
	wordsRe   = regexp.MustCompile(`\S+`)
	boldRe    = regexp.MustCompile(`(?i)^\s*<(b|strong)\b`)
)

// Blocks splits an email into its blocks. Content without MJML (an HTML-only
// email) is one block. A document that doesn't parse is one block too, so
// tracking degrades instead of failing.
func Blocks(mjmlSrc, htmlSrc string) []Block {
	if strings.TrimSpace(mjmlSrc) == "" {
		return htmlBlocks(htmlSrc)
	}
	root, err := mjml.Parse(mjmlSrc)
	if err != nil {
		return htmlBlocks(htmlSrc)
	}
	body := mjml.Body(root)
	if body == nil {
		return []Block{}
	}
	out := make([]Block, 0, len(body.Children))
	firstText := true
	for i, n := range body.Children {
		b := Block{Index: i, Tag: n.Tag, Label: mjml.SectionLabel(n, i, len(body.Children)), Segments: []BlockSegment{}, Start: n.Start, End: n.End}
		collect(n, nil, 0, &b)
		b.Raw = shortHash(mjmlSrc[n.Start:n.End])
		b.Hash = textHash(b.Segments)
		b.Text = summary(b, b.Label == "header")
		for _, s := range b.Segments {
			if s.Heading > 0 && (b.Heading == 0 || s.Heading < b.Heading) {
				b.Heading = s.Heading
			}
		}
		b.Name = blockName(b, n, i, &firstText)
		out = append(out, b)
	}
	return out
}

func htmlBlocks(src string) []Block {
	if strings.TrimSpace(src) == "" {
		return []Block{}
	}
	text := plain(src)
	b := Block{Tag: "html", Label: "email", Name: "The email", Segments: []BlockSegment{{Kind: "raw", Text: text, Path: []int{}}}, Start: 0, End: len(src)}
	b.Raw = shortHash(src)
	b.Hash = textHash(b.Segments)
	b.Text = text
	return []Block{b}
}

// collect walks a block and records its segments.
func collect(n *mjml.Node, p []int, column int, b *Block) {
	switch n.Tag {
	case "mj-column", "mj-group":
		// Columns of a section, or of a group, get their own line.
		if len(p) > 0 && n.Tag == "mj-column" {
			column = p[len(p)-1]
		}
	case "mj-text", "mj-raw", "mj-table", "mj-accordion-title", "mj-accordion-text":
		kind := "text"
		switch n.Tag {
		case "mj-raw":
			kind = "raw"
		case "mj-table":
			kind = "table"
		}
		t := plain(n.Inner)
		if t != "" || kind == "text" {
			s := BlockSegment{Path: clonePath(p), Kind: kind, Text: t, HTML: n.Inner, Column: column}
			if m := headingRe.FindStringSubmatch(n.Inner); m != nil {
				s.Heading, _ = strconv.Atoi(m[1])
			}
			w := strings.TrimSpace(n.Attr("font-weight"))
			s.Strong = s.Heading > 0 || w == "bold" || w == "700" || w == "600" || w == "800" || boldRe.MatchString(n.Inner)
			b.Segments = append(b.Segments, s)
		}
		return
	case "mj-button":
		b.Segments = append(b.Segments, BlockSegment{Path: clonePath(p), Kind: "button", Text: plain(n.Inner), HTML: n.Inner, Column: column, URL: n.Attr("href")})
		return
	case "mj-image":
		b.Segments = append(b.Segments, BlockSegment{Path: clonePath(p), Kind: "image", Alt: strings.TrimSpace(n.Attr("alt")), URL: n.Attr("src"), Column: column})
		return
	case "mj-social-element":
		label := plain(n.Inner)
		if label == "" {
			label = strings.TrimSpace(n.Attr("name"))
		}
		b.Segments = append(b.Segments, BlockSegment{Path: clonePath(p), Kind: "social", Text: label, Alt: strings.TrimSpace(n.Attr("alt")), URL: n.Attr("href"), Column: column})
		return
	case "mj-navbar-link":
		b.Segments = append(b.Segments, BlockSegment{Path: clonePath(p), Kind: "nav", Text: plain(n.Inner), HTML: n.Inner, URL: n.Attr("href"), Column: column})
		return
	}
	for i, c := range n.Children {
		collect(c, append(clonePath(p), i), column, b)
	}
}

func clonePath(p []int) []int {
	out := make([]int, len(p))
	copy(out, p)
	return out
}

// plain turns HTML into one line of text, keeping Liquid tokens.
func plain(s string) string {
	s = brRe.ReplaceAllString(s, " ")
	s = anyTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

// translatable is what a translator writes in a segment.
func (s BlockSegment) translatable() string {
	switch s.Kind {
	case "image":
		return s.Alt
	case "social":
		return s.Text + "\x01" + s.Alt
	}
	return s.Text
}

// textHash fingerprints what a block has to translate; "" when it has
// nothing (an image without alt text, a divider).
func textHash(segs []BlockSegment) string {
	empty := true
	for _, s := range segs {
		if t := strings.Trim(s.translatable(), " \x01"); t != "" {
			empty = false
			break
		}
	}
	if empty {
		return ""
	}
	h := sha256.New()
	for _, s := range segs {
		h.Write([]byte(s.Kind))
		h.Write([]byte{0})
		h.Write([]byte(s.translatable()))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// HashText fingerprints a subject or preheader.
func HashText(s string) string {
	return shortHash(strings.TrimSpace(spaceRe.ReplaceAllString(s, " ")))
}

// summary is the block as one readable text, the way the compare view
// shows it: segments joined with " · ", columns on their own lines.
func summary(b Block, header bool) string {
	cols := map[int][]string{}
	order := []int{}
	for _, s := range b.Segments {
		var t string
		switch s.Kind {
		case "image":
			name := imageName(s.URL)
			switch {
			case s.Alt != "" && header:
				t = `Logo, alt "` + s.Alt + `"`
			case s.Alt != "":
				t = `Image, alt "` + s.Alt + `"`
			case name != "":
				t = name + " · no alt text"
			default:
				t = "Image · no alt text"
			}
		case "social":
			t = s.Text
		default:
			t = s.Text
		}
		if t == "" {
			continue
		}
		if _, ok := cols[s.Column]; !ok {
			order = append(order, s.Column)
		}
		cols[s.Column] = append(cols[s.Column], t)
	}
	lines := make([]string, 0, len(order))
	for _, c := range order {
		lines = append(lines, strings.Join(cols[c], " · "))
	}
	if header {
		// A header's logo and links read as one line.
		return strings.Join(lines, " · ")
	}
	return strings.Join(lines, "\n")
}

// imageName is an image's file name without the library's path.
func imageName(src string) string {
	src = strings.TrimSpace(src)
	if src == "" || strings.HasPrefix(src, "data:") {
		return ""
	}
	if i := strings.IndexAny(src, "?#"); i >= 0 {
		src = src[:i]
	}
	return path.Base(src)
}

var numberWords = []string{"", "One", "Two", "Three", "Four", "Five", "Six"}

// blockName names a block for the compare view.
func blockName(b Block, n *mjml.Node, index int, firstText *bool) string {
	var texts, images, buttons, socials, navs, others int
	for _, s := range b.Segments {
		switch s.Kind {
		case "text":
			texts++
		case "image":
			images++
		case "button":
			buttons++
		case "social":
			socials++
		case "nav":
			navs++
		default:
			others++
		}
	}
	cols := 0
	for _, c := range n.Children {
		if c.Tag == "mj-column" || c.Tag == "mj-group" {
			cols++
		}
	}
	name := ""
	switch {
	case b.Label == "header" || n.Marker == "header-logo" || n.Marker == "header":
		name = "Header"
		if images > 0 && (texts > 0 || navs > 0) {
			name = "Header · logo and links"
		}
	case b.Label == "footer" || n.Marker == "footer":
		name = "Footer"
	case n.Tag == "mj-raw" || (others > 0 && texts+images+buttons == 0):
		name = "Raw HTML"
	case socials > 0 && texts == 0:
		name = "Social links"
	case cols >= 2 && images >= cols && texts > 0:
		name = cardsName(cols)
	case cols >= 2:
		name = strconv.Itoa(cols) + " columns"
	case b.Label == "hero" || n.Tag == "mj-hero":
		name = "Hero"
	case images > 0 && texts == 0 && buttons == 0:
		name = "Image"
		if index <= 2 {
			name = "Hero image"
		}
	case buttons > 0 && texts == 0:
		name = "Button"
	case texts > 0:
		first := b.Segments[0]
		for _, s := range b.Segments {
			if s.Kind == "text" {
				first = s
				break
			}
		}
		switch {
		case texts == 1 && images == 0 && (first.Heading == 1 || fontSize(n) >= 24):
			name = "Headline"
		case *firstText && images == 0 && first.Heading == 0:
			name = "Intro"
		default:
			name = `Text · "` + firstWords(first.Text, 5) + `"`
		}
		if name != "Headline" {
			*firstText = false
		}
	default:
		name = strings.ToUpper(b.Label[:1]) + b.Label[1:]
	}
	if b.Heading > 0 {
		name += " · Heading " + strconv.Itoa(b.Heading)
	}
	return name
}

func cardsName(cols int) string {
	if cols < len(numberWords) {
		return numberWords[cols] + " cards"
	}
	return strconv.Itoa(cols) + " cards"
}

// fontSize is the largest font size of the texts in a block.
func fontSize(n *mjml.Node) int {
	size := 0
	n.Walk(func(x *mjml.Node, _ []int) {
		if x.Tag == "mj-text" {
			v := strings.TrimSuffix(strings.TrimSpace(x.Attr("font-size")), "px")
			if f, err := strconv.ParseFloat(v, 64); err == nil && int(f) > size {
				size = int(f)
			}
		}
	})
	return size
}

// firstWords shortens a text to its first n words.
func firstWords(s string, n int) string {
	w := wordsRe.FindAllString(s, n+1)
	if len(w) <= n {
		return strings.Join(w, " ")
	}
	return strings.TrimRight(strings.Join(w[:n], " "), ",.;:") + "…"
}
