import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Icon } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { TableScroll, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, count, dayMonthDate, orList, shortDate, weekdayDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { ConnectDialog, DisconnectDialog, ReplaceKeyDialog } from './ServiceDialogs'
import {
  base,
  LoadError,
  money,
  Notice,
  ResultBadge,
  resultsLine,
  ServiceTile,
  StatusBadge,
  tagValue,
  useOverview,
  useRealtime,
  useRefreshVerification,
  when,
  type Overview,
  type ServiceKey,
  type ServiceView,
  type Use,
} from './shared'

export type ServicesSearch = { service?: ServiceKey; connect?: boolean }

/** Members with integrations:manage are told when a balance falls under it. */
const LOW_CREDITS = 1000

/** Settings › Integrations › Verification services. */
export function ServicesPage({ search }: { search: ServicesSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useOverview()
  return (
    <Page
      title={t(m.title)}
      documentTitle={t(m.documentTitle)}
      breadcrumb={[
        { label: t(m.settings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(m.integrations), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: t(m.title) },
      ]}
    >
      {q.isPending ? (
        <ServicesLoading />
      ) : q.isError ? (
        <LoadError title={t(m.loadError)} error={q.error} body={(status) => t(m.loadErrorBody, { status })} onRetry={() => void q.refetch()} />
      ) : (
        <Services o={q.data} search={search} />
      )}
    </Page>
  )
}

type Dlg = { kind: 'connect' | 'replace' | 'disconnect'; v: ServiceView }

function Services({ o, search }: { o: Overview; search: ServicesSearch }) {
  const t = useT()
  const { slug, me: ws } = useWorkspace()
  const me = useMe().data
  const navigate = useNavigate()
  const connected = o.services.filter((s) => s.connected)
  const selectedKey = search.service ?? (o.defaultBulk as ServiceKey | undefined) ?? connected[0]?.key
  const selected = o.services.find((s) => s.key === selectedKey)
  const [dlg, setDlg] = useState<Dlg | undefined>(() => {
    const v = o.services.find((s) => s.key === search.service)
    return search.connect && v && !v.connected && o.canManage ? { kind: 'connect', v } : undefined
  })
  const select = (key: ServiceKey) => void navigate({ to: '/w/$workspace/settings/integrations/verification', params: { workspace: slug }, search: { service: key }, replace: true, resetScroll: false })
  const nameOf = (key?: string) => o.services.find((s) => s.key === key)?.name
  return (
    <>
      <div className="row items-end! justify-between gap-x-5 gap-y-3">
        <div className="flex min-w-0 flex-[1_1_520px] flex-col gap-1">
          <h2 className="ih-ttl">{t(m.title)}</h2>
          <p className="ih-lead">{t(m.lead)}</p>
        </div>
        <div className="card flex flex-[0_1_auto] flex-col gap-1.5 px-3.5 py-2.5">
          <span className="k">{t(m.defaults)}</span>
          <div className="row gap-1.5!">
            {o.defaultBulk ? <span className="vf-pill">{t(m.defaultBulkPill, { name: nameOf(o.defaultBulk) ?? '' })}</span> : null}
            {o.defaultForms ? <span className="vf-pill">{t(m.defaultFormsPill, { name: nameOf(o.defaultForms) ?? '' })}</span> : null}
            {!o.defaultBulk && !o.defaultForms ? <span className="vf-pill">{t(m.noDefaults)}</span> : null}
          </div>
        </div>
      </div>
      {connected.length === 0 ? (
        <div className="note n" role="note">
          <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
          <div className="flex min-w-0 flex-col gap-1">
            <b className="font-bold">{t(m.nothingConnectedTitle, { workspace: ws.workspace.name, name: me?.user.name ?? '' })}</b>
            <span>{t(m.nothingConnected)}</span>
          </div>
        </div>
      ) : null}
      {!o.canManage ? (
        <div className="note n" role="note">
          <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
          <div className="flex min-w-0 flex-col gap-1">
            <b className="font-bold">{t(m.readOnlyTitle, { name: me?.user.name ?? '', role: roleName({ key: ws.workspace.roleKey, name: ws.workspace.roleName }) })}</b>
            <span>
              <T msg={o.manageHelpers.length ? m.readOnly : m.readOnlyNobody} values={{ helpers: orList(o.manageHelpers), code: (c) => <span className="mono">{c}</span> }} />
            </span>
          </div>
        </div>
      ) : null}
      <div className="ih-grid" role="list" aria-label={t(m.servicesLabel)}>
        {o.services.map((v) => (
          <ServiceCard
            key={v.key}
            v={v}
            selected={v.connected && v.key === selected?.key}
            canManage={o.canManage}
            onManage={() => select(v.key)}
            onConnect={() => setDlg({ kind: 'connect', v })}
          />
        ))}
      </div>
      {selected?.connected ? <Detail o={o} v={selected} onDialog={(kind) => setDlg({ kind, v: selected })} /> : null}
      {dlg?.kind === 'connect' ? (
        <ConnectDialog v={dlg.v} open onOpenChange={(x) => !x && setDlg(undefined)} otherForms={o.defaultForms && o.defaultForms !== dlg.v.key ? nameOf(o.defaultForms) : undefined} otherBulk={o.defaultBulk && o.defaultBulk !== dlg.v.key ? nameOf(o.defaultBulk) : undefined} />
      ) : null}
      {dlg?.kind === 'replace' ? <ReplaceKeyDialog v={dlg.v} open onOpenChange={(x) => !x && setDlg(undefined)} /> : null}
      {dlg?.kind === 'disconnect' ? <DisconnectFor v={dlg.v} onClose={() => setDlg(undefined)} /> : null}
    </>
  )
}

function DisconnectFor({ v, onClose }: { v: ServiceView; onClose: () => void }) {
  const t = useT()
  const rt = useRealtime(v.key, v.defaultForms)
  const forms = (rt.data?.forms ?? []).map((f) => (f.formKey === 'subscribe_page' ? t(m.theSubscribePage) : f.name))
  return <DisconnectDialog v={v} open onOpenChange={(x) => !x && onClose()} forms={forms} />
}

function capabilityList(v: ServiceView, t: ReturnType<typeof useT>): string[] {
  const c = v.capabilities
  return [c.bulkLists ? t(m.capBulk) : '', c.realTime ? t(m.capRealtime) : '', c.catchAll ? t(m.capCatchAll) : '', c.creditBalance ? t(m.capBalance) : '', c.dataRegion ? t(m.capRegion) : ''].filter(Boolean)
}

function ServiceCard({ v, selected, canManage, onManage, onConnect }: { v: ServiceView; selected: boolean; canManage: boolean; onManage: () => void; onConnect: () => void }) {
  const t = useT()
  const role = v.defaultBulk && v.defaultForms ? t(m.defaultForBoth) : v.defaultBulk ? t(m.defaultForBulk) : v.defaultForms ? t(m.defaultForForms) : t(m.ownAccount)
  return (
    <article className={selected ? 'card ih-card vf-sel' : 'card ih-card'} aria-label={v.name} aria-current={selected || undefined} role="listitem">
      <div className="ih-head">
        <ServiceTile initials={v.initials} />
        <div className="ih-name">
          <h3>{v.name}</h3>
          <span className="k">{v.connected ? role : t(m.ownAccount)}</span>
        </div>
        <StatusBadge v={v} />
      </div>
      <p className="ih-desc">{t(m.cardLine, { name: v.name })}</p>
      <div className="ih-caps">
        {capabilityList(v, t).map((c) => (
          <span key={c} className="ih-cap">
            {c}
          </span>
        ))}
      </div>
      <div className="ih-ft">
        <span className="k">{v.connected ? (v.balance != null ? t(m.cardFact, { credits: v.balance, price: money(v.price, true) }) : t(m.cardFactNoBalance, { price: money(v.price, true) })) : t(m.payPerCheck)}</span>
        {v.connected ? (
          <Button size="sm" onClick={onManage} aria-pressed={selected}>
            {canManage ? t(m.manage) : t(m.view)}
          </Button>
        ) : canManage ? (
          <Button size="sm" variant="primary" onClick={onConnect}>
            {t(m.connect)}
          </Button>
        ) : null}
      </div>
    </article>
  )
}

function Stat({ label, value, sub }: { label: string; value: ReactNode; sub: ReactNode }) {
  return (
    <div className="card ih-stat">
      <span className="k">{label}</span>
      <span className="ih-v">{value}</span>
      <span className="ih-s">{sub}</span>
    </div>
  )
}

function useTest(v: ServiceView) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshVerification()
  const [busy, setBusy] = useState(false)
  const run = async () => {
    setBusy(true)
    const res = await http.POST(`${base}/services/{service}/test`, { params: { path: { workspace: slug, service: v.key } } })
    setBusy(false)
    void refresh()
    if (!res.data) {
      toast({ kind: 'error', title: errorMessage(res.error) })
      return
    }
    toast(res.data.ok ? { kind: 'success', title: t(m.testPassed, { when: when(res.data.at) }) } : { kind: 'error', title: res.data.problem ?? t(m.testFailed, { time: clock(res.data.at) }) })
  }
  return { busy, run }
}

function Detail({ o, v, onDialog }: { o: Overview; v: ServiceView; onDialog: (k: 'replace' | 'disconnect') => void }) {
  const t = useT()
  const test = useTest(v)
  const rt = useRealtime(v.key, v.defaultForms)
  const facts = [
    v.connectedBy && v.connectedAt ? t(m.connectedBy, { name: v.connectedBy, date: weekdayDate(v.connectedAt) }) : '',
    v.regionLabel,
    v.defaultBulk ? t(m.defaultForBulkLong) : '',
    v.defaultForms ? t(m.defaultForFormsLong) : '',
  ].filter(Boolean)
  const uses = o.uses.filter((u) => u.service === v.key && u.kind !== 'realtime')
  return (
    <>
      <div className="vf-section">
        <span className="nm">{v.name}</span>
        <span className="k">{t(m.selectedAbove)}</span>
        <span className="ln" aria-hidden="true" />
      </div>
      <div className="flex min-w-0 flex-col gap-4">
        <section className="card">
          <div className="flex flex-wrap items-start gap-3.5 px-5 py-[18px]">
            <ServiceTile initials={v.initials} size="lg" />
            <div className="flex min-w-0 flex-[1_1_300px] flex-col gap-1">
              <div className="row gap-2.5!">
                <h2 className="m-0 text-[19px] font-bold tracking-[-0.01em]">{v.name}</h2>
                <StatusBadge v={v} />
              </div>
              <span className="k text-ink-2!">{facts.join(' · ')}</span>
            </div>
            {o.canManage ? (
              <div className="flex flex-wrap items-center gap-2">
                <Button size="sm" icon="refresh" onClick={() => void test.run()} loading={test.busy} busyLabel={t(m.testing)}>
                  {t(m.testConnection)}
                </Button>
                <ButtonAnchor size="sm" href={v.website} iconRight="external">
                  {t(m.openService, { name: v.name })}
                  <span className="vh">{t(m.newTab)}</span>
                </ButtonAnchor>
              </div>
            ) : null}
          </div>
        </section>
        {v.status === 'low_credits' ? <LowCredits v={v} onTest={() => void test.run()} /> : null}
        {v.status === 'error' ? <ServiceError v={v} onTest={() => void test.run()} canManage={o.canManage} /> : null}
        {v.defaultForms && rt.data ? <RealtimeCard v={v} rt={rt.data} /> : <Stats v={v} canManage={o.canManage} />}
        {o.canManage ? (
          <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(460px,100%),1fr))' }}>
            <SettingsCard o={o} v={v} onReplace={() => onDialog('replace')} onTest={() => void test.run()} testing={test.busy} />
            <div className="flex min-w-0 flex-col gap-4">
              <Health v={v} />
              <Receives v={v} />
            </div>
          </div>
        ) : null}
        <MapCard v={v} />
        <UsesCard v={v} uses={uses} canManage={o.canManage} />
        {o.canManage ? null : <Receives v={v} />}
        {o.canManage ? (
          <section className="card vf-dz">
            <div className="flex min-w-0 flex-[1_1_360px] flex-col gap-0.5">
              <b className="font-semibold">{t(m.disconnectTitle, { name: v.name })}</b>
              <span className="k text-ink-2!">
                {v.defaultBulk && v.defaultForms
                  ? t(m.disconnectBoth, { name: v.name })
                  : v.defaultForms
                    ? t(m.disconnectForms)
                    : v.defaultBulk
                      ? t(m.disconnectBulk, { name: v.name })
                      : t(m.disconnectPlain, { name: v.name })}
              </span>
            </div>
            <Button variant="danger" onClick={() => onDialog('disconnect')}>
              {t(m.disconnectButton, { name: v.name })}
            </Button>
          </section>
        ) : null}
      </div>
    </>
  )
}

