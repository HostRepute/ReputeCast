import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { problemReference } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Table, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { formatList, T, useT } from '@/i18n'
import { ago, clock, count, dayMonthDate, dayMonthYear, monthName, money, pct } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import {
  BudgetMeter,
  Capabilities,
  CountryChips,
  Masked,
  price,
  ProviderDot,
  providerColor,
  ReadOnlyNote,
  regionLabel,
  RegistrationBadges,
  SmsTabs,
  throwIfForbidden,
  useKindName,
  useRefreshSms,
} from './common'
import { m as sm } from './messages'
import { ProviderDialog } from './ProviderDialog'
import { m } from './providers.messages'
import { RegistrationDialog } from './Registration'
import { AddSenderIdDialog, ImportSendersDialog } from './SenderDialogs'

type Overview = Schemas['SmsOverview']
type Card = Schemas['SmsProviderCard']
type Sender = Schemas['SmsSenderRow']

/** Sending providers › SMS › Providers (SmsProviders, SmsStates). */
export function SmsProvidersPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } }, { refetchInterval: 60_000 })
  throwIfForbidden(q.error)
  const [settings, setSettings] = useState<{ id: string; focus?: 'credentials' } | null>(null)
  const [registration, setRegistration] = useState<string | null>(null)
  const [addSenderId, setAddSenderId] = useState(false)
  const [importing, setImporting] = useState(false)
  const data = q.data
  const empty = data && data.providers.length === 0
  const card = settings && data?.providers.find((p) => p.id === settings.id)
  return (
    <Page title={t(sm.pageTitle)} filters={<SmsTabs />} actions={data && !empty ? <HeaderActions data={data} /> : null}>
      {q.isError ? (
        <ErrorPanel title={t(m.errorTitle)} reference={problemReference(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.errorText)}
        </ErrorPanel>
      ) : q.isPending ? (
        <ProvidersSkeleton />
      ) : empty ? (
        <EmptyProviders />
      ) : (
        <Body
          data={data!}
          onSettings={(id, focus) => setSettings({ id, focus })}
          onRegistration={setRegistration}
          onAddSenderId={() => setAddSenderId(true)}
          onImport={() => setImporting(true)}
        />
      )}
      {card ? <ProviderDialog card={card} focus={settings?.focus} open={Boolean(settings)} onOpenChange={(o) => !o && setSettings(null)} /> : null}
      {registration ? <RegistrationDialog senderId={registration} open onOpenChange={(o) => !o && setRegistration(null)} /> : null}
      {data && addSenderId ? <AddSenderIdDialog providers={data.providers} open onOpenChange={setAddSenderId} /> : null}
      {data && importing ? <ImportSendersDialog providers={data.providers} open onOpenChange={setImporting} /> : null}
    </Page>
  )
}

function AddProviderButton({ size }: { size?: 'sm' | 'md' }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('providers:manage')
  return reason ? (
    <Button variant="primary" size={size} disabledReason={reason}>
      {t(m.addProvider)}
    </Button>
  ) : (
    <ButtonLink variant="primary" size={size} to="/w/$workspace/providers/connect/sms" params={{ workspace: slug }}>
      {t(m.addProvider)}
    </ButtonLink>
  )
}

function HeaderActions({ data }: { data: Overview }) {
  const t = useT()
  const attention = data.providers.filter((p) => p.status !== 'healthy').length
  return (
    <>
      {attention ? (
        <Badge tone={data.providers.some((p) => p.status === 'down' || p.status === 'credentials_rejected') ? 'bad' : 'warn'} dot>
          {t(m.needAttention, { count: attention })}
        </Badge>
      ) : data.checkedAt ? (
        <span className="k">{t(m.checked, { count: data.providers.length, ago: ago(data.checkedAt) })}</span>
      ) : null}
      <AddProviderButton />
    </>
  )
}

