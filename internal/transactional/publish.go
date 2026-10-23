package transactional

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// BreakingWindow is how far back the breaking-change check reads calls.
const BreakingWindow = 7 * 24 * time.Hour

// TransactionalTextChange is a before and after.
type TransactionalTextChange struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

// TransactionalVariableChange is a variable added, removed or changed
// between the live version and the draft.
type TransactionalVariableChange struct {
	Name    string                 `json:"name"`
	Change  string                 `json:"change" enum:"added,removed,changed"`
	Before  *TransactionalVariable `json:"before,omitempty"`
	After   *TransactionalVariable `json:"after,omitempty"`
	Summary string                 `json:"summary" example:"date · required"`
}

// TransactionalDiffLine is one line of a text diff.
type TransactionalDiffLine struct {
	Kind string `json:"kind" enum:"ctx,add,del"`
	Text string `json:"text"`
}

// TransactionalDiff is what changes when the draft is published.
type TransactionalDiff struct {
	Subject   *TransactionalTextChange      `json:"subject,omitempty"`
	Preheader *TransactionalTextChange      `json:"preheader,omitempty"`
	Variables []TransactionalVariableChange `json:"variables"`
	// ContentSummary is "2 text blocks changed · button “Upgrade now” is
	// now “Choose a plan”"; Content is the full text diff.
	ContentSummary string                  `json:"contentSummary"`
	Content        []TransactionalDiffLine `json:"content"`
	// Settings lists other changes (sender, reply-to, tracking).
	Settings []string `json:"settings"`
	Changed  bool     `json:"changed"`
	// contentEn and settingsEn are ContentSummary and Settings in English,
	// for the version's stored change summary.
	contentEn  string
	settingsEn []string
	// Languages are the language versions added, changed or removed.
	Languages []TransactionalLanguageChange `json:"languages,omitempty"`
}

// TransactionalBreakingVariable is a variable that would make recent calls fail.
type TransactionalBreakingVariable struct {
	Name string `json:"name"`
	// Calls is how many of the recent calls didn't send it.
	CallsWithout     int    `json:"callsWithout"`
	Type             string `json:"type"`
	SuggestedDefault string `json:"suggestedDefault"`
}

// TransactionalBreakingChange compares the draft with the requests of the
// last 7 days.
type TransactionalBreakingChange struct {
	Calls int `json:"calls" doc:"Calls to the template in the last 7 days"`
	// Breaking lists required variables recent calls didn't send.
	Breaking []TransactionalBreakingVariable `json:"breaking"`
	// Unused lists variables recent calls send that the draft no longer declares.
	Unused []string `json:"unused"`
	// Summary is the dialog's sentence, empty when nothing breaks.
	Summary string `json:"summary"`
}

// TransactionalPublishApproval says whether the workspace's approval rule applies.
type TransactionalPublishApproval struct {
	Required  bool     `json:"required"`
	Approvers []string `json:"approvers"`
	Reason    string   `json:"reason,omitempty"`
}

// TransactionalPublishPreview is everything the publish dialog shows.
type TransactionalPublishPreview struct {
	TemplateID    string                       `json:"templateId"`
	Key           string                       `json:"key"`
	Name          string                       `json:"name"`
	Category      *TransactionalCategoryRef    `json:"category,omitempty"`
	NextVersion   int                          `json:"nextVersion"`
	LiveVersion   *int                         `json:"liveVersion,omitempty"`
	LiveSince     *time.Time                   `json:"liveSince,omitempty"`
	LiveBy        string                       `json:"liveBy,omitempty"`
	DraftRevision int                          `json:"draftRevision"`
	DraftSavedAt  time.Time                    `json:"draftSavedAt"`
	DraftSavedBy  string                       `json:"draftSavedBy"`
	Diff          TransactionalDiff            `json:"diff"`
	Breaking      TransactionalBreakingChange  `json:"breaking"`
	TestRenders   []TransactionalTestRender    `json:"testRenders"`
	Checks        TransactionalChecks          `json:"checks"`
	Approval      TransactionalPublishApproval `json:"approval"`
	// Blocking lists why the draft can't be published (Liquid errors, test
	// data that doesn't render); empty when it can.
	Blocking   []string `json:"blocking"`
	CanPublish bool     `json:"canPublish"`
	// Languages are the language versions published with it, the source
	// first (empty for a template in one language).
	Languages []TransactionalPublishLanguage `json:"languages"`
}

// TransactionalMakeOptional turns a required variable into an optional one
// with a default when publishing ("Make trial_end_date optional, with the
// default “soon”").
type TransactionalMakeOptional struct {
	Name    string `json:"name" minLength:"1" maxLength:"64"`
	Default string `json:"default" maxLength:"4000"`
}

