-- name: LinkDomainListings :many
SELECT domain, lists, source, checked_at FROM link_domain_listings WHERE domain = ANY(@domains::text[]);

-- name: UpsertLinkDomainListing :exec
INSERT INTO link_domain_listings (domain, lists, source, checked_at) VALUES (@domain, @lists, @source, @checked_at)
ON CONFLICT (domain) DO UPDATE SET lists = EXCLUDED.lists, source = EXCLUDED.source, checked_at = EXCLUDED.checked_at;
