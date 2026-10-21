import { useNavigate } from '@tanstack/react-router'
import { useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { ConnectChannels } from '@/features/channels/ConnectChannels'
import { WizardLayout } from '@/features/connect/Wizard'
import { formatList, T, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clockSeconds } from '@/lib/format'
import { useOverflowX } from '@/lib/hooks'
import { deniedReason, useWorkspace } from '@/lib/session'
import { Capabilities, CopyRow, countryName, CountryName, registrationBadge, useKindName, useRefreshSms } from './common'
import { m } from './connect.messages'
import { AddSenderIdDialog } from './SenderDialogs'
import './sms.css'

export type SmsStep = 'connection' | 'senders' | 'test' | 'countries'
export interface SmsWizardSearch {
  setup?: string
  provider?: string
  step?: SmsStep
}

const STEPS = [
  { title: m.s1, sub: m.s1sub },
  { title: m.s2, sub: m.s2sub },
  { title: m.s3, sub: m.s3sub },
  { title: m.s4, sub: m.s4sub },
  { title: m.s5, sub: m.s5sub },
]

function StepBadge({ n }: { n: number }) {
  const t = useT()
  return <Badge tone="neutral">{t(m.stepOf, { n })}</Badge>
}

/** The five steps of the SMS path, the connect wizard's stepper markup. */
function Stepper({ current, lines }: { current: number; lines?: Record<number, string> }) {
  const t = useT()
  const ref = useRef<HTMLElement>(null)
  // On phones the steps are one sideways-scrolling strip: while it scrolls it
  // takes keyboard focus, as the email wizard's does (WCAG 2.1.1).
  const scrolls = useOverflowX(ref)
  return (
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <nav ref={ref} className="card cn-steps" aria-label={t(m.steps)} tabIndex={scrolls ? 0 : undefined}>
      {STEPS.map((d, i) => {
        const n = i + 1
        const done = n < current
        const cur = n === current
        const sub = lines?.[n] ?? (current === 1 || cur ? t(d.sub) : '')
        return (
          <div key={n} className={cn('step cn-step', cur && 'cur', done && 'done')} aria-current={cur ? 'step' : undefined}>
            <span className="num" aria-hidden="true">
              {done ? '✓' : n}
            </span>
            <div>
              <div className="font-semibold">
                {done ? <span className="vh">{t(m.done)}</span> : null}
                {t(d.title)}
              </div>
              {sub ? <div className="k">{sub}</div> : null}
            </div>
          </div>
        )
      })}
      {current === 1 ? <div className="cn-foot k mt-2.5 border-t border-line-2 pt-3">{t(m.stepperFoot)}</div> : null}
    </nav>
  )
}

function Body({ current, lines, children }: { current: number; lines?: Record<number, string>; children: ReactNode }) {
  return (
    <div className="cn-wiz">
      <Stepper current={current} lines={lines} />
      <main className="cn-main" id="main" tabIndex={-1}>
        {children}
      </main>
    </div>
  )
}

function Heading({ title, children }: { title: ReactNode; children?: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <h2 className="m-0 text-[24px] font-bold tracking-[-0.02em]">{title}</h2>
      {children ? <span className="text-ink-2">{children}</span> : null}
    </div>
  )
}

function Footer({ back, children }: { back?: ReactNode; children?: ReactNode }) {
  return (
    <div className="flex flex-wrap justify-between gap-2.5 border-t border-line-2 pt-3.5">
      {back ?? <span />}
      {children ? <div className="flex flex-wrap gap-2.5">{children}</div> : null}
    </div>
  )
}

/** Step 1: which provider sends your texts (SmsStates). */
export function SmsCatalogPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/catalog', { params: { path: { workspace: slug } } })
  const ov = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } })
  const items = q.data?.items ?? []
  const [picked, setPicked] = useState<string>()
  const choice = picked ?? items.find((i) => i.connected === 0)?.adapter ?? items[0]?.adapter
  const item = items.find((i) => i.adapter === choice)
  const canManage = ov.data?.canManage ?? true
  return (
    <WizardLayout title={t(m.title)} badge={<StepBadge n={1} />}>
      <Body current={1}>
        {!canManage ? (
          <Note tone="neutral">{t(m.readOnly, { helpers: ov.data?.helpers.length ? formatList(ov.data.helpers, 'disjunction') : 'none' })}</Note>
        ) : null}
        <Heading title={t(m.h1)}>{t(m.lead1)}</Heading>
        <div className="flex flex-wrap items-center gap-2.5">
          <ConnectChannels current="sms" />
          <span className="k">{t(m.adapters, { count: items.length })}</span>
        </div>
        {q.isError ? <ErrorPanel title={errorMessage(q.error)} /> : null}
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(300px, 100%), 1fr))' }} role="radiogroup" aria-label={t(m.h1)}>
          {items.map((i) => {
            const on = i.adapter === choice
            return (
              <button
                key={i.adapter}
                type="button"
                role="radio"
                aria-checked={on}
                onClick={() => setPicked(i.adapter)}
                className="card flex min-w-0 flex-col gap-2.5 px-4 py-3.5 text-left"
                style={on ? { borderColor: '#0F766E', boxShadow: '0 0 0 1px #0F766E' } : undefined}
              >
                <span className="flex flex-wrap items-center justify-between gap-2.5">
                  <span className="text-[15px] font-bold">{i.name}</span>
                  {on ? <Badge tone="ok">{t(m.selected)}</Badge> : i.connected ? <Badge tone="neutral">{t(m.connected)}</Badge> : null}
                </span>
                <span className="k text-[#4B5563]">{i.summary}</span>
                <Capabilities badges={i.badges} short />
              </button>
            )
          })}
          <div className="card flex min-w-0 flex-col gap-2.5 px-4 py-3.5" style={{ borderStyle: 'dashed', background: '#FBFBFC' }}>
            <span className="text-[15px] font-bold">{t(m.another)}</span>
            <span className="k text-[#4B5563]">{t(m.anotherText)}</span>
            <ButtonLink size="sm" className="self-start" to="/w/$workspace/providers/adapters" params={{ workspace: slug }}>
              {t(m.manageAdapters)}
            </ButtonLink>
          </div>
        </div>
        <Footer
          back={
            <ButtonLink to="/w/$workspace/providers/sms" params={{ workspace: slug }}>
              {t(m.cancel)}
            </ButtonLink>
          }
        >
          <Button
            variant="primary"
            disabled={!item}
            disabledReason={canManage ? undefined : deniedReason('providers:manage', ov.data?.helpers)}
            onClick={() => item && void navigate({ to: '/w/$workspace/providers/connect/sms/$adapter', params: { workspace: slug, adapter: item.adapter } })}
          >
            {item ? t(m.connectNamed, { name: item.name }) : t(m.continue)}
          </Button>
        </Footer>
      </Body>
    </WizardLayout>
  )
}

