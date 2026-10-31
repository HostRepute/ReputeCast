-- +goose Up
-- The user agents of machines that open and click email (review AN-2):
-- security scanners and link checkers, scripts, headless browsers and link
-- previews, found by a word of their user agent. Like the networks
-- (tracking_networks), the built-in ones come with releases and instance
-- admins add the ones they meet: a gateway's own agent, a new link checker,
-- and the word of a person's device that a rule takes for a machine's (a
-- phone called "…bot"), listed as a person's.
CREATE TABLE tracking_user_agents (
    pattern    text        PRIMARY KEY CHECK (pattern = lower(pattern) AND char_length(pattern) BETWEEN 3 AND 80),
    kind       text        NOT NULL CHECK (kind IN ('scanner', 'script', 'headless', 'preview', 'person')),
    name       text        NOT NULL DEFAULT '',
    source     text        NOT NULL DEFAULT 'admin' CHECK (source IN ('builtin', 'admin')),
    note       text        NOT NULL DEFAULT '',
    created_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO tracking_user_agents (pattern, kind, name, source) VALUES
    ('barracuda', 'scanner', 'Barracuda', 'builtin'),
    ('mimecast', 'scanner', 'Mimecast', 'builtin'),
    ('proofpoint', 'scanner', 'Proofpoint', 'builtin'),
    ('safelinks', 'scanner', 'Microsoft Defender', 'builtin'),
    ('microsoft office protocol discovery', 'scanner', 'Microsoft Defender', 'builtin'),
    ('trendmicro', 'scanner', 'Trend Micro', 'builtin'),
    ('trend micro', 'scanner', 'Trend Micro', 'builtin'),
    ('sophos', 'scanner', 'Sophos', 'builtin'),
    ('fireeye', 'scanner', 'FireEye', 'builtin'),
    ('forcepoint', 'scanner', 'Forcepoint', 'builtin'),
    ('symantec', 'scanner', 'Symantec', 'builtin'),
    ('ironport', 'scanner', 'Cisco Secure Email', 'builtin'),
    ('zscaler', 'scanner', 'Zscaler', 'builtin'),
    ('checkpoint', 'scanner', 'Check Point', 'builtin'),
    ('headlesschrome', 'headless', '', 'builtin'),
    ('phantomjs', 'headless', '', 'builtin'),
    ('python-requests', 'script', '', 'builtin'),
    ('python-urllib', 'script', '', 'builtin'),
    ('curl/', 'script', '', 'builtin'),
    ('wget/', 'script', '', 'builtin'),
    ('go-http-client', 'script', '', 'builtin'),
    ('okhttp', 'script', '', 'builtin'),
    ('java/', 'script', '', 'builtin'),
    ('libwww', 'script', '', 'builtin'),
    ('httpclient', 'script', '', 'builtin'),
    ('linkexpanding', 'preview', '', 'builtin'),
    ('facebookexternalhit', 'preview', '', 'builtin'),
    ('cubot', 'person', 'CUBOT phones', 'builtin');

-- +goose Down
DROP TABLE tracking_user_agents;
