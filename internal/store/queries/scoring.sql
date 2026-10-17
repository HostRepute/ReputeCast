-- Contact scoring (audience-insights brief §3.1).

-- name: ScoringConfigGet :one
SELECT * FROM scoring_configs WHERE workspace_id = @workspace_id;

-- name: ScoringConfigGetForUpdate :one
SELECT * FROM scoring_configs WHERE workspace_id = @workspace_id FOR UPDATE;

-- name: ScoringConfigCreate :one
INSERT INTO scoring_configs (workspace_id, version, applied_version, set_up_by, set_up_at, updated_at)
VALUES (@workspace_id, 0, 0, sqlc.narg(set_up_by), @at::timestamptz, @at::timestamptz)
ON CONFLICT (workspace_id) DO UPDATE SET updated_at = scoring_configs.updated_at
RETURNING *;

-- name: ScoringConfigSetVersion :exec
UPDATE scoring_configs SET version = @version, updated_at = @at WHERE workspace_id = @workspace_id;

-- name: ScoringConfigSetApplied :exec
UPDATE scoring_configs SET applied_version = @applied_version, updated_at = @at WHERE workspace_id = @workspace_id;

-- name: ScoringConfigSetIngested :exec
UPDATE scoring_configs SET ingested_through = @ingested_through WHERE workspace_id = @workspace_id;

-- name: ScoringConfigSetNightly :exec
UPDATE scoring_configs SET nightly_at = @nightly_at WHERE workspace_id = @workspace_id;

-- name: ScoringWorkspaces :many
-- Workspaces that set up scoring, with what the jobs need.
SELECT sc.workspace_id, sc.version, sc.applied_version, sc.ingested_through, sc.nightly_at, w.slug, w.timezone
FROM scoring_configs sc
JOIN workspaces w ON w.id = sc.workspace_id AND w.deleted_at IS NULL
WHERE sc.applied_version > 0
ORDER BY sc.workspace_id;

-- name: ScoringVersionGet :one
SELECT * FROM scoring_versions WHERE workspace_id = @workspace_id AND version = @version;

-- name: ScoringVersionInsert :exec
INSERT INTO scoring_versions (workspace_id, version, config, created_by, created_at)
VALUES (@workspace_id, @version, @config, @created_by, @created_at);

-- name: ScoringRecalcInsert :one
INSERT INTO scoring_recalcs (id, workspace_id, version, reason, start_automations, total, label, changes, started_by, started_at)
VALUES (@id, @workspace_id, @version, @reason, @start_automations, @total, @label, @changes, @started_by, @started_at)
RETURNING *;

-- name: ScoringRecalcGet :one
SELECT * FROM scoring_recalcs WHERE id = @id;

-- name: ScoringRecalcLatest :one
SELECT r.*, coalesce(u.name, '')::text AS started_by_name
FROM scoring_recalcs r
LEFT JOIN users u ON u.id = r.started_by
WHERE r.workspace_id = @workspace_id
ORDER BY r.started_at DESC
LIMIT 1;

-- name: ScoringRecalcRunning :one
SELECT * FROM scoring_recalcs WHERE workspace_id = @workspace_id AND status = 'running';

-- name: ScoringRecalcProgress :exec
UPDATE scoring_recalcs SET done = @done, cursor = @cursor WHERE id = @id;

-- name: ScoringRecalcFinish :exec
UPDATE scoring_recalcs SET status = 'finished', done = total, finished_at = @at, summary = @summary, error = '' WHERE id = @id;

-- name: ScoringRecalcFail :exec
UPDATE scoring_recalcs SET status = 'failed', failed_at = @at, error = @error WHERE id = @id AND status = 'running';

-- name: ScoringRecalcRestart :exec
UPDATE scoring_recalcs SET status = 'running', failed_at = NULL, error = '', reason = 'retry', started_by = @started_by, started_at = @at
WHERE id = @id;

-- name: ScoringRecalcClearStage :exec
DELETE FROM scoring_recalc_scores WHERE recalc_id = @recalc_id;

-- name: ScoringContactsAfter :many
-- The next batch of contacts a recalculation scores, in id order.
SELECT id FROM contacts
WHERE workspace_id = @workspace_id AND erased_at IS NULL AND id > @after
ORDER BY id
LIMIT @lim;

-- name: ScoringCountContacts :one
SELECT count(*)::bigint FROM contacts WHERE workspace_id = @workspace_id AND erased_at IS NULL;

