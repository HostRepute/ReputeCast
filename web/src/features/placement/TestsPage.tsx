import { Link } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Th } from '@/components/ui/Table'
import { T, useT } from '@/i18n'
import { clock, orList, shortDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useNow } from '@/lib/hooks'
import { useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { PlacementShell } from './PlacementShell'
import type { RunPreset } from './RunDialog'
import {
  ContentFacts,
  dayTime,
  elapsed,
  LoadError,
  PBar,
  Pv,
  resultLine,
  SeedAttention,
  throughText,
  Through,
  usePlacementAccess,
  useTests,
  type TestList,
  type TestRow,
} from './shared'
import { TrendChart } from './TrendChart'

/** Deliverability › Placement tests. */
export function TestsPage({ preset }: { preset?: RunPreset }) {
  const t = useT()
  const q = useTests()
  return (
    <PlacementShell current="tests" documentTitle={t(m.documentTests)} preset={preset}>
      {q.isPending ? (
        <TestsLoading />
      ) : q.isError ? (
        <LoadError title={t(m.loadTestsError)} error={q.error} body={(status) => t(m.loadTestsBody, { status })} onRetry={() => void q.refetch()} />
      ) : (
        <TestsBody d={q.data} />
      )}
    </PlacementShell>
  )
}

function TestsBody({ d }: { d: TestList }) {
  const access = usePlacementAccess(d)
  if (d.tests.length === 0) {
    return (
      <>
        {access.canManage && d.attention.length ? <SeedAttention seeds={d.attention} /> : null}
        {access.readOnly ? <ReadOnlyNote helpers={d.runHelpers} /> : null}
        {d.seedsTotal === 0 ? <NoSeeds canManage={access.canManage} /> : <NoTests d={d} />}
      </>
    )
  }
  return (
    <>
      {access.canManage && d.attention.length ? <SeedAttention seeds={d.attention} /> : null}
      {access.readOnly ? <ReadOnlyNote helpers={d.runHelpers} /> : null}
      <TestsCard d={d} />
      <div className="flex flex-wrap items-start gap-4">
        <TrendChart tests={d.tests} providers={d.providers} seeds={d.seedsTotal} />
        <HowItWorks d={d} />
      </div>
    </>
  )
}

const running = (r: TestRow) => r.status === 'sending' || r.status === 'waiting'

function TestsCard({ d }: { d: TestList }) {
  const t = useT()
  const [q, setQ] = useState('')
  const [provider, setProvider] = useState('')
  const [by, setBy] = useState('')
  const now = useNow(d.tests.some(running) ? 1000 : 60_000)
  const shown = useMemo(() => {
    const needle = q.trim().toLowerCase()
    return d.tests.filter(
      (r) =>
        (!needle || r.contentName.toLowerCase().includes(needle) || r.name.toLowerCase().includes(needle)) &&
        (!provider || r.providers.some((p) => p.id === provider)) &&
        (!by || r.startedById === by),
    )
  }, [d.tests, q, provider, by])
  const filtered = shown.length !== d.tests.length
  const live = d.tests.find(running)
  const clear = () => {
    setQ('')
    setProvider('')
    setBy('')
  }
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.testsTitle, { count: d.tests.length })}</h2>
          <span className="k">{live ? <RunningCaption row={live} now={now} /> : t(m.testsLead)}</span>
        </div>
      </div>
      <div className="pl-filters">
        <label className="pl-search">
          <Icon name="search" size={15} className="text-ink-3" />
          <input aria-label={t(m.searchLabel)} placeholder={t(m.searchPlaceholder)} value={q} onChange={(e) => setQ(e.target.value)} />
        </label>
        <select className="in" style={{ width: 170 }} aria-label={t(m.providerFilter)} value={provider} onChange={(e) => setProvider(e.target.value)}>
          <option value="">{t(m.allProviders)}</option>
          {d.providers.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
        <select className="in" style={{ width: 190 }} aria-label={t(m.startedByFilter)} value={by} onChange={(e) => setBy(e.target.value)}>
          <option value="">{t(m.startedByAnyone)}</option>
          {d.starters.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>
      {shown.length === 0 ? (
        <div className="flex flex-wrap items-center gap-3 px-[18px] py-5 text-[13px] text-ink-2" role="status">
          <span>{t(m.noMatches)}</span>
          <Button size="sm" onClick={clear}>
            {t(m.clearFilters)}
          </Button>
        </div>
      ) : (
        <>
          <div className="pl-lg-only">
            <div className="relative overflow-x-auto">
              <table className="tbl" aria-label={t(m.testsTableLabel)}>
                <thead>
                  <tr>
                    <Th>{t(m.colStarted)}</Th>
                    <Th>{t(m.colContent)}</Th>
                    <Th>{t(m.colThrough)}</Th>
                    <Th>{t(m.colBy)}</Th>
                    <Th>{t(m.colResult)}</Th>
                    <Th>
                      <span className="vh">{t(m.view)}</span>
                    </Th>
                  </tr>
                </thead>
                <tbody>
                  {shown.map((r) => (
                    <TestTableRow key={r.id} row={r} now={now} />
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          <div className="pl-md-down">
            {shown.map((r) => (
              <TestListItem key={r.id} row={r} now={now} />
            ))}
          </div>
        </>
      )}
      <div className="pl-foot">
        <span className="k">{live ? t(m.liveCaption) : t(d.deleteAfter === 'never' ? m.footDeletedNever : d.deleteAfter === 'finished' ? m.footDeletedFinished : m.footDeleted24h)}</span>
        <span className="k">{filtered ? t(m.showing, { shown: shown.length, total: d.tests.length }) : t(m.resultsLegend)}</span>
      </div>
    </section>
  )
}

function RunningCaption({ row, now }: { row: TestRow; now: Date }) {
  const t = useT()
  const p = row.providers.map((x) => x.name)
  return <>{t(m.runningCaption, { time: dayTime(now.toISOString()), name: row.startedBy, content: row.contentKey || row.contentName, provider: orList(p) })}</>
}

function TestLink({ row, className, children }: { row: TestRow; className?: string; children: ReactNode }) {
  const { slug } = useWorkspace()
  return (
    <Link to="/w/$workspace/deliverability/placement/$testId" params={{ workspace: slug, testId: row.id }} className={className}>
      {children}
    </Link>
  )
}

function TestTableRow({ row, now }: { row: TestRow; now: Date }) {
  const t = useT()
  const { slug } = useWorkspace()
  const live = running(row)
  return (
    <tr style={live ? { background: 'var(--accent-tint)' } : undefined}>
      <td className="nw">{dayTime(row.startedAt)}</td>
      <td>
        <div className="flex min-w-0 flex-col gap-0.5">
          <TestLink row={row} className="font-semibold text-ink! no-underline hover:underline">
            {row.contentName}
          </TestLink>
          <span className="k">
            <ContentFacts row={row} />
          </span>
        </div>
      </td>
      <td>
        <Through row={row} />
      </td>
      <td>
        <span className="inline-flex items-center gap-2 whitespace-nowrap">
          <Avatar name={row.startedBy} color={row.startedByColor} size={22} fontSize={11} />
          <span>{row.startedBy}</span>
        </span>
      </td>
      <td>
        <Results row={row} now={now} />
      </td>
      <td className="r">
        <ButtonLink size="sm" to="/w/$workspace/deliverability/placement/$testId" params={{ workspace: slug, testId: row.id }} aria-label={t(m.viewTest, { name: row.contentName })}>
          {t(m.view)}
        </ButtonLink>
      </td>
    </tr>
  )
}

/** Inbox, tab, spam and not arrived per provider; found so far while it runs. */
function Results({ row, now }: { row: TestRow; now: Date }) {
  const t = useT()
  const live = running(row)
  const res = (
    <div className="pl-mres">
      {row.providers.map((p) => (
        <div className="ln" key={p.id}>
          <Pv adapter={p.adapter}>{p.short}</Pv>
          <PBar counts={p.counts} waiting={live} />
          <span className="ct">{live ? t(m.foundSoFar, { found: p.counts.found, copies: p.counts.copies, elapsed: elapsed(row.startedAt, now.getTime()) }) : resultLine(p.counts)}</span>
        </div>
      ))}
    </div>
  )
  if (!live) return res
  return (
    <div className="flex flex-col gap-1.5">
      {res}
      {row.waitUntil ? (
        <span className="pl-live text-[12px]!">
          <span className="dot" aria-hidden="true" />
          {t(m.liveLine, { time: clock(row.waitUntil) })}
        </span>
      ) : null}
    </div>
  )
}

function TestListItem({ row, now }: { row: TestRow; now: Date }) {
  return (
    <TestLink row={row} className="flex! flex-col items-stretch! gap-2 border-t border-line-2 px-4 py-3 text-ink! no-underline">
      <span className="flex justify-between gap-2">
        <b className="font-semibold">{row.contentName}</b>
        <span className="k">{shortDate(row.startedAt)}</span>
      </span>
      <span className="k">
        {throughText(row)} · {row.startedBy}
      </span>
      <Results row={row} now={now} />
    </TestLink>
  )
}

function ReadOnlyNote({ helpers }: { helpers: string[] }) {
  const t = useT()
  const me = useMe().data
  const { me: ws } = useWorkspace()
  return (
    <div className="note n" role="note">
      <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
      <div className="flex min-w-0 flex-col gap-1">
        {me ? <span className="font-bold">{t(m.readOnlyWho, { name: me.user.name, role: roleName({ key: ws.workspace.roleKey, name: ws.workspace.roleName }) })}</span> : null}
        <span>
          <T msg={helpers.length ? m.viewerNote : m.viewerNoteNobody} values={{ helpers: orList(helpers), code: (c) => <span className="mono">{c}</span> }} />
        </span>
      </div>
    </div>
  )
}

function Envelopes() {
  return (
    <svg width="72" height="56" viewBox="0 0 72 56" aria-hidden="true">
      <rect x="4" y="12" width="30" height="22" rx="3" fill="none" stroke="#9AA3B2" strokeWidth="2" />
      <path d="m5 14 14 10 14-10" fill="none" stroke="#9AA3B2" strokeWidth="2" />
      <rect x="38" y="22" width="30" height="22" rx="3" fill="none" stroke="var(--accent)" strokeWidth="2" />
      <path d="m39 24 14 10 14-10" fill="none" stroke="var(--accent)" strokeWidth="2" />
    </svg>
  )
}

function NoTests({ d }: { d: TestList }) {
  const t = useT()
  const { slug } = useWorkspace()
  const access = usePlacementAccess(d)
  return (
    <section className="card pl-empty">
      <Envelopes />
      <h2 className="tt">{t(m.emptyTitle)}</h2>
      <span className="ds">{t(m.emptyBody, { count: d.seedsConnected })}</span>
      <div className="flex flex-wrap justify-center gap-2.5">
        {access.readOnly ? null : access.canRun ? (
          <ButtonLink variant="primary" className="min-h-10" to="/w/$workspace/deliverability/placement" params={{ workspace: slug }} search={{ run: 'placement' }}>
            {t(m.runTest)}
          </ButtonLink>
        ) : (
          <Button variant="primary" className="min-h-10" disabledReason={access.runReason}>
            {t(m.runTest)}
          </Button>
        )}
        <ButtonLink to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }}>
          {t(m.seedMailboxes)}
        </ButtonLink>
      </div>
      <span className="k">{t(m.emptyHint)}</span>
    </section>
  )
}

function NoSeeds({ canManage }: { canManage: boolean }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  return (
    <section className="card pl-empty">
      <Envelopes />
      <h2 className="tt">{t(m.noSeedsTitle)}</h2>
      <span className="ds">{t(m.noSeedsBody)}</span>
      <div className="flex flex-wrap justify-center gap-2.5">
        {canManage ? (
          <ButtonLink variant="primary" className="min-h-10" to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }} search={{ add: true }}>
            {t(m.addSeed)}
          </ButtonLink>
        ) : null}
        <ButtonLink to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }} hash="how-seeds-connect">
          {t(m.howSeedsConnect)}
        </ButtonLink>
      </div>
      <span className="k">{t(m.noSeedsWorkspace, { workspace: me.workspace.name })}</span>
    </section>
  )
}

