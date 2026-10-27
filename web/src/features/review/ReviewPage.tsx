import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState, ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Banner } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { monthDayTime, shortDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { deniedReason, useMe, useWorkspace } from '@/lib/session'
import { approverNames, deciderFirst } from '@/features/approvals/parts'
import { m as am } from '@/features/accessibility/messages'
import { accessibilityBlock, OneClickDialog } from '@/features/accessibility/review'
import { t as tx, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { approvalMode, ApprovedCard, BypassDialog, canRestore, HeldActions, HeldCard, InvalidatedCard, RequiredCard, WaitingCard } from './ApprovalCard'
import { AudienceCard, finalCount, RouteCard, ScheduleNote, SchedulingCard, TestSendsCard, ThresholdsCard, ThrottlingCard } from './Aside'
import { Checklist, RunningList, UnknownList } from './Checklist'
import { ConfirmSend, type SendIntent } from './ConfirmSend'
import { campaignWord, confirmNote, fixAllLabel, primaryFix, scheduleSentence, sendLabel, type Check, type Fix, type Review, type Schedule, type Thresholds } from './logic'
import { m } from './messages'
import { fixReason, whenText } from './parts'
import { IncompleteBanner, RunningBanner, ScheduledBanner, ScheduledTiles, StaleBanner, VerdictBanner } from './Verdict'
import { ByVersion, EACH, languageRows, sentVersions } from '@/features/languages/review'
import { mr } from '@/features/languages/messages'
import { digestSendsIssue, IssueReviewCard } from '@/features/digests/IssueParts'
import { im } from '@/features/digests/issueMessages'
import { dateTime, useIssueInfo } from '@/features/digests/shared'
import { ReadinessVerifyDialog } from '@/features/verification/HealthEntry'
import { PlacementReviewCard } from '@/features/placement/EntryPoints'
import './review.css'
import { ReviewWebLine } from '@/features/website/elsewhere'

const reviewPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/review' as const
const progressPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/readiness/progress' as const

/** 7.5 Review & send: Send Readiness, estimates, settings, approvals and the send. */
export function ReviewPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const path = { workspace: slug, campaignId }
  const review = $api.useQuery('get', reviewPath, { params: { path } })
  const refresh = useCallback(
    () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: ['get', reviewPath] }),
        qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] }),
        qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/approvals'] }),
      ]),
    [qc],
  )
  // Runs are background jobs: asking for one answers at once, and the
  // page polls the cheap progress endpoint (not the review) until the run
  // lands or fails, then reads the review again.
  const [awaiting, setAwaiting] = useState<{ since: number }>()
  // Waiting since a run was asked for here, or since the review showed one.
  const since = awaiting?.since ?? (review.data?.running ? review.dataUpdatedAt : undefined)
  const poll = $api.useQuery('get', progressPath, { params: { path } }, { enabled: since !== undefined, refetchInterval: 1000 })
  const landed = since !== undefined && poll.data !== undefined && !poll.data.running && poll.dataUpdatedAt >= since
  const running = since !== undefined && !landed
  // Once the run lands (or fails), the review is read again, once.
  const handled = useRef<number>(undefined)
  useEffect(() => {
    if (!landed || since === undefined || handled.current === since) return
    handled.current = since
    if (poll.data?.failed) toast({ kind: 'error', title: tx(m.runFailed), detail: poll.data.failed.error })
    void refresh()
  }, [landed, since, poll.data, refresh])
  const runChecks = useCallback(async () => {
    const since = Date.now()
    setAwaiting({ since })
    try {
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/readiness', { params: { path: { workspace: slug, campaignId } } })
    } catch (e) {
      if (!(e instanceof ApiError && e.code === 'readiness_running')) {
        setAwaiting(undefined)
        toast({ kind: 'error', title: tx(m.runFailed), detail: errorMessage(e) })
      }
    }
  }, [slug, campaignId])
  const progress = poll.data?.running ?? review.data?.running
  const d = review.data
  const name = d?.campaign.name ?? t(m.campaignFallback)
  return (
    <Page
      title={t(m.pageTitle)}
      documentTitle={d ? t(m.documentTitle, { name: d.campaign.name }) : t(m.pageTitle)}
      breadcrumb={[
        { label: t(m.campaignsCrumb), to: '/w/$workspace/campaigns', params: { workspace: slug } },
        { label: name, to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId } },
        { label: t(m.pageTitle) },
      ]}
      actions={d ? <HeaderActions review={d} running={running} onRun={() => void runChecks()} /> : null}
    >
      {review.isPending ? (
        <ReviewSkeleton />
      ) : review.isError ? (
        review.error instanceof ApiError && review.error.status === 404 ? (
          <EmptyState
            title={t(m.notFoundTitle)}
            action={
              <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
                {t(m.backToCampaigns)}
              </ButtonLink>
            }
          >
            {t(m.notFoundBody)}
          </EmptyState>
        ) : (
          <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => review.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(review.error)} {t(m.nothingChanged)}
          </ErrorPanel>
        )
      ) : (
        <ReviewBody review={review.data} running={running} progress={progress} runChecks={runChecks} awaitRun={() => setAwaiting({ since: Date.now() })} refresh={refresh} />
      )}
    </Page>
  )
}

