-- +goose Up
-- The time step (Unix time / 30 s) of the last authenticator code accepted
-- for each person. A code is accepted once: one read over a shoulder or
-- relayed by a phishing page within its 30 seconds can't sign in again.
ALTER TABLE users ADD COLUMN totp_last_step bigint;

-- +goose Down
ALTER TABLE users DROP COLUMN totp_last_step;
