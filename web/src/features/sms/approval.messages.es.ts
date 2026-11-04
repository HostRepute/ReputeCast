import { translate } from '@/i18n'
import { m } from './approval.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  title: 'Mensaje SMS',
  caption: '{chars} caracteres · {segments, plural, one {# fragmento} other {# fragmentos}} · {encoding}',
  bubble: 'El SMS tal como lo reciben los destinatarios',
  countries: 'Remitentes y costo por país',
  country: 'País',
  sender: 'Remitente',
  recipients: 'Destinatarios',
  price: 'Precio por fragmento',
  cost: 'Costo',
  skipped: 'Omitidos',
  total: '{recipients, plural, one {# mensaje} other {# mensajes}} · {cost}',
  schedule: 'Programación: {schedule}',
})