function Body({
  data,
  onSettings,
  onRegistration,
  onAddSenderId,
  onImport,
}: {
  data: Overview
  onSettings: (id: string, focus?: 'credentials') => void
  onRegistration: (senderId: string) => void
  onAddSenderId: () => void
  onImport: () => void
}) {
  const t = useT()
  const order = data.providers.map((p) => p.name)
  const unhealthy = data.providers.filter((p) => p.status !== 'healthy')
  const tl = data.tiles
  return (
    <>
      {unhealthy.length ? (
        <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(420px, 100%), 1fr))' }}>
          {unhealthy.map((c) => (
            <HealthCard key={c.id} card={c} canManage={data.canManage} onCredentials={() => onSettings(c.id, 'credentials')} />
          ))}
        </div>
      ) : null}
      {tl.budget > 0 && tl.spend >= tl.budget ? <BudgetReached data={data} /> : null}
      <Tiles data={data} />
      <section className="card flex flex-col" aria-label={t(m.providersLabel)}>
        {data.providers.map((c, i) => (
          <ProviderRow key={c.id} card={c} color={providerColor(order, c.name)} month={monthName(tl.month, 'long')} canManage={data.canManage} last={i === data.providers.length - 1} onSettings={() => onSettings(c.id)} />
        ))}
        <div className="sms-foot" style={{ padding: '12px 22px' }}>
          <span className="k">
            {data.notConnected.length ? t(m.notConnected, { names: data.notConnected.map((n) => n.name).join(' · ') }) : t(m.allConnected)}
          </span>
          {data.canManage && data.notConnected.length ? <AddProviderLink /> : null}
        </div>
      </section>
      {!data.canManage ? (
        <ReadOnlyNote>
          {t(m.readOnlyText, { helpers: data.helpers.length ? formatList(data.helpers, 'disjunction') : 'none' })}
        </ReadOnlyNote>
      ) : null}
      <Senders data={data} order={order} onRegistration={onRegistration} onAddSenderId={onAddSenderId} onImport={onImport} />
      <RoutingSummary data={data} order={order} />
    </>
  )
}

function AddProviderLink() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Link to="/w/$workspace/providers/connect/sms" params={{ workspace: slug }} className="sms-lnk">
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
        <path d="M8 3v10M3 8h10" />
      </svg>
      {t(m.addProvider)}
    </Link>
  )
}

/** The four tiles: texts sent, delivered, spend against the budget, the suppression list. */
function Tiles({ data }: { data: Overview }) {
  const t = useT()
  const tl = data.tiles
  const month = monthName(tl.month, 'long')
  const kinds: string[] = []
  const by = tl.byKind
  if (by.confirmation) kinds.push(t(m.kindConfirmation, { count: by.confirmation }))
  if (by.campaign) kinds.push(t(m.kindCampaign, { count: by.campaign }))
  if (by.automation) kinds.push(t(m.kindAutomation, { count: by.automation }))
  if (by.test) kinds.push(t(m.kindTest, { count: by.test }))
  const sb = tl.suppressedBy
  const opted = sb.opted_out ?? 0
  const dead = (sb.landline ?? 0) + (sb.not_in_service ?? 0) + (sb.invalid ?? 0)
  const other = tl.suppressed - opted - dead
  const sup: string[] = []
  if (opted) sup.push(t(m.supOptedOut, { count: count(opted) }))
  if (dead) sup.push(t(m.supDead, { count: dead }))
  if (other > 0) sup.push(t(m.supOther, { count: other }))
  const whole = (v: number) => money(v, Number.isInteger(v) ? 0 : 2)
  return (
    <div className="sms-stats">
      <div className="card sms-stat">
        <span className="k">{t(m.tileSent, { month })}</span>
        <span className="v">{count(tl.sent)}</span>
        <span className="s">{kinds.length ? kinds.join(' · ') : t(m.noTextsYet)}</span>
      </div>
      <div className="card sms-stat">
        <span className="k">{t(m.tileDelivered)}</span>
        <span className="v">{tl.sent ? pct(tl.deliveredPct) : '–'}</span>
        <span className="s">{t(m.deliveredSub, { delivered: count(tl.delivered), sent: count(tl.sent), failed: count(tl.failed) })}</span>
      </div>
      <div className="card sms-stat" style={{ gap: 4 }}>
        <span className="k">{t(m.tileSpend, { month })}</span>
        <span className="v">{money(tl.spend)}</span>
        {tl.budget > 0 ? <BudgetMeter spent={tl.spend} budget={tl.budget} alertPct={tl.alertPct} month={month} /> : null}
        <span className="s">
          {[tl.budget > 0 ? t(m.spendSub, { budget: whole(tl.budget), alert: tl.alertPct }) : t(m.noBudget), ...tl.spendBy.map((s) => `${s.name} ${money(s.spend)}`)].join(' · ')}
        </span>
      </div>
      <div className="card sms-stat">
        <span className="k">{t(m.tileSuppressed)}</span>
        <span className="v">{count(tl.suppressed)}</span>
        <span className="s">{sup.length ? sup.join(' · ') : t(m.supNone)}</span>
      </div>
    </div>
  )
}

