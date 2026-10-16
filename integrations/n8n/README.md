# n8n-nodes-reputecast

The n8n community nodes of [ReputeCast](../../README.md), the open-source,
self-hosted newsletter platform: workflows start from ReputeCast events, or
add contacts, tag them, record events and send transactional email. They are
free, like everything in ReputeCast, and use only the public API
([`docs/integrations/hub.md`](../../docs/integrations/hub.md)).

Version 1.1.0 · built and tested with n8n-workflow 2.43 (nodes API 1) ·
self-hosted n8n, and n8n Cloud once verified: the nodes have no dependencies
and import only n8n-workflow and Node's crypto.

## Nodes

**ReputeCast Trigger** starts a workflow when:

| Event | ReputeCast event | Needs |
|---|---|---|
| Contact Subscribed | `contact.subscribed` | subscribers:view |
| Contact Unsubscribed | `contact.unsubscribed` | subscribers:view |
| Contact Updated | `contact.updated` | subscribers:view |
| Form Submitted (any form, or one) | `form.submitted` | subscribers:view |
| Campaign Sent | `campaign.completed` | campaigns:view |
| Transactional Message Bounced | `message.bounced` | transactional:view |

Activating the workflow creates a webhook in ReputeCast (shown read-only as
"Managed by n8n" on Settings › Webhooks), deactivating it removes it, and n8n
checks the webhook still exists whenever it starts or the workflow is turned
on. Every delivery is checked against the webhook's signing secret; one that
doesn't match, or is more than 5 minutes old, is refused with 401 and starts
nothing.

**ReputeCast** does:

| Resource › Operation | What it does | Needs |
|---|---|---|
| Contact › Create or Update | Matches by email; sets names, custom fields, language and tags; adds to lists with single or double opt-in | subscribers:create, subscribers:edit |
| Contact › Add to List | Single or double opt-in | subscribers:edit |
| Contact › Add Tag | One or more tags | subscribers:edit |
| Event › Record | An API event with properties; starts the automations that listen for it | subscribers:edit |
| Transactional Email › Send | A published template, a recipient, its variables (mapped field by field) and an optional language | transactional:send ⚑ |

n8n's AI Agent can use the ReputeCast node as a tool. What it can do is what
the key may do: Send a transactional email starts off when you connect n8n.

## Install

In n8n, open **Settings › Community nodes › Install**, enter
`n8n-nodes-reputecast` and confirm. On n8n Cloud, the nodes appear in the
nodes panel once n8n has verified them. Self-hosted n8n can also install them
with `npm install n8n-nodes-reputecast` in `~/.n8n/nodes`.

## Credentials

1. In ReputeCast, open **Settings › Integrations › n8n › Connect**. Choose
   what n8n may do (Send a transactional email starts off: it needs the
   sensitive transactional:send), then **Create key**. ReputeCast shows the
   key once.
2. In n8n, add a **ReputeCast API** credential with:
   - **Instance URL**: your ReputeCast address, for example `https://reputecast.acme.io`.
   - **Key**: the key from step 1 (`rc_live_…`).
3. Save. n8n tests the connection (`GET /api/v1/integration`), and
   ReputeCast's connect dialog shows the call arriving. The credential learns
   which workspace the key belongs to and keeps it.

n8n calls ReputeCast for actions, so it only needs to reach your instance.
Triggers are the other way round: ReputeCast posts events to n8n. When n8n
runs on a private address (`n8n.acme.internal`), an instance admin has to
allow webhooks to private addresses in ReputeCast
(`webhooks.allow_private_networks`).

## Good to know

- **Workflow names.** Every call carries `X-ReputeCast-Workflow: <workflow
  name>; id=<workflow ID>`, the name percent-encoded where it isn't plain
  ASCII, so ReputeCast lists the workflows that use it with their calls and
  errors (Settings › Integrations › n8n).
- **Consent is kept.** ReputeCast never adds someone back to a list they left,
  never adds a suppressed address and doesn't send transactional email to
  one. Such an item comes out as `{ "skipped": true, "code": "unsubscribed",
  "reason": "rob@example.net left Beta Program on 2 Oct…" }` (with the list
  and the date, or the suppression's reason and date), and the workflow
  carries on with the other items.
- **Errors in ReputeCast's words.** Other refusals fail the node with the
  sentence ReputeCast answered: a missing template variable, an action
  turned off ("Change what n8n can do in Settings › Integrations"), a
  revoked key ("Reconnect n8n…"), a key whose owner left ("…take it
  over…"). Turn on **Continue On Fail** to keep going instead.
- **Rotating** the key in ReputeCast keeps the old one working for 24 hours:
  paste the new one into the credential in the meantime.
- **Idempotency.** Transactional Email › Send takes an optional Idempotency
  Key, such as the triggering event's ID; ReputeCast sends nothing new when
  the same value comes again within 24 hours (the key sent is
  `n8n-<workflow ID>-<value>`).

## Develop and test

```bash
cd integrations/n8n
npm install
npm test            # vitest: the nodes against fakes of n8n's contexts
npm run lint        # n8n-node lint: n8n's community-node rules, strict
npm run typecheck   # the nodes and their tests
npm run build       # dist/, as published
npm run dev         # n8n on http://localhost:5678 with these nodes loaded
```

The tests run the nodes against fakes of n8n's function contexts. Every
request the nodes make, and every answer the tests mock, is checked against
the OpenAPI document the server generates (`web/openapi.json`, through
[`../contract/openapi-contract.mjs`](../contract/openapi-contract.mjs)).

`test/live.test.mts` runs both nodes against a real instance when you give
it one: it creates contacts, sends a transactional email, and turns a
trigger on, receives a signed delivery and turns it off.

```bash
# a dev instance with webhooks to private addresses allowed, and an n8n
# connection with every trigger and action on:
REPUTECAST_URL=http://localhost:8080 REPUTECAST_KEY=rc_live_… npm test
```

## Publish

Publishing happens on npm and in n8n's creator portal, outside this
repository. Since 1 May 2026 n8n accepts only community nodes published from
GitHub Actions with npm provenance.

1. Raise `version` in `package.json` and add it to `CHANGELOG.md`.
2. `npm run lint` and `npm run build` must pass: verification runs the same
   linter.
3. Publish with provenance from a GitHub Actions workflow (the n8n template's
   `publish.yml`: `npm run release` with `id-token: write`, OIDC trusted
   publishing or an `NPM_TOKEN` secret), from a tag of this repository, in
   `integrations/n8n`.
4. Submit the package for verification in the
   [n8n creator portal](https://creators.n8n.io); once verified it shows in
   n8n Cloud's nodes panel.

## License

[MIT](LICENSE.md): n8n's community-node rules and verification require it.
The rest of ReputeCast is AGPL-3.0.
