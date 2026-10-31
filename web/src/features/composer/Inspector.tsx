import type { Component } from 'grapesjs'
import { useEffect, useId, useMemo, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { useComposer } from './context'
import { AltTextAssist, DecorativeToggle, useAltTextAvailable, useAltVersionLabel } from '@/features/ai/composer/fixes'
import {
  columnsOf,
  columnWidth,
  descendants,
  effective,
  effectiveBox,
  FONTS,
  fontLabel,
  formatBorder,
  hasClass,
  headDefaults,
  hexColor,
  ownAttrs,
  parseBorder,
  pxNumber,
  setAttr,
  setBox,
  setBoxSide,
  toggleClass,
  VISIBILITY,
  withPx,
  type HeadDefaults,
} from './editor/attrs'
import {
  BLOCK_SPACING,
  BUTTON_CORNERS,
  BUTTON_SIZES,
  COLUMN_SPACING,
  CORNERS,
  evenPreset,
  HERO_SPACING,
  lengthPreset,
  LINE_LENGTHS,
  LINE_WEIGHTS,
  presetOf,
  SECTION_SPACING,
  verticalPreset,
} from './stylePresets'
import { mjChildren, sectionLabel, sectionName, sectionOf, sectionWord, tagOf } from './editor/labels'
import { withVisibilityStyles } from './editor/mjmlDoc'
import { TextRoleProp } from '@/features/accessibility/composer/TextRole'
import { ContrastColorField } from '@/features/accessibility/composer/ColorPicker'
import { useContrastSpec, type ContrastSpec } from '@/features/accessibility/composer/contrastSpec'
import { NameInLayers, VersionsWithBlock, VisibilitySection } from '@/features/conditional/Inspector'
import { m as cm } from '@/features/conditional/messages'
import { useConditional } from '@/features/conditional/state'
import { PollSettings } from '@/features/polls/Inspector'
import { UpgradeSettings } from '@/features/earn/composer/UpgradeSettings'
import { SponsorFoot, SponsorSettings } from '@/features/earn/sponsors/composer/SponsorSettings'
import { ShowOnProp } from '@/features/website/composer/campaign'
import { FillFromProduct } from '@/features/products/composer'
import { FillFromStore, StoreBlockStyle, storeSectionOf } from '@/features/commerce/composer'
import { t as tt, useT, type MessageDescriptor } from '@/i18n'
import { mi } from './inspector.messages'
import { mh } from './chrome.messages'
import { ToolIcon } from './icons'
import { useCampaignLanguages } from '@/features/languages/composer'

/** A text input that edits a draft and commits on Enter or blur (one undo step per edit). */
function DraftInput({
  value,
  onCommit,
  label,
  className = 'in',
  mono,
  placeholder,
  id,
  disabled,
  style,
}: {
  value: string
  onCommit: (v: string) => void
  label?: string
  className?: string
  mono?: boolean
  placeholder?: string
  id?: string
  disabled?: boolean
  style?: CSSProperties
}) {
  const [draft, setDraft] = useState(value)
  const focused = useRef(false)
  useEffect(() => {
    if (!focused.current) setDraft(value)
  }, [value])
  const commit = () => {
    if (draft !== value) onCommit(draft)
  }
  return (
    <input
      id={id}
      className={`${className}${mono ? ' mono' : ''}`}
      aria-label={label}
      value={draft}
      placeholder={placeholder}
      disabled={disabled}
      style={style}
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

function Prop({ title, children, id }: { title: ReactNode; children: ReactNode; id?: string }) {
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {title}
      </span>
      {children}
    </div>
  )
}

/** Colour swatch (native picker) + hex input; with `contrast`, the accessibility picker. */
function ColorField({
  value,
  onCommit,
  label,
  children,
  gap = 8,
  contrast,
}: {
  value: string
  onCommit: (v: string) => void
  label: string
  children?: ReactNode
  gap?: number
  contrast?: ContrastSpec
}) {
  const t = useT()
  const timer = useRef<number | undefined>(undefined)
  const hex = hexColor(value)
  if (contrast)
    return (
      <ContrastColorField value={value} onCommit={onCommit} label={label} spec={contrast} gap={gap}>
        {children}
      </ContrastColorField>
    )
  return (
    <div className="flex items-center" style={{ gap }}>
      <input
        type="color"
        className="sw"
        aria-label={t(mi.colourPicker, { label })}
        value={hex || '#ffffff'}
        onChange={(e) => {
          const v = e.target.value.toUpperCase()
          window.clearTimeout(timer.current)
          timer.current = window.setTimeout(() => onCommit(v), 250)
        }}
        style={{ background: value || 'transparent' }}
      />
      <DraftInput label={label} mono value={/^#[0-9a-f]{3,8}$/i.test(value) ? value.toUpperCase() : value} placeholder={t(mi.none)} onCommit={(v) => onCommit(v.trim())} style={{ flex: 1 }} />
      {children}
    </div>
  )
}

const SIDE_LABELS = [
  ['top', mi.sideTop],
  ['right', mi.sideRight],
  ['bottom', mi.sideBottom],
  ['left', mi.sideLeft],
] as const

/** A small labelled input (Top, Column 1, Width…). */
function SubField({ label, value, placeholder, mono, onCommit, id: given }: { label: string; value: string; placeholder?: string; mono?: boolean; onCommit: (v: string) => void; id?: string }) {
  const own = useId()
  const id = given ?? own
  // The board wraps label and input in a label.k, so the input inherits its 500 weight.
  return (
    <div className="k flex min-w-0 flex-col" style={{ gap: 2 }}>
      <label className="k" htmlFor={id}>
        {label}
      </label>
      <DraftInput id={id} mono={mono} value={value} placeholder={placeholder} onCommit={onCommit} />
    </div>
  )
}

function BoxField({ c, k, head }: { c: Component; k: string; head: HeadDefaults }) {
  const t = useT()
  const box = effectiveBox(c, k, head)
  return (
    <div className="row4">
      {SIDE_LABELS.map(([side, label], i) => (
        <SubField key={side} label={t(label)} value={pxNumber(box[i]!)} onCommit={(v) => setBoxSide(c, k, side, withPx(v || '0'), head)} />
      ))}
    </div>
  )
}

const alignWords: Record<string, MessageDescriptor> = {
  left: mi.alignLeft,
  center: mi.alignCenter,
  right: mi.alignRight,
  top: mi.alignTop,
  middle: mi.alignMiddle,
  bottom: mi.alignBottom,
}

function AlignField({ value, onChange, label, options = ['left', 'center', 'right'] }: { value: string; onChange: (v: string) => void; label: string; options?: string[] }) {
  const t = useT()
  return (
    <div className="seg" role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o} type="button" aria-pressed={value === o} onClick={() => onChange(o)}>
          {alignWords[o] ? t(alignWords[o]) : o[0]!.toUpperCase() + o.slice(1)}
        </button>
      ))}
    </div>
  )
}

