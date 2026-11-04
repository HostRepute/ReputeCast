import { translate } from '@/i18n'
import { m } from './messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  label: 'Sugerencia de serie de bienvenida',
  title: 'Los nuevos suscriptores de {list} aún no reciben ningún correo de bienvenida',
  body: 'La plantilla inicial Serie de bienvenida envía tres correos durante la primera semana.',
  covered: '{count, plural, one {{lists} ya tiene una.} other {{lists} ya tienen una.}}',
  start: 'Iniciar la serie de bienvenida',
  notNow: 'Ahora no',
  dismissed: 'No volveremos a sugerir una serie de bienvenida para {list}',
  dismissedDetail: 'Puedes iniciar una cuando quieras desde Automatizaciones › Plantillas iniciales.',
  dismissFailed: 'No se pudo descartar la sugerencia',
})
