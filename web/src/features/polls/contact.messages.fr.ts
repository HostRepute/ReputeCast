import { translate } from '@/i18n'
import { m } from './contact.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  answered: 'A répondu {answer} à «\u00A0{question}\u00A0» dans «\u00A0{campaign}\u00A0»',
  answeredComment: 'A répondu {answer} à «\u00A0{question}\u00A0» dans «\u00A0{campaign}\u00A0» · avec un commentaire',
  changed: 'auparavant\u00A0: {previous}',
  quote: '«\u00A0{text}\u00A0»',
  responded: 'A répondu au questionnaire «\u00A0{name}\u00A0» · réponses\u00A0: {answered} sur {questions}',
  started: 'A commencé le questionnaire «\u00A0{name}\u00A0» · réponses\u00A0: {answered} sur {questions}',
  editable: 'modifiable jusqu’au {date}',
  fromPoll: 'via un sondage, le {date}',
  fromSurvey: 'via un questionnaire, le {date}',
})
