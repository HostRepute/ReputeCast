package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

type ctxKey int

const (
	principalKey ctxKey = iota
	workspaceKey
	userKey
	sessionKey
)

// PrincipalFrom returns the authenticated caller, or nil on public operations.
func PrincipalFrom(ctx context.Context) *authz.Principal {
	p, _ := ctx.Value(principalKey).(*authz.Principal)
	return p
}

// WorkspaceFrom returns the workspace of a workspace operation.
func WorkspaceFrom(ctx context.Context) *store.Workspace {
	w, _ := ctx.Value(workspaceKey).(*store.Workspace)
	return w
}

// UserFrom returns the signed-in user (nil for API keys and public calls).
func UserFrom(ctx context.Context) *store.User {
	u, _ := ctx.Value(userKey).(*store.User)
	return u
}

// SessionFrom returns the current session (nil for API keys and public calls).
func SessionFrom(ctx context.Context) *store.Session {
	s, _ := ctx.Value(sessionKey).(*store.Session)
	return s
}

// WithPrincipal stores a principal (used by tests and internal callers).
func WithPrincipal(ctx context.Context, p *authz.Principal) context.Context {
	ctx = context.WithValue(ctx, principalKey, p)
	return audit.WithPrincipal(ctx, p)
}

// Policy is the part of the instance authentication policy the middleware needs.
type Policy struct {
	SessionIdle      time.Duration
	TwoFAEnforcement string // off | sensitive | everyone
}

// Authenticator resolves and authorizes callers for every operation.
type Authenticator struct {
	Q          *store.Queries
	Audit      *audit.Logger
	Log        *slog.Logger
	CookieName string
	Policy     func(context.Context) (Policy, error)
	Now        func() time.Time
	// OnKeyRejected, when set, records a call refused because its key was
	// revoked or expired (reason "revoked" or "expired"), so Settings › Keys
	// can show what still uses it.
	OnKeyRejected func(ctx context.Context, keyID uuid.UUID, reason string)
	// Integrations knows what integration keys may call and logs their
	// calls. Without it integration keys are refused.
	Integrations Integrations
}

// caller is who a request comes from: a person or a key, and for an
// integration key its integration.
type caller struct {
	p     *authz.Principal
	u     *store.User
	s     *store.Session
	ik    *IntegrationKey
	keyID uuid.UUID
}

