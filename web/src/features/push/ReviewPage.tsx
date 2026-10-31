import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { andList, clock, count, dayMonthDate, dayMonthYear, inZone, orList } from '@/lib/format'
import { deniedReason, useMe, useWorkspace } from '@/lib/session'
import { browserName, throwIfForbidden } from './common'
import { platformNames } from './Previews'
import { ReportError } from './ReportPage'
import { m } from './review.messages'
import './push.css'
import './composer.css'

type Review = Schemas['PushReview']
type Check = Schemas['PushCheck']

const checkName: Record<Check['key'], MessageDescriptor> = {
  keys: m.kKeys,
  worker: m.kWorker,
  audience: m.kAudience,
  frequency_cap: m.kCap,
  quiet_hours: m.kQuiet,
  link: m.kLink,
  lengths: m.kLengths,
  image: m.kImage,
  approval: m.kApproval,
}

const statusBadge: Record<Check['status'], { tone: Tone; msg: MessageDescriptor }> = {
  pass: { tone: 'ok', msg: m.pass },
  warn: { tone: 'warn', msg: m.warn },
  block: { tone: 'bad', msg: m.block },
}

/** "New York" for "America/New_York". */
function city(zone: string): string {
  return (zone.split('/').pop() ?? zone).replace(/_/g, ' ')
}

function bare(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

/** Review & send for a push campaign (board PushStates). */
export function PushReviewPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/review', { params: { path: { workspace: slug, campaignId } } })
  throwIfForbidden(q.error)
  const d = q.data
  return (
    <Page
      title={t(m.title)}
      documentTitle={d ? t(m.docTitle, { name: d.name }) : t(m.title)}
      breadcrumb={[
        { label: t(m.campaignsCrumb), to: '/w/$workspace/campaigns', params: { workspace: slug } },
        { label: d?.name ?? t(m.title), to: '/w/$workspace/campaigns/$campaignId/push', params: { workspace: slug, campaignId } },
        { label: t(m.title) },
      ]}
    >
      {d ? <Review d={d} onRecheck={() => void q.refetch()} rechecking={q.isFetching} /> : q.isError ? <ReportError error={q.error} onRetry={() => void q.refetch()} /> : <ReviewSkeleton />}
    </Page>
  )
}

