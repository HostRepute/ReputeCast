import { useState, type PointerEvent } from 'react'
import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { count, dayMonthDate, dayMonthYear, monthName } from '@/lib/format'
import { textWidth } from '@/features/scoring/SpreadChart'
import { money } from '../format'
import { mr } from './messages'

type Chart = Schemas['RevenueChart']
type Movement = Schemas['RevenueMovement']

const FONT = 'IBM Plex Sans,system-ui,sans-serif'
const INK = '#0B1220'
const MUTED = '#5B6472'
const GRID = '#EEF0F3'
const LINE = '#0F766E'

/** The movement's colours (EarnOverview): start and end, new, expansion, contraction, churn. */
export const moveColors = { ends: '#104281', new: '#2a78d6', reactivated: '#5598e7', expansion: '#1baf7a', contraction: '#eda100', churned: '#eb6834' }

/** "−$60" with a real minus and a word joiner, as the boards set it. */
function signed(cents: number, currency: string): string {
  if (cents < 0) return `−⁠${money(-cents, currency, 'never')}`
  return `+${money(cents, currency, 'never')}`
}

interface LineVariant {
  key: 'wide' | 'narrow'
  w: number
  h: number
  x0: number
  x1: number
  top: number
  bottom: number
  steps: number
  labelY: number
  markY: number
  months: 'quarters' | 'ends'
}

const lineVariants: LineVariant[] = [
  { key: 'wide', w: 680, h: 330, x0: 56, x1: 670, top: 24, bottom: 306, steps: 4, labelY: 325, markY: 10, months: 'quarters' },
  { key: 'narrow', w: 330, h: 220, x0: 52, x1: 322, top: 46, bottom: 198, steps: 2, labelY: 215, markY: 12, months: 'ends' },
]

/** A round axis top above the biggest value: 2,500, 5,000, 10,000… */
function niceTop(max: number, steps: number): number {
  const raw = Math.max(max, 1) / steps
  const pow = 10 ** Math.floor(Math.log10(raw))
  const step = [1, 2, 2.5, 5, 10].map((m) => m * pow).find((s) => s * steps >= max) ?? 10 * pow
  return step * steps
}

/**
 * MRR over the last 12 months: dashed where it comes from the provider's
 * history before the move, solid since; the numbered marks are the move and
 * the offers that started. A crosshair names the month under the pointer.
 */
export function MrrChart({ chart, currency, desc }: { chart: Chart; currency: string; desc: string }) {
  return (
    <>
      {lineVariants.map((v) => (
        <div key={v.key} className={v.key === 'wide' ? 'er-wide' : 'er-narrow'}>
          <MrrSvg v={v} chart={chart} currency={currency} desc={desc} />
        </div>
      ))}
    </>
  )
}

