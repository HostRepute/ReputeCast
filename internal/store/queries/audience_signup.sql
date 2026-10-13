-- Audience · public self-service (internal/audience/signup.go): signups from
-- forms and the hosted page, double opt-in confirmations, the preference
-- center and unsubscribes. The contact acts on these; there is no team
-- member behind them, so consent events carry the IP and browser instead.

-- name: SignupContactByEmail :one
SELECT * FROM contacts
WHERE workspace_id = @workspace_id AND email = @email AND erased_at IS NULL
FOR UPDATE;

-- name: SignupContactByID :one
SELECT * FROM contacts
WHERE workspace_id = @workspace_id AND id = @id AND erased_at IS NULL
FOR UPDATE;

-- name: SignupInsertContact :one
-- A concurrent signup for the same address wins the race; the caller then
-- reads the contact it created.
INSERT INTO contacts (id, workspace_id, email, first_name, last_name, company, mailbox_provider, consent, source,
                      tags, attributes, role_address, created_at, updated_at, last_activity_at)
VALUES (@id, @workspace_id, @email, @first_name, @last_name, @company, @mailbox_provider, @consent, @source,
        '{}', @attributes, @role_address, @at::timestamptz, @at::timestamptz, @at::timestamptz)
ON CONFLICT (workspace_id, email) DO NOTHING
RETURNING *;

-- name: SignupFillContact :exec
-- Fills fields the contact left empty; a public form never overwrites what
-- is already known (anyone can type anyone's address into a form).
UPDATE contacts
SET first_name = CASE WHEN first_name = '' THEN @first_name ELSE first_name END,
    last_name = CASE WHEN last_name = '' THEN @last_name ELSE last_name END,
    company = CASE WHEN company = '' THEN @company ELSE company END,
    attributes = @attributes || attributes,
    updated_at = @at
WHERE id = @id;

-- name: SignupSetFirstName :exec
UPDATE contacts SET first_name = @first_name, updated_at = @at WHERE id = @id;

-- name: SignupSetContactState :exec
UPDATE contacts SET status = @status, consent = @consent, updated_at = @at::timestamptz, last_activity_at = @at::timestamptz WHERE id = @id;

-- name: SignupTouchContact :exec
UPDATE contacts SET last_activity_at = @at::timestamptz WHERE id = @id;

-- name: SignupMemberships :many
SELECT list_id, status, subscribed_at, unsubscribed_at FROM list_memberships WHERE contact_id = @contact_id ORDER BY list_id;

-- name: SignupSetMembership :exec
INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at, unsubscribed_at, created_at)
VALUES (@list_id, @contact_id, @status, @source,
        CASE WHEN @status::text = 'subscribed' THEN @at::timestamptz END,
        CASE WHEN @status::text = 'unsubscribed' THEN @at::timestamptz END, @at)
ON CONFLICT (list_id, contact_id) DO UPDATE
SET status = excluded.status,
    source = CASE WHEN excluded.status = 'unsubscribed' THEN list_memberships.source ELSE excluded.source END,
    subscribed_at = CASE WHEN excluded.status = 'subscribed' AND list_memberships.status <> 'subscribed' THEN @at::timestamptz
                         ELSE list_memberships.subscribed_at END,
    unsubscribed_at = CASE WHEN excluded.status = 'unsubscribed' THEN @at::timestamptz
                           WHEN excluded.status = 'subscribed' THEN NULL
                           ELSE list_memberships.unsubscribed_at END;

-- name: SignupConfirmPending :many
UPDATE list_memberships
SET status = 'subscribed', subscribed_at = @at::timestamptz, unsubscribed_at = NULL
WHERE contact_id = @contact_id AND status = 'pending'
RETURNING list_id;

-- name: SignupUnsubscribeAll :many
UPDATE list_memberships
SET status = 'unsubscribed', unsubscribed_at = @at::timestamptz
WHERE contact_id = @contact_id AND status <> 'unsubscribed'
RETURNING list_id;

