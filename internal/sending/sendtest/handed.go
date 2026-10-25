package sendtest

import (
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending/engine"
)

// HandedOver sends the fixture's campaign to its contacts (without the
// A/B test) until every message is with a provider, then drops what the
// demo transport planned for them, so a test scripts their deliveries,
// opens and clicks itself. It returns the messages in id order.
func (f *Fixture) HandedOver(specs ...ContactSpec) []uuid.UUID {
	f.T.Helper()
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	n := 0
	for _, s := range specs {
		n += s.N
	}
	f.AddContacts(specs...)
	f.StartSend(engine.StartOptions{})
	f.Drive(time.Minute)
	if got := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); got != int64(n) {
		f.Dump()
		f.T.Fatalf("%d of %d sent", got, n)
	}
	f.Exec(`DELETE FROM sending_sim_events WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`UPDATE messages SET status = 'sent', delivered_at = NULL, deferred_at = NULL, soft_bounced_at = NULL, opened_at = NULL,
		human_opened_at = NULL, privacy_opened_at = NULL, clicked_at = NULL, unsubscribed_at = NULL WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`UPDATE campaign_counters SET delivered = 0, deferred = 0, hard_bounced = 0, soft_bounced = 0, complained = 0, rejected = 0,
		opened_human = 0, opened_privacy = 0, opened_machine = 0, clicked = 0, unsubscribed = 0 WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`UPDATE campaign_links SET clicks = 0, unique_clicks = 0, bot_clicks = 0 WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`DELETE FROM campaign_hourly WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`DELETE FROM tracking_pending_clicks WHERE message_id IN (SELECT id FROM messages WHERE campaign_id = $1)`, f.Campaign)
	f.Exec(`DELETE FROM tracking_link_clicks WHERE message_id IN (SELECT id FROM messages WHERE campaign_id = $1)`, f.Campaign)
	f.Exec(`DELETE FROM delivery_events WHERE campaign_id = $1`, f.Campaign)
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
