-- Sponsorships (internal/sponsors): the sponsor library, slots, bookings
-- with their creative and payment, booking events, report stats and the
-- read-only report links.

-- name: SponsorList :many
SELECT * FROM sponsors WHERE workspace_id = @workspace_id AND deleted_at IS NULL ORDER BY lower(name);

-- name: SponsorGet :one
SELECT * FROM sponsors WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: SponsorNameTaken :one
SELECT EXISTS (SELECT 1 FROM sponsors WHERE workspace_id = @workspace_id AND lower(name) = lower(@name::text) AND id <> @id AND deleted_at IS NULL)::boolean;

-- name: SponsorInsert :exec
INSERT INTO sponsors (id, workspace_id, name, website, logo_url, contact_name, contact_email, terms_days, notes, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, @website, @logo_url, @contact_name, @contact_email, @terms_days, @notes, sqlc.narg(created_by), @at, @at);

-- name: SponsorUpdate :exec
UPDATE sponsors SET name = @name, website = @website, logo_url = @logo_url, contact_name = @contact_name, contact_email = @contact_email,
    terms_days = @terms_days, notes = @notes, updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: SponsorDelete :exec
UPDATE sponsors SET deleted_at = @at::timestamptz, updated_at = @at WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: SponsorSlots :many
SELECT s.*, l.name AS list_name
FROM sponsor_slots s JOIN lists l ON l.id = s.list_id
WHERE s.workspace_id = @workspace_id AND s.deleted_at IS NULL
ORDER BY s.created_at;

-- name: SponsorSlotGet :one
SELECT s.*, l.name AS list_name
FROM sponsor_slots s JOIN lists l ON l.id = s.list_id
WHERE s.id = @id AND s.workspace_id = @workspace_id AND s.deleted_at IS NULL;

-- name: SponsorSlotInsert :exec
INSERT INTO sponsor_slots (id, workspace_id, list_id, name, placement, price_cents, currency, first_number, first_on, every_days, creative_days, selling, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @list_id, @name, @placement, @price_cents, @currency, @first_number, @first_on, @every_days, @creative_days, @selling, sqlc.narg(created_by), @at, @at);

-- name: SponsorSlotUpdate :exec
UPDATE sponsor_slots SET list_id = @list_id, name = @name, placement = @placement, price_cents = @price_cents, currency = @currency,
    first_number = @first_number, first_on = @first_on, every_days = @every_days, creative_days = @creative_days, selling = @selling, updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: SponsorSlotDelete :exec
UPDATE sponsor_slots SET deleted_at = @at::timestamptz, updated_at = @at WHERE id = @id AND workspace_id = @workspace_id AND deleted_at IS NULL;

-- name: SponsorBookingsOfSlot :many
-- A slot's bookings of a range of issues, with their sponsor and campaign.
SELECT b.*, coalesce(sp.name, '')::text AS sponsor_name, coalesce(sp.contact_name, '')::text AS sponsor_contact,
    coalesce(sp.contact_email, '')::text AS sponsor_email, coalesce(sp.terms_days, 14)::integer AS sponsor_terms,
    coalesce(c.name, '')::text AS campaign_name, coalesce(c.status, '')::text AS campaign_status,
    c.scheduled_at AS campaign_scheduled_at, c.started_at AS campaign_started_at
FROM sponsor_bookings b
LEFT JOIN sponsors sp ON sp.id = b.sponsor_id
LEFT JOIN campaigns c ON c.id = b.campaign_id AND c.deleted_at IS NULL
WHERE b.slot_id = @slot_id AND b.issue_number BETWEEN @from_number AND @to_number AND b.status <> 'canceled'
ORDER BY b.issue_number, b.created_at;

-- name: SponsorBookingsOfWorkspace :many
SELECT b.*, coalesce(sp.name, '')::text AS sponsor_name, coalesce(sp.contact_name, '')::text AS sponsor_contact,
    coalesce(sp.contact_email, '')::text AS sponsor_email, coalesce(sp.terms_days, 14)::integer AS sponsor_terms,
    coalesce(c.name, '')::text AS campaign_name, coalesce(c.status, '')::text AS campaign_status,
    c.scheduled_at AS campaign_scheduled_at, c.started_at AS campaign_started_at
FROM sponsor_bookings b
LEFT JOIN sponsors sp ON sp.id = b.sponsor_id
LEFT JOIN campaigns c ON c.id = b.campaign_id AND c.deleted_at IS NULL
WHERE b.workspace_id = @workspace_id AND b.status <> 'canceled'
ORDER BY b.issue_on, b.issue_number;

-- name: SponsorBookingGet :one
SELECT b.*, coalesce(sp.name, '')::text AS sponsor_name, coalesce(sp.contact_name, '')::text AS sponsor_contact,
    coalesce(sp.contact_email, '')::text AS sponsor_email, coalesce(sp.terms_days, 14)::integer AS sponsor_terms,
    coalesce(c.name, '')::text AS campaign_name, coalesce(c.status, '')::text AS campaign_status,
    c.scheduled_at AS campaign_scheduled_at, c.started_at AS campaign_started_at
