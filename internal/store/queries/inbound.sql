-- Receiving mail (M58): inbound routes, what arrives, and the replies inbox.

-- name: InboundRouteGet :one
SELECT * FROM inbound_routes WHERE workspace_id = @workspace_id;

-- name: InboundRouteByID :one
SELECT * FROM inbound_routes WHERE id = @id;

-- name: InboundRouteByTokenHash :one
SELECT * FROM inbound_routes WHERE hook_token_hash = @hook_token_hash;

-- name: InboundRouteDomainTaken :one
-- Whether another workspace receives for the domain through a provider.
SELECT EXISTS (SELECT 1 FROM inbound_routes WHERE kind = 'provider' AND lower(domain) = lower(@domain::text) AND workspace_id <> @workspace_id);

-- name: InboundRouteUpsert :one
INSERT INTO inbound_routes (id, workspace_id, kind, provider_id, domain, mailbox_local, mailbox_settings, mailbox_password_sealed, settings,
    secret_sealed, hook_token_hash, hook_token_sealed, status, status_detail, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @kind, @provider_id, @domain, @mailbox_local, @mailbox_settings, @mailbox_password_sealed, @settings,
    @secret_sealed, @hook_token_hash, @hook_token_sealed, 'pending', '', @created_by, @now, @now)
ON CONFLICT (workspace_id) DO UPDATE SET kind = EXCLUDED.kind, provider_id = EXCLUDED.provider_id, domain = EXCLUDED.domain,
    mailbox_local = EXCLUDED.mailbox_local, mailbox_settings = EXCLUDED.mailbox_settings, mailbox_password_sealed = EXCLUDED.mailbox_password_sealed,
    settings = EXCLUDED.settings,
    -- A route that keeps its kind, provider and domain keeps receiving;
    -- another one starts over.
    status = CASE WHEN inbound_routes.kind = EXCLUDED.kind AND inbound_routes.provider_id IS NOT DISTINCT FROM EXCLUDED.provider_id
                       AND lower(inbound_routes.domain) = lower(EXCLUDED.domain) AND inbound_routes.mailbox_local = EXCLUDED.mailbox_local
                  THEN inbound_routes.status ELSE 'pending' END,
    status_detail = '', registered_at = NULL, updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: InboundRouteDelete :one
DELETE FROM inbound_routes WHERE workspace_id = @workspace_id RETURNING *;

-- name: InboundRouteRegistered :exec
UPDATE inbound_routes SET registered_at = sqlc.narg(registered_at), status_detail = @status_detail, updated_at = @now WHERE id = @id;

-- name: InboundRouteFailing :exec
UPDATE inbound_routes SET status = 'failing', status_detail = @status_detail, updated_at = @now WHERE id = @id;

-- name: InboundRouteReceived :exec
-- A message of any kind arrived: the route receives.
UPDATE inbound_routes SET status = 'receiving', status_detail = @status_detail, verified_at = coalesce(verified_at, @at::timestamptz),
    last_received_at = @at::timestamptz, failing_notified_at = NULL
WHERE id = @id;

-- name: InboundRoutePolled :exec
UPDATE inbound_routes SET last_polled_at = @at::timestamptz,
    status = CASE WHEN status = 'failing' THEN (CASE WHEN verified_at IS NULL THEN 'pending' ELSE 'receiving' END) ELSE status END,
    status_detail = CASE WHEN status = 'failing' THEN '' ELSE status_detail END
WHERE id = @id;

-- name: InboundRouteTestSent :exec
UPDATE inbound_routes SET test_sent_at = @at::timestamptz, test_received_at = NULL WHERE id = @id;

-- name: InboundRouteTestReceived :exec
UPDATE inbound_routes SET test_received_at = @at::timestamptz WHERE id = @id AND test_sent_at IS NOT NULL;

-- name: InboundRoutesMailbox :many
SELECT * FROM inbound_routes WHERE kind = 'mailbox' ORDER BY id;

