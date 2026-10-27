import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Grid } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { useNow } from '@/lib/hooks'
import { andList, clock, clockSeconds, count, duration, pct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { failoverText, Fact, sinceText, streamName, type ProviderCard, type Stream } from './common'
import { mi } from './incidents.messages'
import { m } from './messages'
import { mo } from './overview.messages'
import { mr } from './routing.messages'

type Outage = Schemas['ProvidersOverview']['outage'] & {}
type OutageStream = Schemas['ProviderOutageStream']
type Silent = Schemas['ProvidersOverview']['silent'] & {}
type AuthFailure = Schemas['ProvidersOverview']['authFailure'] & {}

const providersKey = ['get', '/api/v1/workspaces/{workspace}/providers'] as const

/** Invalidates the providers screen after an action. */
export function useRefreshProviders() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: providersKey })
}

/** The round alert mark of the incident banners. */
function AlertMark({ fill, stroke }: { fill: string; stroke: string }) {
  return (
    <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
      <circle cx="20" cy="20" r="19" fill={fill} />
      <path d="M20 11v11M20 27v1" stroke={stroke} strokeWidth="3" strokeLinecap="round" />
    </svg>
  )
}

/** Re-checks a provider now (health probe) and refreshes the screen. */
export function RecheckButton({ providerId, label, variant }: { providerId: string; label: string; variant?: 'primary' }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshProviders()
  const reason = usePermissionReason('providers:manage')
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/check', {
    onSuccess: (card) => {
      void refresh()
      const down = card.status === 'down'
      toast({
        kind: down ? 'error' : 'success',
        title: down ? t(mi.stillDown, { name: card.name }) : t(mi.answered, { name: card.name }),
        detail: down ? t(mi.stillDownDetail) : t(mi.refreshed),
      })
    },
    onError: (e) => toast({ kind: 'error', title: t(mi.checkFailed), detail: errorMessage(e) }),
  })
  return (
    <Button variant={variant} disabledReason={reason} loading={check.isPending} onClick={() => check.mutate({ params: { path: { workspace: slug, providerId } } })}>
      {label}
    </Button>
  )
}

/** "SendEmail returns 503 ServiceUnavailable and ThrottlingException on 96% of calls." */
function outageSentence(o: Outage): string {
  const parts: string[] = []
  if (o.codes.length) {
    parts.push(
      tt(mi.outageReturns, {
        hasOp: o.operation ? 'yes' : 'no',
        op: o.operation,
        codes: andList(o.codes),
        hasRate: o.errorRate !== undefined ? 'yes' : 'no',
        rate: o.errorRate !== undefined ? pct(Math.round(o.errorRate), 0) : '',
      }),
    )
  }
  if (o.down.statusSummary) parts.push(o.down.statusSummary)
  const handled = o.streams.filter((s) => s.state === 'failover_active' || s.state === 'fallback_unavailable').map((s) => streamName[s.stream as Stream])
  const waitingStreams = o.streams.filter((s) => s.state === 'waiting' || s.state === 'held')
  const waiting = waitingStreams.map((s) => streamName[s.stream as Stream])
  // "Broadcasts are waiting", "System is waiting": the stream names that read as plurals.
  const verb = waitingStreams.length > 1 || waitingStreams.some((s) => s.stream === 'broadcast' || s.stream === 'automation') ? 'are' : 'is'
  if (handled.length && waiting.length) parts.push(tt(mi.outageHandledWaiting, { handled: andList(handled), waiting: andList(waiting), verb }))
  else if (handled.length) parts.push(tt(mi.outageHandled, { handled: andList(handled) }))
  else if (waiting.length) parts.push(tt(mi.outageWaiting, { waiting: andList(waiting), verb }))
  return parts.join(' ')
}

