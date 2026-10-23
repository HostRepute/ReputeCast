# Sending adapters

ReputeCast sends through pluggable **adapters**. An adapter is the code that
talks to one sending provider: it sends mail, verifies and normalizes the
provider's webhooks, and declares what the provider can do. Core code never
imports an adapter. It finds adapters in a registry and talks to them only
through the contract in `internal/providers`.

This guide covers three things:

- how a built-in adapter is written, tested against the shared contract
  suite, and registered;
- how to connect a provider without writing Go, through the **generic HTTP
  adapter**;
- **community adapters**: HTTP mappings installed for the whole instance, and
  adapters that run in their own process, written with the TypeScript SDK
  and called over the adapter protocol.

## Built-in adapters at a glance

| Adapter | Key | Batch per call | Webhook authentication | Domain setup | Suppression sync |
|---|---|---|---|---|---|
| Amazon SES | `ses` | 50, in parallel | SNS signature (RSA, signing certificate) | API | Two-way |
| Postmark | `postmark` | 500 | Basic auth | API (account token) | Two-way, per stream |
| Postal (self-hosted) | `postal` | 1 | RSA signature (server JWKS) | Manual DNS | Local only |
| Mailgun | `mailgun` | 1 | HMAC-SHA256 over timestamp + token, account signing key; 12-hour window, each token bound to its event | API | Two-way |
| SendGrid | `sendgrid` | 1 | ECDSA P-256 signed Event Webhook; 26-hour window | API | Two-way |
| Resend | `resend` | 100 | Standard Webhooks (Svix) | API | Two-way |
| SparkPost | `sparkpost` | 1 | Basic auth (SparkPost doesn't sign) | API | Two-way |
| Mailjet | `mailjet` | 50 | Basic auth in the callback URL (Mailjet doesn't sign) | API | Two-way (campaign exclusions) |
| Brevo | `brevo` | 1 | Bearer token (Brevo doesn't sign) | API | Pull only |
| Azure Communication Services | `azureacs` | 1 | Entra ID token (protected webhook) or shared secret delivery header (`Authorization: Bearer`); Event Grid validation handshake; Event Grid event schema only | Manual DNS | Local only |
| Generic SMTP | `smtp` | 25 per connection | Bounce mailbox (IMAP/POP3) | Manual DNS | Local only |
| Generic HTTP adapter | `httpgeneric` | 1 | Configurable: HMAC, Standard Webhooks, basic auth, header secret | Manual DNS | Local only |

"Batch per call" is how many distinct, personalized messages go in one API
call. Mailgun, SendGrid, SparkPost and Brevo can batch recipients, but their
recipients share one body. ReputeCast renders every message for its
recipient (tracking links, unsubscribe URL, Liquid fields), so those adapters
make one call per message and run several calls at a time.

Each adapter's package comment quotes the provider's acceptable-use policy,
links to it, and gives the date it was checked. That source backs the "bulk
marketing allowed" badge.

## Writing a built-in adapter

### The contract

`internal/providers/provider.go` defines the contract. It is read-only for
adapter authors. Every adapter implements `providers.Provider`:

| Method | Does |
|---|---|
| `Adapter() string` | The registry key (`"mailgun"`). |
| `Capabilities() Capabilities` | The badges: bulk policy, webhooks, domain API, suppression sync, batch size, dedicated IPs, sandbox, quota and rate discovery, pricing. |
| `Send(ctx, msgs) ([]SendResult, error)` | Sends at most `BatchSize` messages. Returns one result per message, in input order. |
| `ParseWebhook(ctx, r, body) ([]Event, error)` | Verifies the request and maps its payload to normalized events. |

`internal/providers/optional.go` defines optional interfaces. Implement every
one the provider's API really supports. Core code type-asserts before using
them, and the screens explain the missing ones ("webhooks are set up in X
itself"):

| Interface | Used by |
|---|---|
| `Verifier` / `QuotaReporter` | Connect step and the health job (every 5 minutes, every minute while a provider is down). `Quota` wins when both exist. |
| `ReputationReporter` | The provider's own bounce and complaint rates, which its review thresholds apply to. |
| `DomainManager` | DNS records with live status for the domain step and Deliverability. |
| `WebhookRegistrar`, `WebhookInspector`, `WebhookTester` | "Register webhooks", the webhooks-silent diagnosis, "Send test event". |
| `WebhookVerifier`, `WebhookHandshaker` | Ingest verifies before storing. Handshakes (SNS subscriptions, Event Grid validation) are answered and not stored. |
| `WebhookSecrets` | Names the headers that carry a shared webhook secret as it is. Ingest leaves them out of the stored raw event, as it does `Authorization`. |
| `SuppressionSyncer` | Two-way sync with the shared suppression list. If the provider can list suppressions but not add them, return `providers.ErrNotSupported` from `AddSuppressions` and the sync switches to pull only. |
| `TrackingManager` | The provider-side tracking check (no double tracking). |
| `EventBackfiller` | Fetches message status when webhooks went silent. |
| `StatusPage`, `StatusReporter` | The status button and incident text during an outage. |
| `BroadcastStreamCreator` | For providers whose terms want a separate stream for bulk mail (Postmark). |

### Package layout

Follow the existing adapters (`internal/providers/resend` is a compact
example):

```
internal/providers/<name>/
  <name>.go        package comment with the terms, Descriptor, Settings, Credentials, New, Send
  webhook.go       VerifyWebhook, ParseWebhook, normalization
  account.go       Verify/Quota, domains, webhook registration, suppressions, tracking, backfill
  lang.go          the message catalog and its messages (Text for people)
  messages/        en.json, the English people read, and vi.json, its translation
  <name>_test.go   fake API, contracttest.Run, adapter-specific tests
  testdata/        recorded webhook payloads
```

### The descriptor

```go
//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.example")

func init() { providers.Register(Descriptor()) }

func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: "example",
		Name:    "Example ESP",
		Kind:    "api", // api, smtp, self_hosted, http
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkAllowed, // cite the AUP in the package comment
			Webhooks:        "Registered",
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       1,
			Sandbox:         "mode",
			QuotaDiscovery:  true,
			// Pricing and DedicatedIPs are in messages/en.json.
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkAllowed,
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Fields: []providers.Field{
			{Key: "api_key", Secret: true, Required: true},
			{Key: "region", Options: []string{"us", "eu"}},
		},
		New: New,
	}.WithMessages(msgs)
}
```

with its words in `messages/en.json`, a `note` where a translator needs one:

```json
{
  "pricing": { "message": "plan, from $10 per month", "note": "on a capability chip: Pricing · …" },
  "terms.0.note": { "message": "Opted-in lists only…", "note": "what the terms allow, on Routing" },
  "events": "Webhooks",
  "domainSetup": "API",
  "suppressionMode": "Two-way",
  "field.api_key.label": "API key",
  "field.api_key.help": { "message": "Example › Settings › API keys.", "note": "Example's own menus" },
  "field.region.label": "Region"
}
```

The badges must be true for the provider today:

- **Bulk:** `BulkAllowed`, `BulkBroadcastStreams` (only on a dedicated stream),
  `BulkForbidden` or `BulkYourPolicy` (self-hosted). Read the provider's
  current acceptable-use policy. Quote it in the package comment with the URL
  and the date you checked. Put the conditions that matter (opt-in, limits) in
  `Terms.Note`; the routing screen shows that note.
- **Sandbox:** one of `detection`, `test token`, `mode` or `test addresses`.
  Leave it empty when there is none.
- **Quota and rate discovery:** set them only when an API call reports them.
  An allowance the user enters at connect time can supply the maximum.
- **Pricing and dedicated IPs:** short human summaries. Prefer a plan name to
  a price that will go stale.
- **Tracking:** set `RewritesLinks` / `AddsOpenPixel` when the provider tracks
  by default and the adapter can't turn it off per message.

### Text for people

What an adapter tells people reads in each reader's language: server text
follows the reader's app language (`i18n.LangFrom(ctx)`), and API clients
that send no `Accept-Language` read English. An adapter writes its words once,
in English, in its own catalog (`messages/en.json`, loaded with
`i18n.MustLoad(messagesFS, "providers.<name>")`), and its translations beside
it (`messages/vi.json`, with the terms of `docs/i18n/glossary-vi.md`).
Catalogs are global, so core code reads a kept message without importing the
adapter.

- **The descriptor's words.** `WithMessages(msgs)` fills the descriptor from
  well-known keys: `field.<key>.label` and `field.<key>.help`,
  `terms.<n>.variant` and `terms.<n>.note` (the nth Terms, from 0), `pricing`,
  `dedicatedIps`, and the Adapters tab's `events`, `domainSetup` and
  `suppressionMode`. It writes the English where the descriptor leaves a text
  empty and keeps the message beside it (`LabelText`, `NoteText`…). Readers
  call `d.In(l)` or `providers.TextIn(english, text, l)`; logic keeps
  comparing the English.
- **What the adapter writes at runtime** keeps its message beside the
  English: `Quota.AccountText` (the account's standing),
  `ConnectionReport.SummaryText`, `ProvisionStep.LabelText`,
  `ProvisionResult.DoneText`, `Tracking.DetailText` and `ScopeText`,
  `ServiceStatus.SummaryText`, and `CallError.Text` for an error the adapter
  words itself (`providers.ErrorIn` and `MessageIn` read it). Return the
  English as before and the message with `msg.Keep(args)`; most adapters
  have a `say(m, args)` helper that returns both. Pass numbers as numbers
  and times as `time.Time`, so `{n, number}` and `{at, time}` follow the
  reader's region format.
- **Names stay as they are:** the provider's name and its own words for its
  plans, keys, menus and stream types ("Server API token", "luồng
  Broadcast"), and its answers (error codes, API messages), event names,
  logs and internal errors.
- **Stored text.** Code that keeps an adapter's text for later readers (a
  provider's health, a setup's state, an incident) stores the English and
  the message. A row written before its message was kept reads in English.

Community adapters (the generic HTTP adapter's mappings and remote adapters)
have no catalog. Their manifest's words (the name, the terms note, the
pricing, the connection form) read as their author wrote them, in English
for everyone. What ReputeCast says about them (the capability labels, the
Adapters tab's words, the runtime's errors) reads in each reader's language.

The contract suite checks it (its `Texts` subtest): a built-in adapter of
kind `api`, `smtp`, `self_hosted` or `http` names its catalog, every text it
declares has a message whose English is that text, and every message is
translated into each app language. `contracttest.CheckText` checks a text
the adapter writes at runtime.

SMS adapters (`internal/sms/<name>`) work the same way: their catalog's
namespace is `sms.<name>`, `Manifest.WithMessages` reads the keys `summary`,
`badge.<capability>.note`, `field.<key>.label` and `field.<key>.help`, the
SMS screens show `Manifest.In(l)`, and `internal/sms/contracttest` checks
the texts.

### Settings and credentials

`providers.Config.Settings` holds the non-secret fields (the
`providers.config` column). `Config.Credentials` holds the decrypted secrets,
as a flat JSON object of strings whose keys are the `Secret` fields' keys.
Credentials are write-only: they're encrypted at rest, never returned by the
API, and replaced one key at a time. The contract suite fails a field whose
key looks secret (`password`, `token`, `secret`, `…api_key`) but isn't marked
`Secret`.

`New` must not do network I/O, and it must not fail because an optional
value is missing. Report missing credentials from the first call instead.
The factory builds an adapter per request, so anything worth keeping between
requests goes in a package-level cache keyed by the account. Signing
material read from the provider's API (a webhook secret, a public key, a key
set) goes in a `webhookauth.Material`. It reads the material again when a
signature fails (at most once a minute) and every hour, so a key the
provider rotated out stops verifying. See `resend.secrets`. For OAuth tokens,
see `azureacs.token`.

### Sending

- **Errors:** return a per-message `SendError` for a refusal that concerns
  one message (an invalid address, an unverified sender). Return an error
  for a failure that concerns the account or the service, wrapping
  `providers.ErrUnauthorized` (rejected credentials) or
  `providers.ErrUnavailable` (outage). `apiclient` does this mapping:
  `Do` for JSON calls, `Exec` for form or multipart bodies, signed requests
  and response headers. 401 and 403 become rejected credentials, 429
  throttling, 408 and 5xx outages. Adjust that where the provider means
  something else. SendGrid answers 401 when you're out of credits, which is
  a quota, not a bad key. `apiclient.MessageError` turns the remaining 4xx
  answers into the message's permanent `SendError`.
- **One call per message:** use `apiclient.Fanout`. It keeps results in
  order. After an account-level failure it starts no new call, and the
  messages it didn't send get a temporary error so the engine retries them.
- **Echo the ReputeCast message id** in whatever the provider returns on
  events (custom args, tags, metadata, an operation id) so events match even
  when provider ids are shared or missing. Send the `X-ReputeCast-Message`
  header too.
- **Idempotency:** where the provider supports it, derive the idempotency key
  from the message id, so a retry after a timeout never sends twice.
- **Tracking:** turn the provider's open and click tracking off on every
  message. ReputeCast tracks itself.
- **Never mark an accepted message as failed.** If the provider accepts it
  but returns no id, use the ReputeCast id as the provider id. The same goes
  for an accepted batch whose answer can't be matched to its messages:
  sending it again could deliver it twice.

### Webhooks

- `VerifyWebhook` must return an error wrapping `providers.ErrBadSignature`
  for every failure, including "couldn't fetch the key". Ingest stores any
  other error as verified.
- `ParseWebhook` verifies unless `providers.SignatureVerified(ctx)` is set.
  The normalization job sets it when it re-parses a stored payload, which no
  longer has the original URL or the Authorization header.
- Use the scheme the provider signs with. `internal/providers/webhookauth`
  has Standard Webhooks (Svix), HMAC helpers and constant-time shared-secret
  checks. Where the provider doesn't sign (SparkPost, Mailjet, Brevo, Event
  Grid), register the webhook with basic auth or a bearer token carrying the
  provider's webhook secret (`Config.WebhookSecret`). Both travel in
  `Authorization`, which ingest never stores. A secret in any other header
  must be named through `WebhookSecrets`.
- Bound the signed timestamp. Standard Webhooks re-signs every attempt, so
  five minutes is enough (`webhookauth.Fresh`). Where retries may carry the
  first attempt's signature for hours, the window spans the retries
  (`webhookauth.Within`): 26 hours for SendGrid, 12 for Mailgun. Mailgun
  signs a token, not the event, so `webhookauth.Replay` also ties each token
  to the event it came with. A redelivery of the same event passes, and
  ingest's dedupe drops it.
- **Normalize:**
  - Map every delivery-relevant event to `delivered`, `deferred`,
    `hard_bounce`, `soft_bounce`, `complaint`, `rejected` or `unsubscribe`.
  - Keep the provider's type in `OriginalType` ("bounce · class 10 Invalid
    Recipient"), its code in `OriginalCode` ("550 5.1.1", via `bounce.Split`),
    and the rest in `Diagnostic`.
  - Set `BounceClass` with `bounce.Classify`.
  - Lowercase the recipient.
  - Keep the event's raw JSON in `Raw`.
  - Return no event for things that aren't outcomes (sent, processed,
    auto-replies, test pings).
- **Don't suppress on a guess.** A permanent failure caused by the sender (a
  spam or reputation block, retries running out) is a soft bounce. Only an
  address problem is a hard bounce, because hard bounces and complaints go on
  the shared suppression list.

### Testing against the contract suite

Every adapter passes `internal/providers/contracttest`. The suite checks:

- registration;
- capabilities;
- Send, a permanent rejection, an outage and rejected credentials, all
  against your fake API;
- each recorded webhook, verified, normalized, re-parsed with a verified
  context, and refused once tampered with;
- the text people read: its messages, and their translations (Text for
  people).

```go
func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Adapter: example.Adapter,
		API:     fakeAPI, // reads env.Mode(): "", contracttest.ModeOutage, contracttest.ModeAuth
		New:     newProvider,
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			// check env.Requests(): headers, the echoed message id, tracking off…
		},
		Webhooks: []contracttest.Webhook{{
			Name:    "HardBounce",
			Request: signed("testdata/bounce.json"), // sign at test time with a test key
			Want:    []providers.Event{{Type: providers.EventHardBounce, Recipient: "john@example.com", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown" /* … */}},
		}},
		// Tamper defaults to flipping a body byte; override it when the
		// signature doesn't cover the body (Mailgun) or auth is a header.
	})
}
```

Test rules:

- **No network.** Run a fake of the provider's API with `contracttest.NewEnv`
  or the suite's `API`. Record webhook payloads from the provider's
  documentation as fixtures, and sign them in the test with a test key.
  Where the provider publishes a signature test vector, test against it, as
  `sendgrid.TestOfficialSignature` does.
- Test the optional interfaces, the error mapping and every quirk you code
  around: ids above 2^53, a field that is a string or an array, a 400 that
  carries per-message results.
- Run the tests with:

```sh
REPUTECAST_TEST_DATABASE_URL=postgres://reputecast:reputecast@localhost:5433/postgres?sslmode=disable go test ./internal/providers/...
~/go/bin/golangci-lint run ./internal/providers/...
```

### Registering

1. Register from `init` with `providers.Register(Descriptor())`. A duplicate
   key panics.
2. Compile the adapter into the binary with a blank import in
   `internal/app/wire_adapters.go`.
3. The setup catalog, the connect wizard and the Adapters tab read the
   registry, so the adapter appears there. The Adapters tab orders built-in
   adapters by `adapterOrder` in `internal/providers/manage/adapters.go`.
4. Built-in adapters carry the ReputeCast version (`version.Version`) and
   the maintainer "HostRepute".

## The generic HTTP adapter

The generic HTTP adapter (`httpgeneric`) sends through any HTTP API from a
**mapping**, so you can connect a provider without writing Go:

- the send request: method, URL, headers, and a JSON body template;
- where the response carries the message id;
- how the provider's webhooks are authenticated and read.

To create one, an instance admin either fills in **Sending providers ›
Adapters › Generic HTTP adapter** or installs a community manifest of kind
`http`. Workspaces then connect providers through it with their own
credentials. A provider can also carry its mapping in its own settings
(`{"http": {…}}`).

The form covers the send request (method and URL, headers, body template),
the message id path, the webhook mapping (the `status` vocabulary, or a
custom JSONPath map: the events, and the type, recipient and message id of
one event) and the webhook authentication with its header. It is checked
like a manifest before it is saved. Everything else below (event type
names, timestamps, algorithms, a health check, the connection form, terms)
needs a manifest.

### Manifest

The mapping is the manifest's `http` object. The Adapters tab saves it in
`installed_adapters.manifest`; community manifests ship the same object.

```json
{
  "name": "Example ESP",
  "version": "1.0.0",
  "maintainer": "you@example.com",
  "kind": "http",
  "terms": {"bulk": "allowed", "note": "Opted-in lists only.", "url": "https://example-esp.com/aup"},
  "pricing": "$0.80 per 1,000",
  "http": {
    "method": "POST",
    "url": "https://api.example-esp.com/v1/{{ settings.account }}/messages",
    "headers": {"Authorization": "Bearer {{ secrets.api_key }}"},
    "body": "{\"from\": \"{{ sender }}\", \"to\": [{\"email\": \"{{ recipient }}\"}], \"subject\": \"{{ subject }}\", \"html\": \"{{ html }}\", \"text\": \"{{ text }}\", \"headers\": {{ headers_json }}, \"metadata\": {\"reputecast_id\": \"{{ message_id }}\"}}",
    "message_id_path": "$.data.id",
    "error_code_path": "$.error.code",
    "error_message_path": "$.error.message",
    "check": {"url": "https://api.example-esp.com/v1/{{ settings.account }}/me"},
    "webhook_mapping": "jsonpath",
    "webhook": {
      "auth": "hmac",
      "signature_header": "X-Example-Signature",
      "signature_prefix": "sha256=",
      "timestamp_header": "X-Example-Timestamp",
      "secret": "{{ secrets.webhook_signing_key }}",
      "events_path": "$.events[*]",
      "type_path": "$.event",
      "recipient_path": "$.recipient",
      "message_id_path": "$.message_id",
      "reputecast_id_path": "$.metadata.reputecast_id",
      "code_path": "$.smtp.code",
      "diagnostic_path": "$.smtp.message",
      "time_path": "$.ts",
      "types": {"delivered": "delivered", "bounce_permanent": "hard_bounce", "bounce_temporary": "soft_bounce", "complaint": "complaint", "sent": "ignore"}
    }
  }
}
```

`terms.bulk` is `allowed`, `broadcast_streams_only`, `forbidden` or
`your_policy`. It defaults to `your_policy`: ReputeCast can't read a mapped
provider's terms, so the routing screen asks you to check them.

The manifest's words (`name`, `terms.note`, `pricing`, the connection form's
labels) are shown as you wrote them, in every language. Without a
`terms.note`, ReputeCast's own note reads in each reader's language, like the
rest of what it says about the mapping (Text for people).

