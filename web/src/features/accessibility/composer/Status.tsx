import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { t as tt, useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { m } from './messages'
import { useAccessibility, type AccessibilityState } from './state'

type Checks = Schemas['ComposerChecks']

/**
 * The content checks without the images' alt text: the Accessibility tab
 * lists it now, so Deliverability counts one warning less (ComposerWarnings).
 */
export function withoutAltText(checks: Checks | undefined): Checks | undefined {
  if (!checks) return checks
  const alt = checks.issues.filter((i) => i.id === 'content.alt_text')
  if (!alt.length) return checks
  const issues = checks.issues.filter((i) => i.id !== 'content.alt_text')
  const blocks = issues.filter((i) => i.severity === 'block').length
  const warnings = issues.filter((i) => i.severity === 'warn').length
  return { ...checks, issues, blocks, warnings, verdict: blocks ? 'blocked' : warnings ? 'warnings' : 'ready' }
}

/** The right panel's tab: "Accessibility · 6", or "Accessibility"; none outside the campaign composer. */
export function accessibilityTab(a: AccessibilityState | null): string | undefined {
  if (!a?.enabled) return undefined
  const n = a.report?.issues.length ?? 0
  return n ? tt(m.tabCount, { count: n }) : tt(m.tab)
}

/** How many accessibility issues the status bar shows (0 while unknown). */
export function useAccessibilityIssueCount(): number {
  return useAccessibility()?.report?.issues.length ?? 0
}

/** " · Accessibility: all clear", after the size summary (Composer, Weekly Digest #142). */
export function AccessibilityClear() {
  const a = useAccessibility()
  const t = useT()
  if (!a?.report || a.error || a.report.issues.length) return null
  return <> · {t(m.statusAllClear)}</>
}

/** The status bar's chip: "6 accessibility issues (2 serious)", which opens the tab. */
export function AccessibilityChip() {
  const a = useAccessibility()
  const c = useComposer()
  const t = useT()
  const r = a?.report
  if (!r || a.error || !r.issues.length) return null
  const text = r.serious ? t(m.statusIssuesSerious, { count: r.issues.length, serious: r.serious }) : t(m.statusIssues, { count: r.issues.length })
  return (
    <button type="button" className="a11y-chip" onClick={() => c.setRightTab('accessibility')}>
      <Badge tone={r.serious && r.policy === 'block' ? 'bad' : 'warn'} dot>
        {text}
      </Badge>
    </button>
  )
}
