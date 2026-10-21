/**
 * The page builder's model, without React or GrapesJS. A page is the HTML
 * of its blocks (internal/pages): top-level <section data-pb="…"> elements
 * on the public pages' classes, and custom elements (<rc-product>,
 * <rc-countdown>…) the site draws with live data. This module knows what
 * each block is, what the layers call it, which attributes a live block
 * keeps, the theme's colours and spacing as classes, and the dates of
 * countdowns and events.
 */

export const LIVE_TAGS = ['rc-form', 'rc-countdown', 'rc-event', 'rc-product', 'rc-products', 'rc-store-product', 'rc-tiers', 'rc-posts', 'rc-video'] as const
export type LiveTag = (typeof LIVE_TAGS)[number]

export type BlockKind =
  | 'section'
  | 'columns'
  | 'divider'
  | 'spacer'
  | 'hero'
  | 'text'
  | 'image'
  | 'button'
  | 'video'
  | 'features'
  | 'testimonials'
  | 'faq'
  | 'signup'
  | 'countdown'
  | 'event'
  | 'product'
  | 'products'
  | 'store'
  | 'tiers'
  | 'posts'
  | 'bio'

export const BLOCK_KINDS: readonly BlockKind[] = [
  'section',
  'columns',
  'divider',
  'spacer',
  'hero',
  'text',
  'image',
  'button',
  'video',
  'features',
  'testimonials',
  'faq',
  'signup',
  'countdown',
  'event',
  'product',
  'products',
  'store',
  'tiers',
  'posts',
  'bio',
]

export const LIVE_KIND: Record<LiveTag, BlockKind> = {
  'rc-form': 'signup',
  'rc-countdown': 'countdown',
  'rc-event': 'event',
  'rc-product': 'product',
  'rc-products': 'products',
  'rc-store-product': 'store',
  'rc-tiers': 'tiers',
  'rc-posts': 'posts',
  'rc-video': 'video',
}

export const KIND_TAG: Partial<Record<BlockKind, LiveTag>> = Object.fromEntries(Object.entries(LIVE_KIND).map(([tag, kind]) => [kind, tag])) as Partial<Record<BlockKind, LiveTag>>

/** The attributes each live block keeps (internal/pages/blocks.go); anything else is dropped when the page is saved. */
export const LIVE_ATTRS: Record<LiveTag, readonly string[]> = {
  'rc-form': ['form', 'heading', 'hint', 'bg'],
  'rc-countdown': ['at', 'tz', 'units', 'end', 'message', 'reveal', 'hide', 'label'],
  'rc-event': ['date', 'start', 'end', 'tz', 'local', 'where', 'platform', 'venue', 'link', 'calendar', 'title', 'reminders', 'field'],
  'rc-product': ['product', 'button', 'price', 'offer', 'layout', 'eyebrow', 'bg'],
  'rc-products': ['source', 'products', 'count', 'button'],
  'rc-store-product': ['store', 'product', 'button'],
  'rc-tiers': ['tiers', 'period'],
  'rc-posts': ['tag', 'count', 'heading'],
  'rc-video': ['url', 'title'],
}

/** Every live block also keeps its id, its name in the layers, an anchor id and classes. */
export const COMMON_LIVE_ATTRS = ['data-pb-id', 'data-pb-name', 'id', 'class'] as const

export function isLiveTag(tag: string): tag is LiveTag {
  return (LIVE_TAGS as readonly string[]).includes(tag)
}