/** Steps 2 to 5: connection, senders, the test text, countries. */
export function SmsConnectWizard({ adapter, search, onSearch }: { adapter: string; search: SmsWizardSearch; onSearch: (s: SmsWizardSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/catalog', { params: { path: { workspace: slug } } })
  const item = q.data?.items.find((i) => i.adapter === adapter)
  const step: SmsStep = search.provider ? (search.step ?? 'senders') : 'connection'
  const n = { connection: 2, senders: 3, test: 4, countries: 5 }[step]
  return (
    <WizardLayout title={t(m.title)} badge={<StepBadge n={n} />}>
      <Body current={n} lines={item ? { 1: item.name } : undefined}>
        {q.isError ? <ErrorPanel title={errorMessage(q.error)} /> : null}
        {!item ? (
          <Loading label={t(m.title)}>
            <span className="k">{t(m.title)}</span>
          </Loading>
        ) : step === 'connection' || !search.provider ? (
          // Step 2, or its summary once connected.
          <Connection item={item} search={search} onSearch={onSearch} />
        ) : step === 'senders' ? (
          <Senders item={item} providerId={search.provider} search={search} onSearch={onSearch} />
        ) : step === 'test' ? (
          <TestText item={item} providerId={search.provider} search={search} onSearch={onSearch} />
        ) : (
          <Countries item={item} providerId={search.provider} search={search} onSearch={onSearch} />
        )}
      </Body>
    </WizardLayout>
  )
}

type Item = Schemas['SmsCatalogItem']
type StepProps = { item: Item; search: SmsWizardSearch; onSearch: (s: SmsWizardSearch, replace?: boolean) => void }

function Connection({ item, search, onSearch }: StepProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshSms()
  const [values, setValues] = useState<Record<string, string>>(() => Object.fromEntries(item.fields.filter((f) => f.default).map((f) => [f.key, f.default!])))
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [result, setResult] = useState<Schemas['SmsConnectResult']>()
  const ov = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } }, { enabled: Boolean(search.provider) })
  const connect = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers', {
    onSuccess: (r) => {
      void refresh()
      if (r.webhooks.mode === 'manual') {
        setResult(r)
        return
      }
      onSearch({ setup: r.setupId, provider: r.providerId, step: 'senders' })
    },
    onError: (e) => setErrors({ form: errorMessage(e) }),
  })
  const submit = () => {
    const next: Record<string, string> = {}
    for (const f of item.fields) if (f.required && !(values[f.key] ?? '').trim()) next[f.key] = t(m.required, { label: f.label })
    setErrors(next)
    if (Object.keys(next).length) return
    connect.mutate({ params: { path: { workspace: slug } }, body: { adapter: item.adapter, fields: values, setupId: search.setup } })
  }
  if (search.provider) {
    // Already connected (Back from the senders): continue, never connect twice.
    const card = ov.data?.providers.find((p) => p.id === search.provider)
    return (
      <>
        <Heading title={t(m.h2, { name: item.name })} />
        <Note tone="ok">{t(m.connectedAs, { account: card?.accountLabel ?? item.name })}</Note>
        <Footer>
          <Button variant="primary" onClick={() => onSearch({ ...search, step: 'senders' })}>
            {t(m.continue)}
          </Button>
        </Footer>
      </>
    )
  }
  if (result) {
    return (
      <>
        <Heading title={t(m.pasteTitle, { name: item.name })}>{t(m.pasteText)}</Heading>
        <Note tone="ok">{t(m.connectedAs, { account: result.account.name })}</Note>
        {result.webhooks.error ? <Note tone="warn">{t(m.registerFailed, { error: result.webhooks.error })}</Note> : null}
        <CopyRow label={t(m.webhookUrl)} value={result.webhooks.url} mono />
        {result.webhooks.secret ? <CopyRow label={t(m.webhookSecret)} value={result.webhooks.secret} mono /> : null}
        <Footer>
          <Button variant="primary" onClick={() => onSearch({ setup: result.setupId, provider: result.providerId, step: 'senders' })}>
            {t(m.continue)}
          </Button>
        </Footer>
      </>
    )
  }
  const secrets = item.fields.filter((f) => f.secret || f.options?.length)
  const plain = item.fields.filter((f) => !f.secret && !f.options?.length)
  const field = (f: Schemas['Field']) => (
    <Field key={f.key} label={f.label} hint={f.help} error={errors[f.key]}>
      {f.options?.length ? (
        <Select value={values[f.key] ?? f.default ?? f.options[0]} onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}>
          {f.options.map((o) => (
            <option key={o} value={o}>
              {o === 'live' ? t(m.optLive) : o === 'test' ? t(m.optTest) : o}
            </option>
          ))}
        </Select>
      ) : (
        <Input
          className={f.secret ? undefined : 'mono text-[12.5px]'}
          type={f.secret ? 'password' : 'text'}
          autoComplete="off"
          value={values[f.key] ?? ''}
          onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}
        />
      )}
    </Field>
  )
  return (
    <>
      <Heading title={t(m.h2, { name: item.name })}>{item.adapter === 'twilio' ? t(m.lead2Twilio) : t(m.lead2, { name: item.name })}</Heading>
      {errors.form ? <Note tone="bad">{errors.form}</Note> : null}
      <div className="flex flex-col gap-3">
        {plain.map(field)}
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(180px, 100%), 1fr))' }}>
          {secrets.map(field)}
        </div>
        <span className="k">{t(m.writeOnly)}</span>
      </div>
      <div className="sms-cbox">
        <span className="text-[13px] font-semibold">{item.webhooks === 'registered' ? t(m.webhooksApi, { name: item.name }) : t(m.webhooksManual, { name: item.name })}</span>
        <span className="k text-[#374151]">{item.webhooks === 'registered' ? t(m.webhooksApiText, { name: item.name }) : t(m.webhooksManualText, { name: item.name })}</span>
      </div>
      {item.webhooks === 'registered' ? <Note tone="neutral">{t(m.webhooksNote)}</Note> : null}
      <Footer back={<Button onClick={() => void navigate({ to: '/w/$workspace/providers/connect/sms', params: { workspace: slug } })}>{t(m.back)}</Button>}>
        <Button variant="primary" loading={connect.isPending} onClick={submit}>
          {t(m.check)}
        </Button>
      </Footer>
    </>
  )
}

