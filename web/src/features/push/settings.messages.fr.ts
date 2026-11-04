import { translate } from '@/i18n'
import { m } from './settings.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  pushTitle: 'Notifications push web',
  pushBuiltIn: 'Intégré · aucun fournisseur auquel s’inscrire',
  keys: 'Clés',
  keysCreated: 'Créées le {date} · {name}',
  sites: 'Sites',
  noSites: 'Aucun site pour l’instant',
  subscribers: 'Abonnés',
  subscribersLine: '{total} · associés à des contacts\u00A0: {linked}',
  cap: 'Plafond de fréquence',
  capLine: '{n, plural, one {# notification} other {# notifications}} par jour et par abonné',
  pushFoot: 'Les clés, les sites, les invites, les heures de silence et le plafond de fréquence se gèrent dans Fournisseurs d’envoi › Push.',
  pushOpen: 'Ouvrir la configuration push',
  waTitle: 'WhatsApp',
  waProvider: 'WhatsApp Cloud API de Meta',
  business: 'Entreprise',
  businessLine: '{name} · {number}',
  quality: 'Note de qualité',
  limit: 'Limite de messagerie',
  limitLine: '{n, plural, one {# personne} other {# personnes}} par jour',
  templates: 'Modèles',
  templatesLine: 'Approuvés\u00A0: {approved} · en attente\u00A0: {pending} · refusés\u00A0: {rejected}',
  consent: 'Consentement WhatsApp',
  consentLine: '{n, plural, one {# personne} other {# personnes}}',
  waFoot: 'Le compte, les modèles, la grille tarifaire et les événements se gèrent dans Fournisseurs d’envoi › WhatsApp.',
  waOpen: 'Ouvrir WhatsApp',
})