export function escapeAttr(v: string): string {
  return v.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

export function escapeText(v: string): string {
  return v.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

/** The attributes a live block keeps, in a stable order: common ones first, then the block's own. */
export function liveAttrs(tag: LiveTag, attrs: Readonly<Record<string, unknown>>): [string, string][] {
  const out: [string, string][] = []
  for (const k of COMMON_LIVE_ATTRS) {
    const v = attrs[k]
    if (v === undefined || v === null || v === false) continue
    const s = String(v).trim()
    if (s) out.push([k, s])
  }
  // A block's own attributes stay even when empty: form="" is a signup block with no form picked yet.
  for (const k of LIVE_ATTRS[tag]) {
    const v = attrs[k]
    if (v === undefined || v === null || v === false) continue
    out.push([k, String(v)])
  }
  return out
}

/** A live block's HTML as stored and as the server draws it. */
export function liveHTML(tag: LiveTag, attrs: Readonly<Record<string, unknown>>): string {
  const a = liveAttrs(tag, attrs)
    .map(([k, v]) => ` ${k}="${escapeAttr(v)}"`)
    .join('')
  return `<${tag}${a}></${tag}>`
}

// ---------------------------------------------------------------- blocks and layers

/** The part of an element the model reads: a GrapesJS component or a DOM element behind the same face. */
export interface PbNode {
  readonly key: string
  readonly tag: string
  attr(name: string): string
  readonly classes: readonly string[]
  readonly children: readonly PbNode[]
}

const PB_KINDS = new Set<string>(BLOCK_KINDS)

/** What a top-level section is: its data-pb, or what its classes say. */
export function sectionKind(n: PbNode): BlockKind {
  const pb = n.attr('data-pb')
  if (PB_KINDS.has(pb)) return pb as BlockKind
  if (n.classes.includes('sp-hero') || n.classes.includes('lp-band')) return 'hero'
  return 'section'
}

function hasDescendant(n: PbNode, match: (x: PbNode) => boolean, depth = 3): boolean {
  if (depth < 0) return false
  return n.children.some((c) => match(c) || hasDescendant(c, match, depth - 1))
}

/**
 * What an element inside a section is when it's a block of its own (a live
 * block, an image, a named part such as "Heading, text and checklist"), or
 * null for plain content and structure.
 */
export function innerKind(n: PbNode): BlockKind | null {
  if (isLiveTag(n.tag)) return LIVE_KIND[n.tag]
  if (n.tag === 'img') return 'image'
  const pb = n.attr('data-pb')
  if (PB_KINDS.has(pb)) return pb as BlockKind
  const c = n.classes
  if (c.includes('pg-btns') || c.includes('st-btn') || c.includes('st-btn2')) return 'button'
  if (c.includes('bens') || c.includes('chapters') || c.includes('agenda')) return 'features'
  if (c.includes('quote') || c.includes('quotes')) return 'testimonials'
  if (c.includes('faq')) return 'faq'
  if (c.includes('bio') || c.includes('host')) return 'bio'
  if (c.includes('pg-cols')) return 'columns'
  if (c.includes('pg-hr')) return 'divider'
  if (c.includes('pg-space')) return 'spacer'
  if (c.includes('cov')) return 'image'
  if (!n.attr('data-pb-name')) return null
  if (hasDescendant(n, (x) => x.classes.includes('agenda') || x.classes.includes('chapters') || x.classes.includes('bens'))) return 'features'
  if (hasDescendant(n, (x) => x.tag === 'details')) return 'faq'
  return 'text'
}

/** A block in the layers: a top-level section, or a named part of one. */
export function isInnerBlock(n: PbNode): boolean {
  return isLiveTag(n.tag) || n.tag === 'img' || Boolean(n.attr('data-pb-name'))
}

export type LayerTag = { kind: 'columns'; n: number } | { kind: 'count'; n: number } | null

export interface LayerRow {
  /** The node the row selects. */
  key: string
  kind: BlockKind
  /** data-pb-name; empty for the kind's own name. */
  name: string
  sub: boolean
  tag: LayerTag
  /** Images: whether the alt text is set. */
  alt?: boolean
}

function count(n: PbNode, match: (x: PbNode) => boolean): number {
  let out = 0
  const walk = (x: PbNode) => {
    for (const c of x.children) {
      if (match(c)) out++
      else walk(c)
    }
  }
  walk(n)
  return out
}

function findFirst(n: PbNode, match: (x: PbNode) => boolean): PbNode | undefined {
  for (const c of n.children) {
    if (match(c)) return c
    const d = findFirst(c, match)
    if (d) return d
  }
  return undefined
}

/** "2 columns" for a hero or a columns block, the number of items for lists, quotes and questions. */
export function layerTag(n: PbNode, kind: BlockKind): LayerTag {
  switch (kind) {
    case 'hero': {
      const grid = findFirst(n, (x) => x.classes.includes('hero-grid') || x.classes.includes('lp-hero'))
      return grid && grid.children.length >= 2 ? { kind: 'columns', n: grid.children.length } : null
    }
    case 'columns': {
      const cols = findFirst(n, (x) => x.classes.includes('pg-cols'))
      return cols ? { kind: 'columns', n: cols.children.length } : null
    }
    case 'features': {
      const items = count(n, (x) => (x.tag === 'li' && x.attr('data-pb-name') === '') || x.classes.includes('ben'))
      return items ? { kind: 'count', n: items } : null
    }
    case 'testimonials': {
      const q = count(n, (x) => x.classes.includes('quote'))
      return q ? { kind: 'count', n: q } : null
    }
    case 'faq': {
      const q = count(n, (x) => x.tag === 'details')
      return q ? { kind: 'count', n: q } : null
    }
  }
  return null
}

/**
 * The live block a section stands for, when the section is only a frame
 * around it (a product band, a signup band): the layers show one row for
 * both, and the Block tab shows the live block's settings.
 */
export function soleLive(section: PbNode): PbNode | undefined {
  const live: PbNode[] = []
  let other = false
  const walk = (x: PbNode) => {
    for (const c of x.children) {
      if (isLiveTag(c.tag)) live.push(c)
      else if (isInnerBlock(c) || c.tag === 'img' || c.classes.includes('sec-head') || /^h[1-4]$/.test(c.tag) || c.tag === 'p') other = true
      else walk(c)
    }
  }
  walk(section)
  return live.length === 1 && !other ? live[0] : undefined
}

/** The rows of the layers panel for a page's top-level blocks (the site's header and footer come around them). */
export function layerRows(sections: readonly PbNode[]): LayerRow[] {
  const rows: LayerRow[] = []
  for (const s of sections) {
    const kind = sectionKind(s)
    const name = s.attr('data-pb-name')
    rows.push({ key: s.key, kind, name, sub: false, tag: layerTag(s, kind) })
    const sole = soleLive(s)
    const walk = (x: PbNode) => {
      for (const c of x.children) {
        if (isInnerBlock(c)) {
          const k = innerKind(c) ?? 'section'
          const n = c.attr('data-pb-name')
          if (c !== sole && !(n && n === name)) rows.push({ key: c.key, kind: k, name: n, sub: true, tag: null, ...(k === 'image' ? { alt: c.attr('alt').trim() !== '' } : {}) })
        }
        if (!isLiveTag(c.tag)) walk(c)
      }
    }
    walk(s)
  }
  return rows
}

// ---------------------------------------------------------------- style from the theme

export type Bg = 'white' | 'lavender' | 'indigo' | 'ink'
export type Space = 'compact' | 'normal' | 'roomy'
export const BGS: readonly Bg[] = ['white', 'lavender', 'indigo', 'ink']
export const SPACES: readonly Space[] = ['compact', 'normal', 'roomy']

const BG_CLASS: Record<Bg, string> = { white: '', lavender: 'lav', indigo: 'ind', ink: 'ink' }
const SPACE_CLASS: Record<Space, string> = { compact: 'cmp', normal: '', roomy: 'roomy' }
/** Sections whose base class already has the publisher's ground behind it. */
const GROUND_BASES = ['sp-hero', 'lp-band']

export function bgOf(n: Pick<PbNode, 'attr' | 'classes'>): Bg {
  const a = n.attr('data-pb-bg')
  if ((BGS as readonly string[]).includes(a)) return a as Bg
  if (n.classes.includes('ink')) return 'ink'
  if (n.classes.includes('ind')) return 'indigo'
  if (n.classes.includes('wht')) return 'white'
  if (n.classes.includes('lav') || GROUND_BASES.some((c) => n.classes.includes(c))) return 'lavender'
  return 'white'
}

export function spaceOf(n: Pick<PbNode, 'attr' | 'classes'>): Space {
  const a = n.attr('data-pb-space')
  if ((SPACES as readonly string[]).includes(a)) return a as Space
  if (n.classes.includes('cmp')) return 'compact'
  if (n.classes.includes('roomy')) return 'roomy'
  return 'normal'
}

/**
 * A section's classes with another background from the theme. A hero's own
 * class (sp-hero, lp-band) draws the publisher's ground already: lavender
 * needs no class there, and white needs one (wht).
 */
export function withBg(classes: readonly string[], bg: Bg): string[] {
  const out = classes.filter((c) => c !== 'lav' && c !== 'ind' && c !== 'ink' && c !== 'wht')
  const ground = GROUND_BASES.some((c) => out.includes(c))
  const cls = !ground ? BG_CLASS[bg] : bg === 'lavender' ? '' : bg === 'white' ? 'wht' : BG_CLASS[bg]
  if (cls) out.push(cls)
  return out
}

/** A section's classes with other spacing from the theme. */
export function withSpace(classes: readonly string[], space: Space): string[] {
  const out = classes.filter((c) => c !== 'cmp' && c !== 'roomy')
  const cls = SPACE_CLASS[space]
  if (cls) {
    // top0 (no top padding) keeps its place after the spacing.
    const i = out.indexOf('top0')
    if (i >= 0) out.splice(i, 0, cls)
    else out.push(cls)
  }
  return out
}

// ---------------------------------------------------------------- ids

const idRe = /^[a-z][a-z0-9-]{0,60}$/

/** A block id the page doesn't use yet: "b-faq", then "b-faq-2"… (as internal/pages UniqueIDs does). */
export function freeId(base: string, taken: ReadonlySet<string>): string {
  let b = base.toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+|-+$/g, '')
  if (!/^[a-z]/.test(b)) b = `b-${b}`
  b = b.slice(0, 50)
  if (!idRe.test(b)) b = 'b-block'
  let id = b
  for (let i = 2; taken.has(id); i++) id = `${b}-${i}`
  return id
}

// ---------------------------------------------------------------- lists in attributes

/** "eb-after, lead-after,buy" → ["eb-after", "lead-after", "buy"] */
export function parseList(v: string | undefined | null): string[] {
  return (v ?? '')
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean)
}