const statusTone = { healthy: 'ok', degraded: 'warn', down: 'bad', credentials_rejected: 'bad' } as const
const statusWord = { healthy: sm.statusHealthy, degraded: sm.statusDegraded, down: sm.statusDown, credentials_rejected: sm.statusRejected } as const

function accountLine(card: Card): ReactNode {
  const mono = (c: ReactNode[]) => (
    <span className="mono" style={{ fontSize: 12.5 }}>
      {c}
    </span>
  )
  const msg = { twilio: m.acctTwilio, vonage: m.acctVonage, plivo: m.acctPlivo, telnyx: m.acctTelnyx, awseum: m.acctAwseum }[card.adapter] ?? m.acctOther
  return <T msg={msg} values={{ hint: card.accountHint, mono }} />
}

const kindWord = { toll_free: m.kindWordToll_free, '10dlc': m.kindWord10dlc, local: m.kindWordLocal, mobile: m.kindWordMobile } as const

function sendersLine(card: Card, t: ReturnType<typeof useT>): string {
  const parts: string[] = []
  const numbers = card.senderKinds.filter((k) => k !== 'sender_id') as (keyof typeof kindWord)[]
  // Toll-free first, as people say it.
  numbers.sort((a, b) => (a === 'toll_free' ? -1 : b === 'toll_free' ? 1 : 0))
  if (numbers.length) {
    // Sentence case for the line, wherever the language puts the kinds.
    const line = t(m.sendersNumbers, { kinds: formatList(numbers.map((k) => t(kindWord[k]))), count: card.senderCount - card.senderIds.length })
    parts.push(line.charAt(0).toUpperCase() + line.slice(1))
  }
  const ids = [...new Set(card.senderIds)]
  if (ids.length) parts.push(t(m.sendersIds, { ids: formatList(ids), count: card.senderIdCountries }))
  return parts.length ? parts.join(' · ') : t(m.sendersNone)
}

/** The priced countries in the order the routes name them: US, CA. */
function inRouteOrder(card: Card): string[] {
  const order = [...card.countries, ...card.held]
  const at = (c: string) => (order.includes(c) ? order.indexOf(c) : order.length)
  return [...card.priceCountries].sort((a, b) => at(a) - at(b))
}

/** A metric over its pause limit: green, amber near it, red past it. */
function Meter({ label, value, share }: { label: string; value: string; share: number }) {
  const w = Math.max(0, Math.min(100, share))
  const color = share >= 100 ? '#B91C1C' : share >= 60 ? '#B45309' : '#15803D'
  return (
    <div className="sms-mt">
      <div className="l">
        <span className="k">{label}</span>
        <span className="n whitespace-nowrap">{value}</span>
      </div>
      <div className="bar">
        <i style={{ width: `${w.toFixed(0)}%`, background: color }} />
      </div>
    </div>
  )
}

