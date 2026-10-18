-- Address verification services (internal/verification): connections,
-- jobs and their addresses, each contact's latest result, real-time checks
-- on signup forms.

-- name: VerifServiceList :many
SELECT sqlc.embed(v), coalesce(cu.name, '')::text AS connected_by_name, coalesce(lu.name, '')::text AS last_used_by_name,
       coalesce(uu.name, '')::text AS updated_by_name
FROM verification_services v
LEFT JOIN users cu ON cu.id = v.connected_by
LEFT JOIN users lu ON lu.id = v.last_used_by
LEFT JOIN users uu ON uu.id = v.updated_by
WHERE v.workspace_id = @workspace_id
ORDER BY v.connected_at, v.id;

-- name: VerifServiceByKey :one
SELECT v.* FROM verification_services v WHERE v.workspace_id = @workspace_id AND v.service = @service;

-- name: VerifServiceByID :one
SELECT v.* FROM verification_services v WHERE v.id = @id;

-- name: VerifServiceAll :many
-- Every connected service of the instance, for the hourly balance read.
SELECT v.* FROM verification_services v
JOIN workspaces w ON w.id = v.workspace_id AND w.deleted_at IS NULL;

-- name: VerifServiceInsert :exec
INSERT INTO verification_services (id, workspace_id, service, region, credentials_enc, price_micros, default_bulk, default_forms,
                                   balance, balance_at, tested_at, test_ok, connected_by, connected_at, updated_by, updated_at)
VALUES (@id, @workspace_id, @service, @region, @credentials_enc, @price_micros, @default_bulk, @default_forms,
        @balance, @balance_at, @tested_at, @test_ok, @connected_by, @connected_at, @connected_by, @connected_at);

-- name: VerifServiceSettings :exec
UPDATE verification_services
SET price_micros = @price_micros, region = @region, default_bulk = @default_bulk, default_forms = @default_forms,
    updated_by = @updated_by, updated_at = @updated_at
WHERE id = @id;

-- name: VerifServiceClearDefaults :exec
-- One default per use: setting it on one service takes it off the others.
UPDATE verification_services
SET default_bulk = default_bulk AND NOT @bulk::boolean, default_forms = default_forms AND NOT @forms::boolean
WHERE workspace_id = @workspace_id AND id <> @keep_id;

-- name: VerifServiceSetKey :exec
UPDATE verification_services
SET credentials_enc = @credentials_enc, updated_by = @updated_by, updated_at = @updated_at,
    last_error = '', last_error_at = NULL, last_error_since = NULL
WHERE id = @id;

-- name: VerifServiceSetBalance :exec
UPDATE verification_services SET balance = @balance, balance_at = @balance_at WHERE id = @id;

-- name: VerifServiceTested :exec
UPDATE verification_services SET tested_at = @tested_at, test_ok = @test_ok WHERE id = @id;

-- name: VerifServiceUsed :exec
UPDATE verification_services SET last_used_at = @at, last_used_by = @by, last_used_what = @what WHERE id = @id;

-- name: VerifServiceFailed :exec
-- An error answer; the streak keeps its first time ("from 15:05 to 15:10").
UPDATE verification_services
SET last_error = @error, last_error_at = @at, last_error_since = coalesce(last_error_since, @at)
WHERE id = @id;

-- name: VerifServiceAnswered :exec
UPDATE verification_services SET last_success_at = @at, last_success_what = @what, last_error_since = NULL WHERE id = @id;

-- name: VerifServiceDelete :execrows
DELETE FROM verification_services WHERE id = @id AND workspace_id = @workspace_id;

-- name: VerifJobInsert :exec
INSERT INTO verification_jobs (id, workspace_id, service_id, service, kind, target, label, origin, campaign_id, import_id,
                               status, skip_recent, total, skipped, price_micros, balance_before, started_by, created_at)
VALUES (@id, @workspace_id, @service_id, @service, @kind, @target, @label, @origin, @campaign_id, @import_id,
        'queued', @skip_recent, @total, @skipped, @price_micros, @balance_before, @started_by, @created_at);

