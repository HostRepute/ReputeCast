package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// Accessibility checks (design expansion, M19): the checks on compiled
// HTML for the composer, Send Readiness, automation publishing and the
// template gallery, the workspace policy and the linked pages' titles.

func newAccessibility(a *App) *accessibility.Service {
	s := &accessibility.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log,
		Titles: accessibility.RiverTitles{Client: func() *jobs.Client { return a.Jobs }},
	}
	s.Drafts = func(ctx context.Context, ws *store.Workspace) ([]accessibility.DraftEmail, error) {
		return accessibilityDrafts(ctx, a, s, ws)
	}
	s.Versions = func(ctx context.Context, ws *store.Workspace, id uuid.UUID, current accessibility.Version) ([]accessibility.Version, error) {
		return languageVersions(ctx, a, ws.ID, accessibility.OwnerCampaign, id, 0, current, func() (accessibility.Version, error) {
			c, err := a.Q.AccessibilityCampaign(ctx, store.AccessibilityCampaignParams{WorkspaceID: ws.ID, ID: id})
			return accessibility.Version{MJML: c.Mjml, HTML: c.Html}, err
		})
	}
	return s
}

// storedTranslated is a language version marked translated (content
// languages): with the source, those are the versions that are sent.
const storedTranslated = "translated"

// languageVersions lists an email's language versions for the checks: the
// source, and the versions that are sent (translated, not held, in a
// language the workspace offers). current replaces the version it is (the
// composer's unsaved edits), sent or not; source loads the stored source
// when current isn't it.
func languageVersions(ctx context.Context, a *App, wsID uuid.UUID, kind string, id uuid.UUID, version int32, current accessibility.Version,
	source func() (accessibility.Version, error)) ([]accessibility.Version, error) {
	rows, err := a.Q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: kind, OwnerID: id, OwnerVersion: version})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		current.Source = true
		return []accessibility.Version{current}, nil
	}
	settings, err := languagesService(a).Settings(ctx, a.Q, wsID)
	if err != nil {
		return nil, err
	}
	out := make([]accessibility.Version, 0, len(rows)+1)
	placed := current.Source
	if current.Source {
		out = append(out, current)
	} else {
		src, err := source()
		if err != nil {
			return nil, err
		}
		src.Lang, src.Source = accessibility.EmailLanguage(ctx, a.Q, wsID, kind, id, version), true
		out = append(out, src)
	}
	for _, r := range rows {
		switch {
		case !current.Source && r.Language == current.Lang:
			out, placed = append(out, current), true
		case r.Status == storedTranslated && !r.Held && settings.Offers(r.Language):
			out = append(out, accessibility.Version{Lang: r.Language, MJML: r.Mjml, HTML: r.Html})
		}
	}
	if !placed {
		out = append(out, current)
	}
	return out, nil
}

// accessibilityGate is the automations publish dialog's accessibility check.
type accessibilityGate struct{ s *accessibility.Service }

// CheckDraft implements automations.AccessibilityGate.
func (g accessibilityGate) CheckDraft(ctx context.Context, ws *store.Workspace, d automations.AccessibilityDraft) (automations.AccessibilityVerdict, error) {
	steps := make([]accessibility.DraftStep, 0, len(d.Emails))
	for _, e := range d.Emails {
		steps = append(steps, accessibility.DraftStep{Step: e.Step, MJML: e.MJML, HTML: e.HTML})
	}
	res, err := g.s.CheckDraft(ctx, ws, d.Version, d.LiveVersion, steps)
	out := automations.AccessibilityVerdict{Checks: []automations.AutomationPublishCheck{}, Blocked: res.Blocked, Blocker: res.Blocker}
	for _, b := range res.Badges {
		out.Checks = append(out.Checks, automations.AutomationPublishCheck{Status: b.Status, Label: b.Label})
	}
	return out, err
}

// accessibilityDrafts lists the emails of the workspace's automation
// drafts for the policy's impact preview.
func accessibilityDrafts(ctx context.Context, a *App, s *accessibility.Service, ws *store.Workspace) ([]accessibility.DraftEmail, error) {
	rows, err := a.Q.AccessibilityDraftAutomations(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	auto := &automations.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log}
	var out []accessibility.DraftEmail
	for _, r := range rows {
		emails, err := auto.VersionEmails(ctx, ws.ID, r.ID, r.DraftVersion)
		if err != nil {
			continue
		}
		var live *int
		if r.LiveVersion != nil {
			v := int(*r.LiveVersion)
			live = &v
		}
		for _, e := range emails {
			src := accessibility.Version{Lang: accessibility.EmailLanguage(ctx, a.Q, ws.ID, accessibility.OwnerAutomationEmail, e.EmailID, int32(e.Version)),
				MJML: e.MJML, HTML: e.HTML, Source: true}
			versions, err := languageVersions(ctx, a, ws.ID, accessibility.OwnerAutomationEmail, e.EmailID, int32(e.Version), src, nil)
			if err != nil {
				return nil, err
			}
			out = append(out, accessibility.DraftEmail{AutomationID: r.ID, Automation: r.Name, Version: int(r.DraftVersion), LiveVersion: live,
				Step: accessibility.LowerStep(e.Step), Versions: versions})
		}
	}
	return out, nil
}

// titleFetcher reads linked pages' titles. In development and tests the
// demo's links answer offline, as the link check's do.
func titleFetcher(a *App) accessibility.TitleFetcher {
	var f accessibility.TitleFetcher = accessibility.NewHTTPTitles()
	if a.Cfg != nil && (a.Cfg.IsDevelopment() || a.Cfg.Env == "test") {
		f = demo.AccessibilityTitleOverlay(f)
	}
	return f
}

func init() {
	automationAccessibility = func(a *App) automations.AccessibilityGate { return accessibilityGate{newAccessibility(a)} }
	readinessAccessibility = func(a *App) readiness.AccessibilityChecker { return newAccessibility(a) }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		accessibility.Register(h, newAccessibility(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &accessibility.TitleWorker{Q: a.Q, Fetcher: titleFetcher(a)})
	})
}
