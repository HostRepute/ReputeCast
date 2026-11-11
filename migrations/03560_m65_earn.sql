-- +goose Up
-- Teams, pauses and sponsor self-booking (M65, G-38–G-40).
--
-- G-38: a tier can have group prices (per seat, with a seat range); a group
-- checkout makes the buyer's subscription with a quantity and a team
-- (earn_groups) whose seats (earn_group_seats) give invited people access
-- through their own paid_subscriptions row (kind and status 'group').
-- G-39: a Stripe subscription can be paused (Stripe's pause_collection):
-- no paid access and no MRR while paused, with its own MRR movements.
-- G-40: a slot can be booked by sponsors themselves: request, approval,
-- payment through Stripe Checkout before a deadline.

-- G-38 · group prices. A tier has at most one current price per period and
-- audience; a group price is per seat, for min to max seats.
ALTER TABLE earn_prices
    ADD COLUMN audience  text    NOT NULL DEFAULT 'individual' CHECK (audience IN ('individual','group')),
    ADD COLUMN seats_min integer,
    ADD COLUMN seats_max integer,
    ADD CONSTRAINT earn_prices_seats_check CHECK (
        (audience = 'individual' AND seats_min IS NULL AND seats_max IS NULL)
        OR (audience = 'group' AND seats_min BETWEEN 2 AND 1000 AND seats_max BETWEEN seats_min AND 1000));
DROP INDEX earn_prices_current;
CREATE UNIQUE INDEX earn_prices_current ON earn_prices (tier_id, period, audience) WHERE status = 'current';

-- A subscription's quantity (seats for a team; 1 otherwise) and the Stripe
-- item it's on; the team a seat belongs to; a pause: since when, until when
-- Stripe resumes it, and the last one (members pause at most once a year
-- from the cancel page). resume_notified_at is the pause_resumes_at the
-- "resuming soon" email was sent for.
ALTER TABLE paid_subscriptions
    ADD COLUMN quantity           integer     NOT NULL DEFAULT 1 CHECK (quantity >= 1),
    ADD COLUMN stripe_item_id     text        NOT NULL DEFAULT '',
    ADD COLUMN group_id           uuid,
    ADD COLUMN paused_at          timestamptz,
    ADD COLUMN pause_resumes_at   timestamptz,
    ADD COLUMN last_paused_at     timestamptz,
    ADD COLUMN resume_notified_at timestamptz;
ALTER TABLE paid_subscriptions DROP CONSTRAINT paid_subscriptions_kind_check;
ALTER TABLE paid_subscriptions ADD CONSTRAINT paid_subscriptions_kind_check CHECK (kind IN ('stripe','complimentary','gift','group'));
ALTER TABLE paid_subscriptions DROP CONSTRAINT paid_subscriptions_status_check;
ALTER TABLE paid_subscriptions ADD CONSTRAINT paid_subscriptions_status_check
    CHECK (status IN ('incomplete','trialing','active','past_due','canceled','complimentary','gift','group'));
ALTER TABLE paid_subscriptions DROP CONSTRAINT paid_subscriptions_source_kind_check;
ALTER TABLE paid_subscriptions ADD CONSTRAINT paid_subscriptions_source_kind_check
    CHECK (source_kind IN ('','campaign','automation','page','offer','website','import','gift','grant','direct','group'));
CREATE INDEX paid_subscriptions_paused_idx ON paid_subscriptions (pause_resumes_at) WHERE paused_at IS NOT NULL AND ended_at IS NULL;

-- A team: the buyer's subscription, its name and seats (the subscription's
-- quantity; the buyer uses one). over_notified_at: the buyer was told the
-- seats are fewer than the people in the team (nobody is removed
-- automatically). ended_at follows the buyer's subscription.
CREATE TABLE earn_groups (
    id               uuid PRIMARY KEY,
    workspace_id     uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    subscription_id  uuid        NOT NULL UNIQUE REFERENCES paid_subscriptions(id) ON DELETE CASCADE,
    owner_contact_id uuid        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    name             text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
    seats            integer     NOT NULL CHECK (seats >= 1),
    over_notified_at timestamptz,
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL,
    ended_at         timestamptz
);
CREATE INDEX earn_groups_ws_idx ON earn_groups (workspace_id, created_at DESC);
CREATE INDEX earn_groups_owner_idx ON earn_groups (owner_contact_id);

-- A seat's access row goes with its team.
ALTER TABLE paid_subscriptions ADD CONSTRAINT paid_subscriptions_group_fk FOREIGN KEY (group_id) REFERENCES earn_groups(id) ON DELETE CASCADE;
CREATE INDEX paid_subscriptions_group_idx ON paid_subscriptions (group_id) WHERE group_id IS NOT NULL;