function HeaderActions({ review, running, onRun }: { review: Review; running: boolean; onRun: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const c = review.campaign
  const at = c.autosavedAt ?? c.updatedAt
  const reason = !review.access.canEdit ? deniedReason('campaigns:edit', review.access.sendHelpers) : !c.editable ? t(m.alreadySending) : undefined
  return (
    <>
      <Badge tone="neutral">{c.editable && at ? t(m.statusAutosaved, { status: campaignWord(c.status), when: relative(at, now) }) : campaignWord(c.status)}</Badge>
      {c.editable ? (
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: c.id }}>
          {t(m.backToEditor)}
        </ButtonLink>
      ) : null}
      <Button disabledReason={reason} loading={running} onClick={onRun}>
        {t(m.rerun)}
      </Button>
    </>
  )
}

/** "2 min ago", "yesterday" as the composer writes it. */
function relative(d: string, now: Date): string {
  const mins = Math.round((now.getTime() - new Date(d).getTime()) / 60_000)
  if (mins < 1) return tx(m.relJustNow)
  if (mins < 60) return tx(m.relMinutes, { n: mins })
  const h = Math.round(mins / 60)
  if (h < 24) return tx(m.relHours, { n: h })
  return Math.round(h / 24) === 1 ? tx(m.relYesterday) : tx(m.relDays, { n: Math.round(h / 24) })
}

type Busy = string | undefined

/**
 * A send that starts now goes out on the run the person confirms, so the
 * run must be recent (the server says how recent); an older one runs again.
 */
function tooOldToSendNow(d: Review): boolean {
  return Boolean(d.run && d.schedule.mode === 'now' && Date.now() - new Date(d.run.ranAt).getTime() > d.sendNowWithin * 60_000)
}

