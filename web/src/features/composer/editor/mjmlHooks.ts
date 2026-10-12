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

/** Escapes an attribute value for the MJML source (grapesjs-mjml writes values verbatim). */
export function escapeAttr(v: unknown): string {
  return String(v).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function escapeAll(attrs: Record<string, unknown>): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(attrs)) out[k] = v === undefined || v === null || v === '' ? v : escapeAttr(v)
  return out
}

/**
 * Hooks into grapesjs-mjml once its components are registered:
 *  - every canvas render (the plugin compiles each component on its own)
 *    gets the document's mj-attributes, mj-style and mj-font from `head`, so
 *    the canvas looks like the compiled email;
 *  - attribute values are escaped when MJML is written out (the plugin
 *    concatenates them as is, so a quote in alt text would break the file);
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
    const proto = (t.model as unknown as { prototype: Record<string, ((...a: unknown[]) => Record<string, unknown>) | undefined> }).prototype
    const getAttrToHTML = proto.getAttrToHTML
    const getMjmlAttributes = proto.getMjmlAttributes
    if (!getAttrToHTML || !getMjmlAttributes) continue
    editor.Components.addType(t.id, {
      model: {
        getAttrToHTML(this: Component) {
          return escapeAll(getAttrToHTML.call(this))
        },
        getMjmlAttributes(this: Component) {
          return escapeAll(getMjmlAttributes.call(this))
        },
      },
    })
  }
  return (mjml: string) => parser(mjml, { validationLevel: 'soft', keepComments: true, minify: false, beautify: false })
}
