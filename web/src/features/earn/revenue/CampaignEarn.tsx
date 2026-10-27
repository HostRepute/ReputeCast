import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { T, useT } from '@/i18n'
import { andList, count, dayMonthDate, orList, rate, shortDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { day, money } from '../format'
import { ShareDialog } from '../sponsors/dialogs'
import { Initials } from '../sponsors/parts'
import '../sponsors/sponsors.css'
import { mrp } from './report.messages'
import { Def, RevenueCard } from './RevenueOverview'
import './revenue.css'

type Revenue = Schemas['CampaignRevenue']
type TT = ReturnType<typeof useT>

const WS = '/api/v1/workspaces/{workspace}' as const
const mono = (c: ReactNode) => <span className="mono">{c}</span>

/**
 * A sent campaign's Revenue and Sponsor sections (EarnInReport): upgrades,
 * what they collected and the conversions credited to the email, and the
 * issue's sponsor with its report. Shown in the report, never on a shared
 * read-only link.
 */
export function CampaignEarn({ campaignId, conversionsInReport }: { campaignId: string; conversionsInReport?: boolean }) {
  const { slug } = useWorkspace()
  const canView = useCan('earn:view')
  const path = { params: { path: { workspace: slug, id: campaignId } } }
  const rev = $api.useQuery('get', `${WS}/campaigns/{id}/revenue`, path)
  const sp = $api.useQuery('get', `${WS}/campaigns/{id}/sponsor`, path, { enabled: canView })
  const booking = sp.data?.booking
  // The report's "Growth & revenue from this issue" shows an email
  // without upgrade links' conversions (review AN-12); its full Revenue
  // section stays for upgrades.
  const shown = rev.data?.active && !(conversionsInReport && !rev.data.locked && !rev.data.hasUpgradeLinks && rev.data.upgrades === 0)
  if (!shown && !booking && !rev.error) return null
  return (
    <>
      {rev.error ? <RevenueError error={rev.error} onRetry={() => void rev.refetch()} /> : shown && rev.data ? <RevenueSection r={rev.data} /> : null}
      {booking ? <SponsorSection b={booking} /> : null}
    </>
  )
}

function RevenueError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  return (
    <RevenueCard title={t(mrp.revTitle)}>
      <div className="er-pad">
        <Note tone="bad" label={t(mrp.revError)}>
          {errorMessage(error)}{' '}
          <button type="button" className="underline" onClick={onRetry}>
            {t(mrp.retry)}
          </button>
        </Note>
      </div>
    </RevenueCard>
  )
}

function RevenueSection({ r }: { r: Revenue }) {
  const t = useT()
  const sub = r.sentAt ? t(r.pending ? mrp.subPending : mrp.subSent, { name: r.name, date: shortDate(r.sentAt) }) : r.name
  if (r.locked) return <Locked r={r} />
  const full = r.hasUpgradeLinks || r.upgrades > 0
  return (
    <RevenueCard
      title={
        <span className="inline-flex items-center gap-0.5">
          {t(mrp.revTitle)}
          {full ? <Def label={t(mrp.howLabel)} text={t(mrp.how, { days: r.window })} /> : null}
        </span>
      }
      sub={sub}
    >
      {full ? <Upgrades r={r} /> : <ConversionsOnly r={r} />}
    </RevenueCard>
  )
}

