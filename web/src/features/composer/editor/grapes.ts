import type { Component, Editor } from 'grapesjs'
import type { ComponentPluginOptions } from 'grapesjs-mjml'
import { loadFeedItems } from '@/features/digests/composer/feedComponent'
import { loadGrowBlocks } from '@/features/grow/composer/blocks'
import { canvasCss } from './canvasCss'
import { encodeTables, loadTable } from './components'
import { chipify, loadLiquidChips } from './liquid'
import { withLiveText } from './live'
import { installMjmlHooks, type MjmlCompile } from './mjmlHooks'

/**
 * Devices of the device switcher: desktop 600 px and mobile 375 px (MJML
 * stacks columns below 480 px). A website post (post mode) is edited in the
 * site's 680 px reading column instead of the email's 600 px.
 */
export const DEVICES = {
  desktop: { id: 'desktop', name: 'Desktop', width: '600px', height: 'auto', widthMedia: '' },
  mobile: { id: 'mobile', name: 'Mobile', width: '375px', height: 'auto', widthMedia: '480px' },
  web: { id: 'web', name: 'Web', width: '680px', height: 'auto', widthMedia: '' },
} as const

/** The devices an email's switcher offers (the web column is post mode's). */
export const EMAIL_DEVICES: EmailDevice[] = ['desktop', 'mobile']

export type DeviceId = keyof typeof DEVICES

/** An email's devices: desktop and mobile (the address bar keeps one of them). */
export type EmailDevice = Exclude<DeviceId, 'web'>

export interface ComposerEditor {
  editor: Editor
  /** Compiles a full MJML document with the bundled mjml-browser (same engine as the canvas). */
  compile: MjmlCompile
  /** The canvas as MJML: <mjml><mj-body>…</mj-body></mjml>. */
  getBody(): string
  /**
   * The canvas as MJML with the words of a text being typed, without ending
   * its editing: autosave and the browser journal take them while the
   * person is still typing (getBody has them only once the editing ends).
   */
  getLiveBody(): string
  /** Replaces the canvas content (undoable). */
  setBody(body: string): void
  /** True while a text is being edited on the canvas (its latest words aren't in the model yet). */
  isEditingText(): boolean
  /** Ends text editing so the typed text reaches the model (before a save or a compile). */
  flushEditing(): Promise<void>
}

export interface CreateOptions {
  /** <mjml><mj-body>…</mj-body></mjml>, without the head. */
  body: string
  /** The head parts that affect rendering (mj-attributes, mj-style, mj-font). */
  head: () => string
  device: DeviceId
}

/** @font-face rules of the app (IBM Plex), so email text set in Plex renders in the canvas. */
function fontFaces(): string {
  const out: string[] = []
  for (const sheet of Array.from(document.styleSheets)) {
    let rules: CSSRuleList
    try {
      rules = sheet.cssRules
    } catch {
      continue
    }
    for (const r of Array.from(rules)) if (r instanceof CSSFontFaceRule) out.push(r.cssText)
  }
  return out.join('\n')
}

/**
 * Creates the GrapesJS editor with the MJML plugin and every built-in UI
 * turned off: ReputeCast renders its own panels around the canvas. GrapesJS
 * and grapesjs-mjml load on demand, to stay out of the main bundle.
 */
export async function createComposerEditor(container: HTMLElement, opts: CreateOptions): Promise<ComposerEditor> {
  const [{ default: grapesjs }, mjmlModule] = await Promise.all([import('grapesjs'), import('grapesjs-mjml')])
  // The plugin ships as UMD: depending on the bundler's interop the plugin is
  // the default export or the default export's `default`.
  const mjmlPlugin = ((mjmlModule.default as unknown as { default?: typeof mjmlModule.default }).default ?? mjmlModule.default) as typeof mjmlModule.default
  let compile: MjmlCompile | undefined
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
    undoManager: { trackSelection: false },
    panels: { defaults: [] },
    blockManager: { custom: true },
    layerManager: { custom: true },
    styleManager: { custom: true },
    traitManager: { custom: true },
    selectorManager: { custom: true, componentFirst: true },
    assetManager: { custom: true },
    richTextEditor: { custom: true },
    canvasCss: `${canvasCss}\n${fontFaces()}`,
    deviceManager: { default: opts.device, devices: [DEVICES.desktop, DEVICES.mobile, DEVICES.web] },
    canvas: { customSpots: { select: true, target: true }, autoscrollLimit: 60 },
    plugins: [
      (ed: Editor) =>
        mjmlPlugin(ed, {
          blocks: [],
          resetBlocks: true,
          resetDevices: false,
          resetStyleManager: false,
          useCustomTheme: false,
          overwriteExport: false,
          columnsPadding: '0px',
          customComponents: [
            (e: Editor, o: ComponentPluginOptions) => loadTable(e, o),
            // Personalization tokens as chips (CO-4); they export as their Liquid.
            (e: Editor) => loadLiquidChips(e),
            // A recurring digest's Feed items block (it only appears in digest templates).
            (e: Editor, o: ComponentPluginOptions) => loadFeedItems(e, o),
            // Grow's Referral, Recommendations and Reward blocks (design expansion, X15).
            (e: Editor, o: ComponentPluginOptions) => loadGrowBlocks(e, o),
            (_e: Editor, o: ComponentPluginOptions) => {
              compile = installMjmlHooks(ed, o, opts.head)
            },
          ],
        }),
    ],
    components: encodeTables(opts.body),
  })
  await new Promise<void>((resolve) => editor.onReady(() => resolve()))
  if (!compile) throw new Error('grapesjs-mjml did not load')
  chipify(editor)
  // Tokens typed in a text, or brought in by a block, become chips too.
  const pending = new Set<Component>()
  let timer: ReturnType<typeof setTimeout> | undefined
  const later = (c: Component | undefined) => {
    if (!c) return
    pending.add(c)
    if (timer === undefined)
      timer = setTimeout(() => {
        timer = undefined
        const roots = [...pending]
        pending.clear()
        // Removed meanwhile (no parent): nothing to do.
        for (const r of roots) if (r.parent()) chipify(editor, r)
      }, 0)
  }
  editor.on('rte:disable', (view: { model?: Component } | undefined) => later(view?.model))
  editor.on('component:add', (c: Component) => {
    if (c.get('type') !== 'rc-liquid' && c.get('type') !== 'textnode') later(c)
  })
  return {
    editor,
    compile,
    getBody: () => editor.getHtml().trim(),
    getLiveBody: () => withLiveText(editor, () => editor.getHtml().trim()),
    setBody: (body: string) => {
      editor.setComponents(encodeTables(body))
      chipify(editor)
    },
    isEditingText: () => !!editor.getEditing(),
    flushEditing: async () => {
      if (!editor.getEditing()) return
      // GrapesJS ends text editing on 'toolbar:run:before' (as before any toolbar
      // command) and copies the text into the model just after 'rte:disable'.
      editor.trigger('toolbar:run:before')
      for (let i = 0; i < 20 && editor.getEditing(); i++) await new Promise((r) => window.setTimeout(r, 15))
      await new Promise((r) => window.setTimeout(r, 30))
    },
  }
}
