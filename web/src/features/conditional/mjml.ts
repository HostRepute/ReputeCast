/**
 * ReputeCast's own attributes in MJML (design expansion, X11), as the
 * server reads them (internal/mjml/conditional.go, internal/visibility):
 *
 *   rc-visible  a block's Visibility rule (JSON)
 *   rc-name     its name in the layer tree ("Beta corner")
 *   rc-poll     a Poll block's settings (JSON, on its mj-section)
 *   rc-omit     a block the canvas shows and the email leaves out (a Sponsor
 *               block without a booking)
 *
 * Before the composer compiles an email they come out (mjml-browser would
 * report them as unknown attributes) and every conditional block is wrapped
 * in two markers, <mj-raw><!--rc:if ID--></mj-raw> … <mj-raw><!--rc:fi ID--></mj-raw>,
 * so the compiled HTML of the whole email can be cut down to each
 * recipient's version without compiling again. Keep this file in step with
 * the Go side: both must mark the same blocks.
 */

export const ATTR_VISIBLE = 'rc-visible'
export const ATTR_NAME = 'rc-name'
export const ATTR_POLL = 'rc-poll'
export const ATTR_OMIT = 'rc-omit'

export interface MjNode {
  tag: string
  attrs: Record<string, string>
  start: number
  end: number
  children: MjNode[]
  parent: MjNode | null
}

/** Elements whose content is HTML, not MJML children. */
const ENDING = new Set([
  'mj-text',
  'mj-button',
  'mj-raw',
  'mj-table',
  'mj-navbar-link',
  'mj-social-element',
  'mj-title',
  'mj-preview',
  'mj-style',
  'mj-accordion-title',
  'mj-accordion-text',
])

/** Elements that may be left unclosed (closed by their parent or next sibling). */
const VOID = new Set(['mj-image', 'mj-divider', 'mj-spacer', 'mj-carousel-image', 'mj-breakpoint', 'mj-font', 'mj-all', 'mj-class', 'mj-html-attribute', 'mj-include'])

/** Elements a conditional block may sit in (they take <mj-raw> children). */
const RAW_PARENTS = new Set(['mj-body', 'mj-wrapper', 'mj-section', 'mj-group', 'mj-column', 'mj-hero'])

