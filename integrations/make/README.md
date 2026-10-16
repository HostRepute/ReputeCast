# ReputeCast for Make

The Make custom app of [ReputeCast](../../README.md), the open-source,
self-hosted newsletter platform. Scenarios start from ReputeCast events, or
add contacts, tag them, record events and send transactional email from your
other apps. It is free, like everything in ReputeCast, and uses only the
public API ([`docs/integrations/hub.md`](../../docs/integrations/hub.md)).

Release 1.0.4 (Make app version 1). The app is declarative: JSON sections in
`src/` that Make runs, with no code of its own.

## What it can do

| Module | Kind | ReputeCast | Needs |
|---|---|---|---|
| Watch Subscribed Contacts | instant trigger | `contact.subscribed` | subscribers:view |
| Watch Unsubscribed Contacts | instant trigger | `contact.unsubscribed` | subscribers:view |
| Watch Updated Contacts | instant trigger | `contact.updated` | subscribers:view |
| Watch Form Submissions (any form, or one) | instant trigger | `form.submitted` | subscribers:view |
| Watch Sent Campaigns | instant trigger | `campaign.completed` | campaigns:view |
| Watch Transactional Bounces | instant trigger | `message.bounced` | transactional:view |
| Create or Update a Contact | action | names, custom fields, language, tags, lists with single or double opt-in | subscribers:create, subscribers:edit |
| Add a Contact to a List | action | single or double opt-in | subscribers:edit |
| Add a Tag to a Contact | action | one or more tags | subscribers:edit |
| Record an Event | action | an API event with properties; starts automations | subscribers:edit |
| Send a Transactional Email | action | a published template, a recipient, its variables and an optional language | transactional:send ⚑ |

Dropdowns and dynamic fields come from RPCs: lists, published templates,
signup forms, the workspace's custom fields and the chosen template's
variables.

## Connect it

1. In ReputeCast, open **Settings › Integrations › Make › Connect**. Choose
   what Make may do (Send a transactional email starts off: it needs the
   sensitive transactional:send), then **Create key**. ReputeCast shows the
   key once.
2. In a scenario, add a ReputeCast module and **Create a connection** with:
   - **Instance URL**: your ReputeCast address, for example `https://reputecast.acme.io` (no trailing slash).
   - **Key**: the key from step 1 (`rc_live_…`).

   Saving tests the connection (`GET /api/v1/integration`): ReputeCast's
   connect dialog shows "Make called at 14:31 · Test connection · 200", and
   the connection keeps the key's workspace.

Make's servers call your instance, and ReputeCast calls Make's webhook
addresses for the triggers, so the instance must be reachable from the
internet over HTTPS. An instance on a private network can't use Make; use n8n
on the same network instead.

## Good to know

- **Triggers.** Creating a trigger's webhook in Make registers it in
  ReputeCast (shown read-only as "Managed by Make" on Settings › Webhooks);
  deleting it in Make removes it. If the webhook is deleted in Make without
  detaching, ReputeCast removes it at the next delivery, when Make answers
  410 Gone. Watch Form Submissions takes an optional form when you create
  its webhook.
- **Signatures.** ReputeCast signs every delivery (`X-ReputeCast-Signature`),
  but Make doesn't give custom apps a webhook's raw body, so the app can't
  check it. A Make webhook's address is a long random URL: keep it private.
- **Scenario names.** Every call carries `X-ReputeCast-Workflow` with the
  scenario's name and ID where Make gives them to custom apps (`scenario.name`,
  `scenario.id`), so ReputeCast lists the scenarios that use it. Calls without
  them are listed as "Unnamed scenario".
- **Consent is kept.** ReputeCast never adds someone back to a list they
  left, never adds a suppressed address and doesn't send transactional email
  to one. Those bundles end with a **DataError** in ReputeCast's words ("rob@example.net
  left Beta Program on 2 Oct…"). Add an **Ignore** or **Resume** error
  handler to the module so the rest of the scenario keeps running, or turn on
  "Allow storing of incomplete executions".
- **Errors.** A revoked or expired key stops the scenario with
  InvalidAccessTokenError: create a new key in ReputeCast (Reconnect) and
  update the connection. A refused operation or a key whose owner left stops
  it with InvalidConfigurationError and ReputeCast's reason ("Change what
  Make can do in Settings › Integrations", "…take it over…"). A rotated key
  keeps working for 24 hours.
