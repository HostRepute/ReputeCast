package forms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// PublicCopyLabels is the state of a language's built-in labels.
type PublicCopyLabels struct {
	// Shipped is true when ReputeCast ships the labels in the language.
	Shipped bool       `json:"shipped"`
	Written int        `json:"written" doc:"Labels with text in the language (all of them when they ship)"`
	Total   int        `json:"total"`
	By      string     `json:"by" doc:"Who wrote them last"`
	At      *time.Time `json:"at,omitempty"`
}

// PublicPageLanguage is the public pages in one language.
type PublicPageLanguage struct {
	Code       string `json:"code" example:"es"`
	Name       string `json:"name" example:"Spanish"`
	Native     string `json:"native" example:"Español"`
	IsDefault  bool   `json:"isDefault"`
	IsFallback bool   `json:"isFallback"`
	// IsSource marks the language the pages are written in.
	IsSource bool   `json:"isSource"`
	Status   string `json:"status" enum:"live,incomplete,not_set_up"`
	// Missing counts the strings still to write, the built-in labels included.
	Missing int `json:"missing"`
	// Own counts the strings of the publisher's own text; Written, how many have their translation.
	Own     int              `json:"own"`
	Written int              `json:"written"`
	By      string           `json:"by"`
	At      *time.Time       `json:"at,omitempty"`
	Labels  PublicCopyLabels `json:"labels"`
}

// PublicPageCopyString is one string of the pages' own text.
type PublicPageCopyString struct {
	Key    string `json:"key" example:"subscribe.heading"`
	Page   string `json:"page" enum:"subscribe,confirm,preferences,unsubscribe,lists,privacy"`
	ListID string `json:"listId,omitempty"`
	// Source is the text in the pages' own language; Text its translation,
	// empty while it isn't written.
	Source string `json:"source"`
	Text   string `json:"text"`
	// Optional strings may stay as they are: a list's name, the privacy link.
	Optional bool `json:"optional"`
}

// PublicPageCopy is the pages' copy: every language's status and, for the
// language asked for, its strings.
type PublicPageCopy struct {
	CopyLanguage string                 `json:"copyLanguage" doc:"The language the pages are written in"`
	Default      string                 `json:"default"`
	Fallback     string                 `json:"fallback"`
	Languages    []PublicPageLanguage   `json:"languages"`
	Language     string                 `json:"language" doc:"The language of strings (?language=)"`
	Strings      []PublicPageCopyString `json:"strings"`
}

// PublicPageCopyInput saves strings of the pages' own text in a language.
type PublicPageCopyInput struct {
	// Strings are translations by key; an empty text removes one. Keys left
	// out keep their text.
	Strings map[string]string `json:"strings" maxProperties:"500"`
}

// PublicPageLabel is a built-in label in a language.
type PublicPageLabel struct {
	Key    string `json:"key" example:"form.button"`
	Source string `json:"source" doc:"The English text"`
	Note   string `json:"note,omitempty"`
	// Arguments are what the text may use: {publisher}, <b>…
	Arguments []string `json:"arguments"`
	// Text is the label in the language: ReputeCast's translation when it
	// ships the language, else the workspace's ("" until written).
	Text string `json:"text"`
}

// PublicPageLabels is the built-in labels of a language.
type PublicPageLabels struct {
	PublicCopyLabels
	Language string            `json:"language"`
	Labels   []PublicPageLabel `json:"labels"`
}

// PublicPageLabelsInput saves labels a workspace writes for a language
// ReputeCast doesn't ship.
type PublicPageLabelsInput struct {
	// Labels are texts by key; an empty text removes one. Keys left out keep their text.
	Labels map[string]string `json:"labels" maxProperties:"500"`
}

// SignupFormCopyText is a form's own text in one language.
type SignupFormCopyText struct {
	Heading     string `json:"heading" maxLength:"120"`
	Description string `json:"description" maxLength:"500"`
	Button      string `json:"button" maxLength:"40"`
	ConsentText string `json:"consentText" maxLength:"600"`
	AfterSubmit string `json:"afterSubmit" maxLength:"300" doc:"What the form says after a signup"`
	// Fields are the labels of the fields whose label is the publisher's text, by field key.
	Fields map[string]string `json:"fields" maxProperties:"20"`
}

