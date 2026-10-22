import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { TableScroll, Th } from '@/components/ui/Table'
import { T, useT } from '@/i18n'
import { ago, andList, clock, count, dayMonthDate, trimPct } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useMe, useWorkspace } from '@/lib/session'
import { codeSample, highlight } from './codeSamples'
import { mo, mt } from './messages'
import {
  bouncePct,
  CategoryCell,
  CodeBox,
  complaintPct,
  deliveredPct,
  KeyBadges,
  lastSent,
  MiniSpark,
  secs,
  TxTabs,
  VersionBadges,
  type CodeLang,
  type TxPeriod,
} from './shared'

type Overview = Schemas['TransactionalOverview']
type Item = Schemas['TransactionalTemplateListItem']
type Alert = Schemas['TransactionalAlert']

const API = '/api/v1/workspaces/{workspace}/transactional' as const

/** Transactional › Templates (Transactional, TransactionalStates). */
export function TransactionalOverviewPage({ period, onPeriod }: { period: TxPeriod; onPeriod: (p: TxPeriod) => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const canView = me.permissions.includes('transactional:view')
  const canEdit = me.permissions.includes('transactional:edit')
  const path = { workspace: slug }
  const ov = $api.useQuery('get', `${API}/overview`, { params: { path, query: { period } } }, { enabled: canView, refetchInterval: 60_000 })
  const list = $api.useQuery('get', `${API}/templates`, { params: { path, query: { period } } }, { enabled: canView, refetchInterval: 60_000 })
  const firstRun = ov.data && ov.data.firstRun.templates === 0 && ov.data.firstRun.keys === 0
  return (
    <Page
      title={t(mt.title)}
      filters={<TxTabs />}
      actions={
        canEdit ? (
          <>
            <ButtonLink to="/w/$workspace/transactional/import" params={{ workspace: slug }}>
              {t(mt.importTemplates)}
            </ButtonLink>
            <ButtonLink variant="primary" to="/w/$workspace/transactional/templates/new" params={{ workspace: slug }}>
              {t(mt.newTemplate)}
            </ButtonLink>
          </>
        ) : null
      }
    >
      {!canView ? (
        <ForbiddenState permission="transactional:view" title={t(mt.forbiddenTitle)} />
      ) : (
        <>
          {list.data?.readOnlyReason ? <ReadOnlyNote reason={list.data.readOnlyReason} /> : null}
          {ov.data?.failover ? <FailoverCard o={ov.data} /> : null}
          {ov.data?.alerts.map((a) => (
            <AlertNote key={a.id} a={a} hour={ov.data!.hour} />
          ))}
          {ov.data && !ov.data.route.configured ? <NoRouteNote o={ov.data} /> : null}
          {ov.data && !firstRun ? <RouteRow o={ov.data} period={period} onPeriod={onPeriod} /> : null}
          {firstRun ? (
            <FirstRun />
          ) : (
            <>
              <Stats q={ov} period={period} />
              <TemplatesCard q={list} period={period} />
              {ov.data ? (
                <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(460px, 100%), 1fr))' }}>
                  <SendCard o={ov.data} items={list.data?.items ?? []} />
                  <div className="flex min-w-0 flex-col gap-4">
                    <KeysCard o={ov.data} />
                    <IncidentsCard o={ov.data} />
                  </div>
                </div>
              ) : null}
            </>
          )}
        </>
      )}
    </Page>
  )
}

function ReadOnlyNote({ reason }: { reason: string }) {
  const t = useT()
  const { me } = useWorkspace()
  const user = useMe().data?.user
  return (
    <Note tone="neutral" role="note">
      <div className="flex min-w-0 flex-col gap-1">
        <span className="font-bold">{t(mo.readOnlyTitle, { name: user?.name ?? '', role: me.workspace.roleName })}</span>
        <span>{reason}</span>
      </div>
    </Note>
  )
}

function policyLabel(t: ReturnType<typeof useT>, p?: string) {
  if (p === 'pause_and_ask') return t(mo.pauseAndAsk)
  if (p === 'warm_only') return t(mo.warmOnly)
  return t(mo.automaticFailover)
}

