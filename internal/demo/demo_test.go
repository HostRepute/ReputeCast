package demo

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/mailbox"
)

func TestSplitAddsUp(t *testing.T) {
	for _, n := range []int64{47980, 31520, 2098, 1, 0, 7} {
		parts := Split(n, []float64{0.46, 0.21, 0.09, 0.08, 0.16})
		var sum int64
		for _, p := range parts {
			sum += p
		}
		if sum != n {
			t.Fatalf("Split(%d) sums to %d", n, sum)
		}
	}
}

// Gmail and iCloud never report complaints (review F-2): no seeded send,
// ReputeCast's own or history imported from Brevo, holds one there.
func TestNoComplaintsWhereNotReported(t *testing.T) {
	pool, _, _ := seededSample(t)
	rows, err := pool.Query(context.Background(), `SELECT mailbox_provider, sum(complained)::bigint FROM campaign_counters GROUP BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	complained := int64(0)
	for rows.Next() {
		var mbp string
		var n int64
		if err := rows.Scan(&mbp, &n); err != nil {
			t.Fatal(err)
		}
		if n > 0 && !mailbox.ReportsComplaints(mbp) {
			t.Errorf("%d complaints at %s, which doesn't report them", n, mbp)
		}
		complained += n
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if complained == 0 {
		t.Fatal("the sample's sends hold no complaints at all")
	}
}

// The demo seed must run cleanly on a fresh database (seededSample's) and
// be idempotent.
func TestSeed(t *testing.T) {
	pool, box, _ := seededSample(t)
	cfg := config.Defaults()
	cfg.Files.Dir = t.TempDir() // the sample product files (X19)
	ctx := context.Background()
	if _, err := Seed(ctx, pool, &cfg, box, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("second seed must be a no-op: %v", err)
	}
	var campaigns int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM campaigns`).Scan(&campaigns); err != nil {
		t.Fatal(err)
	}
	// The Campaigns board's 48 plus the approval inbox's sample campaigns,
	// the draft blog.acme.io made from a post (integrations hub), the
	// accessibility sample's Weekly Digest #143, the recurring digest's
	// template and sent issues #1 and #2, and the survey's email, Beta
	// Program: October check-in (polls and surveys). SMS adds the sent
	// "Summit: 8 days to go" and the draft "Summit: doors open". Web push
	// adds the sent "New post: Feature flags at scale" and WhatsApp the draft
	// "Summit: see you tomorrow".
	if campaigns != 62 {
		t.Fatalf("want 62 demo campaigns, got %d", campaigns)
	}
}
