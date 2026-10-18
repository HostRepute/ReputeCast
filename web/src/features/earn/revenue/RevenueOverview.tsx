import { keepPreviousData } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Tooltip } from '@/components/ui/Menu'
import { Segmented } from '@/components/ui/Tabs'
import { T, useT } from '@/i18n'
import { andList, count, dayMonthDate, dayMonthYear, orList, pct, shortDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { EARN, useEarnStatus } from '../EarnPage'
import { day, money, platformName } from '../format'
import { ends } from '../sponsors/parts'
import { MovementChart, MrrChart, moveBars, moveColors } from './charts'
import { mr } from './messages'
import './revenue.css'

type Overview = Schemas['RevenueOverview']
type Period = '30d' | '90d' | '12m'
type TT = ReturnType<typeof useT>

const tierColors = ['#104281', '#256abf', '#5598e7', '#86b6ef', '#b8d4f5']

/** ▲ $418 (+4.9%) */
function Delta({ cents, pctValue, currency }: { cents: number; pctValue?: number | null; currency: string }) {
  const t = useT()
  const amount = money(Math.abs(cents), currency, 'never')
  if (pctValue === undefined || pctValue === null) {
    return <span className={`er-mv ${cents < 0 ? 'dn' : 'up'}`}>{cents < 0 ? t(mr.down, { amount }) : t(mr.up, { amount })}</span>
  }
  const p = `${pctValue < 0 ? '−' : '+'}${pct(Math.abs(pctValue), 1)}`
  return <span className={`er-mv ${cents < 0 ? 'dn' : 'up'}`}>{cents < 0 ? t(mr.downPct, { amount, pct: p }) : t(mr.upPct, { amount, pct: p })}</span>
}

/** ▲ 65 */
function CountDelta({ n }: { n: number }) {
  const t = useT()
  return <span className={`er-mv ${n < 0 ? 'dn' : 'up'}`}>{n < 0 ? t(mr.down, { amount: count(-n) }) : t(mr.up, { amount: count(n) })}</span>
}

/** The (i) beside a title, with its definition. */
export function Def({ label, text }: { label: string; text: string }) {
  return (
    <Tooltip content={<span className="er-tipbox block">{text}</span>} side="top">
      <button type="button" className="er-tip" aria-label={label}>
        <Icon name="info" size={14} />
      </button>
    </Tooltip>
  )
}

function Card({ title, sub, aside, children, className, style }: { title: ReactNode; sub?: ReactNode; aside?: ReactNode; children: ReactNode; className?: string; style?: React.CSSProperties }) {
  return (
    <section className={`card ${className ?? ''}`} style={style}>
      <div className="er-sec">
        <div className="er-head">
          <h2 className="h2">{title}</h2>
          {sub ? <span className="k">{sub}</span> : null}
        </div>
        {aside ? <div className="flex flex-wrap items-center gap-2">{aside}</div> : null}
      </div>
      {children}
    </section>
  )
}

/** A card that couldn't be computed: the rest of the page still works. */
function Failed({ card, title, onRetry, reference }: { card: string; title: ReactNode; onRetry: () => void; reference?: string }) {
  const t = useT()
  return (
    <Card title={title}>
      <div className="er-failed" role="alert">
        <span style={{ color: 'var(--bad-strong)' }}>
          <Icon name="alertCircle" size={24} />
        </span>
        <span className="font-bold">{t(mr.failedTitle, { card })}</span>
        <span className="k max-w-[420px]">{t(mr.failedText)}</span>
        <Button size="sm" variant="primary" onClick={onRetry}>
          {t(mr.retry)}
        </Button>
        {reference ? <span className="k mono">{t(mr.failedRef, { ref: reference })}</span> : null}
      </div>
    </Card>
  )
}

function Stat({ value, label }: { value: ReactNode; label: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="er-num">{value}</span>
      <span className="k">{label}</span>
    </div>
  )
}

/** A stacked bar with its legend: became paid, canceled, still trialing… */
function Stack({ parts, label }: { parts: { color: string; value: number; text: string }[]; label: string }) {
  const shown = parts.filter((p) => p.value > 0)
  return (
    <div className="flex flex-col gap-2.5">
      {shown.length ? (
        <div className="er-stack" role="img" aria-label={label}>
          {shown.map((p, i) => (
            <span key={i} style={{ flex: p.value, background: p.color }} />
          ))}
        </div>
      ) : null}
      <div className="er-legend">
        {parts.map((p, i) => (
          <span key={i}>
            <span className="er-sw" style={{ background: p.color }} aria-hidden="true" />
            {p.text}
          </span>
        ))}
      </div>
    </div>
  )
}

/**
 * Earn › Overview once Earn is live (EarnOverview): the period, the
 * numbers, MRR over 12 months and what needs a person, the movement and the
 * tiers, trials, conversions and cancellations, free-to-paid by source,
 * upgrades by email and the latest events. Each card that fails says so on
 * its own.
 */
export function RevenueOverview() {
  const t = useT()
  const { slug } = useWorkspace()
  const [period, setPeriod] = useState<Period>('30d')
  const q = $api.useQuery('get', `${EARN}/revenue`, { params: { path: { workspace: slug }, query: { period } } }, { placeholderData: keepPreviousData })
  const status = useEarnStatus().data
  if (q.error && !q.data) {
    return <Failed card="" title={t(mr.tMrr)} onRetry={() => void q.refetch()} reference={problemReference(q.error)} />
  }
  if (!q.data) return <Loading />
  const o = q.data
  const provider = status?.providerName || 'Stripe'
  const retry = () => void q.refetch()
  const failed = (card: string) => o.failed.includes(card)
  return (
    <div className="flex flex-col gap-5" aria-busy={q.isFetching || undefined}>
      {o.readOnly ? <ReadOnly o={o} /> : null}
      <div className="flex flex-wrap items-center justify-between gap-2.5">
        <span className="k flex-[1_1_420px]">
          {t(mr.periodLine, { period: o.period, from: dayMonthDate(o.from), to: dayMonthYear(o.to), currency: o.currency.toUpperCase() })}
        </span>
        <Segmented<Period>
          value={period}
          onChange={setPeriod}
          label={t(mr.periodLabel)}
          items={[
            { value: '30d', label: t(mr.p30) },
            { value: '90d', label: t(mr.p90) },
            { value: '12m', label: t(mr.p12) },
          ]}
        />
      </div>
      <Tiles o={o} />
      <div className="er-row">
        {failed('chart') ? (
          <div style={{ flex: '2 1 520px', minWidth: 0 }}>
            <Failed card="chart" title={t(mr.chartTitle)} onRetry={retry} reference={o.reference} />
          </div>
        ) : (
          <MrrCard o={o} provider={provider} />
        )}
        <div className="flex flex-[1_1_320px] flex-col gap-4" style={{ minWidth: 0 }}>
          {failed('needs') ? <Failed card="needs" title={t(mr.needsTitle, { count: 0 })} onRetry={retry} reference={o.reference} /> : <Needs o={o} provider={provider} />}
          {o.readOnly ? <CanDo provider={provider} /> : null}
        </div>
      </div>
      <div className="er-grid">
        {failed('movement') ? <Failed card="movement" title={t(mr.moveTitle, { period: o.period })} onRetry={retry} reference={o.reference} /> : <MovementCard o={o} />}
        {failed('tiers') ? <Failed card="tiers" title={t(mr.tierTitle)} onRetry={retry} reference={o.reference} /> : <Tiers o={o} />}
      </div>
      <div className="er-grid">
        <div className="er-col">
          {failed('trials') ? <Failed card="trials" title={t(mr.trialsTitle, { period: o.period })} onRetry={retry} reference={o.reference} /> : <Trials o={o} />}
          {failed('conversions') ? <Failed card="conversions" title={t(mr.convTitle)} onRetry={retry} reference={o.reference} /> : <Conversions c={o.conversions} period={o.period} />}
        </div>
        {failed('cancellations') ? <Failed card="cancellations" title={t(mr.cancelTitle, { period: o.period })} onRetry={retry} reference={o.reference} /> : <Cancellations o={o} />}
      </div>
      {failed('freeToPaid') ? <Failed card="freeToPaid" title={t(mr.ftpTitle, { period: o.period })} onRetry={retry} reference={o.reference} /> : <FreeToPaid o={o} />}
      <div className="er-grid">
        {failed('upgrades') ? <Failed card="upgrades" title={t(mr.upTitle, { period: o.period })} onRetry={retry} reference={o.reference} /> : <Upgrades o={o} />}
        {failed('activity') ? <Failed card="activity" title={t(mr.actTitle)} onRetry={retry} reference={o.reference} /> : <Activity o={o} provider={provider} />}
      </div>
    </div>
  )
}

function Loading() {
  const t = useT()
  return (
    <div className="flex flex-col gap-5" aria-busy="true" aria-label={t(mr.loading)}>
      <div className="er-tiles">
        {[0, 1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="card er-stat">
            <Skeleton w={80} h={12} />
            <Skeleton w={100} h={24} className="mt-1" />
            <Skeleton w={130} h={10} className="mt-1" />
          </div>
        ))}
      </div>
      <div className="card er-pad">
        <Skeleton w="40%" h={18} />
        <Skeleton w="100%" h={240} className="mt-4" />
      </div>
    </div>
  )
}