export function joinList(list: readonly string[]): string {
  return [...new Set(list.filter(Boolean))].join(',')
}

export const UNITS = ['days', 'hours', 'minutes', 'seconds'] as const
export type Unit = (typeof UNITS)[number]

export function parseUnits(v: string | undefined | null): Set<Unit> {
  return new Set(parseList(v).filter((u): u is Unit => (UNITS as readonly string[]).includes(u)))
}

/** The units a countdown shows, in their order; at least one. */
export function formatUnits(units: ReadonlySet<Unit>): string {
  const out = UNITS.filter((u) => units.has(u))
  return (out.length ? out : ['minutes']).join(',')
}

// ---------------------------------------------------------------- dates in a time zone

/** "2026-10-06T09:00" → { date: "2026-10-06", time: "09:00" } */
export function splitAt(at: string | undefined | null): { date: string; time: string } {
  const m = /^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2})$/.exec(at ?? '')
  return m ? { date: m[1]!, time: m[2]! } : { date: '', time: '' }
}

export function joinAt(date: string, time: string): string {
  return `${date}T${time || '09:00'}`
}

export const dateRe = /^\d{4}-\d{2}-\d{2}$/
export const clockRe = /^([01]\d|2[0-3]):[0-5]\d$/

export interface ZoneParts {
  year: number
  month: number
  day: number
  hour: number
  minute: number
  /** 0 is Sunday. */
  weekday: number
}

const weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const partFormats = new Map<string, Intl.DateTimeFormat>()

/** A moment's calendar fields in a time zone. */
export function zoneParts(at: Date, tz: string): ZoneParts | null {
  let f = partFormats.get(tz)
  if (!f) {
    try {
      f = new Intl.DateTimeFormat('en-US', { timeZone: tz, year: 'numeric', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric', weekday: 'short', hourCycle: 'h23' })
    } catch {
      return null
    }
    partFormats.set(tz, f)
  }
  const p: Record<string, string> = {}
  for (const x of f.formatToParts(at)) p[x.type] = x.value
  return {
    year: Number(p.year),
    month: Number(p.month),
    day: Number(p.day),
    hour: Number(p.hour) % 24,
    minute: Number(p.minute),
    weekday: Math.max(0, weekdays.indexOf(p.weekday ?? '')),
  }
}

/** The moment a date and a time on the clock of a time zone stand for ("2026-10-06", "09:00", America/New_York). */
export function zonedInstant(date: string, time: string, tz: string): Date | null {
  const d = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date)
  const t = /^(\d{2}):(\d{2})$/.exec(time)
  if (!d || !t) return null
  const wall = Date.UTC(Number(d[1]), Number(d[2]) - 1, Number(d[3]), Number(t[1]), Number(t[2]))
  // The zone's offset at the guess, then again at the corrected moment (DST edges).
  let guess = wall
  for (let i = 0; i < 2; i++) {
    const p = zoneParts(new Date(guess), tz)
    if (!p) return null
    const seen = Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute)
    guess += wall - seen
  }
  return new Date(guess)
}