function ProviderRow({
  card,
  color,
  month,
  canManage,
  last,
  onSettings,
}: {
  card: Card
  color: string
  month: string
  canManage: boolean
  last: boolean
  onSettings: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const receiptMinutes = card.lastReceiptAt ? (now.getTime() - new Date(card.lastReceiptAt).getTime()) / 60_000 : undefined
  const minPrice = card.priceMin
  const maxPrice = card.priceMax
  return (
    <div className="flex flex-wrap gap-5 px-[22px] py-5" style={last ? undefined : { borderBottom: '1px solid var(--line-2)' }}>
      <div className="flex min-w-0 flex-[1_1_280px] flex-col gap-2.5">
        <div className="flex flex-wrap items-center gap-2.5">
          <span className="inline-flex items-center gap-2 text-[17px] font-bold">
            <span className="dot" style={{ color }} aria-hidden="true" />
            {card.name}
          </span>
          <Badge tone={statusTone[card.status]} dot>
            {t(statusWord[card.status])}
          </Badge>
          {canManage ? (
            <Badge tone="neutral">
              {!card.testMode ? card.accountLabel : card.live ? t(m.accountLive, { account: card.accountLabel }) : t(m.accountTest, { account: card.accountLabel })}
            </Badge>
          ) : (
            <Badge tone="neutral">{t(sm.readOnly)}</Badge>
          )}
        </div>
        {card.countries.length || card.held.length ? (
          <div className="flex flex-wrap gap-1.5">
            {card.countries.length ? <Badge tone="purple">{regionLabel(card.countries)}</Badge> : null}
            {card.held.length ? <Badge tone="warn">{t(m.held, { countries: regionLabel(card.held) })}</Badge> : null}
          </div>
        ) : null}
        <div className="sms-dl">
          <div>
            <span className="k">{t(m.factAccount)}</span>
            <span>{canManage ? accountLine(card) : <Masked />}</span>
          </div>
          {!canManage
            ? card.secretLabels.map((l) => (
                <div key={l}>
                  <span className="k">{l}</span>
                  <span>
                    <Masked />
                  </span>
                </div>
              ))
            : null}
          <div>
            <span className="k">{t(m.factSenders)}</span>
            <span>{sendersLine(card, t)}</span>
          </div>
          <div>
            <span className="k">{t(m.factWebhooks)}</span>
            <span>{card.adapter === 'twilio' ? t(m.webhooksTwilio, { mode: card.webhooks }) : t(m.webhooksOther, { mode: card.webhooks })}</span>
          </div>
          <div>
            <span className="k">{t(m.factConnected)}</span>
            <span>{card.connectedBy ? t(m.connectedOn, { date: dayMonthYear(card.connectedAt), name: card.connectedBy }) : dayMonthYear(card.connectedAt)}</span>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <a href="#senders" className="btn sm">
            {t(m.btnSenders)}
          </a>
          <ButtonLink size="sm" to="/w/$workspace/providers/sms/events" params={{ workspace: slug }} search={{ provider: card.id }}>
            {t(m.btnEvents)}
          </ButtonLink>
          <Button size="sm" onClick={onSettings}>
            {t(m.btnSettings)}
          </Button>
        </div>
      </div>
      <div className="flex min-w-0 flex-[1_1_260px] flex-col gap-3">
        <Meter label={t(m.metricReceipt)} value={card.lastReceiptAt ? ago(card.lastReceiptAt) : t(m.receiptNever)} share={receiptMinutes === undefined ? 0 : (receiptMinutes / 120) * 100} />
        <Meter label={t(m.metricFailed, { limit: card.pauseFailed })} value={pct(card.failedPct)} share={card.pauseFailed ? (card.failedPct / card.pauseFailed) * 100 : 0} />
        <Meter label={t(m.metricFiltered, { limit: card.pauseFiltered })} value={pct(card.filteredPct)} share={card.pauseFiltered ? (card.filteredPct / card.pauseFiltered) * 100 : 0} />
        <div className="grid grid-cols-2 gap-2 text-[13px]">
          <div>
            <div className="k">{t(m.spendMonth, { month })}</div>
            <div className="n">{money(card.spend)}</div>
          </div>
          <div>
            <div className="k">{t(m.perSegment)}</div>
            <div className="n">
              {minPrice === maxPrice ? price(minPrice) : `${price(minPrice)}–${price(maxPrice)}`}{' '}
              {minPrice === maxPrice && card.priceCountries.length ? <span className="k">{inRouteOrder(card).join(', ')}</span> : null}
            </div>
          </div>
        </div>
      </div>
      <div className="flex min-w-0 flex-[1_1_280px] flex-col gap-2">
        <span className="k">{t(m.capsTitle)}</span>
        <Capabilities badges={card.badges} />
      </div>
    </div>
  )
}

/** A provider that needs attention: credentials rejected, down, receipts silent. */
function HealthCard({ card, canManage, onCredentials }: { card: Card; canManage: boolean; onCredentials: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const path = { workspace: slug, providerId: card.id }
  const webhooks = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/webhooks', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.registered, { name: card.name }) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/check', {
    onSuccess: () => void refresh(),
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const reason = usePermissionReason('providers:manage')
  const since = card.since ?? card.lastCheckedAt
  const countries = card.held.length ? regionLabel(card.held) : card.countries.length ? regionLabel(card.countries) : 'none'
  const silent = card.status === 'degraded'
  return (
    <section className="card min-w-0" style={{ borderColor: silent ? '#FCD34D' : '#FCA5A5' }}>
      <div className="sec">
        <div className="flex flex-wrap items-center gap-2.5">
          <span className="text-[16px] font-bold">{card.name}</span>
          <Badge tone={statusTone[card.status]} dot>
            {t(statusWord[card.status])}
          </Badge>
        </div>
        <span className="k">
          {silent
            ? card.lastReceiptAt
              ? t(m.lastReceiptAt, { time: clock(card.lastReceiptAt) })
              : null
            : since
              ? t(m.sinceAt, { date: dayMonthDate(since), time: clock(since) })
              : null}
        </span>
      </div>
      <div className="pad flex flex-col gap-3">
        {silent ? (
          <>
            <Note tone="warn" role="note">
              {t(m.silentNote, { name: card.name, adapter: card.adapter })}
            </Note>
            <dl className="kv">
              <dt>{t(m.textsSince)}</dt>
              <dd>{t(m.textsSinceValue, { sent: count(card.sentSince), receipts: count(card.receiptsSince) })}</dd>
              <dt>{t(m.callback)}</dt>
              <dd>
                <span className="mono text-[12px] [overflow-wrap:anywhere]">{card.webhookUrl}</span>
              </dd>
              <dt>{t(m.effect)}</dt>
              <dd>{t(m.effectValue)}</dd>
            </dl>
            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="primary" disabledReason={reason} loading={webhooks.isPending} onClick={() => webhooks.mutate({ params: { path } })}>
                {t(m.registerAgain)}
              </Button>
              {card.console ? (
                <ButtonAnchor size="sm" href={card.console} newTab>
                  {t(m.openConsole, { name: card.name })}
                </ButtonAnchor>
              ) : null}
            </div>
          </>
        ) : (
          <>
            <Note tone="bad">
              {since
                ? t(card.status === 'down' ? m.downNote : m.rejectedNote, { name: card.name, date: dayMonthDate(since), time: clock(since), countries })
                : null}
            </Note>
            <dl className="kv">
              {card.answer ? (
                <>
                  <dt>{t(m.answer)}</dt>
                  <dd>
                    <span className="mono">{card.answer}</span>
                  </dd>
                </>
              ) : null}
              <dt>{t(m.heldNow)}</dt>
              <dd>{t(m.heldNowValue, { count: card.heldTexts })}</dd>
              {card.told.length ? (
                <>
                  <dt>{t(m.told)}</dt>
                  <dd>{t(m.toldValue, { names: formatList(card.told) })}</dd>
                </>
              ) : null}
            </dl>
            <div className="flex flex-wrap gap-2">
              {card.status === 'credentials_rejected' ? (
                <Button size="sm" variant="primary" disabledReason={reason} onClick={onCredentials}>
                  {t(m.replaceCredentials)}
                </Button>
              ) : (
                <Button size="sm" variant="primary" loading={check.isPending} disabledReason={canManage ? undefined : reason} onClick={() => check.mutate({ params: { path } })}>
                  {t(m.retryNow)}
                </Button>
              )}
              <ButtonLink size="sm" to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }}>
                {t(m.routesBtn)}
              </ButtonLink>
            </div>
          </>
        )}
      </div>
    </section>
  )
}

/** The month's budget is used up: campaigns can't start, automation texts wait. */
function BudgetReached({ data }: { data: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const tl = data.tiles
  const month = monthName(tl.month, 'long')
  return (
    <section className="card flex min-w-0 flex-col gap-3 px-5 py-[18px]" style={{ borderColor: '#FCA5A5' }}>
      <Note tone="bad">
        <span className="flex flex-col gap-1">
          <span className="font-bold">{t(m.budgetReached)}</span>
          <span>{t(m.budgetReachedText, { month, budget: money(tl.budget), held: tl.budgetHeld })}</span>
        </span>
      </Note>
      <div className="sms-bm">
        <div className="sms-sum">
          <span className="k">{t(m.budgetLabel, { month })}</span>
          <span className="n">
            {money(tl.spend)} <span className="k">{t(m.budgetOf, { budget: money(tl.budget) })}</span>
          </span>
        </div>
        <BudgetMeter spent={tl.spend} budget={tl.budget} alertPct={tl.alertPct} month={month} legend />
      </div>
      <div className="flex flex-wrap gap-2">
        {data.canManage ? (
          <ButtonLink size="sm" variant="danger-primary" to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }} hash="budget">
            {t(m.raiseBudget)}
          </ButtonLink>
        ) : (
          <>
            <Button size="sm" variant="danger-primary" disabled>
              {t(m.raiseBudget)}
            </Button>
            <span className="k self-center">{t(m.needsManage, { names: formatList(data.helpers, 'disjunction') })}</span>
          </>
        )}
      </div>
    </section>
  )
}

