import { useState, type KeyboardEvent, type PointerEvent } from 'react'
import type { Schemas } from '@/api/client'
import { Th } from '@/components/ui/Table'
import { Segmented } from '@/components/ui/Tabs'
import { getI18n, useT } from '@/i18n'
import { weekdayLabel } from '@/i18n/regions'
import { count, pct, timeOfDay, weekStartsOn } from '@/lib/format'
import { m } from './messages'
import { niceMax, shareOf } from './words'

type Breakdown = Schemas['ReportBreakdown']
type Bucket = Schemas['ReportTimeBucket']
type By = 'hour' | 'weekday'

/** The chart's box: the buckets across 720 units, the scale over 200. */
const W = 720
const H = 200

/** "09:00" on the reader's clock. */
function hourText(h: number): string {
  return timeOfDay(`${String(h % 24).padStart(2, '0')}:00`)
}

/** An ISO weekday (Monday = 1) by name: "Mon", or "Monday" in full. */
function dayText(key: number, style: 'short' | 'long'): string {
  return weekdayLabel(getI18n().regionFormat, key % 7, style)
}

/** The weekdays from the first day of the reader's week. */
export function weekOrder(days: Bucket[]): Bucket[] {
  if (weekStartsOn() === 1) return days
  return [...days.filter((d) => d.key === 7), ...days.filter((d) => d.key !== 7)]
}

/** The index of the largest value; the first one on a tie, -1 when every value is 0. */
export function peakIndex(values: number[]): number {
  let at = -1
  values.forEach((v, i) => {
    if (v > 0 && (at < 0 || v > values[at]!)) at = i
  })
  return at
}

/**
 * "When people opened" (Report board, M57 G-12): the share of human opens
 * (bars, --chart-human) and of human clicks (a line, --accent) in each
 * hour of the day or day of the week of the workspace's time zone, on one
 * percentage axis, or as a table. The bars can be read one by one with the
 * pointer or the arrow keys.
 */
