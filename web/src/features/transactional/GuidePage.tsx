import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon, LogoMark } from '@/components/icons/Icon'
import { SystemBanners } from '@/components/shell/SystemBanners'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useDocumentTitle, useNow, useOverflowX } from '@/lib/hooks'
import { useMe, usePermissionReason, useWorkspace } from '@/lib/session'
import '@/features/connect/connect.css'
import { perSec, streamName, type Stream } from '@/features/providers/common'
import { codeSample, highlight } from './codeSamples'
import { mu } from './guide.messages'
import { stampMs } from './MessagePage'
import { mt } from './messages'
import { CodeBox, secs, type CodeLang } from './shared'

const API = '/api/v1/workspaces/{workspace}/transactional' as const
const W = '/api/v1/workspaces/{workspace}' as const

export type GuideStep = 1 | 2 | 3 | 4 | 5 | 'done'

type Setup = Schemas['TransactionalSetup']
type Overview = Schemas['TransactionalOverview']
type Detail = Schemas['TransactionalMessageDetail']
type Tr = ReturnType<typeof useT>

/** The step a workspace is at: the first one not done yet. */
export function firstOpen(setup: Pick<Setup, 'senders' | 'keys'>, ov: Pick<Overview, 'route' | 'firstRun'>): GuideStep {
  if (!setup.senders.length) return 1
  if (!ov.route.configured) return 2
  if (!ov.firstRun.published) return 3
  if (!setup.keys.length) return 4
  if (!ov.firstRun.messages) return 5
  return 'done'
}

/** What the guide set up in this visit, for the step list and the summary. */
interface Picked {
  template?: string
  key?: string
  message?: string
  monitored?: string
}

/**
 * The first-send guide (TransactionalGuideSteps, TransactionalGuide,
 * TransactionalGuideStates): sender, route, template and key, then a live
 * timeline of the first message. The step is in the URL, so a reload
 * resumes it; without one the guide opens at the first step not done.
 */
