package loadtest_test

import (
	"context"
	"fmt"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/testdb"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// hanoi stands in for a GeoIP database (no .mmdb file in tests).
type hanoi struct{}

func (hanoi) Lookup(netip.Addr) (string, string) { return "VN", "Hanoi" }

// engagementWorkers record opens and clicks at once on one campaign, each
// like one process's group-commit writer (tracking's loop): pixel and
// redirect requests a few milliseconds apart reach the database together,
// engagementBatch of them per transaction.
const (
	engagementWorkers = 8
	engagementBatch   = 16
)

// BenchmarkEngagementDims measures what the campaign report's breakdowns
// (G-12) add to recording engagement: per op, a person's first open and a
// click that settles, through tracking and Ledger.Flush, from several
// writers on one campaign whose messages go through several providers.
// "off" is a campaign sent before breakdowns with no GeoIP; "on" records
// them with a GeoIP lookup per first engagement. The budget is on ≤ 1.25×
// off:
//
//	go test -run '^$' -bench EngagementDims -benchtime 20000x ./internal/loadtest/
//
// It runs on the test database (REPUTECAST_TEST_DATABASE_URL), not the
// load sample.
func BenchmarkEngagementDims(b *testing.B) {
	env := apptest.New(b)
	env.SeedDemo()
	ctx := context.Background()
	// A pool of its own: the test app's is kept small. A writer's
	// transaction holds one connection and its cached lookups (machine
	// networks, links) take another, so each worker gets two.
	pc, err := pgxpool.ParseConfig(testdb.URL(env.App.Pool))
	if err != nil {
		b.Fatal(err)
	}
	pc.MaxConns = 2*engagementWorkers + 4
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(pool.Close)
	var ws uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&ws); err != nil {
		b.Fatal(err)
	}
	var provs []uuid.UUID
	rows, err := pool.Query(ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND disconnected_at IS NULL ORDER BY id`, ws)
	if err != nil {
		b.Fatal(err)
	}
	provs, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil || len(provs) == 0 {
		b.Fatalf("providers: %v (%d)", err, len(provs))
	}
	for _, mode := range []struct {
		name      string
		breakdown bool
		geo       bool
	}{{"off", false, false}, {"on", true, true}} {
		b.Run(mode.name, func(b *testing.B) {
			base := time.Now().Truncate(time.Second)
			campaign := engagementCampaign(b, pool, ws, provs, mode.breakdown, base, b.N)
			tr := &tracking.Service{Pool: pool, Q: store.New(pool), Links: env.App.Links}
			if mode.geo {
				tr.Geo = hanoi{}
			}
			agents := []string{
				"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
				"Microsoft Office/16.0 (Windows NT 10.0; Microsoft Outlook 16.0.17928; Pro)",
				"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
			}
			var next atomic.Int64
			var wg sync.WaitGroup
			b.ResetTimer()
			for range engagementWorkers {
				wg.Go(func() {
					opens := make([]sending.Engagement, 0, engagementBatch)
					clicks := make([]sending.Engagement, 0, engagementBatch)
					for {
						from := int(next.Add(engagementBatch) - engagementBatch)
						if from >= b.N {
							return
						}
						opens, clicks = opens[:0], clicks[:0]
						for i := from; i < min(from+engagementBatch, b.N); i++ {
							id := campaign.messages[i]
							ua, ip := agents[i%len(agents)], fmt.Sprintf("198.51.%d.%d", 100+i/250%50, i%250+1)
							opens = append(opens, sending.Engagement{MessageID: id, UserAgent: ua, IP: ip, Method: "GET", At: base.Add(-30 * time.Minute)})
							clicks = append(clicks, sending.Engagement{MessageID: id, UserAgent: ua, IP: ip, Method: "GET", Click: true, Link: 0, At: base.Add(-20 * time.Minute)})
						}
						if err := tr.RecordEngagement(ctx, opens); err != nil {
							b.Error(err)
							return
						}
						if err := tr.RecordEngagement(ctx, clicks); err != nil {
							b.Error(err)
							return
						}
					}
				})
			}
			wg.Wait()
			for {
				due, err := tr.SettleEngagement(ctx, base)
				if err != nil {
					b.Fatal(err)
				}
				if due == nil || due.After(base) {
					break
				}
			}
			b.StopTimer()
			var clicked int64
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM messages WHERE campaign_id = $1 AND clicked_at IS NOT NULL`, campaign.id).Scan(&clicked); err != nil {
				b.Fatal(err)
			}
			if clicked != int64(b.N) {
				b.Fatalf("%d of %d clicks settled", clicked, b.N)
			}
			if mode.breakdown {
				var n int64
				if err := pool.QueryRow(ctx, `SELECT coalesce(sum(clicked), 0) FROM campaign_dims WHERE campaign_id = $1 AND dim = 'country'`, campaign.id).Scan(&n); err != nil || n != int64(b.N) {
					b.Fatalf("breakdowns: %d clicks by country, want %d (%v)", n, b.N, err)
				}
			}
		})
	}
}

