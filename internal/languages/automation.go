package languages

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Automation emails have language versions per email version, like their
// content: a step sends one version, so writing a language version makes
// the email's next version (with every language carried over), and a live
// flow sends it only once the flow is published again.

// automationSource loads an automation email version as an owner.
func automationSource(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, version int) (Owner, Source, store.LanguagesAutomationEmailSourceRow, error) {
	r, err := q.LanguagesAutomationEmailSource(ctx, store.LanguagesAutomationEmailSourceParams{WorkspaceID: ws.ID, ID: id, Version: int32(version)})
	if err != nil {
		return Owner{}, Source{}, r, api.MapError(err, "Email")
	}
	o := Owner{Kind: OwnerAutomationEmail, ID: id, Version: r.Version, WorkspaceID: ws.ID, Name: r.Name}
	return o, Source{Subject: r.Subject, Preheader: r.Preheader, MJML: r.Mjml, HTML: r.Html, Version: int(r.Version), By: r.CreatedByName, At: r.CreatedAt}, r, nil
}

// automationTx runs fn on an automation email version's synced source.
func (s *Service) automationTx(ctx context.Context, ws *store.Workspace, id uuid.UUID, version int,
	fn func(q *store.Queries, o Owner, src Source, st sourceState, settings Settings) error) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		o, src, _, err := automationSource(ctx, q, ws, id, version)
		if err != nil {
			return err
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

// AutomationEmailLanguages lists the language versions of an automation
// email version (the step panel's Languages).
func (s *Service) AutomationEmailLanguages(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, version int) (EmailLanguages, error) {
	var views []view
	var st sourceState
	var settings Settings
	err := s.automationTx(ctx, ws, id, version, func(q *store.Queries, o Owner, _ Source, state sourceState, set Settings) error {
		var err error
		views, err = s.versions(ctx, q, o, state)
		st, settings = state, set
		return err
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	out := overview(st, views, settings, nil, nil)
	out.CanChangeSource = false
	s.accessOf(ctx, ws, p, &out, authz.AutomationsEdit)
	return out, nil
}

// AutomationEmailLanguage is one language version of an automation email
// version, to edit.
type AutomationEmailLanguage struct {
	Language  string `json:"language"`
	Name      string `json:"name"`
	Status    string `json:"status" enum:"up_to_date,outdated,draft"`
	Version   int    `json:"version"`
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	HTML      string `json:"html"`
}

// AutomationEmailLanguageContent returns a language version's content.
func (s *Service) AutomationEmailLanguageContent(ctx context.Context, ws *store.Workspace, id uuid.UUID, version int, lang string) (AutomationEmailLanguage, error) {
	var out AutomationEmailLanguage
	err := s.automationTx(ctx, ws, id, version, func(q *store.Queries, o Owner, _ Source, st sourceState, _ Settings) error {
		v, err := s.loadView(ctx, q, o, st, Base(lang))
		if err != nil {
			return err
		}
		out = AutomationEmailLanguage{Language: v.row.Language, Name: NameOf(v.row.Language), Status: v.progress.Status, Version: int(v.row.Version),
			Subject: v.row.Subject, Preheader: v.row.Preheader, HTML: v.row.Html}
		return nil
	})
	return out, err
}

// AutomationLanguageSave is a language version written in the flow editor.
type AutomationLanguageSave struct {
	// Version is the email version the step sends; saving makes the next.
	Version   int    `json:"version" minimum:"1"`
	Subject   string `json:"subject" minLength:"1" maxLength:"998"`
	Preheader string `json:"preheader,omitempty" maxLength:"500"`
	HTML      string `json:"html" minLength:"1" maxLength:"524288"`
	// Translated marks it translated, so it is sent.
	Translated bool `json:"translated,omitempty"`
}

// AutomationLanguageSaved is the email's new version and its languages.
type AutomationLanguageSaved struct {
	Version   int            `json:"version"`
	Languages EmailLanguages `json:"languages"`
}

// nextAutomationVersion copies an email version, with its language
// versions, as the email's next version and runs fn on it. A save is
// audited here; marking and removing audit themselves.
func (s *Service) nextAutomationVersion(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, from int, action, lang string,
	fn func(q *store.Queries, o Owner, src Source, st sourceState, settings Settings) error) (int, error) {
	var next int32
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		latest, err := q.LanguagesAutomationEmailLock(ctx, store.LanguagesAutomationEmailLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Email")
		}
		if _, _, _, err := automationSource(ctx, q, ws, id, from); err != nil {
			return err
		}
		next = latest + 1
		if err := q.LanguagesAutomationEmailCopyVersion(ctx, store.LanguagesAutomationEmailCopyVersionParams{EmailID: id, FromVersion: int32(from), ToVersion: next,
			CreatedBy: actorID(p)}); err != nil {
			return err
		}
		if err := q.LanguagesAutomationEmailSetLatest(ctx, store.LanguagesAutomationEmailSetLatestParams{ID: id, Version: next}); err != nil {
			return err
		}
		if err := s.CopyForward(ctx, q, ws.ID, id, from, int(next)); err != nil {
			return err
		}
		o, src, row, err := automationSource(ctx, q, ws, id, int(next))
		if err != nil {
			return err
		}
		settings, err := s.Settings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		st, err := s.syncSource(ctx, q, o, src, settings.Default)
		if err != nil {
			return err
		}
		if err := fn(q, o, src, st, settings); err != nil {
			return err
		}
		if action != "save" {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "automation_email.language." + action, TargetType: OwnerAutomationEmail, TargetID: id.String(),
			TargetName: row.Name, After: map[string]any{"language": lang, "version": next}, Metadata: map[string]any{"from_version": from, "permission": string(authz.AutomationsEdit)}})
	})
	return int(next), err
}

