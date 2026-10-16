package systemmail

import (
	"bytes"
	"embed"
	"html"
	"html/template"
	"regexp"
	"strings"
	texttemplate "text/template"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/membermail"
)

//go:embed templates/*
var templateFS embed.FS

var (
	htmlLayout = template.Must(template.ParseFS(templateFS, "templates/layout.html"))
	textLayout = texttemplate.Must(texttemplate.ParseFS(templateFS, "templates/layout.txt"))
)

// Brand is who a system message comes from: ReputeCast (the instance) for
// messages to team members, the publisher for messages to subscribers.
type Brand struct {
	Name   string
	Accent string // hex colour of the brand mark
	Footer string // "Sent by Acme ReputeCast · reputecast.acme.io"
}

// subscriberKinds go to subscribers: they carry the publisher's brand, never
// ReputeCast's teal (public pages belong to the publisher).
var subscriberKinds = map[string]bool{"double_opt_in": true, "preference_link": true}

// reasons says why subscribers' messages were sent, under the body. Team
// mail says why in the member's language (internal/membermail).
var reasons = map[string]string{
	"double_opt_in":   "You get this email because this address was used to subscribe. If it wasn't you, ignore it and you won't hear from us again.",
	"preference_link": "You get this email because someone asked for a link to manage this address's subscriptions.",
}

// Rendered is a message ready for a provider.
type Rendered struct {
	HTML string
	Text string
}

// Render wraps a message's body in the branded layout of its kind: the
// brand mark and name, the body, and a footer that says why it was sent.
// Callers pass their own HTML fragment (trusted: they escape their values);
// messages with text only get it converted.
func Render(kind string, brand Brand, subject, text, body string) (Rendered, error) {
	return RenderIn(i18n.Default, kind, brand, subject, text, body)
}

// RenderIn is Render with the layout in the reader's language.
func RenderIn(l i18n.Lang, kind string, brand Brand, subject, text, body string) (Rendered, error) {
	reason, ok := membermail.Reason(l, kind)
	if !ok {
		// Subscribers' mail follows their content language, not the app's.
		if reason, ok = reasons[kind]; !ok {
			reason = membermail.DefaultReason(l, brand.Name)
		}
	}
	if strings.TrimSpace(body) == "" {
		body = TextToHTML(text, brand.Accent)
	}
	if strings.TrimSpace(text) == "" {
		text = HTMLToText(body)
	}
	initial := "R"
	if r, _ := utf8.DecodeRuneInString(brand.Name); r != utf8.RuneError {
		initial = strings.ToUpper(string(r))
	}
	accent := brand.Accent
	if !hexColor.MatchString(accent) {
		accent = "#0F766E"
	}
	lang := string(l.Locale)
	if lang == "" || subscriberKinds[kind] {
		lang = "en"
	}
	data := map[string]any{
		"Lang": lang, "Subject": subject, "BrandName": brand.Name, "Initial": initial,
		"Accent": template.CSS(accent), //nolint:gosec // a validated hex colour
		"Body":   template.HTML(body),  //nolint:gosec // the caller's own HTML fragment, its values already escaped
		"Reason": reason, "Footer": brand.Footer, "Preheader": firstLine(text),
	}
	var hb bytes.Buffer
	if err := htmlLayout.Execute(&hb, data); err != nil {
		return Rendered{}, err
	}
	var tb bytes.Buffer
	if err := textLayout.Execute(&tb, map[string]any{"Text": strings.TrimRight(text, "\n") + "\n", "Reason": reason, "Footer": brand.Footer}); err != nil {
		return Rendered{}, err
	}
	return Rendered{HTML: hb.String(), Text: tb.String()}, nil
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	if len(line) > 140 {
		line = line[:140]
	}
	return line
}

var (
	urlRe  = regexp.MustCompile(`https?://[^\s<>"']+[^\s<>"'.,;:!?)]`)
	paraRe = regexp.MustCompile(`\n\s*\n`)
)

// TextToHTML turns plain text into paragraphs with clickable links in the
// brand's colour.
func TextToHTML(text, linkColor string) string {
	if !hexColor.MatchString(linkColor) {
		linkColor = "#0F766E"
	}
	var b strings.Builder
	for _, para := range paraRe.Split(strings.TrimSpace(text), -1) {
		if strings.TrimSpace(para) == "" {
			continue
		}
		b.WriteString(`<p style="margin:0 0 12px">`)
		lines := strings.Split(para, "\n")
		for i, line := range lines {
			if i > 0 {
				b.WriteString("<br>")
			}
			last := 0
			for _, m := range urlRe.FindAllStringIndex(line, -1) {
				b.WriteString(html.EscapeString(line[last:m[0]]))
				u := line[m[0]:m[1]]
				b.WriteString(`<a href="` + html.EscapeString(u) + `" style="color:` + linkColor + `">` + html.EscapeString(u) + `</a>`)
				last = m[1]
			}
			b.WriteString(html.EscapeString(line[last:]))
		}
		b.WriteString("</p>")
	}
	return b.String()
}

var (
	tagRe   = regexp.MustCompile(`(?s)<[^>]*>`)
	breakRe = regexp.MustCompile(`(?i)<br\s*/?>|</p>`)
	hrefRe  = regexp.MustCompile(`(?is)<a\s[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
)

// HTMLToText is a rough plain-text version of an HTML fragment: links keep
// their URL, paragraphs become lines.
func HTMLToText(h string) string {
	h = hrefRe.ReplaceAllString(h, "$2: $1")
	h = breakRe.ReplaceAllString(h, "\n")
	h = tagRe.ReplaceAllString(h, "")
	return strings.TrimSpace(html.UnescapeString(h))
}
