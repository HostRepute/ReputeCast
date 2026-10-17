-- Importing paid members (internal/earn): an export matched with the
-- connected account's customers and subscriptions.

-- name: EarnImportInsert :exec
INSERT INTO earn_imports (id, workspace_id, mode, source, status, file_name, file_size, columns, mapping, total, created_by, created_at)
VALUES (@id, @workspace_id, @mode, @source, 'uploaded', @file_name, @file_size, @columns, @mapping, @total, sqlc.narg(created_by), @created_at);

-- name: EarnImportFileInsert :exec
INSERT INTO earn_import_files (import_id, data) VALUES (@import_id, @data);

-- name: EarnImportFile :one
SELECT data FROM earn_import_files WHERE import_id = @import_id;

-- name: EarnImportFileDelete :exec
DELETE FROM earn_import_files WHERE import_id = @import_id;

-- name: EarnImportGet :one
SELECT i.*, coalesce(u.name, '')::text AS created_by_name
FROM earn_imports i LEFT JOIN users u ON u.id = i.created_by
WHERE i.id = @id AND i.workspace_id = @workspace_id;

-- name: EarnImportByID :one
SELECT * FROM earn_imports WHERE id = @id;

-- name: EarnImportSetColumns :exec
UPDATE earn_imports SET columns = @columns, mapping = @mapping, total = @total, status = 'uploaded', matched_at = NULL WHERE id = @id;

-- name: EarnImportSetDecisions :exec
UPDATE earn_imports SET decisions = @decisions, send_welcome = @send_welcome WHERE id = @id;

-- name: EarnImportMatched :exec
UPDATE earn_imports SET status = 'matched', matched_at = @at, stats = @stats WHERE id = @id;

-- name: EarnImportStart :execrows
UPDATE earn_imports SET status = 'importing', started_at = @at, done_rows = 0, error = '' WHERE id = @id AND status = 'matched';

-- name: EarnImportProgress :exec
UPDATE earn_imports SET done_rows = @done_rows WHERE id = @id;

-- name: EarnImportFinish :exec
UPDATE earn_imports SET status = @status, finished_at = @at, stats = @stats, error = @error, done_rows = @done_rows WHERE id = @id;

-- name: EarnImportCancel :execrows
UPDATE earn_imports SET status = 'canceled', finished_at = @at WHERE id = @id AND workspace_id = @workspace_id AND status IN ('uploaded','matched');

-- name: EarnImportRowsClear :exec
DELETE FROM earn_import_rows WHERE import_id = @import_id;

-- name: EarnImportRowInsert :copyfrom
INSERT INTO earn_import_rows (import_id, row_no, email, name, customer_id, plan, paid_through) VALUES (@import_id, @row_no, @email, @name, @customer_id, @plan, @paid_through);

-- name: EarnImportRows :many
SELECT * FROM earn_import_rows WHERE import_id = @import_id AND row_no > @after ORDER BY row_no LIMIT @lim;

-- name: EarnImportRowMatch :exec
UPDATE earn_import_rows SET match = @match, matched_by = @matched_by, stripe_customer_id = @stripe_customer_id, stripe_subscription_id = @stripe_subscription_id,
    stripe_price_id = @stripe_price_id, subscription = @subscription, candidates = @candidates
WHERE import_id = @import_id AND row_no = @row_no;

-- name: EarnImportRowResult :exec
UPDATE earn_import_rows SET result = @result, contact_id = sqlc.narg(contact_id) WHERE import_id = @import_id AND row_no = @row_no;

-- name: EarnImportExamples :many
-- A few rows of a group, for the match step's cards.
SELECT * FROM earn_import_rows WHERE import_id = @import_id AND match = @match ORDER BY row_no LIMIT @lim;

-- name: EarnImportConflicts :many
SELECT * FROM earn_import_rows WHERE import_id = @import_id AND match = 'conflict' ORDER BY row_no;

-- name: EarnImportGroupCounts :many
SELECT match, matched_by, count(*)::bigint AS n FROM earn_import_rows WHERE import_id = @import_id GROUP BY match, matched_by;

-- name: EarnImportPriceCounts :many
SELECT stripe_price_id, count(*)::bigint AS n, min(subscription->>'interval')::text AS interval, min((subscription->>'amount')::bigint)::bigint AS amount,
    min(subscription->>'currency')::text AS currency
FROM earn_import_rows WHERE import_id = @import_id AND match = 'active' AND stripe_price_id <> ''
GROUP BY stripe_price_id ORDER BY n DESC;

-- name: EarnImportsRunning :many
SELECT * FROM earn_imports WHERE workspace_id = @workspace_id AND status = 'importing' ORDER BY created_at;

-- name: EarnImportLatest :one
SELECT i.*, coalesce(u.name, '')::text AS created_by_name
FROM earn_imports i LEFT JOIN users u ON u.id = i.created_by
WHERE i.workspace_id = @workspace_id ORDER BY i.created_at DESC LIMIT 1;

-- name: EarnContactUpsertImport :one
-- A paid member from an import: a new contact, or an existing one merged
-- (its fields kept, its empty name filled).
INSERT INTO contacts (id, workspace_id, email, first_name, last_name, mailbox_provider, consent, source, role_address, created_at, updated_at, last_activity_at)
VALUES (@id, @workspace_id, @email, @first_name, @last_name, @mailbox_provider, 'single', @source, @role_address, @at, @at, @at)
ON CONFLICT (workspace_id, email) DO UPDATE SET
    first_name = CASE WHEN contacts.first_name = '' THEN EXCLUDED.first_name ELSE contacts.first_name END,
    last_name = CASE WHEN contacts.last_name = '' THEN EXCLUDED.last_name ELSE contacts.last_name END,
    updated_at = EXCLUDED.updated_at
RETURNING id, (xmax = 0)::boolean AS created, status;

-- name: EarnImportMembership :exec
-- Joins a list (a list the contact unsubscribed from stays unsubscribed).
INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at, created_at)
VALUES (@list_id, @contact_id, 'subscribed', @source, @at, @at)
ON CONFLICT (list_id, contact_id) DO UPDATE SET
    status = CASE WHEN list_memberships.status = 'pending' THEN 'subscribed' ELSE list_memberships.status END,
    subscribed_at = CASE WHEN list_memberships.status = 'pending' THEN EXCLUDED.subscribed_at ELSE list_memberships.subscribed_at END;

-- name: EarnImportConsent :exec
INSERT INTO consent_events (id, contact_id, list_id, action, source, detail, note, occurred_at)
VALUES (@id, @contact_id, @list_id, 'imported', @source, @detail, @note, @at);
