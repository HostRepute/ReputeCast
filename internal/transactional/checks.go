package transactional

import (
	"context"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/store"
)

// Check severities.
const (
	SevBlock = "block"
	SevWarn  = "warn"
	SevInfo  = "info"
	SevPass  = "pass"
)

// TransactionalCheckFix is a one-click fix offered with a finding.
type TransactionalCheckFix struct {
	Kind  string `json:"kind" enum:"declare_variable,remove_variable,make_optional,add_test_value,insert_footer,set_alt,minify,edit_link,replace_link,move_to_body,turn_off_tracking"`
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
	// Variable and TestData say what the fix is about.
	Variable string `json:"variable,omitempty"`
	TestData string `json:"testData,omitempty"`
	Path     []int  `json:"path,omitempty"`
}

// TransactionalCheck is one finding of a template's checks: the composer's
// content checks plus the transactional ones (Liquid errors, undeclared and
// unused variables, test data that fails to render, a sensitive variable in
// the subject, tracking on an Essential template).
type TransactionalCheck struct {
	ID       string                  `json:"id" example:"variables.undeclared"`
	Severity string                  `json:"severity" enum:"block,warn,info,pass"`
	Group    string                  `json:"group" enum:"Content,Links,Compliance,Subject,Variables,Test data,Privacy,Accessibility"`
	Title    string                  `json:"title"`
	Detail   string                  `json:"detail,omitempty"`
	Where    string                  `json:"where,omitempty" enum:"subject,preheader,body,text"`
	Line     int                     `json:"line,omitempty"`
	Variable string                  `json:"variable,omitempty"`
	TestData string                  `json:"testData,omitempty"`
	Path     []int                   `json:"path"`
	Fixes    []TransactionalCheckFix `json:"fixes"`
}

// TransactionalTestRender is the result of rendering one test data set (or
// a recent real call) with the template.
type TransactionalTestRender struct {
	Name    string `json:"name"`
	Source  string `json:"source" enum:"test_data,last_call"`
	OK      bool   `json:"ok"`
	Subject string `json:"subject,omitempty"`
	// Missing lists the required variables the set lacks.
	Missing []string `json:"missing"`
	// Defaults lists the optional variables that fell back to their default.
	Defaults []string `json:"defaults"`
	Problem  string   `json:"problem,omitempty"`
	// Recipient and At describe a real call ("ops@contoso.example", 10:39).
	Recipient string `json:"recipient,omitempty"`
	At        string `json:"at,omitempty"`
}

// TransactionalVariableReport compares what a template declares with what
// it uses.
type TransactionalVariableReport struct {
	Used       []VariableUse `json:"used"`
	Undeclared []VariableUse `json:"undeclared"`
	Unused     []string      `json:"unused"`
}

// TransactionalChecks is the Checks tab and the publish dialog's summary.
type TransactionalChecks struct {
	Verdict   string                      `json:"verdict" enum:"ready,warnings,blocked"`
	Blocks    int                         `json:"blocks"`
	Warnings  int                         `json:"warnings"`
	Passed    int                         `json:"passed"`
	SizeBytes int                         `json:"sizeBytes"`
	ClipBytes int                         `json:"clipBytes"`
	Links     int                         `json:"links"`
	Images    int                         `json:"images"`
	Checks    []TransactionalCheck        `json:"checks"`
	Variables TransactionalVariableReport `json:"variables"`
	TestData  []TransactionalTestRender   `json:"testData"`
	PlainText string                      `json:"plainText"`
}