### Mapping fields

**Send request**

| Field | Meaning |
|---|---|
| `method` | `POST` (default), `PUT` or `PATCH`. |
| `url` | `https://` URL. The host must be written out; `{{ }}` fields may appear only in the path and query. `http://` is accepted for a loopback test server only. |
| `headers` | Header templates. `Content-Type` defaults to `application/json`. `Host`, `Content-Length`, `Transfer-Encoding` and `Connection` can't be set. |
| `body` | JSON body template. It must be valid JSON once its fields are filled in; saving checks this against a sample message. |

**Response**

| Field | Meaning |
|---|---|
| `message_id_path` | JSONPath of the provider's message id in a 2xx response. |
| `message_id_header` | The response header that carries it instead. With neither, the ReputeCast id stands in for it. |
| `error_code_path` / `error_message_path` | Where a failed response puts the provider's error code and message. |

**Health check and connection form**

| Field | Meaning |
|---|---|
| `check.url` (+ `check.headers`) | Optional GET that answers 2xx while the credentials are good. It is used at connect time and by the health job. Without it, health comes from send results alone. |
| `fields` | Extra connection fields. Secret ones are read as `secrets.<key>`, the others as `settings.<key>`. |

**Webhooks**

| Field | Meaning |
|---|---|
| `webhook_mapping` | `status` (default) or `jsonpath`. |
| `webhook.auth` | How a webhook request proves it comes from the provider. See below. |
| `webhook.*_path` | Where each event field is (JSONPath). |
| `webhook.types` | The provider's event types mapped to ReputeCast's. |

