-- +goose Up
-- A placement test of an email with conditional content (X11) sends one of
-- its versions: its number and what it shows ("Beta corner, Start a free
-- trial"; "" for none of the conditional blocks). 0: the email has none.
ALTER TABLE placement_tests
    ADD COLUMN conditional_version integer NOT NULL DEFAULT 0 CHECK (conditional_version >= 0),
    ADD COLUMN conditional_shows   text    NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE placement_tests
    DROP COLUMN conditional_shows,
    DROP COLUMN conditional_version;
