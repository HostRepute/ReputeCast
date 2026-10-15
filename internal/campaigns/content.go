package campaigns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	Source string `json:"source,omitempty" enum:"autosave,manual,import,template" default:"autosave"`
	// Force saves on top of a newer version ("Keep mine").
	Force bool `json:"force,omitempty"`
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
}

// conflict is the 409 sent when someone else saved a newer version.
func conflict(version int32, by string, at time.Time, theirs, mine []string) error {
	who := by
	if who == "" {
		who = "Someone"
	}
	p := api.Err(http.StatusConflict, CodeVersionConflict, fmt.Sprintf("%s saved version %d at %s while you were editing.", who, version, at.Format("15:04")))
	return p.WithMeta("version", version, "savedBy", by, "savedAt", at, "theirChanges", theirs, "yourChanges", mine)
}

func validateSubject(subject string) error {
	if strings.ContainsAny(subject, "\r\n") {
		return api.Invalid("body.subject", "The subject must be one line.")
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
	if err := validateSubject(in.Subject); err != nil {
		return ComposerSavedContent{}, err
	}
	if s.translated(ctx, ws, id, lang) {
		if in.ABTest != nil {
			if err := validateAB(in.ABTest); err != nil {
				return ComposerSavedContent{}, err
			}
		}
		c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return ComposerSavedContent{}, api.MapError(err, "Campaign")
		}
		if !editableStatuses[c.Status] {
			return ComposerSavedContent{}, api.Conflict(s.access(ctx, ws, p, c.Status, c.StartedAt).ReadOnlyReason)
		}
		var settings ComposerContentSettings
		_ = json.Unmarshal(c.ContentSettings, &settings)
		out, err := s.Translations.Save(ctx, ws, p, id, lang, in, settings.Minify)
		if err == nil {
			s.changed(ctx, ws, id)
		}
		return out, err
	}
	if in.Source == "" {
		in.Source = "autosave"
	}
	if in.ABTest != nil {
		if err := validateAB(in.ABTest); err != nil {
			return ComposerSavedContent{}, err
		}
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
		if int(cur.Version) != in.BaseVersion && !in.Force {
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
			return conflict(cur.Version, cur.UpdatedByName, at, nonNil(theirs), nonNil(mine))
		}
		var settings ComposerContentSettings
		_ = json.Unmarshal(cur.ContentSettings, &settings)
		src, html := in.MJML, in.HTML
		if html == "" && strings.TrimSpace(src) != "" {
			src = mjml.SetHeadMeta(src, in.Subject, in.Preheader)
			res := mjml.Compile(src)
			if len(res.Errors) > 0 {
				return api.Invalid("body.mjml", "The MJML doesn't compile: "+res.Errors[0].Message)
			}
			html = res.HTML
		}
		if settings.Minify && html != "" {
			html = contentcheck.Minify(html)
		}
		// lang, dir and the layout tables' role (accessibility).
		html = accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerCampaign, id, 0, html)
		if in.ABTest != nil {
			for i := range in.ABTest.Variants {
				in.ABTest.Variants[i].HTML = accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerCampaign, id, 0, in.ABTest.Variants[i].HTML)
			}
		}
		ab := in.ABTest.toDB()
		plain := contentcheck.PlainText(html)
		hash := ContentHash(in.Subject, in.Preheader, src, html, ab)
		version := cur.Version + 1
		now := s.now()
		if err := q.SaveComposerContent(ctx, store.SaveComposerContentParams{WorkspaceID: ws.ID, ID: id, Subject: in.Subject, Preheader: in.Preheader, Mjml: src, Html: html,
			PlainText: plain, AbTest: ab, ContentHash: hash, Version: version, UpdatedBy: &p.UserID, Now: now}); err != nil {
			return err
		}
		label := strings.TrimSpace(in.Label)
		if err := q.InsertComposerVersion(ctx, store.InsertComposerVersionParams{CampaignID: id, Version: version, Subject: in.Subject, Preheader: in.Preheader,
			Mjml: src, Html: html, PlainText: plain, AbTest: ab, SavedBy: &p.UserID, SavedAt: now, Label: label, Source: in.Source, SizeBytes: int32(len(html))}); err != nil {
			return err
		}
		if _, err := q.PruneComposerVersions(ctx, store.PruneComposerVersionsParams{CampaignID: id, Keep: KeepVersions}); err != nil {
			return err
		}
		before := map[string]any{"version": cur.Version}
		after := map[string]any{"version": version, "sizeBytes": len(html)}
		if cur.Subject != in.Subject {
			before["subject"], after["subject"] = cur.Subject, in.Subject
		}
		if cur.Preheader != in.Preheader {
			before["preheader"], after["preheader"] = cur.Preheader, in.Preheader
		}
		if string(cur.AbTest) != string(ab) {
			after["abTest"] = in.ABTest != nil
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.content.save", TargetType: "campaign", TargetID: id.String(),
			Before: before, After: after, Metadata: map[string]any{"label": label, "source": in.Source, "contentHash": hash}}); err != nil {
			return err
		}
		out = ComposerSavedContent{Version: int(version), SavedAt: now, ContentHash: hash, SizeBytes: len(html), PlainText: plain, HTML: html}
		return nil
	})
	if err == nil {
		s.changed(ctx, ws, id)
	}
	return out, err
}

