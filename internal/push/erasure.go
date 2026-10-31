package push

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure ends the contact's push subscriptions and drops their endpoints,
// keys and where they subscribed; the rows stay, removed, for counts
// (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "push",
		Tables: []erasure.Table{
			{Name: "push_subscriptions", Action: "anonymize", Columns: []string{"endpoint", "p256dh", "auth", "ip", "user_agent", "page", "time_zone"},
				Kept: map[string]string{"contact_id": "subscriber counts; the subscription is removed and the contact scrubbed"}},
			{Name: "push_messages", Action: "keep", Kept: map[string]string{"subscription_id": "delivery counts; the subscription has no endpoint left"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE push_subscriptions SET removed_at = coalesce(removed_at, now()),
				    removed_reason = CASE WHEN removed_at IS NULL THEN 'erased' ELSE removed_reason END,
				    endpoint = '', p256dh = NULL, auth = NULL, ip = NULL, user_agent = '', page = '', time_zone = ''
				 WHERE contact_id = ANY ($1)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "pushSubscriptions", SQL: `SELECT browser, platform, time_zone, page, host(ip) AS ip, user_agent, created_at, last_seen_at, removed_at, removed_reason
				FROM push_subscriptions WHERE contact_id = $1 ORDER BY created_at`},
		},
	}
}
