# Integrations hub: the API for Zapier, Make, n8n and WordPress

Settings › Integrations connects Zapier, Make, n8n and WordPress sites. The
ReputeCast project publishes a Zapier app, a Make app, the n8n community node
`n8n-nodes-reputecast` and the "ReputeCast" WordPress plugin. They use only
the public API described here, so anyone can build another integration the
same way. Every endpoint is in the OpenAPI document (`/api/v1/openapi.json`,
tags **Integration API**, **Contacts**, **WordPress sites** and
**Integrations hub**); this page explains how they fit together.

Everything here is free and part of the open-source edition.

## The apps

Each app lives in this repository, with its own README (install, configure,
test, publish) and tests:

- [Zapier](../../integrations/zapier/README.md): the Zapier Platform CLI app, with instant triggers (REST hooks) and the five actions.
- [Make](../../integrations/make/README.md): the Make custom app, with instant triggers (attached webhooks), actions and dynamic fields.
- [n8n](../../integrations/n8n/README.md): the community node package `n8n-nodes-reputecast`, with the ReputeCast and ReputeCast Trigger nodes.
- [WordPress](../../integrations/wordpress/README.md): the "ReputeCast" plugin, with the signup-form block and shortcode, report-in, post drafts and the status endpoint.

## Keys

Connecting creates a key for the integration: a normal API key with an
`integration_id`, named "{Integration} · {label}" ("Zapier · Acme ops",
"WordPress · blog.acme.io"), shown once.

- **The key knows its workspace.** An app needs only the instance URL and
  the key; `GET /api/v1/integration` says which workspace the key belongs to.
- **Bearer authentication:** `Authorization: Bearer rc_live_…`.
- **Limited to its operations.** The key works only for the operations of
  the triggers and actions turned on when connecting, even where its
  permissions would allow more. Anything else answers
  `403 integration_operation_denied`, with a detail that names the
  integration and where to change what it can do.
- **Rotation keeps the old key for 24 hours.** After Rotate in ReputeCast,
  both keys work until the old one's `grace_until`, so the app can be updated
  without failed runs. "Stop the old key now" ends the overlap.
- **Take over** keeps the key's value and changes its owner. The key is
  limited by its owner's permissions: when the owner is deactivated or
  removed, calls answer `403 owner_deactivated` until someone takes it over.
- **People only.** Connecting, changing, rotating, taking over and
  disconnecting need a signed-in member with `integrations:manage`. API keys
  and MCP agents can't hold that permission.
- **HTTP API only.** The operation check runs in the API's request path, so
  integration keys are for the HTTP API. The SMTP relay (phase 1) must refuse
  keys with an `integration_id`.

### Errors every call can get

| Status | `code` | Meaning | What the app should say |
|---|---|---|---|
| 401 | `key_revoked` | The key was revoked (Disconnect, or Revoke in Keys & MCP) | Reconnect in ReputeCast and paste the new key |
| 401 | `key_expired` | The key reached the expiry chosen when connecting | Reconnect in ReputeCast |
| 403 | `owner_deactivated` | The key's owner was deactivated or removed | Ask a member with integrations:manage to take it over |
| 403 | `integration_operation_denied` | The call isn't one of the integration's operations | Turn the trigger or action on in "Change what … can do" |

Errors are RFC 9457 problems (`application/problem+json`) with `type`,
`title`, `status`, `detail` and `code`; some add fields (`missing`, `meta`).
The `detail` is a sentence a person can act on.

## Workflow names

Send the workflow's name and ID with every call, when the platform gives
them:

```
X-ReputeCast-Workflow: Beta application (Typeform) → ReputeCast contact; id=284113907
```

`name="…"` and RFC 8187 `name*=UTF-8''…` forms are accepted; a percent-encoded
name is decoded. Header values are ASCII, so the apps percent-encode (UTF-8) a
name's characters outside printable ASCII and `%`, `;`, `=` and `"`, and keep
the rest as typed: `Stripe payment failed %E2%86%92 payment-failed email;
id=284113907`. The call log shows the header decoded. ReputeCast lists the
workflows seen in the last 30 days on the integration's page ("Zaps using
ReputeCast"), with calls, errors and last run. A call without a name is listed
as "Unnamed Zap" (scenario, workflow).

