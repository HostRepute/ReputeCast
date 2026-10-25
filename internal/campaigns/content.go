package campaigns

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// KeepVersions is how many autosaved versions a campaign keeps; the first
// version and named saves (⌘S, restores, imports) are always kept.
const KeepVersions = 300

// CodeVersionConflict is the problem code of an autosave based on an old version.
const CodeVersionConflict = "version_conflict"

// ComposerSaveContent is one save of the composer: subject, preheader, MJML and the
// HTML the browser compiled from it, and the A/B variants.
type ComposerSaveContent struct {
	// BaseVersion is the version the editor started from. Saving on top of
	// someone else's newer version fails with 409 unless Force is set.
	BaseVersion int    `json:"baseVersion" minimum:"1"`
	Subject     string `json:"subject,omitempty" maxLength:"998"`
	Preheader   string `json:"preheader,omitempty" maxLength:"500"`
	MJML        string `json:"mjml,omitempty" maxLength:"2000000"`
	// HTML compiled by the composer (mjml-browser). Empty: the server
	// compiles the MJML with gomjml.
	HTML   string          `json:"html,omitempty" maxLength:"2500000"`
	ABTest *ComposerABTest `json:"abTest,omitempty"`
	// Label says what changed ("2-column section padding").
	Label  string `json:"label,omitempty" maxLength:"200"`
	Source string `json:"source,omitempty" enum:"autosave,manual,import,template,recovered" default:"autosave"`
	// Force saves on top of a newer version ("Keep mine").
	Force bool `json:"force,omitempty"`
	// Stash keeps this content in the version history without making it
	// current ("Load theirs" after a conflict): it is saved as a version,
	// then the current content is saved on top of it, unchanged.
	Stash bool `json:"stash,omitempty"`
}

// ComposerSavedContent is the result of a save.
type ComposerSavedContent struct {
	Version     int       `json:"version"`
	SavedAt     time.Time `json:"savedAt"`
	ContentHash string    `json:"contentHash"`
	SizeBytes   int       `json:"sizeBytes"`
	PlainText   string    `json:"plainText"`
	// HTML is what was stored (minified when the campaign asks for it).
	HTML string `json:"html"`
	// Unchanged is true when the content was already the current version's:
	// nothing new was stored and Version is that version.
	Unchanged bool `json:"unchanged,omitempty"`
	// StashedVersion is the version that keeps the stashed content (Stash).
	StashedVersion int `json:"stashedVersion,omitempty"`
}

// conflict is the 409 sent when someone else saved a newer version.
func conflict(l i18n.Lang, version int32, by string, at time.Time, theirs, mine []string) error {
	args := i18n.Args{"name": by, "version": strconv.Itoa(int(version)), "at": at}
	detail := mProblemVersionConflict.Text(l, args)
	if by == "" {
		detail = mProblemVersionConflictBy.Text(l, args)
	}
	p := api.Err(http.StatusConflict, CodeVersionConflict, detail)
	return p.WithMeta("version", version, "savedBy", by, "savedAt", at, "theirChanges", theirs, "yourChanges", mine)
}

func validateSubject(l i18n.Lang, subject string) error {
	if strings.ContainsAny(subject, "\r\n") {
		return api.Invalid("body.subject", mProblemSubjectOneLine.Text(l, nil))
	}
	return nil
}

// Save stores a new version of the content.
func (s *Service) Save(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ComposerSaveContent) (ComposerSavedContent, error) {
	return s.SaveIn(ctx, ws, p, id, "", in)
}

