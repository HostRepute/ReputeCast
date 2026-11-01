import { describe, expect, it } from 'vitest'
import { t } from '@/i18n'
import { blockingRows, checkRows } from './EditorAside'
import type { ProductDetail } from './api'

/** A detail with just what checkRows reads. */
function detail(checks: ProductDetail['checks']): ProductDetail {
  return {
    checks,
    payments: { currency: 'usd' },
    draft: { coverUrl: '', price: { kind: 'one_time', cents: 2900 } },
  } as unknown as ProductDetail
}

const passing: ProductDetail['checks'] = [
  { key: 'stripe', ok: true },
  { key: 'template', ok: true, template: 'product-delivery', version: 1 },
  { key: 'route', ok: true },
]

describe('product publish checks', () => {
  // Review 2026-11 EN-10: the payment-failed notice is a recommendation.
  it('lists an unpublished payment-failed notice without blocking publishing', () => {
    const rows = checkRows(t, detail([...passing, { key: 'failed_notice', ok: false, advisory: true, draft: true, template: 'product-payment-failed', templateId: 'tpl-1' }]))
    const notice = rows.find((r) => r.key === 'failed_notice')
    expect(notice).toMatchObject({ ok: false, advisory: true, fix: 'template', templateId: 'tpl-1' })
    expect(notice?.title).toBe('Publish product-payment-failed too')
    expect(blockingRows(rows)).toEqual([])
  })

  it('puts blocking failures before recommendations', () => {
    const rows = checkRows(
      t,
      detail([
        { key: 'failed_notice', ok: false, advisory: true, template: 'product-payment-failed' },
        { key: 'stripe', ok: false },
        { key: 'template', ok: true, template: 'product-delivery', version: 1 },
      ]),
    )
    expect(rows.map((r) => r.key)).toEqual(['stripe', 'failed_notice', 'template'])
    expect(blockingRows(rows).map((r) => r.key)).toEqual(['stripe'])
  })

  it('shows a published notice as passing', () => {
    const rows = checkRows(t, detail([...passing, { key: 'failed_notice', ok: true, advisory: true, template: 'product-payment-failed', version: 2 }]))
    expect(rows.find((r) => r.key === 'failed_notice')).toMatchObject({ ok: true, title: 'The payment-failed notice is published', detail: 'product-payment-failed v2' })
  })
})
