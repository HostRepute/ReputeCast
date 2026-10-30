/**
 * Text styles of the text toolbar (CO-8), named as in Write mode's
 * formatting bar: Paragraph, Heading, Subheading and Small print. They are
 * the HTML an mj-text holds as it is: <p>, <h2> (an <h1> already there
 * counts as a heading and stays one), <h3>, and a paragraph in <small>.
 */
export type TextStyle = 'p' | 'heading' | 'subheading' | 'small'

export const TEXT_STYLES: TextStyle[] = ['p', 'heading', 'subheading', 'small']

/** What the toolbar formats with: GrapesJS's rich text editor on the edited element. */
export interface TextEditing {
  exec(cmd: string, value?: string): void
  selection(): Selection | null
  el: HTMLElement
  doc: Document
}

const BLOCK = /^(P|H[1-6]|LI|DIV|BLOCKQUOTE|PRE)$/

/** The block (paragraph, heading, list item) a node is in, inside the edited element. */
export function blockAt(node: Node | null | undefined, root: HTMLElement): HTMLElement | null {
  let n: Node | null | undefined = node
  while (n && n !== root) {
    if (n.nodeType === 1 && BLOCK.test((n as HTMLElement).tagName)) return n as HTMLElement
    n = n.parentNode
  }
  return null
}

/** The block's only content is one <small>. */
function smallOf(block: HTMLElement): HTMLElement | null {
  const kids = Array.from(block.childNodes).filter((n) => !(n.nodeType === 3 && !(n.textContent ?? '').trim()))
  return kids.length === 1 && kids[0]!.nodeName === 'SMALL' ? (kids[0] as HTMLElement) : null
}

/** The text style of a block (a paragraph when there is no block). */
export function textStyleOf(block: HTMLElement | null): TextStyle {
  if (!block) return 'p'
  const tag = block.tagName
  if (tag === 'H1' || tag === 'H2') return 'heading'
  if (/^H[3-6]$/.test(tag)) return 'subheading'
  return smallOf(block) ? 'small' : 'p'
}

/** The text style where the caret is. */
export function currentTextStyle(rte: TextEditing): TextStyle {
  return textStyleOf(blockAt(rte.selection()?.anchorNode, rte.el))
}

/**
 * Gives the block the caret is in a text style. The words stay; a list item
 * becomes a paragraph or heading of its own (as formatBlock does).
 */
export function applyTextStyle(rte: TextEditing, style: TextStyle): void {
  let block = blockAt(rte.selection()?.anchorNode, rte.el)
  const isHeading = block && (block.tagName === 'H1' || block.tagName === 'H2')
  if (!(style === 'heading' && isHeading)) {
    const tag = style === 'heading' ? 'h2' : style === 'subheading' ? 'h3' : 'p'
    if (!block || block.tagName.toLowerCase() !== tag) {
      rte.exec('formatBlock', `<${tag}>`)
      block = blockAt(rte.selection()?.anchorNode, rte.el)
    }
  }
  if (!block) return
  const small = smallOf(block)
  if (style === 'small' && !small) {
    const s = rte.doc.createElement('small')
    s.append(...Array.from(block.childNodes))
    block.append(s)
  } else if (style !== 'small' && small) {
    small.replaceWith(...Array.from(small.childNodes))
  } else {
    return
  }
  // A change made on the elements (not by execCommand) is an edit too.
  const view = rte.doc.defaultView
  rte.el.dispatchEvent(new (view?.Event ?? Event)('input', { bubbles: true }))
}
