-- +goose Up
-- Automations: lifecycle details, draft revisions and version notes,
-- the emails used by "Send email" steps, the trigger event queue, per-day step
-- stats, write-only webhook signing secrets, the webhook call log and alerts.

-- Pause state, the editor's optimistic-concurrency revision and who changed it.
ALTER TABLE automations
    ADD COLUMN paused_at      timestamptz,
    ADD COLUMN paused_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN pause_reason   text   NOT NULL DEFAULT '',
    ADD COLUMN draft_revision bigint NOT NULL DEFAULT 1,
    ADD COLUMN updated_by     uuid REFERENCES users(id) ON DELETE SET NULL;
-- Enrollment looks up live automations by trigger kind.
CREATE INDEX automations_live_trigger_idx ON automations (workspace_id, (trigger ->> 'kind'))
    WHERE status = 'live' AND deleted_at IS NULL;

-- summary: what changed in this version ("added "No" path with plain-text
-- follow-up"); migration: what happened to contacts in flight when it was
-- published (finish on the old version, or move to this one).
ALTER TABLE automation_versions
    ADD COLUMN summary   text NOT NULL DEFAULT '',
    ADD COLUMN migration text NOT NULL DEFAULT '' CHECK (migration IN ('', 'finish', 'move'));

-- node_entered_at: when the run reached its current step (delay timers,
-- branch windows). exit_reason: why a run left early (filtered, unsubscribed…).
ALTER TABLE automation_runs
    ADD COLUMN node_entered_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN exit_reason     text        NOT NULL DEFAULT '';
CREATE INDEX automation_runs_contact_idx ON automation_runs (automation_id, contact_id);
CREATE INDEX automation_runs_node_idx ON automation_runs (automation_id, version, node_id) WHERE status IN ('active', 'waiting');

-- Step stats become daily so the screens can show "last 30 days". Counters:
-- entered (arrived), completed (moved on), waiting (had to wait here),
-- exited (finished the flow here), dropped (left early: filtered out,
-- unsubscribed, removed), goal_reached, skipped (email skipped by a rule),
-- failed (webhook calls that failed), email delivery and engagement, paths
-- taken out of a branch or split ({"yes": 778, "no": 498}) and sends per
-- sending provider ({"<provider id>": 1266}).
ALTER TABLE automation_step_stats DROP CONSTRAINT automation_step_stats_pkey;
ALTER TABLE automation_step_stats
    ADD COLUMN day          date   NOT NULL DEFAULT CURRENT_DATE,
    ADD COLUMN exited       bigint NOT NULL DEFAULT 0,
    ADD COLUMN dropped      bigint NOT NULL DEFAULT 0,
    ADD COLUMN skipped      bigint NOT NULL DEFAULT 0,
    ADD COLUMN failed       bigint NOT NULL DEFAULT 0,
    ADD COLUMN bounced      bigint NOT NULL DEFAULT 0,
    ADD COLUMN complained   bigint NOT NULL DEFAULT 0,
    ADD COLUMN opened_human bigint NOT NULL DEFAULT 0,
    ADD COLUMN unsubscribed bigint NOT NULL DEFAULT 0,
    ADD COLUMN paths        jsonb  NOT NULL DEFAULT '{}',
    ADD COLUMN providers    jsonb  NOT NULL DEFAULT '{}';
ALTER TABLE automation_step_stats ADD PRIMARY KEY (automation_id, version, node_id, day);
CREATE INDEX automation_step_stats_day_idx ON automation_step_stats (automation_id, day);

-- Emails sent by "Send email" steps. Content is versioned: a step points at
-- one version, so editing an email never changes what a live flow sends until
-- the new version is published with the flow.
CREATE TABLE automation_emails (
    id             uuid PRIMARY KEY,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    automation_id  uuid        REFERENCES automations(id) ON DELETE SET NULL,
    name           text        NOT NULL,
    latest_version integer     NOT NULL DEFAULT 1,
    created_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    deleted_at     timestamptz
);
CREATE INDEX automation_emails_workspace_idx ON automation_emails (workspace_id) WHERE deleted_at IS NULL;

CREATE TABLE automation_email_versions (
    email_id   uuid        NOT NULL REFERENCES automation_emails(id) ON DELETE CASCADE,
    version    integer     NOT NULL,
    subject    text        NOT NULL DEFAULT '',
    preheader  text        NOT NULL DEFAULT '',
    mjml       text        NOT NULL DEFAULT '',
    html       text        NOT NULL DEFAULT '',
    plain_text text        NOT NULL DEFAULT '',
    created_by uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (email_id, version)
);

-- Trigger events waiting to enroll contacts. List joins and new tags are
-- written by the database triggers below, so every path that changes them
-- (forms, importer, API, contact edits) feeds automations; form submissions
-- and API events are written by their handlers (automations.Emit*); date
-- fields by the hourly scan. The dispatcher consumes them in order.
CREATE TABLE automation_events (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('joined_list', 'form_submitted', 'tag_added', 'api_event', 'date_field')),
    contact_id   uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    -- The list id, form id, tag, event name or date field the event is about.
    ref          text        NOT NULL DEFAULT '',
    payload      jsonb       NOT NULL DEFAULT '{}',
    occurred_at  timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    enrolled     integer     NOT NULL DEFAULT 0
);
CREATE INDEX automation_events_pending_idx ON automation_events (id) WHERE processed_at IS NULL;
CREATE INDEX automation_events_processed_idx ON automation_events (processed_at) WHERE processed_at IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION automation_emit_list_join() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    ws uuid;
BEGIN
    IF NEW.status NOT IN ('pending', 'subscribed') THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.status = NEW.status THEN
        RETURN NULL;
    END IF;
    SELECT l.workspace_id INTO ws FROM lists l WHERE l.id = NEW.list_id;
    -- Only queue events some live automation listens to.
    IF ws IS NULL OR NOT EXISTS (
        SELECT 1 FROM automations a
        WHERE a.workspace_id = ws AND a.status = 'live' AND a.deleted_at IS NULL
          AND a.trigger ->> 'kind' = 'joined_list' AND a.trigger ->> 'listId' = NEW.list_id::text
    ) THEN
        RETURN NULL;
    END IF;
    INSERT INTO automation_events (workspace_id, kind, contact_id, ref, payload)
    VALUES (ws, 'joined_list', NEW.contact_id, NEW.list_id::text,
            jsonb_build_object('status', NEW.status, 'source', NEW.source));
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER list_memberships_automation_events
    AFTER INSERT OR UPDATE OF status ON list_memberships
    FOR EACH ROW EXECUTE FUNCTION automation_emit_list_join();

