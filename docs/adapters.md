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
- where community adapters fit.

## Built-in adapters at a glance

| Adapter | Key | Batch per call | Webhook authentication | Domain setup | Suppression sync |
|---|---|---|---|---|---|
| Amazon SES | `ses` | 50, in parallel | SNS signature (RSA, signing certificate) | API | Two-way |
| Postmark | `postmark` | 500 | Basic auth | API (account token) | Two-way, per stream |
| Postal (self-hosted) | `postal` | 1 | RSA signature (server JWKS) | Manual DNS | Local only |
| Mailgun | `mailgun` | 1 | HMAC-SHA256 over timestamp + token, account signing key | API | Two-way |
| SendGrid | `sendgrid` | 1 | ECDSA P-256 signed Event Webhook | API | Two-way |
| Resend | `resend` | 100 | Standard Webhooks (Svix) | API | Two-way |
| SparkPost | `sparkpost` | 1 | Basic auth (SparkPost doesn't sign) | API | Two-way |
| Mailjet | `mailjet` | 50 | Basic auth in the callback URL (Mailjet doesn't sign) | API | Two-way (campaign exclusions) |
| Brevo | `brevo` | 1 | Bearer token (Brevo doesn't sign) | API | Pull only |
| Azure Communication Services | `azureacs` | 1 | Entra ID token (protected webhook) or shared secret delivery header; Event Grid validation handshake | Manual DNS | Local only |
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
  <name>_test.go   fake API, contracttest.Run, adapter-specific tests
  testdata/        recorded webhook payloads
```

### The descriptor

```go
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
			Pricing:         "plan, from $10 per month",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkAllowed, Note: "Opted-in lists only…",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events: "Webhooks", DomainSetup: "API", SuppressionMode: "Two-way",
		Fields: []providers.Field{
			{Key: "api_key", Label: "API key", Secret: true, Required: true, Help: "Example › Settings › API keys."},
			{Key: "region", Label: "Region", Options: []string{"us", "eu"}},
		},
		New: New,
	}
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
requests (signing keys, OAuth tokens) goes in a package-level cache keyed by
the account. See `resend.signingSecret` and `azureacs.token`.

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
  a quota, not a bad key.
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
  but returns no id, use the ReputeCast id as the provider id.

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
  Grid), register the webhook with basic auth, a bearer token or a secret
  header carrying the provider's webhook secret (`Config.WebhookSecret`).
- Apply a timestamp window only where the provider re-signs retries
  (Standard Webhooks). SendGrid and Mailgun retry for hours with the original
  signature. There, ingest's dedupe drops redeliveries.
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
  context, and refused once tampered with.

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
| `header` | `token_header` holds the secret. |
| `none` | Nothing but the provider's unguessable webhook URL. Use it only when the provider offers none of the above. |

The secret is the `secret` template when set. Otherwise it is the
`webhook_signing_key` credential, and otherwise the webhook secret ReputeCast
generated for the provider. Point the provider's webhook at the URL shown on
the provider's card (`/hooks/httpgeneric/<token>`).

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
| `code` | The SMTP reply |
| `reason` | The diagnostic |
| `timestamp` | Unix seconds or milliseconds, or RFC 3339 |
| `url`, `user_agent` | For clicks |

A plain `bounced` is a hard bounce unless its code is a 4xx.

With `webhook_mapping: "jsonpath"`, give each path, and map the provider's
types in `types` to:

- `delivered`, `deferred`, `hard_bounce`, `soft_bounce`;
- `bounce` (hard, or soft for a 4xx code);
- `complaint`, `rejected`, `open`, `click`, `unsubscribe`;
- `ignore`.

Types you don't list fall back to the status vocabulary above and are
otherwise ignored.

## Community adapters

Community adapters are installed instance-wide from a manifest URL on the
Adapters tab (instance admins only). The tab lists each one with its
version, maintainer, declared capabilities and update status
(`installed_adapters`).

There are two kinds:

- **`kind: "http"`** is a mapping as described above, run by the generic HTTP
  adapter. `httpgeneric.ParseManifest` validates such a manifest, and
  `httpgeneric.FromManifest` builds the adapter for a provider connected
  through it.
- **Out-of-process adapters** are written with the TypeScript adapter SDK
  (`@reputecast/adapter-sdk`, `defineAdapter`). Their protocol and SDK are
  documented separately with the SDK.
