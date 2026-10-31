package audience

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// A contact gets at most one confirmation email every OptInGap and
// OptInPerDay a day, however many times their address is submitted or
// re-sent (review 2026-11 DP-3): a form can't be used to flood someone's
// inbox, even from many IP addresses.
const (
	OptInGap    = 10 * time.Minute
	OptInPerDay = 5
)

// OptInCooldown is a confirmation email held back by the limit; Until is
// when the next may go.
type OptInCooldown struct{ Until time.Time }

func (e *OptInCooldown) Error() string {
	return "audience: a confirmation email went to this contact moments ago"
}

// ErrOptInCooldown is what errors.Is matches for an OptInCooldown.
var ErrOptInCooldown = errors.New("audience: confirmation email held back")

// Is makes errors.Is(err, ErrOptInCooldown) true.
func (e *OptInCooldown) Is(target error) bool { return target == ErrOptInCooldown }

// ClaimOptInSend reserves a confirmation email to the contact, or returns an
// *OptInCooldown when one went too recently. Every path that emails a
// confirmation link claims first and releases the claim if the email
// doesn't go (ReleaseOptInSend).
func (s *Service) ClaimOptInSend(ctx context.Context, contactID uuid.UUID) (time.Time, error) {
	now := s.now().UTC().Truncate(time.Microsecond)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('optin:' || $1::text, 0))`, contactID); err != nil {
			return err
		}
		var n int
		var last *time.Time
		if err := tx.QueryRow(ctx, `SELECT count(*), max(sent_at) FROM optin_sends WHERE contact_id = $1 AND sent_at > $2`, contactID, now.Add(-24*time.Hour)).Scan(&n, &last); err != nil {
			return err
		}
		if last != nil && now.Sub(*last) < OptInGap {
			return &OptInCooldown{Until: last.Add(OptInGap)}
		}
		if n >= OptInPerDay {
			var first time.Time
			if err := tx.QueryRow(ctx, `SELECT min(sent_at) FROM optin_sends WHERE contact_id = $1 AND sent_at > $2`, contactID, now.Add(-24*time.Hour)).Scan(&first); err != nil {
				return err
			}
			return &OptInCooldown{Until: first.Add(24 * time.Hour)}
		}
		_, err := tx.Exec(ctx, `INSERT INTO optin_sends (contact_id, sent_at) VALUES ($1, $2) ON CONFLICT DO NOTHING`, contactID, now)
		return err
	})
	return now, err
}

// ReleaseOptInSend gives back a claim whose email didn't go.
func (s *Service) ReleaseOptInSend(ctx context.Context, contactID uuid.UUID, at time.Time) {
	_, _ = s.Pool.Exec(context.WithoutCancel(ctx), `DELETE FROM optin_sends WHERE contact_id = $1 AND sent_at = $2`, contactID, at)
}

// SendConfirmationFromLink emails a new confirmation link to a contact who
// opened an expired one and isn't waiting on a form signup (added by the
// team, an import's re-permission). It goes through the same limit; the
// email is recorded on the consent history.
func (s *Service) SendConfirmationFromLink(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) error {
	c, err := s.Q.GetContact(ctx, store.GetContactParams{WorkspaceID: ws.ID, ID: contactID})
	if err != nil {
		return err
	}
	sent, _, err := s.sendConfirmation(ctx, ws, c)
	if err != nil || !sent {
		return err
	}
	note := kept(mNoteConfirmSent, nil)
	return s.Q.InsertConsentEvent(ctx, store.InsertConsentEventParams{ID: uuid.Must(uuid.NewV7()), ContactID: c.ID, Action: "optin_sent", Source: SourceConfirmLink,
		Note: note.English, Texts: KeptTexts(map[string]i18n.Text{"note": note.Text}), OccurredAt: s.now()})
}
