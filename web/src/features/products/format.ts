import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { andList, clock, count, dayMonthDate, decimal, zonedParts } from '@/lib/format'
import type { useT } from '@/i18n'
import { day, money } from '../earn/format'
import { pe } from './editor.messages'
import { pm } from './messages'
import { po } from './order.messages'

type T = ReturnType<typeof useT>
type Price = Schemas['ProductPrice']
type FilesSummary = Schemas['ProductFilesSummary']
type Download = Schemas['ProductDownloadState']
type OrderItem = Schemas['ProductOrderItem']

export { day, money }

/** Stripe's least amount in most currencies (the server's MinimumCents). */
export const MINIMUM_CENTS = 100

// Currencies without minor units (Stripe's zero-decimal currencies).
const zeroDecimal = new Set(['bif', 'clp', 'djf', 'gnf', 'jpy', 'kmf', 'krw', 'mga', 'pyg', 'rwf', 'ugx', 'vnd', 'vuv', 'xaf', 'xof', 'xpf'])

const KB = 1024
const MB = 1024 * KB
const GB = 1024 * MB

/** One decimal, dropped when it's zero: 4.2, 38, 2. */
function oneDecimal(v: number): string {
  const r = Math.round(v * 10) / 10
  return Number.isInteger(r) ? count(r) : decimal(r, 1)
}

/** A file size as the products screens say it: "4.2 MB", "210 KB", "2 GB" (binary units, like the server). */
export function fileSize(t: T, bytes: number): string {
  if (bytes >= GB) return t(pm.sizeGB, { n: oneDecimal(bytes / GB) })
  if (bytes >= MB) return t(pm.sizeMB, { n: oneDecimal(bytes / MB) })
  if (bytes >= KB) return t(pm.sizeKB, { n: count(Math.floor(bytes / KB)) })
  return t(pm.sizeBytes, { n: bytes })
}

/** A product's files in one line: "PDF, 86 pages, 4.2 MB", "XLSX, 210 KB", "2 files, 42 MB". */
export function filesText(t: T, f: FilesSummary): string {
  if (!f.count) return t(pm.filesNone)
  const size = fileSize(t, f.bytes)
  if (f.count > 1) return t(pm.filesMany, { count: f.count, size })
  if (f.pages > 0) return t(pm.filesOnePages, { kind: f.kind, pages: f.pages, size })
  return t(pm.filesOne, { kind: f.kind || t(pe.kindUnknown), size })
}

/** Whether a price can be charged: at least $1 (the minimum, for pay what you want). */
export function isPriced(p: Price): boolean {
  return p.kind === 'pwyw' ? p.minimumCents >= MINIMUM_CENTS : p.amountCents >= MINIMUM_CENTS
}

/** The Price column: "$29", or "Pay what you want" with `priceSub` under it. */
export function priceMain(t: T, p: Price, currency: string): string {
  if (!isPriced(p)) return t(pm.priceNone)
  return p.kind === 'pwyw' ? t(pm.pwyw) : money(p.amountCents, currency)
}

/** Under "Pay what you want": "from $5 · suggested $15". */
export function priceSub(t: T, p: Price, currency: string): string | undefined {
  if (p.kind !== 'pwyw' || !isPriced(p)) return undefined
  const min = money(p.minimumCents, currency)
  return p.suggestedCents > 0 ? t(pm.pwywSub, { min, suggested: money(p.suggestedCents, currency) }) : t(pm.pwywSubMin, { min })
}

/** The price on narrow cards and in checks: "$29", "Pay what you want, from $5". */
export function priceCard(t: T, p: Price, currency: string): string {
  if (!isPriced(p)) return t(pm.priceNone)
  return p.kind === 'pwyw' ? t(pm.pwywCard, { min: money(p.minimumCents, currency) }) : money(p.amountCents, currency)
}

/** The price after the files in the order panel: "$29, one-time", "pay what you want, from $5". */
export function priceShort(t: T, p: Price, currency: string): string {
  if (!isPriced(p)) return t(pm.priceNone)
  return p.kind === 'pwyw' ? t(po.pricePwyw, { min: money(p.minimumCents, currency) }) : t(po.priceOneTime, { price: money(p.amountCents, currency) })
}

/** A price in a sentence ("from $29 to $35"): "$29", "pay what you want from $5". */
export function priceWord(t: T, p: Price, currency: string): string {
  return p.kind === 'pwyw' ? t(pe.pcPwyw, { min: money(p.minimumCents, currency) }) : money(p.amountCents, currency)
}

