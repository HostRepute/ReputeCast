package engine

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	osteele "github.com/osteele/liquid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/linkrule"
	"github.com/hostrepute/reputecast/internal/liquid"
)

// Content is what one email renders from: the subject, preheader, compiled
// HTML (MJML compiled by the composer, Liquid still inside) and plain text.
type Content struct {
	Subject   string
	Preheader string
	HTML      string
	Text      string
}

// TrackedLink is a link of the email that goes through the click redirect.
// Position is its index in campaign_links (and in click tokens); URL may
// contain Liquid, which the redirect renders for the clicking contact.
type TrackedLink struct {
	Position int
	URL      string
	Label    string
}

// Template is an email prepared once per send: Liquid parsed, links indexed
// and rewritten to per-recipient tracking variables, the open pixel added.
// Rendering a recipient only fills in variables. A Template is safe for
// concurrent use.
type Template struct {
	subject *osteele.Template
	text    *osteele.Template
	// tracked has our pixel and click redirects; plain is the same email
	// without them, for providers whose own tracking the workspace keeps.
	tracked *osteele.Template
	plain   *osteele.Template
	// Links lists the tracked links in position order.
	Links []TrackedLink
	// Opens and Clicks say what the tracked variant tracks.
	Opens, Clicks bool
	// source is the subject, HTML and text with blocks expanded: what the
	// per-reader variables are looked up for (Source).
	source string
}

// Source is the template's subject, HTML and text, blocks expanded.
func (t *Template) Source() string { return t.source }

// Tracking options of a campaign (campaigns.tracking).
type Tracking struct {
	Opens  bool `json:"opens"`
	Clicks bool `json:"clicks"`
}

// Variables the tracked HTML expects for each recipient.
const (
	varOpenURL = "rc_open_url"
	varLink    = "rc_link_"
)

var (
	// Not osteele.NewEngine: the plain-text part derived from the HTML is
	// parsed here without liquid.Validate, so this engine can't load
	// templates or files either.
	engine     = liquid.NewEngine()
	bodyCloseR = regexp.MustCompile(`(?i)</body\s*>`)
)

// ErrRender wraps personalization errors in an email.
var ErrRender = errors.New("the email's personalization doesn't render")

// Prepare parses an email's Liquid once and indexes its links. Library
// image paths must already be absolute (media.Absolutize).
func Prepare(c Content, t Tracking) (*Template, error) {
	return PrepareCut(c, t, nil)
}

// PrepareCut is Prepare for one version of an email with conditional
// blocks: cut removes the blocks a recipient doesn't get (and the markers)
// after the links were numbered on the whole email, so every version keeps
// the link positions of campaign_links. A nil cut only removes markers.
func PrepareCut(c Content, t Tracking, cut func(string) string) (*Template, error) {
	if cut == nil {
		cut = stripMarkers
	}
	// Blocks that render per reader (Grow's) become Liquid first, so their
	// links are indexed and tracked like the email's own.
	c.HTML = liquid.ExpandBlocks(c.HTML)
	for _, src := range []string{c.Subject, c.HTML, c.Text} {
		if probs := liquid.Validate(src); len(probs) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrRender, probs[0].Message)
		}
	}
	out := &Template{Opens: t.Opens, Clicks: t.Clicks, source: c.Subject + "\n" + c.HTML + "\n" + c.Text}
	var err error
	if out.subject, err = parse(c.Subject); err != nil {
		return nil, err
	}
	plainHTML := cut(c.HTML)
	text := c.Text
	if strings.TrimSpace(text) == "" {
		// Derive the plain-text part once; Liquid inside text nodes survives
		// and is filled in per recipient.
		text = contentcheck.PlainText(plainHTML)
	}
	if out.text, err = parse(text); err != nil {
		return nil, err
	}
	if out.plain, err = parse(plainHTML); err != nil {
		return nil, err
	}
	trackedHTML := c.HTML
	if t.Clicks {
		trackedHTML, out.Links = rewriteLinks(trackedHTML)
	}
	trackedHTML = cut(trackedHTML)
	if t.Opens {
		trackedHTML = addPixel(trackedHTML)
	}
	if t.Clicks {
		// The hidden link only machines follow (honeypot.go).
		trackedHTML = addHoneypot(trackedHTML)
	}
	if out.tracked, err = parse(trackedHTML); err != nil {
		return nil, err
	}
	return out, nil
}

