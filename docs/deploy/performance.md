# Performance

Numbers from the load test data set and how to measure your own (review
2026-11 OP-9).

## The data set

```bash
reputecast seed
```

```bash
reputecast seed --sample load --contacts 1000000
```

The second command adds a "Load test" workspace: a million contacts (2%
unsubscribed, 0.5% bounced, at six mailbox providers, joined over two
years), three lists, three sent campaigns with 2.4 million messages and 3.2
million delivery events over five months. It is generated in Postgres and
takes about 8 minutes; the database grows by 3 GB. Development only.

## Measuring

| Tool | What it measures | How |
|---|---|---|
| Go tests in `internal/loadtest` | Each hot call's median through the whole HTTP stack; the contacts targets of AU-1 and AU-2; ten people browsing at once | `make loadtest`, with `REPUTECAST_LOAD_DATABASE_URL` naming a database to fill |
| Go benchmarks | The same calls, and storing provider events | `go test -run '^$' -bench . ./internal/loadtest/` |
| k6 scripts in `loadtest/k6` | Contacts, segment counts, the send API and webhook ingest against a running server | `k6 run loadtest/k6/contacts.js` with `BASE_URL` and `COOKIE` (`loadtest/session.sh` prints one) |
| The "Load test" workflow | All of the above on a GitHub runner, Postgres 16, 17 or 18 | Actions › Load test › Run workflow; results are an artifact |

## Results

1 November 2026, PostgreSQL 16 with `fsync=off`, 4 vCPUs, the server and
the database on one host. Medians of 9 to 15 requests after a warm-up.

| Call | Before partitioning | After |
|---|---|---|
| Contacts, page 1 (25 a page) | 6 ms | 6 ms |
| Contacts, page 400 (cursor) | 6 ms | 7 ms |
| Contacts, search "load4242" | 23 ms | 23 ms |
| Contacts, exact email | 3 ms | 3 ms |
| Contacts at one mailbox provider | 21 ms | 18 ms |
| Campaigns list | 6 ms | 6 ms |
| Overview, sender health | 8 ms | 8 ms |
| Audience summary (header counts) | 317 ms | 275 ms |
| Campaign report, 975,000 recipients | 344 ms | 272 ms |
| Segment count, two conditions | 990 ms | 927 ms |
| Storing a provider event | 0.32 ms | 0.37 ms |

Targets: the contacts list's pages under 200 ms and search under 150 ms
(AU-1, AU-2) hold with a wide margin. Ten people browsing and searching at
once: 285 requests a second, p50 26 ms, p95 90 ms, p99 118 ms, no errors.

Converting `delivery_events` to monthly partitions took 5 seconds. Storing
an event costs 0.05 ms more afterwards: its dedupe key goes to its own table
([partitioning](partitioning.md)).

### The Load test workflow, first run (review 2026-11 OP-13)

