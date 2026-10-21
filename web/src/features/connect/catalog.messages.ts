import { defineMessages } from '@/i18n'

/** 7.2 step 1 (SetupCatalog): the provider catalog. Names, terms, pricing and chips come from the adapters. */
export const mc = defineMessages('connect.catalog', {
  heading: 'Which provider delivers your mail?',
  lead: "Every adapter reports what it can do. Badges come from each adapter's manifest; terms are reviewed quarterly.",
  searchPlaceholder: 'Provider name',
  filterBulk: 'Allows bulk marketing',
  filterWebhooks: 'Has bounce webhooks',
  filterDedicated: 'Dedicated IPs',
  /** {builtin} and {community} are numbers of adapters. */
  counts: '{builtin} built-in · {community} community',
  loadFailed: "Couldn't load the providers",
  noMatch: 'No provider matches',
  clearFilters: 'Clear the filters',
  noMatchSearch: 'Nothing called “{query}” with the filters you picked.',
  noMatchFilters: 'No provider has every capability you picked.',
  /** A capability chip the adapter lacks: "Not supported:" is read by screen readers only. */
  notSupported: '<vh>Not supported:</vh> {capability}',
  /** {provider} is the short name: "SES", "Postmark". */
  connectProvider: 'Connect {provider}',
  communityTitle: 'Community adapters',
  communityLine: 'Built with the adapter SDK. Review the maintainer and version before connecting.',
  manageAdapters: 'Manage adapters',
  configure: 'Configure',
  connect: 'Connect',
  loading: 'Loading the providers…',
})