// Middleware returns the Huma middleware enforcing each operation's Access.
func (a *Authenticator) Middleware(hapi huma.API) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		access := accessOf(hctx.Operation())
		ctx := hctx.Context()
		if access.Kind == AccessPublic {
			// Public operations still see a signed-in user when there is one
			// (accepting an invite while signed in as someone else).
			if u, s, _ := a.sessionUser(ctx, hctx); u != nil {
				ctx = context.WithValue(ctx, userKey, u)
				ctx = context.WithValue(ctx, sessionKey, s)
			}
			next(huma.WithContext(hctx, ctx))
			return
		}
		start := a.now()
		c, perr := a.resolve(ctx, hctx, start)
		if perr != nil {
			a.write(hapi, hctx, perr)
			return
		}
		p, u, s := c.p, c.u, c.s
		// refuse answers with a problem, logging the call when an
		// integration key made it.
		refuse := func(perr *Problem) {
			a.recordRefused(ctx, hctx, c.ik, c.keyID, start, perr)
			a.write(hapi, hctx, perr)
		}
		if c.ik != nil && !c.ik.Allows(operationID(hctx)) {
			refuse(Err(http.StatusForbidden, CodeNotForThisKey, fmt.Sprintf(
				"This key belongs to %s and works only for what %s can do. Change what %s can do in Settings › Integrations.", c.ik.Name, c.ik.Name, c.ik.Name)))
			return
		}
		if access.Integration != "" && (c.ik == nil || access.Integration != "*" && c.ik.Kind != access.Integration) {
			who := "an integration's key"
			if access.Integration == "wordpress" {
				who = "a WordPress site's key"
			}
			refuse(Err(http.StatusForbidden, CodePermissionDenied, "Only "+who+" calls this."))
			return
		}
		if access.HumansOnly && p.Type != authz.ActorUser {
			refuse(Err(http.StatusForbidden, CodeHumansOnly, "This action can only be taken by a signed-in person, not an API key or agent."))
			return
		}
		if u != nil && !access.AllowPending {
			if perr := a.pendingRestriction(ctx, u, s); perr != nil {
				a.write(hapi, hctx, perr)
				return
			}
		}
		switch access.Kind {
		case AccessInstanceAdmin:
			if !p.IsInstanceAdmin {
				a.write(hapi, hctx, Err(http.StatusForbidden, CodePermissionDenied, "Only instance admins can open Instance admin."))
				return
			}
		case AccessKey:
			if p.Type == authz.ActorUser {
				a.write(hapi, hctx, Err(http.StatusForbidden, CodePermissionDenied, "Call this with an API key: it describes the key that calls it."))
				return
			}
			ws, err := a.Q.GetWorkspace(ctx, p.WorkspaceID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					refuse(Err(http.StatusNotFound, CodeWorkspaceNotFound, "Workspace not found."))
					return
				}
				refuse(a.unavailable(ctx, "workspace", err))
				return
			}
			ctx = context.WithValue(ctx, workspaceKey, &ws)
		case AccessWorkspace:
			ws, perr := a.workspacePrincipal(ctx, hctx.Param("workspace"), p)
			if perr != nil {
				refuse(perr)
				return
			}
			ctx = context.WithValue(ctx, workspaceKey, ws)
			if access.Permission != "" && !p.Can(access.Permission) {
				helpers := a.helpers(ctx, ws.ID, access.Permission, p.UserID)
				a.auditDenied(audit.WithPrincipal(ctx, p), hctx, ws.ID, access.Permission)
				refuse(Forbidden(access.Permission, helpers))
				return
			}
		}
		ctx = context.WithValue(ctx, principalKey, p)
		ctx = audit.WithPrincipal(ctx, p)
		if u != nil {
			ctx = context.WithValue(ctx, userKey, u)
		}
		if s != nil {
			ctx = context.WithValue(ctx, sessionKey, s)
		}
		if c.ik != nil {
			ctx = context.WithValue(ctx, integrationKey, c.ik)
			a.serveRecorded(ctx, hctx, c.ik, c.keyID, start, next)
			return
		}
		next(huma.WithContext(hctx, ctx))
	}
}

func operationID(hctx huma.Context) string {
	if op := hctx.Operation(); op != nil {
		return op.OperationID
	}
	return ""
}

func (a *Authenticator) write(hapi huma.API, hctx huma.Context, p *Problem) {
	if p.Status == http.StatusUnauthorized {
		hctx.SetHeader("WWW-Authenticate", `Bearer realm="reputecast"`)
	}
	_ = huma.WriteErr(hapi, hctx, p.Status, p.Detail, problemMarker{p})
}

// problemMarker carries the original problem through huma.WriteErr so the
// NewError override can restore its code and helpers.
type problemMarker struct{ p *Problem }

func (m problemMarker) Error() string { return m.p.Detail }

