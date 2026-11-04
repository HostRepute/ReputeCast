import { translate } from '@/i18n'
import { m } from './messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  dialog: 'Dialog',
  dismiss: 'Schließen',
  undo: 'Rückgängig',
  hide: 'Ausblenden',
  reference: 'Referenz <mono>{reference}</mono>',
  optional: 'optional',
  typeToConfirm: 'Geben Sie zur Bestätigung {name} ein.',
  reasonTooShort: 'Geben Sie eine Begründung mit mindestens {min} Zeichen an.',
  bulkPartial: 'Massenaktion · Teilergebnis',
  bulkDone: 'Massenaktion · erledigt',
  undoFor: 'Für die {count} rückgängig machen',
  leaveTitle: 'Ohne Speichern verlassen?',
  saveAndLeave: 'Speichern und verlassen',
  unsavedChanges: 'Ungespeicherte Änderungen',
  stay: 'Bleiben',
  leaveWithoutSaving: 'Ohne Speichern verlassen',
  partFailed: 'Dieser Teil konnte nicht angezeigt werden',
  partFailedBody: 'Der Rest der Seite funktioniert. Versuchen Sie es erneut, oder laden Sie die Seite neu, falls es wieder passiert.',
  loadFailedBody: 'Der Server hat nicht geantwortet. Es ging nichts verloren.',
  stepDone: 'Erledigt:',
})
