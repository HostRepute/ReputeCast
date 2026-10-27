import { fireEvent, render, screen } from '@testing-library/react'
import type { Editor } from 'grapesjs'
import { useState } from 'react'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { ComposerCtx, type ComposerState } from './context'
import { tagOf } from './editor/labels'
import { headlessMjmlEditor } from './editor/testEditor'
import { LayersTab } from './LeftPanel'
import { testComposer } from './testing'

// CO-8: Layers is a keyboard tree (↑ ↓ → ← Home End, Enter selects on the email).

let editor: Editor

beforeAll(async () => {
  ;({ editor } = await headlessMjmlEditor(
    '<mjml><mj-body>' +
      '<mj-section><mj-column><mj-text>Weekly digest</mj-text></mj-column></mj-section>' +
      '<mj-section><mj-column><mj-text>Feature flags</mj-text></mj-column><mj-column><mj-text>Audit log</mj-text></mj-column></mj-section>' +
      '</mj-body></mjml>',
  ))
})

afterAll(() => editor.destroy())

/** The composer around the tree: the selection follows the editor's. */
function Harness({ base }: { base: ComposerState }) {
  const [selected, setSelected] = useState(editor.getSelected() ?? null)
  const select = editor.select.bind(editor)
  const ed = Object.assign(Object.create(editor) as Editor, {
    select: (c: Parameters<Editor['select']>[0]) => {
      const r = select(c)
      setSelected(editor.getSelected() ?? null)
      return r
    },
  })
  return (
    <ComposerCtx.Provider value={{ ...base, editor: ed, selected }}>
      <LayersTab />
    </ComposerCtx.Provider>
  )
}

const rows = () => screen.getAllByRole('treeitem')
const focused = () => document.activeElement as HTMLElement
const press = (key: string) => fireEvent.keyDown(focused(), { key })

describe('layer tree', () => {
  it('is a tree with one row in the tab order', () => {
    editor.select(undefined)
    render(<Harness base={testComposer()} />)
    expect(screen.getByRole('tree', { name: 'Layers of the email' })).toBeInTheDocument()
    const tabbable = rows().filter((r) => r.tabIndex === 0)
    expect(tabbable).toHaveLength(1)
    expect(rows()[0]).toHaveAttribute('aria-level', '1')
    expect(rows()[0]).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(/to select it on the email/)).toBeInTheDocument()
  })

  it('moves with the arrow keys, opens and closes rows, and selects with Enter', () => {
    editor.select(undefined)
    const setView = vi.fn()
    render(<Harness base={testComposer({ setView })} />)
    // mjml › head, mj-body › two sections (closed).
    rows()[0]!.focus()
    expect(rows().map((r) => r.getAttribute('aria-level'))).toEqual(['1', '2', '2', '3', '3'])
    press('ArrowDown')
    expect(focused().textContent).toContain('mj-head')
    press('ArrowDown')
    expect(focused().textContent).toContain('mj-body')
    press('ArrowDown')
    const second = rows()[4]!
    press('ArrowDown')
    expect(focused()).toBe(second)
    expect(second).toHaveAttribute('aria-expanded', 'false')
    // → opens the section; → again goes to its first column.
    press('ArrowRight')
    expect(second).toHaveAttribute('aria-expanded', 'true')
    press('ArrowRight')
    expect(focused().textContent).toContain('mj-column')
    expect(focused()).toHaveAttribute('aria-level', '4')
    // ← on a closed row goes up to its parent; ← again closes it.
    press('ArrowLeft')
    expect(focused()).toBe(second)
    press('ArrowLeft')
    expect(second).toHaveAttribute('aria-expanded', 'false')
    // Enter selects the section on the email.
    press('Enter')
    expect(tagOf(editor.getSelected())).toBe('mj-section')
    expect(editor.getSelected()).toBe(editor.getWrapper()!.findType('mj-section')[1])
    expect(second).toHaveAttribute('aria-selected', 'true')
    // Home and End.
    press('Home')
    expect(focused()).toBe(rows()[0])
    press('End')
    expect(focused()).toBe(rows()[rows().length - 1])
    // The head opens in the code view.
    rows()[1]!.focus()
    press('Enter')
    expect(setView).toHaveBeenCalledWith('mjml')
  })

  it('opens the rows down to the selection, and starts the tab order there', () => {
    const text = editor.getWrapper()!.findType('mj-text')[2]!
    editor.select(text)
    render(<Harness base={testComposer()} />)
    const row = rows().find((r) => r.getAttribute('aria-selected') === 'true')!
    expect(row.textContent).toContain('mj-text')
    expect(row.tabIndex).toBe(0)
    expect(row).toHaveAttribute('aria-level', '5')
    // Space selects without leaving the tree.
    row.focus()
    press('ArrowUp')
    press(' ')
    expect(tagOf(editor.getSelected())).toBe('mj-column')
    expect(document.activeElement).toBe(rows().find((r) => r.getAttribute('aria-selected') === 'true'))
  })
})
