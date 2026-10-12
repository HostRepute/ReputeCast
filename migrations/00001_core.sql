-- +goose Up
-- Instance, users, sign-in methods, sessions and notifications.
-- Primary keys are UUIDv7 generated in Go (uuid.NewV7), so Postgres 16 and 17 work.
-- Emails are stored lower-cased (enforced by CHECK) instead of using citext.

CREATE TABLE instance_settings (
    id                   boolean PRIMARY KEY DEFAULT true CHECK (id),
    name                 text        NOT NULL DEFAULT 'ReputeCast',
    logo_svg             text,
    public_url           text        NOT NULL DEFAULT '',
    default_timezone     text        NOT NULL DEFAULT 'UTC',
    update_check         boolean     NOT NULL DEFAULT true,
    setup_completed_at   timestamptz,
    -- Authentication policy (Instance admin › Authentication).
    allowed_methods      text[]      NOT NULL DEFAULT ARRAY['password','passkey','social','sso'],
    password_min_length  integer     NOT NULL DEFAULT 12 CHECK (password_min_length >= 8),
    breached_check       boolean     NOT NULL DEFAULT false,
    twofa_enforcement    text        NOT NULL DEFAULT 'sensitive' CHECK (twofa_enforcement IN ('off','sensitive','everyone')),
    twofa_grace_days     integer     NOT NULL DEFAULT 7 CHECK (twofa_grace_days BETWEEN 0 AND 90),
    session_lifetime     interval    NOT NULL DEFAULT '14 days',
    session_idle         interval    NOT NULL DEFAULT '12 hours',
    lockout_attempts     integer     NOT NULL DEFAULT 10 CHECK (lockout_attempts BETWEEN 3 AND 100),
    lockout_minutes      integer     NOT NULL DEFAULT 15 CHECK (lockout_minutes BETWEEN 1 AND 1440),
    jit_domain_allowlist text[]      NOT NULL DEFAULT '{}',
    -- Social sign-in: {"google":{"enabled":true,"client_id":"..."}, "github":{"org":"acme-eng"}}.
    -- Client secrets live in social_secrets_enc, never here.
    social_providers     jsonb       NOT NULL DEFAULT '{}',
    social_secrets_enc   bytea,
    audit_retention_days integer     NOT NULL DEFAULT 365 CHECK (audit_retention_days BETWEEN 30 AND 3650),
    latest_release       jsonb,
    latest_release_at    timestamptz,
    updated_at           timestamptz NOT NULL DEFAULT now()
);
INSERT INTO instance_settings (id) VALUES (true);

