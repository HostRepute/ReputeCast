import type { Schemas } from '@/api/client'
import type { useT } from '@/i18n'
import { money as dollars } from '@/lib/format'
import { m as um } from './usage.messages'

type Rate = Schemas['AgencyRate']

/** "$1,028.63" (lib/format: the member's region format writes it its own way). */
export const usd = (v: number) => dollars(v)

/** "$300", or "$12.50" with cents. */
export const money = (v: number) => (Number.isInteger(v) ? dollars(v, 0) : usd(v))

/** A rate in words: "Cost + 15%, plus $300 a month", "$120 a month, flat". */
export function rateText(t: ReturnType<typeof useT>, rate: Rate | undefined | null, short = false) {
  if (!rate) return ''
  if (rate.flat != null) return t(um.rateFlat, { amount: money(rate.flat) })
  const terms: string[] = []
  if (rate.costPlusPct != null) terms.push(t(um.termCostPlus, { pct: rate.costPlusPct }))
  if (rate.per1000Emails != null) terms.push(t(um.termPer1000, { amount: usd(rate.per1000Emails) }))
  if (rate.perSegment != null) terms.push(t(um.termPerSegment, { amount: usd(rate.perSegment) }))
  const list = terms.join(', ')
  if (rate.monthlyFee != null) {
    if (!terms.length) return t(um.rateFeeOnly, { amount: money(rate.monthlyFee) })
    return short ? t(um.rateWithFeeShort, { terms: list, amount: money(rate.monthlyFee) }) : t(um.rateWithFee, { terms: list, amount: money(rate.monthlyFee) })
  }
  return list
}
