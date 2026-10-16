import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { formatList, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

type Report = Schemas['AccessibilityReport']
type Issue = Schemas['AccessibilityIssue']
type Run = Schemas['ReadinessRun']

/**
 * When every block of a run is an accessibility issue (the workspace's
 * policy is Block): how many, and the group's note on the policy.
 */
export function accessibilityBlock(run: Run): { count: number; note: string } | undefined {
  const blocks = run.checks.filter((c) => c.status === 'block')
  if (!blocks.length || !blocks.every((c) => c.group === 'accessibility')) return undefined
  return { count: blocks.length, note: run.groups.find((g) => g.key === 'accessibility')?.note ?? '' }
}

/** Send Readiness groups that are new in this phase get a "New" badge. */
export const newGroups: ReadonlySet<string> = new Set(['accessibility'])

/** The "New" badge of a group heading. */
export function NewBadge() {
  const t = useT()
  return (
    <Badge tone="info" className="h-5 text-[11px]">
      {t(m.newBadge)}
    </Badge>
  )
}

/**
 * Text from the API with `code` in backticks: "`lang` and `dir` set for
 * every language". Other text is left alone.
 */
export function CodeText({ text }: { text: string }): ReactNode {
  const parts = text.split('`')
  if (parts.length < 3 || parts.length % 2 === 0) return text
  return parts.map((p, i) =>
    i % 2 === 1 ? (
      <code key={i} className="mono text-[0.95em]">
        {p}
      </code>
    ) : (
      p
    ),
  )
}

/** The small square of a suggested color ("Use #0F766E"). */
export function Swatch({ color }: { color: string }) {
  return <span aria-hidden="true" className="inline-block h-3.5 w-3.5 flex-none rounded border border-black/10" style={{ background: color }} />
}

/** The language names of codes, from the report: "English, Spanish and German". */
function languageNames(report: Report, codes: string[]): string {
  return formatList(codes.map((c) => report.languages.find((l) => l.code === c)?.name ?? c))
}

/**
 * "Fix 2 in one click" (5.13): what the one-click fixes change in every
 * language, before they're applied. The checks run again right after.
 */
export function OneClickDialog({
  campaignId,
  onOpenChange,
  onConfirm,
}: {
  campaignId: string
  onOpenChange: (open: boolean) => void
  onConfirm: () => Promise<void>
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/accessibility', { params: { path: { workspace: slug, campaignId } } })
  const [busy, setBusy] = useState(false)
  const report = q.data
  const fixes = report ? report.issues.filter((i) => i.severity === 'moderate' && i.fix.oneClick) : []
  const stay = report ? report.issues.filter((i) => i.severity === 'moderate' && !i.fix.oneClick) : []
  const many = (report?.languages.length ?? 1) > 1
  const n = fixes.length
  const confirm = async () => {
    setBusy(true)
    try {
      await onConfirm()
      onOpenChange(false)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => !busy && onOpenChange(open)}
      hideClose
      eyebrow={<span className="k">{t(m.oneClickEyebrow)}</span>}
      title={q.isSuccess ? (many ? t(m.oneClickTitle, { count: n }) : t(m.oneClickTitleOne, { count: n })) : t(m.oneClickTitleLoading)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)} disabled={busy}>
            {t(common.cancel)}
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabledReason={q.isPending ? t(m.oneClickLoading) : n === 0 ? t(m.oneClickNothing) : undefined}
            onClick={() => void confirm()}
          >
            {many ? t(m.oneClickConfirm, { count: n }) : t(m.oneClickConfirmOne, { count: n })}
          </Button>
        </>
      }
    >
      {q.isPending ? (
        <div className="col gap-2" aria-busy="true">
          <span className="k" role="status">
            {t(m.oneClickLoading)}
          </span>
          <Skeleton h={120} />
        </div>
      ) : q.isError ? (
        <Note tone="warn">{t(m.oneClickFailed, { error: errorMessage(q.error) })}</Note>
      ) : n === 0 ? (
        <Note tone="neutral">{t(m.oneClickNothing)}</Note>
      ) : (
        <>
          <div className="diff">
            {fixes.map((i) => (
              <FixLines key={i.id} issue={i} languages={languageNames(report!, i.languages)} />
            ))}
          </div>
          {stay.length ? <Note tone="neutral">{t(m.oneClickStay, { items: formatList(stay.map((i) => i.short)) })}</Note> : null}
          <span className="k">{t(m.oneClickAfter)}</span>
        </>
      )}
    </Dialog>
  )
}

function FixLines({ issue, languages }: { issue: Issue; languages: string }) {
  const t = useT()
  const f = issue.fix
  const head = (
    <div className="ctx bg-ground-3 font-semibold">
      {t(m.oneClickSubject, { subject: f.subject ?? issue.title, languages })}
    </div>
  )
  if (issue.kind === 'heading' && f.headings.length) {
    return (
      <>
        {head}
        <div className="del">{t(m.oneClickHeadingFrom, { level: f.headings[0]!.from, texts: f.headings.map((h) => t(m.quoted, { text: h.text })).join(', ') })}</div>
        <div className="add">{t(m.oneClickHeadingTo, { level: f.headings[0]!.to })}</div>
      </>
    )
  }
  return (
    <>
      {head}
      <div className="del">{t(m.oneClickSizeFrom, { px: f.fromPx ?? 0 })}</div>
      <div className="add">{t(m.oneClickSizeTo, { px: f.toPx ?? 0 })}</div>
    </>
  )
}
