import type { Component } from 'grapesjs'
import { useId } from 'react'
import { useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { pathOf } from '@/features/composer/editor/labels'
import { m } from './messages'
import { useAccessibility } from './state'

type Role = 'p' | 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6'

const BLOCK = /^(p|div|h[1-6])$/i

/**
 * The element a Text block's words sit in: a paragraph or a heading level.
 * Several paragraphs make it mixed: a heading gets its own Text block.
 */
export function roleOf(c: Component): { role: Role; mixed: boolean } {
  const kids = (c.components().models as Component[]).filter((k) => k.get('type') !== 'textnode' || String(k.get('content') ?? '').trim())
  if (!kids.length) {
    // Text kept as a content string.
    const doc = new DOMParser().parseFromString(`<div>${String(c.get('content') ?? '')}</div>`, 'text/html')
    const els = Array.from(doc.body.firstElementChild?.children ?? [])
    const blocks = els.filter((e) => BLOCK.test(e.tagName))
    if (!blocks.length) return { role: 'p', mixed: false }
    if (blocks.length > 1 || els.length > 1) return { role: 'p', mixed: true }
    const tag = blocks[0]!.tagName.toLowerCase()
    return { role: (tag === 'div' ? 'p' : tag) as Role, mixed: false }
  }
  const blocks = kids.filter((k) => BLOCK.test(String(k.get('tagName') ?? '')))
  if (!blocks.length) return { role: 'p', mixed: false }
  if (blocks.length > 1 || kids.length > 1) return { role: 'p', mixed: true }
  const tag = String(blocks[0]!.get('tagName')).toLowerCase()
  return { role: (tag === 'div' ? 'p' : tag) as Role, mixed: false }
}

/** Settings › Text role (5.11): Paragraph or a heading level; the text keeps its look. */
export function TextRoleProp({ c }: { c: Component }) {
  const a = useAccessibility()
  const composer = useComposer()
  const t = useT()
  const id = useId()
  if (!a?.enabled) return null
  const { role, mixed } = roleOf(c)
  const levels: Role[] = ['h1', 'h2', 'h3']
  if (!levels.includes(role) && role !== 'p') levels.push(role)
  return (
    <div className="prop" role="group" aria-labelledby={`${id}-t`}>
      <label className="t" id={`${id}-t`} htmlFor={id}>
        {t(m.textRole)}
      </label>
      <select
        id={id}
        className="in"
        value={role}
        disabled={mixed || !composer.canEdit}
        aria-describedby={`${id}-h`}
        onChange={(e) => void a.transform({ kind: 'text_role', path: pathOf(c), role: e.target.value as Role })}
      >
        <option value="p">{t(m.roleParagraph)}</option>
        {levels.map((l) => (
          <option key={l} value={l}>
            {t(m.roleHeading, { level: Number(l.slice(1)) })}
          </option>
        ))}
      </select>
      <span className="k" id={`${id}-h`}>
        {mixed ? t(m.roleMixed) : t(m.roleHelp)}
      </span>
      <span className="k">{t(m.roleDefaults)}</span>
    </div>
  )
}