-- The people a team gave a seat: invited (a link that works until
-- invite_expires_at; only the newest link, by its hash), active (accepted:
-- subscription_id gives access) or removed.
CREATE TABLE earn_group_seats (
    id                uuid PRIMARY KEY,
    group_id          uuid        NOT NULL REFERENCES earn_groups(id) ON DELETE CASCADE,
    workspace_id      uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email             text        NOT NULL,
    contact_id        uuid        REFERENCES contacts(id) ON DELETE SET NULL,
    subscription_id   uuid        REFERENCES paid_subscriptions(id) ON DELETE SET NULL,
    status            text        NOT NULL CHECK (status IN ('invited','active','removed')),
    invite_hash       bytea,
    invited_at        timestamptz NOT NULL,
    invite_sent_at    timestamptz NOT NULL,
    invite_expires_at timestamptz,
    accepted_at       timestamptz,
    removed_at        timestamptz
);
CREATE UNIQUE INDEX earn_group_seats_email ON earn_group_seats (group_id, lower(email)) WHERE status <> 'removed';
CREATE UNIQUE INDEX earn_group_seats_invite ON earn_group_seats (invite_hash) WHERE invite_hash IS NOT NULL;
CREATE INDEX earn_group_seats_contact_idx ON earn_group_seats (contact_id) WHERE contact_id IS NOT NULL;

-- A team checkout's seats and team name.
ALTER TABLE earn_checkouts
    ADD COLUMN quantity  integer NOT NULL DEFAULT 1 CHECK (quantity >= 1),
    ADD COLUMN team_name text    NOT NULL DEFAULT '';

