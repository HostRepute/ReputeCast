import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { Link } from '@tanstack/react-router'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { useCan, useWorkspace } from '@/lib/session'
import { mpi } from './entries.messages'
import './earn.css'

/**
 * The Payments group of Settings › Integrations (IntegrationsCatalog, Earn
 * brief §6): Stripe, connected or not, which opens Earn › Settings ›
 * Payments, and Paddle and Lemon Squeezy, not available yet.
 */

export { mpi }

export const paymentKeys = ['stripe', 'paddle', 'lemonsqueezy'] as const
export type PaymentKey = (typeof paymentKeys)[number]
export const isPayment = (k: string): k is PaymentKey => (paymentKeys as readonly string[]).includes(k)

export const paymentEntryName: Record<PaymentKey, MessageDescriptor> = { stripe: mpi.stripe, paddle: mpi.paddle, lemonsqueezy: mpi.lemonsqueezy }
export const paymentMaintainer: Record<PaymentKey, MessageDescriptor> = { stripe: mpi.byOwnStripe, paddle: mpi.byMerchant, lemonsqueezy: mpi.byMerchant }
export const paymentEntryLine: Record<PaymentKey, MessageDescriptor> = { stripe: mpi.stripeLine, paddle: mpi.merchantLine, lemonsqueezy: mpi.merchantLine }
export const paymentCapabilities: Record<PaymentKey, MessageDescriptor[]> = {
  stripe: [mpi.capSubscriptions, mpi.capOneTime, mpi.capPortal, mpi.capTestMode],
  paddle: [mpi.capSubscriptions, mpi.capTax],
  lemonsqueezy: [mpi.capSubscriptions, mpi.capTax],
}
export const paymentLetters: Record<PaymentKey, string> = { stripe: 'S', paddle: 'P', lemonsqueezy: 'LS' }

type Entry = Schemas['IntegrationEntry']

/** "Not available yet" for the providers that aren't in this release. */
export function paymentUnavailable(e: Entry): boolean {
  return isPayment(e.key) && !e.facts.payment?.available
}

export function UnavailableBadge() {
  const t = useT()
  return <Badge tone="neutral">{t(mpi.notAvailable)}</Badge>
}

/** A payment card's fact line. */
export function PaymentFact({ e }: { e: Entry }) {
  const t = useT()
  const p = e.facts.payment
  if (!p?.available) return <>{t(mpi.factNotInRelease)}</>
  if (e.status === 'not_connected' || !p.account) return <>{t(mpi.factRestricted)}</>
  return <>{t(mpi.factConnected, { account: p.account, mode: p.mode ?? '' })}</>
}

/** A payment card's action: connect Stripe, or open Earn › Settings › Payments. */
export function PaymentAction({ e, canManage }: { e: Entry; canManage: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (!e.facts.payment?.available) return null
  const params = { workspace: slug }
  if (e.status === 'not_connected') {
    return canManage ? (
      <ButtonLink size="sm" to="/w/$workspace/earn/connect" params={params}>
        {t(mpi.connect)}
      </ButtonLink>
    ) : null
  }
  return (
    <ButtonLink size="sm" to="/w/$workspace/earn/settings" params={params} search={{ tab: 'payments' }}>
      {canManage ? t(mpi.manage) : t(mpi.view)}
    </ButtonLink>
  )
}

/** Import contacts › Source, under the source tiles (EarnImportStates): paid members import in Earn. */
export function PaidMembersImportHint() {
  const { slug } = useWorkspace()
  const can = useCan('earn:view')
  if (!can) return null
  return (
    <div className="ea-hintrow">
      <span>
        <T
          msg={mpi.importHint}
          values={{
            a: (chunks) => (
              <Link to="/w/$workspace/earn/members" params={{ workspace: slug }} search={{ import: 'new' }}>
                {chunks}
              </Link>
            ),
          }}
        />
      </span>
    </div>
  )
}
