import { useState, type ReactElement } from 'react'
import type { Schemas } from '@/api/client'
import { TableScroll } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { count, dayMonthDate, shortDate } from '@/lib/format'
import { textWidth } from '@/features/scoring/SpreadChart'
import { m } from './messages'
import { dayOf, shortName, signedCount, sourceName } from './sourceText'
import { useScrollRegion } from '@/lib/hooks'

type Chart = Schemas['GrowthChart']
type Series = Schemas['GrowthSeries']
type Day = Schemas['GrowthChartDay']

const FONT = 'IBM Plex Sans,system-ui,sans-serif'

/** The four biggest sources take the mailbox palette's first four colours; Other is grey. */
const seriesColor = ['var(--chart-gmail)', 'var(--chart-outlook)', 'var(--chart-yahoo)', 'var(--chart-icloud)']
export function colorOf(s: Series, i: number): string {
  return s.other ? 'var(--chart-other)' : (seriesColor[i] ?? 'var(--chart-other)')
}

export function seriesName(s: Series, t: ReturnType<typeof useT>): string {
  if (s.other) return t(m.otherWithImports)
  return sourceName(s.source)
}

interface Variant {
  key: 'wide' | 'narrow'
  w: number
  h: number
  left: number
  right: number
  top: number
  bottom: number
  bar: number
  leftY: number
  labelY: number
}

const variants: Variant[] = [
  { key: 'wide', w: 1100, h: 330, left: 44, right: 1092, top: 22, bottom: 274.7, bar: 19.6, leftY: 294, labelY: 318 },
  { key: 'narrow', w: 720, h: 300, left: 44, right: 712, top: 22, bottom: 247.5, bar: 12.5, leftY: 264, labelY: 288 },
]

/** A bucket of days: one day, or a week when the period is long. */
interface Bucket {
  day: string
  days: number
  joined: number[]
  left: number[]
  imports: number
}

export function buckets(days: Day[]): Bucket[] {
  if (days.length <= 120) return days.map((d) => ({ day: d.day, days: 1, joined: d.joined, left: d.left, imports: d.imports }))
  const out: Bucket[] = []
  for (let i = 0; i < days.length; i += 7) {
    const week = days.slice(i, i + 7)
    const n = week[0]!.joined.length
    out.push({
      day: week[0]!.day,
      days: week.length,
      joined: Array.from({ length: n }, (_, k) => week.reduce((s, d) => s + (d.joined[k] ?? 0), 0)),
      left: Array.from({ length: n }, (_, k) => week.reduce((s, d) => s + (d.left[k] ?? 0), 0)),
      imports: week.reduce((s, d) => s + d.imports, 0),
    })
  }
  return out
}

const sum = (a: number[]) => a.reduce((s, v) => s + v, 0)

/** Ticks every 50 (or a rounder step for big days) from the deepest departures to the busiest joins. */
function scale(bs: Bucket[]): { lo: number; hi: number; step: number } {
  const maxJ = Math.max(1, ...bs.map((b) => sum(b.joined)))
  const maxL = Math.max(1, ...bs.map((b) => sum(b.left)))
  const raw = (maxJ + maxL) / 5
  const step = [50, 100, 200, 250, 500, 1000, 2000, 2500, 5000, 10000].find((s) => s >= raw) ?? 10000
  return { lo: -Math.ceil(maxL / step) * step, hi: Math.ceil(maxJ / step) * step, step }
}

function roundedTop(x: number, y: number, w: number, h: number, r: number): string {
  const rr = Math.min(r, h, w / 2)
  return `M${x},${y + h} v${-(h - rr)} a${rr},${rr} 0 0 1 ${rr},${-rr} h${w - 2 * rr} a${rr},${rr} 0 0 1 ${rr},${rr} v${h - rr} z`
}

function roundedBottom(x: number, y: number, w: number, h: number, r: number): string {
  const rr = Math.min(r, h, w / 2)
  return `M${x},${y} v${h - rr} a${rr},${rr} 0 0 0 ${rr},${rr} h${w - 2 * rr} a${rr},${rr} 0 0 0 ${rr},${-rr} v${-(h - rr)} z`
}

/**
 * Joins above the axis and departures below, stacked by source (the four
 * biggest and Other), with the import days as a marker when imports are
 * included. Chart or table.
 */
