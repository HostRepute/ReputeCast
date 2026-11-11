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
			{Name: "website_members", Action: "delete", Columns: []string{"display_name"}},
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

// CommentsErasure is what a contact wrote under posts and the reply emails
// waiting for them (M61): their comments keep their place, so other
// members' replies under them stay, but lose their text and address and
// read "This comment was removed."; the notices go.
func CommentsErasure() erasure.Source {
	return erasure.Source{
		Name: "website_comments",
		Tables: []erasure.Table{
			{Name: "website_comments", Action: "anonymize", Columns: []string{"body", "ip"},
				Kept: map[string]string{"contact_id": "keeps a thread's shape for the other members' replies under a removed comment; the contact is scrubbed"}},
			{Name: "website_comment_notices", Action: "delete"},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM website_comment_notices WHERE contact_id = ANY ($1)`,
				`UPDATE website_comments SET body = '', ip = NULL, status = 'deleted', pinned_at = NULL, decision_id = NULL, undo_until = NULL,
					before_status = '', before_published_at = NULL
				WHERE contact_id = ANY ($1)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "websiteComments", SQL: `SELECT p.title AS post, c.created_at, c.body FROM website_comments c JOIN website_posts p ON p.id = c.post_id
				WHERE c.contact_id = $1 AND c.status <> 'deleted' ORDER BY c.created_at`},
		},
	}
}
