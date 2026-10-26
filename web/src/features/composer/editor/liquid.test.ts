import type { Component, Editor } from 'grapesjs'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildToken, chipHtml, chipify, parseToken, setChipLabels, splitTokens } from './liquid'
import { headlessMjmlEditor } from './testEditor'

// CO-4: personalization tokens show as chips on the canvas and export as
// their Liquid, unchanged.

describe('tokens', () => {
  it('read the field, the fallback and other filters', () => {
    expect(parseToken('{{ first_name | default: "there" }}')).toEqual({ key: 'first_name', fallback: 'there', filters: [] })
    expect(parseToken("{{plan|capitalize|default:'free'}}")).toEqual({ key: 'plan', fallback: 'free', filters: ['capitalize'] })
    expect(parseToken('{{- last_name -}}')).toEqual({ key: 'last_name', filters: [] })
    // A pipe inside the fallback stays in it.
    expect(parseToken('{{ city | default: "here | there" }}').fallback).toBe('here | there')
  })

  it('write back with the other filters kept, and no empty fallback', () => {
    expect(buildToken({ key: 'first_name', fallback: 'friend', filters: [] })).toBe('{{ first_name | default: "friend" }}')
    expect(buildToken({ key: 'plan', fallback: '', filters: ['capitalize'] })).toBe('{{ plan | capitalize }}')
    expect(buildToken({ key: 'nick', fallback: 'the "boss"', filters: [] })).toBe(`{{ nick | default: 'the "boss"' }}`)
  })

  it('split text around tokens', () => {
    expect(splitTokens('Hi {{ first_name }}, from {{ city | default: "here" }}!')).toEqual([
      { text: 'Hi ' },
      { token: '{{ first_name }}' },
      { text: ', from ' },
      { token: '{{ city | default: "here" }}' },
      { text: '!' },
    ])
    expect(splitTokens('No tokens')).toEqual([{ text: 'No tokens' }])
  })

  it('make the chip the text editor inserts', () => {
    const html = chipHtml('{{ first_name | default: "there" }}', { label: 'First name', fallback: 'there', unknown: false })
    expect(html).toContain('data-rc-liquid="{{ first_name | default: &quot;there&quot; }}"')
    expect(html).toContain('contenteditable="false"')
    expect(html).toContain('aria-label="Personalization: First name, fallback there. Edit"')
  })
})

describe('chips on the canvas', () => {
  let editor: Editor
  const body =
    '<mjml><mj-body><mj-section><mj-column>' +
    '<mj-text>Hi {{ first_name | default: "there" }}, your plan is {{ plan }}.</mj-text>' +
    '<mj-button href="{{ unsubscribe_url }}">Bye {{ first_name }}</mj-button>' +
    '<mj-raw><p>{{ raw_stays }}</p></mj-raw>' +
    '</mj-column></mj-section></mj-body></mjml>'

  beforeAll(async () => {
    ;({ editor } = await headlessMjmlEditor(body))
  })
  afterAll(() => editor.destroy())

  const chips = () => editor.getWrapper()!.findType('rc-liquid') as Component[]

  it('turn tokens in text into chips, and export them unchanged', () => {
    const before = editor.getHtml()
    chipify(editor)
    expect(chips().map((c) => c.get('token'))).toEqual(['{{ first_name | default: "there" }}', '{{ plan }}', '{{ first_name }}'])
    // Raw HTML keeps its text; attributes keep theirs.
    expect(editor.getHtml()).toBe(before)
    expect(editor.getHtml()).toContain('{{ raw_stays }}')
    // Not an edit: nothing to undo.
    expect(editor.UndoManager.hasUndo()).toBe(false)
    // Running again changes nothing.
    chipify(editor)
    expect(chips()).toHaveLength(3)
  })

  it('read as the workspace names its fields, unknown ones flagged', () => {
    setChipLabels(editor, (p) => ({ label: p.key === 'first_name' ? 'First name' : p.key, fallback: p.fallback, unknown: p.key === 'plan' }))
    const [first] = chips()
    expect(first!.get('token')).toBe('{{ first_name | default: "there" }}')
  })

  it('save an edited token as Liquid', () => {
    const [first] = chips()
    first!.set('token', buildToken({ ...parseToken(String(first!.get('token'))), fallback: 'friend' }))
    expect(editor.getHtml()).toContain('Hi {{ first_name | default: "friend" }}, your plan is {{ plan }}.')
    // An edit like any other: undoable.
    expect(editor.UndoManager.hasUndo()).toBe(true)
  })
})
