-- Notification center, preferences and email delivery. Owned by internal/notify.

-- name: InsertNotification :execrows
-- Stores a notification unless the person turned both in-app and email off
-- for its event type. email_status is 'pending' when they asked for email.
INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, data, in_app, email_status)
SELECT @id, u.id, sqlc.narg(workspace_id), @kind, @severity, @title, @summary, @body, @link, @data,
       coalesce(p.in_app, true),
       CASE WHEN coalesce(p.email, @default_email::boolean) THEN 'pending' END
FROM users u
LEFT JOIN notification_preferences p ON p.user_id = u.id AND p.event_type = @event_type::text
WHERE u.id = @user_id AND u.status = 'active'
  AND (coalesce(p.in_app, true) OR coalesce(p.email, @default_email::boolean));

-- name: ListNotifications :many
-- Unread first, then newest first (the notification center and the bell).
SELECT * FROM notifications
WHERE user_id = @user_id AND in_app
  AND (workspace_id = @workspace_id OR workspace_id IS NULL)
  AND (sqlc.narg(kinds)::text[] IS NULL OR kind = ANY(sqlc.narg(kinds)::text[]))
  AND (NOT @unread_only::boolean OR read_at IS NULL)
ORDER BY (read_at IS NULL) DESC, created_at DESC, id DESC
LIMIT @lim OFFSET @off;

-- name: CountNotifications :one
SELECT count(*) AS total, count(*) FILTER (WHERE read_at IS NULL) AS unread
FROM notifications
WHERE user_id = @user_id AND in_app
  AND (workspace_id = @workspace_id OR workspace_id IS NULL)
  AND (sqlc.narg(kinds)::text[] IS NULL OR kind = ANY(sqlc.narg(kinds)::text[]))
  AND (NOT @unread_only::boolean OR read_at IS NULL);

-- name: CountUnreadNotifications :one
SELECT count(*) FROM notifications
WHERE user_id = @user_id AND in_app AND read_at IS NULL
  AND (workspace_id = @workspace_id OR workspace_id IS NULL);

-- name: MarkNotificationRead :execrows
UPDATE notifications SET read_at = now()
WHERE id = @id AND user_id = @user_id AND read_at IS NULL
  AND (workspace_id = @workspace_id OR workspace_id IS NULL);

-- name: MarkAllNotificationsRead :execrows
UPDATE notifications SET read_at = now()
WHERE user_id = @user_id AND read_at IS NULL AND in_app
  AND (workspace_id = @workspace_id OR workspace_id IS NULL);

-- name: ListNotificationPreferences :many
SELECT * FROM notification_preferences WHERE user_id = @user_id;

-- name: UpsertNotificationPreference :exec
INSERT INTO notification_preferences (user_id, event_type, in_app, email)
VALUES (@user_id, @event_type, @in_app, @email)
ON CONFLICT (user_id, event_type) DO UPDATE SET in_app = excluded.in_app, email = excluded.email;

-- name: GetNotificationSettings :one
SELECT * FROM notification_settings WHERE user_id = @user_id;

-- name: UpsertNotificationSettings :exec
INSERT INTO notification_settings (user_id, quiet_start, quiet_end, digest, updated_at)
VALUES (@user_id, sqlc.narg(quiet_start), sqlc.narg(quiet_end), @digest, now())
ON CONFLICT (user_id) DO UPDATE SET quiet_start = excluded.quiet_start, quiet_end = excluded.quiet_end,
    digest = excluded.digest, updated_at = now();

-- name: SetLastDigestAt :exec
INSERT INTO notification_settings (user_id, last_digest_at) VALUES (@user_id, now())
ON CONFLICT (user_id) DO UPDATE SET last_digest_at = now();

-- name: ListUsersWithPendingEmail :many
SELECT DISTINCT n.user_id FROM notifications n WHERE n.email_status = 'pending' LIMIT 500;

-- name: ListPendingEmailNotifications :many
SELECT n.*, coalesce(w.name, '')::text AS workspace_name, coalesce(w.slug, '')::text AS workspace_slug
FROM notifications n
LEFT JOIN workspaces w ON w.id = n.workspace_id
WHERE n.user_id = @user_id AND n.email_status = 'pending'
ORDER BY n.created_at
LIMIT 200;

-- name: SetNotificationsEmailStatus :exec
UPDATE notifications
SET email_status = @email_status::text,
    emailed_at = CASE WHEN @email_status::text = 'sent' THEN now() ELSE emailed_at END
WHERE id = ANY(@ids::uuid[]);

-- name: BumpNotificationEmailAttempts :exec
UPDATE notifications
SET email_attempts = email_attempts + 1,
    email_status = CASE WHEN email_attempts + 1 >= @max_attempts::int THEN 'failed' ELSE email_status END
WHERE id = ANY(@ids::uuid[]);

-- name: DeleteOldNotifications :execrows
DELETE FROM notifications WHERE created_at < now() - make_interval(days => @days::int);