/** Senders · 4: numbers and sender IDs with their registration. */
function Senders({
  data,
  order,
  onRegistration,
  onAddSenderId,
  onImport,
}: {
  data: Overview
  order: string[]
  onRegistration: (id: string) => void
  onAddSenderId: () => void
  onImport: () => void
}) {
  const t = useT()
  const kindName = useKindName()
  const reason = usePermissionReason('providers:manage')
  // By provider, then numbers before sender IDs: toll-free, 10DLC, local, mobile.
  const rows = [...data.senders].sort((a, b) => order.indexOf(a.provider) - order.indexOf(b.provider) || kindRank[a.kind] - kindRank[b.kind])
  const api = [...new Set(data.providers.filter((p) => p.badges.some((b) => b.capability === 'registration' && b.support === 'yes')).map((p) => p.name))]
  const manual = [...new Set(data.providers.filter((p) => !p.badges.some((b) => b.capability === 'registration' && b.support === 'yes')).map((p) => p.name))]
  const regLine = api.length && manual.length
    ? t(m.regBoth, { api: formatList(api), manual: formatList(manual) })
    : api.length
      ? t(m.regApiOnly, { api: formatList(api) })
      : manual.length
        ? t(m.regManualOnly, { manual: formatList(manual) })
        : ''
  const tollFree = rows.some((r) => r.kind === 'toll_free' && r.countries.some((c) => c === 'US' || c === 'CA'))
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="senders">
            {t(m.sendersTitle, { count: rows.length })}
          </h2>
          <span className="k">{t(m.sendersLead)}</span>
        </div>
      </div>
      {rows.length === 0 ? (
        <div className="pad k">{t(m.noSenders)}</div>
      ) : (
        <>
          <div className="sms-no-phone flex-col">
            <div className="sms-scroll-hint k items-center gap-1.5 px-[18px] pt-2.5">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M2 8h12M5 5 2 8l3 3M11 5l3 3-3 3" />
              </svg>
              {t(m.scrollHint)}
            </div>
            <Table className="sms-stk" caption={t(m.sendersTable)}>
              <thead>
                <tr>
                  <Th>{t(m.colSender)}</Th>
                  <Th>{t(m.colKind)}</Th>
                  <Th>{t(m.colProvider)}</Th>
                  <Th>{t(m.colCountries)}</Th>
                  <Th>{t(m.colRegistration)}</Th>
                  <Th className="r">
                    {t(m.colThroughput)}
                  </Th>
                  <Th>{t(m.colRoute)}</Th>
                  <Th>
                    <span className="vh">{t(m.colActions)}</span>
                  </Th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className="nw">
                      <span className="mono text-[13px] font-semibold">{r.label || r.value}</span>
                    </td>
                    <td className="nw">{kindName(r.kind)}</td>
                    <td className="nw">
                      <ProviderDot name={r.provider} color={providerColor(order, r.provider)} />
                    </td>
                    <td>
                      <CountryChips codes={r.countries} nowrap />
                    </td>
                    <td>
                      <div className="flex flex-col items-start gap-1">
                        <RegistrationBadges row={r} />
                      </div>
                    </td>
                    <td className="r num nw">{r.throughput && r.canSend ? t(sm.perSecond, { n: r.throughput }) : t(sm.none)}</td>
                    <td>
                      <SenderRoute row={r} />
                    </td>
                    <td className="r">
                      <Button size="sm" onClick={() => onRegistration(r.id)}>
                        {actionWord(r, t)}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
          <div className="sms-only-phone">
            {rows.map((r) => (
              <button
                key={r.id}
                type="button"
                onClick={() => onRegistration(r.id)}
                className="flex flex-col items-stretch gap-1.5 border-0 border-t border-solid border-line-2 bg-transparent px-4 py-3 text-left text-ink"
              >
                <span className="flex items-center justify-between gap-2">
                  <b className="mono text-[13px] font-semibold">{r.label || r.value}</b>
                  <CountryChips codes={r.countries} />
                </span>
                <span className="k text-[#374151]">
                  {kindName(r.kind)} · <ProviderDot name={r.provider} color={providerColor(order, r.provider)} gap={5} />
                  {r.throughput && r.canSend ? ` · ${t(sm.perSecond, { n: r.throughput })}` : ''} · {phoneRoute(r, t)}
                </span>
                <span className="flex flex-wrap gap-1.5">
                  <RegistrationBadges row={r} compact />
                </span>
              </button>
            ))}
          </div>
        </>
      )}
      <div className="sms-foot">
        <span className="k" style={{ flex: '1 1 420px' }}>
          {t(m.footSenderIds, { kind: tollFree ? 'toll_free' : 'other' })} {regLine}
        </span>
        <span className="flex flex-wrap gap-2">
          <Button size="sm" disabledReason={reason} onClick={onImport}>
            {t(m.btnImport)}
          </Button>
          <Button size="sm" disabledReason={reason} onClick={onAddSenderId}>
            {t(m.btnAddSenderId)}
          </Button>
        </span>
      </div>
    </section>
  )
}

