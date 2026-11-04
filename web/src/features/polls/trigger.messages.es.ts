import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  source: 'Votación o encuesta',
  pick: 'Elige una votación o encuesta',
  polls: 'Votaciones',
  surveys: 'Encuestas',
  pollIn: '{question} · {campaign}',
  answer: 'Respuesta',
  any: 'Cualquier respuesta',
  is: 'Es una de',
  isNot: 'No es una de',
  answers: 'Respuestas',
  onChange: 'Volver a empezar cuando cambie la respuesta',
  note: 'Las respuestas de prueba y las de personas que no son contactos nunca lo inician.',
  surveyNote: 'Una encuesta inicia el recorrido cuando alguien la envía.',
  loading: 'Cargando las votaciones y encuestas…',
})
