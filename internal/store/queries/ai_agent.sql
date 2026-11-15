-- name: AIAgentSettings :one
SELECT * FROM ai_agent_settings WHERE workspace_id = @workspace_id;

-- name: AIAgentSettingsLock :one
SELECT * FROM ai_agent_settings WHERE workspace_id = @workspace_id FOR UPDATE;

-- name: AIAgentCapability :one
SELECT * FROM ai_agent_capabilities WHERE workspace_id = @workspace_id;

-- name: AIAgentCapabilitySave :exec
INSERT INTO ai_agent_capabilities (workspace_id, connection_hash, supported, checked_at)
VALUES (@workspace_id, @connection_hash, @supported, @checked_at)
ON CONFLICT (workspace_id) DO UPDATE SET connection_hash = EXCLUDED.connection_hash,
    supported = EXCLUDED.supported, checked_at = EXCLUDED.checked_at;

-- name: AIAgentSettingsSave :exec
INSERT INTO ai_agent_settings (workspace_id, enabled, tool_limit, token_limit, cost_limit_micros, updated_at)
VALUES (@workspace_id, @enabled, @tool_limit, @token_limit, @cost_limit_micros, @updated_at)
ON CONFLICT (workspace_id) DO UPDATE SET enabled = EXCLUDED.enabled, tool_limit = EXCLUDED.tool_limit,
    token_limit = EXCLUDED.token_limit, cost_limit_micros = EXCLUDED.cost_limit_micros,
    revision = ai_agent_settings.revision + 1, updated_at = EXCLUDED.updated_at;

-- name: AIAgentRunLock :one
SELECT * FROM ai_agent_runs WHERE id = @id AND workspace_id = @workspace_id FOR UPDATE;

-- name: AIAgentRun :one
SELECT * FROM ai_agent_runs WHERE id = @id AND workspace_id = @workspace_id;

-- name: AIAgentRunInsert :exec
INSERT INTO ai_agent_runs (id, workspace_id, user_id, model, state, tool_limit, token_limit, cost_limit_micros, created_at, updated_at, expires_at)
VALUES (@id, @workspace_id, @user_id, @model, 'planning', @tool_limit, @token_limit, @cost_limit_micros, @now, @now, @expires_at);

-- name: AIAgentQueueStatus :one
-- Position is a snapshot, not FIFO scheduling: a user's other workspace
-- may hold their execution slot. Never return another task's identity.
SELECT EXISTS (SELECT 1 FROM ai_agent_slots active WHERE active.run_id = @id AND active.expires_at > @now) AS executing,
    (1 + (SELECT count(*) FROM ai_agent_runs r
     WHERE r.workspace_id = @workspace_id AND r.state IN ('planning','queued','building')
       AND r.expires_at > @now AND (r.created_at, r.id) < (@created_at, @id)
       AND NOT EXISTS (SELECT 1 FROM ai_agent_slots s WHERE s.run_id = r.id AND s.expires_at > @now)))::bigint AS position;

-- name: AIAgentContextInsert :exec
INSERT INTO ai_agent_contexts (run_id, brief, bindings) VALUES (@run_id, @brief, @bindings);

-- name: AIAgentContext :one
SELECT run_id, brief, plan, bindings,
    CASE WHEN @with_messages::boolean THEN messages ELSE '[]'::jsonb END AS messages
FROM ai_agent_contexts WHERE run_id = @run_id;

-- name: AIAgentContextSave :exec
UPDATE ai_agent_contexts SET brief = @brief, plan = @plan, bindings = @bindings, messages = @messages WHERE run_id = @run_id;

-- name: AIAgentRunSave :exec
UPDATE ai_agent_runs SET state = @state, version = @version, plan_revision = @plan_revision,
    accepted_revision = @accepted_revision, tool_limit = @tool_limit, token_limit = @token_limit,
    cost_limit_micros = @cost_limit_micros, tool_calls = @tool_calls, pause_reason = @pause_reason,
    updated_at = @updated_at, closed_at = sqlc.narg(closed_at)
WHERE id = @id;

-- name: AIAgentRuns :many
SELECT r.*, coalesce(nullif(u.name, ''), u.email)::text AS owner_name
FROM ai_agent_runs r JOIN users u ON u.id = r.user_id
WHERE r.workspace_id = @workspace_id
  AND (sqlc.narg(user_id)::uuid IS NULL OR user_id = sqlc.narg(user_id))
  AND (sqlc.narg(before)::timestamptz IS NULL OR (r.updated_at, r.id) < (sqlc.narg(before), sqlc.narg(before_id)::uuid))
ORDER BY r.updated_at DESC, r.id DESC LIMIT @lim;

-- name: AIAgentStepLast :one
SELECT * FROM ai_agent_steps WHERE run_id = @run_id ORDER BY sequence DESC LIMIT 1;

-- name: AIAgentSteps :many
SELECT sequence, state, summary, created_at, updated_at FROM ai_agent_steps WHERE run_id = @run_id ORDER BY sequence;