function MrrSvg({ v, chart, currency, desc }: { v: LineVariant; chart: Chart; currency: string; desc: string }) {
  const t = useT()
  const [hover, setHover] = useState<number | null>(null)
  const pts = chart.points
  if (pts.length < 2) return null
  const t0 = new Date(pts[0]!.at).getTime()
  const t1 = new Date(pts[pts.length - 1]!.at).getTime()
  const max = Math.max(...pts.map((p) => p.mrrCents))
  const top = niceTop(max, 4)
  const x = (at: string) => v.x0 + ((new Date(at).getTime() - t0) / Math.max(1, t1 - t0)) * (v.x1 - v.x0)
  const y = (cents: number) => v.bottom - (cents / top) * (v.bottom - v.top)
  const xy = pts.map((p) => ({ x: x(p.at), y: y(p.mrrCents), p }))
  const histEnd = xy.findIndex((q) => !q.p.history)
  const split = histEnd <= 0 ? 0 : histEnd
  const history = split > 0 ? xy.slice(0, split + 1) : []
  const live = xy.slice(Math.max(0, split))
  const poly = (list: typeof xy) => list.map((q) => `${q.x.toFixed(1)},${q.y.toFixed(1)}`).join(' ')
  const area = `M${xy[0]!.x.toFixed(1)},${v.bottom} ${xy.map((q) => `L${q.x.toFixed(1)},${q.y.toFixed(1)}`).join(' ')} L${xy[xy.length - 1]!.x.toFixed(1)},${v.bottom} Z`
  const gridSteps = v.steps
  const grid = Array.from({ length: gridSteps + 1 }, (_, i) => (top / gridSteps) * i)
  // Marks close to each other step down so their numbers don't collide.
  const marks = chart.marks.map((m, i, all) => {
    const mx = x(m.at)
    const prev = i > 0 ? x(all[i - 1]!.at) : -999
    return { m, mx, my: Math.abs(mx - prev) < 18 ? v.markY + 18 : v.markY }
  })
  const first = xy[0]!
  const last = xy[xy.length - 1]!
  const monthLabel = (at: string, withYear: boolean) => {
    if (v.months === 'ends') return `${monthName(at)} ${new Date(at).getFullYear()}`
    return withYear ? dayMonthYear(at) : dayMonthDate(at)
  }
  const ticks = v.months === 'quarters' ? [3, 6, 9].filter((i) => i < pts.length - 1) : []
  const onMove = (e: PointerEvent<SVGSVGElement>) => {
    const svg = e.currentTarget
    const r = svg.getBoundingClientRect()
    const px = ((e.clientX - r.left) / r.width) * v.w
    let best = 0
    xy.forEach((q, i) => {
      if (Math.abs(q.x - px) < Math.abs(xy[best]!.x - px)) best = i
    })
    setHover(best)
  }
  const tip = hover === null ? null : xy[hover]!
  const tipText = tip ? t(mr.tipMrr, { date: dayMonthYear(tip.p.at), amount: money(tip.p.mrrCents, currency, 'never') }) : ''
  const tipW = tip ? textWidth(tipText, 11.5, 600) + 18 : 0
  const tipX = tip ? Math.min(Math.max(tip.x - tipW / 2, 2), v.w - tipW - 2) : 0
  return (
    <svg viewBox={`0 0 ${v.w} ${v.h}`} width="100%" role="img" aria-label={desc} style={{ display: 'block', maxWidth: '100%' }} onPointerMove={onMove} onPointerLeave={() => setHover(null)}>
      {grid.map((g) => (
        <g key={g}>
          <line x1={v.x0} x2={v.x1} y1={y(g)} y2={y(g)} stroke={GRID} />
          <text x={v.x0 - 8} y={y(g) + 4} textAnchor="end" fontSize={11} fill={MUTED} fontFamily={FONT}>
            {money(g, currency, 'never')}
          </text>
        </g>
      ))}
      <path d={area} fill={LINE} opacity=".08" />
      {history.length > 1 ? <polyline points={poly(history)} fill="none" stroke={LINE} strokeWidth={2} strokeDasharray="5 4" strokeLinejoin="round" strokeLinecap="round" /> : null}
      {live.length > 1 ? <polyline points={poly(live)} fill="none" stroke={LINE} strokeWidth={2.4} strokeLinejoin="round" strokeLinecap="round" /> : null}
      {marks.map(({ m, mx, my }) => (
        <g key={`${m.kind}-${m.n}`} aria-hidden="true">
          <line x1={mx} x2={mx} y1={my} y2={v.bottom} stroke="#8B94A3" strokeDasharray="2 3" />
          <circle cx={mx} cy={my} r={8} fill={INK} />
          <text x={mx} y={my + 4} textAnchor="middle" fontSize={10.5} fontWeight={700} fill="#fff" fontFamily={FONT}>
            {m.n}
          </text>
        </g>
      ))}
      <circle cx={last.x} cy={last.y} r={4} fill="#fff" stroke={LINE} strokeWidth={2.4} />
      <circle cx={first.x} cy={first.y} r={3} fill="#fff" stroke={LINE} strokeWidth={2} />
      <text x={first.x + 2} y={first.y - 10} textAnchor="start" fontSize={11} fontWeight={600} fill={INK} fontFamily={FONT}>
        {money(first.p.mrrCents, currency, 'never')}
      </text>
      <text x={first.x} y={v.labelY} textAnchor="start" fontSize={11} fill={MUTED} fontFamily={FONT}>
        {monthLabel(first.p.at, true)}
      </text>
      {ticks.map((i) => (
        <text key={i} x={xy[i]!.x} y={v.labelY} textAnchor="middle" fontSize={11} fill={MUTED} fontFamily={FONT}>
          {monthLabel(pts[i]!.at, false)}
        </text>
      ))}
      <text x={last.x} y={v.labelY} textAnchor="end" fontSize={11} fill={MUTED} fontFamily={FONT}>
        {monthLabel(last.p.at, true)}
      </text>
      {tip ? (
        <g aria-hidden="true" pointerEvents="none">
          <line x1={tip.x} x2={tip.x} y1={v.top - 6} y2={v.bottom} stroke={INK} strokeWidth={1} />
          <circle cx={tip.x} cy={tip.y} r={4.5} fill={LINE} stroke="#fff" strokeWidth={2} />
          <rect x={tipX} y={Math.max(tip.y - 34, 0)} width={tipW} height={22} rx={6} fill={INK} />
          <text x={tipX + 9} y={Math.max(tip.y - 34, 0) + 15} fontSize={11.5} fontWeight={600} fill="#fff" fontFamily={FONT}>
            {tipText}
          </text>
        </g>
      ) : null}
    </svg>
  )
}

