-- The SMTP relay (Instance admin › Instance settings) and its Let's Encrypt
-- cache.

-- name: RelaySettings :one
SELECT r.enabled, r.hostname, r.starttls_port, r.tls_port, r.cert_source, r.cert_file, r.key_file, r.status, r.cert_not_after, r.cert_issuer,
       r.renew_error, r.renew_failed_at, r.enabled_by, r.enabled_at, r.unreachable_since, r.updated_at,
       coalesce(u.name, '')::text AS enabled_by_name
FROM smtp_relay r LEFT JOIN users u ON u.id = r.enabled_by;

-- name: RelayUpdate :exec
UPDATE smtp_relay SET enabled = @enabled, hostname = @hostname, starttls_port = @starttls_port, tls_port = @tls_port, cert_source = @cert_source,
    cert_file = @cert_file, key_file = @key_file, enabled_by = sqlc.narg(enabled_by), enabled_at = sqlc.narg(enabled_at),
    updated_by = sqlc.narg(updated_by), updated_at = @updated_at;

-- name: RelaySetStatus :exec
UPDATE smtp_relay SET status = @status, unreachable_since = sqlc.narg(unreachable_since);

-- name: RelaySetCert :exec
-- The certificate the relay serves, or why getting one failed.
UPDATE smtp_relay SET cert_not_after = sqlc.narg(cert_not_after), cert_issuer = @cert_issuer, renew_error = @renew_error, renew_failed_at = sqlc.narg(renew_failed_at);

-- name: RelayACMEGet :one
SELECT sealed FROM smtp_relay_acme WHERE key = @key;

-- name: RelayACMEPut :exec
INSERT INTO smtp_relay_acme (key, sealed, updated_at) VALUES (@key, @sealed, now())
ON CONFLICT (key) DO UPDATE SET sealed = excluded.sealed, updated_at = now();

-- name: RelayACMEDelete :exec
DELETE FROM smtp_relay_acme WHERE key = @key;
