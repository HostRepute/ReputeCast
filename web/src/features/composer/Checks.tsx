import { useId, useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useAccessibility } from '@/features/accessibility/composer/state'
import { ReadinessRows } from '@/features/review/ComposerReadiness'
import { useComposer, type Issue } from './context'
import { kb } from './format'
import { mh } from './chrome.messages'

/**
 * The Checks tab (CO-14, Composer and ComposerWarnings boards): a summary
 * per area, each issue with its fixes, dismissing a warning with a reason
 * (and Undo; blocks can't be dismissed and say why), and what passed.
 */

type Dismissed = Schemas['ComposerDismissedIssue']
type Reason = 'decorative' | 'size' | 'subject' | 'wrong' | 'other'

/** The reasons offered for a warning: its own first, then "the check is wrong" and "other". */
export function reasonsFor(issue: Issue): Reason[] {
  const own: Reason[] = issue.id === 'content.alt_text' ? ['decorative'] : issue.id === 'content.size' ? ['size'] : issue.group === 'Subject' ? ['subject'] : []
  return [...own, 'wrong', 'other']
}

/** Why a block can't be dismissed, by check. */
const blockWhy: Record<string, keyof typeof mh> = {
  'links.blocklist': 'whyBlocklist',
  'compliance.footer': 'whyFooter',
  'subject.missing': 'whySubject',
  'personalization.syntax': 'whyLiquid',
}

function reasonLabel(t: ReturnType<typeof useT>, r: Reason): string {
  switch (r) {
    case 'decorative':
      return t(mh.reasonDecorative)
    case 'size':
      return t(mh.reasonSize)
    case 'subject':
      return t(mh.reasonSubject)
    case 'wrong':
      return t(mh.reasonWrong)
    default:
      return t(mh.reasonOther)
  }
}

const dismissalsPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/dismissals' as const

