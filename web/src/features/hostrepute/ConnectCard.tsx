import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { cn } from '@/lib/cn'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { ShieldMark, type Connection } from './shared'

const benefits: [string, string][] = [
  ['Send Readiness gets reputation checks', 'Blocklist status of your domains and IPs becomes a Pass / Warn / Block check before every send.'],
  ['Deliverability gets the full picture', 'Postmaster, SNDS and DMARC panels fill in next to the provider data you already see.'],
  ['Alerts land where you work', 'A new listing or a reputation drop shows up in ReputeCast notifications, with the delisting link.'],
]

/**
 * Before connecting (ConnectHostRepute): what HostRepute adds, exactly what
 * is shared and never shared, and the two ways to connect.
 */
export function ConnectCard({
  conn,
  onConnect,
  onUseKey,
  connecting,
  compact,
}: {
  conn?: Connection
  onConnect: () => void
  onUseKey: () => void
  connecting?: boolean
  /** Notes sized to their content (Settings), not the board's tall notes. */
  compact?: boolean
}) {
  const { me, slug } = useWorkspace()
  const can = useCan('integrations:hostrepute')
  const reason = usePermissionReason('integrations:hostrepute')
  const status = conn?.status ?? 'not_connected'
  const reconnect = status === 'revoked' || status === 'expired'
  const shared = conn?.shared
  const nothingYet = Boolean(shared) && !shared?.domains.length && !shared?.ips.length
  return (
    <section className="card" aria-labelledby="h-hr-before">
      <div className="sec">
        <div className="row gap-3">
          <ShieldMark />
          <div className="col gap-0.5">
            <h2 className="h2" id="h-hr-before">
              Connect HostRepute
            </h2>
            <span className="k">Optional · no paywall in ReputeCast · HostRepute is a separate paid service by the same team</span>
          </div>
        </div>
        {status === 'connected' ? (
          <Badge tone="ok">Connected</Badge>
        ) : reconnect ? (
          <Badge tone="warn">Needs reconnecting</Badge>
        ) : (
          <Badge tone="neutral">Not connected</Badge>
        )}
      </div>
      <div className="pad col gap-4">
        <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(min(220px,100%),1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
          {benefits.map(([title, text]) => (
            <div key={title} className="card flex flex-col gap-1 px-3.5 py-3">
              <span className="font-semibold">{title}</span>
              <span className="k whitespace-normal">{text}</span>
            </div>
          ))}
        </div>
        <div className={cn('grid grid-cols-2 gap-3 max-md:grid-cols-1', compact && '[&_.note>:last-child]:basis-auto')}>
          <div className="note g flex-col items-stretch gap-1.5">
            <span className="font-bold">Shared with HostRepute</span>
            <ul className="m-0 flex list-disc flex-col gap-[3px] pl-[18px]">
              {shared?.domains.length ? <li>Sending domains: {shared.domains.join(', ')}</li> : null}
              {shared?.ips.length ? <li>Dedicated IPs: {shared.ips.join(', ')}</li> : null}
              {nothingYet ? (
                <>
                  <li>Nothing yet: your sending domains and dedicated IPs, once you add them</li>
                  <li>Never contacts, email content or reports</li>
                </>
              ) : (
                <li>Nothing else: no contacts, no email content, no reports</li>
              )}
            </ul>
          </div>
          <div className="note i flex-col items-stretch gap-1.5">
            <span className="font-bold">Received by ReputeCast</span>
            <ul className="m-0 flex list-disc flex-col gap-[3px] pl-[18px]">
              <li>Blocklist status across 80+ sources, with delisting links</li>
              <li>Google Postmaster Tools and Microsoft SNDS data</li>
              <li>DMARC report summaries</li>
              <li>Alerts, delivered as ReputeCast notifications</li>
            </ul>
          </div>
        </div>
        {status === 'connected' ? (
          <div className="row gap-2.5">
            <ButtonLink variant="primary" to="/w/$workspace/settings/integrations/hostrepute" params={{ workspace: slug }}>
              Open Settings › Integrations
            </ButtonLink>
            <span className="k">Connected to team “{conn?.teamName}”</span>
          </div>
        ) : (
          <div className="row gap-2.5">
            <Button variant="primary" onClick={onConnect} disabledReason={reason} loading={connecting}>
              <ShieldMark />
              {reconnect ? 'Reconnect with HostRepute' : 'Connect with HostRepute'}
            </Button>
            <Button onClick={onUseKey} disabledReason={reason}>
              Use an API key instead
            </Button>
            <span className="k">
              {can
                ? `Needs the integrations:hostrepute permission · you have it as ${me.workspace.roleName}`
                : 'Needs the integrations:hostrepute permission · ask an Owner or Admin of this workspace'}
            </span>
          </div>
        )}
      </div>
    </section>
  )
}