function Senders({ item, providerId, search, onSearch }: StepProps & { providerId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const kindName = useKindName()
  const refresh = useRefreshSms()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/numbers', { params: { path: { workspace: slug, providerId } } })
  const ov = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } })
  const found = useMemo(() => q.data?.items ?? [], [q.data])
  const [picked, setPicked] = useState<string[] | null>(null)
  const chosen = picked ?? found.filter((f) => !f.imported && f.registration.status !== 'not_registered').map((f) => f.value)
  const [addingId, setAddingId] = useState(false)
  const imp = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/senders', {
    onSuccess: (d) => {
      void refresh()
      toast({ kind: 'success', title: t(m.importedToast, { count: d.items.length }) })
      onSearch({ ...search, step: 'test' })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const api = item.badges.some((b) => b.capability === 'registration' && b.support === 'yes') ? 'yes' : 'no'
  const fastest = found.filter((f) => chosen.includes(f.value) && f.throughput).sort((a, b) => b.throughput - a.throughput)[0]
  const pending = found.filter((f) => chosen.includes(f.value) && (f.registration.status === 'pending' || f.registration.status === 'not_registered') && f.kind !== 'local')
  const card = ov.data?.providers.find((p) => p.id === providerId)
  return (
    <>
      <Heading title={t(m.h3)}>{q.data ? t(m.lead3, { count: found.length, name: item.name, api }) : null}</Heading>
      {q.isError ? <Note tone="bad">{errorMessage(q.error)}</Note> : null}
      {q.isPending ? (
        <Loading label={t(m.h3)}>
          <span className="k">{t(m.h3)}</span>
        </Loading>
      ) : found.length ? (
        <div className="flex flex-col rounded-lg border border-line">
          {found.map((f, i) => (
            <label key={f.value} className="flex flex-wrap items-center gap-2.5 px-3 py-2.5" style={{ borderTop: i ? '1px solid var(--line-2)' : 0 }}>
              <input
                type="checkbox"
                checked={f.imported || chosen.includes(f.value)}
                disabled={f.imported}
                onChange={(e) => setPicked(e.target.checked ? [...chosen, f.value] : chosen.filter((v) => v !== f.value))}
              />
              <span className="mono text-[13px] font-semibold">{f.label}</span>
              <span className="k">
                {t(m.numberLine, {
                  kind: kindName(f.kind),
                  countries: f.countries.join(', '),
                  throughput: f.throughput ? `${f.throughput} /s` : 'none',
                })}
              </span>
              <span className="ml-auto flex flex-wrap gap-1.5">
                {f.imported ? (
                  <Badge tone="neutral">{t(m.imported)}</Badge>
                ) : (
                  <Badge tone={registrationBadge(f.registration, t).tone}>{registrationBadge(f.registration, t, true).text}</Badge>
                )}
              </span>
            </label>
          ))}
        </div>
      ) : null}
      <div className="sms-cbox">
        <span className="text-[13px] font-semibold">{t(m.orSenderId)}</span>
        <span className="k">{t(m.orSenderIdText)}</span>
        <div>
          <Button size="sm" onClick={() => setAddingId(true)}>
            {t(m.addSenderId)}
          </Button>
        </div>
      </div>
      <div className="sms-cbox">
        <span className="text-[13px] font-semibold">{t(m.fromProvider, { name: item.name })}</span>
        <span className="k text-[#374151]">
          {fastest ? t(m.fromProviderText, { rate: t(m.rate, { n: fastest.throughput }), kind: kindName(fastest.kind).toLowerCase(), name: item.name }) : t(m.fromProviderGeneric, { name: item.name })}
        </span>
      </div>
      {pending.map((p) => (
        <Note key={p.value} tone="neutral">
          {t(m.pendingNote, { number: p.label, kind: kindName(p.kind), name: item.name })}
        </Note>
      ))}
      <Footer back={<Button onClick={() => onSearch({ ...search, step: 'connection' })}>{t(m.back)}</Button>}>
        <Button
          variant="primary"
          loading={imp.isPending}
          onClick={() =>
            chosen.length
              ? imp.mutate({ params: { path: { workspace: slug, providerId } }, body: { senders: chosen.map((value) => ({ value })), setupId: search.setup } })
              : onSearch({ ...search, step: 'test' })
          }
        >
          {t(m.importN, { count: chosen.length })}
        </Button>
      </Footer>
      {addingId && card ? <AddSenderIdDialog providers={[card]} open onOpenChange={setAddingId} /> : null}
    </>
  )
}

