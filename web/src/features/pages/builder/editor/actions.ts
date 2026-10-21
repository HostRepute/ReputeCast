import type { Component } from 'grapesjs'
import { contentHtml, paletteItem, PALETTE, sectionHtml, wrapContent, type BlockContext, type PaletteId } from '../blocks'
import { LIVE_KIND, type BlockKind } from '../model'
import { attrOf, classesOf, elementChildren, freshIds, insertTarget, liveTagOf, tagOf, takenIds, topOf } from './nodes'
import type { PageEditor } from './pageEditor'

/**
 * What the palette and the block toolbar do to the page: add a block next
 * to the selection, move a block up or down among its siblings, duplicate
 * it, and give a content block dropped between blocks a section of its
 * own. Each is one step of the editor's undo history.
 */

function first(added: unknown): Component | undefined {
  return (Array.isArray(added) ? added[0] : added) as Component | undefined
}

/**
 * Adds a palette block: a content block into the selected column (or after
 * the selection in it), anything else as a block after the selected one.
 */
export function insertBlock(pe: PageEditor, id: PaletteId, sel: Component | null, ctx: BlockContext): Component | undefined {
  const wrapper = pe.editor.getWrapper()
  if (!wrapper) return undefined
  const taken = takenIds(wrapper)
  let added: Component | undefined
  if (paletteItem(id).level === 'content') {
    const target = insertTarget(sel)
    if (target) added = first(target.container.append(pe.prepare(contentHtml(id, ctx)), { at: target.at }))
  }
  if (!added) {
    const top = topOf(sel)
    const at = top ? wrapper.components().indexOf(top) + 1 : wrapper.components().length
    added = first(wrapper.append(pe.prepare(sectionHtml(id, ctx)), { at }))
  }
  if (added) freshIds(added, taken)
  return added
}

/** The palette block a dropped component came from, when the drop doesn't say. */
function paletteOf(c: Component): PaletteId | undefined {
  const live = liveTagOf(c)
  if (live) {
    const kind: BlockKind = LIVE_KIND[live]
    return PALETTE.find((p) => p.kind === kind)?.id
  }
  if (tagOf(c) === 'img') return 'image'
  const cls = classesOf(c)
  if (cls.includes('pg-btns')) return 'button'
  if (cls.includes('prose')) return 'text'
  return undefined
}

/**
 * A content block dropped between the page's blocks gets a section of its
 * own (the page's top level holds sections only); one dropped into a
 * column stays there. Either way its ids are its own.
 */
export function settleDrop(pe: PageEditor, c: Component, id: PaletteId | undefined, ctx: { name: (id: PaletteId) => string }): Component {
  const wrapper = pe.editor.getWrapper()
  if (!wrapper) return c
  if (c.parent() === wrapper && tagOf(c) !== 'section') {
    const pid = id ?? paletteOf(c)
    if (pid) {
      const at = wrapper.components().indexOf(c)
      const html = c.toHTML()
      const name = attrOf(c, 'data-pb-name') || ctx.name(pid)
      c.remove()
      const taken = takenIds(wrapper)
      const added = first(wrapper.append(pe.prepare(wrapContent(pid, paletteItem(pid).kind, name, html)), { at }))
      if (added) {
        freshIds(added, taken)
        return added
      }
      return c
    }
  }
  const taken = takenIds(wrapper)
  // The dropped block's own ids are counted once: drop them before checking.
  c.onAll((x: Component) => {
    const own = attrOf(x, 'data-pb-id')
    if (own) taken.delete(own)
  })
  freshIds(c, taken)
  return c
}

/** Moves a block before its previous sibling (-1) or after its next one (1). */
export function moveBlock(c: Component, dir: -1 | 1): boolean {
  const parent = c.parent()
  if (!parent) return false
  const kids = elementChildren(parent)
  const target = kids[kids.indexOf(c) + dir]
  if (!target) return false
  const at = parent.components().indexOf(target)
  c.move(parent, { at: dir < 0 ? at : at + 1 })
  return true
}

/** Whether a block can move that way. */
export function canMove(c: Component | null | undefined, dir: -1 | 1): boolean {
  const parent = c?.parent()
  if (!c || !parent) return false
  const kids = elementChildren(parent)
  const i = kids.indexOf(c)
  return i >= 0 && Boolean(kids[i + dir])
}

/** A copy of a block right after it, with ids of its own. */
export function duplicateBlock(pe: PageEditor, c: Component): Component | undefined {
  const parent = c.parent()
  const wrapper = pe.editor.getWrapper()
  if (!parent || !wrapper) return undefined
  const taken = takenIds(wrapper)
  const copy = first(parent.append(c.clone(), { at: parent.components().indexOf(c) + 1 }))
  if (copy) freshIds(copy, taken)
  return copy
}

/** Adds an item to a list block: a question, a testimonial, a chapter or benefit. */
export function addItem(pe: PageEditor, list: Component, html: string): Component | undefined {
  const res = first(list.append(pe.prepare(html)))
  return res
}
