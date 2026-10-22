package readiness

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Accessibility group (design expansion, X06): one row per serious
// issue, the moderate issues together in one row, the additions made when
// the email is compiled as passing rows.
//
// The accessibility checks write their findings in English (API clients,
// MCP agents); Send Readiness words them again from the issues' fields, in
// each reader's language, and keeps a finding's own English when its
// wording can't be matched.

// GroupAccessibility is the Accessibility group's key.
const GroupAccessibility = "accessibility"

// Fix actions of the Accessibility group.
const (
	FixAccessibilityDecorative = accessibility.ActionDecorative
	FixAccessibilityColor      = accessibility.ActionColor
	FixAccessibilityOneClick   = accessibility.ActionOneClick
)

// AccessibilityChecker runs the accessibility checks for a run and applies
// the group's one-click fixes.
type AccessibilityChecker interface {
	ReadinessFacts(ctx context.Context, ws *store.Workspace, c accessibility.CampaignContent) accessibility.ReadinessFacts
	FixMJML(ctx context.Context, ws *store.Workspace, src, action string, params map[string]string) (string, accessibility.FixChange, error)
	RecordFix(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, campaign, contentHash string, languages []string, changes []accessibility.FixChange) error
}

// fixLanguageVersions applies Review & send's accessibility fixes, which
// need no words, to every language version as the source got them. A
// version whose layout differs where a fix points keeps that issue; its own
// check shows it. It returns the languages it changed.
func (s *Service) fixLanguageVersions(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, fixes []ReadinessFixRequest, label string) ([]string, error) {
	changed := []string{}
	if s.Campaigns == nil || s.Campaigns.Translations == nil || s.Accessibility == nil || len(fixes) == 0 {
		return changed, nil
	}
	rows, err := s.Q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: accessibility.OwnerCampaign, OwnerID: id})
	if err != nil {
		return changed, err
	}
	for _, r := range rows {
		if strings.TrimSpace(r.Mjml) == "" {
			continue
		}
		d, err := s.Campaigns.DetailIn(ctx, ws, p, id, r.Language)
		if err != nil {
			return changed, err
		}
		src := d.MJML
		for _, fx := range fixes {
			if next, _, err := s.Accessibility.FixMJML(ctx, ws, src, fx.Action, fx.Params); err == nil {
				src = next
			}
		}
		if src == d.MJML {
			continue
		}
		if _, err := s.Campaigns.SaveIn(ctx, ws, p, id, r.Language, campaigns.ComposerSaveContent{BaseVersion: d.Version, Subject: d.Subject, Preheader: d.Preheader,
			MJML: src, ABTest: d.ABTest, Label: label, Source: "manual"}); err != nil {
			return changed, err
		}
		changed = append(changed, r.Language)
	}
	return changed, nil
}

// languageList is content languages' names, joined with commas: "English,
// Spanish, German".
func languageList(codes []string) map[string]any {
	names := make([]any, 0, len(codes))
	for _, c := range codes {
		names = append(names, ArgLanguage(c))
	}
	return argJoin(", ", names)
}

// languagesAnd is content languages' names as a list: "English, Spanish
// and German".
func languagesAnd(codes []string) map[string]any {
	names := make([]any, 0, len(codes))
	for _, c := range codes {
		names = append(names, ArgLanguage(c))
	}
	return ArgAnd(names)
}

// languageNamesEn is how the accessibility checks name languages, in English.
func languageNamesEn(codes []string) string {
	names := make([]string, 0, len(codes))
	for _, c := range codes {
		names = append(names, accessibility.LanguageName(c))
	}
	return strings.Join(names, ", ")
}

// a11yTitle words a serious issue (other than contrast) from its fields:
// "The hero image has no alt text".
func a11yTitle(is accessibility.AccessibilityIssue) Text {
	e := is.Element
	section := func(prefix string) Text {
		switch e.Section {
		case "hero":
			return tx(prefix+".hero", nil)
		case "header":
			return tx(prefix+".logo", nil)
		case "", "image", "text":
			return tx(prefix+".image", nil)
		}
		return tx(prefix+".section", A{"section": argSection(e.Section)})
	}
	var t Text
	switch {
	case is.Kind == accessibility.KindAltText:
		t = section("a11y.alt")
		if t.en() != is.Title {
			if m := altSectionRe.FindStringSubmatch(is.Title); m != nil {
				t = tx("a11y.alt.section", A{"section": argSection(m[1])})
			}
		}
	case is.Kind == accessibility.KindLinkNoText && e.Role == "linked_image":
		t = section("a11y.linkedAlt")
		if t.en() != is.Title {
			if m := linkedSectionRe.FindStringSubmatch(is.Title); m != nil {
				t = tx("a11y.linkedAlt.section", A{"section": argSection(m[1])})
			}
		}
	case is.Kind == accessibility.KindLinkNoText:
		role := "link"
		if e.Role == "button" {
			role = "button"
		}
		t = tx("a11y.noText", A{"role": role, "inSection": yes(e.Section != ""), "section": argSection(e.Section)})
	}
	return verified(t, is.Title)
}

