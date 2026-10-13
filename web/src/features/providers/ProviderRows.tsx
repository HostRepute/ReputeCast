import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Grid } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { useNow } from '@/lib/hooks'
import { andList, clock, clockSeconds, count, money, monthName, pct, shortDate, weekdayName } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import {
  agoSeconds,
  bouncePct,
  CapabilityChip,
  complaintPct,
  Fact,
  limitLabel,
  limitPct,
  Meter,
  perSec,
  rateColor,
  sinceText,
  StatusBadge,
  streamName,
  type ProviderCard,
  type Stream,
} from './common'

/** One provider per row inside the providers card; rows are separated by a line. */
export function RowShell({ children, className, last }: { children: ReactNode; className?: string; last?: boolean }) {
  return <div className={cn('flex flex-wrap gap-5 px-[22px] py-5', !last && 'border-b border-line-2', className)}>{children}</div>
}

function Name({ children, size = 17 }: { children: ReactNode; size?: number }) {
  return (
    <span className="font-bold" style={{ fontSize: size }}>
      {children}
    </span>
  )
}

/** Stream role badges: "Broadcasts 90%", "Automations primary", "System fallback". */
export function RoleBadges({ card, outage, silentHeld, down }: { card: ProviderCard; outage?: Schemas['ProviderOutage']; silentHeld?: boolean; down?: boolean }) {
  // Streams it carries first, then the ones it backs up (stream order kept).
  const roles = [...card.roles.filter((r) => r.role !== 'fallback'), ...card.roles.filter((r) => r.role === 'fallback')]
  return (
    <>
      {roles.map((r) => {
        const s = streamName[r.stream as Stream]
        if (r.role === 'fallback') {
          if (r.active || card.takingFailover.includes(r.stream)) {
            return (
              <Badge key={r.stream} tone="purple">
                {s} · active fallback
              </Badge>
            )
          }
          return <Badge key={r.stream}>{s} fallback</Badge>
        }
        let text = r.role === 'split' ? `${s} ${r.percent}%` : r.role === 'primary' ? `${s} primary` : `${s} · conditions`
        if (!down && (r.held || (silentHeld && r.stream === 'broadcast'))) text += ' · held'
        else if (outage && outage.providerId !== card.id && outage.streams.some((x) => x.stream === r.stream && x.state === 'waiting')) text += ' · still sending'
        return (
          <Badge key={r.stream} tone="purple">
            {text}
          </Badge>
        )
      })}
    </>
  )
}

function ipsText(card: ProviderCard) {
  return card.ipMode === 'dedicated' && card.ips.length ? <span className="mono">{`${card.ips.join(', ')} · dedicated`}</span> : <span>Shared pool · checked at domain level</span>
}

function webhookText(card: ProviderCard, withKind = true) {
  const kind = withKind && card.webhook.kind ? `${card.webhook.kind} · ` : ''
  return card.webhook.lastEventAt ? `${kind}last event ${agoSeconds(card.webhook.lastEventAt)}` : `${kind}no events yet`
}

function quotaMeter(card: ProviderCard, mode: 'normal' | 'outage' | 'silent', accepted?: number): ReactNode {
  const q = card.quota
  if (q.kind === 'none' || q.max <= 0) return null
  const share = Math.round((q.used / q.max) * 100)
  if (q.kind === 'warmup_daily') {
    let value = `${count(q.used)} sent · ${weekdayName(new Date())}`
    let used = q.used
    let color = 'var(--accent)'
    if (mode === 'outage') value = `${count(q.used)} sent · ${share}%`
    if (mode === 'silent') {
      used = accepted ?? q.used
      value = `${count(used)} accepted · status unknown`
      color = 'var(--warn-bar)'
    }
    return <Meter label={`Today's warm-up cap · ${count(q.max)}`} value={value} percent={(used / q.max) * 100} color={color} />
  }
  const extra = q.fromFailover > 0 ? ` · +${count(q.fromFailover)} from failover` : ''
  const label = q.kind === 'rolling_24h' ? `24 h quota · ${count(q.max)}` : `Plan quota · ${count(q.max)} per month`
  return <Meter label={label} value={`${count(q.used)} used · ${share}%${extra}`} percent={share} color="var(--accent)" />
}

function rateMeter(card: ProviderCard, mode: 'normal' | 'outage' = 'normal'): ReactNode {
  const r = card.rate
  const ramp = r.label === 'ramp limit'
  const label = r.max > 0 ? (ramp ? `Rate · ramp limit ${perSec(r.max)} /s` : `Max send rate · ${perSec(r.max)} /s`) : 'Send rate · no published limit'
  if (card.status === 'down') {
    return <Meter label={label} value={`${perSec(r.now)} /s · throttled`} percent={0} color="var(--accent)" valueClassName="text-bad-fg" />
  }
  const value = ramp || mode === 'outage' ? (mode === 'outage' ? `${perSec(r.now, true)} /s` : `${perSec(r.now)} /s now`) : `${perSec(r.now)} /s now · peak today ${perSec(r.peakToday)} /s`
  return <Meter label={label} value={value} percent={r.max > 0 ? (r.now / r.max) * 100 : 0} color="var(--accent)" />
}

