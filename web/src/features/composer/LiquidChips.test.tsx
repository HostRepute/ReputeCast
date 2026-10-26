import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Component, Editor } from 'grapesjs'
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ComposerCtx } from './context'
import { chipify } from './editor/liquid'
import { headlessMjmlEditor } from './editor/testEditor'
import { LiquidPopover } from './LiquidChips'
import { testCampaign, testComposer } from './testing'

// CO-4: a chip's fallback editor (Composer board): the field, its fallback,
// how many lack it, what a contact sees, and the Liquid it saves as.

const campaign = testCampaign({ audienceCount: 31192 })
const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit', 'subscribers:view'] })

function preview(tokens: { key: string; token: string; value: string; missing?: boolean; audienceMissing?: number }[]) {
  return {
    subject: 'Hi',
    preheader: '',
    html: '<p>Hi</p>',
    text: 'Hi',
    fromName: 'Acme',
    fromEmail: 'news@acme.io',
    contact: { id: 'c1', name: 'Sam Rivera', email: 'sam.rivera@outlook.com', firstName: 'Sam', lastName: 'Rivera', label: '', language: '' },
    tokens: tokens.map((t) => ({ fallbackUsed: false, perRecipient: false, unknown: false, missing: false, ...t })),
    counted: 31192,
    problems: [],
    renderedAt: '2026-10-25T09:40:00Z',
    language: '',
    languageReason: '',
  }
}

let editor: Editor
let unmock: (() => void) | undefined

beforeAll(async () => {
  ;({ editor } = await headlessMjmlEditor(
    '<mjml><mj-body><mj-section><mj-column><mj-text>Hi {{ first_name | default: "there" }}, and {{ frist_name }}.</mj-text></mj-column></mj-section></mj-body></mjml>',
  ))
  chipify(editor)
})
afterAll(() => editor.destroy())
afterEach(() => unmock?.())

const chips = () => editor.getWrapper()!.findType('rc-liquid') as Component[]

async function open(chip: Component) {
  const host = document.createElement('div')
  await renderInWorkspace(
    <ComposerCtx.Provider value={testComposer({ campaign, editor, variantContent: () => ({ subject: 'Hi', preheader: '', html: '<p>Hi</p>' }) })}>
      <LiquidPopover host={host} tick={0} boxOf={() => ({ top: 10, left: 10, width: 60, height: 20 })} />
    </ComposerCtx.Provider>,
    { me },
  )
  act(() => {
    editor.trigger('rc:liquid:open', chip)
  })
  return screen.getByRole('dialog', { name: 'Personalization' })
}

describe('fallback editor', () => {
  it('says who lacks the field and what they see, and saves the fallback as Liquid', async () => {
    unmock = mockApi({
      [`POST /api/v1/workspaces/acme-newsletter/campaigns/${campaign.id}/preview`]: () =>
        json(preview([{ key: 'first_name', token: '{{ first_name | default: "there" }}', value: 'Sam', audienceMissing: 412 }])),
    })
    const dialog = await open(chips()[0]!)
    expect(within(dialog).getByLabelText('Field')).toHaveValue('first_name')
    expect(within(dialog).getByLabelText(/Fallback/)).toHaveValue('there')
    expect(await within(dialog).findByText('412 of 31,192 recipients have no First name and see “there”.')).toBeInTheDocument()
    expect(dialog).toHaveTextContent('Sam Rivera sees Hi Sam,')
    expect(dialog).toHaveTextContent('Without first_name: Hi there,')
    fireEvent.change(within(dialog).getByLabelText(/Fallback/), { target: { value: 'friend' } })
    expect(within(dialog).getByText('{{ first_name | default: "friend" }}')).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Done' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(editor.getHtml()).toContain('Hi {{ first_name | default: "friend" }}, and {{ frist_name }}.')
  })

  it('flags a field that is not a contact field, and offers the one meant', async () => {
    unmock = mockApi({ [`POST /api/v1/workspaces/acme-newsletter/campaigns/${campaign.id}/preview`]: () => json(preview([])) })
    const dialog = await open(chips()[1]!)
    const note = within(dialog).getByRole('note')
    expect(note).toHaveTextContent("frist_name isn't a contact field. Did you mean first_name? It prints nothing, for everyone.")
    fireEvent.click(within(note).getByRole('button', { name: 'Use first_name' }))
    expect(within(dialog).getByLabelText('Field')).toHaveValue('first_name')
    expect(within(dialog).queryByRole('note')).toBeNull()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Done' }))
    await waitFor(() => expect(editor.getHtml()).toContain('and {{ first_name }}.'))
  })

  it('removes the token', async () => {
    unmock = mockApi({ [`POST /api/v1/workspaces/acme-newsletter/campaigns/${campaign.id}/preview`]: () => json(preview([])) })
    const before = chips().length
    const dialog = await open(chips()[1]!)
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove' }))
    expect(chips()).toHaveLength(before - 1)
    expect(editor.getHtml()).toContain('and .')
  })
})
