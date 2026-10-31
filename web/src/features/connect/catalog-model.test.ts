import type { Schemas } from '@/api/client'
import { accountNote, priceAt, rank, readVolume } from './catalog-model'

type Entry = Schemas['ConnectCatalogEntry']

// The adapters' price lists as the server sends them (connect/prices.go).
const lists: Record<string, Schemas['ConnectPriceList']> = {
  ses: { kind: 'metered', perThousand: 0.1, dedicatedIp: 24.95 },
  postmark: {
    kind: 'plans',
    plans: [
      { upto: 10000, price: 15 },
      { upto: 50000, price: 55 },
      { upto: 125000, price: 115 },
      { upto: 300000, price: 245 },
    ],
  },
  mailgun: {
    kind: 'plans',
    plans: [
      { upto: 3000, price: 0, free: true, perDay: 100 },
      { upto: 10000, price: 15, name: 'Basic' },
      { upto: 50000, price: 35, name: 'Foundation' },
      { upto: 100000, price: 90, name: 'Scale' },
    ],
    dedicatedPlans: [{ upto: 100000, price: 90, name: 'Scale', dedicated: true }],
  },
  sendgrid: {
    kind: 'plans',
    plans: [
      { upto: 50000, price: 19.95, name: 'Essentials', upTo: true },
      { upto: 100000, price: 34.95, name: 'Essentials', upTo: true },
    ],
  },
  sparkpost: { kind: 'contract' },
  mailjet: { kind: 'plans', plans: [{ upto: 6000, price: 0, free: true }], from: 17 },
  azureacs: { kind: 'metered', perThousand: 0.25, dataPerMb: 0.00012 },
  postal: { kind: 'self_hosted' },
  smtp: { kind: 'relay' },
}

function entry(adapter: string, extra: Partial<Entry> = {}): Entry {
  return {
    adapter,
    name: adapter,
    short: adapter,
    kind: adapter === 'postal' ? 'self_hosted' : adapter === 'smtp' ? 'smtp' : 'api',
    recommended: false,
    terms: '',
    termsTone: 'ok',
    pricing: '',
    chips: [],
    bulkAllowed: true,
    webhooks: true,
    dedicatedIps: true,
    connected: 0,
    bulk: '',
    bulkMark: 'yes',
    setup: '',
    priceList: lists[adapter],
    bulkRank: 2,
    capabilityScore: 5,
    capabilityMax: 5,
    ...extra,
  }
}

describe('Help me choose: prices at a volume', () => {
  it('reads the volume as typed', () => {
    expect(readVolume('50,000')).toBe(50000)
    expect(readVolume('50000')).toBe(50000)
    expect(readVolume('50k')).toBe(50000)
    expect(readVolume('1.5m')).toBe(1500000)
    expect(readVolume('lots')).toBe(0)
  })

  it('prices each adapter at 50,000 a month as the board does', () => {
    expect(priceAt(lists.ses, 50000, false)).toMatchObject({ amount: 'about $5.00', note: '$0.10 per 1,000', cost: 5 })
    expect(priceAt(lists.ses, 50000, true)).toMatchObject({ amount: 'about $29.95', note: 'with one dedicated IP' })
    expect(priceAt(lists.mailgun, 50000, false)).toEqual({ amount: '$35.00', note: 'Foundation plan, 50,000 included', cost: 35 })
    expect(priceAt(lists.mailgun, 50000, true)).toMatchObject({ amount: '$90.00', note: 'Scale plan, 100,000 and one dedicated IP included' })
    expect(priceAt(lists.mailgun, 2000, false)).toMatchObject({ amount: '$0.00', note: 'free plan, 100 a day' })
    expect(priceAt(lists.postmark, 50000, false)).toMatchObject({ amount: '$55.00', note: 'plan for up to 50,000' })
    expect(priceAt(lists.sendgrid, 50000, false)).toMatchObject({ amount: '$19.95', note: 'Essentials plan, up to 50,000' })
    expect(priceAt(lists.sparkpost, 50000, false)).toEqual({ amount: 'n/a', note: 'set by your contract' })
    expect(priceAt(lists.mailjet, 50000, false)).toEqual({ amount: 'n/a', note: 'not in the adapter’s price list; plans from $17' })
    expect(priceAt(lists.azureacs, 50000, false)).toMatchObject({ amount: 'about $12.50', note: 'plus data at $0.00012 per MB' })
    expect(priceAt(lists.postal, 50000, false)).toMatchObject({ amount: '$0 metered', note: 'you pay for your server' })
    expect(priceAt(lists.smtp, 50000, false)).toEqual({ amount: 'n/a', note: 'set by your server or relay' })
    expect(priceAt(lists.postmark, 400000, false)).toEqual({ amount: 'n/a', note: 'above the plans in the adapter’s price list' })
    expect(priceAt(lists.ses, 0, false)).toEqual({ amount: '–', note: '' })
  })
})

describe('Help me choose: the recommendation', () => {
  const all = [
    entry('ses'),
    entry('postmark', { bulkRank: 1 }),
    entry('mailgun'),
    entry('sendgrid', { capabilityScore: 4 }),
    entry('sparkpost'),
    entry('mailjet'),
    entry('azureacs', { bulkRank: 0, capabilityScore: 1 }),
    entry('postal'),
    entry('smtp'),
  ]

  it('ranks by bulk terms, then capabilities, then price: SES, then Mailgun at 50,000', () => {
    const r = rank(all, { volume: 50000, account: '', dedicated: false })
    expect(r.slice(0, 2).map((e) => e.adapter)).toEqual(['ses', 'mailgun'])
    // Unknown prices come after known ones; self-hosted and relays never.
    expect(r.map((e) => e.adapter)).not.toContain('postal')
    expect(r.map((e) => e.adapter)).not.toContain('smtp')
    expect(r.map((e) => e.adapter)).not.toContain('azureacs')
  })

  it('keeps only providers with dedicated IPs when one is needed', () => {
    const r = rank([...all.slice(0, 3).map((e) => (e.adapter === 'mailgun' ? e : { ...e, dedicatedIps: e.adapter === 'ses' }))], {
      volume: 50000,
      account: '',
      dedicated: true,
    })
    expect(r.map((e) => e.adapter)).toEqual(['ses', 'mailgun'])
  })

  it('notes an account the person already has', () => {
    expect(accountNote('', all)).toBe('')
    expect(accountNote('ses', all)).toContain('production access')
    expect(accountNote('mailgun', all)).toBe('You already have mailgun: connecting it skips creating an account. Its card is below.')
  })
})
