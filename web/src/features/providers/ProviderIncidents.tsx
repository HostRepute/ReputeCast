import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Grid } from '@/components/ui/Layout'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { useNow } from '@/lib/hooks'
import { andList, clock, clockSeconds, count, duration } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { failoverText, Fact, sinceText, streamName, type ProviderCard, type Stream } from './common'

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

const plural = (name: string) => (name.endsWith('s') ? 'are' : 'is')

/** Re-checks a provider now (health probe) and refreshes the screen. */
export function RecheckButton({ providerId, label, variant }: { providerId: string; label: string; variant?: 'primary' }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshProviders()
  const reason = usePermissionReason('providers:manage')
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/check', {
    onSuccess: (card) => {
      void refresh()
      toast({ kind: card.status === 'down' ? 'error' : 'success', title: card.status === 'down' ? `${card.name} is still down` : `${card.name} answered`, detail: card.status === 'down' ? 'The next automatic check runs within 5 minutes.' : 'Health refreshed.' })
    },
    onError: (e) => toast({ kind: 'error', title: 'Check failed', detail: errorMessage(e) }),
  })
  return (
    <Button
      variant={variant}
      disabledReason={reason}
      loading={check.isPending}
      onClick={() => check.mutate({ params: { path: { workspace: slug, providerId } } })}
    >
      {label}
    </Button>
  )
}

/** "SendEmail returns 503 ServiceUnavailable and ThrottlingException on 96% of calls." */
function outageSentence(o: Outage): string {
  const parts: string[] = []
  if (o.codes.length) {
    parts.push(`${o.operation || 'The API'} returns ${andList(o.codes)}${o.errorRate !== undefined ? ` on ${Math.round(o.errorRate)}% of calls` : ''}.`)
  }
  if (o.down.statusSummary) parts.push(o.down.statusSummary)
  const handled = o.streams.filter((s) => s.state === 'failover_active' || s.state === 'fallback_unavailable').map((s) => streamName[s.stream as Stream])
  const waiting = o.streams.filter((s) => s.state === 'waiting' || s.state === 'held').map((s) => streamName[s.stream as Stream])
  const tail: string[] = []
  if (handled.length) tail.push(`${andList(handled)} handled themselves`)
  if (waiting.length) tail.push(`${andList(waiting)} ${waiting.length > 1 ? 'are' : plural(waiting[0]!)} waiting for you`)
  if (tail.length) parts.push(`${tail.join('; ')}.`)
  return parts.join(' ')
}

/** Outage banner: what is down, and how each stream copes. */
export function OutageAlert({ outage }: { outage: Outage }) {
  // Streams that handled themselves first; the ones waiting for someone last.
  const waits = (s: OutageStream) => (s.state === 'waiting' || s.state === 'held' ? 1 : 0)
  const columns = [...outage.streams].sort((a, b) => waits(a) - waits(b))
  return (
    <section role="alert" className="card flex flex-col border-bad-line">
      <div className="flex flex-wrap items-start gap-[18px] rounded-t-[10px] bg-[#FEF2F2] px-[22px] py-5">
        <AlertMark fill="#FECACA" stroke="#991B1B" />
        <div className="flex flex-[1_1_320px] flex-col gap-1">
          <span className="text-[20px] font-bold tracking-[-0.01em] text-[#7F1D1D]">
            {outage.provider} is down: API errors since {clock(outage.since)} ({sinceText(outage.since)})
          </span>
          <span className="text-[#7F1D1D]">{outageSentence(outage)}</span>
        </div>
        <div className="flex flex-wrap gap-2.5">
          {outage.down.statusUrl ? <ButtonAnchor href={outage.down.statusUrl}>{outage.down.statusLabel || `${outage.short} status`}</ButtonAnchor> : null}
          <RecheckButton providerId={outage.providerId} label={`Re-check ${outage.short} now`} />
        </div>
      </div>
      <Grid min={300} gap={0}>
        {columns.map((s, i) => (
          <OutageColumn key={s.stream} s={s} outage={outage} last={i === columns.length - 1} />
        ))}
      </Grid>
    </section>
  )
}

