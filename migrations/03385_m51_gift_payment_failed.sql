-- +goose Up
-- Review 2026-11 EN-9: a gift whose bank payment fails tells its buyer, with
-- a ninth member email, gift_payment_failed.
ALTER TABLE earn_member_emails DROP CONSTRAINT earn_member_emails_email_check;
ALTER TABLE earn_member_emails ADD CONSTRAINT earn_member_emails_email_check
    CHECK (email IN ('welcome','trial_ending','payment_failed','renewal_reminder','canceled','gift_purchased','gift_payment_failed','gift_received','gift_ending'));
ALTER TABLE earn_gifts ADD COLUMN payment_failed_at timestamptz, ADD COLUMN failed_notice_at timestamptz;

-- +goose Down
ALTER TABLE earn_gifts DROP COLUMN payment_failed_at, DROP COLUMN failed_notice_at;
DELETE FROM earn_member_emails WHERE email = 'gift_payment_failed';
ALTER TABLE earn_member_emails DROP CONSTRAINT earn_member_emails_email_check;
ALTER TABLE earn_member_emails ADD CONSTRAINT earn_member_emails_email_check
    CHECK (email IN ('welcome','trial_ending','payment_failed','renewal_reminder','canceled','gift_purchased','gift_received','gift_ending'));
