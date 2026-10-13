import { PostgreSQL, sql } from '@codemirror/lang-sql'
import { EditorState } from '@codemirror/state'
import { EditorView, lineNumbers, placeholder } from '@codemirror/view'
import { minimalSetup } from 'codemirror'
import { Fragment, useEffect, useRef, type ReactNode } from 'react'

const keywords = new Set(
  'SELECT FROM WHERE AND OR NOT EXISTS IN IS NULL TRUE FALSE INTERVAL AS ON JOIN LEFT INNER GROUP BY ORDER LIMIT DISTINCT CASE WHEN THEN ELSE END LIKE ILIKE ANY BETWEEN UNION ALL WITH HAVING'.split(
    ' ',
  ),
)

/**
 * Highlights SQL the way the Segments board does: keywords blue, strings
 * green, numbers yellow, everything else light grey.
 */
export function highlightSQL(src: string): ReactNode {
  const out: ReactNode[] = []
  const re = /('(?:[^']|'')*')|(\b\d+(?:\.\d+)?\b)|(\b[A-Za-z_]+\b)|(\s+|[^\sA-Za-z_'\d]+)/g
  let m: RegExpExecArray | null
  let i = 0
  while ((m = re.exec(src))) {
    const [tok, str, num, word] = m
    if (str) out.push(<span key={i++} className="st">{tok}</span>)
    else if (num) out.push(<span key={i++} className="nm">{tok}</span>)
    else if (word && keywords.has(word.toUpperCase())) out.push(<span key={i++} className="kw">{tok}</span>)
    else out.push(<Fragment key={i++}>{tok}</Fragment>)
  }
  return out
}

/** Read-only SQL with the board's colours. */
export function SqlBlock({ sql: text, label }: { sql: string; label: string }) {
  return (
    <pre className="sqlpre" aria-label={label}>
      {highlightSQL(text)}
    </pre>
  )
}

const theme = EditorView.theme({
  '&': { backgroundColor: '#fff', color: 'var(--ink)', border: '1px solid var(--line-input)', borderRadius: '8px', fontSize: '12px' },
  '.cm-content': { fontFamily: "'IBM Plex Mono', ui-monospace, monospace", padding: '12px 0', lineHeight: '1.55' },
  '.cm-gutters': { backgroundColor: 'var(--ground-3)', color: 'var(--ink-3)', borderRight: '1px solid var(--line-2)', borderRadius: '8px 0 0 8px' },
  '.cm-activeLine, .cm-activeLineGutter': { backgroundColor: 'var(--accent-tint)' },
  '&.cm-focused': { outline: '2px solid var(--focus)', outlineOffset: '2px' },
})

/**
 * The SQL editor of SQL segments (CodeMirror 6, PostgreSQL dialect). It
 * owns its document; `value` seeds it and `onChange` reports edits.
 */
export function SqlEditor({ value, onChange, label, invalid }: { value: string; onChange: (v: string) => void; label: string; invalid?: boolean }) {
  const host = useRef<HTMLDivElement>(null)
  const view = useRef<EditorView | null>(null)
  const changed = useRef(onChange)
  useEffect(() => {
    changed.current = onChange
  }, [onChange])
  useEffect(() => {
    if (!host.current) return
    const v = new EditorView({
      parent: host.current,
      state: EditorState.create({
        doc: value,
        extensions: [
          lineNumbers(),
          minimalSetup,
          sql({ dialect: PostgreSQL, upperCaseKeywords: true }),
          theme,
          placeholder('SELECT c.id FROM contacts c WHERE …'),
          EditorView.lineWrapping,
          EditorView.contentAttributes.of({ 'aria-label': label, 'aria-multiline': 'true', ...(invalid ? { 'aria-invalid': 'true' } : {}) }),
          EditorView.updateListener.of((u) => {
            if (u.docChanged) changed.current(u.state.doc.toString())
          }),
        ],
      }),
    })
    view.current = v
    return () => {
      v.destroy()
      view.current = null
    }
    // The document is seeded once; later edits come from the editor itself.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  return <div ref={host} className={invalid ? 'sqled invalid' : 'sqled'} />
}