// SignupFormCopyField is a field whose label is the publisher's text.
type SignupFormCopyField struct {
	Key    string `json:"key"`
	Source string `json:"source" doc:"Its label in the form's own language"`
}

// SignupFormConsentEvidence is the latest signup that agreed to a
// language's consent text.
type SignupFormConsentEvidence struct {
	ContactID string    `json:"contactId"`
	Name      string    `json:"name"`
	At        time.Time `json:"at"`
	Version   string    `json:"version" example:"v1"`
}

// SignupFormCopyLanguage is a form in one language.
type SignupFormCopyLanguage struct {
	Code      string `json:"code" example:"es"`
	Name      string `json:"name" example:"Spanish"`
	Native    string `json:"native" example:"Español"`
	IsDefault bool   `json:"isDefault"`
	// IsSource marks the language the form is written in.
	IsSource bool   `json:"isSource"`
	Status   string `json:"status" enum:"live,incomplete,not_set_up"`
	// Missing counts the strings still to write, the built-in labels included;
	// Written how many of the form's Total strings have text.
	Missing int                `json:"missing"`
	Written int                `json:"written"`
	Total   int                `json:"total"`
	By      string             `json:"by"`
	At      *time.Time         `json:"at,omitempty"`
	Copy    SignupFormCopyText `json:"copy"`
	// ConsentVersion is this language's version of the consent text (0
	// before it has one); ConsentFor the version of the form's own consent
	// text it was written or last confirmed for.
	ConsentVersion int  `json:"consentVersion"`
	ConsentFor     int  `json:"consentFor"`
	ConsentReview  bool `json:"consentReview" doc:"The form's own consent text changed since: Review the translation"`
	// Labels are the language's built-in labels; Evidence the latest signup
	// that agreed to its consent text.
	Labels   PublicCopyLabels           `json:"labels"`
	Evidence *SignupFormConsentEvidence `json:"evidence,omitempty"`
}

// SignupFormCopy is a form's copy in every workspace language.
type SignupFormCopy struct {
	FormID string `json:"formId"`
	// LanguageMode says which language the form shows: the page it's on
	// (its lang), the browser, or always FixedLanguage.
	LanguageMode  string `json:"languageMode" enum:"page,browser,fixed"`
	FixedLanguage string `json:"fixedLanguage"`
	CopyLanguage  string `json:"copyLanguage" doc:"The language the form is written in"`
	Fallback      string `json:"fallback" doc:"The language the form shows where one isn't Live"`
	// ConsentVersion is the form's own consent text's version, with who changed it last.
	ConsentVersion   int                      `json:"consentVersion"`
	ConsentChangedBy string                   `json:"consentChangedBy"`
	ConsentChangedAt *time.Time               `json:"consentChangedAt,omitempty"`
	Fields           []SignupFormCopyField    `json:"fields"`
	Languages        []SignupFormCopyLanguage `json:"languages"`
}

// SignupFormLanguageInput picks which language a form shows.
type SignupFormLanguageInput struct {
	Mode  string `json:"mode" enum:"page,browser,fixed"`
	Fixed string `json:"fixed,omitempty" doc:"The language of a form always shown in one (mode fixed)"`
}

func refusedLanguage(code string) error {
	name := languages.NameOf(code)
	if name == "" {
		name = code
	}
	return api.Invalid("path.language", fmt.Sprintf("%s isn't one of your languages. Add it in Settings › General first.", name))
}

// labelsState is the state of a language's built-in labels.
func labelsState(l *pageLanguage) PublicCopyLabels {
	out := PublicCopyLabels{Shipped: pagelabels.Shipped(l.Code), Total: pagelabels.Count()}
	if out.Shipped {
		out.Written = out.Total
		return out
	}
	out.Written = l.LabelsWritten
	if l.row != nil {
		out.By, out.At = l.row.LabelsByName, l.row.LabelsAt
	}
	return out
}

