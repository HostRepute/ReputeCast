import type { Schemas } from '@/api/client'
import type { useT } from '@/i18n'
import { andList } from '@/lib/format'
import { money } from '../format'
import { mof } from './messages'

type TT = ReturnType<typeof useT>
type Offer = Schemas['Offer']
type Applies = Schemas['OfferApplies']

/** The discount's part: "20%", "$5". */
export function offOf(t: TT, o: Pick<Offer, 'discountType' | 'percentOff' | 'amountOffCents' | 'currency'>): string {
  return o.discountType === 'percent' ? t(mof.pct, { n: o.percentOff ?? 0 }) : money(o.amountOffCents ?? 0, o.currency || 'usd')
}

/** A price after the offer's discount. */
export function discounted(price: number, o: Pick<Offer, 'discountType' | 'percentOff' | 'amountOffCents'>): number {
  if (o.discountType === 'percent') return Math.round((price * (100 - (o.percentOff ?? 0))) / 100)
  return Math.max(price - (o.amountOffCents ?? 0), 0)
}

/** "20% off the first year: $64 instead of $80" for the first tier and period it applies to. */
export function discountLine(t: TT, o: Offer): string {
  const off = offOf(t, o)
  const a: Applies | undefined = o.applies[0]
  if (!a) return t(mof.discountPlain, { off })
  const cur = a.currency || o.currency || 'usd'
  const price = money(discounted(a.priceCents, o), cur)
  const full = money(a.priceCents, cur)
  if (o.kind === 'referral' && o.duration === 'months') return t(mof.discountFree, { off, tier: a.short, period: a.period, months: o.months ?? 0 })
  if (o.duration === 'once') return a.period === 'year' ? t(mof.discountOnceYear, { off, price, full }) : t(mof.discountOnceMonth, { off, price, full })
  if (o.duration === 'months') return t(mof.discountMonths, { off, months: o.months ?? 0, price, period: a.period })
  return t(mof.discountForever, { off, price, period: a.period })
}

/** "Insider yearly and Founding member yearly" */
export function tiersLine(t: TT, applies: Applies[]): string {
  return andList(applies.map((a) => t(mof.tierPeriod, { tier: a.short, period: a.period })))
}

/** Who it's for: "Insider yearly · new paid members only". */
export function forLine(t: TT, o: Offer): string {
  const tiers = tiersLine(t, o.applies)
  switch (o.kind) {
    case 'retention':
      return o.minPaidMonths > 0 ? t(mof.forRetention, { tiers, months: o.minPaidMonths }) : t(mof.forRetentionAny, { tiers })
    case 'referral':
      return t(mof.forReferral)
  }
  return o.newOnly ? t(mof.forNew, { tiers }) : t(mof.forLine, { tiers })
}

/** How readers get it: "Code FALL26 and the link news.acme.io/upgrade?offer=FALL26". */
export function howLine(t: TT, o: Offer): string {
  switch (o.kind) {
    case 'retention':
      return t(mof.howCancel)
    case 'referral':
      return t(mof.howReferral)
  }
  const link = o.linkHost ?? ''
  if (o.byCode && o.byLink) return t(mof.howBoth, { code: o.code, link })
  if (o.byLink) return t(mof.howLink, { link })
  return t(mof.howCode, { code: o.code })
}