function ReviewBody({
  review: d,
  running,
  progress,
  runChecks,
  awaitRun,
  refresh,
}: {
  review: Review
  running: boolean
  progress?: Schemas['ReadinessProgress']
  runChecks: () => Promise<void>
  /** A fix or a send queued a run: the page waits for it. */
  awaitRun: () => void
  refresh: () => Promise<unknown>
}) {
  const { slug } = useWorkspace()
  const { data: me } = useMe()
  const navigate = useNavigate()
  const t = useT()
  const versions = sentVersions(languageRows(d.run?.checks))
  const campaignId = d.campaign.id
  const path = { workspace: slug, campaignId }
  const [busy, setBusy] = useState<Busy>()
  const [confirm, setConfirm] = useState<'send' | 'approved' | null>(null)
  const [bypass, setBypass] = useState(false)
  const [unschedule, setUnschedule] = useState(false)
  // "Fix 2 in one click" shows what it changes first.
  const [oneClick, setOneClick] = useState<{ check: Check; fix: Fix }>()
  // A recurring digest's issue: the digest sends it at its time, so Release issue takes Schedule's place.
  const issue = useIssueInfo(d.campaign.id)
  const digestSends = digestSendsIssue(issue)
  // "Verify N addresses first" opens Verify addresses (X14); the dialog starts the job.
  const [verifying, setVerifying] = useState(false)

  // The checks run on their own when the campaign changed since the last
  // run (or never ran): once per visit, for people who may run them.
  const auto = useRef(false)
  useEffect(() => {
    if (auto.current || running || d.running || d.failed || !d.access.canEdit || !d.campaign.editable) return
    if (!d.run || d.stale || tooOldToSendNow(d)) {
      auto.current = true
      void runChecks()
    }
  }, [d, running, runChecks])

  const fail = (title: string) => (e: unknown) => {
    toast({ kind: 'error', title, detail: errorMessage(e) })
  }

  const applyFixes = (items: { check: Check; fix: Fix }[], key: string) =>
    applyActions(
      items.map((x) => ({ action: x.fix.action!, params: x.fix.params })),
      key,
    )

  /** Applies one-click actions: the checks' fixes, "Include them" on the audience. */
  const applyActions = async (fixes: Schemas['ReadinessFixRequest'][], key: string) => {
    setBusy(key)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/readiness/fixes', {
        params: { path },
        body: { fixes },
      })
      const applied = data?.applied ?? []
      toast({
        kind: 'success',
        // The applied fixes are named by the server.
        title: applied.length === 1 ? applied[0]! : t(m.fixesApplied, { count: applied.length }),
        detail: applied.length > 1 ? applied.join(' · ') : t(m.readinessRunsAgain),
      })
      // The checks run again in the background; sending stays locked until they finish.
      if (data?.running) awaitRun()
    } catch (e) {
      fail(fixes.length === 1 ? t(m.fixFailed) : t(m.fixesFailed))(e)
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const saveSettings = async (body: Schemas['ReadinessSendSettings'], what: 'thresholds' | 'schedule'): Promise<boolean> => {
    try {
      await http.PATCH('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/send-settings', { params: { path }, body })
      await refresh()
      return true
    } catch (e) {
      fail(t(m.saveFailed, { what }))(e)
      return false
    }
  }

  const sendTest = async (to: string, language?: string): Promise<boolean> => {
    try {
      const lang = language === EACH ? { eachLanguage: true } : language ? { language } : {}
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/test', { params: { path }, body: { to: [to], ...lang } })
      toast({ kind: 'success', title: t(m.testSent, { to: data?.sent.join(', ') ?? to }), detail: data?.provider ? t(m.via, { provider: data.provider }) : undefined })
      await refresh()
      return true
    } catch (e) {
      fail(t(m.testFailed))(e)
      return false
    }
  }

  const run = d.run
  const sendOrSchedule = async (intent: SendIntent) => {
    if (!run) return
    const body = { runId: run.id, acknowledged: intent.acknowledged, confirm: intent.confirm }
    try {
      const res =
        confirm === 'approved' && d.approval.request
          ? await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/schedule', { params: { path: { workspace: slug, approvalId: d.approval.request.id } }, body })
          : await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/send', { params: { path }, body })
      await afterSend(res.data)
    } catch (e) {
      // The checks are too old for a send that starts now, or are running:
      // the server queued a run; the page shows it and the person sends after.
      if (e instanceof ApiError && (e.code === 'readiness_old' || e.code === 'readiness_running')) {
        setConfirm(null)
        awaitRun()
        toast({ kind: 'info', title: errorMessage(e) })
        return
      }
      throw e
    }
  }

  const afterSend = async (r?: Schemas['ReviewSendResult']) => {
    await refresh()
    if (r?.status === 'sending') {
      toast({ kind: 'success', title: t(m.sendingStarted), detail: t(m.sendingDetail, { recipients: r.recipients }) })
      void navigate({ to: '/w/$workspace/campaigns/$campaignId/live', params: { workspace: slug, campaignId } })
    } else if (r) {
      toast({
        kind: 'success',
        title: t(m.scheduledToast, { name: d.campaign.name }),
        detail: t(m.scheduledDetail, { recipients: r.recipients, when: scheduleSentence(d.schedule) || t(m.atScheduledTime) }),
      })
    }
  }

  const doBypass = async (input: { reason: string; password?: string; acknowledged: string[] }) => {
    if (!run) return t(m.runFirst)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/bypass', {
        params: { path },
        body: { runId: run.id, reason: input.reason, password: input.password, acknowledged: input.acknowledged },
      })
      await afterSend(data)
      return undefined
    } catch (e) {
      return errorMessage(e)
    }
  }

  const requestApproval = async (note: string) => {
    setBusy('request')
    try {
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/approval', { params: { path }, body: { note: note || undefined } })
      try {
        window.sessionStorage.removeItem(`rc-approval-note:${campaignId}`)
      } catch {
        // Storage may be unavailable; nothing to clear then.
      }
      const asked = d.approval.approvers.filter((p) => p.name !== me?.user.name)
      toast({ kind: 'success', title: t(m.approvalRequested), detail: asked.length ? t(m.approversNotified, { count: asked.length, names: approverNames(asked, 'and') }) : undefined })
    } catch (e) {
      fail(t(m.requestFailed))(e)
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const requestAction = async (kind: 'remind' | 'cancel' | 'restore', id: string) => {
    setBusy(kind)
    try {
      if (kind === 'remind') await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/remind', { params: { path: { workspace: slug, approvalId: id } } })
      if (kind === 'cancel') await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/cancel', { params: { path: { workspace: slug, approvalId: id } } })
      if (kind === 'restore') await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/restore', { params: { path: { workspace: slug, approvalId: id } } })
      toast({
        kind: 'success',
        title: kind === 'remind' ? t(m.reminded) : kind === 'cancel' ? t(m.requestCancelled) : t(m.restored, { version: d.approval.request?.version ?? '' }),
        detail: kind === 'remind' ? t(m.remindedDetail) : kind === 'cancel' ? t(m.requestCancelledDetail) : t(m.restoredDetail),
      })
    } catch (e) {
      fail(kind === 'remind' ? t(m.remindFailed) : kind === 'cancel' ? t(m.cancelFailed) : t(m.restoreFailed))(e)
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const acceptUnknown = async () => {
    if (!run) return
    setBusy('accept')
    try {
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/readiness/{runId}/accept-unknown', { params: { path: { ...path, runId: run.id } } })
      toast({ kind: 'success', title: t(m.continuing), detail: t(m.continuingDetail) })
    } catch (e) {
      fail(t(m.continueFailed))(e)
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const doUnschedule = async () => {
    await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/unschedule', { params: { path } })
    toast({ kind: 'success', title: t(m.unscheduled, { name: d.campaign.name }), detail: t(m.unscheduledDetail) })
    await refresh()
  }

  if (running || progress) {
    return (
      <div className="flex flex-col gap-3.5">
        <RunningBanner
          progress={progress}
          action={
            <Button className="rv-off min-h-10" disabledReason={t(m.waitForChecks)}>
              {d.schedule.mode === 'now' ? t(m.send) : t(m.scheduleSend)}
            </Button>
          }
        />
        {progress ? (
          <RunningList progress={progress} />
        ) : (
          <Loading label={t(m.startingChecks)} className="card flex flex-col gap-3 p-4">
            <Skeleton w="60%" />
            <Skeleton w="45%" />
            <Skeleton w="52%" />
          </Loading>
        )}
      </div>
    )
  }

  if (!run) {
    return (
      <>
        {issue ? <IssueReviewCard issue={issue} needsApproval={d.approval.required && !d.approval.valid} /> : null}
        <EmptyState
          title={t(m.neverRanTitle)}
          action={
            d.access.canEdit && d.campaign.editable ? (
              <Button variant="primary" onClick={() => void runChecks()}>
                {t(m.runChecks)}
              </Button>
            ) : null
          }
        >
          {t(m.neverRanBody)}
        </EmptyState>
      </>
    )
  }

  const c = d.campaign
  const a = d.access
  const helpers = a.sendHelpers
  const recipients = finalCount(d)
  const blocked = run.verdict === 'blocked'
  const incomplete = run.incomplete && !run.unknownAcceptedAt
  const scheduled = c.status === 'scheduled'
  const mode = approvalMode(d)
  // A scheduled send its approval no longer covers won't start: the approval card leads.
  const showScheduled = scheduled && mode === 'none'
  const can = { edit: a.canEdit, send: a.canSend, editable: c.editable }
  const reasonFor = (f: Fix) => fixReason(f, can, helpers)
  const apply = (check: Check, fix: Fix) =>
    fix.action === 'verify_addresses'
      ? setVerifying(true)
      : fix.action === 'accessibility_one_click'
        ? setOneClick({ check, fix })
        : void applyFixes([{ check, fix }], `${check.key}:${fix.action}`)
  const fixAll = (status: 'warn' | 'block') => {
    const items = run.checks.filter((x) => x.status === status).flatMap((x) => {
      const f = primaryFix(x)
      return f && f.action !== 'verify_addresses' && !reasonFor(f) ? [{ check: x, fix: f }] : []
    })
    return items
  }
  const warnFixes = fixAll('warn')
  const blockFixes = fixAll('block')
  const sendReason = !a.canSend
    ? deniedReason('campaigns:send', helpers)
    : blocked
      ? t(m.fixBlocksFirst, { count: run.counts.blocks })
      : incomplete
        ? t(m.someChecksFailed)
        : d.stale
          ? t(m.changedRunAgain)
          : undefined
  const label = sendLabel(d.schedule.mode, recipients)
  const openSend = () => setConfirm('send')

  // The top of the page: what to look at first.
  let top: ReactNode
  if (d.held) {
    const req = d.approval.request
    const people = [
      req ? t(m.heldAuthor, { name: req.requestedBy.name }) : '',
      d.approval.approvers.length ? t(m.heldApprovers, { names: approverNames(deciderFirst(d.approval.approvers, req?.decidedBy), 'and') }) : '',
    ]
      .filter(Boolean)
      .join(', ')
    top = (
      <HeldCard
        held={d.held}
        scheduledAt={c.scheduledAt}
        approved={
          req && req.status === 'approved' && req.decidedAt
            ? t(m.heldApproved, { version: req.version, name: req.decidedBy ?? '', when: monthDayTime(req.decidedAt), valid: d.approval.valid ? 'yes' : 'no' })
            : undefined
        }
        notified={people ? t(m.heldNotified, { people, time: whenText(d.held.since) }) : undefined}
      >
        <HeldActions held={d.held} busy={busy} onApply={apply} onCancel={a.canEdit ? () => setUnschedule(true) : undefined} />
      </HeldCard>
    )
  } else if (showScheduled) {
    top = (
      <div className="flex flex-col gap-3.5">
        <ScheduledBanner
          review={d}
          actions={
            <>
              <Button disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => setUnschedule(true)}>
                {t(m.unschedule)}
              </Button>
              <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/live" params={{ workspace: slug, campaignId }}>
                {t(m.openLive)}
              </ButtonLink>
            </>
          }
        />
        <ScheduledTiles review={d} />
      </div>
    )
  } else if (!c.editable) {
    const sending = c.status === 'sending' || c.status === 'paused'
    top = (
      <Banner
        tone={c.status === 'paused' || c.status === 'failed' ? 'bad' : 'info'}
        title={sending ? t(m.bannerSending, { status: c.status, name: c.name }) : t(m.bannerDone, { status: c.status, name: c.name })}
        actions={
          sending ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/live" params={{ workspace: slug, campaignId }}>
              {t(m.openLive)}
            </ButtonLink>
          ) : (
            <ButtonLink to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId }}>
              {t(m.openReport)}
            </ButtonLink>
          )
        }
      >
        {t(m.wentOutWith, { hasDate: c.startedAt ? 'yes' : 'no', date: c.startedAt ? shortDate(c.startedAt) : '' })}
      </Banner>
    )
  } else if (incomplete) {
    top = (
      <div className="flex flex-col gap-3.5">
        <IncompleteBanner
          run={run}
          actions={
            <>
              <Button disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => void runChecks()}>
                {run.checks.some((x) => x.status === 'unknown' && x.group === 'reputation') ? t(m.retryBlocklist) : t(m.retryChecks)}
              </Button>
              <Button variant="primary" loading={busy === 'accept'} disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => void acceptUnknown()}>
                {t(m.continueProvider)}
              </Button>
            </>
          }
        />
        <UnknownList run={run} />
        <span className="k">{t(m.continueNote)}</span>
      </div>
    )
  } else if (d.stale) {
    top = (
      <StaleBanner
        run={run}
        action={
          <Button variant="primary" className="rv-big" disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => void runChecks()}>
            {t(m.runChecksAgain)}
          </Button>
        }
      />
    )
  } else if (mode === 'none') {
    top = (
      <VerdictBanner
        run={run}
        mode={d.schedule.mode}
        actions={
          blocked ? (
            <>
              {accessibilityBlock(run) ? (
                // Alt text and colors are fixed in the composer, where the issues are listed.
                <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId }} search={{ panel: 'accessibility' }}>
                  {t(am.openInComposer)}
                </ButtonLink>
              ) : blockFixes.length ? (
                <Button variant="danger-primary" loading={busy === 'all'} onClick={() => void applyFixes(blockFixes, 'all')}>
                  {fixAllLabel('block', blockFixes.length)}
                </Button>
              ) : null}
              {digestSends ? null : (
                <Button variant="primary" className="rv-big rv-off" disabledReason={sendReason}>
                  {label}
                </Button>
              )}
            </>
          ) : (
            <>
              {warnFixes.length ? (
                <Button loading={busy === 'all'} onClick={() => void applyFixes(warnFixes, 'all')}>
                  {fixAllLabel('warn', warnFixes.length)}
                </Button>
              ) : null}
              {digestSends ? null : (
                <Button variant="primary" className="rv-big" disabledReason={sendReason} onClick={openSend}>
                  {label}
                </Button>
              )}
            </>
          )
        }
      />
    )
  }

  const req = d.approval.request
  const approvalCard =
    d.held || showScheduled || incomplete || d.stale || !c.editable ? null : mode === 'required' ? (
      <RequiredCard
        review={d}
        run={run}
        busy={busy === 'request'}
        onRequest={(note) => void requestApproval(note)}
        onBypass={d.approval.canBypass ? () => setBypass(true) : undefined}
        onSaveDraft={() => {
          toast({ kind: 'success', title: t(m.savedDraft, { name: c.name }), detail: t(m.savedDraftDetail) })
          void navigate({ to: '/w/$workspace/campaigns', params: { workspace: slug } })
        }}
      />
    ) : mode === 'waiting' && req ? (
      <WaitingCard
        review={d}
        run={run}
        busy={busy === 'remind' || busy === 'cancel' ? busy : undefined}
        onRemind={() => void requestAction('remind', req.id)}
        onCancel={() => void requestAction('cancel', req.id)}
        onBypass={d.approval.canBypass ? () => setBypass(true) : undefined}
      />
    ) : mode === 'approved' ? (
      <ApprovedCard review={d} run={run} busy={busy === 'test' ? 'test' : undefined} onTest={() => me?.user.email && void sendTest(me.user.email)} onSchedule={() => setConfirm('approved')} />
    ) : mode === 'invalidated' && req ? (
      <InvalidatedCard
        review={d}
        run={run}
        busy={busy === 'restore' || busy === 'request' ? busy : undefined}
        onRestore={canRestore(req.changes) && a.canEdit ? () => void requestAction('restore', req.id) : undefined}
        onRequest={() => void requestApproval('')}
      />
    ) : null

  const editReason = !a.canEdit ? deniedReason('campaigns:edit', helpers) : !c.editable ? t(m.alreadySending) : undefined
  const startAt = d.schedule.mode === 'now' ? null : (d.schedule.at ?? null)
  const showCTA = c.editable && !scheduled && !d.held && !approvalCard && !incomplete && !digestSends
  const scheduleHint =
    mode === 'waiting' ? t(m.timeRestarts) : mode === 'approved' ? t(m.timeInvalidates) : undefined

  return (
    <>
      {issue ? <IssueReviewCard issue={issue} recipients={recipients} needsApproval={d.approval.required && !d.approval.valid} /> : null}
      {d.failed ? (
        <Banner
          tone="bad"
          title={t(m.runStoppedTitle)}
          actions={
            <Button disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => void runChecks()}>
              {t(m.retryChecks)}
            </Button>
          }
        >
          {/* The error is the server's, as the run recorded it. */}
          {t(m.runStoppedBody, { error: d.failed.error })}
        </Banner>
      ) : null}
      {top}
      <div className="flex flex-wrap items-start gap-5">
        <div className="flex min-w-0 flex-[1_1_560px] flex-col gap-5">
          {approvalCard}
          {/* The website's Web line (features/website): the post this send also publishes. */}
          <ReviewWebLine campaignId={campaignId} />
          <Checklist run={run} busy={busy} reasonFor={reasonFor} onApply={apply} onRerun={a.canEdit && c.editable ? () => void runChecks() : undefined} />
          {oneClick ? (
            <OneClickDialog
              campaignId={campaignId}
              onOpenChange={(open) => (open ? undefined : setOneClick(undefined))}
              onConfirm={() => applyFixes([oneClick], `${oneClick.check.key}:${oneClick.fix.action}`)}
            />
          ) : null}
          {verifying ? (
            <ReadinessVerifyDialog
              campaignId={campaignId}
              campaignName={d.campaign.name}
              onClose={() => {
                setVerifying(false)
                void refresh()
              }}
            />
          ) : null}
        </div>
        <aside className="rv-aside" aria-label={t(m.asideLabel)}>
          <AudienceCard
            audience={run.estimates.audience}
            including={busy === 'include_unengaged'}
            onInclude={a.canEdit && c.editable ? () => void applyActions([{ action: 'include_unengaged' }], 'include_unengaged') : undefined}
          >
            <ByVersion checks={run.checks} />
          </AudienceCard>
          <RouteCard route={run.estimates.route} startAt={startAt} />
          <ThrottlingCard throttling={run.estimates.throttling} note={!blocked} />
          <ThresholdsCard
            thresholds={d.thresholds}
            scope={!blocked}
            disabledReason={editReason}
            onSave={(th: Thresholds) => saveSettings({ thresholds: th }, 'thresholds')}
          />
          {digestSends && issue?.sendAt ? (
            <ScheduleNote text={t(im.scheduleNote, { when: dateTime(issue.sendAt) })} />
          ) : blocked && !scheduled ? (
            <ScheduleNote text={t(m.availableOnceFixed)} />
          ) : scheduled ? (
            <ScheduleNote text={t(m.scheduledNote, { mode: d.schedule.mode === 'now' ? 'now' : 'at', when: scheduleSentence(d.schedule) || t(m.savedTime) })} />
          ) : (
            <SchedulingCard
              schedule={d.schedule}
              timezone={d.timezone}
              disabledReason={editReason}
              hint={scheduleHint}
              onSave={(s: Schedule) => saveSettings({ schedule: { mode: s.mode, date: s.date, time: s.time, timezone: s.timezone } }, 'schedule')}
            />
          )}
          {!blocked ? (
            <TestSendsCard
              sends={d.testSends}
              me={a.email}
              toOthers={a.canSendTestToOthers}
              disabledReason={c.editable ? undefined : t(m.wentOut)}
              onSend={sendTest}
              versions={versions}
            />
          ) : null}
          <PlacementReviewCard
            className="rv-card c-tight"
            campaignId={campaignId}
            campaignName={d.campaign.name}
            providers={run.estimates.route.shares.filter((s) => s.recipients > 0).map((s) => s.providerId)}
          />
          {showCTA ? (
            blocked ? (
              <>
                <Button variant="primary" className="rv-cta rv-off" disabledReason={sendReason}>
                  {label}
                </Button>
                <span className="k text-center">{t(m.disabledBlocks, { count: run.counts.blocks })}</span>
              </>
            ) : (
              <>
                <Button variant="primary" className="rv-cta" disabledReason={sendReason} onClick={openSend}>
                  {label}
                </Button>
                <span className="k text-center">
                  {versions.length > 1 ? t(mr.confirmNoteLanguages, { count: run.counts.warnings }) : confirmNote(run.counts.warnings)}
                </span>
              </>
            )
          ) : null}
        </aside>
      </div>
      {confirm ? (
        <ConfirmSend
          open
          onOpenChange={(v) => !v && setConfirm(null)}
          review={d}
          run={run}
          action={confirm === 'approved' ? (d.schedule.mode === 'now' ? t(m.actionSendNow) : t(m.actionApproved)) : d.schedule.mode === 'now' ? t(m.actionSendNow) : t(m.actionSchedule)}
          onConfirm={sendOrSchedule}
          onExclude={
            a.canEdit
              ? () => {
                  const x = run.checks.find((k) => k.key === 'audience.unengaged')
                  const f = x?.fixes.find((ff) => ff.action === 'exclude_sunset')
                  if (x && f) void applyFixes([{ check: x, fix: f }], `${x.key}:${f.action}`)
                }
              : undefined
          }
        />
      ) : null}
      {bypass ? <BypassDialog open onOpenChange={setBypass} review={d} run={run} onConfirm={doBypass} /> : null}
      <ConfirmDialog
        open={unschedule}
        onOpenChange={setUnschedule}
        tone="warn"
        badge={t(m.unschedule)}
        title={t(m.unscheduleTitle, { name: c.name })}
        description={d.schedule.mode === 'now' ? t(m.unscheduleNow) : t(m.unscheduleAt, { when: scheduleSentence(d.schedule) })}
        confirmLabel={t(m.unschedule)}
        busyLabel={t(m.unscheduling)}
        onConfirm={async () => {
          try {
            await doUnschedule()
          } catch (e) {
            fail(t(m.unscheduleFailed))(e)
            throw e
          }
        }}
      />
    </>
  )
}

