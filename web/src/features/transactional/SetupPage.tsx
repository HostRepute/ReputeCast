import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { KeyActionDialog, type KeyAction } from '@/features/keys/KeyActionDialogs'
import { KeySecretDialog } from '@/features/keys/KeySecretDialog'
import { TableScroll, Th } from '@/components/ui/Table'
import { T, useT } from '@/i18n'
import { ago, dayMonthYear } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { codeSample, highlight } from './codeSamples'
import { CreateTransactionalKeyDialog } from './CreateKeyDialog'
import { me } from './errors.messages'
import { mt } from './messages'
import { useTemplateVariables } from './OverviewPage'
import { ms } from './setup.messages'
import { CodeBox, KeyBadges, RichList, TxTabs, type CodeLang } from './shared'

type Setup = Schemas['TransactionalSetup']
type SendingKey = Schemas['TransactionalSendingKey']

const API = '/api/v1/workspaces/{workspace}/transactional' as const

function mb(bytes: number) {
  return `${Math.round(bytes / (1024 * 1024))} MB`
}

/** Transactional › Setup (TransactionalSetup, TransactionalSetupStates): the API, keys, code samples, the SMTP relay and errors. */
export function TransactionalSetupPage() {
  const t = useT()
  const { slug, me: wme } = useWorkspace()
  const canView = wme.permissions.includes('transactional:view')
  const q = $api.useQuery('get', `${API}/setup`, { params: { path: { workspace: slug } } }, { enabled: canView })
  const [create, setCreate] = useState<'send_only' | 'smtp'>()
  return (
    <Page
      title={t(mt.title)}
      filters={<TxTabs />}
      actions={
        <ButtonLink to="/w/$workspace/transactional/guide" params={{ workspace: slug }}>
          {t(ms.guide)}
        </ButtonLink>
      }
    >
      {!canView ? (
        <ForbiddenState permission="transactional:view" title={t(mt.forbiddenTitle)} />
      ) : q.error ? (
        <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(mt.retry)}</Button>} />
      ) : !q.data ? (
        <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(460px, 100%), 1fr))' }} aria-busy="true">
          {Array.from({ length: 4 }, (_, i) => (
            <div key={i} className="card pad flex flex-col gap-3">
              <Skeleton w="40%" />
              <Skeleton w="100%" h={120} />
            </div>
          ))}
        </div>
      ) : (
        <>
          {q.data.senders.length === 0 ? <NoSender /> : null}
          <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(460px, 100%), 1fr))' }}>
            <ApiCard s={q.data} />
            <KeysCard s={q.data} onCreate={() => setCreate('send_only')} />
          </div>
          <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(460px, 100%), 1fr))' }}>
            <SamplesCard s={q.data} />
            <div className="flex min-w-0 flex-col gap-4">
              <RelayCard s={q.data} onCreate={() => setCreate('smtp')} />
              <EventsCard />
            </div>
          </div>
          <ErrorsCard s={q.data} />
          {create ? <CreateTransactionalKeyDialog preset={create} setup={q.data} onClose={() => setCreate(undefined)} /> : null}
        </>
      )}
    </Page>
  )
}

function NoSender() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="flex flex-col gap-3.5">
      <Note tone="bad">
        <div className="flex min-w-0 flex-col gap-1">
          <span className="font-bold">{t(ms.noSenderTitle)}</span>
          <span>
            <T msg={ms.noSenderBody} values={{ code: <span className="mono">{'sender_not_verified'}</span> }} />
          </span>
          <div className="mt-2">
            <ButtonLink size="sm" variant="primary" to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
              {t(ms.addSender)}
            </ButtonLink>
          </div>
        </div>
      </Note>
    </section>
  )
}

