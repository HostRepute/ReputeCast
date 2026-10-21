import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { setDisplayTimeZone } from '@/lib/format'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { PurchasesCard, purchaseActivity } from './contact'
import { BoughtFilter } from './filter'
import { ReportProductSales } from './report'

beforeAll(() => setDisplayTimeZone('UTC'))
afterEach(() => mockApi({}))

const ws = '/api/v1/workspaces/acme-newsletter'
const contact = '01a155e2-28b4-7b0c-82ef-60303a4bd560'
const owner = workspaceMe({ permissions: ['subscribers:view', 'products:view', 'orders:manage'], workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' } })
const viewer = workspaceMe({ permissions: ['subscribers:view', 'products:view'] })

const purchase = (p: Partial<Schemas['ContactPurchase']>): Schemas['ContactPurchase'] => ({
  id: 'o1107',
  number: 1107,
  paidAt: '2026-10-03T11:20:00Z',
  buyerEmail: 'sam.rivera@outlook.com',
  productId: 'p-fg',
  product: 'Field Guide',
  productName: 'The Deliverability Field Guide',
  amountCents: 2900,
  currency: 'usd',
  payWhatYouWant: false,
  status: 'paid',
  test: false,
  refundedCents: 0,
  disputeStatus: '',
  disputeReason: '',
  download: { state: 'expired', provider: '', used: 2, allowed: 5, expiresAt: '2026-10-10T11:20:00Z', newLink: false, newLinkBy: '', revokedBy: '', revokeReason: '' },
  coverUrl: '/media/fg/field-guide.png',
  coverAlt: 'Cover of The Deliverability Field Guide',
  ...p,
})

const purchases = (items: Schemas['ContactPurchase'][], canManage = true): Schemas['ContactPurchases'] => ({
  items,
  spentCents: items.reduce((s, o) => s + o.amountCents - o.refundedCents, 0),
  currency: 'usd',
  canManage,
})

describe('the Purchases card', () => {
  it('lists what the contact bought, with the download link and the order’s status', async () => {
    mockApi({
      [`GET ${ws}/contacts/${contact}/purchases`]: () =>
        json(
          purchases([
            purchase({}),
            purchase({
              id: 'o1124',
              number: 1124,
              status: 'refunded',
              refundedCents: 2900,
              download: { state: 'revoked', provider: '', used: 2, allowed: 5, newLink: false, newLinkBy: '', revokedBy: '', revokeReason: 'refund' },
            }),
          ]),
        ),
    })
    await renderInWorkspace(<PurchasesCard contactId={contact} />, { me: owner })
    const card = await screen.findByRole('region', { name: 'Purchases · 2' })
    expect(within(card).getByText('$29.00 spent')).toBeInTheDocument()
    const link = within(card).getByRole('link', { name: 'The Deliverability Field Guide, order 1107' })
    expect(link).toHaveAttribute('href', '/w/acme-newsletter/earn/products?order=o1107')
    expect(within(card).getByText(/· \$29\.00 · order 1107$/)).toBeInTheDocument()
    expect(within(card).getByText('Link expired 10 Oct')).toBeInTheDocument()
    expect(within(card).getByText('Paid')).toBeInTheDocument()
    expect(within(card).getByText('Refunded')).toBeInTheDocument()
    expect(within(card).getByText('Access revoked · 2 of 5 downloads used')).toBeInTheDocument()
    // A revoked download can't get a new link: only the paid order offers one.
    expect(within(card).getAllByRole('button', { name: 'Send a new link' })).toHaveLength(1)
  })

  it('sends a new link through the order panel’s dialog, then reads the purchases again', async () => {
    let reads = 0
    let sent: unknown
    const detail = {
      ...purchase({}),
      activity: [],
      route: 'Postmark, then Amazon SES',
      delivery: { template: 'product-delivery', version: 1, provider: 'Postmark', status: 'delivered', newLink: false, seconds: 0 },
      canSendLink: true,
    }
    const working = { state: 'sent', provider: 'Postmark', used: 0, allowed: 5, expiresAt: '2026-10-22T10:00:00Z', newLink: true, newLinkBy: 'Linh Nguyen', revokedBy: '', revokeReason: '' }
    mockApi({
      [`GET ${ws}/contacts/${contact}/purchases`]: () => {
        reads++
        return json(purchases([purchase(reads > 1 ? { download: working as Schemas['ProductDownloadState'] } : {})]))
      },
      [`GET ${ws}/products/orders/o1107`]: () => json(detail),
      [`GET ${ws}/products/p-fg`]: () => json({ id: 'p-fg', draft: { delivery: { linkDays: 7, downloads: 5 } } }),
      [`POST ${ws}/products/orders/o1107/new-link`]: async (r) => {
        sent = await r.json()
        return json({ ...detail, download: working })
      },
    })
    await renderInWorkspace(<PurchasesCard contactId={contact} />, { me: owner })
    fireEvent.click(await screen.findByRole('button', { name: 'Send a new link' }))
    const dialog = await screen.findByRole('dialog', { name: 'Send a new link to sam.rivera@outlook.com?' })
    expect(within(dialog).getByText('1107 · The Deliverability Field Guide · $29.00')).toBeInTheDocument()
    expect(await within(dialog).findByText(/It works for 7 days and 5 downloads/)).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Send the new link' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(sent).toEqual({})
    expect(await screen.findByText('New link works until 22 Oct · 5 of 5 downloads left')).toBeInTheDocument()
    expect(reads).toBe(2)
  })

  it('says why a Viewer can’t send a new link', async () => {
    mockApi({ [`GET ${ws}/contacts/${contact}/purchases`]: () => json(purchases([purchase({})], false)) })
    await renderInWorkspace(<PurchasesCard contactId={contact} />, { me: viewer })
    const button = await screen.findByRole('button', { name: 'Send a new link' })
    expect(button).toHaveAttribute('aria-disabled', 'true')
  })

  it('stays away for contacts who never bought, and without products:view', async () => {
    mockApi({ [`GET ${ws}/contacts/${contact}/purchases`]: () => json(purchases([])) })
    const { container } = await renderInWorkspace(<PurchasesCard contactId={contact} />, { me: owner })
    await waitFor(() => expect(container.querySelector('section')).toBeNull())
    // No request at all without products:view (an unmocked call would fail).
    mockApi({})
    const other = await renderInWorkspace(<PurchasesCard contactId={contact} />, { me: workspaceMe({ permissions: ['subscribers:view'] }) })
    expect(other.container.querySelector('section')).toBeNull()
  })
})

describe('the timeline', () => {
  const item = (type: string): Schemas['ContactActivityItem'] => ({
    id: 'e1',
    category: 'engagement',
    type,
    at: '2026-10-15T10:12:00Z',
    subject: 'The Deliverability Field Guide',
    detail: type === 'bought' ? '$29.00 · order #1311' : '$29.00 refunded · order #1311',
    failover: false,
    machine: false,
  })
  it('names a purchase and a refund, with the server’s detail', () => {
    const bought = purchaseActivity(item('bought'))!
    const { container } = render(<span>{bought.title}</span>)
    expect(container.textContent).toBe('Bought The Deliverability Field Guide')
    expect(container.querySelector('b')?.textContent).toBe('Bought')
    expect(bought.meta).toEqual(['$29.00 · order #1311'])
    const refunded = render(<span>{purchaseActivity(item('refunded'))!.title}</span>)
    expect(refunded.container.textContent).toBe('Refunded The Deliverability Field Guide')
    expect(purchaseActivity({ ...item('bought'), category: 'consent' })).toBeNull()
    expect(purchaseActivity(item('click'))).toBeNull()
  })
})

describe('the contacts filter', () => {
  const overview = (products: { id: string; name: string; status: string }[]) => ({ products })
  it('lists any product and each product, and hands back the choice', async () => {
    mockApi({
      [`GET ${ws}/products`]: () =>
        json(
          overview([
            { id: '01a155e8-1bf3-7979-9a20-374b23a93eba', name: 'The Deliverability Field Guide', status: 'published' },
            { id: '01a155e8-1bf3-79c8-b797-16f2af096836', name: 'Warm-up planner', status: 'draft' },
          ]),
        ),
    })
    let picked: string | undefined = 'none yet'
    await renderInWorkspace(<BoughtFilter value={undefined} onChange={(v) => (picked = v)} />, { me: owner })
    const select = await screen.findByRole('combobox', { name: 'Bought a product' })
    await waitFor(() => expect(within(select).getAllByRole('option')).toHaveLength(3))
    expect(within(select).getAllByRole('option').map((o) => o.textContent)).toEqual(['Any purchase history', 'Bought any product', 'Bought The Deliverability Field Guide'])
    fireEvent.change(select, { target: { value: 'any' } })
    expect(picked).toBe('any')
    fireEvent.change(select, { target: { value: '' } })
    expect(picked).toBeUndefined()
  })

  it('hides without products:view, and while there is nothing to have bought', async () => {
    mockApi({ [`GET ${ws}/products`]: () => json(overview([{ id: 'd', name: 'Warm-up planner', status: 'draft' }])) })
    const { container } = await renderInWorkspace(<BoughtFilter onChange={() => undefined} />, { me: owner })
    await waitFor(() => expect(container.querySelector('select')).toBeNull())
    mockApi({})
    const other = await renderInWorkspace(<BoughtFilter onChange={() => undefined} />, { me: workspaceMe({ permissions: ['subscribers:view'] }) })
    expect(other.container.querySelector('select')).toBeNull()
  })
})

describe('the report’s product sales', () => {
  const sales = (orders: number): Schemas['CampaignProductSales'] => ({
    orders,
    revenueCents: orders ? 11500 : 0,
    currency: 'usd',
    window: 7,
    products: orders
      ? [
          { productId: 'p-fg', name: 'The Deliverability Field Guide', orders: 3, revenueCents: 8700 },
          { productId: 'p-lk', name: 'Newsletter Launch Kit', orders: 1, revenueCents: 2800 },
        ]
      : [],
  })
  it('shows each product’s orders and revenue, and the total', async () => {
    mockApi({ [`GET ${ws}/campaigns/c1/product-sales`]: () => json(sales(4)) })
    await renderInWorkspace(<ReportProductSales campaignId="c1" />, { me: owner })
    const card = await screen.findByRole('region', { name: 'Product sales' })
    const rows = within(card).getAllByRole('row').map((r) => r.textContent)
    expect(rows).toEqual(['ProductOrdersRevenue', 'The Deliverability Field Guide3 orders$87', 'Newsletter Launch Kit1 order$28', 'Total4 orders$115'])
  })

  it('shows nothing when nothing sold, or without products:view', async () => {
    mockApi({ [`GET ${ws}/campaigns/c1/product-sales`]: () => json(sales(0)) })
    const { container } = await renderInWorkspace(<ReportProductSales campaignId="c1" />, { me: owner })
    await waitFor(() => expect(container.querySelector('section')).toBeNull())
    mockApi({})
    const other = await renderInWorkspace(<ReportProductSales campaignId="c1" />, { me: workspaceMe({ permissions: ['reports:view'] }) })
    expect(other.container.querySelector('section')).toBeNull()
  })
})
