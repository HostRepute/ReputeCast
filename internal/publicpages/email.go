package publicpages

// Emails in the browser: "view online" (a signed link from one message) and
// the archive's issue pages. Both serve the campaign's own HTML document,
// rendered with Liquid, with metadata added to its <head>. Email HTML is
// built from inline styles, so these documents allow inline styles; they
// never run scripts.

import (
	"bytes"
	"encoding/json"
	"html"
	"net/http"
	"regexp"
	"strings"

	"github.com/hostrepute/reputecast/internal/liquid"
)

// renderEmail renders a campaign's subject and HTML for one reader. Contact
// fields come from the contact a link was signed for (empty for archive
// readers, so tokens use their fallbacks); links holds the system links
// (unsubscribe, preferences, view in browser). Relative media URLs become
// absolute on base so the document works on a custom domain too.
func renderEmail(subject, body, email, first, last string, attrs map[string]any, links map[string]string, base string) (string, string) {
	vars := liquid.Vars(email, first, last, attrs, links)
	s, err := liquid.Render(subject, vars)
	if err != nil {
		s = subject
	}
	b, err := liquid.Render(body, vars)
	if err != nil {
		b = body
	}
	return strings.Join(strings.Fields(s), " "), absolutizeMedia(b, base)
}

var mediaRef = regexp.MustCompile(`(?i)((?:src|href|background)\s*=\s*["'])(/media/)`)

// absolutizeMedia points /media/… references at the instance.
func absolutizeMedia(doc, base string) string {
	if base == "" {
		return doc
	}
	base = strings.TrimRight(base, "/")
	doc = mediaRef.ReplaceAllString(doc, "${1}"+base+"${2}")
	return strings.ReplaceAll(doc, "url(/media/", "url("+base+"/media/")
}

// firstImage is the first absolute image of a document (an issue's
// Open Graph image).
var imgSrc = regexp.MustCompile(`(?i)<img[^>]+src\s*=\s*["'](https?://[^"']+)["']`)

func firstImage(doc string) string {
	if m := imgSrc.FindStringSubmatch(doc); m != nil {
		return html.UnescapeString(m[1])
	}
	return ""
}

// emailHead is what goes into an email document's <head>.
type emailHead struct {
	Title       string
	Description string
	Robots      string
	Canonical   string
	OG          *og
	JSONLD      any
	Feed        string
	FeedTitle   string
	Icon        string
}

func (e emailHead) String() string {
	esc := html.EscapeString
	var b strings.Builder
	if e.Title != "" {
		b.WriteString("<title>" + esc(e.Title) + "</title>\n")
	}
	if e.Description != "" {
		b.WriteString(`<meta name="description" content="` + esc(e.Description) + `">` + "\n")
	}
	if e.Robots != "" {
		b.WriteString(`<meta name="robots" content="` + esc(e.Robots) + `">` + "\n")
	}
	if e.Canonical != "" {
		b.WriteString(`<link rel="canonical" href="` + esc(e.Canonical) + `">` + "\n")
	}
	if o := e.OG; o != nil {
		meta := func(prop, v string) {
			if v != "" {
				b.WriteString(`<meta property="` + prop + `" content="` + esc(v) + `">` + "\n")
			}
		}
		meta("og:type", o.Type)
		meta("og:site_name", o.SiteName)
		meta("og:title", o.Title)
		meta("og:description", o.Description)
		meta("og:url", o.URL)
		meta("og:image", o.Image)
		meta("article:published_time", o.Published)
		card := "summary"
		if o.Image != "" {
			card = "summary_large_image"
		}
		b.WriteString(`<meta name="twitter:card" content="` + card + `">` + "\n")
		b.WriteString(`<meta name="twitter:title" content="` + esc(o.Title) + `">` + "\n")
		if o.Description != "" {
			b.WriteString(`<meta name="twitter:description" content="` + esc(o.Description) + `">` + "\n")
		}
		if o.Image != "" {
			b.WriteString(`<meta name="twitter:image" content="` + esc(o.Image) + `">` + "\n")
		}
	}
	if e.Feed != "" {
		b.WriteString(`<link rel="alternate" type="application/rss+xml" title="` + esc(e.FeedTitle) + `" href="` + esc(e.Feed) + `">` + "\n")
	}
	if e.Icon != "" {
		b.WriteString(`<link rel="icon" href="` + esc(e.Icon) + `">` + "\n")
	}
	if e.JSONLD != nil {
		if raw, err := json.Marshal(e.JSONLD); err == nil {
			// json.Marshal escapes <, > and &, so the data can't close the tag.
			b.WriteString(`<script type="application/ld+json">` + string(raw) + "</script>\n")
		}
	}
	return b.String()
}

