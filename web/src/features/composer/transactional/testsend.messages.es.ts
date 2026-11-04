import { translate } from '@/i18n'
import { mts } from './testsend.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(mts, {
  menuItem: 'Enviar un correo de prueba…',
  title: 'Enviar una prueba de {key}',
  sendTo: 'Enviar a',
  ownOnly: 'Sin transactional:send, las pruebas solo van a tu propia dirección.',
  fillWith: 'Rellenar con',
  samples: 'Los valores de ejemplo',
  testData: 'Datos de prueba · {name}',
  note: 'El borrador tal como está ahora, por la ruta Sistema, con [Test] en el asunto. No cuenta en las cifras del registro.',
  send: 'Enviar prueba',
  sending: 'Enviando…',
  sent: 'Prueba enviada a {email}',
  sentWith: 'Prueba enviada a {email} con “{name}”',
})