// TransactionalPublishInput publishes the draft.
type TransactionalPublishInput struct {
	// BaseRevision is the draft revision the person reviewed; a newer save
	// since then answers 409 version_conflict.
	BaseRevision int    `json:"baseRevision" minimum:"1"`
	Note         string `json:"note,omitempty" maxLength:"300" doc:"Shown in the version history instead of the generated summary"`
	// MakeOptional applies the dialog's recommended fix before publishing.
	MakeOptional []TransactionalMakeOptional `json:"makeOptional,omitempty" maxItems:"50"`
	// PublishAnyway publishes despite a breaking change.
	PublishAnyway bool `json:"publishAnyway,omitempty"`
}

// TransactionalPublishPreviewInput recomputes the dialog with a fix applied.
type TransactionalPublishPreviewInput struct {
	MakeOptional []TransactionalMakeOptional `json:"makeOptional,omitempty" maxItems:"50"`
}

// TransactionalPublished is the result of publishing or rolling back.
type TransactionalPublished struct {
	Version  int                   `json:"version"`
	Previous *int                  `json:"previous,omitempty" doc:"The version that was live before (Undo publishes it again)"`
	Template TransactionalTemplate `json:"template"`
}

// TransactionalVersion is one published version in the history.
type TransactionalVersion struct {
	Version     int       `json:"version"`
	Live        bool      `json:"live"`
	PublishedAt time.Time `json:"publishedAt"`
	PublishedBy string    `json:"publishedBy"`
	AvatarColor string    `json:"avatarColor"`
	Change      string    `json:"change"`
	Subject     string    `json:"subject"`
	SizeBytes   int       `json:"sizeBytes"`
}

// TransactionalVersionList is the version history, newest first.
type TransactionalVersionList struct {
	Items []TransactionalVersion `json:"items"`
	// CanPublish is transactional:publish ("Publish again").
	CanPublish bool `json:"canPublish"`
}

// TransactionalVersionDetail is a version with its content (compare).
type TransactionalVersionDetail struct {
	TransactionalVersion
	Preheader   string                  `json:"preheader"`
	EditorMode  string                  `json:"editorMode"`
	MJML        string                  `json:"mjml"`
	HTML        string                  `json:"html"`
	PlainText   string                  `json:"plainText"`
	Variables   []TransactionalVariable `json:"variables"`
	TestData    []TransactionalTestData `json:"testData"`
	Sender      *TransactionalSenderRef `json:"sender,omitempty"`
	ReplyTo     string                  `json:"replyTo"`
	TrackOpens  bool                    `json:"trackOpens"`
	TrackClicks bool                    `json:"trackClicks"`
	Note        string                  `json:"note"`
}

// applyOptional makes the named variables optional with their defaults.
func applyOptional(ctx context.Context, vars []TransactionalVariable, fixes []TransactionalMakeOptional) ([]TransactionalVariable, error) {
	out := slices.Clone(vars)
	for _, f := range fixes {
		i := slices.IndexFunc(out, func(v TransactionalVariable) bool { return v.Name == f.Name })
		if i < 0 {
			return nil, api.Invalid("body.makeOptional", say(ctx, "publish.notDeclared", i18n.Args{"name": f.Name}))
		}
		out[i].Required = false
		out[i].Default = f.Default
		if f.Default != "" {
			if _, err := typedValue(out[i].Type, f.Default); err != nil {
				return nil, api.Invalid("body.makeOptional", say(ctx, "variable.badValue",
					i18n.Args{"field": "default", "name": f.Name, "type": out[i].Type, "problem": problemText(lang(ctx), err)}))
			}
		}
	}
	return out, nil
}

// variableSummary is a declaration in the reader's words: "URL · required
// · sensitive".
func variableSummary(l i18n.Lang, v TransactionalVariable) string {
	t := func(key string, args i18n.Args) string { return texts.M(key).Text(l, args) }
	parts := []string{t("variable.type", i18n.Args{"type": v.Type})}
	if v.Required {
		parts = append(parts, t("variable.required", nil))
	} else {
		parts = append(parts, t("variable.optional", nil))
	}
	if v.Default != "" {
		parts = append(parts, t("variable.default", i18n.Args{"value": v.Default}))
	}
	if v.Sensitive {
		parts = append(parts, t("variable.sensitive", nil))
	}
	return strings.Join(parts, " · ")
}

