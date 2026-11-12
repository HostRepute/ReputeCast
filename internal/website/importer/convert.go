package importer

import (
	"encoding/json"
	"html"
	"net/url"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// Convert writes a post's HTML as a web post's MJML: text (paragraphs,
// headings, lists, quotes, code, tables, captioned figures) in mj-text
// blocks, cleaned by the site's own allowlist (render.Sanitize); each
// image an mj-image, its src through rewrite (false keeps it and reports
// it), an empty alt reported; hr an mj-divider; buttons mj-button. An
// embed (YouTube, Vimeo, X, Instagram, Spotify, a Substack or Ghost card)
// becomes a paragraph linking to it, reported. A paywall marker (Substack's
// paywall-jump, Ghost's <!--members-only-->) becomes the free preview
// marker between blocks. Liquid in the text is written as entities, so
// "{{" stays text. The links' words are English; ConvertIn picks the
// language.
func Convert(fragment string, rewrite func(src string) (string, bool)) (string, []Problem) {
	return ConvertIn(i18n.Default, fragment, rewrite)
}

// ConvertIn is Convert with the embed links worded in l, the site's
// language.
func ConvertIn(l i18n.Lang, fragment string, rewrite func(src string) (string, bool)) (string, []Problem) {
	c := &converter{l: l, rewrite: rewrite, seen: map[Problem]bool{}, problems: []Problem{}}
	nodes, err := xhtml.ParseFragment(strings.NewReader(fragment), bodyContext())
	if err != nil {
		nodes = []*xhtml.Node{{Type: xhtml.TextNode, Data: fragment}}
	}
	for _, n := range nodes {
		c.node(n)
	}
	c.closeInline()
	c.flushText()
	var b strings.Builder
	b.WriteString("<mjml>\n<mj-body width=\"680px\">\n<mj-section>\n<mj-column>\n")
	for _, blk := range c.blocks {
		b.WriteString(blk)
		b.WriteByte('\n')
	}
	b.WriteString("</mj-column>\n</mj-section>\n</mj-body>\n</mjml>\n")
	return b.String(), c.problems
}

// Images lists the images Convert would rewrite, in order, each once.
func Images(fragment string) []string {
	out := []string{}
	seen := map[string]bool{}
	Convert(fragment, func(src string) (string, bool) {
		if !seen[src] {
			seen[src] = true
			out = append(out, src)
		}
		return src, true
	})
	return out
}

// markerBlock is the free preview marker, as the composer writes it.
const markerBlock = `<mj-raw rc-preview="end">` + render.PreviewComment + `</mj-raw>`

type converter struct {
	l       i18n.Lang
	rewrite func(string) (string, bool)
	blocks  []string
	// text is the HTML of the text block being gathered, inl the inline
	// content at the top level waiting for its paragraph.
	text, inl strings.Builder
	marked    bool
	problems  []Problem
	seen      map[Problem]bool
}

func (c *converter) problem(code, detail string) {
	p := Problem{Code: code, Detail: detail}
	if !c.seen[p] {
		c.seen[p] = true
		c.problems = append(c.problems, p)
	}
}

func bodyContext() *xhtml.Node {
	return &xhtml.Node{Type: xhtml.ElementNode, Data: "body", DataAtom: atom.Body}
}

// node places one node of the post.
func (c *converter) node(n *xhtml.Node) {
	switch n.Type {
	case xhtml.CommentNode:
		c.comment(n.Data)
		return
	case xhtml.TextNode:
		if c.inl.Len() > 0 || strings.TrimSpace(n.Data) != "" {
			c.inl.WriteString(html.EscapeString(n.Data))
		}
		return
	case xhtml.ElementNode:
	default:
		return
	}
	if dropped(n) || widget(n) {
		return
	}
	if c.card(n) {
		return
	}
	switch n.DataAtom {
	case atom.Iframe, atom.Embed, atom.Object:
		c.embed(firstOf(attr(n, "src"), attr(n, "data-src"), attr(n, "data")))
	case atom.Video, atom.Audio:
		c.media(n)
	case atom.Img:
		c.closeInline()
		c.image(n, "", "")
	case atom.Picture:
		if img := find(n, isTag(atom.Img)); img != nil {
			c.closeInline()
			c.image(img, "", "")
		}
	case atom.Hr:
		c.closeInline()
		c.flushText()
		c.blocks = append(c.blocks, "<mj-divider></mj-divider>")
	case atom.Figure:
		c.closeInline()
		c.figure(n)
	case atom.P, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Blockquote, atom.Pre, atom.Ul, atom.Ol, atom.Table, atom.Dl:
		c.closeInline()
		c.textBlock(n)
	case atom.A:
		if onlyImages(n) {
			c.closeInline()
			c.images(n, linkOf(n), "")
			return
		}
		c.inline(n)
	default:
		if blockLevel(n) || find(n, isEmbed) != nil {
			c.closeInline()
			c.walk(n)
			return
		}
		c.inline(n)
	}
}

func (c *converter) walk(n *xhtml.Node) {
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		c.node(k)
	}
}

