package app_test

import (
	"context"
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Gifts and conversions export as CSV like paid members and orders: the
// filter's rows, behind subscribers:export, and audited (review 2026-11
// EA-2).
func TestEarnExports(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	ctx := context.Background()
	for _, x := range []struct{ path, header, action string }{
		{"/workspaces/acme-newsletter/earn/gifts/export?status=all", "bought_at,buyer,buyer_email", "gift.exported"},
		{"/workspaces/acme-newsletter/conversions/export", "received_at,occurred_at,name", "conversion.exported"},
		{"/workspaces/acme-newsletter/earn/payments/events/export?range=90d", "received_at,occurred_at,type", "payment_events.exported"},
	} {
		r := linh.Do("GET", x.path, nil).MustStatus(t, http.StatusOK)
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/csv") || !strings.Contains(r.Header.Get("Content-Disposition"), ".csv") {
			t.Fatalf("%s headers: %v", x.path, r.Header)
		}
		rows, err := csv.NewReader(strings.NewReader(string(r.Body))).ReadAll()
		if err != nil || len(rows) < 2 || !strings.HasPrefix(strings.Join(rows[0], ","), x.header) {
			t.Fatalf("%s: %v %d rows, header %v", x.path, err, len(rows), rows)
		}
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = $1 AND (metadata->>'rows')::int = $2`, x.action, len(rows)-1).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s audited %d times: %v", x.action, n, err)
		}
	}
	// A Viewer can't export addresses.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("GET", "/workspaces/acme-newsletter/earn/gifts/export", nil).MustStatus(t, http.StatusForbidden)
	aisha.Do("GET", "/workspaces/acme-newsletter/conversions/export", nil).MustStatus(t, http.StatusForbidden)
}
