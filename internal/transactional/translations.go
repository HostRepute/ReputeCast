package transactional

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Translations are a template's language versions (internal/languages):
// the draft endpoints take `language`, publishing publishes every version
// with the template's, and each send picks the version its recipient gets
// (the language asked for, else the linked contact's, else the fallback).
type Translations interface {
	// SourceLanguage is the language the template is written in.
	SourceLanguage(ctx context.Context, ws *store.Workspace, templateID uuid.UUID) (string, error)
	// Draft is a language version of the draft (404 when it has none).
	Draft(ctx context.Context, ws *store.Workspace, templateID uuid.UUID, lang string) (TranslationDraft, error)
	// SaveDraft stores a language version of the draft: its subject,
	// preheader and content (variables, test data and settings are the
	// template's).
	SaveDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, templateID uuid.UUID, lang string, in TransactionalDraftSave) (TransactionalDraftSaved, error)
	// Publishing lists the languages publishing the draft publishes, read
	// with q.
	Publishing(ctx context.Context, q *store.Queries, ws *store.Workspace, templateID uuid.UUID) ([]TransactionalPublishLanguage, error)
	// Publish publishes the draft's language versions with a new version
	// of the template, in the publish transaction.
	Publish(ctx context.Context, q *store.Queries, ws *store.Workspace, templateID uuid.UUID, version int) error
	// Changes lists the language versions the draft adds, changes or
	// removes since a published version, read with q.
	Changes(ctx context.Context, q *store.Queries, ws *store.Workspace, templateID uuid.UUID, version int) ([]TransactionalLanguageChange, error)
	// Choose picks the language version of a published template version a
	// message uses.
	Choose(ctx context.Context, q *store.Queries, ws *store.Workspace, templateID uuid.UUID, version int, requested string, contactID *uuid.UUID) (LanguageChoice, error)
	// Record stores the language a message was sent in, in the send's
	// transaction.
	Record(ctx context.Context, q *store.Queries, messageID uuid.UUID, ch LanguageChoice) error
}

// TranslationDraft is a language version of a template's draft.
type TranslationDraft struct {
	Language  string
	Subject   string
	Preheader string
	MJML      string
	HTML      string
	PlainText string
	// Revision is the version's own save counter (the editor's
	// baseRevision when it edits that language).
	Revision int
	SavedAt  time.Time
	SavedBy  string
	Hash     string
}

// LanguageChoice is the language version a message uses and why.
type LanguageChoice struct {
	// Language is the version's language.
	Language string
	// Reason is why: match (the contact's language, or the one asked for),
	// requested (asked for a language without a version), no_language,
	// not_offered, missing, draft or held.
	Reason string
	// Contact is the language code asked for, else the contact's.
	Contact string
	// Content replaces the template version's when the version isn't the
	// source; nil for the source.
	Content *TranslationText
}

// TranslationText is a published language version's text.
type TranslationText struct {
	Subject   string
	Preheader string
	MJML      string
	HTML      string
	Text      string
}

// TransactionalPublishLanguage is a language version publishing the draft
// publishes: the publish dialog says "Publishes English, Spanish and German
// together" and lists any outdated one.
type TransactionalPublishLanguage struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	// Status is source, up_to_date, outdated or draft. A draft is
	// published with the others but not sent: its readers get the
	// fallback until it is marked translated.
	Status string `json:"status" enum:"source,up_to_date,outdated,draft"`
	// Attention counts what changed in the source since it was translated.
	Attention int `json:"attention"`
	// Gets is the version its recipients get: itself, or the fallback for
	// a draft; GetsName names it.
	Gets     string `json:"gets"`
	GetsName string `json:"getsName"`
}

// TransactionalLanguageChange is a language version the draft adds,
// changes or removes since the live version (publishing one is a change).
type TransactionalLanguageChange struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	Change   string `json:"change" enum:"added,changed,removed"`
}

// languageChangeSummary names the language changes in the version
// history ("Spanish added · German changed").
func languageChangeSummary(changes []TransactionalLanguageChange) string {
	parts := make([]string, 0, len(changes))
	for _, c := range changes {
		parts = append(parts, c.Name+" "+c.Change)
	}
	return strings.Join(parts, " · ")
}

// withLanguage applies a language version's text to a version's content.
func (c content) withLanguage(t *TranslationText) content {
	if t == nil {
		return c
	}
	c.Subject, c.Preheader, c.MJML, c.HTML, c.Text = t.Subject, t.Preheader, t.MJML, t.HTML, t.Text
	return c
}

// translated reports whether lang names a language version of the template
// rather than its source.
func (s *Service) translated(ctx context.Context, ws *store.Workspace, id uuid.UUID, lang string) bool {
	if lang == "" || s.Translations == nil {
		return false
	}
	src, err := s.Translations.SourceLanguage(ctx, ws, id)
	return err == nil && lang != src
}

