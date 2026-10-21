import type { MessageDescriptor, Values } from '@/i18n'
import { clock, dayMonthDate, getDisplayTimeZone } from '@/lib/format'
import type { BlockCopy } from './blocks'
import { escapeText, splitAt, zoneAbbr, zoneCity, zonedInstant, zoneParts, type BlockKind, type LiveTag } from './model'
import { m } from './messages'

/** The message formatter of useT() or t(). */
export type T = (d: MessageDescriptor, values?: Values) => string

const KIND: Record<BlockKind, MessageDescriptor> = {
  section: m.kindSection,
  columns: m.kindColumns,
  divider: m.kindDivider,
  spacer: m.kindSpacer,
  hero: m.kindHero,
  text: m.kindText,
  image: m.kindImage,
  button: m.kindButton,
  video: m.kindVideo,
  features: m.kindFeatures,
  testimonials: m.kindTestimonials,
  faq: m.kindFaq,
  signup: m.kindSignup,
  countdown: m.kindCountdown,
  event: m.kindEvent,
  product: m.kindProduct,
  products: m.kindProducts,
  store: m.kindStore,
  tiers: m.kindTiers,
  posts: m.kindPosts,
  bio: m.kindBio,
}

const DESC: Record<BlockKind, MessageDescriptor> = {
  section: m.descSection,
  columns: m.descColumns,
  divider: m.descDivider,
  spacer: m.descSpacer,
  hero: m.descHero,
  text: m.descText,
  image: m.descImage,
  button: m.descButton,
  video: m.descVideo,
  features: m.descFeatures,
  testimonials: m.descTestimonials,
  faq: m.descFaq,
  signup: m.descSignup,
  countdown: m.descCountdown,
  event: m.descEvent,
  product: m.descProduct,
  products: m.descProducts,
  store: m.descStore,
  tiers: m.descTiers,
  posts: m.descPosts,
  bio: m.descBio,
}

const CRUMB: Record<BlockKind, MessageDescriptor> = {
  section: m.crumbSection,
  columns: m.crumbColumns,
  divider: m.crumbDivider,
  spacer: m.crumbSpacer,
  hero: m.crumbHero,
  text: m.crumbText,
  image: m.crumbImage,
  button: m.crumbButton,
  video: m.crumbVideo,
  features: m.crumbFeatures,
  testimonials: m.crumbTestimonials,
  faq: m.crumbFaq,
  signup: m.crumbSignup,
  countdown: m.crumbCountdown,
  event: m.crumbEvent,
  product: m.crumbProduct,
  products: m.crumbProducts,
  store: m.crumbStore,
  tiers: m.crumbTiers,
  posts: m.crumbPosts,
  bio: m.crumbBio,
}

export function kindName(t: T, kind: BlockKind): string {
  return t(KIND[kind])
}

export function kindDescription(t: T, kind: BlockKind): string {
  return t(DESC[kind])
}

/** The breadcrumb's word for a block: "hero", "product". */
export function kindCrumb(t: T, kind: BlockKind): string {
  return t(CRUMB[kind])
}

/** The words new blocks start with. */
export function blockCopy(t: T): BlockCopy {
  return {
    section: t(m.kindSection),
    sectionHeading: t(m.newSectionHeading),
    sectionText: t(m.newSectionText),
    columns2: t(m.kindColumns2),
    columns3: t(m.kindColumns3),
    columnHeading: t(m.newColumnHeading),
    columnText: t(m.newColumnText),
    divider: t(m.kindDivider),
    spacer: t(m.kindSpacer),
    hero: t(m.kindHero),
    heroEyebrow: t(m.newHeroEyebrow),
    heroHeading: t(m.newHeroHeading),
    heroLead: t(m.newHeroLead),
    heroButton: t(m.newHeroButton),
    heroArt: t(m.newHeroArt),
    heroCopy: t(m.newHeroCopy),
    text: t(m.kindText),
    textBody: t(m.newText),
    image: t(m.kindImage),
    button: t(m.kindButton),
    buttonLabel: t(m.newButton),
    video: t(m.kindVideo),
    features: t(m.kindFeatures),
    featuresEyebrow: t(m.newFeaturesEyebrow),
    featuresHeading: t(m.newFeaturesHeading),
    feature1: t(m.newFeature1),
    feature2: t(m.newFeature2),
    feature3: t(m.newFeature3),
    itemText: t(m.newItemText),
    testimonials: t(m.kindTestimonials),
    quotesEyebrow: t(m.newQuotesEyebrow),
    quotesHeading: t(m.newQuotesHeading),
    quote: t(m.newQuote),
    quoteName: t(m.newQuoteName),
    quoteRole: t(m.newQuoteRole),
    faq: t(m.kindFaq),
    faqEyebrow: t(m.newFaqEyebrow),
    faqHeading: t(m.newFaqHeading),
    faqQ1: t(m.newFaqQ1),
    faqA1: t(m.newFaqA1),
    faqQ2: t(m.newFaqQ2),
    faqA2: t(m.newFaqA2),
    signup: t(m.kindSignup),
    formHeading: t(m.newFormHeading),
    formHint: t(m.newFormHint),
    countdown: t(m.kindCountdown),
    countdownLabel: t(m.newCountdownLabel),
    event: t(m.kindEvent),
    reminders: t(m.newRemindersText),
    product: t(m.kindProduct),
    // {price} stays a placeholder: the site fills the product's price in.
    buyButton: t(m.newBuyButton, { price: '{price}' }),
    products: t(m.kindProducts),
    store: t(m.kindStore),
    tiers: t(m.kindTiers),
    posts: t(m.kindPosts),
    postsHeading: t(m.newPostsHeading),
    bio: t(m.kindBio),
    bioEyebrow: t(m.newBioEyebrow),
    bioName: t(m.newBioName),
    bioRole: t(m.newBioRole),
    bioText: t(m.newBioText),
  }
}

