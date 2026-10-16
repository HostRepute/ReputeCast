import type { Component } from 'grapesjs'
import { useMemo } from 'react'
import { useComposer } from '@/features/composer/context'
import { effective, headDefaults, ownAttrs, type HeadDefaults } from '@/features/composer/editor/attrs'
import { tagOf } from '@/features/composer/editor/labels'
import { parseColor } from '../contrast'
import { roleOf } from './TextRole'

/** A text the contrast is checked for: its color, size and weight. */
export interface TextSample {
  color: string
  sizePx: number
  bold: boolean
  /** "button" for a button's label, "heading" or "text" for a Text block. */
  kind: 'button' | 'heading' | 'text'
}

/**
 * What a color field is compared with (5.12). A text color is checked
 * against the background behind it, or can't be checked over an image; a
 * background against every text color on it, the lowest first.
 */
export type ContrastSpec =
  | { field: 'text'; text: Omit<TextSample, 'color'>; background?: string; image?: string; others: string[] }
  | { field: 'background'; texts: TextSample[]; others: string[] }

function px(v: string, fallback = 13): number {
  const m = /^(\d+(?:\.\d+)?)px$/.exec(v.trim())
  return m ? Number(m[1]) : fallback
}

function isBold(v: string): boolean {
  return v === 'bold' || v === 'bolder' || Number(v) >= 600
}

function walk(c: Component, fn: (x: Component) => void) {
  fn(c)
  c.components().forEach((x: Component) => walk(x, fn))
}

/** A color only when it's one (an image or "inherit" isn't). */
function colorOr(v: string): string | undefined {
  return v && parseColor(v) ? v : undefined
}

/**
 * The background a component's text sits on: its own container background,
 * then its column's, section's, wrapper's and the body's; white without
 * any. `image` names the background image when there is one in between.
 */
export function backgroundOf(c: Component, head: HeadDefaults): { color?: string; image?: string } {
  let cur: Component | undefined = c
  while (cur) {
    const tag = tagOf(cur)
    const own = ownAttrs(cur)
    if (tag === 'mj-text' || tag === 'mj-button' || tag === 'mj-image') {
      const v = colorOr(effective(cur, 'container-background-color', head))
      if (v) return { color: v }
    } else if (tag === 'mj-column' || tag === 'mj-group') {
      const v = colorOr(effective(cur, 'inner-background-color', head)) ?? colorOr(effective(cur, 'background-color', head))
      if (v) return { color: v }
    } else if (tag === 'mj-section' || tag === 'mj-wrapper' || tag === 'mj-hero') {
      const image = own['background-url']
      if (image) return { image: image.split('/').pop()?.split('?')[0] ?? image }
      const v = colorOr(effective(cur, 'background-color', head))
      if (v) return { color: v }
    } else if (tag === 'mj-body') {
      return { color: colorOr(effective(cur, 'background-color', head)) ?? '#FFFFFF' }
    }
    cur = cur.parent()
  }
  return { color: '#FFFFFF' }
}

/** A Text block's or a button's text: color, size, weight. */
function sampleOf(c: Component, head: HeadDefaults): TextSample {
  const tag = tagOf(c)
  const kind = tag === 'mj-button' ? 'button' : roleOf(c).role !== 'p' ? 'heading' : 'text'
  return {
    color: effective(c, 'color', head) || '#000000',
    sizePx: px(effective(c, 'font-size', head)),
    bold: isBold(effective(c, 'font-weight', head)) || kind === 'heading',
    kind,
  }
}

/** The background colors of the email's buttons other than `c`, for the suggestions. */
function otherButtons(c: Component, head: HeadDefaults): string[] {
  let root = c
  while (root.parent()) root = root.parent()!
  const out: string[] = []
  walk(root, (x) => {
    if (x !== c && tagOf(x) === 'mj-button') {
      const v = colorOr(effective(x, 'background-color', head))
      if (v) out.push(v)
    }
  })
  return out
}

/**
 * The contrast a color field of `c` is checked for, or undefined when the
 * field doesn't color text or what's behind it.
 */
export function contrastSpec(c: Component, attr: string, head: HeadDefaults): ContrastSpec | undefined {
  const tag = tagOf(c)
  if (attr === 'color' && (tag === 'mj-text' || tag === 'mj-button')) {
    const s = sampleOf(c, head)
    const text = { sizePx: s.sizePx, bold: s.bold, kind: s.kind }
    if (tag === 'mj-button') {
      const bg = colorOr(effective(c, 'background-color', head))
      return { field: 'text', text, background: bg ?? backgroundOf(c.parent() ?? c, head).color, others: [] }
    }
    const bg = backgroundOf(c, head)
    return { field: 'text', text, background: bg.color, image: bg.image, others: [] }
  }
  if (attr === 'background-color' && tag === 'mj-button') {
    return { field: 'background', texts: [sampleOf(c, head)], others: otherButtons(c, head) }
  }
  if ((attr === 'background-color' || attr === 'container-background-color') && /^mj-(section|wrapper|column|group|text|body|hero)$/.test(tag)) {
    // Every text on this background: texts whose nearest background is it.
    const texts: TextSample[] = []
    walk(c, (x) => {
      if (tagOf(x) !== 'mj-text') return
      let cur: Component | undefined = x
      while (cur && cur !== c) {
        if (cur !== x || attr !== 'container-background-color') {
          const own = ownAttrs(cur)
          const t = tagOf(cur)
          const covering =
            (t === 'mj-text' && colorOr(own['container-background-color'] ?? '')) ||
            ((t === 'mj-column' || t === 'mj-group') && (colorOr(own['inner-background-color'] ?? '') || colorOr(own['background-color'] ?? ''))) ||
            ((t === 'mj-section' || t === 'mj-wrapper') && (own['background-url'] || colorOr(own['background-color'] ?? '')))
          if (covering) return
        }
        cur = cur.parent()
      }
      texts.push(sampleOf(x, head))
    })
    return texts.length ? { field: 'background', texts, others: [] } : undefined
  }
  return undefined
}

/** The spec of a field, re-read as the head changes. */
export function useContrastSpec(c: Component, attr: string): ContrastSpec | undefined {
  const { head, rev } = useComposer()
  const defaults = useMemo(() => headDefaults(head), [head])
  // rev covers attribute edits elsewhere in the email (a button's text color).
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => contrastSpec(c, attr, defaults), [c, attr, defaults, rev])
}
