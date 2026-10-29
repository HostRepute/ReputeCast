import type { Component, Editor } from 'grapesjs'
import { isFeedItems } from '@/features/digests/composer/feedComponent'
import { isGrowBlock } from '@/features/grow/composer/blocks'
import { chipHtml, chipLabelOf, LIQUID_ATTR, splitTokens } from '../editor/liquid'
import { markerOf, mjChildren, sectionName, tagOf } from '../editor/labels'
import { SYNCED_ATTR } from '../editor/synced'
import { parseInert, stripActive } from '../inert'

/**
 * Write mode (CO-9, ComposerWrite board) reads the same email as Design
 * mode: the canvas's components, in reading order. Texts, images, buttons
 * and dividers are edited in place; synced blocks, polls, feeds and the
 * other layout blocks show as cards that say where they change. Nothing
 * here rewrites what it doesn't edit, so switching modes loses nothing.
 */

export type DocItem =
  // synced: "header", "footer" or a named block's key (R4.A3).
  | { kind: 'synced'; key: string; comp: Component; synced: string }
  | { kind: 'column'; key: string; section: string; n: number; count: number }
  | { kind: 'text'; key: string; comp: Component }
  | { kind: 'image'; key: string; comp: Component; hero: boolean }
  | { kind: 'button'; key: string; comp: Component }
  | { kind: 'divider'; key: string; comp: Component }
  | { kind: 'other'; key: string; comp: Component; label: string }

/** The body component of the canvas. */
export function bodyOf(editor: Editor | null | undefined): Component | undefined {
  return editor?.getWrapper()?.findType('mj-body')[0]
}

/** Sections whose words aren't edited as text: they change in Design or elsewhere. */
function wholeBlock(section: Component): boolean {
  if (isFeedItems(section) || isGrowBlock(section)) return true
  const attrs = section.getAttributes() as Record<string, unknown>
  if (attrs['rc-poll'] !== undefined) return true
  const marker = markerOf(section)
  return marker === 'sponsor' || marker === 'product-card' || marker === 'store-product'
}

function contentItems(col: Component, out: DocItem[], heroImage = false) {
  for (const x of mjChildren(col)) {
    const key = x.cid
    switch (tagOf(x)) {
      case 'mj-text':
        out.push({ kind: 'text', key, comp: x })
        break
      case 'mj-image':
        out.push({ kind: 'image', key, comp: x, hero: heroImage })
        break
      case 'mj-button':
        out.push({ kind: 'button', key, comp: x })
        break
      case 'mj-divider':
        out.push({ kind: 'divider', key, comp: x })
        break
      case 'mj-spacer':
        break
      default:
        out.push({ kind: 'other', key, comp: x, label: tagOf(x) })
    }
  }
}

function columnsOf(section: Component): Component[] {
  const cols: Component[] = []
  for (const x of mjChildren(section)) {
    if (tagOf(x) === 'mj-column') cols.push(x)
    else if (tagOf(x) === 'mj-group') cols.push(...mjChildren(x).filter((y) => tagOf(y) === 'mj-column'))
  }
  return cols
}

function sectionItems(section: Component, out: DocItem[], first: boolean) {
  const tag = tagOf(section)
  const synced = String((section.getAttributes() as Record<string, unknown>)[SYNCED_ATTR] ?? '')
  // Any synced block, named ones too (R4.A3): it changes in the brand kit only.
  if (synced) {
    out.push({ kind: 'synced', key: section.cid, comp: section, synced })
    return
  }
  if (wholeBlock(section) || tag === 'mj-raw') {
    out.push({ kind: 'other', key: section.cid, comp: section, label: sectionName(section) })
    return
  }
  if (tag === 'mj-hero') {
    contentItems(section, out)
    return
  }
  if (tag === 'mj-wrapper') {
    for (const inner of mjChildren(section)) sectionItems(inner, out, false)
    return
  }
  const cols = columnsOf(section)
  if (cols.length > 1) {
    const name = sectionName(section)
    cols.forEach((col, i) => {
      out.push({ kind: 'column', key: `${col.cid}-tag`, section: name, n: i + 1, count: cols.length })
      contentItems(col, out)
    })
    return
  }
  // A first section that is one image reads as the hero image.
  const only = cols[0] ? mjChildren(cols[0]) : []
  for (const col of cols) contentItems(col, out, first && only.length === 1 && tagOf(only[0]) === 'mj-image')
}

/** The email in reading order, as Write mode shows it. */
export function docItems(editor: Editor | null | undefined): DocItem[] {
  const body = bodyOf(editor)
  if (!body) return []
  const out: DocItem[] = []
  // The first section after a synced header can be the hero image.
  let first = true
  for (const s of mjChildren(body)) {
    sectionItems(s, out, first)
    if (!(s.getAttributes() as Record<string, unknown>)[SYNCED_ATTR]) first = false
  }
  return out
}

/** A text component's words as MJML holds them (Liquid as written). */
export function textHtml(comp: Component): string {
  return comp.getInnerHTML().trim()
}

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

/**
 * The HTML a text is edited as: Liquid output shows as chips ("First name
 * · fallback “there”") that can't be typed into. Tokens in attributes
 * (an href) stay as they are.
 */
export function toEditable(html: string, editor: Editor | null | undefined): string {
  // The text goes into a live element: parsed inertly, without anything that could run there.
  const box = parseInert(html)
  stripActive(box)
  const doc = box.ownerDocument
  const walker = doc.createTreeWalker(box, 4 /* NodeFilter.SHOW_TEXT */)
  const texts: Text[] = []
  for (let n = walker.nextNode(); n; n = walker.nextNode()) if ((n.textContent ?? '').includes('{{')) texts.push(n as Text)
  for (const tn of texts) {
    const parts = splitTokens(tn.textContent ?? '')
    if (!parts.some((p) => p.token)) continue
    const html = parts.map((p) => (p.token ? chipHtml(p.token, chipLabelOf(editor, p.token)) : esc(p.text ?? ''))).join('')
    const tmp = doc.createElement('span')
    tmp.innerHTML = html
    tn.replaceWith(...Array.from(tmp.childNodes))
  }
  return box.innerHTML
}

/** What an edited text holds, back as MJML: chips become their tokens again. */
export function fromEditable(el: HTMLElement): string {
  const copy = el.cloneNode(true) as HTMLElement
  copy.querySelectorAll(`[${LIQUID_ATTR}]`).forEach((chip) => chip.replaceWith(copy.ownerDocument.createTextNode(chip.getAttribute(LIQUID_ATTR) ?? '')))
  // Browsers leave a lone <br> in an emptied block, and a trailing one after typing.
  return copy.innerHTML
    .replace(/<br>(<\/(?:p|h[1-6]|li)>)/g, '$1')
    .replace(/^(?:<br>)+$/, '')
    .trim()
}

/** True when a text holds nothing to read (an emptied new block). */
export function isBlank(html: string): boolean {
  // Read as the browser reads it rather than stripping tags with a pattern:
  // text only, with non-breaking spaces as the spaces they are.
  const box = parseInert(html)
  return !(box.textContent ?? '').trim() && !box.querySelector('img')
}