Results: 2xx is accepted. 401 and 403 are rejected credentials, 429
throttling, and 5xx an outage, which pauses the route. Any other 4xx fails
that message only.

### Template fields

Templates use Liquid-style output tags: `{{ name }}`, `{{ name | filter: arg }}`.
There are no `{% %}` control-flow tags; a mapping describes one fixed request
shape.

| Field | Value |
|---|---|
| `sender` | From as `"Name" <address>` (the address alone without a name) |
| `sender_email`, `sender_name` | From's parts |
| `recipient`, `recipient_name`, `recipient_formatted` | To's address, name, and `"Name" <address>` |
| `reply_to`, `reply_to_name` | Reply-To (empty when none) |
| `subject`, `html`, `text` | The rendered message |
| `headers` / `headers_json` | Headers as an object: List-Unsubscribe, List-Unsubscribe-Post, Feedback-ID, X-ReputeCast-Message |
| `headers_list` | The same as `[{"name": …, "value": …}]` |
| `tags` | The message's tags as an object |
| `message_id` | The ReputeCast message id. Pass it to the provider so webhooks can echo it. |
| `stream` | `broadcast`, `automation` or `system` |
| `settings.<key>`, `secrets.<key>` | The provider's settings and credentials. `secrets.api_key` and `secrets.webhook_signing_key` are always available. |

