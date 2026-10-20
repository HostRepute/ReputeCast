import { calendarDay, dayRange } from './shared'
import { count } from '@/lib/format'
import { dayMonthDate } from '@/lib/format'

/** One bar: a week (or a day) with a value per series, bottom up. */
export interface BarDatum {
  start: string
  end: string
  partial: boolean
  /** "3 days so far" under a partial bar. */
  sofar?: string
  values: number[]
}

export interface BarSeries {
  key: string
  color: string
}

/** A marker at a day: "Referral program started 23 Sep". */
export interface BarMarker {
  day: string
  text: string
  short: string
}

const font = 'IBM Plex Sans,system-ui,sans-serif'

/** 1, 2 or 5 times a power of ten, at least v. */
function niceStep(v: number): number {
  if (v <= 0) return 1
  const p = 10 ** Math.floor(Math.log10(v))
  for (const f of [1, 2, 5, 10]) if (f * p >= v) return f * p
  return 10 * p
}

/**
 * Bars per week (or day), stacked by series with a 2 px gap, the total over
 * each and a dashed note under a partial one (GrowOverview's chart and
 * Referrals' "Confirmed referrals"). Two drawings: the wide one and the
 * phone's, with shorter labels.
 */
export function WeekBars({ bars, series, label, marker, day = false }: { bars: BarDatum[]; series: BarSeries[]; label: string; marker?: BarMarker; day?: boolean }) {
  return (
    <>
      <div className="max-md:hidden">
        <Drawing bars={bars} series={series} label={label} marker={marker} width={640} height={300} ticks={3} day={day} />
      </div>
      <div className="md:hidden">
        <Drawing bars={bars} series={series} label={label} marker={marker} width={340} height={230} ticks={2} short day={day} />
      </div>
    </>
  )
}

function Drawing({
  bars,
  series,
  label,
  marker,
  width,
  height,
  ticks,
  short,
  day,
}: {
  bars: BarDatum[]
  series: BarSeries[]
  label: string
  marker?: BarMarker
  width: number
  height: number
  ticks: number
  short?: boolean
  day: boolean
}) {
  const left = 40
  const right = width - 8
  const top = 26
  const base = height - 38
  const totals = bars.map((b) => b.values.reduce((s, v) => s + v, 0))
  const max = Math.max(1, ...totals)
  // The tightest of n and n + 1 gridlines: 317 reads 0–400 by 100, not 0–600 by 200.
  let step = niceStep(max / ticks)
  let lines = ticks
  const alt = niceStep(max / (ticks + 1))
  if (alt * (ticks + 1) < step * ticks) {
    step = alt
    lines = ticks + 1
  }
  const yMax = step * lines
  const y = (v: number) => base - (v / yMax) * (base - top)
  const slot = bars.length ? (right - left) / bars.length : right - left
  const bw = Math.max(4, Math.min(short ? 26 : 30, slot * 0.6))
  // Thin the labels when bars are many (days).
  const every = Math.max(1, Math.ceil(bars.length / (short ? 5 : 8)))
  const grid = Array.from({ length: lines + 1 }, (_, i) => i * step)
  let markerX: number | undefined
  if (marker) {
    const md = calendarDay(marker.day).getTime()
    bars.forEach((b, i) => {
      const s = calendarDay(b.start).getTime()
      const e = calendarDay(b.end).getTime()
      if (md >= s && md <= e + 86_400_000 - 1) {
        // In the gap before its bar: "this week and after".
        markerX = left + i * slot + slot / 2 - bw / 2 - Math.min(10, (slot - bw) / 4)
      }
    })
  }
  return (
    <svg width="100%" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={label} style={{ display: 'block', maxWidth: '100%' }}>
      {grid.map((v, i) => (
        <g key={v}>
          <line x1={left} x2={right} y1={y(v)} y2={y(v)} stroke={i === 0 ? 'var(--line-strong)' : 'var(--line-2)'} strokeWidth="1" />
          <text x={left - 8} y={y(v) + 4} textAnchor="end" fontSize="11" fill="var(--ink-3)" fontFamily={font}>
            {count(v)}
          </text>
        </g>
      ))}
      {bars.map((b, i) => {
        const cx = left + i * slot + slot / 2
        const x = cx - bw / 2
        let acc = 0
        const total = totals[i] ?? 0
        const segs: { y0: number; y1: number; color: string }[] = []
        b.values.forEach((v, si) => {
          if (v <= 0) return
          const y0 = y(acc)
          acc += v
          segs.push({ y0, y1: y(acc), color: series[si]?.color ?? 'var(--chart-other)' })
        })
        return (
          <g key={b.start}>
            {segs.map((sg, si) => {
              const last = si === segs.length - 1
              // A 2 px gap between segments.
              const y0 = si === 0 ? sg.y0 : sg.y0 - 2
              const h = Math.max(0, y0 - sg.y1)
              if (!last) return <rect key={si} x={x} y={sg.y1} width={bw} height={h} fill={sg.color} />
              const r = Math.min(4, h)
              const d = `M${x},${y0} L${x},${sg.y1 + r} Q${x},${sg.y1} ${x + r},${sg.y1} L${x + bw - r},${sg.y1} Q${x + bw},${sg.y1} ${x + bw},${sg.y1 + r} L${x + bw},${y0} Z`
              return <path key={si} d={d} fill={sg.color} />
            })}
            {total > 0 ? (
              <text x={cx} y={y(total) - 7} textAnchor="middle" fontSize="11.5" fontWeight="600" fill="var(--ink-nav)" fontFamily={font}>
                {count(total)}
              </text>
            ) : null}
            {i % every === 0 || i === bars.length - 1 ? (
              <text x={cx} y={base + 18} textAnchor="middle" fontSize="11" fill="var(--ink-3)" fontFamily={font}>
                {short || day ? dayMonthDate(calendarDay(b.start)) : dayRange(b.start, b.end)}
              </text>
            ) : null}
            {b.partial && b.sofar && !day ? (
              <text x={cx} y={base + 32} textAnchor="middle" fontSize="10.5" fill="var(--ink-3)" fontFamily={font}>
                {b.sofar}
              </text>
            ) : null}
          </g>
        )
      })}
      {marker && markerX !== undefined ? (
        <g>
          <line x1={markerX} x2={markerX} y1={14} y2={base} stroke="var(--ink-3)" strokeWidth="1" />
          <circle cx={markerX} cy={14} r="3" fill="var(--ink-3)" />
          <text
            x={markerX > width * 0.6 ? markerX - 8 : markerX + 8}
            y={12}
            textAnchor={markerX > width * 0.6 ? 'end' : 'start'}
            fontSize="11"
            fill="var(--ink-nav)"
            fontWeight="600"
            fontFamily={font}
          >
            {short ? marker.short : marker.text}
          </text>
        </g>
      ) : null}
    </svg>
  )
}
