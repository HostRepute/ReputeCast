// Package sealed knows every value stored sealed with the instance's
// secret key (secrets.Box): the table, the column and the associated data
// each was sealed with. `reputecast secrets reseal` uses it to re-encrypt
// every value with the current key, so a previous key can be dropped, and
// `reputecast doctor` to check the key opens them (review 2026-11 OP-4,
// OP-5).
//
// A new sealed column must be added here: TestEverySealedColumnIsKnown
// fails on a bytea column that is neither listed nor exempt.
package sealed

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/secrets"
)

// Column is one place sealed values live.
type Column struct {
	Table, Column string
	// Key names the columns that identify a row, as "name:type".
	Key []string
	// AD is the SQL expression, over the row, of the associated data the
	// value was sealed with; it matches the Go helper byte for byte.
	AD string
	// JSONField, for a jsonb column, is the field holding the sealed value,
	// base64-encoded (Go's encoding of []byte).
	JSONField string
	// Where narrows the rows (a jsonb column's kinds).
	Where string
	// ShortLived values expire on their own within a day; reseal still
	// covers them.
	ShortLived bool
}

// Name is "table.column" (with the JSON field, if any).
func (c Column) Name() string {
	if c.JSONField != "" {
		return c.Table + "." + c.Column + "." + c.JSONField
	}
	return c.Table + "." + c.Column
}

func ids(cols ...string) []string { return cols }