-- name: ScoringEventsOf :many
-- Score events of some contacts within the kept window, in time order.
SELECT contact_id, occurred_at, kind, ref, lists, message_id, counted
FROM score_events
WHERE contact_id = ANY (@contact_ids::uuid[]) AND occurred_at > @since
ORDER BY contact_id, occurred_at, source_id;

-- name: ScoringContactScore :one
SELECT * FROM contact_scores WHERE contact_id = @contact_id;

-- name: ScoringScoresOf :many
SELECT contact_id, score, events_part, fields_part, active_at, changed_at, next_check_at, version
FROM contact_scores
WHERE contact_id = ANY (@contact_ids::uuid[]);

-- Batches of computed scores are written by internal/scoring itself
-- (upsertScoresSQL, stageScoresSQL): their nullable times don't fit
-- sqlc's array parameters.

-- name: ScoringStagedAfter :many
-- The next batch of staged scores to apply, with the score each contact has now.
SELECT st.contact_id, st.score, st.events_part, st.fields_part, st.active_at, st.changed_at, st.next_check_at,
       cs.score AS old_score
FROM scoring_recalc_scores st
LEFT JOIN contact_scores cs ON cs.contact_id = st.contact_id
WHERE st.recalc_id = @recalc_id AND st.contact_id > @after
ORDER BY st.contact_id
LIMIT @lim;

-- name: ScoringDueChecks :many
-- Contacts whose score changes on its own by now: a decay step or an
-- event leaving the look-back.
SELECT contact_id FROM contact_scores
WHERE workspace_id = @workspace_id AND next_check_at <= @at::timestamptz
ORDER BY next_check_at
LIMIT @lim;

-- name: ScoringDirtyTake :many
DELETE FROM scoring_dirty
WHERE contact_id IN (SELECT d.contact_id FROM scoring_dirty d WHERE d.workspace_id = @workspace_id ORDER BY d.queued_at LIMIT @lim FOR UPDATE SKIP LOCKED)
RETURNING contact_id;

-- name: ScoringDirtyMarkSince :exec
-- Contacts with new events since a recalculation started: score them again.
INSERT INTO scoring_dirty (contact_id, workspace_id)
SELECT DISTINCT se.contact_id, se.workspace_id FROM score_events se
WHERE se.workspace_id = @workspace_id AND se.occurred_at >= @since
ON CONFLICT (contact_id) DO NOTHING;

-- name: ScoringPruneEvents :execrows
DELETE FROM score_events WHERE workspace_id = @workspace_id AND occurred_at < @before;

-- name: ScoringEventsByContact :many
-- One contact's events for the profile (the kept window, oldest first).
SELECT se.occurred_at, se.kind, se.ref, se.lists, se.message_id, se.counted, se.source_id
FROM score_events se
WHERE se.contact_id = @contact_id AND se.occurred_at > @since
ORDER BY se.occurred_at, se.source_id;

-- name: ScoringSpread :many
-- Contacts per bucket of 10 points (90–100 is the last); contacts without
-- a score row count as 0.
SELECT least(coalesce(cs.score, 0) / 10, 9)::int AS bucket, count(*)::bigint AS contacts
FROM contacts c
LEFT JOIN contact_scores cs ON cs.contact_id = c.id
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL
GROUP BY 1
ORDER BY 1;

-- name: ScoringBandsByTier :many
SELECT c.tier,
       count(*)::bigint AS contacts,
       count(*) FILTER (WHERE coalesce(cs.score, 0) >= @hot_from::int)::bigint AS hot,
       count(*) FILTER (WHERE coalesce(cs.score, 0) >= @warm_from::int AND coalesce(cs.score, 0) < @hot_from::int)::bigint AS warm,
       count(*) FILTER (WHERE coalesce(cs.score, 0) < @warm_from::int)::bigint AS cold
FROM contacts c
LEFT JOIN contact_scores cs ON cs.contact_id = c.id
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL
GROUP BY c.tier;

-- name: ScoringScoreCounts :many
-- Contacts per exact score: the impact preview re-buckets these when only
-- the thresholds move.
SELECT coalesce(cs.score, 0)::int AS score, count(*)::bigint AS contacts
FROM contacts c
LEFT JOIN contact_scores cs ON cs.contact_id = c.id
WHERE c.workspace_id = @workspace_id AND c.erased_at IS NULL
GROUP BY 1;

-- name: ScoringLiveTriggers :many
-- Automations that start on "Score crosses…", live or not.
SELECT a.id, a.name, a.status, a.trigger, a.created_by, a.live_version, a.created_at,
       coalesce(u.name, '')::text AS created_by_name,
       coalesce((SELECT min(v.published_at) FROM automation_versions v WHERE v.automation_id = a.id AND v.published_at IS NOT NULL), '0001-01-01T00:00:00Z'::timestamptz)::timestamptz AS live_since,
       (SELECT count(*) FROM automation_runs r WHERE r.automation_id = a.id)::bigint AS entered
