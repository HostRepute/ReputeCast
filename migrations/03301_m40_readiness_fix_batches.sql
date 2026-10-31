-- +goose Up
-- One-click fixes applied together from Review & send, kept so they can be
-- undone (review R1.U3). A batch keeps the parts of the campaign it changed,
-- as they were before and after: the content version (content fixes save
-- one through the composer), the route override, the audience and the
-- content settings. Undo puts them back while the campaign is still as the
-- batch left it. blocked says why it can't (languages: it changed a
-- language version; merged: its content was saved merged with someone
-- else's edit). applied is what each fix did, in applied_lang.
CREATE TABLE readiness_fix_batches (
    id               uuid        PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id      uuid        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    applied_by       uuid        REFERENCES users(id) ON DELETE SET NULL,
    applied_at       timestamptz NOT NULL,
    applied          jsonb       NOT NULL DEFAULT '[]',
    applied_lang     text        NOT NULL DEFAULT '',
    blocked          text        NOT NULL DEFAULT '' CHECK (blocked IN ('', 'languages', 'merged')),
    version_before   integer,
    version_after    integer,
    route_changed    boolean     NOT NULL DEFAULT false,
    route_before     jsonb,
    route_after      jsonb,
    audience_changed boolean     NOT NULL DEFAULT false,
    audience_before  jsonb,
    audience_after   jsonb,
    settings_changed boolean     NOT NULL DEFAULT false,
    settings_before  jsonb,
    settings_after   jsonb,
    undone_at        timestamptz,
    undone_by        uuid        REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX readiness_fix_batches_campaign_idx ON readiness_fix_batches (campaign_id, applied_at DESC);
CREATE INDEX readiness_fix_batches_ws_idx ON readiness_fix_batches (workspace_id);

-- +goose Down
DROP TABLE readiness_fix_batches;
