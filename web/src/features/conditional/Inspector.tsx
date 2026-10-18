import type { Component } from 'grapesjs'
import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent, type ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Popover } from '@/components/ui/Menu'
import { formatList, T as Rich, useT } from '@/i18n'
import { clock, count, shortDate } from '@/lib/format'
import { useComposer } from '@/features/composer/context'
import { kb } from '@/features/composer/format'
import { tagOf } from '@/features/composer/editor/labels'
import { EyeOff } from './Canvas'
import { m } from './messages'
import { conditional, isFooter, nameOf, newRuleId, offered, ruleOf, setName, setRule, type VisibilityCondition, type VisibilityRule } from './rules'
import { useConditional, type VisibilityCatalog } from './state'
import './conditional.css'

type T = ReturnType<typeof useT>

const OP_LABEL: Record<string, keyof typeof m> = {
  includes: 'opIncludes',
  excludes: 'opExcludes',
  in: 'opIn',
  not_in: 'opNotIn',
  has: 'opHas',
  has_not: 'opHasNot',
  is: 'opIs',
  is_not: 'opIsNot',
  equals: 'opEquals',
  not_equals: 'opNotEquals',
  contains: 'opContains',
  gt: 'opGt',
  lt: 'opLt',
  before: 'opBefore',
  after: 'opAfter',
  is_empty: 'opIsEmpty',
  is_not_empty: 'opIsNotEmpty',
  is_true: 'opIsTrue',
  is_false: 'opIsFalse',
  at_least: 'opAtLeast',
  at_most: 'opAtMost',
  between: 'opBetween',
}

const KIND_LABEL: Record<string, keyof typeof m> = {
  segment: 'kindSegment',
  list: 'kindList',
  tag: 'kindTag',
  field: 'kindField',
  language: 'kindLanguage',
  tier: 'kindTier',
  score: 'kindScore',
  score_band: 'kindScoreBand',
}

const GROUP_LABEL: Record<string, keyof typeof m> = { lists: 'groupLists', contact: 'groupContact', engagement: 'groupEngagement' }

export function opLabel(t: T, op: string): string {
  const k = OP_LABEL[op]
  return k ? t(m[k]) : op
}

/** The operators of a condition: its kind's, or its field's. */
function opsOf(cat: VisibilityCatalog | undefined, cond: VisibilityCondition): { key: string; noValue?: boolean }[] {
  if (cond.kind === 'field') return cat?.fields.find((f) => f.key === cond.field)?.ops ?? [{ key: 'equals' }, { key: 'is_empty', noValue: true }]
  return cat?.kinds.find((k) => k.key === cond.kind)?.ops ?? []
}

/** The picker's note under a kind: how many there are, or their names. */
function kindNote(t: T, cat: VisibilityCatalog | undefined, kind: string): string {
  if (!cat) return ''
  const names = (xs: { label: string }[]) => xs.map((x) => x.label).join(', ')
  switch (kind) {
    case 'segment':
      return t(m.noteSegments, { count: cat.segments.length })
    case 'list':
      return names(cat.lists)
    case 'tag':
      return t(m.noteTags, { count: cat.tags.length })
    case 'field':
      return t(m.noteField)
    case 'language':
      return names(cat.languages)
    case 'tier':
      return names(cat.tiers)
    case 'score':
      return cat.score.configured ? t(m.noteScore, { hot: cat.score.hotFrom, warm: cat.score.warmFrom }) : t(m.noteNoScoring)
    case 'score_band':
      return cat.score.configured ? names(cat.bands) : t(m.noteNoScoring)
  }
  return ''
}

function rowLabel(t: T, cat: VisibilityCatalog | undefined, cond: VisibilityCondition): string {
  switch (cond.kind) {
    case 'segment':
      return t(m.rowSegment)
    case 'list':
      return t(m.rowList)
    case 'tag':
      return t(m.rowTag)
    case 'field': {
      const f = cat?.fields.find((x) => x.key === cond.field)
      return cond.field ? t(m.rowField, { field: f?.label ?? cond.field }) : t(m.kindField)
    }
    case 'language':
      return t(m.rowLanguage)
    case 'tier':
      return t(m.rowTier)
    case 'score':
      return t(m.rowScore)
    case 'score_band':
      return t(m.rowScoreBand)
  }
  return cond.kind
}