// comment reads the notes posts keep in comments: Ghost's paywall marker,
// and the shortcodes a WordPress post couldn't keep (wpContent).
func (c *converter) comment(data string) {
	d := strings.TrimSpace(data)
	switch {
	case d == "members-only":
		c.marker()
	case strings.HasPrefix(d, "rc:shortcode "):
		c.problem(ProblemShortcode, strings.TrimPrefix(d, "rc:shortcode "))
	}
}

func (c *converter) marker() {
	c.closeInline()
	c.flushText()
	if !c.marked {
		c.marked = true
		c.blocks = append(c.blocks, markerBlock)
	}
}

// card handles the blocks platforms draw with their own markup: Substack's
// paywall, buttons, tweets, Instagram posts, YouTube players and post
// cards; Ghost's buttons, bookmarks and embeds; WordPress's embeds.
func (c *converter) card(n *xhtml.Node) bool {
	switch {
	case hasClass(n, "paywall-jump"):
		c.marker()
	case hasClass(n, "button-wrapper"), hasClass(n, "kg-button-card"):
		c.buttons(n)
	case hasClass(n, "tweet"):
		c.embed(linkIn(n, "tweet-link-top"))
	case n.DataAtom == atom.Div && hasClass(n, "instagram"):
		c.embed(linkIn(n, "instagram-image"))
	case hasClass(n, "youtube-wrap"):
		var a struct {
			VideoID string `json:"videoId"`
		}
		if json.Unmarshal([]byte(attr(n, "data-attrs")), &a) == nil && a.VideoID != "" {
			c.embed("https://www.youtube.com/watch?v=" + url.QueryEscape(a.VideoID))
		} else if f := find(n, isTag(atom.Iframe)); f != nil {
			c.embed(attr(f, "src"))
		}
	case hasClass(n, "embedded-post-wrap"):
		c.embed(firstOf(linkIn(n, "embedded-post"), attrJSON(n, "url")))
	case hasClass(n, "kg-bookmark-card"):
		c.embed(linkIn(n, "kg-bookmark-container"))
	case hasClass(n, "wp-block-embed"), hasClass(n, "kg-embed-card"):
		switch {
		case find(n, isTag(atom.Iframe)) != nil:
			c.embed(attr(find(n, isTag(atom.Iframe)), "src"))
		case find(n, func(x *xhtml.Node) bool { return hasClass(x, "twitter-tweet") }) != nil:
			q := find(n, func(x *xhtml.Node) bool { return hasClass(x, "twitter-tweet") })
			c.embed(lastLink(q))
		default:
			var b strings.Builder
			collectText(&b, n)
			u := strings.TrimSpace(b.String())
			if strings.HasPrefix(u, "http") && !strings.ContainsAny(u, " \n") {
				c.embed(u)
			} else {
				return false
			}
		}
	default:
		return false
	}
	return true
}

// buttons makes mj-buttons of a button card's links, leaving out the old
// platform's own (subscribe, share, comments, the member portal).
func (c *converter) buttons(n *xhtml.Node) {
	for _, a := range findAll(n, isTag(atom.A)) {
		href := linkOf(a)
		label := textOf(renderNode(a))
		if href == "" || label == "" || platformLink(href) {
			continue
		}
		c.closeInline()
		c.flushText()
		c.blocks = append(c.blocks, `<mj-button href="`+attrEsc(href)+`">`+neutralize(html.EscapeString(label))+`</mj-button>`)
	}
}

