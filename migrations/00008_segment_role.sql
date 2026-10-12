-- +goose Up
-- User-written SQL segments run as a role that owns nothing and can read
-- nothing on its own. At query time the app creates a temporary view of one
-- workspace's contacts, grants SELECT on it to this role, switches to the role
-- inside a READ ONLY transaction with a statement_timeout and rolls back.
-- Creating a role needs CREATEROLE; on managed databases without it the
-- migration skips the role and SQL segments stay disabled until an admin
-- runs: CREATE ROLE reputecast_segments NOLOGIN; GRANT reputecast_segments TO <app user>;
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'reputecast_segments') THEN
        BEGIN
            CREATE ROLE reputecast_segments NOLOGIN NOINHERIT;
        EXCEPTION WHEN insufficient_privilege THEN
            RAISE NOTICE 'reputecast_segments role not created: missing CREATEROLE';
            RETURN;
        END;
    END IF;
    BEGIN
        EXECUTE format('GRANT reputecast_segments TO %I', current_user);
    EXCEPTION WHEN insufficient_privilege OR duplicate_object THEN
        RAISE NOTICE 'could not grant reputecast_segments to %', current_user;
    END;
END;
$$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
