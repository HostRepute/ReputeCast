package hostrepute

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"golang.org/x/oauth2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// JobInserter enqueues River jobs inside a transaction (*jobs.Client).
type JobInserter interface {
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// AssetStatus is what Deliverability knows about an asset, shown next to it
// in Settings › Integrations.
type AssetStatus struct {
	Sources              int
	Listings             []string
	PostmasterReputation string
	DMARCPolicy          string
	DMARCAligned         *float64
}

// Service manages a workspace's HostRepute connection.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Cfg   *config.Config
	Log   *slog.Logger
	Jobs  JobInserter
	// PublicURL is the instance's public URL (the OAuth redirect is built on
	// it); nil uses http.public_url.
	PublicURL func() string
	// HTTP carries HostRepute API and OAuth calls; tests inject one.
	HTTP *http.Client
	Now  func() time.Time
	// Status adds Deliverability's findings to the asset rows (wired by the app).
	Status func(ctx context.Context, workspaceID uuid.UUID) (map[string]AssetStatus, error)
	// Fallback queues Deliverability's built-in checks when HostRepute data
	// stops coming (disconnect, token rejected, HostRepute unreachable).
	Fallback func(ctx context.Context, tx pgx.Tx, workspaceID uuid.UUID, trigger string) error
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) oauth() *OAuth {
	pub := s.Cfg.HTTP.PublicURL
	if s.PublicURL != nil {
		pub = s.PublicURL()
	}
	o := NewOAuth(s.Cfg.HostRepute, pub)
	o.HTTP = s.HTTP
	return o
}

// apiURL is the API base new connections use.
func (s *Service) apiURL() string {
	if s.Cfg.HostRepute.Demo {
		return DemoAPIURL
	}
	return strings.TrimRight(s.Cfg.HostRepute.APIURL, "/")
}

func tokenAD(ws uuid.UUID, kind string) string { return "hostrepute:" + ws.String() + ":" + kind }

// tokenWatch remembers the last token handed out, so a refreshed OAuth
// token can be saved after a sync.
type tokenWatch struct {
	mu   sync.Mutex
	src  oauth2.TokenSource
	last *oauth2.Token
}

func (w *tokenWatch) Token() (*oauth2.Token, error) {
	t, err := w.src.Token()
	if err == nil {
		w.mu.Lock()
		w.last = t
		w.mu.Unlock()
	}
	return t, err
}

func (w *tokenWatch) latest() *oauth2.Token {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.last
}

// demoClient returns the demo HostRepute for what the connection monitors
// (monitoring changes are stored on the connection, like a real sync).
func (s *Service) demoClient(_ uuid.UUID, monitored []string) *DemoClient {
	return NewDemoClient(s.now, monitored)
}

// clientFor builds the client of a stored connection.
func (s *Service) clientFor(ctx context.Context, conn store.HostreputeConnection) (Client, *tokenWatch, error) {
	if conn.ApiUrl == DemoAPIURL {
		return s.demoClient(conn.WorkspaceID, monitoredNames(conn.Monitored)), nil, nil
	}
	access, err := s.Box.OpenString(conn.TokenEnc, tokenAD(conn.WorkspaceID, "token"))
	if err != nil {
		return nil, nil, fmt.Errorf("open HostRepute token: %w", err)
	}
	var src oauth2.TokenSource
	tok := &oauth2.Token{AccessToken: access, TokenType: "Bearer"}
	if conn.Method == "oauth" && len(conn.RefreshEnc) > 0 {
		refresh, err := s.Box.OpenString(conn.RefreshEnc, tokenAD(conn.WorkspaceID, "refresh"))
		if err != nil {
			return nil, nil, fmt.Errorf("open HostRepute refresh token: %w", err)
		}
		tok.RefreshToken = refresh
		if conn.TokenExpiresAt != nil {
			tok.Expiry = *conn.TokenExpiresAt
		}
		src = s.oauth().TokenSource(context.WithoutCancel(ctx), tok)
	} else {
		src = oauth2.StaticTokenSource(tok)
	}
	w := &tokenWatch{src: src}
	c := NewHTTPClient(conn.ApiUrl, w, s.HTTP)
	if conn.Method == "oauth" {
		c.RevokeURL, c.ClientID = s.Cfg.HostRepute.RevokeURL, s.Cfg.HostRepute.ClientID
	}
	return c, w, nil
}

