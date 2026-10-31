package readiness

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
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

// FinalRecipients are an audience's final recipients as a run counts them,
// read with q (the demo gives the Weekly Digest's readers their languages
// inside its transaction).
func FinalRecipients(ctx context.Context, q *store.Queries, wsID uuid.UUID, audience, sunsetPolicy json.RawMessage, now time.Time) ([]Recipient, error) {
	_, out, err := (&Service{}).computeAudience(ctx, q, wsID, ParseAudience(audience), sunsetDays(sunsetPolicy), now, true)
	return out, err
}

// VersionReport is the content check of one language version.
type VersionReport struct {
	Language string
	Report   contentcheck.ContentCheckReport
	// TrackedBytes is the version's HTML as sent, with tracking (SR-7).
	TrackedBytes int
}

// sentBytes is the size Gmail measures: the HTML as sent when it was
// measured, the HTML otherwise.
func sentBytes(r contentcheck.ContentCheckReport, tracked int) int { return max(r.SizeBytes, tracked) }

// langArg is a content language's name for a message: written in each
// reader's language, unless the name given isn't the catalog's.
func langArg(code, name string) any {
	if name == "" || languageIn(i18n.Default, code) == name {
		return ArgLanguage(code)
	}
	return name
}

// langText is a content language's name as a text (a row's label).
func langText(code, name string) Text { return tx("languageName", A{"language": langArg(code, name)}) }

// languageFix applies a fix of the Languages group (audited by the
// languages domain).
func (s *Service) languageFix(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, action, lang, field string) (string, error) {
	if s.Languages == nil || lang == "" {
		return "", api.Invalid(field+".action", say(ctx, "problem.unknownFix", nil))
	}
	l := i18n.LangFrom(ctx)
	name := languageIn(l, lang)
	switch action {
	case FixMarkLanguageUpToDate:
		_, err := s.Languages.MarkCampaignLanguage(ctx, ws, p, id, lang, false, languages.MarkInput{})
		return say(ctx, "applied.markedUpToDate", A{"language": name}), err
	case FixMarkLanguageTranslated:
		_, err := s.Languages.MarkCampaignLanguage(ctx, ws, p, id, lang, true, languages.MarkInput{})
		return say(ctx, "applied.markedTranslated", A{"language": name}), err
	case FixHoldLanguage:
		ls, err := s.Languages.HoldCampaignLanguage(ctx, ws, p, id, lang, languages.HoldInput{Held: true})
		return say(ctx, "applied.held", A{"language": name, "source": languageIn(l, ls.Source)}), err
	default:
		_, err := s.Languages.HoldCampaignLanguage(ctx, ws, p, id, lang, languages.HoldInput{Held: false})
		return say(ctx, "applied.released", A{"language": name}), err
	}
}

// blockPhrase names a block in a sentence: "the intro text", "the “Faster
// CSV exports” card".
func blockPhrase(name string) Text {
	if i := strings.Index(name, " · Heading"); i >= 0 {
		name = name[:i]
	}
	if strings.HasPrefix(name, "Text · ") {
		return tx("block.text", A{"name": strings.TrimPrefix(name, "Text · ")})
	}
	if strings.HasPrefix(name, "Header") {
		return tx("block.header", nil)
	}
	r, n := utf8.DecodeRuneInString(name)
	return tx("block.other", A{"name": string(unicode.ToLower(r)) + name[n:]})
}

