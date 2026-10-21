import type { BlockIconName } from './icons'
import { escapeAttr, escapeText, type BlockKind } from './model'

/**
 * The palette (PageBuilder · Blocks) and the markup each block adds: the
 * public pages' own classes (internal/publicpages/static/pages.css), as
 * internal/pages/templates.go writes them, so a block looks on the canvas
 * the way readers will see it. Section blocks go between the page's
 * blocks; content blocks also go inside a column, and get a section of
 * their own when they're dropped between blocks.
 */

export type PaletteGroup = 'layout' | 'content' | 'conversion' | 'site'

export type PaletteId =
  | 'section'
  | 'columns2'
  | 'columns3'
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

export interface PaletteItem {
  id: PaletteId
  kind: BlockKind
  group: PaletteGroup
  /** section: only between blocks; content: also inside columns. */
  level: 'section' | 'content'
  icon: BlockIconName
  /** Words the search also matches (English, never shown). */
  keywords: string
}

export const PALETTE: readonly PaletteItem[] = [
  { id: 'section', kind: 'section', group: 'layout', level: 'section', icon: 'section', keywords: 'block band text' },
  { id: 'columns2', kind: 'columns', group: 'layout', level: 'section', icon: 'columns', keywords: 'two side by side grid' },
  { id: 'columns3', kind: 'columns', group: 'layout', level: 'section', icon: 'columns3', keywords: 'three side by side grid' },
  { id: 'divider', kind: 'divider', group: 'layout', level: 'section', icon: 'divider', keywords: 'line separator hr' },
  { id: 'spacer', kind: 'spacer', group: 'layout', level: 'section', icon: 'spacer', keywords: 'gap space' },
  { id: 'hero', kind: 'hero', group: 'content', level: 'section', icon: 'hero', keywords: 'banner top heading' },
  { id: 'text', kind: 'text', group: 'content', level: 'content', icon: 'text', keywords: 'paragraph copy heading prose' },
  { id: 'image', kind: 'image', group: 'content', level: 'content', icon: 'image', keywords: 'picture photo cover' },
  { id: 'button', kind: 'button', group: 'content', level: 'content', icon: 'button', keywords: 'cta link' },
  { id: 'video', kind: 'video', group: 'content', level: 'content', icon: 'video', keywords: 'youtube vimeo player' },
  { id: 'features', kind: 'features', group: 'content', level: 'section', icon: 'features', keywords: 'benefits list chapters inside' },
  { id: 'testimonials', kind: 'testimonials', group: 'content', level: 'section', icon: 'testimonials', keywords: 'quotes reviews' },
  { id: 'faq', kind: 'faq', group: 'content', level: 'section', icon: 'faq', keywords: 'questions answers' },
  { id: 'signup', kind: 'signup', group: 'conversion', level: 'content', icon: 'signup', keywords: 'form subscribe waitlist register' },
  { id: 'countdown', kind: 'countdown', group: 'conversion', level: 'content', icon: 'countdown', keywords: 'timer launch deadline' },
  { id: 'event', kind: 'event', group: 'conversion', level: 'content', icon: 'event', keywords: 'webinar date time calendar' },
  { id: 'product', kind: 'product', group: 'conversion', level: 'content', icon: 'product', keywords: 'buy price checkout stripe' },
  { id: 'products', kind: 'products', group: 'conversion', level: 'content', icon: 'products', keywords: 'shop cards grid' },
  { id: 'store', kind: 'store', group: 'conversion', level: 'content', icon: 'store', keywords: 'shopify woocommerce' },
  { id: 'tiers', kind: 'tiers', group: 'conversion', level: 'content', icon: 'tiers', keywords: 'pricing paid plans membership' },
  { id: 'posts', kind: 'posts', group: 'site', level: 'content', icon: 'posts', keywords: 'blog latest tag' },
  { id: 'bio', kind: 'bio', group: 'site', level: 'section', icon: 'bio', keywords: 'author about person' },
]

export const PALETTE_GROUPS: readonly PaletteGroup[] = ['layout', 'content', 'conversion', 'site']

/** The words new blocks start with, in the app's language. */
export interface BlockCopy {
  section: string
  sectionHeading: string
  sectionText: string
  columns2: string
  columns3: string
  columnHeading: string
  columnText: string
  divider: string
  spacer: string
  hero: string
  heroEyebrow: string
  heroHeading: string
  heroLead: string
  heroButton: string
  heroArt: string
  heroCopy: string
  text: string
  textBody: string
  image: string
  button: string
  buttonLabel: string
  video: string
  features: string
  featuresEyebrow: string
  featuresHeading: string
  feature1: string
  feature2: string
  feature3: string
  itemText: string
  testimonials: string
  quotesEyebrow: string
  quotesHeading: string
  quote: string
  quoteName: string
  quoteRole: string
  faq: string
  faqEyebrow: string
  faqHeading: string
  faqQ1: string
  faqA1: string
  faqQ2: string
  faqA2: string
  signup: string
  formHeading: string
  formHint: string
  countdown: string
  countdownLabel: string
  event: string
  reminders: string
  product: string
  buyButton: string
  products: string
  store: string
  tiers: string
  posts: string
  postsHeading: string
  bio: string
  bioEyebrow: string
  bioName: string
  bioRole: string
  bioText: string
}