// SaveIn stores a new version of one language version's content ("" or
// the source language: the source).
func (s *Service) SaveIn(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in ComposerSaveContent) (ComposerSavedContent, error) {
	l := i18n.LangFrom(ctx)
	if err := validateSubject(l, in.Subject); err != nil {
		return ComposerSavedContent{}, err
	}
	if in.ABTest != nil {
		if err := validateAB(l, in.ABTest); err != nil {
			return ComposerSavedContent{}, err
		}
	}
	if s.translated(ctx, ws, id, lang) {
		return s.saveTranslation(ctx, ws, p, id, lang, in)
	}
	if in.Source == "" {
		in.Source = "autosave"
	}
	var out ComposerSavedContent
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ComposerLockCampaign(ctx, store.ComposerLockCampaignParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Campaign")
		}
		if !editableStatuses[cur.Status] {
			return api.Conflict(s.access(ctx, ws, p, cur.Status, nil).ReadOnlyReason)
		}
		var settings ComposerContentSettings
		_ = json.Unmarshal(cur.ContentSettings, &settings)
		c, err := s.prepareContent(ctx, q, ws, id, in, settings)
		if err != nil {
			return err
		}
		// Saving what is already the current version stores nothing, and two
		// people saving the same content don't conflict.
		if c.hash == cur.ContentHash && !in.Stash {
			at := cur.UpdatedAt
			if cur.AutosavedAt != nil {
				at = *cur.AutosavedAt
			}
			out = ComposerSavedContent{Version: int(cur.Version), SavedAt: at, ContentHash: cur.ContentHash, SizeBytes: len(cur.Html), PlainText: cur.PlainText, HTML: cur.Html, Unchanged: true}
			return nil
		}
		if int(cur.Version) != in.BaseVersion && !in.Force && !in.Stash {
			return s.conflictFor(ctx, q, l, id, cur, in)
		}
		if in.Stash {
			out, err = s.stash(ctx, q, ws, p, id, cur, c, in)
			return err
		}
		out, err = s.storeVersion(ctx, q, ws, p, id, cur, c, in.Label, in.Source)
		return err
	})
	if err == nil && !out.Unchanged && !in.Stash {
		s.changed(ctx, ws, id)
	}
	return out, err
}

// saveTranslation saves a language version's content (internal/languages).
// Stash is two saves there: yours as a version, then theirs on top.
func (s *Service) saveTranslation(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, lang string, in ComposerSaveContent) (ComposerSavedContent, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerSavedContent{}, api.MapError(err, "Campaign")
	}
	if !editableStatuses[c.Status] {
		return ComposerSavedContent{}, api.Conflict(s.access(ctx, ws, p, c.Status, c.StartedAt).ReadOnlyReason)
	}
	var settings ComposerContentSettings
	_ = json.Unmarshal(c.ContentSettings, &settings)
	if !in.Stash {
		out, err := s.Translations.Save(ctx, ws, p, id, lang, in, settings.Minify)
		if err == nil {
			s.changed(ctx, ws, id)
		}
		return out, err
	}
	theirs, err := s.Translations.Content(ctx, ws, id, lang)
	if err != nil {
		return ComposerSavedContent{}, err
	}
	mine := in
	mine.Stash, mine.Force, mine.Source = false, true, "manual"
	mine.Label = mVersionKeptYours.Text(i18n.Default, i18n.Args{"name": theirs.UpdatedBy, "version": strconv.Itoa(theirs.Version)})
	kept, err := s.Translations.Save(ctx, ws, p, id, lang, mine, settings.Minify)
	if err != nil {
		return ComposerSavedContent{}, err
	}
	back := ComposerSaveContent{BaseVersion: kept.Version, Subject: theirs.Subject, Preheader: theirs.Preheader, MJML: theirs.MJML, HTML: theirs.HTML,
		ABTest: abFromDB(theirs.ABTest), Source: "manual", Label: mVersionLoadedTheirs.Text(i18n.Default, i18n.Args{"name": theirs.UpdatedBy, "version": strconv.Itoa(theirs.Version)})}
	out, err := s.Translations.Save(ctx, ws, p, id, lang, back, settings.Minify)
	if err == nil {
		out.StashedVersion = kept.Version
		s.changed(ctx, ws, id)
	}
	return out, err
}

