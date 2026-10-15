-- Sidebar sections (Settings › General): the sections a workspace hides.

-- name: SidebarHiddenSectionsForUpdate :one
SELECT hidden_sections FROM workspaces WHERE id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: SidebarSetHiddenSections :exec
UPDATE workspaces SET hidden_sections = @hidden_sections::text[], updated_at = now()
WHERE id = @id AND deleted_at IS NULL;
