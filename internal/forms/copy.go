package forms

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// Page and form copy per language (design expansion M17, phase 2 brief
// §3.6). The public pages and each form are written in their copy language
// (the workspace's default when they were made). Every other workspace
// language gets its own copy: the publisher's text and, for a language
// ReputeCast doesn't ship them in, the built-in labels. A language's pages
// and forms are used only once its copy is complete ("Live"); until then
// its visitors get the fallback language's, whole.

// Copy statuses of a language's pages or a form.
const (
	CopyLive       = "live"
	CopyIncomplete = "incomplete"
	CopyNotSetUp   = "not_set_up"
)

// copyStatus is Live when nothing is missing, Not set up while nothing is
// written, Incomplete in between.
func copyStatus(missing, written int) string {
	switch {
	case missing == 0:
		return CopyLive
	case written == 0:
		return CopyNotSetUp
	}
	return CopyIncomplete
}

// Form language modes.
const (
	LanguagePage    = "page"
	LanguageBrowser = "browser"
	LanguageFixed   = "fixed"
)

// ---- The public pages ----

// The pages a string of the public pages' own text belongs to.
const (
	pageSubscribe   = "subscribe"
	pageConfirm     = "confirm"
	pagePreferences = "preferences"
	pageUnsubscribe = "unsubscribe"
	pageLists       = "lists"
	pagePrivacy     = "privacy"
)

// pageString is one string of the public pages' own text, in their copy
// language. List names and the privacy link may stay as they are; every
// other string needs its translation.
type pageString struct {
	Key, Page, List, Source string
	Optional                bool
}

// pageStrings lists the public pages' own text: what each language's copy
// translates. Empty texts have nothing to translate.
func pageStrings(set *Settings, lists []audience.PublicList) []pageString {
	p := set.Pages
	var out []pageString
	add := func(key, page, list, text string, optional bool) {
		if strings.TrimSpace(text) != "" {
			out = append(out, pageString{Key: key, Page: page, List: list, Source: text, Optional: optional})
		}
	}
	add("subscribe.heading", pageSubscribe, "", p.Subscribe.Heading, false)
	add("subscribe.description", pageSubscribe, "", p.Subscribe.Description, false)
	add("subscribe.footnote", pageSubscribe, "", p.Subscribe.Footnote, false)
	for _, l := range lists {
		add("subscribe.list."+l.ID.String(), pageSubscribe, l.ID.String(), p.Subscribe.ListText[l.ID.String()], false)
	}
	for _, l := range lists {
		id := l.ID.String()
		add("confirm.list."+id, pageConfirm, id, p.Confirm.ListText[id], false)
		add("confirm.offer."+id, pageConfirm, id, p.Confirm.OfferText[id], false)
	}
	for _, l := range lists {
		add("preferences.list."+l.ID.String(), pagePreferences, l.ID.String(), p.Preferences.ListText[l.ID.String()], false)
	}
	if p.Unsubscribe.Fewer {
		add("unsubscribe.fewerTitle", pageUnsubscribe, "", p.Unsubscribe.FewerTitle, false)
		add("unsubscribe.fewerText", pageUnsubscribe, "", p.Unsubscribe.FewerText, false)
		add("unsubscribe.fewerButton", pageUnsubscribe, "", p.Unsubscribe.FewerButton, false)
	}
	for _, l := range lists {
		if !l.Public {
			continue
		}
		add("list."+l.ID.String()+".name", pageLists, l.ID.String(), l.Name, true)
		add("list."+l.ID.String()+".description", pageLists, l.ID.String(), l.Description, false)
	}
	add("privacyUrl", pagePrivacy, "", set.Branding.PrivacyURL, true)
	return out
}

