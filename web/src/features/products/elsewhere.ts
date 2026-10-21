import type { Schemas } from '@/api/client'
import type { useT } from '@/i18n'
import { day, money } from '../earn/format'
import { mf, mp } from './elsewhere.messages'
import { panelChip, type ChipTone } from './format'

/*
 * Digital products in other screens (publish-sell brief §6): the words and
 * values the contact profile and the composer's Product card show. The
 * components are in contact.tsx, filter.tsx, report.tsx, composer.tsx and
 * automation.tsx; the automation canvas's words are in automationText.ts.
 */

type T = ReturnType<typeof useT>
type Download = Schemas['ProductDownloadState']
type Purchase = Schemas['ContactPurchase']
type ConsentEvent = Schemas['ContactConsentEvent']
type Price = Schemas['ProductPrice']
type Detail = Schemas['ProductDetail']

/** A contact's purchases (get-contact-purchases, products:view). */
export const PURCHASES = '/api/v1/workspaces/{workspace}/contacts/{contact}/purchases' as const
/** The product sales credited to a campaign (get-campaign-product-sales, products:view). */
export const CAMPAIGN_SALES = '/api/v1/workspaces/{workspace}/campaigns/{campaign}/product-sales' as const

// ── The contact profile ───────────────────────────────────────────────────

/**
 * The Purchases card's download chip: the order panel's ("Link works until
 * 22 Oct · 4 of 5 downloads left"), and "Link expired 10 Oct" as the
 * CommerceEmails board words an expired link.
 */
export function purchaseChip(t: T, d: Download, now: Date = new Date()): { tone: ChipTone; text: string } {
  if (d.state === 'expired' && d.expiresAt) return { tone: 'warn', text: t(mp.linkExpired, { date: day(d.expiresAt, now) }) }
  return panelChip(t, d, now)
}

/** The line under a purchase: "3 Oct · $29.00 · order 1107". */
export function purchaseLine(t: T, o: Pick<Purchase, 'paidAt' | 'amountCents' | 'currency' | 'number' | 'payWhatYouWant'>, now: Date = new Date()): string {
  // An order number is a name, not a quantity: "1311", never "1,311".
  const v = { date: day(o.paidAt, now), amount: money(o.amountCents, o.currency, 'always'), number: String(o.number) }
  return t(o.payWhatYouWant ? mp.linePwyw : mp.line, v)
}

/**
 * A consent history entry for a buyer who ticked a product's list offer at
 * checkout: "Product checkout: ticked “Also send me the Weekly Digest” 15
 * Oct 2026 10:12". Undefined for any other entry, which keeps its wording.
 */
export function checkoutConsentText(t: T, e: Pick<ConsentEvent, 'action' | 'source' | 'sourceLabel' | 'consentText'>, when: string): string | undefined {
  if (e.action !== 'subscribed' || !e.source.startsWith('product_checkout:') || !e.consentText) return undefined
  return t(mp.checkoutTicked, { source: e.sourceLabel, text: e.consentText, when })
}

// ── The composer's Product card ───────────────────────────────────────────

/** A price as the product card says it (design system): "$29", "Pay what you want · from $5". */
export function cardPrice(t: T, p: Price, currency: string): string {
  return p.kind === 'pwyw' ? t(mf.pwywFrom, { min: money(p.minimumCents, currency) }) : money(p.amountCents, currency)
}

/**
 * The page that sells a product, from what's live: its sales page, or the
 * default product page. Pages hang off the site's address: the shop's,
 * without "/products" (the server's siteLinks).
 */
export function productPageUrl(d: Pick<Detail, 'live' | 'draft' | 'pages' | 'shopUrl' | 'pageUrl'>): string {
  const f = d.live ?? d.draft
  const base = d.shopUrl.replace(/\/products\/?$/, '')
  if (!d.shopUrl || base === d.shopUrl) return d.pageUrl
  const page = f.salesPageId ? d.pages.find((p) => p.id === f.salesPageId) : undefined
  return page ? `${base}${page.path}` : `${base}/products/${f.slug}`
}

export interface CardValues {
  src: string
  alt: string
  name: string
  price: string
  href: string
}

/** What the Product card shows for a product: its cover, name, price and page, as they're live. */
export function productCardValues(t: T, d: Detail): CardValues {
  const f = d.live ?? d.draft
  return { src: f.coverUrl, alt: f.coverAlt, name: f.name, price: cardPrice(t, f.price, d.payments.currency || 'usd'), href: productPageUrl(d) }
}
