package app

import (
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/offers"
)

// Offers and the hosted cancel page (design expansion phase 6, X17): signup
// offers by code or link, the retention offer, referral rewards, and their
// coupons in the payment provider.

var offerServices sync.Map // *App → *offers.Service

// wireEarnX17 gives the Earn service being made its offer hook (and, with
// gifts, its one-time payments hook).
func wireEarnX17(a *App, e *earn.Service) {
	v, _ := offerServices.LoadOrStore(a, &offers.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Earn: e, Log: a.Log})
	e.Offers = v.(*offers.Service)
	for _, hook := range earnHooks {
		hook(a, e)
	}
}

// earnHooks add more of X17's hooks to the Earn service as it's made.
var earnHooks []func(*App, *earn.Service)

func offerService(a *App) *offers.Service {
	earnService(a) // makes the offers service with it
	v, _ := offerServices.Load(a)
	return v.(*offers.Service)
}

// Offers is the offers domain (tests set its clock).
func (a *App) Offers() *offers.Service { return offerService(a) }

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		offers.Register(h, offerService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		offers.RegisterJobs(a.JobReg, offerService(a))
	})
}
