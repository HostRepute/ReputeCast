-- +goose Up
-- Mail clients, devices, countries and times in the campaign report (M57,
-- G-12). Dimensions are resolved when a message's first human engagement is
-- recorded and kept on the message, like the counters: event IPs and user
-- agents are cleared by retention, so they can't be read back later.

-- A campaign records breakdowns only when a send built it after this
-- release; older campaigns never get half-filled rows from late clicks.
ALTER TABLE campaigns ADD COLUMN breakdown boolean NOT NULL DEFAULT false;

-- The first human engagement's client, system, device and place, and the
-- local weekday-hour ((isodow-1)*24 + hour, 0-167) of the first human
-- engagement and of the first click in the workspace's time zone at that
-- moment. Country and city are derived from the reader's IP and are cleared
-- with it by retention and erasure.
ALTER TABLE messages
    ADD COLUMN human_client  text NOT NULL DEFAULT '',
    ADD COLUMN human_os      text NOT NULL DEFAULT '',
    ADD COLUMN human_device  text NOT NULL DEFAULT '',
    ADD COLUMN human_country text NOT NULL DEFAULT '',
    ADD COLUMN human_city    text NOT NULL DEFAULT '',
    ADD COLUMN human_local   smallint,
    ADD COLUMN click_local   smallint;

-- Per campaign × dimension × key: human opens and clicks. City keys are
-- "CC|City", hour keys 0-23, weekday keys 1-7 (ISO, Monday = 1).
CREATE TABLE campaign_dims (
    campaign_id  uuid   NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    dim          text   NOT NULL CHECK (dim IN ('client','os','device','country','city','hour','weekday')),
    key          text   NOT NULL,
    opened_human bigint NOT NULL DEFAULT 0,
    clicked      bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (campaign_id, dim, key)
);

-- +goose Down
DROP TABLE campaign_dims;
ALTER TABLE messages
    DROP COLUMN human_client,
    DROP COLUMN human_os,
    DROP COLUMN human_device,
    DROP COLUMN human_country,
    DROP COLUMN human_city,
    DROP COLUMN human_local,
    DROP COLUMN click_local;
ALTER TABLE campaigns DROP COLUMN breakdown;