function SelectField({ value, onChange, label, options }: { value: string; onChange: (v: string) => void; label: string; options: { value: string; label: string }[] }) {
  const t = useT()
  const has = options.some((o) => o.value === value)
  return (
    <select className="in" aria-label={label} value={value} onChange={(e) => onChange(e.target.value)}>
      {has ? null : <option value={value}>{value || t(mi.default)}</option>}
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  )
}

function useHead() {
  const { head } = useComposer()
  return useMemo(() => headDefaults(head), [head])
}

function BackgroundProp({ c, k = 'background-color', image, title: given }: { c: Component; k?: string; image?: boolean; title?: string }) {
  const t = useT()
  const title = given ?? t(mi.background)
  const head = useHead()
  const { openDialog, setAssetTarget } = useComposer()
  const id = useId()
  const contrast = useContrastSpec(c, k)
  return (
    <Prop title={title} id={id}>
      <ColorField label={title} value={effective(c, k, head)} onCommit={(v) => setAttr(c, k, v)} contrast={contrast}>
        {image ? (
          <button
            type="button"
            className="btn"
            style={{ minHeight: 30 }}
            onClick={() => {
              setAssetTarget({ component: c, attr: 'background-url' })
              openDialog('assets')
            }}
          >
            {t(mi.image)}
          </button>
        ) : null}
      </ColorField>
      {image && ownAttrs(c)['background-url'] ? (
        <span className="k flex items-center gap-2">
          <span className="mono min-w-0 flex-1 truncate">{ownAttrs(c)['background-url']}</span>
          <button type="button" className="btn sm" onClick={() => setAttr(c, 'background-url', '')}>
            {t(mi.remove)}
          </button>
        </span>
      ) : null}
    </Prop>
  )
}

function PaddingProp({ c, k = 'padding', title }: { c: Component; k?: string; title?: string }) {
  const t = useT()
  const head = useHead()
  const id = useId()
  return (
    <Prop title={title ?? t(mi.padding)} id={id}>
      <BoxField c={c} k={k} head={head} />
    </Prop>
  )
}

function AlignProp({ c, k = 'align' }: { c: Component; k?: string }) {
  const t = useT()
  const head = useHead()
  const id = useId()
  return (
    <Prop title={t(mi.alignment)} id={id}>
      <AlignField label={t(mi.alignment)} value={effective(c, k, head)} onChange={(v) => setAttr(c, k, v)} />
    </Prop>
  )
}

function BorderProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  const id = useId()
  const b = parseBorder(effective(c, 'border', head))
  return (
    <Prop title={t(mi.border)} id={id}>
      <div className="row2">
        <SelectField
          label={t(mi.borderStyle)}
          value={b.style}
          options={[
            { value: 'none', label: t(mi.styleNone) },
            { value: 'solid', label: t(mi.styleSolid) },
            { value: 'dashed', label: t(mi.styleDashed) },
            { value: 'dotted', label: t(mi.styleDotted) },
          ]}
          onChange={(style) => setAttr(c, 'border', formatBorder({ ...b, style, width: b.width === '0px' ? '1px' : b.width }))}
        />
        <DraftInput label={t(mi.borderWidth)} value={b.width} onCommit={(w) => setAttr(c, 'border', formatBorder({ ...b, style: b.style === 'none' ? 'solid' : b.style, width: withPx(w) }))} />
      </div>
      {b.style !== 'none' ? <ColorField label={t(mi.borderColour)} value={b.color} onCommit={(color) => setAttr(c, 'border', formatBorder({ ...b, color }))} /> : null}
    </Prop>
  )
}

function TextProp({ c, k, title, mono, placeholder, px }: { c: Component; k: string; title: string; mono?: boolean; placeholder?: string; px?: boolean }) {
  const head = useHead()
  const id = useId()
  return (
    <Prop title={title} id={id}>
      <DraftInput label={title} mono={mono} value={effective(c, k, head)} placeholder={placeholder} onCommit={(v) => setAttr(c, k, px ? withPx(v) : v.trim())} />
    </Prop>
  )
}

function ColumnWidthsProp({ section }: { section: Component }) {
  const t = useT()
  const cols = columnsOf(section)
  const id = useId()
  if (cols.length < 2) return null
  return (
    <Prop title={t(mi.columnWidths)} id={id}>
      <div className={cols.length === 2 ? 'row2' : cols.length === 3 ? 'grid grid-cols-3 gap-1.5' : 'row4'}>
        {cols.map((col, i) => (
          <SubField key={col.cid} label={t(mi.columnN, { n: i + 1 })} value={columnWidth(col)} onCommit={(v) => setAttr(col, 'width', /^\d+(\.\d+)?$/.test(v.trim()) ? `${v.trim()}%` : v.trim())} />
        ))}
      </div>
    </Prop>
  )
}

/** The text that carries the section's body copy: the one with the most words (headings are short). */
function bodyText(texts: Component[]): Component {
  let best = texts[0]!
  let most = -1
  for (const t of texts) {
    const n = (t.getEl()?.textContent ?? '').trim().length
    if (n > most) {
      best = t
      most = n
    }
  }
  return best
}

function fontOptions(current: string) {
  const opts = FONTS.map((f) => ({ value: f.value, label: f.label }))
  if (current && !opts.some((o) => o.value === current)) opts.unshift({ value: current, label: fontLabel(current) })
  return opts
}

/** Typography of `targets` (one component, or every text in a section). */
function TypographyProp({ targets, title, sample, weight }: { targets: Component[]; title: string; sample: Component; weight?: boolean }) {
  const t = useT()
  const head = useHead()
  const id = useId()
  const set = (k: string, v: string) => targets.forEach((x) => setAttr(x, k, v))
  const family = effective(sample, 'font-family', head)
  const color = effective(sample, 'color', head)
  const contrast = useContrastSpec(sample, 'color')
  return (
    <Prop title={title} id={id}>
      <div className="row2">
        <SelectField label={title} value={family} options={fontOptions(family)} onChange={(v) => set('font-family', v)} />
        <DraftInput label={t(mi.fontSize)} value={effective(sample, 'font-size', head)} onCommit={(v) => set('font-size', withPx(v))} />
      </div>
      <div className="row2">
        {/* In a two-column row the board spaces swatch and hex 6px apart. */}
        <ColorField label={t(mi.textColour)} value={color} onCommit={(v) => set('color', v)} gap={6} contrast={contrast} />
        {weight ? (
          <SelectField
            label={t(mi.fontWeight)}
            value={effective(sample, 'font-weight', head) || 'normal'}
            options={weightOptions()}
            onChange={(v) => set('font-weight', v)}
          />
        ) : (
          <DraftInput label={t(mi.lineHeight)} value={effective(sample, 'line-height', head)} onCommit={(v) => set('line-height', v.trim())} />
        )}
      </div>
    </Prop>
  )
}

/** The font weights the selects offer. */
function weightOptions() {
  return [
    { value: 'normal', label: tt(mi.weightRegular) },
    { value: '500', label: tt(mi.weightMedium) },
    { value: '600', label: tt(mi.weightSemibold) },
    { value: '700', label: tt(mi.weightBold) },
  ]
}

