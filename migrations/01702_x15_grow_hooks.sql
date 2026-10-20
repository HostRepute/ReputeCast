-- +goose Up
-- Where Grow (M25) meets other domains: milestone emails go out through
-- the Automations outbox from the program's sender, "Reached a milestone"
-- starts automations, and the Recommendations block counts the delivered
-- emails that showed each recommendation.

-- An automation email can come from its own sender and reply-to (milestone
-- emails use the program's); NULL keeps the workspace's default.
ALTER TABLE automation_emails
    ADD COLUMN sender_id uuid REFERENCES senders(id) ON DELETE SET NULL,
    ADD COLUMN reply_to  text NOT NULL DEFAULT '';

-- "Reached a milestone" starts automations: add the kind to the trigger
-- events, keeping every kind another migration may have added.
-- +goose StatementBegin
DO $$
DECLARE
    def text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO def FROM pg_constraint
    WHERE conrelid = 'automation_events'::regclass AND conname = 'automation_events_kind_check';
    IF def IS NOT NULL AND position('milestone_reached' IN def) = 0 THEN
        ALTER TABLE automation_events DROP CONSTRAINT automation_events_kind_check;
        EXECUTE 'ALTER TABLE automation_events ADD CONSTRAINT automation_events_kind_check '
            || replace(def, 'ARRAY[', 'ARRAY[''milestone_reached''::text, ');
    END IF;
END;
$$;
-- +goose StatementEnd

-- The recommendations an email showed, per message, until the provider says
-- it was delivered ("Shown" counts delivered emails, not sends or opens).
CREATE TABLE recommendation_message_shown (
    message_id         uuid PRIMARY KEY,
    workspace_id       uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    recommendation_ids uuid[]      NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX recommendation_message_shown_age_idx ON recommendation_message_shown (created_at);

-- +goose Down
DROP TABLE recommendation_message_shown;
DELETE FROM automation_events WHERE kind = 'milestone_reached';
-- +goose StatementBegin
DO $$
DECLARE
    def text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO def FROM pg_constraint
    WHERE conrelid = 'automation_events'::regclass AND conname = 'automation_events_kind_check';
    IF def IS NOT NULL AND position('milestone_reached' IN def) > 0 THEN
        ALTER TABLE automation_events DROP CONSTRAINT automation_events_kind_check;
        EXECUTE 'ALTER TABLE automation_events ADD CONSTRAINT automation_events_kind_check '
            || replace(def, '''milestone_reached''::text, ', '');
    END IF;
END;
$$;
-- +goose StatementEnd
ALTER TABLE automation_emails DROP COLUMN reply_to, DROP COLUMN sender_id;