/** Read-only · Aisha Bello, Viewer: who can change Earn. */
function ReadOnly({ o }: { o: Overview }) {
  const t = useT()
  const { me: wme } = useWorkspace()
  const user = useMe().data?.user
  const role = roleName({ key: wme.workspace.roleKey, name: wme.workspace.roleName, builtin: true })
  const perms = ['earn:edit', 'earn:manage', 'payments:manage', 'sponsors:manage']
  const permNodes = perms.map((p) => (
    <span key={p} className="mono">
      {p}
    </span>
  ))
  return (
    <div className="note n" role="note">
      <span style={{ color: 'var(--ink-2)', display: 'inline-flex', flex: 'none', marginTop: 1 }}>
        <Icon name="info" size={16} />
      </span>
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(mr.roTitle, { name: user?.name || user?.email || '', role })}</span>
        <div>
          <T
            msg={o.helpers.length ? mr.roText : mr.roTextNoHelpers}
            values={{
              perms: <>{joinNodes(permNodes)}</>,
              helpers: orList(o.helpers),
            }}
          />
        </div>
      </div>
    </div>
  )
}

/** "a, b, c or d" with nodes. */
function joinNodes(nodes: ReactNode[]): ReactNode {
  const words = orList(nodes.map((_, i) => `§${i}§`))
  return words.split(/§(\d+)§/).map((part, i) => (i % 2 === 1 ? nodes[Number(part)] : part))
}

/** What you can do (read-only). */
function CanDo({ provider }: { provider: string }) {
  const t = useT()
  const ok = (text: ReactNode) => (
    <li className="flex items-start gap-2 text-[13px]">
      <span style={{ color: 'var(--ok-fg)', display: 'inline-flex', marginTop: 2 }}>
        <Icon name="check" size={14} />
      </span>
      <span>{text}</span>
    </li>
  )
  const no = (text: ReactNode) => (
    <li className="flex items-start gap-2 text-[13px]">
      <span style={{ color: 'var(--bad-fg)', display: 'inline-flex', marginTop: 2 }}>
        <Icon name="close" size={13} />
      </span>
      <span>{text}</span>
    </li>
  )
  const mono = (p: string) => <span className="mono">{p}</span>
  return (
    <Card title={t(mr.canTitle)}>
      <ul className="er-pad m-0 flex list-none flex-col gap-2">
        {ok(t(mr.canSee))}
        {ok(t(mr.canMembers))}
        {ok(t(mr.canPlans))}
        {no(<T msg={mr.canEdit} values={{ perm: mono('earn:edit') }} />)}
        {no(<T msg={mr.canManage} values={{ perm: mono('earn:manage') }} />)}
        {no(<T msg={mr.canPayments} values={{ provider, perm: mono('payments:manage') }} />)}
      </ul>
    </Card>
  )
}