function sectionPhrase(sec: Component | undefined): string {
  if (!sec) return tt(mi.phraseEmail)
  const l = sectionLabel(sec)
  const m = /^(\d+) columns$/.exec(l)
  if (m) return tt(mi.phraseColumns, { count: Number(m[1]) })
  return l.endsWith('section') ? l : tt(mi.phraseNamed, { name: sectionWord(l) })
}

/** The subtitle under the tag name in the inspector. */
export function describe(c: Component, tab: 'style' | 'settings'): string {
  const tag = tagOf(c)
  const sec = sectionOf(c)
  switch (tag) {
    case 'mj-body':
      return tt(mi.descBody)
    case 'mj-section':
    case 'mj-wrapper':
    case 'mj-hero': {
      const l = sec === c ? sectionName(c) : sectionWord(tag.replace('mj-', ''))
      return tab === 'style' ? tt(mi.descStyle, { name: l }) : tt(mi.descSettings, { name: l })
    }
    case 'mj-column': {
      const parent = c.parent()
      const i = parent ? columnsOf(parent).indexOf(c) : 0
      return tt(mi.descColumn, { n: i + 1, section: sectionPhrase(sec) })
    }
    case 'mj-raw':
      return tt(mi.descRaw)
  }
  return tt(mi.descElement, { element: sectionWord(tag.replace('mj-', '')), section: sectionPhrase(sec) })
}

type Spacing = keyof typeof SECTION_SPACING

/** One choice among presets (S · M · L, Thin · Medium · Thick…): the one the block matches is pressed. */
function ChoiceProp<K extends string>({
  title,
  options,
  current,
  onPick,
  hint,
}: {
  title: string
  options: readonly { key: K; label: string }[]
  current?: K
  onPick: (key: K) => void
  hint?: string
}) {
  const id = useId()
  return (
    <Prop title={title} id={id}>
      <div className="seg3">
        {options.map((o) => (
          <button key={o.key} type="button" aria-pressed={current === o.key} onClick={() => onPick(o.key)}>
            {o.label}
          </button>
        ))}
      </div>
      {hint ? <span className="k">{hint}</span> : null}
    </Prop>
  )
}

/** The px of each side of a box attribute, as the block has it now. */
function useBox(c: Component, k = 'padding'): [number, number, number, number] {
  const head = useHead()
  return effectiveBox(c, k, head).map((v) => Number(pxNumber(v)) || 0) as [number, number, number, number]
}

const SML = ['S', 'M', 'L'] as const

function SpacingProp({ c }: { c: Component }) {
  const t = useT()
  const [top, right, bottom, left] = useBox(c)
  const current = (Object.keys(SECTION_SPACING) as Spacing[]).find(
    (k) => top === SECTION_SPACING[k][0] && bottom === SECTION_SPACING[k][0] && left === SECTION_SPACING[k][1] && right === SECTION_SPACING[k][1],
  )
  const same = top === bottom && left === right
  return (
    <ChoiceProp
      title={t(mh.spacing)}
      options={SML.map((k) => ({ key: k, label: k }))}
      current={current}
      onPick={(k) => setBox(c, 'padding', `${SECTION_SPACING[k][0]}px ${SECTION_SPACING[k][1]}px`)}
      hint={same ? t(mh.spacingHint, { v: top, h: left }) : t(mh.spacingCustom, { padding: `${top} ${right} ${bottom} ${left}` })}
    />
  )
}

/** Space above and below a block (S · M · L); its sides stay as they are. */
function BlockSpacingProp({ c, presets = BLOCK_SPACING }: { c: Component; presets?: Record<(typeof SML)[number], number> }) {
  const t = useT()
  const [top, right, bottom, left] = useBox(c)
  return (
    <ChoiceProp
      title={t(mh.spacing)}
      options={SML.map((k) => ({ key: k, label: k }))}
      current={verticalPreset(presets, top, bottom)}
      onPick={(k) => setBox(c, 'padding', `${presets[k]}px ${right}px ${presets[k]}px ${left}px`)}
      hint={top === bottom ? t(mh.spacingBlockHint, { v: top }) : t(mh.spacingCustom, { padding: `${top} ${right} ${bottom} ${left}` })}
    />
  )
}

/** Space inside a column, the same on every side (None · S · M · L). */
function InsideSpacingProp({ c }: { c: Component }) {
  const t = useT()
  const box = useBox(c)
  const even = box.every((n) => n === box[0])
  return (
    <ChoiceProp
      title={t(mh.spacingInside)}
      options={(['none', ...SML] as const).map((k) => ({ key: k, label: k === 'none' ? t(mh.spacingNone) : k }))}
      current={evenPreset(COLUMN_SPACING, box)}
      onPick={(k) => setBox(c, 'padding', `${COLUMN_SPACING[k]}px`)}
      hint={even ? t(mh.spacingInsideHint, { v: box[0] }) : t(mh.spacingCustom, { padding: box.join(' ') })}
    />
  )
}

/** Corners: square, rounded (and a pill, for buttons). */
function CornersProp({ c, pill }: { c: Component; pill?: boolean }) {
  const t = useT()
  const head = useHead()
  const presets: Record<string, string> = pill ? BUTTON_CORNERS : CORNERS
  const value = effective(c, 'border-radius', head)
  const current = lengthPreset(presets, value || '0px')
  const labels: Record<string, string> = { square: t(mh.cornersSquare), rounded: t(mh.cornersRounded), pill: t(mh.cornersPill) }
  return (
    <ChoiceProp
      title={t(mh.corners)}
      options={Object.keys(presets).map((k) => ({ key: k, label: labels[k]! }))}
      current={current}
      onPick={(k) => setAttr(c, 'border-radius', presets[k]!)}
      hint={current ? undefined : t(mh.customValue, { value })}
    />
  )
}

/** A text's size in its font: Small, Body or Large (as the section's Text style). */
function TextSizeProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  const font = fontLabel(effective(c, 'font-family', head) || head.tags['mj-all']?.['font-family'] || '')
  const size = Number(pxNumber(effective(c, 'font-size', head))) || 13
  const options = [
    { value: '14', label: t(mh.textBody, { font, size: 14 }) },
    { value: '13', label: t(mh.textSmall, { font, size: 13 }) },
    { value: '16', label: t(mh.textLarge, { font, size: 16 }) },
  ]
  if (![13, 14, 16].includes(size)) options.unshift({ value: String(size), label: t(mh.textCustom, { font, size }) })
  const id = useId()
  return (
    <Prop title={t(mh.textStyle)} id={id}>
      <SelectField label={t(mh.textStyle)} value={String(size)} options={options} onChange={(v) => setAttr(c, 'font-size', `${v}px`)} />
    </Prop>
  )
}

