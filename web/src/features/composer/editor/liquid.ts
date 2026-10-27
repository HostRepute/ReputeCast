import type { Component, Editor } from 'grapesjs'
import { t as tt } from '@/i18n'
import { mc } from '../canvas.messages'

/**
 * Liquid chips (CO-4, Composer board): on the canvas a personalization
 * token ({{ first_name | default: "there" }}) shows as a chip, "First name
 * · fallback “there”", that opens its fallback editor. The email keeps the
 * Liquid: a chip exports exactly its token.
 */

/** The canvas attribute that carries a chip's token (the RTE writes chips back through it). */
export const LIQUID_ATTR = 'data-rc-liquid'

/** One {{ … }} output. */
const TOKEN = /\{\{-?[^{}]*?-?\}\}/g

export interface ParsedToken {
  /** The variable ("first_name", "contact.plan"). */
  key: string
  /** The default filter's value, or undefined without one. */
  fallback?: string
  /** Other filters, as written ("capitalize", "truncate: 20"). */
  filters: string[]
}

/** Splits a filter chain on pipes outside quotes. */
function splitPipes(s: string): string[] {
  const out: string[] = []
  let cur = ''
  let quote: string | null = null
  for (const ch of s) {
    if (quote) {
      if (ch === quote) quote = null
      cur += ch
    } else if (ch === '"' || ch === "'") {
      quote = ch
      cur += ch
    } else if (ch === '|') {
      out.push(cur)
      cur = ''
    } else cur += ch
  }
  out.push(cur)
  return out.map((p) => p.trim())
}

/** Reads a token: {{ first_name | default: "there" }} → { key: first_name, fallback: there }. */
export function parseToken(token: string): ParsedToken {
  const inner = token.replace(/^\{\{-?/, '').replace(/-?\}\}$/, '')
  const [head = '', ...rest] = splitPipes(inner)
  const out: ParsedToken = { key: head.trim(), filters: [] }
  for (const f of rest) {
    const m = /^default\s*:\s*(?:"([^"]*)"|'([^']*)')\s*$/.exec(f)
    if (m) out.fallback = m[1] ?? m[2] ?? ''
    else if (f) out.filters.push(f)
  }
  return out
}

/** Writes a token back: the key, its other filters, then the fallback (none when empty). */
export function buildToken(p: ParsedToken): string {
  const parts = [p.key, ...p.filters]
  if (p.fallback !== undefined && p.fallback !== '') {
    // Liquid has no escapes in strings: the other quote wraps a fallback with quotes in it.
    const q = p.fallback.includes('"') ? "'" : '"'
    parts.push(`default: ${q}${p.fallback.replaceAll(q, '')}${q}`)
  }
  return `{{ ${parts.join(' | ')} }}`
}

/** Splits text into plain parts and tokens, in order. */
export function splitTokens(text: string): { text?: string; token?: string }[] {
  const out: { text?: string; token?: string }[] = []
  let last = 0
  for (const m of text.matchAll(TOKEN)) {
    if (m.index! > last) out.push({ text: text.slice(last, m.index) })
    out.push({ token: m[0] })
    last = m.index! + m[0].length
  }
  if (last < text.length) out.push({ text: text.slice(last) })
  return out
}

/** How a chip reads: the field's name, its fallback, and whether the field exists. */
export interface ChipLabel {
  label: string
  fallback?: string
  unknown: boolean
}

type LabelFn = (token: ParsedToken) => ChipLabel
const labels = new WeakMap<Editor, LabelFn>()

function labelOf(editor: Editor | undefined, token: string): ChipLabel {
  const p = parseToken(token)
  const fn = editor ? labels.get(editor) : undefined
  return fn ? fn(p) : { label: p.key, fallback: p.fallback, unknown: false }
}

/** How a token's chip reads in this editor (Write mode draws the same chips, CO-9). */
export function chipLabelOf(editor: Editor | null | undefined, token: string): ChipLabel {
  return labelOf(editor ?? undefined, token)
}

/** Components whose words may print tokens (raw HTML and tables stay as they are). */
const TEXT_TAGS = new Set(['mj-text', 'mj-button', 'mj-navbar-link', 'mj-social-element', 'mj-accordion-title', 'mj-accordion-text'])

/** True inside a text component (where tokens become chips). */
function inText(c: Component | undefined): boolean {
  let p = c?.parent()
  while (p) {
    if (TEXT_TAGS.has(String(p.get('tagName') ?? ''))) return true
    p = p.parent()
  }
  return false
}

