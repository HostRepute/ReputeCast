package forms

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// AddressChecker checks a signup's address with the workspace's own
// verification service as people sign up (internal/verification), when the
// form has real-time checks on. It never blocks a signup on the service:
// without an answer in 3 seconds the signup goes through and is checked later.
type AddressChecker interface {
	CheckSignup(ctx context.Context, ws *store.Workspace, formKey, email string) SignupAddress
	// SignedUp records the answer on the contact (risky and unknown are tagged).
	SignedUp(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, a SignupAddress)
}

// SignupAddress is a service's answer for one signup.
type SignupAddress struct {
	// Invalid: the address can't receive mail, so the form asks to fix it.
	Invalid bool
	// Disposable: a disposable inbox; it follows the form's setting.
	Disposable bool
	// Answer is the checker's own record of the answer, for SignedUp.
	Answer any
}

const (
	msgAddressInvalid = "This address doesn't seem to exist. Check it and try again."
	msgDisposable     = "Use a permanent address: disposable inboxes expire before you could confirm."
)

// checkerKey names a form for the address checks: its id, or the hosted
// subscribe page.
func checkerKey(c Config) string {
	if c.Builtin {
		return "subscribe_page"
	}
	return c.ID.String()
}

// checkAddress asks the checker about a signup; a non-empty outcome
// refuses it.
func (s *Service) checkAddress(ctx context.Context, ws *store.Workspace, c Config, sub Submission) (SignupAddress, *Outcome) {
	if s.Addresses == nil {
		return SignupAddress{}, nil
	}
	a := s.Addresses.CheckSignup(ctx, ws, checkerKey(c), sub.Email)
	switch {
	case a.Invalid:
		return a, &Outcome{Status: OutcomeInvalid, Errors: map[string]string{FieldEmail: msgAddressInvalid}, Message: msgAddressInvalid}
	case a.Disposable && c.Spam.BlockDisposable:
		s.recordReject(ctx, ws, c, "disposable", sub.Origin)
		return a, &Outcome{Status: OutcomeInvalid, Errors: map[string]string{FieldEmail: msgDisposable}, Message: msgDisposable}
	}
	return a, nil
}
