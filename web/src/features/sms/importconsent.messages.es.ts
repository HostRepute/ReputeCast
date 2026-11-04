import { translate } from '@/i18n'
import { m } from './importconsent.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  title: 'Consentimiento para SMS',
  withEvidence: 'Se importa con su evidencia',
  withEvidenceText: 'Las filas que dicen sí, con una fecha, un origen y el texto de consentimiento, pasan a Suscrito a los SMS en la lista de destino, registradas como importadas con esa evidencia. No se envía ningún SMS a nadie.',
  missing: 'El consentimiento para SMS necesita su evidencia: asigna {count, plural, one {la columna} other {las columnas}} {missing} en Asignar campos. Hasta entonces, los números se importan y el consentimiento para SMS queda como No suscrito.',
  numbersOnly: 'Los números se importan sin consentimiento para SMS: quedan como No suscrito a los SMS. Para importar también el consentimiento, asigna en Asignar campos una columna de consentimiento con su fecha, origen y texto.',
  date: 'fecha',
  source: 'origen',
  text: 'texto',
  mapFields: 'Asignar campos',
})
