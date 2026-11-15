-- +goose Up
-- Explicit human-selected sources, scoped to exactly one OAuth grant.
CREATE TABLE mcp_connect_sources (
    handle text PRIMARY KEY,
    grant_id uuid NOT NULL REFERENCES mcp_connect_grants(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('campaign', 'post', 'automation', 'segment', 'template')),
    target_id uuid NOT NULL,
    base_revision bigint NOT NULL,
    snapshot text NOT NULL,
    context jsonb NOT NULL,
    projected jsonb NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX mcp_connect_sources_grant ON mcp_connect_sources (grant_id, created_at DESC);
CREATE INDEX mcp_connect_sources_expiry ON mcp_connect_sources (expires_at);

CREATE TABLE mcp_connect_proposals (
    id uuid PRIMARY KEY,
    grant_id uuid NOT NULL REFERENCES mcp_connect_grants(id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_key text NOT NULL,
    request_hash bytea NOT NULL CHECK (octet_length(request_hash) = 32),
    kind text NOT NULL CHECK (kind IN ('campaign', 'post', 'automation', 'segment')),
    target_id uuid,
    base_revision bigint NOT NULL,
    context jsonb NOT NULL,
    content jsonb NOT NULL,
    preview text NOT NULL,
    decision text NOT NULL DEFAULT '' CHECK (decision IN ('', 'kept', 'discarded', 'accepted', 'rejected')),
    decided_by uuid REFERENCES users(id) ON DELETE SET NULL,
    decided_at timestamptz,
    result_id uuid,
    result_revision bigint,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    purge_at timestamptz NOT NULL,
    purged_at timestamptz,
    UNIQUE (grant_id, request_key),
    CHECK (request_key ~ '^[A-Za-z0-9_-]{1,80}$'),
    CHECK (expires_at > created_at AND expires_at <= created_at + interval '7 days'),
    CHECK (purge_at <= created_at + interval '30 days'),
    CHECK (octet_length(context::text) <= 524288 AND octet_length(content::text) <= 524288 AND octet_length(preview) <= 1048576),
    CHECK ((decision = '') = (decided_at IS NULL)),
    CHECK (decision = '' OR (target_id IS NULL AND decision IN ('kept', 'discarded')) OR (target_id IS NOT NULL AND decision IN ('accepted', 'rejected'))),
    CHECK ((decision IN ('kept', 'accepted')) = (result_id IS NOT NULL)),
    CHECK ((target_id IS NULL AND base_revision = 0) OR (target_id IS NOT NULL AND base_revision > 0))
);
CREATE INDEX mcp_connect_proposals_workspace ON mcp_connect_proposals (workspace_id, id DESC);
CREATE INDEX mcp_connect_proposals_pending ON mcp_connect_proposals (grant_id, expires_at) WHERE decision = '' AND purged_at IS NULL;
CREATE INDEX mcp_connect_proposals_purge ON mcp_connect_proposals (purge_at) WHERE purged_at IS NULL;

-- A decision applies exactly the content the reviewer opened. The only
-- content mutation is retention's irreversible removal of text and bindings.
-- +goose StatementBegin
CREATE FUNCTION mcp_connect_proposal_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (to_jsonb(NEW) - ARRAY['decision', 'decided_by', 'decided_at', 'result_id', 'result_revision', 'purge_at', 'purged_at', 'context', 'content', 'preview'])
       IS DISTINCT FROM
       (to_jsonb(OLD) - ARRAY['decision', 'decided_by', 'decided_at', 'result_id', 'result_revision', 'purge_at', 'purged_at', 'context', 'content', 'preview'])
       OR NEW.purge_at > OLD.purge_at THEN
        RAISE EXCEPTION 'immutable external proposal' USING ERRCODE = '23514';
    END IF;
    IF ROW(NEW.context, NEW.content, NEW.preview, NEW.purged_at) IS DISTINCT FROM ROW(OLD.context, OLD.content, OLD.preview, OLD.purged_at)
       AND NOT (OLD.purged_at IS NULL AND NEW.purged_at IS NOT NULL AND NEW.purged_at >= OLD.purge_at
                AND NEW.context = jsonb_build_object('Source', OLD.context->'Source') AND NEW.content = '{}'::jsonb AND NEW.preview = '') THEN
        RAISE EXCEPTION 'immutable external proposal content' USING ERRCODE = '23514';
    END IF;
    IF OLD.decision <> '' AND ROW(NEW.decision, NEW.decided_at, NEW.result_id, NEW.result_revision)
       IS DISTINCT FROM ROW(OLD.decision, OLD.decided_at, OLD.result_id, OLD.result_revision) THEN
        RAISE EXCEPTION 'external proposal already decided' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER mcp_connect_proposal_immutable BEFORE UPDATE ON mcp_connect_proposals
FOR EACH ROW EXECUTE FUNCTION mcp_connect_proposal_immutable();

-- +goose Down
DROP TABLE mcp_connect_proposals;
DROP FUNCTION mcp_connect_proposal_immutable();
DROP TABLE mcp_connect_sources;
