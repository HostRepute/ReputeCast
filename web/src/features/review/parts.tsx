import { useRouter } from '@tanstack/react-router'
import type { MouseEvent, ReactNode } from 'react'
import { Badge, type Tone } from '@/components/ui/Badge'
import type { ToastInput } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { clock, shortDate } from '@/lib/format'
import { CodeText } from '@/features/accessibility/review'
import { CheckRowsGrid } from '@/features/languages/review'
import { formatList, t, useT } from '@/i18n'
import { statusWord, type Check, type CheckStatus, type Fix, type SkippedKind } from './logic'
import { m } from './messages'

/** The 40px verdict mark of the banners (warn, bad, ok) and the 36px error mark. */
export function VerdictIcon({ kind, size = 40 }: { kind: 'warn' | 'bad' | 'ok' | 'error'; size?: number }) {
  const circle = kind === 'warn' ? '#FDE68A' : kind === 'ok' ? '#BBF7D0' : '#FECACA'
  return (
    <svg width={size} height={size} viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
      <circle cx="20" cy="20" r="19" fill={circle} />
      {kind === 'ok' ? (
        <path d="m12 20.5 5.5 5.5L28 15" stroke="#166534" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
      ) : kind === 'bad' ? (
        <path d="M13 13l14 14M27 13 13 27" stroke="#991B1B" strokeWidth="3" strokeLinecap="round" />
      ) : (
        <path d="M20 11v11M20 27v1" stroke={kind === 'warn' ? '#92400E' : '#991B1B'} strokeWidth="3" strokeLinecap="round" />
      )}
    </svg>
  )
}

const tone: Record<CheckStatus, Tone> = { pass: 'ok', warn: 'warn', block: 'bad', unknown: 'neutral' }

/** Pass, Warn, Block or Unknown. */
export function CheckBadge({ status, className }: { status: CheckStatus; className?: string }) {
  return (
    <Badge tone={tone[status]} className={className}>
      {statusWord(status)}
    </Badge>
  )
}

/**
 * A link to an app path the API handed out ("/w/acme/campaigns/…/edit?panel=…"),
 * navigated by the router; other links open in a new tab.
 */
export function AppHref({ href, className, describedBy, children }: { href: string; className?: string; describedBy?: string; children: ReactNode }) {
  const router = useRouter()
  if (!href.startsWith('/')) {
    return (
      <a href={href} className={className} aria-describedby={describedBy} target="_blank" rel="noopener noreferrer">
        {children}
      </a>
    )
  }
  const go = (e: MouseEvent<HTMLAnchorElement>) => {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0) return
    e.preventDefault()
    router.history.push(href)
  }
  return (
    <a href={href} className={className} aria-describedby={describedBy} onClick={go}>
      {children}
    </a>
  )
}

/**
 * Why the reader can't send, change the campaign or its route: who can,
 * by name ("Marcus Chen or Linh Nguyen can send it."), never a permission
 * key (SR-13).
 */
export function whoCan(what: 'send' | 'edit' | 'route', helpers: string[]): string {
  return t(m.whoCan, { what, names: helpers.length ? formatList(helpers, 'disjunction') : 'none' })
}

/**
 * Why the send can't go out yet, in the order a person deals with it: who
 * can send, a change of theirs still being saved (a fix, Include them, an
 * undo: R1.U4), blocks, checks that couldn't run, a campaign that changed
 * since the checks ran. Undefined when it can.
 */
export function sendLockReason(o: {
  canSend: boolean
  helpers: string[]
  changing: boolean
  blocks: number
  /** Checks couldn't run and nobody continued without them: which kind (skippedKind). */
  incomplete?: SkippedKind
  stale: boolean
}): string | undefined {
  if (!o.canSend) return whoCan('send', o.helpers)
  if (o.changing) return t(m.lockedWhileChanging)
  if (o.blocks > 0) return t(m.fixBlocksFirst, { count: o.blocks })
  if (o.incomplete) return t(m.someChecksFailed, { kind: o.incomplete })
  if (o.stale) return t(m.changedRunAgain)
  return undefined
}

