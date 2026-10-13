/**
 * Helpers for MJML documents as text. The canvas (GrapesJS) only holds
 * <mjml><mj-body>…</mj-body></mjml>; the <mj-head> is kept here, next to it,
 * and joined back when the document is saved or compiled. Title and preview
 * text are written from the subject and preheader fields so they never drift.
 */

/** Tags allowed in the head that change how the body renders. */
const CANVAS_HEAD_TAGS = ['mj-attributes', 'mj-style', 'mj-font', 'mj-breakpoint', 'mj-html-attributes']

interface Span {
  tag: string
  start: number
  end: number
  inner: string
}

/** Finds top-level elements in `src` (quote and comment aware). */
export function topLevelElements(src: string): Span[] {
  const out: Span[] = []
  let i = 0
  while (i < src.length) {
    const lt = src.indexOf('<', i)
    if (lt < 0) break
    if (src.startsWith('<!--', lt)) {
      const end = src.indexOf('-->', lt + 4)
      i = end < 0 ? src.length : end + 3
      continue
    }
    const m = /^<([a-zA-Z][\w-]*)/.exec(src.slice(lt, lt + 64))
    if (!m) {
      i = lt + 1
      continue
    }
    const tag = m[1]!.toLowerCase()
    const openEnd = tagEnd(src, lt)
    if (openEnd < 0) break
    if (src[openEnd - 1] === '/') {
      out.push({ tag, start: lt, end: openEnd + 1, inner: '' })
      i = openEnd + 1
      continue
    }
    const close = findClose(src, tag, openEnd + 1)
    if (close < 0) {
      out.push({ tag, start: lt, end: src.length, inner: src.slice(openEnd + 1) })
      break
    }
    out.push({ tag, start: lt, end: close + tag.length + 3, inner: src.slice(openEnd + 1, close) })
    i = close + tag.length + 3
  }
  return out
}

/** Index of the `>` closing the tag that starts at `lt`, skipping quoted values. */
function tagEnd(src: string, lt: number): number {
  let q: string | null = null
  for (let j = lt + 1; j < src.length; j++) {
    const c = src[j]
    if (q) {
      if (c === q) q = null
    } else if (c === '"' || c === "'") q = c
    else if (c === '>') return j
  }
  return -1
}

/** Index of the matching `</tag` after `from`, counting nested tags of the same name. */
function findClose(src: string, tag: string, from: number): number {
  let depth = 1
  const re = new RegExp(`<(/?)${tag}(?=[\\s/>])`, 'gi')
  re.lastIndex = from
  for (let m = re.exec(src); m; m = re.exec(src)) {
    if (m[1]) {
      depth--
      if (depth === 0) return m.index
    } else {
      const end = tagEnd(src, m.index)
      if (end > 0 && src[end - 1] !== '/') depth++
    }
  }
  return -1
}

/**
 * Rewrites self-closing MJML tags (`<mj-image … />`) as open/close pairs.
 * GrapesJS parses MJML as HTML, where `/>` doesn't close a custom element and
 * the following siblings would end up nested inside it.
 */
export function expandSelfClosing(src: string): string {
  let out = ''
  let i = 0
  while (i < src.length) {
    const lt = src.indexOf('<', i)
    if (lt < 0) {
      out += src.slice(i)
      break
    }
    out += src.slice(i, lt)
    if (src.startsWith('<!--', lt)) {
      const end = src.indexOf('-->', lt + 4)
      const stop = end < 0 ? src.length : end + 3
      out += src.slice(lt, stop)
      i = stop
      continue
    }
    const m = /^<(mj-[\w-]+|mjml)/i.exec(src.slice(lt, lt + 64))
    const end = m ? tagEnd(src, lt) : -1
    if (m && end > 0 && src[end - 1] === '/') {
      const body = src.slice(lt, end - 1).replace(/\s+$/, '')
      out += `${body}></${m[1]!.toLowerCase()}>`
      i = end + 1
      continue
    }
    out += '<'
    i = lt + 1
  }
  return out
}

export interface MjmlParts {
  /** Inner MJML of <mj-head> (may be empty). */
  head: string
  /** The document without its head: <mjml …><mj-body …>…</mj-body></mjml>. */
  body: string
}

const EMPTY_BODY = '<mjml><mj-body width="600px"></mj-body></mjml>'

