-- +goose Up
-- Segments keep when their members entered and left (review 2026-11
-- AU-16), the groundwork for "entered a segment" triggers (AT-10). A
-- refresh of segment_members is a diff: the members that stay keep their
-- entered_at; the others leave an event each way.
ALTER TABLE segment_members ADD COLUMN entered_at timestamptz NOT NULL DEFAULT now();
CREATE TABLE segment_member_events (
    segment_id uuid        NOT NULL REFERENCES segments(id) ON DELETE CASCADE,
    contact_id uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    kind       text        NOT NULL CHECK (kind IN ('entered', 'left')),
    at         timestamptz NOT NULL
);
CREATE INDEX segment_member_events_segment_idx ON segment_member_events (segment_id, at DESC);
CREATE INDEX segment_member_events_contact_idx ON segment_member_events (contact_id);

-- +goose Down
DROP TABLE segment_member_events;
ALTER TABLE segment_members DROP COLUMN entered_at;
