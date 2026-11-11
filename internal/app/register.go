package app

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/maintenance"
	"github.com/hostrepute/reputecast/internal/meta"
	"github.com/hostrepute/reputecast/internal/opsmetrics"
)

// Domains plug themselves in from their own file in this package
// (wire_<domain>.go) with an init function that appends to these lists, so
// adding a domain never touches a shared file:
//
//	func init() {
//		apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
//			campaigns.Register(h, &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
//		})
//	}
var (
	apiRegistrars    []func(*App, huma.API)
	publicRegistrars []func(*App, chi.Router)
	ingestRegistrars []func(*App, chi.Router)
	jobRegistrars    []func(*App)
	// startHooks run after dependencies exist and before serving, in file
	// order (wire domains that need each other, such as the system mailer).
	startHooks []func(*App) error
)

// registerAPI adds every domain's Huma operations.
func registerAPI(a *App, h huma.API) {
	meta.Register(h, &meta.Service{Q: a.Q, Cfg: a.Cfg})
	accounts.Register(h, &accounts.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Cfg: a.Cfg, Mail: a.Mail, Log: a.Log,
		Breach: authn.NewBreachChecker(), OnPolicyChange: a.InvalidatePolicy,
		OnPublicURL: func(ctx context.Context, before, after string) { _, _ = a.publicURLSaved(ctx, before, after) },
	})
	for _, r := range apiRegistrars {
		r(a, h)
	}
}

// registerPublic adds server-rendered public pages (subscribe, confirm,
// preferences, unsubscribe, archive) under /p/ and tracking under /t/.
func registerPublic(a *App, r chi.Router) {
	for _, reg := range publicRegistrars {
		reg(a, r)
	}
}

// registerIngest adds provider webhook endpoints under /hooks/.
func registerIngest(a *App, r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(countIngest)
		for _, reg := range ingestRegistrars {
			reg(a, r)
		}
	})
}

// countIngest counts provider webhook requests by channel and result
// (review 2026-11 OP-1).
func countIngest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/hooks/") {
			next.ServeHTTP(w, r)
			return
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		channel := "email"
		switch seg, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/hooks/"), "/"); seg {
		case "sms", "whatsapp", "stripe":
			channel = seg
		case "shopify", "woocommerce":
			channel = "store"
		case "meta-leads":
			channel = "lead_ads"
		}
		result := "stored"
		switch st := ww.Status(); {
		case st == 0 || st < 300:
		case st == http.StatusUnauthorized || st == http.StatusForbidden:
			result = "rejected"
		case st == http.StatusNotFound:
			result = "unknown"
		case st >= 500:
			result = "retry_later"
		default:
			result = "invalid"
		}
		opsmetrics.IngestRequests.WithLabelValues(channel, result).Inc()
	})
}

// registerJobs adds River workers and periodic jobs.
func registerJobs(a *App) {
	maintenance.Register(a.JobReg, a.Pool, a.Log)
	for _, r := range jobRegistrars {
		r(a)
	}
}

// RegisterOperations registers every operation on h without a database, for
// generating the OpenAPI document. Handlers are never called.
func RegisterOperations(h huma.API) {
	registerAPI(&App{}, h)
}
