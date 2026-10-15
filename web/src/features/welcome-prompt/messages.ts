import { defineMessages } from '@/i18n'

/** The Welcome series prompt on List health, the form builder and the import summary. */
export const m = defineMessages('welcomePrompt', {
  label: 'Welcome series suggestion',
  title: 'New subscribers of {list} get no welcome email yet',
  body: 'The Welcome series starter sends three emails over the first week.',
  covered: '{count, plural, one {{lists} already has one.} other {{lists} already have one.}}',
  start: 'Start the welcome series',
  notNow: 'Not now',
  dismissed: 'We won’t suggest a welcome series for {list} again',
  dismissedDetail: 'Start one any time from Automations › Starter templates.',
  dismissFailed: 'Couldn’t dismiss the suggestion',
})