function TestText({ item, providerId, search, onSearch }: StepProps & { providerId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const nums = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/test-numbers', { params: { path: { workspace: slug } } })
  const list = nums.data?.items ?? []
  const [to, setTo] = useState<string>()
  const target = to ?? list[0]?.id ?? ''
  const [messageId, setMessageId] = useState<string>()
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/test', {
    onSuccess: (msg) => setMessageId(msg.id),
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const msg = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/sms/messages/{id}',
    { params: { path: { workspace: slug, id: messageId ?? '' } } },
    {
      enabled: Boolean(messageId),
      refetchInterval: (query) => {
        const s = query.state.data?.status
        return s === 'delivered' || s === 'failed' || s === 'undelivered' ? false : 1000
      },
    },
  )
  const v = msg.data
  const who = list.find((n) => n.id === target)
  return (
    <>
      <Heading title={t(m.h4)}>{t(m.lead4)}</Heading>
      {list.length ? (
        <div className="flex flex-nowrap gap-2">
          <Select aria-label={t(m.sendTo)} className="flex-1" value={target} onChange={(e) => setTo(e.target.value)}>
            {list.map((n) => (
              <option key={n.id} value={n.id}>
                {n.name} · {n.display}
              </option>
            ))}
          </Select>
          <Button loading={send.isPending} onClick={() => send.mutate({ params: { path: { workspace: slug, providerId } }, body: { testNumberId: target, setupId: search.setup } })}>
            {messageId ? t(m.sendAgain) : t(m.send)}
          </Button>
        </div>
      ) : (
        <Note tone="neutral">{t(m.noTestNumbers)}</Note>
      )}
      {v ? (
        <div>
          <Ev tone="#DCFCE7" title={t(m.sentFrom, { sender: v.sender })} text={t(m.sentBody, { body: v.body, segments: v.segments })} at={v.sentAt ?? v.createdAt} />
          {v.providerMessageId ? (
            <Ev
              tone="#DCFCE7"
              title={t(m.accepted, { name: item.name })}
              text={<T msg={m.acceptedRef} values={{ ref: shortRef(v.providerMessageId), status: v.events[0]?.kind ?? 'queued', mono: (c) => <span className="mono">{c}</span> }} />}
              at={v.sentAt ?? v.createdAt}
            />
          ) : null}
          {v.status === 'delivered' ? (
            <Ev
              tone="#DCFCE7"
              title={t(m.delivered, { who: who?.name ?? v.recipient })}
              text={t(m.deliveredText, { adapter: item.adapter, seconds: (v.seconds ?? 0).toFixed(1) })}
              at={v.deliveredAt ?? v.createdAt}
            />
          ) : v.status === 'failed' || v.status === 'undelivered' ? (
            <Ev tone="#FEE2E2" title={t(m.failed)} text={t(m.failedText, { reason: v.reason ?? '' })} at={v.createdAt} />
          ) : (
            <Ev tone="#F3F4F6" title={t(m.waiting)} text="" at={v.createdAt} />
          )}
        </div>
      ) : null}
      <Footer back={<Button onClick={() => onSearch({ ...search, step: 'senders' })}>{t(m.back)}</Button>}>
        <Button variant="primary" onClick={() => onSearch({ ...search, step: 'countries' })}>
          {v ? t(m.continue) : t(m.skip)}
        </Button>
      </Footer>
    </>
  )
}

