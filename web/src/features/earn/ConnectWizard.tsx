import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon, LogoMark } from '@/components/icons/Icon'
import { SystemBanners } from '@/components/shell/SystemBanners'
import { FullPageState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Toggle } from '@/components/ui/Field'
import { Banner } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T as Msg, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clock, clockSeconds, orList } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { roleName } from '@/lib/roles'
import { useCan, useWorkspace } from '@/lib/session'
import '../connect/connect.css'
import { mc } from './connect.messages'
import { EARN } from './EarnPage'
import { countryName } from './format'
import { resourceName } from './PaymentsTab'
import './earn.css'

type Conn = Schemas['EarnConnection']
export type ConnectSearch = { id?: string; step?: 'provider' | 'key' | 'check' | 'webhook' | 'settings' | 'done'; mode?: 'live' | 'test' }
type T = ReturnType<typeof useT>

const order = ['provider', 'key', 'check', 'webhook', 'settings', 'done'] as const

/** Connect a payment provider (EarnConnect): six steps on their own screen, resumed from the URL. */
export function ConnectWizard({ search, onSearch }: { search: ConnectSearch; onSearch: (next: ConnectSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('payments:manage')
  const pay = $api.useQuery('get', `${EARN}/payments`, { params: { path: { workspace: slug } } })
  const conn = $api.useQuery('get', `${EARN}/connections/{id}`, { params: { path: { workspace: slug, id: search.id ?? '' } } }, { enabled: Boolean(search.id), refetchInterval: (q) => (q.state.data?.step === 'webhook' && q.state.data.webhook.status === 'waiting' ? 3000 : false) })
  // A setup in progress picks up where it stopped.
  useEffect(() => {
    if (search.id || !pay.data) return
    const s = [pay.data.live, pay.data.test].find((c) => c?.status === 'setup')
    if (s) onSearch({ id: s.id, step: s.step }, true)
  }, [search.id, pay.data, onSearch])
  const step = search.step ?? (search.id ? conn.data?.step : undefined) ?? 'provider'
  const n = order.indexOf(step) + 1
  return (
    <Layout badge={<Badge tone="neutral">{t(mc.stepBadge, { n })}</Badge>}>
      <div className="cn-wiz">
        <Steps current={n} conn={conn.data} pay={pay.data} />
        <main className="cn-main" id="main" tabIndex={-1}>
          {!can ? (
            <NoPermission pay={pay.data} />
          ) : pay.error || conn.error ? (
            <ErrorPanel title={t(mc.errorTitle)} actions={<Button onClick={() => void (conn.error ? conn.refetch() : pay.refetch())}>{t(mc.checkAgain)}</Button>}>
              {errorMessage(conn.error ?? pay.error)}
            </ErrorPanel>
          ) : !pay.data || (search.id && !conn.data) ? (
            <Skeleton w="100%" h={360} />
          ) : step === 'provider' ? (
            <ProviderStep pay={pay.data} onNext={() => onSearch({ ...search, step: 'key' })} />
          ) : step === 'key' ? (
            <KeyStep
              pay={pay.data}
              mode={search.mode}
              onMode={(m) => onSearch({ ...search, mode: m }, true)}
              onBack={() => onSearch({ ...search, step: 'provider' })}
              onChecked={(c) => onSearch({ id: c.id, step: 'check', mode: search.mode })}
            />
          ) : conn.data && step === 'check' ? (
            <CheckStep c={conn.data} onBack={() => onSearch({ mode: search.mode, step: 'key' })} onNext={() => onSearch({ ...search, step: 'webhook' })} />
          ) : conn.data && step === 'webhook' ? (
            <WebhookStep c={conn.data} onBack={() => onSearch({ ...search, step: 'check' })} onNext={() => onSearch({ ...search, step: 'settings' })} />
          ) : conn.data && step === 'settings' ? (
            <SettingsStep c={conn.data} pay={pay.data} onBack={() => onSearch({ ...search, step: 'webhook' })} onDone={() => onSearch({ ...search, step: 'done' })} />
          ) : conn.data ? (
            <DoneStep c={conn.data} pay={pay.data} />
          ) : null}
        </main>
      </div>
    </Layout>
  )
}

function Layout({ badge, children }: { badge: ReactNode; children: ReactNode }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  useDocumentTitle(`${t(mc.title)} · ${me.workspace.name} · ReputeCast`)
  return (
    <div className="rc-app flex min-h-screen flex-col">
      <a href="#main" className="skip">
        {t(mc.skip)}
      </a>
      <header className="cn-head">
        <Link to="/w/$workspace" params={{ workspace: slug }} className="cn-home" aria-label={t(mc.home)}>
          <LogoMark size={26} />
          <span className="text-[15px] font-bold">ReputeCast</span>
        </Link>
        <span className="k" aria-hidden="true">
          /
        </span>
        <h1 className="m-0 text-[16px] font-semibold">{t(mc.title)}</h1>
        {badge}
        <ButtonLink to="/w/$workspace/earn" params={{ workspace: slug }} className="ml-auto">
          {t(mc.exit)}
        </ButtonLink>
      </header>
      <SystemBanners />
      {children}
    </div>
  )
}

function Steps({ current, conn, pay }: { current: number; conn?: Conn; pay?: Schemas['EarnPayments'] }) {
  const t = useT()
  const perms = pay?.permissions.length ?? 13
  const events = pay?.webhookEvents.length ?? 12
  const defs = [
    { title: t(mc.sProvider), sub: current > 1 ? 'Stripe' : '' },
    { title: t(mc.sKey), sub: current > 2 && conn ? conn.keyHint : t(mc.sKeySub) },
    { title: t(mc.sCheck), sub: current > 3 && conn ? t(mc.permsOk, { granted: conn.granted, total: perms }) : t(mc.sCheckSub, { count: perms }) },
    { title: t(mc.sWebhook), sub: t(mc.sWebhookSub, { count: events }) },
    { title: t(mc.sSettings), sub: t(mc.sSettingsSub) },
    { title: t(mc.sDone), sub: '' },
  ]
  return (
    <nav className="card cn-steps" aria-label={t(mc.stepsLabel)}>
      {defs.map((d, i) => {
        const n = i + 1
        const done = n < current
        const cur = n === current
        return (
          <div key={d.title} className={cn('step cn-step', cur && 'cur', done && 'done')} aria-current={cur ? 'step' : undefined}>
            <span className="num" aria-hidden="true">
              {done ? '✓' : n}
            </span>
            <div>
              <div className="font-semibold">
                {done ? <span className="vh">{t(mc.done)}</span> : null}
                {d.title}
              </div>
              {d.sub ? <div className="k">{d.sub}</div> : null}
            </div>
          </div>
        )
      })}
    </nav>
  )
}

function StepCard({ n, title, lead, children, footer, done }: { n: number; title: ReactNode; lead?: ReactNode; children?: ReactNode; footer?: ReactNode; done?: boolean }) {
  const t = useT()
  const head = (
    <>
      <h2 className="m-0 text-[24px] font-bold tracking-[-0.02em]">{title}</h2>
      {lead ? <span className="max-w-[760px] text-ink-2">{lead}</span> : null}
    </>
  )
  return (
    <section className="card flex flex-col gap-4 p-6">
      <div className="flex flex-col gap-1.5">
        <Badge tone="neutral" className="w-fit">
          {t(mc.stepBadge, { n })}
        </Badge>
        {done ? (
          <div className="flex items-start gap-3">
            <span className="ea-done-ic" aria-hidden="true">
              <Icon name="check" size={18} strokeWidth={2.6} />
            </span>
            <div className="flex min-w-0 flex-col gap-1">{head}</div>
          </div>
        ) : (
          head
        )}
      </div>
      {children}
      {footer ? <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line-2 pt-4">{footer}</div> : null}
    </section>
  )
}

function NoPermission({ pay }: { pay?: Schemas['EarnPayments'] }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const role = roleName({ key: me.workspace.roleKey, name: me.workspace.roleName })
  const helpers = pay?.helpers ?? []
  return (
    <FullPageState
      icon="lockLarge"
      tone="warn"
      badge={t(mc.forbiddenBadge, { role })}
      title={t(mc.forbiddenTitle)}
      actions={
        <ButtonLink to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'payments' }}>
          {t(mc.earnSettings)}
        </ButtonLink>
      }
    >
      {helpers.length ? t(mc.forbiddenText, { helpers: orList(helpers) }) : t(mc.forbiddenTextNoHelpers)}
    </FullPageState>
  )
}

/** A capability as the cards name it; coupons with promotion codes are one ("Coupons and promotion codes"). */
function capLabel(t: T, c: string, all: string[]): string {
  return (
    {
      subscriptions: t(mc.capSubscriptions),
      one_time: t(mc.capOneTime),
      trials: t(mc.capTrials),
      coupons: all.includes('promotion_codes') ? t(mc.capCoupons) : t(mc.capCouponsOnly),
      promotion_codes: '',
      portal: t(mc.capPortal),
      tax: t(mc.capTax),
      merchant_of_record: t(mc.capMor),
      test_mode: t(mc.capTestMode),
    }[c] ?? ''
  )
}

function ProviderStep({ pay, onNext }: { pay: Schemas['EarnPayments']; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <StepCard
      n={1}
      title={t(mc.chooseTitle)}
      lead={t(mc.chooseLead)}
      footer={
        <>
          <ButtonLink to="/w/$workspace/earn" params={{ workspace: slug }}>
            {t(mc.cancel)}
          </ButtonLink>
          <Button variant="primary" onClick={onNext}>
            {t(mc.continueWith, { provider: 'Stripe' })}
          </Button>
        </>
      }
    >
      <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))' }}>
        {pay.providers.map((p) => (
          <div key={p.key} className={cn('ea-provider', p.available ? 'on' : 'off')} aria-disabled={!p.available || undefined}>
            <div className="flex items-center gap-3">
              <span className="ea-upg-ico" aria-hidden="true">
                {p.name
                  .split(' ')
                  .map((w) => w[0])
                  .join('')}
              </span>
              <div className="flex flex-col gap-1">
                <b>{p.name}</b>
                <Badge tone={p.available ? 'ok' : 'neutral'} className="w-fit">
                  {p.available ? t(mc.available) : t(mc.notAvailable)}
                </Badge>
              </div>
            </div>
            <span className="text-[13px] text-ink-2">{p.merchantOfRecord ? t(mc.morLine) : t(mc.stripeLine)}</span>
            <div className="flex flex-wrap gap-1.5">
              {p.capabilities
                .map((c) => capLabel(t, c, p.capabilities))
                .filter(Boolean)
                .map((c) => (
                  <span key={c} className="ea-cap">
                    {c}
                  </span>
                ))}
            </div>
          </div>
        ))}
      </div>
    </StepCard>
  )
}

