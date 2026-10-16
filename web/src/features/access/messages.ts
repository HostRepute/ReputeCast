import { defineMessages } from '@/i18n'

/**
 * The shared parts of the team and access screens (features/access/parts):
 * sign-in methods, the two-factor badge, scopes and the compare matrix's
 * marks. The built-in role names are in lib/roles.
 */
export const m = defineMessages('access', {
  signInPassword: 'Password',
  signInPasskeyFirst: 'Passkey',
  signInPasskey: 'passkey',

  twoFAOn: '{method, select, passkey {On · passkey} other {On · TOTP}}',
  twoFAIdp: 'Via IdP',
  twoFARequired: 'Required · off',
  graceToday: 'grace ends today',
  graceDays: '{days, plural, one {grace ends in # day} other {grace ends in # days}}',
  graceOver: 'grace period over',
  twoFAOff: 'Off',

  // A member's or a key's scope: "Beta Program list only".
  scopeList: '{list} list only',
  scopeOnly: '{name} only',
  scopeLists: '{count, plural, one {# list} other {# lists}}',
  scopeParts: '{parts} only',

  granted: 'Granted',
  notGranted: 'Not granted',
})
