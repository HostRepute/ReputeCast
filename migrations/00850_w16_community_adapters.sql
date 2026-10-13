-- +goose Up
-- Community adapters that run in their own process and speak the adapter
-- protocol (W16, docs/adapters.md "Community adapters"). installed_adapters
-- gains the "remote" kind; adapter_runtimes keeps, per installed remote
-- adapter, its registry key, where its process listens, its signing secret
-- (sealed with the instance key, write-only; the previous one stays valid
-- for a while after a rotation) and the last health check of the process.

ALTER TABLE installed_adapters DROP CONSTRAINT installed_adapters_kind_check;
ALTER TABLE installed_adapters ADD CONSTRAINT installed_adapters_kind_check CHECK (kind IN ('http','wasm','remote'));

CREATE TABLE adapter_runtimes (
    adapter_id           uuid        PRIMARY KEY REFERENCES installed_adapters (id) ON DELETE CASCADE,
    -- Registry key: "community:" + the manifest id.
    adapter_key          text        NOT NULL,
    -- Base URL of the adapter process; '' until the operator sets it.
    endpoint_url         text        NOT NULL DEFAULT '',
    secret_enc           bytea       NOT NULL,
    secret_issued_at     timestamptz NOT NULL DEFAULT now(),
    previous_secret_enc  bytea,
    previous_valid_until timestamptz,
    checked_at           timestamptz,
    check_ok             boolean,
    check_detail         text        NOT NULL DEFAULT '',
    running_version      text        NOT NULL DEFAULT '',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX adapter_runtimes_key_idx ON adapter_runtimes (adapter_key);

-- +goose Down
DROP TABLE adapter_runtimes;
DELETE FROM installed_adapters WHERE kind = 'remote';
ALTER TABLE installed_adapters DROP CONSTRAINT installed_adapters_kind_check;
ALTER TABLE installed_adapters ADD CONSTRAINT installed_adapters_kind_check CHECK (kind IN ('http','wasm'));