/** Splits a document into head and the body the canvas loads. */
export function splitMjml(mjml: string): MjmlParts {
  const src = expandSelfClosing(mjml.trim())
  if (!src) return { head: '', body: EMPTY_BODY }
  const root = topLevelElements(src).find((e) => e.tag === 'mjml')
  if (!root) return { head: '', body: `<mjml><mj-body width="600px">${src}</mj-body></mjml>` }
  const open = src.slice(root.start, tagEnd(src, root.start) + 1)
  const children = topLevelElements(root.inner)
  const head = children.find((c) => c.tag === 'mj-head')
  const body = children.find((c) => c.tag === 'mj-body')
  const bodySrc = body ? root.inner.slice(body.start, body.end) : '<mj-body width="600px"></mj-body>'
  return { head: head ? head.inner.trim() : '', body: `${open}${bodySrc}</mjml>` }
}

/** Joins the stored head back into the canvas export. */
export function joinMjml(head: string, body: string): string {
  const trimmed = body.trim()
  const h = head.trim() ? `\n  <mj-head>\n${indent(head.trim(), '    ')}\n  </mj-head>` : ''
  const m = /^<mjml(\s[^>]*)?>/.exec(trimmed)
  if (!m) return `<mjml>${h}\n${trimmed}\n</mjml>`
  return `${m[0]}${h}\n  ${trimmed.slice(m[0].length).trim()}`
}

function indent(s: string, pad: string): string {
  return s
    .split('\n')
    .map((l) => (l.trim() ? pad + l.trimStart() : l))
    .join('\n')
}