const kindRank: Record<Sender['kind'], number> = { toll_free: 0, '10dlc': 1, local: 2, mobile: 3, sender_id: 4 }

/** Details, Registration (a 10DLC number's), or Set status (a status set by hand). */
function actionWord(r: Sender, t: ReturnType<typeof useT>): string {
  if (r.brand || r.campaign) return t(m.btnRegistration)
  if (r.manual && (r.registration.status === 'pending' || r.registration.status === 'rejected' || r.registration.status === 'not_registered')) return t(m.btnSetStatus)
  if (r.registration.status === 'pending' || r.registration.status === 'rejected') return t(m.btnRegistration)
  return t(m.btnDetails)
}

function SenderRoute({ row }: { row: Sender }) {
  const t = useT()
  if (!row.routes.length) return <span className="k">{t(sm.routeNone)}</span>
  return (
    <>
      {row.routes.map((countries) => regionLabel(countries)).join('; ')}
      {row.held ? (
        <>
          {' '}
          <span className="k">{t(sm.routeHeld)}</span>
        </>
      ) : null}
    </>
  )
}

function phoneRoute(r: Sender, t: ReturnType<typeof useT>): string {
  if (!r.routes.length) return t(m.phoneNoRoute)
  const region = r.routes.map((c) => regionLabel(c)).join('; ')
  return r.held ? t(m.phoneHeld, { region }) : region
}

