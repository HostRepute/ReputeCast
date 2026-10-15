package readiness

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Accessibility group (design expansion, X06): one row per serious
// issue, the moderate issues together in one row, the additions made when
// the email is compiled as passing rows.

// GroupAccessibility is the Accessibility group's key.
const GroupAccessibility = "accessibility"

func init() {
	groupTitles[GroupAccessibility] = "Accessibility"
	groupPrefixes[GroupAccessibility] = "Accessibility"
}

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

// accessibilityRunning is what the group shows while it runs.
const accessibilityRunning = "alt text, link text, contrast, headings and text size in every language"

func isAccessibilityFix(action string) bool {
	return action == FixAccessibilityDecorative || action == FixAccessibilityColor || action == FixAccessibilityOneClick
}

// languageNames is "English, Spanish, German".
func languageNames(codes []string) string {
	names := make([]string, 0, len(codes))
	for _, c := range codes {
		names = append(names, accessibility.LanguageName(c))
	}
	return strings.Join(names, ", ")
}

// languagesIn is "in English, Spanish and German".
func languagesIn(codes []string) string { return accessibility.LanguageList(codes) }

func (b *builder) accessibility() {
	a := b.f.Accessibility
	if !a.Ran {
		return
	}
	if !a.Checked {
		b.add(ReadinessCheck{Key: "accessibility.unchecked", Group: GroupAccessibility, Status: Unknown, Title: "Couldn't run the accessibility checks",
			Detail: firstNonEmpty(a.Problem, "The other checks finished; the verdict waits for this one."), Note: "not checked", Label: "Accessibility"})
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
	var ackParts []string
	for _, is := range serious {
		ackParts = append(ackParts, seriousAck(is))
	}
	ack := ""
	if len(ackParts) > 0 {
		ack = "I understand " + JoinAnd(ackParts) + "."
	}
	for _, is := range serious {
		c := ReadinessCheck{Key: "accessibility." + is.ID, Group: GroupAccessibility, Status: status, Ack: ack, Fixes: []ReadinessFix{}}
		switch is.Kind {
		case accessibility.KindContrast:
			c.Label = "Contrast"
			if is.Element.Role == "button" {
				c.Title = fmt.Sprintf("“%s” button: text contrast %s, below %s", is.Element.Label, accessibility.FormatRatio(is.Fix.Ratio), needText(is.Fix.Needs))
				c.Detail = fmt.Sprintf("%s on %s. The color is shared by every language, so one fix covers them all.", capitalize(colorWord(is.Fix.Against)), is.Fix.From)
			} else {
				c.Title = fmt.Sprintf("“%s”: text contrast %s, below %s", is.Element.Label, accessibility.FormatRatio(is.Fix.Ratio), needText(is.Fix.Needs))
				c.Detail = fmt.Sprintf("%s on %s. The color is shared by every language, so one fix covers them all.", is.Fix.From, colorWord(is.Fix.Against))
			}
			if len(is.Fix.Suggestions) > 0 {
				s := is.Fix.Suggestions[0]
				c.Fixes = append(c.Fixes, apply(FixAccessibilityColor, "Use "+s.Color, "", map[string]string{"path": pathParam(is.Path), "color": s.Color,
					"target": is.Fix.Target, "from": is.Fix.From, "label": is.Element.Label}))
			}
			c.Fixes = append(c.Fixes, link("Open in composer", b.editor(is.Path, "accessibility")))
		default:
			c.Label = "Alt text"
			c.Title = is.Title + " · " + languageNames(is.Languages)
			c.Detail = "Screen readers skip it or read the file name. Alt text is written in each language."
			if is.Kind == accessibility.KindLinkNoText {
				c.Label = "Link text"
				c.Detail = "Screen readers announce a link and nothing else. Its text is written in each language."
			}
			fixLabel := "Write alt text"
			if is.Fix.Kind == accessibility.KindLinkText {
				fixLabel = "Write link text"
			}
			c.Fixes = append(c.Fixes, link(fixLabel, b.editor(is.Path, "accessibility")))
			if is.Fix.Decorative {
				c.Fixes = append(c.Fixes, apply(FixAccessibilityDecorative, "Mark as decorative", "", map[string]string{"path": pathParam(is.Path),
					"index": fmt.Sprint(is.Element.Index), "name": decorativeName(is)}))
			}
		}
		c.FlatTitle = "Accessibility: " + lowerFirst(c.Title)
		b.add(c)
	}
	if a.Fixed != nil {
		when := b.when(a.Fixed.At)
		detail := fmt.Sprintf("Fixed by %s at %s", firstNonEmpty(a.Fixed.By, "someone"), when)
		if len(a.Fixed.Languages) > 0 {
			detail += " in " + languagesIn(a.Fixed.Languages)
		}
		b.add(ReadinessCheck{Key: "accessibility.fixed", Group: GroupAccessibility, Status: Pass, Title: a.Fixed.Summary,
			Detail: detail + " · recorded in the audit log", Label: "Fixed", Short: lowerFirst(a.Fixed.Summary)})
	}
	if len(moderate) > 0 {
		b.add(b.moderateRow(moderate, r))
	}
	if len(r.Issues) == 0 {
		b.add(ReadinessCheck{Key: "accessibility.issues", Group: GroupAccessibility, Status: Pass, Title: "No accessibility issues", Label: "Accessibility",
			Detail: fmt.Sprintf("%s, link text that says where it goes, AA contrast, headings in order, text at %d px or more.", altPhrase(r.Counts), r.MinTextPx),
			Short:  "no accessibility issues"})
	}
	langs := strings.Join(r.Counts.Languages, ", ")
	b.add(ReadinessCheck{Key: "accessibility.lang_dir", Group: GroupAccessibility, Status: Pass, Title: "`lang` and `dir` set for every language",
		Detail: langs + " · " + r.Counts.Dir + " · added when the email is compiled", Label: "lang and dir", Short: "lang and dir set"})
	tables := ReadinessCheck{Key: "accessibility.layout_tables", Group: GroupAccessibility, Status: Pass, Label: "Layout tables",
		Title: fmt.Sprintf("`role=\"presentation\"` on %s layout %s", Count(int64(r.Counts.LayoutTables)), Plural(int64(r.Counts.LayoutTables), "table", "tables")),
		Short: fmt.Sprintf("role=\"presentation\" on %s layout tables", Count(int64(r.Counts.LayoutTables)))}
	if r.Counts.RoleAdded > 0 {
		tables.Detail = "Added to the tables MJML missed, such as one in a Raw HTML block"
	} else {
		tables.Detail = "MJML marked every layout table; nothing to add"
	}
	b.add(tables)
}

// moderateRow is the moderate issues together: "4 moderate issues: “Click
// here”, a bare URL, a skipped heading level, 11 px footer text".
func (b *builder) moderateRow(moderate []accessibility.AccessibilityIssue, r accessibility.AccessibilityReport) ReadinessCheck {
	var phrases []string
	oneClick := 0
	var kinds []string
	for _, is := range moderate {
		switch is.Kind {
		case accessibility.KindLinkText:
			if strings.HasPrefix(is.Title, "A bare URL") {
				phrases = append(phrases, "a bare URL")
			} else {
				phrases = append(phrases, "“"+is.Element.Label+"”")
			}
		case accessibility.KindHeading:
			phrases = append(phrases, "a skipped heading level")
		case accessibility.KindTextSize:
			what := "text"
			if is.Element.Section != "" {
				what = is.Element.Section + " text"
			}
			phrases = append(phrases, fmt.Sprintf("%s px %s", trimNumber(is.Fix.FromPx, 1), what))
		}
		if is.Fix.OneClick {
			oneClick++
			kinds = append(kinds, is.Kind)
		}
	}
	n := len(moderate)
	title := fmt.Sprintf("%d moderate %s: %s", n, Plural(int64(n), "issue", "issues"), strings.Join(phrases, ", "))
	c := ReadinessCheck{Key: "accessibility.moderate", Group: GroupAccessibility, Status: Warn, Title: title, Label: "Moderate issues",
		Ack: fmt.Sprintf("I understand %d moderate accessibility %s %s.", n, Plural(int64(n), "issue", "issues"), Plural(int64(n), "remains", "remain")), Fixes: []ReadinessFix{}}
	switch {
	case oneClick > 0:
		var what []string
		hasHeading, hasSize := false, false
		for _, k := range kinds {
			if k == accessibility.KindHeading {
				hasHeading = true
			}
			if k == accessibility.KindTextSize {
				hasSize = true
			}
		}
		if hasHeading {
			what = append(what, "the heading level")
		}
		if hasSize {
			what = append(what, "the text size")
		}
		verb, obj := "needs", "it"
		if len(what) > 1 {
			verb, obj = "need", "them"
		}
		c.Detail = fmt.Sprintf("%s %s no words, so one click fixes %s in every language.", capitalize(JoinAnd(what)), verb, obj)
		c.Fixes = append(c.Fixes, apply(FixAccessibilityOneClick, fmt.Sprintf("Fix %d in one click", oneClick), "primary", nil))
	default:
		c.Detail = "Link text is fixed in the composer, language by language."
		for _, is := range moderate {
			if is.Kind == accessibility.KindLinkText && len(is.Languages) > 1 {
				var suggested, own []string
				for _, v := range is.Versions {
					if v.Suggestion != "" {
						suggested = append(suggested, accessibility.LanguageName(v.Lang))
					} else {
						own = append(own, accessibility.LanguageName(v.Lang))
					}
				}
				if len(suggested) > 0 && len(own) > 0 {
					c.Detail += fmt.Sprintf(" The linked page is in %s, so %s %s their own words.", JoinAnd(suggested), JoinAnd(own), Plural(int64(len(own)), "needs", "need"))
				}
				break
			}
		}
	}
	c.Fixes = append(c.Fixes, link("Open in composer", b.editor(moderate[0].Path, "accessibility")))
	c.FlatTitle = "Accessibility: " + lowerFirst(title)
	return c
}

func seriousAck(is accessibility.AccessibilityIssue) string {
	switch is.Kind {
	case accessibility.KindContrast:
		return fmt.Sprintf("“%s” is %s, below %s", is.Element.Label, accessibility.FormatRatio(is.Fix.Ratio), needText(is.Fix.Needs))
	}
	return lowerFirst(is.Title)
}

func needText(n float64) string { return strings.TrimSuffix(fmt.Sprintf("%.1f", n), ".0") + ":1" }

// colorWord is "white" or "black" for those two, the hex value otherwise.
func colorWord(hex string) string {
	switch strings.ToUpper(hex) {
	case "#FFFFFF":
		return "white"
	case "#000000":
		return "black"
	}
	return hex
}

func decorativeName(is accessibility.AccessibilityIssue) string {
	if i := strings.Index(is.Title, " has no alt text"); i > 0 {
		return is.Title[:i]
	}
	return "the image"
}

func altPhrase(c accessibility.AccessibilityCounts) string {
	switch c.Images {
	case 0:
		return "No images"
	case 1:
		return "Alt text on the image"
	case 2:
		return "Alt text on both images"
	}
	return fmt.Sprintf("Alt text on all %d images", c.Images)
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
func (b *builder) accessibilityCaption() string {
	a := b.f.Accessibility
	if !a.Checked {
		return "not checked"
	}
	r := a.Report
	codes := r.Counts.Languages
	if len(r.Issues) == 0 {
		names := make([]string, 0, len(codes))
		for _, c := range codes {
			names = append(names, accessibility.LanguageName(c))
		}
		or := strings.Join(names, ", ")
		if len(names) > 1 {
			or = strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
		}
		return "No issues in " + or
	}
	s := fmt.Sprintf("%d %s in %s", len(r.Issues), Plural(int64(len(r.Issues)), "issue", "issues"), languagesIn(codes))
	if r.Serious > 0 {
		s += fmt.Sprintf(" · %d serious", r.Serious)
	}
	return s
}

// AccessibilityPolicyNote is the line a Blocked verdict adds when the
// workspace blocks on serious issues: "Your workspace blocks sends with
// serious accessibility issues (Settings › General). Changed by Linh Nguyen
// at 14:22."
func (b *builder) accessibilityNote() string {
	a := b.f.Accessibility
	if !a.Checked || a.Report.Policy != accessibility.PolicyBlock {
		return ""
	}
	s := "Your workspace blocks sends with serious accessibility issues (Settings › General)."
	if a.PolicyBy != "" && a.PolicyAt != nil {
		s += fmt.Sprintf(" Changed by %s at %s.", a.PolicyBy, b.when(*a.PolicyAt))
	}
	return s
}