/** The currency's symbol for money inputs: "$", "€", "¥". */
export function currencySymbol(currency: string): string {
  const cur = (currency || 'usd').toUpperCase()
  try {
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: cur, currencyDisplay: 'narrowSymbol' }).formatToParts(0).find((p) => p.type === 'currency')?.value ?? cur
  } catch {
    return cur
  }
}

/** A money input's text for an amount in minor units: 2900 → "29.00"; 0 → "". */
export function amountInput(minor: number, currency: string): string {
  if (!minor) return ''
  return zeroDecimal.has((currency || 'usd').toLowerCase()) ? String(minor) : (minor / 100).toFixed(2)
}

/**
 * The amount a money input holds, in minor units: "29" and "29.00" → 2900,
 * "1,299.50" → 129950, "12,5" → 1250. Empty is null; text that isn't an
 * amount is NaN.
 */
export function parseAmount(input: string, currency: string): number | null {
  const raw = input.replace(/[\s\xa0\u202f]/g, '')
  if (!raw) return null
  // The currency's symbol or code may stand before or after the number.
  const s = raw.replace(/^[^\d.,]+/, '').replace(/[^\d.,]+$/, '')
  if (!/^(\d[\d.,]*|[.,]\d{1,2})$/.test(s)) return Number.NaN
  let whole = s
  let frac = ''
  const sep = Math.max(s.lastIndexOf('.'), s.lastIndexOf(','))
  if (sep >= 0) {
    const tail = s.slice(sep + 1)
    // A separator followed by 1 or 2 digits at the end is the decimal one; the others group thousands.
    if (tail.length > 0 && tail.length <= 2) {
      whole = s.slice(0, sep)
      frac = tail
    }
  }
  const groups = whole.split(/[.,]/)
  if (groups.length > 1 && (groups[0]!.length < 1 || groups[0]!.length > 3 || groups.slice(1).some((g) => g.length !== 3))) return Number.NaN
  whole = groups.join('')
  const digits = zeroDecimal.has((currency || 'usd').toLowerCase()) ? 0 : 2
  if (digits === 0) return frac ? Number.NaN : Number(whole || '0')
  return Number(whole || '0') * 100 + Number(frac.padEnd(2, '0'))
}

/** Whether two moments fall on the same day in the display timezone. */
export function sameDay(a: string | Date, b: string | Date): boolean {
  const x = zonedParts(a)
  const y = zonedParts(b)
  return x.year === y.year && x.month === y.month && x.day === y.day
}

/** An order's time in tables: "15 Oct 10:12". */
export function orderTime(d: string): string {
  return `${dayMonthDate(d)} ${clock(d)}`
}

/** A time today ("09:47"), or the day and time ("14 Oct 19:02"). */
export function shortWhen(d: string, now: Date = new Date()): string {
  return sameDay(d, now) ? clock(d) : orderTime(d)
}

/** An activity line's time: "10:11" today, else "12 Oct". */
export function eventTime(d: string, now: Date = new Date()): string {
  return sameDay(d, now) ? clock(d) : day(d, now)
}

export type ChipTone = 'ok' | 'warn' | 'bad' | 'neutral'

/** The Download column's chip (design system: the download chip). */
export function downloadChip(t: T, d: Download, now: Date = new Date()): { tone: ChipTone; text: string } {
  const used = d.used
  const allowed = d.allowed
  const newLink = (tone: ChipTone, fallback: string) =>
    d.newLink && d.newLinkAt
      ? { tone: 'warn' as const, text: d.newLinkBy ? t(pm.dlNewLink, { when: orderTime(d.newLinkAt), name: d.newLinkBy }) : t(pm.dlNewLinkNoName, { when: orderTime(d.newLinkAt) }) }
      : { tone, text: fallback }
  switch (d.state) {
    case 'delivered':
      return newLink('ok', d.provider ? t(pm.dlDelivered, { provider: d.provider, used, allowed }) : t(pm.dlDeliveredNoProvider, { used, allowed }))
    case 'sent':
      return newLink('ok', t(pm.dlSent, { time: d.sentAt ? shortWhen(d.sentAt, now) : '', used, allowed }))
    case 'pending':
      return { tone: 'neutral', text: t(pm.dlPending) }
    case 'failed':
      return { tone: 'bad', text: d.error ? t(pm.dlFailedReason, { reason: d.error }) : t(pm.dlFailed) }
    case 'expired':
      return { tone: 'warn', text: t(pm.dlExpired, { used, allowed, date: d.expiresAt ? day(d.expiresAt, now) : '' }) }
    case 'used_up':
      return { tone: 'warn', text: t(pm.dlUsedUp, { used, allowed }) }
    case 'revoked':
      return { tone: 'neutral', text: t(pm.dlRevoked) }
    case 'revoked_auto':
      return { tone: 'bad', text: t(pm.dlRevokedAuto) }
  }
  return { tone: 'neutral', text: t(pm.dlPending) }
}

