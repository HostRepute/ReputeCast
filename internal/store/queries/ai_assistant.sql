-- The AI assistant (B3): its settings, threads and their attached context.
-- A thread is read only through its owner: every query that returns one
-- names the person.

-- name: AIAssistantSettings :one
SELECT * FROM ai_assistant_settings WHERE workspace_id = @workspace_id;

-- name: AIAssistantSettingsSave :exec
INSERT INTO ai_assistant_settings (workspace_id, enabled, tool_limit, token_limit, cost_limit_micros, per_person, per_workspace, updated_by, updated_at)
VALUES (@workspace_id, @enabled, @tool_limit, @token_limit, @cost_limit_micros, @per_person, @per_workspace, sqlc.narg(updated_by), @updated_at)
ON CONFLICT (workspace_id) DO UPDATE SET enabled = EXCLUDED.enabled, tool_limit = EXCLUDED.tool_limit, token_limit = EXCLUDED.token_limit,
    cost_limit_micros = EXCLUDED.cost_limit_micros, per_person = EXCLUDED.per_person, per_workspace = EXCLUDED.per_workspace,
    revision = ai_assistant_settings.revision + 1, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: AIAssistantThreadInsert :exec
INSERT INTO ai_assistant_threads (id, workspace_id, user_id, model_id, title_sealed, screen, tool_limit, token_limit, cost_limit_micros,
    request_key, created_at, updated_at, last_activity_at, expires_at)
VALUES (@id, @workspace_id, @user_id, sqlc.narg(model_id), sqlc.narg(title_sealed), @screen, @tool_limit, @token_limit, @cost_limit_micros,
    sqlc.narg(request_key), @now, @now, @now, @expires_at);

-- name: AIAssistantThread :one
-- One of the person's threads; another person's isn't there, a manager's
-- view included.
SELECT * FROM ai_assistant_threads WHERE workspace_id = @workspace_id AND user_id = @user_id AND id = @id;

-- name: AIAssistantThreadLock :one
SELECT * FROM ai_assistant_threads WHERE workspace_id = @workspace_id AND user_id = @user_id AND id = @id FOR UPDATE;

-- name: AIAssistantThreadByRequest :one
SELECT * FROM ai_assistant_threads
WHERE workspace_id = @workspace_id AND user_id = @user_id AND request_key = @request_key;

-- name: AIAssistantThreads :many
-- The person's threads, the most recently active first.
SELECT * FROM ai_assistant_threads
WHERE workspace_id = @workspace_id AND user_id = @user_id
  AND (sqlc.narg(before)::timestamptz IS NULL OR (last_activity_at, id) < (sqlc.narg(before)::timestamptz, sqlc.narg(before_id)::uuid))
ORDER BY last_activity_at DESC, id DESC
LIMIT @lim;

-- name: AIAssistantThreadSave :exec
-- Saves what the person changed. Only an open thread changes.
UPDATE ai_assistant_threads SET title_sealed = sqlc.narg(title_sealed), model_id = sqlc.narg(model_id), updated_at = @now,
    last_activity_at = @now, expires_at = @expires_at
WHERE workspace_id = @workspace_id AND id = @id AND state = 'open';

-- name: AIAssistantThreadDelete :execrows
-- Deleting a thread removes its content at once: messages, contexts and
-- stream events go with it. The usage ledger stays (without the thread).
DELETE FROM ai_assistant_threads WHERE workspace_id = @workspace_id AND user_id = @user_id AND id = @id;

-- name: AIAssistantContextInsert :execrows
INSERT INTO ai_assistant_contexts (id, thread_id, kind, target_id, revision, snapshot_sealed, shared_at)
VALUES (@id, @thread_id, @kind, @target_id, @revision, sqlc.narg(snapshot_sealed), @shared_at)
ON CONFLICT (thread_id, kind, target_id) DO NOTHING;

-- name: AIAssistantContexts :many
SELECT * FROM ai_assistant_contexts WHERE thread_id = @thread_id ORDER BY shared_at, id;

-- name: AIAssistantContextDelete :one
DELETE FROM ai_assistant_contexts WHERE thread_id = @thread_id AND id = @id RETURNING kind, target_id;

