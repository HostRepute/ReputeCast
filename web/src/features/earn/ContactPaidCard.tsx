import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { useT } from '@/i18n'
import { dayMonthYear } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { ComplimentaryDialog } from './ComplimentaryDialog'
import { PaidTierChip } from './ContactsPaid'
import { mpc } from './contacts.messages'
import { day, money, priceLine } from './format'
import { CancelDialog, ResendDialog, SourceValue, StatusLine, TimelineRow } from './MemberPanel'
import { me } from './messages'
import { mm } from './members.messages'
import './earn.css'

type Member = Schemas['EarnMember']

/** A contact's paid access, for the profile (null when they never had any). */
export function useContactPaid(contactId: string) {
  const { slug } = useWorkspace()
  const can = useCan('earn:view')
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/earn/contacts/{contactId}', { params: { path: { workspace: slug, contactId } } }, { enabled: can, staleTime: 30_000 })
}

/**
 * The contact profile's Paid subscription card (EarnInAudience): tier,
 * price, status, renewal, lifetime value and source, the links to Stripe
 * and the paid member, and the member actions of 5.9.
 */
export function ContactPaidCard({ contactId }: { contactId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useContactPaid(contactId)
  const [dialog, setDialog] = useState<'give' | 'cancel' | 'resend'>()
  const m = q.data
  if (q.isPending && q.fetchStatus !== 'idle') {
    return (
      <section className="card flex flex-col gap-3 px-4.5 py-4" aria-busy="true" aria-label={t(mpc.cardTitle)}>
        <Skeleton w={160} h={16} />
        <Skeleton w="100%" h={120} />
      </section>
    )
  }
  if (!m) return null
  const since =
    m.kind === 'complimentary'
      ? t(mpc.cardSinceGrant, { date: dayMonthYear(m.since), publisher: m.source.label || '' })
      : m.kind === 'gift'
        ? t(mpc.cardSinceGift, { date: dayMonthYear(m.since) })
        : t(mpc.cardSince, { date: dayMonthYear(m.since) })
  const rows: [string, ReactNode][] = [
    [t(mpc.tier), m.kind === 'stripe' && m.period ? t(mpc.tierPeriod, { tier: m.tierName, period: m.period }) : m.tierName || m.tier],
  ]
  if (m.kind !== 'gift') rows.push([t(mpc.price), m.kind === 'complimentary' ? t(me.complimentaryNoPayment) : priceLine(t, m, { percent: false, refund: false })])
  rows.push([t(mpc.status), <StatusLine key="s" m={m} />])
  if (m.next === 'renews' && m.nextAt) rows.push([t(mpc.renews), day(m.nextAt)])
  else if ((m.next === 'ends' || m.next === 'ended') && m.nextAt) rows.push([t(mpc.ends), day(m.nextAt)])
  else if (m.next === 'trial_ends' && m.nextAt) rows.push([t(mm.trialEndsLabel), day(m.nextAt)])
  else if (m.next === 'retries') rows.push([t(mm.retryLabel), m.nextAt ? t(mm.retryLine, { date: day(m.nextAt) }) : t(me.retriesSoon)])
  else if (m.next === 'no_end') rows.push([t(mpc.ends), t(me.noEnd)])
  rows.push([t(mpc.ltv), money(m.ltvCents, m.currency || 'usd', 'auto')])
  rows.push([t(mpc.source), <SourceValue key="src" m={m} />])
  const stripe = m.subscriptionUrl || m.customerUrl
  return (
    <section className="card flex flex-col" aria-labelledby="ea-paid-h">
      <div className="flex flex-col gap-0.5 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="ea-paid-h">
          {t(mpc.cardTitle)}
        </h2>
        <span className="k">{since}</span>
      </div>
      <div className="flex flex-col gap-3 px-4.5 py-4">
        <dl className="ea-kv">
          {rows.map(([k, v], i) => (
            <div key={i} className="contents">
              <dt>{k}</dt>
              <dd>{v}</dd>
            </div>
          ))}
        </dl>
        <div className="flex flex-wrap gap-2">
          {stripe ? (
            <ButtonAnchor href={stripe} size="sm" iconRight="external">
              {t(mpc.openStripe)}
            </ButtonAnchor>
          ) : null}
          <ButtonLink to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ member: m.id }} size="sm">
            {t(mpc.openMember)}
          </ButtonLink>
        </div>
      </div>
      {m.canManage && !m.test ? (
        <div className="flex flex-wrap gap-2 border-t border-line-2 px-4.5 py-3">
          {m.kind === 'complimentary' ? null : (
            <Button size="sm" onClick={() => setDialog('give')}>
              {t(mpc.giveComp)}
            </Button>
          )}
          {m.kind === 'stripe' && m.status !== 'canceled' ? (
            <Button size="sm" variant="danger" onClick={() => setDialog('cancel')}>
              {t(mpc.cancel)}
            </Button>
          ) : null}
          {m.emails.length ? (
            <Button size="sm" onClick={() => setDialog('resend')}>
              {t(mpc.resend)}
            </Button>
          ) : null}
        </div>
      ) : null}
      {dialog === 'give' ? <ComplimentaryDialog person={{ id: m.contactId, name: m.name, email: m.email, lists: m.lists }} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'cancel' ? <CancelDialog m={m} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'resend' ? <ResendDialog m={m} onClose={() => setDialog(undefined)} /> : null}
    </section>
  )
}

/** The tier chip next to a paid member's name on the profile ("Insider"). */
export function ContactTierChip({ contactId }: { contactId: string }) {
  const m = useContactPaid(contactId).data
  if (!m || m.status === 'canceled') return null
  return <PaidTierChip paid={{ tier: m.tier, status: m.status, memberId: m.id }} />
}

/** Whether the profile's Activity offers the Paid filter: the contact had paid access. */
export function useHasPaidActivity(contactId: string): boolean {
  return Boolean(useContactPaid(contactId).data)
}

/** The Activity card's Paid filter: the paid member's timeline (EarnInAudience). */
export function PaidActivity({ contactId }: { contactId: string }) {
  const t = useT()
  const q = useContactPaid(contactId)
  const m: Member | null | undefined = q.data
  if (!m) return null
  return m.timeline.length ? (
    <ul className="ea-tl border-t border-line-2 px-4.5 py-3">
      {m.timeline.map((it) => (
        <TimelineRow key={it.id} it={it} m={m} />
      ))}
    </ul>
  ) : (
    <div className="border-t border-line-2 px-4.5 py-8 text-center text-ink-3">{t(mpc.noPaidActivity)}</div>
  )
}
