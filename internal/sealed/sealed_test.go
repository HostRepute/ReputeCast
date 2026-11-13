package sealed_test

import (
	"context"
	"crypto/rand"
	"slices"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/sealed"
	"github.com/hostrepute/reputecast/internal/secrets"
)

// bytea columns that hold no sealed value: hashes, public keys, files and
// raw webhook bodies.
var notSealed = map[string]string{
	"api_keys.token_hash": "hash", "auth_challenges.token_hash": "hash", "campaign_review_links.token_hash": "hash",
	"email_changes.token_hash": "hash", "invitations.token_hash": "hash", "password_resets.token_hash": "hash",
	"report_shares.token_hash": "hash", "sessions.token_hash": "hash", "sponsor_report_shares.token_hash": "hash",
	"subscriber_tokens.token_hash": "hash", "website_sessions.token_hash": "hash", "website_signin_links.token_hash": "hash",
	"user_recovery_codes.code_hash": "hash", "partner_codes.code_hash": "hash", "partner_links.code_hash": "hash",
	"referral_reward_codes.code_hash": "hash", "coupon_codes.code_hash": "hash", "website_signin_links.code_hash": "hash",
	"hostrepute_oauth_states.state_hash": "hash", "placement_oauth_states.state_hash": "hash (the row's key)",
	"media_assets.sha256": "hash", "email_countdowns.hash": "hash of the settings (the row's key)", "referral_files.sha256": "hash", "download_files.sha256": "hash",
	"website_signin_links.browser_hash": "hash", "website_view_seen.hash": "hash", "product_order_downloads.ip_hash": "hash",
	"referral_visit_keys.key":     "HMAC digest",
	"user_passkeys.credential_id": "public WebAuthn data", "user_passkeys.public_key": "public key", "user_passkeys.aaguid": "public WebAuthn data",
	"users.webauthn_id":         "public WebAuthn handle",
	"push_subscriptions.p256dh": "browser's public key", "push_subscriptions.auth": "browser's push secret, sent by the browser to every server it subscribes",
	"push_test_subscriptions.p256dh": "browser's public key", "push_test_subscriptions.auth": "browser's push secret",
	"imports.file_data": "uploaded file", "imports.report_csv": "report file", "imports.skipped_csv": "report file",
	"user_avatars.data": "image", "media_assets.data": "image", "referral_file_chunks.data": "file", "earn_import_files.data": "file",
	"public_page_settings.logo": "image",
	"provider_raw_events.body":  "raw webhook", "store_events.body": "raw webhook", "sms_raw_events.body": "raw webhook",
	"whatsapp_raw_events.body":  "raw webhook",
	"website_view_salts.salt":   "random salt",
	"river_job.unique_key":      "the job queue's dedupe hash",
	"webhook_tokens.token_hash": "hash", "webhook_tokens.payload_hash": "hash",
	"whatsapp_accounts.callback_token_hash": "hash",
	"inbound_raw.body":                      "received mail, as the provider posted it", "inbound_routes.hook_token_hash": "hash",
	// Importing posts (M61): the uploaded export and the run's report.
	"website_import_files.data": "uploaded file", "website_imports.report_csv": "report file",
	"earn_group_seats.invite_hash": "hash", "sponsor_bookings.status_token_hash": "hash",
	"lead_connections.callback_token_hash": "hash", "lead_events.body": "raw webhook (ids only)",
}

// Every bytea column is either a known sealed column or known not to be
// one, so `secrets reseal` never misses a value.
func TestEverySealedColumnIsKnown(t *testing.T) {
	env := apptest.New(t)
	known := map[string]bool{}
	for _, c := range sealed.Columns {
		if c.JSONField == "" {
			known[c.Table+"."+c.Column] = true
		}
	}
	rows, err := env.App.Pool.Query(context.Background(), `
		SELECT table_name || '.' || column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND data_type = 'bytea' ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var seen []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		seen = append(seen, name)
		if !known[name] && notSealed[name] == "" {
			t.Errorf("%s: add it to sealed.Columns (with its associated data) or to notSealed with why", name)
		}
	}
	for name := range known {
		if !slices.Contains(seen, name) {
			t.Errorf("sealed.Columns lists %s, which isn't a bytea column", name)
		}
	}
}

// After a key change, reseal re-encrypts every value with the new key: a
// box without the old key then opens them all.
func TestResealMovesEveryValueToTheCurrentKey(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	oldKey, err := config.DecodeKey(env.App.Cfg.Secrets.Key)
	if err != nil {
		t.Fatal(err)
	}
	newKey := make([]byte, 32)
	_, _ = rand.Read(newKey)
	rotated, err := secrets.New(newKey, oldKey)
	if err != nil {
		t.Fatal(err)
	}
	onlyNew, err := secrets.New(newKey)
	if err != nil {
		t.Fatal(err)
	}

	before, err := sealed.Check(ctx, env.App.Pool, rotated, 3)
	if err != nil {
		t.Fatal(err)
	}
	old := 0
	for _, r := range before {
		if r.Missing {
			t.Errorf("%s: no such column", r.Column.Name())
		}
		if r.Unreadable > 0 {
			t.Errorf("%s: %d of %d values don't open (wrong associated data?)", r.Column.Name(), r.Unreadable, r.Checked)
		}
		old += r.Old
	}
	if old < 20 {
		t.Fatalf("the demo seals %d values with the old key", old)
	}

	dry, err := sealed.Reseal(ctx, env.App.Pool, rotated, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range dry {
		if r.Resealed != 0 {
			t.Fatalf("a dry run wrote %s", r.Column.Name())
		}
	}
	res, err := sealed.Reseal(ctx, env.App.Pool, rotated, false)
	if err != nil {
		t.Fatal(err)
	}
	resealed := 0
	for _, r := range res {
		resealed += r.Resealed
		if r.Unreadable > 0 {
			t.Errorf("%s: %d unreadable", r.Column.Name(), r.Unreadable)
		}
	}
	if resealed != old {
		t.Fatalf("resealed %d of %d", resealed, old)
	}
	after, err := sealed.Check(ctx, env.App.Pool, onlyNew, 1000)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range after {
		if r.Old != 0 || r.Unreadable != 0 {
			t.Errorf("%s after reseal: %d old, %d of %d unreadable", r.Column.Name(), r.Old, r.Unreadable, r.Checked)
		}
	}
	// Twice is a no-op.
	again, _ := sealed.Reseal(ctx, env.App.Pool, rotated, false)
	for _, r := range again {
		if r.Resealed != 0 {
			t.Errorf("%s resealed again", r.Column.Name())
		}
	}
}
