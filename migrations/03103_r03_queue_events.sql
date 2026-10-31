-- +goose Up
-- PR-4: SES events can come through an SQS queue ReputeCast polls (SES →
-- SNS → SQS), so ReputeCast works behind NAT without a public URL. Events
-- pulled from the queue are stored like webhooks, with their source.
ALTER TABLE provider_raw_events DROP CONSTRAINT IF EXISTS provider_raw_events_source_check;
ALTER TABLE provider_raw_events ADD CONSTRAINT provider_raw_events_source_check
    CHECK (source IN ('webhook', 'bounce_mailbox', 'backfill', 'test', 'queue'));

-- +goose Down
UPDATE provider_raw_events SET source = 'webhook' WHERE source = 'queue';
ALTER TABLE provider_raw_events DROP CONSTRAINT IF EXISTS provider_raw_events_source_check;
ALTER TABLE provider_raw_events ADD CONSTRAINT provider_raw_events_source_check
    CHECK (source IN ('webhook', 'bounce_mailbox', 'backfill', 'test'));