// diffVariables lists what changed between two declarations.
func diffVariables(l i18n.Lang, before, after []TransactionalVariable) []TransactionalVariableChange {
	out := []TransactionalVariableChange{}
	old := map[string]TransactionalVariable{}
	for _, v := range before {
		old[v.Name] = v
	}
	now := map[string]bool{}
	for _, v := range after {
		now[v.Name] = true
		b, ok := old[v.Name]
		a := v
		switch {
		case !ok:
			out = append(out, TransactionalVariableChange{Name: v.Name, Change: "added", After: &a, Summary: variableSummary(l, v)})
		case b.Type != v.Type || b.Required != v.Required || b.Sensitive != v.Sensitive || b.Default != v.Default:
			bb := b
			out = append(out, TransactionalVariableChange{Name: v.Name, Change: "changed", Before: &bb, After: &a, Summary: variableSummary(l, b) + " → " + variableSummary(l, v)})
		}
	}
	for _, v := range before {
		if !now[v.Name] {
			b := v
			out = append(out, TransactionalVariableChange{Name: v.Name, Change: "removed", Before: &b, Summary: variableSummary(l, v)})
		}
	}
	return out
}

var buttonRe = regexp.MustCompile(`(?is)<mj-button[^>]*>(.*?)</mj-button>`)
var tagRe = regexp.MustCompile(`<[^>]*>`)

func buttonLabels(src string) []string {
	out := []string{}
	for _, m := range buttonRe.FindAllStringSubmatch(src, -1) {
		out = append(out, strings.TrimSpace(htmlUnescape(tagRe.ReplaceAllString(m[1], ""))))
	}
	return out
}