// preparedContent is a save's content as it will be stored.
type preparedContent struct {
	subject, preheader, mjml, html, plain, hash string
	ab                                          json.RawMessage
	abTest                                      *ComposerABTest
}

// prepareContent turns a save into what is stored: the HTML compiled on
// the server when the browser sent none, minified when the campaign asks
// for it, with lang, dir and the layout tables' role (accessibility).
func (s *Service) prepareContent(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, in ComposerSaveContent, settings ComposerContentSettings) (preparedContent, error) {
	l := i18n.LangFrom(ctx)
	src, html := in.MJML, in.HTML
	if html == "" && strings.TrimSpace(src) != "" {
		src = mjml.SetHeadMeta(src, in.Subject, in.Preheader)
		res := mjml.Compile(src)
		if len(res.Errors) > 0 {
			// The compiler's own message stays as it wrote it.
			return preparedContent{}, api.Invalid("body.mjml", mProblemMJMLCompile.Text(l, i18n.Args{"error": res.Errors[0].Message}))
		}
		html = res.HTML
	}
	if settings.Minify && html != "" {
		html = contentcheck.Minify(html)
	}
	html = accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerCampaign, id, 0, html)
	var abTest *ComposerABTest
	if in.ABTest != nil {
		t := *in.ABTest
		t.Variants = append([]ComposerABVariant(nil), in.ABTest.Variants...)
		for i := range t.Variants {
			t.Variants[i].HTML = accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerCampaign, id, 0, t.Variants[i].HTML)
		}
		abTest = &t
	}
	ab := abTest.toDB()
	return preparedContent{subject: in.Subject, preheader: in.Preheader, mjml: src, html: html, plain: contentcheck.PlainText(html),
		hash: ContentHash(in.Subject, in.Preheader, src, html, ab), ab: ab, abTest: abTest}, nil
}

// conflictFor is the 409 of a save based on an old version: who saved the
// newer one, when, and which sections each side changed.
func (s *Service) conflictFor(ctx context.Context, q *store.Queries, l i18n.Lang, id uuid.UUID, cur store.ComposerLockCampaignRow, in ComposerSaveContent) error {
	var theirs, mine []string
	if base, err := q.GetComposerVersion(ctx, store.GetComposerVersionParams{CampaignID: id, Version: int32(in.BaseVersion)}); err == nil {
		theirs = mjml.ChangedSections(base.Mjml, cur.Mjml)
		mine = mjml.ChangedSections(base.Mjml, in.MJML)
		if base.Subject != cur.Subject {
			theirs = append(theirs, "subject")
		}
		if base.Subject != in.Subject {
			mine = append(mine, "subject")
		}
	}
	at := cur.UpdatedAt
	if cur.AutosavedAt != nil {
		at = *cur.AutosavedAt
	}
	return conflict(l, cur.Version, cur.UpdatedByName, at, nonNil(theirs), nonNil(mine))
}

