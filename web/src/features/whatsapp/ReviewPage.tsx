import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { ReportError } from '@/features/push/ReportPage'
import { useT } from '@/i18n'
import { andList, clock, count, dayMonthDate, money } from '@/lib/format'
import { deniedReason, useWorkspace } from '@/lib/session'
import { throwIfForbidden } from './common'
import { CheckRow, laterDay } from './Checks'
import { m } from './campaign.messages'
import '@/features/push/push.css'
import './whatsapp.css'

type Review = Schemas['WhatsAppReview']

/** "09:00" for a minute of the day. */
function hhmm(minute: number): string {
  return `${String(Math.floor(minute / 60)).padStart(2, '0')}:${String(minute % 60).padStart(2, '0')}`
}

/** Review & send for a WhatsApp campaign (board WhatsAppChannel). */
export function WhatsAppReviewPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/review', { params: { path: { workspace: slug, campaignId } } })
  throwIfForbidden(q.error)
  const d = q.data
  return (
    <Page
      title={t(m.reviewTitle)}
      documentTitle={d ? t(m.docTitleReview, { name: d.name }) : t(m.reviewTitle)}
      breadcrumb={[
        { label: t(m.campaignsCrumb), to: '/w/$workspace/campaigns', params: { workspace: slug } },
        { label: d?.name ?? '…', to: '/w/$workspace/campaigns/$campaignId/whatsapp', params: { workspace: slug, campaignId } },
        { label: t(m.reviewTitle) },
      ]}
    >
      {d ? (
        <ReviewCard d={d} />
      ) : q.isError ? (
        <ReportError error={q.error} onRetry={() => void q.refetch()} />
      ) : (
        <Loading label={t(m.loading)}>
          <Skeleton w="100%" h={360} />
        </Loading>
      )}
    </Page>
  )
}