function Tiles({ o }: { o: Overview }) {
  const t = useT()
  const x = o.tiles
  const cur = o.currency
  const shown = o.shownFrom ? t(mr.shownFrom, { date: dayMonthDate(o.shownFrom) }) : ''
  return (
    <div className="er-tiles">
      <div className="card er-stat">
        <span className="k er-k">
          {t(mr.tMrr)}
          <Def label={t(mr.defMrrLabel)} text={t(mr.defMrr, { currency: cur.toUpperCase() })} />
        </span>
        <span className="v">{money(x.mrrCents, cur, 'never')}</span>
        <span className="s">{o.tooEarly ? ' ' : <Delta cents={x.mrrDeltaCents} pctValue={x.mrrDeltaPct} currency={cur} />}</span>
      </div>
      <div className="card er-stat">
        <span className="k er-k">
          {t(mr.tArr)}
          <Def label={t(mr.defArrLabel)} text={t(mr.defArr)} />
        </span>
        <span className="v">{money(x.arrCents, cur, 'never')}</span>
        <span className="s">{t(mr.tArrSub)}</span>
      </div>
      <div className="card er-stat">
        <span className="k er-k">
          {t(mr.tMembers)}
          <Def label={t(mr.defMembersLabel)} text={t(mr.defMembers)} />
        </span>
        <span className="v">{count(x.members)}</span>
        <span className="s">
          <CountDelta n={x.membersDelta} /> · {t(mr.tMembersSub, { new: count(x.new), churned: count(x.churned) })}
        </span>
      </div>
      <div className="card er-stat">
        <span className="k er-k">
          {t(mr.tChurn)}
          <Def label={t(mr.defChurnLabel)} text={t(mr.defChurn)} />
        </span>
        <span className="v">{x.churnRate === undefined || x.churnRate === null ? '—' : pct(x.churnRate, 1)}</span>
        <span className="s">{o.tooEarly ? shown : t(mr.tChurnSub, { churned: count(x.churned), base: count(x.churnBase) })}</span>
      </div>
      <div className="card er-stat">
        <span className="k er-k">
          {t(mr.tFree)}
          <Def label={t(mr.defFreeLabel)} text={t(mr.defFree)} />
        </span>
        <span className="v">{x.freeToPaidRate === undefined || x.freeToPaidRate === null ? '—' : pct(x.freeToPaidRate, 2)}</span>
        <span className="s">{o.tooEarly ? shown : t(mr.tFreeSub, { paid: count(x.freeToPaid), base: count(x.freeBase) })}</span>
      </div>
      <div className="card er-stat">
        <span className="k">{t(mr.tOther)}</span>
        <span className="v">{money(x.giftsCents + x.sponsorsCents, cur, 'never')}</span>
        <span className="s">{t(mr.tOtherSub, { gifts: money(x.giftsCents, cur, 'never'), sponsors: money(x.sponsorsCents, cur, 'never') })}</span>
      </div>
    </div>
  )
}

/** MRR · last 12 months, or why it's too early for it. */
function MrrCard({ o, provider }: { o: Overview; provider: string }) {
  const t = useT()
  const c = o.chart
  const cur = o.currency
  const style = { flex: '2 1 520px', minWidth: 0 }
  if (o.tooEarly) {
    const days = o.dataSince ? Math.max(1, Math.round((new Date(o.to).getTime() - new Date(o.dataSince).getTime()) / 86_400_000)) : 0
    return (
      <Card title={t(mr.chartTitle)} style={style}>
        <div className="er-pad flex flex-col gap-3">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1.5">
            <span className="er-big">{money(o.tiles.mrrCents, cur, 'never')}</span>
            <span className="k">{t(mr.soFar, { members: o.tiles.members, days })}</span>
          </div>
          <div className="er-early">
            <svg width="120" height="44" viewBox="0 0 120 44" aria-hidden="true">
              <path d="M2 40 L30 34 L58 30" fill="none" stroke="var(--accent)" strokeWidth="2.4" strokeLinecap="round" />
              <path d="M58 30 L86 22 L118 12" fill="none" stroke="var(--line-hover)" strokeWidth="2" strokeDasharray="4 4" strokeLinecap="round" />
            </svg>
            <span className="font-bold">{t(mr.tooEarlyTitle)}</span>
            <span className="k max-w-[440px]">
              {o.connectedAt && o.account && o.shownFrom
                ? t(mr.tooEarlyText, { provider, connected: dayMonthDate(o.connectedAt), account: o.account, start: dayMonthDate(o.shownFrom) })
                : t(mr.tooEarlyTextNoAccount, { start: o.shownFrom ? dayMonthDate(o.shownFrom) : '' })}
            </span>
          </div>
        </div>
      </Card>
    )
  }
  const moved = c.marks.find((m) => m.kind === 'moved')
  const platform = moved ? platformName(moved.name) : ''
  const first = c.points[0]
  const last = c.points[c.points.length - 1]
  const desc = first && last
    ? `${t(mr.chartDesc, { from: money(first.mrrCents, cur, 'never'), fromDate: dayMonthYear(first.at), to: money(last.mrrCents, cur, 'never'), toDate: dayMonthYear(last.at) })}${
        c.historyUntil ? ` ${t(mr.chartDescHistory, { date: dayMonthDate(c.historyUntil), platform })}` : ''
      }`
    : ''
  return (
    <Card
      title={
        <>
          {t(mr.chartTitle)} <Def label={t(mr.defMrrLabel)} text={t(mr.defMrr, { currency: cur.toUpperCase() })} />
        </>
      }
      aside={first ? <span className="k">{t(mr.chartDay, { day: new Date(first.at).getDate() })}</span> : null}
      style={style}
    >
      <div className="er-pad flex flex-col gap-3">
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1.5">
          <span className="er-big">{money(c.endCents, cur, 'never')}</span>
          <Delta cents={c.deltaCents} pctValue={c.deltaPct} currency={cur} />
          <span className="k">{t(mr.chartSince, { date: dayMonthYear(c.since), arr: money(o.tiles.arrCents, cur, 'never') })}</span>
        </div>
        <MrrChart chart={c} currency={cur} desc={desc} />
        {c.marks.length || c.historyUntil ? (
          <div className="er-legend">
            {c.marks.map((m) => (
              <span key={`${m.kind}-${m.n}`}>
                <span className="er-mnum" aria-hidden="true">
                  {m.n}
                </span>
                {m.kind === 'moved' ? t(mr.markMoved, { platform: platformName(m.name), date: dayMonthDate(m.at) }) : t(mr.markOffer, { name: m.name, date: dayMonthDate(m.at) })}
              </span>
            ))}
            {c.historyUntil ? (
              <>
                <span>
                  <svg width="22" height="6" aria-hidden="true">
                    <line x1="0" y1="3" x2="22" y2="3" stroke="#0F766E" strokeWidth="2" strokeDasharray="5 4" />
                  </svg>
                  {t(mr.fromHistory, { provider })}
                </span>
                <span>
                  <svg width="22" height="6" aria-hidden="true">
                    <line x1="0" y1="3" x2="22" y2="3" stroke="#0F766E" strokeWidth="2.4" />
                  </svg>
                  {t(mr.sinceMove)}
                </span>
              </>
            ) : null}
          </div>
        ) : null}
        {c.historyUntil ? (
          <span className="k">
            {o.account
              ? t(mr.historyNote, { date: dayMonthDate(c.historyUntil), account: o.account, provider, platform })
              : t(mr.historyNoteNoAccount, { date: dayMonthDate(c.historyUntil), provider })}
          </span>
        ) : null}
      </div>
    </Card>
  )
}

