package demo

import (
	"context"
	"crypto/rand"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedAccount = seedAccountImpl }

// SSOGoogleKey is the s.Extra key of the Google Workspace SSO provider, so
// the instance seeder can extend it instead of creating a second one.
const SSOGoogleKey = "sso:google"

// seedAccountImpl adds what Account & security shows (A5 §4, AccountSecurity
// board): Linh's passkeys, authenticator, recovery codes, sessions on other
// devices, GitHub and Google Workspace identities and notification
// preferences; and the Google Workspace SSO identities of Marcus, Sofia and
// Daniel, with the enforcing Google Workspace provider they sign in with.
func seedAccountImpl(ctx context.Context, s *Seeder) error {
	linh := s.Users["linh@acme.io"]
	day := 24 * time.Hour

	// Google Workspace SSO: enabled for acme.io with JIT into Acme Newsletter
	// as Viewer. Domain enforcement stays off in the demo: Aisha signs in
	// with a password, which enforcement would turn off (the add-on sample
	// data asks for both). Turn it on in Instance admin › SSO to see the
	// "Continue with Google Workspace" sign-in step and the break-glass link.
	google := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO sso_providers (id, preset, name, issuer_url, client_id, scopes, claim_map, allowed_domains,
		jit_enabled, jit_workspace_id, jit_role_key, enforce_domains, status, discovery, created_at, updated_at)
		VALUES ($1, 'google', 'Google Workspace', 'https://accounts.google.com', '204810937215-acme.apps.googleusercontent.com',
		ARRAY['openid','email','profile'], '{"email":"email","email_verified":"email_verified","name":"name"}', ARRAY['acme.io'],
		true, $2, 'viewer', '{}', 'enabled', '{"ok":true}', $3, $3)`,
		google, s.WS().ID, s.Ago(200*day)); err != nil {
		return err
	}
	s.Extra[SSOGoogleKey] = google

	// Linh: password changed Aug 3 (70 days before the design's today), the
	// authenticator set up the same day in 1Password, recovery codes saved.
	if _, err := s.Tx.Exec(ctx, `UPDATE users SET password_set_at = $2, totp_enabled_at = $2, totp_label = '1Password',
		recovery_codes_saved_at = $2, webauthn_id = $3 WHERE id = $1`, linh.ID, s.Ago(70*day), randomBytes(32)); err != nil {
		return err
	}
	// 7 of 10 recovery codes remain; the last one was used Oct 2. The first
	// code (printed by `reputecast seed`) stays unused.
	for _, used := range []struct {
		code string
		at   time.Duration
	}{{DemoRecoveryCodes[9], 10 * day}, {DemoRecoveryCodes[8], 41 * day}, {DemoRecoveryCodes[7], 63 * day}} {
		if _, err := s.Tx.Exec(ctx, `UPDATE user_recovery_codes SET used_at = $3 WHERE user_id = $1 AND code_hash = $2`,
			linh.ID, authn.HashToken(authn.NormalizeRecoveryCode(used.code)), s.Ago(used.at)); err != nil {
			return err
		}
	}

	// Passkeys. The key material is placeholder data: these three show the
	// list, real passkeys are added from Account & security.
	iCloud := uuid.MustParse("fbfc3007-154e-4ecc-8c0b-6e020557d7bd")
	for _, pk := range []struct {
		name, attachment, provider string
		aaguid                     []byte
		created                    time.Duration
		used                       *time.Duration
	}{
		{"MacBook Pro · Touch ID", "platform", "iCloud Keychain", iCloud[:], 70 * day, ptr(2 * time.Minute)},
		{"iPhone 16", "platform", "", nil, 70 * day, ptr(3 * day)},
		{"YubiKey 5C", "cross-platform", "", nil, 30 * day, nil},
	} {
		var used *time.Time
		if pk.used != nil {
			t := s.Ago(*pk.used)
			used = &t
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO user_passkeys (id, user_id, name, credential_id, public_key, attestation_type, aaguid, sign_count,
			transports, backup_eligible, backup_state, attachment, provider, created_at, last_used_at)
			VALUES ($1, $2, $3, $4, $5, 'none', $6, 0, $7, $8, $8, $9, $10, $11, $12)`,
			ID(), linh.ID, pk.name, randomBytes(32), randomBytes(77), pk.aaguid, transportsFor(pk.attachment), pk.attachment == "platform",
			pk.attachment, pk.provider, s.Ago(pk.created), used); err != nil {
			return err
		}
	}

	// Sessions on Linh's other devices. Their tokens are random and never
	// handed out; the session you sign in with appears as "This session".
	for _, ss := range []struct {
		device, os, browser, ip, location string
		created, lastSeen                 time.Duration
	}{
		{"iPhone 16", "iOS 26", "Safari", "198.51.100.23", "New York, US", 3 * day, 2 * time.Hour},
		{"Windows PC", "Windows 11", "Edge 141", "192.0.2.77", "Newark, US", 5 * day, 9 * time.Hour},
	} {
		_, hash := authn.NewToken()
		ip := netip.MustParseAddr(ss.ip)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sessions (id, user_id, token_hash, auth_method, mfa_verified, ip, user_agent, location,
			created_at, last_seen_at, expires_at, device, os, browser)
			VALUES ($1, $2, $3, 'password+totp', true, $4, '', $5, $6, $7, $8, $9, $10, $11)`,
			ID(), linh.ID, hash, ip, ss.location, s.Ago(ss.created), s.Ago(ss.lastSeen), s.Ago(ss.created).Add(14*day), ss.device, ss.os, ss.browser); err != nil {
			return err
		}
	}

	// Connected sign-ins: GitHub (org acme-eng) and the Google Workspace
	// identity for Linh; Google Workspace for the SSO members.
	if _, err := s.Q.CreateIdentity(ctx, store.CreateIdentityParams{ID: ID(), UserID: linh.ID, Kind: "github", Subject: "8812734", Email: "linh@acme.io"}); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE user_identities SET username = 'linh-nguyen', details = '{"org":"acme-eng"}', created_at = $2, last_used_at = $3
		WHERE user_id = $1 AND kind = 'github'`, linh.ID, s.Ago(150*day), s.Ago(20*day)); err != nil {
		return err
	}
	for _, email := range []string{"linh@acme.io", "marcus@acme.io", "sofia@acme.io", "daniel@acme.io"} {
		u := s.Users[email]
		gid := google
		if _, err := s.Q.CreateIdentity(ctx, store.CreateIdentityParams{ID: ID(), UserID: u.ID, Kind: "sso", SsoProviderID: &gid,
			Subject: "1047" + u.ID.String()[24:], Email: email}); err != nil {
			return err
		}
	}

	// Linh's notification preferences, as on the Notifications board.
	prefs := map[string][2]bool{
		"approval_requested": {true, true}, "approval_decided": {true, true}, "approval_invalidated": {true, true},
		"send_paused": {true, true}, "provider_down": {true, true}, "webhooks_silent": {true, false},
		"invite_accepted": {true, false}, "hostrepute_alert": {true, true},
	}
	for ev, p := range prefs {
		if err := s.Q.UpsertNotificationPreference(ctx, store.UpsertNotificationPreferenceParams{UserID: linh.ID, EventType: ev, InApp: p[0], Email: p[1]}); err != nil {
			return err
		}
	}
	start, end := int16(22*60), int16(7*60)
	return s.Q.UpsertNotificationSettings(ctx, store.UpsertNotificationSettingsParams{UserID: linh.ID, QuietStart: &start, QuietEnd: &end, Digest: "immediate"})
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func transportsFor(attachment string) []string {
	if attachment == "cross-platform" {
		return []string{"usb", "nfc"}
	}
	return []string{"internal", "hybrid"}
}
