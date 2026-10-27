-- Requests through a proxy ReputeCast doesn't trust (review AN-8).

-- name: ProxySeen :exec
-- Adds requests from an untrusted proxy since the last write.
INSERT INTO untrusted_proxies AS p (peer, header, requests, first_seen, last_seen)
VALUES (@peer, @header, @requests, @seen_at, @seen_at)
ON CONFLICT (peer) DO UPDATE
SET header = EXCLUDED.header, requests = p.requests + EXCLUDED.requests, last_seen = greatest(p.last_seen, EXCLUDED.last_seen);

-- name: ProxyList :many
SELECT peer, header, requests, first_seen, last_seen, notified_at
FROM untrusted_proxies
ORDER BY last_seen DESC
LIMIT 50;

-- name: ProxyDue :many
-- Proxies seen often enough, recently, whose admins weren't told lately.
SELECT peer, header, requests, first_seen, last_seen, notified_at
FROM untrusted_proxies
WHERE requests >= @min_requests AND last_seen >= @seen_since AND (notified_at IS NULL OR notified_at < @notified_before)
ORDER BY peer
FOR UPDATE SKIP LOCKED;

-- name: ProxyNotified :exec
UPDATE untrusted_proxies SET notified_at = @at WHERE peer = @peer;

-- name: ProxyForget :exec
-- A proxy that is trusted now.
DELETE FROM untrusted_proxies WHERE peer = @peer;

-- name: ProxyForgetOld :exec
-- Proxies not seen for a long time.
DELETE FROM untrusted_proxies WHERE last_seen < @seen_before;
