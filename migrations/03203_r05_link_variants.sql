-- +goose Up
-- Clicks per link and A/B variant (review AN-5): each variant's click map.
-- campaign_links keeps every recipient's clicks per link; this splits them
-- by the variant each recipient got (the test sample and, for the winner,
-- the recipients who got it after the test). Campaigns without a test have
-- no rows. The counters job rebuilds it from the stored clicks.
CREATE TABLE campaign_link_variants (
    campaign_id   uuid    NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    position      integer NOT NULL,
    variant       text    NOT NULL,
    clicks        bigint  NOT NULL DEFAULT 0,
    unique_clicks bigint  NOT NULL DEFAULT 0,
    bot_clicks    bigint  NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, position, variant)
);

-- +goose Down
DROP TABLE campaign_link_variants;
