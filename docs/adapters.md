# Sending adapters

ReputeCast sends through pluggable sending adapters (spec 5.2). Built-in
adapters are Go packages in `internal/providers/<name>` that ship with every
release. Everything else is a **community adapter**: a small program that runs
in its own process, next to ReputeCast, and speaks the adapter protocol
described below. Every adapter, built-in or community, appears in the
provider catalog with the capabilities and terms it declares and passes the
same contract-test suite.

## Community adapters

A community adapter connects a sending provider that has no built-in adapter.
It is usually written in TypeScript with the adapter SDK
(`@reputecast/adapter-sdk`, in [`sdk/adapter-sdk`](../sdk/adapter-sdk)) from
the template in [`sdk/adapter-template`](../sdk/adapter-template), but any
language works: the protocol is JSON over HTTP(S) with HMAC signatures.

- [How it works](#how-it-works)
- [Build an adapter](#build-an-adapter)
- [Install and run an adapter (operators)](#install-and-run-an-adapter-operators)
- [The manifest](#the-manifest)
- [The protocol, version 1](#the-protocol-version-1)
- [Operations](#operations)
- [Data types](#data-types)
- [Failures, health and the circuit breaker](#failures-health-and-the-circuit-breaker)
- [Security model](#security-model)
- [The SDK](#the-sdk)
- [Testing](#testing)

### How it works

```
provider API ◀── adapter process ◀── signed JSON over HTTP(S) ── ReputeCast
                 (your code)      ──  signed JSON responses   ──▶ (internal/providers/remote)
provider webhook ──▶ ReputeCast /hooks/community:<id>/<token> ──▶ adapter verifies and normalizes
```

- **The author** writes the adapter, publishes it (a container image or an
  npm package) and its `manifest.json` in the adapter registry. The manifest
  declares what the adapter can do (capabilities), what the provider's terms
  allow, the connection form (config schema) and the hosts it calls
  (permissions).
- **The operator** (whoever runs the ReputeCast server) runs the adapter
  process and tells ReputeCast where it listens.
- **An instance admin** installs the adapter from its manifest (Adapters tab
  or `reputecast adapter install`). Installed adapters apply to every
  workspace.
- **A workspace admin** connects a provider through it from *Add provider*,
  like any built-in adapter. The provider's credentials are entered in
  ReputeCast, sealed at rest, never returned by the API, and sent to the
  adapter with each call over the signed channel.

ReputeCast proxies every provider operation to the adapter: sending batches,
webhook verification and normalization, domain setup, quota discovery,
suppression sync and webhook registration. It only calls the operations for
the capabilities the installed manifest declares, and only passes the
connection values its config schema declares.

Inside ReputeCast an installed adapter is registered in the provider registry
as `community:<id>` (for example `community:example-esp`), at startup and
whenever it is installed, updated or removed. Every process (`--role=web`,
`worker`, `ingest`) re-reads the installed adapters every 15 seconds, so a
change made in one process reaches the others.

### Build an adapter

```bash
cp -r sdk/adapter-template my-esp-adapter && cd my-esp-adapter
pnpm install
pnpm dev                       # serves the protocol on http://127.0.0.1:4600
pnpm test                      # unit tests + the contract suite (npx reputecast-adapter test)
```

Start from the template's `src/adapter.ts`:

```ts
import { defineAdapter } from "@reputecast/adapter-sdk";

export default defineAdapter({
  id: "example-esp",
  capabilities: { webhooks: true, batch: 100, suppressionSync: "pull" },
  terms: { bulkMarketing: "allowed" },
  async send(msg, ctx) {
    const r = await ctx.http.post("/v1/messages", toPayload(msg));
    return { providerMessageId: r.data.id };
  },
  normalizeEvent(raw) {
    return { type: map[raw.event], code: raw.smtp_code, raw };
  },
});
```

Then describe it in `manifest.json` (capabilities and terms must match the
code; the contract suite checks it), make `pnpm test` pass, and publish the
image and the manifest. The [SDK section](#the-sdk) lists every hook.

### Install and run an adapter (operators)

1. **Run the adapter process** where ReputeCast can reach it: a sidecar
   container, another host, or the same machine. The template's image:

   ```bash
   docker run -d --name example-esp -p 127.0.0.1:4600:4600 \
     -e REPUTECAST_ADAPTER_SECRET=rcas_… ghcr.io/you/example-esp-adapter
   ```

2. **Install it in ReputeCast**, from the registry manifest (an `https://`
   URL) or a local file, and give its address:

   ```bash
   reputecast adapter install https://raw.githubusercontent.com/you/example-esp-adapter/main/manifest.json \
     --url http://example-esp:4600
   ```

   ReputeCast prints the adapter's **signing secret once**. Set it on the
   adapter process as `REPUTECAST_ADAPTER_SECRET` and restart the process.
   Adapters installed from the Adapters tab get a secret too, but it isn't
   shown there: issue one with `reputecast adapter secret <id>` and set the
   address with `reputecast adapter set-url <id> <url>`.

3. **Check it**: `reputecast adapter check example-esp` calls the adapter's
   signed `health` operation and prints its version, or what failed.

4. **Connect a provider** through it from *Add provider* in any workspace.

| Command | What it does |
|---|---|
| `reputecast adapter list` | Installed community adapters, their address, secret age and last check. |
| `reputecast adapter install <manifest> [--url <url>]` | Installs from an `https://` URL or a file; installing an id that is already installed updates its manifest. Prints the signing secret on first install. |
| `reputecast adapter set-url <id> <url>` | Sets where the adapter process listens. |
| `reputecast adapter secret <id>` | Issues a new signing secret and prints it once. The previous secret keeps working for 24 hours so the process can be updated without downtime. |
| `reputecast adapter check <id>` | Calls `health` and records the result. |
| `reputecast adapter remove <id>` | Uninstalls the adapter. Refused while a connected provider uses it. |

The same actions are in the API for instance admins:
`GET|PUT /api/v1/workspaces/{workspace}/adapters/{adapterId}/runtime`,
`POST …/runtime/secret` and `POST …/runtime/check`. Every change writes an
audit entry (`adapter.runtime_update`, `adapter.secret_issue`).

**Addresses.** The adapter's address is checked like other user-supplied
URLs (`internal/netguard`): link-local and cloud metadata addresses are always
refused, redirects are never followed, and loopback or private (RFC 1918)
addresses are refused unless the operator allows them:

```yaml
adapters:
  allow_private_networks: true   # REPUTECAST_ADAPTERS__ALLOW_PRIVATE_NETWORKS=true
```

`http://` is accepted only for loopback and private addresses (a sidecar on
the same host or network); anything reachable over the internet must use
`https://`, because credentials travel with each call.

### The manifest

`manifest.json` is what ReputeCast reads at install time; it is shown before
anything runs, and the catalog badges come from it.

```json
{
  "protocol": 1,
  "kind": "remote",
  "id": "example-esp",
  "name": "Example ESP",
  "version": "0.1.0",
  "maintainer": "example-esp-maintainers",
  "description": "Sends through the Example ESP v1 API.",
  "source_url": "https://github.com/you/example-esp-adapter",
  "last_release": "2026-10-13T00:00:00Z",
  "capabilities": {
    "batch": 100,
    "webhooks": true,
    "registerWebhooks": false,
    "domainSetup": false,
    "suppressionSync": "pull",
    "quotaDiscovery": true,
    "sandbox": false,
    "dedicatedIps": false
  },
  "terms": {
    "bulkMarketing": "allowed",
    "note": "Example ESP allows newsletters to opted-in lists.",
    "url": "https://example-esp.com/aup"
  },
  "pricing": "$0.80 per 1,000",
  "config": [
    { "key": "api_key", "label": "API key", "secret": true, "required": true, "help": "Example ESP › Settings › API keys." },
    { "key": "webhook_signing_key", "label": "Webhook signing key", "secret": true, "required": true },
    { "key": "region", "label": "Region", "options": ["eu", "us"], "required": true }
  ],
  "permissions": { "network": ["api.example-esp.com"] }
}
```

| Field | Rules |
|---|---|
| `protocol` | `1`. ReputeCast refuses versions it doesn't speak. |
| `kind` | `"remote"` (a process speaking this protocol). `"http"` manifests are generic HTTP adapter mappings; `"wasm"` is reserved. |
| `id` | 2–40 characters, `a-z`, `0-9` and `-`, starting with a letter. Stable across versions; the registry key is `community:<id>` and appears in webhook URLs. |
| `name`, `version`, `maintainer` | Shown on the Adapters tab. `version` is the adapter's own (semver). |
| `description`, `homepage`, `source_url`, `last_release`, `latest_version` | Optional. An adapter without a release for a year is flagged *Unmaintained*. |
| `capabilities` | See below. Unknown keys are refused, so a typo can't silently drop a capability. |
| `terms.bulkMarketing` | `allowed`, `forbidden` or `your_policy` (self-hosted). Send Readiness and routing use it. `note` and `url` are shown next to it. |
| `pricing` | Required: the pricing model as people read it ("$0.80 per 1,000", "plan, from $15 per month"). |
| `config` | The connection form (at most 30 fields). `key`: `a-z`, `0-9`, `_`; `secret: true` values are sealed and never shown again (keys containing `password`, `token`, `secret` or ending in `api_key` must be secret); `options` makes a select. |
| `permissions.network` | Hosts the adapter calls: `api.example.com`, `*.example.com`, `host:port`, or `settings:<key>` for the host of a URL entered in the connection form. The SDK's `ctx.http` refuses every other host. |

**Capabilities**

| Key | Values | Operations ReputeCast calls | Badge |
|---|---|---|---|
| `batch` | 1–1000 (default 1) | `send` with up to this many messages | Batch sending · N per call |
| `webhooks` | `true` / `false` | `webhooks/verify`, `webhooks/parse` | Bounce & complaint webhooks |
| `registerWebhooks` | `true` / `false` (needs `webhooks`) | `webhooks/register` | (webhook set up automatically) |
| `domainSetup` | `"full"`, `"dkim_only"`, `false` | `domains/add`, `domains/status` | Domain setup via API |
| `suppressionSync` | `"pull"`, `"push"`, `"two-way"`, `false` | `suppressions/list` (pull), `suppressions/add` (push) | Suppression-list sync |
| `quotaDiscovery` | `true`, `"rate"` (also reports the send rate), `false` | `quota` | Quota (& rate) discovery |
| `sandbox` | `"detection"`, `"test token"`, `"mode"`, `false` | — | Sandbox |
| `dedicatedIps` | a short text ("optional, $25 per IP per month") or `false` | — | Dedicated IPs |
| `rewritesLinks`, `addsOpenPixel` | `true` when the provider tracks by default | — | double-tracking warning |

`health`, `verify` and `send` are always available. Without
`suppressionSync` ReputeCast enforces the shared suppression list locally
before sending (*Local only*); without `webhooks` there is no bounce
protection on that route, which Send Readiness reports.

### The protocol, version 1

**Transport.** ReputeCast sends `POST <adapter url>/v1/<operation>` with a
JSON body (`Content-Type: application/json`, UTF-8) and expects a JSON
response. HTTP/1.1 and HTTP/2 both work; redirects are not followed.
ReputeCast may call an adapter concurrently; adapters must be safe for
concurrent requests.

**Request headers**

| Header | Value |
|---|---|
| `X-ReputeCast-Protocol` | `1` |
| `X-ReputeCast-Timestamp` | Unix time in seconds |
| `X-ReputeCast-Signature` | `v1=<hex>`; during the 24 hours after a secret rotation, two comma-separated entries (new secret first) |
| `Idempotency-Key` | same as the body's `idempotencyKey` |
| `X-ReputeCast-Operation` | same as the body's `operation` (for logs) |
| `User-Agent` | `ReputeCast/<version> (adapter-protocol 1)` |

**Signatures.** Each `v1` entry is the lowercase hex HMAC-SHA256 of
`<timestamp>.<raw body>` keyed with the installation's signing secret (the
whole `rcas_…` string as bytes). The adapter must:

1. reject requests without a matching `v1` entry, or whose timestamp is more
   than 300 seconds from its clock, with `401` and error code `unauthenticated`;
2. check that the body's `operation` matches the URL path and `adapter`
   matches its id (`400 invalid_request` otherwise);
3. sign every response the same way: `X-ReputeCast-Timestamp` and
   `X-ReputeCast-Signature: v1=<hex HMAC-SHA256(secret, "<timestamp>.<body>")>`
   with its current secret, and echo the request's `idempotencyKey` in the
   body. ReputeCast refuses unsigned, mis-signed or stale responses and
   responses for another request. A `401 unauthenticated` answer may be
   unsigned (the adapter may not have the secret yet).

Signing covers the operation, the idempotency key, the connection and the
input, because they are all in the body.

**Request body**

```json
{
  "protocol": 1,
  "operation": "send",
  "adapter": "example-esp",
  "idempotencyKey": "send-8c3f0a6e2b9d4c71a5e0f3b6d2c1a9e8",
  "deadline": "2026-10-13T09:00:30.000Z",
  "connection": {
    "providerId": "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40",
    "workspaceId": "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e00",
    "settings": { "region": "eu" },
    "credentials": { "api_key": "…", "webhook_signing_key": "…" },
    "webhookSecret": "…"
  },
  "input": { "messages": [ … ] }
}
```

- `deadline` is when ReputeCast stops waiting; stop provider calls by then
  (the SDK's `ctx.signal` aborts at the deadline).
- `connection` is the connected provider the call is for. It is absent for
  `health`. `settings` holds the non-secret values of the config schema,
  `credentials` the secret ones; keys the manifest doesn't declare are never
  sent. `webhookSecret` is the provider's ReputeCast-generated webhook secret,
  sent only to adapters with `webhooks` and only when the connection has one.
- Unknown fields must be ignored, in both directions: new optional fields
  don't change the protocol version.

**Response body**

```json
{ "protocol": 1, "idempotencyKey": "send-8c3f…", "result": { … } }
{ "protocol": 1, "idempotencyKey": "send-8c3f…", "error": { "code": "provider_unavailable", "message": "Example ESP answered 503.", "status": 503, "providerCode": "service_unavailable", "retryAfter": 30 } }
```

A `result` comes with `200`; an `error` with the status in the table below.
`message` is a sentence for the event log (at most 2,000 characters; never
include credentials), `status` and `providerCode` are what the provider
answered, `retryAfter` is in seconds.

| `error.code` | HTTP | Meaning | What ReputeCast does |
|---|---|---|---|
| `invalid_request` | 400 | ReputeCast sent something the adapter can't read | permanent error (a bug: report it) |
| `unauthenticated` | 401 | signature or timestamp rejected | adapter unavailable: issue a new secret or fix the clock |
| `too_large` | 413 | the body is over the adapter's limit | permanent error |
| `bad_signature` | 422 | the provider's webhook signature is invalid | webhook refused (`401` to the provider) |
| `provider_throttled` | 429 | the provider is rate limiting | temporary, retried after `retryAfter` |
| `provider_auth` | 502 | the provider rejected the credentials | provider *Down · auth failed*; asks for new credentials |
| `provider_error` | 502 | the provider refused the request for good | permanent error with the provider's code |
| `provider_unavailable` | 503 | the provider is down or timing out | provider outage: failover per stream policy |
| `not_supported` | 501 | the operation isn't implemented | treated as "not supported by this provider" |
| `internal` | 500 | the adapter failed | adapter unavailable, retried |

**Idempotency.** Every request has an `idempotencyKey` (1–128 characters of
`A-Z a-z 0-9 . _ : -`). For `send` it is derived from the message ids, so a
retry of the same batch carries the same key; other operations get a fresh
key per call, kept when ReputeCast's HTTP client retries a request on a
broken connection. Adapters must return the same result for a key they have
already answered successfully (for at least 15 minutes), and must not run a
key twice concurrently; this is what keeps a retried batch from being sent
twice. Errors are not remembered: a retry after an error runs again. The SDK
does all of this.

**Limits and timeouts** (ReputeCast's side)

| Operation | Path | Timeout | Largest response read |
|---|---|---|---|
| `health` | `/v1/health` | 5 s | 64 KiB |
| `verify` | `/v1/verify` | 20 s | 64 KiB |
| `send` | `/v1/send` | 30 s | 4 MiB |
| `webhooks/verify` | `/v1/webhooks/verify` | 5 s | 64 KiB |
| `webhooks/parse` | `/v1/webhooks/parse` | 15 s | 16 MiB |
| `webhooks/register` | `/v1/webhooks/register` | 30 s | 64 KiB |
| `domains/add`, `domains/status` | `/v1/domains/add`, `/v1/domains/status` | 30 s | 1 MiB |
| `quota` | `/v1/quota` | 20 s | 64 KiB |
| `suppressions/list` | `/v1/suppressions/list` | 60 s | 8 MiB |
| `suppressions/add` | `/v1/suppressions/add` | 60 s | 64 KiB |

Requests carry at most `batch` messages and at most 24 MiB; a larger batch is
split into several `send` calls. Forwarded webhooks are at most 5 MiB (the
ingest endpoint's limit). Identifiers in responses (`providerMessageId`,
`messageId`, emails, record names) longer than 512 characters make the
response invalid; longer texts (`message`, `diagnostic`, `originalType`) are
cut at 2,000 characters.

### Operations

`connection` is in every request below except `health`. Results are the
`result` object of the response.

#### `health`

Is the process up, and is it the adapter ReputeCast expects? Called by
`reputecast adapter check`, the runtime API and the circuit breaker's probe.

- input: `{}`
- result: `{ "adapter": "example-esp", "version": "0.1.0", "protocol": 1, "sdk": "@reputecast/adapter-sdk 0.1.0", "capabilities": { … } }`
  (`sdk` and `capabilities` are optional).

ReputeCast checks that `adapter` is the installed id and that `protocol` is
1, and reports a difference between the running version and the installed
manifest.

#### `verify`

Checks the connection's credentials with the provider (connect wizard,
*Replace credentials*, and the health check every few minutes when the
adapter has no `quotaDiscovery`). Maps to `providers.Verifier`.

- input: `{}`
- result: a [Quota](#quota) where every field is optional, for example
  `{ "account": "Growth plan · 50,000 per month" }`. An adapter that can't check
  credentials without sending returns `{}`.

#### `send`

Sends a batch. Maps to `Provider.Send`.

- input: `{ "messages": [Message, …] }` (1 to `batch` messages)
- result: `{ "results": [SendResult, …] }`: exactly one per message, in the
  same order, each echoing its `messageId`.

A recipient the provider refuses is a per-message error; the call as a whole
succeeds. Fail the whole call (`provider_auth`, `provider_unavailable`,
`provider_throttled`) only when no message of the batch was accepted;
otherwise report the remaining messages as temporary per-message errors, so
a retry never sends a message twice.

#### `webhooks/verify`

ReputeCast received a provider webhook at `/hooks/community:<id>/<token>` and
asks whether it is authentic, before storing it and answering the provider.
Maps to `WebhookVerifier` and `WebhookHandshaker`.

- input: `{ "request": WebhookRequest }`
- result: `{}` when the signature is valid, or
  `{ "handshake": { "body": "<text>" } }` when the request is a subscription
  handshake (for example an SNS-style `SubscriptionConfirmation`) that the
  adapter has completed: ReputeCast answers the provider `200` with `body` as
  JSON and stores nothing.
- error `bad_signature` when the request isn't authentic: ReputeCast answers
  the provider `401`, keeps the first 2 KiB for diagnosis and counts it
  toward *webhooks silent* diagnostics.

Keep this fast (5 s timeout: the provider is waiting). When the adapter can't
be reached, ReputeCast answers the provider `503` so it retries later; an
unverified request is never stored as verified.

#### `webhooks/parse`

Turns a webhook into normalized events, in the background job that follows
`webhooks/verify`. Maps to `Provider.ParseWebhook`.

- input: `{ "request": WebhookRequest, "verified": true }`
- result: `{ "events": [Event, …] }` (`[]` for payloads with nothing to
  record, such as opens the provider tracks on its own).

With `"verified": true` ReputeCast already verified this request at receipt
and the adapter must **not** verify again: the stored request has lost the
headers ReputeCast never stores (`Authorization`, `Cookie`) and time-window
checks would fail. With `false`, verify first and answer `bad_signature` when
it fails. If the adapter can't be reached, ReputeCast retries the job later;
events are not lost.

#### `webhooks/register`

Registers ReputeCast's webhook URL at the provider (`registerWebhooks`). Maps
to `WebhookRegistrar`.

- input: `{ "url": "https://reputecast.acme.io/hooks/community:example-esp/<token>" }`
- result: `{}`

Use `connection.webhookSecret` as the provider-side secret when the provider
lets you choose one (basic-auth password, signing key). Register delivery,
bounce, complaint and unsubscribe events; ReputeCast tracks opens and clicks
itself.

#### `domains/add` and `domains/status`

Set up a sending domain at the provider and read its DNS records
(`domainSetup`). Maps to `DomainManager`.

- input: `{ "domain": "news.acme.io" }`
- result: `{ "records": [DNSRecord, …] }`

#### `quota`

The account's allowance and usage (`quotaDiscovery`). Maps to
`QuotaReporter`; the health job calls it every five minutes (and every
minute while the provider is down).

- input: `{}`
- result: a [Quota](#quota) with `kind`, `max` and `used` (and `maxRate` with
  `"rate"`).

#### `suppressions/list`

Pulls the provider's suppression list into the shared list
(`suppressionSync` `pull` or `two-way`). Maps to
`SuppressionSyncer.ListSuppressions`.

- input: `{ "since": "2026-10-12T00:00:00Z" | null, "cursor": "<opaque>" | null }`
- result: `{ "suppressions": [Suppression, …], "nextCursor": "<opaque>" | null }`,
  at most 1,000 per page. ReputeCast follows `nextCursor` for up to 100
  pages per sync.

#### `suppressions/add`

Pushes addresses from the shared list (`suppressionSync` `push` or
`two-way`). Maps to `SuppressionSyncer.AddSuppressions`.

- input: `{ "emails": ["a@example.com", …] }` (at most 500)
- result: `{}`

### Data types

#### Message

```json
{
  "id": "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40",
  "stream": "broadcast",
  "from": { "name": "Acme Newsletter", "address": "news@news.acme.io" },
  "replyTo": { "name": "", "address": "help@acme.io" },
  "to": { "name": "Linh Nguyen", "address": "linh@example.com" },
  "subject": "October product notes",
  "html": "<p>…</p>",
  "text": "…",
  "headers": {
    "List-Unsubscribe": "<https://news.acme.io/p/acme/unsubscribe/…>, <mailto:unsubscribe@news.acme.io>",
    "List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
    "Feedback-ID": "oct:acme:reputecast",
    "X-ReputeCast-Message": "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"
  },
  "tags": { "campaign": "0199…" }
}
```

`stream` is `broadcast`, `automation` or `system`. `replyTo`, `html`, `text`
and `tags` may be absent. The message is already rendered and personalized:
send it as it is, with every header (the unsubscribe headers are required by
mailbox providers), and pass `id` or the `X-ReputeCast-Message` header back on
events (as provider metadata or a custom header) so they match the message.
Turn off provider-side open and click tracking: ReputeCast tracks itself.

#### SendResult

```json
{ "messageId": "0199…40", "providerMessageId": "msg_7fa3" }
{ "messageId": "0199…41", "error": { "code": "recipient_suppressed", "message": "The recipient is on Example ESP's suppression list.", "temporary": false, "throttled": false } }
```

`providerMessageId` is required on success. `error.code` and `error.message`
are the provider's (kept in the event log); `temporary` errors are retried,
permanent ones are not.

#### WebhookRequest

```json
{
  "method": "POST",
  "url": "https://reputecast.acme.io/hooks/community:example-esp/9f2c…",
  "headers": { "content-type": ["application/json"], "x-example-signature": ["t=1760346062,v1=5b1d…"] },
  "body": "<base64 of the raw body>"
}
```

Header names are lowercase. Hop-by-hop headers and `Cookie` are left out.

#### Event

```json
{
  "type": "hard_bounce",
  "recipient": "john@example.com",
  "providerMessageId": "msg_7fa3",
  "messageId": "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41",
  "occurredAt": "2026-10-12T13:41:02Z",
  "originalType": "bounced · hard",
  "originalCode": "550 5.1.1",
  "diagnostic": "The email account that you tried to reach does not exist.",
  "bounceClass": "mailbox_unknown",
  "url": "",
  "userAgent": "",
  "raw": { "event": "bounced", "…": "…" }
}
```

| Field | Rules |
|---|---|
| `type` | `delivered`, `deferred`, `hard_bounce`, `soft_bounce`, `complaint`, `rejected`, `open`, `click`, `unsubscribe` |
| `recipient` | required, lowercase address |
| `providerMessageId`, `messageId` | the provider's id for the message, and ReputeCast's id when the provider echoes it (metadata or the `X-ReputeCast-Message` header); events are matched to messages by either |
| `occurredAt` | required, RFC 3339 |
| `originalType`, `originalCode` | the provider's own event type and code, kept next to the normalized event |
| `bounceClass` | `mailbox_unknown`, `mailbox_full`, `domain_invalid`, `policy_block`, `spam_block`, `reputation`, `temporary`, `other`. Optional: ReputeCast classifies bounces from `originalCode` and `diagnostic` when it is empty (the SDK does the same). |
| `raw` | required: the provider's event as JSON |

#### DNSRecord

`{ "purpose": "dkim", "type": "CNAME", "name": "rc1._domainkey.news.acme.io", "value": "rc1.dkim.example-esp.com", "status": "pending" }`
— `purpose`: `dkim`, `spf`, `return_path`, `dmarc`, `verification`; `type`:
`CNAME`, `TXT`, `MX`; `status` (optional): `pending`, `verified`,
`wrong_value`, `missing`.

#### Quota

`{ "kind": "monthly_plan", "max": 50000, "used": 1234, "maxRate": 14, "sandbox": false, "account": "Growth · 50,000 per month" }`
— `kind`: `rolling_24h` or `monthly_plan`; `max` 0 means unknown or
unlimited; `maxRate` is messages per second (0 when unknown); `sandbox` is
true while the account can only send to verified addresses.

#### Suppression

`{ "email": "john@example.com", "reason": "hard_bounce", "code": "bounce", "createdAt": "2026-10-11T08:00:00Z" }`
— `reason`: `hard_bounce`, `complaint`, `unsubscribe` or `manual`; `code` is
the provider's own reason.

### Failures, health and the circuit breaker

Community adapters report through the same provider health as built-in ones
(`internal/providers/state`): every failed call is recorded with its
operation, status and code, the health job marks the provider *Down* after
repeated failures (or *Down · auth failed* on `provider_auth`), routing fails
over per stream policy, and Broadcasts wait for a decision.

A misbehaving adapter can't stall sending:

- every call has the timeout above and a response size limit;
- the runtime keeps a **circuit breaker per installed adapter**. Five
  consecutive adapter failures (connection refused, timeouts, `internal`,
  `unauthenticated`, unsigned or invalid responses) open it: for 30 seconds,
  doubling up to 5 minutes, every call fails at once as *adapter unavailable*,
  which the provider health treats as an outage. After the pause one call goes
  through; when it succeeds the breaker closes. Provider errors
  (`provider_*`) don't count: they are the provider's health, not the
  adapter's.
- at most 64 calls run against one adapter at a time.

### Security model

- **Signed both ways** with a per-installation secret (`rcas_` + 32 random
  bytes) generated by ReputeCast at install, sealed at rest with the
  instance's secret key, and shown once. Rotation keeps the previous secret
  valid for 24 hours.
- **Least privilege.** ReputeCast calls only the operations the installed
  manifest declares and sends only the declared connection values of the one
  provider a call is for. The SDK refuses network calls to hosts the manifest
  doesn't declare. Run the process with the template's Dockerfile: a non-root
  user, Node's permission model (`--permission`, reads limited to the app
  directory, no child processes or workers), and egress limited to the
  provider's API where your platform allows it.
- **Write-only credentials.** Provider credentials and the signing secret are
  never returned by the API and never logged, by ReputeCast or the SDK.
- **Validated responses.** Responses must be signed, fresh, for the request
  that was sent, of the expected shape, and within the size limits; anything
  else counts as an adapter failure.
- **Webhooks.** ReputeCast only stores a webhook as verified after the
  adapter verified it; when the adapter is unreachable it asks the provider to
  retry.

### The SDK

`@reputecast/adapter-sdk` (Node 24, TypeScript 6, no runtime dependencies)
provides the protocol types, `defineAdapter`, a server, webhook signature
helpers and the contract-test runner.

```ts
import { defineAdapter, serve, MessageRejected, verifyHmac } from "@reputecast/adapter-sdk";
import manifest from "../manifest.json" with { type: "json" };

const adapter = defineAdapter({
  id: "example-esp",
  capabilities: { batch: 100, webhooks: true, suppressionSync: "pull", quotaDiscovery: true },
  terms: { bulkMarketing: "allowed" },
  http: {
    baseUrl: () => process.env.EXAMPLE_ESP_API_URL ?? "https://api.example-esp.com",
    headers: (ctx) => ({ authorization: `Bearer ${ctx.credentials.api_key}` }),
  },
  async send(msg, ctx) { … },                  // or sendBatch(msgs, ctx) for a batch API
  async verify(ctx) { … },                     // optional credential check
  verifyWebhook(req, ctx) { … },               // required with webhooks
  splitWebhook(req, ctx) { … },                // optional: payload → raw events (default: JSON array or object)
  normalizeEvent(raw, ctx) { … },              // required with webhooks
  async registerWebhook(url, ctx) { … },       // registerWebhooks
  async setupDomain(domain, ctx) { … },        // domainSetup (domainStatus optional)
  async discoverQuota(ctx) { … },              // quotaDiscovery
  async syncSuppressions({ since, cursor }, ctx) { … },   // suppressionSync pull / two-way
  async pushSuppressions(emails, ctx) { … },              // suppressionSync push / two-way
});

await serve(adapter, { manifest });            // PORT (4600), HOST (127.0.0.1), REPUTECAST_ADAPTER_SECRET
```

- `defineAdapter` checks the definition when the process starts: every
  declared capability needs its hook, and `serve` refuses a manifest whose id
  or capabilities differ from the code.
- `ctx` carries the connection (`ctx.settings`, `ctx.credentials`,
  `ctx.webhookSecret`), `ctx.http` (a `fetch` client on `http.baseUrl` with
  JSON helpers that maps provider failures to protocol errors: 401/403 →
  `provider_auth`, 429 → `provider_throttled`, 5xx, timeouts and network
  errors → `provider_unavailable`, other 4xx → `provider_error`), `ctx.signal`
  (aborts at the deadline) and `ctx.log` (never logs credentials).
- `send(msg, ctx)` is called for every message of a batch (5 at a time). Throw
  `new MessageRejected(message, { code, temporary })` for a per-message
  refusal; a 4xx from `ctx.http` is one too. A provider-wide failure before any
  message was accepted fails the call; after that the rest become temporary
  per-message errors, as the protocol requires.
- Webhook helpers: `verifyHmac`, `verifyTimestampedHmac`, `verifyBasicAuth`,
  `timingSafeEqual`, and `classifyBounce(code, diagnostic)` (the taxonomy
  ReputeCast uses).
- `serve` verifies signatures and timestamps, refuses undeclared operations,
  enforces body limits and deadlines, remembers successful answers per
  idempotency key for 15 minutes, signs responses, serves `GET /healthz` for
  container health checks, and accepts several comma-separated secrets in
  `REPUTECAST_ADAPTER_SECRET` during a rotation. `REPUTECAST_ADAPTER_ALLOW_HOSTS`
  adds hosts to the network allowlist (local fakes in tests).

### Testing

Every adapter must pass the contract suite, the same scenarios the Go suite
(`internal/providers/contracttest`) runs against built-in adapters:
registration (manifest, capabilities, config schema), signed health and
authentication, send (success, permanent rejection, outage, rejected
credentials, idempotent retry), and each recorded webhook (valid, already
verified, tampered).

```bash
npx reputecast-adapter test            # reads reputecast.contract.ts
```

`reputecast.contract.ts` gives the runner a fake of the provider's API (it
reads `env.mode`: `""`, `"outage"` or `"auth"`, and must refuse
`rejected@contract.test` permanently), the connection values, and the
recorded webhooks with the events they must produce. See the template.

ReputeCast's own CI runs the template through the Go runtime
(`internal/providers/remote`, `TestTemplateAdapterEndToEnd`): it starts the
Node process and runs `contracttest` against it, so the protocol, the SDK and
the runtime are tested together.