// SaveAutomationEmailLanguage writes a language version of an automation
// email, adding it when the email has none, as the email's next version.
func (s *Service) SaveAutomationEmailLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in AutomationLanguageSave) (AutomationLanguageSaved, error) {
	lang = Base(lang)
	version, err := s.nextAutomationVersion(ctx, ws, p, id, in.Version, "save", lang, func(q *store.Queries, o Owner, src Source, st sourceState, settings Settings) error {
		if lang == st.Language {
			return api.Invalid("path.language", "That's the language the email is written in. Edit the email itself.")
		}
		views, err := s.versions(ctx, q, o, st)
		if err != nil {
			return err
		}
		has := false
		for _, v := range views {
			has = has || v.row.Language == lang
		}
		if !has {
			if err := checkNewVersion(settings, st, views, lang); err != nil {
				return err
			}
			if _, err := s.addVersion(ctx, q, o, p, st, src, lang, StartCopy, nil); err != nil {
				return err
			}
		}
		cur, err := lockVersion(ctx, q, o, lang)
		if err != nil {
			return err
		}
		// The flow editor writes HTML: the version's MJML goes, as the
		// email's own does when it's edited there.
		if _, _, err := s.saveVersion(ctx, q, o, p, lang, SaveInput{BaseVersion: int(cur.Version), Subject: in.Subject, Preheader: in.Preheader, HTML: in.HTML,
			Label: "edited in the flow editor", Source: "manual", Force: true, Audited: true}); err != nil {
			return err
		}
		if in.Translated && cur.Status != storedTranslated {
			return s.mark(ctx, q, o, p, st, src, lang, nil, "mark_translated")
		}
		return nil
	})
	if err != nil {
		return AutomationLanguageSaved{}, err
	}
	out, err := s.AutomationEmailLanguages(ctx, ws, p, id, version)
	return AutomationLanguageSaved{Version: version, Languages: out}, err
}

