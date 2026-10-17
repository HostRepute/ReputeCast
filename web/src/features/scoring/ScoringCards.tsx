import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { CardSection } from '@/components/ui/Card'
import { Input } from '@/components/ui/Field'
import { TableScroll } from '@/components/ui/Table'
import { Segmented } from '@/components/ui/Tabs'
import { tierClass, tierLabel } from '@/features/audience/labels'
import { formatList, useT } from '@/i18n'
import { count, pct, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { bandColor, bandName, bandOrder, bandRange, type Band, type Bands } from './bands'
import { m } from './messages'
import { SpreadChart } from './SpreadChart'
import { configProblems, type Config, type Impact, type ScoringPageData } from './useScoringDraft'

type Use = Schemas['ScoringUse']
type Counts = Schemas['ScoringBandCounts']

export function Swatch({ band, size = 10 }: { band: Band; size?: number }) {
  return <span className="sc-sw" style={{ background: bandColor[band], width: size, height: size }} aria-hidden="true" />
}

/** "Warm → Hot 1,108". */
export function MoveChip({ from, to, n }: { from: Band; to: Band; n: number }) {
  const t = useT()
  return (
    <span className="sc-move">
      <Swatch band={from} />
      {bandName(from)}
      <span className="sc-arrow" aria-hidden="true">
        →
      </span>
      <span className="vh">{t(m.moveTo)}</span>
      <Swatch band={to} />
      {bandName(to)}
      <span className="num" style={{ marginLeft: 2 }}>
        {count(n)}
      </span>
    </span>
  )
}

const signedCount = (n: number) => (n > 0 ? `+${count(n)}` : n < 0 ? `−${count(-n)}` : '0')

/** Cold · Warm · Hot with their counts, and with an edit, before → after. */
export function BandTiles({ bands, counts, after, afterBands, total }: { bands: Bands; counts: Counts; after?: Counts; afterBands?: Bands; total: number }) {
  const t = useT()
  const ab = afterBands ?? bands
  return (
    <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(min(220px,100%),1fr))]">
      {bandOrder.map((b) => {
        const [from, to] = bandRange(b, ab)
        const n = counts[b]
        const a = after?.[b]
        return (
          <div key={b} className="card sc-tile" style={{ borderLeft: `4px solid ${bandColor[b]}` }}>
            <span className="k">{t(m.bandRange, { band: bandName(b), from, to })}</span>
            <span className="v">{a === undefined ? count(n) : `${count(n)} → ${count(a)}`}</span>
            <span className="k">
              {a === undefined ? t(m.bandShare, { pct: pct(total ? (100 * n) / total : 0) }) : t(m.bandAfterShare, { delta: signedCount(a - n), pct: pct(total ? (100 * a) / total : 0) })}
            </span>
          </div>
        )
      })}
    </div>
  )
}

/** The spread card with its Chart/Table switch and the thresholds. */
export function SpreadCard({
  page,
  config,
  impact,
  canEdit,
  onBands,
}: {
  page: ScoringPageData
  config: Config
  impact?: Impact
  canEdit: boolean
  onBands: (b: Bands) => void
}) {
  const t = useT()
  const [view, setView] = useState<'chart' | 'table'>('chart')
  const after = impact && impact.changed + impact.moves.length > 0 ? impact : undefined
  const editing = Boolean(impact)
  return (
    <section className="card min-w-0" aria-labelledby="sc-spread">
      <CardSection
        id="sc-spread"
        title={t(editing ? m.spreadAfterTitle : m.spreadTitle, { contacts: page.contacts })}
        caption={t(editing ? m.spreadAfterLead : m.spreadLead)}
        actions={
          <Segmented
            label={t(m.chartOrTable)}
            value={view}
            onChange={setView}
            items={[
              { value: 'chart', label: t(m.chart) },
              { value: 'table', label: t(m.table) },
            ]}
          />
        }
      />
      <SpreadChart view={view} buckets={page.spread} after={editing ? impact?.spread : undefined} bands={config.bands} />
      {after ? (
        <div className="sc-mrow" style={{ padding: '4px 18px 12px' }}>
          <span style={{ fontWeight: 700 }}>{t(m.moves, { count: after.changed })}</span>
          {after.moves.map((mv) => (
            <MoveChip key={mv.from + mv.to} from={mv.from} to={mv.to} n={mv.count} />
          ))}
        </div>
      ) : null}
      {editing ? (
        <div className="sc-legend" aria-hidden="true">
          <span>
            <span className="sc-sw" style={{ background: bandColor.hot }} />
            {t(m.legendAfter)}
          </span>
          <span>
            <span className="sc-sw" style={{ background: '#D4D8DF' }} />
            {t(m.legendNow)}
          </span>
        </div>
      ) : null}
      <div className="sc-cfoot">
        <div className="flex flex-wrap items-end gap-3">
          <Threshold label={t(m.warmFrom)} aria={t(m.warmFromLabel)} value={config.bands.warmFrom} min={1} max={99} disabled={!canEdit} onChange={(v) => onBands({ ...config.bands, warmFrom: v })} />
          <Threshold label={t(m.hotFrom)} aria={t(m.hotFromLabel)} value={config.bands.hotFrom} min={2} max={100} disabled={!canEdit} onChange={(v) => onBands({ ...config.bands, hotFrom: v })} />
        </div>
        <span className="k" style={{ flex: '1 1 260px', minWidth: 0, paddingBottom: 9 }}>
          {t(m.thresholdsNote)}
        </span>
        {configProblems(config).bands ? (
          <span className="sc-err" role="alert" style={{ flexBasis: '100%' }}>
            {configProblems(config).bands}
          </span>
        ) : null}
      </div>
    </section>
  )
}

