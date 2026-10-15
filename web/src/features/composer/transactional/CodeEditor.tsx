import { html } from '@codemirror/lang-html'
import { EditorState, type Extension } from '@codemirror/state'
import { Decoration, EditorView, lineNumbers, MatchDecorator, ViewPlugin, type DecorationSet, type ViewUpdate } from '@codemirror/view'
import { minimalSetup } from 'codemirror'
import { useEffect, useLayoutEffect, useRef } from 'react'

const marks = {
  key: Decoration.mark({ class: 'tx-k' }),
  str: Decoration.mark({ class: 'tx-s' }),
  lit: Decoration.mark({ class: 'tx-l' }),
  liq: Decoration.mark({ class: 'tx-q' }),
}

/** JSON colouring as on the Test data board: keys, strings, numbers and literals. */
const jsonColors = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    decorator = new MatchDecorator({
      regexp: /("(?:[^"\\\n]|\\.)*")(?=\s*:)|("(?:[^"\\\n]|\\.)*")|(-?\b\d+(?:\.\d+)?\b|\btrue\b|\bfalse\b|\bnull\b)/g,
      decoration: (m) => (m[1] ? marks.key : m[2] ? marks.str : marks.lit),
    })
    constructor(view: EditorView) {
      this.decorations = this.decorator.createDeco(view)
    }
    update(u: ViewUpdate) {
      this.decorations = this.decorator.updateDeco(u, this.decorations)
    }
  },
  { decorations: (v) => v.decorations },
)

/** Liquid outputs and tags in plain text. */
const liquidColors = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    decorator = new MatchDecorator({ regexp: /\{\{[^}\n]*\}\}|\{%[^%\n]*%\}/g, decoration: () => marks.liq })
    constructor(view: EditorView) {
      this.decorations = this.decorator.createDeco(view)
    }
    update(u: ViewUpdate) {
      this.decorations = this.decorator.updateDeco(u, this.decorations)
    }
  },
  { decorations: (v) => v.decorations },
)

const dark = EditorView.theme(
  {
    '&': { color: '#E5E7EB', backgroundColor: 'transparent', height: '100%' },
    '.cm-content': { caretColor: '#F9FAFB', padding: '10px 0' },
    '.cm-gutters': { backgroundColor: 'transparent', color: '#9CA3AF', border: 0 },
    '.cm-lineNumbers .cm-gutterElement': { minWidth: '24px', padding: '0 12px 0 8px', textAlign: 'right' },
    '.cm-activeLine': { backgroundColor: 'rgba(255,255,255,.04)' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: 'rgba(147,197,253,.28) !important' },
    '.cm-cursor': { borderLeftColor: '#F9FAFB' },
    '.tx-k': { color: '#93C5FD' },
    '.tx-s': { color: '#86EFAC' },
    '.tx-l': { color: '#FCD34D' },
    '.tx-q': { color: '#C4B5FD' },
  },
  { dark: true },
)

const light = EditorView.theme({
  '&': { color: '#1F2937', backgroundColor: 'transparent', height: '100%' },
  '.cm-content': { padding: '10px 0' },
  '.cm-gutters': { backgroundColor: 'transparent', color: '#6B7280', border: 0 },
  '.cm-lineNumbers .cm-gutterElement': { minWidth: '24px', padding: '0 12px 0 8px', textAlign: 'right' },
  '.tx-q': { color: '#5B21B6', backgroundColor: '#EDE9FE', borderRadius: '3px' },
})

/**
 * A CodeMirror editor for the transactional editor's JSON test data, HTML
 * and plain text. Loaded on demand with its panel.
 */
export default function CodeEditor({
  value,
  onChange,
  lang,
  label,
  readOnly,
  theme = 'dark',
  describedBy,
}: {
  value: string
  onChange?: (v: string) => void
  lang: 'json' | 'html' | 'text'
  label: string
  readOnly?: boolean
  theme?: 'dark' | 'light'
  describedBy?: string
}) {
  const host = useRef<HTMLDivElement>(null)
  const view = useRef<EditorView | null>(null)
  const cb = useRef(onChange)
  useLayoutEffect(() => {
    cb.current = onChange
  })
  useEffect(() => {
    if (!host.current) return
    const attrs: Record<string, string> = { 'aria-label': label, tabindex: '0' }
    if (describedBy) attrs['aria-describedby'] = describedBy
    const ext: Extension[] = [
      minimalSetup,
      lineNumbers(),
      theme === 'dark' ? dark : light,
      EditorView.lineWrapping,
      EditorView.contentAttributes.of(attrs),
      EditorView.updateListener.of((u) => {
        if (u.docChanged) cb.current?.(u.state.doc.toString())
      }),
    ]
    if (lang === 'json') ext.push(jsonColors)
    else if (lang === 'html') ext.push(html(), liquidColors)
    else ext.push(liquidColors)
    if (readOnly) ext.push(EditorState.readOnly.of(true), EditorView.editable.of(false))
    const v = new EditorView({ state: EditorState.create({ doc: value, extensions: ext }), parent: host.current })
    view.current = v
    return () => {
      v.destroy()
      view.current = null
    }
    // The editor is created once per mode; value changes are dispatched below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lang, readOnly, theme, label])
  useEffect(() => {
    const v = view.current
    if (v && v.state.doc.toString() !== value) v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } })
  }, [value])
  return <div ref={host} className="txe-cm" />
}
