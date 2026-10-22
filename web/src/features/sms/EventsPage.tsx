import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { problemReference } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Select } from '@/components/ui/Field'
import { Table, Th } from '@/components/ui/Table'
import { formatList, useT, type MessageDescriptor } from '@/i18n'
import { clock, clockSeconds, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { SmsTabs, throwIfForbidden } from './common'
import { m } from './events.messages'
import { m as sm } from './messages'

type Event = Schemas['SmsEvent']
export type EventKind = 'all' | 'receipts' | 'keywords' | 'replies' | 'failures'
export interface EventSearch {
  provider?: string
  kind?: EventKind
  page?: number
}

const normalizedWord: Record<string, MessageDescriptor> = {
  queued: m.nQueued,
  sent: m.nSent,
  delivered: m.nDelivered,
  undelivered: m.nUndelivered,
  failed: m.nFailed,
  opted_out: m.nOpted_out,
  opted_in: m.nOpted_in,
  help: m.nHelp,
  yes: m.nYes,
  reply: m.nReply,
  unmatched: m.nUnmatched,
  pending: m.nPending,
  rejected: m.nRejected,
}

const normalizedTone: Record<string, Tone> = {
  delivered: 'ok',
  opted_in: 'ok',
  yes: 'ok',
  undelivered: 'warn',
  failed: 'bad',
  opted_out: 'bad',
  rejected: 'bad',
  sent: 'info',
  queued: 'info',
}

const reasonWord: Record<string, MessageDescriptor> = {
  unreachable: sm.failUnreachable,
  filtered: sm.failFiltered,
  not_in_service: sm.failNot_in_service,
  landline: sm.failLandline,
  invalid: sm.failInvalid,
  opted_out: sm.failOpted_out,
  rejected: sm.failRejected,
  provider_down: sm.failProvider_down,
  other: sm.failOther,
}

/** The detail column, worded from the event's facts. */
export function eventDetail(e: Event, t: ReturnType<typeof useT>): string {
  const answer = e.replied ? 'reply' : e.providerAnswered ? 'provider' : 'none'
  const lists = formatList(e.lists)
  if (e.error) return t(m.dError, { error: e.error })
  switch (e.normalized) {
    case 'opted_out':
      return e.lists.length ? t(m.dOptedOut, { keyword: e.keyword ?? 'STOP', lists, answer }) : t(m.dOptedOutNone, { keyword: e.keyword ?? 'STOP', answer })
    case 'opted_in':
      return t(m.dOptedIn, { keyword: e.keyword ?? 'START', lists, answer })
    case 'help':
      return t(m.dHelp, { keyword: e.keyword ?? 'HELP', answer })
    case 'yes':
      return t(m.dYes, { keyword: e.keyword ?? 'YES', lists })
    case 'reply':
      return t(m.dReply)
    case 'rejected':
      return t(m.dRejected)
    case 'pending':
      return t(m.dPending)
    case 'unmatched':
      return t(m.dUnmatched)
    case 'undelivered':
    case 'failed': {
      const reason = e.reason && reasonWord[e.reason] ? t(reasonWord[e.reason]!) : t(sm.failOther)
      if (e.reason === 'filtered' && e.filtered) return t(m.dFiltered, { reason, count: e.filtered })
      if (e.temporary) return t(m.dTemporary, { reason })
      if (e.suppressed) return t(m.dSuppressed, { reason })
      return t(m.dFailed, { reason })
    }
  }
  return e.adapter === 'twilio' ? t(m.dReceiptTwilio) : t(m.dReceipt)
}

/** Sending providers › SMS › Events (SmsStates › Events tab). */
export function SmsEventsPage({ search, onSearch }: { search: EventSearch; onSearch: (s: EventSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const kind = search.kind ?? 'all'
  const page = search.page ?? 1
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/events', {
    params: { path: { workspace: slug }, query: { kind, provider: search.provider || undefined, page } },
  })
  throwIfForbidden(q.error)
  const [open, setOpen] = useState<Event | null>(null)
  const data = q.data
  return (
    <Page title={t(sm.pageTitle)} filters={<SmsTabs />}>
      {q.isError ? (
        <ErrorPanel title={t(m.errorTitle)} reference={problemReference(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : null}
      <section className="card min-w-0">
        <div className="sec">
          <div className="flex flex-col gap-0.5">
            <h2 className="h2">{t(m.title)}</h2>
            <span className="k">{t(m.lead)}</span>
          </div>
          <div className="flex flex-wrap gap-2">
            <Select aria-label={t(m.provider)} value={search.provider ?? ''} onChange={(e) => onSearch({ ...search, provider: e.target.value || undefined, page: undefined })}>
              <option value="">{t(m.allProviders)}</option>
              {(data?.providers ?? []).map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </Select>
            <Select aria-label={t(m.type)} value={kind} onChange={(e) => onSearch({ ...search, kind: e.target.value as EventKind, page: undefined })}>
              <option value="all">{t(m.all)}</option>
              <option value="receipts">{t(m.receipts)}</option>
              <option value="keywords">{t(m.keywords)}</option>
              <option value="replies">{t(m.replies)}</option>
              <option value="failures">{t(m.failures)}</option>
            </Select>
          </div>
        </div>
        {!data ? (
          <div className="pad flex flex-col gap-2.5" aria-busy="true" aria-label={t(m.loading)}>
            {[0, 1, 2, 3].map((i) => (
              <Skeleton key={i} h={18} />
            ))}
          </div>
        ) : data.items.length === 0 ? (
          <div className="pad k">{kind === 'all' && !search.provider ? t(m.empty) : t(m.emptyFiltered)}</div>
        ) : (
          <Table caption={t(m.table)}>
            <thead>
              <tr>
                <Th>{t(m.colReceived)}</Th>
                <Th>{t(m.colProvider)}</Th>
                <Th>{t(m.colRaw)}</Th>
                <Th>{t(m.colNormalized)}</Th>
                <Th>{t(m.colDetail)}</Th>
                <Th>{t(m.colNumber)}</Th>
                <Th>{t(m.colSource)}</Th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((e) => (
                <tr key={e.id}>
                  <td className="mono nw">{clockSeconds(e.at)}</td>
                  <td className="nw">{e.provider}</td>
                  <td>
                    <span className="mono inline-block rounded-md bg-[#F3F4F6] px-[7px] py-[3px] text-[12px]">{e.raw}</span>
                  </td>
                  <td className="nw">
                    <Badge tone={normalizedTone[e.normalized] ?? 'neutral'}>{normalizedWord[e.normalized] ? t(normalizedWord[e.normalized]!) : e.normalized}</Badge>
                  </td>
                  <td>
                    {e.kind === 'inbound' && e.normalized !== 'reply' ? (
                      <button type="button" className="sms-lbtn" onClick={() => setOpen(e)}>
                        {eventDetail(e, t)}
                      </button>
                    ) : (
                      <>
                        {eventDetail(e, t)}
                        {e.messageRef && (e.normalized === 'delivered' || e.normalized === 'sent') ? (
                          <>
                            {' · '}
                            <span className="mono whitespace-nowrap">{e.messageRef}</span>
                          </>
                        ) : null}
                      </>
                    )}
                  </td>
                  <td className="mono nw">{e.number}</td>
                  <td>{e.source}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        )}
        {data && data.items.length ? (
          <div className="sms-foot">
            <span className="k">
              {kind === 'all' && !search.provider
                ? t(m.showing, { shown: count(data.items.length), total: count(data.thisMonth) })
                : t(m.showingAll, { shown: count(data.items.length), total: count(data.total) })}
            </span>
            <div className="flex gap-1.5">
              <Button size="sm" disabled={page <= 1} onClick={() => onSearch({ ...search, page: page - 1 })}>
                {t(m.previous)}
              </Button>
              <Button size="sm" disabled={page * data.pageSize >= data.total} onClick={() => onSearch({ ...search, page: page + 1 })}>
                {t(m.next)}
              </Button>
            </div>
          </div>
        ) : null}
      </section>
      {data ? <OptOutSync data={data} /> : null}
      {open ? <InboundDialog e={open} onClose={() => setOpen(null)} /> : null}
    </Page>
  )
}

function OptOutSync({ data }: { data: Schemas['SmsEventList'] }) {
  const t = useT()
  const mode = (r: Schemas['SmsOptOutSyncRow']) =>
    r.mode === 'two_way' ? (r.detail ? t(m.modeTwoWayDetail, { detail: r.detail }) : t(m.modeTwoWay)) : r.mode === 'read_only' ? t(m.modeReadOnly) : t(m.modeOff)
  return (
    <section className="card min-w-0">
      <div className="sec">
        <h2 className="h2">{t(m.syncTitle)}</h2>
        <span className="k">{t(m.syncLead, { count: data.suppressed })}</span>
      </div>
      <Table caption={t(m.syncTable)}>
        <thead>
          <tr>
            <Th>{t(m.colProvider)}</Th>
            <Th>{t(m.colSync)}</Th>
            <Th className="r">
              {t(m.colPulled)}
            </Th>
            <Th className="r">
              {t(m.colPushed)}
            </Th>
            <Th>{t(m.colLastSync)}</Th>
            <Th>
              <span className="vh">{t(m.colStatus)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {data.optOutSync.map((r) => (
            <tr key={r.provider}>
              <td>
                <span className="font-semibold">{r.provider}</span>
              </td>
              <td>{mode(r)}</td>
              <td className="r num">{count(r.pulled)}</td>
              {r.mode === 'read_only' || r.mode === 'off' ? <td className="r k">{t(sm.none)}</td> : <td className="r num">{count(r.pushed)}</td>}
              <td>{r.lastSyncAt ? clock(r.lastSyncAt) : t(m.never)}</td>
              <td>
                <Badge tone={r.status === 'in_sync' ? 'ok' : 'bad'}>{r.status === 'in_sync' ? t(m.inSync) : t(m.failing)}</Badge>
              </td>
            </tr>
          ))}
        </tbody>
      </Table>
    </section>
  )
}

/** A keyword received: what it did, step by step (A STOP received). */
function InboundDialog({ e, onClose }: { e: Event; onClose: () => void }) {
  const t = useT()
  const at = clockSeconds(e.at)
  const lists = formatList(e.lists)
  const steps: { title: string; text: string; tone: string }[] = [
    {
      title: t(m.textedKeyword, { keyword: e.keyword ?? '' }),
      text: e.normalized === 'opted_in' ? t(m.matchedOptIn, { keyword: e.keyword ?? '' }) : t(m.matchedKeyword, { keyword: e.keyword ?? '' }),
      tone: '#DBEAFE',
    },
  ]
  if (e.normalized === 'opted_out' && e.lists.length) steps.push({ title: t(m.consentWithdrawn), text: t(m.consentWithdrawnOn, { lists }), tone: '#DCFCE7' })
  if (e.normalized === 'opted_in' && e.lists.length) steps.push({ title: t(m.consentBack), text: t(m.consentBackOn, { lists }), tone: '#DCFCE7' })
  if (e.normalized === 'opted_out') steps.push({ title: t(m.addedToList), text: t(m.addedToListText, { provider: e.provider }), tone: '#DCFCE7' })
  if (e.normalized === 'opted_in') steps.push({ title: t(m.liftedFromList), text: '', tone: '#DCFCE7' })
  if (e.replied) steps.push({ title: t(m.replySent), text: t(m.replySentText), tone: '#DCFCE7' })
  else if (e.providerAnswered) steps.push({ title: t(m.providerAnswered, { provider: e.provider }), text: t(m.providerAnsweredText), tone: '#DCFCE7' })
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} title={t(m.stopTitle, { number: e.number, keyword: e.keyword ?? '' })} footer={<Button onClick={onClose}>{t(m.close)}</Button>}>
      <div className="flex flex-col">
        {steps.map((s, i) => (
          <div key={i} className="sms-ev">
            <span className="ic" style={{ background: s.tone }} aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#0B1220" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                <path d="m3.5 8.5 3 3 6-7" />
              </svg>
            </span>
            <div className="tx">
              <span className="font-semibold">{s.title}</span>
              {s.text ? <span className="k">{s.text}</span> : null}
            </div>
            <span className="tm">{at}</span>
          </div>
        ))}
        <div className="sms-ev">
          <span className="ic" style={{ background: '#F3F4F6' }} aria-hidden="true" />
          <div className="tx">
            <span className="font-semibold">{t(m.received)}</span>
            <span className="mono k">{t(m.receivedText, { raw: e.raw })}</span>
          </div>
          <span className="tm">{at}</span>
        </div>
      </div>
    </Dialog>
  )
}
