package access

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// A send stays within the caller's lists and sender domains: the domain is
// the sender's sending domain, or its address's once that domain is gone,
// and a campaign without a sender has none to check.
func TestSendScopeProblem(t *testing.T) {
	beta, weekly := uuid.New(), uuid.New()
	partners := authz.Scope{SenderDomains: []string{"partners.acme.io"}}
	both := authz.Scope{Lists: []uuid.UUID{beta}, SenderDomains: []string{"partners.acme.io"}}
	en := i18n.Default
	for _, c := range []struct {
		name          string
		scope         authz.Scope
		lists         []uuid.UUID
		segments      int
		domain, email string
		want          string
	}{
		{"no scope", authz.Scope{}, []uuid.UUID{weekly}, 1, "acme.io", "ceo@acme.io", ""},
		{"own domain", partners, []uuid.UUID{weekly}, 0, "partners.acme.io", "hi@partners.acme.io", ""},
		{"own domain in capitals", partners, nil, 0, "Partners.Acme.io", "", ""},
		{"another domain", partners, nil, 0, "acme.io", "ceo@acme.io", "this campaign sends from acme.io."},
		{"a sender whose domain is gone", partners, nil, 0, "", "ceo@acme.io", "this campaign sends from acme.io."},
		{"no sender", partners, nil, 0, "", "", ""},
		{"other lists first", both, []uuid.UUID{weekly}, 0, "acme.io", "ceo@acme.io", "send only to those lists"},
		{"a segment", both, []uuid.UUID{beta}, 1, "partners.acme.io", "", "send only to those lists"},
		{"within both", both, []uuid.UUID{beta}, 0, "partners.acme.io", "", ""},
	} {
		got := SendScopeProblem(en, c.scope, c.lists, c.segments, c.domain, c.email)
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	for _, l := range []i18n.Locale{i18n.German, i18n.Spanish, i18n.French, i18n.Vietnamese} {
		got := SendScopeProblem(i18n.For(l), partners, nil, 0, "acme.io", "")
		if got == "" || got == SendScopeProblem(en, partners, nil, 0, "acme.io", "") || strings.Count(got, "acme.io") != 2 {
			t.Errorf("%s: %q", l, got)
		}
		if missing := msgs.Missing(l); len(missing) > 0 {
			t.Errorf("%s doesn't translate %v", l, missing)
		}
	}
}
