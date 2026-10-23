// Package apikeys manages API keys and MCP agents (A5 §3.9, §5.13). Keys
// never exceed their creator's permissions and are shown exactly once.
// Expired keys stay listed for 90 days with the calls that still try to use
// them.
package apikeys

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Kinds of keys.
const (
	KindAPIKey = "api_key"
	KindAgent  = "mcp_agent"
)

type Service struct {
	Pool      *pgxpool.Pool
	Q         *store.Queries
	Audit     *audit.Logger
	PublicURL func(context.Context) string
	Now       func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// APIKeyCreator is who created a key.
type APIKeyCreator struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	AvatarColor string `json:"avatarColor"`
}

// APIKeyRejections counts calls refused because the key expired or was revoked.
type APIKeyRejections struct {
	Calls  int64      `json:"calls" doc:"In the last 7 days"`
	Since  *time.Time `json:"since,omitempty" doc:"First rejected call"`
	IP     string     `json:"ip,omitempty" doc:"Busiest client address"`
	Client string     `json:"client,omitempty" doc:"Client label of that address"`
}

// APIKeyView is an API key or MCP agent as listed. The secret is never returned.
type APIKeyView struct {
	ID          string              `json:"id"`
	Kind        string              `json:"kind" enum:"api_key,mcp_agent"`
	Name        string              `json:"name"`
	Description string              `json:"description" doc:"Client label for agents (Claude Desktop)"`
	Masked      string              `json:"masked" example:"rc_live_8f2c…a91d"`
	Permissions []string            `json:"permissions"`
	Sensitive   []string            `json:"sensitive" doc:"The ⚑ permissions among them"`
	Scope       access.ScopeSummary `json:"scope"`
	CreatedBy   APIKeyCreator       `json:"createdBy"`
	CreatedAt   time.Time           `json:"createdAt"`
	ExpiresAt   *time.Time          `json:"expiresAt,omitempty"`
	LastUsedAt  *time.Time          `json:"lastUsedAt,omitempty"`
	LastUsedIP  string              `json:"lastUsedIp,omitempty"`
	// LastUsedClient labels the caller ("nightly job", "GitHub Actions runner").
	LastUsedClient string `json:"lastUsedClient,omitempty"`
	// LastCall is the MCP tool of an agent's last call.
	LastCall   string            `json:"lastCall,omitempty"`
	Status     string            `json:"status" enum:"active,expired"`
	Rejections *APIKeyRejections `json:"rejections,omitempty"`
	// Transactional is set for send-only, test-mode and SMTP keys.
	Transactional *APIKeyTransactional `json:"transactional,omitempty"`
}

// APIKeyTransactional describes a send-only key: its preset (send_only, or
// smtp for an SMTP relay credential), its mode (test keys deliver only to
// the workspace's members and the allowed test addresses), the templates
// or categories it is limited to, and its send rate.
type APIKeyTransactional struct {
	Preset        string              `json:"preset" enum:"send_only,smtp"`
	Mode          string              `json:"mode" enum:"live,test"`
	Templates     []string            `json:"templates" doc:"Template IDs it may send; empty for all, including ones published later"`
	Categories    []APIKeyCategoryRef `json:"categories" doc:"Categories it may send (SMTP credentials); empty for all"`
	TestAddresses []string            `json:"testAddresses" doc:"Allowed test addresses besides the members (qa+*@acme.io)"`
	RatePerSecond int                 `json:"ratePerSecond"`
	// Username is an SMTP credential's user name: the key's ID.
	Username string `json:"username,omitempty"`
}

// APIKeyCategoryRef names a transactional category.
type APIKeyCategoryRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// APIKeyList is the API keys or MCP agents card.
type APIKeyList struct {
	Items []APIKeyView `json:"items"`
	// Permissions, Held, Locked and Lists feed the create dialog.
	Permissions []access.PermissionInfo  `json:"permissions"`
	Groups      []string                 `json:"groups"`
	Held        []string                 `json:"held"`
	Locked      []APIKeyLockedPermission `json:"locked"`
	Lists       []access.ScopedList      `json:"lists"`
	// Endpoint is the API base URL (or the MCP endpoint for agents).
	Endpoint string `json:"endpoint"`
}

