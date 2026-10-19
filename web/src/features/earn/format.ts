import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import type { useT } from '@/i18n'
import { count, dayMonthDate, dayMonthYear, money as dollars } from '@/lib/format'
import { me } from './messages'

type T = ReturnType<typeof useT>
export type MemberRow = Schemas['EarnMemberRow']

// Currencies without minor units (Stripe's zero-decimal currencies).
const zeroDecimal = new Set(['bif', 'clp', 'djf', 'gnf', 'jpy', 'kmf', 'krw', 'mga', 'pyg', 'rwf', 'ugx', 'vnd', 'vuv', 'xaf', 'xof', 'xpf'])

/**
 * Money in a currency's minor units, as the design system words it: the
 * symbol before the amount and thousands separators. "auto" drops the cents
 * of a whole amount ($8, $6.67), "always" keeps them ($64.00, single
 * payments), "never" rounds (totals and MRR: $8,960). Negative amounts carry
 * a real minus sign: −$80.00.
 */
export function money(minor: number | null | undefined, currency = 'usd', cents: 'auto' | 'always' | 'never' = 'auto'): string {
  if (minor === null || minor === undefined || Number.isNaN(minor)) return '—'
  const cur = (currency || 'usd').toLowerCase()
  const neg = minor < 0
  const abs = Math.abs(minor)
  const major = zeroDecimal.has(cur) ? abs : abs / 100
  let digits = cents === 'always' ? 2 : cents === 'never' ? 0 : Number.isInteger(major) ? 0 : 2
  if (zeroDecimal.has(cur)) digits = 0
  let out: string
  if (cur === 'usd') {
    out = dollars(digits === 0 ? Math.round(major) : major, digits)
  } else {
    try {
      out = new Intl.NumberFormat('en-US', { style: 'currency', currency: cur.toUpperCase(), minimumFractionDigits: digits, maximumFractionDigits: digits }).format(major)
    } catch {
      out = `${major.toFixed(digits)} ${cur.toUpperCase()}`
    }
  }
  return neg ? `−${out}` : out
}

/** "$8/month", "$80/year" (an amount with its period). */
export function pricePer(t: T, minor: number, currency: string, period: string | undefined): string {
  const amount = money(minor, currency)
  if (period === 'year') return t(me.perYear, { amount })
  if (period === 'month') return t(me.perMonth, { amount })
  return amount
}

/** "$8 a month" (the readers' wording). */
export function priceA(t: T, minor: number, currency: string, period: string | undefined): string {
  const amount = money(minor, currency)
  return period === 'year' ? t(me.aYear, { amount }) : t(me.aMonth, { amount })
}

/** A day with its year when it isn't this year: "12 Nov", "7 Oct 2027". */
export function day(d: string | null | undefined, now = new Date()): string {
  if (!d) return '—'
  const x = new Date(d)
  return x.getFullYear() === now.getFullYear() ? dayMonthDate(x) : dayMonthYear(x)
}

/** Paid status badges (design system): trialing info, active ok, past due warn, canceled neutral, complimentary purple, gift purple. */
export function statusTone(status: string): Tone {
  switch (status) {
    case 'trialing':
      return 'info'
    case 'active':
      return 'ok'
    case 'past_due':
      return 'warn'
    case 'complimentary':
    case 'gift':
      return 'purple'
  }
  return 'neutral'
}

export function statusLabel(t: T, status: string): string {
  switch (status) {
    case 'trialing':
      return t(me.statusTrialing)
    case 'active':
      return t(me.statusActive)
    case 'past_due':
      return t(me.statusPastDue)
    case 'canceled':
      return t(me.statusCanceled)
    case 'complimentary':
      return t(me.statusComplimentary)
    case 'gift':
      return t(me.statusGift)
    case 'incomplete':
      return t(me.statusIncomplete)
  }
  return status
}

export function reasonLabel(t: T, reason: string | undefined): string {
  switch (reason) {
    case 'team':
      return t(me.reasonTeam)
    case 'contributor':
      return t(me.reasonContributor)
    case 'press':
      return t(me.reasonPress)
    case 'partner':
      return t(me.reasonPartner)
    case 'giveaway':
      return t(me.reasonGiveaway)
    case 'other':
      return t(me.reasonOther)
  }
  return reason ?? ''
}

export function cancelReason(t: T, why: string | undefined): string {
  switch (why) {
    case 'not_reading':
      return t(me.whyNotReading)
    case 'too_expensive':
      return t(me.whyTooExpensive)
    case 'not_for_me':
      return t(me.whyNotForMe)
    case 'switching':
      return t(me.whySwitching)
    case 'other':
      return t(me.whyOther)
  }
  return why ?? ''
}

