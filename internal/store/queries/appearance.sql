-- Appearance (M54): the member's light or dark theme (internal/appearance).

-- name: AppearanceSetTheme :one
UPDATE users SET theme = sqlc.narg(theme), updated_at = now()
WHERE id = @id
RETURNING *;
