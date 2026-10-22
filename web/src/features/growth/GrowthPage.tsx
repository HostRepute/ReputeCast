import { Link, useNavigate } from '@tanstack/react-router'
import { Tabs as RTabs } from 'radix-ui'
import { Fragment, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Skeleton } from '@/components/ui/Feedback'
import { Input, Select } from '@/components/ui/Field'
import { TableScroll } from '@/components/ui/Table'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { AudienceTabs } from '@/features/audience/AudienceTabs'
import { saveBlob } from '@/features/audience/download'
import { T, useT } from '@/i18n'
import { clock, count, dayMonthDate, dayMonthYear, monthYear, pct } from '@/lib/format'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { colorOf, GrowthChart, seriesName } from './GrowthChart'
import { m } from './messages'
import { dayOf, factor, isImport, ratePct, signedCount, sourceLine, sourceName } from './sourceText'
import '@/features/scoring/scoring.css'
import './growth.css'

type Data = Schemas['GrowthPage']
type Row = Schemas['GrowthSourceRow']
type Period = Data['period']['kind']

export interface GrowthSearch {
  period?: Period
  from?: string
  to?: string
  list?: string
  imports?: boolean
  cohort?: string
}

const growthPath = '/api/v1/workspaces/{workspace}/growth'

function queryOf(s: GrowthSearch) {
  return {
    period: s.period,
    from: s.period === 'custom' ? s.from : undefined,
    to: s.period === 'custom' ? s.to : undefined,
    list: s.list,
    imports: s.imports || undefined,
    cohortSource: s.cohort,
  }
}

/**
 * Audience › Growth: how people join and leave, by source, with each
 * source's quality, cohort retention, UTM parameters and referring pages.
 * Imports are counted apart unless included: moving a list isn't growth.
 */
