import { translate } from '@/i18n'
import { mbk } from './brandkit.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(mbk, {
  banner: '<b>Empieza desde tu kit de marca</b> El logotipo, los colores y las fuentes de tu sitio web, aplicados a todas las plantillas y bloques nuevos.',
  setUp: 'Configurar kit de marca',
  defaultColours: 'Se muestra con los colores predeterminados hasta que configures el kit de marca',
  synced: 'Sincronizado',
  syncedHeader: 'Cabecera + logotipo',
  syncedFooter: 'Pie + baja',
  syncedHeaderNotice: 'Cabecera sincronizada.',
  syncedFooterNotice: 'Pie sincronizado.',
  syncedNoticeBody: 'Se modifica en el kit de marca, y todos los borradores y correos programados que lo contienen siguen el cambio.',
  editInBrandKit: 'Editar en Kit de marca',
  syncedBlock: 'Bloque sincronizado',
  syncedNamedNotice: 'Bloque sincronizado “{name}”.',
  syncedGone: 'Este bloque sincronizado se eliminó del kit de marca.',
  syncedGoneBody: 'Ya nada lo modifica. Desvincúlalo para editarlo en este correo.',
  unlink: 'Desvincular',
  unlinked: 'Desvinculado del bloque sincronizado',
  unlinkedDetail: 'Ahora la sección solo forma parte de este correo: edítala aquí. Deshacer vuelve a vincularla.',
  saveSyncedTitle: 'Bloque sincronizado',
  saveSyncedLead: 'Reutiliza esta sección en otros correos. Edítala una vez en el kit de marca y todos los borradores que la contienen siguen el cambio.',
  saveSynced: 'Guardar como bloque sincronizado…',
  saveSyncedDialog: 'Guardar como bloque sincronizado',
  saveSyncedDialogLead: 'La sección pasa al kit de marca y este correo contiene el bloque sincronizado en su lugar. Insértalo en otros correos desde Bloques › Tus bloques.',
  syncedName: 'Nombre',
  saveSyncedConfirm: 'Guardar bloque sincronizado',
  saveSyncedFailed: 'No se pudo guardar el bloque sincronizado: {error}',
  savedSynced: '“{name}” se guardó como bloque sincronizado',
  savedSyncedDetail: 'Insértalo desde Bloques › Tus bloques. Edítalo en el kit de marca: todos los borradores que lo contienen siguen el cambio.',
})