// MarkAutomationEmailLanguage marks a language version up to date (or a
// Draft translated) in the email's next version.
func (s *Service) MarkAutomationEmailLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, from int, translated bool) (AutomationLanguageSaved, error) {
	lang = Base(lang)
	action := "mark_up_to_date"
	if translated {
		action = "mark_translated"
	}
	version, err := s.nextAutomationVersion(ctx, ws, p, id, from, action, lang, func(q *store.Queries, o Owner, src Source, st sourceState, _ Settings) error {
		if _, err := lockVersion(ctx, q, o, lang); err != nil {
			return err
		}
		return s.mark(ctx, q, o, p, st, src, lang, nil, action)
	})
	if err != nil {
		return AutomationLanguageSaved{}, err
	}
	out, err := s.AutomationEmailLanguages(ctx, ws, p, id, version)
	return AutomationLanguageSaved{Version: version, Languages: out}, err
}

// RemoveAutomationEmailLanguage removes a language version in the email's
// next version.
func (s *Service) RemoveAutomationEmailLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, from int) (AutomationLanguageSaved, error) {
	lang = Base(lang)
	version, err := s.nextAutomationVersion(ctx, ws, p, id, from, "remove", lang, func(q *store.Queries, o Owner, _ Source, _ sourceState, _ Settings) error {
		if _, err := lockVersion(ctx, q, o, lang); err != nil {
			return err
		}
		return s.removeVersion(ctx, q, o, lang)
	})
	if err != nil {
		return AutomationLanguageSaved{}, err
	}
	out, err := s.AutomationEmailLanguages(ctx, ws, p, id, version)
	return AutomationLanguageSaved{Version: version, Languages: out}, err
}

// CopyForward carries an automation email version's language versions to
// its next version, when its source is edited: the blocks the edit changed
// are Outdated there.
func (s *Service) CopyForward(ctx context.Context, q *store.Queries, wsID, id uuid.UUID, from, to int) error {
	return copyVersions(ctx, q, OwnerAutomationEmail, id, from, to)
}

// copyVersions copies an owner version's source and language versions to
// another of its versions.
func copyVersions(ctx context.Context, q *store.Queries, kind string, id uuid.UUID, from, to int) error {
	if err := q.ContentSourceCopy(ctx, store.ContentSourceCopyParams{OwnerKind: kind, OwnerID: id, FromVersion: int32(from), ToVersion: int32(to)}); err != nil {
		return err
	}
	rows, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: kind, OwnerID: id, OwnerVersion: int32(from)})
	if err != nil || len(rows) == 0 {
		return err
	}
	langs, ids := make([]string, len(rows)), make([]uuid.UUID, len(rows))
	for i, r := range rows {
		langs[i], ids[i] = r.Language, uuid.Must(uuid.NewV7())
	}
	return q.ContentTranslationsCopy(ctx, store.ContentTranslationsCopyParams{Ids: ids, Languages: langs, OwnerKind: kind, OwnerID: id,
		FromVersion: int32(from), ToVersion: int32(to)})
}

// AutomationEmailState is a language version of an automation email
// version, as the publish dialog names it.
type AutomationEmailState struct {
	Language string
	Name     string
	Status   string // up_to_date, outdated, draft
	// Attention counts the blocks to look at when it's outdated.
	Attention int
	By        string
	Held      bool
}

// AutomationEmailStates lists an automation email version's language
// versions (the publish dialog's diff and checks), read with q.
func (s *Service) AutomationEmailStates(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, version int) ([]AutomationEmailState, error) {
	o, src, _, err := automationSource(ctx, q, ws, id, version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	settings, err := s.Settings(ctx, q, ws.ID)
	if err != nil {
		return nil, err
	}
	st, err := s.syncSource(ctx, q, o, src, settings.Default)
	if err != nil {
		return nil, err
	}
	views, err := s.versions(ctx, q, o, st)
	if err != nil {
		return nil, err
	}
	out := []AutomationEmailState{}
	for _, v := range views {
		if !settings.Offers(v.row.Language) {
			continue
		}
		out = append(out, AutomationEmailState{Language: v.row.Language, Name: NameOf(v.row.Language), Status: v.progress.Status, Attention: v.progress.Attention,
			By: v.by, Held: v.row.Held})
	}
	return out, nil
}
