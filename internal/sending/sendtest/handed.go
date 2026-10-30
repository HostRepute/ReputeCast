package sendtest

import (
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending/engine"
)

// HandedOver sends the fixture's campaign to its contacts (without the
// A/B test) until every message is with a provider, and drops what the
// demo transport plans for them before any of it comes due (a chance
// bounce or complaint would suppress the address), so a test scripts their
// deliveries, opens and clicks itself. It returns the messages in id order.
func (f *Fixture) HandedOver(specs ...ContactSpec) []uuid.UUID {
	f.T.Helper()
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	n := 0
	for _, s := range specs {
		n += s.N
	}
	f.AddContacts(specs...)
	f.StartSend(engine.StartOptions{})
	f.DriveWithout(time.Minute)
	if got := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); got != int64(n) {
		f.Dump()
		f.T.Fatalf("%d of %d sent", got, n)
	}
	if got := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND (status <> 'sent' OR delivered_at IS NOT NULL OR deferred_at IS NOT NULL
		OR soft_bounced_at IS NOT NULL OR opened_at IS NOT NULL OR clicked_at IS NOT NULL)`, f.Campaign); got != 0 {
		f.Dump()
		f.T.Fatalf("the demo transport reported on %d messages", got)
	}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT id FROM messages WHERE campaign_id = $1 ORDER BY id`, f.Campaign)
	if err != nil {
		f.T.Fatal(err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			f.T.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

// Counts are the fixture's campaign counters, summed over its cells, by
// column name (sent, delivered, opened_human…).
func (f *Fixture) Counts() map[string]int64 {
	f.T.Helper()
	cols := []string{"sent", "delivered", "deferred", "hard_bounced", "soft_bounced", "complained", "rejected",
		"opened_human", "opened_privacy", "opened_machine", "clicked", "unsubscribed"}
	vals := make([]int64, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT coalesce(sum(sent), 0), coalesce(sum(delivered), 0), coalesce(sum(deferred), 0),
		coalesce(sum(hard_bounced), 0), coalesce(sum(soft_bounced), 0), coalesce(sum(complained), 0), coalesce(sum(rejected), 0),
		coalesce(sum(opened_human), 0), coalesce(sum(opened_privacy), 0), coalesce(sum(opened_machine), 0), coalesce(sum(clicked), 0),
		coalesce(sum(unsubscribed), 0) FROM campaign_counters WHERE campaign_id = $1`, f.Campaign).Scan(ptrs...); err != nil {
		f.T.Fatal(err)
	}
	out := make(map[string]int64, len(cols))
	for i, c := range cols {
		out[c] = vals[i]
	}
	return out
}
