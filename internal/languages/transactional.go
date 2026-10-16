package languages

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Transactional templates have language versions on their draft, edited in
// the template editor. Publishing a version of the template publishes
// every language with it: each published version keeps its own (owner
// version = the template version), so a rollback brings its languages
// back. A send picks the version its recipient gets: the language the call
// asked for, else the linked contact's, else the fallback.

// templateSource loads a template's draft as an owner, by its id or key.
func templateSource(ctx context.Context, q *store.Queries, ws *store.Workspace, ref string) (Owner, Source, store.LanguagesTemplateSourceRow, error) {
	p := store.LanguagesTemplateSourceParams{WorkspaceID: ws.ID}
	if id, err := uuid.Parse(ref); err == nil {
		p.ID = id
	} else {
		p.Key = strings.ToLower(strings.TrimSpace(ref))
	}
	t, err := q.LanguagesTemplateSource(ctx, p)
	if err != nil {
		return Owner{}, Source{}, t, api.MapError(err, "Template")
	}
	o := Owner{Kind: OwnerTransactional, ID: t.ID, WorkspaceID: ws.ID, Name: t.Key}
	return o, Source{Subject: t.Subject, Preheader: t.Preheader, MJML: t.Mjml, HTML: t.Html, Version: int(t.DraftRevision), By: t.SavedByName, At: t.DraftSavedAt}, t, nil
}

// templateTx runs fn on a template draft's synced source in a transaction.
// Changes are refused while the template is archived.
func (s *Service) templateTx(ctx context.Context, ws *store.Workspace, ref string, write bool,
	fn func(q *store.Queries, o Owner, src Source, st sourceState, settings Settings) error) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		o, src, t, err := templateSource(ctx, q, ws, ref)
		if err != nil {
			return err
		}
		if write && t.ArchivedAt != nil {
			return api.Conflict(t.Key + " is archived. Restore it to change its language versions.")
		}
		settings, err := s.Settings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		st, err := s.syncSource(ctx, q, o, src, settings.Default)
		if err != nil {
			return err
		}
		return fn(q, o, src, st, settings)
	})
}

// TemplateLanguages lists a template draft's language versions (the
// editor's language switcher).
func (s *Service) TemplateLanguages(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (EmailLanguages, error) {
	var out EmailLanguages
	err := s.templateTx(ctx, ws, ref, false, func(q *store.Queries, o Owner, _ Source, st sourceState, settings Settings) error {
		views, err := s.versions(ctx, q, o, st)
		out = overview(st, views, settings, nil, nil)
		return err
	})
	if err != nil {
		return out, err
	}
	s.accessOf(ctx, ws, p, &out, authz.TransactionalEdit)
	return out, nil
}

// AddTemplateLanguage adds a Draft language version to a template's draft.
func (s *Service) AddTemplateLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in AddLanguageInput) (EmailLanguages, error) {
	lang := Base(in.Language)
	err := s.templateTx(ctx, ws, ref, true, func(q *store.Queries, o Owner, src Source, st sourceState, settings Settings) error {
		views, err := s.versions(ctx, q, o, st)
		if err != nil {
			return err
		}
		if err := checkNewVersion(settings, st, views, lang); err != nil {
			return err
		}
		start := in.Start
		if start == "" {
			start = StartCopy
		}
		if _, err := s.addVersion(ctx, q, o, p, st, src, lang, start, nil); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: o.auditPrefix() + "add", TargetType: o.Kind, TargetID: o.ID.String(), TargetName: o.Name,
			After: map[string]any{"language": lang, "start": start, "status": "draft"}, Metadata: map[string]any{"language": lang, "draftRevision": st.Version}})
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.TemplateLanguages(ctx, ws, p, ref)
}