interface BarVariant {
  key: 'wide' | 'narrow'
  w: number
  h: number
  x0: number
  x1: number
  base: number
  gridTop: number
  steps: number
  every: number
  barShare: number
  catY: number
  countY: number
  short: boolean
}

const barVariants: BarVariant[] = [
  { key: 'wide', w: 530, h: 250, x0: 50, x1: 524, base: 210, gridTop: 42.1, steps: 5, every: 1, barShare: 0.58, catY: 228, countY: 243, short: false },
  { key: 'narrow', w: 330, h: 240, x0: 46, x1: 326, base: 200, gridTop: 72.9, steps: 4, every: 2, barShare: 0.58, catY: 218, countY: 233, short: true },
]

/** The axis of the movement: a round step, and a base two steps under the lowest level (or zero). */
export function movementScale(levels: number[]): { base: number; step: number } {
  const lo = Math.min(...levels)
  const hi = Math.max(...levels)
  const nice = [100, 200, 250, 500, 1000, 2000, 2500, 5000, 10000, 20000, 25000, 50000, 100000, 200000, 250000, 500000, 1000000, 2000000, 2500000, 5000000]
  for (const s of nice) {
    const step = s * 100
    const base = Math.max(0, Math.floor(lo / step - 2) * step)
    if ((hi - base) / step <= 5.2) return { base, step }
  }
  return { base: 0, step: Math.max(1, Math.ceil(hi / 5)) }
}

export interface MoveBar {
  key: 'start' | 'new' | 'reactivated' | 'expansion' | 'contraction' | 'churned' | 'end'
  cents: number
  count: number
}

/** The bars of the movement: start, the changes (reactivated only when there were some), end. */
export function moveBars(m: Movement): MoveBar[] {
  const out: MoveBar[] = [{ key: 'start', cents: m.fromCents, count: m.fromMembers }, { key: 'new', cents: m.new.cents, count: m.new.count }]
  if (m.reactivated.count > 0) out.push({ key: 'reactivated', cents: m.reactivated.cents, count: m.reactivated.count })
  out.push(
    { key: 'expansion', cents: m.expansion.cents, count: m.expansion.count },
    { key: 'contraction', cents: m.contraction.cents, count: m.contraction.count },
    { key: 'churned', cents: m.churned.cents, count: m.churned.count },
    { key: 'end', cents: m.toCents, count: m.toMembers },
  )
  return out
}

/** The movement as a waterfall from the start to the end of the period. */
export function MovementChart({ movement, currency, desc }: { movement: Movement; currency: string; desc: string }) {
  return (
    <>
      {barVariants.map((v) => (
        <div key={v.key} className={v.key === 'wide' ? 'er-wide' : 'er-narrow'} style={v.key === 'wide' ? { maxWidth: 600, width: '100%', margin: '0 auto' } : undefined}>
          <MovementSvg v={v} movement={movement} currency={currency} desc={desc} />
        </div>
      ))}
    </>
  )
}

