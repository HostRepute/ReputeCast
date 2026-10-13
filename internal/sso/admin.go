package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// SSOJIT is a provider's just-in-time account setting.
type SSOJIT struct {
	Enabled       bool   `json:"enabled"`
	WorkspaceID   string `json:"workspaceId,omitempty"`
	WorkspaceName string `json:"workspaceName,omitempty"`
	RoleKey       string `json:"roleKey,omitempty"`
	RoleName      string `json:"roleName,omitempty"`
}

// SSOProviderItem is one provider in the lists (SSO providers tab,
// Authentication › SSO providers).
type SSOProviderItem struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Preset string `json:"preset" enum:"google,entra,okta,keycloak,authentik,oidc"`
	Name   string `json:"name"`
	Status string `json:"status" enum:"draft,enabled,disabled,error"`
	Issuer string `json:"issuer"`
	// Domains are the allowed email domains.
	Domains []string `json:"domains"`
	JIT     SSOJIT   `json:"jit"`
	// GroupsClaim is false for presets without a groups claim (Google).
	GroupsClaim bool     `json:"groupsClaim"`
	Mappings    int      `json:"mappings"`
	Enforced    []string `json:"enforced"`
	// TwoFactor says how the IdP satisfies two-factor.
	TwoFactor string     `json:"twoFactor"`
	EnabledAt *time.Time `json:"enabledAt,omitempty"`
	// DiscoveryOK is nil before the first discovery test.
	DiscoveryOK *bool `json:"discoveryOk,omitempty"`
	// BreakGlass names the Owners who keep password access while enforced.
	BreakGlass []string `json:"breakGlass"`
	// Members is how many people sign in with this provider.
	Members int `json:"members"`
}

// SSOProviderList is GET /admin/sso.
type SSOProviderList struct {
	Items   []SSOProviderItem `json:"items"`
	Presets []SSOPreset       `json:"presets"`
}

// SSOSecretInfo describes the write-only client secret without revealing it.
type SSOSecretInfo struct {
	Set       bool       `json:"set"`
	SetAt     *time.Time `json:"setAt,omitempty"`
	SetBy     string     `json:"setBy,omitempty"`
	ExpiresOn *time.Time `json:"expiresOn,omitempty" doc:"When the secret expires at the IdP, as entered by the admin"`
}

// SSODiscoveryResult is the last discovery test.
type SSODiscoveryResult struct {
	OK        bool      `json:"ok"`
	CheckedAt time.Time `json:"checkedAt"`
	URL       string    `json:"url,omitempty"`
	Status    int       `json:"status,omitempty"`
	Millis    int64     `json:"ms,omitempty"`
	Error     string    `json:"error,omitempty"`
	// Endpoints found: authorization, token, userinfo, JWKS.
	Endpoints []string `json:"endpoints"`
	Keys      int      `json:"keys"`
}

// SSOGroupMapping is one group → workspace → role mapping.
type SSOGroupMapping struct {
	Group       string `json:"group" minLength:"1" maxLength:"200"`
	Label       string `json:"label,omitempty" maxLength:"80"`
	WorkspaceID string `json:"workspaceId" format:"uuid"`
	RoleKey     string `json:"roleKey" maxLength:"64"`
	// Lists restrict the role to some lists; empty means all lists.
	Lists []string `json:"lists,omitempty" maxItems:"50"`
}

// SSOClaimSample is one row of the claim mapping table: what the last
// test sign-in returned for a field.
type SSOClaimSample struct {
	Field string `json:"field" enum:"email,email_verified,name,groups"`
	Claim string `json:"claim"`
	Found bool   `json:"found"`
	// Value is the sample as shown ("marcus@acme.io", ["3f2a…c91", …]).
	Value string `json:"value,omitempty"`
	Count int    `json:"count,omitempty"`
	// Missing replaces the sample when the claim wasn't sent ("not sent by Entra ID").
	Missing string `json:"missing,omitempty"`
	// Note explains what a missing claim means ("treated as verified for allowed domains").
	Note string `json:"note,omitempty"`
}

// protocolClaims are left out of the test view: they describe the token,
// not the person.
var protocolClaims = map[string]bool{"aud": true, "exp": true, "iat": true, "nbf": true, "nonce": true, "at_hash": true, "c_hash": true, "auth_time": true,
	"azp": true, "uti": true, "rh": true, "ver": true, "sid": true, "jti": true, "tid": true, "oid": true, "s_hash": true}

// claimOrder puts the claims people read first at the top.
var claimOrder = []string{"iss", "sub", "email", "email_verified", "name", "given_name", "family_name", "preferred_username", "groups", "roles", "amr", "acr", "hd"}