// APIKeyLockedPermission is a permission the caller can't put on a key.
type APIKeyLockedPermission struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

func (s *Service) base(ctx context.Context) string {
	if s.PublicURL != nil {
		return strings.TrimRight(s.PublicURL(ctx), "/")
	}
	return ""
}

// viewWith is a key as listed, its words in a language.
func viewWith(l i18n.Lang, r store.KeysListRow, lists, categories map[uuid.UUID]string, now time.Time) APIKeyView {
	perms := authz.ParseSet(r.Permissions)
	v := APIKeyView{
		ID: r.ID.String(), Kind: r.Kind, Name: r.Name, Description: r.Description, Masked: r.Prefix,
		Permissions: StoredOrder(r.Permissions), Sensitive: []string{},
		Scope:     access.ViewIn(l, access.ParseScope(r.Scope), lists, false),
		CreatedBy: APIKeyCreator{Name: r.CreatedByName, AvatarColor: r.CreatedByColor}, CreatedAt: r.CreatedAt,
		ExpiresAt: r.ExpiresAt, LastUsedAt: r.LastUsedAt, LastUsedClient: r.LastUsedClient, LastCall: r.LastCall, Status: "active",
	}
	if r.CreatedBy != nil {
		v.CreatedBy.ID = r.CreatedBy.String()
	}
	if v.CreatedBy.Name == "" {
		v.CreatedBy.Name = msgs.M("label.formerMember").Text(l, nil)
		v.CreatedBy.AvatarColor = "neutral"
	}
	if r.LastUsedIp != nil {
		v.LastUsedIP = r.LastUsedIp.String()
	}
	if r.ExpiresAt != nil && !r.ExpiresAt.After(now) {
		v.Status = "expired"
	}
	for _, k := range perms.Keys() {
		if authz.IsSensitive(k) && containsStr(r.Permissions, string(k)) {
			v.Sensitive = append(v.Sensitive, string(k))
		}
	}
	if r.Preset != "" {
		t := &APIKeyTransactional{Preset: r.Preset, Mode: r.Mode, Templates: nonNil(r.Templates), Categories: []APIKeyCategoryRef{},
			TestAddresses: nonNil(r.TestAddresses), RatePerSecond: int(r.RatePerSecond)}
		for _, c := range r.Categories {
			name := categories[c]
			if name == "" {
				name = msgs.M("label.deletedCategory").Text(l, nil)
			}
			t.Categories = append(t.Categories, APIKeyCategoryRef{ID: c.String(), Name: name})
		}
		if r.Preset == PresetSMTP {
			t.Username = r.ID.String()
		}
		v.Transactional = t
	}
	return v
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// categoryNames maps the workspace's transactional categories to their names.
func (s *Service) categoryNames(ctx context.Context, wsID uuid.UUID) map[uuid.UUID]string {
	out := map[uuid.UUID]string{}
	if rows, err := s.Q.TxnRouteCategories(ctx, wsID); err == nil {
		for _, r := range rows {
			out[r.ID] = r.Name
		}
	}
	return out
}

// StoredOrder returns the keys the creator picked, in the order picked,
// without the view permissions that were added automatically.
func StoredOrder(keys []string) []string {
	set := authz.ParseSet(keys)
	out := []string{}
	for _, k := range keys {
		p := authz.Permission(k)
		if !authz.Valid(p) || containsStr(out, k) {
			continue
		}
		if by := access.ImpliedBy(set, p); by != "" && containsStr(keys, string(by)) {
			continue
		}
		out = append(out, k)
	}
	return out
}

// PickedFirst stores the picked keys in their order, then the implied ones.
func PickedFirst(picked []string, set authz.Set) []string {
	out := []string{}
	for _, k := range picked {
		if set.Has(authz.Permission(k)) && !containsStr(out, k) {
			out = append(out, k)
		}
	}
	for _, k := range set.Strings() {
		if !containsStr(out, k) {
			out = append(out, k)
		}
	}
	return out
}

func containsStr(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// List returns the keys (or agents) of the workspace.
func (s *Service) List(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string) (APIKeyList, error) {
	rows, err := s.Q.KeysList(ctx, store.KeysListParams{WorkspaceID: ws.ID, Kind: kind})
	if err != nil {
		return APIKeyList{}, err
	}
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return APIKeyList{}, err
	}
	lang := i18n.LangFrom(ctx)
	out := APIKeyList{Items: []APIKeyView{}, Permissions: access.Catalog(lang), Groups: access.Groups(lang), Held: p.Permissions.Strings(), Locked: []APIKeyLockedPermission{}, Lists: []access.ScopedList{}}
	now := s.now()
	cats := s.categoryNames(ctx, ws.ID)
	for _, r := range rows {
		v := viewWith(lang, r, lists, cats, now)
		if v.Status == "expired" {
			if rj, err := s.rejections(ctx, r.ID); err == nil && rj.Calls > 0 {
				v.Rejections = &rj
			}
		}
		out.Items = append(out.Items, v)
	}
	for _, d := range authz.Defs {
		if d.OwnerOnly {
			continue
		}
		if !authz.HoldableBy(d.Key, actorOf(kind)) {
			out.Locked = append(out.Locked, APIKeyLockedPermission{Key: string(d.Key), Reason: access.NotHoldable(lang, d.Key, actorOf(kind))})
			continue
		}
		if p.Permissions.Has(d.Key) {
			continue
		}
		out.Locked = append(out.Locked, APIKeyLockedPermission{Key: string(d.Key), Reason: access.CantGrant(lang, d.Key, access.Helpers(ctx, s.Q, ws.ID, d.Key))})
	}
	tl, err := s.Q.TeamWorkspaceLists(ctx, ws.ID)
	if err != nil {
		return APIKeyList{}, err
	}
	for _, l := range tl {
		if p.Scope.AllowsList(l.ID) {
			out.Lists = append(out.Lists, access.ScopedList{ID: l.ID.String(), Name: l.Name})
		}
	}
	if kind == KindAgent {
		out.Endpoint = s.base(ctx) + "/mcp"
	} else {
		out.Endpoint = s.base(ctx) + api.Version
	}
	return out, nil
}

func (s *Service) rejections(ctx context.Context, id uuid.UUID) (APIKeyRejections, error) {
	r, err := s.Q.KeysRejections(ctx, id)
	if err != nil {
		return APIKeyRejections{}, err
	}
	out := APIKeyRejections{Calls: r.Calls, IP: r.TopIp, Client: r.TopClient}
	if r.Calls > 0 {
		if first, err := s.Q.KeysFirstRejection(ctx, id); err == nil {
			out.Since = &first
		}
	}
	return out, nil
}

// Get returns one key with its rejected calls.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, kind string, id uuid.UUID) (APIKeyView, error) {
	r, err := s.Q.KeysGet(ctx, store.KeysGetParams{ID: id, WorkspaceID: ws.ID, Kind: kind})
	if err != nil {
		return APIKeyView{}, api.MapError(err, noun(kind))
	}
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return APIKeyView{}, err
	}
	v := viewWith(i18n.LangFrom(ctx), store.KeysListRow(r), lists, s.categoryNames(ctx, ws.ID), s.now())
	if rj, err := s.rejections(ctx, r.ID); err == nil && rj.Calls > 0 {
		v.Rejections = &rj
	}
	return v, nil
}

