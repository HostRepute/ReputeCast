import { translate } from '@/i18n'
import { m } from './contactfilters.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  phone: 'Téléphone',
  anyPhone: 'Téléphone\u00A0: tous',
  hasPhone: 'Avec téléphone',
  noPhone: 'Sans téléphone',
  phoneIn: 'Téléphone\u00A0: {country}',
  smsConsent: 'Consentement SMS',
  anySmsConsent: 'Consentement SMS\u00A0: tous',
  smsConsentAny: 'Consentement SMS sur une liste',
  smsConsentNone: 'Aucun consentement SMS',
  smsConsentOn: 'Consentement SMS\u00A0: liste {list}',
  column: 'Téléphone',
})