// formatClaims prints claims like the design: one per line in a familiar
// order, short arrays inline, protocol claims left out.
func formatClaims(c map[string]any) string {
	keys := make([]string, 0, len(c))
	for k := range c {
		if !protocolClaims[k] && !strings.HasPrefix(k, "xms_") {
			keys = append(keys, k)
		}
	}
	rank := func(k string) int {
		if i := slices.Index(claimOrder, k); i >= 0 {
			return i
		}
		return len(claimOrder)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if ra, rb := rank(a), rank(b); ra != rb {
			return ra - rb
		}
		return strings.Compare(a, b)
	})
	var b strings.Builder
	b.WriteString("{\n")
	for i, k := range keys {
		b.WriteString("  " + jsonText(k) + ": " + claimValue(c[k]))
		if i < len(keys)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("}")
	return b.String()
}

func claimValue(v any) string {
	if arr, ok := v.([]any); ok {
		parts := make([]string, len(arr))
		for i, e := range arr {
			parts[i] = claimValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	if s, ok := v.(string); ok {
		v = shortValue(s)
	}
	return jsonText(v)
}

// jsonText is v as JSON without HTML escaping ("&" stays "&").
func jsonText(v any) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return strings.TrimSuffix(b.String(), "\n")
}

var guidRe = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// shortValue shortens identifiers in the debug view like the design: a
// GUID value to 3f2a9d1e-…-c91, a GUID inside a URL to 7c1a…5c6d, and a
// long opaque subject to its first 24 characters.
func shortValue(s string) string {
	if len(s) == 36 && guidRe.MatchString(s) {
		return shortID(s)
	}
	s = guidRe.ReplaceAllStringFunc(s, func(g string) string { return g[:4] + "…" + g[32:] })
	if len(s) > 32 && !strings.ContainsAny(s, "/@. ") {
		return s[:24] + "…"
	}
	return s
}

// SSOTestView is the last test sign-in.
type SSOTestView struct {
	At    time.Time `json:"at"`
	By    string    `json:"by,omitempty"`
	Error string    `json:"error,omitempty"`
	// Claims is the pretty-printed claims object.
	Claims  string           `json:"claims,omitempty"`
	Samples []SSOClaimSample `json:"samples"`
	Outcome []SSOTestOutcome `json:"outcome"`
	// GroupsMissing: mappings exist but the IdP sent no groups claim.
	GroupsMissing bool `json:"groupsMissing"`
	GroupsOverage bool `json:"groupsOverage"`
}

// SSORoleChoice is a role a mapping can grant.
type SSORoleChoice struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	// Grantable is false when you don't hold all of the role's permissions
	// in that workspace (escalation guard), or for Owner.
	Grantable bool   `json:"grantable"`
	Reason    string `json:"reason,omitempty"`
}

// SSOListChoice is a list a mapping can be scoped to.
type SSOListChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SSOWorkspaceChoice is a workspace for the JIT default and the mappings.
type SSOWorkspaceChoice struct {
	ID    string          `json:"id"`
	Slug  string          `json:"slug"`
	Name  string          `json:"name"`
	Roles []SSORoleChoice `json:"roles"`
	Lists []SSOListChoice `json:"lists"`
}

// SSOBreakGlassOwner is an Owner who keeps their own sign-in while the
// domain is enforced.
type SSOBreakGlassOwner struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Workspace string `json:"workspace"`
	// Methods they keep: "password + two-factor", "passkey".
	Keeps string `json:"keeps"`
}

// SSOProviderDetail is GET /admin/sso/{id}: everything the setup page shows.
type SSOProviderDetail struct {
	SSOProviderItem
	PresetInfo     SSOPreset            `json:"presetInfo"`
	RedirectURI    string               `json:"redirectUri"`
	ClientID       string               `json:"clientId"`
	Secret         SSOSecretInfo        `json:"secret"`
	Scopes         []string             `json:"scopes"`
	Claims         SSOClaimMap          `json:"claims"`
	TwoFactorMode  string               `json:"twoFactorMode" enum:"amr,trusted"`
	Discovery      *SSODiscoveryResult  `json:"discovery,omitempty"`
	GroupMappings  []SSOGroupMapping    `json:"groupMappings"`
	Test           *SSOTestView         `json:"test,omitempty"`
	Workspaces     []SSOWorkspaceChoice `json:"workspaces"`
	BreakGlassList []SSOBreakGlassOwner `json:"breakGlassOwners"`
	// EnableBlockers say what's missing before the provider can be enabled.
	EnableBlockers []string `json:"enableBlockers"`
	// Rejections are the latest sign-ins this provider turned away.
	Rejections []SSORejection `json:"rejections"`
}

