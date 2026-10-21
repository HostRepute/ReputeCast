import { defineMessages } from '@/i18n'

/** Earn › Overview's "Product sales" card. */
export const ps = defineMessages('products.sales', {
  title: 'Product sales · 30 days',
  products: 'Products',
  orders: '{count, plural, one {order} other {orders}}',
  gross: 'gross',
  refunded: 'refunded · {count, plural, one {# order} other {# orders}}',
  note: '{net} net from {products, plural, one {# product} other {# products}}, before Stripe’s fees. ReputeCast takes nothing.',
})
