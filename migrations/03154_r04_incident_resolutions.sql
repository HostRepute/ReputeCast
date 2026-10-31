-- +goose Up
-- A held send's hold can end as expired (SR-3, 03151) and a guarded send's
-- incidents as continued, stopped or acknowledged (GS, 03050). Both
-- migrations replaced the same check: this one allows every resolution.
ALTER TABLE campaign_incidents DROP CONSTRAINT campaign_incidents_resolution_check;
ALTER TABLE campaign_incidents ADD CONSTRAINT campaign_incidents_resolution_check CHECK (resolution IN
    ('','resumed','resumed_with_changes','cancelled','moved','retried','auto_recovered','expired','continued','stopped','acknowledged'));

-- +goose Down
UPDATE campaign_incidents SET resolution = 'resumed' WHERE resolution IN ('continued','stopped','acknowledged');
ALTER TABLE campaign_incidents DROP CONSTRAINT campaign_incidents_resolution_check;
ALTER TABLE campaign_incidents ADD CONSTRAINT campaign_incidents_resolution_check
    CHECK (resolution IN ('','resumed','resumed_with_changes','cancelled','moved','retried','auto_recovered','expired'));
