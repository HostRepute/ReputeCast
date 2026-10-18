import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { useT } from '@/i18n'
import { shortDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { EARN, EarnPage, useEarnStatus } from '../EarnPage'
import { money } from '../format'
import { useEarnChecklist } from '../Overview'
import { msp } from '../sponsors/messages'
import { OpenBadge, WS } from '../sponsors/parts'
import { mr } from './messages'
import { Conversions, RevenueCard, RevenueOverview, RevenueStat } from './RevenueOverview'
import './revenue.css'

const dismissKey = (slug: string) => `rc.earn.invite.${slug}`

function dismissed(slug: string): boolean {
  try {
    return localStorage.getItem(dismissKey(slug)) === '1'
  } catch {
    return false
  }
}

/**
 * Earn › Overview's first choice: without a payment provider, a workspace
 * that records conversions or sells sponsorships sees them (EarnStates,
 * "Conversions only"); otherwise the launch checklist, then the revenue
 * overview once Earn is live.
 */
export function EarnHome({ checklist }: { checklist: ReactNode }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canView = useCan('earn:view')
  const status = useEarnStatus()
  const noProvider = Boolean(status.data && !status.data.mode && !status.data.setup)
  const live = status.data?.mode === 'live'
  const revenue = $api.useQuery('get', `${EARN}/revenue`, { params: { path: { workspace: slug }, query: { period: '30d' } } }, { enabled: canView && (noProvider || live) })
  const sponsors = $api.useQuery('get', `${WS}/sponsors/overview`, { params: { path: { workspace: slug }, query: { range: 'around' } } }, { enabled: canView && noProvider })
  const launch = useEarnChecklist()
  // Earn is live once the live key has paying members, whatever the launch
  // checklist still lists (the sample's gift-ending email is a draft).
  // While the numbers load, or if they fail, the overview says so itself.
  const members = revenue.data ? revenue.data.tiles.members + revenue.data.tiles.trialing : 0
  if (live && (revenue.isPending || revenue.isError || members > 0 || launch.data?.launched)) {
    return (
      <EarnPage tab="overview" actions={<UpgradePageButton />}>
        <RevenueOverview />
      </EarnPage>
    )
  }
  if (noProvider && (revenue.isPending || sponsors.isPending)) {
    return (
      <EarnPage tab="overview">
        <div className="card er-pad" aria-busy="true" aria-label={t(mr.loading)}>
          <Skeleton w="40%" h={18} />
          <Skeleton w="100%" h={160} className="mt-4" />
        </div>
      </EarnPage>
    )
  }
  const conversions = revenue.data && revenue.data.conversions.total > 0 ? revenue.data : undefined
  const slots = sponsors.data && sponsors.data.slots.length > 0 ? sponsors.data : undefined
  if (!noProvider || (!conversions && !slots)) return <>{checklist}</>
  return (
    <EarnPage tab="overview">
      <ConversionsOnly revenue={conversions} sponsors={slots} />
    </EarnPage>
  )
}

/** The header's "Open upgrade page", the public page in a new tab. */
function UpgradePageButton() {
  const t = useT()
  const { slug } = useWorkspace()
  const pages = $api.useQuery('get', `${EARN}/settings/pages`, { params: { path: { workspace: slug } } })
  if (!pages.data?.upgradeUrl) return null
  return (
    <ButtonAnchor href={pages.data.upgradeUrl} iconRight="external" aria-label={t(mr.openUpgradeLabel)}>
      {t(mr.openUpgrade)}
    </ButtonAnchor>
  )
}

function ConversionsOnly({ revenue, sponsors }: { revenue?: Schemas['RevenueOverview']; sponsors?: Schemas['SponsorOverview'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canConnect = useCan('payments:manage')
  const [hide, setHide] = useState(() => dismissed(slug))
  const notNow = () => {
    try {
      localStorage.setItem(dismissKey(slug), '1')
    } catch {
      // private mode: it comes back next time
    }
    setHide(true)
  }
  return (
    <div className="flex flex-col gap-5">
      {hide ? null : (
        <section className="card er-invite">
          <span className="ico" aria-hidden="true">
            <Icon name="earn" size={20} />
          </span>
          <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-0.5">
            <h2 className="h2">{t(mr.inviteTitle)}</h2>
            <span className="k" style={{ color: 'var(--ink-nav)' }}>
              {t(mr.inviteText)}
            </span>
          </div>
          <div className="flex flex-wrap gap-2">
            {canConnect ? (
              <ButtonLink variant="primary" to="/w/$workspace/earn/connect" params={{ workspace: slug }}>
                {t(mr.connect)}
              </ButtonLink>
            ) : null}
            <Button onClick={notNow}>{t(mr.notNow)}</Button>
          </div>
        </section>
      )}
      <div className="er-grid">
        {revenue ? <Conversions c={revenue.conversions} period={revenue.period} latest /> : null}
        {sponsors ? <SponsorsCard s={sponsors} /> : null}
      </div>
    </div>
  )
}

/** Sponsors without Stripe: income, what's unpaid, and the next 3 issues. */
function SponsorsCard({ s }: { s: Schemas['SponsorOverview'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const cur = s.currency
  const unpaid = [...s.past].reverse().find((r) => r.booking && r.booking.payment.status !== 'paid')
  const next = s.upcoming.slice(0, 3)
  return (
    <RevenueCard
      title={t(mr.sponsorsTitle)}
      sub={t(mr.sponsorsSub)}
      aside={
        <Link to="/w/$workspace/earn/sponsors" params={{ workspace: slug }} className="er-lnk">
          {t(mr.sponsorsLink)}
        </Link>
      }
    >
      <div className="er-pad flex flex-wrap gap-x-7 gap-y-3" style={{ paddingBottom: 12 }}>
        <RevenueStat value={money(s.tiles.income30dCents, cur, 'never')} label={t(mr.sponsorIncome, { count: s.tiles.incomeIssues })} />
        <RevenueStat
          value={money(s.tiles.unpaid30dCents, cur, 'never')}
          label={unpaid?.booking?.sponsor ? t(mr.sponsorUnpaid, { who: `${unpaid.booking.sponsor.name}, #${unpaid.number}` }) : t(mr.sponsorUnpaidNone)}
        />
      </div>
      {next.map((r) => {
        const b = r.booking
        const slotPrice = s.slots.find((x) => x.id === r.slotId)?.priceCents ?? 0
        let badge: ReactNode = <OpenBadge />
        let line = t(mr.openLine, { slot: r.slotName, price: money(slotPrice, cur) })
        if (b?.status === 'hold') {
          badge = <Badge tone="info">{t(mr.holdUntil, { date: b.holdUntil ? shortDate(b.holdUntil) : '' })}</Badge>
          line = t(mr.sponsorLineShort, { sponsor: b.sponsor?.name ?? '', price: money(b.priceCents, cur) })
        } else if (b) {
          badge = b.creativeStatus === 'ready' ? <Badge tone="ok">{t(mr.creativeReady)}</Badge> : <Badge tone="warn">{t(mr.waitingCreative)}</Badge>
          line =
            b.creativeStatus === 'waiting' && b.creativeDue
              ? t(mr.sponsorLine, { sponsor: b.sponsor?.name ?? '', price: money(b.priceCents, cur), date: shortDate(b.creativeDue) })
              : t(mr.sponsorLineShort, { sponsor: b.sponsor?.name ?? '', price: money(b.priceCents, cur) })
        }
        return (
          <div key={r.number} className="er-lrow">
            <span className="lt">
              <b className="font-semibold">{t(msp.issueOn, { issue: r.number, date: shortDate(r.issueOn) })}</b>
              {badge}
            </span>
            <span className="k">{line}</span>
          </div>
        )
      })}
    </RevenueCard>
  )
}