/** Regular or bold; other weights show as they are, and Advanced has them all. */
function WeightProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  const w = effective(c, 'font-weight', head) || 'normal'
  const current = w === 'normal' || w === '400' ? 'regular' : w === 'bold' || w === '700' ? 'bold' : undefined
  return (
    <ChoiceProp
      title={t(mi.weight)}
      options={[
        { key: 'regular', label: t(mi.weightRegular) },
        { key: 'bold', label: t(mi.weightBold) },
      ]}
      current={current}
      onPick={(k) => setAttr(c, 'font-weight', k === 'bold' ? 'bold' : 'normal')}
      hint={current ? undefined : t(mh.customValue, { value: w })}
    />
  )
}

/** A hero's content: at the top, middle or bottom. */
function HeroAlignProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  return (
    <ChoiceProp
      title={t(mi.verticalAlignment)}
      options={(['top', 'middle', 'bottom'] as const).map((k) => ({ key: k, label: t(alignWords[k]!) }))}
      current={effective(c, 'vertical-align', head) as 'top' | 'middle' | 'bottom'}
      onPick={(v) => setAttr(c, 'vertical-align', v)}
    />
  )
}

/** A button's size: Small, Medium or Large (its inner padding). */
function ButtonSizeProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  const value = effective(c, 'inner-padding', head)
  const current = presetOf(BUTTON_SIZES, value)
  return (
    <ChoiceProp
      title={t(mi.size)}
      options={[
        { key: 'S', label: t(mh.sizeSmall) },
        { key: 'M', label: t(mh.sizeMedium) },
        { key: 'L', label: t(mh.sizeLarge) },
      ]}
      current={current}
      onPick={(k) => setAttr(c, 'inner-padding', BUTTON_SIZES[k])}
      hint={current ? undefined : t(mh.customValue, { value })}
    />
  )
}

/** A button as wide as its text, or as its column. */
function ButtonWidthProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  const w = effective(c, 'width', head)
  const current = !w || w === 'auto' ? 'fit' : w === '100%' ? 'full' : undefined
  return (
    <ChoiceProp
      title={t(mi.width)}
      options={[
        { key: 'fit', label: t(mh.widthFit) },
        { key: 'full', label: t(mh.widthFull) },
      ]}
      current={current}
      onPick={(k) => setAttr(c, 'width', k === 'full' ? '100%' : '')}
      hint={current ? undefined : t(mh.customValue, { value: w })}
    />
  )
}

/** An image's width and height ("auto" keeps its proportions). */
function ImageSizeProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  return (
    <Prop title={t(mi.size)}>
      <div className="row2">
        <SubField label={t(mi.width)} value={effective(c, 'width', head)} placeholder={'auto'} onCommit={(v) => setAttr(c, 'width', withPx(v))} />
        <SubField label={t(mi.height)} value={effective(c, 'height', head)} placeholder={'auto'} onCommit={(v) => setAttr(c, 'height', withPx(v))} />
      </div>
    </Prop>
  )
}

/** A divider's line: style, width in px and colour (the MJML attributes). */
function DividerLineProp({ c, title }: { c: Component; title?: string }) {
  const t = useT()
  const head = useHead()
  return (
    <Prop title={title ?? t(mi.line)}>
      <div className="row2">
        <SelectField
          label={t(mi.lineStyle)}
          value={effective(c, 'border-style', head)}
          options={[
            { value: 'solid', label: t(mi.styleSolid) },
            { value: 'dashed', label: t(mi.styleDashed) },
            { value: 'dotted', label: t(mi.styleDotted) },
          ]}
          onChange={(v) => setAttr(c, 'border-style', v)}
        />
        <DraftInput label={t(mi.lineWidth)} value={effective(c, 'border-width', head)} onCommit={(v) => setAttr(c, 'border-width', withPx(v))} />
      </div>
      <ColorField label={t(mi.lineColour)} value={effective(c, 'border-color', head)} onCommit={(v) => setAttr(c, 'border-color', v)} />
    </Prop>
  )
}

/** A divider's line: Thin, Medium or Thick. */
function DividerWeightProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  const value = effective(c, 'border-width', head)
  const current = lengthPreset(LINE_WEIGHTS, value)
  return (
    <ChoiceProp
      title={t(mi.line)}
      options={[
        { key: 'thin', label: t(mh.lineThin) },
        { key: 'medium', label: t(mh.lineMedium) },
        { key: 'thick', label: t(mh.lineThick) },
      ]}
      current={current}
      onPick={(k) => setAttr(c, 'border-width', LINE_WEIGHTS[k])}
      hint={current ? undefined : t(mh.customValue, { value })}
    />
  )
}

/** A divider across the whole column, three quarters of it or half. */
function DividerLengthProp({ c }: { c: Component }) {
  const t = useT()
  const head = useHead()
  const value = effective(c, 'width', head)
  const current = lengthPreset(LINE_LENGTHS, value)
  return (
    <ChoiceProp
      title={t(mh.lineLength)}
      options={[
        { key: 'full', label: t(mh.lengthFull) },
        { key: 'wide', label: t(mh.lengthWide) },
        { key: 'short', label: t(mh.lengthShort) },
      ]}
      current={current}
      onPick={(k) => setAttr(c, 'width', LINE_LENGTHS[k])}
      hint={current ? undefined : t(mh.customValue, { value })}
    />
  )
}

/** "padding 10 25 10 25" (and a button's inner padding): what a block's Advanced holds, closed. */
function blockSummary(c: Component, head: HeadDefaults): string {
  const box = effectiveBox(c, 'padding', head).map((v) => pxNumber(v)).join(' ')
  const inner = tagOf(c) === 'mj-button' ? effective(c, 'inner-padding', head).split(/\s+/).map(pxNumber).join(' ') : 'none'
  return tt(mh.advSummaryBlock, { padding: box, inner })
}

/** White, grey or the brand's tint, or an image (Composer board). */
const BACKGROUNDS = [
  { key: 'white', colour: '#FFFFFF', msg: mh.bgWhite },
  { key: 'grey', colour: '#F3F4F6', msg: mh.bgGrey },
  { key: 'tint', colour: '#F0FDFA', msg: mh.bgTint },
] as const

function BackgroundPresetProp({ c, withImage = true }: { c: Component; withImage?: boolean }) {
  const t = useT()
  const head = useHead()
  const id = useId()
  const { openDialog, setAssetTarget } = useComposer()
  const value = (effective(c, 'background-color', head) || '').toUpperCase()
  const image = withImage ? ownAttrs(c)['background-url'] : undefined
  const known = BACKGROUNDS.some((b) => b.colour === value)
  return (
    <Prop title={t(mh.background)} id={id}>
      <div className="bgr">
        {BACKGROUNDS.map((b) => (
          <button key={b.key} type="button" className="bgp" aria-pressed={!image && value === b.colour} onClick={() => {
            setAttr(c, 'background-color', b.colour)
            if (image) setAttr(c, 'background-url', '')
          }}>
            <span className="chip" style={{ background: b.colour }} aria-hidden="true" />
            {t(b.msg)}
          </button>
        ))}
        {withImage ? (
          <button
            type="button"
            className="bgp"
            aria-pressed={Boolean(image)}
            onClick={() => {
              setAssetTarget({ component: c, attr: 'background-url' })
              openDialog('assets')
            }}
          >
            {t(mh.bgImage)}
          </button>
        ) : null}
      </div>
      {!known && value && !image ? <span className="k">{t(mh.bgOther, { colour: value })}</span> : null}
    </Prop>
  )
}

