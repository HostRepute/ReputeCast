import type { Component, ComponentView, Editor } from 'grapesjs'
import { isLiveTag, liveHTML, type LiveTag } from '../model'
import { canvasCss } from './canvasCss'
import { LiveRenderer, type RenderBlocks } from './live'
import { cleanHtml, CONTENT_TARGETS, prepareHtml, SECTION_TARGETS } from './prepare'

/**
 * The page builder's editor: GrapesJS in page mode, without grapesjs-mjml
 * and with every built-in panel off (ReputeCast draws its own around the
 * canvas). The canvas loads the site's stylesheets and draws the site's
 * header and footer around the blocks, so the page looks as readers see
 * it. Blocks are the page's HTML; live blocks are custom elements whose
 * attributes the Block tab edits and whose drawing comes from the server.
 */

export const PAGE_DEVICES = {
  desktop: { id: 'desktop', name: 'Desktop', width: '1200px', height: 'auto', widthMedia: '' },
  tablet: { id: 'tablet', name: 'Tablet', width: '834px', height: 'auto', widthMedia: '' },
  phone: { id: 'phone', name: 'Phone', width: '390px', height: 'auto', widthMedia: '' },
} as const

export type PageDevice = keyof typeof PAGE_DEVICES

export const DEVICE_WIDTH: Record<PageDevice, number> = { desktop: 1200, tablet: 834, phone: 390 }

/** The site's frame around the blocks (get-page-canvas). */
export interface FrameParts {
  header: string
  footer: string
  lang: string
  dir: string
  /** The page's address (/field-guide): its link in the navigation is the current one. */
  path?: string
}

/** What the canvas shows in a live block while it's drawn, when it can't be, or when it shows readers nothing. */
export type LivePlaceholder = (tag: LiveTag, attrs: Readonly<Record<string, string>>, state: 'loading' | 'failed' | 'empty') => string

export interface PageEditor {
  editor: Editor
  live: LiveRenderer
  /** The page's blocks as saved: top-level sections, one per line. */
  getHtml(): string
  /** Replaces every block (a version loaded or restored); not undoable. */
  setHtml(html: string): void
  /** Blocks' HTML annotated for the editor, to append or insert. */
  prepare(html: string): string
  /** The site's header and footer (a focused landing page shows the logo only). */
  setFrame(parts: FrameParts): void
  /** Draws every live block again (the cache cleared first). */
  repaintLive(): void
  /** True while a text is being edited on the canvas (its latest words aren't in the model yet). */
  isEditingText(): boolean
  /** Ends text editing so the typed text reaches the model. */
  flushEditing(): Promise<void>
  destroy(): void
}

export interface CreateOptions {
  html: string
  /** The site's stylesheets, in order (site.css, pages.css, theme.css). */
  styles: string[]
  frame: FrameParts
  device: PageDevice
  readOnly: boolean
  render: RenderBlocks
  placeholder: LivePlaceholder
  /** "13:00–13:30" for an event's time in the editor's own time zone, and the zone's city ("New York"); null for none. */
  localTime: (start: Date, end: Date) => { time: string; city: string } | null
  frameTitle: string
}

type LiveView = ComponentView & { __rcKey?: string }

/** The live block's drawing inside its element: inert, without ids (they'd repeat the block's own). */
function fill(view: LiveView, html: string, localTime: CreateOptions['localTime']) {
  const el = view.el as HTMLElement | undefined
  if (!el) return
  const doc = el.ownerDocument
  const box = doc.createElement('div')
  box.className = 'rc-live'
  box.setAttribute('inert', '')
  box.setAttribute('aria-hidden', 'true')
  box.innerHTML = html
  box.querySelectorAll('[id]').forEach((x) => x.removeAttribute('id'))
  // The canvas is as tall as the page: every image shows at once.
  box.querySelectorAll('img[loading]').forEach((x) => x.removeAttribute('loading'))
  // Event details: the reader's own time, as pages.js adds it on the site.
  box.querySelectorAll<HTMLElement>('.lt[data-start]').forEach((x) => {
    const start = new Date(x.getAttribute('data-start') ?? '')
    const end = new Date(x.getAttribute('data-end') ?? '')
    if (Number.isNaN(start.getTime())) return
    const t = localTime(start, Number.isNaN(end.getTime()) ? start : end)
    if (!t) return
    x.textContent = (x.getAttribute('data-tpl') || '{time} · {zone}').replace('{time}', t.time).replace('{zone}', t.city)
    x.hidden = false
  })
  el.replaceChildren(box)
}