function rateMeters(card: ProviderCard, stale = false, short = false) {
  const b = card.bounce
  const c = card.complaint
  const suffix = stale ? ' · stale' : ''
  return (
    <>
      <Meter
        label={limitLabel('Bounce', b, short)}
        value={`${bouncePct(b.value)}${b.value !== undefined ? suffix : ''}`}
        percent={b.value !== undefined ? (b.value / b.limit) * 100 : 0}
        color={rateColor(b.value, b.limit, stale)}
      />
      <Meter
        label={limitLabel('Complaint', c, short)}
        value={`${complaintPct(c.value)}${c.value !== undefined ? suffix : ''}`}
        percent={c.value !== undefined ? (c.value / c.limit) * 100 : 0}
        color={rateColor(c.value, c.limit, stale)}
      />
    </>
  )
}

function CostGrid({ card, monthStart }: { card: ProviderCard; monthStart: string }) {
  const c = card.cost
  const mon = monthName(monthStart)
  let left: string
  let rightLabel: string
  if (c.basis === 'plan') {
    left = `${money(c.monthly ?? c.toDate)} plan`
    rightLabel = 'Per 1,000 · at current usage'
  } else if (c.basis === 'self_hosted') {
    left = '$0 metered'
    rightLabel = c.monthly ? `Per 1,000 · you entered ${money(c.monthly, 0)} per month server` : 'Per 1,000 · self-hosted'
  } else {
    left = money(c.toDate)
    rightLabel = `Per 1,000 · from ${c.source || 'the provider’s pricing'}`
  }
  return (
    <Grid cols={2} gap={8} className="text-[13px]">
      <div>
        <div className="k">Cost to date · {mon}</div>
        <div className="n">{left}</div>
      </div>
      <div>
        <div className="k">{rightLabel}</div>
        <div className="n">{c.per1000 !== undefined ? money(c.per1000) : '—'}</div>
      </div>
    </Grid>
  )
}

function Capabilities({ card }: { card: ProviderCard }) {
  return (
    <div className="flex flex-[1_1_300px] flex-col gap-2">
      <span className="k">Capabilities</span>
      <div className="flex flex-wrap gap-1.5">
        {card.capabilities.map((cap) => (
          <CapabilityChip key={cap.label} cap={cap} />
        ))}
      </div>
    </div>
  )
}

/** A provider in its normal state: identity and facts, rates and quota, capabilities. */
export function ProviderRow({ card, monthStart, last, onSettings }: { card: ProviderCard; monthStart: string; last?: boolean; onSettings: () => void }) {
  const { slug } = useWorkspace()
  return (
    <RowShell last={last}>
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <Name>{card.name}</Name>
          <StatusBadge card={card} />
          {card.label ? <Badge>{card.label}</Badge> : null}
        </div>
        {card.roles.length ? (
          <div className="flex flex-wrap gap-1.5">
            <RoleBadges card={card} />
          </div>
        ) : null}
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact k="IPs">{ipsText(card)}</Fact>
          <Fact k="Webhooks">
            <span>{webhookText(card)}</span>
          </Fact>
          <Fact k="API error rate · 7 d">
            <span className="n">{pct(card.errorRate7d ?? 0)}</span>
          </Fact>
          <Fact k={card.accountKind === 'server' ? 'Server' : 'Account status'}>
            <span>{card.account || '—'}</span>
          </Fact>
        </div>
        <div className="flex flex-wrap gap-2">
          {card.warmup ? (
            <ButtonLink size="sm" to="/w/$workspace/providers/warmup" params={{ workspace: slug }}>
              Warm-up plan
            </ButtonLink>
          ) : (
            <ButtonLink size="sm" to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
              Domains
            </ButtonLink>
          )}
          <ButtonLink size="sm" to="/w/$workspace/providers/events" params={{ workspace: slug }} search={{ provider: card.id }}>
            Events
          </ButtonLink>
          <Button size="sm" onClick={onSettings} aria-label={`${card.name} settings`}>
            Settings
          </Button>
        </div>
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-3">
        {rateMeters(card)}
        {quotaMeter(card, 'normal')}
        {rateMeter(card)}
        <CostGrid card={card} monthStart={monthStart} />
      </div>
      <Capabilities card={card} />
    </RowShell>
  )
}