function escapeText(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

/** Writes <mj-title> and <mj-preview> from the subject and preheader. */
export function withHeadMeta(head: string, meta: { title: string; preview: string }): string {
  let out = head
  for (const [tag, value] of [
    ['mj-title', meta.title],
    ['mj-preview', meta.preview],
  ] as const) {
    const el = topLevelElements(out).find((e) => e.tag === tag)
    const next = value ? `<${tag}>${escapeText(value)}</${tag}>` : ''
    if (el) out = out.slice(0, el.start) + next + out.slice(el.end)
    else if (next) out = tag === 'mj-title' ? `${next}\n${out}` : insertAfter(out, 'mj-title', next)
  }
  return out
    .split('\n')
    .filter((l) => l.trim())
    .join('\n')
}

function insertAfter(head: string, tag: string, html: string): string {
  const el = topLevelElements(head).find((e) => e.tag === tag)
  if (!el) return `${html}\n${head}`
  return `${head.slice(0, el.end)}\n${html}${head.slice(el.end)}`
}

/** Reads <mj-title> and <mj-preview> text from a head. */
export function headMeta(head: string): { title?: string; preview?: string } {
  const els = topLevelElements(head)
  const text = (tag: string) => {
    const el = els.find((e) => e.tag === tag)
    return el ? decode(el.inner.trim()) : undefined
  }
  return { title: text('mj-title'), preview: text('mj-preview') }
}

function decode(s: string): string {
  return s.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"').replace(/&#39;/g, "'").replace(/&amp;/g, '&')
}

/** The part of the head that affects rendering, injected into canvas renders. */
export function canvasHead(head: string): string {
  return topLevelElements(head)
    .filter((e) => CANVAS_HEAD_TAGS.includes(e.tag))
    .map((e) => head.slice(e.start, e.end))
    .join('')
}

/** Elements whose content is text or HTML: kept on one line by formatMjml. */
const INLINE_CONTENT = new Set(['mj-text', 'mj-button', 'mj-raw', 'mj-table', 'mj-title', 'mj-preview', 'mj-style', 'mj-navbar-link', 'mj-social-element', 'mj-accordion-title', 'mj-accordion-text'])

/**
 * Re-indents an MJML document: one element per line, two spaces per level.
 * The content of text-like elements (mj-text, mj-button, mj-raw…) and
 * comments is kept as written.
 */
export function formatMjml(src: string): string {
  const out: string[] = []
  const walk = (s: string, depth: number) => {
    let i = 0
    while (i < s.length) {
      const lt = s.indexOf('<', i)
      const text = (lt < 0 ? s.slice(i) : s.slice(i, lt)).trim()
      if (text) out.push('  '.repeat(depth) + text)
      if (lt < 0) break
      if (s.startsWith('<!--', lt)) {
        const end = s.indexOf('-->', lt + 4)
        const stop = end < 0 ? s.length : end + 3
        out.push('  '.repeat(depth) + s.slice(lt, stop).trim())
        i = stop
        continue
      }
      const el = topLevelElements(s.slice(lt))[0]
      if (!el || el.start !== 0) {
        out.push('  '.repeat(depth) + s.slice(lt).trim())
        break
      }
      const whole = s.slice(lt, lt + el.end)
      const open = whole.slice(0, tagEnd(whole, 0) + 1)
      if (open.endsWith('/>') || INLINE_CONTENT.has(el.tag) || !/<[a-z]/i.test(el.inner)) {
        out.push('  '.repeat(depth) + whole.replace(/\s*\n\s*/g, el.tag === 'mj-style' || el.tag === 'mj-raw' ? '\n' + '  '.repeat(depth + 1) : ' ').trim())
      } else {
        out.push('  '.repeat(depth) + open)
        walk(el.inner, depth + 1)
        out.push('  '.repeat(depth) + `</${el.tag}>`)
      }
      i = lt + el.end
    }
  }
  walk(src.trim(), 0)
  return out.join('\n') + '\n'
}

/** Elements whose content MJML doesn't parse as MJML (it may hold any HTML). */
const ENDING_TAGS = new Set(['mj-text', 'mj-button', 'mj-raw', 'mj-table', 'mj-style', 'mj-title', 'mj-preview', 'mj-navbar-link', 'mj-social-element', 'mj-accordion-title', 'mj-accordion-text'])

/**
 * Finds the first structural problem in an MJML document: an mj-* element
 * that is never closed, or a closing tag that doesn't match. HTML inside
 * text-like elements isn't checked (MJML passes it through).
 */
export function checkMjmlTags(src: string): { line: number; message: string } | null {
  const lineAt = (i: number) => src.slice(0, i).split('\n').length
  const stack: { tag: string; at: number }[] = []
  const re = /<!--[\s\S]*?-->|<(\/?)(mjml|mj-[\w-]+)\b(?:[^>"']|"[^"]*"|'[^']*')*?(\/?)>/g
  for (let m = re.exec(src); m; m = re.exec(src)) {
    if (m[0].startsWith('<!--')) continue
    const [, close, tag, selfClose] = m as unknown as [string, string, string, string]
    if (selfClose) continue
    if (!close) {
      stack.push({ tag: tag!, at: m.index })
      if (ENDING_TAGS.has(tag!)) {
        const end = src.indexOf(`</${tag}>`, re.lastIndex)
        if (end < 0) return { line: lineAt(m.index), message: `unclosed <${tag}>` }
        re.lastIndex = end
      }
      continue
    }
    const top = stack.pop()
    if (!top) return { line: lineAt(m.index), message: `</${tag}> has no opening tag` }
    if (top.tag !== tag) return { line: lineAt(top.at), message: `unclosed <${top.tag}>` }
  }
  const open = stack.pop()
  return open ? { line: lineAt(open.at), message: `unclosed <${open.tag}>` } : null
}

const VISIBILITY_MARK = '/* rc:visibility */'

/**
 * Adds or removes the media-query rule behind "Hide on mobile" and "Hide on
 * desktop" (css-class rc-hide-mobile / rc-hide-desktop). MJML stacks columns
 * below 480 px, so that is the breakpoint here too. Clients without media
 * query support (Outlook for Windows) show both.
 */
export function withVisibilityStyles(head: string, used: { mobile: boolean; desktop: boolean }): string {
  let out = head
  const el = topLevelElements(out).find((e) => e.tag === 'mj-style' && e.inner.includes(VISIBILITY_MARK))
  if (el) out = out.slice(0, el.start) + out.slice(el.end)
  if (used.mobile || used.desktop) {
    const rules = [VISIBILITY_MARK]
    if (used.mobile) rules.push('@media only screen and (max-width:480px) { .rc-hide-mobile { display:none !important; } }')
    if (used.desktop) rules.push('@media only screen and (min-width:481px) { .rc-hide-desktop { display:none !important; } }')
    out = `${out.trim()}\n<mj-style>\n${rules.join('\n')}\n</mj-style>`
  }
  return out
    .split('\n')
    .filter((l) => l.trim())
    .join('\n')
}

/**
 * CSS of the head's <mj-style> elements, for the canvas (media queries
 * included). The visibility rule is left out: the canvas dims hidden
 * components instead of hiding them, so they stay selectable.
 */
export function headCss(head: string): string {
  return topLevelElements(head)
    .filter((e) => e.tag === 'mj-style' && !e.inner.includes(VISIBILITY_MARK))
    .map((e) => e.inner)
    .join('\n')
}

/** True when the body has no sections yet (the empty-canvas state). */
export function isEmptyBody(mjml: string): boolean {
  if (!mjml.trim()) return true
  const { body } = splitMjml(mjml)
  const root = topLevelElements(body).find((e) => e.tag === 'mjml')
  const b = root && topLevelElements(root.inner).find((e) => e.tag === 'mj-body')
  return !b || topLevelElements(b.inner).length === 0
}