/** Needs you: failed payments, offers near their end, sponsor creatives and payments, member emails. */
function Needs({ o, provider }: { o: Overview; provider: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const ro = o.readOnly
  const params = { workspace: slug }
  const icon = (kind: 'clock' | 'info', bg: string, fg: string) => (
    <span className="ic" style={{ background: bg, color: fg }} aria-hidden="true">
      <Icon name={kind} size={14} />
    </span>
  )
  const b = { b: (c: ReactNode) => <b>{c}</b>, mono: (c: ReactNode) => <span className="mono font-semibold">{c}</span> }
  return (
    <Card title={t(mr.needsTitle, { count: o.needs.length })}>
      <div>
        {o.needs.length === 0 ? <p className="er-pad m-0 k">{t(mr.needsNone)}</p> : null}
        {o.needs.map((n, i) => {
          let ic: ReactNode = icon('clock', 'var(--warn-bg)', 'var(--warn-fg)')
          let text: ReactNode = null
          let act: ReactNode = null
          // Read-only people get a link after the sentence, the others a button.
          const linkClass = ro ? 'whitespace-nowrap underline' : 'btn sm'
          switch (n.kind) {
            case 'past_due':
              text = <T msg={mr.needPastDue} values={{ count: n.count ?? 0, provider, ...b }} />
              act = (
                <Link to="/w/$workspace/earn/members" params={params} search={{ status: 'past_due' }} className={linkClass}>
                  {ro ? t(mr.viewPastDue) : t(mr.viewPastDue)}
                </Link>
              )
              break
            case 'offer': {
              ic = icon('info', 'var(--info-bg)', 'var(--info-fg)')
              const date = n.endsAt ? dayMonthDate(n.endsAt) : ''
              const msg = n.limit && n.endsAt ? mr.needOfferLimitEnds : n.limit ? mr.needOfferLimit : mr.needOfferEnds
              text = <T msg={msg} values={{ name: n.name ?? '', used: n.used ?? 0, limit: n.limit ?? 0, date, ...b }} />
              act = (
                <Link to="/w/$workspace/earn/plans" params={params} className={linkClass}>
                  {ro ? t(mr.viewOffer) : t(mr.openOffer)}
                </Link>
              )
              break
            }
            case 'creative':
              text = (
                <T
                  msg={n.overdue ? mr.needCreativeLate : mr.needCreative}
                  values={{ list: n.list ?? '', issue: n.issue ?? 0, sponsor: n.sponsor ?? '', ends: ends(n.sponsor ?? ''), date: n.due ? shortDate(n.due) : '', ...b }}
                />
              )
              act = (
                <Link to="/w/$workspace/earn/sponsors" params={params} search={{ booking: n.targetId }} className={linkClass}>
                  {ro ? t(mr.viewBooking) : t(mr.openBooking)}
                </Link>
              )
              break
            case 'sponsor_payment':
              ic = icon('info', 'var(--neutral-bg)', 'var(--neutral-fg)')
              text = <T msg={n.overdue ? mr.needPaymentLate : mr.needPayment} values={{ sponsor: n.sponsor ?? '', issue: n.issue ?? 0, date: n.due ? day(n.due) : '', ...b }} />
              act = (
                <Link to="/w/$workspace/earn/sponsors" params={params} search={{ booking: n.targetId }} className={linkClass}>
                  {ro ? t(mr.viewBooking) : t(mr.openBooking)}
                </Link>
              )
              break
            case 'member_email':
              text = <T msg={mr.needEmail} values={{ template: n.template ?? '', ...b }} />
              act = n.targetId ? (
                <Link to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: n.targetId }} className={linkClass}>
                  {ro ? t(mr.viewTemplate) : t(mr.openTemplate)}
                </Link>
              ) : null
              break
          }
          return (
            <div key={i} className="er-need">
              {ic}
              <div className="tx">
                {ro ? (
                  <span>
                    {text}
                    {act ? <> {act}</> : null}
                  </span>
                ) : (
                  <>
                    <span>{text}</span>
                    {act ? <div>{act}</div> : null}
                  </>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </Card>
  )
}

/** MRR movement: a waterfall or a table; "so far" while it's too early. */
function MovementCard({ o }: { o: Overview }) {
  const t = useT()
  const [view, setView] = useState<'chart' | 'table'>('chart')
  const m = o.movement
  const cur = o.currency
  if (o.tooEarly) {
    const kv: [string, ReactNode][] = [
      [t(mr.kvNew), t(mr.soFarNew, { amount: money(m.new.cents, cur, 'never'), count: m.new.count })],
      [t(mr.kvExpansion), money(m.expansion.cents, cur, 'never')],
      [t(mr.kvContraction), money(Math.abs(m.contraction.cents), cur, 'never')],
      [t(mr.kvChurned), money(Math.abs(m.churned.cents), cur, 'never')],
      [t(mr.kvMembers), t(mr.soFarMembers, { members: count(o.tiles.members), trialing: count(o.tiles.trialing) })],
      [t(mr.kvChurn), o.shownFrom ? t(mr.shownFrom, { date: dayMonthDate(o.shownFrom) }) : '—'],
      [t(mr.kvFree), o.shownFrom ? t(mr.shownFrom, { date: dayMonthDate(o.shownFrom) }) : '—'],
    ]
    return (
      <Card title={t(mr.moveSoFar)}>
        <div className="er-pad flex flex-col gap-2.5">
          <dl className="kv m-0">
            {kv.map(([k, v]) => (
              <div key={k} className="contents">
                <dt>{k}</dt>
                <dd>{v}</dd>
              </div>
            ))}
          </dl>
          <span className="k">{o.shownFrom ? t(mr.soFarNote, { date: dayMonthDate(o.shownFrom) }) : null}</span>
        </div>
      </Card>
    )
  }
  const bars = moveBars(m)
  const name = (k: string) =>
    ({ start: t(mr.mStart), new: t(mr.mNew), reactivated: t(mr.mReactivated), expansion: t(mr.mExpansion), contraction: t(mr.mContraction), churned: t(mr.mChurned), end: t(mr.mEnd) })[k] ??
    k
  const signedMoney = (c: number) => (c < 0 ? `−${money(-c, cur, 'never')}` : `+${money(c, cur, 'never')}`)
  const desc = t(mr.moveDesc, {
    from: dayMonthDate(o.from),
    to: dayMonthDate(o.to),
    start: money(m.fromCents, cur, 'never'),
    startMembers: m.fromMembers,
    end: money(m.toCents, cur, 'never'),
    endMembers: m.toMembers,
    parts: bars
      .filter((b) => b.key !== 'start' && b.key !== 'end')
      .map((b) => t(mr.movePart, { name: name(b.key).toLowerCase(), amount: signedMoney(b.cents), count: count(b.count) }))
      .join('; '),
  })
  const levels = [m.fromCents, m.fromCents + m.new.cents + m.reactivated.cents, m.toCents]
  const lo = Math.min(...levels)
  const base = lo > 0 ? Math.floor(lo / 100000) * 100000 : 0
  return (
    <Card
      title={
        <>
          {t(mr.moveTitle, { period: o.period })} <Def label={t(mr.defMoveLabel)} text={t(mr.defMove)} />
        </>
      }
      sub={t(mr.moveRange, { from: money(m.fromCents, cur, 'never'), fromDate: dayMonthDate(o.from), to: money(m.toCents, cur, 'never'), toDate: dayMonthDate(o.to) })}
      aside={
        <Segmented<'chart' | 'table'>
          value={view}
          onChange={setView}
          label={t(mr.viewLabel)}
          items={[
            { value: 'chart', label: t(mr.viewChart) },
            { value: 'table', label: t(mr.viewTable) },
          ]}
        />
      }
    >
      <div className="er-pad flex flex-col gap-2.5">
        {view === 'chart' ? (
          <MovementChart movement={m} currency={cur} desc={desc} />
        ) : (
          <div className="overflow-x-auto">
            <table className="tbl" aria-label={t(mr.moveTitle, { period: o.period })}>
              <thead>
                <tr>
                  <th scope="col">{t(mr.colKind)}</th>
                  <th scope="col" className="r">
                    {t(mr.colMembers)}
                  </th>
                  <th scope="col" className="r">
                    {t(mr.colMrr)}
                  </th>
                </tr>
              </thead>
              <tbody>
                {bars.map((b) => (
                  <tr key={b.key}>
                    <td>
                      <span className="inline-flex items-center gap-2">
                        <span className="er-sw" style={{ background: b.key === 'start' || b.key === 'end' ? moveColors.ends : moveColors[b.key as keyof typeof moveColors] }} aria-hidden="true" />
                        {name(b.key)}
                      </span>
                    </td>
                    <td className="r num">{count(b.count)}</td>
                    <td className="r num">{b.key === 'start' || b.key === 'end' ? money(b.cents, cur, 'never') : signedMoney(b.cents)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <span className="k">
          {base > 0 ? t(mr.moveNote, { base: money(base, cur, 'never') }) : null}{' '}
          {m.reactivated.count > 0 ? t(mr.moveNoteReact, { count: m.reactivated.count, amount: signedMoney(m.reactivated.cents) }) : t(mr.moveNoteReactNone)}
        </span>
      </div>
    </Card>
  )
}

function tierNote(t: TT, r: Schemas['RevenueTierRow'], cur: string): string {
  if (r.offers.length) {
    return r.offers
      .map((x) =>
        x.duration === 'once'
          ? t(mr.offerOnce, { count: count(x.members), offer: x.name, amount: money(x.paidCents, cur), period: r.period })
          : t(mr.offerOn, { count: count(x.members), offer: x.name, amount: money(x.paidCents, cur), period: r.period }),
      )
      .join(' · ')
  }
  if (r.includes.length) return t(mr.priceIncludes, { amount: money(r.priceCents, cur), period: r.period, tiers: andList(r.includes) })
  return t(mr.priceOnly, { amount: money(r.priceCents, cur), period: r.period })
}

/** By tier and period: paying members and MRR today. */
function Tiers({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const cur = o.currency
  const rows = o.tiers
  const total = rows.reduce((s, r) => ({ members: s.members + r.members, mrr: s.mrr + r.mrrCents }), { members: 0, mrr: 0 })
  const name = (r: Schemas['RevenueTierRow']) => t(mr.tierPeriod, { tier: r.tier, period: r.period })
  return (
    <Card
      title={t(mr.tierTitle)}
      sub={t(mr.tierSub)}
      aside={
        <Link to="/w/$workspace/earn/plans" params={{ workspace: slug }} className="er-lnk">
          {t(mr.plansLink)}
        </Link>
      }
    >
      {rows.length === 0 ? (
        <p className="er-pad m-0 k">{t(mr.noTiers)}</p>
      ) : (
        <>
          <div className="max-md:hidden">
            <div className="overflow-x-auto">
              <table className="tbl er-tbl" aria-label={t(mr.tierTitle)}>
                <thead>
                  <tr>
                    <th scope="col">{t(mr.colTier)}</th>
                    <th scope="col" className="r">
                      {t(mr.colPaid)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colMrr)}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={`${r.tierId}:${r.period}`}>
                      <td>
                        <div className="er-tline">
                          <b>{name(r)}</b>
                          <span className="k">{tierNote(t, r, cur)}</span>
                        </div>
                      </td>
                      <td className="r num">{count(r.members)}</td>
                      <td className="r num">{money(r.mrrCents, cur, 'never')}</td>
                    </tr>
                  ))}
                  <tr className="total">
                    <td>
                      <b className="font-bold">{t(mr.total)}</b>
                    </td>
                    <td className="r num">
                      <b>{count(total.members)}</b>
                    </td>
                    <td className="r num">
                      <b>{money(total.mrr, cur, 'never')}</b>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <div className="hidden max-md:flex max-md:flex-col">
            {rows.map((r) => (
              <div key={`${r.tierId}:${r.period}`} className="er-lrow">
                <span className="lt">
                  <b className="font-semibold">{name(r)}</b>
                  <span className="num whitespace-nowrap font-semibold">{money(r.mrrCents, cur, 'never')}</span>
                </span>
                <span className="k">
                  {t(mr.tierMembers, { count: r.members })} · {tierNote(t, r, cur)}
                </span>
              </div>
            ))}
            <div className="er-lrow">
              <span className="lt">
                <b className="font-semibold">{t(mr.total)}</b>
                <span className="num whitespace-nowrap font-semibold">{money(total.mrr, cur, 'never')}</span>
              </span>
              <span className="k">{t(mr.tierMembers, { count: total.members })}</span>
            </div>
          </div>
          <div className="er-pad" style={{ borderTop: '1px solid var(--line-2)' }}>
            <Stack
              label={t(mr.tierBarLabel, { parts: rows.map((r) => t(mr.tierLegend, { tier: r.tier, period: r.period, amount: money(r.mrrCents, cur, 'never') })).join(', ') })}
              parts={rows.map((r, i) => ({ color: tierColors[i % tierColors.length]!, value: r.mrrCents, text: t(mr.tierLegend, { tier: r.tier, period: r.period, amount: money(r.mrrCents, cur, 'never') }) }))}
            />
          </div>
        </>
      )}
    </Card>
  )
}

/** Trials: started, what became of them, and those ending soon. */
function Trials({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const tr = o.trials
  const rate = tr.ended > 0 ? Math.round((tr.becamePaid / tr.ended) * 100) : null
  return (
    <Card title={t(mr.trialsTitle, { period: o.period })} sub={t(mr.trialsSub, { count: count(tr.started) })}>
      <div className="er-pad flex flex-col gap-3.5">
        <div className="flex flex-wrap gap-x-7 gap-y-3">
          <Stat value={count(tr.trialing)} label={t(mr.trialing)} />
          <Stat value={rate === null ? '—' : `${rate}%`} label={rate === null ? t(mr.noEnded) : t(mr.becamePaidRate, { paid: count(tr.becamePaid), ended: count(tr.ended) })} />
        </div>
        <Stack
          label={t(mr.trialsBar, { paid: tr.becamePaid, canceled: tr.canceled, trialing: tr.trialing })}
          parts={[
            { color: '#2a78d6', value: tr.becamePaid, text: t(mr.becamePaid, { count: count(tr.becamePaid) }) },
            { color: '#8b94a3', value: tr.canceled, text: t(mr.canceled, { count: count(tr.canceled) }) },
            { color: '#1baf7a', value: tr.trialing, text: t(mr.stillTrialing, { count: count(tr.trialing) }) },
          ]}
        />
        <div className="flex flex-wrap items-center justify-between gap-2.5">
          <span className="text-[13px]">
            {tr.endingFrom && tr.endingSoon > 0
              ? t(mr.endingFrom, { count: count(tr.endingSoon), from: count(tr.endingFrom.count), email: tr.endingFrom.name })
              : t(mr.ending, { count: tr.endingSoon })}
          </span>
          <Link to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ status: 'trialing' }} className="er-lnk">
            {t(mr.seeTrialing)}
          </Link>
        </div>
      </div>
    </Card>
  )
}

/** Product conversions: the conversion API's results (not MRR). */
export function Conversions({ c, period, latest }: { c: Schemas['RevenueConversions']; period: string; latest?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const cur = c.currency
  return (
    <Card
      title={t(mr.convTitle)}
      aside={
        <Link to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'attribution' }} className="er-lnk">
          {t(mr.convApi)}
        </Link>
      }
    >
      <div className="er-pad flex flex-col gap-3">
        {c.total === 0 ? (
          <span className="k">{t(mr.convNone)}</span>
        ) : (
          <>
            <div className="flex flex-wrap gap-x-7 gap-y-3">
              <Stat value={count(c.count)} label={c.since ? t(mr.convCount, { date: dayMonthDate(c.since) }) : t(mr.convCountPeriod, { period })} />
              <Stat value={money(c.valueCents, cur, 'never')} label={t(mr.inValue)} />
            </div>
            <div className="flex flex-col">
              {c.byEmail.map((r) => (
                <div key={r.id} className="er-vline">
                  <span>
                    {r.kind === 'campaign' ? (
                      <Link to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: r.id }}>
                        {r.name}
                      </Link>
                    ) : (
                      <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: r.id }}>
                        {r.name}
                      </Link>
                    )}{' '}
                    <span className="k">{t(mr.byTokenEmail, { token: count(r.byToken), email: count(r.byEmail) })}</span>
                  </span>
                  <span className="num whitespace-nowrap font-semibold">{t(mr.countValue, { count: count(r.count), value: money(r.valueCents, cur, 'never') })}</span>
                </div>
              ))}
              {c.uncredited.count > 0 ? (
                <div className="er-vline">
                  <span>{t(mr.notCredited)}</span>
                  <span className="num whitespace-nowrap font-semibold">{t(mr.countValue, { count: count(c.uncredited.count), value: money(c.uncredited.cents, cur, 'never') })}</span>
                </div>
              ) : null}
            </div>
            <span className="k">
              {c.name && c.typicalCents && c.sender
                ? t(mr.convNote, { name: c.name, value: money(c.typicalCents, cur), sender: c.sender })
                : t(mr.convNoteNoName)}
              {c.otherCurrency > 0 ? ` ${t(mr.otherCurrency, { count: c.otherCurrency })}` : null}
            </span>
            {latest && c.latest.length ? (
              <div className="flex flex-col gap-1.5 pt-3" style={{ borderTop: '1px solid var(--line-2)' }}>
                <span className="text-[13px] font-semibold">{t(mr.latest)}</span>
                {c.latest.map((l, i) => (
                  <div key={i} className="flex flex-wrap items-center justify-between gap-2.5 text-[13px]">
                    <span>
                      <span className="mono k">{isToday(l.at) ? timeOf(l.at) : dayMonthDate(l.at)}</span>{' '}
                      {t(mr.latestLine, { value: money(l.valueCents, l.currency, 'always'), credited: l.credited || t(mr.notCreditedShort) })}
                    </span>
                    <span className="k">{l.matched === 'click_token' ? t(mr.byToken) : l.matched === 'email' ? t(mr.byEmail) : t(mr.byNone)}</span>
                  </div>
                ))}
              </div>
            ) : null}
          </>
        )}
      </div>
    </Card>
  )
}

function isToday(at: string): boolean {
  return dayMonthYear(at) === dayMonthYear(new Date())
}

function timeOf(at: string): string {
  const d = new Date(at)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

/** Cancellations: the cancel page's visits, decisions and reasons. */
function Cancellations({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const c = o.cancellations
  const maxReason = Math.max(1, ...c.reasons.map((r) => r.count))
  return (
    <Card
      title={t(mr.cancelTitle, { period: o.period })}
      aside={
        c.offerId ? (
          <Link to="/w/$workspace/earn/plans" params={{ workspace: slug }} className="er-lnk">
            {t(mr.retentionLink)}
          </Link>
        ) : null
      }
    >
      <div className="er-pad flex flex-col gap-3.5">
        {c.visits === 0 ? (
          <span className="k">{t(mr.cancelNone)}</span>
        ) : (
          <>
            <div className="flex flex-wrap gap-x-7 gap-y-3">
              <Stat value={count(c.visits)} label={t(mr.visits)} />
              <Stat value={count(c.kept)} label={c.keptOffer && c.keptTier ? t(mr.keptWith, { tier: c.keptTier, offer: c.keptOffer }) : t(mr.keptPlain)} />
            </div>
            <Stack
              label={t(mr.cancelBar, { kept: c.kept, continued: c.continued, undecided: c.undecided })}
              parts={[
                { color: '#2a78d6', value: c.kept, text: c.keptTier ? t(mr.keptOffer, { tier: c.keptTier, count: count(c.kept) }) : t(mr.keptOfferPlain, { count: count(c.kept) }) },
                { color: '#eb6834', value: c.continued, text: t(mr.wentOn, { count: count(c.continued) }) },
                { color: '#8b94a3', value: c.undecided, text: t(mr.undecided, { count: count(c.undecided) }) },
              ]}
            />
            <div className="flex flex-col gap-2">
              <span className="text-[13px] font-semibold">{t(mr.reasonsTitle)}</span>
              <div className="flex flex-col gap-2.5">
                {c.reasons.map((r) => (
                  <div key={r.reason} className="flex flex-col gap-1">
                    <div className="flex justify-between gap-2.5 text-[13px]">
                      <span>{t(mr.reason, { reason: r.reason })}</span>
                      <span className="num font-semibold">{count(r.count)}</span>
                    </div>
                    <div className="er-bar" aria-hidden="true">
                      <i style={{ width: `${((r.count / maxReason) * 100).toFixed(1)}%` }} />
                    </div>
                  </div>
                ))}
              </div>
            </div>
            <span className="k">{t(mr.reasonNote)}</span>
          </>
        )}
      </div>
    </Card>
  )
}

/** A source of free subscribers in words. */
function sourceName(t: TT, r: Schemas['RevenueSourceRow']): string {
  switch (r.kind) {
    case 'form':
      return r.name ? t(mr.srcForm, { name: r.name }) : t(mr.srcFormless)
    case 'subscribe_page':
      return t(mr.srcSubscribePage)
    case 'import':
    case 'carried':
      return t(mr.srcImport, { name: r.name || 'CSV' })
    case 'api_key':
      return r.name ? t(mr.srcApi, { name: r.name }) : t(mr.srcApiless)
    case 'person':
      return t(mr.srcPerson)
    case 'preference_center':
      return t(mr.srcPreferences)
    case 'other':
      return t(mr.srcOther)
    case 'integration':
      return r.name || r.kind
  }
  return r.name || t(mr.srcUnknown)
}

/** Free-to-paid by source. */
function FreeToPaid({ o }: { o: Overview }) {
  const t = useT()
  const f = o.freeToPaid
  const rate = (r: Schemas['RevenueSourceRow']) => (r.tooFew ? <span className="k">{t(mr.tooFew)}</span> : r.rate === undefined || r.rate === null ? '—' : pct(r.rate, 2))
  return (
    <Card
      title={
        <>
          {t(mr.ftpTitle, { period: o.period })} <Def label={t(mr.defFreeLabel)} text={t(mr.defFree)} />
        </>
      }
      sub={t(mr.ftpSub)}
    >
      {f.rows.length === 0 ? (
        <p className="er-pad m-0 k">{t(mr.ftpNone)}</p>
      ) : (
        <>
          <div className="max-md:hidden">
            <div className="overflow-x-auto">
              <table className="tbl er-tbl" aria-label={t(mr.ftpTitle, { period: o.period })}>
                <thead>
                  <tr>
                    <th scope="col">{t(mr.colSource)}</th>
                    <th scope="col" className="r">
                      {t(mr.colStart)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colJoined)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colFree)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colBecame)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colRate)}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {f.rows.map((r) => (
                    <tr key={r.key}>
                      <td>
                        <span className="font-semibold">{sourceName(t, r)}</span>
                      </td>
                      <td className="r num">{count(r.atStart)}</td>
                      <td className="r num">{count(r.joined)}</td>
                      <td className="r num">{count(r.base)}</td>
                      <td className="r num">{count(r.paid)}</td>
                      <td className="r num">{rate(r)}</td>
                    </tr>
                  ))}
                  <tr className="total">
                    <td>
                      <b className="font-bold">{t(mr.total)}</b>
                    </td>
                    <td className="r num">
                      <b>{count(f.total.atStart)}</b>
                    </td>
                    <td className="r num">
                      <b>{count(f.total.joined)}</b>
                    </td>
                    <td className="r num">
                      <b>{count(f.total.base)}</b>
                    </td>
                    <td className="r num">
                      <b>{count(f.total.paid)}</b>
                    </td>
                    <td className="r num">
                      <b>{f.total.rate === undefined || f.total.rate === null ? '—' : pct(f.total.rate, 2)}</b>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <div className="hidden max-md:flex max-md:flex-col">
            {[...f.rows, f.total].map((r) => (
              <div key={r.key} className="er-lrow">
                <span className="lt">
                  <b className="font-semibold">{r.key === 'total' ? t(mr.total) : sourceName(t, r)}</b>
                  <span className="num whitespace-nowrap font-semibold">{rate(r)}</span>
                </span>
                <span className="k">{t(mr.ftpLine, { paid: count(r.paid), base: count(r.base), start: count(r.atStart), joined: count(r.joined) })}</span>
              </div>
            ))}
          </div>
          <div className="er-pad" style={{ borderTop: '1px solid var(--line-2)' }}>
            <span className="k">{t(mr.ftpDirect, { count: f.direct, all: count(f.all) })}</span>
          </div>
        </>
      )}
    </Card>
  )
}

