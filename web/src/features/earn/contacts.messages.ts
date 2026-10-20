import { defineMessages } from '@/i18n'

/** Paid status in Contacts and the contact profile (EarnInAudience). */
export const mpc = defineMessages('earn.contacts', {
  paidStatus: 'Paid status',
  anyPaidStatus: 'Any paid status',
  withAccess: 'Any paid tier · with access',
  andAbove: '{tier} and above',
  trialing: 'Trialing',
  pastDue: 'Past due',
  complimentary: 'Complimentary',
  gift: 'Gift',
  ended: 'Ended',
  free: 'Free subscribers only',
  tierTitle: 'Paid tier',
  colPaid: 'Paid',
  // The contact profile's Paid subscription card.
  cardTitle: 'Paid subscription',
  cardSince: 'Since {date} · paid status from Stripe',
  cardSinceGrant: 'Since {date} · complimentary access from {publisher}',
  cardSinceGift: 'Since {date} · a gift',
  cardEnded: 'Access ended {date}',
  tier: 'Tier',
  price: 'Price',
  status: 'Status',
  renews: 'Renews',
  ends: 'Ends',
  ltv: 'Lifetime value',
  source: 'Source',
  openStripe: 'Open in Stripe',
  openMember: 'Open paid member',
  giveComp: 'Give complimentary access',
  cancel: 'Cancel subscription',
  resend: 'Resend a member email',
  tierPeriod: '{tier} · {period, select, month {monthly} year {yearly} other {}}',
  // The Activity filter.
  activityPaid: 'Paid',
  noPaidActivity: 'No paid events yet.',
  // The segment builder.
  segmentNote: 'Paid conditions read paid status from Stripe’s events. Test paid members are never included.',
})
