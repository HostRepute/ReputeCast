import { defineMessages } from '@/i18n'

/** Contacts: the phone column and the SMS filters (brief §6). */
export const m = defineMessages('smsContactFilters', {
  phone: 'Phone',
  anyPhone: 'Any phone',
  hasPhone: 'Has a phone',
  noPhone: 'No phone',
  phoneIn: 'Phone in {country}',
  smsConsent: 'SMS consent',
  anySmsConsent: 'Any SMS consent',
  smsConsentAny: 'SMS consent on a list',
  smsConsentNone: 'No SMS consent',
  smsConsentOn: 'SMS consent on {list}',
  column: 'Phone',
})
