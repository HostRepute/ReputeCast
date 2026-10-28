import { useId, useState } from 'react'
import { Button } from '@/components/ui/Button'
import { m as am } from '@/features/accessibility/messages'
import { CodeText, newGroups, Swatch } from '@/features/accessibility/review'
import { CheckRowsGrid } from '@/features/languages/review'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { bySeverity, checksByGroup, flatGroupLabels, flatTitle, groupTitles, type Check, type Fix, type Run } from './logic'
import { m } from './messages'
import { AppHref, CheckBadge, CheckRow, Chevron, TechDetails } from './parts'

type Actions = {
  busy?: string
  reasonFor: (f: Fix) => string | undefined
  onApply: (c: Check, f: Fix) => void
  /** "Run again" on a check that couldn't run (the Accessibility group's). */
  onRerun?: () => void
  /** Undo a warning's dismissal in the composer; undoReason says who can when the person can't. */
  onUndismiss?: (c: Check) => void
  undoReason?: string
}

/**
 * The issues, first (SendReadiness, SendReadinessBlocked, SR-13): blocks,
 * then warnings, then checks that couldn't run, each with its fixes, what
 * applying one changes, and the technical details behind the finding.
 * unknown=false leaves out the checks that couldn't run (the incomplete
 * state lists them above).
 */
export function NeedsAttention({ run, unknown = true, ...actions }: { run: Run; unknown?: boolean } & Actions) {
  const t = useT()
  const blocks = run.checks.filter((c) => c.status === 'block').sort(bySeverity)
  const warns = run.checks.filter((c) => c.status === 'warn').sort(bySeverity)
  const unknowns = unknown ? run.checks.filter((c) => c.status === 'unknown') : []
  if (!blocks.length && !warns.length && !unknowns.length) return null
  if (blocks.length) {
    return (
      <section className="card rv-attn" aria-labelledby="rv-attn">
        <div className="rv-cardh h-bad">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h2 className="h2" id="rv-attn">
              {t(m.blockingIssues, { count: blocks.length })}
            </h2>
            <span className="k rv-cardh-k">{t(m.fixFirst)}</span>
          </div>
        </div>
        {blocks.map((c, i) => (
          <Issue key={c.key} check={c} first={i === 0} {...actions} />
        ))}
        {warns.length ? (
          <>
            <div className="rv-grp">
              <h3 className="h2">{t(m.warningsTitle, { count: warns.length })}</h3>
            </div>
            {warns.map((c, i) => (
              <Issue key={c.key} check={c} first={i === 0} {...actions} />
            ))}
          </>
        ) : null}
        <NotChecked checks={unknowns} {...actions} />
      </section>
    )
  }
  const issues = [...warns, ...unknowns]
  return (
    <section className="card rv-attn" aria-labelledby="rv-attn">
      <div className={cn('rv-cardh', warns.length ? 'h-warn' : undefined)}>
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="rv-attn">
            {t(m.needsAttention, { count: issues.length })}
          </h2>
          {warns.length ? <span className="k rv-cardh-k">{t(m.attnWarnings, { count: warns.length })}</span> : null}
        </div>
      </div>
      {warns.map((c, i) => (
        <Issue key={c.key} check={c} first={i === 0} {...actions} />
      ))}
      {warns.length ? (
        <NotChecked checks={unknowns} {...actions} />
      ) : (
        unknowns.map((c, i) => <Issue key={c.key} check={c} first={i === 0} {...actions} />)
      )}
    </section>
  )
}

function NotChecked({ checks, ...actions }: { checks: Check[] } & Actions) {
  const t = useT()
  if (!checks.length) return null
  return (
    <>
      <div className="rv-grp">
        <h3 className="h2">{t(m.notChecked, { count: checks.length })}</h3>
        <span className="k">{t(m.runChecksAgain)}</span>
      </div>
      {checks.map((c, i) => (
        <Issue key={c.key} check={c} first={i === 0} {...actions} />
      ))}
    </>
  )
}

function Issue({ check: c, first, busy, reasonFor, onApply, onRerun, onUndismiss, undoReason }: { check: Check; first: boolean } & Actions) {
  const t = useT()
  const rerun = c.status === 'unknown' && newGroups.has(c.group)
  return (
    <div className={cn('rv-srow', first && 's-first', c.status === 'block' && 's-bad', c.status === 'warn' && 's-warn')} role={rerun ? 'alert' : undefined}>
      <CheckBadge status={c.status} />
      <span className="rv-st">
        <CodeText text={flatTitle(c)} />
        {c.detail ? <span className={cn('k block font-normal', c.mono && 'mono')}>{c.detail}</span> : null}
        {c.rows?.length ? <CheckRowsGrid rows={c.rows} /> : null}
        {c.dismissal ? <Dismissed check={c} busy={busy === `undo:${c.key}`} reason={undoReason} onUndo={onUndismiss} /> : null}
      </span>
      {c.fixes.length || (rerun && onRerun) ? (
        <div className="rv-sfixes">
          {c.fixes.map((f, i) => (
            <FixWithNote key={i} check={c} fix={f} index={i} busy={busy} reason={reasonFor(f)} onApply={onApply} />
          ))}
          {rerun && onRerun ? (
            <Button size="sm" variant="primary" onClick={onRerun}>
              {t(am.runAgain)}
            </Button>
          ) : null}
        </div>
      ) : c.note ? (
        <span className="k rv-sfixes">{c.note}</span>
      ) : null}
      {c.technical?.length || c.raw?.length ? <TechDetails check={c} /> : null}
    </div>
  )
}

