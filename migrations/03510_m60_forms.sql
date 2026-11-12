-- +goose Up
-- M60 G-23–G-25: multi-step forms, targeting and A/B tests.
--
-- A form's steps and targeting live in forms.content (jsonb). An A/B test
-- compares the form itself (variant A) with up to two variants (B, C) that
-- override its copy, steps and popup triggers; the test's state is
-- forms.ab ({status running|ended, startedOn, endedAt, winner, autoEnd,
-- minViews, confidence}; '{}' = never tested) and its variants, A's weight
-- included, are form_variants rows while it runs.
ALTER TABLE forms ADD COLUMN ab jsonb NOT NULL DEFAULT '{}';

CREATE TABLE form_variants (
    id         uuid PRIMARY KEY,
    form_id    uuid        NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    key        text        NOT NULL CHECK (key IN ('A','B','C')),
    weight     int         NOT NULL CHECK (weight BETWEEN 1 AND 100),
    -- What the variant overrides ('{}' for A, which is the form itself).
    content    jsonb       NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (form_id, key)
);

-- Views, submissions and confirmations per variant ('' outside a test,
-- and for visitors a test doesn't count: hosted pages, other languages).
ALTER TABLE form_daily ADD COLUMN variant text NOT NULL DEFAULT '';
ALTER TABLE form_daily DROP CONSTRAINT form_daily_pkey;
ALTER TABLE form_daily ADD PRIMARY KEY (form_id, day, variant);
ALTER TABLE form_submissions ADD COLUMN variant text NOT NULL DEFAULT '';

-- Views of each step after the first (the first step's views are the
-- form's), per day and variant: the builder's step funnel.
CREATE TABLE form_step_daily (
    form_id uuid     NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    day     date     NOT NULL,
    variant text     NOT NULL DEFAULT '',
    step    smallint NOT NULL CHECK (step BETWEEN 1 AND 4),
    views   bigint   NOT NULL DEFAULT 0,
    PRIMARY KEY (form_id, day, variant, step)
);

-- +goose Down
DROP TABLE form_step_daily;
ALTER TABLE form_submissions DROP COLUMN variant;
INSERT INTO form_daily (form_id, day, variant, views, submissions, confirmed, blocked)
SELECT form_id, day, '~', sum(views), sum(submissions), sum(confirmed), sum(blocked)
FROM form_daily GROUP BY form_id, day;
DELETE FROM form_daily WHERE variant <> '~';
UPDATE form_daily SET variant = '';
ALTER TABLE form_daily DROP CONSTRAINT form_daily_pkey;
ALTER TABLE form_daily ADD PRIMARY KEY (form_id, day);
ALTER TABLE form_daily DROP COLUMN variant;
DROP TABLE form_variants;
ALTER TABLE forms DROP COLUMN ab;
