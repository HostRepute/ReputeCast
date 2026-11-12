# Backup and restore

A ReputeCast instance is three things, and a restore needs all three:

| What | Where | Why it matters |
|---|---|---|
| The database | Postgres | Everything except uploaded files. |
| The secret key | `REPUTECAST_SECRETS__KEY`, or the key file (`data/secret.key`; `/var/lib/reputecast/secret.key` in Docker) | Provider credentials, API secrets, two-factor seeds and other secrets are encrypted with it, and links in sent emails are signed with it. A database without its key restores, but none of those open. |
| The files | The files folder, `REPUTECAST_FILES__DIR` (`files` next to the key file); or the bucket, with `REPUTECAST_STORAGE__BACKEND=s3` ([file storage](configuration.md#file-storage)) | Digital products' files, lead magnets, audience exports and other uploads. |

Keep the key apart from the database backups (a password manager or a
secrets store): whoever holds both can read every stored credential.

## Back up

With Docker Compose (`deploy/docker-compose.yml`):

```bash
docker compose exec -T postgres pg_dump -U reputecast -Fc reputecast > reputecast-$(date +%F).dump
```

```bash
docker compose cp app:/var/lib/reputecast/secret.key ./secret.key
```

```bash
docker compose cp app:/var/lib/reputecast/files ./files
```

With the binary, run `pg_dump -Fc` against the database and copy the key
file and the files folder. A dump taken while ReputeCast runs is consistent;
files uploaded during the dump may be missing from it, which is harmless.

With the files in a bucket, back the bucket up instead of the folder:
turn on versioning, or replicate it (S3 replication to a bucket in another
account or region), or copy it with your provider's tools
(`aws s3 sync s3://<bucket>/<prefix> ./files`, `rclone sync`,
`mc mirror`). A copy taken this way restores into a folder or a bucket
alike: both use the same keys.

Restore a backup on a spare host from time to time. That is the only way to
know it works.

## Restore on a new server

`pg_dump` doesn't carry roles. ReputeCast runs user-written SQL segments as a
role of its own, `reputecast_segments`, which owns nothing and can read
nothing by itself, so a new server needs it before the restore.

1. Install the same ReputeCast release as the backup, or a newer one.
2. As a Postgres superuser, create the app's user, its database and the
   segments role:

   ```sql
   CREATE ROLE reputecast LOGIN PASSWORD '…';
   CREATE DATABASE reputecast OWNER reputecast;
   CREATE ROLE reputecast_segments NOLOGIN NOINHERIT;
   GRANT reputecast_segments TO reputecast;
   ```

   On a managed database without superuser access, run the same statements
   as its admin user. Without the role, ReputeCast works and SQL segments
   stay off.
3. Restore the dump as the app's user:

   ```bash
   pg_restore --no-owner --no-privileges -d "postgres://reputecast:…@db/reputecast" reputecast.dump
   ```

4. Put the secret key and the files in place: the files folder at the same
   path (or set `REPUTECAST_SECRETS__KEY_FILE` and `REPUTECAST_FILES__DIR`),
   or the bucket's objects in the bucket `REPUTECAST_STORAGE__S3__*` names.
5. Apply the migrations of a newer release, if any, and hand the segments
   function back to its role (the restore made the app's user its owner):

   ```bash
   reputecast migrate up
   ```

   ```bash
   reputecast migrate segments-role
   ```

6. Check:

   ```bash
   reputecast doctor
   ```

   Every line should read `ok`. `doctor` checks the database and its schema,
   opens a few sealed values of every kind with the key (a wrong key fails
   here, not later in a provider call), checks the file storage (the
   folder can be written; the bucket answers and takes a test object) and
   the segments role.
7. Start ReputeCast. If the public address changed, set
   `REPUTECAST_PUBLIC_URL` (or Instance admin › Instance & updates), which
   also moves provider webhooks.

We tested this runbook on a fresh Postgres 16 cluster with the demo data
(1 November 2026): `doctor` reported the segments role until `migrate
segments-role` ran, then every check passed.

## Rotating the secret key

Rotate the key when it may have leaked, or as a routine:

1. Make a new key:

   ```bash
   reputecast secrets generate-key
   ```

2. Set it as `REPUTECAST_SECRETS__KEY` and add the old key to
   `REPUTECAST_SECRETS__PREVIOUS_KEYS` (comma-separated). If you use a key
   file, write the new key there and put the old one in the variable.
   Restart every process.
3. Re-encrypt every stored secret with the new key. It is safe while
   ReputeCast runs and safe to run again:

   ```bash
   reputecast secrets reseal
   ```

   `--dry-run` shows what it would do. It reports values no key opens; there
   should be none. The reseal is in the instance audit log.
4. Decide how long the old key stays in `PREVIOUS_KEYS`. Links in emails
   already sent (unsubscribe, preferences, tracking) were signed with it and
   keep working only while it is there. Unsubscribe links must keep working
   (at least 30 days after a send under CAN-SPAM, and people click old
   emails), so for a routine rotation keep the old key for good: it no
   longer encrypts anything, it only checks old links. After a leak, remove
   it once the reseal reports nothing left on it, and accept that older
   links stop working; wait at least a day, since signup forms hand out
   short-lived tokens sealed with it.

`reputecast doctor` warns while stored values still use a previous key.

## Downgrades

Migrations only add within a minor release (1.6.0 → 1.6.3), and a binary
keeps working against a schema a newer patch release migrated. So:

- Rolling back to an earlier **patch** release of the same minor release
  needs nothing: replace the binary or the image.
- Rolling back across a **minor** release (1.7 → 1.6) is supported for one
  minor release: the older binary runs on the newer schema (it logs that the
  database was migrated by a newer release). Read the release notes first:
  a release that drops or renames something says so and isn't covered.
- Further back, restore the backup taken before the upgrade.

`reputecast migrate down` rolls back one migration at a time and is meant
for development. Don't use it on a production database: down migrations
drop data.

A process whose binary is newer than the database (migrations pending)
serves read-only until they are applied; see [Upgrades](README.md#upgrades).
