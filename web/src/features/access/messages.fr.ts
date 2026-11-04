import { translate } from '@/i18n'
import { m } from './messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  signInPassword: 'Mot de passe',
  signInPasskeyFirst: 'Clé d’accès',
  signInPasskey: 'clé d’accès',
  twoFAOn: '{method, select, passkey {Activée · clé d’accès} other {Activée · TOTP}}',
  twoFAIdp: 'Via l’IdP',
  twoFARequired: 'Obligatoire · désactivée',
  graceToday: 'dernier jour du délai de grâce',
  graceDays: '{days, plural, one {encore # jour de délai de grâce} other {encore # jours de délai de grâce}}',
  graceOver: 'délai de grâce écoulé',
  twoFAOff: 'Désactivée',
  scopeList: 'liste {list} uniquement',
  scopeOnly: '{name} uniquement',
  scopeLists: '{count, plural, one {# liste} other {# listes}}',
  scopeParts: '{parts} uniquement',
  granted: 'Accordée',
  notGranted: 'Non accordée',
})