function Threshold({ label, aria, value, min, max, disabled, onChange }: { label: string; aria: string; value: number; min: number; max: number; disabled: boolean; onChange: (v: number) => void }) {
  const [text, setText] = useState(String(value))
  const [last, setLast] = useState(value)
  if (value !== last) {
    setLast(value)
    setText(String(value))
  }
  return (
    <label className="sc-thr">
      {label}
      <Input
        className="num"
        type="number"
        inputMode="numeric"
        min={min}
        max={max}
        value={text}
        disabled={disabled}
        aria-label={aria}
        onChange={(e) => {
          setText(e.target.value)
          const n = Number(e.target.value)
          if (e.target.value !== '' && Number.isInteger(n)) onChange(n)
        }}
      />
    </label>
  )
}


function Mix({ hot, warm, cold, wide }: { hot: number; warm: number; cold: number; wide?: boolean }) {
  const t = useT()
  return (
    <div className={wide ? 'sc-mix wide' : 'sc-mix'} role="img" aria-label={t(m.mixLabel, { hot, warm, cold })}>
      {hot ? <span style={{ flex: hot, background: bandColor.hot }} /> : null}
      {warm ? <span style={{ flex: warm, background: bandColor.warm }} /> : null}
      {cold ? <span style={{ flex: cold, background: bandColor.cold }} /> : null}
    </div>
  )
}

const zeroK = (n: number) => (n === 0 ? <span className="k">0</span> : count(n))