// SSORejection is a recent rejected sign-in.
type SSORejection struct {
	At        time.Time `json:"at"`
	Email     string    `json:"email"`
	Reason    string    `json:"reason"`
	Reference string    `json:"reference,omitempty"`
}

func (s *Service) admin(ctx context.Context) (*store.User, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return nil, api.Err(http.StatusUnauthorized, api.CodeUnauthenticated, "Sign in to continue.")
	}
	return u, nil
}

// List returns every SSO provider.
func (s *Service) List(ctx context.Context) (SSOProviderList, error) {
	ps, err := s.Q.SSOProviders(ctx)
	if err != nil {
		return SSOProviderList{}, err
	}
	refs, err := s.loadRefs(ctx)
	if err != nil {
		return SSOProviderList{}, err
	}
	out := SSOProviderList{Items: make([]SSOProviderItem, 0, len(ps)), Presets: Presets}
	for _, p := range ps {
		out.Items = append(out.Items, s.item(ctx, p, refs))
	}
	return out, nil
}

// refs are the workspaces, roles and lists names resolve against.
type refs struct {
	roles  []store.SSOWorkspaceRolesRow
	owners []store.SSOOwnersRow
	lists  []store.SSOWorkspaceListsRow
	users  map[uuid.UUID]int // identities per provider
}

func (s *Service) loadRefs(ctx context.Context) (refs, error) {
	roles, err := s.Q.SSOWorkspaceRoles(ctx)
	if err != nil {
		return refs{}, err
	}
	owners, err := s.Q.SSOOwners(ctx)
	if err != nil {
		return refs{}, err
	}
	lists, err := s.Q.SSOWorkspaceLists(ctx)
	if err != nil {
		return refs{}, err
	}
	counts, err := s.Q.SSOProviderMemberCounts(ctx)
	if err != nil {
		return refs{}, err
	}
	r := refs{roles: roles, owners: owners, lists: lists, users: map[uuid.UUID]int{}}
	for _, c := range counts {
		if c.SsoProviderID != nil {
			r.users[*c.SsoProviderID] = int(c.Members)
		}
	}
	return r, nil
}

// listIn reports whether a list belongs to a workspace.
func (r refs) listIn(ws, list uuid.UUID) bool {
	for _, l := range r.lists {
		if l.WorkspaceID == ws && l.ID == list {
			return true
		}
	}
	return false
}

func (r refs) role(ws uuid.UUID, key string) (store.SSOWorkspaceRolesRow, bool) {
	for _, x := range r.roles {
		if x.WorkspaceID == ws && x.RoleKey == key {
			return x, true
		}
	}
	return store.SSOWorkspaceRolesRow{}, false
}

func (r refs) workspaceName(ws uuid.UUID) string {
	for _, x := range r.roles {
		if x.WorkspaceID == ws {
			return x.WorkspaceName
		}
	}
	return ""
}

// breakGlass lists the Owners at the domains who keep their own sign-in.
func (r refs) breakGlass(domains []string) []SSOBreakGlassOwner {
	out := []SSOBreakGlassOwner{}
	for _, o := range r.owners {
		if len(domains) > 0 && !slices.Contains(domains, emailDomain(o.Email)) {
			continue
		}
		keeps := "password + two-factor"
		switch {
		case o.HasPassword && !o.HasSecondFactor:
			keeps = "password (no second factor yet)"
		case !o.HasPassword && o.HasSecondFactor:
			keeps = "passkey"
		case !o.HasPassword:
			keeps = "nothing but SSO (no password or passkey)"
		}
		out = append(out, SSOBreakGlassOwner{Name: o.Name, Email: o.Email, Workspace: o.WorkspaceName, Keeps: keeps})
	}
	return out
}

func (s *Service) item(_ context.Context, p store.SsoProvider, r refs) SSOProviderItem {
	pre := PresetFor(p.Preset)
	it := SSOProviderItem{ID: p.ID.String(), Slug: p.Slug, Preset: p.Preset, Name: p.Name, Status: p.Status, Issuer: p.IssuerUrl,
		Domains: nonNil(p.AllowedDomains), GroupsClaim: pre.GroupsClaim, Mappings: len(mappingsOf(p)), Enforced: nonNil(p.EnforceDomains),
		EnabledAt: p.EnabledAt, BreakGlass: []string{}, Members: r.users[p.ID]}
	it.JIT.Enabled = p.JitEnabled
	if p.JitWorkspaceID != nil {
		it.JIT.WorkspaceID = p.JitWorkspaceID.String()
		it.JIT.WorkspaceName = r.workspaceName(*p.JitWorkspaceID)
		if p.JitRoleKey != nil {
			it.JIT.RoleKey = *p.JitRoleKey
			if role, ok := r.role(*p.JitWorkspaceID, *p.JitRoleKey); ok {
				it.JIT.RoleName = role.RoleName
			}
		}
	}
	if twoFAMode(p) == "trusted" {
		it.TwoFactor = pre.TrustedTwoFA
	} else {
		it.TwoFactor = "Counted when the ID token's amr claim includes mfa; otherwise the person's own second factor is asked for"
	}
	if d := discoveryOf(p); d != nil {
		ok := d.OK
		it.DiscoveryOK = &ok
	}
	domains := p.EnforceDomains
	if len(domains) == 0 {
		domains = p.AllowedDomains
	}
	for _, o := range r.breakGlass(domains) {
		it.BreakGlass = append(it.BreakGlass, o.Name)
	}
	return it
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// Get returns a provider with everything the setup page needs.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (SSOProviderDetail, error) {
	p, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOProviderDetail{}, api.MapError(err, "SSO provider")
	}
	return s.detail(ctx, p)
}