func (pc *PagesCopy) view() []PublicPageLanguage {
	out := []PublicPageLanguage{}
	for _, code := range pc.order {
		l := pc.langs[code]
		meta, _ := languages.Lookup(code)
		v := PublicPageLanguage{Code: code, Name: meta.Name, Native: meta.Native, IsDefault: code == pc.Languages.Default, IsFallback: code == pc.Languages.Fallback,
			IsSource: l.Source, Status: l.Status, Missing: l.Missing, Own: l.Own, Written: l.Written, Labels: labelsState(l)}
		if !l.Source {
			_, v.By, v.At = l.writer()
		}
		out = append(out, v)
	}
	return out
}

// PageCopy returns the pages' copy in every language, with the strings of
// one when language is set. It refreshes the statuses Settings › General
// shows.
func (s *Service) PageCopy(ctx context.Context, ws *store.Workspace, language string) (PublicPageCopy, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return PublicPageCopy{}, err
	}
	pc, err := s.loadPagesCopy(ctx, s.Q, ws, set)
	if err != nil {
		return PublicPageCopy{}, err
	}
	if err := s.storePageCopyStatus(ctx, s.Q, ws, pc); err != nil {
		s.log().WarnContext(ctx, "store page copy status", "err", err)
	}
	out := PublicPageCopy{CopyLanguage: pc.CopyLanguage, Default: pc.Languages.Default, Fallback: pc.Languages.Fallback, Languages: pc.view(), Strings: []PublicPageCopyString{}}
	if language == "" {
		return out, nil
	}
	code, err := workspaceLanguage(pc.Languages, language)
	if err != nil && languages.Base(language) != pc.CopyLanguage {
		return out, refusedLanguage(code)
	}
	if code == "" {
		code = languages.Base(language)
	}
	out.Language = code
	l := pc.langs[code]
	for _, ps := range pc.strings {
		v := PublicPageCopyString{Key: ps.Key, Page: ps.Page, ListID: ps.List, Source: ps.Source, Optional: ps.Optional}
		if l != nil && !l.Source {
			v.Text = l.strings[ps.Key]
		}
		out.Strings = append(out.Strings, v)
	}
	return out, nil
}

// storePageCopyStatus keeps workspace_languages' page copy status (what
// Settings › General and Send Readiness read) in step with the copy.
func (s *Service) storePageCopyStatus(ctx context.Context, q *store.Queries, ws *store.Workspace, pc *PagesCopy) error {
	rows, err := q.LanguagesList(ctx, ws.ID)
	if err != nil {
		return err
	}
	ls := &languages.Service{Q: q}
	for _, r := range rows {
		l := pc.langs[r.Code]
		if r.RemovedAt != nil || l == nil {
			continue
		}
		by, _, at := l.writer()
		changed := r.PageCopyStatus != l.Status || int(r.PageCopyMissing) != l.Missing
		if !l.Source && at != nil && (r.PageCopyAt == nil || !r.PageCopyAt.Equal(*at)) {
			changed = true
		}
		if !changed {
			continue
		}
		when := s.now()
		if at != nil && !l.Source {
			when = *at
		} else {
			by = nil
		}
		if err := ls.SetPageCopy(ctx, q, ws.ID, r.Code, l.Status, l.Missing, by, when); err != nil {
			return err
		}
	}
	return nil
}

// syncPageCopy recomputes and stores the statuses inside a change's
// transaction.
func (s *Service) syncPageCopy(ctx context.Context, q *store.Queries, ws *store.Workspace) error {
	set, err := s.settingsQ(ctx, q, ws)
	if err != nil {
		return err
	}
	pc, err := s.loadPagesCopy(ctx, q, ws, set)
	if err != nil {
		return err
	}
	return s.storePageCopyStatus(ctx, q, ws, pc)
}

