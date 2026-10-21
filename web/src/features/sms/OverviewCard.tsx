import { Link } from '@tanstack/react-router'
import { $api } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useChannelSetUp } from '@/features/channels/hooks'
import { useT } from '@/i18n'
import { count, money, monthName, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { BudgetMeter, providerColor } from './common'
import { m } from './messages'
import { m as om } from './overview.messages'
import './sms.css'

const statusTone = { healthy: 'ok', degraded: 'warn', down: 'bad', credentials_rejected: 'bad' } as const
const statusWord = { healthy: m.statusHealthy, degraded: m.statusDegraded, down: m.statusDown, credentials_rejected: m.statusRejected } as const

/**
 * The Overview's provider section adds the SMS providers, with the month's
 * spend against the budget (brief §6). Shown once texting is set up.
 */
export function SmsOverviewCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const sms = useChannelSetUp('sms')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } }, { enabled: sms })
  const d = q.data
  if (!sms || !d || d.providers.length === 0) return null
  const tl = d.tiles
  const month = monthName(tl.month, 'long')
  const order = d.providers.map((p) => p.name)
  return (
    <section className="card flex min-w-0 flex-col" aria-labelledby="ov-sms-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3 className="h2" id="ov-sms-h">
            {t(om.title)}
          </h3>
          <span className="k">{t(om.sub, { month, sent: count(tl.sent), delivered: pct(tl.deliveredPct) })}</span>
        </div>
        <Link to="/w/$workspace/providers/sms" params={{ workspace: slug }} className="text-[13px] font-medium">
          {t(om.manage)}
        </Link>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="flex flex-col gap-2">
          {d.providers.map((p) => (
            <div key={p.id} className="sms-sum items-center">
              <span className="inline-flex flex-wrap items-center gap-2">
                <span className="dot" style={{ color: providerColor(order, p.name) }} aria-hidden="true" />
                <span className="font-semibold">{p.name}</span>
                <Badge tone={statusTone[p.status]} dot>
                  {t(statusWord[p.status])}
                </Badge>
              </span>
              <span className="n">{money(p.spend)}</span>
            </div>
          ))}
        </div>
        {tl.budget > 0 ? (
          <div className="flex flex-col gap-1.5">
            <div className="sms-sum">
              <span className="k">{t(om.budget, { month })}</span>
              <span>
                <b className="n">{money(tl.spend)}</b> <span className="k">{t(om.of, { budget: money(tl.budget, Number.isInteger(tl.budget) ? 0 : 2) })}</span>
              </span>
            </div>
            <BudgetMeter spent={tl.spend} budget={tl.budget} alertPct={tl.alertPct} month={month} />
          </div>
        ) : (
          <span className="k">{t(om.noBudget, { month, spend: money(tl.spend) })}</span>
        )}
      </div>
    </section>
  )
}