// RemoveTemplateLanguage deletes a language version of a template's draft.
// Published versions keep theirs.
func (s *Service) RemoveTemplateLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref, lang string) (EmailLanguages, error) {
	err := s.templateTx(ctx, ws, ref, true, func(q *store.Queries, o Owner, _ Source, _ sourceState, _ Settings) error {
		return s.removeVersion(ctx, q, o, Base(lang))
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.TemplateLanguages(ctx, ws, p, ref)
}

// MarkTemplateLanguage marks a Draft translated, or an Outdated version up
// to date.
func (s *Service) MarkTemplateLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref, lang string, translated bool, in MarkInput) (EmailLanguages, error) {
	action := "mark_up_to_date"
	if translated {
		action, in.Blocks = "mark_translated", nil
	}
	err := s.templateTx(ctx, ws, ref, true, func(q *store.Queries, o Owner, src Source, st sourceState, _ Settings) error {
		return s.mark(ctx, q, o, p, st, src, Base(lang), in.Blocks, action)
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.TemplateLanguages(ctx, ws, p, ref)
}

// --- transactional.Translations ---------------------------------------------

// Transactional adapts the service to the transactional domain's
// Translations.
type Transactional struct{ S *Service }

var _ transactional.Translations = Transactional{}

// SourceLanguage is the language a template is written in: its draft
// source's, or the workspace's default before it has one.
func (t Transactional) SourceLanguage(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, error) {
	row, err := t.S.Q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: OwnerTransactional, OwnerID: id, OwnerVersion: 0})
	if err == nil {
		return row.Language, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	st, err := t.S.Settings(ctx, t.S.Q, ws.ID)
	if err != nil {
		return "", err
	}
	return st.Default, nil
}

// Draft is a language version of a template's draft.
func (t Transactional) Draft(ctx context.Context, ws *store.Workspace, id uuid.UUID, lang string) (transactional.TranslationDraft, error) {
	lang = Base(lang)
	if _, _, _, err := templateSource(ctx, t.S.Q, ws, id.String()); err != nil {
		return transactional.TranslationDraft{}, err
	}
	r, err := t.S.Q.ContentTranslation(ctx, store.ContentTranslationParams{OwnerKind: OwnerTransactional, OwnerID: id, OwnerVersion: 0, Language: lang})
	if err != nil {
		return transactional.TranslationDraft{}, notFoundVersion(err, lang)
	}
	at := r.UpdatedAt
	if r.AutosavedAt != nil {
		at = *r.AutosavedAt
	}
	return transactional.TranslationDraft{Language: lang, Subject: r.Subject, Preheader: r.Preheader, MJML: r.Mjml, HTML: r.Html, PlainText: r.PlainText,
		Revision: int(r.Version), SavedAt: at, SavedBy: r.UpdatedByName, Hash: r.ContentHash}, nil
}

// SaveDraft stores a new version of a language version of the draft. Only
// its subject, preheader and content belong to it: variables, test data,
// sender and tracking are the template's, saved with the draft.
func (t Transactional) SaveDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in transactional.TransactionalDraftSave) (transactional.TransactionalDraftSaved, error) {
	var out transactional.TransactionalDraftSaved
	err := t.S.templateTx(ctx, ws, id.String(), true, func(q *store.Queries, o Owner, _ Source, _ sourceState, _ Settings) error {
		cur, err := lockVersion(ctx, q, o, Base(lang))
		if err != nil {
			return err
		}
		subject, preheader, mj, html := cur.Subject, cur.Preheader, cur.Mjml, cur.Html
		if in.Subject != nil {
			if strings.ContainsAny(*in.Subject, "\r\n") {
				return api.Invalid("body.subject", "The subject must be one line.")
			}
			subject = *in.Subject
		}
		if in.Preheader != nil {
			preheader = *in.Preheader
		}
		if in.MJML != nil {
			mj, html = *in.MJML, ""
		}
		if in.HTML != nil {
			html = *in.HTML
		}
		saved, _, err := t.S.saveVersion(ctx, q, o, p, Base(lang), SaveInput{BaseVersion: in.BaseRevision, Subject: subject, Preheader: preheader, MJML: mj, HTML: html,
			Label: in.Label, Source: draftSource(in.Source), Force: in.Force})
		out = transactional.TransactionalDraftSaved{DraftRevision: saved.Version, SavedAt: saved.SavedAt, DraftHash: saved.ContentHash, DraftChanges: true,
			SizeBytes: saved.SizeBytes, HTML: saved.HTML, PlainText: saved.PlainText}
		return err
	})
	return out, err
}

// draftSource maps the template editor's save source to a language
// version's history source.
func draftSource(s string) string {
	switch s {
	case "", "autosave":
		return "autosave"
	case "import":
		return "import"
	}
	return "manual"
}

// Publishing lists the language versions publishing the draft publishes,
// the source first: the publish dialog's "Publishes English, Spanish and
// German together".
func (t Transactional) Publishing(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) ([]transactional.TransactionalPublishLanguage, error) {
	o, src, _, err := templateSource(ctx, q, ws, id.String())
	if err != nil {
		return nil, err
	}
	settings, err := t.S.Settings(ctx, q, ws.ID)
	if err != nil {
		return nil, err
	}
	st, err := t.S.syncSource(ctx, q, o, src, settings.Default)
	if err != nil {
		return nil, err
	}
	views, err := t.S.versions(ctx, q, o, st)
	if err != nil {
		return nil, err
	}
	out := []transactional.TransactionalPublishLanguage{}
	if len(views) == 0 {
		return out, nil
	}
	states := make([]VersionState, 0, len(views))
	for _, v := range views {
		states = append(states, VersionState{Language: v.row.Language, Status: v.row.Status, Held: v.row.Held})
	}
	ch := NewChooser(st.Language, settings, states)
	out = append(out, transactional.TransactionalPublishLanguage{Language: st.Language, Name: NameOf(st.Language), Status: "source", Gets: st.Language, GetsName: NameOf(st.Language)})
	for _, lang := range settings.Languages {
		for _, v := range views {
			if v.row.Language == lang {
				gets := ch.Pick(lang).Language
				out = append(out, transactional.TransactionalPublishLanguage{Language: lang, Name: NameOf(lang), Status: v.progress.Status, Attention: v.progress.Attention,
					Gets: gets, GetsName: NameOf(gets)})
			}
		}
	}
	return out, nil
}

// Publish publishes the draft's language versions with a new version of
// the template: the draft's source and versions are copied to it.
func (t Transactional) Publish(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, version int) error {
	o, src, _, err := templateSource(ctx, q, ws, id.String())
	if err != nil {
		return err
	}
	settings, err := t.S.Settings(ctx, q, ws.ID)
	if err != nil {
		return err
	}
	// The draft's source in step with what is published.
	if _, err := t.S.syncSource(ctx, q, o, src, settings.Default); err != nil {
		return err
	}
	return copyVersions(ctx, q, OwnerTransactional, id, 0, version)
}

// Changes lists the language versions the draft adds, changes (its
// content, or marked translated) or removes since a published version.
func (t Transactional) Changes(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, version int) ([]transactional.TransactionalLanguageChange, error) {
	draft, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: OwnerTransactional, OwnerID: id, OwnerVersion: 0})
	if err != nil {
		return nil, err
	}
	live, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: OwnerTransactional, OwnerID: id, OwnerVersion: int32(version)})
	if err != nil {
		return nil, err
	}
	was := map[string]store.ContentTranslationsRow{}
	for _, r := range live {
		was[r.Language] = r
	}
	out := []transactional.TransactionalLanguageChange{}
	seen := map[string]bool{}
	for _, r := range draft {
		seen[r.Language] = true
		p, ok := was[r.Language]
		switch {
		case !ok:
			out = append(out, transactional.TransactionalLanguageChange{Language: r.Language, Name: NameOf(r.Language), Change: "added"})
		case p.ContentHash != r.ContentHash || p.Status != r.Status || p.Held != r.Held:
			out = append(out, transactional.TransactionalLanguageChange{Language: r.Language, Name: NameOf(r.Language), Change: "changed"})
		}
	}
	for _, r := range live {
		if !seen[r.Language] {
			out = append(out, transactional.TransactionalLanguageChange{Language: r.Language, Name: NameOf(r.Language), Change: "removed"})
		}
	}
	return out, nil
}