/** The order panel's download chip: "Link works until 22 Oct · 4 of 5 downloads left". */
export function panelChip(t: T, d: Download, now: Date = new Date()): { tone: ChipTone; text: string } {
  const used = d.used
  const allowed = d.allowed
  const left = Math.max(0, allowed - used)
  const date = d.expiresAt ? day(d.expiresAt, now) : ''
  switch (d.state) {
    case 'delivered':
    case 'sent':
      return { tone: 'ok', text: d.newLink ? t(po.pdNewWorks, { date, left, allowed }) : t(po.pdWorks, { date, left, allowed }) }
    case 'pending':
      return { tone: 'neutral', text: t(po.pdPending) }
    case 'failed':
      return { tone: 'bad', text: d.error ? t(po.pdFailedReason, { reason: d.error }) : t(po.pdFailed) }
    case 'expired':
      return { tone: 'warn', text: t(po.pdExpired, { date, used, allowed }) }
    case 'used_up':
      return { tone: 'warn', text: t(po.pdUsedUp, { used, allowed }) }
    case 'revoked':
      return { tone: 'neutral', text: t(po.pdRevoked, { used, allowed }) }
    case 'revoked_auto':
      return { tone: 'bad', text: t(po.pdRevokedAuto, { used, allowed }) }
  }
  return { tone: 'neutral', text: t(po.pdPending) }
}

/** An order's status badge and the line under it (design system: order status badges). */
export function orderStatus(t: T, o: OrderItem, now: Date = new Date()): { tone: Tone; label: string; dot: boolean; sub?: string } {
  if (o.test) return { tone: 'purple', label: t(pm.stTest), dot: false, sub: t(pm.testSub) }
  switch (o.status) {
    case 'paid':
      return { tone: 'ok', label: t(pm.stPaid), dot: true }
    case 'refunded':
      return { tone: 'neutral', label: t(pm.stRefunded), dot: false, sub: o.refundedAt ? t(pm.refundedSub, { when: orderTime(o.refundedAt) }) : undefined }
    case 'partially_refunded':
      return {
        tone: 'neutral',
        label: t(pm.stPartial),
        dot: false,
        sub: o.refundedAt ? t(pm.partialSub, { amount: money(o.refundedCents, o.currency, 'always'), when: orderTime(o.refundedAt) }) : undefined,
      }
    case 'disputed': {
      const reason = o.disputeReason
      let sub: string | undefined
      if (o.disputeStatus === 'won') sub = reason ? t(pm.disputeWonSub, { reason }) : t(pm.disputeWonSubNoReason)
      else if (o.disputeStatus === 'lost') sub = reason ? t(pm.disputeLostSub, { reason }) : t(pm.disputeLostSubNoReason)
      else if (o.disputedAt) sub = reason ? t(pm.disputeOpenSub, { date: day(o.disputedAt, now), reason }) : t(pm.disputeOpenSubNoReason, { date: day(o.disputedAt, now) })
      return { tone: 'bad', label: t(pm.stDisputed), dot: true, sub }
    }
  }
  return { tone: 'neutral', label: o.status, dot: false }
}

/** "Draft changes: price and long description". */
export function changesText(t: T, changes: string[]): string {
  const words: Record<string, string> = {
    name: t(pe.chName),
    address: t(pe.chAddress),
    shortDescription: t(pe.chShortDescription),
    longDescription: t(pe.chLongDescription),
    cover: t(pe.chCover),
    coverAlt: t(pe.chCoverAlt),
    price: t(pe.chPrice),
    salesPage: t(pe.chSalesPage),
    shop: t(pe.chShop),
    offer: t(pe.chOffer),
    delivery: t(pe.chDelivery),
  }
  return andList(changes.map((c) => words[c] ?? c))
}

/** "about 20 seconds left", "about 3 minutes left"; undefined until the speed is known. */
export function etaText(t: T, remaining: number, bytesPerSecond: number | undefined): string | undefined {
  if (!bytesPerSecond || bytesPerSecond <= 0 || remaining <= 0) return undefined
  const s = remaining / bytesPerSecond
  if (s < 57.5) return t(pe.etaSeconds, { n: Math.max(5, Math.round(s / 5) * 5) })
  return t(pe.etaMinutes, { n: Math.max(1, Math.round(s / 60)) })
}

