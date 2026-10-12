-- name: GetSignInThrottle :one
SELECT * FROM sign_in_throttle WHERE email = @email;

-- name: RecordSignInThrottleFailure :one
INSERT INTO sign_in_throttle (email, failures, locked_until, updated_at)
VALUES (@email, 1, NULL, now())
ON CONFLICT (email) DO UPDATE
SET failures = CASE WHEN sign_in_throttle.locked_until IS NOT NULL AND sign_in_throttle.locked_until < now()
                    THEN 1 ELSE sign_in_throttle.failures + 1 END,
    locked_until = CASE WHEN (CASE WHEN sign_in_throttle.locked_until IS NOT NULL AND sign_in_throttle.locked_until < now()
                                   THEN 1 ELSE sign_in_throttle.failures + 1 END) >= @max_attempts::int
                        THEN now() + make_interval(mins => @lock_minutes::int)
                        WHEN sign_in_throttle.locked_until < now() THEN NULL
                        ELSE sign_in_throttle.locked_until END,
    updated_at = now()
RETURNING *;

-- name: ClearSignInThrottle :exec
DELETE FROM sign_in_throttle WHERE email = @email;

-- name: DeleteStaleThrottle :execrows
DELETE FROM sign_in_throttle WHERE updated_at < now() - interval '1 day';