function attrsOf(c: Component): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(c.getAttributes() as Record<string, unknown>)) if (v !== undefined && v !== null && v !== false) out[k] = String(v)
  return out
}

function paint(view: LiveView, live: LiveRenderer, opts: CreateOptions) {
  const model = view.model
  const tag = String(model.get('tagName') ?? '') as LiveTag
  if (!isLiveTag(tag)) return
  const attrs = attrsOf(model)
  const key = liveHTML(tag, attrs)
  view.__rcKey = key
  const show = (html: string | null) => {
    if (view.__rcKey !== key) return
    fill(view, html === null ? opts.placeholder(tag, attrs, 'failed') : html.trim() ? html : opts.placeholder(tag, attrs, 'empty'), opts.localTime)
  }
  const known = live.peek(key)
  if (known !== undefined) {
    show(known)
    return
  }
  fill(view, opts.placeholder(tag, attrs, 'loading'), opts.localTime)
  void live.request(key).then(show)
}

function addTypes(editor: Editor, live: LiveRenderer, opts: CreateOptions) {
  const dc = editor.Components
  // A block: a top-level section of the page.
  dc.addType('pb-section', {
    isComponent: (el: HTMLElement) => (el.tagName === 'SECTION' && typeof el.hasAttribute === 'function' && el.hasAttribute('data-pb') ? { type: 'pb-section' } : false),
    model: {
      defaults: { draggable: SECTION_TARGETS, droppable: false, removable: true, copyable: true, selectable: true, hoverable: true },
    },
  })
  // A live block: its attributes are the model; its drawing comes from the server.
  dc.addType('pb-live', {
    isComponent: (el: HTMLElement) => {
      const tag = typeof el.tagName === 'string' ? el.tagName.toLowerCase() : ''
      return isLiveTag(tag) ? { type: 'pb-live', tagName: tag, components: [] } : false
    },
    model: {
      defaults: { draggable: CONTENT_TARGETS, droppable: false, editable: false, removable: true, copyable: true, selectable: true, hoverable: true, components: [] },
      toHTML() {
        const tag = String(this.get('tagName') ?? '')
        return isLiveTag(tag) ? liveHTML(tag, attrsOf(this)) : ''
      },
    },
    view: {
      init() {
        this.listenTo(this.model, 'change:attributes', () => paint(this as LiveView, live, opts))
      },
      onRender() {
        paint(this as LiveView, live, opts)
      },
    },
  })
}

/** A link's path, or '' when the address doesn't parse. */
function linkPath(href: string): string {
  try {
    return new URL(href, 'http://x').pathname
  } catch {
    return ''
  }
}

/** Puts the site's frame around GrapesJS's wrapper: <div class="st"> header <main class="pg-main"> blocks </main> footer </div>. */
function mountFrame(editor: Editor, parts: FrameParts, title: string) {
  const doc = editor.Canvas.getDocument()
  const wrapperEl = editor.getWrapper()?.getEl()
  if (!doc?.body || !wrapperEl) return
  doc.documentElement.lang = parts.lang || 'en'
  doc.documentElement.dir = parts.dir || 'ltr'
  let st = doc.getElementById('rc-site')
  if (!st) {
    st = doc.createElement('div')
    st.id = 'rc-site'
    st.className = 'st'
    const head = doc.createElement('div')
    head.id = 'rc-head'
    head.className = 'rc-frame'
    head.setAttribute('inert', '')
    const main = doc.createElement('main')
    main.id = 'rc-main'
    main.className = 'pg-main'
    const foot = doc.createElement('div')
    foot.id = 'rc-foot'
    foot.className = 'rc-frame'
    foot.setAttribute('inert', '')
    wrapperEl.parentNode?.insertBefore(st, wrapperEl)
    st.append(head, main, foot)
    main.append(wrapperEl)
  }
  const head = doc.getElementById('rc-head')
  const foot = doc.getElementById('rc-foot')
  if (head) {
    head.innerHTML = parts.header
    // The page's own link in the navigation, as the site marks it.
    const own = parts.path
    if (own) {
      head.querySelectorAll<HTMLAnchorElement>('nav a[href]').forEach((a) => {
        const path = linkPath(a.getAttribute('href') ?? '')
        if (path && (path === own || path.endsWith(own))) a.setAttribute('aria-current', 'page')
      })
    }
  }
  if (foot) foot.innerHTML = parts.footer
  editor.Canvas.getFrameEl()?.setAttribute('title', title)
}

