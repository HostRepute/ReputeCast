import { useEffect, useRef, useState, type KeyboardEvent, type PointerEvent, type ReactNode } from 'react'
import { useT } from '@/i18n'
import { formatRatio, hex, parseColor, suggestions, verdict, type ContrastVerdict, type RGBA, type Suggestion } from '../contrast'
import type { ContrastSpec, TextSample } from './contrastSpec'
import { m } from './messages'
import './accessibility.css'

interface HSV {
  h: number
  s: number
  v: number
}

function toHSV(c: RGBA): HSV {
  const r = c.r / 255
  const g = c.g / 255
  const b = c.b / 255
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const d = max - min
  let h = 0
  if (d) {
    if (max === r) h = ((g - b) / d) % 6
    else if (max === g) h = (b - r) / d + 2
    else h = (r - g) / d + 4
    h *= 60
    if (h < 0) h += 360
  }
  return { h, s: max ? d / max : 0, v: max }
}

function fromHSV({ h, s, v }: HSV): string {
  const c = v * s
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const mm = v - c
  const [r, g, b] = h < 60 ? [c, x, 0] : h < 120 ? [x, c, 0] : h < 180 ? [0, c, x] : h < 240 ? [0, x, c] : h < 300 ? [x, 0, c] : [c, 0, x]
  return hex({ r: Math.round((r + mm) * 255), g: Math.round((g + mm) * 255), b: Math.round((b + mm) * 255), a: 1 })
}

/** The contrast of a color in its field: the verdict and the text it's about, or why it can't be checked. */
export interface Evaluation {
  state: 'ok' | 'image' | 'none'
  verdict?: ContrastVerdict
  /** The text compared: the background's lowest text, or the field's own. */
  text?: Omit<TextSample, 'color'> & { color?: string }
  /** The other color of the pair. */
  against?: string
}

export function evaluate(value: string, spec: ContrastSpec): Evaluation {
  const color = parseColor(value)
  if (!color) return { state: 'none' }
  if (spec.field === 'text') {
    if (spec.image && !spec.background) return { state: 'image', text: spec.text }
    const bg = spec.background ?? '#FFFFFF'
    const against = parseColor(bg)
    if (!against) return { state: 'none' }
    return { state: 'ok', verdict: verdict(color, against, spec.text.sizePx, spec.text.bold), text: spec.text, against: bg }
  }
  // A background: the text that does worst on it, by how far it is from what it needs.
  let worst: Evaluation = { state: 'none' }
  let score = Infinity
  for (const t of spec.texts) {
    const tc = parseColor(t.color)
    if (!tc) continue
    const v = verdict(tc, color, t.sizePx, t.bold)
    if (v.ratio / v.needsAA < score) {
      score = v.ratio / v.needsAA
      worst = { state: 'ok', verdict: v, text: t, against: t.color }
    }
  }
  return worst
}

/** "2.1:1 · Fails AA", "4.5:1 · Passes AA", "7.0:1 · Passes AAA", "3.4:1 · Passes AA for large text". */
function chipText(v: ContrastVerdict, t: ReturnType<typeof useT>): { text: string; tone: 'f' | 'p' } {
  const ratio = formatRatio(v.ratio)
  if (!v.passesAA) return { text: t(m.chipFails, { ratio }), tone: 'f' }
  if (v.large && !v.aaNormal) return { text: t(m.chipLarge, { ratio }), tone: 'p' }
  if (v.passesAAA) return { text: t(m.chipAAA, { ratio }), tone: 'p' }
  return { text: t(m.chipAA, { ratio }), tone: 'p' }
}

/** The small chip a color field keeps when its contrast fails (the picker closed). */
export function ContrastChip({ value, spec }: { value: string; spec: ContrastSpec }) {
  const t = useT()
  const e = evaluate(value, spec)
  if (e.state !== 'ok' || !e.verdict || e.verdict.passesAA) return null
  return <span className="a11y-cchip f">{t(m.chipFailsShort, { ratio: formatRatio(e.verdict.ratio) })}</span>
}