// languages adds the Languages group.
func (b *builder) languages() {
	lf := b.f.Languages
	if !lf.Applies || len(lf.Languages) == 0 {
		return
	}
	src := ArgLanguage(lf.Source)
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
		name := langArg(l.Language, l.Name)
		gets := ArgLanguage(l.Gets)
		label := langText(l.Language, l.Name)
		switch {
		case l.Held:
			b.add(check{Key: "languages.held." + l.Language, Group: GroupLanguages, Status: Pass, Label: label,
				Title:  tx("languages.held", A{"language": name, "gets": gets}),
				Detail: tx("languages.heldDetail", A{"language": name}),
				Fixes:  []fix{apply(FixReleaseLanguage, tx("fix.sendLanguageAgain", A{"language": name}), "", params(l.Language))}})
		case l.Status == languages.StatusOutdated:
			title := tx("languages.outdated", A{"language": name, "count": l.Attention, "source": src})
			var parts []Text
			if len(l.Changed) > 0 {
				phrases := make([]any, 0, len(l.Changed))
				for _, c := range l.Changed {
					phrases = append(phrases, blockPhrase(c))
				}
				parts = append(parts, tx("languages.changed", A{"blocks": argCapital(ArgAnd(phrases)), "version": lf.SourceVersion}))
			}
			if l.SubjectChanged {
				parts = append(parts, tx("languages.subjectChanged", nil))
			}
			parts = append(parts, tx("languages.getsTranslation", A{"language": name, "version": l.TranslatedFrom}))
			b.add(check{Key: "languages.outdated." + l.Language, Group: GroupLanguages, Status: Warn, Title: title, Detail: sentences(parts...), Label: label,
				Ack: tx("languages.outdatedAck", A{"language": name}),
				Fixes: []fix{
					link(tx("fix.compare", nil), editor(l.Language, "compare")),
					apply(FixMarkLanguageUpToDate, tx("fix.markUpToDate", nil), "", params(l.Language)),
					apply(FixHoldLanguage, tx("fix.sendSourceTo", A{"source": src, "language": name}), "", params(l.Language)),
				}})
		case l.Status == languages.StatusDraft:
			b.add(check{Key: "languages.draft." + l.Language, Group: GroupLanguages, Status: Warn, Label: label,
				Title:  tx("languages.draft", A{"count": l.Readers, "language": name, "gets": gets}),
				Detail: tx("languages.draftDetail", A{"language": name, "translated": l.Translated, "blocks": l.Blocks}),
				Ack:    tx("languages.draftAck", A{"language": name, "gets": gets}),
				Fixes: []fix{
					link(tx("fix.openLanguage", A{"language": name}), editor(l.Language, "")),
					apply(FixMarkLanguageTranslated, tx("fix.markTranslated", nil), "", params(l.Language)),
				}})
		}
	}
	// Recipients per language.
	rows := []row{}
	ready := []any{}
	for _, l := range lf.Languages {
		if !l.Offered {
			continue
		}
		name := langArg(l.Language, l.Name)
		gets := ArgLanguage(l.Gets)
		r := row{Label: langText(l.Language, l.Name), Count: l.Readers, Language: l.Language, Gets: l.Gets, GetsName: langText(l.Gets, "")}
		var detail Text
		switch {
		case l.Held:
			detail = tx("languages.rowHeld", A{"gets": gets})
		case l.Status == languages.StatusMissing:
			detail = tx("languages.rowMissing", A{"gets": gets, "language": name})
		case l.Status == languages.StatusDraft:
			detail = tx("languages.rowDraft", A{"gets": gets, "language": name})
		case l.Status == languages.StatusOutdated:
			detail = tx("languages.rowOutdated", A{"gets": gets})
			ready = append(ready, name)
		default:
			detail = tx("languages.rowGets", A{"gets": gets})
			ready = append(ready, name)
		}
		if l.Language == lf.OthersGet && lf.NoLanguage+lf.NotOffered > 0 {
			// Readers with no language, or one the workspace doesn't offer,
			// get this version too.
			r.Count += lf.NoLanguage + lf.NotOffered
			parts := []Text{tx("languages.readersOf", A{"count": l.Readers, "language": name})}
			if lf.NoLanguage > 0 {
				parts = append(parts, tx("languages.noLanguage", A{"count": lf.NoLanguage}))
			}
			if lf.NotOffered > 0 {
				parts = append(parts, tx("languages.otherLanguages", A{"count": lf.NotOffered}))
			}
			detail = tx("languages.rowWithOthers", A{"detail": detail, "parts": argJoin(", ", parts)})
		}
		r.Detail = detail
		if r.Count == 0 && l.Language != lf.Source {
			continue
		}
		rows = append(rows, r)
	}
	b.add(check{Key: "languages.recipients", Group: GroupLanguages, Status: Pass, Title: tx("languages.recipients", nil), Label: tx("languages.recipients", nil),
		Rows: rows, Short: tx("languages.upToDate", A{"languages": ArgAnd(ready)})})
	// Public pages in the fallback.
	for _, l := range lf.Languages {
		if !l.Offered || l.Readers == 0 || l.PageCopy == "live" || l.Language == lf.Source {
			continue
		}
		name := langArg(l.Language, l.Name)
		detail := tx("languages.pagesNotSetUp", A{"language": name})
		if l.PageCopy == "incomplete" {
			detail = tx("languages.pagesIncomplete", A{"language": name})
		}
		b.add(check{Key: "languages.pages." + l.Language, Group: GroupLanguages, Status: Pass, Label: tx("label.languagePages", A{"language": name}),
			Title: tx("languages.pages", A{"language": name, "fallback": ArgLanguage(lf.Fallback)}), Detail: detail,
			Fixes: []fix{link(tx("fix.writePageCopy", A{"language": name}), b.href("/forms?language="+l.Language))}})
	}
}