// textLines splits the plain text of an email into its non-empty lines.
func textLines(html string) []string {
	out := []string{}
	for _, l := range strings.Split(plainText(html), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// lineDiff is a longest-common-subsequence diff of two lists of lines.
func lineDiff(a, b []string) []TransactionalDiffLine {
	n, m := len(a), len(b)
	if n*m > 4_000_000 {
		out := []TransactionalDiffLine{}
		for _, l := range a {
			out = append(out, TransactionalDiffLine{Kind: "del", Text: l})
		}
		for _, l := range b {
			out = append(out, TransactionalDiffLine{Kind: "add", Text: l})
		}
		return out
	}
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	out := []TransactionalDiffLine{}
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, TransactionalDiffLine{Kind: "ctx", Text: a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, TransactionalDiffLine{Kind: "del", Text: a[i]})
			i++
		default:
			out = append(out, TransactionalDiffLine{Kind: "add", Text: b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, TransactionalDiffLine{Kind: "del", Text: a[i]})
	}
	for ; j < m; j++ {
		out = append(out, TransactionalDiffLine{Kind: "add", Text: b[j]})
	}
	return out
}

// versionContent is one side of a diff.
type versionContent struct {
	content
	SenderID    *uuid.UUID
	SenderEmail string
	ReplyTo     string
	TrackOpens  bool
	TrackClicks bool
}

// buildDiff compares the live version with the draft, its summaries in the
// reader's language (l), and in English for the version's stored change
// summary (changeSummary).
func buildDiff(l i18n.Lang, live *versionContent, draft versionContent) TransactionalDiff {
	d := TransactionalDiff{Variables: []TransactionalVariableChange{}, Content: []TransactionalDiffLine{}, Settings: []string{}}
	if live == nil {
		d.Changed = true
		d.Variables = diffVariables(l, nil, draft.Variables)
		html, _ := draft.compiledHTML()
		for _, line := range textLines(html) {
			d.Content = append(d.Content, TransactionalDiffLine{Kind: "add", Text: line})
		}
		d.ContentSummary = texts.M("diff.first").Text(l, nil)
		d.contentEn = texts.M("diff.first").Text(i18n.Default, nil)
		return d
	}
	if live.Subject != draft.Subject {
		d.Subject = &TransactionalTextChange{Before: live.Subject, After: draft.Subject}
	}
	if live.Preheader != draft.Preheader {
		d.Preheader = &TransactionalTextChange{Before: live.Preheader, After: draft.Preheader}
	}
	d.Variables = diffVariables(l, live.Variables, draft.Variables)
	lh, _ := live.compiledHTML()
	dh, _ := draft.compiledHTML()
	d.Content = lineDiff(textLines(lh), textLines(dh))
	changed := 0
	for i := 0; i < len(d.Content); i++ {
		if d.Content[i].Kind == "ctx" {
			continue
		}
		changed++
		for i+1 < len(d.Content) && d.Content[i+1].Kind != "ctx" {
			i++
		}
	}
	// What changed in the content and the settings, in a language.
	words := func(l i18n.Lang) (string, []string) {
		t := func(key string, args i18n.Args) string { return texts.M(key).Text(l, args) }
		parts := []string{}
		lb, db := buttonLabels(live.MJML), buttonLabels(draft.MJML)
		buttonChanges := 0
		for i := 0; i < len(lb) && i < len(db); i++ {
			if lb[i] != db[i] && lb[i] != "" && db[i] != "" {
				parts = append(parts, t("diff.button", i18n.Args{"before": lb[i], "after": db[i]}))
				buttonChanges++
			}
		}
		if blocks := changed - buttonChanges; blocks > 0 {
			parts = append([]string{t("diff.blocks", i18n.Args{"count": blocks})}, parts...)
		}
		if len(parts) == 0 && (lh != dh || live.MJML != draft.MJML) {
			parts = append(parts, t("diff.layoutOnly", nil))
		}
		orNone := func(s string) string {
			if s == "" {
				return t("diff.default", nil)
			}
			return s
		}
		onOff := func(b bool) string {
			if b {
				return t("diff.on", nil)
			}
			return t("diff.off", nil)
		}
		settings := []string{}
		if idString(live.SenderID) != idString(draft.SenderID) {
			settings = append(settings, t("diff.sender", i18n.Args{"before": orNone(live.SenderEmail), "after": orNone(draft.SenderEmail)}))
		}
		if live.ReplyTo != draft.ReplyTo {
			settings = append(settings, t("diff.replyTo", i18n.Args{"before": orNone(live.ReplyTo), "after": orNone(draft.ReplyTo)}))
		}
		if live.TrackOpens != draft.TrackOpens {
			settings = append(settings, t("diff.opens", i18n.Args{"before": onOff(live.TrackOpens), "after": onOff(draft.TrackOpens)}))
		}
		if live.TrackClicks != draft.TrackClicks {
			settings = append(settings, t("diff.clicks", i18n.Args{"before": onOff(live.TrackClicks), "after": onOff(draft.TrackClicks)}))
		}
		return strings.Join(parts, " · "), settings
	}
	d.ContentSummary, d.Settings = words(l)
	d.contentEn, d.settingsEn = words(i18n.Default)
	d.Changed = d.Subject != nil || d.Preheader != nil || len(d.Variables) > 0 || d.ContentSummary != "" || len(d.Settings) > 0
	return d
}

// suggestedDefault is what an optional variable can fall back to so old
// calls keep working. It becomes the template's content, so a date's
// ("soon") is in the workspace's language, wl.
func suggestedDefault(wl i18n.Lang, t string) string {
	switch t {
	case TypeDate:
		return texts.M("publish.soonDefault").Text(wl, nil)
	case TypeNumber:
		return "0"
	case TypeBoolean:
		return "false"
	case TypeList:
		return "[]"
	case TypeObject:
		return "{}"
	}
	return ""
}

// breaking compares the draft's required variables with what the calls of
// the last 7 days sent; wl is the workspace's language.
func breaking(wl i18n.Lang, calls []store.TxnRecentCallKeysRow, live *versionContent, draft []TransactionalVariable) TransactionalBreakingChange {
	out := TransactionalBreakingChange{Calls: len(calls), Breaking: []TransactionalBreakingVariable{}, Unused: []string{}}
	if len(calls) == 0 {
		return out
	}
	declaredBefore := map[string]bool{}
	requiredBefore := map[string]bool{}
	if live != nil {
		for _, v := range live.Variables {
			declaredBefore[v.Name] = true
			requiredBefore[v.Name] = v.Required
		}
	}
	for _, v := range draft {
		if !v.Required || requiredBefore[v.Name] {
			continue
		}
		without := 0
		for _, c := range calls {
			if !slices.Contains(c.DataKeys, v.Name) {
				without++
			}
		}
		if without > 0 {
			out.Breaking = append(out.Breaking, TransactionalBreakingVariable{Name: v.Name, CallsWithout: without, Type: v.Type, SuggestedDefault: suggestedDefault(wl, v.Type)})
		}
	}
	now := map[string]bool{}
	for _, v := range draft {
		now[v.Name] = true
	}
	sent := map[string]int{}
	for _, c := range calls {
		for _, k := range c.DataKeys {
			sent[k]++
		}
	}
	for name := range declaredBefore {
		if !now[name] && sent[name] > 0 {
			out.Unused = append(out.Unused, name)
		}
	}
	slices.Sort(out.Unused)
	return out
}

// breakingSummary says what a breaking change does, in the reader's
// language: "reset_url is now required, and none of the 4 calls in the last
// 7 days sent it. Those calls would fail with missing_variables and send
// nothing." next is the version the draft would become.
func breakingSummary(l i18n.Lang, b TransactionalBreakingChange, next int) string {
	if len(b.Breaking) == 0 {
		return ""
	}
	t := func(key string, args i18n.Args) string { return texts.M(key).Text(l, args) }
	first := b.Breaking[0]
	parts := []string{t("breaking.none", i18n.Args{"name": first.Name, "calls": b.Calls})}
	if first.CallsWithout < b.Calls {
		parts[0] = t("breaking.some", i18n.Args{"name": first.Name, "without": first.CallsWithout, "calls": b.Calls})
	}
	if len(b.Unused) > 0 {
		parts = append(parts, t("breaking.unused", i18n.Args{"names": andList(l, b.Unused), "version": strconv.Itoa(next)}))
	}
	parts = append(parts, t("breaking.fail", nil))
	return strings.Join(parts, " ")
}

// loadLive returns the live version's content, nil before the first publish.
func (s *Service) loadLive(ctx context.Context, q *store.Queries, t store.TxnTemplateRow) (*versionContent, error) {
	if t.LiveVersion == nil {
		return nil, nil
	}
	v, err := q.TxnVersion(ctx, store.TxnVersionParams{TemplateID: t.ID, Version: *t.LiveVersion})
	if err != nil {
		return nil, err
	}
	return &versionContent{
		content:  content{Subject: v.Subject, Preheader: v.Preheader, EditorMode: v.EditorMode, MJML: v.Mjml, HTML: v.Html, Text: v.PlainText, Variables: parseVariables(v.Variables)},
		SenderID: v.SenderID, SenderEmail: v.SenderEmail, ReplyTo: v.ReplyTo, TrackOpens: v.TrackOpens, TrackClicks: v.TrackClicks,
	}, nil
}

// PublishPreview builds the publish dialog.
func (s *Service) PublishPreview(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in TransactionalPublishPreviewInput) (TransactionalPublishPreview, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalPublishPreview{}, err
	}
	t, err := s.Q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return TransactionalPublishPreview{}, mapTemplateErr(ctx, err)
	}
	return s.publishPreview(ctx, s.Q, ws, p, t, in.MakeOptional)
}

