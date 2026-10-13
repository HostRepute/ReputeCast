# Example ESP adapter (template)

A ReputeCast community adapter for **Example ESP**, a fictional provider with
a typical API. Copy this directory to start an adapter for your provider:
replace the API mapping in `src/adapter.ts`, describe it in `manifest.json`,
and make the contract suite pass.

The adapter is a small Node process that speaks ReputeCast's adapter
protocol (signed JSON over HTTP). The protocol, the manifest and the SDK are
documented in [docs/adapters.md › Community adapters](../../docs/adapters.md#community-adapters).

| File | What it is |
|---|---|
| `src/adapter.ts` | The adapter: `defineAdapter({ id, capabilities, terms, send, verifyWebhook, normalizeEvent, … })` |
| `src/main.ts` | Starts the protocol server (`serve`) |
| `manifest.json` | What ReputeCast reads at install time: capabilities, terms, pricing, the connection form, network permissions |
| `reputecast.contract.ts` | The contract suite's setup: a fake of the provider's API and recorded webhooks |
| `test/` | Unit tests, the fake API and the recorded webhooks (`fixtures/webhooks.json`) |
| `Dockerfile` | The production image (non-root, Node's permission model) |

## Run it

Node 24 and pnpm.

```bash
pnpm install
cp .env.example .env        # set REPUTECAST_ADAPTER_SECRET (below)
pnpm dev                    # http://127.0.0.1:4600, restarts on changes
```

`GET /healthz` answers `{"ok":true}` for container health checks; every
other request must be signed by ReputeCast.

## Connect it to ReputeCast

On the ReputeCast server (instance admin, `--url` is where this process
listens; loopback and private addresses need
`REPUTECAST_PROVIDERS__ALLOW_PRIVATE_NETWORKS=true`):

```bash
reputecast adapter install ./manifest.json --url http://127.0.0.1:4600
```

It prints the adapter's signing secret once: put it in `.env` as
`REPUTECAST_ADAPTER_SECRET` and restart the process. Then:

```bash
reputecast adapter check example-esp
```

Now *Add provider* lists Example ESP in every workspace. The API key and
webhook signing key a person enters there are sealed in ReputeCast and sent
with each call; the adapter never stores them.

## Make it yours

1. Change `id` (it becomes `community:<id>` and appears in webhook URLs) in
   `src/adapter.ts` and `manifest.json`, and the `name`, `maintainer` and links.
2. Map your provider's API in `src/adapter.ts`:
   - `send(msg, ctx)` sends one message with `ctx.http` (or implement
     `sendBatch` for a batch API). Send every header in `msg.headers`, return
     `{ providerMessageId }`, and pass `msg.id` back on events. A 4xx from
     `ctx.http` is a per-message refusal; 401/403, 429 and 5xx fail the call
     the way ReputeCast expects.
   - `verifyWebhook(req, ctx)` checks the provider's signature on
     `req.body` (helpers: `verifyHmac`, `verifyTimestampedHmac`,
     `verifyBasicAuth`); `normalizeEvent(raw)` maps one event and keeps the
     provider's type and code.
   - Optional hooks, each needing its capability: `verify`, `discoverQuota`,
     `syncSuppressions` / `pushSuppressions`, `setupDomain` / `domainStatus`,
     `registerWebhook`, `handshake`.
3. Declare the same capabilities and terms in `manifest.json` (the server
   refuses to start when they differ), the connection fields under `config`
   (mark keys and tokens `"secret": true`), and the provider's hosts under
   `permissions.network`: `ctx.http` reaches nothing else.
4. Record a few real webhooks (with personal data removed) in
   `test/fixtures/webhooks.json` with the events they must produce, and adapt
   the fake API in `test/fake-api.ts`.

## Test it

```bash
pnpm test        # unit tests, then the contract suite (npx reputecast-adapter test)
pnpm typecheck
```

The contract suite runs the adapter over the protocol against the fake API:
the scenarios ReputeCast's own Go suite runs (send, permanent rejection,
outage, rejected credentials, each recorded webhook valid, already verified
and tampered), plus signatures, idempotent retries and declared operations.
ReputeCast's CI also runs this template through its Go runtime.

## Ship it

```bash
pnpm build && pnpm start      # dist/, with Node's permission model
docker build -t example-esp-adapter .
docker run -d --name example-esp -p 127.0.0.1:4600:4600 \
  -e REPUTECAST_ADAPTER_SECRET=rcas_… example-esp-adapter
```

Publish the image and `manifest.json` (an `https://` URL) in the adapter
registry; operators install it with `reputecast adapter install <manifest
URL> --url <where it runs>` or from the Adapters tab.

During a secret rotation (`reputecast adapter secret example-esp`), set both
secrets, new first: `REPUTECAST_ADAPTER_SECRET=rcas_new,rcas_old`. The old
one keeps working in ReputeCast for 24 hours.
