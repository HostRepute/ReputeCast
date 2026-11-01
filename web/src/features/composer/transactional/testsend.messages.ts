import { defineMessages } from '@/i18n'

/** The transactional template editor's Send test (review 2026-11 TX-5; TransactionalComplete board, strip 3). */
export const mts = defineMessages('composer.transactional.testSend', {
  menuItem: 'Send a test email…',
  title: 'Send a test of {key}',
  sendTo: 'Send to',
  ownOnly: 'Without transactional:send, tests go to your own address only.',
  fillWith: 'Fill it with',
  samples: 'The sample values',
  testData: 'Test data · {name}',
  note: 'The draft as it is now, through the System route, with [Test] in the subject. It isn\'t counted in the log\'s numbers.',
  send: 'Send test',
  sending: 'Sending…',
  sent: 'Test sent to {email}',
  sentWith: 'Test sent to {email} with "{name}"',
})
