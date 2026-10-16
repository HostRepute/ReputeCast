package accessibility

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

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
		return AccessibilityTemplateResult{ID: id, Issues: len(r.Issues), Serious: r.Serious, Summary: GallerySummary(r.Issues)}, nil
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

var numberWords = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}

func numberWord(n int) string {
	if n >= 0 && n < len(numberWords) {
		return numberWords[n]
	}
	return count(n)
}

// GallerySummary explains a template's issues in a sentence or two: the
// serious ones first, then the rest.
func GallerySummary(issues []AccessibilityIssue) string {
	if len(issues) == 0 {
		return ""
	}
	var serious []string
	for _, is := range issues {
		if is.Severity == Serious {
			serious = append(serious, lowerFirst(strings.TrimSuffix(is.Title, ".")))
		}
	}
	moderate := moderateSummary(issues)
	switch {
	case len(serious) > 0 && moderate != "":
		return "Serious: " + strings.Join(serious, "; ") + ". Also " + moderate + "."
	case len(serious) > 0:
		return "Serious: " + strings.Join(serious, "; ") + "."
	}
	return capitalize(moderate) + "."
}

// moderateSummary is "two “Read more” links, a bare URL and 11 px text".
func moderateSummary(issues []AccessibilityIssue) string {
	type group struct {
		phrase string
		n      int
	}
	var order []string
	groups := map[string]*group{}
	add := func(key, phrase string, n int) {
		g := groups[key]
		if g == nil {
			g = &group{phrase: phrase}
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
				add("bare", "bare URL", is.Element.Count)
			} else {
				add("vague:"+strings.ToLower(is.Element.Label), quote(is.Element.Label)+" link", is.Element.Count)
			}
		case KindHeading:
			add("heading", "skipped heading level", 1)
		case KindTextSize:
			add("size:"+trimFloat(is.Fix.FromPx), trimFloat(is.Fix.FromPx)+" px text", 1)
		}
	}
	var parts []string
	for _, k := range order {
		g := groups[k]
		switch {
		case strings.HasPrefix(k, "size:"):
			parts = append(parts, g.phrase)
		case g.n == 1:
			parts = append(parts, "a "+g.phrase)
		default:
			parts = append(parts, numberWord(g.n)+" "+g.phrase+"s")
		}
	}
	return joinAnd(parts)
}

func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) > 1 && r[0] >= 'A' && r[0] <= 'Z' && r[1] >= 'a' && r[1] <= 'z' {
		r[0] += 'a' - 'A'
	}
	return string(r)
}
