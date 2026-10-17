import { Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { clock, count, pct, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { SkipDialog } from './Dialogs'
import { FeedItemRow } from './FeedItemRow'
import { im } from './issueMessages'
import { m } from './messages'
import { dateTime, DigestIcon, IssueStatusBadge, signed, useRefreshDigest, type IssueInfo } from './shared'

/** "Recurring digest": the badge a digest's issue carries where campaigns are listed; it opens the digest. */
export function RecurringDigestBadge({ issue }: { issue: Pick<IssueInfo, 'digestId' | 'digestName'> }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Link
      to="/w/$workspace/campaigns/digests/$digestId"
      params={{ workspace: slug, digestId: issue.digestId }}
      className="badge mut no-underline"
      aria-label={t(m.digestBadgeLabel, { name: issue.digestName })}
    >
      <DigestIcon />
      {t(im.recurringDigest)}
    </Link>
  )
}

/**
 * The approval review of an issue (DigestStates "Approvals inbox and
 * review"): the issue's posts, and what changed since the last approved
 * issue: template, audience, route and schedule.
 */
export function IssueApprovalCard({ issue }: { issue: IssueInfo }) {
  const t = useT()
  const s = issue.since
  const changes = s ? formatList(s.changed.map((what) => t(im.change, { what }))) : ''
  const approved = s?.approvedAt ? shortDate(s.approvedAt) : ''
  return (
    <section className="card" aria-labelledby="ap-issue">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-issue">
            {t(im.postsTitle)}
          </h2>
          <span className="k">{t(im.postsCaption, { count: issue.posts.length, digest: issue.digestName })}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        {issue.posts.length ? (
          <div className="flex flex-col">
            {issue.posts.map((p) => (
              <FeedItemRow key={p.id} item={p} className="border-b border-line-2 last:border-b-0" />
            ))}
          </div>
        ) : (
          <span className="k">{t(im.noPosts)}</span>
        )}
        <div className="flex flex-col gap-0.5 rounded-lg border border-line bg-ground-3" style={{ padding: '10px 12px' }}>
          {s ? (
            <>
              <span className="font-semibold">{t(im.sinceTitle, { number: s.number })}</span>
              <span className="k">{s.changed.length ? t(im.sinceChanged, { date: approved, changes }) : t(im.sinceNothing, { date: approved })}</span>
            </>
          ) : (
            <span className="k">{t(im.sinceFirst)}</span>
          )}
        </div>
      </div>
    </section>
  )
}

/**
 * Review & send for an issue (DigestStates "Review & send"): which issue
 * it is, that the approval rules apply to each issue, and in place of
 * Schedule: Release issue when it's held, or how the digest sends it.
 */
export function IssueReviewCard({ issue, recipients, needsApproval }: { issue: IssueInfo; recipients?: number; needsApproval: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const releaseReason = usePermissionReason('campaigns:send', issue.releaseHelpers)
  const skipReason = usePermissionReason('campaigns:pause')
  const navigate = useNavigate()
  const [busy, setBusy] = useState(false)
  const [skipping, setSkipping] = useState(false)
  const held = issue.status === 'held'
  const waitingToGo = issue.mode === 'auto' && (issue.status === 'prepared' || issue.status === 'waiting')
  const draft = issue.mode === 'draft' && issue.status === 'draft'
  const sendTime = issue.sendAt ? clock(issue.sendAt) : ''
  const release = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/issues/{issueId}/release', {
        params: { path: { workspace: slug, digestId: issue.digestId, issueId: issue.issueId } },
      })
      refresh()
      toast({ kind: 'success', title: t(im.released, { number: issue.number }) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const late = issue.deadline ? <span className="k">{t(im.lateLimit, { date: dateTime(issue.deadline) })}</span> : null
  const skipButton = (
    <Button disabledReason={skipReason} onClick={() => setSkipping(true)}>
      {t(im.skip)}
    </Button>
  )
  return (
    <section className="card pad flex flex-col gap-3" aria-label={t(im.recurringIssue)}>
      <div className="flex flex-wrap items-center gap-2">
        <RecurringDigestBadge issue={issue} />
        <IssueStatusBadge status={issue.status} />
        <span className="min-w-0 font-semibold">
          {issue.preparedAt
            ? recipients
              ? t(im.issueOf, { number: issue.number, name: issue.digestName, time: clock(issue.preparedAt), recipients: count(recipients) })
              : t(im.issueOfPrepared, { number: issue.number, name: issue.digestName, time: clock(issue.preparedAt) })
            : recipients
              ? t(im.issueOfCollecting, { number: issue.number, name: issue.digestName, recipients: count(recipients) })
              : t(im.issueOfPlain, { number: issue.number, name: issue.digestName })}
        </span>
        <Link to="/w/$workspace/campaigns/digests/$digestId" params={{ workspace: slug, digestId: issue.digestId }} className="btn sm ml-auto">
          {t(im.openDigest)}
        </Link>
      </div>
      <span className="k">{t(im.rulesEach)}</span>
      {held ? (
        <div className="flex flex-col gap-2 rounded-lg border border-line bg-ground-3" style={{ padding: '12px 14px' }}>
          <span className="font-semibold">{t(im.releaseTitle)}</span>
          <span className="text-[13px]">{needsApproval ? t(im.releaseBody, { time: sendTime }) : t(im.releaseBodyNoApproval, { time: sendTime })}</span>
          <div className="flex flex-wrap gap-2">
            <Button variant="primary" loading={busy} disabledReason={releaseReason} onClick={() => void release()}>
              {t(im.release)}
            </Button>
            {skipButton}
          </div>
          {late}
        </div>
      ) : waitingToGo ? (
        <div className="flex flex-col gap-2 rounded-lg border border-line bg-ground-3" style={{ padding: '12px 14px' }}>
          <span className="font-semibold">{t(im.autoTitle)}</span>
          <span className="text-[13px]">{t(im.autoBody, { number: issue.number, time: sendTime, approval: needsApproval ? 'yes' : 'no' })}</span>
          <div className="flex flex-wrap gap-2">{skipButton}</div>
          {late}
        </div>
      ) : draft && issue.deadline ? (
        <span className="text-[13px]">{t(im.draftBody, { number: issue.number, date: dateTime(issue.deadline) })}</span>
      ) : null}
      {skipping ? (
        <SkipDialog
          digestId={issue.digestId}
          issueId={issue.issueId}
          number={issue.number}
          onOpenChange={() => setSkipping(false)}
          onSkipped={() => void navigate({ to: '/w/$workspace/campaigns/digests/$digestId', params: { workspace: slug, digestId: issue.digestId } })}
        />
      ) : null}
    </section>
  )
}

/**
 * The overview's "Next scheduled campaign" when a digest's issue goes out
 * first: "Engineering blog roundup #3 · Fri 17 Oct, 08:00 · 2 posts so far ·
 * needs approval".
 */
export function OverviewIssueCard({ issue }: { issue: Schemas['OverviewNextIssue'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug, digestId: issue.digestId }
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="ov-next-h">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="h2" id="ov-next-h">
          {t(im.nextScheduled)}
        </h2>
        <IssueStatusBadge status={issue.status} />
      </div>
      <div className="flex flex-col gap-1">
        <span className="flex flex-wrap items-center gap-2">
          <Link to="/w/$workspace/campaigns/digests/$digestId" params={params} className="text-[18px] font-semibold text-ink no-underline hover:text-ink">
            {t(im.issueName, { name: issue.digestName, number: issue.number })}
          </Link>
          <RecurringDigestBadge issue={issue} />
        </span>
        <span className="k">{t(im.nextIssueLine, { when: dateTime(issue.sendAt), posts: issue.posts, approval: issue.needsApproval ? 'yes' : 'no' })}</span>
      </div>
      <div className="flex flex-wrap gap-2.5">
        <Link to="/w/$workspace/campaigns/digests/$digestId" params={params} className="btn">
          {t(im.openDigest)}
        </Link>
      </div>
    </section>
  )
}

/** Whether the digest sends this issue itself, so Review & send shows no Schedule or Send. */
export function digestSendsIssue(issue: IssueInfo | undefined): boolean {
  return Boolean(issue && issue.mode === 'auto' && ['prepared', 'waiting', 'held'].includes(issue.status))
}

type Report = Schemas['CampaignReport']

/**
 * The report of an issue: its headline numbers compared with the previous
 * issue's, and a link to the digest.
 */
export function IssueComparison({ issue, report }: { issue: IssueInfo; report: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  const prev = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report',
    { params: { path: { workspace: slug, campaignId: issue.previousCampaignId ?? '' } } },
    { enabled: Boolean(issue.previousCampaignId), staleTime: 300_000 },
  )
  const h = report.headline
  const p = prev.data?.headline
  const metric = (label: string, value: string, delta?: string) => (
    <div className="flex min-w-[150px] flex-1 flex-col gap-0.5">
      <span className="k">{label}</span>
      <span className="n text-[17px]">{value}</span>
      {delta ? <span className="k">{delta}</span> : null}
    </div>
  )
  const vs = (delta: string) => t(im.vsIssue, { delta, number: issue.previousNumber ?? 0 })
  // Rates are points apart; a rate a report doesn't have yet compares with nothing.
  const ptsVs = (a?: number | null, b?: number | null) => (a != null && b != null ? vs(t(im.pts, { delta: signed(a - b, 1) })) : undefined)
  return (
    <section className="card pad flex flex-col gap-3" aria-labelledby="rep-issue-h">
      <div className="flex flex-wrap items-center gap-2">
        <RecurringDigestBadge issue={issue} />
        <h2 className="h2" id="rep-issue-h">
          {issue.previousNumber ? t(im.comparedTitle, { number: issue.previousNumber }) : t(im.comparedCaption, { number: issue.number, name: issue.digestName })}
        </h2>
        <span className="k">{issue.previousNumber ? t(im.comparedCaption, { number: issue.number, name: issue.digestName }) : null}</span>
        <Link to="/w/$workspace/campaigns/digests/$digestId" params={{ workspace: slug, digestId: issue.digestId }} className="btn sm ml-auto">
          {t(im.openDigest)}
        </Link>
      </div>
      {issue.previousCampaignId ? (
        <div className="flex flex-wrap gap-4">
          {metric(t(im.metricDelivered), count(h.delivered), p ? vs(signed(h.delivered - p.delivered)) : undefined)}
          {metric(t(im.metricOpens), pct(h.humanOpenRate), ptsVs(h.humanOpenRate, p?.humanOpenRate))}
          {metric(t(im.metricClicks), pct(h.clickRate), ptsVs(h.clickRate, p?.clickRate))}
          {metric(t(im.metricUnsubscribes), count(h.unsubscribes), p ? vs(signed(h.unsubscribes - p.unsubscribes)) : undefined)}
        </div>
      ) : (
        <span className="k">{t(im.comparedFirst, { name: issue.digestName })}</span>
      )}
    </section>
  )
}
