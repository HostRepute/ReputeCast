-- A/B tests' decisions (review AN-1, internal/sending/engine/ab.go).

-- name: ABSampleCounts :many
-- The test sample's results per variant; the rollout (variant '', the
-- recipients who got the winner after the test) is left out.
SELECT variant,
       coalesce(sum(sent), 0)::bigint AS sent, coalesce(sum(delivered), 0)::bigint AS delivered,
       coalesce(sum(opened_human), 0)::bigint AS opened_human, coalesce(sum(opened_privacy), 0)::bigint AS opened_privacy,
       coalesce(sum(clicked), 0)::bigint AS clicked, coalesce(sum(complained), 0)::bigint AS complained
FROM campaign_counters
WHERE campaign_id = @campaign_id AND variant <> ''
GROUP BY variant
ORDER BY variant;

-- name: ABReleaseHeld :execrows
-- Everyone who waited for the test gets the variant that goes out (the
-- winner, or the default when no variant won), counted apart as the
-- rollout; a best-time send keeps each recipient's time.
UPDATE messages
SET status = 'queued', variant = @variant, ab_rollout = true, claimed_at = NULL
WHERE campaign_id = @campaign_id AND status = 'held' AND variant = '';

-- name: ABHeldZones :many
-- The recipients waiting for a send-time test, with their time zone.
SELECT m.id, coalesce(c.timezone, '')::text AS timezone
FROM messages m
LEFT JOIN contacts c ON c.id = m.contact_id
WHERE m.campaign_id = @campaign_id AND m.status = 'held' AND m.variant = ''
ORDER BY m.id;

-- name: ABReleaseAt :execrows
-- A send-time test's rest get the variant at its time of day in their own
-- time zone.
UPDATE messages m
SET status = 'queued', variant = @variant, ab_rollout = true, claimed_at = NULL, next_attempt_at = x.at
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@ats::timestamptz[]) AS at) x
WHERE m.id = x.id AND m.status = 'held';

-- name: ABLastSampleDue :one
-- When the last recipient of the test sample is due (a send-time test's
-- latest variant time in its recipients' time zones).
SELECT coalesce(max(next_attempt_at), 'epoch'::timestamptz)::timestamptz
FROM messages
WHERE campaign_id = @campaign_id AND variant <> '' AND NOT ab_rollout AND status IN ('queued', 'sent', 'delivered', 'deferred');

-- name: ABSaveDecision :exec
-- The decision is kept with the campaign's test, for its report.
UPDATE campaigns
SET ab_test = jsonb_set(jsonb_set(coalesce(ab_test, '{}'::jsonb), '{winner}', to_jsonb(@winner::text)), '{decision}', @decision::jsonb)
WHERE id = @id AND ab_test IS NOT NULL;
