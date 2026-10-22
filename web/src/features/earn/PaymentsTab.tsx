import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select, Toggle } from '@/components/ui/Field'
import { Banner } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { ago, clock, count, dayMonthDate, dayMonthYear, logTime, monthDayTime } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN } from './EarnPage'
import { cancelReason, countryName, day, money } from './format'
import { me } from './messages'
import { ms } from './settings.messages'

type Payments = Schemas['EarnPayments']
type Conn = Schemas['EarnConnection']
type T = ReturnType<typeof useT>


/** Settings › Payments (EarnSettings): the connection, the webhook, checkout and tax, the payments event log. */
export function PaymentsTab({ pay }: { pay: Payments }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [showTest, setShowTest] = useState(false)
  const live = pay.live
  const main = live ?? pay.test
  if (!main) {
    return (
      <section className="card pad flex flex-col items-start gap-3">
        <h2 className="h2">{t(ms.notConnectedTitle)}</h2>
        <p className="m-0 max-w-[640px] text-ink-2">{t(ms.notConnectedText)}</p>
        {pay.canManage ? (
          <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }} variant="primary">
            {t(ms.connectStripe)}
          </ButtonLink>
        ) : null}
        <OtherProviders pay={pay} />
      </section>
    )
  }
  return (
    <div className="flex flex-col gap-4">
      <HealthBanner pay={pay} c={main} />
      <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(480px, 100%), 1fr))' }}>
        <ConnectionCard pay={pay} c={main} showTest={showTest} onShowTest={setShowTest} />
        <WebhookCard pay={pay} c={main} />
      </div>
      {main.webhook.status === 'silent' || main.webhook.status === 'missing' ? <GapRuns c={main} /> : null}
      <CheckoutCard pay={pay} c={main} />
      <EventLog test={showTest} />
    </div>
  )
}

function healthLabel(t: T, c: Conn): { label: string; tone: Tone } {
  switch (c.health) {
    case 'key_rejected':
      return { label: t(ms.keyRejected), tone: 'bad' }
    case 'webhook_missing':
      return { label: t(ms.webhookMissing), tone: 'bad' }
    case 'webhook_silent':
      return { label: t(ms.webhookSilent), tone: 'warn' }
    case 'missing_permissions':
      return { label: t(ms.permMissing), tone: 'warn' }
    case 'setup':
      return { label: t(ms.setupInProgress), tone: 'neutral' }
  }
  return { label: t(ms.connected), tone: 'ok' }
}

function HealthBanner({ pay, c }: { pay: Payments; c: Conn }) {
  const t = useT()
  const [replacing, setReplacing] = useState(false)
  const again = useRegisterAgain(c)
  switch (c.health) {
    case 'key_rejected':
      return (
        <>
          <Banner tone="bad" title={t(ms.keyRejectedTitle)} actions={pay.canManage ? <Button size="sm" onClick={() => setReplacing(true)}>{t(ms.replaceKey)}</Button> : null}>
            {t(ms.keyRejectedText, { date: c.keyRejectedAt ? monthDayTime(c.keyRejectedAt) : '' })}
          </Banner>
          {replacing ? <ReplaceKeyDialog c={c} onClose={() => setReplacing(false)} /> : null}
        </>
      )
    case 'webhook_missing':
      return (
        <Banner tone="bad" title={t(ms.webhookMissingTitle)} actions={pay.canManage ? <Button size="sm" onClick={again.run} loading={again.busy}>{t(ms.registerAgain)}</Button> : null}>
          {t(ms.webhookMissingText, { date: c.webhook.missingAt ? monthDayTime(c.webhook.missingAt) : '' })}
        </Banner>
      )
    case 'webhook_silent':
      return (
        <Banner
          tone="warn"
          title={t(ms.webhookSilentTitle)}
          actions={
            <ButtonAnchor href={c.webhooksUrl} size="sm" iconRight="external">
              {t(ms.webhookInStripe)}
            </ButtonAnchor>
          }
        >
          {t(ms.webhookSilentText, { hours: c.webhook.silentHours ?? 24, active: pay.activeSubscriptions })}
        </Banner>
      )
    case 'missing_permissions': {
      const miss = c.permissions.find((p) => p.ok === false)
      return (
        <Banner tone="warn" title={t(ms.permMissingTitle)}>
          {miss ? t(ms.permMissingText, { resource: resourceName(miss.resource), access: miss.access === 'write' ? 'Write' : 'Read' }) : null}
        </Banner>
      )
    }
  }
  return null
}

