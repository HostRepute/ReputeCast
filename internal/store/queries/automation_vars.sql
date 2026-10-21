-- Trigger data in automation emails (design expansion phase 7).

-- name: AutomationMessageVarsInsert :exec
INSERT INTO automation_message_vars (message_id, vars) VALUES (@message_id, @vars)
ON CONFLICT (message_id) DO NOTHING;

-- name: AutomationMessageVars :one
SELECT vars FROM automation_message_vars WHERE message_id = @message_id;

-- name: AutomationRunsExitBought :many
-- A refund makes the contact leave the "Bought a product" flows of that
-- product (or of any product) they're in.
UPDATE automation_runs r SET status = 'exited', finished_at = @at::timestamptz, next_at = NULL, exit_reason = 'refunded'
FROM automations a
WHERE a.id = r.automation_id AND a.workspace_id = @workspace_id AND a.deleted_at IS NULL
  AND r.contact_id = @contact_id AND r.status IN ('active', 'waiting')
  AND a.trigger ->> 'kind' = 'bought_product' AND coalesce(a.trigger ->> 'productId', '') IN ('', @product_id::text)
RETURNING r.id, r.automation_id, r.version, r.node_id;

-- name: AutomationGoalRunsBought :many
-- Runs of a version whose contact bought the product (any product when
-- empty) since entering, not refunded in full.
SELECT r.id FROM automation_runs r
WHERE r.automation_id = @automation_id AND r.version = @version AND r.status IN ('active', 'waiting')
  AND EXISTS (SELECT 1 FROM product_orders o WHERE o.contact_id = r.contact_id AND o.paid_at >= r.entered_at AND o.status <> 'refunded'
              AND (@product_id::text = '' OR o.product_id::text = @product_id::text));

-- name: AutomationContactBought :one
-- Whether a contact bought the product (any product when empty) since a
-- time, not refunded in full.
SELECT EXISTS (SELECT 1 FROM product_orders o WHERE o.workspace_id = @workspace_id AND o.contact_id = @contact_id AND o.paid_at >= @since::timestamptz
               AND o.status <> 'refunded' AND (@product_id::text = '' OR o.product_id::text = @product_id::text))::boolean AS bought;

-- name: AutomationProductRefs :many
-- The products a trigger or goal can name.
SELECT id, name FROM products WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY name;
