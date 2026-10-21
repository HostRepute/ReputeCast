import { defineMessages } from '@/i18n'

/** Settings › Providers (SettingsProviders): provider summaries and workspace-wide sending defaults. */
export const mp = defineMessages('senders.providers', {
  documentTitle: 'Providers · Settings',
  loadFailed: "Couldn't load the provider settings",
  loading: 'Loading the provider settings…',
  readOnlyText: 'You can view the providers and the sending defaults. Changing them needs the {perm} permission.',

  connectedProviders: 'Connected providers',
  noneConnected: 'No sending provider is connected yet.',
  pageNote: 'Health, routing, warm-up, events and adapters live on the Sending providers page.',
  addProvider: 'Add provider',
  openProviders: 'Open Sending providers',

  // A provider's column.
  streams: 'Streams',
  notRouted: 'Not routed yet',
  connected: 'Connected',
  /** {date} is when; {via} how ("OAuth", "API key"). */
  connectedValue: '{date} · {via}',
  costMonth: 'Cost · {month}',
  costPlan: '{amount} plan',
  costMetered: '{amount} metered',

  // Workspace-wide sending defaults.
  defaultsTitle: 'Workspace-wide sending defaults',
  defaultsLine: 'New campaigns start from these; each campaign can override them in Review & send.',
  bounce: 'Auto-pause · bounce rate',
  complaint: 'Auto-pause · complaint rate',
  mailboxComplaint: 'Per mailbox provider complaint',
  providerError: 'Provider error rate · 3 min',
  pauseScope: 'Pause scope',
  pauseRoute: 'Only the affected route or mailbox provider',
  pauseCampaign: 'The whole campaign',
  confirmOver: 'Confirmation for sends over',
  /** The field's value: {count} is a number of recipients ("5,000"). */
  confirmOverValue: '{count} recipients · type the count to confirm',

  throttlesTitle: 'Throttles per mailbox provider',
  throttlesLine: 'Applied on every route. ReputeCast lowers them automatically when a provider defers.',
  /** A throttle field's value: {n} messages per second ("5"). */
  perSecond: '{n} per second',
  noLimit: 'no limit',
  suppressionSync: 'Suppression sync interval',
  sync15m: 'Every 15 minutes, plus immediately after each bounce',
  sync1h: 'Hourly',
  secrets: 'Webhook signing secrets',
  rotatedOn: 'Rotated {date}',
  notRotated: 'Not rotated since the providers were connected',
  rotate: 'Rotate',
  connectFirst: 'Connect a provider first.',
  discard: 'Discard',
  saveDefaults: 'Save defaults',

  /** {example} is a percentage as the fields write it ("2.0 %"). */
  needNumber: 'Each threshold needs a number, like {example}.',
  /** {example} is "5 per second" and {noLimit} "no limit", as the fields write them. */
  throttleFormat: 'Write the {provider} throttle as "{example}", or "{noLimit}".',
  saved: 'Sending defaults saved',
  savedDetail: 'New campaigns start from them.',

  rotateSecrets: 'Rotate secrets',
  rotateTitle: "Rotate every provider's webhook signing secret?",
  rotateBody:
    "ReputeCast makes a new secret for each provider and registers it where the provider's API allows. Providers whose webhook you set up by hand need the new secret there too; until then their events are refused.",
  rotating: 'Rotating…',
  rotated: '{count, plural, one {Rotated # secret} other {Rotated # secrets}}',
  /** {providers} lists the providers, comma-separated. */
  rotatedByHand: 'Update the webhook by hand in {providers}.',
})
