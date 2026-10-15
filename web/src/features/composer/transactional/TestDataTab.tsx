import { lazy, Suspense, useId, useMemo, useState } from 'react'
import { Icon } from '@/components/icons/Icon'
import { formatList, useT } from '@/i18n'
import { APPLICABLE, useApplyFix } from './checkFixes'
import { m } from './messages'
import { useTx } from './state'
import { formatJson, parseJsonObject, sampleData, uniqueName } from './variables'

const CodeEditor = lazy(() => import('./CodeEditor'))

/** The JSON of one set, kept as typed: invalid JSON stays in the editor and the last valid data stays in use. */
function SetEditor({ name }: { name: string }) {
  const t = useT()
  const tx = useTx()
  const ids = useId()
  const set = tx.testData.find((s) => s.name === name)
  const [text, setText] = useState(() => (set ? formatJson(set.data) : '{}'))
  const [error, setError] = useState<{ line: number; notObject?: boolean } | null>(null)
  const [label, setLabel] = useState(name)
  // Another change to the set (a fix, a renamed variable, a teammate's draft) replaces the text;
  // what is being typed stays while it parses to the same data.
  const [shown, setShown] = useState(set?.data)
  if (set && set.data !== shown) {
    setShown(set.data)
    const parsed = parseJsonObject(text)
    if (!parsed.ok || JSON.stringify(parsed.data) !== JSON.stringify(set.data)) {
      setText(formatJson(set.data))
      setError(null)
    }
  }
  if (!set) return null
  const onText = (v: string) => {
    setText(v)
    const parsed = parseJsonObject(v)
    if (!parsed.ok) {
      setError({ line: parsed.line, notObject: parsed.notObject })
      return
    }
    setError(null)
    tx.setTestData(tx.testData.map((s) => (s.name === name ? { ...s, data: parsed.data } : s)))
  }
  const taken = tx.testData.filter((s) => s.name !== name).map((s) => s.name)
  const labelError = !label.trim() ? t(m.setNameEmpty) : taken.includes(label.trim()) ? t(m.setNameTaken, { name: label.trim() }) : undefined
  return (
    <div className="flex flex-col gap-2.5">
      {tx.canEdit ? (
        <div className="flex items-end gap-2">
          <label className="lbl" style={{ flex: '1 1 auto' }}>
            {t(m.setName)}
            <input
              className="in"
              value={label}
              maxLength={80}
              aria-invalid={labelError ? true : undefined}
              onChange={(e) => setLabel(e.target.value)}
              onBlur={() => {
                const next = label.trim()
                if (labelError || next === name) {
                  setLabel(name)
                  return
                }
                tx.setTestData(tx.testData.map((s) => (s.name === name ? { ...s, name: next } : s)))
                if (tx.previewWith === name) tx.setPreviewWith(next)
              }}
            />
          </label>
          <button
            type="button"
            className="btn sm"
            onClick={() => {
              const copy = uniqueName(t(m.copyOf, { name }), tx.testData.map((s) => s.name))
              tx.setTestData([...tx.testData, { name: copy, data: set.data }])
              tx.setPreviewWith(copy)
            }}
          >
            {t(m.duplicateSet)}
          </button>
          <button
            type="button"
            className="btn sm danger"
            onClick={() => {
              const rest = tx.testData.filter((s) => s.name !== name)
              tx.setTestData(rest, t(m.noteSetDeleted, { name }))
              tx.setPreviewWith(rest[0]?.name ?? '')
            }}
          >
            {t(m.deleteSet)}
          </button>
        </div>
      ) : null}
      {labelError && label !== name ? <span className="k txe-bad">{labelError}</span> : null}
      <div className="txe-cbx">
        <div className="txe-cbh">
          <span>{t(m.jsonTitle, { name })}</span>
        </div>
        <Suspense fallback={<pre className="txe-pre-dark">{text}</pre>}>
          <CodeEditor lang="json" label={t(m.jsonLabel, { name })} value={text} onChange={tx.canEdit ? onText : undefined} readOnly={!tx.canEdit} describedBy={error ? `${ids}-err` : undefined} />
        </Suspense>
      </div>
      {error ? (
        <span id={`${ids}-err`} className="k txe-bad" role="alert">
          {error.notObject ? t(m.notObject) : t(m.invalidJson, { line: error.line })}
        </span>
      ) : null}
    </div>
  )
}