/** Without earn:view: who can show it. */
function Locked({ r }: { r: Revenue }) {
  const t = useT()
  const { me: wme } = useWorkspace()
  const user = useMe().data?.user
  const role = roleName({ key: wme.workspace.roleKey, name: wme.workspace.roleName, builtin: true })
  return (
    <section className="card max-w-[620px]">
      <div className="er-sec">
        <div className="er-head">
          <h2 className="h2">{t(mrp.revTitle)}</h2>
          <span className="k">{r.name}</span>
        </div>
      </div>
      <div className="er-pad">
        <div className="note n" role="note">
          <span style={{ color: 'var(--ink-2)', display: 'inline-flex', flex: 'none', marginTop: 1 }}>
            <Icon name="info" size={16} />
          </span>
          <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
            <span className="font-bold">{t(mrp.lockedTitle, { name: user?.name || user?.email || '', role })}</span>
            <div>
              <T msg={r.helpers.length ? mrp.locked : mrp.lockedNoHelpers} values={{ mono, helpers: orList(r.helpers) }} />
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

function upgradeLine(t: TT, l: Schemas['CampaignUpgradeLine']): string {
  const values = { count: l.count, tier: l.tier, period: l.period }
  return l.offer ? t(mrp.lineOffer, { ...values, offer: l.offer }) : t(mrp.line, values)
}

/** The full section: the email linked to the upgrade page. */
function Upgrades({ r }: { r: Revenue }) {
  const t = useT()
  const cur = r.currency
  const max = Math.max(r.upgradeClicks, r.checkouts, r.upgrades, 1)
  const bar = (n: number) => `${Math.max(n > 0 ? 1.5 : 0, Math.min(100, (n / max) * 100)).toFixed(1)}%`
  const collected = r.collected.length
    ? t(mrp.collectedSub, { parts: andList(r.collected.map((c) => t(mrp.paidTimes, { count: c.count, amount: money(c.amountCents, cur, 'always') }))) })
    : t(mrp.collectedNone)
  const trialsEnd = r.trialsEnd ? day(r.trialsEnd) : ''
  const c = r.conversions
  const stat = (label: ReactNode, value: string, s: ReactNode) => (
    <div className="card er-stat">
      <span className="k er-k">{label}</span>
      <span className="v">{value}</span>
      <span className="s">{s}</span>
    </div>
  )
  const funnel = (label: string, n: number, extra?: string) => (
    <div className="flex flex-col gap-1">
      <div className="flex justify-between gap-2.5 text-[13px]">
        <span>{label}</span>
        <span className="font-semibold tabular-nums">
          {count(n)}
          {extra ? <span className="k font-normal"> {extra}</span> : null}
        </span>
      </div>
      <div className="bar" aria-hidden="true">
        <i style={{ width: bar(n), background: 'var(--accent)' }} />
      </div>
    </div>
  )
  return (
    <div className="er-pad flex flex-col gap-3.5">
      {r.pending && r.trialsEnd ? (
        <Note tone="info">{t(mrp.pending, { trials: r.trials, date: dayMonthDate(r.trialsEnd), paid: r.paidAtOnce })}</Note>
      ) : null}
      <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}>
        {stat(t(mrp.tUpgrades), count(r.upgrades), t(mrp.upgradesSub, { paid: count(r.paidAtOnce), trials: r.trials }))}
        {stat(t(mrp.tCollected), money(r.collectedCents, cur, 'never'), collected)}
        {stat(
          <>
            {t(mrp.tNewMrr)}
            <Def label={t(mrp.mrrLabel)} text={t(mrp.mrrDef, { currency: cur.toUpperCase() })} />
          </>,
          money(r.newMrrCents, cur, 'never'),
          r.trials > 0 ? t(mrp.newMrrSub, { amount: money(r.potentialMrrCents, cur, 'never') }) : t(mrp.newMrrNoTrials),
        )}
        {stat(t(mrp.tClicks), count(r.upgradeClicks), t(mrp.clicksSub, { count: r.checkouts }))}
        {stat(t(mrp.tUpgraded), rate(r.upgradedRate), t(mrp.upgradedSub, { upgrades: count(r.upgrades), delivered: count(r.delivered) }))}
      </div>
      <div className="grid items-start gap-6" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(360px, 100%), 1fr))' }}>
        <div
          className="flex flex-col gap-2"
          role="img"
          aria-label={t(mrp.funnelLabel, { clicks: count(r.upgradeClicks), checkouts: count(r.checkouts), upgrades: count(r.upgrades) })}
        >
          <span className="text-[13px] font-semibold">{t(mrp.funnelTitle)}</span>
          <div className="flex flex-col gap-2.5">
            {funnel(t(mrp.fClicks), r.upgradeClicks)}
            {funnel(t(mrp.fCheckouts), r.checkouts)}
            {funnel(t(mrp.fUpgrades), r.upgrades, t(mrp.fSplit, { paid: count(r.paidAtOnce), trials: r.trials }))}
          </div>
        </div>
        <div className="flex flex-col gap-1.5">
          <span className="text-[13px] font-semibold">{t(mrp.upTitle)}</span>
          <dl className="kv">
            <dt>{t(mrp.kvPaid)}</dt>
            <dd>{r.paidLines.length ? r.paidLines.map((l) => upgradeLine(t, l)).join(' · ') : <span className="k">{t(mrp.none)}</span>}</dd>
            <dt>{t(mrp.kvTrials)}</dt>
            <dd>
              {r.trialLines.length ? (
                trialsEnd && r.pending ? (
                  t(mrp.trialsEnd, { lines: r.trialLines.map((l) => upgradeLine(t, l)).join(' · '), date: trialsEnd })
                ) : (
                  r.trialLines.map((l) => upgradeLine(t, l)).join(' · ')
                )
              ) : (
                <span className="k">{t(mrp.none)}</span>
              )}
            </dd>
            <dt>{t(mrp.kvConversions)}</dt>
            <dd>
              {c.count ? (
                t(mrp.conversionsLine, { count: count(c.count), value: money(c.valueCents, c.currency, 'never'), token: count(c.byToken), email: count(c.byEmail) })
              ) : (
                <span className="k">{t(mrp.noneCredited)}</span>
              )}
            </dd>
          </dl>
        </div>
      </div>
      <span className="k">{t(mrp.foot, { days: r.window })}</span>
    </div>
  )
}

