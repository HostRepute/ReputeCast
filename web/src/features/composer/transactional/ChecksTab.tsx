import { useState } from 'react'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { useComposer } from '../context'
import { reveal } from '../fixes'
import { APPLICABLE, useApplyFix } from './checkFixes'
import { m } from './messages'
import { useTx } from './state'
import type { Check } from './types'

function SeverityBadge({ severity }: { severity: Check['severity'] }) {
  const t = useT()
  switch (severity) {
    case 'block':
      return <Badge tone="bad">{t(m.sevBlock)}</Badge>
    case 'warn':
      return <Badge tone="warn">{t(m.sevWarn)}</Badge>
    case 'info':
      return <Badge tone="info">{t(m.sevInfo)}</Badge>
  }
  return <Badge tone="ok">{t(m.sevPass)}</Badge>
}

function CheckRow({ check }: { check: Check }) {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const apply = useApplyFix()
  const [alt, setAlt] = useState('')
  const fixes = tx.canEdit ? check.fixes.filter((f) => APPLICABLE.has(f.kind)) : []
  const canReveal = check.path.length > 0 && tx.view === 'visual'
  const select = () => {
    if (canReveal) reveal(c, check.path)
    else if (check.variable && tx.variables.some((v) => v.name === check.variable)) {
      tx.setFocusVariable(check.variable)
      tx.setTab('variables')
    } else if (check.testData) {
      tx.setPreviewWith(check.testData)
      tx.setTab('testdata')
    }
  }
  const linked = canReveal || !!check.testData || (!!check.variable && tx.variables.some((v) => v.name === check.variable))
  return (
    <div className="txe-chk">
      <div className="flex flex-wrap items-center gap-2">
        <SeverityBadge severity={check.severity} />
        {linked ? (
          <button type="button" className="txe-linklike font-semibold" onClick={select}>
            {check.title}
          </button>
        ) : (
          <span className="font-semibold">{check.title}</span>
        )}
      </div>
      {check.detail ? <span className="k">{check.detail}</span> : null}
      {check.line && check.where === 'body' && !tx.htmlMode ? (
        <button type="button" className="txe-linklike k self-start" onClick={() => tx.setView('mjml')}>
          {t(m.openMjmlAtLine, { line: check.line })}
        </button>
      ) : null}
      {fixes.length ? (
        <div className="flex flex-wrap gap-1.5">
          {fixes.map((f) =>
            f.kind === 'set_alt' ? (
              <form
                key={f.kind}
                className="flex flex-1 gap-1.5"
                onSubmit={(e) => {
                  e.preventDefault()
                  apply(check, f, alt)
                  setAlt('')
                }}
              >
                <input aria-label={t(m.describeImage)} className="in" placeholder={t(m.describeImage)} style={{ flex: 1 }} value={alt} onChange={(e) => setAlt(e.target.value)} />
                <button type="submit" className="btn sm" disabled={!alt.trim()}>
                  {f.label}
                </button>
              </form>
            ) : (
              <button key={f.kind + f.label} type="button" className="btn sm" onClick={() => apply(check, f)}>
                {f.label}
              </button>
            ),
          )}
        </div>
      ) : null}
    </div>
  )
}

/** Checks (TransactionalTemplate · Checks tab): the composer's content checks, adapted, and the transactional ones. */
export function ChecksTab() {
  const t = useT()
  const tx = useTx()
  const r = tx.checks
  const [showPassed, setShowPassed] = useState(false)
  if (!r)
    return (
      <div className="pbody">
        <span className="k">{t(m.checking)}</span>
      </div>
    )
  const open = r.checks.filter((c) => c.severity !== 'pass')
  const passed = r.checks.filter((c) => c.severity === 'pass')
  return (
    <div className="pbody">
      <div className="flex items-center justify-between gap-2">
        <span className="font-semibold">
          {r.verdict === 'blocked' ? t(m.verdictBlocked, { count: r.blocks }) : r.verdict === 'warnings' ? t(m.verdictWarnings, { count: r.warnings }) : t(m.verdictReady)}
        </span>
        {tx.checksPending ? <span className="spin" aria-label={t(m.checking)} /> : null}
      </div>
      <span className="k">{t(m.checksIntro)}</span>
      {open.length ? (
        <div className="flex flex-col">
          {open.map((c, i) => (
            <CheckRow key={`${c.id}-${c.variable ?? ''}-${c.testData ?? ''}-${i}`} check={c} />
          ))}
        </div>
      ) : null}
      {passed.length ? (
        <div className="flex flex-col gap-2">
          <button type="button" className="txe-linklike font-semibold self-start" aria-expanded={showPassed} onClick={() => setShowPassed((v) => !v)}>
            {t(m.passedTitle, { count: passed.length })}
          </button>
          {showPassed ? (
            <div className="flex flex-wrap gap-1.5">
              {passed.map((c) => (
                <Badge key={c.id} tone="ok">
                  {c.title}
                </Badge>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
