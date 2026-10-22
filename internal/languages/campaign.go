package languages

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// editable campaign statuses: language versions change with the content.
var editableCampaign = map[string]bool{"draft": true, "approval": true, "approved": true, "scheduled": true}

// campaignSource loads a campaign as an owner with its source content.
func campaignSource(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (Owner, Source, store.LanguagesCampaignSourceRow, error) {
	c, err := q.LanguagesCampaignSource(ctx, store.LanguagesCampaignSourceParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return Owner{}, Source{}, c, api.MapError(err, "Campaign")
	}
	at := c.UpdatedAt
	if c.AutosavedAt != nil {
		at = *c.AutosavedAt
	}
	o := Owner{Kind: OwnerCampaign, ID: c.ID, WorkspaceID: ws.ID, Name: c.Name}
	return o, Source{Subject: c.Subject, Preheader: c.Preheader, MJML: c.Mjml, HTML: c.Html, Version: int(c.Version), By: c.UpdatedByName, At: at}, c, nil
}

func readOnly(l i18n.Lang, c store.LanguagesCampaignSourceRow) error {
	if editableCampaign[c.Status] {
		return nil
	}
	if c.Status == "sending" || c.Status == "paused" {
		return api.Conflict(mSending.Text(l, nil))
	}
	return api.Conflict(mSent.Text(l, nil))
}

// campaignTx runs fn on a campaign's synced source in a transaction, then
// tells the approvals domain when write is true.
func (s *Service) campaignTx(ctx context.Context, ws *store.Workspace, id uuid.UUID, write bool,
	fn func(q *store.Queries, o Owner, src Source, c store.LanguagesCampaignSourceRow, st sourceState, settings Settings) error) error {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		o, src, c, err := campaignSource(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if write {
			if err := readOnly(langOf(ctx), c); err != nil {
				return err
			}
		}
		settings, err := s.Settings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		st, err := s.syncSource(ctx, q, o, src, settings.Default)
		if err != nil {
			return err
		}
		return fn(q, o, src, c, st, settings)
	})
	if err == nil && write && s.Changed != nil {
		s.Changed(ctx, ws, id)
	}
	return err
}

// CampaignLanguages lists a campaign's language versions for the switcher.
func (s *Service) CampaignLanguages(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (EmailLanguages, error) {
	var out EmailLanguages
	var audience json.RawMessage
	var views []view
	var st sourceState
	var settings Settings
	err := s.campaignTx(ctx, ws, id, false, func(q *store.Queries, o Owner, _ Source, c store.LanguagesCampaignSourceRow, state sourceState, set Settings) error {
		var err error
		views, err = s.versions(ctx, q, o, state)
		st, settings, audience = state, set, c.Audience
		return err
	})
	if err != nil {
		return out, err
	}
	readers, total := s.readers(ctx, ws, audience)
	out = overview(langOf(ctx), st, views, settings, readers, total)
	s.accessOf(ctx, ws, p, &out, authz.CampaignsEdit)
	return out, nil
}

// readers counts an audience's recipients per contact language; nil when
// the audience is empty or can't be counted.
func (s *Service) readers(ctx context.Context, ws *store.Workspace, audience json.RawMessage) (map[string]int64, *int64) {
	if s.AudienceLanguages == nil || len(audience) == 0 || string(audience) == "{}" {
		return nil, nil
	}
	m, total, err := s.AudienceLanguages(ctx, ws, audience)
	if err != nil || total == 0 {
		return nil, nil
	}
	return m, &total
}

func (s *Service) accessOf(ctx context.Context, ws *store.Workspace, p *authz.Principal, out *EmailLanguages, edit authz.Permission) {
	if p == nil {
		return
	}
	out.CanEdit = p.Can(edit)
	out.CanManageWorkspace = p.Can(authz.WorkspaceManage)
	if !out.CanEdit {
		out.EditHelpers = nonNil(access.Helpers(ctx, s.Q, ws.ID, edit))
	}
	if !out.CanManageWorkspace {
		out.WorkspaceHelpers = nonNil(access.Helpers(ctx, s.Q, ws.ID, authz.WorkspaceManage))
	}
}

// AddLanguageInput adds a language version to an email.
type AddLanguageInput struct {
	Language string `json:"language" pattern:"^[a-z]{2,3}$" doc:"A workspace language"`
	// Start is "copy" (the source's layout and text) or "layout" (text
	// blocks start empty).
	Start string `json:"start,omitempty" enum:"copy,layout" default:"copy"`
}

func checkNewVersion(rl i18n.Lang, settings Settings, st sourceState, views []view, lang string) error {
	l, ok := Lookup(lang)
	switch {
	case !ok:
		return api.Invalid("body.language", mPickWorkspaceLanguage.Text(rl, nil))
	case l.RTL:
		return rtlRefusal(rl, l, settings.Fallback)
	case !settings.Offers(lang):
		return api.Invalid("body.language", named(rl, mNotWorkspaceLanguage, l.Code, nil))
	case lang == st.Language:
		return api.Conflict(named(rl, mWrittenIn, l.Code, nil))
	}
	for _, v := range views {
		if v.row.Language == lang {
			return api.Conflict(named(rl, mHasVersion, l.Code, nil))
		}
	}
	return nil
}

// AddCampaignLanguage adds a Draft language version to a campaign.
func (s *Service) AddCampaignLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AddLanguageInput) (EmailLanguages, error) {
	lang := Base(in.Language)
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, src Source, c store.LanguagesCampaignSourceRow, st sourceState, settings Settings) error {
		views, err := s.versions(ctx, q, o, st)
		if err != nil {
			return err
		}
		if err := checkNewVersion(langOf(ctx), settings, st, views, lang); err != nil {
			return err
		}
		start := in.Start
		if start == "" {
			start = StartCopy
		}
		if _, err := s.addVersion(ctx, q, o, p, st, src, lang, start, c.AbTest); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: o.auditPrefix() + "add", TargetType: o.Kind, TargetID: id.String(), TargetName: o.Name,
			After: map[string]any{"language": lang, "start": start, "status": "draft"}, Metadata: map[string]any{"language": lang, "sourceVersion": st.Version}})
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.CampaignLanguages(ctx, ws, p, id)
}