-- name: InboundRouteFailingFor :exec
-- A provider route whose MX moved: failing, unless it received since the
-- check began.
UPDATE inbound_routes SET status = 'failing', status_detail = @status_detail, updated_at = @now
WHERE id = @id AND (last_received_at IS NULL OR last_received_at < @since::timestamptz);

-- name: InboundRoutesProvider :many
SELECT * FROM inbound_routes WHERE kind = 'provider' AND provider_id IS NOT NULL ORDER BY id;

-- name: InboundRouteMarkFailingNotified :one
-- Claims the one notice a failing route sends until it receives again.
UPDATE inbound_routes SET failing_notified_at = @at::timestamptz WHERE id = @id AND failing_notified_at IS NULL RETURNING id;

-- name: InboundRoutesWithFreshReplies :many
-- Routes that received human replies since they last told anyone, and
-- last told nobody since the cutoff (15 minutes ago).
SELECT r.id, r.workspace_id, r.notified_at, r.created_at FROM inbound_routes r
WHERE (r.notified_at IS NULL OR r.notified_at <= @cutoff::timestamptz)
  AND EXISTS (SELECT 1 FROM inbound_messages m WHERE m.workspace_id = r.workspace_id AND m.kind = 'reply' AND NOT m.auto_reply AND NOT m.spam
              AND m.status = 'open' AND m.received_at > coalesce(r.notified_at, r.created_at));

-- name: InboundFreshReplies :one
SELECT count(*)::bigint FROM inbound_messages
WHERE workspace_id = @workspace_id AND kind = 'reply' AND NOT auto_reply AND NOT spam AND status = 'open' AND received_at > @since;

-- name: InboundRouteClaimNotice :one
-- Claims a replies notice: only the process that moves notified_at sends it.
UPDATE inbound_routes SET notified_at = @now::timestamptz WHERE id = @id AND notified_at IS NOT DISTINCT FROM sqlc.narg(prev)::timestamptz RETURNING id;

-- name: InboundRawInsert :exec
INSERT INTO inbound_raw (id, workspace_id, route_id, source, received_at, content_type, headers, body, signature_ok)
VALUES (@id, @workspace_id, @route_id, @source, @received_at, @content_type, @headers, @body, @signature_ok);

-- name: InboundRawGet :one
SELECT * FROM inbound_raw WHERE id = @id;

-- name: InboundRawProcessed :exec
UPDATE inbound_raw SET processed_at = @at, process_error = @process_error WHERE id = @id;

-- name: InboundRawStale :many
-- Verified rows still waiting after a while: their job was lost.
SELECT id FROM inbound_raw WHERE processed_at IS NULL AND signature_ok AND received_at < @before ORDER BY received_at LIMIT 1000;

-- name: InboundRawPurge :execrows
DELETE FROM inbound_raw WHERE received_at < @before;

-- name: InboundMessagesPurge :execrows
DELETE FROM inbound_messages WHERE received_at < @before;

-- name: InboundMessageInsert :one
INSERT INTO inbound_messages (id, workspace_id, route_id, raw_id, kind, message_id, contact_id, campaign_id, automation_id, from_email, from_name,
    to_address, subject, text_body, has_html, attachments, auto_reply, spam, asks_unsubscribe, auth, status, header_message_id, received_at)
VALUES (@id, @workspace_id, @route_id, @raw_id, @kind, @message_id, @contact_id, @campaign_id, @automation_id, @from_email, @from_name,
    @to_address, @subject, @text_body, @has_html, @attachments, @auto_reply, @spam, @asks_unsubscribe, @auth, @status, @header_message_id, @received_at)
ON CONFLICT (workspace_id, header_message_id) WHERE header_message_id <> '' DO NOTHING
RETURNING id;

-- name: InboundOurMessage :one
-- The message of ours a tagged address names, in the workspace.
SELECT m.id, m.workspace_id, m.stream, m.kind, m.campaign_id, m.automation_id, m.automation_node, m.contact_id, m.email, m.provider_id,
    m.mailbox_provider, m.sent_at, m.queued_at