- **Idempotency.** Send a Transactional Email takes an optional Idempotency
  key, such as the triggering event's ID: ReputeCast sends nothing new when
  the same value comes again within 24 hours (the key sent is
  `make-<scenario ID>-<value>`).

## Layout

```
app.json                  the app and its components: modules (label, type, connection, webhook), webhooks, RPCs
src/base.imljson          base URL, the key, User-Agent, X-ReputeCast-Workflow, error types
src/groups.imljson        the module groups in the scenario builder
src/readme.md             the app's documentation shown in Make
src/connections/reputecast/{parameters,api}.imljson
src/webhooks/<name>/{parameters,api,attach,detach}.imljson
src/modules/<name>/{expect,parameters,api,interface,samples}.imljson
src/rpcs/<name>/{parameters,api}.imljson
scripts/validate.mjs      the checks (npm run lint)
scripts/iml.mjs           the IML the app uses, evaluated for the tests
scripts/build.mjs         dist/reputecast-make.json: everything in one file
```

Each file is one section of Make's app editor: `api` is Communication,
`expect` Mappable parameters, `parameters` Static parameters, `interface`
Interface, `samples` Samples; a webhook's `attach` and `detach` are its
Attach and Detach.

## Develop and test

No dependencies: Node 24 is enough.

```bash
cd integrations/make
npm run lint    # validates app.json and every section (scripts/validate.mjs)
npm test        # runs the sections through the IML evaluator against web/openapi.json
npm run build   # dist/reputecast-make.json
```

`scripts/validate.mjs` checks that every component has its sections and
only directives Make knows, every `{{ }}` parses and uses declared
parameters, `rpc://` and webhook references exist, the groups list each
module once, samples fit the interfaces, and every communication section
calls an operation of `web/openapi.json` that the hub gives integration keys.
The tests go further: they evaluate each section with realistic parameters
and check the request it would send, and the answers they mock, against the
OpenAPI document (through
[`../contract/openapi-contract.mjs`](../contract/openapi-contract.mjs)).

`test/live.test.mjs` runs the sections against a real instance through the
same evaluator when you give it one:

```bash
REPUTECAST_URL=http://localhost:8080 REPUTECAST_KEY=rc_live_… npm test
```

## Publish

Publishing happens in Make, outside this repository. A custom app is built
in Make's developer platform, then shared by invite link or reviewed and
published as a public app.

1. Raise `release` in `app.json` and `version` in `package.json` together
   (the validator checks `User-Agent` matches), and add a line to the
   changelog below. Make's own app version (`version` in `app.json`) goes up
   only for changes that break existing scenarios.
2. `npm run lint && npm test && npm run build`.
3. In Make, open **Custom apps › Create a new app** (the first time): name
   `reputecast`, label ReputeCast, theme `#0F766E`, language English, and
   the icon (a 512 × 512 PNG exported from
   [`brand/logo/reputecast-app-icon.svg`](../../brand/logo/reputecast-app-icon.svg)).
4. Add the components as `app.json` lists them, and paste each file into its
   tab: Base (`src/base.imljson`), Groups, Readme; the connection (type API
   key / Basic) with its Communication and Parameters; the six webhooks
   (type Dedicated, attached) with Communication, Parameters, Attach and
   Detach; the five RPCs; the eleven modules, each with the type, connection
   and, for instant triggers, webhook `app.json` gives it.

   With the Make CLI (`npm install -g @makehq/cli`, `make-cli login`), the
   same sections go up with `make-cli sdk-apps set-section`,
   `sdk-connections set-section`, `sdk-webhooks set-section`,
   `sdk-rpcs set-section` and `sdk-modules set-section`, after creating the
   components with the matching `create` commands; or clone the app into a
   local workspace with Make's VS Code extension (Make Apps Editor) and
   deploy from there.
5. Test the scenarios in Make with the app private, then request the review
   for a public app (Make's app review checks the same things as
   `npm run lint`, and the documentation in `src/readme.md`).

## Changelog

- **1.0.4**: the six instant triggers (Watch Form Submissions with a form
  filter), the five actions, dynamic fields for custom fields and template
  variables, ReputeCast's words and Make's error types for every refusal,
  scenario names and IDs in `X-ReputeCast-Workflow`.

## License

AGPL-3.0-only, like ReputeCast.