function HexInput({ value, label, onCommit, className }: { value: string; label: string; onCommit: (v: string) => void; className?: string }) {
  const [draft, setDraft] = useState(value)
  const focused = useRef(false)
  useEffect(() => {
    if (!focused.current) setDraft(value)
  }, [value])
  const commit = () => {
    if (draft.trim() !== value) onCommit(draft.trim())
  }
  return (
    <input
      className={`in mono ${className ?? ''}`}
      aria-label={label}
      value={/^#[0-9a-f]{3,8}$/i.test(draft) ? draft.toUpperCase() : draft}
      placeholder="none"
      onFocus={() => (focused.current = true)}
      onBlur={() => {
        focused.current = false
        commit()
      }}
      onChange={(e) => setDraft(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          commit()
          ;(e.target as HTMLInputElement).blur()
        } else if (e.key === 'Escape') {
          setDraft(value)
          ;(e.target as HTMLInputElement).blur()
        }
      }}
    />
  )
}

const PASS_ICON = <path d="m3 8 3.5 3.5L13 5" />
const FAIL_ICON = <path d="M4 4l8 8M12 4l-8 8" />

function PassFail({ ok }: { ok: boolean }) {
  const t = useT()
  return (
    <span className={`a11y-pf ${ok ? 'y' : 'n'}`}>
      <svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        {ok ? PASS_ICON : FAIL_ICON}
      </svg>
      {ok ? t(m.pass) : t(m.fail)}
    </span>
  )
}

/** What the text is called in the picker: "Button text", "Headline text, 26 px bold". */
function whatOf(text: Omit<TextSample, 'color'>, t: ReturnType<typeof useT>): string {
  return text.kind === 'button' ? t(m.buttonText) : text.kind === 'heading' ? t(m.headlineText) : t(m.bodyText)
}

/**
 * A color field with its contrast (ComposerContrast): the swatch opens the
 * picker (shade, hue, hex, eyedropper) with the ratio as it changes, AA and
 * AAA for normal and large text, and colors that pass.
 */
export function ContrastColorField({
  value,
  onCommit,
  label,
  spec,
  gap = 8,
  children,
}: {
  value: string
  onCommit: (v: string) => void
  label: string
  spec: ContrastSpec
  gap?: number
  children?: ReactNode
}) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [live, setLive] = useState<string>()
  const timer = useRef<number | undefined>(undefined)
  const shown = live ?? value
  const commitSoon = (v: string) => {
    setLive(v)
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(() => {
      onCommit(v)
      setLive(undefined)
    }, 250)
  }
  const commitNow = (v: string) => {
    window.clearTimeout(timer.current)
    setLive(undefined)
    onCommit(v)
  }
  useEffect(() => () => window.clearTimeout(timer.current), [])
  return (
    <>
      <div className="flex items-center" style={{ gap }}>
        <button
          type="button"
          className="sw a11y-swatch"
          aria-label={t(m.pickerLabel, { field: label })}
          aria-expanded={open}
          style={{ background: shown || 'transparent' }}
          onClick={() => setOpen(!open)}
        />
        <HexInput label={label} value={shown} onCommit={commitNow} className="min-w-0 flex-1" />
        {/* The closed field keeps a chip when the background fails (a text color's row is too narrow for one). */}
        {!open && spec.field === 'background' ? <ContrastChip value={shown} spec={spec} /> : null}
        {children}
      </div>
      {open ? <Picker label={label} value={shown} spec={spec} onLive={commitSoon} onPick={commitNow} onClose={() => setOpen(false)} /> : null}
    </>
  )
}

