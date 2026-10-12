package demo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

type demoUser struct {
	email, name, color string
	password           bool
	admin, breakGlass  bool
	totp               bool
	deactivated        *time.Duration
	lastActive         time.Duration
	graceDays          int
}

// The cast of the add-on spec (A5 §4).
var demoUsers = []demoUser{
	{email: "linh@acme.io", name: "Linh Nguyen", color: "info", password: true, admin: true, breakGlass: true, totp: true, lastActive: 0},
	{email: "marcus@acme.io", name: "Marcus Chen", color: "ok", lastActive: 12 * time.Minute},
	{email: "sofia@acme.io", name: "Sofia Rossi", color: "purple", lastActive: 2 * time.Hour},
	{email: "daniel@acme.io", name: "Daniel Park", color: "warn", lastActive: 25 * time.Minute},
	{email: "aisha@acme.io", name: "Aisha Bello", color: "neutral", password: true, lastActive: 26 * time.Hour, graceDays: 3},
	{email: "tom@acme.io", name: "Tom Becker", color: "neutral", password: true, deactivated: ptr(13 * 24 * time.Hour), lastActive: 13 * 24 * time.Hour},
}

func ptr[T any](v T) *T { return &v }

// DemoRecoveryCodes are Linh Nguyen's recovery codes in the demo.
var DemoRecoveryCodes = []string{
	"b4xk-9q2m-7hdn", "m2pq-x8c4-1zrt", "d9ws-3kfh-6yvb", "p7nj-5tgc-2qxe", "r3hd-8vmb-4wsk",
	"f6cz-2nqd-9jpa", "w8tk-7xsv-3ebr", "h5gm-1dyc-8kqw", "n9rb-6pfx-4tzs", "c3vj-5hwq-2mnd",
}

// seedCore creates the instance settings, the people and both workspaces.
func seedCore(ctx context.Context, s *Seeder) error {
	if _, err := s.Tx.Exec(ctx, `UPDATE instance_settings SET name = 'Acme ReputeCast', public_url = $1, default_timezone = 'America/New_York',
		setup_completed_at = now() - interval '30 days', twofa_enforcement = 'sensitive', twofa_grace_days = 7,
		session_lifetime = '14 days', session_idle = '12 hours', lockout_attempts = 10, lockout_minutes = 15,
		jit_domain_allowlist = ARRAY['acme.io'],
		social_providers = '{"google":{"enabled":true,"client_id":"demo.apps.googleusercontent.com"},"microsoft":{"enabled":true,"client_id":"demo"},"github":{"enabled":true,"client_id":"demo","org":"acme-eng"}}',
		latest_release = '{"tag":"v1.5.0","published_at":"2026-10-07T00:00:00Z","url":"https://github.com/hostrepute/reputecast/releases/tag/v1.5.0"}',
		latest_release_at = now()`, s.Cfg.HTTP.PublicURL); err != nil {
		return err
	}
	hash, err := authn.HashPassword(Password)
	if err != nil {
		return err
	}
	for _, du := range demoUsers {
		var ph *string
		var setAt *time.Time
		if du.password {
			ph = &hash
			t := s.Ago(200 * 24 * time.Hour)
			setAt = &t
		}
		var grace *time.Time
		if du.graceDays > 0 {
			g := s.Now.Add(time.Duration(du.graceDays) * 24 * time.Hour)
			grace = &g
		}
		u, err := s.Q.CreateUser(ctx, store.CreateUserParams{
			ID: ID(), Email: du.email, Name: du.name, Timezone: "America/New_York", PasswordHash: ph, PasswordSetAt: setAt,
			IsInstanceAdmin: du.admin, BreakGlass: du.breakGlass, AvatarColor: du.color, TwofaGraceUntil: grace,
		})
		if err != nil {
			return err
		}
		if du.totp {
			enc, err := s.Box.SealString(TOTPSecret, "user:"+u.ID.String()+":totp")
			if err != nil {
				return err
			}
			t := s.Ago(180 * 24 * time.Hour)
			if err := s.Q.SetUserTOTP(ctx, store.SetUserTOTPParams{ID: u.ID, TotpSecretEnc: enc, TotpEnabledAt: &t}); err != nil {
				return err
			}
			// The codes shown in the design, so the recovery screens can be tried.
			for _, c := range DemoRecoveryCodes {
				if err := s.Q.CreateRecoveryCode(ctx, store.CreateRecoveryCodeParams{ID: ID(), UserID: u.ID, CodeHash: authn.HashToken(authn.NormalizeRecoveryCode(c))}); err != nil {
					return err
				}
			}
		}
		la := s.Ago(du.lastActive)
		if _, err := s.Tx.Exec(ctx, `UPDATE users SET last_active_at = $2, last_sign_in_at = $2, created_at = $3 WHERE id = $1`, u.ID, la, s.Ago(240*24*time.Hour)); err != nil {
			return err
		}
		if du.deactivated != nil {
			if _, err := s.Tx.Exec(ctx, `UPDATE users SET status = 'deactivated', deactivated_at = $2 WHERE id = $1`, u.ID, s.Ago(*du.deactivated)); err != nil {
				return err
			}
		}
		u, _ = s.Q.GetUser(ctx, u.ID)
		s.Users[du.email] = u
	}
	linh := s.Users["linh@acme.io"]
	for _, w := range []struct{ name, tz string }{{"Acme Newsletter", "America/New_York"}, {"Acme Engineering Blog", "America/New_York"}} {
		ws, err := workspaces.Create(ctx, s.Tx, workspaces.CreateParams{Name: w.name, Timezone: w.tz, Owner: linh.ID, Source: "install"})
		if err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET created_at = $2 WHERE id = $1`, ws.ID, s.Ago(240*24*time.Hour)); err != nil {
			return err
		}
		s.Workspaces[ws.Slug] = ws
		roles, err := s.Q.ListRoles(ctx, ws.ID)
		if err != nil {
			return err
		}
		for _, r := range roles {
			s.Roles[ws.Slug+"/"+r.Key] = store.Role{ID: r.ID, WorkspaceID: r.WorkspaceID, Key: r.Key, Name: r.Name, Builtin: r.Builtin}
		}
	}
	// Linh has TOTP, so her grace period (set by workspaces.Create) is irrelevant; clear it.
	_, err = s.Tx.Exec(ctx, `UPDATE users SET twofa_grace_until = NULL WHERE totp_enabled_at IS NOT NULL`)
	return err
}

// seedTeam adds the members, the Content Writer custom role and the pending invite.
func seedTeam(ctx context.Context, s *Seeder) error {
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

// Seeders filled in by each domain. They are declared here so the order in
// Seed never changes when a domain adds its data.
var (
	seedAudience       = noop
	seedSenders        = noop
	seedProviders      = noop
	seedCampaigns      = noop
	seedReports        = noop
	seedAutomations    = noop
	seedForms          = noop
	seedDeliverability = noop
	seedApprovals      = noop
	seedIntegrations   = noop
	seedNotifications  = noop
	seedAudit          = noop
)

func noop(context.Context, *Seeder) error { return nil }
