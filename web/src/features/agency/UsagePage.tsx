import { Link, useNavigate, useSearch } from '@tanstack/react-router'
import { Fragment, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { RoleBadge } from '@/features/access/parts'
import { formatList, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { useAgencyCan, useRefreshAgency } from './api'
import { AgencyPage } from './AgencyShell'
import { m } from './messages'
import { calendarDate, ClientTile, dayMonth, PersonChip } from './parts'
import { rateText, usd } from './rate'
import { m as um } from './usage.messages'

type View = Schemas['AgencyUsageView']
type Row = Schemas['AgencyUsageRow']
type Rate = Schemas['AgencyRate']
type Search = { month?: string; open?: string }

const num = (v: number) => v.toLocaleString('en-US')

function monthLabel(month: string, style: 'long' | 'short' = 'long') {
  const [y, mo] = month.split('-').map(Number)
  const d = new Date(Date.UTC(y!, mo! - 1, 1))
  return new Intl.DateTimeFormat('en-GB', { month: style, year: 'numeric', timeZone: 'UTC' }).format(d)
}

/** Agency › Usage (AgencyUsage): each client's month, the provider cost and what to rebill. */
export function UsagePage() {
  const t = useT()
  const search = useSearch({ strict: false }) as Search
  const navigate = useNavigate()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/agency/usage', { params: { query: search.month ? { month: search.month } : {} } })
  const d = q.data
  const set = (patch: Partial<Search>) => void navigate({ to: '/w/$workspace/agency/usage', params: { workspace: slug }, search: { month: search.month, open: search.open, ...patch }, replace: true })
  return (
    <AgencyPage
      title={t(m.navUsage)}
      subtitle={t(um.subtitle)}
      actions={
        <>
          <select
            className="in"
            style={{ width: 220 }}
            value={d?.month ?? search.month ?? ''}
            onChange={(e) => set({ month: e.target.value, open: undefined })}
            aria-label={t(um.monthLabel)}
          >
            {(d?.months ?? []).map((mo) => (
              <option key={mo.month} value={mo.month}>
                {mo.partial ? t(um.monthPartial, { month: monthLabel(mo.month) }) : monthLabel(mo.month)}
              </option>
            ))}
          </select>
          {d?.canSeeCosts ? (
            <a className="btn" href={`/api/v1/agency/usage/export?month=${d.month}`} download>
              <Icon name="export" size={16} />
              {t(um.exportCsv)}
            </a>
          ) : null}
        </>
      }
    >
      {q.isError ? (
        <ErrorPanel title={t(um.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !d ? (
        <div className="flex flex-col gap-5" aria-busy="true">
          <Skeleton w="100%" h={96} r={10} />
          <Skeleton w="100%" h={480} r={10} />
        </div>
      ) : (
        <>
          {!d.canSeeCosts && d.reason ? (
            <Note tone="neutral" label={<b>{t(um.readOnly)}</b>}>
              {d.reason}
            </Note>
          ) : null}
          <Tiles view={d} />
          <UsageTable view={d} open={search.open} onOpen={(id) => set({ open: id === search.open ? undefined : id })} />
        </>
      )}
    </AgencyPage>
  )
}

function Tiles({ view }: { view: View }) {
  const t = useT()
  const names = (pick: (r: Row) => number) => formatList(view.rows.filter((r) => pick(r) > 0).map((r) => r.workspace.name))
  const billed = view.totals.billed
  const own = view.rows.find((r) => r.workspace.kind === 'agency')
  const tiles = [
    { label: t(um.tileEmails), value: num(view.totals.emails), sub: t(um.tileEmailsSub, { n: view.totals.workspaces }) },
    { label: t(um.tileSegments), value: num(view.totals.segments), sub: view.totals.segments ? names((r) => r.segments) : t(um.noneYet) },
    { label: t(um.tilePush), value: num(view.totals.push), sub: view.totals.push ? names((r) => r.push) : t(um.noneYet) },
  ]
  if (view.canSeeCosts) {
    tiles.push({ label: t(um.tileCost), value: usd(view.totals.cost), sub: t(um.tileCostSub) })
    tiles.push({
      label: view.partial ? t(um.tileRebillSoFar) : t(um.tileRebill),
      value: usd(view.totals.toRebill),
      sub: own && !own.rate ? t(um.tileRebillSubOwn, { n: billed }) : t(um.tileRebillSub, { n: billed }),
    })
  }
  return (
    <div className="ag-stats">
      {tiles.map((x) => (
        <div key={x.label} className="card ag-stat">
          <span className="k">{x.label}</span>
          <span className="v">{x.value}</span>
          <span className="s">{x.sub}</span>
        </div>
      ))}
    </div>
  )
}

function costLine(t: ReturnType<typeof useT>, r: Row) {
  if (!r.providers && !r.costs.length) return t(um.noProvider)
  if (r.costs.length === 1) {
    const c = r.costs[0]!
    if (c.basis === 'plan') return c.entered ? t(um.planEntered, { provider: c.provider, client: r.workspace.name }) : t(um.plan, { provider: c.provider })
    return c.provider
  }
  return r.costs.map((c) => t(um.costPart, { provider: shortName(c.provider), amount: usd(c.cost) })).join(' · ')
}

function shortName(provider: string) {
  return provider === 'Amazon SES' ? 'SES' : provider
}

function UsageTable({ view, open, onOpen }: { view: View; open?: string; onOpen: (id: string) => void }) {
  const t = useT()
  const canRate = useAgencyCan('agency:usage') && view.canSeeCosts
  const [editing, setEditing] = useState<Row>()
  const month = monthLabel(view.month)
  const openRow = view.rows.find((r) => r.workspace.id === open)
  const caption = view.partial
    ? t(um.partialCaption, { from: dayMonth(view.from, 'UTC'), to: dayMonth(view.to, 'UTC'), updated: view.updatedAt ? updatedClock(view.updatedAt) : '–' })
    : openRow
      ? t(um.captionOpen, { client: openRow.workspace.name })
      : t(um.caption)
  const cols = view.canSeeCosts ? 8 : 4
  return (
    <section className="card" aria-label={view.partial ? t(um.titlePartial, { month }) : t(um.title, { month })}>
      <div className="ag-c8h">
        <div>
          <h2 className="h2 flex items-center gap-2">
            {view.partial ? t(um.titlePartial, { month }) : t(um.title, { month })}
            {view.partial ? <Badge tone="warn">{t(um.partial)}</Badge> : null}
          </h2>
          <span className="k">{view.canSeeCosts ? caption : t(um.volumesOnly)}</span>
        </div>
      </div>
      <div className="relative overflow-x-auto">
        <table className="tbl" aria-label={t(um.title, { month })}>
          <thead>
            <tr>
              <th scope="col">{t(m.colClient)}</th>
              <th scope="col" className="r">
                {t(um.colEmails)}
              </th>
              <th scope="col" className="r">
                {t(um.colSegments)}
              </th>
              <th scope="col" className="r">
                {t(um.colPush)}
              </th>
              {view.canSeeCosts ? (
                <>
                  <th scope="col" className="r">
                    {t(um.colCost)}
                  </th>
                  <th scope="col">{t(um.colRate)}</th>
                  <th scope="col" className="r">
                    {view.partial ? t(um.colRebillSoFar) : t(um.colRebill)}
                  </th>
                  <th scope="col">
                    <span className="vh">{t(um.colActions)}</span>
                  </th>
                </>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {view.rows.map((r) => {
              const isOpen = r.workspace.id === open
              return (
                <Fragment key={r.workspace.id}>
                  <tr className={isOpen ? 'ag-sel' : undefined}>
                    <td>
                      <button
                        type="button"
                        className="flex cursor-pointer items-center gap-2.5 border-0 bg-transparent p-0 text-left font-semibold text-ink"
                        aria-expanded={view.canSeeCosts ? isOpen : undefined}
                        onClick={() => (view.canSeeCosts ? onOpen(r.workspace.id) : undefined)}
                      >
                        <ClientTile ws={r.workspace} size={22} />
                        {r.workspace.kind === 'agency' ? t(m.agencyWorkspaceName, { name: r.workspace.name }) : r.workspace.name}
                      </button>
                    </td>
                    <td className="r num">{num(r.emails)}</td>
                    <td className="r num">{r.segments ? num(r.segments) : t(m.dash)}</td>
                    <td className="r num">{r.push ? num(r.push) : t(m.dash)}</td>
                    {view.canSeeCosts ? (
                      <>
                        <td className="r">
                          <span className="flex flex-col items-end">
                            <b className="num">{usd(r.rebill.cost)}</b>
                            <span className="k">{costLine(t, r)}</span>
                          </span>
                        </td>
                        <td className="whitespace-normal">
                          {r.rate ? (
                            rateText(t, r.rate)
                          ) : (
                            <span className="flex flex-wrap items-center gap-2">
                              <Badge tone="neutral">{t(um.notBilled)}</Badge>
                              {r.workspace.kind === 'client' && canRate ? (
                                <Button size="sm" onClick={() => setEditing(r)}>
                                  {t(um.setRate)}
                                </Button>
                              ) : null}
                            </span>
                          )}
                        </td>
                        <td className="r">
                          <b className="num">{r.rebill.toRebill != null ? usd(r.rebill.toRebill) : t(m.dash)}</b>
                        </td>
                        <td className="r">
                          {canRate && (r.rate || r.workspace.kind === 'agency') ? (
                            <Button size="sm" onClick={() => setEditing(r)}>
                              {t(um.editRate)}
                            </Button>
                          ) : null}
                        </td>
                      </>
                    ) : null}
                  </tr>
                  {isOpen && view.canSeeCosts ? (
                    <tr className="ag-xp">
                      <td colSpan={cols}>
                        <RowDetail row={r} view={view} />
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              )
            })}
            <tr>
              <td className="font-bold">{view.partial ? t(um.totalSoFar, { n: view.totals.workspaces }) : t(um.total, { n: view.totals.workspaces })}</td>
              <td className="r num font-bold">{num(view.totals.emails)}</td>
              <td className="r num font-bold">{num(view.totals.segments)}</td>
              <td className="r num font-bold">{num(view.totals.push)}</td>
              {view.canSeeCosts ? (
                <>
                  <td className="r num font-bold">{usd(view.totals.cost)}</td>
                  <td />
                  <td className="r num font-bold">{usd(view.totals.toRebill)}</td>
                  <td />
                </>
              ) : null}
            </tr>
          </tbody>
        </table>
      </div>
      {view.canSeeCosts ? (
        <div className="flex flex-wrap items-center justify-between gap-2.5 border-t border-line-2 px-[18px] py-3">
          <span className="k">{t(um.foot)}</span>
          <span className="k">{t(um.footExport)}</span>
        </div>
      ) : null}
      {editing ? <RateDialog row={editing} view={view} onClose={() => setEditing(undefined)} /> : null}
    </section>
  )
}

function updatedClock(iso: string) {
  const d = new Date(iso)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const streamLabel: Record<string, typeof um.streamBroadcast> = {
  broadcast: um.streamBroadcast,
  automation: um.streamAutomation,
  transactional: um.streamTransactional,
  system: um.streamSystem,
}

function RowDetail({ row, view }: { row: Row; view: View }) {
  const t = useT()
  const r = row.rebill
  return (
    <div className="grid gap-5 rounded-lg border border-[#BFE5DD] bg-[#F0FDFA] px-4 py-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(200px,100%),1fr))' }}>
      <div className="flex flex-col gap-1.5">
        <b className="text-[13px]">{t(um.byStream)}</b>
        {row.streams.length ? (
          row.streams.map((s) => (
            <span key={s.key} className="flex justify-between gap-3 text-[13px]">
              <span>{t(streamLabel[s.key] ?? um.streamSystem)}</span>
              <b className="num">{num(s.units)}</b>
            </span>
          ))
        ) : (
          <span className="k">{t(um.noEmails)}</span>
        )}
      </div>
      <div className="flex flex-col gap-1.5">
        <b className="text-[13px]">{t(um.byCountry)}</b>
        {row.countries.length ? (
          row.countries.map((c) => (
            <span key={c.key} className="flex justify-between gap-3 text-[13px]">
              <span>{c.label || c.key}</span>
              <b className="num">{num(c.units)}</b>
            </span>
          ))
        ) : (
          <span className="k">{t(um.noTexts, { client: row.workspace.name })}</span>
        )}
        {!row.push ? <span className="k">{t(um.noPush, { client: row.workspace.name })}</span> : null}
      </div>
      <div className="flex flex-col gap-1.5">
        <b className="text-[13px]">{t(um.byProvider)}</b>
        {row.costs.map((c) => (
          <span key={c.provider + c.channel} className="flex justify-between gap-3 text-[13px]">
            <span>
              {c.basis === 'plan'
                ? t(um.costPlan, { provider: c.provider })
                : c.per1000 != null
                  ? t(um.costPer1000, { provider: c.provider, price: usd(c.per1000) })
                  : c.channel === 'sms'
                    ? t(um.costSegments, { provider: c.provider, n: num(c.units) })
                    : c.provider}
            </span>
            <b className="num">{usd(c.cost)}</b>
          </span>
        ))}
        <span className="k">{t(um.pricingOn, { date: dayMonth(view.pricingDate, 'UTC') })}</span>
        {view.entered
          .filter((e) => e.workspace.id === row.workspace.id)
          .map((e) => (
            <Note key={e.provider} tone="info" label={<b>{t(um.enteredTitle, { client: e.workspace.name, provider: e.provider })}</b>}>
              <span className="flex flex-col items-start gap-2">
                <span>{t(um.enteredBody, { provider: e.provider, amount: usd(e.monthly), client: e.workspace.name })}</span>
                <Link to="/w/$workspace/providers" params={{ workspace: e.workspace.slug }} className="btn sm">
                  {t(um.openProviders, { client: e.workspace.name })}
                </Link>
              </span>
            </Note>
          ))}
      </div>
      <div className="flex flex-col gap-1.5">
        <b className="text-[13px]">{row.rate ? t(um.rateHead, { rate: rateText(t, row.rate) }) : t(um.notBilled)}</b>
        <span className="flex justify-between gap-3 text-[13px]">
          <span>{t(um.providerCost)}</span>
          <span className="num">{usd(r.cost)}</span>
        </span>
        {row.rate?.costPlusPct != null && row.rate.flat == null ? (
          <span className="flex justify-between gap-3 text-[13px]">
            <span>{t(um.pctOnCost, { pct: row.rate.costPlusPct })}</span>
            <span className="num">{usd(r.costPlus)}</span>
          </span>
        ) : null}
        {row.rate?.per1000Emails != null && row.rate.flat == null ? (
          <span className="flex justify-between gap-3 text-[13px]">
            <span>{t(um.per1000Line, { amount: usd(row.rate.per1000Emails) })}</span>
            <span className="num">{usd(r.emails)}</span>
          </span>
        ) : null}
        {row.rate?.perSegment != null && row.rate.flat == null ? (
          <span className="flex justify-between gap-3 text-[13px]">
            <span>{t(um.perSegmentLine, { amount: usd(row.rate.perSegment) })}</span>
            <span className="num">{usd(r.segments)}</span>
          </span>
        ) : null}
        {row.rate?.monthlyFee != null && row.rate.flat == null ? (
          <span className="flex justify-between gap-3 text-[13px]">
            <span>{t(um.monthlyFee)}</span>
            <span className="num">{usd(r.monthly)}</span>
          </span>
        ) : null}
        {row.rate?.flat != null ? (
          <span className="flex justify-between gap-3 text-[13px]">
            <span>{t(um.flatLine)}</span>
            <span className="num">{usd(r.flat)}</span>
          </span>
        ) : null}
        {r.toRebill != null ? (
          <span className="flex justify-between gap-3 border-t border-[#BFE5DD] pt-1.5 text-[13px] font-bold">
            <span>{t(um.colRebill)}</span>
            <span className="num">{usd(r.toRebill)}</span>
          </span>
        ) : null}
      </div>
    </div>
  )
}

type Part = 'costPlusPct' | 'per1000Emails' | 'perSegment' | 'monthlyFee' | 'flat'

/** Edit rate (AgencyUsage): combine the parts, see the month at this rate; the client's details beside. */
function RateDialog({ row, view, onClose }: { row: Row; view: View; onClose: () => void }) {
  const t = useT()
  const refresh = useRefreshAgency()
  const start = row.rate ?? {}
  const [on, setOn] = useState<Record<Part, boolean>>({
    costPlusPct: start.costPlusPct != null,
    per1000Emails: start.per1000Emails != null,
    perSegment: start.perSegment != null,
    monthlyFee: start.monthlyFee != null,
    flat: start.flat != null,
  })
  const [vals, setVals] = useState<Record<Part, string>>({
    costPlusPct: start.costPlusPct != null ? String(start.costPlusPct) : '',
    per1000Emails: start.per1000Emails != null ? start.per1000Emails.toFixed(2) : '',
    perSegment: start.perSegment != null ? String(start.perSegment) : '',
    monthlyFee: start.monthlyFee != null ? start.monthlyFee.toFixed(2) : '',
    flat: start.flat != null ? start.flat.toFixed(2) : '',
  })
  const [busy, setBusy] = useState(false)
  const rate: Rate = {}
  for (const k of Object.keys(on) as Part[]) {
    const v = Number(vals[k])
    if (on[k] && vals[k] !== '' && Number.isFinite(v)) rate[k] = v
  }
  // The month at this rate, as the server computes it (cents per part).
  const cents = (v: number) => Math.floor(v * 100 + 0.5 + 1e-9) / 100
  const cost = row.rebill.cost
  const parts: number[] = []
  if (rate.flat != null) parts.push(cents(rate.flat))
  else {
    if (rate.costPlusPct != null) parts.push(cost, cents((cost * rate.costPlusPct) / 100))
    if (rate.per1000Emails != null) parts.push(cents((row.emails / 1000) * rate.per1000Emails))
    if (rate.perSegment != null) parts.push(cents(row.segments * rate.perSegment))
    if (rate.monthlyFee != null) parts.push(cents(rate.monthlyFee))
  }
  const total = cents(parts.reduce((a, b) => a + b, 0))
  const providers = formatList([...new Set(row.costs.map((c) => shortName(c.provider)))])
  const save = async () => {
    setBusy(true)
    try {
      const { data } = await http.PUT('/api/v1/agency/clients/{workspaceId}/rate', { params: { path: { workspaceId: row.workspace.id } }, body: { rate } })
      await refresh()
      toast({ kind: 'success', title: t(um.rateSaved, { client: row.workspace.name }), detail: data ? t(um.appliesFrom, { date: dayMonth(data.effectiveFrom, 'UTC') }) : undefined })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(um.rateFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const fields: { k: Part; label: string; hint: string; unit: '%' | '$' }[] = [
    { k: 'costPlusPct', label: t(um.partCostPlus), hint: providers ? t(um.partCostPlusHint, { providers }) : t(um.partCostPlusHintNone), unit: '%' },
    { k: 'per1000Emails', label: t(um.partPer1000), hint: t(um.partPer1000Hint), unit: '$' },
    { k: 'perSegment', label: t(um.partPerSegment), hint: t(um.partPerSegmentHint), unit: '$' },
    { k: 'monthlyFee', label: t(um.partMonthly), hint: t(um.partMonthlyHint), unit: '$' },
    { k: 'flat', label: t(um.partFlat), hint: t(um.partFlatHint), unit: '$' },
  ]
  const thisMonth = new Date()
  const from = new Date(Date.UTC(thisMonth.getUTCFullYear(), thisMonth.getUTCMonth(), 1))
  return (
    <Dialog
      open
      size="lg"
      onOpenChange={(o) => (o ? null : onClose())}
      eyebrow={<span className="k">{t(um.combine)}</span>}
      title={t(um.editRateTitle, { client: row.workspace.name })}
      footer={
        <>
          <span className="k mr-auto">{t(um.appliesFromAudit, { date: `${dayMonth(from, 'UTC')} ${from.getUTCFullYear()}` })}</span>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={save}>
            {t(um.saveRate)}
          </Button>
        </>
      }
    >
      <div className="grid items-start gap-5" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(320px,100%),1fr))' }}>
        <div className="flex flex-col gap-2.5">
          {fields.map((f) => (
            <label key={f.k} className="ag-optc flex-col! gap-1!">
              <span className="flex w-full items-center gap-2.5">
                <input type="checkbox" checked={on[f.k]} onChange={(e) => setOn({ ...on, [f.k]: e.target.checked })} />
                <b className="flex-1">{f.label}</b>
                {f.unit === '$' ? <span className="k">$</span> : null}
                <input
                  className="in w-[90px] text-right"
                  inputMode="decimal"
                  value={vals[f.k]}
                  disabled={!on[f.k]}
                  aria-label={f.label}
                  onChange={(e) => setVals({ ...vals, [f.k]: e.target.value })}
                />
                {f.unit === '%' ? <span className="k">%</span> : null}
              </span>
              <span className="k pl-[26px]">{f.hint}</span>
            </label>
          ))}
          <div className="rounded-lg border border-line bg-ground px-3 py-2.5 text-[13px]">
            {parts.length ? (
              <>
                {t(um.atThisRate, { month: monthLabel(view.month, 'long').split(' ')[0]!, sum: parts.map(usd).join(' + ') })} <b>{usd(total)}</b>
              </>
            ) : (
              t(um.notBilledAtAll)
            )}
          </div>
        </div>
        <ClientDetails id={row.workspace.id} />
      </div>
    </Dialog>
  )
}

/** Client details (AgencyUsage): what the agency keeps about a client. Client people never see it. */
function ClientDetails({ id }: { id: string }) {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/agency/clients/{workspaceId}/details', { params: { path: { workspaceId: id } } })
  const d = q.data
  if (q.isError) return <span className="err">{errorMessage(q.error)}</span>
  if (!d) return <Skeleton w="100%" h={320} r={10} />
  return (
    <section className="card" aria-label={t(um.detailsTitle, { client: d.workspace.name })}>
      <div className="ag-c8h">
        <div>
          <h3 className="h2">{t(um.detailsTitle, { client: d.workspace.name })}</h3>
          <span className="k">{t(um.detailsCaption)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-4">
        <dl className="kv">
          <dt>{t(um.dManager)}</dt>
          <dd>{d.accountManager ? <PersonChip name={d.accountManager.name} color={d.accountManager.avatarColor} /> : t(m.dash)}</dd>
          <dt>{t(um.dRate)}</dt>
          <dd>{d.rate ? (d.rateSince ? t(um.rateSince, { rate: rateText(t, d.rate), date: `${dayMonth(d.rateSince, 'UTC')} ${new Date(d.rateSince).getUTCFullYear()}` }) : rateText(t, d.rate)) : t(um.notBilled)}</dd>
          <dt>{t(um.dSince)}</dt>
          <dd>
            {d.template
              ? t(um.sinceFrom, { date: calendarDate(d.clientSince), template: d.template })
              : calendarDate(d.clientSince)}
          </dd>
          <dt>{t(um.dProviders)}</dt>
          <dd>{d.providers.length ? d.providers.join(' · ') : t(um.noProvider)}</dd>
          <dt>{t(um.dPeople)}</dt>
          <dd className="flex flex-col gap-1">
            {d.people.length
              ? d.people.map((p) => (
                  <span key={p.email} className="flex flex-wrap items-center gap-1.5">
                    {p.name || p.email} · <RoleBadge role={p.role} />
                    {p.invited ? <Badge tone="warn">{t(um.invited)}</Badge> : null}
                  </span>
                ))
              : t(um.noPeople)}
          </dd>
        </dl>
        {d.billed.length ? (
          <div className="flex flex-col">
            <b className="mb-1 text-[13px]">{t(um.billedTitle)}</b>
            <table className="tbl compact">
              <tbody>
                {d.billed.map((b) => (
                  <tr key={b.month}>
                    <td className="k">{monthLabel(b.month, 'short')}</td>
                    <td>{b.rate ? rateText(t, b.rate, true) : t(um.notBilled)}</td>
                    <td className="r num font-semibold">{b.toRebill != null ? usd(b.toRebill) : t(m.dash)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </div>
    </section>
  )
}