function AlignmentSegProp({ c, k = 'text-align' }: { c: Component; k?: string }) {
  const t = useT()
  const head = useHead()
  const id = useId()
  const value = effective(c, k, head) || 'left'
  const items = [
    { v: 'left', icon: 'alignLeft', label: t(mh.left) },
    { v: 'center', icon: 'alignCenter', label: t(mh.center) },
    { v: 'right', icon: 'alignRight', label: t(mh.right) },
  ]
  return (
    <Prop title={t(mh.alignment)} id={id}>
      <div className="seg3">
        {items.map((it) => (
          <button key={it.v} type="button" aria-pressed={value === it.v} onClick={() => setAttr(c, k, it.v)}>
            <ToolIcon name={it.icon} size={14} strokeWidth={1.6} />
            {it.label}
          </button>
        ))}
      </div>
    </Prop>
  )
}

/** Column presets of a two-column section: 50/50, 60/40, 40/60 (Composer board). */
const SPLITS = [
  [50, 50],
  [60, 40],
  [40, 60],
] as const

function ColumnsPresetProp({ section }: { section: Component }) {
  const t = useT()
  const id = useId()
  const cols = columnsOf(section)
  if (cols.length !== 2) return null
  const widths = cols.map((col) => Math.round(parseFloat(columnWidth(col)) || 50))
  return (
    <Prop title={t(mh.columns)} id={id}>
      <div className="colp">
        {SPLITS.map(([a, b]) => (
          <button
            key={`${a}-${b}`}
            type="button"
            aria-pressed={widths[0] === a && widths[1] === b}
            onClick={() => {
              setAttr(cols[0]!, 'width', `${a}%`)
              setAttr(cols[1]!, 'width', `${b}%`)
            }}
          >
            <span className="bars" aria-hidden="true">
              <i style={{ flex: a }} />
              <i style={{ flex: b }} />
            </span>
            {a}/{b}
          </button>
        ))}
      </div>
      <span className="k">{t(mh.columnsStack)}</span>
    </Prop>
  )
}

/** The section's body text size: Small, Body or Large, in its font (Composer board). */
function TextStylePresetProp({ section }: { section: Component }) {
  const t = useT()
  const head = useHead()
  const id = useId()
  const texts = descendants(section, 'mj-text')
  if (!texts.length) return null
  const sample = bodyText(texts)
  const font = fontLabel(effective(sample, 'font-family', head) || head.tags['mj-all']?.['font-family'] || '')
  const size = Number(pxNumber(effective(sample, 'font-size', head))) || 13
  // Fixed sizes, as the board names them; another size shows as it is.
  const options = [
    { value: '14', label: t(mh.textBody, { font, size: 14 }) },
    { value: '13', label: t(mh.textSmall, { font, size: 13 }) },
    { value: '16', label: t(mh.textLarge, { font, size: 16 }) },
  ]
  if (![13, 14, 16].includes(size)) options.unshift({ value: String(size), label: t(mh.textCustom, { font, size }) })
  return (
    <Prop title={t(mh.textStyle)} id={id}>
      <SelectField
        label={t(mh.textStyle)}
        value={String(size)}
        options={options}
        onChange={(v) => {
          for (const x of texts) {
            // Headings keep their own size; body copy follows the choice.
            if (x === sample || Number(pxNumber(effective(x, 'font-size', head))) === size) setAttr(x, 'font-size', `${v}px`)
          }
        }}
      />
    </Prop>
  )
}

/** "padding 16 24 16 24 · no border · radius 0": what Advanced holds, closed. */
function advancedSummary(c: Component, head: HeadDefaults): string {
  const box = effectiveBox(c, 'padding', head).map((v) => pxNumber(v)).join(' ')
  const b = parseBorder(effective(c, 'border', head))
  const radius = pxNumber(effective(c, 'border-radius', head) || '0')
  return tt(mh.advSummary, { padding: box, border: b.style === 'none' || !b.style ? 'none' : `${pxNumber(b.width)} ${b.style}`, radius })
}

/** Advanced (MJML attributes): the attributes behind the presets, in a disclosure. */
function Advanced({ c, children, block }: { c: Component; children: ReactNode; block?: boolean }) {
  const t = useT()
  const head = useHead()
  return (
    <details className="adv">
      <summary>
        <span className="font-semibold">{t(mh.advanced)}</span>
        <span className="k">{block ? blockSummary(c, head) : advancedSummary(c, head)}</span>
      </summary>
      <div className="adv-b">{children}</div>
    </details>
  )
}

/** The campaign's language versions (internal/languages), source included. */
function useLanguageCount(campaignId: string): number {
  const q = useCampaignLanguages(campaignId)
  return (q.data?.versions ?? []).filter((v) => v.status !== 'missing').length
}

function SharedStyleCount({ campaignId }: { campaignId: string }) {
  const t = useT()
  const n = useLanguageCount(campaignId)
  return n > 1 ? <p className="k m-0 px-3 pt-2">{t(mh.sharedStyle, { count: n })}</p> : null
}

/** "Style is shared by all 3 language versions." when the campaign has language versions. */
function SharedStyleNote() {
  const c = useComposer()
  return c.kind === 'transactional' ? null : <SharedStyleCount campaignId={c.campaign.id} />
}

