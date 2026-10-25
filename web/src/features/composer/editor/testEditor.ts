import type { Editor } from 'grapesjs'
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
