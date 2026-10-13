import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { clock, count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ReportBody, type ReportActions } from './ReportBody'
import { ReportSkeleton } from './ReportSkeleton'
import { copyText, ShareCard, ShareDialog, useShares } from './ShareCard'
import { StateAlert } from './StateAlert'
import { headerCaption, notSentLead, statusBadge, type CampaignReport } from './words'
import './reports.css'

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
  const { slug } = useWorkspace()
  return (
    <nav aria-label="Breadcrumb" className="flex items-center gap-2 text-[13px]">
      <Link to="/w/$workspace/campaigns" params={{ workspace: slug }} className="text-ink-3 no-underline hover:text-ink-3">
        Campaigns
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
  const { slug } = useWorkspace()
  const n = r.notSent!
  return (
    <section className="card flex flex-col items-center gap-2.5 border-dashed px-7 py-9 text-center">
      <svg width="48" height="48" viewBox="0 0 48 48" aria-hidden="true">
        <rect x="6" y="12" width="36" height="26" rx="4" fill="none" stroke="#9AA3B2" strokeWidth="2" />
        <path d="M12 32V22M20 32V18M28 32V26M36 32V15" stroke="#9AA3B2" strokeWidth="2.5" strokeLinecap="round" />
      </svg>
      <h2 className="m-0 text-[17px] font-bold">No results for {r.campaign.name} yet</h2>
      <span className="max-w-[520px] text-ink-2">
        {notSentLead(n)} Delivery, bounce and complaint counts appear here live as it sends; opens and clicks keep updating for 72 hours.
      </span>
      <div className="mt-1 flex flex-wrap justify-center gap-2.5">
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }}>
          View Send Readiness
        </ButtonLink>
        {n.lastReport ? (
          <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: n.lastReport.id }}>
            See the last report instead
          </ButtonLink>
        ) : null}
      </div>
    </section>
  )
}

/** Error: provider events wait to be processed, so some counts are behind. */
function BacklogAlert({ b, onRefresh }: { b: Schemas['ReportIngestBacklog']; onRefresh: () => void }) {
  const { slug } = useWorkspace()
  const first = b.providers[0]!
  const since = clock(b.since)
  const queue = first.feed === 'SNS' ? `The SNS queue for ${first.name}` : `${first.name}'s ${first.feed === 'Bounce mailbox' ? 'bounce mailbox' : 'webhook events'}`
  const waiting = b.providers.reduce((s, p) => s + p.waiting, 0)
  const others = b.providers.slice(1).map((p) => p.name)
  const current = b.current.length ? ` ${b.current.join(' and ')} events are current.` : ''
  return (
    <StateAlert
      title={`Event ingestion is ${b.minutes} ${b.minutes === 1 ? 'minute' : 'minutes'} behind`}
      actions={
        <>
          <Button size="sm" onClick={onRefresh}>
            Refresh
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/providers/events" params={{ workspace: slug }}>
            View event queue
          </ButtonLink>
        </>
      }
    >
      {queue} backed up at {since} ({count(waiting)} events waiting){others.length ? `, and ${others.join(' and ')} too` : ''}. Delivered, bounce and complaint counts below are complete up to {since} and will
      catch up on their own.{current}
    </StateAlert>
  )
}

/** The Review dialog of a bounce type: what the providers said, then suppress or leave them. */
function ReviewDialog({ row, campaignId, onClose, onSuppress, suppressReason }: { row: BounceRow; campaignId: string; onClose: () => void; onSuppress: () => void; suppressReason?: string }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/bounces/{type}', {
    params: { path: { workspace: slug, campaignId, type: row.type } },
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={`${row.label} · ${count(row.count)}`}
      description="What each sending provider said. Addresses aren't shown here; suppressing adds them to the shared list every provider uses."
      size="lg"
      footer={
        <>
          <Button onClick={onClose}>Leave them</Button>
          {row.unsuppressed > 0 ? (
            <Button variant="primary" disabledReason={suppressReason} onClick={onSuppress}>
              Add {count(row.unsuppressed)} to suppression
            </Button>
          ) : null}
        </>
      }
    >
      {q.isError ? (
        <ErrorPanel title="Couldn't load the diagnostics" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <div className="tscroll">
          <table className="tbl rep-tbl phone-wrap">
            <thead>
              <tr>
                <th scope="col">Provider and code</th>
                <th scope="col">What the provider said</th>
                <th scope="col" className="r">
                  Count
                </th>
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
                    Loading…
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      )}
    </Dialog>
  )
}