// RemoveCampaignLanguage deletes a campaign's language version.
func (s *Service) RemoveCampaignLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string) (EmailLanguages, error) {
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, _ Source, _ store.LanguagesCampaignSourceRow, _ sourceState, _ Settings) error {
		return s.removeVersion(ctx, q, o, Base(lang))
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.CampaignLanguages(ctx, ws, p, id)
}

// MarkInput marks a language version translated or up to date.
type MarkInput struct {
	// Blocks limits "Mark as up to date" to some source blocks (their keys,
	// or "subject" and "preheader"); empty marks the whole version.
	Blocks []string `json:"blocks,omitempty" maxItems:"200"`
}

// MarkCampaignLanguage marks a Draft translated, or an Outdated version
// (or some of its blocks) up to date.
func (s *Service) MarkCampaignLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, translated bool, in MarkInput) (EmailLanguages, error) {
	action := "mark_up_to_date"
	if translated {
		action, in.Blocks = "mark_translated", nil
	}
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, src Source, _ store.LanguagesCampaignSourceRow, st sourceState, _ Settings) error {
		return s.mark(ctx, q, o, p, st, src, Base(lang), in.Blocks, action)
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.CampaignLanguages(ctx, ws, p, id)
}

// KeepUpToDateInput keeps versions up to date after a source edit that
// needs no translating ("Keep them up to date").
type KeepUpToDateInput struct {
	// Blocks are the source block keys the edit changed.
	Blocks []string `json:"blocks" minItems:"1" maxItems:"200"`
}

