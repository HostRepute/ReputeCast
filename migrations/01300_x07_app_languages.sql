-- +goose Up
-- App languages (connect-and-reach brief §3.7): a member's app language and
-- region format with its clock and first day of the week, the instance
-- defaults for the sign-in page, members who haven't chosen and
-- invitations, and the language an invitation went out in. NULL means "not
-- chosen": the member follows the instance default, the region follows the
-- language, the clock and the week follow the region. The values are
-- checked against internal/i18n by the API; the constraints only keep them
-- well-formed, so a new translation needs no migration.

ALTER TABLE users
    ADD COLUMN language             text CONSTRAINT users_language_tag CHECK (language ~ '^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$'),
    ADD COLUMN region_format        text CONSTRAINT users_region_format_tag CHECK (region_format = 'standard' OR region_format ~ '^[a-z]{2,3}-[A-Z]{2}$'),
    ADD COLUMN hour_cycle           text CONSTRAINT users_hour_cycle_known CHECK (hour_cycle IN ('h23', 'h12')),
    ADD COLUMN week_start           text CONSTRAINT users_week_start_known CHECK (week_start IN ('monday', 'sunday')),
    -- When the member answered "Your browser prefers …": the app offers it once.
    ADD COLUMN language_prompted_at timestamptz;

ALTER TABLE instance_settings
    ADD COLUMN default_language      text NOT NULL DEFAULT 'en'
        CONSTRAINT instance_settings_default_language_tag CHECK (default_language ~ '^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$'),
    ADD COLUMN default_region_format text NOT NULL DEFAULT 'standard'
        CONSTRAINT instance_settings_default_region_format_tag CHECK (default_region_format = 'standard' OR default_region_format ~ '^[a-z]{2,3}-[A-Z]{2}$');

ALTER TABLE invitations
    ADD COLUMN language text CONSTRAINT invitations_language_tag CHECK (language ~ '^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$');

-- +goose Down
ALTER TABLE invitations DROP COLUMN language;
ALTER TABLE instance_settings DROP COLUMN default_region_format, DROP COLUMN default_language;
ALTER TABLE users
    DROP COLUMN language_prompted_at,
    DROP COLUMN week_start,
    DROP COLUMN hour_cycle,
    DROP COLUMN region_format,
    DROP COLUMN language;
