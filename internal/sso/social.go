package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Social sign-in (add-on spec 3.1): Google and Microsoft over OIDC, GitHub
// over OAuth 2 with an optional organization restriction. It signs in
// people who already have an account (invited) and never creates one on
// its own; an invitation link can be accepted with it.

// SocialKinds in display order.
var SocialKinds = []string{"google", "microsoft", "github"}

// SocialConfig is one provider in instance_settings.social_providers.
// Client secrets live in social_secrets_enc, never here.
type SocialConfig struct {
	Enabled  bool   `json:"enabled"`
	ClientID string `json:"client_id,omitempty"`
	// Org restricts GitHub sign-in to members of an organization.
	Org string `json:"org,omitempty"`
	// Tenant is Microsoft's: common (personal and work), organizations,
	// consumers, or one tenant id or domain.
	Tenant      string     `json:"tenant,omitempty"`
	SecretSetAt *time.Time `json:"secret_set_at,omitempty"`
}

// SocialConfigs reads the social providers of the instance settings.
func SocialConfigs(st store.InstanceSetting) map[string]SocialConfig {
	m := map[string]SocialConfig{}
	_ = json.Unmarshal(st.SocialProviders, &m)
	return m
}

const socialSecretsAD = "instance:social_providers:secrets"

func (s *Service) socialSecrets(st store.InstanceSetting) map[string]string {
	m := map[string]string{}
	if len(st.SocialSecretsEnc) == 0 {
		return m
	}
	b, err := s.Box.Open(st.SocialSecretsEnc, socialSecretsAD)
	if err != nil {
		return m
	}
	_ = json.Unmarshal(b, &m)
	return m
}

// SocialReady reports what's missing before a social provider can be on.
func (s *Service) SocialReady(st store.InstanceSetting, kind string) string {
	c := SocialConfigs(st)[kind]
	if c.ClientID == "" || s.socialSecrets(st)[kind] == "" {
		return socialName(kind) + " needs a client ID and secret before it can be turned on."
	}
	return ""
}

// SSOSocialProvider is a social sign-in provider as the admin sees it.
type SSOSocialProvider struct {
	Kind        string     `json:"kind" enum:"google,microsoft,github"`
	Name        string     `json:"name"`
	Enabled     bool       `json:"enabled"`
	Configured  bool       `json:"configured"`
	ClientID    string     `json:"clientId"`
	SecretSet   bool       `json:"secretSet"`
	SecretSetAt *time.Time `json:"secretSetAt,omitempty"`
	Org         string     `json:"org,omitempty"`
	Tenant      string     `json:"tenant,omitempty"`
	RedirectURI string     `json:"redirectUri"`
	// Linked is how many people have linked this provider.
	Linked int `json:"linked"`
}

// SocialProviders lists the three social providers.
func (s *Service) SocialProviders(ctx context.Context) ([]SSOSocialProvider, error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return nil, err
	}
	cfgs, secrets := SocialConfigs(st), s.socialSecrets(st)
	counts := map[string]int{}
	rows, err := s.Pool.Query(ctx, `SELECT kind, count(*) FROM user_identities WHERE kind <> 'sso' GROUP BY kind`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		counts[k] = n
	}
	base := s.publicURL(ctx)
	out := make([]SSOSocialProvider, 0, len(SocialKinds))
	for _, k := range SocialKinds {
		c := cfgs[k]
		out = append(out, SSOSocialProvider{Kind: k, Name: socialName(k), Enabled: c.Enabled, Configured: c.ClientID != "" && secrets[k] != "",
			ClientID: c.ClientID, SecretSet: secrets[k] != "", SecretSetAt: c.SecretSetAt, Org: c.Org, Tenant: c.Tenant,
			RedirectURI: base + SocialRedirectPath(k), Linked: counts[k]})
	}
	return out, rows.Err()
}

