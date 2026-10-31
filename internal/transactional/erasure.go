package transactional

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is transactional mail to the contact, matched by the contact or
// by the address (a send through the API need not name a contact): the
// recipient's name, copies, template data, metadata, subject and stored
// content go; the log line stays for delivery counts. Opt-outs stay, like
// the suppression list (review 2026-11 DP-1).
func Erasure() erasure.Source {
	const theirs = `SELECT id FROM messages WHERE workspace_id = $4 AND stream = 'transactional' AND (contact_id = ANY ($1) OR lower(email) = ANY ($2))`
	return erasure.Source{
		Name: "transactional",
		Tables: []erasure.Table{
			{Name: "transactional_messages", Action: "clear", Columns: []string{"to_name", "cc", "bcc", "data", "metadata", "ip", "subject", "reject_detail"},
				Kept: map[string]string{"from_email": "the workspace's sender"}},
			{Name: "transactional_message_content", Action: "delete"},
			{Name: "transactional_rejected_calls", Action: "clear", Columns: []string{"recipient", "ip"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM transactional_message_content WHERE message_id IN (`+theirs+`)`,
				`UPDATE transactional_messages SET to_name = '', cc = '{}', bcc = '{}', data = '{}', metadata = '{}', ip = NULL, subject = '', reject_detail = '{}'
				 WHERE message_id IN (`+theirs+`)`,
				`UPDATE transactional_rejected_calls SET recipient = '', ip = NULL WHERE workspace_id = $4 AND lower(recipient) = ANY ($2)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "transactionalMessages", Needs: authz.TransactionalView, SQL: `SELECT t.template_name, t.category_name, t.subject, m.email, t.to_name, t.cc, t.bcc,
				m.status, m.queued_at, m.delivered_at, m.opened_at, m.clicked_at
				FROM transactional_messages t JOIN messages m ON m.id = t.message_id
				WHERE m.workspace_id = $2 AND m.stream = 'transactional' AND (m.contact_id = $1 OR lower(m.email) = $3) ORDER BY m.queued_at`},
		},
	}
}
