// Package contentcheck finds deliverability problems in an email's content:
// size against Gmail's 102 KB clipping limit, images without alt text, the
// image-to-text ratio, links on URI blocklists, URL shorteners and broken
// links, a missing unsubscribe link or postal address, Liquid errors, spam
// heuristics in the subject, and the plain-text part.
//
// The composer runs the fast checks inline as people edit; Send Readiness
// runs the full set (including link probing) before a send.
package contentcheck

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/publicsuffix"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// Severities, in the order checks are listed.
const (
	Block = "block"
	Warn  = "warn"
	Info  = "info"
	Pass  = "pass"
)

// ClipBytes is the size at which Gmail clips a message (102 KB).
const ClipBytes = 102 * 1024

// NearClipBytes is where the size warning starts.
const NearClipBytes = 85 * 1024

// ContentCheckFix is a one-click fix offered with an issue. The composer applies it to
// the canvas (Path points at the component) or to the campaign settings.
type ContentCheckFix struct {
	Kind  string `json:"kind" enum:"replace_link,edit_link,insert_footer,set_alt,minify,remove_section,use_subject"`
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
	Path  []int  `json:"path,omitempty"`
}

// ContentCheckIssue is one finding.
type ContentCheckIssue struct {
	ID       string `json:"id" example:"links.blocklist"`
	Severity string `json:"severity" enum:"block,warn,info,pass"`
	Group    string `json:"group" enum:"Content,Links,Compliance,Subject,Personalization"`
	Title    string `json:"title"`
	Detail   string `json:"detail,omitempty"`
	// Tag is the short label shown on the canvas next to the element.
	Tag string `json:"tag,omitempty"`
	// Path locates the element from mj-body (element indices, comments
	// excluded); empty when the issue concerns the whole email.
	Path []int `json:"path"`
	// Section is the top-level section index, -1 for the whole email.
	Section int               `json:"section"`
	Fixes   []ContentCheckFix `json:"fixes"`
	// URL is the link a link issue is about (to find it inside text).
	URL string `json:"url,omitempty"`
}

// ContentCheckSection is what the inspector shows for the selected section.
type ContentCheckSection struct {
	Index         int    `json:"index"`
	Label         string `json:"label"`
	Images        int    `json:"images"`
	ImagesWithAlt int    `json:"imagesWithAlt"`
	Links         int    `json:"links"`
	LinksOK       int    `json:"linksOk"`
	Issues        int    `json:"issues"`
}

// ContentCheckReport is the outcome of a check run.
type ContentCheckReport struct {
	Verdict          string `json:"verdict" enum:"ready,warnings,blocked"`
	Blocks           int    `json:"blocks"`
	Warnings         int    `json:"warnings"`
	SizeBytes        int    `json:"sizeBytes"`
	ClipBytes        int    `json:"clipBytes"`
	Images           int    `json:"images"`
	ImagesMissingAlt int    `json:"imagesMissingAlt"`
	Links            int    `json:"links"`
	LinksBlocklisted int    `json:"linksBlocklisted"`
	LinksBroken      int    `json:"linksBroken"`
	// SubjectHeuristics names the spam heuristics the subject trips.
	SubjectHeuristics []string              `json:"subjectHeuristics"`
	Issues            []ContentCheckIssue   `json:"issues"`
	Sections          []ContentCheckSection `json:"sections"`
	PlainText         string                `json:"plainText"`
	CheckedAt         time.Time             `json:"checkedAt"`
	// ImageShare is the estimated share of the email's area covered by
	// images (0 to 1; 0 without images). Words counts the plain text.
	ImageShare float64 `json:"imageShare"`
	Words      int     `json:"words"`
	// LinkList is every link with its text and, when links were probed, the
	// response (Send Readiness shows the slowest and the link domains).
	LinkList []ContentCheckLink `json:"linkList"`
}

// ContentCheckLink is one link of the email.
type ContentCheckLink struct {
	URL  string `json:"url"`
	Host string `json:"host"`
	Text string `json:"text"`
	Kind string `json:"kind" enum:"button,link,image,nav,social"`
	Path []int  `json:"path"`
	// Probed is true when the link was requested; Status, Broken, Millis
	// and Final (the address after redirects) describe the response.
	Probed bool   `json:"probed"`
	Status int    `json:"status,omitempty"`
	Broken bool   `json:"broken,omitempty"`
	Millis int    `json:"millis,omitempty"`
	Final  string `json:"final,omitempty"`
}