// settingsQ reads the page settings through q (inside a transaction).
func (s *Service) settingsQ(ctx context.Context, q *store.Queries, ws *store.Workspace) (*Settings, error) {
	if err := q.PublicSettingsEnsure(ctx, ws.ID); err != nil {
		return nil, err
	}
	row, err := q.PublicSettingsGet(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	raw, err := q.FormsWorkspaceBranding(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	return &Settings{Workspace: ws, row: row, Branding: brandingOf(ws, raw), Pages: pagesOf(row.Pages)}, nil
}

// SavePageCopy stores strings of the pages' own text in a language.
func (s *Service) SavePageCopy(ctx context.Context, ws *store.Workspace, language string, in PublicPageCopyInput) (PublicPageCopy, error) {
	var code string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PageCopyLock(ctx, ws.ID); err != nil {
			return err
		}
		set, err := s.settingsQ(ctx, q, ws)
		if err != nil {
			return err
		}
		pc, err := s.loadPagesCopy(ctx, q, ws, set)
		if err != nil {
			return err
		}
		if code, err = workspaceLanguage(pc.Languages, language); err != nil {
			return refusedLanguage(code)
		}
		if code == pc.CopyLanguage {
			return api.Invalid("path.language", "The pages are written in "+languages.NameOf(code)+": change that text on each page's tab.")
		}
		known := map[string]pageString{}
		for _, ps := range pc.strings {
			known[ps.Key] = ps
		}
		l := pc.langs[code]
		before := map[string]string{}
		for k, v := range l.strings {
			before[k] = v
		}
		after := map[string]string{}
		for k, v := range before {
			if _, ok := known[k]; ok {
				after[k] = v
			}
		}
		keys := make([]string, 0, len(in.Strings))
		for k := range in.Strings {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var changed []string
		for _, k := range keys {
			ps, ok := known[k]
			if !ok {
				return api.Invalid("body.strings."+k, "The pages have no text to translate there any more. Reload the page.")
			}
			v := strings.TrimSpace(in.Strings[k])
			if problem := checkPageString(ps, v); problem != "" {
				return api.Invalid("body.strings."+k, problem)
			}
			if after[k] == v {
				continue
			}
			if v == "" {
				delete(after, k)
			} else {
				after[k] = v
			}
			changed = append(changed, k)
		}
		if len(changed) == 0 {
			return nil
		}
		raw, _ := json.Marshal(after)
		if err := q.PageCopySetPages(ctx, store.PageCopySetPagesParams{WorkspaceID: ws.ID, Language: code, Pages: raw, By: actor(ctx), At: s.now()}); err != nil {
			return err
		}
		if err := s.syncPageCopy(ctx, q, ws); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page_copy.update", TargetType: "language", TargetID: code, TargetName: languages.NameOf(code),
			Before: map[string]any{"strings": pick(before, changed)}, After: map[string]any{"strings": pick(after, changed)},
			Metadata: map[string]any{"language": code, "changed": changed}})
	})
	if err != nil {
		return PublicPageCopy{}, err
	}
	return s.PageCopy(ctx, ws, code)
}

func pick(m map[string]string, keys []string) map[string]string {
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}

// checkPageString checks a translation: the privacy link is a web address,
// the rest fit the page. It returns what to fix, or "".
func checkPageString(ps pageString, v string) string {
	if len(v) > 600 {
		return "Keep it under 600 characters."
	}
	if ps.Key == "privacyUrl" && v != "" {
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return "Use a full address like https://acme.io/es/privacidad."
		}
	}
	return ""
}

// PageLabels returns a language's built-in labels.
func (s *Service) PageLabels(ctx context.Context, ws *store.Workspace, language string) (PublicPageLabels, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return PublicPageLabels{}, err
	}
	pc, err := s.loadPagesCopy(ctx, s.Q, ws, set)
	if err != nil {
		return PublicPageLabels{}, err
	}
	code, err := workspaceLanguage(pc.Languages, language)
	if err != nil && languages.Base(language) != pc.CopyLanguage {
		return PublicPageLabels{}, refusedLanguage(code)
	}
	if code == "" {
		code = languages.Base(language)
	}
	return labelsView(pc.langs[code]), nil
}

func labelsView(l *pageLanguage) PublicPageLabels {
	out := PublicPageLabels{PublicCopyLabels: labelsState(l), Language: l.Code, Labels: []PublicPageLabel{}}
	for _, lb := range pagelabels.All() {
		v := PublicPageLabel{Key: lb.Key, Source: lb.Source, Note: lb.Note, Arguments: pagelabels.Arguments(lb.Key)}
		if out.Shipped {
			v.Text = l.Labels.Text(lb.Key, nil)
		} else {
			v.Text = l.labels[lb.Key]
		}
		out.Labels = append(out.Labels, v)
	}
	return out
}