function ReviewCard({ d }: { d: Review }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [requesting, setRequesting] = useState(false)
  const [confirming, setConfirming] = useState<'next_day' | 'skip' | null>(null)
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/send')
  const split = d.checks.find((c) => c.key === 'limit')?.split
  const blocks = d.checks.filter((c) => c.status === 'block').length
  const a = d.approval
  const approvalCheck = d.checks.find((c) => c.key === 'approval')
  const pending = a.request?.status === 'pending' ? a.request : undefined
  const needsApproval = (a.required || approvalCheck?.status === 'warn') && !(a.required && a.valid)
  const done = d.status === 'sent' || d.status === 'sending' || d.status === 'paused' || d.status === 'cancelled' || d.status === 'scheduled'
  const n = d.audience.final
  const lists = andList(d.audience.lists.map((l) => l.name))
  const s = d.schedule
  const whenText =
    s.mode === 'local' && s.localDate && s.localMinute !== undefined
      ? t(m.whenLocal, { date: dayMonthDate(`${s.localDate}T12:00:00Z`), time: hhmm(s.localMinute) })
      : s.mode === 'at' && s.sendAt
        ? t(m.whenAt, { date: dayMonthDate(s.sendAt), time: clock(s.sendAt) })
        : t(m.whenNow, { date: dayMonthDate(new Date()), time: clock(new Date()) })
  const doSend = (overLimit?: 'next_day' | 'skip') =>
    send.mutate(
      { params: { path: { workspace: slug, campaignId: d.id } }, body: { version: d.version, overLimit } },
      {
        onSuccess: (r) => {
          toast({
            kind: 'success',
            title:
              r.status === 'scheduled'
                ? t(m.scheduledToast, { date: dayMonthDate(r.scheduledAt), time: clock(r.scheduledAt) })
                : r.later
                  ? t(m.sentSplitToast, { now: count(r.now), later: count(r.later), laterDay: laterDay(split?.laterAt) })
                  : t(m.sentToast, { n: r.recipients }),
          })
          void qc.invalidateQueries({ predicate: (x) => typeof x.queryKey[1] === 'string' && x.queryKey[1].includes('whatsapp') })
          void navigate({ to: '/w/$workspace/campaigns/$campaignId/whatsapp-report', params: { workspace: slug, campaignId: d.id } })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const blockedReason = blocks ? t(m.blockedReason) : !d.access.canSend ? deniedReason('whatsapp:send', d.access.helpers, t(m.sendAction)) : undefined
  return (
    <section className="card">
      <div className="px-head">
        <div>
          <h2 className="h2">{t(m.reviewHeading, { name: d.name })}</h2>
          <span className="k">{t(m.reviewLine, { n, when: whenText, time: clock(d.checkedAt) })}</span>
        </div>
        <div>
          <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/whatsapp" params={{ workspace: slug, campaignId: d.id }}>
            {t(m.backToEditor)}
          </ButtonLink>
        </div>
      </div>
      {d.checks.map((c) => (
        <CheckRow key={c.key} c={c} split={split} />
      ))}
      <div className="flex flex-col gap-3 border-t border-line-2 p-[18px]">
        {split && split.later > 0 ? (
          <div className="wa-split">
            <div>
              <span className="k">{t(m.todayDay, { date: dayMonthDate(new Date()) })}</span>
              <span className="v">{count(split.now)}</span>
              <span className="k">{t(m.fromTime, { time: clock(new Date()) })}</span>
            </div>
            <div>
              <span className="k">{split.laterAt ? t(m.tomorrowDay, { date: dayMonthDate(split.laterAt) }) : t(m.later)}</span>
              <span className="v">{count(split.later)}</span>
              <span className="k">{split.laterAt ? t(m.fromTime, { time: clock(split.laterAt) }) : null}</span>
            </div>
            <div>
              <span className="k">{t(m.total)}</span>
              <span className="v">{count(n)}</span>
              <span className="k">{money(d.cost.amount)}</span>
            </div>
          </div>
        ) : null}
        {pending ? (
          <div className="note i" role="status">
            <span>{t(m.waitingApproval, { time: clock(pending.requestedAt), name: pending.requestedBy.name })}</span>
          </div>
        ) : null}
        {done ? null : needsApproval && !pending ? (
          a.canRequest ? (
            <div className="flex flex-wrap gap-2">
              <Button variant="primary" onClick={() => setRequesting(true)} disabledReason={blocks ? t(m.blockedReason) : undefined}>
                {t(m.requestApproval)}
              </Button>
            </div>
          ) : null
        ) : pending ? null : split && split.later > 0 ? (
          <div className="flex flex-wrap gap-2">
            <Button variant="primary" onClick={() => setConfirming('next_day')} disabledReason={blockedReason}>
              {t(m.sendSplit, { now: count(split.now), later: count(split.later), laterDay: laterDay(split.laterAt) })}
            </Button>
            <Button onClick={() => setConfirming('skip')} disabledReason={blockedReason}>
              {t(m.sendSkip, { now: count(split.now), later: count(split.later) })}
            </Button>
          </div>
        ) : (
          <div className="flex flex-wrap gap-2">
            <Button variant="primary" icon="send" onClick={() => setConfirming('next_day')} disabledReason={blockedReason}>
              {s.mode === 'now' ? t(m.sendAll, { n }) : t(m.scheduleAll, { n })}
            </Button>
          </div>
        )}
      </div>
      {requesting ? <RequestDialog d={d} lists={lists} onClose={() => setRequesting(false)} /> : null}
      {confirming ? (
        <Dialog
          open
          onOpenChange={(o) => !o && !send.isPending && setConfirming(null)}
          title={t(m.sendConfirmTitle, { name: d.name })}
          description={t(m.sendConfirmText, { n: confirming === 'skip' && split ? split.now : n, amount: money(d.cost.amount) })}
          footer={
            <>
              <Button onClick={() => setConfirming(null)} disabled={send.isPending}>
                {t(m.cancel)}
              </Button>
              <Button variant="primary" icon="send" loading={send.isPending} busyLabel={t(m.sending)} onClick={() => doSend(split && split.later > 0 ? confirming : undefined)}>
                {t(m.sendConfirm)}
              </Button>
            </>
          }
        />
      ) : null}
    </section>
  )
}

function RequestDialog({ d, lists, onClose }: { d: Review; lists: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [note, setNote] = useState('')
  const request = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/approval')
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.requestTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button
            variant="primary"
            loading={request.isPending}
            onClick={() =>
              request.mutate(
                { params: { path: { workspace: slug, campaignId: d.id } }, body: { note: note.trim() || undefined } },
                {
                  onSuccess: () => {
                    toast({ kind: 'success', title: t(m.requested) })
                    void qc.invalidateQueries({ predicate: (x) => typeof x.queryKey[1] === 'string' && x.queryKey[1].includes('whatsapp') })
                    onClose()
                  },
                },
              )
            }
          >
            {t(m.requestApproval)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <span className="text-ink-nav">{t(m.requestText, { name: d.name, n: d.audience.final, lists })}</span>
        <div className="lbl">
          <label htmlFor="wa-approval-note">{t(m.noteLabel)}</label>
          <textarea id="wa-approval-note" className="in" rows={2} value={note} onChange={(e) => setNote(e.target.value)} maxLength={1000} />
        </div>
        {d.approval.approvers.length ? (
          <div className="flex flex-wrap items-center gap-2">
            <span className="k">{t(m.approvers)}</span>
            {d.approval.approvers.map((p) => (
              <span key={p.id} className="inline-flex items-center gap-2 whitespace-nowrap">
                <Avatar name={p.name} color={p.color} size={22} />
                <span>{p.name}</span>
              </span>
            ))}
          </div>
        ) : null}
        {request.isError ? (
          <span className="err" role="alert">
            {errorMessage(request.error)}
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}
