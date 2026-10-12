import type { CSSProperties, ReactNode } from 'react'
import { cn } from '@/lib/cn'

/** A thin track with one filled segment (quota, rate, threshold usage). */
export function Bar({
  value,
  color = 'var(--accent)',
  height = 6,
  label,
  className,
  animate = false,
}: {
  /** 0–100 */
  value: number
  color?: string
  height?: number
  label?: string
  className?: string
  animate?: boolean
}) {
  const v = Math.max(0, Math.min(100, value))
  return (
    <div
      className={cn('bar', className)}
      style={{ height, borderRadius: height / 2 }}
      role={label ? 'meter' : undefined}
      aria-label={label}
      aria-valuenow={label ? Math.round(v) : undefined}
      aria-valuemin={label ? 0 : undefined}
      aria-valuemax={label ? 100 : undefined}
    >
      <i className={animate ? 'grow-x' : undefined} style={{ width: `${v}%`, background: color, borderRadius: height / 2 }} />
    </div>
  )
}

export type Segment = { value: number; color: string; label?: string }

/** Stacked bar with 2px gaps between segments (tiers, routes, mailbox providers). */
export function StackBar({ segments, height = 14, label, className }: { segments: Segment[]; height?: number; label: string; className?: string }) {
  const total = segments.reduce((s, x) => s + x.value, 0) || 1
  return (
    <div className={cn('flex gap-0.5 overflow-hidden', className)} style={{ height, borderRadius: height / 2 }} role="img" aria-label={label}>
      {segments.map((s, i) => (
        <div key={i} style={{ width: `${(s.value / total) * 100}%`, background: s.color }} />
      ))}
    </div>
  )
}

/** Legend item: coloured dot, name and value. */
export function LegendItem({ color, title, detail, className }: { color: string; title: ReactNode; detail?: ReactNode; className?: string }) {
  return (
    <div className={cn('flex items-center gap-2.5', className)}>
      <span className="dot" style={{ color }} aria-hidden="true" />
      <div className="flex flex-col">
        <span className="font-semibold">{title}</span>
        {detail ? <span className="k">{detail}</span> : null}
      </div>
    </div>
  )
}

/** Small label + value pair used in card grids ("Recipients · 31,604"). */
export function Metric({ label, value, sub, size = 18, className, valueClassName, style }: { label: ReactNode; value: ReactNode; sub?: ReactNode; size?: number; className?: string; valueClassName?: string; style?: CSSProperties }) {
  return (
    <div className={className} style={style}>
      <div className="k">{label}</div>
      <div className={cn('n', valueClassName)} style={{ fontSize: size }}>
        {value}
      </div>
      {sub ? <div className="k">{sub}</div> : null}
    </div>
  )
}

/** Stat tile: caption, big tabular number, one-line comparison. */
export function StatTile({ label, value, delta, deltaTone, className }: { label: ReactNode; value: ReactNode; delta?: ReactNode; deltaTone?: 'ok' | 'bad' | 'neutral'; className?: string }) {
  return (
    <div className={cn('tile', className)}>
      <span className="k">{label}</span>
      <span className="n text-[28px] leading-[1.1] tracking-[-0.02em]">{value}</span>
      {delta ? <span className={cn('text-[12px] font-medium', deltaTone === 'ok' && 'text-ok-fg', deltaTone === 'bad' && 'text-bad-fg', (!deltaTone || deltaTone === 'neutral') && 'text-ink-3')}>{delta}</span> : null}
    </div>
  )
}

/** Key/value list (`dl.kv`). */
export function KeyValues({ items, className }: { items: [ReactNode, ReactNode][]; className?: string }) {
  return (
    <dl className={cn('kv', className)}>
      {items.map(([k, v], i) => (
        <div key={i} className="contents">
          <dt>{k}</dt>
          <dd>{v}</dd>
        </div>
      ))}
    </dl>
  )
}

/** Donut score (sender health). */
export function ScoreRing({ score, label, verdict, verdictColor = '#166534', size = 112 }: { score: number; label: string; verdict: string; verdictColor?: string; size?: number }) {
  const r = 46
  const c = 2 * Math.PI * r
  const filled = (Math.max(0, Math.min(100, score)) / 100) * c
  return (
    <svg width={size} height={size} viewBox="0 0 112 112" role="img" aria-label={label}>
      <circle cx="56" cy="56" r={r} fill="none" stroke="#EEF0F3" strokeWidth="10" />
      <circle
        cx="56"
        cy="56"
        r={r}
        fill="none"
        stroke="#0F766E"
        strokeWidth="10"
        strokeLinecap="round"
        strokeDasharray={`${filled.toFixed(1)} ${c.toFixed(0)}`}
        transform="rotate(-90 56 56)"
      />
      <text x="56" y="52" textAnchor="middle" fontFamily="IBM Plex Sans, sans-serif" fontSize="30" fontWeight="700" fill="#0B1220">
        {score}
      </text>
      <text x="56" y="72" textAnchor="middle" fontFamily="IBM Plex Sans, sans-serif" fontSize="12" fontWeight="600" fill={verdictColor}>
        {verdict}
      </text>
    </svg>
  )
}

/** A responsive line for small trends; values are drawn left to right. */
export function Sparkline({
  values,
  height = 120,
  color = '#0F766E',
  label,
  baseline = true,
  strokeWidth = 2.5,
}: {
  values: number[]
  height?: number
  color?: string
  label: string
  baseline?: boolean
  strokeWidth?: number
}) {
  const w = 600
  const pad = 2
  const min = Math.min(...values)
  const max = Math.max(...values)
  const span = max - min || 1
  const top = height * 0.18
  const bottom = height - height * 0.2
  const pts = values.map((v, i) => {
    const x = values.length === 1 ? w / 2 : (i / (values.length - 1)) * w
    const y = bottom - ((v - min) / span) * (bottom - top)
    return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)} ${Math.max(pad, y).toFixed(1)}`
  })
  return (
    <svg viewBox={`0 0 ${w} ${height}`} width="100%" height={height} preserveAspectRatio="none" role="img" aria-label={label}>
      <path d={pts.join(' ')} fill="none" stroke={color} strokeWidth={strokeWidth} strokeLinejoin="round" strokeLinecap="round" vectorEffect="non-scaling-stroke" />
      {baseline ? <line x1="0" y1={height - 2} x2={w} y2={height - 2} stroke="#E3E6EB" strokeWidth="1" vectorEffect="non-scaling-stroke" /> : null}
    </svg>
  )
}
