import type { Component, Editor } from 'grapesjs'
import { mjChildren, tagOf } from './labels'

/**
 * Keys on the canvas (CO-8), for people who don't double-click: Enter
 * starts editing the selected text (the caret at its end), or selects the
 * first element inside a section or column; Esc stops editing a text, or
 * selects the parent. Both work in the canvas frame, where the focus is
 * once something on the email is clicked or picked in Layers.
 */
export type CanvasKeyAction = 'edit' | 'child' | 'stop' | 'parent'

type KeyLike = Pick<KeyboardEvent, 'key' | 'metaKey' | 'ctrlKey' | 'altKey' | 'shiftKey' | 'isComposing' | 'defaultPrevented'>

/** A component whose words are edited on the canvas (a text, a button). */
export function isTextEditable(c: Component | null | undefined): boolean {
  return !!c && !!c.get('editable') && c.isInstanceOf('text')
}

/** What a key does on the canvas, or null when it isn't the canvas's. */
export function canvasKeyAction(editor: Pick<Editor, 'getEditing' | 'getSelected'>, e: KeyLike, canEdit: boolean): CanvasKeyAction | null {
  if (e.defaultPrevented || e.isComposing || e.metaKey || e.ctrlKey || e.altKey) return null
  const editing = editor.getEditing()
  if (e.key === 'Escape') {
    if (editing) return 'stop'
    const parent = editor.getSelected()?.parent()
    return parent && tagOf(parent) && tagOf(parent) !== 'mjml' ? 'parent' : null
  }
  if (e.key === 'Enter' && !e.shiftKey) {
    // While a text is edited, Enter is a new line.
    if (editing) return null
    const sel = editor.getSelected()
    if (!sel) return null
    if (canEdit && isTextEditable(sel)) return 'edit'
    return mjChildren(sel).length ? 'child' : null
  }
  return null
}

/** Puts the caret at the end of an element being edited. */
export function caretAtEnd(el: HTMLElement) {
  const doc = el.ownerDocument
  const sel = doc.getSelection()
  if (!sel) return
  const r = doc.createRange()
  r.selectNodeContents(el)
  r.collapse(false)
  sel.removeAllRanges()
  sel.addRange(r)
}

/** Does what canvasKeyAction decided. */
export function runCanvasKey(editor: Editor, action: CanvasKeyAction) {
  const sel = editor.getSelected()
  switch (action) {
    case 'edit': {
      if (!sel) return
      // The text view starts editing on its model's 'active' (as on a double-click).
      editor.once('rte:enable', (_view: unknown, rte: { el?: HTMLElement } | undefined) => {
        if (rte?.el) caretAtEnd(rte.el)
      })
      sel.trigger('active')
      return
    }
    case 'stop':
      // GrapesJS ends text editing before a toolbar command runs; the text stays selected.
      editor.trigger('toolbar:run:before')
      editor.Canvas.getWindow()?.focus()
      return
    case 'parent':
    case 'child': {
      const target = action === 'parent' ? sel?.parent() : sel ? mjChildren(sel)[0] : undefined
      if (!target) return
      editor.select(target)
      target.getEl()?.scrollIntoView({ block: 'nearest' })
    }
  }
}

/**
 * Listens for Enter and Esc in the canvas frame (again after the frame
 * reloads). Returns the function that stops listening.
 */
export function installCanvasKeys(editor: Editor, canEdit: () => boolean): () => void {
  let doc: Document | undefined
  const onKey = (e: KeyboardEvent) => {
    const action = canvasKeyAction(editor, e, canEdit())
    if (!action) return
    e.preventDefault()
    runCanvasKey(editor, action)
  }
  const attach = () => {
    doc?.removeEventListener('keydown', onKey)
    doc = editor.Canvas.getDocument() ?? undefined
    doc?.addEventListener('keydown', onKey)
  }
  attach()
  editor.on('canvas:frame:load', attach)
  return () => {
    editor.off('canvas:frame:load', attach)
    doc?.removeEventListener('keydown', onKey)
  }
}

/** Moves the keyboard to the canvas (after picking an element in Layers), so Enter and Esc work there. */
export function focusCanvas(editor: Editor) {
  const frame = editor.Canvas.getFrameEl()
  if (!frame) return
  frame.focus({ preventScroll: true })
  frame.contentWindow?.focus()
}
