import { keepPreviousData, useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { Fragment, useEffect, useMemo, useState, type ReactNode } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { TableScroll } from '@/components/ui/Table'
import { useI18n, useT } from '@/i18n'
import { languageLabel } from '@/features/languages/composerParts'
import { clock, clockSeconds, count, dayMonthYear } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ml } from './log.messages'
import { mt } from './messages'
import { secs, StatusBadge, TxTabs } from './shared'

export type LogItem = Schemas['TransactionalLogItem']
type Log = Schemas['TransactionalLog']
export type LogRange = '1h' | '24h' | '7d' | '30d' | 'custom'
export type LogStatus = 'all' | 'delivered' | 'deferred' | 'bounced' | 'complained' | 'rejected' | 'failed' | 'test' | 'queued' | 'sent'

export interface LogSearch {
  q?: string
  template?: string
  provider?: string
  key?: string
  range?: LogRange
  from?: string
  to?: string
  status?: LogStatus
}

const API = '/api/v1/workspaces/{workspace}/transactional' as const

/** The log's query for a page (before = the previous page's next). */
export function logQuery(slug: string, s: LogSearch, extra: { before?: string; after?: string; limit?: number } = {}) {
  return {
    params: {
      path: { workspace: slug },
      query: {
        q: s.q || undefined,
        template: s.template || undefined,
        provider: s.provider || undefined,
        key: s.key || undefined,
        range: s.range ?? '24h',
        from: s.range === 'custom' ? s.from : undefined,
        to: s.range === 'custom' ? s.to : undefined,
        status: s.status && s.status !== 'all' ? s.status : undefined,
        limit: extra.limit ?? 25,
        before: extra.before,
        after: extra.after,
      },
    },
  }
}

/** Messages on a page of the log, as the boards draw it. */
const PAGE = 10

/** One page of the log, newest first; the first page updates every 10 seconds unless paused. */
function useLogPage(s: LogSearch, before: string | undefined, live: boolean) {
  const { slug } = useWorkspace()
  return useQuery({
    queryKey: ['transactional-log-page', slug, s, before ?? ''],
    queryFn: async () => {
      const { data } = await http.GET(`${API}/messages`, logQuery(slug, s, { before, limit: PAGE }))
      return data as Log
    },
    refetchInterval: live && !before ? 10_000 : false,
    placeholderData: keepPreviousData,
  })
}

/** The log, `first` messages and then 25 at a time, with live updates every 10 seconds unless paused (the message detail's list). */
export function useLog(s: LogSearch, live: boolean, first = 25) {
  const { slug } = useWorkspace()
  return useInfiniteQuery({
    queryKey: ['transactional-log', slug, s, first],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const { data } = await http.GET(`${API}/messages`, logQuery(slug, s, { before: pageParam, limit: pageParam ? 25 : first }))
      return data as Log
    },
    getNextPageParam: (last) => last.next || undefined,
    refetchInterval: live ? 10_000 : false,
  })
}

/** "in the last 24 hours", for the log's footer. */
function useRangeIn(s: LogSearch): string {
  const t = useT()
  switch (s.range ?? '24h') {
    case '1h':
      return t(ml.in1h)
    case '7d':
      return t(ml.in7d)
    case '30d':
      return t(ml.in30d)
    case 'custom':
      return t(ml.inCustom, { from: s.from ?? '…', to: s.to ?? '…' })
  }
  return t(ml.in24h)
}

export function useRangeLabel() {
  const t = useT()
  return (r: LogRange | undefined) => {
    switch (r ?? '24h') {
      case '1h':
        return t(ml.range1h)
      case '7d':
        return t(ml.range7d)
      case '30d':
        return t(ml.range30d)
      case 'custom':
        return t(ml.rangeCustom)
    }
    return t(ml.range24h)
  }
}

