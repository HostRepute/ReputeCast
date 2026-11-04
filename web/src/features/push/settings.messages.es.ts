import { translate } from '@/i18n'
import { m } from './settings.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  pushTitle: 'Notificaciones push web',
  pushBuiltIn: 'Integrado · sin proveedor en el que registrarse',
  keys: 'Claves',
  keysCreated: 'Creadas el {date} · {name}',
  sites: 'Sitios',
  noSites: 'Aún no hay sitios',
  subscribers: 'Suscriptores',
  subscribersLine: '{total} · {linked} vinculados a contactos',
  cap: 'Límite de frecuencia',
  capLine: '{n, plural, one {# notificación} other {# notificaciones}} al día por suscriptor',
  pushFoot: 'Las claves, los sitios, los avisos, las horas de silencio y el límite de frecuencia están en Proveedores de envío › Push.',
  pushOpen: 'Abrir configuración de push',
  waTitle: 'WhatsApp',
  waProvider: 'WhatsApp Cloud API de Meta',
  business: 'Empresa',
  businessLine: '{name} · {number}',
  quality: 'Calificación de calidad',
  limit: 'Límite de mensajes',
  limitLine: '{n, plural, one {# persona} other {# personas}} al día',
  templates: 'Plantillas',
  templatesLine: '{approved} aprobadas · {pending} pendientes · {rejected} rechazadas',
  consent: 'Consentimiento para WhatsApp',
  consentLine: '{n, plural, one {# persona} other {# personas}}',
  waFoot: 'La cuenta, las plantillas, las tarifas y los eventos están en Proveedores de envío › WhatsApp.',
  waOpen: 'Abrir WhatsApp',
})
