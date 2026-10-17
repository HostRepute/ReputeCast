-- +goose Up
-- Growth sources (audience-insights brief §3.4, M23): how each contact
-- first joined and whether they left, UTM parameters and the referring page
-- with each consent record, and daily rates per source for "quality by
-- source".

-- Forms and hosted pages save where a signup came from with its consent
-- record: the UTM parameters of the page's address, the page itself and the
-- page the visitor came from, both without their query string.
ALTER TABLE consent_events
    ADD COLUMN utm_source   text NOT NULL DEFAULT '',
    ADD COLUMN utm_medium   text NOT NULL DEFAULT '',
    ADD COLUMN utm_campaign text NOT NULL DEFAULT '',
    ADD COLUMN utm_content  text NOT NULL DEFAULT '',
    ADD COLUMN utm_term     text NOT NULL DEFAULT '',
    ADD COLUMN page         text NOT NULL DEFAULT '',
    ADD COLUMN referrer     text NOT NULL DEFAULT '';

-- How each contact first joined (a source from the growth registry: form,
-- subscribe_page, import, carried, api_key, integration, person,
-- preference_center, unknown; later referral and recommendation), with the
-- first touch's UTM and pages, and the first departure after it. Kept
-- after a contact is erased or deleted, so past growth keeps adding up; it
-- holds no personal data.
CREATE TABLE contact_growth (
    contact_id      uuid        PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    source_kind     text        NOT NULL,
    -- The form, import, API key, integration (plus ":" and the workflow's
    -- key) or person; for "carried" the source the import brought along.
    source_id       text        NOT NULL DEFAULT '',
    joined_at       timestamptz NOT NULL,
    left_at         timestamptz,
    left_kind       text        CHECK (left_kind IN ('unsubscribed','hard_bounce','complaint','erased')),
    utm_source      text        NOT NULL DEFAULT '',
    utm_medium      text        NOT NULL DEFAULT '',
    utm_campaign    text        NOT NULL DEFAULT '',
    utm_content     text        NOT NULL DEFAULT '',
    utm_term        text        NOT NULL DEFAULT '',
    page            text        NOT NULL DEFAULT '',
    referrer        text        NOT NULL DEFAULT '',
    referrer_domain text        NOT NULL DEFAULT '',
    -- Joined through a form or the subscribe page after UTM capture began;
    -- earlier joins read "Not captured".
    captured        boolean     NOT NULL DEFAULT false,
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX contact_growth_joined_idx ON contact_growth (workspace_id, joined_at);
CREATE INDEX contact_growth_left_idx ON contact_growth (workspace_id, left_at) WHERE left_at IS NOT NULL;
CREATE INDEX contact_growth_source_idx ON contact_growth (workspace_id, source_kind, source_id);
CREATE INDEX contact_growth_campaign_idx ON contact_growth (workspace_id, lower(utm_campaign)) WHERE utm_campaign <> '';
CREATE INDEX contact_growth_referrer_idx ON contact_growth (workspace_id, referrer_domain) WHERE referrer_domain <> '';

-- Sends per day to the contacts of each source (dim 'source', key
-- "kind:id"), each UTM combination (dim 'utm', key "source/medium/campaign")
-- and the whole workspace (dim 'workspace', key ''): the period's bounce,
-- complaint and human-open rates add these up. Written by the nightly job
-- for the days before today.
CREATE TABLE growth_quality_daily (
    workspace_id uuid   NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    day          date   NOT NULL,
    dim          text   NOT NULL CHECK (dim IN ('source','utm','workspace')),
    key          text   NOT NULL,
    sent         bigint NOT NULL DEFAULT 0,
    delivered    bigint NOT NULL DEFAULT 0,
    bounced      bigint NOT NULL DEFAULT 0,
    complained   bigint NOT NULL DEFAULT 0,
    human_opened bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, dim, key, day)
);

-- Per workspace: when UTM capture began, and the attribution job's
-- watermarks.
CREATE TABLE growth_state (
    workspace_id       uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    capture_started_at timestamptz,
    attributed_through timestamptz,
    rolled_up_through  date,
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- The attribution job reads new joins and departures by time (suppressions
-- already have an index by workspace and time).
CREATE INDEX consent_events_join_idx ON consent_events (occurred_at) WHERE action IN ('subscribed','confirmed','resubscribed','imported');
CREATE INDEX contacts_erased_idx ON contacts (workspace_id, erased_at) WHERE erased_at IS NOT NULL;

-- +goose Down
DROP INDEX contacts_erased_idx;
DROP INDEX consent_events_join_idx;
DROP TABLE growth_state;
DROP TABLE growth_quality_daily;
DROP TABLE contact_growth;
ALTER TABLE consent_events
    DROP COLUMN referrer, DROP COLUMN page, DROP COLUMN utm_term, DROP COLUMN utm_content,
    DROP COLUMN utm_campaign, DROP COLUMN utm_medium, DROP COLUMN utm_source;
