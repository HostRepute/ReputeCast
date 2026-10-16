import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { m as am } from '@/features/accessibility/messages'
import { NewBadge, newGroups } from '@/features/accessibility/review'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { bySeverity, checksByGroup, flatGroupLabels, flatTitle, groupTitles, type Check, type Fix, type Run } from './logic'
import { m } from './messages'
import { CheckBadge, CheckRow, FixButtons } from './parts'

type Actions = {
  busy?: string
  reasonFor: (f: Fix) => string | undefined
  onApply: (c: Check, f: Fix) => void
  /** "Run again" on a check that couldn't run (the Accessibility group's). */
  onRerun?: () => void
}

/**
 * The Send Readiness checklist. Grouped by category as on SendReadiness; when
 * the verdict is Blocked it lists blocks first, then warnings, then a
 * collapsed list of what passed (SendReadinessBlocked).
 */
export function Checklist({ run, ...actions }: { run: Run } & Actions) {
  const t = useT()
  return (
    <section className="card flex min-w-0 flex-col" aria-label={t(m.checklistLabel)}>
      {run.verdict === 'blocked' ? <BySeverity run={run} {...actions} /> : <ByGroup run={run} {...actions} />}
    </section>
  )
}

function ByGroup({ run, ...actions }: { run: Run } & Actions) {
  return (
    <>
      {checksByGroup(run).map(({ group, checks }) => (
        <GroupRows key={group.key} group={group.key} title={groupTitles[group.key] ?? group.title} caption={group.caption} checks={checks} {...actions} />
      ))}
    </>
  )
}

function GroupRows({ group, title, caption, checks, ...actions }: { group: string; title: string; caption: string; checks: Check[] } & Actions) {
  return (
    <>
      <div className="rv-grp">
        <h2 className="h2 flex items-center gap-2">
          {title}
          {newGroups.has(group) ? <NewBadge /> : null}
        </h2>
        {caption ? <span className="k">{caption}</span> : null}
      </div>
      {checks.map((c) => {
        const rerun = c.status === 'unknown' && newGroups.has(c.group)
        return (
          <CheckRow key={c.key} check={c} alert={rerun}>
            {c.note ? <span className="k ml-auto">{c.note}</span> : null}
            <FixButtons check={c} {...actions} />
            {rerun && actions.onRerun ? <RerunButton onRerun={actions.onRerun} /> : null}
          </CheckRow>
        )
      })}
    </>
  )
}

function RerunButton({ onRerun }: { onRerun: () => void }) {
  const t = useT()
  return (
    <Button size="sm" variant="primary" onClick={onRerun}>
      {t(am.runAgain)}
    </Button>
  )
}

function BySeverity({ run, ...actions }: { run: Run } & Actions) {
  const t = useT()
  const [all, setAll] = useState(false)
  const blocks = run.checks.filter((c) => c.status === 'block').sort(bySeverity)
  const warns = run.checks.filter((c) => c.status === 'warn').sort(bySeverity)
  const unknown = run.checks.filter((c) => c.status === 'unknown')
  const passes = run.checks.filter((c) => c.status === 'pass')
  const summaries = run.groups.filter((g) => g.passed > 0 && g.summary)
  return (
    <>
      <div className="rv-grp g-bad">
        <h2 className="h2">{t(m.blockingIssues, { count: blocks.length })}</h2>
        <span className="k">{t(m.fixFirst)}</span>
      </div>
      {blocks.map((c) => (
        <CheckRow key={c.key} check={c} title={flatTitle(c)}>
          <FixButtons check={c} {...actions} />
        </CheckRow>
      ))}
      {warns.length ? (
        <>
          <div className="rv-grp">
            <h2 className="h2">{t(m.warningsTitle, { count: warns.length })}</h2>
          </div>
          {warns.map((c) => (
            <CheckRow key={c.key} check={c} title={flatTitle(c)}>
              <FixButtons check={c} {...actions} />
            </CheckRow>
          ))}
        </>
      ) : null}
      {unknown.length ? (
        <>
          <div className="rv-grp">
            <h2 className="h2">{t(m.notChecked, { count: unknown.length })}</h2>
            <span className="k">{t(m.runChecksAgain)}</span>
          </div>
          {unknown.map((c) => (
            <CheckRow key={c.key} check={c} title={flatTitle(c)}>
              {c.note ? <span className="k ml-auto">{c.note}</span> : null}
            </CheckRow>
          ))}
        </>
      ) : null}
      {passes.length ? (
        <>
          <div className="rv-grp">
            <h2 className="h2">{t(m.passedTitle, { count: passes.length })}</h2>
            <span className="k">{all ? t(m.expanded) : t(m.collapsed)}</span>
          </div>
          {all
            ? passes.map((c, i) => (
                <CheckRow key={c.key} check={c} title={flatTitle(c)}>
                  {i === passes.length - 1 ? (
                    <Button size="sm" onClick={() => setAll(false)} aria-expanded="true">
                      {t(m.collapse)}
                    </Button>
                  ) : null}
                </CheckRow>
              ))
            : summaries.map((g, i) => (
                <div key={g.key} className="rv-row">
                  <CheckBadge status="pass" />
                  <div className="rv-rt">
                    <span className="rv-t">{t(m.groupSummary, { group: flatGroupLabels[g.key] ?? g.title, summary: g.summary })}</span>
                  </div>
                  {i === summaries.length - 1 ? (
                    <Button size="sm" onClick={() => setAll(true)} aria-expanded="false">
                      {t(m.showAll, { count: passes.length })}
                    </Button>
                  ) : null}
                </div>
              ))}
        </>
      ) : null}
    </>
  )
}

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
      const from = share.priceLabel === 'self-hosted' ? share.short : t(m.fromApi, { name: share.short })
      // The short finding is the server's, in English.
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