1 November 2026, [run 41353963642](https://github.com/HostRepute/ReputeCast/actions/runs/41353963642):
a GitHub runner (4 vCPUs) with PostgreSQL 18, the server, the database and
k6 on one host. The data set loaded in 22 minutes 50 seconds: 1,000,000
contacts, 2,427,738 messages, 3,183,918 delivery events.

| Call | Median | After partitioning |
|---|---|---|
| Contacts, page 1 | 6 ms | 6 ms |
| Contacts, page 400 | 7 ms | 6 ms |
| Contacts, search "load4242" | 54 ms | 51 ms |
| Contacts, exact email | 4 ms | 5 ms |
| Contacts at one mailbox provider | 16 ms | 16 ms |
| Campaigns list | 7 ms | 7 ms |
| Overview, sender health | 12 ms | 11 ms |
| Audience summary | 3 ms (kept) | 3 ms |
| Campaign report, 975,000 recipients | 364 ms | 356 ms |
| Segment count | 3 ms (kept) | 3 ms |

Every call meets its target one at a time. Under load:

| k6 script | Load | Result | Target |
|---|---|---|---|
| contacts.js, pages | 10 people, 1 minute, 116 requests a second | p95 50 ms, no errors | p95 under 200 ms: met |
| contacts.js, search | the same | p95 513 ms, one request 27 s | p95 under 150 ms: missed |
| segments.js | 4 people, 513 counts a second | p95 10.6 ms, no errors | p95 under 2 s: met |
| send.js | rising to 200 a second | 48% accepted, p95 760 ms | see below |

The Go test of ten people browsing at once on the runner: 128 requests a
second, p50 28 ms, p95 270 ms (bound 200 ms), no errors. Job queues and
webhook ingest had no lag at the end.

What the run showed, and what changed:
- **Search under load.** Every sample address contains "load", so the
  email trigram index reads posting lists that hold the whole table for each
  search, and a search also counted its matches with a second scan. The
  first page now counts its own rows when they are all the matches, so a
  search that finds a few contacts reads the indexes once. Measured here
  on 200,000 contacts (4 vCPUs, the same data and machine before and
  after), ten people browsing at once went from 403 to 472 requests a
  second, p95 from 65 ms to 55 ms, and a search's median from 9 ms to
  7 ms. On the runner the server, Postgres and k6 share four cores; the
  next run records the result there.
- **Sending.** send.js sent through an owner's session, and a person's
  sessions share 50 messages a second: the 52% that weren't accepted were
  that limit's 429s, as designed. It now sends with a send-only key at 1,000
  a second (`loadtest/sendkey.sh`), as an application does.
- **The workflow** now fails when k6 or a timing test crosses its threshold
  (pipefail) and still runs every step, so one run records every number. A
  pull request labelled `load-test` runs it before the branch merges.

### Second and third runs, on those changes

[Run 41358794955](https://github.com/HostRepute/ReputeCast/actions/runs/41358794955)
and, after the template compile cache below,
[run 41364026420](https://github.com/HostRepute/ReputeCast/actions/runs/41364026420),
same runner and data; the job fails because two thresholds are still
crossed.

| | First run | Second run | Third run | Target |
|---|---|---|---|---|
| contacts.js, requests a second | 116 | 153 | 160 | — |
| contacts.js, pages p95 | 50 ms | 53 ms | 54 ms | under 200 ms: met |
| contacts.js, search p95 | 513 ms | 291 ms | 307 ms | under 150 ms: missed |
| Ten people browsing (Go test), p95 | 270 ms | 273 ms | 273 ms | under 200 ms: missed |
| segments.js, p95 | 10.6 ms | 10.4 ms | 10.1 ms | under 2 s: met |
| send.js, accepted | 48% (a session's 429s) | 100% with a key | 100% with a key | — |
| send.js, messages a second | — | 76 | 84 | — |
| send.js, median | — | 474 ms | 400 ms | — |
| send.js, p95 | 760 ms | 827 ms | 814 ms | under 300 ms: missed |

Sending through a key measured the send API itself: 76 messages a second,
each taking a median 474 ms with 50 senders, while the worker drained the
queue on the same four cores. A CPU profile of the send call showed most of
it compiling and checking the template's MJML on every message; a version is
now compiled once per process, which cut the send call's CPU by a quarter.
On the runner that moved the median from 474 ms to 400 ms and the rate from
76 to 84 a second, but not the p95: k6's 50 senders, the server, the worker
draining the queue and Postgres share four cores, so the tail is queueing
for CPU. On a development machine, where the server doesn't share its
cores with k6 and the worker, the same send path answered 494 calls a second
(a third of them stopped by the key's rate limit) with a p95 of 118 ms.

Search, a median 52 ms on its own at a million contacts against 7 ms at
200,000, grows with the table rather than with its matches. Its plan is the
right one (EXPLAIN ANALYZE at 200,000: a BitmapOr of the email and name
trigram indexes and the tags index, 11 rows, 6 ms, the GIN pending lists
empty); the cost is the email trigram index itself. Every sample address
contains "load", so two of the six trigrams of "load4242" are in every row,
and their posting lists grow with the table. That is a worst case real lists
don't have, and a search on its own meets its target; ten people searching
at once on four shared cores don't.

The approvals inbox with 1,000 open requests (`TestInboxAtAThousandRequests`,
review 2026-11 AP-3): 118 to 200 ms when every request was checked
recently, as the inbox re-checks only stale ones; 460 ms when ten campaigns
had changed (2.3 s before, and 14 s on a CI runner). The stale requests are
checked four at a time, and those that send to the same audience or route
count it once: an audience count is most of a check's cost. The background
job that re-checks every open request each five minutes shares counts the
same way.

Slow counts that read every contact are kept for a minute (review 2026-11
OP-12): the Audience header's counts (about 300 ms on a million contacts)
and a segment preview's count (about a second for two conditions). Only a
count that took a quarter of a second or more is kept, so a small
workspace always counts afresh. The same request within the minute answers
from memory, and readers asking at the same time share one count. A change made through ReputeCast (a
contact, list, import, field or suppression: anything that writes an audit
entry) drops the workspace's counts on that instance at once, and so does a
product order, which "Bought a product" segments count. A change another
instance makes, a public signup, an open or click, or a store order synced
from Shopify or WooCommerce shows within the minute. A
segment preview says `cached` when its count was kept.

## Worth a look next

- The campaign report (about 300 ms for 975,000 recipients). The load data
  set's campaigns were generated, not sent, so they have no live counters:
  profile it again on a real send.