var (
	altSectionRe    = regexp.MustCompile(`^An image in the (.+) section has no alt text$`)
	linkedSectionRe = regexp.MustCompile(`^A linked image in the (.+) section has no alt text, so its link has no text$`)
)

// a11yShort names a moderate issue inside a sentence: "“Click here”", "a
// bare URL", "11 px footer text".
func a11yShort(is accessibility.AccessibilityIssue) Text {
	var candidates []Text
	switch is.Kind {
	case accessibility.KindLinkText:
		label := is.Element.Label
		if is.Element.Count > 1 {
			candidates = append(candidates, tx("a11y.short.bareMany", A{"count": is.Element.Count}), tx("a11y.short.linksMany", A{"count": is.Element.Count, "label": label}))
		}
		candidates = append(candidates, tx("a11y.short.bare", nil), tx("a11y.short.link", A{"label": label}))
	case accessibility.KindHeading:
		candidates = append(candidates, tx("a11y.short.heading", nil))
	case accessibility.KindTextSize:
		px := strconv.FormatFloat(is.Fix.FromPx, 'f', -1, 64)
		if is.Element.Section != "" {
			candidates = append(candidates, tx("a11y.short.sizeIn", A{"px": px, "section": argSection(is.Element.Section)}))
		}
		candidates = append(candidates, tx("a11y.short.size", A{"px": px}))
	}
	for _, c := range candidates {
		if c.en() == is.Short {
			return c
		}
	}
	return RawText(is.Short)
}

// colorText is "white" or "black" for those two, the hex value otherwise.
func colorText(hex string) any {
	switch strings.ToUpper(hex) {
	case "#FFFFFF":
		return tx("a11y.white", nil)
	case "#000000":
		return tx("a11y.black", nil)
	}
	return hex
}

