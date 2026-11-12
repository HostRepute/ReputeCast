-- +goose Up
-- M59 G-21: countdown timers in email. A Countdown block's image is drawn
-- when it's opened (/t/cd/{token}.gif); each distinct set of settings is
-- one row per workspace, found again by the hash of its canonical JSON, so
-- the same settings always give the same image address.
CREATE TABLE email_countdowns (
    id             uuid        PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    hash           bytea       NOT NULL,
    settings       jsonb       NOT NULL,
    ends_at        timestamptz NOT NULL,
    created_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    -- At most once a minute: when the image was last opened.
    last_served_at timestamptz,
    UNIQUE (workspace_id, hash)
);

-- +goose Down
DROP TABLE email_countdowns;
