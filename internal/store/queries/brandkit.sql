-- The brand kit and its synced blocks (CO-11).

-- name: BrandKitGet :one
SELECT k.workspace_id, k.logo_light_id, k.logo_dark_id, k.logo_alt, k.style, k.website, k.updated_at,
       coalesce(u.name, '')::text AS updated_by_name
FROM brand_kits k
LEFT JOIN users u ON u.id = k.updated_by
WHERE k.workspace_id = @workspace_id;

-- name: BrandKitUpsert :exec
INSERT INTO brand_kits (workspace_id, logo_light_id, logo_dark_id, logo_alt, style, website, updated_by, updated_at)
VALUES (@workspace_id, sqlc.narg(logo_light_id), sqlc.narg(logo_dark_id), @logo_alt, @style, @website, sqlc.narg(updated_by), @updated_at)
ON CONFLICT (workspace_id) DO UPDATE SET logo_light_id = EXCLUDED.logo_light_id, logo_dark_id = EXCLUDED.logo_dark_id,
    logo_alt = EXCLUDED.logo_alt, style = EXCLUDED.style, website = EXCLUDED.website,
    updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: SyncedBlocksList :many
-- The Header, the Footer and the named blocks (R4.A3), oldest first.
SELECT b.kind, b.name, b.mjml, b.version, b.updated_at, coalesce(u.name, '')::text AS updated_by_name
FROM synced_blocks b
LEFT JOIN users u ON u.id = b.updated_by
WHERE b.workspace_id = @workspace_id
ORDER BY b.created_at, b.kind;

-- name: SyncedBlockGet :one
-- Locks the block for a change (inside a transaction).
SELECT b.kind, b.name, b.mjml, b.version, b.updated_at, coalesce(u.name, '')::text AS updated_by_name
FROM synced_blocks b
LEFT JOIN users u ON u.id = b.updated_by
WHERE b.workspace_id = @workspace_id AND b.kind = @kind
FOR UPDATE OF b;

-- name: SyncedBlockInsert :exec
INSERT INTO synced_blocks (id, workspace_id, kind, name, mjml, version, updated_by, updated_at, created_at)
VALUES (@id, @workspace_id, @kind, @name, @mjml, 1, sqlc.narg(updated_by), @now, @now);

-- name: SyncedBlockUpdate :exec
UPDATE synced_blocks SET mjml = @mjml, name = @name, version = version + 1, updated_by = sqlc.narg(updated_by), updated_at = @now
WHERE workspace_id = @workspace_id AND kind = @kind;

-- name: SyncedBlockDelete :execrows
-- A named block (R4.A3); the Header and Footer stay.
DELETE FROM synced_blocks WHERE workspace_id = @workspace_id AND kind = @kind AND kind NOT IN ('header', 'footer');
