package access

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Scopes, catalogue rows and locked permissions read in the reader's
// language; the English stays as it was.
func TestAccessInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	beta, weekly := uuid.New(), uuid.New()
	names := map[uuid.UUID]string{beta: "Beta Program", weekly: "Weekly Digest"}
	one := authz.Scope{Lists: []uuid.UUID{beta}}
	two := authz.Scope{Lists: []uuid.UUID{beta, weekly}, SenderDomains: []string{"news.acme.io"}}

	for _, c := range []struct {
		got, want string
	}{
		{View(one, names, true).Label, "Beta Program list only"},
		{View(one, names, false).Label, "Beta Program only"},
		{View(two, names, true).Label, "Beta Program and Weekly Digest · news.acme.io only"},
		{ViewIn(vi, one, names, true).Label, "chỉ danh sách Beta Program"},
		{ViewIn(vi, two, names, true).Label, "chỉ Beta Program và Weekly Digest · news.acme.io"},
		{ScopeWhat(vi, authz.Scope{}, names), "mọi danh sách"},
		{ScopeWhat(i18n.Default, one, names), "Beta Program"},
		{ScopeText(one, names, true).In(vi), "chỉ danh sách Beta Program"},
		{CantGrant(i18n.Default, authz.SegmentsSQL, []string{"Linh Nguyen"}), "You can't grant segments:sql because you don't hold it. Ask Linh Nguyen."},
		{CantGrant(vi, authz.SegmentsSQL, []string{"Linh Nguyen"}), "Bạn không thể cấp segments:sql vì bạn không có quyền này. Hãy hỏi Linh Nguyen."},
		{NotHoldable(i18n.Default, authz.CampaignsSend, authz.ActorMCPAgent), "MCP agents can't hold campaigns:send. Give it to an API key or a person."},
		{NotHoldable(vi, authz.AIUse, authz.ActorAPIKey), "Tính năng AI dành cho con người. Tác tử dùng mô hình của riêng chúng."},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}

	_, err := (&ScopeRequest{Lists: []string{"nope"}}).Resolve(names, nil)
	var se *ScopeError
	if !errors.As(err, &se) || err.Error() != `unknown list "nope"` || ScopeProblem(i18n.Default, err) != `Unknown list "nope".` ||
		ScopeProblem(vi, err) != `Không có danh sách "nope".` {
		t.Errorf("scope problem: %v", err)
	}

	cat := Catalog(vi)
	if cat[0].Key != "campaigns:view" || cat[0].Group != "Chiến dịch" || cat[0].Label != "Xem chiến dịch" {
		t.Errorf("catalogue: %+v", cat[0])
	}
	if en := Catalog(i18n.Default); en[0].Group != "Campaigns" || en[0].Label != "View campaigns" {
		t.Errorf("English catalogue: %+v", en[0])
	}
	if g := Groups(vi); len(g) != len(authz.Groups) || g[0] != "Chiến dịch" {
		t.Errorf("groups: %v", g)
	}
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("not translated: %v", missing)
	}
}