function Stats({ v, canManage }: { v: ServiceView; canManage: boolean }) {
  const t = useT()
  const last = v.lastUsedAt
  return (
    <div className="ih-stats">
      <Stat
        label={t(m.statBalance)}
        value={count(v.balance)}
        sub={v.balanceAt ? t(m.statBalanceSub, { when: when(v.balanceAt) }) : t(m.hBalanceNone, { name: v.name })}
      />
      <Stat label={t(m.statChecks)} value={count(v.checks30d)} sub={t(m.statChecksSub, { count: v.bulkUses30d })} />
      {canManage ? <Stat label={t(m.statSpent)} value={money(v.spent30d)} sub={t(m.statSpentSub, { price: money(v.price, true) })} /> : null}
      <Stat
        label={t(m.statLastUsed)}
        value={last ? clock(last) : '—'}
        sub={last ? t(m.statLastUsedSub, { when: shortDate(last) === shortDate(new Date()) ? t(m.today) : shortDate(last), name: v.lastUsedBy ?? v.lastUsedWhat ?? '' }) : t(m.statLastUsedNever)}
      />
    </div>
  )
}

function RealtimeCard({ v, rt }: { v: ServiceView; rt: NonNullable<ReturnType<typeof useRealtime>['data']> }) {
  const t = useT()
  const counts = rt.days.map((d) => d.checks)
  const min = counts.length ? Math.min(...counts) : 0
  const max = counts.length ? Math.max(...counts) : 0
  const from = rt.days[0]?.day
  const to = rt.days[rt.days.length - 1]?.day
  const since = rt.since ? shortDate(rt.since) : ''
  return (
    <>
      <div className="ih-stats">
        <Stat label={t(m.statRealtime)} value={count(rt.checks)} sub={t(m.statRealtimeSub, { date: since })} />
        <Stat label={t(m.statSpentRealtime)} value={money(rt.spent)} sub={t(m.statSpentSub, { price: money(v.price, true) })} />
        <Stat label={t(m.statAskedToFix)} value={count(rt.askedToFix)} sub={t(m.statAskedToFixSub)} />
      </div>
      {counts.length > 1 && from && to ? (
        <div className="row gap-3! text-[13px]">
          <span className="font-semibold">{t(m.perDay, { from: dayMonthDate(from), to: dayMonthDate(to) })}</span>
          <DaySpark values={counts} />
          <span className="k">{t(m.perDayRange, { min, max })}</span>
        </div>
      ) : null}
      <section className="card overflow-hidden">
        <TableScroll label={t(m.formsTableLabel)}>
          <table className="tbl" aria-label={t(m.formsTableLabel)}>
            <thead>
              <tr>
                <Th>{t(m.colForm)}</Th>
                <Th align="right">
                  {t(m.colChecks)}
                </Th>
                <Th align="right">
                  {t(m.colValid)}
                </Th>
                <Th align="right">
                  {t(m.colInvalidFix)}
                </Th>
                <Th align="right">
                  {t(m.colRisky)}
                </Th>
                <Th align="right">
                  {t(m.colUnknown)}
                </Th>
                <Th align="right">
                  {t(m.colNoAnswer)}
                </Th>
              </tr>
            </thead>
            <tbody>
              {[...rt.forms, rt.total].map((f, i) => {
                const total = i === rt.forms.length
                return (
                  <tr key={f.formKey || 'total'}>
                    <td className="font-semibold">{total ? t(m.totalSince, { date: since }) : f.name}</td>
                    <td className={total ? 'r font-semibold' : 'r'}>{count(f.checks)}</td>
                    <td className="r">{count(f.valid)}</td>
                    <td className="r">{count(f.invalid)}</td>
                    <td className="r">{count(f.risky)}</td>
                    <td className="r">{count(f.unknown)}</td>
                    <td className="r">{!total && f.noAnswer ? t(m.noAnswerLater, { count: f.noAnswer }) : count(f.noAnswer)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </TableScroll>
        <div className="ih-foot border-t border-line-2 px-[18px] py-3">
          <span className="k">{t(m.realtimeFoot)}</span>
        </div>
      </section>
    </>
  )
}

/** The board's small per-day line: 160 × 30, a light area under it. */
function DaySpark({ values }: { values: number[] }) {
  const min = Math.min(...values)
  const max = Math.max(...values)
  const span = max - min || 1
  const pts = values.map((v, i) => [1 + (i / Math.max(1, values.length - 1)) * 158, 27 - ((v - min) / span) * 24] as const)
  const line = pts.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ')
  return (
    <svg width="160" height="30" viewBox="0 0 160 30" aria-hidden="true" className="flex-none">
      <path d={`M1.0,30 L${pts.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' L')} L159.0,30 Z`} fill="var(--accent)" opacity=".10" />
      <polyline points={line} fill="none" stroke="var(--accent)" strokeWidth="1.6" strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  )
}

function LowCredits({ v, onTest }: { v: ServiceView; onTest: () => void }) {
  const t = useT()
  return (
    <section className="card flex flex-col gap-3 px-[18px] py-4">
      <div className="note w" role="status">
        <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
        <div className="flex min-w-0 flex-col gap-1">
          <b className="font-bold">{t(m.lowCreditsTitle)}</b>
          <span>
            {v.needed && v.neededLabel ? t(m.lowCreditsBody, { balance: v.balance ?? 0, label: v.neededLabel, needed: v.needed }) : t(m.lowCreditsBodyPlain, { balance: v.balance ?? 0, limit: LOW_CREDITS })}
          </span>
          <div className="mt-1.5 flex flex-wrap gap-2">
            <ButtonAnchor size="sm" href={v.website} iconRight="external">
              {t(m.addCredits, { name: v.name })}
              <span className="vh">{t(m.newTab)}</span>
            </ButtonAnchor>
            <Button size="sm" onClick={onTest}>
              {t(m.readBalanceAgain)}
            </Button>
          </div>
        </div>
      </div>
      <span className="k">
        <T msg={m.lowCreditsFoot} values={{ limit: LOW_CREDITS, code: (c) => <span className="mono">{c}</span> }} />
      </span>
    </section>
  )
}

function ServiceError({ v, onTest, canManage }: { v: ServiceView; onTest: () => void; canManage: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-col gap-3 px-[18px] py-4" style={{ borderColor: 'var(--bad-line)' }}>
      <div className="note b" role="alert">
        <Icon name="alertCircle" size={16} className="mt-px flex-none" />
        <div className="flex min-w-0 flex-col gap-1">
          <b className="font-bold">{t(m.lastErrorTitle, { when: v.lastErrorAt ? when(v.lastErrorAt) : '' })}</b>
          <span>
            {v.lastError} {v.pausedJob ? t(m.lastErrorJob) : null}
          </span>
          {canManage ? (
            <div className="mt-1.5 flex flex-wrap gap-2">
              <Button size="sm" onClick={onTest}>
                {t(m.testConnection)}
              </Button>
              {v.pausedJob ? (
                <ButtonLink size="sm" to="/w/$workspace/audience/verification/$jobId" params={{ workspace: slug, jobId: v.pausedJob }}>
                  {t(m.openPausedJob)}
                </ButtonLink>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
      <dl className="kv">
        {v.balance != null ? (
          <>
            <dt>{t(m.hBalance)}</dt>
            <dd>{t(m.balanceLastRead, { balance: v.balance, time: v.balanceAt ? clock(v.balanceAt) : '' })}</dd>
          </>
        ) : null}
        {v.lastSuccessAt ? (
          <>
            <dt>{t(m.lastSuccess)}</dt>
            <dd>{t(m.lastSuccessLine, { time: clock(v.lastSuccessAt), what: v.lastSuccessWhat ?? '' })}</dd>
          </>
        ) : null}
      </dl>
    </section>
  )
}

function SettingsCard({ o, v, onReplace, onTest, testing }: { o: Overview; v: ServiceView; onReplace: () => void; onTest: () => void; testing: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshVerification()
  const [price, setPrice] = useState(String(v.price))
  const [bulk, setBulk] = useState(v.defaultBulk)
  const [forms, setForms] = useState(v.defaultForms)
  const [busy, setBusy] = useState(false)
  const otherForms = o.defaultForms && o.defaultForms !== v.key ? o.services.find((s) => s.key === o.defaultForms)?.name : undefined
  const otherBulk = o.defaultBulk && o.defaultBulk !== v.key ? o.services.find((s) => s.key === o.defaultBulk)?.name : undefined
  const save = async () => {
    setBusy(true)
    const res = await http.PATCH(`${base}/services/{service}`, {
      params: { path: { workspace: slug, service: v.key } },
      body: { price: Number(price) || 0, defaultBulk: bulk, defaultForms: forms, region: (v.region || undefined) as 'us' | undefined },
    })
    setBusy(false)
    if (!res.data) {
      toast({ kind: 'error', title: errorMessage(res.error), retry: () => void save() })
      return
    }
    void refresh()
    toast({ kind: 'success', title: t(m.saved) })
  }
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(m.settingsTitle)}</h2>
      </div>
      <div className="vf-srow border-t-0!">
        <div className="what">
          <span className="font-semibold">{t(m.priceTitle)}</span>
          <span className="k text-ink-2!">{v.priceBy ? t(m.priceBody, { name: v.name, by: v.priceBy }) : t(m.priceBodyNobody, { name: v.name })}</span>
        </div>
        <div className="ctl">
          <span className="k text-[13px]!">$</span>
          <input className="in w-24" inputMode="decimal" value={price} onChange={(e) => setPrice(e.target.value.replace(/[^\d.]/g, ''))} aria-label={t(m.priceLabel)} />
          <span className="k whitespace-nowrap">{t(m.aCheck)}</span>
        </div>
      </div>
      <div className="vf-srow">
        <div className="what">
          <span className="font-semibold">{t(m.defaultTitle)}</span>
          <span className="k text-ink-2!">{t(m.defaultBody, { name: v.name })}</span>
        </div>
        <div className="ctl">
          <div className="flex flex-col gap-2">
            <label className="flex items-start gap-2 text-[13px]">
              <input type="checkbox" className="mt-0.5" checked={bulk} onChange={(e) => setBulk(e.target.checked)} />
              <span className="flex flex-col gap-px">
                <span>{t(m.bulkAndImports)}</span>
                {otherBulk && !bulk ? <span className="k">{t(m.otherDefaultBulk, { name: otherBulk })}</span> : null}
              </span>
            </label>
            <label className="flex items-start gap-2 text-[13px]">
              <input type="checkbox" className="mt-0.5" checked={forms} onChange={(e) => setForms(e.target.checked)} />
              <span className="flex flex-col gap-px">
                <span>{t(m.signupForms)}</span>
                {otherForms && !forms ? <span className="k">{t(m.otherDefault, { name: otherForms })}</span> : null}
              </span>
            </label>
          </div>
        </div>
      </div>
      <div className="vf-srow">
        <div className="what">
          <span className="font-semibold">{t(m.keyTitle)}</span>
          <span className="k text-ink-2!">{t(m.keyBody)}</span>
        </div>
        <div className="ctl">
          <span className="vf-mask">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#5B6472" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M2 2l12 12M6.6 6.6a2 2 0 0 0 2.8 2.8M4.2 4.3C2.9 5.2 2 6.5 1.5 8c1.1 2.9 3.6 4.5 6.5 4.5 1.3 0 2.5-.3 3.6-.9M8 3.5c2.9 0 5.4 1.6 6.5 4.5-.3.9-.8 1.7-1.4 2.4" />
            </svg>
            <span aria-hidden="true">••••••</span>
            <span className="badge mut h-5! text-[11px]!">{t(m.writeOnly)}</span>
          </span>
          <Button size="sm" onClick={onReplace}>
            {t(m.replaceKey)}
          </Button>
        </div>
      </div>
      <div className="vf-srow">
        <div className="what">
          <span className="font-semibold">{t(m.connectionTitle)}</span>
          <span className="k text-ink-2!">{t(m.connectionBody)}</span>
          <div className="mt-2.5 flex flex-wrap items-center gap-2.5">
            <Button size="sm" icon="refresh" onClick={onTest} loading={testing} busyLabel={t(m.testing)}>
              {t(m.testConnection)}
            </Button>
            <span className="k">
              {v.testedAt ? (v.testOk ? t(m.testPassed, { when: when(v.testedAt) }) : t(m.testFailedAt, { when: when(v.testedAt), problem: v.lastError ?? '' })) : t(m.notTested)}
            </span>
          </div>
        </div>
      </div>
      <div className="ih-foot flex! flex-wrap items-center justify-between gap-3 border-t border-line-2 px-[18px] py-3">
        <span className="k">{t(m.settingsFoot)}</span>
        <Button variant="primary" onClick={() => void save()} loading={busy}>
          {t(m.save)}
        </Button>
      </div>
    </section>
  )
}

function Health({ v }: { v: ServiceView }) {
  const t = useT()
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(m.healthTitle)}</h2>
      </div>
      <dl className="kv pad">
        <dt>{t(m.hStatus)}</dt>
        <dd>
          <StatusBadge v={v} />
        </dd>
        <dt>{t(m.hBalance)}</dt>
        <dd>{v.balance != null && v.balanceAt ? t(m.hBalanceLine, { balance: v.balance, when: when(v.balanceAt) }) : t(m.hBalanceNone, { name: v.name })}</dd>
        <dt>{t(m.hLastUsed)}</dt>
        <dd>
          {v.lastUsedAt
            ? v.lastUsedBy
              ? t(m.hLastUsedLine, { when: whenCap(v.lastUsedAt), name: v.lastUsedBy, what: v.lastUsedWhat ?? '' })
              : t(m.hLastUsedLineAnon, { when: whenCap(v.lastUsedAt), what: v.lastUsedWhat ?? '' })
            : t(m.statLastUsedNever)}
        </dd>
        <dt>{t(m.hLastError)}</dt>
        <dd>{v.lastError && v.lastErrorAt ? `${when(v.lastErrorAt)} · ${v.lastError}` : t(m.hLastErrorNone)}</dd>
        {v.capabilities.dataRegion ? (
          <>
            <dt>{t(m.hRegion)}</dt>
            <dd>{v.regionLabel ? t(m.hRegionLine, { region: v.regionLabel, name: v.name }) : t(m.hRegionNone, { name: v.name })}</dd>
          </>
        ) : null}
      </dl>
    </section>
  )
}

/** "Today at 09:12", capitalised at the start of a value. */
function whenCap(d: string): string {
  const s = when(d)
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function Receives({ v }: { v: ServiceView }) {
  const t = useT()
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.receivesTitle, { name: v.name })}</h2>
          <span className="k">{t(m.receivesLead)}</span>
        </div>
      </div>
      <div className="pad">
        <Notice head={v.regionLabel ? t(m.noticeEvery, { name: v.name, region: v.regionLabel }) : t(m.noticeEveryNoRegion, { name: v.name })} name={v.name} />
      </div>
    </section>
  )
}

/** "catch-all", "role": a risky reason in a badge. */
function shortReason(t: ReturnType<typeof useT>, r: string | undefined): string | undefined {
  switch (r) {
    case 'catch_all':
      return t(m.shortCatchAll)
    case 'role':
      return t(m.shortRole)
    case 'disposable':
      return t(m.shortDisposable)
    case 'abuse':
      return t(m.shortAbuse)
  }
  // A service's own reason, as it names it ("low_quality" → "low quality").
  return r?.replace(/_/g, ' ')
}

function MapCard({ v }: { v: ServiceView }) {
  const t = useT()
  const first = v.mappings.find((x) => x.result === 'invalid')
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.mapTitle, { name: v.name })}</h2>
          <span className="k">{t(m.mapLead)}</span>
        </div>
      </div>
      <TableScroll label={t(m.mapTableLabel, { name: v.name })}>
        <table className="tbl" aria-label={t(m.mapTableLabel, { name: v.name })}>
          <thead>
            <tr>
              <Th>{t(m.mapAnswers, { name: v.name })}</Th>
              <Th>{t(m.mapBecomes)}</Th>
              <Th>{t(m.mapWhat)}</Th>
            </tr>
          </thead>
          <tbody>
            {v.mappings.map((x, i) => (
              <tr key={i}>
                <td className="mono text-[12.5px]">{x.codes.join(', ')}</td>
                <td className="nw">
                  <ResultBadge result={x.result} reason={x.result === 'risky' ? shortReason(t, x.reason) : undefined} />
                </td>
                <td>
                  {x.result === 'valid' ? (
                    t(m.mapValid)
                  ) : x.result === 'invalid' ? (
                    t(m.mapInvalid, { reason: x.reason === 'spam_trap' ? t(m.reasonSpamTrap) : t(m.reasonInvalidAddress), name: v.name })
                  ) : x.result === 'unknown' ? (
                    <T msg={m.mapUnknown} values={{ ...tagValue, code: x.tag ?? 'verify:unknown' }} />
                  ) : (
                    <T msg={x.reason === 'abuse' ? m.mapTaggedOwn : m.mapTagged} values={{ ...tagValue, code: x.tag ?? '', name: v.name }} />
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableScroll>
      {first ? (
        <div className="border-t border-line-2 px-[18px] py-3">
          <span className="k">{t(m.mapFoot, { name: v.name, code: first.codes[0] ?? '' })}</span>
        </div>
      ) : null}
    </section>
  )
}

function whatOf(u: Use, t: ReturnType<typeof useT>): string {
  switch (u.kind) {
    case 'import':
      return t(m.useImport, { label: u.label })
    case 'list':
      return t(m.useList, { label: u.label })
    case 'segment':
      return t(m.useSegment, { label: u.label })
    case 'tier':
      return t(m.useTier, { label: u.label })
    case 'contact':
      return t(m.useContact, { label: u.label })
    case 'realtime':
      return t(m.useRealtime)
  }
  return t(m.useSelection, { label: u.label })
}

function UsesCard({ v, uses, canManage }: { v: ServiceView; uses: Use[]; canManage: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const stamp = (d: string) => `${shortDate(d)} ${clock(d)}`
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(m.usesTitle, { count: uses.length })}</h2>
      </div>
      {uses.length === 0 ? (
        <div className="pad">
          <span className="k">{t(m.usesNone)}</span>
        </div>
      ) : (
        <TableScroll label={t(m.usesTableLabel, { name: v.name })}>
          <table className="tbl" aria-label={t(m.usesTableLabel, { name: v.name })}>
            <thead>
              <tr>
                <Th>{t(m.colWhen)}</Th>
                <Th>{t(m.colWho)}</Th>
                <Th>{t(m.colWhat)}</Th>
                <Th align="right">
                  {t(m.colAddresses)}
                </Th>
                <Th align="right">
                  {t(m.colCost)}
                </Th>
                <Th>{t(m.colResults)}</Th>
                {canManage ? (
                  <Th>
                    <span className="vh">{t(m.job)}</span>
                  </Th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {uses.map((u) => (
                <tr key={u.id}>
                  <td className="nw">{stamp(u.at)}</td>
                  <td>
                    <span className="inline-flex items-center gap-2 whitespace-nowrap">
                      <Avatar name={u.by} size={22} fontSize={11} />
                      {u.by}
                    </span>
                  </td>
                  <td>{whatOf(u, t)}</td>
                  <td className="r">{count(u.addresses)}</td>
                  <td className="r">{money(u.cost)}</td>
                  <td>{u.status === 'finished' ? resultsLine(u) : t(m.usesRunning, { status: u.status })}</td>
                  {canManage ? (
                    <td className="r">
                      {u.kind === 'import' ? (
                        <ButtonLink size="sm" to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ category: 'verification' }}>
                          {t(m.auditEntry)}
                        </ButtonLink>
                      ) : (
                        <ButtonLink size="sm" to="/w/$workspace/audience/verification/$jobId" params={{ workspace: slug, jobId: u.id }}>
                          {t(m.job)}
                        </ButtonLink>
                      )}
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line-2 px-[18px] py-3">
        <span className="k flex-[1_1_320px]">{t(m.usesFoot)}</span>
        <Link to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ category: 'verification' }} className="pl-lnk font-semibold">
          {t(m.auditLog)}
        </Link>
      </div>
    </section>
  )
}

function ServicesLoading() {
  const t = useT()
  return (
    <Loading label={t(m.loading)} className="card flex flex-col gap-3.5 p-[18px]">
      <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(160px,100%),1fr))' }}>
        {[0, 1, 2].map((i) => (
          <div key={i} className="card flex flex-col gap-2 p-3.5">
            <Skeleton w={40} h={40} r={10} />
            <Skeleton w={110} h={12} />
            <Skeleton w={140} h={10} />
          </div>
        ))}
      </div>
      {[0, 1, 2].map((i) => (
        <div key={i} className="flex items-center gap-3.5 border-t border-line-2 py-2.5">
          <Skeleton w={160} h={12} />
          <Skeleton w={110} h={12} />
          <Skeleton w={70} h={20} r={999} />
        </div>
      ))}
    </Loading>
  )
}