// Input is the content to check.
type Input struct {
	Subject   string
	Preheader string
	MJML      string
	HTML      string
	// OwnDomains are the workspace's sending domains ("news.acme.io"); fixes
	// for blocklisted links point at their registrable domain.
	OwnDomains []string
	// Address is the postal address from the workspace's branding, if set.
	Address string
	// Fields are the personalization tokens that exist (liquid.Catalog keys).
	Fields []string
	// ProbeLinks requests each link (Send Readiness); the composer skips it.
	ProbeLinks bool
	// Language is the language version's code ("de"); the subject's
	// English phrase heuristics only apply to English (or no language).
	Language string
}

// Blocklist answers which URI blocklists list a domain.
type Blocklist interface {
	// Listings returns, for each listed domain, the lists it is on. Clean
	// and unknown domains are absent.
	Listings(ctx context.Context, domains []string) (map[string][]string, error)
}

// ProbeResult is the outcome of requesting one link.
type ProbeResult struct {
	Status int    `json:"status"`
	Broken bool   `json:"broken"`
	Error  string `json:"error,omitempty"`
	// Millis is how long the response took; Final is the address after
	// redirects (empty when the link didn't redirect).
	Millis int    `json:"millis,omitempty"`
	Final  string `json:"final,omitempty"`
}

// Prober requests links to find broken ones.
type Prober interface {
	Probe(ctx context.Context, urls []string) map[string]ProbeResult
}

type Checker struct {
	Blocklist Blocklist
	Prober    Prober
	Now       func() time.Time
}

type image struct {
	src, alt string
	hasAlt   bool
	w, h     int
	path     []int
	section  int
	inHeader bool
}

type link struct {
	href, text string
	kind       string // button, link, image, nav, social
	path       []int
	section    int
}

type scan struct {
	images   []image
	links    []link
	sections []mjml.Section
	// section byte sizes of the MJML source, to apportion the HTML size
	sectionBytes []int
	// columns and inline style attributes per section (cards pasted with
	// their CSS are a common reason for a heavy email)
	sectionCols   []int
	sectionStyles []int
	parsed        bool
}