Filters: `default`, `prepend`, `append`, `base64_encode`, `url_encode`,
`downcase`, `upcase`, `strip`, `json`. Arguments are quoted strings or
fields. For example, `{{ "api:" | append: secrets.api_key | base64_encode }}`
builds a basic-auth value.

Values are escaped for where they appear, so the data can't break the
request:

- **Inside a JSON string** (`"subject": "{{ subject }}"`), the value is
  JSON-escaped.
- **Outside a string** (`"headers": {{ headers_json }}`), it is JSON-encoded:
  objects stay objects and a missing value becomes `null`.
- **In the URL**, values are path-escaped, or query-escaped after `?`.
- **In headers**, line breaks are removed.

To send a literal `{{` inside a JSON string (an API's own placeholders),
write `\{{`. The backslash only marks it and is dropped, so
`"Hi \{{first_name}}"` sends `"Hi {{first_name}}"`. `{{ '{{' }}` does the
same anywhere.

**Test with a sample message** renders the request with secrets shown as
`••••` (`Provider.Preview`).

### JSONPath

Mappings use a subset of JSONPath:

- `$` — the root;
- `.name`, `['name']`, `["name"]` — a member;
- `[0]` — an array index;
- `[*]` and `.*` — every element.

Filters and recursive descent aren't supported.

### Webhook authentication

| `auth` | The request is accepted when |
|---|---|
| `hmac` (default) | `signature_header` (default `X-Signature`) holds the HMAC of `signed_payload` (default `{body}`, or `{timestamp}.{body}` when `timestamp_header` is set). `algorithm` is `sha256` (default), `sha1` or `sha512`. `encoding` is `hex` (default) or `base64`, and `signature_prefix` (`sha256=`) is stripped. With a timestamp, requests older than five minutes are refused. |
| `standard` | It carries a valid Standard Webhooks / Svix signature (`webhook-id`, `webhook-timestamp`, `webhook-signature`) made with a `whsec_` secret. |
| `basic` | Its basic-auth user is `username` (default `reputecast`) and its password is the secret. |
| `header` | `token_header` holds the secret. That header is never stored with the raw event. |
| `none` | Nothing but the provider's unguessable webhook URL. Use it only when the provider offers none of the above. |

The secret is the `secret` template when set. Otherwise it is the
`webhook_signing_key` credential, and otherwise the webhook secret ReputeCast
generated for the provider (the connection step issues a new one on request
and shows it once; stored secrets are never shown). Point the provider's
webhook at the URL shown on
the provider's card (`/hooks/httpgeneric/<token>`, or
`/hooks/community:<name>/<token>` for a provider connected through an
installed HTTP adapter).

