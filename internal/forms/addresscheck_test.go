package forms

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// fakeChecker answers like a verification service by the address.
type fakeChecker struct {
	keys   []string
	signed []uuid.UUID
}

func (f *fakeChecker) CheckSignup(_ context.Context, _ *store.Workspace, formKey, email string) SignupAddress {
	f.keys = append(f.keys, formKey)
	switch email {
	case "nobody@example.com":
		return SignupAddress{Invalid: true}
	case "temp@mailinator.com":
		return SignupAddress{Disposable: true, Answer: "disposable"}
	}
	return SignupAddress{Answer: "valid"} // valid, or no answer within 3 s
}

func (f *fakeChecker) SignedUp(_ context.Context, _ *store.Workspace, id uuid.UUID, _ SignupAddress) {
	f.signed = append(f.signed, id)
}

func TestCheckAddressAtSignup(t *testing.T) {
	ctx := context.Background()
	ws := &store.Workspace{ID: uuid.New()}
	f := &fakeChecker{}
	s := &Service{Addresses: f}
	// The hosted page's form (built in, so nothing is recorded per form).
	page := Config{Builtin: true}

	if _, out := s.checkAddress(ctx, ws, page, Submission{Email: "nobody@example.com"}); out == nil || out.Status != OutcomeInvalid ||
		out.Errors[FieldEmail] != "This address doesn't seem to exist. Check it and try again." {
		t.Fatalf("an invalid address = %+v, want the form to ask to fix it", out)
	}
	// Disposable follows the form's setting.
	if _, out := s.checkAddress(ctx, ws, page, Submission{Email: "temp@mailinator.com"}); out != nil {
		t.Fatalf("a disposable address was refused with the setting off: %+v", out)
	}
	page.Spam.BlockDisposable = true
	if _, out := s.checkAddress(ctx, ws, page, Submission{Email: "temp@mailinator.com"}); out == nil || out.Errors[FieldEmail] != msgDisposable {
		t.Fatalf("a disposable address with the setting on = %+v", out)
	}
	// Valid, risky, unknown and no answer go through.
	if a, out := s.checkAddress(ctx, ws, page, Submission{Email: "sam@example.org"}); out != nil || a.Answer != "valid" {
		t.Fatalf("a valid address = %+v, %+v", a, out)
	}
	if f.keys[0] != "subscribe_page" {
		t.Fatalf("the hosted page is checked as %q", f.keys[0])
	}
	form := Config{ID: uuid.New()}
	_, _ = s.checkAddress(ctx, ws, form, Submission{Email: "sam@example.org"})
	if f.keys[len(f.keys)-1] != form.ID.String() {
		t.Fatalf("a form is checked as %q, want its id", f.keys[len(f.keys)-1])
	}
	// Without a checker, forms keep ReputeCast's own checks only.
	if _, out := (&Service{}).checkAddress(ctx, ws, page, Submission{Email: "nobody@example.com"}); out != nil {
		t.Fatalf("without a checker = %+v", out)
	}
}
