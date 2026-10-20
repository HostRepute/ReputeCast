// Package render draws a post's web version from its MJML (design
// expansion phase 7, brief §3.1): the email's blocks in the site's theme
// and its 680 px reading column, not the email's tables.
//
//   - Personalization uses each token's fallback: "Hi {{ first_name |
//     default: "there" }}" reads "Hi there".
//   - Blocks set to "Email only" in their Visibility section (rc-show-on)
//     are left out; the footer, unsubscribe, view-online and referral
//     blocks are Email only unless they say otherwise.
//   - Conditional blocks (rc-visible, internal/visibility) are hidden on
//     the web, except blocks for a paid tier: those are drawn apart (Paid),
//     and the page shows them to readers who hold the tier and the paywall
//     card to everyone else.
//   - The "Free preview ends here" marker cuts the preview: readers without
//     access get the post down to it.
//
// The output is HTML the site may serve: only known elements and
// attributes survive (no styles, scripts, frames or forms), links and
// images are http(s) or root-relative.
package render

import (
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// Attributes of ReputeCast's own on MJML elements (they never reach an
// email: internal/mjml.PrepareCompile strips rc-* attributes).
const (
	// AttrShowOn is a block's "Show on": "email" (Email only), "web" (Web
	// only), or absent (email and web).
	AttrShowOn = mjml.AttrShowOn
	// AttrPreview marks the "Free preview ends here" block (an mj-raw).
	AttrPreview = "rc-preview"
	// PreviewComment is the marker block's content.
	PreviewComment = "<!-- rc:preview-end -->"
	// PaidKind is Visibility's Paid access condition (Earn, X16: earn.KindPaid):
	// PaidAny shows a block to any paid tier, PaidAtLeast to the tier in its
	// value and above. Its third operator, free subscribers only, isn't for a
	// paid tier: on the web that block is hidden like other conditional blocks.
	PaidKind    = "paid"
	PaidAny     = "any"
	PaidAtLeast = "at_least"
)

// Show-on values.
const (
	ShowOnEmail = "email"
	ShowOnWeb   = "web"
	ShowOnBoth  = "both"
)

// Options of one render.
type Options struct {
	// Links fill the email's link tokens on the web (preferences_url,
	// unsubscribe_url and view_in_browser_url point to pages of the site).
	Links map[string]string
	// PublicURL absolutizes root-relative /media/ images ("" leaves them).
	PublicURL string
	// CoverSrc is the post's cover: the first image showing it is left out
	// of the body (the page shows the cover above the text).
	CoverSrc string
	// Title is the post's title: a first heading saying the same is left
	// out (the page shows the title).
	Title string
}

// Hidden is a block left out of the web version.
type Hidden struct {
	// ID is the block's Visibility id when it has one (the composer
	// selects it), else "".
	ID string `json:"id,omitempty"`
	// Name is the block's own name (rc-name), else "".
	Name string `json:"name,omitempty"`
	// Kind names a well-known block: view_online, footer, referral,
	// header, or block.
	Kind string `json:"kind"`
	// Reason: email_only (its Show on, or the default for its kind),
	// conditional (a Visibility rule), hidden ("Hide for everyone").
	Reason string `json:"reason"`
	// Condition is the first condition's label ("Beta Program").
	Condition string `json:"condition,omitempty"`
	// Line is the block's line in the MJML.
	Line int `json:"line"`
}

// Fallback is a personalization token the web shows with its fallback.
type Fallback struct {
	Token    string `json:"token"`
	Fallback string `json:"fallback"`
	HasValue bool   `json:"hasFallback"`
	// Source is the text around the token as written ("Hi {{ first_name |
	// default: "there" }}") and Sample how the web reads it ("Hi there").
	Source string `json:"source"`
	Sample string `json:"sample"`
}

// Paid is a block for a paid tier: shown to readers holding one of Tiers
// (empty: any paid tier), the paywall card in its place for everyone else.
type Paid struct {
	ID    string   `json:"id"`
	Tiers []string `json:"tiers"`
	HTML  string   `json:"html"`
	Text  string   `json:"text"`
}

// Result is a post's web version.
type Result struct {
	// Body is the whole post. Paid blocks appear in it as PaidPlaceholder(id).
	Body string
	// Preview is the post down to the free preview marker ("" without one).
	Preview string
	// HasMarker reports a marker; MarkerAfter counts the blocks before it.
	HasMarker   bool
	MarkerAfter int
	// Text is the whole post's text, PreviewText the part before the
	// marker (search reads only what a visitor may read).
	Text        string
	PreviewText string
	Words       int
	Hidden      []Hidden
	Fallbacks   []Fallback
	Paid        []Paid
}

// linkTokens are the email's own links: on the web they point to the
// site's pages (Options.Links) or are left out.
var linkTokens = []string{"unsubscribe_url", "preferences_url", "view_in_browser_url", "referral_url", "referral_link", "referral_hub_url"}

const sentinelPrefix = "rc-link:"

// ReadingMinutes is the reading time of a post: 230 words a minute, at
// least one.
func ReadingMinutes(words int) int {
	return max(1, (words+115)/230)
}

// PaidPlaceholder marks where a paid block sits in Body.
func PaidPlaceholder(id string) string { return "<!--rc:paid " + id + "-->" }

// PaidPlaceholderRe finds paid placeholders.
var PaidPlaceholderRe = regexp.MustCompile(`<!--rc:paid ([a-z0-9]{1,24})-->`)

// IsMarker reports whether an element is the free preview marker.
func IsMarker(n *mjml.Node) bool {
	if n == nil || n.Tag != "mj-raw" {
		return false
	}
	if v, ok := n.Get(AttrPreview); ok && v == "end" {
		return true
	}
	return strings.Contains(n.Inner, "rc:preview-end")
}

// Render draws a post's web version from its MJML.
func Render(src string, o Options) (Result, error) {
	res := Result{Hidden: []Hidden{}, Fallbacks: []Fallback{}, Paid: []Paid{}}
	res.Fallbacks = fallbacks(src)
	// Link tokens become sentinels first, so blocks can be told apart by
	// the links they carry (a footer by its unsubscribe link); visible
	// blocks get the web's addresses for them (Options.Links).
	sentinels := map[string]string{}
	for _, k := range linkTokens {
		sentinels[k] = sentinelPrefix + k
	}
	vars := liquid.Vars("", "", "", nil, sentinels)
	rendered, err := liquid.Render(src, vars)
	if err != nil {
		// A template error leaves the tokens as written rather than failing
		// the post: the composer reports the error.
		rendered = src
	}
	root, err := mjml.Parse(rendered)
	if err != nil {
		return res, err
	}
	body := mjml.Body(root)
	if body == nil {
		return res, nil
	}
	r := &renderer{o: o, res: &res, title: normalize(o.Title)}
	r.out, r.text = &r.main, &r.mainText
	for _, c := range body.Children {
		r.block(c, 0)
	}
	res.Body = strings.TrimSpace(r.main.String())
	res.Text = collapse(r.mainText.String())
	if r.marked {
		res.HasMarker = true
		res.MarkerAfter = r.markerAfter
		res.Preview = strings.TrimSpace(r.preview)
		res.PreviewText = collapse(r.previewText)
	} else {
		res.PreviewText = ""
	}
	res.Words = len(strings.Fields(res.Text))
	return res, nil
}

type renderer struct {
	o        Options
	res      *Result
	main     strings.Builder
	mainText strings.Builder
	out      *strings.Builder
	text     *strings.Builder
	title    string
	// open are the containers open in the main output (closed in the
	// preview when the marker sits inside one).
	open []string
	// At the marker: what was drawn and read so far.
	marked      bool
	preview     string
	previewText string
	markerAfter int
	blocks      int
	coverDone   bool
	titleDone   bool
	// inPaid is set while a paid block is drawn apart.
	inPaid bool
}

func (r *renderer) w(s string) { r.out.WriteString(s) }

// push opens a container; pop closes the last one.
func (r *renderer) push(open, tag string) {
	r.w(open)
	if !r.inPaid {
		r.open = append(r.open, tag)
	}
}

func (r *renderer) pop() {
	if !r.inPaid && len(r.open) > 0 {
		r.w("</" + r.open[len(r.open)-1] + ">")
		r.open = r.open[:len(r.open)-1]
		return
	}
	r.w("</div>")
}

// block draws an element of the body.
func (r *renderer) block(n *mjml.Node, depth int) {
	if IsMarker(n) {
		if !r.marked && !r.inPaid {
			r.marked = true
			r.markerAfter = r.blocks
			var closing strings.Builder
			for i := len(r.open) - 1; i >= 0; i-- {
				closing.WriteString("</" + r.open[i] + ">")
			}
			r.preview = r.main.String() + closing.String()
			r.previewText = r.mainText.String()
		}
		return
	}
	if strings.HasPrefix(n.Tag, "mj-") && n.Tag != "mj-raw" {
		if skip := r.webVisibility(n); skip {
			return
		}
	}
	switch n.Tag {
	case "mj-wrapper", "mj-hero":
		for _, c := range n.Children {
			r.block(c, depth+1)
		}
	case "mj-section", "mj-group":
		cols := visibleColumns(n)
		if len(cols) > 1 {
			r.push(`<div class="st-cols">`, "div")
			for _, c := range n.Children {
				if c.Tag != "mj-column" && c.Tag != "mj-group" {
					r.block(c, depth+1)
					continue
				}
				r.push(`<div>`, "div")
				r.block(c, depth+1)
				r.pop()
			}
			r.pop()
			return
		}
		for _, c := range n.Children {
			r.block(c, depth+1)
		}
	case "mj-column":
		for _, c := range n.Children {
			r.block(c, depth+1)
		}
	case "mj-text":
		r.blocks++
		r.textBlock(n)
	case "mj-image":
		r.blocks++
		r.image(n)
	case "mj-button":
		r.blocks++
		r.button(n)
	case "mj-divider":
		r.blocks++
		r.w("<hr>")
	case "mj-table":
		r.blocks++
		r.w(sanitize("<table>"+n.Inner+"</table>", r.o.PublicURL, r.o.Links, r.text))
	case "mj-social":
		r.blocks++
		r.social(n)
	case "mj-navbar":
		r.blocks++
		r.navbar(n)
	case "mj-carousel":
		r.blocks++
		for _, c := range n.Children {
			if c.Tag == "mj-carousel-image" {
				r.figure(c.Attr("src"), c.Attr("alt"), c.Attr("href"), "", "")
			}
		}
	case "mj-accordion":
		r.blocks++
		r.accordion(n)
	case "mj-raw", "mj-spacer":
		// Raw HTML is the email's own business; spacing is the theme's.
	}
}

// webVisibility decides an element's place on the web: false draws it,
// true leaves it out (recorded in Hidden). A paid-tier block is drawn
// apart and true is returned.
func (r *renderer) webVisibility(n *mjml.Node) bool {
	showOn, _ := n.Get(AttrShowOn)
	kind := knownKind(n)
	rule, hasRule := ruleOf(n)
	if showOn == ShowOnEmail || (showOn == "" && emailOnlyByDefault(kind)) {
		r.hide(n, kind, "email_only", rule)
		return true
	}
	if !hasRule || rule.Show == "" || rule.Show == "everyone" {
		return false
	}
	if rule.Show == "nobody" {
		r.hide(n, kind, "hidden", rule)
		return true
	}
	if tiers, ok := paidRule(rule); ok && rule.ID != "" && !r.inPaid {
		r.paidBlock(n, rule.ID, tiers)
		return true
	}
	r.hide(n, kind, "conditional", rule)
	return true
}

func (r *renderer) hide(n *mjml.Node, kind, reason string, rule visRule) {
	if r.inPaid {
		return
	}
	h := Hidden{ID: rule.ID, Name: n.Attr("rc-name"), Kind: kind, Reason: reason, Line: n.Line}
	if len(rule.Conditions) > 0 {
		h.Condition = rule.Conditions[0].Label
		if h.Condition == "" {
			h.Condition = rule.Conditions[0].Value
		}
	}
	r.res.Hidden = append(r.res.Hidden, h)
}

// paidBlock draws a block for a paid tier apart, leaving its placeholder.
func (r *renderer) paidBlock(n *mjml.Node, id string, tiers []string) {
	var b, t strings.Builder
	saveOut, saveText := r.out, r.text
	r.out, r.text, r.inPaid = &b, &t, true
	// The paid block's own element is drawn without its rule.
	switch n.Tag {
	case "mj-section", "mj-wrapper", "mj-group", "mj-column", "mj-hero":
		for _, c := range n.Children {
			r.block(c, 1)
		}
	default:
		clone := *n
		clone.Attrs = withoutAttr(n.Attrs, "rc-visible")
		r.block(&clone, 1)
	}
	text := collapse(t.String())
	r.out, r.inPaid, r.text = saveOut, false, saveText
	r.res.Paid = append(r.res.Paid, Paid{ID: id, Tiers: tiers, HTML: strings.TrimSpace(b.String()), Text: text})
	r.w(PaidPlaceholder(id))
}

func withoutAttr(attrs []mjml.Attr, name string) []mjml.Attr {
	out := make([]mjml.Attr, 0, len(attrs))
	for _, a := range attrs {
		if a.Name != name {
			out = append(out, a)
		}
	}
	return out
}

// visibleColumns are a section's columns.
func visibleColumns(n *mjml.Node) []*mjml.Node {
	var out []*mjml.Node
	for _, c := range n.Children {
		if c.Tag == "mj-column" || c.Tag == "mj-group" {
			out = append(out, c)
		}
	}
	return out
}

// knownKind names the well-known blocks: the view-online line, the
// footer, the referral block and the email's header.
func knownKind(n *mjml.Node) string {
	switch n.Marker {
	case "footer", "transactional-footer":
		return "footer"
	case "referral", "referral-block":
		return "referral"
	case "view-online", "view-in-browser":
		return "view_online"
	case "header", "header-logo":
		return "header"
	}
	inner := innerOf(n)
	switch {
	case strings.Contains(inner, sentinelPrefix+"unsubscribe_url"):
		return "footer"
	case strings.Contains(inner, sentinelPrefix+"referral"):
		return "referral"
	case strings.Contains(inner, sentinelPrefix+"view_in_browser_url"):
		return "view_online"
	}
	return "block"
}

// emailOnlyByDefault: footers, unsubscribe, view-online and referral
// blocks only make sense in an inbox.
func emailOnlyByDefault(kind string) bool {
	return kind == "footer" || kind == "view_online" || kind == "referral"
}

// innerOf is an element's text content as written (its own and its
// descendants').
func innerOf(n *mjml.Node) string {
	var b strings.Builder
	var walk func(*mjml.Node)
	walk = func(x *mjml.Node) {
		b.WriteString(x.Inner)
		for _, a := range x.Attrs {
			if a.Name == "href" {
				b.WriteString(" " + a.Value)
			}
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// visRule is the part of a Visibility rule the web reads.
type visRule struct {
	ID         string `json:"id"`
	Show       string `json:"show"`
	Match      string `json:"match"`
	Conditions []struct {
		Kind  string `json:"kind"`
		Op    string `json:"op"`
		Value string `json:"value"`
		Label string `json:"label"`
	} `json:"conditions"`
}

func ruleOf(n *mjml.Node) (visRule, bool) {
	raw, ok := n.Get("rc-visible")
	if !ok || strings.TrimSpace(raw) == "" {
		return visRule{}, false
	}
	var v visRule
	if json.Unmarshal([]byte(raw), &v) != nil {
		return visRule{}, false
	}
	return v, true
}

// paidRule reports a rule that shows a block only to paid tiers, with the
// lowest tiers it names (empty: any paid tier; a higher tier includes a
// lower one).
func paidRule(v visRule) ([]string, bool) {
	if v.Show != "only" || len(v.Conditions) == 0 {
		return nil, false
	}
	tiers := []string{}
	for _, c := range v.Conditions {
		if c.Kind != PaidKind {
			return nil, false
		}
		switch c.Op {
		case PaidAny:
			if v.Match != "all" {
				return []string{}, true
			}
		case PaidAtLeast:
			if strings.TrimSpace(c.Value) == "" {
				return nil, false
			}
			tiers = append(tiers, strings.TrimSpace(c.Value))
		default:
			return nil, false
		}
	}
	return tiers, true
}

// textBlock draws an mj-text: its HTML cleaned, a heading when the email
// styled a short line as one.
func (r *renderer) textBlock(n *mjml.Node) {
	inner := strings.TrimSpace(n.Inner)
	if inner == "" {
		return
	}
	var t strings.Builder
	clean := sanitize(inner, r.o.PublicURL, r.o.Links, &t)
	plain := collapse(t.String())
	if plain == "" && !strings.Contains(clean, "<img") {
		return
	}
	if !r.titleDone && r.title != "" && (isHeading(clean) || headingLevel(n, plain) > 0) {
		// The first heading saying the post's title is the page's title.
		r.titleDone = true
		if normalize(plain) == r.title {
			return
		}
	}
	if !hasBlock(clean) {
		if level := headingLevel(n, plain); level > 0 {
			clean = "<h" + strconv.Itoa(level) + ">" + clean + "</h" + strconv.Itoa(level) + ">"
		} else {
			clean = "<p>" + clean + "</p>"
		}
	}
	r.w(clean)
	r.text.WriteString(t.String() + "\n")
}

// headingLevel reads a heading from an mj-text's style: large or bold
// short lines.
func headingLevel(n *mjml.Node, plain string) int {
	if len([]rune(plain)) > 140 || strings.ContainsAny(plain, "\n") {
		return 0
	}
	size := pxOf(n.Attr("font-size"))
	bold := n.Attr("font-weight") == "700" || n.Attr("font-weight") == "bold" || n.Attr("font-weight") == "600"
	switch {
	case size >= 22:
		return 2
	case size >= 17 && bold:
		return 3
	}
	return 0
}

func pxOf(v string) int {
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "px"))
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return int(f)
}

var blockRe = regexp.MustCompile(`^\s*<(p|h[1-6]|ul|ol|blockquote|pre|table|figure|hr|div)\b`)
var headingRe = regexp.MustCompile(`^\s*<h[1-6]\b`)

func hasBlock(s string) bool  { return blockRe.MatchString(s) }
func isHeading(s string) bool { return headingRe.MatchString(s) }

func (r *renderer) image(n *mjml.Node) {
	src := n.Attr("src")
	if src == "" {
		return
	}
	if !r.coverDone && r.o.CoverSrc != "" && sameImage(src, r.o.CoverSrc) {
		r.coverDone = true
		return
	}
	r.figure(src, n.Attr("alt"), n.Attr("href"), n.Attr("width"), n.Attr("height"))
}

func sameImage(a, b string) bool {
	strip := func(s string) string {
		if u, err := url.Parse(s); err == nil {
			return u.Path
		}
		return s
	}
	return strip(a) == strip(b)
}

func (r *renderer) figure(src, alt, href, width, height string) {
	src = safeURL(src, r.o.PublicURL, true, r.o.Links)
	if src == "" {
		return
	}
	var b strings.Builder
	b.WriteString(`<figure>`)
	link := safeURL(href, "", false, r.o.Links)
	if link != "" {
		b.WriteString(`<a href="` + html.EscapeString(link) + `">`)
	}
	b.WriteString(`<img src="` + html.EscapeString(src) + `" alt="` + html.EscapeString(alt) + `"`)
	if w := pxOf(width); w > 0 {
		b.WriteString(` width="` + strconv.Itoa(w) + `"`)
		if h := pxOf(height); h > 0 {
			b.WriteString(` height="` + strconv.Itoa(h) + `"`)
		}
	}
	b.WriteString(` loading="lazy" decoding="async">`)
	if link != "" {
		b.WriteString(`</a>`)
	}
	b.WriteString(`</figure>`)
	r.w(b.String())
}

func (r *renderer) button(n *mjml.Node) {
	var t strings.Builder
	label := strings.TrimSpace(inlineText(n.Inner, &t))
	href := safeURL(n.Attr("href"), "", false, r.o.Links)
	if label == "" || href == "" {
		return
	}
	r.w(`<p class="st-bt"><a class="st-btn" href="` + html.EscapeString(href) + `">` + html.EscapeString(label) + `</a></p>`)
	r.text.WriteString(label + "\n")
}

func (r *renderer) social(n *mjml.Node) {
	var parts []string
	for _, c := range n.Children {
		if c.Tag != "mj-social-element" {
			continue
		}
		href := safeURL(c.Attr("href"), "", false, r.o.Links)
		var t strings.Builder
		label := strings.TrimSpace(inlineText(c.Inner, &t))
		if label == "" {
			label = c.Attr("name")
		}
		if href == "" || label == "" {
			continue
		}
		parts = append(parts, `<a href="`+html.EscapeString(href)+`">`+html.EscapeString(label)+`</a>`)
	}
	if len(parts) > 0 {
		r.w(`<p class="st-soc">` + strings.Join(parts, "") + `</p>`)
	}
}

func (r *renderer) navbar(n *mjml.Node) {
	var parts []string
	for _, c := range n.Children {
		if c.Tag != "mj-navbar-link" {
			continue
		}
		href := safeURL(c.Attr("href"), "", false, r.o.Links)
		var t strings.Builder
		label := strings.TrimSpace(inlineText(c.Inner, &t))
		if href == "" || label == "" {
			continue
		}
		parts = append(parts, `<a href="`+html.EscapeString(href)+`">`+html.EscapeString(label)+`</a>`)
	}
	if len(parts) > 0 {
		r.w(`<p class="st-soc">` + strings.Join(parts, "") + `</p>`)
	}
}

func (r *renderer) accordion(n *mjml.Node) {
	for _, el := range n.Children {
		if el.Tag != "mj-accordion-element" {
			continue
		}
		var title, body string
		for _, c := range el.Children {
			switch c.Tag {
			case "mj-accordion-title":
				var t strings.Builder
				title = strings.TrimSpace(inlineText(c.Inner, &t))
				r.text.WriteString(t.String() + "\n")
			case "mj-accordion-text":
				body = sanitize(c.Inner, r.o.PublicURL, r.o.Links, r.text)
			}
		}
		if title == "" {
			continue
		}
		if !hasBlock(body) && body != "" {
			body = "<p>" + body + "</p>"
		}
		r.w(`<details><summary>` + html.EscapeString(title) + `</summary>` + body + `</details>`)
	}
}

// fallbacks lists the personalization tokens a reader without a contact
// sees with their fallback, with the text around each.
func fallbacks(src string) []Fallback {
	out := []Fallback{}
	seen := map[string]bool{}
	for _, u := range liquid.Uses(src) {
		if !contactToken(u.Key) || seen[u.Key] {
			continue
		}
		seen[u.Key] = true
		f := Fallback{Token: u.Key, Fallback: u.Fallback, HasValue: u.HasFallback}
		f.Source, f.Sample = around(src, u.Key, u.Fallback)
		out = append(out, f)
	}
	return out
}

// contactToken reports a token about the reader (not a link).
func contactToken(key string) bool {
	switch {
	case strings.HasSuffix(key, "_url"), strings.HasSuffix(key, "_link"):
		return false
	case key == "issue", key == "feed", key == "order", key == "campaign", key == "workspace", key == "today", key == "now":
		return false
	}
	return true
}

var tokenSpan = regexp.MustCompile(`\{\{-?\s*([^{}]*?)\s*-?\}\}`)

// around is the token as written with the word before it ("Hi {{
// first_name | default: "there" }}") and how the web reads it ("Hi there").
func around(src, key, fallback string) (string, string) {
	for _, loc := range tokenSpan.FindAllStringSubmatchIndex(src, -1) {
		inner := src[loc[2]:loc[3]]
		name := strings.TrimSpace(strings.SplitN(inner, "|", 2)[0])
		name = strings.TrimPrefix(name, "contact.")
		if name != key {
			continue
		}
		token := strings.Join(strings.Fields(src[loc[0]:loc[1]]), " ")
		before := src[:loc[0]]
		if i := strings.LastIndexAny(before, ">\n"); i >= 0 {
			before = before[i+1:]
		}
		words := strings.Fields(before)
		lead := ""
		if len(words) > 0 {
			lead = words[len(words)-1] + " "
		}
		lead = html.UnescapeString(lead)
		return lead + token, strings.TrimSpace(lead + fallback)
	}
	return "{{ " + key + " }}", fallback
}

// sanitize keeps the HTML the site may serve: known elements and safe
// attributes. Text goes to t.
func sanitize(fragment, publicURL string, links map[string]string, t *strings.Builder) string {
	nodes, err := xhtml.ParseFragment(strings.NewReader(fragment), &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return html.EscapeString(fragment)
	}
	var b strings.Builder
	for _, n := range nodes {
		writeNode(&b, n, publicURL, links, t)
	}
	return strings.TrimSpace(b.String())
}

// inlineText is an element's text only.
func inlineText(fragment string, t *strings.Builder) string {
	nodes, err := xhtml.ParseFragment(strings.NewReader(fragment), &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return fragment
	}
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		switch n.Type {
		case xhtml.TextNode:
			b.WriteString(n.Data)
		case xhtml.ElementNode:
			if n.DataAtom == atom.Script || n.DataAtom == atom.Style {
				return
			}
			if n.DataAtom == atom.Br {
				b.WriteString(" ")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	s := collapse(b.String())
	t.WriteString(s)
	return s
}

// kept are the elements the web version keeps; unwrapped elements keep
// their children only.
var kept = map[atom.Atom]bool{
	atom.P: true, atom.Br: true, atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
	atom.Strong: true, atom.B: true, atom.Em: true, atom.I: true, atom.U: true, atom.S: true, atom.Del: true,
	atom.Sup: true, atom.Sub: true, atom.Small: true, atom.Mark: true, atom.Code: true, atom.Pre: true,
	atom.Blockquote: true, atom.Ul: true, atom.Ol: true, atom.Li: true, atom.A: true, atom.Img: true,
	atom.Table: true, atom.Thead: true, atom.Tbody: true, atom.Tfoot: true, atom.Tr: true, atom.Th: true, atom.Td: true,
	atom.Hr: true, atom.Figure: true, atom.Figcaption: true, atom.Caption: true,
}

// dropped elements go with their content.
var dropped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Iframe: true, atom.Object: true, atom.Embed: true, atom.Form: true,
	atom.Input: true, atom.Button: true, atom.Select: true, atom.Textarea: true, atom.Template: true, atom.Svg: true,
	atom.Math: true, atom.Noscript: true, atom.Head: true, atom.Title: true, atom.Meta: true, atom.Link: true,
	atom.Video: true, atom.Audio: true, atom.Source: true, atom.Canvas: true,
}

func writeNode(b *strings.Builder, n *xhtml.Node, publicURL string, links map[string]string, t *strings.Builder) {
	switch n.Type {
	case xhtml.TextNode:
		b.WriteString(html.EscapeString(n.Data))
		t.WriteString(n.Data)
		return
	case xhtml.ElementNode:
	default:
		return
	}
	if dropped[n.DataAtom] {
		return
	}
	children := func() {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeNode(b, c, publicURL, links, t)
		}
	}
	if !kept[n.DataAtom] {
		// div, span, font, center…: their content stays.
		if n.DataAtom == atom.Div {
			t.WriteString("\n")
		}
		children()
		return
	}
	tag := n.DataAtom.String()
	if tag == "b" {
		tag = "strong"
	}
	if tag == "i" {
		tag = "em"
	}
	switch n.DataAtom {
	case atom.Img:
		src := safeURL(attr(n, "src"), publicURL, true, links)
		if src == "" {
			return
		}
		b.WriteString(`<img src="` + html.EscapeString(src) + `" alt="` + html.EscapeString(attr(n, "alt")) + `"`)
		if w, err := strconv.Atoi(strings.TrimSuffix(attr(n, "width"), "px")); err == nil && w > 0 {
			b.WriteString(` width="` + strconv.Itoa(w) + `"`)
		}
		if h, err := strconv.Atoi(strings.TrimSuffix(attr(n, "height"), "px")); err == nil && h > 0 {
			b.WriteString(` height="` + strconv.Itoa(h) + `"`)
		}
		b.WriteString(` loading="lazy" decoding="async">`)
		return
	case atom.Br, atom.Hr:
		b.WriteString("<" + tag + ">")
		if n.DataAtom == atom.Br {
			t.WriteString("\n")
		}
		return
	case atom.A:
		href := safeURL(attr(n, "href"), "", false, links)
		if href == "" {
			children()
			return
		}
		b.WriteString(`<a href="` + html.EscapeString(href) + `">`)
		children()
		b.WriteString("</a>")
		return
	case atom.Td, atom.Th:
		b.WriteString("<" + tag)
		for _, k := range []string{"colspan", "rowspan"} {
			if v := attr(n, k); v != "" {
				if x, err := strconv.Atoi(v); err == nil && x > 0 && x < 100 {
					b.WriteString(` ` + k + `="` + strconv.Itoa(x) + `"`)
				}
			}
		}
		b.WriteString(">")
		children()
		b.WriteString("</" + tag + ">")
		t.WriteString(" ")
		return
	}
	b.WriteString("<" + tag + ">")
	children()
	b.WriteString("</" + tag + ">")
	switch n.DataAtom {
	case atom.P, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Li, atom.Blockquote, atom.Pre, atom.Tr:
		t.WriteString("\n")
	}
}

func attr(n *xhtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

// safeURL keeps http(s), mailto (links only) and root-relative addresses;
// root-relative /media/ images become absolute when publicURL is set.
func safeURL(raw, publicURL string, image bool, links map[string]string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, sentinelPrefix) {
		if image {
			return ""
		}
		raw = strings.TrimSpace(links[strings.TrimPrefix(raw, sentinelPrefix)])
	}
	if raw == "" || strings.Contains(raw, "{{") || strings.Contains(raw, "{%") {
		return ""
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		if image && publicURL != "" && strings.HasPrefix(raw, "/media/") {
			return strings.TrimRight(publicURL, "/") + raw
		}
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return ""
		}
		return u.String()
	case "mailto":
		if image {
			return ""
		}
		return u.String()
	}
	return ""
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// normalize compares titles: lower case, letters and digits.
func normalize(s string) string {
	s = strings.ToLower(html.UnescapeString(s))
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " ")
}
