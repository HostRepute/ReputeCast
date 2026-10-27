import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Segmented } from '@/components/ui/Tabs'
import { useT } from '@/i18n'
import { clock, count, pct } from '@/lib/format'
import { m } from './messages'
import { engagementNote, niceMax, pct1 } from './words'

type Point = Schemas['ReportEngagementPoint']
type Series = 'humanOpens' | 'privacyOpens' | 'clicks'

/** The chart's box: 72 hours across 720 units, the scale over 200. */
const W = 720
const H = 200
const HOURS = 72

/** x of an hour after the send: the time axis is linear over 72 hours. */
export function hourX(h: number): number {
  return (Math.min(Math.max(h, 0), HOURS) / HOURS) * W
}

/** The path of one series as a share of delivered, from the send (0) on a 0 to yMax scale. */
export function seriesPath(points: Point[], key: Series, delivered: number, yMax: number): string {
  if (!points.length || delivered <= 0 || yMax <= 0) return ''
  const all = [{ hour: 0, v: 0 }, ...points.map((p) => ({ hour: p.hour, v: (p[key] / delivered) * 100 }))]
  return all.map((p, i) => `${i === 0 ? 'M' : 'L'}${hourX(p.hour).toFixed(1)} ${Math.max(0, H - (p.v / yMax) * H).toFixed(1)}`).join(' ')
}

const SERIES: { key: Series; color: string; dash?: string }[] = [
  { key: 'privacyOpens', color: '#B4BAC5', dash: '6 4' },
  { key: 'clicks', color: '#0F766E' },
  { key: 'humanOpens', color: '#0B1220' },
]

/** The hours the table shows: when the winner went out, then 6, 24, 48 and 72 hours. */
export function tableHours(e: Schemas['ReportEngagement']): { hour: number; rollout?: boolean }[] {
  const out: { hour: number; rollout?: boolean }[] = []
  const at = e.rollout ? Math.max(1, Math.round(e.rollout.hour)) : undefined
  if (at !== undefined && at < 6 && at <= e.hours) out.push({ hour: at, rollout: true })
  for (const h of [6, 24, 48, 72]) {
    if (h <= e.hours) out.push({ hour: h, rollout: at === h })
  }
  return out
}

/**
 * "Engagement over the first 72 hours": cumulative human opens,
 * privacy-protected opens and human clicks as shares of delivered, on a
 * linear time axis with a y-axis, or as a table (Report board).
 */
