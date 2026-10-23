// Package access holds the presentation side of authorization shared by the
// team, roles, API key and MCP screens: the permission catalogue as the API
// returns it, scope labels ("Beta Program list only"), sign-in method and
// two-factor labels, and the "who can help" helper names.
package access

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// What these screens read from the server follows the reader's language:
// the permission catalogue's words come from internal/authz, and scopes and
// why a permission can't be granted from messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "access")

var (
	mScopeList    = msgs.M("scope.list")
	mScopeOnly    = msgs.M("scope.only")
	mScopeLists   = msgs.M("scope.lists")
	mScopeParts   = msgs.M("scope.parts")
	mScopeAll     = msgs.M("scope.all")
	mCantGrant    = msgs.M("cantGrant")
	mCantGrantAsk = msgs.M("cantGrantAsk")
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
	// PeopleOnly: API keys and MCP agents can't hold it.
	PeopleOnly bool `json:"peopleOnly"`
	// NoAgents: MCP agents can't hold it.
	NoAgents bool `json:"noAgents"`
}

// Catalog returns every permission in display order, its group, label and
// description in a language.
func Catalog(l i18n.Lang) []PermissionInfo {
	out := make([]PermissionInfo, 0, len(authz.Defs))
	for _, d := range authz.Defs {
		imp := make([]string, 0, len(d.Implies))
		for _, p := range d.Implies {
			imp = append(imp, string(p))
		}
		out = append(out, PermissionInfo{Key: string(d.Key), Group: authz.GroupIn(l, d.Group), Label: authz.LabelIn(l, d.Key), Description: authz.DescriptionIn(l, d.Key),
			Sensitive: d.Sensitive, Implies: imp, OwnerOnly: d.OwnerOnly, PeopleOnly: d.PeopleOnly, NoAgents: d.NoAgents})
	}
	return out
}

// Groups returns the permission groups in display order, named in a
// language as Catalog names them.
func Groups(l i18n.Lang) []string {
	out := make([]string, len(authz.Groups))
	for i, g := range authz.Groups {
		out[i] = authz.GroupIn(l, g)
	}
	return out
}

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
	SenderDomains []string     `json:"senderDomains"`
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

// View describes a scope with list names, its label in English (as audit
// entries keep it). listWord adds " list" after a single list name ("Beta
// Program list only", used for members) or not ("Weekly Digest only", used
// for keys).
func View(s authz.Scope, names map[uuid.UUID]string, listWord bool) ScopeSummary {
	return ViewIn(i18n.Default, s, names, listWord)
}

// ViewIn is View with the label in a language: "chỉ danh sách Beta Program".
func ViewIn(l i18n.Lang, s authz.Scope, names map[uuid.UUID]string, listWord bool) ScopeSummary {
	v := ScopeSummary{Lists: []ScopedList{}, SenderDomains: []string{}, All: s.IsZero()}
	for _, id := range s.Lists {
		v.Lists = append(v.Lists, ScopedList{ID: id.String(), Name: names[id]})
	}
	v.SenderDomains = append(v.SenderDomains, s.SenderDomains...)
	switch {
	case v.All:
	case len(v.Lists) == 1 && len(v.SenderDomains) == 0:
		if listWord {
			v.Label = mScopeList.Text(l, i18n.Args{"list": v.Lists[0].Name})
		} else {
			v.Label = mScopeOnly.Text(l, i18n.Args{"name": v.Lists[0].Name})
		}
	case len(v.Lists) == 0 && len(v.SenderDomains) == 1:
		v.Label = mScopeOnly.Text(l, i18n.Args{"name": v.SenderDomains[0]})
	default:
		v.Label = mScopeParts.Text(l, i18n.Args{"parts": scopeParts(l, v)})
	}
	return v
}

// scopeParts are a scope's lists and sender domains: "Beta Program and
// Weekly Digest · news.acme.io", "3 lists".
func scopeParts(l i18n.Lang, v ScopeSummary) string {
	var parts []string
	if n := len(v.Lists); n > 0 {
		if n <= 2 {
			names := make([]string, n)
			for i, x := range v.Lists {
				names[i] = x.Name
			}
			parts = append(parts, i18n.AndList(l.Locale, names))
		} else {
			parts = append(parts, mScopeLists.Text(l, i18n.Args{"count": n}))
		}
	}
	parts = append(parts, v.SenderDomains...)
	return strings.Join(parts, " · ")
}

// ScopeText keeps a scope's label for later readers (an audit entry's
// summary): "Beta Program list only", or "all lists" when it places no
// restriction.
func ScopeText(s authz.Scope, names map[uuid.UUID]string, listWord bool) i18n.Text {
	v := ViewIn(i18n.Default, s, names, listWord)
	switch {
	case v.All:
		return mScopeAll.Keep(nil)
	case len(v.Lists) == 1 && len(v.SenderDomains) == 0 && listWord:
		return mScopeList.Keep(i18n.Args{"list": v.Lists[0].Name})
	case len(v.Lists) == 1 && len(v.SenderDomains) == 0:
		return mScopeOnly.Keep(i18n.Args{"name": v.Lists[0].Name})
	case len(v.Lists) == 0 && len(v.SenderDomains) == 1:
		return mScopeOnly.Keep(i18n.Args{"name": v.SenderDomains[0]})
	}
	// Several lists: their count, or names joined in English.
	return mScopeParts.Keep(i18n.Args{"parts": scopeParts(i18n.Default, v)})
}

