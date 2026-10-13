import { EditorState, type Extension } from '@codemirror/state'
import { Decoration, EditorView, lineNumbers, MatchDecorator, ViewPlugin, type DecorationSet, type ViewUpdate } from '@codemirror/view'
import { minimalSetup } from 'codemirror'
import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { http } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { useComposer } from './context'
import { checkMjmlTags, formatMjml, headMeta, splitMjml } from './editor/mjmlDoc'
import { kb, lineCount } from './format'
import { ViewSeg } from './CanvasArea'

/* ------------------------------------------------------------ CodeMirror */

const marks = {
  cm: Decoration.mark({ class: 'mj-cm' }),
  tg: Decoration.mark({ class: 'mj-tg' }),
  at: Decoration.mark({ class: 'mj-at' }),
  st: Decoration.mark({ class: 'mj-st' }),
}

/** MJML colouring as on the ComposerCode board: tags, attribute names, values, comments. */
const mjmlColors = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    decorator = new MatchDecorator({
      regexp: /(<!--.*?-->)|(<\/?[a-zA-Z][\w-]*|\/?>)|([\w:-]+)(?==)|("[^"\n]*"|'[^'\n]*')/g,
      decoration: (m) => (m[1] ? marks.cm : m[2] ? marks.tg : m[3] ? marks.at : marks.st),
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

const darkTheme = EditorView.theme(
  {
    '&': { color: '#D1D5DB', backgroundColor: 'transparent', height: '100%' },
    '.cm-content': { caretColor: '#F9FAFB', padding: 0 },
    '.cm-gutters': { backgroundColor: 'transparent', color: '#6B7280', border: 0 },
    '.cm-lineNumbers .cm-gutterElement': { minWidth: '28px', padding: '0 16px 0 0', textAlign: 'right' },
    '.cm-activeLine': { backgroundColor: 'rgba(255,255,255,.04)' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: 'rgba(147,197,253,.28) !important' },
    '.cm-cursor': { borderLeftColor: '#F9FAFB' },
    '.mj-tg': { color: '#93C5FD' },
    '.mj-at': { color: '#FDE68A' },
    '.mj-st': { color: '#86EFAC' },
    '.mj-cm': { color: '#9CA3AF' },
  },
  { dark: true },
)

const lightTheme = EditorView.theme({
  '&': { color: '#374151', backgroundColor: 'transparent', height: '100%' },
  '.cm-content': { padding: 0 },
  '.cm-gutters': { backgroundColor: 'transparent', color: '#6B7280', border: 0 },
  '.cm-lineNumbers .cm-gutterElement': { minWidth: '28px', padding: '0 16px 0 0', textAlign: 'right' },
})

/** A CodeMirror editor bound to `value`. */
function Code({ value, onChange, readOnly, dark, label, onCursor, jumpTo }: { value: string; onChange?: (v: string) => void; readOnly?: boolean; dark?: boolean; label: string; onCursor?: (line: number, col: number) => void; jumpTo?: { line: number; n: number } }) {
  const host = useRef<HTMLDivElement>(null)
  const view = useRef<EditorView | null>(null)
  const cb = useRef({ onChange, onCursor })
  useLayoutEffect(() => {
    cb.current = { onChange, onCursor }
  })
  useEffect(() => {
    if (!host.current) return
    const ext: Extension[] = [
      minimalSetup,
      lineNumbers(),
      dark ? darkTheme : lightTheme,
      EditorView.contentAttributes.of({ 'aria-label': label }),
      EditorView.updateListener.of((u) => {
        if (u.docChanged) cb.current.onChange?.(u.state.doc.toString())
        if (u.selectionSet || u.docChanged) {
          const head = u.state.selection.main.head
          const line = u.state.doc.lineAt(head)
          cb.current.onCursor?.(line.number, head - line.from + 1)
        }
      }),
    ]
    if (dark) ext.push(mjmlColors)
    if (readOnly) ext.push(EditorState.readOnly.of(true), EditorView.editable.of(false))
    const v = new EditorView({ state: EditorState.create({ doc: value, extensions: ext }), parent: host.current })
    view.current = v
    return () => {
      v.destroy()
      view.current = null
    }
    // The editor is created once; value changes are dispatched below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  useEffect(() => {
    const v = view.current
    if (v && v.state.doc.toString() !== value) v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } })
  }, [value])
  useEffect(() => {
    const v = view.current
    if (!v || !jumpTo) return
    const line = v.state.doc.line(Math.min(Math.max(1, jumpTo.line), v.state.doc.lines))
    v.dispatch({ selection: { anchor: line.from }, effects: EditorView.scrollIntoView(line.from, { y: 'center' }) })
    v.focus()
  }, [jumpTo])
  return <div ref={host} className="cmp-cm" />
}