export function StyleFields({ c }: { c: Component }) {
  const t = useT()
  const tag = tagOf(c)
  const head = useHead()
  // The transactional editor keeps the MJML attributes (its boards predate the presets).
  const transactional = useComposer().kind === 'transactional'
  // A store's Product grid or card (X20): its settings, wherever in it the selection is.
  const store = storeSectionOf(c)
  if (store) return <StoreBlockStyle key={store.cid} section={store} />
  switch (tag) {
    case 'mj-body':
      return (
        <>
          <BackgroundProp c={c} />
          <TextProp c={c} k="width" title={t(mi.width)} px />
        </>
      )
    case 'mj-section':
    case 'mj-wrapper': {
      const texts = descendants(c, 'mj-text')
      if (transactional) {
        return (
          <>
            <BackgroundProp c={c} image />
            <PaddingProp c={c} />
            <AlignProp c={c} k="text-align" />
            {tag === 'mj-section' ? <ColumnWidthsProp section={c} /> : null}
            <BorderProp c={c} />
            <TextProp c={c} k="border-radius" title={t(mi.borderRadius)} px />
            {texts.length ? <TypographyProp targets={texts} sample={bodyText(texts)} title={t(mi.typographyInherited)} /> : null}
          </>
        )
      }
      // Writer-friendly first (CO-14, Composer board); the attributes in Advanced.
      return (
        <>
          <SharedStyleNote />
          {tag === 'mj-section' ? <FillFromStore section={c} /> : null}
          <SpacingProp c={c} />
          <BackgroundPresetProp c={c} />
          <AlignmentSegProp c={c} />
          {tag === 'mj-section' ? <ColumnsPresetProp section={c} /> : null}
          <TextStylePresetProp section={c} />
          <Advanced c={c}>
            <PaddingProp c={c} title={t(mh.advPadding, { tag })} />
            <BackgroundProp c={c} image title={t(mh.advBackground, { tag })} />
            {tag === 'mj-section' ? <ColumnWidthsProp section={c} /> : null}
            <BorderProp c={c} />
            <TextProp c={c} k="border-radius" title={t(mi.borderRadius)} px />
            <TextProp c={c} k="mj-class" title={t(mh.mjClass)} mono placeholder={t(mh.mjClassNone)} />
            {texts.length ? <TypographyProp targets={texts} sample={bodyText(texts)} title={t(mi.typographyInherited)} /> : null}
          </Advanced>
        </>
      )
    }
    case 'mj-hero':
      if (transactional) {
        return (
          <>
            <BackgroundProp c={c} image />
            <TextProp c={c} k="height" title={t(mi.height)} px />
            <PaddingProp c={c} />
            <HeroAlignProp c={c} />
          </>
        )
      }
      // Writer-friendly first (R4.U3); the attributes in Advanced.
      return (
        <>
          <BackgroundPresetProp c={c} />
          <BlockSpacingProp c={c} presets={HERO_SPACING} />
          <HeroAlignProp c={c} />
          <Advanced c={c} block>
            <PaddingProp c={c} title={t(mh.advPadding, { tag })} />
            <BackgroundProp c={c} image title={t(mh.advBackground, { tag })} />
            <TextProp c={c} k="height" title={t(mi.height)} px />
          </Advanced>
        </>
      )
    case 'mj-column':
    case 'mj-group':
      if (transactional || tag === 'mj-group') {
        return (
          <>
            <BackgroundProp c={c} />
            {tag === 'mj-column' ? <PaddingProp c={c} /> : null}
            {tag === 'mj-column' ? <BorderProp c={c} /> : null}
            {tag === 'mj-column' ? <TextProp c={c} k="border-radius" title={t(mi.borderRadius)} px /> : null}
          </>
        )
      }
      return (
        <>
          <BackgroundPresetProp c={c} withImage={false} />
          <InsideSpacingProp c={c} />
          <CornersProp c={c} />
          <Advanced c={c} block>
            <PaddingProp c={c} title={t(mh.advPadding, { tag })} />
            <BackgroundProp c={c} title={t(mh.advBackground, { tag })} />
            <BorderProp c={c} />
            <TextProp c={c} k="border-radius" title={t(mi.borderRadius)} px />
          </Advanced>
        </>
      )
    case 'mj-text':
    case 'mj-table':
      if (transactional) {
        return (
          <>
            <TypographyProp targets={[c]} sample={c} title={t(mi.typography)} />
            <Prop title={t(mi.weight)}>
              <SelectField label={t(mi.fontWeight)} value={effective(c, 'font-weight', head) || 'normal'} options={weightOptions()} onChange={(v) => setAttr(c, 'font-weight', v)} />
            </Prop>
            <AlignProp c={c} />
            <PaddingProp c={c} />
            <BackgroundProp c={c} k="container-background-color" title={t(mi.background)} />
          </>
        )
      }
      return (
        <>
          <TextSizeProp c={c} />
          <WeightProp c={c} />
          <AlignmentSegProp c={c} k="align" />
          <BlockSpacingProp c={c} />
          <Advanced c={c} block>
            <PaddingProp c={c} title={t(mh.advPadding, { tag })} />
            <TypographyProp targets={[c]} sample={c} title={t(mi.typography)} />
            <Prop title={t(mh.advWeight, { tag })}>
              <SelectField label={t(mi.fontWeight)} value={effective(c, 'font-weight', head) || 'normal'} options={weightOptions()} onChange={(v) => setAttr(c, 'font-weight', v)} />
            </Prop>
            <BackgroundProp c={c} k="container-background-color" title={t(mi.background)} />
          </Advanced>
        </>
      )
    case 'mj-button':
      if (transactional) {
        return (
          <>
            <BackgroundProp c={c} title={t(mi.buttonColour)} />
            <TypographyProp targets={[c]} sample={c} title={t(mi.typography)} weight />
            <AlignProp c={c} />
            <TextProp c={c} k="inner-padding" title={t(mi.innerPadding)} placeholder={'10px 25px'} />
            <PaddingProp c={c} title={t(mi.outerPadding)} />
            <TextProp c={c} k="border-radius" title={t(mi.borderRadius)} px />
            <TextProp c={c} k="width" title={t(mi.width)} px placeholder={'auto'} />
          </>
        )
      }
      return (
        <>
          <BackgroundProp c={c} title={t(mi.buttonColour)} />
          <ButtonSizeProp c={c} />
          <ButtonWidthProp c={c} />
          <CornersProp c={c} pill />
          <AlignmentSegProp c={c} k="align" />
          <BlockSpacingProp c={c} />
          <Advanced c={c} block>
            <TextProp c={c} k="inner-padding" title={t(mi.innerPadding)} placeholder={'10px 25px'} />
            <PaddingProp c={c} title={t(mi.outerPadding)} />
            <TextProp c={c} k="border-radius" title={t(mi.borderRadius)} px />
            <TextProp c={c} k="width" title={t(mh.advWidth, { tag })} px placeholder={'auto'} />
            <TypographyProp targets={[c]} sample={c} title={t(mi.typography)} weight />
          </Advanced>
        </>
      )
    case 'mj-image':
      if (transactional) {
        return (
          <>
            <ImageSizeProp c={c} />
            <AlignProp c={c} />
            <PaddingProp c={c} />
            <TextProp c={c} k="border-radius" title={t(mi.borderRadius)} px />
            <BackgroundProp c={c} k="container-background-color" title={t(mi.background)} />
          </>
        )
      }
      return (
        <>
          <ImageSizeProp c={c} />
          <AlignmentSegProp c={c} k="align" />
          <CornersProp c={c} />
          <BlockSpacingProp c={c} />
          <Advanced c={c} block>
            <PaddingProp c={c} title={t(mh.advPadding, { tag })} />
            <TextProp c={c} k="border-radius" title={t(mi.borderRadius)} px />
            <BackgroundProp c={c} k="container-background-color" title={t(mi.background)} />
          </Advanced>
        </>
      )
    case 'mj-divider':
      if (transactional) {
        return (
          <>
            <DividerLineProp c={c} />
            <TextProp c={c} k="width" title={t(mi.width)} />
            <AlignProp c={c} />
            <PaddingProp c={c} />
          </>
        )
      }
      return (
        <>
          <DividerWeightProp c={c} />
          <ChoiceProp
            title={t(mi.lineStyle)}
            options={[
              { key: 'solid', label: t(mi.styleSolid) },
              { key: 'dashed', label: t(mi.styleDashed) },
              { key: 'dotted', label: t(mi.styleDotted) },
            ]}
            current={effective(c, 'border-style', head) as 'solid' | 'dashed' | 'dotted'}
            onPick={(v) => setAttr(c, 'border-style', v)}
          />
          <Prop title={t(mi.lineColour)}>
            <ColorField label={t(mi.lineColour)} value={effective(c, 'border-color', head)} onCommit={(v) => setAttr(c, 'border-color', v)} />
          </Prop>
          <DividerLengthProp c={c} />
          <BlockSpacingProp c={c} />
          <Advanced c={c} block>
            <DividerLineProp c={c} title={t(mh.advLine, { tag })} />
            <TextProp c={c} k="width" title={t(mh.advWidth, { tag })} />
            <AlignProp c={c} />
            <PaddingProp c={c} title={t(mh.advPadding, { tag })} />
          </Advanced>
        </>
      )
    case 'mj-spacer':
      return (
        <>
          <TextProp c={c} k="height" title={t(mi.height)} px />
          <BackgroundProp c={c} k="container-background-color" title={t(mi.background)} />
        </>
      )
    case 'mj-social':
      return (
        <>
          <AlignProp c={c} />
          <Prop title={t(mi.layout)}>
            <div className="row2">
              <SelectField
                label={t(mi.mode)}
                value={effective(c, 'mode', head)}
                options={[
                  { value: 'horizontal', label: t(mi.horizontal) },
                  { value: 'vertical', label: t(mi.vertical) },
                ]}
                onChange={(v) => setAttr(c, 'mode', v)}
              />
              <DraftInput label={t(mi.iconSize)} value={effective(c, 'icon-size', head)} onCommit={(v) => setAttr(c, 'icon-size', withPx(v))} />
            </div>
          </Prop>
          <TypographyProp targets={[c]} sample={c} title={t(mi.typography)} />
          <PaddingProp c={c} />
        </>
      )
    case 'mj-social-element':
      return (
        <>
          <BackgroundProp c={c} title={t(mi.iconBackground)} />
          <TypographyProp targets={[c]} sample={c} title={t(mi.typography)} />
        </>
      )
    case 'mj-navbar':
      return <AlignProp c={c} />
    case 'mj-navbar-link':
      return (
        <>
          <TypographyProp targets={[c]} sample={c} title={t(mi.typography)} />
          <PaddingProp c={c} />
        </>
      )
    case 'mj-raw':
      return <RawNote />
  }
  return <p className="k p-3">{tag ? t(mi.noStyle, { tag }) : t(mi.noStyleElement)}</p>
}