FROM sponsor_bookings b
LEFT JOIN sponsors sp ON sp.id = b.sponsor_id
LEFT JOIN campaigns c ON c.id = b.campaign_id AND c.deleted_at IS NULL
WHERE b.id = @id AND b.workspace_id = @workspace_id;

-- name: SponsorBookingOfIssue :one
SELECT * FROM sponsor_bookings WHERE slot_id = @slot_id AND issue_number = @issue_number AND status IN ('hold','booked','sent')
ORDER BY created_at DESC LIMIT 1;

-- name: SponsorBookingProposals :many
SELECT * FROM sponsor_bookings WHERE slot_id = @slot_id AND issue_number = @issue_number AND status = 'proposed' ORDER BY created_at DESC;

-- name: SponsorBookingOfCampaign :one
SELECT b.*, coalesce(sp.name, '')::text AS sponsor_name, coalesce(sp.website, '')::text AS sponsor_website, coalesce(sp.logo_url, '')::text AS sponsor_logo
FROM sponsor_bookings b
LEFT JOIN sponsors sp ON sp.id = b.sponsor_id
WHERE b.campaign_id = @campaign_id AND b.status IN ('booked','sent')
ORDER BY b.issue_number DESC LIMIT 1;

-- name: SponsorBookingInsert :exec
INSERT INTO sponsor_bookings (id, workspace_id, slot_id, issue_number, issue_on, sponsor_id, campaign_id, campaign_by_hand, price_cents, currency, status,
    hold_until, creative, creative_status, creative_due, payment_due, notes, proposed_by, proposed_name, proposed_note, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @slot_id, @issue_number, @issue_on, sqlc.narg(sponsor_id), sqlc.narg(campaign_id), @campaign_by_hand, @price_cents, @currency, @status,
    sqlc.narg(hold_until), @creative, @creative_status, sqlc.narg(creative_due), sqlc.narg(payment_due), @notes, sqlc.narg(proposed_by), @proposed_name, @proposed_note,
    sqlc.narg(created_by), @at, @at);

-- name: SponsorBookingSetTerms :exec
-- The booking's sponsor, price, status, hold and dates.
UPDATE sponsor_bookings SET sponsor_id = sqlc.narg(sponsor_id), price_cents = @price_cents, status = @status, hold_until = sqlc.narg(hold_until),
    creative_due = sqlc.narg(creative_due), payment_due = sqlc.narg(payment_due), updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: SponsorBookingSetCampaign :exec
UPDATE sponsor_bookings SET campaign_id = sqlc.narg(campaign_id), campaign_by_hand = @campaign_by_hand, updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: SponsorBookingSetNotes :exec
UPDATE sponsor_bookings SET notes = @notes, updated_at = @at WHERE id = @id AND workspace_id = @workspace_id;

-- name: SponsorBookingSetCreative :exec
UPDATE sponsor_bookings SET creative = @creative, creative_status = @creative_status, ready_at = sqlc.narg(ready_at), ready_by = sqlc.narg(ready_by), updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: SponsorBookingSetPayment :exec
UPDATE sponsor_bookings SET payment_status = @payment_status, paid_on = sqlc.narg(paid_on), paid_cents = sqlc.narg(paid_cents), paid_note = @paid_note,
    paid_by = sqlc.narg(paid_by), paid_recorded_at = sqlc.narg(paid_recorded_at), updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: SponsorBookingCancel :exec
UPDATE sponsor_bookings SET status = 'canceled', cancel_reason = @cancel_reason, canceled_at = @at::timestamptz, canceled_by = sqlc.narg(canceled_by), updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: SponsorBookingSent :exec
UPDATE sponsor_bookings SET status = 'sent', sent_at = @sent_at::timestamptz, updated_at = @at WHERE id = @id AND status = 'booked';

-- name: SponsorBookingNotified :exec
UPDATE sponsor_bookings SET notified = array_append(notified, @what::text) WHERE id = @id AND NOT (@what::text = ANY(notified));

-- name: SponsorBookingsToSend :many
-- Booked issues whose campaign has been sent.
SELECT b.id, c.started_at FROM sponsor_bookings b JOIN campaigns c ON c.id = b.campaign_id
WHERE b.status = 'booked' AND c.status IN ('sent','sending') AND c.started_at IS NOT NULL;

-- name: SponsorHoldsLapsed :many
SELECT * FROM sponsor_bookings WHERE status = 'hold' AND hold_until IS NOT NULL AND hold_until < @today::date;

-- name: SponsorBookingsDue :many
-- Booked issues and unpaid sent issues, for the creative and payment
-- reminders.
SELECT b.*, coalesce(sp.name, '')::text AS sponsor_name, w.slug AS workspace_slug, w.timezone AS workspace_timezone
FROM sponsor_bookings b
JOIN workspaces w ON w.id = b.workspace_id
LEFT JOIN sponsors sp ON sp.id = b.sponsor_id
WHERE (b.status = 'booked' AND b.creative_status = 'waiting') OR (b.status IN ('booked','sent') AND b.payment_status = 'unpaid' AND b.payment_due IS NOT NULL);

