package demo

import (
	"context"
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

func init() { seedInstance = seedInstanceImpl }

// Entra ID group object IDs of the sample (A5 §4: "Marketing" 3f2a…c91,
// "Eng Leads" 8b7d…02e).
const (
	EntraGroupMarketing = "3f2a9d1e-5b7c-4e21-9a8f-2d4c6e8b0c91"
	EntraGroupEngLeads  = "8b7d44c0-1e2f-4a3b-8c9d-6e7f8a9b002e"
	entraTenant         = "7c1a2b3d-0e4f-4a5b-9c6d-1e2f3a4b5c6d"
)

// seedInstanceImpl fills Instance admin (A5 §4, InstanceUsers, InstanceAuth,
// InstanceSettings, SSOSetup, SSOStates, AuditInstance) with instance-level
// data only: sign-in methods and the security policies, Google and GitHub
// sign-in (GitHub restricted to acme-eng), the Google Workspace provider
// the account seeder created, Microsoft Entra ID mid-setup with its two
// group mappings and a test sign-in, the public URL check, the demo's
// latest release, and the instance-level audit history. People and
// memberships belong to the team seeder.
func seedInstanceImpl(ctx context.Context, s *Seeder) error {
	linh := s.Users["linh@acme.io"]

	// Policies (core set the defaults the boards use; this adds what core
	// doesn't: the second allowlist domain, the password re-use rule, the
	// public URL check and the demo release with its changelog).
	urlCheck := map[string]any{
		"checkedAt": s.Ago(4 * time.Minute),
		"dns":       map[string]string{"status": "ok", "detail": "DNS resolves to 203.0.113.40 (this server)"},
		"tls":       map[string]string{"status": "ok", "detail": "TLS valid until Jan 8, 2027 · auto-renews"},
		"reachable": map[string]string{"status": "ok", "detail": "Reachable from the internet · 142 ms · provider webhooks arrive"},
	}
	release := map[string]any{
		"tag": "v1.5.0", "name": "ReputeCast 1.5.0", "published_at": s.designDay(4, 10, 0), "source": "demo",
		"url": "https://github.com/hostrepute/reputecast/releases/tag/v1.5.0",
		"notes": []string{
			"Send approvals: two-approver policies and approval invalidation diffs",
			"Passkeys as a second factor; recovery-code regeneration from Account & security",
			"Adapter SDK 1.2: quota discovery for Mailjet and SparkPost",
			"Fix: Postal webhook signatures with rotated keys",
		},
		"note": "No database migration downtime: the upgrade runs migrations before switching over.",
	}
	social := map[string]any{
		"google":    map[string]any{"enabled": true, "client_id": "204810937215-acme-signin.apps.googleusercontent.com", "secret_set_at": s.designDay(60, 10, 0)},
		"microsoft": map[string]any{"enabled": false, "tenant": "common"},
		"github":    map[string]any{"enabled": true, "client_id": "Iv23liAcmeReputeCast", "org": "acme-eng", "secret_set_at": s.designDay(60, 10, 5)},
	}
	secrets, _ := json.Marshal(map[string]string{"google": "demo-google-client-secret", "github": "demo-github-client-secret"})
	secretsEnc, err := s.Box.Seal(secrets, "instance:social_providers:secrets")
	if err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE instance_settings SET
		jit_domain_allowlist = ARRAY['acme.io', 'acme-labs.io'], password_reuse = 5, breached_check = true,
		allowed_methods = ARRAY['password', 'passkey', 'social', 'sso'],
		social_providers = $1, social_secrets_enc = $2,
		public_url_check = $3, public_url_checked_at = $4,
		latest_release = $5, latest_release_at = $6, update_checked_at = $6, update_check = true`,
		instanceJSON(social), secretsEnc, instanceJSON(urlCheck), s.Ago(4*time.Minute), instanceJSON(release), s.Ago(2*time.Hour)); err != nil {
		return err
	}

	// Google Workspace: the account seeder's provider, made complete.
	google, ok := s.Extra[SSOGoogleKey].(uuid.UUID)
	if !ok {
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM sso_providers WHERE preset = 'google' ORDER BY created_at LIMIT 1`).Scan(&google); err != nil {
			return err
		}
	}
	gSecret, err := s.Box.SealString("demo-google-workspace-secret", "sso:"+google.String()+":client_secret")
	if err != nil {
		return err
	}
	googleDiscovery := map[string]any{ //nolint:gosec // G101: public endpoint URLs, not credentials
		"ok": true, "checked_at": s.designDay(53, 9, 40), "url": "https://accounts.google.com/.well-known/openid-configuration",
		"status": 200, "ms": 96, "issuer": "https://accounts.google.com", "authorization_endpoint": "https://accounts.google.com/o/oauth2/v2/auth",
		"token_endpoint": "https://oauth2.googleapis.com/token", "userinfo_endpoint": "https://openidconnect.googleapis.com/v1/userinfo",
		"jwks_uri": "https://www.googleapis.com/oauth2/v3/certs", "keys": 2}
	if _, err := s.Tx.Exec(ctx, `UPDATE sso_providers SET client_secret_enc = $2, client_secret_set_at = $3, client_secret_set_by = $4,
		discovery = $5, enabled_at = $6, created_by = $4, created_at = $7, updated_at = $6 WHERE id = $1`,
		google, gSecret, s.designDay(53, 9, 38), linh.ID, instanceJSON(googleDiscovery), s.designDay(53, 10, 0), s.designDay(53, 9, 30)); err != nil {
		return err
	}

	// Microsoft Entra ID: setup in progress (draft), two group mappings, a
	// discovery minute ago and a test sign-in as Marcus.
	newsletter, eng := s.WS(), s.Workspaces["acme-engineering-blog"]
	entra := ID()
	issuer := "https://login.microsoftonline.com/" + entraTenant + "/v2.0"
	mappings := []map[string]any{
		{"group": EntraGroupMarketing, "label": "Marketing", "workspace_id": newsletter.ID, "role_key": "editor", "scope": map[string]any{}},
		{"group": EntraGroupEngLeads, "label": "Eng Leads", "workspace_id": eng.ID, "role_key": "admin", "scope": map[string]any{}},
	}
	entraDiscovery := map[string]any{"ok": true, "checked_at": s.Ago(time.Minute), "url": issuer + "/.well-known/openid-configuration", "status": 200, "ms": 184,
		"issuer": issuer, "authorization_endpoint": "https://login.microsoftonline.com/" + entraTenant + "/oauth2/v2.0/authorize",
		"token_endpoint": "https://login.microsoftonline.com/" + entraTenant + "/oauth2/v2.0/token", "userinfo_endpoint": "https://graph.microsoft.com/oidc/userinfo",
		"jwks_uri": "https://login.microsoftonline.com/" + entraTenant + "/discovery/v2.0/keys", "keys": 3}
	test := map[string]any{"at": s.Ago(time.Minute), "by": "Linh Nguyen", "claims": map[string]any{
		"iss": issuer, "sub": "AAAAAAAAAAAAAAAAAAAAAM4sQ8vD2fX0bYkHw9Lr", "aud": "a9f3c1e2-4b7d-4e8f-9a0b-1c2d3e4f5a6b",
		"email": "marcus@acme.io", "name": "Marcus Chen", "preferred_username": "marcus@acme.io",
		"groups": []string{EntraGroupMarketing, EntraGroupEngLeads}, "amr": []string{"pwd", "mfa"}, "tid": entraTenant,
	}}
	eSecret, err := s.Box.SealString("demo-entra-client-secret", "sso:"+entra.String()+":client_secret")
	if err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO sso_providers (id, preset, name, issuer_url, client_id, client_secret_enc, client_secret_set_at, client_secret_set_by,
		client_secret_expires_on, scopes, claim_map, allowed_domains, jit_enabled, jit_workspace_id, jit_role_key, group_mappings, status, discovery, last_test,
		created_by, created_at, updated_at)
		VALUES ($1, 'entra', 'Microsoft Entra ID', $2, 'a9f3c1e2-4b7d-4e8f-9a0b-1c2d3e4f5a6b', $3, $4, $5, $6, ARRAY['openid', 'profile', 'email'],
		'{"email":"email","email_verified":"email_verified","name":"name","groups":"groups"}', ARRAY['acme.io'], true, $7, 'viewer', $8, 'draft', $9, $10, $5, $11, $12)`,
		entra, issuer, eSecret, s.instanceDay(1, 14, 0), linh.ID, s.instanceDay(-180, 0, 0), newsletter.ID, instanceJSON(mappings),
		instanceJSON(entraDiscovery), instanceJSON(test), s.instanceDay(1, 13, 30), s.instanceDay(1, 14, 2)); err != nil {
		return err
	}
	s.Extra["sso:entra"] = entra

	s.After(seedInstanceLate)
	return seedInstanceAudit(ctx, s, google, entra)
}

// seedInstanceLate runs once every seeder is done: the break-glass Owners
// (Linh and Kenji), and the workspaces' creation dates from InstanceSettings.
func seedInstanceLate(ctx context.Context, s *Seeder) error {
	if _, err := s.Tx.Exec(ctx, `UPDATE users SET break_glass = true WHERE email IN ('linh@acme.io', 'kenji@acme.io')`); err != nil {
		return err
	}
	for slug, daysAgo := range map[string]int{"acme-newsletter": 224, "acme-engineering-blog": 120} {
		if ws, ok := s.Workspaces[slug]; ok {
			if _, err := s.Tx.Exec(ctx, `UPDATE workspaces SET created_at = $2 WHERE id = $1`, ws.ID, s.designDay(daysAgo, 11, 0)); err != nil {
				return err
			}
		}
	}
	return nil
}

// instanceAudit is one instance-level audit entry of the sample history.
type instanceAudit struct {
	at                   time.Time
	actor                string // email, or "" for the system
	action, targetType   string
	targetID, targetName string
	result, reason       string
	ip, ua               string
	meta                 map[string]any
	before, after        any
}

// seedInstanceAudit writes the instance-level history: sign-ins (and the
// rejected kim@gmail.com with its reference), the Entra ID draft, the
// two-factor policy change, Tom's deactivation and Linh's break-glass mark.
func seedInstanceAudit(ctx context.Context, s *Seeder, google, entra uuid.UUID) error {
	const chrome = "Mozilla/5.0 (Macintosh; Intel Mac OS X 26_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	m := func(kv ...any) map[string]any {
		out := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			out[kv[i].(string)] = kv[i+1]
		}
		return out
	}
	tom := s.Users["tom@acme.io"]
	linh := s.Users["linh@acme.io"]
	events := []instanceAudit{
		{at: s.Ago(48 * time.Second), actor: "linh@acme.io", action: "signin.succeeded", targetType: "user", targetID: linh.ID.String(), targetName: "linh@acme.io",
			ip: "203.0.113.9", ua: chrome, meta: m("method", "password+totp", "summary", "Password + TOTP · break-glass Owner", "source", "auth", "via", "203.0.113.9")},
		{at: s.Ago(29*time.Minute + 57*time.Second), action: "signin.rejected", targetType: "user", targetName: "kim@gmail.com", result: "denied",
			reason: "email domain not allowed for Google Workspace SSO", ip: "203.0.113.77",
			meta: m("method", "sso", "provider", "Google Workspace", "provider_id", google.String(), "code", "domain_not_allowed", "reference", "auth_7f3k9dqe",
				"summary", "Google Workspace SSO · email domain not allowed · reference auth_7f3k9dqe", "source", "auth", "via", "203.0.113.77")},
		{at: s.Ago(46*time.Minute + 20*time.Second), actor: "marcus@acme.io", action: "signin.succeeded", targetType: "user", targetName: "marcus@acme.io", ip: "198.51.100.40", ua: chrome,
			meta: m("method", "sso", "provider", "Google Workspace", "provider_id", google.String(), "amr", "pwd+mfa",
				"summary", "Google Workspace SSO · amr pwd+mfa · workspace Acme Newsletter", "source", "auth", "via", "198.51.100.40")},
		{at: s.instanceDay(1, 14, 2), actor: "linh@acme.io", action: "sso.provider.updated", targetType: "sso_provider", targetID: entra.String(), targetName: "Microsoft Entra ID",
			ip: "203.0.113.9", ua: chrome, meta: m("summary", "draft saved · client secret replaced · 2 group mappings", "secret_replaced", true),
			before: m("group_mappings", 0, "status", "draft"), after: m("group_mappings", 2, "status", "draft")},
		{at: s.instanceDay(1, 13, 48), actor: "linh@acme.io", action: "policy.updated", targetType: "instance_policy", targetName: "Two-factor enforcement", ip: "203.0.113.9", ua: chrome,
			meta: m("summary", "required for sensitive permissions · grace 7 days (was 14)"), before: m("two_factor", "sensitive", "grace_days", 14), after: m("two_factor", "sensitive", "grace_days", 7)},
		{at: s.instanceDay(1, 17, 20), actor: "aisha@acme.io", action: "signin.failed", targetType: "user", targetName: "aisha@acme.io", result: "denied", reason: "wrong password",
			ip: "198.51.100.77", meta: m("method", "password", "summary", "wrong password · attempt 1 of 10", "source", "auth", "via", "198.51.100.77")},
		{at: s.instanceDay(1, 17, 21), actor: "aisha@acme.io", action: "signin.failed", targetType: "user", targetName: "aisha@acme.io", result: "denied", reason: "wrong password",
			ip: "198.51.100.77", meta: m("method", "password", "summary", "wrong password · attempt 2 of 10", "source", "auth", "via", "198.51.100.77")},
		{at: s.instanceDay(1, 17, 22), actor: "aisha@acme.io", action: "signin.failed", targetType: "user", targetName: "aisha@acme.io", result: "denied", reason: "wrong password",
			ip: "198.51.100.77", meta: m("method", "password", "summary", "wrong password · attempt 3 of 10", "source", "auth", "via", "198.51.100.77")},
		{at: s.instanceDay(1, 17, 24), actor: "aisha@acme.io", action: "signin.succeeded", targetType: "user", targetName: "aisha@acme.io", ip: "198.51.100.77",
			meta: m("method", "password", "summary", "Password · after 3 failed attempts", "source", "auth", "via", "198.51.100.77")},
		{at: s.instanceDay(12, 17, 10), actor: "linh@acme.io", action: "user.deactivated", targetType: "user", targetID: tom.ID.String(), targetName: "Tom Becker · tom@acme.io",
			reason: "left the company", ip: "203.0.113.9", ua: chrome, meta: m("summary", "reason: left the company · 1 API key revoked · 2 sessions ended"),
			before: m("status", "active"), after: m("status", "deactivated")},
		{at: s.instanceDay(12, 17, 8), actor: "linh@acme.io", action: "instance_admin.granted", targetType: "user", targetID: linh.ID.String(), targetName: "Linh Nguyen",
			ip: "203.0.113.9", ua: chrome, meta: m("summary", "break-glass Owner marked for acme.io enforcement"), before: m("break_glass", false), after: m("break_glass", true)},
	}
	for _, e := range events {
		var actorID *uuid.UUID
		actorType, actorName := "system", "System"
		if e.actor != "" {
			if u, ok := s.Users[e.actor]; ok {
				id := u.ID
				actorID, actorType, actorName = &id, "user", u.Name
			}
		}
		var ip *netip.Addr
		if e.ip != "" {
			a := netip.MustParseAddr(e.ip)
			ip = &a
		}
		result := e.result
		if result == "" {
			result = "allowed"
		}
		var before, after json.RawMessage
		if e.before != nil {
			before = instanceJSON(e.before)
		}
		if e.after != nil {
			after = instanceJSON(e.after)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
			result, reason, ip, user_agent, request_id, before, after, metadata) VALUES ($1, NULL, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
			ID(), e.at, actorType, actorID, actorName, e.action, e.targetType, e.targetID, e.targetName, result, e.reason, ip, e.ua,
			"req_"+ID().String()[24:32], before, after, instanceJSON(e.meta)); err != nil {
			return err
		}
	}
	return nil
}

func instanceJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// instanceDay is daysAgo days before today at hour:minute in the demo's
// timezone (America/New_York), so yesterday's sample events read
// "yesterday" whatever the hour the demo was loaded. Today's events use
// s.Ago instead, which is never in the future.
func (s *Seeder) instanceDay(daysAgo, hour, minute int) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	n := s.Now.In(loc)
	return time.Date(n.Year(), n.Month(), n.Day()-daysAgo, hour, minute, 0, 0, loc)
}