/** What a file is, from its content: "PDF document", "Excel spreadsheet"; else its extension: "MJML file". */
export function kindName(t: T, kind: string, name: string): string {
  const words: Record<string, string> = {
    pdf: t(pe.kindPdf),
    zip: t(pe.kindZip),
    xlsx: t(pe.kindXlsx),
    docx: t(pe.kindDocx),
    pptx: t(pe.kindPptx),
    epub: t(pe.kindEpub),
    png: t(pe.kindPng),
    jpeg: t(pe.kindJpeg),
    gif: t(pe.kindGif),
    webp: t(pe.kindWebp),
    mp3: t(pe.kindMp3),
    mp4: t(pe.kindMp4),
    m4a: t(pe.kindM4a),
    wav: t(pe.kindWav),
  }
  if (words[kind]) return words[kind]
  const ext = /\.([a-z0-9]{1,8})$/i.exec(name)?.[1]
  if (kind === 'text' && !ext) return t(pe.kindText)
  return ext ? t(pe.kindOther, { ext: ext.toUpperCase() }) : t(pe.kindUnknown)
}

/** What the content check found: "a PDF", "a ZIP archive"; undefined when it couldn't tell. */
export function contentWord(t: T, kind: string): string | undefined {
  const words: Record<string, string> = {
    pdf: t(pe.contentPdf),
    zip: t(pe.contentZip),
    xlsx: t(pe.contentXlsx),
    docx: t(pe.contentDocx),
    pptx: t(pe.contentPptx),
    epub: t(pe.contentEpub),
    png: t(pe.contentPng),
    jpeg: t(pe.contentJpeg),
    gif: t(pe.contentGif),
    webp: t(pe.contentWebp),
    mp3: t(pe.contentMp3),
    mp4: t(pe.contentMp4),
    m4a: t(pe.contentM4a),
    wav: t(pe.contentWav),
    text: t(pe.contentText),
  }
  return words[kind]
}

/**
 * The noun a shop card's button names, as the public shop does: "guide" in
 * "The Deliverability Field Guide", "kit" in "Newsletter Launch Kit";
 * undefined for "See it".
 */
export function seeNoun(name: string): string | undefined {
  const l = name.toLowerCase()
  return ['guide', 'kit', 'planner', 'course', 'book', 'ebook', 'template', 'templates', 'checklist', 'workbook', 'toolkit'].find((w) => l.includes(w))
}

/** The shop card's description, as the public shop writes it: the short description, then the files. */
export function shopDescription(short: string, files: string | undefined): string {
  const s = short.trim()
  // A description that already ends with the files ("…there. PDF, 86 pages.") keeps them once.
  if (!files || s.replace(/\.$/, '').endsWith(files)) return s
  return `${s.replace(/\.$/, '')}. ${files}.`.replace(/^\. /, '')
}

/** A shop card's files, as the public shop says them: "PDF, 86 pages", "XLSX, 210 KB". */
export function shopFiles(t: T, f: FilesSummary): string | undefined {
  if (!f.count) return undefined
  if (f.count === 1 && f.pages > 0) return t(pm.filesPages, { kind: f.kind, pages: f.pages })
  return t(pm.filesOne, { kind: f.kind || t(pe.kindUnknown), size: fileSize(t, f.bytes) })
}

/** The page an order or a checkout came from, without the scheme: "news.acme.io/field-guide". */
export function bareUrl(u: string): string {
  return u.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

/**
 * The opening of a long description, for the small default product page: the first paragraph's whole
 * sentences, up to about `max` characters (always at least one sentence).
 */
export function leadText(text: string, max = 200): string {
  const para =
    text
      .split('\n')
      .map((s) => s.trim())
      .find(Boolean) ?? ''
  if (para.length <= max) return para
  let out = ''
  for (const { segment } of new Intl.Segmenter(undefined, { granularity: 'sentence' }).segment(para)) {
    if (out && (out + segment).trimEnd().length > max) break
    out += segment
  }
  return out.trim()
}

/** Joins "{count} {name}" pairs: "214 Field Guide · 97 Launch Kit". */
export function countsLine(t: T, items: { count: number; name: string }[]): string {
  return items.map((i) => t(pm.statOrdersBy, { count: count(i.count), name: i.name })).join(' · ')
}
