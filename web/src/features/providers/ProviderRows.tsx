import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Grid } from '@/components/ui/Layout'
import { t as tt, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useNow } from '@/lib/hooks'
import { andList, clock, clockSeconds, count, dayMonthDate, money, monthName, pct, weekdayName } from '@/lib/format'
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
import { m } from './messages'
import { mo } from './overview.messages'
import { SendingIPsFact } from './SendingIPs'

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
  const t = useT()
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
                {t(mo.roleActiveFallback, { stream: s })}
              </Badge>
            )
          }
          return <Badge key={r.stream}>{t(mo.roleFallback, { stream: s })}</Badge>
        }
        const state =
          !down && (r.held || (silentHeld && r.stream === 'broadcast'))
            ? 'held'
            : outage && outage.providerId !== card.id && outage.streams.some((x) => x.stream === r.stream && x.state === 'waiting')
              ? 'sending'
              : 'none'
        return (
          <Badge key={r.stream} tone="purple">
            {t(mo.roleCarries, { role: r.role, stream: s, share: pct(r.percent, 0), state })}
          </Badge>
        )
      })}
    </>
  )
}

/** A webhook's kind in the reader's words: "Registered" and "Bounce mailbox" are words, SNS and Event Grid product names. */
function webhookKind(kind: string): string {
  if (kind === 'Registered') return tt(mo.webhookRegistered)
  if (kind === 'Bounce mailbox') return tt(mo.webhookBounceMailbox)
  return kind
}

function webhookText(card: ProviderCard, withKind = true) {
  const kind = withKind ? webhookKind(card.webhook.kind) : ''
  return tt(mo.webhookLine, {
    hasKind: kind ? 'yes' : 'no',
    kind,
    hasEvent: card.webhook.lastEventAt ? 'yes' : 'no',
    ago: card.webhook.lastEventAt ? agoSeconds(card.webhook.lastEventAt) : '',
  })
}

function quotaMeter(card: ProviderCard, mode: 'normal' | 'outage' | 'silent', accepted?: number): ReactNode {
  const q = card.quota
  if (q.kind === 'none' || q.max <= 0) return null
  const share = Math.round((q.used / q.max) * 100)
  if (q.kind === 'warmup_daily') {
    let value = tt(mo.warmupSentToday, { sent: count(q.used), weekday: weekdayName(new Date()) })
    let used = q.used
    let color = 'var(--accent)'
    if (mode === 'outage') value = tt(mo.warmupSentShare, { sent: count(q.used), share: pct(share, 0) })
    if (mode === 'silent') {
      used = accepted ?? q.used
      value = tt(mo.acceptedUnknown, { count: count(used) })
      color = 'var(--warn-bar)'
    }
    return <Meter label={tt(mo.warmupCap, { cap: count(q.max) })} value={value} percent={(used / q.max) * 100} color={color} />
  }
  const label = q.kind === 'rolling_24h' ? tt(mo.quota24h, { max: count(q.max) }) : tt(mo.quotaPlan, { max: count(q.max) })
  const value =
    q.fromFailover > 0
      ? tt(mo.quotaUsedFailover, { used: count(q.used), share: pct(share, 0), failover: count(q.fromFailover) })
      : tt(mo.quotaUsed, { used: count(q.used), share: pct(share, 0) })
  return <Meter label={label} value={value} percent={share} color="var(--accent)" />
}

function rateMeter(card: ProviderCard, mode: 'normal' | 'outage' = 'normal'): ReactNode {
  const r = card.rate
  const ramp = r.label === 'ramp limit'
  const label = r.max > 0 ? (ramp ? tt(mo.rateRamp, { rate: perSec(r.max) }) : tt(mo.rateMax, { rate: perSec(r.max) })) : tt(mo.rateNone)
  if (card.status === 'down') {
    return <Meter label={label} value={tt(mo.rateThrottled, { rate: perSec(r.now) })} percent={0} color="var(--accent)" valueClassName="text-bad-fg" />
  }
  const value =
    ramp || mode === 'outage'
      ? mode === 'outage'
        ? tt(mo.rateOnly, { rate: perSec(r.now, true) })
        : tt(mo.rateNow, { rate: perSec(r.now) })
      : tt(mo.rateNowPeak, { rate: perSec(r.now), peak: perSec(r.peakToday) })
  return <Meter label={label} value={value} percent={r.max > 0 ? (r.now / r.max) * 100 : 0} color="var(--accent)" />
}

/** Why a provider's complaint rate is n/a (review F-3). */
function rateReason(reason: Schemas['ProviderLimitRate']['reason']): string | undefined {
  if (reason === 'not_reported') return tt(mo.rateNotReported)
  if (reason === 'no_data') return tt(mo.rateNoData)
  return undefined
}

