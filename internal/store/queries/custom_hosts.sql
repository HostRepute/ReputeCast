-- Hosts the instance serves for a workspace: the public pages and website
-- domain, the SMS short-link domain, a link tracking domain and the push
-- site of a publication website. A CNAME or matching addresses only show
-- that a host points at this server, not whose host it is, so a host
-- belongs to one workspace: the first to use it.

-- name: CustomHostLock :exec
-- Saving a host takes this lock before CustomHostClaims, so two workspaces
-- saving the same host at once can't both find it free.
SELECT pg_advisory_xact_lock(hashtextextended('custom_host:' || @host::text, 0));

-- name: CustomHostClaims :many
-- The uses of host other than workspace_id's, whatever their status: other
-- workspaces' (a deleted workspace keeps its hosts until it's purged, as
-- the custom domain's unique index does) and the instance's own SMTP relay
-- (purpose "instance"). verified says whether the use was proven there,
-- and since when (NULL when nobody recorded it, as for tracking domains
-- verified before migration 03446). A publication website's push site is
-- its pages' host, verified when it was added.
SELECT 'pages'::text AS purpose, (s.domain_status = 'verified')::boolean AS verified,
       s.domain_verified_at AS since
FROM public_page_settings s
WHERE s.custom_domain = @host::text AND s.workspace_id <> @workspace_id
UNION ALL
SELECT 'sms_links', m.short_domain_status = 'verified', m.short_domain_verified_at
FROM sms_settings m
WHERE lower(m.short_domain) = @host::text AND m.workspace_id <> @workspace_id
UNION ALL
SELECT 'link_tracking', d.tracking_status = 'verified', d.tracking_verified_at
FROM sending_domains d
WHERE lower(d.tracking_domain) = @host::text AND d.workspace_id <> @workspace_id
UNION ALL
SELECT 'push_site', true, p.added_at
FROM push_sites p
WHERE p.kind = 'publication' AND p.removed_at IS NULL AND lower(p.host) = @host::text AND p.workspace_id <> @workspace_id
UNION ALL
SELECT 'instance', true, NULL::timestamptz
FROM smtp_relay r
WHERE r.hostname <> '' AND lower(r.hostname) = @host::text;