// ScopeKind describes a scope for a kept message's select: "all", "list"
// (one list: value is its name), "only" (one sender domain: the domain) or
// "other" (value is the lists and domains written in English).
func ScopeKind(s authz.Scope, names map[uuid.UUID]string) (kind, value string) {
	v := ViewIn(i18n.Default, s, names, false)
	switch {
	case v.All:
		return "all", ""
	case len(v.Lists) == 1 && len(v.SenderDomains) == 0:
		return "list", v.Lists[0].Name
	case len(v.Lists) == 0 && len(v.SenderDomains) == 1:
		return "only", v.SenderDomains[0]
	}
	return "other", v.Label
}

// ScopeWhat says what a scope covers, without "only", for a table's Scope
// column: "Beta Program", "2 lists · news.acme.io", or "all lists".
func ScopeWhat(l i18n.Lang, s authz.Scope, names map[uuid.UUID]string) string {
	if s.IsZero() {
		return mScopeAll.Text(l, nil)
	}
	return scopeParts(l, ViewIn(l, s, names, false))
}

// ScopeRequest is the request shape of a scope.
type ScopeRequest struct {
	Lists         []string `json:"lists,omitempty" maxItems:"100" doc:"List ids"`
	SenderDomains []string `json:"senderDomains,omitempty" maxItems:"20"`
}

// ScopeError is a requested list or sender domain the workspace doesn't have.
type ScopeError struct {
	// Domain is true for a sender domain, false for a list.
	Domain bool
	Value  string
}

func (e *ScopeError) Error() string {
	if e.Domain {
		return fmt.Sprintf("unknown sender domain %q", e.Value)
	}
	return fmt.Sprintf("unknown list %q", e.Value)
}

// ScopeProblem is the field problem's sentence for a scope that doesn't
// resolve, in a language: "Unknown list "…"."
func ScopeProblem(l i18n.Lang, err error) string {
	var se *ScopeError
	if errors.As(err, &se) {
		key := "problem.unknownList"
		if se.Domain {
			key = "problem.unknownDomain"
		}
		return msgs.M(key).Text(l, i18n.Args{"value": se.Value})
	}
	msg := err.Error()
	return strings.ToUpper(msg[:1]) + msg[1:] + "."
}

// Resolve validates a requested scope against the workspace's lists and
// sender domains. A list or domain it doesn't know is a *ScopeError.
func (in *ScopeRequest) Resolve(lists map[uuid.UUID]string, domains []string) (authz.Scope, error) {
	var s authz.Scope
	if in == nil {
		return s, nil
	}
	seen := map[uuid.UUID]bool{}
	for _, raw := range in.Lists {
		id, err := uuid.Parse(raw)
		if err != nil {
			return s, &ScopeError{Value: raw}
		}
		if _, ok := lists[id]; !ok {
			return s, &ScopeError{Value: raw}
		}
		if !seen[id] {
			seen[id] = true
			s.Lists = append(s.Lists, id)
		}
	}
	for _, d := range in.SenderDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if !slices.Contains(domains, d) {
			return s, &ScopeError{Domain: true, Value: d}
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

// CantGrant explains an escalation in a language: "You can't grant
// segments:sql because you don't hold it. Ask Linh Nguyen."
func CantGrant(l i18n.Lang, p authz.Permission, helpers []string) string {
	if len(helpers) > 0 {
		return mCantGrantAsk.Text(l, i18n.Args{"permission": string(p), "helpers": i18n.OrList(l.Locale, helpers)})
	}
	return mCantGrant.Text(l, i18n.Args{"permission": string(p)})
}

// NotHoldable explains in a language why a key or agent can't hold p
// (authz.HoldableBy).
func NotHoldable(l i18n.Lang, p authz.Permission, t authz.ActorType) string {
	key := "notHoldable.agents"
	switch d, _ := authz.Lookup(p); {
	case p == authz.AIUse:
		key = "notHoldable.ai"
	case d.PeopleOnly:
		key = "notHoldable.peopleOnly"
	// Address verification and placement tests (X14) say why agents can't.
	case p == authz.SubscribersVerify:
		key = "notHoldable.verify"
	case p == authz.PlacementRun:
		key = "notHoldable.placementRun"
	case p == authz.PlacementManage:
		key = "notHoldable.placementManage"
	}
	return msgs.M(key).Text(l, i18n.Args{"permission": string(p)})
}

// Plural returns "member" or "members".
func Plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