function decode(v: string): string {
  return v.replace(/&(quot|amp|lt|gt|#39|#34|apos);/g, (m, e: string) => ({ quot: '"', amp: '&', lt: '<', gt: '>', '#39': "'", '#34': '"', apos: "'" })[e] ?? m)
}

const NAME = /[A-Za-z0-9_:-]/

/**
 * Parses MJML leniently into a tree of elements with their source offsets.
 * Returns null when the document doesn't parse (the compiler reports it).
 */
export function parseMjml(src: string): MjNode | null {
  const root: MjNode = { tag: '#document', attrs: {}, start: 0, end: src.length, children: [], parent: null }
  const stack: MjNode[] = [root]
  let pos = 0
  while (pos < src.length) {
    const lt = src.indexOf('<', pos)
    if (lt < 0) break
    if (src.startsWith('<!--', lt)) {
      const end = src.indexOf('-->', lt + 4)
      if (end < 0) return null
      pos = end + 3
      continue
    }
    if (src.startsWith('</', lt)) {
      const gt = src.indexOf('>', lt)
      if (gt < 0) return null
      const tag = src.slice(lt + 2, gt).trim().toLowerCase()
      pos = gt + 1
      while (stack.length > 1 && stack[stack.length - 1]!.tag !== tag && VOID.has(stack[stack.length - 1]!.tag)) {
        stack.pop()!.end = lt
      }
      const top = stack[stack.length - 1]!
      if (top.tag !== tag) return null
      top.end = pos
      stack.pop()
      continue
    }
    if (src[lt + 1] === '!' || src[lt + 1] === '?') {
      const gt = src.indexOf('>', lt)
      if (gt < 0) break
      pos = gt + 1
      continue
    }
    // An opening tag.
    let i = lt + 1
    while (i < src.length && NAME.test(src[i]!)) i++
    if (i === lt + 1) {
      pos = lt + 1
      continue
    }
    const node: MjNode = { tag: src.slice(lt + 1, i).toLowerCase(), attrs: {}, start: lt, end: lt, children: [], parent: null }
    let selfClosing = false
    for (;;) {
      while (i < src.length && /\s/.test(src[i]!)) i++
      if (i >= src.length) return null
      if (src[i] === '>') {
        i++
        break
      }
      if (src[i] === '/' && src[i + 1] === '>') {
        i += 2
        selfClosing = true
        break
      }
      const ns = i
      while (i < src.length && !/\s/.test(src[i]!) && src[i] !== '=' && src[i] !== '>' && !(src[i] === '/' && src[i + 1] === '>')) i++
      const name = src.slice(ns, i).toLowerCase()
      while (i < src.length && /\s/.test(src[i]!)) i++
      let val = ''
      if (src[i] === '=') {
        i++
        while (i < src.length && /\s/.test(src[i]!)) i++
        const q = src[i]
        if (q === '"' || q === "'") {
          const end = src.indexOf(q, i + 1)
          if (end < 0) return null
          val = src.slice(i + 1, end)
          i = end + 1
        } else {
          const vs = i
          while (i < src.length && !/\s/.test(src[i]!) && src[i] !== '>') i++
          val = src.slice(vs, i)
        }
      }
      if (name) node.attrs[name] = decode(val)
    }
    let parent = stack[stack.length - 1]!
    if (VOID.has(parent.tag) && stack.length > 1) {
      stack.pop()!.end = lt
      parent = stack[stack.length - 1]!
    }
    node.parent = parent
    parent.children.push(node)
    if (selfClosing) {
      node.end = i
      pos = i
      continue
    }
    if (ENDING.has(node.tag)) {
      const close = src.toLowerCase().indexOf(`</${node.tag}`, i)
      if (close < 0) return null
      const gt = src.indexOf('>', close)
      if (gt < 0) return null
      node.end = gt + 1
      pos = gt + 1
      continue
    }
    stack.push(node)
    pos = i
  }
  while (stack.length > 1) {
    const top = stack.pop()!
    if (!VOID.has(top.tag)) return null
    top.end = src.length
  }
  return root
}

const ID = /^[a-z0-9]{1,24}$/

/** The id of a conditional element: one whose rule doesn't show it to everyone; "" otherwise. */
export function conditionalId(attrs: Record<string, string>): string {
  const raw = attrs[ATTR_VISIBLE]
  if (!raw || !raw.trim()) return ''
  try {
    const h = JSON.parse(raw) as { id?: string; show?: string }
    if (!h.id || !ID.test(h.id) || !h.show || h.show === 'everyone') return ''
    return h.id
  } catch {
    return ''
  }
}

const RC_ATTR = /\s+rc-[a-z-]+\s*=\s*("[^"]*"|'[^']*'|[^\s>"']+)/g

/** Removes ReputeCast's attributes from MJML (what the canvas renders). */
export function stripRcAttrs(src: string): string {
  return src.includes('rc-') ? src.replace(RC_ATTR, '') : src
}

/**
 * What the compiler gets: rc-* attributes out, blocks marked rc-omit left
 * out, conditional blocks wrapped in their markers (each id once, the first
 * time). A document that doesn't parse is returned with the attributes
 * stripped.
 */
export function prepareCompile(src: string): string {
  if (!src.includes('rc-')) return src
  const root = parseMjml(src)
  if (!root) return stripRcAttrs(src)
  type Edit = { start: number; end: number; text: string; order: number }
  const edits: Edit[] = []
  const seen = new Set<string>()
  const walk = (n: MjNode) => {
    // Shown on the canvas only: left out of the email.
    const omit = n.attrs[ATTR_OMIT]
    if (n.tag.startsWith('mj-') && n.end > n.start && n.parent && omit !== undefined && omit !== '' && omit !== '0' && omit !== 'false') {
      edits.push({ start: n.start, end: n.end, text: '', order: 1 })
      return
    }
    if (n.tag.startsWith('mj-') && n.end > n.start) {
      const open = src.indexOf('>', n.start)
      if (open > 0 && open < n.end) {
        const tag = src.slice(n.start, open + 1)
        if (tag.includes('rc-')) {
          const cleaned = tag.replace(RC_ATTR, '')
          if (cleaned !== tag) edits.push({ start: n.start, end: open + 1, text: cleaned, order: 1 })
        }
      }
      const id = conditionalId(n.attrs)
      if (id && n.parent && RAW_PARENTS.has(n.parent.tag) && !seen.has(id)) {
        seen.add(id)
        edits.push({ start: n.start, end: n.start, text: `<mj-raw><!--rc:if ${id}--></mj-raw>`, order: 0 })
        edits.push({ start: n.end, end: n.end, text: `<mj-raw><!--rc:fi ${id}--></mj-raw>`, order: 2 })
      }
    }
    n.children.forEach(walk)
  }
  walk(root)
  if (!edits.length) return src
  edits.sort((a, b) => a.start - b.start || a.order - b.order)
  let out = ''
  let last = 0
  for (const e of edits) {
    if (e.start < last) continue
    out += src.slice(last, e.start) + e.text
    last = e.end
  }
  return out + src.slice(last)
}

const MARKER = /<!--rc:(if|fi) ([a-z0-9]{1,24})-->/g

/** Compiled HTML without the markers: every block shown. */
export function stripMarkers(html: string): string {
  return html.includes('<!--rc:') ? html.replace(MARKER, '') : html
}

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

/** The preview's line where a hidden block was ("Hidden: Beta corner · Sam Rivera isn't on Beta Program"). */
export function whyLine(text: string): string {
  return (
    `<div class="rc-why" style="margin:6px 24px;padding:6px 12px;border:1px dashed #93C5FD;border-radius:6px;background:#EFF6FF;color:#1E3A8A;` +
    `font-family:'IBM Plex Sans',system-ui,sans-serif;font-size:12px;font-weight:500;line-height:1.5;text-align:left">` +
    `<span aria-hidden="true">⊘ </span>${esc(text)}</div>`
  )
}

/** The preview's box where an empty version has nothing between the header and the footer. */
export function emptyLine(text: string): string {
  return (
    `<div class="rc-why-empty" style="margin:12px 24px;padding:28px 12px;border:2px dashed #FCA5A5;border-radius:6px;background:#FFF5F5;color:#991B1B;` +
    `font-family:'IBM Plex Sans',system-ui,sans-serif;font-size:13px;font-weight:600;text-align:center">${esc(text)}</div>`
  )
}

/**
 * Cuts the hidden blocks out of compiled HTML (and every marker): the email
 * one recipient gets. `line` gives a hidden block's "Show why" line.
 */
export function cutHidden(html: string, hidden: ReadonlySet<string>, line?: (id: string) => string | null): string {
  if (!html.includes('<!--rc:')) return html
  let out = ''
  let i = 0
  for (;;) {
    const j = html.indexOf('<!--rc:', i)
    if (j < 0) {
      out += html.slice(i)
      break
    }
    out += html.slice(i, j)
    const end = html.indexOf('-->', j)
    if (end < 0) {
      out += html.slice(j)
      break
    }
    const tag = html.slice(j, end + 3)
    const m = /^<!--rc:(if|fi) ([a-z0-9]{1,24})-->$/.exec(tag)
    if (!m) {
      out += tag
      i = end + 3
      continue
    }
    const [, kind, id] = m as unknown as [string, string, string]
    if (kind !== 'if' || !hidden.has(id)) {
      i = end + 3
      continue
    }
    const closing = `<!--rc:fi ${id}-->`
    const k = html.indexOf(closing, end + 3)
    if (k < 0) {
      i = end + 3
      continue
    }
    const text = line?.(id)
    if (text) out += whyLine(text)
    i = k + closing.length
  }
  return out
}
