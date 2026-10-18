-- SMS (internal/texting): providers, senders and their registrations,
-- routes, prices, the SMS rules, test numbers, the text message terms,
-- consent per channel, the SMS suppression list, SMS campaigns, the message
-- log, inbound texts, short links and raw provider events.

-- name: SmsProviders :many
-- The workspace's connected SMS providers, oldest first.
SELECT p.*, coalesce(u.name, '')::text AS connected_by_name
FROM sms_providers p
LEFT JOIN users u ON u.id = p.connected_by
WHERE p.workspace_id = @workspace_id AND p.removed_at IS NULL
ORDER BY p.connected_at, p.id;

-- name: SmsProvider :one
SELECT * FROM sms_providers WHERE id = @id AND workspace_id = @workspace_id AND removed_at IS NULL;

-- name: SmsProviderAny :one
-- A provider by id, removed or not (old messages still name it).
SELECT * FROM sms_providers WHERE id = @id;

-- name: SmsProviderByToken :one
-- The provider a webhook request is for.
SELECT * FROM sms_providers WHERE webhook_token = @webhook_token AND adapter = @adapter AND removed_at IS NULL;

-- name: SmsProvidersAll :many
-- Every connected SMS provider of the instance, for the health job.
SELECT p.* FROM sms_providers p
JOIN workspaces w ON w.id = p.workspace_id AND w.deleted_at IS NULL
WHERE p.removed_at IS NULL
ORDER BY p.id;

-- name: SmsProviderInsert :exec
INSERT INTO sms_providers (id, workspace_id, adapter, name, account_label, account_hint, config, credentials_enc, webhook_token, webhook_secret_enc,
    webhooks, status, status_since, demo, connected_by, connected_at, last_checked_at)
VALUES (@id, @workspace_id, @adapter, @name, @account_label, @account_hint, @config, @credentials_enc, @webhook_token, @webhook_secret_enc,
    @webhooks, 'healthy', @connected_at, @demo, sqlc.narg(connected_by), @connected_at, @connected_at);

-- name: SmsProviderSetCredentials :exec
UPDATE sms_providers SET credentials_enc = @credentials_enc, config = @config, account_label = @account_label, account_hint = @account_hint,
    status = 'healthy', status_detail = '', status_since = @at, health = '{}', last_checked_at = @at
WHERE id = @id;

-- name: SmsProviderSetWebhooks :exec
UPDATE sms_providers SET webhook_secret_enc = coalesce(sqlc.narg(webhook_secret_enc), webhook_secret_enc), config = @config, webhooks = @webhooks
WHERE id = @id;

-- name: SmsProviderSetHealth :exec
UPDATE sms_providers SET status = @status, status_detail = @status_detail,
    status_since = CASE WHEN status = @status THEN status_since ELSE @at::timestamptz END,
    health = @health, last_checked_at = @at
WHERE id = @id;

-- name: SmsProviderTouchReceipt :exec
UPDATE sms_providers SET last_receipt_at = greatest(coalesce(last_receipt_at, @at::timestamptz), @at::timestamptz) WHERE id = @id;

-- name: SmsProviderTouchInbound :exec
UPDATE sms_providers SET last_inbound_at = greatest(coalesce(last_inbound_at, @at::timestamptz), @at::timestamptz) WHERE id = @id;

-- name: SmsProviderRegistrationsRead :exec
UPDATE sms_providers SET registrations_read_at = @at WHERE id = @id;

-- name: SmsProviderRemove :exec
UPDATE sms_providers SET removed_at = @at WHERE id = @id;

-- name: SmsSetupInsert :exec
INSERT INTO sms_setups (id, workspace_id, adapter, step, status, state, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @adapter, 'connection', 'open', '{}', sqlc.narg(created_by), @at, @at);

-- name: SmsSetup :one
SELECT * FROM sms_setups WHERE id = @id AND workspace_id = @workspace_id;

-- name: SmsSetupUpdate :exec
UPDATE sms_setups SET step = @step, status = @status, state = @state, provider_id = sqlc.narg(provider_id), updated_at = @at WHERE id = @id;

-- name: SmsSenders :many
-- The workspace's senders with their provider.
SELECT s.*, p.name AS provider_name, p.adapter AS provider_adapter
FROM sms_senders s
JOIN sms_providers p ON p.id = s.provider_id AND p.removed_at IS NULL
WHERE s.workspace_id = @workspace_id AND s.removed_at IS NULL
ORDER BY p.connected_at, s.added_at, s.value;

