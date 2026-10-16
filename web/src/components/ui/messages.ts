import { defineMessages } from '@/i18n'

/** The shared components' own words: dialogs, toasts, fields and error panels. */
export const m = defineMessages('ui', {
  dialog: 'Dialog',
  dismiss: 'Dismiss',
  undo: 'Undo',
  hide: 'Hide',
  reference: 'Reference <mono>{reference}</mono>',
  optional: 'optional',
  typeToConfirm: 'Type {name} to confirm.',
  reasonTooShort: 'Give a reason of at least {min} characters.',
  bulkPartial: 'Bulk action · partial result',
  bulkDone: 'Bulk action · done',
  undoFor: 'Undo for the {count}',
  leaveTitle: 'Leave without saving?',
  saveAndLeave: 'Save and leave',
  unsavedChanges: 'Unsaved changes',
  stay: 'Stay',
  leaveWithoutSaving: 'Leave without saving',
})
