# API, webhooks and MCP

- [REST API](#rest-api)
- [API keys](#api-keys)
- [Client SDKs](#client-sdks)
- [Outgoing webhooks](#outgoing-webhooks)
- [MCP server for AI agents](#mcp-server-for-ai-agents)

## REST API

The API lives under `/api/v1`. The OpenAPI 3.1 document at
`<public URL>/api/openapi.json` (or `reputecast openapi`) describes every
operation, including the permission each one needs (`x-permission`).

Errors are [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) problem
documents with a stable `code`, and permission errors name who can help.

## API keys

Create keys in Settings › Keys & MCP. A key carries a subset of your own
permissions, optionally scoped to some lists and with an expiry. It is shown
once. Send it as a bearer token:

```bash
curl -H "Authorization: Bearer rc_live_…" https://news.example.com/api/v1/…
```

## Client SDKs

ReputeCast publishes clients for TypeScript, Python and Go, built in this
repository (`sdk/client-ts`, `sdk/client-python`, `sdk/client-go`) from the
public OpenAPI document: every operation an API key may call
(`reputecast openapi --public`, committed as `sdk/openapi/public.json` and
its OpenAPI 3.0 copy `public-3.0.json`). Each client:

- sends the key as a bearer token and its own User-Agent, so Settings ›
  Integrations names it ("ReputeCast SDK for Python 1.7.0");
- sends a request again after a 429, 502, 503 or 504, twice by default:
  GET, HEAD, PUT and DELETE requests, and POSTs with an `Idempotency-Key`
  (the server answers a repeat with the first result instead of acting
  twice). It waits as `Retry-After` asks, at most 30 seconds, or backs off
  exponentially from half a second, with jitter;
- turns error answers into an exception (an error in Go) with the problem's
  fields;
- verifies webhook signatures.

| Language | Install | Needs |
|---|---|---|
| TypeScript and JavaScript | `npm install @reputecast/client` | Node 20 or later (ESM); its one dependency is `openapi-fetch` |
| Python | `pip install reputecast` | Python 3.11 or later; `httpx` and `attrs` |
| Go | `go get github.com/hostrepute/reputecast/sdk/client-go` | Go 1.27.3 or later |

**Versions follow the server's.** Each release (v1.7.0) publishes
`@reputecast/client` 1.7.0, `reputecast` 1.7.0 and the Go module version
v1.7.0 (tag `sdk/client-go/v1.7.0`), generated from that release's API. Use
the SDK of your server's version or an earlier one: a newer SDK knows
operations an older server doesn't have.

### Create a client

The base URL is your instance's public URL; the paths start with `/api/v1`.

```ts
import { createClient, unwrap } from "@reputecast/client";

const rc = createClient({ baseUrl: "https://news.example.com", apiKey: process.env.REPUTECAST_KEY! });
const lists = unwrap(await rc.GET("/api/v1/workspaces/{workspace}/lists", { params: { path: { workspace: "acme-newsletter" } } }));
```

```python
import os
from reputecast import ReputeCast, unwrap
from reputecast.api.audience import list_lists

rc = ReputeCast(api_key=os.environ["REPUTECAST_KEY"], base_url="https://news.example.com")
lists = unwrap(list_lists.sync_detailed(client=rc, workspace="acme-newsletter"))
```

```go
rc, err := reputecast.New("https://news.example.com", os.Getenv("REPUTECAST_KEY"))
if err != nil {
	return err
}
res, err := rc.ListListsWithResponse(ctx, "acme-newsletter")
if err != nil {
	return err
}
for _, l := range res.JSON200.Items { … }
```

The TypeScript client is [openapi-fetch](https://openapi-ts.dev/openapi-fetch/)
typed with the API's paths: `rc.GET`, `rc.POST`, `rc.PUT`, `rc.PATCH` and
`rc.DELETE` take the path as the document writes it. Options: `fetch` (your
own) and `maxRetries` (0 turns retries off). In Python every operation is a
module of `reputecast.api`, by tag, with `sync`, `sync_detailed`, `asyncio`
and `asyncio_detailed`; bodies and answers are the classes of
`reputecast.models`. `ReputeCast` takes `timeout` (seconds) and
`max_retries`. In Go every operation is a method of the client
(`<Operation>WithResponse`); options are `WithMaxRetries`, `WithTimeout`
(each attempt), `WithTransport` and `WithClientOptions`.

### Send a transactional message

With an `Idempotency-Key`, a retry (the client's or yours) within 24 hours
never sends twice.

```ts
const msg = unwrap(
  await rc.POST("/api/v1/workspaces/{workspace}/transactional/messages", {
    params: { path: { workspace: "acme-newsletter" }, header: { "Idempotency-Key": `pwreset-${user.id}-${token.id}` } },
    body: { template: "password-reset", to: { email: user.email, name: user.name }, data: { first_name: user.firstName, reset_url: resetUrl } },
  }),
);
```

```python
from reputecast.api.transactional_messages import send_transactional_message
from reputecast.models import TransactionalSend

msg = unwrap(send_transactional_message.sync_detailed(
    client=rc,
    workspace="acme-newsletter",
    idempotency_key=f"pwreset-{user.id}-{token.id}",
    body=TransactionalSend.from_dict({
        "template": "password-reset",
        "to": {"email": user.email, "name": user.name},
        "data": {"first_name": user.first_name, "reset_url": reset_url},
    }),
))
```

```go
key := fmt.Sprintf("pwreset-%s-%s", user.ID, token.ID)
data := map[string]any{"first_name": user.FirstName, "reset_url": resetURL}
res, err := rc.SendTransactionalMessageWithResponse(ctx, "acme-newsletter",
	&reputecast.SendTransactionalMessageParams{IdempotencyKey: &key},
	reputecast.TransactionalSend{
		Template: "password-reset",
		To:       reputecast.TransactionalAddress{Email: user.Email, Name: &user.Name},
		Data:     &data,
	})
```

### Paginate

Lists that page with a cursor (contacts, suppressions, the audit log…)
answer `items` and `nextCursor`; pass `nextCursor` back as `cursor` until
it is empty. In TypeScript `paginate` does it:

```ts
import { paginate } from "@reputecast/client";

for await (const contact of paginate(rc, "/api/v1/workspaces/{workspace}/contacts", {
  params: { path: { workspace: "acme-newsletter" }, query: { pageSize: 100 } },
})) {
  console.log(contact.email);
}
```

```python
from reputecast import UNSET
from reputecast.api.audience import list_contacts

cursor = UNSET
while True:
    page = unwrap(list_contacts.sync_detailed(client=rc, workspace="acme-newsletter", page_size=100, cursor=cursor))
    for contact in page.items:
        print(contact.email)
    if not page.next_cursor:
        break
    cursor = page.next_cursor
```

```go
size := int64(100)
params := &reputecast.ListContactsParams{PageSize: &size}
for {
	page, err := rc.ListContactsWithResponse(ctx, "acme-newsletter", params)
	if err != nil {
		return err
	}
	for _, c := range page.JSON200.Items { … }
	if page.JSON200.NextCursor == nil || *page.JSON200.NextCursor == "" {
		break
	}
	params.Cursor = page.JSON200.NextCursor
}
```

### Errors

An error answer (an RFC 9457 problem) becomes a `ReputeCastError` in
TypeScript (thrown by `unwrap` and `paginate`) and Python (raised by
`unwrap`), and an `*reputecast.Error` in Go (the call's error: when it is
nil, the success field such as `JSON200` is set). Each has `status`, the
stable `code` (`permission_denied`, `validation_failed`, `not_found`…),
`title`, `detail` (the sentence to show), `errors` (the fields at fault),
and for a missing permission `permission` and `helpers`, the members who
hold it. An error page that isn't a problem (a proxy's) keeps its status,
with its text as the detail.

```ts
try {
  unwrap(await rc.GET("/api/v1/workspaces/{workspace}/members", { params: { path: { workspace: "acme-newsletter" } } }));
} catch (err) {
  if (err instanceof ReputeCastError && err.code === "permission_denied") console.log(err.detail, err.helpers);
}
```

```python
try:
    unwrap(list_members.sync_detailed(client=rc, workspace="acme-newsletter"))
except ReputeCastError as err:
    print(err.status, err.code, err.detail)
```

```go
var apiErr *reputecast.Error
if errors.As(err, &apiErr) && apiErr.Code == "permission_denied" {
	log.Print(apiErr.Detail, apiErr.Helpers)
}
```

### Verify a webhook

Pass the raw body, exactly as received, the request's headers and the
endpoint's signing secret (or several while you rotate your own). The
check refuses timestamps more than five minutes from now and compares
signatures in constant time; a delivery that fails throws (returns an
error in Go): answer it with 400 or 401.

```ts
import { verifyWebhook } from "@reputecast/client";

await verifyWebhook({ secrets: process.env.REPUTECAST_WEBHOOK_SECRET!, body: rawBody, headers: req.headers });
```

```python
from reputecast import webhooks

webhooks.verify(os.environ["REPUTECAST_WEBHOOK_SECRET"], request.get_data(), request.headers)
```

```go
import "github.com/hostrepute/reputecast/sdk/client-go/webhooks"

if err := webhooks.Verify([]string{os.Getenv("REPUTECAST_WEBHOOK_SECRET")}, body, r.Header, time.Now()); err != nil {
	http.Error(w, "invalid signature", http.StatusUnauthorized)
	return
}
```

The three are tested against the same deliveries, signed by the server
(`sdk/testdata/webhook-vectors.json`), and run against the app with the
demo data in CI (`make sdks`).

## Outgoing webhooks

Settings › Webhooks delivers signed events to your endpoints, with retries (up
to 8 attempts) and a delivery log.

Events:

| Group | Events |
|---|---|
| Contacts | `contact.subscribed`, `contact.unsubscribed`, `contact.updated`, `contact.erased` |
| Email | `email.sent`, `email.delivered`, `email.deferred`, `email.bounced`, `email.complained`, `email.opened`, `email.clicked` |
| Campaigns | `campaign.scheduled`, `campaign.started`, `campaign.paused`, `campaign.resumed`, `campaign.completed` |
| Approvals | `approval.requested`, `approval.decided` |
| Sending providers | `provider.down`, `provider.recovered` |
| Automations | `automation.published` |
| Transactional messages | `message.sent`, `message.delivered`, `message.deferred`, `message.bounced`, `message.complained`, `message.rejected`, `message.failed`, `message.opened`, `message.clicked`, `message.cancelled` |
| Forms | `form.submitted` |
| Referrals | `referral.counted`, `referral.held`, `referral.not_counted`, `referral.milestone_reached`, `referral.reward_sent`, `referral.reward_failed` |
| Paid members | `subscription.trial_started`, `subscription.started`, `subscription.renewed`, `subscription.payment_failed`, `subscription.cancel_scheduled`, `subscription.ended`, `subscription.changed`, `access.complimentary_granted`, `access.complimentary_ended`, `gift.purchased`, `gift.redeemed`, `conversion.recorded` |
| Products and orders | `order.paid`, `order.refunded`, `order.disputed` |
| Website | `post.published`, `post.unpublished` |
| Push | `push.clicked` |
| SMS | `sms.delivered`, `sms.failed`, `sms.clicked`, `sms.opted_out`, `sms.opted_in`, `sms.replied` |
| WhatsApp | `whatsapp.delivered`, `whatsapp.read`, `whatsapp.failed`, `whatsapp.replied`, `whatsapp.opted_out`, `whatsapp.opted_in`, `whatsapp.template_status` |

Each request is a JSON `POST` with these headers:

| Header | Value |
|---|---|
| `X-ReputeCast-Event` | The event name. |
| `X-ReputeCast-Delivery` | The delivery id, the same on every retry. |
| `X-ReputeCast-Timestamp` | Unix time in seconds. |
| `X-ReputeCast-Signature` | Hex HMAC-SHA256 of `<timestamp>.<raw body>`, keyed with the endpoint's signing secret (`whsec_…`). For 24 hours after the secret is rotated it holds two comma-separated signatures, the new secret's first: accept a delivery when either matches. |

An endpoint can take its `email.*` events in batches (the endpoint's "Send
email events in batches" switch, `batchEmail` in the API). It then gets one
request of type `email.batch` with up to 100 events, oldest first, about every
15 seconds while events come in. Each item of `data.events` is the envelope the
event would have been posted as alone; other events still go one by one.

```json
{
  "id": "0193…",
  "type": "email.batch",
  "created_at": "2026-11-01T09:00:15Z",
  "workspace": "acme-newsletter",
  "data": {
    "count": 2,
    "events": [
      { "id": "0193…", "type": "email.delivered", "created_at": "2026-11-01T09:00:02Z", "workspace": "acme-newsletter", "data": { "message_id": "…", "to": "ana@example.com", "at": "2026-11-01T09:00:02Z" } },
      { "id": "0193…", "type": "email.opened", "created_at": "2026-11-01T09:00:09Z", "workspace": "acme-newsletter", "data": { "message_id": "…", "to": "ana@example.com", "at": "2026-11-01T09:00:09Z" } }
    ]
  }
}
```

Verify the signature against the raw body, and refuse old timestamps. The
[client SDKs](#verify-a-webhook) do it; by hand:

```js
import { createHmac, timingSafeEqual } from 'node:crypto'

export function verify(secret, rawBody, headers) {
  const ts = headers['x-reputecast-timestamp']
  const sigs = (headers['x-reputecast-signature'] ?? '').split(',').map((s) => Buffer.from(s.trim()))
  if (!/^\d+$/.test(ts ?? '') || Math.abs(Date.now() / 1000 - Number(ts)) > 300) return false
  const want = Buffer.from(createHmac('sha256', secret).update(`${ts}.${rawBody}`).digest('hex'))
  return sigs.some((sig) => sig.length === want.length && timingSafeEqual(sig, want))
}
```

Outgoing webhooks can't reach private networks unless
`REPUTECAST_WEBHOOKS__ALLOW_PRIVATE_NETWORKS` is set
([configuration](../deploy/configuration.md)).

## MCP server for AI agents

Create an agent in Settings › Keys & MCP and give it permissions: for example
reports and deliverability, or the right to pause a send but never to start
one. Then point your MCP client at `<public URL>/mcp` with the agent's token.

Agents pass the same permission check as people and API keys, can never
exceed the member who created them, and their actions appear in the audit log.
