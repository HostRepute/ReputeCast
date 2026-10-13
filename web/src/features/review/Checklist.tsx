import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { cn } from '@/lib/cn'
import { bySeverity, checksByGroup, flatTitle, type Check, type Fix, type Run } from './logic'
import { CheckBadge, CheckRow, FixButtons } from './parts'

type Actions = {
  busy?: string
  reasonFor: (f: Fix) => string | undefined
  onApply: (c: Check, f: Fix) => void
}

/**
 * The Send Readiness checklist. Grouped by category as on SendReadiness; when
 * the verdict is Blocked it lists blocks first, then warnings, then a
 * collapsed list of what passed (SendReadinessBlocked).
 */
export function Checklist({ run, ...actions }: { run: Run } & Actions) {
  return (
    <section className="card flex min-w-0 flex-col" aria-label="Send Readiness checklist">
      {run.verdict === 'blocked' ? <BySeverity run={run} {...actions} /> : <ByGroup run={run} {...actions} />}
    </section>
  )
}

function ByGroup({ run, ...actions }: { run: Run } & Actions) {
  return (
    <>
      {checksByGroup(run).map(({ group, checks }) => (
        <GroupRows key={group.key} title={group.title} caption={group.caption} checks={checks} {...actions} />
      ))}
    </>
  )
}

function GroupRows({ title, caption, checks, ...actions }: { title: string; caption: string; checks: Check[] } & Actions) {
  return (
    <>
      <div className="rv-grp">
        <h2 className="h2">{title}</h2>
        {caption ? <span className="k">{caption}</span> : null}
      </div>
      {checks.map((c) => (
        <CheckRow key={c.key} check={c}>
          {c.note ? <span className="k ml-auto">{c.note}</span> : null}
          <FixButtons check={c} {...actions} />
        </CheckRow>
      ))}
    </>
  )
}

function BySeverity({ run, ...actions }: { run: Run } & Actions) {
  const [all, setAll] = useState(false)
  const blocks = run.checks.filter((c) => c.status === 'block').sort(bySeverity)
  const warns = run.checks.filter((c) => c.status === 'warn').sort(bySeverity)
  const unknown = run.checks.filter((c) => c.status === 'unknown')
  const passes = run.checks.filter((c) => c.status === 'pass')
  const summaries = run.groups.filter((g) => g.passed > 0 && g.summary)
  return (
    <>
      <div className="rv-grp g-bad">
        <h2 className="h2">Blocking issues · {blocks.length}</h2>
        <span className="k">Fix these first</span>
      </div>
      {blocks.map((c) => (
        <CheckRow key={c.key} check={c} title={flatTitle(c)}>
          <FixButtons check={c} {...actions} />
        </CheckRow>
      ))}
      {warns.length ? (
        <>
          <div className="rv-grp">
            <h2 className="h2">Warnings · {warns.length}</h2>
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
            <h2 className="h2">Not checked · {unknown.length}</h2>
            <span className="k">Run the checks again</span>
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
            <h2 className="h2">Passed · {passes.length}</h2>
            <span className="k">{all ? 'Expanded' : 'Collapsed'}</span>
          </div>
          {all
            ? passes.map((c, i) => (
                <CheckRow key={c.key} check={c} title={flatTitle(c)}>
                  {i === passes.length - 1 ? (
                    <Button size="sm" onClick={() => setAll(false)} aria-expanded="true">
                      Collapse
                    </Button>
                  ) : null}
                </CheckRow>
              ))
            : summaries.map((g, i) => (
                <div key={g.key} className="rv-row">
                  <CheckBadge status="pass" />
                  <div className="rv-rt">
                    <span className="rv-t">
                      {g.title}: {g.summary}
                    </span>
                  </div>
                  {i === summaries.length - 1 ? (
                    <Button size="sm" onClick={() => setAll(true)} aria-expanded="false">
                      Show all {passes.length}
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
  const rank = (s: string) => (s === 'running' ? 1 : s === 'queued' ? 2 : 0)
  const groups = [...progress.groups].sort((a, b) => rank(a.status) - rank(b.status))
  return (
    <div className="card flex flex-col" aria-live="polite">
      {groups.map((g) => (
        <div key={g.key} className={cn('rv-line', g.status === 'queued' && 'l-queued')}>
          {g.status === 'running' ? (
            <span className="spin" aria-hidden="true" />
          ) : g.status === 'queued' ? (
            <Badge tone="neutral">Queued</Badge>
          ) : (
            <CheckBadge status={g.status} className="pop" />
          )}
          <span>
            {g.title}
            {g.text ? ` · ${g.text}` : ''}
          </span>
          {g.status === 'running' && g.checked > 0 && g.total > 0 ? (
            <span className="k ml-auto">
              {g.checked} of {g.total}
            </span>
          ) : null}
          {g.status === 'running' ? <span className="vh">running</span> : null}
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
  const groups = new Set(run.checks.filter((c) => c.status === 'unknown').map((c) => c.group))
  const rows = run.checks
    .filter((c) => groups.has(c.group))
    .sort((a, b) => (a.status === 'unknown' ? 0 : 1) - (b.status === 'unknown' ? 0 : 1))
  return (
    <div className="card flex flex-col">
      {rows.map((c) => (
        <div key={c.key} className="rv-line">
          <CheckBadge status={c.status} />
          <span>{(c.flatTitle || c.title).replace(/^([^:]+): /, '$1 · ')}</span>
          {c.status === 'unknown' && c.note ? <span className="k ml-auto">{c.note}</span> : null}
        </div>
      ))}
    </div>
  )
}
