-- Placement tests (internal/placement): seed mailboxes, tests, copies.

-- name: PlacementSeeds :many
SELECT sqlc.embed(s), coalesce(u.name, '')::text AS added_by_name
FROM placement_seeds s
LEFT JOIN users u ON u.id = s.added_by
WHERE s.workspace_id = @workspace_id
ORDER BY s.added_at, s.id;

-- name: PlacementSeed :one
SELECT s.* FROM placement_seeds s WHERE s.id = @id AND s.workspace_id = @workspace_id;

-- name: PlacementSeedByEmail :one
SELECT s.* FROM placement_seeds s WHERE s.workspace_id = @workspace_id AND s.email = @email;

-- name: PlacementSeedInsert :exec
INSERT INTO placement_seeds (id, workspace_id, email, provider, auth, imap_host, imap_port, username, credentials_enc,
                             status, status_at, status_detail, folders, checked_at, added_by, added_at)
VALUES (@id, @workspace_id, @email, @provider, @auth, @imap_host, @imap_port, @username, @credentials_enc,
        @status, @status_at, @status_detail, @folders, @checked_at, @added_by, @added_at);

-- name: PlacementSeedReconnect :exec
UPDATE placement_seeds
SET auth = @auth, imap_host = @imap_host, imap_port = @imap_port, username = @username, credentials_enc = @credentials_enc,
    status = @status, status_at = @status_at, status_detail = @status_detail, folders = @folders, checked_at = @checked_at
WHERE id = @id;

-- name: PlacementSeedSetCredentials :exec
-- A refreshed OAuth token.
UPDATE placement_seeds SET credentials_enc = @credentials_enc WHERE id = @id;

-- name: PlacementSeedStatus :exec
UPDATE placement_seeds
SET status = @status, status_detail = @status_detail,
    status_at = CASE WHEN status = @status THEN status_at ELSE @at END,
    checked_at = @at
WHERE id = @id;

-- name: PlacementSeedFolders :exec
UPDATE placement_seeds SET folders = @folders WHERE id = @id;

-- name: PlacementSeedUsed :exec
UPDATE placement_seeds SET last_used_at = @at, last_test_id = @test_id, last_test_label = @label WHERE id = ANY(@ids::uuid[]);

-- name: PlacementSeedDelete :execrows
DELETE FROM placement_seeds WHERE id = @id AND workspace_id = @workspace_id;

-- name: PlacementSettings :one
SELECT coalesce((SELECT delete_after FROM placement_settings WHERE workspace_id = @workspace_id), '24h')::text;

-- name: PlacementSettingsSet :exec
INSERT INTO placement_settings (workspace_id, delete_after, updated_by, updated_at)
VALUES (@workspace_id, @delete_after, @updated_by, @updated_at)
ON CONFLICT (workspace_id) DO UPDATE SET delete_after = excluded.delete_after, updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: PlacementOAuthStateInsert :exec
INSERT INTO placement_oauth_states (state_hash, workspace_id, user_id, kind, provider, seed_id, verifier_enc, created_at, expires_at)
VALUES (@state_hash, @workspace_id, @user_id, @kind, @provider, @seed_id, @verifier_enc, @created_at, @expires_at);

-- name: PlacementOAuthStateTake :one
-- Single use: the state is removed as it's read.
DELETE FROM placement_oauth_states WHERE state_hash = @state_hash RETURNING *;

-- name: PlacementOAuthStatesExpire :exec
DELETE FROM placement_oauth_states WHERE expires_at < @now;

-- name: PlacementTestInsert :exec
INSERT INTO placement_tests (id, workspace_id, name, content_kind, content_id, content_name, content_note, content_key, content_version,
                             content_at, variant, language, personalization, through, stream, provider_ids, sender_id, from_name,
                             from_email, origin, status, subject, html, text_body, links, providers, track_opens, track_clicks,
                             started_by, started_at, wait_until, delete_after)
VALUES (@id, @workspace_id, @name, @content_kind, @content_id, @content_name, @content_note, @content_key, @content_version,
        @content_at, @variant, @language, @personalization, @through, @stream, @provider_ids, @sender_id, @from_name,
        @from_email, @origin, 'sending', @subject, @html, @text_body, @links, @providers, @track_opens, @track_clicks,
        @started_by, @started_at, @wait_until, @delete_after);

-- name: PlacementCopyInsert :exec
INSERT INTO placement_copies (id, test_id, workspace_id, seed_id, seed_email, seed_provider, position, provider_id, provider_name,
                              token, round, send_at, status, not_checked)
VALUES (@id, @test_id, @workspace_id, @seed_id, @seed_email, @seed_provider, @position, @provider_id, @provider_name,
        @token, @round, @send_at, @status, @not_checked);