/** 7.7 Campaign report. */
export function ReportPage({ campaignId }: { campaignId: string }) {
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
  const shareReason = usePermissionReason('reports:share')
  const editReason = usePermissionReason('campaigns:edit')
  const suppressReason = usePermissionReason('subscribers:edit')
  const shares = useShares(campaignId, !shareReason && Boolean(r) && !r?.notSent)
  const [sharing, setSharing] = useState(false)
  const [exporting, setExporting] = useState<'csv' | 'pdf'>()
  const [suppressing, setSuppressing] = useState<BounceRow>()
  const [reviewing, setReviewing] = useState<BounceRow>()
  const shareCard = useRef<HTMLElement>(null)
  const print = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/print')
  const suppress = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/bounces/{type}/suppress')

  // A campaign that doesn't exist (or a role without reports) gets the
  // shell's 404 or 403 screen.
  if (!r && q.error instanceof ApiError && (q.error.status === 404 || q.error.status === 403)) throw q.error

  const name = r?.campaign.name ?? 'Report'
  const sent = r && !r.notSent

  const exportCSV = async () => {
    setExporting('csv')
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/export.csv', { params: { path }, parseAs: 'blob' })
      if (data) saveBlob(data as Blob, `${name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'campaign'}-report.csv`)
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't export the report", detail: errorMessage(e), retry: () => void exportCSV() })
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
      toast({ kind: 'error', title: "Couldn't export the report", detail: errorMessage(e) })
    } finally {
      setExporting(undefined)
    }
  }
  const share = async () => {
    const link = shares.data?.items[0]
    if (!link) {
      setSharing(true)
      return
    }
    const ok = await copyText(link.url)
    toast(ok ? { kind: 'success', title: 'Link copied', detail: 'The read-only link is at the bottom of the report.' } : { kind: 'info', title: 'The read-only link is at the bottom of the report' })
    shareCard.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    shareCard.current?.focus({ preventScroll: true })
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
      title={r ? `${name} report` : 'Campaign report'}
      contentClassName="rep-crumbs"
      badges={
        <>
          <Breadcrumb name={r ? name : 'Report'} />
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
            <Button disabledReason={shareReason} loading={exporting === 'csv'} onClick={() => void exportCSV()}>
              Export CSV
            </Button>
            <Button disabledReason={shareReason} loading={exporting === 'pdf'} onClick={() => void exportPDF()}>
              Export PDF
            </Button>
            <Button disabledReason={shareReason} onClick={() => void share()}>
              Share read-only link
            </Button>
            {editReason ? (
              <Button variant="primary" disabledReason={editReason}>
                Duplicate campaign
              </Button>
            ) : (
              <ButtonLink variant="primary" to="/w/$workspace/campaigns/new" params={{ workspace: slug }} search={{ from: campaignId }}>
                Duplicate campaign
              </ButtonLink>
            )}
          </>
        ) : null
      }
    >
      {q.isError && !r ? (
        <ErrorPanel title="Couldn't load this report" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed.
        </ErrorPanel>
      ) : !r ? (
        <ReportSkeleton />
      ) : r.notSent ? (
        <NotSent r={r} campaignId={campaignId} />
      ) : (
        <>
          {r.backlog ? <BacklogAlert b={r.backlog} onRefresh={() => void q.refetch()} /> : null}
          <ReportBody r={r} actions={actions} />
          <ShareCard ref={shareCard} campaignId={campaignId} onCreate={() => setSharing(true)} />
        </>
      )}
      <ShareDialog open={sharing} onOpenChange={setSharing} campaignId={campaignId} />
      {suppressing ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setSuppressing(undefined)}
          tone="warn"
          badge="Suppression list"
          title={`Add ${count(suppressing.unsuppressed)} ${suppressing.unsuppressed === 1 ? 'address' : 'addresses'} to the suppression list?`}
          description={`They bounced as “${suppressing.label}” on ${name}. Every sending provider stops mailing them; you can take an address off the list in Audience › Suppressions.`}
          confirmLabel={`Add ${count(suppressing.unsuppressed)} to suppression`}
          busyLabel="Adding…"
          onConfirm={async () => {
            const res = await suppress.mutateAsync({ params: { path: { ...path, type: suppressing.type as BounceRow['type'] } } })
            await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report'] })
            setReviewing(undefined)
            toast({
              kind: 'success',
              title: `${count(res.added)} ${res.added === 1 ? 'address' : 'addresses'} suppressed`,
              detail: res.already ? `${count(res.already)} were already on the list.` : 'Every sending provider gets the update.',
            })
          }}
        />
      ) : null}
      {reviewing ? (
        <ReviewDialog
          row={reviewing}
          campaignId={campaignId}
          suppressReason={suppressReason}
          onClose={() => setReviewing(undefined)}
          onSuppress={() => setSuppressing(reviewing)}
        />
      ) : null}
    </Page>
  )
}
