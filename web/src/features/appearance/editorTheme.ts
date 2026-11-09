import { Compartment, type Extension } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { useEffect, useState, type RefObject } from 'react'
import { useTheme } from './useTheme'

/**
 * For a CodeMirror editor drawn on the app's surfaces (a code pane is dark in
 * both themes and says so in its own theme): CodeMirror's light or dark base
 * (caret, selection) follows the app's theme. Create the editor with the
 * returned extension; a theme change reconfigures `view` in place. Kept out
 * of useTheme.ts so the shell doesn't load CodeMirror.
 */
export function useEditorTheme(view: RefObject<EditorView | null>): Extension {
  const dark = useTheme().theme === 'dark'
  const [slot] = useState(() => new Compartment())
  useEffect(() => {
    view.current?.dispatch({ effects: slot.reconfigure(EditorView.darkTheme.of(dark)) })
  }, [view, slot, dark])
  return slot.of(EditorView.darkTheme.of(dark))
}