// KeepCampaignUpToDate marks the given source blocks up to date in every
// translated version.
func (s *Service) KeepCampaignUpToDate(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in KeepUpToDateInput) (EmailLanguages, error) {
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, src Source, _ store.LanguagesCampaignSourceRow, st sourceState, _ Settings) error {
		views, err := s.versions(ctx, q, o, st)
		if err != nil {
			return err
		}
		for _, v := range views {
			if v.row.Status != storedTranslated {
				continue
			}
			if err := s.mark(ctx, q, o, p, st, src, v.row.Language, in.Blocks, "keep_up_to_date"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.CampaignLanguages(ctx, ws, p, id)
}

// HoldInput sends the fallback to a language's readers instead.
type HoldInput struct {
	Held bool `json:"held"`
}

// HoldCampaignLanguage holds (or releases) a language version.
func (s *Service) HoldCampaignLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in HoldInput) (EmailLanguages, error) {
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, _ Source, _ store.LanguagesCampaignSourceRow, _ sourceState, _ Settings) error {
		return s.hold(ctx, q, o, Base(lang), in.Held)
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.CampaignLanguages(ctx, ws, p, id)
}

// ApplyInput copies a link or image change of the source into the
// language versions ("Apply to all languages").
type ApplyInput struct {
	Kind string `json:"kind" enum:"link,image"`
	From string `json:"from" minLength:"1" maxLength:"4000"`
	To   string `json:"to" minLength:"1" maxLength:"4000"`
	// Block is the source block's key; empty applies it anywhere it appears.
	Block string `json:"block,omitempty" maxLength:"32"`
}

// ApplyResult says which versions changed.
type ApplyResult struct {
	Languages []string `json:"languages"`
	Replaced  int      `json:"replaced"`
}

// ApplyCampaignChange applies a source link or image change to every
// language version.
func (s *Service) ApplyCampaignChange(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ApplyInput) (ApplyResult, error) {
	out := ApplyResult{Languages: []string{}}
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, _ Source, _ store.LanguagesCampaignSourceRow, st sourceState, _ Settings) error {
		views, err := s.versions(ctx, q, o, st)
		if err != nil {
			return err
		}
		for _, v := range views {
			indexes := []int{}
			for i, l := range v.links.Blocks {
				if in.Block == "" || l.Src == in.Block {
					indexes = append(indexes, i)
				}
			}
			mj, n := replaceInBlock(v.row.Mjml, v.blocks, indexes, in.From, in.To)
			if n == 0 {
				continue
			}
			html := v.row.Html
			if strings.TrimSpace(mj) != "" {
				res := mjml.Compile(mjml.SetHeadMeta(mj, v.row.Subject, v.row.Preheader))
				if len(res.Errors) > 0 {
					return api.Invalid("body.to", named(langOf(ctx), mApplyDoesntCompile, v.row.Language, i18n.Args{"error": res.Errors[0].Message}))
				}
				html = res.HTML
			}
			lab := newLabel(mLabelAppliedFrom, st.Language, i18n.Args{"kind": in.Kind})
			if _, _, err := s.saveVersion(ctx, q, o, p, v.row.Language, SaveInput{BaseVersion: int(v.row.Version), Subject: v.row.Subject, Preheader: v.row.Preheader,
				MJML: mj, HTML: html, ABTest: v.row.AbTest, Label: lab.English, LabelText: lab.Text, Source: "manual"}); err != nil {
				return err
			}
			out.Languages = append(out.Languages, v.row.Language)
			out.Replaced += n
		}
		return nil
	})
	return out, err
}

// SourceInput changes the language an email is written in.
type SourceInput struct {
	Language string `json:"language" pattern:"^[a-z]{2,3}$"`
}

// SetCampaignSource changes the source language while the campaign has no
// language versions.
func (s *Service) SetCampaignSource(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SourceInput) (EmailLanguages, error) {
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, _ Source, _ store.LanguagesCampaignSourceRow, st sourceState, settings Settings) error {
		views, err := s.versions(ctx, q, o, st)
		if err != nil {
			return err
		}
		if len(views) > 0 {
			return api.Conflict(mSourceOneVersion.Text(langOf(ctx), nil))
		}
		lang := Base(in.Language)
		if !settings.Offers(lang) {
			return api.Invalid("body.language", mPickWorkspaceLanguage.Text(langOf(ctx), nil))
		}
		if lang == st.Language {
			return nil
		}
		if err := q.ContentSourceSetLanguage(ctx, store.ContentSourceSetLanguageParams{OwnerKind: o.Kind, OwnerID: o.ID, OwnerVersion: o.Version, Language: lang, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: o.auditPrefix() + "source_change", TargetType: o.Kind, TargetID: id.String(), TargetName: o.Name,
			Before: map[string]any{"source": st.Language}, After: map[string]any{"source": lang}})
	})
	if err != nil {
		return EmailLanguages{}, err
	}
	return s.CampaignLanguages(ctx, ws, p, id)
}

// CompareCampaign returns the compare view of a campaign's language version.
func (s *Service) CompareCampaign(ctx context.Context, ws *store.Workspace, id uuid.UUID, lang string) (LanguageCompare, error) {
	var out LanguageCompare
	err := s.campaignTx(ctx, ws, id, false, func(q *store.Queries, o Owner, src Source, _ store.LanguagesCampaignSourceRow, st sourceState, _ Settings) error {
		views, err := s.versions(ctx, q, o, st)
		if err != nil {
			return err
		}
		for _, v := range views {
			if v.row.Language == Base(lang) {
				out = compareView(langOf(ctx), st, src, v)
				return nil
			}
		}
		return versionNotFound(langOf(ctx), lang)
	})
	return out, err
}