// saveRefreshedToken stores a token the OAuth library refreshed during a call.
func (s *Service) saveRefreshedToken(ctx context.Context, conn store.HostreputeConnection, w *tokenWatch) {
	if w == nil || conn.Method != "oauth" {
		return
	}
	t := w.latest()
	if t == nil {
		return
	}
	old, err := s.Box.OpenString(conn.TokenEnc, tokenAD(conn.WorkspaceID, "token"))
	if err != nil || old == t.AccessToken {
		return
	}
	access, err := s.Box.SealString(t.AccessToken, tokenAD(conn.WorkspaceID, "token"))
	if err != nil {
		return
	}
	var refresh []byte
	if t.RefreshToken != "" {
		refresh, _ = s.Box.SealString(t.RefreshToken, tokenAD(conn.WorkspaceID, "refresh"))
	}
	var exp *time.Time
	if !t.Expiry.IsZero() {
		e := t.Expiry
		exp = &e
	}
	if err := s.Q.UpdateHostReputeTokens(ctx, store.UpdateHostReputeTokensParams{WorkspaceID: conn.WorkspaceID, TokenEnc: access, RefreshEnc: refresh, TokenExpiresAt: exp}); err != nil {
		s.Log.WarnContext(ctx, "hostrepute: saving the refreshed token failed", "workspace", conn.WorkspaceID, "err", err)
	}
}

// connected is the account and tokens of a new connection.
type connected struct {
	method  string
	account Account
	access  string
	refresh string
	expiry  *time.Time
	apiURL  string
	// monitored is what HostRepute monitors after connecting.
	monitored []Asset
}

// register adds the shared candidates to monitoring right after connecting,
// so "Monitored now" is true when the person lands back in ReputeCast.
// Failures are left to the first sync.
func (s *Service) register(ctx context.Context, c Client, ws uuid.UUID) []Asset {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	assets, err := c.Assets(ctx)
	if err != nil {
		return []Asset{}
	}
	cands, err := Candidates(ctx, s.Q, ws)
	if err != nil {
		return assets
	}
	for _, cand := range cands {
		if !cand.Shared || slices.ContainsFunc(assets, func(a Asset) bool { return strings.EqualFold(a.Asset, cand.Asset) }) {
			continue
		}
		a, err := c.AddAsset(ctx, cand.Asset, cand.Type, cand.Role)
		var le *LimitError
		if errors.As(err, &le) {
			break
		}
		if err != nil {
			s.Log.WarnContext(ctx, "hostrepute: adding an asset after connecting failed", "asset", cand.Asset, "err", err)
			continue
		}
		assets = append(assets, a)
	}
	return assets
}

