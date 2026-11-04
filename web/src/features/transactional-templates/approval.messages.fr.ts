import { translate } from '@/i18n'
import { m } from './approval.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  ruleTitle: 'La publication d’un modèle transactionnel nécessite aussi une approbation',
  ruleDetail: 'S’applique à toutes les personnes pouvant publier des modèles transactionnels. Désactivé\u00A0: la publication est seulement consignée, et l’API envoie aussitôt la nouvelle version.',
  cardTitle: 'Modèle transactionnel',
  cardCaption: 'Le brouillon que l’approbation permet à son auteur de publier',
  version: 'Version',
  versionValue: '{key} v{version}',
  versionReplaces: '{key} v{version}, en remplacement de la v{live} en ligne',
  category: 'Catégorie',
  subject: 'Objet',
  variables: 'Variables',
  content: 'Contenu',
  noContentChange: 'Aucune modification du contenu',
  breaking: 'Changement incompatible',
  testRenders: 'Rendus de test',
  renders: 'Rendus',
  fails: 'Échecs',
  checksLine: '{passed, plural, one {# contrôle réussi} other {# contrôles réussis}}{warnings, plural, =0 {} one {, # avertissement} other {, # avertissements}}',
  published: 'Publié en v{version}',
  decisionNote: 'L’approbation porte sur ce brouillon exact. Si {name} le modifie avant de le publier, la publication nécessite une nouvelle approbation.',
  rejectDetail: 'Le refus est notifié à {name} et le brouillon n’est pas publié. L’API continue d’envoyer la version en ligne. Refusez quand il ne doit pas partir du tout\u202F; pour des corrections, demandez plutôt des modifications.',
  changesDetail: 'La demande retourne à {name} avec votre commentaire. Le brouillon reste non publié jusqu’à une nouvelle demande.',
})