-- +goose StatementBegin
CREATE FUNCTION automation_emit_tags() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM automations a
        WHERE a.workspace_id = NEW.workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
          AND a.trigger ->> 'kind' = 'tag_added'
    ) THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'INSERT' THEN
        INSERT INTO automation_events (workspace_id, kind, contact_id, ref)
        SELECT NEW.workspace_id, 'tag_added', NEW.id, t
        FROM unnest(NEW.tags) AS t
        WHERE EXISTS (SELECT 1 FROM automations a
                      WHERE a.workspace_id = NEW.workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
                        AND a.trigger ->> 'kind' = 'tag_added' AND a.trigger ->> 'tag' = t);
    ELSE
        INSERT INTO automation_events (workspace_id, kind, contact_id, ref)
        SELECT NEW.workspace_id, 'tag_added', NEW.id, t
        FROM unnest(NEW.tags) AS t
        WHERE NOT (t = ANY (OLD.tags))
          AND EXISTS (SELECT 1 FROM automations a
                      WHERE a.workspace_id = NEW.workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
                        AND a.trigger ->> 'kind' = 'tag_added' AND a.trigger ->> 'tag' = t);
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER contacts_automation_events
    AFTER INSERT OR UPDATE OF tags ON contacts
    FOR EACH ROW EXECUTE FUNCTION automation_emit_tags();

-- Webhook signing secrets, write-only: sealed with the instance key
-- ("automation:<id>:step:<node>:secret") and never returned by the API.
CREATE TABLE automation_step_secrets (
    automation_id uuid        NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    node_id       text        NOT NULL,
    secret_enc    bytea       NOT NULL,
    updated_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (automation_id, node_id)
);

