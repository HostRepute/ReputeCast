import { translate } from '@/i18n'
import { m } from './approval.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  badge: 'Página',
  badgeLabel: 'Página: abrir “{name}” en el constructor de páginas',
  line: 'Página · {path} · {visibility}',
  whenApproved: 'Al aprobarse',
  scheduled: '{date} a las {time}',
  published: 'Publicada',
  cardTitle: 'Página',
  cardCaption: 'Lo que publica la aprobación: el contenido, la dirección, la visibilidad y la hora de publicación',
  page: 'Página',
  pageValue: '{name} · {template}',
  address: 'Dirección',
  visibility: 'Visibilidad',
  goesLive: 'Se publica',
  goesLiveNow: 'En cuanto se apruebe',
  goesLivePublished: 'Publicada una vez aprobada',
  version: 'Versión',
  versionValue: 'v{version}',
  previewTitle: 'Vista previa de “{name}”, v{version}',
  previewFullTitle: 'Vista previa completa de “{name}”',
  previewCaption: 'La página tal como la verán los lectores, dentro del marco del sitio',
  previewUnavailable: 'La vista previa no está disponible: {error}',
  previewLoading: 'La vista previa se está cargando.',
  openPreview: 'Abrir vista previa completa',
  openBuilder: 'Abrir en el constructor de páginas',
  decisionNote: 'La aprobación cubre exactamente esta versión: su contenido, dirección, visibilidad y hora de publicación. {when, select, at {Se publica a su hora de publicación} other {Se publica en cuanto se aprueba}}. Si {name} la cambia antes, publicarla requiere una nueva aprobación.',
  rejectDetail: 'Se avisa a {name} y la página no se publica. Rechaza cuando no deba publicarse en absoluto; para correcciones, solicita cambios.',
  changesDetail: 'La solicitud vuelve a {name} con tu comentario. La página no se publica hasta que vuelva a solicitarlo.',
})