-- name: SmsSender :one
SELECT s.*, p.name AS provider_name, p.adapter AS provider_adapter
FROM sms_senders s
JOIN sms_providers p ON p.id = s.provider_id
WHERE s.id = @id AND s.workspace_id = @workspace_id AND s.removed_at IS NULL;

-- name: SmsSenderInsert :exec
INSERT INTO sms_senders (id, workspace_id, provider_id, kind, value, countries, throughput, provider_ref, added_by, added_at)
VALUES (@id, @workspace_id, @provider_id, @kind, @value, @countries, @throughput, @provider_ref, sqlc.narg(added_by), @added_at);

-- name: SmsSenderUpdate :exec
UPDATE sms_senders SET countries = @countries, throughput = @throughput WHERE id = @id;

-- name: SmsSenderRemove :exec
UPDATE sms_senders SET removed_at = @at WHERE id = @id;

-- name: SmsRegistrations :many
SELECT r.*, coalesce(u.name, '')::text AS set_by_name
FROM sms_registrations r
LEFT JOIN users u ON u.id = r.set_by
WHERE r.workspace_id = @workspace_id
ORDER BY r.sender_id, r.kind, r.country;

-- name: SmsRegistrationUpsert :exec
INSERT INTO sms_registrations (id, workspace_id, sender_id, kind, country, name, status, reason, status_on, source, details, set_by, read_at, updated_at)
VALUES (@id, @workspace_id, @sender_id, @kind, @country, @name, @status, @reason, sqlc.narg(status_on), @source, @details, sqlc.narg(set_by), sqlc.narg(read_at), @updated_at)
ON CONFLICT (sender_id, kind, country) DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, reason = EXCLUDED.reason,
    status_on = EXCLUDED.status_on, source = EXCLUDED.source,
    details = CASE WHEN EXCLUDED.details = '{}'::jsonb THEN sms_registrations.details ELSE EXCLUDED.details END,
    set_by = EXCLUDED.set_by, read_at = coalesce(EXCLUDED.read_at, sms_registrations.read_at), updated_at = EXCLUDED.updated_at;

-- name: SmsRoutes :many
SELECT * FROM sms_routes WHERE workspace_id = @workspace_id ORDER BY position, created_at;

-- name: SmsRoutesDelete :exec
DELETE FROM sms_routes WHERE workspace_id = @workspace_id;

-- name: SmsRouteInsert :exec
INSERT INTO sms_routes (id, workspace_id, position, countries, sender_id, fallback_sender_id, created_at, updated_at)
VALUES (@id, @workspace_id, @position, @countries, @sender_id, sqlc.narg(fallback_sender_id), @at, @at);

-- name: SmsPrices :many
SELECT * FROM sms_prices WHERE workspace_id = @workspace_id ORDER BY provider_id, country;

-- name: SmsPriceUpsert :exec
INSERT INTO sms_prices (workspace_id, provider_id, country, price, source, as_of)
VALUES (@workspace_id, @provider_id, @country, @price, @source, @as_of)
ON CONFLICT (provider_id, country) DO UPDATE SET price = EXCLUDED.price, source = EXCLUDED.source, as_of = EXCLUDED.as_of
WHERE sms_prices.source = 'api' OR EXCLUDED.source = 'manual';

-- name: SmsSettings :one
SELECT * FROM sms_settings WHERE workspace_id = @workspace_id;

-- name: SmsSettingsEnsure :exec
-- The rules a workspace starts with: the brief's defaults.
INSERT INTO sms_settings (workspace_id, brand_prefix, help_reply, opt_out_reply, opt_in_reply, confirmation_text, updated_at)
VALUES (@workspace_id, @brand_prefix, @help_reply, @opt_out_reply, @opt_in_reply, @confirmation_text, @at)
ON CONFLICT (workspace_id) DO NOTHING;

