import { translate } from '@/i18n'
import { m } from './messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  dialog: 'Diálogo',
  dismiss: 'Descartar',
  undo: 'Deshacer',
  hide: 'Ocultar',
  reference: 'Referencia <mono>{reference}</mono>',
  optional: 'opcional',
  typeToConfirm: 'Escribe {name} para confirmar.',
  reasonTooShort: 'Escribe un motivo de al menos {min} caracteres.',
  bulkPartial: 'Acción en bloque · resultado parcial',
  bulkDone: 'Acción en bloque · completada',
  undoFor: 'Deshacer para los {count}',
  leaveTitle: '¿Salir sin guardar?',
  saveAndLeave: 'Guardar y salir',
  unsavedChanges: 'Cambios sin guardar',
  stay: 'Quedarse',
  leaveWithoutSaving: 'Salir sin guardar',
  partFailed: 'Esta parte no se pudo mostrar',
  partFailedBody: 'El resto de la página funciona. Reintenta o vuelve a cargar la página si sigue pasando.',
  loadFailedBody: 'El servidor no respondió. No se perdió nada.',
  stepDone: 'Completado:',
})
