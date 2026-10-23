package authz

// Agency accounts (design expansion phase 8, M32; channels and agencies
// brief §3.10–§3.11). An agency runs one instance for its clients, each in
// its own workspace.
//
// The agency permissions are instance-level: the agency roles (Agency
// manager, Account manager) hold them, never a workspace role, an API key
// or an MCP agent. They are kept out of Defs on purpose, so Valid rejects
// them everywhere a workspace permission is checked: the role editor, API
// keys and MCP agents can't be given them.
const (
	// AgencyView: the agency overview and search across your clients.
	AgencyView Permission = "agency:view"
	// AgencyUsage: usage and costs per client, rebill rates and the export.
	AgencyUsage Permission = "agency:usage"
	// AgencyLibrary: the shared library and workspace templates.
	AgencyLibrary Permission = "agency:library"
	// AgencyManage: client workspaces, staff and assignments.
	AgencyManage Permission = "agency:manage"
)

// AgencyDefs describes the agency permissions in display order (Instance
// admin and the agency's Staff screen). Group is "Agency".
var AgencyDefs = described([]Def{
	{Key: AgencyView, Group: g("agency"), PeopleOnly: true},
	{Key: AgencyUsage, Group: g("agency"), PeopleOnly: true},
	{Key: AgencyLibrary, Group: g("agency"), PeopleOnly: true},
	{Key: AgencyManage, Group: g("agency"), Sensitive: true, PeopleOnly: true},
})

// IsAgencyPermission reports whether p is one of the agency permissions.
func IsAgencyPermission(p Permission) bool {
	for _, d := range AgencyDefs {
		if d.Key == p {
			return true
		}
	}
	return false
}

// AgencySet builds a set of agency permissions. NewSet would work too, but
// agency keys imply nothing and are never part of a workspace set.
func AgencySet(keys ...Permission) Set {
	s := make(Set, len(keys))
	for _, k := range keys {
		if IsAgencyPermission(k) {
			s[k] = struct{}{}
		}
	}
	return s
}

// AgencyStrings lists the agency permissions of s as strings, in the order
// of AgencyDefs (Set.Strings follows the workspace catalogue, where they
// never appear).
func AgencyStrings(s Set) []string {
	out := []string{}
	for _, d := range AgencyDefs {
		if s.Has(d.Key) {
			out = append(out, string(d.Key))
		}
	}
	return out
}

// Agency staff levels.
const (
	AgencyManagerLevel  = "agency_manager"
	AccountManagerLevel = "account_manager"
)

// AgencyRole is an instance-level agency role.
type AgencyRole struct {
	Key         string
	Name        string
	Description string
	Permissions Set
}

// AgencyRoles in display order. Instance admins are agency managers.
var AgencyRoles = []AgencyRole{
	{Key: AgencyManagerLevel, Name: "Agency manager", Description: roleAbout(AgencyManagerLevel),
		Permissions: AgencySet(AgencyView, AgencyUsage, AgencyLibrary, AgencyManage)},
	{Key: AccountManagerLevel, Name: "Account manager", Description: roleAbout(AccountManagerLevel),
		Permissions: AgencySet(AgencyView, AgencyUsage, AgencyLibrary)},
}

// Agency returns the agency role with the given level.
func Agency(level string) (AgencyRole, bool) {
	for _, r := range AgencyRoles {
		if r.Key == level {
			return r, true
		}
	}
	return AgencyRole{}, false
}

// Client roles: built-in in every client workspace (and only there), read
// only like the other built-in roles, and duplicable.
const (
	RoleClientApprover = "client_approver"
	RoleClientViewer   = "client_viewer"
)

// ClientRoles in display order. Their permissions come from here, never
// from the database, like the other built-in roles.
var ClientRoles = []BuiltinRole{
	{Key: RoleClientApprover, Name: "Client approver", Description: roleAbout(RoleClientApprover),
		Permissions: NewSet(CampaignsView, CampaignsApprove, TemplatesView, AutomationsView, ReportsView, DeliverabilityView, SubscribersView)},
	{Key: RoleClientViewer, Name: "Client viewer", Description: roleAbout(RoleClientViewer),
		Permissions: NewSet(CampaignsView, TemplatesView, AutomationsView, ReportsView, DeliverabilityView)},
}

// IsClientRole reports whether key is one of the client roles.
func IsClientRole(key string) bool { return key == RoleClientApprover || key == RoleClientViewer }

func clientRole(key string) (BuiltinRole, bool) {
	for _, r := range ClientRoles {
		if r.Key == key {
			return r, true
		}
	}
	return BuiltinRole{}, false
}
