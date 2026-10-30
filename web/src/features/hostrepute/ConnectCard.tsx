import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { useT, type MessageDescriptor } from '@/i18n'
import { cn } from '@/lib/cn'
import { roleName } from '@/lib/roles'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { ShieldMark, type Connection } from './shared'

const benefits: [MessageDescriptor, MessageDescriptor][] = [
  [m.benefitReadinessTitle, m.benefitReadinessText],
  [m.benefitDeliverabilityTitle, m.benefitDeliverabilityText],
  [m.benefitAlertsTitle, m.benefitAlertsText],
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
  const t = useT()
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
        <div className="hr-head row gap-3">
          <ShieldMark />
          <div className="col gap-0.5">
            <h2 className="h2" id="h-hr-before">
              {t(m.connectTitle)}
            </h2>
            <span className="k">{t(m.connectLine)}</span>
          </div>
        </div>
        {status === 'connected' ? (
          <Badge tone="ok">{t(m.statusConnected)}</Badge>
        ) : reconnect ? (
          <Badge tone="warn">{t(m.statusNeedsReconnect)}</Badge>
        ) : (
          <Badge tone="neutral">{t(m.statusNotConnected)}</Badge>
        )}
      </div>
      <div className="pad col gap-4">
        <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(min(220px,100%),1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
          {benefits.map(([title, text]) => (
            <div key={title.id} className="card flex flex-col gap-1 px-3.5 py-3">
              <span className="font-semibold">{t(title)}</span>
              <span className="k whitespace-normal">{t(text)}</span>
            </div>
          ))}
        </div>
        <div className={cn('grid grid-cols-2 gap-3 max-md:grid-cols-1', compact && '[&_.note>:last-child]:basis-auto')}>
          <div className="note g flex-col items-stretch gap-1.5">
            <span className="font-bold">{t(m.sharedTitle)}</span>
            <ul className="m-0 flex list-disc flex-col gap-[3px] pl-[18px]">
              {shared?.domains.length ? <li>{t(m.sharedDomains, { domains: shared.domains.join(', ') })}</li> : null}
              {shared?.ips.length ? <li>{t(m.sharedIps, { ips: shared.ips.join(', ') })}</li> : null}
              {nothingYet ? (
                <>
                  <li>{t(m.sharedNothingYet)}</li>
                  <li>{t(m.sharedNever)}</li>
                </>
              ) : (
                <li>{t(m.sharedNothingElse)}</li>
              )}
            </ul>
          </div>
          <div className="note i flex-col items-stretch gap-1.5">
            <span className="font-bold">{t(m.receivedTitle)}</span>
            <ul className="m-0 flex list-disc flex-col gap-[3px] pl-[18px]">
              <li>{t(m.receivedBlocklists)}</li>
              <li>{t(m.receivedPostmaster)}</li>
              <li>{t(m.receivedDmarc)}</li>
              <li>{t(m.receivedAlerts)}</li>
            </ul>
          </div>
        </div>
        {status === 'connected' ? (
          <div className="row gap-2.5">
            <ButtonLink variant="primary" to="/w/$workspace/settings/integrations/hostrepute" params={{ workspace: slug }}>
              {t(m.openSettings)}
            </ButtonLink>
            <span className="k">{t(m.connectedTeam, { team: conn?.teamName ?? '' })}</span>
          </div>
        ) : (
          <div className="row gap-2.5">
            <Button variant="primary" onClick={onConnect} disabledReason={reason} loading={connecting}>
              <ShieldMark />
              {reconnect ? t(m.reconnectWith) : t(m.connectWith)}
            </Button>
            <Button onClick={onUseKey} disabledReason={reason}>
              {t(m.useApiKey)}
            </Button>
            <span className="k">
              {can
                ? t(m.permHave, { role: roleName({ key: me.workspace.roleKey, name: me.workspace.roleName }) })
                : t(m.permAsk)}
            </span>
          </div>
        )}
      </div>
    </section>
  )
}
