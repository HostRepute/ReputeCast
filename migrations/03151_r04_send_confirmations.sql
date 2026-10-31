-- +goose Up
-- What a person confirmed when they scheduled a send (review SR-3). At the
-- send-time recheck, Send Readiness compares the campaign and its fresh
-- checks with it, and holds the send on what nobody confirmed: a warning or
-- an unknown check that wasn't there, the audience grown or shrunk by more
-- than 10%, or the content, sender or route edited after scheduling.
CREATE TABLE campaign_send_confirmations (
    campaign_id    uuid        PRIMARY KEY REFERENCES campaigns(id) ON DELETE CASCADE,
    workspace_id   uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    run_id         uuid        REFERENCES readiness_runs(id) ON DELETE SET NULL,
    confirmed_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    confirmed_at   timestamptz NOT NULL,
    scheduled_at   timestamptz,
    recipients     bigint      NOT NULL,
    -- The warnings the person confirmed, and the checks that were unknown
    -- when they chose to continue without them (check keys).
    acknowledged   text[]      NOT NULL DEFAULT '{}',
    unknown        text[]      NOT NULL DEFAULT '{}',
    -- The campaign as confirmed: its content (with the language versions),
    -- its sender and its route (the override and the Broadcasts rules).
    content_digest text        NOT NULL DEFAULT '',
    sender_id      uuid,
    route_digest   text        NOT NULL DEFAULT ''
);
CREATE INDEX campaign_send_confirmations_ws_idx ON campaign_send_confirmations (workspace_id);

-- A held send nobody reviewed goes back to Draft some hours after its time
-- instead of starting days late: its hold ends as expired.
ALTER TABLE campaign_incidents DROP CONSTRAINT campaign_incidents_resolution_check;
ALTER TABLE campaign_incidents ADD CONSTRAINT campaign_incidents_resolution_check
    CHECK (resolution IN ('','resumed','resumed_with_changes','cancelled','moved','retried','auto_recovered','expired'));

-- +goose Down
UPDATE campaign_incidents SET resolution = 'cancelled' WHERE resolution = 'expired';
ALTER TABLE campaign_incidents DROP CONSTRAINT campaign_incidents_resolution_check;
ALTER TABLE campaign_incidents ADD CONSTRAINT campaign_incidents_resolution_check
    CHECK (resolution IN ('','resumed','resumed_with_changes','cancelled','moved','retried','auto_recovered'));
DROP TABLE campaign_send_confirmations;