function streamBadge(s: OutageStream): { tone: Tone; text: string } {
  switch (s.state) {
    case 'failover_active':
      return { tone: 'ok', text: 'Failover active' }
    case 'fallback_unavailable':
      return { tone: 'warn', text: 'Fallback unavailable' }
    case 'waiting':
      return { tone: 'warn', text: 'Waiting for a decision' }
    case 'held':
      return { tone: 'warn', text: 'Held' }
  }
}

/** "211 welcome series, 97 onboarding" as the board writes it: "welcome series 211, onboarding 97". */
function reroutedList(s: OutageStream) {
  return s.reroutedBy.map((r) => `${r.name} ${count(r.count)}`).join(', ')
}

function OutageColumn({ s, outage, last }: { s: OutageStream; outage: Outage; last: boolean }) {
  const b = streamBadge(s)
  const name = streamName[s.stream as Stream]
  let body: ReactNode
  let note: ReactNode = null
  if (s.state === 'waiting' && s.campaign) {
    body = <HeldCampaign s={s} outage={outage} />
  } else if (s.role === 'fallback') {
    const sent = s.reroutedBy.length
      ? `The ${andList(s.reroutedBy.map((r) => `${count(r.count)} ${r.name}`))} sent since ${clock(s.since)} went through ${s.primary} directly.`
      : `Nothing was sent on ${name} since ${clock(s.since)}.`
    body = (
      <span className="text-[13px]">
        Primary {s.primary} is {s.state === 'fallback_unavailable' ? 'unavailable too' : 'healthy'}; its {outage.short} fallback is unavailable. {sent}
      </span>
    )
    if (s.suggestion) note = <span className="k">Suggestion: add {s.suggestion} as a second fallback for {name}.</span>
  } else if (s.state === 'failover_active') {
    body = (
      <span className="text-[13px]">
        Primary {outage.short} → fallback {s.fallbackShort}, automatically since {clock(s.since)}. {count(s.rerouted)} emails rerouted so far
        {s.reroutedBy.length ? `: ${reroutedList(s)}.` : '.'}
      </span>
    )
    note = (
      <span className="k">
        Will switch back to {outage.short} after {s.switchBackMinutes} healthy minutes.
      </span>
    )
  } else {
    body = (
      <span className="text-[13px]">
        New {name} mail for {outage.short} is held until it recovers{s.policy !== 'automatic' ? `: failover is ${failoverText[s.policy as keyof typeof failoverText]}` : ''}.
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

/** A broadcast whose share on the down provider waits for a decision. */
function HeldCampaign({ s, outage }: { s: OutageStream; outage: Outage }) {
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
        toast({ kind: 'success', title: `Waiting for ${outage.short}`, detail: `"${c.name}" resumes on its own once ${outage.short} is healthy.` })
        void navigate({ to: '/w/$workspace/campaigns/$campaignId/live', params: { workspace: slug, campaignId: c.id } })
      } else if (action === 'move') {
        toast({ kind: 'success', title: `Moved ${count(res.moved)} recipients`, detail: `"${c.name}" continues on ${vars.body.targetProviderId ? s.alternatives.find((a) => a.providerId === vars.body.targetProviderId)?.short : 'the fallback'}.` })
      } else {
        toast({ kind: 'deleted', title: `"${c.name}" cancelled`, detail: `${count(c.held)} held recipients won't be sent.` })
      }
    },
    onError: (e) => toast({ kind: 'error', title: 'The decision was not saved', detail: errorMessage(e) }),
  })
  const through = c.planned > 0 ? Math.round((c.sent / c.planned) * 100) : 0
  const others = c.otherShares.map((o) => `${o.short} (${o.percent}%)`)
  const policy =
    s.policy === 'warm_only'
      ? 'Broadcast failover only moves to warmed-up providers, and none is available right now.'
      : 'Broadcast failover is pause-and-ask because moving a bulk send to a cold provider mid-send can hurt reputation.'
  const run = (action: 'wait' | 'move' | 'cancel', target?: string) =>
    decide.mutate({ params: { path: { workspace: slug, providerId: outage.providerId, campaignId: c.id } }, body: { action, targetProviderId: target } })
  if (c.decision) {
    return (
      <span className="text-[13px]">
        &ldquo;{c.name}&rdquo;: {c.decision === 'wait' ? `waiting for ${outage.short}; it resumes on its own after ${s.switchBackMinutes} healthy minutes.` : c.decision === 'move' ? 'the held share was moved.' : 'the send was cancelled.'}{' '}
        <Link to="/w/$workspace/campaigns/$campaignId/live" params={{ workspace: slug, campaignId: c.id }}>
          Open the live monitor
        </Link>
      </span>
    )
  }
  return (
    <>
      <span className="text-[13px]">
        &ldquo;{c.name}&rdquo; was {through}% through its {outage.short} share when {outage.short} failed: {count(c.sent)} sent, {count(c.held)} held.
        {others.length ? ` ${andList(others)} ${others.length > 1 ? 'keep' : 'keeps'} sending ${others.length > 1 ? 'their shares' : 'its share'}.` : ''} {policy}
      </span>
      <div className="mt-1 flex flex-wrap gap-2">
        <Button size="sm" variant="primary" disabledReason={sendReason} loading={decide.isPending && decide.variables?.body.action === 'wait'} onClick={() => run('wait')}>
          Wait for {outage.short}, auto-resume
        </Button>
        {s.alternatives.map((a) => (
          <Button key={a.providerId} size="sm" disabledReason={sendReason} onClick={() => setConfirm({ action: 'move', target: a })}>
            Move to {a.short}
            {a.exceedsCap ? ' · exceeds warm-up cap, not recommended' : a.warmingUp ? ' · warming up' : ''}
          </Button>
        ))}
        <Button size="sm" variant="danger" disabledReason={pauseReason} onClick={() => setConfirm({ action: 'cancel' })}>
          Cancel send
        </Button>
      </div>
      <Dialog
        open={confirm?.action === 'move'}
        onOpenChange={(o) => !o && setConfirm(null)}
        title={`Move ${count(c.held)} recipients to ${confirm?.target?.short ?? ''}?`}
        description={
          confirm?.target?.exceedsCap
            ? `${confirm.target.short} is warming up: this is more than its cap for today, and the warm-up guard may pause it within minutes. Waiting for ${outage.short} is the safer choice.`
            : `The held share of "${c.name}" continues on ${confirm?.target?.short ?? ''} right away.`
        }
        footer={
          <>
            <Button onClick={() => setConfirm(null)}>Keep waiting</Button>
            <Button variant={confirm?.target?.exceedsCap ? 'danger-primary' : 'primary'} loading={decide.isPending} onClick={() => run('move', confirm?.target?.providerId)}>
              Move to {confirm?.target?.short}
            </Button>
          </>
        }
      />
      <Dialog
        open={confirm?.action === 'cancel'}
        onOpenChange={(o) => !o && setConfirm(null)}
        dismissible={false}
        title={`Cancel "${c.name}"?`}
        description={`${count(c.held)} held recipients won't get this campaign. The ${count(c.sent)} already sent stay sent, and the rest of the send stops. This can't be undone.`}
        footer={
          <>
            <Button onClick={() => setConfirm(null)}>Keep waiting</Button>
            <Button variant="danger-primary" loading={decide.isPending} onClick={() => run('cancel')}>
              Cancel send
            </Button>
          </>
        }
      />
    </>
  )
}

