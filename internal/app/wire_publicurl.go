package app

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	if strings.TrimSpace(saved) != "" {
		return strings.TrimRight(strings.TrimSpace(saved), "/")
	}
	return strings.TrimRight(a.Cfg.HTTP.PublicURL, "/")
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
	if cop := a.cop.Load(); cop != nil {
		trustOrigin(cop, u)
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

// trustOrigin lets the cross-origin check accept browser requests from u's
// origin (behind a proxy the Host header may not name it).
func trustOrigin(cop *http.CrossOriginProtection, u string) {
	if p, err := url.Parse(u); err == nil && p.Scheme != "" && p.Host != "" {
		_ = cop.AddTrustedOrigin(p.Scheme + "://" + p.Host)
	}
}