export interface BlockContext {
  copy: BlockCopy
  /** The workspace's time zone, for countdowns and events. */
  tz: string
  /** The date new countdowns and events start on (two weeks ahead). */
  date: string
  /** The first live signup form, '' for none. */
  formId: string
  /** The first published product, '' for none. */
  productId: string
  /** The page's name: the event's title in calendars. */
  pageName: string
  /** The publisher's initials, for the author bio. */
  initials: string
}

/** The starter picture new image blocks show until one is chosen (alt text missing on purpose: the checks ask for it). */
export const STARTER_IMAGE = '/media/starters/hero.png'

const e = escapeText
const a = escapeAttr

function section(kind: string, name: string, cls: string, bg: string, space: string, inner: string): string {
  return `<section data-pb="${kind}" data-pb-name="${a(name)}" data-pb-bg="${bg}" data-pb-space="${space}"${cls ? ` class="${cls}"` : ''}>${inner}</section>`
}

function column(c: BlockCopy): string {
  return `<div class="pg-col"><div class="prose"><h3>${e(c.columnHeading)}</h3><p>${e(c.columnText)}</p></div></div>`
}

/** A content block's own markup, for a column or a section of its own. */
export function contentHtml(id: PaletteId, ctx: BlockContext): string {
  const c = ctx.copy
  switch (id) {
    case 'text':
      return `<div class="prose" data-pb="text" data-pb-name="${a(c.text)}"><p>${e(c.textBody)}</p></div>`
    case 'image':
      return `<img data-pb="image" data-pb-name="${a(c.image)}" class="pg-img" src="${STARTER_IMAGE}" alt="">`
    case 'button':
      return `<div class="pg-btns" data-pb="button" data-pb-name="${a(c.button)}"><a class="st-btn lg" href="/">${e(c.buttonLabel)}</a></div>`
    case 'video':
      return `<rc-video data-pb-name="${a(c.video)}" url="" title=""></rc-video>`
    case 'signup':
      return `<rc-form data-pb-name="${a(c.signup)}" form="${a(ctx.formId)}" heading="${a(c.formHeading)}" hint="${a(c.formHint)}"></rc-form>`
    case 'countdown':
      return `<rc-countdown data-pb-name="${a(c.countdown)}" at="${ctx.date}T09:00" tz="${a(ctx.tz)}" units="days,hours,minutes" end="hide" label="${a(c.countdownLabel)}"></rc-countdown>`
    case 'event':
      return `<rc-event data-pb-name="${a(c.event)}" date="${ctx.date}" start="17:00" end="17:30" tz="${a(ctx.tz)}" local="1" where="online" calendar="1" title="${a(ctx.pageName)}" reminders="${a(c.reminders)}"></rc-event>`
    case 'product':
      return `<rc-product data-pb-name="${a(c.product)}" product="${a(ctx.productId)}" button="${a(c.buyButton)}" price="1" offer="1" layout="box"></rc-product>`
    case 'products':
      return `<rc-products data-pb-name="${a(c.products)}" source="digital" count="3"></rc-products>`
    case 'store':
      return `<rc-store-product data-pb-name="${a(c.store)}"></rc-store-product>`
    case 'tiers':
      return `<rc-tiers data-pb-name="${a(c.tiers)}" period="month"></rc-tiers>`
    case 'posts':
      return `<rc-posts data-pb-name="${a(c.posts)}" count="3" heading="${a(c.postsHeading)}"></rc-posts>`
  }
  return ''
}

/** The section a content block gets when it's dropped between blocks. */
export function wrapContent(id: PaletteId, kind: BlockKind, name: string, inner: string): string {
  switch (id) {
    case 'signup':
      return section(kind, name, 'sp-sec cmp lav', 'lavender', 'compact', `<div class="sp-in nar">${inner}</div>`)
    case 'products':
    case 'tiers':
    case 'posts':
    case 'image':
      return section(kind, name, 'sp-sec', 'white', 'normal', `<div class="sp-in">${inner}</div>`)
    case 'button':
      return section(kind, name, 'sp-sec cmp', 'white', 'compact', `<div class="sp-in nar">${inner}</div>`)
  }
  return section(kind, name, 'sp-sec', 'white', 'normal', `<div class="sp-in nar">${inner}</div>`)
}

