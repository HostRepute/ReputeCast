-- name: AIBudgetPersonLock :exec
-- A person's own budget (their personal keys' cap) has its own lock, so it
-- never waits for the workspace's nor the workspace's for it. Taken after
-- the AI configuration lock, like the workspace's.
SELECT pg_advisory_xact_lock(hashtextextended('ai_budget:' || (@workspace_id::uuid)::text || ':' || (@user_id::uuid)::text, 0));

-- name: AIBudgetScope :one
-- Whose budget a reservation is in; set when it is made and never changed.
SELECT personal, user_id FROM ai_budget_reservations WHERE id = @id AND workspace_id = @workspace_id;

-- name: AIBudgetWorkspaceLock :exec
-- The workspace budget's lock, across all its accounts and models. It is
-- taken after the AI configuration lock (AIWorkspaceSettingsLock), never
-- before; accounting needs only this one.
SELECT pg_advisory_xact_lock(hashtextextended('ai_budget:' || (@workspace_id::uuid)::text, 0));

-- name: AIBudgetInsert :exec
INSERT INTO ai_budget_reservations (id, workspace_id, user_id, run_id, provider, model, feature, pricing_revision,
    input_price, output_price, state, input_tokens, output_tokens, cost_micros, created_at, account_id, model_id, personal)
VALUES (@id, @workspace_id, sqlc.narg(user_id), sqlc.narg(run_id), @provider, @model, @feature, @pricing_revision,
    @input_price, @output_price, 'reserved', @input_tokens, @output_tokens, @cost_micros, @created_at, sqlc.narg(account_id), sqlc.narg(model_id), @personal);

-- name: AIBudgetLock :one
SELECT * FROM ai_budget_reservations WHERE id = @id AND workspace_id = @workspace_id FOR UPDATE;

-- name: AIBudgetMonthly :one
-- Settled reservations are already in ai_requests. Old uncertain requests
-- stay reserved across month rollover rather than becoming free to retry.
SELECT coalesce(sum(input_tokens + output_tokens), 0)::bigint AS tokens,
       coalesce(sum(cost_micros), 0)::bigint AS cost_micros,
       count(*) FILTER (WHERE state = 'uncertain')::bigint AS uncertain
FROM ai_budget_reservations WHERE workspace_id = @workspace_id AND state <> 'settled' AND NOT personal;

-- name: AIBudgetPersonal :one
-- What a person's own requests in flight hold of their cap.
SELECT coalesce(sum(cost_micros), 0)::bigint AS cost_micros,
       count(*) FILTER (WHERE state = 'uncertain')::bigint AS uncertain
FROM ai_budget_reservations WHERE workspace_id = @workspace_id AND user_id = @user_id AND state <> 'settled' AND personal;

-- name: AIBudgetRun :one
SELECT coalesce(sum(input_tokens + output_tokens), 0)::bigint AS tokens,
       coalesce(sum(cost_micros), 0)::bigint AS cost_micros,
       count(*) FILTER (WHERE state = 'uncertain')::bigint AS uncertain,
       count(*)::bigint AS requests
FROM ai_budget_reservations WHERE run_id = @run_id;

-- name: AIBudgetSettle :exec
UPDATE ai_budget_reservations SET state = 'settled', input_tokens = @input_tokens,
    output_tokens = @output_tokens, cost_micros = @cost_micros, settled_at = @settled_at
WHERE id = @id AND state <> 'settled';

-- name: AIBudgetUncertain :exec
UPDATE ai_budget_reservations SET state = 'uncertain' WHERE id = @id AND state = 'reserved';

-- name: AIBudgetPurge :execrows
DELETE FROM ai_budget_reservations WHERE state = 'settled' AND created_at < @before;

-- name: AIBudgetPending :many
-- The workspace's holds; personal keys' are their owners'.
SELECT * FROM ai_budget_reservations
WHERE workspace_id = @workspace_id AND state <> 'settled' AND NOT personal
ORDER BY created_at, id LIMIT 100;

-- name: AIBudgetPersonalPending :many
-- A person's holds on their own keys.
SELECT * FROM ai_budget_reservations
WHERE workspace_id = @workspace_id AND user_id = @user_id AND state <> 'settled' AND personal
ORDER BY created_at, id LIMIT 100;

-- name: AIBudgetAbandoned :execrows
-- No provider request may run for ten minutes. A dead HTTP/worker process
-- leaves an uncertain reservation, never an automatic release or retry.
UPDATE ai_budget_reservations SET state = 'uncertain'
WHERE state = 'reserved' AND created_at < @before;
