import type { Editor } from 'grapesjs'
import type { ComponentPluginOptions } from 'grapesjs-mjml'

/** Attribute carrying mj-table's rows through GrapesJS's HTML parser (see encodeTables). */
export const RAW_ATTR = 'data-rc-raw'

/**
 * Bare <tr>/<td> outside a <table> are dropped by an HTML parser, so the
 * rows of an <mj-table> are moved into an attribute before GrapesJS parses
 * the document and restored by the component.
 */
export function encodeTables(src: string): string {
  return src.replace(/<mj-table(\s[^>]*)?>([\s\S]*?)<\/mj-table>/gi, (_m, attrs: string | undefined, inner: string) => {
    return `<mj-table${attrs ?? ''} ${RAW_ATTR}="${encodeURIComponent(inner)}"></mj-table>`
  })
}

/**
 * mj-table: grapesjs-mjml has no component for it. Its rows are raw HTML
 * (edited in Settings or the code view); the canvas renders it with MJML
 * like the other content components.
 */
export function loadTable(editor: Editor, { coreMjmlModel, coreMjmlView }: ComponentPluginOptions) {
  editor.Components.addType('mj-table', {
    isComponent: (el: HTMLElement) => {
      if ((el.tagName || '').toLowerCase() !== 'mj-table') return false
      const raw = el.getAttribute(RAW_ATTR)
      el.removeAttribute(RAW_ATTR)
      return { type: 'mj-table', components: [], content: raw ? decodeURIComponent(raw) : el.innerHTML }
    },
    model: {
      ...coreMjmlModel,
      defaults: {
        name: 'Table',
        draggable: '[data-gjs-type="mj-column"], [data-gjs-type="mj-hero"]',
        droppable: false,
        highlightable: false,
        stylable: ['font-family', 'font-size', 'color', 'line-height', 'align', 'width', 'padding', 'padding-top', 'padding-right', 'padding-bottom', 'padding-left', 'container-background-color'],
        'style-default': { 'padding-top': '10px', 'padding-bottom': '10px', 'padding-right': '25px', 'padding-left': '25px', 'font-size': '13px', align: 'left' },
      },
    },
    view: {
      ...coreMjmlView,
      tagName: 'tr',
      attributes: { style: 'pointer-events: all; display: table; width: 100%' },
      getMjmlTemplate() {
        return { start: '<mjml><mj-body><mj-column>', end: '</mj-column></mj-body></mjml>' }
      },
      getInnerMjmlTemplate(this: { model: { get(k: string): unknown } }) {
        const orig = coreMjmlView.getInnerMjmlTemplate.call(this)
        return { start: `${orig.start}${String(this.model.get('content') ?? '')}`, end: orig.end }
      },
      getTemplateFromEl(sandboxEl: HTMLElement) {
        return sandboxEl.querySelector('tr')?.innerHTML ?? ''
      },
      getChildrenSelector() {
        return 'td'
      },
      rerender(this: { render(): void }) {
        this.render()
      },
    },
  })
  editor.Components.addType('mj-table', {
    model: {
      // Rows are raw HTML: written back verbatim.
      toHTML(this: { get(k: string): unknown; getAttrToHTML(): Record<string, unknown> }) {
        const attrs = this.getAttrToHTML()
        const str = Object.entries(attrs)
          .filter(([, v]) => v !== undefined && v !== '')
          .map(([k, v]) => ` ${k}="${String(v)}"`)
          .join('')
        return `<mj-table${str}>${String(this.get('content') ?? '')}</mj-table>`
      },
    },
  })
}
