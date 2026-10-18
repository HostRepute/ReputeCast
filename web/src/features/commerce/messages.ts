import { defineMessages } from '@/i18n'

/**
 * Shopify and WooCommerce (design expansion phase 7): the Commerce cards of
 * the integrations catalog, a store's page, the connect and disconnect
 * dialogs, the activity log, and the stores' parts of the composer,
 * automations, contacts and reports. Platform names come in as `{platform}`.
 */
export const cm = defineMessages('commerce', {
  // ── The catalog's Commerce cards ──────────────────────────────────────
  shopify: 'Shopify',
  woocommerce: 'WooCommerce',
  byShopify: 'A custom app in your store',
  byWoo: 'Store API key and the plugin',
  shopifyLine: 'Orders, products and customers who agreed to marketing, from your Shopify store.',
  wooLine: 'Orders and products from your store. Consent and abandoned checkouts come through the ReputeCast plugin.',
  capOrders: 'Orders',
  capProducts: 'Products',
  capCustomers: 'Customers with consent',
  capCheckouts: 'Abandoned checkouts',
  capWriteBack: 'Consent write-back',
  capAttribution: 'Attribution',
  factShopify: 'Read access, write optional',
  factWoo: 'Uses the WordPress plugin',
  factStore:
    '{name} · {orders, plural, one {# order} other {# orders}} in 30 days{ago, select, none {} other { · last event {ago}}}',
  factStores: '{count, plural, one {# store} other {# stores}} · {name}',
  factImporting: '{name} · first import running',
  factConnecting: '{name} · connecting',
  factWebhookMissing: '{name} · a webhook missing, polling every 15 minutes',
  factRateLimited: '{name} · {platform} is limiting requests',
  factNoPlugin: '{name} · the ReputeCast plugin not found',
  factRevoked: '{name} is disconnected',
})
