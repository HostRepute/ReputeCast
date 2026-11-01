#!/bin/sh
# Prints a session cookie for a workspace's owner, for the k6 scripts:
#   COOKIE=$(loadtest/session.sh load-test)
# Development and load test instances only: it writes a session row
# directly. Needs psql and REPUTECAST_DATABASE__URL.
set -eu
slug=${1:-load-test}
token=$(od -An -tx1 -N24 /dev/urandom | tr -d ' \n')
psql "$REPUTECAST_DATABASE__URL" -qAt -v ON_ERROR_STOP=1 -v token="$token" -v slug="$slug" <<'SQL' >/dev/null
INSERT INTO sessions (id, user_id, token_hash, auth_method, mfa_verified, created_at, last_seen_at, expires_at)
SELECT gen_random_uuid(), m.user_id, sha256(convert_to(:'token', 'UTF8')), 'password', true, now(), now(), now() + interval '1 day'
FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN workspaces w ON w.id = m.workspace_id
WHERE w.slug = :'slug' AND r.key = 'owner' LIMIT 1;
SQL
echo "rc_session=$token"
