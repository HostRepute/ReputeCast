package commerce

import (
	"context"
	"net/http"
	"net/url"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
)

// The Shopify app's two addresses, which the merchant pastes into the app
// they create in Shopify's Dev Dashboard: the App URL (Shopify opens it
// when the app is opened or installed from Shopify) and the redirect URL
// (Shopify sends the browser back there after the approval page). Both are
// browser navigations, answered with a redirect into ReputeCast.

// AppRoutes mounts the app's addresses.
func (s *Service) AppRoutes(r chi.Router) {
	r.Get("/integrations/shopify/app", s.serveApp)
	r.Get("/integrations/shopify/callback", s.serveCallback)
}

// storePage is a store's page in ReputeCast, "" when its workspace is gone.
func (s *Service) storePage(ctx context.Context, st store.Store, query url.Values) string {
	ws, err := s.Q.GetWorkspace(ctx, st.WorkspaceID)
	if err != nil || ws.DeletedAt != nil {
		return ""
	}
	out := "/w/" + url.PathEscape(ws.Slug) + "/settings/integrations/stores/" + st.ID.String()
	if len(query) > 0 {
		out += "?" + query.Encode()
	}
	return out
}

// serveCallback finishes Shopify's approval and returns to the connect
// dialog: step 5 for a new store, the store's page for a reconnection, or
// the step to fix ("error" says what went wrong).
func (s *Service) serveCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	st, code, err := s.Installed(ctx, r.URL.Query())
	if err != nil && code == "" {
		s.log().WarnContext(ctx, "shopify approval not finished", "err", err)
		code = "unavailable"
	}
	target := ""
	if st.ID != uuid.Nil {
		q := url.Values{}
		if st.ImportFinishedAt == nil {
			q.Set("connect", "1")
		}
		if code != "" {
			q.Set("connect", "1")
			q.Set("error", code)
		}
		target = s.storePage(ctx, st, q)
	}
	if target == "" {
		target = "/"
	}
	http.Redirect(w, r, target, http.StatusSeeOther) //nolint:gosec // G710: a path built here from the store's workspace and id
}

// serveApp is the App URL. A request Shopify signed for a store's app
// starts the approval when the app isn't installed yet, and otherwise
// opens the store's page; anything else goes to ReputeCast's home.
func (s *Service) serveApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	shop := stores.NormalizeHost(q.Get("shop"))
	target := "/"
	if shopify.ValidShopDomain(shop) {
		rows, err := s.Q.CommerceStoresByDomain(ctx, shop)
		if err == nil {
			for _, st := range rows {
				creds, err := s.credentials(st)
				if err != nil || creds.ClientSecret == "" || !shopify.VerifyQuery(q, creds.ClientSecret, s.now()) {
					continue
				}
				if st.Step == StepInstall && st.Status != StatusDisconnected {
					if to, err := s.approve(ctx, st); err == nil {
						target = to
						break
					}
				}
				if page := s.storePage(ctx, st, nil); page != "" {
					target = page
				}
				break
			}
		}
	}
	http.Redirect(w, r, target, http.StatusSeeOther) //nolint:gosec // G710: the store's page, or Shopify's approval page for its own domain
}

// approve starts the approval from Shopify's side (the merchant installed
// the app from the Dev Dashboard): no member started it.
func (s *Service) approve(ctx context.Context, st store.Store) (string, error) {
	state := randomToken(24) + "~-"
	now := s.now()
	if err := s.Q.CommerceStoreSetState(ctx, store.CommerceStoreSetStateParams{ID: st.ID, OauthState: &state, ExpiresAt: ptr(now.Add(stateTTL)), At: now}); err != nil {
		return "", err
	}
	scopes := append(slices.Clone(shopify.Scopes), shopify.OptionalScope)
	return shopify.AuthorizeURL(st.Domain, st.ClientID, s.CallbackURL(), state, scopes), nil
}