// SSOSocialInput configures a social provider. The secret is write-only.
type SSOSocialInput struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"clientId" maxLength:"300"`
	ClientSecret string `json:"clientSecret,omitempty" maxLength:"2000"`
	Org          string `json:"org,omitempty" maxLength:"100"`
	Tenant       string `json:"tenant,omitempty" maxLength:"100"`
}

// SaveSocial stores a social provider's settings and secret.
func (s *Service) SaveSocial(ctx context.Context, kind string, in SSOSocialInput) ([]SSOSocialProvider, error) {
	if !slices.Contains(SocialKinds, kind) {
		return nil, api.NotFound("Social provider")
	}
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return nil, err
	}
	cfgs, secrets := SocialConfigs(st), s.socialSecrets(st)
	before := cfgs[kind]
	c := before
	c.Enabled = in.Enabled
	c.ClientID = strings.TrimSpace(in.ClientID)
	if kind == "github" {
		org := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(in.Org), "@"))
		if org != "" && !validGitHubOrg(org) {
			return nil, api.Invalid("body.org", "GitHub organization names use letters, digits and single hyphens, like acme-eng.")
		}
		c.Org = org
	}
	if kind == "microsoft" {
		t := strings.TrimSpace(in.Tenant)
		if t != "" && (len(t) > 100 || strings.ContainsAny(t, "/?#: ")) {
			return nil, api.Invalid("body.tenant", "Use common, organizations, consumers or a tenant ID.")
		}
		c.Tenant = t
	}
	replaced := false
	if v := strings.TrimSpace(in.ClientSecret); v != "" {
		secrets[kind] = v
		now := s.now()
		c.SecretSetAt = &now
		replaced = true
	}
	if c.Enabled && (c.ClientID == "" || secrets[kind] == "") {
		return nil, api.Invalid("body.clientId", socialName(kind)+" needs a client ID and secret before it can be turned on.")
	}
	cfgs[kind] = c
	var enc []byte
	if len(secrets) > 0 {
		b, _ := json.Marshal(secrets)
		if enc, err = s.Box.Seal(b, socialSecretsAD); err != nil {
			return nil, err
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InstanceAdminSaveSocial(ctx, store.InstanceAdminSaveSocialParams{SocialProviders: mustJSON(cfgs), SocialSecretsEnc: enc}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "social.provider.updated", TargetType: "social_provider", TargetID: kind, TargetName: socialName(kind),
			Before: socialAuditView(before), After: socialAuditView(c), Metadata: map[string]any{"secret_replaced": replaced}})
	})
	if err != nil {
		return nil, err
	}
	s.forget(s.social().GoogleIssuer)
	if s.OnPolicyChange != nil {
		s.OnPolicyChange()
	}
	return s.SocialProviders(ctx)
}

func socialAuditView(c SocialConfig) map[string]any {
	return map[string]any{"enabled": c.Enabled, "client_id": c.ClientID, "org": c.Org, "tenant": c.Tenant}
}

func validGitHubOrg(s string) bool {
	if len(s) > 39 || strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") || strings.Contains(s, "--") {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// microsoftIssuer is the discovery base of a Microsoft tenant: common
// (personal and work accounts), organizations, consumers, or one tenant.
func (s *Service) microsoftIssuer(tenant string) string {
	if tenant == "" {
		tenant = "common"
	}
	return strings.TrimRight(s.social().MicrosoftBase, "/") + "/" + tenant + "/v2.0"
}

// socialClient builds the OAuth client of a social provider. For Google
// and Microsoft it also returns the OIDC provider that verifies ID tokens.
func (s *Service) socialClient(ctx context.Context, kind string, st store.InstanceSetting) (*oidc.Provider, *oauth2.Config, error) {
	c := SocialConfigs(st)[kind]
	secret := s.socialSecrets(st)[kind]
	if c.ClientID == "" || secret == "" {
		return nil, nil, &flowError{Code: "not_configured", Detail: socialName(kind) + " sign-in has no client ID or secret yet."}
	}
	conf := &oauth2.Config{ClientID: c.ClientID, ClientSecret: secret, RedirectURL: s.publicURL(ctx) + SocialRedirectPath(kind)}
	e := s.social()
	switch kind {
	case "google":
		p, err := s.provider(ctx, e.GoogleIssuer, false)
		if err != nil {
			return nil, nil, &flowError{Code: "discovery_failed", Detail: "Google's discovery document couldn't be loaded: " + err.Error()}
		}
		conf.Endpoint, conf.Scopes = p.Endpoint(), []string{oidc.ScopeOpenID, "email", "profile"}
		return p, conf, nil
	case "microsoft":
		issuer := s.microsoftIssuer(c.Tenant)
		p, err := s.provider(ctx, issuer, true)
		if err != nil {
			return nil, nil, &flowError{Code: "discovery_failed", Detail: "Microsoft's discovery document couldn't be loaded: " + err.Error()}
		}
		conf.Endpoint, conf.Scopes = p.Endpoint(), []string{oidc.ScopeOpenID, "email", "profile"}
		return p, conf, nil
	case "github":
		conf.Endpoint = oauth2.Endpoint{AuthURL: e.GitHubAuthURL, TokenURL: e.GitHubTokenURL, AuthStyle: oauth2.AuthStyleInParams}
		conf.Scopes = []string{"read:user", "user:email"}
		if c.Org != "" {
			conf.Scopes = append(conf.Scopes, "read:org")
		}
		return nil, conf, nil
	}
	return nil, nil, &flowError{Code: "unknown_provider", Detail: "Unknown sign-in provider."}
}

// StartSocial begins a social sign-in (or links one, or joins with an invitation).
func (s *Service) StartSocial(ctx context.Context, kind string, req StartRequest) Redirect {
	name := socialName(kind)
	fail := func(fe *flowError) Redirect { return s.errorRedirect(ctx, name, nil, kind, fe, req.LoginHint) }
	if !slices.Contains(SocialKinds, kind) {
		return fail(&flowError{Code: "unknown_provider", Detail: "Unknown sign-in provider."})
	}
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return fail(asFlowError(err))
	}
	if !SocialConfigs(st)[kind].Enabled || !ssoAllowed(st, "social") {
		return fail(&flowError{Code: "provider_disabled", Detail: name + " sign-in is turned off on this instance."})
	}
	_, conf, err := s.socialClient(ctx, kind, st)
	if err != nil {
		return fail(asFlowError(err))
	}
	fs := flowState{Social: kind, Mode: modeSignIn, Next: safeNext(req.Next), LoginHint: req.LoginHint, Verifier: oauth2.GenerateVerifier()}
	var userID *uuid.UUID
	switch {
	case req.Invite != "":
		enc, err := s.Box.SealString(req.Invite, "sso:invite")
		if err != nil {
			return fail(asFlowError(err))
		}
		fs.Mode, fs.InviteEnc = modeInvite, enc
	case req.Intent == "link":
		u := api.UserFrom(ctx)
		if u == nil {
			return Redirect{Location: "/sign-in"}
		}
		fs.Mode, userID = modeLink, &u.ID
	}
	opts := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(fs.Verifier)}
	if kind != "github" {
		fs.Nonce = randomString(32)
		opts = append(opts, oidc.Nonce(fs.Nonce))
		if fs.LoginHint != "" {
			opts = append(opts, oauth2.SetAuthURLParam("login_hint", fs.LoginHint))
		}
	} else if fs.LoginHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login", fs.LoginHint))
	}
	state, cookie, err := s.beginFlow(ctx, fs, userID)
	if err != nil {
		return fail(asFlowError(err))
	}
	return Redirect{Location: conf.AuthCodeURL(state, opts...), Cookies: []http.Cookie{cookie}}
}

// CallbackSocial finishes a social flow: GET /auth/social/{kind}/callback.
func (s *Service) CallbackSocial(ctx context.Context, kind string, query url.Values, cookies []*http.Cookie) Redirect {
	state := query.Get("state")
	out := s.callbackSocial(ctx, kind, state, query, cookies)
	if state != "" {
		out.Cookies = append(out.Cookies, clearFlowCookie(state))
	}
	return out
}

func (s *Service) callbackSocial(ctx context.Context, kind, state string, query url.Values, cookies []*http.Cookie) Redirect {
	name := socialName(kind)
	ch, fs, err := s.takeFlow(ctx, state, cookies)
	fail := func(fe *flowError, email string) Redirect {
		if email == "" {
			email = fs.LoginHint
		}
		return s.errorRedirect(ctx, name, nil, kind, fe, email)
	}
	if err != nil {
		return fail(asFlowError(err), "")
	}
	if fs.Social != kind {
		return fail(&flowError{Code: "state_mismatch", Detail: "This callback belongs to another provider."}, "")
	}
	if e := query.Get("error"); e != "" {
		return fail(&flowError{Code: e, Detail: name + " returned " + e + ": " + firstLine(query.Get("error_description"))}, "")
	}
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return fail(asFlowError(err), "")
	}
	prov, conf, err := s.socialClient(ctx, kind, st)
	if err != nil {
		return fail(asFlowError(err), "")
	}
	var id identity
	switch kind {
	case "github":
		id, err = s.githubIdentity(ctx, conf, query.Get("code"), fs.Verifier, SocialConfigs(st)[kind].Org)
	default:
		var claims map[string]any
		claims, err = s.exchange(ctx, prov, conf, query.Get("code"), fs.Verifier, fs.Nonce, kind == "microsoft")
		if err == nil {
			id, err = s.oidcSocialIdentity(kind, claims)
		}
	}
	if err != nil {
		return fail(asFlowError(err), "")
	}
	meta := map[string]any{"provider": name}
	if id.Username != "" {
		meta["username"] = id.Username
	}
	switch fs.Mode {
	case modeLink:
		if ch.UserID == nil {
			return fail(&flowError{Code: "state_expired", Detail: "Sign in again to link an identity."}, id.Email)
		}
		return s.linkIdentity(ctx, *ch.UserID, identityRow{Kind: kind, Name: name, Subject: id.Subject, Email: id.Email, Username: id.Username, Details: id.Details}, fail)
	case modeInvite:
		token, err := s.Box.OpenString(fs.InviteEnc, "sso:invite")
		if err != nil {
			return fail(&flowError{Code: "state_expired", Detail: "Open the invitation link again."}, id.Email)
		}
		res, err := s.Accounts.AcceptInvitationExternal(ctx, token, accounts.ExternalIdentity{Kind: kind, ProviderName: name, Subject: id.Subject,
			Email: id.Email, Name: id.Name, Username: id.Username, Details: mustJSON(id.Details), Metadata: meta})
		if err != nil {
			return fail(asFlowError(err), id.Email)
		}
		return s.finishSession(res, id.Email, name, fs.Next)
	}
	// Sign in: an identity already linked, or an account with this
	// verified email (the identity is linked on the way).
	var u store.User
	linked := false
	ident, err := s.Q.GetIdentity(ctx, store.GetIdentityParams{Kind: kind, Subject: id.Subject})
	switch {
	case err == nil:
		if u, err = s.Q.GetUser(ctx, ident.UserID); err != nil {
			return fail(asFlowError(err), id.Email)
		}
		_ = s.Q.SSOTouchIdentity(ctx, store.SSOTouchIdentityParams{ID: ident.ID, Email: id.Email, Username: id.Username, Details: mustJSON(id.Details)})
	case isNoRows(err):
		if id.EmailVerified == nil || !*id.EmailVerified {
			return fail(&flowError{Code: "no_account", Detail: "no account is linked to this " + name + " sign-in", NoAccount: true, Email: id.Email}, id.Email)
		}
		if u, err = s.Q.GetUserByEmail(ctx, id.Email); err != nil {
			if isNoRows(err) {
				return fail(&flowError{Code: "no_account", Detail: "no account for " + id.Email + "; " + name + " sign-in never creates one", NoAccount: true, Email: id.Email}, id.Email)
			}
			return fail(asFlowError(err), id.Email)
		}
		linked = true
	default:
		return fail(asFlowError(err), id.Email)
	}
	if u.Status != "active" {
		return fail(&flowError{Code: "account_deactivated", Detail: u.Email + " is deactivated.", Email: u.Email}, u.Email)
	}
	// Domain enforcement turns social sign-in off too: go to the IdP.
	if enf := s.enforcer(ctx, u.Email); enf != nil && !u.BreakGlass {
		r := s.StartSSO(ctx, enf.ID, StartRequest{LoginHint: u.Email, Next: fs.Next})
		return r
	}
	if linked {
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if _, err := q.SSOCreateIdentity(ctx, store.SSOCreateIdentityParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, Kind: kind, Subject: id.Subject,
				Email: id.Email, Username: id.Username, Details: mustJSON(id.Details)}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.identity_link", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email,
				Actor: actorOf(u), Metadata: map[string]any{"kind": kind, "provider": name, "by": "verified email"}})
		})
		if err != nil {
			return fail(asFlowError(err), u.Email)
		}
	}
	res, err := s.Accounts.CompleteExternalSignIn(ctx, accounts.ExternalSignIn{User: u, Method: kind, Remember: true, Metadata: meta})
	if err != nil {
		return fail(asFlowError(err), u.Email)
	}
	return s.finishSession(res, u.Email, name, fs.Next)
}

// enforcer returns the enabled SSO provider that enforces the email's domain.
func (s *Service) enforcer(ctx context.Context, email string) *store.SsoProvider {
	d := emailDomain(email)
	ps, err := s.Q.SSOProviders(ctx)
	if err != nil {
		return nil
	}
	for _, p := range ps {
		if p.Status == "enabled" && slices.Contains(p.EnforceDomains, d) {
			return &p
		}
	}
	return nil
}

// oidcSocialIdentity reads Google's and Microsoft's ID token claims.
func (s *Service) oidcSocialIdentity(kind string, c map[string]any) (identity, error) {
	id := identity{Kind: kind, ProviderName: socialName(kind), Claims: c, Details: map[string]any{}}
	id.Subject = claimString(c, "sub")
	id.Email = strings.ToLower(strings.TrimSpace(claimString(c, "email")))
	id.Name = claimString(c, "name")
	switch kind {
	case "google":
		if v, ok := claimBool(c, "email_verified"); ok {
			id.EmailVerified = &v
		}
		if hd := claimString(c, "hd"); hd != "" {
			id.Details["hd"] = hd
		}
	case "microsoft":
		// Tokens from the multi-tenant endpoint carry the person's tenant:
		// the issuer must be that tenant's.
		tid := claimString(c, "tid")
		base := strings.TrimRight(s.social().MicrosoftBase, "/")
		if iss := claimString(c, "iss"); tid == "" || iss != base+"/"+tid+"/v2.0" {
			return id, &flowError{Code: "invalid_issuer", Detail: "The Microsoft token's issuer doesn't match its tenant."}
		}
		id.Username = claimString(c, "preferred_username")
		if id.Email == "" && strings.Contains(id.Username, "@") {
			id.Email = strings.ToLower(id.Username)
		}
		// Microsoft's email claim isn't verified: it never links an account
		// by email (an invitation link or Account › Link does).
		id.Details["tid"] = tid
	}
	if id.Subject == "" {
		return id, &flowError{Code: "no_subject", Detail: "The ID token has no sub claim."}
	}
	return id, nil
}

// githubIdentity exchanges the code and reads the GitHub user, their
// primary verified email and, when restricted, their organization
// membership.
func (s *Service) githubIdentity(ctx context.Context, conf *oauth2.Config, code, verifier, org string) (identity, error) {
	id := identity{Kind: "github", ProviderName: "GitHub", Details: map[string]any{}}
	cctx := context.WithValue(ctx, oauth2.HTTPClient, s.client())
	tok, err := conf.Exchange(cctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return id, &flowError{Code: "token_exchange_failed", Detail: "The GitHub token exchange failed: " + oauthError(err)}
	}
	api := strings.TrimRight(s.social().GitHubAPI, "/")
	var user struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
	}
	if status, err := s.githubGet(ctx, tok, api+"/user", &user); err != nil || status != http.StatusOK || user.ID == 0 {
		return id, &flowError{Code: "github_api", Detail: fmt.Sprintf("GitHub didn't return the user (HTTP %d).", status)}
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if status, err := s.githubGet(ctx, tok, api+"/user/emails", &emails); err != nil || status != http.StatusOK {
		return id, &flowError{Code: "github_api", Detail: fmt.Sprintf("GitHub didn't return the email addresses (HTTP %d). The user:email scope is needed.", status)}
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			id.Email = strings.ToLower(e.Email)
		}
	}
	if id.Email == "" {
		return id, &flowError{Code: "email_not_verified", Detail: "Your GitHub account has no verified primary email."}
	}
	t := true
	id.EmailVerified = &t
	id.Subject = fmt.Sprint(user.ID)
	id.Username = user.Login
	id.Name = user.Name
	if org != "" {
		var m struct {
			State string `json:"state"`
		}
		status, err := s.githubGet(ctx, tok, api+"/user/memberships/orgs/"+url.PathEscape(org), &m)
		switch {
		case err != nil:
			return id, &flowError{Code: "github_api", Detail: "GitHub didn't answer the organization check: " + err.Error()}
		case status == http.StatusOK && m.State == "active":
			id.Details["org"] = org
		case status == http.StatusForbidden:
			return id, &flowError{Code: "org_restricted", Detail: "The GitHub organization " + org + " restricts third-party apps. An owner of " + org + " must approve ReputeCast's OAuth app.", Email: id.Email}
		default:
			return id, &flowError{Code: "not_in_org", Detail: "GitHub sign-in is limited to members of the organization " + org + ".", Email: id.Email}
		}
	}
	return id, nil
}

func (s *Service) githubGet(ctx context.Context, tok *oauth2.Token, u string, v any) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := s.client().Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(b, v); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}
