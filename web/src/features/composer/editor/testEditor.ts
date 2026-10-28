import type { Editor } from 'grapesjs'
import type { ComposerEditor } from './grapes'
import { loadLiquidChips } from './liquid'
import { withLiveText } from './live'
import { installMjmlHooks, type MjmlCompile } from './mjmlHooks'

/**
 * A headless GrapesJS editor with the MJML plugin, as the composer sets it
 * up, for tests: the components are the composer's (mj-section, mj-text…),
 * without a canvas.
 */
export async function headlessMjmlEditor(body = ''): Promise<{ editor: Editor; compile: MjmlCompile }> {
  const { default: grapesjs } = await import('grapesjs')
  const mjmlModule = await import('grapesjs-mjml')
  const mjmlPlugin = ((mjmlModule.default as unknown as { default?: typeof mjmlModule.default }).default ?? mjmlModule.default) as typeof mjmlModule.default
  let compile: MjmlCompile | undefined
  const container = document.createElement('div')
  document.body.appendChild(container)
  const editor = grapesjs.init({
    container,
    headless: true,
    storageManager: false,
    telemetry: false,
    plugins: [
      (ed: Editor) =>
        mjmlPlugin(ed, {
          blocks: [],
          resetBlocks: true,
          customComponents: [
            (e) => loadLiquidChips(e),
            (_e, o) => {
              compile = installMjmlHooks(ed, o, () => '')
            },
          ],
        }),
    ],
  })
  if (body) editor.setComponents(body)
  editor.UndoManager.clear()
  return { editor, compile: compile! }
}

/**
 * createComposerEditor for tests of the whole composer (vi.mock of
 * './editor/grapes'): the composer's interface over a headless editor, so
 * saves, autosave and previews run as in the app, without a canvas.
 */
export async function headlessComposerEditor(_container: HTMLElement, opts: { body: string }): Promise<ComposerEditor> {
  const { editor, compile } = await headlessMjmlEditor(opts.body)
  // There is no canvas frame: its parts read as absent, as before it loads.
  editor.refresh = () => {}
  Object.assign(editor.Canvas, {
    getElement: () => undefined,
    getFrameEl: () => undefined,
    getDocument: () => undefined,
    getWindow: () => undefined,
    getBody: () => undefined,
    getZoom: () => 100,
    setZoom: () => {},
  })
  return {
    editor,
    compile,
    getBody: () => editor.getHtml().trim(),
    getLiveBody: () => withLiveText(editor, () => editor.getHtml().trim()),
    setBody: (b: string) => {
      editor.setComponents(b)
    },
    isEditingText: () => !!editor.getEditing(),
    flushEditing: async () => {},
  }
}
