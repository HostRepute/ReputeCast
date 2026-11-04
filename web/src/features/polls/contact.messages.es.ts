import { translate } from '@/i18n'
import { m } from './contact.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  answered: 'Respondió {answer} a “{question}” en {campaign}',
  answeredComment: 'Respondió {answer} a “{question}” en {campaign} · con un comentario',
  changed: 'antes respondió {previous}',
  quote: '“{text}”',
  responded: 'Respondió a la encuesta “{name}” · {answered} de {questions} respondidas',
  started: 'Empezó la encuesta “{name}” · {answered} de {questions} respondidas',
  editable: 'puede cambiarla hasta el {date}',
  fromPoll: 'de una votación, {date}',
  fromSurvey: 'de una encuesta, {date}',
})