function why(t: T, r: string): string {
  return (
    {
      customers: t(mc.whyCustomers),
      checkout_sessions: t(mc.whyCheckout),
      customer_portal: t(mc.whyPortal),
      products: t(mc.whyProducts),
      prices: t(mc.whyPrices),
      coupons: t(mc.whyCoupons),
      promotion_codes: t(mc.whyPromo),
      subscriptions: t(mc.whySubs),
      invoices: t(mc.whyInvoices),
      charges: t(mc.whyCharges),
      webhook_endpoints: t(mc.whyWebhooks),
      events: t(mc.whyEvents),
      tax_settings: t(mc.whyTax),
    }[r] ?? ''
  )
}

function KeyStep({
  pay,
  mode,
  onMode,
  onBack,
  onChecked,
}: {
  pay: Schemas['EarnPayments']
  mode?: 'live' | 'test'
  onMode: (m: 'live' | 'test') => void
  onBack: () => void
  onChecked: (c: Conn) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const [key, setKey] = useState('')
  const [err, setErr] = useState<{ title?: string; text: string }>()
  const connect = $api.useMutation('post', `${EARN}/connections`)
  const list = pay.permissions.map((p) => `${resourceName(p.resource)}: ${p.access === 'write' ? 'Write' : 'Read'}`).join('\n')
  const stripeKeys = pay.live?.createKeyUrl ?? pay.test?.createKeyUrl ?? 'https://dashboard.stripe.com/apikeys/create'
  const submit = () =>
    connect.mutate(
      { params: { path: { workspace: slug } }, body: { provider: 'stripe', key } },
      {
        onSuccess: onChecked,
        onError: (e) => {
          if (e instanceof ApiError && e.code === 'secret_key') setErr({ title: t(mc.secretKeyTitle), text: e.message })
          else setErr({ text: errorMessage(e) })
        },
      },
    )
  return (
    <StepCard
      n={2}
      title={mode === 'live' ? t(mc.liveTitle) : t(mc.keyTitle)}
      lead={mode === 'live' ? t(mc.liveText) : t(mc.keyLead)}
      footer={
        <>
          <Button onClick={onBack}>{t(mc.back)}</Button>
          <Button variant="primary" disabled={!key.trim()} loading={connect.isPending} busyLabel={t(mc.checking)} onClick={submit}>
            {t(mc.checkKey)}
          </Button>
        </>
      }
    >
      {!mode && !pay.test && !pay.live ? (
        <div className="note i" role="note">
          <Icon name="info" size={16} />
          <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-0.5">
            <span className="font-semibold">{t(mc.recommendTitle)}</span>
            <span>{t(mc.recommendText)}</span>
          </div>
          <Button size="sm" onClick={() => onMode('test')}>
            {t(mc.useTestKey)}
          </Button>
        </div>
      ) : null}
      <section className="card">
        <div className="sec">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h3 className="h2 m-0">{t(mc.permsTitle, { count: pay.permissions.length })}</h3>
            <span className="k">{t(mc.permsCaption)}</span>
          </div>
          <div className="flex gap-2">
            <Button size="sm" icon="copy" onClick={() => void navigator.clipboard.writeText(list).then(() => toast({ kind: 'success', title: t(mc.listCopied) }))}>
              {t(mc.copyList)}
            </Button>
            <ButtonAnchor href={stripeKeys} size="sm" iconRight="external">
              {t(mc.openStripe)}
            </ButtonAnchor>
          </div>
        </div>
        <TableScroll label={t(mc.permsTitle, { count: pay.permissions.length })}>
          <table className="tbl">
            <thead>
              <tr>
                <th scope="col">{t(mc.colResource)}</th>
                <th scope="col">{t(mc.colAccess)}</th>
                <th scope="col">{t(mc.colWhy)}</th>
              </tr>
            </thead>
            <tbody>
              {pay.permissions.map((p) => (
                <tr key={p.resource}>
                  <td className="font-semibold">{resourceName(p.resource)}</td>
                  <td>
                    <Badge tone={p.access === 'write' ? 'info' : 'neutral'}>{p.access === 'write' ? t(mc.write) : t(mc.read)}</Badge>
                  </td>
                  <td>{why(t, p.resource)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      </section>
      <Field
        label={t(mc.keyLabel)}
        hint={<Msg msg={mc.keyHint} values={{ live: <span className="mono">{'rk_live_'}</span>, test: <span className="mono">{'rk_test_'}</span> }} />}
        error={err ? (err.title ? `${err.title}. ${err.text}` : err.text) : undefined}
      >
        <Input type="password" autoComplete="off" spellCheck={false} value={key} onChange={(e) => (setKey(e.target.value), setErr(undefined))} placeholder={mode === 'test' ? 'rk_test_…' : t(mc.keyPlaceholder)} />
      </Field>
    </StepCard>
  )
}

function CheckStep({ c, onBack, onNext }: { c: Conn; onBack: () => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const check = $api.useMutation('post', `${EARN}/connections/{id}/check`)
  const step = $api.useMutation('put', `${EARN}/connections/{id}/step`)
  const miss = c.permissions.filter((p) => p.ok === false)
  return (
    <StepCard
      n={3}
      title={t(mc.checkTitle)}
      lead={t(mc.checkLead)}
      footer={
        <>
          <Button onClick={onBack}>{t(mc.back)}</Button>
          <Button
            variant="primary"
            disabled={miss.length > 0}
            loading={step.isPending}
            onClick={() => step.mutate({ params: { path: { workspace: slug, id: c.id } }, body: { step: 'webhook' } }, { onSuccess: onNext, onError: (e) => toast({ kind: 'error', title: errorMessage(e) }) })}
          >
            {t(mc.toWebhook)}
          </Button>
        </>
      }
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <dl className="ea-kv">
          <dt>{t(mc.account)}</dt>
          <dd>{t(mc.accountLine, { name: c.account.name, country: countryName(c.account.country), currency: c.account.currency.toUpperCase() })}</dd>
          <dt>{t(mc.mode)}</dt>
          <dd className="flex items-center gap-2">
            <Badge tone={c.mode === 'live' ? 'ok' : 'purple'}>{c.mode === 'live' ? t(mc.modeLive) : t(mc.modeTest)}</Badge>
            <span className="k">{t(mc.modeFrom, { prefix: c.mode === 'live' ? 'rk_live_' : 'rk_test_' })}</span>
          </dd>
          <dt>{t(mc.stripeTax)}</dt>
          <dd>{c.account.taxEnabled ? t(mc.taxOn, { account: c.account.name }) : t(mc.taxOff, { account: c.account.name })}</dd>
          <dt>{t(mc.key)}</dt>
          <dd className="ea-mono">{t(mc.keySealed, { hint: c.keyHint })}</dd>
        </dl>
        <Badge tone={miss.length ? 'warn' : 'ok'} dot>
          {t(mc.permsOk, { granted: c.granted, total: c.permissions.length })}
        </Badge>
      </div>
      {miss[0] ? (
        <Banner
          tone="warn"
          title={t(mc.missingLine, { resource: resourceName(miss[0].resource), access: miss[0].access === 'write' ? t(mc.write) : t(mc.read) })}
          actions={
            <>
              <ButtonAnchor href={c.keysUrl} size="sm" iconRight="external">
                {t(mc.editKey)}
              </ButtonAnchor>
              <Button
                size="sm"
                loading={check.isPending}
                onClick={() => check.mutate({ params: { path: { workspace: slug, id: c.id } } }, { onSuccess: () => void qc.invalidateQueries({ queryKey: ['get', `${EARN}/connections/{id}`] }), onError: (e) => toast({ kind: 'error', title: errorMessage(e) }) })}
              >
                {t(mc.checkAgain)}
              </Button>
            </>
          }
        />
      ) : null}
      <ul className="m-0 grid list-none gap-0 p-0" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))' }}>
        {c.permissions.map((p) => (
          <li key={p.resource} className="flex items-center gap-2 border-t border-line-2 py-2.5 text-[13px]">
            <Icon name={p.ok === false ? 'close' : 'check'} size={14} />
            <b>{resourceName(p.resource)}</b> · {p.access === 'write' ? t(mc.write) : t(mc.read)}
            {p.ok === false ? <Badge tone="warn">{t(mc.missing)}</Badge> : null}
          </li>
        ))}
      </ul>
    </StepCard>
  )
}

function WebhookStep({ c, onBack, onNext }: { c: Conn; onBack: () => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [unreachable, setUnreachable] = useState<string>()
  const [secret, setSecret] = useState('')
  const reg = $api.useMutation('post', `${EARN}/connections/{id}/webhook`)
  const save = $api.useMutation('put', `${EARN}/connections/{id}/webhook-secret`)
  const step = $api.useMutation('put', `${EARN}/connections/{id}/step`)
  const w = c.webhook
  const refresh = () => void qc.invalidateQueries({ queryKey: ['get', `${EARN}/connections/{id}`] })
  const register = () =>
    reg.mutate(
      { params: { path: { workspace: slug, id: c.id } } },
      {
        onSuccess: () => {
          setUnreachable(undefined)
          refresh()
        },
        onError: (e) => (e instanceof ApiError && e.code === 'public_url_unreachable' ? setUnreachable(e.message) : toast({ kind: 'error', title: errorMessage(e) })),
      },
    )
  useEffect(() => {
    if (!w.registered && !reg.isPending && !unreachable && !reg.isError) register()
    // Registers once on arrival; the person retries after a failure.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
  const at = (d?: string) => (d ? clockSeconds(d) : '')
  return (
    <StepCard
      n={4}
      title={t(mc.webhookTitle)}
      lead={t(mc.webhookLead)}
      footer={
        <>
          <Button onClick={onBack}>{t(mc.back)}</Button>
          <span className="flex flex-wrap items-center gap-3">
            {w.registered && w.status === 'waiting' ? <span className="k">{t(mc.noWait)}</span> : null}
            <Button variant="primary" disabled={!w.registered} loading={step.isPending} onClick={() => step.mutate({ params: { path: { workspace: slug, id: c.id } }, body: { step: 'settings' } }, { onSuccess: onNext })}>
              {t(mc.toSettings)}
            </Button>
          </span>
        </>
      }
    >
      {unreachable ? (
        <div className="flex flex-col gap-3">
          <ErrorPanel title={t(mc.unreachableTitle)}>{unreachable}</ErrorPanel>
          <section className="card flex flex-col gap-3 p-4">
            <h3 className="m-0 text-[15px] font-semibold">{t(mc.manualTitle)}</h3>
            <ol className="m-0 flex flex-col gap-2 pl-5 text-[13px]">
              <li>
                {t(mc.manual1)} <span className="ea-mono">{w.url}</span>{' '}
                <Button size="xs" icon="copy" onClick={() => void navigator.clipboard.writeText(w.url).then(() => toast({ kind: 'success', title: t(mc.copied) }))}>
                  {t(mc.copy)}
                </Button>
              </li>
              <li>
                {t(mc.manual2, { count: w.events.length })}{' '}
                <Button size="xs" icon="copy" onClick={() => void navigator.clipboard.writeText(w.events.join('\n')).then(() => toast({ kind: 'success', title: t(mc.copied) }))}>
                  {t(mc.copy)}
                </Button>
              </li>
              <li>{t(mc.manual3)}</li>
            </ol>
            <Field label={t(mc.signingSecret)} hint={t(mc.sealedNote)}>
              <Input type="password" autoComplete="off" value={secret} placeholder={'whsec_…'} onChange={(e) => setSecret(e.target.value)} />
            </Field>
            <div className="flex flex-wrap gap-2">
              <Button
                variant="primary"
                disabled={!secret}
                loading={save.isPending}
                onClick={() => save.mutate({ params: { path: { workspace: slug, id: c.id } }, body: { secret } }, { onSuccess: () => (setUnreachable(undefined), refresh()), onError: (e) => toast({ kind: 'error', title: errorMessage(e) }) })}
              >
                {t(mc.saveSecret)}
              </Button>
              <Button onClick={register} loading={reg.isPending}>
                {t(mc.checkAddress)}
              </Button>
            </div>
          </section>
        </div>
      ) : !w.registered ? (
        <Skeleton w="100%" h={160} />
      ) : (
        <ul className="ea-tl">
          <li>
            <span className="ic" aria-hidden="true">
              <Icon name="check" size={14} />
            </span>
            <div className="flex flex-col">
              <span className="t">{t(mc.endpointCreated)}</span>
              <span className="d ea-mono">{w.url.replace(/[^/]+$/, '…')}</span>
            </div>
            <span className="w">{at(w.registeredAt)}</span>
          </li>
          <li>
            <span className="ic" aria-hidden="true">
              <Icon name="check" size={14} />
            </span>
            <div className="flex flex-col gap-1.5">
              <span className="t">{t(mc.eventsSubscribed, { count: w.events.length })}</span>
              <span className="flex flex-wrap gap-1.5">
                {w.events.map((e) => (
                  <span key={e} className="badge sm mut ea-mono">
                    {e}
                  </span>
                ))}
              </span>
            </div>
            <span className="w">{at(w.registeredAt)}</span>
          </li>
          <li>
            <span className="ic" aria-hidden="true">
              <Icon name="check" size={14} />
            </span>
            <div className="flex flex-col">
              <span className="t">{t(mc.secretStored)}</span>
              <span className="d">{t(mc.secretStoredText)}</span>
            </div>
            <span className="w">{at(w.registeredAt)}</span>
          </li>
          <li>
            <span className={w.firstEventAt ? 'ic' : 'ic info'} aria-hidden="true">
              {w.firstEventAt ? <Icon name="check" size={14} /> : <span className="spin sm" />}
            </span>
            <div className="flex flex-col">
              <span className="t">{w.firstEventAt ? t(mc.firstEvent, { type: w.firstEventType ?? '' }) : t(mc.waiting)}</span>
              <span className="d">{w.firstEventAt ? t(mc.firstEventText) : t(mc.waitingText)}</span>
            </div>
            <span className="w">{at(w.firstEventAt)}</span>
          </li>
        </ul>
      )}
    </StepCard>
  )
}

function SettingsStep({ c, pay, onBack, onDone }: { c: Conn; pay: Schemas['EarnPayments']; onBack: () => void; onDone: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [tax, setTax] = useState(c.account.taxEnabled)
  const [incl, setIncl] = useState(false)
  const [portal, setPortal] = useState<Schemas['EarnPortal']>({ updateCard: true, invoices: true, billingEmail: true, switchPeriod: true, cancel: true })
  const finish = $api.useMutation('post', `${EARN}/connections/{id}/finish`)
  const items: [keyof Schemas['EarnPortal'], string][] = [
    ['updateCard', t(mc.pUpdateCard)],
    ['invoices', t(mc.pInvoices)],
    ['billingEmail', t(mc.pBillingEmail)],
    ['switchPeriod', t(mc.pSwitch)],
    ['cancel', t(mc.pCancel)],
  ]
  return (
    <StepCard
      n={5}
      title={t(mc.settingsTitle)}
      lead={t(mc.settingsLead)}
      footer={
        <>
          <Button onClick={onBack}>{t(mc.back)}</Button>
          <Button
            variant="primary"
            loading={finish.isPending}
            onClick={() =>
              finish.mutate(
                { params: { path: { workspace: slug, id: c.id } }, body: { taxAtCheckout: tax, pricesIncludeTax: incl, portal } },
                {
                  onSuccess: (r) => {
                    qc.setQueryData(['earn-finished', c.id], r)
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/status`] })
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
                    onDone()
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(mc.saveFinish)}
          </Button>
        </>
      }
    >
      <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))' }}>
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <span className="text-[13px] font-semibold">{t(mc.currency)}</span>
            <span className="flex items-center gap-2">
              <span className="badge mut">{(pay.checkout.currency || c.account.currency).toUpperCase()}</span>
              <span className="k">{t(mc.currencyHint, { account: c.account.name })}</span>
            </span>
          </div>
          <div className="card flex items-center justify-between gap-3 p-4">
            <div className="flex flex-col">
              <b>{t(mc.taxTitle)}</b>
              <span className="k">{c.account.taxEnabled ? t(mc.taxHintOn, { account: c.account.name }) : t(mc.taxHintOff, { account: c.account.name })}</span>
            </div>
            <Toggle label={t(mc.taxTitle)} hideLabel checked={tax} disabled={!c.account.taxEnabled} onChange={(e) => setTax(e.target.checked)} />
          </div>
          <Field label={t(mc.prices)}>
            <Select value={incl ? 'include' : 'exclude'} onChange={(e) => setIncl(e.target.value === 'include')}>
              <option value="exclude">{t(mc.pricesExclude)}</option>
              <option value="include">{t(mc.pricesInclude)}</option>
            </Select>
          </Field>
        </div>
        <fieldset className="card m-0 flex flex-col gap-2 p-4">
          <legend className="vh">{t(mc.portalTitle)}</legend>
          <b>{t(mc.portalTitle)}</b>
          {items.map(([k, label]) => (
            <label key={k} className="flex items-center gap-2 text-[13px]">
              <input type="checkbox" checked={portal[k]} onChange={(e) => setPortal({ ...portal, [k]: e.target.checked })} />
              {label}
            </label>
          ))}
        </fieldset>
      </div>
      <div className="note n" role="note">
        <Icon name="info" size={16} />
        <div>{t(mc.sellerNote)}</div>
      </div>
    </StepCard>
  )
}

function DoneStep({ c, pay }: { c: Conn; pay: Schemas['EarnPayments'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const fin = qc.getQueryData<Schemas['EarnFinished']>(['earn-finished', c.id])
  const w = c.webhook
  return (
    <StepCard n={6} done title={t(mc.doneTitle, { provider: c.providerName, account: c.account.name, mode: c.mode })} lead={c.mode === 'live' ? t(mc.doneLive, { account: c.account.name }) : t(mc.doneTest)}>
      <dl className="ea-kv">
        <dt>{t(mc.account)}</dt>
        <dd>{t(mc.accountLine, { name: c.account.name, country: countryName(c.account.country), currency: c.account.currency.toUpperCase() })}</dd>
        <dt>{t(mc.key)}</dt>
        <dd className="ea-mono">{t(mc.doneKey, { hint: c.keyHint, granted: c.granted, total: c.permissions.length })}</dd>
        <dt>{t(mc.sWebhook)}</dt>
        <dd>{t(mc.doneWebhook, { count: w.events.length, state: w.firstEventAt ? t(mc.webhookFirst, { time: clock(w.firstEventAt) }) : t(mc.webhookWaiting) })}</dd>
        <dt>{t(mc.checkout)}</dt>
        <dd>{t(mc.checkoutLine, { tax: pay.checkout.taxAtCheckout ? 'on' : 'off', incl: pay.checkout.pricesIncludeTax ? 'include' : 'exclude' })}</dd>
        {c.mode === 'live' ? (
          <>
            <dt>{t(mc.memberEmails)}</dt>
            <dd>{t(mc.memberEmailsLine, { count: fin?.drafts ?? 0 })}</dd>
          </>
        ) : (
          <>
            <dt>{t(mc.pages)}</dt>
            <dd>{t(mc.pagesTest)}</dd>
          </>
        )}
      </dl>
      <div className="flex flex-wrap gap-2">
        <ButtonLink to="/w/$workspace/earn/plans/tiers/new" params={{ workspace: slug }} variant="primary">
          {t(mc.createTier)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ import: 'new' }}>
          {t(mc.importMembers)}
        </ButtonLink>
        {c.mode === 'live' && !pay.test ? (
          <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }} search={{ mode: 'test' }}>
            {t(mc.addTestKey)}
          </ButtonLink>
        ) : c.mode === 'test' && !pay.live ? (
          <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }} search={{ mode: 'live' }}>
            {t(mc.connectLive)}
          </ButtonLink>
        ) : null}
        <ButtonLink to="/w/$workspace/earn" params={{ workspace: slug }}>
          {t(mc.toEarn)}
        </ButtonLink>
      </div>
    </StepCard>
  )
}