-- name: SmsSettingsSave :exec
UPDATE sms_settings SET brand_prefix = @brand_prefix, opt_out_text = @opt_out_text, opt_out_in_steps = @opt_out_in_steps,
    confirmation_on = @confirmation_on, confirmation_text = @confirmation_text, require_yes = @require_yes,
    quiet_from = @quiet_from, quiet_to = @quiet_to, push_same_window = @push_same_window, whatsapp_same_window = @whatsapp_same_window,
    opt_out_keywords = @opt_out_keywords, opt_in_keywords = @opt_in_keywords, help_keywords = @help_keywords,
    help_reply = @help_reply, opt_out_reply = @opt_out_reply, opt_in_reply = @opt_in_reply, keywords_by = @keywords_by,
    budget = @budget, budget_alert_pct = @budget_alert_pct,
    budget_set_by = CASE WHEN budget <> @budget THEN sqlc.narg(updated_by) ELSE budget_set_by END,
    budget_set_at = CASE WHEN budget <> @budget THEN @at::timestamptz ELSE budget_set_at END,
    pause_failed_pct = @pause_failed_pct, pause_filtered_pct = @pause_filtered_pct, pause_opt_out_pct = @pause_opt_out_pct,
    other_countries = @other_countries, other_route_id = sqlc.narg(other_route_id), retention_days = @retention_days,
    updated_by = sqlc.narg(updated_by), updated_at = @at
WHERE workspace_id = @workspace_id;

-- name: SmsSettingsShortDomain :exec
UPDATE sms_settings SET short_domain = @short_domain, short_domain_status = @short_domain_status, short_domain_target = @short_domain_target,
    short_domain_checked_at = sqlc.narg(checked_at), short_domain_verified_at = sqlc.narg(verified_at), short_domain_detail = @short_domain_detail
WHERE workspace_id = @workspace_id;

-- name: SmsShortDomainWorkspace :one
-- The workspace whose verified short-link domain a request came to.
SELECT workspace_id FROM sms_settings WHERE lower(short_domain) = lower(@host) AND short_domain_status = 'verified';

-- name: SmsTestNumbers :many
SELECT t.*, coalesce(u.name, '')::text AS user_name, coalesce(r.name, '')::text AS role_name, coalesce(u.avatar_color, '')::text AS user_color
FROM sms_test_numbers t
LEFT JOIN users u ON u.id = t.user_id
LEFT JOIN workspace_members m ON m.user_id = t.user_id AND m.workspace_id = t.workspace_id
LEFT JOIN roles r ON r.id = m.role_id
WHERE t.workspace_id = @workspace_id
ORDER BY t.added_at;

-- name: SmsTestNumberInsert :exec
INSERT INTO sms_test_numbers (id, workspace_id, user_id, phone, added_by, added_at)
VALUES (@id, @workspace_id, sqlc.narg(user_id), @phone, sqlc.narg(added_by), @at);

-- name: SmsTestNumberDelete :exec
DELETE FROM sms_test_numbers WHERE id = @id AND workspace_id = @workspace_id;

-- name: SmsTermsLatest :one
SELECT * FROM sms_terms_versions WHERE workspace_id = @workspace_id ORDER BY version DESC LIMIT 1;

-- name: SmsTermsInsert :exec
INSERT INTO sms_terms_versions (workspace_id, version, content, published_by, published_at)
VALUES (@workspace_id, @version, @content, sqlc.narg(published_by), @at);

-- name: ChanSuppressions :many
-- A page of the SMS suppression list, newest first.
SELECT s.*, coalesce(p.name, '')::text AS provider_name, coalesce(c.name, '')::text AS campaign_name, coalesce(a.name, '')::text AS automation_name
FROM channel_suppressions s
LEFT JOIN sms_providers p ON p.id = s.source_provider_id
LEFT JOIN campaigns c ON c.id = s.campaign_id
LEFT JOIN automations a ON a.id = s.automation_id
WHERE s.workspace_id = @workspace_id AND s.channel = @channel AND s.removed_at IS NULL
  AND (@reason::text = '' OR s.reason = @reason)
  AND (sqlc.narg(provider_id)::uuid IS NULL OR s.source_provider_id = sqlc.narg(provider_id))
  AND (@q::text = '' OR s.phone LIKE '%' || @q || '%')
ORDER BY s.created_at DESC, s.id
LIMIT @lim OFFSET @off;

-- name: ChanSuppressionCount :one
SELECT count(*)::bigint FROM channel_suppressions s
WHERE s.workspace_id = @workspace_id AND s.channel = @channel AND s.removed_at IS NULL
  AND (@reason::text = '' OR s.reason = @reason)
  AND (sqlc.narg(provider_id)::uuid IS NULL OR s.source_provider_id = sqlc.narg(provider_id))
  AND (@q::text = '' OR s.phone LIKE '%' || @q || '%');