/** Outage banner: what is down, and how each stream copes. */
export function OutageAlert({ outage }: { outage: Outage }) {
  const t = useT()
  // Streams that handled themselves first; the ones waiting for someone last.
  const waits = (s: OutageStream) => (s.state === 'waiting' || s.state === 'held' ? 1 : 0)
  const columns = [...outage.streams].sort((a, b) => waits(a) - waits(b))
  return (
    <section role="alert" className="card flex flex-col border-bad-line">
      <div className="flex flex-wrap items-start gap-[18px] rounded-t-[10px] bg-[#FEF2F2] px-[22px] py-5">
        <AlertMark fill="#FECACA" stroke="#991B1B" />
        <div className="flex flex-[1_1_320px] flex-col gap-1">
          <span className="text-[20px] font-bold tracking-[-0.01em] text-[#7F1D1D]">
            {t(mi.outageTitle, { provider: outage.provider, time: clock(outage.since), since: sinceText(outage.since) })}
          </span>
          <span className="text-[#7F1D1D]">{outageSentence(outage)}</span>
        </div>
        <div className="flex flex-wrap gap-2.5">
          {outage.down.statusUrl ? <ButtonAnchor href={outage.down.statusUrl}>{outage.down.statusLabel || t(mi.statusPage, { provider: outage.short })}</ButtonAnchor> : null}
          <RecheckButton providerId={outage.providerId} label={t(mi.recheckNow, { provider: outage.short })} />
        </div>
      </div>
      <Grid min={300} gap={0}>
        {columns.map((s, i) => (
          <OutageColumn key={s.stream} s={s} outage={outage} last={i === columns.length - 1} />
        ))}
      </Grid>
      {columns
        .filter((s) => s.state === 'waiting' && s.campaign && !s.campaign.decision)
        .map((s) => (
          <HeldDecision key={s.stream} s={s} outage={outage} />
        ))}
    </section>
  )
}

function streamBadge(s: OutageStream): { tone: Tone; text: string } {
  switch (s.state) {
    case 'failover_active':
      return { tone: 'ok', text: tt(mi.failoverActive) }
    case 'fallback_unavailable':
      return { tone: 'warn', text: tt(mi.fallbackUnavailable) }
    case 'waiting':
      return { tone: 'warn', text: tt(mi.waitingDecision) }
    case 'held':
      return { tone: 'warn', text: tt(mi.held) }
  }
}

/** "211 welcome series, 97 onboarding" as the board writes it: "welcome series 211, onboarding 97". */
function reroutedList(s: OutageStream) {
  return s.reroutedBy.map((r) => `${r.name} ${count(r.count)}`).join(', ')
}

function OutageColumn({ s, outage, last }: { s: OutageStream; outage: Outage; last: boolean }) {
  const t = useT()
  const b = streamBadge(s)
  const name = streamName[s.stream as Stream]
  let body: ReactNode
  let note: ReactNode = null
  if (s.state === 'waiting' && s.campaign) {
    body = <HeldCampaign s={s} outage={outage} />
  } else if (s.role === 'fallback') {
    const sent = s.reroutedBy.length
      ? t(mi.sentDirectly, { list: andList(s.reroutedBy.map((r) => `${count(r.count)} ${r.name}`)), time: clock(s.since), primary: s.primary ?? '' })
      : t(mi.nothingSent, { stream: name, time: clock(s.since) })
    body = (
      <span className="text-[13px]">
        {t(mi.fallbackBody, { primary: s.primary ?? '', unavailable: s.state === 'fallback_unavailable' ? 'yes' : 'no', provider: outage.short })} {sent}
      </span>
    )
    if (s.suggestion) note = <span className="k">{t(mi.suggestion, { provider: s.suggestion, stream: name })}</span>
  } else if (s.state === 'failover_active') {
    body = (
      <span className="text-[13px]">
        {t(mi.failoverBody, {
          provider: outage.short,
          fallback: s.fallbackShort ?? '',
          time: clock(s.since),
          count: s.rerouted,
          hasList: s.reroutedBy.length ? 'yes' : 'no',
          list: reroutedList(s),
        })}
      </span>
    )
    note = <span className="k">{t(mi.switchBack, { provider: outage.short, minutes: s.switchBackMinutes })}</span>
  } else {
    body = (
      <span className="text-[13px]">
        {s.policy !== 'automatic'
          ? t(mi.heldBodyPolicy, { stream: name, provider: outage.short, policy: failoverText[s.policy] })
          : t(mi.heldBody, { stream: name, provider: outage.short })}
      </span>
    )
  }
  return (
    <div className={cn('flex flex-col gap-2 px-[22px] py-[18px]', !last && 'border-r border-line-2', s.state === 'waiting' && 'bg-warn-tint')}>
      <div className="flex items-center justify-between">
        <span className="font-semibold">{name}</span>
        <Badge tone={b.tone} dot>
          {b.text}
        </Badge>
      </div>
      {body}
      {note}
    </div>
  )
}