// SavePageLabels stores built-in labels a workspace writes for a language
// ReputeCast doesn't ship.
func (s *Service) SavePageLabels(ctx context.Context, ws *store.Workspace, language string, in PublicPageLabelsInput) (PublicPageLabels, error) {
	var out PublicPageLabels
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PageCopyLock(ctx, ws.ID); err != nil {
			return err
		}
		set, err := s.settingsQ(ctx, q, ws)
		if err != nil {
			return err
		}
		pc, err := s.loadPagesCopy(ctx, q, ws, set)
		if err != nil {
			return err
		}
		code, err := workspaceLanguage(pc.Languages, language)
		if err != nil {
			return refusedLanguage(code)
		}
		if pagelabels.Shipped(code) {
			return api.Conflict("ReputeCast ships the " + languages.NameOf(code) + " labels, so they're ready and can't be changed here.")
		}
		l := pc.langs[code]
		before := map[string]string{}
		for k, v := range l.labels {
			if pagelabels.Known(k) {
				before[k] = v
			}
		}
		after := map[string]string{}
		for k, v := range before {
			after[k] = v
		}
		keys := make([]string, 0, len(in.Labels))
		for k := range in.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var changed []string
		for _, k := range keys {
			if !pagelabels.Known(k) {
				return api.Invalid("body.labels."+k, "That isn't one of the built-in labels. Reload the page.")
			}
			v := strings.TrimSpace(in.Labels[k])
			if len(v) > 600 {
				return api.Invalid("body.labels."+k, "Keep it under 600 characters.")
			}
			if v != "" {
				if err := pagelabels.Check(k, v); err != nil {
					return api.Invalid("body.labels."+k, labelProblem(k, err))
				}
			}
			if after[k] == v {
				continue
			}
			if v == "" {
				delete(after, k)
			} else {
				after[k] = v
			}
			changed = append(changed, k)
		}
		if len(changed) > 0 {
			raw, _ := json.Marshal(after)
			if err := q.PageCopySetLabels(ctx, store.PageCopySetLabelsParams{WorkspaceID: ws.ID, Language: code, Labels: raw, By: actor(ctx), At: s.now()}); err != nil {
				return err
			}
			if err := s.syncPageCopy(ctx, q, ws); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page_copy.labels", TargetType: "language", TargetID: code, TargetName: languages.NameOf(code),
				Before: map[string]any{"labels": pick(before, changed)}, After: map[string]any{"labels": pick(after, changed)},
				Metadata: map[string]any{"language": code, "changed": changed, "written": len(after), "total": pagelabels.Count()}}); err != nil {
				return err
			}
		}
		set2, err := s.settingsQ(ctx, q, ws)
		if err != nil {
			return err
		}
		pc2, err := s.loadPagesCopy(ctx, q, ws, set2)
		if err != nil {
			return err
		}
		out = labelsView(pc2.langs[code])
		return nil
	})
	return out, err
}

// labelProblem says what's wrong with text written for a label, and how to fix it.
func labelProblem(key string, err error) string {
	var ae *pagelabels.ArgError
	if errors.As(err, &ae) {
		if len(ae.Allowed) == 0 {
			return fmt.Sprintf("“%s” isn't in the English label, which has no placeholders. Write the text without it.", display(ae.Name))
		}
		return fmt.Sprintf("“%s” isn't in the English label. Use only %s.", display(ae.Name), strings.Join(pagelabels.Arguments(key), ", "))
	}
	return "Check the braces { } and tags < > of this text: one isn't closed."
}

func display(name string) string {
	if strings.HasPrefix(name, "<") {
		return name
	}
	return "{" + name + "}"
}

// ---- Forms ----

// FormCopy returns a form's copy in every workspace language.
func (s *Service) FormCopy(ctx context.Context, ws *store.Workspace, id string) (SignupFormCopy, error) {
	f, err := s.form(ctx, s.Q, ws, id)
	if err != nil {
		return SignupFormCopy{}, err
	}
	return s.formCopyView(ctx, s.Q, ws, f)
}

