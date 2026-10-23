import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { errorMessage, http } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { clock, count, dayMonthDate, longDate, shortDate, timeOfDay, until, zonedParts } from '@/lib/format'
import { cn } from '@/lib/cn'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { FeedItemRow } from './FeedItemRow'
import { m } from './messages'
import { dateTime, IssueStatusBadge, useRefreshDigest, type Digest, type DigestNextIssue } from './shared'

/** The clauses of "Every issue matches 2 of your approval rules: …". */
export function ruleClauses(t: ReturnType<typeof useT>, d: Digest): string[] {
  return d.approvals.rules.map((r) => {
    if (r.key === 'audience_min') return t(m.ruleAudience, { count: count(r.threshold ?? 10000) })
    if (r.key === 'warming_provider') return r.until ? t(m.ruleWarming, { date: shortDate(r.until) }) : t(m.ruleWarmingOpen)
    return t(m.ruleOther, { label: r.short || r.label })
  })
}

/** "Prepared at 05:00": the preparation time of an issue. */
function prepareTime(n: DigestNextIssue): string {
  return n.prepareAt ? clock(n.prepareAt) : ''
}

/**
 * The digest overview's "Next issue" card, in each state the DigestStates
 * board draws: collecting, waiting for approval, held, feed failing at
 * preparation, draft ready and sending. A paused or off digest shows its
 * pause card instead (PausedCard).
 */
export function NextIssueCard({ d, onSkip }: { d: Digest; onSkip: (issueId: string, number: number) => void }) {
  const n = d.next
  if (!n) return null
  switch (n.status) {
    case 'waiting':
      return <WaitingCard d={d} n={n} onSkip={onSkip} />
    case 'held':
      return <HeldCard d={d} n={n} onSkip={onSkip} />
    case 'preparing':
      return <FeedFailingCard d={d} n={n} onSkip={onSkip} />
    case 'draft':
      return <DraftCard d={d} n={n} />
  }
  return <CollectingCard d={d} n={n} onSkip={onSkip} />
}

function Shell({ tone, title, badge, sub, children, label }: { tone?: 'warn' | 'bad' | 'info'; title: ReactNode; badge: ReactNode; sub?: ReactNode; children: ReactNode; label: string }) {
  return (
    <section
      className={cn('card flex flex-col', tone === 'warn' && 'border-warn-line', tone === 'bad' && 'border-bad-line', tone === 'info' && 'border-info-line')}
      aria-label={label}
    >
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{title}</h2>
          {sub ? <span className="k">{sub}</span> : null}
        </div>
        <div className="flex items-center gap-2">{badge}</div>
      </div>
      <div className="pad flex flex-col gap-4">{children}</div>
    </section>
  )
}

function Stat({ label, value, sub }: { label: string; value: ReactNode; sub?: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="k">{label}</span>
      <span className="n text-[20px] font-bold leading-tight">{value}</span>
      {sub ? <span className="k">{sub}</span> : null}
    </div>
  )
}

function SkipButton({ n, onSkip }: { n: DigestNextIssue; onSkip: (issueId: string, number: number) => void }) {
  const t = useT()
  const reason = usePermissionReason('campaigns:pause')
  return (
    <Button disabledReason={reason} onClick={() => onSkip(n.id, n.number)}>
      {t(m.skipIssue)}
    </Button>
  )
}