// save stores a new or renewed connection, audits it and queues the first
// sync, all in one transaction.
func (s *Service) save(ctx context.Context, ws uuid.UUID, by *audit.Actor, c connected) error {
	now := s.now()
	access, err := s.Box.SealString(c.access, tokenAD(ws, "token"))
	if err != nil {
		return err
	}
	var refresh []byte
	if c.refresh != "" {
		if refresh, err = s.Box.SealString(c.refresh, tokenAD(ws, "refresh")); err != nil {
			return err
		}
	}
	var limit *int32
	if c.account.Team.MonitorLimit > 0 {
		l := int32(c.account.Team.MonitorLimit)
		limit = &l
	}
	var byID *uuid.UUID
	if by != nil {
		byID = by.ID
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.GetHostReputeConnectionForUpdate(ctx, ws)
		existed := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		conn, err := q.UpsertHostReputeConnection(ctx, store.UpsertHostReputeConnectionParams{
			WorkspaceID: ws, Method: c.method, AccountEmail: c.account.Email, TeamName: c.account.Team.Name, Scopes: nonNil(c.account.Scopes),
			TokenEnc: access, RefreshEnc: refresh, TokenExpiresAt: c.expiry, ApiUrl: c.apiURL, AccountID: c.account.ID, TeamID: c.account.Team.ID,
			Plan: c.account.Team.Plan, MonitorLimit: limit, Sources: int32(c.account.Sources), CheckIntervalMinutes: int32(max(c.account.CheckIntervalMinutes, 1)),
			DmarcRua: c.account.DMARCRua, ConnectedBy: byID, ConnectedAt: now,
		})
		if err != nil {
			return err
		}
		if c.monitored != nil {
			if err := q.SetHostReputeMonitored(ctx, store.SetHostReputeMonitoredParams{WorkspaceID: ws, Monitored: monitoredJSON(c.monitored, now), MonitorLimit: limit}); err != nil {
				return err
			}
		}
		action := "hostrepute.connect"
		var prev any
		if existed {
			action = "hostrepute.reconnect"
			prev = connectionAudit(before)
		}
		after := connectionAudit(conn)
		if c.monitored != nil {
			after["monitored"] = assetNames(c.monitored)
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &ws, Action: action, TargetType: "integration", TargetID: "hostrepute", TargetName: "HostRepute · " + c.account.Team.Name,
			Before: prev, After: after, Actor: by,
		}); err != nil {
			return err
		}
		return s.queueSync(ctx, tx, ws, byID, "connect", false, true)
	})
}

// queueSync records a check run and queues the sync that fills it in.
func (s *Service) queueSync(ctx context.Context, tx pgx.Tx, ws uuid.UUID, by *uuid.UUID, trigger string, recheck, initial bool) error {
	if s.Jobs == nil {
		return nil
	}
	q := store.New(tx)
	conn, err := q.GetHostReputeConnection(ctx, ws)
	if err != nil {
		return err
	}
	run, err := q.CreateDeliverabilityCheckRun(ctx, store.CreateDeliverabilityCheckRunParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, Via: "hostrepute", Trigger: trigger, Assets: nonNil(monitoredNames(conn.Monitored)),
		Total: conn.Sources, RequestedBy: by, StartedAt: s.now(),
	})
	if err != nil {
		return err
	}
	_, err = s.Jobs.InsertTx(ctx, tx, SyncArgs{WorkspaceID: ws, RunID: &run.ID, Recheck: recheck, Initial: initial}, nil)
	return err
}

func connectionAudit(c store.HostreputeConnection) map[string]any {
	return map[string]any{"method": c.Method, "team": c.TeamName, "account": c.AccountEmail, "scopes": c.Scopes, "status": c.Status}
}

func assetNames(as []Asset) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.Asset)
	}
	return out
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func userActor(u *store.User) *audit.Actor {
	id := u.ID
	name := u.Name
	if strings.TrimSpace(name) == "" {
		name = u.Email
	}
	return &audit.Actor{Type: authz.ActorUser, ID: &id, Name: name}
}

func principalActor(ctx context.Context) *audit.Actor {
	p := api.PrincipalFrom(ctx)
	if p == nil {
		return nil
	}
	id := p.ID
	return &audit.Actor{Type: p.Type, ID: &id, Name: p.Name}
}

func principalUserID(ctx context.Context) *uuid.UUID {
	if p := api.PrincipalFrom(ctx); p != nil && p.Type == authz.ActorUser {
		id := p.UserID
		return &id
	}
	return nil
}

// unavailableProblem is HostRepute being unreachable during a request, and
// what wasn't done ("saved", "changed", "requested").
func unavailableProblem(ctx context.Context, err error, nothing string) error {
	l := i18n.LangFrom(ctx)
	reach := msgs.M("unreachableHostRepute").Text(l, nil)
	var ue *UnavailableError
	if errors.As(err, &ue) {
		reach = ue.In(l)
	}
	return api.Err(http.StatusServiceUnavailable, "hostrepute_unavailable", msgs.M("problem.unavailable").Text(l, i18n.Args{"reach": reach, "nothing": nothing}))
}

