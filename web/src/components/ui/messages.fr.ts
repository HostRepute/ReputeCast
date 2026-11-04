import { translate } from '@/i18n'
import { m } from './messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  dialog: 'Boîte de dialogue',
  dismiss: 'Ignorer',
  undo: 'Annuler',
  hide: 'Masquer',
  reference: 'Référence <mono>{reference}</mono>',
  optional: 'facultatif',
  typeToConfirm: 'Saisissez {name} pour confirmer.',
  reasonTooShort: 'Indiquez une raison d’au moins {min} caractères.',
  bulkPartial: 'Action groupée · résultat partiel',
  bulkDone: 'Action groupée · terminée',
  undoFor: 'Annuler pour ces {count}',
  leaveTitle: 'Quitter sans enregistrer\u202F?',
  saveAndLeave: 'Enregistrer et quitter',
  unsavedChanges: 'Modifications non enregistrées',
  stay: 'Rester',
  leaveWithoutSaving: 'Quitter sans enregistrer',
  partFailed: 'Cette partie n’a pas pu s’afficher',
  partFailedBody: 'Le reste de la page fonctionne. Réessayez, ou rechargez la page si le problème persiste.',
  loadFailedBody: 'Le serveur n’a pas répondu. Rien n’a été perdu.',
  stepDone: 'Terminé\u00A0:',
})