/** Bands by tier: tiers and bands are separate. */
export function TiersCard({ page }: { page: ScoringPageData }) {
  const t = useT()
  const active = page.byTier.find((r) => r.tier === 'active')
  const lateHot = page.byTier.filter((r) => r.tier === 'at_risk' || r.tier === 'sunset').reduce((s, r) => s + r.hot, 0)
  const ex = page.tierExample
  return (
    <section className="card" aria-labelledby="sc-tiers">
      <CardSection id="sc-tiers" title={t(m.tiersCardTitle)} caption={t(m.tiersCardLead)} />
      <div className="sc-no-phone">
        <TableScroll label={t(m.tiersCardTitle)}>
          <table className="tbl">
            <thead>
              <tr>
                <th scope="col">{t(m.colTier)}</th>
                <th scope="col" className="r">
                  {t(m.colContacts)}
                </th>
                <th scope="col" className="r">
                  {t(m.hot)}
                </th>
                <th scope="col" className="r">
                  {t(m.warm)}
                </th>
                <th scope="col" className="r">
                  {t(m.cold)}
                </th>
                <th scope="col">{t(m.colMix)}</th>
              </tr>
            </thead>
            <tbody>
              {page.byTier.map((r) => (
                <tr key={r.tier}>
                  <td>
                    <span className={`badge ${tierClass[r.tier]}`}>{tierLabel[r.tier]}</span>
                  </td>
                  <td className="r num">{count(r.contacts)}</td>
                  <td className="r num">{zeroK(r.hot)}</td>
                  <td className="r num">{zeroK(r.warm)}</td>
                  <td className="r num">{zeroK(r.cold)}</td>
                  <td>
                    <Mix hot={r.hot} warm={r.warm} cold={r.cold} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      </div>
      <div className="sc-only-phone">
        {page.byTier.map((r) => (
          <div key={r.tier} className="flex flex-col gap-1.5 border-t border-line-2 px-4 py-3">
            <div className="flex items-center justify-between gap-3">
              <span className={`badge ${tierClass[r.tier]}`}>{tierLabel[r.tier]}</span>
              <span className="num font-semibold">{count(r.contacts)}</span>
            </div>
            <Mix hot={r.hot} warm={r.warm} cold={r.cold} wide />
            <span className="k">{t(m.mixLine, { hot: r.hot, warm: r.warm, cold: r.cold })}</span>
          </div>
        ))}
      </div>
      {active ? (
        <div className="sc-foot">
          <span className="k" style={{ flex: '1 1 300px', minWidth: 0, color: 'var(--ink-2)' }}>
            {t(m.tierActiveCold, { count: active.cold })}
            {ex ? ` ${t(m.tierExample, { name: ex.name, score: ex.score })}` : ''}
            {lateHot === 0 ? ` ${t(m.noLateHot)}` : ''}
          </span>
        </div>
      ) : null}
    </section>
  )
}

/** Decay and limits. */
export function DecayCard({ config, canEdit, onDecay }: { config: Config; canEdit: boolean; onDecay: (d: Config['decay']) => void }) {
  const t = useT()
  const d = config.decay
  const num = (v: string, fallback: number) => {
    const n = Number(v)
    return v !== '' && Number.isInteger(n) ? n : fallback
  }
  return (
    <section className="card" aria-labelledby="sc-decay">
      <CardSection id="sc-decay" title={t(m.decayTitle)} />
      <div className="flex flex-wrap items-start justify-between gap-3.5 px-4.5 py-3.5">
        <div className="flex min-w-0 flex-[1_1_300px] flex-col gap-[3px]">
          <span className="font-semibold">{t(m.decay)}</span>
          <span className="k" style={{ color: 'var(--ink-2)' }}>
            {t(m.decayLead)}
          </span>
        </div>
        <input type="checkbox" className="tg" role="switch" checked={d.enabled} disabled={!canEdit} aria-label={t(m.decay)} onChange={(e) => onDecay({ ...d, enabled: e.target.checked })} />
      </div>
      <div className="px-4.5 pb-3.5">
        <div className="sc-dk">
          <span className="k" style={{ color: 'var(--ink-nav)', fontSize: 13 }}>
            {t(m.startAfter)}
          </span>
          <Input
            className="num"
            type="number"
            min={1}
            max={52}
            value={d.afterWeeks}
            disabled={!canEdit || !d.enabled}
            aria-label={t(m.weeksLabel)}
            onChange={(e) => onDecay({ ...d, afterWeeks: num(e.target.value, d.afterWeeks) })}
          />
          <span>{t(m.weeksWords)}</span>
          <span className="k" style={{ color: 'var(--ink-nav)', fontSize: 13 }}>
            {t(m.takeOff)}
          </span>
          <Input
            className="num"
            type="number"
            min={1}
            max={100}
            value={d.percent}
            disabled={!canEdit || !d.enabled}
            aria-label={t(m.percentLabel)}
            onChange={(e) => onDecay({ ...d, percent: num(e.target.value, d.percent) })}
          />
          <span>{t(m.percentWords)}</span>
        </div>
        {configProblems(config).decay ? (
          <span className="sc-err mt-2 block" role="alert">
            {configProblems(config).decay}
          </span>
        ) : null}
      </div>
      <div className="border-t border-line-2 px-4.5 pt-3 pb-4">
        <dl className="kv">
          <dt>{t(m.maximum)}</dt>
          <dd>100</dd>
          <dt>{t(m.lookback)}</dt>
          <dd>{t(m.lookbackText)}</dd>
          <dt>{t(m.updates)}</dt>
          <dd>{t(m.updatesText)}</dd>
        </dl>
      </div>
    </section>
  )
}

/** A segment's conditions in one line: "Score band is Hot, plan is team or business". */
export function conditionsLine(raw: unknown, t: TFn): string {
  const conds = Array.isArray(raw) ? (raw as { field?: string; op?: string; value?: string; to?: string }[]) : []
  const parts = conds.map((c) => {
    const field = c.field ?? ''
    const value = c.value ?? ''
    if (field === 'score_band') {
      const band = (['cold', 'warm', 'hot'] as Band[]).includes(value as Band) ? bandName(value as Band) : value
      return t(c.op === 'is_not' ? m.condBandNot : m.condBand, { band })
    }
    if (field === 'score') {
      if (c.op === 'between') return t(m.condBetween, { a: value, b: c.to ?? '' })
      return t(c.op === 'at_most' ? m.condAtMost : m.condAtLeast, { n: value })
    }
    const name = field.replace(/^custom:/, '')
    switch (c.op) {
      case 'is_one_of': {
        const vals = value
          .split(',')
          .map((v) => v.trim())
          .filter(Boolean)
        return t(m.condOneOf, { field: name, values: formatList(vals, 'disjunction') })
      }
      case 'not_equals':
      case 'is_not':
        return t(m.condNot, { field: name, value })
      case 'contains':
        return t(m.condContains, { field: name, value })
      case 'is_empty':
        return t(m.condEmpty, { field: name })
      case 'is_not_empty':
        return t(m.condSet, { field: name })
    }
    return t(m.condIs, { field: name, value })
  })
  // Conditions read as a list, as the segment builder lists them.
  return parts.join(', ')
}

/** Used by: the automations, segments and conditional blocks that read the score. */
export function UsedByCard({ page }: { page: ScoringPageData }) {
  const t = useT()
  const { slug } = useWorkspace()
  const uses = page.usedBy
  return (
    <section className="card" aria-labelledby="sc-uses">
      <CardSection id="sc-uses" title={t(m.usedByTitle, { count: uses.length })} caption={t(m.usedByLead)} />
      <div style={{ padding: '2px 0' }}>
        {uses.length === 0 ? (
          <p className="k px-4.5 py-3">{t(m.usedByNone)}</p>
        ) : (
          uses.map((u) => (
            <div key={u.kind + u.id} className="sc-use">
              <div className="flex min-w-0 flex-1 flex-col gap-[3px]">
                <span className="flex flex-wrap items-center gap-2">
                  <UseLink u={u} slug={slug} />
                  <UseBadge u={u} />
                </span>
                <span className="k">{lineOf(u, t)}</span>
                <span className="k" style={{ color: 'var(--ink-nav)' }}>
                  {detailOf(u, t)}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
      <div className="sc-foot">
        <span className="k" style={{ flex: '1 1 300px', minWidth: 0 }}>
          {t(m.usedByFoot)}
        </span>
      </div>
    </section>
  )
}

function UseLink({ u, slug }: { u: Use; slug: string }): ReactNode {
  switch (u.kind) {
    case 'automation':
      return (
        <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: u.id }} className="font-semibold">
          {u.name}
        </Link>
      )
    case 'segment':
      return (
        <Link to="/w/$workspace/audience/segments/$segmentId" params={{ workspace: slug, segmentId: u.id }} className="font-semibold">
          {u.name}
        </Link>
      )
  }
  return u.href ? (
    <a href={u.href} className="font-semibold">
      {u.name}
    </a>
  ) : (
    <span className="font-semibold">{u.name}</span>
  )
}

function UseBadge({ u }: { u: Use }) {
  const t = useT()
  switch (u.kind) {
    case 'automation':
      return u.status === 'live' ? (
        <Badge tone="ok" dot>
          {t(m.live)}
        </Badge>
      ) : u.status === 'paused' ? (
        <Badge tone="warn">{t(m.paused)}</Badge>
      ) : (
        <Badge tone="neutral">{t(m.draft)}</Badge>
      )
    case 'segment':
      return <Badge tone="neutral">{t(m.segment)}</Badge>
  }
  return <Badge tone="info">{t(m.block)}</Badge>
}

type TFn = ReturnType<typeof useT>

function lineOf(u: Use, t: TFn): string {
  switch (u.kind) {
    case 'automation':
      return u.since && u.status === 'live' ? t(m.automationLine, { day: shortDate(u.since), by: u.by ?? '' }) : t(m.automationLineNotLive, { by: u.by ?? '' })
    case 'segment':
      return t(m.segmentLine, { by: u.by ?? '' })
  }
  return t(m.blockLine, { where: u.where ?? '' })
}

function detailOf(u: Use, t: TFn): string {
  switch (u.kind) {
    case 'automation':
      return t(m.automationTrigger, {
        direction: u.direction === 'falls' ? 'falls' : 'rises',
        threshold: u.threshold ?? 0,
        once: u.onceInDays ? String(u.onceInDays) : 'none',
        entered: u.entered,
      })
    case 'segment': {
      const line = conditionsLine(u.conditions, t)
      const n = t(m.segmentCount, { count: u.contacts ?? 0 })
      return line ? `${line} · ${n}` : n
    }
  }
  return u.of ? t(m.blockCount, { count: u.contacts ?? 0, of: u.of }) : ''
}