func (a *Authenticator) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// sessionUser loads the user behind the session cookie, if any.
func (a *Authenticator) sessionUser(ctx context.Context, hctx huma.Context) (*store.User, *store.Session, *Problem) {
	token := cookieValue(hctx.Header("Cookie"), a.CookieName)
	if token == "" {
		return nil, nil, Err(http.StatusUnauthorized, CodeUnauthenticated, "Sign in to continue.")
	}
	sess, err := a.Q.GetSessionByToken(ctx, authn.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, Err(http.StatusUnauthorized, CodeUnauthenticated, "Sign in to continue.")
	}
	if err != nil {
		// A database blip is not a signed-out session: keep the cookie.
		return nil, nil, a.unavailable(ctx, "session", err)
	}
	pol, err := a.Policy(ctx)
	if err == nil && pol.SessionIdle > 0 && a.now().Sub(sess.LastSeenAt) > pol.SessionIdle {
		_, _ = a.Q.RevokeSession(ctx, store.RevokeSessionParams{ID: sess.ID, UserID: sess.UserID})
		return nil, nil, Err(http.StatusUnauthorized, CodeSessionExpired, "Your session timed out. Sign in again; unsaved work in this tab is kept.")
	}
	u, err := a.Q.GetUser(ctx, sess.UserID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil, Err(http.StatusUnauthorized, CodeUnauthenticated, "Sign in to continue.")
	case err != nil:
		return nil, nil, a.unavailable(ctx, "session user", err)
	case u.Status != "active":
		return nil, nil, Err(http.StatusUnauthorized, CodeUnauthenticated, "This account is deactivated. Ask an instance admin.")
	}
	_ = a.Q.TouchSession(ctx, sess.ID)
	_ = a.Q.TouchUserActive(ctx, u.ID)
	return &u, &sess, nil
}

func (a *Authenticator) resolve(ctx context.Context, hctx huma.Context, start time.Time) (caller, *Problem) {
	if bearer := bearerToken(hctx.Header("Authorization")); bearer != "" {
		return a.keyPrincipal(ctx, hctx, bearer, start)
	}
	u, s, perr := a.sessionUser(ctx, hctx)
	if perr != nil {
		return caller{}, perr
	}
	p := &authz.Principal{
		Type:            authz.ActorUser,
		ID:              u.ID,
		UserID:          u.ID,
		Name:            displayName(u),
		Email:           u.Email,
		IsInstanceAdmin: u.IsInstanceAdmin,
		SessionID:       s.ID,
		MFAVerified:     s.MfaVerified,
		Permissions:     authz.Set{},
	}
	return caller{p: p, u: u, s: s}, nil
}

func (a *Authenticator) keyRejected(ctx context.Context, id uuid.UUID, reason string) {
	if a.OnKeyRejected != nil {
		a.OnKeyRejected(ctx, id, reason)
	}
}