function MovementSvg({ v, movement, currency, desc }: { v: BarVariant; movement: Movement; currency: string; desc: string }) {
  const t = useT()
  const [hover, setHover] = useState<number | null>(null)
  const bars = moveBars(movement)
  // The running level before and after each bar.
  const spans = runningSpans(bars, movement.fromCents)
  const levels = spans.flatMap((s, i) => (bars[i]!.key === 'start' || bars[i]!.key === 'end' ? [s.to] : [s.from, s.to]))
  const { base, step } = movementScale(levels)
  const per = (v.base - v.gridTop) / (v.steps * step)
  const y = (c: number) => v.base - (Math.max(c, base) - base) * per
  const slot = (v.x1 - v.x0) / bars.length
  const bw = slot * v.barShare
  const bx = (i: number) => v.x0 + slot * i + (slot - bw) / 2
  const label = (b: MoveBar) => {
    const names = {
      start: t(mr.mStart),
      new: t(mr.mNew),
      reactivated: v.short ? t(mr.mReactShort) : t(mr.mReactivated),
      expansion: v.short ? t(mr.mExpShort) : t(mr.mExpansion),
      contraction: v.short ? t(mr.mContrShort) : t(mr.mContraction),
      churned: v.short ? t(mr.mChurnShort) : t(mr.mChurned),
      end: t(mr.mEnd),
    }
    return names[b.key]
  }
  const grid: number[] = []
  for (let i = 0; i <= v.steps; i += v.every) grid.push(base + i * step)
  return (
    <svg viewBox={`0 0 ${v.w} ${v.h}`} width="100%" role="img" aria-label={desc} style={{ display: 'block', maxWidth: '100%' }} onPointerLeave={() => setHover(null)}>
      {grid.map((g) => (
        <g key={g}>
          <line x1={v.x0} x2={v.x1} y1={y(g)} y2={y(g)} stroke={GRID} />
          <text x={v.x0 - 8} y={y(g) + 4} textAnchor="end" fontSize={11} fill={MUTED} fontFamily={FONT}>
            {money(g, currency, 'never')}
          </text>
        </g>
      ))}
      {bars.map((b, i) => {
        const s = spans[i]!
        const ends = b.key === 'start' || b.key === 'end'
        const yTop = ends ? y(s.to) : y(Math.max(s.from, s.to))
        const yBottom = ends ? v.base : y(Math.min(s.from, s.to))
        const h = Math.max(yBottom - yTop, b.cents === 0 && !ends ? 0 : 1)
        const x = bx(i)
        const color = ends ? moveColors.ends : moveColors[b.key as keyof typeof moveColors]
        const value = ends ? money(b.cents, currency, 'never') : signed(b.cents, currency)
        const next = i < bars.length - 1 ? s.to : null
        return (
          <g key={b.key} onPointerEnter={() => setHover(i)}>
            <rect x={x - (slot - bw) / 2} y={v.gridTop - 30} width={slot} height={v.base - v.gridTop + 30} fill="transparent" />
            {h > 0 ? <rect x={x} y={yTop} width={bw} height={h} rx={3} fill={color} opacity={hover === null || hover === i ? 1 : 0.55} /> : null}
            {ends && base > 0 ? (
              <path d={`M${(x - 1).toFixed(1)} ${v.base - 9} ${zigzag(bw)}`} fill="none" stroke="#fff" strokeWidth={2.2} aria-hidden="true" />
            ) : null}
            {next !== null ? <line x1={x + bw} x2={x + slot} y1={y(next)} y2={y(next)} stroke="#8B94A3" strokeDasharray="2 2" /> : null}
            <text x={x + bw / 2} y={yTop - 7} textAnchor="middle" fontSize={12} fontWeight={700} fill={INK} fontFamily={FONT}>
              {value}
            </text>
            <text x={x + bw / 2} y={v.catY} textAnchor="middle" fontSize={11} fontWeight={600} fill="#374151" fontFamily={FONT}>
              {label(b)}
            </text>
            <text x={x + bw / 2} y={v.countY} textAnchor="middle" fontSize={11} fill={MUTED} fontFamily={FONT}>
              {count(b.count)}
            </text>
          </g>
        )
      })}
    </svg>
  )
}

/** Each bar's span: the ends from zero, the changes from the level before them. */
function runningSpans(bars: MoveBar[], start: number): { from: number; to: number }[] {
  const out: { from: number; to: number }[] = []
  let level = start
  for (const b of bars) {
    if (b.key === 'start' || b.key === 'end') {
      out.push({ from: 0, to: b.cents })
      continue
    }
    out.push({ from: level, to: level + b.cents })
    level += b.cents
  }
  return out
}

/** The break mark at the bottom of a bar whose axis doesn't start at zero. */
function zigzag(w: number): string {
  const seg = (w + 2) / 4
  return `l${seg.toFixed(1)} -4 l${seg.toFixed(1)} 4 l${seg.toFixed(1)} -4 l${seg.toFixed(1)} 4`
}
