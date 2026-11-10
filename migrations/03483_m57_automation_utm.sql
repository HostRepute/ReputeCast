-- +goose Up
-- M57 G-13: UTM tags on email links. An automation version's emails add
-- the tags decided when it was published ({"source", "medium",
-- "campaign"}; NULL = untagged), so the workspace's "Add UTM tags to links"
-- reaches a flow at its next publish and a published version never changes.
ALTER TABLE automation_versions ADD COLUMN utm jsonb;

-- One email version can be sent by several automation versions, tagged by
-- one and not by another: its links are stored once per set of tags (''
-- = untagged, else the query the tags add), so a click finds the address
-- of the email it came from.
ALTER TABLE automation_email_links ADD COLUMN utm text NOT NULL DEFAULT '';
ALTER TABLE automation_email_links DROP CONSTRAINT automation_email_links_pkey;
ALTER TABLE automation_email_links ADD PRIMARY KEY (email_id, version, language, utm, position);

-- +goose Down
DELETE FROM automation_email_links WHERE utm <> '';
ALTER TABLE automation_email_links DROP CONSTRAINT automation_email_links_pkey;
ALTER TABLE automation_email_links ADD PRIMARY KEY (email_id, version, language, position);
ALTER TABLE automation_email_links DROP COLUMN utm;
ALTER TABLE automation_versions DROP COLUMN utm;