FROM messages m WHERE m.id = @id AND m.workspace_id = @workspace_id;

-- name: InboundMessageByHeader :one
-- A message of ours by the Message-ID it went out with (<uuid@domain>).
SELECT m.id, m.provider_id, m.email, m.contact_id, m.campaign_id FROM messages m WHERE m.id = @id AND m.workspace_id = @workspace_id;

-- name: RepliesList :many
-- One view of the replies inbox, newest first, keyset-paged.
SELECT im.id, im.kind, im.from_email, im.from_name, im.subject, left(im.text_body, 400)::text AS preview, im.received_at, im.status,
    im.auto_reply, im.spam, im.asks_unsubscribe, im.contact_id, im.campaign_id, im.automation_id,
    coalesce(c.name, '')::text AS campaign_name, coalesce(a.name, '')::text AS automation_name
FROM inbound_messages im
LEFT JOIN campaigns c ON c.id = im.campaign_id
LEFT JOIN automations a ON a.id = im.automation_id
WHERE im.workspace_id = @workspace_id
  AND CASE @view::text
        WHEN 'open' THEN im.kind = 'reply' AND NOT im.auto_reply AND NOT im.spam AND im.status = 'open'
        WHEN 'done' THEN im.kind = 'reply' AND NOT im.auto_reply AND NOT im.spam AND im.status = 'done'
        WHEN 'auto' THEN im.kind = 'reply' AND im.auto_reply AND NOT im.spam
        WHEN 'spam' THEN im.spam AND im.kind IN ('reply', 'other')
        WHEN 'other' THEN im.kind = 'other' AND NOT im.spam
        WHEN 'unsubscribes' THEN im.kind = 'unsubscribe'
        ELSE false END
  AND (sqlc.narg(campaign_id)::uuid IS NULL OR im.campaign_id = sqlc.narg(campaign_id)::uuid)
  AND (sqlc.narg(automation_id)::uuid IS NULL OR im.automation_id = sqlc.narg(automation_id)::uuid)
  AND (@q::text = '' OR (im.subject || ' ' || im.from_email || ' ' || im.from_name) ILIKE '%' || @q::text || '%' OR im.text_body ILIKE '%' || @q::text || '%')
  AND (sqlc.narg(before_at)::timestamptz IS NULL OR (im.received_at, im.id) < (sqlc.narg(before_at)::timestamptz, @before_id::uuid))
ORDER BY im.received_at DESC, im.id DESC
LIMIT @lim;

-- name: RepliesCounts :one
SELECT
    count(*) FILTER (WHERE kind = 'reply' AND NOT auto_reply AND NOT spam AND status = 'open')::bigint AS open,
    count(*) FILTER (WHERE kind = 'reply' AND NOT auto_reply AND NOT spam AND status = 'done')::bigint AS done,
    count(*) FILTER (WHERE kind = 'reply' AND auto_reply AND NOT spam)::bigint AS auto,
    count(*) FILTER (WHERE spam AND kind IN ('reply', 'other'))::bigint AS spam,
    count(*) FILTER (WHERE kind = 'other' AND NOT spam)::bigint AS other,
    count(*) FILTER (WHERE kind = 'unsubscribe')::bigint AS unsubscribes
FROM inbound_messages im
WHERE im.workspace_id = @workspace_id AND im.kind IN ('reply', 'other', 'unsubscribe')
  AND (sqlc.narg(campaign_id)::uuid IS NULL OR im.campaign_id = sqlc.narg(campaign_id)::uuid)
  AND (sqlc.narg(automation_id)::uuid IS NULL OR im.automation_id = sqlc.narg(automation_id)::uuid)
  AND (@q::text = '' OR (im.subject || ' ' || im.from_email || ' ' || im.from_name) ILIKE '%' || @q::text || '%' OR im.text_body ILIKE '%' || @q::text || '%');