/** A broadcast whose share on the down provider waits for a decision: where it stands, or what was decided. */
function HeldCampaign({ s, outage }: { s: OutageStream; outage: Outage }) {
  const t = useT()
  const { slug } = useWorkspace()
  const c = s.campaign!
  if (c.decision) {
    return (
      <span className="text-[13px]">
        {c.decision === 'wait'
          ? t(mi.decidedWait, { name: c.name, provider: outage.short, minutes: s.switchBackMinutes })
          : c.decision === 'move'
            ? t(mi.decidedMove, { name: c.name })
            : t(mi.decidedCancel, { name: c.name })}{' '}
        <Link to="/w/$workspace/campaigns/$campaignId/live" params={{ workspace: slug, campaignId: c.id }}>
          {t(mi.openLive)}
        </Link>
      </span>
    )
  }
  const through = c.planned > 0 ? Math.round((c.sent / c.planned) * 100) : 0
  const others = c.otherShares.map((o) => `${o.short} (${pct(o.percent, 0)})`)
  const policy = s.policy === 'warm_only' ? t(mi.policyWarmOnly) : t(mi.policyPause)
  return (
    <>
      <span className="text-[13px]">
        {t(mi.heldLine, { name: c.name, through: pct(through, 0), provider: outage.short, sent: count(c.sent), held: count(c.held) })}
        {others.length ? ` ${t(mi.othersKeep, { count: others.length, others: andList(others) })}` : ''} {policy}
      </span>
      <span className="k">{t(mi.chooseBelow)}</span>
    </>
  )
}

/**
 * What happens to a held share (ProvidersOutage): wait for the provider, or
 * move to a target whose checks (review GS-5, PR-7: state.Vet) are listed
 * under its button, or cancel. A target that failed a check can't be chosen.
 */
function HeldDecision({ s, outage }: { s: OutageStream; outage: Outage }) {
  const t = useT()
  const id = useId()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshProviders()
  const c = s.campaign!
  const sendReason = usePermissionReason('campaigns:send')
  const pauseReason = usePermissionReason('campaigns:pause')
  const [confirm, setConfirm] = useState<null | { action: 'move' | 'cancel'; target?: Schemas['ProviderFailoverAlternative'] }>(null)
  const decide = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/held/{campaignId}', {
    onSuccess: (res, vars) => {
      void refresh()
      setConfirm(null)
      const action = vars.body.action
      if (action === 'wait') {
        toast({ kind: 'success', title: t(mi.waitingFor, { provider: outage.short }), detail: t(mi.waitingForDetail, { name: c.name, provider: outage.short }) })
        void navigate({ to: '/w/$workspace/campaigns/$campaignId/live', params: { workspace: slug, campaignId: c.id } })
      } else if (action === 'move') {
        const target = vars.body.targetProviderId ? s.alternatives.find((a) => a.providerId === vars.body.targetProviderId)?.short : undefined
        toast({
          kind: 'success',
          title: t(mi.moved, { count: res.moved }),
          detail: target ? t(mi.movedDetail, { name: c.name, provider: target }) : t(mi.movedDetailFallback, { name: c.name }),
        })
      } else {
        toast({ kind: 'deleted', title: t(mi.cancelled, { name: c.name }), detail: t(mi.cancelledDetail, { count: c.held }) })
      }
    },
    onError: (e) => toast({ kind: 'error', title: t(mi.decisionFailed), detail: errorMessage(e) }),
  })
  const run = (action: 'wait' | 'move' | 'cancel', target?: string) =>
    decide.mutate({ params: { path: { workspace: slug, providerId: outage.providerId, campaignId: c.id } }, body: { action, targetProviderId: target } })
  // Waiting is the recommendation unless a target passed every check.
  const moveRecommended = s.alternatives.some((a) => a.recommended)
  const target = confirm?.target?.short ?? ''
  return (
    <div className="flex flex-col gap-3 rounded-b-[10px] border-t border-line-2 bg-warn-tint px-[22px] pt-4 pb-[18px]">
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1.5">
        <h2 className="h2" id={id}>
          {t(mi.heldTitle, { count: c.held, stream: s.stream })}
        </h2>
        <span className="k">{t(mi.checksFirst)}</span>
      </div>
      <div className="pv-dec pv-decide" role="group" aria-labelledby={id}>
        <div className="pv-dopt">
          <Button size="sm" variant="primary" disabledReason={sendReason} loading={decide.isPending && decide.variables?.body.action === 'wait'} onClick={() => run('wait')}>
            {t(mi.waitAutoResume, { provider: outage.short })}
          </Button>
          <span className="k">{t(mi.waitNote, { recommended: moveRecommended ? 'no' : 'yes', minutes: s.switchBackMinutes, provider: outage.short })}</span>
        </div>
        {s.alternatives.map((a) => (
          <MoveOption key={a.providerId} a={a} reason={sendReason} onMove={() => setConfirm({ action: 'move', target: a })} />
        ))}
      </div>
      <Button size="sm" variant="danger" className="self-start" disabledReason={pauseReason} onClick={() => setConfirm({ action: 'cancel' })}>
        {t(mi.cancelSend)}
      </Button>
      <Dialog
        open={confirm?.action === 'move'}
        onOpenChange={(o) => !o && setConfirm(null)}
        title={t(mi.moveTitle, { count: c.held, provider: target })}
        description={
          confirm?.target?.exceedsCap
            ? t(mi.moveOverCap, { provider: confirm.target.short, current: outage.short })
            : t(mi.moveBody, { name: c.name, provider: target })
        }
        footer={
          <>
            <Button onClick={() => setConfirm(null)}>{t(mi.keepWaiting)}</Button>
            <Button variant={confirm?.target?.exceedsCap ? 'danger-primary' : 'primary'} loading={decide.isPending} onClick={() => run('move', confirm?.target?.providerId)}>
              {t(mi.moveTo, { provider: target })}
            </Button>
          </>
        }
      />
      <Dialog
        open={confirm?.action === 'cancel'}
        onOpenChange={(o) => !o && setConfirm(null)}
        dismissible={false}
        title={t(mi.cancelTitle, { name: c.name })}
        description={t(mi.cancelBody, { held: count(c.held), sent: count(c.sent) })}
        footer={
          <>
            <Button onClick={() => setConfirm(null)}>{t(mi.keepWaiting)}</Button>
            <Button variant="danger-primary" loading={decide.isPending} onClick={() => run('cancel')}>
              {t(mi.cancelSend)}
            </Button>
          </>
        }
      />
    </div>
  )
}

