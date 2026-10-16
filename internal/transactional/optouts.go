package transactional

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// Opt-outs of Optional categories. Recipients turn a category off with its
// unsubscribe link (or their mail app's one-click unsubscribe, RFC 8058)
// and turn categories on and off in the preference center's "Notifications
// from …" section. Recipients don't have to be contacts: opt-outs are kept
// per address. Essential categories can't be turned off.

// Where an opt-out came from.
const (
	OptOutPreferenceCenter = "preference_center"
	OptOutUnsubscribeLink  = "unsubscribe_link"
	OptOutOneClick         = "one_click"
)

// OptOutReasons are the answers to "Why did you turn it off?".
var OptOutReasons = []struct{ Value, Label string }{
	{"too_often", "Too often"},
	{"not_useful", "Not useful to me"},
	{"in_app", "I read this in the app"},
	{"other", "Other"},
}

// NotificationMessage is the transactional message a public link came
// with: its recipient and category.
type NotificationMessage struct {
	Email        string
	CategoryID   *uuid.UUID
	CategoryName string
	// Optional is true when the category can be turned off.
	Optional bool
	// Product is how recipients know the sender ("Acme Cloud").
	Product string
}

// NotificationCategory is one Optional category for an address.
type NotificationCategory struct {
	ID          uuid.UUID
	Name        string
	Description string
	On          bool
	// OptedOutAt and Source say when and how it was turned off.
	OptedOutAt *time.Time
	Source     string
}

// NotificationSet is the preference center's "Notifications from …"
// section for an address.
type NotificationSet struct {
	Product    string
	Categories []NotificationCategory
}

// NotificationMessage looks up the transactional message of a link; ok is
// false for any other mail.
func (s *Service) NotificationMessage(ctx context.Context, ws *store.Workspace, messageID uuid.UUID) (NotificationMessage, bool, error) {
	if messageID == uuid.Nil {
		return NotificationMessage{}, false, nil
	}
	m, err := s.Q.TxnMessageCategory(ctx, store.TxnMessageCategoryParams{ID: messageID, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return NotificationMessage{}, false, nil
	}
	if err != nil {
		return NotificationMessage{}, false, err
	}
	out := NotificationMessage{Email: m.Email, CategoryID: m.CategoryID, CategoryName: m.CategoryName, Product: s.product(ctx, ws)}
	if m.CategoryID != nil {
		// The category as it is now: it may have been renamed, or deleted.
		c, err := s.Q.TxnCategory(ctx, store.TxnCategoryParams{WorkspaceID: ws.ID, ID: *m.CategoryID})
		switch {
		case err == nil:
			out.CategoryName, out.Optional = c.Name, c.Kind == KindOptional
		case errors.Is(err, pgx.ErrNoRows):
			out.CategoryID = nil
		default:
			return NotificationMessage{}, false, err
		}
	}
	return out, true, nil
}

// Notifications lists the workspace's Optional categories and whether an
// address gets each.
func (s *Service) Notifications(ctx context.Context, ws *store.Workspace, email string) (NotificationSet, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	out := NotificationSet{Categories: []NotificationCategory{}}
	cats, err := s.Q.TxnCategories(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	off, err := s.Q.TxnOptOutsFor(ctx, store.TxnOptOutsForParams{WorkspaceID: ws.ID, Email: email})
	if err != nil {
		return out, err
	}
	for _, c := range cats {
		if c.Kind != KindOptional {
			continue
		}
		nc := NotificationCategory{ID: c.ID, Name: c.Name, Description: c.Description, On: true}
		for _, o := range off {
			if o.CategoryID == c.ID {
				at := o.OptedOutAt
				nc.On, nc.OptedOutAt, nc.Source = false, &at, o.Source
			}
		}
		out.Categories = append(out.Categories, nc)
	}
	if len(out.Categories) > 0 {
		out.Product = s.product(ctx, ws)
	}
	return out, nil
}

// SetNotifications saves the preference center's section: of the
// categories the page showed, those in on are on and the rest off.
func (s *Service) SetNotifications(ctx context.Context, tx pgx.Tx, ws *store.Workspace, email string, on, shown []uuid.UUID, source string, messageID *uuid.UUID) error {
	email = strings.ToLower(strings.TrimSpace(email))
	q := store.New(tx)
	cats, err := q.TxnCategories(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, c := range cats {
		if c.Kind != KindOptional || !slices.Contains(shown, c.ID) {
			continue
		}
		if slices.Contains(on, c.ID) {
			if _, err := q.TxnOptOutDelete(ctx, store.TxnOptOutDeleteParams{WorkspaceID: ws.ID, Email: email, CategoryID: c.ID}); err != nil {
				return err
			}
			continue
		}
		if _, err := q.TxnOptOutInsert(ctx, store.TxnOptOutInsertParams{WorkspaceID: ws.ID, Email: email, CategoryID: c.ID, OptedOutAt: s.now(), Source: source, MessageID: messageID}); err != nil {
			return err
		}
	}
	return nil
}

// TurnOff turns one Optional category off for an address (its unsubscribe
// link, one-click). Turning off an Essential category does nothing.
func (s *Service) TurnOff(ctx context.Context, tx pgx.Tx, ws *store.Workspace, email string, category uuid.UUID, source string, messageID uuid.UUID) error {
	q := store.New(tx)
	c, err := q.TxnCategory(ctx, store.TxnCategoryParams{WorkspaceID: ws.ID, ID: category})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil || c.Kind != KindOptional {
		return err
	}
	var msg *uuid.UUID
	if messageID != uuid.Nil {
		msg = &messageID
	}
	_, err = q.TxnOptOutInsert(ctx, store.TxnOptOutInsertParams{WorkspaceID: ws.ID, Email: strings.ToLower(strings.TrimSpace(email)), CategoryID: category,
		OptedOutAt: s.now(), Source: source, MessageID: msg})
	return err
}

// TurnOn turns a category back on for an address.
func (s *Service) TurnOn(ctx context.Context, tx pgx.Tx, ws *store.Workspace, email string, category uuid.UUID) error {
	_, err := store.New(tx).TxnOptOutDelete(ctx, store.TxnOptOutDeleteParams{WorkspaceID: ws.ID, Email: strings.ToLower(strings.TrimSpace(email)), CategoryID: category})
	return err
}

// OptOutReason keeps the answer to "Why did you turn it off?".
func (s *Service) OptOutReason(ctx context.Context, ws *store.Workspace, email string, category uuid.UUID, reason string) error {
	known := false
	for _, r := range OptOutReasons {
		known = known || r.Value == reason
	}
	if !known {
		return nil
	}
	_, err := s.Q.TxnOptOutReason(ctx, store.TxnOptOutReasonParams{WorkspaceID: ws.ID, Email: strings.ToLower(strings.TrimSpace(email)), CategoryID: category, Reason: reason})
	return err
}