CREATE TABLE users (
    id                    uuid PRIMARY KEY,
    email                 text        NOT NULL CHECK (email = lower(email) AND position('@' IN email) > 1),
    name                  text        NOT NULL DEFAULT '',
    avatar_color          text        NOT NULL DEFAULT 'info',
    timezone              text        NOT NULL DEFAULT 'UTC',
    password_hash         text,
    password_set_at       timestamptz,
    must_change_password  boolean     NOT NULL DEFAULT false,
    is_instance_admin     boolean     NOT NULL DEFAULT false,
    break_glass           boolean     NOT NULL DEFAULT false,
    status                text        NOT NULL DEFAULT 'active' CHECK (status IN ('active','deactivated')),
    deactivated_at        timestamptz,
    totp_secret_enc       bytea,
    totp_enabled_at       timestamptz,
    twofa_grace_until     timestamptz,
    failed_attempts       integer     NOT NULL DEFAULT 0,
    locked_until          timestamptz,
    last_sign_in_at       timestamptz,
    last_active_at        timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_key ON users (email);

CREATE TABLE user_recovery_codes (
    id         uuid PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  bytea       NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX user_recovery_codes_user_idx ON user_recovery_codes (user_id);

CREATE TABLE user_passkeys (
    id               uuid PRIMARY KEY,
    user_id          uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name             text        NOT NULL,
    credential_id    bytea       NOT NULL,
    public_key       bytea       NOT NULL,
    attestation_type text        NOT NULL DEFAULT '',
    aaguid           bytea,
    sign_count       bigint      NOT NULL DEFAULT 0,
    transports       text[]      NOT NULL DEFAULT '{}',
    backup_eligible  boolean     NOT NULL DEFAULT false,
    backup_state     boolean     NOT NULL DEFAULT false,
    created_at       timestamptz NOT NULL DEFAULT now(),
    last_used_at     timestamptz
);
CREATE UNIQUE INDEX user_passkeys_credential_key ON user_passkeys (credential_id);
CREATE INDEX user_passkeys_user_idx ON user_passkeys (user_id);

-- SSO providers are configured at instance level (OIDC only; SAML/SCIM are out of scope).
CREATE TABLE sso_providers (
    id                uuid PRIMARY KEY,
    preset            text        NOT NULL CHECK (preset IN ('google','entra','okta','keycloak','authentik','oidc')),
    name              text        NOT NULL,
    issuer_url        text        NOT NULL DEFAULT '',
    client_id         text        NOT NULL DEFAULT '',
    client_secret_enc bytea,
    scopes            text[]      NOT NULL DEFAULT ARRAY['openid','email','profile'],
    -- {"email":"email","email_verified":"email_verified","name":"name","groups":"groups"}
    claim_map         jsonb       NOT NULL DEFAULT '{}',
    allowed_domains   text[]      NOT NULL DEFAULT '{}',
    jit_enabled       boolean     NOT NULL DEFAULT false,
    jit_workspace_id  uuid,
    jit_role_key      text,
    -- [{"group":"3f2a…c91","label":"Marketing","workspace_id":"…","role_key":"editor","scope":{...}}]
    group_mappings    jsonb       NOT NULL DEFAULT '[]',
    enforce_domains   text[]      NOT NULL DEFAULT '{}',
    status            text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','enabled','disabled','error')),
    last_test         jsonb,
    discovery         jsonb,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- Social and SSO identities linked to a user.
CREATE TABLE user_identities (
    id              uuid PRIMARY KEY,
    user_id         uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind            text        NOT NULL CHECK (kind IN ('google','microsoft','github','sso')),
    sso_provider_id uuid        REFERENCES sso_providers(id) ON DELETE CASCADE,
    subject         text        NOT NULL,
    email           text        NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_used_at    timestamptz
);
CREATE UNIQUE INDEX user_identities_subject_key ON user_identities (kind, coalesce(sso_provider_id, '00000000-0000-0000-0000-000000000000'::uuid), subject);
CREATE INDEX user_identities_user_idx ON user_identities (user_id);

-- Sessions store only the SHA-256 of the cookie token.
CREATE TABLE sessions (
    id            uuid PRIMARY KEY,
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    bytea       NOT NULL,
    auth_method   text        NOT NULL,
    mfa_verified  boolean     NOT NULL DEFAULT false,
    ip            inet,
    user_agent    text        NOT NULL DEFAULT '',
    location      text        NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    revoked_at    timestamptz
);
CREATE UNIQUE INDEX sessions_token_key ON sessions (token_hash);
CREATE INDEX sessions_user_idx ON sessions (user_id) WHERE revoked_at IS NULL;

-- Short-lived sign-in state: a password that passed but still needs a second
-- factor, a WebAuthn ceremony, an OIDC state/nonce. Keyed by a hashed token.
CREATE TABLE auth_challenges (
    id          uuid PRIMARY KEY,
    token_hash  bytea       NOT NULL,
    kind        text        NOT NULL,
    user_id     uuid        REFERENCES users(id) ON DELETE CASCADE,
    data        jsonb       NOT NULL DEFAULT '{}',
    attempts    integer     NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL
);
CREATE UNIQUE INDEX auth_challenges_token_key ON auth_challenges (token_hash);
CREATE INDEX auth_challenges_expires_idx ON auth_challenges (expires_at);

CREATE TABLE password_resets (
    id          uuid PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz
);
CREATE UNIQUE INDEX password_resets_token_key ON password_resets (token_hash);

CREATE TABLE email_changes (
    id          uuid PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    new_email   text        NOT NULL CHECK (new_email = lower(new_email)),
    token_hash  bytea       NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz
);
CREATE UNIQUE INDEX email_changes_token_key ON email_changes (token_hash);

-- Per-member notification preferences: one row per event type that differs from the default.
CREATE TABLE notification_preferences (
    user_id     uuid    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type  text    NOT NULL,
    in_app      boolean NOT NULL DEFAULT true,
    email       boolean NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, event_type)
);

-- +goose Down
DROP TABLE notification_preferences;
DROP TABLE email_changes;
DROP TABLE password_resets;
DROP TABLE auth_challenges;
DROP TABLE sessions;
DROP TABLE user_identities;
DROP TABLE sso_providers;
DROP TABLE user_passkeys;
DROP TABLE user_recovery_codes;
DROP TABLE users;
DROP TABLE instance_settings;
