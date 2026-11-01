package approvals_test

import (
	"context"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestInboxSearchAndStaleChecks: the inbox searches by what a request is
// about, and re-checks on load only requests that may be stale: never
// checked, their campaign changed since, or checked long ago (review
// 2026-11 AP-1).
func TestInboxSearchAndStaleChecks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var list struct {
		Items []struct{ Name, SubjectName string }
	}
	c.Do("GET", ws+"/approvals?status=all&days=0&q=november", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) == 0 {
		t.Fatal("search by campaign name finds November Product Update")
	}
	c.Do("GET", ws+"/approvals?status=all&days=0&q=%25zz", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 0 {
		t.Fatalf("LIKE's wildcards are literal: %+v", list.Items)
	}

	checked := func() (n int, at time.Time) {
		t.Helper()
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*), coalesce(max(revalidated_at), 'epoch') FROM approval_requests r JOIN campaigns c ON c.id = r.campaign_id
			WHERE r.status IN ('pending', 'approved') AND r.revalidated_at IS NOT NULL`).Scan(&n, &at); err != nil {
			t.Fatal(err)
		}
		return n, at
	}
	n, first := checked()
	if n == 0 {
		t.Fatal("opening the inbox records the checks")
	}
	c.Do("GET", ws+"/approvals", nil).MustStatus(t, 200)
	if _, again := checked(); !again.Equal(first) {
		t.Fatalf("fresh checks aren't run again: %v then %v", first, again)
	}
	// A campaign that changed since is checked again.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET updated_at = now() WHERE id = (SELECT r.campaign_id FROM approval_requests r JOIN campaigns x ON x.id = r.campaign_id
		WHERE r.status IN ('pending', 'approved') AND x.status IN ('draft', 'approval', 'approved', 'scheduled') ORDER BY r.requested_at LIMIT 1)`); err != nil {
		t.Fatal(err)
	}
	c.Do("GET", ws+"/approvals", nil).MustStatus(t, 200)
	if _, after := checked(); !after.After(first) {
		t.Fatalf("a changed campaign's request is checked again: %v %v", first, after)
	}
}