/** Collecting: the posts so far, the subject as it stands, and what the approval rules will ask. */
function CollectingCard({ d, n, onSkip }: { d: Digest; n: DigestNextIssue; onSkip: (issueId: string, number: number) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const tz = d.schedule.timezone
  const clauses = ruleClauses(t, d)
  const approvers = formatList(d.approvals.approvers, 'disjunction')
  const prepared = prepareTime(n)
  const firstEver = d.sentIssues === 0 && n.posts === 0 && d.firstOnAt
  return (
    <Shell
      label={t(m.nextIssue)}
      title={t(m.nextIssue)}
      badge={
        <>
          <IssueStatusBadge status="collecting" />
          <span className="k">{t(m.postsSoFar, { count: n.posts })}</span>
        </>
      }
    >
      <div className="flex flex-col gap-1">
        <h3 className="text-[20px] font-bold leading-tight">{t(m.nextIssueTitle, { name: d.name, number: n.number })}</h3>
        <span className="k">
          {n.sendAt
            ? t(m.nextIssueWhen, { date: longDate(n.sendAt), time: clock(n.sendAt), timezone: tz })
            : t(m.nextIssueWhenItems, { count: d.schedule.count ?? 3, from: timeOfDay(d.schedule.windowFrom || '08:00'), until: timeOfDay(d.schedule.windowUntil || '18:00'), timezone: tz })}
        </span>
      </div>
      <div className="grid grid-cols-4 gap-4 max-md:grid-cols-2">
        <Stat
          label={t(m.sendTime)}
          value={n.sendAt ? t(m.sendTimeValue, { day: String(zonedParts(n.sendAt).weekday), time: clock(n.sendAt) }) : '–'}
          sub={n.sendAt ? t(m.sendTimeIn, { date: dayMonthDate(n.sendAt), until: until(n.sendAt) }) : undefined}
        />
        <Stat label={t(m.postsLabel)} value={count(n.posts)} sub={t(m.postsFit, { count: n.fit })} />
        <Stat label={t(m.recipientsLabel)} value={count(n.recipients)} sub={t(m.recipientsLive)} />
        <Stat label={t(m.preparedLabel)} value={prepared || '–'} sub={prepared ? t(m.preparedBefore, { hours: d.prepareHours }) : undefined} />
      </div>
      {firstEver ? (
        <>
          <p className="m-0 text-[13px]">{t(m.justOnBody)}</p>
          <KeyValues
            items={[
              [t(m.turnedOnLabel), t(m.turnedOnValue, { date: shortDate(d.firstOnAt!), time: clock(d.firstOnAt!), name: d.owner?.name ?? '' })],
              [t(m.startingPoint), d.filters.startMode === 'latest' ? t(m.startLatest, { count: d.filters.startLatest }) : t(m.startAfterOn)],
              [t(m.feedLabel), t(m.feedReadValue, { source: d.site?.host ?? d.feed.url, time: d.feed.lastReadAt ? clock(d.feed.lastReadAt) : '–' })],
            ]}
          />
        </>
      ) : (
        <>
          <div className="flex flex-col gap-1.5">
            <span className="k">{t(m.subjectAsItStands)}</span>
            <div className="rounded-md border border-line bg-ground px-3 py-2.5 font-semibold">{n.posts > 0 ? n.subject : t(m.subjectNoPosts)}</div>
          </div>
          <div className="flex flex-col">
            <span className="k">{t(m.postsInIssue)}</span>
            {n.items.length ? (
              n.items.map((it, i) => <FeedItemRow key={it.id} item={it} withTime={i === 0} showCategory />)
            ) : (
              <p className="k m-0 py-2">{t(m.noPostsYet, { time: prepared || '–' })}</p>
            )}
          </div>
        </>
      )}
      {d.approvals.needed ? (
        <Note tone="info">
          <div className="flex flex-col gap-0.5">
            <span className="font-semibold">{t(m.needsApproval)}</span>
            <span>
              {d.owner
                ? t(m.needsApprovalBody, { count: clauses.length, rules: formatList(clauses), approvers, time: prepared || '–', owner: d.owner.name })
                : t(m.needsApprovalNoOwner, { count: clauses.length, rules: formatList(clauses), approvers, time: prepared || '–' })}
            </span>
          </div>
        </Note>
      ) : null}
      <div className="flex flex-wrap items-center gap-3">
        <SkipButton n={n} onSkip={onSkip} />
        <Link to="/w/$workspace/settings/approvals" params={{ workspace: slug }} className="lnk text-[13px] font-semibold">
          {t(m.approvalRules)}
        </Link>
      </div>
    </Shell>
  )
}

/** Waiting for approval: the request in the owner's name, and the late limit. */
function WaitingCard({ d, n, onSkip }: { d: Digest; n: DigestNextIssue; onSkip: (issueId: string, number: number) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [busy, setBusy] = useState(false)
  const approvers = formatList(d.approvals.approvers, 'disjunction')
  const rules = d.approvals.rules.map((r) => r.short || r.label).join(' · ')
  const remind = async () => {
    if (!n.approval) return
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/remind', { params: { path: { workspace: slug, approvalId: n.approval.id } } })
      toast({ kind: 'success', title: t(m.reminded) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Shell
      tone="warn"
      label={t(m.nextIssueNumber, { number: n.number })}
      title={t(m.nextIssueNumber, { number: n.number })}
      sub={n.sendAt ? t(m.waitingSub, { date: shortDate(n.sendAt), time: clock(n.sendAt), timezone: d.schedule.timezone, count: n.posts, recipients: count(n.recipients) }) : undefined}
      badge={<IssueStatusBadge status="waiting" />}
    >
      <Note tone="warn">
        <div className="flex flex-col gap-0.5">
          <span className="font-semibold">{t(m.waitingTitle)}</span>
          <span>
            {t(m.waitingBody, {
              prepared: n.preparedAt ? clock(n.preparedAt) : '–',
              approvers,
              send: n.sendAt ? clock(n.sendAt) : '–',
              deadline: n.deadline ? dateTime(n.deadline) : '–',
            })}
          </span>
        </div>
      </Note>
      <KeyValues
        items={[
          [
            t(m.readinessLabel),
            n.readiness ? (
              <span className="flex flex-wrap items-center gap-2">
                <Badge tone={n.readiness.verdict === 'blocked' ? 'bad' : n.readiness.verdict === 'warnings' ? 'warn' : 'ok'}>{t(m.verdictWord, { verdict: n.readiness.verdict })}</Badge>
                <span className="k">{t(m.readinessValue, { passed: n.readiness.passed, time: clock(n.readiness.ranAt) })}</span>
              </span>
            ) : (
              '–'
            ),
          ],
          [t(m.rulesMatched), rules || '–'],
          [t(m.requestedLabel), n.approval ? t(m.requestedValue, { time: clock(n.approval.requestedAt), owner: d.owner?.name ?? '' }) : '–'],
        ]}
      />
      <div className="flex flex-wrap gap-2">
        {n.approval ? (
          <ButtonLink variant="primary" to="/w/$workspace/approvals/$approvalId" params={{ workspace: slug, approvalId: n.approval.id }}>
            {t(m.openRequest)}
          </ButtonLink>
        ) : null}
        <Button loading={busy} onClick={() => void remind()} disabled={!n.approval}>
          {t(m.remindApprovers)}
        </Button>
        <SkipButton n={n} onSkip={onSkip} />
      </div>
    </Shell>
  )
}

/** Held: what blocks it, the fixes in Review & send, and Release once fixed. */
function HeldCard({ d, n, onSkip }: { d: Digest; n: DigestNextIssue; onSkip: (issueId: string, number: number) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const releaseReason = usePermissionReason('campaigns:send', d.sendHelpers)
  const [busy, setBusy] = useState<'release' | 'rebuild' | null>(null)
  const block = n.readiness?.blocking[0]
  const blocked = n.holdReason === 'readiness' && n.readiness?.verdict === 'blocked'
  const act = async (what: 'release' | 'rebuild') => {
    setBusy(what)
    try {
      const params = { path: { workspace: slug, digestId: d.id, issueId: n.id } }
      if (what === 'release') await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/issues/{issueId}/release', { params })
      else await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/issues/{issueId}/rebuild', { params })
      refresh()
      toast({ kind: 'success', title: what === 'release' ? t(m.released) : t(m.rebuilt) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(null)
    }
  }
  const title =
    n.holdReason === 'changes' ? t(m.heldChanges) : n.holdReason === 'rejected' ? t(m.heldRejected) : n.holdReason === 'no_approvers' ? t(m.heldNoApprovers) : t(m.heldTitle)
  const body =
    n.holdReason === 'changes'
      ? t(m.heldChangesBody)
      : n.holdReason === 'rejected'
        ? t(m.heldRejectedBody)
        : n.holdReason === 'no_approvers'
          ? t(m.heldNoApproversBody)
          : t(m.heldBody, { block: block ?? t(m.heldTitle) })
  return (
    <Shell
      tone="bad"
      label={t(m.nextIssueNumber, { number: n.number })}
      title={t(m.nextIssueNumber, { number: n.number })}
      sub={
        n.sendAt && n.heldAt
          ? t(n.approval ? m.heldSub : m.heldSubNoRequest, { date: shortDate(n.sendAt), time: clock(n.sendAt), held: clock(n.heldAt), owner: d.owner?.name ?? '' })
          : undefined
      }
      badge={<IssueStatusBadge status="held" />}
    >
      <Note tone="bad">
        <div className="flex flex-col gap-0.5">
          <span className="font-semibold">{title}</span>
          <span>{body}</span>
        </div>
      </Note>
      <div className="flex flex-col gap-1.5">
        <span className="lbl-t text-[12px] font-semibold text-ink-2">{t(m.fixes)}</span>
        <div className="flex flex-wrap gap-2">
          {n.campaignId ? (
            <ButtonLink variant="primary" size="sm" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId: n.campaignId }}>
              {t(m.openChecks)}
            </ButtonLink>
          ) : null}
          <Button size="sm" loading={busy === 'rebuild'} onClick={() => void act('rebuild')}>
            {t(m.rebuild)}
          </Button>
          <SkipButton n={n} onSkip={onSkip} />
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" disabledReason={releaseReason} disabled={blocked} loading={busy === 'release'} onClick={() => void act('release')} aria-describedby="release-why">
          {t(m.releaseIssue)}
        </Button>
        <span className="k" id="release-why">
          {t(m.releaseHint)}
        </span>
      </div>
    </Shell>
  )
}

/** The feed failed at preparation: the tries so far, the next one, and the send time it has until. */
function FeedFailingCard({ d, n, onSkip }: { d: Digest; n: DigestNextIssue; onSkip: (issueId: string, number: number) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const [busy, setBusy] = useState(false)
  const tries = n.feedTries
  const last = tries[tries.length - 1]
  const error = d.feed.error?.detail ?? last?.error ?? ''
  const tryNow = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/read', { params: { path: { workspace: slug, digestId: d.id } } })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Shell
      tone="bad"
      label={t(m.nextIssueNumber, { number: n.number })}
      title={t(m.nextIssueNumber, { number: n.number })}
      sub={n.sendAt && tries[0] ? t(m.feedFailingSub, { date: shortDate(n.sendAt), time: clock(n.sendAt), since: clock(tries[0].at), owner: d.owner?.name ?? '' }) : undefined}
      badge={<IssueStatusBadge status="preparing" />}
    >
      <Note tone="bad">
        <div className="flex flex-col gap-0.5">
          <span className="font-semibold">{t(m.retryingFeed)}</span>
          <span>
            {t(m.retryingBody, {
              error: /[.!?]$/.test(error) ? error : `${error}.`,
              tries: formatList(tries.map((x) => clock(x.at))),
              next: n.nextTry ? clock(n.nextTry) : '–',
              send: n.sendAt ? clock(n.sendAt) : '–',
              number: n.number,
            })}
          </span>
        </div>
      </Note>
      <div className="flex flex-wrap gap-1.5">
        {tries.map((x) => (
          <Badge key={x.at} tone="bad">
            {t(m.tryFailed, { time: clock(x.at), kind: x.error.includes('timed out') ? 'timeout' : x.kind })}
          </Badge>
        ))}
        {n.nextTry ? <Badge tone="neutral">{t(m.nextTry, { time: clock(n.nextTry) })}</Badge> : null}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button icon="refresh" loading={busy} onClick={() => void tryNow()}>
          {t(m.tryNow)}
        </Button>
        <SkipButton n={n} onSkip={onSkip} />
      </div>
    </Shell>
  )
}

/** Draft mode: the issue is a draft someone sends through Review & send. */
function DraftCard({ d, n }: { d: Digest; n: DigestNextIssue }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Shell
      tone="info"
      label={t(m.nextIssueNumber, { number: n.number })}
      title={t(m.nextIssueNumber, { number: n.number })}
      sub={n.sendAt && n.preparedAt ? t(m.draftReadySub, { date: shortDate(n.sendAt), time: clock(n.sendAt), prepared: clock(n.preparedAt), count: n.posts }) : undefined}
      badge={
        <>
          <IssueStatusBadge status="draft" />
          <span className="k">{t(m.draftNotSent)}</span>
        </>
      }
    >
      <Note tone="info">
        <div className="flex flex-col gap-0.5">
          <span className="font-semibold">{t(m.draftTitle)}</span>
          <span>{t(m.draftBody, { number: n.number, owner: d.owner?.name ?? '', deadline: n.deadline ? dateTime(n.deadline) : '–' })}</span>
        </div>
      </Note>
      {n.campaignId ? (
        <div className="flex flex-wrap gap-2">
          <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId: n.campaignId }}>
            {t(m.reviewAndSend)}
          </ButtonLink>
          <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: n.campaignId }}>
            {t(m.openDraft)}
          </ButtonLink>
        </div>
      ) : null}
    </Shell>
  )
}