/** Whether the chosen set renders, from the last check run. */
function RenderStatus({ name }: { name: string }) {
  const t = useT()
  const tx = useTx()
  const apply = useApplyFix()
  const r = tx.checks?.testData.find((x) => x.name === name)
  const check = tx.checks?.checks.find((c) => c.testData === name && (c.id === 'testdata.missing' || c.id === 'testdata.render'))
  if (!r) return null
  if (r.ok)
    return (
      <div className="note g" role="status">
        <Icon name="check" size={14} />
        <div>{r.defaults.length ? t(m.rendersDefaults, { name, names: formatList(r.defaults) }) : t(m.renders, { name })}</div>
      </div>
    )
  return (
    <div className="note b" role="alert">
      <Icon name="alertCircle" size={16} />
      <div className="flex min-w-0 flex-col gap-1.5" style={{ flex: '1 1 220px' }}>
        <span className="font-bold">{t(m.cantRender)}</span>
        <span>{check?.detail || r.problem}</span>
        {check && tx.canEdit
          ? check.fixes
              .filter((f) => APPLICABLE.has(f.kind))
              .map((f) => (
                <button key={f.kind} type="button" className="btn sm self-start" onClick={() => apply(check, f)}>
                  {f.label}
                </button>
              ))
          : null}
      </div>
    </div>
  )
}

/**
 * Test data (TransactionalTemplateStates): named JSON samples used by the
 * preview, the checks and the publish dialog's test renders. The chosen set
 * is the one "Preview with" shows.
 */
export function TestDataTab() {
  const t = useT()
  const tx = useTx()
  const samples = useMemo(() => formatJson(sampleData(tx.variables)), [tx.variables])
  const add = () => {
    const name = uniqueName(t(m.newSet), tx.testData.map((s) => s.name))
    tx.setTestData([...tx.testData, { name, data: sampleData(tx.variables) }], t(m.noteSetAdded, { name }))
    tx.setPreviewWith(name)
  }
  return (
    <div className="pbody">
      <span className="k">{t(m.testDataHelp)}</span>
      <div className="flex flex-wrap gap-1.5" role="group" aria-label={t(m.setsLabel)}>
        {tx.testData.map((s) => (
          <button key={s.name} type="button" className={`txe-fchip${tx.previewWith === s.name ? ' on' : ''}`} aria-pressed={tx.previewWith === s.name} onClick={() => tx.setPreviewWith(s.name)}>
            {s.name}
          </button>
        ))}
        <button type="button" className={`txe-fchip${tx.previewWith === '' ? ' on' : ''}`} aria-pressed={tx.previewWith === ''} onClick={() => tx.setPreviewWith('')}>
          {t(m.samples)}
        </button>
        {tx.canEdit ? (
          <button type="button" className="txe-fchip" onClick={add}>
            {t(m.addSet)}
          </button>
        ) : null}
      </div>
      {tx.previewWith ? (
        <>
          <SetEditor key={tx.previewWith} name={tx.previewWith} />
          <RenderStatus name={tx.previewWith} />
        </>
      ) : (
        <>
          <span className="k">{t(m.samplesHelp)}</span>
          <div className="txe-cbx">
            <div className="txe-cbh">
              <span>{t(m.samplesTitle)}</span>
            </div>
            <pre className="txe-pre-dark">{samples}</pre>
          </div>
          {tx.canEdit ? (
            <button type="button" className="btn sm self-start" onClick={add}>
              {t(m.saveSamplesAsSet)}
            </button>
          ) : null}
        </>
      )}
    </div>
  )
}
