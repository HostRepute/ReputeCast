-- name: CreateSession :one
INSERT INTO sessions (id, user_id, token_hash, auth_method, mfa_verified, ip, user_agent, location, expires_at)
VALUES (@id, @user_id, @token_hash, @auth_method, @mfa_verified, @ip, @user_agent, @location, @expires_at)
RETURNING *;

-- name: GetSessionByToken :one
SELECT * FROM sessions WHERE token_hash = @token_hash AND revoked_at IS NULL AND expires_at > now();

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = now() WHERE id = @id AND last_seen_at < now() - interval '30 seconds';

-- name: ListUserSessions :many
SELECT * FROM sessions WHERE user_id = @user_id AND revoked_at IS NULL AND expires_at > now()
ORDER BY last_seen_at DESC;

-- name: RevokeSession :execrows
UPDATE sessions SET revoked_at = now() WHERE id = @id AND user_id = @user_id AND revoked_at IS NULL;

-- name: RevokeOtherSessions :execrows
UPDATE sessions SET revoked_at = now() WHERE user_id = @user_id AND id <> @keep_id AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :execrows
UPDATE sessions SET revoked_at = now() WHERE user_id = @user_id AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at < now() - interval '30 days' OR revoked_at < now() - interval '30 days';

-- name: CreateChallenge :one
INSERT INTO auth_challenges (id, token_hash, kind, user_id, data, expires_at)
VALUES (@id, @token_hash, @kind, @user_id, @data, @expires_at)
RETURNING *;

-- name: GetChallenge :one
SELECT * FROM auth_challenges WHERE token_hash = @token_hash AND kind = @kind AND expires_at > now();

-- name: BumpChallengeAttempts :one
UPDATE auth_challenges SET attempts = attempts + 1 WHERE id = @id RETURNING attempts;

-- name: DeleteChallenge :exec
DELETE FROM auth_challenges WHERE id = @id;

-- name: DeleteExpiredChallenges :execrows
DELETE FROM auth_challenges WHERE expires_at < now();

-- name: CreatePasswordReset :one
INSERT INTO password_resets (id, user_id, token_hash, expires_at)
VALUES (@id, @user_id, @token_hash, @expires_at)
RETURNING *;

-- name: ConsumePasswordReset :one
UPDATE password_resets SET used_at = now()
WHERE token_hash = @token_hash AND used_at IS NULL AND expires_at > now()
RETURNING *;

-- name: GetPasswordReset :one
SELECT * FROM password_resets WHERE token_hash = @token_hash;
