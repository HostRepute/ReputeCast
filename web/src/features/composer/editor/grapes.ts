import type { Editor } from 'grapesjs'
import type { ComponentPluginOptions } from 'grapesjs-mjml'
import { canvasCss } from './canvasCss'
import { encodeTables, loadTable } from './components'
import { installMjmlHooks, type MjmlCompile } from './mjmlHooks'

/** Devices of the device switcher: desktop 600 px and mobile 375 px (MJML stacks columns below 480 px). */
export const DEVICES = {
  desktop: { id: 'desktop', name: 'Desktop', width: '600px', height: 'auto', widthMedia: '' },
  mobile: { id: 'mobile', name: 'Mobile', width: '375px', height: 'auto', widthMedia: '480px' },
} as const

export type DeviceId = keyof typeof DEVICES

export interface ComposerEditor {
  editor: Editor
  /** Compiles a full MJML document with the bundled mjml-browser (same engine as the canvas). */
  compile: MjmlCompile
  /** The canvas as MJML: <mjml><mj-body>…</mj-body></mjml>. */
  getBody(): string
  /** Replaces the canvas content (undoable). */
  setBody(body: string): void
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
 * turned off: ReputeCast renders its own panels (blocks, layers, style,
 * settings, assets, rich text toolbar, selection toolbar) around the canvas.
 * GrapesJS and grapesjs-mjml are loaded on demand so they stay out of the
 * main bundle.
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
    deviceManager: { default: opts.device, devices: [DEVICES.desktop, DEVICES.mobile] },
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
  // Dev builds expose the editor to the QA scripts (scripts/qa.ts and friends).
  if (import.meta.env.DEV) (window as unknown as { __rcComposer?: Editor }).__rcComposer = editor
  return {
    editor,
    compile,
    getBody: () => editor.getHtml().trim(),
    setBody: (body: string) => {
      editor.setComponents(encodeTables(body))
    },
  }
}