/**
 * Creates the editor. GrapesJS loads on demand, to stay out of the main
 * bundle.
 */
export async function createPageEditor(container: HTMLElement, opts: CreateOptions): Promise<PageEditor> {
  const { default: grapesjs } = await import('grapesjs')
  const live = new LiveRenderer(opts.render)
  let frame = opts.frame
  const editor = grapesjs.init({
    container,
    height: 'auto',
    width: 'auto',
    storageManager: false,
    telemetry: false,
    cssIcons: '',
    noticeOnUnload: false,
    showToolbar: false,
    showOffsets: false,
    multipleSelection: false,
    avoidInlineStyle: true,
    protectedCss: '',
    undoManager: { trackSelection: false },
    panels: { defaults: [] },
    blockManager: { custom: true },
    layerManager: { custom: true },
    styleManager: { custom: true },
    traitManager: { custom: true },
    selectorManager: { custom: true, componentFirst: true },
    assetManager: { custom: true },
    richTextEditor: { custom: true },
    parser: { optionsHtml: { allowScripts: false, allowUnsafeAttr: false, allowUnsafeAttrValue: false } },
    canvasCss,
    deviceManager: { default: opts.device, devices: [PAGE_DEVICES.desktop, PAGE_DEVICES.tablet, PAGE_DEVICES.phone] },
    canvas: { styles: opts.styles, scripts: [], customSpots: { select: true, target: true }, autoscrollLimit: 60 },
    plugins: [(ed: Editor) => addTypes(ed, live, opts)],
    components: prepareHtml(opts.html),
  })
  await new Promise<void>((resolve) => editor.onReady(() => resolve()))
  const mount = () => mountFrame(editor, frame, opts.frameTitle)
  mount()
  editor.on('canvas:frame:load:body', mount)
  if (opts.readOnly) {
    editor.getWrapper()?.onAll((c: Component) => c.set({ draggable: false, droppable: false, editable: false, removable: false, copyable: false }))
  }
  editor.UndoManager.clear()

  const repaint = () => {
    editor.getWrapper()?.onAll((c: Component) => {
      if (c.get('type') === 'pb-live' && c.view) paint(c.view as LiveView, live, opts)
    })
  }

  return {
    editor,
    live,
    getHtml: () => {
      const parts: string[] = []
      editor.getWrapper()?.components().forEach((c: Component) => {
        const h = c.toHTML().trim()
        if (h) parts.push(h)
      })
      return cleanHtml(parts.join('\n'))
    },
    setHtml: (html: string) => {
      editor.UndoManager.skip(() => {
        editor.select(undefined)
        editor.setComponents(prepareHtml(html))
      })
      if (opts.readOnly) editor.getWrapper()?.onAll((c: Component) => c.set({ draggable: false, droppable: false, editable: false, removable: false, copyable: false }))
      editor.UndoManager.clear()
    },
    prepare: prepareHtml,
    setFrame: (parts: FrameParts) => {
      frame = parts
      mount()
    },
    repaintLive: () => {
      live.clear()
      repaint()
    },
    isEditingText: () => Boolean(editor.getEditing()),
    flushEditing: async () => {
      if (!editor.getEditing()) return
      // GrapesJS ends text editing on 'toolbar:run:before' (as before any
      // toolbar command) and copies the text into the model just after.
      editor.trigger('toolbar:run:before')
      for (let i = 0; i < 20 && editor.getEditing(); i++) await new Promise((r) => window.setTimeout(r, 15))
      await new Promise((r) => window.setTimeout(r, 30))
    },
    destroy: () => {
      live.dispose()
      editor.destroy()
    },
  }
}
