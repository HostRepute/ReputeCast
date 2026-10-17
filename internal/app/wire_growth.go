package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		growth.Register(h, a.growth())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		growth.RegisterJobs(a.JobReg, a.growth())
	})
}

func (a *App) growth() *growth.Service {
	return &growth.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, SubscribeURL: a.subscribePageURL}
}

// subscribePageURL is the address of a workspace's subscribe page: on its
// verified custom domain, else on the instance.
func (a *App) subscribePageURL(ctx context.Context, ws *store.Workspace) string {
	var domain *string
	if a.Pool != nil {
		_ = a.Pool.QueryRow(ctx, `SELECT custom_domain FROM public_page_settings WHERE workspace_id = $1 AND domain_status = 'verified'`, ws.ID).Scan(&domain)
	}
	if domain != nil && *domain != "" {
		return "https://" + *domain + "/subscribe"
	}
	return a.URLs.Subscribe(ws.Slug, "")
}
