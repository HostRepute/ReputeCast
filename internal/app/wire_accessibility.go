package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
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
	return s
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
	lang := accessibility.ContentLanguage(ctx, ws)
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
			out = append(out, accessibility.DraftEmail{AutomationID: r.ID, Automation: r.Name, Version: int(r.DraftVersion), LiveVersion: live,
				Step: accessibility.LowerStep(e.Step), Versions: []accessibility.Version{{Lang: lang, MJML: e.MJML, HTML: e.HTML, Source: true}}})
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