/** Routing by country: each route's provider and sender, the fallback, the rest. */
function RoutingSummary({ data, order }: { data: Overview; order: string[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card overflow-hidden">
      <div className="sec">
        <h2 className="h2">{t(m.routingTitle)}</h2>
        <Link to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }} className="sms-lnk">
          {t(m.routingEdit)}
        </Link>
      </div>
      {data.routes.length === 0 ? <div className="pad k">{t(m.noRoutes)}</div> : null}
      <div className="-mt-px flex flex-wrap">
        {data.routes.map((r) => {
          const numeric = r.sender.startsWith('+')
          return (
            <div
              key={r.id}
              className="flex min-w-0 flex-[1_1_230px] flex-col gap-1.5 px-5 py-4"
              style={{ borderTop: '1px solid var(--line-2)', background: r.skipped ? '#FFFBEB' : undefined }}
            >
              <span className="font-semibold">{regionLabel(r.countries)}</span>
              <span className="inline-flex items-center gap-2 text-[13px]">
                <span className="dot" style={{ color: providerColor(order, r.provider) }} aria-hidden="true" />
                <span>
                  {r.skipped ? (
                    t(m.routePending, { provider: r.provider, sender: r.sender })
                  ) : numeric ? (
                    <T msg={m.routeVia} values={{ provider: r.provider, sender: r.sender, mono: (c) => <span className="mono">{c}</span> }} />
                  ) : (
                    t(m.routeViaId, { provider: r.provider, sender: r.sender })
                  )}
                </span>
              </span>
              <span className="k" style={r.skipped ? { color: '#92400E' } : undefined}>
                {r.skipped ? t(m.skippedUntil, { sender: r.sender }) : r.fallback ? t(m.fallbackVia, { provider: r.fallbackProvider, sender: r.fallback }) : t(m.fallbackNone)}
              </span>
            </div>
          )
        })}
        <div className="flex min-w-0 flex-[1_1_230px] flex-col gap-1.5 px-5 py-4" style={{ borderTop: '1px solid var(--line-2)' }}>
          <span className="font-semibold">{t(m.otherCountries)}</span>
          <span className="inline-flex items-center gap-2 text-[13px]">
            <span className="dot" style={{ color: '#8B94A3' }} aria-hidden="true" />
            {data.other === 'route' && data.otherSender ? data.otherSender : t(m.dontSend)}
          </span>
          <span className="k">{data.other === 'route' && data.otherSender ? t(m.elsewhereVia, { sender: data.otherSender }) : t(m.elsewhereNone)}</span>
        </div>
      </div>
    </section>
  )
}

