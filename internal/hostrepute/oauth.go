package hostrepute

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/hostrepute/reputecast/internal/config"
)

// CallbackPath is where HostRepute sends the browser back after consent. It
// is fixed per instance (not per workspace) so one redirect URI is
// registered; the state names the workspace.
const CallbackPath = "/api/v1/integrations/hostrepute/callback"

// DemoAPIURL marks a connection served by the built-in demo HostRepute.
const DemoAPIURL = "demo:"

// demoCode is the authorization code the demo consent step returns.
const demoCode = "demo-consent"

// OAuth runs the authorization-code flow with PKCE against HostRepute.
type OAuth struct {
	cfg  oauth2.Config
	demo bool
	// HTTP is used for token exchange and refresh (tests inject a client).
	HTTP *http.Client
}

// NewOAuth builds the OAuth configuration for an instance. publicURL is the
// instance's external base URL; the redirect URI is publicURL+CallbackPath.
func NewOAuth(c config.HostReputeConfig, publicURL string) *OAuth {
	return &OAuth{
		cfg: oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     oauth2.Endpoint{AuthURL: c.AuthURL, TokenURL: c.TokenURL, AuthStyle: oauth2.AuthStyleInParams},
			RedirectURL:  strings.TrimRight(publicURL, "/") + CallbackPath,
			Scopes:       Scopes,
		},
		demo: c.Demo,
	}
}

// RedirectURL is the redirect URI sent to HostRepute.
func (o *OAuth) RedirectURL() string { return o.cfg.RedirectURL }

// NewVerifier returns a fresh PKCE code verifier.
func NewVerifier() string { return oauth2.GenerateVerifier() }

// AuthCodeURL is the consent page on app.hostrepute.com, with the S256 code
// challenge of verifier. In demo mode consent is instant: the URL goes
// straight back to the callback with a demo code.
func (o *OAuth) AuthCodeURL(state, verifier string) string {
	if o.demo {
		return o.cfg.RedirectURL + "?" + url.Values{"code": {demoCode}, "state": {state}}.Encode()
	}
	return o.cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

func (o *OAuth) ctx(ctx context.Context) context.Context {
	if o.HTTP != nil {
		return context.WithValue(ctx, oauth2.HTTPClient, o.HTTP)
	}
	return context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: DefaultTimeout})
}

// Exchange trades the authorization code (and the verifier) for tokens.
func (o *OAuth) Exchange(ctx context.Context, code, verifier string) (*oauth2.Token, error) {
	if o.demo && code == demoCode {
		return demoToken(), nil
	}
	return o.cfg.Exchange(o.ctx(ctx), code, oauth2.VerifierOption(verifier))
}

// Demo reports whether connections made through this OAuth are demo ones.
func (o *OAuth) Demo() bool { return o.demo }

// TokenSource refreshes tok when it expires.
func (o *OAuth) TokenSource(ctx context.Context, tok *oauth2.Token) oauth2.TokenSource {
	if tok.RefreshToken == "" {
		return oauth2.StaticTokenSource(tok)
	}
	return o.cfg.TokenSource(o.ctx(ctx), tok)
}

func demoToken() *oauth2.Token {
	// Placeholders for the offline demo client; they open nothing.
	return &oauth2.Token{AccessToken: "hr_demo_access", RefreshToken: "hr_demo_refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)} //nolint:gosec // G101: not a credential
}