/** No upgrade links: the conversions credited to the email. */
function ConversionsOnly({ r }: { r: Revenue }) {
  const t = useT()
  const c = r.conversions
  const big = (value: string, label: string) => (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="text-[22px] font-bold leading-tight tracking-[-0.01em] tabular-nums">{value}</span>
      <span className="k">{label}</span>
    </div>
  )
  const typical = money(c.typicalCents ?? 0, c.currency)
  const kind = c.name ? (c.sender ? t(mrp.convKind, { name: c.name, value: typical, sender: c.sender }) : t(mrp.convKindNoSender, { name: c.name, value: typical })) : t(mrp.convKinds)
  return (
    <div className="er-pad flex flex-col gap-3">
      <div className="flex flex-wrap gap-x-7 gap-y-3">
        {big(count(r.upgrades), r.hasUpgradeLinks ? t(mrp.nUpgrades) : t(mrp.nUpgradesNoLinks))}
        {big(count(c.count), t(mrp.nConversions))}
        {big(money(c.valueCents, c.currency, 'never'), t(mrp.nValue))}
      </div>
      <dl className="kv">
        <dt>{t(mrp.kvConv)}</dt>
        <dd>{c.count ? kind : <span className="k">{t(mrp.convNone)}</span>}</dd>
        {c.count ? (
          <>
            <dt>{t(mrp.kvMatched)}</dt>
            <dd>
              {c.domains.length
                ? t(mrp.matchedTokenOn, { token: count(c.byToken), email: count(c.byEmail), domains: andList(c.domains) })
                : t(mrp.matched, { token: count(c.byToken), email: count(c.byEmail) })}
            </dd>
          </>
        ) : null}
        <dt>{t(mrp.kvWindow)}</dt>
        <dd>{t(mrp.window, { days: r.window })}</dd>
      </dl>
    </div>
  )
}

/** The issue's sponsor: price, payment, unique clickers and clicks, and its shared report. */
function SponsorSection({ b }: { b: Schemas['SponsorBookingDetail'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [sharing, setSharing] = useState(false)
  const rep = b.report
  const p = b.payment
  const badge =
    p.status === 'paid' ? (
      <Badge tone="ok">{p.paidOn ? t(mrp.paidOn, { date: dayMonthDate(p.paidOn) }) : t(mrp.paid)}</Badge>
    ) : p.status === 'overdue' ? (
      <Badge tone="warn">{t(mrp.overdue, { date: p.due ? dayMonthDate(p.due) : '' })}</Badge>
    ) : (
      <Badge tone="neutral">{p.due ? t(mrp.unpaidDue, { date: dayMonthDate(p.due) }) : t(mrp.unpaid)}</Badge>
    )
  const share = b.shares[0]
  const parts = rep ? rep.links.filter((l) => l.clicks > 0).map((l) => t(mrp.part, { kind: l.kind, count: count(l.clicks) })) : []
  return (
    <section className="card min-w-0 max-w-[720px]" aria-labelledby="rep-sponsor-h">
      <div className="er-sec">
        <div className="er-head">
          <h2 className="h2" id="rep-sponsor-h">
            {t(mrp.spTitle)}
          </h2>
          <span className="k">{t(mrp.spSub)}</span>
        </div>
      </div>
      <div className="er-pad flex flex-col gap-3">
        <div className="flex items-center gap-3">
          <Initials initials={b.sponsor?.initials ?? ''} logo={b.sponsor?.logoUrl || undefined} size="xl" className="w-9! h-9! rounded-[9px]!" />
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <b className="font-semibold">{b.sponsor?.name}</b>
            <span className="k">{t(mrp.spSlot, { slot: b.slotName, placement: b.placement })}</span>
          </div>
          <span className="font-bold tabular-nums">{money(b.priceCents, b.currency)}</span>
          {badge}
        </div>
        {rep ? (
          <dl className="kv">
            <dt>{t(mrp.kvClickers)}</dt>
            <dd>{t(mrp.clickers, { count: count(rep.uniqueClickers), rate: rate(rep.clickRate), delivered: count(rep.delivered) })}</dd>
            <dt>{t(mrp.kvClicks)}</dt>
            <dd>{parts.length ? t(mrp.clicks, { count: count(rep.clicks), parts: parts.join(', ') }) : count(rep.clicks)}</dd>
            <dt>{t(mrp.kvReport)}</dt>
            <dd>
              {share
                ? share.expiresAt
                  ? t(mrp.sharedExpires, { who: share.createdBy, date: dayMonthDate(share.createdAt), expires: day(share.expiresAt) })
                  : t(mrp.shared, { who: share.createdBy, date: dayMonthDate(share.createdAt) })
                : <span className="k">{t(mrp.notShared)}</span>}
            </dd>
          </dl>
        ) : (
          <span className="k">{t(mrp.noReport)}</span>
        )}
        <div className="flex flex-wrap gap-2">
          {b.canShare && rep ? (
            <Button size="sm" variant="primary" onClick={() => setSharing(true)}>
              {t(mrp.share)}
            </Button>
          ) : null}
          <ButtonLink size="sm" to="/w/$workspace/earn/sponsors" params={{ workspace: slug }} search={{ booking: b.id }}>
            {t(mrp.openBooking)}
          </ButtonLink>
        </div>
      </div>
      {sharing ? <ShareDialog bookingId={b.id} issue={b.issue} sponsor={b.sponsor?.name ?? ''} onClose={() => setSharing(false)} /> : null}
    </section>
  )
}