function Review({ d, onRecheck, rechecking }: { d: Review; onRecheck: () => void; rechecking: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [requesting, setRequesting] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const a = d.approval
  const approvalCheck = d.checks.find((c) => c.key === 'approval')
  const others = d.checks.filter((c) => c.key !== 'approval')
  const blocks = d.checks.filter((c) => c.status === 'block').length
  const warns = others.filter((c) => c.status === 'warn').length
  const passed = d.checks.filter((c) => c.status === 'pass').length
  const pending = a.request?.status === 'pending' ? a.request : undefined
  const approved = a.required && a.valid
  const needsApproval = (a.required || approvalCheck?.status === 'warn') && !approved
  const sendable = d.verdict !== 'blocked' && d.access.canSend && (!a.required || a.valid)
  const helpers = approvalCheck?.helpers.length ? approvalCheck.helpers : d.access.helpers
  const done = d.status === 'sent' || d.status === 'sending' || d.status === 'paused' || d.status === 'cancelled'
  const final = d.audience.final
  const sites = andList(d.sites.map((s) => s.host))
  const sendLabel = d.schedule.mode === 'at' && d.schedule.sendAt ? t(m.scheduleFor, { time: `${dayMonthDate(d.schedule.sendAt)}, ${clock(d.schedule.sendAt)}` }) : t(m.sendTo, { n: final })
  return (
    <>
      <Verdict
        d={d}
        blocks={blocks}
        warns={warns}
        passed={passed}
        needsApproval={needsApproval}
        pending={pending}
        approved={approved}
        helpers={helpers}
        done={done}
        actions={
          done ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/push-report" params={{ workspace: slug, campaignId: d.id }}>
              {t(m.openReport)}
            </ButtonLink>
          ) : (
            <>
              <ButtonLink to="/w/$workspace/campaigns/$campaignId/push" params={{ workspace: slug, campaignId: d.id }}>
                {t(m.backToEditor)}
              </ButtonLink>
              {needsApproval && !pending ? (
                a.canRequest ? (
                  <Button variant="primary" onClick={() => setRequesting(true)} disabledReason={d.verdict === 'blocked' && blocks > 0 ? t(m.blockedReason) : undefined}>
                    {t(m.requestApproval)}
                  </Button>
                ) : null
              ) : pending ? null : (
                <Button
                  variant="primary"
                  onClick={() => setConfirming(true)}
                  disabledReason={blocks ? t(m.blockedReason) : !d.access.canSend ? deniedReason('push:send', d.access.helpers, t(m.sendAction)) : undefined}
                >
                  {sendLabel}
                </Button>
              )}
            </>
          )
        }
      />
      <div className="flex flex-wrap items-start gap-4">
        <div className="min-w-0 flex-[2_1_560px]">
          <section className="card">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2">
                  {needsApproval && approvalCheck?.status === 'warn' && blocks === 0 && warns === 0
                    ? t(m.checksTitleApproval, { passed })
                    : passed === d.checks.length
                      ? t(m.checksTitle, { passed })
                      : t(m.checksTitleMixed, { passed, other: d.checks.length - passed })}
                </h2>
                <span className="k">{t(m.checksRun, { time: clock(d.checkedAt) })}</span>
              </div>
              {done ? null : (
                <Button size="sm" icon="refresh" onClick={onRecheck} loading={rechecking}>
                  {t(m.checkAgain)}
                </Button>
              )}
            </div>
            {d.checks.map((c) => (
              <CheckRow key={c.key} c={c} d={d} />
            ))}
          </section>
        </div>
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-4">
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(m.audienceTitle)}</h2>
            </div>
            <div className="pad flex flex-col gap-2">
              {d.sites.map((s) => (
                <Line key={s.host} label={t(m.siteSubscribers, { host: s.host })} value={count(s.subscribers)} />
              ))}
              <Line label={t(m.linked)} value={count(d.audience.linked)} />
              <Line label={t(m.anonymous)} value={count(d.audience.anonymous)} />
              <Line label={t(m.overCap)} value={count(d.audience.overCap)} />
              <div className="mt-0.5 flex justify-between gap-2.5 border-t border-line-2 pt-2 text-[13px]">
                <span>{t(m.finalRecipients)}</span>
                <span className="num font-bold">{count(final)}</span>
              </div>
            </div>
          </section>
          <ScheduleCard d={d} done={done} />
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(m.testTitle)}</h2>
            </div>
            <div className="pad flex flex-col gap-2">
              {d.test ? (
                <div className="flex items-center justify-between gap-2 text-[13px]">
                  <span className="k text-[12.5px]">{t(m.testLine, { browser: t(browserName(d.test.browser)), time: clock(d.test.at) })}</span>
                  <Badge tone={d.test.status === 'failed' ? 'bad' : d.test.status === 'shown' || d.test.status === 'clicked' ? 'ok' : 'neutral'}>{t(m.testStatus, { status: d.test.status })}</Badge>
                </div>
              ) : (
                <span className="k">{t(m.testNone)}</span>
              )}
            </div>
          </section>
          {done ? null : (
            <div className="flex flex-col gap-2">
              {needsApproval && !pending ? (
                a.canRequest ? (
                  <Button variant="primary" size="lg" className="w-full" onClick={() => setRequesting(true)}>
                    {t(m.requestApproval)}
                  </Button>
                ) : null
              ) : pending ? null : (
                <Button
                  variant="primary"
                  size="lg"
                  className="w-full"
                  icon="send"
                  onClick={() => setConfirming(true)}
                  disabledReason={blocks ? t(m.blockedReason) : !sendable ? deniedReason('push:send', d.access.helpers, t(m.sendAction)) : undefined}
                >
                  {sendLabel}
                </Button>
              )}
              <span className="k text-center">{needsApproval ? t(m.sendNoteApproval, { helpers: orList(helpers) }) : t(m.sendNote)}</span>
            </div>
          )}
        </div>
      </div>
      {requesting ? <RequestDialog d={d} sites={sites} onClose={() => setRequesting(false)} /> : null}
      {confirming ? <SendDialog d={d} sites={sites} onClose={() => setConfirming(false)} /> : null}
    </>
  )
}

function Line({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-2.5 text-[13px]">
      <span className="k text-[13px] text-ink-nav!">{label}</span>
      <span className="num font-semibold">{value}</span>
    </div>
  )
}

