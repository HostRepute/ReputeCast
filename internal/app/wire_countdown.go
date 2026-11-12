package app

import (
	"context"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/countdown"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// Countdown timers in email (M59 G-21): the Countdown block's image, drawn
// when the email is opened, and its checks in the composer, Send Readiness
// and a flow's publish dialog.

var countdownServices sync.Map // *App → *countdown.Service

func countdownService(a *App) *countdown.Service {
	if v, ok := countdownServices.Load(a); ok {
		return v.(*countdown.Service)
	}
	s := &countdown.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Links: a.Links, URLs: a.URLs, Hosts: a.customHosts(), Log: a.Log}
	v, _ := countdownServices.LoadOrStore(a, s)
	return v.(*countdown.Service)
}

// Countdowns is the countdown domain (tests set its clock).
func (a *App) Countdowns() *countdown.Service { return countdownService(a) }

// countdownReadiness is the Countdown block in Send Readiness: the send's
// start against each countdown's end.
type countdownReadiness struct{ a *App }

// ReadinessRows implements readiness.GrowChecker.
func (r countdownReadiness) ReadinessRows(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, html string, _ map[uuid.UUID]int64, _ int64) []readiness.GrowRow {
	svc := countdownService(r.a)
	now := time.Now()
	if svc.Now != nil {
		now = svc.Now()
	}
	start := now
	if raw, err := r.a.Q.CountdownCampaignSchedule(ctx, store.CountdownCampaignScheduleParams{ID: campaignID, WorkspaceID: ws.ID}); err == nil {
		start = readiness.ParseSchedule(raw).StartAt(now)
	}
	var out []readiness.GrowRow
	for _, row := range svc.ReadinessRows(ctx, ws, campaignID, html, start, func(t time.Time) any { return readiness.ArgDayTime(t) }) {
		out = append(out, readiness.GrowRow{Key: row.Key, Status: row.Status, Title: readiness.Kept(row.Title), Detail: readiness.Kept(row.Detail),
			Label: readiness.Kept(row.Label), Ack: readiness.Kept(row.Ack), Links: []readiness.GrowLink{{Label: readiness.Kept(row.LinkLabel), Href: row.Href}}})
	}
	return out
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		countdown.Register(h, countdownService(a))
	})
	// The image is served with the tracking endpoints, on any host that
	// carries the workspace's links.
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		countdownService(a).Routes(r)
	})
	campaignMore = append(campaignMore, func(a *App) campaigns.MoreChecks { return countdownService(a).ComposerIssues })
	readinessBlocks = append(readinessBlocks, func(a *App) readiness.GrowChecker { return countdownReadiness{a} })
	automationEmailChecks = append(automationEmailChecks, func(a *App) automations.EmailPublishChecker {
		return func(ctx context.Context, ws *store.Workspace, emails []automations.EmailContent) []automations.AutomationPublishCheck {
			contents := make([]string, 0, 2*len(emails))
			for _, e := range emails {
				contents = append(contents, e.MJML, e.HTML)
			}
			var out []automations.AutomationPublishCheck
			for _, label := range countdownService(a).FlowWarnings(ctx, ws, contents...) {
				out = append(out, automations.AutomationPublishCheck{Status: "warn", Label: label})
			}
			return out
		}
	})
}
