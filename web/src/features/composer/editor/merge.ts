import type { Editor } from 'grapesjs'
import { encodeTables } from './components'
import { blockContent, mjChildren } from './labels'

/**
 * Puts sections someone else saved into the canvas at their positions
 * (CO-10: a save merged with theirs). `index` counts the body's MJML
 * sections, as the server does. Returns how many were put in.
 */
export function applySections(editor: Editor, sections: { index: number; mjml: string }[]): number {
  const body = editor.getWrapper()?.findType('mj-body')[0]
  if (!body) return 0
  const kids = mjChildren(body)
  let n = 0
  for (const sec of sections) {
    const old = kids[sec.index]
    if (!old) continue
    const at = body.components().indexOf(old)
    old.remove()
    body.append(blockContent(encodeTables(sec.mjml)), { at })
    n++
  }
  return n
}
