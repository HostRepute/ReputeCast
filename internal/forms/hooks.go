package forms

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// SignupHook is told about signups and confirmations inside their
// transactions, so another domain records what it needs with them. Grow's
// referrals record a referral when someone signs up through a referral
// link (the invite page, or a page opened with ?ref=) and start its hold
// when they confirm.
type SignupHook interface {
	// DoubleOptIn reports whether a submission must confirm by email even
	// on a single opt-in form: double opt-in is always on for referrals.
	DoubleOptIn(ctx context.Context, ws *store.Workspace, sub Submission) bool
	// Accepted runs after a signup was recorded.
	Accepted(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e SignupEvent) error
	// Confirmed runs after a contact confirmed their pending lists.
	Confirmed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e ConfirmEvent) error
}

// SignupEvent is an accepted signup.
type SignupEvent struct {
	ContactID uuid.UUID
	Email     string
	FirstName string
	// Created: the signup created the contact.
	Created bool
	// Lists the submission asked for; Pending wait for the confirmation,
	// Subscribed joined at once, Already were joined before.
	Lists      []uuid.UUID
	Pending    []uuid.UUID
	Subscribed []uuid.UUID
	Already    []uuid.UUID
	// Page and Referrer are where the signup came from (the page's address
	// keeps its query: ?ref=, ?via=).
	Page      string
	Referrer  string
	IP        *netip.Addr
	UserAgent string
	FormID    *uuid.UUID
	At        time.Time
}

// ConfirmEvent is a confirmation click that confirmed lists.
type ConfirmEvent struct {
	ContactID uuid.UUID
	Lists     []uuid.UUID
	IP        *netip.Addr
	At        time.Time
}

func (s *Service) hooks() []SignupHook {
	if s.Hooks == nil {
		return nil
	}
	return s.Hooks()
}

// forceDouble is true when a hook needs the submission confirmed by email.
func (s *Service) forceDouble(ctx context.Context, ws *store.Workspace, sub Submission) bool {
	for _, h := range s.hooks() {
		if h.DoubleOptIn(ctx, ws, sub) {
			return true
		}
	}
	return false
}

func (s *Service) signupAccepted(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e SignupEvent) error {
	for _, h := range s.hooks() {
		if err := h.Accepted(ctx, tx, ws, e); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) signupConfirmed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e ConfirmEvent) error {
	for _, h := range s.hooks() {
		if err := h.Confirmed(ctx, tx, ws, e); err != nil {
			return err
		}
	}
	return nil
}
