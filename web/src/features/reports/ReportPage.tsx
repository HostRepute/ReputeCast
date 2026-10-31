import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { TableScroll, Th } from '@/components/ui/Table'
import { andList, clock, count, shortDate } from '@/lib/format'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { useCampaignPolls } from '@/features/polls/report'
import { ReportBody, type ReportActions } from './ReportBody'
import { IssueComparison } from '@/features/digests/IssueParts'
import { useIssueInfo } from '@/features/digests/shared'
import { ReportSkeleton } from './ReportSkeleton'
import { copyText, ShareCard, useShares, type ShareCardHandle } from './ShareCard'
import { StillComing } from './Tiles'
import { StateAlert } from './StateAlert'
import { headerCaption, notSentLead, statusBadge, type CampaignReport } from './words'
import { useReportAI } from '@/features/ai/report'
import { ReportProductSales } from '@/features/products/report'
import './reports.css'
import { CampaignEarn } from '@/features/earn/revenue/CampaignEarn'
import { ReportWebSection } from '@/features/website/elsewhere'
import { m } from './messages'

type BounceRow = Schemas['ReportBounceRow']

/** Saves a blob as a file through a temporary link. */
function saveBlob(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

function Breadcrumb({ name }: { name: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <nav aria-label={t(m.breadcrumb)} className="flex items-center gap-2 text-[13px]">
      <Link to="/w/$workspace/campaigns" params={{ workspace: slug }} className="text-ink-3 no-underline hover:text-ink-3">
        {t(m.campaigns)}
      </Link>
      <span className="k" aria-hidden="true">
        /
      </span>
      <span className="font-semibold" aria-current="page">
        {name}
      </span>
    </nav>
  )
}

/** Empty: the campaign hasn't sent anything yet (ReportStates). */
function NotSent({ r, campaignId }: { r: CampaignReport; campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const n = r.notSent!
  return (
    <section className="card flex flex-col items-center gap-2.5 border-dashed px-7 py-9 text-center">
      <svg width="48" height="48" viewBox="0 0 48 48" aria-hidden="true">
        <rect x="6" y="12" width="36" height="26" rx="4" fill="none" stroke="#9AA3B2" strokeWidth="2" />
        <path d="M12 32V22M20 32V18M28 32V26M36 32V15" stroke="#9AA3B2" strokeWidth="2.5" strokeLinecap="round" />
      </svg>
      <h2 className="m-0 text-[17px] font-bold">{t(m.noResults, { name: r.campaign.name })}</h2>
      <span className="max-w-[520px] text-ink-2">
        {notSentLead(n)} {t(m.notSentBody)}
      </span>
      <div className="mt-1 flex flex-wrap justify-center gap-2.5">
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }}>
          {t(m.viewReadiness)}
        </ButtonLink>
        {n.lastReport ? (
          <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: n.lastReport.id }}>
            {t(m.lastReport)}
          </ButtonLink>
        ) : null}
      </div>
    </section>
  )
}