/** One target to move the held share to, with what ReputeCast checked about it. */
function MoveOption({ a, reason, onMove }: { a: Schemas['ProviderFailoverAlternative']; reason?: string; onMove: () => void }) {
  const t = useT()
  const id = useId()
  return (
    <div className="pv-dopt">
      <Button size="sm" aria-describedby={id} disabledReason={reason ?? (a.allowed ? undefined : t(mi.moveRefused, { provider: a.short }))} onClick={onMove}>
        {a.exceedsCap ? t(mi.moveToOverCap, { provider: a.short }) : t(mi.moveTo, { provider: a.short })}
      </Button>
      {a.recommended ? <span className="k">{t(mi.moveRecommended)}</span> : null}
      <ul className="pv-chk col" id={id} aria-label={t(mi.checksOn, { provider: a.short })}>
        {shownChecks(a.checks).map((c) => (
          <li key={c.key} className={vetMark[c.result as VetResult] ?? 'u'}>
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              {c.result === 'pass' ? <path d="m3.5 8.5 3 3 6-7" /> : c.result === 'fail' ? <path d="m4.5 4.5 7 7M11.5 4.5l-7 7" /> : <path d="M4 8h8" />}
            </svg>
            {vetLine(c)}
          </li>
        ))}
      </ul>
    </div>
  )
}

/**
 * As the board lists a target: its domain, quota and warm-up always, then
 * any other check that didn't pass (available, bulk terms, sandbox).
 */
const vetOrder = ['domain', 'quota', 'warm', 'bulk_terms', 'available', 'sandbox']
function shownChecks(checks: Schemas['VetCheck'][]) {
  return checks.filter((c) => vetOrder.indexOf(c.key) < 3 || c.result !== 'pass').sort((a, b) => vetOrder.indexOf(a.key) - vetOrder.indexOf(b.key))
}

type VetResult = 'pass' | 'warn' | 'fail' | 'unknown'
const vetMark: Record<VetResult, string> = { pass: 'y', warn: 'w', fail: 'x', unknown: 'u' }

/**
 * A check's line: the server's sentence when the check didn't pass (in the
 * reader's language, with its numbers), else the short phrase the routing
 * screen uses for the same check.
 */
function vetLine(c: Schemas['VetCheck']): string {
  if (c.text) return c.text
  const result = 'pass'
  switch (c.key) {
    case 'domain':
      return tt(mr.checkDomain, { result })
    case 'quota':
      return tt(mr.checkQuota, { result, shape: c.left !== undefined ? 'left' : 'none', left: count(c.left ?? 0), period: 'none' })
    case 'warm':
      return tt(mr.checkWarm, { result, hasDay: 'no', day: 0, days: 0 })
    case 'sandbox':
      return tt(mr.checkSandbox, { result })
    case 'available':
      return tt(mr.checkAvailable, { result })
    case 'bulk_terms':
      return tt(mr.checkBulk, { result })
  }
  return c.key
}

const severityTone: Record<Schemas['ProviderIncidentEntry']['severity'], Tone> = { info: 'neutral', ok: 'ok', warn: 'warn', bad: 'bad' }

