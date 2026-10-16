package app

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// draftLanguages writes a WordPress post draft's languages with the
// content-languages service: with plugin 1.3.0, a post's WPML or Polylang
// translations become the draft's language versions, marked up to date.
type draftLanguages struct{ s *languages.Service }

var _ integrations.DraftLanguages = draftLanguages{}

// Add sets the draft's source language to the post's, then adds each
// translation the workspace offers as a version: a copy of the source,
// saved with the translation's content and marked translated.
func (d draftLanguages) Add(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, source string, versions []integrations.DraftVersion) ([]string, error) {
	settings, err := d.s.Settings(ctx, nil, ws.ID)
	if err != nil {
		return nil, err
	}
	c := languages.Campaigns{S: d.s}
	current, err := c.SourceLanguage(ctx, ws, id)
	if err != nil {
		return nil, err
	}
	if lang := languages.Base(source); lang != "" && lang != current && settings.Offers(lang) {
		_, err := d.s.SetCampaignSource(ctx, ws, p, id, languages.SourceInput{Language: lang})
		switch {
		case err == nil:
			current = lang
		case !declined(err):
			return nil, err
		}
	}
	added := []string{}
	for _, v := range versions {
		lang := languages.Base(v.Language)
		if lang == "" || lang == current || slices.Contains(added, lang) || !settings.Offers(lang) {
			continue
		}
		if _, err := d.s.AddCampaignLanguage(ctx, ws, p, id, languages.AddLanguageInput{Language: lang, Start: languages.StartCopy}); err != nil {
			if declined(err) {
				// The draft has this version already, was sent, or the
				// language can't have versions yet (right to left).
				continue
			}
			return added, err
		}
		if err := d.fill(ctx, ws, p, c, id, lang, v); err != nil {
			// No half-made version: a copy of the source isn't a translation.
			_, _ = d.s.RemoveCampaignLanguage(ctx, ws, p, id, lang)
			if declined(err) {
				continue
			}
			return added, err
		}
		added = append(added, lang)
	}
	return added, nil
}

// fill saves the translation's content in its new version, and marks it
// translated: up to date with the source.
func (d draftLanguages) fill(ctx context.Context, ws *store.Workspace, p *authz.Principal, c languages.Campaigns, id uuid.UUID, lang string, v integrations.DraftVersion) error {
	content, err := c.Content(ctx, ws, id, lang)
	if err != nil {
		return err
	}
	if _, err := c.Save(ctx, ws, p, id, lang, campaigns.ComposerSaveContent{BaseVersion: content.Version, Subject: v.Subject, Preheader: v.Preheader,
		MJML: v.MJML, HTML: v.HTML, Label: v.Label, Source: "api"}, false); err != nil {
		return err
	}
	_, err = d.s.MarkCampaignLanguage(ctx, ws, p, id, lang, true, languages.MarkInput{})
	return err
}

// declined is the languages domain saying no (a 4xx problem): not a
// failure of the post's draft.
func declined(err error) bool {
	var p *api.Problem
	return errors.As(err, &p) && p.Status >= 400 && p.Status < 500
}