/** A paused digest: why, by whom, and who can turn it back on. */
export function PausedCard({ d, onTurnOn }: { d: Digest; onTurnOn: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const p = d.pause
  const reason = usePermissionReason('campaigns:edit', d.editHelpers)
  if (!p) return null
  const tone = p.reason === 'feed' ? 'bad' : 'warn'
  let title: string
  let body: string
  switch (p.reason) {
    case 'owner':
      title = t(m.pausedAuto)
      body = p.left
        ? t(m.pausedLeftBody, { owner: d.owner?.name ?? '' })
        : p.from && p.to
          ? t(m.pausedRoleBody, { owner: d.owner?.name ?? '', from: p.from, to: p.to, date: p.at ? shortDate(p.at) : '', permission: d.mode === 'draft' ? 'campaigns:edit' : 'campaigns:send' })
          : t(m.pausedLostBody, { owner: d.owner?.name ?? '', permission: d.mode === 'draft' ? 'campaigns:edit' : 'campaigns:send' })
      break
    case 'feed':
      title = t(m.pausedFeed)
      body = t(m.pausedFeedBody, { skips: formatList(p.skips.map((s) => t(m.pausedFeedSkip, { date: shortDate(s.sendAt), kind: s.kind }))), owner: d.owner?.name ?? '' })
      break
    default:
      title = p.at ? t(m.pausedByPerson, { name: p.by ?? '', date: shortDate(p.at), time: clock(p.at) }) : t(m.statePaused)
      body = p.note ? `“${p.note}”` : ''
  }
  return (
    <section className={cn('card flex flex-col', tone === 'bad' ? 'border-bad-line' : 'border-warn-line')} aria-label={t(m.statePaused)}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{d.name}</h2>
          <span className="k">{d.owner ? t(m.ownerLine, { name: d.owner.name }) : null}</span>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="warn" dot>
            {t(m.statePaused)}
          </Badge>
          {p.reason === 'feed' ? (
            <ButtonLink size="sm" to="/w/$workspace/campaigns/digests/$digestId/settings" params={{ workspace: slug, digestId: d.id }}>
              {t(m.checkFeed)}
            </ButtonLink>
          ) : null}
          <Button size="sm" variant="primary" icon={p.reason === 'owner' ? undefined : 'play'} disabledReason={reason} onClick={onTurnOn}>
            {p.reason === 'owner' ? t(m.turnOnOwner) : t(m.turnOn)}
          </Button>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <Note tone={tone}>
          <div className="flex flex-col gap-0.5">
            <span className="font-semibold">{title}</span>
            {body ? <span>{body}</span> : null}
          </div>
        </Note>
        {p.reason === 'person' ? <p className="k m-0">{t(m.pausedPersonBody)}</p> : null}
        {p.reason === 'owner' && p.notified.length ? <p className="k m-0">{t(m.pausedNotified, { names: formatList(p.notified) })}</p> : null}
      </div>
    </section>
  )
}
