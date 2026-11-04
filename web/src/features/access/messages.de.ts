import { translate } from '@/i18n'
import { m } from './messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  signInPassword: 'Passwort',
  signInPasskeyFirst: 'Passkey',
  signInPasskey: 'Passkey',
  twoFAOn: '{method, select, passkey {An · Passkey} other {An · TOTP}}',
  twoFAIdp: 'Über IdP',
  twoFARequired: 'Erforderlich · aus',
  graceToday: 'Schonfrist endet heute',
  graceDays: '{days, plural, one {Schonfrist endet in # Tag} other {Schonfrist endet in # Tagen}}',
  graceOver: 'Schonfrist abgelaufen',
  twoFAOff: 'Aus',
  scopeList: 'nur Liste {list}',
  scopeOnly: 'nur {name}',
  scopeLists: '{count, plural, one {# Liste} other {# Listen}}',
  scopeParts: 'nur {parts}',
  granted: 'Erteilt',
  notGranted: 'Nicht erteilt',
})
