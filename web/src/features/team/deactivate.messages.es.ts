import { translate } from '@/i18n'
import { m } from './deactivate.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  eyebrow: 'Diálogo · desactivar miembro',
  titleOne: '¿Desactivar a {name}?',
  titleMany: '¿Desactivar a {count, plural, one {# miembro} other {# miembros}}?',
  descriptionOne: '{name} conserva su historial y se puede reactivar más tarde. Quitarlo del espacio de trabajo, en cambio, elimina su pertenencia; el registro de auditoría sigue mostrando su nombre.',
  descriptionMany: 'Conservan su historial y se pueden reactivar más tarde. El registro de auditoría sigue mostrando sus nombres.',
  loadingChanges: 'Cargando qué cambia…',
  confirmOne: 'Desactivar a {name}',
  confirmMany: 'Desactivar a {count, plural, one {# miembro} other {# miembros}}',
  doneOne: 'Se desactivó a {name}',
  doneMany: '{count, plural, one {# miembro desactivado} other {# miembros desactivados}}',
  doneDetail: '{count, plural, one {se cerró # sesión} other {se cerraron # sesiones}}',
  failedOne: 'No se pudo desactivar a {name}',
  failedMany: 'No se pudo desactivar a {count, plural, one {# miembro} other {# miembros}}',
  signedOutEverywhere: '{count, plural, one {Sesión cerrada en todas partes, ahora (# sesión)} other {Sesiones cerradas en todas partes, ahora (# sesiones)}}',
  cantSignIn: 'No puede iniciar sesión en {workspace}; los demás espacios de trabajo no se ven afectados',
  scheduled: '{count, plural, one {Su # campaña programada mantiene su programación y se envía en nombre del espacio de trabajo} other {Sus # campañas programadas mantienen su programación y se envían en nombre del espacio de trabajo}}',
  approvals: '{count, plural, one {# solicitud de aprobación pendiente sigue abierta para los aprobadores} other {# solicitudes de aprobación pendientes siguen abiertas para los aprobadores}}',
  keysLegend: 'Claves de API que creó · {count}',
  keepKey: 'Mantener {keys} en funcionamiento <k>({permissions})</k>',
  keepKeys: 'Mantener {keys} en funcionamiento',
  revoke: '{count, plural, one {Revocarla ahora} other {Revocarlas ahora}}',
  reason: 'Motivo <k>· queda en el registro de auditoría</k>',
  reasonPlaceholder: 'Fin del contrato',
})
