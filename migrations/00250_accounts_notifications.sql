-- +goose Up
-- Accounts, sign-in and notifications (add-on spec 5.1–5.6, 5.16, 3.12).

-- Passkeys keep the full WebAuthn credential record (public data only: the
-- credential id, the public key, flags and counters) so assertions can be
-- verified; the name, attachment and authenticator provider are shown in
-- Account & security.
ALTER TABLE user_passkeys
    ADD COLUMN credential jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN attachment text  NOT NULL DEFAULT '' CHECK (attachment IN ('', 'platform', 'cross-platform')),
    ADD COLUMN provider   text  NOT NULL DEFAULT '';

-- webauthn_id is a random WebAuthn user handle (never the email or the row id).
-- totp_label names the authenticator app when known ("1Password").
-- recovery_codes_saved_at records the "I have saved these codes" confirmation
-- of the first sign-in. avatar_updated_at versions the avatar URL.
ALTER TABLE users
    ADD COLUMN webauthn_id             bytea,
    ADD COLUMN totp_label              text NOT NULL DEFAULT '',
    ADD COLUMN recovery_codes_saved_at timestamptz,
    ADD COLUMN avatar_updated_at       timestamptz;
CREATE UNIQUE INDEX users_webauthn_id_key ON users (webauthn_id) WHERE webauthn_id IS NOT NULL;

-- Profile photos, re-encoded to a 256 px PNG on upload (metadata stripped).
CREATE TABLE user_avatars (
    user_id      uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    content_type text        NOT NULL CHECK (content_type IN ('image/png', 'image/jpeg')),
    data         bytea       NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Sessions: device details override the user agent parsing (set for sessions
-- created by trusted code such as the demo), the passkey a session signed in
-- with ("used now · this session").
ALTER TABLE sessions
    ADD COLUMN device     text NOT NULL DEFAULT '',
    ADD COLUMN os         text NOT NULL DEFAULT '',
    ADD COLUMN browser    text NOT NULL DEFAULT '',
    ADD COLUMN passkey_id uuid REFERENCES user_passkeys(id) ON DELETE SET NULL;

-- Identities: the account name at the provider ("linh-nguyen") and details
-- such as the GitHub organization ("acme-eng").
ALTER TABLE user_identities
    ADD COLUMN username text  NOT NULL DEFAULT '',
    ADD COLUMN details  jsonb NOT NULL DEFAULT '{}';

-- Password reset links remember who created them: the person (email), an
-- instance admin, or the server CLI.
ALTER TABLE password_resets
    ADD COLUMN source text NOT NULL DEFAULT 'email' CHECK (source IN ('email', 'cli', 'admin'));
CREATE INDEX password_resets_user_idx ON password_resets (user_id, created_at DESC);

CREATE INDEX email_changes_user_idx ON email_changes (user_id, created_at DESC);

-- Invitations: when the invitee last asked the inviter for a new one.
ALTER TABLE invitations ADD COLUMN reinvite_requested_at timestamptz;

-- First sign-in, step 4: when the instance basics were confirmed.
ALTER TABLE instance_settings ADD COLUMN basics_confirmed_at timestamptz;

-- Notification delivery settings beyond the per-event rows: quiet hours for
-- email (minutes after local midnight; NULL = none) and the email digest.
CREATE TABLE notification_settings (
    user_id        uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    quiet_start    smallint CHECK (quiet_start BETWEEN 0 AND 1439),
    quiet_end      smallint CHECK (quiet_end BETWEEN 0 AND 1439),
    digest         text NOT NULL DEFAULT 'immediate' CHECK (digest IN ('immediate', 'hourly', 'daily')),
    last_digest_at timestamptz,
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CHECK ((quiet_start IS NULL) = (quiet_end IS NULL))
);

-- Notifications: a short line for the bell popover, whether the item shows in
-- the notification center (in-app preference) and the email delivery state
-- (NULL when the person didn't ask for email).
ALTER TABLE notifications
    ADD COLUMN summary        text    NOT NULL DEFAULT '',
    ADD COLUMN in_app         boolean NOT NULL DEFAULT true,
    ADD COLUMN email_status   text CHECK (email_status IN ('pending', 'sent', 'skipped', 'failed')),
    ADD COLUMN email_attempts integer NOT NULL DEFAULT 0,
    ADD COLUMN emailed_at     timestamptz;
CREATE INDEX notifications_email_pending_idx ON notifications (user_id, created_at) WHERE email_status = 'pending';
CREATE INDEX notifications_ws_idx ON notifications (user_id, workspace_id, created_at DESC);

-- +goose Down
DROP INDEX notifications_ws_idx;
DROP INDEX notifications_email_pending_idx;
ALTER TABLE notifications
    DROP COLUMN emailed_at,
    DROP COLUMN email_attempts,
    DROP COLUMN email_status,
    DROP COLUMN in_app,
    DROP COLUMN summary;
DROP TABLE notification_settings;
ALTER TABLE instance_settings DROP COLUMN basics_confirmed_at;
ALTER TABLE invitations DROP COLUMN reinvite_requested_at;
DROP INDEX email_changes_user_idx;
DROP INDEX password_resets_user_idx;
ALTER TABLE password_resets DROP COLUMN source;
ALTER TABLE user_identities DROP COLUMN details, DROP COLUMN username;
ALTER TABLE sessions DROP COLUMN passkey_id, DROP COLUMN browser, DROP COLUMN os, DROP COLUMN device;
DROP TABLE user_avatars;
DROP INDEX users_webauthn_id_key;
ALTER TABLE users
    DROP COLUMN avatar_updated_at,
    DROP COLUMN recovery_codes_saved_at,
    DROP COLUMN totp_label,
    DROP COLUMN webauthn_id;
ALTER TABLE user_passkeys DROP COLUMN provider, DROP COLUMN attachment, DROP COLUMN credential;
