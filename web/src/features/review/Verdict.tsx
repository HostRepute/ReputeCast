import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Bar } from '@/components/ui/Data'
import { Grid } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { clock, count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { m as am } from '@/features/accessibility/messages'
import { accessibilityBlock } from '@/features/accessibility/review'
import { useT } from '@/i18n'
import { blockedLine, countsLine, recheckText, scheduleSentence, sendsIn, thresholdsLine, verdictTitle, type Review, type Run } from './logic'
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
              : `${countsLine(run.counts)}. Checks finished ${whenText(run.ranAt, now)}, ${recheckText(mode)}.`}
        </span>
      </div>
      {actions ? <div className="rv-vacts">{actions}</div> : null}
    </section>
  )
}

/** "Running Send Readiness · 19 of 31 checks done" with the progress bar. */
export function RunningBanner({ progress, action }: { progress?: Schemas['ReadinessProgress']; action?: ReactNode }) {
  const done = progress?.done ?? 0
  const total = Math.max(progress?.total ?? 0, done)
  return (
    <div className="rv-verdict v-plain" role="status" aria-live="polite">
      <span className="spin rv-spin-lg" aria-hidden="true" />
      <div className="rv-vtext gap-1.5!">
        <span className="text-[18px] font-bold">{total ? `Running Send Readiness · ${done} of ${total} checks done` : 'Running Send Readiness'}</span>
        <Bar value={total ? (done / total) * 100 : 4} height={8} label="Checks done" />
        <span className="k">Blocklist lookups across 84 sources take the longest, usually 8 to 12 seconds.</span>
      </div>
      {action}
    </div>
  )
}

/** "Couldn't finish: HostRepute didn't respond" with retry and continue. */
export function IncompleteBanner({ run, actions }: { run: Run; actions?: ReactNode }) {
  const unknown = run.counts.unknown
  const hostRepute = /hostrepute/i.test(run.problem ?? '') || run.checks.some((c) => c.status === 'unknown' && c.group === 'reputation')
  const known = `${run.counts.passed} ${run.counts.passed === 1 ? 'check' : 'checks'} passed, ${run.counts.warnings} ${run.counts.warnings === 1 ? 'warning' : 'warnings'}${run.counts.blocks ? `, ${run.counts.blocks} ${run.counts.blocks === 1 ? 'block' : 'blocks'}` : ''}`
  const what = hostRepute ? `The ${unknown} blocklist ${unknown === 1 ? 'check is' : 'checks are'} unknown` : `${unknown} ${unknown === 1 ? 'check is' : 'checks are'} unknown`
  return (
    <div className="rv-verdict v-bad" role="alert">
      <VerdictIcon kind="error" size={36} />
      <div className="rv-vtext">
        <span className="text-[18px] font-bold text-[#7F1D1D]">{hostRepute ? "Couldn't finish: HostRepute didn't respond" : "Couldn't finish every check"}</span>
        <span className="text-[13px] text-[#7F1D1D]">
          {run.problem ? `${run.problem.replace(/\.$/, '')}. ` : ''}
          {known}. {what}, so the verdict can't be Ready yet.
        </span>
      </div>
      {actions ? <div className="rv-vacts">{actions}</div> : null}
    </div>
  )
}

/** "Weekly Digest #142 is scheduled" (SendReadinessStates · success). */
export function ScheduledBanner({ review, actions }: { review: Review; actions?: ReactNode }) {
  const c = review.campaign
  const shares = review.run?.estimates.route.shares.filter((s) => s.recipients > 0) ?? []
  const via = shares.length ? ` via ${shares.map((s) => s.short).join(shares.length === 2 ? ' and ' : ', ')}` : ''
  const recipients = review.run?.estimates.audience.final ?? 0
  const at = review.schedule.at ?? c.scheduledAt
  const recheck = at ? clock(new Date(new Date(at).getTime() - 15 * 60_000)) : ''
  const when = review.schedule.mode === 'now' ? 'Starts within a minute' : scheduleSentence(review.schedule)
  return (
    <div className="rv-verdict v-ok rise" role="status">
      <VerdictIcon kind="ok" />
      <div className="rv-vtext">
        <span className="rv-vtitle">{c.name} is scheduled</span>
        <span>
          {when}, to {count(recipients)} recipients{via}. Checks run again{recheck ? ` at ${recheck}` : ' before the send'}; if anything turns to Block, the send pauses and you get an email
          {review.pauseSlack ? ' and a Slack message' : ''}.
        </span>
      </div>
      {actions ? <div className="rv-vacts">{actions}</div> : null}
    </div>
  )
}

/** The four tiles under the scheduled banner. */
export function ScheduledTiles({ review }: { review: Review }) {
  const now = useNow(60_000)
  const at = review.schedule.at ?? review.campaign.scheduledAt
  const ab = review.campaign.ab
  const winner = at && ab ? clock(new Date(new Date(at).getTime() + ab.durationHours * 3_600_000)) : ''
  const tiles: [string, string][] = [
    ['Sends in', at ? sendsIn(at, now) : 'now'],
    ab ? [`A/B ${ab.kind === 'subject' ? 'subject' : ab.kind === 'content' ? 'content' : 'send-time'} test`, `${ab.testPercent}% sample · winner at ${winner}`] : ['Recipients', count(review.run?.estimates.audience.final ?? 0)],
    ['Auto-pause', thresholdsLine(review.thresholds)],
    ['Scheduled by', review.scheduled ? `${review.scheduled.by} · ${whenText(review.scheduled.at, now)}` : '—'],
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
  const now = useNow()
  return (
    <div className="rv-verdict v-warn" role="status">
      <VerdictIcon kind="warn" />
      <div className="rv-vtext">
        <span className="rv-vtitle">The campaign changed since the checks ran</span>
        <span>
          These results are from {whenText(run.ranAt, now)} (v{run.version}). Run the checks again before sending: the result can change.
        </span>
      </div>
      {action ? <div className="rv-vacts">{action}</div> : null}
    </div>
  )
}
