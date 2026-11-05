-- Scope checks shared by every domain (internal/access): a member, API key
-- or agent limited to some lists sees only the contacts on them, and one
-- limited to some sender domains sends only from those.

-- name: AccessScopeContact :one
-- Whether one of a scoped caller's lists has the contact, in any status, as
-- Audience's list and search count it.
SELECT EXISTS (
  SELECT 1 FROM list_memberships WHERE contact_id = @contact_id AND list_id = ANY(@scope_lists::uuid[])
)::boolean AS in_scope;

-- name: AccessScopeCampaign :one
-- A campaign's audience and the domain it sends from, to check that a send,
-- a schedule or a resume stays within the caller's lists and sender
-- domains. Its sender's sending domain, and its address for a sender whose
-- domain is gone.
SELECT c.audience, coalesce(d.domain, '')::text AS sender_domain, coalesce(s.email, '')::text AS sender_email
FROM campaigns c
LEFT JOIN senders s ON s.id = c.sender_id
LEFT JOIN sending_domains d ON d.id = s.domain_id
WHERE c.workspace_id = @workspace_id AND c.id = @id;
