import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState, Loading, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, clock, count, shortDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useCan, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, LoadError, money, ResultBadge, ResultBar, ResultLegend, Tag, tagValue, useJob, useOverview, useRefreshVerification, type Job, type Target } from './shared'
import { targetLabel, tierName, VerifyDialog } from './VerifyDialog'

/** Audience › List health › a verification job: progress, the results, and what went wrong. */
export function JobPage({ jobId }: { jobId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useJob(jobId)
  const crumbs = (label?: string) => [
    { label: t(m.jobCrumbAudience), to: '/w/$workspace/audience' as const, params: { workspace: slug } },
    { label: t(m.jobCrumbHealth), to: '/w/$workspace/audience/health' as const, params: { workspace: slug } },
    ...(label ? [{ label }] : []),
  ]
  if (q.isPending) {
    return (
      <Page title={t(m.verifyTitle)} breadcrumb={crumbs()}>
        <Loading label={t(m.loading)} className="card flex flex-col gap-3 p-[18px]">
          <Skeleton w={280} h={14} />
          <Skeleton w="100%" h={12} r={6} />
          <Skeleton w={220} h={10} />
        </Loading>
      </Page>
    )
  }
  if (q.isError) {
    const missing = q.error instanceof ApiError && q.error.status === 404
    return (
      <Page title={t(m.verifyTitle)} breadcrumb={crumbs()}>
        {missing ? (
          <EmptyState title={t(m.jobNotFound)} />
        ) : (
          <LoadError title={t(m.loadJobError)} error={q.error} body={(status) => t(m.loadJobBody, { status })} onRetry={() => void q.refetch()} />
        )}
      </Page>
    )
  }
  const j = q.data
  return (
    <Page title={title(j, t)} documentTitle={t(m.jobDocument, { label: headingLabel(j, t) })} breadcrumb={crumbs(headingLabel(j, t))}>
      <JobCard j={j} />
    </Page>
  )
}

function jobTarget(j: Job): Target | undefined {
  switch (j.kind) {
    case 'tier':
      return j.tiers.length ? { kind: 'tier', tiers: j.tiers } : undefined
    case 'list':
      return j.listId ? { kind: 'list', listId: j.listId } : undefined
    case 'import':
      return j.importId ? { kind: 'import', importId: j.importId } : undefined
  }
  return undefined
}

function sentenceLabel(j: Job, t: ReturnType<typeof useT>): string {
  const target = jobTarget(j)
  return target ? targetLabel(t, target, j.label) : j.label
}

/** "Sunset tier", "At risk and Sunset", "Product Updates": a job's name as a heading, tiers in the app's language. */
export function headingLabel(j: Job, t: ReturnType<typeof useT>): string {
  if (j.kind !== 'tier' || !j.tiers.length) return j.label
  return j.tiers.length === 1 ? t(m.tierPhrase, { tier: tierName(t, j.tiers[0]!) }) : andList(j.tiers.map((x) => tierName(t, x)))
}

function title(j: Job, t: ReturnType<typeof useT>): string {
  if (j.status === 'running' || j.status === 'queued') return t(m.jobTitleRunning, { label: sentenceLabel(j, t), name: j.serviceName })
  if (j.status === 'finished') return t(m.jobTitleDone, { label: headingLabel(j, t), name: j.serviceName })
  return t(m.jobTitleState, { label: headingLabel(j, t), name: j.serviceName })
}

const statusBadge = (j: Job, t: ReturnType<typeof useT>) =>
  j.status === 'finished' ? (
    <span className="badge ok">
      <span className="dot" aria-hidden="true" />
      {t(m.stFinished)}
    </span>
  ) : j.status === 'cancelled' ? (
    <span className="badge mut">{t(m.stCancelled)}</span>
  ) : j.status === 'stopped' ? (
    <span className="badge warn">{t(m.stStopped)}</span>
  ) : j.status === 'paused' ? (
    <span className="badge bad">
      <span className="dot" aria-hidden="true" />
      {t(m.stPaused)}
    </span>
  ) : (
    <span className="badge info">
      <span className="dot" aria-hidden="true" />
      {j.status === 'queued' ? t(m.stQueued) : t(m.stRunning)}
    </span>
  )

function JobCard({ j }: { j: Job }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshVerification()
  const overview = useOverview(useCan('integrations:view'))
  const website = overview.data?.services.find((s) => s.key === j.service)?.website
  const [confirm, setConfirm] = useState(false)
  const [again, setAgain] = useState<Target>()
  const [busy, setBusy] = useState<string>()
  const act = async (what: 'cancel' | 'resume') => {
    setBusy(what)
    const res = await http.POST(what === 'cancel' ? `${base}/jobs/{job}/cancel` : `${base}/jobs/{job}/resume`, { params: { path: { workspace: slug, job: j.id } } })
    setBusy(undefined)
    if (!res.data) throw new Error(errorMessage(res.error))
    void refresh()
    if (what === 'cancel') toast({ kind: 'success', title: t(m.jobCancelled) })
  }
  const resume = () => act('resume').catch((e: unknown) => toast({ kind: 'error', title: errorMessage(e), retry: () => void resume() }))
  const live = j.status === 'running' || j.status === 'queued'
  const left = Math.max(0, j.total - j.checked)
  const counts = { valid: j.valid, invalid: j.invalid, risky: j.risky, unknown: j.unknown }
  const barLabel = t(m.usesResults, counts)
  const startedAt = j.startedAt ?? j.createdAt
  const head = (lines: ReactNode, right?: ReactNode) => (
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div className="flex min-w-0 flex-[1_1_320px] items-start gap-3">
        {live || j.status === 'paused' ? <span className="vf-spin mt-0.5" aria-hidden="true" /> : null}
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2 m-0">{title(j, t)}</h2>
          <span className="k text-ink-2!">{lines}</span>
        </div>
      </div>
      {right}
    </div>
  )
  const cancelButton = j.canVerify ? (
    <Button variant="danger" onClick={() => setConfirm(true)}>
      {t(m.cancelJob)}
    </Button>
  ) : null
  return (
    <>
      {live ? (
        <section className="card flex flex-col gap-3 px-5 py-[18px]">
          {head(
            <>
              {t(m.jobProgress, { checked: j.checked, total: j.total })}
              {j.etaSeconds != null ? ` · ${t(m.jobEta, { minutes: Math.round(j.etaSeconds / 60) })}` : ''} · {t(m.jobStartedBy, { time: clock(startedAt), name: j.startedBy })}
            </>,
            cancelButton,
          )}
          <div className="vf-bar thin" role="progressbar" aria-valuemin={0} aria-valuemax={j.total} aria-valuenow={j.checked} aria-label={t(m.jobProgress, { checked: j.checked, total: j.total })}>
            <i style={{ flex: `${j.checked} 1 0%`, background: 'var(--accent)' }} />
            <i className="w" style={{ flex: `${left} 1 0%` }} />
          </div>
          <span className="text-[12.5px] font-semibold text-ink-nav">{t(m.resultsSoFar)}</span>
          <ResultBar {...counts} label={barLabel} />
          <ResultLegend {...counts} />
          <span className="k">{t(m.paceNote, { name: j.serviceName })}</span>
        </section>
      ) : j.status === 'paused' ? (
        <section className="card flex flex-col gap-3 px-5 py-[18px]" style={{ borderColor: 'var(--bad-line)' }}>
          {head(t(m.jobStartedOn, { time: clock(startedAt), name: j.startedBy, count: j.total }), statusBadge(j, t))}
          <div className="flex items-center gap-3">
            <div className="vf-bar thin flex-1" role="progressbar" aria-valuemin={0} aria-valuemax={j.total} aria-valuenow={j.checked} aria-label={t(m.jobProgress, { checked: j.checked, total: j.total })}>
              <i style={{ flex: `${j.checked} 1 0%`, background: 'var(--ink-3)' }} />
              <i className="w" style={{ flex: `${left} 1 0%` }} />
            </div>
            <span className="k nw">{t(m.jobProgress, { checked: j.checked, total: j.total })}</span>
          </div>
          <div className="note b" role="alert">
            <Icon name="alertCircle" size={16} className="mt-px flex-none" />
            <div className="flex min-w-0 flex-col gap-1">
              <b className="font-bold">{t(m.serviceDown)}</b>
              <span>
                {j.lastError
                  ? t(m.serviceDownBody, { error: j.lastError, until: j.retryUntil ? clock(j.retryUntil) : '' })
                  : t(m.serviceDownPlain, { name: j.serviceName, until: j.retryUntil ? clock(j.retryUntil) : '' })}
              </span>
            </div>
          </div>
          {j.canVerify ? (
            <div className="flex flex-wrap gap-2">
              <Button onClick={() => void resume()} loading={busy === 'resume'} busyLabel={t(m.resuming)}>
                {t(m.retryNow)}
              </Button>
              <Button variant="danger" onClick={() => setConfirm(true)}>
                {t(m.stopKeep, { count: j.checked })}
              </Button>
            </div>
          ) : null}
        </section>
      ) : j.status === 'stopped' ? (
        <section className="card flex flex-col gap-3 px-5 py-[18px]" style={{ borderColor: '#FCD34D' }}>
          {head(t(m.jobStartedOn, { time: clock(startedAt), name: j.startedBy, count: j.total }), statusBadge(j, t))}
          <div className="vf-bar thin" role="progressbar" aria-valuemin={0} aria-valuemax={j.total} aria-valuenow={j.checked} aria-label={t(m.jobProgress, { checked: j.checked, total: j.total })}>
            <i style={{ flex: `${j.checked} 1 0%`, background: 'var(--warn-bar)' }} />
            <i className="w" style={{ flex: `${left} 1 0%` }} />
          </div>
          <div className="note w" role="status">
            <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
            <div className="flex min-w-0 flex-col gap-1">
              <b className="font-bold">{j.stopReason === 'key_rejected' ? t(m.keyRejectedJob) : t(m.outOfCredits)}</b>
              <span>
                {j.stopReason === 'key_rejected'
                  ? t(m.keyRejectedJobBody, { name: j.serviceName, checked: j.checked, total: j.total })
                  : t(m.outOfCreditsBody, { name: j.serviceName, checked: j.checked, total: j.total })}
              </span>
            </div>
          </div>
          <ResultBar {...counts} total={j.checked} label={barLabel} />
          <ResultLegend {...counts} />
          {j.canVerify ? (
            <div className="flex flex-wrap gap-2">
              <Button variant="primary" onClick={() => void resume()} loading={busy === 'resume'} busyLabel={t(m.resuming)}>
                {t(m.resume, { left, cost: money(left * j.price) })}
              </Button>
              {j.stopReason === 'key_rejected' ? (
                <Link to="/w/$workspace/settings/integrations/verification" params={{ workspace: slug }} search={{ service: j.service as 'zerobounce' }} className="btn">
                  {t(m.openSettings)}
                </Link>
              ) : website ? (
                <ButtonAnchor href={website} iconRight="external">
                  {t(m.addCredits, { name: j.serviceName })}
                  <span className="vh">{t(m.newTab)}</span>
                </ButtonAnchor>
              ) : null}
            </div>
          ) : null}
        </section>
      ) : j.status === 'cancelled' ? (
        <section className="card flex flex-col gap-3 px-5 py-[18px]">
          {head(t(m.jobCancelledLine, { name: j.stoppedBy ?? j.startedBy, time: j.finishedAt ? clock(j.finishedAt) : '', checked: j.checked, total: j.total }), statusBadge(j, t))}
          <ResultBar {...counts} label={barLabel} />
          <ResultLegend {...counts} />
          <div className="note n" role="note">
            <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
            <div className="flex min-w-0 flex-col gap-1">
              <b className="font-bold">{t(m.cancelledTitle)}</b>
              <span>{t(m.cancelledBody, { checked: j.checked, suppressed: j.suppressed, risky: j.risky, left, name: j.serviceName, cost: money(j.cost) })}</span>
            </div>
          </div>
          {j.canVerify && left && jobTarget(j) ? (
            <div>
              <Button variant="primary" onClick={() => setAgain(jobTarget(j))}>
                {t(m.verifyOther, { left })}
              </Button>
            </div>
          ) : null}
        </section>
      ) : (
        <Finished j={j} onAgain={setAgain} />
      )}
      <ConfirmDialog
        open={confirm}
        onOpenChange={setConfirm}
        tone="warn"
        badge={t(m.cancelJob)}
        title={t(m.cancelJobTitle)}
        description={t(m.cancelJobBody, { name: j.serviceName })}
        confirmLabel={t(m.cancelJobConfirm)}
        busyLabel={t(m.cancelling)}
        cancelLabel={t(m.keepRunning)}
        confirmVariant="danger"
        onConfirm={() => act('cancel')}
      />
      {again ? <VerifyDialog open onOpenChange={(o) => !o && setAgain(undefined)} target={again} origin="list_health" from={t(m.fromListHealthList, { list: j.label, name: j.startedBy })} /> : null}
    </>
  )
}

function Finished({ j, onAgain }: { j: Job; onAgain: (t: Target) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const counts = { valid: j.valid, invalid: j.invalid, risky: j.risky, unknown: j.unknown }
  const startedAt = j.startedAt ?? j.createdAt
  const finished = j.finishedAt ?? startedAt
  const traps = j.reasons.spam_trap ?? 0
  const recheck = j.recheckFrom
  const now = useNow(60_000)
  const canRecheck = recheck ? new Date(recheck).getTime() <= now.getTime() : false
  const tier = j.kind === 'tier' && j.tiers.length === 1 ? j.tiers[0] : undefined
  const risky: [string, string, number][] = [
    [t(m.reasonCatchAll), 'verify:catch-all', j.reasons.catch_all ?? 0],
    [t(m.reasonDisposable), 'verify:disposable', j.reasons.disposable ?? 0],
    [t(m.reasonRole), 'verify:role', j.reasons.role ?? 0],
    [t(m.reasonAbuse), 'verify:abuse', j.reasons.abuse ?? 0],
  ]
  const line = j.balanceAfter != null ? m.jobFinishedBalance : m.jobFinished
  const recheckTarget: Target | undefined = j.unknown ? { kind: 'selection', filter: { check: 'unknown', ...(tier ? { tier: tier as 'sunset' } : {}) }, label: t(m.unknownTitle, { count: j.unknown }) } : undefined
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.jobTitleDone, { label: headingLabel(j, t), name: j.serviceName })}</h2>
          <span className="k">
            {t(line, { date: shortDate(finished), time: clock(finished), start: clock(startedAt), name: j.startedBy, credits: j.checked, cost: money(j.cost), balance: j.balanceAfter ?? 0 })}
          </span>
        </div>
        {statusBadge(j, t)}
      </div>
      <div className="pad flex flex-col gap-3">
        <ResultBar {...counts} label={t(m.usesResults, counts)} />
        <ResultLegend {...counts} />
        <div className="vf-rgrid">
          <div className="card vf-rcard">
            <div className="hd">
              <ResultBadge result="valid" />
              <b>{count(j.valid)}</b>
            </div>
            <span className="text-[13px] text-ink-nav">{t(m.cardValidBody)}</span>
          </div>
          <div className="card vf-rcard">
            <div className="hd">
              <ResultBadge result="invalid" />
              <b>{count(j.invalid)}</b>
            </div>
            <span className="text-[13px] text-ink-nav">
              {j.suppressed ? (
                <T msg={m.cardInvalidBody} values={{ count: j.suppressed, name: j.serviceName, invalid: j.suppressed - traps, traps, b: (c) => <b className="font-semibold">{c}</b> }} />
              ) : (
                t(m.cardInvalidBodyNone)
              )}
            </span>
            {j.suppressed ? (
              <Link to="/w/$workspace/audience/suppressions" params={{ workspace: slug }} search={{ source: `verify:${j.service}` }} className="pl-lnk font-semibold">
                {t(m.seeSuppressed, { count: j.suppressed })}
              </Link>
            ) : null}
          </div>
          <div className="card vf-rcard">
            <div className="hd">
              <ResultBadge result="risky" />
              <b>{count(j.risky)}</b>
            </div>
            {risky
              .filter(([, , n]) => n > 0)
              .map(([label, tag, n]) => (
                <div className="vf-rrow" key={tag}>
                  <span>{label}</span>
                  <span className="n">{count(n)}</span>
                  <Tag>{tag}</Tag>
                </div>
              ))}
            <span className="text-[13px] text-ink-nav">{t(m.cardRiskyBody)}</span>
            {j.risky ? (
              <Link to="/w/$workspace/audience" params={{ workspace: slug }} search={{ check: 'risky', ...(tier ? { tier: tier as 'sunset' } : {}) }} className="pl-lnk font-semibold">
                {t(m.seeRisky)}
              </Link>
            ) : null}
          </div>
          <div className="card vf-rcard">
            <div className="hd">
              <ResultBadge result="unknown" />
              <b>{count(j.unknown)}</b>
            </div>
            <span className="text-[13px] text-ink-nav">
              <T msg={m.cardUnknownBody} values={{ ...tagValue, name: j.serviceName }} />
            </span>
            {recheck && j.unknown ? <span className="k">{t(m.recheckFrom, { date: shortDate(recheck), time: clock(recheck) })}</span> : null}
            {j.unknown && recheckTarget && j.canVerify ? (
              <div>
                <Button size="sm" onClick={() => onAgain(recheckTarget)} disabledReason={canRecheck || !recheck ? undefined : t(m.checkAgainWait, { date: shortDate(recheck), time: clock(recheck) })}>
                  {t(m.checkAgain, { count: j.unknown })}
                </Button>
              </div>
            ) : null}
          </div>
        </div>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line-2 px-[18px] py-3">
        <span className="k">
          {t(m.csvNote)}
        </span>
        {j.canExport ? (
          <ButtonAnchor href={`/api/v1/workspaces/${slug}/verification/jobs/${j.id}/results.csv`} icon="download" newTab={false} download="">
            {t(m.downloadCsv)}
          </ButtonAnchor>
        ) : (
          <Button icon="download" disabledReason={t(m.cantExport)}>
            {t(m.downloadCsv)}
          </Button>
        )}
      </div>
    </section>
  )
}
