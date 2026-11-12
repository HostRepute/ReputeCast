-- Pausing a subscription (M65, G-39): Stripe's pause_collection, mirrored
-- on paid_subscriptions (paused_at, pause_resumes_at, last_paused_at).

-- name: EarnSettingsSetPauseMonths :exec
UPDATE earn_settings SET pause_months = @pause_months, updated_at = @at, updated_by = sqlc.narg(updated_by)
WHERE workspace_id = @workspace_id;

-- name: EarnSubscriptionSetPause :exec
-- A pause or resume asked here, applied at once; Stripe's event that
-- follows finds it in place.
UPDATE paid_subscriptions SET paused_at = sqlc.narg(paused_at), pause_resumes_at = sqlc.narg(pause_resumes_at),
    last_paused_at = coalesce(sqlc.narg(paused_at), last_paused_at), updated_at = @at
WHERE id = @id;

-- name: EarnResumingSoon :many
-- Paused subscriptions Stripe resumes within the window whose "resuming
-- soon" email wasn't sent for this resume date.
SELECT s.* FROM paid_subscriptions s
WHERE s.paused_at IS NOT NULL AND s.ended_at IS NULL AND s.kind = 'stripe'
  AND s.pause_resumes_at IS NOT NULL AND s.pause_resumes_at > @now::timestamptz AND s.pause_resumes_at <= @until::timestamptz
  AND s.resume_notified_at IS DISTINCT FROM s.pause_resumes_at
ORDER BY s.pause_resumes_at
LIMIT 500;

-- name: EarnResumeNotified :one
-- Claims the "resuming soon" email of a resume date: once.
UPDATE paid_subscriptions SET resume_notified_at = pause_resumes_at
WHERE id = @id AND pause_resumes_at = @resumes_at AND resume_notified_at IS DISTINCT FROM pause_resumes_at
RETURNING id;

-- name: EarnCancelVisitPaused :exec
UPDATE earn_cancel_visits SET decision = 'paused', decided_at = @at::timestamptz WHERE id = @id AND decision = '';
