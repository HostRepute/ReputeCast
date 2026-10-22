import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { LegendItem, Sparkline, StackBar } from '@/components/ui/Data'
import { BySourceLink } from '@/features/growth/BySource'
import { T, useT } from '@/i18n'
import { count, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { dayLabel, delta, netBadge, tierColor, tierName } from './words'

/**
 * "Subscriber growth" (Main board): the lists' subscribers, their net change
 * (joins minus departures, as on List health), the line over the period and
 * each list; the unique contacts beside them.
 */
export function GrowthCard({ a }: { a: Schemas['OverviewAudience'] }) {
  const t = useT()
  const badge = netBadge(a)
  const values = a.series.map((p) => p.subscribers)
  const first = a.series[0]
  const mid = a.series[Math.floor(a.series.length / 2)]
  const last = a.series[a.series.length - 1]
  const lists = a.lists.length
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="ov-growth-h">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="h2" id="ov-growth-h">
          {t(m.subscriberGrowth)}
        </h2>
        <span className="k">
          <T msg={m.lastDays} values={{ days: a.days, link: <BySourceLink /> }} />
        </span>
      </div>
      <div className="flex flex-wrap items-baseline gap-3">
        <span className="n text-[30px] tracking-[-0.02em]">{count(a.subscribers)}</span>
        <Badge tone={badge.tone}>{badge.text}</Badge>
        <span className="k">{t(lists ? m.listsUnique : m.noListsUnique, { lists, n: a.total, total: count(a.total) })}</span>
      </div>
      {values.length > 1 ? (
        <>
          <Sparkline
            values={values}
            height={120}
            label={t(values[values.length - 1]! >= values[0]! ? m.sparkRising : m.sparkFalling, { days: a.days, from: count(values[0]), to: count(values[values.length - 1]) })}
          />
          <div className="k flex justify-between">
            <span>{first ? dayLabel(first.day) : ''}</span>
            <span>{mid ? dayLabel(mid.day) : ''}</span>
            <span>{last ? dayLabel(last.day) : ''}</span>
          </div>
        </>
      ) : null}
      {lists ? (
        <div className="grid gap-3 border-t border-line-2 pt-3 [grid-template-columns:repeat(3,minmax(0,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
          {a.lists.map((l) => (
            <div key={l.id}>
              <div className="k">{l.name}</div>
              <div className="n">{count(l.contacts)}</div>
              <div className="k" style={{ color: l.net > 0 ? '#166534' : l.net < 0 ? '#991B1B' : undefined }}>
                {delta(l.net)}
              </div>
            </div>
          ))}
        </div>
      ) : null}
    </section>
  )
}

/** "Engagement tiers": the four tiers and the sunset policy (Main board). */
export function TiersCard({ a }: { a: Schemas['OverviewAudience'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const label = t(m.tiersLabel, { tiers: a.tiers.map((x) => t(m.tierShare, { tier: tierName[x.tier] ?? x.tier, percent: `${x.pct}%` })).join(', ') })
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="ov-tiers-h">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="h2" id="ov-tiers-h">
          {t(m.engagementTiers)}
        </h2>
        <Link to="/w/$workspace/audience/health" params={{ workspace: slug }} className="text-[13px] font-medium">
          {t(m.listHealthLink)}
        </Link>
      </div>
      {a.total > 0 ? (
        <StackBar height={14} label={label} segments={a.tiers.map((x) => ({ value: x.count, color: tierColor[x.tier]! }))} />
      ) : (
        <div className="h-3.5 rounded-[7px] bg-line-2" role="img" aria-label={t(m.noContacts)} />
      )}
      <div className="grid gap-x-4 gap-y-2.5 text-[13px] [grid-template-columns:repeat(2,minmax(0,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
        {a.tiers.map((x) => (
          <LegendItem
            key={x.tier}
            className="max-md:flex-wrap"
            color={tierColor[x.tier]!}
            title={`${tierName[x.tier]} · ${count(x.count)}`}
            detail={`${pct(x.pct, 1)} · ${x.description}`}
          />
        ))}
      </div>
      <div className="flex items-center gap-2 rounded-lg bg-ground px-3 py-2.5 text-[12.5px] max-md:flex-wrap">
        <Badge tone="neutral">{t(m.sunsetPolicy)}</Badge>
        <span>{a.policy}</span>
      </div>
    </section>
  )
}