FROM automations a
LEFT JOIN users u ON u.id = a.created_by
WHERE a.workspace_id = @workspace_id AND a.deleted_at IS NULL AND a.trigger ->> 'kind' = 'score_crosses'
ORDER BY a.created_at;

-- name: ScoringSegmentsUsingScore :many
-- Segments with a score or score-band condition.
SELECT s.id, s.name, s.cached_count, s.definition, coalesce(u.name, '')::text AS created_by_name
FROM segments s
LEFT JOIN users u ON u.id = s.created_by
WHERE s.workspace_id = @workspace_id AND s.mode = 'builder'
  AND EXISTS (SELECT 1 FROM jsonb_array_elements(coalesce(s.definition -> 'groups', '[]'::jsonb)) g,
                           jsonb_array_elements(coalesce(g -> 'conditions', '[]'::jsonb)) cond
              WHERE cond ->> 'field' IN ('score', 'score_band'))
ORDER BY s.name;

-- name: ScoringInsertTriggerEvents :exec
-- "Score crosses…" trigger events for contacts whose score crossed a live
-- automation's threshold.
INSERT INTO automation_events (workspace_id, kind, contact_id, ref, payload, occurred_at)
SELECT @workspace_id, 'score_crosses', u.contact_id, '', jsonb_build_object('from', u.from_score, 'to', u.to_score, 'cause', @cause::text), @at
FROM (SELECT unnest(@contact_ids::uuid[]) AS contact_id, unnest(@from_scores::int[]) AS from_score, unnest(@to_scores::int[]) AS to_score) u;

-- name: ScoringFormsOf :many
SELECT id, name, kind FROM forms WHERE workspace_id = @workspace_id ORDER BY created_at;

-- name: ScoringListsOf :many
SELECT id, name FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY created_at;

-- name: ScoringFieldsOf :many
SELECT key, label, type, options FROM custom_fields WHERE workspace_id = @workspace_id ORDER BY position, created_at;

-- name: ScoringAPIEventNames :many
-- API events sent in the last 30 days, with how many: the rule editor
-- suggests them.
SELECT ref AS name, count(*)::bigint AS n
FROM automation_events
WHERE workspace_id = @workspace_id AND kind = 'api_event' AND occurred_at >= @since
GROUP BY ref
ORDER BY 2 DESC, 1
LIMIT 10;

-- name: ScoringAPIEventNamesFromScores :many
-- The same from the scorer's own copy (automation events are purged sooner).
SELECT ref AS name, count(*)::bigint AS n
FROM score_events
WHERE workspace_id = @workspace_id AND kind = 'api_event' AND occurred_at >= @since
GROUP BY ref
ORDER BY 2 DESC, 1
LIMIT 10;

-- name: ScoringMessageLabels :many
-- What a message was: the campaign or the automation and step.
SELECT m.id, coalesce(c.name, a.name, '')::text AS email_name, m.automation_node, m.campaign_id
FROM messages m
LEFT JOIN campaigns c ON c.id = m.campaign_id
LEFT JOIN automations a ON a.id = m.automation_id
WHERE m.id = ANY (@ids::uuid[]);

-- name: ScoringLinkLabels :many
SELECT campaign_id, url, label FROM campaign_links WHERE campaign_id = ANY (@campaign_ids::uuid[]);

-- name: ScoringContactBasics :one
SELECT c.id, c.workspace_id, c.created_at, c.tier, c.erased_at FROM contacts c WHERE c.id = @id;

-- name: ScoringRecalcSupersede :exec
UPDATE scoring_recalcs SET status = 'superseded', finished_at = @at WHERE workspace_id = @workspace_id AND status IN ('running', 'failed');

-- name: ScoringRecalcStatus :one
SELECT status FROM scoring_recalcs WHERE id = @id;

-- name: ScoringFormOrigins :many
-- The sites a workspace's forms are on (the suggested pricing rule names one).
SELECT DISTINCT o::text FROM forms f, unnest(f.allowed_origins) o WHERE f.workspace_id = @workspace_id;

-- name: AutomationScoreRunSince :one
-- Whether a contact entered a "Score crosses…" automation recently, or is in it.
SELECT EXISTS (
    SELECT 1 FROM automation_runs
    WHERE automation_id = @automation_id AND contact_id = @contact_id
      AND (status IN ('active', 'waiting') OR entered_at >= @since::timestamptz)
)::boolean;
