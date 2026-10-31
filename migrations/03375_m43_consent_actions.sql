-- +goose Up
-- Two consent events (review 2026-11 AU-6, AU-7): the team took a contact
-- off a list (not the contact's own unsubscribe, so it never counts as
-- one), and the team corrected a contact's address.
ALTER TABLE consent_events DROP CONSTRAINT consent_events_action_check;
ALTER TABLE consent_events ADD CONSTRAINT consent_events_action_check
    CHECK (action IN ('subscribed','confirmed','unsubscribed','resubscribed','imported','paused','frequency_changed','erased','optin_sent','suppressed',
                      'removed','email_changed'));

-- +goose Down
DELETE FROM consent_events WHERE action IN ('removed', 'email_changed');
ALTER TABLE consent_events DROP CONSTRAINT consent_events_action_check;
ALTER TABLE consent_events ADD CONSTRAINT consent_events_action_check
    CHECK (action IN ('subscribed','confirmed','unsubscribed','resubscribed','imported','paused','frequency_changed','erased','optin_sent','suppressed'));