func (a *Authenticator) keyPrincipal(ctx context.Context, hctx huma.Context, token string, start time.Time) (caller, *Problem) {
	if !authn.LooksLikeAPIKey(token) {
		return caller{}, Err(http.StatusUnauthorized, CodeUnauthenticated, "Unknown API key.")
	}
	k, err := a.Q.GetAPIKeyByToken(ctx, authn.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return caller{}, Err(http.StatusUnauthorized, CodeUnauthenticated, "This API key is revoked or unknown.")
	}
	if err != nil {
		return caller{}, a.unavailable(ctx, "API key", err)
	}
	// An integration key works only through its integration: for the
	// operations it was given, with its own refusals in its call log.
	var ik *IntegrationKey
	if k.IntegrationID != nil {
		if a.Integrations == nil {
			return caller{}, Err(http.StatusUnauthorized, CodeUnauthenticated, "This API key belongs to an integration, and integrations aren't available on this server.")
		}
		if ik, err = a.Integrations.Key(ctx, *k.IntegrationID); err != nil {
			return caller{}, a.unavailable(ctx, "integration", err)
		}
	}
	refuse := func(p *Problem) (caller, *Problem) {
		a.recordRefused(ctx, hctx, ik, k.ID, start, p)
		return caller{}, p
	}
	// A rotated integration key keeps working during its grace period.
	inGrace := k.GraceUntil != nil && k.GraceUntil.After(a.now())
	if k.RevokedAt != nil && !inGrace || ik != nil && ik.Disconnected {
		a.keyRejected(ctx, k.ID, "revoked")
		if ik != nil {
			return refuse(Err(http.StatusUnauthorized, CodeKeyRevoked, fmt.Sprintf("This key was revoked. Reconnect %s in Settings › Integrations to get a new key.", ik.Name)))
		}
		return caller{}, Err(http.StatusUnauthorized, CodeUnauthenticated, "This API key is revoked or unknown.")
	}
	if k.ExpiresAt != nil && !k.ExpiresAt.After(a.now()) {
		a.keyRejected(ctx, k.ID, "expired")
		if ik != nil {
			return refuse(Err(http.StatusUnauthorized, CodeKeyExpired, fmt.Sprintf("This key expired on %s. Reconnect %s in Settings › Integrations to get a new key.", k.ExpiresAt.Format("Jan 2, 2006"), ik.Name)))
		}
		return caller{}, Err(http.StatusUnauthorized, CodeUnauthenticated, "This API key expired on "+k.ExpiresAt.Format("Jan 2, 2006")+". Rotate it in Settings › Keys & MCP.")
	}
	perms := authz.ParseSet(k.Permissions)
	// A key never exceeds what its creator can do right now: nothing once
	// they have left the workspace, and no answer while that can't be read.
	if k.CreatedBy != nil {
		m, err := a.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: k.WorkspaceID, UserID: *k.CreatedBy})
		switch {
		case err == nil && m.Status == "active":
			perms = perms.Intersect(authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions))
		case err == nil || errors.Is(err, pgx.ErrNoRows):
			if ik != nil {
				return refuse(Err(http.StatusForbidden, CodeOwnerDeactivated, fmt.Sprintf(
					"The member who owns this key left the workspace, so it has lost its permissions. Someone with integrations:manage can take it over in Settings › Integrations › %s; the key stays the same.", ik.Name)))
			}
			perms = authz.Set{}
		default:
			return caller{}, a.unavailable(ctx, "API key creator", err)
		}
	}
	var scope authz.Scope
	_ = json.Unmarshal(k.Scope, &scope)
	typ := authz.ActorAPIKey
	if k.Kind == "mcp_agent" {
		typ = authz.ActorMCPAgent
	}
	perms = perms.HoldableBy(typ)
	ip := audit.RequestFrom(ctx).IP
	_ = a.Q.TouchAPIKey(ctx, store.TouchAPIKeyParams{ID: k.ID, Ip: ip})
	creator := uuid.Nil
	if k.CreatedBy != nil {
		creator = *k.CreatedBy
	}
	return caller{p: &authz.Principal{
		Type:        typ,
		ID:          k.ID,
		Name:        k.Name,
		UserID:      creator,
		WorkspaceID: k.WorkspaceID,
		Permissions: perms,
		Scope:       scope,
		MFAVerified: true,
	}, ik: ik, keyID: k.ID}, nil
}

// workspacePrincipal loads the workspace and fills in the principal's
// permissions there.
func (a *Authenticator) workspacePrincipal(ctx context.Context, slug string, p *authz.Principal) (*store.Workspace, *Problem) {
	ws, err := a.Q.GetWorkspaceBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, Err(http.StatusNotFound, CodeWorkspaceNotFound, "Workspace not found.")
	}
	if err != nil {
		return nil, a.unavailable(ctx, "workspace", err)
	}
	switch p.Type {
	case authz.ActorAPIKey, authz.ActorMCPAgent:
		if p.WorkspaceID != ws.ID {
			return nil, Err(http.StatusForbidden, CodePermissionDenied, "This key belongs to another workspace.")
		}
	default:
		m, err := a.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: p.UserID})
		if err != nil || m.Status != "active" {
			if errors.Is(err, pgx.ErrNoRows) || err == nil {
				return nil, Err(http.StatusNotFound, CodeWorkspaceNotFound, "Workspace not found.")
			}
			return nil, a.unavailable(ctx, "membership", err)
		}
		p.WorkspaceID = ws.ID
		p.RoleKey = m.RoleKey
		p.RoleName = m.RoleName
		p.Permissions = authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions)
		_ = json.Unmarshal(m.Scope, &p.Scope)
		_ = a.Q.TouchMemberActive(ctx, store.TouchMemberActiveParams{WorkspaceID: ws.ID, UserID: p.UserID})
	}
	return &ws, nil
}