// TemplateIn loads a template for the editor in one of its languages: a
// language version's subject, preheader, content and save state in place
// of the draft's (?language=).
func (s *Service) TemplateIn(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref, lang string) (TransactionalTemplate, error) {
	t, err := s.Template(ctx, ws, p, ref)
	if err != nil || s.Translations == nil {
		return t, err
	}
	id, _ := uuid.Parse(t.ID)
	src, err := s.Translations.SourceLanguage(ctx, ws, id)
	if err != nil {
		return t, err
	}
	t.SourceLanguage, t.Language = src, src
	if lang == "" || lang == src {
		return t, nil
	}
	d, err := s.Translations.Draft(ctx, ws, id, lang)
	if err != nil {
		return t, err
	}
	t.Language = d.Language
	t.Subject, t.Preheader, t.MJML, t.HTML, t.PlainText = d.Subject, d.Preheader, d.MJML, d.HTML, d.PlainText
	t.DraftRevision, t.DraftSavedAt, t.DraftSavedBy, t.DraftHash = d.Revision, d.SavedAt, d.SavedBy, d.Hash
	return t, nil
}

// SaveDraftIn saves the draft, or one of its language versions.
func (s *Service) SaveDraftIn(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref, lang string, in TransactionalDraftSave) (TransactionalDraftSaved, error) {
	if lang == "" || s.Translations == nil {
		return s.SaveDraft(ctx, ws, p, ref, in)
	}
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalDraftSaved{}, err
	}
	if !s.translated(ctx, ws, id, lang) {
		return s.SaveDraft(ctx, ws, p, ref, in)
	}
	// Variables, test data, sender and tracking are the template's, shared
	// by every language: changed while editing a language, they are saved
	// with the draft.
	shared, err := s.sharedChanges(ctx, ws, id, in)
	if err != nil {
		return TransactionalDraftSaved{}, err
	}
	if shared != nil {
		if _, err := s.SaveDraft(ctx, ws, p, id.String(), *shared); err != nil {
			return TransactionalDraftSaved{}, err
		}
	}
	return s.Translations.SaveDraft(ctx, ws, p, id, lang, in)
}

// sharedChanges is the draft save that carries a language save's changes
// to the template's own fields, or nil when there are none.
func (s *Service) sharedChanges(ctx context.Context, ws *store.Workspace, id uuid.UUID, in TransactionalDraftSave) (*TransactionalDraftSave, error) {
	cur, err := s.Q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return nil, api.MapError(err, "Template")
	}
	out := TransactionalDraftSave{BaseRevision: int(cur.DraftRevision), Label: in.Label, Source: in.Source}
	changed := false
	if in.Variables != nil && string(mustJSON(*in.Variables)) != string(mustJSON(parseVariables(cur.Variables))) {
		out.Variables, changed = in.Variables, true
	}
	if in.TestData != nil && string(mustJSON(*in.TestData)) != string(mustJSON(parseTestData(cur.TestData))) {
		out.TestData, changed = in.TestData, true
	}
	if in.SenderID != nil && *in.SenderID != idString(cur.SenderID) {
		out.SenderID, changed = in.SenderID, true
	}
	if in.ReplyTo != nil && strings.TrimSpace(*in.ReplyTo) != cur.ReplyTo {
		out.ReplyTo, changed = in.ReplyTo, true
	}
	if in.TrackOpens != nil && *in.TrackOpens != cur.TrackOpens {
		out.TrackOpens, changed = in.TrackOpens, true
	}
	if in.TrackClicks != nil && *in.TrackClicks != cur.TrackClicks {
		out.TrackClicks, changed = in.TrackClicks, true
	}
	if !changed {
		return nil, nil
	}
	return &out, nil
}

// templateGetInput loads the editor's template, or one of its language
// versions.
type templateGetInput struct {
	TemplatePath
	Language string `query:"language" maxLength:"16" doc:"A language version's content in place of the draft's (internal/languages)"`
}

// renderLanguage is the content a render uses: the language asked for,
// else the linked contact's (when contact linking is on), else the
// fallback, as a send picks it.
func (s *Service) renderLanguage(ctx context.Context, ws *store.Workspace, lt liveTemplate, requested, email string) (content, string, error) {
	if s.Translations == nil {
		return lt.Content, "", nil
	}
	var contact *uuid.UUID
	if email != "" {
		if st, err := s.settingsRow(ctx, s.Q, ws.ID); err == nil && st.LinkContacts {
			if ct, err := s.Q.TxnContactByEmail(ctx, store.TxnContactByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
				id := ct.ID
				contact = &id
			}
		}
	}
	ch, err := s.Translations.Choose(ctx, s.Q, ws, lt.ID, lt.Version, requested, contact)
	if err != nil {
		return lt.Content, "", err
	}
	return lt.Content.withLanguage(ch.Content), ch.Language, nil
}
