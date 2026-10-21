import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Table, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { channelIcon } from '@/features/channels/ChannelSwitch'
import { formatList, T, useT, type MessageDescriptor } from '@/i18n'
import { clock, clockSeconds, count, dayMonthDate, inZone, money, pct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { DuplicateTextButton } from './CampaignRow'
import { CountryChips, CountryName, countryName, providerColor, regionLabel, regionSentence, throwIfForbidden, useKindName, useRefreshSms } from './common'
import { m as sm } from './messages'
import { m } from './report.messages'
import './sms.css'

type Report = Schemas['SmsReport']

const reasonWord: Record<string, MessageDescriptor> = {
  unreachable: sm.failUnreachable,
  filtered: sm.failFiltered,
  not_in_service: sm.failNot_in_service,
  landline: sm.failLandline,
  invalid: sm.failInvalid,
  opted_out: sm.failOpted_out,
  rejected: sm.failRejected,
  provider_down: sm.failProvider_down,
  other: sm.failOther,
}

/** The SMS campaign report (SmsReport, SmsReportStates). */
export function SmsReportPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/report',
    { params: { path: { workspace: slug, campaignId } } },
    { refetchInterval: (query) => (query.state.data && (query.state.data.phase === 'sending' || query.state.data.phase === 'scheduled') ? 10_000 : false) },
  )
  throwIfForbidden(q.error)
  const r = q.data
  return (
    <Page
      title={r ? t(m.title, { name: r.name }) : t(m.campaigns)}
      breadcrumb={[{ label: t(m.campaigns), to: '/w/$workspace/campaigns', params: { workspace: slug } }, { label: r?.name ?? '' }]}
      badges={
        <>
          {r ? <When r={r} /> : null}
          <Badge tone="neutral" className="gap-[5px]">
            {channelIcon.sms}
            {t(m.sms)}
          </Badge>
        </>
      }
      actions={r ? <Actions r={r} /> : null}
    >
      {q.isError ? (
        <ErrorPanel
          title={t(m.errorTitle)}
          reference={problemReference(q.error)}
          actions={
            <>
              <Button variant="primary" onClick={() => void q.refetch()}>
                {t(m.retry)}
              </Button>
              <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
                {t(m.statusOnOverview)}
              </ButtonLink>
            </>
          }
        >
          {errorMessage(q.error)} {t(m.errorText)}
        </ErrorPanel>
      ) : !r ? (
        <div className="card flex flex-col gap-3 p-5" aria-busy="true" aria-label={t(m.loadingPlain)}>
          <span className="k">{t(m.loadingPlain)}</span>
          <div className="sms-stats">
            {[0, 1, 2, 3].map((i) => (
              <Skeleton key={i} h={70} />
            ))}
          </div>
          <Skeleton h={200} />
        </div>
      ) : (
        <Body r={r} />
      )}
    </Page>
  )
}

/** When it went or goes, as a badge: ● Sent 14 Oct, at 10:00 in each recipient's time zone. */
function When({ r }: { r: Report }) {
  const t = useT()
  const at = r.startedAt ?? r.scheduledAt
  if (!at) return null
  const date = dayMonthDate(at)
  const sent = r.phase === 'completed' || r.phase === 'sending' || r.phase === 'paused'
  const text =
    r.scheduleMode === 'local' && r.localTime
      ? sent
        ? t(m.sentLocal, { date, time: r.localTime })
        : t(m.scheduledLocal, { date, time: r.localTime })
      : sent
        ? t(m.sentAt, { date, time: clock(at) })
        : t(m.scheduledAt, { date, time: clock(at) })
  return (
    <Badge tone={r.phase === 'completed' ? 'ok' : r.phase === 'cancelled' ? 'neutral' : 'info'} dot>
      {text}
    </Badge>
  )
}

function Actions({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <>
      <ButtonAnchor href={`/api/v1/workspaces/${slug}/sms-campaigns/${r.campaignId}/report.csv`} download="">
        {t(m.exportCsv)}
      </ButtonAnchor>
      <DuplicateTextButton campaignId={r.campaignId} label={t(m.duplicate)} small={false} primary />
    </>
  )
}

function Body({ r }: { r: Report }) {
  const sending = r.phase === 'sending' || r.phase === 'paused' || r.phase === 'scheduled'
  return (
    <>
      {r.outage ? <Outage r={r} /> : null}
      {r.pause && !r.outage ? <Paused r={r} /> : null}
      <Tiles r={r} />
      {sending && !r.outage && !r.pause ? <Sending r={r} /> : null}
      {!sending ? (
        <div className="flex flex-wrap items-start gap-4">
          <div className="flex min-w-0 flex-col gap-4" style={{ flex: '2 1 560px' }}>
            <OverTime r={r} />
          </div>
          <div className="flex min-w-0 flex-col gap-4" style={{ flex: '1 1 320px' }}>
            <BySender r={r} />
          </div>
        </div>
      ) : null}
      <ByCountry r={r} />
      <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(420px, 100%), 1fr))' }}>
        <Failures r={r} />
        <Clicks r={r} />
      </div>
      <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(420px, 100%), 1fr))' }}>
        <Replies r={r} />
        <div className="flex min-w-0 flex-col gap-4">
          <Held r={r} />
          {r.sample ? <Sample r={r} /> : null}
        </div>
      </div>
    </>
  )
}

function Tile({ label, value, sub, hot }: { label: string; value: string; sub?: ReactNode; hot?: boolean }) {
  return (
    <div className={`card sms-stat${hot ? ' hot' : ''}`}>
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      {sub ? <span className="s">{sub}</span> : null}
    </div>
  )
}

/** The Failed tile's parts: "9 unreachable · 3 filtered · 2 not in service". */
const failedWord: Record<string, MessageDescriptor> = {
  unreachable: m.fbUnreachable,
  filtered: m.fbFiltered,
  not_in_service: m.fbNot_in_service,
  landline: m.fbLandline,
  invalid: m.fbInvalid,
  opted_out: m.fbOpted_out,
  rejected: m.fbRejected,
  provider_down: m.fbProvider_down,
}