-- name: AIAgentStepInsert :exec
INSERT INTO ai_agent_steps (id, run_id, sequence, run_version, state, request, created_at, updated_at)
VALUES (@id, @run_id, @sequence, @run_version, 'prepared', @request, @now, @now);

-- name: AIAgentStepSave :exec
UPDATE ai_agent_steps SET state = @state, response = @response, results = @results,
    next_call = @next_call, summary = @summary, updated_at = @updated_at WHERE id = @id;

-- name: AIAgentStepRequested :exec
UPDATE ai_agent_steps SET state = 'requested', request = @request, updated_at = @now
WHERE id = @id AND state = 'prepared';

-- name: AIAgentStepResume :exec
-- A human continuation rebinds a saved response to the new run fence, keeping
-- its already committed tool results and next_call. Never resend its request.
UPDATE ai_agent_steps SET run_version = @run_version, updated_at = @now
WHERE id = @id AND state = 'responded';

-- name: AIAgentArtifacts :many
SELECT * FROM ai_agent_artifacts WHERE run_id = @run_id ORDER BY bundle, handle;

-- name: AIAgentReferences :many
SELECT handle, kind, target_id, result_id, decision FROM ai_agent_artifacts WHERE run_id = @run_id;

-- name: AIAgentArtifactSave :exec
INSERT INTO ai_agent_artifacts (id, run_id, handle, kind, bundle, target_id, base_revision, content, context, updated_at)
VALUES (@id, @run_id, @handle, @kind, @bundle, sqlc.narg(target_id), @base_revision, @content, @context, @updated_at)
ON CONFLICT (run_id, handle) DO UPDATE SET content = EXCLUDED.content, context = EXCLUDED.context,
    revision = ai_agent_artifacts.revision + 1, updated_at = EXCLUDED.updated_at
WHERE ai_agent_artifacts.decision = '';

-- name: AIAgentArtifactDecide :exec
UPDATE ai_agent_artifacts SET decision = @decision, result_id = sqlc.narg(result_id), updated_at = @updated_at
WHERE id = @id AND decision = '';

-- name: AIAgentArtifactDiscard :exec
UPDATE ai_agent_artifacts SET content = NULL, context = NULL WHERE id = @id AND decision = 'discarded';

-- name: AIAgentSlotsExpire :exec
-- Never wait for another workspace's in-flight slot while holding this
-- workspace's connection lock. Unique constraints arbitrate claims.
DELETE FROM ai_agent_slots WHERE user_id IN (
    SELECT s.user_id FROM ai_agent_slots s WHERE s.expires_at <= @now
    ORDER BY s.user_id FOR UPDATE SKIP LOCKED
);

-- name: AIAgentSlotClaim :one
INSERT INTO ai_agent_slots (user_id, workspace_id, slot, run_id, lease_id, expires_at)
SELECT @user_id, @workspace_id, s, @run_id, @lease_id, @expires_at FROM generate_series(1, 2) s
WHERE NOT EXISTS (SELECT 1 FROM ai_agent_slots WHERE workspace_id = @workspace_id AND slot = s)
ORDER BY s LIMIT 1 ON CONFLICT DO NOTHING RETURNING *;

-- name: AIAgentSlot :one
SELECT * FROM ai_agent_slots WHERE run_id = @run_id AND expires_at > @now FOR UPDATE;

-- name: AIAgentSlotRelease :exec
DELETE FROM ai_agent_slots WHERE run_id = @run_id AND lease_id = @lease_id;

-- name: AIAgentRetentionBatch :many
-- Taking the same run lock as every continuation/decision prevents a late
-- checkpoint from restoring content after the retention sweep removed it.
SELECT * FROM ai_agent_runs
WHERE (expires_at <= @now AND state <> 'expired')
   OR (closed_at + interval '7 days' <= @now AND EXISTS (
      SELECT 1 FROM ai_agent_contexts c WHERE c.run_id = ai_agent_runs.id AND c.messages <> '[]'::jsonb))
ORDER BY expires_at, id LIMIT 100 FOR UPDATE SKIP LOCKED;

-- name: AIAgentPurgeRunContext :exec
DELETE FROM ai_agent_contexts WHERE run_id = @run_id;

-- name: AIAgentPurgeRunMessages :exec
UPDATE ai_agent_contexts SET messages = '[]', bindings = bindings - 'Conversation' - 'Report' WHERE run_id = @run_id;

-- name: AIAgentPurgeRunSteps :exec
UPDATE ai_agent_steps SET request = NULL, response = NULL, results = '[]',
    summary = CASE WHEN @expired::boolean THEN '{}'::jsonb ELSE summary END
WHERE run_id = @run_id;

-- name: AIAgentPurgeRunArtifacts :exec
UPDATE ai_agent_artifacts SET content = NULL, context = NULL,
    decision = CASE WHEN decision = '' THEN 'expired' ELSE decision END
WHERE run_id = @run_id;

-- name: AIAgentCouponPool :one
-- Explicit selection only. Never load codes, assignments or store credentials.
SELECT id, name, slug, source, status, updated_at
FROM coupon_pools
WHERE workspace_id = @workspace_id AND id = @id
FOR SHARE;