function ApiCard({ s }: { s: Setup }) {
  const t = useT()
  const { slug } = useWorkspace()
  const eps: [string, string, string][] = [
    ['POST', '/messages', t(ms.epSend)],
    ['POST', '/messages/batch', t(ms.epBatch, { count: s.limits.maxBatch })],
    ['POST', '/templates/{id}/render', t(ms.epRender)],
    ['GET', '/messages', t(ms.epLog)],
    ['GET', '/messages/{id}', t(ms.epMessage)],
  ]
  const mono = (x: string) => <span className="mono">{x}</span>
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(ms.apiTitle)}</h2>
          <span className="k">
            {t(ms.baseUrl)} <span className="mono">{s.baseUrl}</span>
          </span>
        </div>
      </div>
      <TableScroll label={t(ms.endpointsLabel)}>
        <table className="tbl" aria-label={t(ms.endpointsLabel)}>
          <thead>
            <tr>
              <Th>{t(ms.colMethod)}</Th>
              <Th>{t(ms.colPath)}</Th>
              <Th>{t(ms.colWhat)}</Th>
            </tr>
          </thead>
          <tbody>
            {eps.map(([m, p, w]) => (
              <tr key={m + p}>
                <td>
                  <Badge tone={m === 'POST' ? 'info' : 'neutral'}>{m}</Badge>
                </td>
                <td className="mono nw" style={{ fontSize: 12.5 }}>
                  {p}
                </td>
                <td>{w}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableScroll>
      <div className="pad flex flex-col gap-3.5" style={{ borderTop: '1px solid #EEF0F3' }}>
        <dl className="kv">
          <dt>{t(ms.auth)}</dt>
          <dd>
            <T msg={ms.authValue} values={{ header: mono('Authorization: Bearer <key>'), permission: mono('transactional:send') }} />
          </dd>
          <dt>{t(ms.idempotency)}</dt>
          <dd>
            <T msg={ms.idempotencyValue} values={{ header: mono('Idempotency-Key'), hours: s.limits.idempotencyHours }} />
          </dd>
          <dt>{t(ms.size)}</dt>
          <dd>
            {!s.limits.routeLimitProvider
              ? t(ms.sizeValueNoProvider, { size: mb(s.limits.routeLimitBytes) })
              : s.limits.routeLimitOthers.length > 0
                ? t(ms.sizeValueOthers, {
                    size: mb(s.limits.routeLimitBytes),
                    provider: s.limits.routeLimitProvider,
                    others: s.limits.routeLimitOthers.map((o) => t(ms.sizeAllows, { provider: o.provider, size: mb(o.bytes) })).join('; '),
                  })
                : t(ms.sizeValue, { size: mb(s.limits.routeLimitBytes), provider: s.limits.routeLimitProvider })}
          </dd>
          <dt>{t(ms.rate)}</dt>
          <dd>{t(ms.rateValue, { rate: s.limits.ratePerSecond })}</dd>
          <dt>{t(ms.scheduling)}</dt>
          <dd>
            <T msg={ms.schedulingValue} values={{ field: mono('send_at'), hours: s.limits.scheduleHours }} />
          </dd>
          <dt>{t(ms.language)}</dt>
          <dd>
            <T msg={ms.languageValue} values={{ field: mono('language') }} />
          </dd>
        </dl>
        <div className="flex flex-wrap gap-2.5">
          <a className="btn sm" href={s.openapiUrl} target="_blank" rel="noopener noreferrer">
            {t(ms.openapi)}
          </a>
          <ButtonLink size="sm" to="/w/$workspace/settings/webhooks" params={{ workspace: slug }}>
            {t(ms.webhooksLink)}
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}

function scopeText(t: ReturnType<typeof useT>, k: SendingKey) {
  if (k.templates.length) return t(ms.scopeTemplates, { names: k.templates.join(', ') })
  if (k.categories.length) return t(ms.scopeCategories, { names: k.categories.join(', ') })
  return t(ms.scopeAll)
}

function KeysCard({ s, onCreate }: { s: Setup; onCreate: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const keys = $api.useQuery('get', '/api/v1/workspaces/{workspace}/api-keys', { params: { path: { workspace: slug } } }, { enabled: s.canManageKeys })
  const byId = useMemo(() => new Map((keys.data?.items ?? []).map((k) => [k.id, k])), [keys.data])
  const [act, setAct] = useState<{ id: string; action: KeyAction }>()
  const [secret, setSecret] = useState<{ created: Schemas['APIKeyCreated']; action: 'rotated' | 'replaced' }>()
  const actKey = act ? byId.get(act.id) : undefined
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(ms.keysTitle, { count: s.keys.length })}</h2>
      </div>
      {s.keys.length === 0 ? <p className="k pad m-0">{t(ms.noKeys)}</p> : null}
      {s.keys.map((k) => (
        <div key={k.id} className="flex flex-col gap-1.5 border-t border-line-2" style={{ padding: '14px 18px' }}>
          <div className="row" style={{ justifyContent: 'space-between', gap: 8 }}>
            <span className="font-semibold">{k.name}</span>
            <span className="inline-flex gap-1.5">
              {k.expired ? <Badge tone="bad">{t(ms.expiredBadge)}</Badge> : null}
              <KeyBadges preset={k.preset} mode={k.mode} />
            </span>
          </div>
          <span className="k">
            <T
              msg={ms.keyMeta}
              values={{
                masked: <span className="mono">{k.masked}</span>,
                scope:
                  k.mode === 'test'
                    ? `${scopeText(t, k)} · ${k.testAddresses.length ? t(ms.testOnly, { addresses: k.testAddresses.join(', ') }) : t(ms.testOnlyMembers)}`
                    : scopeText(t, k),
                by: k.createdBy,
              }}
            />
          </span>
          <div className="row" style={{ justifyContent: 'space-between', gap: 8 }}>
            <span className="k" style={{ color: 'var(--ink-nav)' }}>
              {k.lastUsedAt ? (k.lastUsedIp ? t(ms.lastUsed, { when: ago(k.lastUsedAt, now), ip: k.lastUsedIp }) : t(ms.lastUsedNoIp, { when: ago(k.lastUsedAt, now) })) : t(ms.neverUsed)}
            </span>
            {s.canManageKeys ? (
              <span className="inline-flex gap-1.5">
                <Button size="sm" disabledReason={byId.has(k.id) ? undefined : t(mt.loading)} onClick={() => setAct({ id: k.id, action: 'rotate' })}>
                  {t(ms.rotate)}
                </Button>
                <Button size="sm" variant="danger" disabledReason={byId.has(k.id) ? undefined : t(mt.loading)} onClick={() => setAct({ id: k.id, action: 'revoke' })}>
                  {t(ms.revoke)}
                </Button>
              </span>
            ) : null}
          </div>
        </div>
      ))}
      <div className="tx-foot">
        <span className="k" style={{ flex: '1 1 260px' }}>
          <T msg={ms.keysNote} values={{ permission: <span className="mono">{'transactional:send'}</span> }} />
        </span>
        {s.canManageKeys ? (
          <Button size="sm" variant="primary" onClick={onCreate}>
            {t(ms.createKey)}
          </Button>
        ) : (
          <span className="k">
            <T
              msg={s.keyHelpers.length ? ms.keysForbidden : ms.keysForbiddenNoNames}
              values={{ permission: <span className="mono">{'apikeys:manage'}</span>, names: s.keyHelpers.join(', ') }}
            />
          </span>
        )}
      </div>
      {act && actKey ? (
        <KeyActionDialog
          k={actKey}
          action={act.action}
          onClose={() => setAct(undefined)}
          onSecret={(c, action) => {
            setAct(undefined)
            setSecret({ created: c, action })
          }}
        />
      ) : null}
      {secret ? <KeySecretDialog created={secret.created} action={secret.action} onClose={() => setSecret(undefined)} /> : null}
    </section>
  )
}

function SamplesCard({ s }: { s: Setup }) {
  const t = useT()
  const { slug } = useWorkspace()
  const list = $api.useQuery('get', `${API}/templates`, { params: { path: { workspace: slug }, query: { period: '7d' } } })
  const live = (list.data?.items ?? []).filter((it) => it.liveVersion && !it.archived)
  const [key, setKey] = useState<string>()
  const [lang, setLang] = useState<CodeLang>('python')
  const tpl = key ?? live.find((it) => it.key === 'password-reset')?.key ?? live[0]?.key
  const vars = useTemplateVariables(tpl)
  const endpoint = `${s.baseUrl}/messages`
  const code = tpl ? codeSample(lang, { endpoint, template: tpl, variables: vars }) : ''
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(ms.samplesTitle)}</h2>
          <span className="k">{t(ms.samplesCaption)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        {live.length ? (
          <label className="lbl" style={{ maxWidth: 320 }}>
            {t(ms.sampleTemplateLabel)}
            <select className="in" aria-label={t(ms.sampleTemplate)} value={tpl} onChange={(e) => setKey(e.target.value)}>
              {live.map((it) => (
                <option key={it.key} value={it.key}>
                  {it.key}
                </option>
              ))}
            </select>
          </label>
        ) : null}
        {tpl ? (
          <CodeBox lang={lang} onLang={setLang} code={code}>
            {highlight(code, lang)}
          </CodeBox>
        ) : (
          <p className="k m-0">{t(ms.noTemplates)}</p>
        )}
      </div>
    </section>
  )
}