const incidentBadge: Record<string, string> = {
  broadcast_held: 'waiting',
  failover: 'failover',
  failover_ended: 'switched back',
  provider_down: 'down',
  provider_recovered: 'recovered',
  campaign_started: 'started',
  webhooks_silent: 'silent',
  webhooks_resumed: 'resumed',
}
const severityTone: Record<Schemas['ProviderIncidentEntry']['severity'], Tone> = { info: 'neutral', ok: 'ok', warn: 'warn', bad: 'bad' }

/** Incident timeline under the outage. */
export function IncidentTimeline({ incidents }: { incidents: Schemas['ProviderIncidentEntry'][] }) {
  const { slug } = useWorkspace()
  return (
    <section className="card">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
        <h2 className="h2">Incident timeline</h2>
        <Link to="/w/$workspace/deliverability" params={{ workspace: slug }} className="k">
          All incidents
        </Link>
      </div>
      {incidents.map((e, i) => (
        <div key={e.id} className={cn('pv-ev', i === 0 && 'border-t-0')}>
          <span className="mono k">{clockSeconds(e.at)}</span>
          <Badge tone={severityTone[e.severity]}>{incidentBadge[e.kind] ?? e.kind.replace(/_/g, ' ')}</Badge>
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
  const { slug } = useWorkspace()
  const refresh = useRefreshProviders()
  const reason = usePermissionReason('providers:manage')
  const path = { workspace: slug, providerId: silent.providerId }
  const test = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/webhook/test', {
    onSuccess: (r) =>
      toast({
        kind: r.endpointOk ? 'success' : 'error',
        title: r.endpointOk ? 'Test event received' : 'Test event failed',
        detail: r.providerTest ? (r.providerTestDetail ?? r.endpointDetail) : r.endpointDetail,
      }),
    onError: (e) => toast({ kind: 'error', title: 'Test event failed', detail: errorMessage(e) }),
  })
  const register = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/webhook/register', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: 'Webhook registered again', detail: `${silent.short} now posts events to ${silent.webhookUrl}.` })
    },
    onError: (e) => toast({ kind: 'error', title: "Couldn't re-register the webhook", detail: errorMessage(e) }),
  })
  const backfill = $api.useMutation('post', '/api/v1/workspaces/{workspace}/providers/{providerId}/backfill', {
    onSuccess: (r) => toast({ kind: 'progress', title: `Backfilling ${count(r.queued)} messages`, detail: `Status from ${silent.short} is normalized as it arrives.`, duration: 6000 }),
    onError: (e) => toast({ kind: 'error', title: "Couldn't start the backfill", detail: errorMessage(e) }),
  })
  const [manual, setManual] = useState(false)
  const now = useNow()
  const minutes = (now.getTime() - new Date(silent.since).getTime()) / 60000
  const hours = Math.floor(minutes / 60)
  const host = hostOf(silent.webhookUrl)
  return (
    <section role="alert" className="card flex flex-col border-[#FCD34D]">
      <Dialog
        open={manual}
        onOpenChange={setManual}
        title={`Re-register the webhook in ${silent.short}`}
        description={`${silent.short} has no API for webhooks, so add ReputeCast's endpoint again in ${silent.short} itself.`}
        footer={<Button onClick={() => setManual(false)}>Done</Button>}
      >
        <ol className="m-0 flex list-decimal flex-col gap-2 pl-[18px] text-[13px]">
          <li>In {silent.short}, open the mail server, then Webhooks, and delete the old ReputeCast webhook if it&apos;s still there.</li>
          <li>
            Add a webhook with this URL, for all events:
            <pre className="pre mt-1.5 whitespace-pre-wrap break-all">{silent.webhookUrl}</pre>
          </li>
          <li>Back here, choose Send test event: the banner clears as soon as events arrive.</li>
        </ol>
      </Dialog>
      <div className="flex flex-wrap items-start gap-[18px] rounded-t-[10px] bg-warn-tint px-[22px] py-5">
        <AlertMark fill="#FDE68A" stroke="#92400E" />
        <div className="flex flex-[1_1_320px] flex-col gap-1">
          <span className="text-[20px] font-bold tracking-[-0.01em] text-[#78350F]">
            {silent.short} webhooks have been silent for {hours >= 1 ? `${hours} hour${hours === 1 ? '' : 's'}` : duration(minutes)}
          </span>
          <span className="text-[#78350F]">
            No delivery, bounce or complaint events since {clock(silent.since)}. {silent.short} accepted {count(silent.accepted)} messages in that time, so sending works, but their status is
            unknown and bounce protection for this route is blind. The {silent.short} route is marked Degraded
            {silent.broadcastShareHeld ? ' and its share of Broadcasts is held until events resume.' : '.'}
          </span>
        </div>
        <div className="flex flex-wrap gap-2.5">
          {silent.canSendTest ? (
            <Button disabledReason={reason} loading={test.isPending} onClick={() => test.mutate({ params: { path } })}>
              Send test event
            </Button>
          ) : null}
          <Button
            disabledReason={reason}
            loading={register.isPending}
            onClick={() => (silent.canRegister ? register.mutate({ params: { path } }) : setManual(true))}
          >
            Re-register webhook
          </Button>
          {silent.canBackfill ? (
            <Button variant="primary" disabledReason={reason} loading={backfill.isPending} onClick={() => backfill.mutate({ params: { path } })}>
              Backfill from {silent.short} API
            </Button>
          ) : null}
        </div>
      </div>
      <Grid min={300} gap={0}>
        <div className="flex flex-col gap-2 border-r border-line-2 px-[22px] py-[18px]">
          <span className="font-semibold">What ReputeCast checked</span>
          <ul className="m-0 flex list-disc flex-col gap-1.5 pl-[18px] text-[13px]">
            {silent.checks.map((c) => (
              <li key={c.key}>{checkText(c, silent)}</li>
            ))}
          </ul>
        </div>
        <div className="flex flex-col gap-2 border-r border-line-2 px-[22px] py-[18px]">
          <span className="font-semibold">Likely causes</span>
          <ol className="m-0 flex list-decimal flex-col gap-1.5 pl-[18px] text-[13px]">
            {silent.causes.map((c, i) => (
              <li key={c}>{causeText(c, silent.short, host, i === 0)}</li>
            ))}
          </ol>
        </div>
        <div className="flex flex-col gap-2 px-[22px] py-[18px]">
          <span className="font-semibold">Impact so far</span>
          <ImpactRow k="Messages with unknown status">
            <span className="n">{count(silent.accepted)}</span>
          </ImpactRow>
          <ImpactRow k="Campaigns affected">
            <span className="n">
              {silent.campaigns.length}
              {silent.campaigns.length ? ` · ${silent.campaigns.map((c) => `"${c.name}"`).join(', ')}` : ''}
            </span>
          </ImpactRow>
          <ImpactRow k="Bounces not yet suppressed">
            <span className="n">unknown</span>
          </ImpactRow>
          <ImpactRow k={`${silent.short} share of Broadcasts`}>{silent.broadcastShareHeld ? <Badge tone="warn">held</Badge> : <span className="n">not used</span>}</ImpactRow>
          <span className="k">
            {silent.canBackfill
              ? `Backfill pulls message status from ${silent.short}'s API and normalizes it, so reports and the suppression list catch up.`
              : `${silent.short} can't be asked for past events; they arrive once its webhooks resume.`}
          </span>
        </div>
      </Grid>
    </section>
  )
}

