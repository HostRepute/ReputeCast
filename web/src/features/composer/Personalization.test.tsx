import { fireEvent, render, screen } from '@testing-library/react'
import type { Editor } from 'grapesjs'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import type { Schemas } from '@/api/client'
import { ComposerCtx } from './context'
import { headlessMjmlEditor } from './editor/testEditor'
import { replaceField } from './fixes'
import { PersonalizationCard, fallbackOf, personalizationBadge } from './Personalization'
import { testComposer } from './testing'

// CO-5: the preview's personalization says what each field prints, flags
// fields that aren't contact fields (with a fix), and counts the audience.

type Preview = Schemas['ComposerPreview']
type Token = Preview['tokens'][number]

const token = (t: Partial<Token> & { key: string; token: string }): Token => ({ value: '', missing: false, fallbackUsed: false, perRecipient: false, unknown: false, ...t }) as Token

function preview(tokens: Token[], more: Partial<Preview> = {}): Preview {
  return { subject: 'Hi there', preheader: '', html: '', text: '', fromName: 'Acme', fromEmail: 'news@acme.io', tokens, problems: [], renderedAt: '2026-10-25T09:40:00Z', language: '', languageReason: '', ...more } as Preview
}

let editor: Editor

beforeAll(async () => {
  ;({ editor } = await headlessMjmlEditor(
    '<mjml><mj-body><mj-section><mj-column><mj-text>Hi {{ frist_name | default: "there" }}, and {{frist_name}}. Not {{ frist_names }}.</mj-text><mj-button href="https://acme.io">Go {{ frist_name }}</mj-button></mj-column></mj-section></mj-body></mjml>',
  ))
})

afterAll(() => editor.destroy())

describe('personalization card', () => {
  it('flags a field that is not a contact field, and fixes it everywhere', () => {
    const setSubject = vi.fn()
    const c = testComposer({ subject: 'Hello {{ frist_name | default: "there" }}', setSubject, editor })
    const tokens = [
      token({ key: 'frist_name', token: '{{ frist_name | default: "there" }}', value: 'there', missing: true, fallbackUsed: true, unknown: true, suggestion: 'first_name' }),
      token({ key: 'last_name', token: '{{ last_name }}', value: 'Nguyen', audienceMissing: 1 }),
    ]
    render(
      <ComposerCtx.Provider value={c}>
        <PersonalizationCard preview={preview(tokens, { counted: 3 })} />
      </ComposerCtx.Provider>,
    )
    expect(screen.getByText('1 unknown field')).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent("frist_name isn't a contact field (did you mean first_name?) Everyone sees “there” there, whatever their contact fields say.")
    expect(screen.getByText('1 of 3 recipients has no last_name; it prints nothing for them.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Replace with first_name' }))
    expect(setSubject).toHaveBeenCalledWith('Hello {{ first_name | default: "there" }}')
    const html = editor.getHtml()
    expect(html).toContain('Hi {{ first_name | default: "there" }}, and {{first_name}}.')
    // Only the whole name: frist_names stays.
    expect(html).toContain('Not {{ frist_names }}.')
    expect(html).toContain('Go {{ first_name }}')
  })

  it('counts the audience per field, and says when segments are left out', () => {
    const tokens = [token({ key: 'first_name', token: '{{ first_name | default: "there" }}', value: 'Sam', audienceMissing: 412 })]
    render(
      <ComposerCtx.Provider value={testComposer()}>
        <PersonalizationCard preview={preview(tokens, { counted: 31192, countedListsOnly: true })} />
      </ComposerCtx.Provider>,
    )
    expect(screen.getByText('No missing fields')).toBeInTheDocument()
    expect(screen.getByText('412 of 31,192 recipients have no first_name and will see “there”.')).toBeInTheDocument()
    expect(screen.getByText(/segments aren't in these numbers/)).toBeInTheDocument()
  })
})

describe('personalizationBadge', () => {
  it('counts fields this contact lacks with nothing to print', () => {
    expect(personalizationBadge([token({ key: 'last_name', token: '{{ last_name }}', missing: true })])).toEqual({ tone: 'warn', unknown: 0, missing: 1 })
    expect(personalizationBadge([token({ key: 'first_name', token: '{{ first_name | default: "there" }}', missing: true, fallbackUsed: true })])).toEqual({ tone: 'ok', unknown: 0, missing: 0 })
    expect(personalizationBadge([token({ key: 'unsubscribe_url', token: '{{ unsubscribe_url }}', perRecipient: true })]).tone).toBe('ok')
  })
})

describe('replaceField', () => {
  it('reports how many places changed', () => {
    const c = testComposer({ subject: 'No tokens', preheader: '{{ plann }}', editor })
    expect(replaceField(c, 'plann', 'plan')).toBe(1)
    expect(c.setPreheader).toHaveBeenCalledWith('{{ plan }}')
    expect(c.setSubject).not.toHaveBeenCalled()
  })
})

describe('fallbackOf', () => {
  it('reads the default filter', () => {
    expect(fallbackOf('{{ first_name | default: "there" }}')).toBe('there')
    expect(fallbackOf("{{ plan | default: 'free' }}")).toBe('free')
    expect(fallbackOf('{{ last_name }}')).toBe('')
  })
})
