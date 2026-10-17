package scoring

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// The scorer copies what it counts into score_events: human opens and real
// clicks of campaigns and automation emails (never transactional or system
// mail, never machine opens or bot clicks), confirmed form submissions, API
// events, soft bounces once their retries are over, and list unsubscribes.
// Each source row gets a fixed source id (an MD5 of what it is), so reading
// a window again never counts anything twice: the windows overlap on
// purpose, for rows that commit after their time.

// ingestOverlap is how far each run reads back past the last one.
const ingestOverlap = 2 * time.Minute

// listsSQL is an email's lists: a campaign's audience lists, or the list
// that starts an automation.
const listsSQL = `CASE
    WHEN cp.id IS NOT NULL THEN coalesce(ARRAY(
        SELECT x::uuid FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(cp.audience -> 'lists') = 'array' THEN cp.audience -> 'lists' ELSE '[]'::jsonb END) x
        WHERE x ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'), '{}'::uuid[])
    WHEN a.trigger ->> 'kind' = 'joined_list' AND (a.trigger ->> 'listId') ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        THEN ARRAY[(a.trigger ->> 'listId')::uuid]
    ELSE '{}'::uuid[] END`

var ingestSQL = []string{
	// Human opens, one per email.
	`INSERT INTO score_events (workspace_id, contact_id, source_id, occurred_at, kind, ref, lists, message_id, counted)
	SELECT de.workspace_id, de.contact_id, md5('open:' || de.message_id::text)::uuid, de.occurred_at, 'open', '', ` + listsSQL + `, de.message_id, true
	FROM delivery_events de
	JOIN messages m ON m.id = de.message_id AND m.stream IN ('broadcast', 'automation')
	JOIN contacts c ON c.id = de.contact_id AND c.erased_at IS NULL
	LEFT JOIN campaigns cp ON cp.id = m.campaign_id
	LEFT JOIN automations a ON a.id = m.automation_id
	WHERE de.workspace_id = $1 AND de.received_at > $2 AND de.received_at <= $3
	  AND de.type = 'open' AND NOT de.machine
	ORDER BY de.occurred_at
	ON CONFLICT (contact_id, source_id) DO NOTHING
	RETURNING contact_id`,
	// Real clicks, one per email and link; a click is a human open too.
	`WITH clicks AS (
	    SELECT de.workspace_id, de.contact_id, de.message_id, de.occurred_at, de.url, ` + listsSQL + ` AS lists
	    FROM delivery_events de
	    JOIN messages m ON m.id = de.message_id AND m.stream IN ('broadcast', 'automation')
	    JOIN contacts c ON c.id = de.contact_id AND c.erased_at IS NULL
	    LEFT JOIN campaigns cp ON cp.id = m.campaign_id
	    LEFT JOIN automations a ON a.id = m.automation_id
	    WHERE de.workspace_id = $1 AND de.received_at > $2 AND de.received_at <= $3
	      AND de.type = 'click' AND NOT de.machine
	)
	INSERT INTO score_events (workspace_id, contact_id, source_id, occurred_at, kind, ref, lists, message_id, counted)
	SELECT workspace_id, contact_id, source_id, occurred_at, kind, ref, lists, message_id, true FROM (
	    SELECT workspace_id, contact_id, md5('click:' || message_id::text || ':' || url)::uuid AS source_id, occurred_at, 'click' AS kind, url AS ref, lists, message_id, 1 AS o FROM clicks
	    UNION ALL
	    SELECT workspace_id, contact_id, md5('open:' || message_id::text)::uuid, occurred_at, 'open', '', lists, message_id, 2 FROM clicks
	) x
	ORDER BY occurred_at, o
	ON CONFLICT (contact_id, source_id) DO NOTHING
	RETURNING contact_id`,
	// Soft bounces 72 hours on, when the provider's retries are over: a
	// delivery after them shows, and then they don't count.
	`INSERT INTO score_events (workspace_id, contact_id, source_id, occurred_at, kind, ref, lists, message_id, counted)
	SELECT de.workspace_id, de.contact_id, md5('soft_bounce:' || de.message_id::text)::uuid, de.occurred_at, 'soft_bounce', '', '{}', de.message_id,
	       NOT (m.delivered_at IS NOT NULL AND m.delivered_at >= de.occurred_at)
	FROM delivery_events de
	JOIN messages m ON m.id = de.message_id AND m.stream IN ('broadcast', 'automation')
	JOIN contacts c ON c.id = de.contact_id AND c.erased_at IS NULL
	WHERE de.workspace_id = $1 AND de.received_at > $2::timestamptz - interval '72 hours' AND de.received_at <= $3::timestamptz - interval '72 hours'
	  AND de.type = 'soft_bounce'
	ORDER BY de.occurred_at
	ON CONFLICT (contact_id, source_id) DO NOTHING
	RETURNING contact_id`,
	// Confirmed form submissions (double opt-in ones when confirmed).
	`INSERT INTO score_events (workspace_id, contact_id, source_id, occurred_at, kind, ref, lists, message_id, counted)
	SELECT fs.workspace_id, fs.contact_id, md5('form:' || fs.id::text)::uuid, fs.confirmed_at, 'form', fs.form_id::text, '{}', NULL, true
	FROM form_submissions fs
	JOIN contacts c ON c.id = fs.contact_id AND c.erased_at IS NULL
	WHERE fs.workspace_id = $1 AND fs.form_id IS NOT NULL AND fs.created_at > $2::timestamptz - interval '30 days'
	  AND fs.confirmed_at > $2 AND fs.confirmed_at <= $3
	ON CONFLICT (contact_id, source_id) DO NOTHING
	RETURNING contact_id`,
	// API events.
	`INSERT INTO score_events (workspace_id, contact_id, source_id, occurred_at, kind, ref, lists, message_id, counted)
	SELECT ae.workspace_id, ae.contact_id, md5('api_event:' || ae.id::text)::uuid, ae.occurred_at, 'api_event', ae.ref, '{}', NULL, true
	FROM automation_events ae
	JOIN contacts c ON c.id = ae.contact_id AND c.erased_at IS NULL
	WHERE ae.workspace_id = $1 AND ae.kind = 'api_event' AND ae.occurred_at > $2 AND ae.occurred_at <= $3
	ON CONFLICT (contact_id, source_id) DO NOTHING
	RETURNING contact_id`,
	// Unsubscribes from a list.
	`INSERT INTO score_events (workspace_id, contact_id, source_id, occurred_at, kind, ref, lists, message_id, counted)
	SELECT c.workspace_id, ce.contact_id, md5('unsubscribe:' || ce.id::text)::uuid, ce.occurred_at, 'unsubscribe', ce.list_id::text, '{}', NULL, true
	FROM consent_events ce
	JOIN contacts c ON c.id = ce.contact_id AND c.erased_at IS NULL
	WHERE ce.action = 'unsubscribed' AND ce.list_id IS NOT NULL AND ce.occurred_at > $2 AND ce.occurred_at <= $3 AND c.workspace_id = $1
	ON CONFLICT (contact_id, source_id) DO NOTHING
	RETURNING contact_id`,
}

// ingest copies the source rows of (from, to] into score_events and
// returns the contacts that got new events.
func ingest(ctx context.Context, tx pgx.Tx, wsID uuid.UUID, from, to time.Time) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]bool{}
	var out []uuid.UUID
	for _, sql := range ingestSQL {
		rows, err := tx.Query(ctx, sql, wsID, from, to)
		if err != nil {
			return nil, err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	return out, nil
}

// backfillDays is how far a workspace's first recalculation reads back.
const backfillDays = KeepDays

// backfill copies a workspace's last KeepDays of source rows into
// score_events, a day at a time so no transaction gets large, up to to.
func (s *Service) backfill(ctx context.Context, wsID uuid.UUID, to time.Time) error {
	from := to.AddDate(0, 0, -backfillDays)
	for day := from; day.Before(to); day = day.Add(24 * time.Hour) {
		end := day.Add(24 * time.Hour)
		if end.After(to) {
			end = to
		}
		err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
			_, err := ingest(ctx, tx, wsID, day, end)
			return err
		})
		if err != nil {
			return err
		}
	}
	return store.New(s.Pool).ScoringConfigSetIngested(ctx, store.ScoringConfigSetIngestedParams{WorkspaceID: wsID, IngestedThrough: &to})
}