func (b *builder) accessibility() {
	a := b.f.Accessibility
	if !a.Ran {
		return
	}
	if !a.Checked {
		b.add(check{Key: "accessibility.unchecked", Group: GroupAccessibility, Status: Unknown, Title: tx("a11y.unchecked", nil),
			Detail: a11yProblem(a.Problem), Label: groupTitle(GroupAccessibility)})
		return
	}
	r := a.Report
	status := Warn
	if r.Policy == accessibility.PolicyBlock {
		status = Block
	}
	var serious, moderate []accessibility.AccessibilityIssue
	for _, is := range r.Issues {
		if is.Severity == accessibility.Serious {
			serious = append(serious, is)
		} else {
			moderate = append(moderate, is)
		}
	}
	// Serious issues are confirmed together before sending.
	var ackParts []any
	for _, is := range serious {
		ackParts = append(ackParts, seriousAck(is))
	}
	var ack Text
	if len(ackParts) > 0 {
		ack = tx("a11y.ack", A{"issues": ArgAnd(ackParts)})
	}
	for _, is := range serious {
		c := check{Key: "accessibility." + is.ID, Group: GroupAccessibility, Status: status, Ack: ack, Fixes: []fix{}}
		switch is.Kind {
		case accessibility.KindContrast:
			c.Label = tx("label.contrast", nil)
			args := A{"label": is.Element.Label, "ratio": argDecimal1(accessibility.RoundDown(is.Fix.Ratio)), "needs": is.Fix.Needs}
			if is.Element.Role == "button" {
				c.Title = tx("a11y.contrastButton", args)
				c.Detail = tx("a11y.contrastOnButton", A{"against": argCapital(colorText(is.Fix.Against)), "from": is.Fix.From})
			} else {
				c.Title = tx("a11y.contrast", args)
				c.Detail = tx("a11y.contrastOn", A{"from": is.Fix.From, "against": colorText(is.Fix.Against)})
			}
			if len(r.Languages) > 1 {
				c.Detail = sentences(c.Detail, tx("a11y.colorShared", nil))
			}
			if len(is.Fix.Suggestions) > 0 {
				s := is.Fix.Suggestions[0]
				c.Fixes = append(c.Fixes, apply(FixAccessibilityColor, tx("fix.useColor", A{"color": s.Color}), "", map[string]string{"path": pathParam(is.Path), "color": s.Color,
					"target": is.Fix.Target, "from": is.Fix.From, "label": is.Element.Label}))
			} else {
				// No color passes against what it sits on: the composer's picker helps.
				c.Fixes = append(c.Fixes, link(tx("fix.openInComposer", nil), b.editor(is.Path, "accessibility")))
			}
		default:
			c.Label = tx("label.altText", nil)
			title := tx("a11y.titleIn", A{"title": a11yTitle(is), "languages": languageList(is.Languages)})
			c.Title = verified(title, is.Title+" · "+languageNamesEn(is.Languages))
			c.Detail = tx("a11y.altDetail", nil)
			if is.Kind == accessibility.KindLinkNoText {
				c.Label = tx("label.linkText", nil)
				c.Detail = tx("a11y.linkDetail", nil)
			}
			fixLabel := tx("fix.writeAlt", nil)
			if is.Fix.Kind == accessibility.KindLinkText {
				fixLabel = tx("fix.writeLinkText", nil)
			}
			c.Fixes = append(c.Fixes, link(fixLabel, b.editor(is.Path, "accessibility")))
			if is.Fix.Decorative {
				c.Fixes = append(c.Fixes, apply(FixAccessibilityDecorative, tx("fix.markDecorative", nil), "", map[string]string{"path": pathParam(is.Path),
					"index": fmt.Sprint(is.Element.Index), "name": decorativeName(is)}))
			}
		}
		c.FlatTitle = tx("a11y.flat", A{"title": argLower(c.Title)})
		b.add(c)
	}
	if a.Fixed != nil {
		english := fmt.Sprintf("Fixed by %s at %s", firstNonEmpty(a.Fixed.By, "someone"), Clock(a.Fixed.At, b.f.Now, b.loc))
		if len(a.Fixed.Languages) > 0 {
			english += " in " + accessibility.LanguageList(a.Fixed.Languages)
		}
		detail := tx("a11y.fixed", A{"name": a.Fixed.By, "named": yes(a.Fixed.By != ""), "when": b.when(a.Fixed.At),
			"inLanguages": yes(len(a.Fixed.Languages) > 0), "languages": languagesAnd(a.Fixed.Languages)})
		// The summary is the accessibility checks' own English.
		b.add(check{Key: "accessibility.fixed", Group: GroupAccessibility, Status: Pass, Title: RawText(a.Fixed.Summary),
			Detail: verified(detail, english+" · recorded in the audit log"), Label: tx("label.fixed", nil), Short: RawText(lowerFirst(a.Fixed.Summary))})
	}
	if len(moderate) > 0 {
		b.add(b.moderateRow(moderate, r))
	}
	if len(r.Issues) == 0 {
		b.add(check{Key: "accessibility.issues", Group: GroupAccessibility, Status: Pass, Title: tx("a11y.none", nil), Label: groupTitle(GroupAccessibility),
			Detail: tx("a11y.noneDetail", A{"images": r.Counts.Images, "px": r.MinTextPx}),
			Short:  tx("a11y.noneShort", nil)})
	}
	b.add(check{Key: "accessibility.lang_dir", Group: GroupAccessibility, Status: Pass, Title: tx("a11y.langDir", nil),
		Detail: tx("a11y.langDirDetail", A{"languages": strings.Join(r.Counts.Languages, ", "), "dir": r.Counts.Dir}), Label: tx("label.langDir", nil),
		Short: tx("a11y.langDirShort", nil)})
	tables := check{Key: "accessibility.layout_tables", Group: GroupAccessibility, Status: Pass, Label: tx("label.layoutTables", nil),
		Title: tx("a11y.tables", A{"count": r.Counts.LayoutTables}),
		Short: tx("a11y.tablesShort", A{"count": r.Counts.LayoutTables})}
	if r.Counts.RoleAdded > 0 {
		tables.Detail = tx("a11y.tablesAdded", nil)
	} else {
		tables.Detail = tx("a11y.tablesNone", nil)
	}
	b.add(tables)
}

// a11yProblem is why the accessibility checks couldn't run.
func a11yProblem(p string) Text {
	if strings.TrimSpace(p) == "" {
		return tx("check.waits", nil)
	}
	for _, k := range []string{"check.waits", "a11y.problemNoContent", "a11y.problemCompile"} {
		if t := tx(k, nil); t.en() == p {
			return t
		}
	}
	return RawText(p)
}