/** What a row says next to its status badge: "1.3 s", "suppressed", "soft · mailbox full". */
export function statusDetail(t: ReturnType<typeof useT>, it: LogItem): string {
  switch (it.status) {
    case 'delivered':
    case 'complained':
      if (it.deferrals > 0 && it.deliveryMs !== undefined) return t(ml.afterDeferred, { time: secs(it.deliveryMs), count: it.deferrals })
      return it.deliveryMs !== undefined ? secs(it.deliveryMs) : ''
    case 'deferred':
      return t(ml.deferredTimes, { count: Math.max(1, it.deferrals) })
    case 'bounced': {
      if (!it.bounce) return ''
      // The row says what happened ("mailbox full"); the SMTP code is in the message's detail.
      const detail = (it.bounce.diagnostic ?? '').replace(/^\d{3}(?:[ -]\d\.\d{1,3}\.\d{1,3})?\s+/, '') || it.bounce.class || ''
      return it.bounce.kind === 'soft' ? t(ml.soft, { detail }) : t(ml.hard, { detail })
    }
    case 'rejected':
      return it.rejection?.reason === 'opted_out' ? t(ml.optedOut) : it.rejection?.reason === 'suppressed' ? t(ml.suppressed) : it.error ?? ''
    case 'test':
      return it.testDelivered ? t(ml.testDelivered) : t(ml.testNotDelivered)
    case 'failed':
      return it.error ?? ''
    case 'queued':
      return t(ml.queued)
  }
  return ''
}

export function templateName(t: ReturnType<typeof useT>, it: { raw: boolean; template?: { name: string } }) {
  return it.raw ? t(mt.raw) : (it.template?.name ?? '—')
}

/** A row's language, as its version was sent ("Spanish"), or "—" for a template in one language. */
export function useLanguageName() {
  const { locale } = useI18n()
  return (code?: string) => (code ? languageLabel(code, locale) : '—')
}

function suppressionReason(t: ReturnType<typeof useT>, r: string) {
  switch (r) {
    case 'hard_bounce':
      return t(ml.reasonHardBounce)
    case 'complaint':
      return t(ml.reasonComplaint)
    case 'spam_trap':
      return t(ml.reasonSpamTrap)
    case 'invalid':
      return t(ml.reasonInvalid)
    case 'manual':
      return t(ml.reasonManual)
    case 'imported':
      return t(ml.reasonImported)
    case 'unsubscribe':
      return t(ml.reasonUnsubscribe)
    case 'role':
      return t(ml.reasonRole)
  }
  return r
}

export function optOutSource(t: ReturnType<typeof useT>, s: string) {
  switch (s) {
    case 'preference_center':
      return t(ml.sourcePreferenceCenter)
    case 'unsubscribe_link':
      return t(ml.sourceUnsubscribeLink)
    case 'one_click':
      return t(ml.sourceOneClick)
    case 'member':
      return t(ml.sourceMember)
  }
  return t(ml.sourceApi)
}

function explain(t: ReturnType<typeof useT>, it: Pick<LogItem, 'rejection' | 'error' | 'status'>): { title: string; detail: ReactNode } | null {
  const r = it.rejection
  if (r?.reason === 'suppressed' && r.suppression) {
    const s = r.suppression
    return {
      title: t(ml.suppressedTitle),
      detail: t(ml.suppressedDetail, {
        reason: suppressionReason(t, s.reason),
        date: dayMonthYear(s.since),
        code: s.code ? ` · ${s.code}${s.detail ? ` ${s.detail}` : ''}` : '',
        source: s.source || '—',
      }),
    }
  }
  if (r?.reason === 'opted_out' && r.optOut)
    return { title: t(ml.optedOutTitle, { category: r.optOut.category }), detail: t(ml.optedOutDetail, { date: dayMonthYear(r.optOut.since), source: optOutSource(t, r.optOut.source) }) }
  if (r?.reason === 'test_not_allowed') return { title: t(ml.testTitle), detail: r.allowed?.length ? t(ml.testDetail, { allowed: r.allowed.join(', ') }) : t(ml.testDetailMembers) }
  if (it.status === 'failed') return { title: t(ml.failedTitle), detail: it.error ?? '' }
  return null
}

