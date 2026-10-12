-- Session timing for the app's "Session ending in 2 minutes" banner and its
-- Extend action (internal/sessionext).

-- name: GetSessionTiming :one
SELECT id, created_at, last_seen_at, expires_at
FROM sessions
WHERE id = @id AND revoked_at IS NULL AND expires_at > now();

-- name: ExtendSession :one
-- Restarts the idle clock now (the middleware only touches sessions every
-- 30 seconds). The absolute lifetime (expires_at) never moves.
UPDATE sessions SET last_seen_at = now()
WHERE id = @id AND user_id = @user_id AND revoked_at IS NULL AND expires_at > now()
RETURNING id, created_at, last_seen_at, expires_at;