export function GrowthChart({ chart, view, includeImports, from, to }: { chart: Chart; view: 'chart' | 'table'; includeImports: boolean; from: string; to: string }) {
  const t = useT()
  const bs = buckets(chart.days)
  const joined = sum(chart.series.map((s) => s.joined))
  const left = sum(chart.series.map((s) => s.left))
  const desc = t(m.chartDesc, {
    from: dayMonthDate(dayOf(from)),
    to: dayMonthDate(dayOf(to)),
    joined,
    left,
    series: chart.series.map((s) => t(m.seriesDesc, { name: seriesName(s, t), joined: s.joined, left: s.left })).join(' '),
  })
  const [scrollRef, scroll] = useScrollRegion<HTMLDivElement>(t(m.chartTitle))
  if (view === 'table') return <ChartTable chart={chart} bs={bs} includeImports={includeImports} />
  return (
    <div ref={scrollRef} className="gr-chart" {...scroll}>
      {variants.map((v) => (
        <div key={v.key} className={`gr-v-${v.key}`}>
          <ChartSvg v={v} chart={chart} bs={bs} includeImports={includeImports} desc={desc} />
        </div>
      ))}
    </div>
  )
}

function ChartSvg({ v, chart, bs, includeImports, desc }: { v: Variant; chart: Chart; bs: Bucket[]; includeImports: boolean; desc: string }) {
  const t = useT()
  const [hover, setHover] = useState<number | null>(null)
  const { lo, hi, step } = scale(bs)
  const per = (v.bottom - v.top) / (hi - lo)
  const zero = v.top + hi * per
  const slot = (v.right - v.left) / Math.max(1, bs.length)
  const bar = Math.min(v.bar, slot * 0.6)
  const x0 = (i: number) => v.left + slot * i + (slot - bar) / 2
  const ticks: number[] = []
  for (let n = lo; n <= hi; n += step) ticks.push(n)
  const every = bs.length <= 31 ? 7 : bs.length <= 100 ? 14 : 4
  const tickLabel = (n: number) => (n > 0 ? `+${count(n)}` : n < 0 ? `−${count(-n)}` : '0')
  const marks = includeImports ? bs.map((b, i) => ({ b, i })).filter(({ b }) => b.imports > 0) : []
  const tip = hover === null ? null : tipFor(hover)

  function tipFor(i: number) {
    const b = bs[i]!
    const title = t(m.tipDay, { day: shortDate(dayOf(b.day)), joined: sum(b.joined), left: sum(b.left) })
    const items = chart.series.map((s, k) => ({ color: colorOf(s, k), text: t(m.tipSeries, { name: s.other ? t(m.otherSeries) : shortName(s.source), count: b.joined[k] ?? 0 }) }))
    const colW = Math.max(...items.map((it) => textWidth(it.text, 11.5) + 13)) + 14
    const w = Math.max(textWidth(title, 11.5, 600) + 20, 20 + colW * 2)
    const rows = Math.ceil(items.length / 2)
    const h = 30 + rows * 16
    const colRight = x0(i) + bar + 3
    const x = colRight + 7 + w <= v.w - 4 ? colRight + 7 : Math.max(4, x0(i) - 3 - 7 - w)
    return { title, items, colW, w, h, x }
  }

  return (
    <svg viewBox={`0 0 ${v.w} ${v.h}`} role="img" aria-label={desc} style={{ display: 'block', width: '100%' }} onMouseLeave={() => setHover(null)}>
      {ticks.map((n) => {
        const y = zero - n * per
        return (
          <g key={n}>
            <line x1={v.left} x2={v.right} y1={y} y2={y} stroke={n === 0 ? '#9AA3B2' : '#EEF0F3'} strokeWidth={n === 0 ? 1.2 : 1} />
            <text x={v.left - 8} y={y + 4} textAnchor="end" fontSize={11} fill="#5B6472" fontFamily={FONT}>
              {tickLabel(n)}
            </text>
          </g>
        )
      })}
      <text x={v.left} y={14} fontSize={11} fontWeight={600} fill="#374151" fontFamily={FONT}>
        {t(m.axisJoined)}
      </text>
      <text x={v.left + 4} y={v.leftY} fontSize={11} fontWeight={600} fill="#374151" fontFamily={FONT}>
        {t(m.axisLeft)}
      </text>
      {bs.map((b, i) => {
        const x = x0(i)
        const segs: ReactElement[] = []
        // Joins, from the axis up; the outermost segment is rounded.
        let y = zero
        const lastJ = b.joined.reduce((last, n, k) => (n > 0 ? k : last), -1)
        b.joined.forEach((n, k) => {
          if (n <= 0) return
          const h = n * per
          const top = y - h
          const color = colorOf(chart.series[k]!, k)
          segs.push(k === lastJ ? <path key={`j${k}`} d={roundedTop(x, top, bar, h, 3)} fill={color} /> : <path key={`j${k}`} d={`M${x},${top} h${bar} v${h} h${-bar} z`} fill={color} />)
          y = top - 1.5
        })
        // Departures, from just under the axis down.
        let yd = zero + 1.2
        const lastL = b.left.reduce((last, n, k) => (n > 0 ? k : last), -1)
        b.left.forEach((n, k) => {
          if (n <= 0) return
          const h = n * per
          const color = colorOf(chart.series[k]!, k)
          segs.push(k === lastL ? <path key={`l${k}`} d={roundedBottom(x, yd, bar, h, 3)} fill={color} /> : <path key={`l${k}`} d={`M${x},${yd} h${bar} v${h} h${-bar} z`} fill={color} />)
          yd += h + 1.5
        })
        return (
          <g key={b.day} onMouseEnter={() => setHover(i)}>
            <rect x={v.left + slot * i} y={v.top} width={slot} height={v.bottom - v.top} fill="transparent" />
            {segs}
          </g>
        )
      })}
      {bs.map((b, i) =>
        i % every === 0 ? (
          <text key={b.day} x={x0(i) + bar / 2} y={v.labelY} textAnchor="middle" fontSize={11} fill="#5B6472" fontFamily={FONT}>
            {dayMonthDate(dayOf(b.day))}
          </text>
        ) : null,
      )}
      {marks.map(({ b, i }) => {
        const x = x0(i) + bar / 2
        const label = t(m.importMarker, { day: dayMonthDate(dayOf(b.day)), count: b.imports })
        const w = textWidth(label, 11, 600) + 8
        const lx = x + 6 + w <= v.w - 2 ? x + 6 : x - 6 - w
        return (
          <g key={`imp-${b.day}`} aria-hidden="true">
            <line x1={x} x2={x} y1={20} y2={v.labelY - 12} stroke="#0B1220" strokeWidth={1.4} strokeDasharray="4 3" />
            <rect x={lx} y={18} width={w} height={20} rx={5} fill="#0B1220" />
            <text x={lx + 4} y={32} fontSize={11} fontWeight={600} fill="#FFFFFF" fontFamily={FONT}>
              {label}
            </text>
          </g>
        )
      })}
      {tip && hover !== null ? (
        <g aria-hidden="true" pointerEvents="none">
          <rect x={x0(hover) - 3} y={20} width={bar + 6} height={v.labelY - 16 - 20} rx={4} fill="none" stroke="#0B1220" strokeWidth={1.4} />
          <rect x={tip.x} y={4} width={tip.w} height={tip.h} rx={6} fill="#0B1220" />
          <text x={tip.x + 10} y={22} fontSize={11.5} fill="#FFFFFF" fontWeight={600} fontFamily={FONT}>
            {tip.title}
          </text>
          {tip.items.map((it, k) => {
            const cx = tip.x + 10 + (k % 2) * tip.colW
            const cy = 30 + Math.floor(k / 2) * 16
            return (
              <g key={k}>
                <rect x={cx} y={cy} width={8} height={8} rx={2} fill={it.color} />
                <text x={cx + 13} y={cy + 8} fontSize={11.5} fill="#E5E7EB" fontFamily={FONT}>
                  {it.text}
                </text>
              </g>
            )
          })}
        </g>
      ) : null}
    </svg>
  )
}