// Check runs every check on the content.
func (c *Checker) Check(ctx context.Context, in Input) (ContentCheckReport, error) {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	r := ContentCheckReport{ClipBytes: ClipBytes, Issues: []ContentCheckIssue{}, Sections: []ContentCheckSection{}, SubjectHeuristics: []string{}, LinkList: []ContentCheckLink{}, CheckedAt: now()}
	htmlSrc := in.HTML
	if htmlSrc == "" && strings.TrimSpace(in.MJML) != "" {
		htmlSrc = mjml.Compile(in.MJML).HTML
	}
	r.SizeBytes = len(htmlSrc)
	s := c.scanContent(in.MJML, htmlSrc)
	r.Images = len(s.images)
	r.Links = len(s.links)
	r.PlainText = PlainText(htmlSrc)
	r.Words = len(strings.Fields(r.PlainText))
	if share, ok := imageShare(s.images, r.PlainText); ok {
		r.ImageShare = share
	}

	sectionIssues := map[int]int{}
	add := func(is ContentCheckIssue) {
		if is.Fixes == nil {
			is.Fixes = []ContentCheckFix{}
		}
		if is.Path == nil {
			is.Path = []int{}
		}
		r.Issues = append(r.Issues, is)
		if is.Section >= 0 && (is.Severity == Block || is.Severity == Warn) {
			sectionIssues[is.Section]++
		}
	}

	for _, im := range s.images {
		if im.hasAlt || (im.w > 0 && im.w <= 2 && im.h > 0 && im.h <= 2) {
			continue
		}
		r.ImagesMissingAlt++
		name := imageName(im, s.sections)
		add(ContentCheckIssue{ID: "content.alt_text", Severity: Warn, Group: "Content", Title: name + " has no alt text", Tag: "Image has no alt text",
			Detail: "Images are off by default in Outlook and many corporate clients; alt text is what shows, and spam filters score image-only mail higher.",
			Path:   im.path, Section: im.section, Fixes: []ContentCheckFix{{Kind: "set_alt", Label: "Save", Path: im.path}}})
	}

	if r.SizeBytes >= NearClipBytes {
		kb := int(math.Round(float64(r.SizeBytes) / 1024))
		title := fmt.Sprintf("%d KB of Gmail's 102 KB clipping limit", kb)
		if r.SizeBytes > ClipBytes {
			title = fmt.Sprintf("%d KB is over Gmail's 102 KB clipping limit", kb)
		}
		detail := "Gmail clips the message and hides the footer, including the unsubscribe link."
		fixes := []ContentCheckFix{}
		if saved := r.SizeBytes - len(Minify(htmlSrc)); saved >= 1024 {
			fixes = append(fixes, ContentCheckFix{Kind: "minify", Label: fmt.Sprintf("Minify inline CSS · saves about %d KB", int(math.Round(float64(saved)/1024)))})
		}
		if bulk, fix := bulkOf(s, r.SizeBytes); bulk != "" {
			detail += " " + bulk
			if fix != nil {
				fixes = append(fixes, *fix)
			}
		}
		add(ContentCheckIssue{ID: "content.size", Severity: Warn, Group: "Content", Title: title, Detail: detail, Section: -1, Fixes: fixes})
	}

	if share, ok := imageShare(s.images, r.PlainText); ok && share > 0.85 {
		add(ContentCheckIssue{ID: "content.image_ratio", Severity: Warn, Group: "Content", Title: fmt.Sprintf("The email is %d%% image", int(math.Round(share*100))),
			Detail: "Spam filters score image-heavy mail higher, and most of it disappears when images are off. Add text or use fewer images.", Section: -1})
	}

	// Links: blocklists, shorteners, empty links, broken links.
	c.checkLinks(ctx, in, s, &r, add)

	hasUnsub := false
	for _, l := range s.links {
		if isUnsubscribe(l) {
			hasUnsub = true
			break
		}
	}
	text := r.PlainText
	hasAddress := hasPostalAddress(text, in.Address)
	if (!hasUnsub || !hasAddress) && (len(s.sections) > 0 || strings.TrimSpace(text) != "") {
		title, tag := "No unsubscribe link or postal address in the body", "No unsubscribe link or postal address"
		switch {
		case hasUnsub:
			title, tag = "No postal address in the footer", "No postal address"
		case hasAddress:
			title, tag = "No unsubscribe link in the body", "No unsubscribe link"
		}
		add(ContentCheckIssue{ID: "compliance.footer", Severity: Block, Group: "Compliance", Title: title, Tag: tag,
			Detail:  "Headers are added automatically, but the visible link and address are required too.",
			Section: -1, Fixes: []ContentCheckFix{{Kind: "insert_footer", Label: "Insert footer block"}}})
	}

	for _, src := range []struct{ where, text string }{{"Subject", in.Subject}, {"Preheader", in.Preheader}, {"Body", bodySource(in.MJML, htmlSrc)}} {
		for _, p := range liquid.Validate(src.text) {
			add(ContentCheckIssue{ID: "personalization.syntax", Severity: Block, Group: "Personalization", Title: src.where + ": " + p.Message,
				Detail: "Fix the personalization tag; a broken tag would print as text or stop the send.", Section: -1})
		}
	}
	if len(in.Fields) > 0 {
		known := map[string]bool{"unsubscribe_url": true, "preferences_url": true, "view_in_browser_url": true}
		for _, f := range in.Fields {
			known[f] = true
		}
		for _, u := range liquid.Uses(in.Subject + "\n" + in.Preheader + "\n" + bodySource(in.MJML, htmlSrc)) {
			root := strings.SplitN(u.Key, ".", 2)[0]
			if known[u.Key] || known[root] || root == "contact" || root == "forloop" || u.HasFallback {
				continue
			}
			// Poll answer links and survey links are filled per recipient
			// (internal/polls).
			if strings.HasPrefix(root, "rc_poll_") || strings.HasPrefix(root, "rc_survey_") {
				continue
			}
			add(ContentCheckIssue{ID: "personalization.unknown_field", Severity: Warn, Group: "Personalization",
				Title:  fmt.Sprintf("{{ %s }} isn't a contact field; it prints nothing", u.Key),
				Detail: "Check the spelling, or add a fallback such as | default: \"…\".", Section: -1})
		}
	}

	if strings.TrimSpace(in.Subject) == "" {
		add(ContentCheckIssue{ID: "subject.missing", Severity: Block, Group: "Subject", Title: "The subject is empty", Detail: "Inbox lists show the subject first; write one before sending.", Section: -1})
	} else if h := SubjectHeuristicsIn(in.Subject, in.Language); len(h) > 0 {
		r.SubjectHeuristics = h
		suggestion := SuggestSubject(in.Subject)
		is := ContentCheckIssue{ID: "subject.spam", Severity: Warn, Group: "Subject", Title: fmt.Sprintf("Subject trips %d spam %s", len(h), plural(len(h), "heuristic", "heuristics")), Section: -1}
		if suggestion != "" && suggestion != in.Subject {
			is.Detail = fmt.Sprintf("Suggested: %q", suggestion)
			is.Fixes = []ContentCheckFix{{Kind: "use_subject", Label: "Use suggestion", Value: suggestion}}
		}
		add(is)
	}

	if strings.TrimSpace(r.PlainText) == "" && r.Images > 0 {
		add(ContentCheckIssue{ID: "content.plain_text", Severity: Warn, Group: "Content", Title: "The plain-text version is empty", Detail: "The email has no text, so the text/plain part is blank. Add a sentence that says what the images show.", Section: -1})
	}

	for _, sec := range s.sections {
		sum := ContentCheckSection{Index: sec.Index, Label: sec.Label, Issues: sectionIssues[sec.Index]}
		for _, im := range s.images {
			if im.section == sec.Index {
				sum.Images++
				if im.hasAlt {
					sum.ImagesWithAlt++
				}
			}
		}
		for _, l := range s.links {
			if l.section == sec.Index {
				sum.Links++
			}
		}
		sum.LinksOK = sum.Links
		for _, is := range r.Issues {
			if is.Section == sec.Index && strings.HasPrefix(is.ID, "links.") && (is.Severity == Block || is.Severity == Warn) {
				sum.LinksOK--
			}
		}
		if sum.LinksOK < 0 {
			sum.LinksOK = 0
		}
		r.Sections = append(r.Sections, sum)
	}

	sort.SliceStable(r.Issues, func(i, j int) bool { return rank(r.Issues[i]) < rank(r.Issues[j]) })
	for _, is := range r.Issues {
		switch is.Severity {
		case Block:
			r.Blocks++
		case Warn:
			r.Warnings++
		}
	}
	switch {
	case r.Blocks > 0:
		r.Verdict = "blocked"
	case r.Warnings > 0:
		r.Verdict = "warnings"
	default:
		r.Verdict = "ready"
	}
	return r, nil
}

