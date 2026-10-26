import type { Component, ComponentProperties, Editor } from 'grapesjs'

const KEYS = ['draggable', 'droppable', 'editable', 'removable', 'copyable'] as const

type Allowed = Pick<ComponentProperties, (typeof KEYS)[number]>

/** What a read-only canvas took away from each component. */
export type Locked = Map<Component, Allowed>

/**
 * Makes the canvas read-only (no dragging, dropping, text editing, removing
 * or copying), keeping what each component allowed so unlockCanvas can give
 * it back: an approved campaign opens read-only until the person agrees that
 * editing cancels the approval (CO-7). Nothing here is an edit.
 */
export function lockCanvas(editor: Editor): Locked {
  const saved: Locked = new Map()
  editor.getWrapper()?.onAll((c: Component) => {
    saved.set(c, Object.fromEntries(KEYS.map((k) => [k, c.get(k)])) as Allowed)
    c.set({ draggable: false, droppable: false, editable: false, removable: false, copyable: false }, { avoidStore: true })
  })
  return saved
}

/** Gives the components back what lockCanvas took. */
export function unlockCanvas(saved: Locked) {
  saved.forEach((attrs, c) => c.set(attrs, { avoidStore: true }))
}