function ChartTable({ chart, bs, includeImports }: { chart: Chart; bs: Bucket[]; includeImports: boolean }) {
  const t = useT()
  return (
    <TableScroll label={t(m.chartTitle)}>
      <table className="tbl sc-t11">
        <thead>
          <tr>
            <th scope="col">{t(m.colDay)}</th>
            {chart.series.map((s, k) => (
              <th key={s.key} scope="col" className="r">
                <span className="inline-flex items-center gap-1.5">
                  <span className="gr-sw" style={{ background: colorOf(s, k) }} aria-hidden="true" />
                  {s.other ? t(m.otherSeries) : shortName(s.source)}
                </span>
              </th>
            ))}
            <th scope="col" className="r">
              {t(m.colJoined)}
            </th>
            <th scope="col" className="r">
              {t(m.colLeft)}
            </th>
            <th scope="col" className="r">
              {t(m.colNet)}
            </th>
            {includeImports ? (
              <th scope="col" className="r">
                {t(m.colImports)}
              </th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {bs.map((b) => (
            <tr key={b.day}>
              <td className="nw">{dayMonthDate(dayOf(b.day))}</td>
              {chart.series.map((s, k) => (
                <td key={s.key} className="r num">
                  {count(b.joined[k] ?? 0)}
                </td>
              ))}
              <td className="r num">{count(sum(b.joined))}</td>
              <td className="r num">{count(sum(b.left))}</td>
              <td className="r num">{signedCount(sum(b.joined) - sum(b.left))}</td>
              {includeImports ? <td className="r num">{b.imports ? count(b.imports) : '–'}</td> : null}
            </tr>
          ))}
        </tbody>
      </table>
    </TableScroll>
  )
}