function RouteRow({ o, period, onPeriod }: { o: Overview; period: TxPeriod; onPeriod: (p: TxPeriod) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const r = o.route
  return (
    <div className="row" style={{ justifyContent: 'space-between' }}>
      {r.configured && r.primary ? (
        <div className="tx-flowr" aria-label={t(mo.routeLabel)}>
          <span className="k">{t(mo.route)}</span>
          <span className="tx-pill">
            <span className="dot" style={{ color: o.failover ? '#B91C1C' : '#15803D' }} />
            {r.primary.name}
          </span>
          {r.fallback ? (
            <>
              <span className="arw" aria-hidden="true">
                →
              </span>
              <span className="tx-pill" style={{ background: '#fff' }}>
                {r.fallback.name} <span className="k" style={{ fontWeight: 500 }}>{t(mo.fallback)}</span>
              </span>
            </>
          ) : null}
          <Badge tone="neutral">{policyLabel(t, r.policy)}</Badge>
          {r.priority ? <Badge tone="neutral">{t(mo.priority)}</Badge> : null}
          {o.can.editRoute ? (
            <Link to="/w/$workspace/providers/routing" params={{ workspace: slug }} className="tx-lnk">
              {t(mo.editRoute)}
            </Link>
          ) : null}
        </div>
      ) : (
        <span />
      )}
      <Segmented<TxPeriod>
        label={t(mo.period)}
        value={period}
        onChange={onPeriod}
        items={[
          { value: '24h', label: t(mo.period24h) },
          { value: '7d', label: t(mo.period7d) },
          { value: '30d', label: t(mo.period30d) },
        ]}
      />
    </div>
  )
}

function periodWords(t: ReturnType<typeof useT>, p: TxPeriod) {
  return p === '24h' ? t(mo.period24h) : p === '30d' ? t(mo.period30d) : t(mo.period7d)
}

function Stat({ label, value, sub, hot }: { label: string; value: string; sub: string; hot?: boolean }) {
  return (
    <div className={`card tx-stat${hot ? ' hot' : ''}`}>
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      <span className="s">{sub}</span>
    </div>
  )
}

function Stats({ q, period }: { q: { data?: Overview; isPending: boolean; error: unknown }; period: TxPeriod }) {
  const t = useT()
  if (!q.data) {
    return (
      <div className="tx-stats" aria-busy="true">
        {Array.from({ length: 6 }, (_, i) => (
          <div key={i} className="card tx-stat">
            <Skeleton w="60%" />
            <Skeleton w="40%" h={22} />
            <Skeleton w="70%" />
          </div>
        ))}
      </div>
    )
  }
  const s = q.data.stats
  return (
    <div className="tx-stats">
      <Stat
        label={t(mo.sentLabel, { period: periodWords(t, period) })}
        value={count(s.sent)}
        sub={s.smtp ? t(mo.sentFromKeysAndRelay, { keys: s.keys }) : t(mo.sentFromKeys, { keys: s.keys })}
      />
      <Stat label={t(mo.delivered)} value={s.sent ? deliveredPct(s.deliveryRate) : '—'} sub={t(mo.deliveredMessages, { count: s.delivered })} />
      <Stat label={t(mo.bounced)} value={s.sent ? bouncePct(s.bounceRate, true) : '—'} sub={t(mo.bouncedSub, { count: count(s.bounced), deferred: s.deferred })} />
      <Stat
        label={t(mo.complaints)}
        value={count(s.complained)}
        sub={s.complained ? t(mo.complaintsOn, { rate: complaintPct(s.complaintRate), templates: s.complainedOn.join(', ') }) : t(mo.complaintsNone)}
      />
      <Stat label={t(mo.median)} value={s.medianMs !== undefined && s.medianMs >= 0 ? secs(s.medianMs) : t(mo.noMedian)} sub={t(mo.medianSub)} />
      <Stat label={t(mo.rejected)} value={count(s.rejected)} sub={t(mo.rejectedSub, { suppressed: count(s.suppressed), optedOut: count(s.optedOut) })} />
    </div>
  )
}

type Filter = { kind: 'all' } | { kind: 'category'; id: string } | { kind: 'draft' }

function TemplatesCard({ q, period }: { q: { data?: Schemas['TransactionalTemplateList']; isPending: boolean; error: unknown; refetch: () => unknown }; period: TxPeriod }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState<Filter>({ kind: 'all' })
  const items = useMemo(() => q.data?.items ?? [], [q.data])
  const categories = useMemo(() => {
    const out: { id: string; name: string; n: number }[] = []
    for (const it of items) {
      if (!it.category) continue
      const c = out.find((x) => x.id === it.category!.id)
      if (c) c.n++
      else out.push({ id: it.category.id, name: it.category.name, n: 1 })
    }
    return out
  }, [items])
  const drafts = items.filter((it) => it.liveVersion && it.draftChanges).length
  const ql = query.trim().toLowerCase()
  const shown = items.filter((it) => {
    if (filter.kind === 'category' && it.category?.id !== filter.id) return false
    if (filter.kind === 'draft' && !(it.liveVersion && it.draftChanges)) return false
    return !ql || it.name.toLowerCase().includes(ql) || it.key.includes(ql)
  })
  const canEdit = Boolean(q.data?.canEdit)
  const chip = (on: boolean, label: string, n: number, onClick: () => void) => (
    <button type="button" className={`tx-fchip${on ? ' on' : ''}`} aria-pressed={on} onClick={onClick}>
      {label} <b>{n}</b>
    </button>
  )
  const action = (it: Item) => {
    if (canEdit && it.liveVersion && it.draftChanges && q.data?.canPublish) {
      return (
        <ButtonLink size="sm" to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: it.key }} search={{ publish: '1' } as never}>
          {t(mo.reviewDraft)}
        </ButtonLink>
      )
    }
    return (
      <ButtonLink size="sm" to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: it.key }}>
        {canEdit ? t(mo.edit) : t(mo.view)}
      </ButtonLink>
    )
  }
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mo.templatesTitle, { count: items.length })}</h2>
          <span className="k">{t(mo.templatesCaption)}</span>
        </div>
        <label className="tx-search">
          <Icon name="search" size={16} />
          <input aria-label={t(mo.search)} placeholder={t(mo.searchPlaceholder)} value={query} onChange={(e) => setQuery(e.target.value)} />
        </label>
      </div>
      {q.error ? (
        <div className="pad">
          <ErrorPanel title={t(mo.errorTitle)} actions={<Button onClick={() => void q.refetch()}>{t(mt.retry)}</Button>}>
            {t(mo.errorBody, { reason: errorMessage(q.error) })}
          </ErrorPanel>
        </div>
      ) : !q.data ? (
        <div className="pad flex flex-col gap-3" aria-busy="true" aria-label={t(mo.loadingTemplates)}>
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} w="100%" h={18} />
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="pad">
          <div className="card tx-empty">
            <span className="tt">{t(mo.noTemplatesTitle)}</span>
            <span className="ds">{t(mo.noTemplatesBody)}</span>
            {canEdit ? (
              <ButtonLink variant="primary" to="/w/$workspace/transactional/templates/new" params={{ workspace: slug }}>
                {t(mt.newTemplate)}
              </ButtonLink>
            ) : null}
          </div>
        </div>
      ) : (
        <>
          <div className="tx-fchips" role="group" aria-label={t(mo.filterLabel)}>
            {chip(filter.kind === 'all', t(mo.filterAll), items.length, () => setFilter({ kind: 'all' }))}
            {categories.map((c) => chip(filter.kind === 'category' && filter.id === c.id, c.name, c.n, () => setFilter({ kind: 'category', id: c.id })))}
            {drafts ? chip(filter.kind === 'draft', t(mo.filterDraft), drafts, () => setFilter({ kind: 'draft' })) : null}
          </div>
          {shown.length === 0 ? (
            <div className="pad">
              <div className="card tx-empty">
                <span className="tt">{t(mo.noMatchTitle, { query: query.trim() })}</span>
                <span className="ds">{t(mo.noMatchBody, { count: items.length, query: query.trim() })}</span>
                <div className="row" style={{ justifyContent: 'center' }}>
                  <Button onClick={() => setQuery('')}>{t(mo.clearSearch)}</Button>
                  {canEdit && query.trim() ? (
                    <ButtonLink to="/w/$workspace/transactional/templates/new" params={{ workspace: slug }} search={{ name: query.trim() } as never}>
                      {t(mo.newNamed, { name: query.trim() })}
                    </ButtonLink>
                  ) : null}
                </div>
              </div>
            </div>
          ) : (
            <>
              <div className="tx-no-phone">
                <TableScroll label={t(mo.tableLabel)}>
                  <table className="tbl" aria-label={t(mo.tableLabel)}>
                    <thead>
                      <tr>
                        <Th>{t(mo.colTemplate)}</Th>
                        <Th>{t(mo.colCategory)}</Th>
                        <Th>{t(mo.colVersion)}</Th>
                        <Th align="right">{t(mo.colSent, { period: periodWords(t, period) })}</Th>
                        <Th align="right">{t(mo.colDelivered)}</Th>
                        <Th align="right">{t(mo.colBounced)}</Th>
                        <Th align="right">{t(mo.colMedian)}</Th>
                        <Th>{t(mo.colLastSent)}</Th>
                        <Th>
                          <span className="vh">{t(mo.edit)}</span>
                        </Th>
                      </tr>
                    </thead>
                    <tbody>
                      {shown.map((it) => {
                        const st = it.stats
                        const warnBounce = st.bounceRate >= 1
                        return (
                          <tr key={it.id}>
                            <td>
                              <div className="flex min-w-0 flex-col gap-0.5">
                                <Link
                                  to="/w/$workspace/transactional/templates/$templateId"
                                  params={{ workspace: slug, templateId: it.key }}
                                  style={{ color: '#0B1220', fontWeight: 600 }}
                                >
                                  {it.name}
                                </Link>
                                <span className="mono k">{it.attachments ? t(mo.pdfAttached, { key: it.key }) : it.key}</span>
                                {it.usedByProducts ? <span className="k">{t(mo.usedByProducts, { count: it.usedByProducts })}</span> : null}
                              </div>
                            </td>
                            <td>
                              <CategoryCell category={it.category} />
                            </td>
                            <td>
                              <VersionBadges live={it.liveVersion} draftChanges={it.draftChanges} draftVersion={it.draftVersion} archived={it.archived} />
                            </td>
                            <td className="r">
                              <div className="flex items-center justify-end gap-2.5">
                                <span className="tx-num font-semibold">{count(st.sent)}</span>
                                <MiniSpark values={it.spark} />
                              </div>
                            </td>
                            <td className="r tx-num">{st.sent ? deliveredPct(st.deliveryRate) : '—'}</td>
                            <td className="r tx-num">
                              {st.sent ? <span style={warnBounce ? { color: '#92400E', fontWeight: 600 } : undefined}>{bouncePct(st.bounceRate)}</span> : '—'}
                              {warnBounce && st.bounced ? <div className="k">{t(mo.bouncedOf, { bounced: count(st.bounced), sent: count(st.sent) })}</div> : null}
                              {st.complained ? <div className="k">{t(mo.complaintOne, { count: st.complained })}</div> : null}
                            </td>
                            <td className="r tx-num">{st.medianMs !== undefined && st.medianMs >= 0 ? secs(st.medianMs) : '—'}</td>
                            <td className="nw k">{lastSent(it.lastSentAt, now)}</td>
                            <td className="r">{action(it)}</td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </TableScroll>
              </div>
              <div className="tx-only-phone">
                {shown.map((it) => (
                  <Link
                    key={it.id}
                    to="/w/$workspace/transactional/templates/$templateId"
                    params={{ workspace: slug, templateId: it.key }}
                    className="flex flex-col gap-1.5 border-t border-line-2 px-4 py-3"
                    style={{ color: '#0B1220' }}
                  >
                    <span className="flex items-center justify-between gap-2">
                      <b className="font-semibold">{it.name}</b>
                      <VersionBadges live={it.liveVersion} draftChanges={it.draftChanges} draftVersion={it.draftVersion} archived={it.archived} />
                    </span>
                    <span className="mono k">{it.key}</span>
                    {it.usedByProducts ? <span className="k">{t(mo.usedByProducts, { count: it.usedByProducts })}</span> : null}
                    <span className="flex flex-wrap items-center gap-2">
                      <CategoryCell category={it.category} />
                      <span className="k">
                        {t(mo.phoneLine, {
                          sent: count(it.stats.sent),
                          delivered: it.stats.sent ? deliveredPct(it.stats.deliveryRate) : '—',
                          median: it.stats.medianMs !== undefined && it.stats.medianMs >= 0 ? secs(it.stats.medianMs) : '—',
                        })}
                      </span>
                    </span>
                  </Link>
                ))}
              </div>
            </>
          )}
          <div className="tx-foot">
            <span className="k">{t(mo.foot)}</span>
            {canEdit ? (
              <Link to="/w/$workspace/transactional/import" params={{ workspace: slug }} className="tx-lnk">
                <Icon name="import" size={14} />
                {t(mo.importLink)}
              </Link>
            ) : null}
          </div>
        </>
      )}
    </section>
  )
}

function SendCard({ o, items }: { o: Overview; items: Item[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [lang, setLang] = useState<CodeLang>('node')
  const tpl = items.find((it) => it.key === 'password-reset' && it.liveVersion) ?? items.find((it) => it.liveVersion) ?? items[0]
  const vars = useTemplateVariables(tpl?.key)
  const code = codeSample(lang, { endpoint: o.endpoint, template: tpl?.key ?? 'password-reset', variables: vars, compact: true })
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mo.sendTitle)}</h2>
          <span className="k">{t(mo.sendCaption)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="info">{'POST'}</Badge>
          <span className="mono" style={{ fontSize: 12.5, overflowWrap: 'anywhere' }}>
            {o.endpoint}
          </span>
        </div>
        <CodeBox lang={lang} onLang={setLang} code={code}>
          {highlight(code, lang)}
        </CodeBox>
        <div className="flex flex-wrap items-center gap-2.5">
          <ButtonLink size="sm" to="/w/$workspace/transactional/setup" params={{ workspace: slug }}>
            {t(mo.apiAndRelay)}
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/transactional/guide" params={{ workspace: slug }}>
            {t(mo.guide)}
          </ButtonLink>
          <span className="k" style={{ marginLeft: 'auto' }}>
            {t(mo.idempotency)}
          </span>
        </div>
      </div>
    </section>
  )
}

/** A template's declared variables, for the code samples: its live version's, which the API sends. */
export function useTemplateVariables(key?: string) {
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    `${API}/templates/{template}`,
    { params: { path: { workspace: slug, template: key ?? '' } } },
    { enabled: Boolean(key), staleTime: 60_000 },
  )
  const live = q.data?.liveVersion
  const differs = Boolean(live && q.data?.draftChanges)
  const v = $api.useQuery(
    'get',
    `${API}/templates/{template}/versions/{version}`,
    { params: { path: { workspace: slug, template: key ?? '', version: live ?? 0 } } },
    { enabled: differs, staleTime: 60_000 },
  )
  return (differs ? v.data?.variables : q.data?.variables) ?? []
}

