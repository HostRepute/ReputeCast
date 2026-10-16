import { defineMessages } from '@/i18n'

/**
 * The contact's language (Contacts, ContactProfile, LanguageStates): the
 * Contacts filter, the profile's language field with where it came from,
 * and the language version a delivery sent.
 */
export const mc = defineMessages('pageLanguages.contacts', {
  filterLabel: 'Language',
  filterAny: 'Any language',
  filterNone: 'No language',
  filterNotOffered: 'Not offered',
  fieldLabel: 'Language',
  option: '{language} ({code})',
  optionNone: 'No language',
  sourceImport: '{source} import, {date}',
  sourceForm: '{form} form, from the browser, {date}',
  sourceFormNotBrowser: '{form} form, {date}',
  sourceSubscribePage: 'Subscribe page, from the browser, {date}',
  sourcePreferences: 'Preference center, {date}',
  sourceApi: 'Set by the API, {date}',
  sourceApiNamed: 'Set by {name}, {date}',
  sourceTeam: 'Changed by {name}, {date}',
  sourceTeamUnknown: 'Changed by your team, {date}',
  notOffered: '{language} ({code}) · not one of your languages · gets {fallback}. Add {language} in Settings › General and {name} gets it at once.',
  saved: 'Language saved',
  savedDetail: '{name} gets {language} versions where an email has one.',
  saveFailed: 'Couldn’t change the language',
  deliveredFallback: '{language}, the fallback',
  reasonNotOffered: '{language} isn’t one of {workspace}’s languages',
  reasonNoLanguage: 'No language on the contact, so the fallback',
  reasonMissing: 'No {language} version, so the fallback',
  reasonDraft: 'The {language} version was a draft, so the fallback',
  reasonHeld: '{language} readers were sent the fallback instead',
  consentVersionLanguage: 'consent text {version} · {language}',
})

/** Import › Map fields: a column mapped to the contact's language. */
export const mi = defineMessages('pageLanguages.import', {
  normalized: 'Values become language codes: {examples}.',
  example: '“{value}” → {code}',
  unknown:
    '{values, plural, one {# value isn’t a language and is left empty} other {# values aren’t languages and are left empty}} ({rows, plural, one {# row} other {# rows}}): {examples}.',
  unknownExample: '“{value}” ({rows, plural, one {# row} other {# rows}})',
  allKnown: 'Every value is a language ReputeCast knows.',
  languages: 'Languages in this column: {list}.',
  languageCount: '{language} {rows}',
  loading: 'Reading the column’s values…',
  failed: 'Couldn’t read the column’s values: {error}',
})
