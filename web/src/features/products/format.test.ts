import { beforeAll, describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { setDisplayTimeZone } from '@/lib/format'
import {
  amountInput,
  changesText,
  currencySymbol,
  downloadChip,
  etaText,
  fileSize,
  filesText,
  isPriced,
  kindName,
  leadText,
  orderStatus,
  panelChip,
  parseAmount,
  priceCard,
  priceMain,
  priceShort,
  priceSub,
  seeNoun,
  shopDescription,
  shopFiles,
} from './format'

beforeAll(() => setDisplayTimeZone('UTC'))

const now = new Date('2026-10-15T10:30:00Z')
const oneTime = (amountCents: number): Schemas['ProductPrice'] => ({ kind: 'one_time', amountCents, minimumCents: 0, suggestedCents: 0 })
const pwyw = (minimumCents: number, suggestedCents: number): Schemas['ProductPrice'] => ({ kind: 'pwyw', amountCents: 0, minimumCents, suggestedCents })
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
const order = (o: Partial<Schemas['ProductOrderItem']>): Schemas['ProductOrderItem'] => ({
  id: 'o',
  number: 1311,
  paidAt: '2026-10-15T10:12:00Z',
  buyerEmail: 'ana.ruiz@example.com',
  productId: 'p',
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
  download: dl({}),
  ...o,
})

describe('Sizes and files (Products)', () => {
  it('says sizes in binary units with one decimal, dropped when zero', () => {
    expect(fileSize(t, 4.2 * 1024 * 1024)).toBe('4.2 MB')
    expect(fileSize(t, 38 * 1024 * 1024)).toBe('38 MB')
    expect(fileSize(t, 210 * 1024 + 300)).toBe('210 KB')
    expect(fileSize(t, 2 * 1024 * 1024 * 1024)).toBe('2 GB')
    expect(fileSize(t, 500 * 1024 * 1024)).toBe('500 MB')
    expect(fileSize(t, 1)).toBe('1 byte')
  })

  it('sums up a product’s files as the table does', () => {
    expect(filesText(t, { count: 1, kind: 'PDF', pages: 86, bytes: 4.2 * 1024 * 1024 })).toBe('PDF, 86 pages, 4.2 MB')
    expect(filesText(t, { count: 1, kind: 'XLSX', pages: 0, bytes: 210 * 1024 })).toBe('XLSX, 210 KB')
    expect(filesText(t, { count: 2, kind: 'PDF', pages: 86, bytes: 6 * 1024 * 1024 })).toBe('2 files, 6 MB')
    expect(filesText(t, { count: 0, kind: '', pages: 0, bytes: 0 })).toBe('No file yet')
  })

  it('names files on the shop card as the public shop does', () => {
    expect(shopFiles(t, { count: 1, kind: 'PDF', pages: 86, bytes: 1 })).toBe('PDF, 86 pages')
    expect(shopFiles(t, { count: 1, kind: 'ZIP', pages: 0, bytes: 38 * 1024 * 1024 })).toBe('ZIP, 38 MB')
    expect(shopDescription('Eight chapters on getting into the inbox and staying there.', 'PDF, 86 pages')).toBe(
      'Eight chapters on getting into the inbox and staying there. PDF, 86 pages.',
    )
    // A description that already names the files keeps them once (the demo's Field Guide).
    expect(shopDescription('Eight chapters on getting into the inbox and staying there. PDF, 86 pages.', 'PDF, 86 pages')).toBe(
      'Eight chapters on getting into the inbox and staying there. PDF, 86 pages.',
    )
    // The default product page's opening: whole sentences of the first paragraph, about 200 characters.
    const long =
      'Get your newsletter into the inbox and keep it there. Eight practical chapters on authentication, warm-up, list hygiene and what to do when a mailbox provider starts pushing back. Each chapter ends with a one-page checklist.\n\nChapter one starts with SPF.'
    expect(leadText(long)).toBe(
      'Get your newsletter into the inbox and keep it there. Eight practical chapters on authentication, warm-up, list hygiene and what to do when a mailbox provider starts pushing back.',
    )
    expect(leadText('\n  A short one.  \nSecond paragraph.')).toBe('A short one.')
    expect(leadText(`${'Long sentence '.repeat(20)}ends here. Next.`)).toBe(`${'Long sentence '.repeat(20)}ends here.`)
    expect(leadText('')).toBe('')
    expect(seeNoun('The Deliverability Field Guide')).toBe('guide')
    expect(seeNoun('Newsletter Launch Kit')).toBe('kit')
    expect(seeNoun('Office hours')).toBeUndefined()
  })

  it('names a file by its content, or its extension', () => {
    expect(kindName(t, 'pdf', 'guide.pdf')).toBe('PDF document')
    expect(kindName(t, 'xlsx', 'planner.xlsx')).toBe('Excel spreadsheet')
    expect(kindName(t, 'text', 'kit.mjml')).toBe('MJML file')
    expect(kindName(t, '', 'notes')).toBe('File')
  })
})

describe('Prices (Products, ProductEditor)', () => {
  it('words one-time and pay-what-you-want prices', () => {
    expect(priceMain(t, oneTime(2900), 'usd')).toBe('$29')
    expect(priceSub(t, oneTime(2900), 'usd')).toBeUndefined()
    expect(priceMain(t, pwyw(500, 1500), 'usd')).toBe('Pay what you want')
    expect(priceSub(t, pwyw(500, 1500), 'usd')).toBe('from $5 · suggested $15')
    expect(priceSub(t, pwyw(500, 0), 'usd')).toBe('from $5')
    expect(priceCard(t, pwyw(500, 1500), 'usd')).toBe('Pay what you want, from $5')
    expect(priceShort(t, oneTime(2900), 'usd')).toBe('$29, one-time')
    expect(priceShort(t, pwyw(500, 1500), 'usd')).toBe('pay what you want, from $5')
    expect(priceMain(t, oneTime(1250), 'usd')).toBe('$12.50')
  })

  it('needs at least $1 to sell', () => {
    expect(isPriced(oneTime(100))).toBe(true)
    expect(isPriced(oneTime(99))).toBe(false)
    expect(isPriced(pwyw(50, 500))).toBe(false)
    expect(priceMain(t, oneTime(0), 'usd')).toBe('No price yet')
  })

  it('reads and writes money inputs in minor units', () => {
    expect(parseAmount('29', 'usd')).toBe(2900)
    expect(parseAmount('29.00', 'usd')).toBe(2900)
    expect(parseAmount('0.50', 'usd')).toBe(50)
    expect(parseAmount('$1,299.5', 'usd')).toBe(129950)
    expect(parseAmount('12,5', 'eur')).toBe(1250)
    expect(parseAmount('1.299,00', 'eur')).toBe(129900)
    expect(parseAmount('1500', 'jpy')).toBe(1500)
    expect(parseAmount('', 'usd')).toBeNull()
    expect(parseAmount('abc', 'usd')).toBeNaN()
    expect(parseAmount('1.2.3x', 'usd')).toBeNaN()
    expect(amountInput(2900, 'usd')).toBe('29.00')
    expect(amountInput(0, 'usd')).toBe('')
    expect(amountInput(1500, 'jpy')).toBe('1500')
    expect(currencySymbol('usd')).toBe('$')
    expect(currencySymbol('eur')).toBe('€')
  })
})

describe('Orders in words (Products, ProductsStates)', () => {
  it('fills the Download column as the boards do', () => {
    expect(downloadChip(t, dl({}), now)).toEqual({ tone: 'ok', text: 'Delivered via Postmark · 1 of 5 downloads' })
    expect(downloadChip(t, dl({ state: 'sent', used: 0, sentAt: '2026-10-15T09:47:00Z' }), now)).toEqual({ tone: 'ok', text: 'Link sent 09:47 · 0 of 5' })
    expect(downloadChip(t, dl({ state: 'revoked' }), now)).toEqual({ tone: 'neutral', text: 'Access revoked' })
    expect(downloadChip(t, dl({ state: 'revoked_auto', used: 3 }), now)).toEqual({ tone: 'bad', text: 'Access revoked automatically' })
    expect(downloadChip(t, dl({ state: 'expired', used: 2, expiresAt: '2026-10-10T11:20:00Z' }), now)).toEqual({ tone: 'warn', text: '2 of 5 · link expired 10 Oct' })
    expect(downloadChip(t, dl({ state: 'delivered', used: 0, newLink: true, newLinkAt: '2026-10-15T08:00:00Z', newLinkBy: 'Sofia Rossi' }), now)).toEqual({
      tone: 'warn',
      text: 'Link expired · new link sent 15 Oct 08:00 by Sofia Rossi',
    })
    expect(downloadChip(t, dl({ state: 'used_up', used: 5 }), now)).toEqual({ tone: 'warn', text: '5 of 5 · downloads used up' })
    expect(downloadChip(t, dl({ state: 'failed', error: 'bounced' }), now)).toEqual({ tone: 'bad', text: 'Delivery failed · bounced' })
  })

  it('words the panel’s download link', () => {
    expect(panelChip(t, dl({}), now)).toEqual({ tone: 'ok', text: 'Link works until 22 Oct · 4 of 5 downloads left' })
    expect(panelChip(t, dl({ used: 0, newLink: true }), now)).toEqual({ tone: 'ok', text: 'New link works until 22 Oct · 5 of 5 downloads left' })
    expect(panelChip(t, dl({ state: 'revoked', used: 0 }), now)).toEqual({ tone: 'neutral', text: 'Access revoked · 0 of 5 downloads used' })
    expect(panelChip(t, dl({ state: 'revoked_auto', used: 3 }), now)).toEqual({ tone: 'bad', text: 'Access revoked automatically · 3 of 5 downloads used' })
  })

  it('badges each status with what Stripe said', () => {
    expect(orderStatus(t, order({}), now)).toMatchObject({ tone: 'ok', label: 'Paid', dot: true })
    expect(orderStatus(t, order({ status: 'refunded', refundedAt: '2026-10-15T09:05:00Z' }), now)).toMatchObject({ tone: 'neutral', label: 'Refunded', sub: '15 Oct 09:05, in Stripe' })
    expect(orderStatus(t, order({ status: 'disputed', disputeStatus: 'open', disputeReason: 'fraudulent', disputedAt: '2026-10-14T09:00:00Z' }), now)).toMatchObject({
      tone: 'bad',
      label: 'Disputed',
      sub: '14 Oct · “fraudulent”',
    })
    expect(orderStatus(t, order({ test: true }), now)).toMatchObject({ tone: 'purple', label: 'Test', sub: 'no money moved' })
    expect(orderStatus(t, order({ status: 'partially_refunded', refundedCents: 1000, refundedAt: '2026-10-15T09:05:00Z' }), now)).toMatchObject({
      label: 'Partially refunded',
      sub: '$10.00 on 15 Oct 09:05',
    })
  })
})

describe('The editor’s words (ProductEditor)', () => {
  it('lists the draft’s changes', () => {
    expect(changesText(t, ['longDescription'])).toBe('long description')
    expect(changesText(t, ['price', 'longDescription', 'delivery'])).toBe('price, long description and delivery')
  })

  it('estimates the time left in round numbers', () => {
    expect(etaText(t, 1.8 * 1024 * 1024, 92_000)).toBe('about 20 seconds left')
    expect(etaText(t, 1000, 1000)).toBe('about 5 seconds left')
    expect(etaText(t, 300 * 1024 * 1024, 1024 * 1024)).toBe('about 5 minutes left')
    expect(etaText(t, 1000, undefined)).toBeUndefined()
  })
})