function defaultCondition(cat: VisibilityCatalog | undefined, kind: string): VisibilityCondition {
  const ops = cat?.kinds.find((k) => k.key === kind)?.ops ?? []
  const op = ops[0]?.key ?? 'is'
  switch (kind) {
    case 'segment':
      return { kind, op, value: cat?.segments[0]?.value ?? '', label: cat?.segments[0]?.label }
    case 'list':
      return { kind, op, value: cat?.lists[0]?.value ?? '', label: cat?.lists[0]?.label }
    case 'tier':
      return { kind, op, value: 'at_risk' }
    case 'language':
      return { kind, op, value: cat?.languages[0]?.value ?? '' }
    case 'score':
      return { kind, op, value: String(cat?.score.hotFrom ?? 50) }
    case 'score_band':
      return { kind, op, value: 'hot' }
    case 'field': {
      const f = cat?.fields[0]
      return { kind, op: f?.ops[0]?.key ?? 'equals', field: f?.key ?? '', value: '' }
    }
  }
  return { kind, op, value: '' }
}

/** A text value committed on Enter or blur (one undo step per edit). */
function ValueInput({
  value,
  onCommit,
  label,
  disabled,
  type = 'text',
  list,
  min,
  max,
}: {
  value: string
  onCommit: (v: string) => void
  label: string
  disabled?: boolean
  type?: string
  list?: string
  min?: number
  max?: number
}) {
  const [draft, setDraft] = useState(value)
  const focused = useRef(false)
  useEffect(() => {
    if (!focused.current) setDraft(value)
  }, [value])
  const commit = () => draft !== value && onCommit(draft.trim())
  return (
    <input
      className="in"
      aria-label={label}
      type={type}
      list={list}
      min={min}
      max={max}
      value={draft}
      disabled={disabled}
      onFocus={() => (focused.current = true)}
      onBlur={() => {
        focused.current = false
        commit()
      }}
      onChange={(e) => setDraft(e.target.value)}
      onKeyDown={(e: KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
          commit()
          ;(e.target as HTMLInputElement).blur()
        }
      }}
    />
  )
}

