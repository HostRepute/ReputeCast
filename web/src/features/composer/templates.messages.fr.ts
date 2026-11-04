import { translate } from '@/i18n'
import { mt } from './templates.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(mt, {
  search: 'Rechercher des modèles et des sections',
  found: '{count, plural, =0 {Aucun résultat} one {# résultat} other {# résultats}}',
  moreFor: 'Plus d’actions pour {name}',
  rename: 'Renommer…',
  replace: 'Remplacer par cet e-mail',
  delete: 'Supprimer…',
  renameTitle: 'Renommer «\u00A0{name}\u00A0»',
  name: 'Nom',
  description: 'Description (facultative)',
  renamed: 'Renommé en «\u00A0{name}\u00A0»',
  replaceTitle: 'Remplacer «\u00A0{name}\u00A0» par cet e-mail\u202F?',
  replaceBody: 'Les nouveaux e-mails créés à partir de ce modèle partiront de l’e-mail ouvert. Ceux déjà créés ne changent pas.',
  replaceBadge: 'Remplacer le modèle',
  replaceConfirm: 'Remplacer le modèle',
  replacing: 'Remplacement…',
  replaced: '«\u00A0{name}\u00A0» contient désormais cet e-mail',
  deleteTitle: 'Supprimer «\u00A0{name}\u00A0»\u202F?',
  deleteTemplateBody: 'Il disparaît de la galerie. Les e-mails déjà créés à partir de ce modèle ne changent pas.',
  deleteSectionBody: 'Elle disparaît des sections enregistrées. Les e-mails qui l’utilisent déjà ne changent pas.',
  deleteBadge: 'Supprimer',
  deleteConfirm: 'Supprimer',
  deleting: 'Suppression…',
  deleted: '«\u00A0{name}\u00A0» supprimé',
})