/** Incident timeline under the outage. */
export function IncidentTimeline({ incidents }: { incidents: Schemas['ProviderIncidentEntry'][] }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
        <h2 className="h2">{t(mi.timelineTitle)}</h2>
        <Link to="/w/$workspace/deliverability" params={{ workspace: slug }} className="k">
          {t(mi.allIncidents)}
        </Link>
      </div>
      {incidents.map((e, i) => (
        <div key={e.id} className={cn('pv-ev', i === 0 && 'border-t-0')}>
          <span className="mono k">{clockSeconds(e.at)}</span>
          <Badge tone={severityTone[e.severity]}>{t(mi.incidentKind, { kind: e.kind, raw: e.kind.replace(/_/g, ' ') })}</Badge>
          <span>
            {e.title}
            {e.detail ? ` ${e.detail}` : ''}
          </span>
        </div>
      ))}
    </section>
  )
}

/** Webhooks-silent banner: what ReputeCast checked, likely causes, impact. */
export function SilentAlert({ silent }: { silent: Silent }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshProviders()
  const reason = usePermissionReason('providers:manage')
  const path = { workspace: slug, providerId: silent.providerId }
  const test = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/webhook/test', {
    // PR-9: the provider was asked for a test event; it proves the endpoint
    // reachable by arriving, so the toast says it was asked, not received.
    onSuccess: (r) =>
      toast({
        kind: r.providerTest ? 'success' : 'error',
        title: r.providerTest ? t(mi.testRequested) : t(mi.testFailed),
        detail: [r.providerTest ? t(mi.testRequestedDetail, { provider: silent.short }) : r.providerTestDetail, r.endpointOk ? '' : r.endpointDetail]
          .filter(Boolean)
          .join(' '),
      }),
    onError: (e) => toast({ kind: 'error', title: t(mi.testFailed), detail: errorMessage(e) }),
  })
  const register = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/webhook/register', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(mi.registered), detail: t(mi.registeredDetail, { provider: silent.short, url: silent.webhookUrl }) })
    },
    onError: (e) => toast({ kind: 'error', title: t(mi.registerFailed), detail: errorMessage(e) }),
  })
  const backfill = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/backfill', {
    onSuccess: (r) => toast({ kind: 'progress', title: t(mi.backfilling, { count: r.queued }), detail: t(mi.backfillingDetail, { provider: silent.short }), duration: 6000 }),
    onError: (e) => toast({ kind: 'error', title: t(mi.backfillFailed), detail: errorMessage(e) }),
  })
  const [manual, setManual] = useState(false)
  const noteId = useId()
  const now = useNow()
  const minutes = (now.getTime() - new Date(silent.since).getTime()) / 60000
  const hours = Math.floor(minutes / 60)
  const host = hostOf(silent.webhookUrl)
  return (
    <section role="alert" className="card flex flex-col border-[#FCD34D]">
      <Dialog
        open={manual}
        onOpenChange={setManual}
        title={t(mi.manualTitle, { provider: silent.short })}
        description={t(mi.manualDescription, { provider: silent.short })}
        footer={<Button onClick={() => setManual(false)}>{t(common.done)}</Button>}
      >
        <ol className="m-0 flex list-decimal flex-col gap-2 pl-[18px] text-[13px]">
          <li>{t(mi.manualStep1, { provider: silent.short })}</li>
          <li>
            {t(mi.manualStep2)}
            <pre className="pre mt-1.5 whitespace-pre-wrap break-all">{silent.webhookUrl}</pre>
          </li>
          <li>{t(mi.manualStep3)}</li>
        </ol>
      </Dialog>
      <div className="flex flex-wrap items-start gap-[18px] rounded-t-[10px] bg-warn-tint px-[22px] py-5">
        <AlertMark fill="#FDE68A" stroke="#92400E" />
        <div className="flex flex-[1_1_320px] flex-col gap-1">
          <span className="text-[20px] font-bold tracking-[-0.01em] text-[#78350F]">
            {hours >= 1 ? t(mi.silentForHours, { provider: silent.short, hours }) : t(mi.silentFor, { provider: silent.short, duration: duration(minutes) })}
          </span>
          {silent.overdue > 0 ? (
            <span className="flex flex-wrap items-center gap-2 text-[#78350F]">
              <Badge tone="warn" dot>
                {t(mi.eventsOverdue)}
              </Badge>
              <span className="font-semibold">{t(mi.overdueLine, { count: silent.overdue, hours: Math.max(hours, 6) })}</span>
            </span>
          ) : null}
          <span className="text-[#78350F]">{t(mi.silentBody, { time: clock(silent.since), provider: silent.short, count: count(silent.accepted) })}</span>
          {silent.broadcastShareHeld ? <span className="font-semibold text-[#78350F]">{t(mi.silentNoFailover, { provider: silent.short })}</span> : null}
        </div>
        <div className="flex max-w-[360px] flex-col items-end gap-1.5 max-md:max-w-none max-md:items-start">
          <div className="flex flex-wrap justify-end gap-2.5 max-md:justify-start">
            {silent.canSendTest ? (
              <Button disabledReason={reason} loading={test.isPending} onClick={() => test.mutate({ params: { path } })}>
                {t(mi.sendTestEvent)}
              </Button>
            ) : null}
            <Button disabledReason={reason} loading={register.isPending} onClick={() => (silent.canRegister ? register.mutate({ params: { path } }) : setManual(true))}>
              {t(mi.reregister)}
            </Button>
            {silent.canBackfill ? (
              <Button variant="primary" aria-describedby={noteId} disabledReason={reason} loading={backfill.isPending} onClick={() => backfill.mutate({ params: { path } })}>
                {t(mi.checkNow)}
              </Button>
            ) : null}
          </div>
          {silent.canBackfill ? (
            <span className="k text-right max-md:text-left" id={noteId}>
              {t(mi.checkNowNote, { count: silent.overdue, provider: silent.short })}
            </span>
          ) : null}
        </div>
      </div>
      <Grid min={300} gap={0}>
        <div className="flex flex-col gap-2 border-r border-line-2 px-[22px] py-[18px]">
          <span className="font-semibold">{t(mi.checkedTitle)}</span>
          <ul className="m-0 flex list-disc flex-col gap-1.5 pl-[18px] text-[13px]">
            {silent.checks.map((c) => (
              <li key={c.key}>
                <CheckText c={c} s={silent} />
              </li>
            ))}
          </ul>
        </div>
        <div className="flex flex-col gap-2 border-r border-line-2 px-[22px] py-[18px]">
          <span className="font-semibold">{t(mi.causesTitle)}</span>
          <ol className="m-0 flex list-decimal flex-col gap-1.5 pl-[18px] text-[13px]">
            {silent.causes.map((c, i) => (
              <li key={c}>
                <CauseText cause={c} short={silent.short} host={host} first={i === 0} />
              </li>
            ))}
          </ol>
        </div>
        <div className="flex flex-col gap-2 px-[22px] py-[18px]">
          <span className="font-semibold">{t(mi.impactTitle)}</span>
          <ImpactRow k={t(mi.impactUnknown)}>
            <span className="n">{count(silent.accepted)}</span>
          </ImpactRow>
          <ImpactRow k={t(mi.impactOverdue)} sub>
            <span className="n">{count(silent.overdue)}</span>
          </ImpactRow>
          <ImpactRow k={t(mi.impactInGrace)} sub>
            <span className="n">{count(silent.inGrace)}</span>
          </ImpactRow>
          <ImpactRow k={t(mi.impactCampaigns)}>
            <span className="n">
              {silent.campaigns.length}
              {silent.campaigns.length ? ` · ${silent.campaigns.map((c) => `"${c.name}"`).join(', ')}` : ''}
            </span>
          </ImpactRow>
          <ImpactRow k={t(mi.impactBounces)}>
            <span className="n">{t(mi.unknown)}</span>
          </ImpactRow>
          <ImpactRow k={t(mi.impactShare, { provider: silent.short })}>
            {silent.broadcastShareHeld ? <Badge tone="warn">{t(mi.shareHeld)}</Badge> : <span className="n">{t(mi.shareNotUsed)}</span>}
          </ImpactRow>
          <span className="k">{silent.canBackfill ? t(mi.backfillNote, { provider: silent.short }) : t(mi.noBackfillNote, { provider: silent.short })}</span>
        </div>
      </Grid>
    </section>
  )
}