func (s *Service) formCopyView(ctx context.Context, q *store.Queries, ws *store.Workspace, f store.Form) (SignupFormCopy, error) {
	set, err := s.settingsQ(ctx, q, ws)
	if err != nil {
		return SignupFormCopy{}, err
	}
	c := ConfigOf(f)
	fc, err := s.loadFormCopy(ctx, q, ws, set, c)
	if err != nil {
		return SignupFormCopy{}, err
	}
	customs, err := q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return SignupFormCopy{}, err
	}
	out := SignupFormCopy{FormID: f.ID.String(), LanguageMode: c.LanguageMode, FixedLanguage: c.FixedLanguage, CopyLanguage: c.CopyLanguage, Fallback: fc.Fallback(),
		ConsentVersion: c.ConsentVersion, ConsentChangedAt: f.ConsentChangedAt, Fields: []SignupFormCopyField{}, Languages: []SignupFormCopyLanguage{}}
	if f.ConsentChangedBy != nil {
		if name, err := q.PageCopyUserName(ctx, *f.ConsentChangedBy); err == nil {
			out.ConsentChangedBy = name
		}
	}
	for _, fd := range c.Fields {
		if fieldNeedsLabel(fd) {
			out.Fields = append(out.Fields, SignupFormCopyField{Key: fd.Key, Source: FieldLabel(fd, customs)})
		}
	}
	codes := slices.Clone(fc.Pages.Languages.Languages)
	if !slices.Contains(codes, c.CopyLanguage) {
		codes = append([]string{c.CopyLanguage}, codes...)
	}
	for _, code := range codes {
		meta, _ := languages.Lookup(code)
		st, missing, written := fc.Status(code)
		v := SignupFormCopyLanguage{Code: code, Name: meta.Name, Native: meta.Native, IsDefault: code == fc.Pages.Languages.Default, IsSource: code == c.CopyLanguage,
			Status: st, Missing: missing, Written: written, Total: len(fc.strings), Copy: SignupFormCopyText{Fields: map[string]string{}}}
		if l := fc.Pages.lang(code); l != nil {
			v.Labels = labelsState(l)
		}
		if v.IsSource {
			v.Copy = SignupFormCopyText{Heading: c.Content.Heading, Description: c.Content.Description, Button: c.Content.Button, ConsentText: c.ConsentText,
				AfterSubmit: c.Content.AfterSubmitText, Fields: map[string]string{}}
			for _, fd := range c.Fields {
				if fieldNeedsLabel(fd) && fd.Label != "" {
					v.Copy.Fields[fd.Key] = fd.Label
				}
			}
			v.ConsentVersion, v.ConsentFor = c.ConsentVersion, c.ConsentVersion
		} else if t := fc.tr[code]; t != nil {
			v.Copy = SignupFormCopyText{Heading: t.Doc.Heading, Description: t.Doc.Description, Button: t.Doc.Button, ConsentText: t.Consent, AfterSubmit: t.Doc.AfterSubmit,
				Fields: map[string]string{}}
			for k, lbl := range t.Doc.Fields {
				v.Copy.Fields[k] = lbl
			}
			v.ConsentVersion, v.ConsentFor, v.ConsentReview = t.ConsentVersion, t.ConsentFor, fc.ConsentReview(code)
			at := t.At
			v.By, v.At = t.ByName, &at
		}
		if v.ConsentVersion > 0 && !c.Builtin {
			ev, err := q.FormConsentEvidence(ctx, store.FormConsentEvidenceParams{WorkspaceID: ws.ID, Source: c.Source(), Language: code})
			if err == nil {
				v.Evidence = &SignupFormConsentEvidence{ContactID: ev.ContactID.String(), Name: ev.Name, At: ev.OccurredAt, Version: ev.ConsentVersion}
			} else if !isNoRows(err) {
				return out, err
			}
		}
		out.Languages = append(out.Languages, v)
	}
	return out, nil
}