func (s *Service) publishPreview(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, t store.TxnTemplateRow, fixes []TransactionalMakeOptional) (TransactionalPublishPreview, error) {
	live, err := s.loadLive(ctx, q, t)
	if err != nil {
		return TransactionalPublishPreview{}, err
	}
	vars, err := applyOptional(ctx, parseVariables(t.Variables), fixes)
	if err != nil {
		return TransactionalPublishPreview{}, err
	}
	draft := versionContent{content: content{Subject: t.Subject, Preheader: t.Preheader, EditorMode: t.EditorMode, MJML: t.Mjml, HTML: t.Html, Text: t.PlainText, Variables: vars},
		SenderID: t.SenderID, SenderEmail: t.SenderEmail, ReplyTo: t.ReplyTo, TrackOpens: t.TrackOpens, TrackClicks: t.TrackClicks}
	next := int(t.VersionsCount) + 1
	out := TransactionalPublishPreview{
		TemplateID: t.ID.String(), Key: t.Key, Name: t.Name, Category: categoryRef(t.CategoryID, t.CategoryName, t.CategoryKind), NextVersion: next,
		LiveSince: t.LiveSince, LiveBy: t.LiveByName, DraftRevision: int(t.DraftRevision), DraftSavedAt: t.DraftSavedAt, DraftSavedBy: t.DraftSavedByName,
		Diff: buildDiff(lang(ctx), live, draft), TestRenders: []TransactionalTestRender{}, Blocking: []string{},
		Approval: TransactionalPublishApproval{Approvers: []string{}}, Languages: []TransactionalPublishLanguage{},
	}
	if t.LiveVersion != nil {
		v := int(*t.LiveVersion)
		out.LiveVersion = &v
	}
	if s.Translations != nil {
		if out.Languages, err = s.Translations.Publishing(ctx, q, ws, t.ID); err != nil {
			return out, err
		}
		if t.LiveVersion != nil {
			if out.Diff.Languages, err = s.Translations.Changes(ctx, q, ws, t.ID, int(*t.LiveVersion)); err != nil {
				return out, err
			}
			out.Diff.Changed = out.Diff.Changed || len(out.Diff.Languages) > 0
		}
	}
	calls, err := q.TxnRecentCallKeys(ctx, store.TxnRecentCallKeysParams{TemplateID: &t.ID, Since: s.now().Add(-BreakingWindow)})
	if err != nil {
		return out, err
	}
	out.Breaking = breaking(workspaceLang(ctx, q, ws.ID), calls, live, vars)
	out.Breaking.Summary = breakingSummary(lang(ctx), out.Breaking, next)
	cs := s.draftSubject(ctx, ws, t)
	cs.Variables = vars
	cs.ProbeLinks = true
	out.Checks = publishChecks(s.runChecks(ctx, cs))
	out.TestRenders = append(out.TestRenders, out.Checks.TestData...)
	if len(calls) > 0 {
		c := calls[0]
		var data map[string]any
		_ = json.Unmarshal(c.Data, &data)
		r := s.testRender(lang(ctx), draft.content, say(ctx, "check.realCall", i18n.Args{"time": clockIn(ctx, c.AcceptedAt.In(location(ws)))}), "last_call", data)
		r.Recipient = c.Email
		r.At = c.AcceptedAt.Format(time.RFC3339)
		out.TestRenders = append(out.TestRenders, r)
	}
	for _, c := range out.Checks.Checks {
		if c.Severity == SevBlock {
			out.Blocking = append(out.Blocking, c.Title)
		}
	}
	if !out.Diff.Changed && live != nil {
		out.Blocking = append(out.Blocking, say(ctx, "publish.noChanges", i18n.Args{"version": strconv.Itoa(*out.LiveVersion)}))
	}
	if s.Gate != nil {
		dec, err := s.Gate.Check(ctx, q, PublishRequest{WorkspaceID: ws.ID, TemplateID: t.ID, TemplateKey: t.Key, Version: next, DraftHash: t.DraftHash, Principal: p})
		if err != nil {
			return out, err
		}
		out.Approval = TransactionalPublishApproval{Required: dec.NeedsApproval, Approvers: nonNilStrings(dec.Approvers), Reason: dec.Reason}
	}
	out.CanPublish = p.Can(authz.TransactionalPublish) && len(out.Blocking) == 0 && !out.Approval.Required
	return out, nil
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// location is the workspace's timezone.
func location(ws *store.Workspace) *time.Location {
	if loc, err := time.LoadLocation(ws.Timezone); err == nil {
		return loc
	}
	return time.UTC
}

// changeSummary is the version history's "Change" column.
func changeSummary(d TransactionalDiff, fixes []TransactionalMakeOptional, first bool) string {
	if first {
		return "First version"
	}
	parts := []string{}
	var added, removed []string
	for _, v := range d.Variables {
		switch v.Change {
		case "added":
			added = append(added, v.Name)
		case "removed":
			removed = append(removed, v.Name)
		}
	}
	switch {
	case len(added) > 0 && len(removed) > 0:
		parts = append(parts, joinAnd(added)+" "+plural(len(added), "replaces", "replace")+" "+joinAnd(removed))
	case len(added) > 0:
		parts = append(parts, "adds "+joinAnd(added))
	case len(removed) > 0:
		parts = append(parts, "removes "+joinAnd(removed))
	}
	for _, f := range fixes {
		if f.Default != "" {
			parts = append(parts, "default “"+f.Default+"”")
		}
	}
	if d.Subject != nil {
		parts = append(parts, "new subject")
	}
	if len(parts) == 0 && d.contentEn != "" {
		parts = append(parts, d.contentEn)
	}
	if len(parts) == 0 && len(d.settingsEn) > 0 {
		parts = append(parts, strings.ToLower(d.settingsEn[0][:1])+d.settingsEn[0][1:])
	}
	return strings.Join(parts, " · ")
}

// Publish makes the draft the live version.
func (s *Service) Publish(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in TransactionalPublishInput) (TransactionalPublished, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalPublished{}, err
	}
	var out TransactionalPublished
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.TxnTemplateForUpdate(ctx, store.TxnTemplateForUpdateParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return mapTemplateErr(ctx, err)
		}
		t, err := q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapTemplateErr(ctx, err)
		}
		if t.ArchivedAt != nil {
			return api.Conflict(say(ctx, "publish.archived", i18n.Args{"key": t.Key}))
		}
		if in.BaseRevision != int(t.DraftRevision) {
			who := t.DraftSavedByName
			if who == "" {
				who = say(ctx, "template.someone", nil)
			}
			return api.Err(http.StatusConflict, CodeVersionConflict, say(ctx, "publish.conflict", i18n.Args{"who": who, "time": clockIn(ctx, t.DraftSavedAt)})).
				WithMeta("draftRevision", t.DraftRevision)
		}
		pv, err := s.publishPreview(ctx, q, ws, p, t, in.MakeOptional)
		if err != nil {
			return err
		}
		if len(pv.Blocking) > 0 {
			return api.Err(http.StatusConflict, CodeTemplateDoesntRun, say(ctx, "publish.blocked", i18n.Args{"reason": strings.TrimSuffix(pv.Blocking[0], ".")})).WithMeta("blocking", pv.Blocking)
		}
		if len(pv.Breaking.Breaking) > 0 && !in.PublishAnyway {
			return api.Err(http.StatusConflict, CodeBreakingChange, say(ctx, "publish.breaking", i18n.Args{"summary": pv.Breaking.Summary})).
				WithMeta("breaking", pv.Breaking.Breaking)
		}
		req := PublishRequest{WorkspaceID: ws.ID, TemplateID: id, TemplateKey: t.Key, Version: pv.NextVersion, DraftHash: t.DraftHash, Principal: p}
		if s.Gate != nil {
			dec, err := s.Gate.Check(ctx, q, req)
			if err != nil {
				return err
			}
			if !dec.Allowed {
				return api.Err(http.StatusConflict, CodeApprovalRequired, orDefault(dec.Reason, say(ctx, "publish.needsApproval", nil))).
					WithMeta("approvers", nonNilStrings(dec.Approvers))
			}
		}
		vars, _ := applyOptional(ctx, parseVariables(t.Variables), in.MakeOptional)
		now := s.now()
		version := int32(pv.NextVersion)
		html := t.Html
		if strings.TrimSpace(html) == "" {
			html, _ = content{Subject: t.Subject, Preheader: t.Preheader, MJML: t.Mjml}.compiledHTML()
		}
		// lang, dir and the layout tables' role (accessibility): already there
		// when the draft was saved through the API, added to HTML stored
		// before them.
		html = accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerTransactional, id, 0, html)
		hash := draftHash(t.Subject, t.Preheader, t.EditorMode, t.Mjml, t.Html, t.PlainText, vars, t.SenderID, t.ReplyTo, t.TrackOpens, t.TrackClicks)
		change := strings.TrimSpace(in.Note)
		if change == "" {
			change = changeSummary(pv.Diff, in.MakeOptional, t.LiveVersion == nil)
		}
		if change == "" && len(pv.Diff.Languages) > 0 {
			change = languageChangeSummary(pv.Diff.Languages)
		}
		if err := q.TxnVersionInsert(ctx, store.TxnVersionInsertParams{TemplateID: id, Version: version, Subject: t.Subject, Preheader: t.Preheader, EditorMode: t.EditorMode,
			Mjml: t.Mjml, Html: html, PlainText: t.PlainText, Variables: mustJSON(vars), TestData: t.TestData, SenderID: t.SenderID, ReplyTo: t.ReplyTo,
			TrackOpens: t.TrackOpens, TrackClicks: t.TrackClicks, ContentHash: hash, SizeBytes: int32(len(html)), Change: change, Note: strings.TrimSpace(in.Note),
			PublishedBy: userRef(p), PublishedAt: now}); err != nil {
			return err
		}
		if s.Translations != nil {
			if err := s.Translations.Publish(ctx, q, ws, id, int(version)); err != nil {
				return err
			}
		}
		if err := q.TxnTemplatePublish(ctx, store.TxnTemplatePublishParams{WorkspaceID: ws.ID, ID: id, Version: &version, Now: now, LiveBy: userRef(p), Variables: mustJSON(vars)}); err != nil {
			return err
		}
		if len(in.MakeOptional) > 0 {
			// The draft carries the fix too, so it matches the new version.
			if err := q.TxnTemplateSaveDraft(ctx, store.TxnTemplateSaveDraftParams{WorkspaceID: ws.ID, ID: id, Subject: t.Subject, Preheader: t.Preheader, EditorMode: t.EditorMode,
				Mjml: t.Mjml, Html: t.Html, PlainText: t.PlainText, Variables: mustJSON(vars), TestData: t.TestData, SenderID: t.SenderID, ReplyTo: t.ReplyTo,
				TrackOpens: t.TrackOpens, TrackClicks: t.TrackClicks, DraftRevision: t.DraftRevision + 1, DraftHash: hash, Now: now, SavedBy: userRef(p)}); err != nil {
				return err
			}
		}
		if s.Gate != nil {
			if err := s.Gate.Published(ctx, q, req, int(version)); err != nil {
				return err
			}
		}
		meta := map[string]any{"summary": change, "version": version}
		if len(pv.Breaking.Breaking) > 0 {
			meta["breaking"] = breakingSummary(i18n.Default, pv.Breaking, pv.NextVersion)
		}
		var prev *int
		if t.LiveVersion != nil {
			v := int(*t.LiveVersion)
			prev = &v
		}
		out.Version, out.Previous = int(version), prev
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.template_published", TargetType: "transactional_template", TargetID: id.String(),
			TargetName: fmt.Sprintf("%s v%d", t.Key, version), Before: map[string]any{"live_version": t.LiveVersion}, After: map[string]any{"live_version": version}, Metadata: meta})
	})
	if err != nil {
		return TransactionalPublished{}, err
	}
	out.Template, err = s.Template(ctx, ws, p, id.String())
	return out, err
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