function Picker({
  label,
  value,
  spec,
  onLive,
  onPick,
  onClose,
}: {
  label: string
  value: string
  spec: ContrastSpec
  onLive: (v: string) => void
  onPick: (v: string) => void
  onClose: () => void
}) {
  const t = useT()
  const parsed = parseColor(value) ?? { r: 255, g: 255, b: 255, a: 1 }
  const [hsv, setHsv] = useState<HSV>(() => toHSV(parsed))
  const area = useRef<HTMLDivElement>(null)
  // The suggestions are made for the color the picker opened with: they
  // stay while you drag.
  const [sugs] = useState((): Suggestion[] => {
    const e = evaluate(value, spec)
    if (e.state !== 'ok' || !e.verdict || e.verdict.passesAA) return []
    const from = parseColor(value)
    const against = parseColor(e.against ?? '#FFFFFF')
    if (!from || !against) return []
    return suggestions(from, against, e.verdict.needsAA, e.verdict.needsAAA, spec.others)
  })
  const set = (next: HSV) => {
    setHsv(next)
    onLive(fromHSV(next))
  }
  const fromPointer = (e: PointerEvent<HTMLDivElement>) => {
    const r = area.current?.getBoundingClientRect()
    if (!r) return
    const s = Math.min(1, Math.max(0, (e.clientX - r.left) / r.width))
    const v = 1 - Math.min(1, Math.max(0, (e.clientY - r.top) / r.height))
    set({ ...hsv, s, v })
  }
  const onAreaKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const step = e.shiftKey ? 0.1 : 0.01
    const next = { ...hsv }
    if (e.key === 'ArrowLeft') next.s = Math.max(0, hsv.s - step)
    else if (e.key === 'ArrowRight') next.s = Math.min(1, hsv.s + step)
    else if (e.key === 'ArrowUp') next.v = Math.min(1, hsv.v + step)
    else if (e.key === 'ArrowDown') next.v = Math.max(0, hsv.v - step)
    else if (e.key === 'Escape') return onClose()
    else return
    e.preventDefault()
    set(next)
  }
  const e = evaluate(value, spec)
  const eyedropper = typeof window !== 'undefined' && 'EyeDropper' in window
  return (
    <div className="a11y-picker" role="dialog" aria-label={label}>
      <div
        ref={area}
        className="a11y-sv"
        role="slider"
        tabIndex={0}
        aria-label={t(m.shade)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(hsv.s * 100)}
        aria-valuetext={value}
        style={{ background: `linear-gradient(to top,#000,rgba(0,0,0,0)),linear-gradient(to right,#fff,hsl(${Math.round(hsv.h)},100%,50%))` }}
        onPointerDown={(ev) => {
          ev.currentTarget.setPointerCapture(ev.pointerId)
          fromPointer(ev)
        }}
        onPointerMove={(ev) => {
          if (ev.buttons) fromPointer(ev)
        }}
        onKeyDown={onAreaKey}
      >
        <span className="a11y-sv-dot" style={{ left: `${hsv.s * 100}%`, top: `${(1 - hsv.v) * 100}%`, background: value }} />
      </div>
      <input
        type="range"
        className="a11y-hue"
        aria-label={t(m.hue)}
        min={0}
        max={359}
        value={Math.round(hsv.h)}
        onChange={(ev) => set({ ...hsv, h: Number(ev.target.value) })}
        onKeyDown={(ev) => ev.key === 'Escape' && onClose()}
      />
      <div className="flex items-center gap-1.5">
        <span className="a11y-aa" style={{ background: value }} aria-hidden="true" />
        <HexInput
          label={label}
          value={value}
          className="min-w-0 flex-1"
          onCommit={(v) => {
            const c = parseColor(v)
            if (c) setHsv(toHSV(c))
            onPick(v)
          }}
        />
        {eyedropper ? (
          <button
            type="button"
            className="btn ic"
            aria-label={t(m.eyedropper)}
            onClick={async () => {
              try {
                const picked = await new (window as unknown as { EyeDropper: new () => { open(): Promise<{ sRGBHex: string }> } }).EyeDropper().open()
                const c = parseColor(picked.sRGBHex)
                if (c) setHsv(toHSV(c))
                onPick(picked.sRGBHex.toUpperCase())
              } catch {
                /* cancelled */
              }
            }}
          >
            <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="m10.2 2.6 3.2 3.2-1.6 1.6-3.2-3.2zM8.6 4.2 3 9.8V13h3.2l5.6-5.6" />
            </svg>
          </button>
        ) : null}
      </div>
      <ContrastSection evaluation={e} spec={spec} sugs={sugs} onPick={onPick} />
    </div>
  )
}

