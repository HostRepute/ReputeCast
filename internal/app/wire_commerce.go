package app

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracking"

	// The store platforms. Commerce reaches them through internal/stores
	// only.
	"github.com/hostrepute/reputecast/internal/stores/shopify/shopifytest"
	_ "github.com/hostrepute/reputecast/internal/stores/woocommerce"
	"github.com/hostrepute/reputecast/internal/stores/woocommerce/wootest"
)

// Shopify and WooCommerce (design expansion phase 7, X20): the stores'
// pages and connect dialogs, their webhooks and sync jobs, the Shopify
// app's two addresses, store blocks at send time, the click token on links
// to a store, revenue in reports and automations, and the stores' part of
// the contact profile, the catalog and the content checks.

// The service is shared by the API, the webhooks, the jobs and the hooks of
// one App (tests point its stores at a fake and set its clock).
var commerceServices sync.Map // *App → *commerce.Service

func commerceService(a *App) *commerce.Service {
	if v, ok := commerceServices.Load(a); ok {
		return v.(*commerce.Service)
	}
	s := &commerce.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Links: a.Links, URLs: a.URLs, Log: a.Log,
		Jobs: func() *jobs.Client { return a.Jobs },
		// The demo's store and sample addresses answer in process.
		Demo:        demoStoreClient,
		DemoApprove: func(redirect, state string) string { return demoShopify().Approve(redirect, state) },
		Kick:        func(ctx context.Context) { newAutomations(a).Kick(ctx) },
		Sites: func(ctx context.Context, wsID uuid.UUID, host string) (commerce.PluginSite, bool) {
			return pluginSite(ctx, a, wsID, host)
		},
		PublicCheck: func(ctx context.Context) (bool, time.Time) {
			r := a.integrations().Reachability(ctx)
			at := time.Time{}
			if r.CheckedAt != nil {
				at = *r.CheckedAt
			}
			return r.Reachable, at
		},
	}
	if a.Cfg != nil {
		cfg := a.Cfg
		s.AllowPrivate = func() bool { return cfg.Providers.AllowPrivateNetworks }
	}
	v, _ := commerceServices.LoadOrStore(a, s)
	return v.(*commerce.Service)
}

// Commerce is the stores' domain (tests set its clock and its stores).
func (a *App) Commerce() *commerce.Service { return commerceService(a) }

// The demo's Shopify and WooCommerce: in process, accepting any credentials
// (the demo's store holds a sealed token they never issued) and never
// reaching the internet. The demo's stores are never polled; what a person
// does to them (register webhooks, reconnect) answers from these.
var (
	demoShopify = sync.OnceValue(func() *shopifytest.Fake {
		f := shopifytest.New()
		f.Permissive = true
		return f
	})
	demoWoo = sync.OnceValue(func() *wootest.Fake {
		f := wootest.New()
		f.Permissive = true
		return f
	})
)

func demoStoreClient(platform string) *http.Client {
	if platform == commerce.WooCommerce {
		return &http.Client{Transport: demoWoo().Transport(), Timeout: 10 * time.Second}
	}
	return &http.Client{Transport: demoShopify().Transport(), Timeout: 10 * time.Second}
}

// pluginSite is the connected WordPress site at a store's address, with
// what its plugin reported of WooCommerce.
func pluginSite(ctx context.Context, a *App, wsID uuid.UUID, host string) (commerce.PluginSite, bool) {
	r, err := a.Q.CommercePluginSite(ctx, store.CommercePluginSiteParams{WorkspaceID: wsID, Host: strings.ToLower(host)})
	if err != nil {
		return commerce.PluginSite{}, false
	}
	return commerce.PluginSite{ID: r.IntegrationID, Host: host, PluginVersion: r.PluginVersion, WooCommerce: r.WoocommerceVersion,
		Consent: r.WoocommerceConsent, Checkouts: r.WoocommerceCheckouts, LastReportAt: r.LastReportAt}, true
}

// commerceCatalog is the Commerce group of Settings › Integrations: a card
// per platform, with its store and how its sync stands.
func commerceCatalog(a *App) integrations.GroupSource {
	return func(ctx context.Context, ws *store.Workspace) ([]integrations.IntegrationEntry, error) {
		rows, err := commerceService(a).CatalogEntries(ctx, ws)
		if err != nil {
			return nil, err
		}
		out := make([]integrations.IntegrationEntry, 0, len(rows))
		for _, r := range rows {
			e := integrations.IntegrationEntry{Key: r.Platform, Status: r.Status, ConnectionID: r.ConnectionID, Reasons: []integrations.IntegrationReason{}}
			if r.Stores > 0 {
				e.Facts.Store = &integrations.IntegrationStoreFacts{Stores: r.Stores, Name: r.Name, State: r.State, Orders: r.Orders, LastEventAt: r.LastEventAt}
			}
			out = append(out, e)
		}
		return out, nil
	}
}

func init() {
	sendingStore = func(a *App) engine.StoreContent { return commerceService(a) }
	trackingDecorators = func(a *App) []tracking.LinkDecorator { return []tracking.LinkDecorator{commerceService(a)} }
	trackingMessageVars = func(a *App) func(context.Context, map[string]string) map[string]string {
		return commerceService(a).MessageVars
	}
	audienceStores = func(a *App) audience.ActivitySource { return commerceService(a) }
	automationStoreRevenue = func(a *App) func(context.Context, uuid.UUID, []uuid.UUID, time.Time) map[uuid.UUID]automations.AutomationRevenue {
		return commerceService(a).AutomationRevenue
	}
	integrationCommerce = commerceCatalog
	contentExtras = func(a *App) []contentcheck.Extra { return []contentcheck.Extra{commerceService(a).RemovedProducts} }

	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		commerce.Register(h, commerceService(a))
	})
	// The Shopify app's App URL and redirect URL: browser navigations.
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		commerceService(a).AppRoutes(r)
	})
	// Store webhooks: verified, stored raw, answered at once.
	ingestRegistrars = append(ingestRegistrars, func(a *App, r chi.Router) {
		commerceService(a).Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		commerce.RegisterJobs(a.JobReg, commerceService(a))
	})
}
