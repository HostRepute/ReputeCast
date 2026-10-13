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
import { count, shortDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { deniedReason, useMe, useWorkspace } from '@/lib/session'
import { statusWord as campaignWord } from '@/features/composer/format'
import { approverNames } from '@/features/approvals/parts'
import { approvalMode, ApprovedCard, BypassDialog, canRestore, HeldActions, HeldCard, InvalidatedCard, RequiredCard, WaitingCard } from './ApprovalCard'
import { AudienceCard, finalCount, RouteCard, ScheduleNote, SchedulingCard, TestSendsCard, ThresholdsCard, ThrottlingCard } from './Aside'
import { Checklist, RunningList, UnknownList } from './Checklist'
import { ConfirmSend, type SendIntent } from './ConfirmSend'
import { confirmNote, fixAllLabel, plural, primaryFix, scheduleSentence, sendLabel, type Check, type Fix, type Review, type Schedule, type Thresholds } from './logic'
import { fixReason, whenText } from './parts'
import { IncompleteBanner, RunningBanner, ScheduledBanner, ScheduledTiles, StaleBanner, VerdictBanner } from './Verdict'
import './review.css'

const reviewPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/review' as const

/** 7.5 Review & send: Send Readiness, estimates, settings, approvals and the send. */
export function ReviewPage({ campaignId }: { campaignId: string }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [running, setRunning] = useState(false)
  const review = $api.useQuery(
    'get',
    reviewPath,
    { params: { path: { workspace: slug, campaignId } } },
    { refetchInterval: (q) => (q.state.data?.running || running ? 800 : false) },
  )
  const refresh = useCallback(
    () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: ['get', reviewPath] }),
        qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] }),
        qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/approvals'] }),
      ]),
    [qc],
  )
  const runChecks = useCallback(async () => {
    setRunning(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/readiness', { params: { path: { workspace: slug, campaignId } } })
    } catch (e) {
      if (!(e instanceof ApiError && e.code === 'readiness_running')) {
        toast({ kind: 'error', title: "Couldn't run the checks", detail: errorMessage(e) })
      }
    } finally {
      setRunning(false)
      await refresh()
    }
  }, [slug, campaignId, refresh])
  const d = review.data
  const name = d?.campaign.name ?? 'Campaign'
  return (
    <Page
      title="Review & send"
      documentTitle={d ? `Review & send · ${d.campaign.name}` : 'Review & send'}
      breadcrumb={[
        { label: 'Campaigns', to: '/w/$workspace/campaigns', params: { workspace: slug } },
        { label: name, to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId } },
        { label: 'Review & send' },
      ]}
      actions={d ? <HeaderActions review={d} running={running || Boolean(d.running)} onRun={() => void runChecks()} /> : null}
    >
      {review.isPending ? (
        <ReviewSkeleton />
      ) : review.isError ? (
        review.error instanceof ApiError && review.error.status === 404 ? (
          <EmptyState title="This campaign doesn't exist" action={<ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>Back to campaigns</ButtonLink>}>
            It may have been deleted, or the link is from another workspace.
          </EmptyState>
        ) : (
          <ErrorPanel title="Couldn't load Review & send" actions={<Button onClick={() => review.refetch()}>Retry</Button>}>
            {errorMessage(review.error)} Nothing was changed.
          </ErrorPanel>
        )
      ) : (
        <ReviewBody review={review.data} running={running} runChecks={runChecks} refresh={refresh} />
      )}
    </Page>
  )
}

function HeaderActions({ review, running, onRun }: { review: Review; running: boolean; onRun: () => void }) {
  const { slug } = useWorkspace()
  const now = useNow()
  const c = review.campaign
  const at = c.autosavedAt ?? c.updatedAt
  const reason = !review.access.canEdit ? deniedReason('campaigns:edit', review.access.sendHelpers) : !c.editable ? 'This campaign already started sending.' : undefined
  return (
    <>
      <Badge tone="neutral">
        {campaignWord(c.status)}
        {c.editable && at ? ` · autosaved ${relative(at, now)}` : ''}
      </Badge>
      {c.editable ? (
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: c.id }}>
          Back to editor
        </ButtonLink>
      ) : null}
      <Button disabledReason={reason} loading={running} onClick={onRun}>
        Re-run checks
      </Button>
    </>
  )
}

