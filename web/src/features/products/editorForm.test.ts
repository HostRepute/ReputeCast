import { describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { changed, fieldOf, toFields, toForm, validate } from './editorForm'

const draft = (p: Partial<Schemas['ProductFields']> = {}): Schemas['ProductFields'] => ({
  name: 'Warm-up planner',
  slug: 'warm-up-planner',
  shortDescription: 'A four-week sending plan.',
  longDescription: '',
  coverUrl: '',
  coverAlt: '',
  price: { kind: 'one_time', amountCents: 1200, minimumCents: 0, suggestedCents: 0 },
  salesPageId: '',
  showInShop: true,
  offer: { on: false, listId: '', text: '' },
  delivery: { templateId: 't1', linkDays: 7, downloads: 5, revokeOnRefund: true },
  ...p,
})

describe('The product editor’s form (ProductEditor)', () => {
  it('reads a draft and gives the same fields back', () => {
    const f = toForm(draft(), 'usd')
    expect(f.amount).toBe('12.00')
    expect(toFields(f, 'usd')).toEqual(draft())
    expect(changed(f, draft(), 'usd')).toBe(false)
    expect(changed({ ...f, amount: '15' }, draft(), 'usd')).toBe(true)
  })

  it('keeps only the chosen price kind’s amounts', () => {
    const f = { ...toForm(draft(), 'usd'), kind: 'pwyw' as const, minimum: '5', suggested: '15' }
    expect(toFields(f, 'usd').price).toEqual({ kind: 'pwyw', amountCents: 0, minimumCents: 500, suggestedCents: 1500 })
  })

  it('lets a new draft go without a price, and asks for $1 once there is one', () => {
    const f = toForm(draft({ price: { kind: 'one_time', amountCents: 0, minimumCents: 0, suggestedCents: 0 } }), 'usd')
    expect(validate(t, f, 'usd')).toEqual({})
    expect(validate(t, { ...f, amount: '0.50' }, 'usd').amount).toBe('The price must be at least $1')
    expect(validate(t, { ...f, amount: 'ten' }, 'usd').amount).toBe('The price must be at least $1')
  })

  it('checks a pay-what-you-want minimum and suggestion like the server', () => {
    const f = { ...toForm(draft(), 'usd'), kind: 'pwyw' as const }
    expect(validate(t, { ...f, minimum: '0.50', suggested: '5.00' }, 'usd')).toEqual({ minimum: 'The minimum must be at least $1' })
    expect(validate(t, { ...f, minimum: '', suggested: '' }, 'usd').minimum).toBe('The minimum must be at least $1')
    expect(validate(t, { ...f, minimum: '5', suggested: '3' }, 'usd')).toEqual({ suggested: 'The suggested price can’t be below the minimum' })
    expect(validate(t, { ...f, minimum: '5', suggested: '' }, 'usd')).toEqual({})
  })

  it('asks for the list and the text of a list offer that is on', () => {
    const f = { ...toForm(draft(), 'usd'), offerOn: true }
    expect(Object.keys(validate(t, f, 'usd'))).toEqual(['offerListId'])
    expect(Object.keys(validate(t, { ...f, offerListId: 'l1' }, 'usd'))).toEqual(['offerText'])
    expect(validate(t, { ...f, offerListId: 'l1', offerText: 'Also send me the Weekly Digest' }, 'usd')).toEqual({})
  })

  it('keeps names, addresses and the link’s limits in range', () => {
    const f = toForm(draft(), 'usd')
    expect(validate(t, { ...f, name: '  ' }, 'usd').name).toBe('Give the product a name.')
    expect(validate(t, { ...f, slug: 'Warm up!' }, 'usd').slug).toBe('Use lowercase letters, digits and single hyphens.')
    expect(validate(t, { ...f, slug: 'warm--up' }, 'usd').slug).toBeDefined()
    expect(validate(t, { ...f, linkDays: '0' }, 'usd').linkDays).toBe('Use 1 to 30 days')
    expect(validate(t, { ...f, linkDays: '31' }, 'usd').linkDays).toBe('Use 1 to 30 days')
    expect(validate(t, { ...f, downloads: '21' }, 'usd').downloads).toBe('Use 1 to 20 downloads')
  })

  it('places the server’s errors on their fields', () => {
    expect(fieldOf('body.price.minimumCents')).toBe('minimum')
    expect(fieldOf('body.offer')).toBe('offerListId')
    expect(fieldOf('body.delivery.linkDays')).toBe('linkDays')
    expect(fieldOf('body.unknown')).toBeUndefined()
  })
})