// Columns lists every sealed value, by table.
var Columns = []Column{
	{Table: "adapter_runtimes", Column: "secret_enc", Key: ids("adapter_id:uuid"), AD: `'adapter:' || adapter_id::text || ':secret'`},
	{Table: "adapter_runtimes", Column: "previous_secret_enc", Key: ids("adapter_id:uuid"), AD: `'adapter:' || adapter_id::text || ':secret'`},
	{Table: "ai_connections", Column: "key_sealed", Key: ids("workspace_id:uuid"), AD: `'ai:' || workspace_id::text || ':key'`},
	{Table: "auth_challenges", Column: "data", JSONField: "secret", Where: "kind = 'totp_setup'", Key: ids("id:uuid"), AD: `'totp-pending:' || user_id::text`, ShortLived: true},
	{Table: "auth_challenges", Column: "data", JSONField: "invite_enc", Where: "kind = 'oidc_flow'", Key: ids("id:uuid"), AD: `'sso:invite'`, ShortLived: true},
	{Table: "automation_step_secrets", Column: "secret_enc", Key: ids("automation_id:uuid", "node_id:text"), AD: `'automation:' || automation_id::text || ':step:' || node_id || ':secret'`},
	{Table: "campaign_review_links", Column: "token_sealed", Key: ids("id:uuid"), AD: `'review-link:' || id::text`},
	{Table: "form_challenges", Column: "payload_enc", Key: ids("id:uuid"), AD: `'form_challenge:' || id::text`, ShortLived: true},
	{Table: "hostrepute_connections", Column: "token_enc", Key: ids("workspace_id:uuid"), AD: `'hostrepute:' || workspace_id::text || ':token'`},
	{Table: "hostrepute_connections", Column: "refresh_enc", Key: ids("workspace_id:uuid"), AD: `'hostrepute:' || workspace_id::text || ':refresh'`},
	{Table: "hostrepute_oauth_states", Column: "verifier_enc", Key: ids("id:uuid"), AD: `'hostrepute-oauth:' || id::text`, ShortLived: true},
	{Table: "imports", Column: "credentials_enc", Key: ids("id:uuid"), AD: `'import:' || id::text || ':credentials'`},
	{Table: "instance_settings", Column: "social_secrets_enc", Key: ids("id:boolean"), AD: `'instance:social_providers:secrets'`},
	{Table: "instance_settings", Column: "audit_stream_secret_enc", Key: ids("id:boolean"), AD: `'instance:audit_stream_secret'`},
	{Table: "invitations", Column: "token_enc", Key: ids("id:uuid"), AD: `'invitation:' || id::text || ':token'`},
	{Table: "partner_links", Column: "secret", Key: ids("id:uuid"), AD: `'partner_link:' || id::text || ':secret'`},
	{Table: "partner_links", Column: "prev_secret", Key: ids("id:uuid"), AD: `'partner_link:' || id::text || ':prev_secret'`},
	{Table: "partner_requests_out", Column: "email", Key: ids("id:uuid"), AD: `'partner_request:' || id::text || ':email'`},
	{Table: "payment_connections", Column: "key_sealed", Key: ids("id:uuid"), AD: `'payment_connection:' || id::text || ':key'`},
	{Table: "payment_connections", Column: "webhook_secret_sealed", Key: ids("id:uuid"), AD: `'payment_connection:' || id::text || ':webhook_secret'`},
	{Table: "placement_oauth_states", Column: "verifier_enc", Key: ids("state_hash:bytea"), AD: `'placement:oauth:' || encode(state_hash, 'hex')`, ShortLived: true},
	{Table: "placement_seeds", Column: "credentials_enc", Key: ids("id:uuid"), AD: `'placement:seed:' || id::text || ':credentials'`},
	{Table: "providers", Column: "credentials_enc", Key: ids("id:uuid"), AD: `'provider:' || id::text || ':credentials'`},
	{Table: "providers", Column: "webhook_secret_enc", Key: ids("id:uuid"), AD: `'provider:' || id::text || ':webhook_secret'`},
	{Table: "public_page_settings", Column: "turnstile_secret_enc", Key: ids("workspace_id:uuid"), AD: `'turnstile:' || workspace_id::text || ':secret'`},
	{Table: "push_keys", Column: "private_enc", Key: ids("id:uuid"), AD: `'push_key:' || id::text || ':private'`},
	{Table: "referral_milestones", Column: "webhook_secret", Key: ids("id:uuid"), AD: `'referral_milestone:' || id::text || ':webhook_secret'`},
	{Table: "referral_reward_codes", Column: "code", Key: ids("id:uuid"), AD: `'referral_milestone:' || milestone_id::text || ':code'`},
	{Table: "referral_rewards", Column: "answer", Key: ids("id:uuid"), AD: `'referral_reward:' || id::text || ':answer'`},
	{Table: "referral_rewards", Column: "address", Key: ids("id:uuid"), AD: `'referral_reward:' || id::text || ':address'`},
	{Table: "rendering_services", Column: "credentials", Key: ids("workspace_id:uuid"), AD: `'rendering:' || workspace_id::text || ':credentials'`},
	{Table: "report_shares", Column: "token_sealed", Key: ids("id:uuid"), AD: `'report_share:' || id::text || ':token'`},
	{Table: "sms_providers", Column: "credentials_enc", Key: ids("id:uuid"), AD: `'sms_provider:' || id::text || ':credentials'`},
	{Table: "sms_providers", Column: "webhook_secret_enc", Key: ids("id:uuid"), AD: `'sms_provider:' || id::text || ':webhook_secret'`},
	{Table: "smtp_relay_acme", Column: "sealed", Key: ids("key:text"), AD: `'smtp_relay_acme:' || key`},
	{Table: "sponsor_report_shares", Column: "token_sealed", Key: ids("id:uuid"), AD: `'sponsor_report_share:' || id::text || ':token'`},
	{Table: "sso_providers", Column: "client_secret_enc", Key: ids("id:uuid"), AD: `'sso:' || id::text || ':client_secret'`},
	{Table: "stores", Column: "credentials_sealed", Key: ids("id:uuid"), AD: `'store:' || id::text || ':credentials'`},
	{Table: "stores", Column: "webhook_secret_sealed", Key: ids("id:uuid"), AD: `'store:' || id::text || ':webhook_secret'`},
	{Table: "transactional_outbox", Column: "sealed", Key: ids("message_id:uuid"), AD: `'transactional:' || message_id::text || ':payload'`, ShortLived: true},
	{Table: "transactional_template_imports", Column: "credentials_enc", Key: ids("id:uuid"), AD: `'transactional_import:' || id::text || ':credentials'`},
	{Table: "users", Column: "totp_secret_enc", Key: ids("id:uuid"), AD: `'user:' || id::text || ':totp'`},
	{Table: "verification_services", Column: "credentials_enc", Key: ids("id:uuid"), AD: `'verification:' || workspace_id::text || ':' || service || ':credentials'`},
	{Table: "webhooks", Column: "secret_enc", Key: ids("id:uuid"), AD: `'webhook:' || id::text || ':secret'`},
	{Table: "webhooks", Column: "previous_secret_enc", Key: ids("id:uuid"), AD: `'webhook:' || id::text || ':secret'`},
	{Table: "whatsapp_accounts", Column: "credentials_enc", Key: ids("id:uuid"), AD: `'whatsapp_account:' || id::text || ':credentials'`},
	{Table: "whatsapp_accounts", Column: "callback_token_enc", Key: ids("id:uuid"), AD: `'whatsapp_account:' || id::text || ':callback_token'`},
	{Table: "workspaces", Column: "audit_stream_secret_enc", Key: ids("id:uuid"), AD: `'workspace:' || id::text || ':audit_stream_secret'`},
	// Lead ads from Meta (M66).
	{Table: "lead_connections", Column: "credentials_sealed", Key: ids("id:uuid"), AD: `'lead_connection:' || id::text || ':credentials'`},
	{Table: "lead_connections", Column: "callback_token_sealed", Key: ids("id:uuid"), AD: `'lead_connection:' || id::text || ':callback_token'`},
	{Table: "lead_connections", Column: "verify_token_sealed", Key: ids("id:uuid"), AD: `'lead_connection:' || id::text || ':verify_token'`},
	{Table: "lead_pages", Column: "page_token_sealed", Key: ids("id:uuid"), AD: `'lead_page:' || id::text || ':token'`},
}