-- name: AIAssistantThreadLockByID :one
-- A thread as its turns' worker and the sweep lock it: the owner's lock,
-- without naming the owner (the worker is nobody's session).
SELECT * FROM ai_assistant_threads WHERE workspace_id = @workspace_id AND id = @id FOR UPDATE;

-- name: AIAssistantEventSeq :one
-- The next stream event's number. Only under the thread's lock: events are
-- numbered without gaps, in commit order.
UPDATE ai_assistant_threads SET event_seq = event_seq + 1 WHERE id = @id RETURNING event_seq;

-- name: AIAssistantEventInsert :exec
INSERT INTO ai_assistant_events (thread_id, seq, turn_id, kind, payload_sealed, created_at)
VALUES (@thread_id, @seq, sqlc.narg(turn_id), @kind, @payload_sealed, @created_at);

-- name: AIAssistantNotify :exec
-- Wakes the streams of a thread when the transaction commits. The payload
-- is "thread:seq" only: readers read the events themselves.
SELECT pg_notify('ai_assistant', @payload::text);

-- name: AIAssistantEvents :many
-- A thread's events after a stream's cursor, in order.
SELECT seq, turn_id, kind, payload_sealed FROM ai_assistant_events
WHERE thread_id = @thread_id AND seq > @after ORDER BY seq LIMIT @lim;

-- name: AIAssistantTurnDeltas :many
-- What a turn wrote up to an event: its answers' pieces and the messages
-- that saved them.
SELECT seq, kind, payload_sealed FROM ai_assistant_events
WHERE thread_id = @thread_id AND turn_id = @turn_id AND kind IN ('delta', 'message') AND seq <= @upto ORDER BY seq;

-- name: AIAssistantMessages :many
SELECT * FROM ai_assistant_messages WHERE thread_id = @thread_id ORDER BY sequence;

-- name: AIAssistantMessageInsert :exec
INSERT INTO ai_assistant_messages (id, thread_id, turn_id, sequence, role, model_id, content_sealed, bytes, created_at)
SELECT @id, @thread_id, sqlc.narg(turn_id), coalesce(max(sequence), 0) + 1, @role, sqlc.narg(model_id), @content_sealed, @bytes, @created_at
FROM ai_assistant_messages WHERE thread_id = @thread_id;

-- name: AIAssistantMessageOfTurn :one
-- The person's message a turn answers.
SELECT * FROM ai_assistant_messages WHERE turn_id = @turn_id AND role = 'user';

-- name: AIAssistantTurnInsert :one
INSERT INTO ai_assistant_turns (id, thread_id, sequence, model_id, state, reason, request_key, created_at, ended_at)
SELECT @id, @thread_id, coalesce(max(sequence), 0) + 1, sqlc.narg(model_id), @state, @reason, @request_key, @created_at, sqlc.narg(ended_at)
FROM ai_assistant_turns WHERE thread_id = @thread_id
RETURNING *;

-- name: AIAssistantTurnByRequest :one
SELECT * FROM ai_assistant_turns WHERE thread_id = @thread_id AND request_key = @request_key;

-- name: AIAssistantTurnLock :one
SELECT * FROM ai_assistant_turns WHERE thread_id = @thread_id AND id = @id FOR UPDATE;

-- name: AIAssistantTurnFence :one
-- A turn's version and state, read under its thread's lock.
SELECT version, state FROM ai_assistant_turns WHERE id = @id;

-- name: AIAssistantThreadActiveTurn :one
-- The thread's turn that is queued, answering or waiting, if any.
SELECT * FROM ai_assistant_turns WHERE thread_id = @thread_id AND state IN ('queued', 'answering', 'waiting');

-- name: AIAssistantThreadLastTurn :one
SELECT * FROM ai_assistant_turns WHERE thread_id = @thread_id ORDER BY sequence DESC LIMIT 1;

-- name: AIAssistantTurnSave :exec
UPDATE ai_assistant_turns SET state = @state, reason = @reason, version = @version, started_at = sqlc.narg(started_at), ended_at = sqlc.narg(ended_at)
WHERE id = @id;

-- name: AIAssistantStepInsert :exec
INSERT INTO ai_assistant_steps (id, turn_id, sequence, turn_version, kind, state, created_at, updated_at)
SELECT @id, @turn_id, coalesce(max(sequence), 0) + 1, @turn_version, @kind, 'requested', @now, @now
FROM ai_assistant_steps WHERE turn_id = @turn_id;

-- name: AIAssistantStepLast :one
SELECT * FROM ai_assistant_steps WHERE turn_id = @turn_id ORDER BY sequence DESC LIMIT 1;

-- name: AIAssistantStepSave :exec
UPDATE ai_assistant_steps SET state = @state, updated_at = @now WHERE id = @id;

-- name: AIAssistantSlotsExpire :exec
-- Frees the workspace's slots whose lease lapsed. Their turns are the
-- sweep's: answering without a live slot is a worker that died.
DELETE FROM ai_assistant_slots WHERE workspace_id = @workspace_id AND expires_at <= @now;

-- name: AIAssistantSlotClaim :one
-- A turn's slot: the thread's only one, one of the person's per_person and
-- one of the workspace's per_workspace. None free: no row.
INSERT INTO ai_assistant_slots (thread_id, turn_id, workspace_id, user_id, ws_slot, user_slot, lease_id, expires_at)
SELECT @thread_id, @turn_id, @workspace_id, @user_id, w.s, u.s, @lease_id, @expires_at
FROM (SELECT s FROM generate_series(1, @per_workspace::int) s
      WHERE NOT EXISTS (SELECT 1 FROM ai_assistant_slots x WHERE x.workspace_id = @workspace_id AND x.ws_slot = s) ORDER BY s LIMIT 1) w,
     (SELECT s FROM generate_series(1, @per_person::int) s
      WHERE NOT EXISTS (SELECT 1 FROM ai_assistant_slots x WHERE x.workspace_id = @workspace_id AND x.user_id = @user_id AND x.user_slot = s) ORDER BY s LIMIT 1) u
ON CONFLICT DO NOTHING
RETURNING lease_id;

-- name: AIAssistantSlotOfTurn :one
-- A turn's slot while its lease holds.
SELECT * FROM ai_assistant_slots WHERE turn_id = @turn_id AND expires_at > @now;

-- name: AIAssistantSlotRenew :execrows
UPDATE ai_assistant_slots SET expires_at = @expires_at WHERE turn_id = @turn_id AND lease_id = @lease_id AND expires_at > @now;

-- name: AIAssistantSlotRelease :exec
DELETE FROM ai_assistant_slots WHERE turn_id = @turn_id AND lease_id = @lease_id;

-- name: AIAssistantSlotDrop :exec
-- The sweep frees a dead worker's slot.
DELETE FROM ai_assistant_slots WHERE turn_id = @turn_id;

-- name: AIAssistantSweepTurns :many
-- Turns no worker answers: answering without a live slot (the worker died
-- or lost its lease), or queued since before @queued_before (their job is
-- gone).
SELECT t.id, t.thread_id, h.workspace_id FROM ai_assistant_turns t JOIN ai_assistant_threads h ON h.id = t.thread_id
WHERE (t.state = 'answering' AND NOT EXISTS (SELECT 1 FROM ai_assistant_slots s WHERE s.turn_id = t.id AND s.expires_at > @now))
   OR (t.state = 'queued' AND t.created_at < @queued_before)
ORDER BY t.created_at LIMIT 100;

-- name: AIAssistantThreadToolCall :exec
-- One more of the thread's tool calls, under its lock.
UPDATE ai_assistant_threads SET tool_calls = tool_calls + 1 WHERE id = @id;

-- name: AIAssistantThreadsRemove :execrows
-- A person left the workspace or was removed from it: their threads there
-- go, with all their content. Deactivation keeps them.
DELETE FROM ai_assistant_threads WHERE workspace_id = @workspace_id AND user_id = @user_id;

-- name: AIAssistantEventsPurge :execrows
-- Stream events 24 hours after their turn ended.
DELETE FROM ai_assistant_events e
WHERE (e.thread_id, e.seq) IN (
    SELECT x.thread_id, x.seq FROM ai_assistant_events x JOIN ai_assistant_turns t ON t.id = x.turn_id
    WHERE t.ended_at < @before LIMIT 5000);

-- name: AIAssistantExpiring :many
-- Open threads past their 30 days without activity, locked in the
-- threads' lock order; a thread a worker or a person holds waits for the
-- next run.
SELECT * FROM ai_assistant_threads WHERE state = 'open' AND expires_at <= @now
ORDER BY expires_at, id LIMIT 100 FOR UPDATE SKIP LOCKED;

-- name: AIAssistantThreadTurnsEnd :many
-- Retention ends a thread's active turns: their version moves on, so a
-- worker that comes back writes nothing of them.
UPDATE ai_assistant_turns SET state = 'failed', reason = 'expired', version = version + 1, ended_at = @now
WHERE thread_id = @thread_id AND state IN ('queued', 'answering', 'waiting')
RETURNING id;

-- name: AIAssistantMessagesPurge :exec
DELETE FROM ai_assistant_messages WHERE thread_id = @thread_id;

-- name: AIAssistantContextsPurge :exec
DELETE FROM ai_assistant_contexts WHERE thread_id = @thread_id;

-- name: AIAssistantThreadEventsPurge :exec
DELETE FROM ai_assistant_events WHERE thread_id = @thread_id;

-- name: AIAssistantThreadSlotPurge :exec
DELETE FROM ai_assistant_slots WHERE thread_id = @thread_id;

-- name: AIAssistantThreadExpire :exec
-- An expired thread keeps its row (a tombstone: usage, no content).
UPDATE ai_assistant_threads SET state = 'expired', title_sealed = NULL, updated_at = @now WHERE id = @id;

-- name: AIAssistantTombstonesPurge :execrows
-- Expired threads' tombstones, 90 days after their content went.
DELETE FROM ai_assistant_threads d WHERE d.id IN (
    SELECT x.id FROM ai_assistant_threads x WHERE x.state = 'expired' AND x.expires_at <= @before
    ORDER BY x.expires_at LIMIT 1000 FOR UPDATE SKIP LOCKED);