-- name: PlacementTests :many
SELECT sqlc.embed(t), coalesce(u.name, '')::text AS started_by_name, coalesce(u.avatar_color, '')::text AS started_by_color
FROM placement_tests t
LEFT JOIN users u ON u.id = t.started_by
WHERE t.workspace_id = @workspace_id
ORDER BY t.started_at DESC
LIMIT @lim;

-- name: PlacementTest :one
SELECT sqlc.embed(t), coalesce(u.name, '')::text AS started_by_name, coalesce(u.avatar_color, '')::text AS started_by_color,
       coalesce(su.name, '')::text AS stopped_by_name,
       coalesce(du.name, '')::text AS copies_deleted_by_name
FROM placement_tests t
LEFT JOIN users u ON u.id = t.started_by
LEFT JOIN users su ON su.id = t.stopped_by
LEFT JOIN users du ON du.id = t.copies_deleted_by
WHERE t.id = @id AND t.workspace_id = @workspace_id;

-- name: PlacementTestByID :one
SELECT t.* FROM placement_tests t WHERE t.id = @id;

-- name: PlacementCopies :many
SELECT c.* FROM placement_copies c WHERE c.test_id = @test_id ORDER BY c.position, c.round, c.provider_name;

-- name: PlacementCopiesOfTests :many
SELECT c.* FROM placement_copies c WHERE c.test_id = ANY(@test_ids::uuid[]) ORDER BY c.test_id, c.position, c.round;

-- name: PlacementCopy :one
SELECT c.* FROM placement_copies c WHERE c.id = @id AND c.workspace_id = @workspace_id;

-- name: PlacementCopiesByTokens :many
-- The demo seeds' simulated mailboxes look copies up by token.
SELECT c.* FROM placement_copies c WHERE c.token = ANY(@tokens::text[]);

-- name: PlacementCopyByID :one
SELECT c.* FROM placement_copies c WHERE c.id = @id;

-- name: PlacementCopiesDue :many
-- Copies whose hand-off time has come.
SELECT c.* FROM placement_copies c
WHERE c.test_id = @test_id AND c.status = 'queued' AND c.send_at <= @now
ORDER BY c.send_at, c.position;

-- name: PlacementCopySent :exec
UPDATE placement_copies
SET status = @status, handed_at = @handed_at, provider_message_id = @provider_message_id, error = @error
WHERE id = @id;

-- name: PlacementCopyFound :exec
UPDATE placement_copies
SET status = 'found', folder = @folder, folder_name = @folder_name, tab = @tab, late = @late,
    received_at = @received_at, found_at = @found_at, auth = @auth, hop = @hop, headers = @headers, ref = @ref
WHERE id = @id;

-- name: PlacementCopiesNotArrived :exec
-- Copies still missing when the test finishes or is stopped (reason stopped).
UPDATE placement_copies SET status = 'not_arrived', not_checked = @reason WHERE test_id = @test_id AND status = 'sent';

-- name: PlacementCopiesUnsent :exec
-- Copies of a stopped comparison's later round that never went out.
UPDATE placement_copies SET status = 'not_checked', not_checked = 'unsent' WHERE test_id = @test_id AND status = 'queued';

-- name: PlacementCopiesNotChecked :exec
-- Copies of a seed that couldn't be read (its sign-in failed during the test).
UPDATE placement_copies SET status = 'not_checked', not_checked = @reason
WHERE test_id = @test_id AND seed_id = @seed_id AND status IN ('sent', 'not_arrived');

-- name: PlacementCopiesDeleted :exec
UPDATE placement_copies SET deleted_at = @at WHERE test_id = @test_id AND seed_id = @seed_id AND deleted_at IS NULL AND status = 'found';

-- name: PlacementTestsOpen :many
-- Tests sending or waiting, and finished ones still within the late window.
SELECT t.* FROM placement_tests t
WHERE t.status IN ('sending', 'waiting') OR (t.status = 'finished' AND t.late_until > @now)
ORDER BY t.started_at;

-- name: PlacementTestSetStatus :exec
UPDATE placement_tests
SET status = @status, last_sent_at = coalesce(sqlc.narg(last_sent_at), last_sent_at),
    wait_until = coalesce(sqlc.narg(wait_until), wait_until), late_until = coalesce(sqlc.narg(late_until), late_until),
    finished_at = coalesce(sqlc.narg(finished_at), finished_at), delete_at = coalesce(sqlc.narg(delete_at), delete_at)
WHERE id = @id;

-- name: PlacementTestStop :execrows
UPDATE placement_tests
SET status = 'stopped', stopped_by = @stopped_by, stopped_at = @at, finished_at = @at, late_until = @at,
    delete_at = sqlc.narg(delete_at)