/** Error: provider events wait to be processed, so some counts are behind. */
function BacklogAlert({ b, onRefresh }: { b: Schemas['ReportIngestBacklog']; onRefresh: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const first = b.providers[0]!
  const since = clock(b.since)
  const waiting = b.providers.reduce((s, p) => s + p.waiting, 0)
  const others = b.providers.slice(1).map((p) => p.name)
  const queue = {
    feed: first.feed === 'SNS' ? 'sns' : first.feed === 'Bounce mailbox' ? 'mailbox' : 'webhook',
    name: first.name,
    since,
    n: waiting,
    waiting: count(waiting),
  }
  const sentences = [others.length ? t(m.backlogQueueOthers, { ...queue, others: andList(others) }) : t(m.backlogQueue, queue), t(m.backlogComplete, { since })]
  if (b.current.length) sentences.push(t(m.backlogCurrent, { providers: andList(b.current) }))
  return (
    <StateAlert
      title={t(m.backlogTitle, { n: b.minutes })}
      actions={
        <>
          <Button size="sm" onClick={onRefresh}>
            {t(m.refresh)}
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/providers/events" params={{ workspace: slug }}>
            {t(m.viewQueue)}
          </ButtonLink>
        </>
      }
    >
      {sentences.join(' ')}
    </StateAlert>
  )
}

/**
 * "Retry after fixing" in the Investigate dialog (ReportStates; review
 * AN-7): once the block is fixed, the issue goes again to its blocked
 * addresses still on the list, once each, as a draft that goes out from
 * Review & send. Or the retry made already, or why there's none.
 */
export function BlockedRetry({ retry, campaignId, reason, onClose }: { retry: Schemas['ReportBlockedRetry']; campaignId: string; reason?: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/retry')
  const [error, setError] = useState<string>()
  const c = retry.campaign
  let text: string
  let action: ReactNode = null
  if (c?.startedAt) {
    text = t(m.retrySent, { date: shortDate(c.startedAt), name: c.name })
    action = (
      <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: c.id }}>
        {t(m.retryReport)}
      </ButtonLink>
    )
  } else if (c) {
    text = t(m.retryDraft, { name: c.name })
    action = (
      <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId: c.id }}>
        {t(m.retryOpen)}
      </ButtonLink>
    )
  } else if (retry.unavailable === 'is_retry') {
    text = t(m.retryIsRetry, { name: retry.of?.name ?? '' })
  } else if (retry.unavailable === 'not_finished') {
    text = t(m.retryNotFinished)
  } else if (retry.unavailable === 'none_left') {
    text = t(m.retryNoneLeft)
  } else {
    const left = retry.blocked - retry.recipients
    text = [
      t(m.retryBody, { n: retry.recipients, count: count(retry.recipients) }),
      left > 0 ? t(m.retryLeftOut, { n: left, count: count(left) }) : '',
    ]
      .filter(Boolean)
      .join(' ')
    action = (
      <Button
        size="sm"
        loading={create.isPending}
        disabledReason={reason ?? (retry.recipients > 0 ? undefined : t(m.retryNobodyNow))}
        onClick={async () => {
          setError(undefined)
          try {
            const made = await create.mutateAsync({ params: { path: { workspace: slug, campaignId } } })
            await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/bounces/{type}'] })
            toast({ kind: 'success', title: tt(m.retryDrafted, { name: made.name }), detail: tt(m.retryDraftedDetail) })
            onClose()
            void navigate({ to: '/w/$workspace/campaigns/$campaignId/review', params: { workspace: slug, campaignId: made.id } })
          } catch (e) {
            setError(`${tt(m.retryFailed)}: ${errorMessage(e)}`)
          }
        }}
      >
        {t(m.retryButton, { count: count(retry.recipients) })}
      </Button>
    )
  }
  return (
    <section className="col gap-2 rounded-[10px] border border-line px-3.5 py-3" aria-labelledby="rep-retry-h">
      <h3 id="rep-retry-h" className="m-0 text-[14px] font-semibold">
        {t(m.retryTitle)}
      </h3>
      <p className="m-0 text-[13px] text-ink-2">{text}</p>
      {action ? <div className="row gap-2">{action}</div> : null}
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
    </section>
  )
}

/**
 * The Review dialog of a bounce type: what the providers said, then
 * suppress or leave them. Blocks for the sender's reputation are
 * investigated, never suppressed, and retried once fixed (review AN-7).
 */