export function resourceName(r: string): string {
  return (
    {
      customers: 'Customers',
      checkout_sessions: 'Checkout Sessions',
      customer_portal: 'Customer portal',
      products: 'Products',
      prices: 'Prices',
      coupons: 'Coupons',
      promotion_codes: 'Promotion codes',
      subscriptions: 'Subscriptions',
      invoices: 'Invoices',
      charges: 'Charges and refunds',
      webhook_endpoints: 'Webhook endpoints',
      events: 'Events',
      tax_settings: 'Tax settings',
    }[r] ?? r
  )
}

function portalLine(t: T, p: Schemas['EarnPortal']): string {
  const parts = [p.updateCard && t(ms.portalUpdateCard), p.invoices && t(ms.portalInvoices), p.billingEmail && t(ms.portalBillingEmail), p.switchPeriod && t(ms.portalSwitch), p.cancel && t(ms.portalCancel)].filter(Boolean) as string[]
  return t(ms.portalLine, { features: parts.join(', ') })
}

function ConnectionCard({ pay, c, showTest, onShowTest }: { pay: Payments; c: Conn; showTest: boolean; onShowTest: (on: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [dialog, setDialog] = useState<'replace' | 'disconnect' | 'removeTest' | 'clearTest'>()
  const check = $api.useMutation('post', `${EARN}/connections/{id}/check`)
  const h = healthLabel(t, c)
  const test = pay.test && pay.live ? pay.test : undefined
  // Read-only: the actions stay where they are, disabled, saying why.
  const ro = pay.canManage ? undefined : t(ms.needsPaymentsManage)
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/status`] })
  }
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 items-center gap-3">
          <span className="ea-upg-ico" aria-hidden="true">
            S
          </span>
          <div className="flex min-w-0 flex-col">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="h2 m-0">{c.providerName}</h2>
              <Badge tone={c.mode === 'live' ? 'ok' : 'purple'}>{c.mode === 'live' ? t(me.live) : t(me.test)}</Badge>
              <Badge tone={h.tone} dot>
                {h.label}
              </Badge>
            </div>
            <span className="k">{t(ms.seller, { account: c.account.name })}</span>
          </div>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <dl className="ea-kv">
          <dt>{c.mode === 'live' ? t(ms.liveKey) : t(ms.testKeyLabel)}</dt>
          <dd className="flex flex-wrap items-center gap-2">
            <span className="ea-mono">{c.keyHint}</span>
            {c.keyRejectedAt ? <Badge tone="bad">{t(ms.rejectedAt, { date: monthDayTime(c.keyRejectedAt) })}</Badge> : <Badge tone="neutral">{t(ms.writeOnly)}</Badge>}
            <Button size="sm" disabledReason={ro} onClick={() => setDialog('replace')}>
              {t(ms.replaceKey)}
            </Button>
          </dd>
          <dt>{t(ms.permissions)}</dt>
          <dd className="flex flex-wrap items-center gap-2">
            {c.keyRejectedAt ? (
              <span className="k">{t(ms.permsCantCheck)}</span>
            ) : (
              <span className="inline-flex items-center gap-1.5">
                {c.granted === c.permissions.length ? <Icon name="check" size={14} strokeWidth={2.2} className="text-ok-fg" /> : null}
                {t(ms.permsLine, { granted: c.granted, total: c.permissions.length, when: c.checkedAt ? `${dayMonthDate(c.checkedAt)}, ${clock(c.checkedAt)}` : '—' })}
              </span>
            )}
            {c.keyRejectedAt ? null : (
              <Button
                size="sm"
                disabledReason={ro}
                loading={check.isPending}
                onClick={() =>
                  check.mutate(
                    { params: { path: { workspace: slug, id: c.id } } },
                    {
                      onSuccess: (r) => {
                        refresh()
                        toast({ kind: 'success', title: t(ms.checked, { granted: r.granted, total: r.permissions.length }) })
                      },
                      onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                    },
                  )
                }
              >
                {t(ms.checkAgain)}
              </Button>
            )}
          </dd>
          <dt>{t(ms.account)}</dt>
          <dd>{t(ms.accountLine, { name: c.account.name, country: countryName(c.account.country), currency: c.account.currency.toUpperCase() })}</dd>
          <dt>{t(ms.connectedOn)}</dt>
          <dd>{c.connectedAt ? t(ms.connectedLine, { date: dayMonthYear(c.connectedAt), name: c.connectedBy ?? '' }) : '—'}</dd>
          <dt>{t(ms.portal)}</dt>
          <dd>{c.portalSet ? portalLine(t, pay.checkout.portal) : t(ms.portalNotSet)}</dd>
        </dl>
      </div>
      {test ? (
        <div className="flex flex-col gap-2 border-t border-line-2 px-5 py-4">
          <div className="flex flex-wrap items-center justify-between gap-2 text-[13px]">
            <span className="flex flex-wrap items-center gap-2">
              <b>{t(ms.testKeyLabel)}</b>
              <Badge tone="purple">{t(me.test)}</Badge>
              <span className="ea-mono">{test.keyHint}</span>
            </span>
            <span className="k">{test.connectedAt ? t(ms.testKeyLine, { date: dayMonthDate(test.connectedAt) }) : ''}</span>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Toggle label={t(ms.showTest)} checked={showTest} onChange={(e) => onShowTest(e.target.checked)} />
            <Button size="sm" variant="danger" disabledReason={ro} onClick={() => setDialog('clearTest')}>
              {t(ms.clearTest)}
            </Button>
            <Button size="sm" disabledReason={ro} onClick={() => setDialog('removeTest')}>
              {t(ms.removeTestKey)}
            </Button>
          </div>
        </div>
      ) : null}
      <div className="border-t border-line-2 px-5 py-4">
        <OtherProviders pay={pay} />
      </div>
      <div className="flex flex-wrap justify-between gap-2 border-t border-line-2 px-5 py-3">
        {ro ? (
          <Button size="sm" icon="plus" disabledReason={ro}>
            {pay.live && !pay.test ? t(ms.addTestKey) : t(ms.addProvider)}
          </Button>
        ) : pay.live && !pay.test ? (
          <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }} search={{ mode: 'test' }} size="sm" icon="plus">
            {t(ms.addTestKey)}
          </ButtonLink>
        ) : (
          <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }} search={{ step: 'provider' }} size="sm" icon="plus">
            {t(ms.addProvider)}
          </ButtonLink>
        )}
        <Button size="sm" variant="danger" disabledReason={ro} onClick={() => setDialog('disconnect')}>
          {t(ms.disconnect, { provider: c.providerName })}
        </Button>
      </div>
      {dialog === 'replace' ? <ReplaceKeyDialog c={c} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'disconnect' ? <DisconnectDialog c={c} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'removeTest' && test ? <RemoveTestDialog c={test} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'clearTest' ? <ClearTestDialog n={pay.testMembers} onClose={() => setDialog(undefined)} /> : null}
    </section>
  )
}

function OtherProviders({ pay }: { pay: Payments }) {
  const t = useT()
  const others = pay.providers.filter((p) => !p.available)
  if (!others.length) return null
  return (
    <div className="flex flex-col gap-2">
      <h3 className="m-0 text-[14px] font-semibold">{t(ms.otherProviders)}</h3>
      {others.map((p) => (
        <div key={p.key} className="flex items-start gap-3">
          <span className="ea-upg-ico" aria-hidden="true">
            {p.name
              .split(' ')
              .map((w) => w[0])
              .join('')}
          </span>
          <div className="flex min-w-0 flex-col">
            <span className="flex flex-wrap items-center gap-2">
              <b>{p.name}</b>
              <Badge tone="neutral">{t(ms.notAvailable)}</Badge>
            </span>
            {p.merchantOfRecord ? <span className="k">{t(ms.mor)}</span> : null}
          </div>
        </div>
      ))}
    </div>
  )
}

function useRegisterAgain(c: Conn) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const m = $api.useMutation('post', `${EARN}/connections/{id}/webhook/again`)
  return {
    busy: m.isPending,
    run: () =>
      m.mutate(
        { params: { path: { workspace: slug, id: c.id } } },
        {
          onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
            void qc.invalidateQueries({ queryKey: ['get', `${EARN}/status`] })
            toast({ kind: 'success', title: t(ms.registeredAgain) })
          },
          onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
        },
      ),
  }
}

function webhookStatus(t: T, c: Conn): { label: string; tone: Tone } {
  switch (c.webhook.status) {
    case 'receiving':
      return { label: t(ms.whReceiving), tone: 'ok' }
    case 'waiting':
      return { label: t(ms.whWaiting), tone: 'info' }
    case 'silent':
      return { label: t(ms.whSilent, { hours: c.webhook.silentHours ?? 24 }), tone: 'warn' }
    case 'missing':
      return { label: t(ms.whMissing), tone: 'bad' }
  }
  return { label: t(ms.whUnregistered), tone: 'neutral' }
}

function WebhookCard({ pay, c }: { pay: Payments; c: Conn }) {
  const t = useT()
  const w = c.webhook
  const st = webhookStatus(t, c)
  const again = useRegisterAgain(c)
  const max = Math.max(1, ...w.byType.map((x) => x.count))
  const lastGap = w.gapRuns[0]
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(ms.webhook)}</h2>
        <Badge tone={st.tone} dot>
          {st.label}
        </Badge>
      </div>
      <div className="pad flex flex-col gap-4">
        <dl className="ea-kv">
          <dt>{t(ms.endpoint)}</dt>
          <dd className="ea-mono">{w.url.replace(/[^/]+$/, (tok) => (tok.length > 4 ? '…' : tok))}</dd>
          <dt>{t(ms.events)}</dt>
          <dd>{w.manual ? t(ms.eventsManual, { count: w.events.length }) : t(ms.eventsLine, { count: w.events.length, date: w.registeredAt ? dayMonthDate(w.registeredAt) : '—' })}</dd>
          <dt>{t(ms.lastEvent)}</dt>
          <dd>{w.lastEventAt ? t(ms.lastEventLine, { when: ago(w.lastEventAt), type: w.lastEventType ?? '' }) : t(ms.never)}</dd>
          <dt>{t(ms.events24h)}</dt>
          <dd className="ea-num">{count(w.events24h)}</dd>
          {w.missingAt ? (
            <>
              <dt>{t(ms.deletedInStripe)}</dt>
              <dd>{monthDayTime(w.missingAt)}</dd>
            </>
          ) : (
            <>
              <dt>{t(ms.rejected)}</dt>
              <dd className="ea-num">{count(w.rejected24h)}</dd>
            </>
          )}
          <dt>{t(ms.gapCheck)}</dt>
          <dd>{lastGap ? (lastGap.applied ? t(ms.gapLineApplied, { time: clock(lastGap.at), count: lastGap.applied }) : t(ms.gapLine, { time: clock(lastGap.at) })) : t(ms.gapNever)}</dd>
          <dt>{t(ms.secret)}</dt>
          <dd>{w.secretStored ? t(ms.secretSealed) : '—'}</dd>
        </dl>
        {w.byType.length ? (
          <div className="flex flex-col gap-2">
            <h3 className="m-0 text-[13px] font-semibold">{t(ms.byType)}</h3>
            {w.byType.map((x) => (
              <div key={x.type} className="flex flex-col gap-1">
                <div className="flex justify-between gap-2 text-[13px]">
                  <span className="ea-mono">{x.type}</span>
                  <b className="ea-num">{count(x.count)}</b>
                </div>
                <div className="ea-meter" aria-hidden="true">
                  <span style={{ width: `${(x.count / max) * 100}%` }} />
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-5 py-3">
        <span className="k">{t(ms.kept)}</span>
        {pay.canManage ? (
          <Button size="sm" icon="refresh" onClick={again.run} loading={again.busy}>
            {t(ms.registerAgain)}
          </Button>
        ) : null}
      </div>
    </section>
  )
}

function GapRuns({ c }: { c: Conn }) {
  const t = useT()
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(ms.gapTitle)}</h2>
          <span className="k">{t(ms.gapCaption)}</span>
        </div>
      </div>
      <ul className="ea-tl pad">
        {c.webhook.gapRuns.map((r) => (
          <li key={r.at}>
            <span className={`ic ${r.applied ? 'warn' : 'mut'}`} aria-hidden="true" />
            <div className="flex flex-col">
              <span className="t">{r.applied ? t(ms.gapRunApplied, { time: clock(r.at), count: r.applied }) : t(ms.gapRunNothing, { time: clock(r.at) })}</span>
              <span className="d">
                {Object.entries(r.types)
                  .map(([k, v]) => `${v} × ${k}`)
                  .join(', ')}
              </span>
            </div>
            <span className="w">{clock(r.at)}</span>
          </li>
        ))}
        {c.webhook.nextGapAt ? (
          <li>
            <span className="ic info" aria-hidden="true" />
            <span className="t">{t(ms.gapNext)}</span>
            <span className="w">{clock(c.webhook.nextGapAt)}</span>
          </li>
        ) : null}
      </ul>
    </section>
  )
}

function CheckoutCard({ pay, c }: { pay: Payments; c: Conn }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const ch = pay.checkout
  const [tax, setTaxState] = useState(ch.taxAtCheckout)
  const [incl, setInclState] = useState(ch.pricesIncludeTax)
  const [portal, setPortalState] = useState(ch.portal)
  const save = $api.useMutation('put', `${EARN}/checkout`)
  const ro = !pay.canManage
  // Each change saves at once, with Undo.
  const commit = (next: { taxAtCheckout: boolean; pricesIncludeTax: boolean; portal: typeof portal }, undo?: () => void) => {
    setTaxState(next.taxAtCheckout)
    setInclState(next.pricesIncludeTax)
    setPortalState(next.portal)
    save.mutate(
      { params: { path: { workspace: slug } }, body: next },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
          toast({ kind: 'success', title: t(ms.saved), undo })
        },
        onError: (e) => {
          setTaxState(ch.taxAtCheckout)
          setInclState(ch.pricesIncludeTax)
          setPortalState(ch.portal)
          toast({ kind: 'error', title: errorMessage(e) })
        },
      },
    )
  }
  const before = { taxAtCheckout: tax, pricesIncludeTax: incl, portal }
  const setTax = (v: boolean) => commit({ ...before, taxAtCheckout: v }, () => commit(before))
  const setIncl = (v: boolean) => commit({ ...before, pricesIncludeTax: v }, () => commit(before))
  const setPortal = (v: typeof portal) => commit({ ...before, portal: v }, () => commit(before))
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(ms.checkoutTitle)}</h2>
          <span className="k">{t(ms.checkoutCaption)}</span>
        </div>
      </div>
      <div className="pad grid gap-5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))' }}>
        <div className="flex flex-col gap-2">
          <h3 className="m-0 text-[14px] font-semibold">{t(ms.currency)}</h3>
          <span className="k">{ch.currencyFixed ? t(ms.currencyFixed, { account: c.account.name, currency: ch.currency.toUpperCase() }) : t(ms.currencyFree, { account: c.account.name, currency: ch.currency.toUpperCase() })}</span>
          <span className="ea-pill w-fit">{ch.currency.toUpperCase()}</span>
        </div>
        <div className="flex flex-col gap-2">
          <h3 className="m-0 text-[14px] font-semibold">{t(ms.tax)}</h3>
          <span className="k">{ch.taxAvailable ? t(ms.taxOn, { account: c.account.name }) : t(ms.taxOff, { account: c.account.name })}</span>
          <Toggle label={t(ms.taxToggle)} checked={tax} disabled={ro || !ch.taxAvailable} onChange={(e) => setTax(e.target.checked)} />
        </div>
        <div className="flex flex-col gap-2">
          <h3 className="m-0 text-[14px] font-semibold">{t(ms.prices)}</h3>
          <span className="k">{t(ms.pricesHint)}</span>
          <Select aria-label={t(ms.prices)} value={incl ? 'include' : 'exclude'} disabled={ro} onChange={(e) => setIncl(e.target.value === 'include')}>
            <option value="exclude">{t(ms.pricesExclude)}</option>
            <option value="include">{t(ms.pricesInclude)}</option>
          </Select>
        </div>
      </div>
      <div className="flex flex-col gap-2 border-t border-line-2 px-5 py-4">
        <h3 className="m-0 text-[14px] font-semibold">{t(ms.portalTitle)}</h3>
        <span className="k">{t(ms.portalCaption)}</span>
        <div className="flex flex-wrap gap-x-6 gap-y-2">
          {(
            [
              ['updateCard', t(ms.portalUpdateCard)],
              ['invoices', t(ms.portalInvoices)],
              ['billingEmail', t(ms.portalBillingEmail)],
              ['switchPeriod', t(ms.portalSwitch)],
              ['cancel', t(ms.portalCancel)],
            ] as const
          ).map(([k, label]) => (
            <label key={k} className="flex items-center gap-2 text-[13px]">
              <input type="checkbox" disabled={ro} checked={portal[k]} onChange={(e) => setPortal({ ...portal, [k]: e.target.checked })} />
              {label.charAt(0).toUpperCase() + label.slice(1)}
            </label>
          ))}
        </div>
      </div>
      <div className="px-5 pb-4">
        <div className="note n" role="note">
          <Icon name="info" size={16} />
          <div>{t(ms.sellerNote)}</div>
        </div>
      </div>
    </section>
  )
}

type Group = 'all' | 'payments' | 'failures' | 'refunds' | 'changes' | 'trials'
type Range = '24h' | '7d' | '30d' | '90d'

function EventLog({ test }: { test: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [q, setQ] = useState('')
  const [group, setGroup] = useState<Group>('all')
  const [range, setRange] = useState<Range>('7d')
  const [page, setPage] = useState(1)
  const log = $api.useQuery('get', `${EARN}/payments/events`, { params: { path: { workspace: slug }, query: { q, group, range, page, pageSize: 8, test } } }, { placeholderData: (p) => p })
  const d = log.data
  return (
    <section className="card" aria-labelledby="ea-log">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ea-log">
            {t(ms.logTitle)}
          </h2>
          <span className="k">{t(ms.logCaption)}</span>
        </div>
      </div>
      <div className="ea-logf grid gap-2.5 px-5 pb-3">
        <label className="ea-search">
          <Icon name="search" size={15} />
          <input type="search" aria-label={t(ms.logSearchLabel)} placeholder={t(ms.logSearch)} value={q} onChange={(e) => (setQ(e.target.value), setPage(1))} />
        </label>
        <Select aria-label={t(ms.logGroup)} value={group} onChange={(e) => (setGroup(e.target.value as Group), setPage(1))}>
          <option value="all">{t(ms.groupAll)}</option>
          <option value="payments">{t(ms.groupPayments)}</option>
          <option value="failures">{t(ms.groupFailures)}</option>
          <option value="refunds">{t(ms.groupRefunds)}</option>
          <option value="changes">{t(ms.groupChanges)}</option>
          <option value="trials">{t(ms.groupTrials)}</option>
        </Select>
        <Select aria-label={t(ms.logRange)} value={range} onChange={(e) => (setRange(e.target.value as Range), setPage(1))}>
          <option value="24h">{t(ms.range24h)}</option>
          <option value="7d">{t(ms.range7d)}</option>
          <option value="30d">{t(ms.range30d)}</option>
          <option value="90d">{t(ms.range90d)}</option>
        </Select>
      </div>
      <TableScroll label={t(ms.logLabel)}>
        <table className="tbl" aria-label={t(ms.logLabel)}>
          <thead>
            <tr>
              <th scope="col">{t(ms.colTime)}</th>
              <th scope="col">{t(ms.colEvent)}</th>
              <th scope="col">{t(ms.colMember)}</th>
              <th scope="col" className="r">
                {t(ms.colAmount)}
              </th>
              <th scope="col">{t(ms.colDid)}</th>
            </tr>
          </thead>
          <tbody>
            {d?.items.map((e) => {
              const st = eventStatus(t, e)
              return (
                <tr key={e.id}>
                  <td className="ea-mono whitespace-nowrap">{logWhen(e.receivedAt)}</td>
                  <td className="ea-mono">
                    {e.eventUrl ? (
                      <a className="ea-evlink" href={e.eventUrl} target="_blank" rel="noopener noreferrer">
                        {e.type}
                      </a>
                    ) : (
                      e.type
                    )}
                    {e.mode === 'test' ? (
                      <Badge tone="purple" size="sm" className="ml-2">
                        {t(me.test)}
                      </Badge>
                    ) : null}
                  </td>
                  <td className="font-semibold">{e.memberName || '—'}</td>
                  <td className="r ea-num whitespace-nowrap">{e.amountCents === undefined ? '–' : money(e.amountCents, e.currency || 'usd', 'always')}</td>
                  <td>
                    <span className="inline-flex flex-wrap items-center gap-2">
                      <Badge tone={st.tone}>{st.label}</Badge>
                      {outcomeText(t, e)}
                    </span>
                  </td>
                </tr>
              )
            })}
            {d && d.items.length === 0 ? (
              <tr>
                <td colSpan={5} className="k">
                  {t(ms.logEmpty)}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </TableScroll>
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line-2 px-5 py-3">
        <span className="k">{d ? t(ms.logShowing, { shown: d.items.length, total: d.total, range }) : '…'}</span>
        <div className="flex gap-2">
          <Button size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            {t(ms.previous)}
          </Button>
          <Button size="sm" disabled={!d || page * 8 >= d.total} onClick={() => setPage(page + 1)}>
            {t(ms.next)}
          </Button>
        </div>
      </div>
    </section>
  )
}

function eventStatus(t: T, e: Schemas['EarnEvent']): { label: string; tone: Tone } {
  switch (e.status) {
    case 'applied': {
      const tone: Tone =
        e.outcome === 'past_due'
          ? 'warn'
          : ['access_ended', 'refunded', 'refund_recorded', 'gift_refunded', 'cancel_scheduled', 'ended', 'canceled'].includes(e.outcome)
            ? 'neutral'
            : 'ok'
      return { label: t(ms.stApplied), tone }
    }
    case 'ignored':
      return { label: t(ms.stIgnored), tone: 'neutral' }
    case 'failed':
      return { label: t(ms.stFailed), tone: 'bad' }
    case 'rejected':
      return { label: t(ms.stRejected), tone: 'bad' }
  }
  return { label: t(ms.stPending), tone: 'info' }
}

/** "What ReputeCast did", from the outcome and its facts. */
export function outcomeText(t: T, e: Schemas['EarnEvent']): string {
  const d = e.detail as Record<string, unknown>
  const s = (k: string) => (typeof d[k] === 'string' ? (d[k] as string) : '')
  const num = (k: string) => (typeof d[k] === 'number' ? (d[k] as number) : 0)
  const email = s('emailStatus') && s('emailStatus') !== 'sent' ? t(ms.oEmailNot, { reason: s('emailReason') || s('emailStatus') }) : ''
  if (e.status === 'rejected') return t(ms.oRejected)
  if (e.status === 'failed') return t(ms.oFailed)
  if (e.status === 'ignored') return e.outcome === 'nothing_changed' ? t(ms.oNothing) : t(ms.oIgnored, { why: e.outcome })
  switch (e.outcome) {
    case 'renewed':
      if (s('offer')) return t(ms.oRenewedOffer, { offer: s('offer') })
      if (s('until')) return t(ms.oRenewed, { tier: shortTier(s('tier')), period: s('period'), date: day(s('until')) })
      return t(ms.oRenewedPlain)
    case 'paid':
      return t(ms.oPaid, { tier: shortTier(s('tier')) })
    case 'subscribed':
      return t(ms.oSubscribed, { tier: shortTier(s('tier')) }) + email
    case 'trial_started':
      return t(ms.oTrialStarted, { tier: shortTier(s('tier')) }) + email
    case 'sent_trial_ending':
      return t(ms.oTrialEnding, { date: day(s('date')) }) + email
    case 'sent_renewal_reminder':
      return t(ms.oRenewal, { date: day(s('date')) }) + email
    case 'access_ended':
      return s('by') ? t(ms.oEndedBy, { by: s('by') }) : t(ms.oEnded, { reason: s('reason') || 'canceled' })
    case 'refunded':
      return num('ltv') ? t(ms.oRefund, { ltv: money(num('ltv'), e.currency || 'usd', 'auto') }) : t(ms.oRefundPlain)
    case 'past_due':
      return (s('retry') ? t(ms.oPastDue, { date: day(s('retry')) }) : t(ms.oPastDueNoRetry)) + email
    case 'cancel_scheduled':
      return s('reason') ? t(ms.oCancelReason, { date: day(s('date')), reason: cancelReason(t, s('reason')) }) : t(ms.oCancelScheduled, { date: day(s('date')) })
    case 'cancel_undone':
      return t(ms.oCancelUndone)
    case 'plan_changed':
      return t(ms.oPlanChanged, { from: s('from') })
    case 'offer_applied':
      return t(ms.oOffer)
    case 'checkout_completed':
      return t(ms.oCheckout)
    case 'gift_paid':
      return t(ms.oGiftPaid, { months: num('months') })
    case 'gift_refunded':
      return d.partial ? t(ms.oGiftPartlyRefunded) : t(ms.oGiftRefunded)
    case 'dispute_opened':
      return t(ms.oDispute)
    case 'billing_email':
      return t(ms.oBilling)
  }
  return e.outcome
}

function shortTier(name: string): string {
  return name
}

function ReplaceKeyDialog({ c, onClose }: { c: Conn; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [key, setKey] = useState('')
  const [err, setErr] = useState<string>()
  const replace = $api.useMutation('put', `${EARN}/connections/{id}/key`)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{`${c.providerName} · ${c.account.name} · ${c.mode === 'live' ? t(me.live) : t(me.test)}`}</span>}
      title={t(ms.replaceTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(ms.cancel)}</Button>
          <Button
            variant="primary"
            disabled={!key.trim()}
            loading={replace.isPending}
            onClick={() =>
              replace.mutate(
                { params: { path: { workspace: slug, id: c.id } }, body: { key } },
                {
                  onSuccess: () => {
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/status`] })
                    toast({ kind: 'success', title: t(ms.replaced) })
                    onClose()
                  },
                  onError: (e) => setErr(errorMessage(e)),
                },
              )
            }
          >
            {t(ms.replaceSubmit)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <Field label={t(ms.newKey)} hint={t(ms.newKeyHint)} error={err}>
          <Input value={key} autoComplete="off" spellCheck={false} className="ea-mono" placeholder={c.mode === 'live' ? 'rk_live_…' : 'rk_test_…'} onChange={(e) => setKey(e.target.value)} />
        </Field>
        <span className="k">{t(ms.replaceText, { count: c.permissions.length })}</span>
      </div>
    </Dialog>
  )
}

