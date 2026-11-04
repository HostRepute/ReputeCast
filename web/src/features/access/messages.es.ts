import { translate } from '@/i18n'
import { m } from './messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  signInPassword: 'Contraseña',
  signInPasskeyFirst: 'Llave de acceso',
  signInPasskey: 'llave de acceso',
  twoFAOn: '{method, select, passkey {Activada · llave de acceso} other {Activada · TOTP}}',
  twoFAIdp: 'Vía IdP',
  twoFARequired: 'Obligatoria · desactivada',
  graceToday: 'el periodo de gracia termina hoy',
  graceDays: '{days, plural, one {queda # día de gracia} other {quedan # días de gracia}}',
  graceOver: 'periodo de gracia terminado',
  twoFAOff: 'Desactivada',
  scopeList: 'solo la lista {list}',
  scopeOnly: 'solo {name}',
  scopeLists: '{count, plural, one {# lista} other {# listas}}',
  scopeParts: 'solo {parts}',
  granted: 'Concedido',
  notGranted: 'No concedido',
})