/** "2 min ago", "yesterday" as the composer writes it. */
function relative(d: string, now: Date): string {
  const m = Math.round((now.getTime() - new Date(d).getTime()) / 60_000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m} min ago`
  const h = Math.round(m / 60)
  if (h < 24) return `${h} h ago`
  return Math.round(h / 24) === 1 ? 'yesterday' : `${Math.round(h / 24)} days ago`
}

type Busy = string | undefined

function ReviewBody({
  review: d,
  running,
  runChecks,
  refresh,
}: {
  review: Review
  running: boolean
  runChecks: () => Promise<void>
  refresh: () => Promise<unknown>
}) {
  const { slug } = useWorkspace()
  const { data: me } = useMe()
  const navigate = useNavigate()
  const campaignId = d.campaign.id
  const path = { workspace: slug, campaignId }
  const [busy, setBusy] = useState<Busy>()
  const [confirm, setConfirm] = useState<'send' | 'approved' | null>(null)
  const [bypass, setBypass] = useState(false)
  const [unschedule, setUnschedule] = useState(false)

  // The checks run on their own when the campaign changed since the last
  // run (or never ran): once per visit, for people who may run them.
  const auto = useRef(false)
  useEffect(() => {
    if (auto.current || running || d.running || !d.access.canEdit || !d.campaign.editable) return
    if (!d.run || d.stale) {
      auto.current = true
      void runChecks()
    }
  }, [d, running, runChecks])

  const fail = (title: string) => (e: unknown) => {
    toast({ kind: 'error', title, detail: errorMessage(e) })
  }

  const applyFixes = async (items: { check: Check; fix: Fix }[], key: string) => {
    setBusy(key)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/readiness/fixes', {
        params: { path },
        body: { fixes: items.map((x) => ({ action: x.fix.action!, params: x.fix.params })) },
      })
      const applied = data?.applied ?? []
      toast({
        kind: 'success',
        title: applied.length === 1 ? applied[0]! : `${applied.length} fixes applied`,
        detail: applied.length > 1 ? applied.join(' · ') : 'Send Readiness ran again.',
      })
    } catch (e) {
      fail(items.length === 1 ? "Couldn't apply the fix" : "Couldn't apply the fixes")(e)
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const saveSettings = async (body: Schemas['ReadinessSendSettings'], what: string): Promise<boolean> => {
    try {
      await http.PATCH('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/send-settings', { params: { path }, body })
      await refresh()
      return true
    } catch (e) {
      fail(`Couldn't save the ${what}`)(e)
      return false
    }
  }

  const sendTest = async (to: string): Promise<boolean> => {
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/test', { params: { path }, body: { to: [to] } })
      toast({ kind: 'success', title: `Test sent to ${data?.sent.join(', ') ?? to}`, detail: data?.provider ? `via ${data.provider}` : undefined })
      await refresh()
      return true
    } catch (e) {
      fail("Couldn't send the test")(e)
      return false
    }
  }

  const run = d.run
  const sendOrSchedule = async (intent: SendIntent) => {
    if (!run) return
    const body = { runId: run.id, acknowledged: intent.acknowledged, confirm: intent.confirm }
    const res =
      confirm === 'approved' && d.approval.request
        ? await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/schedule', { params: { path: { workspace: slug, approvalId: d.approval.request.id } }, body })
        : await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/send', { params: { path }, body })
    await afterSend(res.data)
  }

  const afterSend = async (r?: Schemas['ReviewSendResult']) => {
    await refresh()
    if (r?.status === 'sending') {
      toast({ kind: 'success', title: 'Sending started', detail: `${count(r.recipients)} recipients · follow it on the live monitor` })
      void navigate({ to: '/w/$workspace/campaigns/$campaignId/live', params: { workspace: slug, campaignId } })
    } else if (r) {
      toast({ kind: 'success', title: `${d.campaign.name} is scheduled`, detail: `${count(r.recipients)} recipients · ${scheduleSentence(d.schedule) || 'at the scheduled time'}` })
    }
  }

  const doBypass = async (input: { reason: string; password?: string; acknowledged: string[] }) => {
    if (!run) return 'Run the checks first.'
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
      toast({ kind: 'success', title: 'Approval requested', detail: asked.length ? `${approverNames(asked, 'and')} ${asked.length === 1 ? 'was' : 'were'} notified in-app and by email.` : undefined })
    } catch (e) {
      fail("Couldn't request approval")(e)
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
        title: kind === 'remind' ? 'Approvers reminded' : kind === 'cancel' ? 'Request cancelled' : `v${d.approval.request?.version ?? ''} restored`,
        detail: kind === 'remind' ? 'They get another notification and email.' : kind === 'cancel' ? 'The campaign is a draft again.' : 'The approval covers the campaign again.',
      })
    } catch (e) {
      fail(kind === 'remind' ? "Couldn't remind the approvers" : kind === 'cancel' ? "Couldn't cancel the request" : "Couldn't restore the approved version")(e)
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
      toast({ kind: 'success', title: 'Continuing with provider data only', detail: 'The skipped checks are in the audit log and run again during the send.' })
    } catch (e) {
      fail("Couldn't continue without those checks")(e)
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const doUnschedule = async () => {
    await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/unschedule', { params: { path } })
    toast({ kind: 'success', title: `${d.campaign.name} is unscheduled`, detail: 'Nothing goes out until you schedule it again.' })
    await refresh()
  }

  const progress = d.running
  if (running || progress) {
    return (
      <>
        <RunningBanner
          progress={progress}
          action={
            <Button variant="primary" className="rv-big rv-off" disabledReason="Wait for the checks to finish.">
              {d.schedule.mode === 'now' ? 'Send' : 'Schedule send'}
            </Button>
          }
        />
        {progress ? <RunningList progress={progress} /> : <Loading label="Starting the checks" className="card flex flex-col gap-3 p-4"><Skeleton w="60%" /><Skeleton w="45%" /><Skeleton w="52%" /></Loading>}
      </>
    )
  }

  if (!run) {
    return (
      <EmptyState
        title="Send Readiness hasn't run on this campaign yet"
        action={
          d.access.canEdit && d.campaign.editable ? (
            <Button variant="primary" onClick={() => void runChecks()}>
              Run the checks
            </Button>
          ) : null
        }
      >
        The checks cover sender reputation, authentication, content, links, compliance, audience quality and routing. The verdict, the estimates and the send appear here once they ran.
      </EmptyState>
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
  const can = { edit: a.canEdit, send: a.canSend, editable: c.editable }
  const reasonFor = (f: Fix) => fixReason(f, can, helpers)
  const apply = (check: Check, fix: Fix) => void applyFixes([{ check, fix }], `${check.key}:${fix.action}`)
  const fixAll = (status: 'warn' | 'block') => {
    const items = run.checks.filter((x) => x.status === status).flatMap((x) => {
      const f = primaryFix(x)
      return f && !reasonFor(f) ? [{ check: x, fix: f }] : []
    })
    return items
  }
  const warnFixes = fixAll('warn')
  const blockFixes = fixAll('block')
  const sendReason = !a.canSend
    ? deniedReason('campaigns:send', helpers)
    : blocked
      ? `Fix the blocking ${plural(run.counts.blocks, 'issue', 'issues')} first.`
      : incomplete
        ? "Some checks couldn't run. Retry them, or continue with provider data only."
        : d.stale
          ? 'The campaign changed since the checks ran. Run them again first.'
          : undefined
  const label = sendLabel(d.schedule.mode, recipients)
  const openSend = () => setConfirm('send')

  // The top of the page: what to look at first.
  let top: ReactNode
  if (d.held) {
    const req = d.approval.request
    const people = [req ? `${req.requestedBy.name} (author)` : '', d.approval.approvers.length ? `${approverNames(d.approval.approvers, 'and')} (approvers)` : '']
      .filter(Boolean)
      .join(', ')
    top = (
      <HeldCard
        held={d.held}
        scheduledAt={c.scheduledAt}
        approved={
          req && req.status === 'approved' && req.decidedAt
            ? `v${req.version} by ${req.decidedBy}, ${shortDate(req.decidedAt)} · ${d.approval.valid ? 'still valid: nothing in the snapshot changed' : 'no longer valid'}`
            : undefined
        }
        notified={people ? `${people} · in-app and email at ${whenText(d.held.since)}` : undefined}
      >
        <HeldActions held={d.held} busy={busy} onApply={apply} onCancel={a.canEdit ? () => setUnschedule(true) : undefined} />
      </HeldCard>
    )
  } else if (scheduled) {
    top = (
      <>
        <ScheduledBanner
          review={d}
          actions={
            <>
              <Button disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => setUnschedule(true)}>
                Unschedule
              </Button>
              <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/live" params={{ workspace: slug, campaignId }}>
                Open live monitor
              </ButtonLink>
            </>
          }
        />
        <ScheduledTiles review={d} />
      </>
    )
  } else if (!c.editable) {
    const sending = c.status === 'sending' || c.status === 'paused'
    top = (
      <Banner
        tone={c.status === 'paused' || c.status === 'failed' ? 'bad' : 'info'}
        title={sending ? `${c.name} is ${c.status === 'paused' ? 'paused' : 'sending'}` : `${c.name} ${c.status === 'sent' ? 'was sent' : `is ${c.status}`}`}
        actions={
          sending ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/live" params={{ workspace: slug, campaignId }}>
              Open live monitor
            </ButtonLink>
          ) : (
            <ButtonLink to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId }}>
              Open report
            </ButtonLink>
          )
        }
      >
        These are the checks it went out with{c.startedAt ? ` on ${shortDate(c.startedAt)}` : ''}. Nothing here can change any more.
      </Banner>
    )
  } else if (incomplete) {
    top = (
      <>
        <IncompleteBanner
          run={run}
          actions={
            <>
              <Button disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => void runChecks()}>
                {run.checks.some((x) => x.status === 'unknown' && x.group === 'reputation') ? 'Retry blocklist checks' : 'Retry the checks'}
              </Button>
              <Button variant="primary" loading={busy === 'accept'} disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => void acceptUnknown()}>
                Continue with provider data only
              </Button>
            </>
          }
        />
        <UnknownList run={run} />
        <span className="k">“Continue with provider data only” records the skipped checks in the campaign's audit trail and re-tries them during the send.</span>
      </>
    )
  } else if (d.stale) {
    top = (
      <StaleBanner
        run={run}
        action={
          <Button variant="primary" className="rv-big" disabledReason={a.canEdit ? undefined : deniedReason('campaigns:edit', helpers)} onClick={() => void runChecks()}>
            Run the checks again
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
              {blockFixes.length ? (
                <Button variant="danger-primary" loading={busy === 'all'} onClick={() => void applyFixes(blockFixes, 'all')}>
                  {fixAllLabel('block', blockFixes.length)}
                </Button>
              ) : null}
              <Button variant="primary" className="rv-big rv-off" disabledReason={sendReason}>
                {label}
              </Button>
            </>
          ) : (
            <>
              {warnFixes.length ? (
                <Button loading={busy === 'all'} onClick={() => void applyFixes(warnFixes, 'all')}>
                  {fixAllLabel('warn', warnFixes.length)}
                </Button>
              ) : null}
              <Button variant="primary" className="rv-big" disabledReason={sendReason} onClick={openSend}>
                {label}
              </Button>
            </>
          )
        }
      />
    )
  }

  const req = d.approval.request
  const approvalCard =
    d.held || scheduled || incomplete || d.stale || !c.editable ? null : mode === 'required' ? (
      <RequiredCard
        review={d}
        run={run}
        busy={busy === 'request'}
        onRequest={(note) => void requestApproval(note)}
        onBypass={d.approval.canBypass ? () => setBypass(true) : undefined}
        onSaveDraft={() => {
          toast({ kind: 'success', title: `${c.name} is saved as a draft`, detail: 'Request approval from Review & send when it is ready.' })
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

  const editReason = !a.canEdit ? deniedReason('campaigns:edit', helpers) : !c.editable ? 'This campaign already started sending.' : undefined
  const startAt = d.schedule.mode === 'now' ? null : (d.schedule.at ?? null)
  const showCTA = c.editable && !scheduled && !d.held && !approvalCard && !incomplete
  const scheduleHint =
    mode === 'waiting' ? 'Changing the time restarts the approval request.' : mode === 'approved' ? 'Changing the time invalidates the approval.' : undefined

  return (
    <>
      {top}
      <div className="flex flex-wrap items-start gap-5">
        <div className="flex min-w-0 flex-[1_1_560px] flex-col gap-5">
          {approvalCard}
          <Checklist run={run} busy={busy} reasonFor={reasonFor} onApply={apply} />
        </div>
        <aside className="rv-aside" aria-label="Audience, route and send settings">
          <AudienceCard audience={run.estimates.audience} />
          <RouteCard route={run.estimates.route} startAt={startAt} />
          <ThrottlingCard throttling={run.estimates.throttling} note={!blocked} />
          <ThresholdsCard
            thresholds={d.thresholds}
            scope={!blocked}
            disabledReason={editReason}
            onSave={(t: Thresholds) => saveSettings({ thresholds: t }, 'thresholds')}
          />
          {blocked && !scheduled ? (
            <ScheduleNote text="Available once the blocking issues are fixed." />
          ) : scheduled ? (
            <ScheduleNote text={`${d.schedule.mode === 'now' ? 'Sends' : 'Scheduled for'} ${scheduleSentence(d.schedule) || 'the saved time'}. Unschedule to change when it goes out.`} />
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
              disabledReason={c.editable ? undefined : 'This campaign already went out.'}
              onSend={sendTest}
            />
          ) : null}
          {showCTA ? (
            blocked ? (
              <>
                <Button variant="primary" className="rv-cta rv-off" disabledReason={sendReason}>
                  {label}
                </Button>
                <span className="k text-center">
                  Disabled: {run.counts.blocks} blocking {plural(run.counts.blocks, 'issue', 'issues')}.
                </span>
              </>
            ) : (
              <>
                <Button variant="primary" className="rv-cta" disabledReason={sendReason} onClick={openSend}>
                  {label}
                </Button>
                <span className="k text-center">{confirmNote(run.counts.warnings)}</span>
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
          action={confirm === 'approved' ? (d.schedule.mode === 'now' ? 'Send now' : 'Schedule as approved') : d.schedule.mode === 'now' ? 'Send now' : 'Schedule send'}
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
        badge="Unschedule"
        title={`Unschedule “${c.name}”?`}
        description={`It won't go out ${d.schedule.mode === 'now' ? 'now' : `on ${scheduleSentence(d.schedule)}`}. The checks, the settings and any approval stay as they are; schedule it again from here.`}
        confirmLabel="Unschedule"
        busyLabel="Unscheduling…"
        onConfirm={async () => {
          try {
            await doUnschedule()
          } catch (e) {
            fail("Couldn't unschedule the campaign")(e)
            throw e
          }
        }}
      />
    </>
  )
}

/** Loading: the verdict banner, the checklist and the aside as grey blocks. */
function ReviewSkeleton() {
  return (
    <Loading label="Loading Review & send" className="flex flex-col gap-5">
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

