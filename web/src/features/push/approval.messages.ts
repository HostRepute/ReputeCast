import { defineMessages } from '@/i18n'

/** Push and WhatsApp campaigns in the approval review (design expansion phase 8, X22). */
export const m = defineMessages('channelApproval', {
  pushTitle: 'The notification',
  pushCaption: '{n, plural, one {# subscriber} other {# subscribers}} on {sites} · {schedule}',
  link: 'Link',
  buttons: 'Buttons',
  expiry: 'Expires after',
  hours: '{n, plural, one {# hour} other {# hours}}',
  minutes: '{n, plural, one {# minute} other {# minutes}}',
  tag: 'Tag',
  none: '–',
  waTitle: 'The WhatsApp message',
  waCaption: '{template} · {category} · {language} · {n, plural, one {# person} other {# people}} · {schedule}',
  business: 'Sent as',
  variables: 'Variables',
  varField: "'{{'{key}'}}' from {field}",
  varFieldFallback: "'{{'{key}'}}' from {field}, or “{fallback}”",
  varText: "'{{'{key}'}}' {text}",
  cost: 'Estimated cost',
  costLine: '{amount}, charged by Meta when delivered',
})