// SaveFormCopy stores a form's own text in a language: in its copy
// language that is the form's own content and consent text (a new consent
// version when the words change); in another, its copy there, whose
// consent text gets its own version.
func (s *Service) SaveFormCopy(ctx context.Context, ws *store.Workspace, id, language string, in SignupFormCopyText) (SignupFormCopy, error) {
	var out SignupFormCopy
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		c := ConfigOf(f)
		st, err := (&languages.Service{Q: q}).Settings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		code, err := workspaceLanguage(st, language)
		if err != nil && languages.Base(language) != c.CopyLanguage {
			return refusedLanguage(code)
		}
		if code == "" {
			code = languages.Base(language)
		}
		text := SignupFormCopyText{Heading: strings.TrimSpace(in.Heading), Description: strings.TrimSpace(in.Description), Button: strings.TrimSpace(in.Button),
			ConsentText: strings.TrimSpace(in.ConsentText), AfterSubmit: strings.TrimSpace(in.AfterSubmit), Fields: map[string]string{}}
		for k, v := range in.Fields {
			fd, ok := c.Field(k)
			if !ok || !fieldNeedsLabel(fd) {
				return api.Invalid("body.fields."+k, "The form has no field there that takes a label. Reload the page.")
			}
			if v = strings.TrimSpace(v); len(v) > 80 {
				return api.Invalid("body.fields."+k, "Keep a field's label under 80 characters.")
			} else if v != "" {
				text.Fields[k] = v
			}
		}
		if code == c.CopyLanguage {
			if text.Button == "" {
				return api.Invalid("body.button", "Name the button: visitors press it to sign up.")
			}
			return s.saveOwnCopy(ctx, q, ws, f, c, code, text, &out)
		}
		return s.saveTranslation(ctx, q, ws, f, c, code, text, &out)
	})
	return out, err
}

func copySnapshot(t SignupFormCopyText, version int) map[string]any {
	return map[string]any{"heading": t.Heading, "description": t.Description, "button": t.Button, "consentText": t.ConsentText, "consentVersion": version,
		"afterSubmit": t.AfterSubmit, "fields": t.Fields}
}

func (s *Service) saveOwnCopy(ctx context.Context, q *store.Queries, ws *store.Workspace, f store.Form, c Config, code string, text SignupFormCopyText, out *SignupFormCopy) error {
	var content contentDoc
	_ = json.Unmarshal(f.Content, &content)
	beforeCopy := SignupFormCopyText{Heading: content.Heading, Description: content.Description, Button: c.Content.Button, ConsentText: f.ConsentText,
		AfterSubmit: content.AfterSubmitText, Fields: map[string]string{}}
	content.Heading, content.Description, content.Button, content.AfterSubmitText = text.Heading, text.Description, text.Button, text.AfterSubmit
	fields := slices.Clone(c.Fields)
	for i, fd := range fields {
		if fd.Label != "" {
			beforeCopy.Fields[fd.Key] = fd.Label
		}
		if fieldNeedsLabel(fd) {
			fields[i].Label = text.Fields[fd.Key]
		}
	}
	version := int(f.ConsentVersion)
	if text.ConsentText != f.ConsentText && f.ConsentText != "" {
		version++
	}
	if version < 1 {
		version = 1
	}
	contentJSON, _ := json.Marshal(content)
	fieldsJSON, _ := json.Marshal(fields)
	after, err := q.FormSetCopy(ctx, store.FormSetCopyParams{WorkspaceID: ws.ID, ID: f.ID, Content: contentJSON, Fields: fieldsJSON, ConsentText: text.ConsentText,
		ConsentVersion: int32(version), By: actor(ctx), At: s.now()})
	if err != nil {
		return err
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.copy_update", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
		Before: copySnapshot(beforeCopy, int(f.ConsentVersion)), After: copySnapshot(text, version), Metadata: map[string]any{"language": code}}); err != nil {
		return err
	}
	s.forgetHosts()
	v, err := s.formCopyView(ctx, q, ws, after)
	*out = v
	return err
}