-- name: VerifJobItemsInsert :exec
INSERT INTO verification_job_items (job_id, email, contact_id, position)
SELECT @job_id, x.email, nullif(x.contact_id, '00000000-0000-0000-0000-000000000000'::uuid), x.position
FROM (SELECT unnest(@emails::text[]) AS email, unnest(@contact_ids::uuid[]) AS contact_id, unnest(@positions::int[]) AS position) AS x
ON CONFLICT DO NOTHING;

-- name: VerifJobGet :one
SELECT sqlc.embed(j), coalesce(u.name, '')::text AS started_by_name, coalesce(su.name, '')::text AS stopped_by_name
FROM verification_jobs j
LEFT JOIN users u ON u.id = j.started_by
LEFT JOIN users su ON su.id = j.stopped_by
WHERE j.id = @id AND j.workspace_id = @workspace_id;

-- name: VerifJobByID :one
SELECT j.* FROM verification_jobs j WHERE j.id = @id;

-- name: VerifJobList :many
-- Jobs of the workspace since a time, newest first (uses, the jobs tray).
SELECT sqlc.embed(j), coalesce(u.name, '')::text AS started_by_name, coalesce(su.name, '')::text AS stopped_by_name
FROM verification_jobs j
LEFT JOIN users u ON u.id = j.started_by
LEFT JOIN users su ON su.id = j.stopped_by
WHERE j.workspace_id = @workspace_id AND (j.created_at >= @since OR j.status IN ('queued', 'running', 'paused'))
ORDER BY j.created_at DESC
LIMIT @lim;

-- name: VerifJobsRunnable :many
-- Jobs to work on: queued or running, and paused ones whose retry is due.
SELECT j.* FROM verification_jobs j
WHERE j.status IN ('queued', 'running') OR (j.status = 'paused' AND j.next_try_at <= @now)
ORDER BY j.created_at
LIMIT 20;

-- name: VerifJobSetStatus :exec
UPDATE verification_jobs
SET status = @status, stop_reason = @stop_reason, last_error = @last_error,
    started_at = coalesce(started_at, sqlc.narg(started_at)), finished_at = sqlc.narg(finished_at),
    error_since = sqlc.narg(error_since), next_try_at = sqlc.narg(next_try_at), retry_until = sqlc.narg(retry_until)
WHERE id = @id;

-- name: VerifJobSaveCounts :exec
UPDATE verification_jobs
SET checked = @checked, valid = @valid, invalid = @invalid, risky = @risky, unknown = @unknown,
    suppressed = @suppressed, reasons = @reasons, balance_after = sqlc.narg(balance_after)
WHERE id = @id;

-- name: VerifJobCancel :execrows
UPDATE verification_jobs
SET status = 'cancelled', stop_reason = 'cancelled', stopped_by = @stopped_by, finished_at = @at, next_try_at = NULL
WHERE id = @id AND workspace_id = @workspace_id AND status IN ('queued', 'running', 'paused');

-- name: VerifJobResume :execrows
-- A job stopped for credits or an outage goes on with what's left.
UPDATE verification_jobs
SET status = 'queued', stop_reason = '', last_error = '', finished_at = NULL, error_since = NULL, next_try_at = NULL, retry_until = NULL
WHERE id = @id AND workspace_id = @workspace_id AND (status = 'stopped' OR status = 'paused');

-- name: VerifJobPending :many
SELECT email, contact_id FROM verification_job_items
WHERE job_id = @job_id AND NOT done
ORDER BY position
LIMIT @lim;

-- name: VerifJobPendingCount :one
SELECT count(*)::bigint FROM verification_job_items WHERE job_id = @job_id AND NOT done;

-- name: VerifJobItemsDone :exec
UPDATE verification_job_items i
SET done = true, result = x.result, reason = x.reason, code = x.code, checked_at = @checked_at
FROM (SELECT unnest(@emails::text[]) AS email, unnest(@results::text[]) AS result, unnest(@reasons::text[]) AS reason, unnest(@codes::text[]) AS code) AS x
WHERE i.job_id = @job_id AND i.email = x.email;

-- name: VerifJobItemsAll :many
-- A job's checked addresses, for the results CSV.
SELECT email, result, reason, code, checked_at FROM verification_job_items
WHERE job_id = @job_id AND done
ORDER BY position;