function Verdict({
  d,
  blocks,
  warns,
  passed,
  needsApproval,
  pending,
  approved,
  helpers,
  done,
  actions,
}: {
  d: Review
  blocks: number
  warns: number
  passed: number
  needsApproval: boolean
  pending?: Schemas['ApprovalRequestView']
  approved: boolean
  helpers: string[]
  done: boolean
  actions: ReactNode
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const remind = $api.useMutation('post', '/api/v1/workspaces/{workspace}/approvals/{approvalId}/remind', {
    onSuccess: () => toast({ kind: 'success', title: t(m.reminded) }),
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const cancel = $api.useMutation('post', '/api/v1/workspaces/{workspace}/approvals/{approvalId}/cancel', {
    onSuccess: () => {
      toast({ kind: 'success', title: t(m.requestCancelled) })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/review'] })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const at = clock(d.checkedAt)
  let tone: 'ok' | 'warn' | 'bad' | 'info' = 'ok'
  let title: string
  let text: ReactNode
  if (done) {
    tone = 'info'
    title = t(m.sentTitle)
    text = t(m.sentText, { time: d.send?.scheduledAt ? `${dayMonthDate(d.send.scheduledAt)}, ${clock(d.send.scheduledAt)}` : '' })
  } else if (d.status === 'scheduled' && d.send) {
    tone = 'info'
    title = t(m.scheduledTitle)
    const local = inZone(d.send.scheduledAt, d.schedule.zone)
    text = t(m.scheduledText, { time: `${local.date}, ${local.time}`, zone: d.schedule.zone })
  } else if (blocks > 0) {
    tone = 'bad'
    title = t(m.blockedTitle, { n: blocks })
    text = t(m.blockedText, { time: at })
  } else if (pending) {
    tone = 'info'
    title = t(m.pendingTitle)
    text = t(m.pendingText, { time: clock(pending.requestedAt), name: pending.requestedBy.name, helpers: orList(helpers) })
  } else if (needsApproval) {
    tone = 'warn'
    title = t(m.approvalTitle)
    text = (
      <T
        msg={d.approval.required && !d.access.canSend ? m.approvalText : d.approval.required ? m.approvalTextRules : m.approvalText}
        values={{ helpers: orList(helpers) }}
      />
    )
  } else if (approved) {
    title = t(m.approvedTitle)
    const by = d.approval.request?.decisions.find((x) => x.decision === 'approve')
    text = t(m.approvedText, { name: by?.approver.name ?? '', time: by ? clock(by.at) : '' })
  } else if (warns > 0) {
    tone = 'warn'
    title = t(m.warningsTitle, { n: warns })
    text = t(m.warningsText, { passed, time: at })
  } else {
    title = t(m.readyTitle)
    text = t(m.readyText, { n: passed, time: at })
  }
  const colors = {
    ok: { border: '#BBF7D0', bg: '#F0FDF4', ring: '#DCFCE7', fg: '#14532D', icon: '#15803D' },
    warn: { border: '#FCD34D', bg: '#FFFBEB', ring: '#FEF3C7', fg: '#78350F', icon: '#B45309' },
    bad: { border: '#FCA5A5', bg: '#FFF5F5', ring: '#FEE2E2', fg: '#7F1D1D', icon: '#B91C1C' },
    info: { border: '#BFDBFE', bg: '#EFF6FF', ring: '#DBEAFE', fg: '#1E3A8A', icon: '#1E40AF' },
  }[tone]
  return (
    <section className="card flex flex-wrap items-center gap-3.5 px-5 py-4" style={{ borderColor: colors.border, background: colors.bg }} role={tone === 'bad' ? 'alert' : 'status'}>
      <span className="inline-flex size-[38px] flex-none items-center justify-center rounded-full" style={{ background: colors.ring, color: colors.icon }} aria-hidden="true">
        <Icon name={tone === 'ok' ? 'check' : tone === 'info' ? 'clock' : 'alertCircle'} size={18} strokeWidth={2.2} />
      </span>
      <div className="flex min-w-0 flex-[1_1_380px] flex-col gap-0.5">
        <h2 className="m-0 text-[19px] font-bold" style={{ color: colors.fg }}>
          {title}
        </h2>
        <span style={{ color: colors.fg }}>{text}</span>
      </div>
      {pending ? (
        <>
          <Button loading={remind.isPending} onClick={() => remind.mutate({ params: { path: { workspace: slug, approvalId: pending.id } } })}>
            {t(m.remind)}
          </Button>
          <Button variant="danger" loading={cancel.isPending} onClick={() => cancel.mutate({ params: { path: { workspace: slug, approvalId: pending.id } } })}>
            {t(m.cancelRequest)}
          </Button>
        </>
      ) : null}
      {actions}
    </section>
  )
}

function CheckRow({ c, d }: { c: Check; d: Review }) {
  const t = useT()
  const { me } = useWorkspace()
  const b = statusBadge[c.status]
  const mono = (x: ReactNode[]) => <span className="mono">{x}</span>
  let detail: ReactNode
  switch (c.key) {
    case 'keys':
      detail = !c.keysCreatedAt ? t(m.keysNone) : c.keysAcceptedAt ? t(m.keysOk, { date: dayMonthYear(c.keysCreatedAt) }) : t(m.keysNew, { date: dayMonthYear(c.keysCreatedAt) })
      break
    case 'worker': {
      if (!c.sites.length) {
        detail = t(m.workerNone)
        break
      }
      const parts = c.sites.map((s) => {
        switch (s.status) {
          case 'built_in':
            return t(m.workerBuiltIn, { host: s.host })
          case 'found':
            return t(m.workerFound, { host: s.host, version: s.version ?? 1 })
          case 'old_version':
            return t(m.workerOld, { host: s.host, version: s.version ?? 1 })
          case 'missing':
            return t(m.workerMissing, { host: s.host, url: s.url, status: s.httpStatus ?? 404, count: s.subscribers })
          case 'unreachable':
            return t(m.workerUnreachable, { host: s.host, count: s.subscribers })
          case 'not_a_worker':
            return t(m.workerNotWorker, { host: s.host, url: s.url, count: s.subscribers })
          default:
            return t(m.workerUnchecked, { host: s.host })
        }
      })
      if (c.sites.length === 1) parts.push(t(m.workerOnly))
      detail = parts.join(' · ')
      break
    }
    case 'audience':
      detail = c.count ? t(m.audienceLine, { count: c.count, sites: andList(d.sites.map((s) => s.host)), linked: count(c.linked), anonymous: count(c.anonymous) }) : t(m.audienceEmpty)
      break
    case 'frequency_cap':
      detail = c.overCap ? t(m.capOver, { n: c.overCap, cap: c.cap }) : c.hadToday ? t(m.capSome, { n: c.hadToday, cap: c.cap }) : t(m.capNobody, { cap: c.cap })
      break
    case 'quiet_hours': {
      const at = inZone(c.at, c.atZone).time
      const start = c.window.split(/[–-]/)[0]?.trim() ?? '09:00'
      detail = c.quietOff
        ? t(m.quietOff)
        : c.skipped
          ? t(m.quietSkip, { time: at, city: city(c.atZone), window: c.window, inside: c.inside, skipped: c.skipped })
          : c.held
            ? t(m.quietSome, { time: at, city: city(c.atZone), window: c.window, inside: c.inside, held: c.held, start })
            : t(m.quietAll, { time: at, city: city(c.atZone), window: c.window })
      break
    }
    case 'link': {
      const link = bare(c.link ?? '')
      detail = !c.link ? (
        t(m.linkNone)
      ) : c.linkDemo ? (
        <T msg={m.linkDemo} values={{ link, mono }} />
      ) : c.linkError ? (
        <T msg={m.linkError} values={{ link, error: c.linkError, mono }} />
      ) : c.linkStatus ? (
        <T msg={c.linkStatus < 400 ? m.linkOk : m.linkBad} values={{ link, status: c.linkStatus, ms: c.linkMillis ?? 0, mono }} />
      ) : (
        <T msg={m.linkLater} values={{ link, mono }} />
      )
      break
    }
    case 'lengths': {
      if (c.status === 'block' && !c.titleLength) {
        detail = t(m.lengthsNoTitle)
        break
      }
      const where = c.cuts.map((x) => (x.title ? t(m.cutTitle, { platform: t(platformNames[x.platform].name) }) : t(m.cutLines, { platform: t(platformNames[x.platform].name), n: x.lines })))
      detail = where.length ? t(m.lengthsCut, { title: c.titleLength, message: c.messageLength, where: andList(where) }) : t(m.lengthsNone, { title: c.titleLength, message: c.messageLength })
      break
    }
    case 'image':
      detail = c.image
        ? t(c.status === 'warn' ? m.imageWarn : m.imageOk, {
            w: c.image.width,
            h: c.image.height,
            ratio: c.ratio ?? '',
            size: `${Math.max(1, Math.round(c.image.bytes / 1024))} KB`,
          })
        : t(m.imageNone)
      break
    case 'approval':
      detail =
        c.status === 'pass' ? (
          c.author ? (
            <T msg={m.approvalOk} values={{ name: c.author }} />
          ) : (
            t(m.approvalOkAnon)
          )
        ) : (
          <T
            msg={m.approvalNeeds}
            values={{
              name: c.author ?? '',
              role: c.authorRole || me.workspace.roleName,
              helpers: orList(c.helpers),
            }}
          />
        )
      break
  }
  return (
    <div className={c.status === 'warn' ? 'px-chk w' : c.status === 'block' ? 'px-chk b' : 'px-chk'}>
      <Badge tone={b.tone}>{t(b.msg)}</Badge>
      <div className="tx">
        <span className="font-semibold">{t(checkName[c.key])}</span>
        <span className="k text-ink-2!">{detail}</span>
      </div>
    </div>
  )
}

function ScheduleCard({ d, done }: { d: Review; done: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/send-settings', {
    onSuccess: (next) => qc.setQueryData(['get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/review', { params: { path: { workspace: slug, campaignId: d.id } } }], next),
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const s = d.schedule
  const editable = d.access.canEdit && !done && d.status !== 'scheduled'
  const start = d.guards.window.split(/[–-]/)[0]?.trim() ?? '09:00'
  const put = (patch: Partial<Schemas['PushSendSettings']>) =>
    save.mutate({ params: { path: { workspace: slug, campaignId: d.id } }, body: { mode: s.mode, sendAt: s.sendAt, outside: s.outside, ...patch } })
  const local = s.sendAt ? inZone(s.sendAt, s.zone) : undefined
  const ttl = s.ttlMinutes % 60 === 0 ? t(m.hours, { n: s.ttlMinutes / 60 }) : t(m.minutes, { n: s.ttlMinutes })
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(m.scheduleTitle)}</h2>
      </div>
      <div className="pad flex flex-col gap-2">
        <label className={s.mode === 'now' ? 'pc-optc on' : 'pc-optc'}>
          <input type="radio" name="push-when" checked={s.mode === 'now'} disabled={!editable || save.isPending} onChange={() => put({ mode: 'now', sendAt: undefined })} />
          <span className="flex flex-col gap-0.5 font-semibold">
            {t(m.sendNow)}
            <span className="k">{d.guards.quiet ? t(m.sendNowHint, { start }) : t(m.sendNowHintOff)}</span>
          </span>
        </label>
        <label className={s.mode === 'at' ? 'pc-optc on' : 'pc-optc'}>
          <input
            type="radio"
            name="push-when"
            checked={s.mode === 'at'}
            disabled={!editable || save.isPending}
            onChange={() => put({ mode: 'at', sendAt: new Date(Date.now() + 3_600_000).toISOString() })}
          />
          <span className="flex flex-col gap-0.5 font-semibold">
            {t(m.scheduleAt)}
            <span className="k">{t(m.scheduleAtHint, { zone: s.zone })}</span>
          </span>
        </label>
        {s.mode === 'at' ? (
          <div className="lbl">
            <label htmlFor="push-send-at">{t(m.scheduleWhen)}</label>
            <input
              id="push-send-at"
              className="in"
              type="datetime-local"
              disabled={!editable}
              defaultValue={s.sendAt ? toLocalInput(s.sendAt, s.zone) : ''}
              onBlur={(e) => {
                const v = e.target.value
                if (!v) return
                const iso = fromLocalInput(v, s.zone)
                if (iso !== s.sendAt) put({ sendAt: iso })
              }}
            />
            {local ? <span className="k">{`${local.date}, ${local.time} · ${s.zone}`}</span> : null}
          </div>
        ) : null}
        {d.guards.quiet ? (
          <div className="lbl">
            <label htmlFor="push-outside">{t(m.outsideLabel)}</label>
            <select id="push-outside" className="in" value={s.outside} disabled={!editable || save.isPending} onChange={(e) => put({ outside: e.target.value as 'hold' | 'skip' })}>
              <option value="hold">{t(m.hold)}</option>
              <option value="skip">{t(m.skip)}</option>
            </select>
          </div>
        ) : null}
        <div className="flex justify-between pt-1 text-[13px]">
          <span className="k text-[13px]">{t(m.expiresAfter)}</span>
          <span className="font-semibold">{ttl}</span>
        </div>
      </div>
    </section>
  )
}

/** A moment as "2026-10-22T09:00" in a time zone, for a datetime-local field. */
function toLocalInput(iso: string, zone: string): string {
  const p: Record<string, string> = {}
  for (const x of new Intl.DateTimeFormat('en-CA', { timeZone: zone, hourCycle: 'h23', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).formatToParts(new Date(iso)))
    p[x.type] = x.value
  return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`
}

/** The instant of a datetime-local value in a time zone. */
function fromLocalInput(v: string, zone: string): string {
  const [date = '', time = '00:00'] = v.split('T')
  const [y, mo, d] = date.split('-').map(Number)
  const [h, min] = time.split(':').map(Number)
  const guess = Date.UTC(y!, mo! - 1, d!, h!, min!)
  const offset = (t0: number) => {
    const p: Record<string, number> = {}
    for (const x of new Intl.DateTimeFormat('en-US', { timeZone: zone, hourCycle: 'h23', year: 'numeric', month: 'numeric', day: 'numeric', hour: 'numeric', minute: 'numeric' }).formatToParts(new Date(t0)))
      if (x.type !== 'literal') p[x.type] = Number(x.value)
    return Date.UTC(p.year!, p.month! - 1, p.day!, p.hour! % 24, p.minute!) - t0
  }
  const first = guess - offset(guess)
  return new Date(guess - offset(first)).toISOString()
}

function RequestDialog({ d, sites, onClose }: { d: Review; sites: string; onClose: () => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const [note, setNote] = useState('')
  const request = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/approval')
  const user = useMe().data?.user
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{`${user?.name ?? ''} · ${me.workspace.roleName}`}</span>}
      title={t(m.requestTitle)}
      hideClose
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
                    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/review'] })
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
        <span className="text-ink-nav">{t(m.requestText, { name: d.name, n: d.audience.final, sites })}</span>
        <div className="lbl">
          <label htmlFor="push-approval-note">{t(m.noteLabel)}</label>
          <textarea id="push-approval-note" className="in" rows={2} value={note} onChange={(e) => setNote(e.target.value)} maxLength={1000} />
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

function SendDialog({ d, sites, onClose }: { d: Review; sites: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/send')
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !send.isPending && onClose()}
      title={t(m.sendConfirmTitle, { name: d.name })}
      description={t(m.sendConfirmText, { n: d.audience.final, sites })}
      footer={
        <>
          <Button onClick={onClose} disabled={send.isPending}>
            {t(m.cancel)}
          </Button>
          <Button
            variant="primary"
            icon="send"
            loading={send.isPending}
            busyLabel={t(m.sending)}
            onClick={() =>
              send.mutate(
                { params: { path: { workspace: slug, campaignId: d.id } }, body: { version: d.version } },
                {
                  onSuccess: (r) => {
                    toast({
                      kind: 'success',
                      title: r.status === 'scheduled' ? t(m.scheduledToast, { time: `${dayMonthDate(r.scheduledAt)}, ${clock(r.scheduledAt)}` }) : t(m.sentToast, { n: r.recipients }),
                    })
                    void navigate({ to: '/w/$workspace/campaigns/$campaignId/push-report', params: { workspace: slug, campaignId: d.id } })
                  },
                },
              )
            }
          >
            {d.schedule.mode === 'at' ? t(m.scheduleFor, { time: d.schedule.sendAt ? `${dayMonthDate(d.schedule.sendAt)}, ${clock(d.schedule.sendAt)}` : '' }) : t(m.sendConfirm)}
          </Button>
        </>
      }
    >
      {send.isError ? (
        <span className="err" role="alert">
          {errorMessage(send.error)}
        </span>
      ) : null}
    </Dialog>
  )
}

function ReviewSkeleton() {
  const t = useT()
  return (
    <Loading label={t(m.loading)} className="flex flex-col gap-4">
      <div className="card flex items-center gap-3.5 px-5 py-4">
        <Skeleton w={38} h={38} r={19} />
        <div className="flex flex-1 flex-col gap-2">
          <Skeleton w={200} h={16} />
          <Skeleton w="60%" h={11} />
        </div>
      </div>
      <div className="card flex flex-col gap-4 p-5">
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className="flex gap-3">
            <Skeleton w={44} h={20} r={10} />
            <Skeleton w="70%" h={12} />
          </div>
        ))}
      </div>
    </Loading>
  )
}