-- Every outgoing webhook call made by a flow (delivery log).
CREATE TABLE automation_webhook_calls (
    id            uuid PRIMARY KEY,
    automation_id uuid        NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    version       integer     NOT NULL,
    node_id       text        NOT NULL,
    run_id        uuid,
    url           text        NOT NULL,
    status_code   integer     NOT NULL DEFAULT 0,
    error         text        NOT NULL DEFAULT '',
    duration_ms   integer     NOT NULL DEFAULT 0,
    attempt       integer     NOT NULL DEFAULT 1,
    called_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX automation_webhook_calls_step_idx ON automation_webhook_calls (automation_id, node_id, called_at DESC);

-- Problems a flow reports on the automations list: a step that keeps failing
-- (webhook errors) or sending held because no provider on the Automations
-- route is healthy. One open alert per automation, kind and step.
CREATE TABLE automation_alerts (
    id            uuid PRIMARY KEY,
    automation_id uuid        NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    kind          text        NOT NULL CHECK (kind IN ('step_failing', 'sending_held')),
    node_id       text        NOT NULL DEFAULT '',
    -- {"url":"https://hooks.slack.com/…","status":404,"error":"…"} or {"queued":23}
    detail        jsonb       NOT NULL DEFAULT '{}',
    failures      bigint      NOT NULL DEFAULT 0,
    first_at      timestamptz NOT NULL DEFAULT now(),
    last_at       timestamptz NOT NULL DEFAULT now(),
    resolved_at   timestamptz
);
CREATE UNIQUE INDEX automation_alerts_open_key ON automation_alerts (automation_id, kind, node_id) WHERE resolved_at IS NULL;

-- Automation messages are looked up per flow and step (held mail, skip rules).
CREATE INDEX messages_automation_idx ON messages (automation_id, automation_node, queued_at DESC) WHERE automation_id IS NOT NULL;

-- +goose Down
DROP INDEX messages_automation_idx;
DROP TABLE automation_alerts;
DROP TABLE automation_webhook_calls;
DROP TABLE automation_step_secrets;
DROP TRIGGER contacts_automation_events ON contacts;
DROP FUNCTION automation_emit_tags();
DROP TRIGGER list_memberships_automation_events ON list_memberships;
DROP FUNCTION automation_emit_list_join();
DROP TABLE automation_events;
DROP TABLE automation_email_versions;
DROP TABLE automation_emails;

-- Fold the daily stats back into one row per version and step.
CREATE TEMP TABLE automation_step_stats_down AS
SELECT automation_id, version, node_id,
       sum(entered)::bigint AS entered, sum(waiting)::bigint AS waiting, sum(completed)::bigint AS completed,
       sum(sent)::bigint AS sent, sum(delivered)::bigint AS delivered, sum(opened)::bigint AS opened,
       sum(clicked)::bigint AS clicked, sum(branch_yes)::bigint AS branch_yes, sum(branch_no)::bigint AS branch_no,
       sum(goal_reached)::bigint AS goal_reached
FROM automation_step_stats GROUP BY automation_id, version, node_id;
DELETE FROM automation_step_stats;
DROP INDEX automation_step_stats_day_idx;
ALTER TABLE automation_step_stats DROP CONSTRAINT automation_step_stats_pkey;
ALTER TABLE automation_step_stats
    DROP COLUMN day, DROP COLUMN exited, DROP COLUMN dropped, DROP COLUMN skipped, DROP COLUMN failed,
    DROP COLUMN bounced, DROP COLUMN complained, DROP COLUMN opened_human, DROP COLUMN unsubscribed,
    DROP COLUMN paths, DROP COLUMN providers;
ALTER TABLE automation_step_stats ADD PRIMARY KEY (automation_id, version, node_id);
INSERT INTO automation_step_stats (automation_id, version, node_id, entered, waiting, completed, sent, delivered, opened, clicked, branch_yes, branch_no, goal_reached)
SELECT automation_id, version, node_id, entered, waiting, completed, sent, delivered, opened, clicked, branch_yes, branch_no, goal_reached
FROM automation_step_stats_down;
DROP TABLE automation_step_stats_down;

DROP INDEX automation_runs_node_idx;
DROP INDEX automation_runs_contact_idx;
ALTER TABLE automation_runs DROP COLUMN exit_reason, DROP COLUMN node_entered_at;
ALTER TABLE automation_versions DROP COLUMN migration, DROP COLUMN summary;
DROP INDEX automations_live_trigger_idx;
ALTER TABLE automations
    DROP COLUMN updated_by, DROP COLUMN draft_revision, DROP COLUMN pause_reason,
    DROP COLUMN paused_by, DROP COLUMN paused_at;
