import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Component, Editor } from 'grapesjs'
import { useEffect, useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ComposerCtx } from '../context'
import { headlessMjmlEditor } from '../editor/testEditor'
import { testComposer } from '../testing'
import { WriteView } from './WriteView'

// CO-9: Write mode over the canvas's components (ComposerWrite board).

const base = '/api/v1/workspaces/acme-newsletter/campaigns/01970000-0000-7000-8000-0000000000c1'
const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })

const email = `<mjml><mj-body>
<mj-section rc-synced="header"><mj-column><mj-image src="/media/logo.png" alt="Acme"></mj-image></mj-column></mj-section>
<mj-section><mj-column><mj-image src="/media/hero.png" alt="Q4 roadmap illustration"></mj-image></mj-column></mj-section>
<mj-section><mj-column><mj-text font-size="15px" color="#1F2937"><h1>Three launches</h1><p>Hi {{ first_name | default: "there" }}, here's what shipped.</p></mj-text></mj-column></mj-section>
<mj-section rc-name="Feature highlights"><mj-column><mj-text><h3>Feature flags</h3></mj-text><mj-button href="https://docs.acme.io/flags">Read the docs</mj-button></mj-column><mj-column><mj-text><h3>Audit log exports</h3></mj-text><mj-image src="/media/v1-to-v2-migration.png" width="600px"></mj-image></mj-column></mj-section>
<mj-section><mj-column><mj-social><mj-social-element name="x" href="https://x.com/acme">X</mj-social-element></mj-social></mj-column></mj-section>
<mj-section rc-synced="footer"><mj-column><mj-text>Acme Inc · <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

/** The composer bumps rev on every canvas change; so does this harness. */
function Harness({ editor, patch }: { editor: Editor; patch: object }) {
  const [rev, setRev] = useState(0)
  useEffect(() => {
    const bump = () => setRev((r) => r + 1)
    const events = 'component:add component:remove component:update'
    editor.on(events, bump)
    return () => void editor.off(events, bump)
  }, [editor])
  return (
    <ComposerCtx.Provider value={{ ...testComposer({ editor, ...patch }), rev }}>
      <WriteView />
    </ComposerCtx.Provider>
  )
}

async function open(editor: Editor, patch = {}) {
  unmock = mockApi({
    [`GET ${base}/comments`]: () => json({ threads: [], open: 0, resolved: 0 }),
    [`GET ${base}/mentionable`]: () => json({ items: [] }),
  })
  return renderInWorkspace(<Harness editor={editor} patch={patch} />, { me })
}

const texts = (ed: Editor) => ed.getWrapper()!.findType('mj-text') as Component[]

describe('write mode', () => {
  it('shows the email as a document, as the board does', async () => {
    const { editor } = await headlessMjmlEditor(email)
    await open(editor)
    const doc = screen.getByRole('article', { name: 'Email text' })
    expect(within(doc).getAllByText(/synced block/)).toHaveLength(2)
    expect(within(doc).getAllByRole('link', { name: 'Edit in Brand kit' })[0]).toHaveAttribute('href', '/w/acme-newsletter/settings/brand-kit')
    expect(within(doc).getByText('Hero image · Q4 roadmap illustration')).toBeInTheDocument()
    expect(within(doc).getByText('Alt text: Q4 roadmap illustration')).toBeInTheDocument()
    expect(within(doc).getByText('Feature highlights · column 1 of 2 in Design')).toBeInTheDocument()
    expect(within(doc).getByText('Feature highlights · column 2 of 2 in Design')).toBeInTheDocument()
    expect(within(doc).getByRole('textbox', { name: 'Button text' })).toHaveTextContent('Read the docs')
    expect(within(doc).getByText('docs.acme.io/flags')).toBeInTheDocument()
    // An image without alt text asks for it inline.
    expect(within(doc).getByText('No alt text')).toBeInTheDocument()
    expect(within(doc).getByLabelText('Alt text for v1-to-v2-migration.png')).toBeInTheDocument()
    // Blocks Write mode doesn't type in say where they change.
    expect(within(doc).getByRole('button', { name: 'Edit in Design' })).toBeInTheDocument()
    // Personalization shows as a chip.
    const chip = doc.querySelector('[data-rc-liquid]')!
    expect(chip.textContent).toBe('first_name · “there”')
  })

  it('writes typed text to the email, and keeps the rest as it was', async () => {
    const { editor } = await headlessMjmlEditor(email)
    const before = editor.getHtml()
    await open(editor)
    const box = screen.getAllByRole('textbox', { name: 'Text' })[0]!
    const p = box.querySelector('p')!
    p.append(' More soon.')
    fireEvent.input(box)
    fireEvent.blur(box)
    const after = editor.getHtml()
    expect(after).toContain("here's what shipped. More soon.")
    expect(after).toContain('{{ first_name | default: "there" }}')
    expect(after.replace(' More soon.', '')).toBe(before)
  })

  it('adds alt text inline, or marks the image decorative', async () => {
    const { editor } = await headlessMjmlEditor(email)
    await open(editor)
    const input = screen.getByLabelText('Alt text for v1-to-v2-migration.png')
    fireEvent.change(input, { target: { value: 'v1 endpoints and their v2 replacements' } })
    fireEvent.blur(input)
    const img = (editor.getWrapper()!.findType('mj-image') as Component[]).find((x) => String(x.getAttributes().src).includes('v1-to-v2'))!
    expect(img.getAttributes().alt).toBe('v1 endpoints and their v2 replacements')
    expect(screen.getByText('Alt text added')).toBeInTheDocument()
  })

  it('inserts blocks from the / menu (↑ ↓ Enter Esc)', async () => {
    const { editor } = await headlessMjmlEditor(email)
    await open(editor)
    const line = screen.getByRole('combobox', { name: 'Empty line. Type / to insert a block' })
    fireEvent.change(line, { target: { value: '/' } })
    const list = screen.getByRole('listbox', { name: 'Insert a block' })
    expect(within(list).getAllByRole('option').map((o) => o.querySelector('.sl')?.textContent)).toEqual([
      'Heading',
      'Bulleted list',
      'Numbered list',
      'Image',
      'Button',
      'Divider',
      'Article card',
      'Poll',
      'Personalization',
      'Conditional block',
    ])
    expect(line).toHaveAttribute('aria-activedescendant', expect.stringContaining('heading'))
    fireEvent.keyDown(line, { key: 'ArrowDown' })
    expect(line).toHaveAttribute('aria-activedescendant', expect.stringContaining('bullets'))
    fireEvent.keyDown(line, { key: 'ArrowUp' })
    fireEvent.keyDown(line, { key: 'ArrowUp' })
    expect(line).toHaveAttribute('aria-activedescendant', expect.stringContaining('cond'))
    // Filtering, and nothing found.
    fireEvent.change(line, { target: { value: '/zzz' } })
    expect(screen.getByText('No block matches “zzz”. Press Esc to keep it as text.')).toBeInTheDocument()
    fireEvent.keyDown(line, { key: 'Escape' })
    expect(line).toHaveValue('zzz')
    // A divider, at the end of the email (before the synced footer).
    fireEvent.change(line, { target: { value: '/div' } })
    fireEvent.keyDown(line, { key: 'Enter' })
    expect(editor.getHtml()).toMatch(/<mj-divider[^>]*><\/mj-divider><\/mj-column><\/mj-section><mj-section rc-synced="footer"/)
    expect(await screen.findByText('Divider inserted')).toBeInTheDocument()
    // A heading goes in as a text with a heading, styled like the texts around it.
    const line2 = screen.getByRole('combobox', { name: 'Empty line. Type / to insert a block' })
    fireEvent.change(line2, { target: { value: '/head' } })
    fireEvent.keyDown(line2, { key: 'Enter' })
    const added = texts(editor).find((x) => x.getInnerHTML().includes('<h2>'))!
    expect(added).toBeDefined()
    expect(added.getAttributes()['font-size']).toBe('15px')
  })

  it('turns words typed on the empty line into a paragraph', async () => {
    const { editor } = await headlessMjmlEditor(email)
    await open(editor)
    const line = screen.getByRole('combobox', { name: 'Empty line. Type / to insert a block' })
    fireEvent.change(line, { target: { value: 'One more thing.' } })
    fireEvent.keyDown(line, { key: 'Enter' })
    expect(texts(editor).some((x) => x.getInnerHTML() === '<p>One more thing.</p>')).toBe(true)
  })

  it('cleans up a paste, and undoes it', async () => {
    const { editor } = await headlessMjmlEditor(email)
    await open(editor)
    const box = screen.getAllByRole('textbox', { name: 'Text' })[1]!
    const text = texts(editor)[1]!
    const before = text.getInnerHTML()
    const html = '<b style="font-weight:normal" id="docs-internal-guid-1"><p style="color:red"><span style="font-weight:700">New</span> text</p></b>'
    fireEvent.paste(box, { clipboardData: { getData: (type: string) => (type === 'text/html' ? html : 'New text') } })
    expect(text.getInnerHTML()).toContain('<strong>New</strong> text')
    expect(text.getInnerHTML()).not.toContain('style=')
    expect(await screen.findByText('Pasted from Google Docs')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Undo paste' }))
    expect(text.getInnerHTML()).toBe(before)
    expect(await screen.findByText('Paste undone')).toBeInTheDocument()
  })

  it('moves a block with Alt and the arrow keys', async () => {
    const { editor } = await headlessMjmlEditor('<mjml><mj-body><mj-section><mj-column><mj-text><p>First</p></mj-text><mj-text><p>Second</p></mj-text></mj-column></mj-section></mj-body></mjml>')
    await open(editor)
    const second = screen.getAllByRole('textbox', { name: 'Text' })[1]!
    fireEvent.keyDown(second, { key: 'ArrowUp', altKey: true })
    expect(texts(editor).map((x) => x.getInnerHTML())).toEqual(['<p>Second</p>', '<p>First</p>'])
  })

  it('is read-only for people who can’t edit', async () => {
    const { editor } = await headlessMjmlEditor(email)
    const save = vi.fn()
    await open(editor, { canEdit: false, save })
    expect(screen.getAllByRole('textbox', { name: 'Text' })[0]).toHaveAttribute('contenteditable', 'false')
    expect(screen.getByRole('combobox', { name: 'Empty line. Type / to insert a block' })).toBeDisabled()
    await waitFor(() => expect(save).not.toHaveBeenCalled())
  })
})