// languagesCaption is "3 versions · source English · Vietnamese has none".
func (b *builder) languagesCaption() Text {
	lf := b.f.Languages
	n := 1 + len(lf.Versions)
	none := []any{}
	for _, l := range lf.Languages {
		if l.Offered && l.Status == languages.StatusMissing {
			none = append(none, langArg(l.Language, l.Name))
		}
	}
	return tx("languages.caption", A{"count": n, "source": ArgLanguage(lf.Source), "none": len(none), "missing": ArgAnd(none)})
}

// versionCodes are the languages of the versions that are sent, the
// source first.
func (b *builder) versionCodes() []any {
	out := []any{ArgLanguage(b.f.Languages.Source)}
	for _, v := range b.f.Content.Versions {
		out = append(out, ArgLanguage(v.Language))
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
	src := ArgLanguage(b.f.Languages.Source)
	names := b.versionCodes()
	all := append([]VersionReport{{Language: b.f.Languages.Source, Report: b.f.Content.Report}}, b.f.Content.Versions...)
	for i := range b.checks {
		c := &b.checks[i]
		if c.Status != Pass {
			continue
		}
		switch c.Key {
		case "content.spam":
			c.Detail = tx("languages.spamChecked", A{"source": src, "others": ArgAnd(names[1:])})
			if languageIn(i18n.Default, b.f.Languages.Source) != "English" {
				c.Detail = tx("languages.spamCheckedAll", A{"languages": ArgAnd(names)})
			}
		case "content.image_ratio":
			parts := []Text{}
			for _, v := range all {
				parts = append(parts, tx("languages.ratioPart", A{"language": ArgLanguage(v.Language), "share": ArgPercent(math.Round(v.Report.ImageShare * 100))}))
			}
			c.Title = tx("languages.ratioEvery", A{"limit": ArgPercent(contentcheck.ImageShareLimit * 100)})
			c.Detail = tx("languages.ratioEveryDetail", A{"parts": argJoin(" · ", parts), "images": b.f.Content.Report.Images})
		case "content.size":
			parts := []Text{}
			all[0].TrackedBytes = b.f.Content.TrackedBytes
			for _, v := range all {
				parts = append(parts, tx("languages.sizePart", A{"language": ArgLanguage(v.Language), "kb": int(math.Round(float64(sentBytes(v.Report, v.TrackedBytes)) / 1024))}))
			}
			c.Title = tx("languages.sizeEvery", A{"clip": 102, "tracked": yes(b.f.Content.TrackedBytes > 0)})
			c.Detail = tx("languages.sizeEveryDetail", A{"parts": argJoin(" · ", parts), "clip": 102})
		case "content.plain_text":
			c.Title = tx("languages.plainEvery", nil)
			c.Detail = tx("languages.plainEveryDetail", A{"languages": ArgAnd(names)})
		case "links.respond":
			if c.every {
				c.Title = tx("links.inEveryVersion", A{"title": c.Title})
			}
		case "compliance.list_unsubscribe":
			c.Detail = tx("compliance.oneClickEvery", nil)
		case "compliance.footer":
			c.Title = tx("compliance.footerPassEvery", nil)
		}
	}
	// A version that fails where the source passes gets its own row.
	for _, v := range b.f.Content.Versions {
		name := ArgLanguage(v.Language)
		open := link(tx("fix.openLanguage", A{"language": name}), b.href("/campaigns/"+b.f.Campaign.ID+"/edit?language="+v.Language))
		r := v.Report
		src := b.f.Content.Report
		if len(r.SubjectHeuristics) > 0 && len(src.SubjectHeuristics) == 0 {
			b.add(check{Key: "content.spam." + v.Language, Group: GroupContent, Status: Warn, Label: tx("label.spamTriggersIn", A{"language": name}),
				Title:  tx("languages.versionSpam", A{"language": name, "count": len(r.SubjectHeuristics)}),
				Detail: heuristicsText(r.SubjectHeuristics), Ack: tx("languages.versionSpamAck", A{"language": name}),
				Fixes: []fix{open}})
		} else if places := spamPlaces(r, false); len(places) > 0 && len(spamPlaces(src, true)) == 0 {
			// The version's preheader or body trips heuristics the source's
			// doesn't (SR-7).
			b.add(check{Key: "content.spam." + v.Language, Group: GroupContent, Status: Warn, Label: tx("label.spamTriggersIn", A{"language": name}),
				Title:  tx("languages.versionSpamAny", A{"language": name}),
				Detail: joined(" · ", places), Ack: tx("languages.versionSpamAnyAck", A{"language": name}),
				Fixes: []fix{open}})
		}
		if sent := sentBytes(r, v.TrackedBytes); sent >= contentcheck.NearClipBytes && sentBytes(b.f.Content.Report, b.f.Content.TrackedBytes) < contentcheck.NearClipBytes {
			kb := int(math.Round(float64(sent) / 1024))
			b.add(check{Key: "content.size." + v.Language, Group: GroupContent, Status: Warn, Label: tx("label.htmlSizeIn", A{"language": name}),
				Title:  tx("languages.versionSize", A{"language": name, "kb": kb, "clip": 102}),
				Detail: tx("languages.versionSizeDetail", nil), Ack: tx("languages.versionSizeAck", A{"language": name, "kb": kb}),
				Fixes: []fix{open}})
		}
		if r.ImageShare >= contentcheck.ImageShareLimit && b.f.Content.Report.ImageShare < contentcheck.ImageShareLimit {
			b.add(check{Key: "content.image_ratio." + v.Language, Group: GroupContent, Status: Warn, Label: tx("label.imageRatioIn", A{"language": name}),
				Title: tx("languages.versionRatio", A{"language": name, "share": ArgPercent(math.Round(r.ImageShare * 100)), "limit": ArgPercent(contentcheck.ImageShareLimit * 100)}),
				Ack:   tx("languages.versionRatioAck", A{"language": name}),
				Fixes: []fix{open}})
		}
		for _, is := range r.Issues {
			if is.ID != "compliance.footer" {
				continue
			}
			if len(issuesOf(b.f.Content.Report, "compliance.footer")) > 0 {
				continue
			}
			b.add(check{Key: "compliance.footer." + v.Language, Group: GroupCompliance, Status: Block, Label: tx("label.footerIn", A{"language": name}),
				Title:  tx("languages.versionFooter", A{"language": name}),
				Detail: tx("languages.versionFooterDetail", nil),
				Fixes:  []fix{open}})
		}
	}
}
