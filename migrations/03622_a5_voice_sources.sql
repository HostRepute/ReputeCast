-- +goose Up
ALTER TABLE mcp_connect_sources DROP CONSTRAINT mcp_connect_sources_kind_check;
ALTER TABLE mcp_connect_sources ADD CONSTRAINT mcp_connect_sources_kind_check
    CHECK (kind IN ('campaign', 'post', 'automation', 'segment', 'template', 'voice'));

-- +goose Down
DELETE FROM mcp_connect_sources WHERE kind = 'voice';
ALTER TABLE mcp_connect_sources DROP CONSTRAINT mcp_connect_sources_kind_check;
ALTER TABLE mcp_connect_sources ADD CONSTRAINT mcp_connect_sources_kind_check
    CHECK (kind IN ('campaign', 'post', 'automation', 'segment', 'template'));
