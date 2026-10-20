import type { Schemas } from '@/api/client'
import type { useT } from '@/i18n'
import { pe } from './editor.messages'
import { MINIMUM_CENTS, amountInput, money, parseAmount } from './format'

type T = ReturnType<typeof useT>
export type Fields = Schemas['ProductFields']

/** The editor's fields as the inputs hold them: amounts and numbers as text. */
export interface Form {
  name: string
  slug: string
  shortDescription: string
  longDescription: string
  coverUrl: string
  coverAlt: string
  kind: 'one_time' | 'pwyw'
  amount: string
  minimum: string
  suggested: string
  salesPageId: string
  showInShop: boolean
  offerOn: boolean
  offerListId: string
  offerText: string
  templateId: string
  linkDays: string
  downloads: string
  revokeOnRefund: boolean
}

export type FormErrors = Partial<Record<keyof Form, string>>

/** The longest short description: the shop's cards show it. */
export const SHORT_MAX = 120

const slugRe = /^[a-z0-9]+(-[a-z0-9]+)*$/

export function toForm(f: Fields, currency: string): Form {
  return {
    name: f.name,
    slug: f.slug,
    shortDescription: f.shortDescription,
    longDescription: f.longDescription,
    coverUrl: f.coverUrl,
    coverAlt: f.coverAlt,
    kind: f.price.kind,
    amount: amountInput(f.price.amountCents, currency),
    minimum: amountInput(f.price.minimumCents, currency),
    suggested: amountInput(f.price.suggestedCents, currency),
    salesPageId: f.salesPageId,
    showInShop: f.showInShop,
    offerOn: f.offer.on,
    offerListId: f.offer.listId,
    offerText: f.offer.text,
    templateId: f.delivery.templateId,
    linkDays: String(f.delivery.linkDays || 7),
    downloads: String(f.delivery.downloads || 5),
    revokeOnRefund: f.delivery.revokeOnRefund,
  }
}

const cents = (s: string, currency: string) => {
  const v = parseAmount(s, currency)
  return v === null || Number.isNaN(v) ? 0 : v
}
const whole = (s: string) => {
  const v = Number(s.trim())
  return Number.isInteger(v) ? v : 0
}

/** The fields to save. A price kind keeps only its own amounts. */
export function toFields(form: Form, currency: string): Fields {
  const pwyw = form.kind === 'pwyw'
  return {
    name: form.name.trim(),
    slug: form.slug.trim().toLowerCase(),
    shortDescription: form.shortDescription.trim(),
    longDescription: form.longDescription,
    coverUrl: form.coverUrl,
    coverAlt: form.coverAlt.trim(),
    price: {
      kind: form.kind,
      amountCents: pwyw ? 0 : cents(form.amount, currency),
      minimumCents: pwyw ? cents(form.minimum, currency) : 0,
      suggestedCents: pwyw ? cents(form.suggested, currency) : 0,
    },
    salesPageId: form.salesPageId,
    showInShop: form.showInShop,
    offer: { on: form.offerOn, listId: form.offerListId, text: form.offerText.trim() },
    delivery: { templateId: form.templateId, linkDays: whole(form.linkDays), downloads: whole(form.downloads), revokeOnRefund: form.revokeOnRefund },
  }
}

/** Whether the form differs from a saved draft. */
export function changed(form: Form, saved: Fields, currency: string): boolean {
  return JSON.stringify(toFields(form, currency)) !== JSON.stringify(toFields(toForm(saved, currency), currency))
}

/**
 * What the server would refuse, said where it happens: the same rules as
 * internal/products Save (a draft may have no price yet; a pay-what-you-want
 * minimum and any price start at $1).
 */
export function validate(t: T, form: Form, currency: string): FormErrors {
  const e: FormErrors = {}
  const min = money(MINIMUM_CENTS, currency)
  if (!form.name.trim()) e.name = t(pe.nameRequired)
  if (!slugRe.test(form.slug.trim().toLowerCase()) || form.slug.trim().length > 80) e.slug = t(pe.addressInvalid)
  if (form.kind === 'one_time') {
    const a = parseAmount(form.amount, currency)
    if (a !== null && (Number.isNaN(a) || (a > 0 && a < MINIMUM_CENTS))) e.amount = t(pe.priceMin, { min })
  } else {
    const lo = parseAmount(form.minimum, currency)
    const hi = parseAmount(form.suggested, currency)
    if (lo === null || Number.isNaN(lo) || lo < MINIMUM_CENTS) e.minimum = t(pe.pwywMin, { min })
    if (hi !== null && (Number.isNaN(hi) || (hi > 0 && lo !== null && !Number.isNaN(lo) && hi < lo))) e.suggested = t(pe.suggestedBelow)
  }
  if (form.offerOn && (!form.offerListId || !form.offerText.trim())) {
    if (!form.offerListId) e.offerListId = t(pe.offerIncomplete)
    else e.offerText = t(pe.offerIncomplete)
  }
  const days = Number(form.linkDays.trim())
  if (!Number.isInteger(days) || days < 1 || days > 30) e.linkDays = t(pe.linkDaysRange)
  const downloads = Number(form.downloads.trim())
  if (!Number.isInteger(downloads) || downloads < 1 || downloads > 20) e.downloads = t(pe.downloadsRange)
  return e
}

/** Where a server error belongs: body.price.minimumCents → minimum. */
export function fieldOf(location: string | undefined): keyof Form | undefined {
  const map: Record<string, keyof Form> = {
    'body.name': 'name',
    'body.slug': 'slug',
    'body.shortDescription': 'shortDescription',
    'body.longDescription': 'longDescription',
    'body.coverUrl': 'coverUrl',
    'body.coverAlt': 'coverAlt',
    'body.price.amountCents': 'amount',
    'body.price.minimumCents': 'minimum',
    'body.price.suggestedCents': 'suggested',
    'body.price.kind': 'amount',
    'body.salesPageId': 'salesPageId',
    'body.offer': 'offerListId',
    'body.offer.listId': 'offerListId',
    'body.offer.text': 'offerText',
    'body.delivery.templateId': 'templateId',
    'body.delivery.linkDays': 'linkDays',
    'body.delivery.downloads': 'downloads',
  }
  return location ? map[location] : undefined
}