function shortRef(id: string): string {
  return id.length <= 12 ? id : `${id.slice(0, 6)}…${id.slice(-4)}`
}

function Ev({ tone, title, text, at }: { tone: string; title: string; text: ReactNode; at: string }) {
  return (
    <div className="sms-ev">
      <span className="ic" style={{ background: tone }} aria-hidden="true">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#0B1220" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <path d="m3.5 8.5 3 3 6-7" />
        </svg>
      </span>
      <div className="tx">
        <span className="font-semibold">{title}</span>
        {text ? <span className="k">{text}</span> : null}
      </div>
      <span className="tm">{clockSeconds(at)}</span>
    </div>
  )
}

function Countries({ item, providerId, search, onSearch }: StepProps & { providerId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const kindName = useKindName()
  const refresh = useRefreshSms()
  const ov = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/overview', { params: { path: { workspace: slug } } })
  const rules = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/rules', { params: { path: { workspace: slug } } })
  const senders = useMemo(() => (ov.data?.senders ?? []).filter((s) => s.providerId === providerId), [ov.data, providerId])
  const countries = useMemo(() => [...new Set(senders.flatMap((s) => s.countries))], [senders])
  const [extra, setExtra] = useState<string[]>([])
  const [picked, setPicked] = useState<Record<string, string>>({})
  const [other, setOther] = useState('')
  const [adding, setAdding] = useState('')
  const [error, setError] = useState<string>()
  // Each country starts with the first sender that can send there.
  const pick = useMemo(() => {
    const out: Record<string, string> = {}
    for (const c of countries) out[c] = senders.find((s) => s.countries.includes(c) && s.canSend)?.id ?? senders.find((s) => s.countries.includes(c))?.id ?? ''
    return { ...out, ...picked }
  }, [countries, senders, picked])
  const setPick = (next: Record<string, string>) => setPicked(next)
  const routesSave = $api.useMutation('put', '/api/v1/workspaces/{workspace}/sms/routes')
  const finish = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/setups/{id}/finish')
  const all = [...countries, ...extra]
  const label = (s: Schemas['SmsSenderRow']) =>
    s.canSend ? t(m.senderOption, { label: s.label || s.value, kind: kindName(s.kind).toLowerCase() }) : t(m.senderOptionPending, { label: s.label || s.value, kind: kindName(s.kind) })
  const done = async () => {
    setError(undefined)
    const existing = rules.data?.routes ?? []
    const mine = new Set(all.filter((c) => pick[c]))
    const kept = existing
      .map((r) => ({ countries: r.countries.filter((c) => !mine.has(c)), senderId: r.senderId, fallbackId: r.fallbackId || undefined, id: r.id }))
      .filter((r) => r.countries.length)
    const bySender = new Map<string, string[]>()
    for (const c of all) {
      const s = pick[c]
      if (!s) continue
      bySender.set(s, [...(bySender.get(s) ?? []), c])
    }
    const routes = [...kept.map(({ id: _id, ...r }) => r), ...[...bySender.entries()].map(([senderId, cs]) => ({ countries: cs, senderId }))]
    let otherRoute: number | undefined
    if (other) {
      const idx = routes.findIndex((r) => r.senderId === other)
      otherRoute = idx >= 0 ? idx : undefined
    } else if (rules.data?.other === 'route' && rules.data.otherRouteId) {
      const prev = existing.find((r) => r.id === rules.data!.otherRouteId)
      const idx = prev ? routes.findIndex((r) => r.senderId === prev.senderId) : -1
      otherRoute = idx >= 0 ? idx : undefined
    }
    try {
      await routesSave.mutateAsync({
        params: { path: { workspace: slug } },
        body: { routes, other: otherRoute !== undefined ? 'route' : 'dont_send', otherRoute },
      })
      if (search.setup) await finish.mutateAsync({ params: { path: { workspace: slug, id: search.setup } } })
      void refresh()
      toast({ kind: 'success', title: t(m.finished, { name: item.name }) })
      void navigate({ to: '/w/$workspace/providers/sms', params: { workspace: slug } })
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <>
      <Heading title={t(m.h5, { name: item.name })}>{t(m.lead5)}</Heading>
      {error ? <Note tone="bad">{error}</Note> : null}
      <div className="flex flex-col rounded-lg border border-line">
        {all.map((c, i) => (
          <div key={c} className="flex flex-wrap items-center gap-2.5 px-3 py-2.5" style={{ borderTop: i ? '1px solid var(--line-2)' : 0 }}>
            <span className="flex-[1_1_160px]">
              <CountryName code={c} />
            </span>
            <Select aria-label={t(m.senderFor, { country: countryName(c) })} style={{ width: 240 }} value={pick[c] ?? ''} onChange={(e) => setPick({ ...pick, [c]: e.target.value })}>
              <option value="">{t(m.dontSend)}</option>
              {senders
                .filter((s) => s.countries.includes(c) || s.kind === 'sender_id')
                .map((s) => (
                  <option key={s.id} value={s.id}>
                    {label(s)}
                  </option>
                ))}
            </Select>
          </div>
        ))}
        <div className="flex flex-wrap items-center gap-2.5 px-3 py-2.5" style={{ borderTop: all.length ? '1px solid var(--line-2)' : 0 }}>
          <span className="flex-[1_1_160px] font-semibold">{t(m.everyOther)}</span>
          <Select aria-label={t(m.everyOther)} style={{ width: 240 }} value={other} onChange={(e) => setOther(e.target.value)}>
            <option value="">{t(m.dontSend)}</option>
            {senders
              .filter((s) => Object.values(pick).includes(s.id))
              .map((s) => (
                <option key={s.id} value={s.id}>
                  {label(s)}
                </option>
              ))}
          </Select>
        </div>
      </div>
      <div className="flex flex-wrap items-end gap-2">
        <Field label={t(m.countryCode)}>
          <Input value={adding} maxLength={2} className="mono w-[90px]" onChange={(e) => setAdding(e.target.value.toUpperCase())} />
        </Field>
        <Button
          size="sm"
          onClick={() => {
            const c = adding.trim().toUpperCase()
            if (!/^[A-Z]{2}$/.test(c)) {
              setError(t(m.countryInvalid))
              return
            }
            if (!all.includes(c)) setExtra([...extra, c])
            setAdding('')
            setError(undefined)
          }}
        >
          {t(m.addCountry)}
        </Button>
      </div>
      <Note tone="neutral">{t(m.note5)}</Note>
      <Footer back={<Button onClick={() => onSearch({ ...search, step: 'test' })}>{t(m.back)}</Button>}>
        <Button variant="primary" loading={routesSave.isPending || finish.isPending} onClick={() => void done()}>
          {t(m.finish)}
        </Button>
      </Footer>
    </>
  )
}
