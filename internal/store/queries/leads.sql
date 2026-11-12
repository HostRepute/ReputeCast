-- Lead ads from Meta (M66, internal/leads).

-- name: LeadConnectionInsert :one
INSERT INTO lead_connections (id, workspace_id, platform, name, app_id, credentials_sealed, callback_token_hash, callback_token_sealed,
                              verify_token_sealed, status, checked_at, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, 'meta', @name, @app_id, @credentials_sealed, @callback_token_hash, @callback_token_sealed,
        @verify_token_sealed, 'connected', @at::timestamptz, @created_by, @at::timestamptz, @at::timestamptz)
RETURNING *;

-- name: LeadConnectionByID :one
SELECT * FROM lead_connections WHERE workspace_id = @workspace_id AND id = @id;

-- name: LeadConnectionAny :one
-- A connection by id alone (jobs).
SELECT * FROM lead_connections WHERE id = @id;

-- name: LeadConnectionByHash :one
-- The connection a callback URL's token belongs to (by its SHA-256).
SELECT * FROM lead_connections WHERE callback_token_hash = @hash;

-- name: LeadConnectionLive :one
-- The workspace's Meta connection that isn't disconnected.
SELECT * FROM lead_connections WHERE workspace_id = @workspace_id AND status <> 'disconnected' ORDER BY created_at DESC LIMIT 1;

-- name: LeadConnectionsOfWorkspace :many
SELECT c.*, coalesce(u.name, '')::text AS created_by_name
FROM lead_connections c
LEFT JOIN users u ON u.id = c.created_by
WHERE c.workspace_id = @workspace_id
ORDER BY (c.status = 'disconnected'), c.created_at DESC;

-- name: LeadConnectionsActive :many
-- Every connection still collecting (the health job).
SELECT * FROM lead_connections WHERE status <> 'disconnected' ORDER BY created_at;

-- name: LeadConnectionCreatorName :one
SELECT coalesce(u.name, '')::text FROM lead_connections c LEFT JOIN users u ON u.id = c.created_by WHERE c.id = @id;

-- name: LeadConnectionSetCredentials :exec
UPDATE lead_connections
SET name = @name, app_id = @app_id, credentials_sealed = @credentials_sealed, status = 'connected', status_detail = '',
    notified_at = NULL, checked_at = @at::timestamptz, updated_at = @at::timestamptz
WHERE id = @id;

-- name: LeadConnectionSetStatus :exec
UPDATE lead_connections SET status = @status, status_detail = @status_detail, updated_at = @at::timestamptz
WHERE id = @id AND status <> 'disconnected';

-- name: LeadConnectionChecked :exec
UPDATE lead_connections SET checked_at = @at::timestamptz WHERE id = @id;

-- name: LeadConnectionNotified :exec
UPDATE lead_connections SET notified_at = sqlc.narg(at) WHERE id = @id;

-- name: LeadConnectionSubscribed :exec
-- Meta accepted the app's webhook subscription (subscribed), or refused it
-- (NULL: leads are polled).
UPDATE lead_connections
SET subscribed_at = sqlc.narg(at),
    polling_reason = CASE WHEN sqlc.narg(at)::timestamptz IS NULL THEN 'not_subscribed' WHEN polling_reason = 'not_subscribed' THEN '' ELSE polling_reason END,
    updated_at = now()
WHERE id = @id;

-- name: LeadConnectionPolling :exec
UPDATE lead_connections SET polling_reason = @polling_reason WHERE id = @id;

-- name: LeadConnectionVerified :exec
-- Meta's verification request was answered.
UPDATE lead_connections SET verified_at = @at::timestamptz WHERE id = @id;

-- name: LeadConnectionTouchEvent :exec
-- A delivery with a valid signature: the webhook works.
UPDATE lead_connections
SET last_event_at = @at::timestamptz,
    polling_reason = CASE WHEN polling_reason IN ('quiet', 'unreachable') THEN '' ELSE polling_reason END
WHERE id = @id;

-- name: LeadConnectionRateLimited :exec
UPDATE lead_connections SET rate_limited_until = sqlc.narg(until) WHERE id = @id;

-- name: LeadConnectionDisconnect :exec
UPDATE lead_connections
SET status = 'disconnected', status_detail = '', credentials_sealed = NULL, callback_token_hash = NULL, callback_token_sealed = NULL,
    verify_token_sealed = NULL, disconnected_at = @at::timestamptz, disconnected_by = @by, updated_at = @at::timestamptz
WHERE id = @id;

