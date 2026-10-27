-- +goose Up
-- Requests that reach ReputeCast through a proxy it doesn't trust (review
-- AN-8): a private address that sends forwarding headers (X-Forwarded-For,
-- X-Real-IP, Forwarded) while http.trusted_proxies doesn't list it. Every
-- open and click then carries the proxy's address instead of the reader's,
-- so the bot filter can't tell Apple's privacy proxy or a scanner network
-- from a person; instance admins are told how to fix it (notified_at).
CREATE TABLE untrusted_proxies (
    peer        inet        PRIMARY KEY,
    header      text        NOT NULL,
    requests    bigint      NOT NULL DEFAULT 0,
    first_seen  timestamptz NOT NULL,
    last_seen   timestamptz NOT NULL,
    notified_at timestamptz
);

-- +goose Down
DROP TABLE untrusted_proxies;
