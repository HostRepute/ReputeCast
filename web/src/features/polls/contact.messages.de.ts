import { translate } from '@/i18n'
import { m } from './contact.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  answered: 'Hat in {campaign} auf „{question}“ mit {answer} geantwortet',
  answeredComment: 'Hat in {campaign} auf „{question}“ mit {answer} geantwortet · mit Kommentar',
  changed: 'geändert von {previous}',
  quote: '„{text}“',
  responded: 'Hat an der Umfrage „{name}“ teilgenommen · {answered} von {questions} beantwortet',
  started: 'Hat die Umfrage „{name}“ begonnen · {answered} von {questions} beantwortet',
  editable: 'kann die Antwort bis {date} ändern',
  fromPoll: 'aus einer Abstimmung, {date}',
  fromSurvey: 'aus einer Umfrage, {date}',
})