// value is the SQL expression of a column's sealed bytes.
func (c Column) value() string {
	if c.JSONField != "" {
		return fmt.Sprintf("decode(%s->>'%s', 'base64')", c.Column, c.JSONField)
	}
	return c.Column
}

// keys returns the key columns' names and types.
func (c Column) keys() (names, types []string) {
	for _, k := range c.Key {
		n, t, _ := strings.Cut(k, ":")
		names = append(names, n)
		types = append(types, t)
	}
	return names, types
}

// selectSQL reads up to lim rows of the column, with keys after the cursor
// (rk, the keys as text joined by "|"), ordered by it.
func (c Column) selectSQL(onlyOld bool) string {
	names, _ := c.keys()
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = n + "::text"
	}
	rk := "concat_ws('|', " + strings.Join(parts, ", ") + ")"
	where := []string{c.value() + " IS NOT NULL", rk + " > $1"}
	if c.Where != "" {
		where = append(where, c.Where)
	}
	if onlyOld {
		// Values whose key id (bytes 1–4) isn't the current key's.
		where = append(where, "substring("+c.value()+" from 2 for 4) <> $3")
	}
	return "SELECT " + rk + ", " + strings.Join(parts, ", ") + ", " + c.value() + ", " + c.AD +
		" FROM " + c.Table + " WHERE " + strings.Join(where, " AND ") + " ORDER BY 1 LIMIT $2"
}

// updateSQL replaces one value, only if it is still the one read.
func (c Column) updateSQL() string {
	names, types := c.keys()
	where := make([]string, 0, len(names)+1)
	for i, n := range names {
		where = append(where, fmt.Sprintf("%s = ($%d::text)::%s", n, i+3, types[i]))
	}
	if c.JSONField != "" {
		where = append(where, fmt.Sprintf("%s->>'%s' = $2", c.Column, c.JSONField))
		return fmt.Sprintf("UPDATE %s SET %s = jsonb_set(%s, '{%s}', to_jsonb($1::text)) WHERE %s",
			c.Table, c.Column, c.Column, c.JSONField, strings.Join(where, " AND "))
	}
	where = append(where, c.Column+" = $2")
	return fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s", c.Table, c.Column, strings.Join(where, " AND "))
}

// Result counts one column's values.
type Result struct {
	Column Column
	// Checked values; Resealed with the current key; Unreadable by every
	// known key (sealed with a key that is gone, or damaged).
	Checked, Resealed, Unreadable int
	// Old values still sealed with a previous key (Check only).
	Old int
	// Missing: the table or column doesn't exist in this database.
	Missing bool
}