func noun(kind string) string {
	if kind == KindAgent {
		return "MCP agent"
	}
	return "API key"
}

func action(kind, verb string) string {
	if kind == KindAgent {
		return "agent." + verb
	}
	return "apikey." + verb
}

func prefixFor(kind string) string {
	if kind == KindAgent {
		return authn.AgentPrefix
	}
	return authn.APIKeyPrefix
}

// APIKeyCreateInput creates a key or agent.
type APIKeyCreateInput struct {
	Name        string   `json:"name" minLength:"1" maxLength:"80"`
	Description string   `json:"description,omitempty" maxLength:"120" doc:"Client label, e.g. Claude Desktop"`
	Permissions []string `json:"permissions" minItems:"1" maxItems:"60"`
	// Lists restricts the key to these lists (empty = all lists).
	Lists     []string   `json:"lists,omitempty" maxItems:"100"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	// Transactional makes a send-only, test-mode or SMTP key; its only
	// permission is then transactional:send.
	Transactional *APIKeyTransactionalInput `json:"transactional,omitempty"`
}

// Send-only key presets.
const (
	PresetSendOnly = "send_only"
	PresetSMTP     = "smtp"
)

// APIKeyTransactionalInput is the "Send-only key" preset of the create
// dialog, and an SMTP credential.
type APIKeyTransactionalInput struct {
	Preset string `json:"preset" enum:"send_only,smtp"`
	Mode   string `json:"mode,omitempty" enum:"live,test" default:"live"`
	// Templates limits the key to these template IDs (empty: all, including
	// ones published later).
	Templates []string `json:"templates,omitempty" maxItems:"200"`
	// Categories limits the key to these categories (SMTP credentials).
	Categories []string `json:"categories,omitempty" maxItems:"50"`
	// TestAddresses are allowed besides the workspace's members (test mode):
	// addresses or patterns such as qa+*@acme.io.
	TestAddresses []string `json:"testAddresses,omitempty" maxItems:"50"`
	RatePerSecond int      `json:"ratePerSecond,omitempty" minimum:"1" maximum:"10000" doc:"Messages per second (50 by default)"`
}

// APIKeyCreated is shown exactly once.
type APIKeyCreated struct {
	Key    APIKeyView `json:"key"`
	Secret string     `json:"secret" doc:"The full key. Shown only in this response."`
	// Usage is "Authorization: Bearer rc_live_… against https://…/api/v1".
	Endpoint string `json:"endpoint"`
}

// Create makes a key or agent with permissions the caller holds.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, in APIKeyCreateInput) (APIKeyCreated, error) {
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return APIKeyCreated{}, err
	}
	wanted := make([]authz.Permission, 0, len(in.Permissions))
	for _, k := range in.Permissions {
		wanted = append(wanted, authz.Permission(k))
	}
	set, err := s.checkGrant(ctx, ws, p, kind, wanted)
	if err != nil {
		return APIKeyCreated{}, err
	}
	scope, err := (&access.ScopeRequest{Lists: in.Lists}).Resolve(lists, nil)
	if err != nil {
		return APIKeyCreated{}, api.Invalid("body.lists", access.ScopeProblem(i18n.LangFrom(ctx), err))
	}
	if !scope.Within(p.Scope) {
		return APIKeyCreated{}, api.Err(http.StatusForbidden, api.CodeEscalation, say(ctx, "problem.ownScope", nil))
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(s.now()) {
		return APIKeyCreated{}, api.Invalid("body.expiresAt", say(ctx, "problem.expiry", nil))
	}
	var tx0 *transactionalKey
	if in.Transactional != nil {
		if kind != KindAPIKey {
			return APIKeyCreated{}, api.Invalid("body.transactional", say(ctx, "problem.sendOnlyKind", nil))
		}
		tk, err := s.transactionalInput(ctx, ws, in)
		if err != nil {
			return APIKeyCreated{}, err
		}
		tx0 = &tk
	}
	stored := PickedFirst(in.Permissions, set)
	var out APIKeyCreated
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		out, err = s.insertWith(ctx, store.New(tx), ws, p, kind, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), stored, scope, in.ExpiresAt, nil, lists, tx0)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{
			WorkspaceID: &ws.ID, Action: action(kind, "created"), TargetType: kind, TargetID: out.Key.ID, TargetName: out.Key.Name,
			After:    map[string]any{"name": out.Key.Name, "permissions": out.Key.Permissions, "scope": scopeLabel(access.View(scope, lists, false)), "expires_at": in.ExpiresAt, "masked": out.Key.Masked},
			Metadata: audit.KeepSummary(map[string]any{"permission": string(managePerm(kind))}, keySummary(out.Key, scope, lists)...),
		})
	})
	if err != nil {
		return APIKeyCreated{}, err
	}
	return out, nil
}

func managePerm(kind string) authz.Permission {
	if kind == KindAgent {
		return authz.MCPManage
	}
	return authz.APIKeysManage
}

// scopeLabel is a scope's English label, as the audit log keeps it.
func scopeLabel(v access.ScopeSummary) string {
	if v.All {
		return "all lists"
	}
	return v.Label
}

// keySummary is a new key's audit summary, its parts kept for later
// readers: "campaigns:view · Weekly Digest only · no expiry".
func keySummary(v APIKeyView, scope authz.Scope, lists map[uuid.UUID]string) []any {
	parts := []any{strings.Join(v.Permissions, ", "), access.ScopeText(scope, lists, false)}
	if t := v.Transactional; t != nil {
		parts[1] = transactionalScope(t)
		if t.Mode == "test" {
			parts = append(parts, msgs.M("audit.testMode").Keep(nil))
		}
		if t.Preset == PresetSMTP {
			parts = append(parts, msgs.M("audit.smtp").Keep(nil))
		}
	}
	if v.ExpiresAt != nil {
		parts = append(parts, msgs.M("audit.expires").Keep(i18n.Args{"at": *v.ExpiresAt}))
	} else {
		parts = append(parts, msgs.M("audit.noExpiry").Keep(nil))
	}
	return parts
}

func actorOf(kind string) authz.ActorType {
	if kind == KindAgent {
		return authz.ActorMCPAgent
	}
	return authz.ActorAPIKey
}

// checkGrant applies "nobody can grant what they don't hold", and keeps
// people-only permissions off keys and agents.
func (s *Service) checkGrant(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, wanted []authz.Permission) (authz.Set, error) {
	lang := i18n.LangFrom(ctx)
	for _, w := range wanted {
		if !authz.Valid(w) {
			return nil, api.Invalid("body.permissions", msgs.M("problem.unknownPermission").Text(lang, i18n.Args{"permission": string(w)}))
		}
		if !authz.HoldableBy(w, actorOf(kind)) {
			pr := api.Err(http.StatusForbidden, api.CodeEscalation, access.NotHoldable(lang, w, actorOf(kind)))
			pr.Permission = string(w)
			return nil, pr
		}
	}
	set, err := authz.CheckGrant(p.Permissions, wanted)
	if err != nil {
		if esc := (*authz.EscalationError)(nil); errors.As(err, &esc) && len(esc.Missing) > 0 {
			// Name a permission the caller asked for before the ones it implies.
			m := esc.Missing[0]
			for _, w := range wanted {
				if slices.Contains(esc.Missing, w) {
					m = w
					break
				}
			}
			pr := api.Err(http.StatusForbidden, api.CodeEscalation, access.CantGrant(lang, m, access.Helpers(ctx, s.Q, ws.ID, m)))
			pr.Permission = string(m)
			return nil, pr
		}
		return nil, api.MapError(err, "API key")
	}
	return set, nil
}

func (s *Service) insertWith(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, kind, name, desc string, perms []string, scope authz.Scope, expires *time.Time, rotatedFrom *uuid.UUID, lists map[uuid.UUID]string, tk *transactionalKey) (APIKeyCreated, error) {
	if name == "" {
		return APIKeyCreated{}, api.Invalid("body.name", say(ctx, "problem.name", nil))
	}
	prefix := prefixFor(kind)
	if tk != nil {
		prefix = tk.prefix()
	}
	secret, hash, masked := authn.NewAPIKey(prefix)
	creator := p.UserID
	id := uuid.Must(uuid.NewV7())
	row, err := q.KeysCreate(ctx, store.KeysCreateParams{
		ID: id, WorkspaceID: ws.ID, Kind: kind, Name: name, Description: desc, Prefix: masked, TokenHash: hash,
		Permissions: perms, Scope: access.ScopeJSON(scope), CreatedBy: &creator, ExpiresAt: expires, RotatedFrom: rotatedFrom,
	})
	if err != nil {
		return APIKeyCreated{}, err
	}
	u, _ := q.GetUser(ctx, creator)
	lr := store.KeysListRow{
		ID: id, WorkspaceID: ws.ID, Kind: kind, Name: name, Description: desc, Prefix: masked, Permissions: perms, Scope: access.ScopeJSON(scope),
		CreatedBy: &creator, CreatedAt: row.CreatedAt, ExpiresAt: expires, CreatedByName: u.Name, CreatedByColor: u.AvatarColor, Mode: "live", RatePerSecond: 50,
	}
	if tk != nil {
		if err := q.KeysSetTransactional(ctx, store.KeysSetTransactionalParams{ID: id, WorkspaceID: ws.ID, Preset: tk.Preset, Mode: tk.Mode, Templates: tk.Templates,
			Categories: tk.Categories, TestAddresses: tk.TestAddresses, RatePerSecond: int32(tk.Rate)}); err != nil {
			return APIKeyCreated{}, err
		}
		lr.Preset, lr.Mode, lr.Templates, lr.Categories, lr.TestAddresses, lr.RatePerSecond = tk.Preset, tk.Mode, tk.Templates, tk.Categories, tk.TestAddresses, int32(tk.Rate)
	}
	v := viewWith(i18n.LangFrom(ctx), lr, lists, s.categoryNames(ctx, ws.ID), s.now())
	ep := s.base(ctx) + api.Version
	if kind == KindAgent {
		ep = s.base(ctx) + "/mcp"
	}
	return APIKeyCreated{Key: v, Secret: secret, Endpoint: ep}, nil
}

// Rotate replaces a key (active or expired) with a new secret that has the
// same name, permissions and scope. The old one stops working at once.
func (s *Service) Rotate(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, id uuid.UUID) (APIKeyCreated, error) {
	old, err := s.Q.KeysGet(ctx, store.KeysGetParams{ID: id, WorkspaceID: ws.ID, Kind: kind})
	if err != nil {
		return APIKeyCreated{}, api.MapError(err, noun(kind))
	}
	// An integration's key rotates in Integrations, where the old one keeps
	// working for 24 hours (integrations hub, M16).
	if k, err := s.Q.GetAPIKey(ctx, store.GetAPIKeyParams{ID: id, WorkspaceID: ws.ID}); err == nil && k.IntegrationID != nil {
		return APIKeyCreated{}, api.Err(http.StatusConflict, "managed_by_integration", say(ctx, "problem.integrationKey", nil))
	}
	wanted := make([]authz.Permission, 0, len(old.Permissions))
	for _, k := range old.Permissions {
		if pk := authz.Permission(k); authz.Valid(pk) {
			wanted = append(wanted, pk)
		}
	}
	if _, err := s.checkGrant(ctx, ws, p, kind, wanted); err != nil {
		return APIKeyCreated{}, err
	}
	scope := access.ParseScope(old.Scope)
	if !scope.Within(p.Scope) {
		return APIKeyCreated{}, api.Err(http.StatusForbidden, api.CodeEscalation, say(ctx, "problem.rotateScope", nil))
	}
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return APIKeyCreated{}, err
	}
	var expires *time.Time
	if old.ExpiresAt != nil {
		life := old.ExpiresAt.Sub(old.CreatedAt)
		if life < 24*time.Hour {
			life = 24 * time.Hour
		}
		e := s.now().Add(life).Truncate(24 * time.Hour)
		expires = &e
	}
	expired := old.ExpiresAt != nil && !old.ExpiresAt.After(s.now())
	var out APIKeyCreated
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.KeysRevoke(ctx, store.KeysRevokeParams{ID: old.ID, WorkspaceID: ws.ID, Kind: kind}); err != nil {
			return err
		}
		var err error
		var tk *transactionalKey
		if old.Preset != "" {
			tk = &transactionalKey{Preset: old.Preset, Mode: old.Mode, Templates: old.Templates, Categories: old.Categories, TestAddresses: old.TestAddresses, Rate: int(old.RatePerSecond)}
		}
		out, err = s.insertWith(ctx, q, ws, p, kind, old.Name, old.Description, old.Permissions, scope, expires, &old.ID, lists, tk)
		if err != nil {
			return err
		}
		verb := "rotated"
		if expired {
			verb = "replaced"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &ws.ID, Action: action(kind, verb), TargetType: kind, TargetID: out.Key.ID, TargetName: old.Name,
			Before:   map[string]any{"masked": old.Prefix, "expires_at": old.ExpiresAt},
			After:    map[string]any{"masked": out.Key.Masked, "expires_at": expires},
			Metadata: audit.KeepText(map[string]any{"rotated_from": old.ID.String(), "permission": string(managePerm(kind))},
				"summary", msgs.M("audit.rotated").Keep(i18n.Args{"masked": out.Key.Masked, "old": old.Prefix})),
		})
	})
	return out, err
}

// Revoke stops a key immediately.
func (s *Service) Revoke(ctx context.Context, ws *store.Workspace, kind string, id uuid.UUID) error {
	old, err := s.Q.KeysGet(ctx, store.KeysGetParams{ID: id, WorkspaceID: ws.ID, Kind: kind})
	if err != nil {
		return api.MapError(err, noun(kind))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.KeysRevoke(ctx, store.KeysRevokeParams{ID: id, WorkspaceID: ws.ID, Kind: kind})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.NotFound(noun(kind))
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action(kind, "revoked"), TargetType: kind, TargetID: id.String(), TargetName: old.Name,
			Before: map[string]any{"masked": old.Prefix, "permissions": old.Permissions}, Metadata: map[string]any{"permission": string(managePerm(kind))}})
	})
}

// Delete removes an expired or revoked key from the list.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, kind string, id uuid.UUID) error {
	old, err := s.Q.KeysGet(ctx, store.KeysGetParams{ID: id, WorkspaceID: ws.ID, Kind: kind})
	if err != nil {
		return api.MapError(err, noun(kind))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.KeysDelete(ctx, store.KeysDeleteParams{ID: id, WorkspaceID: ws.ID, Kind: kind})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.Conflict(say(ctx, "problem.deleteActive", nil))
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action(kind, "deleted"), TargetType: kind, TargetID: id.String(), TargetName: old.Name,
			Before: map[string]any{"masked": old.Prefix, "expires_at": old.ExpiresAt}, Metadata: map[string]any{"permission": string(managePerm(kind))}})
	})
}

// MCPAgentAction is one recent MCP tool call.
type MCPAgentAction struct {
	ID         string    `json:"id"`
	OccurredAt time.Time `json:"occurredAt"`
	AgentID    string    `json:"agentId,omitempty"`
	Agent      string    `json:"agent"`
	Tool       string    `json:"tool"`
	Target     string    `json:"target"`
	Result     string    `json:"result" enum:"allowed,denied,error"`
	// Note is "campaigns:pause" for an allowed call or "needs campaigns:send ·
	// proposal recorded for Linh Nguyen" for a denied one.
	Note string `json:"note,omitempty"`
	// Proposal tells that a denied call was proposed to a member to review.
	Proposal bool `json:"proposal"`
	// Summary is what the tool returned or did ("Paused October Product Update · Outlook route").
	Summary string `json:"summary,omitempty"`
}

// MCPAgentActionList is the recent agent actions card.
type MCPAgentActionList struct {
	Items []MCPAgentAction `json:"items"`
}

// AgentActions lists the latest MCP calls from the audit log, their
// details, notes and outcomes in the reader's language when the agent's
// call kept them (internal/mcpserver).
func (s *Service) AgentActions(ctx context.Context, ws *store.Workspace, limit int) (MCPAgentActionList, error) {
	rows, err := s.Q.KeysAgentActions(ctx, store.KeysAgentActionsParams{WorkspaceID: &ws.ID, Lim: int32(limit)})
	if err != nil {
		return MCPAgentActionList{}, err
	}
	l := i18n.LangFrom(ctx)
	out := MCPAgentActionList{Items: []MCPAgentAction{}}
	for _, r := range rows {
		var meta struct {
			Tool     string `json:"tool"`
			Detail   string `json:"detail"`
			Note     string `json:"note"`
			Outcome  string `json:"outcome"`
			Perm     string `json:"permission"`
			Proposal bool   `json:"proposal"`
		}
		_ = json.Unmarshal(r.Metadata, &meta)
		kept := audit.KeptIn(r.Metadata)
		a := MCPAgentAction{ID: r.ID.String(), OccurredAt: r.OccurredAt, Agent: r.ActorName, Tool: meta.Tool, Result: r.Result, Summary: audit.In(kept.Outcome, l, meta.Outcome),
			// Calls recorded before the flag say it in their note.
			Proposal: meta.Proposal || strings.Contains(meta.Note, "proposal recorded")}
		if r.ActorID != nil {
			a.AgentID = r.ActorID.String()
		}
		if a.Tool == "" {
			a.Tool = r.Action
		}
		a.Target = r.TargetName
		if detail := audit.In(kept.Detail, l, meta.Detail); detail != "" {
			if a.Target != "" {
				a.Target += " · " + detail
			} else {
				a.Target = detail
			}
		}
		switch {
		case meta.Note != "":
			a.Note = audit.In(kept.Note, l, meta.Note)
		case r.Result == "denied" && strings.HasPrefix(r.Reason, "missing "):
			a.Note = msgs.M("agent.needs").Text(l, i18n.Args{"permission": strings.TrimPrefix(r.Reason, "missing ")})
		case r.Result == "denied" && r.Reason != "":
			a.Note = r.Reason
		}
		out.Items = append(out.Items, a)
	}
	return out, nil
}

// RecordRejection counts a call refused because a key expired or was
// revoked, so the expired-key card can show what still uses it. The REST
// middleware and the MCP endpoint call it.
func RecordRejection(ctx context.Context, q *store.Queries, keyID uuid.UUID, ip *netip.Addr, userAgent, reason string) {
	addr := ""
	if ip != nil {
		addr = ip.String()
	}
	_ = q.KeysRecordRejection(ctx, store.KeysRecordRejectionParams{ApiKeyID: keyID, Ip: addr, Client: ClientLabel(userAgent), Reason: reason})
}

// ClientLabel turns a User-Agent into a short label for the keys list:
// "GitHub Actions runner", "Claude Desktop", "curl", "python-requests".
func ClientLabel(ua string) string {
	l := strings.ToLower(ua)
	switch {
	case ua == "":
		return ""
	case strings.Contains(l, "github-actions") || strings.Contains(l, "github actions"):
		return "GitHub Actions runner"
	case strings.Contains(l, "claude-code") || strings.Contains(l, "claude code"):
		return "Claude Code"
	case strings.Contains(l, "claude"):
		return "Claude Desktop"
	case strings.Contains(l, "zapier"):
		return "Zapier"
	case strings.Contains(l, "n8n"):
		return "n8n"
	}
	tok := strings.Fields(ua)[0]
	if i := strings.IndexByte(tok, '/'); i > 0 {
		tok = tok[:i]
	}
	if len(tok) > 40 {
		tok = tok[:40]
	}
	return tok
}