// BlockInput names a block of the compare view.
type BlockInput struct {
	// Key is the source block's key (Copy into …).
	Key string `json:"key,omitempty" maxLength:"32"`
	// Index is the language version's block position (Remove from …).
	Index *int `json:"index,omitempty" minimum:"0"`
}

// CopyBlockToCampaignLanguage copies a source block into a language version
// ("Copy into German"), after the block that precedes it in the source.
func (s *Service) CopyBlockToCampaignLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in BlockInput) (LanguageCompare, error) {
	lang = Base(lang)
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, src Source, _ store.LanguagesCampaignSourceRow, st sourceState, _ Settings) error {
		v, err := s.loadView(ctx, q, o, st, lang)
		if err != nil {
			return err
		}
		si := -1
		for i, k := range st.Keys {
			if k.Key == in.Key {
				si = i
			}
		}
		if si < 0 {
			return notFound(ctx, mBlockNotFound)
		}
		// Insert after the version block linked to the closest earlier source block.
		insertAt := -1
		for j := si - 1; j >= 0 && insertAt < 0; j-- {
			for vi, l := range v.links.Blocks {
				if l.Src == st.Keys[j].Key {
					insertAt = vi
					break
				}
			}
		}
		block := src.MJML[st.Blocks[si].Start:st.Blocks[si].End]
		mj, ok := insertBlock(v.row.Mjml, v.blocks, insertAt, block)
		if !ok {
			return api.Conflict(named(langOf(ctx), mNoMJML, lang, nil))
		}
		return s.saveWithLinks(ctx, q, o, p, v, st, mj, func(l Links, blocks []Block) Links {
			// The copied block points at its source block, as it is now.
			pos := insertAt + 1
			if pos < len(l.Blocks) {
				l.Blocks[pos].Src, l.Blocks[pos].Hash, l.Blocks[pos].Text, l.Blocks[pos].Version = st.Keys[si].Key, st.Blocks[si].Hash, st.Blocks[si].Text, st.Version
			}
			return l
		}, newLabel(mLabelBlockCopied, st.Language, nil))
	})
	if err != nil {
		return LanguageCompare{}, err
	}
	return s.CompareCampaign(ctx, ws, id, lang)
}

// RemoveBlockFromCampaignLanguage removes a block from a language version
// ("Remove from German"), for blocks removed from the source.
func (s *Service) RemoveBlockFromCampaignLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in BlockInput) (LanguageCompare, error) {
	lang = Base(lang)
	if in.Index == nil {
		return LanguageCompare{}, api.Invalid("body.index", mWhichBlock.Text(langOf(ctx), nil))
	}
	err := s.campaignTx(ctx, ws, id, true, func(q *store.Queries, o Owner, _ Source, _ store.LanguagesCampaignSourceRow, st sourceState, _ Settings) error {
		v, err := s.loadView(ctx, q, o, st, lang)
		if err != nil {
			return err
		}
		i := *in.Index
		if i < 0 || i >= len(v.blocks) {
			return notFound(ctx, mBlockNotFound)
		}
		b := v.blocks[i]
		mj := v.row.Mjml[:b.Start] + v.row.Mjml[b.End:]
		return s.saveWithLinks(ctx, q, o, p, v, st, mj, nil, newLabel(mLabelBlockRemoved, st.Language, nil))
	})
	if err != nil {
		return LanguageCompare{}, err
	}
	return s.CompareCampaign(ctx, ws, id, lang)
}

func (s *Service) loadView(ctx context.Context, q *store.Queries, o Owner, st sourceState, lang string) (view, error) {
	t, err := lockVersion(ctx, q, o, lang)
	if err != nil {
		return view{}, err
	}
	return s.viewOf(st, t), nil
}