/** Loading: the verdict banner, the checklist and the aside as grey blocks. */
function ReviewSkeleton() {
  const t = useT()
  return (
    <Loading label={t(m.loadingReview)} className="flex flex-col gap-5">
      <div className="rv-verdict v-plain">
        <Skeleton w={40} h={40} r={20} />
        <div className="rv-vtext gap-2!">
          <Skeleton w={220} h={18} />
          <Skeleton w="70%" h={12} />
        </div>
        <Skeleton w={240} h={40} r={8} />
      </div>
      <div className="flex flex-wrap items-start gap-5">
        <div className="card flex min-w-0 flex-[1_1_560px] flex-col">
          {Array.from({ length: 3 }, (_, g) => (
            <div key={g}>
              <div className="rv-grp">
                <Skeleton w={140} h={14} />
                <Skeleton w={180} h={10} />
              </div>
              {Array.from({ length: 3 }, (_, i) => (
                <div key={i} className="rv-row">
                  <Skeleton w={44} h={22} r={999} />
                  <div className="rv-rt gap-2!">
                    <Skeleton w="65%" h={12} />
                    <Skeleton w="45%" h={10} />
                  </div>
                </div>
              ))}
            </div>
          ))}
        </div>
        <div className="rv-aside">
          {[5, 4, 5].map((n, i) => (
            <div key={i} className="card rv-card">
              <Skeleton w={130} h={14} />
              {Array.from({ length: n }, (_, j) => (
                <div key={j} className="rv-sum">
                  <Skeleton w={150} h={10} />
                  <Skeleton w={50} h={10} />
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </Loading>
  )
}