export function TransactionalGuidePage({ step, onStep }: { step?: GuideStep; onStep: (s: GuideStep) => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  useDocumentTitle(`${t(mu.title)} · ${me.workspace.name} · ReputeCast`)
  const canEdit = me.permissions.includes('transactional:edit')
  const path = { params: { path: { workspace: slug } } }
  const setup = $api.useQuery('get', `${API}/setup`, path, { enabled: canEdit })
  const ov = $api.useQuery('get', `${API}/overview`, { params: { path: { workspace: slug }, query: { period: '7d' } } }, { enabled: canEdit })
  const settings = $api.useQuery('get', `${API}/settings`, path, { enabled: canEdit })
  const [picked, setPicked] = useState<Picked>({})
  const pick = (p: Picked) => setPicked((x) => ({ ...x, ...p }))
  const current: GuideStep | undefined = step ?? (setup.data && ov.data ? firstOpen(setup.data, ov.data) : undefined)
  const tpl = useLiveTemplate(canEdit ? picked.template : undefined, canEdit)
  const error = setup.error ?? ov.error
  // The sender transactional templates send as: the default, else the first.
  const sender = setup.data?.senders.find((x) => x.id === settings.data?.defaultSender?.id) ?? setup.data?.senders[0]
  return (
    <div className="rc-app flex min-h-screen flex-col">
      <a href="#main" className="skip">
        {t(mu.skip)}
      </a>
      <header className="cn-head">
        <Link to="/w/$workspace" params={{ workspace: slug }} className="cn-home" aria-label={t(mu.home)}>
          <LogoMark size={26} />
          <span className="text-[15px] font-bold">ReputeCast</span>
        </Link>
        <span className="k" aria-hidden="true">
          /
        </span>
        <h1 className="m-0 text-[16px] font-semibold">{t(mu.title)}</h1>
        {current === 'done' ? <Badge tone="ok">{t(mu.doneBadge)}</Badge> : current ? <Badge tone="neutral">{t(mu.stepBadge, { n: current })}</Badge> : null}
        <ButtonLink to="/w/$workspace/transactional" params={{ workspace: slug }} className="ml-auto">
          {t(mu.exit)}
        </ButtonLink>
      </header>
      <SystemBanners />
      {!canEdit ? (
        <main className="cn-wiz" id="main">
          <ForbiddenState permission="transactional:edit" title={t(mt.forbiddenTitle)} />
        </main>
      ) : error ? (
        <main className="cn-wiz" id="main">
          <ErrorPanel
            title={errorMessage(error)}
            actions={
              <Button
                onClick={() => {
                  void setup.refetch()
                  void ov.refetch()
                }}
              >
                {t(mt.retry)}
              </Button>
            }
          />
        </main>
      ) : !setup.data || !ov.data || !current ? (
        <div className="cn-wiz" aria-busy="true">
          <div className="card cn-steps flex flex-col gap-3">
            {Array.from({ length: 5 }, (_, i) => (
              <Skeleton key={i} w="80%" h={14} />
            ))}
          </div>
          <main className="cn-main" id="main">
            <PanelSkeleton />
          </main>
        </div>
      ) : (
        <div className="cn-wiz">
          <Steps current={current} sender={sender} setup={setup.data} ov={ov.data} tpl={tpl} keyName={picked.key} />
          <main className="cn-main" id="main" tabIndex={-1}>
            {current === 1 ? <SenderStep setup={setup.data} onMonitored={(d) => pick({ monitored: d })} onNext={() => onStep(2)} /> : null}
            {current === 2 ? <RouteStep ov={ov.data} onBack={() => onStep(1)} onNext={() => onStep(3)} /> : null}
            {current === 3 ? (
              <TemplateStep
                onBack={() => onStep(2)}
                onNext={(key) => {
                  pick({ template: key })
                  onStep(4)
                }}
              />
            ) : null}
            {current === 4 ? (
              <KeyStep
                setup={setup.data}
                product={settings.data?.productName || me.workspace.name}
                tpl={tpl}
                onBack={() => onStep(3)}
                onNext={(name) => {
                  pick({ key: name })
                  onStep(5)
                }}
              />
            ) : null}
            {current === 5 ? (
              <WatchStep
                setup={setup.data}
                ov={ov.data}
                tpl={tpl}
                keyName={picked.key}
                onBack={() => onStep(4)}
                onDone={(id) => {
                  pick({ message: id })
                  onStep('done')
                }}
              />
            ) : null}
            {current === 'done' ? <Done sender={sender} setup={setup.data} ov={ov.data} tpl={tpl} picked={picked} /> : null}
          </main>
        </div>
      )}
    </div>
  )
}

type LiveTpl = { key: string; version: number; variables: Schemas['TransactionalVariable'][]; testData: Schemas['TransactionalTestData'][] } | undefined

/** The template the guide sends: the one picked, else password-reset, else the first live one. */
function useLiveTemplate(picked: string | undefined, enabled: boolean): LiveTpl {
  const { slug } = useWorkspace()
  const list = $api.useQuery('get', `${API}/templates`, { params: { path: { workspace: slug }, query: { period: '7d' } } }, { enabled })
  const live = (list.data?.items ?? []).filter((it) => it.liveVersion && !it.archived)
  const key = picked ?? live.find((it) => it.key === 'password-reset')?.key ?? live[0]?.key
  const q = $api.useQuery('get', `${API}/templates/{template}`, { params: { path: { workspace: slug, template: key ?? '' } } }, { enabled: enabled && Boolean(key) })
  if (!key || !q.data?.liveVersion) return undefined
  return { key, version: q.data.liveVersion, variables: q.data.variables, testData: q.data.testData }
}

type Sender = Setup['senders'][number]

function Steps({ current, sender, setup, ov, tpl, keyName }: { current: GuideStep; sender?: Sender; setup: Setup; ov: Overview; tpl: LiveTpl; keyName?: string }) {
  const t = useT()
  const ref = useRef<HTMLElement>(null)
  // On phones the steps are one sideways strip: while it scrolls it takes
  // keyboard focus, so it can be read without a mouse (as in the setup wizard).
  const scrolls = useOverflowX(ref)
  const n = current === 'done' ? 6 : current
  const route = ov.route.primary ? (ov.route.fallback ? `${ov.route.primary.name} → ${ov.route.fallback.name}` : ov.route.primary.name) : ''
  const items: { title: string; line: string }[] = [
    { title: t(mu.sender), line: sender?.email ?? '' },
    { title: t(mu.route), line: route },
    { title: t(mu.template), line: tpl ? t(mu.live, { key: tpl.key, version: tpl.version }) : '' },
    { title: t(mu.key), line: keyName ?? setup.keys[0]?.name ?? '' },
    { title: t(mu.watch), line: t(mu.watchLine) },
  ]
  return (
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <nav ref={ref} className="card cn-steps" aria-label={t(mu.stepsLabel)} tabIndex={scrolls ? 0 : undefined}>
      {items.map((d, i) => {
        const k = i + 1
        const done = k < n
        const cur = k === n
        return (
          <div key={d.title} className={cn('step cn-step', cur && 'cur', done && 'done')} aria-current={cur ? 'step' : undefined}>
            <span className="num" aria-hidden="true">
              {done ? <Icon name="check" size={14} /> : k}
            </span>
            <div>
              <div className="font-semibold">
                {done ? <span className="vh">{t(mu.doneSr)}</span> : null}
                {d.title}
              </div>
              {d.line ? <div className="k">{d.line}</div> : null}
            </div>
          </div>
        )
      })}
    </nav>
  )
}

/** A step's panel: the card with its badge, title and lead (steps 1 to 4). */
function Panel({ n, title, lead, children, footer }: { n: number; title: string; lead: ReactNode; children: ReactNode; footer: ReactNode }) {
  const t = useT()
  return (
    <section className="card flex flex-col gap-4" style={{ padding: '22px 24px' }} aria-labelledby={`tx-guide-${n}`}>
      <div className="flex flex-col gap-1">
        <Badge tone="neutral" className="self-start">
          {t(mu.stepBadge, { n })}
        </Badge>
        <h2 id={`tx-guide-${n}`} className="tx-ttl" style={{ fontSize: 21 }}>
          {title}
        </h2>
        <p className="tx-lead">{lead}</p>
      </div>
      {children}
      <div className="flex flex-wrap justify-between gap-2.5 border-t border-line-2 pt-3.5">{footer}</div>
    </section>
  )
}

function PanelSkeleton() {
  return (
    <section className="card flex flex-col gap-4" style={{ padding: '22px 24px' }} aria-busy="true">
      <Skeleton w="20%" h={20} />
      <Skeleton w="60%" h={22} />
      <Skeleton w="100%" h={120} />
    </section>
  )
}

const NEW = '__new'

/** The domains senders can use, Transactional ones first, and a subdomain to suggest when none is. */
function domainChoices(domains: Schemas['SendingDomainRow'][]): { options: { value: string; isNew: boolean }[]; suggestion?: string } {
  const tx = domains.filter((d) => d.transactional)
  const rest = domains.filter((d) => !d.transactional)
  const root = [...domains].sort((a, b) => a.domain.split('.').length - b.domain.split('.').length || a.domain.length - b.domain.length)[0]?.domain
  const suggestion = !tx.length && root && !domains.some((d) => d.domain === `notify.${root}`) ? `notify.${root}` : undefined
  return {
    suggestion,
    options: [...tx.map((d) => ({ value: d.domain, isNew: false })), ...(suggestion ? [{ value: suggestion, isNew: true }] : []), ...rest.map((d) => ({ value: d.domain, isNew: false }))],
  }
}

/** "DKIM for Postmark and for SES, a return path for each" from the records still to add. */
function recordsSentence(t: Tr, records: { purpose: string; provider: string }[]): string {
  const uniq = (xs: string[]) => [...new Set(xs.filter(Boolean))]
  const dkim = uniq(records.filter((r) => r.purpose === 'dkim').map((r) => r.provider))
  const rp = uniq(records.filter((r) => r.purpose === 'return_path').map((r) => r.provider))
  const parts: string[] = []
  if (dkim.length) parts.push(t(mu.dnsDkim, { providers: dkim.join(t(mu.andFor)) }))
  if (rp.length) parts.push(rp.length > 1 && rp.length === dkim.length && rp.every((p) => dkim.includes(p)) ? t(mu.dnsReturnEach) : t(mu.dnsReturn, { providers: rp.join(t(mu.andFor)) }))
  if (records.some((r) => r.purpose === 'spf')) parts.push(t(mu.dnsSpf))
  if (records.some((r) => r.purpose === 'dmarc')) parts.push(t(mu.dnsDmarc))
  if (records.some((r) => r.purpose === 'verification')) parts.push(t(mu.dnsVerification))
  const s = parts.join(', ')
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function SenderStep({ setup, onNext, onMonitored }: { setup: Setup; onNext: () => void; onMonitored: (domain: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const path = { params: { path: { workspace: slug } } }
  const so = $api.useQuery('get', `${W}/senders`, path)
  const st = $api.useQuery('get', `${API}/settings`, path)
  if (so.error || st.error)
    return (
      <ErrorPanel
        title={errorMessage(so.error ?? st.error)}
        actions={
          <Button
            onClick={() => {
              void so.refetch()
              void st.refetch()
            }}
          >
            {t(mt.retry)}
          </Button>
        }
      />
    )
  if (!so.data || !st.data) return <PanelSkeleton />
  return <SenderForm setup={setup} so={so.data} settings={st.data} onNext={onNext} onMonitored={onMonitored} />
}

function SenderForm({
  setup,
  so,
  settings,
  onNext,
  onMonitored,
}: {
  setup: Setup
  so: Schemas['SendersOverview']
  settings: Schemas['TransactionalSettings']
  onNext: () => void
  onMonitored: (domain: string) => void
}) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const current = setup.senders.find((s) => s.id === settings.defaultSender?.id) ?? setup.senders[0]
  const { options, suggestion } = domainChoices(so.domains)
  const [name, setName] = useState(current?.name ?? (settings.productName || me.workspace.name))
  const [local, setLocal] = useState(current ? current.email.slice(0, current.email.lastIndexOf('@')) : 'no-reply')
  const [domain, setDomain] = useState(current?.domain ?? options[0]?.value ?? NEW)
  const [typed, setTyped] = useState('')
  const [busy, setBusy] = useState(false)
  const denied = usePermissionReason('domains:manage', so.helpers)
  const chosen = (domain === NEW ? typed : domain).trim().toLowerCase()
  const row = so.domains.find((d) => d.domain === chosen)
  const email = `${local.trim().toLowerCase()}@${chosen}`
  const existing = so.senders.find((s) => s.email === email)
  const isNew = !row && Boolean(chosen)
  const detail = $api.useQuery('get', `${W}/domains/{domainId}`, { params: { path: { workspace: slug, domainId: row?.id ?? '' } } }, { enabled: Boolean(row && row.status !== 'verified') })
  const todo = (detail.data?.records ?? []).filter((r) => r.status !== 'verified').map((r) => ({ purpose: r.purpose, provider: r.provider }))
  const valid = Boolean(name.trim() && /^[a-z0-9._+-]+$/i.test(local.trim()) && /^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(chosen))
  // Nothing changes when the sender is already there: Continue just moves on.
  const needsWrite = !existing || existing.name !== name.trim() || !row?.transactional
  const save = async () => {
    setBusy(true)
    try {
      let id = row?.id
      if (!row) {
        const { data } = await http.POST(`${W}/domains`, { params: { path: { workspace: slug } }, body: { domain: chosen, note: t(mu.domainNote) } })
        id = data!.domain.id
      }
      if (!row?.transactional) await http.PATCH(`${W}/domains/{domainId}`, { params: { path: { workspace: slug, domainId: id! } }, body: { transactional: true } })
      if (!existing) {
        await http.POST(`${W}/senders`, {
          params: { path: { workspace: slug } },
          body: { name: name.trim(), email, defaultFor: settings.defaultSender ? [] : ['transactional'] },
        })
      } else if (existing.name !== name.trim()) {
        await http.PATCH(`${W}/senders/{senderId}`, { params: { path: { workspace: slug, senderId: existing.id } }, body: { name: name.trim() } })
      }
      await qc.invalidateQueries()
      onNext()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const providerNames = so.providers.map((p) => p.short || p.name)
  return (
    <Panel
      n={1}
      title={t(mu.senderTitle)}
      lead={t(mu.senderBody)}
      footer={
        <>
          <ButtonLink to="/w/$workspace/transactional" params={{ workspace: slug }}>
            {t(mu.cancel)}
          </ButtonLink>
          <Button
            variant="primary"
            loading={busy}
            disabledReason={!valid ? t(mu.senderInvalid) : needsWrite ? denied : undefined}
            onClick={() => (needsWrite ? void save() : onNext())}
          >
            {t(mu.continueRoute)}
          </Button>
        </>
      }
    >
      <div className="grid gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(320px, 100%), 1fr))' }}>
        <label className="lbl">
          {t(mu.fromName)}
          <input className="in" value={name} maxLength={120} onChange={(e) => setName(e.target.value)} />
        </label>
        <div className="lbl" role="group" aria-labelledby="tx-guide-from">
          <span id="tx-guide-from">{t(mu.fromAddress)}</span>
          <div className="flex items-center gap-1.5">
            <input className="in w-0 min-w-0" style={{ flex: 1 }} aria-label={t(mu.localPart)} value={local} maxLength={64} onChange={(e) => setLocal(e.target.value)} />
            <span className="k" aria-hidden="true">
              @
            </span>
            <select className="in w-0 min-w-0" style={{ flex: 1.4 }} aria-label={t(mu.domainLabel)} value={domain} onChange={(e) => setDomain(e.target.value)}>
              {options.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.isNew ? t(mu.domainNew, { domain: o.value }) : o.value}
                </option>
              ))}
              <option value={NEW}>{t(mu.domainOther)}</option>
            </select>
          </div>
          {domain === NEW ? (
            <input
              className="in mt-1.5"
              aria-label={t(mu.newDomain)}
              placeholder={suggestion ?? 'notify.example.com'}
              value={typed}
              maxLength={253}
              onChange={(e) => setTyped(e.target.value)}
            />
          ) : null}
        </div>
      </div>
      {row && row.status !== 'verified' && detail.data ? (
        <Note tone="info">
          <div className="flex flex-col gap-1.5">
            <span className="font-bold">{t(mu.dnsNeeds, { domain: row.domain, count: todo.length })}</span>
            <span>
              {todo.length ? `${recordsSentence(t, todo)}. ` : ''}
              {t(mu.dnsLive)}{' '}
              <Link to="/w/$workspace/settings/senders/$domainId" params={{ workspace: slug, domainId: row.id }}>
                {t(mu.dnsOpen)}
              </Link>
            </span>
          </div>
        </Note>
      ) : isNew && valid ? (
        <Note tone="info">
          <div className="flex flex-col gap-1.5">
            <span className="font-bold">{t(mu.dnsNew, { domain: chosen })}</span>
            <span>
              {providerNames.length
                ? `${recordsSentence(
                    t,
                    providerNames.flatMap((p) => [
                      { purpose: 'dkim', provider: p },
                      { purpose: 'return_path', provider: p },
                    ]),
                  )}. `
                : ''}
              {t(mu.dnsLiveAfter)}
            </span>
          </div>
        </Note>
      ) : null}
      {chosen && valid ? <MonitorRow domain={chosen} onMonitored={onMonitored} /> : null}
    </Panel>
  )
}

/** "Add notify.acme.io to HostRepute monitoring", when HostRepute is connected and the domain isn't watched. */
function MonitorRow({ domain, onMonitored }: { domain: string; onMonitored: (domain: string) => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const connected = me.hostRepute?.status === 'connected'
  const canView = me.permissions.includes('deliverability:view')
  const ints = $api.useQuery('get', `${W}/integrations`, { params: { path: { workspace: slug } } }, { enabled: connected && canView })
  const add = $api.useMutation('post', `${W}/integrations/hostrepute/assets`)
  const denied = usePermissionReason('integrations:hostrepute')
  const hr = ints.data?.hostRepute
  if (!connected || !hr || hr.status !== 'connected') return null
  const asset = hr.assets.find((a) => a.asset === domain)
  if (asset?.monitored)
    return (
      <Note tone="ok">
        <span>{t(mu.monitored, { domain })}</span>
      </Note>
    )
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-[10px] border border-line px-3.5 py-3">
      <span className="flex min-w-0 flex-[1_1_320px] items-center gap-2.5">
        <span className="flex text-accent">
          <Icon name="shield" size={18} />
        </span>
        <span className="flex min-w-0 flex-col">
          <span className="font-semibold">{t(mu.monitorTitle, { domain })}</span>
          <span className="k">{t(mu.monitorBody, { count: hr.sources })}</span>
        </span>
      </span>
      <Button
        size="sm"
        loading={add.isPending}
        disabledReason={denied}
        onClick={() =>
          add.mutate(
            { params: { path: { workspace: slug } }, body: { asset: domain } },
            {
              onSuccess: () => {
                toast({ kind: 'success', title: t(mu.monitored, { domain }) })
                onMonitored(domain)
                void qc.invalidateQueries()
              },
              onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
            },
          )
        }
      >
        {t(mu.monitorAdd)}
      </Button>
    </div>
  )
}

function statusWord(t: Tr, p: Schemas['RoutingProvider']): string {
  switch (p.status) {
    case 'healthy':
      return t(mu.provHealthy)
    case 'warming_up':
      return p.warmupDay && p.warmupDays ? t(mu.provWarmingDay, { day: p.warmupDay, days: p.warmupDays }) : t(mu.provWarming)
    case 'down':
      return t(mu.provDown)
    case 'degraded':
      return t(mu.provDegraded)
    case 'sandbox':
      return t(mu.provSandbox)
    case 'paused':
      return t(mu.provPaused)
  }
  return p.status
}

function RouteStep({ ov, onBack, onNext }: { ov: Overview; onBack: () => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${W}/routing`, { params: { path: { workspace: slug } } })
  // The providers' send rates, for "first within its 14 per second".
  const cards = $api.useQuery('get', `${W}/providers`, { params: { path: { workspace: slug } } })
  if (q.error) return <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(mt.retry)}</Button>} />
  if (!q.data) return <PanelSkeleton />
  const rates = new Map((cards.data?.providers ?? []).map((c) => [c.id, c.rate.max]))
  return <RouteForm key={q.data.savedAt ?? 'none'} cfg={q.data} rates={rates} ov={ov} onBack={onBack} onNext={onNext} />
}

function RouteForm({
  cfg,
  rates,
  ov,
  onBack,
  onNext,
}: {
  cfg: Schemas['RoutingConfig']
  rates: Map<string, number>
  ov: Overview
  onBack: () => void
  onNext: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const providers = cfg.providers
  const stream = cfg.streams.find((s) => s.stream === 'transactional')
  const configured = Boolean(ov.route.configured && stream?.splits.length)
  const healthyFirst = [...providers].sort((a, b) => Number(b.status === 'healthy') - Number(a.status === 'healthy'))
  const [primary, setPrimary] = useState(configured ? stream!.splits[0]!.providerId : (ov.route.system?.id ?? healthyFirst[0]?.id ?? ''))
  const [fallback, setFallback] = useState<string | undefined>(configured ? (stream!.fallback ?? '') : undefined)
  const [priority, setPriority] = useState(stream?.priority ?? ov.route.priority ?? true)
  const save = $api.useMutation('put', `${W}/routing`)
  const denied = usePermissionReason('providers:manage')
  const others = providers.filter((p) => p.id !== primary)
  // The fallback defaults to the first other healthy provider, and never is the primary.
  const fb = fallback !== undefined && fallback !== primary ? fallback : (others.find((p) => p.status === 'healthy')?.id ?? '')
  const usedBy = (id: string) =>
    cfg.streams
      .filter((s) => s.stream !== 'transactional' && s.splits.some((x) => x.providerId === id))
      .map((s) => streamName[s.stream as Stream])
      .filter(Boolean)
  const p = providers.find((x) => x.id === primary)
  const shared = [primary, fb]
    .map((id) => providers.find((x) => x.id === id))
    .find((x) => x && cfg.streams.some((s) => s.stream === 'broadcast' && s.splits.some((y) => y.providerId === x.id)))
  if (!providers.length)
    return (
      <Panel
        n={2}
        title={t(mu.routeTitle)}
        lead={t(mu.routeBody)}
        footer={
          <>
            <Button onClick={onBack}>{t(mu.back)}</Button>
            <ButtonLink variant="primary" to="/w/$workspace/providers" params={{ workspace: slug }}>
              {t(mu.connectProvider)}
            </ButtonLink>
          </>
        }
      >
        <Note tone="warn">{t(mu.noProviders)}</Note>
      </Panel>
    )
  const label = (x: Schemas['RoutingProvider']) => {
    const used = usedBy(x.id)
    return [x.name, statusWord(t, x), used.length ? t(mu.usedBy, { streams: used.join(', ') }) : ''].filter(Boolean).join(' · ')
  }
  const strict = p && (p.bulk === 'forbidden' || p.bulk === 'broadcast_streams_only')
  return (
    <Panel
      n={2}
      title={t(mu.routeTitle)}
      lead={t(mu.routeBody)}
      footer={
        <>
          <Button onClick={onBack}>{t(mu.back)}</Button>
          <Button
            variant="primary"
            loading={save.isPending}
            disabledReason={denied}
            onClick={() =>
              save.mutate(
                {
                  params: { path: { workspace: slug } },
                  body: {
                    streams: [
                      {
                        stream: 'transactional',
                        splits: [{ providerId: primary, weight: 100 }],
                        fallback: fb,
                        failover: stream?.failover ?? 'automatic',
                        // Conditions already on the route stay as they are.
                        conditions: (stream?.conditions ?? []).map((c) => ({ field: c.field, values: c.values, providerId: c.providerId, note: c.note })),
                        priority,
                      },
                    ],
                  },
                },
                {
                  onSuccess: () => {
                    void qc.invalidateQueries()
                    onNext()
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(mu.continueTemplate)}
          </Button>
        </>
      }
    >
      <div className="grid gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(320px, 100%), 1fr))' }}>
        <label className="lbl">
          {t(mu.primary)}
          <select className="in" value={primary} onChange={(e) => setPrimary(e.target.value)}>
            {providers.map((x) => (
              <option key={x.id} value={x.id}>
                {label(x)}
              </option>
            ))}
          </select>
          {p && strict ? <span className="k font-medium">{t(mu.termsTransactional, { name: p.name })}</span> : null}
          {p && p.status === 'warming_up' ? <span className="k font-medium">{t(mu.warmingHint, { name: p.name })}</span> : null}
        </label>
        <label className="lbl">
          {t(mu.fallbackLabel)}
          <select className="in" value={fb} onChange={(e) => setFallback(e.target.value)}>
            {others.map((x) => (
              <option key={x.id} value={x.id}>
                {[x.name, statusWord(t, x)].join(' · ')}
              </option>
            ))}
            <option value="">{t(mu.noFallback)}</option>
          </select>
          <span className="k font-medium">{t(mu.fallbackHint)}</span>
        </label>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-[10px] border border-line px-3.5 py-3">
        <span className="flex min-w-0 flex-[1_1_320px] flex-col">
          <span className="font-semibold" id="tx-guide-priority">
            {t(mu.priority)}
          </span>
          <span className="k">
            {shared
              ? rates.get(shared.id)
                ? t(mu.priorityRate, { name: shared.short || shared.name, rate: perSec(rates.get(shared.id)!) })
                : t(mu.priorityShared, { name: shared.short || shared.name })
              : t(mu.priorityBody)}
          </span>
        </span>
        <input type="checkbox" role="switch" className="tg" aria-labelledby="tx-guide-priority" checked={priority} onChange={(e) => setPriority(e.target.checked)} />
      </div>
    </Panel>
  )
}

function TemplateStep({ onBack, onNext }: { onBack: () => void; onNext: (key: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const starters = $api.useQuery('get', `${API}/starters`, { params: { path: { workspace: slug } } })
  const list = $api.useQuery('get', `${API}/templates`, { params: { path: { workspace: slug }, query: { period: '7d' } } })
  const [picked, setPicked] = useState<string>()
  const [busy, setBusy] = useState(false)
  const canEdit = list.data?.canEdit ?? true
  const denied = usePermissionReason('transactional:publish')
  const items = starters.data?.items ?? []
  const choice = picked ?? items[0]?.key
  const existing = (key?: string) => list.data?.items.find((it) => it.key === key && !it.archived)
  const ex = existing(choice)
  const go = async () => {
    const s = items.find((x) => x.key === choice)
    if (!s) return
    // A template with the starter's ID is used as it is: published if it has
    // only a draft, never created twice.
    if (ex?.liveVersion) return onNext(ex.key)
    setBusy(true)
    try {
      let rev: number
      if (ex) {
        const { data } = await http.GET(`${API}/templates/{template}`, { params: { path: { workspace: slug, template: ex.key } } })
        rev = data!.draftRevision
      } else {
        const { data } = await http.POST(`${API}/templates`, { params: { path: { workspace: slug } }, body: { name: s.name, key: s.key, starter: s.key } })
        rev = data!.draftRevision
      }
      await http.POST(`${API}/templates/{template}/publish`, { params: { path: { workspace: slug, template: s.key } }, body: { baseRevision: rev } })
      toast({ kind: 'success', title: t(mu.published, { key: s.key }) })
      void qc.invalidateQueries()
      onNext(s.key)
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
      void qc.invalidateQueries()
    } finally {
      setBusy(false)
    }
  }
  return (
    <Panel
      n={3}
      title={t(mu.templateTitle)}
      lead={t(mu.templateBody)}
      footer={
        <>
          <Button onClick={onBack}>{t(mu.back)}</Button>
          <span className="flex flex-wrap gap-2">
            <Button
              disabledReason={canEdit ? undefined : list.data?.readOnlyReason}
              onClick={() =>
                ex
                  ? void navigate({ to: '/w/$workspace/transactional/templates/$templateId', params: { workspace: slug, templateId: ex.key } })
                  : void navigate({ to: '/w/$workspace/transactional/templates/new', params: { workspace: slug }, search: { starter: choice, name: items.find((x) => x.key === choice)?.name } })
              }
            >
              {t(mu.editFirst)}
            </Button>
            <Button variant="primary" loading={busy} disabledReason={!choice ? t(mt.loading) : ex?.liveVersion ? undefined : denied} onClick={() => void go()}>
              {ex?.liveVersion ? t(mu.useLive, { version: ex.liveVersion }) : t(mu.publishContinue)}
            </Button>
          </span>
        </>
      }
    >
      {starters.error ? (
        <ErrorPanel title={errorMessage(starters.error)} actions={<Button onClick={() => void starters.refetch()}>{t(mt.retry)}</Button>} />
      ) : !starters.data ? (
        <Skeleton w="100%" h={150} />
      ) : (
        <div className="grid gap-2.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(200px, 100%), 1fr))' }} role="radiogroup" aria-label={t(mu.templateTitle)}>
          {items.map((s) => {
            const on = choice === s.key
            const e = existing(s.key)
            return (
              <button
                key={s.key}
                type="button"
                role="radio"
                aria-checked={on}
                className={cn('flex flex-col items-start gap-1 rounded-[10px] p-3.5 text-left', on ? 'border-2 border-accent' : 'border border-line bg-white')}
                style={on ? { background: '#F0FDFA' } : undefined}
                onClick={() => setPicked(s.key)}
              >
                <span className="font-semibold">{s.name}</span>
                <span className="k">
                  {t(mu.variables, { count: s.variables.length })}
                  {s.attachment ? ` · ${t(mu.attachment)}` : ''}
                  {e ? ` · ${e.liveVersion ? t(mu.liveAs, { version: e.liveVersion }) : t(mu.draftOnly)}` : ''}
                </span>
              </button>
            )
          })}
        </div>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <span className="k">{t(mu.importInstead)}</span>
        <Link to="/w/$workspace/transactional/import" params={{ workspace: slug }} className="tx-lnk">
          {t(mu.importLink)}
        </Link>
      </div>
    </Panel>
  )
}

function KeyStep({ setup, product, tpl, onBack, onNext }: { setup: Setup; product: string; tpl: LiveTpl; onBack: () => void; onNext: (name: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [name, setName] = useState(t(mu.keyNameDefault, { product }))
  const [created, setCreated] = useState<Schemas['APIKeyCreated']>()
  const [lang, setLang] = useState<CodeLang>('curl')
  const create = $api.useMutation('post', `${W}/api-keys`)
  const denied = usePermissionReason('apikeys:manage', setup.keyHelpers)
  const last = useLastCall(tpl?.key)
  const code = codeSample(lang, { endpoint: `${setup.baseUrl}/messages`, template: tpl?.key ?? 'password-reset', variables: tpl?.variables ?? [], literal: { data: sampleData(tpl), ...last } })
  const existing = setup.keys.find((k) => k.mode === 'live' && !k.expired)
  const copy = (s: string) =>
    void navigator.clipboard?.writeText(s).then(
      () => toast({ kind: 'success', title: t(mu.keyCopied) }),
      () => toast({ kind: 'error', title: t(mu.copyFailed) }),
    )
  return (
    <Panel
      n={4}
      title={t(mu.keyTitle)}
      lead={t(mu.keyBody)}
      footer={
        <>
          <Button onClick={onBack}>{t(mu.back)}</Button>
          <span className="flex flex-wrap gap-2">
            {existing && !created ? <Button onClick={() => onNext(existing.name)}>{t(mu.useKey, { name: existing.name })}</Button> : null}
            <Button variant="primary" disabledReason={created ? undefined : t(mu.createFirst)} onClick={() => created && onNext(created.key.name)}>
              {t(mu.copied)}
            </Button>
          </span>
        </>
      }
    >
      {created ? (
        <div className="flex items-center gap-2">
          <input className="in mono min-w-0" style={{ flex: 1, fontSize: 12.5 }} aria-label={t(mu.newKey)} value={created.secret} readOnly onFocus={(e) => e.target.select()} />
          <Button variant="primary" icon="copy" onClick={() => copy(created.secret)}>
            {t(mu.copy)}
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-2.5">
          {existing ? <Note tone="info">{t(mu.keyExists, { name: existing.name })}</Note> : null}
          <div className="flex flex-wrap items-end gap-2">
            <label className="lbl min-w-0" style={{ flex: '1 1 260px' }}>
              {t(mu.keyName)}
              <input className="in" value={name} maxLength={80} onChange={(e) => setName(e.target.value)} />
            </label>
            <Button
              variant="primary"
              loading={create.isPending}
              disabledReason={denied ?? (name.trim() ? undefined : t(mu.keyNameMissing))}
              onClick={() =>
                create.mutate(
                  {
                    params: { path: { workspace: slug } },
                    body: { name: name.trim(), permissions: ['transactional:send'], transactional: { preset: 'send_only', mode: 'live' } },
                  },
                  {
                    onSuccess: (c) => {
                      setCreated(c)
                      void qc.invalidateQueries()
                    },
                    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                  },
                )
              }
            >
              {t(mu.createKey)}
            </Button>
          </div>
        </div>
      )}
      <CodeBox lang={lang} onLang={setLang} code={code}>
        {highlight(code, lang)}
      </CodeBox>
    </Panel>
  )
}

/** The tags and metadata the app sent the template's last message with, for the sample call. */
function useLastCall(template?: string): { tags?: string[]; metadata?: Record<string, string> } {
  const { slug } = useWorkspace()
  const list = $api.useQuery('get', `${API}/messages`, { params: { path: { workspace: slug }, query: { template, range: '30d', limit: 1 } } }, { enabled: Boolean(template) })
  const id = list.data?.items[0]?.id
  const d = $api.useQuery('get', `${API}/messages/{messageId}`, { params: { path: { workspace: slug, messageId: id ?? '' } } }, { enabled: Boolean(id) })
  return d.data ? { tags: d.data.tags, metadata: d.data.metadata } : {}
}

/** The data "Send it from here instead" sends: the template's first test data set, else its samples. */
function sampleData(tpl: LiveTpl): Record<string, unknown> {
  if (!tpl) return {}
  const set = tpl.testData[0]?.data
  if (set) return set as Record<string, unknown>
  const out: Record<string, unknown> = {}
  for (const v of tpl.variables) if (v.sample) out[v.name] = v.type === 'number' ? Number(v.sample) : v.sample
  return out
}

type Row = { title: string; detail?: ReactNode; state: 'done' | 'wait' | 'todo' | 'bad' }

/** The guide's timeline from a message's detail: call, render, hand-off and delivery, with what's still to come. */
function timelineRows(t: Tr, d: Detail, provider: string): Row[] {
  const at = (kind: string) => d.timeline.find((e) => e.kind === kind)
  const accepted = at('accepted')
  const rendered = at('rendered')
  const handed = at('handed')
  const delivered = at('delivered')
  const bad = d.timeline.find((e) => ['bounced', 'failed', 'rejected', 'not_delivered', 'provider_rejected'].includes(e.kind))
  const who = d.keyRef ? t(mu.byKey, { key: d.keyRef.name, ip: d.ip ? ` · ${d.ip}` : '' }) : d.sentBy ? t(mu.byPerson, { name: d.sentBy }) : ''
  const size = d.contentBytes >= 1024 ? `${Math.round(d.contentBytes / 1024)} KB` : `${d.contentBytes} B`
  const rows: Row[] = [
    { title: t(mu.callReceived), detail: t(mu.callReceivedDetail, { time: stampMs(accepted?.at ?? d.at), who, status: accepted?.status ?? 202 }), state: 'done' },
    {
      title: d.template ? t(mu.renderedTpl, { key: d.template.key, version: d.template.version ?? '' }) : t(mu.rendered),
      detail: rendered ? t(mu.renderedDetail, { time: stampMs(rendered.at), size, count: d.variables }) : t(mu.afterCall),
      state: rendered ? 'done' : 'wait',
    },
    {
      title: t(mu.handed, { provider: handed?.provider ?? d.provider?.name ?? provider }),
      detail: handed ? t(mu.handedDetail, { time: stampMs(handed.at), which: d.route.failedOver ? t(mu.fallbackWord) : t(mu.primaryWord) }) : undefined,
      state: handed ? 'done' : rendered ? 'wait' : 'todo',
    },
  ]
  if (bad) {
    rows.push({ title: t(mu.notArrived), detail: `${stampMs(bad.at)} · ${[bad.code, bad.detail].filter(Boolean).join(' ')}`, state: 'bad' })
    return rows
  }
  const host = delivered?.detail?.match(/^Accepted by (\S+)/)?.[1] ?? d.email.slice(d.email.lastIndexOf('@') + 1)
  const answer = delivered ? [delivered.code, delivered.detail?.split(' · ').slice(1).join(' · ')].filter(Boolean).join(' ') : ''
  rows.push({
    title: delivered ? t(mu.deliveredTo, { host }) : t(mu.delivered),
    detail: delivered ? (
      <>
        {t(mu.deliveredDetail, { time: stampMs(delivered.at), provider: delivered.provider ?? d.provider?.name ?? provider })}
        {answer ? (
          <>
            {' · '}
            <span className="mono">{answer}</span>
          </>
        ) : null}
      </>
    ) : undefined,
    state: delivered ? 'done' : handed ? 'wait' : 'todo',
  })
  return rows
}

function TimelineRow({ r }: { r: Row }) {
  return (
    <div className="tx-ev">
      <span className="ic" style={{ background: r.state === 'done' ? '#DCFCE7' : r.state === 'bad' ? '#FEE2E2' : r.state === 'wait' ? '#EEF0F3' : '#F3F4F6' }}>
        {r.state === 'wait' ? <span className="spin sm" aria-hidden="true" /> : r.state === 'done' ? <Icon name="check" size={14} /> : r.state === 'bad' ? <Icon name="alertCircle" size={14} /> : null}
      </span>
      <div className="tx">
        <span className="font-semibold" style={r.state === 'todo' || r.state === 'wait' ? { color: '#5B6472' } : undefined}>
          {r.title}
        </span>
        {r.detail ? <span className="k">{r.detail}</span> : null}
      </div>
    </div>
  )
}

const settled = ['delivered', 'bounced', 'complained', 'failed', 'rejected']

function WatchStep({
  setup,
  ov,
  tpl,
  keyName,
  onBack,
  onDone,
}: {
  setup: Setup
  ov: Overview
  tpl: LiveTpl
  keyName?: string
  onBack: () => void
  onDone: (id: string) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const user = useMe().data?.user
  const now = useNow(1000)
  const [since] = useState(() => Date.now())
  const [lang, setLang] = useState<CodeLang>('curl')
  const [sentId, setSentId] = useState<string>()
  const [rejected, setRejected] = useState<{ at: string; status: number; code: string; detail: string }>()
  const [publishing, setPublishing] = useState(false)
  const provider = ov.route.primary?.name ?? ''
  const key = keyName ?? setup.keys[0]?.name
  const tplKey = tpl?.key ?? 'password-reset'
  const last = useLastCall(tpl?.key)
  const code = codeSample(lang, { endpoint: `${setup.baseUrl}/messages`, template: tplKey, variables: tpl?.variables ?? [], literal: { data: sampleData(tpl), ...last } })
  // Waits for the first message after the step opened, from the app or from here.
  const log = $api.useQuery('get', `${API}/messages`, { params: { path: { workspace: slug }, query: { range: '1h', limit: 10 } } }, { refetchInterval: sentId ? false : 2000 })
  const fresh = (log.data?.items ?? []).filter((it) => Date.parse(it.at) >= since - 1000)
  const found = sentId ?? fresh[fresh.length - 1]?.id
  const detail = $api.useQuery(
    'get',
    `${API}/messages/{messageId}`,
    { params: { path: { workspace: slug, messageId: found ?? '' } } },
    { enabled: Boolean(found), refetchInterval: (q) => (q.state.data && settled.includes(q.state.data.status) ? false : 1500) },
  )
  const d = detail.data
  const delivered = d?.status === 'delivered' || d?.status === 'complained'
  const send = $api.useMutation('post', `${API}/messages`)
  const sendDenied = usePermissionReason('transactional:send')
  const waited = Math.max(0, Math.round((now.getTime() - since) / 1000))
  const waitedText = waited >= 60 ? t(mu.waitedMin, { min: Math.floor(waited / 60), sec: waited % 60 }) : t(mu.waitedSec, { sec: waited })
  const sendHere = (primary: boolean) => (
    <div className="flex flex-wrap items-center gap-2.5">
      <Button
        size={primary ? 'sm' : 'md'}
        variant={primary ? 'primary' : 'default'}
        loading={send.isPending}
        disabledReason={!tpl || !user ? t(mt.loading) : sendDenied}
        onClick={() =>
          send.mutate(
            { params: { path: { workspace: slug } }, body: { template: tpl!.key, to: { email: user!.email, name: user!.name }, data: sampleData(tpl) } },
            {
              onSuccess: (r) => {
                setRejected(undefined)
                setSentId(r.id)
              },
              onError: (e) => {
                if (e instanceof ApiError) setRejected({ at: new Date().toISOString(), status: e.status, code: e.code ?? '', detail: e.message })
                else toast({ kind: 'error', title: errorMessage(e) })
              },
            },
          )
        }
      >
        {t(mu.sendHere)}
      </Button>
      {user ? <span className="k">{t(mu.sendHereTo, { email: user.email })}</span> : null}
    </div>
  )
  const publishNow = async () => {
    setPublishing(true)
    try {
      const { data } = await http.GET(`${API}/templates/{template}`, { params: { path: { workspace: slug, template: tplKey } } })
      await http.POST(`${API}/templates/{template}/publish`, { params: { path: { workspace: slug, template: tplKey } }, body: { baseRevision: data!.draftRevision } })
      toast({ kind: 'success', title: t(mu.published, { key: tplKey }) })
      setRejected(undefined)
      void qc.invalidateQueries()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setPublishing(false)
    }
  }
  const mono = (s: string) => <span className="mono">{s}</span>
  const rows: Row[] = rejected
    ? [
        {
          title: t(mu.callReceived),
          detail: <T msg={mu.rejectedDetail} values={{ time: stampMs(rejected.at).slice(0, 8), answer: mono(`${rejected.status} ${rejected.code}`.trim()) }} />,
          state: 'bad',
        },
      ]
    : d
      ? timelineRows(t, d, provider)
      : [
          {
            title: t(mu.waitingCall),
            detail: key ? (
              <T msg={mu.waitingDetail} values={{ path: mono('POST …/transactional/messages'), key, time: waitedText }} />
            ) : (
              <T msg={mu.waitingDetailAny} values={{ path: mono('POST …/transactional/messages'), time: waitedText }} />
            ),
            state: 'wait',
          },
          { title: t(mu.rendered), detail: t(mu.afterCall), state: 'todo' },
          { title: t(mu.handed, { provider: provider || t(mu.yourProvider) }), state: 'todo' },
          { title: t(mu.delivered), state: 'todo' },
        ]
  const badge = rejected ? (
    <Badge tone="bad" dot>
      {t(mu.rejectedTitle)}
    </Badge>
  ) : delivered ? (
    <Badge tone="ok" dot>
      {t(mu.deliveredBadge)}
    </Badge>
  ) : d ? (
    <Badge tone="ok" dot>
      {t(mu.callReceived)}
    </Badge>
  ) : (
    <Badge tone="ok" dot pulse>
      {t(mu.listening)}
    </Badge>
  )
  return (
    <>
      <div className="flex flex-col gap-1.5">
        <h2 className="tx-ttl" style={{ fontSize: 24 }}>
          {t(mu.watchTitle)}
        </h2>
        <p className="tx-lead">{t(mu.watchBody, { provider: provider || t(mu.yourProvider) })}</p>
      </div>
      <div className="flex flex-wrap items-start gap-5">
        <section className="card flex min-w-0 flex-col gap-3" style={{ flex: '1 1 420px', padding: '18px 20px' }} aria-labelledby="tx-guide-call">
          <h3 className="h2" id="tx-guide-call">
            {t(mu.firstCall)}
          </h3>
          <span className="k" style={{ color: '#374151' }}>
            {t(mu.firstCallBody)}
          </span>
          <CodeBox lang={lang} onLang={setLang} code={code}>
            {highlight(code, lang)}
          </CodeBox>
          {d || rejected ? sendHere(false) : null}
        </section>
        <section
          className="card flex min-w-0 flex-col"
          style={{ flex: '1 1 420px', padding: '18px 20px', borderColor: rejected ? '#FCA5A5' : undefined }}
          aria-labelledby="tx-guide-timeline"
        >
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="h2" id="tx-guide-timeline">
              {t(mu.timelineTitle)}
            </h3>
            <span role="status">{badge}</span>
          </div>
          <div className="mt-2">
            {rows.map((r, i) => (
              <TimelineRow key={i} r={r} />
            ))}
          </div>
          {rejected ? (
            <Note tone="bad" role="alert" className="mt-2.5">
              <div className="flex min-w-0 flex-col gap-1">
                <span className="mono font-bold">{rejected.code || rejected.status}</span>
                {rejected.code === 'template_not_published' ? (
                  <div>
                    <T msg={mu.notPublished} values={{ key: mono(tplKey) }} />
                    <div className="mt-2 flex flex-wrap gap-2">
                      <Button size="sm" variant="primary" loading={publishing} onClick={() => void publishNow()}>
                        {t(mu.publishNow, { key: tplKey })}
                      </Button>
                      <ButtonLink size="sm" to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: tplKey }}>
                        {t(mu.openTemplate)}
                      </ButtonLink>
                    </div>
                  </div>
                ) : (
                  <div>{rejected.detail}</div>
                )}
              </div>
            </Note>
          ) : null}
          {delivered && d.deliveryMs !== undefined ? (
            <Note tone="ok" className="mt-2.5">
              <span>{t(mu.roundTrip, { time: secs(d.deliveryMs) })}</span>
            </Note>
          ) : null}
          {!d && !rejected ? <div className="mt-2.5">{sendHere(true)}</div> : null}
        </section>
      </div>
      <div className="flex flex-wrap justify-between gap-2.5">
        <Button onClick={onBack}>{t(mu.back)}</Button>
        <Button variant="primary" className="min-h-10 px-[18px]" disabledReason={delivered ? undefined : t(mu.finishWhen)} onClick={() => d && onDone(d.id)}>
          {t(mu.finish)}
        </Button>
      </div>
    </>
  )
}

function Done({ sender, setup, ov, tpl, picked }: { sender?: Sender; setup: Setup; ov: Overview; tpl: LiveTpl; picked: Picked }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const so = $api.useQuery('get', `${W}/senders`, { params: { path: { workspace: slug } } })
  const row = so.data?.domains.find((d) => d.domain === sender?.domain)
  const dom = $api.useQuery('get', `${W}/domains/{domainId}`, { params: { path: { workspace: slug, domainId: row?.id ?? '' } } }, { enabled: Boolean(row) })
  const log = $api.useQuery('get', `${API}/messages`, { params: { path: { workspace: slug }, query: { range: '30d', limit: 1, status: 'delivered' } } }, { enabled: !picked.message })
  const watched = $api.useQuery('get', `${API}/messages/{messageId}`, { params: { path: { workspace: slug, messageId: picked.message ?? '' } } }, { enabled: Boolean(picked.message) })
  const first = watched.data ?? log.data?.items[0]
  const key = setup.keys.find((k) => k.name === picked.key) ?? setup.keys[0]
  const counts = dom.data?.counts
  const rows: [string, ReactNode][] = [
    [t(mu.sender), sender ? `${sender.name} <${sender.email}>${counts?.total ? ` · ${t(mu.recordsVerified, { verified: counts.verified, total: counts.total })}` : ''}` : '—'],
    [
      t(mu.route),
      ov.route.primary
        ? ov.route.fallback
          ? t(mu.doneRoute, { primary: ov.route.primary.name, fallback: ov.route.fallback.name })
          : t(mu.doneRouteSingle, { primary: ov.route.primary.name })
        : '—',
    ],
    [t(mu.template), tpl ? t(mu.live, { key: tpl.key, version: tpl.version }) : '—'],
    [t(mu.key), key ? (key.preset === 'send_only' ? t(mu.doneKey, { name: key.name }) : key.name) : '—'],
    [t(mu.doneFirst), first && first.deliveryMs !== undefined ? t(mu.doneFirstValue, { email: first.email, time: secs(first.deliveryMs) }) : '—'],
  ]
  if (picked.monitored && me.hostRepute?.status === 'connected') rows.push([t(mu.hostRepute), t(mu.monitored, { domain: picked.monitored })])
  return (
    <section className="card flex flex-col gap-3.5" style={{ padding: '22px 24px' }} aria-labelledby="tx-guide-done">
      <div className="flex items-center gap-2.5">
        <span className="inline-flex size-9 items-center justify-center rounded-full" style={{ background: '#DCFCE7', color: '#166534' }}>
          <Icon name="check" size={18} />
        </span>
        <h2 className="tx-ttl" style={{ fontSize: 20 }} id="tx-guide-done">
          {t(mu.doneTitle)}
        </h2>
      </div>
      <dl className="kv">
        {rows.map(([k, v]) => (
          <div key={k} className="contents">
            <dt>{k}</dt>
            <dd>{v}</dd>
          </div>
        ))}
      </dl>
      <div className="flex flex-wrap gap-2">
        <ButtonLink variant="primary" to="/w/$workspace/transactional/messages" params={{ workspace: slug }}>
          {t(mu.openLog)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/transactional" params={{ workspace: slug }}>
          {t(mu.templates)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/transactional/setup" params={{ workspace: slug }}>
          {t(mu.apiReference)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/transactional/import" params={{ workspace: slug }}>
          {t(mu.importMore)}
        </ButtonLink>
      </div>
    </section>
  )
}