/** A time zone in a list, with its short name on a day: "America/New_York · EDT"; "UTC" alone. */
export function zoneLabel(t: T, zone: string, on: Date): string {
  const abbr = zoneAbbr(zone, on)
  return abbr && abbr !== zone ? t(m.zoneOption, { zone, abbr }) : zone
}

/** The editor's own time zone: the workspace's, where dates are shown. */
export function displayZone(): string {
  return getDisplayTimeZone() ?? Intl.DateTimeFormat().resolvedOptions().timeZone ?? 'UTC'
}

/**
 * A date and a time on a time zone's clock, in the region format, with
 * the zone's city: "6 Oct", "09:00", "New York".
 */
export function wallText(date: string, time: string, tz: string): { day: string; time: string; zone: string } {
  // The moment that shows that date and time on the display clock formats as them.
  const at = zonedInstant(date, time || '00:00', displayZone())
  return { day: at ? dayMonthDate(at) : date, time: at ? clock(at) : time, zone: zoneCity(tz || 'UTC') }
}

function sameDay(a: Date, b: Date): boolean {
  const zone = displayZone()
  const x = zoneParts(a, zone)
  const y = zoneParts(b, zone)
  return Boolean(x && y && x.year === y.year && x.month === y.month && x.day === y.day)
}

/** When something was saved: "10:41" today, "8 Oct, 09:12" before ("8 Oct" when `short`: the header's room). */
export function whenText(t: T, at: string | Date, now: Date = new Date(), short = false): string {
  const d = at instanceof Date ? at : new Date(at)
  if (Number.isNaN(d.getTime())) return ''
  if (sameDay(d, now)) return clock(d)
  return short ? dayMonthDate(d) : t(m.dayTime, { day: dayMonthDate(d), time: clock(d) })
}

/** A day and a time, always both: "2 Nov, 09:00". */
export function dayTimeText(t: T, at: string | Date): string {
  const d = at instanceof Date ? at : new Date(at)
  return Number.isNaN(d.getTime()) ? '' : t(m.dayTime, { day: dayMonthDate(d), time: clock(d) })
}

/** An event's time in the editor's own zone, as pages.js shows a reader theirs ("13:00–13:30", "New York"). */
export function localTime(start: Date, end: Date): { time: string; city: string } | null {
  const zone = displayZone()
  if (!zone || zone === 'UTC' || zone === 'Etc/UTC') return null
  const time = end.getTime() > start.getTime() ? `${clock(start)}–${clock(end)}` : clock(start)
  return { time, city: zoneCity(zone) }
}

function empty(text: string): string {
  return `<p class="pg-empty">${escapeText(text)}</p>`
}

/** What the canvas shows in a live block while it's drawn, when it couldn't be, or when readers see nothing of it. */
export function livePlaceholder(t: T, tag: LiveTag, attrs: Readonly<Record<string, string>>, state: 'loading' | 'failed' | 'empty'): string {
  if (state === 'loading') return empty(t(m.liveLoading))
  if (state === 'failed') return empty(t(m.liveFailed))
  if (tag === 'rc-countdown') {
    const { date, time } = splitAt(attrs.at)
    const w = wallText(date, time, attrs.tz ?? 'UTC')
    const msg = attrs.end === 'reveal' ? m.countdownEndedReveal : attrs.end === 'message' ? m.countdownEndedMessage : m.countdownEndedHide
    return empty(t(msg, { date: w.day, time: w.time, zone: w.zone }))
  }
  if (tag === 'rc-store-product') return empty(t(m.storeEmpty))
  return empty(t(m.liveFailed))
}
