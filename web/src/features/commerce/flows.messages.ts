import { defineMessages } from '@/i18n'

/**
 * Store flows from Website tracking (M62, AutomationsComplete strip 8) and
 * the product grid's reader-dependent modes (CommerceEmails): the trigger
 * panels of Viewed a product, Added to cart, Back in stock and Price drop,
 * and the Products select of the product grid.
 */
export const cf = defineMessages('commerce.flows', {
  // ── Trigger names and their canvas line ───────────────────────────────
  /** The palette's group of store triggers (AutomationsComplete strip 8). */
  storeGroup: 'Store',
  kindViewedProduct: 'Viewed a product',
  kindAddedToCart: 'Added to cart',
  kindBackInStock: 'Back in stock',
  kindPriceDrop: 'Price drop',
  startsViewed: "Viewed a product and didn't buy",
  startsCarted: "Added to cart and didn't check out",
  /** a trigger on the canvas: Acme Merch · after 4 hours */
  browseAfter: '{store} · after {hours, plural, one {# hour} other {# hours}}',
  /** a Price drop trigger on the canvas: Acme Merch · 10% or more */
  dropAtLeast: '{store} · {percent}% or more',
  /** a Back in stock trigger on the canvas: Acme Merch · people who asked */
  peopleWhoAsked: '{store} · people who asked',

  // ── The panels ────────────────────────────────────────────────────────
  store: 'Store',
  pickStore: 'Pick a store',
  /** Acme Merch · shop.acme.io */
  storeOption: '{name} · {address}',
  noStore: 'Connect a Shopify or WooCommerce store first: Settings › Integrations.',
  wait: 'Wait',
  waitHint: 'hours without an order or a checkout · 1 to 24',
  only: 'Only products in (optional)',
  onlyHint: 'None chosen: any product in the store.',
  addCollection: 'Add a collection',
  /** name: a collection, Apparel */
  removeCollection: 'Remove {name}',
  after: 'After',
  viewsHint: 'views · 1 to 5',
  cartsHint: 'add-to-carts · 1 to 5',
  /** site: shop.acme.io; store: Acme Merch */
  needsTracking: "Needs Website tracking on the store's site. {site} sends {store}'s product views.",
  /** store: Acme Merch */
  needsTrackingNone: "Needs Website tracking on the store's site: no tracked site is linked to {store} yet.",
  setUpTracking: 'Set up Website tracking',
  /** liquid: {{ browse.products }}, as code */
  browseLiquid: 'This flow’s emails can show what they looked at with the Viewed products block, or {liquid}.',
  dropDays: 'People who viewed it or added it to their cart in the last',
  daysHint: 'days · 1 to 60',
  dropPercent: 'When the price falls by at least',
  percentHint: '% · 1 to 90',
  /** days: an input, 14 */
  viewersToo: 'Also people who viewed it in the last {days} days',
  viewersTooLabel: 'Also people who viewed it',
  days: 'Days',
  askedHint: 'People who asked to hear when a sold-out product is back, from the site’s “Notify me” button or the API. Each of them hears once.',
  /** liquid: {{ stock.product.title }}, as code */
  stockLiquid: 'Emails can name the product with {liquid}.',
  /** product, price: Liquid, as code */
  dropLiquid: 'Emails can name the product and the old price with {product} and {price}.',

  // ── The product grid's Products select (composer) ─────────────────────
  products: 'Products',
  sameForEveryone: 'The same for everyone',
  eachReader: "Each reader's own",
  chosen: 'Chosen products',
  newest: 'Newest from a collection',
  bestSellers: 'Best sellers',
  recentlyViewed: 'Recently viewed',
  boughtTogether: 'Bought together',
  viewed: 'Viewed products',
  anyCollection: 'Any collection',
  bestSellersHint: 'The most units sold in the last 30 days.',
  recentlyViewedHint: "What each reader viewed or added to their cart on the store's site, newest first, without what they bought. Readers without any get the best sellers.",
  boughtTogetherHint: 'What is often bought with what each reader bought or viewed. Readers without any get the best sellers.',
  viewedHint: 'Only in emails of browse, cart and stock flows: the products the trigger named.',
})
