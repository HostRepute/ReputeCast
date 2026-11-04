import { translate } from '@/i18n'
import { m } from './settings.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  title: 'SMS-Anbieter',
  senders: '{count, plural, one {# Absender} other {# Absender}} · {kinds}',
  noSenders: 'Noch keine Absender',
  routes: 'Routen',
  notRouted: 'Noch keine Routen',
  held: 'Zurückgehalten: {countries}',
  sendersLabel: 'Absender',
  connected: 'Verbunden',
  connectedOn: '{date} · {name}',
  cost: 'Kosten · {month}',
  budget: 'SMS-Budget · {month}',
  spentOf: '<b>{spent}</b> von {budget}',
  alerts: 'Warnungen bei {alert}\u00A0% und 100\u00A0%. Bei 100\u00A0% können Kampagnen nicht starten und SMS-Schritte warten, bis jemand, der Anbieter verwalten kann, das Budget erhöht.',
  noBudget: 'Kein Monatsbudget: bisher {spend} im {month}.',
  foot: 'Absender, Routen nach Land, Ruhezeiten, Regeln und Budget finden Sie unter Versandanbieter › SMS.',
  changeBudget: 'Budget ändern',
  open: 'SMS-Anbieter öffnen',
})
