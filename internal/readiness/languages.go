package readiness

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// Content languages (design expansion M17): the Languages group, and the
// content, link and compliance checks run on every language version.

// GroupLanguages is the Languages group: who gets which version.
const GroupLanguages = "languages"

// Fix actions of the Languages group.
const (
	FixMarkLanguageUpToDate   = "mark_language_up_to_date"
	FixMarkLanguageTranslated = "mark_language_translated"
	FixHoldLanguage           = "hold_language"
	FixReleaseLanguage        = "release_language"
)

// VersionReport is the content check of one language version.
type VersionReport struct {
	Language string
	Report   contentcheck.ContentCheckReport
}

// languageName is a language's English name.
func languageName(code string) string { return languages.NameOf(code) }

// languageFix applies a fix of the Languages group (audited by the
// languages domain).
func (s *Service) languageFix(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, action, lang, field string) (string, error) {
	if s.Languages == nil || lang == "" {
		return "", api.Invalid(field+".action", "Unknown fix.")
	}
	name := languageName(lang)
	switch action {
	case FixMarkLanguageUpToDate:
		_, err := s.Languages.MarkCampaignLanguage(ctx, ws, p, id, lang, false, languages.MarkInput{})
		return fmt.Sprintf("Marked %s as up to date", name), err
	case FixMarkLanguageTranslated:
		_, err := s.Languages.MarkCampaignLanguage(ctx, ws, p, id, lang, true, languages.MarkInput{})
		return fmt.Sprintf("Marked %s as translated", name), err
	case FixHoldLanguage:
		ls, err := s.Languages.HoldCampaignLanguage(ctx, ws, p, id, lang, languages.HoldInput{Held: true})
		return fmt.Sprintf("%s readers get %s this time", name, languageName(ls.Source)), err
	default:
		_, err := s.Languages.HoldCampaignLanguage(ctx, ws, p, id, lang, languages.HoldInput{Held: false})
		return fmt.Sprintf("%s readers get %s again", name, name), err
	}
}

// blockPhrase names a block in a sentence: "the intro", "the “Faster CSV
// exports” card".
func blockPhrase(name string) string {
	if i := strings.Index(name, " · Heading"); i >= 0 {
		name = name[:i]
	}
	if strings.HasPrefix(name, "Text · ") {
		return "the " + strings.TrimPrefix(name, "Text · ") + " text"
	}
	if strings.HasPrefix(name, "Header") {
		return "the header"
	}
	return "the " + strings.ToLower(name[:1]) + name[1:]
}

