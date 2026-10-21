import { defineMessages } from '@/i18n'

/** The migration importer's Consent step: SMS consent and its evidence columns (brief §3.3, §6). */
export const m = defineMessages('smsImportConsent', {
  title: 'SMS consent',
  withEvidence: 'Comes in with its evidence',
  withEvidenceText:
    'Rows that say yes, with a date, a source and the consent text, become Subscribed to texts on the target list, recorded as imported with that evidence. Nobody is texted.',
  missing: 'SMS consent needs its evidence: map the {missing} {count, plural, one {column} other {columns}} in Map fields. Until then, numbers come in and SMS consent stays Not subscribed.',
  numbersOnly:
    'Numbers come in without SMS consent: they stay Not subscribed to texts. To bring SMS consent along, map a consent column with its date, source and text in Map fields.',
  date: 'date',
  source: 'source',
  text: 'text',
  mapFields: 'Map fields',
})