function rateMeters(card: ProviderCard, stale = false, short = false) {
  const b = card.bounce
  const c = card.complaint
  const withStale = (value: string, has: boolean) => (stale && has ? tt(mo.staleValue, { value }) : value)
  return (
    <>
      <Meter
        label={limitLabel('Bounce', b, short)}
        value={b.value !== undefined ? withStale(bouncePct(b.value), true) : b.reason ? tt(mo.rateNA) : bouncePct(b.value)}
        note={b.value === undefined && b.reason === 'no_data' ? tt(mo.rateNoSends) : undefined}
        percent={b.value !== undefined ? (b.value / b.limit) * 100 : 0}
        color={rateColor(b.value, b.limit, stale)}
      />
      <Meter
        label={limitLabel('Complaint', c, short)}
        value={c.value !== undefined ? withStale(complaintPct(c.value), true) : c.reason ? tt(mo.rateNA) : complaintPct(c.value)}
        percent={c.value !== undefined ? (c.value / c.limit) * 100 : 0}
        color={rateColor(c.value, c.limit, stale)}
        note={c.value === undefined ? rateReason(c.reason) : undefined}
      />
    </>
  )
}

function CostGrid({ card, monthStart }: { card: ProviderCard; monthStart: string }) {
  const t = useT()
  const c = card.cost
  const mon = monthName(monthStart)
  const thousand = count(1000)
  let left: string
  let rightLabel: string
  if (c.basis === 'plan') {
    left = t(mo.costPlan, { amount: money(c.monthly ?? c.toDate) })
    rightLabel = t(mo.per1000Usage, { thousand })
  } else if (c.basis === 'self_hosted') {
    left = t(mo.costMetered, { amount: money(0, 0) })
    rightLabel = c.monthly ? t(mo.per1000Server, { thousand, amount: money(c.monthly, 0) }) : t(mo.per1000SelfHosted, { thousand })
  } else {
    left = money(c.toDate)
    rightLabel = c.source ? t(mo.per1000From, { thousand, source: c.source }) : t(mo.per1000FromPricing, { thousand })
  }
  return (
    <Grid cols={2} gap={8} className="text-[13px]">
      <div>
        <div className="k">{t(mo.costToDate, { month: mon })}</div>
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
  const t = useT()
  return (
    <div className="flex flex-[1_1_300px] flex-col gap-2">
      <span className="k">{t(mo.capabilities)}</span>
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
  const t = useT()
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
          <SendingIPsFact card={card} />
          <Fact className="pv-fact" k={t(mo.factWebhooks)}>
            <span>{webhookText(card)}</span>
          </Fact>
          <Fact className="pv-fact" k={t(mo.factErrorRate7d)}>
            <span className="n">{pct(card.errorRate7d ?? 0)}</span>
          </Fact>
          <Fact className="pv-fact" k={card.accountKind === 'server' ? t(mo.factServer) : t(mo.factAccountStatus)}>
            <span>{card.account || '—'}</span>
          </Fact>
          {card.tenant ? (
            // PR-11: the workspace sends as its own SES tenant.
            <p className="m-0 mt-0.5 rounded-[8px] border border-line bg-ground-3 px-2.5 py-2">
              <span className="k">{t(mo.factTenant)}</span> <span className="font-semibold">{card.tenant.name}</span> ·{' '}
              {card.tenant.policy ? t(mo.tenantLine, { policy: card.tenant.policy }) : t(mo.tenantLineNoPolicy)}
            </p>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-2">
          {card.warmup ? (
            <ButtonLink size="sm" to="/w/$workspace/providers/warmup" params={{ workspace: slug }}>
              {t(mo.warmupPlan)}
            </ButtonLink>
          ) : (
            <ButtonLink size="sm" to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
              {t(mo.domains)}
            </ButtonLink>
          )}
          <ButtonLink size="sm" to="/w/$workspace/providers/events" params={{ workspace: slug }} search={{ provider: card.id }}>
            {t(mo.events)}
          </ButtonLink>
          <Button size="sm" onClick={onSettings} aria-label={t(mo.settingsLabel, { name: card.name })}>
            {t(mo.settings)}
          </Button>
        </div>
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-3">
        {rateMeters(card)}
        {quotaMeter(card, 'normal')}
        {card.warmup?.transactionalOverCap ? <span className="k">{t(mo.warmupOverCap, { count: card.warmup.transactionalOverCap })}</span> : null}
        {rateMeter(card)}
        <CostGrid card={card} monthStart={monthStart} />
      </div>
      <Capabilities card={card} />
    </RowShell>
  )
}

/** "09:15:52 503 ServiceUnavailable · SendEmail": a call as the provider's log writes it. */
export function errorLine(e: Schemas['ProviderCallError']): string {
  const at = clockSeconds(e.at)
  if (e.status >= 500) return `${at} ${e.status} ${e.code || 'error'} · ${e.op}`
  return `${at} ${e.code || e.status} · ${e.message || e.op}`
}

/** The provider that is down, during an outage. */
export function DownRow({ card, outage }: { card: ProviderCard; outage: Schemas['ProviderOutage'] }) {
  const t = useT()
  const down = outage.down
  return (
    <RowShell className="bg-bad-tint">
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <Name>{card.name}</Name>
          <Badge tone="bad" dot>
            {t(m.statusDownSince, { since: sinceText(outage.since) })}
          </Badge>
          {card.label ? <Badge>{card.label}</Badge> : null}
        </div>
        <div className="flex flex-wrap gap-1.5">
          <RoleBadges card={card} down />
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact className="pv-fact" k={t(mo.factErrorRate3m)}>
            {outage.errorRate !== undefined ? <span className="n text-bad-fg">{pct(Math.round(outage.errorRate), 0)}</span> : <span className="k">{t(mo.errorRateNoCalls)}</span>}
          </Fact>
          <Fact className="pv-fact" k={t(mo.factLastSuccess)}>
            <span>{down.lastSuccessAt ? clockSeconds(down.lastSuccessAt) : t(mo.noneRecorded)}</span>
          </Fact>
          <Fact className="pv-fact" k={t(mo.factWebhooks)}>
            <span>{t(mo.webhookBefore, { webhook: webhookText(card), time: clock(outage.since) })}</span>
          </Fact>
          <Fact className="pv-fact" k={t(mo.factAccountStatus)}>
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
        <span className="k">{t(mo.recentErrors)}</span>
        <div className="mono flex flex-col gap-1 text-[12px] text-bad-fg">
          {down.errors.length ? down.errors.slice(0, 4).map((e, i) => <span key={i}>{errorLine(e)}</span>) : <span>{t(mo.noErrorDetails)}</span>}
        </div>
      </div>
    </RowShell>
  )
}

/** Another provider while one is down: how it is affected and what it carries. */
export function OutageRow({ card, outage, last }: { card: ProviderCard; outage: Schemas['ProviderOutage']; last?: boolean }) {
  const t = useT()
  const forecast = outage.forecasts.find((f) => f.providerId === card.id)
  const whyNot = outage.whyNot.find((w) => w.providerId === card.id)
  const taking = card.takingFailover.map((s) => streamName[s as Stream])
  return (
    <RowShell last={last}>
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <Name>{card.name}</Name>
          <StatusBadge card={card} />
          {taking.length ? <Badge tone="info">{t(mo.takingFailover, { streams: andList(taking) })}</Badge> : null}
        </div>
        <div className="flex flex-wrap gap-1.5">
          <RoleBadges card={card} outage={outage} />
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact className="pv-fact" k={t(mo.factWebhooks)}>
            <span>{webhookText(card, false)}</span>
          </Fact>
          <Fact className="pv-fact" k={t(mo.factErrorRate3m)}>
            {card.errorRate3m !== undefined ? <span className="n">{pct(card.errorRate3m)}</span> : <span className="k">{t(mo.errorRateNoCalls)}</span>}
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
          <span className="k">{t(mo.forecastTitle, { provider: outage.short })}</span>
          <span className="text-[13px]">
            {t(mo.forecastBody, {
              stream: streamName[forecast.stream as Stream],
              perDay: count(forecast.perDay),
              runsOut: forecast.runsOutOn ? 'yes' : 'no',
              provider: card.short,
              date: forecast.runsOutOn ? dayMonthDate(forecast.runsOutOn) : '',
              nextTier: forecast.nextTierAdds ? 'yes' : 'no',
              adds: count(forecast.nextTierAdds ?? 0),
              price: money(forecast.nextTierPrice ?? 0, 0),
            })}
          </span>
        </div>
      ) : whyNot ? (
        <div className="flex flex-[1_1_300px] flex-col gap-1.5">
          <span className="k">{t(mo.whyNotTitle, { provider: card.short })}</span>
          <span className="text-[13px]">
            {t(mo.whyNotBody, {
              held: count(whyNot.held),
              over: whyNot.multiple >= 1 ? 'yes' : 'no',
              times: count(Math.round(whyNot.multiple)),
              share: pct(Math.round(whyNot.multiple * 100), 0),
              push: whyNot.rate > 0 ? 'yes' : 'no',
              days: whyNot.ageDays,
              rate: perSec(whyNot.rate),
            })}
          </span>
        </div>
      ) : (
        <div className="flex-[1_1_300px]" />
      )}
    </RowShell>
  )
}

/** The provider whose webhooks went quiet. */
export function SilentRow({ card, silent }: { card: ProviderCard; silent: Schemas['ProviderWebhooksSilent'] }) {
  const t = useT()
  const now = useNow()
  return (
    <RowShell className="bg-warn-tint">
      <div className="flex flex-[1_1_260px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <Name>{card.name}</Name>
          <Badge tone="warn" dot>
            {t(m.statusDegradedSilent, { since: sinceText(silent.since) })}
          </Badge>
          {card.label ? <Badge>{card.label}</Badge> : null}
        </div>
        <div className="flex flex-wrap gap-1.5">
          <RoleBadges card={card} silentHeld={silent.broadcastShareHeld} />
          {card.warmup ? (
            <Badge tone="info">
              {silent.broadcastShareHeld
                ? t(mo.warmingUpDayPaused, { day: card.warmup.day, days: card.warmup.days })
                : t(m.statusWarmingUpDay, { day: card.warmup.day, days: card.warmup.days })}
            </Badge>
          ) : null}
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <Fact className="pv-fact" k={t(mo.factWebhooks)}>
            <span className="font-semibold text-warn-fg">{t(mo.silentLastEvent, { time: clock(silent.since), since: sinceText(silent.since) })}</span>
          </Fact>
          <Fact className="pv-fact" k={t(mo.factErrorRate7d)}>
            <span className="n">{pct(card.errorRate7d ?? 0)}</span>
          </Fact>
          <SendingIPsFact card={card} />
        </div>
      </div>
      <div className="flex flex-[1_1_260px] flex-col gap-3">
        {rateMeters(card, true)}
        {quotaMeter(card, 'silent', silent.accepted)}
      </div>
      <div className="flex flex-[1_1_300px] flex-col gap-1.5">
        <span className="k">{t(mo.freshnessTitle)}</span>
        <Freshness silent={silent} />
        <div className="k flex justify-between">
          <span>{t(mo.freshnessYesterday, { time: clock(now.getTime() - 24 * 3600_000) })}</span>
          <span>{clock(silent.since)}</span>
          <span>{t(mo.freshnessNow)}</span>
        </div>
      </div>
    </RowShell>
  )
}

/** Events per hour for the last 24 hours; silent hours drawn as stubs. */
function Freshness({ silent }: { silent: Schemas['ProviderWebhooksSilent'] }) {
  const t = useT()
  const max = Math.max(1, ...silent.hourly)
  const silentFrom = new Date(silent.since).getTime()
  const from = new Date(silent.hourlyFrom).getTime()
  const quietHours = silent.hourly.filter((_, i) => from + i * 3600_000 >= silentFrom - 3600_000 && silent.hourly[i] === 0).length
  return (
    <div className="flex h-12 items-end gap-0.5" role="img" aria-label={t(mo.freshnessLabel, { count: quietHours })}>
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
  const t = useT()
  const kind = card.webhook.kind && card.webhook.kind !== 'Registered' ? webhookKind(card.webhook.kind) : ''
  const q = card.quota
  const takesBroadcast = silent.broadcastShareHeld && card.roles.some((r) => r.stream === 'broadcast' && r.role !== 'fallback')
  return (
    <div className={cn('flex flex-wrap items-center gap-5 px-[22px] py-4', !last && 'border-b border-line-2')}>
      <div className="flex flex-[1_1_260px] flex-wrap items-center gap-2.5">
        <Name size={16}>{card.name}</Name>
        <StatusBadge card={card} />
        <span className="k">
          {t(mo.compactLine, {
            hasKind: kind ? 'yes' : 'no',
            kind,
            hasEvent: card.webhook.lastEventAt ? 'yes' : 'no',
            ago: card.webhook.lastEventAt ? agoSeconds(card.webhook.lastEventAt) : '',
            rate: pct(card.errorRate7d ?? 0),
          })}
        </span>
      </div>
      <div className="k flex-[1_1_260px]">
        {t(mo.compactRates, {
          bounce: bouncePct(card.bounce.value),
          bounceLimit: limitPct(card.bounce.limit),
          complaint: card.complaint.value !== undefined || !card.complaint.reason ? complaintPct(card.complaint.value) : t(mo.rateNA),
          complaintLimit: limitPct(card.complaint.limit),
          quota: q.kind === 'rolling_24h' ? 'rolling' : q.kind === 'monthly_plan' ? 'plan' : 'none',
          used: count(q.used),
          max: count(q.max),
        })}
      </div>
      {takesBroadcast ? (
        <Badge tone="info">
          {silent.broadcastPolicy === 'automatic'
            ? t(mo.takingHeld, { provider: silent.short })
            : t(mo.takingHeldNo, { provider: silent.short, policy: silent.broadcastPolicy })}
        </Badge>
      ) : null}
    </div>
  )
}