// Choose picks the language version of a published template version a
// message uses: the language asked for, else the linked contact's, else
// the fallback (the source when the fallback has no version).
func (t Transactional) Choose(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, version int, requested string, contactID *uuid.UUID) (transactional.LanguageChoice, error) {
	settings, err := t.S.Settings(ctx, q, ws.ID)
	if err != nil {
		return transactional.LanguageChoice{}, err
	}
	source := settings.Default
	if row, err := q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: OwnerTransactional, OwnerID: id, OwnerVersion: int32(version)}); err == nil {
		source = row.Language
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return transactional.LanguageChoice{}, err
	}
	rows, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: OwnerTransactional, OwnerID: id, OwnerVersion: int32(version)})
	if err != nil {
		return transactional.LanguageChoice{}, err
	}
	states := make([]VersionState, 0, len(rows))
	for _, r := range rows {
		states = append(states, VersionState{Language: r.Language, Status: r.Status, Held: r.Held})
	}
	ch := NewChooser(source, settings, states)
	var pick Choice
	code := strings.TrimSpace(requested)
	if code != "" {
		pick = ch.Request(code)
	} else {
		if contactID != nil {
			if c, err := q.LanguagesContactLanguage(ctx, store.LanguagesContactLanguageParams{WorkspaceID: ws.ID, ID: *contactID}); err == nil {
				code = c
			}
		}
		pick = ch.Pick(code)
	}
	out := transactional.LanguageChoice{Language: pick.Language, Reason: pick.Reason, Contact: code}
	if pick.Language == source {
		return out, nil
	}
	for _, r := range rows {
		if r.Language == pick.Language {
			out.Content = &transactional.TranslationText{Subject: r.Subject, Preheader: r.Preheader, MJML: r.Mjml, HTML: r.Html, Text: r.PlainText}
		}
	}
	return out, nil
}

// Record stores the language a message was sent in.
func (t Transactional) Record(ctx context.Context, q *store.Queries, messageID uuid.UUID, ch transactional.LanguageChoice) error {
	return q.LanguagesSetMessageLanguage(ctx, store.LanguagesSetMessageLanguageParams{ID: messageID, Language: ch.Language, ContactLanguage: ch.Contact,
		LanguageReason: ch.Reason})
}
