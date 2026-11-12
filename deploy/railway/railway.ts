// ReputeCast on Railway, as Railway Infrastructure as Code
// (https://docs.railway.com/infrastructure-as-code): the release image with
// a volume at /var/lib/reputecast, where the secret key file and uploaded
// files live, and Railway Postgres. deploy/railway/README.md has the steps,
// and the same settings for the template composer.
//
//   npm install --prefix deploy/railway --no-save railway@3
//   railway link
//   REPUTECAST_ADMIN_EMAIL=you@example.com railway config apply --file deploy/railway/railway.ts
import { defineRailway, image, postgres, project, service, volume } from 'railway/iac'

export default defineRailway(() => {
  const owner = process.env.REPUTECAST_ADMIN_EMAIL?.trim()
  if (!owner?.includes('@')) {
    throw new Error("Set REPUTECAST_ADMIN_EMAIL to the first Owner's email address, such as REPUTECAST_ADMIN_EMAIL=you@example.com.")
  }
  const db = postgres('Postgres')
  const data = volume('reputecast-data')
  const app = service('ReputeCast', {
    source: image('ghcr.io/hostrepute/reputecast:latest'),
    // Ready once the database answers and the schema is current; the first
    // start applies every migration.
    healthcheck: '/readyz',
    healthcheckTimeout: 300,
    volumeMounts: { '/var/lib/reputecast': data },
    env: {
      DATABASE_URL: db.env.DATABASE_URL,
      // The domain Railway generates (Settings › Networking), or your own.
      REPUTECAST_PUBLIC_URL: 'https://${{RAILWAY_PUBLIC_DOMAIN}}',
      REPUTECAST_ADMIN_EMAIL: owner,
      REPUTECAST_HTTP__TRUSTED_PROXIES: 'private',
      PORT: '8080',
      // Railway mounts volumes as root; the image runs as a non-root user
      // otherwise and couldn't write the key file.
      RAILWAY_RUN_UID: '0',
    },
  })
  return project('reputecast', { resources: [app, db, data] })
})
