import type { ReactNode } from 'react'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { mr } from './referrals.messages'

/** A milestone on the track: its count, what it gives and the line under it. */
export interface TrackPoint {
  count: number
  title: ReactNode
  /** A plain line between the title and the grey one (the reward). */
  mid?: ReactNode
  sub: ReactNode
}

/** Where a count sits on a track whose milestones are evenly spaced. */
export function trackPos(counts: readonly number[], c: number): number {
  const n = counts.length
  if (!n || c <= 0) return 0
  let prev = 0
  for (let i = 0; i < n; i++) {
    const at = counts[i]!
    if (c <= at) {
      const span = at - prev || 1
      return ((i + (c - prev) / span) / n) * 100
    }
    prev = at
  }
  return 100
}

/**
 * The milestone track (GrowOverview's program card, the referrer panel):
 * a rail with a dot per milestone and their labels under it. With
 * `counted`, the rail fills to the referrer's progress, "Counting soon"
 * shows what's in hold, and `here` labels where they are.
 */
export function MilestoneTrack({
  points,
  label,
  counted,
  soon = 0,
  here,
  legend,
}: {
  points: TrackPoint[]
  label: string
  /** The referrer's counted referrals; without it every dot is filled (an aggregate). */
  counted?: number
  soon?: number
  here?: string
  legend?: boolean
}) {
  const tr = useT()
  const counts = points.map((p) => p.count)
  const n = points.length
  const personal = counted !== undefined
  const fill = personal ? trackPos(counts, counted) : 0
  const soonTo = personal ? trackPos(counts, counted + soon) : 0
  const next = personal ? counts.findIndex((c) => c > counted) : -1
  const herePos = fill
  const hereSide = herePos > 85 ? 'r' : herePos < 12 ? 'l' : ''
  return (
    <div className={personal && here ? 'gw-mt here' : 'gw-mt'} role="img" aria-label={label}>
      <div className="gw-mt-rail">
        {personal ? <i className="f" style={{ width: `${fill}%` }} /> : null}
        {personal && soonTo > fill ? <i className="s" style={{ left: `${fill}%`, width: `${soonTo - fill}%` }} /> : null}
        {points.map((p, i) => {
          const at = ((i + 1) / n) * 100
          const done = personal ? counted >= p.count : true
          const cls = personal ? (done ? 'd done' : i === next ? 'd next' : 'd') : 'd agg'
          return (
            <i key={p.count} className={cls} style={{ left: `${at}%` }}>
              {done ? <Icon name="check" size={9} strokeWidth={3} /> : null}
            </i>
          )
        })}
        {personal && here ? (
          <span className={`gw-mt-here ${hereSide}`} style={{ left: `${herePos}%` }}>
            {here}
          </span>
        ) : null}
      </div>
      <div className="gw-mt-lab" style={{ gridTemplateColumns: `repeat(${Math.max(1, n)}, minmax(0, 1fr))` }}>
        {points.map((p) => (
          <div key={p.count}>
            <b>{p.title}</b>
            {p.mid ? <span>{p.mid}</span> : null}
            <span className="k">{p.sub}</span>
          </div>
        ))}
      </div>
      {legend ? (
        <div className="gw-mt-key">
          <span>
            <i style={{ background: 'var(--accent)' }} />
            {tr(mr.keyCounted)}
          </span>
          <span>
            <i style={{ background: 'var(--accent-line)' }} />
            {tr(mr.keySoon)}
          </span>
          <span>
            <i style={{ background: 'var(--line-2)' }} />
            {tr(mr.keyToGo)}
          </span>
        </div>
      ) : null}
    </div>
  )
}
