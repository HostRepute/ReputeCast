# ReputeCast on Railway

One ReputeCast service from the release image, with a volume at
`/var/lib/reputecast` (the secret key file and uploaded files), and Railway
Postgres. Migrations run when the service starts, the first Owner is created
from `REPUTECAST_ADMIN_EMAIL`, and the temporary password is in the
deployment's logs (search for `ReputeCast is installed`).

## Deploy it

**From the template** (once it's published, see below): click the button,
enter the first Owner's email, deploy.

**By hand**, in a new Railway project:

1. **+ New › Database › PostgreSQL.**
2. **+ New › Docker Image** `ghcr.io/hostrepute/reputecast:latest`, then on
   that service:
   - **Variables:** the table below.
   - **Settings › Networking › Generate Domain**, port `8080` (or add your
     own domain).
   - **Settings › Deploy › Healthcheck Path** `/readyz`, timeout `300`.
   - Right-click the service › **Attach Volume**, mount path
     `/var/lib/reputecast`.
3. Deploy, then read the temporary password in the service's logs.

**With the Railway CLI** (5.42.1 or newer), from a checkout, into a linked
project. [`railway.ts`](railway.ts) describes the same project as
[Infrastructure as Code](https://docs.railway.com/infrastructure-as-code):

```bash
npm install --prefix deploy/railway --no-save railway@3
railway link
REPUTECAST_ADMIN_EMAIL=you@example.com railway config apply --file deploy/railway/railway.ts
```

Then generate the domain (Settings › Networking, port `8080`): the public URL
follows it.

## Variables

| Variable | Value | Why |
|---|---|---|
| `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` | The database. ReputeCast reads `DATABASE_URL` when `REPUTECAST_DATABASE__URL` isn't set. |
| `REPUTECAST_PUBLIC_URL` | `https://${{RAILWAY_PUBLIC_DOMAIN}}` | The address people and providers reach: the generated domain, or your own. |
| `REPUTECAST_ADMIN_EMAIL` | *asked for* | The first Owner. |
| `REPUTECAST_HTTP__TRUSTED_PROXIES` | `private` | Trust `X-Forwarded-For` from Railway's proxy, on private networks only. |
| `PORT` | `8080` | The port the image listens on, which the domain routes to. |
| `RAILWAY_RUN_UID` | `0` | Railway mounts volumes as root; without this the image's non-root user can't write the secret key file. |

Back up the volume's `secret.key` with the database: provider credentials are
encrypted with it (Railway backs volumes up, see the service's **Backups**).
Other settings are in the [configuration guide](../../docs/deploy/configuration.md).

## Publishing the template (maintainers)

Railway templates are made in the dashboard, not from a file in the
repository:

1. In a project that runs the setup above (apply `railway.ts`, or follow the
   steps by hand), open **Settings › Generate Template from Project**; or
   create it from scratch in **Workspace › Templates › New Template** with
   the two services, the variables, the volume, the healthcheck and HTTP
   public networking on port `8080`.
2. Give `REPUTECAST_ADMIN_EMAIL` no value and the description "The first
   Owner's email address", so Railway asks for it.
3. **Publish** the template, with the name ReputeCast, the slogan
   "Open-source newsletters that reach the inbox", the
   [logo](../../brand/logo/reputecast-mark.svg), the category Email, and this
   README as its documentation.
4. Put the template's Deploy on Railway button
   (`https://railway.com/new/template/<code>`) in `README.md`,
   `docs/deploy/README.md` and `railway` in `site/src/content/links.ts`,
   instead of the link to this page.
