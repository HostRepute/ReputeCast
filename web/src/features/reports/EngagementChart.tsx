import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { clock, pct } from '@/lib/format'
import { m } from './messages'
import { engagementNote, pct1 } from './words'

/** Hours where the time axis bends: each stretch gets a quarter of the width. */
const BREAKS = [0, 6, 24, 48, 72]
const X0 = 40
const X1 = 700
const BASE = 190
const TOP = 10

/** x of an hour on the chart (0–6 h, 6–24 h, 24–48 h and 48–72 h each a quarter). */
export function hourX(h: number): number {
  const seg = (X1 - X0) / (BREAKS.length - 1)
  for (let i = 1; i < BREAKS.length; i++) {
    if (h <= BREAKS[i]!) {
      const a = BREAKS[i - 1]!
      const b = BREAKS[i]!
      return X0 + seg * (i - 1 + (h - a) / (b - a))
    }
  }
  return X1
}

type Series = 'humanOpens' | 'machineOpens' | 'clicks'

/** The path of one series as a share of delivered; yMax is the top of the chart. */
export function seriesPath(points: Schemas['ReportEngagementPoint'][], key: Series, delivered: number, yMax: number): string {
  if (!points.length || delivered <= 0 || yMax <= 0) return ''
  return points
    .map((p, i) => {
      const v = (p[key] / delivered) * 100
      const y = BASE - (v / yMax) * (BASE - TOP)
      return `${i === 0 ? 'M' : 'L'}${hourX(p.hour).toFixed(1)} ${Math.max(TOP, y).toFixed(1)}`
    })
    .join(' ')
}

const SERIES: { key: Series; label: MessageDescriptor; color: string }[] = [
  { key: 'humanOpens', label: m.seriesHuman, color: 'var(--chart-human)' },
  { key: 'machineOpens', label: m.machineOpens, color: 'var(--chart-machine)' },
  { key: 'clicks', label: m.clicks, color: 'var(--accent)' },
]

/** "Engagement over the first 72 hours": cumulative human opens, machine opens and clicks. */
export function EngagementChart({ e, startedAt, final }: { e: Schemas['ReportEngagement']; startedAt?: string; final: boolean }) {
  const t = useT()
  const b = (c: ReactNode[]) => <b>{c}</b>
  const last = e.points[e.points.length - 1]
  const rate = (k: Series) => (last && e.delivered > 0 ? (last[k] / e.delivered) * 100 : 0)
  const top = Math.max(rate('humanOpens'), rate('machineOpens'), rate('clicks'))
  // The top series ends at 93% of the height, as on the board.
  const yMax = top > 0 ? top / 0.93 : 1
  const note = engagementNote(e, final)
  const reach = { human: pct1(rate('humanOpens')), hours: last?.hour ?? 0, machine: pct1(rate('machineOpens')), clicks: pct1(rate('clicks')) }
  const label = last
    ? e.humanWithin6h !== undefined
      ? t(m.chartLabelWithin, { ...reach, within: pct(Math.round(e.humanWithin6h), 0) })
      : t(m.chartLabel, reach)
    : t(m.noEngagement)
  const sample = [1, 6, 24, 48, 72].map((h) => e.points.find((p) => p.hour === h)).filter((p): p is Schemas['ReportEngagementPoint'] => Boolean(p))
  return (
    <section className="card flex min-w-0 flex-[2_1_560px] flex-col gap-3 px-5 py-4.5" aria-labelledby="rep-eng-h">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2" id="rep-eng-h">
          {t(m.engagementTitle)}
        </h2>
        <div className="flex gap-3.5 text-[12px] max-md:flex-wrap">
          {SERIES.map((s) => (
            <span key={s.key} className="flex items-center gap-1.5">
              <span className="h-0.5 w-3.5" style={{ background: s.color }} aria-hidden="true" />
              {t(s.label)}
            </span>
          ))}
        </div>
      </div>
      {e.available ? (
        <>
          <svg viewBox="0 0 720 220" width="100%" height="220" preserveAspectRatio="none" role="img" aria-label={label}>
            <line x1="40" y1="190" x2="700" y2="190" stroke="#E3E6EB" vectorEffect="non-scaling-stroke" />
            <line x1="40" y1="130" x2="700" y2="130" stroke="#EEF0F3" vectorEffect="non-scaling-stroke" />
            <line x1="40" y1="70" x2="700" y2="70" stroke="#EEF0F3" vectorEffect="non-scaling-stroke" />
            <line x1="40" y1="10" x2="700" y2="10" stroke="#EEF0F3" vectorEffect="non-scaling-stroke" />
            {SERIES.map((s) => (
              <path
                key={s.key}
                d={seriesPath(e.points, s.key, e.delivered, yMax)}
                fill="none"
                stroke={s.key === 'machineOpens' ? '#B4BAC5' : s.key === 'clicks' ? '#0F766E' : '#0B1220'}
                strokeWidth="2.5"
                strokeLinejoin="round"
                vectorEffect="non-scaling-stroke"
              />
            ))}
          </svg>
          {/* A table ignores width: 1px, so the wrapper hides it without widening the page. */}
          <div className="vh">
            <table>
              <caption>{t(m.chartCaption)}</caption>
              <thead>
                <tr>
                  <th scope="col">{t(m.colHoursAfter)}</th>
                  <th scope="col">{t(m.humanOpens)}</th>
                  <th scope="col">{t(m.machineOpens)}</th>
                  <th scope="col">{t(m.clicks)}</th>
                </tr>
              </thead>
              <tbody>
                {sample.map((p) => (
                  <tr key={p.hour}>
                    <td>{p.hour}</td>
                    <td>{pct1((p.humanOpens / e.delivered) * 100)}</td>
                    <td>{pct1((p.machineOpens / e.delivered) * 100)}</td>
                    <td>{pct1((p.clicks / e.delivered) * 100)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="flex items-start justify-between">
            <div className="k flex flex-1 justify-between" aria-hidden="true">
              <span>{startedAt ? t(m.axisSendAt, { time: clock(startedAt) }) : t(m.axisSend)}</span>
              <span>{t(m.axisHours, { n: 6 })}</span>
              <span>{t(m.axisHours, { n: 24 })}</span>
              <span>{t(m.axisHours, { n: 48 })}</span>
              <span>{t(m.axisHours, { n: 72 })}</span>
            </div>
            <div className="ml-4 flex flex-col items-end text-[12px]">
              <span>
                <T msg={m.legendHuman} values={{ rate: pct1(rate('humanOpens')), b }} />
              </span>
              <span className="text-ink-3">
                <T msg={m.legendMachine} values={{ rate: pct1(rate('machineOpens')), b }} />
              </span>
              <span className="text-accent">
                <T msg={m.legendClicks} values={{ rate: pct1(rate('clicks')), b }} />
              </span>
            </div>
          </div>
          {note ? <span className="k">{note}</span> : null}
        </>
      ) : (
        <div className="flex h-[220px] flex-col items-center justify-center gap-1.5 rounded-lg bg-ground-3 px-6 text-center">
          <span className="font-semibold">{t(m.noHourly)}</span>
          <span className="k max-w-[440px]">{t(m.noHourlyBody)}</span>
        </div>
      )}
    </section>
  )
}
