package ingest_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// A provider's complaint rate divides by the mail delivered to mailbox
// providers that report complaints (review F-2): a delivery at Gmail adds
// to what was delivered, not to the rate's base.
func TestComplaintBaseLeavesOutGmail(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var providerID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT p.id FROM providers p JOIN workspaces w ON w.id = p.workspace_id
		WHERE w.slug = 'acme-newsletter' AND p.adapter = 'ses'`).Scan(&providerID); err != nil {
		t.Fatal(err)
	}
	p, err := env.App.Q.ProvGetAny(ctx, providerID)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Now().UTC()
	totals := func() store.ProvDailyTotalsRow {
		t.Helper()
		rows, err := env.App.Q.ProvDailyTotals(ctx, store.ProvDailyTotalsParams{ProviderIds: []uuid.UUID{providerID}, Since: day, Until: day})
		if err != nil || len(rows) > 1 {
			t.Fatalf("totals: %v %v", rows, err)
		}
		if len(rows) == 0 {
			// Events count on their UTC day; the demo's days are the
			// workspace's, so early in a UTC day it may have no row yet.
			return store.ProvDailyTotalsRow{}
		}
		return rows[0]
	}
	before := totals()
	n := &ingest.Normalizer{Pool: env.App.Pool, Q: env.App.Q}
	for _, to := range []string{"reader.one@gmail.com", "reader.two@outlook.com"} {
		ev := providers.Event{Type: providers.EventDelivered, Recipient: to, OriginalType: "Delivery", OccurredAt: day}
		if err := db.Tx(ctx, env.App.Pool, func(tx pgx.Tx) error {
			_, err := n.Apply(ctx, tx, p, nil, "webhook", day, ev)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	after := totals()
	if d, r := after.Delivered-before.Delivered, after.DeliveredReporting-before.DeliveredReporting; d != 2 || r != 1 {
		t.Fatalf("delivered +%d, delivered where complaints are reported +%d; want +2 and +1", d, r)
	}

	// The rate and why it is empty.
	rate, why := manage.ComplaintRate(store.ProvDailyTotalsRow{ComplainedReporting: 4, DeliveredReporting: 8895, DeliveredCounted: 20000})
	if rate == nil || *rate != 0.04 || why != "" {
		t.Fatalf("rate %v (%q)", rate, why)
	}
	if rate, why := manage.ComplaintRate(store.ProvDailyTotalsRow{Delivered: 500, DeliveredCounted: 500}); rate != nil || why != manage.RateNotReported {
		t.Fatalf("only Gmail and iCloud: %v (%q)", rate, why)
	}
	// Days counted before the base existed are unknown, not 0%.
	if rate, why := manage.ComplaintRate(store.ProvDailyTotalsRow{Delivered: 500, Complained: 3}); rate != nil || why != manage.RateNoData {
		t.Fatalf("days before the base: %v (%q)", rate, why)
	}
}