// ConnectAPIKey connects with a HostRepute API key (the alternative to the
// browser handoff). The key is verified against HostRepute before it is
// stored, encrypted; it is never returned.
func (s *Service) ConnectAPIKey(ctx context.Context, ws *store.Workspace, key string) (HostReputeConnection, error) {
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, " \t\r\n") {
		return HostReputeConnection{}, api.Invalid("body.apiKey", say(ctx, "problem.keyFormat", nil))
	}
	var c Client
	apiURL := s.apiURL()
	if apiURL == DemoAPIURL {
		if !strings.HasPrefix(key, "hr_") {
			return HostReputeConnection{}, keyRejected(ctx)
		}
		c = s.demoClient(ws.ID, DemoMonitored)
	} else {
		c = NewHTTPClient(apiURL, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: key, TokenType: "Bearer"}), s.HTTP)
	}
	acct, err := c.Account(ctx)
	switch {
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrForbidden):
		return HostReputeConnection{}, keyRejected(ctx)
	case errors.Is(err, ErrUnavailable):
		return HostReputeConnection{}, unavailableProblem(ctx, err, "saved")
	case err != nil:
		return HostReputeConnection{}, api.Err(http.StatusBadGateway, "hostrepute_error", say(ctx, "problem.answeredError", i18n.Args{"error": err.Error()}))
	}
	monitored := s.register(ctx, c, ws.ID)
	if err := s.save(ctx, ws.ID, principalActor(ctx), connected{method: "api_key", account: acct, access: key, apiURL: apiURL, monitored: monitored}); err != nil {
		return HostReputeConnection{}, err
	}
	return s.Connection(ctx, ws.ID)
}

func keyRejected(ctx context.Context) error {
	return api.Invalid("body.apiKey", say(ctx, "problem.keyRejected", nil))
}

// stateTTL is how long a person has to approve on HostRepute.
const stateTTL = 15 * time.Minute

// StartOAuth begins the browser handoff: it stores a single-use state bound
// to the person and workspace with an encrypted PKCE verifier, and returns
// HostRepute's consent URL.
func (s *Service) StartOAuth(ctx context.Context, ws *store.Workspace, user *store.User) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(state))
	verifier := NewVerifier()
	id := uuid.Must(uuid.NewV7())
	enc, err := s.Box.SealString(verifier, "hostrepute-oauth:"+id.String())
	if err != nil {
		return "", err
	}
	o := s.oauth()
	if _, err := s.Q.DeleteExpiredHostReputeOAuthStates(ctx); err != nil {
		return "", err
	}
	if err := s.Q.CreateHostReputeOAuthState(ctx, store.CreateHostReputeOAuthStateParams{
		ID: id, WorkspaceID: ws.ID, UserID: user.ID, StateHash: sum[:], VerifierEnc: enc, RedirectUri: o.RedirectURL(), ExpiresAt: s.now().Add(stateTTL),
	}); err != nil {
		return "", err
	}
	return o.AuthCodeURL(state, verifier), nil
}

// CallbackParams are what HostRepute sends back to the redirect URI.
type CallbackParams struct {
	Code, State, Error, ErrorDescription string
	// Self is the callback URL as requested, to come back after signing in.
	Self string
}