function KeysCard({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(mo.keysTitle, { count: o.keys.length })}</h2>
        <Link to="/w/$workspace/transactional/setup" params={{ workspace: slug }} className="tx-lnk">
          {t(mo.manageKeys)}
        </Link>
      </div>
      <div className="pad flex flex-col" style={{ paddingTop: 4, paddingBottom: 6 }}>
        {o.keys.length === 0 ? <p className="k my-2">{t(mo.noKeys)}</p> : null}
        {o.keys.map((k) => (
          <div key={k.id} className="tx-vline" style={{ alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap' }}>
            <div className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold">{k.name}</span>
              <span className="k">
                {!k.lastUsedAt
                  ? t(mo.keyNeverUsed)
                  : k.mode === 'test'
                    ? t(mo.keyUsedTest, { when: ago(k.lastUsedAt, now) })
                    : t(mo.keyUsed, { when: ago(k.lastUsedAt, now), ip: k.lastUsedIp ?? '—' })}
              </span>
            </div>
            <KeyBadges preset={k.preset} mode={k.mode} />
          </div>
        ))}
      </div>
    </section>
  )
}

function alertTemplate(a: Alert) {
  return a.template?.name ?? ''
}

/** An outage reads as its span ("9 Oct, 14:02–14:06"), an alert as when it fired ("3 Oct, 09:15"). */
function incidentWhen(a: Alert) {
  const start = new Date(a.startedAt)
  const end = a.kind === 'failover' && a.endedAt ? new Date(a.endedAt) : null
  return end ? `${dayMonthDate(start)}, ${clock(start)}–${clock(end)}` : `${dayMonthDate(start)}, ${clock(start)}`
}

function IncidentsCard({ o }: { o: Overview }) {
  const t = useT()
  const active = o.alerts.length > 0 || Boolean(o.failover)
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(mo.incidentsTitle)}</h2>
        {o.failover ? (
          <Badge tone="info" dot>
            {t(mo.failoverActive)}
          </Badge>
        ) : active ? (
          <Badge tone="bad" dot>
            {t(mo.alertActive)}
          </Badge>
        ) : (
          <Badge tone="ok" dot>
            {t(mo.allClear)}
          </Badge>
        )}
      </div>
      <div className="pad flex flex-col" style={{ paddingTop: 4 }}>
        {o.incidents.length === 0 ? <p className="k my-2">{t(mo.noIncidents)}</p> : null}
        {o.incidents.map((a) => (
          <div key={a.id} className="tx-ev">
            <span className="ic" style={a.kind === 'failover' ? { background: '#DBEAFE', color: '#1E40AF' } : { background: '#FEF3C7', color: '#92400E' }}>
              <Icon name={a.kind === 'failover' ? 'arrowRight' : 'clock'} size={14} />
            </span>
            <div className="tx">
              <span className="font-semibold">{incidentTitle(t, a)}</span>
              <span className="k">{incidentDetail(t, a)}</span>
            </div>
            <span className="tm">{dayMonthDate(a.startedAt)}</span>
          </div>
        ))}
      </div>
    </section>
  )
}

