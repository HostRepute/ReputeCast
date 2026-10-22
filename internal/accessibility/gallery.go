package accessibility

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// AccessibilityTemplateResult is a gallery tile's result.
type AccessibilityTemplateResult struct {
	// ID is the template's ("starter:weekly-digest" or a saved template's id)
	// or, for the last sent campaign, the campaign's.
	ID      string `json:"id"`
	Issues  int    `json:"issues"`
	Serious int    `json:"serious"`
	// Summary explains the issues on the tile: "Serious: the linked logo
	// has no alt text, so its link has no text. Also two “Read more” links."
	Summary string `json:"summary,omitempty"`
}

// AccessibilityGallery is every gallery template's result.
type AccessibilityGallery struct {
	// Language is the language the templates are checked in.
	Language     string                        `json:"language"`
	Templates    []AccessibilityTemplateResult `json:"templates"`
	LastCampaign *AccessibilityTemplateResult  `json:"lastCampaign,omitempty"`
}

// Gallery checks the starters, the saved templates and the last sent
// campaign, in the workspace's default language.
func (s *Service) Gallery(ctx context.Context, ws *store.Workspace, starterIDs []string) (AccessibilityGallery, error) {
	lang := s.language(ctx, ws)
	out := AccessibilityGallery{Language: lang, Templates: []AccessibilityTemplateResult{}}
	_, min, err := s.Settings(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	check := func(id, mjmlSrc, htmlSrc string) (AccessibilityTemplateResult, error) {
		h := sha256.Sum256([]byte(mjmlSrc + "\x00" + htmlSrc))
		key := fmt.Sprintf("%s:gallery:%s:%d:%s", ws.ID, lang, min, hex.EncodeToString(h[:12]))
		r, ok := s.cached(key)
		if !ok {
			var err error
			if r, err = s.run(ctx, ws.ID, []Version{{Lang: lang, MJML: mjmlSrc, HTML: htmlSrc, Source: true}}); err != nil {
				return AccessibilityTemplateResult{ID: id}, err
			}
			s.remember(key, r)
		}
		return AccessibilityTemplateResult{ID: id, Issues: len(r.Issues), Serious: r.Serious, Summary: GallerySummaryIn(i18n.LangFrom(ctx), r.Issues)}, nil
	}
	address := templates.WorkspaceAddress(ws)
	for _, id := range starterIDs {
		key, ok := strings.CutPrefix(id, templates.StarterPrefix)
		if !ok {
			continue
		}
		src, ok := templates.StarterMJML(key, address)
		if !ok {
			continue
		}
		if res, err := check(id, src, ""); err == nil {
			out.Templates = append(out.Templates, res)
		}
	}
	saved, err := s.Q.AccessibilityTemplates(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, t := range saved {
		if res, err := check(t.ID.String(), t.Mjml, t.Html); err == nil {
			out.Templates = append(out.Templates, res)
		}
	}
	last, err := s.Q.ComposerLastSentCampaign(ctx, ws.ID)
	switch {
	case err == nil:
		if res, err := check(last.ID.String(), last.Mjml, ""); err == nil {
			out.LastCampaign = &res
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return out, err
	}
	return out, nil
}

// GallerySummary explains a template's issues in a sentence or two: the
// serious ones first, then the rest.
func GallerySummary(issues []AccessibilityIssue) string {
	return GallerySummaryIn(i18n.Default, issues)
}

// GallerySummaryIn is GallerySummary in a reader's language.
func GallerySummaryIn(l i18n.Lang, issues []AccessibilityIssue) string {
	if len(issues) == 0 {
		return ""
	}
	var serious []string
	for _, is := range issues {
		if is.Severity == Serious {
			title := is.Title
			if s := titleIn(l, is.Texts.Title); s != "" {
				title = s
			}
			serious = append(serious, lowerFirst(strings.TrimSuffix(title, ".")))
		}
	}
	moderate := moderateSummary(l, issues)
	switch {
	case len(serious) > 0 && moderate != "":
		return mGallerySeriousAlso.Text(l, i18n.Args{"list": strings.Join(serious, "; "), "moderate": moderate})
	case len(serious) > 0:
		return mGallerySerious.Text(l, i18n.Args{"list": strings.Join(serious, "; ")})
	}
	return capitalize(mGalleryModerate.Text(l, i18n.Args{"moderate": moderate}))
}

// moderateSummary is "two “Read more” links, a bare URL and 11 px text".
func moderateSummary(l i18n.Lang, issues []AccessibilityIssue) string {
	type group struct {
		m    i18n.Msg
		args i18n.Args
		n    int
	}
	var order []string
	groups := map[string]*group{}
	add := func(key string, m i18n.Msg, args i18n.Args, n int) {
		g := groups[key]
		if g == nil {
			g = &group{m: m, args: args}
			groups[key] = g
			order = append(order, key)
		}
		g.n += max(n, 1)
	}
	for _, is := range issues {
		if is.Severity != Moderate {
			continue
		}
		switch is.Kind {
		case KindLinkText:
			if strings.HasPrefix(is.Title, "A bare URL") {
				add("bare", mGalleryBare, i18n.Args{}, is.Element.Count)
			} else {
				add("vague:"+strings.ToLower(is.Element.Label), mGalleryVague, i18n.Args{"text": is.Element.Label}, is.Element.Count)
			}
		case KindHeading:
			add("heading", mGalleryHeading, i18n.Args{}, 1)
		case KindTextSize:
			add("size:"+trimFloat(is.Fix.FromPx), mGallerySize, i18n.Args{"px": is.Fix.FromPx}, 1)
		}
	}
	parts := make([]string, 0, len(order))
	for _, k := range order {
		g := groups[k]
		args := i18n.Args{"count": g.n}
		for name, v := range g.args {
			args[name] = v
		}
		parts = append(parts, g.m.Text(l, args))
	}
	return i18n.AndList(l.Locale, parts)
}

// lowerFirst puts a sentence's first word in lower case, inside another
// sentence; a word written in capitals (SES) keeps them.
func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) > 1 && unicode.IsUpper(r[0]) && unicode.IsLower(r[1]) {
		r[0] = unicode.ToLower(r[0])
	}
	return string(r)
}
