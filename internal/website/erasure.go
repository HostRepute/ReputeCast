package website

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the contact's membership of the publication website: sign-ins,
// sessions and pending sign-in links (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "website",
		Tables: []erasure.Table{
			{Name: "website_members", Action: "delete"},
			{Name: "website_sessions", Action: "delete"},
			{Name: "website_member_events", Action: "delete"},
			{Name: "website_signin_links", Action: "delete", Columns: []string{"new_email"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM website_sessions WHERE contact_id = ANY ($1)`,
				`DELETE FROM website_signin_links WHERE contact_id = ANY ($1)`,
				`DELETE FROM website_member_events WHERE contact_id = ANY ($1)`,
				`DELETE FROM website_members WHERE contact_id = ANY ($1)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "websiteMember", SQL: `SELECT first_signed_in, last_signed_in, last_method, sign_ins FROM website_members WHERE contact_id = $1`},
			{Section: "websiteSignIns", SQL: `SELECT kind, method, same_browser, detail, occurred_at FROM website_member_events WHERE contact_id = $1 ORDER BY occurred_at`},
		},
	}
}