/** The value control of a condition, by its kind. */
function ValueField({ cond, onChange, disabled }: { cond: VisibilityCondition; onChange: (c: VisibilityCondition) => void; disabled?: boolean }) {
  const t = useT()
  const cat = useConditional()?.catalog
  const listId = useId()
  const label = t(m.value)
  const op = opsOf(cat, cond).find((o) => o.key === cond.op)
  if (op?.noValue) return <span className="k self-center">{t(m.noValue)}</span>
  const pick = (opts: { value: string; label: string; contacts?: number }[], missing: string) => {
    const known = opts.some((o) => o.value === cond.value)
    return (
      <select className="in" aria-label={label} value={cond.value ?? ''} disabled={disabled} onChange={(e) => onChange({ ...cond, value: e.target.value, label: opts.find((o) => o.value === e.target.value)?.label })}>
        {!known ? <option value={cond.value ?? ''}>{cond.value ? missing : t(m.pickValue)}</option> : null}
        {opts.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    )
  }
  switch (cond.kind) {
    case 'segment':
      return pick(cat?.segments ?? [], t(m.deleted, { name: cond.label || cond.value || '' }))
    case 'list':
      return pick(cat?.lists ?? [], t(m.deleted, { name: cond.label || cond.value || '' }))
    case 'language':
      return pick(cat?.languages ?? [], cond.value ?? '')
    case 'tier':
      return pick(cat?.tiers ?? [], cond.value ?? '')
    case 'score_band':
      return pick(cat?.bands ?? [], cond.value ?? '')
    case 'score':
      // A whole score from 0 to 100; "is between" takes both ends.
      return cond.op === 'between' ? (
        <span className="cv-range">
          <ValueInput label={t(m.scoreFrom)} type="number" min={0} max={100} value={cond.value ?? ''} disabled={disabled} onCommit={(v) => onChange({ ...cond, value: v })} />
          <span className="k">{t(m.rangeAnd)}</span>
          <ValueInput label={t(m.scoreTo)} type="number" min={0} max={100} value={cond.to ?? ''} disabled={disabled} onCommit={(v) => onChange({ ...cond, to: v })} />
        </span>
      ) : (
        <ValueInput label={label} type="number" min={0} max={100} value={cond.value ?? ''} disabled={disabled} onCommit={(v) => onChange({ ...cond, value: v })} />
      )
    case 'tag':
      return (
        <>
          <ValueInput label={label} value={cond.value ?? ''} disabled={disabled} list={listId} onCommit={(v) => onChange({ ...cond, value: v })} />
          <datalist id={listId}>
            {(cat?.tags ?? []).map((x) => (
              <option key={x.value} value={x.value} />
            ))}
          </datalist>
        </>
      )
    case 'field': {
      const f = cat?.fields.find((x) => x.key === cond.field)
      if (f?.options.length) return pick(f.options.map((o) => ({ value: o, label: o })), cond.value ?? '')
      return <ValueInput label={label} type={f?.type === 'number' ? 'number' : f?.type === 'date' ? 'date' : 'text'} value={cond.value ?? ''} disabled={disabled} onCommit={(v) => onChange({ ...cond, value: v })} />
    }
  }
  return <ValueInput label={label} value={cond.value ?? ''} disabled={disabled} onCommit={(v) => onChange({ ...cond, value: v })} />
}

/** "Add condition": the picker, grouped and searchable, with room for later kinds. */
function ConditionPicker({ onPick, disabled, trigger, className = 'btn sm self-start' }: { onPick: (kind: string) => void; disabled?: boolean; trigger: ReactNode; className?: string }) {
  const t = useT()
  const cat = useConditional()?.catalog
  const [open, setOpen] = useState(false)
  const [q, setQ] = useState('')
  const needle = q.trim().toLowerCase()
  const groups = (cat?.groups ?? ['lists', 'contact', 'engagement']).map((g) => ({
    key: g,
    label: GROUP_LABEL[g] ? t(m[GROUP_LABEL[g]!]) : g,
    kinds: (cat?.kinds ?? []).filter((k) => k.group === g && (!needle || (KIND_LABEL[k.key] ? t(m[KIND_LABEL[k.key]!]) : k.key).toLowerCase().includes(needle))),
  }))
  const empty = groups.every((g) => !g.kinds.length)
  return (
    <Popover
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) setQ('')
      }}
      align="start"
      className="cv-pick"
      label={t(m.addConditionLabel)}
      trigger={
        <button type="button" className={className} disabled={disabled}>
          {trigger}
        </button>
      }
    >
      <div className="cv-pick-body">
        <input className="in" type="search" aria-label={t(m.searchConditions)} placeholder={t(m.searchConditions)} value={q} onChange={(e) => setQ(e.target.value)} autoFocus />
        {empty ? <p className="k">{t(m.noConditions, { query: q.trim() })}</p> : null}
        {groups.map((g) =>
          g.kinds.length ? (
            <div key={g.key} className="cv-pick-group" role="group" aria-label={g.label}>
              <span className="cv-pick-h">{g.label}</span>
              {g.kinds.map((k) => (
                <button
                  key={k.key}
                  type="button"
                  className="cv-pick-item"
                  onClick={() => {
                    onPick(k.key)
                    setOpen(false)
                    setQ('')
                  }}
                >
                  <span className="font-semibold">{KIND_LABEL[k.key] ? t(m[KIND_LABEL[k.key]!]) : k.key}</span>
                  <span className="k">{kindNote(t, cat, k.key)}</span>
                </button>
              ))}
            </div>
          ) : null,
        )}
      </div>
    </Popover>
  )
}

