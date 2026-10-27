import { defineMessages } from '@/i18n'

/** 7.2 step 1 (SetupCatalog): the provider catalog. Names, terms, pricing and chips come from the adapters. */
export const mc = defineMessages('connect.catalog', {
  heading: 'Which provider delivers your mail?',
  lead: "Every adapter reports what it can do. Facts come from each adapter's manifest; terms and prices are reviewed quarterly.",
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

  // ── Help me choose (PR-13) ───────────────────────────────────────────
  chooseTitle: 'Help me choose',
  chooseLine: 'Answer three questions; every card below then shows its price at your volume.',
  volumeLabel: 'Emails per month',
  accountLabel: 'I already have an account with',
  accountNone: 'None',
  /** The account options' names are the providers' own; these two read in the app language. */
  accountAWS: 'Amazon Web Services',
  accountAzureName: 'Microsoft Azure',
  accountPostalName: 'My own Postal server',
  dedicatedLabel: 'Need a dedicated IP?',
  no: 'No',
  yes: 'Yes',
  recommended: 'Recommended',
  runnerUp: 'Runner-up',
  /** {price} is "about $5.00" or "$35.00"; {volume} is the volume typed. */
  recLine:
    '{price} a month at {volume}{dedicated, select, yes { with one dedicated IP} other {}} · bulk marketing {terms} · setup {setup}',
  recNoVolume: 'Enter your monthly volume to see prices.',
  runnerUpLine: '{price} a month{hasNote, select, yes { · {note}} other {}}{all, select, yes { · every capability ReputeCast uses} other {}}',
  runnerUpNoVolume: 'enter a volume',
  rankLine: 'Ranked by bulk-marketing terms, then the capabilities ReputeCast uses, then price at your volume.',
  noRecommendation: 'No built-in provider meets these answers; every card below shows its price.',
  accountNoteAWS: 'Your AWS account may already have SES production access in a region. ReputeCast checks when you connect; if it does, there’s no 24 h wait.',
  accountNotePostal: 'You already run Postal: connect it below for a weighted split, and plan about 3 weeks of warm-up for a new IP.',
  accountNoteAzure: 'You already use Azure: Azure Communication Services can carry Automations and System mail, but its terms exclude Broadcasts.',
  /** {provider} is the provider's name. */
  accountNoteOther: 'You already have {provider}: connecting it skips creating an account. Its card is below.',
  /** A card's price line: "At 50,000 a month". */
  priceAtLabel: 'At {volume} a month',
  priceAtNoVolume: 'At your volume',
  setupLabel: 'Setup',
  detailsLabel: 'Details',
  bulkLabel: 'Bulk marketing',

  // Prices at a volume, from each adapter's price list (US dollars).
  priceNA: 'n/a',
  /** {price} is a dollar amount: "about $5.00". */
  priceAbout: 'about {price}',
  pricePerThousand: '{price} per 1,000',
  priceWithIP: 'with one dedicated IP',
  pricePlusData: 'plus data at {price} per MB',
  priceContract: 'set by your contract',
  priceSelfHosted: '$0 metered',
  priceSelfHostedNote: 'you pay for your server',
  priceRelay: 'set by your server or relay',
  /** {from} is the cheapest paid plan: "$17". */
  priceNotListed: 'not in the adapter’s price list; plans from {from}',
  priceAbove: 'above the plans in the adapter’s price list',
  /** {perDay} is a daily limit: "100". */
  planFreePerDay: 'free plan, {perDay} a day',
  /** {upto} is the plan's monthly emails: "3,000". */
  planFreeIncluded: 'free plan, {upto} included',
  /** {name} is the provider's own plan name ("Foundation"). */
  planNamedIncluded: '{name} plan, {upto} included',
  planNamedUpTo: '{name} plan, up to {upto}',
  planUpTo: 'plan for up to {upto}',
  planDedicated: '{name} plan, {upto} and one dedicated IP included',
})
