-- +goose Up
-- Automations, complete (review 2026-11 M44).

-- AT-11: a flow starts on up to three triggers. The first is the trigger
-- object itself; the others are in its "also". triggers lists all of them,
-- so whoever asks "does a live flow listen to X?" reads one array.
ALTER TABLE automations ADD COLUMN triggers jsonb GENERATED ALWAYS AS (
    CASE WHEN trigger = '{}'::jsonb THEN '[]'::jsonb
         ELSE ('[]'::jsonb || (trigger - 'also')) || coalesce(trigger -> 'also', '[]'::jsonb) END
) STORED;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION automation_emit_list_join() RETURNS trigger LANGUAGE plpgsql AS $$
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
          AND a.triggers @> jsonb_build_array(jsonb_build_object('kind', 'joined_list', 'listId', NEW.list_id::text))
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION automation_emit_tags() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM automations a
        WHERE a.workspace_id = NEW.workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
          AND a.triggers @> '[{"kind": "tag_added"}]'::jsonb
    ) THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'INSERT' THEN
        INSERT INTO automation_events (workspace_id, kind, contact_id, ref)
        SELECT NEW.workspace_id, 'tag_added', NEW.id, t
        FROM unnest(NEW.tags) AS t
        WHERE EXISTS (SELECT 1 FROM automations a
                      WHERE a.workspace_id = NEW.workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
                        AND a.triggers @> jsonb_build_array(jsonb_build_object('kind', 'tag_added', 'tag', t)));
    ELSE
        INSERT INTO automation_events (workspace_id, kind, contact_id, ref)
        SELECT NEW.workspace_id, 'tag_added', NEW.id, t
        FROM unnest(NEW.tags) AS t
        WHERE NOT (t = ANY (OLD.tags))
          AND EXISTS (SELECT 1 FROM automations a
                      WHERE a.workspace_id = NEW.workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
                        AND a.triggers @> jsonb_build_array(jsonb_build_object('kind', 'tag_added', 'tag', t)));
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- AT-8: leaving the trigger's list wakes the contact's runs in flows that
-- let them go with it; the engine ends those runs on its next tick, with
-- the step's numbers.
-- +goose StatementBegin
CREATE FUNCTION automation_wake_list_leavers() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    old_in boolean := OLD.status IN ('pending', 'subscribed');
    new_in boolean := TG_OP = 'UPDATE' AND NEW.status IN ('pending', 'subscribed');
BEGIN
    IF NOT old_in OR new_in THEN
        RETURN NULL;
    END IF;
    UPDATE automation_runs r SET next_at = now()
    FROM automations a
    WHERE r.contact_id = OLD.contact_id AND r.status IN ('active', 'waiting') AND r.automation_id = a.id
      AND a.deleted_at IS NULL
      AND a.trigger @> jsonb_build_object('kind', 'joined_list', 'listId', OLD.list_id::text, 'exitOnLeave', true);
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER list_memberships_automation_leavers
    AFTER UPDATE OF status OR DELETE ON list_memberships
    FOR EACH ROW EXECUTE FUNCTION automation_wake_list_leavers();

-- AT-10: "enters a segment" and "leaves a segment" read segment_member_events
-- (AU-16). A dynamic segment a live flow listens to is evaluated on a
-- schedule; watched_at is its first evaluation, which only records the
-- members (nobody "enters" a segment they were already in).
ALTER TABLE segments ADD COLUMN watched_at timestamptz;

-- +goose StatementBegin
CREATE FUNCTION automation_emit_segment_events() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO automation_events (workspace_id, kind, contact_id, ref, occurred_at)
    SELECT s.workspace_id, CASE n.kind WHEN 'entered' THEN 'segment_entered' ELSE 'segment_left' END, n.contact_id, n.segment_id::text, n.at
    FROM new_rows n
    JOIN segments s ON s.id = n.segment_id
    WHERE EXISTS (SELECT 1 FROM automations a
                  WHERE a.workspace_id = s.workspace_id AND a.status = 'live' AND a.deleted_at IS NULL
                    AND a.triggers @> jsonb_build_array(jsonb_build_object(
                        'kind', CASE n.kind WHEN 'entered' THEN 'segment_entered' ELSE 'segment_left' END,
                        'segmentId', n.segment_id::text)));
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER segment_member_events_automations
    AFTER INSERT ON segment_member_events
    REFERENCING NEW TABLE AS new_rows
    FOR EACH STATEMENT EXECUTE FUNCTION automation_emit_segment_events();

-- AT-9: at most marketing_cap_count campaign and automation emails per
-- contact in marketing_cap_days days (0: no cap).
ALTER TABLE workspaces
    ADD COLUMN marketing_cap_count int NOT NULL DEFAULT 0 CHECK (marketing_cap_count BETWEEN 0 AND 50),
    ADD COLUMN marketing_cap_days  int NOT NULL DEFAULT 7 CHECK (marketing_cap_days BETWEEN 1 AND 90);

-- +goose Down
ALTER TABLE workspaces DROP COLUMN marketing_cap_count, DROP COLUMN marketing_cap_days;
DROP TRIGGER segment_member_events_automations ON segment_member_events;
DROP FUNCTION automation_emit_segment_events();
ALTER TABLE segments DROP COLUMN watched_at;
DROP TRIGGER list_memberships_automation_leavers ON list_memberships;
DROP FUNCTION automation_wake_list_leavers();
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION automation_emit_list_join() RETURNS trigger LANGUAGE plpgsql AS $$
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
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION automation_emit_tags() RETURNS trigger LANGUAGE plpgsql AS $$
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
ALTER TABLE automations DROP COLUMN triggers;
