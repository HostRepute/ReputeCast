package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// A sign-in flow keeps its state, nonce and PKCE verifier server-side for
// ten minutes (auth_challenges, keyed by the hash of the state) and binds
// the state to the browser that started it with a short cookie, so a
// callback link can't be replayed in another browser.
const (
	flowKind      = "oidc_flow"
	flowTTL       = 10 * time.Minute
	flowCookiePfx = "rc_sso_"
)

const (
	modeSignIn = "signin"
	modeTest   = "test"
	modeLink   = "link"
	modeInvite = "invite"
)

type flowState struct {
	ProviderID string `json:"provider_id,omitempty"`
	Social     string `json:"social,omitempty"`
	Mode       string `json:"mode"`
	Nonce      string `json:"nonce"`
	Verifier   string `json:"verifier"`
	Next       string `json:"next,omitempty"`
	LoginHint  string `json:"login_hint,omitempty"`
	// InviteEnc is the invitation token, sealed.
	InviteEnc []byte `json:"invite_enc,omitempty"`
	// Workspace is where an admin's test sign-in returns.
	Workspace string `json:"workspace,omitempty"`
}

// StartRequest is what a start link carries.
type StartRequest struct {
	LoginHint string
	Next      string
	// Invite is an invitation token: the identity joins its workspace.
	Invite string
	// Intent "link" adds the identity to the signed-in account.
	Intent string
}

// Redirect is a browser navigation: where to go and which cookies to set.
type Redirect struct {
	Location string
	Cookies  []http.Cookie
}

