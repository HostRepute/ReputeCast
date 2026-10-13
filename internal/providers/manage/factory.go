// Package manage is the sending-providers domain: the provider list with
// health, quota, cost and capabilities (spec 7.3), the health job (outages,
// rejected credentials, webhooks silent, recovery, failover per stream),
// provider actions (re-check, credentials, settings, tracking, webhooks,
// backfill, held broadcasts) and the Adapters tab.
package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Factory builds adapters for connected providers: it decrypts the
// write-only credentials and the webhook secret and hands them to the
// adapter registered for the provider's adapter key.
type Factory struct {
	Box  *secrets.Box
	HTTP *http.Client
}

// CredentialsAD is the associated data sealing a provider's credentials.
func CredentialsAD(p store.Provider) string { return "provider:" + p.ID.String() + ":credentials" }

// WebhookSecretAD is the associated data sealing a provider's webhook secret.
func WebhookSecretAD(p store.Provider) string { return "provider:" + p.ID.String() + ":webhook_secret" }

// Open returns the adapter for a connected provider.
func (f *Factory) Open(_ context.Context, p store.Provider) (providers.Provider, error) {
	d, ok := providers.Lookup(p.Adapter)
	if !ok || d.New == nil {
		return nil, fmt.Errorf("adapter %q is not available in this build", p.Adapter)
	}
	creds, err := f.Credentials(p)
	if err != nil {
		return nil, err
	}
	cfg := providers.Config{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Settings: p.Config, Credentials: creds, HTTPClient: f.client()}
	if len(p.WebhookSecretEnc) > 0 {
		s, err := f.Box.OpenString(p.WebhookSecretEnc, WebhookSecretAD(p))
		if err != nil {
			return nil, fmt.Errorf("webhook secret of %s: %w", p.Name, err)
		}
		cfg.WebhookSecret = s
	}
	return d.New(cfg)
}

// OpenWith builds the adapter with replacement credentials (credential checks).
func (f *Factory) OpenWith(p store.Provider, creds json.RawMessage) (providers.Provider, error) {
	d, ok := providers.Lookup(p.Adapter)
	if !ok || d.New == nil {
		return nil, fmt.Errorf("adapter %q is not available in this build", p.Adapter)
	}
	cfg := providers.Config{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Settings: p.Config, Credentials: creds, HTTPClient: f.client()}
	if len(p.WebhookSecretEnc) > 0 {
		if s, err := f.Box.OpenString(p.WebhookSecretEnc, WebhookSecretAD(p)); err == nil {
			cfg.WebhookSecret = s
		}
	}
	return d.New(cfg)
}

// Credentials decrypts a provider's credentials ({} when none are stored).
func (f *Factory) Credentials(p store.Provider) (json.RawMessage, error) {
	if len(p.CredentialsEnc) == 0 {
		return json.RawMessage(`{}`), nil
	}
	b, err := f.Box.Open(p.CredentialsEnc, CredentialsAD(p))
	if err != nil {
		return nil, fmt.Errorf("credentials of %s: %w", p.Name, err)
	}
	return b, nil
}

// WebhookSecret decrypts a provider's webhook secret.
func (f *Factory) WebhookSecret(p store.Provider) string {
	if len(p.WebhookSecretEnc) == 0 {
		return ""
	}
	s, _ := f.Box.OpenString(p.WebhookSecretEnc, WebhookSecretAD(p))
	return s
}

func (f *Factory) client() *http.Client {
	if f.HTTP != nil {
		return f.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// WebhookURL is the URL a provider posts its events to.
func WebhookURL(urls publicurl.Builder, p store.Provider) string {
	return urls.Base() + "/hooks/" + p.Adapter + "/" + p.WebhookToken
}
