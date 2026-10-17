import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { TableScroll } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { count, pct } from '@/lib/format'
import { bandColor, bandName, bandOf, bandRange, bandOrder, type Band, type Bands } from './bands'
import { m } from './messages'

type Bucket = Schemas['ScoringBucket']

const FONT = 'IBM Plex Sans,system-ui,sans-serif'
let canvas: HTMLCanvasElement | undefined

/** The width of a label in the chart's font (an estimate where there is no canvas, as in tests). */
export function textWidth(text: string, size: number, weight = 400): number {
  try {
    canvas ??= document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    if (ctx) {
      ctx.font = `${weight} ${size}px ${FONT}`
      return ctx.measureText(text).width
    }
  } catch {
    // no canvas
  }
  return text.length * size * 0.56
}

interface Variant {
  key: 'desk' | 'tab' | 'phone'
  w: number
  h: number
  left: number
  right: number
  top: number
  base: number
  bar: number
  r: number
  ticks: number[]
  short: boolean
}

const variants: Variant[] = [
  { key: 'desk', w: 540, h: 250, left: 50, right: 532, top: 34, base: 224, bar: 24, r: 4, ticks: [0, 5000, 10000, 15000, 20000], short: false },
  { key: 'tab', w: 720, h: 270, left: 50, right: 712, top: 34, base: 244, bar: 24, r: 4, ticks: [0, 5000, 10000, 15000, 20000], short: false },
  { key: 'phone', w: 340, h: 214, left: 32, right: 334, top: 30, base: 192, bar: 16, r: 3, ticks: [0, 10000, 20000], short: true },
]

/** Ticks of 5,000 up to the next one above the tallest bar (at least 20,000). */
function scaleTop(max: number, short: boolean): { top: number; ticks: number[] } {
  const step = short ? 10000 : 5000
  const top = Math.max(20000, Math.ceil(max / step) * step)
  const ticks: number[] = []
  for (let v = 0; v <= top; v += step) ticks.push(v)
  return { top, ticks }
}

function shortCount(v: number): string {
  return v >= 1000 ? `${v / 1000}k` : String(v)
}

function barPath(x: number, y: number, w: number, base: number, r: number): string {
  const h = base - y
  const rr = Math.min(r, h, w / 2)
  return `M${x},${base} L${x},${y + rr} Q${x},${y} ${x + rr},${y} L${x + w - rr},${y} Q${x + w},${y} ${x + w},${y + rr} L${x + w},${base} Z`
}

/**
 * How contacts spread over the ten buckets (ScoringRules, ScoringStates):
 * bars coloured by band, the two thresholds as lines, a tooltip on hover
 * and focus, and a table view. With `after`, the edited spread is drawn
 * in band colours over today's in grey.
 */
export function SpreadChart({ view, buckets, after, bands }: { view: 'chart' | 'table'; buckets: Bucket[]; after?: Bucket[]; bands: Bands }) {
  const t = useT()
  const shown = after ?? buckets
  const desc = t(after ? m.spreadAfterDesc : m.spreadDesc, { buckets: shown.map((b) => `${b.from}–${b.to} ${count(b.contacts)}`).join(', ') })
  return view === 'chart' ? <Chart buckets={buckets} after={after} bands={bands} desc={desc} /> : <SpreadTable buckets={buckets} after={after} bands={bands} />
}

function Chart({ buckets, after, bands, desc }: { buckets: Bucket[]; after?: Bucket[]; bands: Bands; desc: string }) {
  return (
    <>
      {variants.map((v) => (
        <div key={v.key} className={`sc-chart sc-v-${v.key}`} style={{ padding: v.key === 'phone' ? '10px 12px 4px' : '10px 14px 4px' }}>
          <ChartSvg v={v} buckets={buckets} after={after} bands={bands} desc={desc} />
        </div>
      ))}
    </>
  )
}

