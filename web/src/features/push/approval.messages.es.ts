import { translate } from '@/i18n'
import { m } from './approval.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  pushTitle: 'La notificación',
  pushCaption: '{n, plural, one {# suscriptor} other {# suscriptores}} en {sites} · {schedule}',
  link: 'Enlace',
  buttons: 'Botones',
  expiry: 'Caduca después de',
  hours: '{n, plural, one {# hora} other {# horas}}',
  minutes: '{n, plural, one {# minuto} other {# minutos}}',
  tag: 'Etiqueta',
  none: '–',
  waTitle: 'El mensaje de WhatsApp',
  waCaption: '{template} · {category} · {language} · {n, plural, one {# persona} other {# personas}} · {schedule}',
  business: 'Enviado como',
  variables: 'Variables',
  varField: '\'{{\'{key}\'}}\' de {field}',
  varFieldFallback: '\'{{\'{key}\'}}\' de {field}, o “{fallback}”',
  varText: '\'{{\'{key}\'}}\' {text}',
  cost: 'Costo estimado',
  costLine: '{amount}, que Meta cobra al entregarse',
})
