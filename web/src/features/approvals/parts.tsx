import { Fragment, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, type Tone } from '@/components/ui/Badge'
import { cn } from '@/lib/cn'
import { andList, clock, monthDay, orList, shortDate } from '@/lib/format'
import './approvals.css'

export type Person = Schemas['ApprovalPerson']
export type Change = Schemas['ApprovalChange']
export type RequestStatus = Schemas['ApprovalRequestView']['status']

const statusTone: Record<RequestStatus, Tone> = {
  pending: 'warn',
  approved: 'ok',
  changes_requested: 'info',
  rejected: 'bad',
  invalidated: 'bad',
  bypassed: 'purple',
  sent: 'ok',
  cancelled: 'neutral',
}

const statusText: Record<RequestStatus, string> = {
  pending: 'Pending',
  approved: 'Approved',
  changes_requested: 'Changes requested',
  rejected: 'Rejected',
  invalidated: 'Invalidated',
  bypassed: 'Bypassed',
  sent: 'Sent',
  cancelled: 'Cancelled',
}

export function requestTone(s: RequestStatus): Tone {
  return statusTone[s]
}

/** Pending, Approved, Changes requested, Rejected, Invalidated… */
export function ApprovalStatusBadge({ status, long }: { status: RequestStatus; long?: boolean }) {
  return <Badge tone={statusTone[status]}>{long && status === 'pending' ? 'Pending approval' : statusText[status]}</Badge>
}

/** The verdict strip at the top of an approval card. */
export function MiniBanner({ tone, title, children }: { tone: 'warn' | 'ok' | 'bad'; title: ReactNode; children?: ReactNode }) {
  return (
    <div className={cn('ap-mini', `m-${tone}`)} role={tone === 'bad' ? 'alert' : 'status'}>
      <div className="col gap-0.5!">
        <span className="ap-mini-t">{title}</span>
        {children ? <span className="ap-mini-s">{children}</span> : null}
      </div>
    </div>
  )
}

/** "− Subject: …" / "+ Subject: …"; one change reads without its label. */
export function DiffLines({ changes, labelled }: { changes: Change[]; labelled?: boolean }) {
  const label = labelled ?? changes.length > 1
  return (
    <div className="diff" role="group" aria-label="What changed">
      {changes.map((c, i) => (
        <Fragment key={i}>
          {c.before ? (
            <div className="del">
              <span className="vh">Before: </span>− {label ? `${c.label}: ` : ''}
              {c.before}
            </div>
          ) : null}
          {c.after ? (
            <div className="add">
              <span className="vh">After: </span>+ {label ? `${c.label}: ` : ''}
              {c.after}
            </div>
          ) : null}
        </Fragment>
      ))}
    </div>
  )
}

/** What a request covers that didn't change: "Audience 47,912 (+0.3%) · route · schedule Nov 4, 09:00". */
export function unchangedText(changes: Change[], snapshot: Schemas['ApprovalSnapshotSummary'], audience?: { current: number; changePct: number }) {
  const fields = new Set(changes.map((c) => (c.field === 'audience_count' ? 'audience' : c.field)))
  const parts: string[] = []
  if (!fields.has('audience')) {
    const pct = audience ? ` (${audience.changePct >= 0 ? '+' : '−'}${Math.abs(audience.changePct).toFixed(1)}%)` : ''
    // The approved count, with how far it moved since (ApprovalStates: "Audience 47,912 (+0.3%)").
    parts.push(`Audience ${snapshot.audienceCount.toLocaleString('en-US')}${pct}`)
  }
  if (!fields.has('route')) parts.push('route')
  if (!fields.has('schedule')) parts.push(snapshot.scheduleMode === 'now' ? 'schedule: send now' : `schedule ${snapshot.scheduleText}`)
  return parts.join(' · ')
}

/** Which of audience, route and schedule stayed the same: "Audience, route and schedule are unchanged." */
export function unchangedSentence(changes: Change[]): string {
  const fields = new Set(changes.map((c) => (c.field === 'audience_count' ? 'audience' : c.field)))
  const same = (['audience', 'route', 'schedule'] as const).filter((f) => !fields.has(f))
  if (!same.length) return ''
  const words = same.map((f, i) => (i === 0 ? f[0]!.toUpperCase() + f.slice(1) : f))
  return `${andList(words)} ${same.length === 1 ? 'is' : 'are'} unchanged.`
}

/** "Avatar + name + mono line" used for approvers and authors. */
export function PersonLine({ person, line, size = 28 }: { person: Person; line?: ReactNode; size?: number }) {
  return (
    <div className="ap-person">
      <Avatar name={person.name || person.email} color={person.color} size={size} />
      <div>
        <span className="font-semibold">{person.name || person.email}</span>
        {line ? <span className="k mono">{line}</span> : null}
      </div>
    </div>
  )
}

/** The approver who decided first, then the others (who was told, in the boards' order). */
export function deciderFirst(people: Person[], decidedBy?: string | null): Person[] {
  const i = decidedBy ? people.findIndex((p) => p.name === decidedBy) : -1
  return i > 0 ? [people[i]!, ...people.slice(0, i), ...people.slice(i + 1)] : people
}

export function approverNames(people: Person[], join: 'or' | 'and' | 'comma' = 'or'): string {
  const names = people.map((p) => p.name || p.email)
  return join === 'or' ? orList(names) : join === 'and' ? andList(names) : names.join(', ')
}

/** "Oct 10, 16:20" or "10:42" today. */
export function decidedText(d: string | Date, now: Date = new Date()): string {
  return shortDate(d) === shortDate(now) ? clock(d) : `${monthDay(d)}, ${clock(d)}`
}

/** "rules: audience ≥ 10,000 · warming-up provider" for a list of rule phrases. */
export function rulesLine(rules: string[]): string {
  if (!rules.length) return ''
  return `${rules.length === 1 ? 'rule' : 'rules'}: ${rules.join(' · ')}`
}