function Tiles({ r }: { r: Report }) {
  const t = useT()
  const tl = r.tiles
  const sending = r.phase === 'sending' || r.phase === 'paused'
  const failedBy = Object.entries(tl.failedBy)
    .filter(([, n]) => n > 0)
    .sort((a, b) => b[1] - a[1])
    .map(([reason, n]) => t(failedWord[reason] ?? m.fbOther, { count: n }))
  const providers = tl.byProvider.map((p) => `${p.name} ${count(p.count)}`).join(' · ')
  if (sending) {
    return (
      <div className="sms-stats">
        <Tile label={t(m.tSentSoFar)} value={count(tl.sent)} sub={providers} />
        <Tile label={t(m.tDelivered)} value={count(tl.delivered)} sub={t(m.tDeliveredSoFar, { pct: pct(tl.deliveredPct) })} />
        <Tile label={t(m.tClicks)} value={count(tl.clicks)} sub={t(m.tClicksPeople, { machine: count(tl.machine) })} />
        <Tile label={t(m.tCostSoFar)} value={money(tl.cost)} sub={t(m.tCostOf, { total: money(tl.planned) })} />
      </div>
    )
  }
  return (
    <div className="sms-stats">
      <Tile label={t(m.tSent)} value={count(tl.sent)} sub={t(m.tSentSub, { providers, skipped: tl.skipped })} />
      <Tile label={t(m.tDelivered)} value={count(tl.delivered)} sub={t(m.tDeliveredSub, { pct: pct(tl.deliveredPct) })} />
      <Tile label={t(m.tFailed)} value={count(tl.failed)} sub={failedBy.join(' · ')} />
      <Tile label={t(m.tClicks)} value={count(tl.clicks)} sub={t(m.tClicksSub, { pct: pct(tl.clickPct), machine: count(tl.machine) })} />
      <Tile label={t(m.tOptOuts)} value={count(tl.optOuts)} sub={t(m.tOptOutsSub, { pct: pct(tl.optOutPct), keyword: tl.optOutsByKeyword ? 'yes' : 'no' })} />
      <Tile label={t(m.tCost)} value={money(tl.cost)} sub={t(m.tCostSub, { perText: money(tl.perText, 3), within: tl.withinBudget ? 'yes' : 'no' })} />
    </div>
  )
}

/** A send in progress: each time zone's progress, Pause and Cancel. */
function Sending({ r }: { r: Report }) {
  const t = useT()
  const tl = r.tiles
  const total = r.steps.reduce((n, s) => n + s.total, 0)
  return (
    <section className="card flex min-w-0 flex-col">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.sendingTitle, { sent: count(tl.sent), total: count(total || tl.planned) })}</h2>
          <span className="k">
            {r.localTime
              ? t(m.sendingSub, { time: r.startedAt ? `${dayMonthDate(r.startedAt)}, ${clock(r.startedAt)} UTC` : '', local: r.localTime })
              : t(m.sendingSubNow, { time: r.startedAt ? `${dayMonthDate(r.startedAt)}, ${clock(r.startedAt)}` : '' })}
          </span>
        </div>
        <SendControls r={r} />
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="bar stack" style={{ height: 10 }} role="img" aria-label={t(m.sendingTitle, { sent: count(tl.sent), total: count(total) })}>
          <i style={{ width: `${pctOf(tl.delivered, total)}%`, background: 'var(--accent)' }} />
          <i style={{ width: `${pctOf(tl.queued, total)}%`, background: '#99F6E4' }} />
          <i style={{ width: `${pctOf(tl.failed, total)}%`, background: '#B91C1C' }} />
        </div>
        <div className="sms-bm-l">
          <span>
            <span className="sms-bsw" style={{ background: 'var(--accent)' }} />
            {t(m.sDelivered, { count: count(tl.delivered) })}
          </span>
          <span>
            <span className="sms-bsw" style={{ background: '#99F6E4' }} />
            {t(m.sWaiting, { count: count(tl.queued) })}
          </span>
          <span>
            <span className="sms-bsw" style={{ background: '#B91C1C' }} />
            {t(m.sFailed, { count: count(tl.failed) })}
          </span>
          <span>
            <span className="sms-bsw" style={{ background: 'var(--line-2)' }} />
            {t(m.sZone, { count: count(tl.waiting) })}
          </span>
        </div>
        <ZoneRows r={r} />
      </div>
    </section>
  )
}

function pctOf(n: number, total: number): string {
  return total > 0 ? ((n / total) * 100).toFixed(2) : '0'
}

function ZoneRows({ r }: { r: Report }) {
  const t = useT()
  const statusText = (s: Schemas['SmsZoneStep']) => {
    switch (s.status) {
      case 'sent':
        return { tone: 'ok' as Tone, text: t(m.zSent) }
      case 'sending': {
        const rate = r.senders.find((x) => x.countries.some((c) => s.countries.includes(c)))?.throughput
        return { tone: 'info' as Tone, text: rate ? t(m.zSending, { rate }) : t(m.zSendingPlain) }
      }
      case 'paused':
        return { tone: 'warn' as Tone, text: t(m.zPaused) }
      case 'held':
        return { tone: 'warn' as Tone, text: t(m.zHeld) }
    }
    return { tone: 'neutral' as Tone, text: t(m.zStarts, { time: clock(s.at) }) }
  }
  return (
    <div className="flex flex-col">
      {r.steps.map((s, i) => {
        const st = statusText(s)
        return (
          <div key={i} className="sms-tz">
            <span className="tn">
              <span className="font-semibold">{s.region || regionSentence(s.countries)}</span>
              <span className="k">{t(m.zoneAt, { local: s.local, abbr: s.abbr, utc: utcClock(s.at) })}</span>
            </span>
            <span className="tb">
              <span className="bar">
                <i style={{ width: `${pctOf(s.sent, s.total)}%`, background: 'var(--accent)' }} />
              </span>
            </span>
            <span className="n">{count(s.sent)}</span>
            <span className="k">{t(m.zoneOf, { total: count(s.total) })}</span>
            <Badge tone={st.tone}>{st.text}</Badge>
          </div>
        )
      })}
    </div>
  )
}

