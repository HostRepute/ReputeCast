import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { describe as describeEntry, entryTime } from './Activity'
import { money, share, storefront } from './api'
import { isStoreTrigger, storeOfTrigger, storeTriggerDefaults } from './automation'

type Item = Schemas['CommerceActivityItem']

const ctx = { store: 'Acme Merch', platform: 'Shopify', list: 'Acme Merch', window: 5 }

function item(p: Partial<Item>): Item {
  return { id: 'a1', kind: 'order', tone: 'neutral', at: '2026-10-15T13:58:00Z', ...p } as Item
}

describe('amounts as the boards write them', () => {
  it('formats orders, totals and prices', () => {
    expect(money(6400, 'USD', 'always')).toBe('$64.00')
    expect(money(2183600, 'USD')).toBe('$21,836')
    expect(money(5800, 'usd', 'auto')).toBe('$58')
    expect(share(23.7)).toBe('23.7%')
    expect(share(18.18)).toBe('18.2%')
  })
  it('names the storefront by its public domain', () => {
    const store = { domain: 'acme-merch.myshopify.com', address: 'shop.acme.io' } as Schemas['CommerceStoreSummary']
    expect(storefront({ publicDomain: 'shop.acme.io', store })).toBe('shop.acme.io')
    expect(storefront({ publicDomain: '', store })).toBe('acme-merch.myshopify.com')
  })
})

describe('the activity log in words (CommerceStates)', () => {
  it("words the order of a customer who didn't accept marketing", () => {
    const d = describeEntry(t, item({ order: '#4421', amount: 6400, currency: 'USD', matched: 'none' }), ctx)
    expect(d.title).toBe('Order #4421 · $64.00')
    expect(d.sub).toBe("A customer who didn't accept marketing and isn't a contact: counted in the totals and attribution, without the email")
  })
  it('words a recovered checkout', () => {
    const d = describeEntry(
      t,
      item({ order: '#4417', amount: 7600, currency: 'USD', customer: 'Grace Liu', accepted: true, contact: true, recovered: true, credited: 'Abandoned cart', creditedStep: 1 }),
      ctx,
    )
    expect(d.title).toBe('Order #4417 · $76.00 · Grace Liu')
    expect(d.sub).toBe('Recovered checkout · credited to Abandoned cart, email 1')
  })
  it('shows a clock for today and a date before', () => {
    const now = new Date('2026-10-15T16:00:00Z')
    expect(entryTime('2026-10-15T13:58:00Z', now)).toMatch(/^\d{2}:\d{2}$/)
    expect(entryTime('2026-10-13T13:58:00Z', now)).toMatch(/Oct/)
  })
})

describe('store triggers', () => {
  const stores = [{ id: 's1', status: 'connected' }] as Schemas['CommerceTriggerStore'][]
  it('knows the store triggers', () => {
    expect(isStoreTrigger('abandoned_checkout')).toBe(true)
    expect(isStoreTrigger('order_placed')).toBe(true)
    expect(isStoreTrigger('joined_list')).toBe(false)
  })
  it('starts Abandoned checkout after an hour, with the 7-day skip rule', () => {
    expect(storeTriggerDefaults('abandoned_checkout', stores)).toEqual({ store: 's1', afterHours: 1, skipDays: 7, reentry: 'always' })
    expect(storeTriggerDefaults('order_placed', stores)).toEqual({ store: 's1' })
    expect(storeTriggerDefaults('order_placed', [...stores, { id: 's2', status: 'connected' } as Schemas['CommerceTriggerStore']])).toEqual({ store: undefined })
    expect(storeTriggerDefaults('joined_list', stores)).toEqual({})
  })
  it("finds the flow's store for the Checkout items block", () => {
    const flow = (kind: string, store?: string) => ({ nodes: [{ type: 'trigger', data: { trigger: { kind, store } as Schemas['AutomationTrigger'] } }] })
    expect(storeOfTrigger(flow('abandoned_checkout', 's1'))).toBe('s1')
    expect(storeOfTrigger(flow('joined_list', 's1'))).toBeUndefined()
  })
})
