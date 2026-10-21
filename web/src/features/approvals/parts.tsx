import { Fragment, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, type Tone } from '@/components/ui/Badge'
import { t, useT, type MessageDescriptor } from '@/i18n'
import { cn } from '@/lib/cn'
import { andList, clock, count, monthDayTime, orList, pct, shortDate } from '@/lib/format'
import { m } from './messages'
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

const statusText: Record<RequestStatus, MessageDescriptor> = {
  pending: m.statusPending,
  approved: m.statusApproved,
  changes_requested: m.statusChangesRequested,
  rejected: m.statusRejected,
  invalidated: m.statusInvalidated,
  bypassed: m.statusBypassed,
  sent: m.statusSent,
  cancelled: m.statusCancelled,
}

export function requestTone(s: RequestStatus): Tone {
  return statusTone[s]
}

/** Pending, Approved, Changes requested, Rejected, Invalidated… */
export function ApprovalStatusBadge({ status, long }: { status: RequestStatus; long?: boolean }) {
  const t = useT()
  return <Badge tone={statusTone[status]}>{t(long && status === 'pending' ? m.statusPendingLong : statusText[status])}</Badge>
}

/** A request status in words: "Changes requested". */
export function statusWord(s: RequestStatus): string {
  return t(statusText[s])
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
  const t = useT()
  const label = labelled ?? changes.length > 1
  const line = (c: Change, value: string) => (label ? t(m.diffLabelled, { label: c.label, value }) : value)
  return (
    <div className="diff" role="group" aria-label={t(m.whatChanged)}>
      {changes.map((c, i) => (
        <Fragment key={i}>
          {c.before ? (
            <div className="del">
              <span className="vh">{t(m.before)} </span>− {line(c, c.before)}
            </div>
          ) : null}
          {c.after ? (
            <div className="add">
              <span className="vh">{t(m.after)} </span>+ {line(c, c.after)}
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
    // The approved count, with how far it moved since (ApprovalStates: "Audience 47,912 (+0.3%)").
    const n = count(snapshot.audienceCount)
    parts.push(
      audience
        ? t(m.unchangedAudienceMoved, { count: n, change: `${audience.changePct >= 0 ? '+' : '−'}${pct(Math.abs(audience.changePct), 1)}` })
        : t(m.unchangedAudience, { count: n }),
    )
  }
  if (!fields.has('route')) parts.push(t(m.unchangedRoute))
  if (!fields.has('schedule')) parts.push(snapshot.scheduleMode === 'now' ? t(m.unchangedScheduleNow) : t(m.unchangedSchedule, { when: snapshot.scheduleText }))
  return parts.join(' · ')
}

const fieldWords = { audience: m.fieldAudience, route: m.fieldRoute, schedule: m.fieldSchedule } as const

/** Which of audience, route and schedule stayed the same: "Audience, route and schedule are unchanged." */
export function unchangedSentence(changes: Change[]): string {
  const fields = new Set(changes.map((c) => (c.field === 'audience_count' ? 'audience' : c.field)))
  const same = (['audience', 'route', 'schedule'] as const).filter((f) => !fields.has(f))
  if (!same.length) return ''
  return capital(t(m.unchangedSentence, { count: same.length, fields: andList(same.map((f) => t(fieldWords[f]))) }))
}

/** The sentence with its first letter in capitals, whatever word the language starts it with. */
function capital(s: string): string {
  return s ? s[0]!.toUpperCase() + s.slice(1) : s
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
  return shortDate(d) === shortDate(now) ? clock(d) : monthDayTime(d)
}

/** "rules: audience ≥ 10,000 · warming-up provider" for a list of rule phrases. */
export function rulesLine(rules: string[]): string {
  if (!rules.length) return ''
  return t(m.rulesLine, { count: rules.length, rules: rules.join(' · ') })
}
