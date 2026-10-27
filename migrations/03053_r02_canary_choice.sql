-- +goose Up
-- GS-10 (C1). The canary a send starts with, chosen in Review & send:
-- auto (Traffic rules decide), on or off. Kept on the campaign so a
-- scheduled send starts with it at its time.
ALTER TABLE campaigns ADD COLUMN canary text NOT NULL DEFAULT 'auto' CHECK (canary IN ('auto', 'on', 'off'));

-- +goose Down
ALTER TABLE campaigns DROP COLUMN canary;