func (s *Service) detail(ctx context.Context, p store.SsoProvider) (SSOProviderDetail, error) {
	r, err := s.loadRefs(ctx)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	pre := PresetFor(p.Preset)
	d := SSOProviderDetail{SSOProviderItem: s.item(ctx, p, r), PresetInfo: pre, RedirectURI: s.publicURL(ctx) + RedirectPath(p.Slug),
		ClientID: p.ClientID, Scopes: nonNil(p.Scopes), Claims: claimsOf(p), TwoFactorMode: twoFAMode(p),
		GroupMappings: []SSOGroupMapping{}, Workspaces: []SSOWorkspaceChoice{}, EnableBlockers: []string{}, Rejections: []SSORejection{}}
	d.Secret = SSOSecretInfo{Set: len(p.ClientSecretEnc) > 0, SetAt: p.ClientSecretSetAt, ExpiresOn: p.ClientSecretExpiresOn}
	if p.ClientSecretSetBy != nil {
		if u, err := s.Q.GetUser(ctx, *p.ClientSecretSetBy); err == nil {
			d.Secret.SetBy = displayName(u)
		}
	}
	if disc := discoveryOf(p); disc != nil {
		res := &SSODiscoveryResult{OK: disc.OK, CheckedAt: disc.CheckedAt, URL: disc.URL, Status: disc.Status, Millis: disc.Millis, Error: disc.Error, Endpoints: []string{}, Keys: disc.Keys}
		for _, e := range []struct{ name, url string }{{"authorization", disc.AuthURL}, {"token", disc.TokenURL}, {"userinfo", disc.UserInfo}, {"JWKS", disc.JWKSURL}} {
			if e.url != "" {
				res.Endpoints = append(res.Endpoints, e.name)
			}
		}
		if disc.OK && len(res.Endpoints) == 0 {
			// Discovery recorded before endpoints were kept (demo, upgrades).
			res.Endpoints = []string{"authorization", "token", "userinfo", "JWKS"}
		}
		d.Discovery = res
	}
	for _, m := range mappingsOf(p) {
		lists := []string{}
		for _, l := range m.Scope.Lists {
			lists = append(lists, l.String())
		}
		d.GroupMappings = append(d.GroupMappings, SSOGroupMapping{Group: m.Group, Label: m.Label, WorkspaceID: m.WorkspaceID.String(), RoleKey: m.RoleKey, Lists: lists})
	}
	admin := api.UserFrom(ctx)
	d.Workspaces, err = s.choices(ctx, r, admin)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	domains := p.EnforceDomains
	if len(domains) == 0 {
		domains = p.AllowedDomains
	}
	d.BreakGlassList = r.breakGlass(domains)
	if t := lastTestOf(p); t != nil {
		d.Test, err = s.testView(ctx, p, t)
		if err != nil {
			return SSOProviderDetail{}, err
		}
	}
	d.EnableBlockers = blockers(p)
	rej, err := s.Q.SSOSignInFailures(ctx, p.ID.String())
	if err != nil {
		return SSOProviderDetail{}, err
	}
	for _, e := range rej {
		var meta struct {
			Reference string `json:"reference"`
		}
		_ = json.Unmarshal(e.Metadata, &meta)
		d.Rejections = append(d.Rejections, SSORejection{At: e.OccurredAt, Email: e.TargetName, Reason: e.Reason, Reference: meta.Reference})
	}
	return d, nil
}

// blockers lists what's missing before a provider can be enabled.
func blockers(p store.SsoProvider) []string {
	out := []string{}
	if d := discoveryOf(p); d == nil || !d.OK {
		out = append(out, "Run a successful discovery test of the issuer URL.")
	}
	if p.ClientID == "" {
		out = append(out, "Enter the client ID.")
	}
	if len(p.ClientSecretEnc) == 0 {
		out = append(out, "Save the client secret.")
	}
	return out
}

