// Package sso implements single sign-on over OpenID Connect (add-on spec
// 3.2, 5.11) and social sign-in with Google, Microsoft and GitHub (3.1):
// instance-level provider setup with presets, discovery tests, write-only
// client secrets, claim mapping, just-in-time accounts, group → workspace →
// role mappings, domain enforcement with break-glass Owners, a dry-run test
// sign-in, and the browser flows (start and callback with state, nonce and
// PKCE). Sessions are created by the accounts package exactly like a
// password sign-in.
package sso

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service holds the SSO domain's dependencies.
type Service struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Audit    *audit.Logger
	Box      *secrets.Box
	Cfg      *config.Config
	Log      *slog.Logger
	Accounts *accounts.Service
	// OnPolicyChange drops the cached authentication policy (App.InvalidatePolicy).
	OnPolicyChange func()
	// HTTP talks to identity providers. Nil uses a client with a 10 s timeout.
	HTTP *http.Client
	Now  func() time.Time
	// Endpoints of the social providers; tests point them at a fake server.
	Social SocialEndpoints
}

// SocialEndpoints are the URLs of the social sign-in providers.
type SocialEndpoints struct {
	GoogleIssuer string
	// MicrosoftBase is the Microsoft identity platform; issuers are
	// {base}/{tenant}/v2.0, and the multi-tenant endpoints report a
	// templated issuer ({base}/{tenantid}/v2.0).
	MicrosoftBase  string
	GitHubAuthURL  string
	GitHubTokenURL string
	GitHubAPI      string
}

// DefaultSocialEndpoints are the real providers.
var DefaultSocialEndpoints = SocialEndpoints{
	GoogleIssuer:   "https://accounts.google.com",
	MicrosoftBase:  "https://login.microsoftonline.com",
	GitHubAuthURL:  "https://github.com/login/oauth/authorize",
	GitHubTokenURL: "https://github.com/login/oauth/access_token",
	GitHubAPI:      "https://api.github.com",
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s *Service) social() SocialEndpoints {
	e := s.Social
	d := DefaultSocialEndpoints
	if e.GoogleIssuer == "" {
		e.GoogleIssuer = d.GoogleIssuer
	}
	if e.MicrosoftBase == "" {
		e.MicrosoftBase = d.MicrosoftBase
	}
	if e.GitHubAuthURL == "" {
		e.GitHubAuthURL = d.GitHubAuthURL
	}
	if e.GitHubTokenURL == "" {
		e.GitHubTokenURL = d.GitHubTokenURL
	}
	if e.GitHubAPI == "" {
		e.GitHubAPI = d.GitHubAPI
	}
	return e
}

// publicURL is where people open ReputeCast (instance settings, then
// config), without a trailing slash. Redirect URIs are built from it.
func (s *Service) publicURL(ctx context.Context) string {
	if st, err := s.Q.GetInstanceSettings(ctx); err == nil && st.PublicUrl != "" {
		return strings.TrimRight(st.PublicUrl, "/")
	}
	if s.Cfg != nil {
		return strings.TrimRight(s.Cfg.HTTP.PublicURL, "/")
	}
	return ""
}

// SSOPreset describes an identity provider family: the issuer pattern, scopes,
// claim names and the hints the setup screen shows.
type SSOPreset struct {
	Key  string `json:"key" enum:"google,entra,okta,keycloak,authentik,oidc"`
	Name string `json:"name" example:"Microsoft Entra ID"`
	// Issuer is fixed for Google; IssuerPattern shows the shape for the others.
	Issuer        string   `json:"issuer,omitempty"`
	IssuerPattern string   `json:"issuerPattern"`
	Scopes        []string `json:"scopes"`
	// App names where the redirect URI goes ("the Entra app registration").
	App string `json:"app"`
	// GroupsClaim is false when the IdP sends no groups over OIDC (Google).
	GroupsClaim bool `json:"groupsClaim"`
	// GroupIDs: groups arrive as object IDs that need readable labels (Entra).
	GroupIDs bool `json:"groupIds"`
	// TwoFA is the default two-factor mode: amr or trusted.
	TwoFA string `json:"twoFactor" enum:"amr,trusted"`
	// Hints for the setup screen.
	DiscoveryHint     string `json:"discoveryHint"`
	GroupsHint        string `json:"groupsHint"`
	GroupsMissingHint string `json:"groupsMissingHint"`
	TrustedTwoFA      string `json:"trustedTwoFactor,omitempty" doc:"How the IdP enforces two-factor when it is trusted"`
	DocsURL           string `json:"docsUrl"`
	DocsLabel         string `json:"docsLabel"`
}

