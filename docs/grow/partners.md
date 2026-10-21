# Partner links between ReputeCast instances

Protocol version 1. Implemented in `internal/recommendations` (the endpoints
and the flows) and `internal/recommendations/partnerproto` (messages,
signatures, codes and the key agreement).

A partner link lets one newsletter (the **recommending** side, "Acme")
recommend another newsletter that runs on a different ReputeCast instance
(the **partner**, "Ops Weekly"). Readers press Subscribe on Acme's pages,
Acme's server sends Ops Weekly's server a signed subscribe request, and Ops
Weekly runs its own double opt-in. Ops Weekly tells Acme, signed, when the
reader confirmed. Acme keeps only the count, never which reader confirmed.

A link goes one way. A mutual partnership is two links, set up with two
codes, one in each direction.

## Principles

- Instances talk directly over HTTPS at their public URLs. There is no
  directory or marketplace in between, and nothing goes to HostRepute.
- Every call to another instance goes through the SSRF guard
  (`internal/netguard`): private, loopback and link-local addresses are
  refused, and redirects are never followed. Development instances can
  allow private addresses with `providers.allow_private_networks`, which
  also allows plain HTTP.
- A subscribe request carries only the reader's email address and the
  recommending newsletter's name. Nothing goes in a URL.
- The partner answers the same way whether or not the address already gets
  its list, so the recommending side never learns who reads it.
- Secrets are never sent. The two sides agree on them (X25519 + HKDF) and
  keep them sealed (`internal/secrets`).

## Partner codes

The partner creates a code in Grow › Recommendations › "Let a newsletter
recommend you": a label, the list offered and a daily limit of subscribe
requests (50 by default). The code:

- is 12 characters from `23456789ABCDEFGHJKMNPQRSTUVWXYZ` in three groups,
  `H3RT-6QWN-8ZKB` (about 59 bits);
- is sent as a link: the instance's public URL, `/p/`, the code
  (`https://reputecast.opsweekly.example/p/H3RT-6QWN-8ZKB`). Opening it in a
  browser only explains what it is;
- is valid 7 days, works once, can be revoked, and is shown once: the
  instance keeps `sha256("reputecast-partner-code:" + code)`.

## Endpoints

All under the instance's public URL. Requests and answers are JSON with
`"protocol": 1` and the header `X-ReputeCast-Partner-Protocol: 1`. Errors
are problems:

```json
{"protocol": 1, "error": "code_expired", "detail": "", "at": "2026-10-12T09:52:00Z"}
```

| Method and path | Signed | Purpose |
|---|---|---|
| `GET /partner/v1` | no | The versions this instance speaks: `{"protocols": [1], "instance": "...", "software": "ReputeCast ..."}` |
| `POST /partner/v1/preview` | no | What a code connects to (the code proves the right to ask) |
| `POST /partner/v1/connect` | answer only | Use a code: agree the secret, create both links |
| `POST /partner/v1/links/{link}/subscribe` | yes | A reader asks to subscribe |
| `POST /partner/v1/links/{link}/confirmed` | yes | A reader confirmed (partner → recommending side) |
| `POST /partner/v1/links/{link}/state` | yes | Pause, resume or end the link |
| `POST /partner/v1/links/{link}/profile` | yes | The partner's profile and daily limit changed |
| `POST /partner/v1/links/{link}/rotate` | yes | Agree a new secret |

`{link}` is always the **receiving** instance's id for the link: each side
gives the other its own id when connecting.

Error codes: `code_unknown`, `code_expired`, `code_used`, `code_revoked`,
`signature_rejected`, `unknown_link`, `ended`, `bad_request`,
`unsupported_protocol`, `rate_limited`.

Rate limits: 600 requests a minute per address, of which 20 to `preview`
and `connect`.

## Connecting

1. The recommending side reads the pasted link: the partner's base URL is
   everything before `/p/`.
2. `GET /partner/v1` checks that the instance speaks version 1. A 404 or
   no version 1 means "runs a ReputeCast version without partner links".
3. `POST /partner/v1/preview` with `{"protocol": 1, "code": "H3RT-6QWN-8ZKB"}`
   answers the profile shown before connecting:

   ```json
   {"protocol": 1, "profile": {"name": "Ops Weekly", "description": "...", "logoUrl": "https://...", "list": "Ops Weekly"},
    "dailyLimit": 50, "doubleOptIn": true, "instance": "https://reputecast.opsweekly.example"}
   ```

4. `POST /partner/v1/connect`:

   ```json
   {"protocol": 1, "code": "H3RT-6QWN-8ZKB", "link": "<recommending side's link id>",
    "instance": "https://reputecast.acme.example",
    "recommender": {"name": "Acme Weekly Digest", "description": "...", "list": "Weekly Digest"},
    "key": "<X25519 public key, base64url>"}
   ```

   The partner checks the code (in a transaction, so it is used once),
   creates its link and answers with its own link id and public key:

   ```json
   {"protocol": 1, "link": "<partner's link id>", "key": "<X25519 public key>", "profile": {...}, "dailyLimit": 50}
   ```

   The answer is signed with the new secret (see below). The recommending
   side derives the secret, checks the signature and only then keeps the
   link: both sides proved they hold the same secret.