// choices lists the workspaces, the roles a mapping may grant (escalation
// guard: only roles whose permissions the admin holds there) and lists.
func (s *Service) choices(ctx context.Context, r refs, admin *store.User) ([]SSOWorkspaceChoice, error) {
	out := []SSOWorkspaceChoice{}
	index := map[uuid.UUID]int{}
	for _, row := range r.roles {
		i, ok := index[row.WorkspaceID]
		if !ok {
			out = append(out, SSOWorkspaceChoice{ID: row.WorkspaceID.String(), Slug: row.Slug, Name: row.WorkspaceName, Roles: []SSORoleChoice{}, Lists: []SSOListChoice{}})
			i = len(out) - 1
			index[row.WorkspaceID] = i
		}
		c := SSORoleChoice{Key: row.RoleKey, Name: row.RoleName, Grantable: true}
		if row.RoleKey == authz.RoleOwner && row.Builtin {
			c.Grantable, c.Reason = false, "Owner can't be granted by SSO; transfer ownership in workspace settings."
		} else if admin != nil {
			if err := s.canGrant(ctx, admin.ID, row); err != nil {
				c.Grantable, c.Reason = false, err.Error()
			}
		}
		out[i].Roles = append(out[i].Roles, c)
	}
	for _, l := range r.lists {
		if i, ok := index[l.WorkspaceID]; ok {
			out[i].Lists = append(out[i].Lists, SSOListChoice{ID: l.ID.String(), Name: l.Name})
		}
	}
	return out, nil
}

// canGrant applies the escalation guard: the admin must hold every
// permission of the role in that workspace.
func (s *Service) canGrant(ctx context.Context, adminID uuid.UUID, role store.SSOWorkspaceRolesRow) error {
	held := authz.Set{}
	if m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: role.WorkspaceID, UserID: adminID}); err == nil && m.Status == "active" {
		held = authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions)
	}
	want := authz.Effective(role.RoleKey, role.Builtin, role.Permissions)
	if missing := held.Missing(want); len(missing) > 0 {
		if len(held) == 0 {
			return fmt.Errorf("you aren't a member of %s, so you can't grant its roles", role.WorkspaceName)
		}
		return fmt.Errorf("you can't grant %s in %s: you don't hold %s there", role.RoleName, role.WorkspaceName, missing[0])
	}
	return nil
}

// testView renders the last test sign-in: claims, samples per field and
// what a real sign-in would do now (a dry run against current data).
func (s *Service) testView(ctx context.Context, p store.SsoProvider, t *testRun) (*SSOTestView, error) {
	v := &SSOTestView{At: t.At, By: t.By, Error: t.Error, Samples: []SSOClaimSample{}, Outcome: []SSOTestOutcome{}}
	if t.Error != "" || t.Claims == nil {
		return v, nil
	}
	v.Claims = formatClaims(t.Claims)
	id := extract(p, t.Claims)
	cm := claimsOf(p)
	pre := PresetFor(p.Preset)
	sample := func(field, claim string) SSOClaimSample {
		smp := SSOClaimSample{Field: field, Claim: claim}
		val, ok := t.Claims[claim]
		smp.Found = ok && claim != ""
		if !smp.Found {
			smp.Missing = "not sent by " + pre.ShortName
			return smp
		}
		switch x := val.(type) {
		case string:
			smp.Value = x
		case bool:
			smp.Value = fmt.Sprint(x)
		case []any:
			parts := []string{}
			for _, e := range x {
				parts = append(parts, fmt.Sprintf("%q", sampleID(fmt.Sprint(e))))
			}
			smp.Value = "[" + strings.Join(parts, ", ") + "]"
			smp.Count = len(x)
		default:
			b, _ := json.Marshal(x)
			smp.Value = string(b)
		}
		return smp
	}
	emailS := sample("email", cm.Email)
	verS := sample("email_verified", cm.EmailVerified)
	if !verS.Found {
		if len(p.AllowedDomains) > 0 {
			verS.Note = "treated as verified for allowed domains"
		} else {
			verS.Note = "treated as unverified: add an allowed domain to trust it"
		}
	}
	nameS := sample("name", cm.Name)
	groupsS := sample("groups", cm.Groups)
	if id.GroupsOverage {
		groupsS.Note = "too many groups: " + pre.Name + " sent an overage link instead"
	}
	v.Samples = []SSOClaimSample{emailS, verS, nameS}
	if pre.GroupsClaim || groupsS.Found {
		v.Samples = append(v.Samples, groupsS)
	}
	v.GroupsOverage = id.GroupsOverage
	pl, err := s.plan(ctx, s.Q, p, id)
	if err != nil {
		return nil, err
	}
	v.GroupsMissing = pl.GroupsMissing && pre.GroupsClaim
	v.Outcome = describe(p, id, pl)
	return v, nil
}