/** The block as a section of its own, ready to go between the page's blocks. */
export function sectionHtml(id: PaletteId, ctx: BlockContext): string {
  const c = ctx.copy
  switch (id) {
    case 'section':
      return section('section', c.section, 'sp-sec', 'white', 'normal', `<div class="sp-in nar"><div class="prose"><h2>${e(c.sectionHeading)}</h2><p>${e(c.sectionText)}</p></div></div>`)
    case 'columns2':
      return section('columns', c.columns2, 'sp-sec', 'white', 'normal', `<div class="sp-in"><div class="pg-cols c2">${column(c)}${column(c)}</div></div>`)
    case 'columns3':
      return section('columns', c.columns3, 'sp-sec', 'white', 'normal', `<div class="sp-in"><div class="pg-cols c3">${column(c)}${column(c)}${column(c)}</div></div>`)
    case 'divider':
      return section('divider', c.divider, 'sp-sec cmp', 'white', 'compact', `<div class="sp-in"><hr class="pg-hr"></div>`)
    case 'spacer':
      return section('spacer', c.spacer, '', 'white', 'normal', `<div class="pg-space" aria-hidden="true"></div>`)
    case 'hero':
      return section(
        'hero',
        c.hero,
        'sp-hero',
        'lavender',
        'normal',
        `<div class="sp-in"><div class="hero-grid"><div class="hero-copy" data-pb-name="${a(c.heroCopy)}"><span class="eyebrow">${e(c.heroEyebrow)}</span><h1>${e(c.heroHeading)}</h1><p class="sp-lead">${e(c.heroLead)}</p>` +
          `<div class="pg-btns"><a class="st-btn lg" href="/">${e(c.heroButton)}</a></div></div><div class="hero-art"><img data-pb="image" data-pb-name="${a(c.heroArt)}" class="pg-img" src="${STARTER_IMAGE}" alt=""></div></div></div>`,
      )
    case 'features': {
      const items = [c.feature1, c.feature2, c.feature3]
        .map((t, i) => `<li><span class="n">${String(i + 1).padStart(2, '0')}</span><span class="t"><b>${e(t)}</b><span class="d">${e(c.itemText)}</span></span></li>`)
        .join('')
      return section('features', c.features, 'sp-sec w', 'white', 'normal', `<div class="sp-in"><div class="sec-head"><span class="eyebrow">${e(c.featuresEyebrow)}</span><h2 class="sp-h2">${e(c.featuresHeading)}</h2></div><ol class="chapters">${items}</ol></div>`)
    }
    case 'testimonials':
      return section(
        'testimonials',
        c.testimonials,
        'sp-sec',
        'white',
        'normal',
        `<div class="sp-in"><div class="sec-head"><span class="eyebrow">${e(c.quotesEyebrow)}</span><h2 class="sp-h2">${e(c.quotesHeading)}</h2></div><div class="quotes">${quoteHtml(c).repeat(3)}</div></div>`,
      )
    case 'faq':
      return section(
        'faq',
        c.faq,
        'sp-sec',
        'white',
        'normal',
        `<div class="sp-in nar"><div class="sec-head"><span class="eyebrow">${e(c.faqEyebrow)}</span><h2 class="sp-h2">${e(c.faqHeading)}</h2></div><div class="faq">` +
          `<details open><summary>${e(c.faqQ1)}</summary><p>${e(c.faqA1)}</p></details><details><summary>${e(c.faqQ2)}</summary><p>${e(c.faqA2)}</p></details></div></div>`,
      )
    case 'bio':
      return section(
        'bio',
        c.bio,
        'sp-sec lav',
        'lavender',
        'normal',
        `<div class="sp-in nar"><div class="bio"><span class="av7" aria-hidden="true">${e(ctx.initials)}</span><div class="tx"><span class="eyebrow">${e(c.bioEyebrow)}</span>` +
          `<h2 class="sp-h2">${e(c.bioName)}</h2><span class="role">${e(c.bioRole)}</span><p>${e(c.bioText)}</p></div></div></div>`,
      )
  }
  const item = PALETTE.find((p) => p.id === id)
  return item ? wrapContent(id, item.kind, nameOf(id, c), contentHtml(id, ctx)) : ''
}

/** A testimonial, as the Testimonials block adds them. */
export function quoteHtml(c: Pick<BlockCopy, 'quote' | 'quoteName' | 'quoteRole'>): string {
  return `<figure class="quote"><blockquote>${e(c.quote)}</blockquote><figcaption><span class="qa" aria-hidden="true">AB</span><span class="qn"><b>${e(c.quoteName)}</b><span class="qr">${e(c.quoteRole)}</span></span></figcaption></figure>`
}

/** A question of the FAQ block. */
export function questionHtml(q: string, answer: string): string {
  return `<details><summary>${e(q)}</summary><p>${e(answer)}</p></details>`
}

/** An item of a features list: a numbered chapter, or a benefit card. */
export function featureItemHtml(kind: 'chapter' | 'benefit', n: number, title: string, text: string): string {
  return kind === 'chapter'
    ? `<li><span class="n">${String(n).padStart(2, '0')}</span><span class="t"><b>${e(title)}</b><span class="d">${e(text)}</span></span></li>`
    : `<div class="ben"><b>${e(title)}</b>${e(text)}</div>`
}

/** The block's name in the layers. */
export function nameOf(id: PaletteId, c: BlockCopy): string {
  switch (id) {
    case 'columns2':
      return c.columns2
    case 'columns3':
      return c.columns3
  }
  return (c as unknown as Record<string, string>)[id] ?? id
}

export function paletteItem(id: PaletteId): PaletteItem {
  return PALETTE.find((p) => p.id === id)!
}