function ReviewDialog({
  row,
  campaignId,
  onClose,
  onSuppress,
  suppressReason,
  retryReason,
}: {
  row: BounceRow
  campaignId: string
  onClose: () => void
  onSuppress: () => void
  suppressReason?: string
  /** Why the member can't draft a retry (campaigns they can't edit). */
  retryReason?: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/bounces/{type}', {
    params: { path: { workspace: slug, campaignId, type: row.type } },
  })
  const blocked = row.suppression === 'investigate'
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={`${row.label} · ${count(row.count)}`}
      description={t(blocked ? m.blockedReview : m.reviewBody)}
      size="lg"
      footer={
        blocked ? (
          <>
            <Button onClick={onClose}>{t(common.close)}</Button>
            <ButtonLink variant="primary" to="/w/$workspace/deliverability" params={{ workspace: slug }}>
              {t(m.checkDelisting)}
            </ButtonLink>
          </>
        ) : (
          <>
            <Button onClick={onClose}>{t(m.leaveThem)}</Button>
            {row.unsuppressed > 0 ? (
              <Button variant="primary" disabledReason={suppressReason} onClick={onSuppress}>
                {t(m.addToSuppression, { count: count(row.unsuppressed) })}
              </Button>
            ) : null}
          </>
        )
      }
    >
      {q.isError ? (
        <ErrorPanel title={t(m.diagnosticsFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <TableScroll label={t(m.diagnostics)}>
          <table className="tbl rep-tbl phone-wrap">
            <thead>
              <tr>
                <Th>{t(m.colProviderCode)}</Th>
                <Th>{t(m.colSaid)}</Th>
                <Th align="right">{t(m.colCount)}</Th>
              </tr>
            </thead>
            <tbody>
              {(q.data?.diagnostics ?? []).map((d, i) => (
                <tr key={i}>
                  <td className="mono text-[12px]">{[d.provider, d.type, d.code].filter(Boolean).join(' ')}</td>
                  <td className="wrap">{d.diagnostic || '—'}</td>
                  <td className="r n">{count(d.count)}</td>
                </tr>
              ))}
              {q.isPending ? (
                <tr>
                  <td colSpan={3} className="k">
                    {t(common.loading)}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </TableScroll>
      )}
      {blocked && q.data?.retry ? <BlockedRetry retry={q.data.retry} campaignId={campaignId} reason={retryReason} onClose={onClose} /> : null}
    </Dialog>
  )
}

const finalAlertPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/final-alert' as const

/** How many earlier issues the tiles compare with once the results are final (reports.CompareIssues). */
const COMPARE_ISSUES = 6

/** The banner's button: ask for the email, or stop asking. */
function FinalAlertButton({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const path = { params: { path: { workspace: slug, campaignId } } }
  const q = $api.useQuery('get', finalAlertPath, path)
  const on = $api.useMutation('put', finalAlertPath)
  const off = $api.useMutation('delete', finalAlertPath)
  const busy = on.isPending || off.isPending
  const set = async (want: boolean) => {
    try {
      if (want) {
        const res = await on.mutateAsync(path)
        qc.setQueryData(['get', finalAlertPath, path], res)
        toast({
          kind: 'success',
          title: tt(m.finalAlertOn),
          detail: res.finalAt ? tt(m.finalAlertOnDetail, { day: shortDate(res.finalAt), time: clock(res.finalAt) }) : undefined,
        })
      } else {
        // Stopping answers 204 without a body.
        await off.mutateAsync(path)
        qc.setQueryData(['get', finalAlertPath, path], { ...q.data, on: false })
        toast({ kind: 'info', title: tt(m.finalAlertOff) })
      }
      await qc.invalidateQueries({ queryKey: ['get', finalAlertPath] })
    } catch (e) {
      toast({ kind: 'error', title: tt(m.finalAlertFailed), detail: errorMessage(e) })
    }
  }
  if (q.data?.on) {
    return (
      <span className="flex flex-wrap items-center gap-2">
        <span className="k">{t(m.emailingWhenFinal)}</span>
        <Button size="sm" loading={busy} onClick={() => void set(false)}>
          {t(m.stopEmail)}
        </Button>
      </span>
    )
  }
  return (
    <Button size="sm" loading={busy || q.isPending} onClick={() => void set(true)}>
      {t(m.emailWhenFinal)}
    </Button>
  )
}

/** 7.7 Campaign report. */
export function ReportPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const path = { workspace: slug, campaignId }
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report',
    { params: { path } },
    {
      // Live sends refresh often; results still coming in now and then.
      refetchInterval: (query) => {
        const c = query.state.data?.campaign
        if (!c) return false
        if (c.status === 'sending' || c.status === 'paused') return 15_000
        return c.final ? false : 120_000
      },
    },
  )
  const r = q.data
  // A recurring digest's issue: compared with the previous issue, with a link to the digest.
  const issue = useIssueInfo(campaignId)
  const shareReason = usePermissionReason('reports:share')
  const editReason = usePermissionReason('campaigns:edit')
  const suppressReason = usePermissionReason('subscribers:edit')
  const shares = useShares(campaignId, !shareReason && Boolean(r) && !r?.notSent)
  // The email's polls (design expansion, X11).
  const polls = useCampaignPolls(campaignId, Boolean(r) && !r?.notSent)
  const [exporting, setExporting] = useState<'csv' | 'pdf'>()
  const [suppressing, setSuppressing] = useState<BounceRow>()
  const [reviewing, setReviewing] = useState<BounceRow>()
  const shareCard = useRef<ShareCardHandle>(null)
  // Summarize results and the kept summary (features/ai), on a sent campaign.
  const ai = useReportAI(campaignId, Boolean(r && !r.notSent))
  const print = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/print')
  const suppress = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/bounces/{type}/suppress')

  // A campaign that doesn't exist (or a role without reports) gets the
  // shell's 404 or 403 screen.
  if (!r && q.error instanceof ApiError && (q.error.status === 404 || q.error.status === 403)) throw q.error

  const name = r?.campaign.name ?? t(m.report)
  const sent = r && !r.notSent

  const exportCSV = async () => {
    setExporting('csv')
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/export.csv', { params: { path }, parseAs: 'blob' })
      if (data) saveBlob(data as Blob, `${name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'campaign'}-report.csv`)
    } catch (e) {
      toast({ kind: 'error', title: tt(m.exportFailed), detail: errorMessage(e), retry: () => void exportCSV() })
    } finally {
      setExporting(undefined)
    }
  }
  const exportPDF = async () => {
    setExporting('pdf')
    try {
      await print.mutateAsync({ params: { path } })
      // The printable report: pick "Save as PDF" as the destination.
      window.setTimeout(() => window.print(), 50)
    } catch (e) {
      toast({ kind: 'error', title: tt(m.exportFailed), detail: errorMessage(e) })
    } finally {
      setExporting(undefined)
    }
  }
  // The links are made at the bottom of the report, where what they share is chosen.
  const share = async () => {
    const link = shares.data?.items[0]
    if (link) {
      const ok = await copyText(link.url)
      toast(ok ? { kind: 'success', title: tt(m.linkCopied), detail: tt(m.linkAtBottom) } : { kind: 'info', title: tt(m.linkAtBottomTitle) })
    }
    shareCard.current?.focus()
  }
  const actions: ReportActions = {
    suppressReason,
    onSuppress: (row) => setSuppressing(row),
    onReview: (row) => setReviewing(row),
  }
  const badge = r ? statusBadge(r.campaign) : undefined
  const caption = r ? headerCaption(r.campaign) : ''

  return (
    <Page
      title={r ? t(m.pageTitle, { name }) : t(m.pageTitleLoading)}
      contentClassName="rep-crumbs"
      badges={
        <>
          <Breadcrumb name={r ? name : t(m.report)} />
          {badge ? (
            <Badge tone={badge.tone} dot>
              {badge.text}
            </Badge>
          ) : null}
          {caption ? <span className="k">{caption}</span> : null}
        </>
      }
      actions={
        sent ? (
          <>
            {ai.button}
            <Button disabledReason={shareReason} loading={exporting === 'csv'} onClick={() => void exportCSV()}>
              {t(m.exportCsv)}
            </Button>
            <Button disabledReason={shareReason} loading={exporting === 'pdf'} onClick={() => void exportPDF()}>
              {t(m.exportPdf)}
            </Button>
            <Button disabledReason={shareReason} onClick={() => void share()}>
              {t(m.shareLink)}
            </Button>
            {editReason ? (
              <Button variant="primary" disabledReason={editReason}>
                {t(m.duplicate)}
              </Button>
            ) : (
              <ButtonLink variant="primary" to="/w/$workspace/campaigns/new" params={{ workspace: slug }} search={{ from: campaignId }}>
                {t(m.duplicate)}
              </ButtonLink>
            )}
          </>
        ) : null
      }
    >
      {q.isError && !r ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.nothingChanged)}
        </ErrorPanel>
      ) : !r ? (
        <ReportSkeleton />
      ) : r.notSent ? (
        <NotSent r={r} campaignId={campaignId} />
      ) : (
        <>
          {r.backlog ? <BacklogAlert b={r.backlog} onRefresh={() => void q.refetch()} /> : null}
          {ai.panel}
          {issue ? <IssueComparison issue={issue} report={r} /> : null}
          <ReportBody
            r={r}
            actions={actions}
            banner={
              !r.campaign.final && r.campaign.finalAt && r.campaign.status !== 'sending' && r.campaign.status !== 'paused' ? (
                <StillComing finalAt={r.campaign.finalAt} issues={COMPARE_ISSUES} action={<FinalAlertButton campaignId={campaignId} />} />
              ) : null
            }
            polls={{ items: polls.data?.items, error: polls.error ?? undefined, retry: () => void polls.refetch() }}
            earn={<CampaignEarn campaignId={campaignId} conversionsInReport />}
          />
          {/* The website's Web section (features/website): the campaign's post. */}
          <ReportWebSection campaignId={campaignId} />
          {/* Product sales credited to this email (digital products, X19), with products:view. */}
          <ReportProductSales campaignId={campaignId} />
          <ShareCard ref={shareCard} r={r} campaignId={campaignId} />
        </>
      )}
      {suppressing ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setSuppressing(undefined)}
          tone="warn"
          badge={t(m.suppressBadge)}
          title={t(m.suppressTitle, { n: suppressing.unsuppressed, count: count(suppressing.unsuppressed) })}
          description={t(m.suppressBody, { type: suppressing.label, name })}
          confirmLabel={t(m.addToSuppression, { count: count(suppressing.unsuppressed) })}
          busyLabel={t(m.adding)}
          onConfirm={async () => {
            const res = await suppress.mutateAsync({ params: { path: { ...path, type: suppressing.type as BounceRow['type'] } } })
            await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report'] })
            setReviewing(undefined)
            toast({
              kind: 'success',
              title: tt(m.suppressDone, { n: res.added, count: count(res.added) }),
              detail: res.already ? tt(m.suppressAlready, { count: count(res.already) }) : tt(m.suppressEveryProvider),
            })
          }}
        />
      ) : null}
      {reviewing ? (
        <ReviewDialog
          row={reviewing}
          campaignId={campaignId}
          suppressReason={suppressReason}
          retryReason={editReason}
          onClose={() => setReviewing(undefined)}
          onSuppress={() => setSuppressing(reviewing)}
        />
      ) : null}
    </Page>
  )
}
