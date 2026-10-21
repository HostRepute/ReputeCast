import type { Component } from 'grapesjs'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { useBuilder } from './context'
import { attrOf } from './editor/nodes'

/**
 * The Block tab's fields, in the inspector's vocabulary (.prop, .lbl,
 * .trow): each one reads a block's attribute and writes it back through
 * the editor, so every change is one step of undo. Text fields keep what's
 * typed while they have focus and write it after a pause or on leaving.
 */

/** Writes attributes; an empty value for an attribute in `drop` removes it. */
export function setAttrs(c: Component, patch: Record<string, string>, drop: readonly string[] = []) {
  const keep: Record<string, string> = {}
  const remove: string[] = []
  for (const [k, v] of Object.entries(patch)) {
    if (v === '' && drop.includes(k)) remove.push(k)
    else keep[k] = v
  }
  if (Object.keys(keep).length) c.addAttributes(keep)
  if (remove.length) c.removeAttributes(remove)
}

export function Prop({ children, row, className }: { children: ReactNode; row?: boolean; className?: string }) {
  return <div className={`prop${row ? ' pb-proprow' : ''}${className ? ` ${className}` : ''}`}>{children}</div>
}

/** A toggle row: the setting and what it means, the switch on the right. */
export function ToggleRow({ label, hint, checked, onChange, disabled }: { label: string; hint?: ReactNode; checked: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  const id = useId()
  return (
    <div className="trow">
      <span className="tt">
        <b id={`${id}-l`}>{label}</b>
        {hint ? <span className="k">{hint}</span> : null}
      </span>
      <input type="checkbox" role="switch" className="tg" aria-labelledby={`${id}-l`} checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
    </div>
  )
}

/**
 * A text field bound to an attribute. `show` and `store` map between what
 * the field shows and what's stored ("Buy for $29" ↔ "Buy for {price}").
 */
export function AttrText({
  c,
  name,
  label,
  hint,
  mono,
  multiline,
  placeholder,
  show = (v) => v,
  store = (v) => v,
  invalid,
  id,
  type = 'text',
}: {
  c: Component
  name: string
  label: string
  hint?: ReactNode
  mono?: boolean
  multiline?: boolean
  placeholder?: string
  show?: (stored: string) => string
  store?: (typed: string) => string
  invalid?: (typed: string) => string | undefined
  id?: string
  type?: 'text' | 'url' | 'date' | 'time'
}) {
  const b = useBuilder()
  const [editing, setEditing] = useState<string | null>(null)
  const timer = useRef<number | undefined>(undefined)
  const auto = useId()
  const fid = id ?? auto
  const stored = attrOf(c, name)
  const value = editing ?? show(stored)
  const write = (v: string) => {
    window.clearTimeout(timer.current)
    const next = store(v)
    if (next !== attrOf(c, name)) setAttrs(c, { [name]: next })
  }
  useEffect(() => () => window.clearTimeout(timer.current), [])
  const problem = editing !== null && invalid ? invalid(editing) : invalid?.(value)
  const common = {
    id: fid,
    className: `in${mono ? ' mono' : ''}`,
    value,
    placeholder,
    disabled: !b.canEdit,
    'aria-invalid': problem ? true : undefined,
    onFocus: () => setEditing(show(attrOf(c, name))),
    onChange: (e: { target: { value: string } }) => {
      const v = e.target.value
      setEditing(v)
      window.clearTimeout(timer.current)
      // Typing pauses: the canvas draws the change (live blocks are drawn by the server).
      timer.current = window.setTimeout(() => write(v), 700)
    },
    onBlur: () => {
      if (editing !== null) write(editing)
      setEditing(null)
    },
  }
  return (
    <>
      <label className="lbl" htmlFor={fid}>
        {label}
        {multiline ? <textarea {...common} rows={3} style={{ resize: 'vertical' }} /> : <input {...common} type={type} />}
      </label>
      {problem ? <span className="err">{problem}</span> : hint ? <span className="k">{hint}</span> : null}
    </>
  )
}

/** A select bound to an attribute. */
export function AttrSelect({
  c,
  name,
  label,
  options,
  id,
  onPick,
}: {
  c: Component
  name: string
  label: string
  options: readonly { value: string; label: string; disabled?: boolean }[]
  id?: string
  /** Writes other attributes along with the pick. */
  onPick?: (value: string) => Record<string, string>
}) {
  const b = useBuilder()
  const auto = useId()
  const fid = id ?? auto
  const value = attrOf(c, name)
  return (
    <label className="lbl" htmlFor={fid}>
      {label}
      <select
        id={fid}
        className="in"
        value={value}
        disabled={!b.canEdit}
        onChange={(e) => {
          const v = e.target.value
          setAttrs(c, { [name]: v, ...(onPick?.(v) ?? {}) })
        }}
      >
        {options.some((o) => o.value === value) ? null : <option value={value}>{value}</option>}
        {options.map((o) => (
          <option key={o.value} value={o.value} disabled={o.disabled}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  )
}

/** A segmented control bound to an attribute. */
export function AttrSeg({ c, name, label, items, fallback }: { c: Component; name: string; label: string; items: readonly { value: string; label: string }[]; fallback: string }) {
  const b = useBuilder()
  const value = attrOf(c, name) || fallback
  return (
    <div className="seg" role="group" aria-label={label}>
      {items.map((it) => (
        <button key={it.value} type="button" aria-pressed={value === it.value} disabled={!b.canEdit} onClick={() => setAttrs(c, { [name]: it.value })}>
          {it.label}
        </button>
      ))}
    </div>
  )
}

/** A "1"/"0" flag bound to a toggle row. */
export function AttrFlag({ c, name, label, hint, fallback = true }: { c: Component; name: string; label: string; hint?: ReactNode; fallback?: boolean }) {
  const b = useBuilder()
  const raw = attrOf(c, name)
  const on = raw === '' ? fallback : raw !== '0'
  return <ToggleRow label={label} hint={hint} checked={on} disabled={!b.canEdit} onChange={(v) => setAttrs(c, { [name]: v ? '1' : '0' })} />
}