function ChartSvg({ v, buckets, after, bands, desc }: { v: Variant; buckets: Bucket[]; after?: Bucket[]; bands: Bands; desc: string }) {
  const t = useT()
  const [hover, setHover] = useState<number | null>(null)
  const shown = after ?? buckets
  const max = Math.max(...shown.map((b) => b.contacts), ...buckets.map((b) => b.contacts))
  const { top: maxV, ticks } = scaleTop(max, v.short)
  const plot = v.right - v.left
  const slot = plot / 10
  const y = (n: number) => v.base - ((v.base - v.top) * n) / maxV
  const xOfScore = (s: number) => v.left + (plot * s) / 100
  const bandFor = (b: Bucket): Band => bandOf(b.from, bands)
  // Legend items centred on their band's stretch of the axis.
  const legend = bandOrder.map((band) => {
    const [from, to] = bandRange(band, bands)
    const label = v.short ? bandName(band) : t(m.bandShort, { band: bandName(band), from, to })
    const size = 11.5
    const w = 8 + (v.short ? 4 : 5) + textWidth(label, size, 600)
    const center = (xOfScore(from) + xOfScore(Math.min(100, to + 1))) / 2
    return { band, label, x: Math.max(v.left, Math.min(v.right - w, center - w / 2)), gap: v.short ? 12 : 13 }
  })
  const tipFor = (i: number) => {
    const b = shown[i]
    if (!b) return null
    const text = t(m.bucketTip, { bucket: `${b.from}–${b.to}`, count: b.contacts, band: bandName(bandFor(b)) })
    const w = textWidth(text, 12, 600) + 24
    const cx = v.left + slot * i + slot / 2
    const yb = y(b.contacts)
    const x = Math.max(4, Math.min(v.w - w - 4, cx - w / 2))
    const ty = Math.max(4, yb - 46)
    return { text, w, x, cx, ty }
  }
  const tip = hover === null ? null : tipFor(hover)
  return (
    <svg width="100%" viewBox={`0 0 ${v.w} ${v.h}`} role="img" aria-label={desc}>
      {ticks.map((n) => (
        <g key={n}>
          <line x1={v.left} x2={v.right} y1={y(n)} y2={y(n)} stroke={n === 0 ? '#C9CED6' : '#EEF0F3'} strokeWidth={1} />
          <text x={v.left - 8} y={y(n) + 4} textAnchor="end" fontSize={11} fill="#5B6472" fontFamily={FONT}>
            {v.short ? shortCount(n) : count(n)}
          </text>
        </g>
      ))}
      {shown.map((b, i) => {
        const x0 = v.left + slot * i
        const now = buckets[i]
        const band = bandFor(b)
        const label = t(m.bucketTip, { bucket: `${b.from}–${b.to}`, count: b.contacts, band: bandName(band) })
        return (
          <g
            key={b.from}
            className="bar"
            tabIndex={0}
            role="img"
            aria-label={label}
            onMouseEnter={() => setHover(i)}
            onMouseLeave={() => setHover(null)}
            onFocus={() => setHover(i)}
            onBlur={() => setHover(null)}
          >
            {/* The whole slot takes the pointer. */}
            <rect x={x0} y={v.top} width={slot} height={v.base - v.top} fill="transparent" />
            {after && now ? <path d={barPath(x0 + (slot - v.bar) / 2, y(now.contacts), v.bar, v.base, v.r)} fill="#D4D8DF" /> : null}
            {after ? (
              <path d={barPath(x0 + (slot - v.bar) / 2 + 5, y(b.contacts), v.bar - 10, v.base, v.r)} fill={bandColor[band]} />
            ) : (
              <path d={barPath(x0 + (slot - v.bar) / 2, y(b.contacts), v.bar, v.base, v.r)} fill={bandColor[band]} />
            )}
            {v.short ? null : (
              <text x={x0 + slot / 2} y={v.base + 18} textAnchor="middle" fontSize={11} fill="#5B6472" fontFamily={FONT}>
                {`${b.from}–${b.to}`}
              </text>
            )}
          </g>
        )
      })}
      {legend.map((l) => (
        <g key={l.band} aria-hidden="true">
          <rect x={l.x} y={9} width={8} height={8} rx={2} fill={bandColor[l.band]} />
          <text x={l.x + l.gap} y={17} fontSize={11.5} fontWeight={600} fill="#374151" fontFamily={FONT}>
            {l.label}
          </text>
        </g>
      ))}
      {[bands.warmFrom, bands.hotFrom].map((s) => (
        <g key={s} aria-hidden="true">
          <line x1={xOfScore(s)} x2={xOfScore(s)} y1={v.short ? 24 : 28} y2={v.base} stroke="#374151" strokeWidth={1} />
          <circle cx={xOfScore(s)} cy={v.short ? 24 : 28} r={2.5} fill="#374151" />
        </g>
      ))}
      {v.short ? (
        <g aria-hidden="true">
          {[0, bands.warmFrom, bands.hotFrom, 100].map((s, i, all) => (
            <text key={s} x={xOfScore(s)} y={v.base + 16} textAnchor={i === 0 ? 'start' : i === all.length - 1 ? 'end' : 'middle'} fontSize={11} fill="#5B6472" fontFamily={FONT}>
              {s}
            </text>
          ))}
        </g>
      ) : null}
      {tip ? (
        <g aria-hidden="true" pointerEvents="none">
          <rect x={tip.x} y={tip.ty} width={tip.w} height={26} rx={6} fill="#0B1220" />
          <path d={`M${tip.cx},${tip.ty + 26} l6,0 l-6,6 l-6,-6 Z`} fill="#0B1220" />
          <text x={tip.x + tip.w / 2} y={tip.ty + 17} textAnchor="middle" fontSize={12} fontWeight={600} fill="#fff" fontFamily={FONT}>
            {tip.text}
          </text>
        </g>
      ) : null}
    </svg>
  )
}

function SpreadTable({ buckets, after, bands }: { buckets: Bucket[]; after?: Bucket[]; bands: Bands }) {
  const t = useT()
  const total = (after ?? buckets).reduce((s, b) => s + b.contacts, 0) || 1
  return (
    <TableScroll label={t(m.spreadTitle, { contacts: total })}>
      <table className="tbl sc-t11">
        <thead>
          <tr>
            <th scope="col">{t(m.colScore)}</th>
            <th scope="col">{t(m.colBand)}</th>
            <th scope="col" className="r">
              {after ? t(m.colNow) : t(m.colContacts)}
            </th>
            {after ? (
              <th scope="col" className="r">
                {t(m.colAfter)}
              </th>
            ) : (
              <th scope="col" className="r">
                %
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {(after ?? buckets).map((b, i) => {
            const band = bandOf(b.from, bands)
            return (
              <tr key={b.from}>
                <td className="num">{`${b.from}–${b.to}`}</td>
                <td>
                  <span className="sc-band">
                    <span className="sc-sw" style={{ background: bandColor[band] }} aria-hidden="true" />
                    {bandName(band)}
                  </span>
                </td>
                <td className="r num">{count(buckets[i]?.contacts ?? 0)}</td>
                <td className="r num">{after ? count(b.contacts) : pct((100 * b.contacts) / total)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </TableScroll>
  )
}
