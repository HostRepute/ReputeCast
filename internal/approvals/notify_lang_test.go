package approvals_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Approvals' notifications keep their messages, so each member reads them in
// their own language; API clients without Accept-Language read the English
// they always did.
func TestNotificationsReadInTheReadersLanguage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Sending = &fakeStarter{}
	ws := "/workspaces/acme-newsletter"
	marcus, daniel := env.Client(), env.Client()
	marcus.SignInDemo("marcus@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	nov := campaignID(t, daniel, "November Product Update")
	r := getReview(t, daniel, nov)
	if r.Approval.Request == nil {
		t.Fatal("no approval request for November")
	}
	reqID := r.Approval.Request.ID

	// Daniel reminds the approvers; Marcus reads Vietnamese.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE approval_requests SET reminded_at = NULL WHERE id = $1`, reqID); err != nil {
		t.Fatal(err)
	}
	daniel.Do("POST", ws+"/approvals/"+reqID+"/remind", nil).MustStatus(t, 204)

	type item struct{ Kind, Title, Summary, Body string }
	type list struct{ Items []item }
	find := func(c *apptest.Client, kind string, header http.Header) item {
		t.Helper()
		c.Header = header
		defer func() { c.Header = nil }()
		var l list
		c.Do("GET", ws+"/notifications", nil).MustStatus(t, 200).JSON(t, &l)
		for _, it := range l.Items {
			if it.Kind == kind {
				return it
			}
		}
		t.Fatalf("no %s notification: %+v", kind, l.Items)
		return item{}
	}
	vi := http.Header{"Accept-Language": {"vi"}}
	got := find(marcus, "approval_requested", vi)
	if got.Title != "Có yêu cầu phê duyệt · lời nhắc" {
		t.Errorf("reminder title in Vietnamese: %q", got.Title)
	}
	if !strings.Contains(got.Body, "Daniel Park") || !strings.Contains(got.Body, "“November Product Update”") || !strings.Contains(got.Body, "người nhận") {
		t.Errorf("reminder body in Vietnamese: %q", got.Body)
	}
	got = find(marcus, "approval_requested", nil)
	if got.Title != "Approval requested · reminder" || !strings.HasPrefix(got.Body, "Daniel Park is still waiting for approval of “November Product Update”") {
		t.Errorf("reminder in English: %+v", got)
	}

	// Marcus approves; Daniel reads the decision in Vietnamese, with the comment as written.
	marcus.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve", "comment": "swap the bit.ly link"}).MustStatus(t, 200)
	got = find(daniel, "approval_decided", vi)
	if got.Title != "Đã được phê duyệt" || !strings.HasPrefix(got.Summary, "Marcus Chen đã phê duyệt “November Product Update”") ||
		!strings.Contains(got.Body, ": “swap the bit.ly link”") {
		t.Errorf("decision in Vietnamese: %+v", got)
	}
	got = find(daniel, "approval_decided", nil)
	if got.Title != "Approval given" || !strings.HasPrefix(got.Summary, "Marcus Chen approved “November Product Update”") {
		t.Errorf("decision in English: %+v", got)
	}
}
