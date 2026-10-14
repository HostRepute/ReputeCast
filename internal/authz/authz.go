package authz

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// ActorType identifies who is acting. The same values are written to the
// audit log's actor_type column.
type ActorType string

const (
	ActorUser     ActorType = "user"
	ActorAPIKey   ActorType = "api_key"
	ActorMCPAgent ActorType = "mcp_agent"
	ActorSystem   ActorType = "system"
)

// Scope optionally restricts a member, API key or agent to some lists or
// sender domains ("Editor, Beta Program list only"). Empty means everything.
type Scope struct {
	Lists         []uuid.UUID `json:"lists,omitempty"`
	SenderDomains []string    `json:"senderDomains,omitempty"`
}

// IsZero reports whether the scope places no restriction.
func (s Scope) IsZero() bool { return len(s.Lists) == 0 && len(s.SenderDomains) == 0 }

// AllowsList reports whether the scope includes the list.
func (s Scope) AllowsList(id uuid.UUID) bool {
	return len(s.Lists) == 0 || slices.Contains(s.Lists, id)
}

// AllowsLists reports whether the scope includes every list in ids.
func (s Scope) AllowsLists(ids []uuid.UUID) bool {
	for _, id := range ids {
		if !s.AllowsList(id) {
			return false
		}
	}
	return true
}

// AllowsSenderDomain reports whether the scope includes the sender domain.
func (s Scope) AllowsSenderDomain(domain string) bool {
	if len(s.SenderDomains) == 0 {
		return true
	}
	return slices.ContainsFunc(s.SenderDomains, func(d string) bool { return strings.EqualFold(d, domain) })
}

// Within reports whether s is at most as broad as parent (used so API keys
// never get a wider scope than their creator).
func (s Scope) Within(parent Scope) bool {
	if len(parent.Lists) > 0 {
		if len(s.Lists) == 0 {
			return false
		}
		for _, l := range s.Lists {
			if !slices.Contains(parent.Lists, l) {
				return false
			}
		}
	}
	if len(parent.SenderDomains) > 0 {
		if len(s.SenderDomains) == 0 {
			return false
		}
		for _, d := range s.SenderDomains {
			if !parent.AllowsSenderDomain(d) {
				return false
			}
		}
	}
	return true
}

// Principal is an authenticated actor acting inside one workspace (or at
// instance level when WorkspaceID is zero).
type Principal struct {
	Type ActorType
	// ID is the user id, API key id or agent id.
	ID   uuid.UUID
	Name string
	// UserID is the human behind the action: the user, or the key's creator.
	UserID          uuid.UUID
	Email           string
	WorkspaceID     uuid.UUID
	RoleKey         string
	RoleName        string
	Permissions     Set
	Scope           Scope
	IsInstanceAdmin bool
	SessionID       uuid.UUID
	// MFAVerified is true when the session passed a second factor (or SSO
	// asserted it), used by the "2FA for sensitive permissions" policy.
	MFAVerified bool
}

// Can reports whether the principal holds the permission.
func (p *Principal) Can(perm Permission) bool {
	return p != nil && p.Permissions.Has(perm)
}

// IsHuman reports whether a person (not a key or agent) is acting.
func (p *Principal) IsHuman() bool { return p != nil && p.Type == ActorUser }

// ErrDenied is wrapped by every authorization failure.
var ErrDenied = errors.New("permission denied")

// DeniedError explains a missing permission (principle 7: say why and who can help).
type DeniedError struct {
	Permission Permission
	// Helpers are names of members who hold the permission.
	Helpers []string
	Message string
}

func (e *DeniedError) Error() string { return e.Message }
func (e *DeniedError) Unwrap() error { return ErrDenied }

// Require returns a DeniedError when the principal lacks perm.
func Require(p *Principal, perm Permission) error {
	if p.Can(perm) {
		return nil
	}
	return &DeniedError{Permission: perm, Message: DeniedMessage(perm, nil)}
}

// DeniedMessage builds "Sending needs the campaigns:send permission. Ask Marcus Chen or Linh Nguyen."
func DeniedMessage(perm Permission, helpers []string) string {
	action := "This"
	if d, ok := Lookup(perm); ok {
		action = actionNoun(d)
	}
	msg := fmt.Sprintf("%s needs the %s permission.", action, perm)
	if len(helpers) > 0 {
		msg += " Ask " + joinNames(helpers) + "."
	}
	return msg
}

// actionNoun turns "Send or schedule campaigns" into "Sending" style phrases
// for the most common permissions and falls back to the label.
func actionNoun(d Def) string {
	switch d.Key {
	case CampaignsSend:
		return "Sending"
	case CampaignsPause:
		return "Pausing a send"
	case CampaignsApprove:
		return "Approving"
	case SubscribersExport:
		return "Exporting contacts"
	case SubscribersDelete:
		return "Erasing personal data"
	case ProvidersManage:
		return "Managing providers"
	case DomainsManage:
		return "Managing domains"
	case TeamInvite:
		return "Inviting members"
	case TeamManage:
		return "Managing the team"
	case SegmentsSQL:
		return "Writing SQL segments"
	case APIKeysManage:
		return "Managing API keys"
	case MCPManage:
		return "Managing MCP agents"
	case AutomationsPublish:
		return "Publishing automations"
	case WorkspaceManage:
		return "Changing workspace settings"
	}
	return d.Label
}

func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " or " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
	}
}

// EscalationError lists permissions a granter tried to give away without holding them.
type EscalationError struct {
	Missing []Permission
}

func (e *EscalationError) Error() string {
	keys := make([]string, len(e.Missing))
	for i, p := range e.Missing {
		keys[i] = string(p)
	}
	return fmt.Sprintf("you can't grant %s because you don't hold it", strings.Join(keys, ", "))
}
func (e *EscalationError) Unwrap() error { return ErrDenied }

// CheckGrant enforces "nobody can grant a permission they don't hold" and
// rejects unknown or owner-only keys. It returns the expanded set to store.
func CheckGrant(granter Set, wanted []Permission) (Set, error) {
	for _, p := range wanted {
		if !Valid(p) {
			return nil, fmt.Errorf("unknown permission %q", p)
		}
		if !Grantable(p) {
			return nil, &EscalationError{Missing: []Permission{p}}
		}
	}
	set := NewSet(wanted...)
	if missing := granter.Missing(set); len(missing) > 0 {
		return nil, &EscalationError{Missing: missing}
	}
	return set, nil
}

// Effective resolves the permissions of a role: built-in roles from code,
// custom roles from their stored keys.
func Effective(roleKey string, builtin bool, stored []string) Set {
	if builtin {
		if r, ok := Builtin(roleKey); ok {
			return r.Permissions
		}
	}
	return ParseSet(stored)
}