function RawNote() {
  const c = useComposer()
  const t = useT()
  return (
    <div className="prop">
      <span className="k">{t(mi.rawNote)}</span>
      <button type="button" className="btn sm self-start" onClick={() => c.setView('mjml')}>
        {t(mi.openMjml)}
      </button>
    </div>
  )
}

function textLinks(c: Component): Component[] {
  const out: Component[] = []
  const walk = (x: Component) => {
    if (String(x.get('tagName') ?? '').toLowerCase() === 'a') out.push(x)
    x.components().forEach(walk)
  }
  walk(c)
  return out
}

function LinkProp({ c, title: given }: { c: Component; title?: string }) {
  const t = useT()
  const title = given ?? t(mi.linkUrl)
  const head = useHead()
  const id = useId()
  return (
    <Prop title={title} id={id}>
      <DraftInput id="insp-href" label={title} mono value={effective(c, 'href', head)} placeholder={'https://'} onCommit={(v) => setAttr(c, 'href', v.trim())} />
    </Prop>
  )
}

function ButtonTextProp({ c }: { c: Component }) {
  const tr = useT()
  const id = useId()
  const text = c.getEl()?.textContent?.trim() ?? ''
  return (
    <Prop title={tr(mi.buttonText)} id={id}>
      <DraftInput
        label={tr(mi.buttonText)}
        value={text}
        onCommit={(v) => {
          const t = v.trim()
          if (!t) return
          c.components(t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'))
        }}
      />
    </Prop>
  )
}

function ImageSourceProp({ c }: { c: Component }) {
  const { openDialog, setAssetTarget } = useComposer()
  const t = useT()
  const src = ownAttrs(c).src ?? ''
  const id = useId()
  return (
    <Prop title={t(mi.image)} id={id}>
      <div className="flex items-center gap-2">
        {src ? <img src={src} alt="" style={{ width: 44, height: 32, objectFit: 'cover', borderRadius: 4, border: '1px solid var(--line)' }} /> : null}
        <button
          type="button"
          className="btn sm"
          onClick={() => {
            setAssetTarget({ component: c, attr: 'src' })
            openDialog('assets')
          }}
        >
          {src ? t(mi.replaceImage) : t(mi.chooseImage)}
        </button>
      </div>
      <DraftInput
        label={t(mi.imageUrl)}
        mono
        value={src}
        placeholder={'https://'}
        onCommit={(v) => {
          c.addAttributes({ src: v.trim() })
          c.set('src', v.trim())
        }}
      />
    </Prop>
  )
}

function AltProp({ c }: { c: Component }) {
  const t = useT()
  const id = useId()
  const attrs = ownAttrs(c)
  const alt = attrs.alt ?? ''
  // Suggest alt text (features/ai), for people who hold ai:use.
  const ai = useAltTextAvailable()
  const ctx = useComposer()
  const altVersion = useAltVersionLabel()
  return (
    <Prop title={t(mi.altText)} id={id}>
      <DraftInput id="insp-alt" label={t(mi.altText)} value={alt} placeholder={t(mi.describeImage)} onCommit={(v) => setAttr(c, 'alt', v.trim())} />
      <span className="k">{t(mi.altHelp)}</span>
      {ai ? (
        <AltTextAssist
          component={c}
          src={attrs.src ?? ''}
          beside={<DecorativeToggle component={c} />}
          onAccept={(text) => {
            c.addAttributes({ alt: text })
            void ctx.save({ label: altVersion })
          }}
          onEdit={(text) => {
            c.addAttributes({ alt: text })
            window.setTimeout(() => document.getElementById('insp-alt')?.focus(), 30)
          }}
        />
      ) : (
        <DecorativeToggle component={c} />
      )}
    </Prop>
  )
}

function VisibilityProp({ c }: { c: Component }) {
  const ctx = useComposer()
  const id = useId()
  const t = useT()
  // Beside who sees a block (conditional content), this is about devices.
  const conditional = useConditional()?.enabled
  const toggle = (which: 'mobile' | 'desktop', on: boolean) => {
    toggleClass(c, VISIBILITY[which], on)
    // Keep the media-query rule in the head only while something uses it.
    const body = ctx.editor?.getWrapper()?.findType('mj-body')[0]
    const all: Component[] = []
    const walk = (x: Component) => {
      all.push(x)
      mjChildren(x).forEach(walk)
    }
    if (body) walk(body)
    const used = { mobile: all.some((x) => hasClass(x, VISIBILITY.mobile)), desktop: all.some((x) => hasClass(x, VISIBILITY.desktop)) }
    const next = withVisibilityStyles(ctx.head, used)
    if (next !== ctx.head) ctx.setHead(next)
  }
  return (
    <Prop title={conditional ? t(cm.onDevices) : t(mi.visibility)} id={id}>
      <label className="tg">
        <input type="checkbox" checked={hasClass(c, VISIBILITY.mobile)} onChange={(e) => toggle('mobile', e.target.checked)} />
        {t(mi.hideMobile)}
      </label>
      <label className="tg">
        <input type="checkbox" checked={hasClass(c, VISIBILITY.desktop)} onChange={(e) => toggle('desktop', e.target.checked)} />
        {t(mi.hideDesktop)}
      </label>
      <span className="k">{conditional ? t(cm.devicesNote) : t(mi.visibilityHelp)}</span>
    </Prop>
  )
}

function CssClassProp({ c }: { c: Component }) {
  const t = useT()
  const id = useId()
  const classes = (ownAttrs(c)['css-class'] ?? '')
    .split(/\s+/)
    .filter((x) => x && x !== VISIBILITY.mobile && x !== VISIBILITY.desktop)
    .join(' ')
  return (
    <Prop title={t(mi.cssClass)} id={id}>
      <DraftInput
        label={t(mi.cssClass)}
        mono
        value={classes}
        placeholder={t(mi.none)}
        onCommit={(v) => {
          const keep = [VISIBILITY.mobile, VISIBILITY.desktop].filter((x) => hasClass(c, x))
          setAttr(c, 'css-class', [...v.split(/\s+/).filter(Boolean), ...keep].join(' '))
        }}
      />
    </Prop>
  )
}

function TableRowsProp({ c }: { c: Component }) {
  const t = useT()
  const id = useId()
  const [draft, setDraft] = useState(String(c.get('content') ?? ''))
  return (
    <Prop title={t(mi.rowsHtml)} id={id}>
      <textarea
        className="in mono"
        rows={6}
        aria-label={t(mi.tableRows)}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => {
          if (draft !== c.get('content')) {
            c.set('content', draft)
            ;(c.view as unknown as { rerender?: () => void })?.rerender?.()
          }
        }}
      />
    </Prop>
  )
}