// shortID shortens long group object IDs like the design: 3f2a9d1e-…-c91.
func shortID(s string) string {
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return s[:8] + "-…-" + s[33:]
	}
	return s
}

// sampleID is the tighter form of the claim mapping's samples: 3f2a…c91.
func sampleID(s string) string {
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return s[:4] + "…" + s[33:]
	}
	return s
}

// SSOCreateInput starts a provider from a preset.
type SSOCreateInput struct {
	Preset string `json:"preset" enum:"google,entra,okta,keycloak,authentik,oidc"`
}

// Create adds a draft provider filled from its preset.
func (s *Service) Create(ctx context.Context, in SSOCreateInput) (SSOProviderDetail, error) {
	admin, err := s.admin(ctx)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	pre := PresetFor(in.Preset)
	existing, err := s.Q.SSOProviders(ctx)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	name := pre.Name
	for n := 2; slices.ContainsFunc(existing, func(p store.SsoProvider) bool { return strings.EqualFold(p.Name, name) }); n++ {
		name = fmt.Sprintf("%s %d", pre.Name, n)
	}
	var created store.SsoProvider
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		created, err = q.SSOCreateProvider(ctx, store.SSOCreateProviderParams{ID: uuid.Must(uuid.NewV7()), Preset: pre.Key, Name: name, IssuerUrl: pre.Issuer,
			Scopes: pre.Scopes, ClaimMap: storeClaims(defaultClaims()), AllowedDomains: []string{}, CreatedBy: &admin.ID})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "sso.provider.created", TargetType: "sso_provider", TargetID: created.ID.String(), TargetName: created.Name,
			After: map[string]any{"preset": pre.Key, "name": name}})
	})
	if err != nil {
		return SSOProviderDetail{}, err
	}
	return s.detail(ctx, created)
}

// SSOProviderInput is PUT /admin/sso/{id}. The client secret is write-only:
// omit it to keep the saved one.
type SSOProviderInput struct {
	// Preset switches a draft to another IdP family (enabled providers keep theirs).
	Preset    string `json:"preset,omitempty" enum:"google,entra,okta,keycloak,authentik,oidc"`
	Name      string `json:"name" minLength:"1" maxLength:"80"`
	IssuerURL string `json:"issuerUrl" maxLength:"300"`
	ClientID  string `json:"clientId" maxLength:"300"`
	// ClientSecret replaces the saved secret; it is never returned.
	ClientSecret string `json:"clientSecret,omitempty" maxLength:"4000"`
	// SecretExpiresOn is when the new secret expires at the IdP (YYYY-MM-DD).
	SecretExpiresOn string            `json:"secretExpiresOn,omitempty" maxLength:"10"`
	Scopes          []string          `json:"scopes" maxItems:"20"`
	Claims          SSOClaimMap       `json:"claims"`
	AllowedDomains  []string          `json:"allowedDomains" maxItems:"50"`
	JIT             SSOJITInput       `json:"jit"`
	GroupMappings   []SSOGroupMapping `json:"groupMappings" maxItems:"200"`
	TwoFactorMode   string            `json:"twoFactorMode" enum:"amr,trusted"`
}

// SSOJITInput sets just-in-time accounts.
type SSOJITInput struct {
	Enabled     bool   `json:"enabled"`
	WorkspaceID string `json:"workspaceId,omitempty" maxLength:"36"`
	RoleKey     string `json:"roleKey,omitempty" maxLength:"64"`
}

