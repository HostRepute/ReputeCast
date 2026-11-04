import { translate } from '@/i18n'
import { m } from './campaigns.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  newPush: 'Notificación push',
  newPushBody: 'Una notificación a los navegadores de los lectores en tus sitios con notificaciones push.',
  newPushHint: 'Título, mensaje y enlace, con una vista previa por plataforma.',
  newWhatsApp: 'Mensaje de WhatsApp',
  newWhatsAppBody: 'Una plantilla aprobada para las personas con consentimiento para WhatsApp.',
  newWhatsAppHint: 'Variables a partir de los campos del contacto; el costo se muestra antes de enviar.',
  untitledPush: 'Campaña push sin título',
  untitledWhatsApp: 'Campaña de WhatsApp sin título',
  noOpens: 'Las notificaciones y los mensajes de WhatsApp no tienen aperturas',
  shownTitle: 'Mostradas, según informa el service worker',
  local: '{date} · a las {time} en la zona horaria de cada destinatario',
  notSite: 'Sin sitio',
  siteSubscribers: 'Suscriptores del sitio',
  duplicated: 'Duplicada como “{name}”',
  duplicateFailed: 'No se pudo duplicar',
})