### Webhook payloads

`events_path` selects the events (default `$`: the payload is one event or
an array of events).

With `webhook_mapping: "status"`, each event has these fields:

| Field | Meaning |
|---|---|
| `status` | `delivered`, `deferred`, `bounced`, `soft_bounced`, `complained`, `rejected`, `opened`, `clicked` or `unsubscribed` |
| `recipient` (or `email`) | The recipient |
| `message_id` | The provider's message id |
| `reputecast_id` | The echoed ReputeCast id |
| `code` | The SMTP reply, or the provider's own code (`invalid_recipient`) |
| `reason` | The diagnostic |
| `timestamp` | Unix seconds or milliseconds, or RFC 3339 |
| `url`, `user_agent` | For clicks |

A plain `bounced` is a hard bounce unless its code says otherwise:

- An SMTP reply decides by its class: 5xx is hard and 4xx is soft.
- A provider's own code is hard when its words name an unknown mailbox or
  domain (`invalid_recipient`, `user-unknown`, `domain_not_found`) or say
  `hard` or `permanent`. Any other code is soft, so nothing is suppressed on
  a guess.
- A block caused by the sender (spam, reputation, policy) is always soft,
  whatever the code or the mapped type.

With `webhook_mapping: "jsonpath"`, give each path, and map the provider's
types in `types` to:

