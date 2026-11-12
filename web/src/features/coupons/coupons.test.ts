import type { Schemas } from '@/api/client'
import { zonedMoment } from '@/features/countdown/time'
import { countdownMjml, newCountdown } from '@/features/countdown/block'
import { stripRcAttrs } from '@/features/conditional/mjml'
import { bareAddress, exampleCode, money, redeemedShare, sampleCode } from './api'
import { couponHtml, couponMarker, newCoupon } from './composer/block'
import { slugOf } from './NewPoolDialog'

type Pool = Schemas['CouponPoolListItem']

function pool(p: Partial<Pool>): Pool {
  return {
    id: 'p1',
    name: 'Welcome 15%',
    slug: 'welcome15',
    source: 'shopify',
    status: 'active',
    prefix: 'WELCOME-',
    total: 12400,
    free: 2540,
    sent: 9860,
    redeemed: 312,
    revenueCents: 1824000,
    currency: 'USD',
    lowAt: 200,
    low: false,
    settings: { kind: 'percentage', value: 15 },
    onePerContact: true,
    createdAt: '2026-10-13T15:40:00Z',
    ...p,
  } as Pool
}

describe('pools as the boards write them', () => {
  it('names a pool for Liquid from its name', () => {
    expect(slugOf('Welcome 15%')).toBe('welcome_15')
    expect(slugOf('Partner perks')).toBe('partner_perks')
    expect(slugOf('Ưu đãi Tết')).toBe('uu_ai_tet')
    expect(slugOf('x'.repeat(40))).toHaveLength(32)
  })
  it('writes numbers, codes and addresses', () => {
    expect(money(1824000, 'USD')).toBe('$18,240.00')
    expect(redeemedShare(pool({}))).toBe(3.2)
    expect(redeemedShare(pool({ sent: 0, redeemed: 0 }))).toBe(0)
    expect(exampleCode('WELCOME-')).toBe('WELCOME-7KQ4M2XA')
    expect(sampleCode('WELCOME-')).toBe('WELCOME-SAMPLE1')
    expect(bareAddress('https://shop.acme.io/discount/{code}')).toBe('shop.acme.io/discount/{code}')
  })
})

describe('the Discount code block', () => {
  it('keeps its settings in a marker without "--"', () => {
    const m = couponMarker({ ...newCoupon('welcome15'), label: 'Your code -- now' })
    expect(m.startsWith('<mj-raw><!-- rc:coupon {')).toBe(true)
    expect(m.slice(4, -4).replace('<!--', '').replace('-->', '')).not.toContain('--')
  })
  it('draws the sample, and the button only with an address', () => {
    const b = newCoupon('welcome15')
    expect(couponHtml(b, pool({}))).toContain('WELCOME-SAMPLE1')
    expect(couponHtml(b, pool({}))).toContain(b.button)
    const upload = pool({ source: 'upload', prefix: '', settings: {} })
    expect(couponHtml(b, upload)).not.toContain(b.button)
    expect(couponHtml(b, undefined)).toContain('welcome15')
  })
})

describe('the Countdown block', () => {
  it('reads a local time in its zone, summer time included', () => {
    expect(zonedMoment('2026-11-27T23:59', 'America/New_York')?.toISOString()).toBe('2026-11-28T04:59:00.000Z')
    expect(zonedMoment('2026-07-01T12:00', 'Europe/Berlin')?.toISOString()).toBe('2026-07-01T10:00:00.000Z')
    expect(zonedMoment('tomorrow', 'UTC')).toBeNull()
  })
  it('keeps its settings out of what is compiled', () => {
    const src = countdownMjml(newCountdown('America/New_York', 'vi'))
    expect(src).toContain('rc-countdown=')
    expect(src).toContain('&quot;language&quot;:&quot;vi&quot;')
    expect(stripRcAttrs(src)).not.toContain('rc-countdown')
    expect(newCountdown('UTC', 'pt').language).toBe('en')
  })
})
