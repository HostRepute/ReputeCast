// Package access holds the presentation side of authorization shared by the
// team, roles, API key and MCP screens: the permission catalogue as the API
// returns it, scope labels ("Beta Program list only"), sign-in method and
// two-factor labels, and the "who can help" helper names.
package access

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// PermissionInfo describes one permission key for the role editor, API key
// and agent pickers.
type PermissionInfo struct {
	Key         string   `json:"key" example:"campaigns:send"`
	Group       string   `json:"group" example:"Campaigns"`
	Label       string   `json:"label" example:"Send or schedule campaigns"`
	Description string   `json:"description"`
	Sensitive   bool     `json:"sensitive"`
	Implies     []string `json:"implies"`
	OwnerOnly   bool     `json:"ownerOnly"`
}

// Catalog returns every permission in display order.
func Catalog() []PermissionInfo {
	out := make([]PermissionInfo, 0, len(authz.Defs))
	for _, d := range authz.Defs {
		imp := make([]string, 0, len(d.Implies))
		for _, p := range d.Implies {
			imp = append(imp, string(p))
		}
		out = append(out, PermissionInfo{Key: string(d.Key), Group: d.Group, Label: d.Label, Description: d.Description, Sensitive: d.Sensitive, Implies: imp, OwnerOnly: d.OwnerOnly})
	}
	return out
}

// Groups returns the permission groups in display order.
func Groups() []string { return slices.Clone(authz.Groups) }

// ImpliedBy returns the permission in set that implies p ("campaigns:edit"
// implies "campaigns:view"), or "" when p stands on its own.
func ImpliedBy(set authz.Set, p authz.Permission) authz.Permission {
	for _, d := range authz.Defs {
		if d.Key == p || !set.Has(d.Key) {
			continue
		}
		for _, imp := range d.Implies {
			if imp == p {
				return d.Key
			}
		}
	}
	return ""
}

// Verb returns the action part of a permission key: "campaigns:edit" → "edit".
func Verb(p authz.Permission) string {
	s := string(p)
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return s[i+1:]
	}
	return s
}