var (
	titleTag  = regexp.MustCompile(`(?is)<title[^>]*>.*?</title>`)
	robotsTag = regexp.MustCompile(`(?is)<meta[^>]+name\s*=\s*["']robots["'][^>]*>`)
	headOpen  = regexp.MustCompile(`(?is)<head(\s[^>]*)?>`)
	bodyOpen  = regexp.MustCompile(`(?is)<body(\s[^>]*)?>`)
	htmlOpen  = regexp.MustCompile(`(?is)<html(\s[^>]*)?>`)
)

// injectHead adds the metadata to an email document, replacing its own
// <title> and robots tag. Fragments without a <head> get one.
func injectHead(doc string, h emailHead) string {
	doc = titleTag.ReplaceAllString(doc, "")
	doc = robotsTag.ReplaceAllString(doc, "")
	meta := h.String()
	if loc := headOpen.FindStringIndex(doc); loc != nil {
		return doc[:loc[1]] + "\n" + meta + doc[loc[1]:]
	}
	head := "<head>\n<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n" + meta + "</head>\n"
	if loc := htmlOpen.FindStringIndex(doc); loc != nil {
		return doc[:loc[1]] + "\n" + head + doc[loc[1]:]
	}
	return "<!doctype html>\n<html lang=\"en\">\n" + head + "<body>\n" + doc + "\n</body>\n</html>\n"
}

// injectTop puts markup right after <body> (the archive's bar above an issue).
func injectTop(doc, top string) string {
	if loc := bodyOpen.FindStringIndex(doc); loc != nil {
		return doc[:loc[1]] + top + doc[loc[1]:]
	}
	return top + doc
}

// emailCSP is the policy of an email shown in the browser: inline styles
// (email HTML is made of them), images and fonts from anywhere, no script,
// no frames, no forms.
const emailCSP = "default-src 'none'; img-src 'self' https: http: data:; style-src 'self' 'unsafe-inline' https:; font-src 'self' https: data:; " +
	"form-action 'none'; frame-ancestors 'none'; base-uri 'none'; script-src 'none'"

// emailHeaders sets the headers of an email document. Private documents
// (view online) are never cached or indexed.
func emailHeaders(w http.ResponseWriter, private bool) {
	hd := w.Header()
	hd.Set("Content-Type", "text/html; charset=utf-8")
	hd.Set("Content-Security-Policy", emailCSP)
	hd.Set("X-Frame-Options", "DENY")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cross-Origin-Opener-Policy", "same-origin")
	if private {
		hd.Set("Cache-Control", "no-store")
		hd.Set("X-Robots-Tag", "noindex, nofollow")
		hd.Set("Referrer-Policy", "no-referrer")
	} else {
		hd.Set("Cache-Control", "public, max-age=300")
		hd.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	}
}

// plainDocument wraps a plain-text body when a campaign has no HTML.
func plainDocument(text string) string {
	var b bytes.Buffer
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head>`)
	b.WriteString(`<body style="margin:0;padding:32px 16px;background:#ffffff;color:#1f2937;font:15px/1.6 system-ui,sans-serif">`)
	b.WriteString(`<div style="max-width:600px;margin:0 auto;white-space:pre-wrap">`)
	b.WriteString(html.EscapeString(text))
	b.WriteString(`</div></body></html>`)
	return b.String()
}