function ConditionRow({
  cond,
  index,
  onChange,
  onRemove,
  disabled,
  broken,
}: {
  cond: VisibilityCondition
  index: number
  onChange: (c: VisibilityCondition) => void
  onRemove: () => void
  disabled?: boolean
  broken?: boolean
}) {
  const t = useT()
  const cat = useConditional()?.catalog
  const ops = opsOf(cat, cond)
  return (
    <div className={`cv-cond${broken ? ' broken' : ''}`} role="group" aria-label={`${t(m.condition)} ${index + 1}`}>
      <div className="cv-cond-h">
        {cond.kind === 'field' ? (
          <select
            className="in"
            aria-label={t(m.condition)}
            value={cond.field ?? ''}
            disabled={disabled}
            onChange={(e) => {
              const f = cat?.fields.find((x) => x.key === e.target.value)
              onChange({ ...cond, field: e.target.value, op: f?.ops[0]?.key ?? 'equals', value: '' })
            }}
          >
            {!cat?.fields.some((f) => f.key === cond.field) ? <option value={cond.field ?? ''}>{cond.field ? t(m.deleted, { name: cond.field }) : t(m.pickField)}</option> : null}
            {(cat?.fields ?? []).map((f) => (
              <option key={f.key} value={f.key}>
                {t(m.rowField, { field: f.label })}
              </option>
            ))}
          </select>
        ) : (
          <span className="cv-cond-kind" aria-label={t(m.condition)}>
            {rowLabel(t, cat, cond)}
          </span>
        )}
        {disabled ? null : (
          <button type="button" className="btn sm ic" aria-label={t(m.removeCondition)} title={t(m.removeCondition)} onClick={onRemove}>
            <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
              <path d="M4 4l8 8M12 4l-8 8" />
            </svg>
          </button>
        )}
      </div>
      <div className="row2">
        <select
          className="in"
          aria-label={t(m.operator)}
          value={cond.op}
          disabled={disabled}
          onChange={(e) => {
            const op = e.target.value
            // A score range starts as "… and 100"; other operators take one value.
            onChange(cond.kind === 'score' ? { ...cond, op, to: op === 'between' ? cond.to || '100' : undefined } : { ...cond, op })
          }}
        >
          {ops.map((o) => (
            <option key={o.key} value={o.key}>
              {opLabel(t, o.key)}
            </option>
          ))}
        </select>
        <ValueField cond={cond} onChange={onChange} disabled={disabled} />
      </div>
    </div>
  )
}

function InfoIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true" focusable="false">
      <circle cx="8" cy="8" r="6.5" />
      <path d="M8 7.2v3.6M8 5.2v.1" />
    </svg>
  )
}

/** "Versions 2, 4 and 6". */
function versionsList(t: T, numbers: number[]): string {
  if (numbers.length === 1) return t(m.versionsWithOne, { number: numbers[0]! })
  return t(m.versionsWith, { list: formatList(numbers.map(String)) })
}

/**
 * Visibility in the Settings tab (brief §5.5): who sees the block, its
 * conditions with ALL or ANY, how many recipients see it, and the broken
 * condition's fixes.
 */
