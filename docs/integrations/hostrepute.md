# HostRepute integration

HostRepute (hostrepute.com) is the sibling email-reputation service. Connecting
it is optional and free on the ReputeCast side: ReputeCast stays fully usable
without it, with its own built-in checks. When a workspace connects, HostRepute
feeds blocklist status (84 sources, hourly), Google Postmaster Tools, Microsoft
SNDS, DMARC aggregate reports and alerts into Deliverability, Send Readiness and
notifications.

This page documents the contract ReputeCast assumes. The official Go client
(`hostrepute-go`) is private, so `internal/hostrepute` talks to the API through
its own `Client` interface. `HTTPClient` implements that interface against the
endpoints below. When `hostrepute-go` is public, an adapter to it replaces
`HTTPClient` and nothing else changes.

## What crosses the boundary

| Sent to HostRepute | Never sent |
|---|---|
| Sending domains and their organization domains | Contacts or any personal data |
| Dedicated sending IPs | Email content, subjects or templates |
| Self-hosted server hostnames, only when someone clicks "Add to monitoring" | Campaign reports, events or statistics |

Everything else flows from HostRepute to ReputeCast. The list of assets is
built by `hostrepute.Candidates` from `sending_domains` and the connected
providers. Nothing outside that list is ever sent.

## Configuration

```yaml
hostrepute:
  api_url: https://api.hostrepute.com/v1      # REPUTECAST_HOSTREPUTE__API_URL
  auth_url: https://app.hostrepute.com/oauth/authorize
  token_url: https://api.hostrepute.com/oauth/token
  revoke_url: https://api.hostrepute.com/oauth/revoke
  client_id: reputecast
  client_secret: ""                           # public client (PKCE); set if HostRepute issues one
  demo: false                                 # REPUTECAST_HOSTREPUTE__DEMO
```

`demo: true` (development and the sample data) connects to a built-in demo
client instead of the network: the consent step approves at once, and the data
is the Acme sample (team "Acme Inc", 3 of 3 monitored, UCEPROTECT Level 3
listing on 203.0.113.25). Connections made this way store `api_url = "demo:"`.

## Authentication

### OAuth 2 authorization code with PKCE (default)

1. `POST /api/v1/workspaces/{ws}/integrations/hostrepute/oauth` (needs
   `integrations:hostrepute`, people only) creates a single-use state. Its
   SHA-256 is stored in `hostrepute_oauth_states`, together with the PKCE
   verifier encrypted with the instance key and the workspace, the user and
   a 15-minute expiry. The endpoint returns the authorize URL.
2. The browser goes to `auth_url` with `response_type=code`, `client_id`,
   `redirect_uri`, `scope=monitor:read alerts:read assets:write`, `state`,
   `code_challenge` and `code_challenge_method=S256`.
3. HostRepute redirects to the instance's fixed callback,
   `{public_url}/api/v1/integrations/hostrepute/callback?code&state`. Every
   workspace uses the same redirect URI, and the state names the workspace.
4. The callback consumes the state, checks that the signed-in person is the
   one who started the handoff and still holds `integrations:hostrepute`, and
   exchanges the code at `token_url` (with `code_verifier`). It then calls
   `GET /me` and redirects to
   `/w/{slug}/deliverability/connect?result=connected`, or
   `?result=error&reason=declined|wrong_user|forbidden|exchange_failed|unreachable|hostrepute_error|save_failed`.

Access and refresh tokens are sealed with the instance key (AES-GCM, associated
data `hostrepute:{workspace}:token` and `…:refresh`). They are refreshed by
`golang.org/x/oauth2` and never returned by the API.

### API key (alternative)

`POST /api/v1/workspaces/{ws}/integrations/hostrepute/api-key` with
`{"apiKey": "hr_live_…"}`. The key is sent as `Authorization: Bearer <key>` and
validated with `GET /me` before it is saved, encrypted, like an OAuth token. An
API key can't be revoked from ReputeCast. Disconnecting forgets it, and the
person revokes it in HostRepute.

### Revocation

Disconnect (`DELETE …/integrations/hostrepute`) revokes OAuth tokens at
`revoke_url` (RFC 7009, `token` + `token_type_hint=access_token` + `client_id`).
It then deletes the connection and every row that came from HostRepute, and
starts the built-in checks. A revoke that fails is logged and does not block
the disconnect.

## API endpoints (assumed)

Base URL is `api_url` (`https://api.hostrepute.com/v1`). Requests send
`Authorization: Bearer <token>`, `Accept: application/json` and
`User-Agent: ReputeCast/<version>`. Times are RFC 3339 UTC, days are
`YYYY-MM-DD`, and JSON fields are snake_case.

| Method and path | Scope | Used for | Answer |
|---|---|---|---|
| `GET /me` | any | Validate a token; the team, its plan and limit | `Account` |
| `GET /assets` | `monitor:read` | What the team monitors, with scores | `{"assets": [Asset]}` |
| `POST /assets` | `assets:write` | "Add to monitoring" | `Asset` (body `{"asset", "type": "domain\|ip\|hostname", "label"}`) |
| `DELETE /assets/{id}` | `assets:write` | Swap an asset out when the plan is full | 204 |
| `GET /blocklists` | `monitor:read` | Every source for every monitored asset | `Blocklists` |
| `GET /postmaster/{domain}?from&to` | `monitor:read` | Google Postmaster Tools, 30 days | `{"days": [PostmasterDay]}` |
| `GET /snds/{ip}?from&to` | `monitor:read` | Microsoft SNDS, 30 days | `{"days": [SNDSDay]}` |
| `GET /dmarc/{domain}/reports?from&to` | `monitor:read` | DMARC aggregate reports, 7 days | `{"reports": [DMARCReport]}` |
| `GET /alerts?since` | `alerts:read` | New listings, delistings, reputation drops | `{"alerts": [Alert]}` |
| `POST /checks` | `monitor:read` | "Run all checks now" | `Check` (body `{"assets": [...]}`) |
| `GET /checks/{id}` | `monitor:read` | Progress ("61 of 84 sources answered") | `Check` |
| `POST /delistings` | `assets:write` | "Request delisting" | `Delisting` (body `{"asset", "source"}`) |