function ImpactRow({ k, children }: { k: string; children: ReactNode }) {
  return (
    <div className="flex justify-between gap-3 text-[13px] max-md:flex-wrap">
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

function checkText(c: Schemas['ProviderWebhookCheck'], s: Silent): ReactNode {
  const yes = c.ok === false ? 'no' : 'yes'
  switch (c.key) {
    case 'api':
      return `${s.short} API reachable: ${yes}${c.at ? `, ${clock(c.at)}` : ''}${c.detail ? ` (${c.detail})` : ''}.`
    case 'registered':
      return c.ok === false ? (
        `Webhook registered in ${s.short}: no${c.detail ? `, ${c.detail}` : ''}.`
      ) : (
        <>
          Webhook registered in {s.short}: yes{c.id ? `, id ${c.id}` : ''}, pointing at <span className="mono">{c.url || s.webhookUrl}</span>.
        </>
      )
    case 'endpoint':
      return `Our endpoint reachable from the internet: ${yes}${c.detail ? `, ${c.detail}` : ''}.`
    case 'provider_log':
      return `Events delivered by ${s.short} in the last 6 h: ${count(c.count ?? 0)}${c.queued !== undefined ? ` (${s.short}'s own webhook log shows ${count(c.queued)} queued, ${count(c.count ?? 0)} sent)` : ''}.`
  }
}

function causeText(key: string, short: string, host: string, first: boolean): ReactNode {
  switch (key) {
    case 'worker_stuck':
      return first ? (
        <>
          <strong>{short}&apos;s webhook worker is stuck.</strong> Queued events with none sent points at the {short === 'Postal' ? 'postal-worker process' : 'worker that delivers webhooks'}, which usually restarts cleanly.
        </>
      ) : (
        `${short}'s webhook worker is stuck: events are queued but none are sent.`
      )
    case 'firewall':
      return first ? (
        <>
          <strong>A firewall change on the {short} host</strong> blocking outbound HTTPS to {host}.
        </>
      ) : (
        `A firewall change on the ${short} host blocking outbound HTTPS to ${host}.`
      )
    case 'secret_rotated':
      return `${first ? '' : 'Less likely: '}the endpoint secret rotated in ReputeCast but not in ${short} (signatures would fail with 401, and we've received no requests at all).`
    case 'webhook_removed':
      return first ? (
        <>
          <strong>The webhook was removed or disabled in {short}.</strong> Re-register it to point {short} at ReputeCast again.
        </>
      ) : (
        `The webhook was removed or disabled in ${short}.`
      )
    case 'endpoint_unreachable':
      return first ? (
        <>
          <strong>ReputeCast&apos;s endpoint isn&apos;t reachable from the internet.</strong> Check DNS, TLS and the reverse proxy for {host}.
        </>
      ) : (
        `ReputeCast's endpoint isn't reachable from the internet (DNS, TLS or the reverse proxy for ${host}).`
      )
  }
  return key
}

/** Messages accepted while the webhooks were silent, newest first. */
export function AwaitingTable({ silent }: { silent: Silent }) {
  return (
    <section className="card">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-[18px] py-3.5">
        <h2 className="h2">Messages awaiting status from {silent.short}</h2>
        <span className="k">{count(silent.accepted)} · newest first</span>
      </div>
      <Table className="pv-11" phoneWrap>
        <thead>
          <tr>
            <Th>Accepted</Th>
            <Th>Recipient</Th>
            <Th>Campaign</Th>
            <Th>{silent.short} message id</Th>
            <Th>Status</Th>
          </tr>
        </thead>
        <tbody>
          {silent.awaiting.map((m, i) => (
            <tr key={`${m.providerMessageId}-${i}`}>
              <Td className="mono">{clockSeconds(m.at)}</Td>
              <Td className="mono">{m.recipient}</Td>
              <Td>{m.source}</Td>
              <Td className="mono">{m.providerMessageId || '—'}</Td>
              <Td>
                {m.status === 'accepted' ? (
                  <Badge>accepted · no event</Badge>
                ) : (
                  <Badge tone={m.status === 'delivered' ? 'ok' : m.status === 'deferred' ? 'warn' : 'bad'}>
                    {m.status}
                    {m.deliveredAt ? ` ${clock(m.deliveredAt)}` : ''}
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
 */
export function credentialWords(label?: string): { title: string; button: string; short: string } {
  const l = label || 'Credentials'
  return { title: lowerFirst(l), button: lowerFirst(l.replace(/^Server /, '')), short: (l.split(' ').pop() ?? l).toLowerCase() }
}

const streamDesc: Record<Stream, string> = {
  system: 'double opt-in confirmations, preference-center links',
  automation: 'emails sent by flows',
  broadcast: 'campaigns',
}

/** Rejected credentials: what failed, what ReputeCast did, and the fix. */
export function AuthBanner({ af, onUpdate }: { af: AuthFailure; onUpdate: () => void }) {
  const { slug } = useWorkspace()
  const reason = usePermissionReason('providers:manage')
  const first = af.down.errors[0]
  const status = first?.status || 401
  const words = credentialWords(af.down.credentialLabel)
  const failing = af.streams.filter((s) => s.role !== 'fallback' && s.state === 'failover_active')
  return (
    <div role="alert" className="flex items-start gap-3.5 rounded-[10px] border border-[#FECACA] bg-[#FEF2F2] px-4 py-3.5 max-md:flex-wrap">
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="#B91C1C" strokeWidth="1.8" aria-hidden="true" className="flex-none">
        <circle cx="10" cy="10" r="8" />
        <path d="M10 6v5M10 13.5v.5" strokeLinecap="round" />
      </svg>
      <div className="flex flex-[1_1_300px] flex-col gap-1">
        <span className="font-semibold text-bad-fg">
          {af.provider} returned {status} {statusText[status] ?? 'Unauthorized'}: the {words.title} is no longer valid
        </span>
        <span className="text-[13px] text-[#7F1D1D]">
          Every call since {clock(af.down.since)} has failed.
          {failing.map((s) => ` The ${streamName[s.stream as Stream]} stream (${streamDesc[s.stream as Stream]}) is failing over to ${s.fallback} automatically;`).join('')}
          {failing.length ? ` ${count(af.rerouted)} messages rerouted so far.` : ''} Someone may have rotated the {words.short} in {af.short}.
        </span>
        <div className="mt-1.5 flex flex-wrap gap-2.5">
          <Button size="sm" variant="primary" disabledReason={reason} onClick={onUpdate}>
            Update {words.button}
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/providers/events" params={{ workspace: slug }} search={{ provider: af.providerId }}>
            View failed calls
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }}>
            Audit log
          </ButtonLink>
        </div>
      </div>
    </div>
  )
}

/** The provider card while its credentials are rejected. */
export function AuthCard({ af, card }: { af: AuthFailure; card?: ProviderCard }) {
  const e = af.down.errors[0]
  return (
    <div className="card flex flex-wrap gap-5 border-bad-line px-[22px] py-5">
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <span className="text-[17px] font-bold">{af.provider}</span>
          <Badge tone="bad" dot>
            Down · auth failed
          </Badge>
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact k="Last successful call">
            <span>{af.down.lastSuccessAt ? `${clockSeconds(af.down.lastSuccessAt)}${af.down.lastSuccessOp ? ` · ${af.down.lastSuccessOp}` : ''}` : 'none recorded'}</span>
          </Fact>
          <Fact k="Error">
            <span className="mono text-bad-fg">{e ? [e.status || null, e.code || null, e.message || null].filter(Boolean).join(' · ') : 'Credentials rejected'}</span>
          </Fact>
          <Fact k="Webhooks">
            <span>
              {card?.webhook.lastEventAt ? `last event ${clock(card.webhook.lastEventAt)}` : 'no events yet'} · unaffected until {credentialWords(af.down.credentialLabel).short} is fixed
            </span>
          </Fact>
        </div>
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <span className="k">Streams affected</span>
        {[...af.streams.filter((s) => s.role !== 'fallback'), ...af.streams.filter((s) => s.role === 'fallback')].map((s) => (
          <div key={s.stream} className="flex justify-between gap-3 text-[13px] max-md:flex-wrap">
            <span>
              {streamName[s.stream as Stream]} · {s.role === 'split' ? 'split' : s.role}
            </span>
            {s.state === 'failover_active' && s.role !== 'fallback' ? (
              <Badge tone="ok">Failover to {s.fallbackShort} active</Badge>
            ) : s.state === 'fallback_unavailable' ? (
              <Badge tone="warn">Fallback unavailable</Badge>
            ) : (
              <Badge tone="warn">{s.state === 'waiting' ? 'Waiting for a decision' : 'Held'}</Badge>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