-- name: VerifJobResults :many
-- The results of an import's check, by address.
SELECT i.email, i.result, i.reason, i.code FROM verification_job_items i
WHERE i.job_id = @job_id AND i.done;

-- name: VerifLatestImportJob :one
SELECT j.* FROM verification_jobs j
WHERE j.workspace_id = @workspace_id AND j.import_id = @import_id AND j.kind = 'import' AND j.status <> 'cancelled'
ORDER BY j.created_at DESC
LIMIT 1;

-- name: VerifContactChecksUpsert :exec
INSERT INTO contact_address_checks (contact_id, workspace_id, result, reason, code, service, job_id, checked_at)
SELECT x.contact_id, @workspace_id, x.result, x.reason, x.code, @service, sqlc.narg(job_id), @checked_at
FROM (SELECT unnest(@contact_ids::uuid[]) AS contact_id, unnest(@results::text[]) AS result, unnest(@reasons::text[]) AS reason, unnest(@codes::text[]) AS code) AS x
ON CONFLICT (contact_id) DO UPDATE
SET result = excluded.result, reason = excluded.reason, code = excluded.code, service = excluded.service,
    job_id = excluded.job_id, checked_at = excluded.checked_at;

-- name: VerifContactCheck :one
SELECT c.* FROM contact_address_checks c WHERE c.contact_id = @contact_id AND c.workspace_id = @workspace_id;

-- name: VerifCheckedSince :many
-- Contacts whose address was checked since a time (skipped by "Skip
-- addresses checked in the last 90 days"), with the latest check's day.
SELECT c.contact_id, c.service, c.checked_at FROM contact_address_checks c
WHERE c.workspace_id = @workspace_id AND c.contact_id = ANY(@contact_ids::uuid[]) AND c.checked_at >= @since;

-- name: VerifChecksByEmail :many
-- The latest results of the contacts with these addresses (Send Readiness).
SELECT ct.email, c.result, c.reason, c.service, c.checked_at
FROM contact_address_checks c
JOIN contacts ct ON ct.id = c.contact_id
WHERE c.workspace_id = @workspace_id AND ct.email = ANY(@emails::text[]);

-- name: VerifContactsByEmail :many
SELECT id, email FROM contacts WHERE workspace_id = @workspace_id AND email = ANY(@emails::text[]) AND erased_at IS NULL;

-- name: VerifSetTags :exec
-- Replace a contact's verify:… tags with the latest result's (none for valid).
UPDATE contacts c
SET tags = (SELECT coalesce(array_agg(t ORDER BY t), '{}') FROM (
        SELECT t FROM unnest(c.tags) AS t WHERE t NOT LIKE 'verify:%'
        UNION SELECT x.tag WHERE x.tag <> '') s),
    updated_at = now()
FROM (SELECT unnest(@contact_ids::uuid[]) AS contact_id, unnest(@tags::text[]) AS tag) AS x
WHERE c.id = x.contact_id AND c.workspace_id = @workspace_id
  AND (x.tag = '' AND EXISTS (SELECT 1 FROM unnest(c.tags) t WHERE t LIKE 'verify:%') OR x.tag <> '' AND NOT (x.tag = ANY(c.tags)));

-- name: VerifSuppressContacts :many
-- Invalid addresses: the contacts are suppressed for all mail.
UPDATE contacts SET status = 'suppressed', updated_at = now()
WHERE workspace_id = @workspace_id AND id = ANY(@ids::uuid[]) AND erased_at IS NULL AND status = 'active'
RETURNING id;

-- name: VerifConsentEvents :exec
INSERT INTO consent_events (id, contact_id, action, source, note, actor_id, occurred_at)
SELECT x.id, x.contact_id, 'suppressed', 'verification', x.note, sqlc.narg(actor), @at
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@contact_ids::uuid[]) AS contact_id, unnest(@notes::text[]) AS note) AS x;

-- name: VerifSuppressedEmails :many
SELECT email FROM suppressions WHERE workspace_id = @workspace_id AND email = ANY(@emails::text[]);

