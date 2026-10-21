import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { TableScroll, Th } from '@/components/ui/Table'
import { T, getI18n, t, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { formatDate, formatPattern } from '@/i18n/regions'
import { WelcomePromptCard } from '@/features/welcome-prompt/WelcomePromptCard'
import { NetGrowthBySource } from '@/features/growth/BySource'
import { TiersNote } from '@/features/scoring/TiersNote'
import { clock, count, decimal, pct, rate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ListVerifyDialog, TierChecked, TierVerify, useVerifyLabel } from '@/features/verification/HealthEntry'
import { AudienceTabs } from './AudienceTabs'
import { DeleteListDialog } from './DeleteListDialog'
import { tierColor, tierLabel, tierWindow, tiers, type Tier } from './labels'
import './audience.css'
import { m as am } from './messages'
import { m } from './health.messages'

type Health = Schemas['ListHealth']
type Policy = Schemas['SunsetPolicy']

const minus = (n: number) => (n < 0 ? `−${decimal(Math.abs(n), 1)}` : `+${decimal(n, 1)}`)

/** A calendar day (YYYY-MM-DD) as "1 Sep", or "1 September" with `long`, in the region format; no timezone shifts. */
function calDay(s: string, plusDays = 0, long = false) {
  const [y = 1970, mo = 1, d = 1] = s.split('-').map(Number)
  const day = new Date(Date.UTC(y, mo - 1, d + plusDays))
  const parts = { year: day.getUTCFullYear(), month: day.getUTCMonth() + 1, day: day.getUTCDate(), weekday: day.getUTCDay(), hour: 0, minute: 0, second: 0 }
  const region = getI18n().regionFormat
  return long ? formatPattern(region, 'd MMMM', parts) : formatDate(region, 'dayMonth', parts)
}
/** Shares (0–1) as "55.5" / "61" percent for the chart description. */
const sharePct = (v: number) => {
  const n = Number((v * 100).toFixed(1))
  return decimal(n, Number.isInteger(n) ? 0 : 1)
}
function change(tier: string, a: number, b: number) {
  const from = sharePct(a)
  const to = sharePct(b)
  return from === to ? t(m.tierStays, { tier, to }) : t(b > a ? m.tierGrows : m.tierShrinks, { tier, from, to })
}

/** 7.11 List health: tiers, the 12-week trend, the sunset policy and health per list. */
export function HealthPage() {
  const tr = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/audience/health', { params: { path: { workspace: slug } } })
  const h = q.data
  // No access: the shell shows its 403 screen.
  if (!h && q.error instanceof ApiError && q.error.status === 403) throw q.error
  return (
    <Page
      title={tr(am.title)}
      documentTitle={tr(m.documentTitle)}
      filters={
        <>
          <AudienceTabs />
          {h?.computedAt ? <span className="k ml-auto">{tr(m.recalculated, { time: clock(h.computedAt) })}</span> : null}
        </>
      }
    >
      {q.isError ? (
        <ErrorPanel title={tr(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{tr(common.retry)}</Button>}>
          {errorMessage(q.error)} {tr(m.nothingChanged)}
        </ErrorPanel>
      ) : !h ? (
        <HealthSkeleton />
      ) : (
        <>
          <WelcomePromptCard where={{ context: 'health' }} />
          <TiersNote on="health" />
          <TierCards h={h} />
          <div className="flex flex-wrap items-start gap-5">
            <Trend h={h} />
            <PolicyCard key={JSON.stringify(h.policy)} h={h} />
          </div>
          <PerList h={h} />
        </>
      )}
    </Page>
  )
}

function TierCards({ h }: { h: Health }) {
  const tr = useT()
  return (
    <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
      {h.tiers.map((c, i) => {
        const tier = c.tier as Tier
        return (
          <div key={tier} className="card flex flex-col gap-0.5 px-4.5 py-3.5" style={{ borderLeft: `4px solid ${tierColor[tier]}` }}>
            <span className="k">{tr(m.tierWindow, { tier: tierLabel[tier], window: tierWindow[tier] })}</span>
            <span className="n text-[22px]">{count(c.count)}</span>
            <span className="k">
              {pct(c.pct)}
              {c.deltaPts !== undefined && c.deltaPts !== null ? (
                <>
                  {' · '}
                  <T
                    msg={i === 0 ? m.ptsVsLastMonth : m.pts}
                    values={{ delta: minus(c.deltaPts), c: (chunks) => <span className={c.good ? 'text-ok-fg' : 'text-bad-fg'}>{chunks}</span> }}
                  />
                </>
              ) : null}
              {tier === 'sunset' && h.thisWeek.entering > 0 ? ` · ${tr(m.entering, { count: count(h.thisWeek.entering) })}` : ''}
              {tier === 'at_risk' || tier === 'sunset' ? <TierChecked tier={tier} /> : null}
            </span>
            {tier === 'at_risk' || tier === 'sunset' ? <TierVerify tier={tier} /> : null}
          </div>
        )
      })}
    </div>
  )
}

function Trend({ h }: { h: Health }) {
  const tr = useT()
  const weeks = h.trend
  const first = weeks[0]
  const last = weeks[weeks.length - 1]
  const enabledAt = h.policy.enabled && h.policyWeek >= 0 ? h.policy.enabledAt : undefined
  const share = (t: Tier) => h.tiers.find((c) => c.tier === t)?.pct ?? 0
  const desc =
    first && last
      ? tr(enabledAt ? m.trendDescPolicy : m.trendDesc, {
          weeks: weeks.length,
          active: change(tierLabel.active, first.active, last.active),
          sunset: change(tierLabel.sunset, first.sunset, last.sunset),
          date: enabledAt ? calDay(enabledAt, 0, true) : '',
        })
      : tr(m.noHistory)
  return (
    <section className="card flex min-w-0 flex-[2_1_560px] flex-col gap-3 px-5 py-4.5" aria-labelledby="trend-h">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2" id="trend-h">
          {tr(m.trendTitle)}
        </h2>
        <div className="flex gap-3 text-[12px] max-md:flex-wrap">
          {tiers.map((x) => (
            <span key={x} className="flex items-center gap-1.5">
              <span className="dot" style={{ color: tierColor[x] }} aria-hidden="true" />
              {tierLabel[x]}
            </span>
          ))}
        </div>
      </div>
      <div className="flex h-[200px] items-end gap-1.5" role="img" aria-label={desc}>
        {weeks.map((w, i) => (
          <div key={w.day} className="wk" style={i === h.policyWeek ? { outline: '1px dashed #9AA3B2', outlineOffset: 2 } : undefined}>
            <i style={{ height: Math.round(w.active * 200), background: tierColor.active }} />
            <i style={{ height: Math.round(w.cooling * 200), background: tierColor.cooling }} />
            <i style={{ height: Math.round(w.atRisk * 200), background: tierColor.at_risk }} />
            <i style={{ height: Math.round(w.sunset * 200), background: tierColor.sunset }} />
          </div>
        ))}
      </div>
      <div className="k flex justify-between">
        <span>{first ? calDay(first.day, -7) : ''}</span>
        {enabledAt ? <span>{tr(m.policyEnabledOn, { date: calDay(enabledAt) })}</span> : null}
        <span>{last ? calDay(last.day) : ''}</span>
      </div>
      <span className="k">
        {tr(m.trendNote, { active: pct(share('active')), cooling: pct(share('cooling')), atRisk: pct(share('at_risk')), sunset: pct(share('sunset')) })}
      </span>
    </section>
  )
}

/** "30 days without an open" ↔ 30. */
function daysText(n: number, withoutOpen: boolean) {
  return t(withoutOpen ? m.daysWithoutOpen : m.days, { n: String(n) })
}
function parseDays(s: string) {
  const m = /\d+/.exec(s)
  return m ? Number(m[0]) : NaN
}

function PolicyCard({ h }: { h: Health }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const p = h.policy
  const reason = usePermissionReason('lists:manage')
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/audience/sunset-policy')
  const [active, setActive] = useState(daysText(p.activeDays, true))
  const [atRisk, setAtRisk] = useState(daysText(p.atRiskDays, false))
  const [sunset, setSunset] = useState(daysText(p.sunsetDays, false))
  const [machine, setMachine] = useState(p.countMachineOpens)
  const [reengage, setReengage] = useState(p.reengage)
  const [stop, setStop] = useState(p.stopBroadcasts)
  const [del, setDel] = useState(p.deleteAfter)
  const [err, setErr] = useState<string | null>(null)
  const submit = async () => {
    setErr(null)
    const body: Policy = {
      ...p,
      activeDays: parseDays(active),
      atRiskDays: parseDays(atRisk),
      sunsetDays: parseDays(sunset),
      countMachineOpens: machine,
      reengage,
      stopBroadcasts: stop,
      deleteAfter: del,
    }
    if ([body.activeDays, body.atRiskDays, body.sunsetDays].some((n) => !Number.isFinite(n))) {
      setErr(t(m.thresholdsNumbers))
      return
    }
    try {
      await save.mutateAsync({ params: { path: { workspace: slug } }, body })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/audience/health'] })
      toast({ kind: 'success', title: t(m.policySaved), detail: t(m.policySavedBody, { time: '03:00' }) })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  const winBack = h.winBackRate !== undefined && h.winBackRate !== null ? ` · ${tr(m.winBack, { rate: pct(h.winBackRate) })}` : ''
  const week = h.thisWeek
  const weekText = [
    week.entering > 0 ? tr(m.weekEnter, { day: week.enterDay || 'none', reengage: p.reengage ? 'yes' : 'no', count: count(week.entering) }) : '',
    week.stopping > 0 ? tr(m.weekStop, { day: week.stopDay || 'none', count: count(week.stopping) }) : '',
  ]
    .filter(Boolean)
    .join('; ')
  return (
    <section className="card flex min-w-0 flex-[1_1_360px] flex-col" aria-labelledby="policy-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="policy-h">
          {tr(m.policyTitle)}
        </h2>
        {p.enabled ? (
          <Badge tone="ok" dot>
            {p.enabledAt ? tr(m.onSince, { date: calDay(p.enabledAt) }) : tr(m.onSinceToday)}
          </Badge>
        ) : (
          <Badge tone="neutral">{tr(common.off)}</Badge>
        )}
      </div>
      <form
        className="flex flex-col gap-3 px-4.5 py-4"
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        {err ? <Note tone="bad">{err}</Note> : null}
        <div className="grid grid-cols-2 gap-2.5 max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
          <label className="lbl">
            {tr(m.activeUntil)}
            <input className="in" value={active} onChange={(e) => setActive(e.target.value)} onBlur={() => setActive(daysText(parseDays(active) || p.activeDays, true))} />
          </label>
          <label className="lbl">
            {tr(m.atRiskFrom)}
            <input className="in" value={atRisk} onChange={(e) => setAtRisk(e.target.value)} onBlur={() => setAtRisk(daysText(parseDays(atRisk) || p.atRiskDays, false))} />
          </label>
          <label className="lbl">
            {tr(m.sunsetFrom)}
            <input className="in" value={sunset} onChange={(e) => setSunset(e.target.value)} onBlur={() => setSunset(daysText(parseDays(sunset) || p.sunsetDays, false))} />
          </label>
          {/* hp-machine: a longer language gives this select the whole row (styles/expansion.css). */}
          <label className="lbl hp-machine">
            {tr(m.countMachine)}
            <select className="in" value={machine ? 'yes' : 'no'} onChange={(e) => setMachine(e.target.value === 'yes')}>
              <option value="no">{tr(m.humanOnly)}</option>
              <option value="yes">{tr(common.yes)}</option>
            </select>
          </label>
        </div>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={reengage} onChange={(e) => setReengage(e.target.checked)} />
          <span>
            {tr(m.reengage)}{' '}
            <span className="k">
              {tr(m.reengageAutomation, { name: p.reengageAutomation || tr(m.reengageDefault) })}
              {winBack}
            </span>
          </span>
        </label>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={stop} onChange={(e) => setStop(e.target.checked)} />
          <span>
            {tr(m.stopBroadcasts, { n: p.stopAfterDays })} <span className="k">{tr(m.stopNote)}</span>
          </span>
        </label>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={del} onChange={(e) => setDel(e.target.checked)} />
          <span>
            {tr(m.deleteAfter, { n: p.deleteAfterMonths })} <span className="k">{del ? tr(m.deleteOn) : tr(m.deleteOff)}</span>
          </span>
        </label>
        <div className="flex gap-2 max-md:flex-wrap">
          <Button size="sm" variant="primary" type="submit" disabledReason={reason} loading={save.isPending}>
            {tr(m.savePolicy)}
          </Button>
          {h.reengageFlowId ? (
            <ButtonLink size="sm" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: h.reengageFlowId }}>
              {tr(m.editFlow)}
            </ButtonLink>
          ) : (
            <ButtonLink size="sm" to="/w/$workspace/automations" params={{ workspace: slug }}>
              {tr(m.editFlow)}
            </ButtonLink>
          )}
        </div>
      </form>
      {p.enabled && (week.entering > 0 || week.stopping > 0) ? (
        <div className="rounded-b-[10px] border-t border-line-2 bg-[#FFFBEB] px-4.5 py-3 text-[13px]" role="status">
          <T msg={m.thisWeek} values={{ text: weekText, b: (c) => <strong>{c}</strong> }} />
        </div>
      ) : null}
    </section>
  )
}

const healthTone: Record<string, Tone> = { ok: 'ok', warn: 'warn', bad: 'bad' }

/** The API's list health labels (internal/audience/health.go), as message keys. */
const healthKey: Record<string, string> = { Excellent: 'excellent', Good: 'good', 'Needs attention': 'attention', 'Re-permission needed': 'repermission' }

function PerList({ h }: { h: Health }) {
  const tr = useT()
  const { slug } = useWorkspace()
  // Lists are deleted from their row's menu, for people who manage lists.
  const manage = !usePermissionReason('lists:manage')
  const [deleting, setDeleting] = useState<{ id: string; name: string }>()
  const [menu, setMenu] = useState<string>()
  // Verify addresses… sits in the same menu, for people who can verify (X14).
  const verifyLabel = useVerifyLabel()
  const [verifying, setVerifying] = useState<{ id: string; name: string }>()
  return (
    <section className="card" aria-labelledby="perlist-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="perlist-h">
          {tr(m.perList)}
        </h2>
        <span className="k">
          <T msg={m.figures} values={{ link: <NetGrowthBySource /> }} />
        </span>
      </div>
      <TableScroll label={tr(m.perList)}>
        <table className="tbl compact first160 phone-wrap fit">
          <thead>
            <tr>
              <Th>{tr(m.colList)}</Th>
              <Th align="right">{tr(m.colContacts)}</Th>
              <Th>{tr(m.colMix)}</Th>
              <Th align="right">{tr(m.colNet)}</Th>
              <Th align="right">{tr(m.colHumanOpens)}</Th>
              <Th align="right">{tr(m.colUnsub)}</Th>
              <Th align="right">{tr(m.colComplaint)}</Th>
              <Th align="right">{tr(m.colBounce)}</Th>
              <Th>{tr(m.colHealth)}</Th>
            </tr>
          </thead>
          <tbody>
            {h.rows.map((r) => {
              const total = r.tiers.active + r.tiers.cooling + r.tiers.atRisk + r.tiers.sunset || 1
              const mix: [Tier, number][] = [
                ['active', r.tiers.active],
                ['cooling', r.tiers.cooling],
                ['at_risk', r.tiers.atRisk],
                ['sunset', r.tiers.sunset],
              ]
              const warn = r.healthTone !== 'ok'
              return (
                <tr key={`${r.kind}:${r.id}`} className={warn ? 'bg-[#FFFBEB]' : undefined}>
                  <td className="font-semibold">
                    {r.kind === 'view' ? (
                      <>
                        {r.name} <span className="k">{tr(m.savedView)}</span>
                      </>
                    ) : manage || verifyLabel ? (
                      // The row menu sits with the name, so the table keeps
                      // the board's columns and the menu is in view on phones.
                      <div className="flex items-center justify-between gap-2">
                        <span className="min-w-0">{r.name}</span>
                        <Menu
                          label={tr(m.actionsFor, { name: r.name })}
                          align="start"
                          open={menu === r.id}
                          onOpenChange={(o) => setMenu(o ? r.id : undefined)}
                          trigger={
                            <button
                              type="button"
                              className="ibtn -my-2 flex-none"
                              aria-label={tr(m.actionsFor, { name: r.name })}
                              style={menu === r.id ? { background: 'var(--line-2)' } : undefined}
                            >
                              ⋯
                            </button>
                          }
                        >
                          {verifyLabel ? <MenuItem onSelect={() => setVerifying({ id: r.id, name: r.name })}>{verifyLabel}</MenuItem> : null}
                          {manage ? (
                            <MenuItem className="text-bad-fg" onSelect={() => setDeleting({ id: r.id, name: r.name })}>
                              <Icon name="trash" />
                              {tr(m.deleteList)}
                            </MenuItem>
                          ) : null}
                        </Menu>
                      </div>
                    ) : (
                      r.name
                    )}
                  </td>
                  <td className="r n">{count(r.contacts)}</td>
                  <td>
                    <div
                      className="flex h-2 w-[120px] gap-px overflow-hidden rounded-[4px]"
                      role="img"
                      aria-label={mix.map(([x, n]) => tr(m.tierShare, { tier: tierLabel[x], percent: pct(Math.round((n / total) * 100), 0) })).join(', ')}
                    >
                      {mix.map(([x, n]) => (n > 0 ? <div key={x} style={{ width: `${(n / total) * 100}%`, background: tierColor[x] }} /> : null))}
                    </div>
                  </td>
                  <td className={`r n ${r.netGrowth > 0 ? 'text-ok-fg' : r.netGrowth < 0 ? 'text-bad-fg' : ''}`}>
                    {r.netGrowth > 0 ? `+${count(r.netGrowth)}` : r.netGrowth < 0 ? `−${count(-r.netGrowth)}` : '0'}
                  </td>
                  <td className="r n">{r.humanOpens !== undefined && r.humanOpens !== null ? pct(r.humanOpens) : '–'}</td>
                  <td className="r n">{r.unsubRate !== undefined && r.unsubRate !== null ? rate(r.unsubRate) : '–'}</td>
                  <td className={`r n ${r.complaintHigh ? 'text-warn-fg' : ''}`}>{r.complaintRate !== undefined && r.complaintRate !== null ? pct(r.complaintRate, 2) : '–'}</td>
                  <td className="r n">{r.bounceRate !== undefined && r.bounceRate !== null ? pct(r.bounceRate) : '–'}</td>
                  <td>
                    <Badge tone={healthTone[r.healthTone] ?? 'neutral'}>{tr(m.health, { health: healthKey[r.health] ?? 'other', raw: r.health })}</Badge>
                  </td>
                </tr>
              )
            })}
            {h.rows.length === 0 ? (
              <tr>
                <td colSpan={9} className="py-8 text-center text-ink-3">
                  <T
                    msg={m.noLists}
                    values={{
                      link: (c) => (
                        <Link to="/w/$workspace/audience" params={{ workspace: slug }} className="font-semibold">
                          {c}
                        </Link>
                      ),
                    }}
                  />
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </TableScroll>
      {h.insight ? (
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4.5 py-2.5">
          <span className="k">{h.insight.text}</span>
          <ButtonLink size="sm" to="/w/$workspace/automations" params={{ workspace: slug }}>
            {tr(m.startFlow)}
          </ButtonLink>
        </div>
      ) : null}
      {deleting ? <DeleteListDialog list={deleting} open onOpenChange={(o) => !o && setDeleting(undefined)} /> : null}
      {verifying ? <ListVerifyDialog list={verifying} onClose={() => setVerifying(undefined)} /> : null}
    </section>
  )
}

function HealthSkeleton() {
  const tr = useT()
  return (
    <Loading label={tr(m.loading)} className="flex flex-col gap-5">
      <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
        {tiers.map((x) => (
          <div key={x} className="card flex flex-col gap-2 px-4.5 py-3.5" style={{ borderLeft: `4px solid ${tierColor[x]}` }}>
            <Skeleton w={140} h={11} />
            <Skeleton w={90} h={22} />
            <Skeleton w={160} h={11} />
          </div>
        ))}
      </div>
      <div className="flex flex-wrap items-start gap-5">
        <div className="card flex flex-[2_1_560px] flex-col gap-3 px-5 py-4.5">
          <Skeleton w={200} h={15} />
          <Skeleton w="100%" h={200} r={6} />
        </div>
        <div className="card flex flex-[1_1_360px] flex-col gap-3 px-4.5 py-4">
          <Skeleton w={140} h={15} />
          <Skeleton w="100%" h={36} />
          <Skeleton w="100%" h={36} />
          <Skeleton w="80%" h={14} />
        </div>
      </div>
    </Loading>
  )
}
