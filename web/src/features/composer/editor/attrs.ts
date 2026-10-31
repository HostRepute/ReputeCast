import type { Component } from 'grapesjs'
import { topLevelElements } from './mjmlDoc'
import { expandBox } from './mjmlHooks'
import { mjChildren, tagOf } from './labels'
import { t } from '@/i18n'
import { mi } from '../inspector.messages'

/**
 * Reading and writing MJML attributes for the inspector. A value shown in the
 * Style and Settings panels is what the compiled email uses: the element's
 * own attribute, else its mj-class, else the head's mj-attributes for the tag
 * or mj-all, else MJML's default.
 */

type Attrs = Record<string, string>

/** MJML 4 defaults for the attributes the inspector shows. */
export const MJML_DEFAULTS: Record<string, Attrs> = {
  'mj-body': { width: '600px', 'background-color': '' },
  'mj-section': { 'background-color': '', padding: '20px 0px', 'text-align': 'center', border: 'none', 'border-radius': '0px', direction: 'ltr' },
  'mj-wrapper': { 'background-color': '', padding: '20px 0px', 'text-align': 'center', border: 'none', 'border-radius': '0px' },
  'mj-column': { 'background-color': '', padding: '0px', 'vertical-align': 'top', border: 'none', 'border-radius': '0px' },
  'mj-group': { 'background-color': '', 'vertical-align': 'top', direction: 'ltr' },
  'mj-hero': { 'background-color': '#ffffff', mode: 'fixed-height', height: '0px', padding: '0px', 'vertical-align': 'top', 'background-url': '' },
  'mj-text': {
    color: '#000000',
    'font-family': 'Ubuntu, Helvetica, Arial, sans-serif',
    'font-size': '13px',
    'line-height': '1',
    'font-weight': '',
    'letter-spacing': 'none',
    align: 'left',
    padding: '10px 25px',
    'container-background-color': '',
  },
  'mj-button': {
    'background-color': '#414141',
    color: '#ffffff',
    'font-family': 'Ubuntu, Helvetica, Arial, sans-serif',
    'font-size': '13px',
    'font-weight': 'normal',
    'line-height': '120%',
    'border-radius': '3px',
    border: 'none',
    'inner-padding': '10px 25px',
    padding: '10px 25px',
    align: 'center',
    width: '',
    href: '',
  },
  'mj-image': { align: 'center', padding: '10px 25px', border: '0', 'border-radius': '', width: '', height: 'auto', alt: '', href: '', src: '' },
  'mj-divider': { 'border-color': '#000000', 'border-style': 'solid', 'border-width': '4px', width: '100%', padding: '10px 25px', align: 'center' },
  'mj-spacer': { height: '20px', 'container-background-color': '' },
  'mj-social': { align: 'center', mode: 'horizontal', 'icon-size': '20px', 'font-size': '13px', color: '#333333', padding: '10px 25px', 'border-radius': '3px' },
  'mj-social-element': { 'background-color': '', color: '#000', 'font-size': '13px', href: '', name: '' },
  'mj-navbar': { align: 'center', hamburger: '' },
  'mj-navbar-link': { color: '#000000', 'font-family': 'Ubuntu, Helvetica, Arial, sans-serif', 'font-size': '13px', padding: '15px 10px', 'text-transform': 'uppercase', href: '' },
  'mj-table': { color: '#000000', 'font-family': 'Ubuntu, Helvetica, Arial, sans-serif', 'font-size': '13px', 'line-height': '22px', align: 'left', width: '100%', padding: '10px 25px' },
}

const attrRe = /([\w:-]+)\s*=\s*("([^"]*)"|'([^']*)')/g

function parseAttrs(open: string): Attrs {
  const out: Attrs = {}
  for (const m of open.matchAll(attrRe)) out[m[1]!] = decode(m[3] ?? m[4] ?? '')
  return out
}

