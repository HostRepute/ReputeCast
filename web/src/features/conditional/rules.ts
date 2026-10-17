import type { Component } from 'grapesjs'
import type { Schemas } from '@/api/client'
import { markerOf, sectionLabel, sectionOf, tagOf } from '@/features/composer/editor/labels'
import { ATTR_NAME, ATTR_VISIBLE } from './mjml'

export type VisibilityRule = Schemas['VisibilityRule']
export type VisibilityCondition = Schemas['VisibilityCondition']

/** Elements Visibility is offered on: sections and content blocks (not columns or groups). */
const OFFERED = new Set(['mj-section', 'mj-wrapper', 'mj-hero', 'mj-text', 'mj-image', 'mj-button', 'mj-divider', 'mj-spacer', 'mj-social', 'mj-navbar', 'mj-raw', 'mj-table'])

export function offered(c: Component | null | undefined): boolean {
  return !!c && OFFERED.has(tagOf(c))
}

/** The footer (its rc:footer marker or its label, as the server reads it): always shown. */
export function isFooter(c: Component | null | undefined): boolean {
  const sec = sectionOf(c)
  return !!sec && (markerOf(sec) === 'footer' || sectionLabel(sec) === 'footer')
}

const ID = /^[a-z0-9]{1,24}$/

/** A fresh block id: six characters of [a-z0-9]. */
export function newRuleId(): string {
  const alphabet = 'abcdefghijklmnopqrstuvwxyz0123456789'
  const b = new Uint8Array(6)
  crypto.getRandomValues(b)
  return Array.from(b, (x) => alphabet[x % alphabet.length]).join('')
}

function attrs(c: Component): Record<string, unknown> {
  return (c.getAttributes() ?? {}) as Record<string, unknown>
}

/** A component's rule, null without one (or one that doesn't read). */
export function ruleOf(c: Component | null | undefined): VisibilityRule | null {
  if (!c) return null
  const raw = attrs(c)[ATTR_VISIBLE]
  if (typeof raw !== 'string' || !raw.trim()) return null
  try {
    const r = JSON.parse(raw) as Partial<VisibilityRule>
    if (!r.id || !ID.test(r.id)) return null
    return { id: r.id, show: r.show ?? 'everyone', match: r.match ?? 'all', conditions: r.conditions ?? [] }
  } catch {
    return null
  }
}

/** Whether a rule leaves some contacts out. */
export function conditional(r: VisibilityRule | null): r is VisibilityRule {
  if (!r) return false
  if (r.show === 'nobody') return true
  return (r.show === 'only' || r.show === 'except') && r.conditions.length > 0
}

/** Writes a rule on a component (null removes it), one undo step. */
export function setRule(c: Component, r: VisibilityRule | null) {
  if (!r || r.show === 'everyone') {
    const next = { ...attrs(c) }
    if (!(ATTR_VISIBLE in next)) return
    delete next[ATTR_VISIBLE]
    c.setAttributes(next)
    return
  }
  c.addAttributes({ [ATTR_VISIBLE]: JSON.stringify({ id: r.id, show: r.show, match: r.match ?? 'all', conditions: r.conditions }) })
}

/** The block's own name in layers (rc-name), "" without one. */
export function nameOf(c: Component | null | undefined): string {
  if (!c) return ''
  const v = attrs(c)[ATTR_NAME]
  return typeof v === 'string' ? v : ''
}

export function setName(c: Component, name: string) {
  const v = name.trim()
  if (!v) {
    const next = { ...attrs(c) }
    if (!(ATTR_NAME in next)) return
    delete next[ATTR_NAME]
    c.setAttributes(next)
    return
  }
  c.addAttributes({ [ATTR_NAME]: v })
}

/** Every component with a conditional rule, in document order. */
export function conditionalComponents(root: Component | null | undefined): { c: Component; rule: VisibilityRule }[] {
  const out: { c: Component; rule: VisibilityRule }[] = []
  if (!root) return out
  const walk = (x: Component) => {
    const r = ruleOf(x)
    if (offered(x) && conditional(r) && !isFooter(x)) out.push({ c: x, rule: r })
    x.components().forEach(walk)
  }
  walk(root)
  return out
}

/** The component whose rule has this id. */
export function componentWithRule(root: Component | null | undefined, id: string): Component | null {
  let found: Component | null = null
  const walk = (x: Component) => {
    if (found) return
    if (ruleOf(x)?.id === id) found = x
    x.components().forEach(walk)
  }
  if (root) walk(root)
  return found
}
