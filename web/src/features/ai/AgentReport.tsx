import { useState } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Card, CardHeader } from '@/components/ui/Card'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { calendarDate, count, shortDate } from '@/lib/format'
import { useDebounced } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { ag } from './agent.messages'

/** Human-only discovery. Selecting a report never selects its editorial content. */
export function AgentReportSelection({ value, onChange }: { value: string; onChange: (id: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [query, setQuery] = useState('')
  const search = useDebounced(query, 250)
  const [page, setPage] = useState(1)
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns', { params: { path: { workspace: slug }, query: { status: 'sent', channel: 'email', q: search, page, pageSize: 20 } } })
  return <div className="flex min-w-0 flex-col gap-3">
    <p className="k m-0">{t(ag.reportHint)}</p>
    <Field label={t(ag.reportSearch)}><Input type="search" maxLength={200} value={query} onChange={(e) => { setQuery(e.target.value); setPage(1); onChange('') }} /></Field>
    {q.error ? <ErrorPanel title={t(ag.reportFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>} /> : q.isPending ? <Skeleton h={60} /> : <>
      <Field label={t(ag.reportSelect)}><Select value={value} disabled={query !== search || q.isFetching} onChange={(e) => onChange(e.target.value)}>
        <option value="">{t(ag.reportNone)}</option>{q.data.items.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
      </Select></Field>
      {page > 1 || q.data.total > 20 ? <nav aria-label={t(ag.reportSelect)} className="flex flex-wrap gap-2">
        <Button disabled={page === 1 || q.isFetching} onClick={() => { setPage(page - 1); onChange('') }}>{t(common.back)}</Button>
        <Button disabled={page * 20 >= q.data.total || q.isFetching} onClick={() => { setPage(page + 1); onChange('') }}>{t(common.next)}</Button>
      </nav> : null}
    </>}
    {value ? <div><Button onClick={() => onChange('')}>{t(ag.reportNone)}</Button></div> : null}
  </div>
}

export function AgentReportReview({ report }: { report: Schemas['ModelSnapshot'] }) {
  const t = useT()
  return <Card className="flex min-w-0 flex-col gap-3 p-5 max-md:p-4"><CardHeader title={t(ag.report)} />
    <p className="k m-0">{t(ag.reportPeriod, { from: calendarDate(report.sentFrom.slice(0, 10), 'shortDate'), through: calendarDate(report.sentThrough.slice(0, 10), 'shortDate'), date: shortDate(report.asOf) })}</p>
    <dl className="m-0 grid grid-cols-2 gap-3 max-md:grid-cols-1">{(['sent', 'delivered', 'humanOpens', 'humanClicks', 'unsubscribes', 'bounces'] as const).map((metric) => {
      const value = metric === 'sent' ? report.sent : report.metrics[metric]
      return <div key={metric} className="flex flex-wrap justify-between gap-2 border-b border-line-2 pb-2 text-[13px]"><dt>{t(ag.reportMetric, { metric })}</dt><dd className="m-0 font-semibold tabular-nums">{value === undefined ? t(ag.reportWithheld) : count(value)}</dd></div>
    })}</dl>
    <p className="k m-0">{t(ag.reportHint)}</p>
  </Card>
}