export function VisibilitySection({ c }: { c: Component }) {
  const t = useT()
  const ctx = useComposer()
  const cond = useConditional()
  const id = useId()
  const rule = ruleOf(c)
  // eslint-disable-next-line react-hooks/exhaustive-deps -- rev covers attribute changes
  const info = useMemo(() => (rule && cond ? cond.blockInfo(rule.id) : undefined), [rule?.id, cond?.report, ctx.rev])
  if (!cond?.enabled) return null
  const tag = tagOf(c)
  if (tag === 'mj-column' || tag === 'mj-group') {
    return (
      <div className="prop cv" role="group" aria-labelledby={id}>
        <span className="t" id={id}>
          <EyeOff size={14} />
          {t(m.title)} <span className="vh">· {t(m.subtitle)}</span>
        </span>
        <span className="k">{t(m.notOffered)}</span>
      </div>
    )
  }
  if (!offered(c)) return null
  const footer = isFooter(c)
  const disabled = !ctx.canEdit || footer
  const show = footer ? 'everyone' : (rule?.show ?? 'everyone')
  const update = (next: VisibilityRule | null) => setRule(c, next)
  const setShow = (s: VisibilityRule['show']) => {
    if (s === 'everyone') return update(null)
    const base = rule ?? { id: newRuleId(), show: s, match: 'all', conditions: [] }
    const conditions = base.conditions.length ? base.conditions : [defaultCondition(cond.catalog, 'list')]
    update({ ...base, show: s, conditions })
  }
  const setCondition = (i: number, next: VisibilityCondition) => rule && update({ ...rule, conditions: rule.conditions.map((x, j) => (j === i ? next : x)) })
  const removeCondition = (i: number) => {
    if (!rule) return
    const conditions = rule.conditions.filter((_, j) => j !== i)
    update(conditions.length ? { ...rule, conditions } : null)
  }
  const addCondition = (kind: string) => {
    const base = rule ?? { id: newRuleId(), show: 'only' as const, match: 'all' as const, conditions: [] }
    update({ ...base, show: base.show === 'everyone' || base.show === 'nobody' ? 'only' : base.show, conditions: [...base.conditions, defaultCondition(cond.catalog, kind)] })
  }
  const report = cond.report
  const total = report?.total ?? 0
  const brokenAt = new Map((info?.broken ?? []).map((b) => [b.condition, b]))
  const emptyVersions = (report?.versions ?? []).filter((v) => v.empty && rule && v.hidden.includes(rule.id))
  const emptyCount = emptyVersions.reduce((n, v) => n + v.recipients, 0)
  return (
    <div className={`prop cv${brokenAt.size || emptyCount ? ' bad' : ''}`} role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        <EyeOff size={14} />
        {t(m.title)} <span className="vh">· {t(m.subtitle)}</span>
      </span>
      {!footer && !ctx.canEdit ? (
        <div className="cv-note">
          <InfoIcon />
          <span>{t(m.readOnly)}</span>
        </div>
      ) : null}
      <div className="cv-show" role="radiogroup" aria-label={t(m.subtitle)}>
        {(['everyone', 'only', 'except'] as const).map((s) => (
          <label key={s} className="tg">
            <input type="radio" name={`${id}-show`} checked={show === s} disabled={disabled} onChange={() => setShow(s)} />
            {s === 'everyone' ? t(m.showEveryone) : s === 'only' ? t(m.showOnly) : t(m.showExcept)}
          </label>
        ))}
        {show === 'nobody' ? (
          <label className="tg">
            <input type="radio" name={`${id}-show`} checked disabled={disabled} readOnly />
            {t(m.showNobody)}
          </label>
        ) : null}
      </div>
      {footer ? (
        <div className="cv-note">
          <InfoIcon />
          <span>{t(m.footerAlways)}</span>
        </div>
      ) : null}
      {rule && (show === 'only' || show === 'except') ? (
        <>
          <div className="flex items-center gap-2">
            <span className="k">{t(m.match)}</span>
            <div className="seg" role="group" aria-label={t(m.match)}>
              {(['all', 'any'] as const).map((x) => (
                <button key={x} type="button" aria-pressed={(rule.match ?? 'all') === x} disabled={disabled} onClick={() => update({ ...rule, match: x })}>
                  {x === 'all' ? t(m.matchAll) : t(m.matchAny)}
                </button>
              ))}
            </div>
            <span className="k">{t(m.ofThese)}</span>
          </div>
          {rule.conditions.map((x, i) => {
            const br = brokenAt.get(i)
            return (
              <div key={i} className="flex flex-col gap-1.5">
                <ConditionRow cond={x} index={i} disabled={disabled} broken={!!br} onChange={(n) => setCondition(i, n)} onRemove={() => removeCondition(i)} />
                {br ? (
                  <div className="cv-broken" role="alert">
                    <span>
                      {br.deletedBy && br.deletedAt
                        ? t(br.kind === 'list' ? m.brokenList : br.kind === 'field' ? m.brokenField : m.brokenSegment, {
                            name: br.deletedBy,
                            date: shortDate(br.deletedAt),
                          })
                        : t(m.brokenUnknown)}
                    </span>
                    {ctx.canEdit ? (
                      <div className="flex flex-col gap-1.5">
                        <ConditionPicker
                          className="btn sm pri"
                          trigger={t(br.kind === 'list' ? m.pickAnotherList : br.kind === 'field' ? m.pickAnotherField : m.pickAnotherSegment)}
                          onPick={(kind) => setCondition(i, defaultCondition(cond.catalog, kind))}
                        />
                        <button type="button" className="btn sm" onClick={() => removeCondition(i)}>
                          {t(m.removeTheCondition)}
                        </button>
                        <button type="button" className="btn sm" onClick={() => update({ ...rule, show: 'nobody' })}>
                          {t(m.hideForEveryone)}
                        </button>
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </div>
            )
          })}
          {ctx.canEdit ? (
            <ConditionPicker
              trigger={
                <>
                  <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true" focusable="false">
                    <path d="M8 3v10M3 8h10" />
                  </svg>
                  {t(m.addCondition)}
                </>
              }
              onPick={addCondition}
            />
          ) : null}
        </>
      ) : null}
      {show === 'nobody' && ctx.canEdit ? (
        <button type="button" className="btn sm self-start" onClick={() => update(null)}>
          {t(m.showEveryoneAgain)}
        </button>
      ) : null}
      {rule && conditional(rule) && !footer ? (
        <div className="flex flex-col gap-0.5">
          {cond.counting && !info ? (
            <span className="k">{t(m.counting)}</span>
          ) : cond.error ? (
            <span className="k">{t(m.countFailed, { detail: cond.error })}</span>
          ) : report && !report.counted ? (
            <span className="k">{t(m.noAudience)}</span>
          ) : info && report ? (
            <>
              <span>
                <Rich msg={m.shown} values={{ shown: count(info.shown), total: count(total), b: (ch) => <b>{ch}</b> }} />
              </span>
              <span className="cv-bar" aria-hidden="true">
                <i style={{ width: `${total ? Math.max(1, Math.round((info.shown / total) * 100)) : 0}%` }} />
              </span>
              {report.countedAt ? <span className="k">{report.durationMs < 1000 ? t(m.countedFast, { time: clock(report.countedAt) }) : t(m.counted, { time: clock(report.countedAt) })}</span> : null}
              {info.versions.length ? <span className="k">{versionsList(t, info.versions)}</span> : null}
              {emptyCount ? <span className="cv-empty">{t(m.emptyFor, { count: emptyCount })}</span> : null}
            </>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

/** "Name in layers": the block's own name (rc-name). */
export function NameInLayers({ c }: { c: Component }) {
  const t = useT()
  const ctx = useComposer()
  const cond = useConditional()
  const id = useId()
  if (!cond?.enabled || !offered(c) || isFooter(c)) return null
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {t(m.nameInLayers)}
      </span>
      <ValueInput label={t(m.nameInLayers)} value={nameOf(c)} disabled={!ctx.canEdit} onCommit={(v) => setName(c, v)} />
      <span className="k">{t(m.nameInLayersHint)}</span>
    </div>
  )
}

/** "Versions with this block": which versions show it, and their size. */
export function VersionsWithBlock({ c }: { c: Component }) {
  const t = useT()
  const cond = useConditional()
  const id = useId()
  const rule = ruleOf(c)
  if (!cond?.enabled || !rule || !conditional(rule)) return null
  const info = cond.blockInfo(rule.id)
  if (!info || !info.versions.length) return null
  const over = (cond.report?.versions ?? []).filter((v) => info.versions.includes(v.number) && v.overClip)
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {t(m.versionsWithBlock)}
      </span>
      <span>{t(m.versionsWithBlockText, { versions: versionsList(t, info.versions), count: info.shown })}</span>
      {over.map((v) => (
        <Badge key={v.number} tone="warn" className="self-start" style={{ whiteSpace: 'normal', height: 'auto', minHeight: 22 }}>
          {t(m.versionOverClip, { number: v.number, kb: kb(v.sizeBytes).replace(/\s*KB$/, ''), count: v.recipients })}
        </Badge>
      ))}
    </div>
  )
}
