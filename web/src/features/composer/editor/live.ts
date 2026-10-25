import type { Component, Editor } from 'grapesjs'

/**
 * The words of a text being typed on the canvas. GrapesJS copies them into
 * its model only when the editing ends, so a save that waited for that
 * would lose everything typed since the last click elsewhere if the tab
 * crashed (CO-1). These helpers read the live words from the text's DOM
 * without ending its editing.
 */

/** Attributes the canvas adds to elements and the model never has. */
const CANVAS_ATTR = /^(data-gjs-|contenteditable$|spellcheck$|draggable$)/

/**
 * The HTML of a text as the person typed it, cleaned of what the canvas
 * adds (GrapesJS's data-gjs-* attributes, its gjs-* classes, the editing
 * flags), so it reads as the model would export it.
 */
export function cleanLiveHtml(html: string, doc: Document = document): string {
  const box = doc.createElement('div')
  box.innerHTML = html
  box.querySelectorAll('*').forEach((el) => {
    for (const a of Array.from(el.attributes)) {
      if (CANVAS_ATTR.test(a.name)) el.removeAttribute(a.name)
    }
    if (el.hasAttribute('class')) {
      const kept = Array.from(el.classList).filter((c) => !c.startsWith('gjs-'))
      if (kept.length) el.setAttribute('class', kept.join(' '))
      else el.removeAttribute('class')
    }
    // A Liquid chip exports as its token (see liquidChips), not as its label.
    if (el.hasAttribute('data-rc-liquid')) el.textContent = ''
  })
  return box.innerHTML
}

/** Replaces the inner part of an element's exported MJML: `<mj-text …>stale</mj-text>` → `<mj-text …>live</mj-text>`. */
export function replaceInner(exported: string, tag: string, inner: string): string {
  const open = openingTagEnd(exported)
  const close = exported.toLowerCase().lastIndexOf(`</${tag.toLowerCase()}`)
  if (open < 0 || close < open) return exported
  return `${exported.slice(0, open + 1)}${inner}${exported.slice(close)}`
}

/** Index of the `>` that ends the first tag, skipping quoted attribute values. */
function openingTagEnd(src: string): number {
  let quote: string | null = null
  for (let i = 1; i < src.length; i++) {
    const ch = src[i]
    if (quote) {
      if (ch === quote) quote = null
    } else if (ch === '"' || ch === "'") quote = ch
    else if (ch === '>') return i
  }
  return -1
}

type ViewWithContainer = { getChildrenContainer?: () => HTMLElement; el?: HTMLElement }

/** The live inner HTML of the text being edited, or null when nothing is. */
export function liveInner(editor: Editor): { component: Component; html: string } | null {
  const comp = editor.getEditing() as Component | null | undefined
  if (!comp) return null
  const view = comp.getView?.() as unknown as ViewWithContainer | undefined
  const el = view?.getChildrenContainer?.() ?? view?.el
  if (!el) return null
  return { component: comp, html: cleanLiveHtml(el.innerHTML, el.ownerDocument) }
}

/**
 * Runs `exportFn` (an export of the canvas) as if the text being edited
 * already held the words typed so far. The model isn't touched: the text's
 * own export is replaced for the duration of the call (exports are
 * synchronous, so nothing else sees it).
 */
export function withLiveText<T>(editor: Editor, exportFn: () => T): T {
  const live = liveInner(editor)
  if (!live) return exportFn()
  const comp = live.component as Component & { toHTML: (opts?: unknown) => string }
  const proto = Object.getPrototypeOf(comp) as { toHTML: (this: Component, opts?: unknown) => string }
  const tag = String(comp.get('tagName') ?? '')
  const own = Object.prototype.hasOwnProperty.call(comp, 'toHTML')
  const previous = comp.toHTML
  comp.toHTML = function (this: Component, opts?: unknown) {
    return replaceInner(proto.toHTML.call(this, opts), tag, live.html)
  }
  try {
    return exportFn()
  } finally {
    if (own) comp.toHTML = previous
    else delete (comp as { toHTML?: unknown }).toHTML
  }
}
