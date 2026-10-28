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
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/publicsuffix"

	"github.com/hostrepute/reputecast/internal/i18n"
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

// ImageShareLimit is the share of the email's area images may cover before
// the content checks warn (one threshold for the composer and Send
// Readiness): image-heavy mail scores higher with spam filters.
const ImageShareLimit = 0.6

// ContentCheckFix is a one-click fix offered with an issue. The composer applies it to
// the canvas (Path points at the component) or to the campaign settings; an
// open fix opens the page in Value (a Sponsor block's booking).
type ContentCheckFix struct {
	Kind  string `json:"kind" enum:"replace_link,edit_link,insert_footer,set_alt,minify,remove_section,use_subject,open"`
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
	Path  []int  `json:"path,omitempty"`
	// LabelText is Label as a message, to write it in a reader's language
	// (ContentCheckReport.In).
	LabelText i18n.Text `json:"-"`
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
	// TitleText, DetailText (one message a sentence) and TagText are the
	// issue's words as messages, to write them in a reader's language
	// (ContentCheckReport.In, TextIn). Title, Detail and Tag are English.
	// Issues another domain adds without them keep their English.
	TitleText  i18n.Text   `json:"-"`
	DetailText []i18n.Text `json:"-"`
	TagText    i18n.Text   `json:"-"`
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
	// SubjectHeuristics, PreheaderHeuristics and BodyHeuristics name the
	// spam heuristics the subject, the preheader and the body's text trip.
	SubjectHeuristics   []string              `json:"subjectHeuristics"`
	PreheaderHeuristics []string              `json:"preheaderHeuristics"`
	BodyHeuristics      []string              `json:"bodyHeuristics"`
	Issues              []ContentCheckIssue   `json:"issues"`
	Sections            []ContentCheckSection `json:"sections"`
	PlainText           string                `json:"plainText"`
	CheckedAt           time.Time             `json:"checkedAt"`
	// ImageShare is the estimated share of the email's area covered by
	// images (0 to 1; 0 without images). Words counts the plain text.
	ImageShare float64 `json:"imageShare"`
	Words      int     `json:"words"`
	// ImagesInsecure counts images loaded over plain HTTP; ImageList is
	// every image with, when images were requested, the response.
	ImagesInsecure int                 `json:"imagesInsecure"`
	ImageList      []ContentCheckImage `json:"imageList"`
	// LinkList is every link with its text and, when links were probed, the
	// response (Send Readiness shows the slowest and the link domains).
	LinkList []ContentCheckLink `json:"linkList"`
	// LinkDomains counts the link domains looked up on the URI blocklists
	// (URILists names them); LinkDomainsUnknown are those whose lookup
	// failed: unknown, not clean.
	LinkDomains        int                         `json:"linkDomains"`
	URILists           []string                    `json:"uriLists"`
	LinkDomainsUnknown []ContentCheckUnknownDomain `json:"linkDomainsUnknown"`
}

// ContentCheckUnknownDomain is a link domain that couldn't be looked up.
type ContentCheckUnknownDomain struct {
	Domain string `json:"domain"`
	// Trouble is why: timeout, refused (the list refused the resolver) or other.
	Trouble string `json:"trouble" enum:"timeout,refused,other"`
}

// ContentCheckImage is one image of the email.
type ContentCheckImage struct {
	URL string `json:"url"`
	// Host is where the image loads from; empty for the media library's
	// images (relative until the send makes them absolute) and for images
	// whose address is personalized.
	Host string `json:"host,omitempty"`
	// Insecure is true for an image loaded over plain HTTP.
	Insecure bool  `json:"insecure,omitempty"`
	Path     []int `json:"path"`
	// Probed is true when the image was requested (Send Readiness); Status,
	// Broken and Bytes (the size the server reported, 0 when it didn't)
	// describe the response.
	Probed bool  `json:"probed,omitempty"`
	Status int   `json:"status,omitempty"`
	Broken bool  `json:"broken,omitempty"`
	Bytes  int64 `json:"bytes,omitempty"`
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
	// FinalClean is true when Final, the address the link really leads to,
	// was looked up on the URI blocklists and found on none: only then is a
	// listed or shortened link offered "Replace with destination URL".
	FinalClean bool `json:"finalClean,omitempty"`
	// BotBlocked is true when the site refused the automated request.
	BotBlocked bool `json:"botBlocked,omitempty"`
}