### The shared secret

```
shared = X25519(own private key, other side's public key)
secret = "rcps_" + base64url(HKDF-SHA256(shared, salt, info, 32 bytes))
salt   = sha256("connect:" + code)        when connecting
         sha256("rotate:" + rotation id)  when rotating
info   = "reputecast-partner-v1|" + recommending link id + "|" + partner link id
```

Each key pair is used once and thrown away. Both sides seal the secret.

## Signatures

Every message about a link is signed with HMAC-SHA256:

```
X-ReputeCast-Timestamp: 1760000000
X-ReputeCast-Signature: v1=<hex>, v1=<hex>
signed = timestamp + "\n" + METHOD + "\n" + path + "\n" + body
```

- `path` starts at `/partner/v1` (an instance behind a proxy that adds a
  prefix still verifies).
- There is one `v1=` entry per live secret: the current one, and the
  previous one for 24 hours after a rotation. A receiver accepts a message
  that matches any of its live secrets.
- A message whose timestamp is more than 5 minutes off is refused
  (`401 signature_rejected`).
- Answers to signed messages are signed the same way, with
  `RESPONSE <status>` in place of the method and the request's path. A
  sender ignores an answer whose signature doesn't match (and retries).

## Subscribe requests

```json
{"protocol": 1, "request": "<request id>", "email": "sam.rivera@outlook.com", "source": "Acme Weekly Digest", "at": "2026-10-15T13:41:00Z"}
```

The partner:

1. checks the signature and the timestamp;
2. answers a request id it has seen before with the same answer, and
   nothing else happens (replay protection, together with the timestamp);
3. checks its daily limit for the link (requests accepted since 00:00 UTC);
4. adds the address to the list as pending and sends its double opt-in
   email on its System stream, with the line "You asked for this on the
   Acme Weekly Digest's page." An address that already gets the list gets
   nothing. The new contact's growth source is "Partner · Acme Weekly
   Digest", with the consent evidence of the double opt-in.

Answers: `{"protocol": 1, "status": "accepted"}`, `"limit_reached"` with
`"resetsAt"` (the next 00:00 UTC), `"paused"` or `"ended"`.

The recommending side:

- tries at once, so the reader sees "Ops Weekly will email you to confirm,
  unless you already get it";
- when the partner doesn't answer, retries with back-off (1, 2, 5, 10, 30
  minutes, then hourly) for 24 hours, then marks the request failed. The
  reader sees "We've passed your request to Ops Weekly. Their confirmation
  email can take a few minutes." Members with grow:connect are told when a
  partner has failed for 30 minutes;
- on `limit_reached`, hides the recommendation until `resetsAt`;
- on `401 signature_rejected`, `404 unknown_link` or `410 ended`, marks the
  link invalid, hides the recommendation and tells members with
  grow:connect. The fix is a new partner code;
- clears the reader's address as soon as the partner answered or the
  request failed.

## Confirmations

When the reader confirms, the partner sends

```json
{"protocol": 1, "request": "<request id>", "at": "2026-10-15T13:52:00Z"}
```

to `/partner/v1/links/{recommending side's link}/confirmed`, retried for 24
hours. The recommending side adds one to the recommendation's confirmed
count. The notice names the request, never the reader.

## Pausing, ending, profiles and rotation

- `state`: `{"protocol": 1, "state": "paused" | "active" | "ended"}`. A
  pause from either side stops requests and hides the recommendation; only
  the side that paused can resume. Ending is final for both sides: the
  recommending side shows the link as invalid when the partner ended it.
- `profile`: `{"protocol": 1, "profile": {...}, "dailyLimit": 50}`, sent by
  the partner when its profile or the link's daily limit changes.
- `rotate`: `{"protocol": 1, "rotation": "<id>", "key": "<public key>"}`,
  signed with the current secret. The receiver answers with its public key
  (signed with the current secret), and both sides derive the new secret
  with `salt = sha256("rotate:" + rotation id)`. The previous secret keeps
  working for 24 hours. A rotation id is applied once; repeating it is a
  `409`.

## Records

- People's actions are audited on the side that acted: a code created or
  revoked; a link connected, paused, resumed or ended; its secret rotated;
  its daily limit changed. Actions that arrive from the partner are audited
  as "Partner · <name>".
- Subscribe requests and confirmations are not audited. The partner
  request logs keep them for 90 days, without addresses.

## Versioning

A new version gets a new prefix (`/partner/v2`) and its number in
`GET /partner/v1`'s `protocols`. An instance speaks every version it
lists; a request with an `X-ReputeCast-Partner-Protocol` it doesn't speak
gets `400 unsupported_protocol`.