What each platform gives its app:

| App | Name | ID |
|---|---|---|
| Zapier | Zapier doesn't give apps the Zap's name: each step has an optional "Zap name" field | the Zap's ID (`bundle.meta.zap.id`) |
| Make | the scenario's name (`scenario.name`) | the scenario's ID (`scenario.id`) |
| n8n | the workflow's name | the workflow's ID |
| WordPress | — (the plugin isn't a workflow tool) | — |

## Test connection

```
GET /api/v1/integration
```

Answers the key's workspace, name, integration and what it can do:

```json
{
  "workspace": "acme-newsletter",
  "workspaceName": "Acme Newsletter",
  "integration": "make",
  "key": "Make · Acme ops",
  "triggers": ["contact.subscribed", "form.submitted"],
  "actions": ["upsert_contact", "add_tag"],
  "permissions": ["subscribers:create", "subscribers:edit", "webhooks:manage"]
}
```

The connect dialog's last step waits for this call ("Make called at 14:31 ·
Test connection · 200"); any call with the key counts.

## Triggers (REST hooks)

| Trigger | Event | Needs |
|---|---|---|
| Contact subscribed | `contact.subscribed` | subscribers:view |
| Contact unsubscribed | `contact.unsubscribed` | subscribers:view |
| Contact updated | `contact.updated` | subscribers:view |
| Form submitted | `form.submitted` | subscribers:view |
| Campaign sent | `campaign.completed` | campaigns:view |
| Transactional message bounced | `message.bounced` | transactional:view |

Turning on a workflow that starts with a trigger subscribes; turning it off
unsubscribes:

```
POST /api/v1/workspaces/{workspace}/integration/hooks
X-ReputeCast-Workflow: Unsubscribes → Acme Cloud CRM; id=wf_2Lm8

{ "event": "contact.unsubscribed", "url": "https://n8n.acme.io/webhook/7c1e…/reputecast" }
```

`201` answers the endpoint with its signing secret, **shown only in this
answer**:

```json
{
  "id": "0192…",
  "event": "contact.unsubscribed",
  "url": "https://n8n.acme.io/webhook/7c1e…/reputecast",
  "secret": "whsec_…",
  "signatureHeader": "X-ReputeCast-Signature",
  "timestampHeader": "X-ReputeCast-Timestamp",
  "createdAt": "2026-09-29T11:58:00Z"
}
```

```
DELETE /api/v1/workspaces/{workspace}/integration/hooks/{id}      → 204
GET    /api/v1/workspaces/{workspace}/integration/hooks/{id}      → the webhook without its secret; 404 once it's gone
GET    /api/v1/workspaces/{workspace}/integration/samples/{event} → { "event", "payload" }
GET    /api/v1/workspaces/{workspace}/integration/forms           → { "items": [{ "id": "frm_8c2e1a", "name", "status" }] }
```

- n8n checks a workflow's webhook when the workflow is turned on and when n8n
  starts; a 404 (the endpoint answered 410 Gone, or the integration was
  reconnected) makes it subscribe again.
- The forms list fills the "Form" filter of the Form submitted trigger:
  deliveries carry `form_id`, and the apps drop the other forms' submissions.
  Only keys with that trigger on can call it.

- The trigger must be on for the key (`403 integration_operation_denied`
  otherwise).
- The URL goes through the SSRF guard. A private address answers
  `422 private_address`, naming the instance admins who can allow webhooks to
  private addresses (`webhooks.allow_private_networks`, for n8n on the same
  network as ReputeCast).
- The key's `webhooks:manage` covers only the endpoints it created. People
  see them on Settings › Webhooks as "Managed by n8n", read-only; other keys
  don't see them.
- **410 Gone removes the endpoint** (Zapier's REST Hooks rule). The removal is
  audited with the system as actor.
- Payloads and signatures are the Webhooks page's: the body is the event
  envelope; `X-ReputeCast-Signature` is the hex HMAC-SHA256 of
  `{timestamp}.{raw body}` with the secret, and `X-ReputeCast-Timestamp` is the
  Unix time. Reject anything older than 5 minutes. Failed deliveries retry for
  6 hours (8 attempts), then wait 7 days in the dead-letter queue.
- The sample endpoint answers a fixed, realistic payload of the event for the
  app's field mapping (Lucía Fernández, Weekly Digest), so mapping works before
  anything happens in the workspace.

`form.submitted` fires when a signup is confirmed (at once for single opt-in),
with the form, the answers, the lists, the consent text shown and the
language. The contact events are emitted by the database (an outbox) whatever
changed the contact: forms, imports, the API, the app.

## Actions

| Action | Call | Needs |
|---|---|---|
| Create or update a contact | `POST /api/v1/workspaces/{workspace}/contacts/upsert` | subscribers:create, subscribers:edit |
| Add to a list | `POST /api/v1/workspaces/{workspace}/lists/{listId}/contacts` | subscribers:edit |
| Add a tag | `POST /api/v1/workspaces/{workspace}/contacts/tags` | subscribers:edit |
| Record an event | `POST /api/v1/workspaces/{workspace}/automations/events` | subscribers:edit |
| Send a transactional email ⚑ | `POST /api/v1/workspaces/{workspace}/transactional/messages` | transactional:send |

Dropdowns:

```
GET /api/v1/workspaces/{workspace}/integration/lists      → lists, with double opt-in
GET /api/v1/workspaces/{workspace}/integration/fields     → custom fields: key, label, type
GET /api/v1/workspaces/{workspace}/integration/templates  → published transactional templates and their variables
```

### Create or update a contact

```json
{
  "email": "priya.n@example.com",
  "firstName": "Priya",
  "language": "es_MX",
  "fields": { "plan": "trial" },
  "tags": ["beta-applicant"],
  "lists": ["0192…"],
  "consent": "double"
}
```

- Matches by email. Creating needs subscribers:create; updating an existing
  contact needs subscribers:edit.
- `language` is a BCP 47 code; "Spanish" and "es_MX" are normalized to `es` and
  `es-MX`. An unknown value answers 422 with the closest match.
- `consent`: `single` adds to the lists now; `double` sends the confirmation
  email first. Without it, each list's own setting applies.

### Consent is kept

An action never adds someone back to a list they left, and never adds a
suppressed address:

```json
{
  "status": 409,
  "code": "unsubscribed",
  "detail": "rob@example.net left Beta Program on 2 Oct. ReputeCast doesn't add people back to a list they left.",
  "meta": { "list": "Beta Program", "listId": "0192…", "leftAt": "2026-10-02T13:41:00Z" }
}
```

`409 suppressed` carries `meta.reason` and `meta.since`. Apps should treat both
as "skipped" for that record, not as a broken workflow: the rest of the
workflow keeps running.

### Send a transactional email

Phase 1's send API, unchanged: a published template's key, a recipient, the
data and an optional language; send an `Idempotency-Key`. `422
missing_variables` lists what's missing in `missing`. The call log masks the
variables the template marks sensitive.

## The call log

Every call with an integration key is kept 7 days on the integration's page:
time, workflow, action, target, result and duration, with the request and the
response. Bodies follow the message log's rules: sensitive variables and
secret-like fields are masked, and contacts' addresses show only to members
with subscribers:view ("a contact" otherwise, "Erased contact" after an
erasure). Calls that change something are also in the audit log, with the key
as actor; reads aren't.

## Health

Integrations are watched passively; nothing is polled. An integration needs
attention when calls are rejected because its key was revoked or expired or
its owner left, its key expires within 7 days, its managed webhooks have
failed for 30 minutes, more than half of its calls failed in the last hour (at
least 4, not counting 404, 409 and 422), or a site hasn't reported in for 24
hours. Members with integrations:manage get "Integration needs attention" once
per problem.

## WordPress sites

One key per site; a site connects to one workspace. The key holds:

- `subscribers:view`, only to list the signup forms (it can't read contacts)
- `campaigns:edit`, only to create drafts from posts (it can't open, change or
  send any other campaign)

### Report in

The plugin reports in when **Connect** is pressed in WordPress › Settings ›
ReputeCast, then every 12 hours and with every post:

```
POST /api/v1/workspaces/{workspace}/wordpress/report
{
  "siteUrl": "https://blog.acme.io",
  "name": "Acme Blog",
  "pluginVersion": "1.3.0",
  "wordpressVersion": "6.9.2",
  "phpVersion": "8.3",
  "feedUrl": "https://blog.acme.io/feed/",
  "languagePlugin": "wpml",
  "categories": [{ "id": "3", "name": "Announcements", "slug": "announcements" }],
  "forms": [{ "form": "frm_8c2e1a", "kind": "block", "places": [{ "title": "About", "url": "https://blog.acme.io/about/" }] }]
}
```

It answers the workspace, the site's address as ReputeCast lists it, and the
newest plugin release ReputeCast knows (`latestPlugin`). A site already
connected to another workspace answers 409. The first report connects the site
(audited as `wordpress.site_connected`).

### Signup forms

```
GET /api/v1/workspaces/{workspace}/wordpress/forms
```

Lists the workspace's forms for the block and the shortcode: `id` (`frm_…`),
name, status, `siteAllowed` (the site is one of the form's allowed domains; the
block warns the editor when it isn't), the shortcode
(`[reputecast_form id="frm_8c2e1a"]`) and the form's public script. Forms load
through that script, as embeds do, so they keep working after the site is
disconnected.

### Drafts from posts

```
POST /api/v1/workspaces/{workspace}/wordpress/drafts
{ "post": { "id": "812", "url": "https://blog.acme.io/scheduled-rollouts/", "title": "Scheduled rollouts are here",
            "excerpt": "…", "imageUrl": "https://blog.acme.io/…/hero.png", "categories": ["announcements"],
            "publishedAt": "2026-10-15T14:04:00Z", "language": "en" } }
```

The site's post-draft rule, set in ReputeCast, decides. It answers
`{ "created": true, "campaignId", "name" }`, or `created: false` with a reason:
`no_rule`, `rule_off`, `category` (not one of the rule's categories) or `exists`
(the post already has a draft). The draft takes the title as its subject and
the excerpt as its preheader; the template's post section gets
`{{ post.title }}`, `{{ post.excerpt }}`, `{{ post.image_url }}` and
`{{ post.url }}` (a post without a featured image leaves out the template's
`mj-image` of `{{ post.image_url }}`). Drafts only: nothing is sent until a
person reviews it.

**Translations (plugin 1.3.0, WPML or Polylang).** The plugin sends the post in
the site's default language with its published translations:

```
"translations": [{ "id": "813", "language": "es", "url": "https://blog.acme.io/es/despliegues/",
                   "title": "Los despliegues programados ya están aquí", "excerpt": "…", "imageUrl": "…" }]
```

- The draft is written in the post's `language` when the workspace offers it.
- Each translation in a language the workspace offers becomes a language
  version: the template's post section filled with the translation (its
  featured image, else the post's), marked Up to date. Other languages are left
  out. The answer lists the versions added in `languages`.
- A translation published later is sent with the post again: the answer is
  `exists`, and the new translation joins the draft while the campaign is
  still editable. The post's draft is found through any of its translations'
  ids, so a group of translations makes one draft.

### Check now

"Check now" calls `GET {site}/wp-json/reputecast/v1/status` once, through the
SSRF guard, with a 10-second limit. The plugin should answer 200 quickly. A site
behind a firewall or basic auth can't be checked, but its reports still arrive.

The call is signed with the site's key, so the plugin answers only the
workspace it's connected to:

```
X-ReputeCast-Timestamp: 1792138920
X-ReputeCast-Signature: 45f9841e69974b8168b517da522ad7c914b7464255f7dfeb4612923d5d6cb199
```

The signature is the hex HMAC-SHA256 of `{timestamp}.status`, keyed with the
SHA-256 digest (32 raw bytes) of the site's key: ReputeCast keeps only that
digest, and the plugin computes it from the key it holds. During a rotation
the header carries one signature per live key, comma-separated. The plugin
answers 401 to a call without a valid signature or more than 5 minutes off,
and 200 with its plugin, WordPress and PHP versions otherwise.
