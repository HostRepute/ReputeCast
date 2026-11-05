-- Scope checks shared by every domain (internal/access): a member, API key
-- or agent limited to some lists sees only the contacts on them.

-- name: AccessScopeContact :one
-- Whether one of a scoped caller's lists has the contact, in any status, as
-- Audience's list and search count it.
SELECT EXISTS (
  SELECT 1 FROM list_memberships WHERE contact_id = @contact_id AND list_id = ANY(@scope_lists::uuid[])
)::boolean AS in_scope;
