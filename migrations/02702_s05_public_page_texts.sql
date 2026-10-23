-- +goose Up
-- The public pages' settings keep, beside the English of the custom
-- domain's DNS check and of Turnstile's last error, the messages they came
-- from (internal/i18n Text): {"domain_detail": {...}, "turnstile_error":
-- {...}}, so each member reads them in their language. Rows without one
-- keep reading in their English.
ALTER TABLE public_page_settings ADD COLUMN texts jsonb NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE public_page_settings DROP COLUMN texts;
