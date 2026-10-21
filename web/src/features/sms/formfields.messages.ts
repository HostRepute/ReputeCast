import { defineMessages } from '@/i18n'

/** The form builder's phone field and SMS consent box (brief §3.3, §6). */
export const m = defineMessages('smsFormFields', {
  phone: 'Mobile number',
  phoneOptional: 'Mobile number · optional',
  phoneRequired: 'Mobile number · required',
  textsTitle: 'Texts',
  smsConsent: 'SMS consent checkbox · unchecked by default',
  needsPhone: 'Needs the Mobile number field.',
  consentText: 'SMS consent text',
  evidence: 'The number, the lists, this exact text and its version, the page, the IP address and the time are stored as SMS consent evidence.',
  newVersion: 'Saving starts version {version}; earlier signups keep the text they agreed to.',
  separate: 'Email consent never signs anyone up for texts: this box is its own.',
  confirmation: 'After someone ticks it, they get the opt-in confirmation text from Routing & rules.',
  // The words of an SMS consent text that link to the terms page, as the
  // public form finds them (texting's signup.termsWords on the server).
  termsWords: 'text message terms',
})
