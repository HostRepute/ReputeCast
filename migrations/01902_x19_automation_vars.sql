-- +goose Up
-- Trigger data in automation emails (design expansion phase 7): a flow
-- started by "Bought a product" writes the order in Liquid
-- ({{ order.product_name }}, {{ order.amount }}). The trigger event's data
-- travels in the run's state; each email the run sends keeps a copy here,
-- because the sending engine renders it later, outside the run.

CREATE TABLE automation_message_vars (
    message_id uuid PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    vars       jsonb NOT NULL DEFAULT '{}'
);

-- +goose Down
DROP TABLE automation_message_vars;
