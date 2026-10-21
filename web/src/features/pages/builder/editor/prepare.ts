/**
 * Prepares a page's HTML for the canvas: GrapesJS reads `data-gjs-*`
 * attributes as component properties (and drops them from the HTML it
 * writes back), so each element is told what it may do before the editor
 * parses it. Blocks keep their structure: the section's inner wrappers
 * can't be selected or removed on their own (a click selects the block
 * around them), nothing is dragged around inside a section (blocks move
 * with Move up and Move down), and content blocks from the palette drop
 * only into the page and the columns.
 */

/** Containers content blocks can be dropped into. */
export const DROP_CONTAINERS = ['sp-in', 'pg-col', 'hero-copy', 'lp-copy', 'lp-body', 'lp-cols', 'hero-art'] as const

/** Where a content block from the palette may go: the page (it's wrapped in a section there) or a container. */
export const CONTENT_TARGETS = ['[data-gjs-type="wrapper"]', ...DROP_CONTAINERS.map((c) => `.${c}`)].join(', ')

/** Where a section may go: the page only. */
export const SECTION_TARGETS = '[data-gjs-type="wrapper"]'

/** Items of a list block: a question, a testimonial, a benefit, an agenda line. */
function isItem(el: Element): boolean {
  const tag = el.tagName.toLowerCase()
  return tag === 'li' || tag === 'details' || (tag === 'figure' && el.classList.contains('quote')) || el.classList.contains('ben')
}

function set(el: Element, props: Record<string, boolean | string>) {
  for (const [k, v] of Object.entries(props)) el.setAttribute(`data-gjs-${k}`, String(v))
}

/** Tells one element inside a block what it may do. */
function annotate(el: Element) {
  const tag = el.tagName.toLowerCase()
  if (tag.startsWith('rc-')) return
  const container = DROP_CONTAINERS.some((c) => el.classList.contains(c))
  const base = { draggable: false, droppable: container }
  if (el.hasAttribute('data-pb-name') || tag === 'img' || isItem(el)) {
    set(el, { ...base, selectable: true, hoverable: true, removable: true, copyable: true })
    return
  }
  if (el.classList.contains('pg-space') || el.classList.contains('pg-hr')) {
    // A spacer's or a divider's line: the block itself is selected.
    set(el, { ...base, selectable: false, hoverable: false, removable: false, copyable: false })
    return
  }
  if (el.classList.contains('pg-col')) {
    // A column: selected to drop blocks into it; it goes with its block.
    set(el, { ...base, selectable: true, hoverable: true, removable: false, copyable: false })
    return
  }
  if (el.children.length > 0 && !['p', 'h1', 'h2', 'h3', 'h4', 'summary', 'blockquote', 'a', 'b', 'strong', 'em', 'i', 'u', 's', 'small'].includes(tag)) {
    // Structure: the block around it is selected instead.
    set(el, { ...base, selectable: false, hoverable: false, removable: false, copyable: false })
    return
  }
  set(el, base)
}

/**
 * The page's HTML with every element annotated for the editor. Top-level
 * sections are blocks (their component type says what they may do); a
 * fragment for inside a block (a list item, a content block for a column)
 * is annotated from its first element.
 */
export function prepareHtml(html: string): string {
  const tpl = document.createElement('template')
  tpl.innerHTML = html
  for (const top of Array.from(tpl.content.children)) {
    if (!(top.tagName === 'SECTION' && top.hasAttribute('data-pb'))) annotate(top)
    for (const el of Array.from(top.querySelectorAll('*'))) annotate(el)
  }
  return tpl.innerHTML
}

/** Removes what the editor adds, should any of it reach the saved HTML. */
export function cleanHtml(html: string): string {
  const tpl = document.createElement('template')
  tpl.innerHTML = html
  for (const el of Array.from(tpl.content.querySelectorAll('*'))) {
    for (const a of Array.from(el.attributes)) {
      if (a.name.startsWith('data-gjs-') || a.name === 'contenteditable' || a.name === 'draggable') el.removeAttribute(a.name)
    }
    const cls = Array.from(el.classList).filter((c) => c.startsWith('gjs-'))
    if (cls.length) el.classList.remove(...cls)
    if (el.getAttribute('class') === '') el.removeAttribute('class')
  }
  return tpl.innerHTML.trim()
}
