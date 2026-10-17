import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'

import { T, useT, type MessageDescriptor } from '@/i18n'
import type { AIFeature } from './api'
import { langName } from './format'
import { m } from './messages'

type ActivityRow = Schemas['AIActivityRow']

/** Rich text in the assistant's messages: <b>, <mono>. */
export const tags = {
  b: (c: ReactNode[]) => <b>{c}</b>,
  mono: (c: ReactNode[]) => <span className="mono">{c}</span>,
}

export const featureName: Record<AIFeature, MessageDescriptor> = {
  subject_ideas: m.featureSubjectIdeas,
  rewrite: m.featureRewrite,
  translate: m.featureTranslate,
  alt_text: m.featureAltText,
  accessibility_fix: m.featureAccessibility,
  segment_draft: m.featureSegment,
  report_summary: m.featureSummary,
}

/** What each feature sends ("Sends: …"), in the settings and the connect dialog. */
export const featureSends: Record<AIFeature, MessageDescriptor> = {
  subject_ideas: m.sendsSubjectIdeas,
  rewrite: m.sendsRewrite,
  translate: m.sendsTranslate,
  alt_text: m.sendsAltText,
  accessibility_fix: m.sendsAccessibility,
  segment_draft: m.sendsSegment,
  report_summary: m.sendsSummary,
}

/** The activity log's feature column: "Translate to Spanish", "Shorten". */
export function rowFeature(t: ReturnType<typeof useT>, r: Pick<ActivityRow, 'feature' | 'action'>): string {
  switch (r.feature) {
    case 'subject_ideas':
      return t(m.rowSubjectIdeas)
    case 'rewrite':
      return t(m.rowRewrite, { action: r.action ?? '' })
    case 'translate':
      return r.action ? t(m.rowTranslate, { language: langName(r.action) }) : t(m.featureTranslate)
    case 'alt_text':
      return t(m.rowAltText)
    case 'accessibility_fix':
      return t(m.rowAccessibility)
    case 'segment_draft':
      return t(m.rowSegment)
    case 'report_summary':
      return t(m.rowSummary)
  }
  return r.feature
}

/** The outcome's badge: "Accepted 1 of 5", "Refused by the model", "Error · 529 overloaded". */
export function OutcomeBadge({ outcome, detail }: { outcome: ActivityRow['outcome']; detail?: string }) {
  const t = useT()
  switch (outcome) {
    case 'accepted':
      return <Badge tone="ok">{detail ? t(m.badgeAcceptedOf, { detail }) : t(m.badgeAccepted)}</Badge>
    case 'accepted_edits':
      return <Badge tone="ok">{t(m.badgeAcceptedEdits)}</Badge>
    case 'kept':
      return <Badge tone="ok">{t(m.badgeKept)}</Badge>
    case 'rejected':
      return <Badge tone="neutral">{t(m.badgeRejected)}</Badge>
    case 'discarded':
      return <Badge tone="neutral">{t(m.badgeDiscarded)}</Badge>
    case 'refused':
      return <Badge tone="neutral">{t(m.badgeRefused)}</Badge>
    case 'error':
      return <Badge tone="bad">{detail ? t(m.badgeErrorDetail, { detail }) : t(m.badgeError)}</Badge>
    case 'nothing_drafted':
      return <Badge tone="neutral">{t(m.badgeNothing)}</Badge>
    case 'cancelled':
      return <Badge tone="neutral">{t(m.badgeCancelled)}</Badge>
    default:
      return <Badge tone="neutral">{t(m.badgePending)}</Badge>
  }
}

const noteClass = { info: 'i', warn: 'w', bad: 'b', ok: 'g', neutral: 'n' } as const

/** A note with the boards' leading icon: ⓘ for information, ! for problems, ✓ when it worked. */
export function AINote({ tone = 'info', children, className, role }: { tone?: keyof typeof noteClass; children: ReactNode; className?: string; role?: string }) {
  return (
    <div className={`note ${noteClass[tone]}${className ? ` ${className}` : ''}`} role={role ?? (tone === 'bad' ? 'alert' : 'status')}>
      <Icon name={tone === 'ok' ? 'check' : tone === 'bad' || tone === 'warn' ? 'alertCircle' : 'info'} size={16} className="mt-px flex-none" />
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  )
}

/** The masked key's crossed-out eye. */
export function EyeOffIcon({ size = 14 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="var(--ink-3)" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 8s2.2-4.5 6-4.5S14 8 14 8s-2.2 4.5-6 4.5S2 8 2 8Z" />
      <circle cx="8" cy="8" r="1.8" />
      <path d="M2.5 2.5l11 11" />
    </svg>
  )
}

/**
 * "What leaves this instance" (the settings) or "What goes to Anthropic"
 * (the connect dialog): what is sent, and what never is.
 */
export function LeavesNote({ provider }: { provider?: string }) {
  const t = useT()
  const token = '{{ first_name | default: "there" }}'
  return (
    <AINote tone="info">
      <span className="font-semibold">{provider ? t(m.goesTo, { provider }) : t(m.leavesTitle)}</span>
      <div className="mt-1.5 flex flex-col gap-1.5">
        <span className="font-semibold">{t(m.sent)}</span>
        <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px]">
          {provider ? (
            <>
              <li>{t(m.goesText)}</li>
              <li>{t(m.goesSegments)}</li>
            </>
          ) : (
            <>
              <li>{t(m.leavesOnly)}</li>
              <li>
                <T msg={m.leavesTokens} values={{ ...tags, token }} />
              </li>
              <li>{t(m.leavesSummaries)}</li>
            </>
          )}
        </ul>
        <span>
          <T msg={m.neverSent} values={tags} />
        </span>
        {provider ? null : <span>{t(m.noHostRepute)}</span>}
      </div>
    </AINote>
  )
}
