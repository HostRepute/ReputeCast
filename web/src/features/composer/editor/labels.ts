import type { Component } from 'grapesjs'
import { t as tt } from '@/i18n'
import { feedItemsLabel, isFeedItems } from '@/features/digests/composer/feedComponent'
import { dc } from '@/features/digests/composer/messages'
import { growBlockLabel, growSectionLabel, isGrowBlock } from '@/features/grow/composer/blocks'

/** MJML tag of a component ("mj-section"), or "" for non-MJML nodes. */
export function tagOf(c: Component | undefined | null): string {
  if (!c) return ''
  const t = String(c.get('tagName') ?? '')
  return t.startsWith('mj-') || t === 'mjml' ? t : ''
}

/** Element children that are MJML components (comments and text skipped). */
export function mjChildren(c: Component): Component[] {
  return c.components().filter((x: Component) => tagOf(x) !== '')
}

/** The ReputeCast block a component was inserted from: a preceding <!-- rc:… --> comment. */
export function markerOf(c: Component): string {
  const parent = c.parent()
  if (!parent) return ''
  const siblings = parent.components().models as Component[]
  const i = siblings.indexOf(c)
  for (let j = i - 1; j >= 0; j--) {
    const s = siblings[j]!
    if (s.get('type') === 'comment') {
      const m = /^\s*rc:([\w-]+)\s*$/.exec(String(s.get('content') ?? ''))
      return m ? m[1]! : ''
    }
    if (s.get('type') === 'textnode' && !String(s.get('content') ?? '').trim()) continue
    return ''
  }
  return ''
}

/**
 * Blocks whose marker names the section. Card rows are named by their
 * layout ("2 columns"), as the boards' layer tree and selection label do.
 */
const markerLabels: Record<string, string> = {
  'header-logo': 'header',
  header: 'header',
  footer: 'footer',
  hero: 'hero',
}
const layoutMarkers = new Set(['article-cards', 'article-card', 'product-card', 'product-cards'])

function walk(c: Component, fn: (x: Component) => void) {
  fn(c)
  c.components().forEach((x: Component) => walk(x, fn))
}

function textOf(c: Component): string {
  const el = c.getEl()
  if (el) return el.textContent ?? ''
  return c.toHTML().replace(/<[^>]*>/g, ' ')
}

const unsubscribeRe = /unsubscribe|\{\{-?\s*unsubscribe_url/i
const addressRe = /\b\d{1,6}\s+[A-Za-zÀ-ÿ.'-]+(\s+[A-Za-zÀ-ÿ.'-]+){0,4}\s+(st|street|ave|avenue|rd|road|blvd|boulevard|ln|lane|dr|drive|way|ct|court|pl|place|sq|square)\b/i

function px(v: unknown): number {
  const s = String(v ?? '').trim()
  if (!s || s.endsWith('%')) return 0
  return parseFloat(s) || 0
}

/** Names a top-level body section the way the layer tree and warnings do. */
export function sectionLabel(c: Component): string {
  // A block's own name (rc-name: "Beta corner"), set in its Settings.
  const named = String((c.getAttributes() as Record<string, unknown>)['rc-name'] ?? '').trim()
  if (named) return named
  if (isFeedItems(c)) return tt(dc.blockTitle)
  if (isGrowBlock(c)) return growSectionLabel(c)
  const marker = markerOf(c)
  if (marker && !layoutMarkers.has(marker)) return markerLabels[marker] ?? marker.replace(/-/g, ' ')
  const tag = tagOf(c)
  if (tag === 'mj-hero') return 'hero'
  if (tag === 'mj-raw') return 'raw HTML'
  if (tag === 'mj-wrapper') return 'wrapper'
  const body = c.parent()
  const siblings = body ? mjChildren(body) : [c]
  const index = siblings.indexOf(c)
  let cols = 0,
    images = 0,
    texts = 0,
    buttons = 0,
    dividers = 0,
    navs = 0,
    maxImage = 0
  let html = ''
  walk(c, (x) => {
    switch (tagOf(x)) {
      case 'mj-column':
        cols++
        break
      case 'mj-image':
        images++
        maxImage = Math.max(maxImage, px((x.getAttributes() as Record<string, unknown>).width))
        break
      case 'mj-text':
        texts++
        html += x.toHTML() + ' '
        break
      case 'mj-button':
        buttons++
        break
      case 'mj-divider':
        dividers++
        break
      case 'mj-navbar':
      case 'mj-social':
        navs++
        break
    }
  })
  const plain = html.replace(/<[^>]*>/g, ' ')
  if (unsubscribeRe.test(html) || (index === siblings.length - 1 && addressRe.test(plain))) return 'footer'
  if (dividers > 0 && images === 0 && texts === 0 && buttons === 0) return 'divider'
  if (index === 0 && images > 0 && maxImage > 0 && maxImage <= 200 && (navs > 0 || texts > 0)) return 'header'
  if (cols >= 2) return `${cols} columns`
  if (images > 0 && (maxImage === 0 || maxImage >= 400) && texts > 0) return 'hero'
  if (images > 0 && texts === 0) return 'image'
  if (buttons > 0 && texts === 0) return 'button'
  if (texts > 0) return 'text'
  return tag.replace(/^mj-/, '')
}

/** "mj-section · 2 columns", "mj-column · 50%", "mj-image". */
export function componentLabel(c: Component): string {
  if (isFeedItems(c)) return feedItemsLabel(c)
  if (isGrowBlock(c)) return growBlockLabel(c)
  const tag = tagOf(c)
  if (!tag) return String(c.get('type') ?? c.get('tagName') ?? 'element')
  const parent = c.parent()
  if (tagOf(parent) === 'mj-body') return `${tag} · ${sectionLabel(c)}`
  if (tag === 'mj-column') {
    const w = String((c.getAttributes() as Record<string, unknown>).width ?? '')
    if (w) return `${tag} · ${w}`
    const n = parent ? mjChildren(parent).filter((x) => tagOf(x) === 'mj-column').length : 1
    return `${tag} · ${Math.round(100 / Math.max(1, n))}%`
  }
  return tag
}

/** The section (top-level body child) a component belongs to. */
export function sectionOf(c: Component | undefined | null): Component | undefined {
  let cur = c ?? undefined
  while (cur) {
    const p = cur.parent()
    if (!p) return undefined
    if (tagOf(p) === 'mj-body') return cur
    cur = p
  }
  return undefined
}

/** Element-index path from mj-body (comments skipped), as content checks report. */
export function pathOf(c: Component): number[] {
  const path: number[] = []
  let cur: Component | undefined = c
  while (cur) {
    const p: Component | undefined = cur.parent()
    if (!p) return []
    path.unshift(mjChildren(p).indexOf(cur))
    if (tagOf(p) === 'mj-body') return path
    cur = p
  }
  return []
}

/** The component at an element-index path from mj-body. */
export function atPath(body: Component | undefined, path: number[]): Component | undefined {
  let cur = body
  for (const i of path) {
    if (!cur) return undefined
    cur = mjChildren(cur)[i]
  }
  return cur
}

/** Breadcrumb from mjml to the component: ["mjml", "mj-body", "mj-section", …]. */
export function ancestry(c: Component): Component[] {
  const out: Component[] = []
  let cur: Component | undefined = c
  while (cur && tagOf(cur)) {
    out.unshift(cur)
    cur = cur.parent()
  }
  return out
}

export function textContent(c: Component): string {
  return textOf(c)
}
