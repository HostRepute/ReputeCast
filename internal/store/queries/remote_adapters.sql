-- Community adapters of kind "remote" and their runtime (internal/providers/remote).

-- name: RemoteAdapterList :many
SELECT ia.id, ia.name, ia.version, ia.maintainer, ia.source_url, ia.manifest, ia.installed_at,
       r.adapter_key, r.endpoint_url, r.secret_enc, r.secret_issued_at, r.previous_secret_enc, r.previous_valid_until,
       r.checked_at, r.check_ok, r.check_detail, r.running_version, r.updated_at
FROM installed_adapters ia
JOIN adapter_runtimes r ON r.adapter_id = ia.id
WHERE ia.kind = 'remote'
ORDER BY r.adapter_key;

-- name: RemoteAdapterGet :one
SELECT ia.id, ia.name, ia.version, ia.maintainer, ia.source_url, ia.manifest, ia.installed_at,
       r.adapter_key, r.endpoint_url, r.secret_enc, r.secret_issued_at, r.previous_secret_enc, r.previous_valid_until,
       r.checked_at, r.check_ok, r.check_detail, r.running_version, r.updated_at
FROM installed_adapters ia
JOIN adapter_runtimes r ON r.adapter_id = ia.id
WHERE ia.id = $1 AND ia.kind = 'remote';

-- name: RemoteAdapterGetByKey :one
SELECT ia.id, ia.name, ia.version, ia.maintainer, ia.source_url, ia.manifest, ia.installed_at,
       r.adapter_key, r.endpoint_url, r.secret_enc, r.secret_issued_at, r.previous_secret_enc, r.previous_valid_until,
       r.checked_at, r.check_ok, r.check_detail, r.running_version, r.updated_at
FROM installed_adapters ia
JOIN adapter_runtimes r ON r.adapter_id = ia.id
WHERE r.adapter_key = $1 AND ia.kind = 'remote';

-- name: RemoteAdapterInsert :exec
INSERT INTO installed_adapters (id, name, kind, version, maintainer, source_url, manifest, latest_version)
VALUES (@id, @name, 'remote', @version, @maintainer, @source_url, @manifest, sqlc.narg(latest_version));

-- name: RemoteAdapterUpdateManifest :exec
UPDATE installed_adapters
SET name = @name, version = @version, maintainer = @maintainer, source_url = @source_url, manifest = @manifest, latest_version = NULL
WHERE id = @id AND kind = 'remote';

-- name: RemoteAdapterDelete :execrows
DELETE FROM installed_adapters WHERE id = $1 AND kind = 'remote';

-- name: CommunityAdapterList :many
-- Every installed community adapter, any kind: the runtime registers the
-- kinds it runs, and registry keys must not collide across kinds.
SELECT id, name, kind, version, maintainer, manifest
FROM installed_adapters
ORDER BY installed_at, name;

-- name: RemoteAdapterConnected :one
SELECT count(*)::bigint FROM providers WHERE adapter = $1 AND disconnected_at IS NULL;

-- name: RemoteRuntimeCreate :exec
INSERT INTO adapter_runtimes (adapter_id, adapter_key, endpoint_url, secret_enc)
VALUES (@adapter_id, @adapter_key, @endpoint_url, @secret_enc);

-- name: RemoteRuntimeSetEndpoint :exec
UPDATE adapter_runtimes SET endpoint_url = @endpoint_url, updated_at = now() WHERE adapter_id = @adapter_id;

-- name: RemoteRuntimeSetSecret :exec
UPDATE adapter_runtimes
SET previous_secret_enc = secret_enc, previous_valid_until = @previous_valid_until,
    secret_enc = @secret_enc, secret_issued_at = @issued_at, updated_at = now()
WHERE adapter_id = @adapter_id;

-- name: RemoteRuntimeTouch :exec
UPDATE adapter_runtimes SET updated_at = now() WHERE adapter_id = $1;

-- name: RemoteRuntimeRecordCheck :exec
UPDATE adapter_runtimes
SET checked_at = @checked_at, check_ok = @check_ok, check_detail = @check_detail, running_version = @running_version
WHERE adapter_id = @adapter_id;
