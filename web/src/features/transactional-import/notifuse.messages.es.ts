import { translate } from '@/i18n'
import { m } from './notifuse.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  sourceWhat: 'Clave de API · contactos, listas, plantillas',
  templatesOption: 'Importar plantillas',
  templatesDetail: '{transactional, plural, =0 {ninguna plantilla transaccional} one {# plantilla transaccional llega como borrador} other {# plantillas transaccionales llegan como borradores}}, {saved, plural, =0 {ninguna otra} one {# otra va a las plantillas guardadas} other {# otras van a las plantillas guardadas}}',
  templatesSkipped: '{skipped, plural, one {Su correo de doble confirmación o de baja se queda allí: ReputeCast envía el suyo.} other {Sus # correos de doble confirmación y de baja se quedan allí: ReputeCast envía los suyos.}}',
  doneTemplates: '{transactional, plural, one {# plantilla transaccional llegó como borrador} other {# plantillas transaccionales llegaron como borradores}} y {saved, plural, one {# plantilla se añadió} other {# plantillas se añadieron}} a las plantillas guardadas.',
  doneTemplatesManual: '{manual, plural, one {# de ellas requiere corrección manual} other {# de ellas requieren corrección manual}} antes de poder usarse.',
  doneTemplatesLink: 'Abrir Plantillas transaccionales',
})