-- name: ReplyGet :one
SELECT im.*, coalesce(c.name, '')::text AS campaign_name, coalesce(c.subject, '')::text AS campaign_subject, coalesce(a.name, '')::text AS automation_name,
    m.sent_at AS message_sent_at,
    coalesce(ct.email, '')::text AS contact_email, coalesce(ct.first_name, '')::text AS contact_first_name,
    coalesce(ct.last_name, '')::text AS contact_last_name, coalesce(ct.status, '')::text AS contact_status
FROM inbound_messages im
LEFT JOIN campaigns c ON c.id = im.campaign_id
LEFT JOIN automations a ON a.id = im.automation_id
LEFT JOIN messages m ON m.id = im.message_id
LEFT JOIN contacts ct ON ct.id = im.contact_id AND ct.erased_at IS NULL
WHERE im.id = @id AND im.workspace_id = @workspace_id;

-- name: RepliesSetStatus :many
UPDATE inbound_messages SET status = @status,
    handled_by = CASE WHEN @status::text = 'done' THEN sqlc.narg(handled_by)::uuid ELSE NULL END,
    handled_at = CASE WHEN @status::text = 'done' THEN @at::timestamptz ELSE NULL END
WHERE workspace_id = @workspace_id AND id = ANY (@ids::uuid[]) AND status <> @status
RETURNING id;

-- name: InboundContactActivity :many
-- A contact's replies for the timeline, newest first.
SELECT im.id, im.subject, left(im.text_body, 600)::text AS text_start, im.received_at, coalesce(c.name, '')::text AS campaign_name,
    coalesce(a.name, '')::text AS automation_name
FROM inbound_messages im
LEFT JOIN campaigns c ON c.id = im.campaign_id
LEFT JOIN automations a ON a.id = im.automation_id
WHERE im.workspace_id = @workspace_id AND im.contact_id = @contact_id AND im.kind = 'reply' AND NOT im.auto_reply AND NOT im.spam
  AND (sqlc.narg(before_at)::timestamptz IS NULL OR (im.received_at, im.id) < (sqlc.narg(before_at)::timestamptz, @before_id::uuid))
ORDER BY im.received_at DESC, im.id DESC
LIMIT @lim;

-- name: InboundContactActivityCount :one
SELECT count(*)::bigint FROM inbound_messages
WHERE workspace_id = @workspace_id AND contact_id = @contact_id AND kind = 'reply' AND NOT auto_reply AND NOT spam;

-- name: InboundEraseContactRaw :execrows
-- Erasure: what arrived for a contact's messages (by contact or address).
DELETE FROM inbound_raw r WHERE r.id IN (
    SELECT im.raw_id FROM inbound_messages im
    WHERE im.workspace_id = @workspace_id AND im.raw_id IS NOT NULL AND (im.contact_id = @contact_id OR lower(im.from_email) = lower(@email::text)));

-- name: InboundEraseContact :execrows
DELETE FROM inbound_messages im WHERE im.workspace_id = @workspace_id AND (im.contact_id = @contact_id OR lower(im.from_email) = lower(@email::text));

-- name: InboundExportContact :many
SELECT received_at, subject, text_body FROM inbound_messages
WHERE workspace_id = @workspace_id AND (contact_id = @contact_id OR lower(from_email) = lower(@email::text))
ORDER BY received_at;

-- name: InboundContactCount :one
SELECT count(*)::bigint FROM inbound_messages WHERE workspace_id = @workspace_id AND (contact_id = @contact_id OR lower(from_email) = lower(@email::text));

-- name: InboundLastDMARC :one
SELECT received_at FROM dmarc_reports WHERE workspace_id = @workspace_id AND via = 'inbound' ORDER BY received_at DESC LIMIT 1;

-- name: InboundRepliesInbox :one
-- Whether replies reach ReputeCast's inbox: the workspace has a receiving
-- route and keeps replies in it (stop on reply needs both).
SELECT EXISTS (SELECT 1 FROM inbound_routes r JOIN workspaces w ON w.id = r.workspace_id
               WHERE r.workspace_id = @workspace_id AND w.sending_defaults ->> 'reply_handling' = 'inbox');