-- name: ChanSuppressionCounts :many
SELECT reason, count(*)::bigint AS n FROM channel_suppressions
WHERE workspace_id = @workspace_id AND channel = @channel AND removed_at IS NULL
GROUP BY reason;

-- name: ChanSuppression :one
SELECT * FROM channel_suppressions WHERE id = @id AND workspace_id = @workspace_id AND removed_at IS NULL;

-- name: ChanSuppressionByPhone :one
SELECT * FROM channel_suppressions WHERE workspace_id = @workspace_id AND channel = @channel AND phone = @phone AND removed_at IS NULL;

-- name: ChanSuppressedPhones :many
-- Which of these numbers are suppressed.
SELECT phone, reason FROM channel_suppressions
WHERE workspace_id = @workspace_id AND channel = @channel AND removed_at IS NULL AND phone = ANY(@phones::text[]);

-- name: ChanSuppressionInsert :exec
INSERT INTO channel_suppressions (id, workspace_id, channel, phone, reason, keyword, original_code, source, source_provider_id, campaign_id, automation_id,
    context, created_by, created_at)
VALUES (@id, @workspace_id, @channel, @phone, @reason, @keyword, @original_code, @source, sqlc.narg(source_provider_id), sqlc.narg(campaign_id),
    sqlc.narg(automation_id), @context, sqlc.narg(created_by), @created_at)
ON CONFLICT (workspace_id, channel, phone) WHERE removed_at IS NULL DO NOTHING;

-- name: ChanSuppressionRemove :exec
UPDATE channel_suppressions SET removed_at = @at, removed_by = sqlc.narg(removed_by), removed_reason = @removed_reason WHERE id = @id;

-- name: ChanConsents :many
-- A contact's consent per list on the text channels.
SELECT * FROM channel_consents WHERE contact_id = @contact_id ORDER BY list_id, channel;

-- name: ChanConsentUpsert :exec
INSERT INTO channel_consents (workspace_id, list_id, contact_id, channel, status, source, keyword, stopped_at, changed_at)
VALUES (@workspace_id, @list_id, @contact_id, @channel, @status, @source, @keyword, sqlc.narg(stopped_at), @changed_at)
ON CONFLICT (list_id, contact_id, channel) DO UPDATE SET status = EXCLUDED.status, source = EXCLUDED.source, keyword = EXCLUDED.keyword,
    stopped_at = EXCLUDED.stopped_at, changed_at = EXCLUDED.changed_at;

-- name: ChanConsentEventInsert :exec
INSERT INTO channel_consent_events (id, workspace_id, contact_id, list_id, channel, action, phone, source, page, consent_text, consent_version,
    keyword, ip, user_agent, actor_id, import_id, note, occurred_at)
VALUES (@id, @workspace_id, @contact_id, sqlc.narg(list_id), @channel, @action, @phone, @source, @page, @consent_text, @consent_version,
    @keyword, sqlc.narg(ip), @user_agent, sqlc.narg(actor_id), sqlc.narg(import_id), @note, @occurred_at);

-- name: ChanConsentEvents :many
SELECT e.*, coalesce(u.name, '')::text AS actor_name
FROM channel_consent_events e
LEFT JOIN users u ON u.id = e.actor_id
WHERE e.contact_id = @contact_id
ORDER BY e.occurred_at DESC, e.id DESC;

-- name: ChanContactsByPhone :many
-- Every contact with this number: a STOP applies to all of them.
SELECT id, email, first_name, last_name, timezone, phone, phone_country FROM contacts
WHERE workspace_id = @workspace_id AND phone = @phone AND erased_at IS NULL
ORDER BY created_at;

-- name: ChanSetPhone :exec
UPDATE contacts SET phone = @phone, phone_country = @phone_country, phone_line_type = @phone_line_type, phone_line_source = @phone_line_source,
    phone_set_at = @at, updated_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: ChanSetLineType :exec
-- What a provider learned about a number, for every contact who has it.
UPDATE contacts SET phone_line_type = @phone_line_type, phone_line_source = @phone_line_source
WHERE workspace_id = @workspace_id AND phone = @phone;