// TransactionalRollbackInput publishes an older version again.
type TransactionalRollbackInput struct {
	Version int `json:"version" minimum:"1"`
	// Undo marks the rollback as the Undo of a publish (within seconds).
	Undo bool `json:"undo,omitempty"`
}

// Rollback makes an older version live again ("Publish again", Undo).
func (s *Service) Rollback(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in TransactionalRollbackInput) (TransactionalPublished, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalPublished{}, err
	}
	var out TransactionalPublished
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.TxnTemplateForUpdate(ctx, store.TxnTemplateForUpdateParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapTemplateErr(ctx, err)
		}
		v, err := q.TxnVersion(ctx, store.TxnVersionParams{TemplateID: id, Version: int32(in.Version)})
		if err != nil {
			return mapErr(ctx, err, "version")
		}
		if cur.LiveVersion != nil && *cur.LiveVersion == v.Version {
			return api.Conflict(say(ctx, "publish.alreadyLive", i18n.Args{"version": strconv.Itoa(int(v.Version))}))
		}
		req := PublishRequest{WorkspaceID: ws.ID, TemplateID: id, TemplateKey: cur.Key, Version: int(v.Version), Rollback: true, DraftHash: v.ContentHash, Principal: p}
		if s.Gate != nil {
			dec, err := s.Gate.Check(ctx, q, req)
			if err != nil {
				return err
			}
			if !dec.Allowed {
				return api.Err(http.StatusConflict, CodeApprovalRequired, orDefault(dec.Reason, say(ctx, "publish.needsApprovalShort", nil))).WithMeta("approvers", nonNilStrings(dec.Approvers))
			}
		}
		now := s.now()
		if err := q.TxnTemplateSetLive(ctx, store.TxnTemplateSetLiveParams{WorkspaceID: ws.ID, ID: id, Version: &v.Version, Now: now, LiveBy: userRef(p)}); err != nil {
			return err
		}
		if s.Gate != nil {
			if err := s.Gate.Published(ctx, q, req, int(v.Version)); err != nil {
				return err
			}
		}
		var prev *int
		if cur.LiveVersion != nil {
			pv := int(*cur.LiveVersion)
			prev = &pv
		}
		out.Version, out.Previous = int(v.Version), prev
		meta := map[string]any{"version": v.Version}
		if in.Undo {
			meta["undo"] = true
		}
		if prev != nil {
			meta["summary"] = fmt.Sprintf("v%d published again instead of v%d", v.Version, *prev)
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.template_rolled_back", TargetType: "transactional_template", TargetID: id.String(),
			TargetName: fmt.Sprintf("%s v%d", cur.Key, v.Version), Before: map[string]any{"live_version": cur.LiveVersion}, After: map[string]any{"live_version": v.Version}, Metadata: meta})
	})
	if err != nil {
		return TransactionalPublished{}, err
	}
	out.Template, err = s.Template(ctx, ws, p, id.String())
	return out, err
}