func randomString(n int) string {
	const alphabet = "abcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

// reference is the id shown on the error page and written to the audit log.
func reference() string { return "auth_" + randomString(8) }

func flowCookieName(state string) string {
	sum := sha256.Sum256([]byte(state))
	return flowCookiePfx + hex.EncodeToString(sum[:6])
}

func (s *Service) secureCookies(ctx context.Context) bool {
	if strings.HasPrefix(s.publicURL(ctx), "https://") {
		return true
	}
	return s.Cfg != nil && s.Cfg.Auth.CookieSecure && !s.Cfg.IsDevelopment()
}

// beginFlow stores a flow and returns its state and the binding cookie.
func (s *Service) beginFlow(ctx context.Context, fs flowState, userID *uuid.UUID) (string, http.Cookie, error) {
	state, hash := authn.NewToken()
	if _, err := s.Q.CreateChallenge(ctx, store.CreateChallengeParams{
		ID: uuid.Must(uuid.NewV7()), TokenHash: hash, Kind: flowKind, UserID: userID, Data: mustJSON(fs), ExpiresAt: s.now().Add(flowTTL),
	}); err != nil {
		return "", http.Cookie{}, err
	}
	c := http.Cookie{Name: flowCookieName(state), Value: "1", Path: "/auth/", MaxAge: int(flowTTL.Seconds()), //nolint:gosec // G124: Secure follows the public URL and auth.cookie_secure.
		HttpOnly: true, Secure: s.secureCookies(ctx), SameSite: http.SameSiteLaxMode}
	return state, c, nil
}

// takeFlow loads and deletes the flow of a callback's state. It fails when
// the state is unknown, expired, used, or wasn't started in this browser.
func (s *Service) takeFlow(ctx context.Context, state string, cookies []*http.Cookie) (store.AuthChallenge, flowState, error) {
	var fs flowState
	expired := failure("state_expired", "flow.expired", nil)
	if len(state) < 10 || len(state) > 200 {
		return store.AuthChallenge{}, fs, expired
	}
	ch, err := s.Q.GetChallenge(ctx, store.GetChallengeParams{TokenHash: authn.HashToken(state), Kind: flowKind})
	if err != nil {
		return store.AuthChallenge{}, fs, expired
	}
	_ = s.Q.DeleteChallenge(ctx, ch.ID)
	bound := false
	name := flowCookieName(state)
	for _, c := range cookies {
		if c.Name == name {
			bound = true
		}
	}
	if !bound {
		return store.AuthChallenge{}, fs, failure("state_mismatch", "flow.otherBrowser", nil)
	}
	if err := json.Unmarshal(ch.Data, &fs); err != nil {
		return store.AuthChallenge{}, fs, expired
	}
	return ch, fs, nil
}

func clearFlowCookie(state string) http.Cookie {
	return http.Cookie{Name: flowCookieName(state), Value: "", Path: "/auth/", MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true, SameSite: http.SameSiteLaxMode} //nolint:gosec // G124: deletion cookie.
}

// safeNext keeps only same-origin relative paths. Browsers drop tabs and
// newlines from URLs and read a backslash as a slash, so none of them may
// appear: "/<tab>/host" and "/\host" would both lead to "//host".
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || len(next) >= 1000 || strings.ContainsRune(next, '\\') {
		return ""
	}
	for _, r := range next {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return next
}

// errorRedirect sends the browser to the sign-in page's error state and
// records the failure with the reference the page shows. Without a
// provider's name (an unknown provider) the page names single sign-on, in
// the person's language.
func (s *Service) errorRedirect(ctx context.Context, providerName string, providerID *uuid.UUID, method string, fe *flowError, email string) Redirect {
	ref := reference()
	if email == "" {
		email = fe.Email
	}
	shown := providerName
	if providerName == "" {
		providerName, shown = "Single sign-on", say(ctx, "flow.singleSignOn", nil)
	}
	meta := map[string]any{"method": method, "provider": providerName, "reference": ref, "code": fe.Code}
	if providerID != nil {
		meta["provider_id"] = providerID.String()
	}
	target := email
	if target == "" {
		target = "unknown account"
	}
	reason := fe.Detail
	if !fe.Text.IsZero() {
		reason, meta = audit.KeepReason(meta, fe.Text)
	}
	_ = s.Audit.Log(ctx, audit.Event{Action: "signin.rejected", Result: audit.Denied, Reason: reason, TargetType: "user", TargetName: target,
		Actor: &audit.Actor{Type: "system", Name: "System"}, Metadata: meta})
	q := url.Values{}
	if fe.NoAccount && email != "" {
		q.Set("error", "no_account")
		q.Set("email", email)
		return Redirect{Location: "/sign-in?" + q.Encode()}
	}
	if fe.Code == "account_deactivated" && email != "" {
		q.Set("error", "deactivated")
		q.Set("email", email)
		return Redirect{Location: "/sign-in?" + q.Encode()}
	}
	q.Set("error", "idp")
	q.Set("provider", shown)
	q.Set("code", fe.Code)
	q.Set("ref", ref)
	if email != "" {
		q.Set("email", email)
	}
	return Redirect{Location: "/sign-in?" + q.Encode()}
}

func asFlowError(err error) *flowError {
	var fe *flowError
	if errors.As(err, &fe) {
		return fe
	}
	var p *api.Problem
	if errors.As(err, &p) {
		return &flowError{Code: p.Code, Detail: p.Detail}
	}
	return failure("server_error", "flow.broke", nil)
}

// idpError is the error an identity provider sent back to the callback,
// with the first line of its description: "Okta returned access_denied: …".
func idpError(name, code, description string) *flowError {
	return failure(code, "flow.idpError", i18n.Args{"name": name, "error": code, "has": "yes", "description": firstLine(description)})
}

func secretAD(id uuid.UUID) string { return "sso:" + id.String() + ":client_secret" }

// oauthConfig builds the OAuth client of an SSO provider.
func (s *Service) oauthConfig(ctx context.Context, p store.SsoProvider) (*oidc.Provider, *oauth2.Config, error) {
	if p.ClientID == "" || len(p.ClientSecretEnc) == 0 {
		return nil, nil, failure("not_configured", "flow.notConfigured", i18n.Args{"name": p.Name})
	}
	secret, err := s.Box.OpenString(p.ClientSecretEnc, secretAD(p.ID))
	if err != nil {
		return nil, nil, failure("secret_unreadable", "flow.secretUnreadable", nil)
	}
	prov, err := s.provider(ctx, p.IssuerUrl, false)
	if err != nil {
		return nil, nil, failure("discovery_failed", "flow.discoveryFailed", i18n.Args{"issuer": p.IssuerUrl, "error": err.Error()})
	}
	scopes := slices.Clone(p.Scopes)
	if !slices.Contains(scopes, oidc.ScopeOpenID) {
		scopes = append([]string{oidc.ScopeOpenID}, scopes...)
	}
	return prov, &oauth2.Config{ClientID: p.ClientID, ClientSecret: secret, Endpoint: prov.Endpoint(),
		RedirectURL: s.publicURL(ctx) + RedirectPath(p.Slug), Scopes: scopes}, nil
}

func ssoAllowed(st store.InstanceSetting, method string) bool {
	return len(st.AllowedMethods) == 0 || slices.Contains(st.AllowedMethods, method)
}

// StartSSO begins signing in with an SSO provider: the browser goes to the
// IdP with state, nonce and a PKCE challenge.
func (s *Service) StartSSO(ctx context.Context, id uuid.UUID, req StartRequest) Redirect {
	p, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return s.errorRedirect(ctx, "", nil, "sso", failure("unknown_provider", "flow.noProvider", nil), req.LoginHint)
	}
	fail := func(fe *flowError) Redirect { return s.errorRedirect(ctx, p.Name, &p.ID, "sso", fe, req.LoginHint) }
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return fail(asFlowError(err))
	}
	if p.Status != "enabled" {
		return fail(failure("provider_disabled", "flow.notEnabled", i18n.Args{"name": p.Name}))
	}
	if !ssoAllowed(st, "sso") {
		return fail(failure("method_disabled", "flow.ssoOff", nil))
	}
	fs := flowState{ProviderID: p.ID.String(), Mode: modeSignIn, Next: safeNext(req.Next), LoginHint: req.LoginHint}
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
	return s.redirectToIdP(ctx, p, fs, userID, fail)
}

