import { defineMessages } from '@/i18n'

/** The Payments group of Settings › Integrations (entries.tsx) and the paid members hint in Import contacts. */
export const mpi = defineMessages('earn.integrations', {
  stripe: 'Stripe',
  paddle: 'Paddle',
  lemonsqueezy: 'Lemon Squeezy',
  byOwnStripe: 'Your own Stripe account',
  byMerchant: 'Merchant of record',
  stripeLine: 'Paid subscriptions and one-time payments, with a customer portal for your readers.',
  merchantLine: 'A merchant of record: it sells to your readers for you and handles sales tax and VAT.',
  capSubscriptions: 'Subscriptions',
  capOneTime: 'One-time payments',
  capPortal: 'Customer portal',
  capTestMode: 'Test mode',
  capTax: 'Tax calculation',
  factRestricted: 'A restricted key, never a secret key',
  factConnected: '{account} · {mode, select, live {Live} other {Test}}',
  factNotInRelease: 'Not in this release',
  notAvailable: 'Not available yet',
  connect: 'Connect',
  manage: 'Manage',
  view: 'View',
  importHint: 'Moving paid members from Substack, Ghost, beehiiv or Kit? <a>Import them in Earn</a>, matched with your Stripe subscriptions.',
})
