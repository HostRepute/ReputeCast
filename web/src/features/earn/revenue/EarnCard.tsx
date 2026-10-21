import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonLink } from '@/components/ui/Button'
import { defineMessages, useT } from '@/i18n'
import { count, pct, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN } from '../EarnPage'
import { money } from '../format'
import { RevenueCountDelta, RevenueDelta } from './RevenueOverview'
import './revenue.css'

const mc = defineMessages('earn.overviewCard', {
  title: 'Earn · last 30 days',
  label: 'Earn, last 30 days',
  line: '{provider} · {account} · {mode, select, test {Test} other {Live}}',
  lineNoAccount: '{provider} · {mode, select, test {Test} other {Live}}',
  mrr: 'MRR',
  members: 'Paid members',
  churn: 'Churn',
  trials: 'Trials',
  nextSponsor: 'Next sponsor',
  nextIssue: '#{issue} · {sponsor}',
  creativeDue: 'Creative due {date}',
  creativeReady: 'Creative ready',
  open: 'Open Earn',
  silentTitle: '{provider} webhook silent',
  silentText:
    'No event from {provider} for {hours, plural, one {# hour} other {# hours}}, while {active, plural, one {# subscription is} other {# subscriptions are}} active. Statuses are at most an hour behind.',
  checkWebhook: 'Check the webhook',
  inviteLabel: 'Earn',
  inviteTitle: 'Earn from your newsletter',
  inviteText: 'Paid subscriptions, gifts and sponsorships. Readers pay you through your own Stripe account; ReputeCast takes no fee.',
  setUp: 'Set up Earn',
  notNow: 'Not now',
})

const dismissKey = (slug: string) => `rc.overview.earn.${slug}`

function dismissed(slug: string): boolean {
  try {
    return localStorage.getItem(dismissKey(slug)) === '1'
  } catch {
    return false
  }
}

/**
 * The workspace Overview's Earn card (EarnStates, "changes Main"): MRR,
 * paid members, churn and trials over 30 days with the next sponsored
 * issue, a warning when the provider's webhook goes quiet, and before a
 * payment provider an invitation to set Earn up. People who can see Earn
 * get it while the section is in the sidebar.
 */
export function EarnOverviewCard() {
  const { slug, me } = useWorkspace()
  const can = me.permissions.includes('earn:view') && !(me.hiddenSections ?? []).includes('earn')
  const q = $api.useQuery('get', `${EARN}/card`, { params: { path: { workspace: slug } } }, { enabled: can })
  const c = q.data
  if (!can || !c || c.state === 'none') return null
  if (c.state === 'invite') return c.canSetUp ? <Invite /> : null
  return <Live c={c} />
}

function Live({ c }: { c: Schemas['EarnCard'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const provider = c.providerName || 'Stripe'
  const metric = (label: string, value: string, delta?: ReactNode) => (
    <div className="flex flex-col gap-0.5">
      <span className="k">{label}</span>
      <span className="flex items-baseline gap-1.5 whitespace-nowrap">
        <span className="text-[18px] font-bold tabular-nums">{value}</span>
        {delta ? <span className="text-[12px]">{delta}</span> : null}
      </span>
    </div>
  )
  const next = c.nextSponsor
  return (
    <section className="card flex flex-wrap items-center" style={{ padding: '18px 22px', gap: '14px 30px' }} aria-label={t(mc.label)}>
      <div className="flex min-w-0 flex-col gap-0.5" style={{ flex: '1 1 220px' }}>
        <h2 className="h2">{t(mc.title)}</h2>
        <span className="k">{c.account ? t(mc.line, { provider, account: c.account, mode: c.mode ?? 'live' }) : t(mc.lineNoAccount, { provider, mode: c.mode ?? 'live' })}</span>
      </div>
      {metric(t(mc.mrr), money(c.mrrCents, c.currency, 'never'), c.mrrDeltaCents ? <RevenueDelta cents={c.mrrDeltaCents} currency={c.currency} /> : null)}
      {metric(t(mc.members), count(c.members), c.membersDelta ? <RevenueCountDelta n={c.membersDelta} /> : null)}
      {metric(t(mc.churn), c.churnRate === undefined || c.churnRate === null ? '—' : pct(c.churnRate, 1))}
      {metric(t(mc.trials), count(c.trialing))}
      {next ? (
        <div className="flex min-w-0 flex-col gap-0.5" style={{ flex: '1 1 200px' }}>
          <span className="k">{t(mc.nextSponsor)}</span>
          <span className="text-[13px] font-semibold">{t(mc.nextIssue, { issue: next.issue, sponsor: next.sponsor })}</span>
          <span className="k">{next.ready ? t(mc.creativeReady) : next.creativeDue ? t(mc.creativeDue, { date: shortDate(next.creativeDue) }) : null}</span>
        </div>
      ) : null}
      <Link to="/w/$workspace/earn" params={{ workspace: slug }} className="text-[13px] font-semibold">
        {t(mc.open)}
      </Link>
      {c.silentHours ? (
        <div className="basis-full">
          <div className="note w" role="status">
            <Icon name="alertCircle" size={16} />
            <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
              <span className="font-bold">{t(mc.silentTitle, { provider })}</span>
              <div>
                {t(mc.silentText, { provider, hours: c.silentHours, active: c.active })}{' '}
                <Link to="/w/$workspace/earn/settings" params={{ workspace: slug }}>
                  {t(mc.checkWebhook)}
                </Link>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}

function Invite() {
  const t = useT()
  const { slug } = useWorkspace()
  const [hide, setHide] = useState(() => dismissed(slug))
  if (hide) return null
  const notNow = () => {
    try {
      localStorage.setItem(dismissKey(slug), '1')
    } catch {
      // private mode: it comes back next time
    }
    setHide(true)
  }
  return (
    <section className="card er-invite plain" aria-label={t(mc.inviteLabel)}>
      <span className="ico" aria-hidden="true">
        <Icon name="earn" size={20} />
      </span>
      <div className="flex min-w-0 flex-[1_1_360px] flex-col gap-0.5">
        <h2 className="h2">{t(mc.inviteTitle)}</h2>
        <span className="k" style={{ color: 'var(--ink-nav)' }}>
          {t(mc.inviteText)}
        </span>
      </div>
      <div className="flex flex-wrap gap-2">
        <ButtonLink variant="primary" to="/w/$workspace/earn" params={{ workspace: slug }}>
          {t(mc.setUp)}
        </ButtonLink>
        <Button onClick={notNow}>{t(mc.notNow)}</Button>
      </div>
    </section>
  )
}
