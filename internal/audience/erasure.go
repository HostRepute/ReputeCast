package audience

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/erasure"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// CoreErasure is the contacts table and what the audience itself keeps about
// a contact. It runs after every other source (erasure.Registry), which may
// still match rows by the address or phone number.
func CoreErasure() erasure.Source {
	return erasure.Source{
		Name: "audience",
		Tables: []erasure.Table{
			{Name: "contacts", Action: "anonymize",
				Columns: []string{"email", "first_name", "last_name", "company", "country", "timezone", "language", "tags", "attributes", "consent_evidence",
					"mailbox_detail", "phone", "phone_country", "phone_line_type", "phone_line_source", "phone_set_at"}},
			{Name: "list_memberships", Action: "delete"},
			{Name: "consent_events", Action: "delete", Columns: []string{"ip", "user_agent", "consent_text"}},
			{Name: "contact_preferences", Action: "delete"},
			{Name: "segment_members", Action: "delete"},
			{Name: "messages", Action: "clear", Columns: []string{"email"},
				Kept: map[string]string{"contact_id": "campaign statistics keep adding up; the contact is scrubbed"}},
			{Name: "delivery_events", Action: "clear", Columns: []string{"ip", "user_agent", "url", "recipient"},
				Kept: map[string]string{"contact_id": "reports count events per contact; the contact is scrubbed",
					"recipient": "kept when the address is on the suppression list, which keeps it anyway (bounce reports count suppressed recipients)"}},
			{Name: "unsubscribe_feedback", Action: "clear", Columns: []string{"comment"},
				Kept: map[string]string{"contact_id": "unsubscribe reasons are counted; the contact is scrubbed"}},
			{Name: "subscriber_tokens", Action: "delete"},
			{Name: "campaign_audience_stage", Action: "delete"},
			{Name: "automation_message_vars", Action: "delete", Columns: []string{"vars"}},
			{Name: "optin_sends", Action: "delete"},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM list_memberships WHERE contact_id = ANY ($1)`,
				`DELETE FROM consent_events WHERE contact_id = ANY ($1)`,
				`DELETE FROM contact_preferences WHERE contact_id = ANY ($1)`,
				`DELETE FROM segment_members WHERE contact_id = ANY ($1)`,
				`DELETE FROM subscriber_tokens WHERE contact_id = ANY ($1)`,
				`DELETE FROM campaign_audience_stage WHERE contact_id = ANY ($1)`,
				`DELETE FROM optin_sends WHERE contact_id = ANY ($1)`,
				`DELETE FROM automation_message_vars WHERE message_id IN (SELECT id FROM messages WHERE contact_id = ANY ($1))`,
				`UPDATE messages SET email = '' WHERE contact_id = ANY ($1)`,
				`UPDATE delivery_events d SET ip = NULL, user_agent = '', url = '',
				    recipient = CASE WHEN d.recipient = '' OR EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = d.workspace_id AND s.email = lower(d.recipient))
				                     THEN d.recipient ELSE 'erased-' || d.contact_id || '@erased.invalid' END
				 WHERE d.contact_id = ANY ($1)`,
				`UPDATE unsubscribe_feedback SET comment = '' WHERE contact_id = ANY ($1) AND comment <> ''`,
				`UPDATE contacts SET erased_at = now(), email = 'erased-' || id || '@erased.invalid', first_name = '', last_name = '', company = '',
				    country = '', timezone = '', language = '', tags = '{}', attributes = '{}', consent_evidence = '{}', mailbox_detail = '',
				    phone = '', phone_country = '', phone_line_type = 'unknown', phone_line_source = '', phone_set_at = NULL,
				    status = 'suppressed', updated_at = now()
				 WHERE id = ANY ($1)`,
			)
		},
	}
}

// eraseChunk bounds how many contacts one round of statements covers.
const eraseChunk = 5000

// erase scrubs the personal data of contacts in one workspace through every
// registered source, then queues contact.erased. Contacts a source refuses
// (a live paid subscription in Stripe) are left as they are: with strict
// (a person asked) the whole erase is refused instead, so the typed count
// stays the count erased. It returns the contacts erased and refused.
func (s *Service) erase(ctx context.Context, tx pgx.Tx, wsID uuid.UUID, ids []uuid.UUID, reason erasure.Reason, strict bool) (erased, refused []uuid.UUID, err error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	rows, err := tx.Query(ctx, `SELECT id, email, phone FROM contacts WHERE workspace_id = $1 AND id = ANY ($2) AND erased_at IS NULL ORDER BY id FOR UPDATE`, wsID, ids)
	if err != nil {
		return nil, nil, err
	}
	subjects, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (erasure.Subject, error) {
		var su erasure.Subject
		return su, r.Scan(&su.ID, &su.Email, &su.Phone)
	})
	if err != nil {
		return nil, nil, err
	}
	reg := s.registry()
	b := &erasure.Batch{WorkspaceID: wsID, Subjects: subjects, Reason: reason, At: s.now()}
	refusals, err := reg.Guard(ctx, tx, b)
	if err != nil {
		return nil, nil, err
	}
	if len(refusals) > 0 {
		out := map[uuid.UUID]bool{}
		for _, r := range refusals {
			for _, id := range r.IDs {
				out[id] = true
			}
		}
		for id := range out {
			refused = append(refused, id)
		}
		if strict {
			first := refusals[0]
			return nil, refused, api.Err(http.StatusConflict, first.Code, say(ctx, first.Message, i18n.Args{"n": len(first.IDs)})).
				WithMeta("refused", len(refused), "matched", len(subjects))
		}
		kept := subjects[:0]
		for _, su := range subjects {
			if !out[su.ID] {
				kept = append(kept, su)
			}
		}
		b.Subjects = kept
	}
	all := b.Subjects
	for start := 0; start < len(all); start += eraseChunk {
		part := &erasure.Batch{WorkspaceID: wsID, Subjects: all[start:min(start+eraseChunk, len(all))], Reason: reason, At: b.At}
		if err := reg.Erase(ctx, tx, part); err != nil {
			return nil, nil, err
		}
	}
	erased = b.IDs()
	// contact.erased goes out through the contact-event queue, only when an
	// endpoint listens and only once the erase commits; no address in it.
	if _, err := tx.Exec(ctx, `INSERT INTO webhook_contact_events (workspace_id, event, contact_id, data)
		SELECT $1, 'contact.erased', id, jsonb_build_object('erased_at', $3::timestamptz, 'reason', $4::text)
		FROM unnest($2::uuid[]) AS id WHERE webhook_listens($1, 'contact.erased')`, wsID, erased, b.At, string(reason)); err != nil {
		return nil, nil, err
	}
	return erased, refused, nil
}

// registry is the erasure registry; without one (a test's Service) only the
// core source runs.
func (s *Service) registry() *erasure.Registry {
	if s.Erasure != nil {
		return s.Erasure
	}
	return erasure.New(CoreErasure())
}

// IsPaidRefusal reports whether err is an erase refused for live paid
// subscriptions.
func IsPaidRefusal(err error) bool {
	var p *api.Problem
	return errors.As(err, &p) && p.Code == ErasePaidMembers
}

// ErasePaidMembers is the problem code of an erase refused because some
// contacts still pay through Stripe.
const ErasePaidMembers = "paid_members"
