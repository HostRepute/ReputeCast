package demo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedTeam = seedTeamImpl }

// seedTeam adds the members, the Content Writer custom role and the pending invite.
func seedTeamImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	cw, err := s.Q.CreateRole(ctx, store.CreateRoleParams{
		ID: ID(), WorkspaceID: ws.ID, Key: "content-writer", Name: "Content Writer",
		Description: "Edit campaigns and templates, view reports, request approval. Can't send, export contacts or touch providers",
		Builtin:     false,
		Permissions: authz.NewSet(authz.CampaignsView, authz.CampaignsEdit, authz.TemplatesView, authz.TemplatesEdit, authz.AutomationsView,
			authz.SubscribersView, authz.ReportsView, authz.DeliverabilityView, authz.ProvidersView, authz.TeamView).Strings(),
		CreatedBy: &linh.ID,
	})
	if err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE roles SET created_at = $2 WHERE id = $1`, cw.ID, s.Ago(12*24*time.Hour)); err != nil {
		return err
	}
	s.Roles[ws.Slug+"/content-writer"] = cw
	members := []struct {
		email, role, source string
		joined              time.Duration
	}{
		{"marcus@acme.io", "admin", "sso_group", 200 * 24 * time.Hour},
		{"sofia@acme.io", "editor", "sso_jit", 12 * 24 * time.Hour},
		{"daniel@acme.io", "content-writer", "sso_jit", 11 * 24 * time.Hour},
		{"aisha@acme.io", "viewer", "invite", 4 * 24 * time.Hour},
		{"tom@acme.io", "admin", "invite", 180 * 24 * time.Hour},
	}
	for _, m := range members {
		u := s.Users[m.email]
		role := s.Roles[ws.Slug+"/"+m.role]
		mem, err := s.Q.CreateMember(ctx, store.CreateMemberParams{ID: ID(), WorkspaceID: ws.ID, UserID: u.ID, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: m.source, InvitedBy: &linh.ID})
		if err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE workspace_members SET joined_at = $2, last_active_at = $3 WHERE id = $1`, mem.ID, s.Ago(m.joined), u.LastActiveAt); err != nil {
			return err
		}
		if m.email == "tom@acme.io" {
			if _, err := s.Tx.Exec(ctx, `UPDATE workspace_members SET status = 'deactivated', deactivated_at = $2 WHERE id = $1`, mem.ID, u.DeactivatedAt); err != nil {
				return err
			}
		}
	}
	// Second workspace: Linh (Owner), Marcus (Admin) and Sofia (Editor).
	eng := s.Workspaces["acme-engineering-blog"]
	for _, m := range []struct{ email, role string }{{"marcus@acme.io", "admin"}, {"sofia@acme.io", "editor"}} {
		if _, err := s.Q.CreateMember(ctx, store.CreateMemberParams{ID: ID(), WorkspaceID: eng.ID, UserID: s.Users[m.email].ID, RoleID: s.Roles[eng.Slug+"/"+m.role].ID, Scope: json.RawMessage(`{}`), Source: "sso_group", InvitedBy: &linh.ID}); err != nil {
			return err
		}
	}
	// SSO identities: Marcus, Sofia and Daniel sign in with Google Workspace.
	return nil
}
