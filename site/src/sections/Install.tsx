import { useRef } from 'react'
import { links } from '../content/links'
import { useScrollable } from '../lib/scrollable'

export const INSTALLED_LINE = 'ReputeCast is installed'

const commands = `curl -fsSLO ${links.composeUrl}
REPUTECAST_ADMIN_EMAIL=you@example.com docker compose up -d
docker compose logs app | grep -A4 "${INSTALLED_LINE}"`

/** The dark install band: one command, four steps, requirements. */
export function Install() {
  const term = useRef<HTMLPreElement>(null)
  const scrollable = useScrollable(term)
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
          <span className="text-[13px] text-ink-4">
            Requirements: Docker 24+ with Compose and a domain you control. Load-tested with a million contacts on 4 vCPUs; the docs cover the binary, TLS,
            Postgres, backups and scaling.
          </span>
        </div>
      </div>
    </section>
  )
}