// TransactionalChecksInput runs the checks on the editor's current,
// unsaved content; omitted fields use the saved draft.
type TransactionalChecksInput struct {
	Subject     *string                  `json:"subject,omitempty" maxLength:"998"`
	Preheader   *string                  `json:"preheader,omitempty" maxLength:"500"`
	EditorMode  *string                  `json:"editorMode,omitempty" enum:"visual,mjml,html"`
	MJML        *string                  `json:"mjml,omitempty" maxLength:"2000000"`
	HTML        *string                  `json:"html,omitempty" maxLength:"2500000"`
	PlainText   *string                  `json:"plainText,omitempty" maxLength:"500000"`
	Variables   *[]TransactionalVariable `json:"variables,omitempty" maxItems:"200"`
	TestData    *[]TransactionalTestData `json:"testData,omitempty" maxItems:"50"`
	TrackOpens  *bool                    `json:"trackOpens,omitempty"`
	TrackClicks *bool                    `json:"trackClicks,omitempty"`
}

// checkSubject is what the checks run on.
type checkSubject struct {
	content
	TestData    []TransactionalTestData
	Kind        string // essential, optional
	TrackOpens  bool
	TrackClicks bool
	ProbeLinks  bool
	Address     string
	// WorkspaceID and TemplateID say whose policy and language versions the
	// accessibility checks use.
	WorkspaceID uuid.UUID
	TemplateID  uuid.UUID
}