function ImpactRow({ k, children, sub }: { k: string; children: ReactNode; sub?: boolean }) {
  return (
    <div className={cn('flex justify-between gap-3 text-[13px] max-md:flex-wrap', sub && 'pl-3.5 text-ink-2')}>
      <span>{k}</span>
      {children}
    </div>
  )
}

function hostOf(url: string) {
  try {
    return new URL(url).host
  } catch {
    return 'ReputeCast'
  }
}

/** One check of the silent-webhooks banner, in a sentence. */
function CheckText({ c, s }: { c: Schemas['ProviderWebhookCheck']; s: Silent }): ReactNode {
  const t = useT()
  const ok = c.ok === false ? 'no' : 'yes'
  const detail = { hasDetail: c.detail ? 'yes' : 'no', detail: c.detail ?? '' }
  switch (c.key) {
    case 'api':
      return t(mi.checkApi, { provider: s.short, ok, hasAt: c.at ? 'yes' : 'no', time: c.at ? clock(c.at) : '', ...detail })
    case 'registered':
      return c.ok === false ? (
        t(mi.checkRegisteredNo, { provider: s.short, ...detail })
      ) : (
        <T
          msg={mi.checkRegisteredYes}
          values={{ provider: s.short, hasId: c.id ? 'yes' : 'no', id: c.id ?? '', url: c.url || s.webhookUrl, mono: (chunks) => <span className="mono">{chunks}</span> }}
        />
      )
    case 'endpoint':
      return t(mi.checkEndpoint, { ok, ...detail })
    case 'provider_log':
      return t(mi.checkProviderLog, {
        provider: s.short,
        count: count(c.count ?? 0),
        hasQueued: c.queued !== undefined ? 'yes' : 'no',
        queued: count(c.queued ?? 0),
      })
  }
  return null
}

