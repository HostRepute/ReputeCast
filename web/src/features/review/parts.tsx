import { useRouter } from '@tanstack/react-router'
import type { MouseEvent, ReactNode } from 'react'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { cn } from '@/lib/cn'
import { clock, shortDate } from '@/lib/format'
import { deniedReason } from '@/lib/session'
import { CodeText, Swatch } from '@/features/accessibility/review'
import { statusWord, type Check, type CheckStatus, type Fix } from './logic'

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
export function AppHref({ href, className, children }: { href: string; className?: string; children: ReactNode }) {
  const router = useRouter()
  if (!href.startsWith('/')) {
    return (
      <a href={href} className={className} target="_blank" rel="noopener noreferrer">
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
    <a href={href} className={className} onClick={go}>
      {children}
    </a>
  )
}

/** Who may apply a fix: route changes need campaigns:send, the rest campaigns:edit. */
export function fixReason(f: Fix, can: { edit: boolean; send: boolean; editable: boolean }, helpers: string[]): string | undefined {
  if (!can.editable) return 'This campaign already started sending, so it can’t change.'
  if (f.action === 'route_only' && !can.send) return deniedReason('campaigns:send', helpers).replace('Sending needs', 'Changing the route of a send needs')
  if (!can.edit) return deniedReason('campaigns:edit', helpers)
  return undefined
}

/** The fixes of a check: one-click buttons and links to where it's fixed. */
export function FixButtons({
  check,
  busy,
  reasonFor,
  onApply,
}: {
  check: Check
  busy?: string
  reasonFor: (f: Fix) => string | undefined
  onApply: (c: Check, f: Fix) => void
}) {
  if (!check.fixes.length) return null
  const buttons = check.fixes.map((f, i) =>
    f.kind === 'apply' ? (
      <Button
        key={i}
        size="sm"
        variant={f.tone === 'primary' ? 'primary' : f.tone === 'danger' ? 'danger-primary' : 'default'}
        disabledReason={reasonFor(f)}
        loading={busy === `${check.key}:${f.action}`}
        onClick={() => onApply(check, f)}
      >
        {f.action === 'accessibility_color' && f.params?.color ? <Swatch color={f.params.color} /> : null}
        {f.label}
      </Button>
    ) : (
      <AppHref key={i} href={f.href ?? '#'} className="btn sm">
        {f.label}
      </AppHref>
    ),
  )
  return buttons.length === 1 ? buttons[0]! : <div className="rv-fixes">{buttons}</div>
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
  return (
    <div className={cn('rv-row', check.status === 'warn' && 'r-warn', (check.status === 'block' || alert) && 'r-bad')} role={alert ? 'alert' : undefined}>
      <CheckBadge status={check.status} />
      <div className="rv-rt">
        <span className="rv-t">{title ?? <CodeText text={check.title} />}</span>
        {showDetail && check.detail ? <span className={cn('k', check.mono && 'mono')}>{check.detail}</span> : null}
      </div>
      {children}
    </div>
  )
}

/** "09:40" today, "Mon 12 Oct 09:40" on another day (display timezone). */
export function whenText(d: string | Date, now: Date = new Date()): string {
  return shortDate(d) === shortDate(now) ? clock(d) : `${shortDate(d)} ${clock(d)}`
}