// rank orders issues: blocks, then warnings, then the rest; subject issues
// after content issues of the same severity.
func rank(is ContentCheckIssue) int {
	r := 0
	switch is.Severity {
	case Block:
		r = 0
	case Warn:
		r = 10
	case Info:
		r = 20
	default:
		r = 30
	}
	if is.Group == "Subject" {
		r += 5
	}
	return r
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func bodySource(mjmlSrc, htmlSrc string) string {
	if strings.TrimSpace(mjmlSrc) != "" {
		return mjmlSrc
	}
	return htmlSrc
}

// scanContent collects images and links with their location. MJML gives
// element paths; HTML-only content is scanned without them.
func (c *Checker) scanContent(mjmlSrc, htmlSrc string) scan {
	var s scan
	if strings.TrimSpace(mjmlSrc) != "" {
		if root, err := mjml.Parse(mjmlSrc); err == nil {
			if body := mjml.Body(root); body != nil {
				s.parsed = true
				total := len(body.Children)
				for i, n := range body.Children {
					s.sections = append(s.sections, mjml.Section{Index: i, Tag: n.Tag, Label: mjml.SectionLabel(n, i, total), Marker: n.Marker, Line: n.Line})
					s.sectionBytes = append(s.sectionBytes, n.End-n.Start)
					cols := 0
					for _, ch := range n.Children {
						if ch.Tag == "mj-column" || ch.Tag == "mj-group" {
							cols++
						}
					}
					s.sectionCols = append(s.sectionCols, cols)
					styles := 0
					if n.Start >= 0 && n.End <= len(mjmlSrc) && n.Start < n.End {
						styles = strings.Count(mjmlSrc[n.Start:n.End], "style=")
					}
					s.sectionStyles = append(s.sectionStyles, styles)
					header := s.sections[i].Label == "header"
					n.Walk(func(x *mjml.Node, rel []int) {
						path := append([]int{i}, rel...)
						s.scanNode(x, path, i, header)
					})
				}
				return s
			}
		}
	}
	if htmlSrc != "" {
		doc, err := html.Parse(strings.NewReader(htmlSrc))
		if err == nil {
			s.scanHTML(doc, nil, -1, false)
		}
	}
	return s
}

func (s *scan) scanNode(x *mjml.Node, path []int, section int, header bool) {
	switch x.Tag {
	case "mj-image":
		alt, ok := x.Get("alt")
		s.images = append(s.images, image{src: x.Attr("src"), alt: alt, hasAlt: ok, w: px(x.Attr("width")), h: px(x.Attr("height")), path: path, section: section, inHeader: header})
		if href := x.Attr("href"); href != "" {
			s.links = append(s.links, link{href: href, text: alt, kind: "image", path: path, section: section})
		}
	case "mj-button":
		label := strings.TrimSpace(html.UnescapeString(stripTags(x.Inner)))
		s.links = append(s.links, link{href: x.Attr("href"), text: label, kind: "button", path: path, section: section})
	case "mj-navbar-link":
		s.links = append(s.links, link{href: x.Attr("href"), text: strings.TrimSpace(stripTags(x.Inner)), kind: "nav", path: path, section: section})
	case "mj-social-element":
		if href := x.Attr("href"); href != "" {
			s.links = append(s.links, link{href: href, text: strings.TrimSpace(stripTags(x.Inner)), kind: "social", path: path, section: section})
		}
	case "mj-text", "mj-raw", "mj-table", "mj-accordion-text":
		if strings.TrimSpace(x.Inner) == "" {
			return
		}
		doc, err := html.Parse(strings.NewReader("<body>" + x.Inner + "</body>"))
		if err == nil {
			s.scanHTML(doc, path, section, header)
		}
	}
}

func (s *scan) scanHTML(n *html.Node, path []int, section int, header bool) {
	if n.Type == html.ElementNode {
		switch n.DataAtom {
		case atom.A:
			href := attr(n, "href")
			if href != "" || hasAttr(n, "href") {
				s.links = append(s.links, link{href: href, text: strings.TrimSpace(textOf(n)), kind: "link", path: path, section: section})
			}
		case atom.Img:
			w, h := px(attr(n, "width")), px(attr(n, "height"))
			s.images = append(s.images, image{src: attr(n, "src"), alt: attr(n, "alt"), hasAlt: hasAttr(n, "alt"), w: w, h: h, path: path, section: section, inHeader: header})
		case atom.Style, atom.Script, atom.Head:
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s.scanHTML(c, path, section, header)
	}
}

var tagStrip = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string { return tagStrip.ReplaceAllString(s, "") }

func px(v string) int {
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "px"))
	if v == "" || strings.HasSuffix(v, "%") {
		return 0
	}
	var f float64
	if _, err := fmt.Sscanf(v, "%g", &f); err != nil {
		return 0
	}
	return int(f)
}