/** Upgrades by email: new paid members by the email they're credited to. */
function Upgrades({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const u = o.upgrades
  const cur = o.currency
  const label = (r: Schemas['RevenueUpgradeRow']): ReactNode => {
    if (r.kind === 'campaign' && r.id) {
      return (
        <Link to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: r.id }} className="font-semibold" style={{ color: 'var(--ink)' }}>
          {r.sentAt ? t(mr.upCampaign, { name: r.name, date: dayMonthDate(r.sentAt) }) : r.name}
        </Link>
      )
    }
    if (r.kind === 'automation') {
      return r.id ? (
        <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: r.id }} className="font-semibold" style={{ color: 'var(--ink)' }}>
          {t(mr.upAutomation, { name: r.name })}
        </Link>
      ) : (
        <span className="font-semibold">{t(mr.upAutomation, { name: r.name })}</span>
      )
    }
    return <span className="k">{t(mr.upNone)}</span>
  }
  return (
    <Card
      title={
        <>
          {t(mr.upTitle, { period: o.period })} <Def label={t(mr.defCreditLabel)} text={t(mr.defCredit, { days: u.window })} />
        </>
      }
      sub={t(mr.upSub, { days: u.window })}
      className="overflow-hidden"
    >
      {u.rows.length === 0 ? (
        <p className="er-pad m-0 k">{t(mr.upEmpty)}</p>
      ) : (
        <>
          <div className="max-md:hidden">
            <table className="tbl er-tbl" aria-label={t(mr.upTitle, { period: o.period })}>
              <thead>
                <tr>
                  <th scope="col">{t(mr.colEmail)}</th>
                  <th scope="col" className="r">
                    {t(mr.colNew)}
                  </th>
                  <th scope="col" className="r">
                    {t(mr.colNewMrr)}
                  </th>
                </tr>
              </thead>
              <tbody>
                {u.rows.map((r, i) => (
                  <tr key={`${r.kind}-${r.id ?? i}-${r.name}`}>
                    <td>{label(r)}</td>
                    <td className="r num">{count(r.members)}</td>
                    <td className="r num">{money(r.mrrCents, cur, 'never')}</td>
                  </tr>
                ))}
                <tr className="total">
                  <td>
                    <b className="font-bold">{t(mr.total)}</b>
                  </td>
                  <td className="r num">
                    <b>{count(u.total.count)}</b>
                  </td>
                  <td className="r num">
                    <b>{money(u.total.cents, cur, 'never')}</b>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div className="hidden max-md:flex max-md:flex-col">
            {u.rows.map((r, i) => (
              <div key={`${r.kind}-${r.id ?? i}-${r.name}`} className="er-lrow">
                <span className="lt">
                  {label(r)}
                  <span className="num whitespace-nowrap font-semibold">{money(r.mrrCents, cur, 'never')}</span>
                </span>
                <span className="k">{t(mr.upMembers, { count: r.members })}</span>
              </div>
            ))}
            <div className="er-lrow">
              <span className="lt">
                <b className="font-semibold">{t(mr.total)}</b>
                <span className="num whitespace-nowrap font-semibold">{money(u.total.cents, cur, 'never')}</span>
              </span>
              <span className="k">{t(mr.upMembers, { count: u.total.count })}</span>
            </div>
          </div>
        </>
      )}
    </Card>
  )
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : typeof v === 'number' ? String(v) : ''
}

