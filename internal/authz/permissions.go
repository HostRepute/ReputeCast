// Package authz is ReputeCast's small, in-house authorization layer. Users,
// API keys and MCP agents all hold a set of permission keys (A5 §3.7) and go
// through the same Check. Nobody can grant a permission they don't hold.
package authz

import (
	"slices"
	"sort"
)

// Permission is a permission key such as "campaigns:send". The same keys are
// used for roles, API keys and MCP agents.
type Permission string

// Every permission ReputeCast knows, grouped as in the role editor.
const (
	CampaignsView    Permission = "campaigns:view"
	CampaignsEdit    Permission = "campaigns:edit"
	CampaignsSend    Permission = "campaigns:send"
	CampaignsPause   Permission = "campaigns:pause"
	CampaignsApprove Permission = "campaigns:approve"

	TemplatesView Permission = "templates:view"
	TemplatesEdit Permission = "templates:edit"

	TransactionalView    Permission = "transactional:view"
	TransactionalEdit    Permission = "transactional:edit"
	TransactionalPublish Permission = "transactional:publish"
	TransactionalSend    Permission = "transactional:send"
	TransactionalContent Permission = "transactional:content"

	AutomationsView    Permission = "automations:view"
	AutomationsEdit    Permission = "automations:edit"
	AutomationsPublish Permission = "automations:publish"

	SubscribersView   Permission = "subscribers:view"
	SubscribersCreate Permission = "subscribers:create"
	SubscribersEdit   Permission = "subscribers:edit"
	SubscribersImport Permission = "subscribers:import"
	SubscribersExport Permission = "subscribers:export"
	SubscribersDelete Permission = "subscribers:delete"
	SubscribersVerify Permission = "subscribers:verify"

	ListsManage  Permission = "lists:manage"
	SegmentsEdit Permission = "segments:edit"
	SegmentsSQL  Permission = "segments:sql"

	FormsEdit Permission = "forms:edit"

	ReportsView  Permission = "reports:view"
	ReportsShare Permission = "reports:share"

	DeliverabilityView     Permission = "deliverability:view"
	IntegrationsHostRepute Permission = "integrations:hostrepute"

	ProvidersView   Permission = "providers:view"
	ProvidersManage Permission = "providers:manage"
	DomainsManage   Permission = "domains:manage"

	TeamView   Permission = "team:view"
	TeamInvite Permission = "team:invite"
	TeamManage Permission = "team:manage"

	APIKeysManage  Permission = "apikeys:manage"
	WebhooksManage Permission = "webhooks:manage"
	MCPManage      Permission = "mcp:manage"

	IntegrationsView   Permission = "integrations:view"
	IntegrationsManage Permission = "integrations:manage"

	AIUse Permission = "ai:use"

	ScoringEdit Permission = "scoring:edit"

	PlacementRun    Permission = "placement:run"
	PlacementManage Permission = "placement:manage"

	GrowView    Permission = "grow:view"
	GrowEdit    Permission = "grow:edit"
	GrowReview  Permission = "grow:review"
	GrowConnect Permission = "grow:connect"

	EarnView          Permission = "earn:view"
	EarnEdit          Permission = "earn:edit"
	EarnManage        Permission = "earn:manage"
	PaymentsManage    Permission = "payments:manage"
	SponsorsManage    Permission = "sponsors:manage"
	ConversionsCreate Permission = "conversions:create"

	WebsiteView    Permission = "website:view"
	WebsiteEdit    Permission = "website:edit"
	WebsitePublish Permission = "website:publish"
	WebsiteManage  Permission = "website:manage"

	ProductsView    Permission = "products:view"
	ProductsEdit    Permission = "products:edit"
	ProductsPublish Permission = "products:publish"
	OrdersManage    Permission = "orders:manage"

	SMSSend      Permission = "sms:send"
	PushSend     Permission = "push:send"
	WhatsAppSend Permission = "whatsapp:send"

	AuditView Permission = "audit:view"

	WorkspaceManage Permission = "workspace:manage"

	// Owner-only actions. They are never grantable to custom roles, API keys
	// or agents; only the built-in Owner role holds them.
	WorkspaceTransfer Permission = "workspace:transfer"
	WorkspaceDelete   Permission = "workspace:delete"
)

