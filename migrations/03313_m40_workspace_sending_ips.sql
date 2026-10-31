-- The sending IPs each workspace has sent from (review GS-10): a dedicated
-- IP that never sent for the workspace starts a canary, as a new sending
-- domain does. Only IPs ReputeCast knows are here, the ones a dedicated
-- provider lists; a shared pool's IPs can't be seen. The send loop records
-- an IP when mail goes out through it.

-- +goose Up
CREATE TABLE workspace_sending_ips (
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    ip            text        NOT NULL,
    provider_id   uuid        REFERENCES providers(id) ON DELETE SET NULL,
    first_sent_at timestamptz NOT NULL,
    last_sent_at  timestamptz NOT NULL,
    PRIMARY KEY (workspace_id, ip)
);

-- The dedicated IPs that already sent, from the sends that went through
-- their providers, so they don't start canaries as if they were new.
INSERT INTO workspace_sending_ips (workspace_id, ip, provider_id, first_sent_at, last_sent_at)
SELECT p.workspace_id, a.ip, p.id, s.first_at, s.last_at
FROM providers p
CROSS JOIN LATERAL unnest(p.ips) AS a (ip)
CROSS JOIN LATERAL (
    SELECT min(c.started_at) AS first_at, max(coalesce(c.completed_at, c.started_at)) AS last_at
    FROM campaign_counters cc JOIN campaigns c ON c.id = cc.campaign_id
    WHERE cc.provider_id = p.id AND cc.sent > 0 AND c.started_at IS NOT NULL
) s
WHERE p.ip_mode = 'dedicated' AND s.first_at IS NOT NULL
ON CONFLICT (workspace_id, ip) DO NOTHING;

-- +goose Down
DROP TABLE workspace_sending_ips;
