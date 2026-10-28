import type { Component, Editor } from 'grapesjs'

/**
 * Synced blocks on the canvas (CO-11): a section marked rc-synced="header",
 * "footer" or a named block's key (R4.A3, "b-" and 12 hex digits) is the
 * workspace's synced block. It changes in Settings › Brand kit, and every
 * draft holding it follows, so its content stays read-only here; the
 * section itself can be moved or removed, or unlinked to edit it in this
 * email only.
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

/** Whether a synced kind is a named block's (not the Header or Footer). */
export function isNamedSynced(kind: string): boolean {
  return /^b-[0-9a-f]{12}$/.test(kind)
}

/** A section's MJML without its own rc-synced mark (what's inside keeps theirs). */
export function withoutSyncedMark(mjml: string): string {
  const end = mjml.indexOf('>')
  if (end < 0) return mjml
  return mjml.slice(0, end).replace(/\s+rc-synced\s*=\s*("[^"]*"|'[^']*'|[^\s>"']+)/, '') + mjml.slice(end)
}

/**
 * Puts MJML in place of a section (one change to the email): what's synced
 * in it is locked again, and the new section is selected.
 */
export function replaceSection(editor: Editor, section: Component, mjml: string): Component | undefined {
  const parent = section.parent()
  if (!parent) return undefined
  const at = parent.components().indexOf(section)
  const added = parent.append(mjml, { at }) as Component | Component[]
  section.remove()
  lockSynced(editor)
  const first = (Array.isArray(added) ? added : [added])[0]
  if (first) editor.select(first)
  return first
}

/**
 * Unlinks a synced section in this email (R4.A3): it keeps its content as
 * an ordinary section, editable here, and no longer follows the block.
 */
export function unlinkSynced(editor: Editor, section: Component): Component | undefined {
  return replaceSection(editor, section, withoutSyncedMark(section.toHTML()))
}
