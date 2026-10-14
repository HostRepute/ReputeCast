-- +goose Up
-- Instance admin, SSO and social sign-in, version & updates (add-on spec
-- 3.1–3.3, 3.13, 5.10, 5.11).

ALTER TABLE instance_settings
    -- Password policy: block re-use of the last N passwords (0 allows re-use).
    ADD COLUMN password_reuse        integer     NOT NULL DEFAULT 5 CHECK (password_reuse IN (0, 5, 10)),
    -- The last public URL check (DNS, TLS, reachability) and when it ran.
    ADD COLUMN public_url_check      jsonb,
    ADD COLUMN public_url_checked_at timestamptz,
    -- When the release check last ran and, when it failed, why. The result
    -- itself is latest_release / latest_release_at.
    ADD COLUMN update_checked_at     timestamptz,
    ADD COLUMN update_check_error    text        NOT NULL DEFAULT '';

-- Password history for the re-use policy: the hashes a person had before
-- their current one, newest last. A trigger records them, so every way a
-- password changes (account, reset link, CLI) is covered; ten are kept.
CREATE TABLE user_password_history (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash text        NOT NULL,
    replaced_at   timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX user_password_history_user_idx ON user_password_history (user_id, id DESC);

-- +goose StatementBegin
CREATE FUNCTION users_password_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.password_hash IS NOT NULL AND NEW.password_hash IS DISTINCT FROM OLD.password_hash THEN
        INSERT INTO user_password_history (user_id, password_hash) VALUES (OLD.id, OLD.password_hash);
        DELETE FROM user_password_history h
        WHERE h.user_id = OLD.id
          AND h.id NOT IN (SELECT x.id FROM user_password_history x WHERE x.user_id = OLD.id ORDER BY x.id DESC LIMIT 10);
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER users_password_history AFTER UPDATE OF password_hash ON users
    FOR EACH ROW EXECUTE FUNCTION users_password_history();

ALTER TABLE sso_providers
    -- The redirect URI is {public URL}/auth/oidc/{slug}/callback.
    ADD COLUMN slug                     text,
    -- How the IdP satisfies two-factor: "amr" (the ID token's amr claim says
    -- mfa) or "trusted" (the IdP enforces it, e.g. Google Workspace 2-Step
    -- Verification). NULL means the preset's default.
    ADD COLUMN twofa_mode               text CHECK (twofa_mode IN ('amr', 'trusted')),
    -- Write-only secret: when and by whom it was saved, and when it expires
    -- at the IdP (entered by the admin; Entra secrets expire).
    ADD COLUMN client_secret_set_at     timestamptz,
    ADD COLUMN client_secret_set_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN client_secret_expires_on date,
    ADD COLUMN enabled_at               timestamptz,
    ADD COLUMN created_by               uuid REFERENCES users(id) ON DELETE SET NULL;

UPDATE sso_providers p
SET slug = CASE WHEN x.rn = 1 THEN p.preset ELSE p.preset || '-' || x.rn END
FROM (SELECT id, row_number() OVER (PARTITION BY preset ORDER BY created_at, id) AS rn FROM sso_providers) x
WHERE x.id = p.id;
UPDATE sso_providers SET enabled_at = updated_at WHERE status = 'enabled';

-- New rows get a free slug from their preset ("entra", then "entra-2").
-- +goose StatementBegin
CREATE FUNCTION sso_providers_slug() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    candidate text := NEW.preset;
    n integer := 1;
BEGIN
    IF NEW.slug IS NOT NULL THEN
        RETURN NEW;
    END IF;
    WHILE EXISTS (SELECT 1 FROM sso_providers WHERE slug = candidate) LOOP
        n := n + 1;
        candidate := NEW.preset || '-' || n;
    END LOOP;
    NEW.slug := candidate;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER sso_providers_slug BEFORE INSERT ON sso_providers
    FOR EACH ROW EXECUTE FUNCTION sso_providers_slug();

ALTER TABLE sso_providers
    ALTER COLUMN slug SET NOT NULL,
    ADD CONSTRAINT sso_providers_slug_check CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,40}$');
CREATE UNIQUE INDEX sso_providers_slug_key ON sso_providers (slug);

-- Instance audit view: sign-in events are listed by action and time.
CREATE INDEX audit_events_instance_idx ON audit_events (occurred_at DESC) WHERE workspace_id IS NULL;

-- +goose Down
DROP INDEX audit_events_instance_idx;
DROP INDEX sso_providers_slug_key;
DROP TRIGGER sso_providers_slug ON sso_providers;
DROP FUNCTION sso_providers_slug();
ALTER TABLE sso_providers
    DROP CONSTRAINT sso_providers_slug_check,
    DROP COLUMN created_by,
    DROP COLUMN enabled_at,
    DROP COLUMN client_secret_expires_on,
    DROP COLUMN client_secret_set_by,
    DROP COLUMN client_secret_set_at,
    DROP COLUMN twofa_mode,
    DROP COLUMN slug;
DROP TRIGGER users_password_history ON users;
DROP FUNCTION users_password_history();
DROP TABLE user_password_history;
ALTER TABLE instance_settings
    DROP COLUMN update_check_error,
    DROP COLUMN update_checked_at,
    DROP COLUMN public_url_checked_at,
    DROP COLUMN public_url_check,
    DROP COLUMN password_reuse;