// runChecks runs every check on a template's content, its words in the
// reader's language.
func (s *Service) runChecks(ctx context.Context, cs checkSubject) TransactionalChecks {
	l := lang(ctx)
	t := func(key string, args i18n.Args) string { return texts.M(key).Text(l, args) }
	out := TransactionalChecks{ClipBytes: contentcheck.ClipBytes, Checks: []TransactionalCheck{}, TestData: []TransactionalTestRender{},
		Variables: TransactionalVariableReport{Used: []VariableUse{}, Undeclared: []VariableUse{}, Unused: []string{}}}
	add := func(c TransactionalCheck) {
		if c.Fixes == nil {
			c.Fixes = []TransactionalCheckFix{}
		}
		if c.Path == nil {
			c.Path = []int{}
		}
		out.Checks = append(out.Checks, c)
	}
	body := cs.MJML
	if strings.TrimSpace(body) == "" {
		body = cs.HTML
	}

	// Liquid syntax: a template that doesn't parse can't be published.
	liquidOK := true
	for _, part := range []struct{ where, src string }{{"subject", cs.Subject}, {"preheader", cs.Preheader}, {"body", body}, {"text", cs.Text}} {
		for _, p := range liquid.Validate(part.src) {
			liquidOK = false
			title := t("check.liquid", nil)
			if p.Line > 0 && part.where == "body" {
				title = t("check.liquidLine", i18n.Args{"line": strconv.Itoa(p.Line)})
			} else if part.where != "body" {
				title = t("check.liquidIn", i18n.Args{"part": part.where})
			}
			add(TransactionalCheck{ID: "liquid.syntax", Severity: SevBlock, Group: "Variables", Where: part.where, Line: p.Line, Title: title,
				Detail: t("check.liquidDetail", i18n.Args{"problem": liquidMessage(l, p)})})
		}
	}

	// Declared versus used variables.
	uses := templateUses(cs.Subject, cs.Preheader, body, cs.Text)
	declared := map[string]TransactionalVariable{}
	for _, v := range cs.Variables {
		declared[v.Name] = v
	}
	usedNames := map[string]bool{}
	for _, u := range uses {
		if slices.Contains(reserved, u.Name) {
			continue
		}
		usedNames[u.Name] = true
		out.Variables.Used = append(out.Variables.Used, u)
		if _, ok := declared[u.Name]; ok {
			continue
		}
		out.Variables.Undeclared = append(out.Variables.Undeclared, u)
		add(TransactionalCheck{ID: "variables.undeclared", Severity: SevWarn, Group: "Variables", Where: u.Where, Line: u.Line, Variable: u.Name,
			Title:  t("check.undeclared", i18n.Args{"name": u.Name}),
			Detail: t("check.undeclaredDetail", i18n.Args{"name": u.Name, "where": u.Where, "line": strconv.Itoa(u.Line)}),
			Fixes: []TransactionalCheckFix{{Kind: "declare_variable", Label: t("check.fix.declare", nil), Variable: u.Name},
				{Kind: "remove_variable", Label: t("check.fix.remove", nil), Variable: u.Name}}})
	}
	for _, v := range cs.Variables {
		if usedNames[v.Name] {
			continue
		}
		out.Variables.Unused = append(out.Variables.Unused, v.Name)
		add(TransactionalCheck{ID: "variables.unused", Severity: SevInfo, Group: "Variables", Variable: v.Name,
			Title:  t("check.unused", i18n.Args{"name": v.Name}),
			Detail: t("check.unusedDetail", nil),
			Fixes:  []TransactionalCheckFix{{Kind: "remove_variable", Label: t("check.fix.removeDeclaration", nil), Variable: v.Name}}})
	}

	// A sensitive variable in the subject ends up in the log.
	for _, u := range usedVariables(cs.Subject, "subject") {
		if v, ok := declared[u.Name]; ok && v.Sensitive {
			add(TransactionalCheck{ID: "variables.sensitive_subject", Severity: SevWarn, Group: "Privacy", Where: "subject", Variable: u.Name,
				Title:  t("check.sensitive", i18n.Args{"name": u.Name}),
				Detail: t("check.sensitiveDetail", nil),
				Fixes:  []TransactionalCheckFix{{Kind: "move_to_body", Label: t("check.fix.moveToBody", nil), Variable: u.Name}}})
		}
	}

	// Test data must render: a set missing a required variable is a call
	// that would fail.
	if liquidOK {
		for _, td := range cs.TestData {
			r := s.testRender(l, cs.content, td.Name, "test_data", td.Data)
			out.TestData = append(out.TestData, r)
			if len(r.Missing) > 0 {
				v := r.Missing[0]
				add(TransactionalCheck{ID: "testdata.missing", Severity: SevBlock, Group: "Test data", TestData: td.Name, Variable: v,
					Title:  t("check.testData", i18n.Args{"name": td.Name}),
					Detail: t("check.testDataMissing", i18n.Args{"names": andList(l, r.Missing), "name": td.Name}),
					Fixes:  []TransactionalCheckFix{{Kind: "add_test_value", Label: t("check.fix.addTestValue", i18n.Args{"name": v}), Variable: v, TestData: td.Name}}})
				continue
			}
			if !r.OK {
				add(TransactionalCheck{ID: "testdata.render", Severity: SevBlock, Group: "Test data", TestData: td.Name, Title: t("check.testData", i18n.Args{"name": td.Name}), Detail: r.Problem})
			}
		}
	}

	// Tracking on an Essential template: a privacy note, not a block.
	if cs.Kind == KindEssential && (cs.TrackOpens || cs.TrackClicks) {
		what := "both"
		switch {
		case !cs.TrackClicks:
			what = "opens"
		case !cs.TrackOpens:
			what = "clicks"
		}
		add(TransactionalCheck{ID: "privacy.tracking", Severity: SevInfo, Group: "Privacy", Title: t("check.tracking", i18n.Args{"what": what}),
			Detail: t("check.trackingDetail", nil),
			Fixes:  []TransactionalCheckFix{{Kind: "turn_off_tracking", Label: t("check.fix.trackingOff", nil)}}})
	}

	// The composer's content checks, adapted: the unsubscribe link and the
	// footer address are required for Optional categories only.
	html, err := cs.compiledHTML()
	if err != nil {
		add(TransactionalCheck{ID: "content.compile", Severity: SevBlock, Group: "Content", Title: t("check.compile", nil), Detail: problemText(l, err)})
	}
	// Accessibility: alt text, link text, contrast, headings and text size,
	// listed after the other checks.
	var a11yChecks []TransactionalCheck
	a11y := false
	if err == nil {
		a11yChecks, a11y = s.accessibilityChecks(ctx, cs, html)
	}
	checker := s.Checker
	if checker == nil {
		checker = &contentcheck.Checker{}
	}
	rep, cerr := checker.Check(ctx, contentcheck.Input{Subject: cs.Subject, Preheader: cs.Preheader, MJML: cs.MJML, HTML: html,
		Address: cs.Address, ProbeLinks: cs.ProbeLinks})
	if cerr == nil {
		out.SizeBytes, out.Links, out.Images, out.PlainText = rep.SizeBytes, len(rep.LinkList), rep.Images, rep.PlainText
		hasUnsub, hasAddress := true, true
		for _, is := range rep.Issues {
			switch is.ID {
			case "personalization.syntax", "personalization.unknown_field":
				continue // the transactional checks above cover Liquid and variables
			case "content.alt_text":
				if a11y {
					continue // listed in the Accessibility group
				}
			case "compliance.footer":
				// Which part is missing, from the English title Check writes.
				hasUnsub = !strings.Contains(is.Title, "unsubscribe link")
				hasAddress = !strings.Contains(is.Title, "postal address")
				if cs.Kind != KindOptional {
					continue
				}
				add(TransactionalCheck{ID: is.ID, Severity: SevBlock, Group: "Compliance", Title: is.In(l).Title,
					Detail: t("check.footerDetail", nil),
					Fixes:  []TransactionalCheckFix{{Kind: "insert_footer", Label: t("check.fix.footer", nil)}}})
				continue
			case "subject.spam":
				is.Severity = contentcheck.Info
			}
			is = is.In(l)
			c := TransactionalCheck{ID: is.ID, Severity: is.Severity, Group: is.Group, Title: is.Title, Detail: is.Detail, Path: is.Path}
			for _, f := range is.Fixes {
				switch f.Kind {
				case "set_alt", "minify", "edit_link", "replace_link", "insert_footer":
					c.Fixes = append(c.Fixes, TransactionalCheckFix{Kind: f.Kind, Label: f.Label, Value: f.Value, Path: f.Path})
				}
			}
			add(c)
		}
		// Passing checks, as the publish dialog lists them.
		if cs.Kind == KindOptional && hasUnsub {
			add(TransactionalCheck{ID: "compliance.unsubscribe", Severity: SevPass, Group: "Compliance", Title: t("check.pass.unsubscribe", nil)})
		}
		if hasAddress {
			add(TransactionalCheck{ID: "compliance.address", Severity: SevPass, Group: "Compliance", Title: t("check.pass.address", nil)})
		}
		if rep.SizeBytes < contentcheck.NearClipBytes {
			add(TransactionalCheck{ID: "content.size", Severity: SevPass, Group: "Content",
				Title: t("check.pass.size", i18n.Args{"kb": int(math.Max(1, math.Round(float64(rep.SizeBytes)/1024)))})})
		}
		if n := len(rep.LinkList); n > 0 && rep.LinksBlocklisted == 0 && rep.LinksBroken == 0 {
			probed := "no"
			if cs.ProbeLinks {
				probed = "yes"
			}
			add(TransactionalCheck{ID: "links.ok", Severity: SevPass, Group: "Links", Title: t("check.pass.links", i18n.Args{"count": n, "probed": probed})})
		}
		if strings.TrimSpace(rep.PlainText) != "" {
			add(TransactionalCheck{ID: "content.plain_text", Severity: SevPass, Group: "Content", Title: t("check.pass.plainText", nil)})
		}
	}
	if liquidOK {
		add(TransactionalCheck{ID: "liquid.ok", Severity: SevPass, Group: "Variables", Title: t("check.pass.liquid", nil)})
	}
	if len(out.Variables.Undeclared) == 0 && len(cs.Variables) > 0 {
		add(TransactionalCheck{ID: "variables.ok", Severity: SevPass, Group: "Variables", Title: t("check.pass.variables", i18n.Args{"count": len(usedNames)})})
	}
	for _, c := range a11yChecks {
		add(c)
	}

	slices.SortStableFunc(out.Checks, func(a, b TransactionalCheck) int { return sevRank(a.Severity) - sevRank(b.Severity) })
	for _, c := range out.Checks {
		switch c.Severity {
		case SevBlock:
			out.Blocks++
		case SevWarn:
			out.Warnings++
		case SevPass:
			out.Passed++
		}
	}
	switch {
	case out.Blocks > 0:
		out.Verdict = "blocked"
	case out.Warnings > 0:
		out.Verdict = "warnings"
	default:
		out.Verdict = "ready"
	}
	return out
}