-- name: VerifFormCheckSet :exec
INSERT INTO verification_form_checks (workspace_id, form_key, enabled, updated_by, updated_at)
VALUES (@workspace_id, @form_key, @enabled, @updated_by, @updated_at)
ON CONFLICT (workspace_id, form_key) DO UPDATE SET enabled = excluded.enabled, updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: VerifFormChecks :many
SELECT * FROM verification_form_checks WHERE workspace_id = @workspace_id ORDER BY form_key;

-- name: VerifFormCheckEnabled :one
SELECT coalesce((SELECT enabled FROM verification_form_checks WHERE workspace_id = @workspace_id AND form_key = @form_key), false)::boolean;

-- name: VerifRealtimeCount :exec
INSERT INTO verification_realtime_daily (workspace_id, service, form_key, day, checks, valid, invalid, risky, unknown, no_answer)
VALUES (@workspace_id, @service, @form_key, @day, 1, @valid, @invalid, @risky, @unknown, @no_answer)
ON CONFLICT (workspace_id, service, form_key, day) DO UPDATE
SET checks = verification_realtime_daily.checks + 1,
    valid = verification_realtime_daily.valid + excluded.valid,
    invalid = verification_realtime_daily.invalid + excluded.invalid,
    risky = verification_realtime_daily.risky + excluded.risky,
    unknown = verification_realtime_daily.unknown + excluded.unknown,
    no_answer = verification_realtime_daily.no_answer + excluded.no_answer;

-- name: VerifRealtimeDaily :many
SELECT * FROM verification_realtime_daily
WHERE workspace_id = @workspace_id AND service = @service AND day >= @since
ORDER BY day, form_key;

-- name: VerifRealtimeInvalidByForm :many
-- "Asked to fix their address" per form over a period.
SELECT form_key, coalesce(sum(invalid), 0)::bigint AS invalid FROM verification_realtime_daily
WHERE workspace_id = @workspace_id AND day >= @since
GROUP BY form_key;

-- name: VerifPendingInsert :exec
INSERT INTO verification_realtime_pending (id, workspace_id, email, form_key, created_at)
VALUES (@id, @workspace_id, @email, @form_key, @created_at)
ON CONFLICT (workspace_id, email) DO NOTHING;

-- name: VerifPendingDue :many
SELECT * FROM verification_realtime_pending ORDER BY created_at LIMIT @lim;

-- name: VerifPendingDelete :exec
DELETE FROM verification_realtime_pending WHERE id = @id;

-- name: VerifPendingTried :exec
UPDATE verification_realtime_pending SET tries = tries + 1 WHERE id = @id;

-- name: VerifPendingCount :one
SELECT count(*)::bigint FROM verification_realtime_pending WHERE workspace_id = @workspace_id;

-- name: VerifCheckCounts :one
-- Contacts per latest result, for a set of contacts.
SELECT count(*) FILTER (WHERE c.result = 'valid')::bigint AS valid,
       count(*) FILTER (WHERE c.result = 'invalid')::bigint AS invalid,
       count(*) FILTER (WHERE c.result = 'risky')::bigint AS risky,
       count(*) FILTER (WHERE c.result = 'unknown')::bigint AS unknown
FROM contact_address_checks c
WHERE c.workspace_id = @workspace_id AND c.contact_id = ANY(@contact_ids::uuid[]);

-- name: VerifTierCheckSummary :many
-- Per engagement tier: contacts, and those checked since a time (List
-- health's "never checked").
SELECT ct.tier, count(*)::bigint AS contacts, count(c.contact_id) FILTER (WHERE c.checked_at >= @since)::bigint AS checked,
       max(c.checked_at)::timestamptz AS last_checked
FROM contacts ct
LEFT JOIN contact_address_checks c ON c.contact_id = ct.id
WHERE ct.workspace_id = @workspace_id AND ct.erased_at IS NULL
GROUP BY ct.tier;

-- name: VerifServiceLowNotified :exec
UPDATE verification_services SET low_notified_at = sqlc.narg(at) WHERE id = @id;

-- name: VerifJobClaim :execrows
-- One worker at a time: the job is held until claimed_until.
UPDATE verification_jobs SET claimed_until = @until WHERE id = @id AND (claimed_until IS NULL OR claimed_until < @now);

-- name: VerifJobRelease :exec
UPDATE verification_jobs SET claimed_until = NULL WHERE id = @id;