/** Who may apply a fix: route changes need the right to send, the rest the right to edit. */
export function fixReason(f: Fix, can: { edit: boolean; send: boolean; editable: boolean }, helpers: { send: string[]; edit: string[] }): string | undefined {
  if (!can.editable) return t(m.alreadySendingNoChange)
  if (f.action === 'route_only' && !can.send) return whoCan('route', helpers.send)
  if (!can.edit) return whoCan('edit', helpers.edit)
  return undefined
}

/**
 * The toast after one-click fixes: the fixes as the server names them, and
 * Undo while it can put the campaign back as it was (R1.U3).
 */
export function fixesToast(applied: string[], undo?: () => void): ToastInput {
  return {
    kind: 'success',
    title: applied.length === 1 ? applied[0]! : t(m.fixesApplied, { count: applied.length }),
    detail: applied.length > 1 ? applied.join(' · ') : t(m.readinessRunsAgain),
    undo,
  }
}

/**
 * Who may undo fixes applied together (R1.U3): as applying them, a batch
 * that changed the route needs the right to send, the rest the right to
 * edit.
 */
export function undoReason(
  batch: { route?: boolean },
  can: { edit: boolean; send: boolean; editable: boolean },
  helpers: { send: string[]; edit: string[] },
): string | undefined {
  if (!can.editable) return t(m.alreadySendingNoChange)
  if (batch.route && !can.send) return whoCan('route', helpers.send)
  if (!can.edit) return whoCan('edit', helpers.edit)
  return undefined
}

/** One row of the checklist: badge, the finding and its evidence, the fixes. */
export function CheckRow({
  check,
  title,
  showDetail = true,
  alert,
  children,
}: {
  check: Check
  title?: ReactNode
  showDetail?: boolean
  /** A check that couldn't run, shown as an error (the Accessibility group's). */
  alert?: boolean
  children?: ReactNode
}) {
  // Records, headers and error codes are behind "Technical details" (R1.U2).
  const tech = check.technical?.length || check.raw?.length
  return (
    <div className={cn('rv-row', check.status === 'warn' && 'r-warn', (check.status === 'block' || alert) && 'r-bad')} role={alert ? 'alert' : undefined}>
      <CheckBadge status={check.status} />
      <div className="rv-rt">
        <span className="rv-t">{title ?? <CodeText text={check.title} />}</span>
        {showDetail && check.detail ? <span className={cn('k', check.mono && 'mono')}>{check.detail}</span> : null}
        {check.rows?.length ? <CheckRowsGrid rows={check.rows} /> : null}
        {tech ? <TechDetails check={check} /> : null}
      </div>
      {children}
    </div>
  )
}

/** "09:40" today, "Mon 12 Oct 09:40" on another day (display timezone). */
export function whenText(d: string | Date, now: Date = new Date()): string {
  return shortDate(d) === shortDate(now) ? clock(d) : `${shortDate(d)} ${clock(d)}`
}

/** "Technical details": what the finding rests on, and the raw records or answers. */
export function TechDetails({ check: c }: { check: Check }) {
  const say = useT()
  return (
    <details className="rv-tech">
      <summary>
        <span>{say(m.technicalDetails)}</span>
        <Chevron />
      </summary>
      <div className="rv-tpanel">
        {c.technical?.length ? (
          <dl className="rv-tdl">
            {c.technical.map((r, i) => (
              <div key={i} className="contents">
                <dt>{r.label}</dt>
                <dd className={r.before ? 'mono' : undefined}>{r.value}</dd>
              </div>
            ))}
          </dl>
        ) : null}
        {c.raw?.length ? <pre className="rv-raw">{c.raw.join('\n')}</pre> : null}
      </div>
    </details>
  )
}

export function Chevron() {
  return (
    <svg className="rv-chev" width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
      <path d="m4 6 4 4 4-4" />
    </svg>
  )
}