/**
 * A warning someone dismissed in the composer (CO-14): who, why and their
 * note for reviewers, as the composer's Checks say it, with Undo. It stays
 * a warning here, already confirmed.
 */
function Dismissed({ check: c, busy, reason, onUndo }: { check: Check; busy?: boolean; reason?: string; onUndo?: (c: Check) => void }) {
  const t = useT()
  const d = c.dismissal!
  return (
    <span className="rv-dsm">
      <span className="rv-dsm-by">
        {t(m.dismissedBy, { name: d.by, reason: d.reasonText })}
        {d.count > 1 ? ` ${t(m.dismissedMore, { count: d.count })}` : ''}
      </span>
      {d.note ? <span className="rv-dsm-note">{t(m.dismissedNote, { note: d.note })}</span> : null}
      {onUndo ? (
        <Button size="sm" variant="ghost" aria-label={t(m.undoDismissLabel, { title: c.title })} disabledReason={reason} loading={busy} onClick={() => onUndo(c)}>
          {t(m.undoDismiss)}
        </Button>
      ) : null}
    </span>
  )
}

/** A fix's button with what applying it changes underneath ("+3 min · +$0.31"). */
function FixWithNote({ check, fix: f, index, busy, reason, onApply }: { check: Check; fix: Fix; index: number; busy?: string; reason?: string; onApply: (c: Check, f: Fix) => void }) {
  const id = `fx-${check.key}-${index}`.replace(/[^a-zA-Z0-9_-]/g, '-')
  const note = f.note ? (
    <span className="k" id={id}>
      {f.note}
    </span>
  ) : null
  return (
    <div className="rv-fix">
      {f.kind === 'apply' ? (
        <Button
          size="sm"
          aria-describedby={f.note ? id : undefined}
          disabledReason={reason}
          loading={busy === `${check.key}:${f.action}`}
          onClick={() => onApply(check, f)}
        >
          {f.action === 'accessibility_color' && f.params?.color ? <Swatch color={f.params.color} /> : null}
          {f.label}
        </Button>
      ) : (
        <AppHref href={f.href ?? '#'} className="btn sm" describedBy={f.note ? id : undefined}>
          {f.label}
        </AppHref>
      )}
      {note}
    </div>
  )
}

/**
 * Passed · N (SR-13): the passing checks collapsed, as one chip per group
 * (SendReadiness) or one summary line per group when the send is blocked
 * (SendReadinessBlocked), and every check, grouped, behind "Show all N checks".
 */
export function PassedCard({ run, caption, summaries }: { run: Run; caption?: string; summaries?: boolean }) {
  const t = useT()
  const id = useId()
  const [open, setOpen] = useState(false)
  const passes = run.checks.filter((c) => c.status === 'pass')
  if (!passes.length) return null
  const groups = run.groups.filter((g) => g.passed > 0)
  return (
    <section className="card rv-passed" aria-labelledby={`${id}-h`}>
      <div className="rv-cardh">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id={`${id}-h`}>
            {t(m.passedTitle, { count: passes.length })}
          </h2>
          {caption ? <span className="k">{caption}</span> : null}
        </div>
      </div>
      {summaries ? (
        groups
          .filter((g) => g.summary)
          .map((g, i) => (
            <div key={g.key} className={cn('rv-srow', i === 0 && 's-first')}>
              <CheckBadge status="pass" />
              <span className="rv-st font-normal">{t(m.groupSummary, { group: flatGroupLabels[g.key] ?? g.title, summary: g.summary })}</span>
            </div>
          ))
      ) : (
        <ul className="rv-pchips" aria-label={t(m.passedByGroup)}>
          {groups.map((g) => (
            <li key={g.key} className="rv-pchip">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="var(--ok-fg)" strokeWidth="2" aria-hidden="true">
                <path d="m3.5 8.5 3 3 6-7" />
              </svg>
              {flatGroupLabels[g.key] ?? g.title}
              <span className="n">{g.passed}</span>
            </li>
          ))}
        </ul>
      )}
      <details className="rv-allc" open={open} onToggle={(e) => setOpen((e.target as HTMLDetailsElement).open)}>
        <summary>
          <span>{open ? t(m.hideAllChecks, { count: passes.length }) : t(m.showAllChecks, { count: passes.length })}</span>
          <Chevron />
        </summary>
        {open
          ? checksByGroup(run)
              .map(({ group, checks }) => ({ group, checks: checks.filter((c) => c.status === 'pass') }))
              .filter(({ checks }) => checks.length > 0)
              .map(({ group, checks }) => (
                <div key={group.key}>
                  <div className="rv-grp">
                    <h3 className="h2">{t(m.groupCount, { group: groupTitles[group.key] ?? group.title, count: checks.length })}</h3>
                    {group.caption ? <span className="k">{group.caption}</span> : null}
                  </div>
                  {checks.map((c) => (
                    <CheckRow key={c.key} check={c}>
                      {c.fixes.some((f) => f.kind === 'link') ? (
                        <div className="rv-fixes">
                          {c.fixes
                            .filter((f) => f.kind === 'link')
                            .map((f, i) => (
                              <AppHref key={i} href={f.href ?? '#'} className="btn sm">
                                {f.label}
                              </AppHref>
                            ))}
                        </div>
                      ) : null}
                    </CheckRow>
                  ))}
                </div>
              ))
          : null}
      </details>
    </section>
  )
}
