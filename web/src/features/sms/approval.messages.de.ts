import { translate } from '@/i18n'
import { m } from './approval.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  title: 'SMS',
  caption: '{chars} Zeichen · {segments, plural, one {# SMS-Teil} other {# SMS-Teile}} · {encoding}',
  bubble: 'Die SMS, wie Empfänger sie erhalten',
  countries: 'Absender und Kosten nach Land',
  country: 'Land',
  sender: 'Absender',
  recipients: 'Empfänger',
  price: 'Preis pro SMS-Teil',
  cost: 'Kosten',
  skipped: 'Übersprungen',
  total: '{recipients, plural, one {# SMS} other {# SMS}} · {cost}',
  schedule: 'Zeitplan: {schedule}',
})
