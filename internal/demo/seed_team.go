package demo

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedTeam = seedTeamImpl }

// seedTeamImpl adds the members of A5 §4, the Content Writer custom role,
// the pending invite for jo@contractor.dev (scoped to the Beta Program
// list once the lists exist) and, when no other seeder did, the Google
// Workspace sign-ins and sessions the Team screens show.
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
	if _, err := s.Tx.Exec(ctx, `UPDATE roles SET created_at = $2, updated_at = $2 WHERE id = $1`, cw.ID, s.designDay(11, 10, 12)); err != nil {
		return err
	}
	s.Roles[ws.Slug+"/content-writer"] = cw
	// Avatar colours as drawn on the team boards.
	for email, color := range map[string]string{"marcus@acme.io": "pink", "sofia@acme.io": "ok", "aisha@acme.io": "purple", "tom@acme.io": "muted"} {
		if _, err := s.Tx.Exec(ctx, `UPDATE users SET avatar_color = $2 WHERE email = $1`, email, color); err != nil {
			return err
		}
	}
	members := []struct {
		email, role, source string
		joined              time.Time
	}{
		{"marcus@acme.io", "admin", "sso_group", s.Ago(200 * 24 * time.Hour)},
		{"sofia@acme.io", "editor", "sso_jit", s.designDay(11, 14, 5)},
		{"daniel@acme.io", "content-writer", "sso_jit", s.designDay(11, 9, 30)},
		{"aisha@acme.io", "viewer", "invite", s.designDay(4, 10, 0)},
		{"tom@acme.io", "admin", "invite", s.Ago(180 * 24 * time.Hour)},
	}
	for _, m := range members {
		u := s.Users[m.email]
		role := s.Roles[ws.Slug+"/"+m.role]
		mem, err := s.Q.CreateMember(ctx, store.CreateMemberParams{ID: ID(), WorkspaceID: ws.ID, UserID: u.ID, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: m.source, InvitedBy: &linh.ID})
		if err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE workspace_members SET joined_at = $2, last_active_at = $3 WHERE id = $1`, mem.ID, m.joined, u.LastActiveAt); err != nil {
			return err
		}
		if m.email == "tom@acme.io" {
			// Deactivated Sep 30, 2026 in the prototype: 12 days before "today".
			if _, err := s.Tx.Exec(ctx, `UPDATE workspace_members SET status = 'deactivated', deactivated_at = $2 WHERE id = $1`, mem.ID, s.designDay(12, 16, 0)); err != nil {
				return err
			}
			if _, err := s.Tx.Exec(ctx, `UPDATE users SET deactivated_at = $2, last_active_at = $2 WHERE id = $1`, u.ID, s.designDay(12, 16, 0)); err != nil {
				return err
			}
		}
	}
	// Second workspace (InstanceUsers, ShellSwitcher, AccountSecurity): Kenji
	// Watanabe owns it, Linh is an Admin and Marcus an Editor there; Sofia is
	// only in Acme Newsletter. Kenji becomes Owner before Linh steps down, so
	// the workspace never lacks an Owner.
	eng := s.Workspaces["acme-engineering-blog"]
	kenji, err := ensureKenji(ctx, s)
	if err != nil {
		return err
	}
	km, err := s.Q.CreateMember(ctx, store.CreateMemberParams{ID: ID(), WorkspaceID: eng.ID, UserID: kenji.ID, RoleID: s.Roles[eng.Slug+"/owner"].ID, Scope: json.RawMessage(`{}`), Source: "install", InvitedBy: &linh.ID})
	if err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE workspace_members SET joined_at = $2, last_active_at = $3 WHERE id = $1`, km.ID, s.Ago(230*24*time.Hour), kenji.LastActiveAt); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE workspace_members SET role_id = $3 WHERE workspace_id = $1 AND user_id = $2`, eng.ID, linh.ID, s.Roles[eng.Slug+"/admin"].ID); err != nil {
		return err
	}
	if _, err := s.Q.CreateMember(ctx, store.CreateMemberParams{ID: ID(), WorkspaceID: eng.ID, UserID: s.Users["marcus@acme.io"].ID, RoleID: s.Roles[eng.Slug+"/editor"].ID, Scope: json.RawMessage(`{}`), Source: "sso_group", InvitedBy: &kenji.ID}); err != nil {
		return err
	}
	// The pending invite needs the Beta Program list, seeded later.
	s.After(seedTeamLate)
	return nil
}

// ensureKenji returns Kenji Watanabe, the Owner of Acme Engineering Blog
// (InstanceUsers: Google SSO · passkey, two-factor on through the passkey,
// signed in an hour ago), creating him unless another seeder did.
func ensureKenji(ctx context.Context, s *Seeder) (store.User, error) {
	const email = "kenji@acme.io"
	if u, ok := s.Users[email]; ok {
		return u, nil
	}
	u, err := s.Q.GetUserByEmail(ctx, email)
	if err != nil {
		if u, err = s.Q.CreateUser(ctx, store.CreateUserParams{ID: ID(), Email: email, Name: "Kenji Watanabe", Timezone: "America/New_York", AvatarColor: "sky"}); err != nil {
			return u, err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE users SET last_active_at = $2, last_sign_in_at = $2, created_at = $3 WHERE id = $1`, u.ID, s.Ago(time.Hour), s.Ago(230*24*time.Hour)); err != nil {
			return u, err
		}
		if u, err = s.Q.GetUser(ctx, u.ID); err != nil {
			return u, err
		}
	}
	s.Users[email] = u
	return u, nil
}

