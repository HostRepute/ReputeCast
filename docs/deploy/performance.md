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

The k6 scripts weren't run for these numbers; the workflow runs them.

## Worth a look next

- The segment count (about a second for two conditions on a million
  contacts) and the audience summary's header counts. Both read every
  contact; a count cache would help.
- The campaign report (about 300 ms for 975,000 recipients). The load data
  set's campaigns were generated, not sent, so they have no live counters:
  profile it again on a real send.
