-- name: CreateUser :one
INSERT INTO users (id, email, name, timezone, password_hash, password_set_at, must_change_password,
                   is_instance_admin, break_glass, avatar_color, twofa_grace_until)
VALUES (@id, @email, @name, @timezone, @password_hash, @password_set_at, @must_change_password,
        @is_instance_admin, @break_glass, @avatar_color, @twofa_grace_until)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = @id;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = lower(@email::text);

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at;

-- name: SetUserPassword :exec
UPDATE users SET password_hash = @password_hash, password_set_at = now(),
                 must_change_password = @must_change_password, updated_at = now()
WHERE id = @id;

-- name: UpdateUserProfile :one
UPDATE users SET name = @name, timezone = @timezone, avatar_color = @avatar_color, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: SetUserEmail :exec
UPDATE users SET email = @email, updated_at = now() WHERE id = @id;

-- name: SetUserTOTP :exec
UPDATE users SET totp_secret_enc = @totp_secret_enc, totp_enabled_at = @totp_enabled_at, updated_at = now()
WHERE id = @id;

-- name: SetUserInstanceAdmin :exec
UPDATE users SET is_instance_admin = @is_instance_admin, updated_at = now() WHERE id = @id;

-- name: SetUserBreakGlass :exec
UPDATE users SET break_glass = @break_glass, updated_at = now() WHERE id = @id;

-- name: SetUserStatus :exec
UPDATE users SET status = @status,
                 deactivated_at = CASE WHEN @status::text = 'deactivated' THEN now() ELSE NULL END,
                 updated_at = now()
WHERE id = @id;

-- name: SetUserTwoFAGrace :exec
UPDATE users SET twofa_grace_until = @twofa_grace_until, updated_at = now() WHERE id = @id;

-- name: RecordSignInFailure :one
UPDATE users
SET failed_attempts = failed_attempts + 1,
    locked_until = CASE WHEN failed_attempts + 1 >= @max_attempts::int
                        THEN now() + make_interval(mins => @lock_minutes::int)
                        ELSE locked_until END,
    updated_at = now()
WHERE id = @id
RETURNING failed_attempts, locked_until;

-- name: RecordSignInSuccess :exec
UPDATE users SET failed_attempts = 0, locked_until = NULL, last_sign_in_at = now(), last_active_at = now()
WHERE id = @id;

-- name: TouchUserActive :exec
UPDATE users SET last_active_at = now() WHERE id = @id AND (last_active_at IS NULL OR last_active_at < now() - interval '1 minute');

-- name: ResetUserTwoFactor :exec
UPDATE users SET totp_secret_enc = NULL, totp_enabled_at = NULL, updated_at = now() WHERE id = @id;

-- name: CreateRecoveryCode :exec
INSERT INTO user_recovery_codes (id, user_id, code_hash) VALUES (@id, @user_id, @code_hash);

-- name: DeleteRecoveryCodes :exec
DELETE FROM user_recovery_codes WHERE user_id = @user_id;

-- name: UseRecoveryCode :one
UPDATE user_recovery_codes SET used_at = now()
WHERE user_id = @user_id AND code_hash = @code_hash AND used_at IS NULL
RETURNING id;

-- name: CountUnusedRecoveryCodes :one
SELECT count(*) FROM user_recovery_codes WHERE user_id = @user_id AND used_at IS NULL;

-- name: ListPasskeys :many
SELECT * FROM user_passkeys WHERE user_id = @user_id ORDER BY created_at;

-- name: GetPasskeyByCredential :one
SELECT * FROM user_passkeys WHERE credential_id = @credential_id;

-- name: CreatePasskey :one
INSERT INTO user_passkeys (id, user_id, name, credential_id, public_key, attestation_type, aaguid,
                           sign_count, transports, backup_eligible, backup_state)
VALUES (@id, @user_id, @name, @credential_id, @public_key, @attestation_type, @aaguid,
        @sign_count, @transports, @backup_eligible, @backup_state)
RETURNING *;

-- name: TouchPasskey :exec
UPDATE user_passkeys SET sign_count = @sign_count, backup_state = @backup_state, last_used_at = now() WHERE id = @id;

-- name: RenamePasskey :exec
UPDATE user_passkeys SET name = @name WHERE id = @id AND user_id = @user_id;

-- name: DeletePasskey :execrows
DELETE FROM user_passkeys WHERE id = @id AND user_id = @user_id;

-- name: ListIdentities :many
SELECT * FROM user_identities WHERE user_id = @user_id ORDER BY created_at;

-- name: GetIdentity :one
SELECT * FROM user_identities
WHERE kind = @kind AND coalesce(sso_provider_id, '00000000-0000-0000-0000-000000000000'::uuid) = coalesce(sqlc.narg(sso_provider_id)::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
  AND subject = @subject;

-- name: CreateIdentity :one
INSERT INTO user_identities (id, user_id, kind, sso_provider_id, subject, email)
VALUES (@id, @user_id, @kind, @sso_provider_id, @subject, @email)
RETURNING *;

-- name: TouchIdentity :exec
UPDATE user_identities SET last_used_at = now() WHERE id = @id;

-- name: DeleteIdentity :execrows
DELETE FROM user_identities WHERE id = @id AND user_id = @user_id;
