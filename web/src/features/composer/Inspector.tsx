import type { Component } from 'grapesjs'
import { useEffect, useId, useMemo, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { useComposer } from './context'
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
  setBoxSide,
  toggleClass,
  VISIBILITY,
  withPx,
  type HeadDefaults,
} from './editor/attrs'
import { mjChildren, sectionLabel, sectionOf, tagOf } from './editor/labels'
import { withVisibilityStyles } from './editor/mjmlDoc'

/* ------------------------------------------------------------ inputs */

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

/** Colour swatch (native picker) + hex input. */
function ColorField({ value, onCommit, label, children, gap = 8 }: { value: string; onCommit: (v: string) => void; label: string; children?: ReactNode; gap?: number }) {
  const timer = useRef<number | undefined>(undefined)
  const hex = hexColor(value)
  return (
    <div className="flex items-center" style={{ gap }}>
      <input
        type="color"
        className="sw"
        aria-label={`${label} colour picker`}
        value={hex || '#ffffff'}
        onChange={(e) => {
          const v = e.target.value.toUpperCase()
          window.clearTimeout(timer.current)
          timer.current = window.setTimeout(() => onCommit(v), 250)
        }}
        style={{ background: value || 'transparent' }}
      />
      <DraftInput label={label} mono value={/^#[0-9a-f]{3,8}$/i.test(value) ? value.toUpperCase() : value} placeholder="none" onCommit={(v) => onCommit(v.trim())} style={{ flex: 1 }} />
      {children}
    </div>
  )
}

const SIDE_LABELS = [
  ['top', 'Top'],
  ['right', 'Right'],
  ['bottom', 'Bottom'],
  ['left', 'Left'],
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
  const box = effectiveBox(c, k, head)
  return (
    <div className="row4">
      {SIDE_LABELS.map(([side, label], i) => (
        <SubField key={side} label={label} value={pxNumber(box[i]!)} onCommit={(v) => setBoxSide(c, k, side, withPx(v || '0'), head)} />
      ))}
    </div>
  )
}

function AlignField({ value, onChange, label, options = ['left', 'center', 'right'] }: { value: string; onChange: (v: string) => void; label: string; options?: string[] }) {
  return (
    <div className="seg" role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o} type="button" aria-pressed={value === o} onClick={() => onChange(o)}>
          {o[0]!.toUpperCase() + o.slice(1)}
        </button>
      ))}
    </div>
  )
}

function SelectField({ value, onChange, label, options }: { value: string; onChange: (v: string) => void; label: string; options: { value: string; label: string }[] }) {
  const has = options.some((o) => o.value === value)
  return (
    <select className="in" aria-label={label} value={value} onChange={(e) => onChange(e.target.value)}>
      {has ? null : <option value={value}>{value || 'Default'}</option>}
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  )
}

/* ------------------------------------------------------------ style groups */

function useHead() {
  const { head } = useComposer()
  return useMemo(() => headDefaults(head), [head])
}

function BackgroundProp({ c, k = 'background-color', image, title = 'Background' }: { c: Component; k?: string; image?: boolean; title?: string }) {
  const head = useHead()
  const { openDialog, setAssetTarget } = useComposer()
  const id = useId()
  return (
    <Prop title={title} id={id}>
      <ColorField label={title} value={effective(c, k, head)} onCommit={(v) => setAttr(c, k, v)}>
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
            Image
          </button>
        ) : null}
      </ColorField>
      {image && ownAttrs(c)['background-url'] ? (
        <span className="k flex items-center gap-2">
          <span className="mono min-w-0 flex-1 truncate">{ownAttrs(c)['background-url']}</span>
          <button type="button" className="btn sm" onClick={() => setAttr(c, 'background-url', '')}>
            Remove
          </button>
        </span>
      ) : null}
    </Prop>
  )
}

function PaddingProp({ c, k = 'padding', title = 'Padding' }: { c: Component; k?: string; title?: string }) {
  const head = useHead()
  const id = useId()
  return (
    <Prop title={title} id={id}>
      <BoxField c={c} k={k} head={head} />
    </Prop>
  )
}

function AlignProp({ c, k = 'align' }: { c: Component; k?: string }) {
  const head = useHead()
  const id = useId()
  return (
    <Prop title="Alignment" id={id}>
      <AlignField label="Alignment" value={effective(c, k, head)} onChange={(v) => setAttr(c, k, v)} />
    </Prop>
  )
}