const HOW: [IconName, keyof typeof m][] = [
  ['mail', 'how1'],
  ['check', 'how2'],
  ['clock', 'how3'],
  ['shield', 'how4'],
]

function HowItWorks({ d }: { d: TestList }) {
  const t = useT()
  const { slug } = useWorkspace()
  const items: [IconName, string][] = [
    ...HOW.map(([icon, key]) => [icon, t(m[key])] as [IconName, string]),
    ['trash', t(d.deleteAfter === 'never' ? m.how5DeletedNever : d.deleteAfter === 'finished' ? m.how5DeletedFinished : m.how5Deleted24h)],
  ]
  return (
    <section className="card min-w-0 flex-[1_1_320px]">
      <div className="sec">
        <h2 className="h2">{t(m.howTitle)}</h2>
      </div>
      <ul className="pad pl-how m-0 list-none pt-1.5! pb-2.5!">
        {items.map(([icon, text], i) => (
          <li key={i} className="flex items-start gap-2.5 py-2.5 text-[13px] text-ink-nav" style={{ borderTop: i ? '1px solid var(--line-2)' : 0 }}>
            <Icon name={icon} size={16} className="mt-px flex-none text-accent" />
            <span>{text}</span>
          </li>
        ))}
      </ul>
      <div className="pl-foot">
        <span className="k flex-[1_1_200px]">{t(m.howFoot, { count: d.seedsTotal })}</span>
        {d.canManage ? (
          <Link to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }} hash="deletion" className="pl-lnk">
            {t(m.deletionSettings)}
          </Link>
        ) : null}
      </div>
    </section>
  )
}

function TestsLoading() {
  const t = useT()
  return (
    <Loading label={t(m.loading)} className="card flex flex-col gap-3.5 p-[18px]">
      <span className="k" aria-hidden="true">
        {t(m.loading)}
      </span>
      <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(200px,100%),1fr))' }}>
        {[0, 1].map((i) => (
          <div key={i} className="card flex flex-col gap-2 p-3.5">
            <Skeleton w={110} h={12} />
            <Skeleton w="100%" h={20} r={6} />
            <Skeleton w={140} h={10} />
          </div>
        ))}
      </div>
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="flex items-center gap-3.5 border-t border-line-2 py-2">
          <Skeleton w={130} h={12} />
          <Skeleton w={110} h={20} r={999} />
          <Skeleton h={8} className="flex-1" />
        </div>
      ))}
    </Loading>
  )
}

