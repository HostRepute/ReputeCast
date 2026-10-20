package app

import (
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/publicpages"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/web"
)

// Signup forms and public pages.
//
// The forms service keeps state per process (rate-limit buckets, the
// custom-domain lookup cache), so the API, the public pages and the jobs of
// one App share a single instance.
var (
	formsMu       sync.Mutex
	formsServices = map[*App]*forms.Service{}
	formsPages    = map[*App]*publicpages.Handler{}
)

func (a *App) formsService() *forms.Service {
	formsMu.Lock()
	defer formsMu.Unlock()
	if s, ok := formsServices[a]; ok {
		return s
	}
	cfg := a.Cfg
	if cfg == nil { // the OpenAPI command registers operations on an App without config
		d := config.Defaults()
		cfg = &d
	}
	resolver := dnscheck.NewResolver(dnscheck.Options{Servers: cfg.DNS.Servers, Timeout: cfg.DNS.Timeout})
	if cfg.IsDevelopment() {
		// The demo's custom domain (news.acme.io) answers from the demo zone.
		resolver = demo.FormsDNSOverlay(demo.DeliverabilityDNSOverlay(resolver), a.URLs)
	}
	// Turnstile's siteverify is the only outbound call; it goes through the SSRF guard.
	guard := netguard.Policy{AllowPrivate: cfg.Providers.AllowPrivateNetworks}
	s := &forms.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, Audience: a.audience(), Box: a.Secrets,
		Mail:   func() mailer.Sender { return a.Mail },
		Tokens: a.Links, URLs: a.URLs, DNS: resolver,
		Turnstile: &forms.Turnstile{Client: guard.Client(10 * time.Second)},
		Addresses: formAddressChecks{a: a},
	}
	formsServices[a] = s
	return s
}

// publicNotifications is transactional mail's Optional categories on the
// public pages (wire_transactional.go sets it).
var publicNotifications = func(*App) publicpages.Notifications { return nil }

// publicTextTerms writes the text message terms page (wire_texting.go sets it).
var publicTextTerms = func(*App) publicpages.TextTerms { return nil }

// publicPush adds web push to the pages (wire_webpush.go sets it).
var publicPush = func(*App) publicpages.PushPages { return nil }

func (a *App) publicPages() *publicpages.Handler {
	fs := a.formsService()
	pl := a.polls()
	tt := publicTextTerms(a)
	pp := publicPush(a)
	// The publication website (X18), built before the lock: it reads the
	// forms service too.
	site := websiteService(a)
	formsMu.Lock()
	defer formsMu.Unlock()
	if h, ok := formsPages[a]; ok {
		return h
	}
	h := &publicpages.Handler{
		Pool: a.Pool, Q: a.Q, Forms: fs, Audience: a.audience(), Tokens: a.Links, URLs: a.URLs, Log: a.Log,
		// Root paths on hosts that aren't a custom domain stay the web app's.
		SPA:           server.NewSPA(web.Dist()),
		Notifications: publicNotifications(a),
		// The poll answer pages and survey pages; emails in the browser get
		// their version of conditional content and their poll links.
		Polls:   pl,
		Browser: browserContent{a},
		// The upgrade page, its welcome page and the billing links (X16).
		Earn: earnService(a),
		// The publication website's pages (X18).
		Website: site,
		// The text message terms page (SMS).
		TextTerms: tt,
		// The push prompt and the preference center's "Turn off" (X22).
		Push: pp,
	}
	formsPages[a] = h
	return h
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		forms.Register(h, a.formsService())
		publicpages.RegisterAPI(h, a.publicPages())
	})
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		a.publicPages().Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		forms.RegisterJobs(a.JobReg, a.formsService())
	})
}