export function EngagementChart({ e, startedAt, final, remaining }: { e: Schemas['ReportEngagement']; startedAt?: string; final: boolean; remaining?: number }) {
  const t = useT()
  const [view, setView] = useState<'chart' | 'table'>('chart')
  const last = e.points[e.points.length - 1]
  const rate = (p: Point | undefined, k: Series) => (p && e.delivered > 0 ? (p[k] / e.delivered) * 100 : 0)
  const top = Math.max(rate(last, 'humanOpens'), rate(last, 'privacyOpens'), rate(last, 'clicks'))
  const { max, step } = niceMax(top)
  const ticks: number[] = []
  for (let v = max; v >= 0; v -= step) ticks.push(v)
  const note = engagementNote(e, final, startedAt, remaining)
  const at = { hours: last?.hour ?? 0, human: pct1(rate(last, 'humanOpens')), privacy: pct1(rate(last, 'privacyOpens')), clicks: pct1(rate(last, 'clicks')) }
  const point = (h: number) => e.points.find((p) => p.hour === h)
  const rollout = e.rollout
  const legend = (
    <div className="rep-leg">
      <span>
        <svg width="16" height="4" aria-hidden="true">
          <line x1="0" y1="2" x2="16" y2="2" stroke="#0B1220" strokeWidth="2.5" />
        </svg>
        {t(m.seriesHuman)}
      </span>
      <span>
        <svg width="16" height="4" aria-hidden="true">
          <line x1="0" y1="2" x2="16" y2="2" stroke="#B4BAC5" strokeWidth="2.5" strokeDasharray="4 3" />
        </svg>
        {t(m.seriesPrivacy)}
      </span>
      <span>
        <svg width="16" height="4" aria-hidden="true">
          <line x1="0" y1="2" x2="16" y2="2" stroke="#0F766E" strokeWidth="2.5" />
        </svg>
        {t(m.seriesClicks)}
      </span>
    </div>
  )
  return (
    <section className="card flex min-w-0 flex-col gap-3 px-5 py-4.5" aria-labelledby="rep-eng-h">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <h2 className="h2" id="rep-eng-h">
          {t(m.engagementTitle)}
        </h2>
        {e.available ? (
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            {view === 'chart' ? legend : null}
            <Segmented
              className="rep-seg"
              value={view}
              onChange={setView}
              label={t(m.showEngagementAs)}
              items={[
                { value: 'chart', label: t(m.viewChart) },
                { value: 'table', label: t(m.viewTable) },
              ]}
            />
          </div>
        ) : null}
      </div>
      {!e.available ? (
        <div className="flex h-[220px] flex-col items-center justify-center gap-1.5 rounded-lg bg-ground-3 px-6 text-center">
          <span className="font-semibold">{t(m.noHourly)}</span>
          <span className="k max-w-[440px]">{t(m.noHourlyBody)}</span>
        </div>
      ) : view === 'chart' ? (
        <>
          <div className="flex items-start gap-2 pt-2">
            <div className="rep-yax" aria-hidden="true">
              {ticks.map((v) => (
                <span key={v} style={{ top: H - (v / max) * H }}>
                  {pct(v, 0)}
                </span>
              ))}
            </div>
            <div className="relative min-w-0 flex-1">
              <svg
                viewBox={`0 0 ${W} ${H}`}
                width="100%"
                height={H}
                preserveAspectRatio="none"
                role="img"
                aria-label={t(m.chartAria, { ...at, hours: HOURS, max: pct(max, 0) })}
                className="block overflow-visible"
              >
                {ticks.map((v) => (
                  <line key={v} x1="0" y1={H - (v / max) * H} x2={W} y2={H - (v / max) * H} stroke={v === 0 ? '#D4D8DF' : '#EEF0F3'} vectorEffect="non-scaling-stroke" />
                ))}
                {rollout ? (
                  <line x1={hourX(rollout.hour)} y1="0" x2={hourX(rollout.hour)} y2={H} stroke="#8B94A3" strokeDasharray="3 3" vectorEffect="non-scaling-stroke" />
                ) : null}
                {SERIES.map((s) => (
                  <path
                    key={s.key}
                    d={seriesPath(e.points, s.key, e.delivered, max)}
                    fill="none"
                    stroke={s.color}
                    strokeWidth="2.5"
                    strokeDasharray={s.dash}
                    strokeLinejoin="round"
                    vectorEffect="non-scaling-stroke"
                  />
                ))}
              </svg>
              {rollout && remaining ? (
                <span className="rep-mk" style={{ left: `calc(${((rollout.hour / HOURS) * 100).toFixed(2)}% + 4px)` }} aria-hidden="true">
                  {t(m.rolloutMark, { time: clock(rollout.at), variant: rollout.variant, count: count(remaining) })}
                </span>
              ) : null}
              <div className="rep-xax" aria-hidden="true">
                <span className="x0" style={{ left: 0 }}>
                  {startedAt ? t(m.axisSendAt, { time: clock(startedAt) }) : t(m.axisSend)}
                </span>
                {[12, 24, 36, 48, 60].map((h) => (
                  <span key={h} className={h % 24 === 12 ? 'xo' : undefined} style={{ left: `${(h / HOURS) * 100}%` }}>
                    {t(m.axisHours, { n: h })}
                  </span>
                ))}
                <span className="x9" style={{ left: '100%' }}>
                  {t(m.axisHours, { n: HOURS })}
                </span>
              </div>
            </div>
          </div>
          <span className="k">{t(m.chartShares, at)}</span>
        </>
      ) : (
        <div className="overflow-x-auto">
          <table className="tbl rep-tbl">
            <caption className="vh">{t(m.engagementCaption)}</caption>
            <thead>
              <tr>
                <th scope="col">{t(m.colTimeAfter)}</th>
                <th scope="col" className="r">
                  {t(m.humanOpens)}
                </th>
                <th scope="col" className="r">
                  {t(m.seriesPrivacy)}
                </th>
                <th scope="col" className="r">
                  {t(m.seriesClicks)}
                </th>
              </tr>
            </thead>
            <tbody>
              {tableHours(e).map(({ hour, rollout: r }) => {
                const p = point(hour)
                return (
                  <tr key={hour}>
                    <th scope="row">
                      {r && rollout && remaining
                        ? t(m.rowRollout, { n: hour, time: clock(rollout.at), variant: rollout.variant, count: count(remaining) })
                        : t(m.rowHour, { n: hour })}
                    </th>
                    <td className="r n">{pct1(rate(p, 'humanOpens'))}</td>
                    <td className="r n">{pct1(rate(p, 'privacyOpens'))}</td>
                    <td className="r n">{pct1(rate(p, 'clicks'))}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      {e.available && note ? <span className="k">{note}</span> : null}
    </section>
  )
}