func imageName(im image, sections []mjml.Section) string {
	if im.inHeader {
		return "Logo"
	}
	if im.section >= 0 && im.section < len(sections) {
		switch l := sections[im.section].Label; l {
		case "hero":
			return "Hero image"
		case "header":
			return "Logo"
		default:
			return "Image in the " + l + " section"
		}
	}
	return "An image"
}

// bulkOf explains which sections make up most of the size and offers to
// remove the largest one.
func bulkOf(s scan, size int) (string, *ContentCheckFix) {
	if len(s.sections) < 3 || len(s.sectionBytes) != len(s.sections) {
		return "", nil
	}
	total := 0
	for _, b := range s.sectionBytes {
		total += b
	}
	if total == 0 {
		return "", nil
	}
	// Group sections by label (repeated rows of the same kind).
	type group struct {
		label   string
		bytes   int
		indexes []int
	}
	var groups []*group
	byLabel := map[string]*group{}
	for i, sec := range s.sections {
		if sec.Label == "header" || sec.Label == "footer" || sec.Label == "divider" {
			continue
		}
		g := byLabel[sec.Label]
		if g == nil {
			g = &group{label: sec.Label}
			byLabel[sec.Label] = g
			groups = append(groups, g)
		}
		g.bytes += s.sectionBytes[i]
		g.indexes = append(g.indexes, i)
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].bytes > groups[j].bytes })
	if len(groups) == 0 {
		return "", nil
	}
	g := groups[0]
	share := float64(g.bytes) / float64(total)
	if share < 0.35 {
		return "", nil
	}
	what := "The " + g.label + " section is"
	if len(g.indexes) > 1 {
		what = fmt.Sprintf("The %s %s sections are", numberWord(len(g.indexes)), g.label)
	}
	sentence := fmt.Sprintf("%s the bulk of it (%d%%).", what, int(math.Round(share*100)))
	// Rows of cards: count the cards themselves ("Four product cards with inline CSS…").
	if strings.HasSuffix(g.label, "cards") && len(s.sectionCols) == len(s.sections) {
		cards, styles := 0, 0
		for _, i := range g.indexes {
			cards += s.sectionCols[i]
			styles += s.sectionStyles[i]
		}
		if cards > 1 {
			w := numberWord(cards)
			what = strings.ToUpper(w[:1]) + w[1:] + " " + g.label
			if styles >= cards*4 {
				what += " with inline CSS"
			}
			sentence = what + " are the bulk of it."
		}
	}
	last := g.indexes[len(g.indexes)-1]
	label := "Remove the " + g.label + " section"
	if len(g.indexes) > 1 {
		label = "Remove the " + ordinal(len(g.indexes)) + " row"
	}
	return sentence, &ContentCheckFix{Kind: "remove_section", Label: label, Path: []int{last}}
}

