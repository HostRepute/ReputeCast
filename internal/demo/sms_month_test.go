package demo

import (
	"context"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/texting"
)

// The SMS sample's numbers for this month, whatever the day the demo is
// seeded: the providers page's tiles that TestSMSDemo, TestSMSTools and
// TestSMSInVietnamese read, at the clock they read them with (SMSReadAt,
// apptest's SMSClock). On the 14th and the 20th that's the seed's own
// moment, so the demo shows them too; early on the 1st, when nothing can
// reach the Pacific coast inside quiet hours yet this month, the sample
// keeps its texts in the month before. Either way every text went out in
// one month, inside its recipient's quiet hours, and before the seed.
func TestSMSMonth(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone data")
	}
	now := time.Now()
	for _, c := range []struct {
		name              string
		day, hour, minute int
		thisMonth         bool
	}{
		{name: "the 1st", day: 1, hour: 6},
		// Just after the design's today begins (11:00): the campaign went
		// out yesterday, the draft was written this morning.
		{name: "the 14th", day: 14, hour: 11, minute: 30, thisMonth: true},
		{name: "the 20th", day: 20, hour: 12, thisMonth: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// The last such moment before now.
			l := now.In(ny)
			at := time.Date(l.Year(), l.Month(), c.day, c.hour, c.minute, 0, 0, ny)
			if at.After(now) {
				at = at.AddDate(0, -1, 0)
			}
			pool, _ := seededAt(t, at)
			ctx := context.Background()
			q := store.New(pool)
			ws, err := q.GetWorkspaceBySlug(ctx, "acme-newsletter")
			if err != nil {
				t.Fatal(err)
			}
			read, err := SMSReadAt(ctx, pool, at)
			if err != nil {
				t.Fatal(err)
			}
			if read.Equal(at) != c.thisMonth {
				t.Fatalf("seeded %s, read at %s", at, read)
			}
			r := read.In(ny)
			month := time.Date(r.Year(), r.Month(), 1, 0, 0, 0, 0, ny)

			// Every text the month counts went out between its start and
			// the seed; nothing of the sample comes after the seed.
			var first, last, latest time.Time
			if err := pool.QueryRow(ctx, `SELECT min(sent_at), max(sent_at),
					greatest(max(sent_at), max(delivered_at), max(failed_at), max(created_at), max(updated_at))
				FROM channel_messages WHERE workspace_id = $1 AND channel = 'sms' AND kind <> 'reply' AND sent_at IS NOT NULL`, ws.ID).Scan(&first, &last, &latest); err != nil {
				t.Fatal(err)
			}
			if first.Before(month) || last.After(read) {
				t.Fatalf("texts from %s to %s, read at %s", first.In(ny), last.In(ny), r)
			}
			var after int
			if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM channel_message_events WHERE workspace_id = $1 AND occurred_at > $2)
					+ (SELECT count(*) FROM sms_inbound WHERE workspace_id = $1 AND received_at > $2)
					+ (SELECT count(*) FROM sms_link_clicks WHERE workspace_id = $1 AND occurred_at > $2)
					+ (SELECT count(*) FROM channel_messages WHERE workspace_id = $1 AND channel = 'sms' AND coalesce(sent_at, created_at) > $2)`,
				ws.ID, at).Scan(&after); err != nil {
				t.Fatal(err)
			}
			if after != 0 || latest.After(at) {
				t.Fatalf("%d of the sample's rows come after the seed (%s), its texts reach %s", after, at, latest)
			}
			// The confirmations and the campaign reached everyone inside
			// their quiet hours, 09:00–20:00 where they are.
			var outside int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE workspace_id = $1 AND channel = 'sms'
				AND kind IN ('confirmation', 'campaign') AND sent_at IS NOT NULL
				AND extract(hour FROM sent_at AT TIME ZONE time_zone) NOT BETWEEN 9 AND 19`, ws.ID).Scan(&outside); err != nil {
				t.Fatal(err)
			}
			if outside != 0 {
				t.Fatalf("%d texts went out outside quiet hours", outside)
			}

			// The numbers the tests read.
			s := &texting.Service{Pool: pool, Q: q, URLs: publicurl.New("http://localhost:8080"), Now: func() time.Time { return read }}
			linh := &authz.Principal{Type: authz.ActorUser, WorkspaceID: ws.ID, Permissions: authz.All()}
			ov, err := s.Overview(ctx, &ws, linh)
			if err != nil {
				t.Fatal(err)
			}
			tl := ov.Tiles
			if tl.Sent != 1731 || tl.ByKind["confirmation"] != 871 || tl.ByKind["campaign"] != 854 || tl.ByKind["test"] != 6 || tl.Delivered != 1710 || tl.Failed != 21 {
				t.Fatalf("texts: %+v", tl)
			}
			if tl.Spend != 39.51 || tl.Budget != 100 || len(ov.Providers) != 2 || ov.Providers[0].Spend != 14.71 || ov.Providers[1].Spend != 24.80 {
				t.Fatalf("spend: %+v, providers %+v", tl, ov.Providers)
			}
			rules, err := s.Rules(ctx, &ws, linh)
			if err != nil {
				t.Fatal(err)
			}
			if rules.TestsSent != 6 || rules.TestsCost != 0.07 {
				t.Fatalf("test texts: %d, $%.2f", rules.TestsSent, rules.TestsCost)
			}
		})
	}
}
