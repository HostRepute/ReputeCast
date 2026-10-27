-- +goose Up
-- Read-only report links choose what they share (review AN-16): the
-- numbers, charts and tables always; the cost per provider, the warm-up
-- state and the email itself only when the person who made the link chose
-- them. Links made before showed everything and keep doing so.
ALTER TABLE report_shares
    ADD COLUMN include_cost   boolean NOT NULL DEFAULT true,
    ADD COLUMN include_warmup boolean NOT NULL DEFAULT true,
    ADD COLUMN include_email  boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE report_shares
    DROP COLUMN include_cost,
    DROP COLUMN include_warmup,
    DROP COLUMN include_email;
