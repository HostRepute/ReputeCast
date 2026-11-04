import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  source: 'Abstimmung oder Umfrage',
  pick: 'Abstimmung oder Umfrage auswählen',
  polls: 'Abstimmungen',
  surveys: 'Umfragen',
  pollIn: '{question} · {campaign}',
  answer: 'Antwort',
  any: 'Beliebige Antwort',
  is: 'Ist eine von',
  isNot: 'Ist keine von',
  answers: 'Antworten',
  onChange: 'Erneut starten, wenn sich die Antwort ändert',
  note: 'Testantworten und Antworten von Personen, die keine Kontakte sind, starten die Automation nie.',
  surveyNote: 'Eine Umfrage startet den Ablauf, wenn jemand sie absendet.',
  loading: 'Abstimmungen und Umfragen werden geladen…',
})