func decodeStrings(raw json.RawMessage) map[string]string {
	out := map[string]string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

// pageLanguage is the public pages in one language.
type pageLanguage struct {
	Code   string
	Source bool
	Status string
	// Missing counts the strings still to write: the publisher's own text
	// and the built-in labels.
	Missing       int
	Own, Written  int
	Labels        pagelabels.Set
	LabelsWritten int
	strings       map[string]string
	labels        map[string]string
	row           *store.PageCopyListRow
}

// writer is who changed the language's copy last, and when.
func (l *pageLanguage) writer() (*uuid.UUID, string, *time.Time) {
	if l.row == nil {
		return nil, "", nil
	}
	r := l.row
	if r.LabelsAt != nil && (r.PagesAt == nil || r.LabelsAt.After(*r.PagesAt)) {
		return r.LabelsBy, r.LabelsByName, r.LabelsAt
	}
	return r.PagesBy, r.PagesByName, r.PagesAt
}

// PagesCopy is a workspace's public pages in every language.
type PagesCopy struct {
	Languages languages.Settings
	// CopyLanguage is the language the pages are written in.
	CopyLanguage string
	strings      []pageString
	langs        map[string]*pageLanguage
	order        []string
}

// loadPagesCopy reads the pages' copy in every workspace language.
func (s *Service) loadPagesCopy(ctx context.Context, q *store.Queries, ws *store.Workspace, set *Settings) (*PagesCopy, error) {
	st, err := (&languages.Service{Q: q}).Settings(ctx, q, ws.ID)
	if err != nil {
		return nil, err
	}
	lists, err := s.Audience.PublicLists(ctx, q, ws)
	if err != nil {
		return nil, err
	}
	rows, err := q.PageCopyList(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	return pagesCopyOf(st, set, lists, rows), nil
}

func pagesCopyOf(st languages.Settings, set *Settings, lists []audience.PublicList, rows []store.PageCopyListRow) *PagesCopy {
	pc := &PagesCopy{Languages: st, CopyLanguage: set.CopyLanguage(), strings: pageStrings(set, lists), langs: map[string]*pageLanguage{}}
	byCode := map[string]*store.PageCopyListRow{}
	for i := range rows {
		byCode[rows[i].Language] = &rows[i]
	}
	codes := slices.Clone(st.Languages)
	if !slices.Contains(codes, pc.CopyLanguage) {
		codes = append(codes, pc.CopyLanguage)
	}
	for _, code := range codes {
		l := &pageLanguage{Code: code, Source: code == pc.CopyLanguage, row: byCode[code], strings: map[string]string{}, labels: map[string]string{}}
		if l.row != nil {
			l.strings, l.labels = decodeStrings(l.row.Pages), decodeStrings(l.row.Labels)
		}
		l.Labels = pagelabels.For(code, l.labels)
		if !pagelabels.Shipped(code) {
			for _, v := range l.labels {
				if strings.TrimSpace(v) != "" {
					l.LabelsWritten++
				}
			}
		}
		if l.Source {
			l.Status = CopyLive
		} else {
			for _, ps := range pc.strings {
				if ps.Optional {
					continue
				}
				l.Own++
				if strings.TrimSpace(l.strings[ps.Key]) != "" {
					l.Written++
				}
			}
			l.Missing = l.Own - l.Written + len(l.Labels.Missing())
			l.Status = copyStatus(l.Missing, l.Written+l.LabelsWritten)
		}
		pc.langs[code] = l
		pc.order = append(pc.order, code)
	}
	return pc
}

func (pc *PagesCopy) lang(code string) *pageLanguage {
	if pc == nil {
		return nil
	}
	return pc.langs[languages.Base(code)]
}

// Live reports whether the pages can be shown in a language.
func (pc *PagesCopy) Live(code string) bool {
	l := pc.lang(code)
	return l != nil && l.Status == CopyLive && (l.Source || pc.Languages.Offers(l.Code))
}

// Labels returns the built-in labels of a language.
func (pc *PagesCopy) Labels(code string) pagelabels.Set {
	if l := pc.lang(code); l != nil {
		return l.Labels
	}
	return pagelabels.For(code, nil)
}

// Text is a string of the pages' own text in a language: its translation,
// else the text in the copy language.
func (pc *PagesCopy) Text(code, key, source string) string {
	if l := pc.lang(code); l != nil && !l.Source {
		if v := strings.TrimSpace(l.strings[key]); v != "" {
			return v
		}
	}
	return source
}

// ListName is a list's name in a language.
func (pc *PagesCopy) ListName(code string, l audience.PublicList) string {
	return pc.Text(code, "list."+l.ID.String()+".name", l.Name)
}

// ListDescription is a list's description in a language.
func (pc *PagesCopy) ListDescription(code string, l audience.PublicList) string {
	return pc.Text(code, "list."+l.ID.String()+".description", l.Description)
}

// PrivacyURL is the privacy policy link of a language.
func (pc *PagesCopy) PrivacyURL(code, base string) string {
	return pc.Text(code, "privacyUrl", base)
}

// Pages are the pages' options with their own text in a language.
func (pc *PagesCopy) Pages(code string, p Pages) Pages {
	l := pc.lang(code)
	if l == nil || l.Source {
		return p
	}
	t := func(key, v string) string { return pc.Text(code, key, v) }
	out := p
	out.Subscribe.Heading = t("subscribe.heading", p.Subscribe.Heading)
	out.Subscribe.Description = t("subscribe.description", p.Subscribe.Description)
	out.Subscribe.Footnote = t("subscribe.footnote", p.Subscribe.Footnote)
	out.Subscribe.ListText = translateMap(p.Subscribe.ListText, func(id, v string) string { return t("subscribe.list."+id, v) })
	out.Confirm.ListText = translateMap(p.Confirm.ListText, func(id, v string) string { return t("confirm.list."+id, v) })
	out.Confirm.OfferText = translateMap(p.Confirm.OfferText, func(id, v string) string { return t("confirm.offer."+id, v) })
	out.Preferences.ListText = translateMap(p.Preferences.ListText, func(id, v string) string { return t("preferences.list."+id, v) })
	out.Unsubscribe.FewerTitle = t("unsubscribe.fewerTitle", p.Unsubscribe.FewerTitle)
	out.Unsubscribe.FewerText = t("unsubscribe.fewerText", p.Unsubscribe.FewerText)
	out.Unsubscribe.FewerButton = t("unsubscribe.fewerButton", p.Unsubscribe.FewerButton)
	return out
}

func translateMap(m map[string]string, t func(id, v string) string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = t(k, v)
	}
	return out
}

// LiveLanguages are the workspace languages the pages can be shown in, the
// default first.
func (pc *PagesCopy) LiveLanguages() []string {
	var out []string
	for _, code := range pc.Languages.Languages {
		if pc.Live(code) {
			out = append(out, code)
		}
	}
	return out
}

// Fallback is the language visitors get when theirs isn't Live: the
// workspace's fallback, else its default, else the pages' own language.
func (pc *PagesCopy) Fallback() string {
	for _, c := range []string{pc.Languages.Fallback, pc.Languages.Default} {
		if c != "" && pc.Live(c) {
			return c
		}
	}
	return pc.CopyLanguage
}

// Choose picks the language to show for a visitor's languages, in order of
// preference: the first that's a workspace language decides, its pages when
// they're Live, else the fallback's; with none of them, the default's.
// live says whether a language can be shown (the pages and, on a page with
// a form, the form too).
func (pc *PagesCopy) Choose(live func(string) bool, wants ...string) string {
	for _, w := range wants {
		code := languages.Base(w)
		if code == "" || !pc.Languages.Offers(code) {
			continue
		}
		if live(code) {
			return code
		}
		break
	}
	for _, c := range []string{pc.Languages.Fallback, pc.Languages.Default, pc.CopyLanguage} {
		if c != "" && live(c) {
			return c
		}
	}
	return pc.CopyLanguage
}

// ---- Forms ----

// formCopyDoc is a form's own text in a language other than its copy
// language (form_copy.content).
type formCopyDoc struct {
	Heading     string            `json:"heading,omitempty"`
	Description string            `json:"description,omitempty"`
	Button      string            `json:"button,omitempty"`
	AfterSubmit string            `json:"afterSubmit,omitempty"`
	Fields      map[string]string `json:"fields,omitempty"`
}

// formTranslation is a form's copy in one language.
type formTranslation struct {
	Doc            formCopyDoc
	Consent        string
	ConsentVersion int
	ConsentFor     int
	By             *uuid.UUID
	ByName         string
	At             time.Time
}

// formString is one string of a form's own text, in its copy language.
type formString struct{ Key, Source string }

// fieldNeedsLabel reports whether a field's label is the publisher's text:
// a custom field, or a built-in field given its own label.
func fieldNeedsLabel(f fieldDoc) bool {
	_, builtin := builtinFields[f.Key]
	return !builtin || f.Label != ""
}

// formStrings lists a form's own text: what each language's copy
// translates. The button and the after-submit message always need one;
// the others when the form has them.
func formStrings(c Config, customs []store.CustomField) []formString {
	var out []formString
	if c.Content.Heading != "" {
		out = append(out, formString{"heading", c.Content.Heading})
	}
	if c.Content.Description != "" {
		out = append(out, formString{"description", c.Content.Description})
	}
	out = append(out, formString{"button", c.Content.Button})
	if c.ConsentText != "" {
		out = append(out, formString{"consent", c.ConsentText})
	}
	if c.AfterSubmit != "redirect" {
		out = append(out, formString{"afterSubmit", c.afterSubmitSource()})
	}
	for _, f := range c.Fields {
		if fieldNeedsLabel(f) {
			out = append(out, formString{"field." + f.Key, FieldLabel(f, customs)})
		}
	}
	return out
}

// afterSubmitSource is what the form says after a signup in its copy
// language: the publisher's message, else the built-in one.
func (c Config) afterSubmitSource() string {
	if c.Content.AfterSubmitText != "" {
		return c.Content.AfterSubmitText
	}
	if c.DoubleOptIn {
		return c.Labels.Text("form.confirm.short", nil)
	}
	return c.Labels.Text("form.subscribed.short", i18n.Args{"publisher": c.publisher})
}

func (t *formTranslation) text(key string) string {
	if t == nil {
		return ""
	}
	switch key {
	case "heading":
		return t.Doc.Heading
	case "description":
		return t.Doc.Description
	case "button":
		return t.Doc.Button
	case "consent":
		return t.Consent
	case "afterSubmit":
		return t.Doc.AfterSubmit
	}
	if k, ok := strings.CutPrefix(key, "field."); ok {
		return t.Doc.Fields[k]
	}
	return ""
}

// FormCopy is a form's copy in every language, with the pages' (the
// labels, list names and privacy link come from there).
type FormCopy struct {
	Form    Config
	Pages   *PagesCopy
	strings []formString
	tr      map[string]*formTranslation
}

// loadFormCopy reads a form's copy in every language.
func (s *Service) loadFormCopy(ctx context.Context, q *store.Queries, ws *store.Workspace, set *Settings, c Config) (*FormCopy, error) {
	pc, err := s.loadPagesCopy(ctx, q, ws, set)
	if err != nil {
		return nil, err
	}
	customs, err := q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	var rows []store.FormCopyListRow
	if !c.Builtin {
		if rows, err = q.FormCopyList(ctx, c.ID); err != nil {
			return nil, err
		}
	}
	return formCopyOf(c, set, pc, customs, rows), nil
}

func formCopyOf(c Config, set *Settings, pc *PagesCopy, customs []store.CustomField, rows []store.FormCopyListRow) *FormCopy {
	c.publisher = set.Branding.PublisherName
	c.Labels = pc.Labels(c.CopyLanguage)
	fc := &FormCopy{Form: c, Pages: pc, tr: map[string]*formTranslation{}}
	fc.strings = formStrings(c, customs)
	for _, r := range rows {
		t := &formTranslation{Consent: r.ConsentText, ConsentVersion: int(r.ConsentVersion), ConsentFor: int(r.ConsentFor), By: r.UpdatedBy, ByName: r.UpdatedByName, At: r.UpdatedAt}
		_ = json.Unmarshal(r.Content, &t.Doc)
		fc.tr[r.Language] = t
	}
	return fc
}

// Status is the form's copy in a language: its status, the strings still
// to write (the form's own and the built-in labels) and how many are
// written.
func (fc *FormCopy) Status(code string) (status string, missing, written int) {
	code = languages.Base(code)
	if code == fc.Form.CopyLanguage {
		return CopyLive, 0, len(fc.strings)
	}
	t := fc.tr[code]
	for _, s := range fc.strings {
		if strings.TrimSpace(t.text(s.Key)) != "" {
			written++
		}
	}
	labels := fc.Pages.Labels(code)
	lmissing := len(labels.Missing())
	missing = len(fc.strings) - written + lmissing
	lwritten := 0
	if l := fc.Pages.lang(code); l != nil {
		lwritten = l.LabelsWritten
	}
	return copyStatus(missing, written+lwritten), missing, written
}

// Live reports whether the form can be shown in a language.
func (fc *FormCopy) Live(code string) bool {
	code = languages.Base(code)
	if code == fc.Form.CopyLanguage {
		return true
	}
	if !fc.Pages.Languages.Offers(code) {
		return false
	}
	st, _, _ := fc.Status(code)
	return st == CopyLive
}

// LiveLanguages are the languages the form can be shown in: the workspace's
// Live ones, the default first, and the form's own language.
func (fc *FormCopy) LiveLanguages() []string {
	var out []string
	for _, code := range fc.Pages.Languages.Languages {
		if fc.Live(code) {
			out = append(out, code)
		}
	}
	if !slices.Contains(out, fc.Form.CopyLanguage) {
		out = append(out, fc.Form.CopyLanguage)
	}
	return out
}

// Fallback is the language the form shows when the one asked for isn't
// Live: the workspace's fallback when the form is Live in it, else the
// form's own language.
func (fc *FormCopy) Fallback() string {
	for _, c := range []string{fc.Pages.Languages.Fallback, fc.Pages.Languages.Default} {
		if c != "" && fc.Live(c) {
			return c
		}
	}
	return fc.Form.CopyLanguage
}

// Pick is the language a submission or page shows the form in: the one
// asked for when the form is Live in it ("" is the form's own language,
// which a copied plain HTML form shows), else the fallback. A form always
// in one language shows it when it's Live.
func (fc *FormCopy) Pick(requested string) string {
	if fc.Form.LanguageMode == LanguageFixed && fc.Form.FixedLanguage != "" {
		requested = fc.Form.FixedLanguage
	}
	code := languages.Base(requested)
	if code == "" {
		return fc.Form.CopyLanguage
	}
	if fc.Live(code) {
		return code
	}
	return fc.Fallback()
}

// In is the form as it shows in a language: its copy there, the language's
// labels, list names and privacy link. A language the form isn't Live in
// shows the form in its own language.
func (fc *FormCopy) In(code string) Config {
	code = languages.Base(code)
	c := fc.Form
	c.Fields = slices.Clone(c.Fields)
	if code == "" || !fc.Live(code) {
		code = c.CopyLanguage
	}
	c.Language = code
	c.Labels = fc.Pages.Labels(code)
	c.PrivacyURL = fc.Pages.PrivacyURL(code, "")
	c.ListNames = map[uuid.UUID]string{}
	if code == c.CopyLanguage {
		return c
	}
	t := fc.tr[code]
	c.Content.Heading = t.Doc.Heading
	c.Content.Description = t.Doc.Description
	c.Content.Button = t.Doc.Button
	c.Content.AfterSubmitText = t.Doc.AfterSubmit
	// The footnote and the lists' label aren't translated: the lists'
	// label and the placeholders are the language's built-in labels.
	c.Content.Footnote = ""
	c.Content.ListsLabel = c.Labels.Text("form.lists", nil)
	c.Content.EmailPlaceholder = c.Labels.Text("form.emailPlaceholder", nil)
	c.Content.NamePlaceholder = c.Labels.Text("form.namePlaceholder", nil)
	c.ConsentText, c.ConsentVersion = t.Consent, t.ConsentVersion
	for i, f := range c.Fields {
		if v := t.Doc.Fields[f.Key]; v != "" {
			c.Fields[i].Label = v
		}
	}
	return c
}

// listNamesIn sets the names of the form's lists in its language.
func (fc *FormCopy) listNamesIn(c *Config, lists []audience.PublicList) {
	if c.ListNames == nil {
		c.ListNames = map[uuid.UUID]string{}
	}
	for _, l := range lists {
		c.ListNames[l.ID] = fc.Pages.ListName(c.Language, l)
	}
}

// ConsentReview reports whether a language's consent text was written for
// an older version of the form's own consent text.
func (fc *FormCopy) ConsentReview(code string) bool {
	t := fc.tr[languages.Base(code)]
	return t != nil && t.Consent != "" && t.ConsentFor < fc.Form.ConsentVersion
}

// ShownLanguage is the language a form was shown in, for its evidence and
// emails: the Config's language, else its copy language.
func (c Config) ShownLanguage() string {
	switch {
	case c.Language != "":
		return c.Language
	case c.CopyLanguage != "":
		return c.CopyLanguage
	}
	return "en"
}

// LocalizedForm returns the form as it shows in a language (see
// FormCopy.Pick): the hosted pages and the embed script use it.
func (s *Service) LocalizedForm(ctx context.Context, ws *store.Workspace, set *Settings, c Config, requested string) (Config, *FormCopy, error) {
	fc, err := s.loadFormCopy(ctx, s.Q, ws, set, c)
	if err != nil {
		return c, nil, err
	}
	out := fc.In(fc.Pick(requested))
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, fc, err
	}
	fc.listNamesIn(&out, lists)
	return out, fc, nil
}