// Def describes a permission for the role editor and API docs.
type Def struct {
	Key         Permission `json:"key"`
	Group       string     `json:"group"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	Sensitive   bool       `json:"sensitive"`
	// Implies lists permissions included automatically (edit includes view).
	Implies []Permission `json:"implies,omitempty"`
	// OwnerOnly permissions can't be granted to custom roles, keys or agents.
	OwnerOnly bool `json:"ownerOnly,omitempty"`
	// PeopleOnly permissions are held by members only: API keys and MCP
	// agents can't hold them.
	PeopleOnly bool `json:"peopleOnly,omitempty"`
	// NoAgents permissions can be held by members and API keys, never by
	// MCP agents.
	NoAgents bool `json:"noAgents,omitempty"`
}

// Groups in display order.
var Groups = []string{
	g("campaigns"), g("templates"), g("transactional"), g("automations"), g("contacts"), g("lists"), g("forms"),
	g("reports"), g("deliverability"), g("sending"), g("team"), g("developer"), g("integrations"), g("ai"),
	g("scoring"), g("placement"), g("grow"), g("earn"), g("website"), g("products"), g("sms"), g("push"),
	g("whatsapp"), g("audit"), g("workspace"),
}

// Defs is the catalogue of permissions in display order. Their labels and
// descriptions are in messages/<locale>.json (perm.<key>, perm.<key>.about).
var Defs = described([]Def{
	{Key: CampaignsView, Group: g("campaigns")},
	{Key: CampaignsEdit, Group: g("campaigns"), Implies: []Permission{CampaignsView}},
	{Key: CampaignsSend, Group: g("campaigns"), Sensitive: true, Implies: []Permission{CampaignsView}},
	{Key: CampaignsPause, Group: g("campaigns"), Implies: []Permission{CampaignsView}},
	{Key: CampaignsApprove, Group: g("campaigns"), Sensitive: true, Implies: []Permission{CampaignsView}},

	{Key: TemplatesView, Group: g("templates")},
	{Key: TemplatesEdit, Group: g("templates"), Implies: []Permission{TemplatesView}},

	{Key: TransactionalView, Group: g("transactional")},
	{Key: TransactionalEdit, Group: g("transactional"), Implies: []Permission{TransactionalView}},
	{Key: TransactionalPublish, Group: g("transactional"), Sensitive: true, Implies: []Permission{TransactionalView}},
	{Key: TransactionalSend, Group: g("transactional"), Sensitive: true},
	{Key: TransactionalContent, Group: g("transactional"), Sensitive: true, Implies: []Permission{TransactionalView}},

	{Key: AutomationsView, Group: g("automations")},
	{Key: AutomationsEdit, Group: g("automations"), Implies: []Permission{AutomationsView}},
	{Key: AutomationsPublish, Group: g("automations"), Sensitive: true, Implies: []Permission{AutomationsView}},

	{Key: SubscribersView, Group: g("contacts")},
	{Key: SubscribersCreate, Group: g("contacts"), Implies: []Permission{SubscribersView}},
	{Key: SubscribersEdit, Group: g("contacts"), Implies: []Permission{SubscribersView}},
	{Key: SubscribersImport, Group: g("contacts"), Implies: []Permission{SubscribersView}},
	{Key: SubscribersExport, Group: g("contacts"), Sensitive: true, Implies: []Permission{SubscribersView}},
	{Key: SubscribersDelete, Group: g("contacts"), Sensitive: true, Implies: []Permission{SubscribersView}},
	{Key: SubscribersVerify, Group: g("contacts"), Sensitive: true, NoAgents: true, Implies: []Permission{SubscribersView}},

	{Key: ListsManage, Group: g("lists"), Implies: []Permission{SubscribersView}},
	{Key: SegmentsEdit, Group: g("lists"), Implies: []Permission{SubscribersView}},
	{Key: SegmentsSQL, Group: g("lists"), Sensitive: true, Implies: []Permission{SegmentsEdit, SubscribersView}},

	{Key: FormsEdit, Group: g("forms")},

	{Key: ReportsView, Group: g("reports")},
	{Key: ReportsShare, Group: g("reports"), Implies: []Permission{ReportsView}},

	{Key: DeliverabilityView, Group: g("deliverability")},
	{Key: IntegrationsHostRepute, Group: g("deliverability"), Implies: []Permission{DeliverabilityView}},

	{Key: ProvidersView, Group: g("sending")},
	{Key: ProvidersManage, Group: g("sending"), Sensitive: true, Implies: []Permission{ProvidersView}},
	{Key: DomainsManage, Group: g("sending"), Sensitive: true, Implies: []Permission{ProvidersView}},

	{Key: TeamView, Group: g("team")},
	{Key: TeamInvite, Group: g("team"), Implies: []Permission{TeamView}},
	{Key: TeamManage, Group: g("team"), Sensitive: true, Implies: []Permission{TeamView, TeamInvite}},

	{Key: APIKeysManage, Group: g("developer"), Sensitive: true},
	{Key: WebhooksManage, Group: g("developer")},
	{Key: MCPManage, Group: g("developer"), Sensitive: true},

	{Key: IntegrationsView, Group: g("integrations")},
	{Key: IntegrationsManage, Group: g("integrations"), Sensitive: true, PeopleOnly: true, Implies: []Permission{IntegrationsView}},

	{Key: AIUse, Group: g("ai"), PeopleOnly: true},

	{Key: ScoringEdit, Group: g("scoring"), Implies: []Permission{SubscribersView}},

	{Key: PlacementRun, Group: g("placement"), NoAgents: true, Implies: []Permission{DeliverabilityView}},
	{Key: PlacementManage, Group: g("placement"), Sensitive: true, NoAgents: true, Implies: []Permission{DeliverabilityView}},

	{Key: GrowView, Group: g("grow")},
	{Key: GrowEdit, Group: g("grow"), Implies: []Permission{GrowView}},
	{Key: GrowReview, Group: g("grow"), Implies: []Permission{GrowView}},
	{Key: GrowConnect, Group: g("grow"), Sensitive: true, Implies: []Permission{GrowView}},

	{Key: EarnView, Group: g("earn")},
	{Key: EarnEdit, Group: g("earn"), Sensitive: true, Implies: []Permission{EarnView}},
	{Key: EarnManage, Group: g("earn"), Sensitive: true, Implies: []Permission{EarnView}},
	{Key: PaymentsManage, Group: g("earn"), Sensitive: true, Implies: []Permission{EarnView}},
	{Key: SponsorsManage, Group: g("earn"), Implies: []Permission{EarnView}},
	{Key: ConversionsCreate, Group: g("earn")},

	{Key: WebsiteView, Group: g("website")},
	{Key: WebsiteEdit, Group: g("website"), Implies: []Permission{WebsiteView}},
	{Key: WebsitePublish, Group: g("website"), Sensitive: true, Implies: []Permission{WebsiteView}},
	{Key: WebsiteManage, Group: g("website"), Sensitive: true, Implies: []Permission{WebsiteView}},

	{Key: ProductsView, Group: g("products")},
	{Key: ProductsEdit, Group: g("products"), Implies: []Permission{ProductsView}},
	{Key: ProductsPublish, Group: g("products"), Sensitive: true, Implies: []Permission{ProductsView}},
	{Key: OrdersManage, Group: g("products"), Implies: []Permission{ProductsView}},

	{Key: SMSSend, Group: g("sms"), Sensitive: true},
	{Key: PushSend, Group: g("push"), Sensitive: true},
	{Key: WhatsAppSend, Group: g("whatsapp"), Sensitive: true},

	{Key: AuditView, Group: g("audit")},

	{Key: WorkspaceManage, Group: g("workspace")},

	{Key: WorkspaceTransfer, Group: g("workspace"), Sensitive: true, OwnerOnly: true},
	{Key: WorkspaceDelete, Group: g("workspace"), Sensitive: true, OwnerOnly: true},
})

var defByKey = func() map[Permission]Def {
	m := make(map[Permission]Def, len(Defs))
	for _, d := range Defs {
		m[d.Key] = d
	}
	return m
}()

// Lookup returns the definition of a permission key.
func Lookup(p Permission) (Def, bool) {
	d, ok := defByKey[p]
	return d, ok
}

// Valid reports whether p is a known permission key.
func Valid(p Permission) bool { _, ok := defByKey[p]; return ok }

// IsSensitive reports whether p is flagged ⚑ (two-factor policy, approvals).
func IsSensitive(p Permission) bool { return defByKey[p].Sensitive }

// Grantable reports whether p may appear on a custom role, API key or agent.
func Grantable(p Permission) bool {
	d, ok := defByKey[p]
	return ok && !d.OwnerOnly
}

// HoldableBy reports whether an actor of type t may hold p: people-only
// permissions are kept from API keys and MCP agents, and no-agents ones
// from MCP agents.
func HoldableBy(p Permission, t ActorType) bool {
	d, ok := defByKey[p]
	if !ok {
		return false
	}
	switch t {
	case ActorAPIKey:
		return !d.PeopleOnly
	case ActorMCPAgent:
		return !d.PeopleOnly && !d.NoAgents
	}
	return true
}

// Set is an immutable-by-convention set of permissions.
type Set map[Permission]struct{}

// NewSet builds a set from keys, expanding implied permissions.
func NewSet(keys ...Permission) Set {
	s := make(Set, len(keys))
	var add func(Permission)
	add = func(p Permission) {
		if _, ok := s[p]; ok {
			return
		}
		s[p] = struct{}{}
		for _, imp := range defByKey[p].Implies {
			add(imp)
		}
	}
	for _, k := range keys {
		add(k)
	}
	return s
}

// ParseSet builds a set from string keys, dropping unknown ones.
func ParseSet(keys []string) Set {
	ps := make([]Permission, 0, len(keys))
	for _, k := range keys {
		if p := Permission(k); Valid(p) {
			ps = append(ps, p)
		}
	}
	return NewSet(ps...)
}

// Has reports whether the set contains p.
func (s Set) Has(p Permission) bool { _, ok := s[p]; return ok }

// HasAll reports whether the set contains every permission in ps.
func (s Set) HasAll(ps ...Permission) bool {
	for _, p := range ps {
		if !s.Has(p) {
			return false
		}
	}
	return true
}

// HasSensitive reports whether the set holds any ⚑ permission.
func (s Set) HasSensitive() bool {
	for p := range s {
		if IsSensitive(p) {
			return true
		}
	}
	return false
}

// Keys returns the permissions in catalogue order.
func (s Set) Keys() []Permission {
	out := make([]Permission, 0, len(s))
	for _, d := range Defs {
		if s.Has(d.Key) {
			out = append(out, d.Key)
		}
	}
	return out
}

// Strings returns the keys as strings in catalogue order.
func (s Set) Strings() []string {
	keys := s.Keys()
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = string(k)
	}
	return out
}

// HoldableBy returns the permissions of s that an actor of type t may hold.
func (s Set) HoldableBy(t ActorType) Set {
	out := make(Set, len(s))
	for p := range s {
		if HoldableBy(p, t) {
			out[p] = struct{}{}
		}
	}
	return out
}

// Intersect returns the permissions present in both sets.
func (s Set) Intersect(o Set) Set {
	out := make(Set)
	for p := range s {
		if o.Has(p) {
			out[p] = struct{}{}
		}
	}
	return out
}

// Missing returns the permissions of want that s lacks, sorted.
func (s Set) Missing(want Set) []Permission {
	var out []Permission
	for p := range want {
		if !s.Has(p) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// All returns every permission (the Owner set).
func All() Set {
	keys := make([]Permission, len(Defs))
	for i, d := range Defs {
		keys[i] = d.Key
	}
	return NewSet(keys...)
}

// AllGrantable returns every permission except owner-only ones (the Admin set).
func AllGrantable() Set {
	var keys []Permission
	for _, d := range Defs {
		if !d.OwnerOnly {
			keys = append(keys, d.Key)
		}
	}
	return NewSet(keys...)
}

// Built-in role keys.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// BuiltinRole describes one of the four read-only roles.
type BuiltinRole struct {
	Key         string
	Name        string
	Description string
	Permissions Set
}

// viewAll is what Viewers (and Editors) can read.
var viewAll = []Permission{
	CampaignsView, TemplatesView, TransactionalView, AutomationsView, SubscribersView, ReportsView,
	DeliverabilityView, ProvidersView, TeamView, AuditView, IntegrationsView, GrowView, EarnView,
	WebsiteView, ProductsView,
}

// BuiltinRoles in display order. Their permissions always come from here,
// never from the database, so upgrades can add permissions to them.
var BuiltinRoles = []BuiltinRole{
	{Key: RoleOwner, Name: "Owner", Description: roleAbout(RoleOwner), Permissions: All()},
	{Key: RoleAdmin, Name: "Admin", Description: roleAbout(RoleAdmin), Permissions: AllGrantable()},
	{Key: RoleEditor, Name: "Editor", Description: roleAbout(RoleEditor), Permissions: NewSet(append(slices.Clone(viewAll),
		CampaignsEdit, CampaignsSend, CampaignsPause, TemplatesEdit, AutomationsEdit, AutomationsPublish,
		SubscribersCreate, SubscribersEdit, SubscribersImport, ListsManage, SegmentsEdit, FormsEdit, ReportsShare,
		TransactionalEdit, TransactionalPublish, AIUse, ScoringEdit, PlacementRun, GrowEdit, GrowReview,
		SponsorsManage, WebsiteEdit, WebsitePublish, WebsiteManage, ProductsEdit, ProductsPublish, OrdersManage,
		SMSSend, PushSend, WhatsAppSend)...)},
	{Key: RoleViewer, Name: "Viewer", Description: roleAbout(RoleViewer), Permissions: NewSet(viewAll...)},
}

// Builtin returns the built-in role with the given key: one of the four, or
// a client role (agency.go), which only client workspaces have.
func Builtin(key string) (BuiltinRole, bool) {
	for _, r := range BuiltinRoles {
		if r.Key == key {
			return r, true
		}
	}
	return clientRole(key)
}
