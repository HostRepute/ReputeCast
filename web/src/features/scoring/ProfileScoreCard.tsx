import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { tierLabel, type Tier } from '@/features/audience/labels'
import { T, useT } from '@/i18n'
import { dayMonthDate, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { bandColor, bandName, type Band } from './bands'
import { m } from './profile.messages'
import { RuleWhen } from './RulesCard'
import { textWidth } from './SpreadChart'
import { capText, signed, type Options } from './ruleText'
import './scoring.css'

type Score = Schemas['ContactScore']
type Breakdown = Schemas['ContactScoreRule']
type Field = Schemas['ContactProfileField']
type TFn = ReturnType<typeof useT>

const FONT = 'IBM Plex Sans,system-ui,sans-serif'
const day = (d: string) => new Date(`${d}T12:00:00Z`)

/**
 * The contact profile's score card (ScoringProfile): the score and band, a
 * 0–100 meter, the last change, where the score counts, the last 90 days
 * and the breakdown by rule, with the note on tiers.
 */
export function ProfileScoreCard({ contactId, firstName, tier, fields }: { contactId: string; firstName: string; tier: Tier; fields: Field[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contact}/score', { params: { path: { workspace: slug, contact: contactId } } })
  const s = q.data
  const header = (
    <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
      <div className="flex min-w-0 flex-col gap-0.5">
        <h2 className="h2" id="sc-profile-h">
          {t(m.title)}
        </h2>
        <span className="k">{t(m.lead)}</span>
      </div>
      <ButtonLink size="sm" to="/w/$workspace/audience/scoring" params={{ workspace: slug }}>
        {t(m.rules)}
      </ButtonLink>
    </div>
  )
  if (!s) {
    return (
      <section className="card sc-pscore" aria-labelledby="sc-profile-h" aria-busy={!q.isError}>
        {header}
        {q.isError ? (
          <div className="flex flex-wrap items-center gap-3">
            <span className="sc-err">
              {t(m.loadError)} · {errorMessage(q.error)}
            </span>
            <Button size="sm" onClick={() => void q.refetch()}>
              {t(m.retry)}
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-2.5">
            <Skeleton w={120} h={30} r={6} />
            <Skeleton w="100%" h={8} r={4} />
            <Skeleton w="70%" h={12} r={6} />
          </div>
        )}
      </section>
    )
  }
  if (!s.configured) {
    return (
      <section className="card sc-pscore" aria-labelledby="sc-profile-h">
        {header}
        <div className="flex flex-wrap items-center gap-3">
          <span className="k">{t(m.noScoring)}</span>
          <Link to="/w/$workspace/audience/scoring" params={{ workspace: slug }}>
            {t(m.setUp)}
          </Link>
        </div>
      </section>
    )
  }
  const band = s.band as Band
  return (
    <section className="card sc-pscore" aria-labelledby="sc-profile-h">
      {header}
      {s.recalculating ? (
        <div className="note i" role="status">
          <div>{t(m.recalculating)}</div>
        </div>
      ) : null}
      <div className="sc-scg">
        <div className="sc-lc sc-lc-a">
          <div className="flex flex-wrap items-end gap-x-4 gap-y-1.5">
            <div className="sc-sbig">
              <span className="n">{s.score}</span>
              <span className="b">
                ·
                <span className="sc-sw" style={{ background: bandColor[band], width: 12, height: 12 }} aria-hidden="true" />
                {bandName(band)}
              </span>
            </div>
            {s.bandSince ? (
              <span className="k" style={{ paddingBottom: 4 }}>
                {t(m.bandSince, { band: bandName(band), day: shortDate(s.bandSince) })}
              </span>
            ) : null}
          </div>
          <Meter score={s.score} band={band} warm={s.bands.warmFrom} hot={s.bands.hotFrom} />
          {s.lastChange ? <LastChange c={s.lastChange} /> : null}
          <WhereItCounts s={s} firstName={firstName} />
        </div>
        <div className="sc-lc sc-lc-b">
          <span className="sc-lbl">{t(m.last90)}</span>
          <ScoreLine s={s} />
        </div>
        <div className="sc-lc sc-lc-c">
          <span className="sc-lbl">{t(m.breakdown)}</span>
          <div className="sc-brk">
            {s.breakdown.map((b) => (
              <div key={b.rule.id} className="r">
                <div className="l">
                  <span>
                    <RuleWhen r={b.rule} o={optionsOf(b)} />
                  </span>
                  <span className="k">{detailOf(b, s, fields, t)}</span>
                </div>
                <span className="sc-pts">{b.points ? signed(b.points) : '0'}</span>
              </div>
            ))}
            {s.decay.enabled ? (
              <div className="r">
                <div className="l">
                  <span>{t(m.decay)}</span>
                  <span className="k">{decayLine(s, t)}</span>
                </div>
                <span className="sc-pts">{s.decay.lost ? signed(-s.decay.lost) : '0'}</span>
              </div>
            ) : null}
            <div className="r tot">
              <span>{t(m.total)}</span>
              <span className="inline-flex items-center gap-2">
                <span className="sc-band">
                  <span className="sc-sw" style={{ background: bandColor[band] }} aria-hidden="true" />
                  {bandName(band)}
                </span>
                <span className="sc-pts" style={{ background: 'var(--surface)' }}>
                  {s.score}
                </span>
              </span>
            </div>
          </div>
        </div>
      </div>
      <div className="note sc-tiernote" role="note">
        <span className="sc-note-ic" style={{ color: 'var(--ink-2)' }}>
          <Icon name="info" size={16} strokeWidth={1.8} />
        </span>
        <div>
          <T
            msg={m.tierNote}
            values={{
              tier: tierLabel[tier],
              band: bandName(band),
              b: (c) => <b>{c}</b>,
              link: (c) => (
                <Link to="/w/$workspace/audience/health" params={{ workspace: slug }}>
                  {c}
                </Link>
              ),
            }}
          />
        </div>
      </div>
    </section>
  )
}

/** The options a breakdown row needs to name its form or list. */
function optionsOf(b: Breakdown): Options {
  return {
    lists: b.rule.listId && b.listName ? [{ id: b.rule.listId, name: b.listName }] : [],
    forms: b.rule.formId && b.formName ? [{ id: b.rule.formId, name: b.formName, kind: 'embedded' }] : [],
    apiEvents: [],
    fields: [],
  }
}

function detailOf(b: Breakdown, s: Score, fields: Field[], t: TFn): string {
  const r = b.rule
  if (r.kind === 'field') {
    const f = fields.find((x) => x.key === r.field || x.key === `custom:${r.field}`)
    return f && f.value !== '' ? t(m.fieldNow, { field: r.field ?? '', value: f.value }) : t(m.fieldEmpty, { field: r.field ?? '' })
  }
  if (r.kind === 'soft_bounce' && b.zeroEmail && b.zeroAt) return t(m.bounceRetried, { email: b.zeroEmail, day: dayMonthDate(b.zeroAt) })
  if (b.atMax) return t(m.atMax, { count: b.count, cap: capText(r).toLowerCase() })
  switch (r.kind) {
    case 'open':
      return s.joinedAt ? t(m.opens, { count: b.count, day: dayMonthDate(s.joinedAt) }) : t(m.opensPlain, { count: b.count })
    case 'click':
      return t(m.clicks, { count: b.count })
  }
  return t(m.events, { count: b.count })
}

function decayLine(s: Score, t: TFn): string {
  const d = s.decay
  const last = d.lastOpen ?? d.lastActivity
  if (d.lost > 0 && last) return t(m.decayLost, { weeks: d.quietWeeks, day: shortDate(last) })
  return last ? t(m.decayNone, { day: shortDate(last) }) : t(m.decayNoneNever)
}

/** A 0–100 track in the band colours, with the score marked. */
function Meter({ score, band, warm, hot }: { score: number; band: Band; warm: number; hot: number }) {
  const t = useT()
  return (
    <div className="sc-meter" role="img" aria-label={t(m.meter, { score, band: bandName(band), warm, hot })}>
      <div className="trk">
        <i style={{ flex: warm, background: bandColor.cold }} />
        <i style={{ flex: hot - warm, background: bandColor.warm }} />
        <i style={{ flex: 100 - hot, background: bandColor.hot }} />
        <span className="mk" style={{ left: `${score}%` }} />
      </div>
      <div className="tks" aria-hidden="true">
        <span className="s0" style={{ left: 0 }}>
          0
        </span>
        <span style={{ left: `${warm}%` }}>{warm}</span>
        <span style={{ left: `${hot}%` }}>{hot}</span>
        <span className="s100" style={{ left: '100%' }}>
          100
        </span>
      </div>
    </div>
  )
}

function causeText(c: Schemas['ContactScoreCause'], t: TFn): string {
  switch (c.kind) {
    case 'open':
      return t(m.causeOpen, { email: c.emailName ?? '' })
    case 'click':
      return c.linkLabel ? t(m.causeClick, { link: c.linkLabel, email: c.emailName ?? '' }) : t(m.causeClickNoLink, { email: c.emailName ?? '' })
    case 'form':
      return t(m.causeForm, { form: c.formName ?? '' })
    case 'api_event':
      return t(m.causeApiEvent, { event: c.eventName ?? '' })
    case 'soft_bounce':
      return t(m.causeSoftBounce, { email: c.emailName ?? '' })
    case 'unsubscribe':
      return c.listName ? t(m.causeUnsubscribe, { list: c.listName }) : t(m.causeUnsubscribeAny)
    case 'decay':
      return t(m.causeDecay)
  }
  return t(m.causeField)
}

function LastChange({ c }: { c: Schemas['ContactScoreChange'] }) {
  const t = useT()
  const cause = causeText(c.cause, t)
  const text =
    c.beforeBand !== c.afterBand
      ? t(m.changeBand, { cause, before: c.before, after: c.after, from: bandName(c.beforeBand), to: bandName(c.afterBand) })
      : t(m.change, { cause, before: c.before, after: c.after })
  return (
    <div className="sc-change">
      <span className="sc-pts">{signed(c.delta)}</span>
      <div className="flex min-w-0 flex-col gap-0.5">
        <span>
          <T msg={m.lastChange} values={{ day: shortDate(c.at), b: (x) => <b>{x}</b> }} />
        </span>
        <span className="k">{text}</span>
      </div>
    </div>
  )
}

function WhereItCounts({ s, firstName }: { s: Score; firstName: string }) {
  const t = useT()
  const name = firstName || '—'
  const line = (u: Schemas['ContactScoreUse']): string => {
    switch (u.kind) {
      case 'segment':
        return t(u.in ? m.segmentIn : m.segmentOut, { name })
      case 'automation':
        if (u.in && u.crossedAt) return t(m.automationIn, { threshold: u.threshold ?? 0, day: shortDate(u.crossedAt) })
        if (u.crossedAt && u.liveSince && u.crossedAt < u.liveSince) {
          return t(m.automationBefore, { threshold: u.threshold ?? 0, day: shortDate(u.crossedAt), live: shortDate(u.liveSince) })
        }
        return t(m.automationWaits, { direction: u.direction === 'falls' ? 'falls' : 'rises', threshold: u.threshold ?? 0 })
    }
    return t(u.in ? m.blockIn : m.blockOut, { name })
  }
  return (
    <div className="flex flex-col gap-1.5 text-[13px]">
      <span className="sc-lbl">{t(m.whereItCounts)}</span>
      {s.uses.length === 0 ? <span className="k">{t(m.nothingReads)}</span> : null}
      {s.uses.map((u) => (
        <div key={u.kind + u.id} className="flex flex-col gap-px">
          {u.href ? (
            <a href={u.href} className="self-start">
              {u.name}
            </a>
          ) : (
            <span className="self-start">{u.name}</span>
          )}
          <span className="k">{line(u)}</span>
        </div>
      ))}
    </div>
  )
}

interface LineVariant {
  key: 'desk' | 'tab' | 'phone'
  w: number
  h: number
  x0: number
  x1: number
  top: number
  base: number
  labelY: number
  legendX: number
}

const lineVariants: LineVariant[] = [
  { key: 'desk', w: 360, h: 280, x0: 34, x1: 306, top: 10, base: 258, labelY: 275, legendX: 312 },
  { key: 'tab', w: 700, h: 190, x0: 34, x1: 646, top: 10, base: 168, labelY: 185, legendX: 652 },
  { key: 'phone', w: 330, h: 160, x0: 34, x1: 276, top: 10, base: 138, labelY: 155, legendX: 282 },
]

/** The score over the last 90 days, as steps, with the band thresholds. */
function ScoreLine({ s }: { s: Score }) {
  const t = useT()
  const pts = s.line
  const firstScored = pts.findIndex((p) => p.score !== undefined && p.score !== null)
  const joinedInside = firstScored > 0
  // The description: none before joining, the first score, each band reached, now.
  const parts: string[] = []
  if (joinedInside) parts.push(t(m.lineNone, { day: dayMonthDate(day(pts[firstScored]!.day)) }))
  if (firstScored >= 0) parts.push(t(m.lineFirst, { score: pts[firstScored]!.score ?? 0, day: dayMonthDate(day(pts[firstScored]!.day)) }))
  let prev: Band | undefined
  for (const p of pts) {
    if (p.score === undefined || p.score === null) continue
    const b: Band = p.score >= s.bands.hotFrom ? 'hot' : p.score >= s.bands.warmFrom ? 'warm' : 'cold'
    if (prev && b !== prev) parts.push(t(m.lineBand, { band: bandName(b), day: dayMonthDate(day(p.day)) }))
    prev = b
  }
  parts.push(t(m.lineNow, { score: s.score }))
  const desc = t(m.lineDesc, { parts: parts.join(', ') })
  return (
    <>
      {lineVariants.map((v) => (
        <div key={v.key} className={`sc-chart sc-v-${v.key}`}>
          <LineSvg v={v} s={s} firstScored={firstScored} desc={desc} />
        </div>
      ))}
    </>
  )
}

function LineSvg({ v, s, firstScored, desc }: { v: LineVariant; s: Score; firstScored: number; desc: string }) {
  const t = useT()
  const pts = s.line
  const n = Math.max(1, pts.length - 1)
  const x = (i: number) => v.x0 + ((v.x1 - v.x0) * i) / n
  const y = (score: number) => v.base - ((v.base - v.top) * score) / 100
  const ticks = [0, s.bands.warmFrom, s.bands.hotFrom, 100]
  let d = ''
  if (firstScored >= 0) {
    const xs = x(firstScored)
    d = `M${(xs + 3).toFixed(1)},${y(0).toFixed(1)}`
    pts.forEach((p, i) => {
      if (i < firstScored || p.score === undefined || p.score === null) return
      d += ` H${Math.max(xs + 3, x(i)).toFixed(1)} V${y(p.score).toFixed(1)}`
    })
  }
  const last = pts.length - 1
  const joinX = firstScored > 0 ? x(firstScored) : undefined
  const labels: ReactNode[] = [
    <text key="a" x={v.x0} y={v.labelY} textAnchor="start" fontSize={11} fill="#5B6472" fontFamily={FONT}>
      {pts.length ? dayMonthDate(day(pts[0]!.day)) : ''}
    </text>,
    <text key="c" x={v.x1} y={v.labelY} textAnchor="end" fontSize={11} fill="#5B6472" fontFamily={FONT}>
      {pts.length ? dayMonthDate(day(pts[last]!.day)) : ''}
    </text>,
  ]
  // The join day's label shows when it clears the first and last days' labels.
  const joinLabel = joinX !== undefined ? t(m.joinedTick, { day: dayMonthDate(day(pts[firstScored]!.day)) }) : ''
  const half = textWidth(joinLabel, 11) / 2
  const startW = pts.length ? textWidth(dayMonthDate(day(pts[0]!.day)), 11) : 0
  const endW = pts.length ? textWidth(dayMonthDate(day(pts[last]!.day)), 11) : 0
  if (joinX !== undefined && joinX - half - 6 > v.x0 + startW && joinX + half + 6 < v.x1 - endW) {
    labels.push(
      <text key="b" x={joinX} y={v.labelY} textAnchor="middle" fontSize={11} fill="#5B6472" fontFamily={FONT}>
        {joinLabel}
      </text>,
    )
  }
  return (
    <svg width="100%" viewBox={`0 0 ${v.w} ${v.h}`} role="img" aria-label={desc} style={{ display: 'block', maxWidth: '100%' }}>
      {ticks.map((k, i) => (
        <g key={k}>
          <line x1={v.x0} x2={v.x1} y1={y(k)} y2={y(k)} stroke={i === 0 ? '#C9CED6' : '#EEF0F3'} strokeWidth={1} />
          <text x={v.x0 - 6} y={y(k) + 4} textAnchor="end" fontSize={11} fill="#5B6472" fontFamily={FONT}>
            {k}
          </text>
        </g>
      ))}
      {(['warm', 'hot'] as Band[]).map((b) => {
        const yy = y(b === 'warm' ? s.bands.warmFrom : s.bands.hotFrom)
        return (
          <g key={b} aria-hidden="true">
            <rect x={v.legendX} y={yy - 9} width={7} height={7} rx={2} fill={bandColor[b]} />
            <text x={v.legendX + 10} y={yy - 3} fontSize={11} fill="#374151" fontWeight={600} fontFamily={FONT}>
              {bandName(b)}
            </text>
          </g>
        )
      })}
      {joinX !== undefined ? (
        <>
          {joinX - v.x0 > 110 ? (
            <text x={(v.x0 + joinX) / 2} y={y(30)} textAnchor="middle" fontSize={11} fill="#5B6472" fontFamily={FONT}>
              {t(m.notYet)}
            </text>
          ) : null}
          <line x1={joinX} x2={joinX} y1={v.top} y2={v.base} stroke="#C9CED6" strokeWidth={1} />
        </>
      ) : null}
      {d ? <path d={d} fill="none" stroke="var(--accent)" strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" /> : null}
      {firstScored >= 0 ? (
        <>
          <circle cx={x(last)} cy={y(s.score)} r={5} fill="var(--accent)" stroke="#fff" strokeWidth={2} />
          <text x={x(last) - 8} y={y(s.score) - 8} textAnchor="end" fontSize={12} fontWeight={700} fill="#0B1220" fontFamily={FONT}>
            {s.score}
          </text>
        </>
      ) : null}
      {labels}
    </svg>
  )
}
