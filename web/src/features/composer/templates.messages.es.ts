import { translate } from '@/i18n'
import { mt } from './templates.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(mt, {
  search: 'Buscar plantillas y secciones',
  found: '{count, plural, =0 {No se encontró nada} one {# resultado} other {# resultados}}',
  moreFor: 'Más acciones para {name}',
  rename: 'Cambiar nombre…',
  replace: 'Reemplazar por este correo',
  delete: 'Eliminar…',
  renameTitle: 'Cambiar el nombre de “{name}”',
  name: 'Nombre',
  description: 'Descripción (opcional)',
  renamed: 'Nuevo nombre: “{name}”',
  replaceTitle: '¿Reemplazar “{name}” por este correo?',
  replaceBody: 'Los correos nuevos creados a partir de ella empiezan desde el correo abierto ahora. Los correos ya creados no cambian.',
  replaceBadge: 'Reemplazar plantilla',
  replaceConfirm: 'Reemplazar plantilla',
  replacing: 'Reemplazando…',
  replaced: '“{name}” ahora contiene este correo',
  deleteTitle: '¿Eliminar “{name}”?',
  deleteTemplateBody: 'Sale de la galería. Los correos ya creados a partir de ella no cambian.',
  deleteSectionBody: 'Sale de las secciones guardadas. Los correos que ya la usan no cambian.',
  deleteBadge: 'Eliminar',
  deleteConfirm: 'Eliminar',
  deleting: 'Eliminando…',
  deleted: 'Se eliminó “{name}”',
})
