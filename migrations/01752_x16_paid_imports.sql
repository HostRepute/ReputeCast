-- +goose Up
-- Importing paid members (design expansion phase 6, M26): an export from
-- Substack, Ghost, beehiiv or Kit, or any CSV with an email column, matched
-- row by row with the connected Stripe account's customers and
-- subscriptions (read-only). Nothing changes in Stripe. The file is dropped
-- once the import is done or canceled.

CREATE TABLE earn_imports (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    mode         text        NOT NULL CHECK (mode IN ('live','test')),
    source       text        NOT NULL CHECK (source IN ('substack','ghost','beehiiv','kit','csv')),
    status       text        NOT NULL DEFAULT 'uploaded' CHECK (status IN ('uploaded','matched','importing','done','failed','canceled')),
    file_name    text        NOT NULL DEFAULT '',
    file_size    bigint      NOT NULL DEFAULT 0,
    -- The columns found and what each is used as:
    -- [{"name":"email","usedAs":"email","example":"priya.n@gmail.com"}].
    columns      jsonb       NOT NULL DEFAULT '[]',
    -- The column picked for each use: {"email":"email","customer":"stripe_customer_id",…}.
    mapping      jsonb       NOT NULL DEFAULT '{}',
    -- What to do with the groups: {"noSubscription":"complimentary","prices":{…},"conflicts":{…}}.
    decisions    jsonb       NOT NULL DEFAULT '{}',
    send_welcome boolean     NOT NULL DEFAULT false,
    total        integer     NOT NULL DEFAULT 0,
    done_rows    integer     NOT NULL DEFAULT 0,
    -- The summary: counts per group, contacts merged and new, prices mapped.
    stats        jsonb       NOT NULL DEFAULT '{}',
    error        text        NOT NULL DEFAULT '',
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    matched_at   timestamptz,
    started_at   timestamptz,
    finished_at  timestamptz
);
CREATE INDEX earn_imports_ws_idx ON earn_imports (workspace_id, created_at DESC);

CREATE TABLE earn_import_rows (
    import_id              uuid    NOT NULL REFERENCES earn_imports(id) ON DELETE CASCADE,
    row_no                 integer NOT NULL,
    email                  text    NOT NULL DEFAULT '',
    name                   text    NOT NULL DEFAULT '',
    customer_id            text    NOT NULL DEFAULT '',
    plan                   text    NOT NULL DEFAULT '',
    paid_through           date,
    -- After matching: active, ended, none or conflict, and how it matched
    -- (customer or email).
    match                  text    NOT NULL DEFAULT '' CHECK (match IN ('','active','ended','none','conflict','invalid')),
    matched_by             text    NOT NULL DEFAULT '' CHECK (matched_by IN ('','customer','email')),
    stripe_customer_id     text    NOT NULL DEFAULT '',
    stripe_subscription_id text    NOT NULL DEFAULT '',
    stripe_price_id        text    NOT NULL DEFAULT '',
    -- The matched subscription's state, and a conflict's candidates.
    subscription           jsonb   NOT NULL DEFAULT '{}',
    candidates             jsonb   NOT NULL DEFAULT '[]',
    -- After the import: what became of the row.
    result                 text    NOT NULL DEFAULT '',
    contact_id             uuid,
    PRIMARY KEY (import_id, row_no)
);

-- The uploaded file, kept apart so listing imports never reads it.
CREATE TABLE earn_import_files (
    import_id uuid  PRIMARY KEY REFERENCES earn_imports(id) ON DELETE CASCADE,
    data      bytea NOT NULL
);

ALTER TABLE paid_subscriptions ADD CONSTRAINT paid_subscriptions_import_fk FOREIGN KEY (import_id) REFERENCES earn_imports(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE paid_subscriptions DROP CONSTRAINT paid_subscriptions_import_fk;
DROP TABLE earn_import_files;
DROP TABLE earn_import_rows;
DROP TABLE earn_imports;
