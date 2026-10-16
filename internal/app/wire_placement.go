package app

import (
	"context"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/placement"
	"github.com/hostrepute/reputecast/internal/placement/seedbox"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sso"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Placement tests (design expansion, M24): seed mailboxes the workspace
// owns, tests people start, and where each copy lands.

func (a *App) placement() *placement.Service {
	guard := netguard.Policy{AllowPrivate: a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks}
	return &placement.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
		// Copies go through the normal sending path: the provider's
		// adapter, a messages row (kind placement) and its events.
		Deliver:       a.deliverer(),
		Links:         a.Links,
		URLs:          a.URLs,
		Transactional: placementTransactional{a: a},
		Apps:          placementApps{s: a.ssoService()},
		Readiness:     placementReadiness{a: a},
		Notify:        placementNotifier{a: a},
		LangOf:        func(ctx context.Context, id uuid.UUID) i18n.Lang { return applang.ForUserID(ctx, a.Q, id) },
		Jobs:          func() *jobs.Client { return a.Jobs },
		// Seed mailboxes' IMAP servers can be any host a person types in;
		// Microsoft Graph and the token endpoints are fixed public hosts.
		Guard: guard,
		HTTP:  guard.Client(30 * time.Second),
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		placement.Register(h, a.placement())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		placement.RegisterJobs(a.JobReg, a.placement())
	})
}

// placementTransactional renders a transactional template's live version
// with a test data set, as the API would send it.
type placementTransactional struct{ a *App }

func (t placementTransactional) RenderLive(ctx context.Context, ws *store.Workspace, key string, data map[string]any, to string) (string, string, string, int, error) {
	in := transactional.TransactionalRenderInput{Data: data}
	if to != "" {
		in.To = &transactional.TransactionalAddress{Email: to}
	}
	r, err := transactionalService(t.a).Render(ctx, ws, key, in)
	if err != nil {
		return "", "", "", 0, err
	}
	return r.Subject, r.HTML, r.Text, r.Version, nil
}

// placementApps reads the instance's Google and Microsoft sign-in apps.
type placementApps struct{ s *sso.Service }

func (p placementApps) MailApp(ctx context.Context, kind string) (placement.MailApp, bool, error) {
	id, secret, tenant, ok, err := p.s.MailApp(ctx, kind)
	return placement.MailApp{Kind: kind, ClientID: id, ClientSecret: secret, Tenant: tenant}, ok, err
}

// placementReadiness is a campaign's latest Send Readiness run, for the guidance.
type placementReadiness struct{ a *App }

func (r placementReadiness) Latest(ctx context.Context, _ *store.Workspace, campaignID uuid.UUID) (placement.ReadinessSummary, bool) {
	rs := &readiness.Service{Pool: r.a.Pool, Q: r.a.Q, Log: r.a.Log}
	run, err := rs.Latest(ctx, campaignID)
	if err != nil || run == nil {
		return placement.ReadinessSummary{}, false
	}
	out := placement.ReadinessSummary{Passed: run.Counts.Passed, Warnings: run.Counts.Warnings, Blocks: run.Counts.Blocks,
		Blocking: []string{}, Evidence: []string{}}
	for _, c := range run.Checks {
		if c.Status != "block" {
			continue
		}
		out.Blocking = append(out.Blocking, firstNonEmptyString(c.Label, c.Title))
		out.Evidence = append(out.Evidence, firstNonEmptyString(c.Evidence, c.Title))
	}
	return out, true
}

func firstNonEmptyString(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

// placementNotifier tells the person who started a test that it finished,
// and members with placement:manage that a seed's sign-in failed, each in
// their own language.
type placementNotifier struct{ a *App }

func (n placementNotifier) TestFinished(ctx context.Context, ws *store.Workspace, t store.PlacementTest, summary string) {
	if t.StartedBy == nil {
		return
	}
	l := applang.ForUserID(ctx, n.a.Q, *t.StartedBy)
	nt := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindPlacementFinished, Severity: "ok",
		Title: placement.MFinishedTitle.Text(l, i18n.Args{"name": t.Name}), Body: summary, Summary: summary,
		Link: fmt.Sprintf("/w/%s/deliverability/placement/%s", ws.Slug, t.ID), Data: map[string]any{"test": t.ID.String()}}
	if err := notify.ToUser(ctx, n.a.Pool, *t.StartedBy, nt); err != nil && n.a.Log != nil {
		n.a.Log.WarnContext(ctx, "notify placement test", "err", err)
	}
}

func (n placementNotifier) SeedFailed(ctx context.Context, ws *store.Workspace, seed store.PlacementSeed) {
	ids, err := notify.MembersWith(ctx, n.a.Pool, ws.ID, authz.PlacementManage)
	if err != nil {
		return
	}
	p, _ := seedbox.ProviderOf(seed.Provider)
	for _, id := range ids {
		l := applang.ForUserID(ctx, n.a.Q, id)
		args := i18n.Args{"email": seed.Email, "provider": p.Company}
		body := placement.MSeedFailedBody.Text(l, args)
		if seed.Status == "expired" {
			body = placement.MSeedExpiredBody.Text(l, args)
		}
		_ = notify.ToUser(ctx, n.a.Pool, id, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSeedSignInFailed, Severity: "bad",
			Title: placement.MSeedFailedTitle.Text(l, args), Body: body, Summary: body,
			Link: fmt.Sprintf("/w/%s/deliverability/placement/seeds", ws.Slug), Data: map[string]any{"seed": seed.ID.String()}})
	}
}
