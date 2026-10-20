-- +goose Up
-- A paid gift's receipt goes out from the gifts job, after the payment's
-- webhook is in; this records that it went (or why it couldn't). The gift
-- page's buyer and the recipient sit outside the contacts until the gift
-- is redeemed, so their emails are tracked on the gift.
ALTER TABLE earn_gifts
    ADD COLUMN receipt_sent_at timestamptz,
    -- The last "Your gift" email: sent, or why it couldn't go
    -- (not_published, no_route, suppressed, failed).
    ADD COLUMN email_status    text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE earn_gifts DROP COLUMN email_status, DROP COLUMN receipt_sent_at;
