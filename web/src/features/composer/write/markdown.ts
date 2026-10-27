/**
 * Write mode's markdown shortcuts (CO-9, ComposerWrite: "Type # for a
 * heading, - for a list, ** for bold, / for blocks"). They work on the
 * paragraph being typed in a text's editable element.
 */

export type Shortcut = 'heading' | 'subheading' | 'bullets' | 'numbers'

const BLOCK = /^(P|DIV|H[1-6]|LI)$/

/** The paragraph (or heading, list item) directly holding a node, inside root. */
export function paragraphOf(node: Node | null, root: HTMLElement): HTMLElement | null {
  let n: Node | null = node
  while (n && n !== root) {
    if (n.nodeType === 1 && BLOCK.test((n as HTMLElement).tagName)) return n as HTMLElement
    n = n.parentNode
  }
  return null
}

/** The shortcut a paragraph starts with ("# ", "## ", "- ", "* ", "1. "), and the words after it. */
export function shortcutOf(text: string): { kind: Shortcut; rest: string } | null {
  const t = text.replace(/\u00a0/g, ' ')
  const m = /^(#{1,2}|[-*]|1\.) (.*)$/s.exec(t)
  if (!m) return null
  const kind: Shortcut = m[1] === '#' ? 'heading' : m[1] === '##' ? 'subheading' : m[1] === '1.' ? 'numbers' : 'bullets'
  return { kind, rest: m[2] ?? '' }
}

/** Puts the caret at the end of an element. */
export function caretToEnd(el: Node) {
  const doc = el.ownerDocument
  const sel = doc?.getSelection()
  if (!doc || !sel) return
  const range = doc.createRange()
  range.selectNodeContents(el)
  range.collapse(false)
  sel.removeAllRanges()
  sel.addRange(range)
}

/**
 * Applies a block shortcut to the paragraph it was typed in: "# Title"
 * becomes a heading, "- item" a bulleted list. A paragraph in a list item
 * isn't changed. Returns the element the caret goes in, or null.
 */
export function applyShortcut(block: HTMLElement): HTMLElement | null {
  if (block.tagName === 'LI' || /^H[1-6]$/.test(block.tagName)) return null
  const found = shortcutOf(block.textContent ?? '')
  if (!found) return null
  // Only the leading marker goes; the words (and their formatting) stay.
  stripLeading(block, (block.textContent ?? '').length - found.rest.length)
  const doc = block.ownerDocument
  let target: HTMLElement
  if (found.kind === 'heading' || found.kind === 'subheading') {
    target = doc.createElement(found.kind === 'heading' ? 'h2' : 'h3')
    target.append(...Array.from(block.childNodes))
    block.replaceWith(target)
  } else {
    const list = doc.createElement(found.kind === 'numbers' ? 'ol' : 'ul')
    target = doc.createElement('li')
    target.append(...Array.from(block.childNodes))
    list.append(target)
    block.replaceWith(list)
  }
  if (!target.childNodes.length) target.append(doc.createElement('br'))
  return target
}

/** Removes the first n characters of an element's text, across its text nodes. */
function stripLeading(el: HTMLElement, n: number) {
  const walker = el.ownerDocument.createTreeWalker(el, 4)
  let left = n
  for (let t = walker.nextNode() as Text | null; t && left > 0; t = walker.nextNode() as Text | null) {
    const len = t.data.length
    t.data = t.data.slice(Math.min(left, len))
    left -= len
  }
}

/**
 * "**words**" typed in a text node: the words between the last pair of **
 * become bold. Returns the bold element, or null.
 */
export function applyBold(text: Text): HTMLElement | null {
  const m = /\*\*([^*\n]+)\*\*$/.exec(text.data)
  if (!m) return null
  const doc = text.ownerDocument
  const start = m.index
  const after = text.splitText(start)
  after.data = after.data.slice(m[0].length)
  const strong = doc.createElement('strong')
  strong.textContent = m[1]!
  text.parentNode?.insertBefore(strong, after)
  if (!after.data) after.data = '\u00a0'
  return strong
}

/** "---" alone in a paragraph: a divider goes there (the paragraph is taken out). */
export function isDividerLine(block: HTMLElement): boolean {
  return block.tagName !== 'LI' && /^(-{3,}|\*{3,}|_{3,})$/.test((block.textContent ?? '').replace(/\u00a0/g, ' ').trim())
}