function DisconnectDialog({ c, onClose }: { c: Conn; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const disc = $api.useMutation('post', `${EARN}/connections/{id}/disconnect`)
  const account = c.account.name || c.keyHint
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="bad"
      badge={t(ms.disconnectBadge)}
      title={t(ms.disconnectTitle, { provider: c.providerName })}
      description={
        <>
          <p className="m-0">{t(ms.disconnectWhat)}</p>
          <p className="m-0 mt-2">{t(ms.disconnectKeep)}</p>
        </>
      }
      typedConfirmation={{ label: t(ms.disconnectType, { account }), expected: account }}
      confirmLabel={t(ms.disconnect, { provider: c.providerName })}
      onConfirm={async () => {
        await disc.mutateAsync({ params: { path: { workspace: slug, id: c.id } }, body: { confirm: account } })
        void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
        void qc.invalidateQueries({ queryKey: ['get', `${EARN}/status`] })
        toast({ kind: 'deleted', title: t(ms.disconnected, { provider: c.providerName }) })
        onClose()
      }}
    />
  )
}

function RemoveTestDialog({ c, onClose }: { c: Conn; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const disc = $api.useMutation('post', `${EARN}/connections/{id}/disconnect`)
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge={t(ms.removeTestKey)}
      title={t(ms.removeTestTitle)}
      description={t(ms.removeTestText)}
      confirmLabel={t(ms.removeTestKey)}
      onConfirm={async () => {
        await disc.mutateAsync({ params: { path: { workspace: slug, id: c.id } }, body: { confirm: '' } })
        void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
        toast({ kind: 'deleted', title: t(ms.removedTest) })
        onClose()
      }}
    />
  )
}

function ClearTestDialog({ n, onClose }: { n: number; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const clear = $api.useMutation('post', `${EARN}/test-data/clear`)
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="bad"
      badge={t(ms.clearTest)}
      title={t(ms.clearTest)}
      description={`${count(n)} · ${t(ms.clearTest)}`}
      confirmLabel={t(ms.clearTest)}
      onConfirm={async () => {
        await clear.mutateAsync({ params: { path: { workspace: slug } } })
        void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
        void qc.invalidateQueries({ queryKey: ['get', `${EARN}/members`] })
        onClose()
      }}
    />
  )
}

/** "09:56" today, "13 Oct 16:41" before. */
function logWhen(d: string): string {
  // logTime gives the time alone on the workspace's today.
  return logTime(d) === clock(d) ? clock(d) : `${dayMonthDate(d)} ${clock(d)}`
}
