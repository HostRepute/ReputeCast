import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { groupTitles, type Check, type Run } from './logic'
import { m } from './messages'
import { CheckBadge } from './parts'

/*
 * The running and the incomplete states' lists. The checklist itself is
 * Needs attention and Passed (Issues.tsx, SR-13).
 */

/** "Loading · checks running": one line per group as it ticks in. */
export function RunningList({ progress }: { progress: Schemas['ReadinessProgress'] }) {
  const t = useT()
  const rank = (s: string) => (s === 'running' ? 1 : s === 'queued' ? 2 : 0)
  const groups = [...progress.groups].sort((a, b) => rank(a.status) - rank(b.status))
  return (
    <div className="card flex flex-col" aria-live="polite">
      {groups.map((g) => (
        <div key={g.key} className={cn('rv-line', g.status === 'queued' && 'l-queued')}>
          {g.status === 'running' ? (
            <span className="spin" aria-hidden="true" />
          ) : g.status === 'queued' ? (
            <Badge tone="neutral">{t(m.queued)}</Badge>
          ) : (
            <CheckBadge status={g.status} className="pop" />
          )}
          <span>
            {groupTitles[g.key] ?? g.title}
            {g.text ? ` · ${g.text}` : ''}
          </span>
          {g.status === 'running' && g.checked > 0 && g.total > 0 ? (
            <span className="k ml-auto">{t(m.ofTotal, { checked: g.checked, total: g.total })}</span>
          ) : null}
          {g.status === 'running' ? <span className="vh">{t(m.runningNow)}</span> : null}
        </div>
      ))}
    </div>
  )
}

/**
 * "Error · a check could not run": the checks that couldn't run and the
 * other checks of their groups, which still have answers.
 */
export function UnknownList({ run }: { run: Run }) {
  const t = useT()
  const groups = new Set(run.checks.filter((c) => c.status === 'unknown').map((c) => c.group))
  const rows = run.checks
    .filter((c) => groups.has(c.group))
    .sort((a, b) => (a.status === 'unknown' ? 0 : 1) - (b.status === 'unknown' ? 0 : 1))
  const shares = run.estimates.route.shares
  // Provider rates come from the providers themselves: they still answer.
  const text = (c: Check) => {
    const id = c.key.startsWith('reputation.provider.') ? c.key.slice('reputation.provider.'.length) : ''
    const share = id ? shares.find((s) => s.providerId === id) : undefined
    if (c.status === 'pass' && share && c.short) {
      const from = share.pricing === 'self_hosted' ? share.short : t(m.fromApi, { name: share.short })
      // The short finding is the server's, in the reader's language; in
      // English this line joins its two rates with "and".
      return t(m.unaffected, { group: groupTitles[c.group] ?? c.group, short: c.short.replace(', complaints ', ' and complaint '), source: from })
    }
    return (c.flatTitle || c.title).replace(/^([^:]+): /, '$1 · ')
  }
  return (
    <div className="card flex flex-col">
      {rows.map((c) => (
        <div key={c.key} className="rv-line">
          <CheckBadge status={c.status} />
          <span>{text(c)}</span>
          {c.status === 'unknown' && c.note ? <span className="k ml-auto">{c.note}</span> : null}
        </div>
      ))}
    </div>
  )
}