function ContrastSection({ evaluation: e, spec, sugs, onPick }: { evaluation: Evaluation; spec: ContrastSpec; sugs: Suggestion[]; onPick: (v: string) => void }) {
  const t = useT()
  if (e.state === 'image' && e.text) {
    return (
      <div className="a11y-cp-sec">
        <span className="a11y-cchip u">{t(m.cantCheck)}</span>
        <span className="k text-ink-nav">{t(m.overImage, { what: whatOf(e.text, t), image: spec.field === 'text' ? (spec.image ?? '') : '' })}</span>
        <span className="k">{t(m.overImageDetail)}</span>
        <span className="k">{t(m.notCounted)}</span>
      </div>
    )
  }
  if (e.state !== 'ok' || !e.verdict || !e.text) return null
  const v = e.verdict
  const chip = chipText(v, t)
  const what = whatOf(e.text, t)
  return (
    <div className="a11y-cp-sec">
      <div className="flex flex-wrap items-center gap-2">
        <span className={`a11y-cchip ${chip.tone}`}>{chip.text}</span>
        {v.passesAA && !v.passesAAA && sugs.length ? <span className="k">{t(m.asYouDrag)}</span> : null}
      </div>
      <span className="k text-ink-nav">
        {spec.field === 'background'
          ? t(m.againstBackground, { what, color: (e.against ?? '').toUpperCase() })
          : t(m.againstText, { what: t(m.textOf, { what, px: Math.round(e.text.sizePx), bold: e.text.bold ? 'true' : 'false' }), color: (e.against ?? '').toUpperCase() })}
      </span>
      {spec.field === 'background' && spec.texts.length > 1 ? <span className="k">{t(m.worstFirst)}</span> : null}
      <table className="a11y-wcag">
        <thead>
          <tr>
            <th scope="col">{t(m.colText)}</th>
            <th scope="col">{t(m.colAA)}</th>
            <th scope="col">{t(m.colAAA)}</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>{t(m.normalNeeds)}</td>
            <td>
              <PassFail ok={v.aaNormal} />
            </td>
            <td>
              <PassFail ok={v.aaaNormal} />
            </td>
          </tr>
          <tr>
            <td>{t(m.largeNeeds)}</td>
            <td>
              <PassFail ok={v.aaLarge} />
            </td>
            <td>
              <PassFail ok={v.aaaLarge} />
            </td>
          </tr>
        </tbody>
      </table>
      {sugs.length ? (
        <>
          <span className="a11y-sug-h">{t(m.suggestions)}</span>
          {sugs.map((s) => (
            <button key={s.color} type="button" className="a11y-sug" onClick={() => onPick(s.color)}>
              <span
                className="a11y-aa"
                style={spec.field === 'background' ? { background: s.color, color: e.against } : { background: e.against, color: s.color }}
                aria-hidden="true"
              >
                Aa
              </span>
              <span className="flex min-w-0 flex-1 flex-col">
                <b className="mono text-[12px]">{s.color}</b>
                <span className="k">{s.reason === 'other_buttons' ? t(m.reasonOtherButtons) : s.reason === 'nearest_aa' ? t(m.reasonNearestAA) : t(m.reasonAAA)}</span>
              </span>
              <span className="text-[12.5px] font-bold">{`${s.ratio.toFixed(1)}:1`}</span>
            </button>
          ))}
          <span className="k">{t(m.suggestionsHelp)}</span>
        </>
      ) : (
        <>
          <span className="a11y-sug-h">{t(m.suggestions)}</span>
          <span className="k">{t(m.nothingToSuggest)}</span>
          {v.aaaNormal ? <span className="k">{t(m.passesAll)}</span> : null}
          {v.large && v.passesAA && !v.aaNormal ? <span className="k">{t(m.largeNote, { px: Math.round(e.text.sizePx), bold: e.text.bold ? 'true' : 'false' })}</span> : null}
        </>
      )}
    </div>
  )
}
