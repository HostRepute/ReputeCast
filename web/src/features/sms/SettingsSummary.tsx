import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { channelIcon } from '@/features/channels/ChannelSwitch'
import { useChannelSetUp } from '@/features/channels/hooks'
import { formatList, T, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { dayMonthYear, money, monthName } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { BudgetMeter, CountryChips, providerColor, regionSentence, useKindName } from './common'
import { m } from './messages'
import { m as sm } from './settings.messages'
import './sms.css'

type Card = Schemas['SmsProviderCard']

const statusTone = { healthy: 'ok', degraded: 'warn', down: 'bad', credentials_rejected: 'bad' } as const
const statusWord = { healthy: m.statusHealthy, degraded: m.statusDegraded, down: m.statusDown, credentials_rejected: m.statusRejected } as const

/**
 * Settings › Providers adds the SMS providers to the summary, and the SMS
 * budget (brief §6). Shown once texting is set up; everything else about
 * SMS lives on Sending providers › SMS.
 */
export function SmsSettingsSummary() {
  const t = useT()
  const { slug } = useWorkspace()
  const sms = useChannelSetUp('sms')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } }, { enabled: sms })
  const d = q.data
  if (!sms || !d || d.providers.length === 0) return null
  const tl = d.tiles
  const month = monthName(tl.month)
  const order = d.providers.map((p) => p.name)
  const params = { workspace: slug }
  return (
    <section className="card flex flex-wrap" aria-labelledby="set-sms-h">
      <div className="flex flex-[1_1_100%] items-center gap-2 border-b border-line-2 px-5 py-3">
        <span className="text-ink-3" aria-hidden="true">
          {channelIcon.sms}
        </span>
        <h2 className="h2" id="set-sms-h">
          {t(sm.title)}
        </h2>
      </div>
      {d.providers.map((p, i) => (
        <SmsProviderColumn key={p.id} card={p} month={month} color={providerColor(order, p.name)} last={i === d.providers.length - 1} />
      ))}
      <div className="flex flex-[1_1_100%] flex-col gap-1.5 border-t border-line-2 px-5 py-3.5">
        {tl.budget > 0 ? (
          <>
            <div className="sms-sum">
              <span className="font-semibold">{t(sm.budget, { month })}</span>
              <span className="n">
                <T msg={sm.spentOf} values={{ spent: money(tl.spend), budget: money(tl.budget, Number.isInteger(tl.budget) ? 0 : 2), b: (x) => <b>{x}</b> }} />
              </span>
            </div>
            <BudgetMeter spent={tl.spend} budget={tl.budget} alertPct={tl.alertPct} month={month} />
            <span className="k">{t(sm.alerts, { alert: tl.alertPct })}</span>
          </>
        ) : (
          <span className="k">{t(sm.noBudget, { month, spend: money(tl.spend) })}</span>
        )}
      </div>
      <div className="flex flex-[1_1_100%] flex-wrap items-center justify-between gap-2 border-t border-line-2 px-5 py-3">
        <span className="k">{t(sm.foot)}</span>
        <div className="flex gap-2">
          <ButtonLink size="sm" to="/w/$workspace/providers/sms/rules" params={params}>
            {t(sm.changeBudget)}
          </ButtonLink>
          <ButtonLink size="sm" variant="primary" to="/w/$workspace/providers/sms" params={params}>
            {t(sm.open)}
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}

function SmsProviderColumn({ card: p, month, color, last }: { card: Card; month: string; color: string; last: boolean }) {
  const t = useT()
  const kindName = useKindName()
  const kinds = formatList(p.senderKinds.map(kindName))
  return (
    <div className={cn('flex flex-[1_1_260px] flex-col gap-2 px-5 py-4.5', !last && 'border-r border-line-2')}>
      <div className="flex items-center justify-between gap-2">
        <span className="inline-flex items-center gap-2 font-bold">
          <span className="dot" style={{ color }} aria-hidden="true" />
          {p.name}
        </span>
        <Badge tone={statusTone[p.status]} dot>
          {t(statusWord[p.status])}
        </Badge>
      </div>
      <div className="sd-kv">
        <span className="k">{t(sm.sendersLabel)}</span>
        <span>{p.senderCount ? t(sm.senders, { count: p.senderCount, kinds }) : t(sm.noSenders)}</span>
      </div>
      <div className="sd-kv">
        <span className="k">{t(sm.routes)}</span>
        <span className="flex flex-col items-end gap-1">
          {p.countries.length ? <CountryChips codes={p.countries} /> : <span>{t(sm.notRouted)}</span>}
          {p.held.length ? <span className="k">{t(sm.held, { countries: regionSentence(p.held) })}</span> : null}
        </span>
      </div>
      <div className="sd-kv">
        <span className="k">{t(sm.connected)}</span>
        <span>{t(sm.connectedOn, { date: dayMonthYear(p.connectedAt), name: p.connectedBy })}</span>
      </div>
      <div className="sd-kv">
        <span className="k">{t(sm.cost, { month })}</span>
        <span className="n">{money(p.spend)}</span>
      </div>
    </div>
  )
}