export function GrowthPage({ search }: { search: GrowthSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const q = $api.useQuery('get', growthPath, { params: { path: { workspace: slug }, query: queryOf(search) } }, { placeholderData: (prev) => prev })
  const d = q.data
  if (!d && q.error instanceof ApiError && q.error.status === 403) throw q.error
  const go = (patch: Partial<GrowthSearch>) =>
    void navigate({ to: '/w/$workspace/audience/growth', params: { workspace: slug }, search: { ...search, ...patch }, replace: true })

  return (
    <Page title={t(m.audience)} documentTitle={t(m.documentTitle)} filters={<AudienceTabs />}>
      {!d ? (
        q.isError ? (
          <LoadError error={q.error} onRetry={() => void q.refetch()} />
        ) : (
          <GrowthSkeleton />
        )
      ) : d.contacts === 0 && !d.members ? (
        <NoContacts />
      ) : (
        <>
          <Filters d={d} search={search} onChange={go} />
          {d.onlyImports && !d.includeImports ? (
            <OnlyImports d={d} />
          ) : (
            <GrowthBody d={d} onCohort={(cohort) => go({ cohort: cohort || undefined })} />
          )}
        </>
      )}
    </Page>
  )
}

/** Tiles, the chart, sources, departures, cohorts, UTM and how Growth counts. */
function GrowthBody({ d, onCohort }: { d: Data; onCohort: (key: string) => void }) {
  return (
    <>
      <Tiles d={d} />
      <ChartCard d={d} />
      <SourcesCard d={d} />
      <div className="flex flex-wrap items-stretch gap-4">
        <DeparturesCard d={d} />
        <CohortsCard d={d} onSource={onCohort} />
      </div>
      <div className="flex flex-wrap items-start gap-4">
        <UtmCard d={d} />
        <CountsCard d={d} />
      </div>
    </>
  )
}

/** Period, list, Include imports, what's counted, Export CSV. */
function Filters({ d, search, onChange }: { d: Data; search: GrowthSearch; onChange: (p: Partial<GrowthSearch>) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canExport = useCan('reports:view')
  const [busy, setBusy] = useState(false)
  const period = d.period.kind
  const range = { from: dayMonthDate(dayOf(d.period.from)), to: dayMonthYear(dayOf(d.period.to)) }
  const exportCsv = async () => {
    setBusy(true)
    try {
      const { data } = await http.GET('/api/v1/workspaces/{workspace}/growth/export', { params: { path: { workspace: slug }, query: queryOf(search) }, parseAs: 'blob' })
      if (data) saveBlob(data as unknown as Blob, `growth-${d.period.from}-${d.period.to}.csv`)
      toast({ title: t(m.exported), detail: t(m.exportedDetail) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.exportFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2.5">
      <div className="flex flex-wrap items-center gap-x-3.5 gap-y-2.5">
        <Segmented
          label={t(m.period)}
          value={period}
          onChange={(p) => onChange(p === 'custom' ? { period: p, from: d.period.from, to: d.period.to } : { period: p, from: undefined, to: undefined })}
          items={[
            { value: '30d', label: t(m.days30) },
            { value: '90d', label: t(m.days90) },
            { value: '12m', label: t(m.months12) },
            { value: 'custom', label: t(m.custom) },
          ]}
        />
        {period === 'custom' ? (
          <span className="flex flex-wrap items-center gap-2">
            <Input type="date" aria-label={t(m.from)} value={search.from ?? d.period.from} max={search.to ?? d.period.to} onChange={(e) => e.target.value && onChange({ from: e.target.value })} />
            <Input type="date" aria-label={t(m.to)} value={search.to ?? d.period.to} min={search.from ?? d.period.from} onChange={(e) => e.target.value && onChange({ to: e.target.value })} />
          </span>
        ) : null}
        <Select style={{ width: 220 }} aria-label={t(m.list)} value={d.list?.id ?? ''} onChange={(e) => onChange({ list: e.target.value || undefined, cohort: undefined })}>
          <option value="">{t(m.allLists)}</option>
          {d.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {t(m.listMembers, { list: l.name })}
            </option>
          ))}
        </Select>
        <label className="gr-imp">
          <input type="checkbox" className="tg" role="switch" checked={d.includeImports} aria-label={t(m.includeImports)} onChange={(e) => onChange({ imports: e.target.checked || undefined })} />
          {t(m.includeImports)}
        </label>
      </div>
      <div className="flex flex-wrap items-center gap-x-3.5 gap-y-2.5">
        <span className="k">{t(d.members ? m.countsMembers : m.countsContacts, range)}</span>
        <Button icon="download" loading={busy} disabledReason={canExport ? undefined : deniedReason('reports:view')} onClick={() => void exportCsv()}>
          {t(m.exportCsv)}
        </Button>
      </div>
    </div>
  )
}

/** "18 Sep", or "18 and 22 Sep" style lists of the import days. */
function importDays(d: Data): string {
  const days = [...new Set(d.tiles.importItems.map((i) => i.day))]
  return days.map((x) => dayMonthDate(dayOf(x))).join(', ')
}

function Tiles({ d }: { d: Data }) {
  const t = useT()
  const list = d.list?.name ?? ''
  const tiles = d.tiles
  const items = tiles.importItems
    .slice()
    .sort((a, b) => b.count - a.count)
    .map((i) => t(m.importItem, { name: i.name, count: i.count }))
    .join(' · ')
  const days = importDays(d)
  const oneDay = new Set(tiles.importItems.map((i) => i.day)).size === 1
  const withoutImports = tiles.net - (d.includeImports ? tiles.imports : 0)
  return (
    <div className="gr-tiles">
      <div className="card gr-stat">
        <span className="k">{t(m.joined)}</span>
        <span className="v">{count(tiles.joined)}</span>
        <span className="s">
          {d.includeImports && tiles.imports
            ? t(m.joinedSubImports, { joined: tiles.joined - tiles.imports, imports: tiles.imports, days: t(m.onDays, { days }) })
            : d.members
              ? t(m.joinedSubList, { list })
              : t(m.joinedSub)}
        </span>
      </div>
      <div className="card gr-stat">
        <span className="k">{t(m.left)}</span>
        <span className="v">{count(tiles.left)}</span>
        <span className="s">{d.members ? t(m.leftSubList, { list }) : t(m.leftSub)}</span>
      </div>
      <div className="card gr-stat">
        <span className="k">{t(m.net)}</span>
        <span className="v">
          <span className={tiles.net >= 0 ? 'up' : 'down'}>{signedCount(tiles.net)}</span>{' '}
          <span className={tiles.net >= 0 ? 'gr-pct up' : 'gr-pct down'}>{`${tiles.netPct >= 0 ? '+' : '−'}${pct(Math.abs(tiles.netPct) * 100)}`}</span>
        </span>
        <span className="s">
          {d.includeImports && tiles.imports ? t(m.netSubImports, { net: signedCount(withoutImports) }) : d.members ? t(m.netSubList, { list }) : t(m.netSub)}
        </span>
      </div>
      <div className="card gr-stat">
        <span className="k">
          {tiles.imports && oneDay ? t(d.includeImports ? m.importsOnIncluded : m.importsOn, { days }) : t(d.includeImports ? m.importsIncluded : m.importsApart)}
        </span>
        <span className="v">{count(tiles.imports)}</span>
        <span className="s">
          {tiles.imports ? t(d.includeImports ? m.importsItemsIncluded : m.importsItems, { items }) : d.members ? t(m.noImportsList, { list }) : t(m.noImports)}
        </span>
      </div>
    </div>
  )
}

function ChartCard({ d }: { d: Data }) {
  const t = useT()
  const [view, setView] = useState<'chart' | 'table'>('chart')
  return (
    <section className="card" aria-labelledby="gr-chart">
      <CardSection
        id="gr-chart"
        title={t(m.chartTitle)}
        caption={d.includeImports && d.tiles.imports ? t(m.chartLeadImports, { count: d.tiles.imports, days: importDays(d) }) : t(m.chartLead)}
        actions={
          <Segmented
            label={t(m.view)}
            value={view}
            onChange={setView}
            items={[
              { value: 'chart', label: t(m.chart) },
              { value: 'table', label: t(m.table) },
            ]}
          />
        }
      />
      <div className="pad flex flex-col gap-3">
        <GrowthChart chart={d.chart} view={view} includeImports={d.includeImports} from={d.period.from} to={d.period.to} />
        <div className="gr-legend">
          {d.chart.series.map((s, i) => (
            <span key={s.key}>
              <span className="gr-sw" style={{ background: colorOf(s, i) }} aria-hidden="true" />
              {seriesName(s, t)} <span className="num gr-strong">{count(s.joined)}</span> <span className="k">{t(m.legendJoined)}</span> ·{' '}
              <span className="num gr-strong">{count(s.left)}</span> <span className="k">{t(m.legendLeft)}</span>
            </span>
          ))}
        </div>
        <span className="k">{t(m.chartFoot)}</span>
      </div>
    </section>
  )
}

/**
 * A rate, or the rate with "3.5× the workspace" when it's flagged. On an
 * API key's row the badge opens the note on what to do.
 */
function RateCell({ rate, flag, digits, open, onToggle, label }: { rate?: number; flag?: number; digits: number; open?: boolean; onToggle?: () => void; label?: string }) {
  const t = useT()
  if (!flag) return <td className="r num">{ratePct(rate, digits)}</td>
  const text = (
    <>
      {t(m.times, { factor: factor(flag) })}
      <span className="gr-fx"> {t(m.theWorkspace)}</span>
    </>
  )
  return (
    <td className="r">
      <span className="gr-flagv">
        <span className="num">{ratePct(rate, digits)}</span>
        {onToggle ? (
          <button type="button" className="badge warn gr-flagb gr-flagbtn" aria-expanded={open} onClick={onToggle}>
            {text}
            {label ? <span className="vh">: {label}</span> : null}
          </button>
        ) : (
          <Badge tone="warn" className="gr-flagb">
            {text}
          </Badge>
        )}
      </span>
    </td>
  )
}

function tooFewText(r: Row, d: Data, t: ReturnType<typeof useT>): string {
  return r.tooFew === 'messages' ? t(m.tooFewMessages, { min: d.minDelivered }) : t(m.tooFewContacts, { min: d.minContacts })
}

/** The contacts filter for a source (Audience › Contacts). */
function ContactsLink({ r, label, small = true }: { r: Row; label?: string; small?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <ButtonLink size={small ? 'sm' : 'md'} to="/w/$workspace/audience" params={{ workspace: slug }} search={{ growthSource: r.source.key }} aria-label={label ?? t(m.contactsOf, { source: sourceName(r.source) })}>
      {label ?? t(m.contacts)}
    </ButtonLink>
  )
}

/** By source: joins, departures and each source's quality. */
function SourcesCard({ d }: { d: Data }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [open, setOpen] = useState<string | null>(null)
  const list = d.list?.name
  const rows = d.sources
  const pc = rows.find((r) => r.source.kind === 'preference_center')
  const allTooFew = d.members && rows.length > 0 && rows.every((r) => r.tooFew)
  const ws = d.workspace
  const apart = (r: Row) => r.source.kind === 'import' && !d.includeImports
  const noJoins = (r: Row) => isImport(r.source) && !r.joined
  const joinedCell = (r: Row) => (apart(r) ? <span className="k">{t(m.countedApart)}</span> : noJoins(r) ? <span className="k">–</span> : count(r.joined ?? 0))
  const netCell = (r: Row) =>
    apart(r) || noJoins(r) ? <span className="k">–</span> : <span className={(r.net ?? 0) >= 0 ? 'gr-up' : 'gr-down'}>{signedCount(r.net ?? 0)}</span>
  const flagged = (r: Row) => Boolean(r.bounceFlag || r.complaintFlag)
  // Only the first flag of an API key's row opens the note.
  const toggle = (r: Row, which: 'bounce' | 'complaint') =>
    r.source.kind === 'api_key' && (which === 'bounce' ? r.bounceFlag : !r.bounceFlag && r.complaintFlag)
      ? { open: open === r.source.key, onToggle: () => setOpen(open === r.source.key ? null : r.source.key), label: t(m.flagShow, { source: sourceName(r.source) }) }
      : {}
  return (
    <section className="card" aria-labelledby="gr-src">
      <CardSection
        id="gr-src"
        title={list ? t(m.sourcesTitleList, { list, count: rows.length }) : t(m.sourcesTitle, { count: rows.length })}
        caption={list ? (pc ? t(m.sourcesLeadList, { count: pc.joined ?? 0, list }) : t(m.sourcesLeadListNone)) : t(m.sourcesLead)}
      />
      <div className="sc-no-phone">
        <TableScroll label={t(m.sourcesLabel)}>
          <table className="tbl sc-t11 gr-srct">
            <thead>
              <tr>
                <th scope="col">{t(m.colSource)}</th>
                <th scope="col" className="r">
                  {t(m.joined)}
                </th>
                <th scope="col" className="r">
                  {t(m.left)}
                </th>
                <th scope="col" className="r">
                  {t(m.colNet)}
                </th>
                <th scope="col" className="r">
                  {t(m.colBounces)}
                </th>
                <th scope="col" className="r">
                  {t(m.colComplaints)}
                </th>
                <th scope="col" className="r">
                  {t(m.colHumanOpens)}
                </th>
                <th scope="col">
                  <span className="vh">{t(m.colContacts)}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <Fragment key={r.source.key}>
                  <tr>
                    <td>
                      <div className="flex min-w-0 flex-col gap-0.5">
                        <SourceLink r={r} />
                        <span className="k">{sourceLine(r.source, list)}</span>
                      </div>
                    </td>
                    <td className="r num">{joinedCell(r)}</td>
                    <td className="r num">{count(r.left)}</td>
                    <td className="r num">{netCell(r)}</td>
                    {r.tooFew ? (
                      <td colSpan={3} className="r">
                        <span className="gr-tfew">{t(m.tooFew)}</span> <span className="k">{tooFewText(r, d, t)}</span>
                      </td>
                    ) : (
                      <>
                        <RateCell rate={r.bounceRate} flag={r.bounceFlag} digits={1} {...toggle(r, 'bounce')} />
                        <RateCell rate={r.complaintRate} flag={r.complaintFlag} digits={2} {...toggle(r, 'complaint')} />
                        <td className="r num">{ratePct(r.humanOpenRate, 1)}</td>
                      </>
                    )}
                    <td className="r">
                      <ContactsLink r={r} label={t(m.contacts)} />
                    </td>
                  </tr>
                  {open === r.source.key ? (
                    <tr className="gr-flagrow">
                      <td colSpan={8}>
                        <FlagNote r={r} d={d} slug={slug} />
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              ))}
              <tr className="gr-total">
                <td className="font-bold">{t(m.total)}</td>
                <td className="r num font-bold">{count(d.total.joined)}</td>
                <td className="r num font-bold">{count(d.total.left)}</td>
                <td className={`r num font-bold ${d.total.net >= 0 ? 'gr-up' : 'gr-down'}`}>{signedCount(d.total.net)}</td>
                <td colSpan={3} className="r k">
                  {t(m.workspaceRates, { bounce: ratePct(ws.bounce, 1), complaint: ratePct(ws.complaint, 2) })}
                </td>
                <td />
              </tr>
            </tbody>
          </table>
        </TableScroll>
      </div>
      <div className="sc-only-phone">
        {rows.map((r) => (
          <div key={r.source.key} className="gr-pcard">
            <span className="flex items-baseline justify-between gap-2.5">
              <SourceLink r={r} bold />
              {apart(r) || noJoins(r) ? null : <span className={`num font-bold ${(r.net ?? 0) >= 0 ? 'gr-up' : 'gr-down'}`}>{signedCount(r.net ?? 0)}</span>}
            </span>
            <span className="k">{sourceLine(r.source, list)}</span>
            <span className="text-[12.5px] text-ink-nav">
              {apart(r) ? t(m.phoneApart, { left: r.left }) : noJoins(r) ? t(m.phoneNoJoins, { left: r.left }) : t(m.phoneJoinedLeft, { joined: r.joined ?? 0, left: r.left })}
            </span>
            {r.tooFew ? (
              <span className="gr-tfew">{t(m.tooFew)}</span>
            ) : (
              <span className="k text-ink-2">{t(m.phoneRates, { bounce: ratePct(r.bounceRate, 1), complaint: ratePct(r.complaintRate, 2), opens: ratePct(r.humanOpenRate, 1) })}</span>
            )}
            {flagged(r) ? (
              <span className="flex flex-wrap gap-1.5">
                {r.bounceFlag ? <PhoneFlag r={r} text={t(m.flagBounces, { factor: factor(r.bounceFlag) })} open={open} setOpen={setOpen} first /> : null}
                {r.complaintFlag ? <PhoneFlag r={r} text={t(m.flagComplaints, { factor: factor(r.complaintFlag) })} open={open} setOpen={setOpen} first={!r.bounceFlag} /> : null}
              </span>
            ) : null}
            {open === r.source.key ? <FlagNote r={r} d={d} slug={slug} phone /> : null}
          </div>
        ))}
        <div className="gr-ptotal">
          <b>{t(m.total)}</b>
          <span>
            <T
              msg={m.phoneTotal}
              values={{
                joined: d.total.joined,
                left: d.total.left,
                net: signedCount(d.total.net),
                b: (c) => <b className="num">{c}</b>,
                n: (c) => <b className={`num ${d.total.net >= 0 ? 'gr-up' : 'gr-down'}`}>{c}</b>,
              }}
            />
          </span>
        </div>
      </div>
      <div className="sc-foot">
        <span className="k" style={{ flex: '1 1 420px' }}>
          {t(m.sourcesFoot, { bounce: ratePct(ws.bounce * d.flagFactor, 1), complaint: ratePct(ws.complaint * d.flagFactor, 2), delivered: d.minDelivered, contacts: d.minContacts })}
        </span>
        <span className="k">{t(m.humanOnly)}</span>
      </div>
      {allTooFew ? (
        <div className="px-4.5 pb-4">
          <div className="note n" role="note">
            <div>{t(m.allTooFew, { min: d.minContacts })}</div>
          </div>
        </div>
      ) : null}
    </section>
  )
}

/** A flag on a source's phone card; an API key's opens the note on what to do. */
function PhoneFlag({ r, text, open, setOpen, first }: { r: Row; text: string; open: string | null; setOpen: (k: string | null) => void; first: boolean }) {
  if (r.source.kind !== 'api_key' || !first) {
    return (
      <Badge tone="warn" className="gr-flagb">
        {text}
      </Badge>
    )
  }
  return (
    <button type="button" className="badge warn gr-flagb gr-flagbtn" aria-expanded={open === r.source.key} onClick={() => setOpen(open === r.source.key ? null : r.source.key)}>
      {text}
    </button>
  )
}

function SourceLink({ r, bold }: { r: Row; bold?: boolean }) {
  const { slug } = useWorkspace()
  return (
    <Link to="/w/$workspace/audience" params={{ workspace: slug }} search={{ growthSource: r.source.key }} className={bold ? 'gr-srcname font-semibold' : 'gr-srcname'}>
      {sourceName(r.source)}
    </Link>
  )
}

/** Why an API key is flagged, and what to do about it. */
function FlagNote({ r, d, slug, phone }: { r: Row; d: Data; slug: string; phone?: boolean }) {
  const t = useT()
  const bounces = Boolean(r.bounceFlag)
  const f = factor((bounces ? r.bounceFlag : r.complaintFlag) ?? 0)
  const rate = bounces ? ratePct(r.bounceRate, 1) : ratePct(r.complaintRate, 2)
  const workspace = bounces ? ratePct(d.workspace.bounce, 1) : ratePct(d.workspace.complaint, 2)
  const title = phone
    ? t(bounces ? m.flagPhoneTitle : m.flagPhoneTitleComplaints, { name: r.source.name, rate })
    : t(bounces ? m.flagTitleBounces : m.flagTitleComplaints, { factor: f })
  const n = r.joined ?? r.contacts
  return (
    <div className="note w gr-flagnote" role="note">
      <span className="sc-note-ic" style={{ color: 'var(--warn-bar)' }}>
        <Icon name="warningTriangle" size={16} strokeWidth={1.8} />
      </span>
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{title}</span>
        <span>{t(bounces ? m.flagBodyBounces : m.flagBodyComplaints, { name: r.source.name, factor: f, rate, workspace })}</span>
        <div className="mt-1.5 flex flex-wrap gap-2">
          <ContactsLink r={r} label={t(m.seeContacts, { count: n })} />
          {phone ? null : (
            <ButtonLink size="sm" to="/w/$workspace/settings/keys" params={{ workspace: slug }}>
              {t(m.openKey)}
            </ButtonLink>
          )}
        </div>
      </div>
    </div>
  )
}

function DeparturesCard({ d }: { d: Data }) {
  const t = useT()
  const dep = d.departures
  const total = dep.unsubscribed + dep.hardBounced + dep.complained + dep.erased
  const rows: [string, number][] = [
    [d.list ? t(m.depUnsubscribedList, { list: d.list.name }) : t(m.depUnsubscribed), dep.unsubscribed],
    [t(m.depHardBounced), dep.hardBounced],
    [t(m.depComplained), dep.complained],
    [t(m.depErased), dep.erased],
  ]
  const max = Math.max(1, ...rows.map((r) => r[1]))
  return (
    <section className="card min-w-0 flex-[1_1_340px]" aria-labelledby="gr-dep">
      <CardSection id="gr-dep" title={t(m.departuresTitle, { count: total })} />
      <div className="pad flex flex-col gap-3">
        {total === 0 ? <span className="k">{t(m.departuresNone)}</span> : null}
        {rows.map(([label, n]) => (
          <div key={label} className="flex flex-col gap-[5px]">
            <div className="flex justify-between gap-2.5 text-[13px]">
              <span>{label}</span>
              <span>
                <span className="num font-semibold">{count(n)}</span> <span className="k">{pct(total ? (100 * n) / total : 0)}</span>
              </span>
            </div>
            <div className="bar">
              <i style={{ width: `${(100 * n) / max}%`, background: 'var(--accent)' }} />
            </div>
          </div>
        ))}
      </div>
      <div className="sc-foot">
        <span className="k">{t(m.departuresFoot)}</span>
      </div>
    </section>
  )
}

function Retention({ r, prev }: { r: Schemas['GrowthRetention']; prev?: Schemas['GrowthRetention'] }) {
  const t = useT()
  if (r.state === 'done') return <span className="num">{pct((r.rate ?? 0) * 100)}</span>
  // The 90-day mark shows once the 30-day one is done.
  if (prev && prev.state !== 'done') return <span className="k">–</span>
  return (
    <span className="gr-coh">
      <Icon name="clock" size={13} strokeWidth={1.7} />
      {r.from ? t(m.inProgressFrom, { day: dayMonthDate(dayOf(r.from)) }) : t(m.inProgress, { day: dayMonthDate(dayOf(r.completeOn ?? '')) })}
    </span>
  )
}

function CohortsCard({ d, onSource }: { d: Data; onSource: (key: string) => void }) {
  const t = useT()
  const picked = d.cohortSources.find((s) => s.key === d.cohortSource)
  const carried = d.sources.find((r) => r.source.kind === 'import' && r.source.importCount)
  const firstJoinInRange = picked?.firstJoin && d.cohorts.length && picked.firstJoin.slice(0, 7) >= d.cohorts[0]!.month
  const pickedRow = picked ? d.sources.find((r) => r.source.key === picked.key) : undefined
  const month = (c: Schemas['GrowthCohort']) => monthYear(dayOf(`${c.month}-15`), 'long')
  return (
    <section className="card min-w-0 flex-[2_1_560px]" aria-labelledby="gr-coh">
      <CardSection
        id="gr-coh"
        title={t(m.cohortsTitle)}
        caption={t(m.cohortsLead)}
        actions={
          <Select style={{ width: 230, maxWidth: '100%' }} aria-label={t(m.cohortSource)} value={d.cohortSource} onChange={(e) => onSource(e.target.value)}>
            <option value="">{t(m.allSources)}</option>
            {d.cohortSources.map((s) => (
              <option key={s.key} value={s.key}>
                {sourceName(s)}
              </option>
            ))}
          </Select>
        }
      />
      <div className="sc-no-phone">
        <TableScroll label={t(m.cohortsTitle)}>
          <table className="tbl sc-t11">
            <thead>
              <tr>
                <th scope="col">{t(m.colJoinedIn)}</th>
                <th scope="col" className="r">
                  {t(m.colContacts)}
                </th>
                <th scope="col" className="r">
                  {t(m.colAfter30)}
                </th>
                <th scope="col" className="r">
                  {t(m.colAfter90)}
                </th>
              </tr>
            </thead>
            <tbody>
              {d.cohorts.length === 0 ? (
                <tr>
                  <td colSpan={4} className="k">
                    {t(m.cohortsNone)}
                  </td>
                </tr>
              ) : null}
              {d.cohorts.map((c) => (
                <tr key={c.month}>
                  <td>
                    <span className="font-semibold">{month(c)}</span>
                  </td>
                  <td className="r num">{count(c.contacts)}</td>
                  <td className="r">
                    <Retention r={c.after30} />
                  </td>
                  <td className="r">
                    <Retention r={c.after90} prev={c.after30} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      </div>
      <div className="sc-only-phone">
        {d.cohorts.map((c) => (
          <div key={c.month} className="gr-pcard">
            <span className="flex justify-between gap-2.5">
              <b className="font-semibold">{month(c)}</b>
              <span className="k">{t(m.cohortContacts, { count: c.contacts })}</span>
            </span>
            <span className="text-[12.5px]">
              <span className="k">{t(m.after30)}</span> <Retention r={c.after30} />
            </span>
            <span className="text-[12.5px]">
              <span className="k">{t(m.after90)}</span> <Retention r={c.after90} prev={c.after30} />
            </span>
          </div>
        ))}
      </div>
      <div className="sc-foot">
        <span className="k">
          {picked && firstJoinInRange && picked.firstJoin
            ? `${t(m.cohortsFirstJoin, { name: sourceName(picked), day: dayMonthDate(picked.firstJoin), start: dayMonthDate(new Date(new Date(picked.firstJoin).getTime() + 30 * 86_400_000)) })}${
                pickedRow && pickedRow.contacts < d.minContacts ? ` ${t(m.cohortsTooFew, { count: pickedRow.contacts })}` : ''
              }`
            : carried?.source.importedAt
              ? t(m.cohortsCarried, { day: dayMonthDate(carried.source.importedAt), name: carried.source.name })
              : null}
        </span>
      </div>
    </section>
  )
}

function UtmCard({ d }: { d: Data }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [tab, setTab] = useState<'utm' | 'ref'>('utm')
  const u = d.utm
  const r = d.referrers
  const since = u.captureStartedAt && u.captureStartedAt.slice(0, 10) > d.period.from ? u.captureStartedAt : undefined
  const when = since ? whenText(since, t) : ''
  type Line = { key: string; label: ReactNode; joins: number; rate?: number; search?: { utmSource?: string; utmMedium?: string; utmCampaign?: string; referringDomain?: string } }
  const utmLines: Line[] = [
    ...(u.notCaptured ? [{ key: 'nc', label: <span className="k text-[12.5px]">{t(m.notCaptured)}</span>, joins: u.notCaptured, rate: u.notCapturedHumanOpenRate }] : []),
    { key: 'none', label: <span className="k text-[12.5px]">{t(m.utmNone)}</span>, joins: u.none.joins, rate: u.none.humanOpenRate },
    ...u.rows.map((x) => ({
      key: `${x.source}/${x.medium}/${x.campaign}`,
      label: <span className="mono text-[12.5px]">{[x.source || '–', x.medium || '–', x.campaign || '–'].join(' / ')}</span>,
      joins: x.joins,
      rate: x.humanOpenRate,
      search: x.campaign ? { utmCampaign: x.campaign } : { utmSource: x.source, utmMedium: x.medium },
    })),
    ...(u.other.count ? [{ key: 'other', label: <span className="k text-[12.5px]">{t(m.otherCombinations, { count: u.other.count })}</span>, joins: u.other.joins, rate: u.other.humanOpenRate }] : []),
  ]
  const refLines: Line[] = [
    ...r.rows.map((x) => ({ key: x.page, label: <span className="mono text-[12.5px]">{x.page}</span>, joins: x.joins, search: { referringDomain: x.page.split('/')[0]! } })),
    { key: 'none', label: <span className="k text-[12.5px]">{t(m.noReferrer)}</span>, joins: r.none },
    ...(r.other.count ? [{ key: 'other', label: <span className="k text-[12.5px]">{t(m.otherPages, { count: r.other.count })}</span>, joins: r.other.joins }] : []),
  ]
  const lines = tab === 'utm' ? utmLines : refLines
  const max = Math.max(1, ...lines.map((l) => l.joins))
  const withRates = tab === 'utm'
  const label = (l: Line) =>
    l.search ? (
      <Link className="gr-rowlink" to="/w/$workspace/audience" params={{ workspace: slug }} search={l.search}>
        {l.label}
      </Link>
    ) : (
      l.label
    )
  return (
    <section className="card min-w-0 flex-[2_1_560px]" aria-labelledby="gr-utm">
      <CardSection
        id="gr-utm"
        title={t(m.utmTitle)}
        caption={since ? t(m.utmLeadSince, { count: u.captured, when }) : t(m.utmLead, { count: u.joins })}
      />
      <RTabs.Root value={tab} onValueChange={(v) => setTab(v as 'utm' | 'ref')}>
        <RTabs.List className="gr-pills" aria-label={t(m.utmTabs)}>
          <RTabs.Trigger value="utm">
            {t(m.utmTab)} <b>{utmLines.length - (u.notCaptured ? 1 : 0)}</b>
          </RTabs.Trigger>
          <RTabs.Trigger value="ref">
            {t(m.referrersTab)} <b>{refLines.length}</b>
          </RTabs.Trigger>
        </RTabs.List>
        <RTabs.Content value={tab} tabIndex={-1}>
          {since && tab === 'utm' ? (
            <div className="pad pb-0">
              <div className="note i" role="note">
                <span className="sc-note-ic" style={{ color: '#1D4ED8' }}>
                  <Icon name="info" size={16} strokeWidth={1.8} />
                </span>
                <div className="min-w-0 flex-[1_1_320px]">{t(m.utmCaptureNote, { when, day: dayMonthDate(since) })}</div>
              </div>
            </div>
          ) : null}
          <div className="sc-no-phone">
            <TableScroll label={withRates ? t(m.utmLabel) : t(m.referrersLabel)}>
              <table className="tbl sc-t11">
                <thead>
                  <tr>
                    <th scope="col">{withRates ? t(m.colUtm) : t(m.colReferrer)}</th>
                    <th scope="col">{t(m.colJoins)}</th>
                    {withRates ? (
                      <th scope="col" className="r">
                        {t(m.colHumanOpens)}
                      </th>
                    ) : null}
                  </tr>
                </thead>
                <tbody>
                  {lines.map((l) => (
                    <tr key={l.key}>
                      <td>{label(l)}</td>
                      <td>
                        <div className="gr-hbar">
                          <span className="num w-10 text-right font-semibold">{count(l.joins)}</span>
                          <div className="bar">
                            <i style={{ width: `${(100 * l.joins) / max}%`, background: 'var(--accent)' }} />
                          </div>
                        </div>
                      </td>
                      {withRates ? <td className="r num">{since && l.key !== 'nc' ? '–' : ratePct(l.rate, 1)}</td> : null}
                    </tr>
                  ))}
                </tbody>
              </table>
            </TableScroll>
          </div>
          <div className="sc-only-phone">
            {lines.map((l) => (
              <div key={l.key} className="gr-pcard gr-putm">
                {label(l)}
                <div className="gr-hbar">
                  <span className="num w-10 text-right font-semibold">{count(l.joins)}</span>
                  <div className="bar">
                    <i style={{ width: `${(100 * l.joins) / max}%`, background: 'var(--accent)' }} />
                  </div>
                </div>
                {withRates ? <span className="k">{t(m.humanOpensLine, { rate: since && l.key !== 'nc' ? '–' : ratePct(l.rate, 1) })}</span> : null}
              </div>
            ))}
          </div>
        </RTabs.Content>
      </RTabs.Root>
      <div className="sc-foot">
        <span className="k">{since && tab === 'utm' ? t(m.utmFootNotCaptured) : t(m.utmFoot)}</span>
      </div>
    </section>
  )
}

/** "today at 00:00", or the day. */
function whenText(at: string, t: ReturnType<typeof useT>): string {
  if (dayMonthYear(at) === dayMonthYear(new Date())) return t(m.todayAt, { time: clock(at) })
  return dayMonthDate(at)
}

function CountsCard({ d }: { d: Data }) {
  const t = useT()
  const imp = d.sources.find((r) => r.source.kind === 'import' && r.source.importCount && r.source.tookImportDay !== undefined)
  const items: [string, string][] = [
    [t(m.counts1), t(m.counts1Detail)],
    [
      t(m.counts2),
      imp
        ? t(m.counts2Detail, { name: imp.source.name, kept: (imp.source.importCount ?? 0) - (imp.source.tookImportDay ?? 0), took: imp.source.tookImportDay ?? 0 })
        : t(m.counts2Generic),
    ],
    [t(m.counts3), t(m.counts3Detail)],
    [t(m.counts4), t(m.counts4Detail)],
    [t(m.counts5), t(m.counts5Detail)],
  ]
  return (
    <section className="card min-w-0 flex-[1_1_340px]" aria-labelledby="gr-counts">
      <CardSection id="gr-counts" title={t(m.countsTitle)} />
      <div className="pad pt-1 pb-2">
        <ul className="gr-chk">
          {items.map(([a, b]) => (
            <li key={a}>
              <span className="flex-none" style={{ color: 'var(--accent)', marginTop: 1 }}>
                <Icon name="info" size={15} strokeWidth={1.8} />
              </span>
              <span className="flex min-w-0 flex-col gap-0.5">
                <span className="font-semibold">{a}</span>
                <span className="k" style={{ color: 'var(--ink-nav)' }}>
                  {b}
                </span>
              </span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

/** Only imports so far: moving a list isn't growth. */
function OnlyImports({ d }: { d: Data }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const imp = d.sources.find((r) => isImport(r.source))
  return (
    <section className="card sc-empty" aria-labelledby="gr-only">
      <h2 id="gr-only" className="tt">
        {t(m.onlyImportsTitle)}
      </h2>
      <p className="ds">
        {t(m.onlyImportsBody, {
          workspace: me.workspace.name,
          name: imp?.source.name ?? '',
          day: imp?.source.importedAt ? dayMonthDate(imp.source.importedAt) : '',
          count: imp?.contacts ?? d.tiles.imports,
        })}
      </p>
      <div className="flex flex-wrap justify-center gap-2.5">
        <ButtonLink variant="primary" to="/w/$workspace/forms" params={{ workspace: slug }}>
          {t(m.createForm)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/settings/keys" params={{ workspace: slug }}>
          {t(m.createKey)}
        </ButtonLink>
      </div>
      <span className="k">{t(m.onlyImportsHint)}</span>
    </section>
  )
}

function NoContacts() {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card sc-empty" aria-labelledby="gr-none">
      <h2 id="gr-none" className="tt">
        {t(m.noContactsTitle)}
      </h2>
      <p className="ds">{t(m.noContactsBody)}</p>
      <div className="flex flex-wrap justify-center gap-2.5">
        <ButtonLink variant="primary" to="/w/$workspace/import" params={{ workspace: slug }}>
          {t(m.importContacts)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/forms" params={{ workspace: slug }}>
          {t(m.createForm)}
        </ButtonLink>
      </div>
    </section>
  )
}

function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const ref = problemReference(error)
  return (
    <section className="card sc-loaderr" role="alert">
      <span style={{ color: 'var(--bad-strong)' }}>
        <Icon name="alertCircle" size={28} />
      </span>
      <h2 className="m-0 text-[16px] font-bold">{t(m.loadError)}</h2>
      <span className="k max-w-[440px]" style={{ color: 'var(--ink-2)' }}>
        {t(m.loadErrorBody, { detail: errorMessage(error) })}
      </span>
      <div className="flex flex-wrap justify-center gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace/audience/health" params={{ workspace: slug }}>
          {t(m.listHealth)}
        </ButtonLink>
      </div>
      {ref ? <span className="k mono">{t(m.requestRef, { ref })}</span> : null}
    </section>
  )
}

function GrowthSkeleton() {
  const t = useT()
  return (
    <div className="flex flex-col gap-4" aria-busy="true" aria-label={t(m.loading)}>
      <div className="gr-tiles">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="card gr-stat">
            <Skeleton w={70} h={12} r={6} />
            <Skeleton w={80} h={22} r={6} style={{ marginTop: 6 }} />
            <Skeleton w={150} h={10} r={6} style={{ marginTop: 6 }} />
          </div>
        ))}
      </div>
      <div className="card flex h-[220px] items-center justify-center">
        <span className="k">{t(m.loading)}</span>
      </div>
    </div>
  )
}
