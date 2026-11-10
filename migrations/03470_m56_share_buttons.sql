-- +goose Up
-- Share buttons on posts (M56, G-9): Copy link, Email, X, LinkedIn,
-- Facebook, Bluesky and WhatsApp under every published post. On for every
-- website, existing ones included; Website › Home & navigation turns them off.

ALTER TABLE website_settings ADD COLUMN share_buttons boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE website_settings DROP COLUMN share_buttons;