export function TimesCard({ b }: { b: Breakdown }) {
  const t = useT()
  const [by, setBy] = useState<By>('hour')
  const [view, setView] = useState<'chart' | 'table'>('chart')
  const [active, setActive] = useState<number>()
  const buckets = by === 'hour' ? b.hours : weekOrder(b.weekdays)
  const opens = buckets.reduce((s, x) => s + x.humanOpens, 0)
  const clicks = buckets.reduce((s, x) => s + x.clicks, 0)
  const openShares = buckets.map((x) => (opens > 0 ? (x.humanOpens / opens) * 100 : 0))
  const clickShares = buckets.map((x) => (clicks > 0 ? (x.clicks / clicks) * 100 : 0))
  const { max, step } = niceMax(Math.max(0, ...openShares, ...clickShares))
  const ticks: number[] = []
  for (let v = max; v >= 0; v -= step) ticks.push(v)
  const n = buckets.length
  const slot = W / n
  const bar = slot * (by === 'hour' ? 0.64 : 0.5)
  const y = (v: number) => H - (v / max) * H
  const when = (x: Bucket, style: 'short' | 'long') => (by === 'hour' ? hourText(x.key) : dayText(x.key, style))
  const span = (x: Bucket) => (by === 'hour' ? t(m.hourSpan, { from: hourText(x.key), to: hourText(x.key + 1) }) : dayText(x.key, 'long'))
  const peak = peakIndex(buckets.map((x) => x.humanOpens))
  const clickPeak = peakIndex(buckets.map((x) => x.clicks))
  const zone = b.timezone.replaceAll('_', ' ')
  const empty = opens === 0
  const valueText = (i: number) => {
    const x = buckets[i]!
    return t(m.timesValue, { when: span(x), opens: count(x.humanOpens), openShare: shareOf(openShares[i]!), clicks: count(x.clicks), clickShare: shareOf(clickShares[i]!) })
  }
  const shown = active ?? Math.max(peak, 0)
  const pick = (e: PointerEvent<HTMLDivElement>) => {
    const r = e.currentTarget.getBoundingClientRect()
    if (r.width > 0) setActive(Math.min(n - 1, Math.max(0, Math.floor(((e.clientX - r.left) / r.width) * n))))
  }
  const onKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const keys: Record<string, number> = { ArrowLeft: shown - 1, ArrowDown: shown - 1, ArrowRight: shown + 1, ArrowUp: shown + 1, Home: 0, End: n - 1 }
    const next = keys[e.key]
    if (next === undefined) return
    e.preventDefault()
    setActive(Math.min(n - 1, Math.max(0, next)))
  }
  const clickLine = buckets.map((_, i) => `${i === 0 ? 'M' : 'L'}${(slot * (i + 0.5)).toFixed(1)} ${y(clickShares[i]!).toFixed(1)}`).join(' ')
  const legend = (
    <div className="rep-leg">
      <span>
        <span className="rep-sw" style={{ background: 'var(--chart-human)' }} aria-hidden="true" />
        {t(m.seriesOpensBars)}
      </span>
      <span>
        <svg width="16" height="4" aria-hidden="true">
          <line x1="0" y1="2" x2="16" y2="2" stroke="var(--accent)" strokeWidth="2.5" />
        </svg>
        {t(m.seriesClicksLine)}
      </span>
    </div>
  )
  const tipLeft = ((shown + 0.5) / n) * 100
  return (
    <section className="card flex min-w-0 flex-col gap-3 px-5 py-4.5" aria-labelledby="rep-time-h">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <h2 className="h2" id="rep-time-h">
          {t(m.timesTitle)}
        </h2>
        {empty ? null : (
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            {view === 'chart' ? legend : null}
            <Segmented
              className="rep-seg"
              value={by}
              onChange={(v) => {
                setBy(v)
                setActive(undefined)
              }}
              label={t(m.timesBy)}
              items={[
                { value: 'hour', label: t(m.byHour) },
                { value: 'weekday', label: t(m.byWeekday) },
              ]}
            />
            <Segmented
              className="rep-seg"
              value={view}
              onChange={setView}
              label={t(m.showTimesAs)}
              items={[
                { value: 'chart', label: t(m.viewChart) },
                { value: 'table', label: t(m.viewTable) },
              ]}
            />
          </div>
        )}
      </div>
      {empty ? (
        <p className="k m-0">{t(m.noHumanOpens)}</p>
      ) : view === 'chart' ? (
        <div className="flex items-start gap-2 pt-2">
          <div className="rep-yax" aria-hidden="true">
            {ticks.map((v) => (
              <span key={v} style={{ top: y(v) }}>
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
              aria-label={t(m.timesAria, {
                by,
                max: pct(max, 0),
                at: peak >= 0 ? when(buckets[peak]!, 'long') : '',
                opens: peak >= 0 ? count(buckets[peak]!.humanOpens) : '0',
                clickAt: clickPeak >= 0 ? when(buckets[clickPeak]!, 'long') : '',
                clicks: clickPeak >= 0 ? count(buckets[clickPeak]!.clicks) : '0',
              })}
              className="block overflow-visible"
            >
              {ticks.map((v) => (
                <line key={v} x1="0" y1={y(v)} x2={W} y2={y(v)} stroke={v === 0 ? 'var(--line-strong)' : 'var(--line-2)'} vectorEffect="non-scaling-stroke" />
              ))}
              {buckets.map((x, i) => (
                <rect
                  key={x.key}
                  x={slot * i + (slot - bar) / 2}
                  y={y(openShares[i]!)}
                  width={bar}
                  height={H - y(openShares[i]!)}
                  fill="var(--chart-human)"
                  opacity={active === undefined || active === i ? 1 : 0.55}
                />
              ))}
              <path d={clickLine} fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
            </svg>
            <div
              className="rep-scrub"
              role="slider"
              tabIndex={0}
              aria-label={t(m.timesRead, { by })}
              aria-valuemin={0}
              aria-valuemax={n - 1}
              aria-valuenow={shown}
              aria-valuetext={valueText(shown)}
              onPointerMove={pick}
              onPointerDown={pick}
              onPointerLeave={() => setActive(undefined)}
              onBlur={() => setActive(undefined)}
              onFocus={() => setActive(shown)}
              onKeyDown={onKey}
            />
            {active !== undefined ? (
              <div className={tipLeft < 18 ? 'rep-tip start' : tipLeft > 82 ? 'rep-tip end' : 'rep-tip'} style={{ left: `${tipLeft}%` }} aria-hidden="true">
                <b>{span(buckets[active]!)}</b>
                <span>
                  {t(m.seriesOpensBars)} {count(buckets[active]!.humanOpens)} · {shareOf(openShares[active]!)}
                </span>
                <span>
                  {t(m.seriesClicksLine)} {count(buckets[active]!.clicks)} · {shareOf(clickShares[active]!)}
                </span>
              </div>
            ) : null}
            {by === 'hour' ? (
              // Every 3 hours: in full, and on phones without the minutes ("03", "3 AM") so eight fit at 390px.
              (['long', 'short'] as const).map((style) => (
                <div key={style} className={`rep-xax rep-hours ${style}`} aria-hidden="true">
                  {[0, 3, 6, 9, 12, 15, 18, 21].map((h) => (
                    <span key={h} className={h === 0 ? 'x0' : undefined} style={{ left: `${(h / 24) * 100}%` }}>
                      {style === 'long' ? hourText(h) : hourText(h).replace(':00', '')}
                    </span>
                  ))}
                </div>
              ))
            ) : (
              <div className="rep-xax" aria-hidden="true">
                {buckets.map((x, i) => (
                  <span key={x.key} style={{ left: `${((i + 0.5) / n) * 100}%` }}>
                    {dayText(x.key, 'short')}
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="tbl rep-tbl">
            <caption className="vh">{t(m.timesTable, { by })}</caption>
            <thead>
              <tr>
                <Th>{by === 'hour' ? t(m.colHour) : t(m.colDay)}</Th>
                <Th align="right">{t(m.humanOpens)}</Th>
                <Th align="right">{t(m.colShareOpens)}</Th>
                <Th align="right">{t(m.humanClicks)}</Th>
                <Th align="right">{t(m.colShareClicks)}</Th>
              </tr>
            </thead>
            <tbody>
              {buckets.map((x, i) => (
                <tr key={x.key}>
                  <th scope="row">{when(x, 'long')}</th>
                  <td className="r n">{count(x.humanOpens)}</td>
                  <td className="r n">{shareOf(openShares[i]!)}</td>
                  <td className="r n">{count(x.clicks)}</td>
                  <td className="r n">{shareOf(clickShares[i]!)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {empty ? null : <span className="k">{t(m.timesNote, { opens: count(opens), clicks: count(clicks), zone })}</span>}
    </section>
  )
}
