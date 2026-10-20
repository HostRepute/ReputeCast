import type { Component } from 'grapesjs'
import { freeId, innerKind, isInnerBlock, isLiveTag, sectionKind, type BlockKind, type LiveTag, type PbNode } from '../model'

/**
 * GrapesJS components behind the model's face (PbNode), and the walks the
 * builder needs: a component's block, its section, its column, the block
 * ids in use.
 */

export function tagOf(c: Component | null | undefined): string {
  return c ? String(c.get('tagName') ?? '').toLowerCase() : ''
}

export function attrsOf(c: Component): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(c.getAttributes() as Record<string, unknown>)) if (v !== undefined && v !== null && v !== false) out[k] = v === true ? '' : String(v)
  return out
}

export function attrOf(c: Component, name: string): string {
  const v = (c.getAttributes() as Record<string, unknown>)[name]
  return v === undefined || v === null || v === false ? '' : v === true ? '' : String(v)
}

export function classesOf(c: Component): string[] {
  return (c.getClasses() as unknown[]).map(String)
}

/** Element children: text nodes and comments left out. */
export function elementChildren(c: Component): Component[] {
  return c.components().filter((x: Component) => {
    const type = String(x.get('type') ?? '')
    return type !== 'textnode' && type !== 'comment' && tagOf(x) !== ''
  })
}

export function nodeOf(c: Component): PbNode {
  return {
    key: c.cid,
    tag: tagOf(c),
    attr: (name: string) => (name === 'class' ? classesOf(c).join(' ') : attrOf(c, name)),
    get classes() {
      return classesOf(c)
    },
    get children() {
      return elementChildren(c).map(nodeOf)
    },
  }
}

export function isWrapper(c: Component | null | undefined): boolean {
  return Boolean(c) && c!.get('type') === 'wrapper'
}

/** A top-level block of the page (a child of the wrapper). */
export function isTop(c: Component | null | undefined): boolean {
  return Boolean(c) && isWrapper(c!.parent())
}

/**
 * Whether adding, removing or changing this component changes the page's
 * blocks (a section, a block of its own, an image), not the text inside
 * one: a text edit swaps its text's nodes when it ends.
 */
export function changesBlocks(c: Component): boolean {
  const tag = tagOf(c)
  return tag === 'section' || tag === 'img' || isLiveTag(tag) || attrOf(c, 'data-pb') !== '' || attrOf(c, 'data-pb-name') !== '' || isTop(c)
}

/** The top-level block a component belongs to. */
export function topOf(c: Component | null | undefined): Component | undefined {
  let cur = c ?? undefined
  while (cur && !isTop(cur)) cur = cur.parent()
  return cur
}

export function liveTagOf(c: Component | null | undefined): LiveTag | null {
  const tag = tagOf(c)
  return isLiveTag(tag) ? tag : null
}

/** A block of its own inside a section: a live block, an image, a named part. */
export function isBlock(c: Component): boolean {
  return isTop(c) || isInnerBlock(nodeOf(c))
}

/** The block a component belongs to (itself when it's one). */
export function blockOf(c: Component | null | undefined): Component | undefined {
  let cur = c ?? undefined
  while (cur && !isWrapper(cur) && !isBlock(cur)) cur = cur.parent()
  return cur && !isWrapper(cur) ? cur : undefined
}

export function kindOf(c: Component): BlockKind {
  if (isTop(c)) return sectionKind(nodeOf(c))
  return innerKind(nodeOf(c)) ?? 'text'
}

const COLUMN_CLASSES = ['pg-col', 'hero-copy', 'lp-copy', 'hero-art']

/** A column of a grid: its number from 1, or null. */
export function columnNumber(c: Component): number | null {
  const cls = classesOf(c)
  const parent = c.parent()
  if (!parent) return null
  const grid = classesOf(parent)
  const inGrid = grid.includes('pg-cols') || grid.includes('hero-grid') || grid.includes('lp-hero')
  if (!inGrid && !COLUMN_CLASSES.some((x) => cls.includes(x))) return null
  if (!inGrid) return null
  return elementChildren(parent).indexOf(c) + 1
}

/** Containers a pressed palette tile adds content blocks into (not a section's plain inner wrapper). */
const INSERT_CONTAINERS = ['pg-col', 'hero-copy', 'lp-copy', 'lp-body', 'lp-cols']

/** Where a content block goes when its tile is pressed: the selected column, or the column the selection sits in. */
export function insertTarget(sel: Component | null | undefined): { container: Component; at: number } | undefined {
  if (!sel) return undefined
  const isContainer = (x: Component) => INSERT_CONTAINERS.some((cls) => classesOf(x).includes(cls))
  if (isContainer(sel)) return { container: sel, at: sel.components().length }
  const parent = sel.parent()
  if (parent && isContainer(parent)) return { container: parent, at: parent.components().indexOf(sel) + 1 }
  return undefined
}

/** Every block id the page uses. */
export function takenIds(root: Component | null | undefined): Set<string> {
  const out = new Set<string>()
  root?.onAll((c: Component) => {
    const id = attrOf(c, 'data-pb-id')
    if (id) out.add(id)
    const el = attrOf(c, 'id')
    if (el) out.add(el)
  })
  return out
}

/** A component under root (root included) by its model's cid, as the layers and the model's PbNode keys name it. */
export function byCid(root: Component | null | undefined, cid: string): Component | undefined {
  let hit: Component | undefined
  root?.onAll((c: Component) => {
    if (!hit && c.cid === cid) hit = c
  })
  return hit
}

/** A block by its id (countdowns name the blocks they reveal and hide). */
export function findById(root: Component | null | undefined, id: string): Component | undefined {
  let hit: Component | undefined
  root?.onAll((c: Component) => {
    if (!hit && attrOf(c, 'data-pb-id') === id) hit = c
  })
  return hit
}

/**
 * Gives a copied block and its parts ids of their own: duplicates must not
 * share an id, or a countdown would reveal both. An anchor id that was the
 * block's id follows it.
 */
export function freshIds(c: Component, taken: Set<string>) {
  c.onAll((x: Component) => {
    const id = attrOf(x, 'data-pb-id')
    if (!id) {
      if (!isTop(x) && !liveTagOf(x)) return
      const next = freeId(`b-${kindOf(x)}`, taken)
      taken.add(next)
      x.addAttributes({ 'data-pb-id': next })
      return
    }
    if (!taken.has(id)) {
      taken.add(id)
      return
    }
    const next = freeId(id, taken)
    taken.add(next)
    const patch: Record<string, string> = { 'data-pb-id': next }
    if (attrOf(x, 'id') === id) patch.id = next
    x.addAttributes(patch)
  })
}

/** The components on the way from the top-level block down to `c` (both included). */
export function pathTo(c: Component): Component[] {
  const out: Component[] = []
  let cur: Component | undefined = c
  while (cur && !isWrapper(cur)) {
    out.unshift(cur)
    cur = cur.parent()
  }
  return out
}
