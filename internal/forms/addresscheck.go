package forms

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
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

// addressInvalid is the form's message for an address the service rejects:
// an English form's own sentence (VerificationStates); forms in other
// languages say their own "error.emailInvalid".
func addressInvalid(c Config) string {
	if code := c.Labels.Code(); code == "" || code == "en" {
		return mAddressInvalid.Text(i18n.Default, nil)
	}
	return c.Labels.Text("error.emailInvalid", nil)
}

// labels are the form's built-in labels, English when it has none yet.
func labels(c Config) pagelabels.Set {
	if c.Labels.Code() == "" {
		return pagelabels.For("en", nil)
	}
	return c.Labels
}

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
		msg := addressInvalid(c)
		return a, &Outcome{Status: OutcomeInvalid, Errors: map[string]string{FieldEmail: msg}, Message: msg}
	case a.Disposable && c.Spam.BlockDisposable:
		s.recordReject(ctx, ws, c, "disposable", sub.Origin)
		msg := labels(c).Text("error.emailDisposable", nil)
		return a, &Outcome{Status: OutcomeInvalid, Errors: map[string]string{FieldEmail: msg}, Message: msg}
	}
	return a, nil
}
