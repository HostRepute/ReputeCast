-- +goose Up
-- B6: the monthly caps' running totals. Every request's reservation reads
-- what its budget has used this month, under the budget's lock; summing
-- ai_requests grew with every request of the month. A trigger keeps hourly
-- totals per budget instead, in the statement that changes ai_requests, so
-- they can't drift from it: the workspace's keys (personal false, the nil
-- UUID) and each person's own keys. A personal request whose person was
-- deleted is in nobody's budget any more and isn't counted.
--
-- Each budget's hour is spread over 8 rows (shards) by the request's id, so
-- requests accounted at once update different rows instead of queueing on
-- one (B6's load test: 100 turns ending together in a workspace waited
-- 31 ms each for its hour's row). Readers sum the rows. Eight: a pool of
-- 20 connections accounts some 10 requests of a workspace at once, and
-- each then shares its row with about one other instead of all nine, while
-- a cap check over a month of busy hours sums 6,000 rows (2.5 ms; 16 shards
-- would be 3.9 ms, one row 0.9 ms).
CREATE TABLE ai_usage_hours (
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    personal      boolean     NOT NULL,
    user_id       uuid        NOT NULL,
    -- The hour's start, binned in UTC.
    hour          timestamptz NOT NULL,
    -- ai_usage_shard of the requests' ids.
    shard         smallint    NOT NULL CHECK (shard BETWEEN 0 AND 7),
    requests      bigint      NOT NULL CHECK (requests > 0),
    input_tokens  bigint      NOT NULL,
    output_tokens bigint      NOT NULL,
    cost_micros   bigint      NOT NULL,
    PRIMARY KEY (workspace_id, personal, user_id, hour, shard)
);

-- The hour a time is in, and the first hour's start at or after it.
-- +goose StatementBegin
CREATE FUNCTION ai_usage_hour(at timestamptz) RETURNS timestamptz LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT date_bin('1 hour', at, '2000-01-01 00:00:00+00'::timestamptz)
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION ai_usage_hour_after(at timestamptz) RETURNS timestamptz LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT date_bin('1 hour', at + interval '1 hour' - interval '1 microsecond', '2000-01-01 00:00:00+00'::timestamptz)
$$;
-- +goose StatementEnd

-- A request's shard: its id's last byte (random in UUIDv7 and v4), so the
-- same request always lands in the same row, whatever the server's
-- version or Postgres's.
-- +goose StatementBegin
CREATE FUNCTION ai_usage_shard(id uuid) RETURNS smallint LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT (get_byte(uuid_send(id), 15) % 8)::smallint
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION ai_usage_hours_track() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    v_nil uuid := '00000000-0000-0000-0000-000000000000';
    v_old uuid;
    v_new uuid;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        v_old := CASE WHEN OLD.personal THEN OLD.user_id ELSE v_nil END;
    END IF;
    IF TG_OP <> 'DELETE' THEN
        v_new := CASE WHEN NEW.personal THEN NEW.user_id ELSE v_nil END;
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.id = NEW.id AND OLD.workspace_id = NEW.workspace_id AND OLD.personal = NEW.personal AND v_old IS NOT DISTINCT FROM v_new
       AND ai_usage_hour(OLD.created_at) = ai_usage_hour(NEW.created_at)
       AND OLD.input_tokens = NEW.input_tokens AND OLD.output_tokens = NEW.output_tokens AND OLD.cost_micros = NEW.cost_micros THEN
        RETURN NULL;
    END IF;
    -- The old row's shard comes from OLD.id and the new one's from NEW.id:
    -- a change takes a request out of the shard it was added to.
    IF v_old IS NOT NULL THEN
        DELETE FROM ai_usage_hours
        WHERE workspace_id = OLD.workspace_id AND personal = OLD.personal AND user_id = v_old
          AND hour = ai_usage_hour(OLD.created_at) AND shard = ai_usage_shard(OLD.id) AND requests = 1;
        IF NOT FOUND THEN
            UPDATE ai_usage_hours SET requests = requests - 1, input_tokens = input_tokens - OLD.input_tokens,
                output_tokens = output_tokens - OLD.output_tokens, cost_micros = cost_micros - OLD.cost_micros
            WHERE workspace_id = OLD.workspace_id AND personal = OLD.personal AND user_id = v_old
              AND hour = ai_usage_hour(OLD.created_at) AND shard = ai_usage_shard(OLD.id);
        END IF;
    END IF;
    IF v_new IS NOT NULL THEN
        INSERT INTO ai_usage_hours AS h (workspace_id, personal, user_id, hour, shard, requests, input_tokens, output_tokens, cost_micros)
        VALUES (NEW.workspace_id, NEW.personal, v_new, ai_usage_hour(NEW.created_at), ai_usage_shard(NEW.id), 1,
                NEW.input_tokens, NEW.output_tokens, NEW.cost_micros)
        ON CONFLICT (workspace_id, personal, user_id, hour, shard) DO UPDATE SET requests = h.requests + 1, input_tokens = h.input_tokens + EXCLUDED.input_tokens,
            output_tokens = h.output_tokens + EXCLUDED.output_tokens, cost_micros = h.cost_micros + EXCLUDED.cost_micros;
    END IF;
    RETURN NULL;
END
$$;
-- +goose StatementEnd

-- Created before the totals are filled: it waits for requests being written
-- and holds new ones until this migration commits, so none is missed.
CREATE TRIGGER ai_requests_usage_hours AFTER INSERT OR DELETE OR UPDATE OF id, workspace_id, user_id, personal, created_at, input_tokens, output_tokens, cost_micros
    ON ai_requests FOR EACH ROW EXECUTE FUNCTION ai_usage_hours_track();

INSERT INTO ai_usage_hours (workspace_id, personal, user_id, hour, shard, requests, input_tokens, output_tokens, cost_micros)
SELECT workspace_id, personal, CASE WHEN personal THEN user_id ELSE '00000000-0000-0000-0000-000000000000' END,
       ai_usage_hour(created_at), ai_usage_shard(id), count(*), sum(input_tokens), sum(output_tokens), sum(cost_micros)
FROM ai_requests WHERE NOT personal OR user_id IS NOT NULL
GROUP BY 1, 2, 3, 4, 5;

-- +goose Down
DROP TRIGGER ai_requests_usage_hours ON ai_requests;
DROP FUNCTION ai_usage_hours_track();
DROP FUNCTION ai_usage_shard(uuid);
DROP FUNCTION ai_usage_hour_after(timestamptz);
DROP FUNCTION ai_usage_hour(timestamptz);
DROP TABLE ai_usage_hours;