The JSON shapes are the Go types in `internal/hostrepute/client.go`. The main
fields are below.

- `Account`: `id`, `email`, `scopes`, `sources` (84), `check_interval_minutes`
  (60), `dmarc_rua`, and `team` with `id`, `name`, `plan`, `monitor_limit`
  (0 means unlimited) and `monitored`.
- `Asset`: `id`, `asset`, `type`, `status` (`synced`, `pending` or `error`),
  `score` (0–100), `verdict`, `listings` and `last_checked_at`.
- `Blocklists`: `sources`, `checked_at`, and `results[]`. Each result has
  `asset`, `asset_type`, `source`, `impact` (`high`, `medium` or `low`),
  `rank`, `listed`, `listed_since`, `delist_url`, `detail` and `delisting`.
  `delisting` holds `status` (`requested`, `pending`, `delisted` or `denied`),
  `requested_at` and `url`.
- `PostmasterDay`: `day`, `spam_rate`, `domain_reputation` (`HIGH`, `MEDIUM`,
  `LOW` or `BAD`), `ip_reputation`, `spf_rate`, `dkim_rate`, `dmarc_rate`,
  `tls_rate`, `delivery_errors` and `fbl_flagged`.
- `SNDSDay`: `day`, `filter_result` (`green`, `yellow` or `red`),
  `complaint_rate`, `trap_hits`, `messages` and `jmrp_reports`.
- `DMARCReport`: `domain`, `org_name`, `report_id`, `date_begin`, `date_end`,
  `policy` and `records[]`. Each record has `source_ip`, `ptr`, `count`,
  `disposition`, `dkim`, `spf`, `dkim_result`, `spf_result`, `header_from`,
  `dkim_domain` and `spf_domain`.
- `Alert`: `id`, `created_at`, `kind` (`listed`, `delisted`,
  `reputation_drop`, `dmarc_failure` or `auth_change`), `severity`, `asset`,
  `source`, `title`, `detail` and `delist_url`.
- `Check`: `id`, `status` (`running`, `done` or `failed`), `total`,
  `answered`, `started_at` and `finished_at`.

### Errors

Errors are JSON problems with at least `code` and `detail`.

| Answer | Meaning in ReputeCast |
|---|---|
| 401 | Token revoked or expired. The connection becomes `revoked`, people with `integrations:hostrepute` are notified, the built-in checks take over, and Send Readiness shows the reputation check as "unknown", never as a Block. |
| 403 | A scope is missing. The call fails with that explanation. |
| 404 | The asset, check or report doesn't exist. |
| 409 `monitor_limit_reached` with `limit`, `used` and `team` | The team's plan is full. ReputeCast says so, links to the plan, and offers to swap an asset. This is HostRepute's limit; nothing in ReputeCast is gated. |
| 429, 5xx, timeouts, network errors | Retried 3 times (1 s, then 4 s, honouring `Retry-After` up to 10 s). After that the call fails with "Couldn't reach api.hostrepute.com (timeout after 30 s, 3 attempts)". The cached data stays visible and marked stale. |

## Sync

- `hostrepute.ScheduleWorker` runs hourly, matching HostRepute's check
  interval. It queues `hostrepute.SyncArgs` for every connected workspace,
  unique per workspace while one is pending.
- `hostrepute.SyncWorker` fetches the account, assets, blocklists, 30 days of
  Postmaster and SNDS data, 7 days of DMARC reports and the alerts since the
  last sync. It stores them in `blocklist_results`, `postmaster_daily`,
  `snds_daily`, `dmarc_reports` and `incidents` (all with
  `via = 'hostrepute'`) in one transaction. Alerts newer than 24 hours become
  notifications (`notify.KindHostReputeAlert`) for members who hold
  `deliverability:view`. The first sync after connecting doesn't notify.
- When HostRepute can't be reached, the error and the attempt time are kept
  on the connection (`last_sync_error`). Deliverability shows the
  "HostRepute unreachable, cached results shown" state, and the built-in
  8-source DNSBL check runs as a fallback on the last attempt.
- "Run all checks now" starts `POST /checks` and polls `GET /checks/{id}`,
  reporting progress through `deliverability_check_runs`.

Every state change (connect, reconnect, disconnect, add or swap an asset,
token rejected, delisting request) writes an audit entry in the same
transaction.

## Built-in checks (without HostRepute)

`internal/dnscheck` runs SPF, DKIM, DMARC, BIMI, MAIL FROM alignment,
forward-confirmed reverse DNS and 8 public DNSBLs: Spamhaus ZEN, Spamhaus DBL,
SURBL, URIBL, Barracuda BRBL, SpamCop, Mailspike and PSBL. They run daily, and
on demand from "Run checks now". These checks never call HostRepute.