-- G-39 · the pause lengths members may choose on the cancel page (months;
-- none: members can't pause), and the cancel page's new decision.
ALTER TABLE earn_settings
    ADD COLUMN pause_months integer[] NOT NULL DEFAULT '{}' CHECK (pause_months <@ ARRAY[1,3,6]);
ALTER TABLE earn_cancel_visits DROP CONSTRAINT earn_cancel_visits_decision_check;
ALTER TABLE earn_cancel_visits ADD CONSTRAINT earn_cancel_visits_decision_check CHECK (decision IN ('','kept','continued','paused'));

-- The new member emails: the team's (welcome to the buyer, the invite, the
-- end of a seat's access) and the pause's (paused, resuming soon).
ALTER TABLE earn_member_emails DROP CONSTRAINT earn_member_emails_email_check;
ALTER TABLE earn_member_emails ADD CONSTRAINT earn_member_emails_email_check
    CHECK (email IN ('welcome','trial_ending','payment_failed','renewal_reminder','canceled','gift_purchased','gift_payment_failed','gift_received','gift_ending',
                     'group_welcome','group_invite','group_access_ended','paused','resuming_soon'));

-- MRR movements: a pause takes a subscription's value to 0 without churn,
-- resuming brings it back, and a team's seat change moves it.
ALTER TABLE earn_mrr_changes DROP CONSTRAINT earn_mrr_changes_kind_check;
ALTER TABLE earn_mrr_changes ADD CONSTRAINT earn_mrr_changes_kind_check
    CHECK (kind IN ('new','reactivation','expansion','contraction','churn','paused','resumed','seats'));

-- Paid access, in one place (its Go mirror is earn.hasAccess): access
-- hasn't ended and isn't paused, and the row is trialing, active,
-- complimentary, a gift, a team's seat (unless the access rule paused the
-- buyer's, which the seat copies), or past due while Stripe retries
-- (unless the access rule paused it). Callers choose the mode.
-- +goose StatementBegin
CREATE FUNCTION earn_paid_access(kind text, status text, ended_at timestamptz, access_paused boolean, paused_at timestamptz)
RETURNS boolean LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT ended_at IS NULL AND paused_at IS NULL AND (
        status IN ('trialing','active','complimentary','gift')
        OR (status = 'group' AND kind = 'group' AND NOT access_paused)
        OR (status = 'past_due' AND NOT access_paused))
$$;
-- +goose StatementEnd

-- A subscription's yearly value under the MRR definition, now with its
-- quantity (a team pays per seat) and its pause (nothing while paused). A
-- discount's amountCents is the period's total after the discount. Seats
-- (kind group) count nothing: the buyer's subscription carries the team.
-- +goose StatementBegin
CREATE FUNCTION earn_arr_cents(kind text, mode text, status text, ended_at timestamptz, period text, amount bigint, discount jsonb, quantity integer, paused_at timestamptz)
RETURNS bigint LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE WHEN kind = 'stripe' AND mode = 'live' AND ended_at IS NULL AND paused_at IS NULL AND status IN ('active','past_due') THEN
        CASE WHEN period = 'year' THEN coalesce((discount->>'amountCents')::bigint, amount * quantity)
             ELSE coalesce((discount->>'amountCents')::bigint, amount * quantity) * 12 END
    ELSE 0 END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION earn_mrr_track() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    v_before bigint := 0;
    v_after  bigint;
    v_at     timestamptz;
    v_kind   text;
    v_source text := 'event';
BEGIN
    IF current_setting('reputecast.mrr_ledger', true) = 'off' THEN
        RETURN NULL;
    END IF;
    v_after := earn_arr_cents(NEW.kind, NEW.mode, NEW.status, NEW.ended_at, NEW.period, NEW.amount_cents, NEW.discount, NEW.quantity, NEW.paused_at);
    IF TG_OP = 'UPDATE' THEN
        v_before := earn_arr_cents(OLD.kind, OLD.mode, OLD.status, OLD.ended_at, OLD.period, OLD.amount_cents, OLD.discount, OLD.quantity, OLD.paused_at);
    END IF;
    IF v_before = v_after THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'INSERT' THEN
        v_at := least(coalesce(NEW.trial_end, NEW.started_at), NEW.created_at);
        IF NEW.import_id IS NOT NULL THEN
            v_source := 'import';
        END IF;
    ELSE
        v_at := coalesce(NEW.updated_at, now());
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.paused_at IS NULL AND NEW.paused_at IS NOT NULL AND v_after = 0 AND NEW.ended_at IS NULL THEN
        v_kind := 'paused';
    ELSIF TG_OP = 'UPDATE' AND OLD.paused_at IS NOT NULL AND NEW.paused_at IS NULL AND v_before = 0 THEN
        v_kind := 'resumed';
    ELSIF v_before = 0 THEN
        IF EXISTS (SELECT 1 FROM earn_mrr_changes c WHERE c.workspace_id = NEW.workspace_id AND c.contact_id = NEW.contact_id
                   AND c.kind = 'churn' AND c.at <= v_at AND c.at >= v_at - interval '90 days') THEN
            v_kind := 'reactivation';
        ELSE
            v_kind := 'new';
        END IF;
    ELSIF v_after = 0 THEN
        v_kind := CASE WHEN NEW.ended_at IS NOT NULL THEN 'churn' ELSE 'contraction' END;
    ELSIF TG_OP = 'UPDATE' AND OLD.quantity <> NEW.quantity THEN
        v_kind := 'seats';
    ELSIF v_after > v_before THEN
        v_kind := 'expansion';
    ELSE
        v_kind := 'contraction';
    END IF;
    INSERT INTO earn_mrr_changes (workspace_id, subscription_id, contact_id, tier_id, period, at, kind, arr_before, arr_after, source)
    VALUES (NEW.workspace_id, NEW.id, NEW.contact_id, NEW.tier_id, NEW.period, v_at, v_kind, v_before, v_after, v_source);
    RETURN NULL;
END
$$;
-- +goose StatementEnd

DROP FUNCTION earn_arr_cents(text, text, text, timestamptz, text, bigint, jsonb);

-- G-40 · sponsor self-booking. A self-serve slot is on the sponsor page with
-- its description; sponsors book at least lead_days ahead and pay within
-- pay_days of the approval.
ALTER TABLE sponsor_slots
    ADD COLUMN self_serve  boolean NOT NULL DEFAULT false,
    ADD COLUMN description text    NOT NULL DEFAULT '' CHECK (char_length(description) <= 300),
    ADD COLUMN lead_days   integer NOT NULL DEFAULT 7 CHECK (lead_days BETWEEN 0 AND 60),
    ADD COLUMN pay_days    integer NOT NULL DEFAULT 3 CHECK (pay_days BETWEEN 1 AND 14);

-- A sponsor's request (requested: doesn't hold the issue), the approval
-- waiting for payment (awaiting_payment: holds it until payment_due_at),
-- declined and expired. request keeps what the sponsor sent; the status
-- page's link is a random token whose hash is kept; checkout_id is the
-- Stripe Checkout session; paid_source says who recorded the payment.
ALTER TABLE sponsor_bookings DROP CONSTRAINT sponsor_bookings_status_check;
ALTER TABLE sponsor_bookings ADD CONSTRAINT sponsor_bookings_status_check
    CHECK (status IN ('proposed','hold','booked','sent','canceled','requested','awaiting_payment','declined','expired'));
