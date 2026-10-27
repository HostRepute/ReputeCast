import type { Component } from 'grapesjs'
import { t as tt } from '@/i18n'
import { mh } from './chrome.messages'
import { mjChildren, sectionName, tagOf } from './editor/labels'

/**
 * How the composer names a selection for writers (Composer board): a
 * section is "Section · 2 columns" with its name ("Feature highlights"), a
 * column is "Column 2", anything else its kind ("Image", "Button").
 */
export function friendlyLabel(c: Component): { head: string; name?: string } {
  const tag = tagOf(c)
  if (!tag) return { head: '' }
  const parent = c.parent()
  if (tagOf(parent) === 'mj-body' && tag !== 'mj-raw') {
    const cols = mjChildren(c).filter((x) => tagOf(x) === 'mj-column').length
    return { head: tt(mh.headSection, { tag: tag.replace(/-/g, '_'), count: cols > 1 ? cols : 0 }), name: sectionName(c) }
  }
  if (tag === 'mj-column' && parent) {
    const n = mjChildren(parent).filter((x) => tagOf(x) === 'mj-column').indexOf(c) + 1
    return { head: tt(mh.crumbColumn, { n: Math.max(n, 1) }) }
  }
  return { head: tt(mh.crumbTag, { tag: tag.replace(/-/g, '_') }) }
}
