package app

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
)

// An instance admin can save a public URL (Instance admin › Instance &
// updates); it then replaces http.public_url for every link ReputeCast
// builds. Every process re-reads it every publicURLRefresh; the process that
// saves it switches at once and moves provider webhooks to it.

// publicURLRefresh is how often a process re-reads the saved public URL.
const publicURLRefresh = 30 * time.Second

func init() {
	startHooks = append(startHooks, func(a *App) error {
		a.loadPublicURL(context.Background())
		return nil
	})
}

// effectivePublicURL is the saved public URL, else http.public_url (always
// http.public_url when the server locks it).
func (a *App) effectivePublicURL(saved string) string {
	return publicurl.Effective(saved, a.Cfg.HTTP.PublicURL, a.Cfg.HTTP.PublicURLLocked)
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
// processes that didn't save it (workers, other web processes) follow. It
// first records an address the server locks (PinPublicURL), and tries again
// at each refresh until that works (a schema not migrated yet).
func (a *App) followPublicURL(ctx context.Context) {
	pinned := a.pin(ctx)
	t := time.NewTicker(publicURLRefresh)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if !pinned {
				pinned = a.pin(ctx)
			}
			a.loadPublicURL(ctx)
		}
	}
}

func (a *App) pin(ctx context.Context) bool {
	if err := a.PinPublicURL(ctx); err != nil {
		a.Log.WarnContext(ctx, "record the locked public URL", "err", err)
		return false
	}
	return true
}

// PinPublicURL makes the address the server locks (http.public_url_locked)
// the saved public URL when another one is saved, and moves provider
// webhooks to it when an admin had saved another address. So turning the
// lock on, or the operator changing the locked address, takes effect at the
// next start without anyone saving it in Instance admin. One process wins
// the update, so webhooks move once; the change is audited as the server
// configuration's.
func (a *App) PinPublicURL(ctx context.Context) error {
	if !a.Cfg.HTTP.PublicURLLocked {
		return nil
	}
	to := publicurl.Effective("", a.Cfg.HTTP.PublicURL, true)
	st, err := a.Q.GetInstanceSettings(ctx)
	if err != nil {
		return err
	}
	if publicurl.Same(st.PublicUrl, to) {
		return nil
	}
	n, err := a.Q.InstancePinPublicURL(ctx, store.InstancePinPublicURLParams{ToUrl: to, FromUrl: st.PublicUrl})
	if err != nil || n == 0 {
		return err // n == 0: another process recorded it
	}
	a.setPublicURL(to)
	ev := audit.Event{Action: "instance.public_url_pinned", TargetType: "instance", TargetName: st.Name,
		Actor:  &audit.Actor{Type: authz.ActorSystem, Name: "Server configuration"},
		Reason: "http.public_url_locked", Before: map[string]any{"public_url": st.PublicUrl}, After: map[string]any{"public_url": to}}
	if st.PublicUrl != "" {
		// Links used the saved address until now: webhooks move with them.
		mv, err := a.providersService().PublicURLMoved(ctx, to)
		manual := make([]string, 0, len(mv.Manual))
		for _, m := range mv.Manual {
			manual = append(manual, m.Workspace+" › "+m.Provider+": "+m.URL)
		}
		ev.Metadata = map[string]any{"webhooks_reregistering": mv.Reregistering, "webhooks_manual": manual}
		if err != nil {
			a.Log.WarnContext(ctx, "move provider webhooks to the locked public URL", "err", err)
		}
		if len(manual) > 0 {
			a.Log.WarnContext(ctx, "provider webhooks to change by hand for the locked public URL", "url", to, "webhooks", manual)
		}
	}
	return a.Audit.Log(ctx, ev)
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