WHERE id = @id AND workspace_id = @workspace_id AND status IN ('sending', 'waiting');

-- name: PlacementTestsToDelete :many
-- Tests whose copies are due to be deleted from the seeds.
SELECT t.* FROM placement_tests t
WHERE t.delete_at <= @now AND t.copies_deleted_at IS NULL AND t.status IN ('finished', 'stopped')
ORDER BY t.delete_at
LIMIT 10;

-- name: PlacementTestCopiesDeleted :exec
UPDATE placement_tests SET copies_deleted_at = @at, copies_deleted_by = sqlc.narg(by), delete_at = NULL WHERE id = @id;

-- name: PlacementLatestForCampaign :one
SELECT t.* FROM placement_tests t
WHERE t.workspace_id = @workspace_id AND t.content_kind = 'campaign' AND t.content_id = @content_id
ORDER BY t.started_at DESC
LIMIT 1;

-- name: PlacementTestsForProvider :many
-- Tests through one provider since a time (the warm-up planner's card).
SELECT t.* FROM placement_tests t
WHERE t.workspace_id = @workspace_id AND @provider_id::uuid = ANY(t.provider_ids) AND t.started_at >= @since
ORDER BY t.started_at;

-- name: PlacementCopyEvents :many
-- The provider's events for a copy.
SELECT e.* FROM delivery_events e WHERE e.message_id = @message_id ORDER BY e.occurred_at, e.id;

-- name: PlacementTestEvents :many
-- The provider's events for a test's copies.
SELECT e.* FROM delivery_events e WHERE e.message_id = ANY(@ids::uuid[]) ORDER BY e.message_id, e.occurred_at, e.id;

-- name: PlacementMessageKind :one
SELECT m.kind FROM messages m WHERE m.id = @id;

-- name: PlacementTemplate :one
SELECT t.id, t.name, t.html FROM templates t WHERE t.workspace_id = @workspace_id AND t.id = @id AND t.kind = 'template';

-- name: PlacementTransactional :one
SELECT t.id, t.key, t.name, t.live_version, coalesce(c.name, '')::text AS category_name
FROM transactional_templates t
LEFT JOIN transactional_categories c ON c.id = t.category_id
WHERE t.workspace_id = @workspace_id AND t.id = @id AND t.deleted_at IS NULL;

-- name: PlacementSourceLanguage :one
-- A campaign's source language: its own, or the workspace's default.
SELECT coalesce(
    (SELECT cs.language FROM content_sources cs WHERE cs.owner_kind = 'campaign' AND cs.owner_id = @owner_id AND cs.owner_version = 0),
    (SELECT wl.code FROM workspace_languages wl WHERE wl.workspace_id = @workspace_id AND wl.is_default AND wl.removed_at IS NULL),
    'en')::text;

-- name: PlacementCampaignLanguages :many
-- A campaign's translated language versions.
SELECT t.language FROM content_translations t
WHERE t.owner_kind = 'campaign' AND t.owner_id = @owner_id AND t.owner_version = 0 AND t.status = 'translated' AND NOT t.held
ORDER BY t.language;

-- name: PlacementCampaignTranslation :one
SELECT t.subject, t.html, t.plain_text FROM content_translations t
WHERE t.owner_kind = 'campaign' AND t.owner_id = @owner_id AND t.owner_version = 0 AND t.language = @language AND t.status = 'translated';

-- name: PlacementListNames :many
SELECT l.name FROM lists l WHERE l.workspace_id = @workspace_id AND l.id = ANY(@ids::uuid[]) ORDER BY l.name;

-- name: PlacementContentSearch :many
-- What a test can send, for the dialog's search: campaigns with content,
-- campaign templates and published transactional templates.
SELECT x.kind::text AS kind, x.id, x.name::text AS name, x.note::text AS note, x.at::timestamptz AS at FROM (
    SELECT 'campaign' AS kind, c.id, c.name, c.status AS note, coalesce(c.scheduled_at, c.started_at, c.updated_at) AS at
    FROM campaigns c WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.html <> ''
    UNION ALL
    SELECT 'template', t.id, t.name, 'template', t.updated_at
    FROM templates t WHERE t.workspace_id = @workspace_id AND t.kind = 'template' AND t.html <> ''
    UNION ALL
    SELECT 'transactional', tt.id, tt.name, tt.key, tt.updated_at
    FROM transactional_templates tt WHERE tt.workspace_id = @workspace_id AND tt.live_version IS NOT NULL AND tt.archived_at IS NULL AND tt.deleted_at IS NULL
) x
WHERE @q::text = '' OR x.name ILIKE '%' || @q::text || '%' OR x.note ILIKE '%' || @q::text || '%'
ORDER BY x.at DESC
LIMIT 20;