// CompleteOAuth finishes the handoff and returns where to send the browser:
// the connect screen with the result. The signed-in person must be the one
// who started it and still hold integrations:hostrepute.
func (s *Service) CompleteOAuth(ctx context.Context, user *store.User, p CallbackParams) string {
	if user == nil {
		return "/sign-in?next=" + urlQuery(p.Self)
	}
	sum := sha256.Sum256([]byte(p.State))
	st, err := s.Q.ConsumeHostReputeOAuthState(ctx, sum[:])
	if err != nil || st.ExpiresAt.Before(s.now()) {
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			s.Log.ErrorContext(ctx, "hostrepute: reading the OAuth state failed", "err", err)
		}
		return "/?hostrepute=expired"
	}
	ws, err := s.Q.GetWorkspace(ctx, st.WorkspaceID)
	if err != nil {
		return "/"
	}
	back := func(result, reason string) string {
		u := "/w/" + ws.Slug + "/deliverability/connect?result=" + result
		if reason != "" {
			u += "&reason=" + reason
		}
		return u
	}
	actor := userActor(user)
	fail := func(result, reason, why string) string {
		_ = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "hostrepute.connect", TargetType: "integration", TargetID: "hostrepute",
			TargetName: "HostRepute", Result: result, Reason: why, Actor: actor})
		return back("error", reason)
	}
	if st.UserID != user.ID {
		return fail(audit.Denied, "wrong_user", "the handoff was started by someone else")
	}
	m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: user.ID})
	if err != nil || m.Status != "active" || !authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions).Has(authz.IntegrationsHostRepute) {
		return fail(audit.Denied, "forbidden", "missing "+string(authz.IntegrationsHostRepute))
	}
	if p.Error != "" {
		reason := "declined"
		if p.Error != "access_denied" {
			reason = "hostrepute_error"
		}
		return fail(audit.Failed, reason, "HostRepute answered "+p.Error+" "+p.ErrorDescription)
	}
	verifier, err := s.Box.OpenString(st.VerifierEnc, "hostrepute-oauth:"+st.ID.String())
	if err != nil {
		return fail(audit.Failed, "exchange_failed", "the PKCE verifier could not be read")
	}
	o := s.oauth()
	tok, err := o.Exchange(ctx, p.Code, verifier)
	if err != nil {
		s.Log.WarnContext(ctx, "hostrepute: code exchange failed", "err", err)
		return fail(audit.Failed, "exchange_failed", "code exchange failed")
	}
	apiURL := s.apiURL()
	var c Client
	if apiURL == DemoAPIURL {
		c = s.demoClient(ws.ID, DemoMonitored)
	} else {
		c = NewHTTPClient(apiURL, oauth2.StaticTokenSource(tok), s.HTTP)
	}
	acct, err := c.Account(ctx)
	switch {
	case errors.Is(err, ErrUnavailable):
		return fail(audit.Failed, "unreachable", err.Error())
	case err != nil:
		return fail(audit.Failed, "hostrepute_error", err.Error())
	}
	if len(acct.Scopes) == 0 {
		acct.Scopes = strings.Fields(fmt.Sprint(tok.Extra("scope")))
	}
	var exp *time.Time
	if !tok.Expiry.IsZero() {
		e := tok.Expiry
		exp = &e
	}
	monitored := s.register(ctx, c, ws.ID)
	if err := s.save(ctx, ws.ID, actor, connected{method: "oauth", account: acct, access: tok.AccessToken, refresh: tok.RefreshToken, expiry: exp, apiURL: apiURL, monitored: monitored}); err != nil {
		s.Log.ErrorContext(ctx, "hostrepute: saving the connection failed", "err", err)
		return back("error", "save_failed")
	}
	return back("connected", "")
}

func urlQuery(s string) string {
	r := strings.NewReplacer("%", "%25", "&", "%26", "?", "%3F", "=", "%3D", "#", "%23", " ", "%20", "+", "%2B")
	return r.Replace(s)
}