func numberWord(n int) string {
	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return fmt.Sprint(n)
}

func ordinal(n int) string {
	words := []string{"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth"}
	if n > 0 && n < len(words) {
		return words[n]
	}
	return fmt.Sprintf("%dth", n)
}

// imageShare estimates the share of the email's area covered by images,
// counting text at roughly 7 × 22 px per character.
func imageShare(images []image, text string) (float64, bool) {
	area := 0.0
	for _, im := range images {
		w, h := float64(im.w), float64(im.h)
		if w <= 2 && h <= 2 && w > 0 {
			continue
		}
		if w == 0 {
			w = 552
		}
		if h == 0 {
			h = w * 0.4
		}
		area += w * h
	}
	if area == 0 {
		return 0, false
	}
	textArea := float64(len([]rune(strings.Join(strings.Fields(text), " ")))) * 7 * 22
	return area / (area + textArea), true
}

func isUnsubscribe(l link) bool {
	h := strings.ToLower(l.href)
	if strings.Contains(h, "unsubscribe_url") || strings.Contains(h, "{% unsubscribe") {
		return true
	}
	return l.href != "" && l.href != "#" && strings.Contains(strings.ToLower(l.text), "unsubscribe")
}

var (
	postalRe = regexp.MustCompile(`(?i)\b\d{1,6}\s+[\p{L}.'-]+(\s+[\p{L}.'-]+){0,4}\s+(st|street|ave|avenue|rd|road|blvd|boulevard|ln|lane|dr|drive|way|ct|court|pl|place|sq|square|pkwy|parkway|hwy|highway|straße|strasse|rue|via|calle)\b`)
	zipRe    = regexp.MustCompile(`\b[A-Z]{2}\s+\d{5}(-\d{4})?\b|\b[A-Z]{1,2}\d[A-Z\d]?\s*\d[A-Z]{2}\b|\b\d{4,5}\s+\p{Lu}\p{Ll}+`)
)

func hasPostalAddress(text, address string) bool {
	if a := strings.TrimSpace(address); a != "" && strings.Contains(strings.ToLower(text), strings.ToLower(a)) {
		return true
	}
	if strings.Contains(text, "{{") && (strings.Contains(text, "workspace_address") || strings.Contains(text, "postal_address")) {
		return true
	}
	return postalRe.MatchString(text) || zipRe.MatchString(text)
}

// RegistrableDomain returns the domain someone registers ("acme.io" for
// "news.acme.io").
func RegistrableDomain(host string) string {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	d, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}
	return d
}

func linkHost(href string) (string, *url.URL) {
	h := strings.TrimSpace(href)
	if h == "" || strings.HasPrefix(h, "#") || strings.HasPrefix(h, "mailto:") || strings.HasPrefix(h, "tel:") || strings.Contains(h, "{{") || strings.Contains(h, "{%") {
		return "", nil
	}
	if strings.HasPrefix(h, "//") {
		h = "https:" + h
	}
	if !strings.Contains(h, "://") {
		h = "https://" + h
	}
	u, err := url.Parse(h)
	if err != nil || u.Hostname() == "" {
		return "", nil
	}
	return strings.ToLower(u.Hostname()), u
}

var shorteners = map[string]bool{
	"bit.ly": true, "tinyurl.com": true, "t.co": true, "goo.gl": true, "ow.ly": true, "is.gd": true, "buff.ly": true,
	"rebrand.ly": true, "cutt.ly": true, "shorturl.at": true, "tiny.cc": true, "rb.gy": true, "bl.ink": true,
	"lnkd.in": true, "s.id": true, "t.ly": true, "v.gd": true, "shorte.st": true, "adf.ly": true,
}

