-- +goose Up
-- "Retry after fixing" (review AN-7, R5.L5): the addresses of an issue that
-- a mailbox provider blocked for the sender's reputation or policy are valid,
-- so once the block is fixed they are sent the issue again by a retry: a
-- campaign whose audience names the issue it retries (audience ->>
-- 'retry_of') and goes only to those addresses. An issue has one retry at a
-- time; a deleted retry leaves room for another, which still sends nobody
-- twice.
CREATE UNIQUE INDEX campaigns_retry_of_idx ON campaigns ((audience ->> 'retry_of'))
    WHERE deleted_at IS NULL AND (audience ->> 'retry_of') IS NOT NULL;

-- +goose Down
DROP INDEX campaigns_retry_of_idx;
