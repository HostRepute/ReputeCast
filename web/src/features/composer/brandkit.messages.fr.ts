import { translate } from '@/i18n'
import { mbk } from './brandkit.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(mbk, {
  banner: '<b>Partez de votre kit de marque</b> Logo, couleurs et polices de votre site web, appliqués à chaque modèle et nouveau bloc.',
  setUp: 'Configurer le kit de marque',
  defaultColours: 'Affiché dans les couleurs par défaut tant que le kit de marque n’est pas configuré',
  synced: 'Synchronisé',
  syncedHeader: 'En-tête + logo',
  syncedFooter: 'Pied de page + désabonnement',
  syncedHeaderNotice: 'En-tête synchronisé.',
  syncedFooterNotice: 'Pied de page synchronisé.',
  syncedNoticeBody: 'Il se modifie dans le kit de marque, et chaque brouillon et e-mail programmé qui le contient suit.',
  editInBrandKit: 'Modifier dans le kit de marque',
  syncedBlock: 'Bloc synchronisé',
  syncedNamedNotice: 'Bloc synchronisé «\u00A0{name}\u00A0».',
  syncedGone: 'Ce bloc synchronisé a été supprimé du kit de marque.',
  syncedGoneBody: 'Plus rien ne le modifie. Dissociez-le pour le modifier dans cet e-mail.',
  unlink: 'Dissocier',
  unlinked: 'Dissocié du bloc synchronisé',
  unlinkedDetail: 'La section n’appartient plus qu’à cet e-mail\u00A0: modifiez-la ici. Annuler rétablit le lien.',
  saveSyncedTitle: 'Bloc synchronisé',
  saveSyncedLead: 'Réutilisez cette section dans d’autres e-mails. Modifiez-la une fois dans le kit de marque et chaque brouillon qui la contient suit.',
  saveSynced: 'Enregistrer comme bloc synchronisé…',
  saveSyncedDialog: 'Enregistrer comme bloc synchronisé',
  saveSyncedDialogLead: 'La section rejoint le kit de marque, et cet e-mail contient le bloc synchronisé à sa place. Insérez-le dans d’autres e-mails depuis Blocs › Vos blocs.',
  syncedName: 'Nom',
  saveSyncedConfirm: 'Enregistrer le bloc synchronisé',
  saveSyncedFailed: 'Impossible d’enregistrer le bloc synchronisé\u00A0: {error}',
  savedSynced: '«\u00A0{name}\u00A0» enregistré comme bloc synchronisé',
  savedSyncedDetail: 'Insérez-le depuis Blocs › Vos blocs. Modifiez-le dans le kit de marque\u00A0: chaque brouillon qui le contient suit.',
})
