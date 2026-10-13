import type { Component, Editor } from 'grapesjs'
import type { ComponentPluginOptions } from 'grapesjs-mjml'

export interface MjmlCompileError {
  line: number
  message: string
  tagName: string
  formattedMessage?: string
}

export interface MjmlCompileResult {
  html: string
  errors: MjmlCompileError[]
}

export type MjmlCompile = (mjml: string) => MjmlCompileResult

type Parser = (mjml: string, options?: Record<string, unknown>) => MjmlCompileResult
type Attrs = Record<string, unknown>

const SIDES = ['top', 'right', 'bottom', 'left'] as const

/** Escapes an attribute value for the MJML source (grapesjs-mjml writes values verbatim). */
export function escapeAttr(v: unknown): string {
  return String(v).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function escapeAll(attrs: Attrs): Attrs {
  const out: Attrs = {}
  for (const [k, v] of Object.entries(attrs)) out[k] = v === undefined || v === null || v === '' ? v : escapeAttr(v)
  return out
}

/** "16px 24px" → { top: 16px, right: 24px, bottom: 16px, left: 24px } */
export function expandBox(v: string): Record<(typeof SIDES)[number], string> {
  const p = v.trim().split(/\s+/)
  const [t, r = t, b = t, l = r] = p as [string, string?, string?, string?]
  return { top: t, right: r!, bottom: b!, left: l! }
}

/** Shortest CSS box shorthand: 16px 24px 16px 24px → 16px 24px. */
export function collapseBox(t: string, r: string, b: string, l: string): string {
  if (r === l) {
    if (t === b) return t === r ? t : `${t} ${r}`
    return `${t} ${r} ${b}`
  }
  return `${t} ${r} ${b} ${l}`
}

/**
 * grapesjs-mjml seeds every component with its default paddings as
 * longhands (padding-top: 20px…). A `padding="16px 24px"` shorthand in the
 * source is then overridden by those longhands when the canvas renders, so
 * the canvas and the sent email differ. We expand the shorthand into
 * longhands when a component is created (the inspector edits longhands) and
 * write the shortest shorthand back when MJML is exported.
 */
function expandPadding(c: Component) {
  const defaults = (c.get('style-default') ?? {}) as Attrs
  if (!('padding-top' in defaults)) return
  const attrs = { ...(c.get('attributes') as Attrs) }
  const short = attrs.padding
  if (typeof short !== 'string' || !short.trim()) return
  const box = expandBox(short)
  for (const s of SIDES) {
    const key = `padding-${s}`
    if (attrs[key] === undefined || attrs[key] === '') attrs[key] = box[s]
  }
  delete attrs.padding
  c.set('attributes', attrs, { silent: true })
}

function collapsePadding(c: Component, out: Attrs): Attrs {
  const defaults = (c.get('style-default') ?? {}) as Attrs
  const all = (c.get('attributes') ?? {}) as Attrs
  if (!SIDES.some((s) => `padding-${s}` in all)) return out
  const side = (s: string) => String(all[`padding-${s}`] ?? defaults[`padding-${s}`] ?? '0px')
  const def = (s: string) => String(defaults[`padding-${s}`] ?? '0px')
  const next = { ...out }
  for (const s of SIDES) delete next[`padding-${s}`]
  const value = collapseBox(side('top'), side('right'), side('bottom'), side('left'))
  const fallback = collapseBox(def('top'), def('right'), def('bottom'), def('left'))
  if (value !== fallback) next.padding = escapeAttr(value)
  return next
}

/**
 * Hooks into grapesjs-mjml once its components are registered:
 *  - every canvas render (the plugin compiles each component on its own)
 *    gets the document's mj-attributes, mj-style and mj-font from `head`, so
 *    the canvas looks like the compiled email;
 *  - attribute values are escaped when MJML is written out (the plugin
 *    concatenates them as is, so a quote in alt text would break the file);
 *  - padding shorthands become longhands (see expandPadding);
 *  - returns a compile function using the bundled mjml-browser, so the
 *    exported HTML and the canvas come from the same engine.
 */
export function installMjmlHooks(editor: Editor, o: ComponentPluginOptions, head: () => string): MjmlCompile {
  const opt = o.opt as unknown as { mjmlParser: Parser }
  const parser = opt.mjmlParser
  opt.mjmlParser = (mjml, options) => {
    const h = head()
    if (h && !mjml.includes('<mj-head')) mjml = mjml.replace(/<mjml(\s[^>]*)?>/, (m) => `${m}<mj-head>${h}</mj-head>`)
    return parser(mjml, options)
  }
  for (const t of editor.Components.getTypes()) {
    if (!t.id.startsWith('mj-') && t.id !== 'mjml') continue
    const proto = (t.model as unknown as { prototype: Record<string, ((...a: unknown[]) => unknown) | undefined> }).prototype
    const getAttrToHTML = proto.getAttrToHTML as ((this: Component) => Attrs) | undefined
    const getMjmlAttributes = proto.getMjmlAttributes as ((this: Component) => Attrs) | undefined
    const init = proto.init as ((this: Component, ...a: unknown[]) => void) | undefined
    if (!getAttrToHTML || !getMjmlAttributes) continue
    editor.Components.addType(t.id, {
      model: {
        init(this: Component, ...a: unknown[]) {
          expandPadding(this)
          init?.apply(this, a)
        },
        getAttrToHTML(this: Component) {
          return collapsePadding(this, escapeAll(getAttrToHTML.call(this)))
        },
        getMjmlAttributes(this: Component) {
          return escapeAll(getMjmlAttributes.call(this))
        },
      },
    })
  }
  return (mjml: string) => parser(mjml, { validationLevel: 'soft', keepComments: true, minify: false, beautify: false })
}