-- name: LeadPageUpsert :one
INSERT INTO lead_pages (id, workspace_id, connection_id, page_id, name, page_token_sealed, chosen, subscribed_at, created_at, updated_at)
VALUES (@id, @workspace_id, @connection_id, @page_id, @name, @page_token_sealed, @chosen, sqlc.narg(subscribed_at), @at::timestamptz, @at::timestamptz)
ON CONFLICT (connection_id, page_id) DO UPDATE
SET name = EXCLUDED.name, page_token_sealed = EXCLUDED.page_token_sealed, chosen = EXCLUDED.chosen,
    subscribed_at = EXCLUDED.subscribed_at, updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: LeadPagesOfConnection :many
SELECT * FROM lead_pages WHERE connection_id = @connection_id ORDER BY name, page_id;

-- name: LeadPageByMetaID :one
SELECT * FROM lead_pages WHERE connection_id = @connection_id AND page_id = @page_id;

-- name: LeadPageWipeTokens :exec
UPDATE lead_pages SET page_token_sealed = NULL, subscribed_at = NULL, updated_at = @at::timestamptz WHERE connection_id = @connection_id;

-- name: LeadFormUpsert :one
-- A form as Meta lists it; its mapping and counters stay.
INSERT INTO lead_forms (id, workspace_id, page_id, form_id, name, meta_status, questions, checkboxes, disclaimer, created_at, updated_at)
VALUES (@id, @workspace_id, @page_id, @form_id, @name, @meta_status, @questions, @checkboxes, @disclaimer, @at::timestamptz, @at::timestamptz)
ON CONFLICT (page_id, form_id) DO UPDATE
SET name = EXCLUDED.name, meta_status = EXCLUDED.meta_status, questions = EXCLUDED.questions, checkboxes = EXCLUDED.checkboxes,
    disclaimer = EXCLUDED.disclaimer, updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: LeadFormsOfConnection :many
