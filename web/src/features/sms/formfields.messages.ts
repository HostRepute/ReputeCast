import { defineMessages } from '@/i18n'

/** The form builder's phone field and SMS consent box (brief §3.3, §6). */
export const m = defineMessages('smsFormFields', {
  phone: 'Mobile number',
  optional: ' · optional',
  required: ' · required',
  textsTitle: 'Texts',
  smsConsent: 'SMS consent checkbox · unchecked by default',
  needsPhone: 'Needs the Mobile number field.',
  consentText: 'SMS consent text',
  evidence: 'The number, the lists, this exact text and its version, the page, the IP address and the time are stored as SMS consent evidence.',
  newVersion: 'Saving starts version {version}; earlier signups keep the text they agreed to.',
  separate: 'Email consent never signs anyone up for texts: this box is its own.',
  confirmation: 'After someone ticks it, they get the opt-in confirmation text from Routing & rules.',
  countryAria: 'Country',
  placeholder: '312 555 0123',
  termsWords: 'text message terms',
})