// pendingRestriction blocks users who must change a temporary password or
// set up a required second factor before doing anything else.
func (a *Authenticator) pendingRestriction(ctx context.Context, u *store.User, s *store.Session) *Problem {
	if u.MustChangePassword {
		return Err(http.StatusForbidden, CodePasswordChange, "Replace your temporary password to continue.")
	}
	if u.TotpEnabledAt != nil {
		return nil
	}
	pol, err := a.Policy(ctx)
	if err != nil {
		// Fail closed: without the policy we can't tell whether two-factor
		// setup is required before this user may continue.
		return Err(http.StatusServiceUnavailable, CodeUnavailable, "Couldn't check the sign-in policy. Try again in a moment.")
	}
	if pol.TwoFAEnforcement == "off" {
		return nil
	}
	if s != nil && s.MfaVerified {
		return nil // SSO sessions where the IdP asserted MFA, or passkey sign-ins
	}
	passkeys, _ := a.Q.ListPasskeys(ctx, u.ID)
	if len(passkeys) > 0 {
		return nil
	}
	if u.TwofaGraceUntil != nil && u.TwofaGraceUntil.After(a.now()) {
		return nil
	}
	if pol.TwoFAEnforcement == "sensitive" && !a.holdsSensitive(ctx, u.ID) {
		return nil
	}
	return Err(http.StatusForbidden, CodeTwoFactorSetup, "Your instance requires two-factor authentication. Add a passkey or an authenticator app to continue.")
}

func (a *Authenticator) holdsSensitive(ctx context.Context, userID uuid.UUID) bool {
	wss, err := a.Q.ListUserWorkspaces(ctx, userID)
	if err != nil {
		return false
	}
	for _, w := range wss {
		m, err := a.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: w.ID, UserID: userID})
		if err == nil && authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions).HasSensitive() {
			return true
		}
	}
	return false
}

// helpers names up to two active members who hold perm.
func (a *Authenticator) helpers(ctx context.Context, wsID uuid.UUID, perm authz.Permission, exclude uuid.UUID) []string {
	rows, err := a.Q.ListMembersWithRoles(ctx, wsID)
	if err != nil {
		return nil
	}
	var names []string
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			names = append(names, r.Name)
			if len(names) == 2 {
				break
			}
		}
	}
	return names
}

func (a *Authenticator) auditDenied(ctx context.Context, hctx huma.Context, wsID uuid.UUID, perm authz.Permission) {
	op := hctx.Operation()
	// Reads are not logged unless the permission is sensitive (export, SQL…).
	if hctx.Method() == http.MethodGet && !authz.IsSensitive(perm) {
		return
	}
	_ = a.Audit.Log(ctx, audit.Event{
		WorkspaceID: &wsID,
		Action:      op.OperationID,
		Result:      audit.Denied,
		Reason:      "missing " + string(perm),
		TargetType:  "operation",
		TargetID:    op.Method + " " + op.Path,
	})
}

func displayName(u *store.User) string {
	if strings.TrimSpace(u.Name) != "" {
		return u.Name
	}
	return u.Email
}

func bearerToken(h string) string {
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

func cookieValue(header, name string) string {
	if header == "" || name == "" {
		return ""
	}
	req := http.Request{Header: http.Header{"Cookie": {header}}}
	c, err := req.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// unavailable answers 503 when the database can't say who is asking (the
// request may be retried; the session and key stay valid).
func (a *Authenticator) unavailable(ctx context.Context, what string, err error) *Problem {
	if a.Log != nil {
		a.Log.WarnContext(ctx, "authentication lookup failed", "what", what, "err", err)
	}
	return Err(http.StatusServiceUnavailable, CodeUnavailable, "Couldn't check who you are right now. Try again in a moment.")
}
