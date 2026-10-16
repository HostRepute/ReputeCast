import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Bar } from '@/components/ui/Data'
import { Grid } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { clock, count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { m as am } from '@/features/accessibility/messages'
import { accessibilityBlock } from '@/features/accessibility/review'
import { formatList, useT } from '@/i18n'
import { blockedLine, countsLine, scheduleSentence, sendsIn, thresholdsLine, verdictTitle, type Review, type Run } from './logic'
import { m } from './messages'
import { VerdictIcon, whenText } from './parts'

/** Ready, Ready with warnings or Blocked, with the banner's actions. */
export function VerdictBanner({ run, mode, actions }: { run: Run; mode: Review['schedule']['mode']; actions?: ReactNode }) {
  const now = useNow()
  const t = useT()
  const blocked = run.verdict === 'blocked'
  const kind = blocked ? 'bad' : run.verdict === 'warnings' ? 'warn' : 'ok'
  // Blocked by the workspace's accessibility policy alone: the banner says so.
  const a11y = blocked ? accessibilityBlock(run) : undefined
  return (
    <section className={cn('rv-verdict rise', `v-${kind}`)} role={blocked ? 'alert' : 'status'}>
      <VerdictIcon kind={kind} />
      <div className="rv-vtext">
        <span className="rv-vtitle">{a11y ? t(am.blockedTitle, { count: a11y.count }) : verdictTitle(run.verdict)}</span>
        <span>
          {a11y
            ? t(am.blockedLine, { counts: countsLine(run.counts), note: a11y.note })
            : blocked
              ? blockedLine(run.counts, run.checks)
              : t(m.verdictLine, { counts: countsLine(run.counts), when: whenText(run.ranAt, now), mode: mode === 'now' ? 'now' : 'scheduled' })}
        </span>
      </div>
      {actions ? <div className="rv-vacts">{actions}</div> : null}
    </section>
  )
}

/** "Running Send Readiness · 19 of 31 checks done" with the progress bar. */
export function RunningBanner({ progress, action }: { progress?: Schemas['ReadinessProgress']; action?: ReactNode }) {
  const t = useT()
  const done = progress?.done ?? 0
  const total = Math.max(progress?.total ?? 0, done)
  return (
    <div className="rv-verdict v-plain" role="status" aria-live="polite">
      <span className="spin rv-spin-lg" aria-hidden="true" />
      <div className="rv-vtext gap-1.5!">
        <span className="text-[18px] font-bold">{total ? t(m.runningCount, { done, total }) : t(m.running)}</span>
        <Bar value={total ? (done / total) * 100 : 4} height={8} label={t(m.checksDone)} />
        <span className="k">{t(m.runningNote)}</span>
      </div>
      {action}
    </div>
  )
}

/** "Couldn't finish: HostRepute didn't respond" with retry and continue. */
export function IncompleteBanner({ run, actions }: { run: Run; actions?: ReactNode }) {
  const t = useT()
  const hostRepute = /hostrepute/i.test(run.problem ?? '') || run.checks.some((c) => c.status === 'unknown' && c.group === 'reputation')
  const line = t(m.incompleteLine, { ...run.counts, hostRepute: hostRepute ? 'yes' : 'no' })
  return (
    <div className="rv-verdict v-bad" role="alert">
      <VerdictIcon kind="error" size={36} />
      <div className="rv-vtext">
        <span className="text-[18px] font-bold text-[#7F1D1D]">{hostRepute ? t(m.incompleteHostRepute) : t(m.incompleteTitle)}</span>
        <span className="text-[13px] text-[#7F1D1D]">
          {/* The run's problem comes from the server, as it was recorded. */}
          {run.problem ? `${run.problem.replace(/\.$/, '')}. ` : ''}
          {line}
        </span>
      </div>
      {actions ? <div className="rv-vacts">{actions}</div> : null}
    </div>
  )
}

/** "Weekly Digest #142 is scheduled" (SendReadinessStates · success). */
export function ScheduledBanner({ review, actions }: { review: Review; actions?: ReactNode }) {
  const t = useT()
  const c = review.campaign
  const shares = review.run?.estimates.route.shares.filter((s) => s.recipients > 0) ?? []
  const recipients = review.run?.estimates.audience.final ?? 0
  const at = review.schedule.at ?? c.scheduledAt
  const recheck = at ? clock(new Date(new Date(at).getTime() - 15 * 60_000)) : ''
  const when = review.schedule.mode === 'now' ? t(m.startsWithinMinute) : scheduleSentence(review.schedule)
  return (
    <div className="rv-verdict v-ok rise" role="status">
      <VerdictIcon kind="ok" />
      <div className="rv-vtext">
        <span className="rv-vtitle">{t(m.scheduledTitle, { name: c.name })}</span>
        <span>
          {t(m.scheduledLine, {
            when,
            recipients,
            hasVia: shares.length ? 'yes' : 'no',
            via: formatList(shares.map((s) => s.short)),
            hasRecheck: recheck ? 'yes' : 'no',
            recheck,
            slack: review.pauseSlack ? 'yes' : 'no',
          })}
        </span>
      </div>
      {actions ? <div className="rv-vacts">{actions}</div> : null}
    </div>
  )
}

/** The four tiles under the scheduled banner. */
export function ScheduledTiles({ review }: { review: Review }) {
  const t = useT()
  const now = useNow(60_000)
  const at = review.schedule.at ?? review.campaign.scheduledAt
  const ab = review.campaign.ab
  const winner = at && ab ? clock(new Date(new Date(at).getTime() + ab.durationHours * 3_600_000)) : ''
  const tiles: [string, string][] = [
    [t(m.tileSendsIn), at ? sendsIn(at, now) : t(m.now)],
    ab
      ? [t(m.tileAbTest, { kind: ab.kind }), t(m.tileAbValue, { percent: ab.testPercent, time: winner })]
      : [t(m.tileRecipients), count(review.run?.estimates.audience.final ?? 0)],
    [t(m.tileAutoPause), thresholdsLine(review.thresholds)],
    [t(m.tileScheduledBy), review.scheduled ? `${review.scheduled.by} · ${whenText(review.scheduled.at, now)}` : '—'],
  ]
  return (
    <Grid min={220} gap={16}>
      {tiles.map(([k, v]) => (
        <div key={k} className="card flex flex-col gap-1 px-4.5 py-4">
          <span className="k">{k}</span>
          <span className="n text-[22px]">{v}</span>
        </div>
      ))}
    </Grid>
  )
}

/** The checks describe an older version of the campaign. */
export function StaleBanner({ run, action }: { run: Run; action?: ReactNode }) {
  const t = useT()
  const now = useNow()
  return (
    <div className="rv-verdict v-warn" role="status">
      <VerdictIcon kind="warn" />
      <div className="rv-vtext">
        <span className="rv-vtitle">{t(m.staleTitle)}</span>
        <span>{t(m.staleLine, { when: whenText(run.ranAt, now), version: run.version })}</span>
      </div>
      {action ? <div className="rv-vacts">{action}</div> : null}
    </div>
  )
}
