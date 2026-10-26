import type { Schemas } from '@/api/client'
import { t as tt } from '@/i18n'
import { count, decimal } from '@/lib/format'
import { mc } from './catalog.messages'

type Entry = Schemas['ConnectCatalogEntry']
type PriceList = Schemas['ConnectPriceList']
type Plan = Schemas['ConnectPricePlan']

/** What a month costs at a volume, as a card's "At 50,000 a month" line shows it. */
export interface PriceAt {
  /** "$35.00", "about $5.00", "n/a", "–". */
  amount: string
  /** What the amount is: "Foundation plan, 50,000 included"; empty when nothing is known. */
  note: string
  /** The month in US dollars when the price list gives one (ranking). */
  cost?: number
}

/** US dollars with cents: "$35.00", "$0.00", "$29.95". */
export function dollars(v: number): string {
  return `$${decimal(Math.round(v * 100) / 100, 2)}`
}

/** A price per unit with the decimals it needs: "$0.10", "$0.00012". */
function unitPrice(v: number): string {
  const decimals = Math.max(2, (v.toString().split('.')[1] ?? '').length)
  return `$${decimal(v, decimals)}`
}

/** "Emails per month" as typed ("50,000", "50000", "50k", "1.5m") as a number; 0 when unreadable. */
export function readVolume(raw: string): number {
  const s = raw.trim().toLowerCase().replace(/[\s,_']/g, '')
  const m = /^(\d+(?:\.\d+)?)(k|m)?$/.exec(s)
  if (!m) return 0
  const n = Number(m[1])
  const v = m[2] === 'k' ? n * 1000 : m[2] === 'm' ? n * 1_000_000 : n
  return Number.isFinite(v) ? Math.round(v) : 0
}

function planNote(p: Plan): string {
  if (p.free) return p.perDay ? tt(mc.planFreePerDay, { perDay: count(p.perDay) }) : tt(mc.planFreeIncluded, { upto: count(p.upto) })
  if (p.dedicated) return tt(mc.planDedicated, { name: p.name ?? '', upto: count(p.upto) })
  if (!p.name) return tt(mc.planUpTo, { upto: count(p.upto) })
  return p.upTo ? tt(mc.planNamedUpTo, { name: p.name, upto: count(p.upto) }) : tt(mc.planNamedIncluded, { name: p.name, upto: count(p.upto) })
}

/** What an adapter's price list says a month costs at a volume (PR-13). */
export function priceAt(list: PriceList | undefined, volume: number, dedicated: boolean): PriceAt {
  if (!list) return { amount: tt(mc.priceNA), note: '' }
  switch (list.kind) {
    case 'contract':
      return { amount: tt(mc.priceNA), note: tt(mc.priceContract) }
    case 'self_hosted':
      return { amount: tt(mc.priceSelfHosted), note: tt(mc.priceSelfHostedNote) }
    case 'relay':
      return { amount: tt(mc.priceNA), note: tt(mc.priceRelay) }
  }
  if (volume <= 0) return { amount: '–', note: '' }
  if (list.kind === 'metered') {
    const per = list.perThousand ?? 0
    const ip = dedicated && list.dedicatedIp ? list.dedicatedIp : 0
    const cost = (volume / 1000) * per + ip
    const note = ip ? tt(mc.priceWithIP) : list.dataPerMb ? tt(mc.pricePlusData, { price: unitPrice(list.dataPerMb) }) : tt(mc.pricePerThousand, { price: unitPrice(per) })
    return { amount: tt(mc.priceAbout, { price: dollars(cost) }), note, cost }
  }
  const plans = (dedicated && list.dedicatedPlans?.length ? list.dedicatedPlans : list.plans) ?? []
  const plan = plans.find((p) => volume <= p.upto)
  if (plan) return { amount: dollars(plan.price), note: planNote(plan), cost: plan.price }
  if (list.from) return { amount: tt(mc.priceNA), note: tt(mc.priceNotListed, { from: `$${decimal(list.from, Number.isInteger(list.from) ? 0 : 2)}` }) }
  return { amount: tt(mc.priceNA), note: tt(mc.priceAbove) }
}

/** The answers of "Help me choose". */
export interface Answers {
  volume: number
  /** The adapter the person already has an account with, or "" for none. */
  account: string
  dedicated: boolean
}

/**
 * Ranks the cards for a recommendation: bulk-marketing terms first, then
 * the capabilities ReputeCast uses, then the price at the volume (unknown
 * prices last). Self-hosted servers and SMTP relays are never recommended:
 * they need a server of your own. With "Need a dedicated IP?" only
 * providers that offer one count.
 */
export function rank(entries: Entry[], a: Answers): Entry[] {
  return entries
    .filter((e) => e.kind === 'api' && e.bulkRank > 0 && (!a.dedicated || e.dedicatedIps))
    .map((e, i) => ({ e, i, c: priceAt(e.priceList, a.volume, a.dedicated).cost }))
    .sort((x, y) => y.e.bulkRank - x.e.bulkRank || y.e.capabilityScore - x.e.capabilityScore || (x.c ?? Infinity) - (y.c ?? Infinity) || x.i - y.i)
    .map((x) => x.e)
}

/** The note under the recommendation for an account the person already has. */
export function accountNote(account: string, entries: Entry[]): string {
  switch (account) {
    case '':
      return ''
    case 'ses':
      return tt(mc.accountNoteAWS)
    case 'postal':
      return tt(mc.accountNotePostal)
    case 'azureacs':
      return tt(mc.accountNoteAzure)
  }
  const e = entries.find((x) => x.adapter === account)
  return e ? tt(mc.accountNoteOther, { provider: e.name }) : ''
}