DROP INDEX sponsor_bookings_issue;
CREATE UNIQUE INDEX sponsor_bookings_issue ON sponsor_bookings (slot_id, issue_number) WHERE status IN ('hold','booked','sent','awaiting_payment');
ALTER TABLE sponsor_bookings
    ADD COLUMN request           jsonb       NOT NULL DEFAULT '{}',
    ADD COLUMN status_token_hash bytea,
    ADD COLUMN payment_due_at    timestamptz,
    ADD COLUMN checkout_id       text        NOT NULL DEFAULT '',
    ADD COLUMN paid_source       text        NOT NULL DEFAULT '' CHECK (paid_source IN ('','manual','stripe')),
    ADD COLUMN declined_reason   text        NOT NULL DEFAULT '';
UPDATE sponsor_bookings SET paid_source = 'manual' WHERE payment_status = 'paid';
CREATE UNIQUE INDEX sponsor_bookings_status_token ON sponsor_bookings (status_token_hash) WHERE status_token_hash IS NOT NULL;
CREATE INDEX sponsor_bookings_due_idx ON sponsor_bookings (payment_due_at) WHERE status = 'awaiting_payment';
CREATE INDEX sponsor_bookings_requested_idx ON sponsor_bookings (workspace_id, created_at DESC) WHERE status = 'requested';