// Presets in the order of the setup screen.
var Presets = []SSOPreset{
	{Key: "google", Name: "Google Workspace", Issuer: "https://accounts.google.com", IssuerPattern: "https://accounts.google.com",
		Scopes: []string{"openid", "profile", "email"}, App: "Google Cloud OAuth client", GroupsClaim: false, TwoFA: "trusted",
		DiscoveryHint: "Google's issuer is always https://accounts.google.com.",
		GroupsHint:    "Google doesn't send a groups claim over OIDC. Map people by email domain (JIT default role) or assign roles individually in Team.",
		TrustedTwoFA:  "Trusted from Google (2-Step Verification enforced in Workspace)",
		DocsURL:       "https://developers.google.com/identity/openid-connect/openid-connect", DocsLabel: "Google OIDC docs"},
	{Key: "entra", Name: "Microsoft Entra ID", IssuerPattern: "https://login.microsoftonline.com/{tenant-id}/v2.0",
		Scopes: []string{"openid", "profile", "email"}, App: "Entra app registration", GroupsClaim: true, GroupIDs: true, TwoFA: "amr",
		DiscoveryHint:     "Entra ID issuer URLs use the tenant ID (a GUID) or the verified domain, for example https://login.microsoftonline.com/7c1a2b3d-…/v2.0. Find it under Entra ID → Overview → Tenant ID.",
		GroupsHint:        "Groups arrive as a token claim, not a scope: turn on the groups claim in the app registration's token configuration.",
		GroupsMissingHint: "In the Entra app registration open Token configuration → Add groups claim and include security groups in the ID token. If a user is in more than 200 groups, Entra sends an overage link instead; use app roles then.",
		TrustedTwoFA:      "Trusted from Entra ID (MFA enforced by Conditional Access)",
		DocsURL:           "https://learn.microsoft.com/entra/identity-platform/v2-protocols-oidc", DocsLabel: "Entra OIDC docs"},
	{Key: "okta", Name: "Okta", IssuerPattern: "https://{your-org}.okta.com/oauth2/default",
		Scopes: []string{"openid", "profile", "email", "groups"}, App: "Okta app integration", GroupsClaim: true, TwoFA: "amr",
		DiscoveryHint:     "Okta issuer URLs are your Okta domain, or an authorization server such as https://acme.okta.com/oauth2/default. Find it under Security → API → Authorization Servers.",
		GroupsHint:        "Add a groups claim to the authorization server (filter: matches regex .*) and request the groups scope.",
		GroupsMissingHint: "In Okta open Security → API → your authorization server → Claims and add a groups claim to the ID token, then request the groups scope.",
		TrustedTwoFA:      "Trusted from Okta (MFA enforced by a sign-on policy)",
		DocsURL:           "https://developer.okta.com/docs/guides/customize-tokens-groups-claim/main/", DocsLabel: "Okta OIDC docs"},
	{Key: "keycloak", Name: "Keycloak", IssuerPattern: "https://{host}/realms/{realm}",
		Scopes: []string{"openid", "profile", "email"}, App: "Keycloak client", GroupsClaim: true, TwoFA: "amr",
		DiscoveryHint:     "Keycloak issuer URLs end with the realm, for example https://sso.acme.io/realms/acme.",
		GroupsHint:        "Add a Group Membership mapper to the client (token claim name groups, full group path off).",
		GroupsMissingHint: "In Keycloak open Clients → your client → Client scopes → dedicated scope and add a Group Membership mapper with the claim name groups.",
		TrustedTwoFA:      "Trusted from Keycloak (OTP required by the realm's browser flow)",
		DocsURL:           "https://www.keycloak.org/docs/latest/server_admin/#_oidc", DocsLabel: "Keycloak OIDC docs"},
	{Key: "authentik", Name: "Authentik", IssuerPattern: "https://{host}/application/o/{app-slug}/",
		Scopes: []string{"openid", "profile", "email"}, App: "Authentik provider", GroupsClaim: true, TwoFA: "amr",
		DiscoveryHint:     "Authentik issuer URLs end with the application slug and a slash, for example https://auth.acme.io/application/o/reputecast/.",
		GroupsHint:        "Authentik sends the user's groups in the groups claim of the profile scope.",
		GroupsMissingHint: "In Authentik check that the provider's scope mappings include the default profile mapping, which sends groups.",
		TrustedTwoFA:      "Trusted from Authentik (MFA stage in the authentication flow)",
		DocsURL:           "https://docs.goauthentik.io/docs/add-secure-apps/providers/oauth2/", DocsLabel: "Authentik OIDC docs"},
	{Key: "oidc", Name: "Generic OIDC", IssuerPattern: "https://idp.example.com",
		Scopes: []string{"openid", "profile", "email"}, App: "identity provider's client settings", GroupsClaim: true, TwoFA: "amr",
		DiscoveryHint:     "The issuer is the URL before /.well-known/openid-configuration.",
		GroupsHint:        "Configure your IdP to send the user's groups as a claim in the ID token.",
		GroupsMissingHint: "Configure your IdP to add the user's groups to the ID token, then run the test again.",
		TrustedTwoFA:      "Trusted from the identity provider",
		DocsURL:           "https://openid.net/specs/openid-connect-discovery-1_0.html", DocsLabel: "OIDC discovery spec"},
}

// PresetFor returns the preset with the given key (generic OIDC otherwise).
func PresetFor(key string) SSOPreset {
	for _, p := range Presets {
		if p.Key == key {
			return p
		}
	}
	return Presets[len(Presets)-1]
}