/* ------------------------------------------------------------ view */

interface CompileState {
  html: string
  ms: number
  errors: { line: number; message: string }[]
  warnings: { line: number; message: string }[]
}

function Pane({ title, aside, dark, children }: { title: string; aside: string; dark?: boolean; children: ReactNode }) {
  return (
    <div className={dark ? 'cmp-code' : 'cmp-html'} style={{ flex: dark ? '1 1 420px' : '1 1 380px' }}>
      <div className="flex items-center justify-between gap-3" style={{ marginBottom: 10 }}>
        <span style={{ fontSize: 11, fontWeight: 700, letterSpacing: '.06em', textTransform: 'uppercase', color: dark ? '#9CA3AF' : 'var(--ink-3)' }}>{title}</span>
        <span className={dark ? undefined : 'k'} style={dark ? { fontSize: 11, color: '#9CA3AF' } : undefined}>
          {aside}
        </span>
      </div>
      {children}
    </div>
  )
}

/**
 * The code view (ComposerCode): MJML as the source of truth on the left,
 * the compiled HTML read-only on the right, compile results and the
 * personalization tokens in the side panel.
 */
export function CodeView() {
  const c = useComposer()
  const initial = useMemo(() => c.currentMjml(), []) // eslint-disable-line react-hooks/exhaustive-deps
  const [draft, setDraft] = useState(initial)
  const [cursor, setCursor] = useState({ line: 1, col: 1 })
  const [jump, setJump] = useState<{ line: number; n: number }>()
  const [tab, setTab] = useState<'compile' | 'tokens'>('compile')
  const [result, setResult] = useState<CompileState | null>(null)
  const [lint, setLint] = useState<{ line: number; message: string }[]>([])
  const [failed, setFailed] = useState<{ line: number; message: string } | null>(null)
  const changed = draft !== initial

  // Compile the draft as the person types (the same engine as the canvas).
  useEffect(() => {
    const h = window.setTimeout(() => {
      const t0 = performance.now()
      const out = c.ge?.compile(draft)
      if (!out) return
      setResult({ html: out.html, ms: Math.max(1, Math.round(performance.now() - t0)), errors: out.errors.map((e) => ({ line: e.line, message: e.message })), warnings: [] })
    }, 250)
    return () => window.clearTimeout(h)
  }, [draft, c.ge])
  // Layout warnings (image wider than its column…) come from the server.
  useEffect(() => {
    const h = window.setTimeout(() => {
      void http
        .POST('/api/v1/workspaces/{workspace}/mjml/compile', { params: { path: { workspace: c.slug } }, body: { mjml: draft } })
        .then(({ data }) => data && setLint(data.warnings.map((w) => ({ line: w.line, message: w.message }))))
        .catch(() => undefined)
    }, 700)
    return () => window.clearTimeout(h)
  }, [draft, c.slug])

  // Applying is also what leaving the code view does.
  const apply = (): boolean => {
    if (!changed) return true
    const fatal = checkMjmlTags(draft) ?? (draft.includes('<mj-body') ? null : { line: 1, message: 'the document needs an <mj-body>' })
    if (fatal) {
      setFailed({ line: fatal.line, message: fatal.message })
      return false
    }
    const meta = headMeta(splitMjml(draft).head)
    if (meta.title !== undefined && meta.title !== c.subject) c.setSubject(meta.title)
    if (meta.preview !== undefined && meta.preview !== c.preheader) c.setPreheader(meta.preview)
    c.loadContent(draft, { source: 'code', label: 'edited the MJML' })
    setFailed(null)
    return true
  }
  const applyRef = useRef(apply)
  useLayoutEffect(() => {
    applyRef.current = apply
  })
  const { registerLeaveCheck, setCodeDirty } = c
  // Unapplied edits count as unsaved work for the leave guard.
  useEffect(() => {
    setCodeDirty(changed)
    return () => setCodeDirty(false)
  }, [changed, setCodeDirty])
  useEffect(() => registerLeaveCheck(() => applyRef.current()), [registerLeaveCheck])

  const html = result?.html ?? ''
  const warnings = [...(result?.errors ?? []).map((e) => ({ ...e, kind: 'warn' as const })), ...lint.map((w) => ({ ...w, kind: 'warn' as const }))]
  const errors = failed ? [failed] : []
  const tokens = c.context.tokens
  const fmtToken = (t: (typeof tokens)[number]) => (t.fallback !== undefined && t.fallback !== '' ? `{{ ${t.key} | default: "${t.fallback}" }}` : `{{ ${t.key} }}`)
  // Names first (the address is rarely personalized in copy), then the first custom field.
  const contactTokens = [...tokens.filter((t) => t.kind === 'contact' && t.key !== 'email').slice(0, 2), ...tokens.filter((t) => t.kind === 'custom').slice(0, 1)]
  const systemTokens = tokens.filter((t) => t.kind === 'system').slice(0, 3)

  return (
    <>
      <section id="main" className="cmp-main" style={{ flex: '999 1 620px' }} aria-label="Code view">
        <div className="cmp-tools">
          <ViewSeg />
          {c.view === 'mjml' ? (
            <>
              <button type="button" className="btn" onClick={() => c.openDialog('import')} disabled={!c.canEdit}>
                Import MJML
              </button>
              <button type="button" className="btn" onClick={() => setDraft(formatMjml(draft))} disabled={!c.canEdit}>
                Format
              </button>
            </>
          ) : null}
          <button
            type="button"
            className="btn"
            onClick={() =>
              void navigator.clipboard.writeText(html).then(
                () => toast({ kind: 'success', title: 'Compiled HTML copied' }),
                () => toast({ kind: 'error', title: "Couldn't copy", detail: 'Your browser blocked the clipboard.' }),
              )
            }
          >
            Copy compiled HTML
          </button>
          <span className="k">mjml 4 · grapesjs-mjml 1.0 · changes here update the canvas and the layer tree</span>
          {c.view === 'mjml' && c.canEdit ? (
            <button type="button" className="btn pri" style={{ marginLeft: 'auto' }} onClick={() => apply() && c.setView('visual')}>
              Apply to canvas
            </button>
          ) : null}
        </div>
        {failed ? (
          <div role="alert" className="card mx-4 mt-4 flex flex-col" style={{ padding: '16px 18px', gap: 10, borderColor: '#FCA5A5' }}>
            <div className="flex flex-wrap items-center gap-2.5">
              <Badge tone="bad" dot>
                MJML error
              </Badge>
              <span className="font-semibold">
                Line {failed.line}: {failed.message}
              </span>
            </div>
            <span className="k">The canvas keeps the last good version. Fix the code view or revert the last change.</span>
            <div className="flex gap-2">
              <button type="button" className="btn sm pri" onClick={() => setJump({ line: failed.line, n: Date.now() })}>
                Go to line {failed.line}
              </button>
              <button
                type="button"
                className="btn sm"
                onClick={() => {
                  setDraft(initial)
                  setFailed(null)
                }}
              >
                Undo my changes
              </button>
            </div>
          </div>
        ) : null}
        <div className="flex flex-1 flex-wrap" style={{ minHeight: 0 }}>
          {c.view === 'mjml' ? (
            <Pane dark title="MJML · source of truth" aside={`${lineCount(draft)} lines · line ${cursor.line}, col ${cursor.col}`}>
              <Code dark label="MJML source" value={draft} onChange={c.canEdit ? setDraft : undefined} readOnly={!c.canEdit} onCursor={(line, col) => setCursor({ line, col })} jumpTo={jump} />
            </Pane>
          ) : null}
          <Pane title="Compiled HTML · read only" aside={`${kb(new Blob([html]).size)} · tables and inline CSS for Outlook`}>
            <Code label="Compiled HTML" value={html} readOnly />
          </Pane>
        </div>
      </section>
      <aside className="cpanel right" aria-label="Compile results and tokens">
        <PanelTabs
          value={tab}
          onChange={setTab}
          label="Code tools"
          items={[
            { value: 'compile', label: 'Compile' },
            { value: 'tokens', label: 'Tokens' },
          ]}
          className="flex flex-1 flex-col"
        >
          <TabPanel value="compile" className="flex flex-1 flex-col" tabIndex={-1}>
            {result ? (
              <div className="crow" style={{ background: errors.length ? 'var(--bad-tint)' : '#F0FDF4' }}>
                <Badge tone={errors.length ? 'bad' : 'ok'}>{errors.length ? 'Error' : 'OK'}</Badge>
                <span>
                  Compiled in {result.ms} ms · {errors.length} {errors.length === 1 ? 'error' : 'errors'} · {warnings.length} {warnings.length === 1 ? 'warning' : 'warnings'}
                </span>
              </div>
            ) : (
              <div className="crow">
                <span className="spin" aria-hidden="true" />
                <span>Compiling…</span>
              </div>
            )}
            {errors.map((e, i) => (
              <div key={`e${i}`} className="crow">
                <Badge tone="bad">Error</Badge>
                <span>
                  Line {e.line}: {e.message}
                </span>
              </div>
            ))}
            {warnings.map((w, i) => (
              <div key={`w${i}`} className="crow">
                <Badge tone="warn">Warn</Badge>
                <button type="button" className="text-left" style={{ border: 0, background: 'none', padding: 0, font: 'inherit', color: 'inherit', cursor: 'pointer' }} onClick={() => setJump({ line: w.line, n: Date.now() })}>
                  Line {w.line}: {w.message}
                </button>
              </div>
            ))}
            <div className="crow">
              <Badge tone="info">Info</Badge>
              <span>
                Comments like <span className="mono">rc:footer</span> mark ReputeCast blocks so they stay editable as one unit on the canvas.
              </span>
            </div>
            <div className="crow">
              <Badge tone="info">Info</Badge>
              <span>Tracking links and the open pixel are added at send time, per sending provider, so they don't appear here.</span>
            </div>
            <div className="flex flex-col gap-2 border-t border-line-2" style={{ padding: 12, marginTop: 'auto' }}>
              <span className="k">Personalization tokens available</span>
              <div className="mono flex flex-col gap-1" style={{ fontSize: 12, color: 'var(--ink-nav)' }}>
                {contactTokens.map((t) => (
                  <span key={t.key}>{fmtToken(t)}</span>
                ))}
                {systemTokens.length ? <span>{systemTokens.map(fmtToken).join(' · ')}</span> : null}
              </div>
              <span className="k">Fallbacks after the pipe are used when the field is empty.</span>
            </div>
          </TabPanel>
          <TabPanel value="tokens" className="flex flex-col" tabIndex={-1}>
            {(['contact', 'custom', 'system'] as const).map((kind) => {
              const list = tokens.filter((t) => t.kind === kind)
              if (!list.length) return null
              return (
                <div key={kind} className="flex flex-col">
                  <div className="grp">{kind === 'contact' ? 'Contact fields' : kind === 'custom' ? 'Custom fields' : 'Links and system'}</div>
                  {list.map((t) => (
                    <button
                      key={t.key}
                      type="button"
                      className="crow text-left"
                      style={{ border: 0, borderBottom: '1px solid var(--line-2)', background: 'none', width: '100%', font: 'inherit', cursor: 'pointer' }}
                      onClick={() =>
                        void navigator.clipboard.writeText(fmtToken(t)).then(
                          () => toast({ kind: 'success', title: `Copied ${fmtToken(t)}` }),
                          () => undefined,
                        )
                      }
                      title="Copy"
                    >
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span>{t.label}</span>
                        <span className="mono k" style={{ overflowWrap: 'anywhere' }}>
                          {fmtToken(t)}
                        </span>
                      </span>
                    </button>
                  ))}
                </div>
              )
            })}
          </TabPanel>
        </PanelTabs>
      </aside>
    </>
  )
}