/** No SMS provider yet: what texting gives, Add SMS provider, the terms page. */
function EmptyProviders() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="card sms-empty">
      <svg width="40" height="40" viewBox="0 0 16 16" fill="none" stroke="#0F766E" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M3 2.5h10A1.5 1.5 0 0 1 14.5 4v6a1.5 1.5 0 0 1-1.5 1.5H7.6L4.5 14v-2.5H3A1.5 1.5 0 0 1 1.5 10V4A1.5 1.5 0 0 1 3 2.5z" />
      </svg>
      <span className="tt">{t(m.emptyTitle)}</span>
      <span className="ds">{t(m.emptyText)}</span>
      <div className="flex flex-wrap justify-center gap-2.5">
        <AddProviderButton />
        <ButtonLink to="/w/$workspace/forms/sms-terms" params={{ workspace: slug }}>
          {t(m.seeTerms)}
        </ButtonLink>
      </div>
    </div>
  )
}

function ProvidersSkeleton() {
  const t = useT()
  return (
    <div className="card flex flex-col gap-3.5 p-[18px]" aria-busy="true" aria-label={t(m.loading)}>
      <div className="sms-stats">
        {[0, 1].map((i) => (
          <div key={i} className="card sms-stat gap-2">
            <Skeleton w={90} h={12} r={6} />
            <Skeleton w={60} h={22} r={6} />
            <Skeleton w={120} h={10} r={6} />
          </div>
        ))}
      </div>
      {[0, 1, 2].map((i) => (
        <div key={i} className="flex items-center gap-3.5 border-t border-line-2 py-2">
          <Skeleton w={120} h={14} r={6} />
          <Skeleton w={80} h={20} r={999} />
          <Skeleton className="flex-1" h={10} />
          <Skeleton w={56} h={28} r={6} />
        </div>
      ))}
    </div>
  )
}
