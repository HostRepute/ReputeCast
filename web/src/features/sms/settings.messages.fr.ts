import { translate } from '@/i18n'
import { m } from './settings.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  title: 'Fournisseurs SMS',
  senders: '{count, plural, one {# expéditeur} other {# expéditeurs}} · {kinds}',
  noSenders: 'Aucun expéditeur',
  routes: 'Routes',
  notRouted: 'Pas encore de route',
  held: 'Retenus\u00A0: {countries}',
  sendersLabel: 'Expéditeurs',
  connected: 'Connecté',
  connectedOn: '{date} · {name}',
  cost: 'Coût · {month}',
  budget: 'Budget SMS · {month}',
  spentOf: '<b>{spent}</b> sur {budget}',
  alerts: 'Alertes à {alert}\u00A0% et 100\u00A0%. À 100\u00A0%, les campagnes ne peuvent pas démarrer et les étapes SMS attendent qu’une personne pouvant gérer les fournisseurs l’augmente.',
  noBudget: 'Aucun budget mensuel\u00A0: {spend} dépensés jusqu’ici en {month}.',
  foot: 'Les expéditeurs, les routes par pays, les heures de silence, les règles et le budget se trouvent dans Fournisseurs d’envoi › SMS.',
  changeBudget: 'Modifier le budget',
  open: 'Ouvrir les fournisseurs SMS',
})