function relayBadge(t: ReturnType<typeof useT>, r: Schemas['TransactionalRelay']) {
  switch (r.state) {
    case 'ok':
      return (
        <Badge tone="ok" dot>
          {t(ms.relayOn)}
        </Badge>
      )
    case 'unreachable':
      return (
        <Badge tone="bad" dot>
          {t(ms.relayUnreachable)}
        </Badge>
      )
    case 'certificate':
      return (
        <Badge tone="bad" dot>
          {t(ms.relayCertificate)}
        </Badge>
      )
    case 'unchecked':
      return <Badge tone="warn">{t(ms.relayUnchecked)}</Badge>
  }
  return <Badge tone="neutral">{t(ms.relayOff)}</Badge>
}

function RelayCard({ s, onCreate }: { s: Setup; onCreate: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const r = s.relay
  const mono = (x: string | number) => <span className="mono">{x}</span>
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(ms.relayTitle)}</h2>
        {relayBadge(t, r)}
      </div>
      <div className="pad flex flex-col gap-3">
        {!r.enabled ? (
          <>
            <p className="m-0">{t(ms.relayOffBody)}</p>
            <Note tone="neutral">
              {r.admins.length
                ? t(ms.relayOffAsk, { starttls: r.starttlsPort, tls: r.tlsPort, names: r.admins.join(', ') })
                : t(ms.relayOffAskNoNames, { starttls: r.starttlsPort, tls: r.tlsPort })}
            </Note>
            <div className="flex flex-wrap gap-2">
              <Button size="sm" disabledReason={t(ms.relayOffBody)}>
                {t(ms.createCredential)}
              </Button>
              {r.canManage ? (
                <ButtonLink size="sm" to="/w/$workspace/admin/instance" params={{ workspace: slug }} hash="smtp-relay">
                  {t(ms.instanceSettings)}
                </ButtonLink>
              ) : null}
            </div>
          </>
        ) : (
          <>
            <p className="k m-0" style={{ color: 'var(--ink-nav)' }}>
              {t(ms.relayIntro)}
            </p>
            {r.state === 'unreachable' ? <Note tone="bad">{t(ms.relayUnreachableBody)}</Note> : null}
            <dl className="kv">
              <dt>{t(ms.relayHost)}</dt>
              <dd className="mono">{r.hostname}</dd>
              <dt>{t(ms.relayPorts)}</dt>
              <dd>
                <T msg={ms.relayPortsValue} values={{ starttls: mono(r.starttlsPort), tls: mono(r.tlsPort) }} />
              </dd>
              <dt>{t(ms.relayUsername)}</dt>
              <dd>{t(ms.relayUsernameValue)}</dd>
              <dt>{t(ms.relayPassword)}</dt>
              <dd>{t(ms.relayPasswordValue)}</dd>
              {r.certNotAfter ? (
                <>
                  <dt>{t(ms.relayCertificateRow)}</dt>
                  <dd>{t(ms.relayCertificateValue, { date: dayMonthYear(r.certNotAfter) })}</dd>
                </>
              ) : null}
              <dt>{t(ms.relayFrom)}</dt>
              <dd className="flex flex-col">
                {[...s.senders]
                  .sort((a, b) => Number(b.email.startsWith('no-reply@')) - Number(a.email.startsWith('no-reply@')) || a.email.localeCompare(b.email))
                  .map((sn) => (
                    <span key={sn.id} className="mono">
                      {sn.email}
                    </span>
                  ))}
              </dd>
              <dt>{t(ms.relayHeaders)}</dt>
              <dd className="flex flex-col">
                {s.headers.map((h) => (
                  <span key={h} className="mono">
                    {h}
                  </span>
                ))}
              </dd>
            </dl>
            {r.state === 'unreachable' ? (
              <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
                {r.checks.map((c) => (
                  <li key={c.key} className="flex items-center justify-between gap-2 text-[13px]">
                    <span>{c.detail}</span>
                    <Badge tone={c.ok ? 'ok' : 'bad'}>{c.ok ? t(ms.checkPass) : t(ms.checkFail)}</Badge>
                  </li>
                ))}
              </ul>
            ) : null}
          </>
        )}
      </div>
      {r.enabled ? (
        <div className="tx-foot">
          <span className="k">
            {r.enabledAt
              ? r.enabledBy
                ? t(ms.relayTurnedOn, { name: r.enabledBy, date: dayMonthYear(r.enabledAt) })
                : t(ms.relayTurnedOnNoName, { date: dayMonthYear(r.enabledAt) })
              : null}
          </span>
          {s.canManageKeys ? (
            <Button size="sm" variant="primary" onClick={onCreate}>
              {t(ms.createCredential)}
            </Button>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

const messageEvents = ['message.sent', 'message.delivered', 'message.deferred', 'message.bounced', 'message.complained', 'message.rejected', 'message.failed', 'message.opened', 'message.clicked']

function EventsCard() {
  const t = useT()
  const { slug, me: wme } = useWorkspace()
  const canWebhooks = wme.permissions.includes('webhooks:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/webhooks', { params: { path: { workspace: slug } } }, { enabled: canWebhooks })
  const hook = (q.data?.items ?? []).find((w) => w.events.some((e) => e === 'message.*' || e.startsWith('message.')))
  // The endpoint gets the whole group, or some of its events.
  const some = hook && !hook.events.includes('message.*') ? hook.events.filter((e) => e.startsWith('message.')) : undefined
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(ms.eventsTitle)}</h2>
      </div>
      <div className="pad flex flex-col gap-2.5">
        {hook ? (
          <span className="k" style={{ color: 'var(--ink-nav)' }}>
            {some ? (
              <T msg={ms.eventsSome} values={{ url: <span className="mono">{hook.url}</span>, events: <RichList items={some} render={(e) => <span className="mono">{e}</span>} /> }} />
            ) : (
              <T msg={ms.eventsBody} values={{ url: <span className="mono">{hook.url}</span> }} />
            )}
          </span>
        ) : (
          <span className="k">{t(ms.eventsNone)}</span>
        )}
        <div className="flex flex-wrap gap-1.5">
          {messageEvents.map((e) => (
            <span key={e} className="tx-mk">
              {e}
            </span>
          ))}
        </div>
        {canWebhooks ? (
          <Link to="/w/$workspace/settings/webhooks" params={{ workspace: slug }} className="tx-lnk self-start">
            {t(ms.manageWebhooks)}
          </Link>
        ) : null}
      </div>
    </section>
  )
}

function ErrorsCard({ s }: { s: Setup }) {
  const t = useT()
  const mono = (x: string) => <span className="mono">{x}</span>
  const values = { data: mono('data'), from: mono('from'), header: mono('Retry-After'), size: mb(s.limits.routeLimitBytes), provider: s.limits.routeLimitProvider ?? '' }
  const msg = (code: string, part: 'when' | 'do') => (me as Record<string, (typeof me)[keyof typeof me]>)[`${code}_${part}`]
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(ms.errorsTitle)}</h2>
          <span className="k">
            <T msg={ms.errorsCaption} values={{ detail: mono('detail') }} />
          </span>
        </div>
      </div>
      <TableScroll label={t(ms.errorsLabel)}>
        <table className="tbl" aria-label={t(ms.errorsLabel)}>
          <thead>
            <tr>
              <Th>{t(ms.colCode)}</Th>
              <Th>{t(ms.colWhen)}</Th>
              <Th>{t(ms.colDo)}</Th>
            </tr>
          </thead>
          <tbody>
            {s.errors.map((e) => {
              const when = msg(e.code, 'when')
              const todo = msg(e.code, 'do')
              return (
                <tr key={e.code}>
                  <td className="mono nw" style={{ fontSize: 12.5 }}>
                    {e.code}
                  </td>
                  <td>{when ? <T msg={when} values={values} /> : e.status}</td>
                  <td>{todo ? <T msg={todo} values={values} /> : null}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </TableScroll>
    </section>
  )
}
