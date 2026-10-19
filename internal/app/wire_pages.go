package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/store"
)

// Landing and sales pages (design expansion phase 7, X19): pages built
// with blocks on the publisher's theme, their versions, checks and
// publishing; the shop's switch.

var pagesServices sync.Map // *App → *pages.Service

func pagesService(a *App) *pages.Service {
	if v, ok := pagesServices.Load(a); ok {
		return v.(*pages.Service)
	}
	prober := linkProber(a)
	s := &pages.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log,
		Forms:   a.formsService(),
		Website: websiteService(a),
		Selling: func(ctx context.Context, ws *store.Workspace) (pages.Selling, bool, error) {
			sel, ok, err := earnService(a).Selling(ctx, ws)
			if err != nil || !ok {
				return pages.Selling{}, false, err
			}
			return pages.Selling{Account: sel.Account, Mode: string(sel.Mode), Live: sel.HasLive}, true, nil
		},
		Probe: func(ctx context.Context, u string) (int, error) {
			if st, ok := demo.PagesLinkStatus(u); ok {
				return st, nil
			}
			r := prober.Probe(ctx, []string{u})[u]
			return r.Status, nil
		},
		Approvals: pageApprovals(a),
	}
	v, _ := pagesServices.LoadOrStore(a, s)
	return v.(*pages.Service)
}

// Pages is the pages domain (tests set its clock).
func (a *App) Pages() *pages.Service { return pagesService(a) }

// pageApprovals applies the approval rules to pages (wire_approvals.go
// sets it once approvals know pages).
var pageApprovals = func(*App) pages.Approvals { return nil }

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		pages.Register(h, pagesService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		pages.RegisterJobs(a.JobReg, pagesService(a))
	})
	// The public site draws the pages, the shop, checkout and downloads.
	startHooks = append(startHooks, func(a *App) error {
		h := a.publicPages()
		h.Pages, h.Products = pagesService(a), productsService(a)
		return nil
	})
}