// PagesCopyOf reads the public pages' copy in every language.
func (s *Service) PagesCopyOf(ctx context.Context, ws *store.Workspace, set *Settings) (*PagesCopy, error) {
	return s.loadPagesCopy(ctx, s.Q, ws, set)
}

// PhraseIndex finds phrase in text, ignoring letter case, and returns where
// it starts and ends in text itself (byte offsets), or -1, -1. The consent
// text links its words for the privacy policy this way, in every language.
func PhraseIndex(text, phrase string) (int, int) {
	n := utf8.RuneCountInString(phrase)
	if n == 0 {
		return -1, -1
	}
	for i := range text {
		j, count := i, 0
		for j < len(text) && count < n {
			_, size := utf8.DecodeRuneInString(text[j:])
			j += size
			count++
		}
		if count < n {
			break
		}
		if strings.EqualFold(text[i:j], phrase) {
			return i, j
		}
	}
	return -1, -1
}

// errLanguageNotOffered means a language isn't one of the workspace's.
var errLanguageNotOffered = errors.New("forms: not a workspace language")

// workspaceLanguage reads a language of the path: a code of one of the
// workspace's languages, regional codes included ("es-MX" is es).
func workspaceLanguage(st languages.Settings, raw string) (string, error) {
	code, ok := languages.Normalize(raw)
	if !ok {
		return "", errLanguageNotOffered
	}
	code = languages.Base(code)
	if !st.Offers(code) {
		return code, errLanguageNotOffered
	}
	return code, nil
}

// isNoRows reports pgx's "no rows".
func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