-- The connection's forms with their Page and leads in the last 30 days
-- (Meta's test leads left out).
SELECT f.*, p.name AS page_name, p.page_id AS meta_page_id, p.chosen,
       coalesce((SELECT count(*) FROM lead_entries e WHERE e.form_id = f.id AND e.outcome <> 'test' AND e.lead_at >= @since::timestamptz), 0)::bigint AS leads_30d
FROM lead_forms f
JOIN lead_pages p ON p.id = f.page_id
WHERE p.connection_id = @connection_id
ORDER BY p.name, f.created_at, f.name;

-- name: LeadFormByID :one
SELECT f.*, p.name AS page_name, p.page_id AS meta_page_id, p.chosen, p.page_token_sealed, p.connection_id
FROM lead_forms f
JOIN lead_pages p ON p.id = f.page_id
WHERE f.workspace_id = @workspace_id AND f.id = @id;

-- name: LeadFormAny :one
SELECT f.*, p.name AS page_name, p.page_id AS meta_page_id, p.chosen, p.page_token_sealed, p.connection_id
FROM lead_forms f
JOIN lead_pages p ON p.id = f.page_id
WHERE f.id = @id;

-- name: LeadFormByMetaID :one
SELECT * FROM lead_forms WHERE page_id = @page_id AND form_id = @form_id;

-- name: LeadFormSaveMapping :one
UPDATE lead_forms
SET mapping = @mapping, consent_mode = @consent_mode, consent_key = @consent_key, consent_text = @consent_text,
    consent_version = @consent_version, list_ids = @list_ids, tags = @tags,
    status = CASE WHEN mapped_at IS NULL THEN 'active' ELSE status END,
    mapped_at = @at::timestamptz, updated_at = @at::timestamptz
WHERE id = @id
RETURNING *;

-- name: LeadFormSetStatus :exec
UPDATE lead_forms SET status = @status, updated_at = @at::timestamptz WHERE id = @id;

-- name: LeadFormAddWaiting :exec
UPDATE lead_forms SET waiting = waiting + 1 WHERE id = @id;

-- name: LeadFormSetImport :exec
UPDATE lead_forms SET import_state = @import_state, import_count = @import_count, imported_at = sqlc.narg(imported_at), updated_at = now() WHERE id = @id;

-- name: LeadFormImportProgress :exec
UPDATE lead_forms SET import_count = @import_count WHERE id = @id;

-- name: LeadFormPolled :exec
UPDATE lead_forms SET last_polled_at = @at::timestamptz WHERE id = @id;

-- name: LeadFormLeadAt :exec
UPDATE lead_forms SET last_lead_at = greatest(coalesce(last_lead_at, @at::timestamptz), @at::timestamptz) WHERE id = @id;

-- name: LeadFormsCollecting :many
-- The mapped, active forms of a connection's chosen Pages (polling).
SELECT f.*, p.page_id AS meta_page_id, p.page_token_sealed
FROM lead_forms f
JOIN lead_pages p ON p.id = f.page_id
WHERE p.connection_id = @connection_id AND p.chosen AND f.status = 'active' AND f.mapped_at IS NOT NULL AND f.meta_status = 'ACTIVE'
ORDER BY f.created_at;

-- name: LeadFormNames :many
-- Every lead form of the workspace with its Page (growth sources, the
-- automation trigger), in the connection page's order.
SELECT f.id, f.name, p.name AS page_name, f.status, f.mapped_at
FROM lead_forms f
JOIN lead_pages p ON p.id = f.page_id
WHERE f.workspace_id = @workspace_id
ORDER BY p.name, f.created_at, f.name;

-- name: LeadEventInsert :one
INSERT INTO lead_events (id, workspace_id, connection_id, received_at, body, signature_ok, status)
VALUES (@id, @workspace_id, @connection_id, @received_at, @body, @signature_ok, @status)
RETURNING id;

-- name: LeadEventByID :one
SELECT * FROM lead_events WHERE id = @id;

-- name: LeadEventDone :exec
UPDATE lead_events SET status = @status, error = @error, applied_at = @at::timestamptz WHERE id = @id;

-- name: LeadEventsPending :many
-- Deliveries not applied two minutes after they arrived (a lost job).
SELECT id FROM lead_events WHERE status = 'pending' AND received_at < @before ORDER BY received_at LIMIT 200;

-- name: LeadEventsPurge :execrows
DELETE FROM lead_events WHERE received_at < @before;

-- name: LeadEntryExists :one
SELECT EXISTS (SELECT 1 FROM lead_entries WHERE form_id = @form_id AND leadgen_id = @leadgen_id);

-- name: LeadEntryInsert :execrows
INSERT INTO lead_entries (id, workspace_id, form_id, leadgen_id, contact_id, outcome, source, lead_at, created_at)
VALUES (@id, @workspace_id, @form_id, @leadgen_id, sqlc.narg(contact_id), @outcome, @source, @lead_at, @created_at)
ON CONFLICT (form_id, leadgen_id) DO NOTHING;

-- name: LeadEntriesRecent :many
-- A connection's last leads with their form, Page and contact.
SELECT e.id, e.leadgen_id, e.outcome, e.source, e.lead_at, e.created_at, e.contact_id,
       f.id AS form_id, f.name AS form_name, p.name AS page_name,
       coalesce(c.email, '')::text AS contact_email, coalesce(trim(c.first_name || ' ' || c.last_name), '')::text AS contact_name,
       (c.erased_at IS NOT NULL)::boolean AS contact_erased
FROM lead_entries e
JOIN lead_forms f ON f.id = e.form_id
JOIN lead_pages p ON p.id = f.page_id
LEFT JOIN contacts c ON c.id = e.contact_id
WHERE p.connection_id = @connection_id
ORDER BY e.created_at DESC, e.id DESC
LIMIT @lim;

-- name: LeadEntriesWaitingOf :many
-- A contact's double opt-in leads not confirmed yet, with their form's
-- lists (the confirmation starts their flows).
SELECT e.id, e.form_id, f.list_ids
FROM lead_entries e
JOIN lead_forms f ON f.id = e.form_id
WHERE e.contact_id = @contact_id AND e.outcome = 'pending' AND e.confirmed_at IS NULL;

-- name: LeadEntryConfirmed :exec
UPDATE lead_entries SET confirmed_at = @at::timestamptz WHERE id = @id;

-- name: LeadLeads30d :one
-- A connection's leads in the last 30 days (the catalog card).
SELECT count(*)::bigint
FROM lead_entries e
JOIN lead_forms f ON f.id = e.form_id
JOIN lead_pages p ON p.id = f.page_id
WHERE p.connection_id = @connection_id AND e.outcome <> 'test' AND e.lead_at >= @since::timestamptz;

-- name: LeadContactFillPhone :exec
-- A lead's phone number on a contact that has none.
UPDATE contacts SET phone = @phone, updated_at = @at::timestamptz WHERE id = @id AND phone = '';

-- name: LeadListsOfWorkspace :many
SELECT id, name, double_opt_in FROM lists WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY name;

-- name: LeadLeads30dOfForm :one
SELECT count(*)::bigint FROM lead_entries WHERE form_id = @form_id AND outcome <> 'test' AND lead_at >= @since::timestamptz;

-- name: LeadFormClearWaiting :exec
-- An import finished: the leads that waited for the mapping are in.
UPDATE lead_forms SET waiting = 0 WHERE id = @id;