function incidentTitle(t: ReturnType<typeof useT>, a: Alert) {
  switch (a.kind) {
    case 'failover':
      return t(mo.incidentFailover, { from: a.provider?.name ?? '', to: a.fallback ?? '' })
    case 'bounce_rate':
      return t(mo.incidentBounce, { template: alertTemplate(a) })
    case 'failed':
      return t(mo.incidentFailed, { template: alertTemplate(a) })
  }
  return t(mo.incidentMedian, { template: alertTemplate(a) })
}

function incidentDetail(t: ReturnType<typeof useT>, a: Alert) {
  const when = incidentWhen(a)
  switch (a.kind) {
    case 'failover':
      return t(mo.incidentFailoverDetail, { when, messages: a.messages ?? a.value, to: a.fallback ?? '', lost: a.lost ?? 0 })
    case 'bounce_rate': {
      const top = (a.domains ?? []).slice(0, 2)
      const domains = top.map((d) => d.domain)
      // Mistyped domains (gmial.com) mean bad addresses at sign-up, not a provider problem.
      const typos = top.length > 0 && top.every((d) => d.typoOf)
      return t(mo.incidentBounceDetail, {
        when,
        value: trimPct(a.value),
        limit: trimPct(a.threshold),
        mostly: !domains.length ? 'none' : typos ? 'typos' : 'domains',
        domains: andList(domains),
      })
    }
    case 'failed':
      return t(mo.incidentFailedDetail, { when, value: a.value, limit: a.threshold })
  }
  return t(mo.incidentMedianDetail, { when, value: secs(a.value * 1000), limit: secs(a.threshold * 1000) })
}