function BorderProp({ c }: { c: Component }) {
  const head = useHead()
  const id = useId()
  const b = parseBorder(effective(c, 'border', head))
  return (
    <Prop title="Border" id={id}>
      <div className="row2">
        <SelectField
          label="Border style"
          value={b.style}
          options={[
            { value: 'none', label: 'None' },
            { value: 'solid', label: 'Solid' },
            { value: 'dashed', label: 'Dashed' },
            { value: 'dotted', label: 'Dotted' },
          ]}
          onChange={(style) => setAttr(c, 'border', formatBorder({ ...b, style, width: b.width === '0px' ? '1px' : b.width }))}
        />
        <DraftInput label="Border width" value={b.width} onCommit={(w) => setAttr(c, 'border', formatBorder({ ...b, style: b.style === 'none' ? 'solid' : b.style, width: withPx(w) }))} />
      </div>
      {b.style !== 'none' ? <ColorField label="Border colour" value={b.color} onCommit={(color) => setAttr(c, 'border', formatBorder({ ...b, color }))} /> : null}
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
  const cols = columnsOf(section)
  const id = useId()
  if (cols.length < 2) return null
  return (
    <Prop title="Column widths" id={id}>
      <div className={cols.length === 2 ? 'row2' : cols.length === 3 ? 'grid grid-cols-3 gap-1.5' : 'row4'}>
        {cols.map((col, i) => (
          <SubField key={col.cid} label={`Column ${i + 1}`} value={columnWidth(col)} onCommit={(v) => setAttr(col, 'width', /^\d+(\.\d+)?$/.test(v.trim()) ? `${v.trim()}%` : v.trim())} />
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
  const head = useHead()
  const id = useId()
  const set = (k: string, v: string) => targets.forEach((t) => setAttr(t, k, v))
  const family = effective(sample, 'font-family', head)
  const color = effective(sample, 'color', head)
  return (
    <Prop title={title} id={id}>
      <div className="row2">
        <SelectField label={title} value={family} options={fontOptions(family)} onChange={(v) => set('font-family', v)} />
        <DraftInput label="Font size" value={effective(sample, 'font-size', head)} onCommit={(v) => set('font-size', withPx(v))} />
      </div>
      <div className="row2">
        {/* In a two-column row the board spaces swatch and hex 6px apart. */}
        <ColorField label="Text colour" value={color} onCommit={(v) => set('color', v)} gap={6} />
        {weight ? (
          <SelectField
            label="Font weight"
            value={effective(sample, 'font-weight', head) || 'normal'}
            options={[
              { value: 'normal', label: 'Regular' },
              { value: '500', label: 'Medium' },
              { value: '600', label: 'Semibold' },
              { value: '700', label: 'Bold' },
            ]}
            onChange={(v) => set('font-weight', v)}
          />
        ) : (
          <DraftInput label="Line height" value={effective(sample, 'line-height', head)} onCommit={(v) => set('line-height', v.trim())} />
        )}
      </div>
    </Prop>
  )
}

/* ------------------------------------------------------------ describe */

const ORDINALS = ['first', 'second', 'third', 'fourth', 'fifth', 'sixth']

function sectionPhrase(sec: Component | undefined): string {
  if (!sec) return 'email'
  const l = sectionLabel(sec)
  const m = /^(\d+) columns$/.exec(l)
  if (m) return `${m[1]}-column section`
  return l.endsWith('section') ? l : `${l} section`
}

/** The subtitle under the tag name in the inspector. */
export function describe(c: Component, tab: 'style' | 'settings'): string {
  const tag = tagOf(c)
  const sec = sectionOf(c)
  switch (tag) {
    case 'mj-body':
      return 'the email body · width and background'
    case 'mj-section':
    case 'mj-wrapper':
    case 'mj-hero': {
      const l = sec === c ? sectionLabel(c) : tag.replace('mj-', '')
      return tab === 'style' ? `${l} · only properties MJML supports` : `${l} · settings MJML supports`
    }
    case 'mj-column': {
      const parent = c.parent()
      const i = parent ? columnsOf(parent).indexOf(c) : 0
      return `${ORDINALS[i] ?? `#${i + 1}`} column of the ${sectionPhrase(sec)}`
    }
    case 'mj-raw':
      return 'raw HTML, edited in the code view'
  }
  return `${tag.replace('mj-', '')} in the ${sectionPhrase(sec)}`
}

/* ------------------------------------------------------------ style tab */

export function StyleFields({ c }: { c: Component }) {
  const tag = tagOf(c)
  const head = useHead()
  switch (tag) {
    case 'mj-body':
      return (
        <>
          <BackgroundProp c={c} />
          <TextProp c={c} k="width" title="Width" px />
        </>
      )
    case 'mj-section':
    case 'mj-wrapper': {
      const texts = descendants(c, 'mj-text')
      return (
        <>
          <BackgroundProp c={c} image />
          <PaddingProp c={c} />
          <AlignProp c={c} k="text-align" />
          {tag === 'mj-section' ? <ColumnWidthsProp section={c} /> : null}
          <BorderProp c={c} />
          <TextProp c={c} k="border-radius" title="Border radius" px />
          {texts.length ? <TypographyProp targets={texts} sample={bodyText(texts)} title="Typography · inherited by children" /> : null}
        </>
      )
    }
    case 'mj-hero':
      return (
        <>
          <BackgroundProp c={c} image />
          <TextProp c={c} k="height" title="Height" px />
          <PaddingProp c={c} />
          <Prop title="Vertical alignment">
            <AlignField label="Vertical alignment" options={['top', 'middle', 'bottom']} value={effective(c, 'vertical-align', head)} onChange={(v) => setAttr(c, 'vertical-align', v)} />
          </Prop>
        </>
      )
    case 'mj-column':
    case 'mj-group':
      return (
        <>
          <BackgroundProp c={c} />
          {tag === 'mj-column' ? <PaddingProp c={c} /> : null}
          {tag === 'mj-column' ? <BorderProp c={c} /> : null}
          {tag === 'mj-column' ? <TextProp c={c} k="border-radius" title="Border radius" px /> : null}
        </>
      )
    case 'mj-text':
    case 'mj-table':
      return (
        <>
          <TypographyProp targets={[c]} sample={c} title="Typography" />
          <Prop title="Weight">
            <SelectField
              label="Font weight"
              value={effective(c, 'font-weight', head) || 'normal'}
              options={[
                { value: 'normal', label: 'Regular' },
                { value: '500', label: 'Medium' },
                { value: '600', label: 'Semibold' },
                { value: '700', label: 'Bold' },
              ]}
              onChange={(v) => setAttr(c, 'font-weight', v)}
            />
          </Prop>
          <AlignProp c={c} />
          <PaddingProp c={c} />
          <BackgroundProp c={c} k="container-background-color" title="Background" />
        </>
      )
    case 'mj-button':
      return (
        <>
          <BackgroundProp c={c} title="Button colour" />
          <TypographyProp targets={[c]} sample={c} title="Typography" weight />
          <AlignProp c={c} />
          <TextProp c={c} k="inner-padding" title="Inner padding" placeholder="10px 25px" />
          <PaddingProp c={c} title="Outer padding" />
          <TextProp c={c} k="border-radius" title="Border radius" px />
          <TextProp c={c} k="width" title="Width" px placeholder="auto" />
        </>
      )
    case 'mj-image':
      return (
        <>
          <Prop title="Size">
            <div className="row2">
              <SubField label="Width" value={effective(c, 'width', head)} placeholder="auto" onCommit={(v) => setAttr(c, 'width', withPx(v))} />
              <SubField label="Height" value={effective(c, 'height', head)} placeholder="auto" onCommit={(v) => setAttr(c, 'height', withPx(v))} />
            </div>
          </Prop>
          <AlignProp c={c} />
          <PaddingProp c={c} />
          <TextProp c={c} k="border-radius" title="Border radius" px />
          <BackgroundProp c={c} k="container-background-color" title="Background" />
        </>
      )
    case 'mj-divider':
      return (
        <>
          <Prop title="Line">
            <div className="row2">
              <SelectField
                label="Line style"
                value={effective(c, 'border-style', head)}
                options={[
                  { value: 'solid', label: 'Solid' },
                  { value: 'dashed', label: 'Dashed' },
                  { value: 'dotted', label: 'Dotted' },
                ]}
                onChange={(v) => setAttr(c, 'border-style', v)}
              />
              <DraftInput label="Line width" value={effective(c, 'border-width', head)} onCommit={(v) => setAttr(c, 'border-width', withPx(v))} />
            </div>
            <ColorField label="Line colour" value={effective(c, 'border-color', head)} onCommit={(v) => setAttr(c, 'border-color', v)} />
          </Prop>
          <TextProp c={c} k="width" title="Width" />
          <AlignProp c={c} />
          <PaddingProp c={c} />
        </>
      )
    case 'mj-spacer':
      return (
        <>
          <TextProp c={c} k="height" title="Height" px />
          <BackgroundProp c={c} k="container-background-color" title="Background" />
        </>
      )
    case 'mj-social':
      return (
        <>
          <AlignProp c={c} />
          <Prop title="Layout">
            <div className="row2">
              <SelectField
                label="Mode"
                value={effective(c, 'mode', head)}
                options={[
                  { value: 'horizontal', label: 'Horizontal' },
                  { value: 'vertical', label: 'Vertical' },
                ]}
                onChange={(v) => setAttr(c, 'mode', v)}
              />
              <DraftInput label="Icon size" value={effective(c, 'icon-size', head)} onCommit={(v) => setAttr(c, 'icon-size', withPx(v))} />
            </div>
          </Prop>
          <TypographyProp targets={[c]} sample={c} title="Typography" />
          <PaddingProp c={c} />
        </>
      )
    case 'mj-social-element':
      return (
        <>
          <BackgroundProp c={c} title="Icon background" />
          <TypographyProp targets={[c]} sample={c} title="Typography" />
        </>
      )
    case 'mj-navbar':
      return <AlignProp c={c} />
    case 'mj-navbar-link':
      return (
        <>
          <TypographyProp targets={[c]} sample={c} title="Typography" />
          <PaddingProp c={c} />
        </>
      )
    case 'mj-raw':
      return <RawNote />
  }
  return <p className="k p-3">No style properties for {tag || 'this element'}.</p>
}

function RawNote() {
  const c = useComposer()
  return (
    <div className="prop">
      <span className="k">Raw HTML is passed through MJML untouched, so it has no MJML properties. Edit it in the code view.</span>
      <button type="button" className="btn sm self-start" onClick={() => c.setView('mjml')}>
        Open the MJML view
      </button>
    </div>
  )
}

/* ------------------------------------------------------------ settings tab */

function textLinks(c: Component): Component[] {
  const out: Component[] = []
  const walk = (x: Component) => {
    if (String(x.get('tagName') ?? '').toLowerCase() === 'a') out.push(x)
    x.components().forEach(walk)
  }
  walk(c)
  return out
}

function LinkProp({ c, title = 'Link URL' }: { c: Component; title?: string }) {
  const head = useHead()
  const id = useId()
  return (
    <Prop title={title} id={id}>
      <DraftInput id="insp-href" label={title} mono value={effective(c, 'href', head)} placeholder="https://" onCommit={(v) => setAttr(c, 'href', v.trim())} />
    </Prop>
  )
}

function ButtonTextProp({ c }: { c: Component }) {
  const id = useId()
  const text = c.getEl()?.textContent?.trim() ?? ''
  return (
    <Prop title="Button text" id={id}>
      <DraftInput
        label="Button text"
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
  const src = ownAttrs(c).src ?? ''
  const id = useId()
  return (
    <Prop title="Image" id={id}>
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
          {src ? 'Replace image' : 'Choose image'}
        </button>
      </div>
      <DraftInput
        label="Image URL"
        mono
        value={src}
        placeholder="https://"
        onCommit={(v) => {
          c.addAttributes({ src: v.trim() })
          c.set('src', v.trim())
        }}
      />
    </Prop>
  )
}

function AltProp({ c }: { c: Component }) {
  const id = useId()
  const alt = ownAttrs(c).alt ?? ''
  return (
    <Prop title="Alt text" id={id}>
      <DraftInput id="insp-alt" label="Alt text" value={alt} placeholder="Describe the image" onCommit={(v) => setAttr(c, 'alt', v.trim())} />
      <span className="k">Shown when images are off (Outlook, many corporate clients). Leave it empty only for decorative images.</span>
    </Prop>
  )
}

function VisibilityProp({ c }: { c: Component }) {
  const ctx = useComposer()
  const id = useId()
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
    <Prop title="Visibility" id={id}>
      <label className="tg">
        <input type="checkbox" checked={hasClass(c, VISIBILITY.mobile)} onChange={(e) => toggle('mobile', e.target.checked)} />
        Hide on mobile
      </label>
      <label className="tg">
        <input type="checkbox" checked={hasClass(c, VISIBILITY.desktop)} onChange={(e) => toggle('desktop', e.target.checked)} />
        Hide on desktop
      </label>
      <span className="k">Adds a CSS class and its media-query rule to the head; the layer tree marks hidden components.</span>
    </Prop>
  )
}

function CssClassProp({ c }: { c: Component }) {
  const id = useId()
  const classes = (ownAttrs(c)['css-class'] ?? '')
    .split(/\s+/)
    .filter((x) => x && x !== VISIBILITY.mobile && x !== VISIBILITY.desktop)
    .join(' ')
  return (
    <Prop title="CSS class" id={id}>
      <DraftInput
        label="CSS class"
        mono
        value={classes}
        placeholder="none"
        onCommit={(v) => {
          const keep = [VISIBILITY.mobile, VISIBILITY.desktop].filter((x) => hasClass(c, x))
          setAttr(c, 'css-class', [...v.split(/\s+/).filter(Boolean), ...keep].join(' '))
        }}
      />
    </Prop>
  )
}

function TableRowsProp({ c }: { c: Component }) {
  const id = useId()
  const [draft, setDraft] = useState(String(c.get('content') ?? ''))
  return (
    <Prop title="Rows (HTML)" id={id}>
      <textarea
        className="in mono"
        rows={6}
        aria-label="Table rows"
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
  const tag = tagOf(c)
  const head = useHead()
  const rows: ReactNode[] = []
  switch (tag) {
    case 'mj-image':
      rows.push(<ImageSourceProp key="src" c={c} />, <AltProp key="alt" c={c} />, <LinkProp key="href" c={c} title="Link URL (optional)" />)
      break
    case 'mj-button':
      rows.push(<ButtonTextProp key="text" c={c} />, <LinkProp key="href" c={c} />)
      break
    case 'mj-navbar-link':
    case 'mj-social-element':
      rows.push(<LinkProp key="href" c={c} />)
      break
    case 'mj-text': {
      const links = textLinks(c)
      if (links.length)
        rows.push(
          <Prop key="links" title={`Links in this text · ${links.length}`}>
            {links.map((a, i) => (
              <SubField
                key={a.cid}
                id={i === 0 ? 'insp-href' : undefined}
                label={(a.getEl()?.textContent ?? '').trim() || `Link ${i + 1}`}
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
        <Prop key="width" title="Width">
          <div className="row2">
            <DraftInput label="Desktop width" value={columnWidth(c)} onCommit={(v) => setAttr(c, 'width', /^\d+(\.\d+)?$/.test(v.trim()) ? `${v.trim()}%` : v.trim())} />
            <span className="k" style={{ alignSelf: 'center' }}>
              stacks to 100% on mobile
            </span>
          </div>
        </Prop>,
      )
      break
    case 'mj-section':
    case 'mj-wrapper':
      rows.push(
        <Prop key="full" title="Width">
          <label className="tg">
            <input type="checkbox" checked={ownAttrs(c)['full-width'] === 'full-width'} onChange={(e) => setAttr(c, 'full-width', e.target.checked ? 'full-width' : '')} />
            Full width background
          </label>
        </Prop>,
      )
      break
    case 'mj-hero':
      rows.push(
        <Prop key="mode" title="Mode">
          <SelectField
            label="Mode"
            value={effective(c, 'mode', head)}
            options={[
              { value: 'fixed-height', label: 'Fixed height' },
              { value: 'fluid-height', label: 'Fluid height' },
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
  if (tag === 'mj-column' || tag === 'mj-group')
    rows.push(
      <Prop key="valign" title="Vertical alignment">
        <SelectField
          label="Vertical alignment"
          value={effective(c, 'vertical-align', head)}
          options={[
            { value: 'top', label: 'Top' },
            { value: 'middle', label: 'Middle' },
            { value: 'bottom', label: 'Bottom' },
          ]}
          onChange={(v) => setAttr(c, 'vertical-align', v)}
        />
      </Prop>,
    )
  if (tag === 'mj-body') rows.push(<p key="none" className="k p-3">The body has no settings beyond its style. Select a section or an element on the canvas.</p>)
  return <>{rows}</>
}