/** Why a message didn't go out, with what to do (the log's expanded row and the detail). */
export function RejectionExplain({ it, actions }: { it: Pick<LogItem, 'rejection' | 'error' | 'status'>; actions?: ReactNode }) {
  const t = useT()
  const why = explain(t, it)
  if (!why) return null
  const { title, detail } = why
  return (
    <div className="flex flex-wrap items-start gap-3.5 rounded-lg border border-line bg-white px-3.5 py-3">
      <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-1">
        <span className="font-semibold">{title}</span>
        <span className="k" style={{ color: 'var(--ink-nav)' }}>
          {detail}
        </span>
      </div>
      {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
    </div>
  )
}

function Engagement({ it }: { it: LogItem }) {
  const t = useT()
  if (it.clickedAt) return <Badge tone="info">{t(ml.clicked, { time: clock(it.clickedAt) })}</Badge>
  if (it.openedAt) return <Badge tone="info">{t(ml.opened, { time: clock(it.openedAt) })}</Badge>
  return <span className="k">—</span>
}

/** Transactional › Messages (TransactionalMessages): the log, live. */
export function TransactionalMessagesPage({ search, onSearch }: { search: LogSearch; onSearch: (s: LogSearch) => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const canView = me.permissions.includes('transactional:view')
  const [live, setLive] = useState(true)
  // The pages opened so far, as the cursors they start at: Previous goes
  // back one. A new search starts again at the newest messages.
  const searchKey = JSON.stringify(search)
  const [pages, setPages] = useState<{ key: string; cursors: (string | undefined)[] }>({ key: searchKey, cursors: [undefined] })
  if (pages.key !== searchKey) setPages({ key: searchKey, cursors: [undefined] })
  const cursors = pages.key === searchKey ? pages.cursors : [undefined]
  const page = cursors.length - 1
  const q = useLogPage(search, cursors[page], live && canView)
  const first = q.data
  const items = useMemo(() => q.data?.items ?? [], [q.data])
  const newCount = useNewCount(page === 0 ? items : [], live, search)
  const rangeIn = useRangeIn(search)
  const exportHref = useMemo(() => {
    const qs = new URLSearchParams()
    const p = logQuery(slug, search).params.query
    for (const [k, v] of Object.entries(p)) if (v !== undefined && k !== 'limit') qs.set(k, String(v))
    return `/api/v1/workspaces/${encodeURIComponent(slug)}/transactional/messages/export?${qs}`
  }, [slug, search])
  return (
    <Page
      title={t(mt.title)}
      filters={<TxTabs />}
      actions={
        first?.canExport ? (
          <a className="btn" href={exportHref} download>
            {t(ml.exportCsv)}
          </a>
        ) : null
      }
    >
      {!canView ? (
        <ForbiddenState permission="transactional:view" title={t(mt.forbiddenTitle)} />
      ) : (
        <section className="card" aria-label={t(ml.logLabel)}>
          <LogFilters search={search} onSearch={onSearch} filters={first?.filters} />
          <div className="tx-fchips" role="group" aria-label={t(ml.status)} style={{ alignItems: 'center' }}>
            <StatusChips counts={first?.counts} value={search.status ?? 'all'} onChange={(st) => onSearch({ ...search, status: st === 'all' ? undefined : st })} />
            <div className="ml-auto flex items-center gap-2.5">
              <span className={`tx-live${live ? '' : ' off'}`}>
                <span className="dot" />
                {live ? (newCount ? t(ml.liveNew, { count: newCount }) : t(ml.live)) : t(ml.paused)}
              </span>
              <Button size="sm" icon={live ? 'pause' : 'play'} onClick={() => setLive(!live)}>
                {live ? t(ml.pause) : t(ml.resume)}
              </Button>
            </div>
          </div>
          {q.error ? (
            <div className="pad">
              <ErrorPanel title={t(ml.errorTitle)} actions={<Button onClick={() => void q.refetch()}>{t(mt.retry)}</Button>}>
                {errorMessage(q.error)}
              </ErrorPanel>
            </div>
          ) : !first ? (
            <div className="pad flex flex-col gap-3" aria-busy="true" aria-label={t(ml.loading)}>
              {Array.from({ length: 6 }, (_, i) => (
                <Skeleton key={i} w="100%" h={18} />
              ))}
            </div>
          ) : items.length === 0 ? (
            <EmptyLog filtered={Boolean(search.q || search.template || search.provider || search.key || (search.status && search.status !== 'all'))} onClear={() => onSearch({ range: search.range })} />
          ) : (
            <>
              <LogTable items={items} />
              <LogPhoneList items={items} />
              <div className="tx-foot">
                <span className="k">
                  {page === 0
                    ? t(ml.showing, { shown: count(items.length), total: count(first.counts.all), range: rangeIn })
                    : t(ml.showingPage, { from: count(page * PAGE + 1), to: count(page * PAGE + items.length), total: count(first.counts.all), range: rangeIn })}{' '}
                  · {t(ml.retention, { metadata: first.metadataDays, content: first.contentDays })}
                </span>
                <div className="flex gap-2">
                  <Button size="sm" disabledReason={page > 0 ? undefined : t(ml.newestPage)} onClick={() => setPages({ key: searchKey, cursors: cursors.slice(0, -1) })}>
                    {t(ml.previous)}
                  </Button>
                  <Button
                    size="sm"
                    loading={q.isFetching && q.isPlaceholderData}
                    disabledReason={first.next ? undefined : t(ml.oldestPage)}
                    onClick={() => setPages({ key: searchKey, cursors: [...cursors, first.next] })}
                  >
                    {t(ml.next)}
                  </Button>
                </div>
              </div>
            </>
          )}
        </section>
      )}
    </Page>
  )
}

/** How many rows arrived since the log was opened, filtered or went live again (Live · 3 new). */
function useNewCount(items: LogItem[], live: boolean, search: LogSearch) {
  const key = JSON.stringify(search)
  const [base, setBase] = useState<{ key: string; at: number } | null>(null)
  const newest = items[0] ? Date.parse(items[0].at) : undefined
  if (live && newest !== undefined && base?.key !== key) setBase({ key, at: newest })
  if (!live && base) setBase(null)
  if (!live || !base || base.key !== key) return 0
  let n = 0
  for (const it of items) if (Date.parse(it.at) > base.at) n++
  return n
}

function LogFilters({ search, onSearch, filters }: { search: LogSearch; onSearch: (s: LogSearch) => void; filters?: Schemas['TransactionalLogFilters'] }) {
  const t = useT()
  const [text, setText] = useState(search.q ?? '')
  // A search changed elsewhere (a link, Back) replaces what's typed.
  const [shownQ, setShownQ] = useState(search.q)
  if (shownQ !== search.q) {
    setShownQ(search.q)
    if ((search.q ?? '') !== text.trim()) setText(search.q ?? '')
  }
  useEffect(() => {
    const id = window.setTimeout(() => {
      if ((search.q ?? '') !== text.trim()) onSearch({ ...search, q: text.trim() || undefined })
    }, 350)
    return () => window.clearTimeout(id)
  }, [text, search, onSearch])
  const rangeLabel = useRangeLabel()
  return (
    <div className="tx-filters">
      <label className="tx-search" style={{ flex: '1 1 340px', width: 'auto', height: 36 }}>
        <Icon name="search" size={16} />
        <input aria-label={t(ml.search)} placeholder={t(ml.searchPlaceholder)} value={text} onChange={(e) => setText(e.target.value)} />
      </label>
      <select className="in" aria-label={t(ml.template)} value={search.template ?? ''} onChange={(e) => onSearch({ ...search, template: e.target.value || undefined })}>
        <option value="">{t(ml.allTemplates)}</option>
        {filters?.templates.map((f) => (
          <option key={f.id} value={f.id}>
            {f.id === 'raw' ? t(mt.raw) : f.name}
          </option>
        ))}
      </select>
      <select className="in" aria-label={t(ml.provider)} value={search.provider ?? ''} onChange={(e) => onSearch({ ...search, provider: e.target.value || undefined })}>
        <option value="">{t(ml.allProviders)}</option>
        {filters?.providers.map((f) => (
          <option key={f.id} value={f.id}>
            {f.name}
          </option>
        ))}
      </select>
      <select className="in" aria-label={t(ml.key)} value={search.key ?? ''} onChange={(e) => onSearch({ ...search, key: e.target.value || undefined })}>
        <option value="">{t(ml.allKeys)}</option>
        {filters?.keys.map((f) => (
          <option key={f.id} value={f.id}>
            {f.name}
          </option>
        ))}
      </select>
      <select
        className="in"
        aria-label={t(ml.range)}
        value={search.range ?? '24h'}
        onChange={(e) => onSearch({ ...search, range: e.target.value as LogRange, from: undefined, to: undefined })}
      >
        {(['1h', '24h', '7d', '30d', 'custom'] as LogRange[]).map((r) => (
          <option key={r} value={r}>
            {rangeLabel(r)}
          </option>
        ))}
      </select>
      {search.range === 'custom' ? (
        <>
          <label className="lbl" style={{ flexDirection: 'row', alignItems: 'center', gap: 6 }}>
            {t(ml.from)}
            <input className="in" type="date" value={search.from ?? ''} onChange={(e) => onSearch({ ...search, from: e.target.value || undefined })} />
          </label>
          <label className="lbl" style={{ flexDirection: 'row', alignItems: 'center', gap: 6 }}>
            {t(ml.to)}
            <input className="in" type="date" value={search.to ?? ''} onChange={(e) => onSearch({ ...search, to: e.target.value || undefined })} />
          </label>
        </>
      ) : null}
    </div>
  )
}

function StatusChips({ counts, value, onChange }: { counts?: Schemas['TransactionalLogCounts']; value: LogStatus; onChange: (s: LogStatus) => void }) {
  const t = useT()
  const chips: { key: LogStatus; label: string; n?: number }[] = [
    { key: 'all', label: t(ml.all), n: counts?.all },
    { key: 'delivered', label: t(mt.statusDelivered), n: counts?.delivered },
    { key: 'deferred', label: t(mt.statusDeferred), n: counts?.deferred },
    { key: 'bounced', label: t(mt.statusBounced), n: counts?.bounced },
    { key: 'complained', label: t(mt.statusComplained), n: counts?.complained },
    { key: 'rejected', label: t(mt.statusRejected), n: counts?.rejected },
    { key: 'failed', label: t(mt.statusFailed), n: counts?.failed },
    { key: 'test', label: t(mt.statusTest), n: counts?.test },
  ]
  if (counts?.queued) chips.splice(1, 0, { key: 'queued', label: t(mt.statusQueued), n: counts.queued })
  return (
    <>
      {chips.map((c) => (
        <button key={c.key} type="button" className={`tx-fchip${value === c.key ? ' on' : ''}`} aria-pressed={value === c.key} onClick={() => onChange(c.key)}>
          {c.label} <b>{c.n === undefined ? '…' : count(c.n)}</b>
        </button>
      ))}
    </>
  )
}

function EmptyLog({ filtered, onClear }: { filtered: boolean; onClear: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="pad">
      <div className="card tx-empty">
        <span className="tt">{filtered ? t(ml.noMatchTitle) : t(ml.noMessagesTitle)}</span>
        <span className="ds">{filtered ? t(ml.noMatchBody) : t(ml.noMessagesBody)}</span>
        {filtered ? (
          <Button onClick={onClear}>{t(ml.clearFilters)}</Button>
        ) : (
          <ButtonLink to="/w/$workspace/transactional/setup" params={{ workspace: slug }}>
            {t(mt.tabSetup)}
          </ButtonLink>
        )}
      </div>
    </div>
  )
}

function expandable(it: LogItem) {
  return it.status === 'rejected' || it.status === 'failed' || (it.status === 'test' && !it.testDelivered)
}

function LogTable({ items }: { items: LogItem[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const lang = useLanguageName()
  const [open, setOpen] = useState<string | null>(null)
  const showLanguage = items.some((it) => it.language)
  const go = (id: string) => void navigate({ to: '/w/$workspace/transactional/messages/$messageId', params: { workspace: slug, messageId: id } })
  return (
    <div className="tx-no-phone">
      <TableScroll label={t(ml.tableLabel)}>
        <table className="tbl" aria-label={t(ml.tableLabel)}>
          <thead>
            <tr>
              <th scope="col">{t(ml.colTime)}</th>
              <th scope="col">{t(ml.colRecipient)}</th>
              <th scope="col">{t(ml.colTemplate)}</th>
              <th scope="col">{t(ml.colProvider)}</th>
              {showLanguage ? <th scope="col">{t(ml.colLanguage)}</th> : null}
              <th scope="col">{t(ml.colStatus)}</th>
              <th scope="col">{t(ml.colEngagement)}</th>
            </tr>
          </thead>
          <tbody>
            {items.map((it) => {
              const xp = open === it.id
              const detail = statusDetail(t, it)
              return (
                <Fragment key={it.id}>
                  <tr
                    className={`tx-click${xp ? ' tx-xp' : ''}`}
                    onClick={(e) => {
                      if ((e.target as HTMLElement).closest('a,button')) return
                      if (expandable(it)) setOpen(xp ? null : it.id)
                      else go(it.id)
                    }}
                  >
                    <td className="mono nw">
                      <Link to="/w/$workspace/transactional/messages/$messageId" params={{ workspace: slug, messageId: it.id }} style={{ color: 'inherit' }}>
                        {clockSeconds(it.at)}
                      </Link>
                    </td>
                    <td>
                      <span className="mono" style={{ fontSize: 12.5 }}>
                        {it.email}
                      </span>
                      {it.roleAddress ? (
                        <div>
                          <Badge tone="neutral" className="tx-kind" style={{ marginTop: 3 }}>
                            {t(ml.roleAddress)}
                          </Badge>
                        </div>
                      ) : null}
                    </td>
                    <td>
                      <div className="flex min-w-0 flex-col gap-0.5">
                        {it.template && !it.raw ? (
                          <Link
                            to="/w/$workspace/transactional/templates/$templateId"
                            params={{ workspace: slug, templateId: it.template.key }}
                            style={{ color: '#0B1220', fontWeight: 600 }}
                          >
                            {it.template.name}
                          </Link>
                        ) : it.raw ? (
                          <span className="tx-mk self-start">{t(mt.raw)}</span>
                        ) : (
                          <span style={{ fontWeight: 600 }}>{templateName(t, it)}</span>
                        )}
                        <span className="k" style={{ color: '#4B5563' }}>
                          {it.subject}
                        </span>
                      </div>
                    </td>
                    <td className="nw">{it.provider ? it.provider.name : <span className="k">{t(ml.notSent)}</span>}</td>
                    {showLanguage ? <td className="nw">{lang(it.language)}</td> : null}
                    <td>
                      <span className="inline-flex flex-wrap items-center gap-2">
                        <StatusBadge status={it.status} />
                        {detail ? <span className="k">{detail}</span> : null}
                      </span>
                    </td>
                    <td>
                      <Engagement it={it} />
                    </td>
                  </tr>
                  {xp ? (
                    <tr className="tx-xp">
                      <td />
                      <td colSpan={showLanguage ? 6 : 5} style={{ paddingTop: 0 }}>
                        <RejectionExplain
                          it={it}
                          actions={
                            <>
                              {it.rejection?.suppression?.id ? (
                                <ButtonLink size="sm" to="/w/$workspace/audience/suppressions" params={{ workspace: slug }} search={{ q: it.email } as never}>
                                  {t(ml.viewSuppression)}
                                </ButtonLink>
                              ) : null}
                              <ButtonLink size="sm" to="/w/$workspace/transactional/messages/$messageId" params={{ workspace: slug, messageId: it.id }}>
                                {t(ml.openMessage)}
                              </ButtonLink>
                            </>
                          }
                        />
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              )
            })}
          </tbody>
        </table>
      </TableScroll>
    </div>
  )
}

/** Phones: one line per message, as on TransactionalMessagesPhone. */
export function LogPhoneList({ items, current, compact }: { items: LogItem[]; current?: string; compact?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className={compact ? 'flex flex-col' : 'tx-only-phone'}>
      {items.map((it) => {
        const detail = compact ? '' : statusDetail(t, it)
        const on = current === it.id
        return (
          <Link
            key={it.id}
            to="/w/$workspace/transactional/messages/$messageId"
            params={{ workspace: slug, messageId: it.id }}
            className="flex items-start gap-2.5 border-t border-line-2 px-4 py-[11px]"
            style={{ color: '#0B1220', ...(on ? { background: '#F0FDFA', boxShadow: 'inset 3px 0 0 #0F766E' } : null) }}
            aria-current={on ? 'true' : undefined}
          >
            <span className="mono k flex-none" style={{ paddingTop: 2 }}>
              {clock(it.at)}
            </span>
            <span className="flex min-w-0 flex-1 flex-col gap-[3px]">
              <span className="mono overflow-hidden text-ellipsis whitespace-nowrap" style={{ fontSize: 12.5 }}>
                {it.email}
              </span>
              <span className="k" style={{ color: '#4B5563' }}>
                {templateName(t, it)}
                {detail ? ` · ${detail}` : ''}
              </span>
            </span>
            <span className="flex-none">
              <StatusBadge status={it.status} plain />
            </span>
          </Link>
        )
      })}
    </div>
  )
}
