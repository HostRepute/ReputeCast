package app

import (
	"context"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/coupons"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// Discount codes (M59 G-22): pools and their API, a code per message at
// send time (coupons.* and the Discount code block), the composer's and
// Send Readiness's checks, the publish badges and the email step's wait,
// store orders matched to their codes, the jobs and the erasure source.

var couponServices sync.Map // *App → *coupons.Service

func couponService(a *App) *coupons.Service {
	if v, ok := couponServices.Load(a); ok {
		return v.(*coupons.Service)
	}
	s := &coupons.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
		Jobs: func() *jobs.Client { return a.Jobs },
		Discounts: func(ctx context.Context, ws *store.Workspace, storeID uuid.UUID) (stores.DiscountCreator, error) {
			return commerceService(a).Discounts(ctx, ws, storeID)
		},
		Sources: func(ctx context.Context, ws *store.Workspace) ([]coupons.StoreSource, error) {
			rows, err := commerceService(a).DiscountSources(ctx, ws)
			if err != nil {
				return nil, err
			}
			out := make([]coupons.StoreSource, 0, len(rows))
			for _, r := range rows {
				out = append(out, coupons.StoreSource{StoreID: r.StoreID, Platform: r.Platform, Name: r.Name, Domain: r.Domain, Currency: r.Currency,
					Can: r.Can, Reason: r.Reason, Detail: r.Detail})
			}
			return out, nil
		}}
	v, _ := couponServices.LoadOrStore(a, s)
	return v.(*coupons.Service)
}

// Coupons is the discount codes domain (tests set its clock).
func (a *App) Coupons() *coupons.Service { return couponService(a) }

// couponReadiness is Send Readiness's view of the pools a campaign shows
// codes from.
type couponReadiness struct{ a *App }

// ReadinessRows implements readiness.GrowChecker.
func (c couponReadiness) ReadinessRows(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, html string, _ map[uuid.UUID]int64, recipients int64) []readiness.GrowRow {
	var out []readiness.GrowRow
	at := func(t time.Time) any { return readiness.ArgDayTime(t) }
	for _, r := range couponService(c.a).ReadinessRows(ctx, ws, campaignID, html, recipients, at) {
		row := readiness.GrowRow{Key: coupons.IssuePool, Status: r.Status, Title: readiness.Kept(r.Title), Detail: readiness.Kept(r.Detail),
			Label: readiness.Kept(r.Label), Short: readiness.Kept(r.Short)}
		for _, l := range r.Links {
			row.Links = append(row.Links, readiness.GrowLink{Label: readiness.Kept(l.Label), Href: l.Href})
		}
		out = append(out, row)
	}
	return out
}

func init() {
	recipientDataRegistrars = append(recipientDataRegistrars, func(a *App) liquid.RecipientData { return couponService(a) })
	campaignMore = append(campaignMore, func(a *App) campaigns.MoreChecks { return couponService(a).ComposerIssues })
	campaignTokens = append(campaignTokens, func(a *App) campaigns.TokenSource { return couponService(a).ComposerTokens })
	readinessBlocks = append(readinessBlocks, func(a *App) readiness.GrowChecker { return couponReadiness{a} })
	automationEmailChecks = append(automationEmailChecks, func(a *App) automations.EmailPublishChecker { return couponService(a).PublishChecks })
	automationEmailHolds = append(automationEmailHolds, func(a *App) automations.EmailHold { return couponService(a).EmailHold })
	commerceRedeemed = func(a *App) func(context.Context, *store.Queries, uuid.UUID, uuid.UUID, []string, time.Time) error {
		return couponService(a).Redeemed
	}
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) { coupons.Register(h, couponService(a)) })
	jobRegistrars = append(jobRegistrars, func(a *App) { coupons.RegisterJobs(a.JobReg, couponService(a)) })
}