func (s *Service) redirectToIdP(ctx context.Context, p store.SsoProvider, fs flowState, userID *uuid.UUID, fail func(*flowError) Redirect) Redirect {
	_, conf, err := s.oauthConfig(ctx, p)
	if err != nil {
		return fail(asFlowError(err))
	}
	fs.Nonce, fs.Verifier = randomString(32), oauth2.GenerateVerifier()
	state, cookie, err := s.beginFlow(ctx, fs, userID)
	if err != nil {
		return fail(asFlowError(err))
	}
	opts := []oauth2.AuthCodeOption{oidc.Nonce(fs.Nonce), oauth2.S256ChallengeOption(fs.Verifier)}
	if fs.LoginHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", fs.LoginHint))
	}
	if fs.Mode == modeTest {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "login"))
	}
	return Redirect{Location: conf.AuthCodeURL(state, opts...), Cookies: []http.Cookie{cookie}}
}

// CallbackSSO finishes an SSO flow: GET /auth/oidc/{slug}/callback.
func (s *Service) CallbackSSO(ctx context.Context, slug string, query url.Values, cookies []*http.Cookie) Redirect {
	state := query.Get("state")
	out := s.callbackSSO(ctx, slug, state, query, cookies)
	if state != "" {
		out.Cookies = append(out.Cookies, clearFlowCookie(state))
	}
	return out
}

func (s *Service) callbackSSO(ctx context.Context, slug, state string, query url.Values, cookies []*http.Cookie) Redirect {
	p, err := s.Q.SSOProviderBySlug(ctx, slug)
	if err != nil {
		return s.errorRedirect(ctx, "", nil, "sso", failure("unknown_provider", "flow.noCallback", i18n.Args{"path": RedirectPath(slug)}), "")
	}
	ch, fs, err := s.takeFlow(ctx, state, cookies)
	if err != nil {
		return s.errorRedirect(ctx, p.Name, &p.ID, "sso", asFlowError(err), "")
	}
	if fs.ProviderID != p.ID.String() {
		return s.errorRedirect(ctx, p.Name, &p.ID, "sso", failure("state_mismatch", "flow.otherProvider", nil), fs.LoginHint)
	}
	if fs.Mode == modeTest {
		return s.finishTest(ctx, p, ch, fs, query)
	}
	fail := func(fe *flowError, email string) Redirect {
		if email == "" {
			email = fs.LoginHint
		}
		return s.errorRedirect(ctx, p.Name, &p.ID, "sso", fe, email)
	}
	if e := query.Get("error"); e != "" {
		return fail(idpError(p.Name, e, query.Get("error_description")), "")
	}
	if p.Status != "enabled" && fs.Mode != modeLink {
		return fail(failure("provider_disabled", "flow.turnedOff", i18n.Args{"name": p.Name}), "")
	}
	prov, conf, err := s.oauthConfig(ctx, p)
	if err != nil {
		return fail(asFlowError(err), "")
	}
	claims, err := s.exchange(ctx, prov, conf, query.Get("code"), fs.Verifier, fs.Nonce, false)
	if err != nil {
		return fail(asFlowError(err), "")
	}
	id := extract(p, claims)
	mfaNote := twoFAMode(p) == "trusted" || id.amrMFA()
	meta := map[string]any{"provider": p.Name, "provider_id": p.ID.String(), "preset": p.Preset}
	if len(id.AMR) > 0 {
		meta["amr"] = strings.Join(id.AMR, "+")
	}
	switch fs.Mode {
	case modeLink:
		if ch.UserID == nil {
			return fail(failure("state_expired", "flow.signInToLink", nil), id.Email)
		}
		return s.linkIdentity(ctx, *ch.UserID, identityRow{Kind: "sso", ProviderID: &p.ID, Name: p.Name, Subject: id.Subject, Email: id.Email, Username: id.Username}, fail)
	case modeInvite:
		token, err := s.Box.OpenString(fs.InviteEnc, "sso:invite")
		if err != nil {
			return fail(failure("state_expired", "flow.openInvitation", nil), id.Email)
		}
		if id.EmailVerified != nil && !*id.EmailVerified {
			return fail(failure("email_not_verified", "flow.notVerified", i18n.Args{"name": p.Name, "email": id.Email}), id.Email)
		}
		res, err := s.Accounts.AcceptInvitationExternal(ctx, token, accounts.ExternalIdentity{Kind: "sso", ProviderID: &p.ID, ProviderName: p.Name,
			Subject: id.Subject, Email: id.Email, Name: id.Name, Username: id.Username, MFA: mfaNote, Metadata: meta})
		if err != nil {
			return fail(asFlowError(err), id.Email)
		}
		return s.finishSession(res, id.Email, p.Name, fs.Next)
	}
	var u store.User
	var pl signInPlan
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		pl, err = s.plan(ctx, q, p, id)
		if err != nil || pl.Reject != nil {
			return err
		}
		u, err = s.apply(ctx, tx, p, id, pl)
		return err
	})
	if err != nil {
		s.Log.ErrorContext(ctx, "sso sign-in failed", "provider", p.Slug, "err", err)
		return fail(asFlowError(err), id.Email)
	}
	if pl.Reject != nil {
		return fail(pl.Reject, id.Email)
	}
	if len(id.Groups) > 0 {
		meta["groups"] = len(id.Groups)
	}
	res, err := s.Accounts.CompleteExternalSignIn(ctx, accounts.ExternalSignIn{User: u, Method: "sso", MFA: pl.MFA, Remember: true, Metadata: meta})
	if err != nil {
		return fail(asFlowError(err), id.Email)
	}
	return s.finishSession(res, u.Email, p.Name, fs.Next)
}

