-- +goose Up
-- GS-3. An outage the provider health job finds goes through each running
-- send's outage path: the job looks up the sends with an open or held share
-- on the provider. Only the routes of running sends are indexed.
CREATE INDEX campaign_routes_open_provider_idx ON campaign_routes (provider_id)
    WHERE status IN ('pending', 'sending', 'held', 'paused');

-- +goose Down
DROP INDEX IF EXISTS campaign_routes_open_provider_idx;
