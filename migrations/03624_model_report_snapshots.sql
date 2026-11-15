-- +goose Up
-- One disclosure per campaign, shared across grants and runs: refreshing or
-- reconnecting must not reveal small changes in live counters.
CREATE TABLE report_model_snapshots (
    campaign_id uuid PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    content jsonb,
    created_at timestamptz NOT NULL,
    CHECK (octet_length(content::text) <= 4096)
);
-- Purge counts after thirty days; retain only the campaign tombstone so a
-- later selection cannot reconstruct or refresh an expired disclosure.
CREATE INDEX report_model_snapshots_retention ON report_model_snapshots(created_at)
    WHERE content IS NOT NULL;

ALTER TABLE mcp_connect_sources DROP CONSTRAINT mcp_connect_sources_kind_check;
ALTER TABLE mcp_connect_sources ADD CONSTRAINT mcp_connect_sources_kind_check
    CHECK (kind IN ('campaign', 'post', 'automation', 'segment', 'template', 'voice', 'list', 'report'));

-- +goose Down
DELETE FROM mcp_connect_sources WHERE kind = 'report';
ALTER TABLE mcp_connect_sources DROP CONSTRAINT mcp_connect_sources_kind_check;
ALTER TABLE mcp_connect_sources ADD CONSTRAINT mcp_connect_sources_kind_check
    CHECK (kind IN ('campaign', 'post', 'automation', 'segment', 'template', 'voice', 'list'));
DROP TABLE report_model_snapshots;