// Input is the content to check.
type Input struct {
	Subject   string
	Preheader string
	MJML      string
	HTML      string
	// Address is the postal address from the workspace's branding, if set.
	Address string
	// Fields are the personalization tokens that exist (liquid.Catalog keys).
	Fields []string
	// ProbeLinks requests each link (Send Readiness); the composer skips it.
	ProbeLinks bool
	// Language is the language version's code ("de"); the subject's
	// English phrase heuristics only apply to English (or no language).
	Language string
	// WorkspaceID is whose content it is (Extra checks read its data).
	WorkspaceID uuid.UUID
}

// Extra is a check another domain adds (X20: a store block naming a
// product the store removed). It returns its issues; Section and Path say
// where they are.
type Extra func(ctx context.Context, in Input) []ContentCheckIssue

// Blocklist answers which URI blocklists list a domain.
type Blocklist interface {
	// Listings returns, for each listed domain, the lists it is on. Clean
	// and unknown domains are absent.
	Listings(ctx context.Context, domains []string) (map[string][]string, error)
}

// ProbeResult is the outcome of requesting one link.
type ProbeResult struct {
	Status int  `json:"status"`
	Broken bool `json:"broken"`
	// BotBlocked is true when the site refused an automated request (999,
	// 429): whether the link works isn't known.
	BotBlocked bool   `json:"botBlocked,omitempty"`
	Error      string `json:"error,omitempty"`
	// Bytes is the size the server reported (Content-Length), 0 when it
	// didn't.
	Bytes int64 `json:"bytes,omitempty"`
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
	// Extra checks run after the built-in ones.
	Extra []Extra
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
	r := ContentCheckReport{ClipBytes: ClipBytes, Issues: []ContentCheckIssue{}, Sections: []ContentCheckSection{}, SubjectHeuristics: []string{}, LinkList: []ContentCheckLink{}, CheckedAt: now(),
		URILists: []string{}, LinkDomainsUnknown: []ContentCheckUnknownDomain{}, PreheaderHeuristics: SubjectHeuristicsIn(in.Preheader, in.Language),
		BodyHeuristics: []string{}}
	htmlSrc := in.HTML
	if htmlSrc == "" && strings.TrimSpace(in.MJML) != "" {
		htmlSrc = mjml.Compile(in.MJML).HTML
	}
	r.SizeBytes = len(htmlSrc)
	s := c.scanContent(in.MJML, htmlSrc)
	r.Images = len(s.images)
	r.ImageList = imageList(s.images)
	for _, im := range r.ImageList {
		if im.Insecure {
			r.ImagesInsecure++
		}
	}
	r.Links = len(s.links)
	r.PlainText = PlainText(htmlSrc)
	r.Words = len(strings.Fields(r.PlainText))
	r.BodyHeuristics = BodyHeuristics(r.PlainText, in.Language)
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
		add(worded(ContentCheckIssue{ID: "content.alt_text", Severity: Warn, Group: "Content", Path: im.path, Section: im.section,
			Fixes: []ContentCheckFix{fixOf("set_alt", mAltFix.Keep(nil), im.path)}},
			mAltTitle.Keep(imageArgs(im, s.sections)), mAltTag.Keep(nil), mAltDetail.Keep(nil)))
	}

	if r.SizeBytes >= NearClipBytes {
		kb := i18n.Args{"kb": fmt.Sprint(int(math.Round(float64(r.SizeBytes) / 1024)))}
		title := mSizeNear.Keep(kb)
		if r.SizeBytes > ClipBytes {
			title = mSizeOver.Keep(kb)
		}
		detail := []i18n.Text{mSizeDetail.Keep(nil)}
		fixes := []ContentCheckFix{}
		if saved := r.SizeBytes - len(Minify(htmlSrc)); saved >= 1024 {
			fixes = append(fixes, fixOf("minify", mSizeMinify.Keep(i18n.Args{"kb": fmt.Sprint(int(math.Round(float64(saved) / 1024)))}), nil))
		}
		if bulk, fix := bulkOf(s, r.SizeBytes); !bulk.IsZero() {
			detail = append(detail, bulk)
			if fix != nil {
				fixes = append(fixes, *fix)
			}
		}
		add(worded(ContentCheckIssue{ID: "content.size", Severity: Warn, Group: "Content", Section: -1, Fixes: fixes}, title, i18n.Text{}, detail...))
	}

	if share, ok := imageShare(s.images, r.PlainText); ok && share >= ImageShareLimit {
		pct := math.Round(share * 100)
		add(worded(ContentCheckIssue{ID: "content.image_ratio", Severity: Warn, Group: "Content", Section: -1},
			mRatioTitle.Keep(i18n.Args{"share": percentArg(pct)}), i18n.Text{}, mRatioDetail.Keep(nil)))
	}

	// Links: blocklists, shorteners, empty links, broken links.
	c.checkLinks(ctx, in, s, &r, add)
	if in.ProbeLinks && c.Prober != nil {
		c.probeImages(ctx, r.ImageList)
	}

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
		title, tag := mFooterBoth, mFooterBothTag
		switch {
		case hasUnsub:
			title, tag = mFooterAddress, mFooterAddressTag
		case hasAddress:
			title, tag = mFooterUnsubscribe, mFooterUnsubscribeTag
		}
		add(worded(ContentCheckIssue{ID: "compliance.footer", Severity: Block, Group: "Compliance", Section: -1,
			Fixes: []ContentCheckFix{fixOf("insert_footer", mFooterFix.Keep(nil), nil)}},
			title.Keep(nil), tag.Keep(nil), mFooterDetail.Keep(nil)))
	}

	// Personalization (liquid.Check, C8): a tag that doesn't parse stops the
	// email from rendering; a field that doesn't exist is flagged even with
	// a fallback, which every recipient would get.
	for _, src := range []struct{ where, text string }{{"subject", in.Subject}, {"preheader", in.Preheader}, {"body", bodySource(in.MJML, htmlSrc)}} {
		for _, p := range liquid.Check(src.text, nil) {
			title := mLiquidTitle.Keep(i18n.Args{"where": src.where, "problem": p.Message})
			// "Line 3: …" names its line in the reader's language; the
			// template engine's explanation after it stays English.
			if rest, ok := strings.CutPrefix(p.Message, fmt.Sprintf("Line %d: ", p.Line)); ok && p.Line > 0 {
				title = mLiquidTitleLine.Keep(i18n.Args{"where": src.where, "line": strconv.Itoa(p.Line), "problem": rest})
			}
			add(worded(ContentCheckIssue{ID: "personalization.syntax", Severity: Block, Group: "Personalization", Section: -1},
				title, i18n.Text{}, mLiquidDetail.Keep(nil)))
		}
	}
	if len(in.Fields) > 0 {
		for _, is := range liquid.Check(in.Subject+"\n"+in.Preheader+"\n"+bodySource(in.MJML, htmlSrc), in.Fields) {
			if is.Kind != liquid.IssueUnknownField {
				continue
			}
			token := "{{ " + is.Field + " }}"
			title, detail := mUnknownFieldTitle.Keep(i18n.Args{"token": token}), mUnknownFieldDetail.Keep(nil)
			if is.HasFallback {
				title, detail = mUnknownFieldFallback.Keep(i18n.Args{"token": token, "fallback": is.Fallback}), mUnknownFieldFallbackDetail.Keep(nil)
			}
			add(worded(ContentCheckIssue{ID: "personalization.unknown_field", Severity: Warn, Group: "Personalization", Section: -1}, title, i18n.Text{}, detail))
		}
	}

	if strings.TrimSpace(in.Subject) == "" {
		add(worded(ContentCheckIssue{ID: "subject.missing", Severity: Block, Group: "Subject", Section: -1}, mSubjectMissing.Keep(nil), i18n.Text{}, mSubjectMissingDet.Keep(nil)))
	} else if h := SubjectHeuristicsIn(in.Subject, in.Language); len(h) > 0 {
		r.SubjectHeuristics = h
		suggestion := SuggestSubject(in.Subject)
		is := ContentCheckIssue{ID: "subject.spam", Severity: Warn, Group: "Subject", Section: -1}
		var detail []i18n.Text
		if suggestion != "" && suggestion != in.Subject {
			detail = append(detail, mSubjectSuggested.Keep(i18n.Args{"suggestion": strconv.Quote(suggestion)}))
			fix := fixOf("use_subject", mSubjectUseSuggested.Keep(nil), nil)
			fix.Value = suggestion
			is.Fixes = []ContentCheckFix{fix}
		}
		add(worded(is, mSubjectSpam.Keep(i18n.Args{"count": len(h)}), i18n.Text{}, detail...))
	}

	if strings.TrimSpace(r.PlainText) == "" && r.Images > 0 {
		add(worded(ContentCheckIssue{ID: "content.plain_text", Severity: Warn, Group: "Content", Section: -1}, mPlainTextTitle.Keep(nil), i18n.Text{}, mPlainTextDetail.Keep(nil)))
	}

	for _, x := range c.Extra {
		for _, is := range x(ctx, in) {
			add(is)
		}
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

// AddIssues adds findings of other domains (the Sponsor block's) to a
// report: they count in its verdict and in their sections.
func (r *ContentCheckReport) AddIssues(issues ...ContentCheckIssue) {
	if len(issues) == 0 {
		return
	}
	for _, is := range issues {
		if is.Fixes == nil {
			is.Fixes = []ContentCheckFix{}
		}
		if is.Path == nil {
			is.Path = []int{}
		}
		r.Issues = append(r.Issues, is)
		for i := range r.Sections {
			if r.Sections[i].Index == is.Section && is.Section >= 0 && (is.Severity == Block || is.Severity == Warn) {
				r.Sections[i].Issues++
			}
		}
	}
	sort.SliceStable(r.Issues, func(i, j int) bool { return rank(r.Issues[i]) < rank(r.Issues[j]) })
	r.Blocks, r.Warnings = 0, 0
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

// imageArgs name an image for alt.title: the logo, the hero image, an
// image in a section, or an image.
func imageArgs(im image, sections []mjml.Section) i18n.Args {
	if im.inHeader {
		return i18n.Args{"image": "logo"}
	}
	if im.section >= 0 && im.section < len(sections) {
		switch l := sections[im.section].Label; l {
		case "hero":
			return i18n.Args{"image": "hero"}
		case "header":
			return i18n.Args{"image": "logo"}
		default:
			return i18n.Args{"image": "section", "section": l}
		}
	}
	return i18n.Args{"image": "other"}
}

// bulkOf explains which sections make up most of the size and offers to
// remove the largest one.
func bulkOf(s scan, size int) (i18n.Text, *ContentCheckFix) {
	if len(s.sections) < 3 || len(s.sectionBytes) != len(s.sections) {
		return i18n.Text{}, nil
	}
	total := 0
	for _, b := range s.sectionBytes {
		total += b
	}
	if total == 0 {
		return i18n.Text{}, nil
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
		return i18n.Text{}, nil
	}
	g := groups[0]
	share := float64(g.bytes) / float64(total)
	if share < 0.35 {
		return i18n.Text{}, nil
	}
	args := i18n.Args{"section": g.label, "share": percentArg(math.Round(share * 100))}
	sentence := mSizeBulkOne.Keep(args)
	if len(g.indexes) > 1 {
		args["count"] = len(g.indexes)
		sentence = mSizeBulkMany.Keep(args)
	}
	// Rows of cards: count the cards themselves ("Four product cards with inline CSS…").
	if strings.HasSuffix(g.label, "cards") && len(s.sectionCols) == len(s.sections) {
		cards, styles := 0, 0
		for _, i := range g.indexes {
			cards += s.sectionCols[i]
			styles += s.sectionStyles[i]
		}
		if cards > 1 {
			css := "no"
			if styles >= cards*4 {
				css = "yes"
			}
			sentence = mSizeBulkCards.Keep(i18n.Args{"count": cards, "section": g.label, "css": css})
		}
	}
	last := g.indexes[len(g.indexes)-1]
	label := mSizeRemoveSection.Keep(i18n.Args{"section": g.label})
	if len(g.indexes) > 1 {
		label = mSizeRemoveRow.Keep(i18n.Args{"n": len(g.indexes)})
	}
	fix := fixOf("remove_section", label, []int{last})
	return sentence, &fix
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

// imageList lists the images with where they load from. A relative
// address is the media library's (the send makes it absolute on the public
// URL); a personalized one isn't known before the send.
func imageList(images []image) []ContentCheckImage {
	out := make([]ContentCheckImage, 0, len(images))
	for _, im := range images {
		src := strings.TrimSpace(im.src)
		entry := ContentCheckImage{URL: src, Path: im.path}
		if entry.Path == nil {
			entry.Path = []int{}
		}
		lower := strings.ToLower(src)
		switch {
		case strings.HasPrefix(lower, "http://"):
			entry.Insecure = true
			entry.Host, _ = linkHost(src)
		case strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "//"):
			entry.Host, _ = linkHost(src)
		}
		out = append(out, entry)
	}
	return out
}

// probeImages requests the images that load from the web (Send Readiness):
// whether they load, and their size when the server says.
func (c *Checker) probeImages(ctx context.Context, images []ContentCheckImage) {
	var urls []string
	seen := map[string]bool{}
	for _, im := range images {
		if im.Host == "" || strings.Contains(im.URL, "{{") {
			continue
		}
		u := im.URL
		if strings.HasPrefix(u, "//") {
			u = "https:" + u
		}
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		return
	}
	// Progress is about links; the images' requests don't count there.
	res := c.Prober.Probe(WithProgress(ctx, func(string, int, int) {}), urls)
	for i := range images {
		u := images[i].URL
		if strings.HasPrefix(u, "//") {
			u = "https:" + u
		}
		if pr, ok := res[u]; ok {
			images[i].Probed, images[i].Status, images[i].Broken, images[i].Bytes = true, pr.Status, pr.Broken && !pr.BotBlocked, pr.Bytes
		}
	}
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
		if live, ok := c.Blocklist.(BlocklistChecker); ok {
			// Looked up live: a domain whose lookup failed is unknown.
			if rep, err := live.Check(ctx, domains); err == nil {
				listings, r.LinkDomains, r.URILists = rep.Listed, rep.Domains, rep.Lists
				for _, d := range domains {
					if why, ok := rep.Unknown[d]; ok {
						r.LinkDomainsUnknown = append(r.LinkDomainsUnknown, ContentCheckUnknownDomain{Domain: d, Trouble: why})
					}
				}
			} else {
				for _, d := range domains {
					r.LinkDomainsUnknown = append(r.LinkDomainsUnknown, ContentCheckUnknownDomain{Domain: d, Trouble: TroubleOther})
				}
				r.LinkDomains = len(domains)
			}
		} else if l, err := c.Blocklist.Listings(ctx, domains); err == nil {
			listings = l
		}
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
	clean := c.cleanDestinations(ctx, s.links, probe, listings)
	if n := len(r.LinkDomainsUnknown); n > 0 {
		// Not knowing isn't clean: say which lookups failed (Send Readiness
		// turns this into an Unknown check).
		trouble := r.LinkDomainsUnknown[0].Trouble
		lists := append([]string(nil), r.URILists...)
		add(worded(ContentCheckIssue{ID: "links.blocklist_unknown", Severity: Info, Group: "Links", Section: -1},
			mLinkUnknownTitle.Keep(i18n.Args{"count": n, "lists": lists}), mLinkUnknownTag.Keep(nil), mLinkUnknownDetail.Keep(i18n.Args{"trouble": trouble})))
	}
	editLink := func(path []int) ContentCheckFix { return fixOf("edit_link", mLinkEdit.Keep(nil), path) }
	// A listed or shortened link is replaced only with where it really
	// leads, once that address is known to be clean; ReputeCast never
	// guesses an address (R1.U3). Otherwise the link is edited by hand.
	fixesFor := func(l link, final string) []ContentCheckFix {
		fixes := []ContentCheckFix{}
		if final != "" && clean[final] {
			fix := fixOf("replace_link", mLinkReplaceDestination.Keep(nil), l.path)
			fix.Value = final
			fixes = append(fixes, fix)
		}
		return append(fixes, editLink(l.path))
	}
	for _, l := range s.links {
		kind := linkKind(l.kind)
		if strings.TrimSpace(l.href) == "" || l.href == "#" {
			if l.kind == "button" {
				add(worded(ContentCheckIssue{ID: "links.empty", Severity: Warn, Group: "Links", Path: l.path, Section: l.section, Fixes: []ContentCheckFix{editLink(l.path)}},
					mLinkEmptyTitle.Keep(i18n.Args{"text": truncate(l.text, 40)}), mLinkEmptyTag.Keep(nil), mLinkEmptyDetail.Keep(nil)))
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
			entry.Probed, entry.Status, entry.Broken, entry.Millis, entry.Final, entry.BotBlocked = true, pr.Status, pr.Broken, pr.Millis, pr.Final, pr.BotBlocked
			entry.FinalClean = pr.Final != "" && clean[pr.Final]
		}
		r.LinkList = append(r.LinkList, entry)
		lists := listings[host]
		if len(lists) == 0 {
			lists = listings[RegistrableDomain(host)]
		}
		if len(lists) > 0 {
			r.LinksBlocklisted++
			on := i18n.Args{"lists": append([]string(nil), lists...)}
			add(worded(ContentCheckIssue{ID: "links.blocklist", Severity: Block, Group: "Links", Path: l.path, Section: l.section, Fixes: fixesFor(l, entry.Final), URL: l.href},
				mLinkBlocklistTitle.Keep(i18n.Args{"kind": kind, "host": host, "lists": on["lists"]}), mLinkBlocklistTag.Keep(on), mLinkBlocklistDetail.Keep(nil)))
			continue
		}
		if shorteners[host] {
			add(worded(ContentCheckIssue{ID: "links.shortener", Severity: Warn, Group: "Links", Path: l.path, Section: l.section, Fixes: fixesFor(l, entry.Final), URL: l.href},
				mLinkShortenerTitle.Keep(i18n.Args{"kind": kind, "host": host}), mLinkShortenerTag.Keep(nil), mLinkShortenerDetail.Keep(nil)))
			continue
		}
		if probe != nil {
			if pr, ok := probe[u.String()]; ok && pr.Broken {
				r.LinksBroken++
				add(worded(ContentCheckIssue{ID: "links.broken", Severity: Warn, Group: "Links", Path: l.path, Section: l.section, Fixes: []ContentCheckFix{editLink(l.path)}, URL: l.href},
					mLinkBrokenTitle.Keep(i18n.Args{"kind": kind, "url": truncate(u.String(), 60), "status": pr.Status}), mLinkBrokenTag.Keep(nil), mLinkBrokenDetail.Keep(nil)))
			}
		}
	}
}

// cleanDestinations looks up where listed and shortened links really lead
// (the probe followed their redirects) on the URI blocklists, and returns
// the destinations found on none: the only addresses a fix may point such a
// link at. A destination on the link's own host, another shortener, or one
// whose lookup failed is never clean. Without a probe nothing is known, so
// nothing is clean.
func (c *Checker) cleanDestinations(ctx context.Context, links []link, probe map[string]ProbeResult, listings map[string][]string) map[string]bool {
	out := map[string]bool{}
	if c.Blocklist == nil || len(probe) == 0 {
		return out
	}
	listed := func(host string) bool { return len(listings[host]) > 0 || len(listings[RegistrableDomain(host)]) > 0 }
	finals := map[string]string{}
	domains := []string{}
	seen := map[string]bool{}
	for _, l := range links {
		host, u := linkHost(l.href)
		if host == "" || !(listed(host) || shorteners[host]) {
			continue
		}
		final := probe[u.String()].Final
		fhost, fu := linkHost(final)
		if fhost == "" || (fu.Scheme != "https" && fu.Scheme != "http") || fhost == host || shorteners[fhost] {
			continue
		}
		finals[final] = fhost
		for _, d := range []string{fhost, RegistrableDomain(fhost)} {
			if !seen[d] {
				seen[d] = true
				domains = append(domains, d)
			}
		}
	}
	if len(finals) == 0 {
		return out
	}
	sort.Strings(domains)
	// The link domains' progress is already reported: these lookups don't
	// count there.
	quiet := WithProgress(ctx, func(string, int, int) {})
	var bad map[string][]string
	unknown := map[string]string{}
	if live, ok := c.Blocklist.(BlocklistChecker); ok {
		rep, err := live.Check(quiet, domains)
		if err != nil {
			return out
		}
		bad, unknown = rep.Listed, rep.Unknown
	} else {
		l, err := c.Blocklist.Listings(quiet, domains)
		if err != nil {
			return out
		}
		bad = l
	}
	for final, host := range finals {
		ok := true
		for _, d := range []string{host, RegistrableDomain(host)} {
			if len(bad[d]) > 0 || unknown[d] != "" {
				ok = false
			}
		}
		out[final] = ok
	}
	return out
}

// linkKind is what a link is, for link.*Title's kind: button, image, nav,
// social or link.
func linkKind(k string) string {
	switch k {
	case "button", "image", "nav", "social":
		return k
	}
	return "link"
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
	// urgencyRe catches pressure phrasing. "Only" alone isn't urgency
	// ("Members only"), nor is a number of hours ("2 hours of Q&A"): only
	// "48 hours only", "only 3 days left" and the like are.
	urgencyRe = regexp.MustCompile(`(?i)\b(act now|urgent|hurry|last chance|limited time|today only|only today|expires?|ending soon|ends (today|tonight|soon)|\d+\s+(hours?|days?)\s+(only|left)|only\s+\d+\s+(hours?|days?|left)|now or never|final hours|don'?t miss|before it'?s gone)\b`)
	// moneyRe catches money words; "free" in a compound ("gluten-free",
	// "free-range", "toll-free") isn't one (moneyWords).
	moneyRe = regexp.MustCompile(`(?i)(\$\$+|\bfree\b|\bguaranteed?\b|\bcash\b|\bno cost\b|\brisk[- ]free\b)`)
	// clickbaitRe catches teaser phrases spam filters weigh, as the AI
	// assistant's subject ideas are checked against them too.
	clickbaitRe = regexp.MustCompile(`(?i)\b(you won'?t believe|you will not believe|can'?t believe|shocking|jaw[- ]dropping|mind[- ]blowing|what happens next|this one (weird )?trick|doctors hate)\b`)
	letterRe    = regexp.MustCompile(`\p{L}`)
	upperRe     = regexp.MustCompile(`\p{Lu}`)
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
	if len(moneyWords(subject)) > 0 {
		out = append(out, "money words")
	}
	if clickbaitRe.MatchString(subject) {
		out = append(out, "click-bait")
	}
	return out
}

// moneyWords are the money words of a text: "free" counts on its own, not
// inside a compound ("gluten-free", "free-range"), and "risk-free" counts.
func moneyWords(text string) []string {
	out := []string{}
	for _, loc := range moneyRe.FindAllStringIndex(text, -1) {
		w := text[loc[0]:loc[1]]
		if strings.EqualFold(w, "free") {
			before := loc[0] > 0 && text[loc[0]-1] == '-' || strings.HasSuffix(text[:loc[0]], "\u2011")
			after := loc[1] < len(text) && text[loc[1]] == '-'
			if before || after {
				continue
			}
		}
		out = append(out, strings.ToLower(w))
	}
	return out
}

// BodyHeuristics returns the spam heuristics an email's text trips: spam
// phrases repeated through the body (three or more urgency, money or
// click-bait phrases), and repeated exclamation marks. A word or two is
// normal; filters weigh the pile-up.
func BodyHeuristics(text, language string) []string {
	out := []string{}
	lang := strings.ToLower(strings.TrimSpace(language))
	if i := strings.IndexAny(lang, "-_"); i >= 0 {
		lang = lang[:i]
	}
	if lang == "" || lang == "en" {
		phrases := len(urgencyRe.FindAllString(text, -1)) + len(moneyWords(text)) + len(clickbaitRe.FindAllString(text, -1))
		if phrases >= 3 {
			out = append(out, "spam phrases")
		}
	}
	if len(repeatedPunct.FindAllString(text, -1)) >= 3 {
		out = append(out, "repeated punctuation")
	}
	return out
}

// SubjectHeuristicPhrase returns the words of a subject that trip a phrase
// heuristic ("you won't believe"), "" when only its form does (capitals,
// punctuation).
func SubjectHeuristicPhrase(subject, language string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	if i := strings.IndexAny(lang, "-_"); i >= 0 {
		lang = lang[:i]
	}
	if lang != "" && lang != "en" {
		return ""
	}
	for _, re := range []*regexp.Regexp{clickbaitRe, urgencyRe} {
		if m := re.FindString(subject); m != "" {
			return strings.ToLower(m)
		}
	}
	if w := moneyWords(subject); len(w) > 0 {
		return w[0]
	}
	return ""
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
