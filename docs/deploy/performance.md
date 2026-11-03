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

### Fourth run: what held the send test and the server back

The third run's send log had a shape that CPU alone doesn't make: at under
130 sends a second most sends took 10 to 20 ms, yet every 5 to 10 seconds
all 50 senders stalled at once for about a second. Every send locked two
rows that every other send needed, until it committed:

- The key's rate bucket was locked with an upsert, read in Go and written
  back: four round trips and a WAL flush under the lock. A take is now one
  conditional UPDATE that commits without waiting for its flush (a crash can
  only lose recent takes); a refused take writes nothing.
- Each send inserted the outbox's unique wake-up job in its own
  transaction, and River's insert locks that job's row until the send
  commits. A send now wakes the outbox once it has committed: one goroutine
  inserts the job, so a burst costs a few inserts, and it asks again while a
  running pass may have missed the mail. The watch still wakes anything a
  lost wake leaves within 15 seconds. Automation emails, queued inside the
  automation engine's transaction (or a referral reward's), now do the same
  once that transaction commits (`db.AfterCommit`), so many automation
  workers at once don't queue on that row either.

Locally, with fsync on and the send.js profile replayed by a small Go
client (k6 can't run there), the same 122 accepted sends a second went
from p95 28 ms, p99 151 ms and a 773 ms worst to 22 ms, 46 ms and 298 ms,
with no second of all 50 senders stalled (two before), and half the row
versions on the bucket.

Two more changes went into the run. A search with more than a page of
matches read the trigram indexes twice, for the page and for the count; it
now reads its matches once, up to the 10,000 cap, and reads the page from
them by id. And the jobs a worker runs when it starts ran through every k6
script: the workflow now waits for them and lists those that ran long.

[Run 41381703256](https://github.com/HostRepute/ReputeCast/actions/runs/41381703256)
(fourth) and, with the deadlock below fixed,
[run 41386901262](https://github.com/HostRepute/ReputeCast/actions/runs/41386901262)
(fifth):

| | Third run | Fourth run | Fifth run | Target |
|---|---|---|---|---|
| contacts.js, requests a second | 160 | 219 | 246 | — |
| contacts.js, pages p95 | 54 ms | 44 ms | 40 ms | under 200 ms: met |
| contacts.js, search p95 | 307 ms | 199 ms | 160 ms | under 150 ms: missed |
| Ten people browsing (Go test), p95 | 273 ms | 143 ms | 147 ms | under 200 ms: met |
| Search on its own (Go test) | 51 ms | 27 ms | 28 ms | under 150 ms: met |
| segments.js, p95 | 10.1 ms | 7.0 ms | 5.2 ms | under 2 s: met |
| send.js, messages a second | 84 | 129 | 134 | — |
| send.js, median | 400 ms | 215 ms | 25 ms | — |
| send.js, p95 | 814 ms | 299 ms | 269 ms | under 300 ms: met |
| send.js, worst | 1.89 s | 408 ms | 438 ms | — |
| send.js, arrivals dropped (no free sender) | 7,851 | 1,023 | 304 | — |

The run's Postgres log showed what still slowed search: the first tier run
of the million contacts (`audience.tiers`, which a worker starts with) and
the mailbox classifier deadlocked again and again, one UPDATE over every
contact against one over every contact of a domain. Each loss rolled back up
to a million updated rows and the job started over, writing so much WAL
that Postgres checkpointed every six seconds while k6 ran, and the wait for
start-up jobs ended in one of its pauses between attempts. A tier run now
assigns tiers 5,000 contacts at a time, each batch one statement that locks
only the contacts it changes, in a fixed order; the classifier updates 5,000
at a time and skips contacts another transaction holds, picking them up on
its next pass, so it never waits and can't deadlock
(`TestClassifierSkipsLockedContacts` waited for the held contact before).
People's edits no longer wait behind a tier run either.

In the fifth run that first tier run finished in one go, in 309 seconds, and
nothing deadlocked. Search was the last target missed, by 10 ms, and k6 had
started 36 seconds after the tier run ended: that run had rewritten most of
the million contacts, and as a tier and a tag are indexed, every index took
a new entry for each, the two trigram indexes search reads included.
Autovacuum gets to such a table minutes later, which put it in the middle of
k6. That happens once, after an import or on first start; a daily tier run
changes the contacts that crossed a line that day. The workflow now runs
`VACUUM (ANALYZE) contacts` itself after the start-up jobs, as autovacuum
would have long before a busy hour, and records how long it took.

### Sixth run: every target met

[Run 41390919040](https://github.com/HostRepute/ReputeCast/actions/runs/41390919040),
the same runner and data, passed: the workflow fails on any crossed k6
threshold or timing test, and none was. The start-up jobs settled after 345
seconds (the first tier run of the million contacts took 317), and the
vacuum after them 2 seconds.

| | First run | Fifth run | Sixth run | Target |
|---|---|---|---|---|
| contacts.js, requests a second | 116 | 246 | 278 | — |
| contacts.js, pages p95 | 50 ms | 40 ms | 35 ms | under 200 ms: met |
| contacts.js, search p95 | 513 ms | 160 ms | 140 ms | under 150 ms: met |
| Ten people browsing (Go test), p95 | 270 ms | 147 ms | 140 ms | under 200 ms: met |
| Search on its own (Go test) | 52 ms | 28 ms | 27 ms | under 150 ms: met |
| segments.js, p95 | 10.6 ms | 5.2 ms | 5.0 ms | under 2 s: met |
| send.js, messages a second | — | 134 | 135 | — |
| send.js, median | — | 25 ms | 23 ms | — |
| send.js, p95 | 760 ms | 269 ms | 247 ms | under 300 ms: met |
| send.js, worst | — | 438 ms | 337 ms | — |
| send.js, arrivals dropped (no free sender) | — | 304 | 117 | — |

The first run's send numbers went through a person's session, so its
latency isn't comparable. Search's margin is the thinnest, 10 ms, and a
shared runner moves by tens of milliseconds from one run to the next: the
next runs will tell whether it holds. Its cost is still the email trigram
index on sample addresses that all contain "load" (above); on four cores
shared with k6, ten people with no pause between requests keep the CPU busy.

### Seventh run: partitioned from the start

Since the upgrade converts `delivery_events` (review 2026-11 OP-11), the
seed's `migrate up` partitions the table before the data goes in, so every
timing reads the partitioned table.
[Run 41402124614](https://github.com/HostRepute/ReputeCast/actions/runs/41402124614)
passed every threshold: search p95 96 ms, pages p95 26 ms (392 requests a
second), segment counts p95 3.6 ms, sends p95 94 ms (median 11 ms, 135 a
second, 103 arrivals dropped), ten people browsing p95 108 ms, search on its
own 23 ms. The first tier run took 274 seconds and the vacuum after it 3.
That runner was faster than the sixth run's (the seed took 19 minutes
rather than 23), so part of the gain is the machine: the sixth run is the
fairer bar.

### Eighth run: after the automation wake

[Run 41540067366](https://github.com/HostRepute/ReputeCast/actions/runs/41540067366),
on the commit that wakes the automation outbox after commit and skips the
per-language query of a report in one language, passed every threshold with
no failed request: search p95 73.5 ms, pages p95 18.8 ms (536 requests a
second), segment counts p95 2.5 ms, sends p95 16.2 ms (median 6 ms, 136 a
second, 2 arrivals dropped), ten people browsing p50 16 ms, p95 81 ms and p99
96 ms. Its timing test read a campaign report on the million contacts in
175–202 ms. The seed took 14 minutes, so this runner was faster again;
the sixth run stays the fairer bar. The workflow now masks the k6 session
cookie in its log, as it did the send key.

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

## The web app on a slow phone

Measured on the production build served by the binary, in Chrome emulating a
phone (390 px, the CPU four times slower, "Slow 4G": 150 ms round trips,
1.6 Mbps), signed in to the demo workspace. Paint and layout-shift numbers
come from the page's own PerformanceObserver.

| | Before | After |
|---|---|---|
| Scripts and requests for one screen | 196–260 requests, 690–816 KB | 30–82 requests, 447–680 KB |
| First load (scripts preloaded by `index.html`) | 145 scripts, 388 KB gzipped | 5 scripts, 316 KB gzipped |
| First paint with content | 7.7–9.3 s (a blank page until then) | 1.0–1.2 s |
| Largest paint (the screen's content) | 8.6–10.2 s | 4.7–6.3 s |
| Layout shift (CLS), 390 and 1440 px | up to 0.91 (Campaigns at 1440) | 0.03 at most on phones, under 0.07 on desktop |

- **Chunks.** Left to itself, the bundler cut the app into about 990 files: a
  module that the startup code and a lazy page both used went into a chunk of
  its own. `vite.config.ts` now groups the route definitions and the shared
  components (803 files). `pnpm budget` holds the first load to 340 KB.
- **First paint.** `index.html` shows the ReputeCast mark until the first
  screen has a heading, instead of a blank page while the scripts and the
  session load.
- **Layout shift.** A tab's count keeps its room while it loads, and a screen
  whose parts arrive one by one (Contacts' filters, the Overview's lower
  cards, the automations summary, the campaigns' channel chips, the
  deliverability status) keeps its skeleton until they have
  (`useFirstLoadSettled`, `FilterTabs`' `pending`, `NavTab`'s `reserve`).
  Per screen, at 390 / 1440 px: Overview 0.017 / 0.021, Campaigns 0.002 /
  0.065, Contacts 0.001 / 0.007, Segments 0.000 / 0.015, Automations 0.028 /
  0.012, Deliverability 0.010 / 0.031.
- **Public pages** (home, archive, a tag, subscribe, upgrade): one request of
  3–5 KB each, first and largest paint 0.36–0.52 s, no layout shift.

### Second pass: the screen's own code

The numbers above stopped when the network had been quiet for half a second.
On a CPU four times slower that can happen while the scripts are still
running, before the screen's content, so their largest-paint figures read
low. This pass waits for the screen's heading and then for the network, uses
Chrome's own "Slow 4G" preset (562 ms round trips, 1.44 Mbps), and measures a
returning member (the browser has signed in before). Each figure is the
median of three loads with the cache off. HTTP/2 goes through a TLS proxy,
as a deployment does (Caddy speaks it on its own; the nginx example in
[reverse-proxy.md](reverse-proxy.md) turns it on). HTTP/1.1 is the binary on
its own. `node web/scripts/phone-perf.ts --app <url>` repeats it.

What the Contacts screen loaded besides the first load was 280 KB (brotli).
Of that, 189 KB was copy: one chunk per feature held all of that feature's
text, so a few filter labels brought in all of Earn's, SMS's and the
composer's text. The language names and the brand name the contact screens
show also came from the composer's and the Settings card's modules, 71 KB
and 22 KB that ran on every Contacts load. Two more changes:

- each feature's copy goes with the code that uses it;
- a member's screen starts loading its code while its routes check the
  session (`/api/v1/me`), not after (`preloadScreen` in `router.tsx`).

| HTTP/2, phone | Before | After |
|---|---|---|
| Contacts: heading, largest paint | 8.07 s, 9.75 s | 6.38 s, 7.81 s |
| Campaigns | 6.25 s, 7.40 s | 5.66 s, 6.26 s |
| Overview | 6.80 s, 7.75 s | 5.85 s, 7.00 s |
| All requests, data and fonts included (Contacts, Campaigns, Overview) | 100, 57, 76 | 98, 61, 82 |

Over HTTP/1.1 a browser opens six connections, so the screen's 60 to 100
requests wait their turn. There the two builds come within 0.5 s of each
other, either way: Contacts' heading at 11.9 s, Campaigns' at 8.2 s and the
Overview's at 10.1 s. Serve the app over HTTP/2.

Total blocking time is 1.1–1.6 s on both builds. It is the startup scripts
(React, the router, TanStack Query, zod and every route's definition,
317 KB gzipped) being parsed and run on the slower CPU; the changes above
left it as it was. On Contacts, without the CPU slowdown, the page's own
script work went from about 600 ms to 410–420 ms once the composer's code
stopped loading there.

### Third pass: data alongside code, and fewer forced layouts

With the screen's code loading during the session check, its data still
waited for that code: Contacts asked for its list at 6.1 s, once the code was
in, though the address said what to ask for at 3.9 s. Two changes:

- The Overview, Campaigns and Contacts routes prefetch their first queries
  from their `loader` (`prefetchFirst` in `lib/prefetch.ts`). The route
  doesn't wait for them; only what the person's permissions show is asked
  for; the list only for the screen as the menu opens it, so a filtered
  address can't prefetch a list the screen won't show. The page and the
  loader build each query in the same small module (`queries.ts`), so they
  are the same query.
- Nothing measures the page in the middle of React's work any more. A table
  that may scroll sideways, a name cell that may be cut and the corner stack
  above the toasts measured as soon as they mounted, which made the browser
  lay out the whole page there; they now take the ResizeObserver's first
  report, made after layout, and the corner stack writes its offsets on the
  root element only when they change.

Five loads of each screen, the same conditions as the second pass:

| HTTP/2, phone | Largest paint before | After | Blocking time before | After |
|---|---|---|---|---|
| Overview | 6.81 s | 6.20 s | 1238 ms | 1104 ms |
| Campaigns | 6.53 s | 5.86 s | 1483 ms | 1322 ms |
| Contacts | 7.81 s | 7.40 s | 1355 ms | 1130 ms |

Screenshots of five screens at 390 and 1440 px are the same before and
after, pixel for pixel, except for relative times and live counts.

What is left of the blocking time is the startup scripts (React, the router,
TanStack Query, zod and every route's definition: 318 KB gzipped, about
0.4 s on this CPU) and React rendering each screen as it and its data
arrive (0.1–0.5 s a render). Tried, measured, and not kept, because none
moved the blocking time beyond run-to-run noise (±0.3 s):

- rendering the first screen at transition priority (`useDeferredValue`,
  then `startTransition`), so React would build it in slices: the router and
  TanStack Query hand React their changes through `useSyncExternalStore`,
  which React renders in one pass, and the shell and the screen then took
  two long tasks instead of one;
- building each route's search schema on first use instead of at startup;
- formatting a message without values once per language rather than on
  every render.

Where the rest goes, profiled on Contacts on a phone after the fixes in the
Lighthouse section below (CPU four times slower): React drawing the screen
and its data, 0.7–0.9 s, spread over the shell, the table, the filters and
formatting their words, with no one part above a tenth of it; running the
startup scripts, about 0.19 s, of which every route's definition and search
schema is about a tenth (51 KB of route files and 21 KB of route tree in
800 KB). Loading each section's route definitions with that section would
save 20–40 ms of it, so it wasn't done. What would move it is less to draw
per screen, or drawing the first screen on the server, which changes how
the app is built.

### Lighthouse

Lighthouse 12.8 (installed for the audit, not a dependency) on the production
build behind a TLS proxy (HTTP/2), its default mobile run (a phone, the CPU
four times slower, a slow 4G link, simulated) and its desktop preset, with
the browser's storage cleared before each page as it does. Signed-in screens
carry a session cookie. The public pages are on the demo's own site.

What it found, and what changed:

- **The app drew every screen twice for a member whose browser hadn't kept
  their language** (a new device, cleared site data, every Lighthouse run).
  The app started in the visitor's formats and drew the screen again from
  the top when /me brought the member's (I18nRoot remounts on a change). Such
  a screen now waits for /me and the member's language before its first
  render; the ReputeCast mark is up meanwhile, and the routes ask for /me
  anyway. On a phone that was most of the blocking time: Overview 3.67 s →
  0.94 s, Contacts 4.38 s → 0.71 s, Campaigns 1.57 s → 0.52 s.
- **A scrollbar arriving narrowed every screen.** Where scrollbars take room
  (Windows, Linux), a page growing past the window as its data came lost
  15 px of width, and the Overview's providers grid dropped a column (layout
  shift 0.35 on desktop). The root keeps the scrollbar's room from the first
  paint (`scrollbar-gutter: stable`), and gives it up while a dialog locks
  scrolling, which adds the same margin itself.
- **The sidebar's footer jumped.** It sits at the bottom of the page, as on
  the boards; while a screen's data loaded, the page was shorter than the
  window, so it showed at the bottom of the window and jumped off it (0.10).
  It stays invisible, holding its place, until the screen's first data is in.
- **The web font moved text.** Until IBM Plex Sans arrived, text was set in
  the system font, which is wider or narrower; when it came, lines broke
  again (0.25 on the archive on a phone). Text now waits in Arial (or
  Liberation Sans) scaled to IBM Plex Sans's size and height, measured in
  Chrome against the same files: on the archive the swap moves elements by
  6 px on average instead of 37.
- **zod tried `eval`** to compile its parsers, which the app's CSP forbids,
  and the browser reported a policy violation on every load. zod is told
  not to try, before it loads (`src/boot/zod.ts`).
- **Public pages:** the archive, tag and "gone" pages put their posts' titles
  in h3 straight under the page's h1 (now h2), a post's own headings could
  skip a level (they now follow the outline from h2 and keep the author's
  look), and the page selling gift subscriptions said `noindex` and had no
  description (it is indexed, as the upgrade page is). An email's poll was
  published on the web with its answers as links to /0, /1 … (each answer
  is the recipient's own link): the web leaves a poll out, as it does the
  footer, and a daily job redraws posts published before.

| | Performance | Accessibility | Best practices | SEO |
|---|---|---|---|---|
| Public pages, phone (home, archive, tag, post, subscribe, upgrade, gift) | 98–100 | 100 | 100 | 100 |
| Overview, Campaigns, Contacts, desktop | 99–100 | 100 | 100 | 63 |
| Overview, Campaigns, Contacts, phone | 64–81 | 100 | 100 | 63 |

The app's SEO score is its `noindex`: a signed-in workspace isn't for search
engines. On a phone the app's screens still block for 0.4–0.9 s and paint
their content at 3.3–3.9 s: the startup scripts and each screen's render on
a CPU four times slower. A public page that once read 200–550 ms of
blocking time did so from a task the browser runs itself, with no script in
it, and reads 0 ms on the next load.

### The campaign report on a real send

Profiled on a sent campaign of 29,797 recipients (the QA data set), the
report took 46–54 ms, and reading every message for its per-language results
was half of that. A campaign sent in one language has nothing to show
there, and its send snapshot says so: the report now skips that query for
it, and takes 20–21 ms for the same send in one language. A send in two or
more languages still reads it (46–54 ms).