// platformLink is a link to the old platform's own pages: subscribing,
// sharing, comments, Ghost's member portal.
func platformLink(href string) bool {
	h := strings.ToLower(href)
	return strings.Contains(h, "/subscribe") || strings.Contains(h, "share") || strings.Contains(h, "/comments") || strings.Contains(h, "#/portal")
}

// embed writes a paragraph linking to an embed.
func (c *converter) embed(raw string) {
	raw = strings.TrimSpace(html.UnescapeString(raw))
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return
	}
	link := embedLink(u)
	p, _ := platformOf(link)
	c.linkParagraph("embed."+p.Verb, p.Name, link)
	c.problem(ProblemEmbed, p.Name)
}

// media writes a paragraph linking to a video or an audio file.
func (c *converter) media(n *xhtml.Node) {
	src := attr(n, "src")
	if src == "" {
		if s := find(n, isTag(atom.Source)); s != nil {
			src = attr(s, "src")
		}
	}
	src = strings.TrimSpace(src)
	if strings.HasPrefix(src, "//") {
		src = "https:" + src
	}
	u, err := url.Parse(src)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return
	}
	key := "embed.video"
	if n.DataAtom == atom.Audio {
		key = "embed.audio"
	}
	c.linkParagraph(key, hostOf(u), u.String())
	c.problem(ProblemEmbed, hostOf(u))
}

func (c *converter) linkParagraph(key, name, link string) {
	c.closeInline()
	words := msgs.M(key).HTML(c.l, i18n.Args{"platform": name, "url": link}, i18n.Tags{
		"link": func(inner string) string { return `<a href="` + html.EscapeString(link) + `">` + inner + `</a>` }})
	c.text.WriteString("<p>" + words + "</p>")
}

// figure places a figure: its images as mj-images with the caption after
// them; one without an image (a code card, a quote) as text.
func (c *converter) figure(n *xhtml.Node) {
	if find(n, isEmbed) != nil {
		c.walk(n)
		return
	}
	if find(n, isTag(atom.Img)) == nil {
		c.textBlock(n)
		return
	}
	caption := ""
	if fc := find(n, isTag(atom.Figcaption)); fc != nil {
		caption = strings.TrimSpace(renderChildren(fc))
	}
	c.images(n, "", caption)
}

// images places the images inside n, each linked to href or the link
// wrapping it, and the caption after the last.
func (c *converter) images(n *xhtml.Node, href, caption string) {
	imgs := findAll(n, isTag(atom.Img))
	for i, img := range imgs {
		link := href
		if a := closest(img, atom.A, n); a != nil {
			link = linkOf(a)
		}
		cap := ""
		if i == len(imgs)-1 {
			cap = caption
		}
		c.image(img, link, cap)
	}
}

// image adds an mj-image.
func (c *converter) image(img *xhtml.Node, href, caption string) {
	src := c.src(img)
	if src == "" {
		return
	}
	c.flushText()
	alt := strings.TrimSpace(attr(img, "alt"))
	b := `<mj-image src="` + attrEsc(src) + `" alt="` + attrEsc(alt) + `"`
	if href != "" && !sameAddress(href, attr(img, "src")) && !strings.Contains(href, "substackcdn.com/image") {
		b += ` href="` + attrEsc(href) + `"`
	}
	if w, err := strconv.Atoi(strings.TrimSuffix(attr(img, "width"), "px")); err == nil && w > 0 && w < 680 {
		b += ` width="` + strconv.Itoa(w) + `px"`
	}
	c.blocks = append(c.blocks, b+`></mj-image>`)
	if caption != "" {
		c.text.WriteString("<p><em>" + caption + "</em></p>")
	}
}