-- name: SmsCampaign :one
SELECT c.id, c.workspace_id, c.name, c.status, c.audience, c.version, c.content_hash, c.created_by, c.updated_by, c.created_at, c.updated_at,
    c.autosaved_at, c.scheduled_at, c.started_at, c.completed_at, c.started_by, c.channel,
    s.body, s.schedule_mode, s.send_at, s.local_date, s.local_minute, s.pending_policy, s.thresholds,
    coalesce(cu.name, '')::text AS created_by_name, coalesce(uu.name, '')::text AS updated_by_name
FROM campaigns c
JOIN sms_campaigns s ON s.campaign_id = c.id
LEFT JOIN users cu ON cu.id = c.created_by
LEFT JOIN users uu ON uu.id = c.updated_by
WHERE c.id = @id AND c.workspace_id = @workspace_id AND c.deleted_at IS NULL;

-- name: SmsCampaignInsert :exec
INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, editor_mode, mjml, html, plain_text, version, channel,
    content_hash, created_by, updated_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, 'draft', '', '', @audience, 'html', '', '', '', 1, 'sms', @content_hash, sqlc.narg(by), sqlc.narg(by), @at, @at);

-- name: SmsCampaignDetailInsert :exec
INSERT INTO sms_campaigns (campaign_id, workspace_id, body, schedule_mode, local_date, local_minute, pending_policy)
VALUES (@campaign_id, @workspace_id, @body, @schedule_mode, sqlc.narg(local_date), sqlc.narg(local_minute), 'skip');

-- name: SmsCampaignVersionInsert :exec
INSERT INTO sms_campaign_versions (campaign_id, version, body, saved_by, saved_at)
VALUES (@campaign_id, @version, @body, sqlc.narg(saved_by), @saved_at)
ON CONFLICT (campaign_id, version) DO UPDATE SET body = EXCLUDED.body, saved_by = EXCLUDED.saved_by, saved_at = EXCLUDED.saved_at;

-- name: SmsCampaignVersions :many
SELECT v.*, coalesce(u.name, '')::text AS saved_by_name
FROM sms_campaign_versions v LEFT JOIN users u ON u.id = v.saved_by
WHERE v.campaign_id = @campaign_id ORDER BY v.version DESC;

-- name: SmsCampaignSave :exec
-- An autosave: the text, name and audience, with the next version number.
UPDATE campaigns SET name = @name, audience = @audience, content_hash = @content_hash, version = @version, updated_by = sqlc.narg(by),
    updated_at = @at, autosaved_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: SmsCampaignBodySave :exec
UPDATE sms_campaigns SET body = @body WHERE campaign_id = @campaign_id;

-- name: SmsCampaignScheduleSave :exec
UPDATE sms_campaigns SET schedule_mode = @schedule_mode, send_at = sqlc.narg(send_at), local_date = sqlc.narg(local_date),
    local_minute = sqlc.narg(local_minute), pending_policy = @pending_policy, thresholds = @thresholds
WHERE campaign_id = @campaign_id;

-- name: SmsCampaignSetStatus :exec
UPDATE campaigns SET status = @status, scheduled_at = sqlc.narg(scheduled_at), started_at = coalesce(sqlc.narg(started_at), started_at),
    started_by = coalesce(sqlc.narg(started_by), started_by), completed_at = sqlc.narg(completed_at), updated_at = @at
WHERE id = @id;

-- name: SmsSend :one
SELECT * FROM sms_sends WHERE campaign_id = @campaign_id;

-- name: SmsSendUpsert :exec
INSERT INTO sms_sends (campaign_id, workspace_id, phase, recipients, skipped, left_out, started_by, approved_by, scheduled_at, thresholds)
VALUES (@campaign_id, @workspace_id, @phase, @recipients, @skipped, @left_out, sqlc.narg(started_by), sqlc.narg(approved_by), @scheduled_at, @thresholds)
ON CONFLICT (campaign_id) DO UPDATE SET phase = EXCLUDED.phase, recipients = EXCLUDED.recipients, skipped = EXCLUDED.skipped,
    left_out = EXCLUDED.left_out, started_by = EXCLUDED.started_by, approved_by = EXCLUDED.approved_by, scheduled_at = EXCLUDED.scheduled_at,
    thresholds = EXCLUDED.thresholds, paused_kind = '', paused_at = NULL, pause = '{}', completed_at = NULL;