// saveWithLinks saves new MJML for a version (compiled on the server) and
// lets fix adjust the links of its blocks.
func (s *Service) saveWithLinks(ctx context.Context, q *store.Queries, o Owner, p *authz.Principal, v view, st sourceState, mj string, fix func(Links, []Block) Links, lab label) error {
	res := mjml.Compile(mjml.SetHeadMeta(mj, v.row.Subject, v.row.Preheader))
	if len(res.Errors) > 0 {
		return api.Invalid("body", named(langOf(ctx), mDoesntCompile, v.row.Language, i18n.Args{"error": res.Errors[0].Message}))
	}
	_, t, err := s.saveVersion(ctx, q, o, p, v.row.Language, SaveInput{BaseVersion: int(v.row.Version), Subject: v.row.Subject, Preheader: v.row.Preheader,
		MJML: mj, HTML: res.HTML, ABTest: v.row.AbTest, Label: lab.English, LabelText: lab.Text, Source: "manual"})
	if err != nil || fix == nil {
		return err
	}
	var l Links
	_ = json.Unmarshal(t.Links, &l)
	blocks := Blocks(t.Mjml, t.Html)
	l = fix(padLinks(l, blocks), blocks)
	lb, _ := json.Marshal(l)
	return q.ContentTranslationSetLinks(ctx, store.ContentTranslationSetLinksParams{ID: t.ID, Links: lb})
}

// insertBlock inserts MJML after the block at index after (-1: first) of a
// document's body.
func insertBlock(src string, blocks []Block, after int, block string) (string, bool) {
	root, err := mjml.Parse(src)
	if err != nil {
		return "", false
	}
	body := mjml.Body(root)
	if body == nil {
		return "", false
	}
	var pos int
	switch {
	case after >= 0 && after < len(blocks):
		pos = blocks[after].End
	case len(blocks) > 0:
		pos = blocks[0].Start
	default:
		// An empty body: right after <mj-body …>.
		gt := strings.IndexByte(src[body.Start:], '>')
		if gt < 0 {
			return "", false
		}
		pos = body.Start + gt + 1
	}
	return src[:pos] + "\n    " + block + src[pos:], true
}

// CampaignHistory lists the saves of a campaign's language version.
func (s *Service) CampaignHistory(ctx context.Context, ws *store.Workspace, id uuid.UUID, lang string, limit int) (campaigns.ComposerVersionList, error) {
	out := campaigns.ComposerVersionList{Items: []campaigns.ComposerVersion{}}
	if _, _, _, err := campaignSource(ctx, s.Q, ws, id); err != nil {
		return out, err
	}
	t, err := s.Q.ContentTranslation(ctx, store.ContentTranslationParams{OwnerKind: OwnerCampaign, OwnerID: id, OwnerVersion: 0, Language: Base(lang)})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, versionNotFound(langOf(ctx), lang)
	}
	if err != nil {
		return out, err
	}
	rows, err := s.Q.ContentTranslationVersions(ctx, store.ContentTranslationVersionsParams{TranslationID: t.ID, Lim: int32(limit)})
	if err != nil {
		return out, err
	}
	out.Total = int(t.Version)
	for _, r := range rows {
		src := r.Source
		if src == "fix" {
			src = "manual"
		}
		out.Items = append(out.Items, campaigns.ComposerVersion{Version: int(r.Version), SavedAt: r.SavedAt, SavedBy: r.SavedByName, Label: labelIn(langOf(ctx), r.Label, r.LabelText), Source: src,
			SizeBytes: int(r.SizeBytes), Current: r.Version == t.Version})
	}
	return out, nil
}

// --- campaigns.Translations -------------------------------------------------

// Campaigns adapts the service to the campaigns domain's Translations.
type Campaigns struct{ S *Service }

var _ campaigns.Translations = Campaigns{}

// SourceLanguage is the language a campaign is written in: its source's,
// or the workspace's default for a campaign without versions yet.
func (c Campaigns) SourceLanguage(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, error) {
	row, err := c.S.Q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: OwnerCampaign, OwnerID: id, OwnerVersion: 0})
	if err == nil {
		return row.Language, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	st, err := c.S.Settings(ctx, c.S.Q, ws.ID)
	if err != nil {
		return "", err
	}
	return st.Default, nil
}

// Content returns a language version's content.
func (c Campaigns) Content(ctx context.Context, ws *store.Workspace, id uuid.UUID, lang string) (campaigns.TranslationContent, error) {
	lang = Base(lang)
	if _, _, _, err := campaignSource(ctx, c.S.Q, ws, id); err != nil {
		return campaigns.TranslationContent{}, err
	}
	t, err := c.S.Q.ContentTranslation(ctx, store.ContentTranslationParams{OwnerKind: OwnerCampaign, OwnerID: id, OwnerVersion: 0, Language: lang})
	if errors.Is(err, pgx.ErrNoRows) {
		return campaigns.TranslationContent{}, versionNotFound(langOf(ctx), lang)
	}
	if err != nil {
		return campaigns.TranslationContent{}, err
	}
	return campaigns.TranslationContent{Language: lang, Subject: t.Subject, Preheader: t.Preheader, MJML: t.Mjml, HTML: t.Html, PlainText: t.PlainText, ABTest: t.AbTest,
		Version: int(t.Version), VersionCount: int(t.Version), ContentHash: t.ContentHash, UpdatedAt: t.UpdatedAt, UpdatedBy: t.UpdatedByName, AutosavedAt: t.AutosavedAt}, nil
}

