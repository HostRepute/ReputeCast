import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { statusLabel, statusTone } from './format'
import { mpc } from './contacts.messages'
import './earn.css'

/**
 * Paid status in Contacts (EarnInAudience): the Paid status filter, the
 * tier chip next to a paid member's name and the Paid column. They show to
 * members with earn:view once the workspace has a tier (the audience
 * summary's paidTiers).
 */

type Tier = Schemas['AudiencePaidTier']
type Paid = NonNullable<Schemas['ContactListItem']['paid']>

/** The select's value for a filter: "with_access", "tier:<id>", "trialing"… */
export function paidValue(paid?: string, tier?: string): string {
  if (!paid) return ''
  return paid === 'with_access' && tier ? `tier:${tier}` : paid
}

/** The filter of a select value. */
export function paidFilter(v: string): { paid?: string; paidTier?: string } {
  if (!v) return { paid: undefined, paidTier: undefined }
  if (v.startsWith('tier:')) return { paid: 'with_access', paidTier: v.slice(5) }
  return { paid: v, paidTier: undefined }
}

/** Paid status: any paid tier with access, a tier and above, a status, or free subscribers only. */
export function PaidStatusSelect({ tiers, paid, tier, onChange, className }: { tiers: Tier[]; paid?: string; tier?: string; onChange: (f: { paid?: string; paidTier?: string }) => void; className?: string }) {
  const t = useT()
  if (!tiers.length) return null
  return (
    <select className={className} aria-label={t(mpc.paidStatus)} value={paidValue(paid, tier)} onChange={(e) => onChange(paidFilter(e.target.value))}>
      <option value="">{t(mpc.anyPaidStatus)}</option>
      <option value="with_access">{t(mpc.withAccess)}</option>
      {tiers.map((x) => (
        <option key={x.id} value={`tier:${x.id}`}>
          {x.top ? x.short : t(mpc.andAbove, { tier: x.short })}
        </option>
      ))}
      <option value="trialing">{t(mpc.trialing)}</option>
      <option value="past_due">{t(mpc.pastDue)}</option>
      <option value="complimentary">{t(mpc.complimentary)}</option>
      <option value="gift">{t(mpc.gift)}</option>
      <option value="ended">{t(mpc.ended)}</option>
      <option value="free">{t(mpc.free)}</option>
    </select>
  )
}

/** The coins mark of a paid tier. */
export function CoinIcon({ size = 12 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M10 5.9H7.1a1.3 1.3 0 0 0 0 2.6h1.8a1.3 1.3 0 0 1 0 2.6H6M8 4.4v1.5M8 11.1v1.5" />
    </svg>
  )
}

/** The tier chip next to a paid member's name ("Insider"). */
export function PaidTierChip({ paid }: { paid?: Paid | null }) {
  const t = useT()
  if (!paid || paid.status === 'ended' || !paid.tier) return null
  return (
    <span className="ea-tierchip" title={t(mpc.tierTitle)}>
      <CoinIcon />
      {paid.tier}
    </span>
  )
}

/** The Paid column: the paid status, or a dash. */
export function PaidStatusCell({ paid }: { paid?: Paid | null }) {
  const t = useT()
  if (!paid) return <span className="k">–</span>
  if (paid.status === 'ended') return <Badge tone="neutral">{t(mpc.ended)}</Badge>
  return <Badge tone={statusTone(paid.status)}>{statusLabel(t, paid.status)}</Badge>
}