/** A likely cause; the first, most likely one leads in bold. */
function CauseText({ cause, short, host, first }: { cause: string; short: string; host: string; first: boolean }): ReactNode {
  const t = useT()
  const b = (chunks: ReactNode[]) => <strong>{chunks}</strong>
  switch (cause) {
    case 'worker_stuck':
      return first ? <T msg={mi.causeWorkerStuckFirst} values={{ provider: short, postal: short === 'Postal' ? 'yes' : 'no', b }} /> : t(mi.causeWorkerStuck, { provider: short })
    case 'firewall':
      return first ? <T msg={mi.causeFirewallFirst} values={{ provider: short, host, b }} /> : t(mi.causeFirewall, { provider: short, host })
    case 'secret_rotated':
      return first ? t(mi.causeSecretFirst, { provider: short }) : t(mi.causeSecret, { provider: short })
    case 'webhook_removed':
      return first ? <T msg={mi.causeRemovedFirst} values={{ provider: short, b }} /> : t(mi.causeRemoved, { provider: short })
    case 'endpoint_unreachable':
      return first ? <T msg={mi.causeUnreachableFirst} values={{ host, b }} /> : t(mi.causeUnreachable, { host })
  }
  return cause
}

/** Messages accepted while the webhooks were silent, newest first. */
export function AwaitingTable({ silent }: { silent: Silent }) {
  const t = useT()
  return (
    <section className="card">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
        <h2 className="h2">{t(mi.awaitingTitle, { provider: silent.short })}</h2>
        <span className="k">{t(mi.awaitingCount, { count: count(silent.accepted) })}</span>
      </div>
      <Table className="pv-11" phoneWrap>
        <thead>
          <tr>
            <Th>{t(mi.colAccepted)}</Th>
            <Th>{t(mi.colRecipient)}</Th>
            <Th>{t(mi.colCampaign)}</Th>
            <Th>{t(mi.colMessageId, { provider: silent.short })}</Th>
            <Th>{t(mi.colStatus)}</Th>
          </tr>
        </thead>
        <tbody>
          {silent.awaiting.map((msg, i) => (
            <tr key={`${msg.providerMessageId}-${i}`}>
              <Td className="mono">{clockSeconds(msg.at)}</Td>
              <Td className="mono">{msg.recipient}</Td>
              <Td>{msg.source}</Td>
              <Td className="mono">{msg.providerMessageId || '—'}</Td>
              <Td>
                {msg.status === 'accepted' ? (
                  <Badge>{t(mi.acceptedNoEvent)}</Badge>
                ) : (
                  <Badge tone={msg.status === 'delivered' ? 'ok' : msg.status === 'deferred' ? 'warn' : 'bad'}>
                    {t(mi.awaitingStatus, { status: msg.status, hasTime: msg.deliveredAt ? 'yes' : 'no', time: msg.deliveredAt ? clock(msg.deliveredAt) : '' })}
                  </Badge>
                )}
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
    </section>
  )
}

const statusText: Record<number, string> = { 401: 'Unauthorized', 403: 'Forbidden' }

const lowerFirst = (s: string) => (s.length > 1 && s[1] === s[1]!.toLowerCase() ? s[0]!.toLowerCase() + s.slice(1) : s)

/**
 * How the banner names the rejected credential: "server API token" in the
 * title, "Update API token" on the button, "the token" in running text.
 * The label is the adapter's own English ("Server API token").
 */
export function credentialWords(label?: string): { title: string; button: string; short: string } {
  const l = label || 'Credentials'
  return { title: lowerFirst(l), button: lowerFirst(l.replace(/^Server /, '')), short: (l.split(' ').pop() ?? l).toLowerCase() }
}

/** Rejected credentials: what failed, what ReputeCast did, and the fix. */
export function AuthBanner({ af, onUpdate }: { af: AuthFailure; onUpdate: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('providers:manage')
  const first = af.down.errors[0]
  const status = first?.status || 401
  const words = credentialWords(af.down.credentialLabel)
  const failing = af.streams.filter((s) => s.role !== 'fallback' && s.state === 'failover_active')
  const clauses = failing.map((s, i) => {
    const values = { first: i === 0 ? 'yes' : 'no', stream: streamName[s.stream as Stream], mail: t(mi.streamMail, { stream: s.stream }), fallback: s.fallback ?? '' }
    return i === failing.length - 1 ? t(mi.authFailingOverRerouted, { ...values, count: count(af.rerouted) }) : t(mi.authFailingOver, values)
  })
  return (
    <div role="alert" className="flex items-start gap-3.5 rounded-[10px] border border-[#FECACA] bg-[#FEF2F2] px-4 py-3.5 max-md:flex-wrap">
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="#B91C1C" strokeWidth="1.8" aria-hidden="true" className="flex-none">
        <circle cx="10" cy="10" r="8" />
        <path d="M10 6v5M10 13.5v.5" strokeLinecap="round" />
      </svg>
      <div className="flex flex-[1_1_300px] flex-col gap-1">
        <span className="font-semibold text-bad-fg">{t(mi.authTitle, { provider: af.provider, status, statusText: statusText[status] ?? 'Unauthorized', credential: words.title })}</span>
        <span className="text-[13px] text-[#7F1D1D]">
          {[t(mi.authEveryCall, { time: clock(af.down.since) }), ...clauses, t(mi.authRotated, { credential: words.short, provider: af.short })].join(' ')}
        </span>
        <div className="mt-1.5 flex flex-wrap gap-2.5">
          <Button size="sm" variant="primary" disabledReason={reason} onClick={onUpdate}>
            {t(mi.update, { credential: words.button })}
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/providers/events" params={{ workspace: slug }} search={{ provider: af.providerId }}>
            {t(mi.viewFailed)}
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }}>
            {t(mi.auditLog)}
          </ButtonLink>
        </div>
      </div>
    </div>
  )
}

/** The provider card while its credentials are rejected. */
export function AuthCard({ af, card }: { af: AuthFailure; card?: ProviderCard }) {
  const t = useT()
  const e = af.down.errors[0]
  return (
    <div className="card flex flex-wrap gap-5 border-bad-line px-[22px] py-5">
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <span className="text-[17px] font-bold">{af.provider}</span>
          <Badge tone="bad" dot>
            {t(m.statusDownAuth)}
          </Badge>
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact className="pv-fact" k={t(mi.lastSuccessCall)}>
            <span>{af.down.lastSuccessAt ? `${clockSeconds(af.down.lastSuccessAt)}${af.down.lastSuccessOp ? ` · ${af.down.lastSuccessOp}` : ''}` : t(mi.noneRecorded)}</span>
          </Fact>
          <Fact className="pv-fact" k={t(mi.errorFact)}>
            <span className="mono text-bad-fg">{e ? [e.status || null, e.code || null, e.message || null].filter(Boolean).join(' · ') : t(mi.credentialsRejected)}</span>
          </Fact>
          <Fact className="pv-fact" k={t(mo.factWebhooks)}>
            <span>
              {t(mi.authWebhooks, {
                hasEvent: card?.webhook.lastEventAt ? 'yes' : 'no',
                time: card?.webhook.lastEventAt ? clock(card.webhook.lastEventAt) : '',
                credential: credentialWords(af.down.credentialLabel).short,
              })}
            </span>
          </Fact>
        </div>
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <span className="k">{t(mi.streamsAffected)}</span>
        {[...af.streams.filter((s) => s.role !== 'fallback'), ...af.streams.filter((s) => s.role === 'fallback')].map((s) => (
          <div key={s.stream} className="flex justify-between gap-3 text-[13px] max-md:flex-wrap">
            <span>{t(mi.streamRole, { stream: streamName[s.stream as Stream], role: s.role })}</span>
            {s.state === 'failover_active' && s.role !== 'fallback' ? (
              <Badge tone="ok">{t(mi.failoverTo, { provider: s.fallbackShort ?? '' })}</Badge>
            ) : s.state === 'fallback_unavailable' ? (
              <Badge tone="warn">{t(mi.fallbackUnavailable)}</Badge>
            ) : (
              <Badge tone="warn">{s.state === 'waiting' ? t(mi.waitingDecision) : t(mi.held)}</Badge>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
