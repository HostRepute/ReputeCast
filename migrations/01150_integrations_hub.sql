-- +goose Up
-- Integrations hub (M16): connections to Zapier, Make, n8n and WordPress
-- sites. Each connection has its own API key, limited to the operations of
-- the triggers and actions turned on; calls are kept for 7 days, the
-- workflows seen for 30 days, and the webhooks an integration creates are
-- managed by it.

CREATE TABLE integrations (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind            text        NOT NULL CHECK (kind IN ('zapier','make','n8n','wordpress')),
    -- "Acme ops", or the site's host for WordPress ("blog.acme.io").
    label           text        NOT NULL DEFAULT '',
    -- The catalog keys of the triggers and actions turned on, and the ones
    -- turned off when connecting ("Turned off when connecting").
    items           text[]      NOT NULL DEFAULT '{}',
    off_items       text[]      NOT NULL DEFAULT '{}',
    -- {"address":"https://n8n.acme.io"} for n8n.
    settings        jsonb       NOT NULL DEFAULT '{}',
    connected_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    connected_at    timestamptz NOT NULL DEFAULT now(),
    first_call_at   timestamptz,
    last_call_at    timestamptz,
    -- The reasons it needs attention, computed from what ReputeCast already
    -- knows (no polling): [{"code":"site_silent","since":"…"}]. Notified
    -- holds the codes members were told about.
    attention       jsonb       NOT NULL DEFAULT '[]',
    notified        text[]      NOT NULL DEFAULT '{}',
    disconnected_at timestamptz,
    disconnected_by uuid        REFERENCES users(id) ON DELETE SET NULL
);
-- One connection per automation platform; one per site for WordPress.
CREATE UNIQUE INDEX integrations_platform_key ON integrations (workspace_id, kind) WHERE disconnected_at IS NULL AND kind <> 'wordpress';
CREATE UNIQUE INDEX integrations_site_key ON integrations (workspace_id, lower(label)) WHERE disconnected_at IS NULL AND kind = 'wordpress';
CREATE INDEX integrations_workspace_idx ON integrations (workspace_id, connected_at);

-- An integration key belongs to one integration. A rotated key stays valid
-- until grace_until (24 hours) although it is revoked.
ALTER TABLE api_keys ADD COLUMN integration_id uuid REFERENCES integrations(id) ON DELETE SET NULL;
ALTER TABLE api_keys ADD COLUMN grace_until timestamptz;
CREATE INDEX api_keys_integration_idx ON api_keys (integration_id, created_at DESC) WHERE integration_id IS NOT NULL;

-- Every call made with an integration key, kept 7 days. It isn't the audit
-- log: reads are only here, and state changes are also audited. Contacts are
-- kept by id; addresses in the bodies are shown by permission.
CREATE TABLE integration_calls (
    id             uuid PRIMARY KEY,
    integration_id uuid        NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    api_key_id     uuid,
    occurred_at    timestamptz NOT NULL DEFAULT now(),
    -- X-ReputeCast-Workflow: the workflow's id and name, when the platform sends them.
    workflow_key   text        NOT NULL DEFAULT '',
    workflow_name  text        NOT NULL DEFAULT '',
    operation      text        NOT NULL,
    -- The catalog key of the trigger or action ("upsert_contact").
    item           text        NOT NULL DEFAULT '',
    method         text        NOT NULL,
    path           text        NOT NULL,
    status         integer     NOT NULL,
    code           text        NOT NULL DEFAULT '',
    detail         text        NOT NULL DEFAULT '',
    duration_ms    integer     NOT NULL DEFAULT 0,
    -- What the call was about: {"text":"payment-failed to {who}","email":"…","list":"Beta Program"}.
    target         jsonb       NOT NULL DEFAULT '{}',
    contact_id     uuid,
    -- An optional note on how an error was resolved (otherwise computed).
    note           text        NOT NULL DEFAULT '',
    request        jsonb,
    response       jsonb,
    ip             inet,
    client         text        NOT NULL DEFAULT ''
);
CREATE INDEX integration_calls_time_idx ON integration_calls (integration_id, occurred_at DESC);
CREATE INDEX integration_calls_purge_idx ON integration_calls (occurred_at);