// languages adds the Languages group.
func (b *builder) languages() {
	lf := b.f.Languages
	if !lf.Applies || len(lf.Languages) == 0 {
		return
	}
	src := languageName(lf.Source)
	editor := func(lang, view string) string {
		p := "/campaigns/" + b.f.Campaign.ID + "/edit?language=" + lang
		if view != "" {
			p += "&view=" + view
		}
		return b.href(p)
	}
	params := func(lang string) map[string]string { return map[string]string{"language": lang} }
	for _, l := range lf.Languages {
		if !l.Offered || l.Language == lf.Source {
			continue
		}
		name := l.Name
		switch {
		case l.Held:
			b.add(ReadinessCheck{Key: "languages.held." + l.Language, Group: GroupLanguages, Status: Pass, Label: name,
				Title:  fmt.Sprintf("%s readers get %s, as chosen", name, languageName(l.Gets)),
				Detail: fmt.Sprintf("The %s version is kept but not sent this time.", name),
				Fixes:  []ReadinessFix{apply(FixReleaseLanguage, fmt.Sprintf("Send %s again", name), "", params(l.Language))}})
		case l.Status == languages.StatusOutdated:
			title := fmt.Sprintf("%s is outdated: %d %s changed in %s after translation", name, l.Attention, Plural(int64(l.Attention), "block", "blocks"), src)
			phrases := []string{}
			for _, c := range l.Changed {
				phrases = append(phrases, blockPhrase(c))
			}
			detail := ""
			if len(phrases) > 0 {
				detail = capitalize(JoinAnd(phrases)) + fmt.Sprintf(" changed in v%d. ", lf.SourceVersion)
			}
			if l.SubjectChanged {
				detail += "The subject changed too. "
			}
			detail += fmt.Sprintf("%s readers get the v%d translation as it is.", name, l.TranslatedFrom)
			b.add(ReadinessCheck{Key: "languages.outdated." + l.Language, Group: GroupLanguages, Status: Warn, Title: title, Detail: detail, Label: name,
				Ack: fmt.Sprintf("I understand the %s version is outdated.", name),
				Fixes: []ReadinessFix{
					link("Compare", editor(l.Language, "compare")),
					apply(FixMarkLanguageUpToDate, "Mark as up to date", "", params(l.Language)),
					apply(FixHoldLanguage, fmt.Sprintf("Send %s to %s readers", src, name), "", params(l.Language)),
				}})
		case l.Status == languages.StatusDraft:
			title := fmt.Sprintf("%s %s readers get %s unless %s is marked translated", Count(l.Readers), name, languageName(l.Gets), name)
			b.add(ReadinessCheck{Key: "languages.draft." + l.Language, Group: GroupLanguages, Status: Warn, Title: title, Label: name,
				Detail: fmt.Sprintf("The %s version is a draft: %d of %d blocks translated. Drafts are never sent.", name, l.Translated, l.Blocks),
				Ack:    fmt.Sprintf("I understand %s readers get %s.", name, languageName(l.Gets)),
				Fixes: []ReadinessFix{
					link("Open "+name, editor(l.Language, "")),
					apply(FixMarkLanguageTranslated, "Mark as translated", "", params(l.Language)),
				}})
		}
	}
	// Recipients per language.
	rows := []ReadinessCheckRow{}
	ready := []string{}
	for _, l := range lf.Languages {
		if !l.Offered {
			continue
		}
		gets := languageName(l.Gets)
		row := ReadinessCheckRow{Label: l.Name, Count: l.Readers}
		switch {
		case l.Held:
			row.Detail = fmt.Sprintf("gets %s, as chosen", gets)
		case l.Status == languages.StatusMissing:
			row.Detail = fmt.Sprintf("gets %s: there’s no %s version", gets, l.Name)
		case l.Status == languages.StatusDraft:
			row.Detail = fmt.Sprintf("gets %s: the %s version is a draft", gets, l.Name)
		case l.Status == languages.StatusOutdated:
			row.Detail = fmt.Sprintf("gets %s, the outdated version", gets)
			ready = append(ready, l.Name)
		default:
			row.Detail = "gets " + gets
			ready = append(ready, l.Name)
		}
		if l.Language == lf.OthersGet && lf.NoLanguage+lf.NotOffered > 0 {
			// Readers with no language, or one the workspace doesn't offer,
			// get this version too.
			row.Count += lf.NoLanguage + lf.NotOffered
			parts := []string{fmt.Sprintf("%s %s", Count(l.Readers), l.Name)}
			if lf.NoLanguage > 0 {
				parts = append(parts, fmt.Sprintf("%s with no language", Count(lf.NoLanguage)))
			}
			if lf.NotOffered > 0 {
				parts = append(parts, fmt.Sprintf("%s with other languages", Count(lf.NotOffered)))
			}
			row.Detail += " · " + strings.Join(parts, ", ")
		}
		if row.Count == 0 && l.Language != lf.Source {
			continue
		}
		rows = append(rows, row)
	}
	short := JoinAnd(ready) + " up to date"
	b.add(ReadinessCheck{Key: "languages.recipients", Group: GroupLanguages, Status: Pass, Title: "Recipients per language", Label: "Recipients per language",
		Rows: rows, Short: short})
	// Public pages in the fallback.
	for _, l := range lf.Languages {
		if !l.Offered || l.Readers == 0 || l.PageCopy == "live" || l.Language == lf.Source {
			continue
		}
		fb := languageName(lf.Fallback)
		detail := fmt.Sprintf("%s page copy isn’t set up.", l.Name)
		if l.PageCopy == "incomplete" {
			detail = fmt.Sprintf("%s page copy is incomplete.", l.Name)
		}
		b.add(ReadinessCheck{Key: "languages.pages." + l.Language, Group: GroupLanguages, Status: Pass, Label: l.Name + " pages",
			Title: fmt.Sprintf("%s readers see the preference center and unsubscribe page in %s", l.Name, fb), Detail: detail,
			Fixes: []ReadinessFix{link(fmt.Sprintf("Write %s page copy", l.Name), b.href("/forms?language="+l.Language))}})
	}
}