// src is an image's address for the post: rewritten (copied), reported
// when it couldn't be, and reported for an empty alt. "" leaves the image
// out: one with no address the site may serve.
func (c *converter) src(img *xhtml.Node) string {
	orig := strings.TrimSpace(firstOf(attr(img, "src"), attr(img, "data-src")))
	if orig == "" {
		return ""
	}
	if strings.HasPrefix(orig, "//") {
		orig = "https:" + orig
	}
	src := orig
	if c.rewrite != nil {
		if out, ok := c.rewrite(orig); ok {
			src = out
		} else {
			c.problem(ProblemImage, orig)
		}
	}
	if !servable(src) {
		c.problem(ProblemImage, orig)
		return ""
	}
	if strings.TrimSpace(attr(img, "alt")) == "" {
		c.problem(ProblemAlt, orig)
	}
	return src
}

// servable is an address the site serves images from: http(s), or a path
// on the site.
func servable(src string) bool {
	if strings.HasPrefix(src, "/") && !strings.HasPrefix(src, "//") {
		return true
	}
	u, err := url.Parse(src)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func sameAddress(a, b string) bool {
	return strings.TrimSpace(html.UnescapeString(a)) == strings.TrimSpace(html.UnescapeString(b))
}

// textBlock adds a paragraph, heading, list, quote, code or table to the
// text being gathered: its images rewritten in place, its embeds after it
// as links, an h1 as h2 (the post's title is the page's h1). A paragraph
// holding nothing but images is images.
func (c *converter) textBlock(n *xhtml.Node) {
	if onlyImages(n) {
		c.images(n, "", "")
		return
	}
	var embeds []*xhtml.Node
	for _, e := range findAll(n, isEmbed) {
		e.Parent.RemoveChild(e)
		embeds = append(embeds, e)
	}
	c.prepare(n)
	if n.DataAtom == atom.H1 {
		n.DataAtom, n.Data = atom.H2, "h2"
	}
	c.text.WriteString(renderNode(n))
	for _, e := range embeds {
		if e.DataAtom == atom.Video || e.DataAtom == atom.Audio {
			c.media(e)
		} else {
			c.embed(firstOf(attr(e, "src"), attr(e, "data")))
		}
	}
}

// inline gathers inline content at the top level (text, links, emphasis)
// into the paragraph closeInline writes.
func (c *converter) inline(n *xhtml.Node) {
	c.prepare(n)
	c.inl.WriteString(renderNode(n))
}

// prepare rewrites the images inside a text block.
func (c *converter) prepare(n *xhtml.Node) {
	for _, img := range findAll(n, isTag(atom.Img)) {
		src := c.src(img)
		if src == "" {
			img.Parent.RemoveChild(img)
			continue
		}
		setAttr(img, "src", src)
	}
}

func (c *converter) closeInline() {
	if c.inl.Len() == 0 {
		return
	}
	s := c.inl.String()
	c.inl.Reset()
	if strings.TrimSpace(textOf(s)) != "" || strings.Contains(s, "<img") {
		c.text.WriteString("<p>" + s + "</p>")
	}
}

// flushText writes the gathered text as one mj-text, cleaned.
func (c *converter) flushText() {
	if c.text.Len() == 0 {
		return
	}
	clean := render.Sanitize(c.text.String())
	c.text.Reset()
	if textOf(clean) == "" && !strings.Contains(clean, "<img") {
		return
	}
	c.blocks = append(c.blocks, "<mj-text>"+neutralize(clean)+"</mj-text>")
}

// neutralize writes Liquid's delimiters as entities: imported text is
// never a template.
func neutralize(s string) string {
	s = strings.ReplaceAll(s, "{{", "&#123;&#123;")
	return strings.ReplaceAll(s, "{%", "&#123;%")
}

// attrEsc is an attribute value, escaped, its braces as entities.
func attrEsc(v string) string { return strings.ReplaceAll(html.EscapeString(v), "{", "&#123;") }

// Node helpers.

func attr(n *xhtml.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

// attrJSON is a field of a Substack element's data-attrs.
func attrJSON(n *xhtml.Node, key string) string {
	var m map[string]any
	if json.Unmarshal([]byte(attr(n, "data-attrs")), &m) != nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func setAttr(n *xhtml.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, xhtml.Attribute{Key: key, Val: val})
}

func hasClass(n *xhtml.Node, class string) bool {
	if n.Type != xhtml.ElementNode {
		return false
	}
	for f := range strings.FieldsSeq(attr(n, "class")) {
		if f == class {
			return true
		}
	}
	return false
}

func isTag(a atom.Atom) func(*xhtml.Node) bool {
	return func(n *xhtml.Node) bool { return n.Type == xhtml.ElementNode && n.DataAtom == a }
}

func isEmbed(n *xhtml.Node) bool {
	if n.Type != xhtml.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.Iframe, atom.Embed, atom.Object, atom.Video, atom.Audio:
		return true
	}
	return false
}

// find is the first node under n (or n) that matches.
func find(n *xhtml.Node, match func(*xhtml.Node) bool) *xhtml.Node {
	if n == nil {
		return nil
	}
	if match(n) {
		return n
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if f := find(k, match); f != nil {
			return f
		}
	}
	return nil
}

func findAll(n *xhtml.Node, match func(*xhtml.Node) bool) []*xhtml.Node {
	var out []*xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(x *xhtml.Node) {
		if match(x) {
			out = append(out, x)
			return
		}
		for k := x.FirstChild; k != nil; k = k.NextSibling {
			walk(k)
		}
	}
	walk(n)
	return out
}

// closest is n's nearest ancestor with the tag, up to stop.
func closest(n *xhtml.Node, a atom.Atom, stop *xhtml.Node) *xhtml.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == xhtml.ElementNode && p.DataAtom == a {
			return p
		}
		if p == stop {
			break
		}
	}
	return nil
}