// Versions lists a template's published versions.
func (s *Service) Versions(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (TransactionalVersionList, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalVersionList{}, err
	}
	t, err := s.Q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return TransactionalVersionList{}, mapTemplateErr(ctx, err)
	}
	rows, err := s.Q.TxnVersions(ctx, id)
	if err != nil {
		return TransactionalVersionList{}, err
	}
	out := TransactionalVersionList{Items: make([]TransactionalVersion, 0, len(rows)), CanPublish: p.Can(authz.TransactionalPublish)}
	for _, r := range rows {
		out.Items = append(out.Items, TransactionalVersion{Version: int(r.Version), Live: t.LiveVersion != nil && *t.LiveVersion == r.Version, PublishedAt: r.PublishedAt,
			PublishedBy: r.PublishedByName, AvatarColor: r.PublishedByColor, Change: r.Change, Subject: r.Subject, SizeBytes: int(r.SizeBytes)})
	}
	return out, nil
}

// Version returns one published version with its content.
func (s *Service) Version(ctx context.Context, ws *store.Workspace, ref string, version int) (TransactionalVersionDetail, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalVersionDetail{}, err
	}
	t, err := s.Q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return TransactionalVersionDetail{}, mapTemplateErr(ctx, err)
	}
	v, err := s.Q.TxnVersion(ctx, store.TxnVersionParams{TemplateID: id, Version: int32(version)})
	if err != nil {
		return TransactionalVersionDetail{}, mapErr(ctx, err, "version")
	}
	d := TransactionalVersionDetail{
		TransactionalVersion: TransactionalVersion{Version: int(v.Version), Live: t.LiveVersion != nil && *t.LiveVersion == v.Version, PublishedAt: v.PublishedAt,
			PublishedBy: v.PublishedByName, Change: v.Change, Subject: v.Subject, SizeBytes: int(v.SizeBytes)},
		Preheader: v.Preheader, EditorMode: v.EditorMode, MJML: v.Mjml, HTML: v.Html, PlainText: v.PlainText, Variables: parseVariables(v.Variables),
		TestData: parseTestData(v.TestData), ReplyTo: v.ReplyTo, TrackOpens: v.TrackOpens, TrackClicks: v.TrackClicks, Note: v.Note,
	}
	if v.SenderID != nil {
		d.Sender = &TransactionalSenderRef{ID: v.SenderID.String(), Name: v.SenderName, Email: v.SenderEmail}
	}
	return d, nil
}
