import { translate } from '@/i18n'
import { m } from './trigger.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  source: 'Sondage ou questionnaire',
  pick: 'Choisir un sondage ou un questionnaire',
  polls: 'Sondages',
  surveys: 'Questionnaires',
  pollIn: '{question} · {campaign}',
  answer: 'Réponse',
  any: 'Toute réponse',
  is: 'Fait partie de',
  isNot: 'Ne fait pas partie de',
  answers: 'Réponses',
  onChange: 'Redémarrer quand la réponse change',
  note: 'Les réponses de test et celles de personnes qui ne sont pas des contacts ne le déclenchent jamais.',
  surveyNote: 'Un questionnaire démarre le scénario quand quelqu’un le soumet.',
  loading: 'Chargement des sondages et questionnaires…',
})