/**
 * The chip component: a span on the canvas, its token on export. It can't
 * be selected, dragged or typed into; a click opens the fallback editor
 * (the editor event "rc:liquid:open").
 */
export function loadLiquidChips(editor: Editor) {
  editor.Components.addType('rc-liquid', {
    isComponent: (el: HTMLElement) =>
      el.nodeType === 1 && el.hasAttribute?.(LIQUID_ATTR) ? { type: 'rc-liquid', token: el.getAttribute(LIQUID_ATTR) ?? '', components: [] } : false,
    model: {
      defaults: {
        tagName: 'span',
        token: '',
        draggable: false,
        droppable: false,
        copyable: false,
        selectable: false,
        hoverable: false,
        layerable: false,
        highlightable: false,
        editable: false,
      },
      toHTML(this: Component) {
        return String(this.get('token') ?? '')
      },
    },
    view: {
      events: {
        click: 'onChipClick',
        keydown: 'onChipKey',
      } as never,
      onChipClick(this: { model: Component; em: { trigger(ev: string, ...a: unknown[]): void } }, e: Event) {
        e.preventDefault()
        e.stopPropagation()
        this.em.trigger('rc:liquid:open', this.model)
      },
      onChipKey(this: { model: Component; em: { trigger(ev: string, ...a: unknown[]): void } }, e: KeyboardEvent) {
        if (e.key !== 'Enter' && e.key !== ' ') return
        e.preventDefault()
        e.stopPropagation()
        this.em.trigger('rc:liquid:open', this.model)
      },
      onRender(this: { el: HTMLElement; model: Component; em: { getEditor(): Editor } }) {
        const token = String(this.model.get('token') ?? '')
        const l = labelOf(this.em.getEditor?.(), token)
        const el = this.el
        el.setAttribute(LIQUID_ATTR, token)
        el.setAttribute('contenteditable', 'false')
        // A button for the keyboard too: Enter or Space opens its fallback editor.
        el.setAttribute('role', 'button')
        el.setAttribute('tabindex', '0')
        el.setAttribute('aria-label', l.fallback ? tt(mc.chipAriaFallback, { label: l.label, fallback: l.fallback }) : tt(mc.chipAria, { label: l.label }))
        el.className = `rc-lq${l.unknown ? ' unknown' : ''}`
        el.title = token
        el.textContent = l.label
        if (l.fallback) {
          const fb = el.ownerDocument.createElement('span')
          fb.className = 'fb'
          fb.textContent = ` · “${l.fallback}”`
          el.appendChild(fb)
        }
      },
    },
  })
}

/**
 * Turns the tokens typed in text into chips (after a load, after a text's
 * editing ends). Not an edit: no undo step, no unsaved change.
 */
export function chipify(editor: Editor, root?: Component) {
  const scope = root ?? editor.getWrapper()
  if (!scope) return
  const found: Component[] = []
  const walk = (c: Component) => {
    if (c.get('type') === 'textnode') {
      if (String(c.get('content') ?? '').includes('{{') && inText(c)) found.push(c)
      return
    }
    c.components().forEach(walk)
  }
  walk(scope)
  if (!found.length) return
  editor.UndoManager.skip(() => {
    for (const tn of found) {
      const parent = tn.parent()
      if (!parent) continue
      const parts = splitTokens(String(tn.get('content') ?? ''))
      if (!parts.some((p) => p.token)) continue
      const at = parent.components().indexOf(tn)
      tn.remove({ avoidStore: true } as never)
      parent.components().add(
        parts.map((p) => (p.token ? { type: 'rc-liquid', token: p.token } : { type: 'textnode', content: p.text })),
        { at, avoidStore: true } as never,
      )
    }
  })
}

/** Sets how chips read (the workspace's fields), and redraws them. */
export function setChipLabels(editor: Editor, fn: LabelFn) {
  labels.set(editor, fn)
  editor
    .getWrapper()
    ?.findType('rc-liquid')
    .forEach((c: Component) => c.view?.render())
}

/** The canvas HTML a chip is inserted as while text is being edited (the RTE parses it back into a chip). */
export function chipHtml(token: string, label: ChipLabel): string {
  const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  const aria = label.fallback ? tt(mc.chipAriaFallback, { label: label.label, fallback: label.fallback }) : tt(mc.chipAria, { label: label.label })
  return `<span ${LIQUID_ATTR}="${esc(token)}" contenteditable="false" role="button" tabindex="0" aria-label="${esc(aria)}" class="rc-lq${label.unknown ? ' unknown' : ''}">${esc(label.label)}${label.fallback ? `<span class="fb"> · “${esc(label.fallback)}”</span>` : ''}</span>`
}
