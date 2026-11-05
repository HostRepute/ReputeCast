-- +goose Up
-- The SNS webhook of AWS End User Messaging trusts only the provider's own
-- topic (config.topic_arn), and no topic while none is kept: SNS signs every
-- AWS account's messages alike, and it used to trust any topic until one
-- was kept. Webhooks set up by hand in AWS never had their topic kept: keep
-- the one their verified events came from, when they all came from one, so
-- their receipts and opt-outs keep coming. The others keep the topic of the
-- next subscription confirmation. The body is read through
-- encode(…, 'escape'), which never fails on bytes that aren't UTF-8.
WITH legacy AS MATERIALIZED (
    SELECT id FROM sms_providers
    WHERE adapter = 'aws_eum' AND removed_at IS NULL AND coalesce(config->>'topic_arn', '') = ''
), seen AS (
    SELECT e.provider_id,
        substring(encode(e.body, 'escape') FROM '"TopicArn"\s*:\s*"(arn:aws[a-z-]*:sns:[^"\\]+)"') AS topic
    FROM sms_raw_events e JOIN legacy l ON l.id = e.provider_id
    WHERE e.signature_ok
), single AS (
    SELECT provider_id, min(topic) AS topic FROM seen
    WHERE topic IS NOT NULL
    GROUP BY provider_id
    HAVING count(DISTINCT topic) = 1
)
UPDATE sms_providers p SET config = p.config || jsonb_build_object('topic_arn', s.topic)
FROM single s
WHERE p.id = s.provider_id;

-- +goose Down
-- The topics kept stay: they are the ones the webhook trusted already.
SELECT 1;