/** "EDT", "UTC", "GMT+7": a time zone's short name at a moment. */
export function zoneAbbr(tz: string, at: Date): string {
  if (tz === 'UTC' || tz === 'Etc/UTC') return 'UTC'
  try {
    const p = new Intl.DateTimeFormat('en-US', { timeZone: tz, timeZoneName: 'short' }).formatToParts(at)
    return p.find((x) => x.type === 'timeZoneName')?.value ?? tz
  } catch {
    return tz
  }
}

/** "New York" for America/New_York, "UTC" for UTC. */
export function zoneCity(tz: string): string {
  if (tz === 'UTC' || tz === 'Etc/UTC') return 'UTC'
  const last = tz.split('/').pop() ?? tz
  return last.replace(/_/g, ' ')
}

/** The time zones offered: the browser's list, the given ones first. */
function supportedZones(): string[] {
  try {
    return (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf?.('timeZone') ?? []
  } catch {
    return []
  }
}

export function timeZones(first: readonly string[] = []): string[] {
  return [...new Set([...first.filter(Boolean), 'UTC', ...supportedZones()])]
}

/** Days, hours and minutes left before a moment (zero once it passed). */
export function timeLeft(at: Date, now: Date): { total: number; days: number; hours: number; minutes: number } {
  const total = Math.max(0, Math.floor((at.getTime() - now.getTime()) / 1000))
  return { total, days: Math.floor(total / 86400), hours: Math.floor((total % 86400) / 3600), minutes: Math.floor((total % 3600) / 60) }
}

/** The date fourteen days after a moment, on a time zone's calendar: where new countdowns and events start. */
export function datePlus(now: Date, days: number, tz: string): string {
  const p = zoneParts(new Date(now.getTime() + days * 86400_000), tz) ?? zoneParts(new Date(now.getTime() + days * 86400_000), 'UTC')!
  return `${p.year}-${String(p.month).padStart(2, '0')}-${String(p.day).padStart(2, '0')}`
}

// ---------------------------------------------------------------- the product block's button

function tokenRe(text: string): RegExp {
  const esc = text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  // "$29" in "Buy for $29", not in "Buy for $290".
  return new RegExp(`${esc}(?![\\d.,])`)
}

/** The button as readers see it: {price} and {product} filled in ("Buy for {price}" → "Buy for $29"). */
export function buttonDisplay(tpl: string, price: string, product: string): string {
  return tpl.replaceAll('{price}', price).replaceAll('{product}', product)
}

/**
 * The button's stored text from what the person typed: the product's price
 * and name become {price} and {product}, so the text follows the product.
 */
export function buttonTemplate(text: string, price: string, product: string): string {
  let out = text
  if (price && !out.includes('{price}')) out = out.replace(tokenRe(price), '{price}')
  if (product && !out.includes('{product}')) out = out.replace(tokenRe(product), '{product}')
  return out
}

// ---------------------------------------------------------------- videos

/** YouTube and Vimeo addresses, as internal/pages VideoOf reads them; an empty address is a video not picked yet. */
export function isVideoUrl(raw: string): boolean {
  const v = raw.trim()
  if (!v) return true
  let u: URL
  try {
    u = new URL(v)
  } catch {
    return false
  }
  if (u.protocol !== 'https:' && u.protocol !== 'http:') return false
  const host = u.hostname.replace(/^www\./, '').toLowerCase()
  const yt = /^[A-Za-z0-9_-]{6,20}$/
  if (host === 'youtu.be') return yt.test(u.pathname.replace(/^\/+|\/+$/g, ''))
  if (host === 'youtube.com' || host === 'm.youtube.com' || host === 'youtube-nocookie.com') {
    const id = /^\/(embed|shorts|live)\//.test(u.pathname) ? (u.pathname.split('/')[2] ?? '') : (u.searchParams.get('v') ?? '')
    return yt.test(id)
  }
  if (host === 'vimeo.com' || host === 'player.vimeo.com') return /^\d{4,12}$/.test(u.pathname.split('/').filter(Boolean).pop() ?? '')
  return false
}

// ---------------------------------------------------------------- comparing versions

export interface PageBlockText {
  id: string
  name: string
  lines: string[]
}

export type DiffOp = '-' | '+' | ' '
export interface DiffLine {
  op: DiffOp
  text: string
}

export interface BlockDiff {
  id: string
  name: string
  kind: 'changed' | 'added' | 'removed'
  lines: DiffLine[]
}

const textTags = new Set(['h1', 'h2', 'h3', 'h4', 'p', 'li', 'summary', 'blockquote', 'figcaption'])

function linesOf(root: Element): string[] {
  const out: string[] = []
  const visit = (el: Element) => {
    const tag = el.tagName.toLowerCase()
    if (textTags.has(tag)) {
      const t = (el.textContent ?? '').replace(/\s+/g, ' ').trim()
      if (t) out.push(t)
      return
    }
    if (isLiveTag(tag)) {
      const parts = LIVE_ATTRS[tag].filter((k) => el.getAttribute(k)).map((k) => `${k}: ${el.getAttribute(k)}`)
      out.push([tag, ...parts].join(' · '))
      return
    }
    if (tag === 'img') {
      out.push(`img · ${el.getAttribute('src') ?? ''} · alt: ${el.getAttribute('alt') ?? ''}`)
      return
    }
    for (const c of Array.from(el.children)) visit(c)
  }
  visit(root)
  return out
}

/** A page's top-level blocks with their text, line by line (as internal/pages/diff.go reads them). */
export function pageBlocks(html: string): PageBlockText[] {
  const doc = new DOMParser().parseFromString(`<!doctype html><body>${html}</body>`, 'text/html')
  return Array.from(doc.body.children).map((el, i) => ({
    id: el.getAttribute('data-pb-id') || `#${i}`,
    name: el.getAttribute('data-pb-name') || el.tagName.toLowerCase(),
    lines: linesOf(el),
  }))
}

/** A line diff by longest common subsequence: removed and added lines in order, without the unchanged ones. */
export function diffLines(a: readonly string[], b: readonly string[]): DiffLine[] {
  const n = Math.min(a.length, 400)
  const m = Math.min(b.length, 400)
  const lcs: Uint16Array[] = Array.from({ length: n + 1 }, () => new Uint16Array(m + 1))
  for (let i = n - 1; i >= 0; i--) for (let j = m - 1; j >= 0; j--) lcs[i]![j] = a[i] === b[j] ? lcs[i + 1]![j + 1]! + 1 : Math.max(lcs[i + 1]![j]!, lcs[i]![j + 1]!)
  const out: DiffLine[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      i++
      j++
    } else if (lcs[i + 1]![j]! >= lcs[i]![j + 1]!) out.push({ op: '-', text: a[i++]! })
    else out.push({ op: '+', text: b[j++]! })
  }
  while (i < n) out.push({ op: '-', text: a[i++]! })
  while (j < m) out.push({ op: '+', text: b[j++]! })
  return out
}

