import { translate } from '@/i18n'
import { m } from './campaigns.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  newPush: 'Notification push',
  newPushBody: 'Une notification dans le navigateur des lecteurs, sur vos sites push.',
  newPushHint: 'Titre, message et lien, avec un aperçu par plateforme.',
  newWhatsApp: 'Message WhatsApp',
  newWhatsAppBody: 'Un modèle approuvé, envoyé aux personnes ayant donné leur consentement WhatsApp.',
  newWhatsAppHint: 'Variables issues des champs de contact\u202F; le coût s’affiche avant l’envoi.',
  untitledPush: 'Campagne push sans titre',
  untitledWhatsApp: 'Campagne WhatsApp sans titre',
  noOpens: 'Les notifications et les messages WhatsApp n’ont pas d’ouvertures',
  shownTitle: 'Affichées, selon le service worker',
  local: '{date} · {time} dans le fuseau horaire de chaque destinataire',
  notSite: 'Aucun site',
  siteSubscribers: 'Abonnés du site',
  duplicated: 'Dupliquée sous le nom «\u00A0{name}\u00A0»',
  duplicateFailed: 'Impossible de dupliquer',
})