// ParseScope reads a stored scope ({} = everything).
func ParseScope(raw json.RawMessage) authz.Scope {
	var s authz.Scope
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

// ScopeJSON encodes a scope for storage.
func ScopeJSON(s authz.Scope) json.RawMessage {
	if s.IsZero() {
		return json.RawMessage(`{}`)
	}
	b, _ := json.Marshal(s)
	return b
}

// ScopedList is one list in a scope.
type ScopedList struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ScopeSummary is a scope as shown in the UI.
type ScopeSummary struct {
	Lists         []ScopedList `json:"lists"`
	SenderDomains []string    `json:"senderDomains"`
	// Label is "" for no restriction, otherwise "Beta Program list only",
	// "Weekly Digest only" or "2 lists · news.acme.io".
	Label string `json:"label"`
	// All is true when the scope places no restriction ("All lists").
	All bool `json:"all"`
}

// ListNames maps the workspace's list ids to names.
func ListNames(ctx context.Context, q *store.Queries, wsID uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := q.TeamWorkspaceLists(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out, nil
}

// View describes a scope with list names. listWord adds " list" after a single
// list name ("Beta Program list only", used for members) or not ("Weekly
// Digest only", used for keys).
func View(s authz.Scope, names map[uuid.UUID]string, listWord bool) ScopeSummary {
	v := ScopeSummary{Lists: []ScopedList{}, SenderDomains: []string{}, All: s.IsZero()}
	for _, id := range s.Lists {
		v.Lists = append(v.Lists, ScopedList{ID: id.String(), Name: names[id]})
	}
	v.SenderDomains = append(v.SenderDomains, s.SenderDomains...)
	switch {
	case v.All:
	case len(v.Lists) == 1 && len(v.SenderDomains) == 0:
		if listWord {
			v.Label = v.Lists[0].Name + " list only"
		} else {
			v.Label = v.Lists[0].Name + " only"
		}
	case len(v.Lists) == 0 && len(v.SenderDomains) == 1:
		v.Label = v.SenderDomains[0] + " only"
	default:
		var parts []string
		if n := len(v.Lists); n > 0 {
			if n <= 2 {
				names := make([]string, n)
				for i, l := range v.Lists {
					names[i] = l.Name
				}
				parts = append(parts, strings.Join(names, " and "))
			} else {
				parts = append(parts, fmt.Sprintf("%d lists", n))
			}
		}
		parts = append(parts, v.SenderDomains...)
		v.Label = strings.Join(parts, " · ") + " only"
	}
	return v
}

// ScopeRequest is the request shape of a scope.
type ScopeRequest struct {
	Lists         []string `json:"lists,omitempty" maxItems:"100" doc:"List ids"`
	SenderDomains []string `json:"senderDomains,omitempty" maxItems:"20"`
}

// Resolve validates a requested scope against the workspace's lists and
// sender domains.
func (in *ScopeRequest) Resolve(lists map[uuid.UUID]string, domains []string) (authz.Scope, error) {
	var s authz.Scope
	if in == nil {
		return s, nil
	}
	seen := map[uuid.UUID]bool{}
	for _, raw := range in.Lists {
		id, err := uuid.Parse(raw)
		if err != nil {
			return s, fmt.Errorf("unknown list %q", raw)
		}
		if _, ok := lists[id]; !ok {
			return s, fmt.Errorf("unknown list %q", raw)
		}
		if !seen[id] {
			seen[id] = true
			s.Lists = append(s.Lists, id)
		}
	}
	for _, d := range in.SenderDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if !slices.Contains(domains, d) {
			return s, fmt.Errorf("unknown sender domain %q", d)
		}
		if !slices.Contains(s.SenderDomains, d) {
			s.SenderDomains = append(s.SenderDomains, d)
		}
	}
	return s, nil
}

// MemberSignInMethod is one way a person signs in.
type MemberSignInMethod struct {
	Kind string `json:"kind" enum:"password,passkey,sso,google,microsoft,github"`
	// Label is "Password", "passkey" or "Google Workspace SSO".
	Label string `json:"label"`
	// Preset of the SSO provider (google, entra, okta…) for the icon.
	Preset string `json:"preset,omitempty"`
}

// Identity is a linked social or SSO identity.
type Identity struct {
	Kind           string
	ProviderName   string
	ProviderPreset string
}

// SignInMethods lists how a person signs in, password first.
func SignInMethods(hasPassword bool, passkeys int64, ids []Identity) []MemberSignInMethod {
	out := []MemberSignInMethod{}
	for _, id := range ids {
		switch id.Kind {
		case "sso":
			label := "SSO"
			if id.ProviderName != "" {
				label = id.ProviderName + " SSO"
			}
			out = append(out, MemberSignInMethod{Kind: "sso", Label: label, Preset: id.ProviderPreset})
		case "google":
			out = append(out, MemberSignInMethod{Kind: "google", Label: "Google", Preset: "google"})
		case "microsoft":
			out = append(out, MemberSignInMethod{Kind: "microsoft", Label: "Microsoft", Preset: "microsoft"})
		case "github":
			out = append(out, MemberSignInMethod{Kind: "github", Label: "GitHub", Preset: "github"})
		}
	}
	var head []MemberSignInMethod
	if hasPassword {
		head = append(head, MemberSignInMethod{Kind: "password", Label: "Password"})
	}
	if passkeys > 0 {
		label := "passkey"
		if !hasPassword {
			label = "Passkey"
		}
		head = append(head, MemberSignInMethod{Kind: "passkey", Label: label})
	}
	return append(head, out...)
}

// PrimarySignIn is the sign-in shown in the team list: the member's own
// credentials (password, passkey) when they have any, otherwise their SSO or
// social sign-ins. Linked identities next to a password are listed on the
// member page only.
func PrimarySignIn(all []MemberSignInMethod) []MemberSignInMethod {
	own := []MemberSignInMethod{}
	for _, m := range all {
		if m.Kind == "password" || m.Kind == "passkey" {
			own = append(own, m)
		}
	}
	if len(own) > 0 {
		return own
	}
	return append([]MemberSignInMethod{}, all...)
}

// MemberTwoFA is a member's two-factor status (A5 §5.8): On, Off, Via IdP or
// "Required, grace ends in N days".
type MemberTwoFA struct {
	State string `json:"state" enum:"on,off,idp,required"`
	// Method is "TOTP" or "passkey" when on.
	Method     string     `json:"method,omitempty"`
	GraceUntil *time.Time `json:"graceUntil,omitempty"`
	// GraceDays is the whole days left in the grace period (0 = ends today).
	GraceDays *int `json:"graceDays,omitempty"`
}

// TwoFAInput is what the two-factor status is computed from.
type TwoFAInput struct {
	TOTP       bool
	Passkeys   int64
	SSOOnly    bool // signs in only through an IdP that enforces its own second factor
	GraceUntil *time.Time
	Required   bool // the policy requires two-factor for this member
	Now        time.Time
}

// TwoFAStatus computes the badge shown in the members list.
func TwoFAStatus(in TwoFAInput) MemberTwoFA {
	switch {
	case in.TOTP:
		return MemberTwoFA{State: "on", Method: "TOTP"}
	case in.Passkeys > 0:
		return MemberTwoFA{State: "on", Method: "passkey"}
	case in.SSOOnly:
		return MemberTwoFA{State: "idp"}
	case in.Required || (in.GraceUntil != nil && in.GraceUntil.After(in.Now)):
		t := MemberTwoFA{State: "required", GraceUntil: in.GraceUntil}
		if in.GraceUntil != nil {
			d := int(in.GraceUntil.Sub(in.Now).Hours() / 24)
			if d < 0 {
				d = 0
			}
			if in.GraceUntil.Sub(in.Now) > time.Duration(d)*24*time.Hour+12*time.Hour {
				d++
			}
			t.GraceDays = &d
		}
		return t
	}
	return MemberTwoFA{State: "off"}
}

// Helpers names up to two active members who hold perm, Owners first
// ("Ask Linh Nguyen or Marcus Chen").
func Helpers(ctx context.Context, q *store.Queries, wsID uuid.UUID, perm authz.Permission) []string {
	rows, err := q.ListMembersWithRoles(ctx, wsID)
	if err != nil {
		return nil
	}
	rank := func(key string) int {
		switch key {
		case authz.RoleOwner:
			return 0
		case authz.RoleAdmin:
			return 1
		}
		return 2
	}
	slices.SortStableFunc(rows, func(a, b store.ListMembersWithRolesRow) int { return rank(a.RoleKey) - rank(b.RoleKey) })
	names := []string{}
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

// OrList joins names as "A, B or C".
func OrList(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}

// AndList joins names as "A, B and C".
func AndList(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// CantGrant explains an escalation: "You can't grant segments:sql because you
// don't hold it. Ask Linh Nguyen."
func CantGrant(p authz.Permission, helpers []string) string {
	msg := fmt.Sprintf("You can't grant %s because you don't hold it.", p)
	if len(helpers) > 0 {
		msg += " Ask " + OrList(helpers) + "."
	}
	return msg
}

// Plural returns "member" or "members".
func Plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
