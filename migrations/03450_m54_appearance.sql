-- +goose Up
-- Dark mode (M54): the theme a member chose in Account & security ›
-- Appearance, so it follows them to every browser they sign in on. NULL
-- means "Same as device": the app follows the device's light or dark setting.

ALTER TABLE users
    ADD COLUMN theme text CONSTRAINT users_theme_known CHECK (theme IN ('light', 'dark'));

-- +goose Down
ALTER TABLE users DROP COLUMN theme;
