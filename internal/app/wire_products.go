package app

import (
	"context"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Digital products (design expansion phase 7, X19): files sold through the
// payment connection's Stripe Checkout, delivered by a transactional
// email, downloaded from the publication's site.

// One service per App: the API, the public pages, the payment webhook's
// consumer and the jobs share it (tests set its clock).
var productsServices sync.Map // *App → *products.Service

func productsService(a *App) *products.Service {
	if v, ok := productsServices.Load(a); ok {
		return v.(*products.Service)
	}
	cfg := a.Cfg
	if cfg == nil { // the OpenAPI command registers operations without config
		d := config.Defaults()
		cfg = &d
	}
	s := &products.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, URLs: a.URLs, Links: a.Links,
		Files:         products.Store{Dir: cfg.FilesDir()},
		Forms:         a.formsService(),
		Earn:          func() *earn.Service { return earnService(a) },
		Transactional: func() *transactional.Service { return transactionalService(a) },
		Audience:      func() *audience.Service { return a.audience() },
		Jobs:          func() *jobs.Client { return a.Jobs },
		Flows:         productFlows{},
		Media:         &media.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		SiteOn: func(ctx context.Context, ws *store.Workspace) bool {
			on, _ := websiteService(a).IsOn(ctx, ws)
			return on
		},
	}
	v, _ := productsServices.LoadOrStore(a, s)
	return v.(*products.Service)
}

// Products is the products domain (tests set its clock).
func (a *App) Products() *products.Service { return productsService(a) }

// productFlows makes a refunded buyer leave their "Bought a product" flows.
type productFlows struct{}

func (productFlows) Refunded(ctx context.Context, tx pgx.Tx, wsID, contactID, productID uuid.UUID, at time.Time) error {
	_, err := automations.ExitRefunded(ctx, store.New(tx), wsID, contactID, productID, at)
	return err
}

// purchasesActivity reads a contact's purchases for the timeline, building
// the products service only when asked (the audience service is built
// while the forms service is, and the products service needs both).
type purchasesActivity struct{ a *App }

func (p purchasesActivity) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	return productsService(p.a).ContactActivity(ctx, ws, contactID, before, beforeID, limit)
}

func init() {
	// A contact's timeline shows what they bought and refunds.
	audiencePurchases = func(a *App) audience.ActivitySource { return purchasesActivity{a} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		products.Register(h, productsService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		products.RegisterJobs(a.JobReg, productsService(a))
	})
	// Product payments arrive on the payment connection's webhook: the
	// products domain takes them before the paid-member logic.
	startHooks = append(startHooks, func(a *App) error {
		earn.RegisterPaymentConsumer(earnService(a), productsService(a).Consume)
		return nil
	})
}
