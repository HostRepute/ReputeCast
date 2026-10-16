package engine

import (
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"

	osteele "github.com/osteele/liquid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
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
}

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
	anchorRe   = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a\s*>`)
	hrefRe     = regexp.MustCompile(`(?is)(\shref\s*=\s*)("([^"]*)"|'([^']*)'|([^\s>"']+))`)
	tagRe      = regexp.MustCompile(`(?s)<[^>]*>`)
	spaceRe    = regexp.MustCompile(`\s+`)
	bodyCloseR = regexp.MustCompile(`(?i)</body\s*>`)
	systemLink = regexp.MustCompile(`^\{\{-?\s*(unsubscribe_url|preferences_url|view_in_browser_url)\b`)
)

// ErrRender wraps personalization errors in an email.
var ErrRender = errors.New("the email's personalization doesn't render")

// Prepare parses an email's Liquid once and indexes its links. Library
// image paths must already be absolute (media.Absolutize).
func Prepare(c Content, t Tracking) (*Template, error) {
	for _, src := range []string{c.Subject, c.HTML, c.Text} {
		if probs := liquid.Validate(src); len(probs) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrRender, probs[0].Message)
		}
	}
	out := &Template{Opens: t.Opens, Clicks: t.Clicks}
	var err error
	if out.subject, err = parse(c.Subject); err != nil {
		return nil, err
	}
	text := c.Text
	if strings.TrimSpace(text) == "" {
		// Derive the plain-text part once; Liquid inside text nodes survives
		// and is filled in per recipient.
		text = contentcheck.PlainText(c.HTML)
	}
	if out.text, err = parse(text); err != nil {
		return nil, err
	}
	if out.plain, err = parse(c.HTML); err != nil {
		return nil, err
	}
	trackedHTML := c.HTML
	if t.Clicks {
		trackedHTML, out.Links = rewriteLinks(trackedHTML)
	}
	if t.Opens {
		trackedHTML = addPixel(trackedHTML)
	}
	if out.tracked, err = parse(trackedHTML); err != nil {
		return nil, err
	}
	return out, nil
}

func parse(src string) (*osteele.Template, error) {
	tpl, err := engine.ParseString(liquid.Prepare(src))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRender, err.Error())
	}
	return tpl, nil
}

// rewriteLinks replaces every trackable href with a per-recipient variable
// and returns the links in order. Unsubscribe, preference and view-online
// links, anchors, mailto: and tel: links are left alone.
func rewriteLinks(src string) (string, []TrackedLink) {
	links := []TrackedLink{}
	out := anchorRe.ReplaceAllStringFunc(src, func(a string) string {
		m := anchorRe.FindStringSubmatch(a)
		attrs, inner := m[1], m[2]
		hm := hrefRe.FindStringSubmatchIndex(attrs)
		if hm == nil {
			return a
		}
		raw := ""
		for _, g := range []int{3, 4, 5} {
			if hm[2*g] >= 0 {
				raw = attrs[hm[2*g]:hm[2*g+1]]
				break
			}
		}
		href := strings.TrimSpace(html.UnescapeString(raw))
		if !trackable(href) {
			return a
		}
		pos := len(links)
		links = append(links, TrackedLink{Position: pos, URL: href, Label: linkLabel(inner)})
		newAttrs := attrs[:hm[0]] + attrs[hm[2]:hm[3]] + `"{{ ` + varLink + strconv.Itoa(pos) + ` }}"` + attrs[hm[1]:]
		return "<a" + newAttrs + ">" + inner + "</a>"
	})
	return out, links
}

func trackable(href string) bool {
	if href == "" || strings.HasPrefix(href, "#") {
		return false
	}
	lower := strings.ToLower(href)
	for _, p := range []string{"mailto:", "tel:", "sms:", "javascript:", "data:"} {
		if strings.HasPrefix(lower, p) {
			return false
		}
	}
	if systemLink.MatchString(href) || strings.Contains(href, "unsubscribe_url") || strings.Contains(href, "preferences_url") || strings.Contains(href, "view_in_browser_url") {
		return false
	}
	// Only absolute web links (possibly personalized) are redirected.
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(href, "{{")
}

func linkLabel(inner string) string {
	s := html.UnescapeString(tagRe.ReplaceAllString(inner, " "))
	s = strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
	if r := []rune(s); len(r) > 120 {
		s = string(r[:119]) + "…"
	}
	return s
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