// Disconnect forgets the token (revoking it at HostRepute when possible),
// deletes the data synced from HostRepute and queues the built-in checks.
// The HostRepute team and its monitoring keep running on hostrepute.com.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace) error {
	conn, err := s.Q.GetHostReputeConnection(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return api.NotFound("HostRepute connection")
	}
	if err != nil {
		return err
	}
	if c, _, err := s.clientFor(ctx, conn); err == nil && conn.Method == "oauth" {
		rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		if err := c.Revoke(rctx); err != nil {
			s.Log.WarnContext(ctx, "hostrepute: revoking the token failed; it is forgotten anyway", "err", err)
		}
		cancel()
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.GetHostReputeConnectionForUpdate(ctx, ws.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("HostRepute connection")
		}
		if err != nil {
			return err
		}
		if _, err := q.DeleteHostReputeConnection(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.DeleteHostReputeData(ctx, ws.ID); err != nil {
			return err
		}
		prev := connectionAudit(before)
		prev["monitored"] = monitoredNames(before.Monitored)
		if err := s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &ws.ID, Action: "hostrepute.disconnect", TargetType: "integration", TargetID: "hostrepute",
			TargetName: "HostRepute · " + before.TeamName, Before: prev,
		}); err != nil {
			return err
		}
		if s.Fallback != nil {
			return s.Fallback(ctx, tx, ws.ID, "fallback")
		}
		return nil
	})
}

// requireConnected loads a connection that can act on HostRepute.
func (s *Service) requireConnected(ctx context.Context, ws uuid.UUID) (store.HostreputeConnection, error) {
	conn, err := s.Q.GetHostReputeConnection(ctx, ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return conn, api.Conflict(say(ctx, "problem.notConnected", nil))
	}
	if err != nil {
		return conn, err
	}
	if conn.Status != "connected" && conn.Status != "limit_reached" {
		args := i18n.Args{"when": "no", "date": ""}
		if conn.TokenRejectedAt != nil {
			args["when"], args["date"] = "yes", monthDay(i18n.LangFrom(ctx), *conn.TokenRejectedAt)
		}
		return conn, api.Err(http.StatusConflict, "hostrepute_reconnect", say(ctx, "problem.reconnect", args))
	}
	return conn, nil
}

// AddAsset adds one of the workspace's domains, IPs or hostnames to
// HostRepute monitoring. A full plan answers 409 monitor_limit_reached with
// the limit and a swap suggestion.
func (s *Service) AddAsset(ctx context.Context, ws *store.Workspace, asset string) (HostReputeConnection, error) {
	conn, err := s.requireConnected(ctx, ws.ID)
	if err != nil {
		return HostReputeConnection{}, err
	}
	if !slices.Contains(conn.Scopes, ScopeAssetsWrite) {
		return HostReputeConnection{}, api.Err(http.StatusConflict, "hostrepute_scope", say(ctx, "problem.noAssetsScope", nil))
	}
	cands, err := Candidates(ctx, s.Q, ws.ID)
	if err != nil {
		return HostReputeConnection{}, err
	}
	cand, ok := findCandidate(cands, strings.ToLower(strings.TrimSpace(asset)))
	if !ok {
		return HostReputeConnection{}, api.Invalid("body.asset", say(ctx, "problem.notCandidate", nil))
	}
	monitored := parseMonitored(conn.Monitored)
	if slices.ContainsFunc(monitored, func(m monitoredAsset) bool { return strings.EqualFold(m.Asset, cand.Asset) }) {
		return s.Connection(ctx, ws.ID)
	}
	c, w, err := s.clientFor(ctx, conn)
	if err != nil {
		return HostReputeConnection{}, err
	}
	added, err := c.AddAsset(ctx, cand.Asset, cand.Type, cand.Role)
	s.saveRefreshedToken(ctx, conn, w)
	if err := s.limitOrError(ctx, err, conn, cand, monitored, cands); err != nil {
		return HostReputeConnection{}, err
	}
	names := append(monitoredNames(conn.Monitored), added.Asset)
	if err := s.storeMonitoring(ctx, ws.ID, c, "hostrepute.asset_add", monitoredNames(conn.Monitored), names, cand.Asset); err != nil {
		return HostReputeConnection{}, err
	}
	return s.Connection(ctx, ws.ID)
}

