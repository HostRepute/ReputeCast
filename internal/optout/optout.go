// Package optout applies an unsubscribe that came from an email ReputeCast
// sent: a one-click POST, the unsubscribe page, or an email to the mailto:
// address of List-Unsubscribe (M58). The contact leaves every list and
// marketing to the address stops; the message's campaign counts the
// unsubscribe, the contact's timeline shows it, and its automation step
// counts it.
package optout

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/store"
)

// Deps are what an unsubscribe changes beyond the contact.
type Deps struct {
	Audience *audience.Service
	// MessageUnsubscribed, optional, is told when the unsubscribe came from
	// a message (an automation email counts it on its step).
	MessageUnsubscribed func(ctx context.Context, q *store.Queries, messageID uuid.UUID, at time.Time) error
}

// MessageRef names the email the unsubscribe came from: the message (Nil
// when unknown) and the contact. CampaignID is used when the message isn't
// known (the message's own campaign wins).
type MessageRef struct {
	MessageID  uuid.UUID
	ContactID  uuid.UUID
	CampaignID *uuid.UUID
}

// Source labels of the delivery event an unsubscribe records, by evidence
// source: what the event log and the contact's timeline show.
var sourceLabels = map[string]string{
	audience.SourceOneClick: "List-Unsubscribe one-click",
	audience.SourceMailto:   "List-Unsubscribe mailto",
	audience.SourceReply:    "Replies inbox",
}

// Apply unsubscribes the contact inside tx and records what the email it
// came from needs. reason and comment, when given, are the feedback the
// unsubscribe page asked for.
func Apply(ctx context.Context, tx pgx.Tx, d Deps, ws *store.Workspace, ref MessageRef, ev audience.PublicEvidence, reason, comment string) (audience.UnsubscribeResult, error) {
	q := store.New(tx)
	var msg *store.PublicMessageRow
	campaign := ref.CampaignID
	if ref.MessageID != uuid.Nil {
		m, err := q.PublicMessage(ctx, ref.MessageID)
		switch {
		case err == nil && m.WorkspaceID == ws.ID:
			msg, campaign = &m, m.CampaignID
		case err != nil && !errors.Is(err, pgx.ErrNoRows):
			return audience.UnsubscribeResult{}, err
		}
	}
	res, err := d.Audience.Unsubscribe(ctx, tx, ws, ref.ContactID, campaign, ev)
	if err != nil || res.Already {
		return res, err
	}
	at := ev.At
	if at.IsZero() {
		at = time.Now()
	}
	if msg != nil {
		// The message moves to unsubscribed once; its campaign's count
		// follows (internal/counters).
		if err := counters.Unsubscribed(ctx, q, msg.ID, at); err != nil {
			return res, err
		}
		source, ok := sourceLabels[ev.Source]
		if !ok {
			source = "Unsubscribe page"
		}
		if err := q.PublicInsertUnsubscribeEvent(ctx, store.PublicInsertUnsubscribeEventParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID,
			MessageID: &msg.ID, CampaignID: msg.CampaignID, ContactID: &ref.ContactID, ProviderID: msg.ProviderID, At: at, MailboxProvider: msg.MailboxProvider,
			OriginalType: source, UserAgent: clip(ev.UserAgent, 500), Ip: ev.IP, Recipient: msg.Email, Source: source}); err != nil {
			return res, err
		}
		if d.MessageUnsubscribed != nil {
			if err := d.MessageUnsubscribed(ctx, q, msg.ID, at); err != nil {
				return res, err
			}
		}
	}
	if reason != "" {
		return res, q.PublicInsertFeedback(ctx, store.PublicInsertFeedbackParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ContactID: &ref.ContactID,
			CampaignID: campaign, Reason: reason, Comment: comment, At: at})
	}
	return res, nil
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
