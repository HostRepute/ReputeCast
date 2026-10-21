import { defineMessages } from '@/i18n'

/*
 * Digital products in other screens (publish-sell brief §6; the server's
 * internal/products/elsewhere.go): the contact profile and the contacts
 * list, the campaign report's product sales, the composer's Product card,
 * and the automation trigger and goal "Bought a product".
 */

/** The contact profile's Purchases card, timeline and consent history, and the contacts filter. */
export const mp = defineMessages('products.purchases', {
  title: 'Purchases · {count}',
  spent: '{amount} spent',
  line: '{date} · {amount} · order {number}',
  linePwyw: '{date} · {amount}, pay what you want · order {number}',
  orderLink: '{product}, order {number}',
  linkExpired: 'Link expired {date}',
  sendLink: 'Send a new link',
  sendLinkAction: 'Sending a new link',
  orderError: 'Couldn’t open order {number}',
  error: 'Couldn’t load the purchases',
  retry: 'Retry',
  // The timeline's events.
  bought: '<b>Bought</b> {product}',
  refunded: '<b>Refunded</b> {product}',
  // The consent history: a buyer who ticked the list offer at checkout.
  checkoutTicked: '{source}: ticked “{text}” {when}',
  // The contacts list's filter.
  filterLabel: 'Bought a product',
  filterAll: 'Any purchase history',
  filterAny: 'Bought any product',
  filterProduct: 'Bought {product}',
  filterUnknown: 'Bought a product',
})

/** The campaign report's product sales (the Revenue section's rows, brief §6). */
export const mr = defineMessages('products.reportSales', {
  title: 'Product sales',
  caption: 'Digital products bought after a click on this email',
  products: 'Products',
  colProduct: 'Product',
  colOrders: 'Orders',
  colRevenue: 'Revenue',
  orders: '{count, plural, one {# order} other {# orders}}',
  total: 'Total',
  note: 'An order counts for the email its buyer clicked last, up to {days, plural, one {# day} other {# days}} before paying. Revenue is after refunds and before Stripe’s fees.',
})

/** The composer's Product card, filled from a digital product. */
export const mf = defineMessages('products.fill', {
  title: 'Fill from a product',
  product: 'Published product',
  pick: 'Pick a published product',
  option: '{name} · {price}',
  pwywFrom: 'Pay what you want · from {min}',
  fill: 'Fill the card',
  filling: 'Filling…',
  hint: 'Fills the image and its alt text, the name, the price, and the button’s link to the product’s page.',
  none: 'No published products yet. Publish one in Earn › Products to fill this card from it.',
  loading: 'Loading the products…',
  filled: 'The card shows {name} now',
  failed: 'Couldn’t fill the card',
})

/** The automation trigger and goal "Bought a product". */
export const ma = defineMessages('products.automation', {
  trigger: 'Bought a product',
  condition: 'Bought a product',
  product: 'Product',
  any: 'Any product',
  deleted: 'A deleted product',
  loading: 'Loading…',
  anyRef: 'any product',
  someRef: 'a product',
  phrase: 'bought {product}',
  phraseAny: 'bought a product',
  goal: 'Bought {product}',
  goalAny: 'Bought a product',
  triggerNote: 'Only contacts enter: a buyer who isn’t a contact never does. A refund takes the contact out of the flow.',
  liquid: 'The flow’s emails can use {name}, {amount} and {number}.',
  conditionNote: 'Counts a paid order since the contact entered the flow. An order refunded in full doesn’t count.',
})