function decode(s: string): string {
  return s.replace(/&quot;/g, '"').replace(/&#39;/g, "'").replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&')
}

export interface HeadDefaults {
  /** Attributes per tag from <mj-attributes>, mj-all included. */
  tags: Record<string, Attrs>
  /** <mj-class name="…"> definitions. */
  classes: Record<string, Attrs>
}

/** Reads <mj-attributes> from a head (inner MJML of mj-head). */
export function headDefaults(head: string): HeadDefaults {
  const out: HeadDefaults = { tags: {}, classes: {} }
  for (const el of topLevelElements(head)) {
    if (el.tag !== 'mj-attributes') continue
    for (const child of topLevelElements(el.inner)) {
      const src = el.inner.slice(child.start, child.end)
      const open = src.slice(0, src.indexOf('>') + 1)
      const attrs = parseAttrs(open)
      if (child.tag === 'mj-class') {
        const name = attrs.name
        if (name) {
          delete attrs.name
          out.classes[name] = { ...out.classes[name], ...attrs }
        }
      } else out.tags[child.tag] = { ...out.tags[child.tag], ...attrs }
    }
  }
  return out
}

/** The component's own attributes (never the plugin's defaults). */
export function ownAttrs(c: Component): Attrs {
  const a = (c.getAttributes() ?? {}) as Record<string, unknown>
  const out: Attrs = {}
  for (const [k, v] of Object.entries(a)) if (v !== undefined && v !== null && k !== 'id' && k !== 'style') out[k] = String(v)
  return out
}

const SIDES = ['top', 'right', 'bottom', 'left'] as const

function sideOf(attrs: Attrs, key: string, side: (typeof SIDES)[number]): string | undefined {
  const long = attrs[`${key}-${side}`]
  if (long) return long
  const short = attrs[key]
  if (short) return expandBox(short)[side]
  return undefined
}

/** The value the compiled email uses for `key` on `c`. */
export function effective(c: Component, key: string, head: HeadDefaults): string {
  const tag = tagOf(c)
  const own = ownAttrs(c)
  if (own[key] !== undefined && own[key] !== '') return own[key]!
  for (const cls of (own['mj-class'] ?? '').split(/\s+/).filter(Boolean)) {
    const v = head.classes[cls]?.[key]
    if (v !== undefined) return v
  }
  const v = head.tags[tag]?.[key] ?? head.tags['mj-all']?.[key]
  if (v !== undefined) return v
  return MJML_DEFAULTS[tag]?.[key] ?? ''
}

/** Effective four sides of a box attribute (padding), longhands first. */
export function effectiveBox(c: Component, key: string, head: HeadDefaults): [string, string, string, string] {
  const tag = tagOf(c)
  const own = ownAttrs(c)
  const layers: Attrs[] = [own]
  for (const cls of (own['mj-class'] ?? '').split(/\s+/).filter(Boolean)) if (head.classes[cls]) layers.push(head.classes[cls]!)
  if (head.tags[tag]) layers.push(head.tags[tag]!)
  if (head.tags['mj-all']) layers.push(head.tags['mj-all']!)
  layers.push(MJML_DEFAULTS[tag] ?? {})
  return SIDES.map((s) => {
    for (const l of layers) {
      const v = sideOf(l, key, s)
      if (v) return v
    }
    return '0px'
  }) as [string, string, string, string]
}

/** Sets (or, for an empty value, removes) an attribute; the canvas re-renders. */
export function setAttr(c: Component, key: string, value: string) {
  if (value === '') c.removeAttributes([key])
  else c.addAttributes({ [key]: value })
}

/**
 * Sets all four sides of a box attribute as one shorthand ("16px 24px"),
 * taking out longhands an earlier edit wrote (they'd win over it).
 */
export function setBox(c: Component, key: string, value: string) {
  c.removeAttributes(SIDES.map((s) => `${key}-${s}`))
  setAttr(c, key, value)
}

/** Sets one side of a box attribute, writing all four sides as longhands. */
export function setBoxSide(c: Component, key: string, side: (typeof SIDES)[number], value: string, head: HeadDefaults) {
  const cur = effectiveBox(c, key, head)
  const next: Attrs = {}
  SIDES.forEach((s, i) => (next[`${key}-${s}`] = s === side ? value : cur[i]!))
  c.removeAttributes([key])
  c.addAttributes(next)
}

/** "16px" → "16"; other units are kept ("1em"). */
export function pxNumber(v: string): string {
  const m = /^(-?\d+(?:\.\d+)?)px$/.exec(v.trim())
  return m ? m[1]! : v.trim()
}

/** "16" → "16px"; values with a unit or empty are kept. */
export function withPx(v: string): string {
  const t = v.trim()
  return /^-?\d+(\.\d+)?$/.test(t) ? `${t}px` : t
}

/** Font stacks offered in the inspector (all email safe). */
export const FONTS: { label: string; value: string }[] = [
  { label: 'Georgia', value: "Georgia, 'Times New Roman', serif" },
  { label: 'Helvetica', value: 'Helvetica, Arial, sans-serif' },
  { label: 'Arial', value: 'Arial, Helvetica, sans-serif' },
  { label: 'Verdana', value: 'Verdana, Geneva, sans-serif' },
  { label: 'Trebuchet MS', value: "'Trebuchet MS', Helvetica, sans-serif" },
  { label: 'Times New Roman', value: "'Times New Roman', Times, serif" },
  { label: 'Courier New', value: "'Courier New', Courier, monospace" },
]

/** First family of a font stack, unquoted: "'IBM Plex Sans', Arial" → "IBM Plex Sans". */
export function fontLabel(stack: string): string {
  return (stack.split(',')[0] ?? '').trim().replace(/^['"]|['"]$/g, '') || t(mi.default)
}

/** Normalises a colour for the colour input: #abc → #aabbcc; other formats → "". */
export function hexColor(v: string): string {
  const t = v.trim().toLowerCase()
  if (/^#[0-9a-f]{6}$/.test(t)) return t
  const m = /^#([0-9a-f])([0-9a-f])([0-9a-f])$/.exec(t)
  if (m) return `#${m[1]}${m[1]}${m[2]}${m[2]}${m[3]}${m[3]}`
  if (t === 'white') return '#ffffff'
  if (t === 'black') return '#000000'
  return ''
}

/** Parses an MJML border shorthand: "1px solid #E3E6EB". */
export function parseBorder(v: string): { style: 'none' | 'solid' | 'dashed' | 'dotted'; width: string; color: string } {
  const t = v.trim()
  if (!t || t === 'none' || t === '0' || t === '0px') return { style: 'none', width: '0px', color: '' }
  const parts = t.split(/\s+/)
  const style = (parts.find((p) => ['solid', 'dashed', 'dotted', 'none'].includes(p)) as 'solid' | 'dashed' | 'dotted' | 'none' | undefined) ?? 'solid'
  const width = parts.find((p) => /^\d/.test(p)) ?? '1px'
  const color = parts.find((p) => p.startsWith('#') || /^rgb/.test(p) || /^[a-z]+$/i.test(p) && !['solid', 'dashed', 'dotted', 'none'].includes(p)) ?? ''
  return { style, width, color }
}

export function formatBorder(b: { style: string; width: string; color: string }): string {
  if (b.style === 'none') return ''
  return [b.width && b.width !== '0px' ? b.width : '1px', b.style, b.color || '#E3E6EB'].join(' ')
}

export function descendants(c: Component, tag: string): Component[] {
  const out: Component[] = []
  const walk = (x: Component) => {
    for (const k of mjChildren(x)) {
      if (tagOf(k) === tag) out.push(k)
      walk(k)
    }
  }
  walk(c)
  return out
}

/** Columns (and groups) of a section, in order. */
export function columnsOf(section: Component): Component[] {
  return mjChildren(section).filter((x) => tagOf(x) === 'mj-column' || tagOf(x) === 'mj-group')
}

/** A column's width as MJML computes it when none is set. */
export function columnWidth(col: Component): string {
  const own = ownAttrs(col).width
  if (own) return own
  const parent = col.parent()
  const n = parent ? columnsOf(parent).length : 1
  return `${Math.round((100 / Math.max(1, n)) * 100) / 100}%`
}

export const VISIBILITY = { mobile: 'rc-hide-mobile', desktop: 'rc-hide-desktop' } as const

export function hasClass(c: Component, cls: string): boolean {
  return (ownAttrs(c)['css-class'] ?? '').split(/\s+/).includes(cls)
}

export function toggleClass(c: Component, cls: string, on: boolean) {
  const list = (ownAttrs(c)['css-class'] ?? '').split(/\s+/).filter(Boolean)
  const next = on ? Array.from(new Set([...list, cls])) : list.filter((x) => x !== cls)
  setAttr(c, 'css-class', next.join(' '))
}
