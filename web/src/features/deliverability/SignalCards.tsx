import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { count, trimPct } from '@/lib/format'
import { KV } from './AssetCards'
import { cap, dayMonthOf } from './words'

function SourceBadge({ via, stale }: { via: string; stale: boolean }) {
  if (stale) return <Badge tone="warn">Stale</Badge>
  return <Badge tone="ok">{via === 'direct' ? 'Connected directly' : 'Connected'}</Badge>
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

const pair = 'grid gap-x-4 gap-y-2 [grid-template-columns:repeat(2,minmax(0,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]'

/** "100 · 100 · 100%" */
function successLine(p: Schemas['PostmasterPanel']): string {
  const v = (x?: number) => (x == null ? '—' : String(Number(x.toFixed(1))))
  return `${v(p.spf)} · ${v(p.dkim)} · ${v(p.dmarc)}%`
}

/** Spam rate in two-day bars with a table for screen readers. */
function SpamRateChart({ p }: { p: Schemas['PostmasterPanel'] }) {
  const values = p.series.map((s) => s.spamRate ?? 0)
  const max = Math.max(0.1, ...values)
  const known = p.series.filter((s) => s.spamRate != null).map((s) => s.spamRate!)
  const last = known.at(-1)
  const label = known.length
    ? `Gmail spam rate over 30 days, between ${Math.min(...known)} and ${Math.max(...known)} percent, last value ${last} percent`
    : 'Gmail spam rate over 30 days: no data yet'
  return (
    <>
      <div className="flex h-[40px] items-end gap-[3px]" role="img" aria-label={label}>
        {p.series.map((s, i) => (
          <div key={i} className="flex-1 rounded-[2px_2px_0_0] bg-chart-gmail" style={{ height: `${((s.spamRate ?? 0) / max) * 100}%` }} />
        ))}
      </div>
      <table className="vh">
        <caption>Gmail spam rate by two-day period</caption>
        <thead>
          <tr>
            <th scope="col">Days</th>
            <th scope="col">Spam rate</th>
          </tr>
        </thead>
        <tbody>
          {p.series.map((s, i) => (
            <tr key={i}>
              <td>
                {dayMonthOf(s.from)} to {dayMonthOf(s.to)}
              </td>
              <td>{s.spamRate == null ? 'no data' : trimPct(s.spamRate)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="k flex justify-between max-md:flex-wrap">
        <span>{dayMonthOf(p.from)}</span>
        {p.peak ? (
          <span>
            peak {trimPct(p.peak.spamRate)} on {dayMonthOf(p.peak.day)}
            {p.peak.note ? `, ${p.peak.note}` : ''}
          </span>
        ) : null}
        <span>today</span>
      </div>
    </>
  )
}

/** Google Postmaster Tools for the main sending domain. */
export function PostmasterCard({ p }: { p: Schemas['PostmasterPanel'] }) {
  const direct = p.via === 'direct'
  const fbl = p.feedbackLoopFlagged == null ? 'no data' : p.feedbackLoopFlagged === 0 ? 'no flagged campaigns' : `${count(p.feedbackLoopFlagged)} flagged`
  return (
    <PanelCard title={`Google Postmaster Tools · ${p.domain}`} id="h-postmaster" badge={<SourceBadge via={p.via} stale={p.stale} />}>
      <div className={pair}>
        <KV label="Domain reputation" labelClass="k">
          {cap(p.domainReputation)}
        </KV>
        {!direct ? (
          <KV label={`IP reputation${p.ipReputationPool ? `, ${p.ipReputationPool} pool` : ''}`} labelClass="k">
            {cap(p.ipReputation)}
          </KV>
        ) : null}
        <KV label="Spam rate, 30 d" labelClass="k">
          {trimPct(p.spamRate)}
        </KV>
        <KV label="SPF · DKIM · DMARC success" labelClass="k">
          {successLine(p)}
        </KV>
        <KV label="TLS inbound" labelClass="k">
          {trimPct(p.tls)}
        </KV>
        {!direct ? (
          <KV label="Feedback loop" labelClass="k">
            {fbl}
          </KV>
        ) : null}
      </div>
      {direct ? <span className="k">Postmaster and SNDS connect through their own APIs; HostRepute isn't needed for either.</span> : <SpamRateChart p={p} />}
    </PanelCard>
  )
}

const filterColor: Record<string, string> = { green: 'text-ok-fg', yellow: 'text-warn-fg', red: 'text-bad-fg' }

/** Microsoft SNDS for the dedicated IP. */
export function SndsCard({ s }: { s: Schemas['SNDSPanel'] }) {
  const direct = s.via === 'direct'
  const complaint = s.complaintRate == null ? '—' : s.complaintRate < 0.1 ? 'under 0.1%' : trimPct(s.complaintRate)
  return (
    <PanelCard title={`Microsoft SNDS · ${s.ip}`} id="h-snds" badge={<SourceBadge via={s.via} stale={s.stale} />}>
      <div className={pair}>
        <KV label="Filter result" labelClass="k" valueClass={filterColor[s.filterResult]}>
          {cap(s.filterResult)}
        </KV>
        <KV label="Messages, 7 d" labelClass="k">
          {count(s.messages7d)}
        </KV>
        <KV label="Complaint rate" labelClass="k">
          {complaint}
        </KV>
        <KV label="Spam trap hits" labelClass="k">
          {count(s.trapHits)}
        </KV>
        {!direct ? (
          <>
            <KV label="JMRP feedback" labelClass="k">
              {s.jmrpReports === 1 ? '1 report' : `${count(s.jmrpReports)} reports`}
            </KV>
            <KV label="Shared pools" labelClass="k">
              not covered by SNDS
            </KV>
          </>
        ) : null}
      </div>
    </PanelCard>
  )
}