- `delivered`, `deferred`, `hard_bounce`, `soft_bounce`;
- `bounce` (decided by its code, as above);
- `complaint`, `rejected`, `open`, `click`, `unsubscribe`;
- `ignore`.

Types you don't list fall back to the status vocabulary above and are
otherwise ignored.

## Community adapters

A community adapter connects a sending provider that has no built-in
adapter. Instance admins install it for the whole instance, from a manifest
URL on the Adapters tab or with `reputecast adapter install`; the tab lists
each one with its version, maintainer, declared capabilities and update
status (`installed_adapters`). There are two kinds:

- **`kind: "http"`**: a mapping as described [above](#the-generic-http-adapter),
  saved from the Adapters tab or shipped as a manifest, run by the generic
  HTTP adapter (`httpgeneric.FromManifest`). It is registered as
  `community:<name>`, the name lowercased with dashes (`community:example-esp`
  for "Example ESP").
- **`kind: "remote"`**: a program that runs in its own process and speaks the
  adapter protocol below. It is registered as `community:<id>`.

Both appear in the provider catalog with their declared capabilities, terms
and connection form, so *Add provider* connects them like built-in adapters.
The rest of this section is about remote adapters.

A remote adapter is usually written in TypeScript with the adapter SDK
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

Inside ReputeCast an installed adapter, of either kind, is registered in the
provider registry (`community:<id>`, for example `community:example-esp`)
at startup and whenever it is installed, updated or removed. Every process
(`--role=web`, `worker`, `ingest`) re-reads the installed adapters every 15
seconds, so a change made in one process reaches the others. Code that lists
`installed_adapters` rows finds each one's registry entry, with its
capabilities, terms and connection fields, through
`providers.Lookup(providers.InstalledKey(kind, name, manifest))`; a row that
isn't registered can't be connected from this build.

### Build an adapter

```bash
cp -r sdk/adapter-template my-esp-adapter && cd my-esp-adapter
pnpm install
pnpm dev                       # serves the protocol on http://127.0.0.1:4600
pnpm test                      # unit tests + the contract suite (npx reputecast-adapter test)
```

A copy outside this repository installs `@reputecast/adapter-sdk` from npm.
Inside it, run `pnpm install` in `sdk/` instead: the template is linked to
the workspace's SDK, which the install builds.

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
audit entry (`adapter.runtime_update`, `adapter.secret_issue`, `adapter.check`).

**Addresses.** Calls to the adapter go through the same SSRF-guarded client
as every provider adapter (`internal/netguard`): link-local and cloud
metadata addresses are always refused, redirects are never followed, and
loopback or private (RFC 1918) addresses are refused unless the operator
allows them, with the setting that also covers provider URLs:

```yaml
providers:
  allow_private_networks: true   # REPUTECAST_PROVIDERS__ALLOW_PRIVATE_NETWORKS=true
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

The manifest's texts (`name`, `description`, `terms.note`, `pricing`,
`dedicatedIps`, the `config` labels and help) are shown as written, in
English for every reader: a community adapter has no message catalog. The
badges, the Adapters tab's words and the runtime's errors are ReputeCast's
and read in each reader's language (Text for people).

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
`reputecast adapter check` and the runtime API's check, which go through
even while the circuit breaker is open (a success closes it).

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
be reached, ReputeCast answers the provider `429` with `Retry-After` so it
retries later (like any transient ingest failure); an unverified request is
never stored as verified.

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
- **An address only the operator sets**, called through the SSRF-guarded
  client provider adapters use: loopback and private addresses only with
  `providers.allow_private_networks`, metadata and link-local addresses
  never, no redirects.
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