-- name: SmsSendSetPhase :exec
UPDATE sms_sends SET phase = @phase, paused_kind = @paused_kind, paused_at = sqlc.narg(paused_at), pause = @pause,
    completed_at = sqlc.narg(completed_at)
WHERE campaign_id = @campaign_id;

-- name: SmsSendTouch :exec
UPDATE sms_sends SET first_send_at = coalesce(first_send_at, @at::timestamptz), last_send_at = @at WHERE campaign_id = @campaign_id;

-- name: SmsSendsActive :many
-- Sends that are scheduled, sending or held by an outage.
SELECT * FROM sms_sends WHERE phase IN ('scheduled', 'sending') OR (phase = 'paused' AND paused_kind = 'outage');

-- name: ChanMessageInsert :exec
INSERT INTO channel_messages (id, workspace_id, channel, kind, campaign_id, automation_id, automation_step, run_id, contact_id, list_id, recipient, country,
    time_zone, zone_source, provider_id, sender_id, sender, body, units, encoding, price, cost, status, reason, send_after, expires_at, created_at, updated_at)
VALUES (@id, @workspace_id, @channel, @kind, sqlc.narg(campaign_id), sqlc.narg(automation_id), @automation_step, sqlc.narg(run_id), sqlc.narg(contact_id),
    sqlc.narg(list_id), @recipient, @country, @time_zone, @zone_source, sqlc.narg(provider_id), sqlc.narg(sender_id), @sender, sqlc.narg(body), @units,
    @encoding, sqlc.narg(price), @cost, @status, @reason, sqlc.narg(send_after), sqlc.narg(expires_at), @created_at, @created_at);

-- name: ChanMessage :one
SELECT * FROM channel_messages WHERE id = @id;

-- name: ChanMessageByProviderID :one
SELECT * FROM channel_messages WHERE provider_id = @provider_id AND provider_message_id = @provider_message_id;

-- name: ChanMessageClaim :one
-- Claims a due text for sending: it leaves the queue, so it's never sent twice.
UPDATE channel_messages SET status = 'sending', attempts = attempts + 1, updated_at = @at
WHERE id = @id AND status = 'queued'
RETURNING *;

-- name: ChanMessageSent :exec
UPDATE channel_messages SET status = CASE WHEN status IN ('delivered','undelivered','failed') THEN status ELSE 'sent' END,
    provider_message_id = @provider_message_id, sent_at = @at, units = CASE WHEN @units::int > 0 THEN @units ELSE units END,
    cost = CASE WHEN @units::int > 0 AND price IS NOT NULL THEN price * @units ELSE cost END, updated_at = @at
WHERE id = @id;

-- name: ChanMessageRequeue :exec
-- A temporary failure: back in the queue after a pause.
UPDATE channel_messages SET status = 'queued', send_after = @send_after, reason = @reason, provider_code = @provider_code, updated_at = @at
WHERE id = @id;

-- name: ChanMessageFail :exec
UPDATE channel_messages SET status = @status, reason = @reason, provider_code = @provider_code, failed_at = @at, cost = @cost, updated_at = @at
WHERE id = @id;

-- name: ChanMessageReceipt :exec
-- A delivery receipt: a final status wins over an earlier one.
UPDATE channel_messages SET status = @status, reason = CASE WHEN @reason::text = '' THEN reason ELSE @reason END,
    provider_code = CASE WHEN @provider_code::text = '' THEN provider_code ELSE @provider_code END,
    delivered_at = CASE WHEN @status = 'delivered' THEN coalesce(delivered_at, @at::timestamptz) ELSE delivered_at END,
    failed_at = CASE WHEN @status IN ('undelivered','failed') THEN coalesce(failed_at, @at::timestamptz) ELSE failed_at END,
    updated_at = @at
WHERE id = @id AND status NOT IN ('delivered', 'undelivered', 'failed', 'cancelled', 'skipped');

-- name: ChanMessageEventInsert :exec
INSERT INTO channel_message_events (id, workspace_id, message_id, kind, provider_code, detail, raw_event_id, machine, occurred_at)
VALUES (@id, @workspace_id, @message_id, @kind, @provider_code, @detail, sqlc.narg(raw_event_id), @machine, @occurred_at);

-- name: ChanMessageEvents :many
SELECT * FROM channel_message_events WHERE message_id = @message_id ORDER BY occurred_at, id;

