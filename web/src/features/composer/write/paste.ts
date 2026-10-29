/**
 * Paste cleanup in Write mode (CO-9, ComposerWrite: "Pasted from Google
 * Docs · kept headings, links and lists · removed 14 inline styles"): what
 * comes in keeps its structure (paragraphs, headings, lists, links, bold
 * and italics) and nothing else, so the email's own styles apply.
 */

import { parseInert } from '../inert'

export interface CleanPaste {
  html: string
  /** Where it came from, when it says so: "Google Docs", "Word". */
  source: string
  kept: ('headings' | 'links' | 'lists')[]
  /** Inline styles, classes and fonts taken out. */
  removed: number
  /** Images left out (they go in as Image blocks). */
  images: number
}

const BLOCKS = new Set(['P', 'DIV', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'UL', 'OL', 'LI', 'BLOCKQUOTE', 'PRE', 'TABLE', 'TR'])
const DROP = new Set(['SCRIPT', 'STYLE', 'META', 'LINK', 'TITLE', 'HEAD', 'OBJECT', 'IFRAME', 'SVG', 'NOSCRIPT', 'TEMPLATE', 'BUTTON', 'INPUT', 'SELECT', 'TEXTAREA', 'FORM'])

function safeHref(href: string): string {
  const h = href.trim()
  if (/^\{\{.*\}\}$/.test(h)) return h
  try {
    const u = new URL(h)
    return u.protocol === 'http:' || u.protocol === 'https:' || u.protocol === 'mailto:' ? u.href : ''
  } catch {
    return ''
  }
}

/** Plain text to paragraphs: blank lines split them, single breaks stay. */
export function textToHtml(text: string): string {
  const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return text
    .replace(/\r\n?/g, '\n')
    .split(/\n{2,}/)
    .map((p) => p.trim())
    .filter(Boolean)
    .map((p) => `<p>${esc(p).replace(/\n/g, '<br>')}</p>`)
    .join('')
}

/** Where a paste came from, by the marks each editor leaves in its HTML. */
function pasteSource(html: string): string {
  if (/docs-internal-guid/.test(html)) return 'Google Docs'
  if (/<!--\s*notionvc:/i.test(html)) return 'Notion'
  if (/urn:schemas-microsoft-com:office|class="?Mso/i.test(html)) return 'Word'
  return ''
}

export function cleanPaste(html: string, doc: Document = document): CleanPaste {
  const source = pasteSource(html)
  // Read in an inert document: a paste from any web page must not load or
  // run anything. Comments (Word's conditional ones among them) are left to
  // the parser: they become comment nodes, which the walk below skips.
  const box = parseInert(html)
  const out: CleanPaste = { html: '', source, kept: [], removed: 0, images: 0 }
  const kept = new Set<CleanPaste['kept'][number]>()

  const clean = (node: Node, into: Node) => {
    if (node.nodeType === 3) {
      into.appendChild(doc.createTextNode(node.textContent ?? ''))
      return
    }
    if (node.nodeType !== 1) return
    const el = node as HTMLElement
    const tag = el.tagName
    if (DROP.has(tag)) return
    if (tag === 'IMG') {
      out.images++
      return
    }
    if (el.hasAttribute('style')) out.removed++
    if (el.hasAttribute('class')) out.removed++
    if (tag === 'FONT') out.removed++
    const style = (el.getAttribute('style') ?? '').replace(/\s/g, '').toLowerCase()
    let wrap: HTMLElement | null = null
    switch (tag) {
      case 'H1':
      case 'H2':
        wrap = doc.createElement('h2')
        kept.add('headings')
        break
      case 'H3':
      case 'H4':
      case 'H5':
      case 'H6':
        wrap = doc.createElement('h3')
        kept.add('headings')
        break
      case 'UL':
      case 'OL':
        wrap = doc.createElement(tag.toLowerCase())
        kept.add('lists')
        break
      case 'LI':
        wrap = doc.createElement('li')
        break
      case 'P':
      case 'DIV':
      case 'BLOCKQUOTE':
      case 'PRE':
      case 'TR':
        wrap = doc.createElement('p')
        break
      case 'BR':
        into.appendChild(doc.createElement('br'))
        return
      case 'A': {
        const href = safeHref(el.getAttribute('href') ?? '')
        if (href) {
          wrap = doc.createElement('a')
          wrap.setAttribute('href', href)
          kept.add('links')
        }
        break
      }
      case 'STRONG':
      case 'B':
        // Google Docs wraps everything in <b style="font-weight:normal">.
        if (!/font-weight:(normal|400)/.test(style)) wrap = doc.createElement('strong')
        break
      case 'EM':
      case 'I':
        wrap = doc.createElement('em')
        break
      default:
        // Spans and the rest: their bold or italic style is kept as markup.
        if (/font-weight:(bold|[6-9]00)/.test(style)) wrap = doc.createElement('strong')
        else if (/font-style:italic/.test(style)) wrap = doc.createElement('em')
    }
    const target = wrap ?? into
    for (const child of Array.from(el.childNodes)) clean(child, target)
    if (wrap) {
      // A block inside a paragraph (a <div> in a <p>) ends up beside it.
      if (BLOCKS.has(tag) && into.nodeName === 'P' && into.parentNode) into.parentNode.appendChild(wrap)
      else into.appendChild(wrap)
    }
  }
  const root = doc.createElement('div')
  for (const child of Array.from(box.childNodes)) clean(child, root)
  // Empty paragraphs and list items go; loose words at the top get a paragraph.
  root.querySelectorAll('p, li, h2, h3, strong, em, a').forEach((el) => {
    if (!(el.textContent ?? '').trim() && !el.querySelector('br')) el.remove()
  })
  const blocks = doc.createElement('div')
  let loose: HTMLElement | null = null
  for (const child of Array.from(root.childNodes)) {
    const isBlock = child.nodeType === 1 && /^(P|H2|H3|UL|OL)$/.test((child as HTMLElement).tagName)
    // Notion copies each bullet as a list of its own: one list goes in.
    const prev = blocks.lastChild as HTMLElement | null
    if (isBlock && /^(UL|OL)$/.test((child as HTMLElement).tagName) && !loose && prev?.nodeType === 1 && prev.tagName === (child as HTMLElement).tagName) {
      for (const li of Array.from(child.childNodes)) prev.appendChild(li)
      continue
    }
    if (isBlock) {
      loose = null
      blocks.appendChild(child)
    } else if (child.nodeType === 3 && !(child.textContent ?? '').trim() && !loose) {
      continue
    } else {
      if (!loose) {
        loose = doc.createElement('p')
        blocks.appendChild(loose)
      }
      loose.appendChild(child)
    }
  }
  // A single paragraph goes in where the caret is, without a paragraph of its own.
  const only = blocks.childNodes.length === 1 ? (blocks.firstChild as HTMLElement) : null
  out.html = only && only.tagName === 'P' ? only.innerHTML : blocks.innerHTML
  out.kept = (['headings', 'links', 'lists'] as const).filter((k) => kept.has(k))
  return out
}
