-- +goose Up
-- Bot clicks (review AN-2). A click that looks like a person's waits a
-- minute before it counts: scanners click every link of an email within
-- seconds of its arrival, and follow the hidden link a person never sees,
-- so the clicks around it decide whether it was a person's. Until then it
-- is an event of class "pending" with a row here; settling it counts it
-- (the human open, the unique click, the automation trigger, Earn's
-- credit) or files it as a bot click.
CREATE TABLE tracking_pending_clicks (
    event_id   uuid        PRIMARY KEY,
    message_id uuid        NOT NULL,
    link       integer     NOT NULL,
    url        text        NOT NULL DEFAULT '',
    clicked_at timestamptz NOT NULL,
    settle_at  timestamptz NOT NULL
);
CREATE INDEX tracking_pending_clicks_due_idx ON tracking_pending_clicks (settle_at, message_id);

-- A message's clicks around a time (bursts, the hidden link).
CREATE INDEX delivery_events_message_clicks_idx ON delivery_events (message_id, occurred_at) WHERE type = 'click';

-- Networks of machines that open and click email: privacy proxies (Apple
-- Mail Privacy Protection) and mail-security scanners. The built-in ones
-- come with releases; instance admins add the ones they meet (a
-- gateway in front of their readers, a new scanner range).
CREATE TABLE tracking_networks (
    cidr       cidr        PRIMARY KEY,
    name       text        NOT NULL,
    class      text        NOT NULL CHECK (class IN ('privacy', 'scanner')),
    source     text        NOT NULL DEFAULT 'admin' CHECK (source IN ('builtin', 'admin')),
    note       text        NOT NULL DEFAULT '',
    created_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO tracking_networks (cidr, name, class, source) VALUES
    ('17.0.0.0/8', 'Apple Mail Privacy Protection', 'privacy', 'builtin'),
    ('40.92.0.0/15', 'Microsoft Defender', 'scanner', 'builtin'),
    ('40.107.0.0/16', 'Microsoft Defender', 'scanner', 'builtin'),
    ('52.100.0.0/14', 'Microsoft Defender', 'scanner', 'builtin'),
    ('104.47.0.0/17', 'Microsoft Defender', 'scanner', 'builtin'),
    ('91.220.42.0/24', 'Mimecast', 'scanner', 'builtin'),
    ('207.211.30.0/24', 'Mimecast', 'scanner', 'builtin'),
    ('205.139.110.0/24', 'Mimecast', 'scanner', 'builtin'),
    ('148.163.128.0/19', 'Proofpoint', 'scanner', 'builtin'),
    ('67.231.144.0/20', 'Proofpoint', 'scanner', 'builtin'),
    ('64.235.144.0/20', 'Barracuda', 'scanner', 'builtin'),
    ('209.222.80.0/21', 'Barracuda', 'scanner', 'builtin');

-- +goose Down
DROP TABLE tracking_networks;
DROP INDEX delivery_events_message_clicks_idx;
DROP TABLE tracking_pending_clicks;
