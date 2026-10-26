import type { Component, Editor } from 'grapesjs'

/**
 * Synced blocks on the canvas (CO-11): a section marked rc-synced="header"
 * or "footer" is the workspace's synced block. It changes in Settings ›
 * Brand kit, and every draft holding it follows, so its content stays
 * read-only here; the section itself can be moved or removed.
 */

export const SYNCED_ATTR = 'rc-synced'

/** The synced section a component is in (or is), if any. */
export function syncedSectionOf(c: Component | null | undefined): { section: Component; kind: string } | null {
  for (let x: Component | undefined | null = c; x; x = x.parent()) {
    const kind = (x.getAttributes() as Record<string, unknown>)[SYNCED_ATTR]
    if (typeof kind === 'string' && kind) return { section: x, kind }
  }
  return null
}

/** Locks the content of every synced section. Nothing here is an edit (avoidStore). */
export function lockSynced(editor: Editor) {
  editor.getWrapper()?.onAll((c: Component) => {
    const kind = (c.getAttributes() as Record<string, unknown>)[SYNCED_ATTR]
    if (typeof kind !== 'string' || !kind) return
    c.set({ droppable: false }, { avoidStore: true })
    c.components().forEach((child: Component) =>
      child.onAll((x: Component) => x.set({ editable: false, droppable: false, draggable: false, removable: false, copyable: false }, { avoidStore: true })),
    )
  })
}