// seedTeamLate runs after every seeder: the invite scoped to Beta Program,
// and the SSO identities and sessions if the account and instance seeders
// didn't create them.
func seedTeamLate(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	editor := s.Roles[ws.Slug+"/editor"]
	beta, ok := s.Lists["Beta Program"]
	scope := json.RawMessage(`{}`)
	if ok {
		b, _ := json.Marshal(authz.Scope{Lists: []uuid.UUID{beta}})
		scope = b
	}
	var exists bool
	if err := s.Tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM invitations WHERE workspace_id = $1 AND email = 'jo@contractor.dev')`, ws.ID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		token, hash := authn.NewToken()
		id := ID()
		enc, err := s.Box.SealString(token, "invitation:"+id.String()+":token")
		if err != nil {
			return err
		}
		sent := s.designDay(2, 11, 41)
		via := "Postmark"
		if _, err := s.Q.TeamCreateInvitation(ctx, store.TeamCreateInvitationParams{
			ID: id, WorkspaceID: ws.ID, Email: "jo@contractor.dev", RoleID: editor.ID, Scope: scope,
			Message:   "Hi Jo, you'll run the Beta Program sends with us this quarter. Shout if anything looks off.",
			TokenHash: hash, TokenEnc: enc, InvitedBy: &linh.ID, SentVia: &via, LastSentAt: &sent,
			ExpiresAt: s.Now.Add(5*24*time.Hour + time.Hour), ExpiresDays: 7,
		}); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE invitations SET created_at = $2 WHERE id = $1`, id, sent); err != nil {
			return err
		}
	}
	return seedSignIns(ctx, s)
}

// seedDefaultSender makes sure the senders of the prototype exist (the
// senders seeder normally adds them) and picks the digest sender as the
// workspace default.
func seedDefaultSender(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	domain, ok := s.Domains["news.acme.io"]
	if !ok {
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM sending_domains WHERE workspace_id = $1 AND domain = 'news.acme.io'`, ws.ID).Scan(&domain); err != nil {
			domain = ID()
			if _, err := s.Tx.Exec(ctx, `INSERT INTO sending_domains (id, workspace_id, domain, status, auth_summary, last_checked_at)
				VALUES ($1, $2, 'news.acme.io', 'verified', '{"spf":"pass","dkim":"pass","dmarc":"quarantine","bimi":"missing","return_path":"aligned"}', $3)`, domain, ws.ID, s.Ago(4*time.Minute)); err != nil {
				return err
			}
		}
		s.Domains["news.acme.io"] = domain
	}
	verification := map[string]string{}
	for _, k := range []string{"ses", "postal"} {
		if id, ok := s.Providers[k]; ok {
			verification[id.String()] = "verified"
		}
	}
	ver, _ := json.Marshal(verification)
	for _, sn := range []struct{ name, email string }{
		{"Acme Newsletter", "digest@news.acme.io"}, {"Acme Product", "product@news.acme.io"}, {"Acme Beta", "beta@news.acme.io"},
	} {
		if _, ok := s.Senders[sn.email]; ok {
			continue
		}
		var id uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM senders WHERE workspace_id = $1 AND email = $2`, ws.ID, sn.email).Scan(&id); err != nil {
			id = ID()
			if _, err := s.Tx.Exec(ctx, `INSERT INTO senders (id, workspace_id, domain_id, name, email, verification) VALUES ($1,$2,$3,$4,$5,$6)`,
				id, ws.ID, domain, sn.name, sn.email, ver); err != nil {
				return err
			}
		}
		s.Senders[sn.email] = id
	}
	_, err := s.Tx.Exec(ctx, `UPDATE workspaces SET default_sender_id = coalesce(default_sender_id, $2) WHERE id = $1`, ws.ID, s.Senders["digest@news.acme.io"])
	return err
}