// Update saves a provider (draft or live). Changed group mappings and the
// JIT default role go through the escalation guard.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in SSOProviderInput) (SSOProviderDetail, error) {
	admin, err := s.admin(ctx)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	before, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOProviderDetail{}, api.MapError(err, "SSO provider")
	}
	switchPreset := in.Preset != "" && in.Preset != before.Preset
	if switchPreset && before.Status == "enabled" {
		return SSOProviderDetail{}, api.Conflict("Disable " + before.Name + " before switching it to another preset, or add a new provider.")
	}
	if switchPreset && strings.TrimSpace(in.Name) == before.Name && before.Name == PresetFor(before.Preset).Name {
		in.Name = PresetFor(in.Preset).Name
	}
	params, changes, err := s.validate(ctx, admin, before, in)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	if switchPreset {
		changes = append(changes, "preset "+PresetFor(in.Preset).Name)
	}
	var secret []byte
	var expires *time.Time
	if strings.TrimSpace(in.ClientSecret) != "" {
		secret, err = s.Box.SealString(strings.TrimSpace(in.ClientSecret), secretAD(id))
		if err != nil {
			return SSOProviderDetail{}, err
		}
		if in.SecretExpiresOn != "" {
			t, err := time.Parse("2006-01-02", in.SecretExpiresOn)
			if err != nil {
				return SSOProviderDetail{}, api.Invalid("body.secretExpiresOn", "Use a date like 2027-04-10.")
			}
			expires = &t
		}
		changes = append(changes, "client secret replaced")
	}
	var after store.SsoProvider
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if switchPreset {
			if err := q.SSOSetPreset(ctx, store.SSOSetPresetParams{ID: id, Preset: in.Preset, Name: params.Name}); err != nil {
				return err
			}
		}
		after, err = q.SSOUpdateProvider(ctx, params)
		if err != nil {
			return err
		}
		if secret != nil {
			if err := q.SSOSetClientSecret(ctx, store.SSOSetClientSecretParams{ID: id, ClientSecretEnc: secret, SetBy: &admin.ID, ExpiresOn: expires}); err != nil {
				return err
			}
		}
		summary := "saved"
		if before.Status != "enabled" {
			summary = "draft saved"
		}
		if len(changes) > 0 {
			summary += " · " + strings.Join(changes, " · ")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "sso.provider.updated", TargetType: "sso_provider", TargetID: id.String(), TargetName: after.Name,
			Before: providerAuditView(before), After: providerAuditView(after), Metadata: map[string]any{"summary": summary, "secret_replaced": secret != nil}})
	})
	if err != nil {
		return SSOProviderDetail{}, err
	}
	s.forget(before.IssuerUrl)
	if s.OnPolicyChange != nil {
		s.OnPolicyChange()
	}
	fresh, err := s.Q.SSOProvider(ctx, id)
	if err != nil {
		return SSOProviderDetail{}, err
	}
	return s.detail(ctx, fresh)
}

// providerAuditView is what the audit log keeps of a provider (no secret).
func providerAuditView(p store.SsoProvider) map[string]any {
	return map[string]any{"name": p.Name, "issuer": p.IssuerUrl, "client_id": p.ClientID, "scopes": p.Scopes, "claims": p.ClaimMap,
		"allowed_domains": p.AllowedDomains, "jit": p.JitEnabled, "jit_workspace_id": p.JitWorkspaceID, "jit_role": p.JitRoleKey,
		"group_mappings": p.GroupMappings, "enforced": p.EnforceDomains, "status": p.Status}
}