func (c *Checker) checkLinks(ctx context.Context, in Input, s scan, r *ContentCheckReport, add func(ContentCheckIssue)) {
	hosts := map[string]bool{}
	for _, l := range s.links {
		if host, _ := linkHost(l.href); host != "" {
			hosts[host] = true
			hosts[RegistrableDomain(host)] = true
		}
	}
	listings := map[string][]string{}
	if c.Blocklist != nil && len(hosts) > 0 {
		domains := make([]string, 0, len(hosts))
		for h := range hosts {
			domains = append(domains, h)
		}
		sort.Strings(domains)
		if l, err := c.Blocklist.Listings(ctx, domains); err == nil {
			listings = l
		}
	}
	own := ""
	if len(in.OwnDomains) > 0 {
		own = RegistrableDomain(in.OwnDomains[0])
	}
	var probe map[string]ProbeResult
	if in.ProbeLinks && c.Prober != nil {
		var urls []string
		seen := map[string]bool{}
		for _, l := range s.links {
			if host, u := linkHost(l.href); host != "" && !seen[u.String()] {
				seen[u.String()] = true
				urls = append(urls, u.String())
			}
		}
		probe = c.Prober.Probe(ctx, urls)
	}
	for _, l := range s.links {
		kind := linkKind(l.kind)
		if strings.TrimSpace(l.href) == "" || l.href == "#" {
			if l.kind == "button" {
				add(ContentCheckIssue{ID: "links.empty", Severity: Warn, Group: "Links", Title: fmt.Sprintf("Button “%s” has no link", truncate(l.text, 40)), Tag: "Button has no link",
					Detail: "People will click it; point it at the page it promises.", Path: l.path, Section: l.section, Fixes: []ContentCheckFix{{Kind: "edit_link", Label: "Edit link", Path: l.path}}})
			}
			continue
		}
		host, u := linkHost(l.href)
		if host == "" {
			continue
		}
		entry := ContentCheckLink{URL: l.href, Host: host, Text: l.text, Kind: l.kind, Path: l.path}
		if entry.Path == nil {
			entry.Path = []int{}
		}
		if pr, ok := probe[u.String()]; ok {
			entry.Probed, entry.Status, entry.Broken, entry.Millis, entry.Final = true, pr.Status, pr.Broken, pr.Millis, pr.Final
		}
		r.LinkList = append(r.LinkList, entry)
		lists := listings[host]
		if len(lists) == 0 {
			lists = listings[RegistrableDomain(host)]
		}
		if len(lists) > 0 {
			r.LinksBlocklisted++
			fixes := []ContentCheckFix{}
			if own != "" && RegistrableDomain(host) != own {
				target := own + u.EscapedPath()
				fixes = append(fixes, ContentCheckFix{Kind: "replace_link", Label: "Replace with " + target, Value: "https://" + target, Path: l.path})
			}
			fixes = append(fixes, ContentCheckFix{Kind: "edit_link", Label: "Edit link", Path: l.path})
			add(ContentCheckIssue{ID: "links.blocklist", Severity: Block, Group: "Links",
				Title:  fmt.Sprintf("%s links to %s, listed on %s", kind, host, joinAnd(lists)),
				Tag:    "Link domain on " + joinAnd(lists),
				Detail: "Mail linking to a blocklisted domain gets filtered regardless of your reputation. Use a domain you control.",
				Path:   l.path, Section: l.section, Fixes: fixes, URL: l.href})
			continue
		}
		if shorteners[host] {
			add(ContentCheckIssue{ID: "links.shortener", Severity: Warn, Group: "Links", Title: fmt.Sprintf("%s redirects through %s, a URL shortener", kind, host), Tag: "Shortened link",
				Detail: "Shortened links hide where they go; filters treat them as suspicious and many shortener domains are blocklisted. Link to the page directly.",
				Path:   l.path, Section: l.section, Fixes: []ContentCheckFix{{Kind: "edit_link", Label: "Edit link", Path: l.path}}, URL: l.href})
			continue
		}
		if probe != nil {
			if pr, ok := probe[u.String()]; ok && pr.Broken {
				r.LinksBroken++
				what := fmt.Sprintf("returns %d", pr.Status)
				if pr.Status == 0 {
					what = "doesn't respond"
				}
				add(ContentCheckIssue{ID: "links.broken", Severity: Warn, Group: "Links", Title: fmt.Sprintf("%s to %s %s", kind, truncate(u.String(), 60), what), Tag: "Broken link",
					Detail: "People who click get an error page. Fix the address or remove the link.", Path: l.path, Section: l.section,
					Fixes: []ContentCheckFix{{Kind: "edit_link", Label: "Edit link", Path: l.path}}, URL: l.href})
			}
		}
	}
}

