import { keepPreviousData } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Select } from '@/components/ui/Field'
import { Table } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { channelIcon } from '@/features/channels/ChannelSwitch'
import { m as cm } from '@/features/channels/messages'
import { useChannelSetUp } from '@/features/channels/hooks'
import { formatList, useT, type MessageDescriptor } from '@/i18n'
import { count, dayMonthDate, clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { useRefreshSms } from './common'
import { m as sm } from './messages'
import { m } from './suppressions.messages'
import './sms.css'

type Reason = Schemas['SmsSuppression']['reason']
export interface SmsSuppressionSearch {
  smsReason?: Reason
  smsProvider?: string
  smsPage?: number
}

const reasonWord: Record<Reason, MessageDescriptor> = {
  opted_out: sm.reasonOpted_out,
  landline: sm.reasonLandline,
  not_in_service: sm.reasonNot_in_service,
  invalid: sm.reasonInvalid,
  filtered: sm.reasonFiltered,
  provider_list: sm.reasonProvider_list,
  manual: sm.reasonManual,
}
const reasonTone: Record<Reason, Tone> = { opted_out: 'bad', landline: 'neutral', not_in_service: 'neutral', invalid: 'neutral', filtered: 'warn', provider_list: 'neutral', manual: 'neutral' }

/**
 * Email · 1,874 | SMS · 25: the suppression list's channel, once texting is
 * set up. Above email's filters it heads the card with what the SMS list
 * holds; in the SMS list's filter bar it's the bar's first control.
 */
export function SuppressionChannels({ current, emailTotal, onChange }: { current: 'email' | 'sms'; emailTotal?: number; onChange: (c: 'email' | 'sms') => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const sms = useChannelSetUp('sms')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/suppressions', { params: { path: { workspace: slug } } }, { enabled: sms })
  if (!sms) return null
  const buttons = (
    <>
      <button type="button" className={current === 'email' ? 'on' : undefined} aria-pressed={current === 'email'} onClick={() => onChange('email')}>
        {channelIcon.email}
        {emailTotal !== undefined ? t(m.tabCount, { channel: t(cm.email), count: count(emailTotal) }) : t(cm.email)}
      </button>
      <button type="button" className={current === 'sms' ? 'on' : undefined} aria-pressed={current === 'sms'} onClick={() => onChange('sms')}>
        {channelIcon.sms}
        {q.data ? t(m.tabCount, { channel: t(cm.sms), count: count(q.data.total) }) : t(cm.sms)}
      </button>
    </>
  )
  if (current === 'sms') {
    return (
      <div className="seg" role="group" aria-label={t(cm.switchLabel)}>
        {buttons}
      </div>
    )
  }
  const providers = q.data?.providers.map((p) => p.name) ?? []
  return (
    <div className="sms-supp-ch">
      <div className="ch-seg" role="group" aria-label={t(cm.switchLabel)}>
        {buttons}
      </div>
      <span className="k">{t(m.channelsNote, { count: providers.length, providers: formatList(providers) })}</span>
    </div>
  )
}

/** Audience › Suppressions filtered to SMS (SmsStates). */
export function SmsSuppressions({ search, onSearch, switcher }: { search: SmsSuppressionSearch; onSearch: (s: SmsSuppressionSearch) => void; switcher: ReactNode }) {
  const t = useT()
  const { slug } = useWorkspace()
  const page = search.smsPage ?? 1
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/sms/suppressions',
    { params: { path: { workspace: slug }, query: { reason: search.smsReason, provider: search.smsProvider, page } } },
    { placeholderData: keepPreviousData },
  )
  const [removing, setRemoving] = useState<Schemas['SmsSuppression'] | null>(null)
  const d = q.data
  const providers = d?.providers.map((p) => p.name) ?? []
  return (
    <section className="card min-w-0">
      <div className="sms-filters">
        {switcher}
        <Select aria-label={t(m.reason)} value={search.smsReason ?? ''} onChange={(e) => onSearch({ ...search, smsReason: (e.target.value || undefined) as Reason | undefined, smsPage: undefined })}>
          <option value="">{t(m.anyReason)}</option>
          {(Object.keys(reasonWord) as Reason[]).map((r) => (
            <option key={r} value={r}>
              {t(reasonWord[r])}
            </option>
          ))}
        </Select>
        <Select aria-label={t(m.provider)} value={search.smsProvider ?? ''} onChange={(e) => onSearch({ ...search, smsProvider: e.target.value || undefined, smsPage: undefined })}>
          <option value="">{t(m.anyProvider)}</option>
          {(d?.providers ?? []).map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </Select>
        {d ? <span className="k ml-auto">{t(m.shared, { count: count(d.total), providers: formatList(providers) })}</span> : null}
      </div>
      {q.isError && !d ? (
        <div className="pad">
          <ErrorPanel title={t(m.errorTitle)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : !d ? (
        <div className="pad flex flex-col gap-2.5" aria-busy="true" aria-label={t(m.loading)}>
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} h={18} />
          ))}
        </div>
      ) : d.items.length === 0 ? (
        <div className="pad k">{d.total ? t(m.noMatch) : t(m.empty)}</div>
      ) : (
        <>
          <div className="sms-no-phone flex-col">
            <Table caption={t(m.table)}>
              <thead>
                <tr>
                  <th scope="col">{t(m.colNumber)}</th>
                  <th scope="col">{t(m.colReason)}</th>
                  <th scope="col">{t(m.colSource)}</th>
                  <th scope="col">{t(m.colWhen)}</th>
                  <th scope="col">
                    <span className="vh">{t(m.colActions)}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {d.items.map((s) => (
                  <tr key={s.id}>
                    <td className="mono nw">{s.number}</td>
                    <td>
                      <div className="flex flex-col items-start gap-1">
                        <Badge tone={reasonTone[s.reason]}>{t(reasonWord[s.reason])}</Badge>
                        {s.code ? <span className="mono k text-[12px]">{s.code}</span> : null}
                      </div>
                    </td>
                    <td>{s.source}</td>
                    <td>
                      <div className="flex flex-col gap-0.5">
                        <span>{`${dayMonthDate(s.at)} ${clock(s.at)}`}</span>
                        <span className="k">{s.campaign || s.automation || s.context}</span>
                      </div>
                    </td>
                    <td className="r">
                      {s.removable ? (
                        <Button size="sm" disabled={!d.canEdit} onClick={() => setRemoving(s)}>
                          {t(m.remove)}
                        </Button>
                      ) : (
                        <span className="k inline-block max-w-[210px] whitespace-normal text-right">{t(m.onlyStart)}</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
          <div className="sms-only-phone">
            {d.items.map((s) => (
              <div key={s.id} className="flex flex-col gap-1.5 border-t border-line-2 px-4 py-3">
                <span className="flex items-center justify-between gap-2">
                  <span className="mono text-[13px] font-semibold">{s.number}</span>
                  <Badge tone={reasonTone[s.reason]}>{t(reasonWord[s.reason])}</Badge>
                </span>
                <span className="k">
                  {[s.code, s.source, `${dayMonthDate(s.at)} ${clock(s.at)}`, s.campaign || s.automation || s.context].filter(Boolean).join(' · ')}
                </span>
                {s.removable && d.canEdit ? (
                  <span>
                    <Button size="sm" onClick={() => setRemoving(s)}>
                      {t(m.remove)}
                    </Button>
                  </span>
                ) : null}
              </div>
            ))}
          </div>
          <div className="pad border-t border-line-2">
            <span className="k">{summary(d, t)}</span>
          </div>
          <div className="sms-foot">
            <span className="k">{t(m.showing, { shown: d.items.length, total: count(d.matching) })}</span>
            <div className="flex gap-1.5">
              <Button size="sm" disabled={page <= 1} onClick={() => onSearch({ ...search, smsPage: page - 1 })}>
                {t(m.previous)}
              </Button>
              <Button size="sm" disabled={page * d.pageSize >= d.matching} onClick={() => onSearch({ ...search, smsPage: page + 1 })}>
                {t(m.next)}
              </Button>
            </div>
          </div>
        </>
      )}
      {removing ? <RemoveDialog s={removing} onClose={() => setRemoving(null)} /> : null}
    </section>
  )
}

const nowWord: Record<Reason, MessageDescriptor> = {
  opted_out: m.nowOpted_out,
  landline: m.nowLandline,
  not_in_service: m.nowNot_in_service,
  invalid: m.nowInvalid,
  filtered: m.nowFiltered,
  provider_list: m.nowProvider_list,
  manual: m.nowManual,
}
const laterWord: Record<Reason, MessageDescriptor> = {
  opted_out: m.laterOpted_out,
  landline: m.laterLandline,
  not_in_service: m.laterNot_in_service,
  invalid: m.laterInvalid,
  filtered: m.laterFiltered,
  provider_list: m.laterProvider_list,
  manual: m.laterManual,
}

/** "The 25 today: 16 opted out by text, 5 landlines… The other reasons appear as they happen: …" */
function summary(d: Schemas['SmsSuppressionList'], t: ReturnType<typeof useT>): string {
  const reasons = Object.keys(reasonWord) as Reason[]
  const now = reasons.filter((r) => (d.counts[r] ?? 0) > 0).map((r) => t(nowWord[r], { count: d.counts[r]! }))
  const later = reasons
    .filter((r) => !(d.counts[r] ?? 0))
    .map((r) => (r === 'filtered' && d.filteredRecent ? t(m.laterFilteredRecent, { count: d.filteredRecent }) : t(laterWord[r])))
  return [now.length ? t(m.today, { total: count(d.total), parts: formatList(now) }) : '', later.length ? t(m.otherReasons, { reasons: formatList(later) }) : ''].filter(Boolean).join(' ')
}

function RemoveDialog({ s, onClose }: { s: Schemas['SmsSuppression']; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const remove = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/suppressions/{id}/remove')
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge={t(m.remove)}
      title={t(m.removeTitle, { number: s.number })}
      description={t(m.removeText, { reason: t(reasonWord[s.reason]).toLowerCase() })}
      confirmLabel={t(m.remove)}
      reason={{ label: t(m.removeReason), minLength: 5 }}
      onConfirm={async ({ reason }) => {
        await remove.mutateAsync({ params: { path: { workspace: slug, id: s.id } }, body: { reason: reason ?? '' } })
        void refresh()
        toast({ kind: 'success', title: t(m.removed, { number: s.number }) })
        onClose()
      }}
    />
  )
}