// validate checks a provider input and returns the update and a summary of
// what changed.
func (s *Service) validate(ctx context.Context, admin *store.User, before store.SsoProvider, in SSOProviderInput) (store.SSOUpdateProviderParams, []string, error) {
	var changes []string
	p := store.SSOUpdateProviderParams{ID: before.ID, Name: strings.TrimSpace(in.Name), IssuerUrl: strings.TrimSpace(in.IssuerURL), ClientID: strings.TrimSpace(in.ClientID)}
	if p.Name == "" {
		return p, nil, api.Invalid("body.name", "Give the provider a name people recognise on the sign-in page.")
	}
	if p.IssuerUrl != "" {
		u, err := url.Parse(p.IssuerUrl)
		if err != nil || u.Host == "" || (u.Scheme != "https" && (u.Scheme != "http" || !isLocalHost(u.Hostname()))) {
			return p, nil, api.Invalid("body.issuerUrl", "Enter the issuer as an https URL, like "+PresetFor(before.Preset).IssuerPattern+".")
		}
	}
	scopes := []string{}
	for _, sc := range in.Scopes {
		sc = strings.TrimSpace(sc)
		if sc == "" || slices.Contains(scopes, sc) {
			continue
		}
		if strings.ContainsAny(sc, " \t,") || len(sc) > 100 {
			return p, nil, api.Invalid("body.scopes", "Scopes are single words like openid, profile or groups.")
		}
		scopes = append(scopes, sc)
	}
	if !slices.Contains(scopes, "openid") {
		scopes = append([]string{"openid"}, scopes...)
	}
	p.Scopes = scopes
	claims := in.Claims
	def := defaultClaims()
	if strings.TrimSpace(claims.Email) == "" {
		claims.Email = def.Email
	}
	if strings.TrimSpace(claims.EmailVerified) == "" {
		claims.EmailVerified = def.EmailVerified
	}
	if strings.TrimSpace(claims.Name) == "" {
		claims.Name = def.Name
	}
	if strings.TrimSpace(claims.Groups) == "" {
		claims.Groups = def.Groups
	}
	p.ClaimMap = storeClaims(SSOClaimMap{Email: strings.TrimSpace(claims.Email), EmailVerified: strings.TrimSpace(claims.EmailVerified), Name: strings.TrimSpace(claims.Name), Groups: strings.TrimSpace(claims.Groups)})
	domains, err := normalizeDomains(in.AllowedDomains)
	if err != nil {
		return p, nil, api.Invalid("body.allowedDomains", err.Error())
	}
	p.AllowedDomains = domains
	if in.TwoFactorMode != "" {
		p.TwofaMode = &in.TwoFactorMode
	}
	r, err := s.loadRefs(ctx)
	if err != nil {
		return p, nil, err
	}
	p.JitEnabled = in.JIT.Enabled
	if in.JIT.WorkspaceID != "" || in.JIT.Enabled {
		ws, err := uuid.Parse(in.JIT.WorkspaceID)
		if err != nil {
			return p, nil, api.Invalid("body.jit.workspaceId", "Pick the workspace new people join.")
		}
		key := in.JIT.RoleKey
		if key == "" {
			key = authz.RoleViewer
		}
		role, ok := r.role(ws, key)
		if !ok {
			return p, nil, api.Invalid("body.jit.roleKey", "Pick a role of that workspace.")
		}
		if role.RoleKey == authz.RoleOwner {
			return p, nil, api.Invalid("body.jit.roleKey", "New accounts can't become Owner automatically.")
		}
		changed := before.JitWorkspaceID == nil || *before.JitWorkspaceID != ws || before.JitRoleKey == nil || *before.JitRoleKey != key
		if changed {
			if err := s.canGrant(ctx, admin.ID, role); err != nil {
				return p, nil, escalation(err, "body.jit.roleKey")
			}
		}
		p.JitWorkspaceID, p.JitRoleKey = &ws, &key
	}
	old := mappingsOf(before)
	maps := []mapping{}
	seen := map[string]bool{}
	for i, m := range in.GroupMappings {
		loc := fmt.Sprintf("body.groupMappings[%d]", i)
		group := strings.TrimSpace(m.Group)
		if group == "" {
			return p, nil, api.Invalid(loc+".group", "Enter the IdP group (an object ID for Entra ID).")
		}
		ws, err := uuid.Parse(m.WorkspaceID)
		if err != nil {
			return p, nil, api.Invalid(loc+".workspaceId", "Pick a workspace.")
		}
		if seen[group+"|"+ws.String()] {
			return p, nil, api.Invalid(loc+".group", "Group "+group+" is already mapped in that workspace.")
		}
		seen[group+"|"+ws.String()] = true
		role, ok := r.role(ws, m.RoleKey)
		if !ok {
			return p, nil, api.Invalid(loc+".roleKey", "Pick a role of that workspace.")
		}
		if role.RoleKey == authz.RoleOwner && role.Builtin {
			return p, nil, api.Err(http.StatusForbidden, api.CodeEscalation, "Owner can't be granted by a group mapping. Transfer ownership in the workspace's settings instead.")
		}
		var scope authz.Scope
		for _, l := range m.Lists {
			lid, err := uuid.Parse(l)
			if err != nil || !r.listIn(ws, lid) {
				return p, nil, api.Invalid(loc+".lists", "Pick lists of "+role.WorkspaceName+".")
			}
			if !slices.Contains(scope.Lists, lid) {
				scope.Lists = append(scope.Lists, lid)
			}
		}
		nm := mapping{Group: group, Label: strings.TrimSpace(m.Label), WorkspaceID: ws, RoleKey: role.RoleKey, Scope: scope}
		unchanged := slices.ContainsFunc(old, func(o mapping) bool {
			return o.Group == nm.Group && o.WorkspaceID == nm.WorkspaceID && o.RoleKey == nm.RoleKey && slices.Equal(o.Scope.Lists, nm.Scope.Lists)
		})
		if !unchanged {
			if err := s.canGrant(ctx, admin.ID, role); err != nil {
				return p, nil, escalation(err, loc+".roleKey")
			}
		}
		maps = append(maps, nm)
	}
	p.GroupMappings = mustJSON(maps)
	if len(maps) != len(old) {
		changes = append(changes, fmt.Sprintf("%d group %s", len(maps), plural(len(maps), "mapping", "mappings")))
	}
	if p.IssuerUrl != before.IssuerUrl {
		changes = append(changes, "issuer changed")
	}
	return p, changes, nil
}

func escalation(err error, loc string) error {
	p := api.Err(http.StatusForbidden, api.CodeEscalation, capitalize(err.Error())+". Ask an Owner of that workspace to add the mapping.")
	p.Errors = append(p.Errors, &huma.ErrorDetail{Location: loc, Message: err.Error()})
	return p
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
