import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  kind: 'La puntuación cruza…',
  risesTo: 'Sube a {threshold} o más',
  fallsBelow: 'Baja de {threshold}',
  oncePer: 'Como máximo una vez por contacto cada {days, plural, one {# día} other {# días}}',
  startsWhen: 'Empieza cuando la puntuación',
  rises: 'Sube a',
  falls: 'Baja de',
  direction: 'Dirección',
  threshold: 'Puntuación',
  orMore: 'o más · {band} empieza en {from}',
  below: 'o menos · {band} empieza en {from}',
  orMorePlain: 'o más',
  belowPlain: 'o menos',
  bandHot: 'Caliente',
  bandWarm: 'Templado',
  oncePerLabel: 'Como máximo una vez por contacto cada',
  days: 'días',
  explain: 'Empieza con los cambios por actividad y decaimiento. Por los contactos que cruzan solo porque cambió una regla de puntuación se pregunta al guardar la regla; por defecto, no se inicia para ellos.',
  needs: 'Solo quienes pueden editar automatizaciones pueden elegir este disparador.',
  noScoring: 'La puntuación aún no tiene reglas, así que ninguna puntuación cruza nada.',
  setUp: 'Configurar la puntuación',
  isOneOf: 'es uno de',
  isOneOfPhrase: '{field} es {values}',
})
