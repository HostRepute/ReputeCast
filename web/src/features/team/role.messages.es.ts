import { translate } from '@/i18n'
import { m } from './role.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  eyebrow: 'Diálogo · cambiar rol o ámbito',
  title: 'Cambiar el rol o el ámbito de {name}',
  description: 'Los cambios se aplican al momento, en la siguiente solicitud de {name}. Tanto el valor anterior como el nuevo quedan en el registro de auditoría.',
  nothingChanged: 'Aún no hay cambios.',
  current: '{role} (actual)',
  updated: 'Se actualizó el acceso de {name}',
  scopeChanged: 'ámbito cambiado',
  nowRole: 'ahora {role}',
  ssoGroup: 'El rol de {name} viene de una asignación de grupos de SSO. Un cambio aquí se mantiene hasta que el siguiente inicio de sesión vuelva a aplicar la asignación; cambia la asignación para que sea permanente.',
  allScope: 'Todas las listas y dominios de remitente',
  lastOwnerEyebrow: 'Diálogo · cambiar rol · bloqueado',
  lastOwnerTitle: '{name} es la única persona con el rol Propietario',
  onlyOwnerTransfers: 'Solo quien tiene el rol Propietario puede transferir la propiedad.',
  transfer: 'Transferir la propiedad',
  cantDemote: 'No se puede bajar de rol',
  keepsOwner: 'Un espacio de trabajo siempre conserva al menos un Propietario, así que este rol no se puede cambiar a Administrador y el miembro no se puede quitar ni desactivar.',
  transferFirst: 'Para cambiarlo, primero transfiere la propiedad a otro miembro con la verificación en dos pasos activada. {name} pasa entonces automáticamente al rol Administrador.',
  transferFirstSuggest: 'Para cambiarlo, primero transfiere la propiedad a otro miembro con la verificación en dos pasos activada, por ejemplo {suggest}. {name} pasa entonces automáticamente al rol Administrador.',
})
