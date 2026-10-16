import type { Component } from 'grapesjs'
import { useEffect, useId, useRef, useState } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Icon } from '@/components/icons/Icon'
import { formatList, T, useT } from '@/i18n'
import { useComposer, type ComposerState } from '../context'
import { tagOf } from '../editor/labels'
import { APPLICABLE, useApplyFix } from './checkFixes'
import { m } from './messages'
import { useTx, type TxState } from './state'
import { VAR_TYPES, type Check, type Variable, type VarType } from './types'
import { NAME_RE, newVariable, RESERVED, usesVariable } from './variables'

export function typeLabel(t: ReturnType<typeof useT>, type: VarType): string {
  switch (type) {
    case 'url':
      return t(m.typeUrl)
    case 'number':
      return t(m.typeNumber)
    case 'boolean':
      return t(m.typeBoolean)
    case 'date':
      return t(m.typeDate)
    case 'list':
      return t(m.typeList)
    case 'object':
      return t(m.typeObject)
  }
  return t(m.typeText)
}

function clip(s: string, n = 28): string {
  const x = s.replace(/\s+/g, ' ').trim()
  return x.length > n ? `${x.slice(0, n - 1)}…` : x
}

function walk(c: Component, fn: (x: Component) => void) {
  fn(c)
  c.components().forEach((x: Component) => walk(x, fn))
}

