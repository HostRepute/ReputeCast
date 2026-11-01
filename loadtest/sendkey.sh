#!/bin/sh
# Creates a send-only API key for the send.js load script and prints it:
#   TOKEN=$(COOKIE=$(loadtest/session.sh acme-newsletter) loadtest/sendkey.sh acme-newsletter 1000)
# A transactional sender uses a key, and a key's rate is its own (50 per
# second unless set); a person's session shares 50 per second, so a send
# test through a cookie measures that limit, not the server.
# Needs curl, COOKIE (an owner's session) and BASE_URL (default
# http://localhost:8080).
set -eu
slug=${1:-acme-newsletter}
rate=${2:-1000}
base=${BASE_URL:-http://localhost:8080}
curl -fsS -X POST "$base/api/v1/workspaces/$slug/api-keys" \
  -H "Cookie: $COOKIE" -H 'Content-Type: application/json' \
  -d "{\"name\":\"Load test\",\"permissions\":[\"transactional:send\"],\"transactional\":{\"preset\":\"send_only\",\"mode\":\"live\",\"ratePerSecond\":$rate}}" |
  sed -n 's/.*"secret":"\([^"]*\)".*/\1/p'
