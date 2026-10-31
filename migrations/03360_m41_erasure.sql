-- +goose Up
-- Review 2026-11 DP-1: erasing a contact removes their push subscriptions
-- with their own reason.
ALTER TABLE push_subscriptions DROP CONSTRAINT push_subscriptions_removed_reason_check;
ALTER TABLE push_subscriptions ADD CONSTRAINT push_subscriptions_removed_reason_check
    CHECK (removed_reason IN ('','gone','unsubscribed','turned_off','rotated','site_removed','rejected','removed','erased'));

-- +goose Down
UPDATE push_subscriptions SET removed_reason = 'removed' WHERE removed_reason = 'erased';
ALTER TABLE push_subscriptions DROP CONSTRAINT push_subscriptions_removed_reason_check;
ALTER TABLE push_subscriptions ADD CONSTRAINT push_subscriptions_removed_reason_check
    CHECK (removed_reason IN ('','gone','unsubscribed','turned_off','rotated','site_removed','rejected','removed'));