-- name: SignupInsertConsent :exec
INSERT INTO consent_events (id, contact_id, list_id, action, source, ip, user_agent, detail, note, consent_version, consent_text, occurred_at)
VALUES (@id, @contact_id, @list_id, @action, @source, @ip, @user_agent, @detail, @note, @consent_version, @consent_text, @occurred_at);

-- name: SignupLastConsent :one
-- The latest consent event of one kind (the confirmation shown under the
-- confirmation page, the opt-in that recorded the consent text).
SELECT occurred_at, ip, source, consent_version, consent_text
FROM consent_events
WHERE contact_id = @contact_id AND action = @action
ORDER BY occurred_at DESC, id DESC
LIMIT 1;

-- name: SignupLastUnsubscribedLists :many
-- The lists the latest unsubscribe left, for "Resubscribe".
SELECT DISTINCT ce.list_id::uuid AS list_id
FROM consent_events ce
WHERE ce.contact_id = @contact_id AND ce.action = 'unsubscribed' AND ce.list_id IS NOT NULL
  AND ce.occurred_at >= (SELECT max(x.occurred_at) - interval '1 minute' FROM consent_events x
                         WHERE x.contact_id = ce.contact_id AND x.action = 'unsubscribed');

-- name: SignupPreferences :one
SELECT frequency, paused_until FROM contact_preferences WHERE contact_id = @contact_id;

-- name: SignupUpsertPreferences :exec
INSERT INTO contact_preferences (contact_id, frequency, paused_until, updated_at)
VALUES (@contact_id, @frequency, @paused_until, @at)
ON CONFLICT (contact_id) DO UPDATE
SET frequency = excluded.frequency, paused_until = excluded.paused_until, updated_at = excluded.updated_at;

-- name: SignupSuppression :one
SELECT * FROM suppressions WHERE workspace_id = @workspace_id AND email = @email;

-- name: SignupAddUnsubscribe :exec
-- Unsubscribes block Broadcasts and Automations; System mail (password
-- resets, receipts) still reaches the contact.
INSERT INTO suppressions (id, workspace_id, email, reason, source, detail, scope, campaign_id, context, created_at)
VALUES (@id, @workspace_id, @email, 'unsubscribe', @source, @detail, 'marketing', @campaign_id, @context, @at)
ON CONFLICT (workspace_id, email) DO NOTHING;

-- name: SignupLiftUnsubscribe :execrows
DELETE FROM suppressions WHERE workspace_id = @workspace_id AND email = @email AND reason = 'unsubscribe';

-- name: SignupPublicLists :many
-- Lists a public page can show, with their size.
SELECT l.id, l.name, l.description, l.public, l.created_at,
       (SELECT count(*) FROM list_memberships lm WHERE lm.list_id = l.id AND lm.status = 'subscribed')::bigint AS subscribed
FROM lists l
WHERE l.workspace_id = @workspace_id AND l.deleted_at IS NULL
ORDER BY l.created_at, l.id;

-- name: SignupPreviewConfirmed :one
-- The subscriber who confirmed most recently: the app previews the public
-- pages as they see them.
SELECT ce.contact_id::uuid AS contact_id
FROM consent_events ce
JOIN contacts c ON c.id = ce.contact_id
WHERE c.workspace_id = @workspace_id AND ce.action = 'confirmed' AND c.erased_at IS NULL AND c.status = 'active'
ORDER BY ce.occurred_at DESC, ce.id DESC
LIMIT 1;

-- name: SignupPreviewLatest :one
SELECT id FROM contacts
WHERE workspace_id = @workspace_id AND erased_at IS NULL AND status = 'active'
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: SignupContactNames :many
-- Addresses and first names of some contacts (erased ones left out), for
-- lists of signups.
SELECT id, email, first_name FROM contacts
WHERE workspace_id = @workspace_id AND id = ANY (@ids::uuid[]) AND erased_at IS NULL;
