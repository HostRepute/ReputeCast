import type { Component, Editor } from 'grapesjs'
import { sectionLabel, sectionOf, tagOf } from './labels'

/**
 * Turns canvas edits into the short descriptions shown in the version
 * history ("2-column section padding", "hero copy", "added footer").
 */
export interface ChangeTracker {
  /** Distinct descriptions since the last take(), at most three. */
  take(): string
  /** The same descriptions without clearing them (the leave dialog names them). */
  peek(): string[]
  /** True when a section or element was added or removed since the last take(). */
  structural(): boolean
  note(label: string): void
  dispose(): void
}

function sectionPhrase(c: Component | undefined): string {
  const sec = sectionOf(c)
  if (!sec) return 'email'
  const label = sectionLabel(sec)
  const m = /^(\d+) columns$/.exec(label)
  if (m) return `${m[1]}-column section`
  if (label === 'text') return 'text section'
  return label
}

function attrPhrase(keys: string[]): string {
  const k = keys.join(' ')
  if (/padding/.test(k)) return 'padding'
  if (/background/.test(k)) return 'background'
  if (/\bwidth\b|\bheight\b/.test(k)) return 'widths'
  if (/font|line-height|letter-spacing|text-transform|font-weight/.test(k)) return 'typography'
  if (/\bcolor\b/.test(k)) return 'colours'
  if (/href/.test(k)) return 'links'
  if (/\balt\b/.test(k)) return 'alt text'
  if (/\bsrc\b/.test(k)) return 'image'
  if (/align/.test(k)) return 'alignment'
  if (/border/.test(k)) return 'borders'
  if (/css-class/.test(k)) return 'CSS class'
  return 'settings'
}

export function trackChanges(editor: Editor): ChangeTracker {
  const labels: string[] = []
  let structural = false
  const add = (l: string) => {
    if (!labels.includes(l)) labels.push(l)
  }
  const onAttrs = (c: Component) => {
    const prev = (c.previous('attributes') ?? {}) as Record<string, unknown>
    const next = (c.get('attributes') ?? {}) as Record<string, unknown>
    const keys = Array.from(new Set([...Object.keys(prev), ...Object.keys(next)])).filter((k) => prev[k] !== next[k])
    if (!keys.length) return
    const what = attrPhrase(keys)
    const tag = tagOf(c)
    if (tag === 'mj-column' && /width/.test(keys.join(' '))) {
      add(editor.getDevice() === 'mobile' ? 'mobile column widths' : 'column widths')
      return
    }
    add(`${sectionPhrase(c)} ${what}`)
  }
  const onContent = (c: Component) => {
    const tag = tagOf(c) || tagOf(c.parent())
    if (tag === 'mj-button') add(`${sectionPhrase(c)} button text`)
    else add(`${sectionPhrase(c)} copy`)
  }
  const onAdd = (c: Component) => {
    if (!tagOf(c)) return
    structural = true
    if (tagOf(c.parent()) === 'mj-body') add(`added ${sectionLabel(c)}`)
  }
  const onRemove = (c: Component) => {
    if (!tagOf(c)) return
    structural = true
    add(`removed ${tagOf(c).replace(/^mj-/, '')}`)
  }
  const attrHandler = (c: Component) => onAttrs(c)
  const contentHandler = (c: Component) => onContent(c)
  editor.on('component:update:attributes', attrHandler)
  editor.on('component:update:content', contentHandler)
  editor.on('component:update:components', contentHandler)
  editor.on('component:add', onAdd)
  editor.on('component:remove', onRemove)
  return {
    take() {
      const out = labels.slice(0, 3).join(', ')
      labels.length = 0
      structural = false
      return out
    },
    peek: () => labels.slice(0, 3),
    structural: () => structural,
    note: add,
    dispose() {
      editor.off('component:update:attributes', attrHandler)
      editor.off('component:update:content', contentHandler)
      editor.off('component:update:components', contentHandler)
      editor.off('component:add', onAdd)
      editor.off('component:remove', onRemove)
    },
  }
}
