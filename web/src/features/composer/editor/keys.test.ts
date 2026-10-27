import type { Component, Editor } from 'grapesjs'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { canvasKeyAction, caretAtEnd, installCanvasKeys, isTextEditable, runCanvasKey } from './keys'
import { mjChildren, tagOf } from './labels'
import { headlessMjmlEditor } from './testEditor'

// CO-8: Enter starts editing the selected text, Esc selects the parent.

let editor: Editor

beforeAll(async () => {
  ;({ editor } = await headlessMjmlEditor(
    '<mjml><mj-body><mj-section><mj-column><mj-text>Hello there</mj-text><mj-button href="https://acme.io">Read more</mj-button><mj-image src="https://acme.io/a.png" /></mj-column></mj-section></mj-body></mjml>',
  ))
})

afterAll(() => editor.destroy())

const key = (k: string, more: Partial<KeyboardEvent> = {}) => ({ key: k, metaKey: false, ctrlKey: false, altKey: false, shiftKey: false, isComposing: false, defaultPrevented: false, ...more })

// (A headless editor has no elements to query: findType walks the components.)
function find(tag: string): Component {
  return editor.getWrapper()!.findType(tag)[0]!
}

describe('canvasKeyAction', () => {
  it('Enter edits a text or a button, and goes into a section or column', () => {
    editor.select(find('mj-text'))
    expect(canvasKeyAction(editor, key('Enter'), true)).toBe('edit')
    editor.select(find('mj-button'))
    expect(canvasKeyAction(editor, key('Enter'), true)).toBe('edit')
    editor.select(find('mj-section'))
    expect(canvasKeyAction(editor, key('Enter'), true)).toBe('child')
    // An image has nothing to edit or go into.
    editor.select(find('mj-image'))
    expect(canvasKeyAction(editor, key('Enter'), true)).toBeNull()
  })

  it("doesn't edit for someone who can't, and leaves other keys alone", () => {
    editor.select(find('mj-text'))
    expect(canvasKeyAction(editor, key('Enter'), false)).toBeNull()
    expect(canvasKeyAction(editor, key('Enter', { metaKey: true }), true)).toBeNull()
    expect(canvasKeyAction(editor, key('Enter', { shiftKey: true }), true)).toBeNull()
    expect(canvasKeyAction(editor, key('Enter', { defaultPrevented: true }), true)).toBeNull()
    expect(canvasKeyAction(editor, key('a'), true)).toBeNull()
  })

  it('Esc selects the parent, up to the email body', () => {
    editor.select(find('mj-text'))
    expect(canvasKeyAction(editor, key('Escape'), true)).toBe('parent')
    editor.select(find('mj-body'))
    expect(canvasKeyAction(editor, key('Escape'), true)).toBeNull()
  })

  it('while a text is edited, Enter is a new line and Esc stops editing', () => {
    const text = find('mj-text')
    const editing = { getSelected: () => text, getEditing: () => text } as unknown as Editor
    expect(canvasKeyAction(editing, key('Enter'), true)).toBeNull()
    expect(canvasKeyAction(editing, key('Escape'), true)).toBe('stop')
  })
})

describe('runCanvasKey', () => {
  it('selects the parent and the first element inside', () => {
    const text = find('mj-text')
    editor.select(text)
    runCanvasKey(editor, 'parent')
    expect(tagOf(editor.getSelected())).toBe('mj-column')
    runCanvasKey(editor, 'parent')
    expect(tagOf(editor.getSelected())).toBe('mj-section')
    runCanvasKey(editor, 'child')
    expect(tagOf(editor.getSelected())).toBe('mj-column')
    runCanvasKey(editor, 'child')
    expect(editor.getSelected()).toBe(mjChildren(find('mj-column'))[0])
  })

  it('starts editing the way a double-click does', () => {
    const text = find('mj-text')
    editor.select(text)
    const active = vi.fn()
    text.on('active', active)
    runCanvasKey(editor, 'edit')
    expect(active).toHaveBeenCalled()
    text.off('active', active)
  })

  it('stops editing the way GrapesJS does before a toolbar command', () => {
    const stop = vi.fn()
    editor.on('toolbar:run:before', stop)
    runCanvasKey(editor, 'stop')
    expect(stop).toHaveBeenCalled()
    editor.off('toolbar:run:before', stop)
  })
})

describe('isTextEditable', () => {
  it('is false for a text locked for editing', () => {
    const text = find('mj-text')
    expect(isTextEditable(text)).toBe(true)
    text.set({ editable: false })
    expect(isTextEditable(text)).toBe(false)
    text.set({ editable: true })
  })
})

describe('installCanvasKeys', () => {
  it('listens in the canvas frame and stops listening when asked', () => {
    const text = find('mj-text')
    const frameDoc = document.implementation.createHTMLDocument('canvas')
    const handlers: Record<string, () => void> = {}
    const fake = {
      getSelected: () => text,
      getEditing: () => null,
      Canvas: { getDocument: () => frameDoc },
      on: (ev: string, fn: () => void) => (handlers[ev] = fn),
      off: (ev: string) => delete handlers[ev],
      once: vi.fn(),
    } as unknown as Editor
    const active = vi.fn()
    text.on('active', active)
    const stop = installCanvasKeys(fake, () => true)
    const enter = new KeyboardEvent('keydown', { key: 'Enter', cancelable: true })
    frameDoc.dispatchEvent(enter)
    expect(active).toHaveBeenCalledTimes(1)
    expect(enter.defaultPrevented).toBe(true)
    // A new frame (a reload of the canvas) gets the listener too.
    expect(handlers['canvas:frame:load']).toBeTypeOf('function')
    stop()
    frameDoc.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', cancelable: true }))
    expect(active).toHaveBeenCalledTimes(1)
    text.off('active', active)
  })
})

describe('caretAtEnd', () => {
  it('puts the caret after the last word', () => {
    const el = document.createElement('div')
    el.innerHTML = '<p>One</p><p>Two <b>three</b></p>'
    document.body.appendChild(el)
    caretAtEnd(el)
    const r = document.getSelection()!.getRangeAt(0)
    expect(r.collapsed).toBe(true)
    expect(r.endContainer).toBe(el)
    expect(r.endOffset).toBe(el.childNodes.length)
    el.remove()
  })
})