// stripMarkers removes conditional blocks' markers (mjml.PrepareCompile)
// from an email every recipient gets whole.
func stripMarkers(html string) string {
	if !strings.Contains(html, "<!--rc:") {
		return html
	}
	return markerRe.ReplaceAllString(html, "")
}

var markerRe = regexp.MustCompile(`<!--rc:(?:if|fi) [a-z0-9]{1,24}-->`)

func parse(src string) (*osteele.Template, error) {
	tpl, err := engine.ParseString(liquid.Prepare(src))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRender, err.Error())
	}
	return tpl, nil
}

// rewriteLinks replaces every tracked link's href with a per-recipient
// variable and returns the links in order. Which links are tracked is
// internal/linkrule's rule, the one the report's click map uses too
// (review AN-5).
func rewriteLinks(src string) (string, []TrackedLink) {
	out, ls := linkrule.Rewrite(src, func(l linkrule.Link) string { return "{{ " + varLink + strconv.Itoa(l.Position) + " }}" })
	links := make([]TrackedLink, 0, len(ls))
	for _, l := range ls {
		links = append(links, TrackedLink{Position: l.Position, URL: l.URL, Label: l.Label})
	}
	return out, links
}

// pixel is the open-tracking image: 1×1, no alt text to announce, block so
// it adds no line height.
const pixel = `<img src="{{ ` + varOpenURL + ` }}" width="1" height="1" alt="" style="display:block;width:1px;height:1px;border:0;margin:0;padding:0;overflow:hidden" />`

func addPixel(src string) string {
	if loc := bodyCloseR.FindStringIndex(src); loc != nil {
		return src[:loc[0]] + pixel + src[loc[0]:]
	}
	return src + pixel
}

// Rendered is one recipient's email.
type Rendered struct {
	Subject string
	HTML    string
	Text    string
}

// maxOutput bounds one rendered part (liquid.MaxOutput).
const maxOutput = liquid.MaxOutput

type limitWriter struct{ b strings.Builder }

var errTooLarge = errors.New("the rendered email is larger than 4 MB")

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.b.Len()+len(p) > maxOutput {
		return 0, errTooLarge
	}
	return w.b.Write(p)
}

func renderOne(t *osteele.Template, vars map[string]any) (string, error) {
	w := &limitWriter{}
	if err := t.FRender(io.Writer(w), osteele.Bindings(vars)); err != nil {
		if strings.Contains(err.Error(), errTooLarge.Error()) {
			return "", fmt.Errorf("%w: %s", ErrRender, errTooLarge.Error())
		}
		return "", fmt.Errorf("%w: %s", ErrRender, err.Error())
	}
	return w.b.String(), nil
}

// Render fills in one recipient. vars come from liquid.Vars plus the
// per-recipient links; tracked selects our pixel and click redirects.
func (t *Template) Render(vars map[string]any, tracked bool) (Rendered, error) {
	var out Rendered
	var err error
	if out.Subject, err = renderOne(t.subject, vars); err != nil {
		return out, err
	}
	out.Subject = strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(out.Subject))
	body := t.plain
	if tracked {
		body = t.tracked
	}
	if out.HTML, err = renderOne(body, vars); err != nil {
		return out, err
	}
	if out.Text, err = renderOne(t.text, vars); err != nil {
		return out, err
	}
	return out, nil
}

// LinkVar is the Liquid variable holding the redirect URL of a link.
func LinkVar(position int) string { return varLink + strconv.Itoa(position) }

// OpenVar is the Liquid variable holding the open pixel URL.
const OpenVar = varOpenURL