// storeVersion makes the content the campaign's next version, with its
// history row and its audit entry. cur is the locked campaign.
func (s *Service) storeVersion(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, id uuid.UUID, cur store.ComposerLockCampaignRow, c preparedContent, label, source string) (ComposerSavedContent, error) {
	version := cur.Version + 1
	now := s.now()
	if err := q.SaveComposerContent(ctx, store.SaveComposerContentParams{WorkspaceID: ws.ID, ID: id, Subject: c.subject, Preheader: c.preheader, Mjml: c.mjml, Html: c.html,
		PlainText: c.plain, AbTest: c.ab, ContentHash: c.hash, Version: version, UpdatedBy: &p.UserID, Now: now}); err != nil {
		return ComposerSavedContent{}, err
	}
	label = strings.TrimSpace(label)
	if err := q.InsertComposerVersion(ctx, store.InsertComposerVersionParams{CampaignID: id, Version: version, Subject: c.subject, Preheader: c.preheader,
		Mjml: c.mjml, Html: c.html, PlainText: c.plain, AbTest: c.ab, SavedBy: &p.UserID, SavedAt: now, Label: label, Source: source, SizeBytes: int32(len(c.html))}); err != nil {
		return ComposerSavedContent{}, err
	}
	if _, err := q.PruneComposerVersions(ctx, store.PruneComposerVersionsParams{CampaignID: id, Keep: KeepVersions}); err != nil {
		return ComposerSavedContent{}, err
	}
	before := map[string]any{"version": cur.Version}
	after := map[string]any{"version": version, "sizeBytes": len(c.html)}
	if cur.Subject != c.subject {
		before["subject"], after["subject"] = cur.Subject, c.subject
	}
	if cur.Preheader != c.preheader {
		before["preheader"], after["preheader"] = cur.Preheader, c.preheader
	}
	if string(cur.AbTest) != string(c.ab) {
		after["abTest"] = c.abTest != nil
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.content.save", TargetType: "campaign", TargetID: id.String(),
		Before: before, After: after, Metadata: map[string]any{"label": label, "source": source, "contentHash": c.hash}}); err != nil {
		return ComposerSavedContent{}, err
	}
	return ComposerSavedContent{Version: int(version), SavedAt: now, ContentHash: c.hash, SizeBytes: len(c.html), PlainText: c.plain, HTML: c.html}, nil
}

// stash keeps a save's content in the history as its own version and puts
// the current content back on top ("Load theirs"): the campaign's content
// doesn't change, so neither do its checks or its approval.
func (s *Service) stash(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, id uuid.UUID, cur store.ComposerLockCampaignRow, c preparedContent, in ComposerSaveContent) (ComposerSavedContent, error) {
	theirs := strconv.Itoa(int(cur.Version))
	kept, top := cur.Version+1, cur.Version+2
	now := s.now()
	keptLabel := mVersionKeptYours.Text(i18n.Default, i18n.Args{"name": cur.UpdatedByName, "version": theirs})
	if err := q.InsertComposerVersion(ctx, store.InsertComposerVersionParams{CampaignID: id, Version: kept, Subject: c.subject, Preheader: c.preheader,
		Mjml: c.mjml, Html: c.html, PlainText: c.plain, AbTest: c.ab, SavedBy: &p.UserID, SavedAt: now, Label: keptLabel, Source: "conflict", SizeBytes: int32(len(c.html))}); err != nil {
		return ComposerSavedContent{}, err
	}
	topLabel := mVersionLoadedTheirs.Text(i18n.Default, i18n.Args{"name": cur.UpdatedByName, "version": theirs})
	if err := q.InsertComposerVersion(ctx, store.InsertComposerVersionParams{CampaignID: id, Version: top, Subject: cur.Subject, Preheader: cur.Preheader,
		Mjml: cur.Mjml, Html: cur.Html, PlainText: cur.PlainText, AbTest: cur.AbTest, SavedBy: &p.UserID, SavedAt: now, Label: topLabel, Source: "restore", SizeBytes: int32(len(cur.Html))}); err != nil {
		return ComposerSavedContent{}, err
	}
	if err := q.ComposerBumpVersion(ctx, store.ComposerBumpVersionParams{WorkspaceID: ws.ID, ID: id, Version: top, UpdatedBy: &p.UserID, Now: now}); err != nil {
		return ComposerSavedContent{}, err
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.content.stash", TargetType: "campaign", TargetID: id.String(),
		Before: map[string]any{"version": cur.Version}, After: map[string]any{"version": top, "keptVersion": kept},
		Metadata: map[string]any{"baseVersion": in.BaseVersion, "contentHash": c.hash}}); err != nil {
		return ComposerSavedContent{}, err
	}
	return ComposerSavedContent{Version: int(top), SavedAt: now, ContentHash: cur.ContentHash, SizeBytes: len(cur.Html), PlainText: cur.PlainText, HTML: cur.Html, StashedVersion: int(kept)}, nil
}

