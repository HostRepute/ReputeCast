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
import { TableScroll } from '@/components/ui/Table'
import { clock, count, pct, rate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { AudienceTabs } from './AudienceTabs'
import { DeleteListDialog } from './DeleteListDialog'
import { tierColor, tierLabel, tierWindow, tiers, type Tier } from './labels'
import './audience.css'

type Health = Schemas['ListHealth']
type Policy = Schemas['SunsetPolicy']

const minus = (n: number) => (n < 0 ? `−${Math.abs(n).toFixed(1)}` : `+${n.toFixed(1)}`)

const months = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']
/** A calendar day (YYYY-MM-DD) as "1 Sep", or "1 September" with `long`; no timezone shifts. */
function calDay(s: string, plusDays = 0, long = false) {
  const [y = 1970, m = 1, d = 1] = s.split('-').map(Number)
  const t = new Date(Date.UTC(y, m - 1, d + plusDays))
  const name = months[t.getUTCMonth()] ?? ''
  return `${t.getUTCDate()} ${long ? name : name.slice(0, 3)}`
}
/** Shares (0–1) as "55.5" / "61" percent for the chart description. */
const sharePct = (v: number) => String(Number((v * 100).toFixed(1)))
function change(name: string, a: number, b: number) {
  const from = sharePct(a)
  const to = sharePct(b)
  return from === to ? `${name} stays at ${to} percent` : `${name} ${b > a ? 'grows' : 'shrinks'} from ${from} to ${to} percent`
}

/** 7.11 List health: tiers, the 12-week trend, the sunset policy and health per list. */
export function HealthPage() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/audience/health', { params: { path: { workspace: slug } } })
  const h = q.data
  // No access: the shell shows its 403 screen.
  if (!h && q.error instanceof ApiError && q.error.status === 403) throw q.error
  return (
    <Page
      title="Audience"
      documentTitle="List health"
      filters={
        <>
          <AudienceTabs />
          {h?.computedAt ? <span className="k ml-auto">Tiers recalculated nightly · last run {clock(h.computedAt)}</span> : null}
        </>
      }
    >
      {q.isError ? (
        <ErrorPanel title="Couldn't load list health" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed.
        </ErrorPanel>
      ) : !h ? (
        <HealthSkeleton />
      ) : (
        <>
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
  return (
    <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
      {h.tiers.map((c, i) => {
        const t = c.tier as Tier
        return (
          <div key={t} className="card flex flex-col gap-0.5 px-4.5 py-3.5" style={{ borderLeft: `4px solid ${tierColor[t]}` }}>
            <span className="k">
              {tierLabel[t]} · {tierWindow[t]}
            </span>
            <span className="n text-[22px]">{count(c.count)}</span>
            <span className="k">
              {pct(c.pct)}
              {c.deltaPts !== undefined && c.deltaPts !== null ? (
                <>
                  {' · '}
                  <span className={c.good ? 'text-ok-fg' : 'text-bad-fg'}>{minus(c.deltaPts)} pts</span>
                  {i === 0 ? ' vs last month' : ''}
                </>
              ) : null}
              {t === 'sunset' && h.thisWeek.entering > 0 ? ` · ${count(h.thisWeek.entering)} entering this week` : ''}
            </span>
          </div>
        )
      })}
    </div>
  )
}

function Trend({ h }: { h: Health }) {
  const weeks = h.trend
  const first = weeks[0]
  const last = weeks[weeks.length - 1]
  const enabledAt = h.policy.enabled && h.policyWeek >= 0 ? h.policy.enabledAt : undefined
  const share = (t: Tier) => h.tiers.find((c) => c.tier === t)?.pct ?? 0
  const desc =
    first && last
      ? `Weekly tier mix over ${weeks.length} weeks; ${change('Active', first.active, last.active)}, ${change('Sunset', first.sunset, last.sunset)}${
          enabledAt ? ` after the sunset policy started on ${calDay(enabledAt, 0, true)}` : ''
        }`
      : 'No tier history yet'
  return (
    <section className="card flex min-w-0 flex-[2_1_560px] flex-col gap-3 px-5 py-4.5" aria-labelledby="trend-h">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2" id="trend-h">
          Tier trend · last 12 weeks
        </h2>
        <div className="flex gap-3 text-[12px] max-md:flex-wrap">
          {tiers.map((t) => (
            <span key={t} className="flex items-center gap-1.5">
              <span className="dot" style={{ color: tierColor[t] }} aria-hidden="true" />
              {tierLabel[t]}
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
        {enabledAt ? <span>{calDay(enabledAt)} · sunset policy enabled</span> : null}
        <span>{last ? calDay(last.day) : ''}</span>
      </div>
      <span className="k">
        Each column is one week; heights are shares of the list. Direct values for this week: Active {pct(share('active'))}, Cooling {pct(share('cooling'))}, At risk{' '}
        {pct(share('at_risk'))}, Sunset {pct(share('sunset'))}.
      </span>
    </section>
  )
}

/** "30 days without an open" ↔ 30. */
function daysText(n: number, suffix: string) {
  return `${n} days${suffix}`
}
function parseDays(s: string) {
  const m = /\d+/.exec(s)
  return m ? Number(m[0]) : NaN
}

function PolicyCard({ h }: { h: Health }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const p = h.policy
  const reason = usePermissionReason('lists:manage')
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/audience/sunset-policy')
  const [active, setActive] = useState(daysText(p.activeDays, ' without an open'))
  const [atRisk, setAtRisk] = useState(daysText(p.atRiskDays, ''))
  const [sunset, setSunset] = useState(daysText(p.sunsetDays, ''))
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
      setErr('Enter the tier thresholds as numbers of days.')
      return
    }
    try {
      await save.mutateAsync({ params: { path: { workspace: slug } }, body })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/audience/health'] })
      toast({ kind: 'success', title: 'Saved the sunset policy', detail: 'Tiers follow the new thresholds after tonight’s run at 03:00.' })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  const winBack = h.winBackRate !== undefined && h.winBackRate !== null ? ` · ${pct(h.winBackRate)} win back so far` : ''
  const week = h.thisWeek
  return (
    <section className="card flex min-w-0 flex-[1_1_360px] flex-col" aria-labelledby="policy-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="policy-h">
          Sunset policy
        </h2>
        {p.enabled ? (
          <Badge tone="ok" dot>
            On since {p.enabledAt ? calDay(p.enabledAt) : 'today'}
          </Badge>
        ) : (
          <Badge tone="neutral">Off</Badge>
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
            Active until
            <input className="in" value={active} onChange={(e) => setActive(e.target.value)} onBlur={() => setActive(daysText(parseDays(active) || p.activeDays, ' without an open'))} />
          </label>
          <label className="lbl">
            At risk from
            <input className="in" value={atRisk} onChange={(e) => setAtRisk(e.target.value)} onBlur={() => setAtRisk(daysText(parseDays(atRisk) || p.atRiskDays, ''))} />
          </label>
          <label className="lbl">
            Sunset from
            <input className="in" value={sunset} onChange={(e) => setSunset(e.target.value)} onBlur={() => setSunset(daysText(parseDays(sunset) || p.sunsetDays, ''))} />
          </label>
          <label className="lbl">
            Count machine opens?
            <select className="in" value={machine ? 'yes' : 'no'} onChange={(e) => setMachine(e.target.value === 'yes')}>
              <option value="no">No, human opens and clicks only</option>
              <option value="yes">Yes</option>
            </select>
          </label>
        </div>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={reengage} onChange={(e) => setReengage(e.target.checked)} />
          <span>
            Send one re-engagement email when a contact enters Sunset{' '}
            <span className="k">
              · automation "{p.reengageAutomation || 'Re-engagement'}"{winBack}
            </span>
          </span>
        </label>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={stop} onChange={(e) => setStop(e.target.checked)} />
          <span>
            Stop sending Broadcasts {p.stopAfterDays} days after that if there's still no open <span className="k">· Automations and System mail continue</span>
          </span>
        </label>
        <label className="flex items-start gap-2.5 text-[13px]">
          <input type="checkbox" className="mt-[3px]" checked={del} onChange={(e) => setDel(e.target.checked)} />
          <span>
            Delete the contact after {p.deleteAfterMonths} months in Sunset{' '}
            <span className="k">· {del ? 'erased with their consent history; suppressions stay' : 'off; GDPR retention is handled in Settings'}</span>
          </span>
        </label>
        <div className="flex gap-2 max-md:flex-wrap">
          <Button size="sm" variant="primary" type="submit" disabledReason={reason} loading={save.isPending}>
            Save policy
          </Button>
          {h.reengageFlowId ? (
            <ButtonLink size="sm" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: h.reengageFlowId }}>
              Edit re-engagement flow
            </ButtonLink>
          ) : (
            <ButtonLink size="sm" to="/w/$workspace/automations" params={{ workspace: slug }}>
              Edit re-engagement flow
            </ButtonLink>
          )}
        </div>
      </form>
      {p.enabled && (week.entering > 0 || week.stopping > 0) ? (
        <div className="rounded-b-[10px] border-t border-line-2 bg-[#FFFBEB] px-4.5 py-3 text-[13px]" role="status">
          <strong>This week:</strong>{' '}
          {week.entering > 0
            ? `${count(week.entering)} contacts enter Sunset${week.enterDay ? ` on ${week.enterDay}` : ''}${p.reengage ? ' and get the re-engagement email' : ''}`
            : ''}
          {week.entering > 0 && week.stopping > 0 ? '; ' : ''}
          {week.stopping > 0 ? `${count(week.stopping)} who didn't respond last month stop receiving Broadcasts${week.stopDay ? ` on ${week.stopDay}` : ''}` : ''}.
        </div>
      ) : null}
    </section>
  )
}

const healthTone: Record<string, Tone> = { ok: 'ok', warn: 'warn', bad: 'bad' }

function PerList({ h }: { h: Health }) {
  const { slug } = useWorkspace()
  // Lists are deleted from their row's menu, for people who manage lists.
  const manage = !usePermissionReason('lists:manage')
  const [deleting, setDeleting] = useState<{ id: string; name: string }>()
  const [menu, setMenu] = useState<string>()
  return (
    <section className="card" aria-labelledby="perlist-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="perlist-h">
          Health per list
        </h2>
        <span className="k">30-day figures</span>
      </div>
      <TableScroll label="Health per list">
        <table className="tbl compact first160 phone-wrap fit">
          <thead>
            <tr>
              <th scope="col">List</th>
              <th scope="col" className="r">
                Contacts
              </th>
              <th scope="col">Tier mix</th>
              <th scope="col" className="r">
                Net growth · 30 d
              </th>
              <th scope="col" className="r">
                Human opens
              </th>
              <th scope="col" className="r">
                Unsub rate
              </th>
              <th scope="col" className="r">
                Complaint rate
              </th>
              <th scope="col" className="r">
                Bounce rate
              </th>
              <th scope="col">Health</th>
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
                        {r.name} <span className="k">(saved view)</span>
                      </>
                    ) : manage ? (
                      // The row menu sits with the name, so the table keeps
                      // the board's columns and the menu is in view on phones.
                      <div className="flex items-center justify-between gap-2">
                        <span className="min-w-0">{r.name}</span>
                        <Menu
                          label={`Actions for ${r.name}`}
                          align="start"
                          open={menu === r.id}
                          onOpenChange={(o) => setMenu(o ? r.id : undefined)}
                          trigger={
                            <button
                              type="button"
                              className="ibtn -my-2 flex-none"
                              aria-label={`Actions for ${r.name}`}
                              style={menu === r.id ? { background: 'var(--line-2)' } : undefined}
                            >
                              ⋯
                            </button>
                          }
                        >
                          <MenuItem className="text-bad-fg" onSelect={() => setDeleting({ id: r.id, name: r.name })}>
                            <Icon name="trash" />
                            Delete list
                          </MenuItem>
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
                      aria-label={mix.map(([t, n]) => `${tierLabel[t]} ${Math.round((n / total) * 100)}%`).join(', ')}
                    >
                      {mix.map(([t, n]) => (n > 0 ? <div key={t} style={{ width: `${(n / total) * 100}%`, background: tierColor[t] }} /> : null))}
                    </div>
                  </td>
                  <td className={`r n ${r.netGrowth > 0 ? 'text-ok-fg' : r.netGrowth < 0 ? 'text-bad-fg' : ''}`}>
                    {r.netGrowth > 0 ? `+${count(r.netGrowth)}` : r.netGrowth < 0 ? `−${count(-r.netGrowth)}` : '0'}
                  </td>
                  <td className="r n">{r.humanOpens !== undefined && r.humanOpens !== null ? pct(r.humanOpens) : '–'}</td>
                  <td className="r n">{r.unsubRate !== undefined && r.unsubRate !== null ? rate(r.unsubRate) : '–'}</td>
                  <td className={`r n ${r.complaintHigh ? 'text-warn-fg' : ''}`}>{r.complaintRate !== undefined && r.complaintRate !== null ? `${r.complaintRate.toFixed(2)}%` : '–'}</td>
                  <td className="r n">{r.bounceRate !== undefined && r.bounceRate !== null ? pct(r.bounceRate) : '–'}</td>
                  <td>
                    <Badge tone={healthTone[r.healthTone] ?? 'neutral'}>{r.health}</Badge>
                  </td>
                </tr>
              )
            })}
            {h.rows.length === 0 ? (
              <tr>
                <td colSpan={9} className="py-8 text-center text-ink-3">
                  No lists yet.{' '}
                  <Link to="/w/$workspace/audience" params={{ workspace: slug }} className="font-semibold">
                    Add contacts
                  </Link>{' '}
                  to see how each list is doing.
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
            Start re-permission flow
          </ButtonLink>
        </div>
      ) : null}
      {deleting ? <DeleteListDialog list={deleting} open onOpenChange={(o) => !o && setDeleting(undefined)} /> : null}
    </section>
  )
}

function HealthSkeleton() {
  return (
    <Loading label="Loading list health" className="flex flex-col gap-5">
      <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
        {tiers.map((t) => (
          <div key={t} className="card flex flex-col gap-2 px-4.5 py-3.5" style={{ borderLeft: `4px solid ${tierColor[t]}` }}>
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
