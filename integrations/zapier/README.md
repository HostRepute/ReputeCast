# ReputeCast for Zapier

The Zapier app of [ReputeCast](../../README.md), the open-source, self-hosted
newsletter platform. Zaps start from ReputeCast events, or add contacts, tag
them, record events and send transactional email from your other apps. It
is free, like everything in ReputeCast, and uses only the public API
([`docs/integrations/hub.md`](../../docs/integrations/hub.md)).

Version 1.2.0 · Zapier Platform CLI 19 · Node 22 or later.

## What it can do

| Trigger | Event | Needs |
|---|---|---|
| Contact Subscribed | `contact.subscribed` | subscribers:view |
| Contact Unsubscribed | `contact.unsubscribed` | subscribers:view |
| Contact Updated | `contact.updated` | subscribers:view |
| Form Submitted (optionally one form) | `form.submitted` | subscribers:view |
| Campaign Sent | `campaign.completed` | campaigns:view |
| Transactional Message Bounced | `message.bounced` | transactional:view |

| Action | What it does | Needs |
|---|---|---|
| Create or Update Contact | Matches by email; sets names, custom fields, language and tags; adds to lists with single or double opt-in | subscribers:create, subscribers:edit |
| Add Contact to List | Single or double opt-in | subscribers:edit |
| Add Tag to Contact | One or more tags | subscribers:edit |
| Record Event | An API event with properties; starts the automations that listen for it | subscribers:edit |
| Send Transactional Email | A published template, a recipient, its data and an optional language | transactional:send ⚑ |

Triggers are instant: turning a Zap on creates a webhook in ReputeCast
(shown read-only as "Managed by Zapier" on Settings › Webhooks), turning it
off removes it. Deliveries are signed, and the app drops any delivery whose
signature doesn't match the Zap's webhook secret or that is more than 5
minutes old.

## Connect it

1. In ReputeCast, open **Settings › Integrations › Zapier › Connect**.
   Choose what Zapier may do (Send a transactional email starts off: it
   needs the sensitive transactional:send), then **Create key**. ReputeCast
   shows the key once.
2. In Zapier, add a ReputeCast step and create a connection with:
   - **Instance URL**: your ReputeCast address, for example `https://reputecast.acme.io`.
   - **Key**: the key from step 1 (`rc_live_…`).

   The connection asks ReputeCast which workspace the key belongs to, so
   nothing else is needed. ReputeCast's connect dialog shows the call
   arriving ("Zapier called at 14:31 · Test connection · 200").

Zapier's servers call your instance, so it must be reachable from the
internet over HTTPS. An instance on a private network can't use Zapier; use
n8n on the same network instead.

## Good to know

- **Zap name.** ReputeCast lists the Zaps that use it, with their calls and
  errors, on Settings › Integrations › Zapier. Zapier gives apps a Zap's ID
  but not its name, so every step has an optional **Zap Name** field: type
  the Zap's name there to tell your Zaps apart. Without it, the Zap is
  listed as "Unnamed Zap". The app sends both as
  `X-ReputeCast-Workflow: <name>; id=<Zap ID>`, the name percent-encoded
  where it isn't plain ASCII.
- **Consent is kept.** ReputeCast never adds someone back to a list they
  left, and never adds a suppressed address or sends transactional email to
  one. Those runs end as **Halted** with ReputeCast's reason ("rob@example.net
  left Beta Program on 2 Oct…"); they don't count as errors, so the Zap
  keeps running for everyone else.
- **Errors in ReputeCast's words.** Every other refusal shows the sentence
  ReputeCast answered: a missing template variable ("payment-failed v1 needs
  amount_due."), an action turned off ("Change what Zapier can do in
  Settings › Integrations"), a key whose owner left ("…take it over in
  Settings › Integrations › Zapier; the key stays the same").
- **A revoked or expired key** makes Zapier ask you to reconnect: create a
  new key in ReputeCast (Reconnect) and paste it into the connection.
- **Rotating** the key in ReputeCast keeps the old one working for 24 hours,
  so you can paste the new key without failed runs.
- **Idempotency.** Send Transactional Email takes an optional Idempotency
  Key, such as the triggering event's ID. ReputeCast sends nothing new when
  the same Zap runs again with the same value within 24 hours (the key sent
  is `zap-<Zap ID>-<value>`).
- **Language.** Send Transactional Email's Language picks a language version
  of the template (`es`, `de`…). Without it, the contact's language is used,
  then the workspace's fallback.

## Develop and test

```bash
cd integrations/zapier
npm install
npm test          # unit tests: every request is checked against web/openapi.json
npm run lint      # ESLint, and Zapier's schema validation (zapier-platform validate --without-style)
npm run build     # build/build.zip and build/source.zip, as Zapier receives them
```

The tests mock ReputeCast with [nock](https://github.com/nock/nock). Each
request the app makes, and each answer the tests mock, is checked against
the OpenAPI document the server generates (`web/openapi.json`, through
[`../contract/openapi-contract.mjs`](../contract/openapi-contract.mjs)), so
a field the API doesn't accept fails here.

`test/live.test.mjs` runs the app against a real instance when you give it
one; it creates contacts, sends a transactional email and turns a trigger on
and off:

```bash
# a dev instance (CONVENTIONS.md › Running locally), with webhooks to
# private addresses allowed so the trigger test can receive deliveries
REPUTECAST_WEBHOOKS__ALLOW_PRIVATE_NETWORKS=true go run ./cmd/reputecast serve --auto-migrate
# connect an integration with every trigger and action on, then:
REPUTECAST_URL=http://localhost:8080 REPUTECAST_KEY=rc_live_… npm test
```

To try the app in Zapier itself before publishing, push it as a private
version (below) and invite yourself.

## Publish

Publishing happens in Zapier, outside this repository.

1. Install the CLI and sign in: `npm install -g zapier-platform-cli@19.2.1`,
   then `zapier-platform login`.
2. The first time only: `zapier-platform register "ReputeCast"` creates the
   integration and writes `.zapierapprc` (its ID; commit it). Fill in the
   integration's details in Zapier's developer platform: the logo (a
   256 × 256 PNG exported from
   [`brand/logo/reputecast-app-icon.svg`](../../brand/logo/reputecast-app-icon.svg)),
   the description, the category (Email Newsletters), the homepage
   `https://reputecast.com` and the support address.
3. Raise `version` in `package.json` and add a line to the changelog below.
4. Run the integration checks with style rules, and fix what they report:
   `zapier-platform validate`.
5. `zapier-platform push` uploads the version (it runs the build).
6. Test it in Zaps with a private invite (`zapier-platform users add …`).
7. `zapier-platform promote <version>` makes it the public version. The
   first public version goes through Zapier's review; later ones move
   existing Zaps with `zapier-platform migrate <from> <to>`.

## Changelog

- **1.2.0**: the six instant triggers (Form Submitted with a form filter),
  the five actions, typed transactional variables, idempotency keys, Zap
  names and IDs in `X-ReputeCast-Workflow`, consent kept as halted runs.

## License

AGPL-3.0-only, like ReputeCast. Zapier's directory accepts any license.