func linkKind(k string) string {
	switch k {
	case "button":
		return "Button"
	case "image":
		return "Image"
	case "nav":
		return "Menu link"
	case "social":
		return "Social link"
	}
	return "Link"
}

func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

var (
	repeatedPunct = regexp.MustCompile(`[!?]{2,}|\.{4,}`)
	urgencyRe     = regexp.MustCompile(`(?i)\b(act now|urgent|hurry|last chance|limited time|today only|only|expires?|ending soon|ends (today|tonight|soon)|\d+\s+hours?|now or never|final hours|don'?t miss|before it'?s gone)\b`)
	moneyRe       = regexp.MustCompile(`(?i)(\$\$+|\bfree\b|\b100%\s+free\b|\bguaranteed?\b|\bcash\b|\bno cost\b|\brisk[- ]free\b)`)
	letterRe      = regexp.MustCompile(`\p{L}`)
	upperRe       = regexp.MustCompile(`\p{Lu}`)
)

// SubjectHeuristicsIn returns the heuristics a subject in a language trips:
// the English phrase lists (urgency, money words) apply to English and to
// emails without a language; other languages get the language-neutral ones
// (capitals, punctuation).
func SubjectHeuristicsIn(subject, language string) []string {
	h := SubjectHeuristics(subject)
	lang := strings.ToLower(strings.TrimSpace(language))
	if i := strings.IndexAny(lang, "-_"); i >= 0 {
		lang = lang[:i]
	}
	if lang == "" || lang == "en" {
		return h
	}
	out := []string{}
	for _, x := range h {
		if x == "all caps" || x == "repeated punctuation" {
			out = append(out, x)
		}
	}
	return out
}

// SubjectHeuristics returns the spam heuristics a subject trips: "all caps",
// "repeated punctuation", "urgency", "money words".
func SubjectHeuristics(subject string) []string {
	out := []string{}
	letters := len(letterRe.FindAllString(subject, -1))
	if letters >= 6 && float64(len(upperRe.FindAllString(subject, -1)))/float64(letters) >= 0.7 {
		out = append(out, "all caps")
	}
	if repeatedPunct.MatchString(subject) {
		out = append(out, "repeated punctuation")
	}
	if urgencyRe.MatchString(subject) {
		out = append(out, "urgency")
	}
	if moneyRe.MatchString(subject) {
		out = append(out, "money words")
	}
	return out
}

var (
	suggestPunct    = regexp.MustCompile(`\s*[!?]{2,}\s*`)
	suggestHours    = regexp.MustCompile(`(?i)\b(\d+)\s+(hour|day|minute)s?(\s+only)?\b`)
	suggestUrgent   = regexp.MustCompile(`(?i)\s*\b(act now|hurry|last chance|today only|don'?t miss( out)?|now or never|only)\b\s*`)
	suggestLeadTerm = regexp.MustCompile(`^([^:]+):\s+(\d+-(?:hour|day|minute))$`)
	suggestSpaces   = regexp.MustCompile(`\s{2,}`)
)

// SuggestSubject rewrites a shouty subject so that it trips none of the
// heuristics: sentence case, repeated punctuation collapsed and urgency
// softened ("FLASH SALE!!! 48 HOURS ONLY" → "48-hour flash sale"). It
// returns "" when no such rewrite exists.
func SuggestSubject(subject string) string {
	s := strings.TrimSpace(subject)
	letters := len(letterRe.FindAllString(s, -1))
	if letters > 0 && float64(len(upperRe.FindAllString(s, -1)))/float64(letters) >= 0.7 {
		s = strings.ToLower(s)
	}
	s = suggestPunct.ReplaceAllString(s, ": ")
	s = suggestHours.ReplaceAllString(s, "$1-$2")
	s = suggestUrgent.ReplaceAllString(s, " ")
	s = suggestSpaces.ReplaceAllString(strings.TrimSpace(s), " ")
	s = strings.TrimSpace(strings.TrimSuffix(s, ":"))
	// "flash sale: 48-hour" reads better as "48-hour flash sale".
	if m := suggestLeadTerm.FindStringSubmatch(s); m != nil {
		s = m[2] + " " + strings.ToLower(m[1])
	}
	if s == "" {
		return ""
	}
	// Sentence case: a capital only when the subject starts with a letter.
	if r := []rune(s); letterRe.MatchString(string(r[0])) {
		s = strings.ToUpper(string(r[0])) + string(r[1:])
	}
	if len(SubjectHeuristics(s)) > 0 {
		return ""
	}
	return s
}
