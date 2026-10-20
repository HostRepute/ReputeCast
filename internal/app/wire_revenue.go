package app

import (
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/revenue"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// Revenue and attribution (design expansion phase 6, X17): Earn's
// overview, the campaign report's Revenue section, the Earn card, the
// attribution settings, the conversion API and the click token.

var revenueServices sync.Map // *App → *revenue.Service

func revenueService(a *App) *revenue.Service {
	if v, ok := revenueServices.Load(a); ok {
		return v.(*revenue.Service)
	}
	s := &revenue.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Links: a.Links, URLs: a.URLs, Log: a.Log, Growth: a.growth()}
	v, _ := revenueServices.LoadOrStore(a, s)
	return v.(*revenue.Service)
}

// Revenue is Earn's revenue and attribution (tests set its clock).
func (a *App) Revenue() *revenue.Service { return revenueService(a) }

// trackingHooks run on the click tracker when it's made: other domains
// hear of clicks or add to their destination.
var trackingHooks []func(*App, *tracking.Service)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		revenue.Register(h, revenueService(a))
	})
	trackingHooks = append(trackingHooks, func(a *App, t *tracking.Service) {
		r := revenueService(a)
		t.Clicked, t.Decorate = r.Clicked, r.Decorate
	})
}
