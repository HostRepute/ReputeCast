package app

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/hostrepute/reputecast/internal/publicurl"
)

// The instance's public URL. An instance admin can save one (Instance admin ›
// Instance & updates); it then replaces http.public_url for every link
// ReputeCast builds: tracking, unsubscribe and preference links, public
// pages, shared reports, provider webhooks, sign-in links and system email.
// Every process loads it at start and re-reads it every publicURLRefresh;
// the process that saves it switches at once (instanceService's
// OnPublicURL) and moves provider webhooks to it.

// publicURLRefresh is how often a process re-reads the saved public URL.
const publicURLRefresh = 30 * time.Second

func init() {
	startHooks = append(startHooks, func(a *App) error {
		a.loadPublicURL(context.Background())
		return nil
	})
}

// effectivePublicURL is the saved public URL, else http.public_url.
func (a *App) effectivePublicURL(saved string) string {
	return publicurl.Effective(saved, a.Cfg.HTTP.PublicURL)
}

// loadPublicURL applies the saved public URL and reports whether it changed.
func (a *App) loadPublicURL(ctx context.Context) bool {
	st, err := a.Q.GetInstanceSettings(ctx)
	if err != nil {
		return false // keep the current one (a database without migrations, a blip)
	}
	return a.setPublicURL(st.PublicUrl)
}

// setPublicURL makes saved (or http.public_url when empty) the public URL
// and reports whether it changed.
func (a *App) setPublicURL(saved string) bool {
	u := a.effectivePublicURL(saved)
	if !a.PublicURL.Set(u) {
		return false
	}
	a.Log.Info("public URL", "url", u)
	return true
}

// followPublicURL re-reads the saved public URL until ctx ends, so the
// processes that didn't save it (workers, other web processes) follow.
func (a *App) followPublicURL(ctx context.Context) {
	t := time.NewTicker(publicURLRefresh)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.loadPublicURL(ctx)
		}
	}
}

// currentPublicURL reads the public URL from the database, for work that
// must not use a value up to publicURLRefresh old (moving webhooks).
func (a *App) currentPublicURL(ctx context.Context) string {
	if st, err := a.Q.GetInstanceSettings(ctx); err == nil {
		return a.effectivePublicURL(st.PublicUrl)
	}
	return a.URLs.Base()
}

// crossOrigin is Go's cross-origin protection, trusting the origin of the
// public URL as it is now and of http.public_url (behind a proxy the Host
// header may name neither). An address the instance has moved away from
// stops being trusted at once.
func (a *App) crossOrigin(cop *http.CrossOriginProtection) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := cop.Check(r); err != nil && !a.trustedOrigin(r.Header.Get("Origin")) {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// trustedOrigin reports whether origin is the public URL's or the configured one's.
func (a *App) trustedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, u := range []string{a.URLs.Base(), a.Cfg.HTTP.PublicURL} {
		if p, err := url.Parse(u); err == nil && p.Scheme != "" && p.Host != "" && p.Scheme+"://"+p.Host == origin {
			return true
		}
	}
	return false
}