export function SettingsFields({ c }: { c: Component }) {
  const t = useT()
  const tag = tagOf(c)
  const head = useHead()
  const rows: ReactNode[] = []
  // The Product card filled from a digital product (X19).
  rows.push(<FillFromProduct key="product" c={c} />)
  // A Poll block's settings, and who sees the block (design expansion, X11);
  // the Upgrade block's (Earn, X16) and the Sponsor block's (Earn, X17); where the block shows, email or web (X18).
  rows.push(
    <UpgradeSettings key="upgrade" c={c} />,
    <SponsorSettings key="sponsor" c={c} />,
    <PollSettings key="poll" c={c} />,
    <VisibilitySection key="who" c={c} />,
    <SponsorFoot key="sponsor-foot" c={c} />,
    <ShowOnProp key="showon" c={c} />,
  )
  switch (tag) {
    case 'mj-image':
      rows.push(<ImageSourceProp key="src" c={c} />, <AltProp key="alt" c={c} />, <LinkProp key="href" c={c} title={t(mi.linkUrlOptional)} />)
      break
    case 'mj-button':
      rows.push(<ButtonTextProp key="text" c={c} />, <LinkProp key="href" c={c} />)
      break
    case 'mj-navbar-link':
    case 'mj-social-element':
      rows.push(<LinkProp key="href" c={c} />)
      break
    case 'mj-text': {
      rows.push(<TextRoleProp key="role" c={c} />)
      const links = textLinks(c)
      if (links.length)
        rows.push(
          <Prop key="links" title={t(mi.linksInText, { count: links.length })}>
            {links.map((a, i) => (
              <SubField
                key={a.cid}
                id={i === 0 ? 'insp-href' : undefined}
                label={(a.getEl()?.textContent ?? '').trim() || t(mi.linkN, { n: i + 1 })}
                mono
                value={String((a.getAttributes() as Record<string, string>).href ?? '')}
                onCommit={(v) => a.addAttributes({ href: v.trim() })}
              />
            ))}
          </Prop>,
        )
      break
    }
    case 'mj-column':
      rows.push(
        <Prop key="width" title={t(mi.width)}>
          <div className="row2">
            <DraftInput label={t(mi.desktopWidth)} value={columnWidth(c)} onCommit={(v) => setAttr(c, 'width', /^\d+(\.\d+)?$/.test(v.trim()) ? `${v.trim()}%` : v.trim())} />
            <span className="k" style={{ alignSelf: 'center' }}>
              {t(mi.stacksMobile)}
            </span>
          </div>
        </Prop>,
      )
      break
    case 'mj-section':
    case 'mj-wrapper':
      rows.push(
        <Prop key="full" title={t(mi.width)}>
          <label className="tg">
            <input type="checkbox" checked={ownAttrs(c)['full-width'] === 'full-width'} onChange={(e) => setAttr(c, 'full-width', e.target.checked ? 'full-width' : '')} />
            {t(mi.fullWidth)}
          </label>
        </Prop>,
      )
      break
    case 'mj-hero':
      rows.push(
        <Prop key="mode" title={t(mi.mode)}>
          <SelectField
            label={t(mi.mode)}
            value={effective(c, 'mode', head)}
            options={[
              { value: 'fixed-height', label: t(mi.fixedHeight) },
              { value: 'fluid-height', label: t(mi.fluidHeight) },
            ]}
            onChange={(v) => setAttr(c, 'mode', v)}
          />
        </Prop>,
      )
      break
    case 'mj-table':
      rows.push(<TableRowsProp key="rows" c={c} />)
      break
    case 'mj-raw':
      return <RawNote />
  }
  if (tag !== 'mj-body' && tag !== 'mjml') rows.push(<VisibilityProp key="vis" c={c} />, <CssClassProp key="css" c={c} />)
  rows.push(<NameInLayers key="name" c={c} />, <VersionsWithBlock key="versions" c={c} />)
  if (tag === 'mj-column' || tag === 'mj-group')
    rows.push(
      <Prop key="valign" title={t(mi.verticalAlignment)}>
        <SelectField
          label={t(mi.verticalAlignment)}
          value={effective(c, 'vertical-align', head)}
          options={[
            { value: 'top', label: t(mi.alignTop) },
            { value: 'middle', label: t(mi.alignMiddle) },
            { value: 'bottom', label: t(mi.alignBottom) },
          ]}
          onChange={(v) => setAttr(c, 'vertical-align', v)}
        />
      </Prop>,
    )
  if (tag === 'mj-body') rows.push(<p key="none" className="k p-3">{t(mi.bodyNoSettings)}</p>)
  return <>{rows}</>
}