-- name: ChanMessagesDue :many
-- Texts whose time has come, oldest first, for one sender.
SELECT * FROM channel_messages
WHERE status = 'queued' AND sender_id = @sender_id AND (send_after IS NULL OR send_after <= @now)
ORDER BY send_after NULLS FIRST, created_at
LIMIT @lim;

-- name: ChanSendersDue :many
-- The senders with texts due now.
SELECT DISTINCT sender_id FROM channel_messages
WHERE status = 'queued' AND sender_id IS NOT NULL AND (send_after IS NULL OR send_after <= @now);

-- name: ChanLastTextTo :one
-- The last text a number got from the workspace, for its replies.
SELECT * FROM channel_messages
WHERE workspace_id = @workspace_id AND recipient = @recipient AND channel = 'sms' AND status NOT IN ('queued','held','skipped','cancelled')
ORDER BY coalesce(sent_at, created_at) DESC LIMIT 1;

-- name: SmsInboundInsert :exec
INSERT INTO sms_inbound (id, workspace_id, provider_id, phone, sender, body, kind, keyword, message_id, campaign_id, automation_id, contact_ids, effect,
    reply_message_id, raw_event_id, received_at)
VALUES (@id, @workspace_id, sqlc.narg(provider_id), @phone, @sender, @body, @kind, @keyword, sqlc.narg(message_id), sqlc.narg(campaign_id),
    sqlc.narg(automation_id), @contact_ids, @effect, sqlc.narg(reply_message_id), sqlc.narg(raw_event_id), @received_at);

-- name: SmsLinkInsert :exec
INSERT INTO sms_links (code, workspace_id, message_id, campaign_id, automation_id, contact_id, original, target, created_at, expires_at)
VALUES (@code, @workspace_id, sqlc.narg(message_id), sqlc.narg(campaign_id), sqlc.narg(automation_id), sqlc.narg(contact_id), @original, @target,
    @created_at, @expires_at);

-- name: SmsLink :one
SELECT * FROM sms_links WHERE code = @code;

-- name: SmsLinkClickInsert :exec
INSERT INTO sms_link_clicks (id, workspace_id, code, campaign_id, message_id, contact_id, machine, user_agent, occurred_at)
VALUES (@id, @workspace_id, @code, sqlc.narg(campaign_id), sqlc.narg(message_id), sqlc.narg(contact_id), @machine, @user_agent, @occurred_at);

-- name: SmsRawInsert :exec
INSERT INTO sms_raw_events (id, workspace_id, provider_id, kind, received_at, content_type, headers, body, signature_ok, summary)
VALUES (@id, @workspace_id, @provider_id, @kind, @received_at, @content_type, @headers, @body, @signature_ok, @summary);

-- name: SmsRaw :one
SELECT * FROM sms_raw_events WHERE id = @id;

-- name: SmsRawNormalized :exec
UPDATE sms_raw_events SET normalized_at = @at, normalize_error = @normalize_error, events = @events, summary = CASE WHEN @summary::text = '' THEN summary ELSE @summary END
WHERE id = @id;

-- name: SmsRawPending :many
SELECT id FROM sms_raw_events WHERE normalized_at IS NULL AND received_at < @before ORDER BY received_at LIMIT 500;

-- name: SmsBudgetAlertInsert :one
-- Notes a budget alert once per month and level; false when it was sent already.
WITH ins AS (
    INSERT INTO sms_budget_alerts (workspace_id, month, level, at) VALUES (@workspace_id, @month, @level, @at)
    ON CONFLICT DO NOTHING RETURNING 1
)
SELECT count(*) > 0 FROM ins;

-- name: SmsOptOutSync :many
SELECT * FROM sms_opt_out_sync WHERE workspace_id = @workspace_id;

-- name: SmsOptOutSyncUpsert :exec
INSERT INTO sms_opt_out_sync (provider_id, workspace_id, mode, pulled, pushed, status, detail, last_sync_at)
VALUES (@provider_id, @workspace_id, @mode, @pulled, @pushed, @status, @detail, @last_sync_at)
ON CONFLICT (provider_id) DO UPDATE SET mode = EXCLUDED.mode, pulled = EXCLUDED.pulled, pushed = EXCLUDED.pushed, status = EXCLUDED.status,
    detail = EXCLUDED.detail, last_sync_at = EXCLUDED.last_sync_at;