const batch = 500

type row struct {
	rk     string
	keys   []string
	sealed []byte
	ad     string
}

func read(ctx context.Context, q pgx.Tx, c Column, after string, limit int, onlyOld bool, keyID [4]byte) ([]row, error) {
	args := []any{after, limit}
	if onlyOld {
		args = append(args, keyID[:])
	}
	rows, err := q.Query(ctx, c.selectSQL(onlyOld), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names, _ := c.keys()
	var out []row
	for rows.Next() {
		r := row{keys: make([]string, len(names))}
		dest := []any{&r.rk}
		for i := range r.keys {
			dest = append(dest, &r.keys[i])
		}
		dest = append(dest, &r.sealed, &r.ad)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// missing reports whether err says the table or column doesn't exist.
func missing(err error) bool {
	var pg interface{ SQLState() string }
	if errors.As(err, &pg) {
		return pg.SQLState() == "42P01" || pg.SQLState() == "42703"
	}
	return false
}

// Reseal re-encrypts every value not sealed with box's current key. Each
// value is replaced only if it is still the one read, so a token the app
// refreshed meanwhile is kept. Values no known key opens are counted and
// left alone. With dryRun nothing is written.
func Reseal(ctx context.Context, pool *pgxpool.Pool, box *secrets.Box, dryRun bool) ([]Result, error) {
	keyID := box.CurrentKeyID()
	out := make([]Result, 0, len(Columns))
	for _, c := range Columns {
		res := Result{Column: c}
		after := ""
		for {
			var rows []row
			err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
				var err error
				if rows, err = read(ctx, tx, c, after, batch, true, keyID); err != nil || dryRun {
					return err
				}
				for _, r := range rows {
					pt, err := box.Open(r.sealed, r.ad)
					if err != nil {
						continue
					}
					fresh, err := box.Seal(pt, r.ad)
					if err != nil {
						return err
					}
					args := []any{fresh, r.sealed}
					if c.JSONField != "" {
						args = []any{base64.StdEncoding.EncodeToString(fresh), base64.StdEncoding.EncodeToString(r.sealed)}
					}
					for _, k := range r.keys {
						args = append(args, k)
					}
					tag, err := tx.Exec(ctx, c.updateSQL(), args...)
					if err != nil {
						return err
					}
					res.Resealed += int(tag.RowsAffected())
				}
				return nil
			})
			if err != nil {
				if missing(err) {
					res.Missing = true
					break
				}
				return out, fmt.Errorf("%s: %w", c.Name(), err)
			}
			for _, r := range rows {
				res.Checked++
				if _, err := box.Open(r.sealed, r.ad); err != nil {
					res.Unreadable++
				}
			}
			if len(rows) < batch {
				break
			}
			after = rows[len(rows)-1].rk
		}
		out = append(out, res)
	}
	return out, nil
}

// Check opens up to sample values of every column with box, and counts the
// values still sealed with a previous key.
func Check(ctx context.Context, pool *pgxpool.Pool, box *secrets.Box, sample int) ([]Result, error) {
	keyID := box.CurrentKeyID()
	out := make([]Result, 0, len(Columns))
	for _, c := range Columns {
		res := Result{Column: c}
		err := pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
			rows, err := read(ctx, tx, c, "", sample, false, keyID)
			if err != nil {
				return err
			}
			for _, r := range rows {
				res.Checked++
				if _, err := box.Open(r.sealed, r.ad); err != nil {
					res.Unreadable++
				}
			}
			q := "SELECT count(*) FROM " + c.Table + " WHERE " + c.value() + " IS NOT NULL AND substring(" + c.value() + " from 2 for 4) <> $1"
			if c.Where != "" {
				q += " AND " + c.Where
			}
			return tx.QueryRow(ctx, q, keyID[:]).Scan(&res.Old)
		})
		if err != nil {
			if missing(err) {
				res.Missing = true
			} else {
				return out, fmt.Errorf("%s: %w", c.Name(), err)
			}
		}
		out = append(out, res)
	}
	return out, nil
}
