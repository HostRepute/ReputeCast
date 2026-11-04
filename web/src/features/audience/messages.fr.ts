import { translate } from '@/i18n'
import { m } from './messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  title: 'Audience',
  sections: 'Sections de l’audience',
  tabContacts: 'Contacts',
  tabContactsCount: 'Contacts · {count}',
  tabSegments: 'Segments',
  tabSegmentsCount: 'Segments · {count}',
  tabHealth: 'Santé de la liste',
  tabSuppressions: 'Liste de suppression',
  tabSuppressionsCount: 'Liste de suppression · {count}',
  tierActive: 'Actif',
  tierCooling: 'En baisse',
  tierAtRisk: 'À risque',
  tierSunset: 'Arrêt progressif',
  windowActive: 'ouverture dans les 30 jours',
  windowCooling: '31 à 90 jours',
  windowAtRisk: '91 à 180 jours',
  windowSunset: 'aucune ouverture en 180 jours',
  consentDouble: 'Double opt-in',
  consentSingle: 'Opt-in simple',
  consentPending: 'Importé · en attente',
  consentPendingFilter: 'Importé, en attente',
  mailboxOther: 'Autre',
})
