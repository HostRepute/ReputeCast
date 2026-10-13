import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { LegendItem, Sparkline, StackBar } from '@/components/ui/Data'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { delta, netBadge, tierColor, tierName } from './words'

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

/** "13 Sep" from a calendar date of the series ("2026-09-13"), whatever the timezone. */
function dayLabel(d: string): string {
  const [, m, dd] = d.split('-').map(Number)
  return `${dd} ${MONTHS[(m ?? 1) - 1]}`
}

/** "Subscriber growth": the count, its net change, the 30-day line and each list (Main board). */
export function GrowthCard({ a }: { a: Schemas['OverviewAudience'] }) {
  const badge = netBadge(a)
  const values = a.series.map((p) => p.contacts)
  const first = a.series[0]
  const mid = a.series[Math.floor(a.series.length / 2)]
  const last = a.series[a.series.length - 1]
  const lists = a.lists.length
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="ov-growth-h">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="h2" id="ov-growth-h">
          Subscriber growth
        </h2>
        <span className="k">Last {a.days} days</span>
      </div>
      <div className="flex flex-wrap items-baseline gap-3">
        <span className="n text-[30px] tracking-[-0.02em]">{count(a.total)}</span>
        <Badge tone={badge.tone}>{badge.text}</Badge>
        <span className="k">
          {lists} {lists === 1 ? 'list' : 'lists'} · {count(a.total)} unique contacts
        </span>
      </div>
      {values.length > 1 ? (
        <>
          <Sparkline
            values={values}
            height={120}
            label={`Subscribers over the last ${a.days} days, ${values[values.length - 1]! >= values[0]! ? 'rising' : 'falling'} from ${count(values[0])} to ${count(values[values.length - 1])}`}
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
  const { slug } = useWorkspace()
  const label = `Engagement tiers: ${a.tiers.map((t) => `${tierName[t.tier]} ${t.pct}%`).join(', ')}`
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="ov-tiers-h">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="h2" id="ov-tiers-h">
          Engagement tiers
        </h2>
        <Link to="/w/$workspace/audience/health" params={{ workspace: slug }} className="text-[13px] font-medium">
          List health &amp; sunset policy
        </Link>
      </div>
      {a.total > 0 ? (
        <StackBar height={14} label={label} segments={a.tiers.map((t) => ({ value: t.count, color: tierColor[t.tier]! }))} />
      ) : (
        <div className="h-3.5 rounded-[7px] bg-line-2" role="img" aria-label="No contacts yet" />
      )}
      <div className="grid gap-x-4 gap-y-2.5 text-[13px] [grid-template-columns:repeat(2,minmax(0,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
        {a.tiers.map((t) => (
          <LegendItem key={t.tier} color={tierColor[t.tier]!} title={`${tierName[t.tier]} · ${count(t.count)}`} detail={`${t.pct.toFixed(1)}% · ${t.description}`} />
        ))}
      </div>
      <div className="flex items-center gap-2 rounded-lg bg-ground px-3 py-2.5 text-[12.5px] max-md:flex-wrap">
        <Badge tone="neutral">Sunset policy</Badge>
        <span>{a.policy}</span>
      </div>
    </section>
  )
}
