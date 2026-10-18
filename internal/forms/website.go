package forms

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// What the publication website (design expansion phase 7, internal/website)
// reads and changes of the public pages' settings.

// SystemProviderName names the System stream's provider ("Postmark"), or ""
// when System mail has nowhere to go: the sender line of the website's
// sign-in emails.
func (s *Service) SystemProviderName(ctx context.Context, ws *store.Workspace) string {
	return s.systemProvider(ctx, ws)
}

// DefaultSender is the workspace's default sender ("Acme Newsletter",
// "digest@news.acme.io"), empty without one.
func (s *Service) DefaultSender(ctx context.Context, ws *store.Workspace) (name, email string) {
	if r, err := s.Q.FormsDefaultSender(ctx, ws.ID); err == nil {
		return r.Name, r.Email
	}
	return "", ""
}

// SetFooter saves the footer's branding from Website › Home & navigation:
// the privacy policy's address and "Sent with ReputeCast". It writes in q's
// transaction, and the caller audits the change.
func (s *Service) SetFooter(ctx context.Context, q *store.Queries, ws *store.Workspace, poweredBy bool, privacyURL string) error {
	privacy := strings.TrimSpace(privacyURL)
	if privacy != "" {
		u, err := url.Parse(privacy)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return api.Invalid("body.privacyUrl", "Use a full address like https://acme.io/privacy.")
		}
	}
	patch := map[string]any{"poweredBy": poweredBy}
	if privacy != "" {
		patch["privacyUrl"] = privacy
	} else if err := q.FormsRemoveBrandingKeys(ctx, store.FormsRemoveBrandingKeysParams{ID: ws.ID, Keys: []string{"privacyUrl"}}); err != nil {
		return err
	}
	raw, _ := json.Marshal(patch)
	if _, err := q.FormsUpdateBranding(ctx, store.FormsUpdateBrandingParams{ID: ws.ID, Patch: raw}); err != nil {
		return err
	}
	return s.touchSettings(ctx, q, ws)
}