/** "09:15:52 503 ServiceUnavailable · SendEmail" */
export function errorLine(e: Schemas['ProviderCallError']): string {
  const at = clockSeconds(e.at)
  if (e.status >= 500) return `${at} ${e.status} ${e.code || 'error'} · ${e.op}`
  return `${at} ${e.code || e.status} · ${e.message || e.op}`
}

/** The provider that is down, during an outage. */
export function DownRow({ card, outage }: { card: ProviderCard; outage: Schemas['ProviderOutage'] }) {
  const down = outage.down
  return (
    <RowShell className="bg-bad-tint">
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <Name>{card.name}</Name>
          <Badge tone="bad" dot>
            Down · {sinceText(outage.since)}
          </Badge>
          {card.label ? <Badge>{card.label}</Badge> : null}
        </div>
        <div className="flex flex-wrap gap-1.5">
          <RoleBadges card={card} down />
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact k="API error rate · 3 min">
            <span className="n text-bad-fg">{outage.errorRate !== undefined ? `${Math.round(outage.errorRate)}%` : '—'}</span>
          </Fact>
          <Fact k="Last successful send">
            <span>{down.lastSuccessAt ? clockSeconds(down.lastSuccessAt) : 'none recorded'}</span>
          </Fact>
          <Fact k="Webhooks">
            <span>
              {webhookText(card)} (deliveries from before {clock(outage.since)})
            </span>
          </Fact>
          <Fact k="Account status">
            <span>{card.account || '—'}</span>
          </Fact>
        </div>
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-3">
        {rateMeters(card, false, true)}
        {quotaMeter(card, 'outage')}
        {rateMeter(card, 'outage')}
      </div>
      <div className="flex flex-[1_1_300px] flex-col gap-1.5">
        <span className="k">Recent errors</span>
        <div className="mono flex flex-col gap-1 text-[12px] text-bad-fg">
          {down.errors.length ? down.errors.slice(0, 4).map((e, i) => <span key={i}>{errorLine(e)}</span>) : <span>No error details recorded.</span>}
        </div>
      </div>
    </RowShell>
  )
}

/** Another provider while one is down: how it is affected and what it carries. */
export function OutageRow({ card, outage, last }: { card: ProviderCard; outage: Schemas['ProviderOutage']; last?: boolean }) {
  const forecast = outage.forecasts.find((f) => f.providerId === card.id)
  const whyNot = outage.whyNot.find((w) => w.providerId === card.id)
  const taking = card.takingFailover.map((s) => streamName[s as Stream])
  return (
    <RowShell last={last}>
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <Name>{card.name}</Name>
          <StatusBadge card={card} />
          {taking.length ? <Badge tone="info">Taking {andList(taking)} failover</Badge> : null}
        </div>
        <div className="flex flex-wrap gap-1.5">
          <RoleBadges card={card} outage={outage} />
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact k="Webhooks">
            <span>{webhookText(card, false)}</span>
          </Fact>
          <Fact k="API error rate · 3 min">
            <span className="n">{pct(card.errorRate3m ?? 0)}</span>
          </Fact>
        </div>
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-3">
        {card.warmup ? (
          <>
            {quotaMeter(card, 'outage')}
            {rateMeter(card, 'outage')}
          </>
        ) : (
          <>
            {rateMeters(card, false, true)}
            {quotaMeter(card, 'outage')}
          </>
        )}
      </div>
      {forecast ? (
        <div className="flex flex-[1_1_300px] flex-col gap-1.5">
          <span className="k">Quota forecast while {outage.short} is down</span>
          <span className="text-[13px]">
            {streamName[forecast.stream as Stream]} send about {count(forecast.perDay)} per day.
            {forecast.runsOutOn ? ` At this rate the ${card.short} plan runs out on ${dayMonth(forecast.runsOutOn)}` : ` The ${card.short} plan has room for now`}
            {forecast.nextTierAdds ? `; the next plan tier adds ${count(forecast.nextTierAdds)} for ${money(forecast.nextTierPrice ?? 0, 0)}.` : '.'}
          </span>
        </div>
      ) : whyNot ? (
        <div className="flex flex-[1_1_300px] flex-col gap-1.5">
          <span className="k">Why not move the Broadcast to {card.short}</span>
          <span className="text-[13px]">
            {count(whyNot.held)} held recipients would be {whyNot.multiple >= 1 ? `${Math.round(whyNot.multiple)}×` : `${Math.round(whyNot.multiple * 100)}% of`} today&apos;s cap
            {whyNot.rate > 0 ? ` and would push a ${whyNot.ageDays}-day-old IP to ${perSec(whyNot.rate)} /s` : ''}. The warm-up guard would pause it within minutes. Failover to a warmed-up
            provider only is the safe setting.
          </span>
        </div>
      ) : (
        <div className="flex-[1_1_300px]" />
      )}
    </RowShell>
  )
}

function dayMonth(d: string) {
  // "24 Oct"
  return shortDate(d).split(' ').slice(1).join(' ')
}

