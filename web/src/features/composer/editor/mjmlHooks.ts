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

/** Tags whose padding can't be written per side (padding-top…). */
const NO_SIDES = new Set(['mjml', 'mj-head', 'mj-body', 'mj-group', 'mj-raw', 'mj-navbar'])

/**
 * A `padding="16px 24px"` shorthand becomes longhands when a component is
 * created, so the inspector edits one side at a time (MJML gives a longhand
 * precedence over the shorthand); export writes the shortest shorthand back.
 */
function expandPadding(c: Component) {
  if (NO_SIDES.has(String(c.get('tagName') ?? ''))) return
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

/** All four longhands → one shorthand; a partial set stays as longhands. */
function collapsePadding(out: Attrs): Attrs {
  const sides = SIDES.map((s) => out[`padding-${s}`])
  if (!sides.every((v) => v !== undefined && v !== null && v !== '')) return out
  const next = { ...out }
  for (const s of SIDES) delete next[`padding-${s}`]
  const [t, r, b, l] = sides.map(String) as [string, string, string, string]
  next.padding = collapseBox(t, r, b, l)
  return next
}

/** The component's own MJML attributes. */
function ownAttributes(c: Component): Attrs {
  const a = { ...((c.get('attributes') ?? {}) as Attrs) }
  delete a.style
  delete a.id
  return a
}

/**
 * Hooks into grapesjs-mjml once its components are registered:
 *  - every canvas render (the plugin compiles each component on its own)
 *    gets the document's mj-attributes, mj-style and mj-font from `head`, so
 *    the canvas looks like the compiled email;
 *  - components keep only the attributes the source gives them: the plugin
 *    copies its defaults into every component (for its style manager, which
 *    isn't used), and rendering those would override the head's
 *    mj-attributes in the canvas, while export drops them anyway;
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
    const getMjmlAttributes = proto.getMjmlAttributes as ((this: Component) => Attrs) | undefined
    const init = proto.init as ((this: Component, ...a: unknown[]) => void) | undefined
    if (!proto.getAttrToHTML || !getMjmlAttributes) continue
    editor.Components.addType(t.id, {
      model: {
        init(this: Component, ...a: unknown[]) {
          expandPadding(this)
          const mine = ownAttributes(this)
          init?.apply(this, a)
          this.set('attributes', mine, { silent: true })
          this.set('style', {}, { silent: true })
        },
        // Attributes and style aren't mirrored: the style manager isn't used.
        handleAttributeChange() {},
        handleStyleChange() {},
        getAttrToHTML(this: Component) {
          return collapsePadding(escapeAll(ownAttributes(this)))
        },
        getMjmlAttributes(this: Component) {
          return escapeAll(getMjmlAttributes.call(this))
        },
      },
    })
  }
  return (mjml: string) => parser(mjml, { validationLevel: 'soft', keepComments: true, minify: false, beautify: false })
}
