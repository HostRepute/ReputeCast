import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { count, decimal, pct, trimPct } from '@/lib/format'
import { KV } from './AssetCards'
import { m } from './messages'
import { dayMonthOf, filterWord, reputationWord } from './words'

function SourceBadge({ via, stale }: { via: string; stale: boolean }) {
  const t = useT()
  if (stale) return <Badge tone="warn">{t(m.stale)}</Badge>
  return <Badge tone="ok">{via === 'direct' ? t(m.connectedDirectly) : t(m.connected)}</Badge>
}

function PanelCard({ title, id, badge, children }: { title: string; id: string; badge: ReactNode; children: ReactNode }) {
  return (
    <section className="card flex flex-col gap-2.5 px-4.5 py-4" aria-labelledby={id}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2" id={id}>
          {title}
        </h2>
        {badge}
      </div>
      {children}
    </section>
  )
}

const pair = 'dl-pair grid gap-x-4 gap-y-2 [grid-template-columns:repeat(2,minmax(0,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]'

/** A number with up to `digits` decimals, without trailing zeros, in the region format: 100, 99.5, 0.05. */
function num(x: number, digits: number): string {
  const n = Number(x.toFixed(digits))
  return decimal(n, (String(n).split('.')[1] ?? '').length)
}

/** "100 · 100 · 100%" */
function successLine(p: Schemas['DeliverabilityPostmaster']): string {
  const v = (x?: number) => (x == null ? '—' : num(x, 1))
  const last = p.dmarc == null ? '—%' : pct(Number(p.dmarc.toFixed(1)), Number.isInteger(Number(p.dmarc.toFixed(1))) ? 0 : 1)
  return `${v(p.spf)} · ${v(p.dkim)} · ${last}`
}

/** Spam rate in two-day bars with a table for screen readers. */
function SpamRateChart({ p }: { p: Schemas['DeliverabilityPostmaster'] }) {
  const t = useT()
  const values = p.series.map((s) => s.spamRate ?? 0)
  const max = Math.max(0.1, ...values)
  const known = p.series.filter((s) => s.spamRate != null).map((s) => s.spamRate!)
  const last = known.at(-1)
  const label = known.length
    ? t(m.spamChartLabel, { min: num(Math.min(...known), 3), max: num(Math.max(...known), 3), last: num(last!, 3) })
    : t(m.spamChartNoData)
  return (
    <>
      <div className="flex h-[40px] items-end gap-[3px]" role="img" aria-label={label}>
        {p.series.map((s, i) => (
          <div key={i} className="flex-1 rounded-[2px_2px_0_0] bg-chart-gmail" style={{ height: `${((s.spamRate ?? 0) / max) * 100}%` }} />
        ))}
      </div>
      <table className="vh">
        <caption>{t(m.spamTableCaption)}</caption>
        <thead>
          <tr>
            <th scope="col">{t(m.colDays)}</th>
            <th scope="col">{t(m.colSpamRate)}</th>
          </tr>
        </thead>
        <tbody>
          {p.series.map((s, i) => (
            <tr key={i}>
              <td>{t(m.dayRange, { from: dayMonthOf(s.from), to: dayMonthOf(s.to) })}</td>
              <td>{s.spamRate == null ? t(m.noData) : trimPct(s.spamRate)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="k flex justify-between max-md:flex-wrap">
        <span>{dayMonthOf(p.from)}</span>
        {p.peak ? (
          <span>
            {p.peak.note
              ? t(m.peakNote, { rate: trimPct(p.peak.spamRate), date: dayMonthOf(p.peak.day), note: p.peak.note })
              : t(m.peak, { rate: trimPct(p.peak.spamRate), date: dayMonthOf(p.peak.day) })}
          </span>
        ) : null}
        <span>{t(m.today)}</span>
      </div>
    </>
  )
}

/** Google Postmaster Tools for the main sending domain. */
export function PostmasterCard({ p }: { p: Schemas['DeliverabilityPostmaster'] }) {
  const t = useT()
  const direct = p.via === 'direct'
  const fbl = p.feedbackLoopFlagged == null ? t(m.noData) : p.feedbackLoopFlagged === 0 ? t(m.noFlagged) : t(m.flagged, { count: count(p.feedbackLoopFlagged) })
  return (
    <PanelCard title={`Google Postmaster Tools · ${p.domain}`} id="h-postmaster" badge={<SourceBadge via={p.via} stale={p.stale} />}>
      <div className={pair}>
        <KV label={t(m.domainReputation)} labelClass="k">
          {reputationWord(p.domainReputation)}
        </KV>
        {!direct ? (
          <KV label={p.ipReputationPool ? t(m.ipReputationPool, { pool: p.ipReputationPool }) : t(m.ipReputation)} labelClass="k">
            {reputationWord(p.ipReputation)}
          </KV>
        ) : null}
        <KV label={t(m.spamRate30d)} labelClass="k">
          {trimPct(p.spamRate)}
        </KV>
        <KV label={t(m.authSuccess)} labelClass="k">
          {successLine(p)}
        </KV>
        <KV label={t(m.tlsInbound)} labelClass="k">
          {trimPct(p.tls)}
        </KV>
        {!direct ? (
          <KV label={t(m.feedbackLoop)} labelClass="k">
            {fbl}
          </KV>
        ) : null}
      </div>
      {direct ? <span className="k">{t(m.directNote)}</span> : <SpamRateChart p={p} />}
    </PanelCard>
  )
}

const filterColor: Record<string, string> = { green: 'text-ok-fg', yellow: 'text-warn-fg', red: 'text-bad-fg' }

/** Microsoft SNDS for the dedicated IP. */
export function SndsCard({ s }: { s: Schemas['DeliverabilitySNDS'] }) {
  const t = useT()
  const direct = s.via === 'direct'
  const complaint = s.complaintRate == null ? '—' : s.complaintRate < 0.1 ? t(m.underRate, { rate: trimPct(0.1) }) : trimPct(s.complaintRate)
  return (
    <PanelCard title={`Microsoft SNDS · ${s.ip}`} id="h-snds" badge={<SourceBadge via={s.via} stale={s.stale} />}>
      <div className={pair}>
        <KV label={t(m.filterResultLabel)} labelClass="k" valueClass={filterColor[s.filterResult]}>
          {filterWord(s.filterResult)}
        </KV>
        <KV label={t(m.messages7d)} labelClass="k">
          {count(s.messages7d)}
        </KV>
        <KV label={t(m.complaintRate)} labelClass="k">
          {complaint}
        </KV>
        <KV label={t(m.spamTrapHits)} labelClass="k">
          {count(s.trapHits)}
        </KV>
        {!direct ? (
          <>
            <KV label={t(m.jmrpFeedback)} labelClass="k">
              {t(m.jmrpReports, { count: s.jmrpReports })}
            </KV>
            <KV label={t(m.sharedPools)} labelClass="k">
              {t(m.notCoveredBySnds)}
            </KV>
          </>
        ) : null}
      </div>
    </PanelCard>
  )
}
