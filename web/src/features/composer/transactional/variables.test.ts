import { describe, expect, it } from 'vitest'
import type { Variable } from './types'
import { PREVIEW_TO } from './types'
import { localsOf, lookup, parseJsonObject, parseOutput, removeOutputs, renameInData, renameInSource, sampleData, tokenView, usesVariable } from './variables'

const v = (x: Partial<Variable> & { name: string }): Variable => ({ type: 'text', required: false, sensitive: false, ...x })

describe('parseOutput', () => {
  it('reads the variable, its path and the default filter', () => {
    expect(parseOutput('first_name | default: "there"')).toEqual({ root: 'first_name', path: [], fallback: 'there' })
    expect(parseOutput("items[0].name | upcase")).toEqual({ root: 'items', path: ['0', 'name'], fallback: undefined })
    expect(parseOutput('to.email')).toEqual({ root: 'to', path: ['email'], fallback: undefined })
    expect(parseOutput('"a literal" | upcase').root).toBe('')
  })
  it('keeps pipes inside quotes', () => {
    expect(parseOutput('x | default: "a | b"').fallback).toBe('a | b')
  })
})

describe('tokenView', () => {
  const declared = new Map(
    [
      v({ name: 'first_name', default: 'there' }),
      v({ name: 'reset_url', type: 'url', required: true, sensitive: true }),
      v({ name: 'expires_in', required: true }),
      v({ name: 'nickname' }),
      v({ name: 'items', type: 'list' }),
    ].map((x) => [x.name, x]),
  )
  const ctx = { declared, data: { reset_url: 'https://app.example.com/r/1', items: [{ name: 'Pro' }] }, locals: new Set(['item']) }
  it('prints the data, the filter default or the declared default', () => {
    expect(tokenView('reset_url', ctx)).toEqual({ text: 'https://app.example.com/r/1', tone: 's', name: 'reset_url' })
    expect(tokenView('first_name', ctx)).toEqual({ text: 'there', tone: '', name: 'first_name' })
    expect(tokenView('first_name | default: "friend"', ctx).text).toBe('friend')
    expect(tokenView('items.first.name', ctx).text).toBe('Pro')
    expect(tokenView('items.size', ctx).text).toBe('1')
  })
  it('marks undeclared and missing required variables red, and empty optional ones muted', () => {
    expect(tokenView('company_name', ctx)).toEqual({ text: 'company_name', tone: 'e', name: 'company_name' })
    expect(tokenView('expires_in', ctx).tone).toBe('e')
    expect(tokenView('nickname', ctx).tone).toBe('m')
  })
  it('leaves loop items and the recipient alone', () => {
    expect(tokenView('item.name', ctx)).toEqual({ text: 'item.name', tone: '', name: '' })
    expect(tokenView('to.email', ctx).text).toBe(PREVIEW_TO.email)
    expect(tokenView('to.name', ctx).text).toBe(PREVIEW_TO.name)
  })
})

describe('usesVariable', () => {
  it('finds outputs and tags, not other names that contain it', () => {
    expect(usesVariable('Hi {{ first_name | default: "there" }}', 'first_name')).toBe(true)
    expect(usesVariable('{% if plan == "pro" %}x{% endif %}', 'plan')).toBe(true)
    expect(usesVariable('{% for item in items %}{% endfor %}', 'items')).toBe(true)
    expect(usesVariable('{{ names }}', 'name')).toBe(false)
    expect(usesVariable('{{ user.name }}', 'name')).toBe(false)
  })
})

describe('renaming and removing', () => {
  it('renames outputs of the variable only', () => {
    expect(renameInSource('{{ name }} {{name | upcase}} {{ names }} {{ name.first }}', 'name', 'given_name')).toBe('{{ given_name }} {{given_name | upcase}} {{ names }} {{ given_name.first }}')
  })
  it('removes its outputs', () => {
    expect(removeOutputs('Thanks for joining {{ company_name }}. {{ first_name }}', 'company_name')).toBe('Thanks for joining . {{ first_name }}')
  })
  it('renames the key in every test data set', () => {
    const out = renameInData([{ name: 'New user', data: { a: 1, b: 2 } }, { name: 'Other', data: { b: 3 } }], 'a', 'z')
    expect(out[0]!.data).toEqual({ z: 1, b: 2 })
    expect(out[1]!.data).toEqual({ b: 3 })
  })
})

describe('locals, samples and JSON', () => {
  it('reads loop items, assigns and captures', () => {
    expect([...localsOf('{% for item in items %}{% assign total = 0 %}{%- capture greeting -%}{% endcapture %}')].sort()).toEqual(['greeting', 'item', 'total'])
  })
  it('turns samples into typed values', () => {
    expect(
      sampleData([
        v({ name: 'n', type: 'number', sample: '3' }),
        v({ name: 'b', type: 'boolean', sample: 'true' }),
        v({ name: 'l', type: 'list', sample: '[1,2]' }),
        v({ name: 'empty' }),
        v({ name: 't', sample: 'Sam' }),
      ]),
    ).toEqual({ n: 3, b: true, l: [1, 2], t: 'Sam' })
  })
  it('says where JSON breaks, and that a set is an object', () => {
    const bad = parseJsonObject('{\n  "a": 1,\n  "b": \n}')
    expect(bad.ok).toBe(false)
    if (!bad.ok) expect(bad.line).toBeGreaterThanOrEqual(3)
    const arr = parseJsonObject('[1]')
    expect(arr.ok === false && arr.notObject).toBe(true)
    expect(parseJsonObject('{"a": 1}')).toEqual({ ok: true, data: { a: 1 } })
  })
  it('looks up nested values', () => {
    expect(lookup({ a: { b: [{ c: 'x' }] } }, ['a', 'b', '0', 'c'])).toBe('x')
    expect(lookup({ a: 'text' }, ['a', 'size'])).toBe(4)
  })
})