// seedSignIns links Marcus, Sofia and Daniel to Google Workspace SSO (they
// have no password) and gives Sofia and Daniel two active sessions each.
// Rows that another seeder already created are left alone.
func seedSignIns(ctx context.Context, s *Seeder) error {
	// The Google Workspace provider comes from the account and instance
	// seeders (internal/sso owns SSO); without one there's nothing to link.
	var provider uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM sso_providers WHERE preset = 'google' ORDER BY created_at LIMIT 1`).Scan(&provider); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	for _, email := range []string{"marcus@acme.io", "sofia@acme.io", "daniel@acme.io", "kenji@acme.io"} {
		u, ok := s.Users[email]
		if !ok || provider == uuid.Nil {
			continue
		}
		var n int
		if err := s.Tx.QueryRow(ctx, `SELECT count(*) FROM user_identities WHERE user_id = $1`, u.ID).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO user_identities (id, user_id, kind, sso_provider_id, subject, email, created_at, last_used_at)
				VALUES ($1, $2, 'sso', $3, $4, $5, $6, $7)`, ID(), u.ID, provider, "google-"+email, email, s.Ago(200*24*time.Hour), u.LastActiveAt); err != nil {
				return err
			}
		}
	}
	// Linh signs in with a password and a passkey (A5 §4); Kenji with Google
	// Workspace and a passkey, which is also his second factor (InstanceUsers).
	for _, pk := range []struct {
		email, name string
		used        time.Duration
	}{{"linh@acme.io", "MacBook Pro · Touch ID", 2 * time.Hour}, {"kenji@acme.io", "MacBook Air · Touch ID", time.Hour}} {
		u, ok := s.Users[pk.email]
		if !ok {
			continue
		}
		var passkeys int
		if err := s.Tx.QueryRow(ctx, `SELECT count(*) FROM user_passkeys WHERE user_id = $1`, u.ID).Scan(&passkeys); err != nil {
			return err
		}
		if passkeys == 0 {
			cred := make([]byte, 32)
			_, _ = rand.Read(cred)
			if _, err := s.Tx.Exec(ctx, `INSERT INTO user_passkeys (id, user_id, name, credential_id, public_key, attestation_type, created_at, last_used_at)
				VALUES ($1, $2, $3, $4, $5, 'none', $6, $7)`, ID(), u.ID, pk.name, cred, []byte("demo-public-key"), s.Ago(180*24*time.Hour), s.Ago(pk.used)); err != nil {
				return err
			}
		}
	}
	if err := seedDefaultSender(ctx, s); err != nil {
		return err
	}
	for _, email := range []string{"sofia@acme.io", "daniel@acme.io"} {
		u := s.Users[email]
		var n int
		if err := s.Tx.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE user_id = $1`, u.ID).Scan(&n); err != nil {
			return err
		}
		for i := n; i < 2; i++ {
			_, hash := authn.NewToken()
			ua := []string{
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 26_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
				"Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1",
			}[i]
			if _, err := s.Tx.Exec(ctx, `INSERT INTO sessions (id, user_id, token_hash, auth_method, mfa_verified, user_agent, created_at, last_seen_at, expires_at)
				VALUES ($1, $2, $3, 'sso', true, $4, $5, $6, $7)`, ID(), u.ID, hash, ua, s.Ago(time.Duration(3+i*2)*24*time.Hour), u.LastActiveAt, s.Now.Add(11*24*time.Hour)); err != nil {
				return err
			}
		}
	}
	return nil
}
