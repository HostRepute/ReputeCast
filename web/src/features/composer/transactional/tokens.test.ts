import { describe, expect, it } from 'vitest'
import { decorate, undecorate } from './tokens'
import type { Variable } from './types'

const v = (x: Partial<Variable> & { name: string }): Variable => ({ type: 'text', required: false, sensitive: false, ...x })

describe('canvas variables', () => {
  const ctx = {
    declared: new Map([v({ name: 'first_name', default: 'there' }), v({ name: 'reset_url', sensitive: true, required: true })].map((x) => [x.name, x])),
    data: { first_name: 'Sam' },
    locals: new Set<string>(),
  }
  it('shows chips and puts the Liquid back exactly', () => {
    const root = document.createElement('div')
    root.innerHTML = '<p>Hi {{ first_name }}, reset at {{reset_url}} or ask {{ company_name }}.</p><p>No tokens here</p>'
    const before = root.innerHTML
    decorate(root, ctx, (_v, src) => src)
    const chips = Array.from(root.querySelectorAll('span.rc-tok'))
    expect(chips.map((c) => [c.textContent, c.className])).toEqual([
      ['Sam', 'rc-tok'],
      ['reset_url', 'rc-tok e'],
      ['company_name', 'rc-tok e'],
    ])
    expect(chips[0]!.getAttribute('title')).toBe('{{ first_name }}')
    undecorate(root)
    expect(root.innerHTML).toBe(before)
  })
  it('leaves a text being edited alone', () => {
    const root = document.createElement('div')
    root.innerHTML = '<div contenteditable="true">Hi {{ first_name }}</div>'
    decorate(root, ctx, (_v, src) => src)
    expect(root.querySelectorAll('span.rc-tok')).toHaveLength(0)
  })
})
