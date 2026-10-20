package publicpages

import (
	"context"
	"html"
	"html/template"
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// The text message terms page (SMS, design expansion phase 8, X21): a
// built-in page at /sms-terms that the SMS settings write
// (internal/texting). Its words come written in the language of the terms,
// so the page has no language menu; consent records keep the version that
// was live.

// TextTerms writes the terms page. Nil, or a workspace that doesn't text,
// answers 404.
type TextTerms interface {
	TextTermsPage(ctx context.Context, ws *store.Workspace) (TermsPage, bool)
}

// TermsPage is the page with every word written.
type TermsPage struct {
	// Lang is the language the terms are written in.
	Lang string
	// Title is "Text message terms"; Version "Version 1 · in effect since
	// 1 Oct 2026".
	Title   string
	Version string
	Intro   string
	// Sections are the terms, heading by heading.
	Sections []TermsSection
	// Closing is the box at the end: "Changed your mind?" and how to stop.
	ClosingHeading string
	ClosingText    string
	// ManageLabel links the preference center; BackLabel and BackURL the
	// page where people give SMS consent (empty: no link).
	ManageLabel string
	BackLabel   string
	BackURL     string
	// Footer is the address and how versions work.
	Footer string
}

// TermsSection is a heading and its text. Links turn their words in the
// text into links (a phone number, an email address, "privacy policy").
type TermsSection struct {
	Heading string
	Text    string
	Links   []TermsLink
}

// TermsLink is words in a section's text and where they link. NoWrap keeps
// them on one line (a phone number).
type TermsLink struct {
	Text   string
	Href   string
	NoWrap bool
}

type termsSectionView struct {
	Heading string
	HTML    template.HTML
}

type termsBody struct {
	Title, Version, Intro, ClosingHeading, ClosingText string
	Sections                                           []termsSectionView
	Actions                                            []action
	Footer                                             string
}

// linkify escapes text and links each link's first occurrence in it.
func linkify(text string, links []TermsLink) template.HTML {
	type hit struct {
		at int
		l  TermsLink
	}
	var hits []hit
	for _, l := range links {
		if l.Text == "" {
			continue
		}
		if i := strings.Index(text, l.Text); i >= 0 {
			hits = append(hits, hit{i, l})
		}
	}
	// In order of appearance, without overlaps.
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j].at < hits[j-1].at; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	var b strings.Builder
	pos := 0
	for _, h := range hits {
		if h.at < pos {
			continue
		}
		b.WriteString(html.EscapeString(text[pos:h.at]))
		inner := html.EscapeString(h.l.Text)
		if h.l.NoWrap {
			inner = `<span class="nw">` + inner + `</span>`
		}
		if h.l.Href != "" {
			b.WriteString(`<a href="` + html.EscapeString(h.l.Href) + `">` + inner + `</a>`)
		} else {
			b.WriteString(inner)
		}
		pos = h.at + len(h.l.Text)
	}
	b.WriteString(html.EscapeString(text[pos:]))
	return template.HTML(b.String()) //nolint:gosec // every piece is escaped above
}

// smsTerms is GET /sms-terms.
func (h *Handler) smsTerms(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.TextTerms == nil {
		h.notFound(w, r, s)
		return
	}
	p, ok := h.TextTerms.TextTermsPage(r.Context(), s.ws)
	if !ok {
		h.notFound(w, r, s)
		return
	}
	body := termsBody{Title: p.Title, Version: p.Version, Intro: p.Intro, ClosingHeading: p.ClosingHeading, ClosingText: p.ClosingText, Footer: p.Footer}
	for _, sec := range p.Sections {
		body.Sections = append(body.Sections, termsSectionView{Heading: sec.Heading, HTML: linkify(sec.Text, sec.Links)})
	}
	if p.ManageLabel != "" {
		body.Actions = append(body.Actions, action{Label: p.ManageLabel, Href: s.canon.PreferencesRequest()})
	}
	if p.BackLabel != "" && p.BackURL != "" {
		body.Actions = append(body.Actions, action{Label: p.BackLabel, Href: p.BackURL})
	}
	lang := p.Lang
	if lang == "" {
		lang = s.set.CopyLanguage()
	}
	v := h.view(s, p.Title+" · "+s.set.Branding.PublisherName, body)
	v.Lang, v.Dir, v.labels = lang, pagelabels.Dir(lang), pagelabels.For(lang, nil)
	// The terms are public and meant to be found (carriers check them).
	v.Robots = "index, follow"
	v.Canonical = s.canon.SmsTerms()
	v.Description = p.Intro
	h.render(w, r, s, "smsterms", http.StatusOK, v, csp{}, false)
}
