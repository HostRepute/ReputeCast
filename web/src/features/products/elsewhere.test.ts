import { beforeAll, describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { setDisplayTimeZone } from '@/lib/format'
import { nodeTitle, refsFrom, triggerLabel, type FlowGraph, type FlowNode, type Refs } from '../automations/graph'
import { boughtGoalTitle, boughtPhrase, boughtRef, boughtTriggerName } from './automationText'
import { cardPrice, checkoutConsentText, productCardValues, productPageUrl, purchaseChip, purchaseLine } from './elsewhere'
import { boughtOptions } from './filter'
import { boughtParam } from './params'

beforeAll(() => setDisplayTimeZone('UTC'))

const now = new Date('2026-10-15T10:30:00Z')
const dl = (d: Partial<Schemas['ProductDownloadState']>): Schemas['ProductDownloadState'] => ({
  state: 'delivered',
  provider: 'Postmark',
  used: 1,
  allowed: 5,
  newLink: false,
  newLinkBy: '',
  revokeReason: '',
  revokedBy: '',
  expiresAt: '2026-10-22T10:12:00Z',
  ...d,
})
const oneTime = (amountCents: number): Schemas['ProductPrice'] => ({ kind: 'one_time', amountCents, minimumCents: 0, suggestedCents: 0 })
const pwyw = (minimumCents: number, suggestedCents: number): Schemas['ProductPrice'] => ({ kind: 'pwyw', amountCents: 0, minimumCents, suggestedCents })

describe('the Purchases card', () => {
  it('words a working link as the design system’s download chip', () => {
    expect(purchaseChip(t, dl({}), now)).toEqual({ tone: 'ok', text: 'Link works until 22 Oct · 4 of 5 downloads left' })
  })
  it('says an expired link the way the board does', () => {
    expect(purchaseChip(t, dl({ state: 'expired', used: 2, expiresAt: '2026-10-10T11:20:00Z' }), now)).toEqual({ tone: 'warn', text: 'Link expired 10 Oct' })
  })
  it('keeps the order panel’s words for the other states', () => {
    expect(purchaseChip(t, dl({ state: 'revoked', used: 0 }), now).text).toBe('Access revoked · 0 of 5 downloads used')
    expect(purchaseChip(t, dl({ state: 'used_up', used: 5 }), now)).toEqual({ tone: 'warn', text: 'Downloads used up · 5 of 5' })
  })
  it('writes when and what was paid, with the order number as it is', () => {
    const o = { paidAt: '2026-10-03T11:20:00Z', amountCents: 2900, currency: 'usd', number: 1107, payWhatYouWant: false }
    expect(purchaseLine(t, o, now)).toBe('3 Oct · $29.00 · order 1107')
    expect(purchaseLine(t, { ...o, number: 12345 }, now)).toBe('3 Oct · $29.00 · order 12345')
    expect(purchaseLine(t, { ...o, amountCents: 2000, payWhatYouWant: true, number: 1310 }, now)).toBe('3 Oct · $20.00, pay what you want · order 1310')
    expect(purchaseLine(t, { ...o, paidAt: '2025-12-30T09:00:00Z' }, now)).toBe('30 Dec 2025 · $29.00 · order 1107')
  })
})

describe('the consent history', () => {
  const ticked = { action: 'subscribed', source: 'product_checkout:deliverability-field-guide', sourceLabel: 'Product checkout', consentText: 'Also send me the Weekly Digest' }
  it('says what a buyer ticked at checkout', () => {
    expect(checkoutConsentText(t, ticked, '15 Oct 2026 10:12')).toBe('Product checkout: ticked “Also send me the Weekly Digest” 15 Oct 2026 10:12')
  })
  it('leaves every other entry as it was', () => {
    expect(checkoutConsentText(t, { ...ticked, action: 'confirmed' }, 'x')).toBeUndefined()
    expect(checkoutConsentText(t, { ...ticked, source: 'form:docs', sourceLabel: 'Signup form · docs' }, 'x')).toBeUndefined()
    expect(checkoutConsentText(t, { ...ticked, consentText: undefined }, 'x')).toBeUndefined()
  })
})

describe('the contacts filter', () => {
  const products = [
    { id: '01a155e8-1bf3-7979-9a20-374b23a93eba', name: 'The Deliverability Field Guide', status: 'published' },
    { id: '01a155e8-1bf3-79c7-a221-797f2b5b0d50', name: 'Newsletter Launch Kit', status: 'archived' },
    { id: '01a155e8-1bf3-79c8-b797-16f2af096836', name: 'Warm-up planner', status: 'draft' },
  ]
  it('offers no filter, any product, then each product that can have buyers', () => {
    expect(boughtOptions(t, products)).toEqual([
      { value: '', label: 'Any purchase history' },
      { value: 'any', label: 'Bought any product' },
      { value: products[0]!.id, label: 'Bought The Deliverability Field Guide' },
      { value: products[1]!.id, label: 'Bought Newsletter Launch Kit' },
    ])
  })
  it('keeps a product from a link or a saved view selectable', () => {
    const gone = '01a155e8-0000-7000-8000-000000000000'
    expect(boughtOptions(t, products, gone).at(-1)).toEqual({ value: gone, label: 'Bought a product' })
    expect(boughtOptions(t, products, 'any')).toHaveLength(4)
  })
  it('reads the search parameter: any, or a product’s id', () => {
    expect(boughtParam.parse('any')).toBe('any')
    expect(boughtParam.parse(products[0]!.id)).toBe(products[0]!.id)
    expect(boughtParam.parse(undefined)).toBeUndefined()
    expect(boughtParam.parse('everything')).toBeUndefined()
    expect(boughtParam.parse('1311')).toBeUndefined()
  })
})

describe('the composer’s Product card', () => {
  it('words a price as the public product card does', () => {
    expect(cardPrice(t, oneTime(2900), 'usd')).toBe('$29')
    expect(cardPrice(t, oneTime(2950), 'usd')).toBe('$29.50')
    expect(cardPrice(t, pwyw(500, 1500), 'usd')).toBe('Pay what you want · from $5')
    expect(cardPrice(t, oneTime(2900), 'eur')).toBe('€29')
  })

  const fields = (f: Partial<Schemas['ProductFields']>): Schemas['ProductFields'] => ({
    name: 'The Deliverability Field Guide',
    slug: 'deliverability-field-guide',
    shortDescription: '',
    longDescription: '',
    coverUrl: '/media/fg/field-guide.png',
    coverAlt: 'Cover of The Deliverability Field Guide',
    price: oneTime(2900),
    salesPageId: 'pg-fg',
    showInShop: true,
    offer: { on: false, listId: '', text: '' },
    delivery: { templateId: '', linkDays: 7, downloads: 5, revokeOnRefund: true },
    ...f,
  })
  const detail = (d: Partial<Schemas['ProductDetail']>) =>
    ({
      live: fields({}),
      draft: fields({ name: 'The Deliverability Field Guide, 2nd edition', price: oneTime(3500) }),
      pages: [
        { id: 'pg-fg', name: 'Field Guide', path: '/field-guide', status: 'published' },
        { id: 'pg-lk', name: 'Launch Kit', path: '/launch-kit', status: 'published' },
      ],
      shopUrl: 'https://news.acme.io/products',
      pageUrl: 'https://news.acme.io/products/deliverability-field-guide',
      payments: { account: 'Acme Media', connected: true, currency: 'usd', hasLive: true, mode: 'live', pricesIncludeTax: false, provider: 'stripe', taxAtCheckout: false },
      ...d,
    }) as Schemas['ProductDetail']

  it('links to the sales page, or to the default product page', () => {
    expect(productPageUrl(detail({}))).toBe('https://news.acme.io/field-guide')
    expect(productPageUrl(detail({ live: fields({ salesPageId: '' }) }))).toBe('https://news.acme.io/products/deliverability-field-guide')
    // Without a custom domain the site lives under the instance's /p/<workspace>.
    expect(productPageUrl(detail({ shopUrl: 'https://mail.example.com/p/acme-newsletter/products' }))).toBe('https://mail.example.com/p/acme-newsletter/field-guide')
    // A draft that was never published uses its own fields.
    expect(productPageUrl(detail({ live: undefined, draft: fields({ salesPageId: 'pg-lk' }) }))).toBe('https://news.acme.io/launch-kit')
    // An address it can't read stays the server's.
    expect(productPageUrl(detail({ shopUrl: '' }))).toBe('https://news.acme.io/products/deliverability-field-guide')
  })
  it('fills the card with what buyers see now, not the draft', () => {
    expect(productCardValues(t, detail({}))).toEqual({
      src: '/media/fg/field-guide.png',
      alt: 'Cover of The Deliverability Field Guide',
      name: 'The Deliverability Field Guide',
      price: '$29',
      href: 'https://news.acme.io/field-guide',
    })
    expect(productCardValues(t, detail({ live: fields({ name: 'Newsletter Launch Kit', price: pwyw(500, 1500), salesPageId: 'pg-lk' }) }))).toMatchObject({
      name: 'Newsletter Launch Kit',
      price: 'Pay what you want · from $5',
      href: 'https://news.acme.io/launch-kit',
    })
  })
})

describe('the automation trigger and goal', () => {
  const fg = '01a155e8-1bf3-7979-9a20-374b23a93eba'
  const refs: Refs = refsFrom({
    emails: [],
    events: [],
    fields: [],
    forms: [],
    lists: [],
    products: [{ id: fg, name: 'The Deliverability Field Guide' }],
    providers: [],
    tags: [],
  })
  const g: FlowGraph = { nodes: [], edges: [] }
  const node = (n: Partial<FlowNode>): FlowNode => ({ id: 'n1', type: 'trigger', position: { x: 0, y: 0 }, data: {}, ...n })

  it('names what it points at', () => {
    expect(boughtTriggerName()).toBe('Bought a product')
    expect(boughtRef(undefined, undefined)).toBe('any product')
    expect(boughtRef(fg, 'The Deliverability Field Guide')).toBe('The Deliverability Field Guide')
    expect(boughtRef(fg, undefined)).toBe('a product')
    expect(boughtPhrase(fg, 'Launch Kit')).toBe('bought Launch Kit')
    expect(boughtPhrase(undefined, undefined)).toBe('bought a product')
    expect(boughtGoalTitle(fg, 'Launch Kit')).toBe('Bought Launch Kit')
    expect(boughtGoalTitle(fg, undefined)).toBe('Bought a product')
  })
  it('titles the steps on the canvas and the list', () => {
    expect(nodeTitle(g, node({ data: { trigger: { kind: 'bought_product', productId: fg } } }), refs)).toBe('The Deliverability Field Guide')
    expect(nodeTitle(g, node({ data: { trigger: { kind: 'bought_product' } } }), refs)).toBe('any product')
    expect(triggerLabel({ kind: 'bought_product', productId: fg }, refs)).toBe('Bought a product · The Deliverability Field Guide')
    expect(nodeTitle(g, node({ type: 'goal', data: { condition: { kind: 'bought_product', productId: fg } } }), refs)).toBe('Bought The Deliverability Field Guide')
    expect(nodeTitle(g, node({ type: 'goal', data: { condition: { kind: 'bought_product' } } }), refs)).toBe('Bought a product')
    expect(nodeTitle(g, node({ type: 'branch', data: { condition: { kind: 'bought_product', productId: fg } } }), refs)).toBe('Bought The Deliverability Field Guide?')
    expect(nodeTitle(g, node({ type: 'filter', data: { condition: { kind: 'bought_product' } } }), refs)).toBe('Only contacts bought a product')
  })
})
