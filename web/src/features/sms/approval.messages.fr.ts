import { translate } from '@/i18n'
import { m } from './approval.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  title: 'SMS',
  caption: '{chars} caractères · {segments, plural, one {# segment SMS} other {# segments SMS}} · {encoding}',
  bubble: 'Le SMS tel que le reçoivent les destinataires',
  countries: 'Expéditeurs et coût par pays',
  country: 'Pays',
  sender: 'Expéditeur',
  recipients: 'Destinataires',
  price: 'Prix par segment SMS',
  cost: 'Coût',
  skipped: 'Ignorés',
  total: '{recipients, plural, one {# SMS} other {# SMS}} · {cost}',
  schedule: 'Programmation\u00A0: {schedule}',
})