func (s *Service) saveTranslation(ctx context.Context, q *store.Queries, ws *store.Workspace, f store.Form, c Config, code string, text SignupFormCopyText, out *SignupFormCopy) error {
	rows, err := q.FormCopyList(ctx, f.ID)
	if err != nil {
		return err
	}
	var prev *store.FormCopyListRow
	for i := range rows {
		if rows[i].Language == code {
			prev = &rows[i]
		}
	}
	before := SignupFormCopyText{Fields: map[string]string{}}
	version, consentFor := 0, 0
	if prev != nil {
		var d formCopyDoc
		_ = json.Unmarshal(prev.Content, &d)
		before = SignupFormCopyText{Heading: d.Heading, Description: d.Description, Button: d.Button, ConsentText: prev.ConsentText, AfterSubmit: d.AfterSubmit, Fields: d.Fields}
		if before.Fields == nil {
			before.Fields = map[string]string{}
		}
		version, consentFor = int(prev.ConsentVersion), int(prev.ConsentFor)
	}
	beforeVersion := version
	// A new or changed consent text is a new version in this language,
	// written for the form's current consent text.
	if text.ConsentText != before.ConsentText && text.ConsentText != "" {
		version++
		consentFor = c.ConsentVersion
	}
	doc := formCopyDoc{Heading: text.Heading, Description: text.Description, Button: text.Button, AfterSubmit: text.AfterSubmit, Fields: text.Fields}
	raw, _ := json.Marshal(doc)
	if err := q.FormCopyUpsert(ctx, store.FormCopyUpsertParams{FormID: f.ID, Language: code, Content: raw, ConsentText: text.ConsentText, ConsentVersion: int32(version),
		ConsentFor: int32(consentFor), By: actor(ctx), At: s.now()}); err != nil {
		return err
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.copy_update", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
		Before: copySnapshot(before, beforeVersion), After: copySnapshot(text, version), Metadata: map[string]any{"language": code}}); err != nil {
		return err
	}
	v, err := s.formCopyView(ctx, q, ws, f)
	*out = v
	return err
}

// ConfirmFormConsent records that a language's consent text is still right
// for the form's current consent text ("Spanish is still right").
func (s *Service) ConfirmFormConsent(ctx context.Context, ws *store.Workspace, id, language string) (SignupFormCopy, error) {
	var out SignupFormCopy
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		f, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		st, err := (&languages.Service{Q: q}).Settings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		code, err := workspaceLanguage(st, language)
		if err != nil {
			return refusedLanguage(code)
		}
		n, err := q.FormCopyConfirmConsent(ctx, store.FormCopyConfirmConsentParams{FormID: f.ID, Language: code, ConsentFor: f.ConsentVersion, By: actor(ctx), At: s.now()})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.Conflict("The " + languages.NameOf(code) + " consent text doesn't need a review: it's up to date with the form's.")
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.consent_reviewed", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
			Metadata: map[string]any{"language": code, "consentVersion": f.ConsentVersion}}); err != nil {
			return err
		}
		out, err = s.formCopyView(ctx, q, ws, f)
		return err
	})
	return out, err
}

// SetFormLanguage picks which language a form shows.
func (s *Service) SetFormLanguage(ctx context.Context, ws *store.Workspace, id string, in SignupFormLanguageInput) (SignupFormCopy, error) {
	var out SignupFormCopy
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := s.form(ctx, q, ws, id)
		if err != nil {
			return err
		}
		fixed := ""
		switch in.Mode {
		case LanguagePage, LanguageBrowser:
		case LanguageFixed:
			st, err := (&languages.Service{Q: q}).Settings(ctx, q, ws.ID)
			if err != nil {
				return err
			}
			code, err := workspaceLanguage(st, in.Fixed)
			if err != nil {
				if strings.TrimSpace(in.Fixed) == "" {
					return api.Invalid("body.fixed", "Pick the language the form always shows.")
				}
				return api.Invalid("body.fixed", fmt.Sprintf("%s isn't one of your languages. Add it in Settings › General first.", nzName(code, in.Fixed)))
			}
			fixed = code
		default:
			return api.Invalid("body.mode", "Pick page, browser or fixed.")
		}
		after, err := q.FormSetLanguage(ctx, store.FormSetLanguageParams{WorkspaceID: ws.ID, ID: before.ID, LanguageMode: in.Mode, FixedLanguage: fixed, By: actor(ctx), At: s.now()})
		if err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.language", TargetType: "form", TargetID: after.ID.String(), TargetName: after.Name,
			Before: map[string]any{"mode": before.LanguageMode, "fixed": before.FixedLanguage}, After: map[string]any{"mode": after.LanguageMode, "fixed": after.FixedLanguage}}); err != nil {
			return err
		}
		out, err = s.formCopyView(ctx, q, ws, after)
		return err
	})
	return out, err
}

func nzName(code, raw string) string {
	if n := languages.NameOf(code); n != "" {
		return n
	}
	return raw
}