func validateAB(l i18n.Lang, t *ComposerABTest) error {
	keys := map[string]bool{}
	for _, v := range t.Variants {
		if keys[v.Key] {
			return api.Invalid("body.abTest.variants", mProblemVariantLetter.Text(l, nil))
		}
		keys[v.Key] = true
		if err := validateSubject(l, v.Subject); err != nil {
			return err
		}
	}
	if !keys["A"] || !keys["B"] {
		return api.Invalid("body.abTest.variants", mProblemVariantsAB.Text(l, nil))
	}
	if t.Kind == "subject" {
		for _, v := range t.Variants {
			if v.Key != "A" && strings.TrimSpace(v.Subject) == "" {
				return api.Invalid("body.abTest.variants", mProblemVariantSubject.Text(l, i18n.Args{"variant": v.Key}))
			}
		}
	}
	return nil
}

// ComposerVersion is one entry of the version history.
type ComposerVersion struct {
	Version   int       `json:"version"`
	SavedAt   time.Time `json:"savedAt"`
	SavedBy   string    `json:"savedBy"`
	Label     string    `json:"label"`
	Source    string    `json:"source" enum:"create,autosave,manual,restore,template,import,duplicate,api,conflict,merge,recovered,synced"`
	SizeBytes int       `json:"sizeBytes"`
	Current   bool      `json:"current"`
	// Name is what a person called the version ("Sent to legal"); named
	// versions are never pruned and can be listed on their own.
	Name string `json:"name"`
}

// ComposerVersionList is one page of the version history, newest first.
type ComposerVersionList struct {
	Items []ComposerVersion `json:"items"`
	// Total counts every version kept; Named the named ones.
	Total int `json:"total"`
	Named int `json:"named"`
	// NextBefore asks for the next (older) page; absent on the last page.
	NextBefore *int `json:"nextBefore,omitempty"`
}

// VersionsPage asks for one page of the history.
type VersionsPage struct {
	Limit int
	// Before lists versions older than this one (0: from the newest).
	Before int
	// Named lists named versions only.
	Named bool
}

// Versions lists the history of a campaign, one page at a time.
func (s *Service) Versions(ctx context.Context, ws *store.Workspace, id uuid.UUID, page VersionsPage) (ComposerVersionList, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerVersionList{}, api.MapError(err, "Campaign")
	}
	if page.Limit <= 0 {
		page.Limit = 50
	}
	var before *int32
	if page.Before > 0 {
		b := int32(page.Before)
		before = &b
	}
	// One more than asked tells whether an older page exists.
	rows, err := s.Q.ComposerVersionsPage(ctx, store.ComposerVersionsPageParams{CampaignID: id, Before: before, NamedOnly: page.Named, Lim: int32(page.Limit + 1)})
	if err != nil {
		return ComposerVersionList{}, err
	}
	named, err := s.Q.ComposerNamedVersionCount(ctx, id)
	if err != nil {
		return ComposerVersionList{}, err
	}
	out := ComposerVersionList{Items: make([]ComposerVersion, 0, len(rows)), Total: int(c.VersionCount), Named: int(named)}
	if len(rows) > page.Limit {
		rows = rows[:page.Limit]
		next := int(rows[len(rows)-1].Version)
		out.NextBefore = &next
	}
	l := i18n.LangFrom(ctx)
	for _, r := range rows {
		out.Items = append(out.Items, ComposerVersion{Version: int(r.Version), SavedAt: r.SavedAt, SavedBy: r.SavedByName, Label: versionLabel(l, r.Label), Source: r.Source,
			SizeBytes: int(r.SizeBytes), Current: r.Version == c.Version, Name: r.Name})
	}
	return out, nil
}

// ComposerNameVersion names a version ("" removes its name).
type ComposerNameVersion struct {
	Name string `json:"name" maxLength:"120" doc:"The version's name; empty removes it"`
}

