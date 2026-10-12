-- +goose Up
-- SQL segments (spec 5.10, add-on §6: segments:sql) run user-written SQL.
-- 00008 created reputecast_segments, a role that can read nothing on its
-- own. This adds the function that executes a segment's SQL *as* that role:
-- it is SECURITY DEFINER and owned by reputecast_segments, so inside it the
-- current user is the restricted role, and Postgres refuses any attempt to
-- change roles ("cannot set parameter role within security-definer
-- function"). The app calls it inside a READ ONLY transaction with a
-- statement_timeout, after creating temporary views of one workspace's
-- contacts, list memberships and events and granting SELECT on them to the
-- role, then rolls back (see internal/segments/sqlmode.go).
--
-- Changing a function's owner needs membership in the new owner and CREATE
-- on its schema, so the function lives in a schema of its own. Databases
-- where this isn't possible (no role, managed Postgres without CREATEROLE)
-- skip it; SQL segments then report that they are disabled.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'reputecast_segments') THEN
        RAISE NOTICE 'reputecast_segments role missing: SQL segments stay disabled';
        RETURN;
    END IF;
    IF NOT pg_has_role(current_user, 'reputecast_segments', 'MEMBER') THEN
        RAISE NOTICE '% is not a member of reputecast_segments: SQL segments stay disabled', current_user;
        RETURN;
    END IF;
    BEGIN
        CREATE SCHEMA IF NOT EXISTS reputecast_sql;
        EXECUTE $create$
            CREATE OR REPLACE FUNCTION reputecast_sql.segment_ids(query text) RETURNS SETOF uuid
                LANGUAGE plpgsql SECURITY DEFINER
                SET search_path = pg_temp, pg_catalog
            AS $fn$
            BEGIN
                RETURN QUERY EXECUTE 'SELECT s.id::uuid FROM (' || query || ') s';
            END
            $fn$
        $create$;
        GRANT USAGE, CREATE ON SCHEMA reputecast_sql TO reputecast_segments;
        ALTER FUNCTION reputecast_sql.segment_ids(text) OWNER TO reputecast_segments;
        REVOKE CREATE ON SCHEMA reputecast_sql FROM reputecast_segments;
        REVOKE ALL ON FUNCTION reputecast_sql.segment_ids(text) FROM PUBLIC;
        EXECUTE format('GRANT EXECUTE ON FUNCTION reputecast_sql.segment_ids(text) TO %I', current_user);
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE NOTICE 'could not set up reputecast_sql.segment_ids: SQL segments stay disabled';
    END;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP SCHEMA IF EXISTS reputecast_sql CASCADE;
