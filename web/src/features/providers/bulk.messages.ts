import { defineMessages } from '@/i18n'

/**
 * Routing › the bulk-terms warning (ProvidersBulkWarning): Broadcasts would
 * use a provider whose terms forbid bulk marketing. {name} is a provider's
 * name ("Postmark"); {variant} is its message stream as the adapter names it
 * ("Transactional").
 */
export const mb = defineMessages('providers.bulk', {
  termsAllowed: 'Allowed',
  termsYourPolicy: 'Your own policy',
  termsNotAllowed: 'Not allowed',
  termsBroadcastOnly: 'Broadcast streams only',
  selfHosted: '{name} (self-hosted)',

  streamCreated: '{name} Broadcast stream created',
  streamCreatedDetail: 'Webhooks are registered for it too.',
  streamFailed: "The Broadcast stream wasn't created",

  actionCreate: 'Create Broadcast stream and save',
  actionOther: 'Use {provider} and save',
  actionNone: 'Save without a fallback',
  actionKeep: 'Keep it and save',

  titleFallback: "Broadcasts would fail over to a provider whose terms don't allow bulk marketing",
  titleRoute: "Broadcasts would go through a provider whose terms don't allow bulk marketing",
  /** {where} is the part of the route the provider was set as. */
  bodySet: 'You set the Broadcasts {where, select, fallback {fallback} split {split} other {condition}} to {name}\'s "{variant}"{hasVariant, select, yes { message stream} other {}}.',
  /** {variantLower} is the stream's name in lower case ("transactional"). */
  bodyTerms:
    "{hasVariant, select, yes {{name}'s terms restrict {variantLower} streams to one-to-one mail; newsletters on it risk account suspension.} other {{name}'s terms restrict this account to one-to-one mail; newsletters on it risk account suspension.}}",
  bodyCreatable: '{name} does allow bulk mail on a Broadcast message stream, and ReputeCast can create one.',

  routeTitle: 'Broadcasts route',
  routeSub: 'campaigns',
  /** {smallest} and {largest} are counts of recipients. */
  routeSubVolume: 'campaigns · {smallest} to {largest} recipients per send',
  primary: 'Primary',
  /** {list} lists the other shares: "Postal · 10%, warming up". */
  weightedWith: 'Weighted with {list}',
  /** {share} is a percentage. */
  weightItem: '{provider} · {share}',
  weightItemWarming: '{provider} · {share}, warming up',
  chooseProvider: 'Choose a provider',
  fallback: 'Fallback',
  nonePauseAsk: 'None · pause and ask',
  termsBadge: 'Terms: bulk marketing not allowed',
  failoverPolicy: 'Failover policy',
  failoverHint: 'Applies when the primary fails during a send.',
  policyPause: 'Pause and ask (default for Broadcasts)',
  policyWarm: 'Automatic, only to warmed-up providers',
  policyAuto: 'Automatic',

  fixTitle: 'Fix this',
  createTitle: 'Create a {name} Broadcast stream (recommended)',
  createDescription:
    'ReputeCast creates "ReputeCast Broadcasts" in your {name} server, registers webhooks and uses it as the {role, select, fallback {fallback} other {route}}. About 1 minute.',
  otherTitle: '{role, select, fallback {Use a different fallback} other {Use a different provider}}',
  otherNone: "No other connected provider's terms allow bulk mail yet.",
  otherWarming: "{provider} is the only other provider, and it's warming up: failover to it would be blocked by the warm-up guard until day {day}.",
  otherUse: 'Use {name} instead.',
  noneTitle: 'No fallback, pause and ask',
  /** {primary} is the primary's short name, or "the primary". */
  noneDescription: 'If {primary} fails mid-send, the campaign holds and you decide.',
  thePrimary: 'the primary',
  keepTitle: 'Keep {name} {variant} anyway',
  keepDescription: 'Recorded in the audit log. Send Readiness will show a Block on every broadcast until this is changed.',

  termsTitle: "What each connected provider's terms allow",
  termsSub: "from each adapter's manifest · reviewed quarterly",
  colProvider: 'Provider',
  colBulk: 'Bulk marketing',
  colNotes: 'Notes from the adapter',
  colStreams: 'Streams it can serve',
  notCreated: 'Not created yet in this workspace.',
})
