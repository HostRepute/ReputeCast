# API, webhooks and MCP

- [REST API](#rest-api)
- [API keys](#api-keys)
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
| Transactional messages | `message.sent`, `message.delivered`, `message.deferred`, `message.bounced`, `message.complained`, `message.rejected`, `message.failed`, `message.opened`, `message.clicked` |
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
| `X-ReputeCast-Signature` | Hex HMAC-SHA256 of `<timestamp>.<raw body>`, keyed with the endpoint's signing secret (`whsec_…`). |

Verify the signature against the raw body, and refuse old timestamps:

```js
import { createHmac, timingSafeEqual } from 'node:crypto'

export function verify(secret, rawBody, headers) {
  const ts = headers['x-reputecast-timestamp']
  const sig = headers['x-reputecast-signature'] ?? ''
  if (Math.abs(Date.now() / 1000 - Number(ts)) > 300) return false
  const want = createHmac('sha256', secret).update(`${ts}.${rawBody}`).digest('hex')
  return sig.length === want.length && timingSafeEqual(Buffer.from(sig), Buffer.from(want))
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