// finishSession sends the browser on after the session exists, or to the
// two-factor step when the person must confirm their own second factor.
func (s *Service) finishSession(res accounts.ExternalResult, email, provider, next string) Redirect {
	if res.Cookie == nil {
		q := url.Values{}
		q.Set("challenge", res.Challenge)
		q.Set("methods", strings.Join(res.Methods, ","))
		q.Set("email", email)
		q.Set("provider", provider)
		if res.PasskeyHint != "" {
			q.Set("hint", res.PasskeyHint)
		}
		if next != "" {
			q.Set("next", next)
		}
		return Redirect{Location: "/sign-in?" + q.Encode()}
	}
	to := res.Next
	if n := safeNext(next); n != "" {
		to = n
	}
	return Redirect{Location: to, Cookies: []http.Cookie{*res.Cookie}}
}

// identityRow is an identity to link to a signed-in account.
type identityRow struct {
	Kind       string
	ProviderID *uuid.UUID
	Name       string
	Subject    string
	Email      string
	Username   string
	Details    map[string]any
}

// linkIdentity adds an identity to the account that started the flow
// (Account & security › Connected sign-ins › Link).
func (s *Service) linkIdentity(ctx context.Context, userID uuid.UUID, row identityRow, fail func(*flowError, string) Redirect) Redirect {
	u, err := s.Q.GetUser(ctx, userID)
	if err != nil || u.Status != "active" {
		return fail(failure("account_deactivated", "flow.cantLink", nil), row.Email)
	}
	back := s.Accounts.NextPath(ctx, u)
	if strings.HasPrefix(back, "/w/") {
		back = strings.TrimRight(back, "/") + "/account"
	}
	existing, err := s.Q.GetIdentity(ctx, store.GetIdentityParams{Kind: row.Kind, SsoProviderID: row.ProviderID, Subject: row.Subject})
	switch {
	case err == nil && existing.UserID == u.ID:
		return Redirect{Location: back}
	case err == nil:
		return fail(failure("identity_in_use", "flow.identityInUse", i18n.Args{"name": row.Name}), row.Email)
	case !isNoRows(err):
		return fail(asFlowError(err), row.Email)
	}
	if _, err := s.Q.SSOUserIdentityOfKind(ctx, store.SSOUserIdentityOfKindParams{UserID: u.ID, Kind: row.Kind, SsoProviderID: row.ProviderID}); err == nil {
		return fail(failure("already_linked", "flow.alreadyLinked", i18n.Args{"name": row.Name}), row.Email)
	}
	details := row.Details
	if details == nil {
		details = map[string]any{}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.SSOCreateIdentity(ctx, store.SSOCreateIdentityParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, Kind: row.Kind, SsoProviderID: row.ProviderID,
			Subject: row.Subject, Email: row.Email, Username: row.Username, Details: mustJSON(details)}); err != nil {
			return err
		}
		meta := map[string]any{"kind": row.Kind, "provider": row.Name}
		if row.Username != "" {
			meta["username"] = row.Username
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.identity_link", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorOf(u), Metadata: meta})
	})
	if err != nil {
		return fail(asFlowError(err), row.Email)
	}
	return Redirect{Location: back + "?linked=" + url.QueryEscape(row.Kind)}
}