-- The sponsor page: on or off, what its media kit shows
-- ({"subscribers":true,"openRate":true,"clickRate":false,"about":"…","audience":"…"})
-- and the address sponsors write to.
CREATE TABLE sponsor_settings (
    workspace_id  uuid        PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    self_serve    boolean     NOT NULL DEFAULT false,
    media_kit     jsonb       NOT NULL DEFAULT '{}',
    contact_email text        NOT NULL DEFAULT '',
    updated_by    uuid        REFERENCES users(id) ON DELETE SET NULL,
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE sponsor_settings;
DROP INDEX sponsor_bookings_requested_idx;
DROP INDEX sponsor_bookings_due_idx;
DROP INDEX sponsor_bookings_status_token;
ALTER TABLE sponsor_bookings
    DROP COLUMN request, DROP COLUMN status_token_hash, DROP COLUMN payment_due_at, DROP COLUMN checkout_id,
    DROP COLUMN paid_source, DROP COLUMN declined_reason;
DELETE FROM sponsor_bookings WHERE status IN ('requested','awaiting_payment','declined','expired');
DROP INDEX sponsor_bookings_issue;
CREATE UNIQUE INDEX sponsor_bookings_issue ON sponsor_bookings (slot_id, issue_number) WHERE status IN ('hold','booked','sent');
ALTER TABLE sponsor_bookings DROP CONSTRAINT sponsor_bookings_status_check;
ALTER TABLE sponsor_bookings ADD CONSTRAINT sponsor_bookings_status_check CHECK (status IN ('proposed','hold','booked','sent','canceled'));
ALTER TABLE sponsor_slots DROP COLUMN self_serve, DROP COLUMN description, DROP COLUMN lead_days, DROP COLUMN pay_days;

-- +goose StatementBegin
CREATE FUNCTION earn_arr_cents(kind text, mode text, status text, ended_at timestamptz, period text, amount bigint, discount jsonb)
RETURNS bigint LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE WHEN kind = 'stripe' AND mode = 'live' AND ended_at IS NULL AND status IN ('active','past_due') THEN
        CASE WHEN period = 'year' THEN coalesce((discount->>'amountCents')::bigint, amount)
             ELSE coalesce((discount->>'amountCents')::bigint, amount) * 12 END
    ELSE 0 END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION earn_mrr_track() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    v_before bigint := 0;
    v_after  bigint;
    v_at     timestamptz;
    v_kind   text;
    v_source text := 'event';
BEGIN
    IF current_setting('reputecast.mrr_ledger', true) = 'off' THEN
        RETURN NULL;
    END IF;
    v_after := earn_arr_cents(NEW.kind, NEW.mode, NEW.status, NEW.ended_at, NEW.period, NEW.amount_cents, NEW.discount);
    IF TG_OP = 'UPDATE' THEN
        v_before := earn_arr_cents(OLD.kind, OLD.mode, OLD.status, OLD.ended_at, OLD.period, OLD.amount_cents, OLD.discount);
    END IF;
    IF v_before = v_after THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'INSERT' THEN
        v_at := least(coalesce(NEW.trial_end, NEW.started_at), NEW.created_at);
        IF NEW.import_id IS NOT NULL THEN
            v_source := 'import';
        END IF;
    ELSE
        v_at := coalesce(NEW.updated_at, now());
    END IF;
    IF v_before = 0 THEN
        IF EXISTS (SELECT 1 FROM earn_mrr_changes c WHERE c.workspace_id = NEW.workspace_id AND c.contact_id = NEW.contact_id
                   AND c.kind = 'churn' AND c.at <= v_at AND c.at >= v_at - interval '90 days') THEN
            v_kind := 'reactivation';
        ELSE
            v_kind := 'new';
        END IF;
    ELSIF v_after = 0 THEN
        v_kind := CASE WHEN NEW.ended_at IS NOT NULL THEN 'churn' ELSE 'contraction' END;
    ELSIF v_after > v_before THEN
        v_kind := 'expansion';
    ELSE
        v_kind := 'contraction';
    END IF;
    INSERT INTO earn_mrr_changes (workspace_id, subscription_id, contact_id, tier_id, period, at, kind, arr_before, arr_after, source)
    VALUES (NEW.workspace_id, NEW.id, NEW.contact_id, NEW.tier_id, NEW.period, v_at, v_kind, v_before, v_after, v_source);
    RETURN NULL;
END
$$;
-- +goose StatementEnd

DROP FUNCTION earn_arr_cents(text, text, text, timestamptz, text, bigint, jsonb, integer, timestamptz);
DROP FUNCTION earn_paid_access(text, text, timestamptz, boolean, timestamptz);
DELETE FROM earn_mrr_changes WHERE kind IN ('paused','resumed','seats');
ALTER TABLE earn_mrr_changes DROP CONSTRAINT earn_mrr_changes_kind_check;
ALTER TABLE earn_mrr_changes ADD CONSTRAINT earn_mrr_changes_kind_check CHECK (kind IN ('new','reactivation','expansion','contraction','churn'));
DELETE FROM earn_member_emails WHERE email IN ('group_welcome','group_invite','group_access_ended','paused','resuming_soon');
ALTER TABLE earn_member_emails DROP CONSTRAINT earn_member_emails_email_check;
ALTER TABLE earn_member_emails ADD CONSTRAINT earn_member_emails_email_check
    CHECK (email IN ('welcome','trial_ending','payment_failed','renewal_reminder','canceled','gift_purchased','gift_payment_failed','gift_received','gift_ending'));
UPDATE earn_cancel_visits SET decision = '' WHERE decision = 'paused';
ALTER TABLE earn_cancel_visits DROP CONSTRAINT earn_cancel_visits_decision_check;
ALTER TABLE earn_cancel_visits ADD CONSTRAINT earn_cancel_visits_decision_check CHECK (decision IN ('','kept','continued'));
ALTER TABLE earn_settings DROP COLUMN pause_months;
ALTER TABLE earn_checkouts DROP COLUMN quantity, DROP COLUMN team_name;
DROP TABLE earn_group_seats;
DELETE FROM paid_subscriptions WHERE kind = 'group';
ALTER TABLE paid_subscriptions DROP CONSTRAINT paid_subscriptions_group_fk;
DROP TABLE earn_groups;
DROP INDEX paid_subscriptions_paused_idx;
ALTER TABLE paid_subscriptions DROP CONSTRAINT paid_subscriptions_source_kind_check;
ALTER TABLE paid_subscriptions ADD CONSTRAINT paid_subscriptions_source_kind_check
    CHECK (source_kind IN ('','campaign','automation','page','offer','website','import','gift','grant','direct'));
ALTER TABLE paid_subscriptions DROP CONSTRAINT paid_subscriptions_status_check;
ALTER TABLE paid_subscriptions ADD CONSTRAINT paid_subscriptions_status_check
    CHECK (status IN ('incomplete','trialing','active','past_due','canceled','complimentary','gift'));
ALTER TABLE paid_subscriptions DROP CONSTRAINT paid_subscriptions_kind_check;
ALTER TABLE paid_subscriptions ADD CONSTRAINT paid_subscriptions_kind_check CHECK (kind IN ('stripe','complimentary','gift'));
ALTER TABLE paid_subscriptions
    DROP COLUMN quantity, DROP COLUMN stripe_item_id, DROP COLUMN group_id, DROP COLUMN paused_at,
    DROP COLUMN pause_resumes_at, DROP COLUMN last_paused_at, DROP COLUMN resume_notified_at;
DELETE FROM earn_prices WHERE audience = 'group';
DROP INDEX earn_prices_current;
CREATE UNIQUE INDEX earn_prices_current ON earn_prices (tier_id, period) WHERE status = 'current';
ALTER TABLE earn_prices DROP CONSTRAINT earn_prices_seats_check, DROP COLUMN audience, DROP COLUMN seats_min, DROP COLUMN seats_max;
