-- +goose Up
-- Sponsorships (design expansion phase 6, M27): the sponsor library, slots
-- (a placement in each issue of a list), bookings by issue and slot with
-- their creative and the payment status recorded by hand, the sponsor
-- reports and their read-only links. Sponsorships work without a payment
-- provider; ReputeCast sends no invoices and moves no money.

CREATE TABLE sponsors (
    id            uuid PRIMARY KEY,
    workspace_id  uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          text        NOT NULL,
    website       text        NOT NULL DEFAULT '',
    logo_url      text        NOT NULL DEFAULT '',
    contact_name  text        NOT NULL DEFAULT '',
    contact_email text        NOT NULL DEFAULT '',
    -- Payments are due this many days after the issue.
    terms_days    integer     NOT NULL DEFAULT 14 CHECK (terms_days BETWEEN 0 AND 120),
    notes         text        NOT NULL DEFAULT '',
    created_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
CREATE UNIQUE INDEX sponsors_name ON sponsors (workspace_id, lower(name)) WHERE deleted_at IS NULL;

-- A slot is a placement in each issue of a list ("Main sponsor", after the
-- first story), with a default price and one booking per issue. Issues
-- follow the list's schedule: every N days, numbered from first_number on
-- first_on (the workspace's local date).
CREATE TABLE sponsor_slots (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    list_id      uuid        NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    name         text        NOT NULL,
    placement    text        NOT NULL DEFAULT '',
    price_cents  bigint      NOT NULL CHECK (price_cents >= 0),
    currency     text        NOT NULL DEFAULT 'usd',
    first_number integer     NOT NULL CHECK (first_number >= 1),
    first_on     date        NOT NULL,
    every_days   integer     NOT NULL DEFAULT 7 CHECK (every_days BETWEEN 1 AND 31),
    -- The creative is due this many days before the issue.
    creative_days integer    NOT NULL DEFAULT 4 CHECK (creative_days BETWEEN 0 AND 30),
    selling      boolean     NOT NULL DEFAULT true,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);
CREATE INDEX sponsor_slots_ws_idx ON sponsor_slots (workspace_id) WHERE deleted_at IS NULL;

-- A booking: a sponsor in a slot of one issue. An issue without a booking
-- is open; a hold keeps it until a date, then the issue opens again. A
-- proposal is an MCP agent's suggestion for a person to confirm.
CREATE TABLE sponsor_bookings (
    id              uuid PRIMARY KEY,
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slot_id         uuid        NOT NULL REFERENCES sponsor_slots(id) ON DELETE CASCADE,
    issue_number    integer     NOT NULL,
    issue_on        date        NOT NULL,
    sponsor_id      uuid        REFERENCES sponsors(id) ON DELETE SET NULL,
    -- The issue's campaign: linked by list and date, or by hand.
    campaign_id     uuid        REFERENCES campaigns(id) ON DELETE SET NULL,
    campaign_by_hand boolean    NOT NULL DEFAULT false,
    price_cents     bigint      NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    currency        text        NOT NULL DEFAULT 'usd',
    status          text        NOT NULL CHECK (status IN ('proposed','hold','booked','sent','canceled')),
    hold_until      date,
    -- The creative: {"logo":"…","image":"…","headline":"…","text":"…","button":"…","link":"…",
    -- "utm":{"source":"…","medium":"…","campaign":"…"}}.
    creative        jsonb       NOT NULL DEFAULT '{}',
    creative_status text        NOT NULL DEFAULT 'waiting' CHECK (creative_status IN ('waiting','ready')),
    creative_due    date,
    ready_at        timestamptz,
    ready_by        uuid        REFERENCES users(id) ON DELETE SET NULL,
    -- Payment, recorded by hand: unpaid or paid (overdue once the due date passes).
    payment_status  text        NOT NULL DEFAULT 'unpaid' CHECK (payment_status IN ('unpaid','paid')),
    payment_due     date,
    paid_on         date,
    paid_cents      bigint,
    paid_note       text        NOT NULL DEFAULT '',
    paid_by         uuid        REFERENCES users(id) ON DELETE SET NULL,
    paid_recorded_at timestamptz,
    notes           text        NOT NULL DEFAULT '',
    cancel_reason   text        NOT NULL DEFAULT '',
    canceled_at     timestamptz,
    canceled_by     uuid        REFERENCES users(id) ON DELETE SET NULL,
    sent_at         timestamptz,
    -- An agent's proposal: who, when, and the agent's note.
    proposed_by     uuid,
    proposed_name   text        NOT NULL DEFAULT '',
    proposed_note   text        NOT NULL DEFAULT '',
    -- Reminders already sent: creative due tomorrow, creative late, payment overdue.
    notified        text[]      NOT NULL DEFAULT '{}',
    created_by      uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
-- One booking (or hold) per issue and slot.
CREATE UNIQUE INDEX sponsor_bookings_issue ON sponsor_bookings (slot_id, issue_number) WHERE status IN ('hold','booked','sent');
CREATE INDEX sponsor_bookings_ws_idx ON sponsor_bookings (workspace_id, issue_on);
CREATE INDEX sponsor_bookings_campaign_idx ON sponsor_bookings (campaign_id) WHERE campaign_id IS NOT NULL;
CREATE INDEX sponsor_bookings_sponsor_idx ON sponsor_bookings (sponsor_id, issue_on);

-- A booking's history, for the panel's Activity.
CREATE TABLE sponsor_booking_events (
    id          uuid PRIMARY KEY,
    booking_id  uuid        NOT NULL REFERENCES sponsor_bookings(id) ON DELETE CASCADE,
    kind        text        NOT NULL,
    occurred_at timestamptz NOT NULL,
    actor_id    uuid        REFERENCES users(id) ON DELETE SET NULL,
    actor_name  text        NOT NULL DEFAULT '',
    data        jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX sponsor_booking_events_idx ON sponsor_booking_events (booking_id, occurred_at DESC);

-- A booking's report: who got the block, their human opens, the unique
-- clickers and clicks on the sponsor's links. Machine opens and bot clicks
-- are left out. Refreshed while results come in and frozen 72 hours after
-- the send.
CREATE TABLE sponsor_report_stats (
    booking_id      uuid        PRIMARY KEY REFERENCES sponsor_bookings(id) ON DELETE CASCADE,
    sent_at         timestamptz,
    delivered       bigint      NOT NULL DEFAULT 0,
    human_opens     bigint      NOT NULL DEFAULT 0,
    unique_clickers bigint      NOT NULL DEFAULT 0,
    clicks          bigint      NOT NULL DEFAULT 0,
    -- [{"kind":"button","label":"Try Northwind free","clicks":431},{"kind":"logo","clicks":67}]
    links           jsonb       NOT NULL DEFAULT '[]',
    -- Only the recipients who got the block count (Visibility).
    visibility_note text        NOT NULL DEFAULT '',
    final           boolean     NOT NULL DEFAULT false,
    computed_at     timestamptz NOT NULL
);

-- Read-only links to a sponsor report: one booking, or every booking of a
-- sponsor. Like the campaign report's: optional expiry, revocable.
CREATE TABLE sponsor_report_shares (
    id           uuid PRIMARY KEY,
    workspace_id uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    booking_id   uuid        REFERENCES sponsor_bookings(id) ON DELETE CASCADE,
    sponsor_id   uuid        REFERENCES sponsors(id) ON DELETE CASCADE,
    token_hash   bytea       NOT NULL,
    token_sealed bytea,
    created_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL,
    expires_at   timestamptz,
    revoked_at   timestamptz,
    revoked_by   uuid        REFERENCES users(id) ON DELETE SET NULL,
    views        bigint      NOT NULL DEFAULT 0,
    last_viewed_at timestamptz,
    CHECK (booking_id IS NOT NULL OR sponsor_id IS NOT NULL)
);
CREATE UNIQUE INDEX sponsor_report_shares_token ON sponsor_report_shares (token_hash);
CREATE INDEX sponsor_report_shares_booking_idx ON sponsor_report_shares (booking_id) WHERE booking_id IS NOT NULL;
CREATE INDEX sponsor_report_shares_sponsor_idx ON sponsor_report_shares (sponsor_id) WHERE sponsor_id IS NOT NULL;

-- +goose Down
DROP TABLE sponsor_report_shares;
DROP TABLE sponsor_report_stats;
DROP TABLE sponsor_booking_events;
DROP TABLE sponsor_bookings;
DROP TABLE sponsor_slots;
DROP TABLE sponsors;
