import { defineMessages } from '@/i18n'

/**
 * 7.2 steps 4–6: the test send (SetupTestSend), the streams (SetupStreams)
 * and the contacts (SetupContacts). The timeline's events, a stream's name,
 * description and role, the tips and the import sources come from the
 * server. {provider} is the provider's short name ("SES") or, where the
 * comment says so, its name ("Amazon SES").
 */
export const mst = defineMessages('connect.steps', {
  // ── Test send ────────────────────────────────────────────────────────
  testHeading: 'Send a test and watch it land',
  testLeadSmtp:
    'This proves the whole loop: ReputeCast hands the message to the SMTP server, the server accepts it, and bounces come back through the bounce mailbox.',
  testLead: 'This proves the whole loop: ReputeCast hands the message to {provider}, {provider} delivers it, and the delivery event comes back.',
  testLeadSns: 'This proves the whole loop: ReputeCast hands the message to {provider}, {provider} delivers it, and the delivery event comes back through SNS.',
  testLoadFailed: "Couldn't load the test",
  testLoading: 'Loading the test…',
  continueStreams: 'Continue to streams',
  /** The test message's subject, prefilled; {provider} is its name ("Amazon SES"). */
  defaultSubject: 'ReputeCast test from {provider}',
  testFailed: "Couldn't send the test",
  testMessage: 'Test message',
  from: 'From',
  to: 'To',
  subject: 'Subject',
  sending: 'Sending…',
  addSenderFirst: 'Add a sender first.',
  sendAnother: 'Send another test',
  sendTest: 'Send test',
  sendSeeds: 'Send to a seed list (Gmail, Outlook, Yahoo, iCloud)',
  /** {receiver} is the mailbox that received the test: "Gmail". */
  authSeenBy: 'Authentication seen by {receiver}',
  /** {size} is the message's size in kilobytes. */
  clipped: '{clipped, select, yes {Clipped · {size} KB} other {Not clipped · {size} KB}}',
  /** {addresses} lists the seed addresses. */
  alsoSent: 'Also sent to {addresses}.',
  timelineTitle: 'Delivery timeline',
  listening: 'Listening for events',
  timelineEmpty: 'Send a test: each step shows here as the events come back.',
  seedsSent: 'Test sent to the seed list',
  seedsSentDetail: 'Each mailbox provider shows how it filed the message.',
  seedsTitle: 'Send to a seed list',
  seedsLine: 'One address of yours at each mailbox provider. ReputeCast keeps the list for the next test.',
  enterOne: 'Enter at least one address.',
  send: 'Send',

  // ── Streams ──────────────────────────────────────────────────────────
  rolePrimary: 'Primary',
  roleSplit: 'Weighted split',
  roleFallback: 'Fallback',
  /** {weight} is the provider's share, a percentage. */
  roleSplitWeight: 'Split · {weight}',
  streamsSaveFailed: "Couldn't save the streams",
  /** {provider} is the provider's name ("Amazon SES"). */
  streamsHeading: 'Which streams should {provider} carry?',
  streamsLead: 'A stream is a kind of mail. Each one gets its own route, failover policy and configuration set, so a bad campaign never blocks a password reset.',
  streamsFinished: 'This setup is finished: these are the streams it set up. Routing has its own page now. <link>Open Routing</link>',
  nextTitle: 'Suggested next steps',
  tip: 'Tip',
  set: 'Set',
  pickOne: 'Pick at least one stream for {provider}.',
  continueContacts: 'Continue to contacts',
  /** {providers} lists the providers the stream goes through now. */
  nextTo: 'Next to {providers}',
  primaryReplacing: 'Primary, replacing {providers}',
  share: "{provider}'s share",
  role: 'Role',
  configurationSet: 'Configuration set',
  failover: 'Failover',
  tracking: 'Tracking',

  // ── Contacts ─────────────────────────────────────────────────────────
  contactsHeading: 'Bring your contacts',
  contactsLead: 'Import keeps consent intact: unsubscribed, bounced and complained contacts become suppressions and are never re-subscribed.',
  /** {provider} is the provider's name ("Amazon SES"). */
  contactsLeadPull:
    'Import keeps consent intact: unsubscribed, bounced and complained contacts become suppressions and are never re-subscribed. Existing suppressions can also be pulled from {provider}.',
  optionsFailed: "Couldn't load the import options",
  optionsLoading: 'Loading the import options…',
  finishFailed: "Couldn't finish the setup",
  /** {provider} is the provider's name ("Amazon SES"). */
  pullTitle: 'Pull existing suppressions from {provider}',
  recommended: 'Recommended',
  /** {short} is the provider's short name ("SES"). */
  pullLine:
    'Your {short} account-level suppression list has {count, plural, one {# address} other {# addresses}} from earlier sending. Importing them now means ReputeCast never tries them, on any provider.',
  /** {bounces} and {complaints} are numbers. */
  pullCheckbox: 'Import {count, plural, one {# suppressed address} other {# suppressed addresses}} from {short} (bounces {bounces}, complaints {complaints})',
  suppressionsFrom: 'Suppressions from {provider}',
  emptyTitle: 'Or start empty',
  emptySuggested:
    'Create the lists now and grow them with signup forms, the API or a later import. {count, plural, =1 {One list is} =2 {Two lists are} =3 {Three lists are} =4 {Four lists are} =5 {Five lists are} =6 {Six lists are} other {# lists are}} suggested from your workspace name; rename them any time.',
  emptyKeep:
    "{count, plural, one {Keep the workspace's list and grow them with signup forms, the API or a later import.} other {Keep the workspace's # lists and grow them with signup forms, the API or a later import.}}",
  startEmpty: 'Start empty and finish',
  setupFinished: 'This setup is finished.',
  finish: 'Finish setup',
})
