-- +goose Up
-- First-party, cookie-free views of a campaign's archive page (review
-- AN-13), counted like the website's pages: one view per visitor, page and
-- day, from a daily salted hash of the address and browser that is
-- deleted once the day is over. page_id is the campaign.
ALTER TABLE website_view_days DROP CONSTRAINT website_view_days_page_kind_check;
ALTER TABLE website_view_days ADD CONSTRAINT website_view_days_page_kind_check
    CHECK (page_kind IN ('post','home','archive','tag','page','issue'));

-- +goose Down
DELETE FROM website_view_days WHERE page_kind = 'issue';
ALTER TABLE website_view_days DROP CONSTRAINT website_view_days_page_kind_check;
ALTER TABLE website_view_days ADD CONSTRAINT website_view_days_page_kind_check
    CHECK (page_kind IN ('post','home','archive','tag','page'));
