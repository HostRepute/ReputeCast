import { defineMessages } from '@/i18n'

/** The Overview's SMS providers (brief §6). */
export const m = defineMessages('smsOverview', {
  title: 'SMS providers',
  sub: '{month}: {sent} texts sent · {delivered} delivered',
  manage: 'SMS providers & routing',
  budget: '{month} spend',
  of: 'of {budget}',
  noBudget: '{month} spend {spend} · no monthly budget set',
})