/** Recent activity: the latest of each kind of paid-member event. */
function Activity({ o, provider }: { o: Overview; provider: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Card
      title={t(mr.actTitle)}
      sub={t(mr.actSub, { provider })}
      aside={
        <Link to="/w/$workspace/earn/settings" params={{ workspace: slug }} className="er-lnk">
          {t(mr.eventLog)}
        </Link>
      }
    >
      <div className="er-pad pb-1.5 pt-1.5">
        {o.activity.length === 0 ? <p className="m-0 py-3 k">{t(mr.actNone)}</p> : null}
        {o.activity.map((a, i) => {
          const line = activityLine(t, a, o.currency, provider)
          return (
            <div key={i} className="er-ev">
              <span className="ic" style={{ background: line.bg, color: line.fg }} aria-hidden="true">
                <Icon name={line.icon} size={14} />
              </span>
              <div className="tx">
                <span className="font-semibold">{line.title}</span>
                {line.detail ? <span className="k">{line.detail}</span> : null}
              </div>
              <span className="tm max-md:hidden">{dayMonthDate(a.at)}</span>
            </div>
          )
        })}
      </div>
    </Card>
  )
}

type ActivityIcon = 'clock' | 'more' | 'arrowRight' | 'check'

function activityLine(t: TT, a: Schemas['RevenueActivity'], cur: string, provider: string): { title: string; detail?: string; icon: ActivityIcon; bg: string; fg: string } {
  const d = a.data ?? {}
  const c = a.currency || cur
  const tier = t(mr.tierPeriodPlain, { tier: a.tier, period: a.period })
  const amountOf = (n: number | null | undefined) => (n === null || n === undefined ? '' : n < 0 ? `−${money(-n, c, 'always')}` : money(n, c, 'always'))
  const neutral = { bg: 'var(--neutral-bg)', fg: 'var(--neutral-fg)' }
  switch (a.kind) {
    case 'payment_failed': {
      const retry = str(d.retry)
      const values = { tier, reason: str(d.reason), provider, date: retry ? day(retry) : '', first: a.firstName }
      return {
        title: t(mr.actFailed, { name: a.name, ends: ends(a.name), amount: amountOf(a.amountCents) }),
        detail: !retry ? t(mr.actFailedDetailNoRetry, values) : a.accessPaused ? t(mr.actFailedDetailPaused, values) : t(mr.actFailedDetail, values),
        icon: 'clock',
        bg: 'var(--warn-bg)',
        fg: 'var(--warn-fg)',
      }
    }
    case 'refunded':
      return {
        title: t(mr.actRefunded, { provider, name: a.name, amount: amountOf(a.amountCents) }),
        detail: a.canceledBy ? t(mr.actRefundedDetail, { tier, who: a.canceledBy }) : tier,
        icon: 'more',
        ...neutral,
      }
    case 'cancel_scheduled': {
      const date = str(d.date)
      const reason = str(d.reason)
      const offer = a.sawOffer ?? ''
      let detail: string | undefined
      if (date && reason) detail = offer ? t(mr.actCanceledDetailSaw, { date: day(date), reason, offer }) : t(mr.actCanceledDetail, { date: day(date), reason })
      else if (date) detail = offer ? t(mr.actCanceledDetailNoReasonSaw, { date: day(date), offer }) : t(mr.actCanceledDetailNoReason, { date: day(date) })
      return { title: t(mr.actCanceled, { name: a.name, tier: a.tier }), detail, icon: 'more', ...neutral }
    }
    case 'offer_applied': {
      let detail = tier
      if (a.offerCents !== undefined && a.offerCents !== null) {
        const values = { offer: money(a.offerCents, c), period: a.period, price: money(a.priceCents, c) }
        detail = a.offerEndsAt ? t(mr.actOfferDetail, { ...values, date: day(a.offerEndsAt) }) : t(mr.actOfferDetailForever, values)
      }
      return { title: t(mr.actOffer, { name: a.name, offer: str(d.offer) }), detail, icon: 'arrowRight', bg: 'var(--info-bg)', fg: 'var(--info-fg)' }
    }
    case 'gift_redeemed': {
      const values = { tier: a.tier, months: Number(d.months ?? 12), from: str(d.from) }
      return {
        title: t(mr.actGift, { name: a.name }),
        detail: !values.from ? a.tier : a.endsAt ? t(mr.actGiftDetailEnds, { ...values, date: day(a.endsAt) }) : t(mr.actGiftDetail, values),
        icon: 'check',
        bg: 'var(--ok-bg)',
        fg: 'var(--ok-fg)',
      }
    }
  }
  const offer = str(d.offer)
  return {
    title: offer ? t(mr.actStartedOffer, { name: a.name, tier, offer, amount: amountOf(a.amountCents) }) : t(mr.actStarted, { name: a.name, tier, amount: amountOf(a.amountCents) }),
    detail: a.source ? t(mr.actFrom, { source: a.source }) : undefined,
    icon: 'check',
    bg: 'var(--ok-bg)',
    fg: 'var(--ok-fg)',
  }
}

export { Card as RevenueCard, Stat as RevenueStat, Delta as RevenueDelta, CountDelta as RevenueCountDelta }