export function platformName(p: string | undefined): string {
  switch (p) {
    case 'substack':
      return 'Substack'
    case 'ghost':
      return 'Ghost'
    case 'beehiiv':
      return 'beehiiv'
    case 'kit':
      return 'Kit'
  }
  return 'a CSV file'
}

/**
 * The tier with its period: "Insider yearly"; just "Founding member" for a
 * tier sold for one period only, and "Insider" for access without a price.
 */
export function tierLine(t: T, r: MemberRow): string {
  if (r.kind !== 'stripe' || !r.period || !r.periodShown) return r.tier
  return t(me.tierPeriod, { tier: r.tier, period: r.period })
}

/** The price line under the tier: "$64 for the first year with FALL26, then $80/year". */
export function priceLine(t: T, r: MemberRow): string {
  if (r.kind === 'complimentary') return t(me.complimentaryNoPayment)
  if (r.kind === 'gift') {
    const months = r.giftMonths ?? 12
    const length = months === 12 ? 'year' : 'months'
    return r.giftFrom ? t(me.giftFrom, { length, months, from: r.giftFrom }) : t(me.giftFromNoName, { length, months })
  }
  const full = pricePer(t, r.amountCents, r.currency, r.period)
  let line = full
  const d = r.discount
  if (d && d.amountCents < r.amountCents) {
    const amount = money(d.amountCents, r.currency)
    if (d.duration === 'once') line = t(me.discountOnce, { amount, period: r.period, offer: d.label, price: full })
    else if (d.duration === 'forever') line = t(me.discountForever, { amount: pricePer(t, d.amountCents, r.currency, r.period), offer: d.label, price: full })
    else if (d.endsAt) line = t(me.discountUntil, { amount: pricePer(t, d.amountCents, r.currency, r.period), date: day(d.endsAt), offer: d.label, full: money(r.amountCents, r.currency) })
  }
  if (r.status === 'trialing') line = t(me.afterTrial, { price: line })
  if (r.refundedCents > 0) line = `${line} · ${t(me.refundedOn, { amount: money(r.refundedCents, r.currency, 'always'), date: day(r.nextAt) })}`
  if (r.legacy) line = `${line} · ${t(me.legacyPrice)}`
  return line
}

/** "Renews 7 Oct 2027", "Ends 2 Nov", "Stripe tries again 16 Oct". */
export function nextLine(t: T, r: MemberRow): string {
  switch (r.next) {
    case 'renews':
      return r.nextAt ? t(me.renews, { date: day(r.nextAt) }) : '—'
    case 'ends':
      return r.nextAt ? t(me.ends, { date: day(r.nextAt) }) : '—'
    case 'retries':
      return r.nextAt ? t(me.retries, { date: day(r.nextAt) }) : t(me.retriesSoon)
    case 'trial_ends':
      return r.nextAt ? t(me.trialEnds, { date: day(r.nextAt) }) : '—'
    case 'no_end':
      return t(me.noEnd)
    case 'ended': {
      const base = r.nextAt ? t(me.ended, { date: day(r.nextAt) }) : t(me.statusCanceled)
      const why = r.cancelReason ? cancelReason(t, r.cancelReason) : endReason(t, r.endReason)
      return why ? `${base} · ${why}` : base
    }
  }
  return '—'
}

export function endReason(t: T, reason: string | undefined): string {
  switch (reason) {
    case 'trial_ended':
      return t(me.endTrial)
    case 'payment_failed':
      return t(me.endPayment)
    case 'grant_ended':
      return t(me.endGrant)
    case 'gift_ended':
      return t(me.endGift)
    case 'disconnected':
      return t(me.endDisconnected)
    case 'grant_removed':
      return t(me.endRemoved)
  }
  return ''
}

/** Where a paid member came from, as the Source column words it. */
export function sourceLine(t: T, r: MemberRow): string {
  const s = r.source
  switch (s.kind) {
    case 'import':
      return t(me.sourceImport, { platform: platformName(s.platform) })
    case 'grant':
      return s.reason ? t(me.sourceGrantReason, { name: s.grantedBy || s.label, reason: reasonLabel(t, s.reason) }) : t(me.sourceGrant, { name: s.grantedBy || s.label })
    case 'page':
      return s.label || t(me.sourcePage)
    case 'direct':
    case '':
      return s.label || t(me.sourceDirect)
  }
  return s.label
}

/** Counts in the chips: "1,252". */
export const n = count
