# ReputeCast Helm chart

Runs [ReputeCast](https://github.com/hostrepute/reputecast), the open-source
newsletter platform, on Kubernetes 1.29 or newer with Helm 3.8 or newer (OCI
charts). It brings no Postgres of its own: point it at Postgres 16, 17 or 18,
such as a [CloudNativePG](#cloudnativepg) cluster or a
[managed database](#a-managed-database).

- **`mode: single`** (the default): one StatefulSet pod runs the web app, the
  API, webhook ingest and the workers, with a 10 Gi volume at
  `/var/lib/reputecast` for uploaded files.
- **`mode: split`**: Deployments for the web app (2 replicas), the workers (1)
  and, with `ingest.separate`, provider webhooks. They share uploaded files
  through an S3-compatible bucket, so split needs `storage.backend: s3`.

Migrations run in a Job before every install and upgrade
(`reputecast migrate up`); the pods never migrate themselves. A Job creates
the first Owner after the first install when `admin.email` is set.

## Install

```bash
helm install reputecast oci://ghcr.io/hostrepute/charts/reputecast \
  --namespace reputecast --create-namespace \
  --set publicURL=https://news.example.com \
  --set database.existingSecret=reputecast-db-app \
  --set admin.email=you@example.com \
  --set ingress.enabled=true --set ingress.className=nginx \
  --set 'ingress.tls[0].secretName=reputecast-tls' \
  --set 'ingress.tls[0].hosts[0]=news.example.com'
```

Read the first Owner's temporary password from the install Job, then open the
public URL:

```bash
kubectl logs --namespace reputecast job/reputecast-install
```

At first sign-in the Owner replaces it, adds a passkey or an authenticator
app, saves recovery codes and creates the first workspace. Providers deliver
webhooks to `<publicURL>/hooks/…`, so that address must be reachable from the
internet. A chart from a checkout (`helm install reputecast
deploy/helm/reputecast …`) runs the `latest` image; published charts run the
release they were published with.

## Postgres

### CloudNativePG

With the [CloudNativePG](https://cloudnative-pg.io) operator installed, a
three-instance cluster:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: reputecast-db
  namespace: reputecast
spec:
  instances: 3
  imageName: ghcr.io/cloudnative-pg/postgresql:18
  bootstrap:
    initdb:
      database: reputecast
      owner: reputecast
  storage:
    size: 20Gi
```

The operator writes the connection URL into the `reputecast-db-app` Secret
under `uri`, which is what `database.existingSecret` and
`database.existingSecretKey` (default `uri`) read. Its application user can't
create roles, so SQL segments stay off until an admin adds their role
([SQL segments on managed Postgres](../../../docs/deploy/postgres-tuning.md#sql-segments-on-managed-postgres)).

### A managed database

Amazon RDS, Cloud SQL, Azure Database for PostgreSQL, DigitalOcean or any
other Postgres 16–18. Keep the URL in a Secret:

```bash
kubectl create secret generic reputecast-db --namespace reputecast \
  --from-literal=uri='postgres://reputecast:…@db.example.com:5432/reputecast?sslmode=require'
```

and install with `--set database.existingSecret=reputecast-db`, or pass the
URL as `database.url` and the chart keeps it in a Secret of its own.
Migrations need a direct or session-mode connection, not PgBouncer in
transaction mode.

## Split mode and the bucket

```yaml
mode: split
web:
  replicas: 3
worker:
  replicas: 2
ingest:
  separate: true
storage:
  backend: s3
  s3:
    bucket: reputecast-files
    region: eu-central-1
    existingSecret: reputecast-s3   # access-key-id and secret-access-key
hpa:
  enabled: true
```

Any S3-compatible service works: set `storage.s3.endpoint` (and
`pathStyle: true` for MinIO). Without access keys the default AWS credential
chain is used, so IRSA works through `serviceAccount.annotations`
(`eks.amazonaws.com/role-arn`). `helm install --set mode=split` with
`storage.backend: local` stops with what to do: pods can't share a local
folder.

## The secret key

ReputeCast encrypts provider credentials, OIDC secrets and other secrets at
rest with a 32-byte key. Without `secretKey.existingSecret` the chart
generates one into the `<release>-key` Secret at install, reads it back at
every upgrade, and leaves that Secret in place when the release is
uninstalled. Back it up with the database; a restore needs both:

```bash
kubectl get secret --namespace reputecast reputecast-key -o jsonpath='{.data.secret-key}' | base64 -d
```

Tools that render the chart without a cluster (Argo CD) can't read the key
back and would generate a new one at every sync: create the Secret yourself
(`reputecast secrets generate-key` prints a key) and set
`secretKey.existingSecret`.

## Behind the Ingress

`http.trustedProxies` defaults to `private`, so `X-Forwarded-For` is trusted
from the Ingress controller on the cluster's private networks and nowhere
else. The Ingress host defaults to `publicURL`'s. Imports and uploads can be
larger than some controllers allow by default (ingress-nginx:
`nginx.ingress.kubernetes.io/proxy-body-size: 100m`). [Reverse proxy and
TLS](../../../docs/deploy/reverse-proxy.md) has more.

## The SMTP relay

The relay for transactional email listens in the pods that run the web role
once an instance admin turns it on (Instance admin › Instance settings).
`smtpRelay.service.enabled` publishes its ports through a LoadBalancer
Service, with `externalTrafficPolicy: Local` so the relay sees each client's
address; keep `smtpRelay.starttlsPort` and `smtpRelay.tlsPort` the same as the
ports set in the app (587 and 465). For ports below 1024 the pods get the
`net.ipv4.ip_unprivileged_port_start` sysctl, which Kubernetes counts as safe,
since the container runs as a non-root user without capabilities. A Let's
Encrypt certificate is checked over HTTP: add the relay's hostname to
`ingress.hosts`.

## GeoIP

Readers' countries and cities in campaign reports come from a MaxMind DB file
(GeoLite2-City or -Country, DB-IP Lite) that you download. Mount any volume
that holds it:

```yaml
geoip:
  volume:
    persistentVolumeClaim:
      claimName: geoip
  file: GeoLite2-City.mmdb
```

Replace the file and restart the pods to update it.

## Metrics and network policy

Every pod serves Prometheus metrics on port 9464 (a worker also answers its
health checks there), through the `<release>-metrics` headless Service.
`serviceMonitor.enabled` adds a Prometheus Operator ServiceMonitor.
`networkPolicy.enabled` limits what reaches the pods: the HTTP port from
`networkPolicy.httpFrom` (anywhere by default), metrics from
`networkPolicy.metricsFrom` (the release's namespace by default) and the SMTP
relay's ports when its Service is on. Egress stays open, since sending
providers, DNS and the bucket are wherever they are.

## Upgrades and uninstalling

`helm upgrade` runs the migrations first; a large upgrade (the one that
partitions `delivery_events`, say) can take longer than Helm's five-minute
default, so give it `--timeout 30m`. Until the migrations finish, pods of the
previous release keep serving. Read the
[release notes](https://github.com/hostrepute/reputecast/releases) first.

`helm uninstall` leaves the `<release>-key` Secret and, in single mode, the
`data-<release>-0` PersistentVolumeClaim, so nothing that can't be recreated
is lost. Delete them yourself once you're sure.

## Values

| Value | Default | What it does |
|---|---|---|
| `publicURL` | — (required) | The address people and providers reach. |
| `image.repository` / `.tag` / `.digest` | `ghcr.io/hostrepute/reputecast` / the chart's appVersion / — | The image; a digest wins over the tag. |
| `mode` | `single` | `single` or `split`. |
| `database.url` | — | A Postgres URL, kept in a Secret by the chart. |
| `database.existingSecret` / `.existingSecretKey` | — / `uri` | Or a Secret that holds it. One of the two is required. |
| `secretKey.existingSecret` / `.existingSecretKey` | — / `secret-key` | The encryption key's Secret; generated when empty. |
| `admin.email` / `.name` | — | The first Owner, created after the first install. |
| `admin.passwordSecret` / `.passwordSecretKey` | — / `password` | Their password, instead of a temporary one. |
| `web.replicas` / `worker.replicas` | `2` / `1` | Split mode. |
| `ingest.separate` / `.replicas` | `false` / `2` | Split mode: webhooks in pods of their own, `/hooks/` routed to them. |
| `web.resources` / `worker.resources` / `ingest.resources` | — | Replace `resources` for those pods. |
| `storage.backend` | `local` | `local` or `s3`. |
| `storage.s3.*` | — | `bucket`, `region`, `endpoint`, `pathStyle`, `prefix`, `sse` (`AES256`, `aws:kms`), `kmsKeyID`, `accessKeyID` and `secretAccessKey`, or `existingSecret` with `accessKeyIDKey` / `secretAccessKeyKey`. |
| `http.trustedProxies` | `[private]` | CIDRs, addresses or `private`. |
| `log.level` | `info` | `debug`, `info`, `warn` or `error`. |
| `geoip.volume` / `.file` | — / `GeoLite2-City.mmdb` | The MaxMind DB file's volume. |
| `smtpRelay.service.*`, `smtpRelay.starttlsPort` / `.tlsPort` | off, `LoadBalancer` / `587` / `465` | The SMTP relay's Service. |
| `service.type` / `.port` | `ClusterIP` / `80` | The web Service. |
| `ingress.*` | off | `className`, `annotations`, `hosts` (default: publicURL's host), `tls`. |
| `persistence.*` | on, `10Gi`, `ReadWriteOnce` | Single mode's volume: `storageClass`, `size`, `annotations`, `existingClaim`. |
| `resources` | 100m CPU and 256Mi requested, 1Gi memory limit | Every ReputeCast container. |
| `podSecurityContext` / `securityContext` | non-root 65532, read-only root filesystem, no capabilities, RuntimeDefault seccomp | |
| `serviceAccount.*` | created, no token mounted | `name`, `annotations`, `automount`. |
| `migrations.backoffLimit` / `.resources` | `4` / `resources` | The migration Job. |
| `serviceMonitor.*` | off | `interval`, `scrapeTimeout`, `labels`. |
| `networkPolicy.*` | off | `httpFrom`, `metricsFrom`, `extraIngress`. |
| `pdb.enabled` / `.maxUnavailable` | on / `1` | Split mode's web and ingest pods. |
| `hpa.*` | off, 2–6 replicas at 70% CPU | Split mode's web pods. |
| `extraEnv` / `extraEnvFrom` | — | More settings ([configuration](../../../docs/deploy/configuration.md)). |
| `podAnnotations`, `podLabels`, `nodeSelector`, `tolerations`, `affinity`, `topologySpreadConstraints`, `priorityClassName`, `imagePullSecrets`, `terminationGracePeriodSeconds` | | The usual. |

`values.schema.json` checks types and names, so a misspelt value fails the
install instead of being ignored. `helm test <release>` asks the web Service
for `/readyz`.
