// Package accessrequests backs the "Request Editor access" action on the 403
// screen: it names the smallest built-in role that grants a missing
// permission and sends the request to the members who manage the team.
// Nothing changes until one of them does.
package accessrequests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// KindAccessRequested is the notification kind team managers receive.
const KindAccessRequested = "access_requested"

// Service suggests roles and records access requests.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
}

// AccessSuggestion is what the 403 screen shows before the person asks.
type AccessSuggestion struct {
	Permission string `json:"permission" doc:"The missing permission"`
	Action     string `json:"action" doc:"What the permission allows, as a sentence subject" example:"Managing providers"`
	RoleKey    string `json:"roleKey" doc:"Smallest built-in role that grants it" example:"editor"`
	RoleName   string `json:"roleName" example:"Editor"`
	YourRole   string `json:"yourRole" example:"Viewer"`
	// Owners are the workspace's Owners; Managers can change roles (team:manage).
	Owners   []string `json:"owners"`
	Managers []string `json:"managers"`
	// RequestedAt is set when you asked for this role in the last 24 hours.
	RequestedAt *time.Time `json:"requestedAt,omitempty"`
}

// AccessRequestResult says who was asked.
type AccessRequestResult struct {
	RoleName string   `json:"roleName"`
	Notified []string `json:"notified" doc:"Members who received the request"`
}

// SmallestRole returns the first built-in role, from Viewer up, that holds perm.
func SmallestRole(perm authz.Permission) authz.BuiltinRole {
	for i := len(authz.BuiltinRoles) - 1; i >= 0; i-- {
		if r := authz.BuiltinRoles[i]; r.Permissions.Has(perm) {
			return r
		}
	}
	owner, _ := authz.Builtin(authz.RoleOwner)
	return owner
}

func actionFor(perm authz.Permission) string {
	msg := authz.DeniedMessage(perm, nil) // "Managing providers needs the providers:manage permission."
	if i := strings.Index(msg, " needs the "); i > 0 {
		return msg[:i]
	}
	return string(perm)
}

func parsePermission(p string) (authz.Permission, error) {
	perm := authz.Permission(strings.TrimSpace(p))
	if !authz.Valid(perm) {
		return "", api.Invalid("permission", fmt.Sprintf("%q isn't a permission.", p))
	}
	return perm, nil
}

type people struct{ owners, managers []string }

func (s *Service) people(ctx context.Context, ws *store.Workspace) (people, error) {
	rows, err := s.Q.ListMembersWithRoles(ctx, ws.ID)
	if err != nil {
		return people{}, err
	}
	out := people{owners: []string{}, managers: []string{}}
	for _, r := range rows {
		if r.RoleKey == authz.RoleOwner {
			out.owners = append(out.owners, r.Name)
		}
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(authz.TeamManage) {
			out.managers = append(out.managers, r.Name)
		}
	}
	return out, nil
}

func (s *Service) lastRequest(ctx context.Context, q *store.Queries, p *authz.Principal, ws *store.Workspace, roleKey string) (*time.Time, error) {
	at, err := q.AccessRequestsLatest(ctx, store.AccessRequestsLatestParams{UserID: &p.UserID, WorkspaceID: &ws.ID, RoleKey: roleKey})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &at, nil
}

// Suggest describes the role that would grant perm and who can grant it.
func (s *Service) Suggest(ctx context.Context, permission string) (AccessSuggestion, error) {
	perm, err := parsePermission(permission)
	if err != nil {
		return AccessSuggestion{}, err
	}
	p, ws := api.PrincipalFrom(ctx), api.WorkspaceFrom(ctx)
	role := SmallestRole(perm)
	ppl, err := s.people(ctx, ws)
	if err != nil {
		return AccessSuggestion{}, err
	}
	last, err := s.lastRequest(ctx, s.Q, p, ws, role.Key)
	if err != nil {
		return AccessSuggestion{}, err
	}
	return AccessSuggestion{
		Permission: string(perm), Action: actionFor(perm), RoleKey: role.Key, RoleName: role.Name, YourRole: p.RoleName,
		Owners: ppl.owners, Managers: ppl.managers, RequestedAt: last,
	}, nil
}

// Request asks the team's managers for the role that grants permission.
func (s *Service) Request(ctx context.Context, permission, note string) (AccessRequestResult, error) {
	perm, err := parsePermission(permission)
	if err != nil {
		return AccessRequestResult{}, err
	}
	p, ws := api.PrincipalFrom(ctx), api.WorkspaceFrom(ctx)
	if p.Can(perm) {
		return AccessRequestResult{}, api.Conflict("You already have this permission. Reload the page to use it.")
	}
	role := SmallestRole(perm)
	ppl, err := s.people(ctx, ws)
	if err != nil {
		return AccessRequestResult{}, err
	}
	if len(ppl.managers) == 0 {
		return AccessRequestResult{}, api.Conflict("Nobody in this workspace can change roles. Ask an instance admin.")
	}
	member, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: p.UserID})
	if err != nil {
		return AccessRequestResult{}, api.MapError(err, "Membership")
	}
	note = strings.TrimSpace(note)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		last, err := s.lastRequest(ctx, q, p, ws, role.Key)
		if err != nil {
			return err
		}
		if last != nil {
			return api.Conflict(fmt.Sprintf("You already asked for %s access in the last day; %s will see it. Ask them directly if it's urgent.", role.Name, joinNames(ppl.managers))).
				WithMeta("requestedAt", last.UTC().Format(time.RFC3339))
		}
		body := note
		if body == "" {
			body = "No note."
		}
		if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.TeamManage, notify.Notification{
			Kind:     KindAccessRequested,
			Severity: "info",
			Title:    fmt.Sprintf("%s asks for %s access", p.Name, role.Name),
			Body:     body,
			Link:     fmt.Sprintf("/w/%s/settings/team/%s", ws.Slug, member.ID),
			Data:     map[string]any{"permission": string(perm), "roleKey": role.Key, "userId": p.UserID.String()},
		}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &ws.ID,
			Action:      "access.request",
			TargetType:  "role",
			TargetID:    role.Key,
			TargetName:  role.Name,
			Metadata:    map[string]any{"permission": string(perm), "note": note, "currentRole": p.RoleName},
		})
	})
	if err != nil {
		return AccessRequestResult{}, err
	}
	return AccessRequestResult{RoleName: role.Name, Notified: ppl.managers}, nil
}

func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return "the team's admins"
	case 1:
		return names[0]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
}

// statusAccepted is returned by Request (the request waits for a manager).
const statusAccepted = http.StatusAccepted