function AlertNote({ a, hour }: { a: Alert; hour: Schemas['TransactionalHour'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const seen = $api.useMutation('post', `${API}/alerts/{alertId}/seen`)
  const name = alertTemplate(a)
  const title =
    a.kind === 'bounce_rate'
      ? t(mo.alertBounceTitle, { template: name, value: trimPct(a.value), limit: trimPct(a.threshold) })
      : a.kind === 'failed'
        ? name
          ? t(mo.alertFailedTitle, { template: name, value: a.value, limit: a.threshold })
          : t(mo.alertWorkspaceFailedTitle, { value: a.value, limit: a.threshold })
        : name
          ? t(mo.alertMedianTitle, { template: name, value: secs(a.value * 1000), limit: secs(a.threshold * 1000) })
          : t(mo.alertWorkspaceMedianTitle, { value: secs(a.value * 1000), limit: secs(a.threshold * 1000) })
  const domains = (a.domains ?? []).map((d) => `${d.domain} ${count(d.count)}`).join(', ')
  const status = a.kind === 'bounce_rate' ? 'bounced' : a.kind === 'failed' ? 'failed' : undefined
  return (
    <section className="flex flex-col gap-3.5">
      <Note tone={a.kind === 'median_delivery' ? 'warn' : 'bad'}>
        <div className="flex min-w-0 flex-col gap-1">
          <span className="font-bold">{title}</span>
          {a.kind === 'bounce_rate' && domains ? <span>{t(mo.alertBounceDomains, { domains })}</span> : null}
          <div className="mt-2 flex flex-wrap gap-2">
            <ButtonLink
              size="sm"
              to="/w/$workspace/transactional/messages"
              params={{ workspace: slug }}
              search={{ template: a.template?.id, status, range: '1h' } as never}
            >
              {a.bounced || a.kind === 'failed' ? t(mo.openMessages, { count: a.kind === 'failed' ? a.value : (a.bounced ?? 0) }) : t(mo.openTheMessages)}
            </ButtonLink>
            <ButtonLink size="sm" to="/w/$workspace/transactional/settings" params={{ workspace: slug }} hash="alerts">
              {t(mo.adjustLimit)}
            </ButtonLink>
            {!a.seenAt ? (
              <Button
                size="sm"
                loading={seen.isPending}
                onClick={() =>
                  seen.mutate(
                    { params: { path: { workspace: slug, alertId: a.id } } },
                    {
                      onSuccess: () => {
                        toast({ kind: 'success', title: t(mo.markedSeen) })
                        void qc.invalidateQueries()
                      },
                      onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                    },
                  )
                }
              >
                {t(mo.markSeen)}
              </Button>
            ) : null}
          </div>
        </div>
      </Note>
      {a.kind === 'bounce_rate' ? (
        <div className="tx-stats">
          <Stat label={t(mo.hourTemplate, { template: name })} value={count(a.sent ?? 0)} sub={t(mo.hourSent)} />
          <Stat label={t(mo.hourBounced)} value={trimPct(a.value)} sub={t(mo.hourBouncedSub, { count: a.hard ?? a.bounced ?? 0 })} hot />
          <Stat label={t(mo.hourOthers)} value={trimPct(hour.bounceRate)} sub={t(mo.hourOthersSub)} />
          <Stat label={t(mo.hourDelivery)} value={secs(hour.medianMs)} sub={t(mo.hourDeliverySub)} />
        </div>
      ) : null}
    </section>
  )
}

function NoRouteNote({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/routing')
  const system = o.route.system
  return (
    <Note tone="warn" role="note">
      <div className="flex min-w-0 flex-col gap-1">
        <span className="font-bold">{t(mo.noRouteTitle)}</span>
        <span>
          <T msg={mo.noRouteBody} values={{ code: <span className="mono">{'no_healthy_route'}</span> }} />
        </span>
        {o.can.editRoute ? (
          <div className="mt-2 flex flex-wrap gap-2">
            <ButtonLink size="sm" variant="primary" to="/w/$workspace/providers/routing" params={{ workspace: slug }}>
              {t(mo.chooseProvider)}
            </ButtonLink>
            {system ? (
              <Button
                size="sm"
                loading={save.isPending}
                onClick={() =>
                  save.mutate(
                    {
                      params: { path: { workspace: slug } },
                      body: { streams: [{ stream: 'transactional', splits: [{ providerId: system.id, weight: 100 }], failover: 'automatic', conditions: [] }] },
                    },
                    {
                      onSuccess: () => {
                        toast({ kind: 'success', title: t(mo.usedSystem, { provider: system.name }) })
                        void qc.invalidateQueries()
                      },
                      onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                    },
                  )
                }
              >
                {t(mo.useSystem, { provider: system.name })}
              </Button>
            ) : null}
          </div>
        ) : null}
      </div>
    </Note>
  )
}

function FailoverCard({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const f = o.failover!
  const since = clock(f.since)
  return (
    <section className="card flex flex-col gap-3" style={{ padding: '16px 18px', borderColor: '#BFDBFE', background: '#F8FBFF' }} role="status">
      <div className="row" style={{ justifyContent: 'space-between' }}>
        <span className="inline-flex items-center gap-2 font-bold">
          <Icon name="alertCircle" size={16} />
          {t(mo.failoverTitle, { from: f.from.name, to: f.to.name })}
        </span>
        <Badge tone="info" dot>
          {t(mo.failoverActive)}
        </Badge>
      </div>
      <div className="tx-flowr">
        <span className="tx-pill" style={{ borderColor: '#FCA5A5', background: '#FFF5F5' }}>
          <span className="dot" style={{ color: '#B91C1C' }} />
          {f.code ? t(mo.failoverFrom, { from: f.from.name, code: f.code, since }) : t(mo.failoverFromNoCode, { from: f.from.name, since })}
        </span>
        <span className="arw" aria-hidden="true">
          →
        </span>
        <span className="tx-pill" style={{ borderColor: '#86EFAC', background: '#F0FDF4' }}>
          <span className="dot" style={{ color: '#15803D' }} />
          {t(mo.failoverTo, { to: f.to.name })}
        </span>
        <span className="k">{t(mo.failoverPolicy)}</span>
      </div>
      <div className="flex flex-wrap gap-5 text-[13px]">
        <span>
          <T msg={mo.failoverMessages} values={{ count: <b>{count(f.messages)}</b>, to: f.to.name }} />
        </span>
        <span>
          <T msg={mo.failoverLost} values={{ count: <b>{count(f.lost)}</b> }} />
        </span>
        {f.medianMs !== undefined ? (
          <span>
            <T msg={mo.failoverMedian} values={{ value: <b>{secs(f.medianMs)}</b> }} />
          </span>
        ) : null}
        <span className="k">{f.recovering ? t(mo.failoverRecovering, { from: f.from.name }) : t(mo.failoverBack, { from: f.from.name })}</span>
      </div>
      <div className="flex flex-wrap gap-2">
        <ButtonLink size="sm" to="/w/$workspace/providers" params={{ workspace: slug }}>
          {t(mo.providerStatus)}
        </ButtonLink>
        <ButtonLink size="sm" to="/w/$workspace/transactional/messages" params={{ workspace: slug }} search={{ provider: f.to.id, range: '24h' } as never}>
          {t(mo.messagesSince, { since })}
        </ButtonLink>
      </div>
    </section>
  )
}

function FirstRun() {
  const t = useT()
  const { slug, me } = useWorkspace()
  const canEdit = me.permissions.includes('transactional:edit')
  const steps = [t(mo.stepSender), t(mo.stepRoute), t(mo.stepTemplate), t(mo.stepKey), t(mo.stepWatch)]
  return (
    <section className="flex flex-col gap-3.5">
      <div className="card tx-empty">
        <span className="tt">{t(mo.firstRunTitle)}</span>
        <span className="ds">{t(mo.firstRunBody)}</span>
        <ol className="m-0 mt-1.5 flex list-none flex-wrap justify-center gap-2 p-0">
          {steps.map((s, i) => (
            <li key={s} className="chip" style={{ height: 30 }}>
              <span className="tx-num" style={{ fontWeight: 700, color: '#0F766E' }}>
                {i + 1}
              </span>
              {s}
            </li>
          ))}
        </ol>
        <div className="mt-2 flex flex-wrap justify-center gap-2.5">
          {canEdit ? (
            <ButtonLink variant="primary" size="lg" to="/w/$workspace/transactional/guide" params={{ workspace: slug }}>
              {t(mo.startGuide)}
            </ButtonLink>
          ) : null}
          {canEdit ? (
            <ButtonLink to="/w/$workspace/transactional/import" params={{ workspace: slug }}>
              {t(mt.importTemplates)}
            </ButtonLink>
          ) : null}
          <ButtonLink to="/w/$workspace/transactional/setup" params={{ workspace: slug }}>
            {t(mo.apiReference)}
          </ButtonLink>
        </div>
        <span className="k">
          <T msg={mo.firstRunNote} values={{ code: <span className="mono">{'template_not_found'}</span> }} />
        </span>
      </div>
    </section>
  )
}
