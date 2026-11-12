import { useRef } from 'react'
import { links } from '../content/links'
import { useScrollable } from '../lib/scrollable'

export const INSTALLED_LINE = 'ReputeCast is installed'

const commands = `curl -fsSLO ${links.composeUrl}
REPUTECAST_ADMIN_EMAIL=you@example.com docker compose up -d
docker compose logs app | grep -A4 "${INSTALLED_LINE}"`

/** The Helm chart's install, with the values it can't start without. */
export const HELM_COMMAND = `helm install reputecast \\
  ${links.chartUrl} \\
  --set publicURL=https://news.example.com \\
  --set database.url=postgres://… \\
  --set admin.email=you@example.com`

/** One-click installs; each brings its own Postgres. */
export const DEPLOY_PLATFORMS = [
  { name: 'Render', href: links.render },
  { name: 'Railway', href: links.railway },
  { name: 'DigitalOcean', href: links.digitalOcean },
  { name: 'Coolify', href: links.coolify },
] as const

/** The dark install band: one command, four steps, the chart and the platforms, requirements. */
export function Install() {
  const term = useRef<HTMLPreElement>(null)
  const scrollable = useScrollable(term)
  const helm = useRef<HTMLPreElement>(null)
  const helmScrollable = useScrollable(helm)
  return (
    <section className="bg-ink text-white" id="install" aria-labelledby="install-h">
      <div className="wrap flex flex-wrap items-center gap-10 pt-16 pb-16">
        <div className="reveal flex flex-[1_1_380px] flex-col gap-3.5">
          <h2 className="sec text-white" id="install-h">
            Install in one command
          </h2>
          {/* The compose file runs Postgres and the app; there is no Redis (River queues jobs in Postgres). */}
          <p className="m-0 text-[17px] text-dark-muted">
            Docker Compose brings up Postgres, the app and the workers. Upgrades are one pull away. Back up with a pg_dump and the app&apos;s data volume, which
            holds the secret key and uploads.
          </p>
          <ol className="m-0 flex flex-col gap-1.5 pl-5 text-[15px] text-dark-text">
            <li>Download the compose file and start it with your email for the first Owner.</li>
            <li>Open localhost:8080 and sign in with the temporary password from the logs.</li>
            <li>
              Connect a sending provider: most take 10–15 minutes, DNS included. Amazon SES takes longer when ReputeCast doesn&apos;t run on AWS, and a new AWS
              account can wait up to 24 hours for production access.
            </li>
            <li>Import contacts with consent intact, send a test, go.</li>
          </ol>
        </div>
        <div className="reveal flex flex-[1_1_420px] flex-col gap-3">
          <pre
            ref={term}
            className="border border-dark-line bg-dark-surface"
            role={scrollable ? 'region' : undefined}
            aria-label={scrollable ? 'Install commands, scroll sideways' : undefined}
            tabIndex={scrollable ? 0 : undefined}
          >
            {commands}
            <span className="cursor" aria-hidden="true" />
          </pre>
          <div className="flex flex-wrap gap-2.5">
            <a href={links.installGuide} className="btn bg-white hover:bg-white">
              Read the install guide
            </a>
            <a href={links.releases} className="btn border-[#374151] bg-transparent text-white hover:bg-transparent hover:text-white">
              Releases
            </a>
          </div>
          <div className="flex flex-col gap-3 border-t border-dark-line pt-4">
            <span className="text-[13px] text-ink-4">On Kubernetes, with your own Postgres:</span>
            <pre
              ref={helm}
              className="border border-dark-line bg-dark-surface"
              role={helmScrollable ? 'region' : undefined}
              aria-label={helmScrollable ? 'Helm install command, scroll sideways' : undefined}
              tabIndex={helmScrollable ? 0 : undefined}
            >
              {HELM_COMMAND}
            </pre>
            <div role="group" aria-labelledby="install-platforms" className="flex flex-col gap-3">
              <span id="install-platforms" className="text-[13px] text-ink-4">
                On a platform, with Postgres beside it:
              </span>
              <div className="flex flex-wrap gap-2.5">
                {DEPLOY_PLATFORMS.map((p) => (
                  <a key={p.name} href={p.href} className="btn border-[#374151] bg-transparent text-white hover:bg-transparent hover:text-white">
                    {p.name}
                  </a>
                ))}
              </div>
            </div>
          </div>
          <span className="text-[13px] text-ink-4">
            Requirements: Docker 24+ with Compose and a domain you control; for the chart, Kubernetes 1.29+ and Postgres 16–18. Load-tested with a million
            contacts on 4 vCPUs; the docs cover the binary, Helm, TLS, Postgres, backups and scaling.
          </span>
        </div>
      </div>
    </section>
  )
}