/** "14:00" in UTC. */
function utcClock(at: string): string {
  return inZone(at, 'UTC').time
}

function SendControls({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const qc = useQueryClient()
  const [cancelling, setCancelling] = useState(false)
  const pauseReason = usePermissionReason('campaigns:pause')
  const again = () => {
    void refresh()
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/report'] })
  }
  const path = { workspace: slug, campaignId: r.campaignId }
  const pause = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/pause', {
    onSuccess: () => {
      again()
      toast({ kind: 'success', title: t(m.paused) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const resume = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/resume', {
    onSuccess: () => {
      again()
      toast({ kind: 'success', title: t(m.resumed) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const cancel = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/cancel')
  const left = r.tiles.waiting + r.tiles.held + r.tiles.queued
  return (
    <div className="flex flex-wrap gap-2">
      {r.phase === 'paused' ? (
        <>
          <Button variant="primary" disabled={!r.canResume} loading={resume.isPending} onClick={() => resume.mutate({ params: { path }, body: {} })}>
            {t(m.resume)}
          </Button>
          <ButtonLink to="/w/$workspace/campaigns/$campaignId/sms" params={path}>
            {t(m.resumeChanges)}
          </ButtonLink>
        </>
      ) : (
        <Button disabled={!r.canPause} disabledReason={pauseReason} loading={pause.isPending} onClick={() => pause.mutate({ params: { path }, body: {} })}>
          <Icon name="pause" size={14} />
          {t(m.pause)}
        </Button>
      )}
      <Button variant="danger" disabledReason={pauseReason} onClick={() => setCancelling(true)}>
        {t(m.cancelRest)}
      </Button>
      <ConfirmDialog
        open={cancelling}
        onOpenChange={setCancelling}
        badge={t(m.cancelRest)}
        title={t(m.cancelTitle)}
        description={t(m.cancelText, { count: left })}
        confirmLabel={t(m.cancelRest)}
        onConfirm={async () => {
          await cancel.mutateAsync({ params: { path }, body: {} })
          again()
          toast({ kind: 'success', title: t(m.cancelledRest) })
          setCancelling(false)
        }}
      />
    </div>
  )
}

/** Auto-paused (or paused by a person, or by the budget). */
function Paused({ r }: { r: Report }) {
  const t = useT()
  const p = r.pause!
  const pctText = (v: number) => pct(v)
  return (
    <section className="card flex min-w-0 flex-col" style={{ borderColor: '#FCD34D' }}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{p.kind === 'auto' ? t(m.autoPausedTitle, { metric: p.metric || 'other' }) : p.kind === 'budget' ? t(m.budgetPaused, { held: p.held }) : t(m.pausedBy, { name: p.by ?? '', time: p.at ? clock(p.at) : '' })}</h2>
          {p.kind === 'auto' && p.at ? <span className="k">{t(m.autoPausedAt, { time: utcClock(p.at) })}</span> : null}
        </div>
        <SendControls r={r} />
      </div>
      <div className="pad flex flex-col gap-3">
        {p.kind === 'auto' ? (
          <Note tone="warn" role="note">
            <span className="flex flex-col gap-1">
              <span>{t(m.autoPausedText, { metric: p.metric || 'other', value: pctText(p.value), limit: pctText(p.limit), held: p.held })}</span>
              {p.code ? (
                <span>
                  <T msg={m.autoPausedCode} values={{ provider: p.provider ?? '', code: p.code, count: p.codeCount, base: count(p.base), mono: (c) => <span className="mono">{c}</span> }} />{' '}
                  {p.metric === 'filtered' ? t(m.filteredHint) : null}
                </span>
              ) : null}
            </span>
          </Note>
        ) : null}
        <ZoneRows r={r} />
        <span className="k">{t(m.resumeNote)}</span>
      </div>
    </section>
  )
}

/** A provider outage: what's held, the provider's answer, the retries. */
function Outage({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  const o = r.outage!
  const refresh = useRefreshSms()
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/check', {
    onSuccess: () => void refresh(),
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const region = regionSentence(o.countries)
  return (
    <section className="card flex min-w-0 flex-col" style={{ borderColor: '#FCA5A5' }}>
      <div className="sec">
        <h2 className="h2">{t(m.outageTitle, { provider: o.provider })}</h2>
        <div className="flex flex-wrap gap-2">
          <Button variant="primary" loading={check.isPending} onClick={() => check.mutate({ params: { path: { workspace: slug, providerId: o.providerId } } })}>
            {t(m.retryNow)}
          </Button>
          <SendControls r={r} />
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <Note tone="bad">
          <span className="flex flex-col gap-1">
            <span className="font-semibold">{t(m.outageText, { provider: o.provider, time: o.since ? clock(o.since) : '', held: o.held, region })}</span>
            {o.answer ? <span>{t(m.outageAnswer, { provider: o.provider, answer: o.answer, count: o.failures, fallback: o.hasFallback ? 'yes' : 'no', region })}</span> : null}
          </span>
        </Note>
        <div className="flex flex-col">
          {o.lastAccepted ? <Ev tone="#DCFCE7" title={t(m.outageLast, { provider: o.provider })} at={o.lastAccepted} /> : null}
          {o.since ? <Ev tone="#FEE2E2" title={t(m.outageAnswered, { provider: o.provider, answer: o.answer ?? '' })} text={t(m.outageStop)} at={o.since} /> : null}
          {o.told.length && o.since ? <Ev tone="#DBEAFE" title={t(m.outageTold, { names: formatList(o.told) })} text={t(m.outageToldText)} at={o.since} /> : null}
          {o.nextTry ? <Ev tone="#F3F4F6" title={t(m.outageRetrying)} text={t(m.outageNext, { time: clock(o.nextTry) })} at={o.nextTry} /> : null}
        </div>
        <span className="k">{t(m.outageExpiry, { window: r.window })}</span>
      </div>
    </section>
  )
}

function Ev({ tone, title, text, at }: { tone: string; title: string; text?: ReactNode; at: string }) {
  return (
    <div className="sms-ev">
      <span className="ic" style={{ background: tone }} aria-hidden="true">
        <Icon name="check" size={13} />
      </span>
      <div className="tx">
        <span className="font-semibold">{title}</span>
        {text ? <span className="k">{text}</span> : null}
      </div>
      <span className="tm">{clockSeconds(at)}</span>
    </div>
  )
}

/** The chart's geometry for one size: the line, its area, the grid, the hour ticks and each rise. */
interface ChartShape {
  w: number
  h: number
  x0: number
  x1: number
  base: number
  line: string
  area: string
  grid: { y: number; v: number }[]
  ticks: { x: number; label: string }[]
  marks: { x: number; y: number; label: string }[]
}

function chartShape(r: Report, zone: string, size: { w: number; h: number; x0: number; x1: number; y0: number; y1: number; ticks: number }, label: (s: Report['steps'][number]) => string): ChartShape | null {
  const pts = r.points
  const last = pts[pts.length - 1]
  if (!last) return null
  const t0 = new Date(pts[0]!.at).getTime()
  const t1 = new Date(last.at).getTime()
  const hour = 3_600_000
  const from = Math.floor(t0 / hour) * hour - hour
  const to = Math.ceil(t1 / hour) * hour + hour
  const { x0: X0, x1: X1, y0: Y0, y1: Y1 } = size
  const total = last.delivered
  const step = niceStep(total / 3)
  const top = step * 3
  const x = (ts: number) => X0 + ((ts - from) / (to - from)) * (X1 - X0)
  const y = (v: number) => Y0 - (v / top) * (Y0 - Y1)
  const line: string[] = [`${X0.toFixed(1)},${Y0.toFixed(1)}`]
  let prev = 0
  for (const p of pts) {
    const px = x(new Date(p.at).getTime())
    line.push(`${px.toFixed(1)},${y(prev).toFixed(1)}`, `${(px + 4.5).toFixed(1)},${y(p.delivered).toFixed(1)}`)
    prev = p.delivered
  }
  line.push(`${X1.toFixed(1)},${y(prev).toFixed(1)}`)
  const ticks: { x: number; label: string }[] = []
  const every = Math.max(1, Math.round((to - from) / hour / size.ticks))
  for (let ts = from + hour; ts <= to; ts += hour * every) ticks.push({ x: x(ts), label: inZone(new Date(ts), zone).time })
  const marks = r.steps
    .filter((s) => s.delivered > 0)
    .map((s) => {
      const at = new Date(s.at).getTime()
      const after = pts.filter((p) => new Date(p.at).getTime() <= at + 15 * 60_000).pop()
      // A rise of a few texts (Canada's Atlantic zone) keeps its dot but not a label.
      return { x: x(at) + 4.5, y: y(after?.delivered ?? 0), label: s.delivered * 50 >= total ? label(s) : '' }
    })
  return {
    w: size.w,
    h: size.h,
    x0: X0,
    x1: X1,
    base: size.h - 6,
    line: line.join(' '),
    area: `M${line.join(' L')} L${X1},${Y0} Z`,
    grid: [0, step, step * 2, step * 3].map((v) => ({ y: y(v), v })),
    ticks,
    marks,
  }
}

/** The cumulative line with its rises, as the board draws it; labels only where there's room. */
function ChartSvg({ c, aria, labels }: { c: ChartShape; aria: string; labels?: boolean }) {
  return (
    <svg width="100%" viewBox={`0 0 ${c.w} ${c.h}`} role="img" aria-label={aria} style={{ display: 'block', maxWidth: '100%' }}>
      {c.grid.map((g) => (
        <g key={g.v}>
          <line x1={c.x0} x2={c.x1} y1={g.y} y2={g.y} stroke="#EEF0F3" />
          <text x={c.x0 - 8} y={g.y + 4} textAnchor="end" fontSize="11" fill="#5B6472">
            {count(g.v)}
          </text>
        </g>
      ))}
      <path d={c.area} fill="#0F766E" opacity=".10" />
      <polyline points={c.line} fill="none" stroke="#0F766E" strokeWidth="2" strokeLinejoin="round" />
      {c.marks.map((mk, i) => (
        <g key={i}>
          <circle cx={mk.x} cy={mk.y} r="3" fill="#0F766E" />
          {labels && mk.label ? (
            <text x={mk.x - 2} y={mk.y - 9} textAnchor="middle" fontSize="11" fontWeight="600" fill="#374151">
              {mk.label}
            </text>
          ) : null}
        </g>
      ))}
      {c.ticks.map((tk, i) => (
        <text key={i} x={tk.x} y={c.base} textAnchor={i === c.ticks.length - 1 ? 'end' : 'middle'} fontSize="11" fill="#5B6472">
          {tk.label}
        </text>
      ))}
    </svg>
  )
}

/** Delivery over time: the cumulative count, each time zone's rise (SVG, as the board draws it). */
function OverTime({ r }: { r: Report }) {
  const t = useT()
  const { me } = useWorkspace()
  const [mine, setMine] = useState(false)
  const zone = mine ? r.timezone || me.timezone : 'UTC'
  const charts = useMemo(() => {
    const label = (s: Report['steps'][number]) => (s.region ? t(m.zoneRegion, { region: s.region }) : s.countries.join(', '))
    return {
      wide: chartShape(r, zone, { w: 680, h: 240, x0: 44, x1: 668, y0: 214, y1: 26, ticks: 5 }, label),
      phone: chartShape(r, zone, { w: 340, h: 200, x0: 40, x1: 332, y0: 176, y1: 16, ticks: 3 }, label),
    }
  }, [r, zone, t])
  const steps = r.steps.filter((s) => s.delivered > 0)
  const aria = t(m.chartAria, {
    zone: mine ? zone : 'UTC',
    steps: steps.map((s) => t(m.chartStep, { count: count(s.delivered), time: inZone(s.at, zone).time, region: s.region ? t(m.zoneRegion, { region: s.region }) : regionLabel(s.countries) })).join(', '),
  })
  return (
    <section className="card flex min-w-0 flex-col">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.overTime)}</h2>
          <span className="k">{t(m.overTimeSub, { count: count(r.tiles.delivered) })}</span>
        </div>
        <div className="seg" role="group" aria-label={t(m.time)}>
          <button type="button" aria-pressed={!mine} onClick={() => setMine(false)}>
            {t(m.utc)}
          </button>
          <button type="button" aria-pressed={mine} onClick={() => setMine(true)}>
            {t(m.myTime)}
          </button>
        </div>
      </div>
      <div className="pad flex flex-col gap-2.5">
        {charts.wide && charts.phone ? (
          <>
            <div className="sms-no-phone">
              <ChartSvg c={charts.wide} aria={aria} labels />
            </div>
            <div className="sms-only-phone">
              <ChartSvg c={charts.phone} aria={aria} />
            </div>
          </>
        ) : (
          <span className="k">{t(m.noDeliveries)}</span>
        )}
        {/* Each rise as a list: on phones under the chart (SmsReportPhone), for screen readers everywhere. */}
        <div className="sms-vh-wide flex flex-col gap-1">
          {steps.map((s, i) => (
            <div key={i} className="sms-sum">
              <span>
                <T
                  msg={m.zoneRow}
                  values={{
                    time: mine ? inZone(s.at, zone).time : t(m.utcTime, { time: inZone(s.at, zone).time }),
                    region: s.region ? t(m.zoneRegion, { region: s.region }) : regionSentence(s.countries),
                    mono: (c) => <span className="mono k">{c}</span>,
                  }}
                />
              </span>
              <span className="n">+{count(s.delivered)}</span>
            </div>
          ))}
        </div>
        <span className="k">{r.localTime ? t(m.chartNote, { local: r.localTime, zone: mine ? zone : 'UTC' }) : t(m.chartNoteNow, { zone: mine ? zone : 'UTC' })}</span>
      </div>
    </section>
  )
}

/** 300 for 280, 500 for 410: a round step for the grid. */
function niceStep(v: number): number {
  if (v <= 0) return 1
  const p = 10 ** Math.floor(Math.log10(v))
  for (const k of [1, 2, 3, 5, 10]) if (k * p >= v) return k * p
  return 10 * p
}

function BySender({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  const kindName = useKindName()
  const order = r.senders.map((s) => s.provider)
  const duration = (secs: number) => (secs < 60 ? t(m.seconds) : t(m.underMinutes, { n: Math.ceil(secs / 60) }))
  return (
    <section className="card flex min-w-0 flex-col">
      <div className="sec">
        <h2 className="h2">{t(m.bySender)}</h2>
        <span className="k">{t(m.cost)}</span>
      </div>
      {r.senders.map((s) => (
        <div key={s.provider + s.sender} className="flex flex-col gap-2 border-t border-line-2 px-[18px] py-3.5">
          <div className="sms-sum">
            <span className="inline-flex items-center gap-2 font-semibold">
              <span className="dot" style={{ color: providerColor(order, s.provider) }} aria-hidden="true" />
              {s.provider} · <span className={s.sender.startsWith('+') ? 'mono' : undefined}>{s.sender}</span>
            </span>
            <span className="n">{money(s.cost)}</span>
          </div>
          <CountryChips codes={s.countries} />
          <div className="grid grid-cols-3 gap-2 text-[13px]">
            <div>
              <div className="k">{t(m.sent)}</div>
              <div className="n">{count(s.sent)}</div>
            </div>
            <div>
              <div className="k">{t(m.delivered)}</div>
              <div className="n">
                {count(s.delivered)} <span className="k">{pct(s.deliveredPct)}</span>
              </div>
            </div>
            <div>
              <div className="k">{t(m.failed)}</div>
              <div className="n">{count(s.failed)}</div>
            </div>
          </div>
          <span className="k">{t(m.senderFoot, { kind: kindName(s.senderKind), rate: s.throughput, duration: duration(s.zoneSeconds) })}</span>
        </div>
      ))}
      <div className="sms-foot">
        <span className="k">{r.receiptsWithin ? t(m.receiptsWithin, { seconds: r.receiptsWithin }) : t(m.receiptsSome)}</span>
        <Link to="/w/$workspace/providers/sms/events" params={{ workspace: slug }} className="sms-lnk">
          {t(m.rawEvents)}
        </Link>
      </div>
    </section>
  )
}

function ByCountry({ r }: { r: Report }) {
  const t = useT()
  const sent = r.countries.reduce((n, c) => n + c.sent, 0)
  const delivered = r.countries.reduce((n, c) => n + c.delivered, 0)
  const failed = r.countries.reduce((n, c) => n + c.failed, 0)
  const clicks = r.countries.reduce((n, c) => n + c.clicks, 0)
  const optOuts = r.countries.reduce((n, c) => n + c.optOuts, 0)
  const cost = r.countries.reduce((n, c) => n + c.cost, 0)
  const sub = (v: string) => <span className="k ml-1">{v}</span>
  return (
    <section className="card min-w-0">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.byCountry)}</h2>
          <span className="k">{t(m.byCountryLead)}</span>
        </div>
      </div>
      <div className="sms-no-phone flex-col">
        <Table caption={t(m.countryTable)} className="sms-stk">
          <thead>
            <tr>
              <Th>{t(m.colCountry)}</Th>
              <Th>{t(m.colSender)}</Th>
              <Th className="r">
                {t(m.colSent)}
              </Th>
              <Th className="r">
                {t(m.colDelivered)}
              </Th>
              <Th className="r">
                {t(m.colFailed)}
              </Th>
              <Th className="r">
                {t(m.colClicks)}
              </Th>
              <Th className="r">
                {t(m.colOptOuts)}
              </Th>
              <Th className="r">
                {t(m.colCost)}
              </Th>
            </tr>
          </thead>
          <tbody>
            {r.countries.map((c) =>
              c.skipped && !c.sent ? (
                <tr key={c.country} style={{ background: '#FFFBEB' }}>
                  <td className="nw">
                    <CountryName code={c.country} />
                  </td>
                  <td className="nw">{c.sender}</td>
                  <td colSpan={5}>
                    <span className="flex flex-wrap items-center gap-2">
                      <Badge tone="warn">{t(m.skipped, { count: count(c.skipped) })}</Badge>
                      <span className="k">{t(m.skippedWhy, { sender: c.sender, country: countryName(c.country), count: c.skipped })}</span>
                    </span>
                  </td>
                  <td className="r num">{money(0)}</td>
                </tr>
              ) : (
                <tr key={c.country}>
                  <td className="nw">
                    <CountryName code={c.country} />
                  </td>
                  <td className={`nw${c.sender.startsWith('+') ? ' mono' : ''}`}>{c.sender}</td>
                  <td className="r num">{count(c.sent)}</td>
                  <td className="r num nw">
                    {count(c.delivered)}
                    {sub(pct(c.deliveredPct))}
                  </td>
                  <td className="r num">{count(c.failed)}</td>
                  <td className="r num nw">
                    {count(c.clicks)}
                    {sub(pct(c.clickPct))}
                  </td>
                  <td className="r num">{count(c.optOuts)}</td>
                  <td className="r num">{money(c.cost)}</td>
                </tr>
              ),
            )}
            <tr>
              <td className="font-semibold" style={{ borderTop: '1px solid var(--line)' }}>
                {t(m.total)}
              </td>
              <td style={{ borderTop: '1px solid var(--line)' }} />
              <td className="r num font-semibold" style={{ borderTop: '1px solid var(--line)' }}>
                {count(sent)}
              </td>
              <td className="r num nw font-semibold" style={{ borderTop: '1px solid var(--line)' }}>
                {count(delivered)}
                {sub(pct(sent ? (delivered / sent) * 100 : 0))}
              </td>
              <td className="r num font-semibold" style={{ borderTop: '1px solid var(--line)' }}>
                {count(failed)}
              </td>
              <td className="r num nw font-semibold" style={{ borderTop: '1px solid var(--line)' }}>
                {count(clicks)}
                {sub(pct(delivered ? (clicks / delivered) * 100 : 0))}
              </td>
              <td className="r num nw font-semibold" style={{ borderTop: '1px solid var(--line)' }}>
                {count(optOuts)}
                {sub(pct(sent ? (optOuts / sent) * 100 : 0))}
              </td>
              <td className="r num font-semibold" style={{ borderTop: '1px solid var(--line)' }}>
                {money(cost)}
              </td>
            </tr>
          </tbody>
        </Table>
      </div>
      <div className="sms-only-phone">
        {r.countries.map((c) => (
          <div key={c.country} className="flex flex-col gap-1 border-t border-line-2 px-4 py-3" style={c.skipped && !c.sent ? { background: '#FFFBEB' } : undefined}>
            <span className="flex items-center justify-between gap-2">
              <CountryName code={c.country} />
              {c.skipped && !c.sent ? <Badge tone="warn">{t(m.skipped, { count: count(c.skipped) })}</Badge> : <span className="n">{money(c.cost)}</span>}
            </span>
            <span className="k">
              {c.skipped && !c.sent
                ? t(m.phoneSkipped, { count: count(c.skipped), sender: c.sender, country: countryName(c.country) })
                : t(m.phoneLine, { sent: count(c.sent), delivered: count(c.delivered), failed: count(c.failed), clicks: c.clicks, optOuts: c.optOuts })}
            </span>
          </div>
        ))}
        <div className="flex items-center justify-between gap-2 border-t border-line px-4 py-3">
          <span className="font-semibold">{t(m.phoneTotal, { sent: count(sent), delivered: count(delivered) })}</span>
          <span className="n">{money(cost)}</span>
        </div>
      </div>
    </section>
  )
}

function Failures({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  const total = r.failures.reduce((n, f) => n + f.count, 0)
  return (
    <section className="card flex min-w-0 flex-col">
      <div className="sec">
        <h2 className="h2">{t(m.failures, { count: total })}</h2>
      </div>
      {r.failures.length === 0 ? <div className="pad k">{t(m.noFailures)}</div> : null}
      {r.failures.map((f) => {
        const text =
          f.reason === 'unreachable'
            ? t(m.failTemp, { rate: pct(f.rate), limit: pct(f.limit, 0) })
            : f.reason === 'filtered'
              ? t(m.failFiltered, { rate: pct(f.rate), limit: pct(f.limit, 0) })
              : f.reason === 'not_in_service'
                ? t(m.failNotInService, { count: f.count })
                : f.reason === 'landline'
                  ? t(m.failLandline, { count: f.count })
                  : f.reason === 'invalid'
                    ? t(m.failInvalid, { count: f.count })
                    : t(m.failOther, { rate: pct(f.rate), limit: pct(f.limit, 0) })
        return (
          <div key={f.reason} className="sms-srow">
            <b className="n w-7 flex-none text-[16px]">{count(f.count)}</b>
            <div className="tx">
              <span className="flex flex-wrap items-center gap-2">
                <Badge tone={f.reason === 'not_in_service' || f.reason === 'invalid' || f.reason === 'landline' ? 'bad' : 'warn'}>
                  {reasonWord[f.reason] ? t(reasonWord[f.reason]!) : f.reason}
                </Badge>
                <span className="k">
                  {f.codes.map((c, i) => (
                    <span key={c.provider + c.code}>
                      {i > 0 ? '\u00a0 ' : null}
                      {c.provider} {/^\d+$/.test(c.code) ? <span className="mono">{c.code}</span> : t(m.quotedCode, { code: c.code })} · {count(c.count)}
                    </span>
                  ))}
                </span>
              </span>
              <span className="text-[13px] text-ink-nav">{text}</span>
            </div>
          </div>
        )
      })}
      <div className="sms-foot">
        <span className="k">{t(m.failCodes)}</span>
        <Link to="/w/$workspace/providers/sms/events" params={{ workspace: slug }} search={{ kind: 'failures' }} className="sms-lnk">
          {t(m.openEvents)}
        </Link>
      </div>
    </section>
  )
}

function Clicks({ r }: { r: Report }) {
  const t = useT()
  const people = r.tiles.clicks
  return (
    <section className="card flex min-w-0 flex-col">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.clicks, { count: count(people) })}</h2>
          <span className="k">{t(m.clicksLead)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        {r.links.length === 0 ? <span className="k">{t(m.noClicks)}</span> : null}
        {r.links.map((l) => {
          const all = l.people + l.machine
          return (
            <div key={l.url} className="flex flex-col gap-2">
              <div className="sms-lbox">
                <Icon name="external" size={14} className="mt-0.5 flex-none text-accent" />
                <span className="flex min-w-0 flex-1 flex-col gap-[3px]">
                  <span className="mono text-[12.5px] [overflow-wrap:anywhere]">{l.url}</span>
                  <span className="k">{t(m.viaDomain, { domain: l.domain })}</span>
                </span>
                <span className="flex flex-none flex-col items-end">
                  <span className="n">{t(m.people, { count: count(l.people) })}</span>
                  <span className="k">{t(m.machine, { count: count(l.machine) })}</span>
                </span>
              </div>
              <div className="bar stack" style={{ height: 8 }}>
                <i style={{ width: `${pctOf(l.people, all)}%`, background: 'var(--accent)' }} />
                <i style={{ width: `${pctOf(l.machine, all)}%`, background: 'var(--line-input)' }} />
              </div>
              <div className="sms-bm-l">
                <span>
                  <span className="sms-bsw" style={{ background: 'var(--accent)' }} />
                  {t(m.peopleLabel, { count: count(l.people) })}
                </span>
                <span>
                  <span className="sms-bsw" style={{ background: 'var(--line-input)' }} />
                  {t(m.machineLabel, { count: count(l.machine) })}
                </span>
              </div>
            </div>
          )
        })}
        {r.links.length ? (
          <div className="flex flex-col gap-1.5">
            <span className="text-[13px] font-semibold">{t(m.clickRate)}</span>
            {r.countries
              .filter((c) => c.sent > 0)
              .map((c) => (
                <div key={c.country} className="flex flex-col gap-1">
                  <div className="sms-sum items-center">
                    <span className="inline-flex items-center gap-2">
                      <CountryChips codes={[c.country]} />
                      {countryName(c.country)}
                    </span>
                    <span className="flex items-center gap-2">
                      <span className="n">{pct(c.clickPct)}</span>
                      <span className="k">{t(m.countryPeople, { count: c.clicks })}</span>
                    </span>
                  </div>
                  {/* The board's scale: half the bar is 25%. */}
                  <div className="bar" aria-hidden="true">
                    <i style={{ width: `${Math.min(100, c.clickPct * 2).toFixed(1)}%`, background: 'var(--accent)' }} />
                  </div>
                </div>
              ))}
          </div>
        ) : null}
      </div>
    </section>
  )
}

type ReplyKind = 'all' | 'opt_out' | 'help' | 'other'

function Replies({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [kind, setKind] = useState<ReplyKind>('all')
  const [all, setAll] = useState(false)
  const counts = r.replyCounts
  const total = Object.values(counts).reduce((n, v) => n + v, 0)
  const other = total - (counts.opt_out ?? 0) - (counts.help ?? 0)
  const full = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/replies',
    { params: { path: { workspace: slug, campaignId: r.campaignId }, query: { kind, limit: 200 } } },
    { enabled: all || kind !== 'all' },
  )
  const items = all || kind !== 'all' ? (full.data?.items ?? []) : r.replies
  const chips: { k: ReplyKind; label: string; n: number }[] = [
    { k: 'all', label: t(m.rAll), n: total },
    { k: 'opt_out', label: t(m.rStop), n: counts.opt_out ?? 0 },
    { k: 'help', label: t(m.rHelp), n: counts.help ?? 0 },
    { k: 'other', label: t(m.rOther), n: other },
  ]
  return (
    <section className="card flex min-w-0 flex-col">
      <div className="sec">
        <h2 className="h2">{t(m.replies, { count: total })}</h2>
      </div>
      <div className="sms-fchips" role="group" aria-label={t(m.replies, { count: total })}>
        {chips.map((c) => (
          <button key={c.k} type="button" className={`sms-fchip${kind === c.k ? ' on' : ''}`} aria-pressed={kind === c.k} onClick={() => setKind(c.k)}>
            {c.label} <b>{count(c.n)}</b>
          </button>
        ))}
      </div>
      {items.length === 0 ? <div className="pad k">{t(m.noReplies)}</div> : null}
      {items.map((x) => (
        <ReplyRow key={x.id} x={x} />
      ))}
      <div className="sms-foot">
        <span className="k">{t(m.showing, { shown: items.length, total })}</span>
        {!all && items.length < total ? (
          <Button size="sm" onClick={() => setAll(true)}>
            {t(m.showAll, { count: total })}
          </Button>
        ) : null}
      </div>
    </section>
  )
}

function ReplyRow({ x }: { x: Schemas['SmsReply'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const zone = x.zone.split('/').pop()?.replace(/_/g, ' ') ?? x.zone
  const lists = formatList(x.lists)
  const text =
    x.kind === 'opt_out'
      ? x.anyCase
        ? t(m.rOptOutAnyCase, { zone })
        : t(m.rOptOut, { lists, zone })
      : x.kind === 'opt_in'
        ? t(m.rOptIn, { lists, zone })
        : x.kind === 'help'
          ? t(m.rHelpText, { chars: x.replyChars, segments: x.replySegments, zone })
          : x.replied
            ? t(m.rReply, { zone })
            : t(m.rReplyNoAnswer, { zone })
  const badge = { opt_out: { tone: 'bad', text: t(m.bStop) }, help: { tone: 'info', text: t(m.bHelp) }, opt_in: { tone: 'ok', text: t(m.bStart) }, yes: { tone: 'ok', text: t(m.bYes) }, reply: { tone: 'neutral', text: t(m.bReply) } }[x.kind] as { tone: Tone; text: string }
  return (
    <div className="sms-srow">
      <span className="mono k flex-none pt-0.5">{clock(x.at)}</span>
      <div className="tx">
        <span className="flex flex-wrap items-baseline gap-2">
          {x.contactId ? (
            <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: x.contactId }} className="font-semibold text-ink no-underline hover:underline">
              {x.contact}
            </Link>
          ) : (
            <span className="font-semibold">{x.contact}</span>
          )}
          <span className="sms-mask">{x.number}</span>
        </span>
        <span>“{x.body}”</span>
        <span className="k">{text}</span>
      </div>
      <div className="ct">
        <Badge tone={badge.tone}>{badge.text}</Badge>
      </div>
    </div>
  )
}

/** One line of Held and skipped: the count, what it is and why. */
function Row({ n, title, tag, text }: { n: number; title: string; tag?: ReactNode; text: ReactNode }) {
  return (
    <div className="sms-srow">
      <b className="n w-12 flex-none text-[17px]">{count(n)}</b>
      <div className="tx">
        <span className="flex flex-wrap items-center gap-2">
          <span className="font-semibold">{title}</span>
          {tag}
        </span>
        <span className="k">{text}</span>
      </div>
    </div>
  )
}

/** Why texts were cancelled before they went. */
const notSentWord: Record<string, MessageDescriptor> = {
  cancelled: m.notCancelled,
  provider_down: m.notProviderDown,
  pending_sender: m.notPendingSender,
  sender_removed: m.notSenderRemoved,
  provider_removed: m.notProviderRemoved,
}

function Held({ r }: { r: Report }) {
  const t = useT()
  const h = r.held
  const notSent = Object.entries(h.notSent).filter(([, n]) => n > 0)
  const notSentTotal = notSent.reduce((n, [, v]) => n + v, 0)
  return (
    <section className="card flex min-w-0 flex-col">
      <div className="sec">
        <h2 className="h2">{t(m.heldTitle)}</h2>
      </div>
      {h.skipped ? (
        <Row
          n={h.skipped}
          title={t(m.hSkipped)}
          tag={<Badge tone="warn">{regionLabel(h.skippedCountries)}</Badge>}
          text={t(m.hSkippedText, { sender: h.skipSender ?? '', country: regionSentence(h.skippedCountries) })}
        />
      ) : null}
      <Row
        n={h.quietHours}
        title={t(m.hQuiet)}
        tag={h.quietHours ? null : <Badge tone="ok">{t(m.hQuietNone)}</Badge>}
        text={h.quietHours ? t(m.hQuietText, { count: h.quietHours, window: r.window }) : t(m.hQuietNoneText, { local: r.localTime ?? '', window: r.window })}
      />
      <Row
        n={h.noConsent + h.suppressed}
        title={t(m.hLeftOut)}
        tag={<Badge tone="neutral">{t(m.hAudience)}</Badge>}
        text={t(m.hLeftOutText, { noConsent: count(h.noConsent), lists: formatList(h.lists), suppressed: h.suppressed })}
      />
      {notSentTotal ? (
        <Row
          n={notSentTotal}
          title={t(m.hNotSent)}
          text={t(m.hNotSentText, { count: notSentTotal, reasons: formatList(notSent.map(([reason, n]) => t(notSentWord[reason] ?? m.notOther, { count: n }))) })}
        />
      ) : null}
    </section>
  )
}

/** One recipient's timeline: the text, the click, the reply. */
function Sample({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  const s = r.sample!
  return (
    <section className="card flex min-w-0 flex-col">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.onTimeline, { name: s.name })}</h2>
          <span className="k">{t(m.onTimelineLead)}</span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: s.contactId }}>
          {t(m.openContact)}
        </ButtonLink>
      </div>
      <div className="px-[18px] py-1">
        {s.events.map((e, i) => {
          if (e.kind === 'delivered') {
            return (
              <Ev
                key={i}
                tone="#DCFCE7"
                title={t(m.evDelivered, { name: r.name })}
                text={<T msg={m.evDeliveredText} values={{ provider: e.provider ?? '', sender: e.sender ?? '', seconds: e.seconds ?? 0, mono: (c) => <span className="mono">{c}</span> }} />}
                at={e.at}
              />
            )
          }
          if (e.kind === 'clicked') return <Ev key={i} tone="#DBEAFE" title={t(m.evClicked, { link: e.link ?? '' })} text={t(m.evClickedText, { target: e.target ?? '' })} at={e.at} />
          if (e.kind === 'replied') return <Ev key={i} tone="#F3F4F6" title={t(m.evReplied, { body: e.body ?? '' })} text={t(m.evRepliedText)} at={e.at} />
          return <Ev key={i} tone="#F3F4F6" title={t(m.evOther, { kind: e.kind })} at={e.at} />
        })}
      </div>
    </section>
  )
}

