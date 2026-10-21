import type { Component } from 'grapesjs'
import type { FormItem, ProductItem } from './data'
import { attrOf, attrsOf, blockOf, classesOf, columnNumber, isBlock, isTop, kindOf, liveTagOf, pathTo, tagOf } from './editor/nodes'
import { splitAt, zonedInstant } from './model'
import { m } from './messages'
import { kindCrumb, kindName, wallText, type T } from './wording'

/**
 * What the canvas and the status bar call the selection: the label over
 * the selected block ("Product · The Deliverability Field Guide",
 * "Countdown · ends 6 Oct, 09:00 New York") and the breadcrumb words
 * ("page › hero › column 1 › product").
 */

export interface LabelData {
  products: readonly ProductItem[]
  forms: readonly FormItem[]
  now: Date
}

/** A text part's word: a heading, a link, a question… */
function partLabel(t: T, c: Component): string {
  const tag = tagOf(c)
  const cls = classesOf(c)
  if (/^h[1-4]$/.test(tag)) return t(m.selHeading)
  if (tag === 'a') return cls.includes('st-btn') || cls.includes('st-btn2') ? t(m.kindButton) : t(m.selLink)
  if (tag === 'summary' || tag === 'details') return t(m.selQuestion)
  if (tag === 'li' || cls.includes('ben')) return t(m.selItem)
  if (tag === 'blockquote' || (tag === 'figure' && cls.includes('quote'))) return t(m.selQuote)
  return t(m.selText)
}

/** A text part's breadcrumb word: "heading", "item"… */
function partCrumb(t: T, c: Component): string {
  const tag = tagOf(c)
  const cls = classesOf(c)
  if (/^h[1-4]$/.test(tag)) return t(m.crumbHeading)
  if (tag === 'a') return cls.includes('st-btn') || cls.includes('st-btn2') ? t(m.crumbButton) : t(m.crumbLink)
  if (tag === 'summary' || tag === 'details') return t(m.crumbQuestion)
  if (tag === 'li' || cls.includes('ben')) return t(m.crumbItem)
  if (tag === 'blockquote' || (tag === 'figure' && cls.includes('quote'))) return t(m.crumbQuote)
  return t(m.crumbText)
}

/** The label over the selected block. */
export function selectionLabel(t: T, c: Component, d: LabelData): string {
  const live = liveTagOf(c)
  const a = attrsOf(c)
  if (live === 'rc-countdown') {
    const { date, time } = splitAt(a.at)
    const at = zonedInstant(date, time, a.tz || 'UTC')
    const w = wallText(date, time, a.tz || 'UTC')
    return t(at && at.getTime() <= d.now.getTime() ? m.labelCountdownEnded : m.labelCountdownEnds, { date: w.day, time: w.time, zone: w.zone })
  }
  if (live === 'rc-event') {
    const w = wallText(a.date ?? '', a.start ?? '', a.tz || 'UTC')
    return t(m.labelEvent, { date: w.day, time: w.time, zone: w.zone })
  }
  if (live === 'rc-product') {
    const p = d.products.find((x) => x.id === a.product)
    if (p) return t(m.labelProduct, { product: p.name })
  }
  if (live === 'rc-form') {
    const f = d.forms.find((x) => x.id === a.form)
    if (f) return t(f.status === 'live' ? m.labelForm : m.labelFormPaused, { form: f.name })
  }
  if (isBlock(c)) return attrOf(c, 'data-pb-name') || kindName(t, kindOf(c))
  const col = columnNumber(c)
  if (col) return t(m.columnTitle, { n: col })
  return partLabel(t, c)
}

export interface Crumb {
  c: Component
  word: string
}

/** The selection's breadcrumb from its block down: blocks, columns, then the selection itself. */
export function crumbs(t: T, sel: Component): Crumb[] {
  const out: Crumb[] = []
  for (const c of pathTo(sel)) {
    const col = columnNumber(c)
    if (isTop(c) || (isBlock(c) && c !== sel && !col)) out.push({ c, word: kindCrumb(t, kindOf(c)) })
    else if (col) out.push({ c, word: t(m.crumbColumn, { n: col }) })
    else if (c === sel) out.push({ c, word: isBlock(c) ? kindCrumb(t, kindOf(c)) : partCrumb(t, c) })
  }
  return out
}

/** The block the Block tab shows for a selection: the selection's block, or the live block a section stands for. */
export function inspectedBlock(sel: Component | null): Component | undefined {
  return sel ? (blockOf(sel) ?? undefined) : undefined
}
