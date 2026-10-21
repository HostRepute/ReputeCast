import { defineMessages } from '@/i18n'

/** A text campaign in the approval inbox and its review (brief §6). */
export const m = defineMessages('smsApproval', {
  title: 'Text message',
  caption: '{chars} characters · {segments, plural, one {# segment} other {# segments}} · {encoding}',
  bubble: 'The text as recipients get it',
  countries: 'Senders and cost by country',
  country: 'Country',
  sender: 'Sender',
  recipients: 'Recipients',
  price: 'Price per segment',
  cost: 'Cost',
  skipped: 'Skipped',
  total: '{recipients, plural, one {# text} other {# texts}} · {cost}',
  schedule: 'Schedule: {schedule}',
})