-- name: SponsorBookingEvents :many
SELECT * FROM sponsor_booking_events WHERE booking_id = @booking_id ORDER BY occurred_at DESC, id DESC LIMIT 50;

-- name: SponsorBookingEventInsert :exec
INSERT INTO sponsor_booking_events (id, booking_id, kind, occurred_at, actor_id, actor_name, data)
VALUES (@id, @booking_id, @kind, @occurred_at, sqlc.narg(actor_id), @actor_name, @data);

-- name: SponsorReportStats :one
SELECT * FROM sponsor_report_stats WHERE booking_id = @booking_id;

-- name: SponsorReportStatsOf :many
SELECT * FROM sponsor_report_stats WHERE booking_id = ANY(@ids::uuid[]);

-- name: SponsorReportStatsUpsert :exec
INSERT INTO sponsor_report_stats (booking_id, sent_at, delivered, human_opens, unique_clickers, clicks, links, visibility_note, final, computed_at)
VALUES (@booking_id, sqlc.narg(sent_at), @delivered, @human_opens, @unique_clickers, @clicks, @links, @visibility_note, @final, @computed_at)
ON CONFLICT (booking_id) DO UPDATE SET sent_at = EXCLUDED.sent_at, delivered = EXCLUDED.delivered, human_opens = EXCLUDED.human_opens,
    unique_clickers = EXCLUDED.unique_clickers, clicks = EXCLUDED.clicks, links = EXCLUDED.links, visibility_note = EXCLUDED.visibility_note,
    final = EXCLUDED.final, computed_at = EXCLUDED.computed_at;

-- name: SponsorReportsToRefresh :many
-- Sent bookings whose report isn't final yet.
SELECT b.id, b.workspace_id FROM sponsor_bookings b
LEFT JOIN sponsor_report_stats s ON s.booking_id = b.id
WHERE b.status = 'sent' AND b.campaign_id IS NOT NULL AND (s.booking_id IS NULL OR NOT s.final);

-- name: SponsorShareInsert :exec
INSERT INTO sponsor_report_shares (id, workspace_id, booking_id, sponsor_id, token_hash, token_sealed, created_by, created_at, expires_at)
VALUES (@id, @workspace_id, sqlc.narg(booking_id), sqlc.narg(sponsor_id), @token_hash, @token_sealed, sqlc.narg(created_by), @created_at, sqlc.narg(expires_at));

-- name: SponsorSharesOf :many
-- A booking's or a sponsor's active links, newest first.
SELECT sh.*, coalesce(u.name, '')::text AS created_by_name
FROM sponsor_report_shares sh LEFT JOIN users u ON u.id = sh.created_by
WHERE sh.workspace_id = @workspace_id AND (sh.booking_id = sqlc.narg(booking_id) OR sh.sponsor_id = sqlc.narg(sponsor_id))
  AND sh.revoked_at IS NULL AND (sh.expires_at IS NULL OR sh.expires_at > @now::timestamptz)
ORDER BY sh.created_at DESC;

-- name: SponsorShareGet :one
SELECT * FROM sponsor_report_shares WHERE id = @id AND workspace_id = @workspace_id;

-- name: SponsorShareByToken :one
SELECT * FROM sponsor_report_shares WHERE token_hash = @token_hash;

-- name: SponsorShareRevoke :exec
UPDATE sponsor_report_shares SET revoked_at = @at::timestamptz, revoked_by = sqlc.narg(revoked_by) WHERE id = @id AND workspace_id = @workspace_id AND revoked_at IS NULL;

-- name: SponsorShareTouch :exec
UPDATE sponsor_report_shares SET views = views + 1, last_viewed_at = @at::timestamptz WHERE id = @id;

-- name: SponsorCampaignsOfList :many
-- A list's campaigns scheduled or sent in a time range, to link issues by
-- list and date.
SELECT c.id, c.name, c.status, c.scheduled_at, c.started_at
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND c.status NOT IN ('cancelled','failed')
  AND coalesce(c.started_at, c.scheduled_at) >= @from_at::timestamptz AND coalesce(c.started_at, c.scheduled_at) < @to_at::timestamptz
  AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(coalesce(c.audience->'lists', '[]'::jsonb)) x(id) WHERE x.id = @list_id::text)
ORDER BY coalesce(c.started_at, c.scheduled_at);

-- name: SponsorCampaignBrief :one
SELECT c.id, c.name, c.status, c.audience, c.scheduled_at, c.started_at, c.mjml, coalesce(u.name, '')::text AS author
FROM campaigns c LEFT JOIN users u ON u.id = c.created_by
WHERE c.id = @id AND c.workspace_id = @workspace_id AND c.deleted_at IS NULL;

-- name: SponsorCampaignChips :many
-- The sponsor of each campaign with a booking: the Campaigns list's chip.
SELECT b.campaign_id, sp.name AS sponsor
FROM sponsor_bookings b JOIN sponsors sp ON sp.id = b.sponsor_id
WHERE b.workspace_id = @workspace_id AND b.campaign_id IS NOT NULL AND b.status IN ('booked','sent')
ORDER BY b.issue_on DESC;