// limitOrError maps HostRepute's answer to an add-to-monitoring request.
func (s *Service) limitOrError(ctx context.Context, err error, conn store.HostreputeConnection, cand Candidate, monitored []monitoredAsset, cands []Candidate) error {
	if err == nil {
		return nil
	}
	var le *LimitError
	switch {
	case errors.As(err, &le):
		team := le.Team
		if team == "" {
			team = conn.TeamName
		}
		p := api.Err(http.StatusConflict, "monitor_limit_reached", say(ctx, "problem.limit", i18n.Args{"team": team, "used": le.Used, "limit": le.Limit, "asset": cand.Asset})).
			WithMeta("limit", le.Limit, "used", le.Used, "team", team, "asset", cand.Asset)
		if swap := swapSuggestion(monitored, cands, cand); swap != nil {
			p = p.WithMeta("swap", swap.Asset, "swapRole", swap.Role)
		}
		_ = s.Audit.Log(ctx, audit.Event{WorkspaceID: &conn.WorkspaceID, Action: "hostrepute.asset_add", TargetType: "hostrepute_asset", TargetID: cand.Asset,
			TargetName: cand.Asset, Result: audit.Failed, Reason: fmt.Sprintf("HostRepute monitoring limit reached (%d of %d)", le.Used, le.Limit)})
		return p
	case errors.Is(err, ErrUnauthorized):
		return api.Err(http.StatusConflict, "hostrepute_reconnect", say(ctx, "problem.reconnect", i18n.Args{"when": "no", "date": ""}))
	case errors.Is(err, ErrUnavailable):
		return unavailableProblem(ctx, err, "changed")
	}
	return err
}

// swapSuggestion picks a monitored asset that could make room: an
// organization domain (it doesn't send mail itself), when the new asset sends.
func swapSuggestion(monitored []monitoredAsset, cands []Candidate, adding Candidate) *Candidate {
	if adding.Role == RoleOrganizationDomain {
		return nil
	}
	for _, m := range monitored {
		if c, ok := findCandidate(cands, m.Asset); ok && c.Role == RoleOrganizationDomain {
			return &c
		}
	}
	return nil
}

// SwapAsset stops monitoring one asset and monitors another instead, for a
// full plan ("swap acme.io for mta.acme.io").
func (s *Service) SwapAsset(ctx context.Context, ws *store.Workspace, remove, add string) (HostReputeConnection, error) {
	conn, err := s.requireConnected(ctx, ws.ID)
	if err != nil {
		return HostReputeConnection{}, err
	}
	monitored := parseMonitored(conn.Monitored)
	i := slices.IndexFunc(monitored, func(m monitoredAsset) bool { return strings.EqualFold(m.Asset, remove) })
	if i < 0 {
		return HostReputeConnection{}, api.Invalid("body.remove", say(ctx, "problem.notMonitored", i18n.Args{"asset": remove}))
	}
	cands, err := Candidates(ctx, s.Q, ws.ID)
	if err != nil {
		return HostReputeConnection{}, err
	}
	cand, ok := findCandidate(cands, strings.ToLower(strings.TrimSpace(add)))
	if !ok {
		return HostReputeConnection{}, api.Invalid("body.add", say(ctx, "problem.notCandidate", nil))
	}
	c, w, err := s.clientFor(ctx, conn)
	if err != nil {
		return HostReputeConnection{}, err
	}
	id := monitored[i].ID
	if id == "" {
		id = monitored[i].Asset
	}
	if err := c.RemoveAsset(ctx, id); err != nil && !errors.Is(err, ErrNotFound) {
		return HostReputeConnection{}, s.limitOrError(ctx, err, conn, cand, monitored, cands)
	}
	if _, err := c.AddAsset(ctx, cand.Asset, cand.Type, cand.Role); err != nil {
		// Put the removed asset back so nothing is lost.
		if old, ok := findCandidate(cands, monitored[i].Asset); ok {
			_, _ = c.AddAsset(ctx, old.Asset, old.Type, old.Role)
		}
		s.saveRefreshedToken(ctx, conn, w)
		return HostReputeConnection{}, s.limitOrError(ctx, err, conn, cand, monitored, cands)
	}
	s.saveRefreshedToken(ctx, conn, w)
	before := monitoredNames(conn.Monitored)
	after := slices.DeleteFunc(slices.Clone(before), func(n string) bool { return strings.EqualFold(n, remove) })
	after = append(after, cand.Asset)
	if err := s.storeMonitoring(ctx, ws.ID, c, "hostrepute.asset_swap", before, after, remove+" → "+cand.Asset); err != nil {
		return HostReputeConnection{}, err
	}
	return s.Connection(ctx, ws.ID)
}