/** What changed between two versions of a page, block by block. */
export function diffPages(before: string, after: string): BlockDiff[] {
  const a = pageBlocks(before)
  const b = pageBlocks(after)
  const byId = new Map(a.map((x) => [x.id, x]))
  const inAfter = new Set<string>()
  const out: BlockDiff[] = []
  for (const x of b) {
    inAfter.add(x.id)
    const old = byId.get(x.id)
    if (!old) {
      out.push({ id: x.id, name: x.name, kind: 'added', lines: x.lines.map((text) => ({ op: '+', text })) })
      continue
    }
    const lines = diffLines(old.lines, x.lines)
    if (lines.length) out.push({ id: x.id, name: x.name, kind: 'changed', lines })
  }
  for (const x of a) if (!inAfter.has(x.id)) out.push({ id: x.id, name: x.name, kind: 'removed', lines: x.lines.map((text) => ({ op: '-', text })) })
  return out
}

/** The blocks (by name) that two people both changed since the version they started from. */
export function conflictBlocks(base: string, mine: string, theirs: string): { both: string[]; mine: string[]; theirs: string[] } {
  const raw = (html: string) => {
    const doc = new DOMParser().parseFromString(`<!doctype html><body>${html}</body>`, 'text/html')
    return new Map(Array.from(doc.body.children).map((el, i) => [el.getAttribute('data-pb-id') || `#${i}`, { name: el.getAttribute('data-pb-name') || el.tagName.toLowerCase(), html: el.outerHTML }]))
  }
  const b = raw(base)
  const changed = (other: Map<string, { name: string; html: string }>) => {
    const out = new Map<string, string>()
    for (const [id, x] of other) if (b.get(id)?.html !== x.html) out.set(id, x.name)
    for (const [id, x] of b) if (!other.has(id)) out.set(id, x.name)
    return out
  }
  const m = changed(raw(mine))
  const t = changed(raw(theirs))
  const both = [...m.keys()].filter((id) => t.has(id)).map((id) => m.get(id)!)
  return { both, mine: [...m.values()], theirs: [...t.values()] }
}