type benchCampaign struct {
	id       uuid.UUID
	messages []uuid.UUID
}

// engagementCampaign is a sent copy of Weekly Digest #142 with n delivered
// messages to n contacts of its own, through the workspace's providers in
// turn, and one link.
func engagementCampaign(b *testing.B, pool *pgxpool.Pool, ws uuid.UUID, provs []uuid.UUID, breakdown bool, base time.Time, n int) benchCampaign {
	b.Helper()
	ctx := context.Background()
	c := benchCampaign{id: uuid.Must(uuid.NewV7())}
	if _, err := pool.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, reply_to, audience, editor_mode, mjml, html, plain_text,
		tracking, created_by, updated_by, content_hash, started_at, breakdown)
		SELECT $1, workspace_id, $2, 'sent', subject, preheader, sender_id, reply_to, '{}', editor_mode, mjml, html, plain_text,
		tracking, created_by, updated_by, content_hash, $3, $4
		FROM campaigns WHERE workspace_id = $5 AND name = 'Weekly Digest #142'`,
		c.id, "Engagement bench "+c.id.String(), base.Add(-2*time.Hour), breakdown, ws); err != nil {
		b.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO campaign_links (id, campaign_id, url, position) VALUES ($1, $2, 'https://acme.io/blog/boring-infrastructure', 0)`,
		uuid.Must(uuid.NewV7()), c.id); err != nil {
		b.Fatal(err)
	}
	mailboxes := []string{"gmail", "outlook", "yahoo", "icloud"}
	contacts := make([][]any, 0, n)
	messages := make([][]any, 0, n)
	sent, delivered := base.Add(-time.Hour), base.Add(-59*time.Minute)
	for i := range n {
		cid, mid := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
		email := fmt.Sprintf("bench.%s.%d@example.com", c.id, i)
		mbx := mailboxes[i%len(mailboxes)]
		contacts = append(contacts, []any{cid, ws, email, mbx})
		messages = append(messages, []any{mid, ws, "broadcast", c.id, cid, email, provs[i%len(provs)], mbx, "delivered", sent, delivered})
		c.messages = append(c.messages, mid)
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.CopyFrom(ctx, pgx.Identifier{"contacts"}, []string{"id", "workspace_id", "email", "mailbox_provider"}, pgx.CopyFromRows(contacts)); err != nil {
		b.Fatal(err)
	}
	if _, err := conn.CopyFrom(ctx, pgx.Identifier{"messages"}, []string{"id", "workspace_id", "stream", "campaign_id", "contact_id", "email", "provider_id",
		"mailbox_provider", "status", "sent_at", "delivered_at"}, pgx.CopyFromRows(messages)); err != nil {
		b.Fatal(err)
	}
	return c
}