/** Where the template uses a variable: "Subject", "Button “Reset password” · link"… */
function usedIn(t: ReturnType<typeof useT>, c: ComposerState, tx: TxState, name: string): string[] {
  const out: string[] = []
  if (usesVariable(c.subject, name)) out.push(t(m.usedSubject))
  if (usesVariable(c.preheader, name)) out.push(t(m.usedPreheader))
  if (tx.htmlMode && usesVariable(tx.html, name)) out.push(t(m.usedHtml))
  const body = c.editor?.getWrapper()?.findType('mj-body')[0]
  if (body)
    walk(body, (x) => {
      const tag = tagOf(x)
      if (!tag || tag === 'mj-body' || tag === 'mj-section' || tag === 'mj-column' || tag === 'mj-wrapper' || tag === 'mj-group') return
      const attrs = x.getAttributes() as Record<string, unknown>
      const inLink = ['href', 'src', 'background-url'].some((a) => usesVariable(String(attrs[a] ?? ''), name))
      const inner = (x as unknown as { getInnerHTML?: () => string }).getInnerHTML?.() ?? ''
      const raw = tag === 'mj-table' ? String(x.get('content') ?? '') : inner
      const inText = usesVariable(raw, name)
      if (!inLink && !inText) return
      const text = clip(raw.replace(/<[^>]*>/g, ' '))
      if (tag === 'mj-button') out.push(inLink ? t(m.usedButtonLink, { label: text }) : t(m.usedButton, { label: text }))
      else if (tag === 'mj-image') out.push(inLink ? t(m.usedImageLink) : t(m.usedImage))
      else if (tag === 'mj-text') out.push(/href=["'][^"']*\{\{/.test(raw) && !text.includes('{{') ? t(m.usedTextLink, { label: text }) : t(m.usedText, { label: text }))
      else if (tag === 'mj-table') out.push(t(m.usedTable))
      else out.push(tag)
    })
  if (usesVariable(tx.plainText, name)) out.push(t(m.usedPlainText))
  return out
}

/** One declared variable: a summary line, or its form when open. */
function VarRow({ v, open, onToggle, onRenamed }: { v: Variable; open: boolean; onToggle: () => void; onRenamed: (name: string) => void }) {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const ids = useId()
  const [name, setName] = useState(v.name)
  // A rename from elsewhere (a fix, a teammate's draft) replaces what the field shows.
  const [savedName, setSavedName] = useState(v.name)
  if (savedName !== v.name) {
    setSavedName(v.name)
    setName(v.name)
  }
  const read = !tx.canEdit
  const others = tx.variables.filter((x) => x !== v).map((x) => x.name)
  const nameError = !NAME_RE.test(name)
    ? t(m.nameInvalid)
    : RESERVED.has(name)
      ? t(m.nameReserved, { name })
      : others.includes(name)
        ? t(m.nameTaken, { name })
        : undefined
  const update = (patch: Partial<Variable>) => tx.setVariables(tx.variables.map((x) => (x === v ? { ...x, ...patch } : x)))
  const sample = v.sample ?? ''
  const summary =
    sample && v.default
      ? t(m.sampleDefault, { sample, fallback: v.default })
      : sample
        ? t(m.sampleOnly, { sample })
        : v.default
          ? t(m.defaultOnly, { fallback: v.default })
          : t(m.noSample)
  const where = open ? usedIn(t, c, tx, v.name) : []
  const head = (
    <span className="flex flex-wrap items-center gap-2">
      <span className="txe-vname">{v.name}</span>
      <span className="k">{v.required ? t(m.rowRequired, { type: typeLabel(t, v.type) }) : t(m.rowOptional, { type: typeLabel(t, v.type) })}</span>
      {v.sensitive ? (
        <Badge tone="warn" size="sm">
          {t(m.sensitive)}
        </Badge>
      ) : null}
    </span>
  )
  return (
    <div className={`txe-vrow${open ? ' on' : ''}`}>
      <button type="button" className="txe-vhead" aria-expanded={open} aria-controls={`${ids}-form`} onClick={onToggle}>
        {head}
        {open ? null : <span className="k txe-vsum">{summary}</span>}
      </button>
      {open ? (
        <fieldset id={`${ids}-form`} className="txe-vform" disabled={read}>
          <div className="grid grid-cols-2 gap-2">
            <label className="lbl">
              {t(m.varName)}
              <input
                className="in mono"
                style={{ fontSize: 12 }}
                value={name}
                maxLength={64}
                aria-invalid={nameError ? true : undefined}
                aria-describedby={nameError ? `${ids}-nerr` : undefined}
                onChange={(e) => setName(e.target.value.trim())}
                onBlur={() => {
                  if (nameError) setName(v.name)
                  else if (name !== v.name) {
                    tx.renameVariable(v.name, name)
                    onRenamed(name)
                  }
                }}
              />
            </label>
            <label className="lbl">
              {t(m.varType)}
              <select className="in" value={v.type} onChange={(e) => update({ type: e.target.value as VarType })}>
                {VAR_TYPES.map((x) => (
                  <option key={x} value={x}>
                    {typeLabel(t, x)}
                  </option>
                ))}
              </select>
            </label>
          </div>
          {nameError ? (
            <span id={`${ids}-nerr`} className="k txe-bad">
              {nameError}
            </span>
          ) : null}
          <label className="lbl">
            {t(m.sampleValue)}
            {v.type === 'list' || v.type === 'object' ? (
              <textarea className="in mono" style={{ fontSize: 12 }} rows={3} value={sample} maxLength={4000} onChange={(e) => update({ sample: e.target.value })} />
            ) : (
              <input className="in mono" style={{ fontSize: 12 }} value={sample} maxLength={4000} onChange={(e) => update({ sample: e.target.value })} />
            )}
          </label>
          {v.required ? null : (
            <label className="lbl">
              {t(m.defaultValue)}
              <input className="in mono" style={{ fontSize: 12 }} value={v.default ?? ''} maxLength={4000} placeholder={t(m.defaultPlaceholder)} onChange={(e) => update({ default: e.target.value })} />
            </label>
          )}
          <div className="txe-sw">
            <span className="font-semibold">{t(m.required)}</span>
            <input type="checkbox" role="switch" className="tg" aria-label={t(m.required)} checked={v.required} onChange={(e) => update({ required: e.target.checked, default: e.target.checked ? undefined : v.default })} />
          </div>
          <div className="txe-sw items-start!">
            <span className="flex flex-col gap-0.5">
              <span className="font-semibold">{t(m.sensitive)}</span>
              <span className="k">{t(m.sensitiveHelp)}</span>
            </span>
            <input type="checkbox" role="switch" className="tg" aria-label={t(m.sensitive)} checked={v.sensitive} onChange={(e) => update({ sensitive: e.target.checked })} />
          </div>
          <label className="lbl">
            {t(m.varDescription)}
            <input className="in" value={v.description ?? ''} maxLength={300} placeholder={t(m.varDescriptionPlaceholder)} onChange={(e) => update({ description: e.target.value })} />
          </label>
          <span className="k">{where.length ? t(m.usedIn, { places: formatList(where) }) : t(m.notUsed)}</span>
          {read ? null : (
            <button
              type="button"
              className="btn sm danger self-start"
              onClick={() => {
                tx.setVariables(
                  tx.variables.filter((x) => x !== v),
                  t(m.noteRemoved, { name: v.name }),
                )
              }}
            >
              {t(m.removeVariable)}
            </button>
          )}
        </fieldset>
      ) : null}
    </div>
  )
}

function CheckNote({ tone, title, checks }: { tone: 'w' | 'n'; title: string; checks: Check[] }) {
  const t = useT()
  const tx = useTx()
  const apply = useApplyFix()
  return (
    <div className={`note ${tone}`} role="note">
      <Icon name={tone === 'w' ? 'alertCircle' : 'info'} size={16} />
      <div className="flex min-w-0 flex-col gap-1.5" style={{ flex: '1 1 220px' }}>
        <span className="font-bold">{title}</span>
        {checks.map((ch) => (
          <div key={`${ch.id}-${ch.variable}-${ch.line ?? 0}`} className="flex flex-col gap-1.5">
            <span>{ch.id === 'variables.unused' && ch.variable ? t(m.unusedDetail, { name: ch.variable }) : ch.detail || ch.title}</span>
            {tx.canEdit && ch.fixes.some((f) => APPLICABLE.has(f.kind)) ? (
              <span className="flex flex-wrap gap-1.5">
                {ch.fixes
                  .filter((f) => APPLICABLE.has(f.kind))
                  .map((f) => (
                    <button key={f.kind + f.label} type="button" className="btn sm" onClick={() => apply(ch, f)}>
                      {f.label}
                    </button>
                  ))}
              </span>
            ) : null}
          </div>
        ))}
        {checks.length === 0 ? <span>{t(m.checking)}</span> : null}
      </div>
    </div>
  )
}

/**
 * Variables (TransactionalTemplate board): name, type, required, sample,
 * default and Sensitive, with the warnings for undeclared and unused ones.
 */
export function VariablesTab() {
  const t = useT()
  const tx = useTx()
  const [open, setOpen] = useState<string | null>(null)
  const list = useRef<HTMLDivElement>(null)
  // A check (or a fix that declares a variable) opens its row.
  if (tx.focusVariable && open !== tx.focusVariable) setOpen(tx.focusVariable)
  const { focusVariable, setFocusVariable } = tx
  useEffect(() => {
    if (!focusVariable) return
    setFocusVariable(null)
    // Not cleared when focusVariable resets: the row should still scroll into view.
    window.setTimeout(() => list.current?.querySelector('.txe-vrow.on')?.scrollIntoView({ block: 'nearest', behavior: 'smooth' }), 60)
  }, [focusVariable, setFocusVariable])
  const checks = tx.checks?.checks ?? []
  const undeclared = checks.filter((c) => c.id === 'variables.undeclared')
  const unused = checks.filter((c) => c.id === 'variables.unused')
  const render = tx.checks?.testData.find((r) => r.name === tx.previewWith)
  const allGood = !!tx.checks && tx.variables.length > 0 && undeclared.length === 0 && unused.length === 0
  const declare = () => {
    const taken = tx.variables.map((v) => v.name)
    let n = 'new_variable'
    for (let i = 2; taken.includes(n); i++) n = `new_variable_${i}`
    tx.setVariables([...tx.variables, newVariable(n)], t(m.noteDeclared, { name: n }))
    setOpen(n)
  }
  return (
    <div className="pbody" ref={list}>
      <div className="flex items-center justify-between gap-2">
        <span className="font-semibold">{t(m.variablesCount, { count: tx.variables.length })}</span>
        {tx.canEdit ? (
          <button type="button" className="btn sm" onClick={declare}>
            <Icon name="plus" size={13} />
            {t(m.declare)}
          </button>
        ) : null}
      </div>
      {tx.variables.length === 0 ? <p className="k m-0">{t(m.noVariables)}</p> : null}
      {tx.variables.map((v, i) => (
        <VarRow key={i} v={v} open={open === v.name} onToggle={() => setOpen((o) => (o === v.name ? null : v.name))} onRenamed={setOpen} />
      ))}
      {undeclared.length ? <CheckNote tone="w" title={t(m.undeclaredTitle, { count: undeclared.length })} checks={undeclared} /> : null}
      {unused.length ? <CheckNote tone="n" title={t(m.unusedTitle, { count: unused.length })} checks={unused} /> : null}
      {allGood ? (
        <div className="note g" role="status">
          <Icon name="check" size={14} />
          <div>
            {render?.ok && tx.previewWith
              ? t(m.allGoodWith, { count: tx.variables.length, name: tx.previewWith })
              : t(m.allGood, { count: tx.variables.length })}
          </div>
        </div>
      ) : null}
      <span className="k">
        <T msg={m.apiNote} values={{ mono: (chunks) => <span className="mono">{chunks}</span> }} />
      </span>
    </div>
  )
}