/** Dismiss…, the reason form, and the dismissed line with Undo. */
export function DismissControls({ issue, index, dismissed }: { issue: Issue; index: number; dismissed?: Dismissed }) {
  const c = useComposer()
  const t = useT()
  const id = useId()
  const [open, setOpen] = useState(false)
  const reasons = reasonsFor(issue)
  const [reason, setReason] = useState<Reason>(reasons[0]!)
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const path = { workspace: c.slug, campaignId: c.campaign.id }
  if (issue.severity === 'block') {
    return <p className="k m-0 dsm-why">{t(mh.cantDismiss, { why: t(mh[blockWhy[issue.id] ?? 'cantDismissDefault']) })}</p>
  }
  if (issue.severity !== 'warn' || c.kind === 'transactional') return null
  if (dismissed) {
    const d = dismissed.dismissal
    return (
      <div className="dsm-done" role="status">
        <span className="k">{t(mh.dismissedBy, { name: d.by, reason: d.reasonText })}</span>
        {c.campaign.access.canEdit ? (
          <button
            type="button"
            className="btn sm"
            disabled={busy}
            onClick={async () => {
              setBusy(true)
              try {
                await http.DELETE(`${dismissalsPath}/{dismissalId}`, { params: { path: { ...path, dismissalId: d.id } } })
                c.retryChecks?.()
              } catch (e) {
                toast({ kind: 'error', title: t(mh.dismissFailed), detail: errorMessage(e) })
              } finally {
                setBusy(false)
              }
            }}
          >
            {t(mh.undo)}
          </button>
        ) : null}
      </div>
    )
  }
  if (!c.campaign.access.canEdit) return null
  const key = c.checks?.issueKeys?.[index]
  if (!key) return null
  const back = reasons[0] === 'size' ? 'size' : reasons[0] === 'subject' ? 'subject' : 'other'
  const submit = async () => {
    setBusy(true)
    try {
      await http.POST(dismissalsPath, { params: { path }, body: { key, checkId: issue.id, title: issue.title, reason, note: note.trim() || undefined } })
      setOpen(false)
      setNote('')
      c.retryChecks?.()
    } catch (e) {
      toast({ kind: 'error', title: t(mh.dismissFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  if (!open)
    return (
      <button type="button" className="btn sm ghost dsm-open" onClick={() => setOpen(true)}>
        {t(mh.dismiss)}
      </button>
    )
  return (
    <div className="dsm" role="group" aria-labelledby={`${id}-why`}>
      <span id={`${id}-why`} className="dsm-h">
        {t(mh.whyDismiss)}
      </span>
      {reasons.map((r) => (
        <label key={r} className="rad">
          <input type="radio" name={`${id}-reason`} value={r} checked={reason === r} onChange={() => setReason(r)} />
          {reasonLabel(t, r)}
        </label>
      ))}
      <label className="lbl">
        {t(mh.note)}
        <input className="in" value={note} maxLength={500} placeholder={t(mh.notePlaceholder)} onChange={(e) => setNote(e.target.value)} aria-required={reason === 'other' || undefined} />
      </label>
      <span className="k">{t(mh.staysListed, { back })}</span>
      <div className="flex flex-wrap gap-2">
        <button type="button" className="btn sm pri" disabled={busy || (reason === 'other' && !note.trim())} aria-busy={busy || undefined} onClick={() => void submit()}>
          {t(mh.dismissWarning)}
        </button>
        <button type="button" className="btn sm" onClick={() => setOpen(false)}>
          {t(mh.cancel)}
        </button>
      </div>
    </div>
  )
}

/** The summary rows: Deliverability and Accessibility, with how they stand. */
export function ChecksSummary({ onAccessibility }: { onAccessibility?: () => void }) {
  const c = useComposer()
  const t = useT()
  const a11y = useAccessibility()
  const r = c.checks
  if (!r) return null
  const dismissed = new Set((r.dismissed ?? []).map((d) => d.index))
  const blocks = r.issues.filter((i) => i.severity === 'block').length
  const warnings = r.issues.filter((i, n) => i.severity === 'warn' && !dismissed.has(n)).length
  const a11yCount = a11y?.report?.issues.length ?? 0
  return (
    <div className="cksum">
      <div className="cksr">
        <span className="font-semibold">{t(mh.groupDeliverability)}</span>
        {blocks || warnings ? (
          <Badge tone={blocks ? 'bad' : 'warn'} dot>
            {t(mh.summaryIssues, { blocks, warnings, both: blocks && warnings ? 'yes' : 'no' })}
          </Badge>
        ) : (
          <Badge tone="ok" dot>
            {t(mh.pass)}
          </Badge>
        )}
      </div>
      {a11y?.enabled ? (
        <div className="cksr">
          <span className="font-semibold">{t(mh.groupAccessibility)}</span>
          {a11yCount ? (
            <button type="button" className="btn sm ghost" onClick={onAccessibility}>
              <Badge tone="warn" dot>
                {t(mh.summaryA11y, { count: a11yCount })}
              </Badge>
            </button>
          ) : (
            <Badge tone="ok" dot>
              {t(mh.pass)}
            </Badge>
          )}
        </div>
      ) : null}
      {/* Send Readiness's workspace rules and sender reputation, from its latest run. */}
      {c.kind !== 'transactional' ? <ReadinessRows slug={c.slug} campaignId={c.campaign.id} /> : null}
    </div>
  )
}

/** What passed, from the last check: size, alt text, links, the footer, the subject, images. */
export function PassedChecks() {
  const c = useComposer()
  const t = useT()
  const r = c.checks
  if (!r) return null
  const ids = new Set(r.issues.map((i) => i.id))
  const passed: string[] = []
  if (!ids.has('content.size')) passed.push(t(mh.passSize, { size: kb(r.sizeBytes) }))
  if (r.images && !r.imagesMissingAlt && !ids.has('content.alt_text')) passed.push(t(mh.passAlt, { count: r.images }))
  if (r.links && !r.linksBlocklisted && !r.linksBroken) passed.push(t(mh.passLinks, { count: r.links }))
  if (!ids.has('compliance.footer')) passed.push(t(mh.passFooter))
  if (!r.subjectHeuristics.length && !ids.has('subject.missing')) passed.push(t(mh.passSubject))
  if (!ids.has('content.image_ratio') && r.imageShare < 0.6) passed.push(t(mh.passRatio, { pct: Math.round(r.imageShare * 100) }))
  if (!ids.has('personalization.syntax') && !ids.has('personalization.unknown_field')) passed.push(t(mh.passLiquid))
  if (!passed.length) return null
  return (
    <details className="ckpass">
      <summary>
        <Badge tone="ok" dot>
          {t(mh.passedCount, { count: passed.length })}
        </Badge>
      </summary>
      <ul>
        {passed.map((p) => (
          <li key={p}>{p}</li>
        ))}
      </ul>
    </details>
  )
}
