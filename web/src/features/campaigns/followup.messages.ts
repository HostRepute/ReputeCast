import { defineMessages } from '@/i18n'

/** Follow-ups from a report (review 2026-11 CA-4; CampaignsComplete board, strip 3). */
export const mf = defineMessages('campaigns.followUp', {
  menu: 'Follow up',
  nonOpeners: 'Resend to non-openers…',
  clickers: 'Send to clickers…',
  titleNonOpeners: "Resend to people who didn't open",
  titleClickers: 'Send to people who clicked',
  bodyNonOpeners: '{count, plural, one {<b># contact</b> got "{name}" and hasn\'t opened it.} other {<b># contacts</b> got "{name}" and haven\'t opened it.}}',
  machineOnly: '{count, plural, one {# opened only through a privacy proxy such as Apple Mail Privacy Protection: they count as unknown and are left out.} other {# opened only through a privacy proxy such as Apple Mail Privacy Protection: they count as unknown and are left out.}}',
  bodyClickers: '{count, plural, one {<b># person</b> clicked a link in "{name}".} other {<b># people</b> clicked a link in "{name}".}} Clicks by link scanners don\'t count.',
  nobody: 'Nobody is in this group yet, so the follow-up would reach no one for now. Its audience stays up to date until it goes out.',
  who: 'Send it to',
  whoNonOpeners: "People who didn't open it",
  whoClickers: 'People who clicked',
  subject: 'New subject',
  subjectNonOpeners: 'In case you missed it: {subject}',
  subjectClickers: 'More on: {subject}',
  send: 'Send',
  after: '{days, plural, one {# day after the original ({when})} other {# days after the original ({when})}}',
  later: 'Choose on Review & send',
  note: 'Creates a draft with the same content. Suppressions, unsubscribes and the marketing cap apply as usual.',
  create: 'Create the follow-up',
  creating: 'Creating…',
  created: 'Follow-up "{name}" created',
  loading: 'Counting who it would reach…',
  subjectRequired: 'Give the follow-up a subject.',
})