// SSOClaimMap names the claims that fill the account.
type SSOClaimMap struct {
	Email         string `json:"email" maxLength:"64"`
	EmailVerified string `json:"emailVerified" maxLength:"64"`
	Name          string `json:"name" maxLength:"64"`
	Groups        string `json:"groups" maxLength:"64"`
}

// storedClaimMap is the claim_map column ({"email":"email","email_verified":…}).
type storedClaimMap struct {
	Email         string `json:"email,omitempty"`
	EmailVerified string `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Groups        string `json:"groups,omitempty"`
}

func defaultClaims() SSOClaimMap {
	return SSOClaimMap{Email: "email", EmailVerified: "email_verified", Name: "name", Groups: "groups"}
}

func claimsOf(p store.SsoProvider) SSOClaimMap {
	var c storedClaimMap
	_ = json.Unmarshal(p.ClaimMap, &c)
	out := defaultClaims()
	if c.Email != "" {
		out.Email = c.Email
	}
	if c.EmailVerified != "" {
		out.EmailVerified = c.EmailVerified
	}
	if c.Name != "" {
		out.Name = c.Name
	}
	if c.Groups != "" {
		out.Groups = c.Groups
	}
	return out
}

func storeClaims(c SSOClaimMap) json.RawMessage {
	b, _ := json.Marshal(storedClaimMap(c))
	return b
}

// mapping is one stored group → workspace → role mapping.
type mapping struct {
	Group       string      `json:"group"`
	Label       string      `json:"label"`
	WorkspaceID uuid.UUID   `json:"workspace_id"`
	RoleKey     string      `json:"role_key"`
	Scope       authz.Scope `json:"scope"`
}

func mappingsOf(p store.SsoProvider) []mapping {
	var m []mapping
	_ = json.Unmarshal(p.GroupMappings, &m)
	return m
}

// twoFAMode returns how the provider satisfies two-factor.
func twoFAMode(p store.SsoProvider) string {
	if p.TwofaMode != nil && *p.TwofaMode != "" {
		return *p.TwofaMode
	}
	return PresetFor(p.Preset).TwoFA
}

// discovery is the stored result of the last discovery test.
type discovery struct {
	OK        bool      `json:"ok"`
	CheckedAt time.Time `json:"checked_at"`
	URL       string    `json:"url,omitempty"`
	Status    int       `json:"status,omitempty"`
	Millis    int64     `json:"ms,omitempty"`
	Error     string    `json:"error,omitempty"`
	// From the document.
	Issuer   string   `json:"issuer,omitempty"`
	AuthURL  string   `json:"authorization_endpoint,omitempty"`
	TokenURL string   `json:"token_endpoint,omitempty"`
	UserInfo string   `json:"userinfo_endpoint,omitempty"`
	JWKSURL  string   `json:"jwks_uri,omitempty"`
	Algs     []string `json:"id_token_signing_alg_values_supported,omitempty"`
	Keys     int      `json:"keys,omitempty"`
}

func discoveryOf(p store.SsoProvider) *discovery {
	if len(p.Discovery) == 0 || string(p.Discovery) == "null" {
		return nil
	}
	var d discovery
	if json.Unmarshal(p.Discovery, &d) != nil {
		return nil
	}
	return &d
}

// testRun is the stored result of the last test sign-in.
type testRun struct {
	At     time.Time      `json:"at"`
	By     string         `json:"by,omitempty"`
	Claims map[string]any `json:"claims,omitempty"`
	// Error is set when the IdP or the token exchange failed.
	Error string `json:"error,omitempty"`
}

func lastTestOf(p store.SsoProvider) *testRun {
	if len(p.LastTest) == 0 || string(p.LastTest) == "null" {
		return nil
	}
	var t testRun
	if json.Unmarshal(p.LastTest, &t) != nil {
		return nil
	}
	return &t
}

// RedirectPath is where the IdP sends the browser back for a provider.
func RedirectPath(slug string) string { return "/auth/oidc/" + slug + "/callback" }

// SocialRedirectPath is the callback of a social provider.
func SocialRedirectPath(kind string) string { return "/auth/social/" + kind + "/callback" }

func emailDomain(email string) string {
	_, d, _ := strings.Cut(strings.ToLower(email), "@")
	return d
}

// normalizeDomains lower-cases, trims and de-duplicates domains.
func normalizeDomains(in []string) ([]string, error) {
	out := []string{}
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(d), "@")))
		if d == "" {
			continue
		}
		if !validDomain(d) {
			return nil, &domainError{d}
		}
		if !slices.Contains(out, d) {
			out = append(out, d)
		}
	}
	return out, nil
}

type domainError struct{ d string }

func (e *domainError) Error() string { return e.d + " isn't a domain like acme.io." }

func validDomain(d string) bool {
	if len(d) > 253 || !strings.Contains(d, ".") || strings.HasPrefix(d, ".") || strings.HasSuffix(d, ".") {
		return false
	}
	for _, label := range strings.Split(d, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}

func displayName(u store.User) string {
	if strings.TrimSpace(u.Name) != "" {
		return u.Name
	}
	return u.Email
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func ptr[T any](v T) *T { return &v }