func linkOf(a *xhtml.Node) string {
	if a == nil || hasClass(a, "image-link") {
		return ""
	}
	return attr(a, "href")
}

// linkIn is the address of the link with the class inside n.
func linkIn(n *xhtml.Node, class string) string {
	return attr(find(n, func(x *xhtml.Node) bool { return x.DataAtom == atom.A && hasClass(x, class) }), "href")
}

// lastLink is the address of the last link inside n (an embedded tweet's).
func lastLink(n *xhtml.Node) string {
	links := findAll(n, isTag(atom.A))
	if len(links) == 0 {
		return ""
	}
	return attr(links[len(links)-1], "href")
}

// onlyImages reports an element whose content is images (maybe linked)
// and nothing to read.
func onlyImages(n *xhtml.Node) bool {
	if find(n, isTag(atom.Img)) == nil {
		return false
	}
	var b strings.Builder
	collectText(&b, n)
	return strings.TrimSpace(b.String()) == ""
}

func renderNode(n *xhtml.Node) string {
	var b strings.Builder
	_ = xhtml.Render(&b, n)
	return b.String()
}

func renderChildren(n *xhtml.Node) string {
	var b strings.Builder
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		_ = xhtml.Render(&b, k)
	}
	return b.String()
}

// dropped elements go with their content.
func dropped(n *xhtml.Node) bool {
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Noscript, atom.Template, atom.Head, atom.Title, atom.Meta, atom.Link, atom.Form, atom.Input,
		atom.Button, atom.Select, atom.Textarea, atom.Svg, atom.Math, atom.Canvas:
		return true
	}
	return false
}

// widget is a platform's own signup block: it goes, it signed readers up
// to the old site.
func widget(n *xhtml.Node) bool {
	return hasClass(n, "subscription-widget-wrap") || hasClass(n, "subscription-widget-wrap-editor") || hasClass(n, "kg-signup-card")
}

func blockLevel(n *xhtml.Node) bool {
	switch n.DataAtom {
	case atom.Div, atom.Section, atom.Article, atom.Main, atom.Header, atom.Footer, atom.Aside, atom.Nav, atom.Figure, atom.Figcaption,
		atom.P, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Ul, atom.Ol, atom.Li, atom.Dl, atom.Dt, atom.Dd, atom.Blockquote,
		atom.Pre, atom.Table, atom.Tr, atom.Hr, atom.Address, atom.Details, atom.Summary, atom.Center, atom.Fieldset:
		return true
	}
	return false
}