func validateAB(t *ComposerABTest) error {
	keys := map[string]bool{}
	for _, v := range t.Variants {
		if keys[v.Key] {
			return api.Invalid("body.abTest.variants", "Each variant needs its own letter.")
		}
		keys[v.Key] = true
		if err := validateSubject(v.Subject); err != nil {
			return err
		}
	}
	if !keys["A"] || !keys["B"] {
		return api.Invalid("body.abTest.variants", "An A/B test needs at least variants A and B.")
	}
	if t.Kind == "subject" {
		for _, v := range t.Variants {
			if v.Key != "A" && strings.TrimSpace(v.Subject) == "" {
				return api.Invalid("body.abTest.variants", fmt.Sprintf("Write a subject for variant %s.", v.Key))
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
	Source    string    `json:"source" enum:"create,autosave,manual,restore,template,import,duplicate,api"`
	SizeBytes int       `json:"sizeBytes"`
	Current   bool      `json:"current"`
}

// ComposerVersionList is the version history, newest first.
type ComposerVersionList struct {
	Items []ComposerVersion `json:"items"`
	Total int               `json:"total"`
}

// Versions lists the history of a campaign.
func (s *Service) Versions(ctx context.Context, ws *store.Workspace, id uuid.UUID, limit int) (ComposerVersionList, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerVersionList{}, api.MapError(err, "Campaign")
	}
	rows, err := s.Q.ListComposerVersions(ctx, store.ListComposerVersionsParams{CampaignID: id, Lim: int32(limit)})
	if err != nil {
		return ComposerVersionList{}, err
	}
	out := ComposerVersionList{Items: make([]ComposerVersion, 0, len(rows)), Total: int(c.VersionCount)}
	for _, r := range rows {
		out.Items = append(out.Items, ComposerVersion{Version: int(r.Version), SavedAt: r.SavedAt, SavedBy: r.SavedByName, Label: r.Label, Source: r.Source,
			SizeBytes: int(r.SizeBytes), Current: r.Version == c.Version})
	}
	return out, nil
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
		ComposerVersion: ComposerVersion{Version: int(v.Version), SavedAt: v.SavedAt, SavedBy: v.SavedByName, Label: v.Label, Source: v.Source, SizeBytes: int(v.SizeBytes), Current: v.Version == c.Version},
		Subject:         v.Subject, Preheader: v.Preheader, MJML: v.Mjml, HTML: v.Html, ABTest: abFromDB(v.AbTest), Sections: []string{},
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
		ABTest: abFromDB(v.AbTest), Label: fmt.Sprintf("restored v%d", version), Source: "restore"})
}

// Duplicate copies a campaign (any status) into a new draft.
func (s *Service) Duplicate(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (ComposerCampaign, error) {
	return s.Create(ctx, ws, p, ComposerCreateCampaign{FromCampaignID: id.String()})
}