func sevRank(s string) int {
	switch s {
	case SevBlock:
		return 0
	case SevWarn:
		return 1
	case SevInfo:
		return 2
	}
	return 3
}

// testRender renders a template with one data set; its problem is in the
// reader's language (l).
func (s *Service) testRender(l i18n.Lang, c content, name, source string, data map[string]any) TransactionalTestRender {
	r := TransactionalTestRender{Name: name, Source: source, Missing: missingRequired(c.Variables, data), Defaults: []string{}}
	if len(r.Missing) > 0 {
		r.Problem = texts.M("check.requiredMissing").Text(l, i18n.Args{"names": andList(l, r.Missing)})
		return r
	}
	for _, v := range c.Variables {
		if val, ok := data[v.Name]; (!ok || val == nil || val == "") && !v.Required && v.Default != "" {
			r.Defaults = append(r.Defaults, v.Name)
		}
	}
	out, err := render(c, data, recipient{Email: "test@example.com", Name: "Test"}, map[string]string{"unsubscribe_url": "#", "preferences_url": "#"})
	if err != nil {
		r.Problem = problemText(l, err)
		return r
	}
	r.OK, r.Subject = true, out.Subject
	return r
}

func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// TemplateChecks runs the checks on a template's draft, or on the editor's
// unsaved content.
func (s *Service) TemplateChecks(ctx context.Context, ws *store.Workspace, ref string, in TransactionalChecksInput) (TransactionalChecks, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalChecks{}, err
	}
	t, err := s.Q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return TransactionalChecks{}, mapTemplateErr(ctx, err)
	}
	cs := s.draftSubject(ctx, ws, t)
	if in.Subject != nil {
		cs.Subject = *in.Subject
	}
	if in.Preheader != nil {
		cs.Preheader = *in.Preheader
	}
	if in.EditorMode != nil {
		cs.EditorMode = *in.EditorMode
	}
	if in.MJML != nil {
		cs.MJML = *in.MJML
		if in.HTML == nil {
			cs.HTML = ""
		}
	}
	if in.HTML != nil {
		cs.HTML = *in.HTML
	}
	if in.PlainText != nil {
		cs.Text = *in.PlainText
	}
	if in.Variables != nil {
		cs.Variables = *in.Variables
	}
	if in.TestData != nil {
		cs.TestData = *in.TestData
	}
	if in.TrackOpens != nil {
		cs.TrackOpens = *in.TrackOpens
	}
	if in.TrackClicks != nil {
		cs.TrackClicks = *in.TrackClicks
	}
	return s.runChecks(ctx, cs), nil
}

// draftSubject is what the checks of a template's draft run on.
func (s *Service) draftSubject(ctx context.Context, ws *store.Workspace, t store.TxnTemplateRow) checkSubject {
	return checkSubject{
		content:  content{Subject: t.Subject, Preheader: t.Preheader, EditorMode: t.EditorMode, MJML: t.Mjml, HTML: t.Html, Text: t.PlainText, Variables: parseVariables(t.Variables)},
		TestData: parseTestData(t.TestData), Kind: t.CategoryKind, TrackOpens: t.TrackOpens, TrackClicks: t.TrackClicks,
		Address: s.footer(ctx, ws), WorkspaceID: ws.ID, TemplateID: t.ID,
	}
}
