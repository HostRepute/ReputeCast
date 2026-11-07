-- +goose Up
-- An endpoint sends its events on behalf of the person who last set them
-- or resumed it (created_by). When that person can no longer see what an
-- event carries (they left, lost the view permission, or now reach some
-- lists only), ReputeCast pauses the endpoint and keeps why:
-- {"why":"left|lacks|scoped","name":…,"event":…,"permission":…}, written
-- in each reader's language. NULL when a person paused it, or it runs.
ALTER TABLE webhooks ADD COLUMN paused_reason jsonb;

-- +goose Down
ALTER TABLE webhooks DROP COLUMN paused_reason;