// languagesCaption is "3 versions · source English · Vietnamese has none".
func (b *builder) languagesCaption() string {
	lf := b.f.Languages
	n := 1 + len(lf.Versions)
	none := []string{}
	for _, l := range lf.Languages {
		if l.Offered && l.Status == languages.StatusMissing {
			none = append(none, l.Name)
		}
	}
	s := fmt.Sprintf("%d %s · source %s", n, Plural(int64(n), "version", "versions"), languageName(lf.Source))
	switch len(none) {
	case 0:
	case 1:
		s += " · " + none[0] + " has none"
	default:
		s += " · " + JoinAnd(none) + " have none"
	}
	return s
}

// versionNames names the versions that are sent: "English, Spanish and German".
func (b *builder) versionNames() []string {
	out := []string{languageName(b.f.Languages.Source)}
	for _, v := range b.f.Content.Versions {
		out = append(out, languageName(v.Language))
	}
	return out
}

// everyVersion rewrites content, link and compliance checks for an email
// sent in several languages: passing results name each version, and a
// version that fails on its own gets its own row.
func (b *builder) everyVersion() {
	if len(b.f.Content.Versions) == 0 || !b.f.Content.Checked {
		return
	}
	src := languageName(b.f.Languages.Source)
	names := b.versionNames()
	all := append([]VersionReport{{Language: b.f.Languages.Source, Report: b.f.Content.Report}}, b.f.Content.Versions...)
	for i := range b.checks {
		c := &b.checks[i]
		switch c.Key {
		case "content.spam":
			if c.Status == Pass {
				others := names[1:]
				c.Detail = fmt.Sprintf("%s checked for English phrases; %s for capitals, punctuation and length", src, JoinAnd(others))
				if src != "English" {
					c.Detail = fmt.Sprintf("%s checked for capitals, punctuation and length", JoinAnd(names))
				}
			}
		case "content.image_ratio":
			if c.Status == Pass {
				parts := []string{}
				for _, v := range all {
					parts = append(parts, fmt.Sprintf("%s %d%%", languageName(v.Language), int(math.Round(v.Report.ImageShare*100))))
				}
				c.Title = "Image-to-text ratio under 60% in every version"
				c.Detail = strings.Join(parts, " · ") + fmt.Sprintf(" · %d %s", b.f.Content.Report.Images, Plural(int64(b.f.Content.Report.Images), "image", "images"))
			}
		case "content.size":
			if c.Status == Pass {
				parts := []string{}
				for _, v := range all {
					parts = append(parts, fmt.Sprintf("%s %d KB", languageName(v.Language), int(math.Round(float64(v.Report.SizeBytes)/1024))))
				}
				c.Title = "HTML size under Gmail’s 102 KB clipping limit"
				c.Detail = strings.Join(parts, " · ") + ", under 102 KB"
			}
		case "content.plain_text":
			if c.Status == Pass {
				c.Title = "Plain-text version generated for each language"
				c.Detail = "multipart/alternative · " + JoinAnd(names)
			}
		case "links.respond":
			if c.Status == Pass && strings.HasPrefix(c.Title, "All ") {
				c.Title += " in every version"
			}
		case "compliance.list_unsubscribe":
			if c.Status == Pass {
				c.Detail = "List-Unsubscribe-Post: List-Unsubscribe=One-Click · in every version"
			}
		case "compliance.footer":
			if c.Status == Pass {
				c.Title = "Footer has an unsubscribe link and a postal address in every version"
			}
		}
	}
	// A version that fails where the source passes gets its own row.
	for _, v := range b.f.Content.Versions {
		name := languageName(v.Language)
		r := v.Report
		if len(r.SubjectHeuristics) > 0 && len(b.f.Content.Report.SubjectHeuristics) == 0 {
			words := make([]string, 0, len(r.SubjectHeuristics))
			for _, h := range r.SubjectHeuristics {
				words = append(words, heuristicWord(h))
			}
			b.add(ReadinessCheck{Key: "content.spam." + v.Language, Group: GroupContent, Status: Warn, Label: "Spam triggers · " + name,
				Title:  fmt.Sprintf("The %s subject trips %d spam %s", name, len(r.SubjectHeuristics), Plural(int64(len(r.SubjectHeuristics)), "heuristic", "heuristics")),
				Detail: capitalize(JoinAnd(words)) + ".", Ack: fmt.Sprintf("I understand the %s subject trips spam heuristics.", name),
				Fixes: []ReadinessFix{link("Open "+name, b.href("/campaigns/"+b.f.Campaign.ID+"/edit?language="+v.Language))}})
		}
		if r.SizeBytes >= contentcheck.NearClipBytes && b.f.Content.Report.SizeBytes < contentcheck.NearClipBytes {
			kb := int(math.Round(float64(r.SizeBytes) / 1024))
			b.add(ReadinessCheck{Key: "content.size." + v.Language, Group: GroupContent, Status: Warn, Label: "HTML size · " + name,
				Title:  fmt.Sprintf("The %s version is %d KB, close to Gmail’s 102 KB clipping limit", name, kb),
				Detail: "Gmail clips the message and hides the footer, including the unsubscribe link.", Ack: fmt.Sprintf("I understand the %s version is %d KB.", name, kb),
				Fixes: []ReadinessFix{link("Open "+name, b.href("/campaigns/"+b.f.Campaign.ID+"/edit?language="+v.Language))}})
		}
		if r.ImageShare >= 0.6 && b.f.Content.Report.ImageShare < 0.6 {
			b.add(ReadinessCheck{Key: "content.image_ratio." + v.Language, Group: GroupContent, Status: Warn, Label: "Image-to-text ratio · " + name,
				Title: fmt.Sprintf("The %s version is %d%% image, over the recommended 60%%", name, int(math.Round(r.ImageShare*100))),
				Ack:   fmt.Sprintf("I understand the %s version is image-heavy.", name),
				Fixes: []ReadinessFix{link("Open "+name, b.href("/campaigns/"+b.f.Campaign.ID+"/edit?language="+v.Language))}})
		}
		for _, is := range r.Issues {
			if is.ID != "compliance.footer" {
				continue
			}
			if len(issuesOf(b.f.Content.Report, "compliance.footer")) > 0 {
				continue
			}
			b.add(ReadinessCheck{Key: "compliance.footer." + v.Language, Group: GroupCompliance, Status: Block, Label: "Footer · " + name,
				Title:  fmt.Sprintf("The %s version has no unsubscribe link or postal address", name),
				Detail: "Every version needs the footer: Gmail and Yahoo reject bulk mail without a one-click unsubscribe, and the law asks for an address.",
				Fixes:  []ReadinessFix{link("Open "+name, b.href("/campaigns/"+b.f.Campaign.ID+"/edit?language="+v.Language))}})
		}
	}
}