// moderateRow is the moderate issues together: "4 moderate issues: “Click
// here”, a bare URL, a skipped heading level, 11 px footer text".
func (b *builder) moderateRow(moderate []accessibility.AccessibilityIssue, r accessibility.AccessibilityReport) check {
	var phrases []any
	oneClick := 0
	var kinds []string
	for _, is := range moderate {
		phrases = append(phrases, a11yShort(is))
		if is.Fix.OneClick {
			oneClick++
			kinds = append(kinds, is.Kind)
		}
	}
	n := len(moderate)
	// Two read as a pair ("“Click here” and a bare URL"); more as a list.
	list := argJoin(", ", phrases)
	if len(phrases) == 2 {
		list = ArgAnd(phrases)
	}
	title := tx("a11y.moderate", A{"count": n, "list": list})
	c := check{Key: "accessibility.moderate", Group: GroupAccessibility, Status: Warn, Title: title, Label: tx("label.moderate", nil),
		Ack: tx("a11y.moderateAck", A{"count": n}), Fixes: []fix{}}
	switch {
	case oneClick > 0:
		hasHeading, hasSize := false, false
		for _, k := range kinds {
			if k == accessibility.KindHeading {
				hasHeading = true
			}
			if k == accessibility.KindTextSize {
				hasSize = true
			}
		}
		what := "none"
		switch {
		case hasHeading && hasSize:
			what = "both"
		case hasHeading:
			what = "heading"
		case hasSize:
			what = "size"
		}
		c.Detail = tx("a11y.oneClickDetail", A{"what": what})
		c.Fixes = append(c.Fixes, apply(FixAccessibilityOneClick, tx("fix.oneClick", A{"count": oneClick}), "primary", nil))
	default:
		c.Detail = tx("a11y.linkTextDetail", nil)
		for _, is := range moderate {
			if is.Kind == accessibility.KindLinkText && len(is.Languages) > 1 {
				var suggested, own []string
				for _, v := range is.Versions {
					if v.Suggestion != "" {
						suggested = append(suggested, v.Lang)
					} else {
						own = append(own, v.Lang)
					}
				}
				if len(suggested) > 0 && len(own) > 0 {
					page := tx("a11y.linkedPage", A{"page": languagesAnd(suggested), "own": languagesAnd(own), "count": len(own)})
					english := fmt.Sprintf("The linked page is in %s, so %s %s their own words.", accessibility.LanguageList(suggested), accessibility.LanguageList(own),
						Plural(int64(len(own)), "needs", "need"))
					c.Detail = sentences(c.Detail, verified(page, english))
				}
				break
			}
		}
	}
	c.Fixes = append(c.Fixes, link(tx("fix.openInComposer", nil), b.editor(moderate[0].Path, "accessibility")))
	c.FlatTitle = tx("a11y.flat", A{"title": argLower(title)})
	return c
}

// seriousAck is a serious issue inside the sentence a person confirms.
func seriousAck(is accessibility.AccessibilityIssue) any {
	switch is.Kind {
	case accessibility.KindContrast:
		return tx("a11y.contrastAck", A{"label": is.Element.Label, "ratio": argDecimal1(accessibility.RoundDown(is.Fix.Ratio)), "needs": is.Fix.Needs})
	}
	return argLower(a11yTitle(is))
}

func decorativeName(is accessibility.AccessibilityIssue) string {
	if i := strings.Index(is.Title, " has no alt text"); i > 0 {
		return is.Title[:i]
	}
	return "the image"
}

func pathParam(p []int) string {
	parts := make([]string, len(p))
	for i, x := range p {
		parts[i] = fmt.Sprint(x)
	}
	return strings.Join(parts, "-")
}

// accessibilityCaption is "6 issues in English, Spanish and German · 2
// serious", or "No issues in English, Spanish or German".
func (b *builder) accessibilityCaption() Text {
	a := b.f.Accessibility
	if !a.Checked {
		return tx("note.notChecked", nil)
	}
	r := a.Report
	codes := r.Counts.Languages
	if len(r.Issues) == 0 {
		names := make([]any, 0, len(codes))
		english := make([]string, 0, len(codes))
		for _, c := range codes {
			names = append(names, ArgLanguage(c))
			english = append(english, accessibility.LanguageName(c))
		}
		or := strings.Join(english, ", ")
		if len(english) > 1 {
			or = strings.Join(english[:len(english)-1], ", ") + " or " + english[len(english)-1]
		}
		return verified(tx("a11y.captionClean", A{"languages": ArgOr(names)}), "No issues in "+or)
	}
	t := tx("a11y.caption", A{"count": len(r.Issues), "languages": languagesAnd(codes), "serious": r.Serious})
	english := fmt.Sprintf("%d %s in %s", len(r.Issues), Plural(int64(len(r.Issues)), "issue", "issues"), accessibility.LanguageList(codes))
	if r.Serious > 0 {
		english += fmt.Sprintf(" · %d serious", r.Serious)
	}
	return verified(t, english)
}

// accessibilityNote is the line a Blocked verdict adds when the workspace
// blocks on serious issues: "Your workspace blocks sends with serious
// accessibility issues (Settings › General). Changed by Linh Nguyen at
// 14:22."
func (b *builder) accessibilityNote() Text {
	a := b.f.Accessibility
	if !a.Checked || a.Report.Policy != accessibility.PolicyBlock {
		return nil
	}
	if a.PolicyBy != "" && a.PolicyAt != nil {
		return sentences(tx("a11y.policy", nil), tx("a11y.policyChanged", A{"name": a.PolicyBy, "when": b.when(*a.PolicyAt)}))
	}
	return tx("a11y.policy", nil)
}