-- The workflows (Zaps, scenarios, n8n workflows) seen in the last 30 days.
CREATE TABLE integration_workflows (
    integration_id uuid        NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    -- The platform's id, else the name; '' for calls that send neither.
    key            text        NOT NULL,
    external_id    text        NOT NULL DEFAULT '',
    name           text        NOT NULL DEFAULT '',
    -- What it uses: [{"item":"upsert_contact","detail":"Beta Program · tag beta-applicant"}]
    uses           jsonb       NOT NULL DEFAULT '[]',
    first_seen_at  timestamptz NOT NULL DEFAULT now(),
    last_seen_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (integration_id, key)
);

-- Webhooks an integration created through its REST hooks: they can't be
-- edited by hand and go when the integration disconnects.
CREATE TABLE integration_webhooks (
    webhook_id     uuid        PRIMARY KEY REFERENCES webhooks(id) ON DELETE CASCADE,
    integration_id uuid        NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    api_key_id     uuid,
    event          text        NOT NULL,
    workflow_key   text        NOT NULL DEFAULT '',
    workflow_name  text        NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX integration_webhooks_integration_idx ON integration_webhooks (integration_id);

-- WordPress sites: what the plugin reports, the last "Check now" and the
-- post-draft rule.
CREATE TABLE wordpress_sites (
    integration_id    uuid        PRIMARY KEY REFERENCES integrations(id) ON DELETE CASCADE,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    -- What was typed when connecting ("acme.io"); the plugin reports site_url.
    address           text        NOT NULL,
    site_url          text        NOT NULL DEFAULT '',
    name              text        NOT NULL DEFAULT '',
    plugin_version    text        NOT NULL DEFAULT '',
    wp_version        text        NOT NULL DEFAULT '',
    php_version       text        NOT NULL DEFAULT '',
    feed_url          text        NOT NULL DEFAULT '',
    language_plugin   text        NOT NULL DEFAULT '',
    -- [{"id":"3","name":"Announcements","slug":"announcements"}]
    categories        jsonb       NOT NULL DEFAULT '[]',
    -- [{"form":"frm_8c2e1a","kind":"block","places":[{"title":"Sidebar","url":"…"}]}]
    forms             jsonb       NOT NULL DEFAULT '[]',
    first_report_at   timestamptz,
    last_report_at    timestamptz,
    last_check_at     timestamptz,
    last_check_ok     boolean,
    last_check_detail text        NOT NULL DEFAULT '',
    -- Posts in these categories become a draft for the audience, with the
    -- template and sender; notify_user_id is told.
    rule_enabled      boolean     NOT NULL DEFAULT false,
    -- {"categories":["announcements"],"listIds":["…"],"templateId":"…","senderId":"…","notifyUserId":"…"}
    rule              jsonb       NOT NULL DEFAULT '{}',
    rule_updated_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    rule_updated_at   timestamptz
);

-- Drafts created from posts, one per post.
CREATE TABLE wordpress_drafts (
    id             uuid        PRIMARY KEY,
    integration_id uuid        NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id    uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    post_id        text        NOT NULL,
    post_title     text        NOT NULL,
    post_url       text        NOT NULL DEFAULT '',
    published_at   timestamptz,
    notified_user  uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (integration_id, post_id)
);
CREATE INDEX wordpress_drafts_recent_idx ON wordpress_drafts (integration_id, created_at DESC);

-- The latest release of the apps ReputeCast publishes ("wordpress": the
-- plugin), from the daily update check.
CREATE TABLE integration_releases (
    kind       text        PRIMARY KEY,
    version    text        NOT NULL,
    source     text        NOT NULL DEFAULT '',
    checked_at timestamptz NOT NULL DEFAULT now()
);

-- Contact events for outgoing webhooks (contact.subscribed, .unsubscribed,
-- .updated), queued by triggers whatever wrote the change, only while an
-- endpoint of the workspace listens. A job turns them into deliveries.
CREATE TABLE webhook_contact_events (
    id           bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    event        text        NOT NULL,
    contact_id   uuid        NOT NULL,
    list_id      uuid,
    data         jsonb       NOT NULL DEFAULT '{}',
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose StatementBegin
CREATE FUNCTION webhook_listens(ws uuid, ev text) RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM webhooks h
        WHERE h.workspace_id = ws
          AND (ev = ANY (h.events) OR split_part(ev, '.', 1) || '.*' = ANY (h.events) OR '*' = ANY (h.events))
    );
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION webhook_contact_membership() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    ws uuid;
    ev text;
BEGIN
    IF NEW.status = 'subscribed' AND (TG_OP = 'INSERT' OR OLD.status <> 'subscribed') THEN
        ev := 'contact.subscribed';
    ELSIF NEW.status = 'unsubscribed' AND TG_OP = 'UPDATE' AND OLD.status <> 'unsubscribed' THEN
        ev := 'contact.unsubscribed';
    ELSE
        RETURN NULL;
    END IF;
    SELECT l.workspace_id INTO ws FROM lists l WHERE l.id = NEW.list_id;
    IF ws IS NULL OR NOT webhook_listens(ws, ev) THEN
        RETURN NULL;
    END IF;
    INSERT INTO webhook_contact_events (workspace_id, event, contact_id, list_id, data)
    VALUES (ws, ev, NEW.contact_id, NEW.list_id, jsonb_build_object('source', NEW.source));
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER list_memberships_webhook_events
    AFTER INSERT OR UPDATE OF status ON list_memberships
    FOR EACH ROW EXECUTE FUNCTION webhook_contact_membership();

-- +goose StatementBegin
CREATE FUNCTION webhook_contact_updated() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    changed text[] := '{}';
BEGIN
    IF NEW.erased_at IS NOT NULL THEN
        RETURN NULL;
    END IF;
    IF NEW.first_name IS DISTINCT FROM OLD.first_name THEN changed := changed || 'first_name'; END IF;
    IF NEW.last_name IS DISTINCT FROM OLD.last_name THEN changed := changed || 'last_name'; END IF;
    IF NEW.company IS DISTINCT FROM OLD.company THEN changed := changed || 'company'; END IF;
    IF NEW.country IS DISTINCT FROM OLD.country THEN changed := changed || 'country'; END IF;
    IF NEW.timezone IS DISTINCT FROM OLD.timezone THEN changed := changed || 'timezone'; END IF;
    IF NEW.language IS DISTINCT FROM OLD.language THEN changed := changed || 'language'; END IF;
    IF NEW.tags IS DISTINCT FROM OLD.tags THEN changed := changed || 'tags'; END IF;
    IF NEW.attributes IS DISTINCT FROM OLD.attributes THEN changed := changed || 'fields'; END IF;
    IF cardinality(changed) = 0 OR NOT webhook_listens(NEW.workspace_id, 'contact.updated') THEN
        RETURN NULL;
    END IF;
    INSERT INTO webhook_contact_events (workspace_id, event, contact_id, data)
    VALUES (NEW.workspace_id, 'contact.updated', NEW.id, jsonb_build_object('changed', to_jsonb(changed)));
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER contacts_webhook_events
    AFTER UPDATE OF first_name, last_name, company, country, timezone, language, tags, attributes ON contacts
    FOR EACH ROW EXECUTE FUNCTION webhook_contact_updated();

-- +goose StatementBegin
CREATE FUNCTION webhook_contact_preferences() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    ws uuid;
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.frequency IS NOT DISTINCT FROM OLD.frequency AND NEW.paused_until IS NOT DISTINCT FROM OLD.paused_until THEN
        RETURN NULL;
    END IF;
    SELECT c.workspace_id INTO ws FROM contacts c WHERE c.id = NEW.contact_id;
    IF ws IS NULL OR NOT webhook_listens(ws, 'contact.updated') THEN
        RETURN NULL;
    END IF;
    INSERT INTO webhook_contact_events (workspace_id, event, contact_id, data)
    VALUES (ws, 'contact.updated', NEW.contact_id, jsonb_build_object('changed', jsonb_build_array('preferences')));
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER contact_preferences_webhook_events
    AFTER INSERT OR UPDATE ON contact_preferences
    FOR EACH ROW EXECUTE FUNCTION webhook_contact_preferences();

-- +goose Down
DROP TRIGGER contact_preferences_webhook_events ON contact_preferences;
DROP FUNCTION webhook_contact_preferences();
DROP TRIGGER contacts_webhook_events ON contacts;
DROP FUNCTION webhook_contact_updated();
DROP TRIGGER list_memberships_webhook_events ON list_memberships;
DROP FUNCTION webhook_contact_membership();
DROP FUNCTION webhook_listens(uuid, text);
DROP TABLE webhook_contact_events;
DROP TABLE integration_releases;
DROP TABLE wordpress_drafts;
DROP TABLE wordpress_sites;
DROP TABLE integration_webhooks;
DROP TABLE integration_workflows;
DROP TABLE integration_calls;
DROP INDEX api_keys_integration_idx;
ALTER TABLE api_keys DROP COLUMN grace_until;
ALTER TABLE api_keys DROP COLUMN integration_id;
DROP TABLE integrations;