/** The provider whose webhooks went quiet. */
export function SilentRow({ card, silent }: { card: ProviderCard; silent: Schemas['ProviderWebhooksSilent'] }) {
  const now = useNow()
  return (
    <RowShell className="bg-warn-tint">
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <Name>{card.name}</Name>
          <Badge tone="warn" dot>
            Degraded · webhooks silent {sinceText(silent.since)}
          </Badge>
          {card.label ? <Badge>{card.label}</Badge> : null}
        </div>
        <div className="flex flex-wrap gap-1.5">
          <RoleBadges card={card} silentHeld={silent.broadcastShareHeld} />
          {card.warmup ? (
            <Badge tone="info">
              Warming up · day {card.warmup.day} of {card.warmup.days}
              {silent.broadcastShareHeld ? ', paused' : ''}
            </Badge>
          ) : null}
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact k="Webhooks">
            <span className="font-semibold text-warn-fg">
              last event {clock(silent.since)} · {sinceText(silent.since)} ago
            </span>
          </Fact>
          <Fact k="API error rate · 7 d">
            <span className="n">{pct(card.errorRate7d ?? 0)}</span>
          </Fact>
          <Fact k="IPs">{ipsText(card)}</Fact>
        </div>
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-3">
        {rateMeters(card, true)}
        {quotaMeter(card, 'silent', silent.accepted)}
      </div>
      <div className="flex flex-[1_1_300px] flex-col gap-1.5">
        <span className="k">Webhook freshness, last 24 h</span>
        <Freshness silent={silent} />
        <div className="k flex justify-between">
          <span>{clock(now.getTime() - 24 * 3600_000)} yesterday</span>
          <span>{clock(silent.since)}</span>
          <span>now</span>
        </div>
      </div>
    </RowShell>
  )
}

/** Events per hour for the last 24 hours; silent hours drawn as stubs. */
function Freshness({ silent }: { silent: Schemas['ProviderWebhooksSilent'] }) {
  const max = Math.max(1, ...silent.hourly)
  const silentFrom = new Date(silent.since).getTime()
  const from = new Date(silent.hourlyFrom).getTime()
  const quietHours = silent.hourly.filter((_, i) => from + i * 3600_000 >= silentFrom - 3600_000 && silent.hourly[i] === 0).length
  return (
    <div className="flex h-12 items-end gap-0.5" role="img" aria-label={`Events per hour for the last 24 hours; none in the last ${quietHours} hours`}>
      {silent.hourly.map((n, i) => {
        const quiet = n === 0
        return (
          <div
            key={i}
            className="flex-1 rounded-t-[2px]"
            style={{ height: quiet ? '3%' : `${Math.max(4, (n / max) * 100)}%`, background: quiet ? 'var(--bad-line)' : 'var(--accent)' }}
          />
        )
      })}
    </div>
  )
}

/** Healthy providers condensed to one line while another provider is degraded. */
export function CompactRow({ card, silent, last }: { card: ProviderCard; silent: Schemas['ProviderWebhooksSilent']; last?: boolean }) {
  const kind = card.webhook.kind && card.webhook.kind !== 'Registered' ? `${card.webhook.kind} · ` : ''
  const event = card.webhook.lastEventAt ? `last event ${agoSeconds(card.webhook.lastEventAt)}` : 'no events yet'
  const q = card.quota
  const quota = q.kind === 'rolling_24h' ? ` · quota ${count(q.used)} of ${count(q.max)}` : q.kind === 'monthly_plan' ? ` · plan ${count(q.used)} of ${count(q.max)}` : ''
  const takesBroadcast = silent.broadcastShareHeld && card.roles.some((r) => r.stream === 'broadcast' && r.role !== 'fallback')
  return (
    <div className={cn('flex flex-wrap items-center gap-5 px-[22px] py-4', !last && 'border-b border-line-2')}>
      <div className="flex flex-[1_1_260px] flex-wrap items-center gap-2.5">
        <Name size={16}>{card.name}</Name>
        <StatusBadge card={card} />
        <span className="k">
          {kind}
          {event} · error rate {pct(card.errorRate7d ?? 0)}
        </span>
      </div>
      <div className="k flex-[1_1_260px]">
        Bounce {bouncePct(card.bounce.value)} of {limitPct(card.bounce.limit)} · complaint {complaintPct(card.complaint.value)} of {limitPct(card.complaint.limit)}
        {quota}
      </div>
      {takesBroadcast ? (
        <Badge tone="info">
          {silent.broadcastPolicy === 'automatic'
            ? `Taking the held ${silent.short} share`
            : `Taking the held ${silent.short} share? No: Broadcasts failover is ${silent.broadcastPolicy === 'warm_only' ? 'warmed-up only' : 'pause-and-ask'}`}
        </Badge>
      ) : null}
    </div>
  )
}