// storeMonitoring saves the monitored list after a change, audits it and
// queues a sync for the new asset's data.
func (s *Service) storeMonitoring(ctx context.Context, ws uuid.UUID, c Client, action string, before, after []string, target string) error {
	assets, err := c.Assets(ctx)
	if err != nil {
		// HostRepute accepted the change; keep the list we know.
		assets = nil
		for _, n := range after {
			assets = append(assets, Asset{Asset: n, Type: GuessType(n), Status: "pending"})
		}
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		conn, err := q.GetHostReputeConnectionForUpdate(ctx, ws)
		if err != nil {
			return err
		}
		if err := q.SetHostReputeMonitored(ctx, store.SetHostReputeMonitoredParams{WorkspaceID: ws, Monitored: monitoredJSON(assets, s.now()), MonitorLimit: conn.MonitorLimit}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws, Action: action, TargetType: "hostrepute_asset", TargetID: target, TargetName: target,
			Before: map[string]any{"monitored": before}, After: map[string]any{"monitored": after}}); err != nil {
			return err
		}
		return s.queueSync(ctx, tx, ws, principalUserID(ctx), "manual", false, false)
	})
}

// SyncNow queues a sync of the HostRepute data ("Sync now").
func (s *Service) SyncNow(ctx context.Context, ws *store.Workspace) (HostReputeSyncQueued, error) {
	conn, err := s.requireConnected(ctx, ws.ID)
	if err != nil {
		return HostReputeSyncQueued{}, err
	}
	var out HostReputeSyncQueued
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "hostrepute.sync", TargetType: "integration", TargetID: "hostrepute", TargetName: "HostRepute · " + conn.TeamName}); err != nil {
			return err
		}
		if err := s.queueSync(ctx, tx, ws.ID, principalUserID(ctx), "manual", false, false); err != nil {
			return err
		}
		run, err := q.LatestDeliverabilityCheckRun(ctx, ws.ID)
		if err != nil {
			return err
		}
		out = HostReputeSyncQueued{RunID: run.ID.String(), StartedAt: run.StartedAt}
		return nil
	})
	return out, err
}

// QueueRecheck asks HostRepute to re-check every monitored asset now and
// syncs the result (Deliverability's "Run all checks now").
func (s *Service) QueueRecheck(ctx context.Context, tx pgx.Tx, ws uuid.UUID, by *uuid.UUID) error {
	return s.queueSync(ctx, tx, ws, by, "manual", true, false)
}

// RequestDelisting asks HostRepute to request the delisting of an asset
// from a blocklist (HostRepute's delisting workflow).
func (s *Service) RequestDelisting(ctx context.Context, ws uuid.UUID, asset, source string) (Delisting, error) {
	conn, err := s.requireConnected(ctx, ws)
	if err != nil {
		return Delisting{}, err
	}
	c, w, err := s.clientFor(ctx, conn)
	if err != nil {
		return Delisting{}, err
	}
	d, err := c.RequestDelisting(ctx, asset, source)
	s.saveRefreshedToken(ctx, conn, w)
	switch {
	case errors.Is(err, ErrUnauthorized):
		return d, api.Err(http.StatusConflict, "hostrepute_reconnect", say(ctx, "problem.reconnect", i18n.Args{"when": "no", "date": ""}))
	case errors.Is(err, ErrUnavailable):
		return d, unavailableProblem(ctx, err, "requested")
	case errors.Is(err, ErrNotFound):
		return d, api.NotFound("Listing")
	}
	if err == nil && d.RequestedAt.IsZero() {
		d.RequestedAt = s.now()
	}
	return d, err
}
