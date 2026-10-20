package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/visibility"
)

// Conditional content (design expansion, X11): the composer's versions and
// previews, Send Readiness' Conditional content group, and each recipient's
// version when a send runs.

func (a *App) visibility() *visibility.Service {
	return &visibility.Service{Pool: a.Pool, Q: a.Q, Segments: a.segments(), Log: a.Log}
}

// sendConditional plugs conditional content and polls into the sending
// engine.
type sendConditional struct{ a *App }

// Plan implements engine.Conditional.
func (c sendConditional) Plan(ctx context.Context, wsID, campaignID uuid.UUID) (engine.Planner, error) {
	p, err := c.a.visibility().SendPlanner(ctx, wsID, campaignID)
	if err != nil || p == nil {
		return nil, err
	}
	return p, nil
}

// Links implements engine.Conditional: the polls of the content are
// created (or brought up to date) as the send starts.
func (c sendConditional) Links(ctx context.Context, wsID, campaignID uuid.UUID) ([]engine.RecipientLink, error) {
	links, err := c.a.polls().SendLinks(ctx, wsID, campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]engine.RecipientLink, 0, len(links))
	for _, l := range links {
		out = append(out, engine.RecipientLink{Var: l.Var, URL: l.URL})
	}
	// Earn's personal links: upgrade_url and manage_subscription_url.
	paid, err := earnService(c.a).ReaderLinks(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, l := range paid {
		out = append(out, engine.RecipientLink{Var: l.Var, URL: l.URL})
	}
	return out, nil
}

// Cut implements engine.Conditional.
func (sendConditional) Cut(html, hidden string) string { return visibility.Cut(html, hidden) }

// readinessConditionalChecker gives Send Readiness the Conditional content
// group and the poll checks.
type readinessConditionalChecker struct{ a *App }

// ConditionalFacts implements readiness.ConditionalChecker: the versions of
// the saved content, counted (or read from the last count) within the
// audience count's time limit.
func (c readinessConditionalChecker) ConditionalFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) readiness.ConditionalFacts {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	r, err := c.a.visibility().Report(ctx, ws, campaignID, visibility.ReportInput{})
	if err != nil {
		problem := "The versions couldn't be counted in time. Run the checks again."
		if ctx.Err() == nil {
			problem = "The versions couldn't be counted: " + err.Error()
		}
		return readiness.ConditionalFacts{Ran: true, Problem: problem}
	}
	return readiness.ConditionalFacts{Ran: true, Checked: true, Report: r}
}

// PollFacts implements readiness.ConditionalChecker.
func (c readinessConditionalChecker) PollFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src string) readiness.PollFacts {
	return c.a.polls().CampaignReadiness(ctx, ws, campaignID, src)
}

func init() {
	sendingConditional = func(a *App) engine.Conditional { return sendConditional{a} }
	readinessConditional = func(a *App) readiness.ConditionalChecker { return readinessConditionalChecker{a} }
	campaignExtras = func(a *App) campaigns.Extras { return composerExtras{a} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		visibility.Register(h, a.visibility())
	})
}

// browserContent gives the public pages' emails (view online, the archive)
// their version of conditional content and their poll links.
type browserContent struct{ a *App }

// BrowserHTML implements publicpages.Browser.
func (b browserContent) BrowserHTML(ctx context.Context, ws *store.Workspace, campaignID, messageID uuid.UUID, html string) string {
	return b.a.visibility().BrowserHTML(ctx, ws.ID, campaignID, messageID, html)
}

// BrowserLinks implements publicpages.Browser.
func (b browserContent) BrowserLinks(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) map[string]string {
	out := b.a.polls().BrowserLinks(ctx, ws, campaignID)
	return withPaidLinks(ctx, b.a, ws.ID, out)
}

// withPaidLinks adds Earn's links as anyone sees them (the archive, test
// sends): the upgrade page itself, without a reader.
func withPaidLinks(ctx context.Context, a *App, wsID uuid.UUID, links map[string]string) map[string]string {
	paid, err := earnService(a).ReaderLinks(ctx, wsID)
	if err != nil || len(paid) == 0 {
		return links
	}
	if links == nil {
		links = map[string]string{}
	}
	for _, l := range paid {
		links[l.Var] = l.URL(uuid.Nil, uuid.Nil)
	}
	return links
}

// composerExtras gives the composer's test emails their poll links and
// the previewed contact's version of conditional content.
type composerExtras struct{ a *App }

// TestLinks implements campaigns.Extras.
func (c composerExtras) TestLinks(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, email string) (map[string]string, error) {
	links, err := c.a.polls().TestLinks(ctx, ws, campaignID, email)
	if err != nil {
		return nil, err
	}
	return withPaidLinks(ctx, c.a, ws.ID, links), nil
}

// TestVersion implements campaigns.Extras.
func (c composerExtras) TestVersion(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, contactID, html string) string {
	id, err := uuid.Parse(contactID)
	if err != nil {
		return visibility.Cut(html, "")
	}
	hidden, err := c.a.visibility().HiddenFor(ctx, ws.ID, campaignID, id)
	if err != nil {
		return visibility.Cut(html, "")
	}
	return visibility.Cut(html, hidden)
}
