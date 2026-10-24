# Postgres

ReputeCast stores everything in Postgres: data, background jobs (River) and
the audit log. Postgres 16, 17 and 18 are supported and tested; the Docker
Compose file uses 18.

## Connections

Each ReputeCast process keeps a pool of up to `REPUTECAST_DATABASE__MAX_CONNS`
connections (default 20). With several processes, keep the total below the
server's `max_connections` (100 by default), leaving room for backups and your
own sessions.

For a managed database that requires TLS, add `sslmode=require` (or
`verify-full`) to `REPUTECAST_DATABASE__URL`.

## Tuning for larger audiences

Postgres ships with conservative memory settings. With tens of thousands of
contacts, segment counts with event conditions (opened, clicked…) can get
slow on them. A live count that takes longer than 10 seconds shows the
last known count instead. The segment still works at send time, but counting
it in the editor will keep timing out until the database has more room.

Starting points for a server that runs only Postgres:

| Setting | Starting point | Why |
|---|---|---|
| `shared_buffers` | 25% of RAM | Keeps contacts and recent events in memory. |
| `effective_cache_size` | 50–75% of RAM | Tells the planner how much the OS caches. |
| `work_mem` | 32MB | Sorts and hashes for segment counts and reports. It applies per operation, so mind `max_connections`. |
| `maintenance_work_mem` | 512MB | Faster vacuum and index builds after large imports. |
| `random_page_cost` | `1.1` | Index scans are cheap on SSD storage. |

After a large import, run `ANALYZE;` so the planner sees the new data straight
away instead of waiting for autovacuum.

With the Docker Compose file, pass settings to the `postgres` service:

```yaml
services:
  postgres:
    command: ["postgres", "-c", "shared_buffers=1GB", "-c", "effective_cache_size=3GB", "-c", "work_mem=32MB"]
```

## SQL segments on managed Postgres

SQL segments run as a role that can read nothing on its own: for each query,
ReputeCast creates a temporary view of one workspace's contacts, grants the
role access to it, and runs the query in a read-only transaction with a
statement timeout.

The migrations create that role, `reputecast_segments`. Creating a role needs
the `CREATEROLE` privilege. On managed databases without it, the migration
skips the role and SQL segments stay turned off. To turn them on, have an admin
run this and then migrate again:

```sql
CREATE ROLE reputecast_segments NOLOGIN;
GRANT reputecast_segments TO <app user>;
```

```bash
reputecast migrate up
```

## Backups

Back up the database with `pg_dump` **and** keep the secret key with it. See
[Backups](README.md#backups).
