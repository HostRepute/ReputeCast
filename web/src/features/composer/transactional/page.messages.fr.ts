import { translate } from '@/i18n'
import { mt } from './page.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(mt, {
  documentTitle: '{name} · Modèle transactionnel · ReputeCast',
  documentTitleLoading: 'Modèle transactionnel · ReputeCast',
  newTitle: 'Nouveau modèle transactionnel · ReputeCast',
  opening: 'Ouverture du modèle',
  openingDetail: 'Contenu, variables et données de test · généralement moins d’une seconde',
  creating: 'Création d’un brouillon',
  creatingDetail: 'Nouveau modèle · rien n’est envoyé avant sa publication',
  untitled: 'Modèle sans titre',
  backToTemplates: 'Retour aux modèles',
  goToTemplates: 'Aller aux modèles',
  notFoundBody: 'Le modèle a peut-être été supprimé, ou le lien a été copié avant son enregistrement. Les modèles supprimés restent dans le journal d’audit.',
  cantOpen: 'Votre rôle ne permet pas d’ouvrir ce modèle',
  cantCreate: 'Votre rôle ne permet pas de créer des modèles transactionnels',
  titleNotFound: 'Introuvable · ReputeCast',
  titleForbidden: 'Accès refusé · ReputeCast',
  titleBroken: 'Une erreur s’est produite · ReputeCast',
})