// NameVersion names one version of the history.
func (s *Service) NameVersion(ctx context.Context, ws *store.Workspace, id uuid.UUID, version int, in ComposerNameVersion) (ComposerVersion, error) {
	name := strings.Join(strings.Fields(in.Name), " ")
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Campaign")
		}
		old, err := q.ComposerNameVersion(ctx, store.ComposerNameVersionParams{CampaignID: id, Version: int32(version), Name: name})
		if err != nil {
			return api.MapError(err, "Version")
		}
		if old == name {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.version.name", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"version": version, "name": old}, After: map[string]any{"version": version, "name": name}})
	})
	if err != nil {
		return ComposerVersion{}, err
	}
	d, err := s.Version(ctx, ws, id, version)
	return d.ComposerVersion, err
}

// ComposerVersionDetail is a version with its content (compare, restore).
type ComposerVersionDetail struct {
	ComposerVersion
	Subject   string          `json:"subject"`
	Preheader string          `json:"preheader"`
	MJML      string          `json:"mjml"`
	HTML      string          `json:"html"`
	ABTest    *ComposerABTest `json:"abTest,omitempty"`
	// Sections are the labels of the sections that differ from the version before.
	Sections []string `json:"sections"`
}

// Version returns one version.
func (s *Service) Version(ctx context.Context, ws *store.Workspace, id uuid.UUID, version int) (ComposerVersionDetail, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerVersionDetail{}, api.MapError(err, "Campaign")
	}
	v, err := s.Q.GetComposerVersion(ctx, store.GetComposerVersionParams{CampaignID: id, Version: int32(version)})
	if err != nil {
		return ComposerVersionDetail{}, api.MapError(err, "Version")
	}
	d := ComposerVersionDetail{
		ComposerVersion: ComposerVersion{Version: int(v.Version), SavedAt: v.SavedAt, SavedBy: v.SavedByName, Label: versionLabel(i18n.LangFrom(ctx), v.Label), Source: v.Source,
			SizeBytes: int(v.SizeBytes), Current: v.Version == c.Version, Name: v.Name},
		Subject: v.Subject, Preheader: v.Preheader, MJML: v.Mjml, HTML: v.Html, ABTest: abFromDB(v.AbTest), Sections: []string{},
	}
	if prev, err := s.Q.GetComposerVersion(ctx, store.GetComposerVersionParams{CampaignID: id, Version: int32(version - 1)}); err == nil {
		d.Sections = mjml.ChangedSections(prev.Mjml, v.Mjml)
	}
	return d, nil
}

// ComposerRestoreVersion makes an old version current again, as a new version.
type ComposerRestoreVersion struct {
	BaseVersion int `json:"baseVersion" minimum:"1"`
}

// Restore copies a version on top of the history.
func (s *Service) Restore(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, version int, in ComposerRestoreVersion) (ComposerSavedContent, error) {
	// The campaign must belong to the workspace before its versions are read.
	if _, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id}); err != nil {
		return ComposerSavedContent{}, api.MapError(err, "Campaign")
	}
	v, err := s.Q.GetComposerVersion(ctx, store.GetComposerVersionParams{CampaignID: id, Version: int32(version)})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ComposerSavedContent{}, api.NotFound("Version")
		}
		return ComposerSavedContent{}, err
	}
	return s.Save(ctx, ws, p, id, ComposerSaveContent{BaseVersion: in.BaseVersion, Subject: v.Subject, Preheader: v.Preheader, MJML: v.Mjml, HTML: v.Html,
		ABTest: abFromDB(v.AbTest), Label: mVersionRestored.Text(i18n.Default, i18n.Args{"version": strconv.Itoa(version)}), Source: "restore"})
}

// Duplicate copies a campaign (any status) into a new draft.
func (s *Service) Duplicate(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (ComposerCampaign, error) {
	return s.Create(ctx, ws, p, ComposerCreateCampaign{FromCampaignID: id.String()})
}
