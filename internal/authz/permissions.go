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
	"Campaigns", "Templates", "Transactional", "Automations", "Contacts",
	"Lists and segments", "Forms and pages", "Reports", "Deliverability",
	"Sending", "Team", "Developer", "Integrations", "AI", "Audience scoring",
	"Placement tests", "Grow", "Earn", "Website and pages",
	"Products and orders", "SMS", "Push", "WhatsApp", "Audit", "Workspace",
}

// Defs is the catalogue of permissions in display order.
var Defs = []Def{
	{Key: CampaignsView, Group: "Campaigns", Label: "View campaigns", Description: "See campaigns, their content, audience and status."},
	{Key: CampaignsEdit, Group: "Campaigns", Label: "Edit campaigns", Description: "Create and edit drafts, request approval.", Implies: []Permission{CampaignsView}},
	{Key: CampaignsSend, Group: "Campaigns", Label: "Send or schedule campaigns", Description: "Start or schedule a send, and resume a paused one.", Sensitive: true, Implies: []Permission{CampaignsView}},
	{Key: CampaignsPause, Group: "Campaigns", Label: "Pause or cancel a live send", Description: "Stop a send that is running. Resuming needs campaigns:send.", Implies: []Permission{CampaignsView}},
	{Key: CampaignsApprove, Group: "Campaigns", Label: "Approve campaigns", Description: "Approve, reject or request changes on sends that need approval.", Sensitive: true, Implies: []Permission{CampaignsView}},

	{Key: TemplatesView, Group: "Templates", Label: "View templates", Description: "See saved templates and sections."},
	{Key: TemplatesEdit, Group: "Templates", Label: "Edit templates", Description: "Create, edit and delete templates and saved sections.", Implies: []Permission{TemplatesView}},

	{Key: TransactionalView, Group: "Transactional", Label: "View templates, messages and reports", Description: "See transactional templates, the message log and its numbers. Never message content."},
	{Key: TransactionalEdit, Group: "Transactional", Label: "Edit transactional templates", Description: "Create and edit template drafts, import templates.", Implies: []Permission{TransactionalView}},
	{Key: TransactionalPublish, Group: "Transactional", Label: "Publish transactional templates", Description: "Make a draft the live version, or roll back to an earlier one.", Sensitive: true, Implies: []Permission{TransactionalView}},
	{Key: TransactionalSend, Group: "Transactional", Label: "Send through the API or SMTP", Description: "Send transactional messages from your product. Send-only keys hold this alone.", Sensitive: true},
	{Key: TransactionalContent, Group: "Transactional", Label: "View the content of sent messages", Description: "Open the rendered content of a sent message. Every view is logged.", Sensitive: true, Implies: []Permission{TransactionalView}},

	{Key: AutomationsView, Group: "Automations", Label: "View automations", Description: "See flows and their live stats."},
	{Key: AutomationsEdit, Group: "Automations", Label: "Edit automations", Description: "Change drafts of flows.", Implies: []Permission{AutomationsView}},
	{Key: AutomationsPublish, Group: "Automations", Label: "Publish automations", Description: "Publish, pause and resume flows.", Sensitive: true, Implies: []Permission{AutomationsView}},

	{Key: SubscribersView, Group: "Contacts", Label: "View contacts", Description: "See contacts, profiles and activity."},
	{Key: SubscribersCreate, Group: "Contacts", Label: "Create contacts (forms and API)", Description: "Add contacts through signup forms and the API.", Implies: []Permission{SubscribersView}},
	{Key: SubscribersEdit, Group: "Contacts", Label: "Edit contacts", Description: "Change fields, tags and list membership.", Implies: []Permission{SubscribersView}},
	{Key: SubscribersImport, Group: "Contacts", Label: "Import contacts", Description: "Run the importer. Consent is always kept.", Implies: []Permission{SubscribersView}},
	{Key: SubscribersExport, Group: "Contacts", Label: "Export contacts", Description: "Download contacts. Every export is logged.", Sensitive: true, Implies: []Permission{SubscribersView}},
	{Key: SubscribersDelete, Group: "Contacts", Label: "Erase personal data", Description: "Permanently erase a contact's personal data.", Sensitive: true, Implies: []Permission{SubscribersView}},
	{Key: SubscribersVerify, Group: "Contacts", Label: "Verify addresses with a service", Description: "Send contacts' addresses to a verification service. It spends credits; every use is logged.", Sensitive: true, NoAgents: true, Implies: []Permission{SubscribersView}},

	{Key: ListsManage, Group: "Lists and segments", Label: "Manage lists", Description: "Create, rename and delete lists.", Implies: []Permission{SubscribersView}},
	{Key: SegmentsEdit, Group: "Lists and segments", Label: "Edit segments", Description: "Build segments with the condition builder.", Implies: []Permission{SubscribersView}},
	{Key: SegmentsSQL, Group: "Lists and segments", Label: "Write SQL segments", Description: "Write read-only SQL against a contacts view.", Sensitive: true, Implies: []Permission{SegmentsEdit, SubscribersView}},

	{Key: FormsEdit, Group: "Forms and pages", Label: "Edit forms and public pages", Description: "Change signup forms, the preference center and the archive."},

	{Key: ReportsView, Group: "Reports", Label: "View reports", Description: "See campaign and automation reports."},
	{Key: ReportsShare, Group: "Reports", Label: "Create public read-only report links", Description: "Share a report with people outside the workspace.", Implies: []Permission{ReportsView}},

	{Key: DeliverabilityView, Group: "Deliverability", Label: "View deliverability", Description: "See domain, IP and provider health."},
	{Key: IntegrationsHostRepute, Group: "Deliverability", Label: "Connect HostRepute", Description: "Connect, reconnect or disconnect HostRepute.", Implies: []Permission{DeliverabilityView}},

	{Key: ProvidersView, Group: "Sending", Label: "View sending providers", Description: "See provider status and routing. Never credentials."},
	{Key: ProvidersManage, Group: "Sending", Label: "Manage providers, credentials, routing and failover", Description: "Connect providers, replace credentials, change routing.", Sensitive: true, Implies: []Permission{ProvidersView}},
	{Key: DomainsManage, Group: "Sending", Label: "Manage domains and senders", Description: "Add sending domains and sender identities.", Sensitive: true, Implies: []Permission{ProvidersView}},

	{Key: TeamView, Group: "Team", Label: "View the team", Description: "See members, roles and invitations."},
	{Key: TeamInvite, Group: "Team", Label: "Invite members", Description: "Invite people with roles you hold.", Implies: []Permission{TeamView}},
	{Key: TeamManage, Group: "Team", Label: "Manage roles and remove members", Description: "Change roles, deactivate and remove members.", Sensitive: true, Implies: []Permission{TeamView, TeamInvite}},

	{Key: APIKeysManage, Group: "Developer", Label: "Manage API keys", Description: "Create, rotate and revoke API keys.", Sensitive: true},
	{Key: WebhooksManage, Group: "Developer", Label: "Manage webhooks", Description: "Add webhook endpoints and see delivery logs."},
	{Key: MCPManage, Group: "Developer", Label: "Manage MCP agents", Description: "Create and revoke AI agents and see their actions.", Sensitive: true},

	{Key: IntegrationsView, Group: "Integrations", Label: "See integrations, their activity and webhooks", Description: "See the integrations hub, each integration's health, activity and managed webhooks."},
	{Key: IntegrationsManage, Group: "Integrations", Label: "Connect, change, take over and disconnect integrations", Description: "Creates, changes and revokes the integrations' keys and webhooks.", Sensitive: true, PeopleOnly: true, Implies: []Permission{IntegrationsView}},

	{Key: AIUse, Group: "AI", Label: "Use the AI assistant", Description: "Subject ideas, rewrites, translations and alt text with the workspace's own model.", PeopleOnly: true},

	{Key: ScoringEdit, Group: "Audience scoring", Label: "Edit scoring rules, decay and bands", Description: "Change how contacts are scored.", Implies: []Permission{SubscribersView}},

	{Key: PlacementRun, Group: "Placement tests", Label: "Run a placement test", Description: "Send copies through the workspace's providers to its own seed mailboxes.", NoAgents: true, Implies: []Permission{DeliverabilityView}},
	{Key: PlacementManage, Group: "Placement tests", Label: "Add, reconnect and remove seed mailboxes", Description: "Their passwords and tokens can read whole mailboxes.", Sensitive: true, NoAgents: true, Implies: []Permission{DeliverabilityView}},

	{Key: GrowView, Group: "Grow", Label: "View referral and recommendation results", Description: "See referrals, referrers, rewards and recommendations."},
	{Key: GrowEdit, Group: "Grow", Label: "Edit the referral program, its rewards and recommendations", Description: "Change the program, its milestones and rewards, and the newsletters you recommend.", Implies: []Permission{GrowView}},
	{Key: GrowReview, Group: "Grow", Label: "Review held referrals, pause links, resend and fulfill rewards", Description: "Decide on held referrals and handle rewards.", Implies: []Permission{GrowView}},
	{Key: GrowConnect, Group: "Grow", Label: "Connect, pause and end partner links", Description: "Exchange signed subscribe links with other ReputeCast instances.", Sensitive: true, Implies: []Permission{GrowView}},

	{Key: EarnView, Group: "Earn", Label: "View revenue, paid members, gifts and sponsors", Description: "See Earn's numbers, paid members, gifts and sponsor bookings."},
	{Key: EarnEdit, Group: "Earn", Label: "Edit tiers, prices, offers, gift options and the Earn pages", Description: "Change what readers pay and the upgrade, cancel and gift pages.", Sensitive: true, Implies: []Permission{EarnView}},
	{Key: EarnManage, Group: "Earn", Label: "Manage paid members: complimentary access, imports, cancellations", Description: "Grant complimentary access, import paid members, cancel subscriptions and change gifts.", Sensitive: true, Implies: []Permission{EarnView}},
	{Key: PaymentsManage, Group: "Earn", Label: "Connect and manage payment providers, their keys and webhooks", Description: "Connect Stripe and change its keys and webhook.", Sensitive: true, Implies: []Permission{EarnView}},
	{Key: SponsorsManage, Group: "Earn", Label: "Manage sponsors, bookings and sponsor payment status", Description: "Keep the sponsor library and book sponsors into issues.", Implies: []Permission{EarnView}},
	{Key: ConversionsCreate, Group: "Earn", Label: "Record conversions through the API", Description: "Report a conversion with its value. The Conversions key preset holds this alone."},

	{Key: WebsiteView, Group: "Website and pages", Label: "View posts, pages and website numbers", Description: "See the website's posts, pages, members and numbers."},
	{Key: WebsiteEdit, Group: "Website and pages", Label: "Edit posts and pages", Description: "Write and change drafts of posts and pages.", Implies: []Permission{WebsiteView}},
	{Key: WebsitePublish, Group: "Website and pages", Label: "Publish, schedule and unpublish posts and pages", Description: "Put posts and pages on the website, or take them down.", Sensitive: true, Implies: []Permission{WebsiteView}},
	{Key: WebsiteManage, Group: "Website and pages", Label: "Manage the website: turn it on or off, home, navigation, tags, members, meter and paywall", Description: "Change how the website works and who can read what.", Sensitive: true, Implies: []Permission{WebsiteView}},

	{Key: ProductsView, Group: "Products and orders", Label: "View products and orders", Description: "See digital products, their sales and orders."},
	{Key: ProductsEdit, Group: "Products and orders", Label: "Edit products", Description: "Change products' files, descriptions and drafts.", Implies: []Permission{ProductsView}},
	{Key: ProductsPublish, Group: "Products and orders", Label: "Publish, archive and change the price of products", Description: "Put products on sale, take them off and change what they cost.", Sensitive: true, Implies: []Permission{ProductsView}},
	{Key: OrdersManage, Group: "Products and orders", Label: "Manage orders: new links, revoking and restoring access", Description: "Send new download links and change buyers' access.", Implies: []Permission{ProductsView}},

	{Key: SMSSend, Group: "SMS", Label: "Send or schedule SMS campaigns, and publish automations with SMS steps", Description: "Needed with campaigns:send to send text messages.", Sensitive: true},
	{Key: PushSend, Group: "Push", Label: "Send or schedule push campaigns, and publish automations with push steps", Description: "Needed with campaigns:send to send push notifications.", Sensitive: true},
	{Key: WhatsAppSend, Group: "WhatsApp", Label: "Send or schedule WhatsApp campaigns, and publish automations with WhatsApp steps", Description: "Needed with campaigns:send to send WhatsApp messages.", Sensitive: true},

	{Key: AuditView, Group: "Audit", Label: "View the audit log", Description: "See who did what, when and from where."},

	{Key: WorkspaceManage, Group: "Workspace", Label: "Manage workspace settings", Description: "Name, default sender, timezone and the approval policy."},

	{Key: WorkspaceTransfer, Group: "Workspace", Label: "Transfer ownership", Description: "Make another member the Owner.", Sensitive: true, OwnerOnly: true},
	{Key: WorkspaceDelete, Group: "Workspace", Label: "Delete the workspace", Description: "Delete the workspace and everything in it.", Sensitive: true, OwnerOnly: true},
}

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
	{Key: RoleOwner, Name: "Owner", Description: "Everything, including transferring ownership and deleting the workspace", Permissions: All()},
	{Key: RoleAdmin, Name: "Admin", Description: "Everything except transferring ownership and deleting the workspace", Permissions: AllGrantable()},
	{Key: RoleEditor, Name: "Editor", Description: "View everything; edit campaigns, templates, automations, segments (no SQL) and forms; send, pause and publish; import contacts", Permissions: NewSet(append(slices.Clone(viewAll),
		CampaignsEdit, CampaignsSend, CampaignsPause, TemplatesEdit, AutomationsEdit, AutomationsPublish,
		SubscribersCreate, SubscribersEdit, SubscribersImport, ListsManage, SegmentsEdit, FormsEdit, ReportsShare,
		TransactionalEdit, TransactionalPublish, AIUse, ScoringEdit, PlacementRun, GrowEdit, GrowReview,
		SponsorsManage, WebsiteEdit, WebsitePublish, WebsiteManage, ProductsEdit, ProductsPublish, OrdersManage,
		SMSSend, PushSend, WhatsAppSend)...)},
	{Key: RoleViewer, Name: "Viewer", Description: "Read-only. Never sees secrets", Permissions: NewSet(viewAll...)},
}

// Builtin returns the built-in role with the given key.
func Builtin(key string) (BuiltinRole, bool) {
	for _, r := range BuiltinRoles {
		if r.Key == key {
			return r, true
		}
	}
	return BuiltinRole{}, false
}