// Save stores a new version of a campaign's language version.
func (c Campaigns) Save(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in campaigns.ComposerSaveContent, minify bool) (campaigns.ComposerSavedContent, error) {
	var out campaigns.ComposerSavedContent
	var ab json.RawMessage
	if in.ABTest != nil {
		ab, _ = json.Marshal(abForStore(in.ABTest))
	}
	err := c.S.campaignTx(ctx, ws, id, false, func(q *store.Queries, o Owner, _ Source, row store.LanguagesCampaignSourceRow, _ sourceState, _ Settings) error {
		if err := readOnly(langOf(ctx), row); err != nil {
			return err
		}
		saved, _, err := c.S.saveVersion(ctx, q, o, p, Base(lang), SaveInput{BaseVersion: in.BaseVersion, Subject: in.Subject, Preheader: in.Preheader, MJML: in.MJML,
			HTML: in.HTML, ABTest: ab, Label: in.Label, Source: in.Source, Force: in.Force, Minify: minify})
		out = campaigns.ComposerSavedContent{Version: saved.Version, SavedAt: saved.SavedAt, ContentHash: saved.ContentHash, SizeBytes: saved.SizeBytes,
			PlainText: saved.PlainText, HTML: saved.HTML}
		return err
	})
	return out, err
}

// abForStore converts the composer's A/B test to the stored shape (the
// same as campaigns.ab_test).
func abForStore(t *campaigns.ComposerABTest) map[string]any {
	variants := []map[string]any{}
	for _, v := range t.Variants {
		m := map[string]any{"key": v.Key}
		if v.Subject != "" {
			m["subject"] = v.Subject
		}
		if v.Preheader != "" {
			m["preheader"] = v.Preheader
		}
		if v.MJML != "" {
			m["mjml"] = v.MJML
		}
		if v.HTML != "" {
			m["html"] = v.HTML
		}
		if v.SendTime != "" {
			m["send_time"] = v.SendTime
		}
		variants = append(variants, m)
	}
	out := map[string]any{"kind": t.Kind, "variants": variants, "test_percent": t.TestPercent, "metric": t.Metric, "duration_hours": t.DurationHours}
	if t.BestTime {
		out["best_time"] = true
	}
	if t.Winner != "" {
		out["winner"] = t.Winner
	}
	return out
}

// chooser builds the chooser of a campaign.
func (s *Service) campaignChooser(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (Chooser, error) {
	settings, err := s.Settings(ctx, q, ws.ID)
	if err != nil {
		return Chooser{}, err
	}
	source := settings.Default
	if row, err := q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: OwnerCampaign, OwnerID: id}); err == nil {
		source = row.Language
	}
	rows, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: OwnerCampaign, OwnerID: id})
	if err != nil {
		return Chooser{}, err
	}
	states := make([]VersionState, 0, len(rows))
	for _, r := range rows {
		states = append(states, VersionState{Language: r.Language, Status: r.Status, Held: r.Held})
	}
	return NewChooser(source, settings, states), nil
}

// Choose picks the version a reader with this contact language gets.
func (c Campaigns) Choose(ctx context.Context, ws *store.Workspace, id uuid.UUID, contactLanguage string) (string, string, error) {
	ch, err := c.S.campaignChooser(ctx, c.S.Q, ws, id)
	if err != nil {
		return "", "", err
	}
	pick := ch.Pick(contactLanguage)
	return pick.Language, pick.Reason, nil
}

// Ready lists the versions that would be sent, the source first.
func (c Campaigns) Ready(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]string, error) {
	ch, err := c.S.campaignChooser(ctx, c.S.Q, ws, id)
	if err != nil {
		return nil, err
	}
	out := []string{ch.Source}
	rows, err := c.S.Q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: OwnerCampaign, OwnerID: id})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Language != ch.Source && ch.Ready(r.Language) {
			out = append(out, r.Language)
		}
	}
	return out, nil
}
